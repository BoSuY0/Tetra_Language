#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
repo=""
branch=""
workflow="ci"
report_path="$repo_root/reports/full-platform-ui-runtime/github-actions-startup-blocker.json"
limit=10
remote_url=""

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/full_platform/github-actions-startup-diagnostic.sh [--repo OWNER/REPO] [--branch BRANCH] [--report FILE]

Writes a diagnostic-only blocker report when GitHub Actions runs are created
but fail before any job starts. This report never counts as production runtime
evidence; use manual or self-hosted target-host Windows/macOS reports instead.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      if [[ $# -lt 2 ]]; then
        echo "error: --repo requires a value" >&2
        usage >&2
        exit 2
      fi
      repo="$2"
      shift 2
      ;;
    --branch)
      if [[ $# -lt 2 ]]; then
        echo "error: --branch requires a value" >&2
        usage >&2
        exit 2
      fi
      branch="$2"
      shift 2
      ;;
    --workflow)
      if [[ $# -lt 2 ]]; then
        echo "error: --workflow requires a value" >&2
        usage >&2
        exit 2
      fi
      workflow="$2"
      shift 2
      ;;
    --report)
      if [[ $# -lt 2 ]]; then
        echo "error: --report requires a value" >&2
        usage >&2
        exit 2
      fi
      report_path="$2"
      shift 2
      ;;
    --limit)
      if [[ $# -lt 2 ]]; then
        echo "error: --limit requires a value" >&2
        usage >&2
        exit 2
      fi
      limit="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

cd "$repo_root"

if [[ -z "$repo" ]]; then
  remote_url="$(git remote get-url origin 2>/dev/null || true)"
  repo="$(printf '%s' "$remote_url" | sed -E 's#^git@github.com:##; s#^https://github.com/##; s#\\.git$##')"
fi
if [[ -z "$branch" ]]; then
  branch="$(git rev-parse --abbrev-ref HEAD)"
fi
if [[ -z "$repo" || "$repo" == "$remote_url" ]]; then
  echo "error: could not infer GitHub repo; pass --repo OWNER/REPO" >&2
  exit 2
fi
if ! command -v gh >/dev/null 2>&1; then
  echo "error: gh is required for GitHub Actions startup diagnostics" >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "error: jq is required for GitHub Actions startup diagnostics" >&2
  exit 2
fi

mkdir -p "$(dirname "$report_path")"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
runs_path="$tmp_dir/runs.json"
blocked_runs_path="$tmp_dir/blocked-runs.json"

gh run list \
  --repo "$repo" \
  --branch "$branch" \
  --limit "$limit" \
  --json databaseId,event,conclusion,headSha,workflowName \
  >"$runs_path"

jq '[.[] | select(.conclusion == "startup_failure")]' "$runs_path" >"$blocked_runs_path"

jq -n \
  --arg repo "$repo" \
  --arg branch "$branch" \
  --arg workflow "$workflow" \
  --slurpfile runs "$blocked_runs_path" \
  '{
    schema: "tetra.actions.startup-blocker.v1",
    status: "blocked",
    repo: $repo,
    branch: $branch,
    workflow: $workflow,
    summary: "GitHub Actions created runs but no jobs or logs were available; this is diagnostic only and not runtime evidence.",
    runs: ($runs[0] | map({
      id: .databaseId,
      event: .event,
      conclusion: .conclusion,
      head_sha: .headSha,
      jobs: 0,
      logs_available: false
    })),
    next_action: "Use manual or self-hosted target-host Windows/macOS reports; do not count startup_failure as runtime evidence."
  }' >"$report_path"

go run ./tools/cmd/validate-actions-startup-blocker --report "$report_path"
echo "GitHub Actions startup blocker report: $report_path"
