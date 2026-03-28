#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
web_dir="${repo_root}/apps/web"
embed_dir="${repo_root}/internal/platform/webui/dist"

if ! command -v bun >/dev/null 2>&1; then
  echo "bun is required to build the embedded web UI" >&2
  exit 1
fi

cd "${web_dir}"
if [[ -f bun.lock || -f bun.lockb ]]; then
  bun install --frozen-lockfile
else
  bun install
fi
bun run build

rm -rf "${embed_dir}"
mkdir -p "${embed_dir}"
cp -R "${web_dir}/dist/." "${embed_dir}/"
