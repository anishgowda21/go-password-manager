package api

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// The database, not the JWT, decides whether a session is alive.
func TestSession_RevokedInDB(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	env.mustExec(t, `UPDATE sessions SET revoked_at = ? WHERE id = ?`, time.Now().Unix(), u.SessionID)

	wantError(t, env.me(t, u.Access), http.StatusUnauthorized, "session ended")
	wantStatus(t, env.refresh(t, u.Refresh), http.StatusUnauthorized)

	// Un-revoking brings the same (still unexpired) access token back to life.
	env.mustExec(t, `UPDATE sessions SET revoked_at = NULL WHERE id = ?`, u.SessionID)
	wantStatus(t, env.me(t, u.Access), http.StatusOK)
}

func TestSession_ExpiredInDB(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	env.mustExec(t, `UPDATE sessions SET expires_at = 0 WHERE id = ?`, u.SessionID)

	wantError(t, env.me(t, u.Access), http.StatusUnauthorized, "session ended")
	wantStatus(t, env.refresh(t, u.Refresh), http.StatusUnauthorized)
}

func TestLogin_NewDeviceGetsItsOwnSession(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	second := env.login(t, u, newUUID(t))

	if claimsOf(t, second.AccessToken).SessionID == u.SessionID {
		t.Fatal("a different device must get a different session id")
	}
	// Logging in elsewhere must not end the first device's session.
	wantStatus(t, env.me(t, u.Access), http.StatusOK)
	wantStatus(t, env.me(t, second.AccessToken), http.StatusOK)
}

func TestLogin_SameDeviceReplacesTheOldSession(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	again := env.login(t, u, u.DeviceID)

	wantError(t, env.me(t, u.Access), http.StatusUnauthorized, "session ended")
	wantStatus(t, env.refresh(t, u.Refresh), http.StatusUnauthorized)
	wantStatus(t, env.me(t, again.AccessToken), http.StatusOK)

	n := env.count(t, `SELECT count(*) FROM sessions WHERE user_id = ? AND device_id = ?`, u.UserID, u.DeviceID)
	if n != 1 {
		t.Fatalf("want exactly one session row per (user, device), got %d", n)
	}
}

func TestLogout(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)
	other := env.login(t, u, newUUID(t)) // a second device

	rec := env.logout(t, u.Refresh)
	wantStatus(t, rec, http.StatusNoContent)
	if rec.Body.Len() != 0 {
		t.Fatalf("logout must return an empty body, got %q", rec.Body.String())
	}

	t.Run("the logged-out session is dead", func(t *testing.T) {
		wantStatus(t, env.refresh(t, u.Refresh), http.StatusUnauthorized)
		wantError(t, env.me(t, u.Access), http.StatusUnauthorized, "session ended")
	})

	t.Run("the other device is unaffected", func(t *testing.T) {
		wantStatus(t, env.me(t, other.AccessToken), http.StatusOK)
		wantStatus(t, env.refresh(t, other.RefreshToken), http.StatusOK)
	})

	t.Run("is idempotent", func(t *testing.T) {
		wantStatus(t, env.logout(t, u.Refresh), http.StatusNoContent)
	})

	t.Run("an unknown token of valid length is still 204", func(t *testing.T) {
		wantStatus(t, env.logout(t, strings.Repeat("nonsense", 4)), http.StatusNoContent)
	})

	t.Run("rejects a too-short token", func(t *testing.T) {
		wantStatus(t, env.logout(t, "x"), http.StatusUnauthorized)
	})

	t.Run("rejects an unknown field", func(t *testing.T) {
		rec := env.do(t, http.MethodPost, "/api/logout", "",
			`{"refreshToken":"`+strings.Repeat("x", 32)+`","extra":1}`)
		wantStatus(t, rec, http.StatusBadRequest)
	})

	t.Run("rejects malformed JSON", func(t *testing.T) {
		wantStatus(t, env.do(t, http.MethodPost, "/api/logout", "", "{nope"), http.StatusBadRequest)
	})
}

// Logging in again on a logged-out device must clear revoked_at.
func TestLogin_AfterLogoutOnSameDeviceWorks(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	wantStatus(t, env.logout(t, u.Refresh), http.StatusNoContent)

	again := env.login(t, u, u.DeviceID)
	wantStatus(t, env.me(t, again.AccessToken), http.StatusOK)
}
