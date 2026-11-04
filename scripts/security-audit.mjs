import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { evaluateAudit } from "./security-audit-lib.mjs";

try {
  const root = process.cwd();
  const policy = JSON.parse(
    readFileSync(resolve(root, "security/audit-exceptions.json"), "utf8"),
  );
  const audit = spawnSync(
    "pnpm",
    ["audit", "--json", "--audit-level", "high"],
    {
      cwd: root,
      encoding: "utf8",
      env: process.env,
      maxBuffer: 16 * 1024 * 1024,
    },
  );
  // The scanner exit must agree with the complete parsed JSON report.
  // Diagnostics, signals and unexpected process results always fail closed.
  if (
    audit.error ||
    audit.signal ||
    ![0, 1].includes(audit.status) ||
    audit.stderr.trim()
  ) {
    throw new Error("pnpm audit process failed unexpectedly");
  }
  const report = JSON.parse(audit.stdout);
  const result = evaluateAudit({ report, policy });
  if (audit.status !== result.expectedExitCode) {
    throw new Error("pnpm audit exit status does not match advisory counts");
  }
  for (const advisory of result.nonBlockingAdvisories) {
    console.log(
      `pnpm audit advisory below policy threshold: ${JSON.stringify(advisory)}`,
    );
  }
  if (result.failures.length > 0) {
    throw new Error(result.failures.join("; "));
  }
  console.log(
    "Security audit gate passed with no active exceptions and no High/Critical advisories.",
  );
} catch (error) {
  console.error(`Security audit gate failed: ${error.message}`);
  process.exitCode = 1;
}
