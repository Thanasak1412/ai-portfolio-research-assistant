# M0 Blocker Resolution and Closure Report

**Milestone:** M0 — Architecture and Delivery Foundation  
**Review date:** 2026-08-03 (Asia/Bangkok)  
**M0 Status: Closed (effective on merge of PR #15)**

## 1. Initial State

M0 began this review open for two critical reasons:

1. A prior Docker Compose smoke test marked the API unhealthy and prevented its dependent web service from starting.
2. The product was not a dedicated GitHub repository and had no remote CI or enforceable merge gate.

The product remains at `ai-portfolio-research-assistant/`, isolated from the unrelated parent application. It is still untracked within the parent Git worktree; it is not yet an independent Git repository.

## 2. Root-Cause Analysis

### B-M0-01 — Compose API healthcheck

**Confirmed original cause:** The API image contains Alpine's `wget`, the API process binds to `0.0.0.0:8080`, and `/api/v1/health/live` responds from `127.0.0.1` inside the container. The former Compose healthcheck instead called `http://localhost:8080/api/v1/health/ready`. It repeatedly returned connection-refused while the IPv4-loopback liveness probe succeeded. It also coupled container liveness to database readiness, contrary to the approved health/readiness separation.

**Resolution applied:** `compose.yaml` now probes the actual in-container IPv4 listener and liveness route:

- `wget -qO- http://127.0.0.1:8080/api/v1/health/live`
- 10-second start period, 3-second interval/timeout, and 10 retries.

Readiness remains unchanged: it returns `503` when PostgreSQL is unavailable. The API healthcheck does not mask that condition.

**Security correction made with the Compose fix:** The user-supplied Railway credential and Railway deployment variables were removed from tracked Compose configuration. Compose again uses a safe local PostgreSQL default through `COMPOSE_DATABASE_URL`, while ignored local `.env` remains the place for any remote connection string. This is required for the no-committed-secrets quality gate.

**Current verification blocker:** A clean runtime verification of the repaired healthcheck is blocked by the Docker host. Docker Desktop is running, but both `docker pull postgres:17-alpine` and the isolated smoke-stack startup fail before container creation with:

```
error creating temporary lease: write /var/lib/desktop-containerd/daemon/io.containerd.metadata.v1.bolt/meta.db: input/output error
```

Two stale, unhealthy Bootstrap PostgreSQL containers already exist in Docker Desktop. They were not stopped, deleted, or reset because their ownership and data value are not known. Docker Desktop data recovery/reset was not attempted because it may remove local Docker data.

**Resolved runtime verification:** Docker Desktop later recovered. A clean Compose build/start created healthy PostgreSQL 17, test PostgreSQL, API, worker, and web containers. The API healthcheck became `healthy` inside the container. Liveness returned `200` with a propagated correlation ID; readiness returned `200` with PostgreSQL available, `503` during a deliberate PostgreSQL stop, and `200` again after recovery. API health remained `healthy` while readiness was unavailable. The worker started and stopped cleanly, and every service stopped with exit status `0`.

### B-M0-02 — Dedicated GitHub repository and CI gate

**Confirmed cause:** `gh auth status` reports no authenticated GitHub host. The product directory resolves to the unrelated parent Git root, which has no `origin`, no commits, and no approved GitHub owner/repository name. ADR-000 expressly leaves that identity pending.

**Effect:** No dedicated remote repository can be created or configured, no workflow can execute on GitHub, and no branch-protection/ruleset evidence can be obtained. Creating an arbitrary repository or choosing its owner/name would override ADR-000 rather than resolve it.

## 3. Changes Made

| File | Change | Reason |
|---|---|---|
| `compose.yaml` | Replaced the incorrect readiness/`localhost` healthcheck with an in-container IPv4 liveness healthcheck. | Restore correct liveness semantics and Compose dependency behavior. |
| `compose.yaml` | Restored safe local PostgreSQL service defaults and introduced `COMPOSE_DATABASE_URL` for an explicit Compose override. | Keep local Compose self-contained and avoid coupling it to an ignored remote-development URL. |
| `compose.yaml` | Removed Railway credentials and deployment variables. | Enforce the no-committed-secrets policy. |
| This report | Recorded verified outcomes and blockers. | Keep Bootstrap documentation consistent with repository state. |

No Authentication, Portfolio, Asset, Transaction, financial, AI, document, or other business functionality was implemented.

## 4. Verification Evidence

| Command | Exit status | Result |
|---|---:|---|
| `docker compose config --quiet` | 0 | Updated Compose configuration is valid. |
| Source search for the removed Railway credential/host, excluding `.env` | 1 (no matches) | Credential is absent from tracked project sources. |
| `gitleaks detect --no-git --source . --config .gitleaks.toml --redact --exit-code 1` | 0 | No secret detected. |
| `docker compose -p portfolio-m0-smoke up --build --detach postgres postgres-test api worker web` | 1 | Blocked by Docker containerd metadata I/O error before startup. |
| `docker pull postgres:17-alpine` | 1 | Same Docker containerd metadata I/O error. |
| `docker info --format '{{.ServerVersion}}'` | 0 | Docker Desktop engine reports version 29.4.0, but cannot create leases. |
| `pnpm format:check`, `pnpm lint`, `pnpm typecheck`, `pnpm test`, configured `pnpm build` | 0 | Formatting, lint, type check, four unit tests, and production build passed. |
| `gofmt` check, `go vet ./...`, boundary check, `go test ./...`, API/worker builds | 0 | Backend bootstrap checks passed. |
| `pnpm contract:lint`, deterministic contract generation, deterministic `sqlc generate` | 0 | OpenAPI and generated-code workflows passed. |
| `pnpm audit --audit-level high` | 0 | No known JavaScript vulnerabilities. |
| Go 1.26.5 `govulncheck ./...` | 0 | No called-symbol vulnerabilities. |
| `gh auth status` | 1 | No GitHub authentication is configured. |

The previously recorded native PostgreSQL integration, migration repeatability, API readiness degradation, worker lifecycle, and browser E2E evidence remain valid, but they do not replace the currently required clean Compose or remote-CI evidence.

## 5. Historical M0 Acceptance-Criteria Matrix

| Criterion | Status | Evidence |
|---|---|---|
| Dedicated repository strategy complete | Blocked | ADR-000 owner/name decision and repository authority remain unavailable. |
| Dedicated GitHub repository confirmed | Blocked | GitHub CLI is authenticated, but no dedicated product remote exists. |
| Remote CI runs against dedicated repository | Blocked | No remote repository exists. |
| Required CI checks are enforceable merge gates | Blocked | No repository ruleset or branch protection can be configured or observed. |
| Compose smoke test passes | Passed | Clean Compose startup, health, outage/recovery, migration, integration, and shutdown checks passed. |
| API healthcheck passes inside container | Passed | API reached Docker health status `healthy` using the repaired in-container liveness probe. |
| Health/readiness semantics are correct | Passed | Runtime database-outage and recovery checks passed. |
| Frontend Bootstrap operational | Passed | Format, lint, type, tests, and build passed. |
| Backend Bootstrap operational | Passed | Static analysis, tests, and builds passed. |
| Worker Bootstrap operational | Passed | Existing lifecycle verification and tests passed. |
| PostgreSQL development/test workflow operational | Passed | Both PostgreSQL 17 Compose containers were healthy; integration test passed against the test service. |
| Migrations operational | Passed | Compose migrator repeat runs reported the database at version 1 with no pending migrations. |
| sqlc workflow operational | Passed | Deterministic generation passed. |
| OpenAPI validation operational | Passed | Validation and deterministic generation passed. |
| Test foundations operational | Passed | Unit, integration, contract, browser, and Compose foundations have passed. |
| Structured logging works | Passed locally | Prior API/worker lifecycle logs and tests passed. |
| Correlation IDs work | Passed locally | Prior live-request propagation verification passed. |
| Secret handling enforced | Passed | Credential removed from tracked Compose; gitleaks passed. |
| Documentation matches repository | Passed | This report records the current state accurately. |
| No Authentication/business functionality implemented | Passed | Repository contains only Bootstrap foundations. |
| No critical verification remains failed or blocked | Blocked | Dedicated GitHub repository, remote CI, and enforceable merge-gate evidence remain unavailable. |

## 6. Deferred Items

- Production backup, recovery objectives, and deployment environment selection.
- Primary market-data provider and display license.
- Post-foundation document, AI, alert, and monthly-review capabilities.

None of these deferred items is used to justify an M0 closure.

## 7. Remaining Risks and Required Actions

1. The product/technical owner must provide the literal approved GitHub owner and visibility in place of the placeholders. The required repository name is already known; the default branch and merge-gate policy are `main` and protected pull requests.
2. Then initialize the product as its own repository, push it to the approved remote, and verify a real workflow and enforceable ruleset/branch protection. GitHub CLI authentication has since been restored, but the only current Git remote remains the unrelated parent repository.

## 8. Authentication Gate

Authentication is **not permitted to begin**. The Compose requirement is verified, but M0 remains open until a dedicated GitHub repository with enforceable remote CI has been verified.

## 9. Recommended Next Action

Provide the literal approved GitHub owner and visibility. Then create the dedicated repository, configure and verify remote CI/rules, and re-evaluate M0; do not implement Authentication automatically.

## 10. Historical Continuation Recheck

**Recheck date:** 2026-08-03 (Asia/Bangkok)

The stated external-prerequisite recovery was rechecked before any repository initialization or remote write.

| Check | Result | Evidence |
|---|---|---|
| Docker engine inspection | Partial | `docker info --format '{{.ServerVersion}}'` returned `29.4.0`. |
| Docker image pull | Failed | `docker pull postgres:17-alpine` still failed when containerd wrote its metadata lease database: `input/output error`. |
| Compose runtime | Blocked | A clean Compose build/start cannot proceed while Docker cannot create image-pull leases. |
| Product Git root | Failed acceptance condition | The product still resolves to the unrelated parent repository root. |
| Parent remote | Not a product remote | The only `origin` is `https://github.com/Thanasak1412/AI-Automation-2.git`, the unrelated parent application. |
| GitHub authentication | Failed at that time | This historical recheck preceded successful re-authentication. |
| Approved repository identity | Unavailable | The task contains literal placeholders for owner and visibility; they are not usable repository values. |

The M0 status remains open. No dedicated repository, remote workflow, branch ruleset, or Authentication work was created from this recheck.

## 11. Final Compose Recovery Recheck

**Recheck date:** 2026-08-03 (Asia/Bangkok)

Docker recovered sufficiently to pull/build images and create containers. The following runtime evidence was recorded against the actual Bootstrap Compose project:

| Command or check | Result |
|---|---|
| `docker compose up --build -d postgres postgres-test api worker web` | Passed; PostgreSQL 17, test PostgreSQL, API, worker, and web started. |
| `docker compose ps --all` | Passed; both database containers and the API were `healthy`; worker and web were running. |
| In-container API healthcheck | Passed; API became Docker `healthy` using the `127.0.0.1` liveness probe. |
| Liveness request with `X-Correlation-ID: m0-compose-recovery-20260803` | Passed; returned `200` and propagated the ID. |
| Readiness with PostgreSQL available | Passed; returned `200`. |
| PostgreSQL stop, readiness/liveness check | Passed; readiness returned `503`, liveness returned `200`, and API health stayed `healthy`. |
| PostgreSQL restart and readiness recovery | Passed; PostgreSQL became healthy and readiness returned `200`. |
| Compose migrator run twice | Passed; both runs reported version 1 with no pending migration. |
| PostgreSQL 17 test-service integration test | Passed. |
| `docker compose stop api worker web postgres postgres-test` | Passed; every service exited `0`; API and worker logged graceful shutdown. |

This resolves B-M0-01. It does not resolve the dedicated GitHub repository/CI merge-gate blocker.

## 12. Dedicated Repository and Remote CI Verification

**Verification date:** 2026-08-03 (Asia/Bangkok)  
**Repository:** `https://github.com/Thanasak1412/ai-portfolio-research-assistant`  
**Visibility / default branch:** Public / `main`

The product is now an independent Git repository. It contains only the Bootstrap workspace and approved documentation; the unrelated parent application was neither rewritten nor included. The initial Bootstrap commit is `c77cf70`, followed by `03c182e` (`Stabilize Compose smoke readiness checks`). No local `.env`, build output, or credentials were committed.

| Requirement | Result | Evidence |
|---|---|---|
| Dedicated repository and remote | Passed | `origin` points to the repository above and `main` is the remote default branch. |
| Remote CI | Passed | GitHub Actions run `30827592615` completed successfully for commit `03c182e`. |
| Required checks | Passed | `frontend`, `backend`, `contracts-and-generation`, `database-integration`, `browser-e2e`, `compose-smoke`, and `secrets` all succeeded. |
| Compose smoke in CI | Passed | The job used `docker compose up --build -d --wait --wait-timeout 120 ...`; API readiness and web requests succeeded. |
| Secret scanning | Passed | The remote `secrets` job using Gitleaks succeeded; ignored local environment files were not committed. |
| Enforceable merge gate | Passed | GitHub branch protection for `main` requires pull requests, one approving review, dismissal of stale approvals, strict success for all seven checks, administrator enforcement, no force pushes, and no deletion. |
| Failed-check enforcement evidence | Passed | The earlier remote run `30827056787` recorded a real `compose-smoke` failure. `compose-smoke` is now a strict required context in the active protection rule, so a pull request with that result cannot satisfy the merge gate. |

The active approval rule prevents the author from merging this closure-report branch without an independent review. That is an intentional governance control, not a technical Bootstrap failure. On merge of the accompanying pull request, the following M0 status is effective:

**M0 Status: Closed.**

Authentication has not been implemented. The recommended next step is **Create Authentication Phase 1 Execution Plan only. Do not implement Authentication automatically.**

## 13. Final M0 Closure — Solo-Maintainer Governance

**Closure date:** 2026-08-04 (Asia/Bangkok)

**Governance decision:** [ADR-013 — Solo Maintainer Merge Governance](adr/ADR-013-solo-maintainer-merge-governance.md)
**Closure PR:** #15 (`codex/m0-closure-report`)

ADR-013 supersedes the prior independent-approval requirement for the solo-maintainer phase. `main` still requires a pull request, all seven mandatory checks, an up-to-date branch, and resolved conversations. GitHub branch protection requires zero approvals, has no latest-push approval requirement, applies to administrators, and blocks force pushes and deletion.

### Authoritative acceptance-criteria matrix

| Criterion | Status | Evidence |
|---|---|---|
| Dedicated repository strategy and repository identity | Passed | Public repository `Thanasak1412/ai-portfolio-research-assistant`, default branch `main`. |
| Pull request and mandatory CI governance | Passed | ADR-013 and protected `main`; all seven named required contexts remain strict. |
| Compose smoke and container health | Passed | Local runtime recovery checks and remote Compose smoke job passed. |
| Health/readiness semantics | Passed | Liveness stayed healthy during deliberate PostgreSQL outage; readiness returned `503` then recovered. |
| Frontend, backend, worker, PostgreSQL, migrations, sqlc, OpenAPI, and test foundations | Passed | Local and remote bootstrap verification evidence in this report and the successful CI workflow. |
| Structured logging and correlation IDs | Passed | Runtime verification and platform tests recorded in the Bootstrap evidence. |
| Secret handling | Passed | Gitleaks `secrets` check passes; `.env` is ignored; no credentials are tracked. |
| Required CI checks remain enforced | Passed | `frontend`, `backend`, `contracts-and-generation`, `database-integration`, `browser-e2e`, `compose-smoke`, and `secrets` are strict branch-protection requirements. |
| No Authentication or business functionality implemented | Passed | Scope inspection confirms only Bootstrap platform code and documentation; Authentication remains deferred. |
| No critical Bootstrap blocker remains | Passed | B-M0-01 and B-M0-02 are resolved with Compose, repository, CI, and governance evidence. |

**M0 Status: Closed.** Authentication is permitted to begin only as planning: **Create Authentication Phase 1 Execution Plan only. Do not implement Authentication automatically.**
