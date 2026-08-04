# Configuration and Secrets

The backend validates environment, host/port, database URL, pool sizes, timeouts, worker heartbeat, and log level before startup. Development defaults are safe where possible; `DATABASE_URL` is always required. The frontend validates its public API base URL.

`.env.example` contains local placeholders. Real passwords, tokens, cookies, authorization headers, provider keys, private keys, or production connection strings must never be committed, logged, embedded in browser code, fixtures, screenshots, or tickets.

Development, CI, staging, and production use separate databases and secret scopes. The approved Authentication configuration contract, key-ring policy, and secret-manager rules are in [Authentication Deployment Contract](authentication-deployment-contract.md) and [TOKEN_SIGNING-v1](../policies/TOKEN_SIGNING-v1.md). No signing keys exist in M0.
