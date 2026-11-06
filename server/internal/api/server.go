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
	jwt_sec := os.Getenv("JWT_SECRET")
	if jwt_sec == "" {
		panic("JWT Secret not avilable in environment!")
	}
	if len(jwt_sec) < 32 {
		panic("JWT Secret too short at least 32 bytes needed!")
	}

	auth_key := os.Getenv("AUTH_HASH_KEY")

	if auth_key == "" {
		panic("JWT Secret not avilable in environment!")
	}
	if len(auth_key) < 32 {
		panic("JWT Secret too short at least 32 bytes needed!")
	}
	s.jwtSecret = []byte(jwt_sec)
	s.authHashKey = []byte(auth_key)
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
	mux.HandleFunc("/", catchAll)
	return mux
}
