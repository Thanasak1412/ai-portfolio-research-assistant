# M2 Portfolio & Asset Acceptance Matrix

- Milestone: M2 — Portfolio & Asset Foundation
- Verification task: `M2-VERIFY-001`
- Approved plan: `M2-PORTFOLIO-ASSET-PLAN-v1`
- Verification branch: `codex/m2-verify-001`
- Protected `main` base: `55fcbc943b75f2f9bc7a2bea06fe20df12180943`
- M2 implementation merge sequence: PRs #34–#40, #43, and #44
- Draft verification PR: [#45](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/45); its current-head CI is the final merge gate.

Result values are `PASS`, `FAIL`, `BLOCKED`, and `NOT_APPLICABLE`. A result is
not final closure evidence until the draft verification PR's required CI checks
pass on its current head.

| Requirement                                                     | Evidence                                                                                                         | Result | Notes                                                                                                      |
| --------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | -----: | ---------------------------------------------------------------------------------------------------------- |
| Approved implementation sequence is merged into `main`          | GitHub PRs #34–#40, #43, #44; merge commits in `main` ancestry                                                   |   PASS | Ordered M2 task sequence is complete.                                                                      |
| Portfolio API is OpenAPI-first and contract-valid               | `pnpm contract:check`; M2 contract tests 10–13                                                                   |   PASS | Create, list, get, update, archive only; all require Bearer authentication.                                |
| Public Portfolio ID remains opaque and ownership-safe           | Contract tests; Portfolio HTTP tests                                                                             |   PASS | Missing, cross-owner, and unparseable IDs use safe not-found behavior.                                     |
| Portfolio create/list/get/update/archive lifecycle              | Portfolio domain, application, HTTP, integration, and real E2E evidence                                          |   PASS | No delete or restore operation exists.                                                                     |
| ACTIVE-only name uniqueness and archived-name reuse             | Migration partial unique index; integration tests; real M2 E2E                                                   |   PASS | Duplicate ACTIVE names reject safely; an archived name may be reused.                                      |
| Archived original aggregate remains after name reuse            | `m2-critical-flow.spec.ts`; merged PR #44                                                                        |   PASS | Archived-list link retains the original archived path.                                                     |
| Portfolio naming and USD base-currency invariants               | Migration constraints; contract tests; frontend validation tests                                                 |   PASS | Names are normalized server-side; base currency is immutable USD.                                          |
| Portfolio state and archive idempotency                         | Contract, application, HTTP, and integration tests                                                               |   PASS | Archive is active-only and repeat archive remains contract-safe.                                           |
| Portfolio persistence constraints, ownership, and concurrency   | `database-integration` CI; Portfolio persistence integration tests                                               |   PASS | Includes owner-scoped listing and active-name contention.                                                  |
| Canonical Asset catalog is read/search only                     | Contract tests 14–17; asset query review                                                                         |   PASS | No public Asset mutation operation or UI is present.                                                       |
| Asset identity and canonical metadata                           | Migration constraints; Asset integration tests; M2 E2E                                                           |   PASS | Supports only EQUITY, ETF, and CRYPTO; public currency is USD.                                             |
| CRYPTO canonical exchange representation                        | Migration `00003`; real M2 E2E direct field assertion                                                            |   PASS | Exchange=`CRYPTO` and Currency=`USD` are targeted directly.                                                |
| Asset deterministic ordering, cursor, filters, and bounds       | Asset query/index review; contract and integration tests                                                         |   PASS | `(normalized_symbol, normalized_exchange, asset_id)` ordering and cursor semantics are tested.             |
| Required database indexes and query plans                       | Asset and Portfolio persistence integration tests; migration review                                              |   PASS | Query-plan verification covers owner/status listing and Asset discovery.                                   |
| Portfolio and Asset frontend flows                              | 73 frontend unit tests; real M2 E2E                                                                              |   PASS | Portfolio create/rename/archive and Asset discovery are browser-observable.                                |
| Authentication ownership boundary remains enforced              | Bearer contract requirements; real HTTPS E2E ownership-isolation flow                                            |   PASS | No client-supplied owner/user ID is authority.                                                             |
| Real HTTPS topology                                             | Local Caddy → Next.js → Go API → PostgreSQL stack                                                                |   PASS | Auth 21/21 and M2 4/4 real browser tests passed.                                                           |
| Protected routing and cross-owner isolation                     | M2 routing suite and real M2 critical flow                                                                       |   PASS | `/app`, Portfolio, and Asset views protect anonymous users.                                                |
| M2 scope excludes financial, provider, and AI functionality     | Source/contract/UI review; negative E2E assertions                                                               |   PASS | No transactions, holdings, prices, valuations, allocations, dashboards, alerts, documents, or AI features. |
| Migration and sqlc workflow                                     | `sqlc generate`; clean generated-code diff; remote `database-integration`                                        |   PASS | Remote CI verifies empty, upgrade, down, and re-up migration paths.                                        |
| Module boundaries remain enforced                               | `sh scripts/check-module-boundaries.sh`                                                                          |   PASS | Domain/application layers do not import transport or generated database packages directly.                 |
| Frontend formatting, lint, typecheck, unit tests, and build     | Local commands; remote `frontend` CI                                                                             |   PASS | Local Node 24 runner passed 21 files / 73 tests.                                                           |
| Backend formatting, vet, tests, builds, and vulnerability check | Local Go 1.26.6 commands; remote `backend` CI                                                                    |   PASS | `govulncheck` found no reachable vulnerabilities.                                                          |
| OpenAPI lint, contract tests, generation, and drift             | `pnpm contract:check`; `sqlc generate`; clean diff                                                               |   PASS | 17 contract tests passed locally.                                                                          |
| Dependency and secret scanning                                  | `pnpm audit --audit-level high`; remote `secrets`; GitGuardian on PR #44                                         |   PASS | Local audit reports no known vulnerabilities; remote Gitleaks and GitGuardian passed.                      |
| Compose smoke and worker/API/web health                         | Remote `compose-smoke` job 96767104609                                                                           |   PASS | API readiness and web diagnostic endpoint passed.                                                          |
| Seven required remote CI jobs on final M2 implementation head   | [workflow 32480991049](https://github.com/Thanasak1412/ai-portfolio-research-assistant/actions/runs/32480991049) |   PASS | All seven required jobs passed.                                                                            |

## Verification limitation and final gate

The isolated local Compose profile deliberately does not publish `postgres-test`
to the host. A direct host integration-test command therefore cannot connect to
`127.0.0.1:5433`; this is not a product failure. The same disposable database
was migrated to version 3, seeded with 26 synthetic Assets, and used by the
passing real browser suites. The required remote `database-integration` job
also passed the full PostgreSQL integration suite.

The initial local parallel M2 browser run had one route-warm-up timing failure
at the first Portfolio creation. A serial rerun passed, followed by the complete
configured two-worker M2 suite passing 4/4. No production code was changed.

M2 can be proposed for closure only after this branch's draft PR is reviewed,
all seven required CI jobs pass on its current head, and ADR-013's
solo-maintainer merge governance is satisfied.
