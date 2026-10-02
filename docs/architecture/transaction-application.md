# Transaction application and delivery foundation

Task: `M3-BE-002`. Base: `dfda4b4b677e540a5392e80a2d181ba6e0a27c3f`.

The Transaction application exposes create, correct, list, and get operations
using the existing authenticated principal and immutable domain values. HTTP
DTOs, cursor encoding, handlers, and route activation remain `M3-BE-003`.

## Authority and transaction boundary

Portfolio supplies an owner-scoped public reader bound to the caller's PostgreSQL
transaction. Its shared row lock prevents an archive from racing an accepted
financial write. Asset supplies its public lookup on that same connection,
avoiding pool exhaustion from nested connection acquisition. Both modules own
their queries; Transaction never reads their private tables directly.

Commands acquire the Portfolio ownership lock, then the existing idempotency
advisory lock, then the Portfolio sequence lock. Sequence allocation precedes
reading every affected Asset stream for the unchanged domain replay validator.
Create commits its immutable fact, idempotency result, success audit, and outbox
event together. Correction commits its relationship, reversal, replacement,
four success audit actions, idempotency result, and correction event together.
Every failed write rolls back the entire unit, including sequence allocation.

Expected rejected commands append only a safe failure audit after rollback;
infrastructure write failures do not emit independent success evidence. There
are no request bodies, amounts, notes, or arbitrary errors in audit/outbox data.

## Idempotency

The existing authority is `(portfolio_id, command_scope, idempotency_key)`.
Scopes remain `transaction.create.v1` and `transaction.correct.v1`. Under the
advisory lock, a matching SHA-256 fingerprint rehydrates the committed immutable
result. A mismatch returns a conflict. No sequence or event is created on replay.
Replay succeeds even after archival or subsequent changes to Asset eligibility,
because it represents an already accepted financial fact.

Fingerprint v1 serializes a fixed-order JSON object containing its version,
scope, correction target, kind, Asset ID, canonical decimals, USD, six-digit UTC
effective time, and optional text. Absent text is null and distinct from empty;
text is not trimmed or Unicode-normalized. Omitted trade fee becomes exact zero.
Correlation IDs, Asset metadata, and generated IDs/timestamps are excluded.
The existing persistence computes expiry as 8760 hours after completion. An
expired key is removed under its advisory lock before reuse.

History uses the frozen descending effective-time/sequence/ID tuple and typed
cursor positions, with one-row lookahead. Future read-filter timestamps are
valid. Incoming and outgoing correction relationships are kept independently,
so replacements can be corrected again.

## Delivery policy

The Platform delivery runner consumes the existing durable claim interface and
an injected publisher. The publisher may return success only after successful
delivery and must tolerate duplicate calls across lease recovery.

| Environment variable | Default |
| --- | --- |
| `OUTBOX_LEASE_DURATION` | `60s` |
| `OUTBOX_RETRY_BASE_DELAY` | `5s` |
| `OUTBOX_RETRY_MAX_DELAY` | `5m` |
| `OUTBOX_MAX_DELIVERY_INVOCATIONS` | `10` |
| `OUTBOX_BATCH_SIZE` | `50` |
| `OUTBOX_POLL_INTERVAL` | `2s` |

Invalid configured values fail configuration loading. The invocation override
cannot exceed 2147483646, reserving an int32 claim count for final recovery.
Backoff doubles with an
overflow-safe cap and inclusive full jitter. Polling waits between all claim
cycles, even when jitter is zero. Each claim can invoke the publisher once.
Failures on the final allowed attempt are dead-lettered; recovered claims above
the maximum never invoke the publisher and use `delivery_attempts_exhausted`.
Raw publisher errors are not persisted.

All durable transitions use the claim token. Expired batch items are left for
recovery. Cancellation stops new claims; successful in-flight publication can
acknowledge with a bounded context and its original token. Other interrupted
work recovers through lease expiry. A dead-letter predecessor continues to
block its aggregate. There is no automatic dead-letter replay.

`RunWithDelivery` supervises heartbeat and delivery with an explicit publisher.
**Runtime composition remains pending a receiver decision:** the repository has
no publisher destination or registered event consumers. `cmd/worker` still runs
its heartbeat and does not claim events. No logging/no-op publisher has been
installed. M3-BE-002 is not complete until this activation decision is resolved.

No M4 consumer, external broker, financial projection, provider integration,
frontend change, or public Transaction route is introduced.
