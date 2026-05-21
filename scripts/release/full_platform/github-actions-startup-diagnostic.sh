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
canary_branch=""
billing_owner=""

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/full_platform/github-actions-startup-diagnostic.sh [--repo OWNER/REPO] [--branch BRANCH] [--report FILE] [--canary-branch BRANCH]

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
    --canary-branch)
      if [[ $# -lt 2 ]]; then
        echo "error: --canary-branch requires a value" >&2
        usage >&2
        exit 2
      fi
      canary_branch="$2"
      shift 2
      ;;
    --billing-owner)
      if [[ $# -lt 2 ]]; then
        echo "error: --billing-owner requires a value" >&2
        usage >&2
        exit 2
      fi
      billing_owner="$2"
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
if [[ -z "$billing_owner" ]]; then
  billing_owner="${repo%%/*}"
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
permissions_path="$tmp_dir/permissions.json"
runners_path="$tmp_dir/runners.json"
billing_path="$tmp_dir/billing.json"
billing_err_path="$tmp_dir/billing.err"
canary_path="$tmp_dir/canary.json"

gh run list \
  --repo "$repo" \
  --branch "$branch" \
  --limit "$limit" \
  --json databaseId,event,conclusion,headSha,workflowName \
  >"$runs_path"

jq '[.[] | select(.conclusion == "startup_failure")]' "$runs_path" >"$blocked_runs_path"

repo_actions_enabled="false"
repo_allowed_actions="unknown"
if gh api "repos/$repo/actions/permissions" >"$permissions_path"; then
  repo_actions_enabled="$(jq -r '.enabled // false' "$permissions_path")"
  repo_allowed_actions="$(jq -r '.allowed_actions // "unknown"' "$permissions_path")"
fi

self_hosted_runner_count=0
if gh api "repos/$repo/actions/runners" >"$runners_path"; then
  self_hosted_runner_count="$(jq -r '.total_count // 0' "$runners_path")"
fi

billing_actions_status="unavailable"
billing_actions_detail="billing API unavailable"
if gh api "users/$billing_owner/settings/billing/actions" >"$billing_path" 2>"$billing_err_path"; then
  billing_actions_status="available"
  billing_actions_detail="$(jq -c '{total_minutes_used, included_minutes, minutes_used_breakdown}' "$billing_path")"
else
  billing_err="$(tr '\n' ' ' <"$billing_err_path" | sed -E 's/[[:space:]]+/ /g; s/^ //; s/ $//' | cut -c 1-240)"
  if grep -q '"user" scope' "$billing_err_path"; then
    billing_actions_status="unavailable_missing_user_scope"
    billing_actions_detail="requires gh auth refresh -h github.com -s user"
  elif [[ -n "$billing_err" ]]; then
    billing_actions_detail="$billing_err"
  fi
fi

printf 'null\n' >"$canary_path"
if [[ -n "$canary_branch" ]]; then
  canary_runs_path="$tmp_dir/canary-runs.json"
  gh run list \
    --repo "$repo" \
    --branch "$canary_branch" \
    --limit "$limit" \
    --json databaseId,event,conclusion,headSha,workflowName \
    >"$canary_runs_path"
  jq -e \
    --arg branch "$canary_branch" \
    '[.[] | select(.conclusion == "startup_failure")][0] | {
      branch: $branch,
      workflow: (.workflowName // ""),
      id: .databaseId,
      event: .event,
      conclusion: .conclusion,
      head_sha: .headSha,
      jobs: 0,
      logs_available: false
    }' \
    "$canary_runs_path" >"$canary_path"
fi

jq -n \
  --arg repo "$repo" \
  --arg branch "$branch" \
  --arg workflow "$workflow" \
  --argjson repoActionsEnabled "$repo_actions_enabled" \
  --arg repoAllowedActions "$repo_allowed_actions" \
  --argjson selfHostedRunnerCount "$self_hosted_runner_count" \
  --arg billingActionsStatus "$billing_actions_status" \
  --arg billingActionsDetail "$billing_actions_detail" \
  --slurpfile canary "$canary_path" \
  --slurpfile runs "$blocked_runs_path" \
  '{
    schema: "tetra.actions.startup-blocker.v1",
    status: "blocked",
    repo: $repo,
    branch: $branch,
    workflow: $workflow,
    summary: "GitHub Actions created runs but no jobs or logs were available; this is diagnostic only and not runtime evidence.",
    diagnostics: {
      repo_actions_enabled: $repoActionsEnabled,
      repo_allowed_actions: $repoAllowedActions,
      self_hosted_runner_count: $selfHostedRunnerCount,
      billing_actions_status: $billingActionsStatus,
      billing_actions_detail: $billingActionsDetail,
      minimal_canary: $canary[0]
    },
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
