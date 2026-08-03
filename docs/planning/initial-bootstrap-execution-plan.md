/Users/mac/.rvm/scripts/rvm:29: operation not permitted: ps
# Initial Project Bootstrap — Execution Plan

## Purpose

Prepare the engineering foundation for the AI Portfolio Research & Monitoring Assistant. This is milestone M0: architecture and delivery foundation only. It intentionally implements no authentication, portfolio, asset, transaction, price, holding, valuation, allocation, dashboard, AI, or document behavior.

## Inputs and Constraints

This plan is based on the [implementation planning](/Users/mac/Documents/AI%20Automation%202/docs/ai-portfolio-research/implementation-planning.md) and [decision closure specification](/Users/mac/Documents/AI%20Automation%202/docs/ai-portfolio-research/decision-closure-specification.md).

The existing workspace contains an unrelated LINE/lead application. The portfolio product must receive its own repository/workspace or be explicitly approved as a replacement; the products must not be combined by accident.

The bootstrap establishes delivery controls for the modular monolith, OpenAPI-first workflow, PostgreSQL/sqlc/goose governance, test pyramid, outbox-ready operations, and future `AUTH-v1` policy. It must not create product code, schemas, API operations, UI, jobs, Docker/configuration, CI configuration, or environment files.

## Execution Sequence

### 0. Close the repository decision

**Action:** Decide whether to create a dedicated repository or explicitly replace the current unrelated application.

**Decision:** A dedicated repository is recommended because it prevents unrelated dependencies, CI, secrets, and deployment history from mixing.

**Deliverable:** A decision record with repository name, ownership, default branch, branch protections, and the role of this current documentation workspace.

**Gate:** No repository mutation begins until this decision is approved.

### 1. Approve the workspace map and module boundary rules

**Action:** Publish the intended layout: deployable web/API/worker applications; versioned API-contract package; presentation-only design-system package; test-support package; backend modules under `internal`; and documentation, contract, integration, and end-to-end test areas.

**Rules:**

- A bounded context owns its internal implementation; other modules use only its public domain/application interface.
- No generic `common` or `utils` package; every shared package has a named responsibility.
- Authoritative, derived/rebuildable, and platform data retain the ownership model approved in the planning baseline.
- No domain module implementation or empty product scaffolding is created in M0.

**Gate:** Architecture owner approves the map and allowed dependency directions.

### 2. Freeze the engineering toolchain policy

**Action:** Select and document currently supported versions, installation method, and upgrade cadence for Go, Node.js, pnpm, PostgreSQL, Next.js/TypeScript, formatters, linters, static analysis, OpenAPI validation, contract generation, testing, dependency scanning, and secret scanning.

**Gate:** A compatibility matrix and owner for periodic updates exist. Tool versions are selected once centrally, not ad hoc by individual contributors.

### 3. Define quality gates and pull-request policy

**Action:** Specify mandatory change checks:

1. Formatting, linting, Go static analysis, and TypeScript type checks.
2. Unit tests when implementation begins.
3. Isolated PostgreSQL integration tests when persistence begins.
4. OpenAPI validation and compatibility checks for contract changes.
5. Dependency vulnerability/license checks and secret scanning.
6. Build verification for affected deployables.

**Policy:** Require passing checks, one reviewer, domain ownership review for ADR/contract/security-sensitive changes, and no direct protected-branch pushes.

**Gate:** The CI/PR policy is approved. Implementing CI workflow files is a separate execution action, not this planning task.

### 4. Establish the OpenAPI-first lifecycle

**Action:** Document the required order for future public API changes:

1. Propose the versioned OpenAPI change.
2. Review authorization, validation, error, idempotency, pagination, and compatibility effects.
3. Validate/lint the approved contract.
4. Generate or validate typed consumer artifacts.
5. Add contract tests.
6. Implement frontend and backend against the reviewed contract.

**Gate:** Contract owners and breaking-change/deprecation policy are published. No authentication operations or schemas are drafted in M0.

### 5. Establish database and migration governance

**Action:** Publish repository rules: PostgreSQL is the system of record; sqlc query definitions and repositories are module-owned; goose migrations need owner, review, forward plan, rollback expectation, and backfill note when relevant.

**Rules:**

- Cross-module writes require explicit architecture review.
- Integration tests use isolated disposable databases and synthetic data only.
- Local reset targets are explicit and recoverable where practical.
- A projection rebuild procedure must be designed before a derived table is introduced.

