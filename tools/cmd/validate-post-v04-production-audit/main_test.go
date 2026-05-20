package main

import (
	"testing"
)

func TestValidatePostV04ProductionAuditRejectsMissingAudit(t *testing.T) {
	dir := t.TempDir()
	err := validatePostV04ProductionAudit(dir, "")
	if err == nil {
		t.Fatalf("expected missing audit to fail")
	}
}

func TestWritePostV04ProductionAuditRejectsMissingLayerReports(t *testing.T) {
	dir := t.TempDir()
	err := writePostV04ProductionAudit(dir, "")
	if err == nil {
		t.Fatalf("expected missing layer reports to fail")
	}
}
