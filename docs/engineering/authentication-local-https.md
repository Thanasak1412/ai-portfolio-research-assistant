# Authentication Local HTTPS

Browser Authentication uses the approved same-origin endpoint: `https://app.localhost:3443`. The `auth-proxy` Compose profile terminates TLS, routes `/api/v1/*` to the private Go API on port 8080, and routes all other paths to Next.js on port 3000. Direct `http://localhost:8080` remains a health and diagnostic port only; it cannot satisfy refresh/logout browser security.

## First-time setup

Install [`mkcert`](https://github.com/FiloSottile/mkcert), then trust its local CA once with `mkcert -install`. Run `make auth-dev-up`; it calls `scripts/prepare-local-auth-https.sh`, which creates an ignored leaf certificate and owner-readable key in `.local/auth-tls/`. The script never installs packages or alters the trust store itself. If `mkcert` is missing, it exits with installation guidance.

Certificate and key material are developer-only and ignored. Renew a local certificate by stopping Compose, deleting only `.local/auth-tls/app.localhost.pem` and `.local/auth-tls/app.localhost-key.pem`, then running `make auth-dev-up` again. Do not rotate the local CA automatically.

## Security topology

The proxy has the fixed address `172.30.20.2` on the dedicated internal `proxy-api` network. Compose supplies only `172.30.20.2/32` to `AUTH_TRUSTED_HTTPS_PROXY_CIDRS`; this list is independent of `AUTH_TRUSTED_PROXY_CIDRS`, which stays empty in local Compose. Caddy deletes any browser-supplied `X-Forwarded-Proto` and sends exactly one `X-Forwarded-Proto: https` to the API. The API and proxy are the only services on that path, so web and database containers cannot spoof HTTPS attestation.

The refresh cookie remains host-only, Secure, HttpOnly, SameSite=Lax, and scoped to `/api/v1/auth`. Do not set `Secure=false`, add a cookie Domain, use a direct API URL for browser Authentication, or add CORS.

## CI and E2E

CI creates an isolated one-day self-signed certificate in `.local/ci-auth-tls/` and configures only the Playwright test browser to accept it. Application runtime code does not ignore certificate failures. The real Authentication suite uses the disposable `postgres-test` database, performs no `/api/v1/auth/*` request interception, and verifies registration, reload refresh rotation, logout, protected routing, cookie attributes, and direct-proxy spoof rejection.

Use `make auth-e2e` for the same disposable local operational suite. It creates only ignored temporary material. Stop it with `docker compose --env-file .compose.auth.e2e.env --profile auth-https down -v` when finished.
