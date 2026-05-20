package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestUICheckStateViewBindingsEventsCommandsOK(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
state CounterState:
    var count: Int = 0
    val title: String = "Counter"

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    bind titleText: String = state.title
    event click -> increment
    command increment:
        state.count = state.count + 1
    style width: Int = 320
    accessibility label: String = "Increment counter"

func main() -> Int:
    return 0
`)
}

func TestUICheckEventRequiresExistingCommand(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
state CounterState:
    var count: Int = 0

view CounterView(state: CounterState):
    event click -> missing
    command increment:
        state.count = state.count + 1
`, "references unknown command")
}

func TestUICheckStyleTypeMismatch(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
state CounterState:
    var count: Int = 0

view CounterView(state: CounterState):
    command increment:
        state.count = state.count + 1
    style width: Int = "wide"
`, "style 'width' type mismatch")
}

func TestUICheckRejectsImmutableStateWrites(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
state CounterState:
    const seed: Int = 1

view CounterView(state: CounterState):
    command reset:
        state.seed = 0
	`, "cannot assign to immutable state field")
}

func TestUICheckRequiresAtLeastOneCommandPerView(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
state CounterState:
    var count: Int = 0

view CounterView(state: CounterState):
    bind countValue: Int = state.count
`, "view requires at least one command")
}

func TestUICheckRejectsReturnInsideViewCommand(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
state CounterState:
    var count: Int = 0

view CounterView(state: CounterState):
    command bad:
        return 0
`, "return is not allowed inside view commands")
}
