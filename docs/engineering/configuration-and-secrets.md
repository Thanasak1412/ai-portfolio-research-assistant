# Configuration and Secrets

The backend validates environment, host/port, database URL, pool sizes, timeouts, worker heartbeat, and log level before startup. Development defaults are safe where possible; `DATABASE_URL` is always required. The frontend validates its public API base URL.

`.env.example` contains local placeholders. Real passwords, tokens, cookies, authorization headers, provider keys, private keys, or production connection strings must never be committed, logged, embedded in browser code, fixtures, screenshots, or tickets.

Development, CI, staging, and production use separate databases and secret scopes. Production uses an approved secret manager. Before Authentication Phase 1, owners must approve the deployment hostname/cookie topology and Ed25519 key storage/rotation mechanism. No signing keys exist in M0.

