# M3 Primary US Price Provider Permission Gate

| Metadata                       | Value                                                                                                                                                      |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Task                           | `M3-GATE-001`                                                                                                                                              |
| Milestone                      | M3 — Transaction Ledger                                                                                                                                    |
| Status                         | `BLOCKED`                                                                                                                                                  |
| Policy source                  | [Decision Closure Specification / `PRICE_SELECTION-v1`](../planning/decision-closure-specification.md#4-price-selection-and-freshness--price_selection-v1) |
| Plan source                    | [M3 Transaction Ledger Foundation Execution Plan §2.4](../planning/transaction-ledger-foundation-execution-plan.md#24-price-provider-permission-gate)      |
| Protected-main base            | `3ed1d915a43365adfc3dfd515abd3ad55357be4c`                                                                                                                 |
| Evidence owner                 | Product / Legal                                                                                                                                            |
| Technical implementation owner | Engineering                                                                                                                                                |
| Gate review date               | 2026-08-29                                                                                                                                                 |

Engineering and Codex are not legal approvers. This record contains evidence
references and Engineering constraints only; it must not contain provider
credentials, confidential contract text, account or billing information, or
private legal documents.

## Purpose and decision boundary

Before transaction work begins, the Decision Closure Specification requires
confirmation that one approved primary US price-data provider has contractual
permission to retrieve and display official close data for the intended users.
M3's execution plan makes this a hard gate for `M3-CONTRACT-001`.

This document separates four distinct statements:

1. **Product decision:** which provider and user population Product selects.
2. **Legal/contractual permission evidence:** what actual terms, agreement, or
   approved legal interpretation permit.
3. **Engineering interpretation:** the technical restrictions that follow from
   verified evidence.
4. **Gate result:** the conservative result computed from the required rows.

Evidence statuses are `VERIFIED`, `MISSING`, `NOT_APPLICABLE`, and
`UNRESOLVED`. Only `VERIFIED` satisfies a required gate row.

## A. Product decision

The Product decision supplied for this review selects Twelve Data as the
primary official-close provider for a private beta. This is a Product decision
only; it does not establish contractual permission.

| Product decision               | Current value                                                            | Evidence status | Evidence reference                         |
| ------------------------------ | ------------------------------------------------------------------------ | --------------- | ------------------------------------------ |
| Provider legal/company name    | Twelve Data                                                              | VERIFIED        | Supplied Product decision dated 2026-08-29 |
| Provider product/plan name     | UNRESOLVED                                                               | UNRESOLVED      | Exact plan not supplied                    |
| Provider service/API name      | Twelve Data API (plan details unresolved)                                | UNRESOLVED      | Exact service/plan reference not supplied  |
| Primary-provider status        | Primary Official-Close Provider                                          | VERIFIED        | Supplied Product decision dated 2026-08-29 |
| Intended user population       | Project owner and explicitly invited authenticated beta users            | VERIFIED        | Supplied Product decision dated 2026-08-29 |
| Public commercial availability | OUT OF CURRENT SCOPE; separate licensing review required                 | VERIFIED        | Supplied Product decision dated 2026-08-29 |
| Product approver               | Thanasak Srisaeng                                                        | VERIFIED        | Supplied Product decision dated 2026-08-29 |
| Product approval date          | 2026-08-29                                                               | VERIFIED        | Supplied Product decision dated 2026-08-29 |
| Product approval evidence      | Human-supplied Product decision; controlled record location not supplied | VERIFIED        | Supplied Product decision dated 2026-08-29 |

The recorded intended population is the project owner and explicitly invited
authenticated beta users. Public commercial availability is outside the
current scope and requires a separate licensing review. Engineering must not
expand this audience or infer additional rights from repository visibility,
technical API access, or provider marketing material.

### Product requirements

These are Product requirements and restrictions, not Legal permission. Legal
must still verify that the selected Twelve Data product/plan permits the
required uses.

| Product requirement               | Decision state               | Recorded requirement                                                 |
| --------------------------------- | ---------------------------- | -------------------------------------------------------------------- |
| Server-side retrieval             | REQUIRED                     | Required for the approved MVP                                        |
| Authenticated in-app display      | REQUIRED                     | Required for the approved MVP                                        |
| Persistent observation storage    | REQUIRED                     | Required for provenance, deterministic replay, and valuation history |
| Historical observation retention  | REQUIRED                     | Duration is subject to provider/Legal terms                          |
| Derived portfolio valuation input | REQUIRED                     | Future valuation use; no calculation is implemented by M3            |
| Public raw-data API               | PROHIBITED BY PRODUCT POLICY | No public raw-data exposure                                          |
| CSV/raw-price export              | NOT APPROVED                 | Not approved by Product                                              |
| External redistribution           | PROHIBITED BY PRODUCT POLICY | Only approved authenticated in-app display is in scope               |
| Raw provider data sent to an LLM  | NOT APPROVED                 | No AI-provider redistribution                                        |
| Embeddings/vectorization          | NOT APPROVED                 | No vectorization                                                     |
| Model training                    | PROHIBITED                   | No training use                                                      |

## B. Legal and contractual permission evidence

### Approved data scope to be evidenced

The narrowly approved MVP scope from `PRICE_SELECTION-v1` is official **daily
closing price** data for supported US-listed `EQUITY` and `ETF` assets in `USD`.
This gate does not seek approval for intraday pricing, CRYPTO, FX, non-US
markets, multiple providers, manual prices, news, or fundamentals.

Twelve Data is selected by Product, but no Legal/contract-owner evidence is
recorded. The exact Twelve Data product/plan and the official-close semantics
remain unresolved.

### Retrieval rights

Actual provider evidence must confirm permission to retrieve official daily
close data programmatically through the selected product/API, for the intended
user population and deployment model.

| Retrieval requirement                               | Permission status | Evidence type | Evidence reference |
| --------------------------------------------------- | ----------------- | ------------- | ------------------ |
| Official daily close retrieval                      | UNRESOLVED        | Not supplied  | Not supplied       |
| Programmatic retrieval through selected API/product | UNRESOLVED        | Not supplied  | Not supplied       |
| Intended-user use                                   | UNRESOLVED        | Not supplied  | Not supplied       |
| Intended deployment-model use                       | UNRESOLVED        | Not supplied  | Not supplied       |

Acceptable evidence types include a contract, order form, subscription terms,
provider terms, license, written provider confirmation, or Legal memo. Public
source research may assist review but is not Legal approval and is not itself a
license grant.

### Display rights

Retrieval and display are independent requirements. Legal/contract evidence
must expressly cover display to the intended users inside AI Portfolio Research
Assistant.

| Display use                            | Permission status | Evidence reference |
| -------------------------------------- | ----------------- | ------------------ |
| Application screens                    | UNRESOLVED        | Not supplied       |
| Future portfolio valuation             | UNRESOLVED        | Not supplied       |
| Future historical views                | UNRESOLVED        | Not supplied       |
| Future exports                         | UNRESOLVED        | Not supplied       |
| Minimum MVP official-close display use | UNRESOLVED        | Not supplied       |

An unclear display right remains `UNRESOLVED`; Engineering must not treat API
access, a public endpoint, or a free tier as permission to display data.

### Derived calculation rights

This row anticipates later holding market value, portfolio valuation, and
allocation work. Those calculations are not implemented by M3.

| Derived-use question                                          | Status     | Evidence / rationale                               |
| ------------------------------------------------------------- | ---------- | -------------------------------------------------- |
| Official-close input to future derived portfolio calculations | UNRESOLVED | No provider terms or Legal interpretation supplied |

If Product/Legal later determines this is outside the selected provider's
license scope, record `NOT_APPLICABLE` with the approved rationale. It is not a
reason to infer permission now.

### Storage, retention, and redistribution

| Requirement                        | Status     | Evidence / restriction |
| ---------------------------------- | ---------- | ---------------------- |
| Raw observation storage permitted  | UNRESOLVED | Not supplied           |
| Maximum retention                  | UNRESOLVED | Not supplied           |
| Required deletion behavior         | UNRESOLVED | Not supplied           |
| Cache duration                     | UNRESOLVED | Not supplied           |
| Historical display permitted       | UNRESOLVED | Not supplied           |
| Derived-data persistence permitted | UNRESOLVED | Not supplied           |
| Raw data through public API        | UNRESOLVED | Not supplied           |
| Raw data in downloadable exports   | UNRESOLVED | Not supplied           |
| Client-side caching                | UNRESOLVED | Not supplied           |
| Redistribution restrictions        | UNRESOLVED | Not supplied           |

Terms that are silent are `UNRESOLVED` until Legal records an interpretation.

### Attribution and branding

| Requirement                            | Status     | Evidence / restriction |
| -------------------------------------- | ---------- | ---------------------- |
| Attribution required                   | UNRESOLVED | Not supplied           |
| Approved attribution wording/reference | UNRESOLVED | Not supplied           |
| Logo requirement                       | UNRESOLVED | Not supplied           |
| Source-link requirement                | UNRESOLVED | Not supplied           |
| Placement requirement                  | UNRESOLVED | Not supplied           |
| Branding restrictions                  | UNRESOLVED | Not supplied           |

No provider logo or copyrighted attribution asset is included in this
repository. A later Price/UI task must implement only the verified requirements.

### Rate and usage limits

These limits do not establish legal permission, but must be captured before a
future ingestion design is approved.

| Operational limit           | Status     | Evidence / value |
| --------------------------- | ---------- | ---------------- |
| Request quota               | UNRESOLVED | Not supplied     |
| Per-minute/day/month limits | UNRESOLVED | Not supplied     |
| Concurrency limits          | UNRESOLVED | Not supplied     |
| Asset-count limits          | UNRESOLVED | Not supplied     |
| Historical-depth limits     | UNRESOLVED | Not supplied     |
| Commercial-use restrictions | UNRESOLVED | Not supplied     |

### AI-specific data use

AI data use is future legal-risk evidence, not an M3 implementation feature.

| Future use                                | Current approval state | Evidence / restriction         |
| ----------------------------------------- | ---------------------- | ------------------------------ |
| Send provider data to an LLM/API provider | NOT APPROVED           | Not approved by Product policy |
| Use provider data in AI prompts           | NOT APPROVED           | Not approved by Product policy |
| Store provider data in AI-vendor logs     | NOT APPROVED           | Not approved by Product policy |
| Model training                            | PROHIBITED             | Prohibited by Product policy   |
| Embedding/vectorization                   | NOT APPROVED           | Not approved by Product policy |

`NOT APPROVED` is an Engineering default pending evidence, not an inference
that a provider contract forbids the use. Future AI redistribution is not a
blocking requirement for M3 if the required normal retrieval/display rights are
later verified.

## C. Engineering interpretation and restrictions

The following are Engineering defaults pending verified Product/Legal evidence:

- Do not select or integrate a provider.
- Do not retrieve, store, cache, display, export, redistribute, prompt with, or
  embed provider market data.
- Do not expose provider data through a public API or client-side cache.
- Treat provider credentials as server-side secrets; never commit them or expose
  them to browser code.
- Do not build ingestion, caching, retention, attribution, rate-limit, or UI
  behavior until verified evidence specifies the applicable constraints.

Provider credential requirements remain unresolved until a selected provider's
terms are reviewed:

| Credential requirement       | Status     | Recorded value                                        |
| ---------------------------- | ---------- | ----------------------------------------------------- |
| API-key classification       | UNRESOLVED | Exact Twelve Data product/plan not supplied           |
| Secret-storage requirement   | UNRESOLVED | Exact Twelve Data product/plan not supplied           |
| Rotation requirement         | UNRESOLVED | Exact Twelve Data product/plan not supplied           |
| Environment separation       | UNRESOLVED | Exact Twelve Data product/plan not supplied           |
| Client-side exposure allowed | UNRESOLVED | Not supplied; Engineering default is server-side only |

## D. Evidence references

The Product decision is recorded from the human-supplied decision dated
2026-08-29. No Legal/contract evidence reference has been supplied. When
evidence is available, reference it without copying confidential content.
Preferred forms are an official public terms URL, Product approval record,
internal Legal ticket ID, provider support email/ticket ID, or contract/document
title with a controlled location. For confidential material, record only:

`Evidence location: Internal restricted record <identifier>`

Public-source research must be labeled **Public-source research — not legal
approval** and paired with a Legal interpretation/approval record.

## Required approval table

| Requirement                            | Evidence                            | Owner           | Status     |
| -------------------------------------- | ----------------------------------- | --------------- | ---------- |
| Provider selected                      | Supplied Product decision           | Product         | VERIFIED   |
| Intended users defined                 | Supplied Product decision           | Product         | VERIFIED   |
| Official-close retrieval permitted     | No Legal/contract evidence supplied | Legal           | UNRESOLVED |
| Official-close display permitted       | No Legal/contract evidence supplied | Legal           | UNRESOLVED |
| Commercial/intended-user use permitted | No Legal/contract evidence supplied | Legal           | UNRESOLVED |
| Storage/retention rules known          | No Legal/contract evidence supplied | Legal           | UNRESOLVED |
| Attribution rules known                | No Legal/contract evidence supplied | Legal / Product | UNRESOLVED |
| Redistribution rules known             | No Legal/contract evidence supplied | Legal           | UNRESOLVED |
| Engineering constraints captured       | This document §C                    | Engineering     | VERIFIED   |

The mandatory rows—provider selected, intended users defined, retrieval
permitted, display permitted, intended-user/commercial use permitted,
storage/retention resolved, and attribution resolved—must all be `VERIFIED`
before the gate can be approved.

## Human approval

### Product approval

| Field                        | Value                                                                                     |
| ---------------------------- | ----------------------------------------------------------------------------------------- |
| Product approver name / role | Thanasak Srisaeng (as supplied Product approver)                                          |
| Decision                     | APPROVED                                                                                  |
| Date                         | 2026-08-29                                                                                |
| Evidence reference           | Human-supplied Product decision dated 2026-08-29; controlled record location not supplied |

### Legal/contract approval

| Field                               | Value        |
| ----------------------------------- | ------------ |
| Legal/contract approver name / role | Not supplied |
| Decision                            | PENDING      |
| Date                                | Not supplied |
| Evidence reference                  | Not supplied |

An AI-generated self-review is not Product or Legal approval. Codex must not
populate approver names or `APPROVED` without an actual human-approved source.

## Gate decision

**Gate Status: `BLOCKED`**

The gate is blocked because the following mandatory evidence is absent or
unresolved:

1. The exact Twelve Data product/plan and official-close semantics remain
   unresolved.
2. Legal/contract confirmation of official-close retrieval has not been
   recorded.
3. Legal/contract confirmation of display rights for the intended deployment
   and users has not been recorded.
4. Intended-user/commercial-use coverage, storage/retention requirements, and
   attribution requirements have not been resolved.
5. No controlled Legal/contract evidence references exist for the required
   decisions.

`M3-CONTRACT-001` is therefore **BLOCKED**. No provider integration, price
ingestion, price storage, or M4 projection work is authorized by this document.

## Two-phase workflow

### Phase A — Engineering preparation

This PR completes Phase A: it records the evidence model, repository-known
facts, missing Product/Legal inputs, and Engineering restrictions. The PR must
remain draft/open under the preferred governance strategy while the gate is
blocked.

### Phase B — Human evidence

1. Product's selected provider, intended users, and Product approval are
   recorded above from the supplied decision.
2. Legal/contract owner reviews the actual terms or agreement for the exact
   Twelve Data product/plan.
3. Human Legal decision plus controlled evidence references are supplied for
   retrieval, display, intended-user use, storage/retention, attribution, and
   redistribution.
4. This same document and PR are updated and re-reviewed.
5. Change the status to `APPROVED` only when every mandatory row is
   `VERIFIED`.

Do not open `M3-CONTRACT-001` before Phase B completes.

## Self-review

| Requirement                              | Status     |
| ---------------------------------------- | ---------- |
| M3 plan merged                           | COVERED    |
| Provider explicitly selected             | COVERED    |
| Intended users explicitly defined        | COVERED    |
| Product approval                         | COVERED    |
| Retrieval rights                         | UNRESOLVED |
| Display rights                           | UNRESOLVED |
| Intended-use rights                      | UNRESOLVED |
| Retention/storage                        | UNRESOLVED |
| Attribution                              | UNRESOLVED |
| Redistribution                           | UNRESOLVED |
| Engineering restrictions                 | COVERED    |
| Evidence references                      | MISSING    |
| No credentials committed                 | COVERED    |
| No confidential contract text            | COVERED    |
| No runtime changes                       | COVERED    |
| `M3-CONTRACT-001` blocked until approved | COVERED    |

**M3-GATE-001 Review: Blocked**
