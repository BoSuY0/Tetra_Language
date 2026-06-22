#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
cd "$repo_root"

report_dir=""
require_clean=0
release_version="v0.3.0"
release_artifact="tetra.release.v0_3_0.gate-report.v1"
release_gate_command="bash scripts/release/v0_3_0/gate.sh"

usage() {
  cat << 'USAGE'
Usage: bash scripts/release/v0_3_0/gate.sh [--report-dir DIR] [--require-clean]

Notes:
- This gate is for the future v0.3.0 release line.
- It requires ./tetra version == v0.3.0 and ./t version parity.
- Artifact mapping: tetra.release.v0_3_0.gate-report.v1
- It runs the v0.3.0 stabilization, short fuzz, docs, security signoff, and
  whitespace envelope.
- Use --require-clean for tag-ready promotion so dirty tracked or untracked
  files block the gate before report artifacts are created.
- macOS/Windows runtime execution evidence is host-gated. Provide
  TETRA_MACOS_RUNTIME_SMOKE_REPORT and TETRA_WINDOWS_RUNTIME_SMOKE_REPORT,
  each produced by `tetra smoke --target <target> --run=true --report <path>`
  on its matching native host or downloaded from the matching CI smoke artifact.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --report-dir)
      if [[ $# -lt 2 || -z "${2:-}" ]]; then
        echo "release_v0_3_0_gate: --report-dir requires a directory" >&2
        usage >&2
        exit 2
      fi
      report_dir="$2"
      shift 2
      ;;
    --require-clean)
      require_clean=1
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "release_v0_3_0_gate: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$report_dir" ]]; then
  report_dir="reports/release-v0.3.0-gate-$(date -u +%Y%m%d-%H%M%S)"
fi

check_report_dir_fresh() {
  if [[ (-e "$report_dir" || -L "$report_dir") && ! -d "$report_dir" ]]; then
    echo "release_v0_3_0_gate: refusing to use non-directory report path: $report_dir" >&2
    echo "release_v0_3_0_gate: choose a fresh --report-dir directory" >&2
    exit 2
  fi
  if [[ ! -d "$report_dir" ]]; then
    return 0
  fi
  local first_entry
  local find_report_dir="$report_dir"
  if [[ "$find_report_dir" == -* ]]; then
    find_report_dir="./$find_report_dir"
  fi
  first_entry="$(find -H "$find_report_dir" -mindepth 1 -print -quit)"
  if [[ -n "$first_entry" ]]; then
    echo "release_v0_3_0_gate: refusing to reuse non-empty report directory: $report_dir" >&2
    echo "release_v0_3_0_gate: choose a fresh --report-dir so stale reports cannot be reused" >&2
    exit 2
  fi
}

check_report_dir_fresh

check_tag_ready_clean_worktree() {
  local status
  if ! status="$(git status --porcelain --untracked-files=all 2>&1)"; then
    echo "release_v0_3_0_gate: tag-ready clean worktree check failed" >&2
    printf '%s\n' "$status" >&2
    return 1
  fi
  if [[ -n "$status" ]]; then
    echo "release_v0_3_0_gate: blocked: tag-ready clean worktree required (--require-clean)" >&2
    echo "release_v0_3_0_gate: git status --porcelain --untracked-files=all" \
      "reported dirty state:" >&2
    printf '%s\n' "$status" >&2
    return 1
  fi
}

if [[ "$require_clean" -eq 1 ]]; then
  check_tag_ready_clean_worktree
fi

logs_dir="$report_dir/logs"
summary_md="$report_dir/summary.md"
summary_json="$report_dir/summary.json"
artifacts_dir="$report_dir/artifacts"
residual_risks_json="$artifacts_dir/residual-risks.json"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
staged_security_review_signoff=""
ci_missing_security_signoff=0

mkdir -p "$logs_dir" "$artifacts_dir"
: > "$tmp_dir/steps.md"
: > "$tmp_dir/steps.jsonl"

started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
step_count=0
failed_count=0

json_escape() {
  local s="${1-}"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  printf '%s' "$s"
}

sha256_file() {
  local path="$1"
  if command -v sha256sum > /dev/null 2>&1; then
    sha256sum "$path" | awk '{print "sha256:" $1}'
    return
  fi
  shasum -a 256 "$path" | awk '{print "sha256:" $1}'
}

slugify() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//'
}

record_step() {
  local name="$1"
  local status="$2"
  local seconds="$3"
  local exit_code="$4"
  local log_rel="$5"
  local command="$6"

  printf -- \
    '- `%s`: `%s` in %ss, exit `%s`, command `%s` ([%s](%s))\n' \
    "$name" \
    "$status" \
    "$seconds" \
    "$exit_code" \
    "$command" \
    "$log_rel" \
    "$log_rel" >> "$tmp_dir/steps.md"

  local step_json_format
  step_json_format='{"name":"%s","status":"%s","duration_seconds":%s,'
  step_json_format+='"exit_code":%s,"command":"%s","log":"%s"}\n'
  printf "$step_json_format" \
    "$(json_escape "$name")" \
    "$(json_escape "$status")" \
    "$seconds" \
    "$exit_code" \
    "$(json_escape "$command")" \
    "$(json_escape "$log_rel")" >> "$tmp_dir/steps.jsonl"
}

