import { readFileSync } from "node:fs";
import { resolve } from "node:path";

export const REQUIRED_CHECKS = [
  "frontend",
  "backend",
  "contracts-and-generation",
  "database-integration",
  "browser-e2e",
  "compose-smoke",
  "secrets",
];

export const HUMAN_ONLY_ACTIONS = new Set([
  "merge_protected_main",
  "accept_or_change_adr",
  "change_task_graph",
  "start_task_with_unmerged_dependency",
  "accept_or_extend_security_exception",
  "weaken_required_gate",
  "change_secrets_or_permissions",
  "destructive_database_operation",
  "production_deploy",
  "introduce_unapproved_external_infrastructure",
]);

export const FORBIDDEN_AUTO_ACTIONS = new Set([
  "force_push_main",
  "direct_write_main",
  "mask_flake_with_retries",
  "weaken_test_to_green",
  "suppress_security_finding_without_decision",
  "log_secrets",
  "fake_successful_external_handoff",
  "mix_unrelated_refactor",
]);

const ALLOWED_STATUSES = new Set([
  "ready",
  "in_progress",
  "waiting_ci",
  "needs_repair",
  "human_gate",
  "blocked",
  "merged",
  "done",
]);

const AUTO_ACTIONS = new Set([
  "inspect_repository",
  "inspect_pr",
  "inspect_ci",
  "inspect_non_secret_artifacts",
  "edit_within_task_scope",
  "run_tests",
  "create_feature_branch",
  "commit_task_scoped_changes",
  "push_feature_branch",
  "open_or_update_draft_pr",
  "merge_main_into_feature_branch",
  "rerun_ci",
  "collect_failure_evidence",
  "update_pr_evidence",
]);

const ACTIVE_STATUSES = new Set([
  "in_progress",
  "waiting_ci",
  "needs_repair",
  "human_gate",
]);

