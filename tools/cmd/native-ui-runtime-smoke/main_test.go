package main

import (
	"encoding/json"
	"strings"
	"testing"

	"tetra_language/tools/validators/nativeui"
)

func TestRunRuntimeScenarioProducesValidNativeRuntimeEvidence(t *testing.T) {
	widgets, events, cases, err := runRuntimeScenario(nativeRuntimeSmokeFixture())
	if err != nil {
		t.Fatalf("runRuntimeScenario failed: %v", err)
	}
	report := nativeui.Report{
		Schema:   nativeui.SchemaV1,
		Status:   "pass",
		Target:   "linux-x64",
		Host:     "linux-x64",
		Runtime:  "native-ui-linux-x64",
		UISchema: uiBundleSchemaV1,
		Source:   "examples/ui_native_shell_smoke.tetra",
		Processes: []nativeui.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "/tmp/tetra", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "native app", Kind: "app", Path: "/tmp/ui-native", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "native ui runtime", Kind: "runtime", Path: "tools/cmd/native-ui-runtime-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		},
		Widgets: widgets,
		Events:  events,
		Cases:   cases,
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := nativeui.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if events[1].AfterState["ShellState.toggles"] != "2" {
		t.Fatalf("second click after_state toggles = %q, want 2", events[1].AfterState["ShellState.toggles"])
	}
}

func TestNativeRuntimeRejectsInvalidDispatchPaths(t *testing.T) {
	rt, err := loadNativeRuntime(nativeRuntimeSmokeFixture())
	if err != nil {
		t.Fatalf("loadNativeRuntime failed: %v", err)
	}
	if _, err := rt.dispatch("__missing__", "click", "", 1); err == nil || !strings.Contains(err.Error(), "unknown widget") {
		t.Fatalf("invalid widget error = %v, want unknown widget", err)
	}
	if _, err := rt.dispatch("ShellView.submit", "hover", "", 1); err == nil || !strings.Contains(err.Error(), "unsupported event") {
		t.Fatalf("unsupported event error = %v, want unsupported event", err)
	}
	if _, err := rt.dispatch("ShellView.submit", "click", "__missing__", 1); err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("unknown command error = %v, want unknown command", err)
	}
}

func nativeRuntimeSmokeFixture() []byte {
	return []byte(`{
  "schema": "tetra.ui.native-shell.v1",
  "ui_schema": "tetra.ui.v0.4.0",
  "runtime": "native shell command dispatch",
  "states": [
    {"name":"ShellState","fields":[{"name":"toggles","type":"i32","mutable":true,"value":"0"}]}
  ],
  "views": [
    {
      "name":"ShellView",
      "state_type":"ShellState",
      "bindings":[{"name":"toggles","type":"i32","value":"0"}],
      "widgets":[
        {"id":"ShellView.toggles","kind":"value","binding":"toggles","type":"i32","value":"0"},
        {"id":"ShellView.submit","kind":"action","event":"submit","command":"toggle"}
      ],
      "events":[
        {"name":"submit","command":"toggle","operations":[{"kind":"state_add","target":"state.toggles","value":"1","state_field":"toggles","state_value":"1"}],"bindings":[{"name":"toggles","type":"i32","value":"1"}]}
      ]
    }
  ]
}`)
}
