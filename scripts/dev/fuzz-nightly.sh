#!/usr/bin/env bash
set -euo pipefail

fuzztime="10m"
out_dir=""
short=false
min_fuzztime_seconds=1
max_fuzztime_seconds=600
max_fuzztime_label="10m"
fuzz_parallel_args=()

usage() {
  cat << 'USAGE'
Usage: bash scripts/dev/fuzz-nightly.sh [--short] [--fuzztime DURATION] [--out-dir DIR]

Runs the bounded fuzz/property/stress nightly suite one package at a time and
writes logs plus summary.md and summary.json. Crashers remain in Go's package-local testdata
fuzz corpus; copy new reproducers into deterministic tests before release.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --short)
      short=true
      fuzztime="2s"
      shift
      ;;
    --fuzztime)
      if [[ $# -lt 2 ]]; then
        echo "--fuzztime requires a value" >&2
        exit 2
      fi
      fuzztime="$2"
      shift 2
      ;;
    --out-dir)
      if [[ $# -lt 2 ]]; then
        echo "--out-dir requires a directory" >&2
        exit 2
      fi
      out_dir="$2"
      shift 2
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

reject_fuzztime() {
  local value="$1"
  echo "invalid --fuzztime: $value" >&2
  echo "--fuzztime must use Go duration units ns/us/ms/s/m/h; min 1s, max $max_fuzztime_label" >&2
  usage >&2
  exit 2
}

validate_fuzztime() {
  local value="$1"

  if [[ -z "$value" ]]; then
    reject_fuzztime "$value"
  fi
  if ! awk -v value="$value" -v min="$min_fuzztime_seconds" -v max="$max_fuzztime_seconds" '
    BEGIN {
      rest = value
      total = 0
      while (length(rest) > 0) {
        if (match(rest, /^[0-9]+([.][0-9]+)?(ns|us|ms|s|m|h)/) != 1) {
          exit 1
        }
        token = substr(rest, 1, RLENGTH)
        rest = substr(rest, RLENGTH + 1)
        if (token ~ /ns$/) {
          unit = "ns"
          scale = 0.000000001
        } else if (token ~ /us$/) {
          unit = "us"
          scale = 0.000001
        } else if (token ~ /ms$/) {
          unit = "ms"
          scale = 0.001
        } else if (token ~ /s$/) {
          unit = "s"
          scale = 1
        } else if (token ~ /m$/) {
          unit = "m"
          scale = 60
        } else if (token ~ /h$/) {
          unit = "h"
          scale = 3600
        } else {
          exit 1
        }
        amount = substr(token, 1, length(token) - length(unit)) + 0
        total += amount * scale
        if (total > max) {
          exit 1
        }
      }
      if (total < min || total > max) {
        exit 1
      }
    }
  '; then
    reject_fuzztime "$value"
  fi
}

validate_fuzztime "$fuzztime"
if [[ "$short" == true ]]; then
  fuzz_parallel_args=("-parallel=1")
fi

timestamp="$(date -u +%Y%m%d-%H%M%S)"
if [[ -z "$out_dir" ]]; then
  out_dir="reports/fuzz-nightly-$timestamp"
fi

check_out_dir_fresh() {
  local find_out_dir="$out_dir"
  if [[ "$find_out_dir" == -* ]]; then
    find_out_dir="./$find_out_dir"
  fi

  if [[ -L "$find_out_dir" ]]; then
    if [[ -d "$find_out_dir" ]]; then
      local symlink_entry
      symlink_entry="$(find -H -- "$find_out_dir" -mindepth 1 -print -quit)"
      if [[ -n "$symlink_entry" ]]; then
        echo "refusing to reuse non-empty out-dir: $out_dir" >&2
        echo "choose a fresh --out-dir so stale fuzz reports cannot be reused" >&2
        exit 2
      fi
    fi
    echo "refusing to use symlink out-dir: $out_dir" >&2
    echo "choose a real fresh --out-dir so fuzz reports cannot escape the selected directory" >&2
    exit 2
  fi

  if [[ -e "$find_out_dir" && ! -d "$find_out_dir" ]]; then
    echo "refusing to use non-directory out-dir: $out_dir" >&2
    echo "choose a fresh --out-dir directory for fuzz reports" >&2
    exit 2
  fi

  if [[ ! -d "$find_out_dir" ]]; then
    return 0
  fi
  local first_entry
  first_entry="$(find -H -- "$find_out_dir" -mindepth 1 -print -quit)"
  if [[ -n "$first_entry" ]]; then
    echo "refusing to reuse non-empty out-dir: $out_dir" >&2
    echo "choose a fresh --out-dir so stale fuzz reports cannot be reused" >&2
    exit 2
  fi
}

check_out_dir_fresh

logs_dir="$out_dir/logs"
summary="$out_dir/summary.md"
summary_json="$out_dir/summary.json"
crasher_inventory="$out_dir/crasher-inventory.json"
unstable_seeds="$out_dir/unstable-seeds.md"
tmp_dir="$(mktemp -d)"
started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
started_s="$(date +%s)"
step_count=0
failed_count=0
mkdir -p -- "$logs_dir"
: > "$tmp_dir/steps.jsonl"

cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

format_command() {
  local formatted=""
  local quoted=""
  local arg
  for arg in "$@"; do
    printf -v quoted "%q" "$arg"
    if [[ -z "$formatted" ]]; then
      formatted="$quoted"
    else
      formatted+=" $quoted"
    fi
  done
  printf "%s" "$formatted"
}

json_escape() {
  local s="${1-}"
  local LC_ALL=C
  local out=""
  local c
  local escaped
  local i
  for ((i = 0; i < ${#s}; i++)); do
    c="${s:i:1}"
    case "$c" in
      $'\\') out+="\\\\" ;;
      $'"') out+="\\\"" ;;
      $'\b') out+="\\b" ;;
      $'\f') out+="\\f" ;;
      $'\n') out+="\\n" ;;
      $'\r') out+="\\r" ;;
      $'\t') out+="\\t" ;;
      *)
        if [[ "$c" < $' ' ]]; then
          printf -v escaped "\\u%04x" "'$c"
          out+="$escaped"
        else
          out+="$c"
        fi
        ;;
    esac
  done
  printf "%s" "$out"
}

