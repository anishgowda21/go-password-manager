package database

import (
	"context"
	"database/sql"
	"time"
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
	now := getUnixTimeStamp(time.Now())

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
