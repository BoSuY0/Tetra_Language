package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tetra_language/compiler/target"
)

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

func TestActorFairnessYieldingWorkersBothMakeBoundedProgressBuildAndRun(t *testing.T) {
	src := `
func worker_a() -> Int
uses actors, runtime:
    let parent: actor = core.sender()
    var i: Int = 0
    while i < 4:
        let _yielded: Int = core.yield()
        let payload: Int = 10 + i
        let sent: Int = core.send(parent, payload)
        if sent != payload:
            return 50 + i
        i = i + 1
    return 0

func worker_b() -> Int
uses actors, runtime:
    let parent: actor = core.sender()
    var i: Int = 0
    while i < 4:
        let _yielded: Int = core.yield()
        let payload: Int = 20 + i
        let sent: Int = core.send(parent, payload)
        if sent != payload:
            return 70 + i
        i = i + 1
    return 0

func main() -> Int
uses actors, runtime:
    let _a: actor = core.spawn("worker_a")
    let _b: actor = core.spawn("worker_b")
    var total: Int = 0
    var seen_a: Int = 0
    var seen_b: Int = 0
    var last_lane: Int = 0
    var run_len: Int = 0
    var max_run: Int = 0
    while total < 8:
        let msg: actor.recv_result_i32 = core.recv_until(core.deadline_ms(1))
        if msg.error != 0:
            return 80 + msg.error
        var lane: Int = 0
        if msg.value < 20:
            lane = 1
            seen_a = seen_a + 1
        if msg.value >= 20:
            lane = 2
            seen_b = seen_b + 1
        if lane == last_lane:
            run_len = run_len + 1
        if lane != last_lane:
            run_len = 1
            last_lane = lane
        if run_len > max_run:
            max_run = run_len
        if max_run > 2:
            return 100 + lane
        total = total + 1
    if seen_a != 4:
        return 10 + seen_a
    if seen_b != 4:
        return 20 + seen_b
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; yielding actors should both make bounded progress")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want bounded round-robin progress for yielding actors", exitCode)
	}
}

func TestActorStarvationTimedSleepersWakeInDeadlineOrderBuildAndRun(t *testing.T) {
	src := `
func slow() -> Int
uses actors, runtime:
    let _sleep: Int = core.sleep_ms(5)
    let _sent: Int = core.send(core.sender(), core.time_now_ms())
    return 0

func fast() -> Int
uses actors, runtime:
    let _sleep: Int = core.sleep_ms(2)
    let _sent: Int = core.send(core.sender(), core.time_now_ms())
    return 0

func main() -> Int
uses actors, runtime:
    let _slow: actor = core.spawn("slow")
    let _fast: actor = core.spawn("fast")
    let first: actor.recv_result_i32 = core.recv_until(core.deadline_ms(6))
    if first.error != 0:
        return 20 + first.error
    if first.value != 2:
        return 40 + first.value
    let second: actor.recv_result_i32 = core.recv_until(core.deadline_ms(6))
    if second.error != 0:
        return 60 + second.error
    if second.value != 5:
        return 80 + second.value
    let now: Int = core.time_now_ms()
    if now != 5:
        return 100 + now
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{}, 250*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out; sleeping actors should wake in deterministic deadline order")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want deterministic deadline-order wake for actor sleepers", exitCode)
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
