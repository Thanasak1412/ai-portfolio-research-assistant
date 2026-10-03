import assert from "node:assert/strict";
import test from "node:test";
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

import {
  REQUIRED_CHECKS,
  readControlPlane,
  resolveNextAction,
  validateControlPlane,
} from "./agent-control-lib.mjs";

const installed = readControlPlane(
  fileURLToPath(new URL("..", import.meta.url)),
);

function fixture() {
  return {
    policy: structuredClone(installed.policy),
    queue: {
      schemaVersion: 1,
      temporaryGovernance: structuredClone(installed.queue.temporaryGovernance),
      activeTaskId: "M2-STABILITY-PR63",
      tasks: [
        {
          id: "M2-STABILITY-PR63",
          status: "in_progress",
          branch: "codex/m2-stability",
          pullRequest: 63,
          dependsOn: [],
          onReadyForMerge: "human_gate",
        },
        {
          id: "M3-BE-002",
          status: "blocked",
          branch: "codex/m3-be-002",
          pullRequest: 64,
          dependsOn: [{ task: "M2-STABILITY-PR63", requiredStatus: "merged" }],
          onReadyForMerge: "human_gate",
        },
        {
          id: "M3-BE-003",
          status: "blocked",
          branchHint: "codex/m3-be-003",
          dependsOn: [{ task: "M3-BE-002", requiredStatus: "merged" }],
          onReadyForMerge: "human_gate",
        },
        {
          id: "M3-FE-001",
          status: "blocked",
          dependsOn: [{ task: "M3-BE-003", requiredStatus: "merged" }],
          onReadyForMerge: "human_gate",
        },
        {
          id: "M3-FE-002",
          status: "blocked",
          dependsOn: [{ task: "M3-FE-001", requiredStatus: "merged" }],
          onReadyForMerge: "human_gate",
        },
        {
          id: "M3-E2E-001",
          status: "blocked",
          dependsOn: [{ task: "M3-FE-002", requiredStatus: "merged" }],
          onReadyForMerge: "human_gate",
        },
        {
          id: "M3-VERIFY-001",
          status: "blocked",
          dependsOn: [{ task: "M3-E2E-001", requiredStatus: "merged" }],
          onReadyForMerge: "human_gate",
        },
      ],
    },
  };
}

test("accepts the approved control plane", () => {
  assert.deepEqual(installed.policy.requiredChecks, REQUIRED_CHECKS);
  assert.deepEqual(validateControlPlane(fixture()), []);
});

test("rejects runnable tasks until every dependency is merged", () => {
  for (const status of [
    "ready",
    "in_progress",
    "waiting_ci",
    "needs_repair",
    "human_gate",
  ]) {
    const value = fixture();
    value.queue.tasks[0].status = "done";
    value.queue.tasks[1].status = status;
    value.queue.activeTaskId = status === "ready" ? null : "M3-BE-002";
    assert.match(validateControlPlane(value).join("\n"), /unmerged dependency/);
    assert.equal(resolveNextAction(value.queue).action, "stop_for_maintainer");
  }
});

test("selects a reviewed ready task only after its predecessor is merged", () => {
  const value = fixture();
  value.queue.tasks[0].status = "merged";
  value.queue.tasks[1].status = "ready";
  value.queue.activeTaskId = null;
  assert.deepEqual(validateControlPlane(value), []);
  assert.equal(resolveNextAction(value.queue).taskId, "M3-BE-002");
  assert.equal(resolveNextAction(value.queue).action, "start_task");
});

test("rejects weakened dependency status requirements", () => {
  for (const requiredStatus of ["done", "waiting_ci", undefined]) {
    const value = fixture();
    value.queue.tasks[1].dependsOn[0].requiredStatus = requiredStatus;
    assert.match(
      validateControlPlane(value).join("\n"),
      /requiredStatus must be merged/,
    );
    assert.equal(resolveNextAction(value.queue).action, "stop_for_maintainer");
  }
});

test("rejects unknown automatic permissions and missing safety policy", () => {
  const value = fixture();
  value.policy.autoAllowed.push("approve_anything");
  assert.match(validateControlPlane(value).join("\n"), /unknown action/);
  delete value.policy.autoAllowed;
  delete value.policy.forbidden;
  assert.match(
    validateControlPlane(value).join("\n"),
    /autoAllowed must be an array/,
  );
  assert.match(
    validateControlPlane(value).join("\n"),
    /forbidden must preserve/,
  );
});

test("rejects weakening branch, CI, and scope safety switches", () => {
  for (const section of ["branchPolicy", "ciPolicy", "scopePolicy"]) {
    for (const field of Object.keys(fixture().policy[section])) {
      const value = fixture();
      value.policy[section][field] = !value.policy[section][field];
      assert.match(
        validateControlPlane(value).join("\n"),
        new RegExp(`${section}.${field}`),
      );
    }
  }
});

test("requires a human merge gate even when the field is omitted", () => {
  const value = fixture();
  delete value.queue.tasks[0].onReadyForMerge;
  assert.match(
    validateControlPlane(value).join("\n"),
    /must stop at human_gate/,
  );
});

