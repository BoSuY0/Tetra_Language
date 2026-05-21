package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateNativeUIRuntimeReportAcceptsExecutableEvidence(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "native-ui.json")
	if err := os.WriteFile(reportPath, validNativeUIRuntimeReportJSON(), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateNativeUIRuntimeReport(reportPath); err != nil {
		t.Fatalf("validateNativeUIRuntimeReport failed: %v", err)
	}
}

func TestValidateNativeUIRuntimeReportRejectsSidecarOnlyEvidence(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "native-ui.json")
	raw := []byte(`{"schema":"tetra.ui.native-shell.v1","ui_schema":"tetra.ui.v0.4.0","runtime":"native shell command dispatch"}`)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateNativeUIRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected sidecar-only report to fail")
	}
	if !strings.Contains(err.Error(), "tetra.ui.native-runtime.v1") {
		t.Fatalf("error = %v, want native-runtime schema rejection", err)
	}
}

func validNativeUIRuntimeReportJSON() []byte {
	return []byte(`{
  "schema": "tetra.ui.native-runtime.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "native-ui-linux-x64",
  "ui_schema": "tetra.ui.v0.4.0",
  "source": "examples/ui_native_shell_smoke.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"/tmp/tetra","ran":true,"pass":true,"exit_code":0},
    {"name":"native app","kind":"app","path":"/tmp/ui-native","ran":true,"pass":true,"exit_code":0},
    {"name":"native ui runtime","kind":"runtime","path":"tools/cmd/native-ui-runtime-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "widgets": [
    {"id":"ShellView","kind":"view","parent":"","enabled":true,"visible":true,"bounds":{"x":0,"y":0,"width":320,"height":96}},
    {"id":"ShellView.toggles","kind":"value","parent":"ShellView","binding":"toggles","value":"0","enabled":true,"visible":true,"bounds":{"x":8,"y":8,"width":304,"height":24}},
    {"id":"ShellView.submit","kind":"action","parent":"ShellView","event":"submit","command":"toggle","enabled":true,"visible":true,"bounds":{"x":8,"y":40,"width":304,"height":24}}
  ],
  "events": [
    {"order":1,"widget_id":"ShellView.submit","event":"click","command":"toggle","pass":true,"before_state":{"ShellState.toggles":"0"},"after_state":{"ShellState.toggles":"1"},"operations":[{"kind":"state_add","target":"state.toggles","value":"1","state_field":"toggles","state_value":"1"}],"widget_updates":[{"id":"ShellView.toggles","before":"0","after":"1"}]},
    {"order":2,"widget_id":"ShellView.submit","event":"click","command":"toggle","pass":true,"before_state":{"ShellState.toggles":"1"},"after_state":{"ShellState.toggles":"2"},"operations":[{"kind":"state_add","target":"state.toggles","value":"1","state_field":"toggles","state_value":"2"}],"widget_updates":[{"id":"ShellView.toggles","before":"1","after":"2"}]}
  ],
  "cases": [
    {"name":"load widget tree","ran":true,"pass":true},
    {"name":"dispatch click command","ran":true,"pass":true},
    {"name":"propagate state update","ran":true,"pass":true},
    {"name":"dispatch multiple ordered events","ran":true,"pass":true},
    {"name":"reject invalid widget id","ran":true,"pass":true,"expected_error":"unknown widget"},
    {"name":"reject malformed metadata","ran":true,"pass":true,"expected_error":"malformed metadata"},
    {"name":"reject unsupported event kind","ran":true,"pass":true,"expected_error":"unsupported event"},
    {"name":"reject command failure","ran":true,"pass":true,"expected_error":"unknown command"},
    {"name":"close runtime","ran":true,"pass":true}
  ]
}`)
}
