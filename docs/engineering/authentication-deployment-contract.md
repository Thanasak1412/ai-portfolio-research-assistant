# Authentication Deployment Contract

Required before a staging/production API starts: `AUTH_PUBLIC_ORIGIN` (single HTTPS public origin); Ed25519 active key ID/private key/verification ring; `AUTH_NETWORK_HMAC_KEY`; `AUTH_RATE_LIMIT_HMAC_KEY`; exact `AUTH_TRUSTED_PROXY_CIDRS`; exact `AUTH_TRUSTED_HTTPS_PROXY_CIDRS`; and approved database URL. Railway service variables are the approved staging/production secret store. Local development uses ignored owner-only development secrets; tests generate ephemeral signing material.

The public proxy is the sole TLS endpoint and forwards to private API/web services. It must preserve the configured origin, send X-Forwarded-For only from the approved ingress chain, strip every client-supplied `X-Forwarded-Proto`, and write exactly one `X-Forwarded-Proto: https` for HTTPS frontend traffic. It must never expose API/private ports publicly. Startup rejects invalid origin, malformed keys, missing active kid, or absent/invalid production trusted-proxy configuration. The HTTPS proxy CIDRs authorize scheme attestation only and must not be inferred from client-IP proxy CIDRs.

The HTTPS attestation decision is approved in [HTTPS_ATTESTATION-v1](../policies/HTTPS_ATTESTATION-v1.md), but routes remain unmounted until `AUTH-BE-003A` implements it in runtime composition. `X-Forwarded-Proto` is never trusted merely because it is present.
