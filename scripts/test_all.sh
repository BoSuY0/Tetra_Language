#!/usr/bin/env bash
set -euo pipefail

mode="full"
report_dir=""
keep_going=false
json_only=false

usage() {
  cat <<'USAGE'
Usage: bash scripts/test_all.sh [--quick|--full] [--keep-going] [--json-only] [--report-dir DIR]

Modes:
  --quick  Run the fast stabilization gate for local iteration.
  --full   Run the full v0.6.x stabilization gate with logs and summaries.

Output:
  --keep-going  Run remaining steps after a failure and exit 1 at the end.
  --json-only   Suppress progress logs and print summary JSON to stdout.

Artifacts:
  The script writes per-step logs plus summary.md and summary.json to DIR.
  summary.json records each step name, status, duration_seconds, exit_code, and log.
  It also includes top-level step_count and failed_count fields.
  If DIR is omitted, reports/test-all-<UTC timestamp> is used.

Exit codes:
  0  All selected checks passed.
  1  One or more checks failed.
  2  Usage/configuration error.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --quick)
      mode="quick"
      shift
      ;;
    --full)
      mode="full"
      shift
      ;;
    --report-dir)
      if [[ $# -lt 2 ]]; then
        echo "--report-dir requires a directory" >&2
        exit 2
      fi
      report_dir="$2"
      shift 2
      ;;
    --keep-going)
      keep_going=true
      shift
      ;;
    --json-only)
      json_only=true
      shift
      ;;
    -h|--help)
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

timestamp="$(date -u +%Y%m%d-%H%M%S)"
if [[ -z "$report_dir" ]]; then
  report_dir="reports/test-all-$timestamp"
fi

logs_dir="$report_dir/logs"
summary_md="$report_dir/summary.md"
summary_json="$report_dir/summary.json"
tmp_dir="$(mktemp -d)"
started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
step_count=0
failed_count=0

mkdir -p "$logs_dir"
: >"$tmp_dir/steps.md"
: >"$tmp_dir/steps.jsonl"

cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

json_escape() {
  local s="${1-}"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  printf '%s' "$s"
}

write_summary() {
  local status="$1"
  local ended_at
  ended_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  {
    echo "# Tetra v0.6.x Test Report"
    echo
    echo "- mode: \`$mode\`"
    echo "- status: \`$status\`"
    echo "- started_at: \`$started_at\`"
    echo "- ended_at: \`$ended_at\`"
    echo "- step_count: \`$step_count\`"
    echo "- failed_count: \`$failed_count\`"
    echo
    echo "## Steps"
    echo
    cat "$tmp_dir/steps.md"
  } >"$summary_md"

  {
    echo "{"
    printf '  "mode": "%s",\n' "$(json_escape "$mode")"
    printf '  "status": "%s",\n' "$(json_escape "$status")"
    printf '  "started_at": "%s",\n' "$(json_escape "$started_at")"
    printf '  "ended_at": "%s",\n' "$(json_escape "$ended_at")"
    printf '  "step_count": %s,\n' "$step_count"
    printf '  "failed_count": %s,\n' "$failed_count"
    echo '  "steps": ['
    awk 'NR > 1 { printf ",\n" } { printf "    %s", $0 } END { if (NR > 0) printf "\n" }' "$tmp_dir/steps.jsonl"
    echo '  ]'
    echo "}"
  } >"$summary_json"
}

validate_summary() {
  go run ./tools/cmd/validate-test-all-summary --summary "$summary_json" --report-dir "$report_dir"
}

validate_summary_best_effort() {
  if ! validate_summary; then
    printf 'warning: summary validation failed; preserving original test failure\n' >&2
  fi
}

