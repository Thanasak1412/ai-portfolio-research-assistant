# CI Quality Gates

The workflow at `.github/workflows/ci.yml` separates frontend, backend, contract/generation, database integration, browser E2E, Compose smoke, and secret-scanning jobs. It fails on format, lint/type/static-analysis, tests, builds, OpenAPI validation, generated drift, high-severity JavaScript audit findings, Go vulnerability findings, migration failure, or detected secrets.

The product is a dedicated GitHub repository. ADR-013 requires pull requests, self-review, resolved conversations, up-to-date branches, and all seven jobs; it intentionally requires zero independent approvals during the solo-maintainer phase. Dependabot covers npm, Go, actions, and Docker dependencies.