run_step() {
  local name="$1"
  shift
  step_count=$((step_count + 1))

  local id
  local slug
  local log_rel
  local log_path
  local command
  local start_s
  local end_s

  id="$(printf '%02d' "$step_count")"
  slug="$(slugify "$name")"
  log_rel="logs/${id}-${slug}.log"
  log_path="$report_dir/$log_rel"
  command="$*"

  printf '== [%s] %s ==\n' "$id" "$name"
  start_s="$(date +%s)"

  if "$@" > "$log_path" 2>&1; then
    end_s="$(date +%s)"
    record_step "$name" "pass" "$((end_s - start_s))" 0 "$log_rel" "$command"
    printf '   pass (%ss)\n' "$((end_s - start_s))"
  else
    local rc="$?"
    end_s="$(date +%s)"
    record_step "$name" "fail" "$((end_s - start_s))" "$rc" "$log_rel" "$command"
    failed_count=$((failed_count + 1))
    printf '   fail (%ss), exit %s\n' "$((end_s - start_s))" "$rc" >&2
    tail -n 60 "$log_path" >&2 || true
  fi
}

record_failed_step() {
  local name="$1"
  local command="$2"
  local exit_code="$3"
  local message="$4"

  step_count=$((step_count + 1))

  local id
  local slug
  local log_rel
  local log_path

  id="$(printf '%02d' "$step_count")"
  slug="$(slugify "$name")"
  log_rel="logs/${id}-${slug}.log"
  log_path="$report_dir/$log_rel"

  {
    printf 'post-summary release gate failure\n'
    printf 'command: %s\n' "$command"
    printf 'exit_code: %s\n' "$exit_code"
    printf 'reason: %s\n' "$message"
  } > "$log_path"

  record_step "$name" "fail" 0 "$exit_code" "$log_rel" "$command"
  failed_count=$((failed_count + 1))
}

write_summary() {
  local status="$1"
  local ended_at
  ended_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  {
    echo "# Tetra $release_version Release Gate Report"
    echo
    echo "- status: \`$status\`"
    echo "- release_version: \`$release_version\`"
    echo "- release_artifact: \`$release_artifact\`"
    echo "- release_gate_command: \`$release_gate_command\`"
    echo "- started_at: \`$started_at\`"
    echo "- ended_at: \`$ended_at\`"
    echo "- step_count: \`$step_count\`"
    echo "- failed_count: \`$failed_count\`"
    echo "- report_dir: \`$report_dir\`"
    echo
    echo "## Steps"
    echo
    cat "$tmp_dir/steps.md"
  } > "$summary_md"

  {
    echo "{"
    printf '  "status": "%s",\n' "$(json_escape "$status")"
    printf '  "release_version": "%s",\n' "$(json_escape "$release_version")"
    printf '  "release_artifact": "%s",\n' "$(json_escape "$release_artifact")"
    printf '  "release_gate_command": "%s",\n' "$(json_escape "$release_gate_command")"
    printf '  "started_at": "%s",\n' "$(json_escape "$started_at")"
    printf '  "ended_at": "%s",\n' "$(json_escape "$ended_at")"
    printf '  "step_count": %s,\n' "$step_count"
    printf '  "failed_count": %s,\n' "$failed_count"
    printf '  "report_dir": "%s",\n' "$(json_escape "$report_dir")"
    echo '  "steps": ['
    awk '
      NR > 1 {
        printf ",\n"
      }
      {
        printf "    %s", $0
      }
      END {
        if (NR > 0) {
          printf "\n"
        }
      }
    ' "$tmp_dir/steps.jsonl"
    echo '  ]'
    echo "}"
  } > "$summary_json"
}

validate_summary() {
  go run ./tools/cmd/validate-release-gate-summary --summary "$summary_json" --report-dir "$report_dir"
}

check_release_version() {
  local version
  version="$(./tetra version 2> /dev/null || true)"
  if [[ "$version" != "$release_version" ]]; then
    echo "expected ./tetra version to be $release_version, got '${version:-<missing>}'" >&2
    return 1
  fi
}

check_short_alias_version() {
  local version
  local short_version
  version="$(./tetra version 2> /dev/null || true)"
  short_version="$(./t version 2> /dev/null || true)"
  if [[ "$short_version" != "$version" ]]; then
    echo "expected ./t version to match ./tetra version ($version)," \
      "got '${short_version:-<missing>}'" >&2
    return 1
  fi
}

check_go_test_packages() {
  env \
    -u TETRA_SECURITY_REVIEW_SIGNOFF \
    -u TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF \
    -u TETRA_MACOS_RUNTIME_SMOKE_REPORT \
    -u TETRA_WINDOWS_RUNTIME_SMOKE_REPORT \
    -u TETRA_RESIDUAL_RISKS_JSON \
    go test ./compiler/... ./cli/... ./tools/... -count=1
}

check_release_state() {
  go run ./tools/cmd/validate-release-state \
    --expected-version "$release_version" \
    --format=json \
    --report-dir "$report_dir" > "$artifacts_dir/release-state.json"
  go run ./tools/cmd/validate-release-state \
    --expected-version "$release_version" \
    --format=text \
    --report-dir "$report_dir" > "$artifacts_dir/release-state.txt"
}

