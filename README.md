# AI Portfolio Research Assistant — Agent Workflow Review Pack

This zip contains a proposed nine-file instruction/documentation change set, reconstructed against the actual GitHub source blobs referenced in `VERIFICATION.json`. It is **not** a deployed or committed update.

## Files

- `agent-workflow-nine-files.patch`: minimal patch to apply against the verified repository baseline.
- `proposed/`: complete proposed contents of the nine changed files, for review.
- `PR_DESCRIPTION.md`: Draft PR summary and self-review checklist.
- `VERIFICATION.json`: exact offline checks and their limitations.

## Apply (only after maintainer scope approval)

From your repository root, preserve existing changes and confirm your branch is current, then run:

```sh
git status --short
git apply --check /path/to/agent-workflow-nine-files.patch
git apply /path/to/agent-workflow-nine-files.patch
pnpm exec prettier --check AGENTS.md .ai/README.md docs/README.md docs/engineering/ai-continuous-development-agent.md docs/engineering/templates/ai-task-prompt.md .agents/skills/real-stack-regression/SKILL.md .agents/skills/pr-evidence-review/SKILL.md apps/web/.codex/prompts/playwright-test-coverage.md
pnpm agent:test
pnpm agent:validate
pnpm agent:next
git diff --check
```

Do not open a non-Draft PR, edit `main`, bypass required checks, or merge automatically. Review the original authorizations, queue scope and current GitHub state first. No production infrastructure and no M4 receiver are touched by this package.
