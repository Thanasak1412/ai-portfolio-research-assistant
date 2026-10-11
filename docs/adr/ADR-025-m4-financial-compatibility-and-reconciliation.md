# ADR-025 — M4 Financial Compatibility and Reconciliation

**Status:** Proposed

**Date:** 2026-10-11

**Task:** M4-GATE-001

**Decision owner:** Financial-policy maintainer / architecture maintainer

**Proposed policy versions:** `FIFO_CORRECTION_COMPAT-v1`; `DECIMAL_RECONCILIATION-v2`

**Identifier check:** ADR-025 absent from local and remote protected-main ADR inventory at `4c212f3e139cd08745ec531779317852600bbf64`.

## Context

[Decision Closure](../planning/decision-closure-specification.md) fixes FIFO, immutable correction relationships, signed quantity validity, 18-place proportional allocations, aggregate half-even at 12 places and final residual conservation. The [M4 plan](../planning/price-data-holding-projection-execution-plan.md) explicitly identifies two unresolved intersections. Neither can be settled in a calculator/schema as a hidden default.

M3 [ReplayLedger](../../backend/internal/transaction/domain/replay.go) validates signed quantities, not costs. [BuildCorrection](../../backend/internal/transaction/domain/correction.go) preserves original financial dimensions/time in the reversal, assigns later reversal/replacement sequences, and permits replacement chains. An M3-valid stream can become FIFO-negative if corrected originals and reversals are simply removed. [Exact review vectors](../governance/m4-financial-policy-proposal-vectors.md) demonstrate this without changing M3.

Independent 12-place lot rounding also conflicts with aggregate round-once: two exact lot costs of `0.0000000000006` total `0.0000000000012`; rounded individually their sum is `0.000000000002`, but rounded together the total is `0.000000000001`.

## Proposed decisions — no adoption yet

### Correction compatibility: decision required before adoption

Recommend the M4 plan's **reversible lot-consumption journal** direction for further financial approval, not deleting facts or treating a reversed SELL as a BUY at sale price. Preserve source graph, original acquisition costs and consumption provenance; rebuild the whole Portfolio at a complete committed checkpoint. A reversal of SELL must undo the original consumed-lot allocations, not acquire at sale price. Replacements retain their own effective time/sequence, including cross-Asset and backdated cases.

This direction is not a complete policy for a reversal of an already-consumed BUY. The maintainer must decide which surviving lot bears removed quantity/cost and whether historical sale attribution is restated. No authoritative numeric cost is proposed for those ambiguous streams. Specifically, decide CF1/CF2/CF5/CF7/CF9 in the vector document, including exact lot lineage, sale recognized cost/proceeds/realized result and final basis. Acceptance must include those missing numeric oracles; accepting only this direction cannot close the gate.

| Approach                                                                           | Compatibility consequences                                                                                                                            | Recommendation                                                                                                 |
| ---------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| Delete corrected original/reversal and replay terminal replacements                | CF1/CF2 fail availability despite M3 validity; can alter historical sale basis. Needs explicit incompatible-stream treatment/version                  | Not a universal default                                                                                        |
| Move replacement to original acquisition time/sequence                             | Can produce computable costs but contradicts immutable replacement ordering and backdating semantics                                                  | Reject without separate M3 policy/contract change, outside this task                                           |
| Signed reversible lot journal with explicit consumption/neutralization attribution | Can preserve ordering and provenance, but fully consumed BUY neutralization needs a new exact financial rule; complexity is real, not solved by dedup | Preferred proposal; pending complete approved oracles                                                          |
| Reject such corrections in M3                                                      | Retroactively narrows accepted command space and changes runtime policy                                                                               | Not authorized                                                                                                 |
| Report projection incompatible with no new financial output                        | Honest fail-closed handling, not full FIFO coverage; last-good generation must be explicitly stale, never current                                     | Temporary safety proposal only; Product/financial approval needed, cannot silently substitute for gate closure |

### Cross-lot reconciliation: explicit versioned proposal

Recommend `DECIMAL_RECONCILIATION-v2` as an **explicit limited change** to v1 reconciliation granularity, not an assertion that v1 already decided it:

- Retain exact per-lot working acquisition/remaining cost and 18-place half-even proportional allocations. On full closure consume the exact remaining cost. Never round each lot to 12 before aggregating.
- Persist each sale's aggregate base recognized cost/proceeds at 12 places with half-even. Exact contribution/residual provenance is lossless (coefficient/scale or exact rational where needed), not a second independently rounded authoritative lot ledger. Schema encoding is deferred; no floating point and no precision loss are permitted.
- For a closed Portfolio/Asset inventory episode (position zero after the ordered fact), conservation target is half-even-12 of the **exact aggregate acquisition cost for that episode**, not the sum of independently rounded lot targets. Add a separately identified closure reconciliation amount to that closing sale so accumulated persisted recognized cost equals that target. Base rounded cost and reconciliation must remain separately inspectable; recognized cost = base + reconciliation. Realized result uses recognized cost. No hidden epsilon or discarded residual.
- Open episodes retain the difference between exact recognized cost and accumulated persisted recognized cost as provenance, without adding it to a user's position. A checkpoint does not close an episode. Rebuild computes the same episodes in immutable order; policy version is part of the generation identity.
- This changes strict per-lot persisted conservation and, on closure, independently rounded per-sale output. It therefore requires express acceptance of the new version and compatibility/rebuild planning, not a DECIMAL-v1 implementation shortcut. Existing M3 decimals/commands and public semantics stay unchanged.

Competing alternatives: independently round every lot (violates aggregate-once); keep only aggregate totals with no residual evidence (cannot explain conservation); higher-precision monetary public outputs (changes contract precision); cumulative rounding every sale (moves each sale's round-once boundary). The proposal keeps base round-once evidence and makes only final closure adjustment explicit. Approval must cover DR1–DR6 and whether episode-level rather than per-lot persisted conservation is acceptable.

Corrections can alter episodes. Recompute complete history under the accepted correction policy; do not reuse a prior episode residual blindly. Combined correction/reconciliation expected outputs remain unresolved until the correction oracles exist.

## Versioning and approval requirements

If accepted, new policy/calculation version produces a new derived generation from the unchanged ledger; old generation/calculation evidence remains identifiable. No in-place mutation of ledger, M3 acceptance, input scales, currency scope or provider selection. Financial maintainer must approve version tags, correction allocation and all vector outputs; architecture maintainer must approve provenance/rebuild compatibility. Update higher-level policy references in a separately reviewed acceptance change before downstream schema/contract implementation.

This ADR is **Proposed**, not effective. No financial output, M4-CONTRACT-001 authorization, migration or calculator is implemented. Overall gate remains **BLOCKED**. An AI self-review cannot supply the missing financial decision.
