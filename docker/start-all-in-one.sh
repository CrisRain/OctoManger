#!/usr/bin/env bash
set -euo pipefail

export API_ADDR="${API_ADDR:-:8080}"
export WEB_DIST_DIR="${WEB_DIST_DIR:-/app/web-dist}"
export PYTHON_BIN="${PYTHON_BIN:-python3}"
export PLUGINS_DIR="${PLUGINS_DIR:-/app/plugins/modules}"
export PLUGIN_SDK_DIR="${PLUGIN_SDK_DIR:-/app/plugins/sdk/python}"
export LOG_LEVEL="${LOG_LEVEL:-info}"
export DATABASE_DSN="${DATABASE_DSN:-}"

db_conn="${DATABASE_DSN}"

wait_for_postgres() {
  local conn="$1"
  local max_attempts="${2:-60}"
  local sleep_sec="${3:-2}"

  echo "Waiting for PostgreSQL..."
  for i in $(seq 1 "${max_attempts}"); do
    if psql "${conn}" -c "SELECT 1" >/dev/null 2>&1; then
      echo "PostgreSQL is ready."
      return 0
    fi
    if [[ "${i}" -eq "${max_attempts}" ]]; then
      echo "PostgreSQL is not reachable."
      return 1
    fi
    sleep "${sleep_sec}"
  done
}

if [[ -n "${db_conn}" ]]; then
  wait_for_postgres "${db_conn}"
else
  echo "DATABASE_DSN is empty; PostgreSQL checks are skipped."
fi

# In the new architecture we start API and Worker in the background
echo "Starting migrations..."
/app/migrate

echo "Starting worker..."
/app/worker &

echo "Starting API server..."
exec /app/api

