package database

import (
	"errors"
	"testing"
)

func TestCreateUser_StoresAndReturnsTheAccount(t *testing.T) {
	db := newTestDB(t)

	email := uniq("user") + "@example.com"
	id, err := CreateUser(ctx, db, NewUser{
		Email:          email,
		HashedAuthKey:  "hash-1",
		WrappedUserKey: "wrapped-1",
		Salt:           "salt-1",
		KDF:            KDF{Type: 1, Time: 4, MemoryKiB: 32768, Parallelism: 2},
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := GetAccountByEmail(ctx, db, email)
	if err != nil {
		t.Fatal(err)
	}

	want := Account{
		UserID:         id,
		HashedAuthKey:  "hash-1",
		WrappedUserKey: "wrapped-1",
		Salt:           "salt-1",
		KDF:            KDF{Type: 1, Time: 4, MemoryKiB: 32768, Parallelism: 2},
	}
	if got != want {
		t.Fatalf("account = %+v, want %+v", got, want)
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	db := newTestDB(t)

	email := uniq("user") + "@example.com"
	u := NewUser{Email: email, HashedAuthKey: "h", WrappedUserKey: "w", Salt: "s", KDF: testKDF}

	if _, err := CreateUser(ctx, db, u); err != nil {
		t.Fatal(err)
	}

	if _, err := CreateUser(ctx, db, u); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}

	// The failed attempt runs in a transaction, so it must leave nothing behind.
	for _, table := range []string{"users", "user_keys", "kdf_settings"} {
		if n := count(t, db, `SELECT count(*) FROM `+table); n != 1 {
			t.Fatalf("%s has %d rows after a duplicate signup, want 1", table, n)
		}
	}
}

func TestGetAccountByEmail_NotFound(t *testing.T) {
	db := newTestDB(t)

	if _, err := GetAccountByEmail(ctx, db, "nobody@example.com"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// Deleting a user must remove every row that belongs to them.
func TestDeleteUser_CascadesToAllTables(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db)
	bystander := newTestUser(t, db)

	if _, err := CreateItem(ctx, db, user, uniq("item"), blobOf(1, 48)); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateItem(ctx, db, bystander, uniq("item"), blobOf(1, 48)); err != nil {
		t.Fatal(err)
	}
	newTestSession(t, db, user, uniq("hash"))
	newTestSession(t, db, bystander, uniq("hash"))

	mustExec(t, db, `DELETE FROM users WHERE id = ?`, user)

	for _, table := range []string{"user_keys", "kdf_settings", "sessions", "items"} {
		if n := count(t, db, `SELECT count(*) FROM `+table+` WHERE user_id = ?`, user); n != 0 {
			t.Errorf("%s still has %d rows for the deleted user", table, n)
		}
		if n := count(t, db, `SELECT count(*) FROM `+table+` WHERE user_id = ?`, bystander); n != 1 {
			t.Errorf("%s has %d rows for the other user, want 1 (must be untouched)", table, n)
		}
	}
}
