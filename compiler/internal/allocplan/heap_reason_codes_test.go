package allocplan

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/plir"
)

func TestPlannerReportsHeapReasonCodesForRemainingHeapAllocations(t *testing.T) {
	plan := allocationPlanWithOptions(t, `
func ret() -> []u8
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    return xs

func large_local() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(5000)
    xs[0] = 1
    return xs[0]

func main() -> Int
uses alloc, mem:
    return large_local()
`, Options{EnableSmallHeapRuntime: true})

	ret := findAllocation(t, plan, "ret", "xs")
	assertHeapReasonCode(t, ret, "heap.required_escape_return")
	large := findAllocation(t, plan, "large_local", "xs")
	assertHeapReasonCode(t, large, "heap.required_large_object")

	unknownPlan, err := FromPLIRWithOptions(&plir.Program{Funcs: []plir.Function{
		syntheticEscapeFunction("unknown_call", plir.Operation{Kind: plir.OpCall, Inputs: []string{"xs"}, Note: "external call without escape facts"}),
	}}, Options{EnableSmallHeapRuntime: true})
	if err != nil {
		t.Fatalf("FromPLIRWithOptions unknown: %v", err)
	}
	assertHeapReasonCode(t, findAllocation(t, unknownPlan, "unknown_call", "xs"), "heap.required_unknown_call")

	summary := Summarize(plan)
	for _, code := range []string{"heap.required_escape_return", "heap.required_large_object"} {
		if summary.HeapReasonCodes[code] != 1 {
			t.Fatalf("summary heap reason code %s = %d, want 1: %+v", code, summary.HeapReasonCodes[code], summary.HeapReasonCodes)
		}
	}
	text := FormatText(plan)
	for _, want := range []string{
		"heap_reason_codes: heap.required_escape_return",
		"heap_reason_codes: heap.required_large_object",
		"heap_reason_codes:heap.required_escape_return=1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("FormatText missing %q:\n%s", want, text)
		}
	}
}

func TestPlannerReportsHeapReasonCodesForActorTaskAndRegionFallbacks(t *testing.T) {
	plan, err := FromPLIRWithOptions(&plir.Program{Funcs: []plir.Function{
		syntheticEscapeFunction("actor", plir.Operation{Kind: plir.OpCall, Inputs: []string{"xs"}, Note: "core.send_typed sends actor payload"}),
		syntheticEscapeFunction("task", plir.Operation{Kind: plir.OpCall, Inputs: []string{"xs"}, Note: "core.task_spawn_i32_typed captures payload"}),
	}}, Options{EnableSmallHeapRuntime: true})
	if err != nil {
		t.Fatalf("FromPLIRWithOptions: %v", err)
	}
	assertHeapReasonCode(t, findAllocation(t, plan, "actor", "xs"), "heap.required_actor_boundary")
	assertHeapReasonCode(t, findAllocation(t, plan, "task", "xs"), "heap.required_task_boundary")

	regionPlan := allocationPlanWithOptions(t, `
func local_copy(n: Int) -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(8)
    let copied: []u8 = xs.window(0, n).copy()
    return copied.len

func main() -> Int
uses alloc, mem:
    return local_copy(2)
`, Options{EnableRegionPlanning: true})
	copied := findAllocation(t, regionPlan, "local_copy", "copied")
	assertHeapReasonCode(t, copied, "heap.required_region_lowering_unavailable")
}

func assertHeapReasonCode(t *testing.T, alloc Allocation, want string) {
	t.Helper()
	if !containsString(alloc.HeapReasonCodes, want) {
		t.Fatalf("allocation %s heap reason codes = %v, want %s: %+v", alloc.ID, alloc.HeapReasonCodes, want, alloc)
	}
	if !containsString(alloc.ReasonCodes, want) {
		t.Fatalf("allocation %s reason codes = %v, want %s: %+v", alloc.ID, alloc.ReasonCodes, want, alloc)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
