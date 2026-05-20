package lower

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
)

func TestLowerCatchHandlerCollectsStagedTypedTaskWrapper(t *testing.T) {
	prog := lowerProgramForCatchTest(t, `
enum OuterErr:
    case nope

enum TaskErr:
    case boom(Int, Int)
    case stopped

func worker() -> Int throws TaskErr:
    return 11

func fail() -> Int throws OuterErr:
    throw OuterErr.nope

func main() -> Int
uses runtime:
    return catch fail():
    case OuterErr.nope:
        catch core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<TaskErr>("worker")):
        case TaskErr.boom(left, right):
            left + right
        case TaskErr.stopped:
            0
`)

	wrapperName := typedTaskWrapperName("worker", "TaskErr")
	if !programHasFunc(prog, wrapperName) {
		t.Fatalf("typed task wrapper %q was not collected from catch handler; funcs=%v", wrapperName, programFuncNames(prog))
	}

	mainFn := requireIRFunc(t, prog, "main")
	if !hasCall(mainFn.Instrs, "__tetra_task_join_typed_5", 1) {
		t.Fatalf("main IR lacks staged typed-task join call: %#v", mainFn.Instrs)
	}
	if countCallsNamed(mainFn.Instrs, "__tetra_task_result_get") < 4 {
		t.Fatalf("main IR lacks staged result-slot loads: %#v", mainFn.Instrs)
	}
	if !hasInstructionPair(mainFn.Instrs, ir.IRCmpEqI32, ir.IRJmpIfZero) {
		t.Fatalf("main IR lacks catch enum compare/branch checks: %#v", mainFn.Instrs)
	}
}

func TestLowerMatchExprCollectsStagedTypedTaskWrapper(t *testing.T) {
	prog := lowerProgramForCatchTest(t, `
enum Choice:
    case left
    case right

enum TaskErr:
    case boom(Int, Int)
    case stopped

func worker() -> Int throws TaskErr:
    return 13

func main() -> Int
uses runtime:
    let choice: Choice = Choice.left
    return match choice:
    case Choice.left:
        catch core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<TaskErr>("worker")):
        case TaskErr.boom(left, right):
            left + right
        case TaskErr.stopped:
            0
    case Choice.right:
        catch core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<TaskErr>("worker")):
        case TaskErr.boom(otherLeft, otherRight):
            otherLeft + otherRight
        case TaskErr.stopped:
            0
`)

	wrapperName := typedTaskWrapperName("worker", "TaskErr")
	if !programHasFunc(prog, wrapperName) {
		t.Fatalf("typed task wrapper %q was not collected from match expression; funcs=%v", wrapperName, programFuncNames(prog))
	}
}

func TestLowerTryTypedTaskJoinUsesStagedResultSlots(t *testing.T) {
	prog := lowerProgramForCatchTest(t, `
enum TaskErr:
    case boom(Int, Int)
    case stopped

func worker() -> Int throws TaskErr:
    return 17

func caller() -> Int throws TaskErr
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)

func main() -> Int:
    return 0
`)

	wrapperName := typedTaskWrapperName("worker", "TaskErr")
	if !programHasFunc(prog, wrapperName) {
		t.Fatalf("typed task wrapper %q was not collected for try join; funcs=%v", wrapperName, programFuncNames(prog))
	}

	callerFn := requireIRFunc(t, prog, "caller")
	if !hasCall(callerFn.Instrs, "__tetra_task_join_typed_5", 1) {
		t.Fatalf("caller IR lacks staged typed-task try join call: %#v", callerFn.Instrs)
	}
	if countCallsNamed(callerFn.Instrs, "__tetra_task_result_get") < 4 {
		t.Fatalf("caller IR lacks staged typed-task result loads: %#v", callerFn.Instrs)
	}
	if countKind(callerFn.Instrs, ir.IRReturn) < 1 {
		t.Fatalf("caller IR lacks propagation return path: %#v", callerFn.Instrs)
	}
}

func TestLowerStagedTypedTaskPolicyFailureStagesStatus(t *testing.T) {
	prog := lowerProgramForCatchTest(t, `
enum TaskErr:
    case boom(Int, Int)
    case stopped

func worker() -> Int throws TaskErr
uses budget
budget(4):
    return 17

func main() -> Int
uses runtime, budget
budget(8):
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(left, right):
        left + right
    case TaskErr.stopped:
        0
`)

	workerFn := requireIRFunc(t, prog, "worker")
	if !workerFn.Policy.HasBudget || workerFn.Policy.FailLabel < 0 {
		t.Fatalf("worker IR lacks budget policy failure metadata: %#v", workerFn.Policy)
	}
	if countCallsNamed(workerFn.Instrs, "__tetra_task_result_begin") < 2 {
		t.Fatalf("worker IR did not stage both normal and policy-failure typed task results: %#v", workerFn.Instrs)
	}
}

func lowerProgramForCatchTest(t *testing.T, src string) *ir.IRProgram {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	return irProg
}

func requireIRFunc(t *testing.T, prog *ir.IRProgram, name string) ir.IRFunc {
	t.Helper()
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("function %q not found; funcs=%v", name, programFuncNames(prog))
	return ir.IRFunc{}
}

func programHasFunc(prog *ir.IRProgram, name string) bool {
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return true
		}
	}
	return false
}

func programFuncNames(prog *ir.IRProgram) []string {
	if prog == nil {
		return nil
	}
	names := make([]string, 0, len(prog.Funcs))
	for _, fn := range prog.Funcs {
		names = append(names, fn.Name)
	}
	return names
}

func hasCall(instrs []ir.IRInstr, name string, retSlots int) bool {
	for _, instr := range instrs {
		if instr.Kind == ir.IRCall && instr.Name == name && instr.RetSlots == retSlots {
			return true
		}
	}
	return false
}

func countCallsNamed(instrs []ir.IRInstr, name string) int {
	count := 0
	for _, instr := range instrs {
		if instr.Kind == ir.IRCall && instr.Name == name {
			count++
		}
	}
	return count
}
