package database

import "database/sql"

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id			TEXT PRIMARY KEY,
	email		TEXT NOT NULL UNIQUE,
	created_at	INTEGER NOT NULL,
	updated_at	INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS user_keys (
	user_id          TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
	hashed_auth_key  TEXT NOT NULL,
	wrapped_user_key TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS kdf_settings (
	user_id     TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
	kdf_type    INTEGER NOT NULL,
	time        INTEGER NOT NULL,
	memory_kib  INTEGER NOT NULL,
	parallelism INTEGER NOT NULL,
	salt        TEXT NOT NULL
);
`

func OpenDB(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
