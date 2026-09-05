# Migrations

M0 contains one no-op platform migration so goose can establish and verify its migration chain. Goose may create its own version metadata table.

Migration `00002_authentication_database_foundation.sql` owns the M1 database foundation. It creates identity-authoritative `users` and `refresh_sessions`, platform-owned append-only `audit_logs`, and platform-owned operational `auth_rate_limit_events`. It contains no credentials, token values, seed users, or financial tables.

Migration `00003_portfolio_asset_foundation.sql` owns the M2 persistence
foundation. It creates Portfolio-owned `portfolios` and Asset-owned `assets`
with no seed catalog records, holding relation, financial data, or public
mutation mechanism. See [Portfolio and Asset Database Ownership](../../docs/architecture/portfolio-asset-database.md).

Migration `00004_m3_platform_eventing.sql` extends the Platform-owned,
append-only `audit_logs` allowlist with safe M3 reference fields and creates
the Platform-owned `platform_outbox_events` and
`platform_outbox_streams` and `platform_consumer_deduplications` delivery
primitives. Outbox payloads contain only bounded role/UUID references and
their immutable Platform stream position. It creates no
Transaction ledger, idempotency, financial, provider, or projection table.
