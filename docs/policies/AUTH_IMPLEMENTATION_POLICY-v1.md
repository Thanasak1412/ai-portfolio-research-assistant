# AUTH_IMPLEMENTATION_POLICY-v1

**Status:** Approved | **Effective:** 2026-08-04

## Composition

This is the implementation policy identifier for M1 Authentication Phase 1. An Authentication deployment, verification record, and completion report must name this version and all component versions below:

| Component | Required version |
|---|---|
| Core authentication requirements | [AUTH-v1](../planning/decision-closure-specification.md) |
| Password hashing | [PASSWORD_HASH-v1](PASSWORD_HASH-v1.md) |
| Token signing | [TOKEN_SIGNING-v1](TOKEN_SIGNING-v1.md) |
| Browser security | [AUTH_BROWSER_SECURITY-v1](AUTH_BROWSER_SECURITY-v1.md) |
| Network identity | [CLIENT_NETWORK_IDENTITY-v1](CLIENT_NETWORK_IDENTITY-v1.md) |
| Rate limiting | [AUTH_RATE_LIMIT-v1](AUTH_RATE_LIMIT-v1.md) |

The supporting decisions are ADR-014 through ADR-018. This composition refines implementation details without weakening AUTH-v1.

## Change control

Changing a component requires a new component version and a new composition version, compatibility analysis, session and cookie impact analysis, updated automated tests, and documentation. Signing-key or cookie changes additionally require rollout and rollback analysis. A policy version alone does not authorize a code change; normal pull-request and CI governance from ADR-013 continues to apply.
