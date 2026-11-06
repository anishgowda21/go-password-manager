package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var (
	ErrItemExists = errors.New("item already exists")
	ErrConflict   = errors.New("revision conflict")
)

type Item struct {
	ID        string
	Blob      []byte
	Revision  int
	CreatedAt int64
	UpdatedAt int64
}

func CreateItem(ctx context.Context, db *sql.DB, userID, id string, blob []byte) (Item, error) {
	now := getUnixTimeStamp(time.Now())
	_, err := db.ExecContext(ctx,
		`INSERT INTO items (user_id, id, blob, revision, created_at, updated_at)
		 VALUES (?, ?, ?, 1, ?, ?)`,
		userID, id, blob, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return Item{}, ErrItemExists
		}
		return Item{}, err
	}
	return Item{ID: id, Blob: blob, Revision: 1, CreatedAt: now, UpdatedAt: now}, nil
}

func ListItems(ctx context.Context, db *sql.DB, userID string) ([]Item, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, blob, revision, created_at, updated_at
		 FROM items WHERE user_id = ? ORDER BY created_at, id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Item, 0)
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Blob, &it.Revision, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func UpdateItem(ctx context.Context, db *sql.DB, userID, id string, revision int, blob []byte) (Item, error) {
	it := Item{ID: id, Blob: blob}
	err := db.QueryRowContext(ctx,
		`UPDATE items SET blob = ?, revision = revision + 1, updated_at = ?
		 WHERE user_id = ? AND id = ? AND revision = ?
		 RETURNING revision, created_at, updated_at`,
		blob, time.Now().Unix(), userID, id, revision,
	).Scan(&it.Revision, &it.CreatedAt, &it.UpdatedAt)

	//We will get a no rows error when the revision does not match the item does not belong to user
	//So we again try to check dropping the revision filter to get accurate info
	// Whether it was a revision conflict or a row not belonging to the user.
	if errors.Is(err, sql.ErrNoRows) {
		var one int
		err = db.QueryRowContext(ctx,
			`SELECT 1 FROM items WHERE user_id = ? AND id = ?`, userID, id).Scan(&one)
		if errors.Is(err, sql.ErrNoRows) {
			return Item{}, ErrNotFound
		}
		if err != nil {
			return Item{}, err
		}
		return Item{}, ErrConflict
	}
	if err != nil {
		return Item{}, err
	}
	return it, nil
}

func DeleteItem(ctx context.Context, db *sql.DB, userID, id string) error {
	res, err := db.ExecContext(ctx,
		`DELETE FROM items WHERE user_id = ? AND id = ?`, userID, id)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}
