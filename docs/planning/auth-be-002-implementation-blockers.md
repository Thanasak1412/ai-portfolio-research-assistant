# AUTH-BE-002 Security Decision Gaps

**Status:** Resolved on merge of this decision closure
**Reviewed:** 2026-08-08
**Scope:** AUTH-BE-002

The original approved Authentication policy package lacked two representation decisions. This decision closure resolves them through `REFRESH_TOKEN-v1`, `AUTH_HMAC_KEYS-v1`, ADR-019, ADR-020, and `AUTH_IMPLEMENTATION_POLICY-v2`.

## Refresh-token format and digest

Resolved by [REFRESH_TOKEN-v1](../policies/REFRESH_TOKEN-v1.md) and ADR-019:

- 32 `crypto/rand` bytes, 256 bits of entropy;
- `rt_v1_` plus canonical unpadded base64url payload;
- SHA-256 over exact canonical external token bytes, including the prefix;
- raw 32-byte digest storage with no pepper/HMAC secret; and
- explicit future-version and 90-day compatibility rules.

## HMAC secret input contract

Resolved by [AUTH_HMAC_KEYS-v1](../policies/AUTH_HMAC_KEYS-v1.md) and ADR-020:

- two independent canonical standard-Base64 secrets that decode to exactly 32 bytes;
- explicit NUL-delimited HMAC-SHA-256 derivation domains;
- lowercase hexadecimal network identity and raw 32-byte rate-limit output;
- canonical identity requirements; and
- planned and emergency rotation rules without silent overlap.

## Schema representation verification

No migration is required solely for this closure:

| Representation | Required size | Existing schema capacity | Result |
|---|---:|---:|---|
| Refresh-token SHA-256 digest | 32 bytes | `refresh_sessions.token_digest` is constrained to 32 bytes | Fits |
| Rate-limit HMAC-SHA-256 | 32 bytes | `auth_rate_limit_events.derived_key` is constrained to 32 bytes | Fits |
| `ip_hmac_v1:` + lowercase hex digest | 75 characters | Network-identity fields are bounded to 128 characters | Fits |

## Resumption boundary

After this decision closure is merged, resume only the previously blocked AUTH-BE-002 components: refresh-token generation/parsing/digesting, network-identity HMAC, rate-limit key derivation, their secret configuration parsing, and focused tests. Do not begin AUTH-BE-003 or expose any Authentication endpoint through this closure.
