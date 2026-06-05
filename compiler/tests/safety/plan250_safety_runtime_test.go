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

func TestPlan250RuntimeRejectsReservedExportOutsideRuntimeModule(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
@export("__tetra_custom")
func main() -> Int:
    return 0
`, "@export name '__tetra_custom' is reserved")
}

func TestPlan250RuntimeRejectsOpaqueCapabilityTokensInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_forged_fs_exists")
func forged_fs_exists(io_cap: cap.io) -> Int
uses io:
    if core.fs_exists("README.md", io_cap):
        return 42
    return 0

func main() -> Int:
    return 0
`, "exported function 'forged_fs_exists' cannot expose opaque capability token 'cap.io' in parameter 'io_cap'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_forged_mem_load")
func forged_mem_load(p: ptr, mem_cap: cap.mem) -> Int
uses mem:
    unsafe:
        return core.load_i32(p, mem_cap)

func main() -> Int:
    return 0
`, "exported function 'forged_mem_load' cannot expose opaque capability token 'cap.mem' in parameter 'mem_cap'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_cap")
func forged_cap() -> cap.io
uses capability:
    unsafe:
        return core.cap_io()

func main() -> Int:
    return 0
	`, "exported function 'forged_cap' cannot expose opaque capability token 'cap.io' in return type")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_mem_cap")
func forged_mem_cap() -> cap.mem
uses capability:
    unsafe:
        return core.cap_mem()

func main() -> Int:
    return 0
	`, "exported function 'forged_mem_cap' cannot expose opaque capability token 'cap.mem' in return type")
}

func TestPlan250RuntimeRejectsOpaqueIslandHandlesInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_island_byte_roundtrip")
func island_byte_roundtrip(isl: island) -> Int
uses alloc, islands, mem:
    var buf: []u8 = core.island_make_u8(isl, 1)
    buf[0] = 42
    return buf[0]

func main() -> Int:
    return 0
`, "exported function 'island_byte_roundtrip' cannot expose opaque island handle 'island' in parameter 'isl'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_free_island")
func free_island(isl: island) -> Int
uses islands, mem:
    free(isl)
    return 42

func main() -> Int:
    return 0
`, "exported function 'free_island' cannot expose opaque island handle 'island' in parameter 'isl'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_mint_island")
func mint_island() -> island
uses alloc, islands, mem:
    unsafe:
        return core.island_new(16)

func main() -> Int:
    return 0
`, "exported function 'mint_island' cannot expose opaque island handle 'island' in return type")
}

func TestPlan250RuntimeRejectsOpaqueActorAndTaskHandlesInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_send_actor")
func send_actor(peer: actor, value: Int) -> Int
uses actors:
    return core.send(peer, value)

func main() -> Int:
    return 0
	`, "exported function 'send_actor' cannot expose opaque runtime handle 'actor' in parameter 'peer'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_spawn_peer")
func spawn_peer() -> actor
uses actors:
    return core.spawn("worker")

func worker() -> Int:
    return 0

func main() -> Int:
    return 0
`, "exported function 'spawn_peer' cannot expose opaque runtime handle 'actor' in return type")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_close_group")
func close_group(group: task.group) -> Int
uses runtime:
    return core.task_group_close(group)

func main() -> Int:
    return 0
`, "exported function 'close_group' cannot expose opaque runtime handle 'task.group' in parameter 'group'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_join_task")
func join_task(task: task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int:
    return 0
	`, "exported function 'join_task' cannot expose opaque runtime handle 'task.i32' in parameter 'task'")

	testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 42

@export("ffi_spawn_task")
func spawn_task() -> task.i32
uses runtime:
    return core.task_spawn_i32("worker")

func main() -> Int:
    return 0
`, "exported function 'spawn_task' cannot expose opaque runtime handle 'task.i32' in return type")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_open_group")
func open_group() -> task.group
uses runtime:
    return core.task_group_open()

func main() -> Int:
    return 0
	`, "exported function 'open_group' cannot expose opaque runtime handle 'task.group' in return type")
}

