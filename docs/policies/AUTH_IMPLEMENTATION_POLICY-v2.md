# AUTH_IMPLEMENTATION_POLICY-v2

**Status:** Approved | **Effective:** 2026-08-08

## Composition

This is the next M1 Authentication Phase 1 implementation-policy composition. Authentication implementation, deployment verification, and completion records using the resolved refresh-token and HMAC decisions must identify this version and every component below.

| Component | Required version |
|---|---|
| Core authentication requirements | [AUTH-v1](../planning/decision-closure-specification.md) |
| Password hashing | [PASSWORD_HASH-v1](PASSWORD_HASH-v1.md) |
| Token signing | [TOKEN_SIGNING-v1](TOKEN_SIGNING-v1.md) |
| Browser security | [AUTH_BROWSER_SECURITY-v1](AUTH_BROWSER_SECURITY-v1.md) |
| Network identity | [CLIENT_NETWORK_IDENTITY-v1](CLIENT_NETWORK_IDENTITY-v1.md) |
| Rate limiting | [AUTH_RATE_LIMIT-v1](AUTH_RATE_LIMIT-v1.md) |
| Refresh-token representation | [REFRESH_TOKEN-v1](REFRESH_TOKEN-v1.md) |
| Authentication HMAC keys | [AUTH_HMAC_KEYS-v1](AUTH_HMAC_KEYS-v1.md) |

`AUTH_IMPLEMENTATION_POLICY-v1` remains a historical approved composition. This v2 composition adds the explicit refresh-token and HMAC-key decisions; it does not weaken `AUTH-v1` or replace prior audit records.

The supporting decisions are ADR-014 through ADR-020. Normal pull-request, CI, and solo-maintainer governance from ADR-013 remain mandatory.

## Change control

Changing a component requires a new component version and a new composition version, compatibility analysis, session/cookie impact analysis, updated tests, and documentation. Refresh-token representation changes must account for the 90-day absolute session lifetime. HMAC-key changes additionally require network-correlation and active-rate-limit-window impact analysis. A policy version alone does not authorize a code or deployment change.
