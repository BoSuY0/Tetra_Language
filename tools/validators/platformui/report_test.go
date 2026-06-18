package platformui

import (
	"strings"
	"testing"
)

func TestValidateReportAcceptsTargetHostRuntimeEvidence(t *testing.T) {
	for _, target := range []string{"windows-x64", "macos-x64"} {
		t.Run(target, func(t *testing.T) {
			if err := ValidateReport([]byte(validPlatformUIReport(target)), target); err != nil {
				t.Fatalf("ValidateReport(%s): %v", target, err)
			}
		})
	}
}

func TestValidateReportRejectsBlockedRuntimeLessEvidence(t *testing.T) {
	raw := strings.Replace(
		validPlatformUIReport("windows-x64"),
		`"status":"pass"`,
		`"status":"blocked"`,
		1,
	)
	raw = strings.Replace(raw, `"host":"windows-x64"`, `"host":"linux-x64"`, 1)
	raw = strings.Replace(
		raw,
		`"source":"tools/cmd/platform-ui-runtime-smoke"`,
		`"source":"docs-only-runtime-less-placeholder.md"`,
		1,
	)
	err := ValidateReport([]byte(raw), "windows-x64")
	if err == nil {
		t.Fatalf("expected blocked/runtime-less report to fail")
	}
	for _, want := range []string{"status", "host", "runtime-less"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}

func TestValidateReportRejectsBuildOnlyEvidence(t *testing.T) {
	raw := `{"schema":"tetra.ui.platform-runtime.v1","status":"pass","target":"windows-x64","host":"windows-x64","runtime":"platform-ui-windows-x64","ui_schema":"tetra.ui.v0.4.0","source":"build-only","runner":"target-host-runtime-child"}`
	err := ValidateReport([]byte(raw), "windows-x64")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "build-only") {
		t.Fatalf("build-only report error = %v", err)
	}
}

func TestValidateReportRejectsNonChildRunner(t *testing.T) {
	raw := strings.Replace(
		validPlatformUIReport("windows-x64"),
		`"runner":"target-host-runtime-child"`,
		`"runner":"host-native"`,
		1,
	)
	err := ValidateReport([]byte(raw), "windows-x64")
	if err == nil || !strings.Contains(err.Error(), "target-host-runtime-child") {
		t.Fatalf("non-child runner error = %v", err)
	}
}

