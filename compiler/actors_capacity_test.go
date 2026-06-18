package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/target"
)

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
val MAILBOX_CAPACITY: i32 = 256
val THIRD_ACTOR_LIVE_MESSAGES: i32 = 232

func sleeper() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 0

func main() -> Int
uses actors:
    let peer_a: actor = core.spawn("sleeper")
    let peer_b: actor = core.spawn("sleeper")
    let peer_c: actor = core.spawn("sleeper")

    var sent_a: Int = 0
    while sent_a < MAILBOX_CAPACITY:
        let ack_a: Int = core.send(peer_a, sent_a)
        if ack_a != sent_a:
            return 10
        sent_a = sent_a + 1

    var sent_b: Int = 0
    while sent_b < MAILBOX_CAPACITY:
        let ack_b: Int = core.send(peer_b, sent_b)
        if ack_b != sent_b:
            return 20
        sent_b = sent_b + 1

    var sent_c: Int = 0
    while sent_c < THIRD_ACTOR_LIVE_MESSAGES:
        let ack_c: Int = core.send(peer_c, sent_c)
        if ack_c != sent_c:
            return 30
        sent_c = sent_c + 1

    let overflow: Int = core.send(peer_c, 123)
    if overflow != -1:
        return 40
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

func TestActorMessagePoolReclaimsDrainedMessagesBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
val TOTAL_MESSAGES: i32 = 1000
val MAILBOX_CAPACITY: i32 = 256

func main() -> Int
uses actors:
    let me: actor = core.self()
    var sent: Int = 0
    var drift: Int = 0
    while sent < TOTAL_MESSAGES:
        var batch: Int = 0
        while batch < MAILBOX_CAPACITY && sent < TOTAL_MESSAGES:
            let ack: Int = core.send(me, sent)
            if ack == -1:
                return 21
            if ack == -2:
                return 22
            if ack != sent:
                return 23
            sent = sent + 1
            batch = batch + 1

        var received: Int = 0
        while received < batch:
            let msg: actor.recv_result_i32 = core.recv_poll()
            if msg.error != 0:
                return 30 + msg.error
            drift = drift + (msg.value - ((sent - batch) + received))
            received = received + 1

    if drift != 0:
        return 40
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want drained message nodes to be reusable beyond the old bump-only pool budget", exitCode)
	}
}

func TestActorTypedMessagePoolReclaimsDrainedMessagesBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum Telemetry:
    case sample(Int, Int, Int)
    case reset

val TOTAL_MESSAGES: i32 = 1000
val MAILBOX_CAPACITY: i32 = 256

func main() -> Int
uses actors:
    let me: actor = core.self()
    var sent: Int = 0
    var drift: Int = 0
    while sent < TOTAL_MESSAGES:
        var batch: Int = 0
        while batch < MAILBOX_CAPACITY && sent < TOTAL_MESSAGES:
            let ack: Int = core.send_typed(me, Telemetry.sample(sent, sent + 1, sent + 2))
            if ack == -1:
                return 21
            if ack == -2:
                return 22
            if ack != 0:
                return 23
            sent = sent + 1
            batch = batch + 1

        var received: Int = 0
        while received < batch:
            let msg: Telemetry = core.recv_typed<Telemetry>()
            match msg:
            case Telemetry.sample(a, b, c):
                let expected: Int = (sent - batch) + received
                drift = drift + (a - expected)
                drift = drift + (b - (expected + 1))
                drift = drift + (c - (expected + 2))
            case Telemetry.reset:
                return 30
            received = received + 1

    if drift != 0:
        return 40
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want drained typed message nodes and payload slots to be reusable beyond the old bump-only pool budget", exitCode)
	}
}