check_artifact_hash_manifest() {
  go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"
  canonicalize_artifact_hash_manifest "$report_dir/artifact-hashes.json"
  go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"
}

resolve_python() {
  if command -v python3 > /dev/null 2>&1; then
    command -v python3
    return 0
  fi
  if command -v python > /dev/null 2>&1; then
    command -v python
    return 0
  fi
  echo "release_v0_3_0_gate: artifact hash canonicalization requires python3 or python" >&2
  return 1
}

canonicalize_artifact_hash_manifest() {
  local manifest_path="$1"
  if [[ ! -f "$manifest_path" ]]; then
    return 0
  fi

  local python_bin
  if ! python_bin="$(resolve_python)"; then
    return 1
  fi

  "$python_bin" - "$manifest_path" << 'PY'
import json
import os
import sys

manifest_path = sys.argv[1]
with open(manifest_path, "r", encoding="utf-8") as fh:
    manifest = json.load(fh)

artifacts = manifest.get("artifacts", [])
manifest["artifacts"] = [
    artifact for artifact in artifacts
    if artifact.get("path") not in {
        "artifacts/security-review.md",
        "artifacts/security-review.md.sha256",
    }
]

tmp_path = manifest_path + ".tmp"
with open(tmp_path, "w", encoding="utf-8") as fh:
    json.dump(manifest, fh, indent=2)
    fh.write("\n")
os.replace(tmp_path, manifest_path)
PY
}

check_short_fuzz_summary() {
  go run ./tools/cmd/validate-fuzz-summary --report-dir "$artifacts_dir/fuzz-short"
}

check_unstable_seed_triage() {
  local seed_log="$artifacts_dir/fuzz-short/unstable-seeds.md"
  if [[ ! -f "$seed_log" ]]; then
    echo "release_v0_3_0_gate: unstable-seeds.md triage blocked: missing $seed_log" >&2
    return 1
  fi

  awk '
    function trim(s) {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", s)
      return s
    }
    BEGIN {
      header_seen = 0
      separator_seen = 0
      data_row = 0
      failed = 0
    }
    $0 == "| package | fuzz target | seed/crasher path | status | owner | next command |" {
      header_seen = 1
      next
    }
    header_seen && $0 == "| --- | --- | --- | --- | --- | --- |" {
      separator_seen = 1
      next
    }
    separator_seen && /^[[:space:]]*$/ {
      next
    }
    separator_seen && /^\|/ {
      data_row++
      n = split($0, cells, /\|/)
      if (n != 8) {
        printf "release_v0_3_0_gate: unstable-seeds.md triage " \
          "data row %d malformed table row\n", data_row > "/dev/stderr"
        failed = 1
        next
      }
      package_name = trim(cells[2])
      fuzz_target = trim(cells[3])
      seed_path = trim(cells[4])
      status = trim(cells[5])
      owner = trim(cells[6])
      next_command = trim(cells[7])
      seed_label = seed_path
      if (seed_label == "") {
        seed_label = package_name "/" fuzz_target
      }
      if (status == "") {
        printf "release_v0_3_0_gate: unstable-seeds.md triage " \
          "data row %d missing status for seed %s\n", data_row, seed_label \
          > "/dev/stderr"
        failed = 1
      }
      if (owner == "") {
        printf "release_v0_3_0_gate: unstable-seeds.md triage " \
          "data row %d missing owner for seed %s\n", data_row, seed_label \
          > "/dev/stderr"
        failed = 1
      }
      if (next_command == "") {
        printf "release_v0_3_0_gate: unstable-seeds.md triage " \
          "data row %d missing next command for seed %s\n", data_row, seed_label \
          > "/dev/stderr"
        failed = 1
      }
      next
    }
    END {
      if (!header_seen) {
        print "release_v0_3_0_gate: unstable-seeds.md triage blocked: " \
          "missing triage table header" > "/dev/stderr"
        failed = 1
      }
      if (!separator_seen) {
        print "release_v0_3_0_gate: unstable-seeds.md triage blocked: " \
          "missing triage table separator" > "/dev/stderr"
        failed = 1
      }
      exit failed
    }
  ' "$seed_log"
}

