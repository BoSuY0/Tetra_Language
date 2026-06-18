#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
signoff_path=""

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/v0_3_0/security-review.sh --signoff PATH
       bash scripts/release/v0_3_0/security-review.sh --write-template PATH

Validates or writes the v0.3.0 security review signoff artifact.

When a signoff names an existing release report directory, the validator also
requires the canonical gate artifacts to be listed and verified:
summary.json, artifact-hashes.json, release-state.json, and security-review.md.
All artifact hash entries listed in the signoff are checked against files under
the named Report directory.
USAGE
}

sha256_file() {
  local path="$1"
  if command -v sha256sum > /dev/null 2>&1; then
    sha256sum "$path" | awk '{print "sha256:" $1}'
    return
  fi
  shasum -a 256 "$path" | awk '{print "sha256:" $1}'
}

listed_hash_for_artifact() {
  local artifact="$1"
  awk -v artifact="$artifact" '
    /^## Artifact Hashes$/ { in_hashes=1; next }
    /^## / && in_hashes { in_hashes=0 }
    in_hashes && /^- / {
      line=$0
      sub(/^-[[:space:]]+/, "", line)
      split(line, fields, /:[[:space:]]+/)
      if ((fields[1] == artifact || fields[1] == "artifacts/" artifact) && fields[2] ~ /^sha256:[0-9a-f]{64}$/) {
        print fields[2]
        exit
      }
    }
  ' "$signoff_path"
}

require_listed_artifact() {
  local artifact="$1"
  if ! grep -Fq "$artifact" "$signoff_path"; then
    echo "release_v0_3_0_security_review: missing required canonical gate artifact listing: $artifact" >&2
    exit 1
  fi
}

require_hashed_artifact_match() {
  local artifact="$1"
  local path="$2"
  require_listed_artifact "$artifact"
  if [[ ! -f "$path" ]]; then
    echo "release_v0_3_0_security_review: missing required canonical gate artifact: $artifact ($path)" >&2
    exit 1
  fi
  local listed_hash
  listed_hash="$(listed_hash_for_artifact "$artifact")"
  if [[ -z "$listed_hash" ]]; then
    echo "release_v0_3_0_security_review: missing required canonical gate artifact hash: $artifact" >&2
    exit 1
  fi
  local actual_hash
  actual_hash="$(sha256_file "$path")"
  if [[ "$listed_hash" != "$actual_hash" ]]; then
    echo "release_v0_3_0_security_review: canonical gate artifact $artifact hash mismatch: listed $listed_hash, actual $actual_hash" >&2
    exit 1
  fi
}

listed_artifact_hash_lines() {
  awk '
    /^## Artifact Hashes$/ { in_hashes=1; next }
    /^## / && in_hashes { in_hashes=0 }
    in_hashes && /^- / { print }
  ' "$signoff_path"
}

resolve_report_artifact_path() {
  local artifact="$1"
  local report_dir="$2"
  local candidate="$report_dir/$artifact"
  if [[ -f "$candidate" ]]; then
    printf '%s\n' "$candidate"
    return 0
  fi
  if [[ "$artifact" != */* ]]; then
    candidate="$report_dir/artifacts/$artifact"
    if [[ -f "$candidate" ]]; then
      printf '%s\n' "$candidate"
      return 0
    fi
  fi
  return 1
}

verify_listed_artifact_hashes() {
  local report_dir="$1"
  local line
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    if [[ ! "$line" =~ ^-[[:space:]]([A-Za-z0-9._/-]+):[[:space:]](sha256:[0-9a-f]{64})$ ]]; then
      echo "release_v0_3_0_security_review: invalid listed artifact hash entry: $line" >&2
      exit 1
    fi

    local artifact="${BASH_REMATCH[1]}"
    local listed_hash="${BASH_REMATCH[2]}"
    case "$artifact" in
      /* | ../* | */../* | */..)
        echo "release_v0_3_0_security_review: listed artifact path must be relative to Report directory: $artifact" >&2
        exit 1
        ;;
    esac

    local artifact_path
    if ! artifact_path="$(resolve_report_artifact_path "$artifact" "$report_dir")"; then
      echo "release_v0_3_0_security_review: missing listed artifact file: $artifact (Report directory: $report_dir)" >&2
      exit 1
    fi

    local actual_hash
    actual_hash="$(sha256_file "$artifact_path")"
    if [[ "$listed_hash" != "$actual_hash" ]]; then
      echo "release_v0_3_0_security_review: listed artifact $artifact hash mismatch: listed $listed_hash, actual $actual_hash" >&2
      exit 1
    fi
  done < <(listed_artifact_hash_lines)
}

validate_canonical_gate_artifacts() {
  if [[ -z "$signoff_path" || ! -f "$signoff_path" ]]; then
    return
  fi

  local report_dir
  report_dir="$(sed -nE 's/^Report directory:[[:space:]]*(.+)$/\1/p' "$signoff_path" | head -n 1)"
  if [[ -z "$report_dir" ]]; then
    return
  fi

  require_hashed_artifact_match "summary.json" "$report_dir/summary.json"
  require_hashed_artifact_match "artifact-hashes.json" "$report_dir/artifact-hashes.json"
  require_hashed_artifact_match "release-state.json" "$report_dir/artifacts/release-state.json"

  local security_artifact="$report_dir/artifacts/security-review.md"
  require_listed_artifact "security-review.md"
  if [[ ! -f "$security_artifact" ]]; then
    echo "release_v0_3_0_security_review: missing required canonical gate artifact: security-review.md ($security_artifact)" >&2
    exit 1
  fi
  if ! cmp -s "$signoff_path" "$security_artifact"; then
    echo "release_v0_3_0_security_review: canonical gate artifact security-review.md mismatch: archived artifact differs from signoff" >&2
    exit 1
  fi

  verify_listed_artifact_hashes "$report_dir"
}

case "${1:-}" in
  -h | --help)
    usage
    exit 0
    ;;
  "")
    usage >&2
    exit 2
    ;;
esac

args=("$@")
arg_index=0
while [[ "$arg_index" -lt "${#args[@]}" ]]; do
  case "${args[$arg_index]}" in
    --signoff)
      next_index=$((arg_index + 1))
      signoff_path="${args[$next_index]:-}"
      arg_index=$((arg_index + 2))
      ;;
    --write-template)
      arg_index=$((arg_index + 2))
      ;;
    *)
      arg_index=$((arg_index + 1))
      ;;
  esac
done

if [[ ! -f "$repo_root/scripts/release/v1_0/security-review.sh" ]]; then
  echo "release_v0_3_0_security_review: missing validator implementation: $repo_root/scripts/release/v1_0/security-review.sh" >&2
  exit 2
fi

validate_canonical_gate_artifacts

exec bash "$repo_root/scripts/release/v1_0/security-review.sh" "$@"
