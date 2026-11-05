package api

import (
	"database/sql"
	"log"
	"net/http"
	"os"
)

type server struct {
	db        *sql.DB
	jwtSecret []byte
}

func getJWTSecret() []byte {
	sec := os.Getenv("JWT_SECRET")
	if sec == "" {
		panic("JWT Secret not avilable in environment!")
	}
	if len(sec) < 32 {
		panic("JWT Secret too short at least 32 bytes needed!")
	}

	return []byte(sec)
}

func catchAll(w http.ResponseWriter, r *http.Request) {
	log.Printf("unhandled: %s %s", r.Method, r.URL.Path)
	http.NotFound(w, r)
}

func NewServer(db *sql.DB) *http.ServeMux {
	s := &server{db: db, jwtSecret: getJWTSecret()}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/signup", s.signupHandler)
	mux.HandleFunc("POST /api/prelogin", s.preloginHandler)
	mux.HandleFunc("/*", catchAll)
	return mux
}