func TestPlan250RuntimeRejectsAggregateOpaqueHandlesInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct IoBox:
    io: cap.io

@export("ffi_struct_fs_exists")
func struct_fs_exists(box: IoBox) -> Int
uses io:
    if core.fs_exists("README.md", box.io):
        return 42
    return 0

func main() -> Int:
    return 0
`, "exported function 'struct_fs_exists' cannot expose opaque capability token 'cap.io' through parameter 'box' type 'IoBox'")

	testkit.RequireFileCheckErrorContains(t, `
enum ActorEnvelope:
    case peer(actor)
    case empty

@export("ffi_send_enveloped_actor")
func send_enveloped_actor(msg: ActorEnvelope, value: Int) -> Int
uses actors:
    match msg:
    case ActorEnvelope.peer(peer):
        return core.send(peer, value)
    case ActorEnvelope.empty:
        return 0

func main() -> Int:
    return 0
`, "exported function 'send_enveloped_actor' cannot expose opaque runtime handle 'actor' through parameter 'msg' type 'ActorEnvelope'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_optional_group_status")
func optional_group_status(maybe: task.group?) -> Int
uses runtime:
    match maybe:
    case some(group):
        return core.task_group_status(group)
    case none:
        return 0

func main() -> Int:
    return 0
`, "exported function 'optional_group_status' cannot expose opaque runtime handle 'task.group' through parameter 'maybe' type 'task.group?'")

	testkit.RequireFileCheckErrorContains(t, `
struct IoBox:
    io: cap.io

@export("ffi_mint_io_box")
func mint_io_box() -> IoBox
uses capability:
    unsafe:
        return IoBox(io: core.cap_io())

func main() -> Int:
    return 0
`, "exported function 'mint_io_box' cannot expose opaque capability token 'cap.io' through return type 'IoBox'")
}

func TestPlan250RuntimeRejectsFunctionTypedValuesInExportedSignatures(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
@export("ffi_apply_callback")
func apply_callback(cb: fn(Int) -> Int, value: Int) -> Int:
    return cb(value)

func main() -> Int:
    return 0
`, "exported function 'apply_callback' cannot expose function-typed value 'fnptr' in parameter 'cb'")

	testkit.RequireFileSemanticCheckErrorContains(t, `
func add_one(value: Int) -> Int:
    return value + 1

@export("ffi_make_callback")
func make_callback() -> fn(Int) -> Int:
    return add_one

func main() -> Int:
    return 0
`, "exported function 'make_callback' cannot expose function-typed value 'fnptr' in return type")
}

func TestPlan250RuntimeRejectsAggregateFunctionTypedValuesInExportedSignatures(t *testing.T) {
	testkit.RequireFileSemanticCheckErrorContains(t, `
struct CallbackBox:
    cb: fn(Int) -> Int

@export("ffi_boxed_callback_apply")
func boxed_callback_apply(box: CallbackBox, value: Int) -> Int:
    return box.cb(value)

func main() -> Int:
    return 0
`, "exported function 'boxed_callback_apply' cannot expose function-typed value 'fnptr' through parameter 'box' type 'CallbackBox'")

	testkit.RequireFileSemanticCheckErrorContains(t, `
enum CallbackEnvelope:
    case call(fn(Int) -> Int)

@export("ffi_enveloped_callback_apply")
func enveloped_callback_apply(env: CallbackEnvelope, value: Int) -> Int:
    match env:
    case CallbackEnvelope.call(cb):
        return cb(value)

func main() -> Int:
    return 0
`, "exported function 'enveloped_callback_apply' cannot expose function-typed value 'fnptr' through parameter 'env' type 'CallbackEnvelope'")

	testkit.RequireFileSemanticCheckErrorContains(t, `
struct CallbackBox:
    cb: fn(Int) -> Int

func add_one(value: Int) -> Int:
    return value + 1

@export("ffi_make_callback_box")
func make_callback_box() -> CallbackBox:
    return CallbackBox(cb: add_one)

func main() -> Int:
    return 0
`, "exported function 'make_callback_box' cannot expose function-typed value 'fnptr' through return type 'CallbackBox'")
}

func TestPlan250RuntimeRejectsRawStringAndSliceViewsInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_string_first_byte")
func string_first_byte(text: String) -> Int:
    if text[0] == 65:
        return 42
    return 0

func main() -> Int:
    return 0
`, "exported function 'string_first_byte' cannot expose raw string view 'str' in parameter 'text'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_slice_first_byte")
func slice_first_byte(bytes: []u8) -> Int:
    if bytes[0] == 42:
        return 42
    return 0

