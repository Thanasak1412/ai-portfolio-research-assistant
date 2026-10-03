# AGENTS.md — Continuous Development Operating Contract

This repository permits AI-assisted implementation only inside the governance and task boundaries below.

## Sources of truth

Read these before making changes, in this order:

1. `docs/adr/ADR-013-solo-maintainer-merge-governance.md`
2. The approved task-specific ADRs/policies referenced by the current task.
3. `docs/planning/transaction-ledger-foundation-execution-plan.md` for M3 task ordering and scope.
4. `.ai/agent-policy.json`
5. `.ai/development-queue.json`
6. Existing implementation and repository conventions.

If these sources conflict, stop the affected work and report the conflict. Do not hide an architecture or policy choice in code.

## Operating loop

For every run:

1. Read `.ai/agent-policy.json` and `.ai/development-queue.json`.
2. Run `pnpm agent:validate`.
3. Run `pnpm agent:next`.
4. Work only on the returned task and action.
5. Inspect the current protected `main`, task branch, PR, review conversations, and current-head CI before editing.
6. Keep changes inside task scope.
7. Run focused checks first, then all applicable repository checks.
8. Push only to the task branch.
9. Open or update a Draft PR when permitted.
10. Diagnose failing CI and repair only when the root cause is inside the approved task scope.
11. Stop at every human gate.

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

A green job is insufficient when a task-specific gate reports a retry/flaky result that its acceptance criteria forbid. Preserve historical failures and retry counts.

## Reporting

At the end of a run report the task/action, branch/head, files changed, exact verification results, CI/PR state, retained failure evidence, blockers, and next permitted action. Clearly separate facts from hypotheses.
