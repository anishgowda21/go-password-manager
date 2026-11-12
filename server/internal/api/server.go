package api

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"
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
	loginLimit := newRateLimiter(5, 12*time.Second)
	mux.HandleFunc("POST /api/login", rateLimit(loginLimit, s.loginHandler))
	mux.HandleFunc("GET /api/me", s.requireAuth(s.meHandler))
	mux.HandleFunc("GET /api/sync", s.requireAuth(s.syncHandler))
	mux.HandleFunc("POST /api/items", s.requireAuth(s.createItemHandler))
	mux.HandleFunc("PUT /api/items/{id}", s.requireAuth(s.updateItemHandler))
	mux.HandleFunc("DELETE /api/items/{id}", s.requireAuth(s.deleteItemHandler))
	mux.HandleFunc("POST /api/refresh", s.refreshHandler)
	mux.HandleFunc("POST /api/logout", s.logoutHandler)
	mux.HandleFunc("/", catchAll)
	return mux
}
