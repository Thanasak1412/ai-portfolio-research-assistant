import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { evaluateAudit } from "./security-audit-lib.mjs";

try {
  const root = process.cwd();
  const policy = JSON.parse(
    readFileSync(resolve(root, "security/audit-exceptions.json"), "utf8"),
  );
  const manifests = new Map(
    policy.exceptions.map((exception) => [
      exception.manifestPath,
      JSON.parse(readFileSync(resolve(root, exception.manifestPath), "utf8")),
    ]),
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
  if (audit.error || audit.signal || ![0, 1].includes(audit.status)) {
    throw new Error("pnpm audit process failed unexpectedly");
  }
  const report = JSON.parse(audit.stdout);
  const result = evaluateAudit({
    report,
    policy,
    manifests,
    today: new Date(),
  });
  if (result.failures.length > 0) {
    throw new Error(result.failures.join("; "));
  }
  for (const item of result.accepted) {
    console.warn(
      `TEMPORARY ACCEPTED RISK: ${item.id} permits ${item.advisory} for ${item.package}@${item.version} until ${item.expiresOn}. This vulnerability is not fixed or remediated.`,
    );
  }
  console.log(
    "Security audit gate passed with one explicit temporary exception and no other High/Critical advisories.",
  );
} catch (error) {
  console.error(`Security audit gate failed: ${error.message}`);
  process.exitCode = 1;
}
