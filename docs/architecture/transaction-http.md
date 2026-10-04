# Transaction HTTP transport

`M3-BE-003` activates the four frozen Transaction operations under `/api/v1`:

| Method | Resource                                                             |
| ------ | -------------------------------------------------------------------- |
| POST   | `/portfolios/{portfolioId}/transactions`                             |
| GET    | `/portfolios/{portfolioId}/transactions`                             |
| GET    | `/portfolios/{portfolioId}/transactions/{transactionId}`             |
| POST   | `/portfolios/{portfolioId}/transactions/{transactionId}/corrections` |

The source of truth remains [OpenAPI](../../packages/api-contracts/openapi/v1.yaml).
There is no edit/delete, direct reversal, adjustment, frontend, or M4 operation.

## Authentication and composition

`cmd/api` builds Transaction HTTP with the existing pool, Portfolio ownership
binder, Asset lookup binder, Identity bearer middleware, and principal extractor.
Nested Transaction routes are mounted before Portfolio's prefix middleware so
Identity resolves the principal once. Platform remains a generic registrar.
No ownership headers or body fields are accepted. The application remains the
authority for ownership, financial eligibility, replay, and atomic persistence.

## Input and errors

Commands accept one UTF-8 JSON object of at most 8192 bytes (including a correction
wrapper), with exact field names, no duplicate keys, and no unknown fields.
Financial values are strings parsed with the domain's exact decimal constructors.
Kind-specific required/forbidden fields are enforced before application dispatch.
Optional text is preserved; absent, empty, and nonempty remain distinct. Null is
not a command string. No transport financial arithmetic is performed.

Both commands require exactly one `Idempotency-Key` header conforming to the
frozen 16–128 ASCII grammar. Reads do not require it. The existing platform
correlation ID is passed to application metadata and returned in error envelopes.
Errors use the frozen status/code mappings; SQL and submitted command contents
are never serialized or logged by this transport.

Collection operations use `PORTFOLIO_NOT_FOUND`; individual get/correct operations
use `TRANSACTION_NOT_FOUND` for inaccessible or unrepresentable Portfolio and
Transaction identities. This includes another owner's objects and mismatched
Portfolio/Transaction pairs.

Command timestamps use UTC-Z and at most six fractional digits. The application
checks future-time and replay validity. Read-filter timestamps use the same
lexical parser but allow future dates, with inclusive ordered range bounds.

## History and response mapping

History defaults to 50 rows, permits 1–100, and includes reversals by default.
The opaque cursor is `v1.` plus unpadded base64url of the canonical JSON tuple
`[effectiveAt, portfolioSequence, transactionId]`, each a string. Encoding uses
UTC RFC3339Nano (microsecond precision), canonical positive sequence digits, and
the immutable ID. Decoding re-encodes to reject noncanonical/trailing input.
The application retains descending time/sequence/ID ordering.

Responses expose only frozen public facts and nullable correction links. No
ownership, Asset eligibility snapshot, audit/outbox, provider, or projection data
is added. Decimal strings remain exact; an omitted trade fee is returned as zero.

A correction maps the committed application result's reversal, replacement, and
relationship. Since that application result does not carry the original fact,
the transport retrieves the immutable original through the same authenticated
application read boundary after commit. It uses the committed relationship,
not later outgoing corrections, to preserve identical replay output even when
the replacement is subsequently corrected. A failed original lookup returns a
generic internal error; retrying the same key remains safe and cannot add facts.

## Verification and publication boundary

Transport unit tests cover strict decoding, the field matrix, body/key limits,
time/cursor/query syntax, and all public error mappings. Real API tests live in
`transaction/infrastructure/database/http*_integration_test.go` so the existing
database-integration CI registration executes them. They construct real Identity,
Portfolio, Asset, and Transaction composition over an isolated synthetic schema;
no API endpoints are mocked and no credential values are logged.

ADR-022 and ADR-023 are unchanged. HTTP commands atomically persist ledger,
idempotency, audit, and pending outbox records through the existing application.
No approved receiver is configured: runtime publication remains inactive, with
no claims, attempts, retries, dead-letter transitions, or acknowledgements caused
by receiver absence. This task does not start M3-FE-001.
