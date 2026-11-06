package api

import (
	"encoding/json/v2"
	"errors"
	"log"
	"net/http"
	"strings"

	"gitlab.com/anishgowda21/go-password-manager/server/internal/database"
	"golang.org/x/crypto/bcrypt"
)

var defaultKDF = KDFPayload{Type: 1, Time: 3, MemoryKiB: 64 * 1024, Parallelism: 1}

type preloginRequest struct {
	Email string `json:"email"`
}

type preloginResponse struct {
	Salt string     `json:"salt"`
	KDF  KDFPayload `json:"kdf"`
}

type loginRequest struct {
	Email      string `json:"email"`
	AuthKey    string `json:"authKey"`
	DeviceID   string `json:"deviceId"`
	DeviceName string `json:"deviceName"`
}

func (s *server) preloginHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)

	var req preloginRequest
	if err := json.UnmarshalRead(r.Body, &req, json.RejectUnknownMembers(true)); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := validateEmail(req.Email); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	acct, err := database.GetAccountByEmail(r.Context(), s.db, req.Email)
	switch {
	case errors.Is(err, database.ErrNotFound):
		writeJSON(w, http.StatusOK, preloginResponse{Salt: s.fakeSalt(req.Email), KDF: defaultKDF})
	case err != nil:
		log.Printf("prelogin: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	default:
		writeJSON(w, http.StatusOK, preloginResponse{Salt: acct.Salt, KDF: KDFPayload(acct.KDF)})
	}
}

func (s *server) loginHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)

	var req loginRequest
	if err := json.UnmarshalRead(r.Body, &req, json.RejectUnknownMembers(true)); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := validateEmail(req.Email); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if decodedLen(req.AuthKey) != 32 {
		writeError(w, http.StatusBadRequest, "invalid auth key")
		return
	}

	deviceName, err := validateDevice(req.DeviceID, req.DeviceName)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	acct, err := database.GetAccountByEmail(r.Context(), s.db, req.Email)
	switch {
	case errors.Is(err, database.ErrNotFound):
		//Military grade technique to even out time diff, lol
		bcrypt.CompareHashAndPassword(dummyHash, []byte(req.AuthKey))
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	case err != nil:
		log.Printf("login: lookup: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(acct.HashedAuthKey), []byte(req.AuthKey)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	resp, err := s.issueAuth(r.Context(), acct.UserID, req.DeviceID, deviceName,
		acct.WrappedUserKey, acct.Salt, KDFPayload(acct.KDF))
	if err != nil {
		log.Printf("login: issue auth: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, resp)
}
