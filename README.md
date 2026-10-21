# AI Portfolio Research & Monitoring Assistant

Bootstrap-only modular-monolith workspace. No authentication or financial business functionality is implemented in M0.

## Applications

- `apps/web`: Next.js App Router presentation foundation.
- `backend/cmd/api`: Go/Fiber operational API with liveness/readiness.
- `backend/cmd/worker`: Go worker lifecycle foundation; it runs no business jobs.
- `packages/api-contracts`: OpenAPI v1 source and generated TypeScript types.

Start with [Local Development](docs/engineering/local-development.md) and [Repository Structure](docs/architecture/repository-structure.md).

