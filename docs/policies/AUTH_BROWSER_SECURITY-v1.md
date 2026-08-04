# AUTH_BROWSER_SECURITY-v1

**Status:** Approved | **Effective:** 2026-08-04 | **Policy version:** v1

## Topology

Production and staging use one same-origin HTTPS public endpoint. `AUTH_PUBLIC_ORIGIN` is a required absolute HTTPS origin with no path, query, fragment, credentials, or trailing slash. An HTTPS reverse proxy serves Next.js at `/` and proxies `/api/v1` to the private API. There is no public browser cross-origin API topology for M1; CORS is therefore disabled and wildcard credentialed CORS is prohibited.

Local integration/E2E uses `https://app.localhost:3443`, with a reverse proxy routing `/` to web port 3000 and `/api/v1` to API port 8080. Developers use `mkcert` to install a trusted local CA and create a certificate for `app.localhost`; CI generates an isolated test certificate and Playwright accepts that test certificate only. Plain `http://localhost` remains useful only for non-cookie unit checks and cannot validate browser authentication.

## Refresh cookie

Name: `pra_rt_v1`. It is Secure, HttpOnly, SameSite=Lax, host-only (no Domain attribute), and has Path `/api/v1/auth`. It is issued/rotated with Max-Age equal to the smaller of 30 days and remaining 90-day family lifetime; server-side expiry remains authoritative. Clearing uses the identical name/path/host-only/Secure/SameSite attributes with Max-Age zero. JavaScript never reads or stores it.

## CSRF and origin policy

Refresh and logout require: HTTPS; cookie presence; `Origin` exactly equal to `AUTH_PUBLIC_ORIGIN`; and `X-Requested-With: portfolio-web`. Missing, malformed, or mismatched Origin is rejected. There is no Referer fallback and no non-browser cookie client in M1. SameSite is defense in depth, not the sole control. API CORS sends no allow-origin response; unexpected preflight/origin requests are rejected. Allowed browser methods are limited by routing, and the client sends only JSON, Authorization, X-Correlation-ID, and X-Requested-With where applicable.

Tests cover cookie attributes/path/clearing, valid origin, rejected/missing origin, cross-site refresh/logout, direct HTTP limitation, same-origin HTTPS reload recovery, and preflight rejection.
