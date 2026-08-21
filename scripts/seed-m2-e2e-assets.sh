#!/bin/sh
set -eu

# TEST-ONLY safety boundary: this script can seed only the tmpfs-backed
# postgres-test / portfolio_test service. It never accepts a database URL or
# targets the persistent postgres / portfolio development service.
environment_file=${1:-.compose.auth.e2e.env}
expected_database_url='COMPOSE_DATABASE_URL=postgres://portfolio:portfolio_test_local_only@postgres-test:5432/portfolio_test?sslmode=disable'
fixture=apps/web/tests/m2-e2e/fixtures/assets.sql

if [ ! -f "$environment_file" ]; then
  echo "M2 E2E environment file not found: $environment_file" >&2
  exit 1
fi

if ! grep -Fqx "$expected_database_url" "$environment_file"; then
  echo "Refusing to seed: COMPOSE_DATABASE_URL must target postgres-test / portfolio_test" >&2
  exit 1
fi

if [ ! -f "$fixture" ]; then
  echo "M2 E2E fixture not found: $fixture" >&2
  exit 1
fi

docker compose --env-file "$environment_file" exec -T postgres-test \
  psql -v ON_ERROR_STOP=1 -U portfolio -d portfolio_test < "$fixture"

echo "Seeded synthetic M2 E2E Assets into postgres-test / portfolio_test."
