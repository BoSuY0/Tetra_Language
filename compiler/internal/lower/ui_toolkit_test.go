package lower

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func TestLowerUIToolkitBundle(t *testing.T) {
	bundle := lowerUIToolkitForTest(t, `
state FormState:
    var name: String = "tetra"
    var saved: Bool = false

view FormView(state: FormState):
    bind nameInput: String = state.name
    bind savedText: Bool = state.saved
    event input -> setName
    event click -> save
    command setName:
        state.name = "toolkit"
    command save:
        state.saved = true
    style width: Int = 640
    accessibility label: String = "Form"

func main() -> Int:
    return 0
`)
	if bundle.Schema != UIToolkitSchema {
		t.Fatalf("schema = %q, want %q", bundle.Schema, UIToolkitSchema)
	}
	if len(bundle.Views) != 1 {
		t.Fatalf("views = %#v", bundle.Views)
	}
	view := bundle.Views[0]
	for _, want := range []string{"window", "root", "panel", "text", "button", "input", "list", "table", "dialog", "menu"} {
		if !contains(view.WidgetKinds, want) {
			t.Fatalf("widget kinds missing %q: %#v", want, view.WidgetKinds)
		}
	}
	for _, want := range []string{"stack", "row", "column", "grid", "flex"} {
		if !contains(view.LayoutKinds, want) {
			t.Fatalf("layout kinds missing %q: %#v", want, view.LayoutKinds)
		}
	}
	if len(view.Widgets) < 5 {
		t.Fatalf("widgets = %#v", view.Widgets)
	}
	if len(view.Events) != 2 || view.Events[0].Name != "click" || view.Events[1].Name != "input" {
		t.Fatalf("events should be deterministic and sorted: %#v", view.Events)
	}
	if len(view.Commands) != 2 || len(view.Commands[0].Operations) == 0 {
		t.Fatalf("commands = %#v", view.Commands)
	}
}

func TestLowerUIToolkitRejectsUnsupportedCommandOperation(t *testing.T) {
	_, err := LowerUIToolkit(&UILoweredBundle{
		Schema: UIBundleSchema,
		Views: []UILoweredView{{
			Name:      "BadView",
			Module:    "main",
			StateType: "BadState",
			Commands: []UILoweredCommand{{
				Name:           "unsupported",
				StatementCount: 1,
			}},
		}},
	})
	if err == nil {
		t.Fatalf("expected unsupported toolkit operation error")
	}
	if !strings.Contains(err.Error(), "unsupported UI toolkit command operation") {
		t.Fatalf("error = %v", err)
	}
}

func lowerUIToolkitForTest(t *testing.T, src string) *UIToolkitBundle {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	ui, err := LowerUI(checked)
	if err != nil {
		t.Fatalf("LowerUI: %v", err)
	}
	bundle, err := LowerUIToolkit(ui)
	if err != nil {
		t.Fatalf("LowerUIToolkit: %v", err)
	}
	if bundle == nil {
		t.Fatalf("bundle = nil")
	}
	return bundle
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
