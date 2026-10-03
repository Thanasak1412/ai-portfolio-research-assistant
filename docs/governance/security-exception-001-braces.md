# SECURITY_EXCEPTION-001 — Temporary braces audit exception

- **Status:** Accepted temporary risk
- **Advisory:** GHSA-vfj7-8cjw-p6xm
- **Package:** `braces@3.0.3`
- **Severity:** High
- **Expires:** 2026-10-31

## Decision

The High/Critical audit gate remains enforced. Only the exact
GHSA-vfj7-8cjw-p6xm finding is temporarily accepted while all of these remain true:

- package is exactly `braces` at resolved version `3.0.3`;
- vulnerable range remains `<=3.0.3`, with no patched range available;
- severity remains High, never Critical;
- the only path is
  `apps__web>eslint-config-next>@next/eslint-plugin-next>fast-glob>micromatch>braces`;
- `eslint-config-next` remains development-only in `apps/web/package.json`;
- the exception has not expired.

The policy fails closed for any other High/Critical advisory, path/version/scope
drift, newly available patch, stale exception, malformed audit output, or expiry.

This is an accepted temporary risk, **not** a claim that the vulnerability is
fixed, remediated, or harmless. No audit ignore, severity downgrade, dependency
override, fork, or scanner suppression is authorized.

## Removal trigger

Remove the exception and upgrade or eliminate the vulnerable path as soon as a
non-vulnerable published release exists, the dependency path disappears, or the
exception reaches 2026-10-31. Do not extend expiry silently.