check_residual_risks() {
  local source_path="${TETRA_RESIDUAL_RISKS_JSON:-}"
  if [[ -n "$source_path" ]]; then
    if [[ ! -f "$source_path" ]]; then
      echo "release_v0_3_0_gate: residual-risks.json source missing: $source_path" >&2
      return 1
    fi
    cp -- "$source_path" "$residual_risks_json" || return
  else
    cat > "$residual_risks_json" << EOF
{
  "schema": "tetra.release.residual-risks.v1",
  "release_version": "$release_version",
  "artifact": "residual-risks.json",
  "risks": []
}
EOF
  fi

  if [[ ! -s "$residual_risks_json" ]]; then
    echo "release_v0_3_0_gate: residual-risks.json is empty" >&2
    return 1
  fi

  local validator_go="$tmp_dir/validate_residual_risks.go"
  cat > "$validator_go" << 'GO'
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

const residualRiskPrefix = "release_v0_3_0_gate: residual-risks.json "

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "release_v0_3_0_gate: residual-risks.json validator usage error")
		os.Exit(2)
	}

	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "release_v0_3_0_gate: residual-risks.json read failed: %v\n", err)
		os.Exit(1)
	}

	var root map[string]json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&root); err != nil {
		fmt.Fprintf(os.Stderr, "release_v0_3_0_gate: residual-risks.json malformed JSON: %v\n", err)
		os.Exit(1)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		fmt.Fprintln(os.Stderr, "release_v0_3_0_gate: residual-risks.json malformed JSON: trailing data")
		os.Exit(1)
	}
	if root == nil {
		fmt.Fprintln(
			os.Stderr,
			residualRiskPrefix+"invalid JSON structure: top-level object required",
		)
		os.Exit(1)
	}

	var schema string
	if rawSchema, ok := root["schema"]; !ok {
		fmt.Fprintln(
			os.Stderr,
			residualRiskPrefix+"missing schema tetra.release.residual-risks.v1",
		)
		os.Exit(1)
	} else if err := json.Unmarshal(rawSchema, &schema); err != nil ||
		schema != "tetra.release.residual-risks.v1" {
		fmt.Fprintln(
			os.Stderr,
			residualRiskPrefix+"missing schema tetra.release.residual-risks.v1",
		)
		os.Exit(1)
	}

	expectedReleaseVersion := os.Args[2]
	var releaseVersion string
	if rawReleaseVersion, ok := root["release_version"]; !ok {
		fmt.Fprintf(
			os.Stderr,
			residualRiskPrefix+"missing release_version %s\n",
			expectedReleaseVersion,
		)
		os.Exit(1)
	} else if err := json.Unmarshal(rawReleaseVersion, &releaseVersion); err != nil ||
		releaseVersion != expectedReleaseVersion {
		fmt.Fprintf(
			os.Stderr,
			residualRiskPrefix+"release_version must be %s, got %q\n",
			expectedReleaseVersion,
			releaseVersion,
		)
		os.Exit(1)
	}

	rawRisks, ok := root["risks"]
	if !ok {
		fmt.Fprintln(os.Stderr, "release_v0_3_0_gate: residual-risks.json missing risks array")
		os.Exit(1)
	}
	if trimmedRisks := bytes.TrimSpace(rawRisks); len(trimmedRisks) == 0 || trimmedRisks[0] != '[' {
		fmt.Fprintln(
			os.Stderr,
			residualRiskPrefix+"invalid JSON structure: risks array required",
		)
		os.Exit(1)
	}
	var risks []json.RawMessage
	if err := json.Unmarshal(rawRisks, &risks); err != nil {
		fmt.Fprintln(
			os.Stderr,
			residualRiskPrefix+"invalid JSON structure: risks array required",
		)
		os.Exit(1)
	}

	failed := false
	for idx, rawRisk := range risks {
		var risk map[string]json.RawMessage
		if err := json.Unmarshal(rawRisk, &risk); err != nil || risk == nil {
			fmt.Fprintf(
				os.Stderr,
				residualRiskPrefix+"malformed risk object at index %d\n",
				idx,
			)
			failed = true
			continue
		}

		id := optionalString(risk, "id")
		if id == "" {
			id = fmt.Sprintf("<index %d>", idx)
		}

		for _, field := range []string{"id", "severity", "owner", "status"} {
			if _, ok := risk[field]; !ok {
				fmt.Fprintf(
					os.Stderr,
					residualRiskPrefix+"risk %s missing required field %s\n",
					id,
					field,
				)
				failed = true
				continue
			}
			if _, ok := requiredString(risk, field, id); !ok {
				failed = true
			}
		}

		severity, ok := requiredString(risk, "severity", id)
		if !ok {
			failed = true
			continue
		}
		severity = strings.ToLower(strings.TrimSpace(severity))
		if severity == "" {
			fmt.Fprintf(os.Stderr, "release_v0_3_0_gate: residual-risks.json risk %s missing severity\n", id)
			failed = true
			continue
		}
		if !knownSeverity(severity) {
			fmt.Fprintf(
				os.Stderr,
				residualRiskPrefix+"risk %s has unknown severity %s\n",
				id,
				severity,
			)
			failed = true
			continue
		}

		owner := optionalString(risk, "owner")
		status := optionalString(risk, "status")
		requiresKnownOwner := severity == "high" ||
			severity == "medium" ||
			severity == "critical"
		if requiresKnownOwner && (missingOrUnknown(owner) || missingOrUnknown(status)) {
			fmt.Fprintf(
				os.Stderr,
				residualRiskPrefix+
					"%s residual risk %s requires known status and owner "+
					"(owner=%s, status=%s)\n",
				severity,
				id,
				owner,
				status,
			)
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
}

func optionalString(risk map[string]json.RawMessage, field string) string {
	value, ok := risk[field]
	if !ok {
		return ""
	}
	var out string
	if err := json.Unmarshal(value, &out); err != nil {
		return ""
	}
	return out
}

func requiredString(risk map[string]json.RawMessage, field, id string) (string, bool) {
	value, ok := risk[field]
	if !ok {
		return "", false
	}
	var out string
	if err := json.Unmarshal(value, &out); err != nil {
		fmt.Fprintf(
			os.Stderr,
			residualRiskPrefix+"risk %s field %s must be a string\n",
			id,
			field,
		)
		return "", false
	}
	return out, true
}

func knownSeverity(severity string) bool {
	switch severity {
	case "none", "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func missingOrUnknown(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "unknown", "unowned", "todo", "tbd":
		return true
	default:
		return false
	}
}
GO
  go run "$validator_go" "$residual_risks_json" "$release_version"
}

