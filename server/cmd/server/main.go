package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"gitlab.com/anishgowda21/go-password-manager/server/internal/api"
	"gitlab.com/anishgowda21/go-password-manager/server/internal/database"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	db, err := database.OpenDB("vault.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux := api.NewServer(db)
	host := getEnv("HOST", "127.0.0.1")
	port := getEnv("PORT", "8090")
	addr := net.JoinHostPort(host, port)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}
