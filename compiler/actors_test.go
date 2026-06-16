package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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

func TestActorFanoutMailboxDrainSoakBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
val P09_SOAK_TOTAL: i32 = 512
val P09_SOAK_BATCH: i32 = 64

func echo_worker() -> Int
uses actors:
    var i: Int = 0
    while i < P09_SOAK_TOTAL:
        let msg: Int = core.recv()
        let reply: Int = msg + 1
        let sent: Int = core.send(core.sender(), reply)
        if sent != reply:
            return 10
        i = i + 1
    return 0

func main() -> Int
uses actors, runtime:
    let left: actor = core.spawn("echo_worker")
    let right: actor = core.spawn("echo_worker")
    var sent: Int = 0
    while sent < P09_SOAK_TOTAL:
        var batch: Int = 0
        var expected: Int = 0
        while batch < P09_SOAK_BATCH:
            let base: Int = sent + batch
            let left_payload: Int = base
            let right_payload: Int = 10000 + base
            let left_sent: Int = core.send(left, left_payload)
            if left_sent != left_payload:
                return 20
            let right_sent: Int = core.send(right, right_payload)
            if right_sent != right_payload:
                return 30
            expected = expected + left_payload + 1 + right_payload + 1
            batch = batch + 1

        var received: Int = 0
        var observed: Int = 0
        while received < P09_SOAK_BATCH * 2:
            let msg: actor.recv_result_i32 = core.recv_until(core.deadline_ms(1))
            if msg.error != 0:
                return 40 + msg.error
            observed = observed + msg.value
            received = received + 1
        if observed != expected:
            return 60
        sent = sent + P09_SOAK_BATCH
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(t, src, BuildOptions{Runtime: RuntimeBuiltin}, 2*time.Second)
	if timedOut {
		t.Fatalf("program timed out; actor fanout mailbox drain soak should stay bounded")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want bounded actor fanout mailbox drain soak", exitCode)
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
