package database

import (
	"context"
	"database/sql"
	"errors"
	"uuid"
)

const SessionTTL = 60 * 60 * 24

type SessionRequest struct {
	UserID           string
	DeviceID         string
	DeviceName       string
	RefreshTokenHash string
}

func CreateSession(ctx context.Context, db *sql.DB, req SessionRequest) (string, error) {
	id := uuid.NewV7().String()
	now := nowUnix()

	_, err := db.ExecContext(ctx, `
		INSERT INTO sessions
			(id, user_id, refresh_token_hash, device_id, device_name, created_at, last_used_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id, device_id) DO UPDATE SET
			id                 = excluded.id,
			refresh_token_hash = excluded.refresh_token_hash,
			device_name        = excluded.device_name,
			created_at         = excluded.created_at,
			last_used_at       = excluded.last_used_at,
			expires_at         = excluded.expires_at,
			revoked_at         = NULL`,
		id, req.UserID, req.RefreshTokenHash, req.DeviceID, req.DeviceName,
		now, now, now+SessionTTL)
	if err != nil {
		return "", err
	}
	return id, nil
}

func SessionIsActive(ctx context.Context, db *sql.DB, sessionID, userID string) (bool, error) {
	var one int
	now := nowUnix()
	err := db.QueryRowContext(ctx, `
		SELECT 1 FROM sessions
		WHERE id = ? AND user_id = ? AND revoked_at IS NULL AND expires_at > ?`,
		sessionID, userID, now).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func RotateRefreshToken(ctx context.Context, db *sql.DB, oldHash, newHash string, finish func(sessionID, userID string) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var sessionID, userID string

	now := nowUnix()

	err = tx.QueryRowContext(ctx, `
		UPDATE sessions
		SET refresh_token_hash = ?, last_used_at = ?
		WHERE refresh_token_hash = ? AND revoked_at IS NULL AND expires_at > ?
		RETURNING id, user_id`,
		newHash, now, oldHash, now,
	).Scan(&sessionID, &userID)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	if err != nil {
		return err
	}

	if err := finish(sessionID, userID); err != nil {
		return err
	}

	return tx.Commit()
}

func RevokeByRefreshHash(ctx context.Context, db *sql.DB, hash string) error {
	now := nowUnix()
	_, err := db.ExecContext(ctx, `
		UPDATE sessions SET revoked_at = ?
		WHERE refresh_token_hash = ? AND revoked_at IS NULL`,
		now, hash)
	return err
}
