/Users/mac/.rvm/scripts/rvm:29: operation not permitted: ps
# Portfolio Foundation MVP — Decision Closure Specification

## Status and Purpose

**Status:** Approved architectural baseline for implementation planning.  
**Scope:** Authentication, FIFO lot semantics, decimal and rounding rules, price selection/freshness, and base-currency restriction.  
**Effective calculation-policy version:** `v1`.

This specification closes choices that were previously marked as open in ADR-008 through ADR-012. A later change is a product and architecture change: it requires a new ADR, a new policy/calculation version, migration/rebuild planning, and a compatibility decision. It must not be made implicitly in a handler, UI, provider adapter, or background job.

## 1. Authentication Policy — `AUTH-v1`

### Identity and account rules

- Authentication is email-and-password only in Phase 1.
- A user account is identified by an immutable opaque user ID. Email is a unique login attribute, not the primary key.
- Registration accepts an email after leading/trailing whitespace removal and lowercase normalization. The normalized value is unique. Email aliases are not collapsed; for example, provider-specific plus aliases remain distinct addresses.
- Registration requires a password of at least 12 characters. Password managers and pasted passwords are allowed. Complexity composition rules are not imposed because they discourage strong password-manager-generated passwords.
- Phase 1 does not require email verification, password reset, MFA, SSO, or social login. The UI and public wording must state this limitation only in internal release readiness material, not expose security implementation detail to attackers.
- Account status is `active` or `disabled`. Disabled accounts cannot create sessions or refresh tokens. Account deletion is out of scope; user data is retained according to the future retention policy.

### Password policy

- Passwords are hashed with Argon2id using parameters selected from the then-current OWASP guidance and recorded as hash metadata. The application never stores reversible passwords.
- A successful login may transparently rehash a password if its stored parameters are outdated.
- Registration, login, and password validation errors never log submitted credentials, password length, token values, or password hashes.

### Session and token rules

- Access token: signed JWT, 15-minute lifetime, bearer token used only for API authorization.
- Refresh token: opaque, cryptographically random token, 30-day idle lifetime and 90-day absolute lifetime from the original login.
- A refresh token is stored in the database only as a hash. Each refresh token belongs to a token family and is single-use.
- On every successful refresh, revoke/replace the presented token and issue a new token in the same family. The new token inherits the original 90-day absolute expiry.
- The browser retains the access token in memory only. It is never placed in local storage, session storage, IndexedDB, URL parameters, logs, analytics events, or persisted React state.
- The refresh token is sent only as a `Secure`, `HttpOnly`, `SameSite=Lax` cookie. Production requires HTTPS. The cookie is host-only and scoped to the authentication refresh path where deployment routing permits; otherwise it is scoped to the narrowest feasible auth path.
- Logout revokes the current refresh token. Logout-all-sessions, when added, revokes every currently active token family for the user.
- JWTs use EdDSA with Ed25519 keys. Tokens contain an issuer, audience, subject/user ID, issued-at, expiry, JWT ID, and key ID. Token verification requires all of these claims to be valid.
- Signing keys are managed through secret management, have a key ID, and support overlap during rotation. Private keys are never committed or logged.

### Abuse and failure policy

- Login is limited per normalized email and source IP: 5 failed attempts in 15 minutes per email, and 30 attempts in 15 minutes per source IP. Registration is limited to 5 attempts per source IP per hour. Refresh is limited to 20 attempts per session family per 15 minutes.
- Login failure always returns the same generic authentication error whether the account is absent, disabled, or the password is wrong.
- A reused refresh token, including a token replaced during rotation, is treated as suspected token theft. The system immediately revokes the entire token family, rejects the request, records a high-severity audit event, and requires a fresh login. It does not revoke unrelated families in Phase 1.
- Authenticated authorization defaults to deny. A later resource module must explicitly prove the principal has ownership/membership before access; it must not trust a client-supplied user or portfolio ID.

### Required audit actions

Record: registration success/failure, login success/failure, refresh success/failure, logout, token-family revocation, refresh-token reuse detection, and account-disabled rejection. Audit records include actor when known, request correlation ID, timestamp, result, and limited network/device metadata. They never include credentials or raw tokens.

## 2. FIFO Lot Semantics — `COST_BASIS-v1`

