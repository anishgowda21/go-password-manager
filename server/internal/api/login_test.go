package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func (e *testEnv) prelogin(t *testing.T, email string) preloginResponse {
	t.Helper()
	rec := e.do(t, http.MethodPost, "/api/prelogin", "", map[string]string{"email": email})
	wantStatus(t, rec, http.StatusOK)
	return decode[preloginResponse](t, rec)
}

func ghostEmail(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("ghost-%x@example.com", randBytes(t, 6))
}

// ---- prelogin ----

func TestPrelogin_ExistingUser(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	rec := env.do(t, http.MethodPost, "/api/prelogin", "", map[string]string{"email": u.Email})
	wantStatus(t, rec, http.StatusOK)

	if raw := decode[map[string]any](t, rec); len(raw) != 2 {
		t.Fatalf("response must have exactly salt and kdf; got %v", raw)
	}

	resp := decode[preloginResponse](t, rec)
	if resp.Salt != u.Salt {
		t.Fatal("salt must match what was registered")
	}
	if resp.KDF != testKDF {
		t.Fatalf("kdf = %+v, want %+v", resp.KDF, testKDF)
	}

	t.Run("email is case-insensitive", func(t *testing.T) {
		if got := env.prelogin(t, strings.ToUpper(u.Email)); got.Salt != u.Salt {
			t.Fatal("an uppercase email must return the same salt")
		}
	})
}

// An unknown email must look like a real account: same shape, a stable salt
// that is still different per email, and a plausible length.
func TestPrelogin_UnknownEmailLooksLikeARealAccount(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	ghost := ghostEmail(t)

	rec := env.do(t, http.MethodPost, "/api/prelogin", "", map[string]string{"email": ghost})
	wantStatus(t, rec, http.StatusOK)
	if raw := decode[map[string]any](t, rec); len(raw) != 2 {
		t.Fatalf("response must have the same shape as a real one (salt, kdf); got %v", raw)
	}

	first := decode[preloginResponse](t, rec)
	again := env.prelogin(t, ghost)
	other := env.prelogin(t, ghostEmail(t))

	if first.KDF != defaultKDF {
		t.Fatalf("kdf = %+v, want the default %+v", first.KDF, defaultKDF)
	}
	if first.Salt != again.Salt {
		t.Fatal("the fake salt must be stable for the same email")
	}
	if first.Salt == other.Salt {
		t.Fatal("the fake salt must differ between emails")
	}
	if first.Salt == u.Salt {
		t.Fatal("the fake salt must not equal a real salt")
	}
	if raw, err := base64.StdEncoding.DecodeString(first.Salt); err != nil || len(raw) != 16 {
		t.Fatalf("fake salt must be 16 bytes of base64, got %q (err %v)", first.Salt, err)
	}
}

func TestPrelogin_Validation(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	tests := []struct {
		name string
		body any
	}{
		{"empty email", map[string]any{"email": ""}},
		{"not an email", map[string]any{"email": "not-an-email"}},
		{"empty object", "{}"},
		{"unknown field", map[string]any{"email": u.Email, "x": 1}},
		{"malformed JSON", "{nope"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantStatus(t, env.do(t, http.MethodPost, "/api/prelogin", "", tc.body), http.StatusBadRequest)
		})
	}
}

// ---- login ----

func TestLogin_Success(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	rec := env.do(t, http.MethodPost, "/api/login", "", map[string]string{
		"email":      u.Email,
		"authKey":    u.AuthKey,
		"deviceId":   newUUID(t),
		"deviceName": "second-device",
	})
	wantStatus(t, rec, http.StatusOK)

	if raw := decode[map[string]any](t, rec); len(raw) != 7 {
		t.Fatalf("response must have exactly the 7 auth fields; got %v", raw)
	}

	resp := decode[AuthResponse](t, rec)
	if resp.UserID != u.UserID {
		t.Fatal("userId must match the one from signup")
	}
	if resp.WrappedVaultKey != u.WrappedKey {
		t.Fatal("the wrapped vault key must be returned so the client can unwrap it")
	}
	if resp.Salt != u.Salt || resp.KDF != testKDF {
		t.Fatal("salt and kdf must be returned")
	}
	if resp.ExpiresIn != 3600 {
		t.Fatalf("expiresIn = %d, want 3600", resp.ExpiresIn)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cc)
	}

	t.Run("email is case-insensitive", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/login", "", map[string]string{
			"email":      strings.ToUpper(u.Email),
			"authKey":    u.AuthKey,
			"deviceId":   newUUID(t),
			"deviceName": "third-device",
		})
		wantStatus(t, rec, http.StatusOK)
	})
}

// A wrong password and an unknown account must be indistinguishable.
func TestLogin_WrongPasswordLooksLikeUnknownEmail(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	wrongPassword := env.do(t, http.MethodPost, "/api/login", "", map[string]string{
		"email": u.Email, "authKey": randB64(t, 32), "deviceId": newUUID(t), "deviceName": "d",
	})
	unknownEmail := env.do(t, http.MethodPost, "/api/login", "", map[string]string{
		"email": ghostEmail(t), "authKey": u.AuthKey, "deviceId": newUUID(t), "deviceName": "d",
	})

	wantError(t, wrongPassword, http.StatusUnauthorized, "invalid email or password")
	wantError(t, unknownEmail, http.StatusUnauthorized, "invalid email or password")

	if wrongPassword.Body.String() != unknownEmail.Body.String() {
		t.Fatalf("bodies differ:\n  wrong password: %q\n  unknown email:  %q",
			wrongPassword.Body.String(), unknownEmail.Body.String())
	}
	if body := decode[map[string]string](t, wrongPassword); len(body) != 1 {
		t.Fatalf("the error body must contain only an error field, got %v", body)
	}
}

func TestLogin_Validation(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	base := func() map[string]any {
		return map[string]any{
			"email":      u.Email,
			"authKey":    u.AuthKey,
			"deviceId":   newUUID(t),
			"deviceName": "device",
		}
	}

	tests := []struct {
		name string
		body any
	}{
		{"authKey of 31 bytes", mergeBody(base(), "authKey", randB64(t, 31))},
		{"authKey that is not base64", mergeBody(base(), "authKey", "!!!")},
		{"deviceId that is not a uuid", mergeBody(base(), "deviceId", "hello")},
		{"empty deviceName", mergeBody(base(), "deviceName", "")},
		{"deviceName over 64 chars", mergeBody(base(), "deviceName", strings.Repeat("d", 65))},
		{"empty email", mergeBody(base(), "email", "")},
		{"unknown field", mergeBody(base(), "extra", 1)},
		{"empty object", "{}"},
		{"malformed JSON", "{nope"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t) // fresh server = fresh limiter
			wantStatus(t, env.do(t, http.MethodPost, "/api/login", "", tc.body), http.StatusBadRequest)
		})
	}
}

func TestLogin_RateLimited(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	attempt := func(authKey string) *httptest.ResponseRecorder {
		return env.do(t, http.MethodPost, "/api/login", "", map[string]string{
			"email": u.Email, "authKey": authKey, "deviceId": newUUID(t), "deviceName": "d",
		})
	}

	// The login burst is 5, so these all reach the handler and fail normally.
	for i := 0; i < 5; i++ {
		wantStatus(t, attempt(randB64(t, 32)), http.StatusUnauthorized)
	}

	rec := attempt(u.AuthKey)
	wantError(t, rec, http.StatusTooManyRequests, "too many requests")
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("429 response is missing the Retry-After header")
	}
}
