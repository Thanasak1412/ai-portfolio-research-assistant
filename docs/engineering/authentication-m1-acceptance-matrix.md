# Authentication M1 Acceptance Matrix

Verification task: `AUTH-VERIFY-001`  
Policy composition: `AUTH_IMPLEMENTATION_POLICY-v3`  
Verification branch: `codex/auth-verify-001`  
Base: protected `main` at `57f6dde` (required ancestor `ef4c40ff6c5afe0c1a751b174bc1f6a0f655217f`)

Result values are `PASS`, `FAIL`, `BLOCKED`, or `NOT_APPLICABLE`. A blocked row is not treated as a pass.

| Requirement | Policy / ADR | Evidence | Result | Notes |
|---|---|---|---|---|
| Approved five Authentication operations only | AUTH-v1 | `pnpm contract:check`; contract tests 1–9 | PASS | No M2 operation introduced. |
| Generated TypeScript contract has no drift | Contract workflow | `pnpm contract:check`; `sqlc generate`; generated diff | PASS | |
| Email normalization and opaque user identity | AUTH-v1 | `domain/email_test.go`, `domain/user_test.go` | PASS | Database rerun blocked. |
| Password bounds and no composition rules | PASSWORD_HASH-v1 | `validation/credentials.test.ts`; contract tests | PASS | 12 characters and 1024 UTF-8-byte ceiling. |
| Argon2id parameters, PHC format, and rehash | PASSWORD_HASH-v1; ADR-014 | `infrastructure/password/argon2id_test.go` | PASS | |
| Registration persistence and transaction | AUTH-v1 | application tests; database integration suite | BLOCKED | PostgreSQL integration unavailable after Docker failure. |
| Generic duplicate-registration failure | AUTH-v1 | application/HTTP tests; contract test 5 | PASS | |
| Generic unknown/wrong/disabled login failure | AUTH-v1 | application/HTTP tests; contract test 5 | BLOCKED | Final DB-backed rerun blocked. |
| EdDSA/Ed25519 JWT and required claims | TOKEN_SIGNING-v1; ADR-015 | `infrastructure/token/jwt_test.go` | PASS | |
| JWT issuer, audience, expiry, skew, kid, and algorithm rejection | TOKEN_SIGNING-v1 | `infrastructure/token/jwt_test.go` | PASS | |
| Refresh token v1 representation and digest | REFRESH_TOKEN-v1; ADR-019 | `infrastructure/token/refresh_test.go`; schema | PASS | |
| Atomic refresh rotation and family preservation | AUTH-v1 | application/repository integration tests | BLOCKED | Database rerun blocked. |
| Concurrent refresh and replay family revocation | AUTH-v1 | concurrency integration tests | BLOCKED | PostgreSQL test database unavailable. |
| Logout and cookie clearing | AUTH-v1; AUTH_BROWSER_SECURITY-v1 | HTTP tests; real E2E suite | BLOCKED | Final runtime rerun blocked. |
| Disabled-account enforcement | AUTH-v1 | domain/application/JWT tests; DB-backed HTTP tests | BLOCKED | Final DB-backed rerun blocked. |
| Default-deny principal extraction | AUTH-v1 | `domain/principal_test.go`, HTTP tests | PASS | |
| PostgreSQL-backed rate limits and advisory locking | AUTH_RATE_LIMIT-v1; ADR-018 | unit/integration tests | BLOCKED | PostgreSQL integration rerun blocked. |
| Network identity and HMAC derivation | CLIENT_NETWORK_IDENTITY-v1; AUTH_HMAC_KEYS-v1 | network/authhmac unit tests | PASS | |
| Browser Origin, custom header, and HTTPS requirements | AUTH_BROWSER_SECURITY-v1; ADR-016/021 | HTTP security tests; stack script | BLOCKED | Runtime rerun blocked. |
| Strict trusted HTTPS attestation | HTTPS_ATTESTATION-v1; ADR-021 | `https_attestor_test.go`; handler tests | PASS | Unit evidence available. |
| Secure HttpOnly host-only refresh cookie | AUTH_BROWSER_SECURITY-v1 | contract/HTTP tests; E2E suite | BLOCKED | Final runtime rerun blocked. |
| Access token memory-only frontend behavior | AUTH-v1 | `auth-security.test.ts`; session tests | PASS | |
| Single-flight refresh and one retry maximum | Frontend runtime policy | coordinator/session tests | PASS | |
| Protected route and no content flash | AUTH-v1 | routing tests; E2E suite | BLOCKED | Final E2E rerun blocked. |
| Real HTTPS registration/reload/logout flow | AUTH-OPS-001 | Playwright suite previously 3/3; final rerun | BLOCKED | Docker daemon stopped. |
| Empty/up/down migration and sqlc reproducibility | AUTH-DB-001 | migration setup; `sqlc generate` | BLOCKED | Final migration rerun blocked. |
| Audit append-only safe metadata | AUTH-v1 | audit tests; DB integration suite | BLOCKED | Final DB-backed rerun blocked. |
| Log redaction and forbidden secret fields | AUTH-v1 | source review; GitGuardian scan | PASS | No secrets found locally. |
| Secret-management and ignored fixtures | TOKEN_SIGNING-v1; AUTH_HMAC_KEYS-v1 | `.gitignore`; key-ring checks; scan | PASS | |
| JavaScript dependency vulnerabilities | CI policy | `pnpm audit --audit-level high` | PASS | Patched nanoid 3.3.18; no known vulnerabilities. |
| Go reachable vulnerabilities | CI policy | `govulncheck ./...` with Go 1.26.6 | PASS | No reachable vulnerabilities. |
| Formatting, lint, typecheck, unit tests, backend build | CI policy | Make/Go checks | PASS | |
| Seven mandatory remote CI jobs on final head | ADR-013; CI policy | Remote PR run | BLOCKED | Branch not yet pushed; no final CI run. |

## Closure decision

M1 cannot close while any critical or major row remains `BLOCKED`. Current blockers are Docker recovery for PostgreSQL/Compose reruns and the required remote CI run on the final verification head.
