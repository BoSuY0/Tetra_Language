#!/usr/bin/env bash
set -euo pipefail

fuzztime="10m"
out_dir=""
short=false

usage() {
  cat <<'USAGE'
Usage: bash scripts/fuzz_nightly.sh [--short] [--fuzztime DURATION] [--out-dir DIR]

Runs the bounded fuzz/property/stress nightly suite one package at a time and
writes logs plus summary.md. Crashers remain in Go's package-local testdata
fuzz corpus; copy new reproducers into deterministic tests before release.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --short)
      short=true
      fuzztime="1s"
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
if [[ -z "$out_dir" ]]; then
  out_dir="reports/fuzz-nightly-$timestamp"
fi
logs_dir="$out_dir/logs"
summary="$out_dir/summary.md"
mkdir -p "$logs_dir"

steps=(
  "compiler-frontend-lexer|go test ./compiler/internal/frontend -run '^$' -fuzz=FuzzLexer -fuzztime=$fuzztime"
  "compiler-frontend-parser|go test ./compiler/internal/frontend -run '^$' -fuzz=FuzzParser -fuzztime=$fuzztime"
  "validate-manifest|go test ./tools/cmd/validate-manifest -run '^$' -fuzz=. -fuzztime=$fuzztime"
  "eco-capsule|go test ./cli/cmd/tetra -run '^$' -fuzz=FuzzParseCapsuleDoesNotPanic -fuzztime=$fuzztime"
  "property-stress-regressions|go test ./compiler/... ./cli/... ./tools/cmd/validate-manifest -run 'Fuzz|Property|Stress' -count=1"
)

{
  echo "# Fuzz Nightly Summary"
  echo
  echo "- mode: \`$([[ "$short" == true ]] && echo short || echo nightly)\`"
  echo "- fuzztime: \`$fuzztime\`"
  echo "- output_dir: \`$out_dir\`"
  echo "- crasher_archive_path: \`<package>/testdata/fuzz/<FuzzName>/\`"
  echo
  echo "## Steps"
} >"$summary"

failed=0
for step in "${steps[@]}"; do
  name="${step%%|*}"
  command="${step#*|}"
  log="$logs_dir/$name.log"
  if bash -c "$command" >"$log" 2>&1; then
    printf -- '- `%s`: pass, command `%s`, log `%s`\n' "$name" "$command" "$log" >>"$summary"
  else
    code="$?"
    failed=$((failed + 1))
    printf -- '- `%s`: fail exit `%s`, command `%s`, log `%s`\n' "$name" "$code" "$command" "$log" >>"$summary"
    tail -n 80 "$log" >&2 || true
  fi
done

if [[ "$failed" -ne 0 ]]; then
  echo "fuzz nightly failed: $failed step(s); see $summary" >&2
  exit 1
fi

echo "fuzz nightly passed: $summary"
