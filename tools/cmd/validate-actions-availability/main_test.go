package main

import (
	"strings"
	"testing"
)

func TestValidateActionsAvailabilityAcceptsJobBackedSuccess(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.actions.availability.v1",
  "status": "pass",
  "repo": "BoSuY0/Tetra_Language",
  "branch": "codex/full-platform-ui-runtime",
  "workflow": "ci",
  "expected_git_head": "abcdef1234567890",
  "run_selection": "workflow_name",
  "summary": "GitHub Actions can start jobs and expose logs.",
  "production_evidence": false,
  "repo_actions_enabled": true,
  "repo_allowed_actions": "all",
  "self_hosted_runner_count": 0,
  "billing_actions_status": "available",
  "billing_actions_detail": "billing API available",
  "workflows": {
    "total_count": 1,
    "active_count": 1,
    "entries": [
      {
        "id": 220876851,
        "name": "ci",
        "path": ".github/workflows/ci.yml",
        "state": "active"
      }
    ]
  },
  "run": {
    "id": 26250000001,
    "event": "workflow_dispatch",
    "status": "completed",
    "conclusion": "success",
    "head_sha": "abcdef1234567890",
    "workflow_name": "ci",
    "workflow_path": ".github/workflows/ci.yml",
    "workflow_id": 220876851,
    "check_suite_id": 70197691298,
    "check_suite": {
      "id": 70197691298,
      "app": "github-actions",
      "status": "completed",
      "conclusion": "success",
      "latest_check_runs_count": 1,
      "head_sha": "abcdef1234567890"
    },
    "jobs": 1,
    "logs_available": true
  },
  "next_action": "Proceed to target-host Windows/macOS UI runtime reports; this is not runtime evidence."
}`)
	if err := validateActionsAvailability(raw); err != nil {
		t.Fatalf("validateActionsAvailability: %v", err)
	}
}

func TestValidateActionsAvailabilityRejectsZeroJobStartupFailure(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.actions.availability.v1",
  "status": "blocked",
  "repo": "BoSuY0/Tetra_Language",
  "branch": "codex/full-platform-ui-runtime",
  "workflow": "ci",
  "expected_git_head": "ecd8e2fcd06d26b4e79f603788ecb8842f641a32",
  "run_selection": "empty_workflow_fallback",
  "summary": "GitHub Actions cannot start jobs.",
  "production_evidence": false,
  "repo_actions_enabled": true,
  "repo_allowed_actions": "all",
  "self_hosted_runner_count": 0,
  "billing_actions_status": "unavailable_missing_user_scope",
  "billing_actions_detail": "requires gh auth refresh -h github.com -s user",
  "workflows": {
    "total_count": 2,
    "active_count": 1,
    "entries": [
      {
        "id": 220876851,
        "name": "ci",
        "path": ".github/workflows/ci.yml",
        "state": "active"
      },
      {
        "id": 220876857,
        "name": "",
        "path": "BuildFailed",
        "state": "deleted"
      }
    ]
  },
  "run": {
    "id": 26248635631,
    "event": "push",
    "status": "completed",
    "conclusion": "startup_failure",
    "head_sha": "ecd8e2fcd06d26b4e79f603788ecb8842f641a32",
    "workflow_name": "",
    "workflow_path": "BuildFailed",
    "workflow_id": 220876857,
    "check_suite_id": 70197691298,
    "check_suite": {
      "id": 70197691298,
      "app": "github-actions",
      "status": "completed",
      "conclusion": "startup_failure",
      "latest_check_runs_count": 0,
      "head_sha": "ecd8e2fcd06d26b4e79f603788ecb8842f641a32"
    },
    "jobs": 0,
    "logs_available": false
  },
  "next_action": "Refresh the GitHub CLI user scope or use self-hosted target-host runners; this is not runtime evidence."
}`)
	err := validateActionsAvailability(raw)
	if err == nil {
		t.Fatalf("expected startup_failure availability report to fail")
	}
	for _, want := range []string{
		"status",
		"success",
		"jobs",
		"logs_available",
		"billing_actions_status",
		"BuildFailed",
		"check_suite",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("availability error missing %q: %v", want, err)
		}
	}
}

