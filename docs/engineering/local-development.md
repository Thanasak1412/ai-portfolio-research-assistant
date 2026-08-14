# Local Development

## Prerequisites

- Node.js 24 LTS and pnpm 10.
- Go 1.26.6 or newer supported 1.26 patch release. The module toolchain directive prevents builds with vulnerable older Go runtimes.
- PostgreSQL 17 through Docker Compose.
- Docker 27+ with Compose v2.
- sqlc 1.28.

Copy `.env.example` to `.env` only for local use. The example contains disposable local credentials, never production secrets.

## Commands

| Command | Purpose |
|---|---|
| `pnpm install` and `go mod download` | Install workspace dependencies. |
| `make dev-up` / `make dev-down` | Start/stop PostgreSQL, test PostgreSQL, API, worker, and web. |
| `make migrate-up` | Apply goose migrations to the explicit local database URL. |
| `make sqlc-generate` | Reproduce sqlc output. |
| `make format` / `make format-check` | Format or validate formatting. |
| `make lint` | Go vet, ESLint, and TypeScript checks. |
| `make test` | Go and frontend unit tests. |
| `make test-integration` | Disposable PostgreSQL integration test. |
| `make test-e2e` | Playwright Chromium smoke test. |
| `make contract-check` | Validate OpenAPI and generated-type drift. |
| `make build` | Build API, worker, and production web application. |

Local database reset is guarded: `make db-reset-local CONFIRM_RESET=portfolio-local`. It removes only Compose volumes in this product root.
