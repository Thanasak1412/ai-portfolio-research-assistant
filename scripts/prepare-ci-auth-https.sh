#!/bin/sh
set -eu

tls_directory=${1:-.local/ci-auth-tls}
certificate="$tls_directory/app.localhost.pem"
private_key="$tls_directory/app.localhost-key.pem"

umask 077
mkdir -p "$tls_directory"
openssl req -x509 -newkey rsa:2048 -nodes -sha256 -days 1 \
  -subj "/CN=app.localhost" \
  -addext "subjectAltName=DNS:app.localhost" \
  -keyout "$private_key" -out "$certificate" >/dev/null 2>&1
chmod 600 "$private_key"
echo "Ephemeral CI Authentication TLS material is ready."
