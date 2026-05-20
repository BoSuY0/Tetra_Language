package main

import (
	"strconv"
	"strings"
	"testing"
)

func TestValidateWorkspaceExecAcceptsBuildPassReport(t *testing.T) {
	raw := []byte(`{
  "workspace_root": "/workspace",
  "command": "build",
  "target": "linux-x64",
  "total": 2,
  "passed": 2,
  "failed": 0,
  "skipped": 0,
  "members": [
    {"path": "App", "capsule_id": "tetra://app", "status": "pass", "exit_code": 0},
    {"path": "Tool", "capsule_id": "tetra://tool", "status": "pass", "exit_code": 0}
  ]
}`)
	if err := validateWorkspaceExecReport(raw); err != nil {
		t.Fatalf("validate build pass report: %v", err)
	}
}

func TestValidateWorkspaceExecAcceptsTestFailureAndSkipReport(t *testing.T) {
	raw := []byte(`{
  "workspace_root": "/workspace",
  "command": "test",
  "target": "linux-x64",
  "total": 3,
  "passed": 1,
  "failed": 1,
  "skipped": 1,
  "members": [
    {"path": "Pass", "capsule_id": "tetra://pass", "status": "pass", "exit_code": 0},
    {"path": "Fail", "capsule_id": "tetra://fail", "status": "fail", "detail": "assertion failed", "exit_code": 1},
    {"path": "Later", "capsule_id": "tetra://later", "status": "skipped", "detail": "fail-fast after Fail"}
  ]
}`)
	if err := validateWorkspaceExecReport(raw); err != nil {
		t.Fatalf("validate test failure report: %v", err)
	}
}

