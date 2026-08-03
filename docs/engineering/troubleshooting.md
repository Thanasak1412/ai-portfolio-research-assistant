# Troubleshooting

- Port 5432/5433/8080/3000 already used: stop the conflicting local process or adjust only local Compose port mappings.
- Readiness returns 503: check `docker compose ps postgres` and API logs; liveness can remain 200 during a database outage.
- pnpm/Corepack cannot write its user cache in a restricted environment: set `COREPACK_HOME` and cache directories to a writable workspace/temp location.
- sqlc drift: run `make sqlc-generate` and review generated changes; do not hand-edit generated files.
- OpenAPI drift: run `pnpm contract:generate`; change the OpenAPI source rather than generated types.
- Goose reports no pending migrations after `00001_platform_bootstrap.sql`: this is correct for M0; no application-owned table is created.
