#!/usr/bin/env bash
set -euo pipefail

report_path=""
source_path="examples/ui_web_smoke.tetra"
targets=("linux-x64" "macos-x64" "windows-x64" "wasm32-wasi" "wasm32-web")

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/v1_0/reproducible-build.sh --report PATH [--source examples/ui_web_smoke.tetra]

Produces a reproducibility proof JSON for every v1 build-only/release target.
USAGE
}

require_flag_value() {
  local flag="$1"
  local value="${2:-}"
  if [[ -z "$value" ]]; then
    echo "release/v1_0/reproducible-build: ${flag} requires a path" >&2
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
    echo "release/v1_0/reproducible-build: refusing to use directory report path: $report_path" >&2
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
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "release/v1_0/reproducible-build: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_path" ]]; then
  echo "release/v1_0/reproducible-build: --report is required" >&2
  exit 2
fi
report_path="$(normalize_relative_dash_path "$report_path")"
source_path="$(normalize_relative_dash_path "$source_path")"
prepare_output_file_path

if [[ ! -f "$source_path" ]]; then
  echo "release/v1_0/reproducible-build: missing source $source_path" >&2
  exit 1
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
build_a_dir="$tmp_dir/a"
build_b_dir="$tmp_dir/b"
mkdir -p "$build_a_dir" "$build_b_dir"

json_escape() {
  local s="${1-}"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  printf '%s' "$s"
}

sha256_file() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
    return 0
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
    return 0
  fi
  if command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 "$file" | awk '{print $NF}'
    return 0
  fi
  echo "release/v1_0/reproducible-build: no sha256 tool found (expected sha256sum, shasum, or openssl)" >&2
  exit 1
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

artifact_base_path() {
  local path="$1"
  case "$path" in
    *.wasm|*.WASM|*.exe|*.EXE)
      printf '%s' "${path%.*}"
      ;;
    *)
      printf '%s' "$path"
      ;;
  esac
}

artifact_extension() {
  case "$1" in
    *.ui.web.mjs)
      printf '.ui.web.mjs'
      ;;
    *.ui.json)
      printf '.ui.json'
      ;;
    *.ui.html)
      printf '.ui.html'
      ;;
    *.mjs)
      printf '.mjs'
      ;;
    *.wasm)
      printf '.wasm'
      ;;
    *.exe)
      printf '.exe'
      ;;
    *)
      printf ''
      ;;
  esac
}

record_artifact_pair() {
  local target="$1"
  local path_a="$2"
  local path_b="$3"
  local sidecar="$4"
  local required="${5:-false}"
  last_artifact_json=""
  last_artifact_recorded="false"

  local exists_a="false"
  local exists_b="false"
  if [[ -f "$path_a" ]]; then
    exists_a="true"
  fi
  if [[ -f "$path_b" ]]; then
    exists_b="true"
  fi
  if [[ "$sidecar" == "true" && "$required" != "true" && "$exists_a" == "false" && "$exists_b" == "false" ]]; then
    return 0
  fi

  local hash_a_json="null"
  local hash_b_json="null"
  local size_a_json="null"
  local size_b_json="null"
  if [[ "$exists_a" == "true" ]]; then
    hash_a_json="\"sha256:$(sha256_file "$path_a")\""
    size_a_json="$(wc -c <"$path_a" | tr -d '[:space:]')"
  fi
  if [[ "$exists_b" == "true" ]]; then
    hash_b_json="\"sha256:$(sha256_file "$path_b")\""
    size_b_json="$(wc -c <"$path_b" | tr -d '[:space:]')"
  fi

  local match="false"
  if [[ "$exists_a" == "true" && "$exists_b" == "true" ]] && cmp -s "$path_a" "$path_b"; then
    match="true"
    matched_count=$((matched_count + 1))
  else
    all_match="false"
    mismatched_count=$((mismatched_count + 1))
  fi

  local extension
  extension="$(artifact_extension "$path_a")"
  last_artifact_json="$(printf '{"target":"%s","extension":"%s","sidecar":%s,"exists_a":%s,"exists_b":%s,"hash_a":%s,"hash_b":%s,"size_a":%s,"size_b":%s,"match":%s}' \
    "$(json_escape "$target")" \
    "$(json_escape "$extension")" \
    "$sidecar" \
    "$exists_a" \
    "$exists_b" \
    "$hash_a_json" \
    "$hash_b_json" \
    "$size_a_json" \
    "$size_b_json" \
    "$match")"
  printf '%s\n' "$last_artifact_json" >>"$artifacts_json"
  last_artifact_recorded="true"
}

