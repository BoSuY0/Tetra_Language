#!/usr/bin/env bash
set -euo pipefail

report_path=""
source_path="examples/flow_hello.tetra"
native_target="linux-x64"
wasm_target="wasm32-wasi"

usage() {
  cat <<'USAGE'
Usage: bash scripts/release_v1_0_repro.sh --report PATH [--source examples/flow_hello.tetra]

Produces a reproducibility proof JSON for one native and one WASM target.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report)
      report_path="$2"
      shift 2
      ;;
    --source)
      source_path="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "release_v1_0_repro: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_path" ]]; then
  echo "release_v1_0_repro: --report is required" >&2
  exit 2
fi

if [[ ! -f "$source_path" ]]; then
  echo "release_v1_0_repro: missing source $source_path" >&2
  exit 1
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

native_a="$tmp_dir/native-a"
native_b="$tmp_dir/native-b"
wasm_a="$tmp_dir/wasm-a.wasm"
wasm_b="$tmp_dir/wasm-b.wasm"

./tetra build --target "$native_target" -o "$native_a" "$source_path"
./tetra build --target "$native_target" -o "$native_b" "$source_path"
./tetra build --target "$wasm_target" -o "$wasm_a" "$source_path"
./tetra build --target "$wasm_target" -o "$wasm_b" "$source_path"

native_hash_a="sha256:$(sha256sum "$native_a" | awk '{print $1}')"
native_hash_b="sha256:$(sha256sum "$native_b" | awk '{print $1}')"
wasm_hash_a="sha256:$(sha256sum "$wasm_a" | awk '{print $1}')"
wasm_hash_b="sha256:$(sha256sum "$wasm_b" | awk '{print $1}')"

native_match="false"
wasm_match="false"
if cmp -s "$native_a" "$native_b"; then
  native_match="true"
fi
if cmp -s "$wasm_a" "$wasm_b"; then
  wasm_match="true"
fi

mkdir -p "$(dirname "$report_path")"
cat >"$report_path" <<JSON
{
  "schema": "tetra.reproducible-build-proof.v1alpha1",
  "generated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "source": "$source_path",
  "native": {
    "target": "$native_target",
    "hash_a": "$native_hash_a",
    "hash_b": "$native_hash_b",
    "match": $native_match
  },
  "wasm": {
    "target": "$wasm_target",
    "hash_a": "$wasm_hash_a",
    "hash_b": "$wasm_hash_b",
    "match": $wasm_match
  },
  "status": "$( [[ "$native_match" == "true" && "$wasm_match" == "true" ]] && echo pass || echo fail )"
}
JSON

if [[ "$native_match" != "true" || "$wasm_match" != "true" ]]; then
  echo "release_v1_0_repro: reproducible build mismatch (native=$native_match wasm=$wasm_match)" >&2
  exit 1
fi

echo "reproducible build proof: $report_path"
