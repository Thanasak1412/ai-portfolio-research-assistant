# ADR-015 — Ed25519 Key Ring and Secret Management

**Status:** Accepted | **Date:** 2026-08-04

## Decision

Use TOKEN_SIGNING-v1: EdDSA/Ed25519 only, Railway service-variable secrets for staging/production, ignored owner-only development key files, and ephemeral test keys. Use PKCS#8 private and SPKI public DER base64, named key IDs, active signing key plus 24-hour public-key overlap, exact audience `ai-portfolio-research-assistant-api`, HTTPS public-origin issuer, and 60-second clock skew.

## Consequences

Startup fails closed for invalid/missing material. Private keys are never committed/logged. Rotation publishes verification first, then activates signing, removes old private material, retains old public verification material for overlap, and supports emergency immediate revocation.

## Testing and revision

Test claims, algorithms, invalid signatures, unknown/revoked keys, overlap, and startup validation. Secret-store or rotation changes require a new policy version and session-impact review.
