.PHONY: run build tidy db-up db-down test

# Postgres'i docker-compose ile başlat (repo kökünden)
db-up:
	cd .. && docker compose up -d db

db-down:
	cd .. && docker compose down

tidy:
	go mod tidy

build:
	go build -o bin/server ./cmd/server

# Sunucuyu çalıştır (migration'lar otomatik uygulanır)
run:
	go run ./cmd/server

test:
	go test ./...