record_step() {
  local name="$1"
  local status="$2"
  local seconds="$3"
  local exit_code="$4"
  local log_rel="$5"
  local command="$6"

  printf -- '- `%s`: `%s` in %ss, exit `%s`, command `%s` ([%s](%s))\n' "$name" "$status" "$seconds" "$exit_code" "$command" "$log_rel" "$log_rel" >>"$tmp_dir/steps.md"
  printf '{"name":"%s","status":"%s","duration_seconds":%s,"exit_code":%s,"command":"%s","log":"%s"}\n' \
    "$(json_escape "$name")" \
    "$(json_escape "$status")" \
    "$seconds" \
    "$exit_code" \
    "$(json_escape "$command")" \
    "$(json_escape "$log_rel")" >>"$tmp_dir/steps.jsonl"
}

slugify() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//'
}

run_step() {
  local name="$1"
  shift
  step_count=$((step_count + 1))

  local step_id
  local slug
  local log_rel
  local log_path
  local command
  step_id="$(printf '%02d' "$step_count")"
  slug="$(slugify "$name")"
  log_rel="logs/${step_id}-${slug}.log"
  log_path="$report_dir/$log_rel"
  command="$*"

  if [[ "$json_only" != true ]]; then
    printf '== [%s] %s ==\n' "$step_id" "$name"
  fi
  local start_s
  local end_s
  start_s="$(date +%s)"

  if "$@" >"$log_path" 2>&1; then
    end_s="$(date +%s)"
    record_step "$name" "pass" "$((end_s - start_s))" 0 "$log_rel" "$command"
    if [[ "$json_only" != true ]]; then
      printf '   pass (%ss)\n' "$((end_s - start_s))"
    fi
  else
    local exit_code="$?"
    end_s="$(date +%s)"
    record_step "$name" "fail" "$((end_s - start_s))" "$exit_code" "$log_rel" "$command"
    failed_count=$((failed_count + 1))
    if [[ "$json_only" != true ]]; then
      printf '   fail (%ss). Last log lines:\n' "$((end_s - start_s))" >&2
      tail -n 80 "$log_path" >&2 || true
    fi
    if [[ "$keep_going" != true ]]; then
      write_summary "fail"
      validate_summary_best_effort
      if [[ "$json_only" == true ]]; then
        cat "$summary_json"
      else
        printf '\nFull report: %s\n' "$summary_md" >&2
      fi
      exit 1
    fi
  fi
}

check_version_prefix() {
  local version
  version="$(./tetra version)"
  case "$version" in
    v0.6.*)
      echo "$version"
      ;;
    *)
      echo "expected v0.6.x, got $version" >&2
      exit 1
      ;;
  esac
}

check_short_alias_version() {
  local version
  local short_version
  version="$(./tetra version)"
  short_version="$(./t version)"
  if [[ "$short_version" != "$version" ]]; then
    echo "expected ./t version to match ./tetra version ($version), got $short_version" >&2
    return 1
  fi
  echo "$short_version"
}

check_test_json() {
  ./tetra test --report=json examples >"$report_dir/tetra-test-report.json"
  test -s "$report_dir/tetra-test-report.json"
  go run ./tools/cmd/validate-test-report --report "$report_dir/tetra-test-report.json"
}

check_tetra_doc() {
  ./tetra doc examples >"$report_dir/tetra-docs.md"
  go run ./tools/cmd/validate-api-docs --docs "$report_dir/tetra-docs.md"
}

check_json_diagnostic_case() {
  local name="$1"
  local contains="$2"
  local source="$tmp_dir/$name.tetra"
  local stdout="$tmp_dir/$name.out"
  local diagnostic="$report_dir/$name.json"
  shift 2
  cat >"$source"
  if ./tetra check --diagnostics=json "$source" >"$stdout" 2>"$diagnostic"; then
    echo "expected tetra check --diagnostics=json to fail for $name" >&2
    return 1
  fi
  test ! -s "$stdout"
  go run ./tools/cmd/validate-diagnostic --diagnostic "$diagnostic" --severity error --contains "$contains" --require-position
}

