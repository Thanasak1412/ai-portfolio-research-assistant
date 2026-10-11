# M4 Holdings Operating Decision Proposal — O1–O8

**Task:** M4-GATE-001

**Status:** PROPOSED; O1–O8 approval BLOCKED

**Evidence base:** protected main `b25f6043db9435e434aeb6185de69a7016d5bead`, merged [PR #85](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/85)

**Review date:** 2026-10-11

**Decision owners:** architecture maintainer / Platform / Holdings / Transaction / operator, as specified below

## Authority and two-phase boundary

This is a reviewable candidate operating contract, **not approved configuration**.
Every numerical candidate is test-only until individually approved with an
attributable date/reference. No production benchmark, capacity, named operator,
on-call coverage, receiver, or deployed control is claimed here.

[ADR-024](../adr/ADR-024-m4-holdings-durable-handoff.md) A1–A7 is Accepted for
architecture only. Its [decision record](m4-adr-024-maintainer-decision-record.md)
defines O1–O8 and their existing owners. [ADR-022](../adr/ADR-022-transaction-outbox-delivery-operational-policy.md)
and [ADR-023](../adr/ADR-023-transaction-publication-activation-prerequisite.md)
remain unchanged. The [M4 plan](../planning/price-data-holding-projection-execution-plan.md),
especially §§4, 7–8, 11 and 17, controls implementation sequencing.

Phase A approves designs, mechanisms, budgets, responsible humans and benchmark
acceptance criteria that can be specified before implementation. Missing actual
M4 runtime tests are **Phase B activation blockers**, not circular prerequisites
for authorizing their implementation. Phase A nevertheless needs a credible
resource envelope: where current capacity measurements are unavailable, record
the gap and obtain an explicit bounded pilot/benchmark decision, not a production
guarantee. No benchmark code or deployment is authorized by this proposal.

Overall **M4-GATE-001 = in_progress / BLOCKED**; **M4-CONTRACT-001 = blocked**.
ADR-025 stays Proposed; consumed-BUY cost oracles, DECIMAL reconciliation/DR7,
provider/account/calendar/mapping, credentials, retention/deletion, admission,
revocation and no-store prerequisites are unchanged. Receiver activation is
NOT AUTHORIZED. SECURITY_EXCEPTION-001 remains accepted temporary risk through
2026-10-31, not remediated or extended. Green proposal CI is not operating approval.

## Verified current capabilities and limits

These are code/configuration facts at the evidence base, not production observations.

| Source                                                                                                                                                                                                                                                                                                                                                                                                                        | Verified fact and limit                                                                                                                                                                                                                                                                                                                                                                |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [Config](../../backend/internal/platform/config/config.go), [pool](../../backend/internal/platform/database/pool.go), [Compose](../../compose.yaml)                                                                                                                                                                                                                                                                           | `DB_MAX_CONNS=10`, `DB_MIN_CONNS=1` per process in Compose and defaults; validated max range 1–100, min 0–100 with min≤max. API and worker have separate pools. `Open` explicitly applies only min/max to pgxpool after parsing the DSN. `DB_CONNECT_TIMEOUT=5s` is parsed but is not applied here as a pool-acquisition/transaction deadline. DSN transport settings remain separate. |
| [Worker root](../../backend/cmd/worker/main.go), [supervisor](../../backend/internal/platform/worker/supervisor.go), [heartbeat](../../backend/internal/platform/worker/runtime.go)                                                                                                                                                                                                                                           | Empty delivery dependencies; no outbox store/publisher constructed; absent receiver performs zero claims/attempts. Heartbeat default 30s. SIGINT/SIGTERM cancel context; pool close is deferred. `SHUTDOWN_TIMEOUT=10s` is parsed but not used to bound this worker shutdown. Compose gives 15s grace, not a measured shutdown guarantee.                                              |
| [Delivery engine](../../backend/internal/platform/worker/delivery.go)                                                                                                                                                                                                                                                                                                                                                         | Lease 60s, batch 50, poll 2s, sequential publishers. Publisher context ends at claim expiry; expired batch items are skipped, but already claimed attempts remain consumed. After success ACK uses an independent context of up to 60s, even during cancellation. Persistence errors terminate the runner, not a guaranteed self-restarting supervisor.                                |
| [Outbox SQL](../../backend/queries/platform/outbox.sql), [adapter](../../backend/internal/platform/database/platform_eventing.go)                                                                                                                                                                                                                                                                                             | SKIP LOCKED claims, predecessor blocking on immutable aggregate positions, expired PROCESSING recovery; attempts increment on every claim. State transitions require PROCESSING/current token, not an additional wall-clock-expiry predicate. Store opens no independent transaction.                                                                                                  |
| [Outbox types](../../backend/internal/platform/outbox/outbox.go), [migration 00004](../../backend/migrations/00004_m3_platform_eventing.sql)                                                                                                                                                                                                                                                                                  | Generic bounded references (≤8); DB JSON payload ≤2048 bytes. Payload size is not total event/work/index/WAL size. Consumer/event uniqueness exists; no Holdings durable responsibility/job/generation schema exists.                                                                                                                                                                  |
| [Dedup SQL](../../backend/queries/platform/consumer_dedup.sql) / Platform adapter                                                                                                                                                                                                                                                                                                                                             | Caller DBTX supports same-transaction composition; a pool also satisfies DBTX. `RecordIfNew=false` proves only an existing marker; Holdings must subsequently prove committed responsibility.                                                                                                                                                                                          |
| [Transaction transactor](../../backend/internal/transaction/infrastructure/database/transactor.go)                                                                                                                                                                                                                                                                                                                            | Ledger/idempotency/audit/outbox share READ COMMITTED write transaction. Rollback cleanup has an independent 5s context. No receiver availability check governs financial acceptance. Future snapshot/checkpoint ports are not implemented.                                                                                                                                             |
| [Integration delivery tests](../../backend/internal/transaction/infrastructure/database/delivery_integration_test.go), [inactive integration](../../backend/internal/transaction/infrastructure/database/inactive_publication_integration_test.go), [Platform tests](../../backend/internal/platform/database/platform_eventing_integration_test.go), [engine tests](../../backend/internal/platform/worker/delivery_test.go) | Repository tests cover attempts/final-lease recovery, predecessor/stale-token protection, atomic generic dedup and inactive zero-claim preservation. They do not prove Holdings receiver durability, workload capacity or production shutdown.                                                                                                                                         |
| Compose deployment                                                                                                                                                                                                                                                                                                                                                                                                            | Worker networks are internal database/test-database only; no reviewed provider egress or CPU/memory limits, restart policy, named alert transport or production database connection/storage budget is established by this file. Local Compose capacity is not production capacity.                                                                                                     |

Existing outbox retry is full jitter `U[0,min(5s×2^(attempt−1),5m)]`, ≤10
publisher invocations. Claim >10 performs administrative DEAD_LETTER with
`delivery_attempts_exhausted` and no publisher call. Dead-letter blocks later
events in its aggregate. No automatic replay, skip or fabricated ACK is permitted.

## Candidate budgets and consistency checks

| Candidate (PROPOSED, test-only) | Value                                                                      | Constraint / evidence still needed                                                                                                                                           |
| ------------------------------- | -------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Handoff                         | 5s end-to-end, one in flight per worker                                    | Includes pool acquisition, validation, locks, dedup/work insert and commit; ≤remaining outbox lease. No FIFO/provider call.                                                  |
| Receiver locks/statements       | 250ms lock wait; 1s per statement                                          | Transaction-local bounds, further limited by remaining 5s. Query counts/timeout support require review and measurement.                                                      |
| Receiver rollback cleanup       | ≤1s separate bounded cleanup                                               | Cannot imply commit succeeded; abandon broken connection if cleanup uncertain. New receiver procedure, not a change to M3 transactor.                                        |
| Calculation claim               | 5min lease; 4min total attempt; one worker/per-Portfolio invocation        | Proposed no lease renewal in v1. Total attempt includes read, calculation and staged/final publication; fresh fence needed at completion.                                    |
| Attempt stages                  | Snapshot ≤30s; calculation ≤3min; staging/final commit ≤30s                | 30+180+30=240s; 300s lease leaves 60s recovery margin. Acquisition/lock waits count, not extra deadlines.                                                                    |
| Calculation memory              | ≤256MiB per in-flight calculation working set                              | Streaming/spooling design must stay bounded and preserve exact full history. Process RSS, Go runtime, buffers and staging are additional; production limit/headroom unknown. |
| Holdings scheduling             | Poll 10s, claim batch 1, ≤5 actual calculation invocations                 | Lease reclaim is not permission to exceed maximum actual invocations. Independent from outbox policy.                                                                        |
| Holdings retry                  | Full jitter `U[0,min(30s×2^(n−1),10min)]`                                  | Retry caps for failed invocations 1–4: 30/60/120/240s; 10min cap remains explicit, not reached before limit. No tight loop when jitter=0.                                    |
| Reconciliation                  | Every 5min; ≤50 Portfolios/cycle; ≤20s cycle DB budget, one scan in flight | Durable/fair scan cursor; full sweep time `ceil(P/50)×5min`, not 5min for every Portfolio.                                                                                   |
| Warning / critical              | Lag >15min / >60min; storage ≥70% / ≥85% of approved usable budget         | Exact strict/inclusive boundaries stated; integrity and terminal-work failures alert immediately regardless of lag. Alert delivery/coverage still missing.                   |

**Batch hazard, not resolved by 5s:** 50 sequential worst-case handoffs take
250s before ACK overhead, exceeding a 60s common lease. To accept unchanged
batch 50 for a deployment, measure complete oldest-item age and batch duration
under competing workers/pool contention, duplicates and slow commits; propose
batch-duration p99≤50s and no skipped expired items in the approved workload
matrix, leaving 10s lease margin. A stress run at each 5s limit must explicitly
show safe skips/reclaims and attempt consumption, not falsely count them as
delivery. If capacity cannot satisfy normal-load margin, activation is BLOCKED:
request separately reviewed operational tuning under ADR-022, never silently
change batch/lease/poll, parallelize the engine, or mask attempts in this PR.

Proposed worker connection accounting: handoff 1 + snapshot/staging 1 +
claim/ACK mechanics 1 + heartbeat/reconciliation allowance 1 = ≤4 concurrently
occupied slots; calculation releases its snapshot connection before computing.
This is a proposed admission budget, not enforced pool partitioning. Max 10
leaves nominal 6 slots, not reserved production capacity. Operator must account
for all worker/API replicas, migrations, admin and other jobs against actual
PostgreSQL limits; do not borrow API capacity or introduce a second pool silently.
Monitor pool wait; test starvation and overlapping ACKs. Production replicas,
CPU, RSS, IOPS, max_connections and disk/WAL/backup budget are MISSING.

## O1 — Accountable ownership

**Current:** module ownership exists; named accountable humans, deputies,
coverage, alert destinations and escalation acknowledgement are MISSING.

**Proposed contract:** Platform owns delivery/dedup/technical job mechanics;
Holdings owns responsibility, generation and calculation incidents; Transaction
owns immutable source/checkpoint integrity. Operator leads incident response;
architecture maintainer authorizes exceptional repairs/reactivation. One person
may hold multiple roles only when explicitly recorded; no 24×7 claim is inferred.

| Action                                 | Responsible role               | Required approval / escalation                                                                                           |
| -------------------------------------- | ------------------------------ | ------------------------------------------------------------------------------------------------------------------------ |
| Triage delivery/DB/pool/lease incident | Platform + operator            | Named incident lead; notify Holdings for projection impact                                                               |
| Triage failed/stale generation         | Holdings + operator            | Transaction consulted for checkpoint/source discrepancy; financial maintainer for policy ambiguity                       |
| Backlog drain within approved envelope | Operator + Platform/Holdings   | Named operator approval; recorded capacity preflight and monitoring                                                      |
| Controlled integrity repair            | Owning module + operator       | Architecture maintainer plus evidence owner; destructive action needs separate explicit human approval                   |
| Emergency deactivation                 | Named operator                 | Explicit predelegated authority required; notify architecture/affected owners; if absent, escalate, not invent authority |
| Reactivation / changed budgets         | Operator + module owners       | Explicit architecture/operator approval and new evidence; no automatic restart-to-activation                             |
| Dead-letter intervention               | Platform + Holdings + operator | Separate maintainer-approved incident action; no automatic replay/skip/ACK                                               |

**Rationale/alternative:** role-based matrix is reviewable before staffing;
unassigned service ownership or assumed maintainer on-call is unsafe. Obtain
named primary/deputy, contact/coverage windows, response expectations and
out-of-coverage handling. No name or response SLA is populated by Codex.

**Failure/recovery:** preserve state and incident ID, scope/affected streams,
safe reason and correlation; never log financial bodies/tokens/provider data.
**Phase B tests:** route synthetic integrity/lag/pool alerts to approved contacts;
exercise escalation, emergency pause, repair refusal without approval and signed
reactivation checklist. **Owner/approver:** existing architecture / Platform /
Holdings / operator roles. **Approval:** BLOCKED; evidence MISSING.

## O2 — Receiver transaction envelope

**Current:** generic same-DBTX dedup exists, no receiver transaction. Current
outbox is sequential; token fencing does not make delivery exactly once.

**Proposed contract:** one 5s operation per worker includes acquiring the
connection and all validation through Transaction ports. Inside one caller-owned
READ COMMITTED transaction, bind Platform dedup and Holdings responsibility to
the exact same transaction. Bound locks/statements as above; do not hold the
ledger sequence-write lock for calculation. No provider/FIFO/unbounded history
read here. Validation and duplicate proof must fit the envelope; new public
read ports and indexes are later scoped implementation, not existing features.

First event commits immutable event→Portfolio/checkpoint/version responsibility
and dedup or neither. Duplicate uses a fresh authoritative read after conflict
to prove the matching committed pair, not the losing statement snapshot. Missing
or inconsistent pair fails handoff and alerts as integrity incident. Cancellation,
timeout, DB loss or uncertain COMMIT returns failure/uncertainty, never success.
Bound cleanup to ≤1s separately. Redelivery resolves uncertain outcomes by pair
lookup; only proven commit returns success, followed by current-token ACK.
No nested retry loop inside Publish: actual failures use unchanged ADR-022.

**Rationale/alternative:** 5s caps DB occupancy below lease; one invocation
avoids connection bursts. Shorter deadline may fail under contention; longer
deadline worsens batch expiry. Neither is chosen without timing evidence.
**Capacity evidence:** pool acquisition/lock/statement/commit percentiles,
complete batch age, duplicate contention and rollback timing, real DB topology.
**Phase B tests:** concurrent first/duplicate delivery; rollback each seam;
lost COMMIT response; killed connection; cancellation; marker without work;
crash after pair commit/before ACK; reclaim/stale ACK; all slow-batch outcomes.
**Owner/approver:** Platform / Holdings / operator. **Approval:** BLOCKED;
5s/lock/connection candidates PROPOSED; approval/benchmarks MISSING.

## O3 — Backlog capacity and admission

**Current:** outbox/dedup durable storage and ordering exist; production arrival
rate, row/index/WAL/work sizes, storage headroom and drain rate are not measured.
No Holdings work records exist to measure. Ledger remains authoritative.

**Proposed sizing:** measure event arrival λ (sustained/peak and burst window),
outage D, initial backlog N0 and real indexed byte growth per event E, per-event
responsibility R, dedup U and calculation/generation G. `N=N0+ceil(λ×D)`.
Keep outbox storage even after handoff; coalescing does not remove R/U.
Size event evidence as `N×(E+R+U)` plus generation/history, measured WAL,
backup/replica growth and operator-approved safety reserve. Use measured drain μ;
if μ≤λ recovery cannot catch up. Otherwise ideal drain time `N/(μ−λ)` is only
a lower bound: stream head-of-line failures, DB pressure and retries add delay.

Illustration ONLY, not quota/measurement: λ=0.2 events/s, D=24h, N0=0 gives
17,280 events; hypothetical combined 2KiB/event with amplification 3 and safety
factor 2 gives 202.5MiB. At that λ, unpruned 30-day evidence alone is 518,400
records / 1,012.5MiB before amplification, generations/backups. The 2KiB DB
payload limit does not establish that combined size. Operator must replace all
hypothetical inputs with evidence and approve the sizing horizon.

**Threshold candidates:** disk warning ≥70%, critical ≥85% of separately
approved usable event/job budget (reserve excluded), plus O6 age alerts. Record
pending count/oldest age, accepted-work lag, μ, pool wait, WAL and free disk.
Prefer durable capacity planning to arbitrary event-count rejection. Stop/reduce
projection admission within approved controls when unsafe; retain all events
and keep independently valid M3 writes operational. Do not add a Holdings
backlog validation rule to financial commands. Required outbox-persistence
failure still rolls back writes; physical DB exhaustion can affect M3 and must
be escalated honestly, never claim unlimited write availability.

**Recovery:** capacity preflight, scope/oldest stream inventory, pilot drain,
observe latency/headroom, expand only with approval; integrity/dead-letter cases
require separate action. Never discard, skip, purge or ACK to lower metrics.
**Capacity evidence:** measured λ/E/R/U/G/μ, approved outage horizon, disk/WAL/
backup/connection capacity and concurrent M3 latency baseline. **Phase B tests:**
24h-equivalent backlog/burst, multi-stream drain with blocked predecessor,
near-threshold storage, receiver outage with valid M3 writes, restart without
loss, retained evidence growth. **Owner/approver:** Platform / Holdings /
operator. **Approval:** BLOCKED; threshold candidates PROPOSED; sizing MISSING.

## O4 — Independent Holdings job lifecycle

**Current:** no generic durable Holdings jobs, work leases or invocation journal.
Outbox PUBLISHED means accepted responsibility, not successful calculation.

**Proposed contract:** candidate work states PENDING, PROCESSING, COMPLETED,
FAILED_MANUAL_REVIEW; no change to outbox states. Claim batch 1 every 10s,
one admitted invocation per worker and one live claim per Portfolio, 5min
non-renewing lease, 4min total attempt measured from claim time (not reset when
calculation starts); independent durable work token/generation fence. Persist
invocation start before executing; maintain a separate invocation count from
claim count. Crash after recorded start counts conservatively as an invocation.
Reclaim must not permit two live owners to publish, even if canceled old code
continues running. Global per-Portfolio ownership is DB-enforced, not memory.
Fencing prevents stale publication, not physical stale computation: a partitioned
or uncooperative expired worker may still consume CPU/memory alongside the new
owner. Phase B must prove cooperative cancellation and bound this overlap; do
not claim a lease alone guarantees exactly one physical calculation everywhere.

At most 5 actual calculation invocations. Transient DB/resource failures retry
after invocations 1–4 with the stated full jitter; cancellation during operator
pause retains recoverability without immediately retrying. Unknown final
invocation outcome is manual review, not a sixth invocation. Recovery claims
may exist administratively but never calculate past the limit. Invalid source,
unsupported approved-version combination, unresolved financial semantics or
integrity mismatch goes to manual review without fabricating output; retain
last-good generation labeled stale/failed, or no-generation failed state.
Proposed bounded reasons: `snapshot_timeout`, `calculation_timeout`,
`memory_budget_exceeded`, `source_integrity_failure`, `policy_incompatible`,
`generation_fence_lost`, `calculation_attempts_exhausted`; no raw exception text.
Final stage fence failure cannot overwrite/completely acknowledge newer work.

**Rationale/alternative:** no renewal simplifies bounded v1 execution; renewal
or larger budgets require evidence/review, not silent indefinite execution.
Full replay can be costly; coalesce scheduling without losing per-event proof.
**Capacity evidence:** worst-case claim-to-commit duration, memory, multi-worker
fencing, crash-count accuracy and starvation/fairness. **Phase B tests:** crash
before/after invocation start, each retry cap/jitter endpoint, fifth failure,
expired final lease recovery with zero extra invocation, stale generation CAS,
two workers/same Portfolio, different Portfolios and pause/resume. **Owner/
approver:** architecture / Platform / Holdings / operator. **Approval:** BLOCKED;
lifecycle and values PROPOSED, approval/benchmark evidence MISSING.

## O5 — Snapshot and calculation budgets

**Current:** private Transaction replay is not the future worker snapshot port;
no Holdings calculator/benchmark or deployed memory limit exists.

**Proposed contract:** Transaction public port supplies a full committed prefix
and correction graph from one read-only REPEATABLE READ snapshot. Acquisition
and complete read ≤30s, then release connection before ≤3min exact calculation.
All pages share that snapshot; checkpoint cannot bisect a correction. No sequence
allocation/write lock, independently paged snapshots, partial history or silent
cardinality truncation. Candidate 256MiB working set includes buffered facts,
graph/journal/lots; streaming/spooling must preserve provenance and ordering.
No temporary storage mechanism or production volume is approved here.

Within remaining ≤30s stage output and atomically publish complete generation/
work completion under its fence. Staging cannot expose partial rows; large
staging must still fit total 4min. Quantity/cost results depend only on separately
approved policy, never resource budgets. On any timeout/memory/policy failure,
discard visibility of the candidate, preserve durable source and last-good
generation, accurately mark failed/stale and use O4 recovery. If no generation
exists, do not represent failure as empty holdings. No authoritative FIFO oracle
is invented for ADR-025 unresolved histories.

**Rationale/alternative:** full replay favors reproducibility; streaming plus
bounded working state may fail for large graphs. Incremental state or bigger
memory is not silently substituted. **Evidence needed:** production CPU/RSS/
IOPS envelope, p95/p99 timings/peak memory at 0, 1k, 10k, 100k and 1m ledger
facts, long correction chains, many Assets/lots, fractional vectors, concurrent
M3 writes and other workers. These are benchmark cardinalities, NOT accepted
ledger-size limits. Phase A approve the workload/resource acceptance matrix;
Phase B measure real implementation and block activation if it cannot fit.
**Phase B tests:** coherent snapshot during corrections, cancellation and lock
contention, memory/deadline exhaustion without truncation, deterministic full
rebuild, hidden partial stage, fenced final commit and stale C versus C′.
**Owner/approver:** Holdings / Transaction / operator. **Approval:** BLOCKED;
budgets PROPOSED, capacity evidence MISSING.

## O6 — Reconciliation and monitoring

**Current:** no reconciliation loop or Holdings checkpoints; current heartbeat
only detects database connectivity and is not freshness monitoring.

**Proposed contract:** 5min cycle, max 50 Portfolios and 20s DB time/cycle,
one scan in flight with a resumable fair cursor. Read Transaction checkpoint
through public port; compare active generation C, required accepted work and
approved calculation/policy versions. Matching checkpoint alone does not prove
every event's responsibility. Detect marker-without-responsibility, conflicting
binding and orphan responsibility as integrity incidents; do not repair by ACK,
marker deletion or fabricated work. Queue a normal fenced rebuild for legitimate
missing/stale generations without replaying dead-letter outbox events.

Projection lag is age of the oldest committed change not covered by the active
generation, not merely age of the last unchanged generation. Warning >15min,
critical >60min; integrity, terminal calculation failure and dead-letter
predecessor alert immediately. Report pending/no-generation versus failed/stale,
checkpoint/version mismatch, oldest event/work age, attempts, pool waits and
scan coverage. Safe IDs/codes only, no tokens, body or financial-value logging.

**Rationale/alternative:** periodic scans complement events, not replace them.
For P=200, four cycles take 20min: candidate 50/5min cannot guarantee detection
within a 15min warning target without additional metrics. Phase A needs intended
Portfolio population and monitoring-channel design; Phase B must verify worst
case coverage/detection. Raise separately reviewed capacity or accept explicit
latency; do not repeatedly scan the first 50 or claim fleet-wide 5min detection.
**Evidence needed:** population, scan cost, approved lag tolerance, alert delivery/
coverage and acknowledgement. **Phase B tests:** missed notification, advancing
checkpoint mid-scan, wrong version, absent pair, no-work/no-generation, many
Portfolios/cursor wrap, threshold boundaries and dead-letter non-replay.
**Owner/approver:** Holdings / Platform / operator. **Approval:** BLOCKED;
cadence/thresholds PROPOSED, account population/operating approval MISSING.

## O7 — Deactivation, pause and restart

**Current:** cancellation stops future claim cycles, but successful handoff ACK
may wait up to 60s. Compose grace is 15s; configured 10s shutdown timeout does
not govern current worker. No calculation can be promised to finish on shutdown.

**Proposed contract:** pause accepted-work execution rather than drain long
calculations. Stop new outbox/work claims first; cancel snapshot/calculation,
retain recoverable work and discard candidate visibility. A genuinely committed
handoff may ACK under current token while process remains alive. Operator may
wait at most 10s from shutdown signal before accepting forced termination under
15s Compose grace; 5s is nominal safety margin, not a guarantee. Do not infer
an implemented ACK cap from that procedure: current independent ACK can outlive
grace and be killed. Pair durability plus duplicate-safe reclaim, not a long
drain promise, is the recovery mechanism. Bounded future worker composition and
cleanup mechanics need separate implementation/test review; ADR-022 unchanged.

Startup is paused for new work until preflight/explicit approved activation:
verify schema/version, source/accepted responsibility, job/outbox states,
fences, remaining invocation budget, last-good generations, pool/disk/monitoring
and named incident authority. Resume/recompute eligible expired work under a
new fence; never reset attempts or PENDING/PROCESSING/PUBLISHED/DEAD_LETTER,
completed/failed evidence. Graceful pause does not authorize repair, dead-letter
replay or live receiver activation. No approved receiver still means zero
outbox claims under ADR-023.

**Rationale/alternative:** bounded pause fits short grace; drain-to-completion
cannot fit 4min computation/5min lease. Changing grace or adding renewal is a
separate deployment decision. **Evidence needed:** observed cancellation, DB
cleanup/ACK timing, orchestrator restart policy and approved pause/reactivation
owners. **Phase B tests:** SIGTERM during each seam, SIGKILL after pair commit,
ACK timeout/claim reclaim, computation interrupted, stale old owner after restart,
paused startup, retained failed/dead-letter state and denied unapproved activation.
**Owner/approver:** Platform / Holdings / operator. **Approval:** BLOCKED;
pause/10s procedure PROPOSED, ownership/production shutdown evidence MISSING.

## O8 — Duplicate-proof lifetime and recovery

**Current:** durable consumer/event uniqueness only. No responsibility lookup,
completion binding, evidence cleanup, backup restore proof or retention budget.

**Proposed contract:** retain immutable event→responsibility binding containing
event/consumer identity, Portfolio, required complete checkpoint, calculation
version and accepted-work identity, even after coalescing/completion. Preserve
stable linkage to completion/generation; coalescing adds mappings, never removes
per-event proof. Duplicate succeeds only on exact authoritative committed pair;
a later checkpoint/unrelated completed job is insufficient. No automatic deletion
or TTL for dedup/responsibility/completion proof until a separately approved
retention policy demonstrates every supported redelivery/restore is safe.

**Recovery:** restore ledger/outbox/dedup/responsibility/work/generation data to
one consistent recovery point, hold publication paused, reconcile pair/version
integrity and uncertain post-backup handoffs with retained incident evidence.
Mixed-time snapshots or missing bindings fail closed; no inferred repair/delete/
ACK. Operator must approve RPO/RTO, backup coverage, recovery authority and
storage horizon. Restrict integrity inspection to safe references; no price
payload or financial request body added to evidence.

**Rationale/alternative:** durable proof avoids successful duplicate lies;
TTL or retaining only latest job breaks old-event duplicates. Assess unbounded
growth explicitly with O3 rates/sizes and historical generations, including
backup/restore cost; retention needs its own review, not accidental forever
capacity guarantees. This ledger-reference evidence is not Twelve Data raw data.
It neither exempts raw prices from the separate 30-day termination/deletion
obligation nor authorizes retaining reconstructable provider data in bindings.
**Phase B tests:** duplicate after completion/coalescing, changed checkpoint/
version binding, missing work/generation evidence, concurrent duplicate, coherent
restore and intentionally mixed restore, inspection/repair refusal without
approval. **Owner/approver:** architecture / Holdings / operator. **Approval:**
BLOCKED; no-auto-deletion contract PROPOSED, retention/capacity/recovery evidence MISSING.

## Approval and evidence register

Approval must identify this proposal/base, individual O IDs, exact candidate
values/conditions, approving human's role/name, UTC date and verifiable decision
reference. One decision may cover multiple IDs only explicitly; merges/CI are
not approvals. Record superseded candidates historically. No fallback default
applies to unanswered decisions.

| ID  | Owner / approver roles (unchanged)            | Phase A missing evidence / decision                                                                        | Approval / human reference |
| --- | --------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | -------------------------- |
| O1  | Architecture / Platform / Holdings / operator | Names, deputies, coverage, alert channel, authority and escalation                                         | BLOCKED / MISSING          |
| O2  | Platform / Holdings / operator                | 5s/lock/cleanup/connection budget approval, real pool envelope, batch benchmark criteria                   | BLOCKED / MISSING          |
| O3  | Platform / Holdings / operator                | λ/sizes/μ, outage horizon, usable storage/WAL/backup capacity, threshold approval                          | BLOCKED / MISSING          |
| O4  | Architecture / Platform / Holdings / operator | Lifecycle, actual invocation accounting, lease/deadline/retry/fence approval and workload matrix           | BLOCKED / MISSING          |
| O5  | Holdings / Transaction / operator             | Full-history read/calculation/staging/memory budget approval, production envelope and benchmark acceptance | BLOCKED / MISSING          |
| O6  | Holdings / Platform / operator                | Intended Portfolio population, fair scan/detection bounds, lag thresholds, alerts/coverage approval        | BLOCKED / MISSING          |
| O7  | Platform / Holdings / operator                | Pause-not-drain, bounded shutdown/restart procedure, reactivation authority and deployment model           | BLOCKED / MISSING          |
| O8  | Architecture / Holdings / operator            | Evidence lifetime/storage, coherent backup/restore RPO/RTO, approved integrity repair procedure            | BLOCKED / MISSING          |

Required later Phase B report: environment/resource/version identifiers without
credentials; exact workload/population/rate/record sizes; measured timing/RSS/DB
wait/WAL/disk/drain; failures/retries/uncertainties; all O1–O8 executable scenarios;
approved financial vector results (not invented ones); separate operator
activation/deactivation sign-off. Ordinary CI tests existing regressions only.

**Next permitted action:** maintainer/operator review these candidates, supply
missing capacity/ownership evidence and explicitly accept/revise each O ID in a
separate recorded decision. This proposal's merge alone neither closes the gate
nor activates any downstream task or receiver.
