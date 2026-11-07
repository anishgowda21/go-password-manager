package api

import (
	"database/sql"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ---- helpers ----

func kdfBody(over map[string]any) map[string]any {
	k := map[string]any{"type": 1, "time": 1, "memoryKiB": 16 * 1024, "parallelism": 1}
	for key, v := range over {
		k[key] = v
	}
	return k
}

// newSignupBody returns a valid signup request with a fresh email.
func newSignupBody(t *testing.T) map[string]any {
	t.Helper()
	return map[string]any{
		"email":           "signup-" + base64.RawURLEncoding.EncodeToString(randBytes(t, 6)) + "@example.com",
		"authKey":         randB64(t, 32),
		"wrappedVaultKey": randB64(t, 61),
		"salt":            randB64(t, 16),
		"deviceId":        newUUID(t),
		"deviceName":      "test-device",
		"kdf":             kdfBody(nil),
	}
}

// mergeBody returns a copy of base with one field replaced.
func mergeBody(base map[string]any, key string, value any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	out[key] = value
	return out
}

// ---- success ----

func TestSignup_Success(t *testing.T) {
	env := newTestEnv(t)
	body := newSignupBody(t)

	rec := env.do(t, http.MethodPost, "/api/signup", "", body)
	wantStatus(t, rec, http.StatusCreated)

	raw := decode[map[string]any](t, rec)
	if len(raw) != 7 {
		t.Fatalf("response must have exactly userId, accessToken, refreshToken, expiresIn, wrappedVaultKey, salt, kdf; got %v", raw)
	}

	resp := decode[AuthResponse](t, rec)

	if !uuidRe.MatchString(resp.UserID) {
		t.Fatalf("userId %q is not a uuid", resp.UserID)
	}
	if resp.ExpiresIn != 3600 {
		t.Fatalf("expiresIn = %d, want 3600 (seconds)", resp.ExpiresIn)
	}
	if resp.WrappedVaultKey != body["wrappedVaultKey"] {
		t.Fatal("wrappedVaultKey must be echoed back")
	}
	if resp.Salt != body["salt"] {
		t.Fatal("salt must be echoed back")
	}
	if resp.KDF != testKDF {
		t.Fatalf("kdf = %+v, want %+v", resp.KDF, testKDF)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cc)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want JSON", ct)
	}

	tok, _, err := jwt.NewParser().ParseUnverified(resp.AccessToken, &accessClaims{})
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if alg := tok.Header["alg"]; alg != "HS256" {
		t.Fatalf("access token alg = %v, want HS256", alg)
	}

	claims := claimsOf(t, resp.AccessToken)
	if claims.Subject != resp.UserID {
		t.Fatal("access token sub must be the user id")
	}
	if claims.SessionID == "" || claims.SessionID == resp.UserID {
		t.Fatalf("access token needs a session id different from the user id, got %q", claims.SessionID)
	}
	if lifetime := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); lifetime != time.Hour {
		t.Fatalf("access token lifetime = %v, want 1h", lifetime)
	}

	if len(resp.RefreshToken) < 32 {
		t.Fatalf("refresh token is too short: %d chars", len(resp.RefreshToken))
	}
	if resp.RefreshToken == resp.AccessToken {
		t.Fatal("refresh token must not be the access token")
	}
}

func TestSignup_DuplicateEmail(t *testing.T) {
	env := newTestEnv(t)
	body := newSignupBody(t)
	wantStatus(t, env.do(t, http.MethodPost, "/api/signup", "", body), http.StatusCreated)

	t.Run("same email", func(t *testing.T) {
		again := mergeBody(newSignupBody(t), "email", body["email"])
		wantError(t, env.do(t, http.MethodPost, "/api/signup", "", again), http.StatusConflict, "email already registered")
	})
	t.Run("same email in another case", func(t *testing.T) {
		again := mergeBody(newSignupBody(t), "email", strings.ToUpper(body["email"].(string)))
		wantError(t, env.do(t, http.MethodPost, "/api/signup", "", again), http.StatusConflict, "email already registered")
	})
}

