const exactException = {
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
};

function record(value, label) {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new Error(`${label} must be an object`);
  }
  return value;
}

function validatePolicy(policy) {
  record(policy, "security audit policy");
  if (policy.schemaVersion !== 1 || policy.auditLevel !== "high") {
    throw new Error("unsupported security audit policy");
  }
  if (!Array.isArray(policy.exceptions) || policy.exceptions.length !== 1) {
    throw new Error("security audit policy must contain exactly one exception");
  }
  const candidate = record(policy.exceptions[0], "security exception");
  if (JSON.stringify(candidate) !== JSON.stringify(exactException)) {
    throw new Error("security exception differs from the approved exact scope");
  }
  return candidate;
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
    return { advisory, findings };
  });
  for (const severity of Object.keys(actualCounts)) {
    if (actualCounts[severity] !== counts[severity]) {
      throw new Error(`pnpm audit ${severity} count does not match advisories`);
    }
  }
  return parsed;
}

export function evaluateAudit({ report, policy, manifests, today }) {
  const exception = validatePolicy(policy);
  if (
    !(manifests instanceof Map) ||
    !(today instanceof Date) ||
    Number.isNaN(today.getTime())
  ) {
    throw new Error("security audit inputs are invalid");
  }
  const parsed = advisoriesFrom(report);
  const failures = [];
  const accepted = [];
  const highOrCritical = parsed.filter(({ advisory }) =>
    ["high", "critical"].includes(advisory.severity),
  );
  if (highOrCritical.length !== 1) {
    failures.push(
      `expected exactly one scoped High advisory, found ${highOrCritical.length}`,
    );
  }
  for (const { advisory, findings } of highOrCritical) {
    if (advisory.severity === "critical") {
      failures.push("Critical advisories cannot be exempted");
      continue;
    }
    if (
      advisory.github_advisory_id !== exception.advisory ||
      advisory.module_name !== exception.package ||
      advisory.severity !== exception.severity ||
      advisory.vulnerable_versions !== exception.vulnerableVersions ||
      advisory.patched_versions !== "<0.0.0" ||
      findings.length !== 1 ||
      findings[0]?.version !== exception.resolvedVersion ||
      findings[0]?.paths.length !== 1 ||
      findings[0]?.paths[0] !== exception.allowedPaths[0]
    ) {
      failures.push(
        "High advisory differs from the exact approved exception or has a patch",
      );
      continue;
    }
    accepted.push({
      id: exception.id,
      advisory: exception.advisory,
      package: exception.package,
      version: exception.resolvedVersion,
      expiresOn: exception.expiresOn,
    });
  }
  if (accepted.length !== 1) {
    failures.push(
      "security exception is stale or no longer precisely reported",
    );
  }
  const expiresAt = new Date(`${exception.expiresOn}T23:59:59.999Z`);
  if (
    Number.isNaN(expiresAt.getTime()) ||
    today.getTime() > expiresAt.getTime()
  ) {
    failures.push(`${exception.id} expired on ${exception.expiresOn}`);
  }
  const manifest = manifests.get(exception.manifestPath);
  if (!manifest || typeof manifest !== "object") {
    failures.push("exception dependency manifest was not loaded");
  } else if (
    !Object.hasOwn(
      manifest.devDependencies ?? {},
      exception.topLevelDependency,
    ) ||
    ["dependencies", "optionalDependencies", "peerDependencies"].some((field) =>
      Object.hasOwn(manifest[field] ?? {}, exception.topLevelDependency),
    )
  ) {
    failures.push(
      `${exception.topLevelDependency} is no longer development-only`,
    );
  }
  return { failures, accepted };
}
