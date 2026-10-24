# M4 O1-S4 — Resend Provider and Transport Formal Decision Review

**Task:** `M4-GATE-001 / O1-S4`

**Review packet:** `O1-S4-REVIEW-2026-10-23-v1`

**Document version:** `O1-S4-RESEND-REVIEW-2026-10-24-v1`

**Status:** `PROPOSED / HUMAN DECISION PENDING` — not an accepted O1-S4 decision

**Repository evidence base:** protected `main` at `c844b45125bf8501015ef53781c4ec156c89dbf7`, after merged [PR #100](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/100). The original proposal followed [PR #96](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/96); this synchronization inherits the Zod security remediation without changing any human decision, approval date or S4 acceptance criterion.

**Human decision owner:** repository maintainer in the already accepted Platform/Holdings/Primary Operator roles; provider-account/security evidence belongs to the respective responsible owner. No additional financial-policy or Product authority is inferred.

## Authority and relationship to the existing packet

This record adapts the maintainer-supplied `m4-o1-s4-resend-formal-decision-review.md` review draft. The current task explicitly preserves **Resend as the preferred provider**, not as a formally approved S4-1 choice. There is no account activation or usage approval evidence in this draft.

The merged [O1-S4 review packet](m4-o1-s4-review-packet.md) remains the authority for S4-1 through S4-8 acceptance criteria; this document is a provider-specific proposal to review against those criteria, not a replacement policy. Existing decisions are authoritative only within their recorded scope in the [O1 formal record](m4-o1-solo-maintainer-formal-decision-record.md), accepted ADRs and [readiness gate](m4-implementation-readiness-gate.md).

Neither creation nor merge of this document approves a provider account, credentials, domain purchase, DNS changes, deployment, automated incident integration, O1 closure or receiver activation. An accountable human must supply a dated attributable decision identifying exact approved clauses and controlled evidence. No approval is fabricated below.

## 1. Evidence states and existing direction

`VERIFIED` qualifies a cited fact only. `PROPOSED` is a review candidate, `MISSING` is absent evidence/approval, and `BLOCKED` is an unmet gate. Public technical documentation is not contractual permission, account entitlement or human approval.

| Item                                    | Evidence/status                                                                                                                                    | Boundary                                                                                                                                            |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| Email alert channel                     | Recorded preference in the [O1 formal record](m4-o1-solo-maintainer-formal-decision-record.md)                                                     | No tested M4 delivery or fallback implied.                                                                                                          |
| Transactional Email API transport class | VERIFIED as owner direction in the [O1-S4 packet](m4-o1-s4-review-packet.md)                                                                       | A class selection, not a named-provider or implementation approval.                                                                                 |
| Dedicated Gmail forwarding/labels       | OWNER-ATTESTED only, as recorded in O1                                                                                                             | Use logical destination `private-beta-primary-alert-inbox`; no physical addresses or independent end-to-end M4 verification.                        |
| Manual private Incident Register        | Prepared repository/form/procedure, as referenced by the packet                                                                                    | Final access/workflow approval remains MISSING; no bot, GitHub API or webhook integration authorized.                                               |
| O1-S1/S2/S3/S5 and limited Phase A S6   | Accepted through the [PR #90 maintainer decision](https://github.com/Thanasak1412/ai-portfolio-research-assistant/pull/90#issuecomment-6220004375) | Preserve roles, no-backup review trigger, coverage, next-window acknowledgement and limited pause-authority boundaries unchanged.                   |
| Resend preferred provider               | Maintainer direction to preserve the preference in this task and supplied draft                                                                    | PROPOSED named-provider choice; S4-1 formal acceptance and account evidence MISSING. No provider comparison is reopened without a concrete blocker. |
| Domain/subdomain pattern                | Illustrative proposal in the supplied draft                                                                                                        | Not evidence of registration, DNS control, verified sender or approved hosting.                                                                     |

## 2. S4-1 — Proposed provider and sender decision

**Preferred candidate:** Resend Transactional Email API, using outbound HTTPS from a future approved server-side notification adapter. HTTPS REST rather than SMTP is the proposed choice for explicit request IDs, HTTP error classification and provider-event correlation. This is not approval of live sending.

Propose one owner-controlled registered domain with separate sending and web subdomains, illustrated as `alerts.<controlled-domain>` and `app.<controlled-domain>`. Actual domain ownership, DNS authority and approved sender identity are MISSING evidence. Review provider-required SPF/DKIM, DMARC, return-path and bounce handling before authorizing their configuration. Resend's [verified-domain documentation](https://resend.com/docs/dashboard/domains/introduction) describes owned-domain verification and subdomain separation; it does not verify this project's domain. A `.vercel.app` hostname is not assumed to be an authorized sender domain.

Required human evidence/decision remains:

- Named Resend account owner, permitted operational use and controlled account reference.
- Exact selected plan, account-specific transactional/daily/rate limits, costs, overage/stop-spending envelope and intended private-beta use. [Public pricing](https://resend.com/pricing) is research only; no advertised allowance is adopted as this account's entitlement or operating budget.
- Controlled domain registration/DNS-authority and sending-domain/from-identity evidence, plus approved sender authentication and return-path/bounce requirements. No actual domain, mailbox, account identifier or DNS secret is disclosed here.
- Attributable S4-1 approval and external-provider use authorization. If actual technical, licensing or delivery evidence blocks Resend, stop and request an explicit decision; do not silently switch providers.

No account is activated, domain bought, DNS record changed or sender verified by this PR. Operational proof and live sending remain gated separately from review of this design.

## 3. S4-2 — Proposed server-side configuration boundary

- Keep API credentials and any future webhook signing secret server-side, with environment separation, least privilege, logical configuration references and synthetic CI fixtures. Never publish physical mailboxes, credentials or raw provider payloads in Git, URLs, logs or frontend bundles.
- [ADR-015](../adr/ADR-015-ed25519-key-ring-and-secret-management.md) already selects Railway service-variable secrets for staging/production. This proposal does not change that policy or assume an email-specific injection design is approved. The supplied draft's Vercel/domain pattern is not a verified hosting decision and does not approve Vercel secret injection, a hosted Go worker or another secret store. Any policy/infrastructure change needs separate human approval.
- Obtain approval for exact deployment/config references, access and environment isolation, rotation/revocation, emergency suspension, missing/invalid-credential behavior, redaction, outbound host/TLS restrictions and retention ownership. No credentials are requested or handled here.
- Preserve [ADR-023](../adr/ADR-023-transaction-publication-activation-prerequisite.md): email transport gains no authority to claim, ACK or replay Transaction outbox events, or activate a worker publisher.

## 4. S4-3 — Private Incident Register workflow proposal

Use the prepared access-restricted `incident-register` repository as the proposed **manual** acknowledgement/evidence surface, subject to its own access, privacy and workflow acceptance. The original packet contains the preparation references; they are not copied into a new approval claim here.

Required decisions cover actual access review and owner roles; manual create/investigate/mitigate/resolve/close/reopen procedure; retention/audit ownership; and privacy/access rechecks. A merged Issue Form, prior synthetic exercise or prepared procedure is not final approval. No automated issue creation, email-to-Issue ingestion, bot or webhook may infer acknowledgement or closure.

## 5. S4-4 — Proposed transport outcomes and provider events

Propose a durable incident identity, separate notification-intent identity, attempt evidence and provider message correlation. Keep incident lifecycle independent from transport outcomes. These are proposed application labels, not a frozen provider-event enumeration or implemented storage schema:

| State/evidence                               | Interpretation / proposed invariant                                                                                        |
| -------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| `PENDING`                                    | Notification intent durably recorded; no delivery outcome asserted.                                                        |
| `SUBMITTING` / `SUBMITTED`                   | Request in flight/submitted; timeout or crash leaves uncertainty until reconciled.                                         |
| `ACCEPTED`                                   | Provider API acceptance with attributable message ID; not recipient delivery, reading or project approval.                 |
| `DELIVERED`                                  | Provider-reported recipient-server delivery; not Gmail inbox placement, forwarding or human acknowledgement.               |
| `DELAYED`, `BOUNCED`, `SUPPRESSED`, `FAILED` | Distinct evidence-backed outcomes; exact event mapping, subtype handling and retry/terminal classification remain pending. |
| `UNKNOWN`                                    | Ambiguous submission, timeout or missing callback; no fabricated success/failure or blind resend after idempotency expiry. |
| `HUMAN_ACKNOWLEDGED`                         | Separate incident lifecycle event with actual human actor and UTC time, never inferred from email state.                   |

Propose signed provider webhooks only after separately approved design, topology and implementation. [Resend verification guidance](https://resend.com/docs/webhooks/verify-webhooks-requests) requires the raw body for signature verification. Review authentication before parsing/accepting content, timestamp/replay protection, deduplication, safe message-ID correlation, durable receipt before HTTP acknowledgement, and reconciliation for missing/out-of-order/redelivered events. This callback HTTP acknowledgement is not a Transaction outbox ACK or operator acknowledgement.

Resend's [January 2026 event visibility change](https://resend.com/changelog/webhook-event-visibility) documents recipient-specific delivery outcomes. For the initial single-recipient proposal, review per-recipient outcome handling without persisting private addresses in public artifacts. Open/click tracking is never human ACK. Exact event mappings and evidence-retention rules remain unapproved.

Phase A must approve the callback/reconciliation design and owners. A public callback endpoint, durable event storage or background worker must then be separately authorized and implemented; Phase B supplies runtime activation proof. A serverless request handler is not assumed to provide durable background responsibility. No endpoint, webhook or integration is created here.

## 6. S4-5 — Proposed idempotency, retry and containment

- Use one stable application idempotency key per immutable notification intent; same-key retries preserve the identical payload. [Resend documents a 24-hour key-retention window](https://resend.com/docs/dashboard/emails/idempotency-keys), not lifetime deduplication. The proposal therefore requires durable application intent/outcome history beyond that window; its lifetime and implementation are not approved here.
- Approve an explicit transient/permanent error matrix before retrying. Distinguish concurrent-request `409` from changed-payload `409`; they are not the same failure. Credential, sender/DNS, suppression and invalid-request failures must not become unbounded retry loops.
- Exact timeout, attempts, backoff/jitter, rate/cost budget, retention, failure-review delay and containment bounds are **MISSING** until the accountable owner approves them with actual account evidence. ADR-022 outbox values are not email defaults.
- Preserve `UNKNOWN` results for controlled reconciliation/manual review, especially after provider idempotency expires. Do not create a second untracked incident or another M3 Transaction/outbox event to retry email.
- The operator needs an approved non-email path to inspect failed/missed intents and unacknowledged incidents. Email failure cannot be its sole detection or containment signal. No fallback channel or backup operator is invented.

## 7. S4-6 / S4-7 — Escalation, safe envelope and authority

Accepted coverage, absence and risk terms remain in the [O1 formal record](m4-o1-solo-maintainer-formal-decision-record.md): weekdays 20:00–23:00 Asia/Bangkok, weekends/public holidays best effort, next-coverage-window acknowledgement, no 24/7 SLA and no backup. Planned absence needs a documented safety assessment. O1-S2's original no-backup risk review trigger is neither renewed nor conflated with a dependency-security exception.

Escalation, independent containment, pause and restart procedures still need the relevant S4/O3/O6/O7 decisions and implemented/tested controls. No new operator, fallback channel, automatic pause/restart, repair or receiver activation is approved.

Propose only approved severity, sanitized incident ID, subsystem, UTC timestamps, safe reason code and bounded correlation/reference fields. Exclude financial amounts and Transaction/Holding bodies, licensed provider data, mailboxes, credentials, raw exception traces, account IDs and private incident narratives from shared artifacts. Restricted callback ingestion still needs an approved minimal schema, redaction/access policy and retention; a signature does not authorize arbitrary payload retention.

S4-7 must approve the exact envelope and severity mapping; numerical WARNING thresholds and associated CRITICAL triggers retain their separate O3/O6 owners. No classification creates Transaction ACK, replay, repair or reactivation authority.

## 8. Outstanding Phase A human decisions

All rows remain **PROPOSED / HUMAN DECISION PENDING**. The original packet's full criteria still apply; this table maps the provider proposal to them without replacing their authority.

| ID   | Proposed position                                                  | Required human action                                                                                                                                                        |
| ---- | ------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| S4-1 | Resend preferred; HTTPS API; controlled sending identity           | Approve provider/account/use/plan, quota/cost envelope, sender/domain evidence and authentication requirements. No account activation implied.                               |
| S4-2 | Server-side secrets within existing ADR-015                        | Approve deployment/config references, access/environment separation, rotation/revocation and failure behavior; any infrastructure/policy change needs its separate decision. |
| S4-3 | Prepared private register, manual workflow                         | Approve actual access/privacy review, procedure, retention and audit ownership; no automated integration.                                                                    |
| S4-4 | Separate send/delivery/human-ACK states; signed callbacks proposed | Approve event mapping, authenticated/replay-safe durable receipt, correlation and missing-callback reconciliation.                                                           |
| S4-5 | Stable intent identity and bounded retry                           | Approve numeric timeout/attempt/backoff/rate/cost/retention bounds, uncertainty and terminal/manual-review procedure.                                                        |
| S4-6 | Next-window review and independent containment                     | Approve remaining escalation, absence and delivery-failure handling consistent with scoped O1 and O3/O6/O7.                                                                  |
| S4-7 | Minimal safe metadata                                              | Approve exact envelope, severity mapping and no-sensitive-data validation; separate thresholds remain gated.                                                                 |
| S4-8 | Formal S4 decision, then consolidated O1 sign-off                  | Obtain dated attributable human evidence identifying exact accepted/revised clauses and controlled references. Neither approval is supplied here.                            |

Provider/account and security owners must supply their own attributable evidence; assigning a role does not invent its approval. O2–O8, OP1–OP9, actual Twelve Data quota/calendar/mapping evidence, ADR-025 CF/DR oracles including DR7, and Phase B tests remain independent blockers. `M4-GATE-001` stays `in_progress / BLOCKED`, `M4-CONTRACT-001` stays `blocked`, and receiver/Transaction publication stays inactive under ADR-023.

## 9. Human decision submission template

Only after evidence exists may the accountable maintainer submit an attributable decision. This list is a template, not a submitted approval:

- Decision package/version: `O1-S4-RESEND-REVIEW-2026-10-24-v1`.
- Exact S4-1 through S4-7 clauses accepted/revised/rejected, listed individually.
- Named provider/product/use and controlled account/plan/quota/cost evidence references.
- Controlled sender/domain authority and authentication evidence; no physical addresses or secrets.
- Deployment/config/access and ADR-015 compatibility approval references.
- Exact approved delivery/reconciliation/retry/failure/containment values and procedures.
- Private register access/manual-workflow acceptance and retention ownership.
- Actual approving human/role and GitHub comment creation timestamp/reference; never fabricated by Codex.
- Exclusions, unresolved items and re-review trigger.
- Separate explicit O1-S4 decision and final O1 sign-off; both **PENDING**.

Review, green CI or merge of this proposal is not acceptance of S4-1, O1-S4 or O1. Only direct human evidence may change those states. ADR-024 A1–A7 remains accepted architecture only; ADR-025 stays Proposed. No development-queue, financial, audit-policy, runtime, credential, DNS or deployment change is authorized by this record.

## References and evidence limits

Repository authorities: [O1-S4 packet](m4-o1-s4-review-packet.md), [O1 formal record](m4-o1-solo-maintainer-formal-decision-record.md), [Phase A closure register](m4-phase-a-closure-register.md), [ADR-013](../adr/ADR-013-solo-maintainer-merge-governance.md), [ADR-015](../adr/ADR-015-ed25519-key-ring-and-secret-management.md), [ADR-023](../adr/ADR-023-transaction-publication-activation-prerequisite.md).

Official public technical sources checked on **2026-10-24**: [pricing](https://resend.com/pricing), [idempotency](https://resend.com/docs/dashboard/emails/idempotency-keys), [webhook verification](https://resend.com/docs/webhooks/verify-webhooks-requests), [recipient event visibility](https://resend.com/changelog/webhook-event-visibility), [verified domains](https://resend.com/docs/dashboard/domains/introduction). These describe public capabilities only, not private-beta entitlement, a verified account, legal approval or deployed controls. No numerical account allowance, operating budget or credential is recorded.
