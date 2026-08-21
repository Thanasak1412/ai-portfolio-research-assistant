# M2 Portfolio & Asset Completion Report

## Metadata

- Milestone: M2 — Portfolio & Asset Foundation
- Verification task: `M2-VERIFY-001`
- Approved plan: `M2-PORTFOLIO-ASSET-PLAN-v1`
- Verification date: 2026-08-21 (Asia/Bangkok)
- Protected `main` base SHA: `55fcbc943b75f2f9bc7a2bea06fe20df12180943`
- Verification branch: `codex/m2-verify-001`
- Verification PR: draft PR created from this branch; its pull-request record is the authoritative source for its current head and CI run.

## Implementation Traceability

| Task              | Merged PR | Merge commit                               |
| ----------------- | --------: | ------------------------------------------ |
| `M2-PLAN-001`     |       #34 | `e237d7…`                                  |
| `M2-CONTRACT-001` |       #35 | `bcd733…`                                  |
| `M2-DB-001`       |       #36 | `e752933…`                                 |
| `M2-BE-001`       |       #37 | `3d0c807…`                                 |
| `M2-BE-002`       |       #38 | `9807ac…`                                  |
| `M2-BE-003`       |       #39 | `6c3256…`                                  |
| `M2-FE-001`       |       #40 | `cf3f6…`                                   |
| `M2-FE-002`       |       #43 | `bd42a3…`                                  |
| `M2-E2E-001`      |       #44 | `55fcbc943b75f2f9bc7a2bea06fe20df12180943` |

## Delivered Scope

M2 delivers an authenticated, owner-scoped Portfolio lifecycle: create, list
by ACTIVE or ARCHIVED status, retrieve, rename, and archive. A Portfolio has an
opaque identifier, a normalized unique ACTIVE name per owner, immutable USD
base currency, and an ACTIVE/ARCHIVED lifecycle.

M2 also delivers global canonical Asset discovery: authenticated read/search,
exact AssetType filters, deterministic cursor pagination, and the EQUITY, ETF,
and CRYPTO types. Asset metadata remains provider-neutral and uses canonical
exchange and USD currency values.

## Explicitly Excluded Scope

No user administration, Portfolio deletion/restoration, Asset mutation UI,
Transactions, cash ledger, prices, providers, holdings, lots, valuations,
allocations, dashboards, alerts, documents, AI, or monthly-review functionality
was added. M2 also does not add client-side ownership authority or persisted
Authentication tokens.

## Acceptance Evidence

See the complete [M2 Portfolio & Asset Acceptance Matrix](m2-portfolio-asset-acceptance-matrix.md). All reviewed implementation requirements are `PASS` before the final verification PR gate.

### Functional and persistence evidence

- `00003_portfolio_asset_foundation.sql` provides Portfolio and Asset tables,
  lifecycle constraints, owner foreign keys, ACTIVE-name uniqueness, canonical
  Asset constraints, and query-supporting indexes.
- The Portfolio and Asset sqlc query groups expose only the approved lifecycle
  and catalog-read behavior.
- The remote database job on workflow 32480991049 completed empty-database,
  current-main upgrade, down, and re-up migration verification, then passed
  the platform, identity, Portfolio, and Asset integration packages.
- Contract tests enforce the frozen public API and forbid later M2 operations
  and schemas.

### Real browser evidence

The local real stack was rebuilt with Caddy HTTPS, Next.js, Go API, and
PostgreSQL. Goose migrated `postgres-test` through versions 1, 2, and 3, then
inserted 26 synthetic M2 Assets. The following passed without Auth API mocks:

- `PLAYWRIGHT_AUTH_E2E_IGNORE_HTTPS_ERRORS=true pnpm test:e2e:auth` — 21 passed.
- `PLAYWRIGHT_AUTH_E2E_IGNORE_HTTPS_ERRORS=true pnpm test:e2e:m2` — 4 passed.

