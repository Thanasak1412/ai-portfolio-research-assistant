# M0 Blocker Resolution and Closure Review

**Review date:** 2026-08-02 (Asia/Bangkok)  
**M0 status:** Historical review; superseded by the 2026-08-03 closure evidence pending independent PR approval.  
**Authentication Phase 1 Execution Plan:** Not created because the M0 gate did not pass.

## 1. Review Basis

This review used the required precedence order:

1. Portfolio Foundation MVP Decision Closure Specification.
2. Approved ADRs in the Planning Baseline.
3. Planning Baseline.
4. Initial Project Bootstrap Execution Plan.
5. Bootstrap Completion Report.
6. Repository engineering and operations documentation.

The reviewed repository state has no commit or remote identity. The product directory resolves to the unrelated parent Git root at `/Users/mac/Documents/AI Automation 2`, that parent branch has no commits, and no `origin` remote exists. Consequently, no commit SHA can be used as M0 closure evidence.

## 2. Reported Blockers

### B-M0-01 — Docker Desktop and Compose runtime unavailable

| Item | Finding |
|---|---|
| Affected component | Docker Desktop, PostgreSQL 17 development/test services, API/worker/web Compose wiring, container health checks, migrator container, and Compose shutdown. |
| Root cause | Docker Desktop is configured to create its engine disk under `/Volumes/External SSD/docker-data/DockerDesktop`. `/Volumes/External SSD` is not mounted, so the Docker engine cannot start. |
| Violated requirement | The Docker Compose environment and PostgreSQL 17 development/test workflows must be operational and runtime-verified. Configuration validation alone is insufficient. |
| Repository files involved | `compose.yaml`, `Dockerfile.backend`, `Dockerfile.web`, `Makefile`, `.github/workflows/ci.yml`, and Docker workflow documentation. The immediate fault is in host Docker Desktop state outside the repository. |
| Failing checks | `docker info --format '{{.ServerVersion}}'` reports that Docker Desktop is unable to start. `docker compose up --build -d postgres postgres-test api worker web` cannot connect to `/Users/mac/.docker/run/docker.sock`. |
| Architectural decision required | No product architecture decision is required if the approved external volume is mounted. Moving or resetting Docker Desktop's disk image is an operational, potentially data-affecting decision and requires the workstation owner. |
| Minimum resolution | Mount the expected external volume at `/Volumes/External SSD`, or explicitly approve a safe Docker Desktop disk-image relocation/recovery procedure. Then run the complete Compose build, migration, health, readiness, worker, and shutdown smoke sequence. |

Recent Docker error evidence:

> `starting engine: engine linux/virtualization-framework failed to start: creating disk folder /Volumes/External SSD/docker-data/DockerDesktop: mkdir /Volumes/External SSD: permission denied`

No Docker settings or data were reset, moved, or deleted during this review.

### B-M0-02 — Dedicated repository and target GitHub CI unavailable

| Item | Finding |
|---|---|
| Affected component | Repository boundary, GitHub Actions discovery, remote CI evidence, branch protection, review policy, and final Go module identity. |
| Root cause | ADR-000 leaves the GitHub organization and repository name pending. The product remains an untracked subdirectory of an unrelated parent Git repository. The parent has no commits and no `origin`. |
| Violated requirement | The repository target must be approved, the product directory must be the dedicated repository root, and the CI quality gates must execute in their target GitHub environment. |
| Repository files involved | `docs/adr/ADR-000-bootstrap-repository.md`, `.github/workflows/ci.yml`, `go.mod`, `docs/engineering/ci.md`, and repository-structure documentation. |
| Failing or unavailable checks | `git remote get-url origin` fails with `No such remote 'origin'`; `git log -1` fails because the parent branch has no commits. GitHub-hosted workflow execution and required-check evidence are unavailable. |
| Architectural decision required | Yes. The product/technical owner must approve the GitHub organization, repository name, default branch, branch protections, and whether the neutral Go module path is final. External GitHub repository access is also required. |
| Minimum resolution | Approve the repository identity, establish this product directory as the dedicated repository root without absorbing the unrelated parent application, confirm or replace the Go module path, publish to GitHub, configure required branch checks/review, and run the workflow successfully. |

Creating an arbitrary local nested repository or selecting a GitHub owner/name would not close the approved decision and was intentionally not done.

## 3. Changes Made

No application, infrastructure, CI, database, API contract, or business-feature implementation was changed. The existing bootstrap implementation did not cause either blocker.

This review added only closure evidence and linked it from the original Bootstrap Completion Report. Temporary test databases, binaries, Corepack data, and Playwright browser data were placed under `/private/tmp`; they are not repository artifacts.

No Authentication schema, migration, endpoint, service, repository, middleware, UI, key, password hash, or token behavior was created.

## 4. Verification Evidence

