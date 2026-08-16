# Migrations

M0 contains one no-op platform migration so goose can establish and verify its migration chain. Goose may create its own version metadata table.

Migration `00002_authentication_database_foundation.sql` owns the M1 database foundation. It creates identity-authoritative `users` and `refresh_sessions`, platform-owned append-only `audit_logs`, and platform-owned operational `auth_rate_limit_events`. It contains no credentials, token values, seed users, or financial tables.

Migration `00003_portfolio_asset_foundation.sql` owns the M2 persistence
foundation. It creates Portfolio-owned `portfolios` and Asset-owned `assets`
with no seed catalog records, holding relation, financial data, or public
mutation mechanism. See [Portfolio and Asset Database Ownership](../../docs/architecture/portfolio-asset-database.md).
