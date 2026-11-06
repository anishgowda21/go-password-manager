package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"gitlab.com/anishgowda21/go-password-manager/server/internal/database"
)

type ctxKey int

const (
	ctxUserID ctxKey = iota
	ctxSessionID
)

func userIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(ctxUserID).(string)
	return id
}

func bearerToken(r *http.Request) (string, bool) {
	const prefix = "bearer "
	h := r.Header.Get("Authorization")
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(h[len(prefix):]), true
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	writeError(w, http.StatusUnauthorized, msg)
}

func (s *server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, ok := bearerToken(r)
		if !ok {
			unauthorized(w, "missing token")
			return
		}

		claims := &accessClaims{}
		_, err := jwt.ParseWithClaims(raw, claims,
			func(*jwt.Token) (any, error) { return s.jwtSecret, nil },
			jwt.WithValidMethods([]string{"HS256"}),
			jwt.WithExpirationRequired(),
		)

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				unauthorized(w, "token expired")
				return
			}
			unauthorized(w, "invalid token")
			return
		}
		if claims.Subject == "" || claims.SessionID == "" {
			unauthorized(w, "invalid token")
			return
		}

		active, err := database.SessionIsActive(r.Context(), s.db, claims.SessionID, claims.Subject)

		if err != nil {
			log.Printf("auth: session lookup: %v", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		if !active {
			unauthorized(w, "session ended")
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserID, claims.Subject)
		ctx = context.WithValue(ctx, ctxSessionID, claims.SessionID)
		next(w, r.WithContext(ctx))

	}
}

func (s *server) meHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"userId": userIDFrom(r.Context())})
}
