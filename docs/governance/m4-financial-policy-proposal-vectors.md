# M4 Financial Policy Proposal and Exact Review Vectors

**Status:** Proposed review fixtures, not accepted calculator policy

**Task:** M4-GATE-001

**Evidence base:** `4c212f3e139cd08745ec531779317852600bbf64`

**Owner:** Financial-policy maintainer / architecture maintainer

Read with [ADR-025](../adr/ADR-025-m4-financial-compatibility-and-reconciliation.md), [Decision Closure](../planning/decision-closure-specification.md) and the [M4 plan](../planning/price-data-holding-projection-execution-plan.md). M4-GATE-001 remains **BLOCKED**. These are deterministic inputs and review oracles, not executed Holdings tests or self-approved decisions.

## Fixture conventions

One Portfolio P; synthetic eligible US-listed USD EQUITY Assets X and Y. T0=`2026-09-01T13:30:00Z`, T1=`2026-09-02T13:30:00Z`, T2=`2026-09-03T13:30:00Z`, T3=`2026-09-04T13:30:00Z`; creation time=`2026-10-01T00:00:00Z`. Each row lists immutable sequence, kind, Asset, quantity, unit price, effective time. Fee is decimal string `0` unless specified. Symbols P/X/Y and A/B/R are stable fixture aliases; assign fixed UUIDs by alias in future tests, never randomize the financial inputs. All quantities/prices/costs are exact decimal strings; no binary floats. IDs break ties only after sequence; sequences are unique per Portfolio.

`R(A)` is an internal reversal copying A's dimensions and effective time, with its own later sequence. A correction group links original A, R(A), replacement B; B can be the original of a later group. Snapshots include every complete group and source fact; never select only terminal transactions before checking compatibility. M3 quantity traces below are algebraic derivations from inspected ReplayLedger, not claims these new fixtures were run against a Holdings implementation.

**APPROVED-BASELINE ARITHMETIC** means unambiguous ordinary FIFO/half-even consequences. **PROPOSED OUTPUT** is conditional on ADR-025 acceptance. **UNRESOLVED COST** deliberately has no numeric cost oracle. These labels must survive translation into later tests; a missing oracle cannot become a guessed expectation.

## Correction compatibility inputs

### CF1 — fully consumed BUY; M3-valid cancellation counterexample

| Sequence / ID | Fact            | Asset | Quantity | Unit price | Time |
| ------------- | --------------- | ----- | -------- | ---------- | ---- |
| 1 / A         | BUY             | X     | `10`     | `10`       | T1   |
| 2 / S         | SELL            | X     | `10`     | `30`       | T1   |
| 3 / L         | BUY             | X     | `10`     | `20`       | T1   |
| 4 / R         | R(A)            | X     | `10`     | `10`       | T1   |
| 5 / B         | replacement BUY | X     | `10`     | `15`       | T2   |

Group A→R/B. M3 ordered X balances=`10,0,10,0,10`; final quantity=`10`. Before correction S's FIFO cost=`100`, proceeds=`300`, realized=`200`. Removing A/R starts with S and has no available lot. **UNRESOLVED COST:** after correction, which cost/quantity is neutralized from L, whether S is restated, final open-lot basis and recognized cost cannot be inferred. The pre-correction `100` is not a post-correction oracle. Required decision: explicit journal entries with source lot, removed quantity, removed cost, any sale restatement, final basis and conservation proof.

### CF2 — partially consumed BUY

Use CF1 with S quantity=`4` and B quantity=`8`; all prices/times/other facts unchanged. M3 balances=`10,6,16,6,14`. Pre-correction S cost=`40`, proceeds=`120`, realized=`80`; A remainder quantity=`6`, cost=`60`, L=`10`/`200`. **UNRESOLVED COST:** cancellation again leaves S before any BUY; final M3 quantity=`14` does not determine surviving-lot costs. Decision must account for both consumed and unconsumed parts of A, not only its original total.

### CF3 — SELL correction without intervening dependency

Facts: sequence1 A BUY X `10` at `10`, T0; sequence2 S SELL X `4` at `20`, T1; sequence3 R(S), T1; sequence4 B replacement SELL X `2` at `30`, T2. Group S→R/B. M3 balances=`10,6,10,8`.

**PROPOSED OUTPUT (undo consumed-lot journal):** S's `40` consumed cost and `80` proceeds are neutralized, not recognized twice; restore the original A allocation. B consumes `2`/`20`; proceeds=`60`; realized=`40`; A remaining quantity=`8`, cost=`80`. The reversal is not a new lot at S's sale price. This simple case does not decide intervening-sale/BUY reversal attribution.

### CF4 — replacement corrected again; multi-generation chain

All X at T1: seq1 A BUY `10` at `10`; seq2 R(A); seq3 B replacement BUY `8` at `12`; seq4 R(B); seq5 C replacement BUY `6` at `15`. Groups A→R(A)/B and B→R(B)/C. M3 balances=`10,0,8,0,6`.

