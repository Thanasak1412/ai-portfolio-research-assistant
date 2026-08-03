# Worker and Internal Event Standard

The M0 worker contains lifecycle and dependency-health behavior only. It executes no business job and consumes no event.

Future jobs require a durable job ID, type/version, idempotency key, aggregate/scope identifier, attempt count, exponential backoff with jitter, maximum-attempt policy, dead-letter/manual-review state, correlation ID, and safe structured logs. Events are delivered at least once; consumers deduplicate by consumer name and event ID. Ordering is defined per aggregate stream, never globally.

API and worker binaries are released from the same modular-monolith version but may run and scale separately. Adding a broker or external event bus requires a new ADR.