func main() -> Int:
    return 0
`, "exported function 'slice_first_byte' cannot expose raw slice view '[]u8' in parameter 'bytes'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_make_string")
func make_string() -> String:
    return "A"

func main() -> Int:
    return 0
`, "exported function 'make_string' cannot expose raw string view 'str' in return type")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_make_slice")
func make_slice() -> []u8
uses alloc, mem:
    var bytes: []u8 = core.make_u8(1)
    bytes[0] = 42
    return bytes

func main() -> Int:
    return 0
`, "exported function 'make_slice' cannot expose raw slice view '[]u8' in return type")
}

func TestPlan250RuntimeRejectsAggregateRawStringAndSliceViewsInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct TextBox:
    text: String

@export("ffi_boxed_string_first_byte")
func boxed_string_first_byte(box: TextBox) -> Int:
    if box.text[0] == 65:
        return 42
    return 0

func main() -> Int:
    return 0
`, "exported function 'boxed_string_first_byte' cannot expose raw string view 'str' through parameter 'box' type 'TextBox'")

	testkit.RequireFileCheckErrorContains(t, `
struct BytesBox:
    bytes: []u8

@export("ffi_boxed_slice_first_byte")
func boxed_slice_first_byte(box: BytesBox) -> Int:
    if box.bytes[0] == 42:
        return 42
    return 0

func main() -> Int:
    return 0
`, "exported function 'boxed_slice_first_byte' cannot expose raw slice view '[]u8' through parameter 'box' type 'BytesBox'")

	testkit.RequireFileCheckErrorContains(t, `
enum TextEnvelope:
    case text(String)

@export("ffi_enveloped_string_first_byte")
func enveloped_string_first_byte(env: TextEnvelope) -> Int:
    match env:
    case TextEnvelope.text(text):
        if text[0] == 65:
            return 42
        return 0

func main() -> Int:
    return 0
`, "exported function 'enveloped_string_first_byte' cannot expose raw string view 'str' through parameter 'env' type 'TextEnvelope'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_optional_string_first_byte")
func optional_string_first_byte(maybe: String?) -> Int:
    match maybe:
    case some(text):
        if text[0] == 65:
            return 42
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`, "exported function 'optional_string_first_byte' cannot expose raw string view 'str' through parameter 'maybe' type 'str?'")

	testkit.RequireFileCheckErrorContains(t, `
struct TextBox:
    text: String

@export("ffi_make_string_box")
func make_string_box() -> TextBox:
    return TextBox(text: "A")

func main() -> Int:
    return 0
`, "exported function 'make_string_box' cannot expose raw string view 'str' through return type 'TextBox'")

	testkit.RequireFileCheckErrorContains(t, `
struct BytesBox:
    bytes: []u8

@export("ffi_make_slice_box")
func make_slice_box() -> BytesBox
uses alloc, mem:
    var bytes: []u8 = core.make_u8(1)
    bytes[0] = 42
    return BytesBox(bytes: bytes)

func main() -> Int:
    return 0
`, "exported function 'make_slice_box' cannot expose raw slice view '[]u8' through return type 'BytesBox'")
}

func TestPlan250RuntimeRejectsRawFixedArrayViewsInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_fixed_array_len")
func fixed_array_len(xs: [1]Int) -> Int:
    if xs.len == 1:
        return 42
    return xs.len

func main() -> Int:
    return 0
`, "exported function 'fixed_array_len' cannot expose raw fixed-array view '[1]i32' in parameter 'xs'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_fixed_array_first")
func fixed_array_first(xs: [1]Int) -> Int:
    return xs[0]

