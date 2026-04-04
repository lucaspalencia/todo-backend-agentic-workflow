.PHONY: run test migrate-up migrate-down docker-up docker-down docker-test-up docker-test-down

# Load .env if present so DATABASE_URL and friends are available in shell targets.
ifneq (,$(wildcard .env))
  include .env
  export
endif

run:
	go run ./cmd/api

test:
	go test ./...

migrate-up:
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DATABASE_URL)" down 1

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-test-up:
	docker compose -f docker-compose.test.yml up -d

docker-test-down:
	docker compose -f docker-compose.test.yml down
