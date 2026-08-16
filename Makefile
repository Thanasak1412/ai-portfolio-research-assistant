SHELL := /bin/sh
PRODUCT_DATABASE_URL ?= postgres://portfolio:portfolio_local_only@localhost:5432/portfolio?sslmode=disable
PRODUCT_TEST_DATABASE_URL ?= postgres://portfolio:portfolio_test_local_only@localhost:5433/portfolio_test?sslmode=disable
COMPOSE_AUTH_ENV ?= .compose.auth.env
AUTH_E2E_ENV ?= .compose.auth.e2e.env
AUTH_TLS_DIR ?= .local/auth-tls
AUTH_E2E_TLS_DIR ?= .local/ci-auth-tls

.PHONY: install dev-up dev-down auth-dev-up auth-dev-down auth-dev-reset auth-e2e migrate-up migrate-status sqlc-generate format format-check lint test test-integration test-e2e contract-check build db-reset-local

install:
	pnpm install --frozen-lockfile
	go mod download

dev-up:
	sh scripts/prepare-compose-auth-env.sh "$(COMPOSE_AUTH_ENV)"
	docker compose --env-file "$(COMPOSE_AUTH_ENV)" up --build -d postgres postgres-test
	docker compose --env-file "$(COMPOSE_AUTH_ENV)" --profile tools run --rm migrate
	docker compose --env-file "$(COMPOSE_AUTH_ENV)" up --build -d api worker web

dev-down:
	docker compose --env-file "$(COMPOSE_AUTH_ENV)" down

auth-dev-up:
	sh scripts/prepare-local-auth-https.sh "$(AUTH_TLS_DIR)"
	AUTH_TLS_DIR="$(AUTH_TLS_DIR)" sh scripts/prepare-compose-auth-env.sh "$(COMPOSE_AUTH_ENV)"
	docker compose --env-file "$(COMPOSE_AUTH_ENV)" up --build -d postgres postgres-test
	docker compose --env-file "$(COMPOSE_AUTH_ENV)" --profile tools run --rm migrate
	docker compose --env-file "$(COMPOSE_AUTH_ENV)" --profile auth-https up --build -d --wait --wait-timeout 120 api worker web auth-proxy

auth-dev-down:
	docker compose --env-file "$(COMPOSE_AUTH_ENV)" --profile auth-https down

auth-dev-reset:
	@test "$(CONFIRM_RESET)" = "portfolio-auth-local" || (echo "Set CONFIRM_RESET=portfolio-auth-local to remove only Compose application volumes"; exit 1)
	docker compose --env-file "$(COMPOSE_AUTH_ENV)" --profile auth-https down -v

auth-e2e:
	AUTH_TLS_DIR="$(AUTH_E2E_TLS_DIR)" sh scripts/prepare-ci-auth-https.sh "$(AUTH_E2E_TLS_DIR)"
	AUTH_TLS_DIR="$(AUTH_E2E_TLS_DIR)" sh scripts/prepare-auth-e2e-env.sh "$(AUTH_E2E_ENV)"
	docker compose --env-file "$(AUTH_E2E_ENV)" up --build -d --wait --wait-timeout 120 postgres postgres-test
	docker compose --env-file "$(AUTH_E2E_ENV)" --profile tools run --rm migrate-test
	docker compose --env-file "$(AUTH_E2E_ENV)" --profile auth-https up --build -d --wait --wait-timeout 120 api worker web auth-proxy
	sh scripts/verify-auth-https-stack.sh "$(AUTH_E2E_ENV)"
	PLAYWRIGHT_AUTH_E2E_IGNORE_HTTPS_ERRORS=true pnpm test:e2e:auth

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
	TEST_DATABASE_URL="$(PRODUCT_TEST_DATABASE_URL)" go test -tags=integration ./backend/internal/platform/database/... ./backend/internal/identity/... ./backend/internal/portfolio/infrastructure/database/... ./backend/internal/asset/infrastructure/database/...

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
