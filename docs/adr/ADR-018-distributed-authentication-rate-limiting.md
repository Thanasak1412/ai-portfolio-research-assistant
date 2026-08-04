# ADR-018 — Distributed Authentication Rate Limiting

**Status:** Accepted | **Date:** 2026-08-04

## Decision

Use AUTH_RATE_LIMIT-v1: PostgreSQL rolling-window event records, HMAC-derived keys, transaction advisory locking, worker cleanup, and fail-closed registration/login/refresh behavior when the store is unavailable.

## Alternatives and consequences

Process-local memory was rejected because it is inconsistent across API instances. Redis was rejected for M1 because it adds distributed infrastructure before demonstrated need. PostgreSQL adds bounded authentication writes and cleanup responsibility but provides atomicity with existing operational dependencies.

## Testing and revision

Test concurrency, expiry, limits, store failure, normalization, and family isolation. Store/algorithm changes require a new policy version and availability/security assessment.
