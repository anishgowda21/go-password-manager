package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"gitlab.com/anishgowda21/go-password-manager/server/internal/database"
)

type sessionRow struct {
	hash      string
	expiresAt int64
	createdAt int64
}

func (e *testEnv) sessionRow(t *testing.T, sessionID string) sessionRow {
	t.Helper()
	var r sessionRow
	err := e.db.QueryRow(
		`SELECT refresh_token_hash, expires_at, created_at FROM sessions WHERE id = ?`, sessionID,
	).Scan(&r.hash, &r.expiresAt, &r.createdAt)
	if err != nil {
		t.Fatalf("load session %s: %v", sessionID, err)
	}
	return r
}

func TestRefresh_RotatesTheToken(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	before := env.sessionRow(t, u.SessionID)
	if before.hash != hashRefreshToken(u.Refresh) {
		t.Fatal("the database must store the sha256 of the refresh token")
	}
	if before.hash == u.Refresh {
		t.Fatal("the database must not store the raw refresh token")
	}
	if got := before.expiresAt - before.createdAt; got != database.SessionTTL {
		t.Fatalf("session lifetime = %d s, want %d", got, database.SessionTTL)
	}

	rec := env.refresh(t, u.Refresh)
	wantStatus(t, rec, http.StatusOK)

	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cc)
	}

	raw := decode[map[string]any](t, rec)
	if len(raw) != 3 {
		t.Fatalf("response must have exactly accessToken, refreshToken, expiresIn; got %v", raw)
	}
	for _, k := range []string{"accessToken", "refreshToken", "expiresIn"} {
		if _, ok := raw[k]; !ok {
			t.Fatalf("response is missing %q", k)
		}
	}
	if raw["expiresIn"] != float64(3600) {
		t.Fatalf("expiresIn = %v, want 3600", raw["expiresIn"])
	}

	resp := decode[refreshResponse](t, rec)
	if resp.RefreshToken == "" || resp.RefreshToken == u.Refresh {
		t.Fatal("the refresh token must be rotated")
	}

	claims := claimsOf(t, resp.AccessToken)
	if claims.SessionID != u.SessionID || claims.Subject != u.UserID {
		t.Fatalf("new access token must keep the session and user; got sid=%q sub=%q", claims.SessionID, claims.Subject)
	}
	if lifetime := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); lifetime != time.Hour {
		t.Fatalf("access token lifetime = %v, want 1h", lifetime)
	}

	after := env.sessionRow(t, u.SessionID)
	if after.hash != hashRefreshToken(resp.RefreshToken) {
		t.Fatal("the stored hash must now be the hash of the new refresh token")
	}
	if after.expiresAt != before.expiresAt {
		t.Fatal("refreshing must not slide the session expiry")
	}

	// Same session, so both access tokens keep working.
	wantStatus(t, env.me(t, resp.AccessToken), http.StatusOK)
	wantStatus(t, env.me(t, u.Access), http.StatusOK)
}

// Documents CURRENT behaviour: replaying a used refresh token is rejected, but
// it does NOT revoke the session, so the legitimate holder of the newest token
// keeps working. If you later add reuse detection (revoke the whole session on
// replay), this is the test to flip.
func TestRefresh_ReplayedTokenIsRejected(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	first := decode[refreshResponse](t, env.refresh(t, u.Refresh))

	wantError(t, env.refresh(t, u.Refresh), http.StatusUnauthorized, "invalid refresh token")
	wantStatus(t, env.refresh(t, first.RefreshToken), http.StatusOK)
}

func TestRefresh_RejectsBadInput(t *testing.T) {
	env := newTestEnv(t)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"unknown token of valid length", `{"refreshToken":"` + strings.Repeat("nonsense", 4) + `"}`, http.StatusUnauthorized},
		{"too short", `{"refreshToken":"x"}`, http.StatusUnauthorized},
		{"empty", `{"refreshToken":""}`, http.StatusUnauthorized},
		{"missing field", `{}`, http.StatusUnauthorized},
		{"too long (257 chars)", `{"refreshToken":"` + strings.Repeat("a", 257) + `"}`, http.StatusUnauthorized},
		{"unknown field", `{"refreshToken":"` + strings.Repeat("x", 32) + `","extra":1}`, http.StatusBadRequest},
		{"malformed JSON", `{nope`, http.StatusBadRequest},
		{"body over the 2 KiB limit", `{"refreshToken":"` + strings.Repeat("a", 3000) + `"}`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantStatus(t, env.do(t, http.MethodPost, "/api/refresh", "", tc.body), tc.want)
		})
	}
}

// Many requests present the SAME refresh token at once. Exactly one may win;
// the rest must be rejected. Run with -race to also catch data races.
func TestRefresh_ConcurrentRequestsOnlyOneWins(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	const n = 8
	body := fmt.Sprintf(`{"refreshToken":%q}`, u.Refresh)

	codes := make([]int, n)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			req := httptest.NewRequest(http.MethodPost, "/api/refresh", strings.NewReader(body))
			rec := httptest.NewRecorder()
			env.handler.ServeHTTP(rec, req)
			codes[i] = rec.Code
		}(i)
	}
	close(start)
	wg.Wait()

	var ok, rejected int
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusUnauthorized:
			rejected++
		}
	}
	if ok != 1 || rejected != n-1 {
		t.Fatalf("want exactly 1 success and %d rejections, got codes %v", n-1, codes)
	}
}
