package api

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"gitlab.com/anishgowda21/go-password-manager/server/internal/web"
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

func NewServer(db *sql.DB) http.Handler {
	s := &server{db: db}
	getEnvKeys(s)
	mux := http.NewServeMux()
	signupLimit := newRateLimiter(3, 20*time.Minute)
	mux.HandleFunc("POST /api/signup", rateLimit(signupLimit, s.signupHandler))
	preLoginLimit := newRateLimiter(10, 6*time.Second)
	mux.HandleFunc("POST /api/prelogin", rateLimit(preLoginLimit, s.preloginHandler))
	loginLimit := newRateLimiter(5, 12*time.Second)
	mux.HandleFunc("POST /api/login", rateLimit(loginLimit, s.loginHandler))
	mux.HandleFunc("GET /api/me", s.requireAuth(s.meHandler))
	mux.HandleFunc("GET /api/sync", s.requireAuth(s.syncHandler))
	mux.HandleFunc("POST /api/items", s.requireAuth(s.createItemHandler))
	mux.HandleFunc("PUT /api/items/{id}", s.requireAuth(s.updateItemHandler))
	mux.HandleFunc("DELETE /api/items/{id}", s.requireAuth(s.deleteItemHandler))
	refreshLimit := newRateLimiter(30, 2*time.Second)
	mux.HandleFunc("POST /api/refresh", rateLimit(refreshLimit, s.refreshHandler))
	mux.HandleFunc("POST /api/logout", s.logoutHandler)
	mux.Handle("/", web.Handler())
	return securityHeaders(mux)
}
