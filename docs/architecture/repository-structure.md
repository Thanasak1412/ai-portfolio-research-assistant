# Repository Structure

| Path | Responsibility |
|---|---|
| `apps/web` | Next.js presentation, providers, transport client, neutral bootstrap route. |
| `backend/cmd/api` | API process composition and lifecycle only. |
| `backend/cmd/worker` | Worker process composition and lifecycle only. |
| `backend/internal/platform` | Named technical foundations: configuration, database, HTTP, logging, worker runtime. |
| `backend/migrations` | Goose migrations owned by feature milestones; empty in M0. |
| `backend/queries` | Module-owned sqlc inputs; M0 contains only a platform health query. |
| `packages/api-contracts` | Versioned OpenAPI source and generated TypeScript schema types. |
| `tests` | Future cross-application contract/integration/E2E assets. Application-local tests remain near their application. |
| `docs` | Actual architecture, engineering, operations, and ADR documentation. |
| `.github` | CI and dependency update policy for the future dedicated repository. |

Product domain directories are deliberately absent until their owning milestones start. See [Module Boundaries](module-boundaries.md).