### Scope

FIFO is applied independently for each `(portfolio, asset)` pair. MVP supports long-only cash purchases and sales of assets priced in the portfolio's base currency. Short selling, margin, options, derivatives, stock splits, mergers, spin-offs, transfers, and tax-lot selection are out of scope.

### Transaction semantics

| Transaction type | FIFO/holding effect |
|---|---|
| `BUY` | Creates one acquisition lot for the purchased quantity. Lot total cost is quantity × unit price plus transaction fee. |
| `SELL` | Consumes the oldest remaining acquisition lots, in ascending effective timestamp and then immutable portfolio transaction sequence. Fee reduces sale proceeds. A sell cannot exceed available settled quantity. |
| `DIVIDEND` | Records a cash event only. It does not create an asset lot or change the asset quantity. Dividend reinvestment is recorded as two separate events: `DIVIDEND` and `BUY`. |
| `DEPOSIT` | Increases portfolio cash only; it does not affect an asset lot. |
| `WITHDRAWAL` | Decreases portfolio cash only; it does not affect an asset lot. |
| `FEE` | Decreases portfolio cash only, unless it is a fee attached to a `BUY` or `SELL`, in which case it is included in that transaction's cost/proceeds as above. |
| `ADJUSTMENT` | Disabled for ordinary users in MVP. It is reserved for a later governed operational process with a specific adjustment reason and audit approval. |

### Ordering and validity rules

- The transaction effective timestamp is the business ordering time. A server-assigned immutable portfolio-local sequence breaks same-timestamp ties.
- A backdated transaction is allowed only if it does not produce a negative quantity at any point when the entire ordered ledger is replayed. If it changes previous lots, the system rebuilds all later projections for that portfolio.
- A `SELL` is rejected if replayed FIFO availability is insufficient at its position in the ordered ledger. Negative positions are not represented in MVP.
- Quantity must be strictly positive for `BUY` and `SELL`. Price must be non-negative; MVP requires a strictly positive unit price for `BUY` and `SELL`. Fee is zero or positive.
- A correction uses the ADR-008 reversal-and-replacement relationship. The original transaction remains effective in history but its reversal neutralizes its financial impact. The replacement receives its own sequence and is recalculated as part of the whole ledger.

### Cost and gain calculation

- A buy lot records original quantity, remaining quantity, acquisition timestamp, source transaction ID/sequence, and total acquisition cost.
- For a sale, each consumed lot contributes its proportional remaining acquisition cost to the sale's FIFO cost basis. Sale proceeds equal quantity × sale unit price minus attached sale fee.
- Realized gain/loss equals sale proceeds minus consumed FIFO cost basis. This is a deterministic portfolio metric, not an AI calculation.
- Holding cost basis is the sum of remaining lot cost. Holding quantity is the sum of remaining lot quantity.
- Cash balances are a separate derived projection from cash-affecting transactions. They are not asset lots and are excluded from this initial asset holding model until the cash presentation policy is implemented.

### Projection persistence

The authoritative input is the ordered transaction ledger. Lot state and holding projections are derived/rebuildable. A holding calculation record stores `COST_BASIS-v1`, transaction sequence range, relevant transaction IDs, and rounding-policy version.

## 3. Decimal Precision and Rounding — `DECIMAL-v1`

### Canonical representations

| Semantic value | Transport and storage precision | Rule |
|---|---|---|
| Asset quantity | Decimal string; up to 12 fractional digits | Must be greater than zero where a trade quantity is required. |
| Unit price | Decimal string; up to 12 fractional digits | Must be greater than zero for MVP buys and sells. |
| Monetary input (gross amount/fee/dividend) | Decimal string; up to 12 fractional digits | Currency is mandatory and must equal portfolio base currency in MVP. |
| Monetary calculated output | Decimal, calculated to 18 fractional digits internally; persisted at 12 fractional digits | Display is rounded to the currency's standard minor unit. |
| Allocation percentage | Decimal, calculated/persisted at 6 fractional digits | Display is rounded to 2 fractional digits. |
| Intermediate calculation | Decimal with at least 18 fractional digits | Never serialized as float and never rounded before the designated output boundary. |

