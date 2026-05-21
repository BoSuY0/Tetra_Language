#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../.." && pwd)"
repo=""
branch=""
workflow="ci"
report_path="$repo_root/reports/full-platform-ui-runtime/actions-availability.json"
limit=10
remote_url=""
billing_owner=""

usage() {
  cat <<'USAGE'
Usage: bash scripts/release/full_platform/actions-availability-preflight.sh [--repo OWNER/REPO] [--branch BRANCH] [--report FILE]

Writes and validates a GitHub Actions availability preflight report. This proves
only that Actions can start jobs and expose logs; it is not runtime evidence and
never replaces Windows/macOS target-host UI runtime reports.
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
  echo "error: gh is required for GitHub Actions availability preflight" >&2
  exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "error: jq is required for GitHub Actions availability preflight" >&2
  exit 2
fi

mkdir -p "$(dirname "$report_path")"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

runs_path="$tmp_dir/runs.json"
run_path="$tmp_dir/run.json"
run_details_path="$tmp_dir/run-details.json"
permissions_path="$tmp_dir/permissions.json"
runners_path="$tmp_dir/runners.json"
jobs_path="$tmp_dir/jobs.json"
check_suite_path="$tmp_dir/check-suite.json"
billing_path="$tmp_dir/billing.json"
billing_err_path="$tmp_dir/billing.err"
workflows_path="$tmp_dir/workflows.json"
logs_path="$tmp_dir/logs.zip"

gh run list \
  --repo "$repo" \
  --branch "$branch" \
  --limit "$limit" \
  --json databaseId,event,status,conclusion,headSha,workflowName \
  >"$runs_path"

jq -e \
  --arg workflow "$workflow" \
  '[.[] | select(.workflowName == $workflow or .workflowName == "" or .workflowName == null)][0] // {
    databaseId: 0,
    event: "",
    status: "",
    conclusion: "",
    headSha: "",
    workflowName: ""
  }' \
  "$runs_path" >"$run_path"

run_id="$(jq -r '.databaseId // 0' "$run_path")"
run_event="$(jq -r '.event // ""' "$run_path")"
run_status="$(jq -r '.status // ""' "$run_path")"
run_conclusion="$(jq -r '.conclusion // ""' "$run_path")"
run_head_sha="$(jq -r '.headSha // ""' "$run_path")"
run_workflow_name="$(jq -r '.workflowName // ""' "$run_path")"
run_workflow_path=""
run_workflow_id=0
run_check_suite_id=0
run_check_suite_app=""
run_check_suite_status=""
run_check_suite_conclusion=""
run_check_suite_latest_check_runs_count=0
run_check_suite_head_sha=""
run_jobs=0
logs_available="false"

if [[ "$run_id" =~ ^[0-9]+$ && "$run_id" -gt 0 ]]; then
  if gh api "repos/$repo/actions/runs/$run_id" >"$run_details_path"; then
    run_workflow_name="$(jq -r '.name // ""' "$run_details_path")"
    run_workflow_path="$(jq -r '.path // ""' "$run_details_path")"
    run_workflow_id="$(jq -r '.workflow_id // 0' "$run_details_path")"
    run_check_suite_id="$(jq -r '.check_suite_id // 0' "$run_details_path")"
  fi
  if [[ "$run_check_suite_id" =~ ^[0-9]+$ && "$run_check_suite_id" -gt 0 ]] &&
     gh api "repos/$repo/check-suites/$run_check_suite_id" >"$check_suite_path"; then
    run_check_suite_app="$(jq -r '.app.slug // ""' "$check_suite_path")"
    run_check_suite_status="$(jq -r '.status // ""' "$check_suite_path")"
    run_check_suite_conclusion="$(jq -r '.conclusion // ""' "$check_suite_path")"
    run_check_suite_latest_check_runs_count="$(jq -r '.latest_check_runs_count // 0' "$check_suite_path")"
    run_check_suite_head_sha="$(jq -r '.head_sha // ""' "$check_suite_path")"
  fi
  if gh api "repos/$repo/actions/runs/$run_id/jobs" >"$jobs_path"; then
    run_jobs="$(jq -r '.total_count // 0' "$jobs_path")"
  fi
  if gh api "repos/$repo/actions/runs/$run_id/logs" >"$logs_path" 2>/dev/null; then
    logs_available="true"
  fi
fi

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

