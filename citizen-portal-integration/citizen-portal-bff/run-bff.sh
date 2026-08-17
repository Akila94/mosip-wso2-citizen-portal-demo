#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
ENV_FILE=".env"
while IFS='=' read -r key value; do
  [[ "$key" =~ ^[[:space:]]*# ]] && continue
  [[ -z "$key" ]] && continue
  key="$(echo -n "$key" | xargs)"
  [[ -z "$key" ]] && continue
  export "$key=$value"
done < <(grep -E '^[A-Za-z_][A-Za-z0-9_]*=' "$ENV_FILE")
exec go run ./cmd/bff
