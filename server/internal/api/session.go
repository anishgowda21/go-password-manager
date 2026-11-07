package api

import (
	"encoding/json/v2"
	"errors"
	"log"
	"net/http"

	"gitlab.com/anishgowda21/go-password-manager/server/internal/database"
)

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type refreshResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"` // seconds
}

func readRefreshRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<10)
	var req refreshRequest
	if err := json.UnmarshalRead(r.Body, &req, json.RejectUnknownMembers(true)); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return "", false
	}
	if n := len(req.RefreshToken); n < 16 || n > 256 {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return "", false
	}
	return req.RefreshToken, true
}

func (s *server) refreshHandler(w http.ResponseWriter, r *http.Request) {
	presented, ok := readRefreshRequest(w, r)
	if !ok {
		return
	}

	newToken, newTokenHash, err := createRefreshToken()

	if err != nil {
		log.Printf("refresh: new token: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var access string
	err = database.RotateRefreshToken(r.Context(), s.db,
		hashRefreshToken(presented), newTokenHash,
		func(sessionID, userID string) error {
			t, err := s.newAccessToken(userID, sessionID)
			if err != nil {
				return err
			}
			access = t
			return nil
		})

	switch {
	case errors.Is(err, database.ErrNotFound):
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	case err != nil:
		log.Printf("refresh: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, refreshResponse{
		AccessToken:  access,
		RefreshToken: newToken,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	})
}

func (s *server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	presented, ok := readRefreshRequest(w, r)
	if !ok {
		return
	}

	if err := database.RevokeByRefreshHash(r.Context(), s.db, hashRefreshToken(presented)); err != nil {
		log.Printf("logout: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