**PROPOSED OUTPUT:** only C has net acquisition effect; open lot C quantity=`6`, cost=`90`; net recognized sale cost/proceeds/realized=`0`. Preserve both groups and all five immutable identities. No direct reversal has two targets; replacement and original roles coexist on B.

### CF5 — consumed replacement in a chain

Start CF1, then add at T2: seq6 S2 SELL X `10` at `25`; seq7 L2 BUY X `10` at `22`; seq8 R(B); seq9 C replacement BUY X `5` at `18`, T3. Second group B→R(B)/C. Full M3 balances=`10,0,10,0,10,0,10,0,5`.

**UNRESOLVED COST:** final quantity=`5`, but both consumed-original neutralizations require exact lineage/recognition decisions. A terminal-only replay is invalid at S; resolving only the first group does not resolve the second. Supply every affected sale and residual, not merely final quantity.

### CF6 — same-effective-time order

Ordinary baseline, all X/T1: seq1 BUY `2` at `10`; seq2 BUY `2` at `20`; seq3 SELL `3` at `30`. **APPROVED-BASELINE ARITHMETIC:** cost=`20+20=40`, proceeds=`90`, realized=`50`, second lot remaining=`1`/`20`. Reversing BUY order would cost `50` instead; input permutation must not change sorted output. CF1 separately covers the same-time correction counterexample; its post-correction cost remains unresolved.

### CF7 — cross-Asset correction

Use CF1, but B is BUY Y `5` at `7`, T2. Group A→R/B. M3 X balances through sequence4=`10,0,10,0`; Y final=`5`. Y acquisition cost=`35` is ordinary arithmetic. **UNRESOLVED COST:** X sale neutralization/recognized cost cannot be settled by Y's valid independent lot; no final combined financial oracle or approval until X attribution is decided. Never transfer cost from X to Y merely to balance a projection.

### CF8 — backdated replacement with an unambiguous candidate

Seq1 A BUY X `3` at `10`, T2; seq2 S SELL X `2` at `20`, T3; seq3 R(A), T2; seq4 B replacement BUY X `2` at `5`, T1. Group A→R/B. Sorted signed trace B,A,R,S=`2,5,2,0`.

**PROPOSED OUTPUT under neutralization/rebuild:** B supplies S's `2`, cost=`10`, proceeds=`40`, realized=`30`; final quantity/cost=`0`/`0`. A's `30` acquisition and its reversal neutralize. This case alone cannot justify using cancellation for CF1. Source checkpoint remains sequence4; business ordering is not insertion ordering.

### CF9 — backdating still does not resolve consumed BUY

Use CF1 but B quantity=`8`, price=`15`, time=T0. Sorted balances B,A,S,L,R=`8,18,8,18,8`; M3-valid. Canceling A/R gives B `8` before S `10` and fails. **UNRESOLVED COST:** final X quantity=`8`; no numeric FIFO cost is authorized. Require explicit attribution, not a hidden restriction of backdating or use of a future lot before its acquisition.

## Exact ordinary FIFO and decimal vectors

Costs below exclude correction ambiguity. Proposed closure reconciliation is explicitly distinct from fixed base rounding. Define H12=half-even to 12 fractional places, H18=half-even to 18; trailing zero presentation does not change values.

### DR1 — multi-lot, fees and fractional quantity

BUY X `1.5` at `10`, fee=`0.5` → lot A cost=`15.5`. BUY X `2` at `20`, fee=`1` → lot B cost=`41`. SELL X `2` at `30`, fee=`2`.

**APPROVED-BASELINE ARITHMETIC:** close A cost=`15.5`; B contribution=H18(`41*0.5/2`)=`10.25`; sale cost=`25.75`; proceeds=`58`; realized=`32.25`; B remaining quantity=`1.5`, cost=`30.75`. No reconciliation adjustment; exact total acquisition=`56.5` = recognized `25.75` + remaining `30.75`.

### DR2 — thirds, final residual and persisted conservation

BUY X quantity=`3`, price=`0.3`, fee=`0.1` → exact cost=`1`. Three sequential SELLs of X quantity=`1`, price=`1`, fee=`0`.

| Sale              | Exact/18-place contribution | Remaining exact cost   | H12 base recognized | Proposed closure adjustment | Proposed recognized | Proposed realized (proceeds `1`) |
| ----------------- | --------------------------- | ---------------------- | ------------------- | --------------------------- | ------------------- | -------------------------------- |
| S1                | `0.333333333333333333`      | `0.666666666666666667` | `0.333333333333`    | `0`                         | `0.333333333333`    | `0.666666666667`                 |
| S2                | `0.333333333333333334`      | `0.333333333333333333` | `0.333333333333`    | `0`                         | `0.333333333333`    | `0.666666666667`                 |
| S3 (full closure) | `0.333333333333333333`      | `0`                    | `0.333333333333`    | `0.000000000001`            | `0.333333333334`    | `0.666666666666`                 |

