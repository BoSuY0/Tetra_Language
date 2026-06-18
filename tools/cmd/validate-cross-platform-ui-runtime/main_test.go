package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCrossPlatformUIRuntimeAcceptsAllPlatforms(t *testing.T) {
	dir := t.TempDir()
	inputs := writeCrossPlatformReports(t, dir)

	if err := validateCrossPlatformUIRuntime(inputs); err != nil {
		t.Fatalf("validateCrossPlatformUIRuntime: %v", err)
	}
}

func TestValidateCrossPlatformUIRuntimeRejectsBlockedTargetReport(t *testing.T) {
	dir := t.TempDir()
	inputs := writeCrossPlatformReports(t, dir)
	blockedWindows := strings.Replace(
		validPlatformUIReport("windows-x64"),
		`"status":"pass"`,
		`"status":"blocked"`,
		1,
	)
	blockedWindows = strings.Replace(
		blockedWindows,
		`"host":"windows-x64"`,
		`"host":"linux-x64"`,
		1,
	)
	writeFile(t, inputs.Windows, blockedWindows)

	err := validateCrossPlatformUIRuntime(inputs)
	if err == nil {
		t.Fatal("expected blocked Windows report to fail")
	}
	for _, want := range []string{"windows", "status", "host"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}

func TestValidateCrossPlatformUIRuntimeRejectsStaleTargetHostReport(t *testing.T) {
	dir := t.TempDir()
	inputs := writeCrossPlatformReports(t, dir)
	staleWindows := strings.Replace(
		validPlatformUIReport("windows-x64"),
		`"git_head":"abcdef1234567890"`,
		`"git_head":"stale123"`,
		1,
	)
	writeFile(t, inputs.Windows, staleWindows)

	err := validateCrossPlatformUIRuntimeWithOptions(inputs, validationOptions{
		ExpectedVersion: "v0.4.0",
		ExpectedGitHead: "abcdef1234567890",
	})
	if err == nil {
		t.Fatal("expected stale Windows report to fail")
	}
	for _, want := range []string{"windows", "git_head"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}

func writeCrossPlatformReports(t *testing.T, dir string) crossPlatformInputs {
	t.Helper()
	inputs := crossPlatformInputs{
		Linux:   filepath.Join(dir, "linux.json"),
		Windows: filepath.Join(dir, "windows.json"),
		MacOS:   filepath.Join(dir, "macos.json"),
		Web:     filepath.Join(dir, "web.json"),
	}
	writeFile(
		t,
		inputs.Linux,
		`{"schema":"tetra.ui.desktop-runtime.v1","status":"pass","target":"linux-x64","runtime":"desktop-ui-linux-x64","ui_schema":"tetra.ui.v1"}`,
	)
	writeFile(t, inputs.Windows, validPlatformUIReport("windows-x64"))
	writeFile(t, inputs.MacOS, validPlatformUIReport("macos-x64"))
	writeFile(
		t,
		inputs.Web,
		`{"schema":"tetra.web-ui-smoke.v1alpha1","status":"pass","target":"wasm32-web","ui_schema":"tetra.ui.v1","ui_bundle_path":"web-smoke.ui.json","ui_module_path":"web-smoke.ui.web.mjs","dom_snapshot":"web-smoke.dom.html","runtime_trace":"window-mount:ok;root-mount:ok;layout:ok;text:ok;button:ok;input:ok;list:ok;panel:ok;focus:ok;input-event:ok;change:ok;select:ok;click:ok;timer:ok;async-command:ok;redraw-update:ok;error-recovery:ok;ui-event-dispatch:web-command-dispatch"}`,
	)
	return inputs
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func validPlatformUIReport(target string) string {
	return `{"schema":"tetra.ui.platform-runtime.v1","status":"pass","version":"v0.4.0","git_head":"abcdef1234567890","target":"` + target + `","host":"` + target + `","runtime":"platform-ui-` + target + `","runtime_trace":"platform-process-spawn:ok;platform-window-api:ok;platform-widget-tree:ok;platform-event-dispatch:ok;platform-timer:ok;platform-redraw:ok;window-create:ok;window-show:ok;widget-tree-load:ok;layout-measure:ok;layout-place:ok;event-loop-start:ok;focus-dispatch:ok;input-dispatch:ok;select-dispatch:ok;click-dispatch:ok;state-update:ok;async-command:ok;timer-tick:ok;redraw:ok;error-recovery:ok;window-close:ok","ui_schema":"tetra.ui.v1","source":"tools/cmd/platform-ui-runtime-smoke","runner":"target-host-runtime-child","processes":[{"name":"build","kind":"build","path":"go build ./tools/cmd/platform-ui-runtime-smoke","ran":true,"pass":true,"exit_code":0},{"name":"app","kind":"app","path":"platform-ui-runtime-smoke --child-runtime --target ` + target + `","ran":true,"pass":true,"exit_code":0},{"name":"runtime","kind":"runtime","path":"child-runtime event loop","ran":true,"pass":true,"exit_code":0},{"name":"stress","kind":"stress","path":"child-runtime stress sweep","ran":true,"pass":true,"exit_code":0}],"widgets":[{"id":"AppWindow","kind":"window","enabled":true,"visible":true,"bounds":{"width":640,"height":480}},{"id":"RootPanel","kind":"panel","enabled":true,"visible":true,"bounds":{"width":624,"height":464}},{"id":"TitleText","kind":"text","enabled":true,"visible":true,"bounds":{"width":608,"height":32}},{"id":"NameInput","kind":"input","enabled":true,"visible":true,"bounds":{"width":608,"height":32}},{"id":"ItemList","kind":"list","enabled":true,"visible":true,"bounds":{"width":608,"height":240}},{"id":"SaveButton","kind":"button","enabled":true,"visible":true,"bounds":{"width":200,"height":44}}],"events":[{"order":1,"widget_id":"NameInput","event":"focus","command":"focusName","pass":true,"before_state":{"focused":"none"},"after_state":{"focused":"NameInput"},"operations":[{"kind":"focus"}],"widget_updates":[{"id":"NameInput","before":"blurred","after":"focused"}]},{"order":2,"widget_id":"NameInput","event":"input","command":"setName","pass":true,"before_state":{"name":"tetra"},"after_state":{"name":"tetra-ui"},"operations":[{"kind":"state_set"}],"widget_updates":[{"id":"NameInput","before":"tetra","after":"tetra-ui"}]},{"order":3,"widget_id":"ItemList","event":"select","command":"selectItem","pass":true,"before_state":{"selected":"item-1"},"after_state":{"selected":"item-2"},"operations":[{"kind":"state_set"}],"widget_updates":[{"id":"ItemList","before":"item-1","after":"item-2"}]},{"order":4,"widget_id":"SaveButton","event":"click","command":"saveAsync","pass":true,"before_state":{"saved":"false"},"after_state":{"saved":"true"},"operations":[{"kind":"async_command"},{"kind":"redraw"}],"widget_updates":[{"id":"TitleText","before":"Editing","after":"Saved"}]},{"order":5,"widget_id":"AppWindow","event":"tick","command":"timerTick","pass":true,"before_state":{"dirty":"true"},"after_state":{"dirty":"false"},"operations":[{"kind":"timer_tick"},{"kind":"redraw"}],"widget_updates":[{"id":"TitleText","before":"Saved","after":"Saved after timer"}]}],"cases":[{"name":"window lifecycle","kind":"positive","ran":true,"pass":true},{"name":"layout measure and place","kind":"positive","ran":true,"pass":true},{"name":"widget tree load","kind":"positive","ran":true,"pass":true},{"name":"event loop dispatch","kind":"positive","ran":true,"pass":true},{"name":"state binding update","kind":"positive","ran":true,"pass":true},{"name":"redraw update lifecycle","kind":"positive","ran":true,"pass":true},{"name":"async UI command completion","kind":"positive","ran":true,"pass":true},{"name":"timer scheduled redraw","kind":"positive","ran":true,"pass":true},{"name":"invalid widget diagnostic","kind":"negative","ran":true,"pass":true,"expected_error":"unknown widget"},{"name":"command failure recovery","kind":"negative","ran":true,"pass":true,"expected_error":"unknown command"},{"name":"crash error handling","kind":"negative","ran":true,"pass":true,"expected_error":"runtime panic recovered"}],"audit":[{"requirement":"real platform runtime evidence","artifact":"target-host-runtime child process","evidence":"target-host child process executed runtime loop, widgets, events, cases executed","result":"pass"},{"requirement":"reject runtime-less evidence","artifact":"tools/validators/platformui","evidence":"validator rejects runtime-less evidence","result":"pass"}]}`
}
