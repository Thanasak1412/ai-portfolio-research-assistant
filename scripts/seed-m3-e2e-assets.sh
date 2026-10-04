#!/bin/sh
set -eu

# TEST ONLY. No arbitrary URL, persistent service, or production DB is accepted.
environment_file=${1:-.compose.auth.e2e.env}
expected_url='postgres://portfolio:portfolio_test_local_only@postgres-test:5432/portfolio_test?sslmode=disable'
fixture=apps/web/tests/m3-e2e/fixtures/assets.sql

if [ ! -f "$environment_file" ] || [ ! -f "$fixture" ]; then
  echo 'M3 E2E environment or fixture file missing.' >&2
  exit 1
fi

# Require one unambiguous literal assignment; never source the environment file.
if ! awk -v expected="COMPOSE_DATABASE_URL=$expected_url" '
  /^[[:space:]]*(export[[:space:]]+)?COMPOSE_DATABASE_URL[[:space:]]*=/ {
    count++; if ($0 != expected) invalid=1
  }
  END { exit !(count == 1 && !invalid) }
' "$environment_file"; then
  echo 'Refusing to seed: exact postgres-test / portfolio_test target required.' >&2
  exit 1
fi
if [ "${COMPOSE_DATABASE_URL-$expected_url}" != "$expected_url" ]; then
  echo 'Refusing to seed: conflicting database environment override.' >&2
  exit 1
fi

docker compose --env-file "$environment_file" exec -T postgres-test \
  psql -v ON_ERROR_STOP=1 -U portfolio -d portfolio_test < "$fixture"
echo 'Seeded synthetic M3 E2E Assets into postgres-test / portfolio_test.'