func TestValidateReportWithOptionsRejectsStaleVersionAndGitHead(t *testing.T) {
	raw := strings.Replace(
		validPlatformUIReport("windows-x64"),
		`"version":"v0.4.0"`,
		`"version":"v0.3.0"`,
		1,
	)
	raw = strings.Replace(raw, `"git_head":"abcdef1234567890"`, `"git_head":"stale123"`, 1)
	err := ValidateReportWithOptions([]byte(raw), ValidateOptions{
		ExpectedTarget:  "windows-x64",
		ExpectedVersion: "v0.4.0",
		ExpectedGitHead: "abcdef1234567890",
	})
	if err == nil {
		t.Fatalf("expected stale version/git_head report to fail")
	}
	for _, want := range []string{"version", "git_head"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}

func TestValidateReportRejectsMissingRuntimeTraceMarkers(t *testing.T) {
	raw := strings.Replace(
		validPlatformUIReport("windows-x64"),
		`"runtime_trace":"platform-process-spawn:ok;platform-window-api:ok;platform-widget-tree:ok;platform-event-dispatch:ok;platform-timer:ok;platform-redraw:ok;window-create:ok;window-show:ok;widget-tree-load:ok;layout-measure:ok;layout-place:ok;event-loop-start:ok;focus-dispatch:ok;input-dispatch:ok;select-dispatch:ok;click-dispatch:ok;state-update:ok;async-command:ok;timer-tick:ok;redraw:ok;error-recovery:ok;window-close:ok"`,
		`"runtime_trace":"platform-process-spawn:ok;widget-tree-load:ok"`,
		1,
	)
	err := ValidateReport([]byte(raw), "windows-x64")
	if err == nil {
		t.Fatalf("expected missing runtime_trace markers to fail")
	}
	for _, want := range []string{"runtime_trace", "window-create", "error-recovery"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}

func TestValidateReportRejectsWindowOnlyPlatformProbe(t *testing.T) {
	raw := strings.Replace(
		validPlatformUIReport("windows-x64"),
		"platform-widget-tree:ok;platform-event-dispatch:ok;platform-timer:ok;platform-redraw:ok;",
		"",
		1,
	)
	err := ValidateReport([]byte(raw), "windows-x64")
	if err == nil {
		t.Fatalf("expected window-only platform probe to fail")
	}
	for _, want := range []string{
		"platform-widget-tree:ok",
		"platform-event-dispatch:ok",
		"platform-timer:ok",
		"platform-redraw:ok",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}

func validPlatformUIReport(target string) string {
	return `{"schema":"tetra.ui.platform-runtime.v1","status":"pass","version":"v0.4.0","git_head":"abcdef1234567890","target":"` + target + `","host":"` + target + `","runtime":"platform-ui-` + target + `","runtime_trace":"platform-process-spawn:ok;platform-window-api:ok;platform-widget-tree:ok;platform-event-dispatch:ok;platform-timer:ok;platform-redraw:ok;window-create:ok;window-show:ok;widget-tree-load:ok;layout-measure:ok;layout-place:ok;event-loop-start:ok;focus-dispatch:ok;input-dispatch:ok;select-dispatch:ok;click-dispatch:ok;state-update:ok;async-command:ok;timer-tick:ok;redraw:ok;error-recovery:ok;window-close:ok","ui_schema":"tetra.ui.v0.4.0","source":"tools/cmd/platform-ui-runtime-smoke","runner":"target-host-runtime-child","processes":[{"name":"build","kind":"build","path":"go build ./tools/cmd/platform-ui-runtime-smoke","ran":true,"pass":true,"exit_code":0},{"name":"app","kind":"app","path":"platform-ui-runtime-smoke --child-runtime --target ` + target + `","ran":true,"pass":true,"exit_code":0},{"name":"runtime","kind":"runtime","path":"child-runtime event loop","ran":true,"pass":true,"exit_code":0},{"name":"stress","kind":"stress","path":"child-runtime stress sweep","ran":true,"pass":true,"exit_code":0}],"widgets":[{"id":"AppWindow","kind":"window","enabled":true,"visible":true,"bounds":{"width":640,"height":480}},{"id":"RootPanel","kind":"panel","enabled":true,"visible":true,"bounds":{"width":624,"height":464}},{"id":"TitleText","kind":"text","enabled":true,"visible":true,"bounds":{"width":608,"height":32}},{"id":"NameInput","kind":"input","enabled":true,"visible":true,"bounds":{"width":608,"height":32}},{"id":"ItemList","kind":"list","enabled":true,"visible":true,"bounds":{"width":608,"height":240}},{"id":"SaveButton","kind":"button","enabled":true,"visible":true,"bounds":{"width":200,"height":44}}],"events":[{"order":1,"widget_id":"NameInput","event":"focus","command":"focusName","pass":true,"before_state":{"focused":"none"},"after_state":{"focused":"NameInput"},"operations":[{"kind":"focus"}],"widget_updates":[{"id":"NameInput","before":"blurred","after":"focused"}]},{"order":2,"widget_id":"NameInput","event":"input","command":"setName","pass":true,"before_state":{"name":"tetra"},"after_state":{"name":"tetra-ui"},"operations":[{"kind":"state_set"}],"widget_updates":[{"id":"NameInput","before":"tetra","after":"tetra-ui"}]},{"order":3,"widget_id":"ItemList","event":"select","command":"selectItem","pass":true,"before_state":{"selected":"item-1"},"after_state":{"selected":"item-2"},"operations":[{"kind":"state_set"}],"widget_updates":[{"id":"ItemList","before":"item-1","after":"item-2"}]},{"order":4,"widget_id":"SaveButton","event":"click","command":"saveAsync","pass":true,"before_state":{"saved":"false"},"after_state":{"saved":"true"},"operations":[{"kind":"async_command"},{"kind":"redraw"}],"widget_updates":[{"id":"TitleText","before":"Editing","after":"Saved"}]},{"order":5,"widget_id":"AppWindow","event":"tick","command":"timerTick","pass":true,"before_state":{"dirty":"true"},"after_state":{"dirty":"false"},"operations":[{"kind":"timer_tick"},{"kind":"redraw"}],"widget_updates":[{"id":"TitleText","before":"Saved","after":"Saved after timer"}]}],"cases":[{"name":"window lifecycle","kind":"positive","ran":true,"pass":true},{"name":"layout measure and place","kind":"positive","ran":true,"pass":true},{"name":"widget tree load","kind":"positive","ran":true,"pass":true},{"name":"event loop dispatch","kind":"positive","ran":true,"pass":true},{"name":"state binding update","kind":"positive","ran":true,"pass":true},{"name":"redraw update lifecycle","kind":"positive","ran":true,"pass":true},{"name":"async UI command completion","kind":"positive","ran":true,"pass":true},{"name":"timer scheduled redraw","kind":"positive","ran":true,"pass":true},{"name":"invalid widget diagnostic","kind":"negative","ran":true,"pass":true,"expected_error":"unknown widget"},{"name":"command failure recovery","kind":"negative","ran":true,"pass":true,"expected_error":"unknown command"},{"name":"crash error handling","kind":"negative","ran":true,"pass":true,"expected_error":"runtime panic recovered"}],"audit":[{"requirement":"real platform runtime evidence","artifact":"target-host-runtime child process","evidence":"target-host child process executed runtime loop, widgets, events, cases executed","result":"pass"},{"requirement":"reject runtime-less evidence","artifact":"tools/validators/platformui","evidence":"validator rejects runtime-less evidence","result":"pass"}]}`
}
