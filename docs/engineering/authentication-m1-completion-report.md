# Authentication M1 Completion Report

## Metadata

- Milestone: M1 Authentication Phase 1
- Verification task: `AUTH-VERIFY-001`
- Policy composition: `AUTH_IMPLEMENTATION_POLICY-v3`
- Verification date: 2026-08-14 (Asia/Bangkok)
- Protected main base: `57f6dde`
- Required merged ancestor: `ef4c40ff6c5afe0c1a751b174bc1f6a0f655217f`
- Verification branch: `codex/auth-verify-001`
- Final verification PR/head: pending draft PR and remote CI

## Implemented scope under review

This review covers M1 Authentication only: registration, login, current-user resolution, Ed25519 access JWTs, refresh sessions and rotation, replay/family revocation, logout, disabled-account enforcement, principal extraction, PostgreSQL-backed rate limits, audit evidence, browser security, HTTPS attestation, frontend memory-only sessions, protected routing, and operational verification.

## Explicitly out of scope

No MFA, password reset, email verification, SSO, roles, Portfolio, Asset, Transaction Ledger, Price Data, holdings, valuation, allocation, dashboard, alerts, documents, AI, or other M2 functionality was added.

## Verification summary

Passed evidence includes OpenAPI lint, contract tests, generated TypeScript generation/typecheck, sqlc generation, Go unit tests, `go vet`, backend builds, module-boundary checks, frontend lint/typecheck, 37 frontend unit tests, the production web build with `NEXT_PUBLIC_API_BASE_URL=https://app.localhost:3443/api/v1`, and local GitGuardian scanning.

The explicit `nanoid` override was updated from vulnerable `3.3.17` to patched `3.3.18`. The Go module toolchain was updated from `go1.26.5` to `go1.26.6` because the former had reachable standard-library vulnerabilities. `pnpm audit --audit-level high` and `govulncheck ./...` are now clean.

The real HTTPS Playwright Authentication suite previously executed against the live Caddy → Next.js → API → PostgreSQL stack with 3/3 tests passing before the final dependency/toolchain changes.

## Current closure blockers

1. Docker Desktop stopped during verification after repeated containerd overlay metadata I/O errors. The Docker socket is unavailable, so PostgreSQL integration, migration reruns, Compose health/readiness, HTTPS runtime checks, and final browser reruns cannot execute against the final verification head.
2. The required seven remote CI jobs have not yet run on the final verification branch/head.

The initial host-side integration run also found that the disposable `postgres-test` container had no active host port despite Compose rendering `5433:5432`; recreating it led to Docker overlay/socket failures. The host cannot route to its internal test-network IP, so this is not treated as passing evidence.

## Findings

- Critical: 0 confirmed.
- Major: 2 open verification blockers (Docker runtime unavailable; remote CI not executed on final head).
- Minor: 0 open implementation findings.
- Informational: host pnpm shims emitted Node engine warnings; checks passed with the Node 24 runtime path.

## Acceptance matrix

See [Authentication M1 Acceptance Matrix](authentication-m1-acceptance-matrix.md). `BLOCKED` rows are not closure passes.

## Security review

Static and unit evidence supports the approved AUTH-v1 component policies for password hashing, JWT validation, refresh-token representation, HMAC derivation, network identity, browser security, HTTPS attestation, memory-only frontend state, and default-deny principal extraction. Database-backed concurrency, account-state, rate-limit, audit, and runtime browser-security evidence must be rerun after Docker recovery.

No Authentication policy was changed. The only verification-only remediations are the patched `nanoid` override and Go 1.26.6 toolchain update.

## Commands and results

### Passed

- `sqlc generate`
- `sh scripts/check-module-boundaries.sh`
- `make format-check`
- `make lint`
- `make test`
- `pnpm contract:check`
- `go test ./...`
- `go vet ./...`
- `go build ./backend/cmd/api ./backend/cmd/worker`
- `NEXT_PUBLIC_API_BASE_URL=https://app.localhost:3443/api/v1 pnpm --filter @portfolio/web build`
- `pnpm audit --audit-level high`
- `GOTOOLCHAIN=go1.26.6 govulncheck ./...`
- `ggshield secret scan path . --recursive --yes --use-gitignore`

### Passed before final runtime outage

- `PLAYWRIGHT_AUTH_E2E_IGNORE_HTTPS_ERRORS=true pnpm test:e2e:auth` — 3 passed.

### Blocked or failed due environment

- `make test-integration` — host `localhost:5433` was not accepting connections; subsequent Docker overlay metadata I/O errors prevented a reliable rerun.
- `sh scripts/verify-auth-https-stack.sh .compose.auth.env` — Docker services became unavailable.
- Compose health/readiness and migration reruns — Docker daemon unavailable.
- Seven remote CI jobs — not yet run on this branch.

## M1 decision

M1 remains open until Docker is recovered, all affected database/runtime/browser checks are rerun on the final head, and the seven mandatory remote CI jobs pass.

M1 Status: Open

## Recommended next step

Recover Docker Desktop/containerd, recreate the disposable test database with host port `5433`, rerun all blocked checks, push this branch as a Draft PR, and obtain the required remote CI evidence. Do not begin M2.
