package compiler_test

import (
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/testkit"
)

func requireCheckFileErrorContainsAll(t *testing.T, src string, wants ...string) {
	t.Helper()
	file, err := compiler.ParseFile([]byte(src), "test.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: file.Module,
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{file.Module: file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected error containing %v, got nil", wants)
	}
	for _, want := range wants {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error containing %q, got: %v", want, err)
		}
	}
}

func TestPlan250SafetyBorrowEscapeAcrossBranchMerge(t *testing.T) {
	requireCheckFileErrorContainsAll(t, `
func leak(flag: Int, borrowed: borrow []u8) -> []u8
uses alloc, mem:
    var out: []u8 = make_u8(1)
    if flag:
        out = borrowed
    return out

func main() -> Int:
    return 0
`, "ambiguous region for 'out'", "control-flow merge")
}

func TestPlan250SafetyUseAfterConsumeAcrossLoopsAndConditionals(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let value: Int = 1
    var i: Int = 0
    while i < 2:
        let moved: Int = take(value)
        i = i + 1
    return value
`, "cannot use consumed value 'value'")

	testkit.RequireFileCheckErrorContains(t, `
func take(value: consume Int) -> Int:
    return value

func main() -> Int:
    let value: Int = 1
    if 1:
        let left: Int = take(value)
    else:
        let right: Int = take(value)
    return value
`, "cannot use consumed value 'value'")
}

func TestPlan250SafetyResourceLifetimeForLoopedTaskGroupAndIslandHandles(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    var i: Int = 0
    while i < 2:
        let _: Int = core.task_group_close(group)
        i = i + 1
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    return core.task_join_i32(task)
`, "cannot use closed resource 'group'")

	testkit.RequireFileCheckErrorContains(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe:
        let isl: island = core.island_new(16)
        var i: Int = 0
        while i < 2:
            free(isl)
            i = i + 1
        free(isl)
    return 0
`, "cannot use freed resource 'isl'")
}

func TestPlan250SafetySendabilityAcrossModuleBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		wantErr []string
	}{
		{
			name: "reject uncopied string payload",
			files: map[string]string{
				"lib/messages.tetra": `module lib.messages

enum Msg:
    case text(String)
`,
				"app/main.tetra": `module app.main
import lib.messages.{Msg}

func worker() -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send_typed(peer, Msg.text("remote"))
`,
			},
			wantErr: []string{"cannot cross actor boundary without copy", "<borrow>"},
		},
		{
			name: "reject actor handle payload",
			files: map[string]string{
				"lib/messages.tetra": `module lib.messages

enum Msg:
    case peer(actor)
`,
				"app/main.tetra": `module app.main
import lib.messages.{Msg}

func worker() -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send_typed(peer, Msg.peer(peer))
`,
			},
			wantErr: []string{"typed actor message payload must be value-only", "actor"},
		},
		{
			name: "reject raw pointer payload",
			files: map[string]string{
				"lib/messages.tetra": `module lib.messages

enum Msg:
    case raw(ptr)
`,
				"app/main.tetra": `module app.main
import lib.messages.{Msg}

func worker() -> Int:
    return 0

func main() -> Int
uses actors, alloc, mem:
    let peer: actor = core.spawn("worker")
    unsafe:
        let raw: ptr = core.alloc_bytes(4)
        return core.send_typed(peer, Msg.raw(raw))
`,
			},
			wantErr: []string{"typed actor message payload must be value-only", "ptr"},
		},
		{
			name: "reject capability payload",
			files: map[string]string{
				"lib/messages.tetra": `module lib.messages

enum Msg:
    case token(cap.mem)
`,
				"app/main.tetra": `module app.main
import lib.messages.{Msg}

func worker() -> Int:
    return 0

func main() -> Int
uses actors, mem, effects.cap.mem, effects.memory:
    let peer: actor = core.spawn("worker")
    unsafe:
        let token: cap.mem = core.cap_mem()
        return core.send_typed(peer, Msg.token(token))
`,
			},
			wantErr: []string{"typed actor message payload must be value-only", "cap.mem"},
		},
		{
			name: "allow value-only payload",
			files: map[string]string{
				"lib/messages.tetra": `module lib.messages

enum Msg:
    case ok(Int, Bool)
`,
				"app/main.tetra": `module app.main
import lib.messages.{Msg}

func worker() -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send_typed(peer, Msg.ok(7, true))
`,
			},
		},
		{
			name: "allow island transfer payload",
			files: map[string]string{
				"lib/messages.tetra": `module lib.messages

enum Msg:
    case move(island)
`,
				"app/main.tetra": `module app.main
import lib.messages.{Msg}

func worker() -> Int:
    return 0

func main() -> Int
uses actors, alloc, islands, mem:
    let peer: actor = core.spawn("worker")
    unsafe:
        let isl: island = core.island_new(8)
        return core.send_typed(peer, Msg.move(isl))
`,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			testkit.WriteFiles(t, tmp, tt.files)
			entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

			world, err := compiler.LoadWorld(entry)
			if err != nil {
				t.Fatalf("LoadWorld: %v", err)
			}
			checked, err := compiler.CheckWorld(world)
			if len(tt.wantErr) == 0 {
				if err != nil {
					t.Fatalf("unexpected cross-module typed actor payload error: %v", err)
				}
				if _, lowerErr := compiler.Lower(checked); lowerErr != nil {
					t.Fatalf("Lower: %v", lowerErr)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected cross-module typed actor payload diagnostic")
			}
			for _, want := range tt.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error = %v, want substring %q", err, want)
				}
			}

			worldAgain, loadErr := compiler.LoadWorld(entry)
			if loadErr != nil {
				t.Fatalf("LoadWorld second pass: %v", loadErr)
			}
			_, errAgain := compiler.CheckWorld(worldAgain)
			if errAgain == nil {
				t.Fatalf("expected cross-module typed actor payload diagnostic on second pass")
			}
			if err.Error() != errAgain.Error() {
				t.Fatalf("diagnostic changed between runs:\nfirst: %v\nsecond: %v", err, errAgain)
			}
		})
	}
}

func TestPlan250SafetyRaceRejectsWorkerEffectCapabilityCallGraphs(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "actor worker rejects transitive memory allocation effect",
			src: `
func allocate() -> Int
uses alloc, mem:
    unsafe:
        let _: ptr = core.alloc_bytes(4)
    return 1

func worker() -> Int
uses actors, alloc, mem:
    return allocate()

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send(peer, 1)
`,
			want: "spawn target 'worker' uses effect 'alloc' and cannot cross actor boundary",
		},
		{
			name: "task worker rejects capability effect",
			src: `
func worker() -> Int
uses capability, mem:
    unsafe:
        let _: cap.mem = core.cap_mem()
    return 1

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			want: "task_spawn_i32 target 'worker' uses effect 'capability' and cannot cross task boundary",
		},
		{
			name: "typed task worker rejects capability effect",
			src: `
enum TaskErr:
    case failed

func worker() -> Int throws TaskErr
uses capability, mem:
    unsafe:
        let _: cap.mem = core.cap_mem()
    return 1

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return core.task_join_i32_typed<TaskErr>(task)
`,
			want: "task_spawn_i32_typed target 'worker' uses effect 'capability' and cannot cross task boundary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireFileCheckErrorContains(t, tt.src, tt.want)
		})
	}
}

