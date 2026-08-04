# Authentication Phase 1 — Execution Plan

**Status:** Approved
**Version:** AUTH-PLAN-v1
**Effective date:** 2026-08-04
**Depends on:** M0 closed; `AUTH-v1`; ADR-013 through ADR-018
**Scope:** M1 only. Authentication implementation had not started when this plan was approved.

## Objective and scope

Deliver registration, login, current-user lookup, 15-minute Ed25519 access JWTs, single-use refresh-session rotation, token-family replay response, current-session logout, disabled-account enforcement, default-deny principal resolution, durable rate limits, security audits, browser session bootstrap, protected routes, and automated verification.

Out of scope: email verification, password reset/change, MFA, passkeys, SSO, social login, teams, administration, profile management, account deletion, logout-all-sessions, and every Portfolio Foundation or later business capability.

## Architecture

Authentication belongs exclusively to `backend/internal/identity`. Its domain layer owns users, account status, refresh-session family transitions, principal rules, and domain errors. Its application layer owns register, login, refresh, logout, current-user, and principal-resolution use cases. Infrastructure implements pgx/sqlc repositories, Argon2id, token generation/digesting, Ed25519 signing/verification, durable rate limits, and public audit-writing interfaces. Fiber handlers only validate DTOs, serialize cookies/responses, map errors, and invoke use cases.

Later modules consume only a small identity principal/authorization public boundary. They must not import identity infrastructure or write identity tables. Platform owns correlation, structured logging, transactions, and the generic append-only audit mechanism.

## Repository assessment and inherited constraints

M0 provides the Go/Fiber API and worker entry points, PostgreSQL connection lifecycle, goose/sqlc workflow, OpenAPI v1 health contract, structured logging, correlation IDs, standard error envelopes, the Next.js App Router shell, React Query, React Hook Form, Zod, browser-test foundation, Compose smoke checks, and seven mandatory CI jobs. Authentication adds the `identity` bounded context without changing the modular-monolith boundary. No current repository package may be treated as an Authentication implementation: there are no identity migrations, endpoints, forms, signing keys, or user records at this plan version.

M1 must reuse platform configuration, error, logging, database, and audit public interfaces where they exist. If their documented public interface differs from repository reality, implementation must stop at that boundary, record the mismatch, and make a focused platform correction rather than importing a private package.

## Delivery order

1. Complete the implementation gate in the Authentication Security Decision Closure Specification.
2. Propose/review OpenAPI operations and generated TypeScript types.
3. Add identity, audit, and rate-limit migrations; then sqlc queries and repository integration tests.
4. Add pure identity domain/application rules and crypto adapters.
5. Deliver registration/login/current-user vertical slice.
6. Deliver refresh rotation/replay, logout, rate limiting, audit, and principal middleware.
7. Deliver frontend forms, memory-only session state, controlled refresh/retry, and protected neutral `/app` route.
8. Deliver security/E2E/Compose verification, operational runbooks, and completion report.

Every PR is focused, self-reviewed under ADR-013, up to date, conversation-clean, and passes all seven required checks. No insecure temporary behavior may reach `main`.

## Domain and database plan

`users` is identity-authoritative: opaque immutable server-generated ID, unique trimmed/lowercased email, Argon2id encoded hash, active/disabled status, and lifecycle timestamps. Email aliases are preserved. `refresh_sessions` stores a token generation: session ID, family ID, user ID, digest only, active/replaced/revoked/expired state, replacement link, idle/absolute expiry, timestamps, revocation reason, and bounded safe device metadata. `audit_logs` is platform-owned append-only evidence written through an interface in the same transaction. `auth_rate_limit_events` is platform operational state.

Refresh is one PostgreSQL transaction using row locking on the presented digest row. It verifies state/expiry/status, replaces the active row, inserts a new one in the same family, and writes audit evidence. A concurrent replay observes replacement and revokes only that family. Expired/revoked sessions are cleaned by the existing worker after retention approval.

## Contract plan

Add version-1 operations for registration, login, refresh, logout, and current user under `/api/v1/auth`. Register/login set the refresh cookie and return only the access token plus safe user DTO. Refresh rotates the cookie and returns a new access token. Logout clears the matching cookie and revokes the current session. `GET /auth/me` requires bearer JWT.

Use existing correlation/error conventions. Login returns one generic failure for unknown email, disabled account, and wrong password. Duplicate registration is generic after syntactic validation. Rate limiting returns stable `429` behavior without account disclosure. Raw refresh tokens never enter JSON, logs, audits, URLs, or frontend state.

Registration and login both establish one refresh-session family and return a 15-minute access token plus safe authenticated-user data. Refresh requires the browser cookie and the browser-security policy's origin/header controls. Logout requires that same protection, revokes only the current session/family according to the adopted model, and clears the matching cookie. Each operation has request/response/error schemas, cookie documentation, correlation headers, and contract tests before a handler is added.

