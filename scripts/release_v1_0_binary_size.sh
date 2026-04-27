#!/usr/bin/env bash
set -euo pipefail

report_path=""
source_path="examples/flow_hello.tetra"
targets=("linux-x64" "macos-x64" "windows-x64" "wasm32-wasi" "wasm32-web")

usage() {
  cat <<'USAGE'
Usage: bash scripts/release_v1_0_binary_size.sh --report PATH [--source examples/flow_hello.tetra]

Builds the v1 release target matrix and records soft/hard binary-size threshold evidence.
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
      echo "release_v1_0_binary_size: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_path" ]]; then
  echo "release_v1_0_binary_size: --report is required" >&2
  exit 2
fi
if [[ ! -f "$source_path" ]]; then
  echo "release_v1_0_binary_size: missing source $source_path" >&2
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

soft_limit() {
  case "$1" in
    wasm32-wasi|wasm32-web)
      printf '524288'
      ;;
    *)
      printf '2097152'
      ;;
  esac
}

hard_limit() {
  case "$1" in
    wasm32-wasi|wasm32-web)
      printf '1048576'
      ;;
    *)
      printf '4194304'
      ;;
  esac
}

artifacts_json="$tmp_dir/artifacts.jsonl"
: >"$artifacts_json"
status="pass"

for target in "${targets[@]}"; do
  ext="$(target_ext "$target")"
  out="$tmp_dir/$target$ext"
  ./tetra build --target "$target" -o "$out" "$source_path"
  size="$(wc -c <"$out" | tr -d '[:space:]')"
  soft="$(soft_limit "$target")"
  hard="$(hard_limit "$target")"
  result="pass"
  if (( size > hard )); then
    result="fail"
    status="fail"
  elif (( size > soft )); then
    result="warn"
    if [[ "$status" == "pass" ]]; then
      status="warn"
    fi
  fi
  printf '{"target":"%s","artifact_name":"%s","size_bytes":%s,"soft_limit_bytes":%s,"hard_limit_bytes":%s,"status":"%s"}\n' \
    "$(json_escape "$target")" \
    "$(json_escape "$target$ext")" \
    "$size" \
    "$soft" \
    "$hard" \
    "$result" >>"$artifacts_json"
done

mkdir -p "$(dirname "$report_path")"
{
  echo "{"
  echo '  "schema": "tetra.binary-size-thresholds.v1alpha1",'
  printf '  "source": "%s",\n' "$(json_escape "$source_path")"
  echo '  "threshold_policy": "soft limits require release-owner review; hard limits block the gate",'
  echo '  "artifacts": ['
  awk 'NR > 1 { printf ",\n" } { printf "    %s", $0 } END { if (NR > 0) printf "\n" }' "$artifacts_json"
  echo '  ],'
  printf '  "status": "%s"\n' "$status"
  echo "}"
} >"$report_path"

if [[ "$status" == "fail" ]]; then
  echo "release_v1_0_binary_size: one or more artifacts exceeded hard size thresholds" >&2
  exit 1
fi

echo "binary size threshold report: $report_path"
