# Authentication Database Ownership

## Ownership and classification

| Table | Owner | Classification |
|---|---|---|
| `users` | Identity | Authoritative identity record |
| `refresh_sessions` | Identity | Authoritative session-generation record |
| `audit_logs` | Platform audit | Append-only security evidence |
| `auth_rate_limit_events` | Platform rate limiting | Expiring operational state, not business history |

Financial and Portfolio modules must not write these tables. Identity infrastructure accesses identity-generated sqlc code; platform audit and rate-limit implementations access platform-generated sqlc code. Domain and application packages must not import either generated package directly.

## Sensitive-data boundary

The database stores an encoded Argon2id password hash and a fixed-length opaque refresh-token digest because those are required security verifiers. It has no fields for raw passwords, raw refresh tokens, access tokens, cookie content, authorization headers, credential request bodies, or signing keys. Generated database structs have no JSON tags and must never be serialized as HTTP responses.

Network identity and rate-limit keys are HMAC-derived outside this database layer. `network_identity_hash` is bounded safe metadata; `derived_key` is a 32-byte HMAC result. Raw email and IP values are not rate-limit event fields.

## Refresh-session lifecycle

Each `refresh_sessions` row represents one generation in one token family. States are `active`, `replaced`, `revoked`, and `expired`. A partial unique index permits exactly one active generation per family. A deferred same-family replacement foreign key lets one transaction mark the active row replaced and then insert its replacement. Token rotation must select the presented digest row `FOR UPDATE`; application-controlled replacement inserts must retain the family's original absolute expiry.

Family revocation updates only the supplied family. Cleanup first marks active sessions expired and deletes inactive sessions only with a caller-supplied, separately approved retention cutoff. This schema does not define a retention period.

## Audit and rate-limit semantics

`audit_logs` accepts only the approved Authentication actions, results, severities, correlation identifiers, and bounded safe metadata. Its sqlc surface only appends; it exposes no modification or deletion query. Retention is intentionally deferred.

`auth_rate_limit_events` stores rolling-window events for AUTH_RATE_LIMIT-v1. The future runtime transaction must acquire the advisory transaction lock, delete expired key events, count active events, decide, insert when permitted, and read the earliest expiry. Global expiration cleanup belongs to the worker. The database queries provide these primitives but do not implement the runtime decision service.

## Index rationale

- `users_normalized_email_uidx` enforces canonical login uniqueness and serves normalized-email lookup.
- `refresh_sessions_token_digest_uidx` serves presented-digest lookup and prevents digest reuse.
- `refresh_sessions_one_active_family_uidx` enforces one active generation per family.
- Family/user creation indexes serve family-state and scoped session lookups.
- The active-expiry index serves expiry marking and cleanup discovery.
- The rate-limit window index serves policy/key rolling counts and earliest expiry; the expiry index serves worker cleanup.
