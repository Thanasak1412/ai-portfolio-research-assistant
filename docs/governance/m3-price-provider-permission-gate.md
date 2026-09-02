# M3 Primary US Price Provider Permission Gate

| Metadata                       | Value                                                                                                                                                      |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Task                           | `M3-GATE-001`                                                                                                                                              |
| Milestone                      | M3 — Transaction Ledger                                                                                                                                    |
| Status                         | `APPROVED`                                                                                                                                                 |
| Policy source                  | [Decision Closure Specification / `PRICE_SELECTION-v1`](../planning/decision-closure-specification.md#4-price-selection-and-freshness--price_selection-v1) |
| Plan source                    | [M3 Transaction Ledger Foundation Execution Plan §2.4](../planning/transaction-ledger-foundation-execution-plan.md#24-price-provider-permission-gate)      |
| Protected-main base            | `3ed1d915a43365adfc3dfd515abd3ad55357be4c`                                                                                                                 |
| Evidence owner                 | Product / Legal                                                                                                                                            |
| Technical implementation owner | Engineering                                                                                                                                                |
| Gate review date               | 2026-08-31                                                                                                                                                 |

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

| Product decision               | Current value                                                            | Evidence status | Evidence reference                                                  |
| ------------------------------ | ------------------------------------------------------------------------ | --------------- | ------------------------------------------------------------------- |
| Provider legal/company name    | Twelve Data                                                              | VERIFIED        | Supplied Product decision dated 2026-08-29                          |
| Provider product/plan name     | Venture subscription                                                     | VERIFIED        | Written Twelve Data Technical Support confirmation dated 2026-08-31 |
| Provider service/API name      | Time Series endpoint                                                     | VERIFIED        | Written Twelve Data Technical Support confirmation dated 2026-08-31 |
| Primary-provider status        | Primary Official-Close Provider                                          | VERIFIED        | Supplied Product decision dated 2026-08-29                          |
| Intended user population       | Project owner and explicitly invited authenticated beta users            | VERIFIED        | Supplied Product decision dated 2026-08-29                          |
| Public commercial availability | OUT OF CURRENT SCOPE; separate licensing review required                 | VERIFIED        | Supplied Product decision dated 2026-08-29                          |
| Product approver               | Thanasak Srisaeng                                                        | VERIFIED        | Supplied Product decision dated 2026-08-29                          |
| Product approval date          | 2026-08-29                                                               | VERIFIED        | Supplied Product decision dated 2026-08-29                          |
| Product approval evidence      | Human-supplied Product decision; controlled record location not supplied | VERIFIED        | Supplied Product decision dated 2026-08-29                          |

The recorded intended population is the project owner and explicitly invited
authenticated beta users. Public commercial availability is outside the
current scope and requires a separate licensing review. Engineering must not
expand this audience or infer additional rights from repository visibility,
technical API access, or provider marketing material.

### Product requirements

These are Product requirements and restrictions. The written Twelve Data
confirmation described below verifies the listed use case for the Venture plan;
it does not authorize use outside that scope.

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

The provider's written Technical Support confirmation dated 2026-08-31 states
that the Venture subscription is sufficient for the described use case. The
same confirmation, governed by Twelve Data's Terms of Use, confirms the
retrieval, storage, display, derived-analytics, retention, and attribution
conditions recorded below. It is written provider-contract evidence, not an
Engineering inference or a substitute for a different use case.

### Canonical Official Close v1

For the approved initial provider scope, the canonical observation is:

| Property          | Approved value                                                  | Evidence                                                                               |
| ----------------- | --------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| Provider          | Twelve Data                                                     | Product decision and written confirmation dated 2026-08-31                             |
| Endpoint          | Time Series                                                     | Written Technical Support confirmation dated 2026-08-31                                |
| Interval          | `1day`                                                          | Written Technical Support confirmation dated 2026-08-31                                |
| Adjustment        | `adjust=none`                                                   | Written Technical Support confirmation dated 2026-08-31                                |
| Canonical price   | Official, unadjusted `close` for the regular trading session    | Written Technical Support confirmation dated 2026-08-31                                |
| Exchange basis    | Instrument's primary listed exchange/MIC                        | Written Technical Support confirmation dated 2026-08-31                                |
| Market-data basis | 100% consolidated EOD market data published after session close | Written Technical Support confirmation dated 2026-08-31                                |
| Trading date      | Exchange-local trading date                                     | Approved Product decision; provider confirmation describes the applicable trading date |

Pre-market and post-market values are not canonical. The raw `close` remains
separate from any future `adjusted_close` field; the approved canonical request
uses `adjust=none` and must not silently substitute an adjusted value.

### Retrieval rights

Actual provider evidence must confirm permission to retrieve official daily
close data programmatically through the selected product/API, for the intended
user population and deployment model.

| Retrieval requirement                               | Permission status | Evidence type                 | Evidence reference                                            |
| --------------------------------------------------- | ----------------- | ----------------------------- | ------------------------------------------------------------- |
| Official daily close retrieval                      | VERIFIED          | Written provider confirmation | Twelve Data Technical Support confirmation, 2026-08-31        |
| Programmatic retrieval through selected API/product | VERIFIED          | Written provider confirmation | Venture use case includes server-side retrieval               |
| Intended-user use                                   | VERIFIED          | Written provider confirmation | Project owner and explicitly invited authenticated beta users |
| Intended deployment-model use                       | VERIFIED          | Written provider confirmation | Private authenticated in-app display; no public access        |

Acceptable evidence types include a contract, order form, subscription terms,
provider terms, license, written provider confirmation, or Legal memo. Public
source research may assist review but is not Legal approval and is not itself a
license grant.

### Display rights

Retrieval and display are independent requirements. Legal/contract evidence
must expressly cover display to the intended users inside AI Portfolio Research
Assistant.

| Display use                            | Permission status | Evidence reference                                                                            |
| -------------------------------------- | ----------------- | --------------------------------------------------------------------------------------------- |
| Application screens                    | VERIFIED          | Venture external-display rights for invited authenticated beta users                          |
| Future portfolio valuation             | VERIFIED          | Written confirmation permits valuation, returns, allocations, and research analytics          |
| Future historical views                | VERIFIED          | Persistent EOD observations and historical values are permitted during an active subscription |
| Future exports                         | NOT_APPLICABLE    | Product policy does not approve CSV/raw-price export                                          |
| Minimum MVP official-close display use | VERIFIED          | Written confirmation; attribution applies to every data or derived-chart display              |

An unclear display right remains `UNRESOLVED`; Engineering must not treat API
access, a public endpoint, or a free tier as permission to display data.

### Derived calculation rights

This row anticipates later holding market value, portfolio valuation, and
allocation work. Those calculations are not implemented by M3.

| Derived-use question                                          | Status   | Evidence / rationale                                                                                                |
| ------------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------- |
| Official-close input to future derived portfolio calculations | VERIFIED | Written confirmation permits portfolio valuation, returns, allocations, and research analytics from stored EOD data |

If Product/Legal later determines this is outside the selected provider's
license scope, record `NOT_APPLICABLE` with the approved rationale. It is not a
reason to infer permission now.

### Storage, retention, and redistribution

| Requirement                        | Status         | Evidence / restriction                                                                                                                           |
| ---------------------------------- | -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Raw observation storage permitted  | VERIFIED       | Permitted for the duration of an active subscription                                                                                             |
| Maximum retention                  | VERIFIED       | Raw EOD observations/historical values may be retained while the subscription is active                                                          |
| Required deletion behavior         | VERIFIED       | Delete all raw data within 30 days after subscription termination                                                                                |
| Cache duration                     | VERIFIED       | No additional short-term cache limit for EOD historical data during an active subscription                                                       |
| Historical display permitted       | VERIFIED       | Permitted for the approved authenticated private-beta use case during an active subscription                                                     |
| Derived-data persistence permitted | VERIFIED       | Non-reconstructable derived analytics may be retained after termination                                                                          |
| Raw data through public API        | VERIFIED       | Prohibited by Product policy and outside the confirmed use case                                                                                  |
| Raw data in downloadable exports   | NOT_APPLICABLE | Product policy does not approve CSV/raw-price export                                                                                             |
| Client-side caching                | UNRESOLVED     | Written confirmation permits authenticated display but does not separately define browser-cache behavior; no persistent client cache is approved |
| Redistribution restrictions        | VERIFIED       | No public access, raw-data API, or redistribution; use is limited to approved authenticated in-app display                                       |

Terms that are silent are `UNRESOLVED` until Legal records an interpretation.

### Attribution and branding

| Requirement                            | Status   | Evidence / restriction                                                                                                                                   |
| -------------------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Attribution required                   | VERIFIED | Required on every display of data or charts derived from it                                                                                              |
| Approved attribution wording/reference | VERIFIED | `Data provided by Twelve Data` or `Source: Twelve Data`                                                                                                  |
| Logo requirement                       | VERIFIED | Not required; if a logo is later used, follow Twelve Data Brand Guidelines without modification, recoloring, or implied endorsement                      |
| Source-link requirement                | VERIFIED | Attribution must include a dofollow link to the main Twelve Data website, subject to any later exchange-specific requirement communicated by Twelve Data |
| Placement requirement                  | VERIFIED | On every display of data or charts derived from it                                                                                                       |
| Branding restrictions                  | VERIFIED | Do not modify or recolor a logo or imply endorsement, partnership, or affiliation; no logo is planned for this version                                   |

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

The following constraints apply to any later Twelve Data integration:

- Use Twelve Data as the sole initial source of truth; do not add a fallback or
  reconciliation provider without a new approved decision.
- Implement the integration behind a provider abstraction so a future provider
  can be added without changing domain logic.
- Preserve provider name, provider symbol, exchange/MIC, currency, trading
  date, retrieval timestamp, and the raw provider observation for provenance.
- Do not expose provider data through a public API, raw export, or client-side
  cache.
- Do not use provider data in LLM prompts, embeddings, AI-vendor logs, or model
  training.
- Treat provider credentials as server-side secrets; never commit them or expose
  them to browser code.
- Implement attribution on every approved data or derived-chart display before
  that display ships.
- Enforce the raw-data deletion requirement if the subscription terminates.

Provider credential details remain an operational configuration concern for a
future provider-integration task. They do not expand the approved data-use
scope:

| Credential requirement       | Status     | Recorded value                                                  |
| ---------------------------- | ---------- | --------------------------------------------------------------- |
| API-key classification       | UNRESOLVED | Exact credential mechanics are deferred to provider integration |
| Secret-storage requirement   | VERIFIED   | Engineering policy: server-side secret only                     |
| Rotation requirement         | UNRESOLVED | Exact provider operational procedure not yet recorded           |
| Environment separation       | UNRESOLVED | Exact provider operational procedure not yet recorded           |
| Client-side exposure allowed | VERIFIED   | Prohibited by Engineering policy                                |

## D. Evidence references

The Product decision is recorded from the human-supplied decision dated
2026-08-29. The contractual evidence is an owner-held Gmail export titled
`Request for contractual usage-rights confirmation - US EOD portfolio
application`, sent by Twelve Data Technical Support on 2026-08-31. A companion
owner-held export with the same subject confirms Time Series semantics on the
same date. The exports are not committed to this repository.

The written confirmation identifies the current Twelve Data Terms of Use and
the Venture subscription as the contractual basis for the stated scope. Public
supporting references, which are not Legal approval on their own, are:

- <https://twelvedata.com/terms>
- <https://twelvedata.com/pricing-business>
- <https://support.twelvedata.com/en/articles/12647398-attribution-guidelines-for-using-twelve-data>
- <https://support.twelvedata.com/en/articles/9935903-us-equities-market-data>

When later evidence is available, reference it without copying confidential
content. Preferred forms are an official public terms URL, Product approval
record, internal Legal ticket ID, provider support email/ticket ID, or
contract/document title with a controlled location. For confidential material,
record only:

`Evidence location: Internal restricted record <identifier>`

Public-source research must be labeled **Public-source research — not legal
approval** and paired with a Legal interpretation/approval record.

## Required approval table

| Requirement                            | Evidence                                                                                                                                                                | Owner                     | Status   |
| -------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------- | -------- |
| Provider selected                      | Supplied Product decision                                                                                                                                               | Product                   | VERIFIED |
| Intended users defined                 | Supplied Product decision                                                                                                                                               | Product                   | VERIFIED |
| Official-close retrieval permitted     | Written Twelve Data Technical Support confirmation, 2026-08-31                                                                                                          | Provider / contract owner | VERIFIED |
| Official-close display permitted       | Written Twelve Data Technical Support confirmation, 2026-08-31                                                                                                          | Provider / contract owner | VERIFIED |
| Commercial/intended-user use permitted | Venture external-display rights for explicitly invited authenticated beta users                                                                                         | Provider / contract owner | VERIFIED |
| Storage/retention rules known          | Active-subscription retention; raw-data deletion within 30 days after termination                                                                                       | Provider / contract owner | VERIFIED |
| Attribution rules known                | Required textual attribution and dofollow main-Twelve-Data-website link on every data/derived-chart display; later exchange-specific requirements apply if communicated | Provider / contract owner | VERIFIED |
| Redistribution rules known             | No public access, raw-data API, raw export, or redistribution                                                                                                           | Product / provider        | VERIFIED |
| Engineering constraints captured       | This document §C                                                                                                                                                        | Engineering               | VERIFIED |

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

### Provider contractual confirmation

| Field                                | Value                                                                               |
| ------------------------------------ | ----------------------------------------------------------------------------------- |
| Provider confirmation contact / role | Artemis, Technical Support, Twelve Data                                             |
| Confirmation                         | VERIFIED for the stated Venture private-beta use case                               |
| Contractual basis                    | Current Twelve Data Terms of Use; written Twelve Data confirmation dated 2026-08-31 |
| Date                                 | 2026-08-31                                                                          |
| Evidence reference                   | Owner-held Twelve Data written contractual usage-rights confirmation                |

An AI-generated self-review is not Product or Legal approval. Codex must not
populate approver names or `APPROVED` without an actual human-approved source.

## Gate decision

**Gate Evidence Decision: `APPROVED`**

Every mandatory row is `VERIFIED`: Twelve Data and the intended private-beta
users are selected, and written provider-contract evidence records permitted
retrieval, display, intended-user use, storage/retention, attribution, and the
applicable redistribution restrictions. The evidence also fixes Canonical
Official Close v1 for the initial provider.

`M3-GATE-001` remains **PENDING MERGE**.

`M3-CONTRACT-001` remains **BLOCKED UNTIL PR #48 IS MERGED**. After this PR is
merged into protected `main`, `M3-CONTRACT-001` becomes unblocked under the
approved M3 task sequence. This approval does not itself authorize a provider
adapter, price ingestion, price storage implementation, or M4 projection work;
those remain subject to their separate approved tasks and constraints.

## Two-phase workflow

### Phase A — Engineering preparation

This PR initially recorded the evidence model and missing inputs. It now also
records the supplied Product decision and written provider-contract evidence.

### Phase B — Human evidence

This phase is complete for the stated Twelve Data Venture private-beta use
case. Any change to provider, plan, intended user population, deployment
model, redistribution, retention, or canonical-price semantics requires a new
evidence review before implementation proceeds under the changed scope.

## Self-review

| Requirement                                      | Status  |
| ------------------------------------------------ | ------- |
| M3 plan merged                                   | COVERED |
| Provider explicitly selected                     | COVERED |
| Intended users explicitly defined                | COVERED |
| Product approval                                 | COVERED |
| Retrieval rights                                 | COVERED |
| Display rights                                   | COVERED |
| Intended-use rights                              | COVERED |
| Retention/storage                                | COVERED |
| Attribution                                      | COVERED |
| Redistribution                                   | COVERED |
| Engineering restrictions                         | COVERED |
| Evidence references                              | COVERED |
| No credentials committed                         | COVERED |
| No confidential contract text                    | COVERED |
| No runtime changes                               | COVERED |
| `M3-CONTRACT-001` blocked until PR #48 is merged | COVERED |

**M3-GATE-001 Review: Ready for Review**
