-include .env
export

MIGRATIONS_DIR ?= migrations

.PHONY: dev test build migrate-up migrate-down migrate-status seed docker-up docker-down docker-logs docker-migrate docker-seed

dev:
	go run ./cmd/api

test:
	go test ./...

build:
	go build -o bin/car-rental-api ./cmd/api
	go build -o bin/car-rental-migrate ./cmd/migrate
	go build -o bin/car-rental-seed ./cmd/seed

migrate-up:
	go run ./cmd/migrate -dir $(MIGRATIONS_DIR) up

migrate-down:
	go run ./cmd/migrate -dir $(MIGRATIONS_DIR) down

migrate-status:
	go run ./cmd/migrate -dir $(MIGRATIONS_DIR) status

seed:
	go run ./cmd/seed

docker-up:
	docker compose up --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f app

docker-migrate:
	docker compose run --rm migrate

docker-seed:
	docker compose run --rm seed
