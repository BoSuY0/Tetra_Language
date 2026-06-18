#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
cd "$repo_root"

report_path=""
source_path="examples/flow/flow_struct_smoke.tetra"
targets=("linux-x64" "macos-x64" "windows-x64" "wasm32-wasi" "wasm32-web")

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/v1_0/binary-size.sh --report PATH [--source examples/flow/flow_struct_smoke.tetra]

Builds the v1 release target matrix and records soft/hard binary-size threshold evidence.
USAGE
}

require_flag_value() {
  local flag="$1"
  local value="${2:-}"
  if [[ -z "$value" ]]; then
    echo "release/v1_0/binary-size: ${flag} requires a path" >&2
    exit 2
  fi
}

normalize_relative_dash_path() {
  local path="$1"
  if [[ "$path" == -* ]]; then
    printf './%s' "$path"
  else
    printf '%s' "$path"
  fi
}

prepare_output_file_path() {
  if [[ -d "$report_path" || -L "$report_path" ]]; then
    echo "release/v1_0/binary-size: refusing to use directory report path: $report_path" >&2
    exit 2
  fi
  local parent_dir
  parent_dir="$(dirname "$report_path")"
  mkdir -p -- "$parent_dir"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report)
      require_flag_value "$1" "${2:-}"
      report_path="$2"
      shift 2
      ;;
    --source)
      require_flag_value "$1" "${2:-}"
      source_path="$2"
      shift 2
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "release/v1_0/binary-size: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_path" ]]; then
  echo "release/v1_0/binary-size: --report is required" >&2
  exit 2
fi
report_path="$(normalize_relative_dash_path "$report_path")"
source_path="$(normalize_relative_dash_path "$source_path")"
prepare_output_file_path
if [[ ! -f "$source_path" ]]; then
  echo "release/v1_0/binary-size: missing source $source_path" >&2
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

sha256_file() {
  local file="$1"
  if command -v sha256sum > /dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
    return 0
  fi
  if command -v shasum > /dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
    return 0
  fi
  if command -v openssl > /dev/null 2>&1; then
    openssl dgst -sha256 "$file" | awk '{print $NF}'
    return 0
  fi
  echo "release/v1_0/binary-size: no sha256 tool found (expected sha256sum, shasum, or openssl)" >&2
  exit 1
}

target_ext() {
  case "$1" in
    windows-x64)
      printf '.exe'
      ;;
    wasm32-wasi | wasm32-web)
      printf '.wasm'
      ;;
    *)
      printf ''
      ;;
  esac
}

soft_limit() {
  case "$1" in
    wasm32-wasi | wasm32-web)
      printf '524288'
      ;;
    *)
      printf '2097152'
      ;;
  esac
}

hard_limit() {
  case "$1" in
    wasm32-wasi | wasm32-web)
      printf '1048576'
      ;;
    *)
      printf '4194304'
      ;;
  esac
}

artifacts_json="$tmp_dir/artifacts.jsonl"
: > "$artifacts_json"
status="pass"
pass_count=0
warn_count=0
fail_count=0
total_size_bytes=0
max_size_bytes=0
compiler_version="$(./tetra version)"
source_sha256="sha256:$(sha256_file "$source_path")"

for target in "${targets[@]}"; do
  ext="$(target_ext "$target")"
  out="$tmp_dir/$target$ext"
  ./tetra build --target "$target" -o "$out" "$source_path"
  size="$(wc -c < "$out" | tr -d '[:space:]')"
  soft="$(soft_limit "$target")"
  hard="$(hard_limit "$target")"
  result="pass"
  if ((size > hard)); then
    result="fail"
    status="fail"
    fail_count=$((fail_count + 1))
  elif ((size > soft)); then
    result="warn"
    if [[ "$status" == "pass" ]]; then
      status="warn"
    fi
    warn_count=$((warn_count + 1))
  else
    pass_count=$((pass_count + 1))
  fi
  total_size_bytes=$((total_size_bytes + size))
  if ((size > max_size_bytes)); then
    max_size_bytes="$size"
  fi
  printf '{"target":"%s","artifact_name":"%s","size_bytes":%s,"soft_limit_bytes":%s,"hard_limit_bytes":%s,"status":"%s"}\n' \
    "$(json_escape "$target")" \
    "$(json_escape "$target$ext")" \
    "$size" \
    "$soft" \
    "$hard" \
    "$result" >> "$artifacts_json"
done

{
  echo "{"
  echo '  "schema": "tetra.binary-size-thresholds.v1alpha1",'
  echo '  "timestamp_policy": "no wall-clock timestamps; release gate summary records run time",'
  printf '  "compiler_version": "%s",\n' "$(json_escape "$compiler_version")"
  printf '  "source": "%s",\n' "$(json_escape "$source_path")"
  printf '  "source_sha256": "%s",\n' "$(json_escape "$source_sha256")"
  printf '  "target_count": %s,\n' "${#targets[@]}"
  printf '  "pass_count": %s,\n' "$pass_count"
  printf '  "warn_count": %s,\n' "$warn_count"
  printf '  "fail_count": %s,\n' "$fail_count"
  printf '  "total_size_bytes": %s,\n' "$total_size_bytes"
  printf '  "max_size_bytes": %s,\n' "$max_size_bytes"
  echo '  "targets": ['
  for i in "${!targets[@]}"; do
    sep=","
    if [[ "$i" -eq $((${#targets[@]} - 1)) ]]; then
      sep=""
    fi
    printf '    "%s"%s\n' "$(json_escape "${targets[$i]}")" "$sep"
  done
  echo '  ],'
  echo '  "threshold_policy": "soft limits require release-owner review; hard limits block the gate",'
  echo '  "artifacts": ['
  awk 'NR > 1 { printf ",\n" } { printf "    %s", $0 } END { if (NR > 0) printf "\n" }' "$artifacts_json"
  echo '  ],'
  printf '  "status": "%s"\n' "$status"
  echo "}"
} > "$report_path"

if [[ "$status" == "fail" ]]; then
  echo "release/v1_0/binary-size: one or more artifacts exceeded hard size thresholds" >&2
  exit 1
fi

echo "binary size threshold report: $report_path"