All numeric database columns must accommodate at least 38 significant digits and the listed fractional scale. API schemas express these values as strings with explicit decimal-pattern and maximum-scale validation.

### Rounding rule

- The sole rounding mode is **round half to even** (banker's rounding).
- Input values are validated but not re-rounded; values exceeding allowed fractional scale are rejected.
- Per-lot proportional cost for a sale is calculated with 18 internal fractional digits. The aggregate sale cost basis and sale proceeds are rounded once to 12 fractional digits.
- Portfolio valuation totals are accumulated from unrounded eligible holding values and rounded once to 12 fractional digits.
- Allocation is calculated from unrounded portfolio and position valuations, then rounded once to 6 fractional digits. The displayed allocation is not used for totals or comparison rules.
- Currency display formatting occurs only at the UI/export boundary, using each currency's ISO minor-unit convention; displayed rounding cannot alter persisted calculations.

### Reconciliation rule

Any remainder caused by proportional lot consumption stays with the remaining lot at internal precision. On final lot closure, any residual within 12 decimal places is assigned to that closing sale so that the total recognized cost equals the lot's original acquisition cost exactly at the persisted precision.

## 4. Price Selection and Freshness — `PRICE_SELECTION-v1`

### Supported market-data model

MVP uses one approved primary price-data provider and one normalized observation type: official daily closing price for supported US-listed equities and ETFs. Intraday pricing, multiple-provider fallback, manual price overrides, cryptocurrency, and non-US markets are out of scope.

The provider adapter records the provider's source identifier, reported trading date/time, retrieval time, currency, and validation status. An observation is eligible only when its status is `accepted`, currency is USD, asset identifier matches the recognized listing, and its market date is not after the requested valuation cutoff.

### Selection rule

- The dashboard valuation cutoff is the most recently completed US market close.
- For each asset, select the accepted official close with the latest market date at or before that cutoff.
- If multiple eligible observations exist for the same asset and market date, select the most recently retrieved accepted observation from the approved primary provider; retain every observation for provenance.
- There is no cross-provider fallback in MVP. Missing or rejected data produces an incomplete valuation; the system must not carry forward an older value as if it were current.
- Every selected price in a valuation records price-observation ID, source identifier, data-as-of time, retrieval time, and `PRICE_SELECTION-v1`.

### Freshness classification

| Condition at dashboard generation | Classification | Dashboard behavior |
|---|---|---|
| Selected close is for the most recently completed US trading day and was retrieved within 36 hours of that close | `fresh` | Include in total valuation. |
| Selected close is for the prior US trading day, or retrieval is later than 36 hours but no more than 72 hours after that close | `stale` | Include only as explicitly stale value; show warning and exclude portfolio from “fully current” state. |
| No eligible close, close is older than one prior trading day, retrieval is more than 72 hours late, or observation is rejected | `unavailable` | Do not value the holding or include it in total/allocation denominator. Mark valuation incomplete. |

US market holidays and weekends use the provider-supported US exchange calendar. A failed market-calendar lookup results in `unavailable`, not an inferred freshness classification.

## 5. Base-Currency Restriction — `CURRENCY_SCOPE-v1`

- MVP portfolio base currency is **USD only**.
- A portfolio's base currency is selected as USD during creation and is immutable.
- MVP accepts only US-listed equities and ETFs whose trading/price currency is USD.
- Every financial transaction in MVP must have USD as its transaction currency. Any non-USD amount, asset, price observation, fee, dividend, or cash event is rejected as unsupported.
- No foreign-exchange rate is stored, selected, inferred, or calculated in MVP.
- Existing transaction or price records cannot be converted by editing their currency. A future multi-currency release must introduce explicit FX observations and a new valuation policy/version.

This restriction deliberately favors trustworthy valuation over broad market coverage. It is removed only after ADR-012 is replaced by an approved multi-currency policy.

## 6. Implementation Gates

Before Phase 1 begins, implementers must confirm the deployment hostname/cookie topology can safely support the refresh-cookie rule and select a secrets-management mechanism for Ed25519 keys. Before transaction work begins, implementers must confirm the approved primary US price-data provider has contractual permission to provide and display official close data for the intended users.

No code may add an unsupported transaction type, asset class, currency, price source, rounding mode, or session behavior without a documented policy change.
