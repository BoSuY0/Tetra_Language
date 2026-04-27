package main

import (
	"strings"
	"testing"
)

func TestValidateDoctorReportAcceptsExpectedShape(t *testing.T) {
	raw := []byte(`{
  "status": "pass",
  "checks": [
    {"name":"version","status":"pass","detail":"v0.6.0"},
    {"name":"supported targets","status":"pass","detail":"linux-x64, windows-x64, macos-x64"},
    {"name":"build-only targets","status":"pass","detail":"wasm32-wasi"},
    {"name":"planned targets","status":"pass","detail":"wasm32-web"},
    {"name":"repo root","status":"pass","detail":"/tmp/tetra"},
    {"name":"__rt/actors_sysv.tetra","status":"pass","detail":"found"},
    {"name":"__rt/actors_win64.tetra","status":"pass","detail":"found"},
    {"name":"compiler/selfhostrt/actors_sysv.tetra","status":"pass","detail":"found"},
    {"name":"compiler/selfhostrt/actors_win64.tetra","status":"pass","detail":"found"},
    {"name":"examples/flow_hello.tetra","status":"pass","detail":"found"},
    {"name":"docs/generated/manifest.json","status":"pass","detail":"found"},
    {"name":"docs manifest version","status":"pass","detail":"v0.6.0"},
    {"name":"docs manifest surface","status":"pass","detail":"3 targets, 6 runtime symbols"},
    {"name":"smoke sources","status":"pass","detail":"40 sources"},
    {"name":"runtime exports","status":"pass","detail":"4 files, 6 symbols"},
    {"name":"target metadata","status":"pass","detail":"5 targets, 2 build-only"},
    {"name":"tooling commands","status":"pass","detail":"fmt, test, doc, smoke, lsp, eco"}
  ]
}`)
	if err := validateDoctorReport(raw); err != nil {
		t.Fatalf("validate doctor: %v", err)
	}
}

func TestValidateDoctorReportRejectsFailingStatus(t *testing.T) {
	raw := []byte(`{"status":"fail","checks":[{"name":"version","status":"pass"}]}`)
	if err := validateDoctorReport(raw); err == nil {
		t.Fatalf("expected status failure")
	}
}

func TestValidateDoctorReportRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{"status":"pass","checks":[],"extra":true}`)
	if err := validateDoctorReport(raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field failure, got %v", err)
	}
	raw = []byte(`{"status":"pass","checks":[{"name":"version","status":"pass","extra":true}]}`)
	if err := validateDoctorReport(raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected nested unknown field failure, got %v", err)
	}
}

func TestValidateDoctorReportRejectsMissingRequiredCheck(t *testing.T) {
	raw := []byte(`{"status":"pass","checks":[{"name":"version","status":"pass"}]}`)
	if err := validateDoctorReport(raw); err == nil {
		t.Fatalf("expected missing check failure")
	}
}

func TestValidateDoctorReportRejectsFailingRequiredCheck(t *testing.T) {
	raw := []byte(`{
  "status": "pass",
  "checks": [
    {"name":"version","status":"pass"},
    {"name":"supported targets","status":"pass"},
    {"name":"build-only targets","status":"pass"},
    {"name":"planned targets","status":"pass"},
    {"name":"repo root","status":"pass"},
    {"name":"__rt/actors_sysv.tetra","status":"pass"},
    {"name":"__rt/actors_win64.tetra","status":"pass"},
    {"name":"compiler/selfhostrt/actors_sysv.tetra","status":"pass"},
    {"name":"compiler/selfhostrt/actors_win64.tetra","status":"pass"},
    {"name":"examples/flow_hello.tetra","status":"pass"},
    {"name":"docs/generated/manifest.json","status":"pass"},
    {"name":"docs manifest version","status":"pass"},
    {"name":"docs manifest surface","status":"pass"},
    {"name":"smoke sources","status":"pass"},
    {"name":"runtime exports","status":"pass"},
    {"name":"target metadata","status":"fail","detail":"duplicate target linux-x64"},
    {"name":"tooling commands","status":"pass"}
  ]
}`)
	if err := validateDoctorReport(raw); err == nil {
		t.Fatalf("expected failing check rejection")
	}
}
