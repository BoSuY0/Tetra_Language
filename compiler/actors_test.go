package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/target"
)

func TestActorsPingPongBuildAndRun(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors_pingpong.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	if err := BuildFile(srcPath, outPath, tgt.Triple); err != nil {
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

func TestActorsPingPongBuildAndRunBuiltinRuntime(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors_pingpong.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
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

func TestActorsPingPongBuildAndRunSelfHostRuntime(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors_pingpong.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Runtime: RuntimeSelfHost}); err != nil {
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

func TestActorsTaggedStressBuildAndRunWithBothRuntimes(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors_tagged_stress.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	cases := []struct {
		name string
		rt   RuntimeMode
	}{
		{name: "selfhost", rt: RuntimeSelfHost},
		{name: "builtin", rt: RuntimeBuiltin},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			outPath := filepath.Join(tmp, "actors_tagged_stress"+tgt.ExeExt)
			if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Runtime: tc.rt}); err != nil {
				t.Fatalf("build: %v", err)
			}
			stdout, exitCode := runBinary(t, outPath)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != 0 {
				t.Fatalf("exit code mismatch: %d", exitCode)
			}
		})
	}
}

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
	`, "actor state field 'title' type 'str' is not supported in this MVP")
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
`, "actor state field 'raw' type 'ptr' is not supported in this MVP")
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
`, "initializer must be a compile-time constant i32/bool")
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

func TestActorsTypedMessagesRejectNonEnumSend(t *testing.T) {
	src := []byte(`
func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send_typed(peer, 1)

func worker() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected send_typed non-enum diagnostic")
	}
	if !strings.Contains(err.Error(), "send_typed expects an enum message") {
		t.Fatalf("error = %v", err)
	}
}

