package api

import (
	"database/sql"
	"log"
	"net/http"
)

type server struct {
	db *sql.DB
}

func catchAll(w http.ResponseWriter, r *http.Request) {
	log.Printf("unhandled: %s %s", r.Method, r.URL.Path)
	http.NotFound(w, r)
}

func NewServer(db *sql.DB) *http.ServeMux {
	s := &server{db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/signup", s.signupHandler)
	mux.HandleFunc("/", catchAll)
	return mux
}
