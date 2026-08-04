# CLIENT_NETWORK_IDENTITY-v1

**Status:** Approved | **Effective:** 2026-08-04 | **Policy version:** v1

The request path is client → HTTPS reverse proxy/ingress → private API. The API uses the direct socket peer unless that peer belongs to non-empty `AUTH_TRUSTED_PROXY_CIDRS`. Only then it reads `X-Forwarded-For`, parses addresses with a strict IP parser, scans right-to-left while removing trusted hops, and selects the first remaining client address. If no valid client remains, it rejects the forwarding identity and uses the direct trusted peer only for operational diagnostics, never a forged header.

Local development, Docker Compose, browser E2E, and GitHub Actions leave trusted CIDRs empty and use direct peer identity. Staging/production must set exact ingress CIDRs; startup fails when a public deployment enables forwarding without them. IPv4 is canonicalized; IPv6 is canonicalized; IPv4-mapped IPv6 is unmapped before policy use. Malformed/oversized headers and unknown proxy peers are ignored, never trusted.

Audit and rate-limit storage use `ip_hmac_v1:` plus HMAC-SHA-256 digest with `AUTH_NETWORK_HMAC_KEY` from the same approved secret store; raw IP addresses are not persisted or used as metric labels. Tests cover direct, trusted/untrusted one and multiple hops, spoofed headers, malformed data, IPv6, mapped IPv6, and missing headers.