test("rejects protected branch targets with or without a PR", () => {
  for (const field of ["branch", "branchHint"]) {
    for (const branch of ["main", "refs/heads/main"]) {
      const value = fixture();
      delete value.queue.tasks[0].pullRequest;
      value.queue.tasks[0][field] = branch;
      assert.match(
        validateControlPlane(value).join("\n"),
        /branch cannot be main/,
      );
    }
  }
});

test("fails closed for malformed task and dependency shapes", () => {
  for (const task of [
    null,
    {},
    { id: "INVALID", dependsOn: {} },
    { id: "INVALID", dependsOn: [null] },
  ]) {
    const value = fixture();
    value.queue.tasks[0] = task;
    assert.ok(validateControlPlane(value).length > 0);
    assert.equal(resolveNextAction(value.queue).action, "stop_for_maintainer");
  }
  assert.ok(validateControlPlane(null).length > 0);
  assert.equal(resolveNextAction(null).action, "stop_for_maintainer");
});

test("rejects security-exception removal, alteration, or expiry extension", () => {
  for (const change of [
    (queue) => {
      queue.temporaryGovernance = [];
    },
    (queue) => {
      queue.temporaryGovernance[0].status = "remediated";
    },
    (queue) => {
      queue.temporaryGovernance[0].expiresOn = "2026-11-30";
    },
  ]) {
    const value = fixture();
    change(value.queue);
    assert.match(
      validateControlPlane(value).join("\n"),
      /SECURITY_EXCEPTION-001 must preserve/,
    );
  }
});

test("CLI rejects malformed JSON without printing its contents or a work decision", (t) => {
  const root = mkdtempSync(join(tmpdir(), "agent-control-test-"));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  mkdirSync(join(root, ".ai"));
  writeFileSync(
    join(root, ".ai/agent-policy.json"),
    "fixture-private-content-invalid-json",
  );
  for (const command of ["validate-agent-control.mjs", "next-agent-task.mjs"]) {
    const result = spawnSync(
      process.execPath,
      [fileURLToPath(new URL(command, import.meta.url))],
      { cwd: root, encoding: "utf8" },
    );
    assert.equal(result.status, 1);
    assert.equal(result.stdout, "");
    assert.match(result.stderr, /read failed/);
    assert.doesNotMatch(result.stderr, /fixture-private-content/);
  }
});

test("CLI refuses dependency bypass without emitting a work decision", (t) => {
  const root = mkdtempSync(join(tmpdir(), "agent-control-test-"));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  mkdirSync(join(root, ".ai"));
  const value = fixture();
  value.queue.tasks[1].status = "ready";
  writeFileSync(
    join(root, ".ai/agent-policy.json"),
    JSON.stringify(value.policy),
  );
  writeFileSync(
    join(root, ".ai/development-queue.json"),
    JSON.stringify(value.queue),
  );
  const result = spawnSync(
    process.execPath,
    [fileURLToPath(new URL("next-agent-task.mjs", import.meta.url))],
    { cwd: root, encoding: "utf8" },
  );
  assert.equal(result.status, 1);
  assert.equal(result.stdout, "");
  assert.match(result.stderr, /unmerged dependency/);
});

test("rejects drift from ADR-013 required checks", () => {
  const value = fixture();
  value.policy.requiredChecks.pop();
  assert.match(validateControlPlane(value).join("\n"), /exactly match ADR-013/);
});

test("rejects merge permission in autoAllowed", () => {
  const value = fixture();
  value.policy.autoAllowed.push("merge_protected_main");
  assert.match(validateControlPlane(value).join("\n"), /prohibited action/);
});

test("rejects multiple active tasks", () => {
  const value = fixture();
  value.queue.tasks[1].status = "waiting_ci";
  assert.match(
    validateControlPlane(value).join("\n"),
    /at most one active task/,
  );
});

test("rejects missing merge dependency", () => {
  const value = fixture();
  value.queue.tasks.find((task) => task.id === "M3-BE-003").dependsOn = [];
  assert.match(
    validateControlPlane(value).join("\n"),
    /must depend on M3-BE-002/,
  );
});

test("rejects dependency cycles", () => {
  const value = fixture();
  value.queue.tasks[0].dependsOn = [
    { task: "M3-VERIFY-001", requiredStatus: "merged" },
  ];
  assert.match(validateControlPlane(value).join("\n"), /dependency cycle/);
});

test("resolves current in-progress task", () => {
  const value = fixture();
  assert.deepEqual(resolveNextAction(value.queue), {
    taskId: "M2-STABILITY-PR63",
    status: "in_progress",
    action: "implement_or_diagnose",
    branch: "codex/m2-stability",
    pullRequest: 63,
  });
});

test("stops at a human gate", () => {
  const value = fixture();
  value.queue.tasks[0].status = "human_gate";
  assert.equal(resolveNextAction(value.queue).action, "stop_for_maintainer");
});

test("will not silently unblock a stale blocked task", () => {
  const value = fixture();
  value.queue.activeTaskId = null;
  value.queue.tasks[0].status = "merged";
  assert.equal(
    resolveNextAction(value.queue).action,
    "queue_state_stale_require_review_before_unblock",
  );
});
