package main

import (
	"strings"
	"testing"
)

func TestValidateTargetHostEvidenceRequestAcceptsExactHandoff(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.ui.target-host-evidence-request.v1",
  "status": "request",
  "production_evidence": false,
  "repo": "BoSuY0/Tetra_Language",
  "branch": "codex/full-platform-ui-runtime",
  "expected_version": "v0.4.0",
  "expected_git_head": "abcdef1234567890",
  "warning": "This request bundle is not runtime evidence. Only validator-passing target-host reports from the same Git commit count.",
  "targets": [
    {
      "target": "windows-x64",
      "host_requirement": "real Windows x64 host",
      "report": "windows-ui-runtime.json",
      "command": "git clone https://github.com/BoSuY0/Tetra_Language.git tetra-ui-runtime && cd tetra-ui-runtime && git fetch origin codex/full-platform-ui-runtime && git checkout abcdef1234567890 && pwsh -File scripts/release/full_platform/windows-ui-runtime-smoke.ps1 -Report windows-ui-runtime.json -ExpectedVersion v0.4.0 -ExpectedGitHead abcdef1234567890"
    },
    {
      "target": "macos-x64",
      "host_requirement": "real macOS x64 host",
      "report": "macos-ui-runtime.json",
      "command": "git clone https://github.com/BoSuY0/Tetra_Language.git tetra-ui-runtime && cd tetra-ui-runtime && git fetch origin codex/full-platform-ui-runtime && git checkout abcdef1234567890 && bash scripts/release/full_platform/target-host-ui-runtime-smoke.sh --target macos-x64 --report macos-ui-runtime.json --expected-version v0.4.0 --expected-git-head abcdef1234567890"
    }
  ],
  "aggregation": {
    "host_requirement": "Linux aggregation host with the same Git commit checked out",
    "command": "TETRA_WINDOWS_UI_RUNTIME_REPORT=/path/windows-ui-runtime.json TETRA_MACOS_UI_RUNTIME_REPORT=/path/macos-ui-runtime.json bash scripts/release/full_platform/ui-runtime-gate.sh --report-dir reports/full-platform-ui-runtime"
  }
}`)
	opts := requestValidationOptions{
		ExpectedRepo:    "BoSuY0/Tetra_Language",
		ExpectedBranch:  "codex/full-platform-ui-runtime",
		ExpectedVersion: "v0.4.0",
		ExpectedGitHead: "abcdef1234567890",
	}
	if err := validateTargetHostEvidenceRequest(raw, opts); err != nil {
		t.Fatalf("validateTargetHostEvidenceRequest: %v", err)
	}
}

func TestValidateTargetHostEvidenceRequestRejectsRuntimeEvidenceClaims(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.ui.target-host-evidence-request.v1",
  "status": "ready",
  "production_evidence": true,
  "repo": "BoSuY0/Tetra_Language.git",
  "branch": "codex/full-platform-ui-runtime",
  "expected_version": "v0.4.0",
  "expected_git_head": "abcdef1234567890",
  "warning": "READY production runtime evidence",
  "targets": [
    {
      "target": "windows-x64",
      "host_requirement": "real Windows x64 host",
      "report": "windows-ui-runtime.json",
      "command": "git clone https://github.com/BoSuY0/Tetra_Language.git.git tetra-ui-runtime && git checkout abcdef1234567890"
    }
  ],
  "aggregation": {
    "host_requirement": "Linux aggregation host",
    "command": "echo READY"
  }
}`)
	err := validateTargetHostEvidenceRequest(raw, requestValidationOptions{
		ExpectedRepo:    "BoSuY0/Tetra_Language",
		ExpectedBranch:  "codex/full-platform-ui-runtime",
		ExpectedVersion: "v0.4.0",
		ExpectedGitHead: "abcdef1234567890",
	})
	if err == nil {
		t.Fatalf("expected fake target-host request to fail")
	}
	for _, want := range []string{"status", "production_evidence", "repo", ".git.git", "READY", "macos-x64"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}
