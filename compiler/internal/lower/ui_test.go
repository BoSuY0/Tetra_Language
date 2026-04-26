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
	if bundle.Schema != "tetra.ui.v1" {
		t.Fatalf("schema = %q", bundle.Schema)
	}
	if len(bundle.States) != 1 || len(bundle.Views) != 1 {
		t.Fatalf("bundle = %#v", bundle)
	}
	if bundle.Views[0].StateType == "" || len(bundle.Views[0].Commands) != 1 {
		t.Fatalf("view payload = %#v", bundle.Views[0])
	}
}