### Blocker diagnostics

| Command | Result | Output summary |
|---|---|---|
| `docker desktop status` | Blocked | Docker Desktop remained `starting`. |
| `docker info --format '{{.ServerVersion}}'` | Failed | Docker Desktop reported that it was unable to start. |
| `docker desktop logs --priority 2 --since -10m` | Diagnostic passed | Identified the absent `/Volumes/External SSD` engine-disk path as the startup root cause. |
| `ls -ld /Volumes '/Volumes/External SSD' ...` | Diagnostic passed | `/Volumes` exists; `/Volumes/External SSD` does not. |
| `docker compose config --quiet` | Passed | Compose configuration is syntactically valid. |
| `docker compose up --build -d postgres postgres-test api worker web` | Failed/blocked | Docker socket does not exist because the engine did not start. No container was created. |
| `git -C . rev-parse --show-toplevel` | Failed acceptance condition | Resolved to `/Users/mac/Documents/AI Automation 2`, not the product directory. |
| `git -C . remote get-url origin` | Failed/blocked | No `origin` remote exists. |
| `git -C . status --short` | Diagnostic passed | Product and unrelated parent files are untracked in the parent repository. |
| `git -C . log -1 --format='%H %cI %s'` | Failed/blocked | Parent branch has no commits. |

### Frontend

The final checks used Node `24.16.0` and repository-pinned pnpm `10.18.3`.

| Command | Result | Output summary |
|---|---|---|
| `pnpm format:check` | Passed | All matched files use Prettier formatting. |
| `pnpm lint` | Passed | ESLint completed with zero warnings. |
| `pnpm typecheck` | Passed | TypeScript check completed without errors. |
| `pnpm test` | Passed | Two Vitest files and four tests passed. |
| `NEXT_PUBLIC_API_BASE_URL='http://localhost:8080/api/v1' pnpm build` | Passed | Next.js 16.2.11 production build completed; `/` and `/_not-found` were generated. |
| `pnpm --filter @portfolio/web exec playwright install chromium` | Passed | Lockfile-compatible Chromium was made available in a temporary browser cache. |
| `PLAYWRIGHT_BROWSERS_PATH='/private/tmp/portfolio-playwright' pnpm test:e2e` | Passed | The Chromium bootstrap journey passed: one test, zero failures. |

An initial build without `NEXT_PUBLIC_API_BASE_URL` failed as designed because production configuration validation rejects the missing public API URL. The configured build passed. Initial E2E attempts were blocked by sandbox port permissions and a missing matching Chromium binary; the final authorized, isolated run passed.

### Backend

The final checks used Go `1.26.5`.

| Command | Result | Output summary |
|---|---|---|
| `test -z "$(gofmt -l backend)"` | Passed | No unformatted Go files. |
| `go vet ./...` | Passed | No vet findings. |
| `sh scripts/check-module-boundaries.sh` | Passed | No forbidden cross-module infrastructure import. |
| `go test ./...` | Passed | Config, HTTP server, and worker test packages passed; composition/generated packages contain no tests. |
| `go build -o /private/tmp/portfolio-bootstrap-api ./backend/cmd/api` | Passed | API binary built. |
| `go build -o /private/tmp/portfolio-bootstrap-worker ./backend/cmd/worker` | Passed | Worker binary built. |

### Database, contracts, and process lifecycle

| Command | Result | Output summary |
|---|---|---|
| `pnpm contract:lint` | Passed | OpenAPI 3.1 document validated. |
| `pnpm contract:generate` plus before/after SHA comparison | Passed | Generated TypeScript contract was deterministic and unchanged. |
| `sqlc generate` plus before/after generated-directory SHA comparison | Passed | Generated Go queries were deterministic and unchanged. |
| `go run github.com/pressly/goose/v3/cmd/goose@v3.26.0 -dir backend/migrations postgres "$m0_db_url" up` against an empty isolated database | Passed | Platform bootstrap migration applied and database reached version 1. |
| The same goose command a second time | Passed | Goose reported no migrations to run at version 1. |
| `TEST_DATABASE_URL="$m0_db_url" go test -count=1 -tags=integration ./backend/internal/platform/database` | Passed | Generated sqlc health query reached the isolated PostgreSQL database. |
| Local API liveness/readiness and correlation curls | Passed | Liveness and readiness returned 200 with the database available; supplied `m0-check-123` was propagated. |
| Readiness after stopping PostgreSQL | Passed | Readiness returned 503 with the standard error envelope and a correlation ID; liveness remained 200. |
| `kill -INT` and `wait` for API and worker | Passed | Both processes shut down cleanly; worker emitted start, dependency-success, and stop logs. |

The isolated native PostgreSQL check confirms repository behavior but does not close B-M0-01 because it is not the required PostgreSQL 17 Compose topology.

