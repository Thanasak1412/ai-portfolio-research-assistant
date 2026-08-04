# Authentication Security Decision Closure Specification

**Status:** Approved
**Version:** AUTH-IMPLEMENTATION-POLICY-v1
**Effective date:** 2026-08-04
**Supersedes:** Open M1 implementation gates only; it does not change `AUTH-v1`.

## Authority and composition

This specification composes `AUTH-v1`, `PASSWORD_HASH-v1`, `TOKEN_SIGNING-v1`, `AUTH_BROWSER_SECURITY-v1`, `CLIENT_NETWORK_IDENTITY-v1`, and `AUTH_RATE_LIMIT-v1`. Authentication deployments and the M1 Completion Report must identify this complete policy set. Any component change requires a new version, compatibility/session-impact analysis, tests, and documentation.

The persisted execution plan is [Authentication Phase 1 — Execution Plan](authentication-phase-1-execution-plan.md). M0 is closed; no Authentication implementation existed at approval.

## Closed decisions

| Area | Approved decision | Reference |
|---|---|---|
| Password hashing | Argon2id, 64 MiB, 3 iterations, 2 lanes, 16-byte salt, 32-byte key. | ADR-014; PASSWORD_HASH-v1 |
| Token signing | Ed25519/EdDSA only; 15-minute JWT; 60-second clock skew; secret-manager key ring. | ADR-015; TOKEN_SIGNING-v1 |
| Browser topology | Same-origin HTTPS reverse proxy; host-only secure refresh cookie scoped to `/api/v1/auth`. | ADR-016; AUTH_BROWSER_SECURITY-v1 |
| Client network identity | Forwarding headers only from configured trusted proxy CIDRs; otherwise direct peer. | ADR-017; CLIENT_NETWORK_IDENTITY-v1 |
| Rate limits | PostgreSQL rolling-window event store with transaction advisory locking; fail closed. | ADR-018; AUTH_RATE_LIMIT-v1 |

## Implementation gate

| Critical item | Status | Evidence |
|---|---|---|
| Execution Plan persisted | Approved | `AUTH-PLAN-v1` in planning documentation. |
| Argon2id/rehash policy | Approved | `PASSWORD_HASH-v1`, ADR-014, recorded benchmark. |
| Key provider/key ring/issuer/audience/skew | Approved | `TOKEN_SIGNING-v1`, ADR-015. |
| Production/local HTTPS, cookie, CORS, CSRF policy | Approved | `AUTH_BROWSER_SECURITY-v1`, ADR-016. |
| Trusted proxy/source-IP policy | Approved | `CLIENT_NETWORK_IDENTITY-v1`, ADR-017. |
| Durable limits/failure behavior | Approved | `AUTH_RATE_LIMIT-v1`, ADR-018. |
| Required ADRs accepted | Approved | ADR-014 through ADR-018. |
| Contradiction with AUTH-v1 | Approved | None; all refinements preserve its security requirements. |
| Authentication implementation prematurely started | Approved | Repository contains documentation and M0 platform foundations only. |
| Decision package accepted on protected `main` | Blocked | Local commit `f9e4992` awaits governed PR publication, CI, and merge. |

**Authentication Implementation Gate: Blocked.** The decisions are closed locally, but M1 may begin only after this package is accepted on protected `main` through ADR-013 governance. Do not begin M2.
