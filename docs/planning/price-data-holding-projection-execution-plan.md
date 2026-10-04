# Price Data and Holding Projection — Execution Plan

**Status:** Proposed for maintainer review
**Version:** `M4-PRICE-HOLDING-PLAN-v1`
**Milestone:** M4 — Price Data and Holding Projection
**Planning task:** `M4-PLAN-001`
**Branch:** `codex/m4-plan-001-price-holding-projection`
**Protected-main base:** `2ca1259828c10b8a82d1867a301b6c9d87f188ec`
**Dependency:** M3 closed through [PR #77](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/77), merge `cef20765bb0bdcc9d082f3773e2c58605ba84008`.
**Planning activation:** [PR #78](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/78), merge equal to the base above.
**Approval:** This plan proposes technical decisions and downstream gates. It neither accepts a new ADR nor authorizes runtime activation. Unresolved prerequisite evidence is not fabricated.

## 1. Purpose, precedence and completion boundary

Deliver two independent capabilities: sourced official-close observations and deterministic asset Holding/FIFO projections. The immutable Transaction ledger, canonical Asset catalog and sourced Price observations remain authoritative inputs. Holding outputs are derived and rebuildable. No Transaction command depends on provider, calendar, receiver or projection availability.

Precedence:

1. [Decision Closure](decision-closure-specification.md): COST_BASIS-v1, DECIMAL-v1, PRICE_SELECTION-v1 and CURRENCY_SCOPE-v1.
2. Accepted architecture/governance decisions: ADR-002/003/004/008/009/010/011/012 in the [Planning Baseline](planning-baseline.md), [ADR-013](../adr/ADR-013-solo-maintainer-merge-governance.md), [ADR-022](../adr/ADR-022-transaction-outbox-delivery-operational-policy.md), [ADR-023](../adr/ADR-023-transaction-publication-activation-prerequisite.md).
3. Recorded [provider permission](../governance/m3-price-provider-permission-gate.md), without expanding its approved audience/use.
4. This plan after review/merge; frozen M3 contract and implementation remain compatible.
5. Repository conventions, [module boundaries](../architecture/module-boundaries.md), [repository structure](../architecture/repository-structure.md) and [event standard](../architecture/worker-event-standard.md).

If sources conflict, stop the affected downstream task. Financial-policy changes require an expressly accepted ADR/version, not an implementation default. The gate in §4 is proposed, not approved here.

Only this plan and the documentation index change in M4-PLAN-001. The queue deliberately still references the existing M3 plan until a separately authorized control-plane PR updates that pointer and activates the reviewed downstream graph. No downstream task is executable from this PR alone.

## 2. Repository reality and reuse seams

Inspected at the base: [M3 plan](transaction-ledger-foundation-execution-plan.md), [acceptance matrix](../engineering/m3-transaction-ledger-acceptance-matrix.md), [completion report](../engineering/m3-transaction-ledger-completion-report.md), [domain](../architecture/transaction-domain.md), [application](../architecture/transaction-application.md), code/query/migration inventories, OpenAPI/types/tests, frontend features, Compose and CI. Older “Proposed”/“pending merge” prose in historical M3 documents is a delivery snapshot; verified PR #77 merge and maintainer closure establish the dependency.

| Existing capability                                                                       | Reuse and missing capability                                                                                                                                                                                 |
| ----------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Transaction immutable ledger/correction chains and exact decimal input                    | Preserve unchanged; no mutable edit or financial truth in projections. Domain decimal supports exact input arithmetic, not a completed FIFO rounding engine.                                                 |
| Portfolio-local sequence serialization and complete correction groups                     | Source checkpoint can identify a committed ledger prefix; current private replay query requires a sequence lock and is not a worker-facing snapshot API. Add a narrow Transaction-owned snapshot port later. |
| Five sqlc targets, migrations 00001–00005                                                 | No Price/Holdings target, observation/projection/lot table, provider mapping or calendar adapter exists.                                                                                                     |
| Platform stream-ordered outbox, caller-owned transaction dedup, ADR-022 engine            | Reuse; worker currently has no publisher and must claim nothing under ADR-023. No durable job-executions/inbox implementation exists.                                                                        |
| Asset has canonical symbol/exchange/type/USD identity                                     | No provider-specific symbol/MIC mapping exists. Do not assume normalized exchange equals provider MIC or ticker alone identifies a listing.                                                                  |
| Bearer authorization, owner-scoped Portfolio public reader, opaque IDs/errors/correlation | Reuse for every Portfolio-scoped M4 read; no new auth policy.                                                                                                                                                |
| Generated TypeScript, strict runtime schemas, React Query and memory session              | Reuse for server-authoritative views; no frontend FIFO, price selection or arithmetic.                                                                                                                       |
| Real HTTPS production-browser CI, disposable postgres-test, seven gates                   | Add synthetic M4 coverage inside existing gates later; no live Twelve Data dependency.                                                                                                                       |
| Worker attached only to internal database networks in Compose                             | Live-provider egress is not operationally ready. A narrowly reviewed worker egress/config change belongs to later Price work, never widening Caddy HTTPS trust.                                              |

## 3. Scope decisions proposed for approval

| Area               | M4 decision                                                                                                     | Exclusion                                                                        |
| ------------------ | --------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| Holding            | Quantity and remaining FIFO cost basis per Portfolio/Asset; calculation provenance and status                   | No manual Holding mutation, negative positions, shorting or margin               |
| Lots               | Internal deterministic rebuild state only (§6); no holding_lots table                                           | No user lot selection or tax reporting                                           |
| Realized gain/loss | Persist per-sale calculation output/provenance internally for reconciliation; not exposed in M4 UI/API          | Portfolio return/performance reporting remains M5 or later                       |
| Prices             | Immutable accepted/rejected observations, selection and freshness foundation, authenticated selected-price read | No raw provider proxy/export or price overrides                                  |
| Valuation          | No quantity × selected-price field in public Holding views                                                      | Holding market value, portfolio totals, allocation, dashboard and returns are M5 |
| Cash               | Explicitly outside M4; cash events do not change asset quantity/cost                                            | No cash balance, cash sufficiency/overdraft rule or cash UI                      |
| Eligibility        | US EQUITY/ETF, USD, recognized primary listing; existing NYSE/NASDAQ/NYSEARCA/AMEX financial scope              | No CRYPTO, FX, non-US market, corporate actions, transfers, derivatives          |
| Provider           | Twelve Data Venture, private beta, Time Series 1day/adjust=none                                                 | No fallback, reconciliation provider, news/fundamentals/intraday                 |
| Operations         | Durable ingestion/rebuild responsibility, safe recovery and retention enforcement                               | No broker, automatic dead-letter replay, public admin API or new admin UI        |
| AI                 | None                                                                                                            | No provider data in prompts, embeddings, logs of AI vendors or training          |

Price refresh must not rebuild FIFO cost basis: acquisition cost comes from ledger trade values, not market observations. This narrows the baseline's future Price→Holdings dependency: M4 Holdings does not need Price to calculate; M5 combines the two modules' public outputs.

## 4. M4-GATE-001 — required decisions/evidence before implementation

This is a downstream human-reviewed prerequisite, not a claim that M3 permission is blocked again. Recommended artifact: `docs/governance/m4-implementation-readiness-gate.md`. Engineering prepares references and test vectors; maintainer/Product/contract owner approves the relevant decisions. Never put credentials or confidential provider terms in it.

| Gate item                    | Present evidence/gap                                                                                       | Required closure and owner                                                                                                                                            |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Receiver/handoff             | ADR-023 permits future reviewed receiver; none approved                                                    | Accept the §8 concrete receiver contract, deployment/rollback and recovery ownership; architecture maintainer                                                         |
| Provider operating envelope  | Quotas, request weights, concurrency, asset-count and historical depth UNRESOLVED in M3 gate               | Written actual Venture entitlements, endpoint weighting, retry hints, EOD availability timing, permitted backfill range; provider/account owner                       |
| Credentials and environments | Server-side only is settled; mechanics/rotation/separation unresolved                                      | Secret delivery mechanism, rotation/revocation procedure, environment isolation and no-production-key CI; operator/account owner                                      |
| Canonical listing mapping    | Asset has no MIC/provider identity                                                                         | Verified supported-listing mapping, timezone and mapping-change procedure; no ticker guessing; provider/Engineering                                                   |
| Calendar                     | Policy requires provider-supported US exchange calendar; no source/adapter                                 | Approved Twelve Data-supported calendar source/service, entitlement, early-close/holiday coverage, failure behavior and versioned evidence (§10); operator/maintainer |
| Retention                    | Active subscription and deletion within 30 days after termination approved                                 | Termination signal/owner, raw-data inventory including backups, deletion/restore-sanitization proof, deadline monitoring and reviewed destructive procedure (§12)     |
| Display delivery             | Authenticated private beta approved; persistent browser caching unresolved                                 | Approve no persistent client cache and verify response/SSR/cache controls; any broader caching needs provider/contract-owner evidence                                 |
| FIFO correction edge         | M3 signed reversal replay and naive removal of canceled BUY can disagree (§5.3)                            | Approve complete cost-lot neutralization examples for all M3-valid same-time streams; cannot change Transaction validity implicitly                                   |
| Decimal reconciliation edge  | Aggregate-once rounding and per-lot persisted equality need a defined cross-lot residual allocation (§5.2) | Approve executable expected values/precision interpretation; any departure from DECIMAL-v1 requires an accepted policy ADR                                            |

No plan-tier quota or numerical concurrency/backoff default is guessed. Evidence absence leaves this gate BLOCKED and M4-CONTRACT-001 blocked. Commercial availability remains out of scope, not an invitation to invent commercial rights.

Recommend a dedicated new ADR, numbered from the next available slot when authored, for the concrete internal receiver/durable-job handoff. ADR-022/023 are not edited or marked superseded. If the FIFO/rounding examples require a policy extension, that is a separate expressly accepted financial ADR/version in the gate, not a receiver ADR footnote.

## 5. Holding calculation specification and policy checks

### 5.1 Ordinary FIFO, fixed by existing policy

Process each (portfolio, asset) independently. Read immutable facts through committed checkpoint C in ascending `effectiveAt, portfolioSequence, transactionId`; never createdAt order. Historical accepted Asset snapshots govern replay even if current display metadata changes.

- BUY creates a lot keyed by source transaction ID, with original/remaining quantity, acquisition order and exact original cost `quantity × unitPrice + fee`.
- SELL consumes oldest nonempty lots. For each lot consume `x = min(remainingQuantity, outstandingSaleQuantity)`. For partial consumption, recognized internal cost is `roundHalfEven18(remainingCost × x / remainingQuantity)`; subtract x and that cost from the lot, preserving its residual. Full consumption recognizes the entire remaining internal cost, including its residual, and closes the lot. Persisted cross-lot reconciliation still requires the explicit §5.2 gate.
- Sale proceeds are `quantity × unitPrice - fee`; realized gain/loss is proceeds minus consumed FIFO cost. Negative proceeds/gain are allowed: do not add a fee≤gross rule absent from M3.
- Holding quantity is remaining lot quantity sum; cost basis is remaining lot cost sum. Zero-quantity rows are not open positions; retain their calculation provenance.
- DIVIDEND, DEPOSIT, WITHDRAWAL and standalone FEE do not create/consume lots or change Holding cost. Reinvestment remains separate DIVIDEND and BUY.
- Oversell/negative quantity in projection is an invariant failure, never a negative Holding, invented lot or clamp to zero. Ledger validity remains M3's responsibility; projection failure does not roll it back.
- A reversal neutralizes its referenced original, never becomes a public trade; replacement is independently ordered and may itself be corrected. Complete relationship graph and both affected assets must be available.

### 5.2 Exact representation, rounding and reconciliation

Use a Holdings-owned pure calculation value type over arbitrary-precision base-ten integer coefficients/scales (`math/big.Int`), with exact integer/rational division where needed. No float32/float64, SQL financial arithmetic, frontend Number/parseFloat or database implicit rounding. Reuse M3 public immutable values through lossless adapters; do not expand Transaction.Decimal with projection responsibilities.

Input quantity/unitPrice/fee/amount: validate ≤12 fractional digits without rounding. Exact multiplication may produce 24 fractional digits and must retain them. Proportional allocation uses 18 fractional digits with half-even at that policy-defined division boundary; retain greater exact precision where already present. Keep remainders on remaining lots, not lost to each partial-sale rounding.

Aggregate sale cost/proceeds and Holding monetary outputs round once to 12 places with half-even. Persist finite exact NUMERIC with explicit scale checks ≤12 and no undocumented integer cap (at least 38 significant digits); reject excess-scale inputs before storage, never use a typmod as the rounding implementation. Internal lot working state is not persisted at a lower precision. Currency display formatting uses exact decimal strings and USD minor units only at presentation; displayed values do not feed calculations.

Final lot closure assigns its residual so total recognized acquisition cost equals original acquisition cost at persisted precision. Required vectors include 1/3, repeated tiny partial sells, ties with odd/even last digits, multi-lot sales and final closure after many rounded outputs. Quantity/cost conservation and deterministic hashes must hold across rebuilds.

**Explicit unresolved precision intersection, blocking §4:** two lots each costing `0.0000000000006` USD, both closed in one sale, demonstrate why independent per-lot 12-place rounding cannot be assumed: each rounds to `0.000000000001`, while the aggregate `0.0000000000012` rounds to `0.000000000001`. These input costs are representable by legal ≤12-scale quantity and price products. The gate must specify the granularity of “sum recognized cost” and residual attribution consistent with aggregate-once rounding. Recommended direction is internal per-lot precision plus explicitly reconciled calculation-level rounding provenance, not silently round every lot first. Do not invent an epsilon, discard a residual or assert both inconsistent 12-place sums are equal. Ordinary equations above are fixed; cross-lot persistence allocation is not claimed resolved before the gate.

### 5.3 Corrections, same-time ordering and compatibility gate

Always rebuild affected history, not apply a delta to the last rounded projection. Original/reversal/replacement and all later replacements remain immutable source references. Complete correction relationships are read at checkpoint C; never snapshot only one half of an atomic correction.

Inspection of `transaction/domain/replay.go` and `correction.go` shows reversal retains original effectiveAt but receives a later sequence. A concrete same-time example, all for one Asset:

1. BUY A: quantity 10, sequence 1, time T.
2. SELL: quantity 10, sequence 2, time T.
3. BUY B: quantity 10, sequence 3, time T.
4. Correct A: reversal sequence 4 at T, replacement sequence 5 at later T2.

M3 signed-quantity replay is 10, 0, 10, 0, then replacement quantity: nonnegative. Simply deleting A and its reversal before FIFO replay starts with SELL and becomes negative. This is a compatibility counterexample, not authorization to change M3 or a claim that M3 quantity validation is broken.

Gate recommendation: a reversible lot-consumption journal with original cost/proceeds provenance, followed by affected-history recomputation, rather than treating reversal of SELL as a new BUY at sale price or canceling facts without checking their ordered dependencies. The gate must settle cost attribution for the consumed-original-BUY example and chains, supply expected remaining basis/realized outputs and prove compatibility. No runtime, schema or financial policy is implemented until that decision is approved. Any proposed restriction of previously accepted commands requires a separate compatibility decision.

## 6. Lot persistence, calculation provenance and versioning

Proposed decision: **lots are internal rebuild state only** in v1. No `holding_lots` table, lot endpoints or partial incremental checkpoint of rounded lot balances. Rationale: correctness-first full replay avoids two competing lot truths and difficult intermediate-precision persistence. This is a deliberate choice, not a missing migration.

Persist Holdings-owned generations and calculation records with:
Portfolio/Asset identity; generation ID; calculation algorithm version; COST_BASIS-v1; DECIMAL-v1/half-even rounding version; committed ledger checkpoint C; reproducible sequence range 1..C plus source digest and relevant transaction/correction identities; per-sale recognized cost/proceeds/realized output; final Holding quantities/bases; calculatedAt; status; request/rebuild identity.

Range includes cash facts and canceled records even if they have no quantity effect, so the source digest has a deterministic fixed serialization. Sequence is a source boundary, not a business timestamp. M4-GATE-001 resolves the correction/cost attribution vectors before calculation-record shape is frozen.

Generation identity binds Portfolio, C, calculation versions and requested rebuild scope. Same source/version recomputes equal financial results; generated IDs/time/job metadata are excluded from financial equality/hash assertions. New algorithm/policy versions build separate generations. Never replace active output piecemeal or overwrite prior calculation evidence. No automatic historical calculation-record deletion in M4; derived retention/compaction requires a separately reviewed policy. Raw provider deletion (§12) is distinct.

## 7. Snapshot, rebuild and concurrency protocol

Recommended v1 optimization policy: **full Portfolio replay for every relevant change**. Incremental lot updates are deferred. Single-Asset operator requests also execute full Portfolio replay in v1; scope narrows diagnostics, not publication atomicity. This handles cross-Asset corrections and retains one coherent generation.

1. Durable rebuild request identifies Portfolio, requested minimum checkpoint, algorithm/version and cause (event/operator/reconciliation).
2. Transaction exposes a new internal public snapshot reader, implemented in Transaction-owned queries. It returns a complete committed prefix C, facts and correction links in one consistent PostgreSQL snapshot. No Holdings query reaches into Transaction tables and no source sequence is allocated for a read. Existing owner-facing history API/private replay query is not repurposed without an explicit new port.
3. Use a read-only REPEATABLE READ snapshot; stream/paginate within that same transaction/snapshot, not independent page snapshots. Ledger inserts/sequence/correction writes are atomic, so C cannot bisect a committed correction. Empty ledger C=0.
4. Calculate outside the ledger write critical section. Durable per-Portfolio work lease/generation token serializes publication; PostgreSQL fencing prevents a crashed/expired worker from replacing a newer generation.
5. Write candidate rows/calculation records under an unexposed generation. At final commit compare-and-swap the generation pointer/checkpoint; never decrease C or switch to an obsolete calculation version. Mark completed request in the same owning transaction.
6. A later ledger write C'>C does not invalidate the coherent C snapshot, but the response must be pending/stale, not “current”. Read-time comparison through Transaction's public checkpoint port detects backlog even before event handoff. Queue another rebuild via real event/reconciliation.
7. Failed/canceled attempts leave the previous complete generation visible with status and source metadata. No partial candidate is readable. Retry with same request is idempotent; expired leases are reclaimed with fenced writes.
8. On restart inspect durable work and resume/recompute from source, not memory. Missing projection schedules full replay; stale checkpoint/version schedules full replay; duplicate events/jobs do not duplicate active output.
9. An operator rebuild uses an authenticated operational command/CLI with named scope, correlation, dry-run/preflight, bounded resource limits and explicit approval for destructive cleanup. No public user rebuild endpoint. Full-history computation is not an automatic replay of dead-letter outbox events.
10. Periodic reconciliation compares Transaction public checkpoints and projection generations to detect missed/failed work. It does not mark missing events PUBLISHED or rewrite consumer dedup. Work/lease/retry operational values must be approved in §4 and tested before activation, not guessed by a handler.

Costly snapshots have bounded execution/memory budgets and explicit failed/pending state rather than silently truncate history. Scheduling constraints do not change financial answers. Resource exhaustion or unresolved correction vectors leave state failed with a safe reason and retained last-good generation.

## 8. Concrete receiver proposal — human approval required

Recommended receiver identity: `holdings_rebuild_request_v1`, Holdings-owned application receiver composed inside the existing worker, no broker or network event bus. Its job is a **durable handoff**, not synchronous unbounded FIFO computation inside the 60-second outbox lease.

Accept exactly `transaction.recorded.v1` and `transaction.corrected.v1`, event version 1 and reference payload schema 1. Validate required reference roles and Portfolio/aggregate consistency by reading authoritative facts through Transaction public ports. Unknown versions/invalid references fail closed; no success for an event not understood.

For each event, in one caller-owned database transaction:

- call Platform ConsumerDeduplicator.RecordIfNew with `holdings_rebuild_request_v1` and immutable event ID;
- if new, insert a durable idempotent rebuild request referencing the Portfolio, event, required source checkpoint and calculation version;
- commit both or neither.

A duplicate returns success only when the previously committed dedup/request pair proves durable responsibility. Missing responsibility evidence is an integrity failure, not a successful acknowledgement. Retain that evidence while duplicate delivery remains possible; no automatic dedup/request deletion is authorized by this plan. Different events may coalesce calculation work while retaining their individual responsibility records. No dedup insert before an independently committed side effect.

Publisher success means this transaction committed, **not that projection is current**. Only then may the existing engine mark PUBLISHED with its owned claim token. Crash after handoff/before ACK re-delivers safely; duplicate handoff cannot lose or duplicate work. Missing receiver still means zero claims/attempts exactly as ADR-023.

Separate projection workers consume durable work with leases/fencing and bounded retries. Their failure does not revoke a real handoff or falsify outbox publication; expose failed/pending projections. Generic job mechanics belong to Platform; calculation and rebuild intent belong to Holdings. Platform must not contain Asset/FIFO/price-selection rules.

Activation gate must approve topology, backlog capacity, consumer identity/version, handoff atomicity, processing retry/dead-letter policy, operator ownership and rollback. ADR-022 delivery defaults remain 60s lease, 5s base, 5m max, 10 invocations, batch 50, poll 2s. Invocation 10 failure dead-letters; >10 recovery invokes no publisher. Dead-letter predecessor blocks its stream; no automatic replay/skip/ACK. Receiver disabled/unconfigured returns to inactive composition with no new claims, never resets durable state.

Deploy migration/receiver-compatible code before enabling receiver composition. Test backlog from M3, duplicate claims and worker restart. Activation is not a Boolean flag with an unimplemented receiver. A separate named Platform job foundation task precedes database/business tasks (§17); it is not assumed already delivered.

## 9. Price observations and ingestion identity

### Canonical record

Price owns source mapping and observations; Asset still owns canonical catalog identity. Mapping is versioned with approved provider symbol, primary MIC, exchange timezone, valid interval and evidence reference. Two provider symbols must not ambiguously resolve one listing/version. No public mapping mutation API.

Each immutable observation records: ID; Asset ID; Twelve Data/source identifier; mapping version; provider symbol/MIC; USD; exchange-local market date; reported timestamp if supplied (nullable, never fabricated); regular-session classification; interval 1day; adjust=none; raw close text; exact normalized close; retrievedAt UTC; ingestion request/run identity; provider-response/item digest; validation version; accepted/rejected status and bounded rejection reason; provenance schema version. Preserve an allowlisted bounded raw observation sufficient to explain normalization, not credential-bearing HTTP request/headers or an arbitrary full provider envelope. adjusted_close, if present in source evidence, remains separate and never substitutes for close.

Accepted requires recognized eligible EQUITY/ETF listing, USD, valid official regular-session completed-day semantics and valid decimal. Do not silently round an over-scale provider close into eligibility: require provider precision evidence at gate, reject incompatibility and retain safe evidence. No runtime price policy may invent a finance-compatible value.

Malformed items identifiable to an Asset become rejected observations; response-wide unparseable transport/schema failures become safe ingestion-attempt records, not fabricated Asset/date observations. Oversize content is rejected with bounded digest/reason rather than copied into logs. Raw evidence still falls under retention deletion.

### Idempotent acquisition, not destructive latest-price uniqueness

- Durable acquisition job key: provider, mapping version, Asset, requested date/range, scheduled run identity and operation version. Retries of the same run share this identity.
- A committed response has a stable response/item digest and the first successful retrieval timestamp for that run. Identical persistence retries return its same observations; transaction rollback retries cannot duplicate them.
- A new intentional scheduled fetch/re-fetch is a new run, even for the same price/date. Record its retrievedAt and observation; it is not destructive deduplication by (Asset,date).
- Changed close for a market date is a new sourced observation, never UPDATE of the old one. Same date unchanged close at a genuinely later retrieval is also retained.
- Concurrent execution of one job has a durable lease/token plus unique run/item identity; only the fenced owner commits. Different fetches remain distinct.
- Selection ties use immutable retrieval ordinal assigned by Price at persistence, after retrievedAt ordering, then observation ID as final deterministic tiebreaker. A technical tie never changes date-first/most-recently-retrieved policy.
- No public “latest price” column on Asset; no price row in Holding cost calculation. Rejected observations do not overwrite earlier accepted ones.

## 10. Calendar, selection and freshness

Define Price application ports `OfficialCloseProvider` and `USExchangeCalendar`; concrete provider adapters remain infrastructure. Calendar returns a versioned session record: primary MIC, exchange-local trading date, timezone, actual regular-close UTC instant, session completed state, holiday/early-close evidence and source/retrieval/version.

**Source decision:** use the approved provider-supported US exchange calendar required by PRICE_SELECTION-v1; the concrete Twelve Data-supported endpoint/dataset and entitlement are not in repository evidence. M4-GATE-001 must name and approve them before integration. Do not assume Time Series dates constitute a future holiday calendar, use weekday-only logic, infer holidays, or silently add a third-party dependency. A new external calendar source needs explicit governance approval; provider-incompatible policy needs an ADR.

As of a supplied evaluation instant, resolve most recently completed session D and prior trading session P using this calendar. API caller-supplied optional historical cutoff, if approved in contract, cannot be after the server's completed-session cutoff. M4 defaults to current completed session; no future-price/valuation prediction.

Select accepted, USD, listing-matched observations from the approved provider with market date≤cutoff, latest market date first then latest retrievedAt (technical tie as §9). Preserve rejected/prior observations and selection references. Record PRICE_SELECTION-v1, observation ID, source, data-as-of close, retrievedAt, calendar version and evaluatedAt.

Classification uses retrieval delay from the observation's actual session close, not from midnight or browser timezone:

- fresh: date D and retrieval within 36h after that close;
- stale: date P, or retrieval later than 36h and no more than 72h;
- unavailable overrides both: no eligible data, older than P, retrieval more than 72h late, invalid/rejected-only data or calendar unavailable.

Retrieval before the official close cannot establish an accepted close. Test inclusive 36h/72h boundaries, holiday gaps, DST and early closes. A latest rejected observation does not erase an older eligible one, but older data may still be stale/unavailable under these rules.

Selected observation and usability are separate. An unavailable response has no usable price; it may carry permitted source/status metadata, never zero or last-known-as-current. The M5 inclusion/denominator rules are inherited policy references only; no valuation total is computed in M4.

## 11. Scheduling and operational readiness

Price ingestion runs after the calendar-confirmed regular close plus a provider-evidenced publication delay. Exact delay/quota/window is a gate input, not a guessed time. Universe is the reviewed bounded set of canonical supported Assets with verified provider mapping; enumerate through an Asset public paging port, not direct table access. Gate must prove account asset-count/quota capacity before approving that universe. No ticker discovery/provider calls triggered by anonymous or ordinary-user arbitrary query input.

Platform durable jobs record job ID/type/version, semantic idempotency key, scope, due time, attempts, lease/token, next attempt, state, correlation and safe failure code. Price interprets provider outcomes; Platform only schedules technical work. Multi-worker shared rate budget enforces actual request weighting and all licensed limits; partition/batch only after approved capability evidence. Jitter distributes scheduled requests; bounded backoff honors verified retry hints. No unlimited retry storm or synchronized polling from every browser.

Partial response: commit independently validated observations with their run identities; mark missing/rejected items and retry only approved missing scope, preserving provenance. Timeouts/network/429/server errors are not empty successful prices. Invalid credentials suspend ingestion and alert safely. Provider failure never blocks ledger or FIFO jobs.

Operator backfill/re-fetch uses the same durable jobs/idempotency/quota controls, explicit date range and historical-depth limits. No accidental full-history fetch on restart. Job dedup lifetime, retry caps, timeout/concurrency and manual recovery rules must be recorded at the gate; ADR-022 outbox defaults are not silently reused as provider quota policy.

Secrets: server-side secret reference through approved deployment injection; ignored local secret files only; synthetic non-secret placeholders for ordinary CI. Production and test keys never share default configuration. Missing/invalid secrets prevent live-provider job activation (or fail explicitly configured live-worker startup), not API/ledger startup. Rotation procedure includes provider revocation, overlap only if supported, safe reload/restart and verification without exposing values. Redact URLs/query parameters and error bodies; allowlist outbound host/TLS, no user-controlled provider URL. Actual infrastructure/secret changes remain human-governed.

## 12. Permission, retention and attribution

M3 permission remains APPROVED for Twelve Data Venture, owner and explicitly invited authenticated beta users; it is not operational-readiness evidence. No new licensing approval is inferred here. Private-beta admission must be operationally enforced before licensed display is deployed; existing public registration alone is not proof a caller is in the approved invited audience. Gate records the approved deployment/admission control, without redesigning Authentication silently.

Subscription lifecycle must be an operator-owned recorded input. On termination: stop ingestion and licensed display; schedule bounded deletion of raw provider data to complete within 30 days, not merely start then. Inventory covers accepted/rejected observations, raw/normalized prices, source payloads, caches, replicas, exports (none approved), diagnostic artifacts, backups and retained snapshots. No “immutable” constraint may defeat the legal deletion operation.

Price-owned purge deletes licensed values only via a reviewed controlled operation; retain minimal non-reconstructive deletion audit/counts/job status where permitted. No raw close hidden in digests/metadata intended to evade deletion. Backup expiry or selective restore sanitization must be approved and demonstrably satisfy the deadline; otherwise live ingestion remains blocked. Alert on approaching deadline, retry safely and provide completion evidence. New migrations' Down must not be used as a purge tool.

Ledger-derived Holding/FIFO records contain no provider close and remain reproducible without it. “Derived” is not a blanket license to retain reconstructable raw-price analytics. Future M5/provider-derived outputs need an explicit classification/deletion dependency.

Attribution on every selected-price display and any future provider-derived chart: “Data provided by Twelve Data” (or approved “Source: Twelve Data”), linked to the main Twelve Data website with no nofollow. No logo planned; no implied endorsement. Later exchange-specific requirements are gate/operational update inputs. Pure ledger quantity/basis panels do not falsely attribute ledger numbers to Twelve Data; a combined price panel still carries attribution.

Authenticated responses use no-store; disable SSR/shared CDN caching of licensed responses and browser persistence. React Query data is memory-only, removed on logout. No localStorage/sessionStorage/IndexedDB/service-worker persistence, raw API, CSV export or AI forwarding. Persistent browser caching remains unapproved, not “resolved” by assumption.

## 13. Persistence ownership and constraints

New forward Goose migrations after current 00005, assigned when tasks run; never rewrite M1–M3 migrations. Tables below are planned, not existing. No Holding lot table. Financial calculations remain Go-side.

| Owner/table                                  | Identity and constraints                                                                                                                                          | Access/index/lifecycle                                                                                                                                                                   |
| -------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Platform job_executions                      | UUID job ID, bounded type/version/scope/idempotency identity, unique versioned scope key, nonnegative attempts, consistent state/lease/time, safe code only       | Due-state/due-time index, SKIP LOCKED claims and fenced updates; no business/financial payload                                                                                           |
| Price provider_asset_mappings                | Asset FK RESTRICT, provider/symbol/MIC/timezone, mapping version/effective range/evidence; unique nonambiguous listing mapping                                    | Provider+Asset/version lookup and eligibility scan; version append, no mutation of observations' source meaning                                                                          |
| Price price_observations                     | UUID, Asset/mapping references, run/item identity, date/time/provenance/validation; USD accepted values, exact finite NUMERIC ≤12, accepted/rejected shape checks | Accepted partial index (Asset, market_date DESC, retrieved_at DESC, ordinal DESC, ID DESC); run/item unique; retention scan by subscription/source. Append except controlled legal purge |
| Price ingestion run/item records             | Link Platform job to provider run/attempt, immutable successful response digest/retrieval time, bounded rejection facts                                           | Atomic observations/run completion; no credential/request-header persistence; governed raw-data purge                                                                                    |
| Price calendar session records               | Provider-supported source/MIC/date/version, close instant/timezone, evidence/version                                                                              | Unique source-version-session identity; lookup completed/prior sessions, no inferred sessions; provider retention classification required                                                |
| Holdings projection_generations              | UUID, Portfolio FK RESTRICT, C, algorithm/policy versions, input digest, calculatedAt, consistent pending/running/completed/failed metadata                       | Unique semantic rebuild identity; generations immutable when completed; Portfolio/checkpoint/version index                                                                               |
| Holdings holding_projections                 | generation+Portfolio+Asset scoped key/FKs, USD, quantity≥0 and basis≥0 with finite scale≤12 checks                                                                | Generation/Asset keyset index; rows published only as a complete generation; no manual write                                                                                             |
| Holdings calculation_records                 | Generation/Portfolio/Asset, source range/IDs/digest, formula/version, per-sale outputs and safe provenance                                                        | Generation/Asset/source transaction indexes; immutable completed evidence; exact monetary numeric outputs; no provider prices                                                            |
| Holdings projection_heads / rebuild requests | One active generation pointer per Portfolio, checkpoint, fencing token; durable event/request identity and desired checkpoint                                     | CAS/row-lock active swap; event uniqueness; pending lease recovery; Platform dedup inserted atomically with durable request                                                              |

Additional tables are justified by durable responsibility, reproducible generations and mapping/calendar provenance, not financial source duplication. Source Transaction IDs/ranges are read through Transaction interfaces; FK integrity may reference public identity columns but no cross-module direct SQL reads/writes. Separate Price and Holdings sqlc targets; Platform job SQL remains Platform. Gate must finalize precise record granularity after its financial vectors; M4-DB-001 may not choose financial semantics.

Integration evidence: realistic-cardinality EXPLAIN plans for selection, keyset Holding pages, generation lookup, due jobs and deletion batches; indexes merely existing is insufficient. Independent pools prove claims, dedup, fenced replacement, repeated run identity and no torn snapshots. Empty/current/up/down/up plus fail-closed Down when durable observations/calculation evidence/jobs exist. Data removal requires explicit retention/archive approval; rollback never silently drops evidence.

## 14. Module and API direction

Introduce `backend/internal/price` and `backend/internal/holdings`, each domain/application/infrastructure/transport/composition. Public ports exchange domain-neutral source DTOs, not pgx/sqlc. Composition may bind module-owned adapters to a caller transaction as M3 does; generated types stay private.

Transaction owns new snapshot/checkpoint reader and queries; Asset owns canonical listing enumeration; Portfolio owns access proof including archived read access. Holdings owns FIFO and rebuild intent; Price owns provider mapping/validation/selection/calendar. Platform owns generic jobs, outbox, dedup, locks and observability only. Extend boundary checks and generated-drift registration in their owning implementation PRs, not now.

Proposed public reads for M4-CONTRACT-001 to freeze (no writes):

- GET `/api/v1/portfolios/{portfolioId}/holdings`: owned Portfolio, generation-bound cursor page of positive-quantity Asset positions; quantity/basis decimal strings, USD, source C/version/status.
- GET `/api/v1/portfolios/{portfolioId}/holdings/{assetId}`: owned position plus permitted calculation/provenance summary, not lots/realized return or market value.
- GET `/api/v1/portfolios/{portfolioId}/holdings/{assetId}/price`: owned existing position, single selected official close/status/provenance/attribution, licensed audience check. No arbitrary-symbol provider proxy, raw history/export or combined valuation.

All require BearerAuth; no idempotency header on GET. Derive principal server-side. Cross-owner/nonexistent/unrepresentable Portfolio/Asset IDs have ownership-safe 404 using existing ErrorEnvelope/correlation convention. Contract task freezes distinctions for absent position vs missing projection, bounded opaque cursor/limit (recommended default 50/max100), schema/text/time bounds and exact errors before runtime. No endpoint is frozen by this planning document alone.

A no-generation/pending Portfolio response is distinguishable from a current genuinely empty Portfolio; quantity/basis are absent until known, never fabricated zero. Generation-bound cursors carry generation ID and full Asset-ID ordering boundary; retain referenced complete generation during pagination, authorize every page, mark its snapshot status. Obsolete unavailable generation yields a stable restart-pagination response rather than mixing generations.

Single reads/status use current checkpoint to label current/pending/stale/failed/rebuilding, with last-good C/time where available. Price freshness is separate fresh/stale/unavailable, never confused with projection lag. Data unavailable is an explicit read-model state; unexpected infrastructure errors use safe 5xx/retry behavior, not empty success. Exact 200/404/conflict/service-unavailable distinctions belong to contract tests.

No public ingestion, rebuild, correction-of-Holding, lot, realized-gain, market-value, valuation, allocation or dashboard endpoint. Operator interfaces are process/CLI boundaries with explicit authority, not an ungoverned user-admin surface.

## 15. Frontend and eventual-consistency UX

Add Portfolio→Holdings navigation and `/app/portfolios/[portfolioId]/holdings` only after HTTP delivery. Use generated types plus strict response validation, owner/session-scoped React Query keys, server pagination/order and accessible controls.

Show quantity, USD cost basis, source checkpoint/time and current/pending/stale/failed/rebuilding state. A pending/no-generation result is not “no holdings”; preserve last-good data labeled stale while offering safe read retry. New ledger acceptance invalidates queries and displays projection pending, not an optimistic calculated Holding. Archived Portfolio remains read-only.

Separate selected-price panel shows close/date/source/retrieval/freshness and required attribution or explicit unavailable. Never multiply price by quantity, compute average cost/FIFO/gain, infer freshness, or synthesize missing values. No charts or provider-derived export in M4. Client/server component boundaries must keep provider secrets out of SSR props, bundles, errors and hydration data; only authorized response DTOs cross them.

## 16. Verification, security and observability

| Layer                   | Required tests/evidence                                                                                                                                                                                                                                                                                          |
| ----------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Holding domain/property | Exact FIFO, one/multiple lots, partial/full closure, fractional quantities, half-even ties, reconciliation vectors from gate, negative proceeds, six public kinds, no negative positions, input permutation→same sorted result, correction chains/backdates/same-time sequences and compatibility counterexample |
| Rebuild application     | Forward/backdated BUY/SELL, old BUY/SELL correction, cross-Asset replacement, stale/missing generation, single-Asset request/full replay, duplicate event, concurrent ledger write during rebuild, crash before/after swap, version upgrade and idempotent operator rerun                                        |
| PostgreSQL              | Migrations/constraints, realistic query plans, independent-pool claims/dedup, one handoff responsibility, atomic candidate swap, no checkpoint regression, immutable prior generations, failed/pending output visibility, observation identity/re-fetch/correction, purge and sanitized restore                  |
| Price adapter/calendar  | Synthetic deterministic fixtures, exact raw/normalized mapping, rejected schema/currency/MIC/session/scale, 429/timeouts/auth errors/partial response, date/close/DST/holiday/early-close boundaries; fresh/stale/unavailable at 36h/72h, failed calendar                                                        |
| Contract/HTTP           | Owned and cross-owner reads, opaque IDs, no writes/raw proxy/export, generation cursor stability, decimal strings, metadata/status/incomplete distinction, no-store and safe errors/correlation                                                                                                                  |
| Frontend                | Server authoritative display, no financial math, paging/loading/empty/pending/rebuilding/error retry, logout cache removal, source attribution/dofollow link, absent usable price and no M5 UI                                                                                                                   |
| Real HTTPS E2E          | Chromium→Caddy→Next production→API/worker→postgres-test; synthetic licensed-data stand-ins inserted only into explicitly isolated test DB; no API/Auth mocks, no live provider key/call; real BUY/SELL/FIFO, backdate/correction/rebuild, status/freshness/attribution, owner isolation and retry-free results   |
| Receiver/operations     | M3 backlog acceptance, commit-before-ACK, duplicate handoff, lease-expiry fences, ADR-022 attempt limits/dead letters, no receiver zero claims, durable work failure/restart, termination/purge drill                                                                                                            |

Ordinary CI uses synthetic provider HTTP fixtures or an explicit test-only adapter; never a production no-op receiver or silently injected production prices. E2E fixtures require literal test-DB safety checks, provenance labels, repeat safety and teardown. Adapter contract tests are separate from real browser integration.

Optional operator live-provider smoke is non-mandatory: approved active subscription, server-side secret injection, tightly bounded approved Asset/date request/quota, no secrets/raw response artifacts, safe result/provenance summary. Provider downtime never makes mandatory PR CI dependent on live service. Real sourcing evidence for M4 closure must come from an approved operator smoke; synthetic evidence alone does not prove the production adapter can retrieve official close.

Metrics/logs: fetch outcome/latency, rate-limited count, accepted/rejected counts by safe reason, job backlog/age, lease conflicts, dedup hits, projection checkpoint/lag, rebuild start/complete/failure, stale/unavailable counts, retention deadline/purge completion. Use bounded labels; Portfolio/Asset/event IDs in access-controlled diagnostics, not high-cardinality metric labels. Never log credentials, Authorization, raw provider body, financial command bodies or cookie/token values.

SECURITY_EXCEPTION-001 remains `braces@3.0.3 / GHSA-vfj7-8cjw-p6xm / ACCEPTED TEMPORARY RISK / expires 2026-10-31`. It is not fixed, remediated, suppressed or extended. Every later task evaluates actual audit state; reaching expiry or a remediation trigger requires separate action, not continued reliance on today's green scan.

## 17. Proposed ordered task graph

All arrows below are **merge dependencies**, not permission to start parallel unmerged work:

```text
M4-PLAN-001
→ separately reviewed control-plane activation
→ M4-GATE-001
→ M4-CONTRACT-001
→ M4-PLATFORM-001
→ M4-DB-001
→ M4-BE-001
→ M4-BE-002
→ M4-BE-003
→ M4-BE-004
→ M4-FE-001
→ M4-E2E-001
→ M4-VERIFY-001
```

Platform foundation is explicit because durable jobs do not yet exist and a receiver cannot acknowledge responsibility to memory. This adds no broker. Gate failure stops all implementation; planning completeness is not evidence of gate approval.

| Task / complexity        | Dependencies                             | Scope and acceptance                                                                                                                                                                               | Intended write paths                                                                                                                                                  | Non-goals / risk / human gate                                                                                                                     |
| ------------------------ | ---------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| M4-PLAN-001 / Large      | M3 closure, merged planning activation   | This plan, requirements mapping, proposed decisions/graph and index                                                                                                                                | This document; docs/README.md                                                                                                                                         | No implementation or queue mutation; maintainer approves plan                                                                                     |
| M4-GATE-001 / Large      | PLAN merged, graph activation authorized | §4 receiver ADR/evidence, actual provider/calendar/secret/retention readiness, approved correction/rounding vectors; every required row resolved with owner/reference                              | docs/governance/m4-implementation-readiness-gate.md, new scoped ADR(s)/policy clarification only if explicitly approved, docs index                                   | No adapter/schema/runtime; missing evidence blocks; never fabricate Product/legal/architecture approval                                           |
| M4-CONTRACT-001 / Medium | GATE approved and merged                 | Freeze §14 reads, state/decimal/paging/error/no-store bounds; contract tests and deterministic TS generation                                                                                       | packages/api-contracts/openapi/v1.yaml, tests/m4-contract.test.mjs, generated/api.ts; only obsolete earlier future-scope assertions                                   | No runtime/schema; preserve M1–M3; gate-resolved policy only                                                                                      |
| M4-PLATFORM-001 / Large  | CONTRACT merged                          | Generic durable job persistence/interfaces, leases/fencing/idempotency/retry, safe observability and caller-owned handoff seam; real concurrency tests                                             | new Platform-owned migration, queries/platform, internal/platform job/database packages and sqlcgen, directly related docs/CI drift tests                             | No business FIFO/provider rules or delivery activation; approved gate operational values; maintainer review                                       |
| M4-DB-001 / Large        | PLATFORM merged                          | Price/Holdings tables from §13, separate sqlc targets, constraints/query-plan/retention/migration tests; Transaction-owned snapshot query seam without changing financial writes                   | new migrations, sqlc.yaml, queries/price and holdings, module infrastructure/database + generated; narrow queries/transaction snapshot reads; CI registration         | No financial calculation in SQL; no holding_lots assumption; no M3 migration rewrite; residual record shape already approved                      |
| M4-BE-001 / Large        | DB merged                                | Pure FIFO/rebuild calculator, exact precision/approved vectors, provenance/generation atomic publication, Transaction public snapshot/checkpoint port and adapters                                 | internal/holdings domain/application/infrastructure/composition; narrow transaction application read ports/owned adapter; module boundary checks/tests/docs           | No Transaction semantic redesign, Price/network, active receiver or public routes; correction/reconciliation proof is blocking                    |
| M4-BE-002 / Large        | BE-001 merged                            | Price mapping/provider/calendar, observation validation/idempotency, selection/freshness, quota-controlled jobs, secrets/retention operations, synthetic adapter tests and optional operator smoke | internal/price, narrow Asset public enumeration/mapping seams, worker composition for approved Price jobs, approved Compose egress/config, focused scripts/docs/tests | No Holding valuation or raw proxy; no credentials committed; live activation only with gate evidence and operator approval                        |
| M4-BE-003 / Large        | BE-002 merged                            | Real durable Holdings receiver, Platform dedup+request atomicity, rebuild job processing/reconciliation, backlog/restart/failure runbook and explicit activation proof                             | internal/holdings receiver/orchestration, platform integration adapters only, cmd/worker composition, focused integration tests/config/docs                           | No broker/no-op ACK or ADR-022 changes; receiver runtime remains inactive until accepted ADR and reviewed implementation/deployment prerequisites |
| M4-BE-004 / Medium       | BE-003 merged                            | Owner-scoped frozen read routes, generation/paging/state/error/correlation/no-store behavior; authenticated integration tests                                                                      | price/holdings transport/http and composition, cmd/api, focused tests                                                                                                 | No compute-on-request FIFO, ingestion/rebuild user routes, M5; no backend policy shortcuts                                                        |
| M4-FE-001 / Large        | BE-004 merged                            | Holding and selected-price read-only UI, typed adapter/query keys, status/provenance/attribution, no-store/logout behavior and focused tests                                                       | apps/web/src/features/holdings and price presentation, relevant app routes/Portfolio navigation                                                                       | No client financial calculation, realized-return UI or provider secret; human UX review                                                           |
| M4-E2E-001 / Medium      | FE-001 merged                            | Real synthetic-stack §16 flows/seed safety, no external CI dependence, retries=0; retain Auth/M2/M3 regression evidence                                                                            | apps/web/tests/m4-e2e, Playwright config/scripts, synthetic test fixtures, existing browser-e2e job/package scripts/runbook                                           | No runtime fix disguised as test change; failures report scoped follow-up; no production fixture loading                                          |
| M4-VERIFY-001 / Large    | Every preceding task merged              | Acceptance matrix/report, actual source/receiver/precision/rebuild/security/scope evidence and current-head CI; closure recommendation                                                             | docs/engineering/m4-price-holding-acceptance-matrix.md, completion report, docs index; factual approved-status reconciliation only                                    | No hidden implementation; blockers prevent closure; final maintainer gate                                                                         |

Every task carries the same ADR-013 seven checks, GitGuardian, current-main branch, resolved conversations and unchecked maintainer approval until supplied. Proposed paths are scope ceilings, not permission for unrelated rewrites. A new infrastructure/dependency/financial policy outside the approved gate stops the task.

## 18. Definition of Done and planning acceptance

M4 closes only after M4-VERIFY-001 verifies and maintainer merges its evidence:
approved gate/receiver contract; real sourced official close with provenance and operator evidence; legal audience/attribution/retention compliance; deterministic observation identity with no destructive overwrite; approved calendar/freshness; exact FIFO/half-even/reconciliation/correction vectors; restartable fenced rebuild; no negative Holdings; no cash/M5/provider fallback/CRYPTO/AI leakage; backend authority; real synthetic-stack E2E independent of provider uptime; clean migrations/sqlc/contract drift; current security/audit and seven CI gates.

| Planning requirement                             | Coverage                                                         |
| ------------------------------------------------ | ---------------------------------------------------------------- |
| M3 closure and activation proven                 | §§1–2, verified PR #77/#78 SHAs                                  |
| COST_BASIS/DECIMAL/currency policies             | §§3,5–6; explicit blocking edge decisions, no invented semantics |
| PRICE_SELECTION, freshness/calendar              | §§9–10                                                           |
| M4/M5, cash, realized output and lot decisions   | §§3,5–6                                                          |
| Rebuild/correction/backdating/version/provenance | §§5–8                                                            |
| Observation identity and provider gaps           | §§4,9–11                                                         |
| Credential/retention/attribution boundaries      | §§4,11–12                                                        |
| ADR-023 receiver gate, no fake handoff           | §§4,8                                                            |
| Public contract, DB/module/sqlc ownership        | §§13–14                                                          |
| Tests/E2E independence/observability             | §16                                                              |
| Security expiry risk                             | §16; no exception change                                         |
| Exact downstream graph and DoD                   | §§17–18                                                          |
| No implementation in planning                    | Two documentation files only; queue unchanged                    |

Coverage means the plan specifies responsibility, boundary and gate, **not** that unresolved provider or financial evidence is verified. Ordinary FIFO equations, lot model and conservative UI boundary are proposed closed choices for this plan's review; the explicit §4 prerequisites must be resolved before contract/database/implementation. If the maintainer requires those financial edge choices closed in this planning PR instead, keep this PR Draft and obtain that decision without inventing one.

## 19. Planning PR verification and next permitted action

Run git diff --check; pnpm format:check/lint/typecheck/test/contract:check;
pnpm agent:test/agent:validate/agent:next; pnpm security:audit:test/security:audit;
go vet ./...; go test ./...; module-boundary checks. Resolver must remain
M4-PLAN-001/in_progress/implement_or_diagnose. Record exact actual results and
current-head workflow/job evidence in the Draft PR, not historical M3 results.

No generated/runtime changes are expected. Only this file and docs/README.md
may change. All seven checks and GitGuardian must pass; behind main and unresolved
review threads must be zero. Keep maintainer final review unchecked.

Next: maintainer reviews the proposed plan, especially §4 prerequisites and
§5 compatibility examples. After approval and merge, separately review and
authorize the control-plane graph/sourceOfTruth transition to M4-GATE-001.
Do not begin implementation, approve the receiver, activate jobs/publication,
change secrets, extend the exception, or begin M5 automatically.
