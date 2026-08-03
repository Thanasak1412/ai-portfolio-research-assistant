# Logging, Correlation, and Health

Development uses human-readable structured logs; staging/production use JSON. Request logs include method, matched path, status, latency, and correlation ID. Startup/shutdown, readiness failures, database pool failures, and worker lifecycle are logged.

`X-Correlation-ID` is accepted only when it matches the documented safe character/length policy; otherwise the API generates one. It is returned on liveness, readiness, and errors.

`/api/v1/health/live` reports process liveness. `/api/v1/health/ready` checks PostgreSQL and returns 503 with the standard envelope when unavailable. Neither endpoint reports secrets or connection details.

Passwords, tokens, cookies, authorization headers, private keys, database URLs, and complete request bodies are prohibited from logs.