validate_runtime_smoke_report_source() {
  local target="$1"
  local source_path="$2"
  local expected_git_head="$3"

  if [[ -z "$source_path" ]]; then
    echo "release_v0_3_0_gate: missing runtime execution evidence for $target" >&2
    echo "release_v0_3_0_gate: set TETRA_MACOS_RUNTIME_SMOKE_REPORT and" \
      "TETRA_WINDOWS_RUNTIME_SMOKE_REPORT to host-gated --run=true smoke reports" >&2
    return 1
  fi
  if [[ ! -f "$source_path" ]]; then
    echo "release_v0_3_0_gate: runtime execution evidence for $target does not exist:" \
      "$source_path" >&2
    return 1
  fi

  go run ./tools/cmd/smoke-report-to-checklist \
    --validate-only \
    --report "$source_path" || return
  validate_runtime_smoke_report "$source_path" "$target" "$release_version" "$expected_git_head"
}

validate_runtime_smoke_report() {
  local report_path="$1"
  local expected_target="$2"
  local expected_version="$3"
  local expected_git_head="$4"
  local python_bin

  if ! python_bin="$(resolve_python)"; then
    return 1
  fi
  "$python_bin" - "$report_path" "$expected_target" "$expected_version" "$expected_git_head" << 'PY'
import json
import re
import sys
from datetime import datetime

report_path = sys.argv[1]
expected_target = sys.argv[2]
expected_version = sys.argv[3]
expected_git_head = sys.argv[4]
with open(report_path, "r", encoding="utf-8") as fh:
    report = json.load(fh)

def fail(message):
    print(f"release_v0_3_0_gate: runtime smoke evidence invalid: {message}", file=sys.stderr)
    sys.exit(1)

timestamp = report.get("timestamp")
try:
    if not isinstance(timestamp, str):
        raise ValueError("timestamp must be a string")
    timestamp_pattern = (
        r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}"
        r"(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})"
    )
    if not re.fullmatch(timestamp_pattern, timestamp):
        raise ValueError("timestamp must use RFC3339 separators and timezone")
    parsed_timestamp = datetime.fromisoformat(timestamp.replace("Z", "+00:00"))
    if parsed_timestamp.tzinfo is None:
        raise ValueError("timestamp must include a timezone")
except ValueError:
    fail(f"timestamp is not RFC3339: {timestamp!r}")

target = report.get("target")
if target != expected_target:
    fail(f"target is {target!r}, want {expected_target!r}")
host = report.get("host")
if host != expected_target:
    fail(f"host is {host!r}, want {expected_target!r}")
version = report.get("version")
if version != expected_version:
    fail(f"version is {version!r}, want {expected_version!r}")
git_head = report.get("git_head")
if git_head != expected_git_head:
    fail(f"git_head is {git_head!r}, want {expected_git_head!r}")
runner = report.get("runner", "")
if not isinstance(runner, str):
    fail("runner must be a string")
if runner:
    fail(f"runner is {runner!r}, want empty host-native runtime")
if not isinstance(report.get("islands_debug"), bool):
    fail("islands_debug must be a boolean")
build_only = report.get("build_only", False)
if not isinstance(build_only, bool):
    fail("build_only must be a boolean")
if build_only:
    fail(f"{expected_target} report is build_only")
cases = report.get("cases")
if not isinstance(cases, list) or not cases:
    fail("cases must be a non-empty array")
total = report.get("total")
passed = report.get("passed")
failed = report.get("failed")
if not isinstance(total, int) or isinstance(total, bool):
    fail("total must be an integer")
if not isinstance(passed, int) or isinstance(passed, bool):
    fail("passed must be an integer")
if not isinstance(failed, int) or isinstance(failed, bool):
    fail("failed must be an integer")
if total != len(cases) or passed != len(cases) or failed != 0:
    fail(
        f"counts got total={total!r} passed={passed!r} failed={failed!r}, "
        f"want total=passed={len(cases)} failed=0"
    )
