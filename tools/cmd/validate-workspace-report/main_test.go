package main

import (
	"strings"
	"testing"
)

func TestValidateWorkspaceReportAcceptsList(t *testing.T) {
	raw := []byte(`{
  "root": "/workspace",
  "workspace_path": "/workspace/Tetra.workspace",
  "members": [
    {
      "path": "Math",
      "resolved_path": "/workspace/Math",
      "capsule_path": "/workspace/Math/Capsule.t4",
      "capsule_id": "tetra://math",
      "version": "0.1.0",
      "status": "ok"
    }
  ]
}`)
	if err := validateWorkspaceReport(raw, "list"); err != nil {
		t.Fatalf("validate list: %v", err)
	}
}

func TestValidateWorkspaceReportAcceptsCheckFailure(t *testing.T) {
	raw := []byte(`{
  "status": "fail",
  "root": "/workspace",
  "workspace_path": "/workspace/Tetra.workspace",
  "members": [
    {
      "path": "Missing",
      "resolved_path": "/workspace/Missing",
      "status": "missing",
      "detail": "stat /workspace/Missing: no such file or directory"
    }
  ]
}`)
	if err := validateWorkspaceReport(raw, "check"); err != nil {
		t.Fatalf("validate check failure: %v", err)
	}
}

func TestValidateWorkspaceReportAcceptsGraph(t *testing.T) {
	raw := []byte(`{
  "status": "pass",
  "root": "/workspace",
  "workspace_path": "/workspace/Tetra.workspace",
  "nodes": [
    {
      "path": "Math",
      "resolved_path": "/workspace/Math",
      "capsule_path": "/workspace/Math/Capsule.t4",
      "capsule_id": "tetra://math",
      "version": "0.1.0",
      "status": "ok"
    },
    {
      "path": "App",
      "resolved_path": "/workspace/App",
      "capsule_path": "/workspace/App/Capsule.t4",
      "capsule_id": "tetra://app",
      "version": "0.1.0",
      "status": "ok"
    }
  ],
  "edges": [
    {"from": "App", "to": "Math", "id": "tetra://math"}
  ]
}`)
	if err := validateWorkspaceReport(raw, "graph"); err != nil {
		t.Fatalf("validate graph: %v", err)
	}
}

