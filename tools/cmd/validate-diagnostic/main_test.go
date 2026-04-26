package main

import "testing"

func TestValidateDiagnosticAcceptsStableShape(t *testing.T) {
	diag, err := parseDiagnostic([]byte(`{"code":"TETRA2001","message":"unknown function","file":"bad.tetra","line":2,"column":5,"severity":"error","hint":"check spelling"}`))
	if err != nil {
		t.Fatalf("parse diagnostic: %v", err)
	}
	if err := validateDiagnostic(diag, "TETRA2001", "error", "unknown function", true); err != nil {
		t.Fatalf("validate diagnostic: %v", err)
	}
}

func TestValidateDiagnosticRejectsMissingRequiredFields(t *testing.T) {
	diag, err := parseDiagnostic([]byte(`{"code":"TETRA0001","severity":"error"}`))
	if err != nil {
		t.Fatalf("parse diagnostic: %v", err)
	}
	if err := validateDiagnostic(diag, "", "error", "", false); err == nil {
		t.Fatalf("expected missing message failure")
	}
}

func TestValidateDiagnosticRejectsWrongCodeSeverityAndMessage(t *testing.T) {
	diag, err := parseDiagnostic([]byte(`{"code":"TETRA0001","message":"parse failed","severity":"warning"}`))
	if err != nil {
		t.Fatalf("parse diagnostic: %v", err)
	}
	if err := validateDiagnostic(diag, "TETRA2001", "error", "unknown", false); err == nil {
		t.Fatalf("expected mismatch failure")
	}
}

func TestValidateDiagnosticRejectsInvalidSeverity(t *testing.T) {
	diag, err := parseDiagnostic([]byte(`{"code":"TETRA2001","message":"bad","severity":"fatal"}`))
	if err != nil {
		t.Fatalf("parse diagnostic: %v", err)
	}
	if err := validateDiagnostic(diag, "", "", "", false); err == nil {
		t.Fatalf("expected invalid severity failure")
	}
}

func TestValidateDiagnosticRejectsMissingRequiredPosition(t *testing.T) {
	diag, err := parseDiagnostic([]byte(`{"code":"TETRA2001","message":"bad","severity":"error"}`))
	if err != nil {
		t.Fatalf("parse diagnostic: %v", err)
	}
	if err := validateDiagnostic(diag, "", "", "", true); err == nil {
		t.Fatalf("expected missing position failure")
	}
}
