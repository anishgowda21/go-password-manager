package database

import (
	"errors"
	"testing"
	"time"
)

func TestCreateSession_SameDeviceReplacesTheSession(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db)
	device := uniq("device")

	first, err := CreateSession(ctx, db, SessionRequest{
		UserID: user, DeviceID: device, DeviceName: "laptop", RefreshTokenHash: "hash-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Revoke it, then log in again on the same device.
	if err := RevokeByRefreshHash(ctx, db, "hash-1"); err != nil {
		t.Fatal(err)
	}

	second, err := CreateSession(ctx, db, SessionRequest{
		UserID: user, DeviceID: device, DeviceName: "laptop (renamed)", RefreshTokenHash: "hash-2",
	})
	if err != nil {
		t.Fatal(err)
	}

	if first == second {
		t.Fatal("re-login on the same device must get a new session id")
	}
	if n := count(t, db, `SELECT count(*) FROM sessions WHERE user_id = ? AND device_id = ?`, user, device); n != 1 {
		t.Fatalf("want one row per (user, device), got %d", n)
	}
	if n := count(t, db, `SELECT count(*) FROM sessions WHERE refresh_token_hash = 'hash-1'`); n != 0 {
		t.Fatal("the old refresh hash must be gone")
	}

	if active, err := SessionIsActive(ctx, db, first, user); err != nil || active {
		t.Fatalf("old session active = %v (err %v), want false", active, err)
	}
	if active, err := SessionIsActive(ctx, db, second, user); err != nil || !active {
		t.Fatalf("new session active = %v (err %v), want true (revoked_at must be cleared)", active, err)
	}
}

func TestCreateSession_DifferentDevicesCoexist(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db)

	a := newTestSession(t, db, user, uniq("hash"))
	b := newTestSession(t, db, user, uniq("hash"))

	for _, id := range []string{a, b} {
		if active, err := SessionIsActive(ctx, db, id, user); err != nil || !active {
			t.Fatalf("session %s active = %v (err %v), want true", id, active, err)
		}
	}
}

func TestSessionIsActive(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db)
	other := newTestUser(t, db)
	hash := uniq("hash")
	id := newTestSession(t, db, user, hash)

	tests := []struct {
		name      string
		sessionID string
		userID    string
		setup     func()
		want      bool
	}{
		{"active", id, user, func() {}, true},
		{"wrong user", id, other, func() {}, false},
		{"unknown session", uniq("nope"), user, func() {}, false},
		{"revoked", id, user, func() {
			mustExec(t, db, `UPDATE sessions SET revoked_at = ? WHERE id = ?`, time.Now().Unix(), id)
		}, false},
		{"un-revoked again", id, user, func() {
			mustExec(t, db, `UPDATE sessions SET revoked_at = NULL WHERE id = ?`, id)
		}, true},
		{"expired", id, user, func() {
			mustExec(t, db, `UPDATE sessions SET expires_at = 0 WHERE id = ?`, id)
		}, false},
	}

	// Subtests run in order and share one session, so each setup builds on the last.
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			got, err := SessionIsActive(ctx, db, tc.sessionID, tc.userID)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("active = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRotateRefreshToken(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db)
	sessionID := newTestSession(t, db, user, "old-hash")

	var gotSession, gotUser string
	err := RotateRefreshToken(ctx, db, "old-hash", "new-hash", func(sid, uid string) error {
		gotSession, gotUser = sid, uid
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotSession != sessionID || gotUser != user {
		t.Fatalf("finish got (%q, %q), want (%q, %q)", gotSession, gotUser, sessionID, user)
	}

	t.Run("the old hash is dead after a rotation", func(t *testing.T) {
		err := RotateRefreshToken(ctx, db, "old-hash", "another-hash", func(string, string) error { return nil })
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("the new hash works", func(t *testing.T) {
		err := RotateRefreshToken(ctx, db, "new-hash", "third-hash", func(string, string) error { return nil })
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
	})
}

// If issuing the new tokens fails, the rotation is rolled back, so the client
// can retry with the token it already has instead of being locked out.
func TestRotateRefreshToken_RollsBackWhenFinishFails(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db)
	newTestSession(t, db, user, "old-hash")

	boom := errors.New("boom")
	err := RotateRefreshToken(ctx, db, "old-hash", "new-hash", func(string, string) error { return boom })
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the finish error", err)
	}

	if n := count(t, db, `SELECT count(*) FROM sessions WHERE refresh_token_hash = 'old-hash'`); n != 1 {
		t.Fatal("the old hash must still be stored after a failed rotation")
	}
	if n := count(t, db, `SELECT count(*) FROM sessions WHERE refresh_token_hash = 'new-hash'`); n != 0 {
		t.Fatal("the new hash must not be stored after a failed rotation")
	}

	// And a retry with the original token succeeds.
	err = RotateRefreshToken(ctx, db, "old-hash", "new-hash", func(string, string) error { return nil })
	if err != nil {
		t.Fatalf("retry = %v, want nil", err)
	}
}

func TestRotateRefreshToken_RejectsRevokedAndExpiredSessions(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db)
	revokedID := newTestSession(t, db, user, "revoked-hash")
	expiredID := newTestSession(t, db, user, "expired-hash")

	mustExec(t, db, `UPDATE sessions SET revoked_at = ? WHERE id = ?`, time.Now().Unix(), revokedID)
	mustExec(t, db, `UPDATE sessions SET expires_at = 0 WHERE id = ?`, expiredID)

	for _, hash := range []string{"revoked-hash", "expired-hash"} {
		err := RotateRefreshToken(ctx, db, hash, "new-"+hash, func(string, string) error { return nil })
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: err = %v, want ErrNotFound", hash, err)
		}
	}
}

func TestRevokeByRefreshHash(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db)
	target := newTestSession(t, db, user, "target-hash")
	bystander := newTestSession(t, db, user, "bystander-hash")

	if err := RevokeByRefreshHash(ctx, db, "target-hash"); err != nil {
		t.Fatal(err)
	}

	if active, _ := SessionIsActive(ctx, db, target, user); active {
		t.Fatal("the revoked session must be inactive")
	}
	if active, _ := SessionIsActive(ctx, db, bystander, user); !active {
		t.Fatal("other sessions must be unaffected")
	}

	t.Run("is idempotent", func(t *testing.T) {
		if err := RevokeByRefreshHash(ctx, db, "target-hash"); err != nil {
			t.Fatalf("second revoke = %v, want nil", err)
		}
	})
	t.Run("an unknown hash is not an error", func(t *testing.T) {
		if err := RevokeByRefreshHash(ctx, db, uniq("unknown")); err != nil {
			t.Fatalf("revoke unknown = %v, want nil", err)
		}
	})
}
