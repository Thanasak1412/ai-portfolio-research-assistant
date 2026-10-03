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

## GitHub truth vs queue state

GitHub is authoritative for PR state, head SHA, current-head CI, review conversations, and protected-main SHA.

The queue is authoritative for intended work order and agent scope. If GitHub truth disagrees with queue state, stop and reconcile through normal review.

## CI integration

Control-plane tests and validation run inside the existing `contracts-and-generation` protected job. No eighth protected check is added.

## Flaky-test policy

A green job after retry is not automatically acceptable. Task acceptance can be stricter than the job badge.

For current M2 stabilization, the clean requirement is:

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

## Current bootstrap state

- `M2-STABILITY-PR63` — active.
- `M3-BE-002` / PR #64 — implementation complete, blocked on PR #63 merge.
- `M3-BE-003` — blocked until M3-BE-002 merge.
- remaining M3 tasks follow the execution-plan merge sequence.
- `SECURITY_EXCEPTION-001` is an accepted temporary risk through 2026-10-31 and cannot be extended automatically.

## Recommended external automation cadence

Prefer event-driven wakeups for push/PR/workflow/review/merge events, plus low-frequency reconciliation (for example hourly) to recover missed events. Avoid tight polling loops.

## Safety and operational limits

The external runner credential should not be able to bypass branch protection, change secrets, deploy production, administer the repository, or merge protected-main PRs.
