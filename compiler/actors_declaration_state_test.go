package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/target"
)

func TestActorsTypedMessagesCheckAndLower(t *testing.T) {
	src := []byte(`
enum CounterMsg:
    case inc(Int, Int)
    case reset

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _sent: Int = core.send_typed(peer, CounterMsg.inc(20, 22))
    let msg: CounterMsg = core.recv_typed<CounterMsg>()
    match msg:
    case CounterMsg.inc(lhs, rhs):
        return lhs + rhs
    case CounterMsg.reset:
        return 0

func worker() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestActorDeclarationMVPCheckAndLower(t *testing.T) {
	src := []byte(`
actor Worker:
    func run() -> Int:
        return 7

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Worker.run")
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestActorDeclarationAllowsImmutableStateFields(t *testing.T) {
	src := []byte(`
actor Worker:
    val id: Int = 7
    const limit: Int = 9
    func run() -> Int:
        return 7

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Worker.run")
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestActorDeclarationAllowsMutableStateField(t *testing.T) {
	requireCheckFileOK(t, `
actor Worker:
    var count: Int = 0
    func run() -> Int:
        count = count + 1
        return count

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Worker.run")
    return 0
`)
}

func TestActorDeclarationStateFieldAccessUsesConstInitializer(t *testing.T) {
	src := []byte(`
actor Worker:
    val step: Int = 7
    const enabled: Bool = true
    func run() -> Int:
        if enabled:
            return step
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Worker.run")
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestActorStateLowerUsesRuntimeLoadStoreCalls(t *testing.T) {
	src := []byte(`
actor Worker:
    var count: Int = 0
    const enabled: Bool = true
    func run() -> Int:
        if enabled:
            count = count + 1
        return count

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Worker.run")
    return 0
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
	runFn := findIRFunc(t, irProg.Funcs, "Worker.run")
	if !hasIRCall(runFn, "__tetra_actor_state_load") {
		t.Fatalf("Worker.run missing __tetra_actor_state_load call: %#v", runFn.Instrs)
	}
	if !hasIRCall(runFn, "__tetra_actor_state_store") {
		t.Fatalf("Worker.run missing __tetra_actor_state_store call: %#v", runFn.Instrs)
	}
}

func TestActorStateRuntimeAutoBuildAndRunSmoke(t *testing.T) {
	src := `
actor Counter:
    var count: Int = 0
    const enabled: Bool = true
    func run() -> Int
    uses actors:
        let delta: Int = core.recv()
        if enabled:
            count = count + delta + 1
        let _sent: Int = core.send(core.sender(), count)
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Counter.run")
    let _sent: Int = core.send(peer, 41)
    return core.recv()
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeAuto})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want 42", exitCode)
	}
}

func TestDocumentedActorStateRuntimeBoundaryAndDiagnostics(t *testing.T) {
	mode, err := selectRuntimeMode(RuntimeAuto, runtimeUsageProfile{actorStateUsed: true})
	if err != nil {
		t.Fatalf("selectRuntimeMode: %v", err)
	}
	if mode != RuntimeBuiltin {
		t.Fatalf("actor-state auto runtime = %v, want builtin", mode)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "main")
	if err := os.WriteFile(srcPath, []byte(`
actor Worker:
    val title: String = "worker"
    func run() -> Int:
        return 0

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Worker.run")
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, err = BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Runtime: RuntimeSelfHost})
	if err == nil {
		t.Fatalf("expected actor-state unsupported type diagnostic")
	}
	if !strings.Contains(err.Error(), "actor state field 'title' type 'str' is not supported; supported actor state field types are Int, Bool, UInt8, UInt16, and task.error") {
		t.Fatalf("error = %v", err)
	}
}

func TestActorStateExtendedScalarsRuntimeAutoBuildAndRunSmoke(t *testing.T) {
	src := `
actor Counter:
    var err: task.error = 0
    var step: UInt8 = 1
    const boost: UInt16 = 2
    func run() -> Int
    uses actors:
        let delta: Int = core.recv()
        err = err + 1
        step = step + 1
        let total: Int = delta + err + step + boost
        let _sent: Int = core.send(core.sender(), total)
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Counter.run")
    let _sent: Int = core.send(peer, 1)
    return core.recv()
`
	for _, tc := range []struct {
		name string
		rt   RuntimeMode
	}{
		{name: "auto", rt: RuntimeAuto},
		{name: "selfhost", rt: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: tc.rt})
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != 6 {
				t.Fatalf("exit code = %d, want 6", exitCode)
			}
		})
	}
}

func TestActorStateSelfHostRuntimeBuildAndRunSmoke(t *testing.T) {
	src := `
actor Counter:
    var count: Int = 0
    func run() -> Int
    uses actors:
        let delta: Int = core.recv()
        count = count + delta
        let _sent: Int = core.send(core.sender(), count)
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Counter.run")
    let _sent: Int = core.send(peer, 42)
    return core.recv()
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeSelfHost})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want 42", exitCode)
	}
}

func TestActorDeclarationRequiresStateFieldInitializer(t *testing.T) {
	requireCheckFileErrorContains(t, `
actor Worker:
    val step: Int
    func run() -> Int:
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Worker.run")
    return 0
`, "requires a compile-time constant initializer")
}

func TestActorDeclarationRejectsUnsupportedStateFieldType(t *testing.T) {
	requireCheckFileErrorContains(t, `
actor Worker:
    val title: String = "worker"
    func run() -> Int:
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Worker.run")
    return 0
	`, "actor state field 'title' type 'str' is not supported; supported actor state field types are Int, Bool, UInt8, UInt16, and task.error")
}

func TestActorDeclarationAllowsExtendedScalarStateFieldTypes(t *testing.T) {
	requireCheckFileOK(t, `
actor Worker:
    var err: task.error = 0
    val step: UInt8 = 1
    const boost: UInt16 = 2
    func run() -> Int:
        err = err + 1
        return err + step + boost

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Worker.run")
    let _sent: Int = core.send(peer, 1)
    return 0
`)
}

func TestActorDeclarationRejectsPtrStateFieldType(t *testing.T) {
	requireCheckFileErrorContains(t, `
actor Worker:
    val raw: ptr = 0
    func run() -> Int:
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Worker.run")
    return 0
`, "actor state field 'raw' type 'ptr' is not supported; supported actor state field types are Int, Bool, UInt8, UInt16, and task.error")
}

func TestActorDeclarationRejectsNonConstStateInitializer(t *testing.T) {
	requireCheckFileErrorContains(t, `
actor Worker:
    val step: Int = core.recv()
    func run() -> Int
    uses actors:
        return step

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Worker.run")
    return 0
`, "initializer must be a compile-time constant Int/Bool expression")
}

func TestActorDeclarationMethodRequiresExplicitUsesActors(t *testing.T) {
	requireCheckFileErrorContains(t, `
actor Worker:
    func run() -> Int:
        let me: actor = core.self()
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Worker.run")
    return 0
`, "function 'Worker.run' uses effect 'actors'")

	requireCheckFileOK(t, `
actor Worker:
    func run() -> Int
    uses actors:
        let me: actor = core.self()
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Worker.run")
    return 0
`)
}

func TestActorDeclarationSpawnBuildAndRunBuiltinRuntime(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors_decl_spawn.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "actors_decl_spawn"+tgt.ExeExt)
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Runtime: RuntimeBuiltin}); err != nil {
		t.Fatalf("build: %v", err)
	}
	stdout, exitCode := runBinary(t, outPath)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}
