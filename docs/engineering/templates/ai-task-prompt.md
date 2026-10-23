# AI Task Prompt Template

Use this template only for a task already permitted by the repository queue and governing human approvals. A filled-in template is **not** itself authorization to change the task queue, ADRs, secrets, infrastructure, production state, or M4 receiver activation.

## Task: <ID and title>

### Context

- Repository:
- Protected-main SHA (current verified):
- Existing branch / PR, if any:
- Queue task, status and `pnpm agent:next` resolver result:
- Queue-selected execution plan, ADRs and attributable approval references:
- Existing staged/unstaged/untracked changes to preserve:

### Objective

Describe the independently observable outcome and the reason it belongs to the authorized task.

### Scope

- Allowed files/components:
- Excluded files/components:
- Existing implementation to preserve:
- Authorized actions and delivery boundary:
- Stop/ask for maintainer if scope expansion is required:

### Functional requirements

Document required behavior, validation and failure behavior.

### Business rules

Reference accepted policies and exact acceptance vectors, not guessed results. Identify unresolved decisions explicitly and stop the affected work rather than inventing values.

### Constraints

State architecture, auth/security, sensitive-data, test isolation and backward-compatibility limits. Treat proposals as evidence, not automatic approval. Preserve unrelated local changes, all existing coverage and required CI.

### Acceptance criteria

Define testable outcomes, negative cases, edge cases and **zero-flaky/zero-retry requirements where applicable**.

### Verification requirements

- Focused static/unit tests and formatting:
- Real PostgreSQL integration and browser HTTPS suites as applicable:
- Contract/sqlc generation and diff:
- Secrets/security checks (including GitGuardian when required):
- Seven ADR-013 required current-head CI jobs:
- Task-specific clean-run requirements and evidence artifacts:

### Delivery and stop conditions

Report protected-main/base SHA, task branch and PR head SHA, changed paths, exact commands and outcomes, unresolved findings/risks and next permitted action. Open/update a **Draft PR** only where authorized; never merge or claim approval by an agent.

Stop at human review, policy/ADR/secret/permission/deployment gates; do not activate the M4 receiver or change M4 queue status without the separately accepted gate evidence.
