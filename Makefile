SHELL := /bin/sh
PRODUCT_DATABASE_URL ?= postgres://portfolio:portfolio_local_only@localhost:5432/portfolio?sslmode=disable
PRODUCT_TEST_DATABASE_URL ?= postgres://portfolio:portfolio_test_local_only@localhost:5433/portfolio_test?sslmode=disable

.PHONY: install dev-up dev-down migrate-up migrate-status sqlc-generate format format-check lint test test-integration test-e2e contract-check build db-reset-local

install:
	pnpm install --frozen-lockfile
	go mod download

dev-up:
	docker compose up --build -d postgres postgres-test
	docker compose --profile tools run --rm migrate
	docker compose up --build -d api worker web

dev-down:
	docker compose down

migrate-up:
	@DATABASE_URL="$(PRODUCT_DATABASE_URL)" go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir backend/migrations postgres "$(PRODUCT_DATABASE_URL)" up

migrate-status:
	@DATABASE_URL="$(PRODUCT_DATABASE_URL)" go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir backend/migrations postgres "$(PRODUCT_DATABASE_URL)" status

sqlc-generate:
	sqlc generate

format:
	gofmt -w backend
	pnpm format

format-check:
	@test -z "$$(gofmt -l backend)"
	pnpm format:check

lint:
	go vet ./...
	pnpm lint
	pnpm typecheck

test:
	go test ./...
	pnpm test

test-integration:
	TEST_DATABASE_URL="$(PRODUCT_TEST_DATABASE_URL)" go test -tags=integration ./backend/internal/platform/database/... ./backend/internal/identity/...

test-e2e:
	pnpm test:e2e

contract-check:
	pnpm contract:check

build:
	mkdir -p bin
	go build -o bin/api ./backend/cmd/api
	go build -o bin/worker ./backend/cmd/worker
	pnpm build

db-reset-local:
	@test "$(CONFIRM_RESET)" = "portfolio-local" || (echo "Set CONFIRM_RESET=portfolio-local to reset only the local Compose databases"; exit 1)
	docker compose down -v
	docker compose up -d postgres postgres-test