artifacts_json="$tmp_dir/artifacts.jsonl"
: >"$artifacts_json"
all_match="true"
legacy_native_json=""
legacy_wasm_json=""
matched_count=0
mismatched_count=0
source_sha256="sha256:$(sha256_file "$source_path")"
compiler_version="$(./tetra version)"
required_ui_sidecars="false"
if [[ "$source_path" == "examples/ui_web_smoke.tetra" ]]; then
  required_ui_sidecars="true"
fi

for target in "${targets[@]}"; do
  ext="$(target_ext "$target")"
  out_a="$build_a_dir/${target}${ext}"
  out_b="$build_b_dir/${target}${ext}"

  ./tetra build --target "$target" -o "$out_a" "$source_path"
  ./tetra build --target "$target" -o "$out_b" "$source_path"

  record_artifact_pair "$target" "$out_a" "$out_b" "false"
  artifact_json="$last_artifact_json"
  if [[ "$target" == "linux-x64" ]]; then
    legacy_native_json="$artifact_json"
  fi
  if [[ "$target" == "wasm32-wasi" ]]; then
    legacy_wasm_json="$artifact_json"
  fi
  case "$target" in
    wasm32-wasi)
      base_a="$(artifact_base_path "$out_a")"
      base_b="$(artifact_base_path "$out_b")"
      record_artifact_pair "$target" "$base_a.ui.json" "$base_b.ui.json" "true" "$required_ui_sidecars"
      ;;
    wasm32-web)
      base_a="$(artifact_base_path "$out_a")"
      base_b="$(artifact_base_path "$out_b")"
      record_artifact_pair "$target" "$base_a.mjs" "$base_b.mjs" "true" "$required_ui_sidecars"
      record_artifact_pair "$target" "$base_a.ui.json" "$base_b.ui.json" "true" "$required_ui_sidecars"
      record_artifact_pair "$target" "$base_a.ui.web.mjs" "$base_b.ui.web.mjs" "true" "$required_ui_sidecars"
      record_artifact_pair "$target" "$base_a.ui.html" "$base_b.ui.html" "true" "$required_ui_sidecars"
      ;;
  esac
done

{
  echo "{"
  echo '  "schema": "tetra.reproducible-build-proof.v1alpha1",'
  echo '  "timestamp_policy": "no wall-clock timestamps; release gate summary records run time",'
  printf '  "compiler_version": "%s",\n' "$(json_escape "$compiler_version")"
  printf '  "source": "%s",\n' "$(json_escape "$source_path")"
  printf '  "source_sha256": "%s",\n' "$(json_escape "$source_sha256")"
  printf '  "target_count": %s,\n' "${#targets[@]}"
  printf '  "matched_count": %s,\n' "$matched_count"
  printf '  "mismatched_count": %s,\n' "$mismatched_count"
  echo '  "targets": ['
  for i in "${!targets[@]}"; do
    sep=","
    if [[ "$i" -eq $((${#targets[@]} - 1)) ]]; then
      sep=""
    fi
    printf '    "%s"%s\n' "$(json_escape "${targets[$i]}")" "$sep"
  done
  echo '  ],'
  printf '  "native": %s,\n' "$legacy_native_json"
  printf '  "wasm": %s,\n' "$legacy_wasm_json"
  echo '  "artifacts": ['
  awk 'NR > 1 { printf ",\n" } { printf "    %s", $0 } END { if (NR > 0) printf "\n" }' "$artifacts_json"
  echo '  ],'
  printf '  "status": "%s"\n' "$( [[ "$all_match" == "true" ]] && echo pass || echo fail )"
  echo "}"
} >"$report_path"

if [[ "$all_match" != "true" ]]; then
  echo "release/v1_0/reproducible-build: reproducible build mismatch" >&2
  exit 1
fi

echo "reproducible build proof: $report_path"
