#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
cd "$repo_root"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

check_json_diagnostic_case() {
  local name="$1"
  local contains="$2"
  local source="$tmp_dir/$name.tetra"
  local stdout="$tmp_dir/$name.out"
  local diagnostic="$tmp_dir/$name.json"
  shift 2
  cat >"$source"
  if ./tetra check --diagnostics=json "$source" >"$stdout" 2>"$diagnostic"; then
    echo "expected tetra check --diagnostics=json to fail for $name" >&2
    exit 1
  fi
  test ! -s "$stdout"
  go run ./tools/cmd/validate-diagnostic --diagnostic "$diagnostic" --severity error --contains "$contains" --require-position
}

echo "== Go test =="
go test ./compiler/... ./cli/... ./tools/...

echo "== Repo test script =="
bash scripts/ci/test.sh

echo "== Bootstrap =="
bash scripts/dev/bootstrap.sh

echo "== Version =="
version="$(./tetra version)"
if [[ "$version" != "v0.6.0" ]]; then
  echo "expected v0.6.0, got $version" >&2
  exit 1
fi
echo "$version"
short_version="$(./t version)"
if [[ "$short_version" != "$version" ]]; then
  echo "expected ./t version to match ./tetra version ($version), got $short_version" >&2
  exit 1
fi
echo "$short_version"

echo "== Formatter/test/smoke =="
./tetra fmt --check examples lib __rt compiler/selfhostrt
go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt
./tetra targets --format=json >"$tmp_dir/targets.json"
go run ./tools/cmd/validate-targets --report "$tmp_dir/targets.json"
./tetra doctor --format=json >"$tmp_dir/doctor.json"
go run ./tools/cmd/validate-doctor --report "$tmp_dir/doctor.json"
./tetra check examples/flow_hello.tetra
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
wasm_out="$tmp_dir/flow_hello.wasm"
./tetra build --target wasm32-wasi -o "$wasm_out" examples/hello.tetra >"$tmp_dir/wasm-target-build.out" 2>"$tmp_dir/wasm-target-build.err"
test -s "$wasm_out"
test "$(od -An -tx1 -N4 "$wasm_out" | tr -d ' \n')" = "0061736d"
./tetra smoke --list --format=json >"$tmp_dir/smoke-list.json"
go run ./tools/cmd/validate-smoke-list --report "$tmp_dir/smoke-list.json"
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

echo "== LSP =="
./tetra lsp --stdio-smoke examples/flow_hello.tetra >"$tmp_dir/lsp-smoke.json"
go run ./tools/cmd/validate-lsp-smoke --report "$tmp_dir/lsp-smoke.json"
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

echo "== Generated API docs =="
./tetra doc examples >"$tmp_dir/tetra-docs.md"
go run ./tools/cmd/validate-api-docs --docs "$tmp_dir/tetra-docs.md"
go run ./tools/cmd/gen-docs examples >"$tmp_dir/api-docs.md"
go run ./tools/cmd/validate-api-docs --docs "$tmp_dir/api-docs.md"

echo "== Eco graph, bundle, and local Todex vault =="
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

echo "== Cross-target build-only smoke =="
./tetra smoke --target linux-x64 --run=false --report "$tmp_dir/linux-smoke.json"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/linux-smoke.json"
./tetra smoke --target macos-x64 --run=false --report "$tmp_dir/macos-smoke.json"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/macos-smoke.json"
./tetra smoke --target windows-x64 --run=false --report "$tmp_dir/windows-smoke.json"
go run ./tools/cmd/smoke-report-to-checklist --validate-only --report "$tmp_dir/windows-smoke.json"

echo "v0.6.0 release gate passed"
