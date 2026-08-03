# Database, Goose, and sqlc Workflow

M0 creates no application-owned table. Its no-op platform migration establishes the goose chain; goose may create its own version metadata table. Feature migrations must name their owning module, document forward/rollback behavior and backfills, and never write another module's data without review.

sqlc reads module-owned queries from `backend/queries`. The bootstrap health query proves generation and pgx compatibility without creating a domain repository. Generated output is committed and CI rejects drift.

Integration tests use `postgres-test` on port 5433 with synthetic data. Production data must never be copied into local or CI databases. Projection modules must document rebuild procedures before introducing derived tables.
