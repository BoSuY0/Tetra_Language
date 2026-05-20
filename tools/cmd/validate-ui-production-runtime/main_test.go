package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateUIProductionRuntimeReportAcceptsValidReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ui-production.json")
	if err := os.WriteFile(path, []byte(validUIProductionRuntimeReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateUIProductionRuntimeReport(path); err != nil {
		t.Fatalf("validateUIProductionRuntimeReport failed: %v", err)
	}
}

func TestValidateUIProductionRuntimeReportRejectsInvalidReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ui-production.json")
	raw := strings.Replace(validUIProductionRuntimeReport(), `"schema": "tetra.ui.desktop-runtime.v1"`, `"schema": "tetra.ui.desktop-fake.v1"`, 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateUIProductionRuntimeReport(path)
	if err == nil {
		t.Fatalf("expected invalid UI production runtime report to fail")
	}
	if !strings.Contains(err.Error(), "tetra.ui.desktop-runtime.v1") {
		t.Fatalf("error = %v, want schema rejection", err)
	}
}

func validUIProductionRuntimeReport() string {
	return `{
  "schema": "tetra.ui.desktop-runtime.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "desktop-ui-linux-x64",
  "ui_schema": "tetra.ui.v1",
  "source": "tools/cmd/ui-production-runtime-smoke",
  "processes": [
    {"name":"tetra build","kind":"build","path":"/tmp/tetra","ran":true,"pass":true,"exit_code":0},
    {"name":"desktop UI app","kind":"app","path":"/tmp/ui-desktop","ran":true,"pass":true,"exit_code":0},
    {"name":"desktop UI runtime","kind":"runtime","path":"tools/cmd/ui-production-runtime-smoke","ran":true,"pass":true,"exit_code":0},
    {"name":"native shell runtime integration","kind":"runtime","path":"go run ./tools/cmd/native-ui-runtime-smoke","ran":true,"pass":true,"exit_code":0},
    {"name":"native runtime evidence validator","kind":"runtime","path":"go run ./tools/cmd/validate-native-ui-runtime","ran":true,"pass":true,"exit_code":0},
    {"name":"desktop UI widget stress","kind":"stress","path":"/tmp/ui-widget-stress","ran":true,"pass":true,"exit_code":0}
  ],
  "contracts": [
    {"name":"Linux-x64 desktop UI runtime","status":"pass","evidence":"desktop UI and sidecar-driven native runtime process evidence ran on linux-x64"},
    {"name":"window lifecycle","status":"pass","evidence":"window create, show, close, and teardown are covered"},
    {"name":"layout system","status":"pass","evidence":"layout measure/place and panel nesting cases ran"},
    {"name":"buttons text input lists panels state binding","status":"pass","evidence":"button, text, input, focus/change, list, panel, and bound state widgets are present"},
    {"name":"event loop","status":"pass","evidence":"focus, input, change, select, click, and timer events ran through the runtime"},
    {"name":"async UI commands","status":"pass","evidence":"async command completion case runs through the UI runtime"},
    {"name":"timers","status":"pass","evidence":"timer scheduled redraw case records a real timer tick event and timer_tick operation"},
    {"name":"redraw update model","status":"pass","evidence":"redraw/update lifecycle case records dirty state to redraw"},
    {"name":"error crash handling","status":"pass","evidence":"invalid widget, command failure recovery, and crash handling cases are required"},
    {"name":"real dogfood applications","status":"pass","evidence":"dogfood application smoke case uses real Tetra UI source"}
  ],
  "widgets": [
    {"id":"AppWindow","kind":"window","parent":"","binding":"app.open","enabled":true,"visible":true,"bounds":{"x":0,"y":0,"width":640,"height":480}},
    {"id":"RootPanel","kind":"panel","parent":"AppWindow","binding":"layout.root","enabled":true,"visible":true,"bounds":{"x":0,"y":0,"width":640,"height":480}},
    {"id":"TitleText","kind":"text","parent":"RootPanel","binding":"state.title","value":"Saved after timer","enabled":true,"visible":true,"bounds":{"x":16,"y":16,"width":608,"height":32}},
    {"id":"NameInput","kind":"input","parent":"RootPanel","binding":"state.name","event":"input","value":"tetra-prod","enabled":true,"visible":true,"bounds":{"x":16,"y":64,"width":608,"height":32}},
    {"id":"ItemList","kind":"list","parent":"RootPanel","binding":"state.items","event":"select","value":"item-1","enabled":true,"visible":true,"bounds":{"x":16,"y":112,"width":608,"height":240}},
    {"id":"SaveButton","kind":"button","parent":"RootPanel","binding":"state.saved","event":"click","command":"saveAsync","enabled":true,"visible":true,"bounds":{"x":16,"y":368,"width":200,"height":44}}
  ],
  "events": [
    {"order":1,"widget_id":"NameInput","event":"focus","command":"focusName","pass":true,"before_state":{"AppState.focused":"none"},"after_state":{"AppState.focused":"NameInput"},"operations":[{"kind":"focus","target":"widget.NameInput","value":"focused","state_field":"focused","state_value":"NameInput"}],"widget_updates":[{"id":"TitleText","before":"Ready","after":"Editing name"}]},
    {"order":2,"widget_id":"NameInput","event":"input","command":"setName","pass":true,"before_state":{"AppState.name":"tetra"},"after_state":{"AppState.name":"tetra-lang"},"operations":[{"kind":"state_set","target":"state.name","value":"tetra-lang","state_field":"name","state_value":"tetra-lang"}],"widget_updates":[{"id":"NameInput","before":"tetra","after":"tetra-lang"}]},
    {"order":3,"widget_id":"NameInput","event":"change","command":"commitName","pass":true,"before_state":{"AppState.name":"tetra-lang","AppState.changed":"false"},"after_state":{"AppState.name":"tetra-prod","AppState.changed":"true"},"operations":[{"kind":"change","target":"state.name","value":"tetra-prod","state_field":"name","state_value":"tetra-prod"},{"kind":"state_set","target":"state.changed","value":"true","state_field":"changed","state_value":"true"}],"widget_updates":[{"id":"NameInput","before":"tetra-lang","after":"tetra-prod"}]},
    {"order":4,"widget_id":"ItemList","event":"select","command":"selectItem","pass":true,"before_state":{"AppState.selected":"item-1"},"after_state":{"AppState.selected":"item-2"},"operations":[{"kind":"state_set","target":"state.selected","value":"item-2","state_field":"selected","state_value":"item-2"}],"widget_updates":[{"id":"ItemList","before":"item-1","after":"item-2"}]},
    {"order":5,"widget_id":"SaveButton","event":"click","command":"saveAsync","pass":true,"before_state":{"AppState.saved":"false"},"after_state":{"AppState.saved":"true"},"operations":[{"kind":"async_command","target":"command.saveAsync","value":"completed","state_field":"saved","state_value":"true"},{"kind":"redraw","target":"AppWindow","value":"scheduled","state_field":"dirty","state_value":"false"}],"widget_updates":[{"id":"TitleText","before":"Editing name","after":"Saved"}]},
    {"order":6,"widget_id":"AppWindow","event":"tick","command":"timerTick","pass":true,"before_state":{"AppState.dirty":"true"},"after_state":{"AppState.dirty":"false"},"operations":[{"kind":"timer_tick","target":"timer.redraw","value":"fired","state_field":"dirty","state_value":"false"},{"kind":"redraw","target":"AppWindow","value":"completed","state_field":"dirty","state_value":"false"}],"widget_updates":[{"id":"TitleText","before":"Saved","after":"Saved after timer"}]}
  ],
  "cases": [
    {"name":"window lifecycle","kind":"positive","ran":true,"pass":true},
    {"name":"layout measure and place","kind":"positive","ran":true,"pass":true},
    {"name":"button command dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"text render","kind":"positive","ran":true,"pass":true},
    {"name":"input focus traversal","kind":"positive","ran":true,"pass":true},
    {"name":"input edit","kind":"positive","ran":true,"pass":true},
    {"name":"input change commit","kind":"positive","ran":true,"pass":true},
    {"name":"list selection","kind":"positive","ran":true,"pass":true},
    {"name":"panel nesting","kind":"positive","ran":true,"pass":true},
    {"name":"state binding update","kind":"positive","ran":true,"pass":true},
    {"name":"event loop dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"async UI command completion","kind":"positive","ran":true,"pass":true},
    {"name":"timer scheduled redraw","kind":"positive","ran":true,"pass":true},
    {"name":"redraw update lifecycle","kind":"positive","ran":true,"pass":true},
    {"name":"compiler UI bundle runtime load","kind":"positive","ran":true,"pass":true},
    {"name":"native shell runtime integration","kind":"positive","ran":true,"pass":true},
    {"name":"native runtime sidecar consistency","kind":"positive","ran":true,"pass":true},
    {"name":"invalid widget diagnostic","kind":"negative","ran":true,"pass":true,"expected_error":"unknown widget"},
    {"name":"command failure recovery","kind":"negative","ran":true,"pass":true,"expected_error":"command failed"},
    {"name":"crash error handling","kind":"negative","ran":true,"pass":true,"expected_error":"runtime panic recovered"},
    {"name":"dogfood application smoke","kind":"positive","ran":true,"pass":true},
    {"name":"widget tree stress","kind":"stress","ran":true,"pass":true}
  ],
  "audit": [
    {"requirement":"Linux-x64 desktop UI runtime","artifact":"tools/cmd/ui-production-runtime-smoke; compiler/internal/backend/native_shell","evidence":"build, app, desktop runtime, native runtime, stress, and compiler-emitted UI bundle load evidence ran on linux-x64","result":"pass"},
    {"requirement":"window lifecycle","artifact":"examples/ui_desktop_runtime_smoke.tetra","evidence":"window create, show, close, and teardown cases are required","result":"pass"},
    {"requirement":"layout system","artifact":"compiler/internal/lower/ui.go; docs/spec/ui_v1.md","evidence":"layout measure/place and panel nesting cases are required","result":"pass"},
    {"requirement":"buttons/text/input/lists/panels widgets","artifact":"examples/ui_desktop_runtime_smoke.tetra","evidence":"widget tree must include button, text, input, list, and panel widgets","result":"pass"},
    {"requirement":"state binding","artifact":"tools/validators/uiprod","evidence":"state binding update plus input focus/change widget update evidence are required","result":"pass"},
    {"requirement":"event loop and redraw/update model","artifact":"tools/cmd/ui-production-runtime-smoke","evidence":"focus, input, change, select, click, timer, and redraw/update lifecycle cases are required","result":"pass"},
    {"requirement":"async commands and timers","artifact":"tools/cmd/ui-production-runtime-smoke","evidence":"async UI command completion, timer tick event evidence, and timer scheduled redraw cases are required","result":"pass"},
    {"requirement":"error/crash handling","artifact":"tools/validators/uiprod","evidence":"invalid widget diagnostic, command failure recovery, and crash error handling cases are required","result":"pass"},
    {"requirement":"real examples and dogfood applications","artifact":"examples/ui_desktop_runtime_smoke.tetra; examples/ui_native_shell_smoke.tetra","evidence":"dogfood application smoke, compiler-emitted UI bundle/runtime trace load, and native runtime integration cases are required","result":"pass"},
    {"requirement":"compiler-emitted UI bundle/native-shell trace load evidence","artifact":"examples/ui_desktop_runtime_smoke.tetra; <output>.ui.json; <output>.ui.shell.json","evidence":"UI production smoke loads compiler-emitted tetra.ui.v1 and tetra.ui.native-shell.v1 artifacts before accepting runtime evidence","result":"pass"},
    {"requirement":"sidecar-driven native UI runtime integration","artifact":"tools/cmd/native-ui-runtime-smoke; tools/cmd/validate-native-ui-runtime; native-ui-runtime-linux-x64.integration.json","evidence":"UI production smoke runs the sidecar-driven native UI runtime and validates tetra.ui.native-runtime.v1 consistency before accepting the release gate","result":"pass"},
    {"requirement":"stable UI diagnostics","artifact":"tools/cmd/ui-production-runtime-smoke; tools/validators/uiprod","evidence":"negative UI cases require stable expected_error evidence for invalid widget diagnostics, command failure recovery, and crash error handling","result":"pass"},
    {"requirement":"release-gate entrypoint rejecting runtime-less evidence","artifact":"scripts/release/post_v0_4/ui-production-runtime-linux-x64-smoke.sh","evidence":"validator rejects metadata-only, runtime-less, fake, mock, placeholder, docs-only, and build-only evidence and requires compiler UI bundle plus native runtime integration evidence","result":"pass"}
  ]
}`
}
