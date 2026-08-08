# AUTH_HMAC_KEYS-v1

**Status:** Approved | **Effective:** 2026-08-08 | **Policy version:** v1

## Secret representation

Authentication uses two cryptographically independent secrets:

- `AUTH_NETWORK_HMAC_KEY`
- `AUTH_RATE_LIMIT_HMAC_KEY`

Each variable is canonical RFC 4648 standard Base64 for exactly 32 cryptographically random decoded bytes. Parsing must use strict Base64 decoding and round-trip comparison to reject empty, malformed, wrong-length, or noncanonical values. Encoded and decoded secret values must never be logged, returned in errors, emitted as metrics, or exposed to frontend configuration.

The two secrets are separate security domains and must not intentionally share source material or a value.

## Network-identity derivation

For the canonical source IP defined by `CLIENT_NETWORK_IDENTITY-v1`, calculate:

```text
HMAC-SHA-256(
  AUTH_NETWORK_HMAC_KEY,
  "ip_hmac\x00v1\x00" || canonicalIP
)
```

The persisted and display-safe value is:

```text
ip_hmac_v1:<lowercase-hex-hmac>
```

where `<lowercase-hex-hmac>` contains all 32 HMAC bytes as 64 lowercase hexadecimal characters. Raw IP addresses remain prohibited from persistent data and metric labels.

## Rate-limit derivation

For the approved rate-limit policy and its canonical identity, calculate:

```text
HMAC-SHA-256(
  AUTH_RATE_LIMIT_HMAC_KEY,
  "auth_rate_limit\x00v1\x00" ||
  policyName || "\x00" ||
  policyVersion || "\x00" ||
  canonicalIdentity
)
```

The raw 32-byte result is stored in `auth_rate_limit_events.derived_key`. The derivation includes the policy name and policy version so the same identity cannot share a rate-limit namespace across policies or policy revisions.

Canonical identities are:

| Policy name | Canonical identity |
|---|---|
| `login_email_failure` | normalized email |
| `login_ip_attempt` | canonical source IP |
| `registration_ip_attempt` | canonical source IP |
| `refresh_family_attempt` | lowercase canonical UUID for the token-family ID |

Raw email, IP, refresh-token, cookie, and authorization-header values must never enter the rate-limit table, logs, or metric labels.

## Rotation and incident response

M1 does not support transparent active/previous HMAC-key overlap. A planned rotation requires a new approved key/policy version, compatibility analysis, and an explicit rollout plan before the secret changes.

Emergency secret replacement is permitted as incident response. Existing audit records remain immutable; post-cutover network identities are not expected to correlate with identities made under the previous key. Replacing `AUTH_RATE_LIMIT_HMAC_KEY` resets continuity of active rate-limit windows because newly derived keys differ, so it is an operational security event that may require temporary compensating controls. The implementation must never silently try old and new keys without a separately approved versioned policy.

## Security and test requirements

Tests must verify strict canonical secret parsing, exact decoded length, independent-key configuration, domain separation, deterministic output, canonical IPv4/IPv6 and UUID inputs, lowercase network output, raw 32-byte rate-limit output, redaction, and documented rotation effects.
