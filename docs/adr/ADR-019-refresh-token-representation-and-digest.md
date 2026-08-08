# ADR-019 — Refresh Token Representation and Digest

**Status:** Accepted | **Date:** 2026-08-08

## Context

`AUTH-v1` requires opaque, single-use refresh tokens stored only as digests, but the original decision package did not fix their entropy, external representation, or digest construction. Those choices must be stable before an adapter can issue session verifiers safely.

## Decision

Adopt `REFRESH_TOKEN-v1`: generate 32 `crypto/rand` bytes; represent them as `rt_v1_` plus canonical unpadded RFC 4648 base64url; and persist SHA-256 of the exact canonical external token bytes, including the prefix. The digest is raw 32-byte data in `refresh_sessions.token_digest`. v1 uses no pepper or HMAC secret.

## Alternatives considered

- An unversioned token was rejected because it prevents safe future parsing and migration.
- Digesting only the payload was rejected because the explicit version prefix must be bound to the verifier input.
- A pepper or HMAC digest was rejected for v1 because no additional secret-management dependency is required for a 256-bit random bearer token and it would create an unnecessary coupling with secret rotation.

## Consequences

The database verifier cannot be reversed practically from the stored digest because the input space is 256-bit random. The implementation must never apply this password-inappropriate construction to human passwords. Future formats require a new prefix and v1 compatibility until the maximum 90-day family lifetime expires, or an explicit family revocation plan.

## Security and operational impact

Raw tokens remain prohibited from persistence, logs, audits, metrics, URLs, and JSON. Token-format changes are session-impacting and require rollout/rollback analysis. No schema migration is required because the v1 SHA-256 digest is 32 bytes and the existing column is constrained to 32 bytes.

## Testing and revision

Test canonical generation/parsing, malformed inputs, digest determinism, redaction, and v1 migration compatibility. A change requires a new refresh-token policy version, an updated composition, migration analysis, and this ADR to be superseded or revised.
