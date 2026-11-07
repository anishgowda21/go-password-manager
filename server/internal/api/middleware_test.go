package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestRequireAuth_ValidToken(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	rec := env.me(t, u.Access)
	wantStatus(t, rec, http.StatusOK)

	if got := decode[map[string]string](t, rec)["userId"]; got != u.UserID {
		t.Fatalf("userId = %q, want %q", got, u.UserID)
	}
}

// Sanity check for the forging helpers: a token we sign ourselves with the
// server's secret and good claims MUST be accepted. If this fails, the
// rejection cases below would be passing for the wrong reason.
func TestRequireAuth_ForgedValidTokenIsAccepted(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	token := forgeHS256(t, testJWTSecret, goodClaims(u))
	wantStatus(t, env.me(t, token), http.StatusOK)
}

func TestRequireAuth_AuthorizationHeader(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"missing header", "", http.StatusUnauthorized},
		{"basic scheme", "Basic abc123", http.StatusUnauthorized},
		{"bearer without a token", "Bearer", http.StatusUnauthorized},
		{"bearer with an empty token", "Bearer ", http.StatusUnauthorized},
		{"lowercase scheme", "bearer " + u.Access, http.StatusOK},
		{"uppercase scheme", "BEARER " + u.Access, http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := env.doAuthHeader(t, "/api/me", tc.header)
			wantStatus(t, rec, tc.want)

			if tc.want == http.StatusUnauthorized &&
				!strings.Contains(strings.ToLower(rec.Header().Get("WWW-Authenticate")), "bearer") {
				t.Fatalf("401 must carry WWW-Authenticate: Bearer, got %q", rec.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

func TestRequireAuth_RejectedTokens(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)
	other := env.registerUser(t)

	good := goodClaims(u)
	expired := withClaim(good, "exp", time.Now().Add(-10*time.Second).Unix())

	tests := []struct {
		name    string
		token   string
		wantMsg string // empty = don't check the message
	}{
		{"garbage", "garbage", "invalid token"},
		{"flipped signature", flipSignature(u.Access), "invalid token"},
		{"payload edited, signature kept", swapSubject(t, u.Access, other.UserID), "invalid token"},
		{"alg=none", forge(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, good), "invalid token"},
		{"HS384 with the right secret", forge(t, jwt.SigningMethodHS384, []byte(testJWTSecret), good), "invalid token"},
		{"signed with the wrong secret", forgeHS256(t, testJWTSecret+"x", good), "invalid token"},
		{"expired", forgeHS256(t, testJWTSecret, expired), "token expired"},
		{"missing exp", forgeHS256(t, testJWTSecret, withoutClaim(good, "exp")), "invalid token"},
		{"missing sub", forgeHS256(t, testJWTSecret, withoutClaim(good, "sub")), "invalid token"},
		{"missing sid", forgeHS256(t, testJWTSecret, withoutClaim(good, "sid")), "invalid token"},
		{"valid signature, unknown session", forgeHS256(t, testJWTSecret, withClaim(good, "sid", newUUID(t))), "session ended"},
		{"valid signature, someone else's sub with my sid", forgeHS256(t, testJWTSecret, withClaim(good, "sub", other.UserID)), "session ended"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := env.me(t, tc.token)
			wantError(t, rec, http.StatusUnauthorized, tc.wantMsg)

			if !strings.Contains(strings.ToLower(rec.Header().Get("WWW-Authenticate")), "bearer") {
				t.Fatalf("401 must carry WWW-Authenticate: Bearer, got %q", rec.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

// One table instead of a "without a token" check per endpoint.
func TestRequireAuth_ProtectedRoutesNeedAToken(t *testing.T) {
	env := newTestEnv(t)
	itemID := newUUID(t)

	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/me"},
		{http.MethodGet, "/api/sync"},
		{http.MethodPost, "/api/items"},
		{http.MethodPut, "/api/items/" + itemID},
		{http.MethodDelete, "/api/items/" + itemID},
	}

	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			wantStatus(t, env.do(t, r.method, r.path, "", nil), http.StatusUnauthorized)
		})
	}
}
