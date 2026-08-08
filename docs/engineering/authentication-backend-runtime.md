# Authentication Backend Runtime

AUTH-BE-003 provides the transport-independent register, login, refresh, logout, current-user, and principal-resolution operations; PostgreSQL-backed composition; and Fiber transport. The Identity composition root validates the approved origin, environment-specific Ed25519 key-ring inputs, independent HMAC keys, and trusted proxy CIDRs. Staging and production reject an empty trusted-proxy set. Development loads an owner-only ignored key-ring file; test/staging/production use explicit environment key material according to TOKEN_SIGNING-v1.

The platform HTTP server still owns lifecycle, health/readiness, correlation IDs, safe request logging, and recovery. Identity transport mounts beneath `/api/v1/auth`, receives the validated platform correlation ID, derives the direct peer from the socket address, and passes raw `X-Forwarded-For` separately to the approved resolver. Request bodies, Authorization, Cookie, passwords, and tokens are not logged.

## Runtime activation status

`cmd/api` composes Identity and mounts `POST /api/v1/auth/register`, `POST /api/v1/auth/login`, `POST /api/v1/auth/refresh`, `POST /api/v1/auth/logout`, and `GET /api/v1/auth/me`. The platform server owns `/api/v1` and health routes; Identity supplies a generic route registrar, so platform code does not depend on Identity internals.

The concrete attestor accepts either a completed direct TLS connection or one exact raw `X-Forwarded-Proto: https` header from the actual plaintext socket peer when that peer belongs to `AUTH_TRUSTED_HTTPS_PROXY_CIDRS`. This setting is independent from `AUTH_TRUSTED_PROXY_CIDRS`; duplicate, comma-separated, padded, mixed-case, malformed, and untrusted assertions fail closed. Staging and production require non-empty non-universal HTTPS-proxy CIDRs. Existing health routes remain unchanged.

## Security invariants

- Refresh credentials appear only in the approved Secure, HttpOnly, SameSite=Lax, host-only cookie at `/api/v1/auth`.
- Browser refresh/logout require approved HTTPS attestation, exact Origin, and exact `X-Requested-With: portfolio-web`.
- Bearer middleware validates JWTs, reloads the user, and rejects disabled or missing users before publishing a private typed principal.
- Successful security mutations and their required audit records share one PostgreSQL transaction.
- Refresh replay commits affected-family revocation and high-severity evidence before returning a generic rejection.
- CORS middleware is not enabled.
