package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/mail"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
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
const dummyAuthHash = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

func (s *server) hashAuthKey(authKey []byte) string {
	mac := hmac.New(sha256.New, s.authHashKey)
	mac.Write(authKey)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
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

func userIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(ctxUserID).(string)
	return id
}

// trustedHops is how many proxies sit in front of the app and append to
// X-Forwarded-For. 0 means no proxy
func hopsFromEnv() int {
	n, err := strconv.Atoi(os.Getenv("TRUSTED_PROXY_HOPS"))
	if err != nil || n < 0 {
		return 0
	}
	return n
}
