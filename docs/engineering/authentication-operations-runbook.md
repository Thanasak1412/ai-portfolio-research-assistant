# Authentication Operations Runbook

## Prerequisites

- Docker Desktop or a compatible Docker engine
- Node.js 24 and pnpm 10
- Go 1.26 for direct backend checks
- `mkcert` and a browser for local trusted HTTPS

## Local browser Authentication

Start the supported local stack with `make auth-dev-up`. It validates or creates the ignored `.local/auth-tls/` certificate, prepares ignored ephemeral Authentication runtime material in `.compose.auth.env`, migrates the persistent local database, and starts the HTTPS profile. Open `https://app.localhost:3443`.

Stop safely with `make auth-dev-down`. This stops containers but preserves the local PostgreSQL volume and the local certificate. To remove Compose application volumes, use the deliberately destructive command `CONFIRM_RESET=portfolio-auth-local make auth-dev-reset`; it does not remove the developer certificate or local CA.

## Disposable operational E2E

Run `make auth-e2e`. It creates an isolated self-signed test certificate and ephemeral Authentication secrets, migrates `postgres-test`, starts the HTTPS proxy stack, runs proxy security checks, then runs the real Playwright Authentication suite. It never uses the persistent local portfolio database. Shut down and remove the disposable containers and volumes with:

```sh
docker compose --env-file .compose.auth.e2e.env --profile auth-https down -v
```

## Verification and troubleshooting

- Certificate not trusted: run `mkcert -install`, confirm `.local/auth-tls/` exists, and use exactly `https://app.localhost:3443`.
- Proxy returns 502: inspect `docker compose --env-file .compose.auth.env --profile auth-https ps`, then safe logs with `docker compose --env-file .compose.auth.env logs --tail=200 auth-proxy web api`.
- Refresh/logout returns 403: confirm the browser uses the HTTPS origin, `AUTH_PUBLIC_ORIGIN` is exact, `AUTH_TRUSTED_HTTPS_PROXY_CIDRS` is `172.30.20.2/32`, the proxy is running, and the request includes Origin plus `X-Requested-With: portfolio-web`.
- Login succeeds but reload does not: inspect cookie metadata in browser developer tools without copying its value; verify the cookie path, HTTPS proxy, refresh request, `/me`, and server session logs.
- Direct API works but browser Authentication does not: this is expected when using `http://localhost:8080`; direct HTTP is intentionally not an approved cookie-authentication topology.

For failed CI-like runs, `sh scripts/collect-auth-ops-diagnostics.sh .compose.auth.e2e.env` prints service status and bounded safe container logs. It never prints secret environment files, cookies, Authorization headers, or request bodies.

## Production boundary

This repository does not provision production. Operators must provide external TLS ingress, private API routing, exact `AUTH_PUBLIC_ORIGIN`, narrow `AUTH_TRUSTED_PROXY_CIDRS` and `AUTH_TRUSTED_HTTPS_PROXY_CIDRS`, the approved JWT key ring, independent network/rate-limit HMAC keys, and a migrated database. Before release, confirm TLS validity, same-origin route splitting, API privacy, proxy replacement of `X-Forwarded-Proto`, Secure host-only cookie behavior, no CORS, external secret storage, and health/readiness behavior. TLS certificate renewal, JWT rotation, HMAC rotation, and refresh-token lifecycle are separate procedures governed by their respective policies.
