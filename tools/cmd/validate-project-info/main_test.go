package main

import (
	"strings"
	"testing"
)

func TestValidateProjectInfoReportAcceptsFoundProject(t *testing.T) {
	raw := []byte(`{
  "found": true,
  "root": "/repo/App",
  "capsule_path": "/repo/App/Capsule.t4",
  "lock_path": "/repo/App/Tetra.lock",
  "entry_path": "src/main.t4",
  "source_roots": ["src", "tests"],
  "targets": ["linux-x64"],
  "dependency_roots": ["/repo/Math"],
  "artifact_counts": {"interface": 1, "object": 2},
  "dependency_capsules": ["/repo/Math/Capsule.t4"]
}`)
	if err := validateProjectInfoReport(raw); err != nil {
		t.Fatalf("validate project info: %v", err)
	}
}

func TestValidateProjectInfoReportAcceptsNotFoundProject(t *testing.T) {
	raw := []byte(`{"found":false}`)
	if err := validateProjectInfoReport(raw); err != nil {
		t.Fatalf("validate project info not found: %v", err)
	}
}

func TestValidateProjectInfoReportRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{"found":true,"root":"/repo/App","capsule_path":"/repo/App/Capsule.t4","entry_path":"src/main.t4","source_roots":["src"],"targets":[],"extra":true}`)
	if err := validateProjectInfoReport(raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field failure, got %v", err)
	}
}

func TestValidateProjectInfoReportRejectsMissingFoundProjectFields(t *testing.T) {
	raw := []byte(`{"found":true,"root":"/repo/App","capsule_path":"/repo/App/Capsule.t4","source_roots":["src"],"targets":[]}`)
	if err := validateProjectInfoReport(raw); err == nil {
		t.Fatalf("expected missing entry path failure")
	}
}

func TestValidateProjectInfoReportRejectsNotFoundWithProjectFields(t *testing.T) {
	raw := []byte(`{"found":false,"root":"/repo/App"}`)
	if err := validateProjectInfoReport(raw); err == nil {
		t.Fatalf("expected not found project fields failure")
	}
}

func TestValidateProjectInfoReportRejectsNegativeArtifactCount(t *testing.T) {
	raw := []byte(`{"found":true,"root":"/repo/App","capsule_path":"/repo/App/Capsule.t4","entry_path":"src/main.t4","source_roots":["src"],"targets":[],"artifact_counts":{"object":-1}}`)
	if err := validateProjectInfoReport(raw); err == nil {
		t.Fatalf("expected negative artifact count failure")
	}
}