func TestPlan250SafetyTypedTaskRejectsNonSendableErrorPayloads(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "string payload",
			src: `
enum TaskErr:
    case boom(String)

func worker() -> Int throws TaskErr:
    return 1

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return 0
`,
			want: "str",
		},
		{
			name: "ptr payload",
			src: `
enum TaskErr:
    case bad(ptr)

func worker() -> Int throws TaskErr:
    return 1

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return 0
`,
			want: "ptr",
		},
		{
			name: "capability payload",
			src: `
enum TaskErr:
    case token(cap.mem)

func worker() -> Int throws TaskErr:
    return 1

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return 0
`,
			want: "cap.mem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireCheckFileErrorContainsAll(t, tt.src, "typed task error payload must be sendable across task boundary", tt.want)
		})
	}
}

func TestPlan250SafetyBudgetGuardsAreDeterministicAcrossLowerings(t *testing.T) {
	src := []byte(`
func tick(x: Int) -> Int
uses budget
budget(3):
    return x + 1

func main() -> Int
uses budget
budget(6):
    return tick(41)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	first, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower first: %v", err)
	}
	second, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower second: %v", err)
	}
	firstMain := findPlan250IRFunc(t, first.Funcs, "main")
	secondMain := findPlan250IRFunc(t, second.Funcs, "main")
	if !hasPlan250InstrKind(&firstMain, ir.IRSubI32) || !hasPlan250InstrKind(&firstMain, ir.IRJmpIfZero) {
		t.Fatalf("main missing budget guard instructions: %#v", firstMain.Instrs)
	}
	if len(firstMain.Instrs) != len(secondMain.Instrs) {
		t.Fatalf("budget lowering changed instruction count: %d != %d", len(firstMain.Instrs), len(secondMain.Instrs))
	}
	for i := range firstMain.Instrs {
		if firstMain.Instrs[i].Kind != secondMain.Instrs[i].Kind ||
			firstMain.Instrs[i].Label != secondMain.Instrs[i].Label ||
			firstMain.Instrs[i].Local != secondMain.Instrs[i].Local ||
			firstMain.Instrs[i].Imm != secondMain.Instrs[i].Imm {
			t.Fatalf("budget lowering differs at %d:\nfirst=%#v\nsecond=%#v", i, firstMain.Instrs[i], secondMain.Instrs[i])
		}
	}
}

func TestPlan250SafetyBudgetGuardsCoverIndexedAccess(t *testing.T) {
	src := []byte(`
func main() -> Int
uses alloc, budget, mem
budget(8):
    var xs: []i32 = core.make_i32(1)
    xs[0] = 41
    return xs[0] + 1
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findPlan250IRFunc(t, irProg.Funcs, "main")
	for _, kind := range []ir.IRInstrKind{ir.IRIndexStoreI32, ir.IRIndexLoadI32} {
		idx := firstPlan250InstrKind(mainFn.Instrs, kind)
		if idx < 0 {
			t.Fatalf("main missing %v in indexed-access budget test: %#v", kind, mainFn.Instrs)
		}
		if !hasPlan250BudgetGuardBefore(mainFn, idx) {
			t.Fatalf("missing budget guard before %v at instr %d: %#v", kind, idx, mainFn.Instrs)
		}
	}
}

func TestPlan250SafetyBudgetContextRejectsOversizedDirectTaskAndActorTargets(t *testing.T) {
	requireCheckFileErrorContainsAll(t, `
func callee() -> Int
uses budget
budget(6):
    return 1

func main() -> Int
uses budget
budget(5):
    return callee()
`, "budget context", "callee", "requires caller budget at least 6", "got 5")

	requireCheckFileErrorContainsAll(t, `
func worker() -> Int
uses budget
budget(5):
    return 1

func main() -> Int
uses runtime, budget
budget(4):
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`, "budget context", "task_spawn_i32 target 'worker'", "requires caller budget at least 5", "got 4")

	requireCheckFileErrorContainsAll(t, `
func actor_worker() -> Int
uses budget
budget(5):
    return 1

func main() -> Int
uses actors, budget
budget(4):
    let peer: actor = core.spawn("actor_worker")
    return 0
`, "budget context", "spawn target 'actor_worker'", "requires caller budget at least 5", "got 4")
}

func TestPlan250SafetyBudgetContextRejectsOversizedFunctionTypedTargets(t *testing.T) {
	requireCheckFileErrorContainsAll(t, `
func callee(x: Int) -> Int
uses budget
budget(6):
    return x + 1

func main() -> Int
uses budget
budget(5):
    let f: fn(Int) -> Int uses budget = callee
    return f(41)
`, "budget context", "call to callback 'f'", "requires caller budget at least 6", "got 5")

	requireCheckFileErrorContainsAll(t, `
var cb: fn(Int) -> Int uses budget = callee

func callee(x: Int) -> Int
uses budget
budget(6):
    return x + 1

func main() -> Int
uses budget
budget(5):
    return cb(41)
`, "budget context", "function-typed global call 'cb'", "requires caller budget at least 6", "got 5")

	requireCheckFileErrorContainsAll(t, `
func callee(x: Int) -> Int
uses budget
budget(6):
    return x + 1

func apply(x: Int, cb: fn(Int) -> Int uses budget) -> Int
uses budget
budget(5):
    return cb(x)

func main() -> Int
uses budget
budget(5):
    return apply(41, callee)
`, "budget context", "callee", "requires caller budget at least 6", "got 5")
}

func TestPlan250SafetyBudgetContextRejectsZeroCallerBudgetForBudgetedEdges(t *testing.T) {
	requireCheckFileErrorContainsAll(t, `
func callee() -> Int
uses budget
budget(6):
    return 1

func main() -> Int
uses budget
budget(0):
    return callee()
`, "budget context", "callee", "requires caller budget at least 6", "got 0")

	requireCheckFileErrorContainsAll(t, `
func worker() -> Int
uses budget
budget(5):
    return 1

func main() -> Int
uses runtime, budget
budget(0):
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`, "budget context", "task_spawn_i32 target 'worker'", "requires caller budget at least 5", "got 0")

	requireCheckFileErrorContainsAll(t, `
func actor_worker() -> Int
uses budget
budget(5):
    return 1

func main() -> Int
uses actors, budget
budget(0):
    let peer: actor = core.spawn("actor_worker")
    return 0
`, "budget context", "spawn target 'actor_worker'", "requires caller budget at least 5", "got 0")
}

func TestPlan250SafetyBudgetContextClosureUsesOwnBudgetContext(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
func callee() -> Int
uses budget
budget(6):
    return 1

func main() -> Int
uses budget
budget(1):
    let f: ptr = fn() -> Int
    uses budget
    budget(6):
        return callee()
    return 0
`)
}

func TestPlan250SafetyBudgetContextAllowsCoveredDirectTaskAndActorTargets(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
func callee() -> Int
uses budget
budget(5):
    return 1

func worker() -> Int
uses budget
budget(4):
    return 2

func actor_worker() -> Int
uses budget
budget(4):
    return 3

func main() -> Int
uses actors, budget, runtime
budget(12):
    let task: task.i32 = core.task_spawn_i32("worker")
    let peer: actor = core.spawn("actor_worker")
    let ignored: Int = callee()
    return core.task_join_i32(task)
`)
}

