# Portfolio and Asset Database Ownership

## Ownership

| Table        | Owner     | Classification                              |
| ------------ | --------- | ------------------------------------------- |
| `portfolios` | Portfolio | Authoritative owner-scoped aggregate record |
| `assets`     | Asset     | System-managed canonical reference data     |

Portfolio infrastructure writes only `portfolios`. Asset infrastructure writes
only `assets`. The Portfolio foreign key to Identity's `users` table does not
permit Portfolio code to import Identity infrastructure or generated sqlc code.
No M2 table models a holding, position, transaction, price, valuation,
allocation, or other financial state.

## IDs, timestamps, and Portfolio lifecycle

Both tables use trusted application-generated UUID primary keys and `timestamptz`
fields, matching Identity persistence. Portfolio owner references use
`ON DELETE RESTRICT`; no user deletion cascade can remove Portfolio history.

`portfolios` stores display `name` and a database-generated `normalized_name`.
The lifecycle is constrained to `ACTIVE` or `ARCHIVED`:

- ACTIVE requires `archived_at IS NULL`.
- ARCHIVED requires `archived_at IS NOT NULL`.
- `base_currency` is constrained to `USD`.
- `updated_at` cannot precede `created_at`.

The partial unique index `portfolios_owner_normalized_active_uidx` is the final
concurrency authority for `(owner_user_id, normalized_name)` while status is
ACTIVE. Archived rows do not participate, which releases a name immediately
after archive. There is no Portfolio delete query.

## Normalization

The database is the canonical normalizer; applications submit display values
and must not use an independently computed key as an authority. M2 trims only
the documented ASCII whitespace set (space, tab, line feed, carriage return,
form feed, and vertical tab) at either end. It preserves internal whitespace,
punctuation, aliases, and Unicode code points. Portfolio normalization is the
database-generated lowercase trimmed display name. Asset symbol and exchange
normalization are database-generated uppercase trimmed display values. M2
introduces no Unicode normalization, transliteration, alias mapping, or ticker
mapping.

Future Go domain work must use the same display-validation boundary and rely on
the database uniqueness constraint for final authority.

## Assets and canonical identity

`assets` is user-independent and has no `owner_user_id`, `portfolio_id`, or
financial fields. It enforces:

- `asset_type IN ('EQUITY', 'ETF', 'CRYPTO')`;
- `currency = 'USD'`;
- unique `(normalized_symbol, normalized_exchange)`;
- CRYPTO rows require display and normalized exchange `CRYPTO`.

The Asset ordering index is on the public `symbol`/`exchange`/`asset_id` tuple,
matching the frozen API order and keyset continuation. The unique identity
constraint supports canonical lookup and bootstrap idempotency. M2 intentionally
adds no fuzzy-search extension or speculative search index; search is
deterministic case-insensitive symbol/display-name discovery over the
controlled catalog.

## sqlc primitives

Portfolio sqlc queries are owner-scoped create, get, list by status, active
name update, and active archive. Future application code uses the owner-scoped
read after an archive mutation to implement the reviewed idempotent HTTP
behavior. Database queries do not encode HTTP status mapping.

Asset sqlc queries are get, stable public-field keyset-compatible search, and
`BootstrapCanonicalAsset`. The bootstrap primitive is system-maintenance only:
it uses canonical identity as its upsert target, preserves an existing ID, may
refresh display metadata, and refuses to silently change type or currency.
There is no normal-user Asset mutation query.

## Catalog bootstrap

No approved initial production Asset list exists in the repository. Therefore
M2-DB-001 deliberately inserts no arbitrary production catalog records. The
versioned `BootstrapCanonicalAsset` query plus synthetic integration fixtures
provides the deterministic, idempotent bootstrap mechanism and verifies that
repeat execution does not create a second identity or change its ID.

An approved, versioned initial production catalog manifest remains required
before any deployment populates `assets`. It must identify the supported
canonical records and be executed through the system-maintenance primitive; it
must not be replaced by a market-data-provider import or a user API.

## Migration and rollback

Migration `00003_portfolio_asset_foundation.sql` creates only `portfolios`,
`assets`, and their constraints/indexes. It contains no seeds. Its down section
drops `assets` then `portfolios`, leaving Identity and Platform-owned data
untouched. CI verifies empty migration, current-main upgrade from migration 2,
and down/up reapplication.