### Security and delivery controls

| Command | Result | Output summary |
|---|---|---|
| `pnpm audit --audit-level high` | Passed | No known JavaScript vulnerabilities. |
| `GOTOOLCHAIN=go1.26.5 ... govulncheck ./...` | Passed | No called-symbol vulnerabilities; zero code-affecting vulnerabilities. |
| `gitleaks detect --no-git --source . --config .gitleaks.toml --redact --exit-code 1` | Passed | No leaks found in the product source scan. |
| GitHub Actions workflow execution | Blocked | No dedicated GitHub repository or remote exists. Local constituent checks do not replace target-environment execution. |

An initial `govulncheck` invocation automatically selected Go 1.25 and could not load this Go 1.26 module. Rebuilding and running the scanner with `GOTOOLCHAIN=go1.26.5` passed; this was a verification-toolchain correction, not a code change.

## 5. M0 Acceptance-Criteria Evaluation

| Criterion | Status | Evidence and disposition |
|---|---|---|
| Approved architecture is represented in the repository | Passed | Modular-monolith platform-only layout, OpenAPI-first contract, separate API/worker, and no business modules match the approved design. |
| Workspace structure follows the approved strategy | Blocked | Isolation exists, but the approved dedicated repository identity/root gate remains open. |
| Frontend foundation is operational | Passed | Format, lint, type, unit, production build, and browser smoke checks passed. |
| Backend API foundation is operational | Passed | Vet, tests, build, startup, request behavior, and graceful shutdown passed. |
| Worker foundation is operational | Passed | Build, lifecycle test, live dependency heartbeat, and graceful shutdown passed. |
| PostgreSQL development and testing workflows are operational | Blocked | Native disposable PostgreSQL passed; required PostgreSQL 17 Compose execution is unavailable. |
| Goose migration workflow is operational | Passed | Empty-database migration and repeat execution passed. |
| sqlc workflow is operational | Passed | Generation was deterministic and the generated health query passed integration testing. |
| OpenAPI-first workflow is operational | Passed | Source validation and deterministic consumer generation passed. |
| Contract validation is operational | Passed | Redocly validation passed; only approved bootstrap operations exist. |
| Test foundations are operational | Blocked | Unit, integration, contract, and browser foundations pass; required Compose smoke remains blocked. |
| CI quality gates are operational | Blocked | Workflow coverage is present, but GitHub has not discovered or executed it in a dedicated repository. |
| Logging and correlation IDs are operational | Passed | Safe structured request/lifecycle logs and correlation generation/propagation were observed. |
| Health and readiness checks behave correctly | Passed | Database-up and database-down behavior passed. |
| Secret handling is documented and enforced | Passed | Policy is documented; gitleaks and dependency/vulnerability scans passed. |
| Module boundary rules are documented and enforceable | Passed | Rules are documented and the boundary check passed. |
| Documentation reflects the actual implementation | Passed | The original report and this dated review record actual capabilities and external blockers. |
| No business functionality was implemented accidentally | Passed | Repository inspection found only neutral/platform foundation code and operational contracts. |
| Both reported blockers are resolved | Blocked | Both require external state or an approved ownership decision. |
| No critical Bootstrap verification remains failed or blocked | Blocked | Compose/PostgreSQL 17 runtime and GitHub-hosted CI remain unavailable. |

## 6. M0 Closure Decision

**M0 remains Open.** The closure rule is not satisfied because B-M0-01 and B-M0-02 remain blocked, and critical Compose and target-CI evidence is unavailable. Documentation evidence cannot substitute for runtime evidence.

The Authentication Phase 1 Execution Plan was not created. Creating it while M0 is open would violate the requested gated workflow.

## 7. Deferred Non-Blocking Items

These items do not resolve either current M0 blocker and remain deferred to their owning milestones:

- Production backup retention, RPO, and RTO.
- Primary price-provider selection and display license, required before price work.
- Object storage, documents, AI research, alerts, and monthly review infrastructure.
- Production deployment and monitoring stack selection.

No critical M0 capability is classified as a deferred non-blocking item.

## 8. Risks and Required External Actions

Before M0 can be re-reviewed:

1. Restore Docker Desktop by mounting `/Volumes/External SSD` or approving a data-safe Docker disk-image relocation. Then run the PostgreSQL 17 Compose smoke, migration, readiness-degradation, worker, web, and shutdown checks.
2. Approve the GitHub organization, repository name, default branch, required checks/review policy, and Go module identity. Establish the product directory as the dedicated repository root, publish it, and obtain a fully passing GitHub Actions run.

Before Authentication implementation (after M0 closure), the Decision Closure Specification also requires approval of deployment hostname/cookie topology and an Ed25519 secret-management and rotation mechanism. These decisions do not authorize Authentication implementation in this review.