func TestActorMessagePoolExhaustionCoversTaggedMessages(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
val MAILBOX_CAPACITY: i32 = 256
val THIRD_ACTOR_LIVE_MESSAGES: i32 = 232

func sleeper() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 0

func main() -> Int
uses actors:
    let peer_a: actor = core.spawn("sleeper")
    let peer_b: actor = core.spawn("sleeper")
    let peer_c: actor = core.spawn("sleeper")

    var sent_a: Int = 0
    while sent_a < MAILBOX_CAPACITY:
        let ack_a: Int = core.send_msg(peer_a, sent_a, 7)
        if ack_a != sent_a:
            return 10
        sent_a = sent_a + 1

    var sent_b: Int = 0
    while sent_b < MAILBOX_CAPACITY:
        let ack_b: Int = core.send_msg(peer_b, sent_b, 7)
        if ack_b != sent_b:
            return 20
        sent_b = sent_b + 1

    var sent_c: Int = 0
    while sent_c < THIRD_ACTOR_LIVE_MESSAGES:
        let ack_c: Int = core.send_msg(peer_c, sent_c, 7)
        if ack_c != sent_c:
            return 30
        sent_c = sent_c + 1

    let overflow: Int = core.send_msg(peer_c, 123, 8)
    if overflow != -1:
        return 40
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

val MAILBOX_CAPACITY: i32 = 256
val THIRD_ACTOR_LIVE_MESSAGES: i32 = 232

func sleeper() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 0

func main() -> Int
uses actors:
    let peer_a: actor = core.spawn("sleeper")
    let peer_b: actor = core.spawn("sleeper")
    let peer_c: actor = core.spawn("sleeper")

    var sent_a: Int = 0
    while sent_a < MAILBOX_CAPACITY:
        let ack_a: Int = core.send_typed(peer_a, Telemetry.item(sent_a))
        if ack_a != 0:
            return 10
        sent_a = sent_a + 1

    var sent_b: Int = 0
    while sent_b < MAILBOX_CAPACITY:
        let ack_b: Int = core.send_typed(peer_b, Telemetry.item(sent_b))
        if ack_b != 0:
            return 20
        sent_b = sent_b + 1

    var sent_c: Int = 0
    while sent_c < THIRD_ACTOR_LIVE_MESSAGES:
        let ack_c: Int = core.send_typed(peer_c, Telemetry.item(sent_c))
        if ack_c != 0:
            return 30
        sent_c = sent_c + 1

    let overflow: Int = core.send_typed(peer_c, Telemetry.item(123))
    if overflow != -1:
        return 40
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

func TestActorMailboxBackpressureRecoversAfterSelfDrainBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
val MAILBOX_CAPACITY: i32 = 256

func main() -> Int
uses actors:
    let me: actor = core.self()
    var sent: Int = 0
    while sent < MAILBOX_CAPACITY:
        let ack: Int = core.send(me, sent)
        if ack != sent:
            return 10
        sent = sent + 1

    let full: Int = core.send(me, 900)
    if full != -2:
        return 20

    var received: Int = 0
    var drift: Int = 0
    while received < MAILBOX_CAPACITY:
        let msg: actor.recv_result_i32 = core.recv_poll()
        if msg.error != 0:
            return 30 + msg.error
        if msg.value == 900:
            return 40
        drift = drift + (msg.value - received)
        received = received + 1

    if drift != 0:
        return 50

    let retry: Int = core.send(me, 777)
    if retry != 777:
        return 60
    let after: actor.recv_result_i32 = core.recv_poll()
    if after.error != 0:
        return 70 + after.error
    if after.value != 777:
        return 80
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want i32 mailbox backpressure to recover after drain", exitCode)
	}
}

func TestActorTaggedMailboxBackpressureRecoversAfterSelfDrainBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
val MAILBOX_CAPACITY: i32 = 256

func main() -> Int
uses actors:
    let me: actor = core.self()
    var sent: Int = 0
    while sent < MAILBOX_CAPACITY:
        let ack: Int = core.send_msg(me, sent, 7)
        if ack != sent:
            return 10
        sent = sent + 1

    let full: Int = core.send_msg(me, 900, 99)
    if full != -2:
        return 20

    var received: Int = 0
    var drift: Int = 0
    while received < MAILBOX_CAPACITY:
        let msg: actor.msg = core.recv_msg()
        if msg.value == 900:
            return 30
        if msg.tag == 99:
            return 40
        if msg.tag != 7:
            return 50 + msg.tag
        drift = drift + (msg.value - received)
        received = received + 1

    if drift != 0:
        return 60

    let retry: Int = core.send_msg(me, 777, 9)
    if retry != 777:
        return 70
    let after: actor.msg = core.recv_msg()
    if after.value != 777:
        return 80
    if after.tag != 9:
        return 90 + after.tag
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want tagged mailbox backpressure to recover after drain without enqueued overflow message", exitCode)
	}
}

