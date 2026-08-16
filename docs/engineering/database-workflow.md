# Database, Goose, and sqlc Workflow

M0's no-op platform migration establishes the goose chain; goose may create its own version metadata table. M1 adds the approved Authentication foundation documented in [Authentication Database Ownership](../architecture/authentication-database.md). Feature migrations must name their owning module, document forward/rollback behavior and backfills, and never write another module's data without review.

sqlc reads module-owned queries from `backend/queries`. Platform output is generated under `backend/internal/platform/database/sqlcgen`; identity output is generated under `backend/internal/identity/infrastructure/database/sqlcgen`; Portfolio output is generated under `backend/internal/portfolio/infrastructure/database/sqlcgen`; and Asset output is generated under `backend/internal/asset/infrastructure/database/sqlcgen`. Generated structs omit JSON tags, output is committed, and CI rejects drift across every target.

Integration tests use `postgres-test` on port 5433 with synthetic data. Production data must never be copied into local or CI databases. Projection modules must document rebuild procedures before introducing derived tables.
