#!/usr/bin/env bash
set -euo pipefail

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

echo "== Go test =="
go test ./compiler/... ./cli/... ./tools/...

echo "== Repo test script =="
bash scripts/test.sh

echo "== Bootstrap =="
bash scripts/bootstrap.sh

echo "== Version =="
version="$(./tetra version)"
if [[ "$version" != "v0.5.0" ]]; then
  echo "expected v0.5.0, got $version" >&2
  exit 1
fi
echo "$version"

echo "== Formatter/test/smoke =="
./tetra fmt --check examples/flow_hello.tetra
./tetra test examples
./tetra test --report=json examples >"$tmp_dir/tetra-test-report.json"
go run ./tools/cmd/validate-test-report --report "$tmp_dir/tetra-test-report.json"
./tetra smoke --target linux-x64 --run=true --report "$tmp_dir/host-smoke.json"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/host-smoke.json"

echo "== Docs manifest =="
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/gen-manifest -o "$tmp_dir/manifest.json"
go run ./tools/cmd/validate-manifest --manifest "$tmp_dir/manifest.json"
diff -u docs/generated/manifest.json "$tmp_dir/manifest.json"
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json

echo "== LSP and generated API docs =="
./tetra lsp --stdio-smoke examples/flow_hello.tetra >"$tmp_dir/lsp.json"
go run ./tools/cmd/validate-lsp-smoke --report "$tmp_dir/lsp.json"
go run ./tools/cmd/gen-docs examples >"$tmp_dir/api-docs.md"
go run ./tools/cmd/validate-api-docs --docs "$tmp_dir/api-docs.md"

echo "== Eco graph and local Todex vault =="
cat >"$tmp_dir/Core.capsule" <<'CAPSULE'
capsule Core:
  id "tetra://core"
  version "0.1.0"
  target "linux-x64"
CAPSULE
cat >"$tmp_dir/App.capsule" <<'CAPSULE'
capsule App:
  id "tetra://app"
  version "0.1.0"
  target "linux-x64"
  dependency "tetra://core" "0.1.0"
CAPSULE
./tetra eco verify --target linux-x64 --lock "$tmp_dir/tetra.lock.json" "$tmp_dir/App.capsule" "$tmp_dir/Core.capsule"
go run ./tools/cmd/validate-eco-lock --lock "$tmp_dir/tetra.lock.json"
./tetra eco vault add --store "$tmp_dir/vault" --kind source examples/flow_hello.tetra
./tetra eco vault list --store "$tmp_dir/vault"
./tetra eco vault verify --store "$tmp_dir/vault"
go run ./tools/cmd/validate-eco-vault --store "$tmp_dir/vault"

echo "== Cross-target build-only smoke =="
./tetra smoke --target linux-x64 --run=false --report "$tmp_dir/linux-smoke.json"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/linux-smoke.json"
./tetra smoke --target macos-x64 --run=false --report "$tmp_dir/macos-smoke.json"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/macos-smoke.json"
./tetra smoke --target windows-x64 --run=false --report "$tmp_dir/windows-smoke.json"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/windows-smoke.json"

echo "v0.5.0 release gate passed"