check_json_diagnostic() {
  check_json_diagnostic_case "invalid-diagnostic" "unknown function" <<'TETRA'
func main() -> Int:
    return missing_call()
TETRA
  check_json_diagnostic_case "missing-effect-diagnostic" "uses effect 'io'" <<'TETRA'
func main() -> Int:
    print("missing uses\n")
    return 0
TETRA
  check_json_diagnostic_case "tabs-diagnostic" "tabs are not supported" <<'TETRA'
func main() -> Int:
	return 0
TETRA
  check_json_diagnostic_case "planned-actor-diagnostic" "planned feature 'actor'" <<'TETRA'
actor Worker:
    return 0
TETRA

  if ./tetra build --diagnostics=json --target wasm32-wasi examples/flow_hello.tetra >"$tmp_dir/wasm-target-diagnostic.out" 2>"$report_dir/wasm-target-diagnostic.json"; then
    echo "expected tetra build --target wasm32-wasi to report planned target diagnostic" >&2
    return 1
  fi
  test ! -s "$tmp_dir/wasm-target-diagnostic.out"
  go run ./tools/cmd/validate-diagnostic --diagnostic "$report_dir/wasm-target-diagnostic.json" --severity error --contains "planned target not implemented: wasm32-wasi"
}

check_targets_report() {
  ./tetra targets --format=json >"$report_dir/targets.json"
  go run ./tools/cmd/validate-targets --report "$report_dir/targets.json"
}

check_doctor_report() {
  ./tetra doctor --format=json >"$report_dir/doctor.json"
  go run ./tools/cmd/validate-doctor --report "$report_dir/doctor.json"
}

check_docs_manifest() {
  go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
  go run ./tools/cmd/gen-manifest -o "$tmp_dir/manifest.json"
  go run ./tools/cmd/validate-manifest --manifest "$tmp_dir/manifest.json"
  diff -u docs/generated/manifest.json "$tmp_dir/manifest.json"
}

check_lsp_stdio() {
  local lsp_init
  local lsp_open
  local lsp_shutdown
  local lsp_exit
  lsp_init='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
  lsp_open='{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"func main() -> Int:\n    return 0\n"}}}'
  lsp_shutdown='{"jsonrpc":"2.0","id":2,"method":"shutdown","params":{}}'
  lsp_exit='{"jsonrpc":"2.0","method":"exit","params":{}}'

  {
    for body in "$lsp_init" "$lsp_open" "$lsp_shutdown" "$lsp_exit"; do
      printf 'Content-Length: %s\r\n\r\n%s' "$(printf '%s' "$body" | wc -c)" "$body"
    done
  } | ./tetra lsp --stdio >"$tmp_dir/lsp-stdio.out"

  go run ./tools/cmd/validate-lsp-stdio --transcript "$tmp_dir/lsp-stdio.out"
  grep -q '"capabilities"' "$tmp_dir/lsp-stdio.out"
  grep -q '"textDocument/publishDiagnostics"' "$tmp_dir/lsp-stdio.out"
}

check_lsp_smoke() {
  ./tetra lsp --stdio-smoke examples/flow_hello.tetra >"$report_dir/lsp-smoke.json"
  go run ./tools/cmd/validate-lsp-smoke --report "$report_dir/lsp-smoke.json"
}

check_host_smoke() {
  ./tetra smoke --target linux-x64 --run=true --report "$report_dir/host-smoke.json"
  go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$report_dir/host-smoke.json"
}

check_smoke_list() {
  ./tetra smoke --list --format=json >"$report_dir/smoke-list.json"
  go run ./tools/cmd/validate-smoke-list --report "$report_dir/smoke-list.json"
}

check_generated_api_docs() {
  go run ./tools/cmd/gen-docs examples >"$report_dir/api-docs.md"
  go run ./tools/cmd/validate-api-docs --docs "$report_dir/api-docs.md"
}

