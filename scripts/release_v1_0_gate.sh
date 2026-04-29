#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
cd "$repo_root"

report_dir=""
release_version="v1.0.0"
release_artifact="tetra.release.v1_0.gate-report.v1"
formatter_gate_command='./tetra fmt --check examples lib __rt compiler/selfhostrt && go test ./compiler/... ./cli/... -run "Format|Formatter|Comment" -count=1'

usage() {
  cat <<'USAGE'
Usage: bash scripts/release_v1_0_gate.sh [--report-dir DIR]

Notes:
- This gate is for the future v1.0.0 release line.
- It requires ./tetra version == v1.0.0 before mandatory release checks run.
- Artifact mapping: tetra.release.v1_0.gate-report.v1
- Formatter closure gate: ./tetra fmt --check examples lib __rt compiler/selfhostrt && go test ./compiler/... ./cli/... -run "Format|Formatter|Comment" -count=1
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      if [[ $# -lt 2 || -z "${2:-}" ]]; then
        echo "release_v1_0_gate: --report-dir requires a directory" >&2
        usage >&2
        exit 2
      fi
      report_dir="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "release_v1_0_gate: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

echo "release_v1_0_gate: bootstrapping local binaries before v1 version preflight" >&2
bash scripts/bootstrap.sh >&2

version="$(./tetra version 2>/dev/null || true)"
if [[ "$version" != "$release_version" ]]; then
  echo "release_v1_0_gate: blocked: expected ./tetra version to be $release_version, got '${version:-<missing>}'" >&2
  echo "release_v1_0_gate: keep the repository on the current release line until v1 scope evidence is complete." >&2
  echo "release_v1_0_gate: do not promote version metadata until docs/spec/v1_scope.md and docs/checklists/v1_0_release_gate.md are satisfied." >&2
  exit 1
fi

echo "release_v1_0_gate: using shared release gate workflow with v1.0.0 boundaries" >&2
echo "release_v1_0_gate: artifact mapping $release_artifact" >&2
echo "release_v1_0_gate: formatter closure gate: $formatter_gate_command" >&2
if [[ -n "$report_dir" ]]; then
  exec env \
    TETRA_RELEASE_GATE_VERSION="$release_version" \
    TETRA_RELEASE_GATE_ARTIFACT="$release_artifact" \
    TETRA_RELEASE_GATE_COMMAND="bash scripts/release_v1_0_gate.sh" \
    TETRA_RELEASE_GATE_ACTOR_DIAGNOSTIC_CONTAINS="actor declarations currently support state fields and func methods only" \
    bash "$script_dir/release_v0_1_3_gate.sh" --report-dir "$report_dir"
fi
exec env \
  TETRA_RELEASE_GATE_VERSION="$release_version" \
  TETRA_RELEASE_GATE_ARTIFACT="$release_artifact" \
  TETRA_RELEASE_GATE_COMMAND="bash scripts/release_v1_0_gate.sh" \
  TETRA_RELEASE_GATE_ACTOR_DIAGNOSTIC_CONTAINS="actor declarations currently support state fields and func methods only" \
  bash "$script_dir/release_v0_1_3_gate.sh"