func TestActorsTypedMessagesRejectReferencePayload(t *testing.T) {
	src := []byte(`
enum BadMsg:
    case text(String)

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send_typed(peer, BadMsg.text("bad"))

func worker() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected typed actor payload diagnostic")
	}
	if !strings.Contains(err.Error(), "typed actor message payload must be value-only") {
		t.Fatalf("error = %v", err)
	}
}

func TestActorsTypedMessagesAllowIslandTransferCheckAndLower(t *testing.T) {
	src := []byte(`
enum MoveMsg:
    case take(island)

func main() -> Int
uses actors, alloc, islands, mem:
    let peer: actor = core.spawn("worker")
    unsafe:
        let isl: island = core.island_new(16)
        return core.send_typed(peer, MoveMsg.take(isl))

func worker() -> Int
uses actors:
    let msg: MoveMsg = core.recv_typed<MoveMsg>()
    match msg:
    case MoveMsg.take(isl):
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

func TestActorsTypedMessagesIslandTransferConsumesSource(t *testing.T) {
	src := []byte(`
enum MoveMsg:
    case take(island)

func main() -> Int
uses actors, alloc, islands, mem:
    let peer: actor = core.spawn("worker")
    unsafe:
        var isl: island = core.island_new(16)
        let _sent: Int = core.send_typed(peer, MoveMsg.take(isl))
        return core.send_typed(peer, MoveMsg.take(isl))

func worker() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected island transfer consume diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use consumed value 'isl'") {
		t.Fatalf("error = %v", err)
	}
}

func TestActorsTypedMessagesEnumConstructionConsumesIslandSource(t *testing.T) {
	src := []byte(`
enum MoveMsg:
    case take(island)

func main() -> Int
uses actors, alloc, islands, mem:
    let peer: actor = core.spawn("worker")
    unsafe:
        var isl: island = core.island_new(16)
        let msg: MoveMsg = MoveMsg.take(isl)
        let _sent: Int = core.send_typed(peer, msg)
        return core.send_typed(peer, MoveMsg.take(isl))

func worker() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected island construction consume diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use consumed value 'isl'") {
		t.Fatalf("error = %v", err)
	}
}

func TestActorsTypedMessagesStructConstructionConsumesIslandSource(t *testing.T) {
	src := []byte(`
struct MoveBox:
    token: island

enum MoveMsg:
    case box(MoveBox)

func main() -> Int
uses actors, alloc, islands, mem:
    let peer: actor = core.spawn("worker")
    unsafe:
        var isl: island = core.island_new(16)
        let box: MoveBox = MoveBox{token: isl}
        let _sent: Int = core.send_typed(peer, MoveMsg.box(box))
        return core.send_typed(peer, MoveMsg.box(MoveBox{token: isl}))

func worker() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected island struct construction consume diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use consumed value 'isl'") {
		t.Fatalf("error = %v", err)
	}
}

func TestActorsTypedPayloadBuildAndRunWithBothRuntimes(t *testing.T) {
	_, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	src := `
enum CounterMsg:
    case inc(Int, Int)
    case reset

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _sent: Int = core.send_typed(peer, CounterMsg.inc(20, 22))
    let reply: CounterMsg = core.recv_typed<CounterMsg>()
    match reply:
    case CounterMsg.inc(lhs, rhs):
        return lhs + rhs
    case CounterMsg.reset:
        return 0

func worker() -> Int
uses actors:
    let msg: CounterMsg = core.recv_typed<CounterMsg>()
    match msg:
    case CounterMsg.inc(lhs, rhs):
        let incSent: Int = core.send_typed(core.sender(), CounterMsg.inc(lhs, rhs))
        return 0
    case CounterMsg.reset:
        let resetSent: Int = core.send_typed(core.sender(), CounterMsg.reset)
        return 0
`
	for _, tc := range []struct {
		name string
		rt   RuntimeMode
	}{
		{name: "selfhost", rt: RuntimeSelfHost},
		{name: "builtin", rt: RuntimeBuiltin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: tc.rt})
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != 42 {
				t.Fatalf("exit code = %d, want 42", exitCode)
			}
		})
	}
}

func TestRuntimeSchedulerActorSleepDoesNotBlockSendWakeBuildAndRun(t *testing.T) {
	src := `
func slow() -> Int
uses actors, runtime:
    let _sleep: Int = core.sleep_ms(10)
    let _sent: Int = core.send(core.sender(), 1)
    return 0

func fast() -> Int
uses actors:
    let _sent: Int = core.send(core.sender(), 2)
    return 0

func main() -> Int
uses actors, runtime:
    let _slow: actor = core.spawn("slow")
    let _fast: actor = core.spawn("fast")
    let first: Int = core.recv()
    if first != 2:
        return 10 + first
    let second: Int = core.recv()
    if second != 1:
        return 20 + second
    let now: Int = core.time_now_ms()
    if now != 10:
        return 40 + now
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want actor sleep/send wake ordering", exitCode)
	}
}

func TestActorRecvUntilTimesOutWithNoMessagesBuildAndRun(t *testing.T) {
	src := `
func main() -> Int
uses actors, runtime:
    let result: actor.recv_result_i32 = core.recv_until(core.deadline_ms(4))
    if result.error != 2:
        return 20 + result.error
    if result.value != 0:
        return 40 + result.value
    return core.time_now_ms()
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 4 {
		t.Fatalf("exit code = %d, want recv_until timeout at logical time 4", exitCode)
	}
}

func TestActorRecvUntilReturnsMessageBeforeDeadlineBuildAndRun(t *testing.T) {
	src := `
func delayed() -> Int
uses actors, runtime:
    let _sleep: Int = core.sleep_ms(2)
    let _sent: Int = core.send(core.sender(), 7)
    return 0

func main() -> Int
uses actors, runtime:
    let _child: actor = core.spawn("delayed")
    let result: actor.recv_result_i32 = core.recv_until(core.deadline_ms(5))
    if result.error != 0:
        return 20 + result.error
    if result.value != 7:
        return 40 + result.value
    return core.time_now_ms()
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want recv_until message at logical time 2", exitCode)
	}
}

func TestActorRecvPollReturnsTimeoutThenMessageBuildAndRun(t *testing.T) {
	src := `
func delayed() -> Int
uses actors, runtime:
    let _sleep: Int = core.sleep_ms(2)
    let _sent: Int = core.send(core.sender(), 8)
    return 0

func main() -> Int
uses actors, runtime:
    let _child: actor = core.spawn("delayed")
    let early: actor.recv_result_i32 = core.recv_poll()
    if early.error != 2:
        return 20 + early.error
    if early.value != 0:
        return 40 + early.value
    let _sleep: Int = core.sleep_ms(3)
    let late: actor.recv_result_i32 = core.recv_poll()
    if late.error != 0:
        return 60 + late.error
    return late.value
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 8 {
		t.Fatalf("exit code = %d, want recv_poll message after timeout", exitCode)
	}
}

func TestActorRecvMsgUntilTimesOutAndReturnsTaggedMessageBuildAndRun(t *testing.T) {
	src := `
func tagged() -> Int
uses actors, runtime:
    let _sleep: Int = core.sleep_ms(2)
    let _sent: Int = core.send_msg(core.sender(), 11, 4)
    return 0

func main() -> Int
uses actors, runtime:
    let first: actor.recv_msg_result = core.recv_msg_until(core.deadline_ms(1))
    if first.error != 2:
        return 20 + first.error
    if first.value != 0:
        return 40 + first.value
    let _child: actor = core.spawn("tagged")
    let second: actor.recv_msg_result = core.recv_msg_until(core.deadline_ms(5))
    if second.error != 0:
        return 60 + second.error
    if second.value != 11:
        return 80 + second.value
    if second.tag != 4:
        return 100 + second.tag
    return core.time_now_ms()
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 3 {
		t.Fatalf("exit code = %d, want tagged message at logical time 3", exitCode)
	}
}

func TestActorsPingPongRuntimeModeParity(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors_pingpong.tetra")
	results := map[RuntimeMode]struct {
		stdout string
		exit   int
	}{}
	for _, rt := range []RuntimeMode{RuntimeBuiltin, RuntimeSelfHost} {
		tmp := t.TempDir()
		outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
		if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Runtime: rt}); err != nil {
			t.Fatalf("build runtime %d: %v", rt, err)
		}
		stdout, exitCode := runBinary(t, outPath)
		results[rt] = struct {
			stdout string
			exit   int
		}{stdout: stdout, exit: exitCode}
	}

	if results[RuntimeBuiltin] != results[RuntimeSelfHost] {
		t.Fatalf("runtime parity mismatch: builtin=%#v selfhost=%#v", results[RuntimeBuiltin], results[RuntimeSelfHost])
	}
}

