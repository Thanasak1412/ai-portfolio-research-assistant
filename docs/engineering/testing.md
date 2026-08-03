# Testing Workflow

- Go unit tests verify configuration, HTTP behavior, correlation IDs, readiness failure, and worker cancellation.
- Tagged Go integration tests verify a disposable PostgreSQL connection.
- Vitest/Testing Library verifies the neutral frontend shell.
- Playwright verifies browser startup in Chromium.
- Redocly and generated-code drift checks verify the API contract.
- Docker Compose smoke checks verify process composition.

Tests use synthetic data and isolated dependencies. Feature tests begin with their owning milestone. Run the commands documented in [Local Development](local-development.md).

