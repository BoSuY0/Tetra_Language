#!/usr/bin/env bash
set -euo pipefail

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

fail() {
  echo "release_v1_0_gate: $*" >&2
  exit 1
}

echo "== v1.0 preflight =="
version="$(./tetra version 2>/dev/null || true)"
if [[ "$version" != v1.0* ]]; then
  fail "expected v1.0.x version, got '${version:-<missing>}'"
fi
short_version="$(./t version 2>/dev/null || true)"
if [[ "$short_version" != "$version" ]]; then
  fail "expected ./t version to match ./tetra version ($version), got '${short_version:-<missing>}'"
fi

echo "== Go test =="
go test ./compiler/... ./cli/... ./tools/...

echo "== Full stabilization wrapper =="
bash scripts/test_all.sh --full --report-dir "$tmp_dir/test-all"

echo "== Flow-only formatter/test/docs =="
go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
./tetra targets --format=json >"$tmp_dir/targets.json"
go run ./tools/cmd/validate-targets --report "$tmp_dir/targets.json"
./tetra doctor --format=json >"$tmp_dir/doctor.json"
go run ./tools/cmd/validate-doctor --report "$tmp_dir/doctor.json"
./tetra check examples/flow_hello.tetra
./tetra fmt --check examples lib __rt compiler/selfhostrt
./tetra test --report=json examples >"$tmp_dir/tetra-test-report.json"
go run ./tools/cmd/validate-test-report --report "$tmp_dir/tetra-test-report.json"
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
./tetra doc examples >"$tmp_dir/tetra-docs.md"
go run ./tools/cmd/validate-api-docs --docs "$tmp_dir/tetra-docs.md"
./tetra smoke --list --format=json >"$tmp_dir/smoke-list.json"
go run ./tools/cmd/validate-smoke-list --report "$tmp_dir/smoke-list.json"

echo "== Mandatory native targets =="
./tetra smoke --target linux-x64 --run=false --report "$tmp_dir/linux-smoke.json"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/linux-smoke.json"
./tetra smoke --target macos-x64 --run=false --report "$tmp_dir/macos-smoke.json"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/macos-smoke.json"
./tetra smoke --target windows-x64 --run=false --report "$tmp_dir/windows-smoke.json"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/windows-smoke.json"

echo "== Mandatory WASM targets =="
./tetra smoke --target wasm32-wasi --run=false --report "$tmp_dir/wasm32-wasi-smoke.json"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/wasm32-wasi-smoke.json"
./tetra smoke --target wasm32-web --run=false --report "$tmp_dir/wasm32-web-smoke.json"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/wasm32-web-smoke.json"

echo "== Eco/UI/API release checks =="
test -x ./tetra
./tetra eco verify --help >/dev/null
go run ./tools/cmd/gen-docs examples >"$tmp_dir/api-docs.md"
go run ./tools/cmd/validate-api-docs --docs "$tmp_dir/api-docs.md"

echo "v1.0 release gate passed"
