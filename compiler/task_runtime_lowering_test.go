package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestTaskSpawnI32LowersToRuntimeSpawn(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(mainFn, "__tetra_task_spawn_i32") {
		t.Fatalf("main does not call __tetra_task_spawn_i32: %#v", mainFn.Instrs)
	}
	if !hasIRCall(mainFn, "__tetra_task_join_i32") {
		t.Fatalf("main does not call __tetra_task_join_i32: %#v", mainFn.Instrs)
	}
	if hasIRCall(mainFn, "worker") {
		t.Fatalf("main still calls worker directly during task spawn: %#v", mainFn.Instrs)
	}
}

func TestTaskSpawnI32TypedPayloadLowersToRuntimeWrapper(t *testing.T) {
	src := []byte(`
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(7)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	used, entries, spawnCount, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collectActorEntries: %v", err)
	}
	if !used {
		t.Fatalf("typed task runtime was not collected")
	}
	if hasString(entries, "worker") {
		t.Fatalf("typed task entries should use a wrapper, got %#v", entries)
	}
	if !hasPrefix(entries, "__tetra_task_typed_") {
		t.Fatalf("typed task wrapper entry missing: %#v", entries)
	}
	if spawnCount != 1 {
		t.Fatalf("typed task spawn count = %d, want 1", spawnCount)
	}

	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(mainFn, "__tetra_task_spawn_i32") {
		t.Fatalf("main does not call __tetra_task_spawn_i32: %#v", mainFn.Instrs)
	}
	if !hasIRCall(mainFn, "__tetra_task_join_typed_4") {
		t.Fatalf("main does not call __tetra_task_join_typed_4: %#v", mainFn.Instrs)
	}
	if hasIRCall(mainFn, "worker") {
		t.Fatalf("main still calls worker directly during typed task spawn: %#v", mainFn.Instrs)
	}
	if !hasIRFuncPrefix(irProg.Funcs, "__tetra_task_typed_") {
		var names []string
		for _, fn := range irProg.Funcs {
			names = append(names, fn.Name)
		}
		t.Fatalf("typed task wrapper IR function missing; funcs=%#v", names)
	}
}

func TestTaskSpawnI32TypedStagedSlotsFiveLowersToRuntimeStagedPath(t *testing.T) {
	src := []byte(`
enum TaskErr:
    case boom(Int, Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(5, 7)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b):
        a + b
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(mainFn, "__tetra_task_join_typed_5") {
		t.Fatalf("main does not call __tetra_task_join_typed_5: %#v", mainFn.Instrs)
	}
	if !hasIRCall(mainFn, "__tetra_task_result_get") {
		t.Fatalf("main does not call __tetra_task_result_get in staged path: %#v", mainFn.Instrs)
	}
}

func TestTaskSpawnI32TypedStagedSlotBuildAndRunSmoke(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		exitCode int
	}{
		{
			name: "slots_5",
			src: `
enum TaskErr:
    case boom(Int, Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(9, 12)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b):
        a + b
`,
			exitCode: 21,
		},
		{
			name: "slots_8",
			src: `
enum TaskErr:
    case boom(Int, Int, Int, Int, Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(1, 2, 3, 4, 5)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b, c, d, e):
        a + b + c + d + e
`,
			exitCode: 15,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			stdout, exitCode := buildAndRunWithOptions(t, tc.src, BuildOptions{})
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != tc.exitCode {
				t.Fatalf("exit code = %d, want %d", exitCode, tc.exitCode)
			}
		})
	}
}

func TestTaskSpawnI32TypedRejectsExplicitSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "main")
	if err := os.WriteFile(srcPath, []byte(`
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(42)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Runtime: RuntimeSelfHost})
	if err == nil {
		t.Fatalf("expected explicit selfhost typed task rejection")
	}
	if !strings.Contains(err.Error(), "self-host runtime does not support typed task handles") {
		t.Fatalf("error = %v", err)
	}
}

func TestTaskSpawnI32TypedStagedSlotsEightNestedSpawnAutoAndBuiltinRuntimeParity(t *testing.T) {
	src := `
enum TaskErr:
    case boom(Int, Int, Int, Int, Int)

func child() -> Int throws TaskErr:
    throw TaskErr.boom(1, 2, 3, 4, 5)

func worker() -> Int throws TaskErr
uses runtime:
    let child_task = core.task_spawn_i32_typed<TaskErr>("child")
    return catch core.task_join_i32_typed<TaskErr>(child_task):
    case TaskErr.boom(a, b, c, d, e):
        a + b + c + d + e

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b, c, d, e):
        90 + a + b + c + d + e
`
	for _, tc := range []struct {
		name string
		rt   RuntimeMode
	}{
		{name: "auto", rt: RuntimeAuto},
		{name: "builtin", rt: RuntimeBuiltin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: tc.rt})
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != 15 {
				t.Fatalf("exit code = %d, want 15", exitCode)
			}
		})
	}
}

func TestTaskSpawnI32TypedStagedSlotsRejectsExplicitSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "main")
	if err := os.WriteFile(srcPath, []byte(`
enum TaskErr:
    case boom(Int, Int, Int, Int, Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(1, 2, 3, 4, 5)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b, c, d, e):
        a + b + c + d + e
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Runtime: RuntimeSelfHost})
	if err == nil {
		t.Fatalf("expected explicit selfhost staged typed task rejection")
	}
	if !strings.Contains(err.Error(), "self-host runtime does not support typed task handles") {
		t.Fatalf("error = %v", err)
	}
}

func TestDocumentedTypedTaskSelfHostRuntimeDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "direct_slots",
			src: `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(42)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
`,
		},
		{
			name: "staged_slots",
			src: `
enum TaskErr:
    case boom(Int, Int, Int, Int, Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(1, 2, 3, 4, 5)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b, c, d, e):
        a + b + c + d + e
`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "main.tetra")
			outPath := filepath.Join(tmp, "main")
			if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}
			_, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Runtime: RuntimeSelfHost})
			if err == nil {
				t.Fatalf("expected explicit selfhost typed task rejection")
			}
			if !strings.Contains(err.Error(), "self-host runtime does not support typed task handles") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestTaskSpawnI32TypedRejectsHandleSlotsAboveEightEarly(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
enum TaskErr:
    case huge(Int, Int, Int, Int, Int, Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.huge(1, 2, 3, 4, 5, 6)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.huge(a, b, c, d, e, f):
        a + b + c + d + e + f
`, "typed task supports at most 8 slots")
}
