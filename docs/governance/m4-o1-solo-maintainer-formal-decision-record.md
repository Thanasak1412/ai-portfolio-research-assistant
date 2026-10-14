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
approve an emergency delegation, approve operational alerting, or authorize implementation
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

| Field                        | Confirmed direction                                           | Remaining decision / evidence                                                               |
| ---------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| Timezone                     | `Asia/Bangkok` (UTC+07:00)                                    | No alternative timezone inferred                                                            |
| Monday–Friday                | 20:00–23:00 `Asia/Bangkok`                                    | Weekday start/end confirmed; no overnight or immediate-response guarantee inferred          |
| Saturday–Sunday              | All-day best-effort availability                              | NOT guaranteed 24-hour monitoring or continuously staffed coverage                          |
| Public holidays              | NOT YET CONFIRMED                                             | Holiday, leave and absence handling pending                                                 |
| In-window response           | Best-effort direction; no SLA                                 | Exact response/acknowledgement expectations pending                                         |
| Outside coverage             | Respond during the next defined coverage window               | Holiday/absence handling and unavailable-operator/escalation procedure pending              |
| Preferred alert channel      | Email; logical destination `private-beta-primary-alert-inbox` | Gmail setup owner-attested only; sender/transport approval and M4 delivery evidence pending |
| Escalation / incident record | GitHub Issues proposed, not approved or integrated            | Severity routing, bounce/failure handling and acknowledgement workflow approval pending     |

Email preference is not evidence of a working mail pipeline. Do not infer the
operational destination from the connected GitHub account or publish private
contact information. Record only approved, appropriately controlled references.
No M4 alert transport is implemented, tested or operational by this document.

## O1-S4 — Proposed Email Alert and Escalation Design

**Status:** PROPOSED / PENDING FORMAL APPROVAL — limited private beta only.

**Accountable operator direction:** GitHub `Thanasak1412`, subject to final O1 sign-off.

### Owner-attested Gmail evidence and its limits

On 2026-10-14 the maintainer supplied written direction in the current
conversation attesting that a dedicated Gmail inbox has been selected,
automatic forwarding to the primary inbox is configured and tested, and
Gmail label classification is configured and tested. This is
**OWNER-ATTESTED GMAIL FORWARDING AND LABEL VERIFICATION ONLY**.
The attestation record date is not an independently verified test timestamp
or final operating-approval date.

- Logical destination: `private-beta-primary-alert-inbox`.
- Physical destination: maintainer-designated dedicated Gmail inbox, with
  forwarding to the primary inbox; physical addresses/targets are excluded.
- Classification direction: `AI Portfolio / Alerts`.
- Backup destination/operator: none; residual-risk approval still pending.
- Sender/transport and restricted configuration approach: **PENDING APPROVAL**.

No independent delivery verification, end-to-end M4 Monitoring test, approved
sender/transport implementation, tested fallback escalation, operational
alerting/monitoring, or production readiness is established. Gmail forwarding
alone is not a reliable fallback channel. Email addresses, credentials, tokens
and sensitive message contents must not enter this public governance record;
future physical routing must use separately approved restricted configuration.

### Proposed severity and safe incident envelope

| Severity | Proposed classification                                                                                                                                                                          | Approval boundary                                                |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------- |
| CRITICAL | Integrity invariant violations; missing/inconsistent durable responsibility; terminal failures or dead-letter conditions requiring human intervention; unauthorized/unsafe activation conditions | Classification does not authorize repair, replay or activation   |
| WARNING  | Backlog/reconciliation lag beyond separately approved thresholds; repeated nonterminal processing failures                                                                                       | O3/O6 numeric thresholds remain pending their separate approvals |
| INFO     | Routine diagnostics and normal status, generally retained in logs                                                                                                                                | No new delivery/retention policy approved                        |

