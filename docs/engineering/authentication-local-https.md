# Authentication Local HTTPS

Browser Authentication tests must use `https://app.localhost:3443`; plain HTTP cannot exercise the approved Secure refresh cookie. Install `mkcert`, trust its local CA, and issue a development-only certificate for `app.localhost`. The later M1 reverse-proxy Compose profile terminates TLS on port 3443, proxies `/` to web port 3000, and proxies `/api/v1` to API port 8080. Certificate files remain ignored and owner-readable only.

The browser/E2E base URL uses the HTTPS proxy. CI uses an isolated generated certificate accepted only by its test browser. Do not set `Secure=false`, add a cookie Domain, or use the direct API port for browser-auth validation. If certificate trust fails, verify the local CA installation, hostname, proxy route, and browser profile before testing Authentication.
