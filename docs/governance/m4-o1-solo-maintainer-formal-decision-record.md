# M4-GATE-001 — O1 Formal Operating Decision Record

**Document status:** Scoped Phase A decisions recorded — O1 remains blocked

**Decision status:** O1-S1, O1-S2, O1-S3, O1-S5, and the limited Phase A O1-S6 authority model are ACCEPTED within the scope below. O1 remains BLOCKED; O1-S4 and final O1 closure remain pending.

**Task:** M4-GATE-001

**Evidence base:** protected main `50123611c8903074deee679c0edfa9d8f2dd427a`, merged [PR #90](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/90)

**Scoped approval date:** 2026-10-16

**Scoped approval reference:** [Maintainer decision comment on PR #90](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/90#issuecomment-6220004375), authored by `Thanasak1412` (`OWNER`) at `2026-10-16T16:29:21Z`.

**Record history:** Initial pending record dated 2026-10-11; coverage clarification dated 2026-10-12; O1-S4 proposal dated 2026-10-14; scoped Phase A decision recorded 2026-10-16. This scoped approval is not final O1 acceptance.

**Evidence reconciliation:** 2026-10-23 at protected main `8c126ff0858ff9cb7c17435d6bfea42a951f3196`. The [O1-S4 review packet](m4-o1-s4-review-packet.md) links verified private register preparation and the selected Transactional Email API class; complete sender/transport, access/configuration and failure/escalation approval remain pending. No accepted O1 clause is changed by that packet or the separately merged dependency security policy.

**Coverage update:** 2026-10-12 — maintainer confirmed Monday–Friday 20:00–23:00 `Asia/Bangkok` in the current conversation; coverage clarification only, not final O1 approval

**Decision scope:** The linked maintainer comment approves only O1-S1, O1-S2, O1-S3, O1-S5, and the limited Phase A O1-S6 authority model. It expressly leaves O1-S4 pending and O1 overall blocked.

**Policy sources:** [O1 operating proposal](m4-holdings-operating-decision-proposal.md), [ADR-013](../adr/ADR-013-solo-maintainer-merge-governance.md), [ADR-022](../adr/ADR-022-transaction-outbox-delivery-operational-policy.md), [ADR-023](../adr/ADR-023-transaction-publication-activation-prerequisite.md), [ADR-024](../adr/ADR-024-m4-holdings-durable-handoff.md)

## Authority and scope

This record preserves the exact scope of the maintainer's **scoped Phase A
decision**, not complete O1 approval. The maintainer accepted the limited
single-operator/no-backup risk for the initial private beta, with the review
trigger below, and approved the limited emergency-pause authority model as a
design decision only. Sender/transport, secure configuration, and an
access-restricted private Incident Register remain unresolved under O1-S4.
No runtime pause capability, alert delivery, or receiver activation is
authorized by this record. The attributable approval evidence is the linked
GitHub comment; its author and actual creation time are recorded above.

Architecture clauses A1–A7 are already accepted under the separate
[ADR-024 maintainer decision](m4-adr-024-maintainer-decision-record.md).
That architecture approval expressly excludes O1–O8 operating decisions.
This document, its PR review, green CI, or merge cannot substitute for the
missing complete operating decision.

## Accepted O1-S1 ownership and O1-S2 risk boundary

| Function           | Accepted Phase A assignment / direction   | Boundary                                                                               |
| ------------------ | ----------------------------------------- | -------------------------------------------------------------------------------------- |
| Architecture Owner | GitHub `Thanasak1412`                     | No additional financial-policy authority                                               |
| Platform Owner     | GitHub `Thanasak1412`                     | No additional financial-policy authority                                               |
| Holdings Owner     | GitHub `Thanasak1412`                     | No additional financial-policy authority                                               |
| Primary Operator   | GitHub `Thanasak1412`                     | Phase A ownership only; execution depends on approved, implemented and tested controls |
| Deputy / backup    | None for the initial limited private beta | Accepted temporary risk; no automatic extension                                        |
| Availability       | Coverage defined below; no 24/7 SLA       | No continuous-monitoring or uptime guarantee                                           |

The maintainer accepts possible incident-response delay and service
interruption. Data integrity takes precedence over availability. The risk
review is due within 30 days after the scoped approval or before expanding the
beta population, whichever occurs first. There is no silent extension. This
approval does not assign Transaction data-integrity or financial-policy
authority beyond the stated operating roles.

## Accepted O1-S3 coverage, O1-S5 acknowledgement, and alerting status

| Field                        | Confirmed direction                                           | Remaining decision / evidence                                                                     |
| ---------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| Timezone                     | `Asia/Bangkok` (UTC+07:00)                                    | No alternative timezone inferred                                                                  |
| Monday–Friday                | 20:00–23:00 `Asia/Bangkok`                                    | Weekday start/end confirmed; no overnight or immediate-response guarantee inferred                |
| Saturday–Sunday              | All-day best-effort availability                              | NOT guaranteed 24-hour monitoring or continuously staffed coverage                                |
| Public holidays              | All-day best-effort, same as weekends                         | No guaranteed continuous monitoring                                                               |
| In-window response           | Best-effort; no SLA                                           | Acknowledge during the next applicable coverage window                                            |
| Outside coverage             | Respond during the next applicable coverage window            | No immediate-response guarantee; unresolved incidents remain open                                 |
| Planned absence              | Documented coverage exception and safety assessment required  | No backup is implied; follow separately approved safety controls                                  |
| Unplanned absence            | No backup coverage implied                                    | Incidents remain unresolved until the operator returns                                            |
| Preferred alert channel      | Email; logical destination `private-beta-primary-alert-inbox` | Gmail setup owner-attested only; sender/transport approval and M4 delivery evidence pending       |
| Escalation / incident record | Access-restricted private Incident Register required          | O1-S4 pending; sender, secure configuration, register and remaining escalation details unresolved |

Email preference is not evidence of a working mail pipeline. Do not infer the
operational destination from the connected GitHub account or publish private
contact information. Record only approved, appropriately controlled references.
The application repository is public, so its GitHub Issues do not meet the
access-restricted private Incident Register requirement. The separate
`Thanasak1412/incident-register` repository was verified private with Issues
enabled on 2026-10-23; its [documentation package](https://github.com/Thanasak1412/incident-register/pull/1)
is merged. Formal selection/access/workflow approval remains pending under
O1-S4. Prepared private documentation does not establish an implemented,
tested or operational M4 alert transport.

## O1-S4 — Proposed Email Alert and Escalation Design

**Status:** PROPOSED / PENDING FORMAL APPROVAL — limited private beta only.

**Accountable operator:** GitHub `Thanasak1412`, as assigned by accepted O1-S1. The O1-S4 alert route and implementation remain unapproved.

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
- Backup destination/operator: none; the temporary no-backup risk is accepted
  only within O1-S2's recorded scope and review trigger, not as a fallback route.
- Sender/transport and restricted configuration approach: **PENDING APPROVAL**.

The later [restricted transport proposal](https://github.com/Thanasak1412/incident-register/blob/c03c9c2a3ec6719bd2b238bf61e6024906ea02ca/docs/m4-transactional-email-transport-phase-a.md)
records the maintainer-selected **Transactional Email API class**. Provider,
sender identity, credential/configuration lifecycle and formal O1-S4 approval
remain pending. This direction does not approve an email provider or deployment.

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
   coverage window: Mon–Fri 20:00–23:00 `Asia/Bangkok`; weekends/public holidays all-day
   best effort, not continuous monitoring. No 24/7 response SLA is implied.
4. Record acknowledgement, investigation, resolution or escalation in the
   prepared access-restricted private Incident Register only after O1-S4
   approves its identity/access/manual workflow. The public application's
   Issues must not be used for private incident data. The private package's
   merge is preparation evidence; final operating approval remains pending.
   No automatic Issues delivery integration is approved or implemented.
5. If the operator is unavailable, retain the incident as unresolved.
   No nonexistent backup, immediate response or automatic fallback is implied.

The O1-S5 policy is accepted: acknowledge during the next applicable coverage
window; an unacknowledged incident remains unresolved, and delivery failure is
not acknowledgement. There is no automatic escalation to a backup. Sender /
transport, bounce detection and response, secure configuration, private
register workflow approval and remaining escalation details are O1-S4 blockers. This
document does not claim an implemented mechanism or delivery guarantee.

### Phase A approval versus Phase B verification

The maintainer has approved O1-S1, O1-S2, O1-S3, O1-S5 and the limited O1-S6
authority model as Phase A decisions, as referenced above. O1-S4 still needs
sender/transport and secure configuration approval, an access-restricted
Incident Register, and resolution of delivery-failure and remaining escalation
details. Final O1 sign-off remains pending while O1-S4 is unresolved.

Phase B must separately prove delivery from the actual approved environment,
receipt in dedicated and forwarded inboxes, correct severity/incident ID/time/
safe metadata, failed-delivery and unattended-operator cases, acknowledgement/
escalation/audit exercises, and ADR-preserving safety/shutdown behavior.
These tests are activation prerequisites, not circular preconditions requiring
unimplemented M4 functionality before Phase A design can be reviewed.

Automated safety controls, if later implemented and approved, must not depend
on reading email. O1-S6 approves only the limited authority model; it does not
make emergency-pause operation available. No automatic
dead-letter replay, speculative ACK, integrity repair or evidence deletion is
authorized. Reactivation requires explicit human authorization and approved
preflight; process restart alone is never approval to reactivate.

## Emergency operations and safety boundaries

- O1-S6 accepts `Thanasak1412`'s limited Phase A authority model to pause new
  receiver/calculation work when safety or integrity is at risk. The authority
  cannot be exercised until an approved, implemented and tested pause
  mechanism exists. This decision does not claim such a mechanism exists.
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

| Subdecision                     | Recorded partial direction                                                                                  | Status / required completion                                                                                                                 |
| ------------------------------- | ----------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| O1-S1 Staffing                  | `Thanasak1412` assigned all four roles; no financial-policy authority                                       | ACCEPTED for the stated Phase A scope; no operational-control claim                                                                          |
| O1-S2 No-backup risk            | Temporary no-backup risk accepted for initial limited private beta                                          | ACCEPTED; review within 30 days or before beta expansion, whichever comes first; no automatic extension                                      |
| O1-S3 Coverage                  | Weekdays 20:00–23:00; weekends/public holidays all-day best effort; Asia/Bangkok                            | ACCEPTED; planned absence requires documented safety assessment; unplanned absence implies no backup                                         |
| O1-S4 Alert route               | Proposed email design; Gmail forwarding/labels owner-attested only                                          | PENDING; sender/transport, secure configuration, private register and remaining escalation details; Phase B delivery evidence still required |
| O1-S5 Outside coverage          | Next applicable window; unacknowledged incidents remain unresolved; delivery failure is not acknowledgement | ACCEPTED; no backup escalation or response-time SLA                                                                                          |
| O1-S6 Emergency pause / restart | Limited authority model for `Thanasak1412`; separate reactivation authorization and preflight               | ACCEPTED as Phase A design only; operation prohibited until approved, implemented and tested pause mechanism exists                          |
| O1-S7 Final sign-off            | Scoped decision evidence recorded above                                                                     | BLOCKED; O1-S4 and consolidated final O1 closure remain outstanding                                                                          |

**Full O1 approval:** PENDING — O1-S4 remains unresolved. **Scoped approval
timestamp:** `2026-10-16T16:29:21Z`. **Scoped GitHub reference:** [PR #90
decision comment](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/90#issuecomment-6220004375).
This is not final O1 acceptance.

Phase A has accepted the limited ownership, coverage, no-backup risk,
acknowledgement and emergency-pause authority decisions referenced above.
O1-S4 remains the unresolved Phase A blocker. Phase B separately requires real
alert delivery/receipt and acknowledgement evidence, escalation exercises,
safe pause/recovery tests and explicit activation authorization. Missing Phase B
proof blocks activation; the unresolved O1-S4 decision keeps O1 blocked now.

## Preserved project state and next action

- ADR-024 A1–A7: **Accepted — architecture only**.
- O1: **BLOCKED — partial direction only**; O2–O8: **BLOCKED**.
- ADR-025: **Proposed**; financial/provider prerequisites unchanged.
- M4-GATE-001: **in_progress / BLOCKED**; M4-CONTRACT-001: **blocked**.
- Receiver activation and implementation: **NOT AUTHORIZED**.
- SECURITY_EXCEPTION-001: retired from the active audit policy after upstream
  `GHSA-vfj7-8cjw-p6xm` withdrawal, per the merged
  [retirement record](security-exception-001-braces.md). The policy has zero
  active exceptions; `braces@3.0.3` remains in the graph. Withdrawal is not
  package remediation; **2026-10-31** is the unchanged historical expiry.
  O1-S2's separate accepted no-backup operating risk and review trigger remain
  unchanged; this security retirement does not close any O1 decision.

Next human action: close O1-S4 sender/transport, secure configuration,
access-restricted private Incident Register and remaining escalation
decisions; then make the separate final O1 decision. The scoped approvals are
recorded, but O1 remains **BLOCKED**, not accepted. All documentation changes
remain subject to ADR-013;
no automatic Ready, merge, queue transition or runtime activation is permitted.
