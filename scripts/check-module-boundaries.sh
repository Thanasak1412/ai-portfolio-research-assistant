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

echo "Module boundary check passed"
