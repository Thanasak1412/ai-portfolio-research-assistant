# Runbook Index

- Failed deployment: retain prior release, inspect build/check logs, and use correlation IDs for runtime failures.
- Failed migration: stop rollout; do not manually modify production schema; follow the owning migration's rollback/forward note.
- Database outage: liveness remains up, readiness becomes 503; restore dependency before serving traffic.
- Worker shutdown/backlog: stop with SIGTERM, allow graceful window, inspect lifecycle logs; future jobs require idempotent replay.
- Secret compromise: revoke/rotate in the approved manager, audit access, redeploy, and never paste values into incident tickets.
- Provider outage: deferred until a provider exists; future adapters must expose failure metrics without leaking credentials.

