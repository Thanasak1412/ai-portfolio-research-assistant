#!/bin/sh
set -eu

environment_file=${1:-.compose.auth.e2e.env}
base_url=https://app.localhost:3443
temporary_directory=$(mktemp -d)
trap 'rm -rf "$temporary_directory"' EXIT HUP INT TERM

curl --fail --silent --show-error --retry 20 --retry-delay 2 --insecure "$base_url/" >/dev/null
curl --fail --silent --show-error --retry 20 --retry-delay 2 --insecure "$base_url/api/v1/health/ready" >/dev/null

refresh_response=$(curl --silent --show-error --insecure --output "$temporary_directory/initial-refresh.json" --write-out '%{http_code}' \
  -X POST "$base_url/api/v1/auth/refresh" \
  -H 'Origin: https://app.localhost:3443' \
  -H 'X-Requested-With: portfolio-web')
if [ "$refresh_response" != "401" ] || ! grep -q 'SESSION_REFRESH_REJECTED' "$temporary_directory/initial-refresh.json"; then
  echo "Expected an unauthenticated browser refresh to reach Authentication and return SESSION_REFRESH_REJECTED" >&2
  exit 1
fi

expect_browser_rejection() {
  name=$1
  shift
  status=$(curl --silent --show-error --output "$temporary_directory/$name.json" --write-out '%{http_code}' "$@")
  if [ "$status" != "403" ] || ! grep -q 'BROWSER_SECURITY_REJECTED' "$temporary_directory/$name.json"; then
    echo "Expected $name to be rejected by browser security" >&2
    exit 1
  fi
}

expect_browser_rejection direct-spoof -X POST http://localhost:8080/api/v1/auth/refresh -H 'X-Forwarded-Proto: https' -H 'Origin: https://app.localhost:3443' -H 'X-Requested-With: portfolio-web'
expect_browser_rejection wrong-origin --insecure -X POST "$base_url/api/v1/auth/refresh" -H 'Origin: https://evil.example' -H 'X-Requested-With: portfolio-web'
expect_browser_rejection missing-requested-with --insecure -X POST "$base_url/api/v1/auth/refresh" -H 'Origin: https://app.localhost:3443'

if curl --silent --show-error --insecure --head "$base_url/api/v1/health/live" | grep -qi '^access-control-allow-origin:'; then
  echo "Unexpected CORS response header" >&2
  exit 1
fi
docker compose --env-file "$environment_file" ps
