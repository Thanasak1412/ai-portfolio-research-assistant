# ADR-021 — Trusted HTTPS Scheme Attestation

**Status:** Accepted | **Date:** 2026-08-08

## Context

M1 browser Authentication requires Secure cookies and HTTPS refresh/logout enforcement. The approved topology terminates TLS at a same-origin reverse proxy and keeps the Go API private, but the API previously had no approved way to distinguish a proxy assertion from an attacker-supplied `X-Forwarded-Proto` header. Mounting routes without that distinction would weaken `AUTH_BROWSER_SECURITY-v1`.

## Decision

Adopt `HTTPS_ATTESTATION-v1`. The API treats a request as HTTPS only when its direct connection has completed authenticated TLS or when its plaintext direct socket peer is in the separately configured `AUTH_TRUSTED_HTTPS_PROXY_CIDRS` and sends exactly one lowercase `X-Forwarded-Proto: https` value. Missing, repeated, comma-separated, malformed, mixed-case, or non-HTTPS values fail closed. Untrusted peers' forwarded-scheme values are ignored.

`AUTH_TRUSTED_HTTPS_PROXY_CIDRS` is independent from `AUTH_TRUSTED_PROXY_CIDRS`. The direct ingress peer is responsible for sanitizing and emitting the sole authoritative scheme assertion; M1 does not parse a multi-hop scheme chain.

## Alternatives considered

- Trusting every `X-Forwarded-Proto` header was rejected because a direct attacker could claim HTTPS.
- Reusing all client-IP trusted proxy CIDRs was rejected because client-IP forwarding and TLS termination have different authorities.
- Inferring HTTPS from Origin, Host, port, or framework protocol helpers was rejected because those inputs do not prove the original transport.
- Enabling CORS or a cross-origin cookie flow was rejected because the approved M1 topology is same-origin.

## Consequences

The later runtime attestor must inspect the actual peer and repeated header representation, validate configuration at startup, and mount browser-session routes only after this policy is implemented. Local and CI browser Authentication require the HTTPS proxy path. An ingress misconfiguration causes refresh/logout to fail closed rather than weaken the Secure-cookie requirement.

## Security and operational impact

Production and staging must provision exact ingress CIDRs, keep API ports private, and ensure proxies strip/rewrite the scheme header. Direct TLS remains supported without a forwarding header. `Origin`, `X-Requested-With`, cookie attributes, CORS prohibition, and client-IP derivation remain independent controls.

## Testing and revision

Test direct TLS, trusted-proxy acceptance, untrusted spoofing, missing/invalid/repeated scheme values, and independent Origin/header failures in unit, integration, and HTTPS browser CI paths. Any change to the header contract, peer trust rule, or proxy topology requires a new policy/composition version, security review, and deployment compatibility analysis.
