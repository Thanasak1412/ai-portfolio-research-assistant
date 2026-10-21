# Draft PR: Refresh Codex workflow guidance and evidence-based testing

**Status: REVIEW PACK ONLY — no branch, PR, or merge created.**

## Summary
- Route agent instruction authority through the queue-selected execution plan, preserving ADR-013 and explicit human gates.
- Replace historical PR #63/#64 bootstrap labels with pointers to the live queue and verified GitHub state.
- Add reusable real-stack regression and PR-evidence review skills without new permissions.
- Add a task prompt template with scope, acceptance evidence, and stop conditions.
- Remove the Playwright healer's skip-to-green instruction, and make coverage generation stop after plan review until separately approved.
- Link the skills/template in the documentation index and add explicit changed-root-Markdown formatting guidance.

## Scope
Exactly nine instruction/documentation files. No application behavior, CI workflow, task queue, ADR, backend, database, cloud, provider or M4 receiver modifications.

## Governance / blockers
- Review-only: current queue selects `M4-GATE-001`; this workflow proposal is not an authorized queue transition or M4 implementation.
- ADR-013 requires a focused Draft PR, all seven current-head checks, self-review, resolved threads and human merge decision.
- `M4-GATE-001` remains BLOCKED; ADR-023 receiver inactive; `SECURITY_EXCEPTION-001` expires 2026-10-31.

## Verification of this offline patch
- Original six modified source blobs verified by Git object SHA against fetched current files.
- Patch `git apply --check` and application/re-comparison passed on an isolated snapshot.
- Scope=9, `git diff --check`, TOML parse, skill frontmatter, new relative links and anti-skip invariants passed.
- NOT RUN: repository's `pnpm agent:test`, `pnpm agent:validate`, Playwright, seven GitHub CI jobs, GitGuardian, source repository branch permissions.

## Self-review checklist (maintainer to complete after a real PR)
- [ ] New branch is in authorized task scope; checked live GitHub protected-main and existing changes
- [ ] Checked full diff, links, Markdown formatting, skill discovery and task-specific acceptance
- [ ] Validated all seven CI jobs on exact current PR head, plus applicable security checks
- [ ] Inspected skipped/flaky/retried tests and unresolved review threads
- [ ] Human decision/approval recorded with correct scope; no unauthorized M4 activation
- [ ] Merge into `main` performed only by authorized maintainer
