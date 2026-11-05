package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/mail"
)

type KDFPayload struct {
	Type        int `json:"type"`
	Time        int `json:"time"`
	MemoryKiB   int `json:"memoryKiB"`
	Parallelism int `json:"parallelism"`
}

func decodedLen(s string) int {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return -1
	}
	return len(b)
}

func validateEmail(email string) error {
	addr, err := mail.ParseAddress(email)

	if err != nil || addr.Address != email || len(email) > 254 {
		return errors.New("invalid email")
	}
	return nil
}

func (s *server) fakeSalt(email string) string {
	mac := hmac.New(sha256.New, s.jwtSecret)
	mac.Write([]byte("prelogin-fake-salt:" + email))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)[:16])
}
