#!/usr/bin/env bash
set -euo pipefail

release_version="v0.4.0"
signoff_path=""
template_path=""

usage() {
  cat << 'USAGE'
Usage:
  bash scripts/release/v0_4_0/security-review.sh --signoff PATH
  bash scripts/release/v0_4_0/security-review.sh --write-template PATH

Validates or writes the v0.4.0 security review signoff artifact. Validation is
scoped to the Linux-x64 v0.4.0 production objective. EcoNet, WASI/Web runtime
targets, Windows, macOS, and full v1.0 language guarantees are outside this
signoff boundary.
USAGE
}

current_release_version() {
  local version=""
  if [[ -f compiler/internal/version/version.go ]]; then
    version="$(sed -nE 's/^const CompilerVersion = "([^"]+)"/\1/p' compiler/internal/version/version.go | head -n 1)"
  fi
  if [[ -z "$version" && -x ./tetra ]]; then
    version="$(./tetra version 2> /dev/null || true)"
  fi
  if [[ -z "$version" ]]; then
    echo "release_v0_4_0_security_review: cannot determine current release version" >&2
    exit 2
  fi
  printf '%s\n' "$version"
}

regex_escape() {
  printf '%s' "$1" | sed -E 's/[][\\.^$*+?(){}|]/\\&/g'
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --signoff)
      signoff_path="${2:-}"
      shift 2
      ;;
    --write-template)
      template_path="${2:-}"
      shift 2
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "release_v0_4_0_security_review: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -n "$template_path" && -n "$signoff_path" ]]; then
  echo "release_v0_4_0_security_review: choose either --signoff or --write-template" >&2
  exit 2
fi

if [[ -n "$template_path" ]]; then
  mkdir -p "$(dirname "$template_path")"
  cat > "$template_path" << 'TEMPLATE'
# v0.4.0 Security Review Signoff

Reviewer: <name and contact>
Reviewed commit: <git commit sha>
Report directory: <release report directory>
Decision: <approved for v0.4.0 release | blocked>

## Evidence Commands

- `go run ./tools/cmd/validate-v0-4-readiness --features <features.json> --targets <targets.json> --manifest docs/generated/manifest.json --scope-decisions docs/release/v0_4/data/v0_4_0_scope_decisions.json`: <pass/fail, date, log path>
- `go test ./compiler/... ./cli/... ./tools/... -count=1`: <pass/fail, date, log path>
- `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: <pass/fail, date, log path>
- `go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --report reports/v0.4.0/linux-host-smoke.json`: <pass/fail, date, log path>
- `bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh --report-dir reports/v0.4.0`: <pass/fail, date, log path>
- `bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir reports/v0.4.0`: <pass/fail, date, log path>
- `git diff --check`: <pass/fail, date, log path>

## Security Areas

- filesystem, networking, crypto, and capability effects for the scoped Linux surface: <approved/blocked>
- Linux-x64 runtime execution and native process boundaries: <approved/blocked>
- UI event dispatch and command execution: <approved/blocked>
- distributed actors, scheduling, cancellation, and failure modes: <approved/blocked>
- artifact hashes and release-state integrity: <approved/blocked>
- excluded EcoNet/WASI/Web/Windows/macOS boundaries are not part of this v0.4.0 production signoff: <approved/blocked>

## Artifact Hashes

- <artifact file name>: sha256:<64 lowercase hex chars>

## Residual Risks

- <accepted residual risk or "None">
TEMPLATE
  echo "v0.4.0 security review signoff template: $template_path"
  exit 0
fi

if [[ -z "$signoff_path" ]]; then
  usage >&2
  exit 2
fi

current_version="$(current_release_version)"
if [[ "$current_version" != "$release_version" ]]; then
  echo "release_v0_4_0_security_review: expected current release version $release_version, got $current_version" >&2
  echo "release_v0_4_0_security_review: v0.4.0 security review remains blocked" >&2
  exit 1
fi

if [[ ! -f "$signoff_path" ]]; then
  echo "release_v0_4_0_security_review: missing signoff artifact $signoff_path" >&2
  exit 1
fi

current_head="$(git rev-parse HEAD)"
release_version_re="$(regex_escape "$release_version")"
text="$(cat -- "$signoff_path")"

require_line() {
  local pattern="$1"
  local description="$2"
  if ! grep -Eq "$pattern" -- "$signoff_path"; then
    echo "release_v0_4_0_security_review: missing or invalid $description in $signoff_path" >&2
    exit 1
  fi
}

if grep -Eq "<(name and contact|git commit sha|release report directory|approved for ${release_version_re} release \| blocked|pass/fail, date, log path|artifact file name|64 lowercase hex chars|accepted residual risk or \"None\")>|TODO|TBD" -- "$signoff_path"; then
  echo "release_v0_4_0_security_review: signoff contains template placeholder text" >&2
  exit 1
fi

require_line '^Reviewer: .+' 'Reviewer'
require_line "^Reviewed commit: $current_head$" 'Reviewed commit'
require_line '^Report directory: .+' 'Report directory'
require_line "^Decision: approved for ${release_version_re} release$" 'Decision'
require_line '^## Evidence Commands$' 'Evidence Commands section'
require_line '^## Security Areas$' 'Security Areas section'
require_line '^## Artifact Hashes$' 'Artifact Hashes section'
require_line '^## Residual Risks$' 'Residual Risks section'

for command in \
  'go run ./tools/cmd/validate-v0-4-readiness --features <features.json> --targets <targets.json> --manifest docs/generated/manifest.json --scope-decisions docs/release/v0_4/data/v0_4_0_scope_decisions.json' \
  'go test ./compiler/... ./cli/... ./tools/... -count=1' \
  'go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json' \
  'go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --report reports/v0.4.0/linux-host-smoke.json' \
  'bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh --report-dir reports/v0.4.0' \
  'bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir reports/v0.4.0' \
  'git diff --check'; do
  if [[ "$text" != *"\`$command\`: pass"* ]]; then
    echo "release_v0_4_0_security_review: missing passing evidence command: $command" >&2
    exit 1
  fi
done

for area in \
  'filesystem, networking, crypto, and capability effects for the scoped Linux surface: approved' \
  'Linux-x64 runtime execution and native process boundaries: approved' \
  'UI event dispatch and command execution: approved' \
  'distributed actors, scheduling, cancellation, and failure modes: approved' \
  'artifact hashes and release-state integrity: approved' \
  'excluded EcoNet/WASI/Web/Windows/macOS boundaries are not part of this v0.4.0 production signoff: approved'; do
  if [[ "$text" != *"$area"* ]]; then
    echo "release_v0_4_0_security_review: missing approved security area: $area" >&2
    exit 1
  fi
done

artifact_hash_lines="$(awk '
  /^## Artifact Hashes$/ { in_hashes=1; next }
  /^## / && in_hashes { in_hashes=0 }
  in_hashes && /^- / { print }
' "$signoff_path")"
if [[ -z "$artifact_hash_lines" ]]; then
  echo "release_v0_4_0_security_review: missing artifact hash entries" >&2
  exit 1
fi
while IFS= read -r line; do
  if [[ ! "$line" =~ ^-[[:space:]][A-Za-z0-9._/-]+:[[:space:]]sha256:[0-9a-f]{64}$ ]]; then
    echo "release_v0_4_0_security_review: invalid artifact hash entry: $line" >&2
    exit 1
  fi
done <<< "$artifact_hash_lines"

residual_risk_lines="$(awk '
  /^## Residual Risks$/ { in_risks=1; next }
  /^## / && in_risks { in_risks=0 }
  in_risks && /^- / { print }
' "$signoff_path")"
if [[ -z "$residual_risk_lines" ]]; then
  echo "release_v0_4_0_security_review: missing accepted residual risk entries" >&2
  exit 1
fi

echo "v0.4.0 security review signoff valid: $signoff_path"
