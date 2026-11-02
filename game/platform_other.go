//go:build !js

package main

import "os"

// apiBase can be set with GAME_API (e.g. http://localhost:8787) for desktop runs.
func apiBase() string { return os.Getenv("GAME_API") }

var memStore = map[string]string{}

func loadStore(key string) string { return memStore[key] }

func saveStore(key, value string) { memStore[key] = value }

var pendingName *string

func askName(current string) {
	n := current
	if n == "" {
		n = "ななし"
	}
	pendingName = &n
}

func pollName() (string, bool) {
	if pendingName == nil {
		return "", false
	}
	n := *pendingName
	pendingName = nil
	return n, true
}