func TestValidateWorkspaceExecRejectsUnknownField(t *testing.T) {
	raw := []byte(`{
  "workspace_root": "/workspace",
  "command": "build",
  "total": 0,
  "passed": 0,
  "failed": 0,
  "skipped": 0,
  "members": [],
  "extra": true
}`)
	err := validateWorkspaceExecReport(raw)
	if err == nil {
		t.Fatalf("expected unknown field failure")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkspaceExecRejectsMissingRequiredReportFields(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "missing total",
			raw: `{
  "workspace_root": "/workspace",
  "command": "build",
  "passed": 0,
  "failed": 0,
  "skipped": 0,
  "members": []
}`,
		},
		{
			name: "missing passed",
			raw: `{
  "workspace_root": "/workspace",
  "command": "build",
  "total": 0,
  "failed": 0,
  "skipped": 0,
  "members": []
}`,
		},
		{
			name: "missing failed",
			raw: `{
  "workspace_root": "/workspace",
  "command": "build",
  "total": 0,
  "passed": 0,
  "skipped": 0,
  "members": []
}`,
		},
		{
			name: "missing skipped",
			raw: `{
  "workspace_root": "/workspace",
  "command": "build",
  "total": 0,
  "passed": 0,
  "failed": 0,
  "members": []
}`,
		},
		{
			name: "missing members",
			raw: `{
  "workspace_root": "/workspace",
  "command": "build",
  "total": 0,
  "passed": 0,
  "failed": 0,
  "skipped": 0
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkspaceExecReport([]byte(tt.raw))
			if err == nil {
				t.Fatalf("expected missing required report field failure")
			}
			if !strings.Contains(err.Error(), "required") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateWorkspaceExecRejectsNullRequiredReportFields(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "null total",
			raw: `{
  "workspace_root": "/workspace",
  "command": "build",
  "total": null,
  "passed": 0,
  "failed": 0,
  "skipped": 0,
  "members": []
}`,
		},
		{
			name: "null passed",
			raw: `{
  "workspace_root": "/workspace",
  "command": "build",
  "total": 0,
  "passed": null,
  "failed": 0,
  "skipped": 0,
  "members": []
}`,
		},
		{
			name: "null failed",
			raw: `{
  "workspace_root": "/workspace",
  "command": "build",
  "total": 0,
  "passed": 0,
  "failed": null,
  "skipped": 0,
  "members": []
}`,
		},
		{
			name: "null skipped",
			raw: `{
  "workspace_root": "/workspace",
  "command": "build",
  "total": 0,
  "passed": 0,
  "failed": 0,
  "skipped": null,
  "members": []
}`,
		},
		{
			name: "null members",
			raw: `{
  "workspace_root": "/workspace",
  "command": "build",
  "total": 0,
  "passed": 0,
  "failed": 0,
  "skipped": 0,
  "members": null
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkspaceExecReport([]byte(tt.raw))
			if err == nil {
				t.Fatalf("expected null required report field failure")
			}
			if !strings.Contains(err.Error(), "must not be null") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateWorkspaceExecRejectsInvalidCommand(t *testing.T) {
	raw := []byte(`{
  "workspace_root": "/workspace",
  "command": "sync",
  "total": 0,
  "passed": 0,
  "failed": 0,
  "skipped": 0,
  "members": []
}`)
	err := validateWorkspaceExecReport(raw)
	if err == nil {
		t.Fatalf("expected invalid command failure")
	}
	if !strings.Contains(err.Error(), `command = "sync"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkspaceExecRejectsCountMismatch(t *testing.T) {
	raw := []byte(`{
  "workspace_root": "/workspace",
  "command": "build",
  "total": 2,
  "passed": 2,
  "failed": 0,
  "skipped": 0,
  "members": [
    {"path": "App", "capsule_id": "tetra://app", "status": "pass", "exit_code": 0}
  ]
}`)
	err := validateWorkspaceExecReport(raw)
	if err == nil {
		t.Fatalf("expected count mismatch failure")
	}
	if !strings.Contains(err.Error(), "count mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkspaceExecRejectsStatusExitCodeMismatch(t *testing.T) {
	raw := []byte(`{
  "workspace_root": "/workspace",
  "command": "test",
  "total": 1,
  "passed": 1,
  "failed": 0,
  "skipped": 0,
  "members": [
    {"path": "App", "capsule_id": "tetra://app", "status": "pass", "exit_code": 1}
  ]
}`)
	err := validateWorkspaceExecReport(raw)
	if err == nil {
		t.Fatalf("expected status/exit_code mismatch failure")
	}
	if !strings.Contains(err.Error(), "pass member App has non-zero exit_code 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkspaceExecRejectsInvalidMemberPaths(t *testing.T) {
	tests := []struct {
		name        string
		memberPath  string
		wantMessage string
	}{
		{
			name:        "posix absolute path",
			memberPath:  "/workspace/App",
			wantMessage: "member /workspace/App path must be workspace-relative",
		},
		{
			name:        "parent relative path",
			memberPath:  "../App",
			wantMessage: "member ../App path must be workspace-relative",
		},
		{
			name:        "windows absolute path",
			memberPath:  `C:\workspace\App`,
			wantMessage: `member C:\workspace\App path must be workspace-relative`,
		},
		{
			name:        "backslash delimited path",
			memberPath:  `Apps\Tool`,
			wantMessage: `member Apps\Tool path must use portable slash delimiters`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := []byte(`{
  "workspace_root": "/workspace",
  "command": "test",
  "total": 1,
  "passed": 1,
  "failed": 0,
  "skipped": 0,
  "members": [
    {"path": ` + strconv.Quote(tt.memberPath) + `, "capsule_id": "tetra://app", "status": "pass", "exit_code": 0}
  ]
}`)
			err := validateWorkspaceExecReport(raw)
			if err == nil {
				t.Fatalf("expected invalid member path failure")
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateWorkspaceExecRejectsSkippedExitCode(t *testing.T) {
	raw := []byte(`{
  "workspace_root": "/workspace",
  "command": "test",
  "total": 1,
  "passed": 0,
  "failed": 0,
  "skipped": 1,
  "members": [
    {"path": "App", "capsule_id": "tetra://app", "status": "skipped", "detail": "fail-fast after Lib", "exit_code": 0}
  ]
}`)
	err := validateWorkspaceExecReport(raw)
	if err == nil {
		t.Fatalf("expected skipped exit_code failure")
	}
	if !strings.Contains(err.Error(), "skipped member App must not include exit_code") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkspaceExecRejectsDuplicateMember(t *testing.T) {
	raw := []byte(`{
  "workspace_root": "/workspace",
  "command": "build",
  "total": 2,
  "passed": 2,
  "failed": 0,
  "skipped": 0,
  "members": [
    {"path": "App", "capsule_id": "tetra://app", "status": "pass", "exit_code": 0},
    {"path": "App", "capsule_id": "tetra://app2", "status": "pass", "exit_code": 0}
  ]
}`)
	err := validateWorkspaceExecReport(raw)
	if err == nil {
		t.Fatalf("expected duplicate member failure")
	}
	if !strings.Contains(err.Error(), "duplicate member path App") {
		t.Fatalf("unexpected error: %v", err)
	}
}
