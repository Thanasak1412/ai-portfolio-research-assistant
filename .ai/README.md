# AI continuous-development control plane

This directory defines the repository-side contract for a long-running AI development agent.

It does **not** itself call an AI provider and it does not contain credentials. An external Codex/agent runtime may use these files to decide what it is permitted to do.

## Files

- `agent-policy.json` — permissions, human gates, CI rules, and forbidden shortcuts.
- `development-queue.json` — current operational task queue and dependency graph.
- root `AGENTS.md` — instructions an implementation agent must follow.

## State model

The queue uses these task states:

- `ready`
- `in_progress`
- `waiting_ci`
- `needs_repair`
- `human_gate`
- `blocked`
- `merged`
- `done`

Approved plans/ADRs remain authoritative.

## Continuous loop

An external orchestrator should:

1. fetch protected `main`;
2. inspect `.ai/agent-policy.json` and `.ai/development-queue.json`;
3. run `pnpm agent:validate`;
4. run `pnpm agent:next`;
5. inspect GitHub truth for the returned task;
6. perform only the returned permitted action;
7. run verification;
8. push/update a Draft PR if allowed;
9. repeat while no human gate is reached.

A dependency is satisfied only by a merged predecessor, not by a green downstream branch.

## Human-in-the-loop boundary

The unattended window ends when merge, ADR/policy/security-risk decisions, scope expansion, secret/permission changes, destructive DB work, production deployment, or unapproved infrastructure are required.

## GitHub permissions for an external runner

Use least privilege:

- repository contents: read plus feature-branch write;
- pull requests: read/write;
- checks/actions: read, with rerun permission only if explicitly configured;
- no protected-branch bypass;
- no administration permission;
- no environment/secret write permission.

Never store an OpenAI or GitHub token in this repository.

## Current queue

At foundation creation, PR #63 is active. PR #64 is implementation-complete but blocked until PR #63 merges. M3-BE-003 remains blocked until M3-BE-002 merges.

Run:

```bash
pnpm agent:validate
pnpm agent:next
```