function isRecord(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function mergedDependencies(task, tasksById) {
  return dependencyIds(task).every(
    (id) => tasksById.get(id)?.status === "merged",
  );
}

function sameSet(left, right) {
  const a = [...left].sort();
  const b = [...right].sort();
  return a.length === b.length && a.every((value, index) => value === b[index]);
}

function dependencyIds(task) {
  return (task.dependsOn ?? []).map((dependency) =>
    typeof dependency === "string" ? dependency : dependency.task,
  );
}

function assertNoCycles(tasksById) {
  const visiting = new Set();
  const visited = new Set();

  function visit(id) {
    if (visiting.has(id))
      throw new Error(`task dependency cycle includes ${id}`);
    if (visited.has(id)) return;

    visiting.add(id);
    const task = tasksById.get(id);
    for (const dependency of dependencyIds(task)) visit(dependency);
    visiting.delete(id);
    visited.add(id);
  }

  for (const id of tasksById.keys()) visit(id);
}

export function validateControlPlane(control) {
  const { policy, queue } = control ?? {};
  const errors = [];

  if (policy?.schemaVersion !== 1)
    errors.push("agent policy schemaVersion must be 1");

  if (policy?.protectedBranch !== "main") {
    errors.push("protected branch must remain main");
  }

  if (
    !Array.isArray(policy?.requiredChecks) ||
    !sameSet(policy.requiredChecks, REQUIRED_CHECKS)
  ) {
    errors.push("required checks must exactly match ADR-013 seven checks");
  }

  const autoAllowed = Array.isArray(policy?.autoAllowed)
    ? policy.autoAllowed
    : [];
  if (!Array.isArray(policy?.autoAllowed))
    errors.push("autoAllowed must be an array");
  for (const action of autoAllowed) {
    if (HUMAN_ONLY_ACTIONS.has(action) || FORBIDDEN_AUTO_ACTIONS.has(action)) {
      errors.push(`autoAllowed contains prohibited action ${action}`);
    } else if (!AUTO_ACTIONS.has(action)) {
      errors.push("autoAllowed contains an unknown action");
    }
  }

  if (!Array.isArray(policy?.humanRequired)) {
    errors.push("humanRequired must be an array");
  } else {
    for (const action of HUMAN_ONLY_ACTIONS) {
      if (!policy.humanRequired.includes(action)) {
        errors.push(`humanRequired is missing ${action}`);
      }
    }
  }

  if (
    !Array.isArray(policy?.forbidden) ||
    !sameSet(policy.forbidden, FORBIDDEN_AUTO_ACTIONS)
  ) {
    errors.push("forbidden must preserve all approved forbidden actions");
  }
  for (const [section, values] of Object.entries({
    branchPolicy: {
      preferMergeMainIntoFeature: true,
      preserveExistingTaskCommitShas: true,
      forcePushFeatureBranches: false,
    },
    ciPolicy: {
      allRequiredChecksMustPass: true,
      taskSpecificAcceptanceAlsoRequired: true,
      flakyOrRetryIsNotCleanWhenTaskForbidsIt: true,
      doNotHideHistoricalFailureEvidence: true,
    },
    scopePolicy: {
      scopeExpansionRequiresHuman: true,
      unrelatedRefactorForbidden: true,
    },
  })) {
    for (const [field, expected] of Object.entries(values)) {
      if (policy?.[section]?.[field] !== expected) {
        errors.push(`${section}.${field} must remain ${expected}`);
      }
    }
  }

  return [...errors, ...validateQueue(queue)];
}

function validateQueue(queue) {
  const errors = [];
  if (queue?.schemaVersion !== 1)
    errors.push("development queue schemaVersion must be 1");

  const tasks = Array.isArray(queue?.tasks) ? queue.tasks : [];
  if (tasks.length === 0) errors.push("development queue must contain tasks");

  // Check shapes before traversal so malformed input cannot select work or crash
  // partway through dependency validation.
  for (const task of tasks) {
    if (
      !isRecord(task) ||
      typeof task.id !== "string" ||
      task.id.length === 0
    ) {
      errors.push("every task must have a string id");
      continue;
    }
    if (!Array.isArray(task.dependsOn)) {
      errors.push(`task ${task.id} dependsOn must be an array`);
      continue;
    }
    for (const dependency of task.dependsOn) {
      if (typeof dependency === "string" && dependency.length > 0) continue;
      if (
        !isRecord(dependency) ||
        typeof dependency.task !== "string" ||
        !dependency.task
      ) {
        errors.push(`task ${task.id} has a malformed dependency`);
      } else if (dependency.requiredStatus !== "merged") {
        errors.push(`task ${task.id} dependency requiredStatus must be merged`);
      }
    }
  }
  if (errors.length > 0) return errors;

  const tasksById = new Map();
  for (const task of tasks) {
    if (!task?.id || typeof task.id !== "string") {
      errors.push("every task must have a string id");
      continue;
    }
    if (tasksById.has(task.id)) errors.push(`duplicate task id ${task.id}`);
    tasksById.set(task.id, task);

    if (!ALLOWED_STATUSES.has(task.status)) {
      errors.push(`task ${task.id} has unsupported status ${task.status}`);
    }

    if (
      [task.branch, task.branchHint].some(
        (branch) => branch === "main" || branch === "refs/heads/main",
      )
    ) {
      errors.push(`task ${task.id} branch cannot be main`);
    }

    if (task.status === "blocked" && (task.dependsOn ?? []).length === 0) {
      errors.push(`blocked task ${task.id} must declare a dependency`);
    }

    if (task.onReadyForMerge !== "human_gate") {
      errors.push(`task ${task.id} must stop at human_gate before merge`);
    }
  }

  for (const task of tasks) {
    for (const dependency of dependencyIds(task)) {
      if (!tasksById.has(dependency)) {
        errors.push(`task ${task.id} depends on missing task ${dependency}`);
      }
    }
    if (
      (task.status === "ready" || ACTIVE_STATUSES.has(task.status)) &&
      !mergedDependencies(task, tasksById)
    ) {
      errors.push(
        `task ${task.id} cannot be runnable with an unmerged dependency`,
      );
    }
  }

  if (
    tasks.every((task) => dependencyIds(task).every((id) => tasksById.has(id)))
  ) {
    try {
      assertNoCycles(tasksById);
    } catch (error) {
      errors.push(error.message);
    }
  }

  const active = tasks.filter((task) => ACTIVE_STATUSES.has(task.status));

  if (active.length > 1) {
    errors.push(
      `at most one active task is allowed, found ${active.map((task) => task.id).join(", ")}`,
    );
  }

  if (queue?.activeTaskId) {
    if (!tasksById.has(queue.activeTaskId)) {
      errors.push(`activeTaskId ${queue.activeTaskId} does not exist`);
    } else if (!active.some((task) => task.id === queue.activeTaskId)) {
      errors.push(`activeTaskId ${queue.activeTaskId} is not the active task`);
    }
  } else if (active.length > 0) {
    errors.push("activeTaskId is required while a task is active");
  }

  const requiredEdges = [
    ["M3-BE-002", "M2-STABILITY-PR63"],
    ["M3-BE-003", "M3-BE-002"],
    ["M3-FE-001", "M3-BE-003"],
    ["M3-FE-002", "M3-FE-001"],
    ["M3-E2E-001", "M3-FE-002"],
    ["M3-VERIFY-001", "M3-E2E-001"],
  ];

  for (const [taskId, dependencyId] of requiredEdges) {
    const task = tasksById.get(taskId);
    if (!task) {
      errors.push(`queue is missing required task ${taskId}`);
      continue;
    }
    if (!dependencyIds(task).includes(dependencyId)) {
      errors.push(`${taskId} must depend on ${dependencyId}`);
    }
  }

  const exceptions = Array.isArray(queue?.temporaryGovernance)
    ? queue.temporaryGovernance
    : [];
  const exception = exceptions.filter(
    (entry) => entry?.id === "SECURITY_EXCEPTION-001",
  );
  if (
    exception.length !== 1 ||
    exception[0].status !== "accepted_temporary_risk" ||
    exception[0].expiresOn !== "2026-10-31"
  ) {
    errors.push(
      "SECURITY_EXCEPTION-001 must preserve its accepted temporary risk status and 2026-10-31 expiry",
    );
  }

  return errors;
}

export function resolveNextAction(queue) {
  if (validateQueue(queue).length > 0) {
    return {
      taskId: null,
      status: "invalid_queue",
      action: "stop_for_maintainer",
      branch: null,
      pullRequest: null,
    };
  }
  const tasks = Array.isArray(queue?.tasks) ? queue.tasks : [];
  const tasksById = new Map(tasks.map((task) => [task.id, task]));
  const activeTask = queue?.activeTaskId
    ? tasksById.get(queue.activeTaskId)
    : undefined;

  if (activeTask) {
    const actionByStatus = {
      in_progress: "implement_or_diagnose",
      waiting_ci: "inspect_ci",
      needs_repair: "repair_verified_task_scoped_failure",
      human_gate: "stop_for_maintainer",
    };
    return {
      taskId: activeTask.id,
      status: activeTask.status,
      action: actionByStatus[activeTask.status] ?? "inspect",
      branch: activeTask.branch ?? activeTask.branchHint ?? null,
      pullRequest: activeTask.pullRequest ?? null,
    };
  }

  for (const task of tasks) {
    if (task.status === "ready") {
      return {
        taskId: task.id,
        status: task.status,
        action: "start_task",
        branch: task.branch ?? task.branchHint ?? null,
        pullRequest: task.pullRequest ?? null,
      };
    }

    if (task.status === "blocked") {
      const satisfied = mergedDependencies(task, tasksById);

      if (satisfied) {
        return {
          taskId: task.id,
          status: task.status,
          action: "queue_state_stale_require_review_before_unblock",
          branch: task.branch ?? task.branchHint ?? null,
          pullRequest: task.pullRequest ?? null,
        };
      }
    }
  }

  return {
    taskId: null,
    status: "idle",
    action: "no_permitted_work",
    branch: null,
    pullRequest: null,
  };
}

export function readControlPlane(root = process.cwd()) {
  const policy = JSON.parse(
    readFileSync(resolve(root, ".ai/agent-policy.json"), "utf8"),
  );
  const queue = JSON.parse(
    readFileSync(resolve(root, ".ai/development-queue.json"), "utf8"),
  );
  return { policy, queue };
}
