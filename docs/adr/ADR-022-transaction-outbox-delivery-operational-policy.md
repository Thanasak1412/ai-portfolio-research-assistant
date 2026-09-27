# ADR-022 — Transaction Outbox Delivery Operational Policy

**Status:** Accepted
**Date:** 2026-09-27

## Context

M3-BE-002 will activate delivery of the Platform transactional outbox. The
Planning Baseline, M3 execution plan, and Worker and Internal Event Standard
already require at-least-once delivery, lease-based claims, per-aggregate
predecessor ordering, consumer deduplication, exponential retry with jitter,
and a dead-letter/manual-review state. They do not set the operational values
needed to configure a production worker. M3-BE-002 must not choose those values
implicitly in implementation code.

## Decision

Adopt `OUTBOX_DELIVERY-v1` for the initial M3 worker. The settings below are
operational controls; they do not change the authoritative Transaction ledger.

### Claim lease

The claim lease duration is **60 seconds**. A crashed worker's `PROCESSING`
claim becomes eligible for another worker after its lease expires, subject to
the next claim cycle. A worker must use the durable claim token when
acknowledging, rescheduling, or dead-lettering a claimed event. A stale worker
must not acknowledge an event after another worker has reclaimed it.

### Retry and dead-letter

Retryable delivery failures use exponential backoff with full jitter:

```text
retryCap(attempt) = min(5 seconds × 2^(attempt - 1), 5 minutes)
retryDelay       = a uniform random duration in [0, retryCap(attempt)]
```

The base is **5 seconds**, multiplier **2**, and maximum cap **5 minutes**.
The maximum is **10 delivery invocations** of the publisher. Platform's
`ClaimDue` increments durable `attempt_count` on every claim, including an
expired-lease reclaim, so that count is not itself the number of publisher
invocations. For a claimed event with `attempt_count` from 1 through 10, the
worker may invoke the publisher once. Retryable failures after invocations 1
through 9 are rescheduled. A retryable failure after invocation 10 marks the
event `DEAD_LETTER` and is not rescheduled.

An expired final claim can be reclaimed after the former worker may have
invoked the publisher but failed to acknowledge. If such a recovery claim has
`attempt_count > 10`, the worker must not invoke the publisher. It immediately
marks its currently owned claim `DEAD_LETTER` with the bounded safe failure
code `delivery_attempts_exhausted`. This administrative recovery is not a
delivery invocation. Durable `attempt_count` may exceed 10 only in this final
lease-expiry recovery case. The uncertain external outcome is sent to manual
review; no automatic redelivery or dead-letter replay is authorized. Claim
cycles are paced by the poll interval below, including when a jitter result is
zero, so retries cannot form a tight loop.

A `DEAD_LETTER` event is not `PUBLISHED`. Platform's predecessor check therefore
blocks later events in the same aggregate stream. This is an operational
incident requiring investigation and an explicitly approved recovery action;
the worker must not silently skip, publish, or automatically replay the failed
predecessor. M3-BE-002 does not implement dead-letter recovery.

### Batch and polling

Each claim cycle requests at most **50** events, below Platform's hard
`MaximumClaimBatchSize` of 100. Claim cycles start no more often than every
**2 seconds**, whether or not the previous batch was full. A worker may stop
polling immediately on cancellation.

### Shutdown

Cancellation stops new claims. Work already in progress is acknowledged as
published only if delivery completed successfully and the worker still owns
the claim. Otherwise, lease expiry permits recovery by another worker.
At-least-once delivery means a publisher must tolerate duplicate delivery;
consumers use the durable `(consumer_name, event_id)` deduplication invariant.

## Configuration contract

| Setting | v1 default |
| --- | --- |
| `OUTBOX_LEASE_DURATION` | `60s` |
| `OUTBOX_RETRY_BASE_DELAY` | `5s` |
| `OUTBOX_RETRY_MAX_DELAY` | `5m` |
| `OUTBOX_MAX_DELIVERY_INVOCATIONS` | `10` |
| `OUTBOX_BATCH_SIZE` | `50` |
| `OUTBOX_POLL_INTERVAL` | `2s` |

Configuration validation must reject non-positive durations, a retry maximum
below its base, a maximum delivery-invocation count below one, and a batch size outside
`1..100`. Overrides must preserve the delivery algorithm and ordering
invariants. Changes to the algorithm, at-least-once guarantee, dead-letter
semantics, or aggregate ordering require architecture review. No M4 consumer
is authorized by this configuration contract.

## Consequences

- Temporary publisher failures retry with bounded database polling pressure.
- An expired claim can be recovered after its 60-second lease, subject to the
  next poll; duplicate delivery remains possible and must be safe.
- A tenth failed attempt becomes a visible operational failure and blocks
  subsequent events in that aggregate stream until separately resolved.
- Ledger writes remain authoritative regardless of delivery timing or state.
- No external broker, automatic dead-letter replay, M4 projection consumer,
  or M3-BE-003 HTTP work is introduced.

## Approval and implementation gate

This ADR is accepted as `OUTBOX_DELIVERY-v1`.

It was reviewed under ADR-013 and authorizes M3-BE-002 to implement the
approved outbox delivery policy.

Changes to the delivery algorithm, at-least-once guarantee, dead-letter
semantics, aggregate ordering, or the approved operational defaults require
architecture review.

M3-BE-002 may proceed. M3-BE-003 and M4 remain outside scope.
