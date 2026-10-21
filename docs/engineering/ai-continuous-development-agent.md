# Continuous AI Development Agent

## Purpose

The repository can support long-running AI-assisted development without giving an agent authority to make product/security/governance decisions or merge directly to protected `main`.

The design separates approved work, operational state, agent authority, quality authority, and human authority.

## Architecture

```text
Approved plans / ADRs
        |
        v
.ai/development-queue.json ---- .ai/agent-policy.json
        |                              |
        +------------+-----------------+
                     v
              external agent
                     |
        inspect -> implement -> test
                     |
                     v
                Draft PR / CI
                 /          \
              fail          clean
               |              |
        task-scoped repair     v
               |          HUMAN GATE
               +----------> merge decision
                                  |
                             protected main
                                  |
                         unlock next task
```

The agent is deliberately not an autonomous merger.

## Repository commands

```bash
pnpm agent:test
pnpm agent:validate
pnpm agent:next
```

`agent:test` verifies validator/resolver behavior.

`agent:validate` protects governance invariants such as protected branch identity, ADR-013 required-check names, one active task, valid acyclic dependencies, M3 merge ordering, and separation of automatic vs human-only actions.

`agent:next` emits a JSON decision for an orchestrator. It never mutates the queue or GitHub.

Validation rejects runnable tasks with unmerged dependencies, non-merge dependency
requirements, unknown automatic permissions, missing human merge gates, weakened
safety switches, and changes to the recorded security exception. Malformed input
produces no work decision. File-read and JSON-parse failures do not print file contents.

These commands are local policy checks, not a sandbox or a grant of GitHub
permissions. A returned action is a candidate within the task scope: the external
runner must still apply the policy and inspect live GitHub evidence before acting.
They do not prove that a PR merged, approve a task-graph change, or extend a security
exception. The existing security audit remains authoritative for exception expiry
and advisory drift. Human-reviewed policy changes must update their regression tests.

## Reusable task and regression workflow

Use the [task prompt template](templates/ai-task-prompt.md) for an approved task and the repository skills [real-stack-regression](../../.agents/skills/real-stack-regression/SKILL.md) and [pr-evidence-review](../../.agents/skills/pr-evidence-review/SKILL.md) for their respective evidence workflows. Skills do not supersede `AGENTS.md`, queue scope, policies or human gates.

Before implementation, examine existing working-tree changes and toolchain versions; obtain the queue action and current GitHub evidence. Implement only inside scope. For failed checks, classify the cause as application defect, test defect, environment/infrastructure failure, or unresolved policy/decision. Preserve traces and previous CI failures rather than masking them with retries or skips. An environment failure is not proof that application behavior passed.

For docs and root skill files, root `pnpm format:check` currently covers the web package only. Explicitly run Prettier (or another approved formatter) over all newly changed Markdown files, validate skill YAML frontmatter, review relative links, and inspect `git diff --check`. This is additional local evidence, **not** a new protected CI check or replacement for the seven ADR-013 jobs.

At delivery, verify the current PR head SHA, protected-main freshness, seven required CI jobs, task-specific clean runs, additional applicable security evidence (including GitGuardian where required), outstanding review threads, and human review. Do not assume a past green head applies to a newer commit.

## GitHub truth vs queue state

GitHub is authoritative for PR state, head SHA, current-head CI, review conversations, and protected-main SHA.

The queue is authoritative for intended work order and agent scope. If GitHub truth disagrees with queue state, stop and reconcile through normal review.

## CI integration

Control-plane tests and validation run inside the existing `contracts-and-generation` protected job. No eighth protected check is added.

## Flaky-test policy

A green job after retry is not automatically acceptable. Task acceptance can be stricter than the job badge.

The historical M2 stability task required the following clean result (check each *current* task's own acceptance rather than blindly reusing this M2-specific vector):

```text
4 passed
0 failed
0 flaky
0 retries
```

## 24/7 external runner contract

A Codex/agent service can run continuously using GitHub webhooks or periodic polling:

```text
loop:
  sync protected main
  validate control plane
  resolve next task
  inspect PR + CI + review state

  if human gate:
      notify maintainer
      stop

  if waiting CI:
      observe only

  if task-scoped failure:
      reproduce
      preserve evidence
      repair within scope
      verify
      push

  if implementation incomplete:
      implement next approved slice
      verify
      push/update Draft PR

  repeat
```

The runner must not use an unmerged dependency as an implementation base.

## Historical bootstrap record (not live status)

During the initial control-plane bootstrap, the M2 stability PR #63 preceded M3 PR #64 and M3-BE-003. These entries describe **past state**, not the active task. For current execution, inspect `.ai/development-queue.json` and its `sourceOfTruth.executionPlan`, run `pnpm agent:validate` / `pnpm agent:next`, and reconcile with live GitHub evidence. Do not change queue tasks or promote readiness to fit an instruction-only workflow.

`SECURITY_EXCEPTION-001` is a temporary accepted risk expiring **2026-10-31**; neither these instructions nor passing tests extend or remediate it.

## Recommended external automation cadence

Prefer event-driven wakeups for push/PR/workflow/review/merge events, plus low-frequency reconciliation (for example hourly) to recover missed events. Avoid tight polling loops.

## Safety and operational limits

The external runner credential should not be able to bypass branch protection, change secrets, deploy production, administer the repository, or merge protected-main PRs.
