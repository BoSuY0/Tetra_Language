package main

import "testing"

func TestValidateTargetsReportAcceptsExpectedShape(t *testing.T) {
	raw := []byte(`{"supported":["linux-x64","windows-x64","macos-x64"],"build_only":["wasm32-wasi"],"planned":["wasm32-web"]}`)
	if err := validateTargetsReport(raw); err != nil {
		t.Fatalf("validate targets: %v", err)
	}
}

func TestValidateTargetsReportRejectsWrongOrder(t *testing.T) {
	raw := []byte(`{"supported":["windows-x64","linux-x64","macos-x64"],"build_only":["wasm32-wasi"],"planned":["wasm32-web"]}`)
	if err := validateTargetsReport(raw); err == nil {
		t.Fatalf("expected wrong-order failure")
	}
}

func TestValidateTargetsReportRejectsDuplicate(t *testing.T) {
	if err := validateTargetList("supported", []string{"linux-x64", "linux-x64"}, []string{"linux-x64", "linux-x64"}); err == nil {
		t.Fatalf("expected duplicate failure")
	}
}