func main() -> Int:
    return 0
`, "exported function 'fixed_array_first' cannot expose raw fixed-array view '[1]i32' in parameter 'xs'")

	testkit.RequireFileCheckErrorContains(t, `
struct ArrayBox:
    items: [1]Int

var leaked: ArrayBox

@export("ffi_make_fixed_array")
func make_fixed_array() -> [1]Int:
    return leaked.items

func main() -> Int:
    return 0
`, "exported function 'make_fixed_array' cannot expose raw fixed-array view '[1]i32' in return type")
}

func TestPlan250RuntimeRejectsAggregateFixedArrayViewsInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct ArrayBox:
    items: [1]Int

@export("ffi_boxed_fixed_array_len")
func boxed_fixed_array_len(box: ArrayBox) -> Int:
    if box.items.len == 1:
        return 42
    return box.items.len

func main() -> Int:
    return 0
`, "exported function 'boxed_fixed_array_len' cannot expose raw fixed-array view '[1]i32' through parameter 'box' type 'ArrayBox'")

	testkit.RequireFileCheckErrorContains(t, `
enum ArrayEnvelope:
    case items([1]Int)

@export("ffi_enveloped_fixed_array_len")
func enveloped_fixed_array_len(env: ArrayEnvelope) -> Int:
    match env:
    case ArrayEnvelope.items(xs):
        if xs.len == 1:
            return 42
        return xs.len

func main() -> Int:
    return 0
`, "exported function 'enveloped_fixed_array_len' cannot expose raw fixed-array view '[1]i32' through parameter 'env' type 'ArrayEnvelope'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_optional_fixed_array_len")
func optional_fixed_array_len(maybe: [1]Int?) -> Int:
    match maybe:
    case some(xs):
        if xs.len == 1:
            return 42
        return xs.len
    case none:
        return 0

func main() -> Int:
    return 0
`, "exported function 'optional_fixed_array_len' cannot expose raw fixed-array view '[1]i32' through parameter 'maybe' type '[1]i32?'")

	testkit.RequireFileCheckErrorContains(t, `
struct ArrayBox:
    items: [1]Int

var leaked: ArrayBox

@export("ffi_make_fixed_array_box")
func make_fixed_array_box() -> ArrayBox:
    return leaked

func main() -> Int:
    return 0
`, "exported function 'make_fixed_array_box' cannot expose raw fixed-array view '[1]i32' through return type 'ArrayBox'")

	testkit.RequireFileCheckErrorContains(t, `
enum ArrayEnvelope:
    case items([1]Int)

var leaked: ArrayEnvelope

@export("ffi_make_fixed_array_envelope")
func make_fixed_array_envelope() -> ArrayEnvelope:
    return leaked

func main() -> Int:
    return 0
`, "exported function 'make_fixed_array_envelope' cannot expose raw fixed-array view '[1]i32' through return type 'ArrayEnvelope'")
}

func TestPlan250RuntimeRejectsBoolValuesInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_bool_gate")
func bool_gate(flag: Bool) -> Int:
    if flag:
        return 42
    return 0

func main() -> Int:
    return 0
`, "exported function 'bool_gate' cannot expose unnormalized bool 'bool' in parameter 'flag'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_is_ready")
func is_ready() -> Bool:
    return true

func main() -> Int:
    return 0
