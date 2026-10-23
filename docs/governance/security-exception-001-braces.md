# SECURITY_EXCEPTION-001 — Temporary braces audit exception

- **Status:** Retired from active audit policy — upstream advisory withdrawn
- **Advisory:** GHSA-vfj7-8cjw-p6xm
- **Package:** `braces@3.0.3`
- **Severity:** High
- **Original expiry:** 2026-10-31 (unchanged; not extended)

## Retirement decision

GitHub [withdrew GHSA-vfj7-8cjw-p6xm](https://github.com/advisories/GHSA-vfj7-8cjw-p6xm)
at **2026-10-23T12:53:02Z**, stating that it is not a practically exploitable
vulnerability. The raw registry audit consequently no longer reports it.
`braces@3.0.3` and the original development dependency path remain present;
withdrawal is not a package patch and must not be described as remediation.

On 2026-10-23 the maintainer explicitly authorized a separate reviewed
exception-retirement PR through the task's authorization reply. This document
records that proposed policy change; it takes repository effect only through
ADR-013 review and merge. No agent approval or automatic merge is implied.

The schema-v2 audit policy has **zero active exceptions**. It retains the exact
original exception, scope and expiry in `retiredExceptions` for history, without
granting any allowance. Every reported High/Critical advisory, including a
reappearing GHSA-vfj7-8cjw-p6xm, fails the audit. Clean scans no longer require a
withdrawn finding. Malformed/muted reports, inconsistent counts and scanner exit
status, diagnostics and process errors continue to fail closed.

The development queue and its original temporary-governance declaration are
unchanged by this narrowly scoped retirement. That preserved declaration does
not grant an active audit allowance; the active machine policy is
`security/audit-exceptions.json`. No task activation or queue transition is
authorized here.

## Original decision (historical; no longer an active allowance)

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

## Original removal trigger (historical)

Remove the exception and upgrade or eliminate the vulnerable path as soon as a
non-vulnerable published release exists, the dependency path disappears, or the
exception reaches 2026-10-31. Do not extend expiry silently.
