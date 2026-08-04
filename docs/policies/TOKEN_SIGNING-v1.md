# TOKEN_SIGNING-v1

**Status:** Approved | **Effective:** 2026-08-04 | **Policy version:** v1

## JWT policy

Access tokens use only EdDSA with Ed25519, have a 15-minute lifetime, and carry issuer, audience, subject, issued-at, expiry, JWT ID, and key ID. Audience is exactly `ai-portfolio-research-assistant-api`. Issuer is the validated HTTPS `AUTH_PUBLIC_ORIGIN` without a trailing slash. Subject is the immutable opaque user ID. JWT IDs are cryptographically random 128-bit values. Permitted clock skew is 60 seconds.

Verification allows only `EdDSA`; rejects missing required claims, unknown/inactive key IDs, invalid signature, incorrect issuer/audience, expiry, and issued-at/not-before more than 60 seconds in the future. Incoming header algorithm selection never changes this allowlist.

## Key ring and environments

Key IDs match `auth-ed25519-YYYYMMDD-NN`. The active key signs; the verification ring includes active and overlap public keys. Private keys are PKCS#8 DER base64; public keys are SPKI DER base64. Staging and production use Railway service-variable secrets: `AUTH_JWT_ACTIVE_KID`, `AUTH_JWT_ACTIVE_PRIVATE_KEY_B64`, and `AUTH_JWT_VERIFICATION_KEYS_JSON`. The active public key must appear in the verification ring.

Local development uses an ignored, owner-only file referenced by `AUTH_JWT_LOCAL_KEY_RING_PATH`; it must contain development-only material and never enter Git. Automated unit/integration tests generate ephemeral keys in process. GitHub Actions stores no deployment private key; tests use ephemeral fixtures. Mounted-secret support is allowed later only when it supplies the same validated interface.

Startup validates encoding, active kid/private-key match, active public presence, unique kids, and issuer/audience. Missing or invalid active material fails startup. Keys are never logged, returned, or placed in frontend variables.

## Rotation and emergency response

Create and validate a new key, publish it to the verification ring, make it active, then retain the prior public key for 24 hours after activation. This exceeds the 15-minute token lifetime plus skew and deployment propagation. Remove old private material immediately after activation; remove old public material after overlap. Emergency compromise removes the affected key immediately, rejects its outstanding tokens, records an incident, and requires affected users to reauthenticate.

Tests cover valid/invalid signatures, only-EdDSA enforcement, claims, unknown kid, expiry, future issue time, overlap, revoked key, and missing startup material.
