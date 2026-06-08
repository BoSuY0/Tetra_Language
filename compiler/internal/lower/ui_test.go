package lower

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func TestLowerUIBundle(t *testing.T) {
	src := []byte(`
state CounterState:
    var count: Int = 0

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> increment
    command increment:
        state.count = state.count + 1
    style width: Int = 320
    accessibility label: String = "Increment"

func main() -> Int:
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	bundle, err := LowerUI(checked)
	if err != nil {
		t.Fatalf("LowerUI: %v", err)
	}
	if bundle == nil {
		t.Fatalf("bundle = nil")
	}
	if bundle.Schema != "tetra.ui.v0.4.0" {
		t.Fatalf("schema = %q", bundle.Schema)
	}
	if len(bundle.States) != 1 || len(bundle.Views) != 1 {
		t.Fatalf("bundle = %#v", bundle)
	}
	if bundle.Views[0].StateType == "" || len(bundle.Views[0].Commands) != 1 {
		t.Fatalf("view payload = %#v", bundle.Views[0])
	}
	view := bundle.Views[0]
	if len(view.Events) != 1 || view.Events[0].Command != "increment" {
		t.Fatalf("events payload = %#v", view.Events)
	}
	if len(view.Commands) != 1 || len(view.Commands[0].Operations) != 1 {
		t.Fatalf("command operations = %#v", view.Commands)
	}
	op := view.Commands[0].Operations[0]
	if op.Kind != "state_add" || op.Target != "state.count" || op.Value != "1" {
		t.Fatalf("command operation = %#v, want state_add state.count by 1", op)
	}
	if len(view.Styles) != 1 || view.Styles[0].Value != "320" {
		t.Fatalf("styles payload = %#v", view.Styles)
	}
	if len(view.Accessibility) != 1 || view.Accessibility[0].Value != `"Increment"` {
		t.Fatalf("accessibility payload = %#v", view.Accessibility)
	}
}

func TestLowerUIBundleRecognizesStateSubtractCommands(t *testing.T) {
	src := []byte(`
state CounterState:
    var count: Int = 5

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> decrement
    command decrement:
        state.count = state.count - 2

func main() -> Int:
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	bundle, err := LowerUI(checked)
	if err != nil {
		t.Fatalf("LowerUI: %v", err)
	}
	view := bundle.Views[0]
	if len(view.Commands) != 1 || len(view.Commands[0].Operations) != 1 {
		t.Fatalf("command operations = %#v", view.Commands)
	}
	op := view.Commands[0].Operations[0]
	if op.Kind != "state_sub" || op.Target != "state.count" || op.Value != "2" {
		t.Fatalf("command operation = %#v, want state_sub state.count by 2", op)
	}
}

func TestLowerUIBundleRecognizesCompoundStateDeltaCommands(t *testing.T) {
	src := []byte(`
state CounterState:
    var count: Int = 5

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> adjust
    command adjust:
        state.count += 2
        state.count -= 1

func main() -> Int:
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	bundle, err := LowerUI(checked)
	if err != nil {
		t.Fatalf("LowerUI: %v", err)
	}
	view := bundle.Views[0]
	if len(view.Commands) != 1 || len(view.Commands[0].Operations) != 2 {
		t.Fatalf("command operations = %#v", view.Commands)
	}
	wants := []UILoweredCommandOperation{
		{Kind: "state_add", Target: "state.count", Value: "2"},
		{Kind: "state_sub", Target: "state.count", Value: "1"},
	}
	for i, want := range wants {
		if got := view.Commands[0].Operations[i]; got != want {
			t.Fatalf("operation %d = %#v, want %#v", i, got, want)
		}
	}
}

func TestLowerUIBundleRejectsNilCheckedProgram(t *testing.T) {
	if _, err := LowerUI(nil); err == nil {
		t.Fatalf("expected nil checked program error")
	}
}

func TestLowerUIBundleReturnsNilWhenUIDeclsAreMissing(t *testing.T) {
	src := []byte(`
func main() -> Int:
    return 0
`)
	prog, err := frontend.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	bundle, err := LowerUI(checked)
	if err != nil {
		t.Fatalf("LowerUI: %v", err)
	}
	if bundle != nil {
		t.Fatalf("bundle = %#v, want nil", bundle)
	}
}
