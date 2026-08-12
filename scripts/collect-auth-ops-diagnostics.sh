#!/bin/sh
set -eu

environment_file=${1:-.compose.auth.e2e.env}
docker compose --env-file "$environment_file" ps
docker compose --env-file "$environment_file" logs --no-color --tail=200 auth-proxy api web postgres-test