S2 is an H18 half-even tie (odd last retained digit rounds upward). Exact contributions sum=`1`. Base persisted total=`0.999999999999`; **PROPOSED OUTPUT** total with separate closure adjustment=`1`. Final quantity/basis=`0`/`0`. This is a policy-version decision, not quietly changing S3's rounded base cost.

### DR3 — cross-lot rounding counterexample and conservation granularity

BUY A X quantity=`0.000001`, price=`0.0000006`; BUY B same values. Each exact acquisition cost=`0.0000000000006` (legal input scales). One SELL X quantity=`0.000002`, price=`1` closes both lots.

**APPROVED-BASELINE ARITHMETIC:** exact aggregate cost=`0.0000000000012`; H12 sale cost=`0.000000000001`; proceeds=`0.000002`; base realized=`0.000001999999`; final quantity/basis=`0`/`0`. Exact per-lot contribution strings remain `0.0000000000006`, not independently rounded ledger values. **PROPOSED OUTPUT:** episode target H12(`0.0000000000012`)=`0.000000000001`; closure adjustment=`0`; exact-to-persisted difference=`0.0000000000002` retained as rounding provenance. Independently rounded per-lot target sums=`0.000000000002` and is explicitly NOT the proposal's conservation target.

### DR4 — half-even ties at both boundaries

| Exact operation              | Expected rounded value | Classification      |
| ---------------------------- | ---------------------- | ------------------- |
| H12(`0.0000000000025`)       | `0.000000000002`       | even retained digit |
| H12(`0.0000000000035`)       | `0.000000000004`       | odd retained digit  |
| H18(`0.0000000000000000025`) | `0.000000000000000002` | even retained digit |
| H18(`0.0000000000000000035`) | `0.000000000000000004` | odd retained digit  |

All are **APPROVED-BASELINE ARITHMETIC**. Full-lot inputs for the first two are quantity=`0.000001`, price=`0.0000025` or `0.0000035`. Round at the designated output boundary, not on the original product.

### DR5 — repeated tiny partial sales and final closure

BUY X quantity=`0.000010`, price=`0.0000001` → exact acquisition cost=`0.000000000001`. Ten SELLs, each quantity=`0.000001`, price=`1`, fee=`0`. Every proportional/full contribution=`0.0000000000001`; each sale proceeds=`0.000001`. After sale i, remaining quantity=`(10-i)*0.000001`, exact remaining cost=`(10-i)*0.0000000000001` for integer i in 1..10 (exact coefficient arithmetic).

Each H12 base cost=`0`. **PROPOSED OUTPUT:** S1..S9 recognized cost=`0`, realized=`0.000001`; S10 base=`0`, closure adjustment and recognized=`0.000000000001`, realized=`0.000000999999`. Recognized total=`0.000000000001`; exact remainder and quantity at closure=`0`. No threshold discards the tiny residual. Run after shuffled input to verify sorting, not dependence on delivery order.

### DR6 — separate inventory episodes

Repeat DR3 twice with distinct immutable acquisitions/sales, second episode later than the first closure. **PROPOSED OUTPUT:** each closed episode target/recognized=`0.000000000001`; combined recognized=`0.000000000002`. H12 of both episodes' exact acquisition sum (`0.0000000000024`) happens to agree here; it is not the algorithm. Contrast three episodes: recognized=`0.000000000003`, whereas H12(`0.0000000000036`)=`0.000000000004`. The proposed episode boundary, not whole-history net rounding, is authoritative **only if approved**. Maintainer must explicitly accept this consequence. Correction changes to episode boundaries require recomputation and accepted combined oracles.

## Review checklist and unresolved decisions

| Decision                                        | Recommendation / deliverable                                                                                     | Owner / state                                        |
| ----------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------- |
| Consumed BUY neutralization                     | Preferred reversible journal; choose exact CF1/CF2/CF5/CF7/CF9 cost and lineage, including restatement           | Financial-policy maintainer / UNRESOLVED             |
| Simple SELL undo / replacement chain / backdate | Review CF3/CF4/CF8 proposed numeric tuples; verify against chosen universal rule                                 | Financial-policy maintainer / PENDING                |
| Rounding granularity                            | Approve or reject ADR-025 episode-level target, lossless contribution provenance and explicit closure adjustment | Financial-policy / architecture maintainer / PENDING |
| Compatibility/version                           | Approve new version tags, retained generations and deterministic rebuild; no M3 mutation                         | Financial-policy / architecture maintainer / PENDING |

Required later test assertions: complete graph/source checkpoint, sorted quantities, every consumed/open lot lineage and cost, base cost, reconciliation, recognized cost, proceeds, realized, exact residual/provenance, final Holding basis, deterministic rebuild/permutation equality. An incompatible vector must not publish a partial/current generation. No valuation, Price, tax-lot choice or client calculation is introduced.

**M4-GATE-001 Review: Blocked** — exact inputs are prepared; unresolved cost oracles and human policy decisions remain mandatory.
