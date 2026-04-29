#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
cd "$repo_root"

mode="full"
report_dir=""
keep_going=false
json_only=false
release_version="${TETRA_TEST_ALL_RELEASE_VERSION:-v0.2.0}"
release_artifact="tetra.release.v0_2_0.test-all-summary.v1"

usage() {
  cat <<'USAGE'
Usage: bash scripts/test_all.sh [--quick|--full] [--keep-going] [--json-only] [--report-dir DIR]

Modes:
  --quick  Run the fast stabilization gate for local iteration.
  --full   Run the full v0.2.0 stabilization gate with logs and summaries.

Output:
  --keep-going  Run remaining steps after a failure and exit 1 at the end.
  --json-only   Suppress progress logs and print summary JSON to stdout.

Artifacts:
  The script writes per-step logs plus summary.md and summary.json to DIR.
  summary.json records each step name, status, duration_seconds, exit_code, and log.
  It also includes top-level step_count and failed_count fields.
  release_artifact is fixed to tetra.release.v0_2_0.test-all-summary.v1.
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
    echo "# Tetra $release_version Test Report"
    echo
    echo "- mode: \`$mode\`"
    echo "- status: \`$status\`"
    echo "- started_at: \`$started_at\`"
    echo "- ended_at: \`$ended_at\`"
    echo "- step_count: \`$step_count\`"
    echo "- failed_count: \`$failed_count\`"
    echo "- release_artifact: \`$release_artifact\`"
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
    printf '  "release_artifact": "%s",\n' "$(json_escape "$release_artifact")"
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

check_release_version() {
  local version
  version="$(./tetra version)"
  if [[ "$version" != "$release_version" ]]; then
    echo "expected $release_version, got $version" >&2
    exit 1
  fi
  echo "$version"
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
  check_json_diagnostic_case "planned-actor-diagnostic" "actor declarations currently support state fields and func methods only" <<'TETRA'
actor Worker:
    return 0
TETRA

  local wasm_out="$tmp_dir/hello.wasm"
  ./tetra build --target wasm32-wasi -o "$wasm_out" examples/hello.tetra >"$tmp_dir/wasm-target-build.out" 2>"$report_dir/wasm-target-build.err"
  test -s "$wasm_out"
  test "$(od -An -tx1 -N4 "$wasm_out" | tr -d ' \n')" = "0061736d"
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

check_performance_report() {
  local report="docs/generated/v1_0/performance-regression.json"
  if [[ ! -f "$report" ]]; then
    echo "performance report not found at $report; skipping in compatibility mode"
    return 0
  fi
  go run ./tools/cmd/validate-performance-report --report "$report"
}

check_lsp_stdio() {
  local lsp_init
  local lsp_open
  local lsp_symbols
  local lsp_hover
  local lsp_completion
  local lsp_definition
  local lsp_references
  local lsp_rename
  local lsp_formatting
  local lsp_change
  local lsp_code_action
  local lsp_shutdown
  local lsp_exit
  lsp_init='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
  lsp_open='{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///sample.tetra","languageId":"tetra","version":1,"text":"const answer: Int = 42\n\nfunc main() -> Int:\n  return answer\n"}}}'
  lsp_symbols='{"jsonrpc":"2.0","id":2,"method":"textDocument/documentSymbol","params":{"textDocument":{"uri":"file:///sample.tetra"}}}'
  lsp_hover='{"jsonrpc":"2.0","id":3,"method":"textDocument/hover","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":0,"character":6}}}'
  lsp_completion='{"jsonrpc":"2.0","id":4,"method":"textDocument/completion","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":9}}}'
  lsp_definition='{"jsonrpc":"2.0","id":5,"method":"textDocument/definition","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":9}}}'
  lsp_references='{"jsonrpc":"2.0","id":6,"method":"textDocument/references","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":9},"context":{"includeDeclaration":true}}}'
  lsp_rename='{"jsonrpc":"2.0","id":7,"method":"textDocument/rename","params":{"textDocument":{"uri":"file:///sample.tetra"},"position":{"line":3,"character":9},"newName":"value"}}'
  lsp_formatting='{"jsonrpc":"2.0","id":8,"method":"textDocument/formatting","params":{"textDocument":{"uri":"file:///sample.tetra"},"options":{"tabSize":4,"insertSpaces":true}}}'
  lsp_change='{"jsonrpc":"2.0","method":"textDocument/didChange","params":{"textDocument":{"uri":"file:///sample.tetra","version":2},"contentChanges":[{"text":"const answer: Int = 42\n\nfunc main() -> Int:\n    print(\"x\")\n    return answer\n"}]}}'
  lsp_code_action='{"jsonrpc":"2.0","id":9,"method":"textDocument/codeAction","params":{"textDocument":{"uri":"file:///sample.tetra"},"range":{"start":{"line":3,"character":4},"end":{"line":3,"character":9}},"context":{"diagnostics":[{"range":{"start":{"line":3,"character":4},"end":{"line":3,"character":9}},"severity":1,"code":"TETRA2001","source":"tetra","message":"function '\''main'\'' uses effect '\''io'\'' but does not declare it"}]}}}'
  lsp_shutdown='{"jsonrpc":"2.0","id":10,"method":"shutdown","params":{}}'
  lsp_exit='{"jsonrpc":"2.0","method":"exit","params":{}}'

  {
    for body in "$lsp_init" "$lsp_open" "$lsp_symbols" "$lsp_hover" "$lsp_completion" "$lsp_definition" "$lsp_references" "$lsp_rename" "$lsp_formatting" "$lsp_change" "$lsp_code_action" "$lsp_shutdown" "$lsp_exit"; do
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

release_smoke_cases=(
  "flow_hello:examples/flow_hello.tetra:0"
  "flow_struct_smoke:examples/flow_struct_smoke.tetra:42"
  "flow_islands_smoke:examples/flow_islands_smoke.tetra:0"
  "flow_unsafe_cap_mem_smoke:examples/flow_unsafe_cap_mem_smoke.tetra:42"
)

write_release_smoke_list() {
  local out="$1"
  local first=true
  local entry
  local name
  local src
  local expected

  {
    echo "{"
    printf '  "total": %s,\n' "${#release_smoke_cases[@]}"
    echo '  "islands_debug": false,'
    echo '  "cases": ['
    for entry in "${release_smoke_cases[@]}"; do
      IFS=':' read -r name src expected <<<"$entry"
      if [[ "$first" == true ]]; then
        first=false
      else
        echo ","
      fi
      printf '    {"name":"%s","src_path":"%s","expected_exit":%s}' "$(json_escape "$name")" "$(json_escape "$src")" "$expected"
    done
    echo
    echo "  ]"
    echo "}"
  } >"$out"
}

run_release_smoke_target() {
  local target="$1"
  local run_binaries="$2"
  local report_path="$3"
  local cases_jsonl="$tmp_dir/release-smoke-${target}.jsonl"
  local timestamp
  local version
  local git_head
  local host
  local total=0
  local passed=0
  local failed=0
  local entry
  local name
  local src
  local expected

  timestamp="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  version="$(./tetra version)"
  git_head="$(git rev-parse --short HEAD 2>/dev/null || true)"
  host="$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)"

  : >"$cases_jsonl"

  for entry in "${release_smoke_cases[@]}"; do
    local case_json
    local out_path
    local actual_exit
    IFS=':' read -r name src expected <<<"$entry"
    total=$((total + 1))
    case_json="{\"name\":\"$(json_escape "$name")\",\"src_path\":\"$(json_escape "$src")\",\"expected_exit\":$expected"

    if [[ "$run_binaries" == "true" ]]; then
      if ./tetra run --target "$target" "$src" >"$tmp_dir/${name}-${target}.run.out" 2>"$tmp_dir/${name}-${target}.run.err"; then
        actual_exit=0
      else
        actual_exit="$?"
      fi
      case_json="$case_json,\"actual_exit\":$actual_exit,\"ran\":true"
      if [[ "$actual_exit" -eq "$expected" ]]; then
        passed=$((passed + 1))
        case_json="$case_json,\"pass\":true"
      else
        failed=$((failed + 1))
        case_json="$case_json,\"pass\":false,\"error\":\"unexpected exit: got $actual_exit, want $expected\""
      fi
    else
      local build_error
      out_path="$tmp_dir/${name}-${target}"
      case_json="$case_json,\"out_path\":\"$(json_escape "$out_path")\""
      if ./tetra build --target "$target" -o "$out_path" "$src" >"$tmp_dir/${name}-${target}.build.out" 2>"$tmp_dir/${name}-${target}.build.err"; then
        passed=$((passed + 1))
        case_json="$case_json,\"ran\":false,\"pass\":true"
      else
        build_error="$(head -n 1 "$tmp_dir/${name}-${target}.build.err" || true)"
        failed=$((failed + 1))
        case_json="$case_json,\"ran\":false,\"pass\":false,\"error\":\"$(json_escape "${build_error:-build failed}")\""
      fi
    fi

    case_json="$case_json}"
    printf '%s\n' "$case_json" >>"$cases_jsonl"
  done

  {
    echo "{"
    printf '  "timestamp": "%s",\n' "$(json_escape "$timestamp")"
    printf '  "target": "%s",\n' "$(json_escape "$target")"
    printf '  "host": "%s",\n' "$(json_escape "$host")"
    printf '  "version": "%s",\n' "$(json_escape "$version")"
    printf '  "git_head": "%s",\n' "$(json_escape "$git_head")"
    echo '  "islands_debug": false,'
    printf '  "total": %s,\n' "$total"
    printf '  "passed": %s,\n' "$passed"
    printf '  "failed": %s,\n' "$failed"
    echo '  "cases": ['
    awk 'NR > 1 { printf ",\n" } { printf "    %s", $0 } END { if (NR > 0) printf "\n" }' "$cases_jsonl"
    echo '  ]'
    echo "}"
  } >"$report_path"

  [[ "$failed" -eq 0 ]]
}

check_host_smoke() {
  run_release_smoke_target "linux-x64" "true" "$report_dir/host-smoke.json"
  go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$report_dir/host-smoke.json"
}

check_smoke_list() {
  write_release_smoke_list "$report_dir/smoke-list.json"
  go run ./tools/cmd/validate-flow-only examples/flow_hello.tetra examples/flow_struct_smoke.tetra examples/flow_islands_smoke.tetra examples/flow_unsafe_cap_mem_smoke.tetra
}

check_generated_api_docs() {
  go run ./tools/cmd/gen-docs examples >"$report_dir/api-docs.md"
  go run ./tools/cmd/validate-api-docs --docs "$report_dir/api-docs.md"
}

check_eco_suite() {
  mkdir -p "$tmp_dir/project/src"
  cat >"$tmp_dir/project/Tetra.capsule" <<'CAPSULE'
manifest "tetra.capsule.v1"
capsule App:
  id "tetra://app"
  version "0.1.0"
  target "linux-x64"
  permission "io"
  dependency "tetra://core" "0.1.0"
CAPSULE
  cat >"$tmp_dir/Core.capsule" <<'CAPSULE'
manifest "tetra.capsule.v1"
capsule Core:
  id "tetra://core"
  version "0.1.0"
  target "linux-x64"
  permission "io"
CAPSULE
  cat >"$tmp_dir/project/src/main.tetra" <<'TETRA'
func main() -> Int:
    return 0
TETRA

  ./tetra eco verify --target linux-x64 --lock "$tmp_dir/tetra.lock.json" "$tmp_dir/project/Tetra.capsule" "$tmp_dir/Core.capsule"
  go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/tetra.lock.json"
  ./tetra eco seed export --out "$tmp_dir/tetra.seed.json" "$tmp_dir/project/Tetra.capsule" "$tmp_dir/Core.capsule"
  ./tetra eco seed import --seed "$tmp_dir/tetra.seed.json" --lock "$tmp_dir/tetra.seed.lock.json" --capsules-dir "$tmp_dir/seed-capsules"
  go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/tetra.seed.lock.json"
  ./tetra eco needmap --lock "$tmp_dir/tetra.lock.json" -o "$tmp_dir/tetra.needmap.json"
  ./tetra eco pack "$tmp_dir/project/Tetra.capsule" -o "$tmp_dir/single.todex"
  ./tetra eco pack --project "$tmp_dir/project/Tetra.capsule" -o "$tmp_dir/project.todex"
  ./tetra eco unpack "$tmp_dir/project.todex" -C "$tmp_dir/unpacked"
  go run ./tools/cmd/validate-eco-unpack --dir "$tmp_dir/unpacked"
  test -f "$tmp_dir/unpacked/src/main.tetra"
  ./tetra eco vault add --store "$tmp_dir/vault" --kind source examples/flow_hello.tetra
  ./tetra eco vault list --store "$tmp_dir/vault"
  ./tetra eco vault verify --store "$tmp_dir/vault"
  go run ./tools/cmd/validate-eco-vault --store "$tmp_dir/vault"
  ./tetra eco trust snapshot --lock "$tmp_dir/tetra.lock.json" --store "$tmp_dir/vault" -o "$tmp_dir/tetra.trust-snapshot.json"
  ./tetra eco materialize "$tmp_dir/project.todex" --target linux-x64 --trust "$tmp_dir/tetra.trust-snapshot.json" -C "$tmp_dir/materialized"
  test -f "$tmp_dir/materialized/tetra.materialization.json"
  ./tetra eco publish --package "$tmp_dir/project.todex" --registry "$tmp_dir/registry" --target linux-x64 --trust "$tmp_dir/tetra.trust-snapshot.json"
  go run ./tools/cmd/validate-eco-publish --registry "$tmp_dir/registry" --id tetra://app --version 0.1.0 --target linux-x64
  ./tetra eco download --id tetra://app --version 0.1.0 --target linux-x64 --registry "$tmp_dir/registry" -o "$tmp_dir/downloaded.todex"
  test -f "$tmp_dir/downloaded.todex"
  ./tetra eco tetrahub publish --package "$tmp_dir/project.todex" --store "$tmp_dir/tetrahub-beta" --target linux-x64 --trust "$tmp_dir/tetra.trust-snapshot.json"
  ./tetra eco tetrahub download --id tetra://app --version 0.1.0 --target linux-x64 --store "$tmp_dir/tetrahub-beta" -o "$tmp_dir/hub-downloaded.todex"
  test -f "$tmp_dir/hub-downloaded.todex"
}

check_cross_target_smoke() {
  run_release_smoke_target "linux-x64" "false" "$tmp_dir/linux-smoke.json"
  go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/linux-smoke.json"
  run_release_smoke_target "macos-x64" "false" "$tmp_dir/macos-smoke.json"
  go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/macos-smoke.json"
  run_release_smoke_target "windows-x64" "false" "$tmp_dir/windows-smoke.json"
  go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/windows-smoke.json"
  ./tetra smoke --target wasm32-wasi --run=false --report "$tmp_dir/wasm32-wasi-smoke.json"
  go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/wasm32-wasi-smoke.json"
  ./tetra smoke --target wasm32-web --run=false --report "$tmp_dir/wasm32-web-smoke.json"
  go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/wasm32-web-smoke.json"
}

if [[ "$json_only" != true ]]; then
  printf 'Tetra %s test wrapper\n' "$release_version"
  printf 'mode: %s\n' "$mode"
  printf 'report_dir: %s\n\n' "$report_dir"
fi

run_step "go test all packages" go test ./compiler/... ./cli/... ./tools/...

if [[ "$mode" == "full" ]]; then
  run_step "repo test script" bash scripts/test.sh
fi

run_step "bootstrap" bash scripts/bootstrap.sh
run_step "version preflight" check_release_version
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
  run_step "performance report schema" check_performance_report
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