required_cases = [
    "actors_pingpong",
    "actor_sleep_pingpong",
    "task_smoke",
    "time_sleep_smoke",
    "task_sleep_deadline_smoke",
    "task_join_wait_smoke",
    "deadline_aware_waits_smoke",
    "wait_composition_smoke",
]
case_by_name = {}
for case in cases:
    name = case.get("name", "<missing>")
    if not isinstance(name, str):
        fail("case name must be a string")
    if not str(name).strip():
        fail("contains a case with empty name")
    if name in case_by_name:
        fail(f"duplicate runtime case {name}")
    case_by_name[name] = case
    if case.get("unsupported", False):
        fail(f"case {name} is marked unsupported")
    if not isinstance(case.get("pass"), bool):
        fail(f"case {name} pass must be a boolean")
    if not isinstance(case.get("ran"), bool):
        fail(f"case {name} ran must be a boolean")
    if not case.get("pass", False):
        fail(f"case {name} did not pass")
    if not case.get("ran", False):
        fail(f"case {name} did not run")
    if "actual_exit" not in case:
        fail(f"case {name} is missing actual_exit")
    expected_exit = case.get("expected_exit")
    actual_exit = case.get("actual_exit")
    if not isinstance(expected_exit, int) or isinstance(expected_exit, bool):
        fail(f"case {name} expected_exit must be an integer")
    if not isinstance(actual_exit, int) or isinstance(actual_exit, bool):
        fail(f"case {name} actual_exit must be an integer")
    if actual_exit != expected_exit:
        fail(
            f"case {name} actual_exit={actual_exit!r}, "
            f"want expected_exit={expected_exit!r}"
        )
    if str(case.get("error", "")).strip():
        fail(f"case {name} has error text")
for required in required_cases:
    if required not in case_by_name:
        fail(f"missing required runtime case {required}")
PY
}

check_runtime_execution_evidence() {
  local current_git_head
  if ! current_git_head="$(git rev-parse --short HEAD 2> /dev/null)"; then
    echo "release_v0_3_0_gate: unable to determine current Git head for" \
      "runtime execution evidence" >&2
    return 1
  fi
  validate_runtime_smoke_report_source \
    "macos-x64" \
    "${TETRA_MACOS_RUNTIME_SMOKE_REPORT:-}" \
    "$current_git_head" || return
  validate_runtime_smoke_report_source \
    "windows-x64" \
    "${TETRA_WINDOWS_RUNTIME_SMOKE_REPORT:-}" \
    "$current_git_head" || return
  local runtime_tmp_dir="$tmp_dir/runtime-smoke-artifacts"
  local macos_tmp="$runtime_tmp_dir/macos-runtime-smoke.json"
  local windows_tmp="$runtime_tmp_dir/windows-runtime-smoke.json"
  local runtime_artifacts=(
    "$macos_tmp"
    "$windows_tmp"
    "$artifacts_dir/macos-runtime-smoke.json"
    "$artifacts_dir/windows-runtime-smoke.json"
  )
  mkdir -p "$runtime_tmp_dir"
  rm -f "${runtime_artifacts[@]}"
  cp -- "${TETRA_MACOS_RUNTIME_SMOKE_REPORT:-}" "$macos_tmp" || return
  cp -- "${TETRA_WINDOWS_RUNTIME_SMOKE_REPORT:-}" "$windows_tmp" || {
    rm -f "${runtime_artifacts[@]}"
    return 1
  }
  mv "$macos_tmp" "$artifacts_dir/macos-runtime-smoke.json" || return
  mv "$windows_tmp" "$artifacts_dir/windows-runtime-smoke.json" || {
    rm -f "$artifacts_dir/macos-runtime-smoke.json" "$artifacts_dir/windows-runtime-smoke.json"
    return 1
  }
}

check_security_review_signoff() {
  local signoff_path="${TETRA_SECURITY_REVIEW_SIGNOFF:-}"
  if [[ -z "$signoff_path" ]]; then
    if [[ "$require_clean" -eq 0 \
      && "${TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF:-}" == "1" ]]; then
      cat > "$artifacts_dir/security-review.md" << EOF
# $release_version Security Review CI Placeholder

Decision: blocked: missing human security signoff
CI status: missing-security-signoff
Report directory: $report_dir
Generated at: $(date -u +%Y-%m-%dT%H:%M:%SZ)

This artifact records the CI release-gate contract:

- TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF=1
- TETRA_SECURITY_REVIEW_SIGNOFF was not set
- This is not a full release evidence pass.
- CI release-gate runs may collect evidence without hard-blocking on the
  missing human security signoff.
- Tag-ready release promotion must provide TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md>.
EOF
      ci_missing_security_signoff=1
      echo "release_v0_3_0_gate: CI mode recorded missing security signoff;" \
        "release evidence remains incomplete"
      return 0
    fi
    echo "release_v0_3_0_gate: missing TETRA_SECURITY_REVIEW_SIGNOFF=<security-review.md>" >&2
    echo "release_v0_3_0_gate: create a signoff with:" \
      "bash scripts/release/v0_3_0/security-review.sh" \
      "--write-template <security-review.md>" >&2
    return 1
  fi
  if [[ ! -f "$signoff_path" ]]; then
    echo "release_v0_3_0_gate: missing security review signoff artifact: $signoff_path" >&2
    return 1
  fi
  staged_security_review_signoff="$tmp_dir/security-review-source.md"
  cp -- "$signoff_path" "$staged_security_review_signoff" || return
  echo "release_v0_3_0_gate: staged security signoff for final same-run artifact validation"
}

security_review_report_dir() {
  local path="$1"
  sed -nE 's/^Report directory:[[:space:]]*(.+)$/\1/p' "$path" | head -n 1
}

canonical_dir() {
  local path="$1"
  (cd "$path" && pwd -P)
}

