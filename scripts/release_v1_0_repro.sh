#!/usr/bin/env bash
set -euo pipefail

report_path=""
source_path="examples/flow_hello.tetra"
targets=("linux-x64" "macos-x64" "windows-x64" "wasm32-wasi" "wasm32-web")

usage() {
  cat <<'USAGE'
Usage: bash scripts/release_v1_0_repro.sh --report PATH [--source examples/flow_hello.tetra]

Produces a reproducibility proof JSON for every v1 build-only/release target.
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

json_escape() {
  local s="${1-}"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  printf '%s' "$s"
}

target_ext() {
  case "$1" in
    windows-x64)
      printf '.exe'
      ;;
    wasm32-wasi|wasm32-web)
      printf '.wasm'
      ;;
    *)
      printf ''
      ;;
  esac
}

artifacts_json="$tmp_dir/artifacts.jsonl"
: >"$artifacts_json"
all_match="true"
legacy_native_json=""
legacy_wasm_json=""

for target in "${targets[@]}"; do
  ext="$(target_ext "$target")"
  out_a="$tmp_dir/${target}-a${ext}"
  out_b="$tmp_dir/${target}-b${ext}"

  ./tetra build --target "$target" -o "$out_a" "$source_path"
  ./tetra build --target "$target" -o "$out_b" "$source_path"

  hash_a="sha256:$(sha256sum "$out_a" | awk '{print $1}')"
  hash_b="sha256:$(sha256sum "$out_b" | awk '{print $1}')"
  match="false"
  if cmp -s "$out_a" "$out_b"; then
    match="true"
  else
    all_match="false"
  fi
  size_a="$(wc -c <"$out_a" | tr -d '[:space:]')"
  size_b="$(wc -c <"$out_b" | tr -d '[:space:]')"

  artifact_json="$(printf '{"target":"%s","hash_a":"%s","hash_b":"%s","size_a":%s,"size_b":%s,"match":%s}' \
    "$(json_escape "$target")" \
    "$(json_escape "$hash_a")" \
    "$(json_escape "$hash_b")" \
    "$size_a" \
    "$size_b" \
    "$match")"
  printf '%s\n' "$artifact_json" >>"$artifacts_json"
  if [[ "$target" == "linux-x64" ]]; then
    legacy_native_json="$artifact_json"
  fi
  if [[ "$target" == "wasm32-wasi" ]]; then
    legacy_wasm_json="$artifact_json"
  fi
done

mkdir -p "$(dirname "$report_path")"
{
  echo "{"
  echo '  "schema": "tetra.reproducible-build-proof.v1alpha1",'
  echo '  "timestamp_policy": "no wall-clock timestamps; release gate summary records run time",'
  printf '  "source": "%s",\n' "$(json_escape "$source_path")"
  printf '  "native": %s,\n' "$legacy_native_json"
  printf '  "wasm": %s,\n' "$legacy_wasm_json"
  echo '  "artifacts": ['
  awk 'NR > 1 { printf ",\n" } { printf "    %s", $0 } END { if (NR > 0) printf "\n" }' "$artifacts_json"
  echo '  ],'
  printf '  "status": "%s"\n' "$( [[ "$all_match" == "true" ]] && echo pass || echo fail )"
  echo "}"
} >"$report_path"

if [[ "$all_match" != "true" ]]; then
  echo "release_v1_0_repro: reproducible build mismatch" >&2
  exit 1
fi

echo "reproducible build proof: $report_path"
