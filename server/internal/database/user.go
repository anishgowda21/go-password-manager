package database

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"uuid"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var ErrEmailTaken = errors.New("email already registered")

type KDF struct {
	Type        int `json:"type"`
	Time        int `json:"time"`
	MemoryKiB   int `json:"memoryKiB"`
	Parallelism int `json:"parallelism"`
}

type NewUser struct {
	Email          string
	HashedAuthKey  string
	WrappedUserKey string
	Salt           string
	KDF            KDF
}

func CreateUser(ctx context.Context, db *sql.DB, u NewUser) (string, error) {
	var se *sqlite.Error
	id := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO users (id, email, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		id, u.Email, now, now); err != nil {
		if errors.As(err, &se) && se.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
			return "", ErrEmailTaken
		}
		return "", err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO user_keys (user_id, hashed_auth_key, wrapped_user_key) VALUES (?, ?, ?)`,
		id, u.HashedAuthKey, u.WrappedUserKey); err != nil {
		return "", err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO kdf_settings (user_id, kdf_type, time, memory_kib, parallelism, salt) VALUES (?, ?, ?, ?, ?, ?)`,
		id, u.KDF.Type, u.KDF.Time, u.KDF.MemoryKiB, u.KDF.Parallelism, u.Salt); err != nil {
		return "", err
	}

	return id, tx.Commit()
}