func TestValidateActionsAvailabilityRejectsStaleWorkflowRun(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.actions.availability.v1",
  "status": "pass",
  "repo": "BoSuY0/Tetra_Language",
  "branch": "codex/full-platform-ui-runtime",
  "workflow": "ci",
  "expected_git_head": "current1234567890",
  "run_selection": "workflow_name_stale",
  "summary": "GitHub Actions can start jobs and expose logs.",
  "production_evidence": false,
  "repo_actions_enabled": true,
  "repo_allowed_actions": "all",
  "self_hosted_runner_count": 0,
  "billing_actions_status": "available",
  "billing_actions_detail": "billing API available",
  "workflows": {
    "total_count": 1,
    "active_count": 1,
    "entries": [
      {
        "id": 220876851,
        "name": "ci",
        "path": ".github/workflows/ci.yml",
        "state": "active"
      }
    ]
  },
  "run": {
    "id": 26250000001,
    "event": "workflow_dispatch",
    "status": "completed",
    "conclusion": "success",
    "head_sha": "stale1234567890",
    "workflow_name": "ci",
    "workflow_path": ".github/workflows/ci.yml",
    "workflow_id": 220876851,
    "check_suite_id": 70197691298,
    "check_suite": {
      "id": 70197691298,
      "app": "github-actions",
      "status": "completed",
      "conclusion": "success",
      "latest_check_runs_count": 1,
      "head_sha": "stale1234567890"
    },
    "jobs": 1,
    "logs_available": true
  },
  "next_action": "Proceed to target-host Windows/macOS UI runtime reports; this is not runtime evidence."
}`)
	err := validateActionsAvailability(raw)
	if err == nil {
		t.Fatalf("expected stale workflow run to fail")
	}
	for _, want := range []string{
		"run_selection",
		"workflow_name_stale",
		"run.head_sha",
		"run.check_suite.head_sha",
		"expected_git_head",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("availability error missing %q: %v", want, err)
		}
	}
}

func TestValidateActionsAvailabilityRejectsProductionEvidenceClaim(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.actions.availability.v1",
  "status": "pass",
  "repo": "BoSuY0/Tetra_Language",
  "branch": "codex/full-platform-ui-runtime",
  "workflow": "ci",
  "expected_git_head": "abcdef1234567890",
  "run_selection": "workflow_name",
  "summary": "READY production evidence",
  "production_evidence": true,
  "repo_actions_enabled": true,
  "repo_allowed_actions": "all",
  "self_hosted_runner_count": 0,
  "billing_actions_status": "available",
  "billing_actions_detail": "billing API available",
  "workflows": {
    "total_count": 1,
    "active_count": 1,
    "entries": [
      {
        "id": 220876851,
        "name": "ci",
        "path": ".github/workflows/ci.yml",
        "state": "active"
      }
    ]
  },
  "run": {
    "id": 26250000001,
    "event": "workflow_dispatch",
    "status": "completed",
    "conclusion": "success",
    "head_sha": "abcdef1234567890",
    "workflow_name": "ci",
    "workflow_path": ".github/workflows/ci.yml",
    "workflow_id": 220876851,
    "check_suite_id": 70197691298,
    "check_suite": {
      "id": 70197691298,
      "app": "github-actions",
      "status": "completed",
      "conclusion": "success",
      "latest_check_runs_count": 1,
      "head_sha": "abcdef1234567890"
    },
    "jobs": 1,
    "logs_available": true
  },
  "next_action": "READY"
}`)
	err := validateActionsAvailability(raw)
	if err == nil {
		t.Fatalf("expected production evidence claim to fail")
	}
	for _, want := range []string{"production_evidence", "READY"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("availability error missing %q: %v", want, err)
		}
	}
}