// ---- validation ----

func TestSignup_Validation(t *testing.T) {
	env := newTestEnv(t)

	tests := []struct {
		name  string
		field string
		value any
	}{
		{"email without @", "email", "not-an-email"},
		{"empty email", "email", ""},
		{"email with a display name", "email", "Name <someone@example.com>"},
		{"email over 254 chars", "email", strings.Repeat("a", 250) + "@x.io"},

		{"authKey of 31 bytes", "authKey", randB64(t, 31)},
		{"authKey of 33 bytes", "authKey", randB64(t, 33)},
		{"authKey that is not base64", "authKey", "!!!not-base64!!!"},

		{"wrappedVaultKey of 60 bytes", "wrappedVaultKey", randB64(t, 60)},
		{"wrappedVaultKey of 62 bytes", "wrappedVaultKey", randB64(t, 62)},
		{"empty wrappedVaultKey", "wrappedVaultKey", ""},

		{"salt of 15 bytes", "salt", randB64(t, 15)},
		{"salt of 65 bytes", "salt", randB64(t, 65)},

		{"kdf type 0", "kdf", kdfBody(map[string]any{"type": 0})},
		{"kdf type 2", "kdf", kdfBody(map[string]any{"type": 2})},
		{"kdf time 0", "kdf", kdfBody(map[string]any{"time": 0})},
		{"kdf time 6", "kdf", kdfBody(map[string]any{"time": 6})},
		{"kdf memory below 16 MiB", "kdf", kdfBody(map[string]any{"memoryKiB": 16383})},
		{"kdf memory above 64 MiB", "kdf", kdfBody(map[string]any{"memoryKiB": 65537})},
		{"kdf parallelism 0", "kdf", kdfBody(map[string]any{"parallelism": 0})},
		{"kdf parallelism 5", "kdf", kdfBody(map[string]any{"parallelism": 5})},
		{"kdf time as a string", "kdf", kdfBody(map[string]any{"time": "3"})},
		{"kdf time as a float", "kdf", kdfBody(map[string]any{"time": 3.5})},

		{"deviceId that is not a uuid", "deviceId", "hello"},
		{"uppercase deviceId", "deviceId", strings.ToUpper(newUUID(t))},
		{"empty deviceId", "deviceId", ""},
		{"empty deviceName", "deviceName", ""},
		{"blank deviceName", "deviceName", "   "},
		{"deviceName over 64 chars", "deviceName", strings.Repeat("d", 65)},

		{"an unknown field", "extra", 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := mergeBody(newSignupBody(t), tc.field, tc.value)
			wantStatus(t, env.do(t, http.MethodPost, "/api/signup", "", body), http.StatusBadRequest)
		})
	}

	t.Run("empty object", func(t *testing.T) {
		wantStatus(t, env.do(t, http.MethodPost, "/api/signup", "", "{}"), http.StatusBadRequest)
	})
	t.Run("malformed JSON", func(t *testing.T) {
		wantStatus(t, env.do(t, http.MethodPost, "/api/signup", "", "{not json"), http.StatusBadRequest)
	})
	t.Run("body over the size limit", func(t *testing.T) {
		raw := `{"email":"` + strings.Repeat("a", 20000) + `"}`
		wantStatus(t, env.do(t, http.MethodPost, "/api/signup", "", raw), http.StatusBadRequest)
	})
}

