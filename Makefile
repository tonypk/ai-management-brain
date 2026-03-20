.PHONY: build run test lint migrate sqlc dev

build:
	CGO_ENABLED=0 go build -o brain ./cmd/brain

run: build
	./brain

test:
	go test ./... -v -count=1

test-cover:
	go test ./... -coverprofile=coverage.out -count=1
	go tool cover -func=coverage.out

lint:
	go vet ./...

sqlc:
	sqlc generate

migrate-up:
	migrate -path sql/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path sql/migrations -database "$(DATABASE_URL)" down 1

dev:
	docker compose up -d postgres redis
	@echo "Postgres: localhost:5432, Redis: localhost:6379"

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o brain ./cmd/brain