The future approved monitoring mechanism would preserve an incident ID,
event timestamp, affected subsystem and bounded safe diagnostic metadata.
Do not include financial bodies, provider payloads, private messages or secrets.
Classification cannot override ADR-022, ADR-023, ADR-024 or O3/O6 decisions.

### Proposed acknowledgement and escalation workflow

1. Detect/classify through a separately approved monitoring mechanism;
   preserve incident identity/time, subsystem and safe context.
2. Send to the logical destination only when the approved alert mechanism
   is implemented. Neither transport nor integration exists by this record.
3. Operator acknowledgement is best-effort during the next applicable
   coverage window: Mon–Fri 20:00–23:00 `Asia/Bangkok`; weekends all-day
   best effort, not continuous monitoring. No 24/7 response SLA is implied.
4. Record acknowledgement, investigation, resolution or escalation in
   **GitHub Issues as a proposed Incident Register**, not an approved workflow
   or implemented integration. Register/access and acknowledgement approval
   remain pending; preserve only safe references in any future public issue.
5. If the operator is unavailable, retain the incident as unresolved.
   No nonexistent backup, immediate response or automatic fallback is implied.

Delivery/forwarding failures, bounces, unanswered alerts, severity escalation,
public holidays, leave and absence handling remain unresolved design
requirements. Eventual delivery failure must be observable, but this document
does not claim a mechanism or approved response deadline exists.

### Phase A approval versus Phase B verification

Phase A still needs explicit maintainer design approval, approved sender/
transport and secure configuration, Incident Register/acknowledgement workflow,
escalation/delivery-failure/absence procedures and no-backup risk acceptance.
O1-S6 emergency-pause delegation/procedure and final dated O1 sign-off with a
verifiable reference remain pending. Recording this proposal is not approval.

Phase B must separately prove delivery from the actual approved environment,
receipt in dedicated and forwarded inboxes, correct severity/incident ID/time/
safe metadata, failed-delivery and unattended-operator cases, acknowledgement/
escalation/audit exercises, and ADR-preserving safety/shutdown behavior.
These tests are activation prerequisites, not circular preconditions requiring
unimplemented M4 functionality before Phase A design can be reviewed.

Automated safety controls, if later implemented and approved, must not depend
on reading email. Emergency pause remains subject to O1-S6. No automatic
dead-letter replay, speculative ACK, integrity repair or evidence deletion is
authorized. Reactivation requires explicit human authorization and approved
preflight; process restart alone is never approval to reactivate.

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

| Subdecision                     | Recorded partial direction                                         | Status / required completion                                                                                                   |
| ------------------------------- | ------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------ |
| O1-S1 Staffing                  | Solo model and provisional assignments above                       | PARTIAL; complete role attestation, effective scope and verifiable approval reference pending                                  |
| O1-S2 No-backup risk            | No backup, limited private beta, no 24/7 SLA                       | PARTIAL; explicit residual-risk acceptance, bounded period, review/expiry and beta-expansion trigger pending                   |
| O1-S3 Coverage                  | Mon–Fri 20:00–23:00; Sat–Sun all-day best effort; Asia/Bangkok     | PARTIAL; public holidays/absence handling and response expectations pending                                                    |
| O1-S4 Alert route               | Proposed email design; Gmail forwarding/labels owner-attested only | PROPOSED / PENDING FORMAL APPROVAL; transport/configuration/register/escalation approval and Phase B M4 delivery proof missing |
| O1-S5 Outside coverage          | Next defined window; safety boundary above                         | PARTIAL; final unavailable-operator and escalation procedure pending                                                           |
| O1-S6 Emergency pause / restart | No restart-to-activation; separate reactivation authorization      | BLOCKED; explicit pause delegation, procedure, preflight and audit design pending                                              |
| O1-S7 Final sign-off            | Pending record authorized only                                     | BLOCKED; complete O1 decision, authorized approver/role, UTC date and verifiable evidence reference missing                    |

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
