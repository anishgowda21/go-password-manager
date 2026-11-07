package database

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"
)

var (
	ctx     = context.Background()
	seq     atomic.Int64
	testKDF = KDF{Type: 1, Time: 1, MemoryKiB: 16 * 1024, Parallelism: 1}
)

// uniq returns a string that is unique within the test run (emails, ids, hashes).
func uniq(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, seq.Add(1))
}

// newTestDB opens a fresh SQLite database (with the real schema) in a temp dir.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// newTestUser inserts a user. Items and sessions have a foreign key to users,
// so almost every test needs one first.
func newTestUser(t *testing.T, db *sql.DB) string {
	t.Helper()
	id, err := CreateUser(ctx, db, NewUser{
		Email:          uniq("user") + "@example.com",
		HashedAuthKey:  "hashed-auth-key",
		WrappedUserKey: "wrapped-user-key",
		Salt:           "salt",
		KDF:            testKDF,
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	return id
}

// newTestSession creates a session for the user on a fresh device.
func newTestSession(t *testing.T, db *sql.DB, userID, refreshHash string) string {
	t.Helper()
	id, err := CreateSession(ctx, db, SessionRequest{
		UserID:           userID,
		DeviceID:         uniq("device"),
		DeviceName:       "test",
		RefreshTokenHash: refreshHash,
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	return id
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func count(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return n
}
