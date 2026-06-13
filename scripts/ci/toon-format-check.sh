#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
cd "$repo_root"

export GOTELEMETRY="${GOTELEMETRY:-off}"
export GOCACHE="${GOCACHE:-${XDG_CACHE_HOME:-$HOME/.cache}/tetra-language/go-build-toon-format-check}"
export GOTMPDIR="${GOTMPDIR:-.cache/go-tmp-toon-format-check}"
mkdir -p "$GOTMPDIR"

tmp_parent="${TETRA_TOON_CHECK_TMPDIR:-.cache/toon-format-check}"
mkdir -p "$tmp_parent"
tmp_dir="$(mktemp -d "$tmp_parent/run.XXXXXX")"
trap 'rm -rf "$tmp_dir"' EXIT
tetra_bin=""
run_tetra() {
  if [[ -n "${TETRA_CMD:-}" ]]; then
    "$TETRA_CMD" "$@"
  else
    if [[ -z "$tetra_bin" ]]; then
      tetra_bin="$tmp_dir/tetra"
      go build -buildvcs=false -o "$tetra_bin" ./cli/cmd/tetra
    fi
    "$tetra_bin" "$@"
  fi
}

go_test() {
  go test -buildvcs=false "$@"
}

check_pair() {
  local command_name="$1"
  local validator="$2"
  shift 2

  local json_path="$tmp_dir/${command_name}.json"
  local toon_path="$tmp_dir/${command_name}.toon"

  run_tetra "$@" --format=json >"$json_path"
  run_tetra "$@" --format=toon >"$toon_path"

  go run "$validator" --report "$json_path"
  go run "$validator" --report "$toon_path"
}

check_pair targets ./tools/cmd/validate-targets targets
check_pair features ./tools/cmd/validate-features features
check_pair formats ./tools/cmd/validate-formats formats
check_pair doctor ./tools/cmd/validate-doctor doctor

diagnostic_json="$tmp_dir/diagnostic.json"
diagnostic_toon="$tmp_dir/diagnostic.toon"
if run_tetra run --diagnostics=json --target not-a-target >"$tmp_dir/diagnostic-json.stdout" 2>"$diagnostic_json"; then
  echo "expected JSON diagnostic command to fail" >&2
  exit 1
fi
if run_tetra run --diagnostics=toon --target not-a-target >"$tmp_dir/diagnostic-toon.stdout" 2>"$diagnostic_toon"; then
  echo "expected TOON diagnostic command to fail" >&2
  exit 1
fi
go run ./tools/cmd/validate-diagnostic --diagnostic "$diagnostic_json" --contains "unsupported target"
go run ./tools/cmd/validate-diagnostic --diagnostic "$diagnostic_toon" --contains "unsupported target"

test_src="$tmp_dir/toon_report.tetra"
cat >"$test_src" <<'TETRA'
test "toon report":
    expect 40 + 2 == 42
TETRA
run_tetra test --report=json "$test_src" >"$tmp_dir/test-report.json"
run_tetra test --report=toon "$test_src" >"$tmp_dir/test-report.toon"
go run ./tools/cmd/validate-test-report --report "$tmp_dir/test-report.json"
go run ./tools/cmd/validate-test-report --report "$tmp_dir/test-report.toon"

run_tetra lsp --stdio-smoke examples/flow_hello.tetra --format=json >"$tmp_dir/lsp-smoke.json"
run_tetra lsp --stdio-smoke examples/flow_hello.tetra --format=toon >"$tmp_dir/lsp-smoke.toon"
go run ./tools/cmd/validate-lsp-smoke --report "$tmp_dir/lsp-smoke.json" --format=json
go run ./tools/cmd/validate-lsp-smoke --report "$tmp_dir/lsp-smoke.toon" --format=toon
if run_tetra lsp --stdio --format=toon >"$tmp_dir/lsp-stdio-format.stdout" 2>"$tmp_dir/lsp-stdio-format.stderr"; then
  echo "expected lsp --stdio --format=toon to fail" >&2
  exit 1
fi
grep -q "JSON-RPC" "$tmp_dir/lsp-stdio-format.stderr"

run_tetra smoke --list --target linux-x64 --format=json >"$tmp_dir/smoke-list.json"
run_tetra smoke --list --target linux-x64 --format=toon >"$tmp_dir/smoke-list.toon"
go run ./tools/cmd/validate-smoke-list --report "$tmp_dir/smoke-list.json" --examples-root examples --format=json
go run ./tools/cmd/validate-smoke-list --report "$tmp_dir/smoke-list.toon" --examples-root examples --format=toon

run_tetra smoke --target linux-x64 --run=false --report "$tmp_dir/host-smoke.json" --report-format=both
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/host-smoke.json" --format=json
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/host-smoke.toon" --format=toon

