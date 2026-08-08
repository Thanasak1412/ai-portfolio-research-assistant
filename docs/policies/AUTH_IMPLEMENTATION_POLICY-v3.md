# AUTH_IMPLEMENTATION_POLICY-v3

**Status:** Approved | **Effective:** 2026-08-08

## Composition

This is the current M1 Authentication Phase 1 implementation-policy composition. Authentication implementation, deployment verification, and completion records using trusted HTTPS scheme attestation must identify this version and every component below.

| Component | Required version |
|---|---|
| Core authentication requirements | [AUTH-v1](../planning/decision-closure-specification.md) |
| Password hashing | [PASSWORD_HASH-v1](PASSWORD_HASH-v1.md) |
| Token signing | [TOKEN_SIGNING-v1](TOKEN_SIGNING-v1.md) |
| Refresh-token representation | [REFRESH_TOKEN-v1](REFRESH_TOKEN-v1.md) |
| Authentication HMAC keys | [AUTH_HMAC_KEYS-v1](AUTH_HMAC_KEYS-v1.md) |
| Network identity | [CLIENT_NETWORK_IDENTITY-v1](CLIENT_NETWORK_IDENTITY-v1.md) |
| Rate limiting | [AUTH_RATE_LIMIT-v1](AUTH_RATE_LIMIT-v1.md) |
| Browser security | [AUTH_BROWSER_SECURITY-v1](AUTH_BROWSER_SECURITY-v1.md) |
| HTTPS scheme attestation | [HTTPS_ATTESTATION-v1](HTTPS_ATTESTATION-v1.md) |

`AUTH_IMPLEMENTATION_POLICY-v1` and `AUTH_IMPLEMENTATION_POLICY-v2` remain historical approved compositions. This v3 composition adds the trusted HTTPS scheme-attestation decision; it does not weaken `AUTH-v1`, cookie security, Origin validation, or trusted client-IP derivation.

The supporting decisions are ADR-014 through ADR-021. Normal pull-request, CI, and solo-maintainer governance from ADR-013 remain mandatory.

## Change control

Changing any component requires a new component and composition version, compatibility analysis, session/cookie impact analysis, automated tests, and documentation. Changes to HTTPS attestation additionally require ingress-proxy and private-network review. A policy version alone does not authorize a code or deployment change.
