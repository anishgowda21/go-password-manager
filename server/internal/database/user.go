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

type Account struct {
	UserID         string
	HashedAuthKey  string
	WrappedUserKey string
	Salt           string
	KDF            KDF
}

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

var ErrNotFound = errors.New("not found")

func CreateUser(ctx context.Context, db *sql.DB, u NewUser) (string, error) {
	var se *sqlite.Error
	id := uuid.NewV7().String()
	now := getUnixTimeStamp(time.Now())

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

func GetAccountByEmail(ctx context.Context, db *sql.DB, email string) (Account, error) {
	var a Account
	err := db.QueryRowContext(ctx, `
		SELECT u.id, k.hashed_auth_key, k.wrapped_user_key, 
		s.salt, s.kdf_type, s.time, s.memory_kib, s.parallelism
		FROM users u JOIN user_keys k ON k.user_id = u.id
		JOIN kdf_settings s ON s.user_id = u.id WHERE u.email = ?`, email).Scan(
		&a.UserID, &a.HashedAuthKey, &a.WrappedUserKey, &a.Salt, &a.KDF.Type, &a.KDF.Time, &a.KDF.MemoryKiB, &a.KDF.Parallelism)

	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, ErrNotFound
	}

	return a, err
}