`, "exported function 'is_ready' cannot expose unnormalized bool 'bool' in return type")
}

func TestPlan250RuntimeRejectsAggregateBoolValuesInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct Gate:
    allow: Bool

@export("ffi_boxed_bool_gate")
func boxed_bool_gate(gate: Gate) -> Int:
    if gate.allow:
        return 42
    return 0

func main() -> Int:
    return 0
`, "exported function 'boxed_bool_gate' cannot expose unnormalized bool 'bool' through parameter 'gate' type 'Gate'")

	testkit.RequireFileCheckErrorContains(t, `
enum GateMsg:
    case allow(Bool)

@export("ffi_enveloped_bool_gate")
func enveloped_bool_gate(msg: GateMsg) -> Int:
    match msg:
    case GateMsg.allow(flag):
        if flag:
            return 42
        return 0

func main() -> Int:
    return 0
`, "exported function 'enveloped_bool_gate' cannot expose unnormalized bool 'bool' through parameter 'msg' type 'GateMsg'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_optional_bool_gate")
func optional_bool_gate(maybe: Bool?) -> Int:
    match maybe:
    case some(flag):
        if flag:
            return 42
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`, "exported function 'optional_bool_gate' cannot expose unnormalized bool 'bool' through parameter 'maybe' type 'bool?'")

	testkit.RequireFileCheckErrorContains(t, `
struct Gate:
    allow: Bool

var leaked: Gate

@export("ffi_make_gate")
func make_gate() -> Gate:
    return leaked

func main() -> Int:
    return 0
`, "exported function 'make_gate' cannot expose unnormalized bool 'bool' through return type 'Gate'")
}

func TestPlan250RuntimeRejectsEnumValuesInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum Route:
    case public
    case admin

@export("ffi_route_decision")
func route_decision(route: Route) -> Int:
    match route:
    case Route.public:
        return 1
    case Route.admin:
        return 42

func main() -> Int:
    return 0
`, "exported function 'route_decision' cannot expose forgeable enum discriminant 'Route' through parameter 'route' type 'Route'")

	testkit.RequireFileCheckErrorContains(t, `
enum Request:
    case read(Int)
    case admin(Int)

@export("ffi_request_decision")
func request_decision(req: Request) -> Int:
    match req:
    case Request.read(id):
        return id
    case Request.admin(level):
        return level

func main() -> Int:
    return 0
`, "exported function 'request_decision' cannot expose forgeable enum discriminant 'Request' through parameter 'req' type 'Request'")

	testkit.RequireFileCheckErrorContains(t, `
enum Route:
    case public
    case admin

var leaked: Route

@export("ffi_make_route")
func make_route() -> Route:
    return leaked

func main() -> Int:
    return 0
`, "exported function 'make_route' cannot expose forgeable enum discriminant 'Route' through return type 'Route'")
}

func TestPlan250RuntimeRejectsAggregateEnumValuesInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum Route:
    case public
    case admin

struct RouteBox:
    route: Route

@export("ffi_boxed_route_decision")
func boxed_route_decision(box: RouteBox) -> Int:
    match box.route:
    case Route.public:
        return 1
    case Route.admin:
        return 42

func main() -> Int:
    return 0
`, "exported function 'boxed_route_decision' cannot expose forgeable enum discriminant 'Route' through parameter 'box' type 'RouteBox'")

	testkit.RequireFileCheckErrorContains(t, `
enum Route:
    case public
    case admin

struct RouteBox:
    route: Route

var leaked: RouteBox

@export("ffi_make_route_box")
func make_route_box() -> RouteBox:
    return leaked

func main() -> Int:
    return 0
`, "exported function 'make_route_box' cannot expose forgeable enum discriminant 'Route' through return type 'RouteBox'")
}

func TestPlan250RuntimeRejectsOptionalValuesInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_optional_status")
func optional_status(code: Int?) -> Int:
    match code:
    case none:
        return 1
    case some(value):
        return value

func main() -> Int:
    return 0
`, "exported function 'optional_status' cannot expose forgeable optional presence tag 'i32?' through parameter 'code' type 'i32?'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_optional_iflet")
func optional_iflet(code: Int?) -> Int:
    if let some(value) = code:
        return value
    else:
        return 1

func main() -> Int:
    return 0
`, "exported function 'optional_iflet' cannot expose forgeable optional presence tag 'i32?' through parameter 'code' type 'i32?'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_make_optional")
func make_optional() -> Int?:
    return 42

func main() -> Int:
    return 0
`, "exported function 'make_optional' cannot expose forgeable optional presence tag 'i32?' through return type 'i32?'")
}

func TestPlan250RuntimeRejectsAggregateOptionalValuesInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
struct CodeBox:
    code: Int?

