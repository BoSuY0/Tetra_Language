#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cat >&2 <<'NOTICE'
release_v1_0_gate.sh is a compatibility alias.
The current public release gate is scripts/release_v0_1_2_gate.sh.
NOTICE

exec bash "$script_dir/release_v0_1_2_gate.sh" "$@"
