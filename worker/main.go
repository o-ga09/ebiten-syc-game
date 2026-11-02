//go:build js

// Command worker is the Cloudflare Worker serving the game API (workers-go + D1).
// Static files in public/ are served by Workers Static Assets before this runs.
package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/syumai/workers-go"
	_ "github.com/syumai/workers-go/cloudflare/d1" // registers the "d1" driver
)

func main() {
	db, err := sql.Open("d1", "DB")
	if err != nil {
		log.Fatalf("open D1: %v", err)
	}
	mux := http.NewServeMux()
	NewAPI(NewSQLStore(db)).Register(mux)
	workers.Serve(mux)
}