func TestActorsPingPongBuildsSelfHostRuntimeForAllX64Targets(t *testing.T) {
	srcPath := filepath.Join("..", "examples", "actors_pingpong.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	tmp := t.TempDir()
	for _, triple := range []string{"linux-x64", "macos-x64", "windows-x64"} {
		t.Run(triple, func(t *testing.T) {
			tgt, err := target.Parse(triple)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			outPath := filepath.Join(tmp, "actors_"+strings.ReplaceAll(triple, "-", "_")+tgt.ExeExt)
			if _, err := BuildFileWithStatsOpt(srcPath, outPath, triple, BuildOptions{Runtime: RuntimeSelfHost}); err != nil {
				t.Fatalf("build: %v", err)
			}
			if _, err := os.Stat(outPath); err != nil {
				t.Fatalf("missing output: %v", err)
			}
		})
	}
}

func TestCanonicalSelfHostRuntimeSources(t *testing.T) {
	tests := []struct {
		path       string
		wantModule string
	}{
		{filepath.Join("..", "__rt", "actors_sysv.tetra"), "__rt.actors_sysv"},
		{filepath.Join("..", "__rt", "actors_win64.tetra"), "__rt.actors_win64"},
		{filepath.Join("selfhostrt", "actors_sysv.tetra"), "__rt.actors_sysv"},
		{filepath.Join("selfhostrt", "actors_win64.tetra"), "__rt.actors_win64"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			raw, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("read runtime source: %v", err)
			}
			file, err := frontend.ParseFile(raw, tt.path)
			if err != nil {
				t.Fatalf("parse runtime source: %v", err)
			}
			if file.Module != tt.wantModule {
				t.Fatalf("module = %q, want %q", file.Module, tt.wantModule)
			}
		})
	}
}

