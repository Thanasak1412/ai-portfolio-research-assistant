#!/bin/sh
set -eu

output=${1:-.compose.auth.e2e.env}
tls_directory=${AUTH_TLS_DIR:-.local/ci-auth-tls}
AUTH_TLS_DIR="$tls_directory" sh scripts/prepare-compose-auth-env.sh "$output"
printf '%s\n' 'COMPOSE_DATABASE_URL=postgres://portfolio:portfolio_test_local_only@postgres-test:5432/portfolio_test?sslmode=disable' >> "$output"
chmod 600 "$output"
