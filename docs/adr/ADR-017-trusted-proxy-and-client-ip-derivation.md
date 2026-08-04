# ADR-017 — Trusted Proxy and Client IP Derivation

**Status:** Accepted | **Date:** 2026-08-04

## Decision

Use CLIENT_NETWORK_IDENTITY-v1: trust forwarding headers only when the direct peer is in configured trusted CIDRs; parse X-Forwarded-For right-to-left; canonicalize IPs; and store/use only HMAC-derived IP representations for audit/rate limits.

## Consequences

Public deployments fail startup if forwarding is enabled without exact trusted CIDRs. Local/CI direct traffic remains deterministic. Spoofed headers cannot select a rate-limit identity.

## Testing and revision

Test direct/trusted/untrusted chains, malformed inputs, IPv4/IPv6/mapped forms. Ingress changes require CIDR review and deployment verification.
