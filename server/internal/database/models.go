package database

import "time"

type KDFType int

const (
	KDFPBKDF2   KDFType = 0
	KDFArgon2id KDFType = 1
)

type User struct {
	ID        string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserKeys struct {
	UserID         string
	HashedAuthKey  string
	WrappedUserKey string
}

type KDFSettings struct {
	UserID      string
	KDFType     KDFType
	Iterations  int
	Salt        string //stored as base64
	MemoryKiB   *int
	Parallelism *int
}

type CipherItem struct {
	ID        string
	UserID    string
	Blob      []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string `json:"-"`
	DeviceID         string
	DeviceName       string
	CreatedAt        time.Time
	LastUsedAt       time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
}