record_step_json() {
  local name="$1"
  local status="$2"
  local seconds="$3"
  local exit_code="$4"
  local command="$5"
  local log_rel="$6"

  printf '{"name":"%s","status":"%s","duration_seconds":%s,"exit_code":%s,"command":"%s","log":"%s"}\n' \
    "$(json_escape "$name")" \
    "$(json_escape "$status")" \
    "$seconds" \
    "$exit_code" \
    "$(json_escape "$command")" \
    "$(json_escape "$log_rel")" >> "$tmp_dir/steps.jsonl"
}

fuzz_inventory_roots=(
  "cli/cmd/tetra/testdata/fuzz"
  "compiler/internal/frontend/testdata/fuzz"
  "compiler/internal/httprt/testdata/fuzz"
  "compiler/internal/jsonrt/testdata/fuzz"
  "compiler/internal/linker/linkcore/testdata/fuzz"
  "compiler/internal/pgrt/testdata/fuzz"
  "compiler/tests/fuzz/testdata/fuzz"
  "tools/cmd/validate-manifest/testdata/fuzz"
)

count_fuzz_file() {
  local file="$1"
  local base="${file##*/}"

  if [[ "$base" == .* || "$base" == *~ ]]; then
    return 1
  fi
  return 0
}

write_crasher_inventory_json() {
  local roots_count=0
  local existing_roots_count=0
  local targets_count=0
  local corpus_files_count=0
  local crasher_files_count=0
  local total_files_count=0
  local root

  : > "$tmp_dir/inventory-roots.jsonl"

  for root in "${fuzz_inventory_roots[@]}"; do
    roots_count=$((roots_count + 1))
    local root_exists=false
    local root_targets=0
    local root_corpus_files=0
    local root_crasher_files=0
    local root_total_files=0
    local target

    if [[ -d "$root" ]]; then
      root_exists=true
      existing_roots_count=$((existing_roots_count + 1))
      while IFS= read -r target; do
        root_targets=$((root_targets + 1))
        targets_count=$((targets_count + 1))
        local file
        while IFS= read -r file; do
          if ! count_fuzz_file "$file"; then
            continue
          fi
          root_total_files=$((root_total_files + 1))
          total_files_count=$((total_files_count + 1))
          if [[ "$file" == */crashers/* || "${file##*/}" == crasher* ]]; then
            root_crasher_files=$((root_crasher_files + 1))
            crasher_files_count=$((crasher_files_count + 1))
          else
            root_corpus_files=$((root_corpus_files + 1))
            corpus_files_count=$((corpus_files_count + 1))
          fi
        done < <(find -H "$target" -type f -print | LC_ALL=C sort)
      done < <(find -H "$root" -mindepth 1 -maxdepth 1 -type d -print | LC_ALL=C sort)
    fi

    printf '{"root":"%s","exists":%s,"targets":%s,"corpus_files":%s,"crasher_files":%s,"total_files":%s}\n' \
      "$(json_escape "$root")" \
      "$root_exists" \
      "$root_targets" \
      "$root_corpus_files" \
      "$root_crasher_files" \
      "$root_total_files" >> "$tmp_dir/inventory-roots.jsonl"
  done

  {
    echo "{"
    echo '  "schema_version": 1,'
    echo '  "kind": "go-testdata-fuzz-inventory",'
    echo '  "scanned_roots": ['
    awk 'NR > 1 { printf ",\n" } { printf "    %s", $0 } END { if (NR > 0) printf "\n" }' "$tmp_dir/inventory-roots.jsonl"
    echo '  ],'
    echo '  "counts": {'
    printf '    "roots": %s,\n' "$roots_count"
    printf '    "existing_roots": %s,\n' "$existing_roots_count"
    printf '    "targets": %s,\n' "$targets_count"
    printf '    "corpus_files": %s,\n' "$corpus_files_count"
    printf '    "crasher_files": %s,\n' "$crasher_files_count"
    printf '    "total_files": %s\n' "$total_files_count"
    echo '  }'
    echo "}"
  } > "$crasher_inventory"
}

write_summary_json() {
  local status="$1"
  local exit_code="$2"
  local ended_at
  local ended_s
  ended_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  ended_s="$(date +%s)"

  {
    echo "{"
    printf '  "mode": "%s",\n' "$([[ "$short" == true ]] && echo short || echo nightly)"
    printf '  "status": "%s",\n' "$(json_escape "$status")"
    printf '  "exit_code": %s,\n' "$exit_code"
    printf '  "duration_seconds": %s,\n' "$((ended_s - started_s))"
    printf '  "started_at": "%s",\n' "$(json_escape "$started_at")"
    printf '  "ended_at": "%s",\n' "$(json_escape "$ended_at")"
    printf '  "fuzztime": "%s",\n' "$(json_escape "$fuzztime")"
    printf '  "step_count": %s,\n' "$step_count"
    printf '  "failed_count": %s,\n' "$failed_count"
    echo '  "artifacts": {'
    printf '    "summary_md": "%s",\n' "$(json_escape "$summary")"
    printf '    "summary_json": "%s",\n' "$(json_escape "$summary_json")"
    printf '    "crasher_inventory_json": "%s",\n' "$(json_escape "$crasher_inventory")"
    printf '    "logs_dir": "%s",\n' "$(json_escape "$logs_dir")"
    printf '    "unstable_seed_log": "%s",\n' "$(json_escape "$unstable_seeds")"
    printf '    "crasher_archive_path": "%s"\n' "$(json_escape "<package>/testdata/fuzz/<FuzzName>/")"
    echo '  },'
    echo '  "steps": ['
    awk 'NR > 1 { printf ",\n" } { printf "    %s", $0 } END { if (NR > 0) printf "\n" }' "$tmp_dir/steps.jsonl"
    echo '  ]'
    echo "}"
  } > "$summary_json"
}

{
  echo "# Fuzz Nightly Summary"
  echo
  echo "- mode: \`$([[ "$short" == true ]] && echo short || echo nightly)\`"
  echo "- fuzztime: \`$fuzztime\`"
  echo "- output_dir: \`$out_dir\`"
  echo "- crasher_archive_path: \`<package>/testdata/fuzz/<FuzzName>/\`"
  echo "- crasher_inventory_json: \`$crasher_inventory\`"
  echo "- unstable_seed_log: \`$unstable_seeds\`"
  echo
  echo "## Steps"
} > "$summary"

{
  echo "# Unstable Fuzz Seeds"
  echo
  echo "Record any flaky, timeout-sensitive, or non-deterministic fuzz seed observed"
  echo "during this run. Before release promotion, every listed seed must either be"
  echo "converted into a deterministic regression test or have an explicit owner and"
  echo "next command."
  echo
  echo "| package | fuzz target | seed/crasher path | status | owner | next command |"
  echo "| --- | --- | --- | --- | --- | --- |"
} > "$unstable_seeds"

failed=0
run_step() {
  local name="$1"
  shift
  step_count=$((step_count + 1))
  local command
  command="$(format_command "$@")"
  local log
  local log_rel
  log="$logs_dir/$name.log"
  log_rel="logs/$name.log"
  local start_s
  local end_s
  start_s="$(date +%s)"
  if "$@" > "$log" 2>&1; then
    end_s="$(date +%s)"
    record_step_json "$name" "pass" "$((end_s - start_s))" 0 "$command" "$log_rel"
    printf -- '- `%s`: pass, command `%s`, log `%s`\n' "$name" "$command" "$log" >> "$summary"
  else
    local code="$?"
    end_s="$(date +%s)"
    failed=$((failed + 1))
    failed_count=$((failed_count + 1))
    record_step_json "$name" "fail" "$((end_s - start_s))" "$code" "$command" "$log_rel"
    printf -- '- `%s`: fail exit `%s`, command `%s`, log `%s`\n' "$name" "$code" "$command" "$log" >> "$summary"
    tail -n 80 "$log" >&2 || true
  fi
}

run_step "compiler-frontend-lexer" go test ./compiler/internal/frontend -run '^$' -fuzz=FuzzLexer "-fuzztime=$fuzztime" "${fuzz_parallel_args[@]}"
run_step "compiler-frontend-parser" go test ./compiler/internal/frontend -run '^$' -fuzz=FuzzParser "-fuzztime=$fuzztime" "${fuzz_parallel_args[@]}"
run_step "compiler-format" go test ./compiler/tests/fuzz -run '^$' -fuzz=FuzzFormatSourceIdempotent "-fuzztime=$fuzztime" "${fuzz_parallel_args[@]}"
run_step "compiler-lowering" go test ./compiler/tests/fuzz -run '^$' -fuzz=FuzzLoweringPipelineVerifiesIR "-fuzztime=$fuzztime" "${fuzz_parallel_args[@]}"
run_step "compiler-linker-linkcore" go test ./compiler/internal/linker/linkcore -run '^$' -fuzz=FuzzLinkX64ObjectsDoesNotPanic "-fuzztime=$fuzztime" "${fuzz_parallel_args[@]}"
run_step "http-runtime" go test ./compiler/internal/httprt -run '^$' -fuzz=FuzzHTTPParseRequest "-fuzztime=$fuzztime" "${fuzz_parallel_args[@]}"
run_step "json-runtime" go test ./compiler/internal/jsonrt -run '^$' -fuzz=FuzzAppendStringProducesValidJSON "-fuzztime=$fuzztime" "${fuzz_parallel_args[@]}"
run_step "postgres-wire" go test ./compiler/internal/pgrt -run '^$' -fuzz=FuzzReadFrameDoesNotPanic "-fuzztime=$fuzztime" "${fuzz_parallel_args[@]}"
run_step "validate-manifest" go test ./tools/cmd/validate-manifest -run '^$' -fuzz=. "-fuzztime=$fuzztime" "${fuzz_parallel_args[@]}"
run_step "eco-capsule" go test ./cli/cmd/tetra -run '^$' -fuzz=FuzzParseCapsuleDoesNotPanic "-fuzztime=$fuzztime" "${fuzz_parallel_args[@]}"
run_step "property-stress-regressions" go test ./compiler/... ./cli/... ./tools/cmd/validate-manifest -run 'Fuzz|Property|Stress' -count=1

if [[ "$failed" -ne 0 ]]; then
  write_crasher_inventory_json
  write_summary_json "fail" 1
  echo "fuzz nightly failed: $failed step(s); see $summary" >&2
  exit 1
fi

write_crasher_inventory_json
write_summary_json "pass" 0
echo "fuzz nightly passed: $summary"
