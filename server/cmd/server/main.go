package main

import (
	"log"
	"net/http"

	"gitlab.com/anishgowda21/go-password-manager/server/internal/api"
	"gitlab.com/anishgowda21/go-password-manager/server/internal/database"
)

func main() {
	db, err := database.OpenDB("vault.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux := api.NewServer(db)

	addr := ":8090"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