@export("ffi_boxed_optional_status")
func boxed_optional_status(box: CodeBox) -> Int:
    match box.code:
    case none:
        return 1
    case some(value):
        return value

func main() -> Int:
    return 0
`, "exported function 'boxed_optional_status' cannot expose forgeable optional presence tag 'i32?' through parameter 'box' type 'CodeBox'")

	testkit.RequireFileCheckErrorContains(t, `
struct CodeBox:
    code: Int?

@export("ffi_make_optional_box")
func make_optional_box() -> CodeBox:
    return CodeBox(code: 42)

func main() -> Int:
    return 0
`, "exported function 'make_optional_box' cannot expose forgeable optional presence tag 'i32?' through return type 'CodeBox'")
}

func TestPlan250RuntimeRejectsConsentTokensInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_require_consent")
func require_consent(token: consent.token) -> Int
uses privacy
privacy
consent(token):
    return 42

func main() -> Int:
    return 0
`, "exported function 'require_consent' cannot expose forgeable consent token 'consent.token' in parameter 'token'")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_make_consent")
func make_consent() -> consent.token
uses privacy
privacy:
    return core.consent_token()

func main() -> Int:
    return 0
`, "exported function 'make_consent' cannot expose forgeable consent token 'consent.token' in return type")
}

func TestPlan250RuntimeRejectsAggregateConsentTokensInExportedSignatures(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
repr(C) struct ConsentBox:
    token: consent.token

@export("ffi_boxed_consent")
func boxed_consent(box: ConsentBox) -> Int:
    return 42

func main() -> Int:
    return 0
`, "exported function 'boxed_consent' cannot expose forgeable consent token 'consent.token' through parameter 'box' type 'ConsentBox'")

	testkit.RequireFileCheckErrorContains(t, `
repr(C) struct ConsentBox:
    token: consent.token

@export("ffi_make_consent_box")
func make_consent_box() -> ConsentBox
uses privacy
privacy:
    let token: consent.token = core.consent_token()
    return ConsentBox(token: token)

func main() -> Int:
    return 0
`, "exported function 'make_consent_box' cannot expose forgeable consent token 'consent.token' through return type 'ConsentBox'")
}

func TestPlan250RuntimeRejectsGenericFunctionsWithExport(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_generic_id")
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return 42
`, "generic function 'id' cannot be exported; export a concrete monomorphic wrapper")

	testkit.RequireFileCheckErrorContains(t, `
@export("ffi_generic_id")
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return id(42)
`, "generic function 'id' cannot be exported; export a concrete monomorphic wrapper")
}

func TestPlan250RuntimeRejectsThrowingFunctionsWithExport(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
enum ReadError:
    case eof

@export("ffi_read_compact")
func read_compact(flag: Int) -> Int throws ReadError:
    if flag == 1:
        return 42
    throw ReadError.eof

func main() -> Int:
    return 0
`, "exported function 'read_compact' cannot throw typed error 'ReadError'; export a non-throwing wrapper with an explicit result type")

	testkit.RequireFileCheckErrorContains(t, `
enum ServiceError:
    case denied(Int)
    case offline

@export("ffi_read_payload")
func read_payload(flag: Int) -> Int throws ServiceError:
    if flag == 1:
        return 42
    throw ServiceError.denied(7)

func main() -> Int:
    return 0
`, "exported function 'read_payload' cannot throw typed error 'ServiceError'; export a non-throwing wrapper with an explicit result type")
}

