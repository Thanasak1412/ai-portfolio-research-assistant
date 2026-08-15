# Authentication Coverage Test Plan

## Scope and environment

This plan covers the browser-visible Authentication behavior of the portfolio
application. It is intended for the real Playwright project configured by
`apps/web/playwright.auth.config.ts`, with:

- Base URL: `https://app.localhost:3443`
- Browser project: Chromium
- HTTPS errors ignored only when the local test certificate requires it via
  `PLAYWRIGHT_AUTH_E2E_IGNORE_HTTPS_ERRORS=true`
- Real Caddy, Next.js, Go API, and PostgreSQL services
- No interception, stubbing, or mocking of `/api/v1/auth/*`

The supported browser entrypoint is the HTTPS URL above. `localhost:3000` and
`localhost:8080` are diagnostic ports only and are not Authentication test
origins.

## Seed and isolation

The seed is `tests/auth-e2e/seed.spec.ts`. It intentionally starts a clean,
unauthenticated visitor on `/register`; it is environment setup, not a business
scenario, and must remain a minimal seed. Each scenario should use a fresh
browser context and a unique email address (for example, a UUID-based
`@example.test` address) unless it explicitly consumes a setup `storageState`.

Do not put credentials, access tokens, or refresh-cookie values in source,
logs, screenshots, traces, or reports. Any generated authenticated state is
temporary test data and must be written only to an ignored path such as
`playwright/.auth/`.

## Authenticated `storageState` strategy

Use a real setup project for scenarios that begin authenticated rather than
repeating registration in every test:

1. In a setup test, create a unique account through the visible registration
   form and wait for `/app`.
2. Save the browser context state to an ignored, per-run file such as
   `playwright/.auth/authenticated.json`.
3. Use that state only for tests whose starting condition is an active session
   (public-route redirects, protected-page rendering, and logout setup).
4. Do not parse JWT claims, read the HttpOnly refresh cookie from application
   JavaScript, or persist the in-memory access token. Playwright may restore the
   browser cookie as part of its context state; the application itself must not
   inspect it.
5. Tests that verify logout, revoked sessions, or clean unauthenticated behavior
   must use a fresh context or a newly generated state so they cannot inherit a
   prior test's mutations.

If the test runner cannot safely share a setup account across workers, keep the
Auth project single-worker and generate one state per worker/run. Never commit
the state file.

## Observable locator inventory

The current pages expose accessible, stable controls:

- Register page heading: `Create account`
- Login page heading: `Sign in`
- Email textbox: label `Email`
- Password textbox: label `Password`
- Register submit: button `Create account`
- Login submit: button `Sign in`
- Register-to-login link: `Sign in instead`
- Login-to-register link: `Create an account`
- Protected identity text: `Signed in as <email>`
- Protected logout control: button `Sign out`
- Bootstrap state: `Restoring your secure session…`
- Recoverable bootstrap error: `Authentication is temporarily unavailable` and
  button `Retry`

Assertions should prefer these roles, labels, visible text, and URL changes over
CSS selectors or implementation details.

## Scenarios

### 1. Bootstrap and public registration route

**Starting state:** Run `tests/auth-e2e/seed.spec.ts` with the Auth config. The
seed enters the supported HTTPS origin at `/register` with no session.

**Expected results:**

- The page becomes usable in an unauthenticated state after the one normal
  refresh/bootstrap attempt is rejected without a refresh loop.
- The `Create account` heading, `Email` textbox, `Password` textbox, and
  `Create account` button are visible.
- `Sign in instead` navigates to `/login`; `/login` exposes the `Create an
  account` link back to `/register`.
- No protected shell or authenticated identity is visible.

### 2. Registration with valid credentials

**Starting state:** Clean visitor from the seed on `/register`.

**Steps:**

1. Fill `Email` with a unique syntactically valid address, including optional
   surrounding whitespace if normalization is being covered.
2. Fill `Password` with a password of at least 12 characters.
3. Submit `Create account`.

**Expected results:**

- The request uses the real HTTPS same-origin Authentication endpoint.
- The form does not expose credential details in the UI.
- Registration succeeds and the browser navigates to `/app`.
- The protected shell shows `Signed in as <normalized email>`.
- A Secure, HttpOnly, host-only `pra_rt_v1` cookie exists with `SameSite=Lax`
  and path `/api/v1/auth`; its value is never asserted or printed.
- No access token or identity is written to localStorage, sessionStorage,
  IndexedDB, URL parameters, analytics, or application JavaScript cookies.

