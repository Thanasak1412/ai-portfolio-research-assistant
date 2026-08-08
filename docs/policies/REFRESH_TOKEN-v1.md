# REFRESH_TOKEN-v1

**Status:** Approved | **Effective:** 2026-08-08 | **Policy version:** v1

## Purpose

This policy closes the v1 refresh-token representation and digest decisions required by `AUTH-v1`. It applies only to opaque refresh tokens used by the Authentication session lifecycle; it does not alter cookie, rotation, reuse-detection, or expiry policy.

## Generation and external representation

A refresh token is generated from exactly 32 bytes read from `crypto/rand`, providing 256 bits of entropy. Its only external representation is:

```text
rt_v1_<payload>
```

`<payload>` is RFC 4648 base64url without padding for the 32 random bytes. It is exactly 43 characters; the complete v1 token is exactly 49 ASCII characters. Leading and trailing whitespace are not normalized.

The raw token is an opaque, short-lived in-process value. It must not be persisted, logged, included in audits, URLs, JSON responses, metrics, analytics, error strings, or browser-readable state.

## Parser requirements

The parser accepts only values that:

1. begin with the exact, case-sensitive `rt_v1_` prefix;
2. contain exactly 43 base64url payload characters with no padding;
3. decode through strict RFC 4648 base64url parsing to exactly 32 bytes; and
4. round-trip to the identical canonical unpadded payload.

Any other input—including whitespace, an unknown prefix, a padded encoding, a noncanonical encoding, or an incorrect decoded length—is malformed and receives only a generic Authentication failure. There is no unversioned legacy format before M1.

## Database verifier

The database verifier is the raw 32-byte SHA-256 digest of the exact canonical external token bytes, including `rt_v1_`:

```text
SHA-256("rt_v1_" || base64urlPayload)
```

The raw digest is stored in `refresh_sessions.token_digest`. v1 uses no pepper and no HMAC secret for this digest. SHA-256 is appropriate here because the presented value has 256 bits of cryptographically random entropy; passwords must never use this construction.

## Versioning and migration

Any future token representation must use an explicit new prefix, such as `rt_v2_`. A format or digest change must continue v1 lookup until all v1 families have expired within the approved 90-day absolute lifetime, or deliberately revoke the affected families as part of an approved migration. It must not silently reinterpret a v1 token under a new digest policy.

## Security and test requirements

Tests must cover exact entropy length, distinct generated values, canonical encoding, malformed parsing, deterministic v1 digesting, digest separation from raw-token representations, and secret-safe errors/logging. Rotation, replay detection, cookies, and session persistence remain responsibilities of their respective Authentication components.
