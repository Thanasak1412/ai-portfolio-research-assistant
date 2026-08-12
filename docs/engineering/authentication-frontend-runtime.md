# Authentication Frontend Runtime

**Status:** AUTH-FE-002 implementation documentation
**Policy:** `AUTH_IMPLEMENTATION_POLICY-v3`

The root `AuthSessionProvider` starts in `bootstrapping`, performs one
cookie-authenticated refresh, validates identity through `/api/v1/auth/me`, and
constructs an access-token session in React memory. A rejected refresh becomes
`unauthenticated`; a temporary service failure becomes the recoverable
`bootstrap-error` state. No JWT claim is decoded and no Authentication state is
written to browser persistence.

Replay-safe bearer operations use the provider's `runAuthenticated` boundary.
Only `401 ACCESS_TOKEN_INVALID` starts recovery. Concurrent callers in one tab
share one refresh promise, the operation is retried once with the replacement
access token, and a second invalid-token response clears the memory session.
Other HTTP, business, and network failures are not automatically replayed.

Refresh and logout acquire the fixed `portfolio-auth-session-v1` Web Lock when
the browser supports Web Locks. Browsers without that API use in-tab
single-flight coordination only; no storage-based cross-tab lock is introduced.
`BroadcastChannel` carries only allowlisted state signals. It never carries a
token, credential, user record, email address, cookie, or authorization header.

`/app` is a neutral protected shell. It renders no protected content while
bootstrapping, redirects confirmed unauthenticated users to `/login`, and offers
current-session logout. Logout always clears local sensitive state; a network
failure means server revocation was not confirmed, so a later full reload may
recover the still-valid server session. JavaScript never reads or clears the
HttpOnly cookie.

The current Playwright suite verifies frontend lifecycle behavior with
same-origin route interception. End-to-end evidence using the real Secure cookie
and `https://app.localhost:3443` remains the explicit scope of `AUTH-OPS-001`.
