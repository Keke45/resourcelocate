.PHONY: dev-api dev-web build test

# Terminal 1: API on :8080.
dev-api:
	go run ./cmd/server

# Terminal 2: React dev server on :5173, proxying /api to :8080.
dev-web:
	cd web && npm install && npm run dev

# One binary with the web app embedded.
build:
	cd web && npm ci && npm run build
	go build -o server ./cmd/server

test:
	go test ./...
