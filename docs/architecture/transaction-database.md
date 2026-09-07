# Transaction Ledger Database Foundation

Task: `M3-DB-001`. Base: `e5fbffc903efe170abbf313cf87180186ef13d63` (PR #50).
Sources: [Decision Closure](../planning/decision-closure-specification.md),
[M3 plan](../planning/transaction-ledger-foundation-execution-plan.md), and
the frozen [M3 OpenAPI](../../packages/api-contracts/openapi/v1.yaml).

## Ownership and scope

Migration `00005_m3_transaction_ledger.sql` creates only Transaction-owned tables:

| Table                             | Responsibility                                                           |
| --------------------------------- | ------------------------------------------------------------------------ |
| `transactions`                    | Authoritative immutable accepted facts, including internal reversals     |
| `transaction_corrections`         | Authoritative immutable original/reversal/replacement group              |
| `transaction_portfolio_sequences` | Portfolio-local write serialization and ordering counter                 |
| `transaction_idempotency`         | Completed command identity/result references; expiring operational state |

Portfolio ownership is derived through Portfolio, not duplicated as an owner
column. `created_by_user_id` is immutable provenance, never authorization proof.
The caller must prove ownership through Portfolio's public boundary before using
any scoped query. Asset references and acceptance snapshots do not change M2
catalog eligibility. All parent deletion behavior is restrictive.

Only Transaction infrastructure imports the fifth sqlc target at
`backend/internal/transaction/infrastructure/database/sqlcgen`. Generated rows
have no JSON tags and must not become HTTP DTOs or domain/application contracts.
No domain, application, repository adapter, HTTP route, worker delivery loop,
provider, price, holding, FIFO, valuation, or M4 behavior is implemented here.
Platform audit/outbox remain unchanged and Platform-owned.

## Financial storage boundary

Unconstrained PostgreSQL `NUMERIC` stores quantity, unit price, fee, and amount.
This supports at least 38 significant digits without inventing a public maximum
or rounding values to fit a typmod. Each non-null value must be finite, at most
12 fractional digits, and strictly positive except fee, which permits zero.
Database implementation limits are not new API validation limits. Later request
validation enforces the frozen decimal-string grammar and command body bound.

BUY/SELL require an eligible Asset snapshot, quantity, unit price, and fee;
amount is null. The caller supplies zero for an omitted trade fee. There is no
global fee default. DIVIDEND requires Asset and amount; DEPOSIT/WITHDRAWAL/FEE
require amount with no Asset. REVERSAL permits those copied magnitude shapes,
but requires correction links and null note/external reference. Only the seven
approved stored kinds exist; no ADJUSTMENT, lifecycle/status, gross, or net column.

Snapshots are all present or all absent with the Asset: EQUITY/ETF, one of
NYSE/NASDAQ/NYSEARCA/AMEX, and USD. Application validation against the current
Asset, positive-position replay, future effective-time rejection, exact reversal
copy/equality, and correction target eligibility remain later Go responsibilities.
No SQL trigger, financial aggregation, or replay decision is introduced.

`timestamptz` stores finite microsecond instants; UTC-Z lexical precision is a
transport obligation. History bounds may be in the future. Optional text uses
nullable `text`: null is absent, empty is supplied empty. No trimming or Unicode
normalization occurs; bounds are 2,000 and 256 characters respectively.

## Immutable corrections and chains

An ordinary record has no creation links. A reversal records
`reversal_of_transaction_id` and `originating_correction_id`; a replacement
records `correction_of_transaction_id` and `originating_correction_id`.
The original is never updated to receive an outgoing correction identifier.
`GetDirectTransactionCorrection` derives its outgoing links from the group.

Composite foreign keys enforce Portfolio scope and matching original/group/
reversal/replacement identities in both directions. The circular foreign keys
are initially deferred, so all three new rows can be inserted in one transaction,
but neither partial groups nor orphan reversal/replacement rows can commit.
Unique direct-role indexes prevent two reversals or replacements of one original.
A group's replacement can be a later group's original, retaining both directions
of the chain. No UPDATE/DELETE fact or correction query exists. Privileged SQL
access is not an application mutation API; this task adds no trigger-based policy.

## Caller-owned atomic transaction and lock order

Later application composition must use one `pgx.Tx`, with this order:

1. Acquire `LockTransactionIdempotency` for Portfolio/scope/key.
2. Read the unexpired completed result; compare the fingerprint in Go.
3. Remove an expired exact key if necessary.
4. Allocate one sequence, or two for correction, using the stream-row UPSERT.
5. Read the relevant asset-ledger facts under that stream lock, then validate in Go.
6. Insert immutable facts/group, Platform audit/outbox, and completed idempotency.
7. Commit once. Return success only after commit.

Sequence UPSERT locks the Portfolio counter through caller commit/rollback. It
starts at one and reserves reversal before replacement. Rollback restores the
allocation; unique `(portfolio_id, portfolio_sequence)` is the final authority.
Neither `SELECT MAX+1` nor process locks are used. Overflow fails rather than
wrapping. Autocommit sequence allocation is not the supported write workflow.

Idempotency advisory-lock input is PostgreSQL `hashtextextended(..., 0)` over
`transaction_idempotency:v1:` + canonical UUID + `:` + decimal scope length +
`:` + scope + `:` + decimal key length + `:` + key. It is an internal signed
bigint, not a persisted fingerprint or external format. A hash collision only
over-serializes; the composite primary key remains decisive. All participating
writers must use the same namespace and PostgreSQL function, acquiring this
lock before the sequence lock. Both locks participate in the caller transaction.

## Completed idempotency and cleanup

Key grammar is `[A-Za-z0-9][A-Za-z0-9._~-]{15,127}`. Scopes are exactly
`transaction.create.v1` and `transaction.correct.v1`. Fingerprints are raw
32-byte SHA-256 digests generated later; no canonical command or response JSON
is stored. Create references one result; correction references its original,
complete group, and public replacement via matching composite foreign keys.
No pending reservation or incomplete committed result shape is representable.

The insert query uses the caller's final completion timestamp plus 8,760 hours
(365 days independent of DST). The caller supplies that timestamp immediately
before commit; PostgreSQL cannot know a future commit instant inside the write.
Lookup uses `expires_at > as_of`; cleanup uses `<=`. Before expiry replay returns
the committed identity; after expiry a key may be reused. The future named worker
owns bounded global cleanup with `FOR UPDATE SKIP LOCKED`. Exact-key cleanup runs
under the command advisory lock. Neither cleanup path deletes financial facts.

## Query/index inventory

| Queries                                                           | Supporting index / constraint                                                     |
| ----------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| Insert/read scoped fact                                           | Primary ID and scoped identity unique keys                                        |
| History with kind/time/visibility filters and full decoded cursor | `(portfolio_id, effective_at DESC, portfolio_sequence DESC, transaction_id DESC)` |
| Asset ledger replay, ascending, no calculations                   | `(portfolio_id, asset_id, effective_at, portfolio_sequence, transaction_id)`      |
| Insert/read group by ID or original; resolve outgoing links       | Correction primary key and direct-original uniqueness                             |
| Complete correction/chain integrity                               | Composite role identities and direct reversal/replacement unique indexes          |
| One/two sequence allocation                                       | Portfolio counter primary key                                                     |
| Idempotency lookup / exact expiry / completed insert              | `(portfolio_id, command_scope, idempotency_key)` primary key                      |
| Bounded expiry cleanup                                            | `(expires_at, portfolio_id, command_scope, idempotency_key)`                      |

The cursor decoder is not implemented here. History receives either the complete
decoded time/sequence/ID tuple or all nulls, uses exclusive tuple continuation,
inclusive time filters, and the contract page limit (default 50, max 100).
Public kind includes REVERSAL for reads; include-reversals defaults to true at
the later transport boundary. These indexes support history/replay only, not M4.

## Verification and rollback

Integration tests use fresh isolated schemas only in an explicitly named
`*_test` database; independent pools share that schema for lock tests. They run
the migration SQL transactionally and verify empty upgrade, v4 upgrade, empty
down/up, retained-state Down refusal, constraints, exact decimals, scoped queries,
chains, cursor order, index planner usability, idempotency, and concurrency.
CI also runs actual Goose empty/reset/v2/v4/v5/down/up against PostgreSQL 17.

Down takes exclusive table locks and fails before dropping anything when any
ledger, group, idempotency, or sequence state exists. An approved retention/archive
decision is required before a populated rollback; do not delete records simply
to force Down to pass. Migrations 00001–00004 remain unchanged.

M3-BE-001 has **not started**. This document describes persistence primitives,
not an active Transaction API or a completed M3 milestone.
