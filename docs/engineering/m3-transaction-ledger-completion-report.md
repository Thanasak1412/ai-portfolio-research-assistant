# M3 Transaction Ledger Completion Report

## Metadata

- Task: `M3-VERIFY-001`; milestone: M3 — Transaction Ledger.
- Verification date: 2026-10-04.
- Protected-main base: `1de0dc9a51f97fbeefb3c03920568b22236701e4` (merged queue-transition PR #76).
- Branch: `codex/m3-verify-001`.
- Plan: `M3-TRANSACTION-LEDGER-PLAN-v1`; Decision Closure remains authoritative.
- Final evidence surface: the Draft PR titled **docs(transaction): verify M3 ledger completion** on this branch. Its **Current-head evidence** section is authoritative for the final SHA, workflow/job IDs, GitGuardian, browser counts and review/freshness checks. A document cannot embed its own resulting commit SHA; every later push invalidates earlier current-head CI evidence.
- [Acceptance matrix](m3-transaction-ledger-acceptance-matrix.md): 45 implementation PASS, 0 FAIL, 0 BLOCKED, 1 NOT_APPLICABLE. This does not mark the milestone closed.

This task changes documentation only. It reconciles the obsolete PR #48 pending
merge wording without changing the approved provider, audience, contractual
interpretation, evidence or constraints. No production behavior, test, contract,
migration, generated output, agent policy or security exception is changed.

## Implementation Traceability

Live GitHub merged PR metadata/file inventories were compared with protected-main
first-parent merge ancestry. All eleven task merges below precede the verified
base. PR #76 independently reconciled the queue; the resolver now returns
`M3-VERIFY-001 / in_progress / implement_or_diagnose`.

| Task            | Merged PR                                                                      | Merge SHA                                  | Delivered scope / key evidence                                                                                             |
| --------------- | ------------------------------------------------------------------------------ | ------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| M3-PLAN-001     | [#47](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/47) | `3ed1d915a43365adfc3dfd515abd3ad55357be4c` | Execution plan, decisions, task graph and docs index                                                                       |
| M3-GATE-001     | [#48](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/48) | `0300f194b505b5dbec2af1a991438aead23a5785` | Approved provider-permission record and human/provider evidence references                                                 |
| M3-CONTRACT-001 | [#49](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/49) | `4005c1f46ce67e706d54306df60da918cf32720a` | Four operations, discriminated command matrix, errors/cursor/idempotency contract, generated TypeScript and contract tests |
| M3-PLATFORM-001 | [#50](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/50) | `e5fbffc903efe170abbf313cf87180186ef13d63` | Migration 00004, safe audit extension, stream-ordered outbox and consumer dedup primitives/tests                           |
| M3-DB-001       | [#52](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/52) | `672535c8ce78b98f78d888acc8e2e5e20285fd67` | Migration 00005, four Transaction tables, fifth sqlc target, migration/concurrency/atomicity tests and CI registration     |
| M3-BE-001       | [#57](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/57) | `a3c6e6e7473443dcd52fc54d312297a90b1f9a51` | Exact decimal/private immutable domain, correction chains, transient replay and unit tests                                 |
| M3-BE-002       | [#64](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/64) | `1c2adbd6fb9848b4b31b902c7c4fb8fd5b54d28f` | Owner/Asset public ports, atomic application/persistence, delivery engine and ADR-023 inactive composition                 |
| M3-BE-003       | [#69](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/69) | `7215427ac25ea6efcd2b5e7706b04901c28df4f8` | Strict HTTP transport, runtime routes, error/correlation mapping and real authenticated API tests                          |
| M3-FE-001       | [#71](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/71) | `3d4eab4307b64f84a3678dc7cb9217b2b9ebda6a` | Generated-type API adapter, history/filter/paging, entry/review, canonical Asset picker and memory retry keys              |
| M3-FE-002       | [#73](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/73) | `7c4a6e9d6a0f691b3e1ba69be642adb079ff5a82` | Complete correction, immutable relationships/chains, failure/retry UX and frontend tests                                   |
| M3-E2E-001      | [#75](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/75) | `d41dfce261dd541aae76e1a76836e8fd02d4a38d` | Real HTTPS critical lifecycle, synthetic test-only Asset fixtures, safe seed guard, existing CI integration and runbook    |

Supporting governance: ADR-022 was accepted through PR #60
(`dfda4b4b677e540a5392e80a2d181ba6e0a27c3f`); ADR-023 acceptance was recorded
through PR #62 (`15279c03e4f8141a370cf81c968981b68005b18e`). These precede
M3-BE-002. M2 navigation fixes were inherited from protected main, not recreated
by this verification task.

## Delivered Scope

An authenticated Portfolio owner can record BUY, SELL, DIVIDEND, DEPOSIT,
WITHDRAWAL and FEE, list deterministic cursor history, retrieve immutable facts,
and correct a public fact by atomically adding a reversal and full replacement.
Same-key semantic retries preserve the committed result. Accepted mutations
commit ledger, completed idempotency, audit and versioned outbox together.

The frontend supplies canonical Asset selection, UTC entry, decimal-string
validation, review/confirmation, safe loading/empty/error/retry states and
relationship navigation. Archived Portfolio history is read-only. The server
remains the authority for ownership, eligibility, decimal/time validity and
transient ordered quantity replay.

## Explicitly Excluded Scope

Repository runtime module/route inventory, migrations, M3 PR file inventories
and contract/browser negative assertions contain no Holding projection, lots,
FIFO/cost basis, cash projection, price observations, provider adapter/ingestion,
valuation, allocation, financial dashboard, alerts, documents or AI runtime.
No CRYPTO financial processing, shorting, margin, derivatives, public ADJUSTMENT,
direct REVERSAL, DRAFT, PATCH/DELETE or mutable edit is delivered.

Planning descriptions of those future modules are not implementation. M3 cash
event records are not a cash-balance projection. Transient quantity validation
is not persisted Holding/FIFO state.

## Provider Permission Gate Evidence

[PR #48](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/48)
is merged and its SHA is an ancestor of this base. The
[gate record](../governance/m3-price-provider-permission-gate.md) has APPROVED
metadata and every mandatory approval-table row VERIFIED.

Product approver Thanasak Srisaeng's 2026-08-29 decision selects Twelve Data.
Provider contractual confirmation is from Artemis, Technical Support, Twelve
Data, dated 2026-08-31, for Venture and the owner/explicitly invited authenticated
private-beta users. These are recorded human/provider decisions, not a new
Engineering legal interpretation or a fresh license grant.

The record specifies official regular-session unadjusted close, Time Series
1day/adjust=none, USD US EQUITY/ETF, primary exchange/MIC and exchange-local date.
Active-subscription retention, raw-data deletion within 30 days after termination,
approved attribution including a dofollow source link, and no unapproved
redistribution remain unchanged. Public commercial availability, exports and AI
data use are not authorized. Future operational quotas/client caching/credential
mechanics remain future-provider constraints, not an M3 adapter permission.

The old “pending merge / blocked until PR #48” text is reconciled factually.
No confidential emails, contracts, logos, API keys or credentials are added.

## Contract Evidence

[OpenAPI](../../packages/api-contracts/openapi/v1.yaml) defines only:

- POST and GET `/api/v1/portfolios/{portfolioId}/transactions`;
- GET `/api/v1/portfolios/{portfolioId}/transactions/{transactionId}`;
- POST `/api/v1/portfolios/{portfolioId}/transactions/{transactionId}/corrections`.

All require BearerAuth; only commands require Idempotency-Key. Trade inputs are
Asset/quantity/unitPrice/optional non-negative fee; dividend is Asset/positive
amount; deposit/withdrawal/standalone fee are positive amount without Asset.
All financial transport values are strings. Gross/net are not command truths.
Command UTC-Z timestamps cannot be future; read filters may be future.
Text preserves supplied contents and distinguishes empty from absent.

The frozen bounds, cursor, correction result and status/error mappings are
covered by nine M3 tests within the 26-test contract suite. Reads expose
correction links without owner authority, snapshots or persistence internals.
Contract generation/typecheck/drift passed locally with no changes.

## Database and Persistence Evidence

Migration 00004 extends Platform; migration 00005 owns transactions,
transaction_corrections, transaction_portfolio_sequences and
transaction_idempotency. Existing migrations are unchanged by this task.

Financial columns are exact NUMERIC with finite/scale≤12/positive constraints,
not floats or fixed typmods that silently round. History and replay use the
full effective-time/sequence/ID tuple in descending and ascending order.
Portfolio-local sequence UPSERT and idempotency advisory transaction locks
serialize competing commands. Correction allocates reversal before replacement.

Deferred scoped FKs make incomplete correction groups uncommittable; direct
role uniqueness permits chains while preventing a second direct reversal.
No update/delete query for ledger facts or corrections exists. This is an
application/persistence API immutability guarantee, not a claim that privileged
database operators lack SQL write authority.

Existing PostgreSQL tests cover constraints/index planner shapes, separate-pool
sequence/idempotency/correction concurrency, 32-byte digest/365-day expiry,
caller-owned atomicity and fail-closed populated Down. CI executes actual Goose
empty/reset/v2/v4/v5/down/up and those integration packages. Local execution was
environment-blocked as detailed below; current-head remote DB evidence is required.

## Domain and Application Evidence

[Domain tests](../../backend/internal/transaction/domain/domain_test.go) prove
exact decimal grammar/arithmetic, kind/field/currency/time/Asset rules, immutable
facts/chains and ordered replay. Replay rejects negative asset quantity at every
ordered position, including later existing SELL affected by backdating.

Application ownership uses the Portfolio public reader on the same transaction;
Asset lookup uses its own public boundary. Lock order is Portfolio ownership,
idempotency, sequence, then replay/write. Portfolio archive cannot race acceptance.

Fingerprint v1 includes scope and correction target, canonical decimals, UTC
microseconds, kind/Asset/currency and exact optional text. Metadata and generated
identities are excluded. Replays read the prior committed identity, including
after archival or changed Asset eligibility. Different semantics conflict.
`TestApplicationConcurrentSameKeyAndPortfolioSequence` and
`TestApplicationConcurrentCorrectionHasIndependentScope` exercise independent
pools; `TestApplicationRollsBackEveryWriteBoundary` injects failures at financial,
relationship, replacement, audit, outbox and idempotency boundaries.

## HTTP and Security Evidence

Strict request decoding enforces UTF-8, duplicate/unknown field rejection,
financial strings, body/key/text limits and command-specific fields. The real
authenticated integration test composes Identity, Portfolio, Asset and
Transaction over PostgreSQL, not mocked HTTP endpoints.

Missing/cross-owner/unrepresentable opaque identities use ownership-safe 404:
Portfolio absence for collection operations, Transaction absence for individual
get/correct. Standard ErrorEnvelope and correlation behavior are retained.
Malformed input/kind/currency use 400; deterministic conflicts 409; financially
invalid/archived/non-correctable commands 422. Internal failure is generic 500,
never SQL or submitted financial data.

## Frontend Evidence

Generated contract types are reused with strict runtime response/error checking.
History preserves server order/cursors; filters reset pagination. Client decimal
checks are string-based; date handling validates timestamps, not money.
No authoritative financial Number/parseFloat calculation was found.

Entry and correction maintain one memory-only attempt object per mounted form.
Canonical semantic retries—including token refresh—retain the same key;
meaningful changes/target changes produce a new key; success clears it.
Mutations disable automatic retry and retain no mutation cache after disposal.
No localStorage/sessionStorage/cookie/URL command-key setup exists.

Correction explicitly keeps the original and submits a complete replacement,
not an edit. Tests cover chain links in both directions, related records beyond
current pages/filters, stale/already-corrected/not-correctable handling and
archived read-only views. Local frontend suite: 34 files / 191 tests PASS.

## Audit Evidence

Platform's append-only allowlist retains Authentication compatibility and adds:
transaction_create_success, transaction_create_failure,
transaction_idempotent_replay, transaction_idempotency_conflict,
transaction_correction_initiated, transaction_correction_completed,
transaction_correction_rejected, transaction_reversal_created,
transaction_replacement_created and transaction_ownership_rejection.

Only safe actor/Portfolio/Transaction/correction references, action/result,
severity, time and correlation are stored. No arbitrary request/financial map
is added. Create appends success audit; correction appends initiated,
reversal-created, replacement-created and completed actions. All success
evidence shares the financial transaction. Rejected-command failure audit after
rollback is a distinct safe outcome, not a leaked partial success.

## Outbox Evidence

transaction.recorded.v1 and transaction.corrected.v1 commit in the same pgx.Tx
as the financial facts. Payload is schemaVersion plus bounded role/UUID
references, not duplicated prices, amounts, notes or credentials. Platform owns
stream counters, leases, safe failure codes and consumer dedup;
Transaction owns the event's business roles. Outbox persistence failure rolls
back the entire authoritative mutation.

## ADR-022 Delivery Engine Evidence

Defaults remain lease 60s, retry base 5s, maximum 5m, maximum delivery invocations 10,
batch 50 and poll 2s. Full-jitter exponential backoff is overflow-safe and polling
is paced even for zero jitter. SKIP LOCKED claims respect immutable aggregate
predecessors; claim/reclaim increments attempt_count. Durable claim tokens fence
stale publication, reschedule and dead-letter updates.

Attempts 1–10 may invoke the publisher; tenth retryable failure dead-letters.
Administrative recovery after an expired final claim may observe attempt 11:
zero publisher calls, delivery_attempts_exhausted, DEAD_LETTER. Later events in
that aggregate stay blocked; other streams may proceed. There is no automatic
dead-letter replay. Cancellation stops new claims; successful in-flight work
may acknowledge only through its current durable token.

Unit tests and PostgreSQL delivery tests cover these outcomes, including stale
owners and expired batch items. Injected test publishers prove engine behavior;
they are not an approved production receiver.

## ADR-023 Runtime-Inactive Publication Evidence

`cmd/worker` passes no publisher to `RunConfigured`. The inactive path validates
configuration, logs “transaction publication inactive”/no_approved_receiver and
runs dependency heartbeat without constructing an outbox store/delivery runner.

`TestRunConfiguredWithoutReceiverNeverStartsDelivery` proves zero claims.
`TestCommittedTransactionRemainsPendingDuringInactiveWorker` commits a real
command then runs the same composition boundary: claim count stays 0 and the
entire durable outbox row remains unchanged, PENDING/attempt 0.

Thus no receiver consumes claims, attempts, publisher invocations, retries,
dead-letter transitions or acknowledgements. Ledger acceptance and atomic
outbox persistence do not depend on receiver availability. This is the expected
PASS state, **not active end-to-end publication**. A separately approved real
receiver/handoff contract is required for activation. No broker/no-op sink/M4
consumer is present.

## Real HTTPS Browser Evidence

The merged suite/configuration/CI were inspected directly:
[runbook](m3-real-stack-e2e.md),
[critical flow](../../apps/web/tests/m3-e2e/m3-ledger-critical-flow.spec.ts).
It runs Chromium→Caddy HTTPS→Next.js production→Go API→postgres-test with
synthetic EQUITY/ETF/CRYPTO catalog fixtures, real registration and Portfolio.

It exercises canonical BUY review/confirmation, exact same-key replay,
unchanged history after rejected CRYPTO, complete correction, retained original,
reversal/replacement links, replacement correctability and cross-owner safe
absence. Negative assertions cover Edit/Delete, direct public internal kinds,
financial projection UI, provider calls and browser credential persistence.

Same-key replay and CRYPTO rejection use supplemental real HTTPS commands with
the captured principal only in memory. They do not mock endpoints or prove an
uncertain browser-delivery retry; focused frontend tests prove key retention.
The critical browser flow is one BUY/correction lifecycle, not exhaustive browser
execution of every transaction kind or race; lower-layer suites cover that breadth.

M3 is serial with retries 0. Trace snapshots/network capture are disabled to avoid
credential artifacts. Current-head CI must also show Auth, M2 and ten repeated M2
production lifecycle passes with no retry/flaky result. Local browser PASS is
not claimed.

## Local Verification

All commands used Node 24.19.0 / pnpm 10.18.3 where Node was required.

| Command/check                                                                       | Actual result                                                           |
| ----------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `pnpm agent:test`                                                                   | PASS 20/20                                                              |
| `pnpm agent:validate`                                                               | PASS                                                                    |
| `pnpm agent:next`                                                                   | M3-VERIFY-001 / in_progress / implement_or_diagnose                     |
| `pnpm format:check`                                                                 | PASS                                                                    |
| `pnpm lint`                                                                         | PASS                                                                    |
| `pnpm typecheck`                                                                    | PASS                                                                    |
| `pnpm test`                                                                         | PASS 34 files / 191 tests                                               |
| `NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1 pnpm build`                  | PASS; this build variable is not the supported browser Auth origin      |
| `pnpm contract:check`                                                               | PASS 26 tests, lint/generate/typecheck/drift                            |
| `pnpm security:audit:test`                                                          | PASS 14/14                                                              |
| `pnpm security:audit`                                                               | PASS under exact temporary exception; no other High/Critical advisories |
| `test -z "$(gofmt -l backend)"`                                                     | PASS                                                                    |
| `go vet ./...`                                                                      | PASS                                                                    |
| `go test ./...`                                                                     | PASS; untagged integration tests are not claimed executed               |
| `go build ./backend/cmd/api`; `go build ./backend/cmd/worker`                       | PASS                                                                    |
| `sh scripts/check-module-boundaries.sh`                                             | PASS                                                                    |
| `sqlc generate` plus generated-target diff                                          | PASS, zero drift across all five targets                                |
| `go test -race -count=1` on Transaction domain/application/HTTP and Platform worker | PASS, all four packages                                                 |
| `git diff --check` and changed-document link/format checks                          | PASS; 95 local link targets exist; new reports pass Prettier            |

### Local database/browser limitation

Docker 29.7.2 answers version queries, but `docker pull postgres:17-alpine`
fails with:

```text
error creating temporary lease: write /var/lib/desktop-containerd/daemon/io.containerd.metadata.v1.bolt/meta.db: input/output error
```

An exec probe of the existing isolated test container also fails with an
overlay2 I/O error. Its PostgreSQL 17.11 TCP endpoint remains reachable. A fresh,
verification-only database was therefore created on 127.0.0.1:55433 and actual
Goose up attempted with existing migrations. Versions 1–3 applied, but 00004
failed with SQLSTATE 58030:

```text
could not load library "/usr/local/lib/postgresql/plpgsql.so":
Error loading shared library /usr/local/lib/postgresql/plpgsql.so: I/O error
```

Goose remained at version 3. The new disposable database was removed after this
diagnostic attempt; existing databases/volumes/containers were not reset or
stopped. No local integration/concurrency/migration-completion or real HTTPS
browser PASS is claimed. Further local integration/browser execution was not
attempted against the damaged runtime. Current-head remote PostgreSQL 17 and
real browser gates must supply the execution evidence.

## Remote CI Evidence

The verification PR is the final evidence surface, not merged PR #75 or #76.
Its Current-head evidence section must record the live SHA and workflow ID,
with exact job IDs/results for frontend, backend, contracts-and-generation,
database-integration, browser-e2e, compose-smoke and secrets, plus GitGuardian.

Inspection must include actual database migration/integration logs and browser
logs, not just green job badges: Auth suite, M2 suite, ten M2 lifecycle repeats,
M3 real-stack flow, fixture seeding/repeat safety and teardown. No retry/flaky
result is acceptable. Branch behind protected main must be 0 and unresolved
review threads 0; maintainer final review stays unchecked.

No earlier workflow is represented here as final-head evidence.

## Security Evidence

`GHSA-vfj7-8cjw-p6xm`, `braces@3.0.3`: **ACCEPTED TEMPORARY RISK**,
expires **2026-10-31**. It is not fixed, not remediated, not silently suppressed,
and not extended. The exact fail-closed audit policy and exception remain
unchanged. Local audit-policy tests and live audit passed. Remote secrets and
GitGuardian are mandatory on the final head.

No confidential provider documents, keys, credentials or raw token/cookie output
are included. Ownership, HTTPS topology and memory-only Authentication remain
unchanged.

## Deviations and Findings

- Critical: 0.
- Major: 0.
- Blocking Minor: 0 identified in the implementation audit.
- Informational: local Docker/container filesystem I/O failure prevents local
  DB/browser execution; remote evidence is mandatory, not waived.
- Informational (reconciled): provider-gate PR #48 pending-merge wording was
  obsolete despite APPROVED evidence and a verified protected-main merge.
- Informational (historical): the original M3 plan still carries Proposed/
  initial repository-state wording, and foundation architecture documents
  describe later tasks as not started at their delivery snapshot. Verified merge
  ancestry and this completion report establish present implementation state.
  No ADR acceptance or policy is changed to rewrite those historical snapshots.

Any failed current-head required gate or newly discovered acceptance defect
overrides this recommendation and blocks closure. It must be reported, not
repaired through hidden product changes in this documentation task.

## Remaining Limitations

- No approved concrete publication receiver: durable pending events can
  accumulate. Activation and operational recovery require separate review.
- Idempotency expiry is 365 days, not permanent business duplicate detection.
  Exact-key expired cleanup is active; bounded batch-cleanup SQL is provided,
  but no global background cleanup job is wired in the heartbeat-only runtime.
- Local Docker must be repaired by its operator before local stack evidence can
  be repeated; no destructive recovery was performed here.
- Provider scope remains private beta; operational ingestion configuration and
  unapproved redistribution/AI/commercial expansion require later decisions.
- This is foundation verification, not M6 production readiness, load testing,
  disaster recovery, or M4/M5 projection delivery.
- Temporary braces risk expires 2026-10-31 and remains subject to its removal trigger.

## Closure Recommendation

M3 Closure Recommendation:
All reviewed acceptance criteria PASS. Recommend closing M3 after the
M3-VERIFY-001 PR passes its current-head required gates and is reviewed and
merged under ADR-013.

M3 Status: Pending M3-VERIFY-001 PR review and merge

## Recommended Next Step

Maintainer reviews this acceptance matrix/report and the verification PR's exact
current-head evidence, then decides whether to merge under ADR-013. Do not
automatically merge protected main, mark M3 closed while the PR is open, change
the queue, or begin M4.
