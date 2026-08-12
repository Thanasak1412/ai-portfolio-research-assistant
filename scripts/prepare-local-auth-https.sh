#!/bin/sh
set -eu

tls_directory=${1:-.local/auth-tls}
certificate="$tls_directory/app.localhost.pem"
private_key="$tls_directory/app.localhost-key.pem"

if ! command -v mkcert >/dev/null 2>&1; then
  echo "mkcert is required for local HTTPS. Install it, run 'mkcert -install', then retry." >&2
  exit 1
fi

umask 077
mkdir -p "$tls_directory"
if [ ! -f "$certificate" ] || [ ! -f "$private_key" ]; then
  mkcert -cert-file "$certificate" -key-file "$private_key" app.localhost
fi
chmod 600 "$private_key"
echo "Local Authentication TLS certificate is ready at $certificate."
