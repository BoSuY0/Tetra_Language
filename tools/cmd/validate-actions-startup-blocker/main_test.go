package main

import (
	"strings"
	"testing"
)

func TestValidateStartupBlockerAcceptsZeroJobStartupFailure(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.actions.startup-blocker.v1",
  "status": "blocked",
  "repo": "BoSuY0/Tetra_Language",
  "branch": "codex/full-platform-ui-runtime",
  "workflow": "ci",
  "summary": "GitHub Actions created runs but no jobs or logs were available.",
  "runs": [
    {
      "id": 26246281021,
      "event": "workflow_dispatch",
      "conclusion": "startup_failure",
      "head_sha": "160a68184fd779bcc3797acf2bed65a6c9c83d78",
      "jobs": 0,
      "logs_available": false
    },
    {
      "id": 26246557763,
      "event": "push",
      "conclusion": "startup_failure",
      "head_sha": "57650e5324754acc828dad90adcc67bc0dd2499b",
      "jobs": 0,
      "logs_available": false
    }
  ],
  "next_action": "Use manual or self-hosted target-host Windows/macOS reports; do not count startup_failure as runtime evidence."
}`)
	if err := validateStartupBlocker(raw); err != nil {
		t.Fatalf("validateStartupBlocker: %v", err)
	}
}

func TestValidateStartupBlockerRejectsPassingOrJobBackedEvidence(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.actions.startup-blocker.v1",
  "status": "pass",
  "repo": "BoSuY0/Tetra_Language",
  "branch": "codex/full-platform-ui-runtime",
  "workflow": "ci",
  "summary": "fake pass",
  "runs": [
    {
      "id": 1,
      "event": "workflow_dispatch",
      "conclusion": "success",
      "head_sha": "abcdef1234567890",
      "jobs": 1,
      "logs_available": true
    }
  ],
  "next_action": "READY"
}`)
	err := validateStartupBlocker(raw)
	if err == nil {
		t.Fatalf("expected passing/job-backed startup blocker report to fail")
	}
	for _, want := range []string{"status", "startup_failure", "jobs", "logs_available", "next_action"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}
