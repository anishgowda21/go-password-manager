package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

type KDFPayload struct {
	Type        int `json:"type"`
	Time        int `json:"time"`
	MemoryKiB   int `json:"memoryKiB"`
	Parallelism int `json:"parallelism"`
}

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Saw this in twitter Military grade techinique to
// even out the time diff b/w account not existing and wrong password lol
var dummyHash = mustBcrypt("dummy-password")

func mustBcrypt(s string) []byte {
	h, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return h
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

func validateDevice(id, name string) (string, error) {
	if !uuidRe.MatchString(id) {
		return "", errors.New("invalid device id")
	}

	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n == 0 || n > 64 {
		return "", errors.New("invalid device name")
	}

	return name, nil
}
