# ADR-024 Maintainer Architecture Decision Record

**Task:** M4-GATE-001

**Status:** A1–A7 accepted by maintainer; O1–O8 BLOCKED; implementation and activation not authorized

**Evidence base:** `2c653ce74ce737ab32ac40e11b13bd916962719f` (protected main; PR #83 merged)

**Decision owner:** Maintainer / architecture owner

**Operational decision owners:** Architecture maintainer / Platform / Holdings / operator

## Requested decision and authority

The maintainer explicitly accepted the architectural contract in [ADR-024](../adr/ADR-024-m4-holdings-durable-handoff.md), limited to clauses **A1–A7** below, on 2026-10-11. [Approval evidence](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/84#issuecomment-6107239122) records the exclusions. This document records architecture acceptance only. Neither merging this record nor accepting the architecture closes [M4-GATE-001](m4-implementation-readiness-gate.md), authorizes M4-CONTRACT-001, configures a receiver, or permits runtime implementation or deployment.

The [Phase A recommendation](m4-phase-a-architecture-review.md) supplies the earlier source review. This record turns that recommendation into individually reviewable decisions and identifies the remaining operating-design blockers. Phase A requires approved mechanisms, owners and budgets, not deployed tests. Phase B separately requires implemented and deployed evidence before activation. No Phase B test is claimed to exist here.

## Verified implementation evidence

| Source on the evidence base                                                                                                                                               | Verified capability / limit                                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| [Platform adapter](../../backend/internal/platform/database/platform_eventing.go), `NewPostgresConsumerDeduplicator` / `RecordIfNew`                                      | Accepts caller DBTX; opens no transaction. A pool also satisfies DBTX, so composition must explicitly bind the same PostgreSQL transaction as durable work.                                                                                                  |
| [Dedup SQL](../../backend/queries/platform/consumer_dedup.sql)                                                                                                            | Unique consumer/event insert with conflict detection. `false` proves an existing marker, not durable business responsibility.                                                                                                                                |
| [Outbox SQL](../../backend/queries/platform/outbox.sql)                                                                                                                   | Immutable aggregate-position ordering, SKIP LOCKED claims, predecessor blocking, PROCESSING + claim-token transition predicates. ACK does not additionally test wall-clock lease expiry.                                                                     |
| [Transaction transactor](../../backend/internal/transaction/infrastructure/database/transactor.go) / [service](../../backend/internal/transaction/application/service.go) | Ledger, idempotency, audit and reference-only events share the write transaction. Recorded events reference `transaction`; corrected events reference correction/original/reversal/replacement. No receiver availability check governs financial acceptance. |
| [Delivery engine](../../backend/internal/platform/worker/delivery.go)                                                                                                     | Calls Publisher before ACK, cooperatively bounds Publish to lease, and uses a bounded ACK context after successful in-flight delivery. A nil Publisher result cannot itself prove a durable handoff.                                                         |
| [Composition](../../backend/cmd/worker/main.go) / [supervisor](../../backend/internal/platform/worker/supervisor.go)                                                      | Empty delivery dependencies; absent Publisher chooses heartbeat before any claim. No implemented Holdings receiver or rebuild responsibility exists.                                                                                                         |

These findings are source evidence, not receiver integration or deployment evidence. The Transaction snapshot/checkpoint ports and Holdings work/generation persistence required below are future seams, not existing runtime capabilities.

## Accepted architecture clauses A1–A7

The clause text below is preserved from the reviewed proposal. "Accept" describes the approved architecture contract, not permission to implement or activate it. All seven clauses are accepted together within the maintainer's limited scope.

### A1 — Internal receiver identity and accepted envelope

Accept `holdings_rebuild_request_v1` as the proposed Holdings-owned, in-process durable receiver inside the existing worker, not an external broker and not synchronous FIFO calculation inside the outbox lease. Accept only `transaction.recorded.v1` and `transaction.corrected.v1`, event version 1 and reference payload schema 1. Validate required roles, authoritative correction relationships, and Portfolio/aggregate consistency through Transaction-owned public ports. Unsupported types/versions, missing or inconsistent references fail closed without successful handoff.

Each accepted event must bind its immutable identity, Portfolio, required committed ledger checkpoint and calculation version to durable responsibility. A complete correction checkpoint must not bisect original/reversal/replacement. Schema/query details belong to separately authorized downstream tasks; this ballot does not invent their implementation.

### A2 — Atomic deduplication and durable responsibility

One caller-owned PostgreSQL transaction inserts Platform's `(holdings_rebuild_request_v1, event_id)` marker and Holdings' durable event responsibility. Commit both or neither. Never insert a marker through a pool in an independently committed operation and then enqueue business work separately.

Coalescing may reduce calculation jobs, but must preserve a durable binding for every accepted event to work and its checkpoint/version. A higher Portfolio checkpoint or some unrelated job is not evidence that this event was accepted. Holdings must provide the responsibility lookup; Platform remains domain-neutral.

### A3 — Commit-before-ACK and duplicate integrity

Receiver success means durable responsibility committed, not a calculation completed. Only after success may the delivery engine attempt `PUBLISHED` under the still-owned outbox token. Crash before commit leaves neither marker nor responsibility; crash after commit before ACK is redelivered as a duplicate. An uncertain commit returns failure/uncertainty, never speculative success; redelivery resolves it by reading the durable pair.

After `RecordIfNew` returns false, a subsequent authoritative read must prove the matching committed responsibility and its identity/checkpoint/version binding. With PostgreSQL READ COMMITTED, a conflicting insert may wait for the competing transaction; a fresh read must not rely on the losing statement's earlier snapshot. A missing/inconsistent durable pair is an integrity incident: fail handoff, preserve evidence, alert the named owner and require an approved repair. No marker deletion, fabricated work, skip, ACK, or automatic dead-letter replay is an integrity repair.

Completed calculation is valid duplicate evidence only while its durable responsibility binding is retained. Retention or later cleanup cannot destroy duplicate proof; no deletion policy is authorized by this ballot.

### A4 — Two independent fences

Outbox claim-token fencing governs publication state transitions only. Once another worker reclaims an event, the stale token cannot ACK/reschedule/dead-letter it. Lease expiry makes a claim eligible for reclaim; existing SQL does not reject an unreclaimed token solely because time elapsed. Cooperative cancellation and the token predicate are not exactly-once delivery. Late handoffs must therefore remain dedup-safe.

Holdings work ownership and generation fencing independently govern calculation publication. A reclaimed/stale work token must not activate a candidate generation or complete another worker's request. Final generation swap and completion commit atomically; checkpoint cannot decrease or obsolete calculation versions displace approved ones. A coherent checkpoint C may become stale when the ledger advances to C'; it must not be falsely reported current. An outbox token is not a Holdings generation token. Exact downstream CAS/interface design remains subject to the approved M4 sequence and review.

### A5 — Ownership

| Owner                                                 | Responsibility                                                                                                                    | Forbidden substitution                                                                              |
| ----------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| Platform                                              | Generic durable jobs, DBTX mechanics, dedup, claims/leases/fences and technical scheduling                                        | No FIFO, correction attribution, Asset eligibility or Portfolio ownership rules.                    |
| Transaction                                           | Immutable facts/correction graph and complete committed snapshot/checkpoint through public ports                                  | No Holdings direct SQL/table reads; no mutation of ledger facts to repair projections.              |
| Holdings                                              | Event responsibility, rebuild intent, source/version identity, calculation, candidate/active generations and financial provenance | No outbox ACK for merely starting work; no unapproved financial policy.                             |
| Operator with named Platform/Holdings incident owners | Approved capacity, monitoring, restart/recovery, deactivation and controlled incident actions                                     | No destructive cleanup, receiver activation or dead-letter replay inferred from generic job access. |

These are module/role assignments, not named human incident ownership. Named accountable owners and escalation contacts remain BLOCKED below.

### A6 — Backlog, recovery and deactivation

Preserve accumulated M3 events with their IDs/positions and ADR-022 predecessor ordering. Admit backlog only within approved capacity; capacity failure is not successful handoff. Durable accepted work survives restarts; rebuild from Transaction snapshots, not memory. Reconciliation detects missing/stale projection work but does not publish missing events or erase deduplication evidence. Failed rebuild leaves the last complete generation visible with pending/stale/failed metadata, never partial output or a negative Holding invented to continue processing.

Deactivation stops new outbox claims. A genuinely committed in-flight handoff may ACK only under its owned token, preserving ADR-022 shutdown semantics. Do not reset PENDING/PROCESSING/PUBLISHED/DEAD_LETTER state, work or generations. Whether already accepted Holdings jobs drain or pause must be an explicit operator-approved procedure, not an assumed default. Reactivation, controlled repair and incident escalation require named authority and preserved evidence; automatic dead-letter replay remains prohibited.

### A7 — Phase A acceptance is not Phase B activation

[ADR-023](../adr/ADR-023-transaction-publication-activation-prerequisite.md) remains operative: without an approved, configured receiver, zero claims, attempts, publisher calls, retries, dead letters and ACKs. Ledger/idempotency/audit/outbox writes remain atomic and independent of receiver availability.

[ADR-022](../adr/ADR-022-transaction-outbox-delivery-operational-policy.md) stays unchanged: 60-second lease, 5-second retry base, full jitter capped at 5 minutes, maximum 10 delivery invocations, batch 50/poll 2 seconds; attempt count >10 is administrative dead-letter recovery without a publisher invocation. Dead-letter predecessor blocks its stream. These values are not defaults for Holdings jobs, receiver database deadlines, provider ingestion or reconciliation.

Phase B activation requires separately reviewed implementation, real PostgreSQL proof of atomic first/duplicate handoff and missing-work integrity rejection, rollback/uncertain commit, concurrency, crash-before/after-commit-and-ACK, lease reclaim/stale ACK, worker restart, fenced generation swap, backlog and failed-work recovery, monitoring and safe deactivation. Retain the absent-receiver zero-claim regression. Operator deployment approval, remaining financial/provider/security prerequisites and current-head verification are required; successful proposal CI is not activation evidence.

## Unresolved architecture and operating decisions

Every row remains **BLOCKED**: no exact value or operating-approver evidence is present. Acceptance of A1–A7 settles the architecture direction only; it does not approve these operating decisions or close Phase A. No guessed defaults apply if unanswered.

| ID  | Decision required                                                     | Evidence needed for Phase A                                                                                                                                                    | Owner                                                    |
| --- | --------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------- |
| O1  | Accountable receiver/backlog/recovery/deactivation incident ownership | Named responsible owner, escalation/coverage and authority for controlled repair/reactivation                                                                                  | Architecture maintainer / Platform / Holdings / operator |
| O2  | Receiver transaction execution envelope                               | Approved DB deadline, connection/concurrency budget, cancellation/uncertain-commit procedure that fits handoff rather than unbounded calculation                               | Platform / Holdings / operator                           |
| O3  | Backlog capacity and admission                                        | Event/work storage envelope, maximum concurrent handoffs, throughput/resource budget, lag/age thresholds, alert and saturation procedure; no event discard                     | Platform / Holdings / operator                           |
| O4  | Holdings job lifecycle and retry/failure policy                       | Lease, renewal if used, execution deadline, batch/poll/concurrency, retry schedule and limit, terminal/manual-review behavior and safe reason codes                            | Architecture maintainer / Platform / Holdings / operator |
| O5  | Snapshot/calculation resource bounds                                  | Full replay time/memory/history streaming budgets and exhaustion handling without truncated financial history                                                                  | Holdings / Transaction / operator                        |
| O6  | Reconciliation and freshness monitoring                               | Cadence, scan/batch budget, checkpoint/version comparison, lag/failure/integrity thresholds and alerts                                                                         | Holdings / Platform / operator                           |
| O7  | Deactivation, draining and restart operations                         | Approved drain-versus-pause behavior for accepted work, shutdown deadline and restart/reactivation preflight; no state reset                                                   | Platform / Holdings / operator                           |
| O8  | Durable duplicate-proof lifetime and recovery                         | Responsibility-to-work/completion binding and retention mechanism sufficient for every supported redelivery/recovery; integrity inspection and human-approved repair procedure | Architecture maintainer / Holdings / operator            |

O4 must distinguish accepted-work failure from outbox delivery failure. O8 must not contradict the plan's prohibition on automatic dedup/request deletion. Numerical budgets need capacity/risk rationale; named owners need actual human evidence. Provider quotas are a separate account-specific register, never inferred here.

## Maintainer approval evidence — architecture only

| Field                                     | Current record                                                                                                                                                                           |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Architecture decision on A1–A7            | ACCEPTED — architecture design only                                                                                                                                                      |
| Approver name / role                      | GitHub `Thanasak1412`, repository maintainer / architecture owner; comment author association OWNER                                                                                      |
| Decision date                             | 2026-10-11, 08:42:42 UTC                                                                                                                                                                 |
| Written approval reference                | [Maintainer decision on PR #84](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/84#issuecomment-6107239122)                                                         |
| Conditions/exclusions explicitly approved | O1–O8 BLOCKED; ADR-025/financial policy, provider/account/credentials and other Phase A prerequisites excluded; Phase B/deployment/activation and all downstream implementation excluded |
| Operating decisions O1–O8                 | BLOCKED; owner-approved records missing                                                                                                                                                  |
| Runtime implementation authorization      | NOT AUTHORIZED by this record                                                                                                                                                            |
| Receiver activation decision/evidence     | NOT AUTHORIZED; separate Phase B review                                                                                                                                                  |

The approval text was supplied by the maintainer in the current conversation and posted to PR #84 by Codex at their explicit direction. GitHub author/date/OWNER attribution and the exact scope were verified. This is not approval inferred from proposal merge, AI self-review or green CI. The acceptance documentation remains subject to a separate ADR-013 review and protected-main merge. A1–A7 acceptance leaves O1–O8 and the overall gate BLOCKED.

## Preserved boundaries and review outcome

[ADR-025](../adr/ADR-025-m4-financial-compatibility-and-reconciliation.md) remains **Proposed**. [FIFO correction and decimal vectors](m4-financial-policy-proposal-vectors.md), including unresolved consumed-BUY oracles and DR7, remain unchanged. The [provider/operations register](m4-provider-operations-evidence-register.md) is unchanged; no account entitlement, secret mechanism or permission is inferred. SECURITY_EXCEPTION-001 remains the unchanged accepted temporary risk through 2026-10-31, not remediated or extended.

**Overall M4-GATE-001: BLOCKED. M4-CONTRACT-001: BLOCKED.** This document records the human-approved architecture acceptance only, pending review/merge of this documentation update. No financial-policy acceptance, runtime, API, database, worker composition, implementation authorization or queue activation is delivered.
