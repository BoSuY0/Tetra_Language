#!/usr/bin/env bash
set -euo pipefail

signoff_path=""
template_path=""

usage() {
  cat <<'USAGE'
Usage:
  bash scripts/release_v1_0_security_review.sh --signoff PATH
  bash scripts/release_v1_0_security_review.sh --write-template PATH

Validates the named security review signoff artifact required by the current
release gate, or writes a fill-in template for that artifact.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --signoff)
      signoff_path="$2"
      shift 2
      ;;
    --write-template)
      template_path="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "security_review: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -n "$template_path" && -n "$signoff_path" ]]; then
  echo "security_review: choose either --signoff or --write-template" >&2
  exit 2
fi

if [[ -n "$template_path" ]]; then
  mkdir -p "$(dirname "$template_path")"
  cat >"$template_path" <<'TEMPLATE'
# v0.1.1 Security Review Signoff

Reviewer: <name and contact>
Reviewed commit: <git commit sha>
Report directory: <release report directory>
Decision: <approved for v0.1.1 release | blocked>

## Evidence Commands

- `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: <pass/fail, date, log path>
- `go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1`: <pass/fail, date, log path>
- `go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1`: <pass/fail, date, log path>
- `go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1`: <pass/fail, date, log path>
- `bash scripts/release_v1_0_wasi_smoke.sh --report <path>`: <pass/fail, date, log path>
- `bash scripts/release_v1_0_web_smoke.sh --report <path>`: <pass/fail, date, log path>

## Artifact Hashes

- <artifact file name>: sha256:<64 lowercase hex chars>

## Residual Risks

- <accepted residual risk or "None">
TEMPLATE
  echo "security review signoff template: $template_path"
  exit 0
fi

if [[ -z "$signoff_path" ]]; then
  usage >&2
  exit 2
fi

if [[ ! -f "$signoff_path" ]]; then
  echo "security_review: missing signoff artifact $signoff_path" >&2
  echo "security_review: create one with: bash scripts/release_v1_0_security_review.sh --write-template $signoff_path" >&2
  exit 1
fi

current_head="$(git rev-parse HEAD)"
text="$(cat "$signoff_path")"

require_line() {
  local pattern="$1"
  local description="$2"
  if ! grep -Eq "$pattern" "$signoff_path"; then
    echo "security_review: missing or invalid $description in $signoff_path" >&2
    exit 1
  fi
}

if grep -Eq '<(name and contact|git commit sha|release report directory|approved for v0\.1\.1 release \| blocked|pass/fail, date, log path|artifact file name|64 lowercase hex chars|accepted residual risk or "None")>|TODO|TBD' "$signoff_path"; then
  echo "security_review: signoff contains template placeholder text" >&2
  exit 1
fi

require_line '^Reviewer: .+' 'Reviewer'
require_line "^Reviewed commit: $current_head$" 'Reviewed commit'
require_line '^Report directory: .+' 'Report directory'
require_line '^Decision: approved for v0\.1\.1 release$' 'Decision'
require_line '^## Evidence Commands$' 'Evidence Commands section'
require_line '^## Artifact Hashes$' 'Artifact Hashes section'
require_line '^## Residual Risks$' 'Residual Risks section'

for command in \
  'go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json' \
  "go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1" \
  "go test ./compiler/... -run 'Privacy|Consent|Budget|Effect' -count=1" \
  "go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1" \
  'bash scripts/release_v1_0_wasi_smoke.sh --report <path>' \
  'bash scripts/release_v1_0_web_smoke.sh --report <path>'
do
  if [[ "$text" != *"\`$command\`: pass"* ]]; then
    echo "security_review: missing passing evidence command: $command" >&2
    exit 1
  fi
done

artifact_hash_lines="$(awk '
  /^## Artifact Hashes$/ { in_hashes=1; next }
  /^## / && in_hashes { in_hashes=0 }
  in_hashes && /^- / { print }
' "$signoff_path")"
if [[ -z "$artifact_hash_lines" ]]; then
  echo "security_review: missing artifact hash entries" >&2
  exit 1
fi
while IFS= read -r line; do
  if [[ ! "$line" =~ ^-[[:space:]][A-Za-z0-9._/-]+:[[:space:]]sha256:[0-9a-f]{64}$ ]]; then
    echo "security_review: invalid artifact hash entry: $line" >&2
    exit 1
  fi
done <<< "$artifact_hash_lines"

residual_risk_lines="$(awk '
  /^## Residual Risks$/ { in_risks=1; next }
  /^## / && in_risks { in_risks=0 }
  in_risks && /^- / { print }
' "$signoff_path")"
if [[ -z "$residual_risk_lines" ]]; then
  echo "security_review: missing accepted residual risk entries" >&2
  exit 1
fi

echo "security review signoff valid: $signoff_path"
