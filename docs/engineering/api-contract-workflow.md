# API Contract Workflow

`packages/api-contracts/openapi/v1.yaml` is the external HTTP source of truth. M0 defines only liveness/readiness, the standard error envelope, `X-Correlation-ID`, ISO-8601 timestamp, and decimal-string conventions.

Future API work must propose and approve OpenAPI first, run validation, generate consumer types, add contract tests, and then implement backend/frontend behavior. Breaking semantics require a new major version; fields are not repurposed. Authentication and product operations are intentionally absent.

Use `pnpm contract:lint`, `pnpm contract:generate`, and `pnpm contract:check`.

