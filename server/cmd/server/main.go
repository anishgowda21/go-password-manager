package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
)

type KDFPayload struct {
	Type        int `json:"type"`
	Time        int `json:"time"`
	MemoryKiB   int `json:"memoryKiB"`
	Parallelism int `json:"parallelism"`
}

type SignupRequest struct {
	Email           string     `json:"email"`
	AuthKey         string     `json:"authKey"`
	WrappedVaultKey string     `json:"wrappedVaultKey"`
	Salt            string     `json:"salt"`
	KDF             KDFPayload `json:"kdf"`
}

func decodedLen(s string) int {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return -1
	}
	return len(b)
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("signup: bad body: %v", err)
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("signup: email=%q kdf=%+v", req.Email, req.KDF)
	log.Printf("signup: authKey=%d bytes, wrappedVaultKey=%d bytes, salt=%d bytes",
		decodedLen(req.AuthKey), decodedLen(req.WrappedVaultKey), decodedLen(req.Salt))

	w.WriteHeader(http.StatusCreated)
}

func catchAll(w http.ResponseWriter, r *http.Request) {
	log.Printf("unhandled: %s %s", r.Method, r.URL.Path)
	http.NotFound(w, r)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/signup", signupHandler)
	mux.HandleFunc("/", catchAll)

	addr := ":8090"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