func TestSelfHostRuntimeObjectsExportRequiredSymbols(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		target string
	}{
		{"sysv-linux", filepath.Join("..", "__rt", "actors_sysv.tetra"), "linux-x64"},
		{"sysv-macos", filepath.Join("..", "__rt", "actors_sysv.tetra"), "macos-x64"},
		{"win64", filepath.Join("..", "__rt", "actors_win64.tetra"), "windows-x64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			objPath := filepath.Join(tmp, "runtime.tobj")
			if _, err := BuildFileWithStatsOpt(tt.src, objPath, tt.target, BuildOptions{Emit: EmitLibrary}); err != nil {
				t.Fatalf("build runtime object: %v", err)
			}
			obj, err := ReadObject(objPath)
			if err != nil {
				t.Fatalf("read runtime object: %v", err)
			}
			required := append(requiredActorRuntimeSymbols(), requiredTimeRuntimeSymbols()...)
			required = append(required, requiredActorStateRuntimeSymbols()...)
			required = append(required, requiredTypedTaskRuntimeSymbols(8)...)
			assertObjectHasSymbols(t, obj, required...)
		})
	}
}

func TestRequiredTimeRuntimeSymbols(t *testing.T) {
	got := map[string]struct{}{}
	for _, name := range requiredTimeRuntimeSymbols() {
		got[name] = struct{}{}
	}

	for _, name := range []string{
		"__tetra_time_now_ms",
		"__tetra_sleep_ms",
		"__tetra_sleep_until_ms",
		"__tetra_deadline_ms",
		"__tetra_timer_ready_ms",
	} {
		if _, ok := got[name]; !ok {
			t.Fatalf("required time runtime symbols missing %q", name)
		}
	}
}

func TestRequiredActorRuntimeSymbolsIncludeTaggedMessageABI(t *testing.T) {
	got := map[string]struct{}{}
	for _, name := range requiredActorRuntimeSymbols() {
		got[name] = struct{}{}
	}

	for _, name := range []string{
		"__tetra_actor_send_msg",
		"__tetra_actor_recv_msg",
		"__tetra_actor_recv_poll",
		"__tetra_actor_send_begin",
		"__tetra_actor_send_slot",
		"__tetra_actor_send_commit",
		"__tetra_actor_recv_msg_until",
		"__tetra_actor_recv_begin",
		"__tetra_actor_recv_slot",
		"__tetra_actor_recv_count",
		"__tetra_actor_yield_now",
	} {
		if _, ok := got[name]; !ok {
			t.Fatalf("required actor runtime symbols missing tagged message ABI symbol %q", name)
		}
	}
}

func TestRequiredActorStateRuntimeSymbols(t *testing.T) {
	got := map[string]struct{}{}
	for _, name := range requiredActorStateRuntimeSymbols() {
		got[name] = struct{}{}
	}
	for _, name := range []string{
		"__tetra_actor_state_load",
		"__tetra_actor_state_store",
	} {
		if _, ok := got[name]; !ok {
			t.Fatalf("required actor-state runtime symbols missing %q", name)
		}
	}
}

func TestActorGlueExportsProgramRuntimeSymbols(t *testing.T) {
	dispatchFn, err := buildActorDispatchFunc([]string{"main", "pong"}, nil)
	if err != nil {
		t.Fatalf("build dispatch: %v", err)
	}
	mainIDFn, err := buildActorMainEntryIDFunc("main")
	if err != nil {
		t.Fatalf("build main entry id: %v", err)
	}
	obj, err := CodegenObjectLinuxX64([]IRFunc{dispatchFn, mainIDFn})
	if err != nil {
		t.Fatalf("codegen glue object: %v", err)
	}
	assertObjectHasSymbols(t, obj, "__tetra_actor_dispatch", "__tetra_actor_main_entry_id")
}

func assertObjectHasSymbols(t *testing.T, obj *Object, names ...string) {
	t.Helper()
	symbols := make(map[string]struct{}, len(obj.Symbols))
	for _, sym := range obj.Symbols {
		symbols[sym.Name] = struct{}{}
	}
	for _, name := range names {
		if _, ok := symbols[name]; !ok {
			t.Fatalf("missing symbol %q", name)
		}
	}
}
