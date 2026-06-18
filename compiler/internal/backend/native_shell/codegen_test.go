package native_shell

import (
	"encoding/json"
	"strings"
	"testing"

	"tetra_language/compiler/internal/lower"
)

func TestRenderIncludesStateAndViewMetadata(t *testing.T) {
	bundle := &lower.UILoweredBundle{
		Schema: "tetra.ui.v0.4.0",
		States: []lower.UILoweredState{
			{
				Name:   "ShellState",
				Module: "main",
				Fields: []lower.UILoweredStateField{
					{Name: "toggles", Type: "i32", Mutable: true, Init: "0"},
					{Name: "label", Type: "str", Const: true, Init: `"Native"`},
				},
			},
		},
		Views: []lower.UILoweredView{
			{
				Name:      "ShellView",
				StateType: "ShellState",
				Bindings: []lower.UILoweredBinding{
					{Name: "toggles", Type: "i32", Source: "state.toggles"},
				},
				Events: []lower.UILoweredEvent{{Name: "submit", Command: "toggle"}},
				Commands: []lower.UILoweredCommand{{
					Name:           "toggle",
					StatementCount: 1,
					Operations: []lower.UILoweredCommandOperation{
						{Kind: "state_add", Target: "state.toggles", Value: "1"},
					},
				}},
			},
		},
	}

	out := string(Render(bundle))
	for _, want := range []string{
		"runtime: native shell command dispatch",
		"state ShellState",
		"  var toggles: i32 = 0",
		"  const label: str = \"Native\"",
		"view ShellView (state: ShellState)",
		"  bind toggles: i32 = 0",
		"  event submit -> toggle",
		"  command toggle (1 stmt)",
		"    op state_add state.toggles 1",
		"  dispatch submit -> toggle",
		"    state.toggles = 1",
		"  bind toggles: i32 = 1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("render output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderDispatchesStateSubtractOperations(t *testing.T) {
	bundle := &lower.UILoweredBundle{
		Schema: "tetra.ui.v0.4.0",
		States: []lower.UILoweredState{
			{
				Name:   "ShellState",
				Module: "main",
				Fields: []lower.UILoweredStateField{
					{Name: "count", Type: "i32", Mutable: true, Init: "5"},
				},
			},
		},
		Views: []lower.UILoweredView{
			{
				Name:      "ShellView",
				StateType: "ShellState",
				Bindings: []lower.UILoweredBinding{
					{Name: "count", Type: "i32", Source: "state.count"},
				},
				Events: []lower.UILoweredEvent{{Name: "submit", Command: "decrement"}},
				Commands: []lower.UILoweredCommand{{
					Name:           "decrement",
					StatementCount: 1,
					Operations: []lower.UILoweredCommandOperation{
						{Kind: "state_sub", Target: "state.count", Value: "2"},
					},
				}},
			},
		},
	}

	out := string(Render(bundle))
	for _, want := range []string{
		"    op state_sub state.count 2",
		"  dispatch submit -> decrement",
		"    state.count = 3",
		"  bind count: i32 = 3",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("render output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderDispatchesStringStateSetWithoutLiteralQuotes(t *testing.T) {
	bundle := &lower.UILoweredBundle{
		Schema: "tetra.ui.v0.4.0",
		States: []lower.UILoweredState{
			{
				Name:   "ShellState",
				Module: "main",
				Fields: []lower.UILoweredStateField{
					{Name: "label", Type: "str", Mutable: true, Init: `"Ready"`},
				},
			},
		},
		Views: []lower.UILoweredView{
			{
				Name:      "ShellView",
				StateType: "ShellState",
				Bindings: []lower.UILoweredBinding{
					{Name: "labelText", Type: "str", Source: "state.label"},
				},
				Events: []lower.UILoweredEvent{{Name: "submit", Command: "rename"}},
				Commands: []lower.UILoweredCommand{{
					Name:           "rename",
					StatementCount: 1,
					Operations: []lower.UILoweredCommandOperation{
						{Kind: "state_set", Target: "state.label", Value: `"Done"`},
					},
				}},
			},
		},
	}

	out := string(Render(bundle))
	for _, want := range []string{
		"  bind labelText: str = Ready",
		"  dispatch submit -> rename",
		"    state.label = Done",
		"  bind labelText: str = Done",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("render output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderDispatchesStateSetFromSameStateField(t *testing.T) {
	bundle := &lower.UILoweredBundle{
		Schema: "tetra.ui.v0.4.0",
		States: []lower.UILoweredState{
			{
				Name:   "ShellState",
				Module: "main",
				Fields: []lower.UILoweredStateField{
					{Name: "source", Type: "str", Mutable: true, Init: `"Copied"`},
					{Name: "label", Type: "str", Mutable: true, Init: `"Ready"`},
				},
			},
		},
		Views: []lower.UILoweredView{
			{
				Name:      "ShellView",
				StateType: "ShellState",
				Bindings: []lower.UILoweredBinding{
					{Name: "labelText", Type: "str", Source: "state.label"},
				},
				Events: []lower.UILoweredEvent{{Name: "submit", Command: "copy"}},
				Commands: []lower.UILoweredCommand{{
					Name:           "copy",
					StatementCount: 1,
					Operations: []lower.UILoweredCommandOperation{
						{Kind: "state_set", Target: "state.label", Value: "state.source"},
					},
				}},
			},
		},
	}

	out := string(Render(bundle))
	for _, want := range []string{
		"  bind labelText: str = Ready",
		"  dispatch submit -> copy",
		"    state.label = Copied",
		"  bind labelText: str = Copied",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("render output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderDispatchesCommandOperationsInOrder(t *testing.T) {
	bundle := &lower.UILoweredBundle{
		Schema: "tetra.ui.v0.4.0",
		States: []lower.UILoweredState{
			{
				Name:   "ShellState",
				Module: "main",
				Fields: []lower.UILoweredStateField{
					{Name: "source", Type: "i32", Mutable: true, Init: "1"},
					{Name: "target", Type: "i32", Mutable: true, Init: "0"},
				},
			},
		},
		Views: []lower.UILoweredView{
			{
				Name:      "ShellView",
				StateType: "ShellState",
				Bindings: []lower.UILoweredBinding{
					{Name: "targetValue", Type: "i32", Source: "state.target"},
				},
				Events: []lower.UILoweredEvent{{Name: "submit", Command: "copyAfterIncrement"}},
				Commands: []lower.UILoweredCommand{{
					Name:           "copyAfterIncrement",
					StatementCount: 2,
					Operations: []lower.UILoweredCommandOperation{
						{Kind: "state_add", Target: "state.source", Value: "2"},
						{Kind: "state_set", Target: "state.target", Value: "state.source"},
					},
				}},
			},
		},
	}

	out := string(Render(bundle))
	for _, want := range []string{
		"  command copyAfterIncrement (2 stmt)",
		"    op state_add state.source 2",
		"    op state_set state.target state.source",
		"  dispatch submit -> copyAfterIncrement",
		"    state.source = 3",
		"    state.target = 3",
		"  bind targetValue: i32 = 3",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("render output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderJSONIncludesDispatchTrace(t *testing.T) {
	bundle := &lower.UILoweredBundle{
		Schema: "tetra.ui.v0.4.0",
		States: []lower.UILoweredState{
			{
				Name:   "ShellState",
				Module: "main",
				Fields: []lower.UILoweredStateField{
					{Name: "count", Type: "i32", Mutable: true, Init: "1"},
				},
			},
		},
		Views: []lower.UILoweredView{
			{
				Name:      "ShellView",
				StateType: "ShellState",
				Bindings: []lower.UILoweredBinding{
					{Name: "countValue", Type: "i32", Source: "state.count"},
				},
				Events: []lower.UILoweredEvent{{Name: "submit", Command: "increment"}},
				Commands: []lower.UILoweredCommand{{
					Name:           "increment",
					StatementCount: 1,
					Operations: []lower.UILoweredCommandOperation{
						{Kind: "state_add", Target: "state.count", Value: "2"},
					},
				}},
			},
		},
	}

	var report struct {
		Schema   string `json:"schema"`
		UISchema string `json:"ui_schema"`
		Runtime  string `json:"runtime"`
		Views    []struct {
			Name   string `json:"name"`
			Events []struct {
				Name       string `json:"name"`
				Command    string `json:"command"`
				Operations []struct {
					Kind       string `json:"kind"`
					Target     string `json:"target"`
					Value      string `json:"value"`
					StateField string `json:"state_field"`
					StateValue string `json:"state_value"`
				} `json:"operations"`
				Bindings []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"bindings"`
			} `json:"events"`
		} `json:"views"`
	}
	if err := json.Unmarshal(RenderJSON(bundle), &report); err != nil {
		t.Fatalf("RenderJSON produced invalid JSON: %v\n%s", err, RenderJSON(bundle))
	}
	if report.Schema != "tetra.ui.native-shell.v1" || report.UISchema != "tetra.ui.v0.4.0" ||
		report.Runtime != "native shell command dispatch" {
		t.Fatalf("report header = %#v", report)
	}
	if len(report.Views) != 1 || len(report.Views[0].Events) != 1 {
		t.Fatalf("report views = %#v", report.Views)
	}
	event := report.Views[0].Events[0]
	if event.Name != "submit" || event.Command != "increment" || len(event.Operations) != 1 {
		t.Fatalf("event trace = %#v", event)
	}
	op := event.Operations[0]
	if op.Kind != "state_add" || op.Target != "state.count" || op.Value != "2" ||
		op.StateField != "count" ||
		op.StateValue != "3" {
		t.Fatalf("operation trace = %#v", op)
	}
	if len(event.Bindings) != 1 || event.Bindings[0].Name != "countValue" ||
		event.Bindings[0].Value != "3" {
		t.Fatalf("event bindings = %#v", event.Bindings)
	}
}

func TestRenderJSONIncludesNativeWidgetTree(t *testing.T) {
	bundle := &lower.UILoweredBundle{
		Schema: "tetra.ui.v0.4.0",
		States: []lower.UILoweredState{
			{
				Name:   "ShellState",
				Module: "main",
				Fields: []lower.UILoweredStateField{
					{Name: "label", Type: "str", Mutable: true, Init: `"Ready"`},
				},
			},
		},
		Views: []lower.UILoweredView{
			{
				Name:      "ShellView",
				StateType: "ShellState",
				Bindings: []lower.UILoweredBinding{
					{Name: "labelText", Type: "str", Source: "state.label"},
				},
				Events: []lower.UILoweredEvent{{Name: "submit", Command: "rename"}},
				Commands: []lower.UILoweredCommand{{
					Name:           "rename",
					StatementCount: 1,
					Operations: []lower.UILoweredCommandOperation{
						{Kind: "state_set", Target: "state.label", Value: `"Done"`},
					},
				}},
				Styles: []lower.UILoweredStyle{
					{Name: "accent", Type: "str", Value: `"amber"`},
				},
				Accessibility: []lower.UILoweredAccessibility{
					{Name: "role", Type: "str", Value: `"status"`},
				},
			},
		},
	}

	var report struct {
		Views []struct {
			Widgets []struct {
				ID            string                   `json:"id"`
				Kind          string                   `json:"kind"`
				Binding       string                   `json:"binding,omitempty"`
				Event         string                   `json:"event,omitempty"`
				Command       string                   `json:"command,omitempty"`
				Type          string                   `json:"type,omitempty"`
				Value         string                   `json:"value,omitempty"`
				Styles        []lower.UILoweredStyle   `json:"styles,omitempty"`
				Accessibility []shellAccessibilityItem `json:"accessibility,omitempty"`
			} `json:"widgets"`
		} `json:"views"`
	}
	if err := json.Unmarshal(RenderJSON(bundle), &report); err != nil {
		t.Fatalf("RenderJSON produced invalid JSON: %v\n%s", err, RenderJSON(bundle))
	}
	if len(report.Views) != 1 || len(report.Views[0].Widgets) != 2 {
		t.Fatalf("widgets = %#v", report.Views)
	}
	label := report.Views[0].Widgets[0]
	if label.ID != "ShellView.labelText" || label.Kind != "text" || label.Binding != "labelText" ||
		label.Type != "str" ||
		label.Value != "Ready" {
		t.Fatalf("binding widget = %#v", label)
	}
	if len(label.Styles) != 1 || label.Styles[0].Name != "accent" ||
		len(label.Accessibility) != 1 ||
		label.Accessibility[0].Name != "role" {
		t.Fatalf("binding widget metadata = %#v", label)
	}
	action := report.Views[0].Widgets[1]
	if action.ID != "ShellView.submit" || action.Kind != "action" || action.Event != "submit" ||
		action.Command != "rename" {
		t.Fatalf("action widget = %#v", action)
	}
}

func TestRenderRejectsUnsupportedSchema(t *testing.T) {
	out := string(Render(&lower.UILoweredBundle{Schema: "tetra.ui.v0"}))
	for _, want := range []string{
		"unsupported UI schema: tetra.ui.v0",
		"runtime: unavailable",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("render output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderNilBundleIncludesNoMetadataMarker(t *testing.T) {
	out := string(Render(nil))
	if !strings.Contains(out, "(no UI metadata)") {
		t.Fatalf("nil bundle output = %q", out)
	}
}