write_final_security_review_signoff() {
  if [[ -z "$staged_security_review_signoff" ]]; then
    return 0
  fi

  local signoff_report_dir
  signoff_report_dir="$(security_review_report_dir "$staged_security_review_signoff")"
  if [[ -z "$signoff_report_dir" ]]; then
    cp -- "$staged_security_review_signoff" "$artifacts_dir/security-review.md"
    bash scripts/release/v0_3_0/security-review.sh --signoff "$artifacts_dir/security-review.md"
    return
  fi

  local canonical_signoff_report_dir
  local canonical_report_dir
  if ! canonical_signoff_report_dir="$(canonical_dir "$signoff_report_dir")"; then
    echo "release_v0_3_0_gate: security signoff Report directory does not exist:" \
      "$signoff_report_dir" >&2
    return 1
  fi
  canonical_report_dir="$(canonical_dir "$report_dir")"
  if [[ "$canonical_signoff_report_dir" != "$canonical_report_dir" ]]; then
    echo "release_v0_3_0_gate: security signoff Report directory mismatch:" \
      "got '$signoff_report_dir', want '$report_dir'" >&2
    return 1
  fi

  local summary_hash
  local artifact_hash
  local release_state_hash
  summary_hash="$(sha256_file "$summary_json")"
  artifact_hash="$(sha256_file "$report_dir/artifact-hashes.json")"
  release_state_hash="$(sha256_file "$artifacts_dir/release-state.json")"

  local final_tmp="$tmp_dir/security-review-final.md"
  if ! awk \
    -v summary_hash="$summary_hash" \
    -v artifact_hash="$artifact_hash" \
    -v release_state_hash="$release_state_hash" '
      BEGIN {
        in_hashes = 0
        inserted = 0
      }
      /^## Artifact Hashes$/ {
        print
        print ""
        print "- summary.json: " summary_hash
        print "- artifact-hashes.json: " artifact_hash
        print "- artifacts/release-state.json: " release_state_hash
        print ""
        print "## Archived Artifacts"
        print ""
        print "- artifacts/security-review.md is generated by " \
          "scripts/release/v0_3_0/gate.sh from " \
          "TETRA_SECURITY_REVIEW_SIGNOFF after same-run canonical artifacts exist."
        in_hashes = 1
        inserted = 1
        next
      }
      in_hashes && /^## / {
        in_hashes = 0
      }
      in_hashes {
        next
      }
      {
        print
      }
      END {
        if (!inserted) {
          exit 3
        }
      }
    ' "$staged_security_review_signoff" > "$final_tmp"; then
    echo "release_v0_3_0_gate: security signoff missing ## Artifact Hashes section" >&2
    return 1
  fi

  cp -- "$final_tmp" "$artifacts_dir/security-review.md"
  bash scripts/release/v0_3_0/security-review.sh --signoff "$artifacts_dir/security-review.md"
}

write_security_review_detached_hash() {
  local review_path="$artifacts_dir/security-review.md"
  local detached_hash_path="$artifacts_dir/security-review.md.sha256"
  local review_hash
  if [[ ! -f "$review_path" ]]; then
    echo "release_v0_3_0_gate: cannot attest missing final security review: $review_path" >&2
    return 1
  fi
  if ! review_hash="$(sha256_file "$review_path")"; then
    echo "release_v0_3_0_gate: cannot hash final security review: $review_path" >&2
    return 1
  fi
  printf '%s  artifacts/security-review.md\n' "$review_hash" > "$detached_hash_path"
}

echo "release_v0_3_0_gate: bootstrapping local binaries before version preflight" >&2
bash scripts/dev/bootstrap.sh >&2

version="$(./tetra version 2> /dev/null || true)"
if [[ "$version" != "$release_version" ]]; then
  echo "release_v0_3_0_gate: blocked: expected ./tetra version to be" \
    "$release_version, got '${version:-<missing>}'" >&2
  echo "release_v0_3_0_gate: keep the repository on the current release line" \
    "until v0.3.0 evidence is complete." >&2
  write_summary "blocked"
  validate_summary
  exit 1
fi

echo "release_v0_3_0_gate: artifact mapping $release_artifact" >&2
run_step "version preflight ($release_version required)" check_release_version
run_step "short alias version parity" check_short_alias_version
run_step "go test packages" check_go_test_packages
run_step "stabilization wrapper" env TETRA_TEST_ALL_RELEASE_VERSION="$release_version" bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir "$artifacts_dir/test-all"
run_step "short fuzz smoke" bash scripts/dev/fuzz-nightly.sh --short --out-dir "$artifacts_dir/fuzz-short"
run_step "fuzz artifact validation" check_short_fuzz_summary
run_step "unstable seed triage" check_unstable_seed_triage
run_step "docs verification" go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
run_step "manifest validation" \
  go run ./tools/cmd/validate-manifest \
  --manifest docs/generated/manifest.json
run_step "security review signoff" check_security_review_signoff
run_step "residual risk validation" check_residual_risks
run_step "macOS and Windows runtime execution evidence" check_runtime_execution_evidence
run_step "working tree whitespace audit" git diff --check
if [[ "$failed_count" -eq 0 && "$ci_missing_security_signoff" -eq 0 ]]; then
  write_summary "pass"
else
  write_summary "blocked"