check_eco_suite() {
  mkdir -p "$tmp_dir/project/src"
  cat >"$tmp_dir/project/Tetra.capsule" <<'CAPSULE'
capsule App:
  id "tetra://app"
  version "0.1.0"
  target "linux-x64"
  dependency "tetra://core" "0.1.0"
CAPSULE
  cat >"$tmp_dir/Core.capsule" <<'CAPSULE'
capsule Core:
  id "tetra://core"
  version "0.1.0"
  target "linux-x64"
CAPSULE
  cat >"$tmp_dir/project/src/main.tetra" <<'TETRA'
func main() -> Int:
    return 0
TETRA

  ./tetra eco verify --target linux-x64 --lock "$tmp_dir/tetra.lock.json" "$tmp_dir/project/Tetra.capsule" "$tmp_dir/Core.capsule"
  go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/tetra.lock.json"
  ./tetra eco pack "$tmp_dir/project/Tetra.capsule" -o "$tmp_dir/single.todex"
  ./tetra eco pack --project "$tmp_dir/project/Tetra.capsule" -o "$tmp_dir/project.todex"
  ./tetra eco unpack "$tmp_dir/project.todex" -C "$tmp_dir/unpacked"
  go run ./tools/cmd/validate-eco-unpack --dir "$tmp_dir/unpacked"
  test -f "$tmp_dir/unpacked/src/main.tetra"
  ./tetra eco vault add --store "$tmp_dir/vault" --kind source examples/flow_hello.tetra
  ./tetra eco vault list --store "$tmp_dir/vault"
  ./tetra eco vault verify --store "$tmp_dir/vault"
  go run ./tools/cmd/validate-eco-vault --store "$tmp_dir/vault"
}

check_cross_target_smoke() {
  ./tetra smoke --target linux-x64 --run=false --report "$tmp_dir/linux-smoke.json"
  go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/linux-smoke.json"
  ./tetra smoke --target macos-x64 --run=false --report "$tmp_dir/macos-smoke.json"
  go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/macos-smoke.json"
  ./tetra smoke --target windows-x64 --run=false --report "$tmp_dir/windows-smoke.json"
  go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/windows-smoke.json"
}

if [[ "$json_only" != true ]]; then
  printf 'Tetra v0.6.x test wrapper\n'
  printf 'mode: %s\n' "$mode"
  printf 'report_dir: %s\n\n' "$report_dir"
fi

run_step "go test all packages" go test ./compiler/... ./cli/... ./tools/...

if [[ "$mode" == "full" ]]; then
  run_step "repo test script" bash scripts/test.sh
fi

run_step "bootstrap" bash scripts/bootstrap.sh
run_step "version prefix" check_version_prefix
run_step "short alias version" check_short_alias_version
run_step "formatter check examples lib runtime" ./tetra fmt --check examples lib __rt compiler/selfhostrt
run_step "flow-only source scan" go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
run_step "targets json report" check_targets_report
run_step "doctor json report" check_doctor_report
run_step "tetra check flow hello" ./tetra check examples/flow_hello.tetra
run_step "json diagnostic shape" check_json_diagnostic
run_step "smoke list json report" check_smoke_list
run_step "tetra test examples" ./tetra test examples

if [[ "$mode" == "full" ]]; then
  run_step "tetra test json report" check_test_json
fi

run_step "host smoke linux-x64" check_host_smoke

if [[ "$mode" == "full" ]]; then
  run_step "docs manifest diff" check_docs_manifest
  run_step "docs verification" go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
  run_step "lsp stdio smoke" check_lsp_smoke
  run_step "lsp json-rpc stdio" check_lsp_stdio
  run_step "tetra doc examples" check_tetra_doc
  run_step "generated api docs" check_generated_api_docs
  run_step "eco graph bundle vault" check_eco_suite
  run_step "cross-target build smoke" check_cross_target_smoke
fi

if [[ "$failed_count" -gt 0 ]]; then
  write_summary "fail"
  validate_summary_best_effort
  if [[ "$json_only" == true ]]; then
    cat "$summary_json"
  else
    printf '\n%s %s check(s) failed.\n' "$failed_count" "$mode" >&2
    printf 'Summary: %s\n' "$summary_md" >&2
    printf 'JSON: %s\n' "$summary_json" >&2
  fi
  exit 1
fi

write_summary "pass"
validate_summary
if [[ "$json_only" == true ]]; then
  cat "$summary_json"
else
  printf '\nAll %s checks passed.\n' "$mode"
  printf 'Summary: %s\n' "$summary_md"
  printf 'JSON: %s\n' "$summary_json"
fi
