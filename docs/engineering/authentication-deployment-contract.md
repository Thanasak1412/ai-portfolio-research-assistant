# Authentication Deployment Contract

Required before a staging/production API starts: `AUTH_PUBLIC_ORIGIN` (single HTTPS public origin); Ed25519 active key ID/private key/verification ring; `AUTH_NETWORK_HMAC_KEY`; `AUTH_RATE_LIMIT_HMAC_KEY`; exact `AUTH_TRUSTED_PROXY_CIDRS`; and approved database URL. Railway service variables are the approved staging/production secret store. Local development uses ignored owner-only development secrets; tests generate ephemeral signing material.

The public proxy is the sole TLS endpoint and forwards to private API/web services. It must preserve the configured origin, send X-Forwarded-For only from the approved ingress chain, and never expose API/private ports publicly. Startup rejects invalid origin, malformed keys, missing active kid, or absent production trusted-proxy CIDRs.
