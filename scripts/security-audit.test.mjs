import assert from "node:assert/strict";
import test from "node:test";

import { evaluateAudit } from "./security-audit-lib.mjs";

const path =
  "apps__web>eslint-config-next>@next/eslint-plugin-next>fast-glob>micromatch>braces";

function fixture() {
  return {
    policy: {
      schemaVersion: 1,
      auditLevel: "high",
      exceptions: [
        {
          id: "SECURITY_EXCEPTION-001",
          advisory: "GHSA-vfj7-8cjw-p6xm",
          package: "braces",
          severity: "high",
          vulnerableVersions: "<=3.0.3",
          resolvedVersion: "3.0.3",
          manifestPath: "apps/web/package.json",
          topLevelDependency: "eslint-config-next",
          allowedPaths: [path],
          expiresOn: "2026-10-31",
        },
      ],
    },
    report: {
      advisories: {
        1240992: {
          github_advisory_id: "GHSA-vfj7-8cjw-p6xm",
          module_name: "braces",
          severity: "high",
          vulnerable_versions: "<=3.0.3",
          patched_versions: "<0.0.0",
          findings: [{ version: "3.0.3", paths: [path] }],
        },
      },
      muted: [],
      metadata: {
        vulnerabilities: { info: 0, low: 0, moderate: 0, high: 1, critical: 0 },
      },
    },
    manifests: new Map([
      [
        "apps/web/package.json",
        { devDependencies: { "eslint-config-next": "16.3.6" } },
      ],
    ]),
    today: new Date("2026-10-03T00:00:00Z"),
  };
}

function failures(change) {
  const input = fixture();
  change(input);
  return evaluateAudit(input).failures.join("; ");
}

test("accepts only the exact temporary finding", () => {
  const result = evaluateAudit(fixture());
  assert.deepEqual(result.failures, []);
  assert.equal(result.accepted.length, 1);
});

const driftCases = [
  [
    "another High advisory",
    (input) => {
      input.report.advisories.other = { ...input.report.advisories[1240992] };
      input.report.metadata.vulnerabilities.high = 2;
    },
  ],
  [
    "a Critical advisory",
    (input) => {
      input.report.advisories[1240992].severity = "critical";
      input.report.metadata.vulnerabilities.high = 0;
      input.report.metadata.vulnerabilities.critical = 1;
    },
  ],
  [
    "a different path",
    (input) => {
      input.report.advisories[1240992].findings[0].paths = ["runtime>braces"];
    },
  ],
  [
    "an additional path",
    (input) => {
      input.report.advisories[1240992].findings[0].paths.push("runtime>braces");
    },
  ],
  [
    "a different resolved version",
    (input) => {
      input.report.advisories[1240992].findings[0].version = "3.0.4";
    },
  ],
  [
    "a patched range",
    (input) => {
      input.report.advisories[1240992].patched_versions = ">=3.0.4";
    },
  ],
  [
    "an expired exception",
    (input) => {
      input.today = new Date("2026-11-01T00:00:00Z");
    },
  ],
  [
    "a stale exception",
    (input) => {
      input.report.advisories = {};
      input.report.metadata.vulnerabilities.high = 0;
    },
  ],
  [
    "a runtime-scoped dependency",
    (input) => {
      input.manifests.get("apps/web/package.json").dependencies = {
        "eslint-config-next": "16.3.6",
      };
    },
  ],
];

for (const [name, change] of driftCases) {
  test(`fails for ${name}`, () => {
    assert.notEqual(failures(change), "");
  });
}

test("fails for a broadened exception", () => {
  const input = fixture();
  input.policy.exceptions[0].allowedPaths.push("runtime>braces");
  assert.throws(() => evaluateAudit(input), /approved exact scope/);
});

test("fails for malformed audit JSON", () => {
  const input = fixture();
  input.report.advisories = [];
  assert.throws(() => evaluateAudit(input), /advisories must be an object/);
});

test("fails if audit counts conceal an advisory", () => {
  const input = fixture();
  input.report.metadata.vulnerabilities.critical = 1;
  assert.throws(() => evaluateAudit(input), /count does not match/);
});

test("fails if any advisory is muted", () => {
  const input = fixture();
  input.report.muted.push("GHSA-other");
  assert.throws(() => evaluateAudit(input), /muted advisories/);
});
