package main

import (
	"strings"
	"testing"
)

func TestValidateProjectDepsReportAcceptsListReport(t *testing.T) {
	raw := []byte(`{
  "root": "/repo/App",
  "capsule_path": "/repo/App/Capsule.t4",
  "dependencies": [
    {
      "id": "tetra://math",
      "version": "0.1.0",
      "path": "../Math",
      "resolved_path": "/repo/Math",
      "status": "ok"
    }
  ]
}`)
	if err := validateProjectDepsReport(raw); err != nil {
		t.Fatalf("validate project deps: %v", err)
	}
}

func TestValidateProjectDepsReportAcceptsCheckPassReport(t *testing.T) {
	raw := []byte(`{
  "status": "pass",
  "root": "/repo/App",
  "capsule_path": "/repo/App/Capsule.t4",
  "dependencies": []
}`)
	if err := validateProjectDepsReport(raw); err != nil {
		t.Fatalf("validate project deps check pass: %v", err)
	}
}

func TestValidateProjectDepsReportAcceptsCheckFailReport(t *testing.T) {
	raw := []byte(`{
  "status": "fail",
  "root": "/repo/App",
  "capsule_path": "/repo/App/Capsule.t4",
  "dependencies": [
    {
      "id": "tetra://math",
      "version": "0.1.0",
      "path": "../Math",
      "status": "missing",
      "detail": "stat /repo/Math: no such file or directory"
    }
  ]
}`)
	if err := validateProjectDepsReport(raw); err != nil {
		t.Fatalf("validate project deps check fail: %v", err)
	}
}

func TestValidateProjectDepsReportRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{"root":"/repo/App","capsule_path":"/repo/App/Capsule.t4","dependencies":[],"extra":true}`)
	if err := validateProjectDepsReport(raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown top-level field failure, got %v", err)
	}
	raw = []byte(`{"root":"/repo/App","capsule_path":"/repo/App/Capsule.t4","dependencies":[{"id":"tetra://math","version":"0.1.0","path":"../Math","resolved_path":"/repo/Math","status":"ok","extra":true}]}`)
	if err := validateProjectDepsReport(raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown dependency field failure, got %v", err)
	}
}

func TestValidateProjectDepsReportRejectsInvalidTopLevelStatus(t *testing.T) {
	raw := []byte(`{"status":"ok","root":"/repo/App","capsule_path":"/repo/App/Capsule.t4","dependencies":[]}`)
	if err := validateProjectDepsReport(raw); err == nil {
		t.Fatalf("expected invalid top-level status failure")
	}
}

func TestValidateProjectDepsReportRejectsPassWithNonOKDependency(t *testing.T) {
	raw := []byte(`{"status":"pass","root":"/repo/App","capsule_path":"/repo/App/Capsule.t4","dependencies":[{"id":"tetra://math","version":"0.1.0","path":"../Math","status":"missing","detail":"missing"}]}`)
	if err := validateProjectDepsReport(raw); err == nil {
		t.Fatalf("expected pass with non-ok dependency failure")
	}
}

func TestValidateProjectDepsReportRejectsFailWithoutIssue(t *testing.T) {
	raw := []byte(`{"status":"fail","root":"/repo/App","capsule_path":"/repo/App/Capsule.t4","dependencies":[{"id":"tetra://math","version":"0.1.0","path":"../Math","resolved_path":"/repo/Math","status":"ok"}]}`)
	if err := validateProjectDepsReport(raw); err == nil {
		t.Fatalf("expected fail without issue failure")
	}
}

func TestValidateProjectDepsReportRejectsBadDependencyFields(t *testing.T) {
	raw := []byte(`{"root":"/repo/App","capsule_path":"/repo/App/Capsule.t4","dependencies":[{"id":"math","version":"0.1.0","path":"../Math","resolved_path":"/repo/Math","status":"ok"}]}`)
	if err := validateProjectDepsReport(raw); err == nil {
		t.Fatalf("expected invalid dependency id failure")
	}
	raw = []byte(`{"root":"/repo/App","capsule_path":"/repo/App/Capsule.t4","dependencies":[{"id":"tetra://math","version":"0.1","path":"../Math","resolved_path":"/repo/Math","status":"ok"}]}`)
	if err := validateProjectDepsReport(raw); err == nil {
		t.Fatalf("expected invalid dependency version failure")
	}
	raw = []byte(`{"root":"/repo/App","capsule_path":"/repo/App/Capsule.t4","dependencies":[{"id":"tetra://math","version":"0.1.0","path":"../Math","status":"ok"}]}`)
	if err := validateProjectDepsReport(raw); err == nil {
		t.Fatalf("expected ok dependency missing resolved path failure")
	}
}
