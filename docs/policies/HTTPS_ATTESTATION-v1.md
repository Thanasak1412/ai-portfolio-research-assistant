# HTTPS_ATTESTATION-v1

**Status:** Approved | **Effective:** 2026-08-08 | **Policy version:** v1

## Purpose and principle

This policy defines the only M1 evidence by which the private Go API may determine that the original browser request used HTTPS. The API never trusts a forwarded scheme merely because a header exists. HTTPS is established only by authenticated direct TLS at the API connection or by one exact scheme assertion received directly from an explicitly configured TLS-terminating proxy.

This policy refines, and does not replace, `AUTH_BROWSER_SECURITY-v1`, `CLIENT_NETWORK_IDENTITY-v1`, or ADR-016/ADR-017. It does not authorize CORS, a cross-origin browser API, a non-secure refresh cookie, or a client-supplied identity.

## Configuration

`AUTH_TRUSTED_HTTPS_PROXY_CIDRS` is a separate, comma-separated CIDR configuration for controlled TLS-terminating ingress peers. It must not be copied implicitly from `AUTH_TRUSTED_PROXY_CIDRS`: the latter controls client-IP forwarding, while this setting authorizes a scheme assertion.

- Each non-empty entry must be a valid CIDR; malformed entries fail startup.
- An empty setting means that forwarded scheme headers are never trusted.
- Staging and production must provide exact, non-universal ingress CIDRs. `0.0.0.0/0` and `::/0` are prohibited.
- The setting identifies only controlled ingress/TLS-termination peers. It is environment configuration, not Go source code.

## Attestation algorithm

The later attestor evaluates the direct API transport first.

1. If the API connection has a completed, authenticated TLS connection state, it is HTTPS. Forwarded scheme headers are unnecessary and ignored for this decision.
2. If the direct connection is plaintext, parse the actual socket peer as an IP address. Only when it belongs to `AUTH_TRUSTED_HTTPS_PROXY_CIDRS` may the attestor inspect `X-Forwarded-Proto`.
3. A trusted plaintext peer attests HTTPS only when there is exactly one header value and it is the exact lowercase ASCII value `https`.
4. All other cases fail closed.

The forwarded value is rejected when it is missing, empty, `http`, mixed-case, comma-separated, repeated, whitespace-padded, malformed, or any URI/text other than the exact canonical value. The implementation must inspect repeated header instances, not merely accept a parser-normalized value. It must not infer TLS from Host, Origin, port, Fiber protocol helpers, or an arbitrary forwarding chain.

An untrusted direct peer's `X-Forwarded-Proto` is ignored. A public client that reaches a plaintext API port cannot claim HTTPS by sending `X-Forwarded-Proto: https`.

## Proxy and multi-hop contract

Every approved TLS-terminating proxy must remove browser-provided `X-Forwarded-Proto`, emit exactly one authoritative replacement, use `https` for an HTTPS browser request, never append a client-provided value, and connect to the API through the controlled private network.

M1 trusts only the API's direct peer. When several infrastructure hops exist, the direct API peer must be in `AUTH_TRUSTED_HTTPS_PROXY_CIDRS` and must emit the one authoritative assertion itself. M1 does not select a left-most or right-most value from a forwarding chain.

## Independent browser controls

HTTPS attestation is necessary but not sufficient for refresh and logout. They also require cookie presence, exact `Origin == AUTH_PUBLIC_ORIGIN`, `X-Requested-With: portfolio-web`, and the approved Secure, HttpOnly, SameSite=Lax, host-only `pra_rt_v1` cookie. CORS remains disabled.

## Environment and verification contract

Local browser Authentication remains `https://app.localhost:3443`. The local TLS proxy is explicitly configured as a trusted HTTPS proxy and forwards `/api/v1` privately to the API with exactly one `X-Forwarded-Proto: https`. Plain `http://localhost:8080` may serve health and non-cookie diagnostics but must fail refresh/logout browser-security enforcement.

Browser Authentication CI must exercise the HTTPS reverse-proxy path. Unit, integration, and browser tests must prove valid trusted-proxy HTTPS acceptance; direct plaintext rejection; untrusted-peer spoof rejection; missing, `http`, repeated/comma-separated, and malformed scheme rejection; and independent Origin/custom-header rejection. Staging and production deployment records must identify the configured ingress CIDRs and prove that arbitrary public clients cannot be direct trusted peers.

## Revision and rollback

Changing the header, trusted-peer rule, direct-TLS rule, or multi-hop behavior requires a new policy and implementation-policy composition version, security review, deployment compatibility analysis, and tests. If ingress cannot uphold the proxy contract, remove its CIDR from `AUTH_TRUSTED_HTTPS_PROXY_CIDRS`; browser-session requests then fail closed until a compliant path is restored.
