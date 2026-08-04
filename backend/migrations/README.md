# Migrations

M0 contains one no-op platform migration so goose can establish and verify its migration chain. Goose may create its own version metadata table.

Migration `00002_authentication_database_foundation.sql` owns the M1 database foundation. It creates identity-authoritative `users` and `refresh_sessions`, platform-owned append-only `audit_logs`, and platform-owned operational `auth_rate_limit_events`. It contains no credentials, token values, seed users, or financial tables.
