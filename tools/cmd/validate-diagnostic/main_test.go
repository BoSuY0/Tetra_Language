package main

import (
	"testing"

	"tetra_language/internal/toon"
)

func TestValidateDiagnosticAcceptsStableShape(t *testing.T) {
	diag, err := parseDiagnostic(
		[]byte(
			`{"code":"TETRA2001","message":"unknown function","file":"bad.tetra","line":2,"column":5,"severity":"error","hint":"check spelling"}`,
		),
	)
	if err != nil {
		t.Fatalf("parse diagnostic: %v", err)
	}
	if err := validateDiagnostic(diag, "TETRA2001", "error", "unknown function", true); err != nil {
		t.Fatalf("validate diagnostic: %v", err)
	}
}

func TestValidateDiagnosticAcceptsTOONStableShape(t *testing.T) {
	raw, err := toon.ConvertJSONToTOON(
		[]byte(
			`{"code":"TETRA2001","message":"unknown function","file":"bad.tetra","line":2,"column":5,"severity":"error","hint":"check spelling"}`,
		),
		toon.Options{Deterministic: true, Strict: true},
	)
	if err != nil {
		t.Fatalf("json->toon: %v", err)
	}
	diag, err := parseDiagnostic(raw)
	if err != nil {
		t.Fatalf("parse diagnostic: %v\n%s", err, raw)
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

func TestValidateDiagnosticRejectsUnknownFields(t *testing.T) {
	if _, err := parseDiagnostic(
		[]byte(`{"code":"TETRA2001","message":"bad","severity":"error","extra":true}`),
	); err == nil {
		t.Fatalf("expected unknown field failure")
	}
}

func TestValidateDiagnosticRejectsWrongCodeSeverityAndMessage(t *testing.T) {
	diag, err := parseDiagnostic(
		[]byte(`{"code":"TETRA0001","message":"parse failed","severity":"warning"}`),
	)
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

func TestValidateDiagnosticRejectsWhitespaceDrift(t *testing.T) {
	diag, err := parseDiagnostic([]byte(`{"code":"TETRA2001 ","message":"bad","severity":"error"}`))
	if err != nil {
		t.Fatalf("parse diagnostic: %v", err)
	}
	if err := validateDiagnostic(diag, "", "", "", false); err == nil {
		t.Fatalf("expected whitespace drift failure")
	}
}

func TestValidateDiagnosticRejectsPartialPositionWithoutFile(t *testing.T) {
	diag, err := parseDiagnostic(
		[]byte(`{"code":"TETRA2001","message":"bad","severity":"error","line":1,"column":1}`),
	)
	if err != nil {
		t.Fatalf("parse diagnostic: %v", err)
	}
	if err := validateDiagnostic(diag, "", "", "", false); err == nil {
		t.Fatalf("expected partial position failure")
	}
}
