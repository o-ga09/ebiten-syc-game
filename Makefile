GOROOT := $(shell go env GOROOT)
WASM_EXEC := $(firstword $(wildcard $(GOROOT)/lib/wasm/wasm_exec.js $(GOROOT)/misc/wasm/wasm_exec.js))

.PHONY: build build-game build-worker dev local migrate-local migrate-remote deploy test

build: build-game build-worker

# Ebitengine game -> public/game.wasm
build-game:
	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o public/game.wasm ./game
	cp "$(WASM_EXEC)" public/wasm_exec.js

# workers-go API -> build/
build-worker:
	go run github.com/syumai/workers-go/cmd/workers-assets-gen -mode=go -o build
	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o build/app.wasm ./worker

test:
	go test ./...

# Cloudflare local dev (wrangler + local D1)
dev: migrate-local
	npx wrangler dev

migrate-local:
	npx wrangler d1 migrations apply jinsei-game --local

migrate-remote:
	npx wrangler d1 migrations apply jinsei-game --remote

deploy: migrate-remote
	npx wrangler deploy

# Quick local run without wrangler (in-memory store)
local: build-game
	go run ./worker
