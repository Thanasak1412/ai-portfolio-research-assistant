---
agent: default
description: Produce test coverage
---

Parameters:

- Task: the task to perform
- Seed file (optional): the seed file to use, defaults to `tests/auth-e2e/seed.spec.ts`
- Test plan file (optional): the test plan file to write, under `specs/` folder.

1. Call #playwright-test-planner subagent with prompt:

<plan>
  <task-text><!-- the task --></task-text>
  <seed-file><!-- path to seed file --></seed-file>
  <plan-file><!-- path to test plan file to generate --></plan-file>
</plan>

2. **Human plan-review gate:** Stop after planning. Present the exact plan file, scope, fixtures, real-HTTPS environment, negative cases, and expected assertions for explicit maintainer review. Do not generate tests merely because a plan exists. Do not infer approval from a green CI run or from this prompt. If approval has not been given for this plan and task, stop here and report the pending decision.

3. After approval, for each approved test case from the test plan file (1.1, 1.2, ...), one after another, not in parallel, call #playwright-test-generator subagent with prompt:

<generate>
  <test-suite><!-- Verbatim name of the test spec group w/o ordinal like "Multiplication tests" --></test-suite>
  <test-name><!-- Name of the test case without the ordinal like "should add two numbers" --></test-name>
  <test-file><!-- Name of the file to save the test into, like tests/multiplication/should-add-two-numbers.spec.ts --></test-file>
  <seed-file><!-- Seed file path from test plan --></seed-file>
  <body><!-- Test case content including steps and expectations --></body>
</generate>

4. Run approved, scoped test cases against the real isolated HTTPS stack, then any required broader regression. Record results and retries without suppression. If failures occur, classify them and request a separately scoped healing stage where authorized; do not invoke the healer automatically.

5. Only when separately authorized, call #playwright-test-healer for specific failing tests and exact evidence:

<heal>Investigate approved failing test cases, preserve failure evidence, and fix only proven task-scoped causes. Do not skip tests, add retries to conceal failures, or weaken required coverage. Stop at human gates.</heal>

Never replace the existing Auth/M2/M3 real-stack suites with mocks, change fixed fixture isolation, or treat a flaky/retried outcome as clean. See `docs/engineering/ai-continuous-development-agent.md` and `.agents/skills/real-stack-regression/SKILL.md`.