func firstPlan250InstrKind(instrs []ir.IRInstr, kind ir.IRInstrKind) int {
	for i, instr := range instrs {
		if instr.Kind == kind {
			return i
		}
	}
	return -1
}

func findPlan250IRFunc(t *testing.T, funcs []compiler.IRFunc, name string) compiler.IRFunc {
	t.Helper()
	for _, fn := range funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("missing IR func %q in %#v", name, funcs)
	return compiler.IRFunc{}
}

func hasPlan250InstrKind(fn *compiler.IRFunc, kind ir.IRInstrKind) bool {
	for _, instr := range fn.Instrs {
		if instr.Kind == kind {
			return true
		}
	}
	return false
}

func hasPlan250BudgetGuardBefore(fn compiler.IRFunc, chargedIdx int) bool {
	policy := fn.Policy
	if !policy.HasBudget {
		return false
	}
	for guardStart := chargedIdx - 8; guardStart >= 0 && guardStart >= chargedIdx-16; guardStart-- {
		if !matchesPlan250BudgetGuardAt(fn.Instrs, guardStart, policy.BudgetLocal, policy.FailLabel) {
			continue
		}
		guardEnd := guardStart + 8
		for i := guardEnd; i < chargedIdx; i++ {
			if fn.Instrs[i].Kind != ir.IRLoadLocal {
				return false
			}
		}
		return true
	}
	return false
}

func matchesPlan250BudgetGuardAt(instrs []ir.IRInstr, idx int, budgetLocal int, failLabel int) bool {
	return idx >= 0 &&
		idx+7 < len(instrs) &&
		instrs[idx].Kind == ir.IRLoadLocal &&
		instrs[idx].Local == budgetLocal &&
		instrs[idx+1].Kind == ir.IRConstI32 &&
		instrs[idx+1].Imm == 1 &&
		instrs[idx+2].Kind == ir.IRSubI32 &&
		instrs[idx+3].Kind == ir.IRStoreLocal &&
		instrs[idx+3].Local == budgetLocal &&
		instrs[idx+4].Kind == ir.IRLoadLocal &&
		instrs[idx+4].Local == budgetLocal &&
		instrs[idx+5].Kind == ir.IRConstI32 &&
		instrs[idx+5].Imm == 0 &&
		instrs[idx+6].Kind == ir.IRCmpGeI32 &&
		instrs[idx+7].Kind == ir.IRJmpIfZero &&
		instrs[idx+7].Label == failLabel
}
