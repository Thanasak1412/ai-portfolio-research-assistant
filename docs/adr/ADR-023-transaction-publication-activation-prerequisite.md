# ADR-023 — Transaction Outbox Publication Activation Prerequisite

**Status:** Accepted
**Date:** 2026-10-02

## Context

ADR-022 states that M3-BE-002 will activate Platform outbox delivery. The M3
execution plan requires worker delivery, but excludes M4 projection consumers.
The Worker and Internal Event Standard requires a new ADR for an external
broker or event bus. No concrete receiver is approved in the current repository.

At protected-main base `dfda4b4b677e540a5392e80a2d181ba6e0a27c3f`, Platform
provides outbox append/claim/state-transition and consumer-dedup interfaces;
these are persistence capabilities, not a publication receiver. `cmd/worker`
composes only dependency heartbeat behavior. The separate, paused M3-BE-002
implementation provides an injected publisher/delivery engine, not a concrete
receiver. Treating a no-op publisher as success would falsely acknowledge
pending events and release subsequent aggregate events without a real handoff.

## Decision

M3-BE-002 establishes the Transaction application, atomic
ledger/audit/outbox persistence, and the complete publication engine implementing
`OUTBOX_DELIVERY-v1`.

This ADR supersedes only the unconditional delivery-activation expectation in
ADR-022 and the corresponding M3-BE-002 execution-plan acceptance wording. It
does not supersede or alter ADR-022's delivery algorithm.

### Successful publication

An event may transition to `PUBLISHED` only after a real, approved
publisher/receiver accepts the handoff according to its reviewed acknowledgement
contract and the worker still owns the applicable durable claim.

Invocation alone, logging, in-memory discard, no-op success, and black-hole
publisher behavior are not successful publication. Publication means successful
responsibility handoff, not necessarily completion of downstream business
processing. Absence of a receiver is never successful publication.

### No configured receiver

Until an approved receiver is configured, Transaction outbox claim cycles must
not start. Events must not be claimed merely to discover that no receiver
exists. Receiver absence must consume zero claims/attempts and zero publisher
invocations: it must not increase `attempt_count`, schedule retries, dead-letter
events, or mark events `PUBLISHED`.

Existing heartbeat-only worker behavior may continue. Runtime must accurately
represent Transaction publication as inactive, not pretend delivery is active.

### M3 event state and ledger independence

New Transaction events continue to commit durably as `PENDING`, retaining
immutable event identity, aggregate identity, predecessor ordering, due-time
metadata, and payload references. Once publication is activated, eligibility
follows normal Platform claim and predecessor rules. This decision does not
reset existing `PROCESSING`, `PUBLISHED`, or `DEAD_LETTER` state.

Receiver availability is not required to accept a valid Transaction command.
A successful financial command still atomically persists ledger facts,
idempotency result, audit, and outbox event. Failure to persist a required
outbox row still rolls back the financial mutation. Publication availability
is independent from ledger authority.

### ADR-022 remains unchanged

Once an approved receiver is configured and runtime publication is activated,
`OUTBOX_DELIVERY-v1` applies unchanged:

```text
OUTBOX_LEASE_DURATION=60s
OUTBOX_RETRY_BASE_DELAY=5s
OUTBOX_RETRY_MAX_DELAY=5m
OUTBOX_MAX_DELIVERY_INVOCATIONS=10
OUTBOX_BATCH_SIZE=50
OUTBOX_POLL_INTERVAL=2s
```

Retain exponential full jitter, claim-token ownership, stale-claim protection,
expired-lease recovery, invocation-10 dead-letter behavior, `attempt_count > 10`
administrative recovery without publisher invocation, and the bounded safe
failure code `delivery_attempts_exhausted`. Aggregate predecessor blocking,
shutdown semantics, and the prohibition of automatic dead-letter replay remain
unchanged. Absence of a receiver before activation is not a publisher delivery
failure; actual failures after activation remain governed by ADR-022.

### Future activation

A separately reviewed receiver implementation must define the concrete
publisher/receiver, successful handoff semantics, duplicate-delivery tolerance,
runtime composition, and operational failure behavior not already governed by
ADR-022. M4 may eventually supply an internal consumer; separately approved
infrastructure work may supply another receiver. External broker/event-bus
adoption still requires its own ADR. This ADR introduces neither a receiver,
an M4 consumer, nor an external broker.

### Scope and completion effect

M3-BE-002 may be completed as Transaction application
orchestration, exactly-once financial mutation, atomic audit/outbox persistence,
a complete `OUTBOX_DELIVERY-v1` delivery engine, and runtime-safe inactive
publication when no approved receiver exists. It must not be described as
active end-to-end Transaction event delivery. Concrete activation remains
deferred until an approved receiver exists.

This changes neither later M3 task dependency ordering nor their scope.
Completion still requires implementation review and all mandatory verification;
acceptance of this ADR alone does not complete M3-BE-002 or authorize M4 work.

## Consequences and review

- Durable pending events can accumulate until receiver activation; they are
  not discarded or falsely acknowledged to hide the backlog.
- Ledger authority and atomic persistence remain independent of publication
  availability. No migration or financial policy change is introduced.
- Engine tests can establish ADR-022 behavior without claiming that a production
  receiver exists. Future activation must supply real handoff evidence.
- Accepted ADR-022 remains historically intact; this ADR narrows only its
  activation expectation under ADR-013 governance.

## Sources

- [ADR-022 — Transaction Outbox Delivery Operational Policy](ADR-022-transaction-outbox-delivery-operational-policy.md)
- [ADR-013 — Solo Maintainer Merge Governance](ADR-013-solo-maintainer-merge-governance.md)
- [Worker and Internal Event Standard](../architecture/worker-event-standard.md)
- [M3 execution plan](../planning/transaction-ledger-foundation-execution-plan.md), §5.2 and §10
