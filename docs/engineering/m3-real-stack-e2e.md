# M3 real-stack Transaction Ledger verification

`M3-E2E-001` exercises one serial critical lifecycle through:

```text
Chromium → https://app.localhost:3443 → Caddy → Next.js production → Go API → postgres-test
```

The suite uses real browser registration, Portfolio creation and navigation,
canonical Asset selection, BUY review/confirmation, history, complete correction,
immutable relationship navigation, and cross-owner not-found behavior. It checks
the absence of Edit/Delete, direct REVERSAL/ADJUSTMENT creation, derived financial
UI, external provider traffic, and script-visible Authentication credentials.

Two supplemental HTTPS commands use the same captured real principal in memory:
same-key/same-body replay must return the identical committed Transaction, and a
CRYPTO command must return `422 ASSET_FINANCIALLY_INELIGIBLE` without adding a fact.
These requests reach Caddy and the real API; no endpoints are intercepted or
mocked. Replay proves backend safe retry; existing focused frontend tests cover
memory-only retry-key retention. The suite does not simulate an uncertain browser
delivery or claim that it does. It does not inspect internal database tables.

## Fixtures and isolation

`apps/web/tests/m3-e2e/fixtures/assets.sql` contains only synthetic references:
M3EQ01 (EQUITY/NYSE/USD), M3ETF01 (ETF/NYSEARCA/USD), and M3CR01
(CRYPTO/CRYPTO/USD). No prices or provider data are seeded. Inserts are safe to
repeat and coexist with the M2 pagination fixtures.

`scripts/seed-m3-e2e-assets.sh` requires exactly one literal environment assignment:

```text
COMPOSE_DATABASE_URL=postgres://portfolio:portfolio_test_local_only@postgres-test:5432/portfolio_test?sslmode=disable
```

It rejects other URLs, duplicate/ambiguous assignments and conflicting inherited
overrides. It never sources the file or accepts a URL argument. Its command targets
only `postgres-test` / `portfolio_test`; SQL also checks the database name.
`node --test scripts/seed-m3-e2e-assets.test.mjs` verifies this guard independently
of Docker. The real fixture execution and repeat seeding are verified on the
disposable stack, not by the safety-test CLI stub.

## Safe local execution

Use Node 24 and the repository's pnpm version. Install the matching Chromium with
`pnpm --filter @portfolio/web exec playwright install chromium` if needed.

The stack uses fixed ports and the existing narrow proxy subnet. Do not run it
alongside the normal development project. Coordinate downtime before taking the
development stack down **without `-v`**; preserve its volumes. Never remove normal
development data to make an E2E test pass.

Run from the repository root in a shell with no conflicting Compose overrides:

```bash
export COMPOSE_PROJECT_NAME=m3-e2e-001
sh scripts/prepare-local-auth-https.sh .local/m3-e2e-tls
AUTH_TLS_DIR=.local/m3-e2e-tls \
  sh scripts/prepare-auth-e2e-env.sh .compose.m3-e2e.env

docker compose --env-file .compose.m3-e2e.env \
  up --build -d --wait --wait-timeout 120 postgres postgres-test
docker compose --env-file .compose.m3-e2e.env --profile tools run --rm migrate-test
sh scripts/seed-m2-e2e-assets.sh .compose.m3-e2e.env
sh scripts/seed-m3-e2e-assets.sh .compose.m3-e2e.env
sh scripts/seed-m3-e2e-assets.sh .compose.m3-e2e.env

WEB_BUILD_TARGET=production NEXT_PUBLIC_API_BASE_URL=https://app.localhost:3443/api/v1 \
  docker compose --env-file .compose.m3-e2e.env --profile auth-https \
  up --build -d --wait --wait-timeout 120 api worker web auth-proxy
sh scripts/verify-auth-https-stack.sh .compose.m3-e2e.env

PLAYWRIGHT_AUTH_E2E_IGNORE_HTTPS_ERRORS=true \
  pnpm --filter @portfolio/web exec playwright test --config playwright.auth.config.ts --retries=0
sh scripts/reset-e2e-auth-rate-limits.sh .compose.m3-e2e.env
PLAYWRIGHT_AUTH_E2E_IGNORE_HTTPS_ERRORS=true pnpm test:e2e:m2
sh scripts/reset-e2e-auth-rate-limits.sh .compose.m3-e2e.env
PLAYWRIGHT_AUTH_E2E_IGNORE_HTTPS_ERRORS=true pnpm test:e2e:m3
```

The Auth zero-retry override may also be supplied directly to Playwright:
`pnpm --filter @portfolio/web exec playwright test --config playwright.auth.config.ts --retries=0`.
The TLS exception is only for ephemeral certificates; all requests remain HTTPS.
For a trusted local certificate omit it. `M3_E2E_BASE_URL` can select another
equivalent real HTTPS test deployment; plaintext origins are rejected.

Clean up only the explicitly named disposable project after verification:

```bash
docker compose --project-name m3-e2e-001 --env-file .compose.m3-e2e.env \
  --profile auth-https down -v
unset COMPOSE_PROJECT_NAME
```

This removes only the disposable project's containers/networks/volumes, including
its tmpfs-backed test database. Never substitute the normal development project.
Generated environment and TLS files are ignored and must not be committed.

## CI and failure evidence

The existing `browser-e2e` job seeds M2 and M3 fixtures, builds the production
HTTPS stack, verifies it, runs Auth, resets test-only rate limits, runs M2 and
its ten lifecycle repetitions, resets rate limits, then executes M3. All seven
ADR-013 job names and failure diagnostics remain unchanged; no eighth gate or
external infrastructure is added. A failing invocation remains a failure.

M3 uses one worker, `fullyParallel: false`, and `retries: 0`. Synchronization uses
exact method/path response matches, route assertions and real server-result UI;
there are no sleeps, `networkidle` correctness claims or retry-until-green loops.

Failure traces retain action diagnostics but disable snapshots, screenshots and
sources. Network recording is tied to snapshots in Playwright, so bearer headers,
Auth response bodies and refresh cookies are not captured. Supplemental HTTPS
requests use a Node HTTPS helper rather than a traced APIRequestContext or a
browser evaluate argument containing credentials. Storage checks return booleans,
never raw values. Do not log or attach captured headers, cookies or credentials.

## Unchanged boundaries

No production frontend/backend behavior, OpenAPI, migrations, sqlc, policy or M4
implementation changes belong here. ADR-023 publication stays inactive without
an approved receiver: this suite does not add or claim an active publisher.

`SECURITY_EXCEPTION-001` / `GHSA-vfj7-8cjw-p6xm` / `braces@3.0.3` remains an
unrelated **accepted temporary risk through 2026-10-31**, not remediated or extended.
This document is not the M3 completion report; only M3-VERIFY-001 may recommend
milestone closure after its prerequisites and queue transition merge.