func TestPlan250RuntimeRejectsOwnershipMarkedParametersWithExport(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "borrow",
			src: `
@export("ffi_borrow_int")
func borrow_int(value: borrow Int) -> Int:
    return value

func main() -> Int:
    return 0
`,
			want: "exported function 'borrow_int' cannot expose ownership marker 'borrow' on parameter 'value'; export a plain FFI-safe wrapper",
		},
		{
			name: "consume",
			src: `
@export("ffi_consume_int")
func consume_int(value: consume Int) -> Int:
    return value

func main() -> Int:
    return 0
`,
			want: "exported function 'consume_int' cannot expose ownership marker 'consume' on parameter 'value'; export a plain FFI-safe wrapper",
		},
		{
			name: "inout",
			src: `
@export("ffi_inout_int")
func inout_int(value: inout Int) -> Int:
    value = value + 1
    return value

func main() -> Int:
    return 0
`,
			want: "exported function 'inout_int' cannot expose ownership marker 'inout' on parameter 'value'; export a plain FFI-safe wrapper",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testkit.RequireFileCheckErrorContains(t, tc.src, tc.want)
		})
	}
}

func TestPlan250RuntimeAllowsOpaqueHandlesOnlyForInternalRuntimeExports(t *testing.T) {
	testkit.RequireFileSemanticCheckOK(t, `
module __rt.actors_sysv

@export("__tetra_task_spawn_i32")
func rt_task_spawn_i32(entry_id: i32) -> task.i32
uses runtime:
    return core.task_spawn_i32("worker")

@export("__tetra_task_join_i32")
func rt_task_join_i32(task: task.i32) -> i32
uses runtime:
    return core.task_join_i32(task)

@export("__tetra_timer_ready_ms")
func rt_timer_ready_ms(now_ms: i32, deadline_ms: i32) -> Bool:
    return now_ms >= deadline_ms

func worker() -> Int:
    return 1

func main() -> Int:
    return 0
`)

	testkit.RequireFileCheckErrorContains(t, `
module __rt.actors_sysv

@export("ffi_join_task")
func join_task(task: task.i32) -> Int:
    return 0

func main() -> Int:
    return 0
`, "exported function 'join_task' cannot expose opaque runtime handle 'task.i32' in parameter 'task'")

	testkit.RequireFileCheckErrorContains(t, `
module __rt.actors_sysv

@export("ffi_timer_ready")
func timer_ready() -> Bool:
    return true

func main() -> Int:
    return 0
`, "exported function 'timer_ready' cannot expose unnormalized bool 'bool' in return type")
}

func TestPlan250LinkObjectRejectsWrongCompilerVersionMetadata(t *testing.T) {
	tmp := t.TempDir()
	objPath := filepath.Join(tmp, "link_wrong_version.tobj")
	if err := compiler.WriteObject(objPath, &compiler.Object{
		Target:          "linux-x64",
		CompilerVersion: "wrong-version",
		Module:          "bad.link",
		Code:            []byte{0xC3},
		Symbols:         []compiler.Symbol{{Name: "bad_link_symbol", Offset: 0}},
	}); err != nil {
		t.Fatalf("WriteObject: %v", err)
	}
	outPath := filepath.Join(tmp, "hello")
	_, err := compiler.BuildFileWithStatsOpt(testkit.RepoPath(t, "examples", "hello.tetra"), outPath, "linux-x64", compiler.BuildOptions{LinkObjectPaths: []string{objPath}})
	if err == nil {
		t.Fatalf("expected link object compiler version diagnostic")
	}
	if !strings.Contains(err.Error(), "link object compiler version mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestPlan250RuntimeDiagnosticsPreserveExitAndPanicBoundaries(t *testing.T) {
	exit := compiler.TestRunnerSource{Name: "nonzero"}.ResultWithDuration(9, nil, 12)
	if exit.Passed || exit.Error != "exit code 9" || exit.ExitCode != 9 || exit.DurationMS != 12 {
		t.Fatalf("unexpected nonzero exit result: %#v", exit)
	}

	panicLike := compiler.TestRunnerSource{Name: "panic"}.ResultWithDuration(1, errForPlan250RuntimeDiagnostic("tetra panic(7): bounds"), 3)
	if panicLike.Passed || panicLike.ExitCode != 1 || !strings.Contains(panicLike.Error, "tetra panic(7): bounds") {
		t.Fatalf("unexpected panic-like result: %#v", panicLike)
	}
}

type errForPlan250RuntimeDiagnostic string

func (e errForPlan250RuntimeDiagnostic) Error() string {
	return string(e)
}
