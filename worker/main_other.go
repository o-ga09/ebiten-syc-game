//go:build !js

package main

import (
	"log"
	"net/http"
	"sync"
)

// Local development server without Cloudflare: in-memory store, serves public/.
func main() {
	mux := http.NewServeMux()
	NewAPI(&MemStore{}).Register(mux)
	mux.Handle("/", http.FileServer(http.Dir("public")))
	log.Println("listening on http://localhost:8787 (in-memory store)")
	var mu sync.Mutex
	serial := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		mux.ServeHTTP(w, r)
	})
	log.Fatal(http.ListenAndServe(":8787", serial))
}