### 3. Registration validation

**Starting state:** Fresh visitor on `/register`.

Run each input case independently and submit the form:

- Empty email: `Email is required`.
- Malformed email: `Enter a valid email address`.
- Empty password: required/minimum validation is shown without submission.
- Password shorter than 12 characters: `Password must be at least 12 characters`.
- Password whose UTF-8 representation exceeds 1,024 bytes: the maximum-size
  validation is shown and no request is sent.

Verify that valid passwords do not require artificial composition rules, that
the password field remains a password input, and that validation messages are
associated with the corresponding labelled control. A validation failure must
not navigate or create a session.

### 4. Duplicate registration uses a generic failure

**Starting state:** Create one real account with a unique normalized email, end
that setup session, and open `/register` in a fresh context.

**Steps:** Submit registration again with the same email and a valid password.

**Expected results:**

- The browser remains on `/register`.
- The response is rendered as the same safe generic registration/Authentication
  failure defined by the contract; it must not say that the email already
  exists.
- The password field is cleared and no second authenticated session is
  established.

### 5. Login page and valid login

**Starting state:** Fresh visitor on `/login` for the display check; use a real
account fixture for the submission check.

**Expected results before submission:**

- `Sign in`, `Email`, `Password`, and `Sign in` submit controls are visible.
- `Create an account` navigates to `/register`.

**Steps for the valid login:** Fill the labelled controls with the fixture
credentials and submit `Sign in`.

**Expected results:**

- Login succeeds through the real API and navigates to `/app`.
- The protected shell identifies the authenticated email.
- The refresh cookie has the approved security attributes, without exposing its
  value.
- The access token is used only in memory.

### 6. Login validation

**Starting state:** Fresh visitor on `/login`.

Repeat the registration validation cases for the login form (empty/malformed
email, password shorter than 12 characters, and password over 1,024 UTF-8
bytes). Verify the same accessible field-level messages and that no request or
navigation occurs for client-side validation failures.

### 7. Invalid credentials use a generic failure

**Starting state:** Fresh visitor on `/login`. Create a known real account in a
separate setup context, then use a fresh context for the failure cases.

Submit a validly shaped email/password pair for an unknown account, then repeat
with a known account and an incorrect password. If a disabled-account fixture
is available, repeat with that account.

**Expected results:**

- Each case stays on `/login`.
- The public message is the same generic Authentication failure and does not
  reveal whether the email is unknown, disabled, or merely has a wrong
  password.
- The password input is cleared after submission.
- No password, hash, token, cookie, or credential body appears in console logs,
  URL, or rendered error details.

### 8. Unauthenticated `/app` protection

**Starting state:** Fresh context with no session, seeded as a visitor.

Navigate directly to `/app`.

**Expected results:**

- The application may briefly show the secure-session restoration state.
- Protected headings, identity text, logout controls, and workspace content are
  not visible during the bootstrap/redirect transition (no protected-content
  flash).
- It ultimately redirects to `/login`.
- No protected shell or `Signed in as` text is visible.
- The failed bootstrap is treated as an unauthenticated state, not an infinite
  refresh/reload loop.

### 9. Authenticated redirects from `/login` and `/register`

**Starting state:** Context created with the real authenticated `storageState`.

Navigate to `/login`, then `/register`.

**Expected results:**

- Each public Authentication route redirects to `/app`.
- The protected identity remains visible.
- No registration or login form is submitted and no second account is created.

### 10. Session recovery after browser reload

**Starting state:** Real authenticated session reached through registration or a
  setup state; `/app` is visible.

**Steps:** Reload the page (a full browser reload, not client-side navigation).

**Expected results:**

- The in-memory access token is lost during reload.
- The provider performs one controlled refresh through the HTTPS same-origin
  route and then validates the current user.
- `/app` is restored with the same safe user identity.
- There is no refresh storm, hard-reload loop, or token in persistent storage.

### 11. Logout

**Starting state:** Authenticated `/app` session.

Click `Sign out`.

**Expected results:**

- The real logout request succeeds and the browser navigates to `/login`.
- The current session is revoked server-side and the refresh cookie is cleared
  using compatible attributes.
- The protected identity is no longer rendered.
- The application clears its in-memory access-token/session state.

### 12. Session invalidation after logout

**Starting state:** The same context after successful logout.

Reload `/login`, then attempt to open `/app`.

**Expected results:**

- Refresh is rejected as an absent/revoked session and is handled as a normal
  unauthenticated state.
