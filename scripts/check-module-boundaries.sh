#!/bin/sh
set -eu

backend_root="backend/internal"
violations=""

for module_path in "$backend_root"/*; do
  [ -d "$module_path" ] || continue
  module_name=$(basename "$module_path")
  [ "$module_name" = "platform" ] && continue
  import_path="github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/$module_name/infrastructure"
  matches=$(cd "$backend_root" && rg -n --glob '*.go' "\"$import_path" . --glob "!$module_name/**" || true)
  if [ -n "$matches" ]; then
    violations="$violations\n$matches"
  fi
done

if [ -n "$violations" ]; then
  echo "Cross-module infrastructure imports are forbidden:$violations" >&2
  exit 1
fi

identity_domain_violations=$(rg -n --glob '*.go' \
  '"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/(application|infrastructure)|"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform|"github.com/gofiber/|"github.com/jackc/pgx/|/sqlcgen"' \
  "$backend_root/identity/domain" || true)
if [ -n "$identity_domain_violations" ]; then
  echo "Identity domain imports a forbidden outer layer or infrastructure dependency:$identity_domain_violations" >&2
  exit 1
fi

identity_application_violations=$(rg -n --glob '*.go' \
  '"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure|"github.com/gofiber/|"github.com/jackc/pgx/|/sqlcgen"' \
  "$backend_root/identity/application" || true)
if [ -n "$identity_application_violations" ]; then
  echo "Identity application imports a forbidden transport or persistence implementation:$identity_application_violations" >&2
  exit 1
fi

echo "Module boundary check passed"
