#!/bin/sh
set -eu

output=${1:-.compose.auth.env}
umask 077
temporary_directory=$(mktemp -d)
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM

openssl genpkey -algorithm ED25519 -out "$temporary_directory/private.pem" >/dev/null 2>&1
private_key=$(openssl pkcs8 -topk8 -nocrypt -in "$temporary_directory/private.pem" -outform DER | base64 | tr -d '\n')
public_key=$(openssl pkey -in "$temporary_directory/private.pem" -pubout -outform DER | base64 | tr -d '\n')
network_key=$(openssl rand -base64 32 | tr -d '\n')
rate_limit_key=$(openssl rand -base64 32 | tr -d '\n')

cat > "$output" <<EOF
COMPOSE_APP_ENV=test
AUTH_PUBLIC_ORIGIN=https://app.localhost:3443
AUTH_JWT_ACTIVE_KID=auth-ed25519-20260808-01
AUTH_JWT_ACTIVE_PRIVATE_KEY_B64=$private_key
AUTH_JWT_VERIFICATION_KEYS_JSON=[{"kid":"auth-ed25519-20260808-01","publicKeyB64":"$public_key"}]
AUTH_JWT_LOCAL_KEY_RING_PATH=
AUTH_TRUSTED_PROXY_CIDRS=
AUTH_TRUSTED_HTTPS_PROXY_CIDRS=
AUTH_NETWORK_HMAC_KEY=$network_key
AUTH_RATE_LIMIT_HMAC_KEY=$rate_limit_key
EOF
chmod 600 "$output"