- The browser remains on `/login` (or is redirected there from `/app`) without
  an infinite request loop.
- No protected shell appears and a new login is required.

### 13. Cross-tab logout invalidation (recommended)

**Starting state:** Two pages in the same real browser context, with one
authenticated session established through the visible registration flow or a
temporary authenticated `storageState`. Both pages are on `/app` and show the
same safe user identity.

**Steps:** Click `Sign out` in the first page, then observe the second page
without reloading it.

**Expected results:**

- The first page redirects to `/login` and clears the server session.
- The second page reacts to the supported browser session signal, clears its
  in-memory session, and no longer shows protected content.
- Navigating or reloading the second page remains unauthenticated.

If the browser/runtime cannot deliver the supported same-origin session signal
reliably, record the limitation and keep this scenario as a documented
non-blocking assessment; do not invent a backend or test-only invalidation
endpoint.

## Cross-scenario security assertions

For every real Authentication scenario:

- Use `https://app.localhost:3443`; never switch to HTTP localhost ports.
- Do not intercept `/api/v1/auth/*` or fabricate responses.
- Do not read or write the HttpOnly refresh cookie from application JavaScript.
- Do not persist access tokens in browser storage or test fixtures beyond the
  temporary Playwright context state needed to restore a real cookie session.
- Do not assert raw token values or include them in artifacts.
- Verify no unexpected M2/business routes are introduced by the Auth suite.

## Execution and evidence

Start the real HTTPS stack before running the Auth project, then run:

```bash
make auth-dev-up
pnpm --filter @portfolio/web exec playwright test \
  --config playwright.auth.config.ts
```

For a single scenario file, retain the Auth config explicitly:

```bash
pnpm --filter @portfolio/web exec playwright test \
  --config playwright.auth.config.ts \
  tests/auth-e2e/authentication.spec.ts
```

The default `playwright.config.ts` targets `tests/e2e`, so it must not be used
for these Auth scenarios. Collect traces only on retry and review reports for
credential/token leakage before sharing them.

## Planner review coverage matrix

| Area | Required scenario | Plan section | Status | Required action |
| --- | --- | --- | --- | --- |
| Bootstrap | Seed reaches usable unauthenticated `/register` | 1 | COVERED | None |
| Register | Form and navigation | 1 | COVERED | None |
| Register verify | Required fields and invalid email | 3 | COVERED | None |
| Password verification | Under-12 password rejected | 3, 6 | COVERED | None |
| Registration successful | Valid registration to `/app` | 2 | COVERED | None |
| Duplicate registration | Safe generic duplicate failure | 4 | COVERED | None |
| Login | Form and navigation | 5 | COVERED | None |
| Login validation | Empty/invalid input | 6 | COVERED | None |
| Login successful | Valid login to `/app` | 5 | COVERED | None |
| Login incorrect | Unknown/wrong credentials not disclosed | 7 | COVERED | None |
| Protected path | `/app` redirects with no content flash | 8 | COVERED | None |
| Session recovery | Reload restores `/app` | 10 | COVERED | None |
| Public route security | Authenticated `/login` and `/register` redirect | 9 | COVERED | None |
| Logout | Sign out to `/login` | 11 | COVERED | None |
| Logout persistence | Reload/direct `/app` remain unauthenticated | 12 | COVERED | None |
| Browser security smoke | Cookie attributes and storage boundary | 2, 10, Cross-scenario security assertions | COVERED | None |
| Cross-tab | Logout invalidates another supported tab | 13 | COVERED | SHOULD run when same-context session signaling is available; otherwise document limitation |
| Actual topology | HTTPS Caddy stack and no Auth mocks | Scope, Cross-scenario security assertions | COVERED | None |

Review totals: 18 required/should scenarios, 18 covered, 0 partial, 0
missing, 0 not applicable. Six covered areas overlap existing deterministic
regression coverage and are intentionally marked as duplicate coverage rather
than missing behavior: valid registration, protected `/app` redirect, public
route redirects, reload recovery, logout, and post-logout persistence. Existing
tests in `tests/auth-e2e/authentication.spec.ts` remain preserved; Generator
should add breadth from sections 1, 3, 4, 5, 6, 7, and 13 rather than create a
second copy of that regression file.

## Completion criteria

The Authentication plan is covered when all required scenario groups and the
cross-tab assessment pass against the real HTTPS stack, validation and
generic-error assertions pass, storage and cookie security assertions pass, and
no test relies on mocked Authentication responses or an HTTP bypass.
