# CI Quality Gates

The workflow at `.github/workflows/ci.yml` separates frontend, backend, contract/generation, database integration, browser E2E, Compose smoke, and secret-scanning jobs. It fails on format, lint/type/static-analysis, tests, builds, OpenAPI validation, generated drift, high-severity JavaScript audit findings, Go vulnerability findings, migration failure, or detected secrets.

This product directory must become the repository root for GitHub to discover `.github`. Branch protection should require all jobs and one review. Dependabot covers npm, Go, actions, and Docker dependencies.

