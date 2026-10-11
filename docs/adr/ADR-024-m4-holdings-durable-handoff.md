# ADR-024 — M4 Holdings Durable Handoff

**Status:** Proposed
**Date:** 2026-10-10
**Task:** M4-GATE-001
**Decision owner:** Maintainer / architecture owner

## Context

M3 commits immutable Transaction facts and versioned outbox events atomically.
Under accepted [ADR-023](ADR-023-transaction-publication-activation-prerequisite.md),
the worker has no configured receiver today: it performs zero claims, attempts,
publisher calls, retries, dead-letter transitions, or acknowledgements, and
events remain `PENDING`. M4 needs a concrete internal receiver before that
delivery engine can be activated for Holdings rebuild work.

The M4 plan proposes durable responsibility handoff to Holdings rather than
performing an unbounded projection inside an outbox lease. This ADR is a
proposal only. It does not approve or activate a receiver, alter ADR-022, or
authorize an external broker.

## Proposed decision

If accepted, M4 will implement the receiver identity
`holdings_rebuild_request_v1` inside the existing worker. It will accept only
`transaction.recorded.v1` and `transaction.corrected.v1`, each at event version
1 with the approved bounded reference-payload schema. It will validate event
references and Portfolio/aggregate consistency through Transaction-owned
public ports. Unknown versions, unsupported event types, malformed references,
or inconsistent identities fail closed.

For each event, the receiver will use one caller-owned PostgreSQL transaction:

1. Call Platform consumer deduplication for
   `(consumer_name = holdings_rebuild_request_v1, event_id)`.
2. If this is the first delivery, persist durable Holdings rebuild
   responsibility that references the event, Portfolio, required ledger
   checkpoint, and calculation version.
3. Commit the dedup record and responsibility together, or commit neither.

A duplicate is successful only when the already committed dedup record and
durable responsibility can both be proven. Deduplication without durable work
is an integrity failure, not success. Events may be coalesced for calculation,
but each accepted event retains its durable responsibility evidence.

The receiver returns success only after durable responsibility commits. The
outbox engine may then mark the event `PUBLISHED` using the still-owned claim
token. `PUBLISHED` means responsibility was durably accepted; it does not mean
the Holding projection completed. A crash after handoff commit but before the
outbox acknowledgement is safe: redelivery finds the committed pair and
returns success without duplicating work.

## Ownership and boundaries

- **Platform** owns generic durable job mechanics, leases, fencing, technical
  retry, and consumer deduplication.
- **Holdings** owns rebuild intent, Portfolio/checkpoint semantics, calculation
  version, FIFO behavior, and projection responsibility.
- **Transaction** owns the immutable ledger and the public snapshot/checkpoint
  port used to validate and read committed facts.
- The receiver is an in-process worker adapter. No external broker or event
  bus is introduced.
- Platform must not implement Asset eligibility, Transaction kinds, FIFO,
  corrections, or financial calculations. Holdings must not read Transaction
  tables directly.

## Failure, activation, and operational boundary

The existing ADR-022 delivery algorithm remains unchanged, including its
60-second lease, 5-second retry base, 5-minute retry maximum, 10 publisher
invocations, batch size 50, 2-second poll interval, claim-token fencing,
aggregate predecessor ordering, dead-letter behavior, and prohibition of
automatic dead-letter replay. Those outbox values do not define Holdings job
retry policy.

Before receiver approval and explicit runtime activation, ADR-023 remains
operative: no receiver means no outbox claims or attempt increments and all
events remain durably `PENDING`. A fake, no-op, in-memory, or discard receiver
cannot acknowledge an event. Ledger, idempotency, audit, and outbox persistence
remain atomic and independent of receiver availability.

Implementation and deployment must separately demonstrate backlog acceptance,
atomic handoff, duplicate delivery, crash-before-ack recovery, worker restart,
lease expiry/fencing, durable-work failure/recovery, monitoring, and safe
deactivation. Deactivation stops new claims and must not reset durable state.
This ADR does not choose Product-facing behavior for a permanently failed
projection or authorize destructive cleanup.

## Consequences if accepted

- A committed Transaction event cannot be acknowledged before Holdings owns
  durable responsibility for it.
- The outbox may be marked published while the projection remains pending,
  rebuilding, or failed; those are distinct states.
- Existing M3 events can accumulate as pending until the separately reviewed
  receiver implementation and deployment prerequisites are met.
- No financial behavior, public API, database schema, worker composition, or
  deployment behavior changes merely by proposing this ADR.

## Approval

This ADR remains **Proposed**. It becomes effective only after an explicit
maintainer decision and merge under ADR-013. Codex does not accept this ADR.
Until then, the worker remains inactive under ADR-023 and M4-GATE-001 remains
blocked on receiver approval and implementation readiness evidence.

## Sources

- [M4 Price Data and Holding Projection Execution Plan](../planning/price-data-holding-projection-execution-plan.md), §§4, 7–8, 11, 17
- [ADR-013 — Solo Maintainer Merge Governance](ADR-013-solo-maintainer-merge-governance.md)
- [ADR-022 — Transaction Outbox Delivery Operational Policy](ADR-022-transaction-outbox-delivery-operational-policy.md)
- [ADR-023 — Transaction Outbox Publication Activation Prerequisite](ADR-023-transaction-publication-activation-prerequisite.md)
- [Worker and Internal Event Standard](../architecture/worker-event-standard.md)
