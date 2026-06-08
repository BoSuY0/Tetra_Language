#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"

exec bash "$repo_root/scripts/release/post_v0_4/wasm-ui-gui-production-gate.sh" "$@"
