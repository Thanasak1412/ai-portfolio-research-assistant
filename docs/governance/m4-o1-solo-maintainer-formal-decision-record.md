# M4-GATE-001 — O1 Formal Operating Decision Record

**Document status:** PENDING documentation record — partial human direction only

**Decision status:** O1 BLOCKED; NOT ACCEPTED

**Task:** M4-GATE-001

**Evidence base:** protected main `0f9325f051c2bf27e43c158dceb6e7f91b1a5f31`, merged [PR #86](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/86)

**Record date:** 2026-10-11; no final UTC operating-approval timestamp recorded

**Coverage update:** 2026-10-12 — maintainer confirmed Monday–Friday 20:00–23:00 `Asia/Bangkok` in the current conversation; coverage clarification only, not final O1 approval

**Source:** owner-supplied O1 decision record and explicit maintainer direction in the current conversation, authorizing a pending documentation PR only

**Policy sources:** [O1 operating proposal](m4-holdings-operating-decision-proposal.md), [ADR-013](../adr/ADR-013-solo-maintainer-merge-governance.md), [ADR-022](../adr/ADR-022-transaction-outbox-delivery-operational-policy.md), [ADR-023](../adr/ADR-023-transaction-publication-activation-prerequisite.md), [ADR-024](../adr/ADR-024-m4-holdings-durable-handoff.md)

## Authority and scope

This record preserves confirmed **partial direction**, not complete O1 approval.
The maintainer authorized recording a solo-maintainer, no-backup, limited
private-beta operating direction. They did not accept the outstanding risk,
approve an emergency delegation, verify alerting, or authorize implementation
or activation. No GitHub operating-approval evidence is claimed.

Architecture clauses A1–A7 are already accepted under the separate
[ADR-024 maintainer decision](m4-adr-024-maintainer-decision-record.md).
That architecture approval expressly excludes O1–O8 operating decisions.
This document, its PR review, green CI, or merge cannot substitute for the
missing complete operating decision.

## Confirmed direction and provisional assignments

| Function           | Assignment / direction                    | Approval boundary                                                 |
| ------------------ | ----------------------------------------- | ----------------------------------------------------------------- |
| Architecture Owner | GitHub `Thanasak1412`                     | Existing architecture ownership; full O1 sign-off remains pending |
| Platform Owner     | GitHub `Thanasak1412`                     | Provisional                                                       |
| Holdings Owner     | GitHub `Thanasak1412`                     | Provisional                                                       |
| Primary Operator   | GitHub `Thanasak1412`                     | Provisional; emergency authority/procedure pending                |
| Deputy / backup    | None for the limited private beta         | Subject to final, explicit no-backup risk acceptance              |
| Availability       | Limited best-effort coverage; no 24/7 SLA | No continuous monitoring or uptime guarantee                      |

One person may provisionally carry these responsibilities. This does not
establish uninterrupted availability, implemented operational controls, or
assign Transaction data-integrity or financial-policy approval for other tasks.
No solo-maintainer risk exception is accepted by this pending record.

## Coverage and alerting

| Field                        | Confirmed direction                             | Remaining decision / evidence                                                                            |
| ---------------------------- | ----------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| Timezone                     | `Asia/Bangkok` (UTC+07:00)                      | No alternative timezone inferred                                                                         |
| Monday–Friday                | 20:00–23:00 `Asia/Bangkok`                      | Weekday start/end confirmed; no overnight or immediate-response guarantee inferred                       |
| Saturday–Sunday              | All-day best-effort availability                | NOT guaranteed 24-hour monitoring or continuously staffed coverage                                       |
| Public holidays              | NOT YET CONFIRMED                               | Holiday, leave and absence handling pending                                                              |
| In-window response           | Best-effort direction; no SLA                   | Exact response/acknowledgement expectations pending                                                      |
| Outside coverage             | Respond during the next defined coverage window | Holiday/absence handling and unavailable-operator/escalation procedure pending                           |
| Preferred alert channel      | Email                                           | Destination, delivery mechanism and successful alert test NOT YET VERIFIED                               |
| Escalation / incident record | No approved route or register recorded          | Sender/transport, recipient, severity routing, bounce/failure handling and acknowledgement route pending |

Email preference is not evidence of a working mail pipeline. Do not infer the
operational destination from the connected GitHub account or publish private
contact information. Record only approved, appropriately controlled references.
No alert transport is implemented, tested or operational by this document.

## Emergency operations and safety boundaries

- Emergency-pause authority and procedure remain **PENDING**. The proposed
  scope is stopping new receiver/calculation claims when safety or integrity
  is at risk; no emergency delegation is accepted here.
- Do not claim that pause, fail-closed automation, recovery, alerting or
  reactivation controls exist. Outside coverage, fail closed where necessary
  and supported by approved controls; otherwise preserve inactive/paused
  operation under a separately approved procedure, not an invented mechanism.
- Preserve ledger/outbox evidence, committed responsibility, fences, attempts
  and last-good complete generations. Operator absence does not authorize
  speculative ACK, silent loss, partial publication or integrity repair.
- Exceptional data repair, dead-letter intervention and destructive actions
  require separately recorded human decisions under their governing policies.
  No automatic replay, skip, fabricated ACK or evidence deletion is authorized.
- Reactivation requires separately approved authorization and preflight;
  process restart alone never authorizes receiver activation.
- These role directions do not override ADR-022, ADR-023 or ADR-024. Before
  approved receiver activation, ADR-023 continues to require zero claims,
  attempts, publisher calls, retries, dead-letter transitions and ACKs.

## Pending approval and evidence register

| Subdecision                     | Recorded partial direction                                     | Status / required completion                                                                                 |
| ------------------------------- | -------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| O1-S1 Staffing                  | Solo model and provisional assignments above                   | PARTIAL; complete role attestation, effective scope and verifiable approval reference pending                |
| O1-S2 No-backup risk            | No backup, limited private beta, no 24/7 SLA                   | PARTIAL; explicit residual-risk acceptance, bounded period, review/expiry and beta-expansion trigger pending |
| O1-S3 Coverage                  | Mon–Fri 20:00–23:00; Sat–Sun all-day best effort; Asia/Bangkok | PARTIAL; public holidays/absence handling and response expectations pending                                  |
| O1-S4 Alert route               | Email preferred                                                | BLOCKED; approved destination/transport/escalation design and later delivery evidence missing                |
| O1-S5 Outside coverage          | Next defined window; safety boundary above                     | PARTIAL; final unavailable-operator and escalation procedure pending                                         |
| O1-S6 Emergency pause / restart | No restart-to-activation; separate reactivation authorization  | BLOCKED; explicit pause delegation, procedure, preflight and audit design pending                            |
| O1-S7 Final sign-off            | Pending record authorized only                                 | BLOCKED; complete O1 decision, authorized approver/role, UTC date and verifiable evidence reference missing  |

**Full O1 approval:** PENDING. **Final UTC approval time:** PENDING.
**GitHub / controlled operating-approval reference:** PENDING.
No approver timestamp, missing operating value or final approval is fabricated.

Phase A must approve an implementable ownership, coverage, alert/escalation,
risk and emergency-operations design. It does not require unimplemented M4
controls to exist first. Any limited design approval must identify its scope
without promoting incomplete O1 to VERIFIED. Phase B separately requires real
alert delivery/receipt and acknowledgement evidence, escalation exercises,
safe pause/recovery tests and explicit activation authorization. Missing Phase B
proof blocks activation; the missing Phase A decisions above keep O1 blocked now.

## Preserved project state and next action

- ADR-024 A1–A7: **Accepted — architecture only**.
- O1: **BLOCKED — partial direction only**; O2–O8: **BLOCKED**.
- ADR-025: **Proposed**; financial/provider prerequisites unchanged.
- M4-GATE-001: **in_progress / BLOCKED**; M4-CONTRACT-001: **blocked**.
- Receiver activation and implementation: **NOT AUTHORIZED**.
- SECURITY_EXCEPTION-001: unchanged accepted temporary risk through
  **2026-10-31**, not remediated or extended.

Next human action: confirm holiday/absence handling and response expectations;
approve alert/escalation design, emergency authority/procedure
and dated no-backup risk boundaries; then supply explicit, scoped approval
with a verifiable date/reference. Until then this is a **pending record, not an
O1 acceptance PR**. All documentation changes remain subject to ADR-013;
no automatic Ready, merge, queue transition or runtime activation is permitted.
