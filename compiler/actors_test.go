package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/semantics"
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

func TestActorRuntimeBuiltinCapacityLimitReturnsNoExtraActor(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func worker() -> Int
uses actors:
    let _sent: Int = core.send(core.sender(), 1)
    return 0

func main() -> Int
uses actors, runtime:
    var spawned: Int = 0
    while spawned < 128:
        let _peer: actor = core.spawn("worker")
        spawned = spawned + 1

    var received: Int = 0
    while received < 128:
        let msg: actor.recv_result_i32 = core.recv_until(core.deadline_ms(1))
        if msg.error == 2:
            return received
        if msg.error != 0:
            return 200 + msg.error
        received = received + msg.value
    return 250
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 127 {
		t.Fatalf("exit code = %d, want 127 successful child actors before builtin capacity failure", exitCode)
	}
}

func TestActorStateSlotLimitRejectsMoreThanEightBeforeRuntime(t *testing.T) {
	requireCheckFileErrorContains(t, `
actor Slots:
    var s0: Int = 1
    var s1: Int = 2
    var s2: Int = 3
    var s3: Int = 4
    var s4: Int = 5
    var s5: Int = 6
    var s6: Int = 7
    var s7: Int = 8
    var s8: Int = 9
    func run() -> Int
    uses actors:
        return s8

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Slots.run")
    return 0
`, "actor 'Slots' state supports at most 8 slots, got 9")

	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "actor_state_too_many_slots.tetra")
	if err := os.WriteFile(srcPath, []byte(`
actor Slots:
    var s0: Int = 1
    var s1: Int = 2
    var s2: Int = 3
    var s3: Int = 4
    var s4: Int = 5
    var s5: Int = 6
    var s6: Int = 7
    var s7: Int = 8
    var s8: Int = 9
    func run() -> Int
    uses actors:
        return s8

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Slots.run")
    return 0
`), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "actor_state_too_many_slots"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Runtime: RuntimeBuiltin})
	if err == nil {
		t.Fatalf("expected build to fail before runtime for actor state slot limit")
	}
	if !strings.Contains(err.Error(), "actor 'Slots' state supports at most 8 slots, got 9") {
		t.Fatalf("error = %v", err)
	}
	if _, statErr := os.Stat(outPath); statErr == nil {
		t.Fatalf("unexpected output binary after semantic actor-state slot failure: %s", outPath)
	} else if !os.IsNotExist(statErr) {
		t.Fatalf("stat output: %v", statErr)
	}
}

func TestActorStateSlotLimitAllowsEightSlots(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
actor Slots:
    var s0: Int = 1
    var s1: Int = 2
    var s2: Int = 3
    var s3: Int = 4
    var s4: Int = 5
    var s5: Int = 6
    var s6: Int = 7
    var s7: Int = 8
    func run() -> Int
    uses actors:
        let total: Int = s0 + s1 + s2 + s3 + s4 + s5 + s6 + s7
        let _sent: Int = core.send(core.sender(), total)
        return 0

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Slots.run")
    return core.recv()
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 36 {
		t.Fatalf("exit code = %d, want eight actor state slots to sum to 36", exitCode)
	}
}

func TestActorMessagePoolBudgetAtDocumentedCapacityBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
val MESSAGE_POOL_SAFE_MESSAGES: i32 = 744
val MAILBOX_CAPACITY: i32 = 256

func main() -> Int
uses actors:
    let me: actor = core.self()
    var sent: Int = 0
    var drift: Int = 0
    while sent < MESSAGE_POOL_SAFE_MESSAGES:
        var batch: Int = 0
        while batch < MAILBOX_CAPACITY && sent < MESSAGE_POOL_SAFE_MESSAGES:
            let ack: Int = core.send(me, sent)
            if ack != sent:
                return 10
            sent = sent + 1
            batch = batch + 1

        var received: Int = 0
        while received < batch:
            let msg: Int = core.recv()
            drift = drift + (msg - ((sent - batch) + received))
            received = received + 1

    if drift != 0:
        return 31
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want documented message pool budget smoke to complete", exitCode)
	}
}

