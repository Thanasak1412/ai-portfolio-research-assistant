const retiredException = {
  id: "SECURITY_EXCEPTION-001",
  advisory: "GHSA-vfj7-8cjw-p6xm",
  package: "braces",
  severity: "high",
  vulnerableVersions: "<=3.0.3",
  resolvedVersion: "3.0.3",
  manifestPath: "apps/web/package.json",
  topLevelDependency: "eslint-config-next",
  allowedPaths: [
    "apps__web>eslint-config-next>@next/eslint-plugin-next>fast-glob>micromatch>braces",
  ],
  expiresOn: "2026-10-31",
  retirementReason: "upstream_advisory_withdrawn",
  withdrawnAt: "2026-10-23T12:53:02Z",
};

function record(value, label) {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new Error(`${label} must be an object`);
  }
  return value;
}

function validatePolicy(policy) {
  record(policy, "security audit policy");
  if (policy.schemaVersion !== 2 || policy.auditLevel !== "high") {
    throw new Error("unsupported security audit policy");
  }
  if (!Array.isArray(policy.exceptions) || policy.exceptions.length !== 0) {
    throw new Error("security audit policy must contain no active exceptions");
  }
  if (
    !Array.isArray(policy.retiredExceptions) ||
    policy.retiredExceptions.length !== 1 ||
    JSON.stringify(policy.retiredExceptions[0]) !==
      JSON.stringify(retiredException)
  ) {
    throw new Error(
      "retired exception must preserve its exact historical record",
    );
  }
}

function advisoriesFrom(report) {
  record(report, "pnpm audit JSON");
  const advisories = record(report.advisories, "pnpm audit JSON advisories");
  const counts = record(
    record(report.metadata, "pnpm audit JSON metadata").vulnerabilities,
    "pnpm audit JSON vulnerability counts",
  );
  for (const severity of ["info", "low", "moderate", "high", "critical"]) {
    if (!Number.isSafeInteger(counts[severity]) || counts[severity] < 0) {
      throw new Error(`pnpm audit JSON has invalid ${severity} count`);
    }
  }
  if (!Array.isArray(report.muted) || report.muted.length !== 0) {
    throw new Error("pnpm audit must not contain muted advisories");
  }
  const actualCounts = { info: 0, low: 0, moderate: 0, high: 0, critical: 0 };
  const findingCounts = { ...actualCounts };
  const parsed = Object.values(advisories).map((item) => {
    const advisory = record(item, "pnpm audit advisory");
    if (!Object.hasOwn(actualCounts, advisory.severity)) {
      throw new Error("pnpm audit advisory has unknown severity");
    }
    actualCounts[advisory.severity]++;
    if (
      typeof advisory.github_advisory_id !== "string" ||
      typeof advisory.module_name !== "string" ||
      typeof advisory.vulnerable_versions !== "string" ||
      typeof advisory.patched_versions !== "string" ||
      !Array.isArray(advisory.findings) ||
      advisory.findings.length === 0
    ) {
      throw new Error(
        "pnpm audit advisory is missing required identity or findings",
      );
    }
    const findings = advisory.findings.map((finding) => {
      record(finding, "pnpm audit finding");
      if (
        typeof finding.version !== "string" ||
        !Array.isArray(finding.paths) ||
        finding.paths.length === 0 ||
        finding.paths.some((path) => typeof path !== "string")
      ) {
        throw new Error(
          "pnpm audit finding has invalid version or dependency paths",
        );
      }
      return finding;
    });
    findingCounts[advisory.severity] += findings.length;
    return { advisory, findings };
  });
  // The legacy registry can group multiple findings under one advisory ID.
  // Require one consistent counting unit across the whole report.
  let byAdvisory = true;
  let byFinding = true;
  for (const severity of Object.keys(actualCounts)) {
    const matchesAdvisories = actualCounts[severity] === counts[severity];
    const matchesFindings = findingCounts[severity] === counts[severity];
    if (!matchesAdvisories && !matchesFindings) {
      throw new Error(`pnpm audit ${severity} count does not match advisories`);
    }
    byAdvisory &&= matchesAdvisories;
    byFinding &&= matchesFindings;
  }
  if (!byAdvisory && !byFinding) {
    throw new Error("pnpm audit uses inconsistent severity counting units");
  }
  return parsed;
}

export function evaluateAudit({ report, policy }) {
  validatePolicy(policy);
  const parsed = advisoriesFrom(report);
  const highOrCritical = parsed.filter(({ advisory }) =>
    ["high", "critical"].includes(advisory.severity),
  );
  return {
    failures: highOrCritical.map(
      ({ advisory }) =>
        `${advisory.severity} advisory ${advisory.github_advisory_id} is not exempted`,
    ),
    nonBlockingAdvisories: parsed
      .filter(
        ({ advisory }) => !["high", "critical"].includes(advisory.severity),
      )
      .map(({ advisory, findings }) => ({
        id: advisory.github_advisory_id,
        package: advisory.module_name,
        severity: advisory.severity,
        findings: findings.map(({ version, paths }) => ({ version, paths })),
      })),
    // pnpm 10.18.3 JSON mode exits 1 for findings at any severity;
    // the repository still applies its own High/Critical policy.
    expectedExitCode: parsed.length > 0 ? 1 : 0,
  };
}
