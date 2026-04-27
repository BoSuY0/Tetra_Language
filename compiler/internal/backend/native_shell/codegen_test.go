package native_shell

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/lower"
)

func TestRenderIncludesStateAndViewMetadata(t *testing.T) {
	bundle := &lower.UILoweredBundle{
		Schema: "tetra.ui.v1",
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
				Events:    []lower.UILoweredEvent{{Name: "submit", Command: "toggle"}},
				Commands:  []lower.UILoweredCommand{{Name: "toggle", StatementCount: 1}},
			},
		},
	}

	out := string(Render(bundle))
	for _, want := range []string{
		"state ShellState",
		"  var toggles: i32 = 0",
		"  const label: str = \"Native\"",
		"view ShellView (state: ShellState)",
		"  event submit -> toggle",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("render output missing %q:\n%s", want, out)
		}
	}
}
