---
name: real-stack-regression
description: Select and verify repository Auth/M2/M3 real-stack regression suites using isolated fixtures, approved HTTPS topology, and safe failure evidence.
---

# Real-stack regression

Use this **instruction-only** skill for a task already permitted by `AGENTS.md`, `.ai/agent-policy.json`, the queue-selected execution plan, and its governing ADRs. It does not authorize implementation, policy changes, external infrastructure or M4 receiver activation.

## Before any run

1. Inspect `git status --short` (including untracked files), record protected-main/task head and preserve unrelated work.
2. Run `pnpm agent:validate` and `pnpm agent:next`; confirm task, scope, and approved test requirements. If outside scope or a decision is blocked, stop rather than expanding scope.
3. Check actual Node, pnpm, Go, Docker/Compose and Playwright versions against repository manifests/CI. Check port ownership before touching containers.
4. Choose the smallest meaningful existing suite: focused unit tests for changed code, Authentication for session/security, M2 for Portfolio/Asset, M3 for Transaction ledger, plus the broader required regression on current head. Do not invent an M4 live suite while M4 gates are blocked.

## Real-stack safety

- Reference [`docs/engineering/authentication-local-https.md`](../../../docs/engineering/authentication-local-https.md), [`docs/engineering/m2-real-stack-e2e.md`](../../../docs/engineering/m2-real-stack-e2e.md), and [`docs/engineering/m3-real-stack-e2e.md`](../../../docs/engineering/m3-real-stack-e2e.md) for actual commands and fixture rules, instead of copying a stale script sequence into this skill.
- Browser Auth/M2/M3 must use `https://app.localhost:3443`, the approved Caddy same-origin proxy, real Next.js and Go API, and disposable `postgres-test` / `portfolio_test`. `localhost:3000` and `localhost:8080` are diagnostic ports, **not** the browser Authentication entrypoint.
- Before seeding, prove that scripts target only the isolated `postgres-test` service and fixture database. Do not modify persistent local, staging, or production data; never run destructive `down -v` against the normal development project. Keep fixture environment and TLS material ignored.
- Use repository setup/seed/reset scripts as documented. Reset only disposable E2E rate-limit data when directed. Never mock `/api/v1` responses for tests claiming real integration.
- Respect approved HTTPS proxy attestation, CSRF Origin/X-Requested-With, cookie, no-CORS, provider-data and ADR-023 no-receiver boundaries. Do not introduce a fake publisher, broker, claims or ACKs.

## Failure classification and handoff

1. Capture exact test command, suite, commit SHA, environment/fixture identity, counts (passed, failed, flaky, retries, skipped), and whether tests ran on real HTTPS and PostgreSQL.
2. Diagnose each failure as **application defect**, **test defect**, **infrastructure/environment issue**, or **unresolved policy/design decision**; include bounded sanitized evidence. Missing Docker/Chromium/proxy is _not_ a passing application test.
3. Repair only when permitted by task scope, rerun focused tests, then required broader checks. Do not add `.skip`/`.fixme`, disable an assertion, hide existing failure evidence or retry until green.
4. Preserve no-secret screenshots/traces/logs only where explicitly safe; avoid cookies, bearer headers, provider payloads, holdings, monetary amounts and transaction bodies in shared artifacts.
5. Report blockers with the next permitted action. Stop before policy changes, M4 activation, destructive data operations or human approval gates.

**Success evidence:** Required suites passed on current head with task-specific cleanliness (zero retry/flaky where required), correct isolated real-stack topology, documented failure history, and unchanged governance boundaries. A passing local test does not establish remote CI or deployment readiness.
