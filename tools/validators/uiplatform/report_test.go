package uiplatform

import (
	"strings"
	"testing"
	"time"
)

func TestValidateReportAcceptsWindowsTargetHostRuntimeEvidence(t *testing.T) {
	if err := ValidateReport([]byte(validPlatformReport("windows-x64", "windows", "platform-ui-windows-x64")), Options{
		Target:  "windows-x64",
		Host:    "windows-x64",
		Runtime: "platform-ui-windows-x64",
		Now:     mustTime(t, "2026-05-22T12:00:00Z"),
		MaxAge:  24 * time.Hour,
	}); err != nil {
		t.Fatalf("ValidateReport valid windows evidence: %v", err)
	}
}

func TestValidateReportAcceptsMacOSTargetHostRuntimeEvidence(t *testing.T) {
	if err := ValidateReport([]byte(validPlatformReport("macos-x64", "macos", "platform-ui-macos-x64")), Options{
		Target:  "macos-x64",
		Host:    "macos-x64",
		Runtime: "platform-ui-macos-x64",
		Now:     mustTime(t, "2026-05-22T12:00:00Z"),
		MaxAge:  24 * time.Hour,
	}); err != nil {
		t.Fatalf("ValidateReport valid macos evidence: %v", err)
	}
}

func TestValidateReportRejectsMissingOrStaleGeneratedAt(t *testing.T) {
	opts := Options{
		Target:  "windows-x64",
		Host:    "windows-x64",
		Runtime: "platform-ui-windows-x64",
		Now:     mustTime(t, "2026-05-22T12:00:00Z"),
		MaxAge:  24 * time.Hour,
	}
	raw := strings.Replace(validPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"), `  "generated_at": "2026-05-22T11:00:00Z",`+"\n", "", 1)
	err := ValidateReport([]byte(raw), opts)
	if err == nil || !strings.Contains(err.Error(), "generated_at is required") {
		t.Fatalf("expected missing generated_at rejection, got %v", err)
	}

	raw = strings.Replace(validPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"), `"generated_at": "2026-05-22T11:00:00Z"`, `"generated_at": "2026-05-20T11:00:00Z"`, 1)
	err = ValidateReport([]byte(raw), opts)
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("expected stale generated_at rejection, got %v", err)
	}
}

func TestValidateReportRejectsBlockedOrBuildOnlyEvidence(t *testing.T) {
	raw := strings.Replace(validPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"), `"status": "pass"`, `"status": "blocked"`, 1)
	err := ValidateReport([]byte(raw), Options{Target: "windows-x64", Host: "windows-x64", Runtime: "platform-ui-windows-x64"})
	if err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatalf("expected blocked report rejection, got %v", err)
	}

	raw = strings.Replace(validPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"), `"evidence_kind": "target-host-runtime"`, `"evidence_kind": "build-only"`, 1)
	err = ValidateReport([]byte(raw), Options{Target: "windows-x64", Host: "windows-x64", Runtime: "platform-ui-windows-x64"})
	if err == nil || !strings.Contains(err.Error(), "build-only") {
		t.Fatalf("expected build-only report rejection, got %v", err)
	}
}

func TestValidateReportRejectsFakeOrRuntimeLessEvidence(t *testing.T) {
	raw := strings.Replace(validPlatformReport("macos-x64", "macos", "platform-ui-macos-x64"), `"source": "examples/ui_desktop_runtime_smoke.tetra"`, `"source": "docs-only fake runtime-less placeholder"`, 1)
	err := ValidateReport([]byte(raw), Options{Target: "macos-x64", Host: "macos-x64", Runtime: "platform-ui-macos-x64"})
	if err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("expected fake/runtime-less rejection, got %v", err)
	}
}

func TestValidateReportRejectsFakeRuntimePathPrefix(t *testing.T) {
	raw := strings.Replace(validPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"), `"source": "examples/ui_desktop_runtime_smoke.tetra"`, `"source": "fake-runtime.exe"`, 1)
	err := ValidateReport([]byte(raw), Options{Target: "windows-x64", Host: "windows-x64", Runtime: "platform-ui-windows-x64"})
	if err == nil || !strings.Contains(err.Error(), "fake") {
		t.Fatalf("expected fake runtime path rejection, got %v", err)
	}
}

func validPlatformReport(target string, platform string, runtime string) string {
	return `{
  "schema": "tetra.ui.platform.v1",
  "generated_at": "2026-05-22T11:00:00Z",
  "status": "pass",
  "target": "` + target + `",
  "host": "` + target + `",
  "platform": "` + platform + `",
  "runtime": "` + runtime + `",
  "ui_schema": "tetra.ui.v1",
  "evidence_kind": "target-host-runtime",
  "source": "examples/ui_desktop_runtime_smoke.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"./tetra","ran":true,"pass":true,"exit_code":0},
    {"name":"platform UI app","kind":"app","path":"./ui-app","ran":true,"pass":true,"exit_code":0},
    {"name":"platform UI runtime","kind":"runtime","path":"./ui-runtime","ran":true,"pass":true,"exit_code":0},
    {"name":"platform UI stress","kind":"stress","path":"./ui-stress","ran":true,"pass":true,"exit_code":0}
  ],
  "contracts": [
    {"name":"window lifecycle","status":"pass","evidence":"target host process created, showed, closed, and tore down a UI window"},
    {"name":"widget tree","status":"pass","evidence":"target host runtime instantiated window, root, text, button, input, list, and panel widgets"},
    {"name":"layout","status":"pass","evidence":"target host runtime measured and placed widgets"},
    {"name":"event dispatch","status":"pass","evidence":"target host runtime dispatched focus, input, change, select, click, and timer events"},
    {"name":"state redraw async timers","status":"pass","evidence":"target host runtime updated state, redrew widgets, completed async command, and fired timer"},
    {"name":"negative diagnostics","status":"pass","evidence":"target host runtime rejected unsupported UI features with stable diagnostics"}
  ],
  "widgets": [
    {"id":"AppWindow","kind":"window","parent":"","binding":"app.open","enabled":true,"visible":true,"bounds":{"x":0,"y":0,"width":640,"height":480}},
    {"id":"RootPanel","kind":"panel","parent":"AppWindow","binding":"layout.root","enabled":true,"visible":true,"bounds":{"x":0,"y":0,"width":640,"height":480}},
    {"id":"TitleText","kind":"text","parent":"RootPanel","binding":"state.title","value":"Ready","enabled":true,"visible":true,"bounds":{"x":16,"y":16,"width":608,"height":32}},
    {"id":"NameInput","kind":"input","parent":"RootPanel","binding":"state.name","event":"input","value":"tetra","enabled":true,"visible":true,"bounds":{"x":16,"y":64,"width":608,"height":32}},
    {"id":"ItemList","kind":"list","parent":"RootPanel","binding":"state.items","event":"select","value":"item-1","enabled":true,"visible":true,"bounds":{"x":16,"y":112,"width":608,"height":240}},
    {"id":"SaveButton","kind":"button","parent":"RootPanel","binding":"state.saved","event":"click","command":"saveAsync","enabled":true,"visible":true,"bounds":{"x":16,"y":368,"width":200,"height":44}}
  ],
  "events": [
    {"order":1,"widget_id":"NameInput","event":"focus","command":"focusName","pass":true,"before_state":{"focused":"none"},"after_state":{"focused":"NameInput"},"operations":[{"kind":"focus","target":"widget.NameInput","value":"focused","state_field":"focused","state_value":"NameInput"}],"widget_updates":[{"id":"TitleText","before":"Ready","after":"Editing"}]},
    {"order":2,"widget_id":"NameInput","event":"input","command":"setName","pass":true,"before_state":{"name":"tetra"},"after_state":{"name":"tetra-lang"},"operations":[{"kind":"state_set","target":"state.name","value":"tetra-lang","state_field":"name","state_value":"tetra-lang"}],"widget_updates":[{"id":"NameInput","before":"tetra","after":"tetra-lang"}]},
    {"order":3,"widget_id":"NameInput","event":"change","command":"commitName","pass":true,"before_state":{"changed":"false"},"after_state":{"changed":"true"},"operations":[{"kind":"change","target":"state.changed","value":"true","state_field":"changed","state_value":"true"}],"widget_updates":[{"id":"TitleText","before":"Editing","after":"Changed"}]},
    {"order":4,"widget_id":"ItemList","event":"select","command":"selectItem","pass":true,"before_state":{"selected":"item-1"},"after_state":{"selected":"item-2"},"operations":[{"kind":"state_set","target":"state.selected","value":"item-2","state_field":"selected","state_value":"item-2"}],"widget_updates":[{"id":"ItemList","before":"item-1","after":"item-2"}]},
    {"order":5,"widget_id":"SaveButton","event":"click","command":"saveAsync","pass":true,"before_state":{"saved":"false"},"after_state":{"saved":"true"},"operations":[{"kind":"async_command","target":"command.saveAsync","value":"completed","state_field":"saved","state_value":"true"},{"kind":"redraw","target":"AppWindow","value":"scheduled","state_field":"dirty","state_value":"true"}],"widget_updates":[{"id":"TitleText","before":"Changed","after":"Saved"}]},
    {"order":6,"widget_id":"AppWindow","event":"tick","command":"timerTick","pass":true,"before_state":{"dirty":"true"},"after_state":{"dirty":"false"},"operations":[{"kind":"timer_tick","target":"timer.redraw","value":"fired","state_field":"dirty","state_value":"false"},{"kind":"redraw","target":"AppWindow","value":"completed","state_field":"dirty","state_value":"false"}],"widget_updates":[{"id":"TitleText","before":"Saved","after":"Saved after timer"}]}
  ],
  "cases": [
    {"name":"window lifecycle","kind":"positive","ran":true,"pass":true},
    {"name":"widget tree","kind":"positive","ran":true,"pass":true},
    {"name":"layout measure and place","kind":"positive","ran":true,"pass":true},
    {"name":"event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"state update redraw","kind":"positive","ran":true,"pass":true},
    {"name":"async command completion","kind":"positive","ran":true,"pass":true},
    {"name":"timer tick","kind":"positive","ran":true,"pass":true},
    {"name":"unsupported feature diagnostic","kind":"negative","ran":true,"pass":true,"expected_error":"unsupported UI feature"},
    {"name":"invalid widget diagnostic","kind":"negative","ran":true,"pass":true,"expected_error":"unknown widget"},
    {"name":"stress deterministic event loop","kind":"stress","ran":true,"pass":true}
  ]
}`
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