func TestActorTypedMailboxBackpressureRecoversWithoutPartialPayloadBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum Telemetry:
    case sample(Int, Int, Int)
    case poison(Int, Int, Int)

val MAILBOX_CAPACITY: i32 = 256

func main() -> Int
uses actors:
    let me: actor = core.self()
    var sent: Int = 0
    while sent < MAILBOX_CAPACITY:
        let ack: Int = core.send_typed(me, Telemetry.sample(sent, sent + 1, sent + 2))
        if ack != 0:
            return 10
        sent = sent + 1

    let full: Int = core.send_typed(me, Telemetry.poison(900, 901, 902))
    if full != -2:
        return 20

    var received: Int = 0
    var drift: Int = 0
    while received < MAILBOX_CAPACITY:
        let msg: Telemetry = core.recv_typed<Telemetry>()
        match msg:
        case Telemetry.sample(a, b, c):
            drift = drift + (a - received)
            drift = drift + (b - (received + 1))
            drift = drift + (c - (received + 2))
        case Telemetry.poison(a, b, c):
            return 30 + a + b + c
        received = received + 1

    if drift != 0:
        return 40

    let retry: Int = core.send_typed(me, Telemetry.sample(777, 778, 779))
    if retry != 0:
        return 50
    let after: Telemetry = core.recv_typed<Telemetry>()
    match after:
    case Telemetry.sample(a, b, c):
        if a != 777:
            return 60
        if b != 778:
            return 70
        if c != 779:
            return 80
    case Telemetry.poison(a, b, c):
        return 90 + a + b + c
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want typed mailbox backpressure to recover after drain without exposing failed payload", exitCode)
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

func TestActorFailureNonzeroExitBecomesDoneWithoutRestartBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum Signal:
    case item(Int)

func fails_after_notify() -> Int
uses actors:
    let _sent: Int = core.send(core.sender(), 21)
    return 9

func notifier() -> Int
uses actors:
    let _sent: Int = core.send(core.sender(), 33)
    return 0

func main() -> Int
uses actors, runtime:
    let failed: actor = core.spawn("fails_after_notify")
    let first: actor.recv_result_i32 = core.recv_until(core.deadline_ms(5))
    if first.error != 0:
        return 10 + first.error
    if first.value != 21:
        return 20 + first.value

    let _settle: Int = core.sleep_ms(5)
    let sent: Int = core.send(failed, 1)
    if sent != -4:
        return 40
    let tagged: Int = core.send_msg(failed, 2, 7)
    if tagged != -4:
        return 50
    let typed: Int = core.send_typed(failed, Signal.item(3))
    if typed != -4:
        return 60

    let _notifier: actor = core.spawn("notifier")
    let wake: actor.recv_result_i32 = core.recv_until(core.deadline_ms(5))
    if wake.error != 0:
        return 70 + wake.error
    if wake.value != 33:
        return 80 + wake.value

    let extra: actor.recv_result_i32 = core.recv_until(core.deadline_ms(1))
    if extra.error != 2:
        return 90 + extra.error
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want nonzero actor exit to become done without restart while scheduler keeps running", exitCode)
	}
}

func TestActorLifecycleReceivesPendingMessageFromDoneSenderBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func worker() -> Int
uses actors:
    let _sent: Int = core.send(core.sender(), 42)
    return 0

func main() -> Int
uses actors, runtime:
    let _peer: actor = core.spawn("worker")
    let _sleep: Int = core.sleep_ms(1)
    let msg: Int = core.recv()
    if msg != 42:
        return 20 + msg
    let reply_to_done: Int = core.send(core.sender(), 7)
    if reply_to_done != -4:
        return 40
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want pending message from done sender to remain receivable with done reply rejected", exitCode)
	}
}

