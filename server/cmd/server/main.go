package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"gitlab.com/anishgowda21/go-password-manager/server/internal/database"
	"golang.org/x/crypto/bcrypt"
)

type server struct {
	db *sql.DB
}

type SignupRequest struct {
	Email           string       `json:"email"`
	AuthKey         string       `json:"authKey"`
	WrappedVaultKey string       `json:"wrappedVaultKey"`
	Salt            string       `json:"salt"`
	KDF             database.KDF `json:"kdf"`
}

func decodedLen(s string) int {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return -1
	}
	return len(b)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		log.Printf("write JSON: encode response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(append(body, '\n')); err != nil {
		log.Printf("write JSON: write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func validateSignup(req *SignupRequest) error {
	addr, err := mail.ParseAddress(req.Email)

	if err != nil || addr.Address != req.Email || len(req.Email) > 254 {
		return errors.New("invalid email")
	}

	if decodedLen(req.AuthKey) != 32 {
		return errors.New("invalid auth key")
	}

	if decodedLen(req.WrappedVaultKey) != 60 {
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

	var req SignupRequest
	if err := json.UnmarshalRead(r.Body, &req, json.RejectUnknownMembers(true)); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := validateSignup(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.AuthKey), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("signup: hash error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	id, err := database.CreateUser(r.Context(), s.db, database.NewUser{
		Email:          req.Email,
		HashedAuthKey:  string(hash),
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

	log.Printf("signup: created user %s (%s)", id, req.Email)
	writeJSON(w, http.StatusCreated, map[string]string{"userId": id})
}

func catchAll(w http.ResponseWriter, r *http.Request) {
	log.Printf("unhandled: %s %s", r.Method, r.URL.Path)
	http.NotFound(w, r)
}

func main() {
	db, err := database.OpenDB("vault.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	s := &server{db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/signup", s.signupHandler)
	mux.HandleFunc("/", catchAll)

	addr := ":8090"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