func TestValidateWorkspaceReportRejectsUnknownField(t *testing.T) {
	raw := []byte(`{
  "root": "/workspace",
  "workspace_path": "/workspace/Tetra.workspace",
  "members": [],
  "extra": true
}`)
	err := validateWorkspaceReport(raw, "list")
	if err == nil {
		t.Fatalf("expected unknown field failure")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkspaceReportRejectsDuplicateMemberPath(t *testing.T) {
	raw := []byte(`{
  "root": "/workspace",
  "workspace_path": "/workspace/Tetra.workspace",
  "members": [
    {"path": "App", "resolved_path": "/workspace/App", "capsule_path": "/workspace/App/Capsule.t4", "capsule_id": "tetra://app", "version": "0.1.0", "status": "ok"},
    {"path": "App", "resolved_path": "/workspace/App2", "capsule_path": "/workspace/App2/Capsule.t4", "capsule_id": "tetra://app2", "version": "0.1.0", "status": "ok"}
  ]
}`)
	err := validateWorkspaceReport(raw, "list")
	if err == nil {
		t.Fatalf("expected duplicate member path failure")
	}
	if !strings.Contains(err.Error(), "duplicate workspace member path App") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkspaceReportRejectsDuplicateOKCapsuleID(t *testing.T) {
	raw := []byte(`{
  "status": "pass",
  "root": "/workspace",
  "workspace_path": "/workspace/Tetra.workspace",
  "members": [
    {"path": "App", "resolved_path": "/workspace/App", "capsule_path": "/workspace/App/Capsule.t4", "capsule_id": "tetra://shared", "version": "0.1.0", "status": "ok"},
    {"path": "Tool", "resolved_path": "/workspace/Tool", "capsule_path": "/workspace/Tool/Capsule.t4", "capsule_id": "tetra://shared", "version": "0.1.0", "status": "ok"}
  ]
}`)
	err := validateWorkspaceReport(raw, "check")
	if err == nil {
		t.Fatalf("expected duplicate capsule id failure")
	}
	if !strings.Contains(err.Error(), "duplicate ok capsule_id tetra://shared") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkspaceReportRejectsCheckStatusMismatch(t *testing.T) {
	raw := []byte(`{
  "status": "pass",
  "root": "/workspace",
  "workspace_path": "/workspace/Tetra.workspace",
  "members": [
    {"path": "Missing", "resolved_path": "/workspace/Missing", "status": "missing", "detail": "missing"}
  ]
}`)
	err := validateWorkspaceReport(raw, "check")
	if err == nil {
		t.Fatalf("expected status mismatch failure")
	}
	if !strings.Contains(err.Error(), "status = \"pass\", want fail") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkspaceReportRejectsMissingMembersField(t *testing.T) {
	list := []byte(`{
  "root": "/workspace",
  "workspace_path": "/workspace/Tetra.workspace"
}`)
	check := []byte(`{
  "status": "pass",
  "root": "/workspace",
  "workspace_path": "/workspace/Tetra.workspace"
}`)
	for _, tc := range []struct {
		kind string
		raw  []byte
	}{
		{kind: "list", raw: list},
		{kind: "check", raw: check},
	} {
		err := validateWorkspaceReport(tc.raw, tc.kind)
		if err == nil {
			t.Fatalf("expected missing members failure for %s", tc.kind)
		}
		if !strings.Contains(err.Error(), "members is required") {
			t.Fatalf("unexpected error for %s: %v", tc.kind, err)
		}
	}
}

func TestValidateWorkspaceReportRejectsMissingGraphShapeFields(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "nodes",
			raw: `{
  "status": "pass",
  "root": "/workspace",
  "workspace_path": "/workspace/Tetra.workspace",
  "edges": []
}`,
			want: "nodes is required",
		},
		{
			name: "edges",
			raw: `{
  "status": "pass",
  "root": "/workspace",
  "workspace_path": "/workspace/Tetra.workspace",
  "nodes": []
}`,
			want: "edges is required",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateWorkspaceReport([]byte(tc.raw), "graph")
			if err == nil {
				t.Fatalf("expected missing graph shape field failure")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateWorkspaceReportRejectsFailStatusWhenAllEntriesOK(t *testing.T) {
	check := []byte(`{
  "status": "fail",
  "root": "/workspace",
  "workspace_path": "/workspace/Tetra.workspace",
  "members": [
    {"path": "App", "resolved_path": "/workspace/App", "capsule_path": "/workspace/App/Capsule.t4", "capsule_id": "tetra://app", "version": "0.1.0", "status": "ok"}
  ]
}`)
	err := validateWorkspaceReport(check, "check")
	if err == nil {
		t.Fatalf("expected check fail status mismatch")
	}
	if !strings.Contains(err.Error(), "status = \"fail\", want pass") {
		t.Fatalf("unexpected check error: %v", err)
	}

	graph := []byte(`{
  "status": "fail",
  "root": "/workspace",
  "workspace_path": "/workspace/Tetra.workspace",
  "nodes": [
    {"path": "App", "resolved_path": "/workspace/App", "capsule_path": "/workspace/App/Capsule.t4", "capsule_id": "tetra://app", "version": "0.1.0", "status": "ok"}
  ],
  "edges": []
}`)
	err = validateWorkspaceReport(graph, "graph")
	if err == nil {
		t.Fatalf("expected graph fail status mismatch")
	}
	if !strings.Contains(err.Error(), "status = \"fail\", want pass") {
		t.Fatalf("unexpected graph error: %v", err)
	}
}

func TestValidateWorkspaceReportRejectsGraphEdgeUnknownNode(t *testing.T) {
	raw := []byte(`{
  "status": "pass",
  "root": "/workspace",
  "workspace_path": "/workspace/Tetra.workspace",
  "nodes": [
    {"path": "App", "resolved_path": "/workspace/App", "capsule_path": "/workspace/App/Capsule.t4", "capsule_id": "tetra://app", "version": "0.1.0", "status": "ok"}
  ],
  "edges": [
    {"from": "App", "to": "Math", "id": "tetra://math"}
  ]
}`)
	err := validateWorkspaceReport(raw, "graph")
	if err == nil {
		t.Fatalf("expected unknown edge node failure")
	}
	if !strings.Contains(err.Error(), "edge App -> Math references unknown to node") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkspaceReportRejectsInvalidKind(t *testing.T) {
	err := validateWorkspaceReport([]byte(`{}`), "build")
	if err == nil {
		t.Fatalf("expected invalid kind failure")
	}
	if !strings.Contains(err.Error(), "unsupported --kind") {
		t.Fatalf("unexpected error: %v", err)
	}
}