fi
validate_summary
if [[ "$failed_count" -eq 0 && "$ci_missing_security_signoff" -eq 1 ]]; then
  if write_security_review_detached_hash; then
    :
  else
    rc="$?"
    record_failed_step \
      "CI missing-signoff detached security review hash" \
      "write_security_review_detached_hash" \
      "$rc" \
      "CI missing-security-signoff detached security review hash generation failed"
    write_summary "blocked"
    validate_summary
    check_release_state || true
    echo "release_v0_3_0_gate: blocked: CI missing-security-signoff detached" \
      "security review hash generation failed" >&2
    echo "summary: $summary_md" >&2
    echo "json: $summary_json" >&2
    exit 1
  fi
  run_step "artifact hash manifest" check_artifact_hash_manifest
  if [[ "$failed_count" -gt 0 ]]; then
    write_summary "blocked"
    validate_summary
    echo "release_v0_3_0_gate: blocked: $failed_count step(s) failed" >&2
    echo "summary: $summary_md" >&2
    echo "json: $summary_json" >&2
    exit 1
  fi
  write_summary "blocked"
  validate_summary
  if check_release_state; then
    :
  else
    rc="$?"
    record_failed_step \
      "CI missing-signoff release-state artifact refresh" \
      "check_release_state" \
      "$rc" \
      "CI missing-security-signoff release-state artifact generation failed"
    write_summary "blocked"
    validate_summary
    echo "release_v0_3_0_gate: blocked: release-state artifact generation failed" >&2
    echo "summary: $summary_md" >&2
    echo "json: $summary_json" >&2
    exit 1
  fi
  if check_artifact_hash_manifest; then
    :
  else
    rc="$?"
    record_failed_step \
      "CI missing-signoff artifact hash refresh" \
      "check_artifact_hash_manifest" \
      "$rc" \
      "CI missing-security-signoff artifact hash refresh failed"
    write_summary "blocked"
    validate_summary
    check_release_state || true
    echo "release_v0_3_0_gate: blocked: CI missing-security-signoff" \
      "artifact hash refresh failed" >&2
    echo "summary: $summary_md" >&2
    echo "json: $summary_json" >&2
    exit 1
  fi
  echo "release_v0_3_0_gate: CI missing security signoff recorded;" \
    "not a full release evidence pass"
  echo "summary: $summary_md"
  echo "json: $summary_json"
  exit 0
fi
run_step "release state audit" check_release_state
run_step "artifact hash manifest" check_artifact_hash_manifest

if [[ "$failed_count" -gt 0 ]]; then
  write_summary "blocked"
  validate_summary
  check_release_state || true
  echo "release_v0_3_0_gate: blocked: $failed_count step(s) failed" >&2
  echo "summary: $summary_md" >&2
  echo "json: $summary_json" >&2
  exit 1
fi

write_summary "pass"
validate_summary
if check_release_state; then
  :
else
  rc="$?"
  record_failed_step \
    "final release-state artifact refresh" \
    "check_release_state" \
    "$rc" \
    "final release-state artifact generation failed"
  write_summary "blocked"
  validate_summary
  echo "release_v0_3_0_gate: blocked: final release-state artifact generation failed" >&2
  echo "summary: $summary_md" >&2
  echo "json: $summary_json" >&2
  exit 1
fi
if check_artifact_hash_manifest; then
  :
else
  rc="$?"
  record_failed_step \
    "final artifact hash manifest refresh" \
    "check_artifact_hash_manifest" \
    "$rc" \
    "final release artifact hash validation failed"
  write_summary "blocked"
  validate_summary
  check_release_state || true
  echo "release_v0_3_0_gate: blocked: final release artifact hash validation failed" >&2
  echo "summary: $summary_md" >&2
  echo "json: $summary_json" >&2
  exit 1
fi
if write_final_security_review_signoff; then
  :
else
  rc="$?"
  record_failed_step \
    "final security signoff validation" \
    "write_final_security_review_signoff" \
    "$rc" \
    "final security signoff validation failed"
  write_summary "blocked"
  validate_summary
  check_release_state || true
  echo "release_v0_3_0_gate: blocked: final security signoff validation failed" >&2
  echo "summary: $summary_md" >&2
  echo "json: $summary_json" >&2
  exit 1
fi
if write_security_review_detached_hash; then
  :
else
  rc="$?"
  record_failed_step \
    "detached security review hash generation" \
    "write_security_review_detached_hash" \
    "$rc" \
    "detached security review hash generation failed"
  write_summary "blocked"
  validate_summary
  check_release_state || true
  echo "release_v0_3_0_gate: blocked: detached security review hash generation failed" >&2
  echo "summary: $summary_md" >&2
  echo "json: $summary_json" >&2
  exit 1
fi
if check_artifact_hash_manifest; then
  :
else
  rc="$?"
  record_failed_step \
    "final artifact hash manifest validation" \
    "check_artifact_hash_manifest" \
    "$rc" \
    "final release artifact hash validation failed"
  write_summary "blocked"
  validate_summary
  check_release_state || true
  echo "release_v0_3_0_gate: blocked: final release artifact hash validation failed" >&2
  echo "summary: $summary_md" >&2
  echo "json: $summary_json" >&2
  exit 1
fi
echo "release_v0_3_0_gate: passed"
echo "summary: $summary_md"
echo "json: $summary_json"
