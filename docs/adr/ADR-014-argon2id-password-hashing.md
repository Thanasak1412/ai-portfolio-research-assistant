# ADR-014 — Argon2id Password Hashing Parameters

**Status:** Accepted | **Date:** 2026-08-04

## Context

AUTH-v1 requires Argon2id but left exact parameters to benchmark approval.

## Decision

Use PASSWORD_HASH-v1: Argon2id with 64 MiB, 3 iterations, 2 lanes, 16-byte salt, 32-byte key, PHC serialization, 12-character minimum, and 1,024-byte maximum input. Rehash after successful verification when policy metadata differs.

## Alternatives and consequences

32 MiB was rejected for lower memory resistance; 96 MiB reduced burst headroom. Benchmark evidence supports the selected set. Rebenchmark on production architecture/capacity or crypto-runtime changes. Password values and metadata remain secret; rehash-write failure does not invalidate a successful login.

## Testing and revision

Test malformed hashes, limits, rehash, and log/audit redaction. Revision requires a new policy version and compatibility review.
