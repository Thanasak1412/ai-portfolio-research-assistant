# AUTH_RATE_LIMIT-v1

**Status:** Approved | **Effective:** 2026-08-04 | **Policy version:** v1

PostgreSQL is the approved shared store. `auth_rate_limit_events` contains only namespaced HMAC-derived key IDs, policy name/version, event timestamp, and expiry. Each check takes a transaction advisory lock derived from the policy/key digest, removes expired events for that key, counts the rolling window, and inserts only when permitted. This is atomic across API instances and avoids process-local enforcement or new broker/Redis infrastructure.

Keys are HMAC-SHA-256 with secret `AUTH_RATE_LIMIT_HMAC_KEY`, namespaced by policy/version: normalized-email failure, canonical source-IP login attempt, canonical source-IP registration attempt, and refresh family ID. Raw emails, IPs, and tokens are never table values or metric labels.

Limits: login email failures 5 rolling 15 minutes; login source-IP attempts 30 rolling 15 minutes; registration source-IP attempts 5 rolling one hour; refresh family attempts 20 rolling 15 minutes. Successful login removes only that email's failure events; IP attempts remain. Refresh limits apply after safe family resolution; unknown tokens remain generic failures and cannot target a known family.

When the store is unavailable, registration, login, and refresh fail closed with a safe `503 AUTH_RATE_LIMIT_UNAVAILABLE` envelope and correlation ID. The condition creates a secret-free operational log/metric. A worker cleanup removes expired events. `429` includes safe Retry-After derived from the earliest active event expiry. Required tests cover boundaries, expiry, concurrency, two-pool behavior, store failure, normalization, proxy spoofing, successful-login reset, and family isolation.
