# M2 real-stack browser verification

`M2-E2E-001` verifies Portfolio and Asset behavior through the supported
browser topology:

```text
Chromium → https://app.localhost:3443 → Caddy → Next.js → Go API → postgres-test
```

The suite uses real registration, session, Portfolio, and Asset requests. It
does not mock any `/api/v1/auth`, Portfolio, or Asset endpoint.

## Test data isolation

`apps/web/tests/m2-e2e/fixtures/assets.sql` is synthetic, test-only catalog
data. `scripts/seed-m2-e2e-assets.sh` accepts only an E2E environment file
that targets the `postgres-test` Compose service and the `portfolio_test`
database. It rejects any other database target.

Never run the seed script against the persistent `postgres` development
service. The `postgres-test` service is tmpfs-backed and is removed with only
the disposable E2E Compose project.

## Local execution

The regular browser Authentication entrypoint remains
`https://app.localhost:3443`. `localhost:3000` and `localhost:8080` are not
M2 browser test targets.

If the normal local stack owns the fixed ports, stop its containers without
removing their volumes before starting the disposable stack:

```bash
docker compose --env-file .compose.auth.env --profile auth-https stop
```

Then run the isolated M2 suite with the `m2-e2e-001` Compose project name:

```bash
sh scripts/prepare-local-auth-https.sh .local/m2-e2e-tls
AUTH_TLS_DIR=.local/m2-e2e-tls \
  sh scripts/prepare-auth-e2e-env.sh .compose.m2-e2e.env

COMPOSE_PROJECT_NAME=m2-e2e-001 \
  docker compose --env-file .compose.m2-e2e.env \
  up --build -d --wait --wait-timeout 120 postgres postgres-test

COMPOSE_PROJECT_NAME=m2-e2e-001 \
  docker compose --env-file .compose.m2-e2e.env --profile tools \
  run --rm migrate-test

COMPOSE_PROJECT_NAME=m2-e2e-001 \
  sh scripts/seed-m2-e2e-assets.sh .compose.m2-e2e.env

COMPOSE_PROJECT_NAME=m2-e2e-001 \
  docker compose --env-file .compose.m2-e2e.env --profile auth-https \
  up --build -d --wait --wait-timeout 120 api worker web auth-proxy

sh scripts/verify-auth-https-stack.sh .compose.m2-e2e.env
PLAYWRIGHT_AUTH_E2E_IGNORE_HTTPS_ERRORS=true pnpm test:e2e:auth
COMPOSE_PROJECT_NAME=m2-e2e-001 \
  sh scripts/reset-e2e-auth-rate-limits.sh .compose.m2-e2e.env
PLAYWRIGHT_AUTH_E2E_IGNORE_HTTPS_ERRORS=true pnpm test:e2e:m2
```

The Authentication and M2 suites both use real registration. The approved
registration rate limit is intentionally low, so the reset script clears only
operational rate-limit events between independent suites in `postgres-test`.
It never runs against the persistent development database.

Afterward, remove only the disposable M2 test project and its tmpfs-backed
test database:

```bash
COMPOSE_PROJECT_NAME=m2-e2e-001 \
  docker compose --env-file .compose.m2-e2e.env --profile auth-https down -v
```

Do not use that cleanup command against `portfolio-research-bootstrap`, which
is the normal development project.
