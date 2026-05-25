package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateCrossPlatformUIRuntimeAcceptsAllRuntimeBackedReports(t *testing.T) {
	dir := t.TempDir()
	linux := filepath.Join(dir, "linux.json")
	windows := filepath.Join(dir, "windows.json")
	macos := filepath.Join(dir, "macos.json")
	web := filepath.Join(dir, "web.json")
	writeFile(t, linux, validLinuxUIProductionRuntimeReport())
	writeFile(t, windows, validCrossPlatformPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"))
	writeFile(t, macos, validCrossPlatformPlatformReport("macos-x64", "macos", "platform-ui-macos-x64"))
	writeFile(t, web, validCrossPlatformWebReport(t, dir))

	if err := validateCrossPlatformUIRuntime(crossPlatformInputs{
		Linux: linux, Windows: windows, MacOS: macos, Web: web,
	}); err != nil {
		t.Fatalf("validateCrossPlatformUIRuntime: %v", err)
	}
}

func TestValidateCrossPlatformUIRuntimeRejectsBlockedWindows(t *testing.T) {
	dir := t.TempDir()
	linux := filepath.Join(dir, "linux.json")
	windows := filepath.Join(dir, "windows.json")
	macos := filepath.Join(dir, "macos.json")
	web := filepath.Join(dir, "web.json")
	writeFile(t, linux, validLinuxUIProductionRuntimeReport())
	writeFile(t, windows, strings.Replace(validCrossPlatformPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"), `"status": "pass"`, `"status": "blocked"`, 1))
	writeFile(t, macos, validCrossPlatformPlatformReport("macos-x64", "macos", "platform-ui-macos-x64"))
	writeFile(t, web, validCrossPlatformWebReport(t, dir))

	err := validateCrossPlatformUIRuntime(crossPlatformInputs{
		Linux: linux, Windows: windows, MacOS: macos, Web: web,
	})
	if err == nil || !strings.Contains(err.Error(), "windows") {
		t.Fatalf("expected windows blocker rejection, got %v", err)
	}
}

func TestValidateCrossPlatformUIRuntimeAggregatesPlatformBlockers(t *testing.T) {
	dir := t.TempDir()
	linux := filepath.Join(dir, "linux.json")
	windows := filepath.Join(dir, "windows.json")
	macos := filepath.Join(dir, "macos.json")
	web := filepath.Join(dir, "web.json")
	writeFile(t, linux, validLinuxUIProductionRuntimeReport())
	writeFile(t, windows, strings.Replace(validCrossPlatformPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"), `"status": "pass"`, `"status": "blocked"`, 1))
	writeFile(t, macos, strings.Replace(validCrossPlatformPlatformReport("macos-x64", "macos", "platform-ui-macos-x64"), `"status": "pass"`, `"status": "blocked"`, 1))
	writeFile(t, web, validCrossPlatformWebReport(t, dir))

	err := validateCrossPlatformUIRuntime(crossPlatformInputs{
		Linux: linux, Windows: windows, MacOS: macos, Web: web,
	})
	if err == nil {
		t.Fatalf("expected aggregated platform blocker rejection")
	}
	text := err.Error()
	if !strings.Contains(text, "windows UI runtime evidence invalid") || !strings.Contains(text, "macos UI runtime evidence invalid") {
		t.Fatalf("expected windows and macos blockers, got %v", err)
	}
}

func TestValidateCrossPlatformUIRuntimeRejectsWebTraceWithoutPlatformMarkers(t *testing.T) {
	dir := t.TempDir()
	linux := filepath.Join(dir, "linux.json")
	windows := filepath.Join(dir, "windows.json")
	macos := filepath.Join(dir, "macos.json")
	web := filepath.Join(dir, "web.json")
	writeFile(t, linux, validLinuxUIProductionRuntimeReport())
	writeFile(t, windows, validCrossPlatformPlatformReport("windows-x64", "windows", "platform-ui-windows-x64"))
	writeFile(t, macos, validCrossPlatformPlatformReport("macos-x64", "macos", "platform-ui-macos-x64"))
	writeFile(t, web, strings.Replace(validCrossPlatformWebReport(t, dir), "window/root mount:ok;", "", 1))

	err := validateCrossPlatformUIRuntime(crossPlatformInputs{
		Linux: linux, Windows: windows, MacOS: macos, Web: web,
	})
	if err == nil || !strings.Contains(err.Error(), "window/root mount") {
		t.Fatalf("expected web runtime marker rejection, got %v", err)
	}
}

func TestValidateCrossPlatformUIRuntimeRejectsStaleWebReport(t *testing.T) {
	dir := t.TempDir()
	web := filepath.Join(dir, "web.json")
	writeFile(t, web, strings.Replace(validCrossPlatformWebReport(t, dir), webGeneratedAt(), "2000-01-01T00:00:00Z", 1))

	err := validateWeb(web)
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("expected stale web report rejection, got %v", err)
	}
}

func writeFile(t *testing.T, path string, raw string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
}

func validLinuxUIProductionRuntimeReport() string {
	return `{
  "schema": "tetra.ui.desktop-runtime.v1",
  "status": "pass",
  "target": "linux-x64",
  "host": "linux-x64",
  "runtime": "desktop-ui-linux-x64",
  "source": "tools/cmd/ui-production-runtime-smoke"
}`
}

func validCrossPlatformPlatformReport(target string, platform string, runtime string) string {
	return `{
  "schema": "tetra.ui.platform.v1",
  "generated_at": "` + time.Now().UTC().Format(time.RFC3339) + `",
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

func validCrossPlatformWebReport(t *testing.T, dir string) string {
	t.Helper()
	uiBundle := filepath.Join(dir, "web.ui.json")
	uiModule := filepath.Join(dir, "web.ui.web.mjs")
	dom := filepath.Join(dir, "web.dom.html")
	writeFile(t, uiBundle, `{"schema":"tetra.ui.v1","states":[{"name":"AppState","module":"main","fields":[{"name":"count","type":"Int","mutable":true,"init":"0"}]}],"views":[{"name":"AppView","module":"main","state_type":"AppState","bindings":[{"name":"count","type":"Int","source":"state.count"}],"events":[{"name":"click","command":"increment"}],"commands":[{"name":"increment","statement_count":1}]}]}`)
	writeFile(t, uiModule, `export async function mountTetraUI(){ return {schema: "tetra.ui.v1", views: []}; }`)
	writeFile(t, dom, `<html><body><main data-tetra-ui-root="true"><button>Increment</button><input value="tetra"><ul><li>item</li></ul><section>panel</section></main></body></html>`)
	trace := "window/root mount:ok;layout:ok;text:ok;button:ok;input:ok;list:ok;panel:ok;focus:ok;change:ok;select:ok;click:ok;timer:ok;async command:ok;redraw/update:ok;error recovery:ok;main-exit:ok;stdout:ok;nonzero-exit:ok;failure-propagation:ok;repeated-instantiation:ok;ui-event-dispatch:web-command-dispatch"
	return `{
  "schema": "tetra.web-ui-smoke.v1alpha1",
  "generated_at": "` + webGeneratedAt() + `",
  "target": "wasm32-web",
  "ui_scope_active": true,
  "source": "examples/projects/dogfood_web_ui/src/main.tetra",
  "used_fallback_source": false,
  "automation": "chromium --headless --dump-dom",
  "status": "pass",
  "result": "ok:0:ui=1:runtime=ok",
  "runtime_trace": "` + trace + `",
  "blocker": "",
  "dom_snapshot": "` + filepath.ToSlash(dom) + `",
  "chromium_stderr": "",
  "ui_schema": "tetra.ui.v1",
  "ui_bundle_path": "` + filepath.ToSlash(uiBundle) + `",
  "ui_module_path": "` + filepath.ToSlash(uiModule) + `"
}`
}

func webGeneratedAt() string {
	return time.Now().UTC().Format(time.RFC3339)
}
