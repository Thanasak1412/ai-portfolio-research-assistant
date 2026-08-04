# API Contract Workflow

`packages/api-contracts/openapi/v1.yaml` is the external HTTP source of truth. It defines liveness/readiness, the standard error envelope, `X-Correlation-ID`, ISO-8601 timestamp, decimal-string conventions, and the approved Authentication Phase 1 operations. Contract publication does not imply that an operation has runtime implementation.

Future API work must propose and approve OpenAPI first, run validation, generate consumer types, add contract tests, and then implement backend/frontend behavior. Breaking semantics require a new major version; fields are not repurposed. Portfolio and other product operations remain intentionally absent.

Use `pnpm contract:lint`, `pnpm contract:test`, `pnpm contract:generate`, `pnpm contract:typecheck`, and `pnpm contract:check`.
