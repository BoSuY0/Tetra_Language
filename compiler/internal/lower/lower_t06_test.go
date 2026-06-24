package lower

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/memorypipeline"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/semantics"
)

func TestLowerPlannedProgramEmitsOneEvidenceRowPerAllocation(t *testing.T) {
	checked, plan := t06CheckedAndPlan(t, `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = core.make_u8(4)
    xs[0] = 7
    return xs[0]
`, allocplan.Options{EnableStackLowering: true})

	result, err := LowerPlannedProgram(
		checked,
		plan,
		Options{StackAllocationLowering: true},
	)
	if err != nil {
		t.Fatalf("LowerPlannedProgram: %v", err)
	}
	if result == nil || result.Program == nil {
		t.Fatalf("LowerPlannedProgram returned nil result/program")
	}
	if got, want := len(result.Evidence.Allocations), countPlanAllocations(plan); got != want {
		t.Fatalf("evidence rows = %d, want one per allocation %d: %+v", got, want, result.Evidence.Allocations)
	}
	ev := result.Evidence.Allocations[0]
	if ev.Function != "main" || ev.AllocationID != "xs" {
		t.Fatalf("evidence identity = %+v, want main/xs", ev)
	}
	if ev.PlannedStorage != allocplan.StorageStack || ev.ActualStorage != allocplan.StorageStack {
		t.Fatalf("evidence storage = %+v, want planned/actual Stack", ev)
	}
	if !strings.HasPrefix(ev.ArtifactID, "ir:main:") || !strings.HasSuffix(ev.ArtifactID, ":xs") {
		t.Fatalf("artifact id = %q, want ir:main:<first>:<last>:xs", ev.ArtifactID)
	}
	if ev.FirstInstruction < 0 || ev.LastInstruction < ev.FirstInstruction {
		t.Fatalf("instruction range = %d..%d, want valid range", ev.FirstInstruction, ev.LastInstruction)
	}
}

func TestLowerPlannedProgramNamesHeapCopyAllocationEvidence(t *testing.T) {
	checked, state := t06CheckedAndState(t, `
var saved: []u8? = none

func stash_copied_optional(xs: borrow []u8) -> Int
uses alloc, mem:
    saved = xs.window(0, 1).copy()
    return 0

func main() -> Int:
    return 0
`, allocplan.Options{})

	result, err := LowerPlannedProgram(checked, state.Plan, Options{})
	if err != nil {
		t.Fatalf("LowerPlannedProgram: %v", err)
	}
	copyAlloc := findT06AllocationByBuiltin(t, state.Plan, "stash_copied_optional", "core.slice_copy_u8")
	var got *AllocationLoweringEvidence
	for i := range result.Evidence.Allocations {
		ev := &result.Evidence.Allocations[i]
		if ev.Function == "stash_copied_optional" && ev.AllocationID == copyAlloc.ID {
			got = ev
			break
		}
	}
	if got == nil {
		t.Fatalf("copy allocation evidence missing for %+v: %+v", copyAlloc, result.Evidence.Allocations)
	}
	if got.ActualStorage != allocplan.StorageHeap ||
		got.FirstInstruction < 0 ||
		!strings.Contains(got.ArtifactID, copyAlloc.ID) {
		t.Fatalf("copy allocation evidence = %+v, want named heap artifact", *got)
	}
	if err := state.ApplyLowering(result.Program, result.Evidence); err != nil {
		t.Fatalf("ApplyLowering: %v", err)
	}
}

func TestLowerPlannedProgramRejectsMissingPlan(t *testing.T) {
	checked, _ := t06CheckedAndPlan(t, `
func main() -> Int:
    return 0
`, allocplan.Options{})

	_, err := LowerPlannedProgram(checked, nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "plan") {
		t.Fatalf("LowerPlannedProgram error = %v, want missing plan", err)
	}
}

func TestLowerWithOptionsRejectsImplicitPlanConstruction(t *testing.T) {
	checked, _ := t06CheckedAndPlan(t, `
func main() -> Int:
    return 0
`, allocplan.Options{})

	_, err := LowerWithOptions(checked, Options{})
	if err == nil || !strings.Contains(err.Error(), "explicit allocation plan") {
		t.Fatalf("LowerWithOptions error = %v, want explicit allocation plan requirement", err)
	}
}

