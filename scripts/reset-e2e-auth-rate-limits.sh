#!/bin/sh
set -eu

# TEST-ONLY suite isolation: this script clears operational rate-limit events
# only in the tmpfs-backed postgres-test / portfolio_test database. It never
# accepts a database URL or targets the persistent postgres / portfolio service.
environment_file=${1:-.compose.auth.e2e.env}
expected_database_url='COMPOSE_DATABASE_URL=postgres://portfolio:portfolio_test_local_only@postgres-test:5432/portfolio_test?sslmode=disable'

if [ ! -f "$environment_file" ]; then
  echo "M2 E2E environment file not found: $environment_file" >&2
  exit 1
fi

if ! grep -Fqx "$expected_database_url" "$environment_file"; then
  echo "Refusing to reset: COMPOSE_DATABASE_URL must target postgres-test / portfolio_test" >&2
  exit 1
fi

docker compose --env-file "$environment_file" exec -T postgres-test \
  psql -v ON_ERROR_STOP=1 -U portfolio -d portfolio_test \
  -c 'DELETE FROM auth_rate_limit_events;'

echo "Reset test-only Authentication rate-limit events in postgres-test / portfolio_test."
