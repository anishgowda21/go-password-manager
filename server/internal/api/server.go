package api

import (
	"database/sql"
	"log"
	"net/http"
	"os"
)

type server struct {
	db          *sql.DB
	jwtSecret   []byte
	authHashKey []byte
}

func getEnvKeys(s *server) {
	jwtSec := os.Getenv("JWT_SECRET")
	if jwtSec == "" {
		panic("JWT Secret not avilable in environment!")
	}
	if len(jwtSec) < 32 {
		panic("JWT Secret too short at least 32 bytes needed!")
	}

	authKey := os.Getenv("AUTH_HASH_KEY")

	if authKey == "" {
		panic("Auth hash key not avilable in environment!")
	}
	if len(authKey) < 32 {
		panic("Auth hash key too short at least 32 bytes needed!")
	}
	s.jwtSecret = []byte(jwtSec)
	s.authHashKey = []byte(authKey)
}

func catchAll(w http.ResponseWriter, r *http.Request) {
	log.Printf("unhandled: %s %s", r.Method, r.URL.Path)
	http.NotFound(w, r)
}

func NewServer(db *sql.DB) *http.ServeMux {
	s := &server{db: db}
	getEnvKeys(s)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/signup", s.signupHandler)
	mux.HandleFunc("POST /api/prelogin", s.preloginHandler)
	mux.HandleFunc("POST /api/login", s.loginHandler)
	mux.HandleFunc("GET /api/me", s.requireAuth(s.meHandler))
	mux.HandleFunc("/", catchAll)
	return mux
}
