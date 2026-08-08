# ADR-020 — Authentication HMAC Key Representation and Derivation

**Status:** Accepted | **Date:** 2026-08-08

## Context

The approved network-identity and rate-limit policies require HMAC-SHA-256 but did not define secret representation, exact derivation domains, output encoding, or rotation consequences. Without those details, independently implemented adapters could produce incompatible identifiers or weaken key validation.

## Decision

Adopt `AUTH_HMAC_KEYS-v1`: use two independent canonical standard-Base64 secrets, each decoding to exactly 32 random bytes. Use the specified NUL-delimited HMAC-SHA-256 domains for network identity and rate limiting. Persist network identity as `ip_hmac_v1:` plus 64 lowercase hexadecimal characters, and store the raw 32-byte rate-limit HMAC result.

## Alternatives considered

- A shared HMAC secret was rejected because network correlation and rate limiting are separate security domains.
- Permissive Base64 parsing was rejected because multiple textual representations complicate validation and secret rotation.
- Process-local or unversioned derivation domains were rejected because they cannot preserve stable multi-instance semantics.
- Silent previous-key overlap was rejected for M1 because it hides rotation state and would require additional audit, compatibility, and operational controls.

## Consequences

Invalid secret configuration fails closed. Planned rotation requires a new policy/version and compatibility plan. Emergency replacement is allowed but breaks network-identity correlation across the cutover and resets active rate-limit-key continuity; it must be handled as an operational security event. No migration is required: both stored HMAC outputs are exactly 32 bytes and the 75-character network-identity representation fits existing 128-character bounds.

## Security and operational impact

Raw IPs, emails, refresh tokens, and secret values are prohibited from persistence and metric labels. The explicit derivation domains prevent semantic collisions among policies. Deployment procedures must provision both secrets independently and never log either encoding.

## Testing and revision

Test strict parsing, exact lengths, domain separation, canonical inputs, output formats, redaction, and planned/emergency rotation behavior. Any algorithm, key representation, or overlap change requires a new policy version, composition version, compatibility analysis, and ADR revision.