**Gate:** Migration/query checklist and database test-data policy are approved. No migration, query, table, or data model is created in M0.

### 6. Establish testing conventions

**Action:** Define ownership and entry/exit criteria for unit, integration, contract, end-to-end, and security tests.

**Required conventions:** pure domain tests where applicable; disposable dependency strategy for integration; contract conformance against OpenAPI; browser journey policy for E2E; synthetic fixtures only; test names that state behavior; and CI tiers that give fast feedback before slower suites.

**Gate:** The test pyramid, fixture policy, and future CI execution tiers are published. No product test cases are written in M0.

### 7. Establish observability and operational standards

**Action:** Define correlation ID, structured-log, metric, tracing, health/readiness, and runbook standards.

**Required standard:** Every future ingress request creates or propagates a correlation ID; logs include safe component/request/job/event identifiers; sensitive values are prohibited; traces connect request-to-worker paths when implemented; health and readiness have separate operational meanings.

**Metrics vocabulary:** request latency/error, job duration/error, outbox backlog, projection lag, and provider outcome. This is a naming/ownership standard only; no telemetry implementation occurs in M0.

**Gate:** Observability vocabulary and runbook outline for failed deploy, failed migration, job backlog, provider outage, secret rotation, and correlation-ID investigation are approved.

### 8. Establish secrets and environment governance

**Action:** Select/document secret-management access roles for development, CI, staging, and production; rotation ownership; compromised-secret response; and future environment-variable naming conventions.

**Non-negotiable rule:** Secrets must not enter source control, logs, client bundles, test fixtures, screenshots, or tickets.

**Future authentication gate:** Before Phase 1 implementation, deployment hostname/cookie topology must support `AUTH-v1`, and Ed25519 keys must be managed outside source control.

**Gate:** Security/operations owner approves the secret lifecycle. No secret, environment, or configuration file is created in this task.

### 9. Establish worker/event delivery standard

**Action:** Define the standard for future outbox consumers and scheduled jobs: durable job identity, idempotency key, retry count, exponential backoff with jitter, dead-letter/review state, correlation fields, at-least-once delivery, consumer deduplication, and documented per-aggregate ordering.

**Gate:** Backend/platform owners approve the worker operational contract. No worker, outbox, event, or job is implemented in M0.

### 10. Publish developer and release documentation

**Action:** Publish onboarding, supported-toolchain setup, local test execution, module-boundary guide, API-change guide, migration/query checklist, secrets guide, runbook index, and release/rollback decision procedure.

**Gate:** A new engineer can prepare a compliant environment and understand contribution/release rules without tribal knowledge.

## Deliverables and Ownership

| Deliverable | Responsible owner | Approval |
|---|---|---|
| Repository decision record | Technical lead and product owner | Required |
| Workspace/module-boundary guide | Architecture owner | Required |
| Toolchain compatibility matrix | Technical lead | Required |
| Quality-gate and PR policy | Technical lead and DevOps | Required |
| OpenAPI-first governance guide | Backend and frontend leads | Required |
| Database migration/query governance | Backend/data owner | Required |
| Test strategy and fixture policy | QA and engineering leads | Required |
| Observability standard/runbook index | Platform/operations owner | Required |
| Secrets/environment policy | Security and operations owner | Required |
| Worker/event-delivery standard | Backend/platform owner | Required |

## Dependencies and Decisions

| Dependency | Needed by | Status |
|---|---|---|
| Dedicated repository versus replacement of current app | All bootstrap execution | Open; first gate |
| Hosting/deployment topology | Future cookie scope, CI/CD, secret access | Open |
| Secret-management system | Future `AUTH-v1` key handling | Open |
| Supported tool versions | Developer setup and CI | Open |
| CI provider/branch-protection access | Quality gates | Open |
| Primary price-data provider and license | Price module only | Deferred; not a bootstrap blocker |

## M0 Completion Criteria

M0 is complete when the repository target is approved; module boundaries and prohibited dependencies are published; toolchain, quality, contract, database, test, observability, secrets, worker, release, and incident standards have owners; and a new engineer can set up a compliant environment from documentation.

M0 is not complete if any authentication or financial-domain behavior, schema, API, UI, job, provider integration, or calculation has been implemented. After approval, Phase 1 authentication may begin under the separate `AUTH-v1` plan; financial work remains scheduled by its later milestones.
