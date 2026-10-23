---
name: pr-evidence-review
description: Check protected-main freshness, Draft PR current-head tests, ADR-013 CI jobs, security evidence and review threads before human handoff.
---

# PR evidence review

This **instruction-only** skill supports a task already approved under `AGENTS.md` and the queue. It cannot merge to protected `main`, approve policies or exception extensions, deploy, activate receivers, or expand the task graph.

## Current-head evidence checklist

1. Read `AGENTS.md`, `.ai/agent-policy.json`, `.ai/development-queue.json`, the queue's `sourceOfTruth.executionPlan`, and task-specific accepted ADRs/approval records. Run `pnpm agent:validate` / `pnpm agent:next`; confirm scope and the resolved action.
2. Inspect the local working tree (staged, unstaged, untracked), commit range, current protected `main` SHA, feature branch SHA, PR state and base, and unresolved review conversations. Do not overwrite unrelated changes.
3. Verify the reviewed PR commit is exactly the head for which results were obtained; if new commits landed or base is stale, rerun required checks and re-review diff.
4. Confirm **all seven ADR-013 required jobs** are green on the current PR head: `frontend`, `backend`, `contracts-and-generation`, `database-integration`, `browser-e2e`, `compose-smoke`, `secrets`. Do not treat a workflow summary or older head as evidence. Inspect applicable additional GitGuardian/secrets results; distinguish gitleaks from GitGuardian rather than silently treating them as interchangeable.
5. Verify task-specific acceptance separately: clean pass/fail/flaky/retry counts, regression coverage, generated-contract/sqlc drift, migration/database tests, and current security exception/approval references. Passing CI must not waive stricter zero-retry or human acceptance requirements.
6. For each check, record link/run ID, conclusion, head SHA and coverage limitations. Keep historic failed runs and diagnostics, but redact credentials, cookies, provider payloads, customer details and financial transaction bodies.
7. Inspect final diff scope (`git diff --check` plus staged/unstaged/untracked files), required formatting including changed root documentation, and any pending review threads. Confirm no weakened tests, new skip/fixme, unauthorized secrets, permissions or deployment changes.
8. Prepare a Draft PR or an evidence-backed maintainer handoff **only if permitted**. The maintainer supplies the actual self-review and merge decision under ADR-013; never present agent evidence as an independent human approval.

## Blocker handling

- If branch/PR write access is denied: state the exact error and provide a reviewable local patch; do not write to protected `main` or forge a PR.
- If task scope, CI, current-head freshness, GitGuardian or review conversation status cannot be verified: label **NOT VERIFIED** and do not claim Ready.
- If M4-GATE-001 remains blocked or ADR-023 has no receiver: preserve zero claims, attempts, publisher invocations, retries, dead-letter transitions and ACKs; this workflow never activates the M4 worker.
- For security exceptions, preserve recorded expiry and follow human governance for changes; **no automatic renewal**.

**Handoff fields:** task/action; protected-main/base and PR head SHA; files changed; commands and exact results; seven current-head jobs; applicable security results; retry/flaky/skip evidence; review threads; unresolved issues; next allowed human action.
