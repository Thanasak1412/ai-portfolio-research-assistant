# Authentication M1 Acceptance Matrix

Verification task: `AUTH-VERIFY-001`
Policy composition: `AUTH_IMPLEMENTATION_POLICY-v3`
Verification branch: `codex/auth-verify-001`
Protected `main` base: `57f6dde` (required ancestor `ef4c40ff6c5afe0c1a751b174bc1f6a0f655217f`)
Final verification head: `5f6fc992a2e94e47172cfe00879a42c6896912ad`
Final remote workflow: [run 31761799437](https://github.com/Thanasak1412/ai-portfolio-research-assistant/actions/runs/31761799437)

Result values are `PASS`, `FAIL`, `BLOCKED`, or `NOT_APPLICABLE`. A blocked row is not treated as a pass. Remote CI evidence is used where the local Docker daemon became unavailable during verification.

| Requirement | Policy / ADR | Evidence | Result | Notes |
|---|---|---|---|---|
| Approved five Authentication operations only | AUTH-v1 | `pnpm contract:check`; contract tests 1–9 | PASS | No M2 operation introduced. |
| Generated TypeScript contract has no drift | Contract workflow | `pnpm contract:check`; remote `contracts-and-generation` | PASS | |
| Email normalization and opaque user identity | AUTH-v1 | `domain/email_test.go`, `domain/user_test.go`; remote `database-integration` | PASS | |
| Password bounds and no composition rules | PASSWORD_HASH-v1 | `validation/credentials.test.ts`; contract tests | PASS | 12 characters and 1024 UTF-8-byte ceiling. |
| Argon2id parameters, PHC format, and rehash | PASSWORD_HASH-v1; ADR-014 | `infrastructure/password/argon2id_test.go`; remote `backend` | PASS | |
| Registration persistence and transaction | AUTH-v1 | application tests; remote `database-integration` and `browser-e2e` | PASS | |
| Generic duplicate-registration failure | AUTH-v1 | application/HTTP tests; contract test 5 | PASS | |
| Generic unknown/wrong/disabled login failure | AUTH-v1 | application/HTTP tests; remote `database-integration` | PASS | Public failure remains generic. |
| EdDSA/Ed25519 JWT and required claims | TOKEN_SIGNING-v1; ADR-015 | `infrastructure/token/jwt_test.go`; remote `backend` | PASS | |
| JWT issuer, audience, expiry, skew, kid, and algorithm rejection | TOKEN_SIGNING-v1 | `infrastructure/token/jwt_test.go` | PASS | |
| Refresh token v1 representation and digest | REFRESH_TOKEN-v1; ADR-019 | `infrastructure/token/refresh_test.go`; schema | PASS | |
| Atomic refresh rotation and family preservation | AUTH-v1 | application/repository integration tests; remote `database-integration` | PASS | |
| Concurrent refresh and replay family revocation | AUTH-v1 | concurrency integration tests; remote `database-integration` | PASS | PostgreSQL locking exercised remotely. |
| Logout and cookie clearing | AUTH-v1; AUTH_BROWSER_SECURITY-v1 | HTTP tests; remote `browser-e2e` | PASS | |
| Disabled-account enforcement | AUTH-v1 | domain/application/JWT tests; remote `database-integration` and `browser-e2e` | PASS | |
| Default-deny principal extraction | AUTH-v1 | `domain/principal_test.go`, HTTP tests | PASS | |
| PostgreSQL-backed rate limits and advisory locking | AUTH_RATE_LIMIT-v1; ADR-018 | unit/integration tests; remote `database-integration` | PASS | Durable shared persistence verified. |
| Network identity and HMAC derivation | CLIENT_NETWORK_IDENTITY-v1; AUTH_HMAC_KEYS-v1 | network/authhmac unit tests | PASS | |
| Browser Origin, custom header, and HTTPS requirements | AUTH_BROWSER_SECURITY-v1; ADR-016/021 | HTTP security tests; remote `browser-e2e` | PASS | |
| Strict trusted HTTPS attestation | HTTPS_ATTESTATION-v1; ADR-021 | `https_attestor_test.go`; handler tests; remote `browser-e2e` | PASS | |
| Secure HttpOnly host-only refresh cookie | AUTH_BROWSER_SECURITY-v1 | contract/HTTP tests; remote `browser-e2e` | PASS | |
| Access token memory-only frontend behavior | AUTH-v1 | `auth-security.test.ts`; session tests | PASS | |
| Single-flight refresh and one retry maximum | Frontend runtime policy | coordinator/session tests | PASS | |
| Protected route and no content flash | AUTH-v1 | routing tests; remote `browser-e2e` | PASS | |
| Real HTTPS registration/reload/logout flow | AUTH-OPS-001 | remote `browser-e2e` job: `3 passed (9.2s)` | PASS | Live Caddy → Next.js → API → PostgreSQL stack. |
| Empty/up/down migration and sqlc reproducibility | AUTH-DB-001 | remote `database-integration`; remote `contracts-and-generation` | PASS | Empty, reset/up/down/up and drift checks passed. |
| Audit append-only safe metadata | AUTH-v1 | audit tests; remote `database-integration` | PASS | |
| Log redaction and forbidden secret fields | AUTH-v1 | source review; GitGuardian/Gitleaks | PASS | No secrets found. |
| Secret-management and ignored fixtures | TOKEN_SIGNING-v1; AUTH_HMAC_KEYS-v1 | `.gitignore`; key-ring checks; remote `secrets` | PASS | |
| JavaScript dependency vulnerabilities | CI policy | `pnpm audit --audit-level high`; remote `frontend` | PASS | Patched nanoid 3.3.18; no known high vulnerabilities. |
| Go reachable vulnerabilities | CI policy | `govulncheck ./...` with Go 1.26.6; remote `backend` | PASS | No reachable vulnerabilities. |
| Formatting, lint, typecheck, unit tests, backend build | CI policy | Make/Go checks; remote `frontend` and `backend` | PASS | |
| Seven mandatory remote CI jobs on final head | ADR-013; CI policy | [run 31761799437](https://github.com/Thanasak1412/ai-portfolio-research-assistant/actions/runs/31761799437) | PASS | All seven jobs passed; GitGuardian passed separately. |

## Closure decision

All verification rows pass on the final reviewed head, with equivalent remote evidence covering the local Docker outage. Formal M1 closure remains pending AUTH-VERIFY-001 review approval and merge into protected `main`; this draft PR does not authorize M2 work.