func TestActorMessagePoolExhaustionReturnsCheckedFailure(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
val MESSAGE_POOL_SAFE_MESSAGES: i32 = 744
val MAILBOX_CAPACITY: i32 = 256

func main() -> Int
uses actors:
    let me: actor = core.self()
    var sent: Int = 0
    var drift: Int = 0
    while sent < MESSAGE_POOL_SAFE_MESSAGES:
        var batch: Int = 0
        while batch < MAILBOX_CAPACITY && sent < MESSAGE_POOL_SAFE_MESSAGES:
            let ack: Int = core.send(me, sent)
            if ack != sent:
                return 10
            sent = sent + 1
            batch = batch + 1

        var received: Int = 0
        while received < batch:
            let msg: actor.recv_result_i32 = core.recv_poll()
            if msg.error != 0:
                return 30 + msg.error
            drift = drift + (msg.value - ((sent - batch) + received))
            received = received + 1

    let overflow: Int = core.send(me, 123)
    if overflow != -1:
        return 20

    let empty: actor.recv_result_i32 = core.recv_poll()
    if empty.error != 2:
        return 40 + empty.error
    if drift != 0:
        return 50
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want checked message pool exhaustion without corrupting mailbox", exitCode)
	}
}

func TestActorMessagePoolExhaustionCoversTaggedMessages(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
val MESSAGE_POOL_SAFE_MESSAGES: i32 = 744
val MAILBOX_CAPACITY: i32 = 256

func main() -> Int
uses actors, runtime:
    let me: actor = core.self()
    var sent: Int = 0
    var drift: Int = 0
    while sent < MESSAGE_POOL_SAFE_MESSAGES:
        var batch: Int = 0
        while batch < MAILBOX_CAPACITY && sent < MESSAGE_POOL_SAFE_MESSAGES:
            let ack: Int = core.send_msg(me, sent, 7)
            if ack != sent:
                return 10
            sent = sent + 1
            batch = batch + 1

        var received: Int = 0
        while received < batch:
            let msg: actor.recv_msg_result = core.recv_msg_until(core.deadline_ms(1))
            if msg.error != 0:
                return 30 + msg.error
            if msg.tag != 7:
                return 40
            drift = drift + (msg.value - ((sent - batch) + received))
            received = received + 1

    let overflow: Int = core.send_msg(me, 123, 8)
    if overflow != -1:
        return 20

    let empty: actor.recv_msg_result = core.recv_msg_until(core.deadline_ms(1))
    if empty.error != 2:
        return 50 + empty.error
    if drift != 0:
        return 60
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want checked tagged message pool exhaustion", exitCode)
	}
}

func TestActorMessagePoolExhaustionCoversTypedMessages(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum Telemetry:
    case item(Int)

val MESSAGE_POOL_SAFE_MESSAGES: i32 = 744
val MAILBOX_CAPACITY: i32 = 256

func main() -> Int
uses actors:
    let me: actor = core.self()
    var sent: Int = 0
    var drift: Int = 0
    while sent < MESSAGE_POOL_SAFE_MESSAGES:
        var batch: Int = 0
        while batch < MAILBOX_CAPACITY && sent < MESSAGE_POOL_SAFE_MESSAGES:
            let ack: Int = core.send_typed(me, Telemetry.item(sent))
            if ack != 0:
                return 10
            sent = sent + 1
            batch = batch + 1

        var received: Int = 0
        while received < batch:
            let msg: Telemetry = core.recv_typed<Telemetry>()
            match msg:
            case Telemetry.item(value):
                drift = drift + (value - ((sent - batch) + received))
            received = received + 1

    let overflow: Int = core.send_typed(me, Telemetry.item(123))
    if overflow != -1:
        return 20

    if drift != 0:
        return 30
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want checked typed message pool exhaustion", exitCode)
	}
}

func TestActorMailboxFullReturnsCheckedBackpressure(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
val MAILBOX_CAPACITY: i32 = 256

func sleeper() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("sleeper")
    var sent: Int = 0
    while sent < MAILBOX_CAPACITY:
        let ack: Int = core.send(peer, sent)
        if ack != sent:
            return 10
        sent = sent + 1

    let full: Int = core.send(peer, 777)
    if full != -2:
        return 20
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want checked mailbox backpressure", exitCode)
	}
}

