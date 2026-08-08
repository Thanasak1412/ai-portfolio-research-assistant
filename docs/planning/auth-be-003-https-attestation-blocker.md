# AUTH-BE-003 HTTPS Attestation Blocker

- Status: Blocked
- Scope: Runtime activation of the refresh and logout browser-session routes
- Recorded: 2026-08-08
- Policy sources: `AUTH_BROWSER_SECURITY-v1`, ADR-016, `CLIENT_NETWORK_IDENTITY-v1`

## Confirmed gap

The approved deployment topology terminates TLS at a same-origin reverse proxy and keeps the Go API private. The current policy package does not define how the API proves that the original browser request used HTTPS after TLS termination. It neither authorizes unconditional trust in `X-Forwarded-Proto` nor defines a trusted-proxy scheme-attestation header, hop-selection rule, or authenticated internal transport signal.

Fiber's request protocol describes the direct proxy-to-API connection in this topology. Trusting an arbitrary forwarded scheme would allow an untrusted direct caller to claim HTTPS and bypass an AUTH_BROWSER_SECURITY-v1 requirement.

## Impact

The application operations, strict HTTP DTOs, cookie serialization, Bearer middleware, exact Origin/header checks, and an injectable HTTPS-attestation boundary can be implemented and tested. Production composition must not activate refresh/logout until an approved attestor can be constructed. This document does not weaken the Secure-cookie, exact-Origin, or HTTPS requirements.

## Decision required

Approve a versioned scheme-attestation rule that specifies all of the following:

- the trusted TLS-terminating proxy identities or CIDRs;
- the exact scheme header or authenticated internal signal;
- canonical accepted value and multi-value rejection behavior;
- behavior for direct TLS, untrusted peers, missing values, and malformed values;
- local HTTPS, CI, staging, and production tests.

After approval, this blocker may be resolved by implementing the attestor and activating the browser-session routes in runtime composition. No Authentication or proxy policy is implicitly approved by this record.
