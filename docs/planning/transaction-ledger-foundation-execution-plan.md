# Transaction Ledger Foundation — Execution Plan

**Status:** Proposed
**Version:** `M3-TRANSACTION-LEDGER-PLAN-v1`
**Milestone:** M3 — Transaction Ledger
**Planning task:** `M3-PLAN-001`
**Planning branch:** `codex/m3-transaction-ledger-plan`
**Base:** protected `main` at `a294cc4347b6141491070946d5da73fe20c71cd1` (PR #45 / M2-VERIFY-001 merge)
**Depends on:** M0, M1, and M2 closed; the Decision Closure Specification; ADR-002, ADR-003, ADR-004, ADR-008, ADR-009, ADR-010, ADR-012; and ADR-013.
**Approval state:** Proposed for review. No M3 implementation begins until this plan is approved and merged.

This document is an implementation plan and M3 decision closure only. It adds
no OpenAPI operations, migrations, sqlc queries, backend code, frontend UI,
worker behavior, E2E tests, configuration, or M4 behavior.

## 1. Objective and success criteria

M3 establishes the immutable Transaction Ledger as the authoritative financial
record for a Portfolio. It records user financial facts reliably, enforces the
approved long-only validity rule, maintains an auditable correction trail, and
leaves a durable outbox seam for later projections.

M3 is complete only when an authenticated owner can record an eligible,
effective Transaction exactly once despite safe retries; retrieve deterministic
ledger history; and correct a record through an atomic reversal and replacement
workflow. Audit evidence and a versioned outbox event must commit in the same
database transaction as every accepted financial write.

M3 explicitly does **not** deliver a holding, lot, cost-basis, cash, price,
valuation, allocation, or dashboard projection. The M3 ledger-validity replay
is transient validation only; it is not a persisted Holding or FIFO projection.

## 2. Sources, precedence, and current repository assessment

### 2.1 Precedence

1. [Decision Closure Specification](decision-closure-specification.md),
   including `COST_BASIS-v1`, `DECIMAL-v1`, and `CURRENCY_SCOPE-v1`.
2. Approved ADRs, especially ADR-002, ADR-003, ADR-004, ADR-008, ADR-009,
   ADR-010, ADR-012, and [ADR-013](../adr/ADR-013-solo-maintainer-merge-governance.md).
3. M3 decisions expressly closed by this plan.
4. [Planning Baseline](planning-baseline.md).
5. M2 completion evidence and architecture documents.
6. Current repository conventions.

If a later implementation discovers a conflict with a source above, it must
stop the affected task, document the conflict and consequence, and obtain a
new decision. It must not hide a policy choice in a handler, migration, or UI.

### 2.2 Sources reviewed

- Planning Baseline and Decision Closure Specification.
- [M2 Portfolio & Asset plan](portfolio-asset-foundation-execution-plan.md),
  [acceptance matrix](../engineering/m2-portfolio-asset-acceptance-matrix.md),
  and [completion report](../engineering/m2-portfolio-asset-completion-report.md).
- [Module boundaries](../architecture/module-boundaries.md),
  [repository structure](../architecture/repository-structure.md), and
  [Portfolio and Asset database ownership](../architecture/portfolio-asset-database.md).
- Current OpenAPI source, `sqlc.yaml`, migrations 00001–00003, query groups,
  platform HTTP/database/worker packages, Portfolio and Asset modules,
  Next.js feature conventions, real HTTPS Playwright conventions, and the
  seven-job CI workflow.

### 2.3 Repository reality and reuse seams

| Area          | Current state                                                                                                                            | M3 direction                                                                                                                   |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| API contract  | `packages/api-contracts/openapi/v1.yaml` is OpenAPI-first; generated TypeScript is committed.                                            | M3-CONTRACT-001 extends this source before runtime work.                                                                       |
| Database/sqlc | Four targets exist: Platform, Identity, Portfolio, Asset. There is no Transaction target.                                                | Add a Transaction-owned target and retain Platform ownership of shared audit/outbox queries.                                   |
| Transactions  | No Transaction module, tables, idempotency store, or history route exists.                                                               | Introduce `backend/internal/transaction` only in approved downstream tasks.                                                    |
| Ownership     | Portfolio queries are owner scoped; Asset catalog is global and read-only.                                                               | Transaction application receives a principal and proves owned Portfolio access; it reads Asset only through a public boundary. |
| Audit         | Platform `audit_logs` is append-only but currently constrained to Authentication actions and has no Portfolio/Transaction target fields. | Extend this one Platform audit mechanism; do not create a second audit subsystem.                                              |
| Outbox/worker | Planning Baseline defines an outbox, but the repository has no outbox tables or delivery primitive. Worker is heartbeat-only.            | M3 adds a named Platform outbox capability and worker delivery task before using events.                                       |
| HTTP          | Platform owns `/api/v1`, correlation IDs, errors, and generic route registration.                                                        | Transaction transport mounts through the existing registrar seam.                                                              |
| Frontend      | Next.js feature folders use generated contracts, React Query, React Hook Form, Zod, accessible controls, and real HTTPS E2E.             | Add a Transaction feature only after contract, backend transport, and approved dependencies merge.                             |
| CI/governance | Seven required jobs and ADR-013 PR governance are active.                                                                                | Every M3 PR follows ADR-013; generated-output drift paths must include the new target.                                         |

### 2.4 Price-provider permission gate

The required repository search found no approved primary US price provider, no
contractual/display-permission evidence, and no legal record. The Decision
Closure Specification requires that evidence **before transaction work begins**.
Planning may proceed, but M3 implementation is gated.

`M3-GATE-001` must obtain a written product/legal confirmation identifying the
approved provider, the intended users, the permitted official-close retrieval
and display rights, applicable attribution/retention restrictions, and the
evidence location. The Product/Legal owner must approve it. No provider adapter
or price observation is created in M3. `M3-CONTRACT-001` is blocked until this
gate is complete because a transaction contract would otherwise start work
contrary to the approved implementation gate.

## 3. Scope and explicit non-goals

### In scope

- Immutable user ledger records for `BUY`, `SELL`, `DIVIDEND`, `DEPOSIT`,
  `WITHDRAWAL`, and `FEE`.
- USD-only, approved US-listed Equity and ETF financial entries.
- Owner-scoped create, list, retrieve, and correction operations.
- Deterministic effective-time ordering, transient ledger-validity replay,
  idempotency, audit, transactional outbox, and at-least-once delivery seam.
- Transaction entry, history, and correction UI with real HTTPS E2E evidence.

### Explicit non-goals

M3 must not implement price providers or observations, market calendars,
holdings/lots/cost basis, cash projection or cash UI, valuation, allocation,
dashboard, alerts, documents, AI, news, reviews, crypto trading, short sales,
margin, derivatives, corporate actions, tax-lot selection, FX, generic
transaction editing/deleting, or ordinary-user adjustments/reversals.

## 4. Closed M3 financial policy

### 4.1 Public and reserved transaction kinds

| Classification           | Kinds                                                     | Rule                                                                                                                                                                   |
| ------------------------ | --------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Public user-creatable    | `BUY`, `SELL`, `DIVIDEND`, `DEPOSIT`, `WITHDRAWAL`, `FEE` | Accepted only through the approved create or correction-replacement command.                                                                                           |
| Internal only            | `REVERSAL`                                                | Created exclusively by a correction transaction; no public create route or UI.                                                                                         |
| Reserved/future governed | `ADJUSTMENT`                                              | Not accepted by M3 API, not displayed as an entry option, and not created by ordinary users. A later ADR/policy must define reason, authorization, and audit approval. |

No unversioned or implicit kind is valid. A public unknown kind is rejected
without exposing persistence details.

### 4.2 Asset eligibility compatibility rule

The M2 catalog is broader than financial eligibility. An Asset is eligible for
M3 asset-backed financial records only when all backend-checked conditions hold:

1. it exists in the canonical catalog;
2. `assetType` is `EQUITY` or `ETF`;
3. `currency` is `USD`; and
4. its canonical normalized exchange is one of the M3 US-listing v1 venues:
   `NYSE`, `NASDAQ`, `NYSEARCA`, or `AMEX`.

This intentionally narrow venue allowlist is a M3 transaction policy, not a
change to catalog search support. It gives the current M2 schema a deterministic
US-listing discriminator without adding an unapproved provider or inferring
that every catalog item is tradable. A future venue requires a versioned policy
change and eligibility tests. `CRYPTO`, including the canonical `CRYPTO`
exchange namespace, is catalog-only and is rejected for `BUY`, `SELL`, and
`DIVIDEND`. The frontend may filter choices but is never authoritative.

### 4.3 Lifecycle and immutability

M3 has **no DRAFT state**. A successful create produces one `EFFECTIVE`,
immutable public ledger record. There is no generic `PATCH`, `DELETE`, edit,
cancel, restore, or mutable financial-status operation. Database design,
repository methods, application operations, contracts, and UI must all retain
that rule.

`REVERSAL` is an immutable internal record. It is visible as a relationship in
history but cannot be directly created by a user. Original, reversal, and
replacement facts are never physically deleted.

### 4.4 Field matrix and authoritative input model

All financial values are decimal strings. `portfolioId` is the route scope and
never appears as a client ownership assertion. `effectiveAt`, `note`, and
`externalReference` are common fields; note and external reference are optional
non-authoritative annotations. An absent fee means exactly zero; a supplied fee
must be non-negative.

| Type                | Asset                              | Quantity       | Unit price     | Amount         | Fee             | Currency       | Effective time                        | Note / external reference          |
| ------------------- | ---------------------------------- | -------------- | -------------- | -------------- | --------------- | -------------- | ------------------------------------- | ---------------------------------- |
| `BUY`               | Required, eligible                 | Required `> 0` | Required `> 0` | Forbidden      | Optional `>= 0` | Required `USD` | Required                              | Optional / optional                |
| `SELL`              | Required, eligible                 | Required `> 0` | Required `> 0` | Forbidden      | Optional `>= 0` | Required `USD` | Required                              | Optional / optional                |
| `DIVIDEND`          | Required, eligible                 | Forbidden      | Forbidden      | Required `> 0` | Forbidden       | Required `USD` | Required                              | Optional / optional                |
| `DEPOSIT`           | Forbidden                          | Forbidden      | Forbidden      | Required `> 0` | Forbidden       | Required `USD` | Required                              | Optional / optional                |
| `WITHDRAWAL`        | Forbidden                          | Forbidden      | Forbidden      | Required `> 0` | Forbidden       | Required `USD` | Required                              | Optional / optional                |
| `FEE`               | Forbidden                          | Forbidden      | Forbidden      | Required `> 0` | Forbidden       | Required `USD` | Required                              | Optional / optional                |
| internal `REVERSAL` | Copied from original where present | Copied         | Copied         | Copied         | Copied          | Copied         | Server-set to original effective time | Server-generated relationship only |

`DIVIDEND` is asset-referenced so the immutable ledger retains the corporate
cash-event context; it changes no asset quantity. Reinvestment is two records:
`DIVIDEND` then `BUY`. A standalone `FEE` is a portfolio cash event and has no
asset. Trade fees belong to their `BUY`/`SELL` record; M3 never creates both a
trade-attached fee and a separate fee for the same submission.

For `BUY` and `SELL`, only `quantity`, `unitPrice`, and `fee` are financial
inputs. Gross is deterministically `quantity × unitPrice`; net is gross plus
fee for a buy and gross minus fee for a sell. Gross/net may be returned as
derived views later, but are not accepted or persisted as competing truths.
Cash-event `amount` is a strictly positive magnitude: type determines its
direction (`DIVIDEND`/`DEPOSIT` increase; `WITHDRAWAL`/`FEE` decrease). Negative
or zero input cannot invert a transaction kind.

### 4.5 Decimal, currency, and time rules

`DECIMAL-v1` applies without alteration: quantities, unit prices, fees, and
cash amounts are decimal strings with at most 12 fractional digits; no
float32/float64 or client financial arithmetic is authoritative; database
columns support at least 38 significant digits; deterministic calculations use
round-half-to-even at the designated output boundary. All M3 currency-bearing
facts explicitly carry `USD`; non-USD input is unsupported.

`effectiveAt` is a required RFC 3339 UTC (`Z`) instant with 0–6 fractional
digits. PostgreSQL microsecond precision is the persisted contract boundary.
Past timestamps are allowed. A future effective timestamp is rejected: M3
records financial facts already effective, not scheduled or draft work. A
correction reversal receives the original effective timestamp, while its
replacement uses the replacement command's validated effective timestamp.

### 4.6 Ordering and ledger-validity replay

Ledger order is ascending `(effective_at, portfolio_sequence)` for validation.
`portfolio_sequence` is a server-assigned, immutable, strictly monotonic,
portfolio-local positive sequence. Clients cannot set it. M3-DB-001 must choose
and prove a PostgreSQL serialization mechanism; `SELECT MAX(sequence) + 1`
without a lock/serialization guarantee is forbidden. A correction allocates new
sequences for reversal then replacement in the same transaction; the original
keeps its original sequence.

History defaults to newest effective record first:
`effectiveAt DESC, portfolioSequence DESC, transactionId DESC`. Its cursor is
opaque and represents this full stable ordering. Replay always uses the inverse
ascending order.

The Transaction domain owns a **transient Ledger Validity Replay**. It replays
only immutable facts needed to ensure, for every affected `(portfolio, asset)`,
that `BUY` adds quantity, `SELL` subtracts quantity, and an internal reversal
negates the original's quantity effect. A sell and any backdated create or
correction are rejected when ordered replay would produce a negative quantity
at any point. The replay can consider all affected asset streams for a
correction, but stores no lots, remaining quantity, holding, cost basis, gain,
cash balance, or valuation. M4 owns persisted FIFO lots and projections.

M3 records cash-event facts but imposes **no cash-overdraft/sufficient-cash
rule**: no approved policy grants one. It may validate cash-field shape and USD
only. A later cash-projection policy may introduce cash availability rules.

### 4.7 Correction and reversal policy

A correction is one atomic command against an owned original effective public
Transaction. It authorizes the owner, validates the original and complete
replacement, allocates sequences, validates replay, inserts an internal
`REVERSAL` plus a normal replacement, appends audit records and outbox events,
persists idempotency result, then commits once.

A reversal has an immutable `reversal_of_transaction_id` relationship and
copies the original financial dimensions; its replay sign is negative rather
than storing negative quantity/money. A replacement stores a
`correction_of_transaction_id` relationship to the original. All three records
share a generated correction identifier for history grouping. The original is
therefore preserved and logically neutralized—not changed or deleted.

One public effective transaction can have at most one direct reversal. A
replacement is itself a normal public fact and may later be corrected, creating
a traceable chain. A previously reversed original cannot receive another direct
reversal. The correction's reversal and replacement are inseparable: partial
corrections are forbidden.

### 4.8 Idempotency and semantic fingerprint

Every create and correction requires `Idempotency-Key`. Its scope is
`(portfolio_id, command_scope, key)`, where command scope is separately
versioned as `transaction.create.v1` or `transaction.correct.v1`. The same key
and canonical semantic fingerprint returns the previously committed response;
the same scope/key with a different fingerprint is a deterministic conflict.
Concurrent matching requests yield at most one financial mutation. The ledger
records, idempotency record, audit evidence, and outbox events commit in one
database transaction, so no committed transaction lacks its idempotency result.

The v1 fingerprint is SHA-256 over a versioned canonical representation of the
command's semantic fields: command scope, target/original ID when applicable,
kind, canonical Asset ID, canonical decimal values, `USD`, UTC microsecond
effective time, optional note, and optional external reference. It excludes JSON
member order, JSON formatting, header whitespace, Authorization, correlation
ID, client IP, and server-generated IDs/sequences/timestamps. Input text is
validated and normalized according to its field policy before fingerprinting;
the implementation must not accept a semantically distinct request as a replay.

Idempotency records are retained for 365 days from commit. Before expiry, all
replays preserve the result; after expiry the key may be reused as a new command
and is not a durable duplicate-detection substitute. Cleanup is a named worker
responsibility with tests. A retry after an uncertain response reuses the same
key. A user who intentionally starts a new command receives a new key.

### 4.9 Ownership, snapshots, and security

Every Transaction belongs to a Portfolio. The application derives the principal
from M1 authentication, reads the Portfolio through an owner-scoped public
interface, and returns the same ownership-safe absence behavior for missing,
cross-owner, or unrepresentable opaque IDs. A client never supplies an owner
ID as authorization proof. Asset lookup is global but only M3 eligibility
passes are acceptable for asset-backed records.

Transactions reference immutable Asset IDs and snapshot only the financial
context needed to reproduce their validity scope: `asset_id`, canonical asset
type, canonical exchange, and currency at acceptance. They do not duplicate
mutable display name/symbol metadata. The snapshot protects later replay from a
catalog metadata edit while Asset remains the display authority.

M1 HTTPS, authentication, error envelope, correlation ID, logging redaction,
and no-token-persistence policies remain unchanged. Financial request bodies,
Bearer tokens, refresh tokens, and secrets must never be logged. Backend
authorization and eligibility validation are authoritative.

## 5. Audit, outbox, and worker policy

### 5.1 Audit

Platform owns append-only audit persistence. M3 extends the existing Platform
audit action constraints and public adapter rather than creating a parallel
ledger audit table. Its allowlisted M3 targets are Portfolio ID, transaction
ID, correction ID, actor ID, action, result, timestamp, and correlation ID.
It does not store full financial request bodies, credentials, tokens, secrets,
or arbitrary metadata.

Required actions include create success/failure, idempotent replay,
idempotency conflict, correction initiated/completed/rejected, reversal
created, replacement created, and ownership/authorization rejection. Expected
invalid validation failures may be logged as safe aggregate outcomes but must
not disclose a cross-owner record or body values.

### 5.2 Transactional outbox

Platform owns the new outbox persistence and delivery primitive. An outbox row
is append-only at creation and includes immutable event ID, event type and
version, aggregate type/ID, Portfolio ID, transaction/correction references,
occurred-at, correlation ID, minimal versioned payload, publication state,
attempt count, and safe failure metadata. The transaction write and outbox row
are committed atomically.

M3 event families are:

- `transaction.recorded.v1` for a public accepted record; and
- `transaction.corrected.v1` for one correction, referencing original,
  reversal, and replacement IDs.

Payloads contain stable references only; future consumers re-read the ledger
and never rely on a duplicated complete financial payload. Delivery is
at-least-once. Consumers must deduplicate by consumer name and event ID, in
line with the existing Worker Event Standard. M3 provides delivery and safe
observability but does not implement M4 projection consumers.

## 6. Architecture and contract direction

### 6.1 Module boundaries

`backend/internal/transaction` will use the same layers as existing modules:

- `domain`: financial types, immutable record, correction relationship,
  decimal boundary, ordering, and transient replay; no Fiber, pgx, sqlc, or
  infrastructure imports.
- `application`: owner-scoped operations and ports for Portfolio access, Asset
  eligibility lookup, Transaction persistence/transactor, audit, outbox, clock,
  and IDs; no Fiber or sqlc.
- `infrastructure/database`: Transaction-owned repository and sqlc mapping;
  no direct Portfolio/Asset table writes.
- `transport/http`: DTO validation, idempotency/header extraction, safe error
  mapping, and route mounting; no sqlc imports.
- `composition`: only place that assembles Transaction adapters.

Portfolio and Asset expose narrow public application/domain interfaces for
owner access and canonical eligibility lookup. Platform remains technical: its
named audit/outbox packages cannot contain Transaction business rules. The
module-boundary script must be extended for Transaction and never weakened.

### 6.2 Public-contract inventory

After the planning gate, M3-CONTRACT-001 will add only the reviewed resource
area below; it must not change M1/M2 semantics:

| Operation direction                                                              | Purpose                                                                                                           |
| -------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `POST /api/v1/portfolios/{portfolioId}/transactions`                             | Create one immutable public Transaction; requires Bearer auth and `Idempotency-Key`.                              |
| `GET /api/v1/portfolios/{portfolioId}/transactions`                              | Owner-scoped deterministic cursor history, filters limited to approved kind/effective-time/correction visibility. |
| `GET /api/v1/portfolios/{portfolioId}/transactions/{transactionId}`              | Retrieve one owner-scoped immutable record and correction relationships.                                          |
| `POST /api/v1/portfolios/{portfolioId}/transactions/{transactionId}/corrections` | Atomically create reversal plus replacement; requires a separate idempotency scope/key.                           |

Contract work owns exact paths, schemas, status and error codes, cursor codec,
request-size bounds, idempotency-key grammar, and generated TypeScript. It must
reuse the standard error envelope and correlation header. It must declare no
`PATCH`/`DELETE`, public `ADJUSTMENT`, public `REVERSAL`, financial JSON numbers,
or client owner fields.

Semantic errors to freeze include ownership-safe Portfolio/Transaction absence,
unsupported type/Asset/currency, invalid field combination/decimal/effective
time, insufficient ordered asset quantity, invalid backdated ledger,
idempotency conflict, and already-corrected target. No database error reaches
the caller.

## 7. Database and persistence direction

M3-DB-001 will introduce only the authoritative `transactions` and
`transaction_idempotency` records plus required Platform audit/outbox changes.
It must use new forward Goose migrations and matching down migrations; it must
not amend M1/M2 migrations. Transaction IDs are server-generated opaque UUIDs.

`transactions` conceptually requires Portfolio FK, optional Asset FK according
to the matrix, public/internal kind, snapshot type/exchange/currency, decimal
inputs, effective timestamp, immutable portfolio sequence, creation metadata,
correction ID, reversal/original relationships, and constraints that make
forbidden field combinations impossible where practical. It requires indexes
for owner-authorized portfolio history/replay ordering, asset-stream replay,
correction lookup, and direct reversal uniqueness. No Holding, lot, price,
valuation, or aggregate financial columns appear.

`transaction_idempotency` requires Portfolio ID, command scope, opaque key,
fingerprint digest, resulting primary transaction ID and correction references
as appropriate, committed response identity, created/expiry timestamps, and a
unique `(portfolio_id, command_scope, key)` authority. It must share the write
transaction with its result. M3-DB-001 selects a demonstrated PostgreSQL
serialization method for both idempotency and sequence allocation; concurrent
writers cannot receive duplicate sequences or create duplicate financial facts.

Audit and outbox remain Platform-owned sqlc groups. The new Transaction sqlc
target must be deterministic and private to Transaction infrastructure; CI's
generation drift command must cover it. No module writes another module's
tables directly.

## 8. Frontend direction

M3 frontend work adds `/app/portfolios/[portfolioId]/transactions` after the
contract and backend routes merge. It includes ledger history, loading/empty/
error/retry states, type-dependent entry forms, approved canonical Asset search,
UTC effective-time entry, decimal usability validation, review-before-submit,
accepted-result navigation, and visible original/reversal/replacement links.

An effective record has **Correct**, never Edit/Delete. Correction displays the
original, captures a complete replacement, uses its own one-command
Idempotency-Key, and tells the user that the original remains in history. The
frontend retains a command key only in component memory for a controlled retry;
it does not persist commands, access tokens, or refresh cookies in browser
storage. It never predicts holdings or performs authoritative financial math.

## 9. Test and operational strategy

| Layer                | Required evidence                                                                                                                                                                                                                      |
| -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Domain               | Kind/field matrix; decimal/currency/time rules; immutable relations; ordering; transient replay; backdating; sell insufficiency; reversal chains.                                                                                      |
| Application          | Principal ownership; create; same-key replay; conflicting key; concurrent command behavior; correction atomic intent; Asset/CRYPTO rejection; audit/outbox invocation.                                                                 |
| Database integration | Empty/current-main/down-up migrations; sequence and idempotency concurrency using separate pools; constraints; owner scope; ordering/cursor support; correction uniqueness; audit/outbox atomic rollback.                              |
| Contract             | Header/key rules; decimal strings; field matrix; public operation inventory; standard errors; cursor behavior; generated output; no patch/delete/adjustment/reversal.                                                                  |
| Frontend             | Type-specific fields; input usability; idempotent controlled retry; immutable history; correction UX; no financial projection metrics or token persistence.                                                                            |
| Real HTTPS E2E       | Chromium → Caddy → Next.js → Go API → PostgreSQL; registration; Portfolio; create/retry/history/correction; cross-owner isolation; rejected CRYPTO action; no edit/delete/derived finance UI. No API/Auth mocks or external providers. |

All M3 PRs run applicable local checks. The final verification PR must pass
the seven protected remote jobs: `frontend`, `backend`,
`contracts-and-generation`, `database-integration`, `browser-e2e`,
`compose-smoke`, and `secrets`; its current head must be up to date and all
review conversations resolved under ADR-013.

## 10. Ordered task sequence

| Task              | Depends on                               | Scope and acceptance criteria                                                                                                                         | Complexity | Gate / risk                                                                |
| ----------------- | ---------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- | -------------------------------------------------------------------------- |
| `M3-PLAN-001`     | M2 closed                                | This plan, decision closure, task graph, and docs index only.                                                                                         | Medium     | Must merge before all later tasks.                                         |
| `M3-GATE-001`     | `M3-PLAN-001` merged                     | Record Product/Legal confirmation of the approved primary US official-close provider's permitted retrieval/display rights and restrictions.           | Small      | **Blocking:** absent evidence blocks M3 implementation.                    |
| `M3-CONTRACT-001` | Plan merged; provider gate approved      | Freeze the four operation contracts, matrix, errors, idempotency headers, cursor, contract tests, generated types.                                    | Medium     | Zero unresolved contract decisions; no runtime work.                       |
| `M3-PLATFORM-001` | Contract merged                          | Extend Platform audit allowlist and create versioned transactional outbox/consumer-dedup persistence and interfaces.                                  | Large      | Existing audit schema and heartbeat-only worker are insufficient.          |
| `M3-DB-001`       | Contract and Platform persistence merged | Transaction migrations, sqlc target/queries, sequence/idempotency/correction constraints, migration/query integration tests.                          | Large      | Must prove cross-connection concurrency and no unsafe sequence allocation. |
| `M3-BE-001`       | DB merged                                | Transaction domain, decimal adapters, field rules, transient replay, immutable correction model, focused unit tests.                                  | Large      | Must not persist projections or business rules in SQL.                     |
| `M3-BE-002`       | BE-001 and Platform merged               | Application operations, owner/Asset public ports, PostgreSQL transactor/repositories, idempotency, audit/outbox atomic write, worker outbox delivery. | Large      | Exactly-once financial mutation and at-least-once event delivery.          |
| `M3-BE-003`       | BE-002 merged                            | HTTP transport, DTO/error mapping, route composition, API/integration security tests.                                                                 | Medium     | No route mounts without all security dependencies.                         |
| `M3-FE-001`       | BE-003 merged                            | Ledger history and create-entry flow using generated contract types, query keys, accessible forms, memory-only retry key.                             | Large      | No derived financial UI or client calculations.                            |
| `M3-FE-002`       | FE-001 merged                            | Correction/history relationship UI, retry/error UX, focused frontend tests.                                                                           | Medium     | No Edit/Delete semantics.                                                  |
| `M3-E2E-001`      | FE-002 merged                            | Real-stack ledger critical flow and fixtures only.                                                                                                    | Medium     | No endpoint mocks, no external market provider.                            |
| `M3-VERIFY-001`   | All preceding tasks merged               | Acceptance matrix, completion report, full local/remote evidence, scope review, closure recommendation.                                               | Large      | Only this task may propose M3 closure.                                     |

## 11. Dependency graph and delivery governance

```text
M3-PLAN-001
  -> M3-GATE-001 (provider permission evidence)
  -> M3-CONTRACT-001
  -> M3-PLATFORM-001
  -> M3-DB-001
  -> M3-BE-001
  -> M3-BE-002
  -> M3-BE-003
  -> M3-FE-001
  -> M3-FE-002
  -> M3-E2E-001
  -> M3-VERIFY-001
```

The arrows are merge gates, not merely technical suggestions. Each focused PR
follows ADR-013: a PR to protected `main`, self-review checklist, seven passing
checks without bypass, up-to-date branch, resolved conversations, and no force
push/deletion. No parallel branch may use an unmerged dependency as if it were
approved.

## 12. Scope guards and stop conditions

Stop the affected task and obtain a documented decision for any contradiction
in public kinds, asset eligibility, lifecycle, field matrix, gross/net truth,
ordering, backdating, replay, correction, idempotency, audit, outbox, decimal,
currency, ownership, or task dependencies. Do not defer those choices to a
handler or migration.

M3-CONTRACT-001 begins only after M3-GATE-001 and this plan are approved. M3
implementation never adds M4 projections, price/provider behavior, or crypto
financial processing as a workaround.

## 13. M3 Definition of Done and closure criteria

M3 can close only when evidence proves all of the following:

- public kinds are limited to the approved set; `ADJUSTMENT` and direct
  `REVERSAL` are unavailable to ordinary users;
- only eligible USD US-listed Equity/ETF Assets are accepted; CRYPTO financial
  actions are rejected server-side;
- effective rows are immutable, idempotent, ordered safely under concurrency,
  and backdated replay never permits a negative asset quantity;
- correction atomically records traceable reversal and replacement while
  retaining original history;
- principal-scoped ownership, decimal/currency policy, audit, outbox, worker
  delivery, deterministic cursor history, and standard error/correlation
  behavior all pass;
- frontend remains non-authoritative and omits financial projections;
- real HTTPS E2E, migrations/sqlc drift, security scanning, and every required
  remote CI job pass; and
- `M3-VERIFY-001` acceptance matrix and completion report match repository
  reality and its verification PR is reviewed and merged.

Only `M3-VERIFY-001` may state `M3 Status: Closed`.

## 14. Self-review coverage matrix

| Required area                                        | Status  | Evidence in this plan                                                            |
| ---------------------------------------------------- | ------- | -------------------------------------------------------------------------------- |
| M2 closed dependency and source precedence           | COVERED | §§2.1–2.2                                                                        |
| M3 objective and ledger authority                    | COVERED | §1                                                                               |
| Public transaction types / ADJUSTMENT restriction    | COVERED | §4.1                                                                             |
| Equity/ETF eligibility / CRYPTO prohibition          | COVERED | §4.2                                                                             |
| Draft decision / effective immutability              | COVERED | §4.3                                                                             |
| Field matrix / Dividend and standalone Fee semantics | COVERED | §4.4                                                                             |
| Decimal, currency, gross/net truth                   | COVERED | §§4.4–4.5                                                                        |
| Ordering, sequence, concurrency, backdating          | COVERED | §4.6 and §7                                                                      |
| SELL validity / minimal replay / cash boundary       | COVERED | §4.6                                                                             |
| Correction, reversal representation, cardinality     | COVERED | §4.7                                                                             |
| Idempotency scope, fingerprint, retry/retention      | COVERED | §4.8                                                                             |
| Ownership and Asset reference/snapshot               | COVERED | §4.9                                                                             |
| Audit                                                | COVERED | §5.1                                                                             |
| Outbox and worker/event delivery                     | COVERED | §5.2 and §10                                                                     |
| Contract, database, and frontend direction           | COVERED | §§6–8                                                                            |
| Real HTTPS E2E and CI                                | COVERED | §9                                                                               |
| Price-provider permission gate                       | COVERED | §2.4 and `M3-GATE-001`; implementation is intentionally blocked pending evidence |
| Ordered tasks and dependency graph                   | COVERED | §§10–11                                                                          |
| Non-goals, Definition of Done, closure semantics     | COVERED | §§3, 12–13                                                                       |
| No implementation changes                            | COVERED | This planning-only change set                                                    |

**Coverage totals:** COVERED 21; PARTIAL 0; MISSING 0; BLOCKED 0. The provider
permission evidence is a downstream implementation gate, not an unresolved
planning decision.

## 15. Planning-task completion report template

`M3-VERIFY-001` will record implementation PRs/commits, changed modules,
migrations, contracts, generated output, audit/outbox delivery evidence,
commands and results, the M3 acceptance matrix, deviations, risks, remaining
limitations, M3 status, and the recommended next action. It must explicitly
confirm no M4 projection, provider, or crypto-financial functionality slipped
into M3.

## 16. Recommended next step

Review and approve `M3-TRANSACTION-LEDGER-PLAN-v1`. After merge, complete
`M3-GATE-001` by recording the required provider-permission evidence; only then
start `M3-CONTRACT-001`. Do not implement M3 or begin M4 automatically.
