package tasks

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func TestWrapperNameIsStableAndDistinct(t *testing.T) {
	name := WrapperName("worker", "TaskErr")
	if !strings.HasPrefix(name, "__tetra_task_typed_") {
		t.Fatalf("wrapper name %q missing prefix", name)
	}
	if name != WrapperName("worker", "TaskErr") {
		t.Fatalf("wrapper name is not stable")
	}
	if name == WrapperName("worker", "OtherErr") {
		t.Fatalf("wrapper name did not include error type")
	}
}

func TestCollectStagedTypedTaskTargets(t *testing.T) {
	wrappers := []Wrapper{
		{Target: "small", SlotCount: 4, ErrorType: "TaskErr", TargetThrowsType: "TaskErr"},
		{Target: "large", SlotCount: 6, ErrorType: "TaskErr", TargetThrowsType: "TaskErr"},
		{Target: "no_error", SlotCount: 6},
		{Target: "mismatch", SlotCount: 6, ErrorType: "TaskErr", TargetThrowsType: "OtherErr"},
	}

	targets := CollectStagedTargets(wrappers)
	if len(targets) != 1 {
		t.Fatalf("targets = %#v", targets)
	}
	if got := targets["large"]; got != (StagedTarget{SlotCount: 6, ErrorType: "TaskErr"}) {
		t.Fatalf("large staged target = %#v", got)
	}
}

func TestLowerWrapperUsesInjectedUnsupportedDiagnostic(t *testing.T) {
	_, err := LowerWrapper(
		Wrapper{Name: "bad", SlotCount: 9},
		func(pos frontend.Position, format string, args ...interface{}) error {
			return &frontend.DiagnosticError{
				Info: frontend.Diagnostic{Code: "TETRA3002", Message: "unsupported"},
			}
		},
	)
	if err == nil {
		t.Fatalf("expected unsupported wrapper error")
	}
	if got := err.Error(); !strings.Contains(got, "unsupported") {
		t.Fatalf("unexpected error %q", got)
	}
}

func TestLowerWrapperBuildsSmallSlotIR(t *testing.T) {
	fn, err := LowerWrapper(Wrapper{
		Name:       "__tetra_task_typed_test",
		Target:     "worker",
		SlotCount:  3,
		StatusSlot: 2,
	}, nil)
	if err != nil {
		t.Fatalf("LowerWrapper: %v", err)
	}
	if fn.Name != "__tetra_task_typed_test" || fn.LocalSlots != 4 || fn.ReturnSlots != 1 {
		t.Fatalf("unexpected wrapper func metadata: %#v", fn)
	}
	if len(fn.Instrs) == 0 || fn.Instrs[0].Kind != ir.IRCall || fn.Instrs[0].Name != "worker" ||
		fn.Instrs[0].RetSlots != 3 {
		t.Fatalf("first instr = %#v", fn.Instrs)
	}
	if last := fn.Instrs[len(fn.Instrs)-1]; last.Kind != ir.IRReturn {
		t.Fatalf("last instr = %#v", last)
	}
}

func TestCallExprCloneAndThrowingLayout(t *testing.T) {
	orig := &frontend.CallExpr{
		Name: "worker",
		Args: []frontend.Expr{&frontend.NumberExpr{Value: 1}},
	}
	clone := CallExprWithName(orig, "other")
	if clone == orig {
		t.Fatalf("expected cloned call when name changes")
	}
	if clone.Name != "other" || orig.Name != "worker" {
		t.Fatalf("call clone names = clone %q original %q", clone.Name, orig.Name)
	}
	if CallExprWithName(orig, "worker") != orig {
		t.Fatalf("same-name call should be reused")
	}

	types := map[string]*semantics.TypeInfo{
		"i32":     {Name: "i32", SlotCount: 1},
		"TaskErr": {Name: "TaskErr", SlotCount: 2},
	}
	success, failure, compact, err := ThrowingLayout("i32", "TaskErr", types)
	if err != nil {
		t.Fatalf("ThrowingLayout: %v", err)
	}
	if success != 1 || failure != 2 || compact {
		t.Fatalf("layout = success %d failure %d compact %v", success, failure, compact)
	}
	if ThrowingReturnSlotCount(1, 1) != 2 || ThrowingReturnSlotCount(1, 2) != 4 {
		t.Fatalf("unexpected throwing return slot counts")
	}
	if !IsTypedTaskJoinCall("core.task_join_i32_typed") ||
		TypedTaskJoinRuntimeSymbol(6) != "__tetra_task_join_typed_6" {
		t.Fatalf("typed task join helpers mismatch")
	}
	if !IsThrowIntLike("task.error") || !IsThrowIntLike("c_uint") {
		t.Fatalf("throw int-like classifier mismatch")
	}
}
