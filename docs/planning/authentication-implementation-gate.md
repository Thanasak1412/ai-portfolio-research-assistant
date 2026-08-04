# Authentication Implementation Gate

**Status:** Open
**Effective:** 2026-08-04
**Policy composition:** [AUTH_IMPLEMENTATION_POLICY-v1](../policies/AUTH_IMPLEMENTATION_POLICY-v1.md)

| Critical prerequisite | Status | Evidence |
|---|---|---|
| Approved execution plan is persisted | Approved | [AUTH-PLAN-v1](authentication-phase-1-execution-plan.md) |
| Argon2id parameters and rehash policy | Approved | PASSWORD_HASH-v1, benchmark report, ADR-014 |
| Ed25519 secret provider and key ring | Approved | TOKEN_SIGNING-v1, ADR-015 |
| Issuer, audience, active key ID, overlap, and skew | Approved | TOKEN_SIGNING-v1, ADR-015 |
| Production origins and local HTTPS workflow | Approved | AUTH_BROWSER_SECURITY-v1, ADR-016 |
| Refresh cookie, CORS, and CSRF policy | Approved | AUTH_BROWSER_SECURITY-v1, ADR-016 |
| Trusted proxy and source-IP derivation | Approved | CLIENT_NETWORK_IDENTITY-v1, ADR-017 |
| Shared rate-limit store and failure behavior | Approved | AUTH_RATE_LIMIT-v1, ADR-018 |
| Required ADRs | Approved | ADR-014 through ADR-018 |
| No conflict with AUTH-v1 | Approved | Closure specification review |
| No premature Authentication implementation | Approved | Repository inspection; documentation only |
| Decision package accepted on protected `main` | Blocked | Local commit `f9e4992` awaits governed PR publication, CI, and merge. |

**Authentication Implementation Gate: Blocked.** The security decisions are complete locally, but implementation may not begin until this package is accepted on protected `main` through ADR-013 governance. This gate does not authorize M2 work.
