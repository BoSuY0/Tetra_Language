package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateUIToolkitCoreReportAcceptsValidReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ui-toolkit-core.json")
	if err := os.WriteFile(path, []byte(validUIToolkitCoreReport(t)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateUIToolkitCoreReport(path); err != nil {
		t.Fatalf("validateUIToolkitCoreReport failed: %v", err)
	}
}

func TestValidateUIToolkitCoreReportRejectsInvalidReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ui-toolkit-core.json")
	raw := strings.Replace(validUIToolkitCoreReport(t), `"schema": "tetra.ui.toolkit.v1"`, `"schema": "tetra.ui.toolkit.fake.v1"`, 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateUIToolkitCoreReport(path)
	if err == nil {
		t.Fatalf("expected invalid UI toolkit core report to fail")
	}
	if !strings.Contains(err.Error(), "tetra.ui.toolkit.v1") {
		t.Fatalf("error = %v, want schema rejection", err)
	}
}

func validUIToolkitCoreReport(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bundle := filepath.Join(dir, "ui-toolkit-core.bundle.json")
	trace := filepath.Join(dir, "ui-toolkit-core.trace.json")
	for _, path := range []string{bundle, trace} {
		if err := os.WriteFile(path, []byte(`{"schema":"tetra.ui.toolkit.v1"}`+"\n"), 0o644); err != nil {
			t.Fatalf("write artifact %s: %v", path, err)
		}
	}
	return strings.ReplaceAll(strings.ReplaceAll(validUIToolkitCoreReportTemplate, "__BUNDLE__", bundle), "__TRACE__", trace)
}

const validUIToolkitCoreReportTemplate = `{
  "schema": "tetra.ui.toolkit.v1",
  "status": "pass",
  "target": "toolkit-core",
  "host": "linux-x64",
  "runtime": "toolkit-core",
  "ui_schema": "tetra.ui.toolkit.v1",
  "source": "tools/cmd/ui-toolkit-core-smoke",
  "artifacts": [
    {"name":"toolkit bundle","kind":"bundle","path":"__BUNDLE__","schema":"tetra.ui.toolkit.v1","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
    {"name":"runtime trace","kind":"trace","path":"__TRACE__","schema":"tetra.ui.toolkit.trace.v1","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
  ],
  "processes": [
    {"name":"toolkit core runtime","kind":"runtime","path":"tools/cmd/ui-toolkit-core-smoke --internal-check runtime","ran":true,"pass":true,"exit_code":0},
    {"name":"toolkit layout stress","kind":"stress","path":"tools/cmd/ui-toolkit-core-smoke --internal-check stress","ran":true,"pass":true,"exit_code":0},
    {"name":"toolkit validator","kind":"validator","path":"go run ./tools/cmd/validate-ui-toolkit-core","ran":true,"pass":true,"exit_code":0}
  ],
  "contracts": [
    {"name":"toolkit schema","status":"pass","evidence":"tetra.ui.toolkit.v1 runtime contract emitted and validated"},
    {"name":"widget model","status":"pass","evidence":"window/root/panel/text/label/button/input/checkbox/select/list/table/dialog/menu/menu-item/spacer/divider widgets executed"},
    {"name":"layout model","status":"pass","evidence":"stack row column grid flex constraints bounds overflow scroll layout evidence executed"},
    {"name":"style model","status":"pass","evidence":"deterministic style resolution for enabled disabled visible focused selected error states executed"},
    {"name":"accessibility model","status":"pass","evidence":"roles labels descriptions focus order keyboard activation state metadata projected"},
    {"name":"event model","status":"pass","evidence":"click activate focus blur input change select submit key timer async redraw error recovery dispatched"},
    {"name":"state binding model","status":"pass","evidence":"scalar list table and two-way input binding updates ran in deterministic order"}
  ],
  "widgets": [
    {"id":"AppWindow","kind":"window","parent":"","binding":"app.open","enabled":true,"visible":true,"focusable":false,"bounds":{"x":0,"y":0,"width":960,"height":640},"layout":{"kind":"window","order":0},"style":{"class":"window","state":"visible"},"accessibility":{"role":"application","label":"Toolkit Core","description":"Toolkit root window","focus_order":0,"keyboard_activation":[]}},
    {"id":"AppRoot","kind":"root","parent":"AppWindow","binding":"layout.root","enabled":true,"visible":true,"focusable":false,"bounds":{"x":0,"y":0,"width":960,"height":640},"layout":{"kind":"column","order":0,"gap":8},"style":{"class":"root"},"accessibility":{"role":"group","label":"Root","description":"Root mount","focus_order":0,"keyboard_activation":[]}},
    {"id":"Toolbar","kind":"panel","parent":"AppRoot","binding":"layout.toolbar","enabled":true,"visible":true,"focusable":false,"bounds":{"x":8,"y":8,"width":944,"height":48},"layout":{"kind":"row","order":1,"gap":8},"style":{"class":"toolbar"},"accessibility":{"role":"toolbar","label":"Toolbar","description":"Actions","focus_order":0,"keyboard_activation":[]}},
    {"id":"TitleText","kind":"text","parent":"Toolbar","binding":"state.title","value":"Saved","enabled":true,"visible":true,"focusable":false,"bounds":{"x":16,"y":16,"width":160,"height":24},"layout":{"kind":"row","order":1},"style":{"class":"title","state":"visible"},"accessibility":{"role":"text","label":"Title","description":"Title text","focus_order":0,"keyboard_activation":[]}},
    {"id":"NameLabel","kind":"label","parent":"Toolbar","binding":"state.name_label","value":"Name","enabled":true,"visible":true,"focusable":false,"bounds":{"x":184,"y":16,"width":80,"height":24},"layout":{"kind":"row","order":2},"style":{"class":"label"},"accessibility":{"role":"label","label":"Name","description":"Input label","focus_order":0,"keyboard_activation":[]}},
    {"id":"NameInput","kind":"input","parent":"Toolbar","binding":"state.name","event":"input","command":"setName","value":"tetra-toolkit","enabled":true,"visible":true,"focusable":true,"bounds":{"x":272,"y":12,"width":220,"height":32},"layout":{"kind":"row","order":3},"style":{"class":"input","state":"focused"},"accessibility":{"role":"textbox","label":"Name input","description":"Two-way name input","focus_order":1,"keyboard_activation":["tab","enter"]}},
    {"id":"EnabledToggle","kind":"checkbox","parent":"Toolbar","binding":"state.enabled","event":"change","command":"toggleEnabled","value":"true","enabled":true,"visible":true,"focusable":true,"bounds":{"x":500,"y":12,"width":32,"height":32},"layout":{"kind":"row","order":4},"style":{"class":"checkbox","state":"selected"},"accessibility":{"role":"checkbox","label":"Enabled","description":"Toggle enabled state","focus_order":2,"keyboard_activation":["space"]}},
    {"id":"ModeSelect","kind":"select","parent":"Toolbar","binding":"state.mode","event":"select","command":"selectMode","value":"advanced","enabled":true,"visible":true,"focusable":true,"bounds":{"x":540,"y":12,"width":128,"height":32},"layout":{"kind":"row","order":5},"style":{"class":"select","state":"selected"},"accessibility":{"role":"combobox","label":"Mode","description":"Mode selector","focus_order":3,"keyboard_activation":["arrowdown","enter"]}},
    {"id":"SaveButton","kind":"button","parent":"Toolbar","binding":"state.saved","event":"click","command":"saveAsync","value":"Save","enabled":true,"visible":true,"focusable":true,"bounds":{"x":676,"y":12,"width":88,"height":32},"layout":{"kind":"row","order":6},"style":{"class":"button"},"accessibility":{"role":"button","label":"Save","description":"Save changes","focus_order":4,"keyboard_activation":["enter","space"]}},
    {"id":"ContentPanel","kind":"panel","parent":"AppRoot","binding":"layout.content","enabled":true,"visible":true,"focusable":false,"bounds":{"x":8,"y":64,"width":944,"height":512},"layout":{"kind":"grid","order":2,"gap":12},"style":{"class":"content"},"accessibility":{"role":"group","label":"Content","description":"Content panel","focus_order":0,"keyboard_activation":[]}},
    {"id":"ItemList","kind":"list","parent":"ContentPanel","binding":"state.items","event":"select","command":"selectItem","value":"item-2","enabled":true,"visible":true,"focusable":true,"bounds":{"x":16,"y":72,"width":240,"height":240},"layout":{"kind":"grid","order":1},"style":{"class":"list","state":"selected"},"accessibility":{"role":"listbox","label":"Items","description":"Selectable items","focus_order":5,"keyboard_activation":["arrowdown","enter"]}},
    {"id":"DataTable","kind":"table","parent":"ContentPanel","binding":"state.rows","event":"select","command":"selectRow","value":"row-2","enabled":true,"visible":true,"focusable":true,"bounds":{"x":268,"y":72,"width":420,"height":240},"layout":{"kind":"grid","order":2},"style":{"class":"table"},"accessibility":{"role":"grid","label":"Rows","description":"Data table","focus_order":6,"keyboard_activation":["arrowdown","enter"]}},
    {"id":"OpenDialog","kind":"dialog","parent":"AppWindow","binding":"state.dialog","event":"submit","command":"closeDialog","value":"open","enabled":true,"visible":true,"focusable":true,"bounds":{"x":300,"y":180,"width":360,"height":220},"layout":{"kind":"modal","order":3},"style":{"class":"dialog","state":"visible"},"accessibility":{"role":"dialog","label":"Confirm","description":"Confirmation dialog","focus_order":7,"keyboard_activation":["escape","enter"]}},
    {"id":"FileMenu","kind":"menu","parent":"AppWindow","binding":"menu.file","enabled":true,"visible":true,"focusable":true,"bounds":{"x":0,"y":0,"width":160,"height":24},"layout":{"kind":"menu","order":0},"style":{"class":"menu"},"accessibility":{"role":"menu","label":"File","description":"File menu","focus_order":8,"keyboard_activation":["alt+f"]}},
    {"id":"MenuItemOpen","kind":"menu-item","parent":"FileMenu","binding":"command.open","event":"activate","command":"openDialog","enabled":true,"visible":true,"focusable":true,"bounds":{"x":0,"y":0,"width":120,"height":24},"layout":{"kind":"menu","order":0},"style":{"class":"menu-item"},"accessibility":{"role":"menuitem","label":"Open","description":"Open dialog","focus_order":9,"keyboard_activation":["enter"]}},
    {"id":"ContentSpacer","kind":"spacer","parent":"ContentPanel","binding":"layout.spacer","enabled":true,"visible":true,"focusable":false,"bounds":{"x":700,"y":72,"width":16,"height":240},"layout":{"kind":"grid","order":3},"style":{"class":"spacer"},"accessibility":{"role":"presentation","label":"Spacer","description":"Layout spacer","focus_order":0,"keyboard_activation":[]}},
    {"id":"ContentDivider","kind":"divider","parent":"ContentPanel","binding":"layout.divider","enabled":true,"visible":true,"focusable":false,"bounds":{"x":724,"y":72,"width":1,"height":240},"layout":{"kind":"grid","order":4},"style":{"class":"divider"},"accessibility":{"role":"separator","label":"Divider","description":"Content divider","focus_order":0,"keyboard_activation":[]}}
  ],
  "layouts": [
    {"kind":"stack","widgets":["AppWindow","AppRoot"],"pass":true,"evidence":"root stack is stable"},
    {"kind":"row","widgets":["Toolbar","TitleText","NameInput","SaveButton"],"pass":true,"evidence":"toolbar row placed deterministically"},
    {"kind":"column","widgets":["AppRoot","Toolbar","ContentPanel"],"pass":true,"evidence":"root column measured deterministically"},
    {"kind":"grid","widgets":["Toolbar","ContentPanel"],"pass":true,"evidence":"grid columns placed deterministically"},
    {"kind":"flex","widgets":["NameInput","SaveButton"],"pass":true,"evidence":"flex preferred/min/max widths respected"},
    {"kind":"overflow-scroll","widgets":["ItemList","DataTable"],"pass":true,"evidence":"overflow and scroll metadata retained"}
  ],
  "events": [
    {"order":1,"widget_id":"SaveButton","event":"click","command":"saveAsync","pass":true,"before_state":{"AppState.saved":"false"},"after_state":{"AppState.saved":"true"},"operations":[{"kind":"async_command","target":"command.saveAsync","value":"completed","state_field":"saved","state_value":"true"},{"kind":"redraw","target":"AppWindow","value":"scheduled","state_field":"dirty","state_value":"true"}],"widget_updates":[{"id":"TitleText","before":"Ready","after":"Saved"}]},
    {"order":2,"widget_id":"MenuItemOpen","event":"activate","command":"openDialog","pass":true,"before_state":{"AppState.dialog":"closed"},"after_state":{"AppState.dialog":"open"},"operations":[{"kind":"state_set","target":"state.dialog","value":"open","state_field":"dialog","state_value":"open"}],"widget_updates":[{"id":"OpenDialog","before":"closed","after":"open"}]},
    {"order":3,"widget_id":"NameInput","event":"focus","command":"focusName","pass":true,"before_state":{"AppState.focused":"none"},"after_state":{"AppState.focused":"NameInput"},"operations":[{"kind":"focus","target":"widget.NameInput","value":"focused","state_field":"focused","state_value":"NameInput"}],"widget_updates":[{"id":"NameInput","before":"blurred","after":"focused"}]},
    {"order":4,"widget_id":"NameInput","event":"blur","command":"blurName","pass":true,"before_state":{"AppState.focused":"NameInput"},"after_state":{"AppState.focused":"none"},"operations":[{"kind":"blur","target":"widget.NameInput","value":"blurred","state_field":"focused","state_value":"none"}],"widget_updates":[{"id":"NameInput","before":"focused","after":"blurred"}]},
    {"order":5,"widget_id":"EnabledToggle","event":"change","command":"toggleEnabled","pass":true,"before_state":{"AppState.enabled":"false"},"after_state":{"AppState.enabled":"true"},"operations":[{"kind":"state_set","target":"state.enabled","value":"true","state_field":"enabled","state_value":"true"}],"widget_updates":[{"id":"EnabledToggle","before":"false","after":"true"}]},
    {"order":6,"widget_id":"ModeSelect","event":"select","command":"selectMode","pass":true,"before_state":{"AppState.mode":"basic"},"after_state":{"AppState.mode":"advanced"},"operations":[{"kind":"state_set","target":"state.mode","value":"advanced","state_field":"mode","state_value":"advanced"}],"widget_updates":[{"id":"ModeSelect","before":"basic","after":"advanced"}]},
    {"order":7,"widget_id":"OpenDialog","event":"submit","command":"closeDialog","pass":true,"before_state":{"AppState.dialog":"open"},"after_state":{"AppState.dialog":"closed"},"operations":[{"kind":"state_set","target":"state.dialog","value":"closed","state_field":"dialog","state_value":"closed"}],"widget_updates":[{"id":"OpenDialog","before":"open","after":"closed"}]},
    {"order":8,"widget_id":"NameInput","event":"input","command":"setName","pass":true,"before_state":{"AppState.name":"tetra"},"after_state":{"AppState.name":"tetra-toolkit"},"operations":[{"kind":"two_way_bind","target":"state.name","value":"tetra-toolkit","state_field":"name","state_value":"tetra-toolkit"}],"widget_updates":[{"id":"NameInput","before":"tetra","after":"tetra-toolkit"}]},
    {"order":9,"widget_id":"DataTable","event":"key","command":"keySelect","pass":true,"before_state":{"AppState.row":"row-1"},"after_state":{"AppState.row":"row-2"},"operations":[{"kind":"key_activate","target":"widget.DataTable","value":"arrowdown","state_field":"row","state_value":"row-2"}],"widget_updates":[{"id":"DataTable","before":"row-1","after":"row-2"}]},
    {"order":10,"widget_id":"AppWindow","event":"timer","command":"timerTick","pass":true,"before_state":{"AppState.dirty":"true"},"after_state":{"AppState.dirty":"false"},"operations":[{"kind":"timer_tick","target":"timer.redraw","value":"fired","state_field":"dirty","state_value":"false"},{"kind":"redraw","target":"AppWindow","value":"completed","state_field":"dirty","state_value":"false"}],"widget_updates":[{"id":"TitleText","before":"Saved","after":"Saved after timer"}]},
    {"order":11,"widget_id":"AppWindow","event":"error_recovery","command":"recoverCommand","pass":true,"before_state":{"AppState.error":"panic"},"after_state":{"AppState.error":"recovered"},"operations":[{"kind":"error_recovery","target":"runtime.command","value":"recovered","state_field":"error","state_value":"recovered"}],"widget_updates":[{"id":"TitleText","before":"Error","after":"Recovered"}]}
  ],
  "state_transitions": [
    {"name":"scalar binding update","before":{"AppState.saved":"false"},"after":{"AppState.saved":"true"},"operations":["state_set"],"widgets":["SaveButton","TitleText"]},
    {"name":"list selection binding","before":{"AppState.selected":"item-1"},"after":{"AppState.selected":"item-2"},"operations":["state_set"],"widgets":["ItemList"]},
    {"name":"table selection binding","before":{"AppState.row":"row-1"},"after":{"AppState.row":"row-2"},"operations":["key_activate"],"widgets":["DataTable"]},
    {"name":"two-way input binding","before":{"AppState.name":"tetra"},"after":{"AppState.name":"tetra-toolkit"},"operations":["two_way_bind"],"widgets":["NameInput"]},
    {"name":"deterministic update order","before":{"order":"0"},"after":{"order":"11"},"operations":["click","activate","focus","blur","change","select","submit","input","key","timer","error_recovery"],"widgets":["AppWindow"]}
  ],
  "cases": [
    {"name":"positive widget tree","kind":"positive","ran":true,"pass":true},
    {"name":"layout stress","kind":"stress","ran":true,"pass":true},
    {"name":"event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"state binding update","kind":"positive","ran":true,"pass":true},
    {"name":"input focus select key","kind":"positive","ran":true,"pass":true},
    {"name":"timer async redraw","kind":"positive","ran":true,"pass":true},
    {"name":"dialog menu","kind":"positive","ran":true,"pass":true},
    {"name":"table list binding","kind":"positive","ran":true,"pass":true},
    {"name":"accessibility metadata","kind":"positive","ran":true,"pass":true},
    {"name":"unsupported widget diagnostic","kind":"negative","ran":true,"pass":true,"expected_error":"unsupported widget kind"},
    {"name":"unsupported operation diagnostic","kind":"negative","ran":true,"pass":true,"expected_error":"unsupported toolkit operation"},
    {"name":"malformed metadata","kind":"negative","ran":true,"pass":true,"expected_error":"malformed toolkit metadata"},
    {"name":"command failure recovery","kind":"negative","ran":true,"pass":true,"expected_error":"command failed"},
    {"name":"crash error recovery","kind":"negative","ran":true,"pass":true,"expected_error":"runtime panic recovered"}
  ],
  "audit": [
    {"requirement":"toolkit core contract","artifact":"tools/validators/uitoolkit; docs/spec/ui_toolkit_core.md","evidence":"tetra.ui.toolkit.v1 report validated","result":"pass"},
    {"requirement":"real runtime evidence","artifact":"tools/cmd/ui-toolkit-core-smoke","evidence":"runtime and stress internal checks executed","result":"pass"},
    {"requirement":"widget model","artifact":"ui-toolkit-core.bundle.json","evidence":"all selected widget kinds have runtime evidence","result":"pass"},
    {"requirement":"layout focus accessibility","artifact":"ui-toolkit-core.trace.json","evidence":"layout, focus order, keyboard activation, and accessibility metadata are present","result":"pass"},
    {"requirement":"event state update model","artifact":"ui-toolkit-core.trace.json","evidence":"events dispatch state transitions and widget updates","result":"pass"},
    {"requirement":"negative diagnostics","artifact":"tools/cmd/ui-toolkit-core-smoke","evidence":"unsupported widget/operation, malformed metadata, command failure, and crash recovery cases ran","result":"pass"}
  ]
}`
