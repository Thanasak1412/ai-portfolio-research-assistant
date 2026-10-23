import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  mkdtempSync,
  mkdirSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { evaluateAudit } from "./security-audit-lib.mjs";

const policy = JSON.parse(
  readFileSync(
    new URL("../security/audit-exceptions.json", import.meta.url),
    "utf8",
  ),
);

function fixture() {
  return {
    policy: structuredClone(policy),
    report: {
      advisories: {},
      muted: [],
      metadata: {
        vulnerabilities: { info: 0, low: 0, moderate: 0, high: 0, critical: 0 },
      },
    },
  };
}

function addAdvisory(input, severity, id = "GHSA-new-finding") {
  input.report.advisories[id] = {
    github_advisory_id: id,
    module_name: "example-package",
    severity,
    vulnerable_versions: "<2.0.0",
    patched_versions: ">=2.0.0",
    findings: [{ version: "1.0.0", paths: ["app>example-package"] }],
  };
  input.report.metadata.vulnerabilities[severity]++;
}

test("accepts a clean scan without requiring a withdrawn advisory", () => {
  assert.deepEqual(evaluateAudit(fixture()), {
    failures: [],
    expectedExitCode: 0,
  });
});

test("preserves the High threshold for reported Moderate findings", () => {
  const input = fixture();
  addAdvisory(input, "moderate");
  assert.deepEqual(evaluateAudit(input), { failures: [], expectedExitCode: 0 });
});

for (const severity of ["high", "critical"]) {
  test(`rejects every ${severity} finding`, () => {
    const input = fixture();
    addAdvisory(input, severity);
    const result = evaluateAudit(input);
    assert.equal(result.expectedExitCode, 1);
    assert.match(
      result.failures.join("; "),
      /GHSA-new-finding is not exempted/,
    );
  });
}

test("a reappearing retired advisory has no exemption", () => {
  const input = fixture();
  addAdvisory(input, "high", "GHSA-vfj7-8cjw-p6xm");
  assert.equal(evaluateAudit(input).failures.length, 1);
});

test("rejects simultaneous High and Critical findings", () => {
  const input = fixture();
  addAdvisory(input, "high", "GHSA-high");
  addAdvisory(input, "critical", "GHSA-critical");
  assert.equal(evaluateAudit(input).failures.length, 2);
});

const malformedCases = [
  [
    "non-object advisories",
    (input) => {
      input.report.advisories = [];
    },
  ],
  [
    "concealed advisory count",
    (input) => {
      input.report.metadata.vulnerabilities.high = 1;
    },
  ],
  [
    "negative count",
    (input) => {
      input.report.metadata.vulnerabilities.high = -1;
    },
  ],
  [
    "muted finding",
    (input) => {
      input.report.muted.push("GHSA-muted");
    },
  ],
  [
    "missing metadata",
    (input) => {
      delete input.report.metadata;
    },
  ],
  [
    "unknown severity",
    (input) => {
      addAdvisory(input, "high");
      input.report.advisories["GHSA-new-finding"].severity = "unknown";
    },
  ],
  [
    "missing identity",
    (input) => {
      addAdvisory(input, "high");
      delete input.report.advisories["GHSA-new-finding"].github_advisory_id;
    },
  ],
  [
    "empty findings",
    (input) => {
      addAdvisory(input, "high");
      input.report.advisories["GHSA-new-finding"].findings = [];
    },
  ],
  [
    "invalid path",
    (input) => {
      addAdvisory(input, "high");
      input.report.advisories["GHSA-new-finding"].findings[0].paths = [null];
    },
  ],
  [
    "active exception",
    (input) => {
      input.policy.exceptions.push({ id: "NEW-EXCEPTION" });
    },
  ],
  [
    "threshold change",
    (input) => {
      input.policy.auditLevel = "critical";
    },
  ],
  [
    "old schema",
    (input) => {
      input.policy.schemaVersion = 1;
    },
  ],
  [
    "history deletion",
    (input) => {
      input.policy.retiredExceptions = [];
    },
  ],
  [
    "historical expiry extension",
    (input) => {
      input.policy.retiredExceptions[0].expiresOn = "2026-11-30";
    },
  ],
  [
    "historical path alteration",
    (input) => {
      input.policy.retiredExceptions[0].allowedPaths.push("another>path");
    },
  ],
];

for (const [name, change] of malformedCases) {
  test(`fails closed for ${name}`, () => {
    const input = fixture();
    change(input);
    assert.throws(() => evaluateAudit(input));
  });
}

function runCLI({
  report = fixture().report,
  status = 0,
  stderr = "",
  stdout,
} = {}) {
  const directory = mkdtempSync(join(tmpdir(), "security-audit-test-"));
  try {
    mkdirSync(join(directory, "security"));
    writeFileSync(
      join(directory, "security", "audit-exceptions.json"),
      JSON.stringify(policy),
    );
    writeFileSync(
      join(directory, "pnpm"),
      `#!${process.execPath}\nprocess.stdout.write(process.env.AUDIT_STDOUT);process.stderr.write(process.env.AUDIT_STDERR);process.exitCode=Number(process.env.AUDIT_STATUS);\n`,
      { mode: 0o755 },
    );
    return spawnSync(
      process.execPath,
      [fileURLToPath(new URL("./security-audit.mjs", import.meta.url))],
      {
        cwd: directory,
        encoding: "utf8",
        env: {
          ...process.env,
          PATH: `${directory}:${process.env.PATH}`,
          AUDIT_STDOUT: stdout ?? JSON.stringify(report),
          AUDIT_STDERR: stderr,
          AUDIT_STATUS: String(status),
        },
      },
    );
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
}

test("CLI accepts exit 0 with a clean report", () => {
  const result = runCLI();
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /no active exceptions/);
});

test("CLI rejects a High report even when scanner exit is correct", () => {
  const input = fixture();
  addAdvisory(input, "high");
  const result = runCLI({ report: input.report, status: 1 });
  assert.equal(result.status, 1);
  assert.match(result.stderr, /not exempted/);
});

test("CLI rejects false scanner success for a High finding", () => {
  const input = fixture();
  addAdvisory(input, "high");
  const result = runCLI({ report: input.report });
  assert.equal(result.status, 1);
  assert.match(result.stderr, /exit status does not match/);
});

test("CLI rejects scanner failure with a clean report", () => {
  const result = runCLI({ status: 1 });
  assert.equal(result.status, 1);
  assert.match(result.stderr, /exit status does not match/);
});

for (const [name, options] of [
  ["diagnostic stderr", { stderr: "registry unavailable" }],
  ["unexpected exit", { status: 2 }],
  ["invalid JSON", { stdout: "not JSON" }],
]) {
  test(`CLI fails closed for ${name}`, () => {
    assert.equal(runCLI(options).status, 1);
  });
}