workflows_total_count=0
workflows_active_count=0
workflows_entries='[]'
if gh api "repos/$repo/actions/workflows" >"$workflows_path"; then
  workflows_total_count="$(jq -r '.total_count // 0' "$workflows_path")"
  workflows_active_count="$(jq -r '[.workflows[]? | select(.state == "active")] | length' "$workflows_path")"
  workflows_entries="$(jq -c '[.workflows[]? | {id, name, path, state}]' "$workflows_path")"
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

availability_status="blocked"
summary="GitHub Actions did not prove job-backed availability."
if [[ "$repo_actions_enabled" == "true" &&
      "$run_status" == "completed" &&
      "$run_conclusion" == "success" &&
      "$run_jobs" -gt 0 &&
      "$logs_available" == "true" &&
      "$run_workflow_path" != "BuildFailed" &&
      "$run_workflow_id" -gt 0 &&
      "$run_check_suite_id" -gt 0 &&
      "$run_check_suite_app" == "github-actions" &&
      "$run_check_suite_status" == "completed" &&
      "$run_check_suite_conclusion" == "success" &&
      "$run_check_suite_latest_check_runs_count" -gt 0 &&
      "$billing_actions_status" != "unavailable_missing_user_scope" ]]; then
  availability_status="pass"
  summary="GitHub Actions can start jobs and expose logs."
fi

jq -n \
  --arg repo "$repo" \
  --arg branch "$branch" \
  --arg workflow "$workflow" \
  --arg status "$availability_status" \
  --arg summary "$summary" \
  --argjson repoActionsEnabled "$repo_actions_enabled" \
  --arg repoAllowedActions "$repo_allowed_actions" \
  --argjson selfHostedRunnerCount "$self_hosted_runner_count" \
  --arg billingActionsStatus "$billing_actions_status" \
  --arg billingActionsDetail "$billing_actions_detail" \
  --argjson workflowsTotalCount "$workflows_total_count" \
  --argjson workflowsActiveCount "$workflows_active_count" \
  --argjson workflowsEntries "$workflows_entries" \
  --argjson runID "$run_id" \
  --arg runEvent "$run_event" \
  --arg runStatus "$run_status" \
  --arg runConclusion "$run_conclusion" \
  --arg runHeadSHA "$run_head_sha" \
  --arg runWorkflowName "$run_workflow_name" \
  --arg runWorkflowPath "$run_workflow_path" \
  --argjson runWorkflowID "$run_workflow_id" \
  --argjson runCheckSuiteID "$run_check_suite_id" \
  --arg runCheckSuiteApp "$run_check_suite_app" \
  --arg runCheckSuiteStatus "$run_check_suite_status" \
  --arg runCheckSuiteConclusion "$run_check_suite_conclusion" \
  --argjson runCheckSuiteLatestCheckRunsCount "$run_check_suite_latest_check_runs_count" \
  --arg runCheckSuiteHeadSHA "$run_check_suite_head_sha" \
  --argjson runJobs "$run_jobs" \
  --argjson logsAvailable "$logs_available" \
  '{
    schema: "tetra.actions.availability.v1",
    status: $status,
    repo: $repo,
    branch: $branch,
    workflow: $workflow,
    summary: $summary,
    production_evidence: false,
    repo_actions_enabled: $repoActionsEnabled,
    repo_allowed_actions: $repoAllowedActions,
    self_hosted_runner_count: $selfHostedRunnerCount,
    billing_actions_status: $billingActionsStatus,
    billing_actions_detail: $billingActionsDetail,
    workflows: {
      total_count: $workflowsTotalCount,
      active_count: $workflowsActiveCount,
      entries: $workflowsEntries
    },
    run: {
      id: $runID,
      event: $runEvent,
      status: $runStatus,
      conclusion: $runConclusion,
      head_sha: $runHeadSHA,
      workflow_name: $runWorkflowName,
      workflow_path: $runWorkflowPath,
      workflow_id: $runWorkflowID,
      check_suite_id: $runCheckSuiteID,
      check_suite: {
        id: $runCheckSuiteID,
        app: $runCheckSuiteApp,
        status: $runCheckSuiteStatus,
        conclusion: $runCheckSuiteConclusion,
        latest_check_runs_count: $runCheckSuiteLatestCheckRunsCount,
        head_sha: $runCheckSuiteHeadSHA
      },
      jobs: $runJobs,
      logs_available: $logsAvailable
    },
    next_action: "Collect target-host Windows/macOS UI runtime reports after Actions availability passes; this is not runtime evidence."
  }' >"$report_path"

go run ./tools/cmd/validate-actions-availability --report "$report_path"
echo "GitHub Actions availability preflight report: $report_path"