func TestProgramResultModuleFuncsAreDefensiveAndDigestIsStable(t *testing.T) {
	checked, plan := t06CheckedAndPlan(t, `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = core.make_u8(4)
    return xs[0]
`, allocplan.Options{EnableStackLowering: true})

	result, err := LowerPlannedProgram(
		checked,
		plan,
		Options{StackAllocationLowering: true},
	)
	if err != nil {
		t.Fatalf("LowerPlannedProgram: %v", err)
	}
	funcs1, err := result.ModuleFuncs("")
	if err != nil {
		t.Fatalf("ModuleFuncs first: %v", err)
	}
	funcs2, err := result.ModuleFuncs("")
	if err != nil {
		t.Fatalf("ModuleFuncs second: %v", err)
	}
	if len(funcs1) == 0 || len(funcs1[0].Instrs) == 0 || len(funcs2) == 0 {
		t.Fatalf("ModuleFuncs returned empty funcs: %#v %#v", funcs1, funcs2)
	}
	originalKind := funcs2[0].Instrs[0].Kind
	funcs1[0].Instrs[0].Kind = ir.IRReturn
	funcs3, err := result.ModuleFuncs("")
	if err != nil {
		t.Fatalf("ModuleFuncs third: %v", err)
	}
	if funcs3[0].Instrs[0].Kind != originalKind {
		t.Fatalf("ModuleFuncs returned shared IR slice; got %v want %v", funcs3[0].Instrs[0].Kind, originalKind)
	}
	digest1, err := result.ModuleLoweringDigest("")
	if err != nil {
		t.Fatalf("ModuleLoweringDigest first: %v", err)
	}
	digest2, err := result.ModuleLoweringDigest("")
	if err != nil {
		t.Fatalf("ModuleLoweringDigest second: %v", err)
	}
	if !strings.HasPrefix(digest1, "lowering:sha256:") || digest1 != digest2 {
		t.Fatalf("ModuleLoweringDigest = %q/%q, want stable lowering sha256 digest", digest1, digest2)
	}
}

func TestProgramResultModuleFuncsIncludesGeneratedWrappers(t *testing.T) {
	checked, plan := t06CheckedAndPlan(t, `
enum TaskErr:
    case boom(Int, Int)
    case stopped

func worker() -> Int throws TaskErr:
    return 11

func main() -> Int
uses runtime:
    return catch core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<TaskErr>("worker")):
    case TaskErr.boom(left, right):
        left + right
    case TaskErr.stopped:
        0
`, allocplan.Options{})

	result, err := LowerPlannedProgram(checked, plan, Options{})
	if err != nil {
		t.Fatalf("LowerPlannedProgram: %v", err)
	}
	funcs, err := result.ModuleFuncs("")
	if err != nil {
		t.Fatalf("ModuleFuncs: %v", err)
	}
	wrapperName := typedTaskWrapperName("worker", "TaskErr")
	if !hasT06Func(funcs, wrapperName) {
		t.Fatalf("ModuleFuncs missing typed task wrapper %q; funcs=%v", wrapperName, t06FuncNames(funcs))
	}
}

func t06CheckedAndPlan(
	t *testing.T,
	src string,
	planOpt allocplan.Options,
) (*semantics.CheckedProgram, *allocplan.Plan) {
	t.Helper()
	checked, state := t06CheckedAndState(t, src, planOpt)
	return checked, state.Plan
}

func t06CheckedAndState(
	t *testing.T,
	src string,
	planOpt allocplan.Options,
) (*semantics.CheckedProgram, *memorypipeline.State) {
	t.Helper()
	file, err := frontend.ParseFile([]byte(src), "t06.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &module.World{
		EntryModule: file.Module,
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{file.Module: file},
	}
	checked, err := semantics.CheckWorld(world)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	state, err := memorypipeline.Build(checked, memorypipeline.Options{AllocPlan: planOpt})
	if err != nil {
		t.Fatalf("memorypipeline.Build: %v", err)
	}
	return checked, state
}

func countPlanAllocations(plan *allocplan.Plan) int {
	if plan == nil {
		return 0
	}
	total := 0
	for _, fn := range plan.Functions {
		total += len(fn.Allocations)
	}
	return total
}

func findT06AllocationByBuiltin(
	t *testing.T,
	plan *allocplan.Plan,
	function string,
	builtin string,
) allocplan.Allocation {
	t.Helper()
	if plan == nil {
		t.Fatalf("missing plan")
	}
	for _, fn := range plan.Functions {
		if fn.Name != function {
			continue
		}
		for _, alloc := range fn.Allocations {
			if alloc.Builtin == builtin {
				return alloc
			}
		}
	}
	t.Fatalf("allocation for %s/%s not found in %+v", function, builtin, plan.Functions)
	return allocplan.Allocation{}
}

func hasT06Func(funcs []ir.IRFunc, name string) bool {
	for _, fn := range funcs {
		if fn.Name == name {
			return true
		}
	}
	return false
}

func t06FuncNames(funcs []ir.IRFunc) []string {
	out := make([]string, 0, len(funcs))
	for _, fn := range funcs {
		out = append(out, fn.Name)
	}
	return out
}
