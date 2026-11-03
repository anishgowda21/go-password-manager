package main

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"net/http"

	"gitlab.com/anishgowda21/go-password-manager/server/internal/database"
)

const SaltLen = 16

type PreloginRes struct {
	KdfSettings struct {
		KdfType     database.KDFType `json:"kdfType"`
		Iterations  int              `json:"iterations"`
		Memory      int              `json:"memory"`
		Parallelism int              `json:"parallelism"`
	} `json:"kdfSettings"`
	Salt []byte `json:"salt"`
}

func newSalt() ([]byte, error) {
	s := make([]byte, SaltLen)
	if _, err := rand.Read(s); err != nil {
		return nil, err
	}

	return s, nil
}

func preLogin(w http.ResponseWriter, _ *http.Request) {
	s, err := newSalt()
	if err != nil {
		http.Error(w, "failed to generate salt", http.StatusInternalServerError)
		return
	}

	res := PreloginRes{Salt: s}
	res.KdfSettings.KdfType = database.KDFArgon2id
	res.KdfSettings.Iterations = 100_000
	res.KdfSettings.Memory = 64 * 1024
	res.KdfSettings.Parallelism = 4

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	http.HandleFunc("/api/prelogin", preLogin)

	log.Fatal(http.ListenAndServe(":8090", withCORS(http.DefaultServeMux)))
}
