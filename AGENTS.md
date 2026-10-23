# AGENTS.md — Continuous Development Operating Contract

This repository permits AI-assisted implementation only inside the governance and task boundaries below.

## Sources of truth

Read these before making changes, in this order:

1. `docs/adr/ADR-013-solo-maintainer-merge-governance.md`
2. The approved task-specific ADRs/policies referenced by the current task.
3. `.ai/agent-policy.json` and `.ai/development-queue.json` for permitted actions, active task, and its `sourceOfTruth.executionPlan`.
4. The execution plan selected by that queue reference; do not assume an earlier milestone's plan still governs the active task.
5. Existing implementation and repository conventions.

Repository workflow references: [continuous-development guide](docs/engineering/ai-continuous-development-agent.md), [task prompt template](docs/engineering/templates/ai-task-prompt.md), [real-stack regression skill](.agents/skills/real-stack-regression/SKILL.md), and [PR evidence review skill](.agents/skills/pr-evidence-review/SKILL.md). These are reusable instructions, not independent permissions.

If these sources conflict, stop the affected work and report the conflict. Do not hide an architecture or policy choice in code.

## Operating loop

For every run:

1. Inspect `git status --short` and existing staged, unstaged, and untracked changes; preserve unrelated work and avoid destructive cleanup.
2. Verify installed Node/pnpm/Go versions against the repository manifests and toolchain constraints rather than copied example versions.
3. Read `.ai/agent-policy.json` and `.ai/development-queue.json`; check the queue-selected execution plan and approved task-specific ADRs.
4. Run `pnpm agent:validate` and `pnpm agent:next`; read the result as a _candidate action_, not new authorization.
5. Inspect the current protected `main`, task branch, PR, review conversations, current-head CI, and task-specific acceptance evidence before editing.
6. Work only on the returned task/action and permitted paths; stop for missing approval, mismatched live state, or scope expansion.
7. Run focused checks first, then applicable integration/browser checks; classify failures before repair.
8. Inspect `git diff --check`, committed/staged/unstaged/untracked changes, generated artifacts, and secrets before committing.
9. Push only to the task branch; open/update a Draft PR only when task scope and access permit it.
10. Inspect the final PR head, seven CI jobs, additional security checks and review threads; never relabel a flaky/retried test as clean.
11. Stop at every human gate, including maintainer review and merge.

Never skip a merge dependency because a downstream branch happens to compile.

## Automatically permitted

Subject to task scope, an agent may inspect repository/PR/CI evidence, create or update a non-protected task branch, implement an already-approved task, run tests, commit and push task-scoped changes, open/update a Draft PR, merge protected `main` into a feature branch, rerun CI, collect failure evidence, and update PR evidence.

## Human gates

Stop and require the maintainer before:

- merging any PR into protected `main`;
- accepting/rejecting/superseding/materially changing an ADR;
- changing the approved task graph or starting a task whose dependency is not merged;
- accepting or extending a security exception/risk;
- lowering, bypassing, suppressing, or weakening a required CI/security/test gate;
- changing secrets, permissions, branch protection, or deployment environments;
- destructive database/data operations;
- production deployment;
- introducing unapproved external infrastructure/provider behavior.

## Forbidden shortcuts

Do not force-push/write directly to protected `main`, weaken tests to get green, mask flakes with retries, fake successful integrations, suppress vulnerabilities without an approved decision, expose secrets, rewrite preserved task history without approval, or mix unrelated refactoring into the current task.

## CI interpretation

ADR-013 requires all seven protected checks:

- `frontend`
- `backend`
- `contracts-and-generation`
- `database-integration`
- `browser-e2e`
- `compose-smoke`
- `secrets`

A green job is insufficient when a task-specific gate reports a retry/flaky result that its acceptance criteria forbid. Preserve historical failures and retry counts. The `contracts-and-generation` job runs control-plane tests; it does not independently grant implementation authority. For root Markdown/skills not covered by `pnpm format:check` (currently scoped to `apps/web`), run explicit formatting and frontmatter checks on changed files.

## Reporting

At the end of a run report the task/action, branch/head, files changed, exact verification results, CI/PR state, retained failure evidence, blockers, and next permitted action. Clearly separate facts from hypotheses.