## Frontend plan

Use `/register`, `/login`, and protected neutral `/app`. React Hook Form and Zod provide usability validation; backend remains authoritative. An identity API adapter uses generated types. Access token lives only in non-persisted memory. Bootstrap performs one controlled refresh, then calls `me`. A single-flight coordinator permits one retry of one failed bearer request, prevents loops, clears state on failure, and uses browser locking/broadcast only for state changes—not token sharing.

## Required verification

Run `make format-check`, `make lint`, `make test`, `make test-integration`, `make contract-check`, `make sqlc-generate` plus drift check, disposable goose migration verification, `make test-e2e`, `make build`, Compose smoke, dependency vulnerability scans, secret scanning, and all seven remote CI checks. Add domain, application, repository concurrency, API/contract, frontend, browser, and security regression coverage.

## Security and operational plan

The final implementation uses [AUTH_IMPLEMENTATION_POLICY-v1](../policies/AUTH_IMPLEMENTATION_POLICY-v1.md). It mitigates credential stuffing and brute force with shared limits; account enumeration with generic failures; replay with atomic row-lock rotation and family revocation; token forgery with a fixed EdDSA-only verifier and validated claims; XSS exposure by memory-only access tokens; CSRF with same-origin HTTPS, Origin, and custom-header checks; proxy spoofing with explicit trusted CIDRs; and logging leakage with allowlisted fields and redaction tests.

Authentication audit records include registration/login/refresh success and failure, logout, family revocation, replay, and disabled-account rejection. They may hold actor ID, correlation ID, outcome, safe family/session identifier, HMAC-derived IP identity, and bounded user-agent metadata. They must never hold credentials, token values, cookie/header content, hashes, or private keys. Metrics use policy/result categories, not raw email/IP labels.

## Ordered implementation tasks

| Task ID | Objective | Depends on | Acceptance evidence | Complexity |
|---|---|---|---|---|
| AUTH-CONTRACT-001 | Add reviewed v1 auth schemas and contract tests. | Gate open | Validated contract; no undocumented sensitive fields. | Medium |
| AUTH-DB-001 | Design/apply identity, audit, and rate-limit migrations; add sqlc queries. | Contract; policy | Empty/upgrade migration and generation/drift tests. | Large |
| AUTH-BE-001 | Add pure identity domain/application rules and repositories. | DB | Unit/application/repository tests, including locks. | Large |
| AUTH-BE-002 | Add password, key-ring/JWT, refresh-token, audit, and rate-limit adapters. | AUTH-BE-001 | Policy conformance and secret-safety tests. | Large |
| AUTH-BE-003 | Add transport, principal middleware, CSRF checks, and auth operations. | Contract; adapters | API/contract/security integration tests. | Large |
| AUTH-FE-001 | Add forms and API adapter with memory-only session state. | Contract | Component validation and no-persistence tests. | Medium |
| AUTH-FE-002 | Add bootstrap, single-flight refresh, protected route, and logout. | AUTH-BE-003 | Browser recovery, retry, and loop-prevention tests. | Large |
| AUTH-OPS-001 | Add local HTTPS/Compose/E2E operational verification and runbooks. | Browser flow | Secure-cookie and Compose evidence. | Medium |
| AUTH-VERIFY-001 | Execute full M1 verification and prepare completion report. | All prior | Seven CI jobs and acceptance matrix. | Medium |

Each task is a focused pull request with the ADR-013 self-review checklist. Intermediate PRs must not route an incomplete or insecure flow. Feature work may remain non-routed only when its migrations, configuration, and tests are internally complete and do not alter public behavior.

## Rollout and Definition of Done

Run migrations before code requiring them, test an empty database and an upgrade path, generate sqlc output, then deploy with the secret-manager configuration, exact public origin, trusted proxy CIDRs, and HMAC secrets present. Production requires TLS and same-origin proxy routing; GitHub Actions uses ephemeral signing keys and an isolated HTTPS test certificate. Rollback analysis must account for active signing-key overlap and refresh-session schema compatibility. No development private key is committed.

M1 is done only when every AUTH-v1 rule and component policy is implemented, registrations and generic login failures are correct, disabled accounts are rejected, JWT validation is complete, refresh rotation is atomic and replay revokes only its family, logout clears/revokes correctly, shared rate limits and audits work, browser state is memory-only, cookies and CSRF controls are enforced, migrations/contracts/generated output are current, all test layers pass, sensitive data is absent from logs/storage, and all required CI checks pass.

## Completion criteria

M1 closes only when every `AUTH-v1` rule is implemented and evidenced: credentials/tokens are absent from logs and persistent browser storage; cookie/CSRF policy is enforced; durable rate limits and audits work; replay revokes the family; disabled users are rejected; contracts, migrations, sqlc, tests, Compose, builds, and required CI pass; and no M2 behavior was added.
