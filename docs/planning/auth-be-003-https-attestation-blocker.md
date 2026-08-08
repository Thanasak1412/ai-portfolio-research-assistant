# AUTH-BE-003 HTTPS Attestation Decision

- Status: Resolved
- Scope: Runtime activation of the refresh and logout browser-session routes
- Recorded: 2026-08-08
- Policy sources: `AUTH_BROWSER_SECURITY-v1`, `HTTPS_ATTESTATION-v1`, ADR-016, ADR-017, ADR-021

## Resolved decision

The approved deployment topology terminates TLS at a same-origin reverse proxy and keeps the Go API private. [HTTPS_ATTESTATION-v1](../policies/HTTPS_ATTESTATION-v1.md) and ADR-021 now define how the API proves the original browser request used HTTPS after TLS termination: direct TLS, or one exact `X-Forwarded-Proto: https` assertion from the direct peer in the separate `AUTH_TRUSTED_HTTPS_PROXY_CIDRS` set.

The policy explicitly rejects arbitrary forwarded scheme assertions, forwarding chains, and untrusted plaintext peers. Fiber's direct-proxy transport view is therefore not used as proof of original HTTPS without the approved direct-peer check.

## Remaining implementation work

The application operations, strict HTTP DTOs, cookie serialization, Bearer middleware, exact Origin/header checks, and an injectable HTTPS-attestation boundary are already available. Runtime composition must remain unmounted until `AUTH-BE-003A` implements this approved attestor, validates `AUTH_TRUSTED_HTTPS_PROXY_CIDRS`, and adds the required HTTPS proxy tests. This decision does not itself activate routes or weaken Secure-cookie, exact-Origin, or HTTPS requirements.

## Closure evidence

The decision records trusted TLS-terminating CIDRs, exact header semantics, direct-TLS behavior, multi-hop behavior, untrusted-peer handling, and local/CI/staging/production verification. `AUTH-BE-003A — Implement trusted HTTPS attestation and activate Authentication runtime routes` is now the only follow-on task for this decision.
