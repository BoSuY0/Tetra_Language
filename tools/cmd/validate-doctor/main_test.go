package main

import "testing"

func TestValidateDoctorReportAcceptsExpectedShape(t *testing.T) {
	raw := []byte(`{
  "status": "pass",
  "checks": [
    {"name":"version","status":"pass","detail":"v0.6.0"},
    {"name":"supported targets","status":"pass","detail":"linux-x64, windows-x64, macos-x64"},
    {"name":"planned targets","status":"pass","detail":"wasm32-wasi, wasm32-web"},
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
    {"name":"runtime exports","status":"pass","detail":"4 files, 6 symbols"}
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

func TestValidateDoctorReportRejectsMissingRequiredCheck(t *testing.T) {
	raw := []byte(`{"status":"pass","checks":[{"name":"version","status":"pass"}]}`)
	if err := validateDoctorReport(raw); err == nil {
		t.Fatalf("expected missing check failure")
	}
}