The M2 critical flow verifies Portfolio creation, duplicate-name rejection,
rename, archive, ACTIVE/ARCHIVED selection, archived-name reuse, persistence of
the original archived aggregate, cross-owner isolation, canonical EQUITY/ETF/
CRYPTO discovery, CRYPTO Exchange=`CRYPTO`, Currency=`USD`, filtering, and
cursor pagination.

### Local verification commands

| Command                                                                                         | Result                                           |
| ----------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| `git diff --check`                                                                              | PASS                                             |
| `pnpm format:check`                                                                             | PASS                                             |
| `pnpm lint`                                                                                     | PASS                                             |
| `pnpm typecheck`                                                                                | PASS                                             |
| `pnpm contract:check`                                                                           | PASS — 17 contract tests; generated output clean |
| `NEXT_PUBLIC_API_BASE_URL=https://app.localhost:3443/api/v1 pnpm --filter @portfolio/web build` | PASS                                             |
| Node 24 Vitest runner                                                                           | PASS — 21 files / 73 tests                       |
| `sqlc generate` and generated-code drift diff                                                   | PASS                                             |
| `test -z "$(gofmt -l backend)"`                                                                 | PASS                                             |
| `go vet ./...`                                                                                  | PASS                                             |
| `sh scripts/check-module-boundaries.sh`                                                         | PASS                                             |
| `go test ./...`                                                                                 | PASS                                             |
| `go build ./backend/cmd/api ./backend/cmd/worker`                                               | PASS                                             |
| `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`                                         | PASS — no reachable vulnerabilities              |
| `pnpm audit --audit-level high`                                                                 | PASS — no known vulnerabilities                  |
| Real Auth browser suite                                                                         | PASS — 21/21                                     |
| Real M2 browser suite                                                                           | PASS — 4/4                                       |

The canonical `pnpm test` command cannot start locally under Node 20.11.1 due
to Vite's ESM loader requirements. The project requires Node 24; the direct
Node 24 Vitest runner passed. The final remote `frontend` job runs Node 24 and
is the mandatory canonical proof.

The direct host integration-test command could not reach `postgres-test`
because the isolated Compose profile has no host port mapping. This does not
block closure: migrations, seeding, and real browser tests executed against the
same disposable database, while the remote `database-integration` job passed
its complete suite.

## Prior Remote CI Evidence

The merged M2 E2E head `b5df67ecc7740f7038e00a6e5b5a2d50e84bcc21` passed
[bootstrap-quality-gates workflow 32480991049](https://github.com/Thanasak1412/ai-portfolio-research-assistant/actions/runs/32480991049):

| Required job               | Result |         Job |
| -------------------------- | -----: | ----------: |
| `frontend`                 |   PASS | 96767104789 |
| `backend`                  |   PASS | 96767104836 |
| `contracts-and-generation` |   PASS | 96767104802 |
| `database-integration`     |   PASS | 96767104803 |
| `browser-e2e`              |   PASS | 96767104911 |
| `compose-smoke`            |   PASS | 96767104609 |
| `secrets`                  |   PASS | 96767104814 |

GitGuardian Security Checks also passed on PR #44. The browser job completed
migrations to version 3, the 26-Asset seed, Auth E2E, the rate-limit reset, M2
E2E, and teardown successfully.

## Findings and Risks

- Critical: 0
- Major: 0
- Blocking minor: 0
- Informational: 2

The two informational local-environment limitations are the host Node 20
runtime and the intentionally unexposed `postgres-test` port. A first parallel
M2 browser run encountered a local route-warm-up timing failure; the serial
rerun and subsequent full configured 4-test run passed without code changes.
These are not a substitute for the final PR's required CI.

## M2 Closure Recommendation

All implementation, security-boundary, migration, contract, test, and real
HTTPS runtime evidence is present. M2 may be closed after this verification PR
passes all seven required CI jobs on its current head and satisfies ADR-013.

M2 Status: Pending M2-VERIFY-001 PR review and merge

## Recommended Next Step

Review `M2-VERIFY-001`, confirm its current-head seven-job CI evidence, and
merge it under ADR-013 Solo Maintainer Merge Governance. Do not begin M3 before
that merge.
