package api

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gitlab.com/anishgowda21/go-password-manager/server/internal/database"
)

const (
	testJWTSecret   = "test-jwt-secret-that-is-at-least-32-bytes-long"
	testAuthHashKey = "test-auth-hash-key-that-is-at-least-32-bytes"
)

// Smallest valid KDF settings, so the payload is accepted by signup validation.
var testKDF = KDFPayload{Type: 1, Time: 1, MemoryKiB: 16 * 1024, Parallelism: 1}

// testEnv is a real server (real handlers, real SQLite file in a temp dir).
// Tests talk to it in-process through httptest, so no port or cleanup is needed.
type testEnv struct {
	handler http.Handler
	db      *sql.DB
}

// newTestEnv uses t.Setenv, so tests using it must not call t.Parallel().
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("AUTH_HASH_KEY", testAuthHashKey)

	db, err := database.OpenDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return &testEnv{handler: NewServer(db), db: db}
}

// do sends a request. body may be nil, a string (sent raw, for malformed-JSON
// cases) or any value (sent as JSON). token is sent as a Bearer token if set.
func (e *testEnv) do(t *testing.T, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	switch v := body.(type) {
	case nil:
	case string:
		payload = []byte(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		payload = b
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

// doAuthHeader sends a GET with an arbitrary Authorization header value.
func (e *testEnv) doAuthHeader(t *testing.T, path, header string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) me(t *testing.T, token string) *httptest.ResponseRecorder {
	t.Helper()
	return e.do(t, http.MethodGet, "/api/me", token, nil)
}

func (e *testEnv) refresh(t *testing.T, refreshToken string) *httptest.ResponseRecorder {
	t.Helper()
	return e.do(t, http.MethodPost, "/api/refresh", "", map[string]string{"refreshToken": refreshToken})
}

func (e *testEnv) logout(t *testing.T, refreshToken string) *httptest.ResponseRecorder {
	t.Helper()
	return e.do(t, http.MethodPost, "/api/logout", "", map[string]string{"refreshToken": refreshToken})
}

func (e *testEnv) mustExec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := e.db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func (e *testEnv) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := e.db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return n
}

// ---- users ----

type testUser struct {
	Email      string
	AuthKey    string
	WrappedKey string
	Salt       string
	DeviceID   string

	UserID    string
	SessionID string
	Access    string
	Refresh   string
}

// registerUser signs up a brand-new user on a fresh device and returns its tokens.
func (e *testEnv) registerUser(t *testing.T) testUser {
	t.Helper()

	u := testUser{
		Email:      fmt.Sprintf("user-%x@example.com", randBytes(t, 6)),
		AuthKey:    randB64(t, 32),
		WrappedKey: randB64(t, 61),
		Salt:       randB64(t, 16),
		DeviceID:   newUUID(t),
	}

	rec := e.do(t, http.MethodPost, "/api/signup", "", map[string]any{
		"email":           u.Email,
		"authKey":         u.AuthKey,
		"wrappedVaultKey": u.WrappedKey,
		"salt":            u.Salt,
		"deviceId":        u.DeviceID,
		"deviceName":      "test-device",
		"kdf":             testKDF,
	})
	wantStatus(t, rec, http.StatusCreated)

	resp := decode[AuthResponse](t, rec)
	u.UserID = resp.UserID
	u.Access = resp.AccessToken
	u.Refresh = resp.RefreshToken
	u.SessionID = claimsOf(t, resp.AccessToken).SessionID
	return u
}

// login logs the user in from the given device and returns the auth response.
func (e *testEnv) login(t *testing.T, u testUser, deviceID string) AuthResponse {
	t.Helper()
	rec := e.do(t, http.MethodPost, "/api/login", "", map[string]string{
		"email":      u.Email,
		"authKey":    u.AuthKey,
		"deviceId":   deviceID,
		"deviceName": "test-login",
	})
	wantStatus(t, rec, http.StatusOK)
	return decode[AuthResponse](t, rec)
}

// ---- assertions / decoding ----

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, want, strings.TrimSpace(rec.Body.String()))
	}
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return v
}

// wantError checks the status and, if wantMsg is non-empty, the {"error": ...} message.
func wantError(t *testing.T, rec *httptest.ResponseRecorder, status int, wantMsg string) {
	t.Helper()
	wantStatus(t, rec, status)
	if wantMsg == "" {
		return
	}
	if got := decode[map[string]string](t, rec)["error"]; got != wantMsg {
		t.Fatalf("error = %q, want %q", got, wantMsg)
	}
}

// ---- randomness ----

func randBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return b
}

func randB64(t *testing.T, n int) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString(randBytes(t, n))
}

// newUUID returns a lowercase v4-style UUID that matches uuidRe.
func newUUID(t *testing.T) string {
	t.Helper()
	b := randBytes(t, 16)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// ---- JWT helpers ----

// claimsOf reads a token's claims WITHOUT verifying it.
func claimsOf(t *testing.T, token string) *accessClaims {
	t.Helper()
	claims := &accessClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(token, claims); err != nil {
		t.Fatalf("parse token: %v", err)
	}
	return claims
}

// goodClaims are claims the middleware would accept for this user's session.
func goodClaims(u testUser) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"sub": u.UserID,
		"sid": u.SessionID,
		"iat": now.Unix(),
		"exp": now.Add(time.Hour).Unix(),
	}
}

func withClaim(c jwt.MapClaims, key string, value any) jwt.MapClaims {
	out := jwt.MapClaims{}
	for k, v := range c {
		out[k] = v
	}
	out[key] = value
	return out
}

func withoutClaim(c jwt.MapClaims, key string) jwt.MapClaims {
	out := jwt.MapClaims{}
	for k, v := range c {
		if k != key {
			out[k] = v
		}
	}
	return out
}

// forge signs claims with any method and key (including alg=none).
func forge(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("forge token: %v", err)
	}
	return tok
}

func forgeHS256(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	return forge(t, jwt.SigningMethodHS256, []byte(secret), claims)
}

// flipSignature corrupts the first character of the signature segment.
func flipSignature(token string) string {
	parts := strings.Split(token, ".")
	sig := parts[2]
	first := "A"
	if sig[:1] == "A" {
		first = "B"
	}
	parts[2] = first + sig[1:]
	return strings.Join(parts, ".")
}

// swapSubject rewrites the payload's "sub" but keeps the original signature.
func swapSubject(t *testing.T, token, sub string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	payload["sub"] = sub
	raw, err = json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	parts[1] = base64.RawURLEncoding.EncodeToString(raw)
	return strings.Join(parts, ".")
}
