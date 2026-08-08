# AUTH-BE-003 HTTPS Attestation Decision

- Status: Resolved
- Scope: Runtime activation of the refresh and logout browser-session routes
- Recorded: 2026-08-08
- Policy sources: `AUTH_BROWSER_SECURITY-v1`, `HTTPS_ATTESTATION-v1`, ADR-016, ADR-017, ADR-021

## Resolved decision

The approved deployment topology terminates TLS at a same-origin reverse proxy and keeps the Go API private. [HTTPS_ATTESTATION-v1](../policies/HTTPS_ATTESTATION-v1.md) and ADR-021 now define how the API proves the original browser request used HTTPS after TLS termination: direct TLS, or one exact `X-Forwarded-Proto: https` assertion from the direct peer in the separate `AUTH_TRUSTED_HTTPS_PROXY_CIDRS` set.

The policy explicitly rejects arbitrary forwarded scheme assertions, forwarding chains, and untrusted plaintext peers. Fiber's direct-proxy transport view is therefore not used as proof of original HTTPS without the approved direct-peer check.

## Historical implementation record

The application operations, strict HTTP DTOs, cookie serialization, Bearer middleware, exact Origin/header checks, and an injectable HTTPS-attestation boundary were available before runtime activation. `AUTH-BE-003A` implements the concrete attestor, validates `AUTH_TRUSTED_HTTPS_PROXY_CIDRS`, and mounts the approved Identity routes. This decision does not weaken Secure-cookie, exact-Origin, or HTTPS requirements.

## Closure evidence

The decision records trusted TLS-terminating CIDRs, exact header semantics, direct-TLS behavior, multi-hop behavior, untrusted-peer handling, and local/CI/staging/production verification. Runtime activation is delivered by `AUTH-BE-003A`; its pull-request review and merge remain subject to ADR-013 governance.