func TestActorInvalidHandleSendReturnsCheckedFailure(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum Signal:
    case item(Int)

func sleeper() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 0

func main() -> Int
uses actors:
    var spawned: Int = 0
    while spawned < 127:
        let _peer: actor = core.spawn("sleeper")
        spawned = spawned + 1

    let failed: actor = core.spawn("sleeper")
    let sent: Int = core.send(failed, 1)
    if sent != -3:
        return 20
    let tagged: Int = core.send_msg(failed, 2, 7)
    if tagged != -3:
        return 30
    let typed: Int = core.send_typed(failed, Signal.item(3))
    if typed != -3:
        return 40
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want checked invalid actor handle send-path failures", exitCode)
	}
}

func TestActorSendToDoneActorReturnsCheckedFailure(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum Signal:
    case item(Int)

func done() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let peer: actor = core.spawn("done")
    let _sleep: Int = core.sleep_ms(1)
    let sent: Int = core.send(peer, 1)
    if sent != -4:
        return 20
    let tagged: Int = core.send_msg(peer, 2, 7)
    if tagged != -4:
        return 30
    let typed: Int = core.send_typed(peer, Signal.item(3))
    if typed != -4:
        return 40
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want checked done actor send-path failures", exitCode)
	}
}

func TestActorRuntimeCapacityLimitsDocumented(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "docs", "spec", "actors.md"))
	if err != nil {
		t.Fatalf("read actors spec: %v", err)
	}
	doc := string(raw)
	for _, want := range []string{
		"## Runtime Capacity Limits",
		"`maxActors = 128`",
		"127 child actors",
		"`maxActorMailboxMsgs = 256`",
		"full mailbox",
		"`-2`",
		"64 KiB",
		"744",
		"single-slot",
		"checked failure",
		"`-1`",
		"does not enqueue an overflow message",
		"reclaim message-pool capacity",
		"invalid actor handle",
		"`-3`",
		"done actor",
		"`-4`",
		"8 state slots",
		"rejects programs that require more than 8 actor-state slots before lowering",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("actors spec missing capacity-limit text %q", want)
		}
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
	if !strings.Contains(err.Error(), "actor state field 'title' type 'str' is not supported in this MVP") {
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
	if !strings.Contains(err.Error(), "cannot send borrowed view across actor boundary; use .copy()") {
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

func TestActorsTypedMessagesOwnedRegionSliceMoveBuildAndRun(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum MoveMsg:
    case region(island, []i32)

enum Reply:
    case value(Int)

func main() -> Int
uses actors, alloc, islands, mem:
    unsafe:
        var region: island = core.island_new(128)
        var xs: []i32 = core.island_make_i32(region, 2)
        xs[0] = 20
        xs[1] = 22
        let _sent: Int = core.send_typed(core.self(), MoveMsg.region(region, xs))
        let msg: MoveMsg = core.recv_typed<MoveMsg>()
        match msg:
        case MoveMsg.region(moved_region, moved_xs):
            let sum: Int = moved_xs[0] + moved_xs[1]
            free(moved_region)
            return sum
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want 42", exitCode)
	}
	_ = tgt
}

func TestActorsTypedMessagesOwnedRegionSliceMoveConsumesSenderSlice(t *testing.T) {
	src := []byte(`
enum MoveMsg:
    case region(island, []i32)

func main() -> Int
uses actors, alloc, islands, mem:
    unsafe:
        var region: island = core.island_new(128)
        var xs: []i32 = core.island_make_i32(region, 2)
        let _sent: Int = core.send_typed(core.self(), MoveMsg.region(region, xs))
        return xs[0]

func worker() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected region-backed slice move consume diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use consumed value 'xs'") {
		t.Fatalf("error = %v", err)
	}
}

func TestActorsTypedMessagesOwnedRegionSliceMoveExplainReport(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "actor_region_slice_move.tetra")
	outPath := filepath.Join(tmp, "actor_region_slice_move")
	if err := os.WriteFile(srcPath, []byte(`
enum MoveMsg:
    case region(island, []i32)

func main() -> Int
uses actors, alloc, islands, mem:
    unsafe:
        var region: island = core.island_new(128)
        var xs: []i32 = core.island_make_i32(region, 2)
        return core.send_typed(core.self(), MoveMsg.region(region, xs))
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{
		Runtime: RuntimeBuiltin,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}
	raw, err := os.ReadFile(outPath + ".actor-transfer.json")
	if err != nil {
		t.Fatalf("read actor transfer report: %v", err)
	}
	for _, want := range []string{
		`"kind": "actor_transfer"`,
		`"transfer_mode": "zero_copy_move"`,
		`"runtime_path": "actor_mailbox_zero_copy_region_slot"`,
		`"payload_type": "[]i32"`,
		`"owner": "region"`,
		`"bytes_copied": 0`,
		`"zero_copy": true`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("actor transfer report missing %s:\n%s", want, raw)
		}
	}
	var report struct {
		Sends []struct {
			PayloadType                string `json:"payload_type"`
			TransferMode               string `json:"transfer_mode"`
			RuntimePath                string `json:"runtime_path"`
			ClaimLevel                 string `json:"claim_level"`
			ProductionRuntimeValidated bool   `json:"production_runtime_validated"`
		} `json:"sends"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode actor transfer report: %v\n%s", err, raw)
	}
	var sawZeroCopyMove bool
	for _, row := range report.Sends {
		if row.TransferMode != "zero_copy_move" {
			continue
		}
		sawZeroCopyMove = true
		if row.PayloadType != "[]i32" || row.RuntimePath != "actor_mailbox_zero_copy_region_slot" {
			t.Fatalf("zero-copy row = %+v, want owned region-backed slice runtime path", row)
		}
		if row.ClaimLevel != "evidence_only" {
			t.Fatalf("zero-copy row claim_level = %q, want evidence_only: %+v", row.ClaimLevel, row)
		}
		if row.ProductionRuntimeValidated {
			t.Fatalf("zero-copy row must not claim production runtime validation: %+v", row)
		}
	}
	if !sawZeroCopyMove {
		t.Fatalf("actor transfer report missing zero_copy_move row: %+v", report.Sends)
	}
}

func TestActorsTypedMailboxExplainReportIncludesMetadataAndCopyMove(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "typed_mailbox_report.tetra")
	outPath := filepath.Join(tmp, "typed_mailbox_report")
	if err := os.WriteFile(srcPath, []byte(`
enum Telemetry:
    case inc(Int, Bool)
    case move(island)

func main() -> Int
uses actors, alloc, islands, mem:
    unsafe:
        var region: island = core.island_new(32)
        let _copy: Int = core.send_typed(core.self(), Telemetry.inc(7, true))
        return core.send_typed(core.self(), Telemetry.move(region))
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{
		Runtime: RuntimeBuiltin,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}
	raw, err := os.ReadFile(outPath + ".actor-transfer.json")
	if err != nil {
		t.Fatalf("read actor transfer report: %v", err)
	}
	for _, want := range []string{
		`"mailboxes"`,
		`"message_schema": "Telemetry"`,
		`"capacity": 744`,
		`"backpressure": "blocking_recv_yield"`,
		`"transfer_mode": "copy"`,
		`"ownership": "copy"`,
		`"runtime_path": "actor_mailbox_value_slot"`,
		`"payload_type": "i32"`,
		`"payload_type": "bool"`,
		`"transfer_mode": "move"`,
		`"ownership": "owned_region"`,
		`"runtime_path": "actor_mailbox_resource_slot"`,
		`"owner": "region"`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("typed mailbox report missing %s:\n%s", want, raw)
		}
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

func TestActorSpawnsTaskAndReceivesCompletionBuildAndRun(t *testing.T) {
	src := `
func task_worker() -> Int:
    return 5

func actor_worker() -> Int
uses actors, runtime:
    let task: task.i32 = core.task_spawn_i32("task_worker")
    let taskResult: task.result_i32 = core.task_join_result_i32(task)
    if taskResult.error != 0:
        return 40 + taskResult.error
    let _sent: Int = core.send(core.sender(), taskResult.value + 1)
    return 0

func main() -> Int
uses actors, runtime:
    let _actor: actor = core.spawn("actor_worker")
    let reply: actor.recv_result_i32 = core.recv_until(core.deadline_ms(5))
    if reply.error != 0:
        return 60 + reply.error
    return reply.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; actor-spawned task should wake actor receive")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 6 {
		t.Fatalf("exit code = %d, want actor/task interaction result 6", exitCode)
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

func TestActorsPingPongBuildsSelfHostRuntimeForX32(t *testing.T) {
	srcPath := filepath.Join("..", "examples", "actors_pingpong.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "actors_x32")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Runtime: RuntimeSelfHost, Jobs: 1}); err != nil {
		t.Fatalf("build x32 self-host runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read x32 executable: %v", err)
	}
	if len(data) < 20 {
		t.Fatalf("x32 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		t.Fatalf("x32 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 1 {
		t.Fatalf("x32 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x3e {
		t.Fatalf("x32 executable machine = %#x, want EM_X86_64", got)
	}
}

func TestX32MultiSpawnActorRuntimeBuildsSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "actors_x32_multi_spawn.tetra")
	outPath := filepath.Join(tmp, "actors-x32-multi-spawn")
	if err := os.WriteFile(srcPath, []byte(`
func slow() -> Int
uses actors, runtime:
    let _sleep: Int = core.sleep_ms(10)
    let _sent: Int = core.send(core.sender(), 1)
    return 0

func fast() -> Int
uses actors, runtime:
    let _sleep: Int = core.sleep_ms(2)
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
    if core.time_now_ms() != 10:
        return 40 + core.time_now_ms()
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x32 two-spawn actor self-host runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read x32 executable: %v", err)
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		t.Fatalf("x32 executable missing ELF magic or too small: len=%d", len(data))
	}
	if data[4] != 1 {
		t.Fatalf("x32 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x3e {
		t.Fatalf("x32 executable machine = %#x, want EM_X86_64", got)
	}
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 0 {
		t.Fatalf("x32 two-spawn actor runtime exit=%d stdout=%q, want 0", code, stdout)
	}
}

func TestX86SingleActorRuntimeBuildsAndRunsWhenHostSupportsI386(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "actors_x86.tetra")
	outPath := filepath.Join(tmp, "actors-x86")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int
uses actors:
    let value: Int = core.recv()
    if value == 41:
        let _sent: Int = core.send(core.sender(), 42)
        return 0
    return 1

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _sent: Int = core.send(peer, 41)
    let reply: Int = core.recv()
    if reply == 42:
        return 0
    return reply
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 single-actor auto self-host runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read x86 executable: %v", err)
	}
	if len(data) < 20 {
		t.Fatalf("x86 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		t.Fatalf("x86 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 1 {
		t.Fatalf("x86 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x03 {
		t.Fatalf("x86 executable machine = %#x, want EM_386", got)
	}
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 0 {
		t.Fatalf("x86 single-actor runtime exit=%d stdout=%q, want 0", code, stdout)
	}
}

func TestX86MultiSpawnActorsRuntimeBuildsSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "actors_multi_spawn_x86.tetra")
	outPath := filepath.Join(tmp, "actors-multi-spawn-x86")
	if err := os.WriteFile(srcPath, []byte(`
func slow() -> Int
uses actors, runtime:
    let _sleep: Int = core.sleep_ms(10)
    let _sent: Int = core.send(core.sender(), 1)
    return 0

func fast() -> Int
uses actors, runtime:
    let _sleep: Int = core.sleep_ms(2)
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
    if core.time_now_ms() != 10:
        return 40 + core.time_now_ms()
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 two-spawn actor self-host runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read x86 executable: %v", err)
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		t.Fatalf("x86 executable missing ELF magic or too small: len=%d", len(data))
	}
	if data[4] != 1 {
		t.Fatalf("x86 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x03 {
		t.Fatalf("x86 executable machine = %#x, want EM_386", got)
	}
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 0 {
		t.Fatalf("x86 two-spawn actor runtime exit=%d stdout=%q, want 0", code, stdout)
	}
}

func TestX86ActorStateRuntimeBuildsAndRunsWhenHostSupportsI386(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "actor_state_x86.tetra")
	outPath := filepath.Join(tmp, "actor-state-x86")
	if err := os.WriteFile(srcPath, []byte(`
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
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 actor-state auto self-host runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read x86 executable: %v", err)
	}
	if len(data) < 20 {
		t.Fatalf("x86 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		t.Fatalf("x86 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 1 {
		t.Fatalf("x86 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x03 {
		t.Fatalf("x86 executable machine = %#x, want EM_386", got)
	}
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 42 {
		t.Fatalf("x86 actor-state runtime exit=%d stdout=%q, want 42", code, stdout)
	}
}

func TestCanonicalSelfHostRuntimeSources(t *testing.T) {
	tests := []struct {
		path       string
		wantModule string
	}{
		{filepath.Join("..", "__rt", "actors_sysv.tetra"), "__rt.actors_sysv"},
		{filepath.Join("..", "__rt", "actors_i386.tetra"), "__rt.actors_i386"},
		{filepath.Join("..", "__rt", "actors_win64.tetra"), "__rt.actors_win64"},
		{filepath.Join("selfhostrt", "actors_sysv.tetra"), "__rt.actors_sysv"},
		{filepath.Join("selfhostrt", "actors_i386.tetra"), "__rt.actors_i386"},
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
		{"sysv-linux-x32", filepath.Join("..", "__rt", "actors_sysv.tetra"), "linux-x32"},
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
		"__tetra_actor_recv_until",
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

func TestActorDispatchStateInitializationMatchesRuntimeStoreABI(t *testing.T) {
	checked := &semantics.CheckedProgram{
		Funcs: []semantics.CheckedFunc{
			{
				Name: "Counter.run",
				ActorState: map[string]semantics.ActorStateField{
					"count": {Name: "count", Slot: 0, TypeName: "Int", Mutable: true, Init: 7},
				},
			},
		},
	}
	dispatchFn, err := buildActorDispatchFunc([]string{"Counter.run"}, checked)
	if err != nil {
		t.Fatalf("build dispatch: %v", err)
	}
	if err := lower.VerifyFunc(dispatchFn); err != nil {
		t.Fatalf("dispatch verifier: %v", err)
	}

	for _, instr := range dispatchFn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == "__tetra_actor_state_store" {
			if instr.ArgSlots != 2 || instr.RetSlots != 1 {
				t.Fatalf("state store ABI = args %d rets %d, want args 2 rets 1", instr.ArgSlots, instr.RetSlots)
			}
			return
		}
	}
	t.Fatalf("dispatch missing __tetra_actor_state_store call: %#v", dispatchFn.Instrs)
}

func TestGeneratedActorGlueIsVerifiedBeforeNativeCodegen(t *testing.T) {
	checked := &semantics.CheckedProgram{
		Funcs: []semantics.CheckedFunc{
			{
				Name: "stateful",
				ActorState: map[string]semantics.ActorStateField{
					"count": {Name: "count", Slot: 0, TypeName: "Int", Mutable: true, Init: 1},
				},
			},
		},
	}

	codegenCalled := false
	native := nativeBuildTarget{
		triple: "linux-x64",
		backend: nativeExecutableBackend{
			actorRuntime: func(actorEntries []string) (*Object, error) {
				symbolNames := append([]string{}, requiredActorRuntimeSymbols()...)
				symbolNames = append(symbolNames, requiredActorStateRuntimeSymbols()...)
				symbolNames = append(symbolNames, "__tetra_actor_main_entry_id")
				symbols := make([]Symbol, 0, len(symbolNames))
				for _, name := range symbolNames {
					symbols = append(symbols, Symbol{Name: name})
				}
				return &Object{Symbols: symbols}, nil
			},
			link: func(outputPath string, objects []*Object, mainName string) error {
				return nil
			},
		},
		codegen: func(funcs []IRFunc, dataPrefix [][]byte) (*Object, error) {
			for _, fn := range funcs {
				if fn.Name == "__tetra_actor_dispatch" {
					codegenCalled = true
				}
			}
			return &Object{}, nil
		},
	}

	err := linkNativeExecutable(filepath.Join(t.TempDir(), "out"), native, BuildOptions{}, checked, nil, nil)
	if err == nil || !strings.Contains(err.Error(), `call is missing target name`) {
		t.Fatalf("linkNativeExecutable error = %v, want generated IR verifier error", err)
	}
	if codegenCalled {
		t.Fatalf("generated actor glue reached native codegen before verifier")
	}
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
