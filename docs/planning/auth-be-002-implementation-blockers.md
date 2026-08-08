# AUTH-BE-002 Security Decision Gaps

**Status:** Blocking selected adapters only
**Reviewed:** 2026-08-08
**Scope:** AUTH-BE-002

The approved Authentication policy package was searched before selecting refresh-token and HMAC-secret parameters. Two implementation decisions are not present and therefore must not be invented inside adapter code.

## Refresh-token format and digest

`AUTH-v1` defines opaque, cryptographically random, single-use refresh tokens stored only as secure hashes, but the approved package does not define the exact entropy, external encoding, or digest algorithm/format. The refresh-token generator, parser, and digester remain blocked until a versioned policy approves all three values and their migration compatibility.

Required owner decision:

- token entropy in bytes or bits;
- canonical external encoding and accepted parser behavior;
- digest algorithm, optional secret/pepper policy, and stored representation;
- compatibility/versioning rules for previously issued tokens.

## HMAC secret input contract

`CLIENT_NETWORK_IDENTITY-v1` and `AUTH_RATE_LIMIT-v1` approve HMAC-SHA-256, distinct environment-variable names, namespaces, and persisted outputs. They do not define secret encoding, minimum entropy/length, or key-version/rotation handling. The network-identity HMAC adapter, rate-limit HMAC key-derivation adapter, and their secret configuration parsers remain blocked until those properties are approved.

Required owner decision:

- environment secret encoding;
- minimum decoded key strength and validation behavior;
- key version and rotation/overlap behavior;
- output digest encoding;
- compatibility impact on existing audit identities and active rate-limit windows.

## Unaffected work

The gaps do not block the approved Argon2id adapter, Ed25519 key ring and JWT adapter, trusted-proxy resolver, append-only audit writer, or PostgreSQL rate-limit transaction adapter when it receives an already-derived opaque rate-limit key. No Authentication endpoint or workflow is activated by this work.

No pre-M1 password population or approved legacy Argon2id parameter set exists. The password port exposes rehash evaluation, but verification accepts only `PASSWORD_HASH-v1` until a future policy version explicitly approves a legacy-compatible transition set.
