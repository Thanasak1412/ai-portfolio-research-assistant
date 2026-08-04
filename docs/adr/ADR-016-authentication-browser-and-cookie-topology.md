# ADR-016 — Authentication Browser and Cookie Topology

**Status:** Accepted | **Date:** 2026-08-04

## Decision

Use AUTH_BROWSER_SECURITY-v1: one same-origin HTTPS reverse proxy for web and API; CORS disabled; host-only `pra_rt_v1` refresh cookie with Secure, HttpOnly, SameSite=Lax, and Path `/api/v1/auth`; strict Origin plus `X-Requested-With` checks on refresh/logout; and HTTPS local proxy at `app.localhost:3443`.

## Alternatives and consequences

Direct cross-origin localhost and `Secure=false` cookies were rejected because they cannot validate AUTH-v1 browser security. The proxy adds local/deployment operations but avoids credentialed wildcard CORS and preserves the narrow cookie path.

## Testing and revision

Test HTTPS cookie behavior, origin rejection, CSRF, clearing, and browser reload. Topology changes require compatibility and cookie-scope analysis.