func TestSignup_AcceptsBoundaryValues(t *testing.T) {
	env := newTestEnv(t)

	tests := []struct {
		name  string
		field string
		value any
	}{
		{"lowest KDF settings", "kdf", map[string]any{"type": 1, "time": 1, "memoryKiB": 16384, "parallelism": 1}},
		{"highest KDF settings", "kdf", map[string]any{"type": 1, "time": 5, "memoryKiB": 65536, "parallelism": 4}},
		{"64-char deviceName", "deviceName", strings.Repeat("d", 64)},
		{"64-byte salt", "salt", randB64(t, 64)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := mergeBody(newSignupBody(t), tc.field, tc.value)
			wantStatus(t, env.do(t, http.MethodPost, "/api/signup", "", body), http.StatusCreated)
		})
	}
}

// ---- what actually lands in the database ----

func TestSignup_StoresTheRightThings(t *testing.T) {
	env := newTestEnv(t)

	authBytes := randBytes(t, 32)
	body := newSignupBody(t)
	body["authKey"] = base64.StdEncoding.EncodeToString(authBytes)
	mixedCase := strings.ToUpper(body["email"].(string))
	body["email"] = mixedCase

	rec := env.do(t, http.MethodPost, "/api/signup", "", body)
	wantStatus(t, rec, http.StatusCreated)
	resp := decode[AuthResponse](t, rec)
	sessionID := claimsOf(t, resp.AccessToken).SessionID

	t.Run("email is stored lowercase", func(t *testing.T) {
		var id string
		err := env.db.QueryRow(`SELECT id FROM users WHERE email = ?`, strings.ToLower(mixedCase)).Scan(&id)
		if err != nil {
			t.Fatalf("lookup by lowercase email: %v", err)
		}
		if id != resp.UserID {
			t.Fatalf("stored id = %q, want %q", id, resp.UserID)
		}
	})

	t.Run("auth key is stored as a keyed hash, never raw", func(t *testing.T) {
		var hashed, wrapped string
		err := env.db.QueryRow(
			`SELECT hashed_auth_key, wrapped_user_key FROM user_keys WHERE user_id = ?`, resp.UserID,
		).Scan(&hashed, &wrapped)
		if err != nil {
			t.Fatalf("load user_keys: %v", err)
		}

		if hashed == body["authKey"] {
			t.Fatal("the raw auth key must never be stored")
		}
		if len(hashed) != 44 {
			t.Fatalf("stored auth hash has %d chars, want 44", len(hashed))
		}
		want := (&server{authHashKey: []byte(testAuthHashKey)}).hashAuthKey(authBytes)
		if hashed != want {
			t.Fatal("stored auth hash is not HMAC-SHA256(authHashKey, authKey)")
		}

		if wrapped != body["wrappedVaultKey"] || len(wrapped) != 84 {
			t.Fatalf("wrapped key must be stored exactly as sent (84 base64 chars), got %d chars", len(wrapped))
		}
	})

	t.Run("kdf settings and salt are stored", func(t *testing.T) {
		var kdfType, tm, mem, par int
		var salt string
		err := env.db.QueryRow(
			`SELECT kdf_type, time, memory_kib, parallelism, salt FROM kdf_settings WHERE user_id = ?`, resp.UserID,
		).Scan(&kdfType, &tm, &mem, &par, &salt)
		if err != nil {
			t.Fatalf("load kdf_settings: %v", err)
		}
		if kdfType != 1 || tm != 1 || mem != 16384 || par != 1 || salt != body["salt"] {
			t.Fatalf("stored kdf = (%d, %d, %d, %d, %q)", kdfType, tm, mem, par, salt)
		}
	})

	t.Run("session is stored with its device and is not revoked", func(t *testing.T) {
		var deviceID string
		var revoked sql.NullInt64
		err := env.db.QueryRow(
			`SELECT device_id, revoked_at FROM sessions WHERE id = ?`, sessionID,
		).Scan(&deviceID, &revoked)
		if err != nil {
			t.Fatalf("load session: %v", err)
		}
		if deviceID != body["deviceId"] {
			t.Fatalf("device_id = %q, want %q", deviceID, body["deviceId"])
		}
		if revoked.Valid {
			t.Fatal("a fresh session must not be revoked")
		}
	})
}