go run ./tools/cmd/gen-manifest -o "$tmp_dir/manifest.json" --format=both
go run ./tools/cmd/validate-manifest --manifest "$tmp_dir/manifest.json" --format=json
go run ./tools/cmd/validate-manifest --manifest "$tmp_dir/manifest.toon" --format=toon
diff -u docs/generated/manifest.json "$tmp_dir/manifest.json"

summary_dir="$tmp_dir/test-all-summary"
mkdir -p "$summary_dir/logs"
printf 'ok\n' >"$summary_dir/logs/01-go-test-all-packages.log"
printf 'ok\n' >"$summary_dir/logs/02-json-diagnostic-shape.log"
printf 'ok\n' >"$summary_dir/logs/03-host-smoke-linux-x64.log"
cat >"$summary_dir/summary.json" <<'JSON'
{
  "mode": "quick",
  "status": "pass",
  "started_at": "2026-06-13T12:00:00Z",
  "ended_at": "2026-06-13T12:00:01Z",
  "step_count": 3,
  "failed_count": 0,
  "release_version": "v0.2.0",
  "release_artifact": "tetra.release.v0_2_0.test-all-summary.v1",
  "steps": [
    {
      "name": "go test all packages",
      "status": "pass",
      "duration_seconds": 1,
      "exit_code": 0,
      "command": "go test ./compiler/... ./cli/... ./tools/... -count=1",
      "log": "logs/01-go-test-all-packages.log"
    },
    {
      "name": "json diagnostic shape",
      "status": "pass",
      "duration_seconds": 1,
      "exit_code": 0,
      "command": "check_json_diagnostic",
      "log": "logs/02-json-diagnostic-shape.log"
    },
    {
      "name": "host smoke linux-x64",
      "status": "pass",
      "duration_seconds": 1,
      "exit_code": 0,
      "command": "check_host_smoke",
      "log": "logs/03-host-smoke-linux-x64.log"
    }
  ]
}
JSON
go run ./tools/cmd/json-to-toon --in "$summary_dir/summary.json" --out "$summary_dir/summary.toon"
go run ./tools/cmd/validate-test-all-summary --summary "$summary_dir/summary.json" --report-dir "$summary_dir" --format=json
go run ./tools/cmd/validate-test-all-summary --summary "$summary_dir/summary.toon" --report-dir "$summary_dir" --format=toon

eco_capsule="$tmp_dir/Tetra.capsule"
cat >"$eco_capsule" <<'CAPSULE'
manifest "tetra.capsule.v1"
capsule App:
  id "tetra://app"
  version "0.1.0"
  target "linux-x64"
  permission "io"
CAPSULE
run_tetra eco verify --target linux-x64 --lock "$tmp_dir/eco.lock.json" --lock-format=both "$eco_capsule"
go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/eco.lock.json" --format=json
go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/eco.lock.toon" --format=toon
run_tetra eco seed export --out "$tmp_dir/eco.seed.json" --format=both "$eco_capsule"
go run ./tools/cmd/validate-eco-seed --seed "$tmp_dir/eco.seed.json" --format=json
go run ./tools/cmd/validate-eco-seed --seed "$tmp_dir/eco.seed.toon" --format=toon
run_tetra eco seed import --seed "$tmp_dir/eco.seed.toon" --seed-format=toon --lock "$tmp_dir/eco.seed.lock.json" --lock-format=both --capsules-dir "$tmp_dir/seed-capsules"
go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/eco.seed.lock.json" --format=json
go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/eco.seed.lock.toon" --format=toon
run_tetra eco needmap --lock "$tmp_dir/eco.lock.json" -o "$tmp_dir/eco.needmap.json" --format=both
go run ./tools/cmd/validate-eco-needmap --needmap "$tmp_dir/eco.needmap.json" --format=json
go run ./tools/cmd/validate-eco-needmap --needmap "$tmp_dir/eco.needmap.toon" --format=toon

go_test ./compiler/internal/webrt -run 'TestAcceptExplicitTOON|TestServer(JSON|DB)EndpointSupportsExplicitTOONAccept|TestServerDBEndpointsSupportExplicitTOONAccept' -count=1
go_test ./cli/cmd/tetra -run 'TestLSPCommandSmokeTOONFormat|TestLSPStdioRejectsTOONFormat|TestSmokeCommandWritesTOONReportMirror|TestSmokeCommandListsCasesAsTOON' -count=1
go_test ./tools/cmd/json-to-toon ./tools/cmd/validate-test-all-summary ./tools/cmd/validate-smoke-list ./tools/cmd/smoke-report-to-checklist ./tools/cmd/validate-lsp-smoke ./tools/cmd/validate-manifest ./tools/cmd/validate-eco-lock ./tools/cmd/validate-eco-seed ./tools/cmd/validate-eco-needmap -run 'TOON|JSONToTOON|Smoke|Summary|Manifest|Eco' -count=1
go_test ./internal/toon -run 'TestExampleFixturesMatchCanonicalJSON' -count=1

printf 'OK toon-format-check\n'
