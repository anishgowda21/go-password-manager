package api

import (
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"log"
	"net/http"
	"strings"

	"gitlab.com/anishgowda21/go-password-manager/server/internal/database"
)

type signupRequest struct {
	Email           string       `json:"email"`
	AuthKey         string       `json:"authKey"`
	WrappedVaultKey string       `json:"wrappedVaultKey"`
	Salt            string       `json:"salt"`
	DeviceID        string       `json:"deviceId"`
	DeviceName      string       `json:"deviceName"`
	KDF             database.KDF `json:"kdf"`
}

func validateSignup(req *signupRequest) error {
	if err := validateEmail(req.Email); err != nil {
		return err
	}

	if decodedLen(req.WrappedVaultKey) != 61 {
		return errors.New("invalid wrapped vault key")
	}

	if n := decodedLen(req.Salt); n < 16 || n > 64 {
		return errors.New("invalid salt")
	}

	k := req.KDF

	if k.Type != 1 ||
		k.Time < 1 || k.Time > 5 ||
		k.MemoryKiB < 16*1024 || k.MemoryKiB > 64*1024 ||
		k.Parallelism < 1 || k.Parallelism > 4 {
		return errors.New("invalid KDF settings")
	}
	return nil
}

func (s *server) signupHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)

	var req signupRequest
	if err := json.UnmarshalRead(r.Body, &req, json.RejectUnknownMembers(true)); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := validateSignup(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	deviceName, err := validateDevice(req.DeviceID, req.DeviceName)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	authBytes, err := base64.StdEncoding.DecodeString(req.AuthKey)
	if err != nil || len(authBytes) != 32 {
		writeError(w, http.StatusBadRequest, "invalid auth key")
		return
	}

	hash := s.hashAuthKey(authBytes)

	id, err := database.CreateUser(r.Context(), s.db, database.NewUser{
		Email:          req.Email,
		HashedAuthKey:  hash,
		WrappedUserKey: req.WrappedVaultKey,
		Salt:           req.Salt,
		KDF:            req.KDF,
	})

	switch {
	case errors.Is(err, database.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email already registered")
		return
	case err != nil:
		log.Printf("signup: store error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	authResponse, err := s.issueAuth(r.Context(), id, req.DeviceID, deviceName, req.WrappedVaultKey, req.Salt, KDFPayload(req.KDF))

	if err != nil {
		log.Printf("signup: auth issue error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, authResponse)
}