func TestActorLifecycleDoneActorWithPendingMailboxDoesNotStallBlockedActorsBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func exits_without_receiving() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return 0

func blocked_forever() -> Int
uses actors:
    let _msg: Int = core.recv()
    return 0

func notifier() -> Int
uses actors:
    let _sent: Int = core.send(core.sender(), 33)
    return 0

func main() -> Int
uses actors, runtime:
    let done_with_pending: actor = core.spawn("exits_without_receiving")
    let queued: Int = core.send(done_with_pending, 11)
    if queued != 11:
        return 10
    let _blocked: actor = core.spawn("blocked_forever")
    let _notifier: actor = core.spawn("notifier")
    let msg: Int = core.recv()
    if msg != 33:
        return 20 + msg
    let _sleep: Int = core.sleep_ms(3)
    let after_done: Int = core.send(done_with_pending, 12)
    if after_done != -4:
        return 60
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want done actor with pending mailbox not to stall blocked actors and later sends to return -4", exitCode)
	}
}

func TestActorRejectedSendsDoNotConsumeMessagePool(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum Signal:
    case item(Int)

val MAILBOX_CAPACITY: i32 = 256
val THIRD_ACTOR_LIVE_MESSAGES: i32 = 232

func sleeper() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 0

func done() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let peer_a: actor = core.spawn("sleeper")
    let peer_b: actor = core.spawn("sleeper")
    let peer_c: actor = core.spawn("sleeper")
    var peer_sent: Int = 0
    while peer_sent < MAILBOX_CAPACITY:
        let ack: Int = core.send(peer_a, peer_sent)
        if ack != peer_sent:
            return 10
        peer_sent = peer_sent + 1

    let full: Int = core.send(peer_a, 777)
    if full != -2:
        return 20

    let finished: actor = core.spawn("done")
    let _sleep_done: Int = core.sleep_ms(1)
    let done_send: Int = core.send(finished, 1)
    if done_send != -4:
        return 30

    var spawned: Int = 0
    while spawned < 123:
        let _actor: actor = core.spawn("sleeper")
        spawned = spawned + 1
    let failed: actor = core.spawn("sleeper")
    let invalid: Int = core.send(failed, 1)
    if invalid != -3:
        return 40

    var sent_b: Int = 0
    while sent_b < MAILBOX_CAPACITY:
        let ack_b: Int = core.send(peer_b, sent_b)
        if ack_b != sent_b:
            return 50
        sent_b = sent_b + 1

    var sent_c: Int = 0
    while sent_c < THIRD_ACTOR_LIVE_MESSAGES:
        let ack_c: Int = core.send(peer_c, sent_c)
        if ack_c != sent_c:
            return 60
        sent_c = sent_c + 1

    let overflow: Int = core.send(peer_c, 123)
    if overflow != -1:
        return 70
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want rejected sends to leave message pool budget unchanged", exitCode)
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
		"backpressure is recoverable when the receiver drains messages",
		"does not enqueue a partial typed payload",
		"64 KiB",
		"744",
		"single-slot",
		"checked failure",
		"`-1`",
		"does not enqueue an",
		"overflow message",
		"Drained message nodes are reclaimed",
		"invalid actor handle",
		"`-3`",
		"done actor",
		"`-4`",
		"nonzero actor entry returns become the same user-visible `done`",
		"no actor status,",
		"actor join, actor exit-code, supervision, or restart API",
		"Missing-node/node_down is status/failure evidence only",
		"`node_down`",
		"does not imply automatic retry, reconnect, restart,",
		"supervision, or delivery retry",
		"## Lifecycle Matrix",
		"`ready`",
		"`blocked`",
		"`sleeping`",
		"`waiting`",
		"`done`",
		"message already queued in another actor's mailbox remains receivable after the",
		"sender is done",
		"Pending mailbox entries are not drained or delivered after the actor reaches",
		"`done`; this is a bounded local completion state",
		"no shutdown API",
		"supervision, restart, linking, or OTP-style lifecycle guarantee",
		"8 state slots",
		"rejects programs that require more than 8 actor-state slots before lowering",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("actors spec missing capacity-limit text %q", want)
		}
	}
}
