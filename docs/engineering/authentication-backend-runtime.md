# Authentication Backend Runtime

AUTH-BE-003 provides the transport-independent register, login, refresh, logout, current-user, and principal-resolution operations; PostgreSQL-backed composition; and Fiber transport. The Identity composition root validates the approved origin, environment-specific Ed25519 key-ring inputs, independent HMAC keys, and trusted proxy CIDRs. Staging and production reject an empty trusted-proxy set. Development loads an owner-only ignored key-ring file; test/staging/production use explicit environment key material according to TOKEN_SIGNING-v1.

The platform HTTP server still owns lifecycle, health/readiness, correlation IDs, safe request logging, and recovery. Identity transport mounts beneath `/api/v1/auth`, receives the validated platform correlation ID, derives the direct peer from the socket address, and passes raw `X-Forwarded-For` separately to the approved resolver. Request bodies, Authorization, Cookie, passwords, and tokens are not logged.

## Runtime activation status

Authentication route activation in `cmd/api` is intentionally blocked by [AUTH-BE-003 HTTPS Attestation Blocker](../planning/auth-be-003-https-attestation-blocker.md). The current approved reverse-proxy topology does not yet define how the private API proves original browser HTTPS after TLS termination. The transport requires an injected HTTPS attestor and cannot be constructed without one; it never falls back to trusting arbitrary `X-Forwarded-Proto` or Fiber protocol convenience behavior.

Consequently, the application and HTTP behavior are executable in deterministic and PostgreSQL integration tests using an explicit test attestor, but the production API composition does not mount Authentication routes yet. This is a blocking limitation, not an insecure development fallback. Existing health routes remain unchanged.

## Security invariants

- Refresh credentials appear only in the approved Secure, HttpOnly, SameSite=Lax, host-only cookie at `/api/v1/auth`.
- Browser refresh/logout require approved HTTPS attestation, exact Origin, and exact `X-Requested-With: portfolio-web`.
- Bearer middleware validates JWTs, reloads the user, and rejects disabled or missing users before publishing a private typed principal.
- Successful security mutations and their required audit records share one PostgreSQL transaction.
- Refresh replay commits affected-family revocation and high-severity evidence before returning a generic rejection.
- CORS middleware is not enabled.
