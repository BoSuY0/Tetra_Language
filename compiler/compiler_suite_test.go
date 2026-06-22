package compiler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"tetra_language/compiler/internal/actorsrt"
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	lowerpkg "tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/machine"
	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/runtimeabi"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/internal/testkit"
	"tetra_language/compiler/memoryvocab"
	ctarget "tetra_language/compiler/target"
	"time"
)

// ---- abi_suite_test.go ----

func TestRunTargetABIChecksCoversP21Targets(t *testing.T) {
	tests := []struct {
		target string
		names  []string
	}{
		{
			target: "x86",
			names: []string{
				"x86 target model",
				"x86 i386 SysV classifier",
				"x86 varargs and sret ABI",
				"x86 pointer FFI object smoke",
				"x86 c_int FFI object smoke",
				"x86 c_uint FFI object smoke",
				"x86 ILP32 native/libc FFI object smoke",
				"x86 ref FFI null-return diagnostics",
				"x86 function-pointer FFI diagnostics",
				"x86 source native scalar diagnostics",
				"x86 stdout executable smoke",
				"x86 stderr fd runtime smoke",
				"x86 allocator executable smoke",
				"x86 allocator failure executable smoke",
				"x86 raw memory bounds executable smoke",
				"x86 raw pointer slot executable smoke",
				"x86 raw pointer offset slot executable smoke",
				"x86 island free executable smoke",
				"x86 stdlib runtime boundary diagnostics",
				"x86 filesystem runtime smoke",
				"x86 filesystem scheduler composition smoke",
				"x86 time runtime smoke",
				"x86 single-actor self-host runtime smoke",
				"x86 single-task self-host runtime smoke",
				"x86 typed-task self-host runtime smoke",
				"x86 staged typed-task self-host runtime smoke",
				"x86 task-group self-host runtime smoke",
				"x86 typed-task-group self-host runtime smoke",
				"x86 actor-state self-host runtime smoke",
				"x86 ctx_switch object smoke",
				"x86 target runtime boundary diagnostics",
				"x86 networking runtime boundary diagnostics",
				"x86 networking lifecycle runtime smoke",
				"x86 surface/distributed runtime boundary diagnostics",
				"x86 pointer atomic ABI width",
			},
		},
		{
			target: "x64",
			names: []string{
				"x64 target model",
				"x64 SysV classifier",
				"x64 SysV varargs and aggregates",
				"x64 source native scalar diagnostics",
				"x64 pointer FFI regression smoke",
				"x64 c_int FFI object smoke",
				"x64 c_uint FFI object smoke",
				"x64 filesystem scheduler composition smoke",
				"x64 networking runtime smoke",
				"x64 scheduler restriction regression smoke",
				"x64 pointer atomic ABI width",
			},
		},
		{
			target: "windows-x64",
			names: []string{
				"windows-x64 target model",
				"windows-x64 Win64 classifier",
				"windows-x64 Win64 varargs and aggregates",
				"windows-x64 object ABI smoke",
				"windows-x64 source native scalar diagnostics",
				"windows-x64 pointer atomic ABI width",
			},
		},
		{
			target: "macos-x64",
			names: []string{
				"macos-x64 target model",
				"macos-x64 SysV classifier",
				"macos-x64 SysV varargs and aggregates",
				"macos-x64 object ABI smoke",
				"macos-x64 source native scalar diagnostics",
				"macos-x64 pointer atomic ABI width",
			},
		},
		{
			target: "x32",
			names: []string{
				"x32 target model",
				"x32 SysV classifier",
				"x32 SysV varargs and aggregates",
				"x32 pointer FFI object smoke",
				"x32 c_int FFI object smoke",
				"x32 c_uint FFI object smoke",
				"x32 ILP32 native/libc FFI object smoke",
				"x32 ref FFI null-return diagnostics",
				"x32 function-pointer FFI diagnostics",
				"x32 source native scalar diagnostics",
				"x32 stdout executable smoke",
				"x32 stderr fd runtime smoke",
				"x32 allocator executable smoke",
				"x32 allocator failure executable smoke",
				"x32 raw memory bounds executable smoke",
				"x32 raw pointer slot executable smoke",
				"x32 raw pointer offset slot executable smoke",
				"x32 island free executable smoke",
				"x32 stdlib runtime boundary diagnostics",
				"x32 time runtime smoke",
				"x32 filesystem runtime smoke",
				"x32 filesystem scheduler composition smoke",
				"x32 single-actor self-host runtime smoke",
				"x32 single-task self-host runtime smoke",
				"x32 typed-task self-host runtime smoke",
				"x32 staged typed-task self-host runtime smoke",
				"x32 task-group self-host runtime smoke",
				"x32 typed-task-group self-host runtime smoke",
				"x32 actor-state self-host runtime smoke",
				"x32 ctx_switch object smoke",
				"x32 target runtime boundary diagnostics",
				"x32 networking runtime boundary diagnostics",
				"x32 networking lifecycle runtime smoke",
				"x32 surface/distributed runtime boundary diagnostics",
				"x32 pointer atomic ABI width",
			},
		},
		{
			target: "wasm32-wasi",
			names: []string{
				"wasm32-wasi target model",
				"wasm32-wasi slot ABI metadata",
				"wasm32-wasi struct/enum/slice/String return layout",
				"wasm32-wasi call boundary validation",
				"wasm32-wasi FFI repr(C) boundary policy",
			},
		},
		{
			target: "wasm32-web",
			names: []string{
				"wasm32-web target model",
				"wasm32-web slot ABI metadata",
				"wasm32-web struct/enum/slice/String return layout",
				"wasm32-web call boundary validation",
				"wasm32-web FFI repr(C) boundary policy",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			checks, err := RunTargetABIChecks(tt.target)
			if err != nil {
				t.Fatalf("RunTargetABIChecks(%s): %v", tt.target, err)
			}
			if len(checks) != len(tt.names) {
				t.Fatalf("checks = %#v, want %d checks", checks, len(tt.names))
			}
			for i, want := range tt.names {
				if checks[i].Name != want || checks[i].Error != "" {
					t.Fatalf("check[%d] = %#v, want passing %q", i, checks[i], want)
				}
			}
		})
	}
}

func TestP21ABIVerificationReportCoversTargetsTasksAndNonClaims(t *testing.T) {
	report := BuildP21ABIVerificationReport()
	if report.Schema != abiVerificationSchemaV1 {
		t.Fatalf("ABI report schema = %q, want %q", report.Schema, abiVerificationSchemaV1)
	}
	if report.Scope != abiVerificationScopeP211 {
		t.Fatalf("ABI report scope = %q, want %q", report.Scope, abiVerificationScopeP211)
	}
	if err := ValidateP21ABIVerificationReport(report); err != nil {
		t.Fatalf("ValidateP21ABIVerificationReport: %v", err)
	}

	targetRows := map[string]ABIVerificationTargetRow{}
	for _, row := range report.Targets {
		if row.Target == "" || row.ABI == "" || row.Status == "" || len(row.Evidence) == 0 {
			t.Fatalf("ABI target row missing required metadata: %#v", row)
		}
		targetRows[row.Target] = row
	}
	for _, target := range []string{
		"linux-x64",
		"linux-x86",
		"linux-x32",
		"macos-x64",
		"windows-x64",
		"wasm32-wasi",
		"wasm32-web",
	} {
		row, ok := targetRows[target]
		if !ok {
			t.Fatalf("ABI report missing target %s: %#v", target, report.Targets)
		}
		for _, task := range p21ABIVerificationTaskIDs() {
			if !p21ABIHasString(row.TaskCoverage, task) {
				t.Fatalf("ABI target %s missing task %s coverage: %#v", target, task, row)
			}
		}
	}

	taskRows := map[string]ABIVerificationTaskRow{}
	for _, row := range report.Tasks {
		if row.ID == "" || row.Name == "" || len(row.Targets) == 0 || len(row.Evidence) == 0 {
			t.Fatalf("ABI task row missing required metadata: %#v", row)
		}
		taskRows[row.ID] = row
	}
	for _, task := range p21ABIVerificationTaskIDs() {
		if _, ok := taskRows[task]; !ok {
			t.Fatalf("ABI report missing task row %s: %#v", task, report.Tasks)
		}
	}
	for _, nonClaim := range []string{
		"no runtime execution claim for build-only or wasm targets",
		"no C ABI claim for default structs",
		"no native C aggregate ABI claim for wasm targets",
		"no performance claim",
		"no safe-program semantics change",
	} {
		if !p21ABIHasString(report.NonClaims, nonClaim) {
			t.Fatalf("ABI report missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP21ABIVerificationReportRejectsFakeClaims(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ABIVerificationReport)
		want   string
	}{
		{
			name: "missing target",
			mutate: func(report *ABIVerificationReport) {
				report.Targets = report.Targets[1:]
			},
			want: "missing target",
		},
		{
			name: "missing task",
			mutate: func(report *ABIVerificationReport) {
				report.Tasks = report.Tasks[1:]
			},
			want: "missing task",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *ABIVerificationReport) {
				report.Targets[0].Evidence = []string{"placeholder evidence"}
			},
			want: "placeholder",
		},
		{
			name: "fake full runtime",
			mutate: func(report *ABIVerificationReport) {
				report.Claims = append(
					report.Claims,
					"full runtime execution verified for wasm32-wasi and linux-x86",
				)
			},
			want: "runtime execution",
		},
		{
			name: "fake wasm C aggregate ABI",
			mutate: func(report *ABIVerificationReport) {
				report.Claims = append(report.Claims, "wasm32-web native C aggregate ABI verified")
			},
			want: "wasm",
		},
		{
			name: "fake default struct C ABI",
			mutate: func(report *ABIVerificationReport) {
				report.Claims = append(report.Claims, "default structs have C ABI")
			},
			want: "default structs",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := cloneABIVerificationReport(BuildP21ABIVerificationReport())
			tc.mutate(&report)
			err := ValidateP21ABIVerificationReport(report)
			if err == nil {
				t.Fatalf("ValidateP21ABIVerificationReport accepted %#v", report)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func cloneABIVerificationReport(report ABIVerificationReport) ABIVerificationReport {
	report.Claims = append([]string{}, report.Claims...)
	report.NonClaims = append([]string{}, report.NonClaims...)
	report.Targets = append([]ABIVerificationTargetRow{}, report.Targets...)
	for i := range report.Targets {
		report.Targets[i].TaskCoverage = append([]string{}, report.Targets[i].TaskCoverage...)
		report.Targets[i].Evidence = append([]string{}, report.Targets[i].Evidence...)
		report.Targets[i].Claims = append([]string{}, report.Targets[i].Claims...)
	}
	report.Tasks = append([]ABIVerificationTaskRow{}, report.Tasks...)
	for i := range report.Tasks {
		report.Tasks[i].Targets = append([]string{}, report.Tasks[i].Targets...)
		report.Tasks[i].Evidence = append([]string{}, report.Tasks[i].Evidence...)
	}
	return report
}

func p21ABIHasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// ---- actors_capacity_test.go ----

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
uses actors, runtime:
    let _peer: actor = core.spawn("Slots.run")
    return 0
`, "actor 'Slots' state supports at most 8 slots, got 9")

	tgt, ok := ctarget.Host()
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
uses actors, runtime:
    let _peer: actor = core.spawn("Slots.run")
    return 0
`), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "actor_state_too_many_slots"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{Runtime: RuntimeBuiltin},
	)
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
val MESSAGE_POOL_SAFE_MESSAGES: i32 = 682
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
val THIRD_ACTOR_LIVE_MESSAGES: i32 = 170

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
		t.Fatalf(
			"exit code = %d, want checked message pool exhaustion without corrupting mailbox",
			exitCode,
		)
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
		t.Fatalf(
			"exit code = %d, want drained message nodes to be reusable beyond the old bump-only pool budget",
			exitCode,
		)
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
		t.Fatalf(
			("exit code = %d, want drained typed message nodes and " +
				"payload slots to be reusable beyond the old bump-only pool " +
				"budget"),
			exitCode,
		)
	}
}

func TestActorMessagePoolExhaustionCoversTaggedMessages(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
val MAILBOX_CAPACITY: i32 = 256
val THIRD_ACTOR_LIVE_MESSAGES: i32 = 170

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
val THIRD_ACTOR_LIVE_MESSAGES: i32 = 170

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
		t.Fatalf(
			("exit code = %d, want tagged mailbox backpressure to recover " +
				"after drain without enqueued overflow message"),
			exitCode,
		)
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
		t.Fatalf(
			("exit code = %d, want typed mailbox backpressure to recover " +
				"after drain without exposing failed payload"),
			exitCode,
		)
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

func TestActorSendToStoppingActorReturnsCheckedFailure(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum Signal:
    case item(Int)

func target() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let peer: actor = core.spawn("target")
    let reason: actor.exit_reason = core.actor_exit_reason(peer)
    let stopped: Int = core.actor_stop(peer, reason)
    if stopped != 0:
        return 10 + stopped
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
		t.Fatalf("exit code = %d, want checked stopping actor send-path failures", exitCode)
	}
}

func TestActorSendToCanceledActorReturnsCheckedFailure(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
enum Signal:
    case item(Int)

func parked() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 0

func worker() -> Int
uses actors, runtime:
    let peer: actor = core.spawn("parked")
    let current: task.group = core.task_group_current()
    let _canceled: task.group = core.task_group_cancel(current)
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

func main() -> Int
uses actors, runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if result.error != 0:
        return 50 + result.error
    return result.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{Runtime: RuntimeBuiltin},
		2*time.Second,
	)
	if timedOut {
		t.Fatalf("program timed out; task_group_close should treat joined reclaimable task actors as done")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want checked canceled actor send-path failures", exitCode)
	}
}

func TestActorSlotReuseInvalidatesStaleHandleBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func done() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let first: actor = core.spawn("done")
    let _first_settle: Int = core.sleep_ms(2)
    let _first_waited: actor.wait_result = core.actor_wait(first)
    let second: actor = core.spawn("done")
    let stale_send: Int = core.send(first, 1)
    if stale_send != -3:
        return 20
    let _second_settle: Int = core.sleep_ms(1)
    let done_send: Int = core.send(second, 2)
    if done_send != -4:
        return 30
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want stale reused actor handle rejected", exitCode)
	}
}

func TestActorDoneSlotsRequireWaitBeforeReuseBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func status_code(status: actor.status) -> Int:
    match status:
    case actor.status.starting:
        return 0
    case actor.status.ready:
        return 1
    case actor.status.running:
        return 2
    case actor.status.blocked:
        return 3
    case actor.status.sleeping:
        return 4
    case actor.status.waiting:
        return 5
    case actor.status.stopping:
        return 6
    case actor.status.exited_normal:
        return 7
    case actor.status.exited_error:
        return 8
    case actor.status.canceled:
        return 9
    case actor.status.restarting:
        return 10
    case actor.status.dead:
        return 11

func done() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    var spawned: Int = 0
    while spawned < 127:
        let _peer: actor = core.spawn("done")
        spawned = spawned + 1

    let _settle: Int = core.sleep_ms(5)
    let overflow: actor = core.spawn("done")
    let overflow_wait: actor.wait_result = core.actor_wait(overflow)
    let overflow_status: Int = status_code(overflow_wait.status)
    if overflow_status != 11:
        return 20 + overflow_status
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{Runtime: RuntimeBuiltin},
		2*time.Second,
	)
	if timedOut {
		t.Fatalf("actor done-slot reclaimability smoke timed out; stdout=%q", stdout)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf(
			"exit code = %d, want done-but-unwaited actor slots not reused before wait",
			exitCode,
		)
	}
}

func TestActorWaitBlocksUntilTargetDoneBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func slow_exit() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return 7

func main() -> Int
uses actors:
    let peer: actor = core.spawn("slow_exit")
    let _waited: actor.wait_result = core.actor_wait(peer)
    let after_wait: Int = core.send(peer, 1)
    if after_wait != -4:
        return 20
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{Runtime: RuntimeBuiltin},
		2*time.Second,
	)
	if timedOut {
		t.Fatalf("actor_wait smoke timed out; stdout=%q", stdout)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want actor_wait to return after target exits", exitCode)
	}
}

func TestActorWaitResultStatusUsesPublicLifecycleEnumBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func status_code(status: actor.status) -> Int:
    match status:
    case actor.status.starting:
        return 0
    case actor.status.ready:
        return 1
    case actor.status.running:
        return 2
    case actor.status.blocked:
        return 3
    case actor.status.sleeping:
        return 4
    case actor.status.waiting:
        return 5
    case actor.status.stopping:
        return 6
    case actor.status.exited_normal:
        return 7
    case actor.status.exited_error:
        return 8
    case actor.status.canceled:
        return 9
    case actor.status.restarting:
        return 10
    case actor.status.dead:
        return 11

func done() -> Int:
    return 0

func fail() -> Int:
    return 9

func main() -> Int
uses actors:
    let normal: actor = core.spawn("done")
    let normal_result: actor.wait_result = core.actor_wait(normal)
    let normal_status: Int = status_code(normal_result.status)
    if normal_status != 7:
        return 20 + normal_status

    let failed: actor = core.spawn("fail")
    let failed_result: actor.wait_result = core.actor_wait(failed)
    let failed_status: Int = status_code(failed_result.status)
    if failed_status != 8:
        return 60 + failed_status
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{Runtime: RuntimeBuiltin},
		2*time.Second,
	)
	if timedOut {
		t.Fatalf("actor_wait result status smoke timed out; stdout=%q", stdout)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want actor_wait result to expose public lifecycle statuses", exitCode)
	}
}

func TestActorWaitInvalidAndStaleRefsReturnDeadStatusBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func status_code(status: actor.status) -> Int:
    match status:
    case actor.status.starting:
        return 0
    case actor.status.ready:
        return 1
    case actor.status.running:
        return 2
    case actor.status.blocked:
        return 3
    case actor.status.sleeping:
        return 4
    case actor.status.waiting:
        return 5
    case actor.status.stopping:
        return 6
    case actor.status.exited_normal:
        return 7
    case actor.status.exited_error:
        return 8
    case actor.status.canceled:
        return 9
    case actor.status.restarting:
        return 10
    case actor.status.dead:
        return 11

func sleeper() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 0

func done() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    var spawned: Int = 0
    while spawned < 127:
        let _peer: actor = core.spawn("sleeper")
        spawned = spawned + 1

    let invalid: actor = core.spawn("sleeper")
    let invalid_wait: actor.wait_result = core.actor_wait(invalid)
    let invalid_status: Int = status_code(invalid_wait.status)
    if invalid_status != 11:
        return 20 + invalid_status

    let _advance: Int = core.sleep_ms(100)
    let first: actor = core.spawn("done")
    let _first_settle: Int = core.sleep_ms(2)
    let _first_waited: actor.wait_result = core.actor_wait(first)
    let _second: actor = core.spawn("done")
    let stale_wait: actor.wait_result = core.actor_wait(first)
    let stale_status: Int = status_code(stale_wait.status)
    if stale_status != 11:
        return 40 + stale_status
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{Runtime: RuntimeBuiltin},
		2*time.Second,
	)
	if timedOut {
		t.Fatalf("actor_wait invalid/stale status smoke timed out; stdout=%q", stdout)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want invalid and stale actor_wait results to expose dead status", exitCode)
	}
}

func TestLibCoreActorsLifecycleWrappersBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
module main

import lib.core.actors as actors

func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(25)
    return 0

func done() -> Int:
    return 0

func fail() -> Int:
    return 9

func status_score(result: actors.StatusResult) -> Int:
    match result:
    case actors.StatusResult.ok(status):
        match status:
        case actors.ActorStatus.starting:
            return 0
        case actors.ActorStatus.ready:
            return 1
        case actors.ActorStatus.running:
            return 2
        case actors.ActorStatus.blocked:
            return 3
        case actors.ActorStatus.sleeping:
            return 4
        case actors.ActorStatus.waiting:
            return 5
        case actors.ActorStatus.stopping:
            return 6
        case actors.ActorStatus.exited_normal:
            return 7
        case actors.ActorStatus.exited_error(code):
            return 80 + code
        case actors.ActorStatus.canceled:
            return 9
        case actors.ActorStatus.restarting:
            return 10
        case actors.ActorStatus.dead:
            return 11
        case actors.ActorStatus.unknown(code):
            return 100 + code
    case actors.StatusResult.invalid:
        return 200
    case actors.StatusResult.stale:
        return 201
    case actors.StatusResult.node_down:
        return 202

func wait_score(result: actors.WaitResult) -> Int:
    match result:
    case actors.WaitResult.exited(reason):
        match reason:
        case actors.ExitReason.normal:
            return 0
        case actors.ExitReason.shutdown(code):
            return 10 + code
        case actors.ExitReason.error(code):
            return 20 + code
        case actors.ExitReason.canceled:
            return 30
        case actors.ExitReason.killed:
            return 40
        case actors.ExitReason.node_down(code):
            return 50 + code
        case actors.ExitReason.protocol_error(code):
            return 60 + code
        case actors.ExitReason.runtime_error(code):
            return 70 + code
        case actors.ExitReason.unknown(kind, code):
            return 80 + kind + code
    case actors.WaitResult.timeout:
        return 100
    case actors.WaitResult.canceled:
        return 101
    case actors.WaitResult.invalid:
        return 102
    case actors.WaitResult.stale:
        return 103
    case actors.WaitResult.node_down:
        return 104

func stop_score(result: actors.StopResult) -> Int:
    match result:
    case actors.StopResult.requested:
        return 0
    case actors.StopResult.already_exited(reason):
        return 10
    case actors.StopResult.invalid:
        return 20
    case actors.StopResult.stale:
        return 21
    case actors.StopResult.node_down:
        return 22

func link_score(result: actors.LinkResult) -> Int:
    match result:
    case actors.LinkResult.linked:
        return 0
    case actors.LinkResult.already_linked:
        return 1
    case actors.LinkResult.target_exited(reason):
        return 2
    case actors.LinkResult.resource_exhausted:
        return 3
    case actors.LinkResult.invalid:
        return 4
    case actors.LinkResult.stale:
        return 5
    case actors.LinkResult.node_down:
        return 6

func monitor_score(result: actors.MonitorResult) -> Int
uses actors:
    match result:
    case actors.MonitorResult.monitoring(reference):
        if actors.demonitor(reference, false):
            return 0
        return 40
    case actors.MonitorResult.target_already_exited(reference):
        if actors.demonitor(reference, false):
            return 1
        return 41
    case actors.MonitorResult.resource_exhausted:
        return 2
    case actors.MonitorResult.invalid:
        return 3
    case actors.MonitorResult.stale:
        return 4
    case actors.MonitorResult.node_down:
        return 5

func main() -> Int
uses actors, runtime:
    let self_status: Int = status_score(actors.status(core.self()))
    if self_status != 2:
        return 10 + self_status

    let normal: actor = core.spawn("done")
    let normal_wait: Int = wait_score(actors.wait(normal))
    if normal_wait != 0:
        return 40 + normal_wait

    let failed: actor = core.spawn("fail")
    let failed_wait: Int = wait_score(actors.wait(failed))
    if failed_wait != 29:
        return 80 + failed_wait

    let timed_peer: actor = core.spawn("slow")
    let timed_wait: Int = wait_score(actors.wait_until(timed_peer, core.deadline_ms(1)))
    if timed_wait != 100:
        return 120 + timed_wait
    let _timed_done: actors.WaitResult = actors.wait(timed_peer)

    let stoppable: actor = core.spawn("slow")
    let stopped: Int = stop_score(actors.stop(stoppable, actors.ExitReason.normal))
    if stopped != 0:
        return 160 + stopped

    let done_peer: actor = core.spawn("done")
    let _settle: Int = core.sleep_ms(3)
    let already_done: Int = stop_score(actors.stop(done_peer, actors.ExitReason.normal))
    if already_done != 10:
        return 200 + already_done

    let linked_peer: actor = core.spawn("slow")
    let linked: Int = link_score(actors.link(linked_peer))
    if linked != 0:
        return 220 + linked
    if !actors.unlink(linked_peer):
        return 230

    let monitored_peer: actor = core.spawn("slow")
    let monitored: Int = monitor_score(actors.monitor(monitored_peer))
    if monitored != 0:
        return 240 + monitored

    if !actors.set_trap_exit(true):
        return 250
    if !actors.set_trap_exit(false):
        return 251

    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{
			Runtime:         RuntimeBuiltin,
			DependencyRoots: []ModuleRoot{{Root: projectRoot(t)}},
		},
		2*time.Second,
	)
	if timedOut {
		t.Fatalf("lib.core.actors lifecycle wrapper smoke timed out; stdout=%q", stdout)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want lib.core.actors lifecycle wrappers to map current runtime results", exitCode)
	}
}

func TestLibCoreActorsStatusResultInvalidAndStaleBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
module main

import lib.core.actors as actors

func done() -> Int:
    return 0

func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(200)
    return 0

func status_result_code(result: actors.StatusResult) -> Int:
    match result:
    case actors.StatusResult.ok(status):
        return 0
    case actors.StatusResult.invalid:
        return 1
    case actors.StatusResult.stale:
        return 2
    case actors.StatusResult.node_down:
        return 3

func main() -> Int
uses actors, runtime:
    let first: actor = core.spawn("done")
    let _first_waited: actors.WaitResult = actors.wait(first)
    let second: actor = core.spawn("done")
    let stale_status: Int = status_result_code(actors.status(first))
    if stale_status != 2:
        return 10 + stale_status
    let _second_waited: actors.WaitResult = actors.wait(second)

    var spawned: Int = 0
    while spawned < 127:
        let _peer: actor = core.spawn("slow")
        spawned = spawned + 1
    let invalid: actor = core.spawn("slow")
    let invalid_status: Int = status_result_code(actors.status(invalid))
    if invalid_status != 1:
        return 30 + invalid_status
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{
			Runtime:         RuntimeBuiltin,
			DependencyRoots: []ModuleRoot{{Root: projectRoot(t)}},
		},
		2*time.Second,
	)
	if timedOut {
		t.Fatalf("lib.core.actors status invalid/stale smoke timed out; stdout=%q", stdout)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want StatusResult.invalid/stale from lib.core.actors.status", exitCode)
	}
}

func TestLibCoreActorsLifecycleWrappersInvalidAndStaleTaxonomyBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
module main

import lib.core.actors as actors

func done() -> Int:
    return 0

func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(200)
    return 0

func stop_result_code(result: actors.StopResult) -> Int:
    match result:
    case actors.StopResult.requested:
        return 0
    case actors.StopResult.already_exited(reason):
        return 4
    case actors.StopResult.invalid:
        return 1
    case actors.StopResult.stale:
        return 2
    case actors.StopResult.node_down:
        return 3

func link_result_code(result: actors.LinkResult) -> Int:
    match result:
    case actors.LinkResult.linked:
        return 0
    case actors.LinkResult.already_linked:
        return 5
    case actors.LinkResult.target_exited(reason):
        return 4
    case actors.LinkResult.resource_exhausted:
        return 6
    case actors.LinkResult.invalid:
        return 1
    case actors.LinkResult.stale:
        return 2
    case actors.LinkResult.node_down:
        return 3

func monitor_result_code(result: actors.MonitorResult) -> Int
uses actors:
    match result:
    case actors.MonitorResult.monitoring(reference):
        let removed_live: Bool = actors.demonitor(reference, false)
        return 0
    case actors.MonitorResult.target_already_exited(reference):
        let removed_done: Bool = actors.demonitor(reference, false)
        return 4
    case actors.MonitorResult.resource_exhausted:
        return 6
    case actors.MonitorResult.invalid:
        return 1
    case actors.MonitorResult.stale:
        return 2
    case actors.MonitorResult.node_down:
        return 3

func main() -> Int
uses actors, runtime:
    let first: actor = core.spawn("done")
    let _first_waited: actors.WaitResult = actors.wait(first)
    let second: actor = core.spawn("done")

    let stale_stop: Int = stop_result_code(actors.stop(first, actors.ExitReason.normal))
    if stale_stop != 2:
        return 10 + stale_stop
    let stale_link: Int = link_result_code(actors.link(first))
    if stale_link != 2:
        return 20 + stale_link
    let stale_monitor: Int = monitor_result_code(actors.monitor(first))
    if stale_monitor != 2:
        return 30 + stale_monitor
    let _second_waited: actors.WaitResult = actors.wait(second)

    var spawned: Int = 0
    while spawned < 127:
        let _peer: actor = core.spawn("slow")
        spawned = spawned + 1
    let invalid: actor = core.spawn("slow")

    let invalid_stop: Int = stop_result_code(actors.stop(invalid, actors.ExitReason.normal))
    if invalid_stop != 1:
        return 40 + invalid_stop
    let invalid_link: Int = link_result_code(actors.link(invalid))
    if invalid_link != 1:
        return 50 + invalid_link
    let invalid_monitor: Int = monitor_result_code(actors.monitor(invalid))
    if invalid_monitor != 1:
        return 60 + invalid_monitor
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{
			Runtime:         RuntimeBuiltin,
			DependencyRoots: []ModuleRoot{{Root: projectRoot(t)}},
		},
		2*time.Second,
	)
	if timedOut {
		t.Fatalf("lib.core.actors lifecycle taxonomy smoke timed out; stdout=%q", stdout)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want StopResult/LinkResult/MonitorResult invalid/stale taxonomy", exitCode)
	}
}

func TestLibCoreActorsWaitInvalidAndStaleTaxonomyBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
module main

import lib.core.actors as actors

func done() -> Int:
    return 0

func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(200)
    return 0

func wait_result_code(result: actors.WaitResult) -> Int:
    match result:
    case actors.WaitResult.exited(reason):
        return 0
    case actors.WaitResult.timeout:
        return 4
    case actors.WaitResult.canceled:
        return 5
    case actors.WaitResult.invalid:
        return 1
    case actors.WaitResult.stale:
        return 2
    case actors.WaitResult.node_down:
        return 3

func main() -> Int
uses actors, runtime:
    let first: actor = core.spawn("done")
    let _first_waited: actors.WaitResult = actors.wait(first)
    let second: actor = core.spawn("done")

    let stale_wait: Int = wait_result_code(actors.wait(first))
    if stale_wait != 2:
        return 10 + stale_wait
    let stale_wait_until: Int = wait_result_code(actors.wait_until(first, 0))
    if stale_wait_until != 2:
        return 20 + stale_wait_until
    let _second_waited: actors.WaitResult = actors.wait(second)

    var spawned: Int = 0
    while spawned < 127:
        let _peer: actor = core.spawn("slow")
        spawned = spawned + 1
    let invalid: actor = core.spawn("slow")

    let invalid_wait: Int = wait_result_code(actors.wait(invalid))
    if invalid_wait != 1:
        return 30 + invalid_wait
    let invalid_wait_until: Int = wait_result_code(actors.wait_until(invalid, 0))
    if invalid_wait_until != 1:
        return 40 + invalid_wait_until
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{
			Runtime:         RuntimeBuiltin,
			DependencyRoots: []ModuleRoot{{Root: projectRoot(t)}},
		},
		2*time.Second,
	)
	if timedOut {
		t.Fatalf("lib.core.actors wait taxonomy smoke timed out; stdout=%q", stdout)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want WaitResult invalid/stale taxonomy", exitCode)
	}
}

func TestActorWaitUntilTimesOutBeforeTargetDoneBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func slow_exit() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(10)
    return 7

func main() -> Int
uses actors, runtime:
    let peer: actor = core.spawn("slow_exit")
    let _early: actor.wait_result = core.actor_wait_until(peer, core.deadline_ms(3))
    let after_timeout: Int = core.time_now_ms()
    if after_timeout != 3:
        return 10 + after_timeout
    let still_alive_send: Int = core.send(peer, 1)
    if still_alive_send != 1:
        return 30
    let _final: actor.wait_result = core.actor_wait(peer)
    let done_send: Int = core.send(peer, 2)
    if done_send != -4:
        return 40
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{Runtime: RuntimeBuiltin},
		2*time.Second,
	)
	if timedOut {
		t.Fatalf("actor_wait_until smoke timed out; stdout=%q", stdout)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want actor_wait_until to time out before target exits", exitCode)
	}
}

func TestActorStatusEnumReadyAndExitedBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func status_code(status: actor.status) -> Int:
    match status:
    case actor.status.starting:
        return 0
    case actor.status.ready:
        return 1
    case actor.status.running:
        return 2
    case actor.status.blocked:
        return 3
    case actor.status.sleeping:
        return 4
    case actor.status.waiting:
        return 5
    case actor.status.stopping:
        return 6
    case actor.status.exited_normal:
        return 7
    case actor.status.exited_error:
        return 8
    case actor.status.canceled:
        return 9
    case actor.status.restarting:
        return 10
    case actor.status.dead:
        return 11

func done() -> Int:
    return 0

func fail() -> Int:
    return 9

func main() -> Int
uses actors, runtime:
    let self_status: Int = status_code(core.actor_status(core.self()))
    if self_status != 2:
        return 20 + self_status
    let normal: actor = core.spawn("done")
    let failed: actor = core.spawn("fail")
    let _settle: Int = core.sleep_ms(2)
    let normal_status: Int = status_code(core.actor_status(normal))
    if normal_status != 7:
        return 40 + normal_status
    let failed_status: Int = status_code(core.actor_status(failed))
    if failed_status != 8:
        return 60 + failed_status
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want actor_status v1 enum mapping with running self", exitCode)
	}
}

func TestActorStatusRunningSurvivesYieldBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func status_code(status: actor.status) -> Int:
    match status:
    case actor.status.running:
        return 2
    case actor.status.ready:
        return 1
    case actor.status.blocked:
        return 3
    case actor.status.waiting:
        return 5
    case actor.status.dead:
        return 11
    case actor.status.starting:
        return 0
    case actor.status.sleeping:
        return 4
    case actor.status.stopping:
        return 6
    case actor.status.exited_normal:
        return 7
    case actor.status.exited_error:
        return 8
    case actor.status.canceled:
        return 9
    case actor.status.restarting:
        return 10

func main() -> Int
uses actors, runtime:
    let before: Int = status_code(core.actor_status(core.self()))
    if before != 2:
        return 20 + before
    let _yielded: Int = core.yield()
    let after: Int = status_code(core.actor_status(core.self()))
    if after != 2:
        return 40 + after
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want current actor to observe running before and after yield", exitCode)
	}
}

func TestActorStatusStartingBeforeFirstDispatchBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func status_code(status: actor.status) -> Int:
    match status:
    case actor.status.starting:
        return 0
    case actor.status.ready:
        return 1
    case actor.status.running:
        return 2
    case actor.status.blocked:
        return 3
    case actor.status.sleeping:
        return 4
    case actor.status.waiting:
        return 5
    case actor.status.stopping:
        return 6
    case actor.status.exited_normal:
        return 7
    case actor.status.exited_error:
        return 8
    case actor.status.canceled:
        return 9
    case actor.status.restarting:
        return 10
    case actor.status.dead:
        return 11

func done() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let peer: actor = core.spawn("done")
    let initial: Int = status_code(core.actor_status(peer))
    if initial != 0:
        return 20 + initial
    let _yielded: Int = core.yield()
    let after: Int = status_code(core.actor_status(peer))
    if after != 7:
        return 40 + after
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want spawned actor to start as starting then exit normally", exitCode)
	}
}

func TestActorStatusStoppingAfterActorStopBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func status_code(status: actor.status) -> Int:
    match status:
    case actor.status.starting:
        return 0
    case actor.status.ready:
        return 1
    case actor.status.running:
        return 2
    case actor.status.blocked:
        return 3
    case actor.status.sleeping:
        return 4
    case actor.status.waiting:
        return 5
    case actor.status.stopping:
        return 6
    case actor.status.exited_normal:
        return 7
    case actor.status.exited_error:
        return 8
    case actor.status.canceled:
        return 9
    case actor.status.restarting:
        return 10
    case actor.status.dead:
        return 11

func target() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let peer: actor = core.spawn("target")
    let reason: actor.exit_reason = core.actor_exit_reason(peer)
    let requested: Int = core.actor_stop(peer, reason)
    if requested != 0:
        return 10 + requested
    let stopping: Int = status_code(core.actor_status(peer))
    if stopping != 6:
        return 20 + stopping
    let _yielded: Int = core.yield()
    let after: Int = status_code(core.actor_status(peer))
    if after != 7:
        return 40 + after
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want actor_stop to expose stopping then finalize normal exit", exitCode)
	}
}

func TestActorStatusCanceledAfterTaskGroupCancelBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func status_code(status: actor.status) -> Int:
    match status:
    case actor.status.starting:
        return 0
    case actor.status.ready:
        return 1
    case actor.status.running:
        return 2
    case actor.status.blocked:
        return 3
    case actor.status.sleeping:
        return 4
    case actor.status.waiting:
        return 5
    case actor.status.stopping:
        return 6
    case actor.status.exited_normal:
        return 7
    case actor.status.exited_error:
        return 8
    case actor.status.canceled:
        return 9
    case actor.status.restarting:
        return 10
    case actor.status.dead:
        return 11

func worker() -> Int
uses actors, runtime:
    let group: task.group = core.task_group_current()
    let _canceled: task.group = core.task_group_cancel(group)
    return status_code(core.actor_status(core.self()))

func main() -> Int
uses actors, runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if result.error != 0:
        return 20 + result.error
    if result.value != 9:
        return 40 + result.value
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want task-group cancellation to expose actor.status.canceled", exitCode)
	}
}

func TestActorMonitorRefsAreUniqueAndDemonitorableBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func idle() -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("idle")
    let first: actor.monitor = core.actor_monitor(peer)
    let second: actor.monitor = core.actor_monitor(peer)
    if first == second:
        return 20
    let removed_first: Int = core.actor_demonitor(first)
    if removed_first != 0:
        return 30
    let removed_second: Int = core.actor_demonitor(second)
    if removed_second != 0:
        return 40
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want unique monitor refs and successful demonitor cleanup", exitCode)
	}
}

func TestActorLinkPropagatesAbnormalExitBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func waits_then_fails() -> Int
uses actors:
    let _wake: Int = core.recv()
    return 9

func main() -> Int
uses actors, runtime:
    let peer: actor = core.spawn("waits_then_fails")
    let linked: Int = core.actor_link(peer)
    if linked != 0:
        return 20 + linked
    let wake: Int = core.send(peer, 1)
    if wake != 1:
        return 30
    let _settle: Int = core.sleep_ms(5)
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{Runtime: RuntimeBuiltin},
		2*time.Second,
	)
	if timedOut {
		t.Fatalf("actor_link smoke timed out; stdout=%q", stdout)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 9 {
		t.Fatalf("exit code = %d, want linked abnormal exit to propagate reason 9", exitCode)
	}
}

func TestActorUnlinkStopsAbnormalExitPropagationBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func waits_then_fails() -> Int
uses actors:
    let _wake: Int = core.recv()
    return 9

func main() -> Int
uses actors, runtime:
    let peer: actor = core.spawn("waits_then_fails")
    let linked: Int = core.actor_link(peer)
    if linked != 0:
        return 20 + linked
    let unlinked: Int = core.actor_unlink(peer)
    if unlinked != 0:
        return 40 + unlinked
    let wake: Int = core.send(peer, 1)
    if wake != 1:
        return 60
    let _settle: Int = core.sleep_ms(5)
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{Runtime: RuntimeBuiltin},
		2*time.Second,
	)
	if timedOut {
		t.Fatalf("actor_unlink smoke timed out; stdout=%q", stdout)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want actor_unlink to remove abnormal exit propagation", exitCode)
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
		t.Fatalf(
			("exit code = %d, want nonzero actor exit to become done " +
				"without restart while scheduler keeps running"),
			exitCode,
		)
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
		t.Fatalf(
			("exit code = %d, want pending message from done sender to " +
				"remain receivable with done reply rejected"),
			exitCode,
		)
	}
}

func TestActorLifecycleDoneActorWithPendingMailboxDoesNotStallBlockedActorsBuildAndRun(
	t *testing.T,
) {
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
		t.Fatalf(
			("exit code = %d, want done actor with pending mailbox not to " +
				"stall blocked actors and later sends to return -4"),
			exitCode,
		)
	}
}

func TestActorLifecycleDoneActorDrainsPendingMailboxIntoMessagePoolBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
val MAILBOX_CAPACITY: i32 = 256
val THIRD_ACTOR_LIVE_MESSAGES: i32 = 170

func exits_without_receiving() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return 0

func sleeper() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 0

func main() -> Int
uses actors, runtime:
    let done_with_pending: actor = core.spawn("exits_without_receiving")
    var queued: Int = 0
    while queued < MAILBOX_CAPACITY:
        let ack: Int = core.send(done_with_pending, queued)
        if ack != queued:
            return 10
        queued = queued + 1

    let _settle_done: Int = core.sleep_ms(5)
    let done_send: Int = core.send(done_with_pending, 900)
    if done_send != -4:
        return 20

    let peer_a: actor = core.spawn("sleeper")
    let peer_b: actor = core.spawn("sleeper")
    let peer_c: actor = core.spawn("sleeper")

    var sent_a: Int = 0
    while sent_a < MAILBOX_CAPACITY:
        let ack_a: Int = core.send(peer_a, sent_a)
        if ack_a != sent_a:
            return 30
        sent_a = sent_a + 1

    var sent_b: Int = 0
    while sent_b < MAILBOX_CAPACITY:
        let ack_b: Int = core.send(peer_b, sent_b)
        if ack_b != sent_b:
            return 40
        sent_b = sent_b + 1

    var sent_c: Int = 0
    while sent_c < THIRD_ACTOR_LIVE_MESSAGES:
        let ack_c: Int = core.send(peer_c, sent_c)
        if ack_c != sent_c:
            return 50
        sent_c = sent_c + 1

    let overflow: Int = core.send(peer_c, 123)
    if overflow != -1:
        return 60
    return 0
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Runtime: RuntimeBuiltin})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf(
			"exit code = %d, want done actor pending mailbox nodes reclaimed into message pool",
			exitCode,
		)
	}
}

func TestActorLifetimeSpawnsExceedTenThousandUnderConcurrentCapBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
val TOTAL_LIFETIME_SPAWNS: i32 = 10001

func done() -> Int:
    return 0

func main() -> Int
uses actors:
    var spawned: Int = 0
    while spawned < TOTAL_LIFETIME_SPAWNS:
        let peer: actor = core.spawn("done")
        let _waited: actor.wait_result = core.actor_wait(peer)
        let done_send: Int = core.send(peer, spawned)
        if done_send == -3:
            return 20
        if done_send != -4:
            return 30
        spawned = spawned + 1
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{Runtime: RuntimeBuiltin},
		5*time.Second,
	)
	if timedOut {
		t.Fatalf("program timed out; actor lifetime spawn soak should stay under the concurrent cap")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf(
			"exit code = %d, want more than 10000 lifetime spawns under the concurrent cap",
			exitCode,
		)
	}
}

func TestTaskLifetimeSpawnsExceedTenThousandUnderConcurrentCapBuildAndRun(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
val TOTAL_TASK_LIFETIME_SPAWNS: i32 = 10001

func worker() -> Int:
    return 1

func main() -> Int
uses runtime:
    var spawned: Int = 0
    var total: Int = 0
    while spawned < TOTAL_TASK_LIFETIME_SPAWNS:
        let task: task.i32 = core.task_spawn_i32("worker")
        let result: task.result_i32 = core.task_join_result_i32(task)
        if result.error != 0:
            return 20 + result.error
        total = total + result.value
        spawned = spawned + 1
    if total != TOTAL_TASK_LIFETIME_SPAWNS:
        return 40
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{Runtime: RuntimeBuiltin},
		5*time.Second,
	)
	if timedOut {
		t.Fatalf("program timed out; joined task lifetime spawn soak should stay under the concurrent cap")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf(
			"exit code = %d, want more than 10000 joined task lifetime spawns under the concurrent cap",
			exitCode,
		)
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
val THIRD_ACTOR_LIVE_MESSAGES: i32 = 170
val SLEEPERS_AFTER_DONE_REUSE: i32 = 124

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
    while spawned < SLEEPERS_AFTER_DONE_REUSE:
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
		t.Fatalf(
			"exit code = %d, want rejected sends to leave message pool budget unchanged",
			exitCode,
		)
	}
}

func TestActorRuntimeCapacityLimitsDocumented(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "docs", "spec", "runtime", "actors.md"))
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
		"682",
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
		"Waited reclaimable actor slots reset stored",
		"initial stack frames instead of mapping a fresh stack",
		"more than 10,000 lifetime spawns",
		"nonzero actor entry returns become the same user-visible `done`",
		"public `lib.core.actors` lifecycle wrapper surface is present",
		"not supervision, restart",
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
		"Pending mailbox entries are drained into the runtime message-pool free list when the actor reaches",
		"`done`; they are not delivered after completion",
		"not a shutdown API",
		"bounded link/unlink",
		"monitor/demonitor",
		"cleanup, and trap-exit toggling",
		"local lifecycle wrappers now exist",
		"Task join, timed join, poll, and typed task join observers",
		"`reclaimable` target actor slots as terminal result states",
		"Successful `core.task_join_i32(task)`, `core.task_join_result_i32(task)`",
		"mark the target actor slot `reclaimable`",
		"`core.task_poll_i32(task)`",
		"non-consuming: it treats",
		"`core.task_group_close(group)` treats joined",
		"reclaimable task actors as terminal",
		"more than 10,000 sequential task lifetimes",
		"supervision, restart, remote lifecycle completion",
		"8 state slots",
		"rejects programs that require more than 8 actor-state slots before lowering",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("actors spec missing capacity-limit text %q", want)
		}
	}
}

// ---- actors_declaration_state_test.go ----

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
	_, err = BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"linux-x64",
		BuildOptions{Runtime: RuntimeSelfHost},
	)
	if err == nil {
		t.Fatalf("expected actor-state unsupported type diagnostic")
	}
	if !strings.Contains(
		err.Error(),
		("actor state field 'title' type 'str' is not supported; " +
			"supported actor state field types are Int, Bool, UInt8, " +
			"UInt16, and task.error"),
	) {
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
	`, ("actor state field 'title' type 'str' is not supported; " +
		"supported actor state field types are Int, Bool, UInt8, " +
		"UInt16, and task.error"))
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
`, ("actor state field 'raw' type 'ptr' is not supported; " +
		"supported actor state field types are Int, Bool, UInt8, " +
		"UInt16, and task.error"))
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
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors", "actors_decl_spawn.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "actors_decl_spawn"+tgt.ExeExt)
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{Runtime: RuntimeBuiltin},
	); err != nil {
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

// ---- actors_runtime_symbols_test.go ----

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
			if _, err := BuildFileWithStatsOpt(
				tt.src,
				objPath,
				tt.target,
				BuildOptions{Emit: EmitLibrary},
			); err != nil {
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
	if err := lowerpkg.VerifyFunc(dispatchFn); err != nil {
		t.Fatalf("dispatch verifier: %v", err)
	}

	for _, instr := range dispatchFn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == "__tetra_actor_state_store" {
			if instr.ArgSlots != 2 || instr.RetSlots != 1 {
				t.Fatalf(
					"state store ABI = args %d rets %d, want args 2 rets 1",
					instr.ArgSlots,
					instr.RetSlots,
				)
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

	err := linkNativeExecutable(
		filepath.Join(t.TempDir(), "out"),
		native,
		BuildOptions{},
		checked,
		nil,
		nil,
	)
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

// ---- actors_scheduler_targets_test.go ----

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
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
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
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf("program timed out; sleeping actors should wake in deterministic deadline order")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf(
			"exit code = %d, want deterministic deadline-order wake for actor sleepers",
			exitCode,
		)
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
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
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
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors", "actors_pingpong.tetra")
	results := map[RuntimeMode]struct {
		stdout string
		exit   int
	}{}
	for _, rt := range []RuntimeMode{RuntimeBuiltin, RuntimeSelfHost} {
		tmp := t.TempDir()
		outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
		if _, err := BuildFileWithStatsOpt(
			srcPath,
			outPath,
			tgt.Triple,
			BuildOptions{Runtime: rt},
		); err != nil {
			t.Fatalf("build runtime %d: %v", rt, err)
		}
		stdout, exitCode := runBinary(t, outPath)
		results[rt] = struct {
			stdout string
			exit   int
		}{stdout: stdout, exit: exitCode}
	}

	if results[RuntimeBuiltin] != results[RuntimeSelfHost] {
		t.Fatalf(
			"runtime parity mismatch: builtin=%#v selfhost=%#v",
			results[RuntimeBuiltin],
			results[RuntimeSelfHost],
		)
	}
}

func TestActorsPingPongBuildsSelfHostRuntimeForAllX64Targets(t *testing.T) {
	srcPath := filepath.Join("..", "examples", "actors", "actors_pingpong.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	tmp := t.TempDir()
	for _, triple := range []string{"linux-x64", "macos-x64", "windows-x64"} {
		t.Run(triple, func(t *testing.T) {
			tgt, err := ctarget.Parse(triple)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			outPath := filepath.Join(tmp, "actors_"+strings.ReplaceAll(triple, "-", "_")+tgt.ExeExt)
			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				triple,
				BuildOptions{Runtime: RuntimeSelfHost},
			); err != nil {
				t.Fatalf("build: %v", err)
			}
			if _, err := os.Stat(outPath); err != nil {
				t.Fatalf("missing output: %v", err)
			}
		})
	}
}

func TestActorsPingPongBuildsSelfHostRuntimeForX32(t *testing.T) {
	srcPath := filepath.Join("..", "examples", "actors", "actors_pingpong.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "actors_x32")
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"x32",
		BuildOptions{Runtime: RuntimeSelfHost, Jobs: 1},
	); err != nil {
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

// ---- actors_test.go ----

func TestActorsPingPongBuildAndRun(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors", "actors_pingpong.tetra")
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
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors", "actors_pingpong.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{Runtime: RuntimeBuiltin},
	); err != nil {
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
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors", "actors_pingpong.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{Runtime: RuntimeSelfHost},
	); err != nil {
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
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors", "actors_tagged_stress.tetra")
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
			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				tgt.Triple,
				BuildOptions{Runtime: tc.rt},
			); err != nil {
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
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{Runtime: RuntimeBuiltin},
		2*time.Second,
	)
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
		t.Fatalf(
			"exit code = %d, want 127 successful child actors before builtin capacity failure",
			exitCode,
		)
	}
}

// ---- actors_typed_messages_test.go ----

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
	if !strings.Contains(
		err.Error(),
		"cannot send borrowed view across actor boundary; use .copy()",
	) {
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
	tgt, ok := ctarget.Host()
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
		`"capacity": 682`,
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

func TestActorTransferScalarSendExplainReportIncludesCopyEvidence(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "actor_ping_pong_tetra.tetra")
	outPath := filepath.Join(tmp, "actor_ping_pong_tetra")
	src := []byte(`func pong() -> i32
uses actors:
    var v: i32 = core.recv()
    if v == 41:
        var _sent: i32 = core.send(core.sender(), 42)
        return 0
    return 1

func main() -> i32
uses actors:
    var p: actor = core.spawn("pong")
    var _sent: i32 = core.send(p, 41)
    var r: i32 = core.recv()
    if r == 42:
        return 0
    return 1
`)
	if err := os.WriteFile(srcPath, src, 0o644); err != nil {
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
	var report actorTransferReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode actor transfer report: %v\n%s", err, raw)
	}
	if report.Totals.Copy <= 0 || report.Totals.BytesCopied <= 0 {
		t.Fatalf("scalar actor transfer totals = %+v, want non-zero copy bytes\n%s", report.Totals, raw)
	}
	if report.Totals.Move != 0 || report.Totals.ZeroCopyMove != 0 {
		t.Fatalf("scalar actor transfer totals = %+v, want no move or zero-copy move", report.Totals)
	}
	var scalarRows int
	for _, row := range report.Sends {
		if row.Case != "core.send" {
			continue
		}
		scalarRows++
		if row.MessageType != "scalar:i32" ||
			row.PayloadType != "i32" ||
			row.Ownership != "copy" ||
			row.TransferMode != "copy" ||
			row.RuntimePath != "actor_mailbox_scalar_value_slot" ||
			row.BytesCopied <= 0 ||
			row.ZeroCopy ||
			row.ClaimLevel != "evidence_only" ||
			row.BoundaryScope != "local_scalar_mailbox" ||
			row.ProductionRuntimeValidated {
			t.Fatalf("scalar core.send actor transfer row = %+v", row)
		}
	}
	if scalarRows != 2 {
		t.Fatalf("scalar core.send rows = %d, want 2: %+v", scalarRows, report.Sends)
	}
}

func TestActorPingPongRuntimeCallBackendSidecarUsesRegisterPath(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "actor_ping_pong_tetra.tetra")
	outPath := filepath.Join(tmp, "actor_ping_pong_tetra")
	if err := os.WriteFile(srcPath, []byte(`func pong() -> i32
uses actors:
    var v: i32 = core.recv()
    if v == 41:
        var _sent: i32 = core.send(core.sender(), 42)
        return 0
    return 1

func main() -> i32
uses actors:
    var p: actor = core.spawn("pong")
    var _sent: i32 = core.send(p, 41)
    var r: i32 = core.recv()
    if r == 42:
        return 0
    return 1
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{
		Runtime: RuntimeBuiltin,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}
	raw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	var report backendReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode backend report: %v\n%s", err, raw)
	}
	rows := backendRowsByFunction(report.Functions)
	for _, tc := range []struct {
		fn       string
		wantPath string
	}{
		{fn: "pong", wantPath: "machine-ir-actor-ping-pong-pong"},
		{fn: "main", wantPath: "machine-ir-actor-ping-pong-main"},
	} {
		row := rows[tc.fn]
		if row.BackendPath != "register" || row.Category != "register_path" ||
			row.Reason != "eligible_machine_ir_subset" {
			t.Fatalf(
				("%s backend row = %+v, want register_path; target=%s mode=%s " +
					"machine=%+v\n%s"),
				tc.fn,
				row,
				report.Target,
				report.Mode,
				report.MachineFunctions,
				raw,
			)
		}
		if row.Detail != tc.wantPath {
			t.Fatalf(
				"%s backend detail = %q, want %s; row=%+v\n%s",
				tc.fn,
				row.Detail,
				tc.wantPath,
				row,
				raw,
			)
		}
		assertStringSliceEqual(
			t,
			tc.fn+" runtime_features_required",
			row.RuntimeFeaturesRequired,
			[]string{"actor_runtime"},
		)
		assertStringSliceEqual(
			t,
			tc.fn+" runtime_features_linked",
			row.RuntimeFeaturesLinked,
			[]string{"actor_runtime"},
		)
		assertStringSliceEqual(
			t,
			tc.fn+" runtime_features_initialized",
			row.RuntimeFeaturesInitialized,
			[]string{"actor_runtime"},
		)
	}
	if report.Summary.RegisterPath != 2 || report.Summary.StackFallback != 0 ||
		report.Summary.Categories["register_path"] != 2 ||
		report.Summary.Categories["unsupported_effect_runtime_call"] != 0 {
		t.Fatalf(
			("actor ping-pong backend summary = %+v, want two register paths " +
				"and no unsupported runtime fallback\n%s"),
			report.Summary,
			raw,
		)
	}
	assertStringSliceEqual(
		t,
		"summary runtime_features_required",
		report.Summary.RuntimeFeaturesRequired,
		[]string{"actor_runtime"},
	)
	assertStringSliceEqual(
		t,
		"summary runtime_features_linked",
		report.Summary.RuntimeFeaturesLinked,
		[]string{"actor_runtime"},
	)
	assertStringSliceEqual(
		t,
		"summary runtime_features_initialized",
		report.Summary.RuntimeFeaturesInitialized,
		[]string{"actor_runtime"},
	)
	if !report.Summary.RuntimeObjectPlan.RuntimeObjectLinked ||
		!report.Summary.RuntimeObjectPlan.RuntimeObjectInitialized ||
		!containsReportString(
			report.Summary.RuntimeObjectPlan.RuntimeObjectFeaturesRequired,
			"actor_runtime",
		) ||
		!containsReportString(
			report.Summary.RuntimeObjectPlan.RuntimeObjectFeaturesLinked,
			"actor_runtime",
		) ||
		!containsReportString(
			report.Summary.RuntimeObjectPlan.RuntimeObjectFeaturesInitialized,
			"actor_runtime",
		) {
		t.Fatalf("actor runtime object evidence missing: %+v\n%s", report.Summary.RuntimeObjectPlan, raw)
	}
}

func TestActorsTypedPayloadBuildAndRunWithBothRuntimes(t *testing.T) {
	_, ok := ctarget.Host()
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

// ---- atomic_suite_test.go ----

func TestRunTargetAtomicStressChecksCoversX86AndX64Family(t *testing.T) {
	tests := []struct {
		target string
		names  []string
	}{
		{
			target: "x86",
			names: []string{
				"x86 atomic validation matrix",
				"x86 atomic object matrix",
				"x86 pointer atomic object width",
				"x86 atomic concurrency stress oracle",
				"x86 atomic diagnostics",
			},
		},
		{
			target: "x64",
			names: []string{
				"x64 atomic validation matrix",
				"x64 atomic object matrix",
				"x64 pointer atomic object width",
				"x64 atomic concurrency stress oracle",
				"x64 atomic diagnostics",
			},
		},
		{
			target: "windows-x64",
			names: []string{
				"windows-x64 atomic validation matrix",
				"windows-x64 atomic object matrix",
				"windows-x64 pointer atomic object width",
				"windows-x64 atomic concurrency stress oracle",
				"windows-x64 atomic diagnostics",
			},
		},
		{
			target: "macos-x64",
			names: []string{
				"macos-x64 atomic validation matrix",
				"macos-x64 atomic object matrix",
				"macos-x64 pointer atomic object width",
				"macos-x64 atomic concurrency stress oracle",
				"macos-x64 atomic diagnostics",
			},
		},
		{
			target: "x32",
			names: []string{
				"x32 atomic validation matrix",
				"x32 atomic object matrix",
				"x32 pointer atomic object width",
				"x32 atomic concurrency stress oracle",
				"x32 atomic diagnostics",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			checks, err := RunTargetAtomicStressChecks(tt.target)
			if err != nil {
				t.Fatalf("RunTargetAtomicStressChecks(%s): %v", tt.target, err)
			}
			if len(checks) != len(tt.names) {
				t.Fatalf("checks = %#v, want %d checks", checks, len(tt.names))
			}
			for i, want := range tt.names {
				if checks[i].Name != want {
					t.Fatalf("check[%d] = %#v, want name %q", i, checks[i], want)
				}
				if checks[i].Error != "" {
					t.Fatalf("check[%d] = %#v, want passing %q", i, checks[i], want)
				}
			}
		})
	}
}

func TestAtomicStressIterationsEnvOverride(t *testing.T) {
	t.Setenv("TETRA_ATOMIC_STRESS_ITERS", "7")
	got, err := atomicStressIterations()
	if err != nil {
		t.Fatalf("atomicStressIterations: %v", err)
	}
	if got != 7 {
		t.Fatalf("iterations = %d, want 7", got)
	}
}

func TestAtomicStressIterationsRejectsInvalidEnv(t *testing.T) {
	for _, raw := range []string{"0", "-1", "abc", "100001"} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("TETRA_ATOMIC_STRESS_ITERS", raw)
			if got, err := atomicStressIterations(); err == nil {
				t.Fatalf("atomicStressIterations() = %d, want error", got)
			}
		})
	}
}

// ---- atomic_target_diagnostics_test.go ----

func TestAtomicIRTargetInfoUsesX32PointerWidth(t *testing.T) {
	tgt, err := ctarget.Parse("linux-x32")
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	info, ok := atomicIRTargetInfo(ir.IRAtomicLoadPtr, tgt)
	if !ok {
		t.Fatalf("IRAtomicLoadPtr was not classified as a target atomic")
	}
	if info.widthBits != tgt.PointerWidthBits {
		t.Fatalf(
			"x32 pointer atomic width = %d, want pointer width %d",
			info.widthBits,
			tgt.PointerWidthBits,
		)
	}
	if info.widthBits == tgt.RegisterWidthBits {
		t.Fatalf(
			"x32 pointer atomic width followed register width %d instead of pointer width",
			tgt.RegisterWidthBits,
		)
	}
}

func TestX86RejectsI64AtomicWithTargetDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "atomic_i64_x86.tetra")
	outPath := filepath.Join(tmp, "atomic_i64_x86.tobj")
	if err := os.WriteFile(srcPath, []byte(`
func atomic_i64_probe() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let loaded: i64 = core.atomic_load_i64_acquire(p, mem)
        return 0
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"x86",
		BuildOptions{Emit: EmitLibrary, Jobs: 1},
	)
	if err == nil {
		t.Fatalf("expected x86 i64 atomic target diagnostic")
	}
	diag := DiagnosticFromError(err)
	if diag.Code != DiagnosticCodeTargetRuntime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v", diag)
	}
	for _, want := range []string{
		"linux-x86",
		"atomic load",
		"64-bit",
		"unsupported atomic width 64 bits",
	} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("message = %q, want substring %q", diag.Message, want)
		}
	}
	if !strings.Contains(diag.Hint, "Use 8/16/32-bit or pointer atomics on linux-x86") {
		t.Fatalf("hint = %q, want x86 atomic-width guidance", diag.Hint)
	}
	if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
		t.Fatalf("x86 i64 atomic rejection wrote object %s, stat error = %v", outPath, statErr)
	}
}

// ---- compatibility_stability_v1_test.go ----

func TestP24CompatibilityStabilityV1CoversMasterPlanTargets(t *testing.T) {
	report, err := BuildP24CompatibilityStabilityV1Report()
	if err != nil {
		t.Fatalf("BuildP24CompatibilityStabilityV1Report: %v", err)
	}
	if report.SchemaVersion != compatibilityStabilityV1Schema {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, compatibilityStabilityV1Schema)
	}
	if report.Scope != compatibilityStabilityV1ScopeP242 {
		t.Fatalf("scope = %q, want %q", report.Scope, compatibilityStabilityV1ScopeP242)
	}
	if err := ValidateP24CompatibilityStabilityV1Report(report); err != nil {
		t.Fatalf("ValidateP24CompatibilityStabilityV1Report: %v", err)
	}

	rows := map[CompatibilityStabilityV1ID]CompatibilityStabilityV1Row{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 ||
			len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p24CompatibilityStabilityV1IDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p24AssertCompatibilityStabilityRow(
		t,
		rows[CompatibilityStableDiagnosticCodes],
		[]string{"DiagnosticCodeRegistry", "TETRA0001", "TETRA2001", "validate-diagnostic"},
	)
	p24AssertCompatibilityStabilityRow(
		t,
		rows[CompatibilityVersionedReportSchemas],
		[]string{"tetra.translation.validation.v2", "tetra.runtime.hardening.v1", "schema_version"},
	)
	p24AssertCompatibilityStabilityRow(
		t,
		rows[CompatibilityManifestChecks],
		[]string{"validate-manifest", "manifest-json.v1", "FeatureRegistry", "runtime ABI"},
	)
	p24AssertCompatibilityStabilityRow(
		t,
		rows[CompatibilityBreakingChangeMigrationGuide],
		[]string{"breaking_requires_review", "migration guide", "no-change"},
	)
	p24AssertCompatibilityStabilityRow(
		t,
		rows[CompatibilityDeprecationPolicy],
		[]string{"Deprecation Policy", "replacement path", "removals wait"},
	)

	if !report.StableDiagnosticCodesReviewed || !report.VersionedReportSchemasReviewed ||
		!report.ManifestCompatibilityChecksReviewed ||
		!report.BreakingChangeMigrationGuidePresent ||
		!report.DeprecationPolicyPresent {
		t.Fatalf("compatibility/stability flags missing: %#v", report)
	}
	for _, nonClaim := range []string{
		"full backward compatibility for all future versions is not claimed",
		"diagnostic messages are not frozen",
		"automatic migration for every breaking change is not claimed",
		"manifest/runtime ABI stability beyond current validated evidence is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p24CompatibilityStabilityHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP24CompatibilityStabilityV1RejectsFakeClaimsAndWeakEvidence(t *testing.T) {
	base, err := BuildP24CompatibilityStabilityV1Report()
	if err != nil {
		t.Fatalf("BuildP24CompatibilityStabilityV1Report: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*CompatibilityStabilityV1Report)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "missing diagnostics",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.StableDiagnosticCodesReviewed = false
			},
			want: "diagnostic",
		},
		{
			name: "missing manifest checks",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.ManifestCompatibilityChecksReviewed = false
			},
			want: "manifest",
		},
		{
			name: "fake full backward compatibility",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.FullBackwardCompatibilityClaimed = true
			},
			want: "full backward compatibility",
		},
		{
			name: "fake frozen diagnostics",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.FrozenDiagnosticMessagesClaimed = true
			},
			want: "diagnostic messages",
		},
		{
			name: "fake automatic migration",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.AutomaticMigrationClaimed = true
			},
			want: "automatic migration",
		},
		{
			name: "fake manifest abi stability",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.ManifestABIStabilityClaimed = true
			},
			want: "manifest/runtime ABI",
		},
		{
			name: "breaking change without migration",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.BreakingChangesWithoutMigrationClaimed = true
			},
			want: "breaking change",
		},
		{
			name: "removal without deprecation",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.RemovalWithoutDeprecationClaimed = true
			},
			want: "deprecation",
		},
		{
			name: "runtime behavior change",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics change",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.SafeSemanticsChanged = true
			},
			want: "safe semantics",
		},
		{
			name: "performance claim",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := base
			report.Rows = append([]CompatibilityStabilityV1Row(nil), base.Rows...)
			report.Witnesses = append([]CompatibilityStabilityV1Witness(nil), base.Witnesses...)
			report.NonClaims = append([]string(nil), base.NonClaims...)
			tc.mutate(&report)
			err := ValidateP24CompatibilityStabilityV1Report(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf(
					"ValidateP24CompatibilityStabilityV1Report error = %v, want %q",
					err,
					tc.want,
				)
			}
		})
	}
}

func p24AssertCompatibilityStabilityRow(
	t *testing.T,
	row CompatibilityStabilityV1Row,
	wants []string,
) {
	t.Helper()
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}

// ---- compiler_build_helpers_test.go ----

func buildAndRunFile(t *testing.T, srcPath string) (string, int) {
	t.Helper()

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "app")
	if err := BuildFile(srcPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func buildAndRunFileWithOptions(t *testing.T, srcPath string, opt BuildOptions) (string, int) {
	t.Helper()

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "app")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", opt); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func projectRoot(t *testing.T) string {
	t.Helper()
	// Walk up from the test binary's working directory to find the project root.
	// The go test framework runs in the package dir, so we go up from compiler/.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// wd is .../compiler, project root is parent
	return filepath.Dir(wd)
}

func buildAndRun(t *testing.T, src string) (string, int) {
	t.Helper()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")

	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := BuildFile(srcPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func buildAndRunWithOptions(t *testing.T, src string, opt BuildOptions) (string, int) {
	t.Helper()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")

	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", opt); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func buildOnly(t *testing.T, src string) error {
	t.Helper()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")

	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	return BuildFile(srcPath, outPath, "linux-x64")
}

func buildAndRunFiles(t *testing.T, files map[string]string, entry string) (string, int) {
	t.Helper()

	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)

	entryPath := filepath.Join(tmp, filepath.FromSlash(entry))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := BuildFile(entryPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	return runBinary(t, outPath)
}

func buildOnlyFiles(t *testing.T, files map[string]string, entry string) error {
	t.Helper()

	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)

	entryPath := filepath.Join(tmp, filepath.FromSlash(entry))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	return BuildFile(entryPath, outPath, "linux-x64")
}

func writeTestFiles(t *testing.T, base string, files map[string]string) {
	t.Helper()

	for path, src := range files {
		full := filepath.Join(base, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatalf("write source: %v", err)
		}
	}
}

func runBinary(t *testing.T, path string) (string, int) {
	t.Helper()

	cmd := exec.Command(path)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return out.String(), exitErr.ProcessState.ExitCode()
		}
		t.Fatalf("run binary: %v", err)
	}
	return out.String(), cmd.ProcessState.ExitCode()
}

func verifyELF(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	hdr := make([]byte, 64)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return err
	}
	if !bytes.Equal(hdr[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		return fmt.Errorf("missing ELF magic")
	}
	if hdr[4] != 2 {
		return fmt.Errorf("expected ELF64")
	}
	if hdr[5] != 1 {
		return fmt.Errorf("expected little-endian")
	}
	eType := binary.LittleEndian.Uint16(hdr[16:18])
	eMachine := binary.LittleEndian.Uint16(hdr[18:20])
	entry := binary.LittleEndian.Uint64(hdr[24:32])
	if eType != 2 {
		return fmt.Errorf("expected ET_EXEC")
	}
	if eMachine != 0x3e {
		return fmt.Errorf("expected x86_64 machine")
	}
	if entry == 0 {
		return fmt.Errorf("entrypoint is zero")
	}
	return nil
}

// ---- compiler_core_runtime_test.go ----

func TestBuildCoreSerializationSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "data", "core_serialization_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreFilesystemSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "platform", "core_filesystem_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreNetworkingSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "platform", "core_networking_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreNetSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "platform", "core_net_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreJSONSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "data", "core_json_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreHTTPSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "platform", "core_http_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCorePostgresSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "runtime", "core_postgres_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCorePostgresPreparedSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "runtime", "core_postgres_prepared_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCorePostgresResultSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "runtime", "core_postgres_result_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreAsyncSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "async", "core_async_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreSyncSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "runtime", "core_sync_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreTimeSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "platform", "core_time_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreCryptoSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "memory", "core_crypto_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildExtensionSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "smoke", "language", "extension_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildGenericSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "smoke", "language", "generic_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestCoreV015SemanticDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "bool from int",
			src:  "func main() -> Int:\n  let x: Bool = 1\n  return 0\n",
			want: "type mismatch: expected 'bool', got 'i32'",
		},
		{
			name: "int from bool",
			src:  "func main() -> Int:\n  let x: Int = true\n  return x\n",
			want: "type mismatch: expected 'i32', got 'bool'",
		},
		{
			name: "duplicate enum case",
			src:  "enum Color:\n  case red\n  case red\nfunc main() -> Int:\n  return 0\n",
			want: "duplicate enum case 'red'",
		},
		{
			name: "unknown enum case",
			src:  "enum Color:\n  case red\nfunc main() -> Int:\n  let c: Color = Color.blue\n  return 0\n",
			want: "unknown enum case 'blue'",
		},
		{
			name: "compare different enums",
			src: ("enum A:\n  case one\nenum B:\n  case one\nfunc main() -> Int:\n  " +
				"let a: A = A.one\n  let b: B = B.one\n  if a == b:\n    return " +
				"1\n  return 0\n"),
			want: "cannot compare 'A' and 'B'",
		},
		{
			name: "invalid match pattern",
			src: ("enum Color:\n  case red\nfunc main() -> Int:\n  let c: Color = " +
				"Color.red\n  match c:\n  case 1:\n    return 1\n  return 0\n"),
			want: "match pattern type mismatch",
		},
		{
			name: "multiple defaults",
			src: ("func main() -> Int:\n  match 1:\n  case _:\n    return 1\n  " +
				"case _:\n    return 2\n  return 0\n"),
			want: "match default must be last",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := buildOnly(t, tt.src)
			if err == nil {
				t.Fatalf("expected build error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestBuildFlowStructSyntax(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("struct Vec2:\n  x: Int\n  y: Int\n\nfunc sum(v: Vec2) -> Int:\n  " +
		"return v.x + v.y\n\nfunc main() -> Int:\n  let v: Vec2 = " +
		"Vec2(x: 40, y: 2)\n  return sum(v)\n")
	_, exitCode := buildAndRun(t, src)
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFlowIslandSyntax(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("func main() -> Int\nuses alloc, islands, io, mem:\n  " +
		"island(64) as isl:\n    var msg: []UInt8 = " +
		"core.island_make_u8(isl, 2)\n    msg[0] = 79\n    msg[1] = " +
		"10\n    print(msg)\n  return 0\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "O\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFlowUnsafeCapMemSyntax(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("func main() -> Int\nuses alloc, capability, mem:\n  var out: " +
		"Int = 1\n  unsafe:\n    let mem: cap.mem = core.cap_mem()\n    " +
		"let p: ptr = core.alloc_bytes(4)\n    let _: Int = " +
		"core.store_i32(p, 42, mem)\n    out = core.load_i32(p, mem)\n " +
		" return out\n")
	_, exitCode := buildAndRun(t, src)
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildBudgetedUnsafeCallsPreserveIRStack(t *testing.T) {
	src := `func main() -> Int
uses alloc, budget, capability, mem
budget(16):
    var out: Int = 1
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let _: Int = core.store_i32(p, 42, mem)
        out = core.load_i32(p, mem)
    return out
`
	if err := buildOnly(t, src); err != nil {
		t.Fatalf("BuildFile: %v", err)
	}
}

func TestBuildBudgetRuntimeGuardAllowsAndFailsDeterministically(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	okSrc := `func tick() -> Int
uses budget
budget(1):
    return 9

func main() -> Int
uses budget
budget(4):
    return tick()
`
	stdout, exitCode := buildAndRun(t, okSrc)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 9 {
		t.Fatalf("exit code = %d, want 9", exitCode)
	}

	failSrc := `func tick() -> Int
uses budget
budget(1):
    return 9

func main() -> Int
uses budget
budget(0):
    return tick()
`
	err := buildOnly(t, failSrc)
	if err == nil {
		t.Fatalf("expected compile-time budget context rejection")
	}
	if !strings.Contains(
		err.Error(),
		"budget context for call to 'tick' requires caller budget at least 1, got 0",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildBudgetFailureABIReturnAndThrowShapes(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tests := []struct {
		name     string
		src      string
		wantExit int
	}{
		{
			name: "non throwing multi slot return defaults to zero slots",
			src: `struct Pair:
    x: Int
    y: Int

func one() -> Int:
    return 7

func pair() -> Pair
uses budget
budget(0):
    return Pair(x: one(), y: 8)

func main() -> Int
uses budget
budget(16):
    let p: Pair = pair()
    return p.x + p.y
`,
			wantExit: 0,
		},
		{
			name: "throwing compact result returns thrown default payload",
			src: `enum BudgetTrap:
    case exhausted
    case other

func one() -> Int:
    return 99

func guarded() -> Int throws BudgetTrap
uses budget
budget(0):
    return one()

func main() -> Int
uses budget
budget(16):
    return catch guarded():
    case BudgetTrap.exhausted:
        21
    case BudgetTrap.other:
        22
`,
			wantExit: 21,
		},
		{
			name: "throwing non compact result returns thrown zero payload",
			src: `enum BudgetTrap:
    case exhausted(Int)
    case other(Int)

func one() -> Int:
    return 99

func guarded() -> Int throws BudgetTrap
uses budget
budget(0):
    return one()

func main() -> Int
uses budget
budget(16):
    return catch guarded():
    case BudgetTrap.exhausted(code):
        30 + code
    case BudgetTrap.other(otherCode):
        40 + otherCode
`,
			wantExit: 30,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, exitCode := buildAndRun(t, tt.src)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != tt.wantExit {
				t.Fatalf("exit code = %d, want %d", exitCode, tt.wantExit)
			}
		})
	}
}

func TestBuildPrivacyConsentRuntimeSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func seal(token: consent.token) -> secret.i32
uses privacy
privacy
consent(token):
    return core.secret_seal_i32(33, token)

func reveal(token: consent.token, value: secret.i32) -> Int
uses privacy
privacy
consent(token):
    return core.secret_unseal_i32(value, token)

func main() -> Int
uses privacy
privacy:
    let token: consent.token = core.consent_token()
    let secret: secret.i32 = seal(token)
    return reveal(token, secret)
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 33 {
		t.Fatalf("exit code = %d, want 33", exitCode)
	}
}

func TestBuildPrivacySealUnsealStaticOnlyDeterministicIdentity(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func roundtrip(token: consent.token, value: Int) -> Int
uses privacy
privacy
consent(token):
    let sealed: secret.i32 = core.secret_seal_i32(value, token)
    return core.secret_unseal_i32(sealed, token)

func main() -> Int
uses privacy
privacy:
    let token: consent.token = core.consent_token()
    let first: Int = roundtrip(token, 17)
    let second: Int = roundtrip(token, 17)
    let third: Int = roundtrip(token, 9)
    return (first - second) + third
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 9 {
		t.Fatalf("exit code = %d, want 9", exitCode)
	}
}

// ---- compiler_examples_microservices_test.go ----

func TestExampleHello(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "smoke", "basic", "hello.tetra"),
	)
	if stdout != "Hello from Tetra!\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestExampleGlobalsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "smoke", "language", "globals_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	// g_x = g_y + 2 = 40 + 2 = 42, store 7 at g_p, out = 7 + 42 = 49
	if exitCode != 49 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestExampleStructCtorSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "smoke", "language", "struct_ctor_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	// v.x + v.y + 52 = 40 + 2 + 52 = 94
	if exitCode != 94 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestExampleExperimentalMath(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "experimental", "experimental_math_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	// math.add_i32(40, 2) = 42
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestExampleExperimentalMemcpy(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "experimental", "experimental_memcpy_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 93 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestExampleCapMemPtr(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "memory", "raw", "cap_mem_ptr_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 77 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestExampleCapMemPtrAddLocal(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "memory", "raw", "cap_mem_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 77 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestMicroserviceExamplesAndBugLedger(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	for _, name := range []string{
		filepath.FromSlash("business/services/inventory_service.tetra"),
		filepath.FromSlash("business/services/payments_service.tetra"),
		filepath.FromSlash("business/services/orders_gateway.tetra"),
		filepath.FromSlash("memory/misc/state/memory_cache_service.tetra"),
		filepath.FromSlash("parallel/defer_fanout/parallel_fanout_service.tetra"),
		filepath.FromSlash("compiler/pipeline/compiler_pipeline_service.tetra"),
		filepath.FromSlash("memory/islands/core/island_cache_pool_service.tetra"),
		filepath.FromSlash("parallel/task_core/parallel_task_pool_service.tetra"),
		filepath.FromSlash("compiler/pipeline/compiler_artifact_router_service.tetra"),
		filepath.FromSlash("memory/misc/state/memory_journal_service.tetra"),
		filepath.FromSlash("parallel/tasks/group_core/task_group_service.tetra"),
		filepath.FromSlash("parallel/tasks/typed/typed_task_error_service.tetra"),
		filepath.FromSlash("parallel/tasks/group_cancel/task_group_cancel_service.tetra"),
		filepath.FromSlash("parallel/tasks/typed/wait_select_service.tetra"),
		filepath.FromSlash("memory/misc/state/memory_bounds_probe_service.tetra"),
		filepath.FromSlash("compiler/callables/callable_router_service.tetra"),
		filepath.FromSlash("compiler_modular_gateway/app/main.tetra"),
		filepath.FromSlash("memory/islands/core/island_slice_matrix_service.tetra"),
		filepath.FromSlash("compiler/optionals/generic_optional_router_service.tetra"),
		filepath.FromSlash("actor/core/actor_deadline_router_service.tetra"),
		filepath.FromSlash("parallel/tasks/typed/typed_task_success_service.tetra"),
		filepath.FromSlash("memory/misc/state/memory_byte_window_service.tetra"),
		filepath.FromSlash("compiler/callables/callable_return_router_service.tetra"),
		filepath.FromSlash("compiler_callable_pack/app/main.tetra"),
		filepath.FromSlash("actor/core/actor_tagged_loop_service.tetra"),
		filepath.FromSlash("parallel/tasks/group_core/task_group_lifecycle_service.tetra"),
		filepath.FromSlash("memory/misc/ops/memory_negative_guard_service.tetra"),
		filepath.FromSlash("compiler/callables/callable_identity_router_service.tetra"),
		filepath.FromSlash("compiler_throwing_callable_pack/app/main.tetra"),
		filepath.FromSlash("actor/timers/actor_poll_timeout_service.tetra"),
		filepath.FromSlash("parallel/tasks/deadlines/task_timeout_recovery_service.tetra"),
		filepath.FromSlash("memory/scalars/loops/memory_u16_lane_service.tetra"),
		filepath.FromSlash("compiler/generics/generic_struct_router_service.tetra"),
		filepath.FromSlash("compiler_generic_box_pack/app/main.tetra"),
		filepath.FromSlash("parallel/tasks/group_core/task_group_payload_service.tetra"),
		filepath.FromSlash("actor/core/actor_sender_snapshot_service.tetra"),
		filepath.FromSlash("memory/misc/state/memory_copy_window_service.tetra"),
		filepath.FromSlash("compiler/protocols/protocol_bound_generic_service.tetra"),
		filepath.FromSlash("compiler_protocol_pack/app/main.tetra"),
		filepath.FromSlash("actor/state/actor_state_counter_service.tetra"),
		filepath.FromSlash("parallel/tasks/group_cancel/task_group_self_cancel_service.tetra"),
		filepath.FromSlash("compiler/generics/generic_typed_error_service.tetra"),
		filepath.FromSlash("compiler_generic_error_pack/app/main.tetra"),
		filepath.FromSlash("parallel/tasks/group_cancel/task_group_current_status_service.tetra"),
		filepath.FromSlash("actor/mailbox/actor_dual_mailbox_service.tetra"),
		filepath.FromSlash("memory/misc/ops/memory_memset_stride_service.tetra"),
		filepath.FromSlash("memory/islands/core/island_bool_flags_service.tetra"),
		filepath.FromSlash("compiler_generic_pair_pack/app/main.tetra"),
		filepath.FromSlash("actor/mailbox/actor_dual_value_mailbox_service.tetra"),
		filepath.FromSlash("parallel/tasks/deadlines/task_dual_deadline_service.tetra"),
		filepath.FromSlash("memory/misc/ops/memory_zero_copy_service.tetra"),
		filepath.FromSlash("compiler_optional_box_pack/app/main.tetra"),
		filepath.FromSlash("actor/timers/actor_timeout_retry_service.tetra"),
		filepath.FromSlash("parallel/tasks/deadlines/task_poll_deadline_matrix_service.tetra"),
		filepath.FromSlash("memory/base_ptrs/helpers/memory_ptr_table_service.tetra"),
		filepath.FromSlash("compiler/optionals/optional_enum_router_service.tetra"),
		filepath.FromSlash("compiler/optionals/optional_field_update_service.tetra"),
		filepath.FromSlash("actor/core/actor_chain_reply_service.tetra"),
		filepath.FromSlash("parallel/tasks/group_core/task_group_poll_service.tetra"),
		filepath.FromSlash("memory/scalars/loops/memory_i32_stride_service.tetra"),
		filepath.FromSlash("actor/core/actor_value_chain_service.tetra"),
		filepath.FromSlash("parallel/tasks/group_core/task_group_typed_success_service.tetra"),
		filepath.FromSlash("memory/base_ptrs/helpers/memory_chained_ptr_stride_service.tetra"),
		filepath.FromSlash("compiler_optional_enum_pack/app/main.tetra"),
		filepath.FromSlash("actor/typed/actor_typed_payload_service.tetra"),
		filepath.FromSlash("parallel/tasks/deadlines/task_select_timeout_service.tetra"),
		filepath.FromSlash("memory/scalars/loops/memory_mixed_width_service.tetra"),
		filepath.FromSlash("compiler_extension_pack/app/main.tetra"),
		filepath.FromSlash("actor/mailbox/actor_self_mailbox_service.tetra"),
		filepath.FromSlash("parallel/tasks/group_cancel/task_group_cancel_after_spawn_service.tetra"),
		filepath.FromSlash("memory/base_ptrs/derived/memory_derived_copy_service.tetra"),
		filepath.FromSlash("compiler_protocol_extension_pack/app/main.tetra"),
		filepath.FromSlash("actor/typed/actor_typed_chain_service.tetra"),
		filepath.FromSlash("parallel/tasks/group_cancel/task_group_multi_cancel_service.tetra"),
		filepath.FromSlash("memory/base_ptrs/derived/memory_derived_ptr_table_service.tetra"),
		filepath.FromSlash("compiler_generic_function_pack/app/main.tetra"),
		filepath.FromSlash("actor/mailbox/actor_self_typed_mailbox_service.tetra"),
		filepath.FromSlash("actor/tasks/actor_task_bridge_service.tetra"),
		filepath.FromSlash("memory/base_ptrs/core/memory_aggregate_ptr_service.tetra"),
		filepath.FromSlash("compiler/generics/compiler_generic_extension_local_service.tetra"),
		filepath.FromSlash("actor/typed_tasks/actor_typed_task_bridge_service.tetra"),
		filepath.FromSlash("parallel/tasks/group_core/task_group_actor_fanout_service.tetra"),
		filepath.FromSlash("memory/optional/pointers/memory_optional_ptr_service.tetra"),
		filepath.FromSlash("compiler/callables/compiler_callable_generic_route_service.tetra"),
		filepath.FromSlash("parallel/tasks/deadlines/task_actor_roundtrip_service.tetra"),
		filepath.FromSlash("actor/typed/actor_typed_task_group_service.tetra"),
		filepath.FromSlash("memory/base_ptrs/helpers/memory_function_ptr_service.tetra"),
		filepath.FromSlash("parallel/tasks/deadlines/task_typed_actor_roundtrip_service.tetra"),
		filepath.FromSlash("actor/tasks/actor_task_select_service.tetra"),
		filepath.FromSlash("compiler/optionals/compiler_generic_optional_route_service.tetra"),
		filepath.FromSlash("memory/misc/state/memory_global_state_service.tetra"),
		filepath.FromSlash("actor/typed_tasks/actor_typed_task_error_bridge_service.tetra"),
		filepath.FromSlash("actor/tasks/actor_task_cancel_select_service.tetra"),
		filepath.FromSlash("compiler_generic_optional_import_pack/app/main.tetra"),
		filepath.FromSlash("memory/misc/ops/memory_mutable_ptr_service.tetra"),
		filepath.FromSlash("memory/misc/offsets/memory_struct_offset_service.tetra"),
		filepath.FromSlash("actor/tasks/actor_task_recovery_service.tetra"),
		filepath.FromSlash("compiler_generic_nested_optional_pack/app/main.tetra"),
		filepath.FromSlash("memory/misc/offsets/memory_function_offset_service.tetra"),
		filepath.FromSlash("memory/misc/offsets/memory_expression_offset_service.tetra"),
		filepath.FromSlash("actor/timers/actor_timer_task_matrix_service.tetra"),
		filepath.FromSlash("compiler_generic_enum_import_pack/app/main.tetra"),
		filepath.FromSlash("memory/tasks/results/memory_task_result_offset_service.tetra"),
		filepath.FromSlash("memory/actor_offsets/core/memory_actor_message_offset_service.tetra"),
		filepath.FromSlash("memory/actor_offsets/core/memory_actor_recv_value_offset_service.tetra"),
		filepath.FromSlash("memory/actor_offsets/core/memory_actor_poll_value_offset_service.tetra"),
		filepath.FromSlash("memory/actor_offsets/typed/memory_actor_tag_offset_service.tetra"),
		filepath.FromSlash("compiler_actor_wait_memory_pack/app/main.tetra"),
		filepath.FromSlash("memory/actor_offsets/core/memory_actor_recv_error_offset_service.tetra"),
		filepath.FromSlash("memory/actor_offsets/core/memory_actor_poll_error_offset_service.tetra"),
		filepath.FromSlash("memory/actor_offsets/core/memory_actor_recv_msg_error_offset_service.tetra"),
		filepath.FromSlash("compiler_actor_error_memory_pack/app/main.tetra"),
		filepath.FromSlash("actor/tasks/actor_task_group_error_recovery_service.tetra"),
		filepath.FromSlash("compiler_generic_struct_field_pack/app/main.tetra"),
		filepath.FromSlash("memory/misc/offsets/memory_indexed_metadata_offset_service.tetra"),
		filepath.FromSlash("parallel/typed_task/parallel_typed_task_payload_handle_service.tetra"),
		filepath.FromSlash("actor/mailbox/actor_typed_dual_mailbox_service.tetra"),
		filepath.FromSlash("parallel/tasks/group_cancel/task_group_nested_service.tetra"),
		filepath.FromSlash("compiler_generic_optional_struct_pack/app/main.tetra"),
		filepath.FromSlash("memory/base_ptrs/core/memory_direct_base_offset_service.tetra"),
		filepath.FromSlash("parallel/typed_task/parallel_typed_task_wide_payload_service.tetra"),
		filepath.FromSlash("actor/typed/actor_typed_wide_payload_service.tetra"),
		filepath.FromSlash("compiler_cross_module_runtime_pack/app/main.tetra"),
		filepath.FromSlash("actor/typed/actor_typed_envelope_service.tetra"),
		filepath.FromSlash("parallel/task_core/parallel_time_window_service.tetra"),
		filepath.FromSlash("actor/state/actor_state_status_service.tetra"),
		filepath.FromSlash("memory/base_ptrs/derived/memory_inline_ptradd_window_service.tetra"),
		filepath.FromSlash("parallel/typed_task/parallel_typed_task_struct_payload_service.tetra"),
		filepath.FromSlash("actor/typed/actor_typed_struct_payload_service.tetra"),
		filepath.FromSlash("memory/base_ptrs/core/memory_callable_ptr_base_service.tetra"),
		filepath.FromSlash("memory/optional/pointers/memory_callable_optional_ptr_service.tetra"),
		filepath.FromSlash("compiler/protocols/compiler_match_ptr_base_service.tetra"),
		filepath.FromSlash("memory/typed_errors/core/memory_typed_error_ptr_base_service.tetra"),
		filepath.FromSlash("parallel/selection/parallel_join_until_rejoin_service.tetra"),
		filepath.FromSlash("actor/tasks/actor_task_result_window_service.tetra"),
		filepath.FromSlash("compiler/callables/compiler_inout_return_service.tetra"),
		filepath.FromSlash("memory/base_ptrs/core/memory_dynamic_base_offset_service.tetra"),
		filepath.FromSlash("parallel/group_cancel/parallel_group_close_before_join_service.tetra"),
		filepath.FromSlash("parallel/group_cancel/parallel_group_cancel_after_join_service.tetra"),
		filepath.FromSlash("parallel_cross_module_typed_task_pack/app/main.tetra"),
		filepath.FromSlash("memory/base_ptrs/derived/memory_struct_base_dynamic_service.tetra"),
		filepath.FromSlash("memory/base_ptrs/derived/memory_enum_base_dynamic_service.tetra"),
		filepath.FromSlash("memory/typed_errors/core/memory_typed_error_base_dynamic_service.tetra"),
		filepath.FromSlash("parallel/selection/parallel_select_recovery_service.tetra"),
		filepath.FromSlash("compiler/pipeline/compiler_pattern_binding_unique_service.tetra"),
		filepath.FromSlash("memory/base_ptrs/derived/memory_base_dynamic_copy_service.tetra"),
		filepath.FromSlash("parallel/selection/parallel_select_rejoin_service.tetra"),
		filepath.FromSlash("parallel/group_cancel/parallel_group_cancel_select_service.tetra"),
		filepath.FromSlash("compiler_interface_jobs_pack/app/main.tetra"),
		filepath.FromSlash("memory/base_ptrs/helpers/memory_zero_length_derived_helper_service.tetra"),
		filepath.FromSlash("parallel/group_cancel/parallel_group_spawn_after_cancel_service.tetra"),
		filepath.FromSlash("parallel/selection/parallel_join_until_poll_service.tetra"),
		filepath.FromSlash("compiler_interface_control_pack/app/main.tetra"),
		filepath.FromSlash("memory/base_ptrs/helpers/memory_zero_length_base_helper_service.tetra"),
		filepath.FromSlash("parallel/task_core/parallel_yield_join_window_service.tetra"),
		filepath.FromSlash("parallel/group_status/parallel_group_status_roundtrip_service.tetra"),
		filepath.FromSlash("memory/tasks/groups/memory_group_status_direct_offset_service.tetra"),
		filepath.FromSlash("memory/tasks/groups/memory_group_current_status_offset_service.tetra"),
		filepath.FromSlash("parallel/group_cancel/parallel_group_cancel_close_direct_service.tetra"),
		filepath.FromSlash("compiler_group_status_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_import_alias_pack/app/main.tetra"),
		filepath.FromSlash("memory/heap_slices/memory_heap_u16_slice_service.tetra"),
		filepath.FromSlash("memory/heap_slices/memory_heap_bool_flags_service.tetra"),
		filepath.FromSlash("parallel/actor_send/core/parallel_actor_yield_mailbox_service.tetra"),
		filepath.FromSlash("parallel/group_cancel/parallel_group_current_cancel_status_service.tetra"),
		filepath.FromSlash("compiler_cross_module_actor_pack/app/main.tetra"),
		filepath.FromSlash("memory/heap_slices/memory_heap_i32_bool_slice_service.tetra"),
		filepath.FromSlash("parallel/task_core/parallel_task_actor_deadline_service.tetra"),
		filepath.FromSlash("compiler_actor_resource_pack/app/main.tetra"),
		filepath.FromSlash("memory/heap_slices/memory_heap_u8_slice_service.tetra"),
		filepath.FromSlash("parallel/group_status/parallel_typed_group_cancel_status_service.tetra"),
		filepath.FromSlash("compiler_callable_return_pack/app/main.tetra"),
		filepath.FromSlash("compiler_callable_optional_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_interface_pack/app/main.tetra"),
		filepath.FromSlash("memory/slices/memory_slice_optional_service.tetra"),
		filepath.FromSlash("memory/slices/memory_slice_enum_service.tetra"),
		filepath.FromSlash("parallel/task_result/parallel_task_result_box_service.tetra"),
		filepath.FromSlash("parallel/task_result/parallel_task_result_enum_service.tetra"),
		filepath.FromSlash("compiler/pipeline/compiler_test_command_service.tetra"),
		filepath.FromSlash("parallel/task_result/parallel_task_test_command_service.tetra"),
		filepath.FromSlash("compiler/generics/generic_typed_result_payload_service.tetra"),
		filepath.FromSlash("memory/slices/memory_slice_struct_loop_service.tetra"),
		filepath.FromSlash("memory/slices/memory_slice_generic_box_service.tetra"),
		filepath.FromSlash("parallel/task_result/parallel_task_result_optional_service.tetra"),
		filepath.FromSlash("parallel/task_core/parallel_nested_task_spawn_service.tetra"),
		filepath.FromSlash("compiler_generic_slice_pack/app/main.tetra"),
		filepath.FromSlash("memory/slices/memory_slice_for_loop_service.tetra"),
		filepath.FromSlash("memory/slices/memory_slice_inout_mutation_service.tetra"),
		filepath.FromSlash("parallel/task_join/core/parallel_task_handle_optional_join_service.tetra"),
		filepath.FromSlash("compiler_optional_task_pack/app/main.tetra"),
		filepath.FromSlash("memory/scalars/loops/memory_bool_for_loop_service.tetra"),
		filepath.FromSlash("memory/scalars/loops/memory_i32_for_loop_service.tetra"),
		filepath.FromSlash("parallel/actor_send/core/parallel_actor_handle_optional_send_service.tetra"),
		filepath.FromSlash("compiler_optional_actor_pack/app/main.tetra"),
		filepath.FromSlash("memory/scalars/loops/memory_u16_for_loop_service.tetra"),
		filepath.FromSlash("parallel/group_optional/parallel_group_optional_close_service.tetra"),
		filepath.FromSlash("parallel/group_optional/parallel_group_optional_cancel_service.tetra"),
		filepath.FromSlash("compiler_optional_group_pack/app/main.tetra"),
		filepath.FromSlash("memory/scalars/inout/memory_bool_inout_toggle_service.tetra"),
		filepath.FromSlash("memory/scalars/inout/memory_i32_inout_fill_service.tetra"),
		filepath.FromSlash("parallel/group_optional/parallel_group_optional_match_close_service.tetra"),
		filepath.FromSlash("compiler_optional_group_match_pack/app/main.tetra"),
		filepath.FromSlash("parallel/group_core/parallel_group_struct_spawn_service.tetra"),
		filepath.FromSlash("parallel/group_core/parallel_group_enum_spawn_service.tetra"),
		filepath.FromSlash("parallel/group_core/parallel_group_typed_struct_spawn_service.tetra"),
		filepath.FromSlash("parallel/group_core/parallel_group_typed_enum_spawn_service.tetra"),
		filepath.FromSlash("memory/scalars/inout/memory_u16_inout_stride_service.tetra"),
		filepath.FromSlash("compiler_group_aggregate_pack/app/main.tetra"),
		filepath.FromSlash("parallel/group_core/parallel_group_alias_spawn_service.tetra"),
		filepath.FromSlash("parallel/group_core/parallel_group_generic_box_spawn_service.tetra"),
		filepath.FromSlash("memory/optional/boxes/memory_optional_generic_u16_box_service.tetra"),
		filepath.FromSlash("compiler_group_generic_pack/app/main.tetra"),
		filepath.FromSlash("parallel/task_join/core/parallel_task_alias_join_service.tetra"),
		filepath.FromSlash("parallel/task_join/core/parallel_task_generic_box_join_service.tetra"),
		filepath.FromSlash(("parallel/task_join/optional/parallel_task_optional_struct_bo" +
			"x_join_service.tetra")),
		filepath.FromSlash(("parallel/task_join/optional/parallel_task_optional_generic_b" +
			"ox_join_service.tetra")),
		filepath.FromSlash("memory/optional/boxes/memory_optional_generic_bool_box_service.tetra"),
		filepath.FromSlash("compiler_task_generic_pack/app/main.tetra"),
		filepath.FromSlash("parallel/actor_send/core/parallel_actor_alias_send_service.tetra"),
		filepath.FromSlash("parallel/actor_send/core/parallel_actor_generic_box_send_service.tetra"),
		filepath.FromSlash(("parallel/actor_send/optional/parallel_actor_optional_struct_" +
			"box_send_service.tetra")),
		filepath.FromSlash(("parallel/actor_send/optional/parallel_actor_optional_generic" +
			"_box_send_service.tetra")),
		filepath.FromSlash("memory/optional/boxes/memory_optional_generic_i32_box_service.tetra"),
		filepath.FromSlash("compiler_actor_generic_pack/app/main.tetra"),
		filepath.FromSlash("memory/islands/core/memory_island_alias_region_service.tetra"),
		filepath.FromSlash("memory/islands/core/memory_island_generic_box_region_service.tetra"),
		filepath.FromSlash("memory/islands/optional/memory_island_optional_struct_box_service.tetra"),
		filepath.FromSlash("memory/islands/optional/memory_island_optional_generic_box_service.tetra"),
		filepath.FromSlash("compiler_island_generic_pack/app/main.tetra"),
		filepath.FromSlash("memory/base_ptrs/core/memory_ptr_alias_base_service.tetra"),
		filepath.FromSlash("memory/base_ptrs/core/memory_ptr_generic_identity_base_service.tetra"),
		filepath.FromSlash("compiler_ptr_generic_pack/app/main.tetra"),
		filepath.FromSlash("memory/tasks/results/memory_task_result_optional_offset_service.tetra"),
		filepath.FromSlash("parallel/task_result/parallel_task_result_generic_box_service.tetra"),
		filepath.FromSlash("compiler_task_result_generic_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_optional_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_resource_wrapper_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_resource_wrapper_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_generic_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_generic_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_optional_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_generic_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_enum_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_slice_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_slice_lane_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_slice_shape_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_string_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_optional_generic_string_pack/app/main.tetra"),
		filepath.FromSlash("memory/optional/pointers/memory_ptr_generic_optional_field_service.tetra"),
		filepath.FromSlash(("parallel/task_result/parallel_task_result_generic_optional_f" +
			"ield_service.tetra")),
		filepath.FromSlash("compiler_generic_optional_field_pack/app/main.tetra"),
		filepath.FromSlash("memory/optional/pointers/memory_ptr_generic_optional_call_service.tetra"),
		filepath.FromSlash("memory/optional/pointers/memory_optional_ptr_inout_return_service.tetra"),
		filepath.FromSlash("compiler_ptr_optional_generic_call_pack/app/main.tetra"),
		filepath.FromSlash(("parallel/actor_send/optional/parallel_actor_optional_alias_s" +
			"end_service.tetra")),
		filepath.FromSlash("parallel/task_join/optional/parallel_task_optional_alias_join_service.tetra"),
		filepath.FromSlash("parallel/group_optional/parallel_group_optional_alias_close_service.tetra"),
		filepath.FromSlash("compiler_optional_alias_resource_pack/app/main.tetra"),
		filepath.FromSlash(("parallel/actor_send/optional/parallel_actor_optional_enum_se" +
			"nd_service.tetra")),
		filepath.FromSlash("parallel/task_join/optional/parallel_task_optional_enum_join_service.tetra"),
		filepath.FromSlash("parallel/group_optional/parallel_group_optional_enum_close_service.tetra"),
		filepath.FromSlash("memory/tasks/results/memory_task_result_optional_enum_offset_service.tetra"),
		filepath.FromSlash("compiler_optional_enum_resource_pack/app/main.tetra"),
		filepath.FromSlash(("parallel/actor_send/optional/parallel_actor_typed_optional_a" +
			"lias_send_service.tetra")),
		filepath.FromSlash(("parallel/group_optional/parallel_typed_group_optional_alias_" +
			"spawn_service.tetra")),
		filepath.FromSlash("compiler_typed_optional_alias_resource_pack/app/main.tetra"),
		filepath.FromSlash("parallel/typed_task/parallel_typed_task_match_catch_service.tetra"),
		filepath.FromSlash("memory/typed_errors/tasks/memory_typed_task_error_offset_service.tetra"),
		filepath.FromSlash("compiler_typed_task_match_pack/app/main.tetra"),
		filepath.FromSlash(("memory/typed_errors/tasks/memory_typed_task_error_struct_off" +
			"set_service.tetra")),
		filepath.FromSlash(("memory/typed_errors/tasks/memory_typed_task_error_nested_enu" +
			"m_offset_service.tetra")),
		filepath.FromSlash(("memory/typed_errors/tasks/memory_typed_task_error_optional_o" +
			"ffset_service.tetra")),
		filepath.FromSlash(("memory/typed_errors/tasks/memory_typed_task_error_guarded_of" +
			"fset_service.tetra")),
		filepath.FromSlash("compiler_typed_error_payload_memory_pack/app/main.tetra"),
		filepath.FromSlash("parallel/defer_fanout/parallel_defer_group_close_service.tetra"),
		filepath.FromSlash("memory/defer/memory_defer_store_service.tetra"),
		filepath.FromSlash("parallel/defer_fanout/parallel_defer_group_cancel_checkpoint_service.tetra"),
		filepath.FromSlash("memory/defer/memory_defer_task_result_offset_service.tetra"),
		filepath.FromSlash("compiler_defer_cleanup_pack/app/main.tetra"),
		filepath.FromSlash("memory/defer/memory_defer_throw_base_store_service.tetra"),
		filepath.FromSlash("memory/defer/memory_defer_return_base_store_service.tetra"),
		filepath.FromSlash("parallel/defer_fanout/parallel_typed_task_defer_actor_reply_service.tetra"),
		filepath.FromSlash("compiler_defer_unwind_pack/app/main.tetra"),
		filepath.FromSlash("memory/tasks/results/memory_join_until_result_offset_service.tetra"),
		filepath.FromSlash("memory/tasks/results/memory_poll_result_offset_service.tetra"),
		filepath.FromSlash("memory/tasks/results/memory_select_result_offset_service.tetra"),
		filepath.FromSlash("compiler_task_wait_memory_pack/app/main.tetra"),
		filepath.FromSlash("memory/tasks/errors/memory_join_until_error_offset_service.tetra"),
		filepath.FromSlash("memory/tasks/errors/memory_poll_error_offset_service.tetra"),
		filepath.FromSlash("memory/tasks/errors/memory_select_error_offset_service.tetra"),
		filepath.FromSlash("compiler_task_wait_error_memory_pack/app/main.tetra"),
		filepath.FromSlash("memory/typed_errors/core/memory_typed_error_optional_ptr_base_service.tetra"),
		filepath.FromSlash(("memory/typed_errors/core/memory_typed_error_optional_ptr_dyn" +
			"amic_service.tetra")),
		filepath.FromSlash("compiler_typed_error_optional_ptr_pack/app/main.tetra"),
		filepath.FromSlash("memory/actor_offsets/typed/memory_actor_typed_payload_offset_service.tetra"),
		filepath.FromSlash(("memory/actor_offsets/typed/memory_actor_typed_struct_payload" +
			"_offset_service.tetra")),
		filepath.FromSlash("compiler_typed_actor_payload_memory_pack/app/main.tetra"),
		filepath.FromSlash(("memory/actor_offsets/typed/memory_actor_typed_enum_payload_o" +
			"ffset_service.tetra")),
		filepath.FromSlash(("memory/actor_offsets/typed/memory_actor_typed_enum_struct_pa" +
			"yload_offset_service.tetra")),
		filepath.FromSlash("compiler_typed_actor_enum_payload_memory_pack/app/main.tetra"),
		filepath.FromSlash("backend/http/writer/backend_http_pipeline_gateway_service.tetra"),
		filepath.FromSlash("backend/postgres/protocol/backend_postgres_prepared_pipeline_service.tetra"),
		filepath.FromSlash("backend/net/backend_net_epoll_lifecycle_service.tetra"),
		filepath.FromSlash("backend/net/backend_net_epoll_event_bounds_guard_service.tetra"),
		filepath.FromSlash("backend/net/backend_net_port_bounds_guard_service.tetra"),
		filepath.FromSlash("backend/postgres/protocol/backend_postgres_result_guard_service.tetra"),
		filepath.FromSlash("backend/postgres/protocol/backend_postgres_session_state_service.tetra"),
		filepath.FromSlash(("backend/postgres/protocol/backend_postgres_cstring_bounds_gu" +
			"ard_service.tetra")),
		filepath.FromSlash("backend/postgres/protocol/backend_postgres_cstring_nul_guard_service.tetra"),
		filepath.FromSlash("backend/postgres/row/backend_postgres_data_row_length_guard_service.tetra"),
		filepath.FromSlash(("backend/postgres/ascii/backend_postgres_ascii_i32_bounds_gua" +
			"rd_service.tetra")),
		filepath.FromSlash("backend/postgres/ascii/backend_postgres_ascii_i32_min_guard_service.tetra"),
		filepath.FromSlash(("backend/postgres/command/backend_postgres_command_tag_bounds" +
			"_guard_service.tetra")),
		filepath.FromSlash(("backend/postgres/command/backend_postgres_command_tag_overfl" +
			"ow_guard_service.tetra")),
		filepath.FromSlash(("backend/postgres/command/backend_postgres_command_tag_traili" +
			"ng_guard_service.tetra")),
		filepath.FromSlash("backend/postgres/command/backend_postgres_parse_count_guard_service.tetra"),
		filepath.FromSlash("backend/postgres/read/backend_postgres_parser_short_guard_service.tetra"),
		filepath.FromSlash(("backend/postgres/row/backend_postgres_row_description_bounds" +
			"_guard_service.tetra")),
		filepath.FromSlash("backend/postgres/row/backend_postgres_data_row_bounds_guard_service.tetra"),
		filepath.FromSlash(("backend/postgres/row/backend_postgres_data_row_truncated_val" +
			"ue_guard_service.tetra")),
		filepath.FromSlash(("backend/postgres/frame/backend_postgres_frame_header_bounds_" +
			"guard_service.tetra")),
		filepath.FromSlash(("backend/postgres/frame/backend_postgres_frame_signed_length_" +
			"guard_service.tetra")),
		filepath.FromSlash(("backend/postgres/frame/backend_postgres_frame_total_overflow" +
			"_guard_service.tetra")),
		filepath.FromSlash("backend/postgres/frame/backend_postgres_frame_short_guard_service.tetra"),
		filepath.FromSlash(("backend/postgres/frame/backend_postgres_frame_writer_short_g" +
			"uard_service.tetra")),
		filepath.FromSlash(("backend/postgres/read/backend_postgres_ready_status_bounds_g" +
			"uard_service.tetra")),
		filepath.FromSlash(("backend/postgres/read/backend_postgres_ready_status_short_gu" +
			"ard_service.tetra")),
		filepath.FromSlash(("backend/postgres/row/backend_postgres_column_count_bounds_gu" +
			"ard_service.tetra")),
		filepath.FromSlash(("backend/postgres/row/backend_postgres_column_count_signed_gu" +
			"ard_service.tetra")),
		filepath.FromSlash("backend/postgres/read/backend_postgres_read_bounds_guard_service.tetra"),
		filepath.FromSlash("backend/postgres/read/backend_postgres_read_short_guard_service.tetra"),
		filepath.FromSlash("backend/postgres/read/backend_postgres_high_bit_read_guard_service.tetra"),
		filepath.FromSlash("backend/postgres/write/backend_postgres_write_bounds_guard_service.tetra"),
		filepath.FromSlash("backend/postgres/write/backend_postgres_signed_write_guard_service.tetra"),
		filepath.FromSlash("backend/postgres/write/backend_postgres_write_short_guard_service.tetra"),
		filepath.FromSlash(("backend/postgres/write/backend_postgres_text_write_bounds_gu" +
			"ard_service.tetra")),
		filepath.FromSlash(("backend/postgres/write/backend_postgres_text_write_short_gua" +
			"rd_service.tetra")),
		filepath.FromSlash(("backend/postgres/ascii/backend_postgres_ascii_i32_overflow_g" +
			"uard_service.tetra")),
		filepath.FromSlash("backend/http/response/backend_http_response_guard_service.tetra"),
		filepath.FromSlash("backend/http/writer/backend_http_response_writer_short_guard_service.tetra"),
		filepath.FromSlash(("backend/http/request/backend_http_negative_content_length_gu" +
			"ard_service.tetra")),
		filepath.FromSlash("backend/http/response/backend_http_status_code_guard_service.tetra"),
		filepath.FromSlash("backend/http/headers/backend_http_header_injection_guard_service.tetra"),
		filepath.FromSlash("backend/http/headers/backend_http_header_control_guard_service.tetra"),
		filepath.FromSlash("backend/http/writer/backend_http_writer_bounds_guard_service.tetra"),
		filepath.FromSlash("backend/http/writer/backend_http_writer_short_guard_service.tetra"),
		filepath.FromSlash("backend/http/response/backend_http_status_matrix_service.tetra"),
		filepath.FromSlash("backend/json/backend_http_json_i32_min_guard_service.tetra"),
		filepath.FromSlash("backend/http/headers/backend_http_header_whitespace_service.tetra"),
		filepath.FromSlash("backend/http/connection/backend_http_connection_list_service.tetra"),
		filepath.FromSlash("backend/http/connection/backend_http_connection_scope_service.tetra"),
		filepath.FromSlash(("backend/http/connection/backend_http_connection_token_bounda" +
			"ry_service.tetra")),
		filepath.FromSlash("backend/http/response/backend_http_version_scope_service.tetra"),
		filepath.FromSlash("backend/http/request/backend_http_request_target_guard_service.tetra"),
		filepath.FromSlash("backend/http/request/backend_http_request_target_char_guard_service.tetra"),
		filepath.FromSlash("backend/http/request/backend_http_request_line_token_guard_service.tetra"),
		filepath.FromSlash("backend/http/request/backend_http_request_crlf_guard_service.tetra"),
		filepath.FromSlash("backend/http/request/backend_http_request_short_guard_service.tetra"),
		filepath.FromSlash("backend/http/connection/backend_http_keep_alive_target_guard_service.tetra"),
		filepath.FromSlash("backend/http/connection/backend_http_connection_body_scope_service.tetra"),
		filepath.FromSlash("backend/http/connection/backend_http_keep_alive_method_guard_service.tetra"),
		filepath.FromSlash("backend/json/backend_json_escape_guard_service.tetra"),
		filepath.FromSlash("backend/json/backend_json_control_matrix_service.tetra"),
		filepath.FromSlash("backend/json/backend_json_hex_digit_guard_service.tetra"),
		filepath.FromSlash("backend/json/backend_json_writer_bounds_guard_service.tetra"),
		filepath.FromSlash("backend/json/backend_json_writer_short_guard_service.tetra"),
		filepath.FromSlash("backend/net/backend_network_policy_guard_service.tetra"),
		filepath.FromSlash("backend/net/backend_network_backoff_overflow_guard_service.tetra"),
		filepath.FromSlash("backend/crypto_fs/backend_crypto_serialization_guard_service.tetra"),
		filepath.FromSlash("backend/crypto_fs/backend_crypto_mix_min_guard_service.tetra"),
		filepath.FromSlash("backend/crypto_fs/backend_filesystem_path_policy_service.tetra"),
		filepath.FromSlash("backend/crypto_fs/backend_filesystem_nul_exists_guard_service.tetra"),
		filepath.FromSlash("backend/time/backend_time_collection_window_service.tetra"),
		filepath.FromSlash("backend/time/backend_time_overflow_guard_service.tetra"),
		filepath.FromSlash("backend/time/backend_time_negative_base_delta_guard_service.tetra"),
		filepath.FromSlash("backend/misc/backend_math_testing_decision_service.tetra"),
		filepath.FromSlash("backend/misc/backend_slice_capability_window_service.tetra"),
		filepath.FromSlash("backend/misc/backend_memory_io_buffer_service.tetra"),
		filepath.FromSlash("backend/misc/backend_sync_async_status_service.tetra"),
		filepath.FromSlash("backend/misc/backend_experimental_route_policy_service.tetra"),
		filepath.FromSlash("backend/misc/backend_experimental_buffer_mirror_service.tetra"),
		filepath.FromSlash("backend_modular_web_stack_pack/app/main.tetra"),
	} {
		stdout, exitCode := buildAndRunFile(
			t,
			filepath.Join(root, "examples", "microservices", name),
		)
		if stdout != "" {
			t.Fatalf("%s stdout mismatch: %q", name, stdout)
		}
		if exitCode != 0 {
			t.Fatalf("%s exit code mismatch: %d", name, exitCode)
		}
	}
	for _, name := range []string{
		filepath.FromSlash("backend_modular_web_stack_pack/app/main.tetra"),
		filepath.FromSlash("compiler_parallel_jobs_pack/app/main.tetra"),
	} {
		stdout, exitCode := buildAndRunFileWithOptions(
			t,
			filepath.Join(root, "examples", "microservices", name),
			BuildOptions{Jobs: 4},
		)
		if stdout != "" {
			t.Fatalf("%s stdout mismatch: %q", name, stdout)
		}
		if exitCode != 0 {
			t.Fatalf("%s exit code mismatch: %d", name, exitCode)
		}
	}
	for _, name := range []string{
		filepath.FromSlash("backend_modular_web_stack_pack/app/main.tetra"),
		filepath.FromSlash("compiler_interface_jobs_pack/app/main.tetra"),
		filepath.FromSlash("compiler_interface_control_pack/app/main.tetra"),
		filepath.FromSlash("compiler_import_alias_pack/app/main.tetra"),
		filepath.FromSlash("compiler_cross_module_actor_pack/app/main.tetra"),
		filepath.FromSlash("compiler_actor_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_callable_return_pack/app/main.tetra"),
		filepath.FromSlash("compiler_callable_optional_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_interface_pack/app/main.tetra"),
		filepath.FromSlash("compiler_generic_slice_pack/app/main.tetra"),
		filepath.FromSlash("compiler_optional_task_pack/app/main.tetra"),
		filepath.FromSlash("compiler_optional_actor_pack/app/main.tetra"),
		filepath.FromSlash("compiler_optional_group_pack/app/main.tetra"),
		filepath.FromSlash("compiler_optional_group_match_pack/app/main.tetra"),
		filepath.FromSlash("compiler_group_aggregate_pack/app/main.tetra"),
		filepath.FromSlash("compiler_group_generic_pack/app/main.tetra"),
		filepath.FromSlash("compiler_task_generic_pack/app/main.tetra"),
		filepath.FromSlash("compiler_actor_generic_pack/app/main.tetra"),
		filepath.FromSlash("compiler_island_generic_pack/app/main.tetra"),
		filepath.FromSlash("compiler_ptr_generic_pack/app/main.tetra"),
		filepath.FromSlash("compiler_task_result_generic_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_optional_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_resource_wrapper_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_resource_wrapper_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_generic_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_generic_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_throw_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_optional_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_generic_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_enum_memory_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_slice_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_slice_lane_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_slice_shape_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_string_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_async_optional_generic_string_pack/app/main.tetra"),
		filepath.FromSlash("compiler_generic_optional_field_pack/app/main.tetra"),
		filepath.FromSlash("compiler_ptr_optional_generic_call_pack/app/main.tetra"),
		filepath.FromSlash("compiler_optional_alias_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_optional_enum_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_typed_optional_alias_resource_pack/app/main.tetra"),
		filepath.FromSlash("compiler_typed_task_match_pack/app/main.tetra"),
		filepath.FromSlash("compiler_typed_error_payload_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_defer_cleanup_pack/app/main.tetra"),
		filepath.FromSlash("compiler_defer_unwind_pack/app/main.tetra"),
		filepath.FromSlash("compiler_task_wait_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_task_wait_error_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_actor_wait_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_actor_error_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_group_status_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_typed_error_optional_ptr_pack/app/main.tetra"),
		filepath.FromSlash("compiler_typed_actor_payload_memory_pack/app/main.tetra"),
		filepath.FromSlash("compiler_typed_actor_enum_payload_memory_pack/app/main.tetra"),
	} {
		outPath := filepath.Join(t.TempDir(), "interface-only")
		if _, err := BuildFileWithStatsOpt(
			filepath.Join(root, "examples", "microservices", name),
			outPath,
			"linux-x64",
			BuildOptions{Jobs: 4, InterfaceOnly: true},
		); err != nil {
			t.Fatalf("%s interface-only build: %v", name, err)
		}
	}
	for _, name := range []string{
		filepath.FromSlash("parallel/task_core/parallel_selfhost_deadline_service.tetra"),
	} {
		stdout, exitCode := buildAndRunFileWithOptions(
			t,
			filepath.Join(root, "examples", "microservices", name),
			BuildOptions{Runtime: RuntimeSelfHost},
		)
		if stdout != "" {
			t.Fatalf("%s stdout mismatch: %q", name, stdout)
		}
		if exitCode != 0 {
			t.Fatalf("%s exit code mismatch: %d", name, exitCode)
		}
	}

	raw, err := os.ReadFile(filepath.Join(root, "Tetra_BUGS.md"))
	if err != nil {
		t.Fatalf("read Tetra_BUGS.md: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"# Tetra Bugs",
		"Confirmed Language Bugs",
		"TETRA-BUG-0001",
		"cannot infer generic argument",
		"TETRA-BUG-0002",
		"unknown function 'Unit.score'",
		"TETRA-BUG-0003",
		"count mismatch: expected 1, got 2",
		"TETRA-BUG-0004",
		"Formatter drops function-typed local annotations",
		"TETRA-BUG-0005",
		"unknown function 'Router.run'",
		"TETRA-BUG-0006",
		"global var requires an explicit type annotation",
		"TETRA-BUG-0007",
		"Derived pointer arithmetic loses allocation provenance",
		"TETRA-BUG-0008",
		"Formatter rewrites mutable actor state fields as immutable",
		"TETRA-BUG-0009",
		"Blocking tagged receive fails in dual actor fan-in",
		"TETRA-BUG-0010",
		"Blocking value receive fails in dual actor fan-in",
		"TETRA-BUG-0011",
		"Struct constructors do not wrap scalar values into optional fields",
		"TETRA-BUG-0012",
		"Enum constructors do not wrap scalar values into optional payloads",
		"TETRA-BUG-0013",
		"Derived pointer loop arithmetic fails after pointer parameters",
		"TETRA-BUG-0014",
		"Formatter drops generic protocol requirement type parameters",
		"TETRA-BUG-0015",
		"Imported generic extension static calls do not monomorphize",
		"TETRA-BUG-0016",
		"Match case payload bindings leak across sibling cases",
		"TETRA-BUG-0017",
		"Stored derived pointers lose loadable memory provenance",
		"TETRA-BUG-0018",
		"Struct pointer fields lose memory provenance",
		"TETRA-BUG-0019",
		"Enum derived pointer payloads lose memory provenance",
		"TETRA-BUG-0020",
		"Generic identity over function-typed locals lowers to an unknown fn type",
		"TETRA-BUG-0021",
		"Optional derived pointer payloads lose memory provenance",
		"TETRA-BUG-0022",
		"Generic callback parameters do not accept compatible function symbols",
		"TETRA-BUG-0023",
		"Function returns of derived pointers lose memory provenance",
		"Function-typed callable returns of derived pointers hit the same guard",
		"TETRA-BUG-0024",
		"Global pointer variables lose memory provenance",
		"TETRA-BUG-0025",
		"Global integer offsets break raw pointer arithmetic provenance",
		"TETRA-BUG-0026",
		"Mutable local derived pointer variables lose memory provenance",
		"TETRA-BUG-0027",
		"Struct field offsets break raw pointer arithmetic provenance",
		"TETRA-BUG-0028",
		"Function-call offset operands break raw pointer arithmetic provenance",
		"TETRA-BUG-0029",
		"Expression offset operands break raw pointer arithmetic provenance",
		"TETRA-BUG-0030",
		"Runtime result and message fields break raw pointer arithmetic provenance",
		"TETRA-BUG-0031",
		"Enum payloads reject generic struct instantiations",
		"TETRA-BUG-0032",
		"Indexed and metadata offsets break raw pointer arithmetic provenance",
		"TETRA-BUG-0033",
		"Payload-typed task handles reject explicit task.i32 annotations",
		"TETRA-BUG-0034",
		"Direct pointer base expressions break raw pointer arithmetic provenance",
		"TETRA-BUG-0035",
		"Typed actor receives silently reinterpret mismatched enum message types",
		"TETRA-BUG-0036",
		"Typed error payloads of derived pointers lose memory provenance",
		"TETRA-BUG-0037",
		"Global fixed-array element writes do not round-trip at runtime",
		"TETRA-BUG-0038",
		"Scalar inout writes do not propagate back to caller locals",
		"TETRA-BUG-0039",
		"Dynamic ptr_add offsets from derived pointer locals lose memory provenance",
		"TETRA-BUG-0040",
		"Explicit selfhost task-group builds fail with raw missing ABI symbol",
		"TETRA-BUG-0041",
		"Scoped if-let and catch payload bindings remain reserved after scope exit",
		"TETRA-BUG-0042",
		"Stdlib byte helpers fail on valid derived memory windows",
		"TETRA-BUG-0043",
		"Formatter drops public visibility modifiers from declarations",
		"TETRA-BUG-0044",
		"Formatter corrupts selective import declarations",
		"TETRA-BUG-0045",
		"Spawning through an optional task-group payload returns the wrong worker value",
		"TETRA-BUG-0046",
		"Generic identity over actor/task/island resources loses usable provenance",
		"TETRA-BUG-0047",
		"Island parameters cannot be returned inside aggregate constructors",
		"TETRA-BUG-0048",
		"Function-typed local, field, and payload calls returning optionals fail as unknown functions",
		"TETRA-BUG-0049",
		"Generic inference fails on generic struct field selections",
		"TETRA-BUG-0050",
		"Task spawns inside match expressions miss required runtime symbols",
		"TETRA-BUG-0051",
		"Formatter corrupts nested catch cases inside match expression arms",
		"TETRA-BUG-0052",
		"Formatter corrupts nested match cases inside catch expression arms",
		"TETRA-BUG-0053",
		"Awaited optional resource locals lose provenance",
		"TETRA-BUG-0054",
		"Awaited resource aggregate locals lose provenance",
		"TETRA-BUG-0055",
		"Direct awaited pointer returns ignore await and try",
		"TETRA-BUG-0056",
		"Go workspace validation fails the SCRAM local benchmark package",
		"tetra_language/compiler@v0.0.0: malformed module path",
		"TETRA-BUG-0057",
		"Zero-length heap slice constructors call mmap with length 0",
		"TETRA-BUG-0058",
		"Explicit source-file CLI inputs inherit capsule source roots",
		"module 'examples.projects.dogfood_cli.src.main' must be in",
		"TETRA-BUG-0059",
		"HTTP Connection-close detection is case-sensitive",
		"service exited with status `22`",
		"TETRA-BUG-0060",
		"HTTP Connection-close detection rejects optional whitespace",
		"service exited with status `3`",
		"TETRA-BUG-0061",
		"HTTP Connection-close detection ignores close after comma",
		"service exited with status `4`",
		"TETRA-BUG-0062",
		"HTTP Connection-close detection matches suffix headers",
		"service exited with status `1`",
		"TETRA-BUG-0063",
		"HTTP Connection-close detection accepts close token prefixes",
		"Connection: closex",
		"TETRA-BUG-0064",
		"HTTP/1.1 detection scans beyond the request line",
		"X-Debug: HTTP/1.1",
		"TETRA-BUG-0065",
		"HTTP route helper classifies empty request targets as not found",
		"GET  /json HTTP/1.1",
		"TETRA-BUG-0066",
		"HTTP/1.1 detection accepts extra request-line tokens",
		"GET /json debug HTTP/1.1",
		"TETRA-BUG-0067",
		"HTTP keep-alive detection accepts malformed request targets",
		"GET noslash HTTP/1.1",
		"TETRA-BUG-0068",
		"HTTP Connection-close detection scans request bodies",
		"Connection: close` in the request body",
		"TETRA-BUG-0069",
		"HTTP keep-alive detection accepts malformed method tokens",
		"GE:T /json HTTP/1.1",
		"TETRA-BUG-0070",
		"JSON lowercase hex digit helper returns non-hex bytes out of range",
		"hex_digit_lower(-1)",
		"TETRA-BUG-0071",
		"Time duration helpers wrap positive overflow",
		"millis_from_seconds(2147484)",
		"TETRA-BUG-0072",
		"PostgreSQL C-string scan traps on negative start",
		"cstring_end_at(payload, -1, 6)",
		"TETRA-BUG-0073",
		"PostgreSQL signed DataRow length leaks malformed negative values",
		"0xfffffffe",
		"TETRA-BUG-0074",
		"PostgreSQL ASCII i32 parser traps on negative start",
		"parse_ascii_i32_at(bytes, -1, 2)",
		"TETRA-BUG-0075",
		"PostgreSQL CommandComplete parser traps on negative start",
		"command_complete_affected_rows(update_tag, -1, 4)",
		"TETRA-BUG-0076",
		"PostgreSQL RowDescription type-OID scan traps on negative start",
		"row_description_type_oid_at(desc, -1, i, 0)",
		"TETRA-BUG-0077",
		"PostgreSQL DataRow value helpers trap on negative start",
		"data_row_value_len_at(row, -1, 0)",
		"TETRA-BUG-0078",
		"PostgreSQL frame header readers trap on negative start",
		"frame_type_at(frame, -1)",
		"TETRA-BUG-0079",
		"PostgreSQL ReadyForQuery status reader traps on negative start",
		"ready_for_query_status(states, -1)",
		"TETRA-BUG-0080",
		"PostgreSQL column-count readers trap on negative start",
		"row_description_column_count(desc, -1)",
		"TETRA-BUG-0081",
		"PostgreSQL big-endian readers trap on negative start",
		"read_i32_be(bytes, -1)",
		"TETRA-BUG-0082",
		"PostgreSQL big-endian writers trap on negative start",
		"write_i32_be_at(bytes, -1, 1)",
		"TETRA-BUG-0083",
		"PostgreSQL text writers trap on negative start",
		"write_ascii_at(bytes, -1, \"Z\")",
		"TETRA-BUG-0084",
		"HTTP writer helpers trap on negative start",
		"write_ascii_at(out, -1, \"Z\")",
		"TETRA-BUG-0085",
		"JSON writers trap on negative start",
		"write_json_string_at(buf, -1, \"Z\")",
		"TETRA-BUG-0086",
		"HTTP/JSON i32 digit helpers collapse minimum value",
		"http.digits_i32(min_i32)",
		"TETRA-BUG-0087",
		"Crypto mix_seed overflows normalizing i32 minimum",
		"crypto.mix_seed(-65075262, -2)",
		"TETRA-BUG-0088",
		"Networking retry backoff overflows before cap",
		"retry_backoff_ms(1, 1073741824, 2147483647)",
		"TETRA-BUG-0089",
		"Epoll event extractors trap on short buffers",
		"net.epoll_event_fd(...)",
		"TETRA-BUG-0090",
		"PostgreSQL frame header readers trap on short buffers",
		"frame_length_at(...)",
		"TETRA-BUG-0091",
		"PostgreSQL big-endian readers trap on short buffers",
		"read_i32_be(...)",
		"TETRA-BUG-0092",
		"PostgreSQL big-endian writers trap on short buffers",
		"write_i32_be_at(...)",
		"TETRA-BUG-0093",
		"PostgreSQL text writers trap on short buffers",
		"write_cstring_pair_at(...)",
		"TETRA-BUG-0094",
		"HTTP writer helpers trap on short buffers",
		"write_header_at(...)",
		"TETRA-BUG-0095",
		"JSON writers trap on short buffers",
		"write_message_object_at(...)",
		"TETRA-BUG-0096",
		"PostgreSQL frame writers trap on short buffers",
		"write_startup_message(...)",
		"TETRA-BUG-0097",
		"PostgreSQL bounded parsers trap on short buffers",
		"parse_ascii_i32_at(...)",
		"TETRA-BUG-0098",
		"PostgreSQL ReadyForQuery status reader traps on short buffers",
		"ready_for_query_status(...)",
		"TETRA-BUG-0099",
		"HTTP request scanners trap on short buffers",
		"request_head_len_bytes_at(...)",
		"TETRA-BUG-0100",
		"HTTP response writers trap or partially write on short buffers",
		"write_response_head(...)",
		"TETRA-BUG-0101",
		"PostgreSQL big-endian writers encode negative values incorrectly",
		"write_i32_be_at(...)",
		"TETRA-BUG-0102",
		"PostgreSQL frame length reader leaks malformed signed lengths",
		"frame_length_at(...)",
		"TETRA-BUG-0103",
		"PostgreSQL big-endian reader leaks high-bit i32 values",
		"read_i32_be(...)",
		"TETRA-BUG-0104",
		"Time duration addition clamps negative base before applying positive delta",
		"add_duration_ms(-5, 10)",
		"TETRA-BUG-0105",
		"HTTP request-line scanner accepts LF-only or bare-CR HTTP/1.1 terminators",
		"HTTP/1.1\\n",
		"TETRA-BUG-0106",
		"filesystem.exists accepts embedded-NUL paths and checks only the prefix",
		"filesystem.exists",
		"TETRA-BUG-0107",
		"HTTP request-target scanners accept control bytes as route misses or keep-alive requests",
		"GET /json\\t HTTP/1.1",
		"TETRA-BUG-0108",
		"PostgreSQL ASCII i32 parser wraps out-of-range values",
		"parse_ascii_i32_at(\"2147483648\")",
		"TETRA-BUG-0109",
		"PostgreSQL CommandComplete affected-row parser wraps out-of-range values",
		"command_complete_affected_rows(\"UPDATE 2147483648\")",
		"TETRA-BUG-0110",
		"PostgreSQL CommandComplete parser returns non-trailing digit runs",
		"command_complete_affected_rows(\"UPDATE 12 rows\")",
		"TETRA-BUG-0111",
		"PostgreSQL DataRow helpers accept truncated positive value windows",
		"data_row_value_len_at(row, 0, 0)",
		"TETRA-BUG-0112",
		"PostgreSQL frame total length overflows at max signed length",
		"frame_total_len_at(frame, 0)",
		"TETRA-BUG-0113",
		"HTTP response writer accepts negative Content-Length",
		"response_head_len(200, \"OK\", \"Tetra\", date, \"text/plain\", -1, true)",
		"TETRA-BUG-0114",
		"HTTP response writer accepts non-three-digit status codes",
		"response_head_len(99, \"Bad\", \"Tetra\", date, \"text/plain\", 0, true)",
		"TETRA-BUG-0115",
		"HTTP header writers accept CR/LF header injection",
		"write_header_at(header, 0, \"X\\rBad\", \"ok\")",
		"TETRA-BUG-0116",
		"HTTP header writers accept non-HTAB control bytes",
		"ASCII 0x1f",
		"TETRA-BUG-0117",
		"PostgreSQL Parse writer wraps parameter counts above signed i16 range",
		"parse_payload_len(\"\", \"SELECT 1\", many)",
		"TETRA-BUG-0118",
		"TCP loopback bind accepts out-of-range ports",
		"bind_tcp4_loopback(fd, 65536, io_cap)",
		"TETRA-BUG-0119",
		"PostgreSQL column-count readers accept high-bit signed counts",
		"row_description_column_count(malformed, 0)",
		"TETRA-BUG-0120",
		"PostgreSQL C-string writers accept embedded NUL fields",
		"write_cstring_at(bytes, 0, bad)",
		"Microservice Bug-Hunt Runs",
		"Added backend HTTP/JSON pipeline and PostgreSQL prepared wire-frame microservice examples;",
		"Added backend net epoll lifecycle and PostgreSQL result guard microservice examples;",
		"Added backend HTTP response guard and JSON escape guard microservice examples;",
		"Added backend networking policy and crypto/serialization guard microservice examples;",
		"Added backend filesystem path-policy and time/collection window microservice examples;",
		"Added backend math/testing decision and slice/capability window microservice examples;",
		"Added backend memory/io buffer and sync/async status microservice examples;",
		"Added backend experimental route-policy and buffer mirror microservice examples;",
		"Added backend modular web-stack pack microservice example;",
		"Added backend capsule source-root microservice project;",
		"Added backend PostgreSQL session-state wire microservice example;",
		"Added backend HTTP status/header matrix microservice example;",
		"Added backend JSON control-character matrix microservice example;",
		"Added backend HTTP header-whitespace microservice example;",
		"Added backend HTTP Connection token-list microservice example;",
		"Added backend HTTP Connection header-scope microservice example;",
		"Added backend HTTP Connection token-boundary microservice example;",
		"Added backend HTTP request-version scope microservice example;",
		"Added backend HTTP request-target guard microservice example;",
		"Added backend HTTP request-line token guard microservice example;",
		"Added backend HTTP keep-alive request-target guard microservice example;",
		"Added backend HTTP Connection header/body scope microservice example;",
		"Added backend HTTP keep-alive method-token guard microservice example;",
		"Added backend JSON hex-digit guard microservice example;",
		"Added backend time overflow guard microservice example;",
		"Added backend PostgreSQL C-string bounds guard microservice example;",
		"Added backend PostgreSQL DataRow signed-length guard microservice example;",
		"Added backend PostgreSQL ASCII i32 bounds guard microservice example;",
		"Added backend PostgreSQL CommandComplete bounds guard microservice example;",
		"Added backend PostgreSQL RowDescription bounds guard microservice example;",
		"Added backend PostgreSQL DataRow bounds guard microservice example;",
		"Added backend PostgreSQL frame-header bounds guard microservice example;",
		"Added backend PostgreSQL ReadyForQuery status bounds guard microservice example;",
		"Added backend PostgreSQL column-count bounds guard microservice example;",
		"Added backend PostgreSQL big-endian reader bounds guard microservice example;",
		"Added backend PostgreSQL big-endian writer bounds guard microservice example;",
		"Added backend PostgreSQL text writer bounds guard microservice example;",
		"Added backend HTTP writer bounds guard microservice example;",
		"Added backend JSON writer bounds guard microservice example;",
		"Added backend HTTP/JSON i32 minimum guard microservice example;",
		"Added backend crypto mix i32 minimum guard microservice example;",
		"Added backend networking backoff overflow guard microservice example;",
		"Added backend net epoll event bounds guard microservice example;",
		"Added backend PostgreSQL short frame-header guard microservice example;",
		"Added backend PostgreSQL big-endian reader short-buffer guard microservice example;",
		"Added backend PostgreSQL big-endian writer short-buffer guard microservice example;",
		"Added backend PostgreSQL text writer short-buffer guard microservice example;",
		"Added backend HTTP writer short-buffer guard microservice example;",
		"Added backend JSON writer short-buffer guard microservice example;",
		"Added backend PostgreSQL frame writer short-buffer guard microservice example;",
		"Added backend PostgreSQL bounded parser short-buffer guard microservice example;",
		"Added backend PostgreSQL ReadyForQuery status short-buffer guard microservice example;",
		"Added backend HTTP request short-buffer guard microservice example;",
		"Added backend HTTP response writer short-buffer guard microservice example;",
		"Added backend PostgreSQL signed big-endian writer guard microservice example;",
		"Added backend PostgreSQL signed frame-length guard microservice example;",
		"Added backend PostgreSQL high-bit big-endian reader guard microservice example;",
		"Added backend PostgreSQL ASCII i32 minimum guard microservice example;",
		"Added backend time negative-base duration guard microservice example;",
		"Added backend HTTP request-line CRLF guard microservice example;",
		"Added backend filesystem embedded-NUL exists guard microservice example;",
		"Added backend HTTP request-target character guard microservice example;",
		"Added backend PostgreSQL ASCII i32 overflow guard microservice example;",
		"Added backend PostgreSQL CommandComplete affected-row overflow guard microservice example;",
		"Added backend PostgreSQL CommandComplete trailing-count guard microservice example;",
		"Added backend PostgreSQL DataRow truncated-value guard microservice example;",
		"Added backend PostgreSQL frame total-length overflow guard microservice example;",
		"Added backend HTTP negative Content-Length guard microservice example;",
		"Added backend HTTP status-code guard microservice example;",
		"Added backend HTTP header injection guard microservice example;",
		"Added backend HTTP header control-byte guard microservice example;",
		"Added backend PostgreSQL Parse parameter-count guard microservice example;",
		"Added backend net TCP port bounds guard microservice example;",
		"Added backend PostgreSQL signed column-count guard microservice example;",
		"Added backend PostgreSQL embedded-NUL C-string guard microservice example;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Tetra_BUGS.md missing %q", want)
		}
	}
}

// ---- compiler_interface_only_callable_test.go ----

func TestBuildInterfaceOnlyModeFunctionTypedParameterReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.identity

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`), "lib/identity.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.identity as id

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    cb = id.identity(f)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/identity.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only function-typed parameter-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedParameterLocalAliasReturnGlobalEscapeDiagnostic(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.identity

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    let alias: fn(Int) -> Int = f
    return alias
`), "lib/identity.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.identity as id

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    cb = id.identity(f)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/identity.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only function-typed parameter local-alias return global escape diagnostic",
		)
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedStructFieldReturnGlobalEscapeDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub func pick(holder: Holder) -> fn(Int) -> Int:
    return holder.cb
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	libIface, err := ParseFile(iface, "lib/callbacks.t4i")
	if err != nil {
		t.Fatalf("ParseFile interface: %v\ninterface:\n%s", err, iface)
	}
	checkedIface, err := CheckWorldOpt(&World{
		EntryModule:      "lib.callbacks",
		Files:            []*FileAST{libIface},
		InterfaceModules: map[string]bool{"lib.callbacks": true},
		ByModule: map[string]*FileAST{
			"lib.callbacks": libIface,
		},
	}, CheckOptions{RequireMain: false})
	if err != nil {
		t.Fatalf("CheckWorld interface: %v\ninterface:\n%s", err, iface)
	}
	pickSig := checkedIface.FuncSigs["lib.callbacks.pick"]
	if got := pickSig.ReturnFunctionParamName; got != "holder.cb" {
		t.Fatalf("pick ReturnFunctionParamName = %q, want holder.cb; interface:\n%s", got, iface)
	}
	if len(pickSig.ParamTypes) != 1 || pickSig.ParamTypes[0] != "lib.callbacks.Holder" {
		t.Fatalf(
			"pick ParamTypes = %#v, want lib.callbacks.Holder; interface:\n%s",
			pickSig.ParamTypes,
			iface,
		)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let holder: callbacks.Holder = callbacks.Holder(cb: f)
    cb = callbacks.pick(holder)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only function-typed struct-field-return global escape diagnostic",
		)
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedNestedStructFieldReturnGlobalEscapeDiagnostic(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func pick(box: Box) -> fn(Int) -> Int:
    return box.holder.cb
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let box: callbacks.Box = callbacks.Box(holder: callbacks.Holder(cb: f))
    cb = callbacks.pick(box)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only function-typed nested-struct-field-return global escape diagnostic",
		)
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedStructParameterWholeReturnGlobalEscapeDiagnostic(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func echo(box: Box) -> Box:
    return box
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let box: callbacks.Box = callbacks.Box(holder: callbacks.Holder(cb: f))
    let returned: callbacks.Box = callbacks.echo(box)
    cb = returned.holder.cb
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only function-typed struct-parameter whole-return global escape diagnostic",
		)
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedEnumParameterWholeReturnGlobalEscapeDiagnostic(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func echo(choice: MaybeCallback) -> MaybeCallback:
    return choice
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let choice: callbacks.MaybeCallback = callbacks.echo(callbacks.MaybeCallback.some(f))
    match choice:
    case callbacks.MaybeCallback.some(local):
        cb = local
        return 0
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only function-typed enum-parameter whole-return global escape diagnostic",
		)
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeFunctionTypedEnumPayloadMatchReturnGlobalEscapeDiagnostic(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func fallback(x: Int) -> Int:
    return x

pub func pick(choice: MaybeCallback) -> fn(Int) -> Int:
    match choice:
    case some(local):
        return local
    case empty:
        return fallback
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    cb = callbacks.pick(callbacks.MaybeCallback.some(f))
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			("expected interface-only function-typed enum-payload match " +
				"return global escape diagnostic\ninterface:\n%s"),
			iface,
		)
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedAggregateClosurePayloadStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        cb = local
        return 0
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned aggregate closure stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedEnumClosurePayloadStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        cb = local
        return 0
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only returned enum closure stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller() -> Int throws callbacks.Boom:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        return try local(41)
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf(
			"BuildFileWithStatsOpt interface-only returned throwing aggregate closure stub: %v",
			err,
		)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller() -> Int throws callbacks.Boom:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        return try local(41)
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf(
			"BuildFileWithStatsOpt interface-only returned throwing enum closure stub: %v",
			err,
		)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadRequiresTryDiagnostic(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        return local(41)
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only returned throwing aggregate closure payload requires-try diagnostic",
		)
	}
	want := "call to throwing function 'local' requires try"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadRequiresTryDiagnostic(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        return local(41)
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only returned throwing enum closure payload requires-try diagnostic",
		)
	}
	want := "call to throwing function 'local' requires try"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func caller() -> Int throws callbacks.Boom:
    let holder: callbacks.Holder = callbacks.makeHolder()
    return try holder.cb(41)

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf(
			"BuildFileWithStatsOpt interface-only returned throwing struct-field closure stub: %v",
			err,
		)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureRequiresTryDiagnostic(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func main() -> Int:
    let holder: callbacks.Holder = callbacks.makeHolder()
    return holder.cb(41)
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only returned throwing struct-field closure requires-try diagnostic",
		)
	}
	want := "call to throwing function 'holder.cb' requires try"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureCallbackStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int throws callbacks.Boom, x: Int) -> Int throws callbacks.Boom:
    return try f(x)

func caller() -> Int throws callbacks.Boom:
    let holder: callbacks.Holder = callbacks.makeHolder()
    return try apply(holder.cb, 41)

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf(
			"BuildFileWithStatsOpt interface-only returned throwing struct-field closure callback stub: %v",
			err,
		)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingStructFieldClosureCallbackThrowsMismatchDiagnostic(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    let holder: callbacks.Holder = callbacks.makeHolder()
    return apply(holder.cb, 41)
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			("expected interface-only returned throwing struct-field " +
				"closure callback throws mismatch diagnostic"),
		)
	}
	want := ("callback function symbol 'holder.cb' throws type mismatch: " +
		"expected '', got 'lib.callbacks.Boom'")
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadCallbackStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int throws callbacks.Boom, x: Int) -> Int throws callbacks.Boom:
    return try f(x)

func caller() -> Int throws callbacks.Boom:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        return try apply(local, 41)
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf(
			"BuildFileWithStatsOpt interface-only returned throwing aggregate closure callback stub: %v",
			err,
		)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadCallbackStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int throws callbacks.Boom, x: Int) -> Int throws callbacks.Boom:
    return try f(x)

func caller() -> Int throws callbacks.Boom:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        return try apply(local, 41)
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case callbacks.Boom.bad:
        0
`,
		"lib/callbacks.t4i": string(iface),
	})

	if _, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	); err != nil {
		t.Fatalf(
			"BuildFileWithStatsOpt interface-only returned throwing enum closure callback stub: %v",
			err,
		)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingEnumClosurePayloadCallbackThrowsMismatchDiagnostic(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    let choice: callbacks.MaybeCallback = callbacks.makeChoice()
    match choice:
    case callbacks.MaybeCallback.some(local):
        return apply(local, 41)
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only returned throwing enum closure callback throws mismatch diagnostic",
		)
	}
	want := ("callback function symbol 'local' throws type mismatch: " +
		"expected '', got 'lib.callbacks.Boom'")
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestBuildInterfaceOnlyModeReturnedThrowingAggregateClosurePayloadCallbackThrowsMismatchDiagnostic(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    ))
`), "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

func apply(f: fn(Int) -> Int, x: Int) -> Int:
    return f(x)

func main() -> Int:
    let box: callbacks.Box = callbacks.makeBox()
    match box.choice:
    case callbacks.MaybeCallback.some(local):
        return apply(local, 41)
    case callbacks.MaybeCallback.empty:
        return 0
`,
		"lib/callbacks.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			("expected interface-only returned throwing aggregate closure " +
				"callback throws mismatch diagnostic"),
		)
	}
	want := ("callback function symbol 'local' throws type mismatch: " +
		"expected '', got 'lib.callbacks.Boom'")
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

// ---- compiler_interface_only_region_test.go ----

func TestBuildInterfaceOnlyModeDoesNotRequireMain(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"math/core.t4": "module math.core\npub func add(a: Int, b: Int) -> Int:\n    return a + b\n",
	})

	outPath := filepath.Join(tmp, "out", "app")
	stats, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("math/core.t4")),
		outPath,
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only no main: %v", err)
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("interface-only build should not emit %s, stat err=%v", outPath, err)
	}
	if len(stats.InterfaceModules) != 0 {
		t.Fatalf("InterfaceModules = %#v, want none for source-only graph", stats.InterfaceModules)
	}
}

func TestBuildInterfaceOnlyModeAcceptsGeneratedT4IWithImportedSignatureType(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module math.core

import math.types as mt

pub func norm(v: mt.Vec) -> Int:
    return v.x
`), "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4":   "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return 0\n",
		"math/core.t4i": string(iface),
		"math/types.t4": "module math.types\npub struct Vec:\n    x: Int\n",
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only imported signature type: %v", err)
	}
}

func TestBuildInterfaceOnlyModeAcceptsGeneratedT4IWithStructReturnStub(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module math.core

pub struct Point:
    x: Int

pub func origin() -> Point:
    return Point(x: 0)
`), "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": ("module app.main\nimport math.core as math\nfunc main() -> Int:" +
			"\n    math.origin()\n    return 0\n"),
		"math/core.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only struct return stub: %v", err)
	}
}

func TestBuildInterfaceOnlyModeRejectsAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func make_pair(a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.make_pair(a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only aggregate region return escape diagnostic")
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsOptionalAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func maybe_pair(a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.maybe_pair(a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only optional aggregate region return escape diagnostic\ninterface:\n%s",
			iface,
		)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsEnumPayloadRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum BufMsg:
    case both([]u8, []u8)
    case empty

pub func make_msg(a: island, b: island) -> BufMsg
uses alloc, islands, mem:
    return BufMsg.both(core.island_make_u8(a, 1), core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var msg: buffers.BufMsg = buffers.BufMsg.empty
    island(64) as a:
        island(64) as b:
            msg = buffers.make_msg(a, b)
    match msg:
    case buffers.BufMsg.both(left, right):
        return left[0]
    case buffers.BufMsg.empty:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only enum payload region return escape diagnostic\ninterface:\n%s",
			iface,
		)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsOptionalEnumPayloadRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum BufMsg:
    case both([]u8, []u8)
    case empty

pub func maybe_msg(a: island, b: island) -> BufMsg?
uses alloc, islands, mem:
    var out: BufMsg? = none
    out = BufMsg.both(core.island_make_u8(a, 1), core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.BufMsg? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.maybe_msg(a, b)
    match maybe:
    case some(msg):
        match msg:
        case buffers.BufMsg.both(left, right):
            return left[0]
        case buffers.BufMsg.empty:
            return 0
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only optional enum payload region return escape diagnostic\ninterface:\n%s",
			iface,
		)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsBranchAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool, a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    if flag:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    else:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.choose_pair(true, a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only branch aggregate region return escape diagnostic\ninterface:\n%s",
			iface,
		)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsBranchOptionalAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool, a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    if flag:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    else:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.choose_pair(true, a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			("expected interface-only branch optional aggregate region " +
				"return escape diagnostic\ninterface:\n%s"),
			iface,
		)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsBranchOptionalMixedAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool, a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    if flag:
        out = PairBuf(left: make_u8(1), right: make_u8(1))
    else:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.choose_pair(false, a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			("expected interface-only branch optional mixed aggregate " +
				"region return escape diagnostic\ninterface:\n%s"),
			iface,
		)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsMatchAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum Mode:
    case fast
    case slow

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(mode: Mode, a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    match mode:
    case Mode.fast:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    case Mode.slow:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.choose_pair(buffers.Mode.fast, a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only match aggregate region return escape diagnostic\ninterface:\n%s",
			iface,
		)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsMatchOptionalAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum Mode:
    case fast
    case slow

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(mode: Mode, a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    match mode:
    case Mode.fast:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    case Mode.slow:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.choose_pair(buffers.Mode.fast, a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			("expected interface-only match optional aggregate region " +
				"return escape diagnostic\ninterface:\n%s"),
			iface,
		)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsIfLetOptionalAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool?, a: island, b: island) -> PairBuf?
uses alloc, islands, mem:
    var out: PairBuf? = none
    if let enabled = flag:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    else:
        out = PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
    return out
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var maybe: buffers.PairBuf? = none
    island(64) as a:
        island(64) as b:
            maybe = buffers.choose_pair(true, a, b)
    match maybe:
    case some(pair):
        return pair.left[0]
    case none:
        return 0
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			("expected interface-only if-let optional aggregate region " +
				"return escape diagnostic\ninterface:\n%s"),
			iface,
		)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsIfLetMixedAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(flag: Bool?, a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    if let enabled = flag:
        return PairBuf(left: make_u8(1), right: make_u8(1))
    else:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.choose_pair(none, a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only if-let mixed aggregate region return escape diagnostic\ninterface:\n%s",
			iface,
		)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

func TestBuildInterfaceOnlyModeRejectsMatchMixedAggregateRegionReturnEscape(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.buffers

pub enum Mode:
    case fast
    case slow

pub struct PairBuf:
    left: []u8
    right: []u8

pub func choose_pair(mode: Mode, a: island, b: island) -> PairBuf
uses alloc, islands, mem:
    match mode:
    case Mode.fast:
        return PairBuf(left: make_u8(1), right: make_u8(1))
    case Mode.slow:
        return PairBuf(left: core.island_make_u8(a, 1), right: core.island_make_u8(b, 1))
`), "lib/buffers.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.buffers as buffers

func main() -> Int
uses alloc, islands, mem:
    var pair: buffers.PairBuf = buffers.PairBuf(left: make_u8(1), right: make_u8(1))
    island(64) as a:
        island(64) as b:
            pair = buffers.choose_pair(buffers.Mode.slow, a, b)
    return pair.left[0]
`,
		"lib/buffers.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only match mixed aggregate region return escape diagnostic\ninterface:\n%s",
			iface,
		)
	}
	if !strings.Contains(err.Error(), "slice from scoped island cannot escape") {
		t.Fatalf("error = %v, want scoped island escape diagnostic\ninterface:\n%s", err, iface)
	}
}

// ---- compiler_interface_only_resources_test.go ----

func TestBuildRejectsInterfaceOnlyDependencyWithoutInterfaceOnlyMode(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`), "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": ("module app.main\nimport math.core as math\nfunc main() -> Int:" +
			"\n    return math.add(40, 2)\n"),
		"math/core.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "app"),
		"linux-x64",
		BuildOptions{Jobs: 1},
	)
	if err == nil {
		t.Fatalf("expected interface-only dependency build rejection")
	}
	if !strings.Contains(
		err.Error(),
		"missing implementation object for interface module 'math.core'",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildInterfaceOnlyModeAllowsT4IDependencyWithoutOutput(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`), "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": ("module app.main\nimport math.core as math\nfunc main() -> Int:" +
			"\n    return math.add(40, 2)\n"),
		"math/core.t4i": string(iface),
	})

	outPath := filepath.Join(tmp, "out", "app")
	stats, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		outPath,
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only: %v", err)
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("interface-only build should not emit %s, stat err=%v", outPath, err)
	}
	if len(stats.InterfaceModules) != 1 || stats.InterfaceModules[0] != "math.core" {
		t.Fatalf("InterfaceModules = %#v, want [math.core]", stats.InterfaceModules)
	}
}

func TestBuildInterfaceOnlyModeRejectsTamperedBorrowedReturnLifetimeMetadata(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.views

pub func view(xs: borrow []u8) -> borrow []u8:
    return xs.borrow()
`), "lib/views.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	tampered := strings.Replace(string(iface), "source=xs", "source=ys", 1)
	if tampered == string(iface) {
		t.Fatalf("test fixture did not find borrowed return lifetime metadata:\n%s", iface)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.views as views

func relay(xs: borrow []u8) -> borrow []u8:
    return views.view(xs)

func main() -> Int:
    return 0
`,
		"lib/views.t4i": tampered,
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil || !strings.Contains(err.Error(), "invalid .t4i hash") {
		t.Fatalf(
			"BuildFileWithStatsOpt tampered lifetime metadata error = %v, want invalid .t4i hash",
			err,
		)
	}
}

func TestBuildInterfaceOnlyModePreservesTypedErrorResourceThrowProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub enum TaskErr:
    case wrap(task.i32)

pub func fail(task: task.i32) -> Int throws TaskErr
uses runtime:
    throw TaskErr.wrap(task)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return catch resources.fail(task):
    case resources.TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only typed-error resource provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesTypedErrorFieldLocalAliasResourceThrowProvenance(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub enum TaskErr:
    case wrap(task.i32)

pub func fail(box: TaskBox) -> Int throws TaskErr
uses runtime:
    let other: task.i32 = box.handle
    throw TaskErr.wrap(other)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    return catch resources.fail(box):
    case resources.TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only typed-error field local alias resource provenance diagnostic",
		)
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesTypedErrorResourceRethrowProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub enum TaskErr:
    case wrap(task.i32)

pub func fail(task: task.i32) -> Int throws TaskErr
uses runtime:
    throw TaskErr.wrap(task)

pub func wrapper(task: task.i32) -> Int throws TaskErr
uses runtime:
    return try fail(task)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return catch resources.wrapper(task):
    case resources.TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only typed-error rethrow resource provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesTypedErrorFieldLocalAliasResourceRethrowProvenance(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub enum TaskErr:
    case wrap(task.i32)

pub func fail(task: task.i32) -> Int throws TaskErr
uses runtime:
    throw TaskErr.wrap(task)

pub func wrapper(box: TaskBox) -> Int throws TaskErr
uses runtime:
    let other: task.i32 = box.handle
    return try fail(other)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    return catch resources.wrapper(box):
    case resources.TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only typed-error field local alias rethrow resource provenance diagnostic",
		)
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub enum TaskMsg:
    case wrap(task.i32)

pub func wrap(task: task.i32) -> TaskMsg:
    return TaskMsg.wrap(task)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: resources.TaskMsg = resources.wrap(task)
    match returned:
    case resources.TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub func maybe(task: task.i32) -> task.i32?:
    var out: task.i32? = none
    out = task
    return out
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: task.i32? = resources.maybe(task)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesLocalAliasResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub func alias(task: task.i32) -> task.i32:
    let other: task.i32 = task
    return other
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let other: task.i32 = resources.alias(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only local alias resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesAggregateLocalAliasResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func box(task: task.i32) -> TaskBox:
    let other: task.i32 = task
    return TaskBox(handle: other)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: resources.TaskBox = resources.box(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only aggregate local alias resource return provenance diagnostic",
		)
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'returned.handle'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesAggregateFieldResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func pass(box: TaskBox) -> TaskBox:
    return TaskBox(handle: box.handle)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    let returned: resources.TaskBox = resources.pass(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only aggregate field resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'returned.handle'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesAggregateFieldLocalAliasResourceReturnProvenance(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func pass(box: TaskBox) -> TaskBox:
    let other: task.i32 = box.handle
    return TaskBox(handle: other)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    let returned: resources.TaskBox = resources.pass(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only aggregate field local alias resource return provenance diagnostic",
		)
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'returned.handle'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesLetOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub func maybe(task: task.i32) -> task.i32?:
    let out: task.i32? = task
    return out
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: task.i32? = resources.maybe(task)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only let optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesOptionalFieldLocalAliasResourceReturnProvenance(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func maybe(box: TaskBox) -> task.i32?:
    let out: task.i32? = box.handle
    return out
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    let returned: task.i32? = resources.maybe(box)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only optional field local alias resource return provenance diagnostic",
		)
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesDirectIfLetOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    maybe: task.i32?

pub func maybe(input: TaskBox) -> task.i32?:
    if let other = input.maybe:
        return other
    else:
        return none
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.TaskBox = resources.TaskBox(maybe: task)
    let returned: task.i32? = resources.maybe(input)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only direct if-let optional resource return provenance diagnostic",
		)
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesDirectMatchOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    maybe: task.i32?

pub func maybe(input: TaskBox) -> task.i32?:
    match input.maybe:
    case some(other):
        return other
    case none:
        return none
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.TaskBox = resources.TaskBox(maybe: task)
    let returned: task.i32? = resources.maybe(input)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			"expected interface-only direct match optional resource return provenance diagnostic",
		)
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesStructOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    maybe: task.i32?

pub func box(task: task.i32) -> TaskBox:
    var out: task.i32? = none
    out = task
    return TaskBox(maybe: out)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: resources.TaskBox = resources.box(task)
    if let other = returned.maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only struct optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesStructOptionalFieldLocalAliasResourceReturnProvenance(
	t *testing.T,
) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    maybe: task.i32?

pub struct InputBox:
    handle: task.i32

pub func box(input: InputBox) -> TaskBox:
    let out: task.i32? = input.handle
    return TaskBox(maybe: out)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.InputBox = resources.InputBox(handle: task)
    let returned: resources.TaskBox = resources.box(input)
    if let other = returned.maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf(
			("expected interface-only struct optional field local alias " +
				"resource return provenance diagnostic"),
		)
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesIfLetOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct InputBox:
    maybe: task.i32?

pub struct TaskBox:
    maybe: task.i32?

pub func box(input: InputBox) -> TaskBox:
    if let other = input.maybe:
        return TaskBox(maybe: other)
    else:
        return TaskBox(maybe: none)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.InputBox = resources.InputBox(maybe: task)
    let returned: resources.TaskBox = resources.box(input)
    if let other = returned.maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only if-let optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesMatchOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct InputBox:
    maybe: task.i32?

pub struct TaskBox:
    maybe: task.i32?

pub func box(input: InputBox) -> TaskBox:
    match input.maybe:
    case some(other):
        return TaskBox(maybe: other)
    case none:
        return TaskBox(maybe: none)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.InputBox = resources.InputBox(maybe: task)
    let returned: resources.TaskBox = resources.box(input)
    if let other = returned.maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only match optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

// ---- compiler_language_world_test.go ----

func TestBuildValAssignmentFails(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 {\n  val x: i32 = 1\n  x = 2\n  return x\n}\n"
	if err := buildOnly(t, src); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestBuildStructLiteralAndFieldAssign(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("struct Vec2 { x: i32, y: i32 }\nfun main(): i32 {\n  var v: " +
		"Vec2 = Vec2{ x: 1, y: 2 }\n  v.x = 10\n  return v.x + v.y\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 12 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStructParam(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("struct Vec2 { x: i32, y: i32 }\nfun sum(v: Vec2): i32 {\n  " +
		"return v.x + v.y\n}\nfun main(): i32 {\n  return sum(Vec2{ x: " +
		"5, y: 7 })\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 12 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStructCrossModule(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/math.tetra": ("module engine.math\nstruct Vec2 { x: i32, y: i32 }\nfun sum(v:" +
			" Vec2): i32 {\n  return v.x + v.y\n}\n"),
		"app/game.tetra": ("module app.game\nimport engine.math as m\nfun main(): i32 {\n  " +
			"var v: m.Vec2 = m.Vec2{ x: 2, y: 3 }\n  return m.sum(v)\n}\n"),
	}
	stdout, exitCode := buildAndRunFiles(t, files, "app/game.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildImportedGlobalOnlyModuleSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/constants.t4": `module lib.constants

pub val answer: Int = 42
`,
		"app/main.t4": `module app.main
import lib.constants as constants

func main() -> Int:
    return 42
`,
	}
	stdout, exitCode := buildAndRunFiles(t, files, "app/main.t4")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStructValFieldAssignFails(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("struct Vec2 { x: i32, y: i32 }\nfun main(): i32 {\n  val v: " +
		"Vec2 = Vec2{ x: 1, y: 2 }\n  v.x = 3\n  return v.x\n}\n")
	if err := buildOnly(t, src); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestBuildFunctionCall(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun add(a: i32, b: i32): i32 {\n  return a + b\n}\nfun main(): " +
		"i32 {\n  return add(2, 3)\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallSevenArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun sum7(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: " +
		"i32): i32 {\n  return a + b + c + d + e + f + g\n}\nfun main():" +
		" i32 {\n  return sum7(1, 2, 3, 4, 5, 6, 7)\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 28 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallEightArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun sum8(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: " +
		"i32, h: i32): i32 {\n  return a + b + c + d + e + f + g + " +
		"h\n}\nfun main(): i32 {\n  return sum8(1, 2, 3, 4, 5, 6, 7, 8)" +
		"\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 36 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallNineArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun sum9(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: " +
		"i32, h: i32, i: i32): i32 {\n  return a + b + c + d + e + f " +
		"+ g + h + i\n}\nfun main(): i32 {\n  return sum9(1, 2, 3, 4, 5," +
		" 6, 7, 8, 9)\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 45 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallPackNineArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun pack9(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g:" +
		" i32, h: i32, i: i32): i32 {\n  var ok: i32 = 1\n  if (a == 1)" +
		" { ; } else { ok = 0 }\n  if (b == 2) { ; } else { ok = 0 }\n " +
		" if (c == 3) { ; } else { ok = 0 }\n  if (d == 4) { ; } else " +
		"{ ok = 0 }\n  if (e == 5) { ; } else { ok = 0 }\n  if (f == 6)" +
		" { ; } else { ok = 0 }\n  if (g == 7) { ; } else { ok = 0 }\n " +
		" if (h == 8) { ; } else { ok = 0 }\n  if (i == 9) { ; } else " +
		"{ ok = 0 }\n  return ok\n}\nfun main(): i32 {\n  return pack9(1," +
		" 2, 3, 4, 5, 6, 7, 8, 9)\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallEightArgsNonEmptyStack(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun sum8(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: " +
		"i32, h: i32): i32 {\n  return a + b + c + d + e + f + g + " +
		"h\n}\nfun main(): i32 {\n  return 1 + sum8(1, 2, 3, 4, 5, 6, 7," +
		" 8)\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 37 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallNineArgsNonEmptyStack(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun sum9(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: " +
		"i32, h: i32, i: i32): i32 {\n  return a + b + c + d + e + f " +
		"+ g + h + i\n}\nfun main(): i32 {\n  return 1 + sum9(1, 2, 3, " +
		"4, 5, 6, 7, 8, 9)\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 46 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallSevenArgsNonEmptyStack(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun sum7(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: " +
		"i32): i32 {\n  return a + b + c + d + e + f + g\n}\nfun main():" +
		" i32 {\n  return 1 + sum7(1, 2, 3, 4, 5, 6, 7)\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 29 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFunctionCallNestedArgs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun sum2(a: i32, b: i32): i32 {\n  return a + b\n}\nfun " +
		"pack9(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: " +
		"i32, h: i32, i: i32): i32 {\n  var ok: i32 = 1\n  if (a == 21)" +
		" { ; } else { ok = 0 }\n  if (b == 2) { ; } else { ok = 0 }\n " +
		" if (c == 3) { ; } else { ok = 0 }\n  if (d == 4) { ; } else " +
		"{ ok = 0 }\n  if (e == 5) { ; } else { ok = 0 }\n  if (f == 6)" +
		" { ; } else { ok = 0 }\n  if (g == 7) { ; } else { ok = 0 }\n " +
		" if (h == 8) { ; } else { ok = 0 }\n  if (i == 9) { ; } else " +
		"{ ok = 0 }\n  return ok\n}\nfun main(): i32 {\n  return " +
		"pack9(sum2(10, 11), 2, 3, 4, 5, 6, 7, 8, 9)\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMultiFileCrossModuleCall(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra": ("module app.game\nimport engine.render as render\nfun main(): " +
			"i32 {\n  val v: i32 = render.add_one(41)\n  if (v == 42) { " +
			"return 1 }\n  return 0\n}\n"),
	}
	stdout, exitCode := buildAndRunFiles(t, files, "app/game.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMultiFileAliasCall(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra": ("module app.game\nimport engine.render as r\nfun main(): i32 " +
			"{\n  return r.add_one(41)\n}\n"),
	}
	stdout, exitCode := buildAndRunFiles(t, files, "app/game.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMultiFileMissingModule(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"app/game.tetra": ("module app.game\nimport engine.render as r\nfun main(): i32 " +
			"{\n  return r.add_one(1)\n}\n"),
	}
	if err := buildOnlyFiles(t, files, "app/game.tetra"); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestBuildMultiFileImportCycle(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"app/game.tetra": "module app.game\nimport mod.a as a\nfun main(): i32 {\n  return a.ping()\n}\n",
		"mod/a.tetra":    "module mod.a\nimport mod.b as b\nfun ping(): i32 {\n  return b.pong()\n}\n",
		"mod/b.tetra":    "module mod.b\nimport mod.a as a\nfun pong(): i32 {\n  return 1\n}\n",
	}
	if err := buildOnlyFiles(t, files, "app/game.tetra"); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestBuildMultiFileDuplicateModule(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/render.tetra": ("module engine.render\nfun add_one(x: i32): i32 {\n  return x " +
			"+ 1\n}\n"),
		"engine/render_alias.tetra": ("module engine.render\nfun add_two(x: i32): i32 {\n  return x " +
			"+ 2\n}\n"),
		"app/game.tetra": ("module app.game\nimport engine.render as r\nimport " +
			"engine.render_alias as r2\nfun main(): i32 {\n  return " +
			"r.add_one(1) + r2.add_two(1)\n}\n"),
	}
	if err := buildOnlyFiles(t, files, "app/game.tetra"); err == nil {
		t.Fatalf("expected compilation error")
	}
}

func TestLoadWorldRejectsDuplicateImportPath(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra": ("module app.game\nimport engine.render as render\nimport " +
			"engine.render as r\nfun main(): i32 {\n  return " +
			"render.add_one(41)\n}\n"),
	}
	writeTestFiles(t, tmp, files)

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err == nil {
		t.Fatalf("expected duplicate import path error")
	}
	if !strings.Contains(err.Error(), "duplicate import 'engine.render'") {
		t.Fatalf("error = %v, want duplicate import path diagnostic", err)
	}
}

func TestLoadWorldReportsMissingImportPath(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"app/game.tetra": ("module app.game\nimport engine.missing as missing\nfun main():" +
			" i32 {\n  return missing.add_one(41)\n}\n"),
	}
	writeTestFiles(t, tmp, files)

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err == nil {
		t.Fatalf("expected missing import error")
	}
	if !strings.Contains(err.Error(), "load module 'engine.missing'") ||
		!strings.Contains(err.Error(), "read source") {
		t.Fatalf("error = %v, want missing import path diagnostic", err)
	}
}

func TestLoadWorldReportsImportCycle(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"app/game.tetra": "module app.game\nimport mod.a as a\nfun main(): i32 {\n  return a.ping()\n}\n",
		"mod/a.tetra":    "module mod.a\nimport mod.b as b\nfun ping(): i32 {\n  return b.pong()\n}\n",
		"mod/b.tetra":    "module mod.b\nimport mod.a as a\nfun pong(): i32 {\n  return 1\n}\n",
	}
	writeTestFiles(t, tmp, files)

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	if !strings.Contains(err.Error(), "import cycle detected at 'mod.a'") {
		t.Fatalf("error = %v, want import cycle diagnostic", err)
	}
}

func TestLoadWorldReportsDuplicateModuleDeclaration(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/render.tetra": ("module engine.render\nfun add_one(x: i32): i32 {\n  return x " +
			"+ 1\n}\n"),
		"engine/render_alias.tetra": ("module engine.render\nfun add_two(x: i32): i32 {\n  return x " +
			"+ 2\n}\n"),
		"app/game.tetra": ("module app.game\nimport engine.render as r\nimport " +
			"engine.render_alias as r2\nfun main(): i32 {\n  return " +
			"r.add_one(1) + r2.add_two(1)\n}\n"),
	}
	writeTestFiles(t, tmp, files)

	_, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err == nil {
		t.Fatalf("expected duplicate module error")
	}
	if !strings.Contains(err.Error(), "duplicate module 'engine.render'") {
		t.Fatalf("error = %v, want duplicate module diagnostic", err)
	}
}

func TestCheckWorldRejectsImportAliasShadowingTopLevelName(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/math.tetra": "module engine.math\nfun inc(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra": ("module app.game\nimport engine.math as math\nfun math(): i32 " +
			"{\n  return 1\n}\nfun main(): i32 {\n  return math()\n}\n"),
	}
	writeTestFiles(t, tmp, files)

	world, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/game.tetra")))
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	_, err = CheckWorld(world)
	if err == nil {
		t.Fatalf("expected alias shadowing error")
	}
	if !strings.Contains(err.Error(), "import alias 'math' conflicts with declaration 'math'") {
		t.Fatalf("error = %v, want alias shadowing diagnostic", err)
	}
}

func TestBuildValArgument(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun add(a: i32, b: i32): i32 {\n  return a + b\n}\nfun main(): " +
		"i32 {\n  val x: i32 = 4\n  return add(x, 1)\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildEmptyStatements(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  ;;;\n  let x: i32 = 2 + 3;\n  ;;;\n  return x;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

// ---- compiler_pipeline_stage_test.go ----

func TestPipelineResolveNativeTargetStage(t *testing.T) {
	native, handled, stats, err := resolveExecutableBuildTarget(
		"missing.tetra",
		"out",
		"linux-x64",
		BuildOptions{Jobs: 1},
	)
	if err != nil {
		t.Fatalf("resolve native target: %v", err)
	}
	if handled {
		t.Fatalf("linux-x64 executable build should continue through native pipeline")
	}
	if stats != nil {
		t.Fatalf("native resolve should not produce stats before pipeline execution")
	}
	if native.triple != "linux-x64" || native.codegen == nil {
		t.Fatalf("native target = %#v, want linux-x64 with codegen", native)
	}
	if native.target.Format != ctarget.FormatELF {
		t.Fatalf("native target format = %s, want elf", native.target.Format)
	}
	if native.backend.name != "linux-x64" || native.backend.link == nil ||
		native.backend.actorRuntime == nil {
		t.Fatalf(
			"native backend = %#v, want linux-x64 backend with linker and actor runtime",
			native.backend,
		)
	}

	for _, triple := range []string{"windows-x64", "macos-x64"} {
		tgt, err := ctarget.Parse(triple)
		if err != nil {
			t.Fatalf("parse %s: %v", triple, err)
		}
		backend, ok := nativeExecutableBackendForTarget(tgt)
		if !ok {
			t.Fatalf("missing native backend for %s", triple)
		}
		if backend.link == nil || backend.codegen == nil || backend.actorRuntime == nil {
			t.Fatalf("incomplete backend for %s: %#v", triple, backend)
		}
	}

	_, handled, _, err = resolveExecutableBuildTarget(
		"missing.tetra",
		"out.wasm",
		"wasm32-wasi",
		BuildOptions{Jobs: 1, DebugInfo: true},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "target does not support debug info: wasm32-wasi") {
		t.Fatalf("wasm debug-info rejection error = %v", err)
	}
	if handled {
		t.Fatalf("capability rejection should fail before wasm build dispatch")
	}

	for _, triple := range []string{"wasm32-wasi", "wasm32-web"} {
		_, handled, stats, err = resolveExecutableBuildTarget(
			"missing.tetra",
			"out.wasm",
			triple,
			BuildOptions{Jobs: 1, Emit: EmitObject},
		)
		if err == nil || !strings.Contains(err.Error(), "supports only --emit=exe") {
			t.Fatalf("%s object emit error = %v, want build-only emit rejection", triple, err)
		}
		if !handled {
			t.Fatalf(
				"%s should be handled by build-only WASM pipeline before native emit/object dispatch",
				triple,
			)
		}
		if stats != nil {
			t.Fatalf("%s failed dispatch returned stats %#v", triple, stats)
		}
	}

	_, _, _, err = resolveExecutableBuildTarget(
		"missing.tetra",
		"out",
		"unknown-target",
		BuildOptions{Jobs: 1},
	)
	if err == nil || !strings.Contains(err.Error(), "unsupported target: unknown-target") {
		t.Fatalf("unknown target error = %v", err)
	}

	native, handled, stats, err = resolveExecutableBuildTarget(
		"missing.tetra",
		"out",
		"x32",
		BuildOptions{Jobs: 1},
	)
	if err != nil {
		t.Fatalf("resolve x32 executable target: %v", err)
	}
	if handled {
		t.Fatalf("x32 executable build should continue through native pipeline")
	}
	if stats != nil {
		t.Fatalf("x32 native resolve should not produce stats before pipeline execution")
	}
	if native.triple != "linux-x32" || native.codegen == nil {
		t.Fatalf("x32 native target = %#v, want linux-x32 with codegen", native)
	}
	if native.backend.name != "linux-x32" || native.backend.link == nil ||
		native.backend.actorRuntime != nil {
		t.Fatalf("x32 backend = %#v, want linker/codegen without runtime", native.backend)
	}

	native, handled, stats, err = resolveExecutableBuildTarget(
		"missing.tetra",
		"out",
		"x86",
		BuildOptions{Jobs: 1},
	)
	if err != nil {
		t.Fatalf("resolve x86 executable target: %v", err)
	}
	if handled {
		t.Fatalf("x86 executable build should continue through native pipeline")
	}
	if stats != nil {
		t.Fatalf("x86 native resolve should not produce stats before pipeline execution")
	}
	if native.triple != "linux-x86" || native.codegen == nil {
		t.Fatalf("x86 native target = %#v, want linux-x86 with codegen", native)
	}
	if native.backend.name != "linux-x86" || native.backend.link == nil ||
		native.backend.actorRuntime != nil {
		t.Fatalf("x86 backend = %#v, want linker/codegen without runtime", native.backend)
	}
}

func TestNativeCodegenOptionsInheritTargetWidths(t *testing.T) {
	for _, tc := range []struct {
		target        string
		pointerWidth  int
		nativeWidth   int
		registerWidth int
	}{
		{"linux-x64", 64, 64, 64},
		{"windows-x64", 64, 64, 64},
		{"x32", 32, 32, 64},
		{"x86", 32, 32, 32},
	} {
		tgt, err := ctarget.Parse(tc.target)
		if err != nil {
			t.Fatalf("parse %s: %v", tc.target, err)
		}
		got := nativeCodegenOptionsForTarget(
			tgt,
			BuildOptions{IslandsDebug: true, DebugInfo: true, ReleaseOptimize: true},
		)
		if got.PointerWidthBits != tc.pointerWidth || got.NativeIntWidthBits != tc.nativeWidth ||
			got.RegisterWidthBits != tc.registerWidth {
			t.Fatalf(
				"%s codegen widths ptr=%d native=%d reg=%d, want ptr=%d native=%d reg=%d",
				tc.target,
				got.PointerWidthBits,
				got.NativeIntWidthBits,
				got.RegisterWidthBits,
				tc.pointerWidth,
				tc.nativeWidth,
				tc.registerWidth,
			)
		}
		if !got.IslandsDebug || !got.DebugInfo || !got.ReleaseOptimize {
			t.Fatalf("%s codegen flags not preserved: %#v", tc.target, got)
		}
	}
}

func TestNativeCodegenOptionsUsePortableTargetFeatureBaseline(t *testing.T) {
	for _, tc := range []struct {
		target      string
		wantFeature bool
	}{
		{"linux-x64", true},
		{"windows-x64", true},
		{"macos-x64", true},
		{"x32", true},
		{"x86", false},
	} {
		tgt, err := ctarget.Parse(tc.target)
		if err != nil {
			t.Fatalf("parse %s: %v", tc.target, err)
		}
		opt := nativeCodegenOptionsForTarget(tgt, BuildOptions{ReleaseOptimize: true})
		evidence, err := opt.TargetFeatureEvidence()
		if err != nil {
			t.Fatalf("%s TargetFeatureEvidence: %v", tc.target, err)
		}
		if evidence.Source != string(x64.TargetFeatureSourcePortableBaseline) ||
			!evidence.PortableBaselineFallback {
			t.Fatalf("%s target feature source = %#v, want portable baseline", tc.target, evidence)
		}
		hasSSE2 := containsString(evidence.Features, string(x64.TargetFeatureSSE2))
		if hasSSE2 != tc.wantFeature {
			t.Fatalf(
				"%s sse2 baseline = %v, want %v; evidence=%#v",
				tc.target,
				hasSSE2,
				tc.wantFeature,
				evidence,
			)
		}
		if evidence.ChangesSafeSemantics || evidence.EnablesTargetSpecificOptimization {
			t.Fatalf(
				"%s target feature evidence changed semantics or enabled tuning: %#v",
				tc.target,
				evidence,
			)
		}
		if opt.TargetFeatures.Source != "" || len(opt.TargetFeatures.Features) != 0 {
			t.Fatalf(
				"%s native codegen options should not get explicit target features from BuildOptions: %#v",
				tc.target,
				opt.TargetFeatures,
			)
		}
	}
}

func TestNativeX86CodegenHonorsIslandsDebugOption(t *testing.T) {
	tgt, err := ctarget.Parse("x86")
	if err != nil {
		t.Fatalf("parse x86: %v", err)
	}
	codegen, err := nativeCodegenForTarget(tgt, BuildOptions{IslandsDebug: true})
	if err != nil {
		t.Fatalf("native codegen: %v", err)
	}
	obj, err := codegen([]IRFunc{{
		Name:        "main",
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 64},
			{Kind: ir.IRIslandNew},
			{Kind: ir.IRIslandFree},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}}, nil)
	if err != nil {
		t.Fatalf("x86 native codegen: %v", err)
	}
	if !bytes.Contains(obj.Code, []byte{0xC7, 0x00, 0x00, 0x10, 0x00, 0x00}) {
		t.Fatalf("x86 native codegen ignored IslandsDebug header:\n% x", obj.Code)
	}
	if !bytes.Contains(obj.Code, []byte{0xB8, 0x7D, 0x00, 0x00, 0x00, 0xCD, 0x80}) {
		t.Fatalf("x86 native codegen ignored IslandsDebug free protect path:\n% x", obj.Code)
	}
}

func TestBuildTagFromOptionsIncludesLinkedObjectContentDeterministically(t *testing.T) {
	firstHash := sha256.Sum256([]byte("first-object"))
	secondHash := sha256.Sum256([]byte("second-object"))
	changedHash := sha256.Sum256([]byte("second-object-changed"))

	base := BuildOptions{DebugInfo: true}
	firstOrder := []linkedObject{
		{path: "b.tobj", obj: &Object{Module: "dep.b"}, contentHash: secondHash},
		{path: "a.tobj", obj: &Object{Module: "dep.a"}, contentHash: firstHash},
	}
	secondOrder := []linkedObject{
		{path: "a.tobj", obj: &Object{Module: "dep.a"}, contentHash: firstHash},
		{path: "b.tobj", obj: &Object{Module: "dep.b"}, contentHash: secondHash},
	}

	got := buildTagFromOptions(base, firstOrder)
	if got == "" || !strings.Contains(got, "debug-info") || !strings.Contains(got, "link=") {
		t.Fatalf("build tag = %q, want debug/link components", got)
	}
	ownedDrop := buildTagFromOptions(BuildOptions{OwnedAllocDropLowering: true}, nil)
	if !strings.Contains(ownedDrop, "owned-alloc-drop-v1") {
		t.Fatalf("owned alloc drop build tag = %q, want owned-alloc-drop-v1", ownedDrop)
	}
	if reordered := buildTagFromOptions(base, secondOrder); reordered != got {
		t.Fatalf("linked object build tag should be order independent: %q vs %q", got, reordered)
	}

	changed := buildTagFromOptions(base, []linkedObject{
		{path: "a.tobj", obj: &Object{Module: "dep.a"}, contentHash: firstHash},
		{path: "b.tobj", obj: &Object{Module: "dep.b"}, contentHash: changedHash},
	})
	if changed == got {
		t.Fatalf("linked object build tag did not change after content hash changed: %q", got)
	}
}

func TestPipelineLoadCheckedBuildWorldRequireMainStage(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"math/core.t4": "module math.core\npub func add(a: Int, b: Int) -> Int:\n    return a + b\n",
	})
	entry := filepath.Join(tmp, filepath.FromSlash("math/core.t4"))

	build, err := loadCheckedBuildWorld(entry, BuildOptions{Jobs: 1}, false, "linux-x64")
	if err != nil {
		t.Fatalf("load checked world without main requirement: %v", err)
	}
	if build.world == nil || build.checked == nil {
		t.Fatalf("checked build world has nil fields: %#v", build)
	}
	if len(build.world.ByModule) != 1 || build.world.ByModule["math.core"] == nil {
		t.Fatalf("world modules = %#v, want math.core", build.world.ByModule)
	}

	_, err = loadCheckedBuildWorld(entry, BuildOptions{Jobs: 1}, true, "linux-x64")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "main") {
		t.Fatalf("require-main error = %v", err)
	}
}

func TestPipelineNativeModulePlanInvalidatesWhenLinkedObjectContentChanges(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": "module app.main\nfun main(): i32 {\n  return 42\n}\n",
	})
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.t4"))
	opt := BuildOptions{Jobs: 1}

	build, err := loadCheckedBuildWorld(entry, opt, true, "linux-x64")
	if err != nil {
		t.Fatalf("load checked world: %v", err)
	}
	target := "linux-x64"
	linkedA := []linkedObject{
		{
			path:        "dep.tobj",
			obj:         &Object{Module: "dep.lib"},
			contentHash: sha256.Sum256([]byte("dep-v1")),
		},
	}
	plan1, stats1, err := planNativeModuleBuild(build.world, build.checked, target, opt, linkedA)
	if err != nil {
		t.Fatalf("plan first build: %v", err)
	}
	if len(plan1.ToCompile) != 1 {
		t.Fatalf("first plan ToCompile = %#v, want app.main", plan1.ToCompile)
	}
	tgt, err := ctarget.Parse(target)
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	codegen, err := nativeCodegenForTarget(tgt, opt)
	if err != nil {
		t.Fatalf("native codegen: %v", err)
	}
	backend, ok := nativeExecutableBackendForTarget(tgt)
	if !ok {
		t.Fatalf("native backend: %s", tgt.Triple)
	}
	native := nativeBuildTarget{target: tgt, triple: tgt.Triple, backend: backend, codegen: codegen}
	if err := compileNativeModulePlan(
		build.world,
		build.checked,
		native,
		opt,
		plan1,
		stats1,
		nil,
	); err != nil {
		t.Fatalf("compile first plan: %v", err)
	}

	plan2, stats2, err := planNativeModuleBuild(build.world, build.checked, target, opt, linkedA)
	if err != nil {
		t.Fatalf("plan cached build: %v", err)
	}
	if len(plan2.ToCompile) != 0 || len(stats2.CacheHits) != 1 {
		t.Fatalf(
			"cached plan ToCompile=%#v cacheHits=%#v, want one cache hit",
			plan2.ToCompile,
			stats2.CacheHits,
		)
	}

	linkedB := []linkedObject{
		{
			path:        "dep.tobj",
			obj:         &Object{Module: "dep.lib"},
			contentHash: sha256.Sum256([]byte("dep-v2")),
		},
	}
	plan3, stats3, err := planNativeModuleBuild(build.world, build.checked, target, opt, linkedB)
	if err != nil {
		t.Fatalf("plan changed link object build: %v", err)
	}
	if len(plan3.ToCompile) != 1 {
		t.Fatalf("changed link object plan ToCompile=%#v, want rebuild", plan3.ToCompile)
	}
	if len(stats3.CacheHits) != 0 {
		t.Fatalf("changed link object cache hits=%#v, want none", stats3.CacheHits)
	}
}

func TestPipelineNativeModulePlanCacheStages(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/render.t4": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.t4": ("module app.game\nimport engine.render as r\nfun main(): i32 " +
			"{\n  return r.add_one(41)\n}\n"),
	})
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.t4"))
	opt := BuildOptions{Jobs: 1}

	build, err := loadCheckedBuildWorld(entry, opt, true, "linux-x64")
	if err != nil {
		t.Fatalf("load checked world: %v", err)
	}
	tgt, err := ctarget.Parse("linux-x64")
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	codegen, err := nativeCodegenForTarget(tgt, opt)
	if err != nil {
		t.Fatalf("native codegen: %v", err)
	}
	backend, ok := nativeExecutableBackendForTarget(tgt)
	if !ok {
		t.Fatalf("native backend: %s", tgt.Triple)
	}
	native := nativeBuildTarget{target: tgt, triple: tgt.Triple, backend: backend, codegen: codegen}

	plan1, stats1, err := planNativeModuleBuild(build.world, build.checked, native.triple, opt, nil)
	if err != nil {
		t.Fatalf("plan first build: %v", err)
	}
	testkit.AssertModules(t, plan1.Modules, []string{"app.game", "engine.render"})
	if len(plan1.ToCompile) != 2 {
		t.Fatalf("first plan ToCompile = %#v, want two modules", plan1.ToCompile)
	}
	if len(stats1.CacheHits) != 0 {
		t.Fatalf("first plan cache hits = %#v, want none", stats1.CacheHits)
	}
	if err := compileNativeModulePlan(
		build.world,
		build.checked,
		native,
		opt,
		plan1,
		stats1,
		nil,
	); err != nil {
		t.Fatalf("compile first plan: %v", err)
	}
	testkit.AssertModules(t, stats1.CompiledModules, []string{"app.game", "engine.render"})
	testkit.AssertModules(t, stats1.LoweredModules, []string{"app.game", "engine.render"})
	objects, err := objectsFromModulePlan(plan1)
	if err != nil {
		t.Fatalf("objects from first plan: %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("objects len = %d, want 2", len(objects))
	}

	plan2, stats2, err := planNativeModuleBuild(build.world, build.checked, native.triple, opt, nil)
	if err != nil {
		t.Fatalf("plan cached build: %v", err)
	}
	if len(plan2.ToCompile) != 0 {
		t.Fatalf("cached plan ToCompile = %#v, want none", plan2.ToCompile)
	}
	testkit.AssertModules(t, stats2.CacheHits, []string{"app.game", "engine.render"})
	if err := compileNativeModulePlan(
		build.world,
		build.checked,
		native,
		opt,
		plan2,
		stats2,
		nil,
	); err != nil {
		t.Fatalf("compile cached plan: %v", err)
	}
	if len(stats2.CompiledModules) != 0 || len(stats2.LoweredModules) != 0 {
		t.Fatalf(
			"cached stats compiled=%#v lowered=%#v, want none",
			stats2.CompiledModules,
			stats2.LoweredModules,
		)
	}
	objects, err = objectsFromModulePlan(plan2)
	if err != nil {
		t.Fatalf("objects from cached plan: %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("cached objects len = %d, want 2", len(objects))
	}
}

func TestP7CompilerPhaseProfileMemoryBudgetReducesWorkerCount(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/math.t4": "module engine.math\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"engine/render.t4": ("module engine.render\nimport engine.math as m\n" +
			"fun draw(x: i32): i32 {\n  return m.add_one(x)\n}\n"),
		"app/game.t4": ("module app.game\nimport engine.render as r\nfun main(): i32 " +
			"{\n  return r.draw(41)\n}\n"),
	})
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.t4"))
	outPath := filepath.Join(tmp, "game")
	profilePath := filepath.Join(tmp, "compiler-profile.json")
	if _, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", BuildOptions{
		Jobs:                    4,
		MemoryBudgetBytes:       64 * 1024 * 1024,
		EmitCompilerPhaseReport: true,
		CompilerPhaseReportPath: profilePath,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}
	raw, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("read compiler phase profile: %v", err)
	}
	var profile struct {
		MemoryBudgetBytes int64  `json:"memory_budget_bytes"`
		WorkerCount       int    `json:"worker_count"`
		WorkerReason      string `json:"worker_reason"`
	}
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatalf("parse compiler phase profile: %v\n%s", err, raw)
	}
	if profile.MemoryBudgetBytes != 64*1024*1024 ||
		profile.WorkerCount != 1 ||
		!strings.Contains(profile.WorkerReason, "memory_budget_bytes") {
		t.Fatalf("memory-budget worker decision = %+v\n%s", profile, raw)
	}
}

func TestP7NativeExecutableHashStableAcrossWorkerCounts(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/math.t4": "module engine.math\nfun add(x: i32, y: i32): i32 {\n  return x + y\n}\n",
		"engine/logic.t4": ("module engine.logic\nimport engine.math as m\n" +
			"fun twice_plus_one(x: i32): i32 {\n  return m.add(x, x) + 1\n}\n"),
		"engine/render.t4": ("module engine.render\nimport engine.logic as l\n" +
			"fun draw(x: i32): i32 {\n  return l.twice_plus_one(x)\n}\n"),
		"app/game.t4": ("module app.game\nimport engine.render as r\nfun main(): i32 " +
			"{\n  return r.draw(20)\n}\n"),
	})
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.t4"))
	var want [32]byte
	for i, jobs := range []int{1, 2, 4} {
		if err := os.RemoveAll(filepath.Join(tmp, ".tetra_cache")); err != nil {
			t.Fatalf("clear module cache before jobs=%d build: %v", jobs, err)
		}
		outPath := filepath.Join(tmp, "out", fmt.Sprintf("game-jobs-%d", jobs))
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			t.Fatalf("mkdir output dir for jobs=%d build: %v", jobs, err)
		}
		stats, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", BuildOptions{Jobs: jobs})
		if err != nil {
			t.Fatalf("BuildFileWithStatsOpt jobs=%d: %v", jobs, err)
		}
		if len(stats.CacheHits) != 0 {
			t.Fatalf("jobs=%d build used cache hits after cache clear: %#v", jobs, stats.CacheHits)
		}
		testkit.AssertModules(t, stats.CompiledModules, []string{
			"app.game",
			"engine.logic",
			"engine.math",
			"engine.render",
		})
		raw, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("read executable jobs=%d: %v", jobs, err)
		}
		got := sha256.Sum256(raw)
		if i == 0 {
			want = got
			continue
		}
		if got != want {
			t.Fatalf("jobs=%d executable hash = %x, want jobs=1 hash %x", jobs, got, want)
		}
	}
}

func TestP7JSONReportHashesStableAcrossWorkerCounts(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/math.t4": "module engine.math\nfun add(x: i32, y: i32): i32 {\n  return x + y\n}\n",
		"engine/logic.t4": ("module engine.logic\nimport engine.math as m\n" +
			"fun twice_plus_one(x: i32): i32 {\n  return m.add(x, x) + 1\n}\n"),
		"engine/render.t4": ("module engine.render\nimport engine.logic as l\n" +
			"fun draw(x: i32): i32 {\n  return l.twice_plus_one(x)\n}\n"),
		"app/game.t4": ("module app.game\nimport engine.render as r\nfun main(): i32 " +
			"{\n  return r.draw(20)\n}\n"),
	})
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.t4"))
	reportSuffixes := []string{".proof.json", ".bounds.json", ".alloc.json", ".memory.json"}
	want := make(map[string][32]byte, len(reportSuffixes))
	for i, jobs := range []int{1, 2, 4} {
		if err := os.RemoveAll(filepath.Join(tmp, ".tetra_cache")); err != nil {
			t.Fatalf("clear module cache before jobs=%d report build: %v", jobs, err)
		}
		outPath := filepath.Join(tmp, "out", fmt.Sprintf("game-reports-jobs-%d", jobs))
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			t.Fatalf("mkdir output dir for jobs=%d report build: %v", jobs, err)
		}
		stats, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", BuildOptions{
			Jobs:             jobs,
			EmitProof:        true,
			EmitBoundsReport: true,
			EmitAllocReport:  true,
			EmitMemoryReport: true,
		})
		if err != nil {
			t.Fatalf("BuildFileWithStatsOpt jobs=%d reports: %v", jobs, err)
		}
		if len(stats.CacheHits) != 0 {
			t.Fatalf("jobs=%d report build used cache hits after cache clear: %#v", jobs, stats.CacheHits)
		}
		for _, suffix := range reportSuffixes {
			raw, err := os.ReadFile(outPath + suffix)
			if err != nil {
				t.Fatalf("read %s for jobs=%d: %v", suffix, jobs, err)
			}
			got := sha256.Sum256(raw)
			if i == 0 {
				want[suffix] = got
				continue
			}
			if got != want[suffix] {
				t.Fatalf(
					"jobs=%d %s hash = %x, want jobs=1 hash %x",
					jobs,
					suffix,
					got,
					want[suffix],
				)
			}
		}
	}
}

func TestP7ReportBuildReleasesTransientMemoryBeforeProfileSnapshot(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/math.t4": "module engine.math\nfun add(x: i32, y: i32): i32 {\n  return x + y\n}\n",
		"app/game.t4": ("module app.game\nimport engine.math as m\nfun main(): i32 " +
			"{\n  return m.add(20, 22)\n}\n"),
	})
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.t4"))
	outPath := filepath.Join(tmp, "out", "game")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir output dir: %v", err)
	}
	profilePath := filepath.Join(tmp, "compiler-profile.json")
	releaseCalls := 0
	originalRelease := compilerProcessMemoryRelease
	compilerProcessMemoryRelease = func() { releaseCalls++ }
	t.Cleanup(func() { compilerProcessMemoryRelease = originalRelease })

	if _, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", BuildOptions{
		Jobs:                    1,
		EmitProof:               true,
		EmitBoundsReport:        true,
		EmitAllocReport:         true,
		EmitMemoryReport:        true,
		EmitCompilerPhaseReport: true,
		CompilerPhaseReportPath: profilePath,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt reports: %v", err)
	}
	if releaseCalls != 3 {
		t.Fatalf(
			"compilerProcessMemoryRelease calls = %d, want pre-object-retention, post-report, and pre-final-cleanup releases",
			releaseCalls,
		)
	}
	raw, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("read compiler phase profile: %v", err)
	}
	var profile struct {
		Phases []struct {
			Name string `json:"name"`
		} `json:"phases"`
	}
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatalf("parse compiler phase profile: %v\n%s", err, raw)
	}
	foundReportGeneration := false
	for _, phase := range profile.Phases {
		if phase.Name == "report_generation" {
			foundReportGeneration = true
			break
		}
	}
	if !foundReportGeneration {
		t.Fatalf("profile missing report_generation after post-report release: %s", raw)
	}
}

func TestP7FailedProfiledBuildReleasesMemoryBeforeFinalCleanupSnapshot(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"app/game.t4": "module app.game\nfun main(): i32 {\n  return missing_symbol\n}\n",
	})
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.t4"))
	outPath := filepath.Join(tmp, "out", "game")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir output dir: %v", err)
	}
	profilePath := filepath.Join(tmp, "failed-compiler-profile.json")
	releaseCalls := 0
	originalRelease := compilerProcessMemoryRelease
	compilerProcessMemoryRelease = func() { releaseCalls++ }
	t.Cleanup(func() { compilerProcessMemoryRelease = originalRelease })

	_, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", BuildOptions{
		Jobs:                    1,
		EmitCompilerPhaseReport: true,
		CompilerPhaseReportPath: profilePath,
	})
	if err == nil {
		t.Fatal("BuildFileWithStatsOpt succeeded, want semantic failure")
	}
	if releaseCalls != 1 {
		t.Fatalf(
			"compilerProcessMemoryRelease calls = %d, want failed-build pre-final-cleanup release",
			releaseCalls,
		)
	}
	raw, readErr := os.ReadFile(profilePath)
	if readErr != nil {
		t.Fatalf("read failed compiler phase profile: %v", readErr)
	}
	var profile struct {
		Notes  []string `json:"notes"`
		Phases []struct {
			Name                 string `json:"name"`
			SourceFileCount      int    `json:"source_file_count"`
			CheckedFunctionCount int    `json:"checked_function_count"`
			CheckedTypeCount     int    `json:"checked_type_count"`
		} `json:"phases"`
	}
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatalf("parse failed compiler phase profile: %v\n%s", err, raw)
	}
	foundNote := false
	for _, note := range profile.Notes {
		if strings.Contains(note, "build ended before successful completion") {
			foundNote = true
			break
		}
	}
	if !foundNote {
		t.Fatalf("failed profile missing failure note: %s", raw)
	}
	foundFinalCleanup := false
	var finalCleanup struct {
		Name                 string `json:"name"`
		SourceFileCount      int    `json:"source_file_count"`
		CheckedFunctionCount int    `json:"checked_function_count"`
		CheckedTypeCount     int    `json:"checked_type_count"`
	}
	for _, phase := range profile.Phases {
		if phase.Name == "final_cleanup" {
			foundFinalCleanup = true
			finalCleanup = phase
			break
		}
	}
	if !foundFinalCleanup {
		t.Fatalf("failed profile missing final_cleanup after failure release: %s", raw)
	}
	if finalCleanup.SourceFileCount != 0 ||
		finalCleanup.CheckedFunctionCount != 0 ||
		finalCleanup.CheckedTypeCount != 0 {
		t.Fatalf("failed final_cleanup retained counts = %+v, want zero after failure cleanup", finalCleanup)
	}
}

func TestP7WriteReportStreamsJSONToTemporaryFileAndRenames(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "compiler_reports.go"))
	if err != nil {
		t.Fatalf("read compiler_reports.go: %v", err)
	}
	body := functionBodyFromSourceForTest(t, string(raw), "writeReport")
	for _, forbidden := range []string{"bytes.Buffer", "os.WriteFile"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("writeReport still uses buffered report emission via %s:\n%s", forbidden, body)
		}
	}
	for _, want := range []string{"os.CreateTemp(", "json.NewEncoder(f)", "os.Rename(", "os.Remove("} {
		if !strings.Contains(body, want) {
			t.Fatalf("writeReport missing streaming marker %q:\n%s", want, body)
		}
	}
}

func TestP7CompilerPhaseProfileStreamsJSONToTemporaryFileAndRenames(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "compiler_phase_profile.go"))
	if err != nil {
		t.Fatalf("read compiler_phase_profile.go: %v", err)
	}
	body := functionBodyForNeedleFromSourceForTest(
		t,
		string(raw),
		"func (p *compilerPhaseProfiler) write(",
	)
	for _, forbidden := range []string{"json.MarshalIndent", "os.WriteFile"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf(
				"compilerPhaseProfiler.write still uses buffered profile emission via %s:\n%s",
				forbidden,
				body,
			)
		}
	}
	if !strings.Contains(body, "writeReport(outPath, p.report)") {
		t.Fatalf("compilerPhaseProfiler.write must reuse streaming writeReport:\n%s", body)
	}
}

func TestP7EmitPLIROnlyReturnsBeforeAllocPlanAndIRReportIntermediates(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "compiler_reports.go"))
	if err != nil {
		t.Fatalf("read compiler_reports.go: %v", err)
	}
	body := functionBodyFromSourceForTest(t, string(raw), "emitExplainReports")
	fastPath := strings.Index(body, "if plirOnly {")
	if fastPath < 0 {
		t.Fatal("emitExplainReports missing PLIR-only fast path before heavy report intermediates")
	}
	for _, heavy := range []string{"allocplan.FromPLIRWithOptions", "lower.LowerWithOptions"} {
		pos := strings.Index(body, heavy)
		if pos < 0 {
			t.Fatalf("emitExplainReports missing expected heavy report intermediate %q", heavy)
		}
		if fastPath > pos {
			t.Fatalf(
				"PLIR-only fast path appears after %s, so EmitPLIR still builds unnecessary report intermediates",
				heavy,
			)
		}
	}
}

type p7FailingJSONReport struct{}

func (p7FailingJSONReport) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("forced P7 report encode failure")
}

func TestP7WriteReportRemovesTemporaryFileAfterFailure(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "alloc-report.json")
	original := []byte("{\"version\":\"original\"}\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("write original report: %v", err)
	}

	err := writeReport(path, p7FailingJSONReport{})
	if err == nil {
		t.Fatal("writeReport succeeded with failing JSON marshaler")
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read original report after failed write: %v", readErr)
	}
	if !bytes.Equal(raw, original) {
		t.Fatalf("failed write clobbered original report:\ngot  %q\nwant %q", raw, original)
	}
	matches, globErr := filepath.Glob(filepath.Join(tmp, filepath.Base(path)+".*.tmp"))
	if globErr != nil {
		t.Fatalf("glob temporary reports: %v", globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("writeReport left temporary files after failure: %v", matches)
	}
}

func functionBodyFromSourceForTest(t *testing.T, source string, name string) string {
	t.Helper()
	return functionBodyForNeedleFromSourceForTest(t, source, "func "+name+"(")
}

func functionBodyForNeedleFromSourceForTest(t *testing.T, source string, needle string) string {
	t.Helper()
	start := strings.Index(source, needle)
	if start < 0 {
		t.Fatalf("source missing %s", needle)
	}
	open := strings.Index(source[start:], "{")
	if open < 0 {
		t.Fatalf("%s missing opening brace", needle)
	}
	open += start
	depth := 0
	for i := open; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[open+1 : i]
			}
		}
	}
	t.Fatalf("%s missing closing brace", needle)
	return ""
}

// ---- compiler_targets_cache_test.go ----

func TestNewOperatorMul(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { return 6 * 7 }"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestNewOperatorDiv(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { return 84 / 2 }"
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestNewOperatorMod(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { return 47 % 5 }"
	_, code := buildAndRun(t, src)
	if code != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", code)
	}
}

func TestNewOperatorGreater(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (5 > 3) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorGreaterEq(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (3 >= 3) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorLessEq(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (3 <= 3) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorBangEq(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (2 != 3) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorAmpAmp(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (true && true) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorAmpAmpFalse(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (true && false) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 0 {
		t.Fatalf("exit code mismatch: got %d, want 0", code)
	}
}

func TestNewOperatorPipePipe(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (false || true) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 1 {
		t.Fatalf("exit code mismatch: got %d, want 1", code)
	}
}

func TestNewOperatorPipePipeFalse(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 { if (false || false) { return 1 } return 0 }"
	_, code := buildAndRun(t, src)
	if code != 0 {
		t.Fatalf("exit code mismatch: got %d, want 0", code)
	}
}

func TestNewOperatorPrecedenceMixed(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	// 2 + 3 * 4 = 2 + 12 = 14
	src := "fn main() -> i32 { return 2 + 3 * 4 }"
	_, code := buildAndRun(t, src)
	if code != 14 {
		t.Fatalf("exit code mismatch: got %d, want 14", code)
	}
}

func TestExprStmt(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `fun side(): i32 { return 0 }
fun main(): i32 { side(); return 42 }`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestExprStmtQualified(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun noop(): i32 {\n  return 0\n}\n",
		"app/game.tetra": ("module app.game\nimport engine.render as r\nfun main(): i32 " +
			"{\n  r.noop()\n  return 42\n}\n"),
	}
	_, code := buildAndRunFiles(t, files, "app/game.tetra")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildWASMHelloWritesModule(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join("..", "examples", "smoke", "basic", "hello.tetra")

	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		outPath := filepath.Join(tmp, target+".wasm")
		if _, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1}); err != nil {
			t.Fatalf("build %s: %v", target, err)
		}
		data, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("read wasm: %v", err)
		}
		if len(data) < 8 {
			t.Fatalf("wasm too short: %d bytes", len(data))
		}
		if !bytes.Equal(data[:4], []byte{0x00, 0x61, 0x73, 0x6d}) {
			t.Fatalf("missing wasm magic: % x", data[:4])
		}
		if !bytes.Equal(data[4:8], []byte{0x01, 0x00, 0x00, 0x00}) {
			t.Fatalf("unexpected wasm version header: % x", data[4:8])
		}
		if target == "wasm32-web" {
			loaderPath := strings.TrimSuffix(outPath, ".wasm") + ".mjs"
			loaderRaw, err := os.ReadFile(loaderPath)
			if err != nil {
				t.Fatalf("read web loader: %v", err)
			}
			loader := string(loaderRaw)
			if !strings.Contains(loader, "tetra_web_v0.4.0") ||
				!strings.Contains(loader, "tetra_main") {
				t.Fatalf("unexpected web loader content:\n%s", loader)
			}
		}
	}
}

func TestBuildWASMWebUIWritesSidecars(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "ui_web.tetra")
	src := `state CounterState:
    var count: Int = 0
    val title: String = "Wave 9 Web"

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> increment
    command increment:
        state.count = state.count + 1
    style width: Int = 320
    accessibility label: String = "Increment"

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "ui.wasm")
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"wasm32-web",
		BuildOptions{Jobs: 1},
	); err != nil {
		t.Fatalf("build wasm32-web: %v", err)
	}
	uiJSON := strings.TrimSuffix(outPath, ".wasm") + ".ui.json"
	uiToolkitJSON := strings.TrimSuffix(outPath, ".wasm") + ".ui.toolkit.json"
	uiModule := strings.TrimSuffix(outPath, ".wasm") + ".ui.web.mjs"
	uiHTML := strings.TrimSuffix(outPath, ".wasm") + ".ui.html"

	jsonRaw, err := os.ReadFile(uiJSON)
	if err != nil {
		t.Fatalf("read ui json: %v", err)
	}
	if !strings.Contains(string(jsonRaw), `"schema": "tetra.ui.v0.4.0"`) ||
		!strings.Contains(string(jsonRaw), "CounterView") {
		t.Fatalf("unexpected ui json:\n%s", string(jsonRaw))
	}
	toolkitRaw, err := os.ReadFile(uiToolkitJSON)
	if err != nil {
		t.Fatalf("read ui toolkit json: %v", err)
	}
	if !strings.Contains(string(toolkitRaw), `"schema": "tetra.ui.toolkit.v1"`) ||
		!strings.Contains(string(toolkitRaw), `"compatibility_schema": "tetra.ui.v0.4.0"`) {
		t.Fatalf("unexpected ui toolkit json:\n%s", string(toolkitRaw))
	}
	moduleRaw, err := os.ReadFile(uiModule)
	if err != nil {
		t.Fatalf("read ui module: %v", err)
	}
	if !strings.Contains(string(moduleRaw), "mountTetraUI") {
		t.Fatalf("unexpected ui module:\n%s", string(moduleRaw))
	}
	htmlRaw, err := os.ReadFile(uiHTML)
	if err != nil {
		t.Fatalf("read ui html: %v", err)
	}
	if !strings.Contains(string(htmlRaw), ".ui.web.mjs") ||
		!strings.Contains(string(htmlRaw), "runTetra") {
		t.Fatalf("unexpected ui html:\n%s", string(htmlRaw))
	}
}

func TestBuildNativeUIWritesShellSidecar(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "ui_native.tetra")
	src := `state ShellState:
    var toggles: Int = 0
    val title: String = "Wave 9 Native"

view ShellView(state: ShellState):
    bind toggles: Int = state.toggles
    event submit -> toggle
    command toggle:
        state.toggles = state.toggles + 1
    style width: Int = 80
    accessibility label: String = "Toggle"

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "ui-app")
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"linux-x64",
		BuildOptions{Jobs: 1},
	); err != nil {
		t.Fatalf("build linux-x64: %v", err)
	}
	sidecarPath := outPath + ".ui.shell.txt"
	sidecar, err := os.ReadFile(sidecarPath)
	if err != nil {
		t.Fatalf("read native ui shell sidecar: %v", err)
	}
	if !strings.Contains(string(sidecar), "ShellView") ||
		!strings.Contains(string(sidecar), "event submit -> toggle") ||
		!strings.Contains(string(sidecar), "state.toggles = 1") {
		t.Fatalf("unexpected native ui sidecar:\n%s", string(sidecar))
	}
	tracePath := outPath + ".ui.shell.json"
	trace, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("read native ui shell json trace: %v", err)
	}
	for _, want := range []string{
		`"schema": "tetra.ui.native-shell.v1"`,
		`"runtime": "native shell command dispatch"`,
		`"widgets":`,
		`"kind": "value"`,
		`"binding": "toggles"`,
		`"kind": "action"`,
		`"event": "submit"`,
		`"state_field": "toggles"`,
		`"state_value": "1"`,
	} {
		if !strings.Contains(string(trace), want) {
			t.Fatalf("native ui shell json trace missing %q:\n%s", want, trace)
		}
	}
}

func TestBuildCacheSeparatesNativeDebugAndReleaseModes(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra": ("module app.game\nimport engine.render as r\nfun main(): i32 " +
			"{\n  return r.add_one(41)\n}\n"),
	}
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	baseOpt := BuildOptions{Jobs: 1}
	stats1, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", baseOpt)
	if err != nil {
		t.Fatalf("build1: %v", err)
	}
	testkit.AssertModules(t, stats1.CompiledModules, []string{"app.game", "engine.render"})
	if len(stats1.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits on first base build")
	}

	stats2, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", baseOpt)
	if err != nil {
		t.Fatalf("build2: %v", err)
	}
	if len(stats2.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on base cache hit")
	}
	testkit.AssertModules(t, stats2.CacheHits, []string{"app.game", "engine.render"})

	debugOpt := BuildOptions{Jobs: 1, DebugInfo: true}
	stats3, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", debugOpt)
	if err != nil {
		t.Fatalf("build3 debug: %v", err)
	}
	testkit.AssertModules(t, stats3.CompiledModules, []string{"app.game", "engine.render"})
	if len(stats3.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits on first debug build")
	}

	stats4, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", debugOpt)
	if err != nil {
		t.Fatalf("build4 debug: %v", err)
	}
	if len(stats4.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on debug cache hit")
	}
	testkit.AssertModules(t, stats4.CacheHits, []string{"app.game", "engine.render"})

	releaseOpt := BuildOptions{Jobs: 1, ReleaseOptimize: true}
	stats5, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", releaseOpt)
	if err != nil {
		t.Fatalf("build5 release: %v", err)
	}
	testkit.AssertModules(t, stats5.CompiledModules, []string{"app.game", "engine.render"})
	if len(stats5.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits on first release build")
	}

	stats6, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", releaseOpt)
	if err != nil {
		t.Fatalf("build6 release: %v", err)
	}
	if len(stats6.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on release cache hit")
	}
	testkit.AssertModules(t, stats6.CacheHits, []string{"app.game", "engine.render"})

	stats7, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", baseOpt)
	if err != nil {
		t.Fatalf("build7 base: %v", err)
	}
	if len(stats7.CompiledModules) != 0 {
		t.Fatalf("expected base mode cache to remain warm")
	}
	testkit.AssertModules(t, stats7.CacheHits, []string{"app.game", "engine.render"})
}

func TestBuildWASMCacheStatsRemainColdAcrossBuilds(t *testing.T) {
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		tmp := t.TempDir()
		files := map[string]string{
			"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
			"app/game.tetra": ("module app.game\nimport engine.render as r\nfun main(): i32 " +
				"{\n  return r.add_one(41)\n}\n"),
		}
		writeTestFiles(t, tmp, files)
		entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))
		outPath := filepath.Join(tmp, "out", target+".wasm")
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		stats1, err := BuildFileWithStatsOpt(entry, outPath, target, BuildOptions{Jobs: 1})
		if err != nil {
			t.Fatalf("build1 %s: %v", target, err)
		}
		testkit.AssertModules(t, stats1.CompiledModules, []string{"app.game", "engine.render"})
		if len(stats1.CacheHits) != 0 {
			t.Fatalf("%s unexpected cache hits on first build: %#v", target, stats1.CacheHits)
		}

		stats2, err := BuildFileWithStatsOpt(entry, outPath, target, BuildOptions{Jobs: 1})
		if err != nil {
			t.Fatalf("build2 %s: %v", target, err)
		}
		testkit.AssertModules(t, stats2.CompiledModules, []string{"app.game", "engine.render"})
		if len(stats2.CacheHits) != 0 {
			t.Fatalf("%s expected cache to stay cold: %#v", target, stats2.CacheHits)
		}
	}
}

// ---- compiler_test.go ----

func requireCheckFileErrorContains(t *testing.T, src string, want string) {
	t.Helper()
	testkit.RequireFileSemanticCheckErrorContains(t, src, want)
}

func requireCheckFileOK(t *testing.T, src string) {
	t.Helper()
	testkit.RequireFileSemanticCheckOK(t, src)
}

func TestBuildHello(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 uses io {\n  print(\"Hello from Tetra!\\n\");\n  return 0;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "Hello from Tetra!\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildTwoPrints(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 uses io {\n  print(\"A\");\n  print(\"B\\n\");\n  return 0;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "AB\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStrLiteralValue(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 uses io {\n  val s: str = \"A\\n\"\n  print(s)\n  return 0\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "A\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStrParam(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun echo(x: str): i32 uses io {\n  print(x)\n  return 0\n}\nfun " +
		"main(): i32 uses io {\n  return echo(\"Hi\\n\")\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "Hi\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildStrReturn(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun greet(): str {\n  return \"Hey\\n\"\n}\nfun main(): i32 uses " +
		"io {\n  print(greet())\n  return 0\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "Hey\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildExpressionBodiedFunction(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "func add(a: Int, b: Int) -> Int = a + b\nfunc main() -> Int = add(40, 2)\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMakeI32Slice(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun main(): i32 uses alloc, mem {\n  var xs: []i32 = " +
		"make_i32(3)\n  xs[0] = 10\n  xs[1] = 20\n  xs[2] = xs[0] + " +
		"xs[1]\n  return xs[2]\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 30 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMakeZeroLengthSlices(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("module test.zero_length_slices\n\nfun count_i32(values: []i32)" +
		" -> Int uses mem {\n  var count: Int = 0\n  for value in " +
		"values {\n    count = count + 1\n  }\n  return count\n}\nfun " +
		"first_or_i32(values: []i32, fallback: Int) -> Int uses mem " +
		"{\n  for value in values {\n    return value\n  }\n  return " +
		"fallback\n}\nfun main(): i32 uses alloc, mem {\n  var bytes: []" +
		"u8 = make_u8(0)\n  for byte in bytes {\n    return 1\n  }\n  " +
		"var words: []u16 = make_u16(0)\n  for word in words {\n    " +
		"return 2\n  }\n  var ints: []i32 = make_i32(0)\n  for value in " +
		"ints {\n    return 3\n  }\n  if count_i32(ints) != 0 {\n    " +
		"return 6\n  }\n  if first_or_i32(ints, 42) != 42 {\n    return " +
		"7\n  }\n  var flags: []bool = make_bool(0)\n  for flag in " +
		"flags {\n    if flag {\n      return 4\n    }\n    return 5\n  " +
		"}\n  return 42\n}\n")
	stdout, exitCode := buildAndRunFiles(t, map[string]string{
		"test/zero_length_slices.tetra": src,
	}, "test/zero_length_slices.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMakeU8Print(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun main(): i32 uses alloc, io, mem {\n  var xs: []u8 = " +
		"make_u8(2)\n  xs[0] = 65\n  xs[1] = 66\n  print(xs)\n  return " +
		"0\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "AB" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildMmioSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun main(): i32 uses alloc, capability, io, mem, mmio {\n  " +
		"var out: i32 = 0\n  unsafe {\n    let io: cap.io = " +
		"core.cap_io()\n    let p: ptr = core.alloc_bytes(4)\n    let " +
		"_w: i32 = core.mmio_write_i32(p, 123, io)\n    out = " +
		"core.mmio_read_i32(p, io)\n  }\n  return out\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 123 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildIslandsDebugDoubleFreeRejectedBySemantics(t *testing.T) {
	requireCheckFileErrorContains(t, `
func alias(isl: island) -> island:
    return isl

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(64)
        let other: island = alias(isl)
        free(isl)
        free(other)
    }
    return 0
`, "cannot use freed resource 'other'")
}

func TestBuildScopedIslandAutoFreeRunsInDebugAndNonDebug(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun main(): i32 uses alloc, islands, mem {\n  var out: i32 = " +
		"0\n  island(64) as isl {\n    var xs: []u8 = " +
		"core.island_make_u8(isl, 1)\n    xs[0] = 7\n    out = xs[0]\n  " +
		"}\n  return out\n}\n")
	for _, tc := range []struct {
		name string
		opt  BuildOptions
	}{
		{name: "non_debug", opt: BuildOptions{Jobs: 1}},
		{name: "debug", opt: BuildOptions{Jobs: 1, IslandsDebug: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, exitCode := buildAndRunWithOptions(t, src, tc.opt)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != 7 {
				t.Fatalf("exit code mismatch: %d", exitCode)
			}
		})
	}
}

func TestBuildIslandsDebugOverflowFails(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun main(): i32 uses alloc, islands, mem {\n  island(64) as " +
		"isl {\n    var xs: []u8 = core.island_make_u8(isl, 65)\n    " +
		"xs[0] = 1\n  }\n  return 0\n}\n")
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{Jobs: 1, IslandsDebug: true})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildSliceBoundsCheck(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fun main(): i32 uses alloc, mem {\n  var xs: []i32 = " +
		"make_i32(2)\n  xs[2] = 1\n  return 0\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildRawPtrAddNegativeOffsetBoundsDiagnostic(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let q: ptr = core.ptr_add(p, 0 - 1, mem)
        let _: UInt8 = core.store_u8(q, 7, mem)
        return 0
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", exitCode)
	}
}

func TestBuildRawPtrAddAllocationUpperBoundDiagnostic(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let q: ptr = core.ptr_add(p, 4, mem)
        let _: UInt8 = core.store_u8(q, 7, mem)
        return 0
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", exitCode)
	}
}

func TestBuildRawPtrAddDirectI32OffsetAccess(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let _: Int = core.store_i32(core.ptr_add(p, 4, mem), 42, mem)
        return core.load_i32(core.ptr_add(p, 4, mem), mem)
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildRawPtrAddDirectPtrOffsetAccess(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        let stored: ptr = core.store_ptr(core.ptr_add(p, 8, mem), p, mem)
        let loaded: ptr = core.load_ptr(core.ptr_add(p, 8, mem), mem)
        return 42
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildRawAllocZeroSizeDiagnostic(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int
uses alloc, mem:
    unsafe:
        let _: ptr = core.alloc_bytes(0)
        return 0
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", exitCode)
	}
}

func TestBuildRawStoreI32AllocationBaseWidthDiagnostic(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(3)
        let _: Int = core.store_i32(p, 123, mem)
        return 0
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", exitCode)
	}
}

func TestBuildRawStorePtrAllocationBaseWidthDiagnostic(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(7)
        let _: ptr = core.store_ptr(p, p, mem)
        return 0
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", exitCode)
	}
}

func TestBuildMemoryHelpersRejectNegativeLength(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(
			root,
			"examples",
			"core",
			"memory",
			"core_memory_negative_length_smoke.tetra",
		),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: got %d, want 2", exitCode)
	}
}

func TestBuildNonZeroReturn(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 uses io {\n  print(\"Done\\n\");\n  return 7;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "Done\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 7 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildLetExpr(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  let x: i32 = 2 + 3;\n  return x;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildIfElseReturn(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  if (0) { return 1; } else { return 2; }\n  return 3;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildWhileCounter(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fn main() -> i32 {\n  var n: i32 = 3;\n  var acc: i32 = 0;\n  " +
		"while (n) {\n    acc = acc + 1;\n    n = n - 1;\n  }\n  return " +
		"acc;\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 3 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildLessThan(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  if (2 < 3) { return 1; }\n  return 0;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildEqEqFalse(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fn main() -> i32 {\n  if (2 == 3) { return 1; }\n  return 0;\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildWhileLess(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("fn main() -> i32 {\n  var i: i32 = 0;\n  while (i < 3) {\n    " +
		"i = i + 1;\n  }\n  return i;\n}\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 3 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildNewStyleNoSemicolons(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := "fun main(): i32 {\n  var x: i32 = 2 + 3\n  return x\n}\n"
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildFlowSyntaxHelloWithAliases(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("func main() -> Int\nuses io:\n  let msg: String = \"Flow\\n\"\n  " +
		"let ok: Bool = true\n  print(msg)\n  if ok:\n    return 0\n  " +
		"else:\n    return 1\n  return 1\n")
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "Flow\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestBuildBoolBranchSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("func main() -> Int:\n  let ok: Bool = true\n  if ok && (3 > 2)" +
		":\n    return 42\n  return 1\n")
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildForRangeSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("func main() -> Int:\n  var total: Int = 0\n  for i in 0..<11:" +
		"\n    total = total + i\n  return total\n")
	_, code := buildAndRun(t, src)
	if code != 55 {
		t.Fatalf("exit code mismatch: got %d, want 55", code)
	}
}

func TestBuildEnumMatchSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("enum Color:\n  case red\n  case green\n  case blue\n\nfunc main()" +
		" -> Int:\n  let color: Color = Color.green\n  match color:\n  " +
		"case Color.red:\n    return 1\n  case Color.green:\n    return " +
		"42\n  case _:\n    return 0\n")
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildEnumMatchExhaustiveNoDefaultSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("enum Color:\n  case red\n  case green\n\nfunc main() -> Int:\n  " +
		"let color: Color = Color.green\n  match color:\n  case " +
		"Color.red:\n    return 1\n  case Color.green:\n    return 42\n")
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildEnumPayloadMatchSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("enum Result:\n  case ok(Int)\n  case err(Int, Int)\n  case " +
		"empty\n\nfunc main() -> Int:\n  let result: Result = " +
		"Result.ok(42)\n  match result:\n  case Result.ok(value):\n    " +
		"return value\n  case Result.err(code, detail):\n    return " +
		"code + detail\n  case Result.empty:\n    return 0\n")
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildEnumPayloadMultiValueCaseSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("enum Result:\n  case ok(Int)\n  case err(Int, Int)\n\nfunc " +
		"main() -> Int:\n  let result: Result = Result.err(40, 2)\n  " +
		"match result:\n  case Result.ok(value):\n    return value\n  " +
		"case Result.err(code, detail):\n    return code + detail\n")
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildEnumPayloadNoPayloadCaseInWideEnumSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("enum Result:\n  case ok(Int)\n  case empty\n\nfunc main() -> " +
		"Int:\n  let result: Result = Result.empty\n  match result:\n  " +
		"case Result.ok(value):\n    return value\n  case Result.empty:" +
		"\n    return 42\n")
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildEnumPayloadActorMessageDataSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("enum CounterMsg:\n  case inc(Int)\n  case reset\n\nfunc " +
		"handle(msg: CounterMsg) -> Int:\n  match msg:\n  case " +
		"CounterMsg.inc(delta):\n    return delta\n  case " +
		"CounterMsg.reset:\n    return 0\n\nfunc main() -> Int:\n  let " +
		"msg: CounterMsg = CounterMsg.inc(42)\n  return handle(msg)\n")
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildMatchExpressionEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("enum Result:\n  case ok(Int)\n  case err(Int)\n\nfunc main() -> " +
		"Int:\n  let result: Result = Result.ok(42)\n  let score: Int " +
		"= match result:\n  case Result.ok(value):\n    value\n  case " +
		"Result.err(code):\n    code\n  return score\n")
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestMatchExpressionRequiresExhaustiveCases(t *testing.T) {
	src := ("enum Result:\n  case ok(Int)\n  case err(Int)\n\nfunc main() -> " +
		"Int:\n  let result: Result = Result.ok(42)\n  let score: Int " +
		"= match result:\n  case Result.ok(value):\n    value\n  return " +
		"score\n")
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected non-exhaustive match expression diagnostic")
	} else if !strings.Contains(err.Error(), "match expression must be exhaustive") {
		t.Fatalf("error = %v", err)
	}
}

func TestMatchExpressionRejectsMismatchedCaseTypes(t *testing.T) {
	src := ("enum Result:\n  case ok(Int)\n  case err(Int)\n\nfunc main() -> " +
		"Int:\n  let result: Result = Result.ok(42)\n  let score: Int " +
		"= match result:\n  case Result.ok(value):\n    value\n  case " +
		"Result.err(code):\n    \"bad\"\n  return score\n")
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected match expression case type diagnostic")
	} else if !strings.Contains(err.Error(), "match expression case type mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestMatchExpressionBindingScopeDiagnostic(t *testing.T) {
	src := ("enum Result:\n  case ok(Int)\n  case err(Int)\n\nfunc main() -> " +
		"Int:\n  let result: Result = Result.ok(42)\n  let score: Int " +
		"= match result:\n  case Result.ok(value):\n    value\n  case " +
		"Result.err(code):\n    code\n  return value\n")
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected match expression binding scope diagnostic")
	} else if !strings.Contains(err.Error(), "out of scope") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildIfLetEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("enum Result:\n  case ok(Int)\n  case err(Int)\n\nfunc main() -> " +
		"Int:\n  let result: Result = Result.ok(42)\n  if let " +
		"Result.ok(value) = result:\n    return value\n  else:\n    " +
		"return 0\n")
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildIfLetEnumNoPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("enum Result:\n  case ok(Int)\n  case empty\n\nfunc main() -> " +
		"Int:\n  let result: Result = Result.empty\n  if let " +
		"Result.empty = result:\n    return 42\n  else:\n    return 0\n")
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildMatchGuardEnumPayloadSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("enum Result:\n  case ok(Int)\n  case err(Int)\n\nfunc main() -> " +
		"Int:\n  let result: Result = Result.ok(42)\n  match result:\n  " +
		"case Result.ok(value) if value > 40:\n    return value\n  " +
		"case Result.ok(other):\n    return 1\n  case Result.err(code):" +
		"\n    return code\n")
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestEnumMatchExhaustiveThreeCasesNoDefaultCheck(t *testing.T) {
	src := ("enum Color:\n  case red\n  case green\n  case blue\n\nfunc main()" +
		" -> Int:\n  let color: Color = Color.blue\n  match color:\n  " +
		"case Color.red:\n    return 1\n  case Color.green:\n    return " +
		"2\n  case Color.blue:\n    return 3\n")
	if err := testkit.CheckProgram(src); err != nil {
		t.Fatalf("unexpected non-exhaustive enum diagnostic: %v", err)
	}
}

func TestEnumMatchMissingCaseStillNeedsReturn(t *testing.T) {
	src := ("enum Color:\n  case red\n  case green\n\nfunc main() -> Int:\n  " +
		"let color: Color = Color.green\n  match color:\n  case " +
		"Color.red:\n    return 1\n")
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected missing return for non-exhaustive enum match")
	} else if !strings.Contains(err.Error(), "must end with return") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumPayloadConstructorArityDiagnostic(t *testing.T) {
	src := ("enum Result:\n  case ok(Int)\n\nfunc main() -> Int:\n  let " +
		"result: Result = Result.ok()\n  return 0\n")
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected enum payload arity diagnostic")
	} else if !strings.Contains(err.Error(), "expects 1 payload argument") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumPayloadConstructorTypeDiagnostic(t *testing.T) {
	src := ("enum Result:\n  case ok(Int)\n\nfunc main() -> Int:\n  let " +
		"result: Result = Result.ok(\"nope\")\n  return 0\n")
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected enum payload type diagnostic")
	} else if !strings.Contains(err.Error(), "payload 1 expects 'i32', got 'str'") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumPayloadBindingScopeDiagnostic(t *testing.T) {
	src := ("enum Result:\n  case ok(Int)\n  case empty\n\nfunc main() -> " +
		"Int:\n  let result: Result = Result.ok(1)\n  match result:\n  " +
		"case Result.ok(value):\n    let inside: Int = value\n  case " +
		"Result.empty:\n    let other: Int = 0\n  return value\n")
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected enum payload binding scope diagnostic")
	} else if !strings.Contains(err.Error(), "out of scope") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumNoPayloadConstructorCallDiagnostic(t *testing.T) {
	src := ("enum Color:\n  case red\n\nfunc main() -> Int:\n  let color: " +
		"Color = Color.red()\n  return 0\n")
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected no-payload enum constructor diagnostic")
	} else if !strings.Contains(err.Error(), "has no payload; use 'Color.red'") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumPayloadPatternArityDiagnostic(t *testing.T) {
	src := ("enum Result:\n  case ok(Int, Int)\n\nfunc main() -> Int:\n  let " +
		"result: Result = Result.ok(1, 2)\n  match result:\n  case " +
		"Result.ok(value):\n    return value\n")
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected enum payload pattern arity diagnostic")
	} else if !strings.Contains(err.Error(), "pattern expects 2 binding(s), got 1") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumPayloadPatternRequiresPayloadSyntaxDiagnostic(t *testing.T) {
	src := `
enum Result:
  case ok(Int)
  case empty

func main() -> Int:
  let result: Result = Result.ok(1)
  let score: Int = match result:
  case Result.ok:
    1
  case Result.empty:
    0
  return score
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected enum payload syntax diagnostic")
	} else if !strings.Contains(err.Error(), "carries 1 payload value(s); use 'Result.ok(value1)'") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumPayloadGuardedBarePatternStillRequiresDestructuring(t *testing.T) {
	src := `
enum Result:
  case ok(Int)
  case empty

func main() -> Int:
  let result: Result = Result.ok(1)
  match result:
  case Result.ok if true:
    return 1
  case Result.ok(value):
    return value
  case Result.empty:
    return 0
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected guarded enum payload syntax diagnostic")
	} else if !strings.Contains(
		err.Error(),
		"requires payload arguments",
	) && !strings.Contains(
		err.Error(),
		"carries 1 payload value(s); use 'Result.ok(value1)'",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestMatchExpressionGuardedEnumPayloadCaseIsNotExhaustive(t *testing.T) {
	src := `
enum Result:
  case ok(Int)
  case empty

func main() -> Int:
  let result: Result = Result.ok(1)
  let score: Int = match result:
  case Result.ok(value) if value > 0:
    value
  case Result.empty:
    0
  return score
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected non-exhaustive guarded enum payload match expression diagnostic")
	} else if !strings.Contains(err.Error(), "match expression must be exhaustive") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumMatchGuardedCasesDoNotCountAsExhaustive(t *testing.T) {
	src := `
enum Result:
  case ok(Int)
  case empty

func main() -> Int:
  let result: Result = Result.ok(1)
  match result:
  case Result.ok(value) if value > 0:
    return value
  case Result.empty:
    return 0
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected missing return for guarded non-exhaustive enum match")
	} else if !strings.Contains(err.Error(), "must end with return") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumMatchDuplicateUnguardedPayloadCaseDiagnostic(t *testing.T) {
	src := `
enum Result:
  case ok(Int)
  case empty

func main() -> Int:
  let result: Result = Result.ok(1)
  match result:
  case Result.ok(value):
    return value
  case Result.ok(other):
    return other
  case Result.empty:
    return 0
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected duplicate enum payload case diagnostic")
	} else if !strings.Contains(err.Error(), "duplicate match pattern") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumMatchDefaultMustBeLastDiagnostic(t *testing.T) {
	src := `
enum Color:
  case red
  case green

func main() -> Int:
  let color: Color = Color.red
  match color:
  case _:
    return 0
  case Color.red:
    return 1
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected default ordering diagnostic")
	} else if !strings.Contains(err.Error(), "match default must be last") {
		t.Fatalf("error = %v", err)
	}
}

func TestMatchExpressionDefaultMustBeLastDiagnostic(t *testing.T) {
	src := `
enum Color:
  case red
  case green

func main() -> Int:
  let color: Color = Color.red
  let score: Int = match color:
  case _:
    0
  case Color.red:
    1
  return score
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected match expression default ordering diagnostic")
	} else if !strings.Contains(err.Error(), "match default must be last") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumMatchRejectsWrongEnumCaseDiagnostic(t *testing.T) {
	src := `
enum Result:
  case ok(Int)
  case empty
enum Other:
  case ok(Int)

func main() -> Int:
  let result: Result = Result.ok(1)
  match result:
  case Other.ok(value):
    return value
  case Result.empty:
    return 0
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected wrong enum case diagnostic")
	} else if !strings.Contains(
		err.Error(),
		"enum pattern type mismatch",
	) && !strings.Contains(
		err.Error(),
		"match pattern type mismatch",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumNoPayloadPatternRejectsPayloadSyntaxDiagnostic(t *testing.T) {
	src := `
enum Result:
  case ok(Int)
  case empty

func main() -> Int:
  let result: Result = Result.empty
  match result:
  case Result.ok(value):
    return value
  case Result.empty(value):
    return value
`
	if err := testkit.CheckProgram(src); err == nil {
		t.Fatalf("expected no-payload enum pattern diagnostic")
	} else if !strings.Contains(err.Error(), "has no payload; use 'Result.empty'") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnumPayloadPatternDuplicateBindingParseDiagnostic(t *testing.T) {
	src := ("enum Result:\n  case ok(Int, Int)\n\nfunc main() -> Int:\n  let " +
		"result: Result = Result.ok(1, 2)\n  match result:\n  case " +
		"Result.ok(value, value):\n    return value\n")
	if _, err := Parse([]byte(src)); err == nil {
		t.Fatalf("expected duplicate payload binding parse diagnostic")
	} else if !strings.Contains(err.Error(), "duplicate enum payload binding 'value'") {
		t.Fatalf("error = %v", err)
	}
}

func TestCrossModuleEnumPayloadConstructorAndMatchCheckLower(t *testing.T) {
	files := map[string]string{
		"lib/result.tetra": "module lib.result\n\npub enum Result:\n  case ok(Int)\n  case err(Int)\n",
		"app/main.tetra": ("module app.main\nimport lib.result as res\n\nfunc main() -> " +
			"Int:\n  let result: res.Result = res.Result.ok(42)\n  let " +
			"score: Int = match result:\n  case res.Result.ok(value):\n    " +
			"value\n  case res.Result.err(code):\n    code\n  return score\n"),
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	world, err := LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, err := LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestBuildCrossModuleNoPayloadEnumMatchSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"lib/result.tetra": "module lib.result\n\npub enum Result:\n  case ok(Int)\n  case empty\n",
		"app/main.tetra": ("module app.main\nimport lib.result as res\n\nfunc main() -> " +
			"Int:\n  let result: res.Result = res.Result.empty\n  match " +
			"result:\n  case res.Result.ok(value):\n    return value\n  " +
			"case res.Result.empty:\n    return 42\n"),
	}

	_, code := buildAndRunFiles(t, files, "app/main.tetra")
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildIntMatchSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := ("func main() -> Int:\n  let value: Int = 7\n  match value:\n  " +
		"case 1:\n    return 1\n  case 7:\n    return 42\n  case _:\n    " +
		"return 0\n")
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildTypedErrorsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "smoke", "errors", "typed_errors_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildEnumPayloadSmokeFile(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "smoke", "types", "enum_payload_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildAsyncSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "async", "async_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildTaskSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "tasks", "task_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreMathSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "data", "core_math_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreMemorySmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "memory", "core_memory_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreStringsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "data", "core_strings_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreSlicesSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "data", "core_slices_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreIOSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "platform", "core_io_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreTestingSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "runtime", "core_testing_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreCollectionsSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	root := projectRoot(t)
	stdout, exitCode := buildAndRunFile(
		t,
		filepath.Join(root, "examples", "core", "data", "core_collections_smoke.tetra"),
	)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCoreCollectionsGenericCacheKeyIncludesMonomorphizedFuncs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))
	writeTestFiles(t, tmp, map[string]string{
		"app/main.tetra": `module app.main
import lib.core.collections as collections

func main() -> Int
uses alloc, mem:
    var xs: []i32 = core.make_i32(1)
    xs[0] = 42
    return collections.len_i32(xs)
`,
	})
	opt := BuildOptions{
		ProjectRoot:     tmp,
		DependencyRoots: []ModuleRoot{{Root: projectRoot(t)}},
	}
	if _, err := BuildFileWithStatsOpt(
		entry,
		filepath.Join(tmp, "plain"),
		"linux-x64",
		opt,
	); err != nil {
		t.Fatalf("plain collections build: %v", err)
	}

	writeTestFiles(t, tmp, map[string]string{
		"app/main.tetra": `module app.main
import lib.core.collections as collections

func main() -> Int
uses alloc, mem:
    var xs: []i32 = core.make_i32(1)
    xs[0] = 42
    let vec: collections.Vec<Int> = collections.vec_from_slice(xs)
    if collections.vec_len(vec) == 1:
        return 42
    return 0
`,
	})
	outPath := filepath.Join(tmp, "generic")
	if _, err := BuildFileWithStatsOpt(entry, outPath, "linux-x64", opt); err != nil {
		t.Fatalf("generic collections build after plain cache entry: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	_, exitCode := runBinary(t, outPath)
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

// ---- distributed_actor_runtime_test.go ----

func TestCollectDistributedActorRuntimeUsage(t *testing.T) {
	checked := checkedDistributedActorProgram(t, `
func worker() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let _connected: Int = core.actor_node_connect(2, 5010)
    let _peer: actor = core.spawn_remote(2, "worker")
    return core.actor_node_status(2)
`)

	used, _ := collectDistributedActorRuntimeUsagePosition(checked)
	if !used {
		t.Fatalf("distributed actor runtime usage was not detected")
	}

	actorsUsed, entries, _, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collectActorEntries: %v", err)
	}
	if !actorsUsed {
		t.Fatalf("actor runtime usage was not detected")
	}
	if !containsString(entries, "worker") {
		t.Fatalf("actor entries = %v, want remote spawn target worker", entries)
	}
}

func TestRequiredDistributedActorRuntimeSymbols(t *testing.T) {
	got := map[string]struct{}{}
	for _, name := range requiredDistributedActorRuntimeSymbols() {
		got[name] = struct{}{}
	}
	wantReturnSlots := map[string]int{
		"__tetra_actor_node_connect": 1,
		"__tetra_actor_spawn_remote": runtimeabi.ActorHandleABI().RefSlots,
		"__tetra_actor_node_status":  1,
	}
	for _, name := range []string{
		"__tetra_actor_node_connect",
		"__tetra_actor_spawn_remote",
		"__tetra_actor_node_status",
	} {
		if _, ok := got[name]; !ok {
			t.Fatalf("required distributed actor runtime symbols missing %q", name)
		}
		sig, ok := runtimeObjectSignature(name)
		if !ok {
			t.Fatalf("missing runtime signature for %q", name)
		}
		if sig.returnSlots != wantReturnSlots[name] {
			t.Fatalf("%s return slots = %d, want %d", name, sig.returnSlots, wantReturnSlots[name])
		}
	}
}

func TestRuntimeObjectValidationRejectsMissingDistributedActorSymbols(t *testing.T) {
	obj := &Object{Symbols: runtimeObjectSymbols(requiredActorRuntimeSymbols())}
	annotateRuntimeObjectSignatures(obj)
	err := validateDistributedActorRuntimeObject(obj)
	if err == nil {
		t.Fatalf("expected missing distributed actor runtime symbol error")
	}
	if !strings.Contains(
		err.Error(),
		"runtime object missing required symbol '__tetra_actor_node_connect'",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestDistributedActorRuntimeRejectsUnsupportedNativeTargets(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "distributed_actor_status.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses actors, runtime:
    return core.actor_node_status(2)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	for _, target := range []string{"macos-x64", "windows-x64"} {
		t.Run(target, func(t *testing.T) {
			outPath := filepath.Join(tmp, "distributed-"+target)
			_, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1})
			if err == nil {
				t.Fatalf("expected unsupported distributed actors runtime diagnostic")
			}
			want := "distributed actors runtime not supported on " + target
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want %q", err, want)
			}
		})
	}
}

func TestDistributedActorRuntimeBuildsWithLinuxBuiltinRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "distributed_actor_linux.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let _connected: Int = core.actor_node_connect(1, 4599)
    let _peer: actor = core.spawn_remote(2, "worker")
    return core.actor_node_status(2)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "distributed-actor-linux")
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"linux-x64",
		BuildOptions{Jobs: 1},
	); err != nil {
		t.Fatalf("linux distributed actor build: %v", err)
	}
}

func checkedDistributedActorProgram(t *testing.T, src string) *semantics.CheckedProgram {
	t.Helper()
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	world, err := LoadWorld(srcPath)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	return checked
}

// ---- feature_surface_audit_test.go ----

func TestP22FeatureSurfaceAuditCoversMasterPlanCategoriesAndRegistryBoundaries(t *testing.T) {
	report := BuildP22FeatureSurfaceAudit()
	if report.SchemaVersion != featureSurfaceAuditSchemaV1 {
		t.Fatalf(
			"feature surface audit schema = %q, want %q",
			report.SchemaVersion,
			featureSurfaceAuditSchemaV1,
		)
	}
	if report.Scope != featureSurfaceAuditScopeP220 {
		t.Fatalf(
			"feature surface audit scope = %q, want %q",
			report.Scope,
			featureSurfaceAuditScopeP220,
		)
	}
	if err := ValidateP22FeatureSurfaceAudit(report); err != nil {
		t.Fatalf("ValidateP22FeatureSurfaceAudit: %v", err)
	}

	rows := map[FeatureSurfaceAuditCategory]FeatureSurfaceAuditRow{}
	for _, row := range report.Rows {
		if row.Category == "" || row.Decision == "" || len(row.Evidence) == 0 ||
			len(row.Boundaries) == 0 ||
			len(row.RequiredPromotionEvidence) == 0 {
			t.Fatalf("P22.0 row missing required metadata: %#v", row)
		}
		rows[row.Category] = row
	}
	for _, category := range p22FeatureSurfaceAuditCategories() {
		if _, ok := rows[category]; !ok {
			t.Fatalf("P22.0 audit missing category %s: %#v", category, report.Rows)
		}
	}

	p22AssertFeatureSurfaceRow(
		t,
		rows[FeatureSurfaceFirstClassCallables],
		[]string{
			"language.callable-mvp",
			"language.callable-level1",
			"language.callable-level2",
			"language.full-first-class-callables",
		},
		[]string{
			"fixed 4-slot callable handle",
			"mutable by-reference capture",
			"thread-boundary callable escape",
			"P22.1",
		},
	)
	p22AssertFeatureSurfaceRow(
		t,
		rows[FeatureSurfaceClosures],
		[]string{"language.callable-level2", "language.full-first-class-callables"},
		[]string{
			"safe by-value captures",
			"pointer/resource capture",
			"generic closure",
			"same-branch evidence",
		},
	)
	p22AssertFeatureSurfaceRow(
		t,
		rows[FeatureSurfaceProtocolsTraitObjects],
		[]string{"language.protocol-conformance-mvp", "language.protocol-bound-generics-static"},
		[]string{
			"static conformance",
			"no witness tables",
			"trait objects",
			"runtime protocol values",
			"P22.2",
		},
	)
	p22AssertFeatureSurfaceRow(
		t,
		rows[FeatureSurfaceRuntimeGenerics],
		[]string{"language.generics-mvp", "language.protocol-bound-generics-static"},
		[]string{
			"statically monomorphized",
			"runtime generic values",
			"explicit type arguments",
			"generic structs",
			"higher-ranked generics",
		},
	)
	p22AssertFeatureSurfaceRow(
		t,
		rows[FeatureSurfaceAdvancedEnumsPatternMatching],
		[]string{"language.enum-payload-match"},
		[]string{
			"positional enum payload",
			"nested destructuring patterns",
			"guard expansion",
			"richer payload algebra",
		},
	)
	p22AssertFeatureSurfaceRow(t, rows[FeatureSurfaceAsyncTypedErrors],
		[]string{"language.task-handles-mvp", "language.resource-lifetime-mvp"},
		[]string{"try await", "typed-error", "await try", "cancellation"})
	p22AssertFeatureSurfaceRow(
		t,
		rows[FeatureSurfaceStructuredConcurrency],
		[]string{
			"actors.task-transfer-safety",
			"language.task-handles-mvp",
			"actors.distributed-runtime",
		},
		[]string{
			"conservative local MVP",
			"full cancellation",
			"full race-safety proof",
			"broader structured-concurrency guarantees",
		},
	)
	p22AssertFeatureSurfaceRow(t, rows[FeatureSurfaceModulesPackages],
		[]string{"language.globals-properties-capsule-mvp", "eco.local-package-lifecycle"},
		[]string{"local package lifecycle", "capsule metadata", "distributed EcoNet"})
	p22AssertFeatureSurfaceRow(t, rows[FeatureSurfaceMacrosMetaprogramming],
		nil,
		[]string{"no current macro/metaprogramming feature", "post-v1", "same-branch evidence"})
	p22AssertFeatureSurfaceRow(
		t,
		rows[FeatureSurfaceUISurface],
		[]string{
			"ui.metadata-v1",
			"ui.surface-core",
			"ui.surface-linux-x64",
			"ui.surface-web-wasm",
			"ui.native-runtime",
			"ui.platform-runtime",
			"ui.surface-macos-x64",
			"ui.surface-windows-x64",
			"ui.surface-wasm32-wasi",
		},
		[]string{"Linux-x64", "wasm32-web", "macOS", "Windows", "cross-platform"},
	)
	p22AssertFeatureSurfaceRow(
		t,
		rows[FeatureSurfaceEcoCapsules],
		[]string{
			"language.globals-properties-capsule-mvp",
			"eco.local-package-lifecycle",
			"eco.distributed-network",
		},
		[]string{"local Eco", "proof-carrying capsules", "distributed EcoNet", "post-v1"},
	)

	for _, nonClaim := range []string{
		"no full v1 language guarantee is claimed",
		"no runtime generic values are claimed",
		"no trait objects or runtime protocol values are claimed",
		"no macro/metaprogramming system is claimed",
		"no full structured concurrency guarantee is claimed",
		"no cross-platform production UI runtime is claimed",
		"no distributed EcoNet or proof-carrying capsule promotion is claimed",
		"no performance claim is made",
		"safe-program semantics do not change",
	} {
		if !p22FeatureSurfaceHasString(report.NonClaims, nonClaim) {
			t.Fatalf("P22.0 audit missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP22FeatureSurfaceAuditRejectsFakePromotionAndDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*FeatureSurfaceAuditReport)
		want   string
	}{
		{
			name: "report level promotion without same branch evidence",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.PromotedWithoutSameBranchEvidence = true
			},
			want: "same-branch",
		},
		{
			name: "row promotion without same branch evidence",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.Rows[0].PromotedInThisAudit = true
				report.Rows[0].SameBranchEvidence = false
			},
			want: "same-branch",
		},
		{
			name: "full v1 claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.FullV1GuaranteesClaimed = true
			},
			want: "full v1",
		},
		{
			name: "runtime generic values claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.RuntimeGenericValuesClaimed = true
			},
			want: "runtime generic",
		},
		{
			name: "trait objects claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.TraitObjectsClaimed = true
			},
			want: "trait object",
		},
		{
			name: "macro system claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.MacroSystemClaimed = true
			},
			want: "macro",
		},
		{
			name: "structured concurrency claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.StructuredConcurrencyClaimed = true
			},
			want: "structured concurrency",
		},
		{
			name: "cross platform UI runtime claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.CrossPlatformUIRuntimeClaimed = true
			},
			want: "cross-platform",
		},
		{
			name: "distributed Eco claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.DistributedEcoClaimed = true
			},
			want: "distributed Eco",
		},
		{
			name: "proof carrying capsules claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.ProofCarryingCapsulesClaimed = true
			},
			want: "proof-carrying",
		},
		{
			name: "performance claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
		{
			name: "safe semantics change",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.SafeSemanticsChanged = true
			},
			want: "safe-program semantics",
		},
		{
			name: "missing category",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.Rows = report.Rows[1:]
			},
			want: "missing category",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.Rows[0].Evidence = []string{"TODO fill this in"}
			},
			want: "placeholder",
		},
		{
			name: "unknown feature ID",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.Rows[0].FeatureIDs = append(
					report.Rows[0].FeatureIDs,
					"language.fake-runtime-generics",
				)
				report.Rows[0].RegistryStatuses["language.fake-runtime-generics"] = FeatureStatusCurrent
			},
			want: "unknown feature",
		},
		{
			name: "registry status drift",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.Rows[0].RegistryStatuses[report.Rows[0].FeatureIDs[0]] = FeatureStatusPostV1
			},
			want: "registry status drift",
		},
		{
			name: "macro row invents current feature",
			mutate: func(report *FeatureSurfaceAuditReport) {
				for i := range report.Rows {
					if report.Rows[i].Category == FeatureSurfaceMacrosMetaprogramming {
						report.Rows[i].FeatureIDs = []string{"language.macro-system"}
						report.Rows[i].RegistryStatuses = map[string]FeatureStatus{
							"language.macro-system": FeatureStatusCurrent,
						}
						return
					}
				}
			},
			want: "unknown feature",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := cloneFeatureSurfaceAuditReport(BuildP22FeatureSurfaceAudit())
			tc.mutate(&report)
			err := ValidateP22FeatureSurfaceAudit(report)
			if err == nil {
				t.Fatalf("ValidateP22FeatureSurfaceAudit accepted fake report: %#v", report)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func p22AssertFeatureSurfaceRow(
	t *testing.T,
	row FeatureSurfaceAuditRow,
	featureIDs []string,
	wants []string,
) {
	t.Helper()
	for _, id := range featureIDs {
		if !p22FeatureSurfaceHasString(row.FeatureIDs, id) {
			t.Fatalf("row %s missing feature id %s: %#v", row.Category, id, row)
		}
		if row.RegistryStatuses[id] == "" {
			t.Fatalf(
				"row %s missing registry status for %s: %#v",
				row.Category,
				id,
				row.RegistryStatuses,
			)
		}
	}
	combined := row.Name + " " + row.Decision + " " + strings.Join(
		row.Evidence,
		" ",
	) + " " + strings.Join(
		row.Boundaries,
		" ",
	) + " " + strings.Join(
		row.RequiredPromotionEvidence,
		" ",
	)
	for _, want := range wants {
		if !strings.Contains(combined, want) {
			t.Fatalf("row %s missing %q: %#v", row.Category, want, row)
		}
	}
}

func cloneFeatureSurfaceAuditReport(report FeatureSurfaceAuditReport) FeatureSurfaceAuditReport {
	report.Rows = append([]FeatureSurfaceAuditRow{}, report.Rows...)
	for i := range report.Rows {
		report.Rows[i].FeatureIDs = append([]string{}, report.Rows[i].FeatureIDs...)
		report.Rows[i].Evidence = append([]string{}, report.Rows[i].Evidence...)
		report.Rows[i].Boundaries = append([]string{}, report.Rows[i].Boundaries...)
		report.Rows[i].RequiredPromotionEvidence = append(
			[]string{},
			report.Rows[i].RequiredPromotionEvidence...)
		registryStatuses := report.Rows[i].RegistryStatuses
		report.Rows[i].RegistryStatuses = map[string]FeatureStatus{}
		for id, status := range registryStatuses {
			report.Rows[i].RegistryStatuses[id] = status
		}
	}
	report.NonClaims = append([]string{}, report.NonClaims...)
	return report
}

// ---- ffi_target_diagnostics_test.go ----

func TestX32PointerFFIGateCoversOnlyUnverifiedPointerLikeAndFunctionPointerSpellings(t *testing.T) {
	for _, typeName := range []string{
		"fnptr",
		"fn(Int) -> Int",
	} {
		t.Run(typeName, func(t *testing.T) {
			if !targetExportedFFIRequiresX32PointerBoundaryGate("linux-x32", typeName) {
				t.Fatalf("linux-x32 FFI gate did not cover %q", typeName)
			}
			if targetExportedFFIRequiresX32PointerBoundaryGate("linux-x64", typeName) {
				t.Fatalf("linux-x64 FFI gate unexpectedly covered %q", typeName)
			}
		})
	}

	for _, typeName := range []string{
		"ptr",
		"rawptr",
		"nullable_ptr",
		"ref",
		"i32",
		"u32",
		"c_int",
		"c_uint",
		"bool",
		"usize",
		"isize",
		"size_t",
		"ssize_t",
		"native_int",
		"native_uint",
		"c_long",
		"c_ulong",
	} {
		t.Run(typeName, func(t *testing.T) {
			if targetExportedFFIRequiresX32PointerBoundaryGate("linux-x32", typeName) {
				t.Fatalf(
					"linux-x32 FFI gate should not cover scalar or source-level target-layout type %q",
					typeName,
				)
			}
		})
	}
}

func TestNativeTargetsRejectExportedAggregateFFIParameters(t *testing.T) {
	src := `repr(C) struct Pair:
    lo: Int
    hi: Int

@export("ffi_pair_c")
func ffi_pair(p: Pair) -> Int:
    return p.lo + p.hi
`

	for _, target := range []string{
		"linux-x86",
		"linux-x64",
		"linux-x32",
		"macos-x64",
		"windows-x64",
	} {
		t.Run(target, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_pair.t4")
			outPath := filepath.Join(tmp, "ffi_pair.tobj")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			_, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				target,
				BuildOptions{Emit: EmitLibrary, Jobs: 1},
			)
			if err == nil {
				t.Fatalf("expected aggregate FFI diagnostic")
			}
			for _, want := range []string{
				"exported function 'ffi_pair'",
				"parameter 'p'",
				"type 'Pair'",
				"aggregate C ABI is not supported on " + target,
			} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("diagnostic = %v, want substring %q", err, want)
				}
			}
			if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
				t.Fatalf("output object should not be written, stat error = %v", statErr)
			}
		})
	}
}

func TestNativeTargetsRejectExportedAggregateFFIReturns(t *testing.T) {
	src := `repr(C) struct Pair:
    lo: Int
    hi: Int

@export("ffi_make_pair_c")
func ffi_make_pair() -> Pair:
    return Pair(lo: 1, hi: 2)
`

	for _, target := range []string{
		"linux-x86",
		"linux-x64",
		"linux-x32",
		"macos-x64",
		"windows-x64",
	} {
		t.Run(target, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_make_pair.t4")
			outPath := filepath.Join(tmp, "ffi_make_pair.tobj")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			_, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				target,
				BuildOptions{Emit: EmitLibrary, Jobs: 1},
			)
			if err == nil {
				t.Fatalf("expected aggregate FFI diagnostic")
			}
			for _, want := range []string{
				"exported function 'ffi_make_pair'",
				"return type 'Pair'",
				"aggregate C ABI is not supported on " + target,
			} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("diagnostic = %v, want substring %q", err, want)
				}
			}
			if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
				t.Fatalf("output object should not be written, stat error = %v", statErr)
			}
		})
	}
}

func TestLinuxX86AndX32BuildExportedPointerFFIBoundaryObjects(t *testing.T) {
	src := `@export("ffi_ptr_identity_c")
func ffi_ptr_identity(p: ptr) -> ptr:
    return p

@export("ffi_rawptr_identity_c")
func ffi_rawptr_identity(p: rawptr) -> rawptr:
    return p

@export("ffi_nullable_ptr_identity_c")
func ffi_nullable_ptr_identity(p: nullable_ptr) -> nullable_ptr:
    return p

@export("ffi_nullable_ptr_null_c")
func ffi_nullable_ptr_null() -> nullable_ptr:
    return 0

@export("ffi_ref_identity_c")
func ffi_ref_identity(p: ref) -> ref:
    return p
`

	for _, target := range []string{"linux-x86", "linux-x32"} {
		t.Run(target, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_ptr_identity.t4")
			outPath := filepath.Join(tmp, "ffi_ptr_identity.tobj")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				target,
				BuildOptions{Emit: EmitLibrary, Jobs: 1},
			); err != nil {
				t.Fatalf("build %s pointer FFI object: %v", target, err)
			}
			obj, err := ReadObject(outPath)
			if err != nil {
				t.Fatalf("read object: %v", err)
			}
			if obj.Target != target {
				t.Fatalf("object target = %q, want %s", obj.Target, target)
			}
			if !abiSuiteObjectHasSymbolSignature(obj, "ffi_ptr_identity_c", 1, 1) {
				t.Fatalf(
					"%s object missing exported ffi_ptr_identity_c(1)->1 symbol: %#v",
					target,
					obj.Symbols,
				)
			}
			if !abiSuiteObjectHasSymbolSignature(obj, "ffi_rawptr_identity_c", 1, 1) {
				t.Fatalf(
					"%s object missing exported ffi_rawptr_identity_c(1)->1 symbol: %#v",
					target,
					obj.Symbols,
				)
			}
			if !abiSuiteObjectHasSymbolSignature(obj, "ffi_nullable_ptr_identity_c", 1, 1) {
				t.Fatalf(
					"%s object missing exported ffi_nullable_ptr_identity_c(1)->1 symbol: %#v",
					target,
					obj.Symbols,
				)
			}
			if !abiSuiteObjectHasSymbolSignature(obj, "ffi_nullable_ptr_null_c", 0, 1) {
				t.Fatalf(
					"%s object missing exported ffi_nullable_ptr_null_c(0)->1 symbol: %#v",
					target,
					obj.Symbols,
				)
			}
			if !abiSuiteObjectHasSymbolSignature(obj, "ffi_ref_identity_c", 1, 1) {
				t.Fatalf(
					"%s object missing exported ffi_ref_identity_c(1)->1 symbol: %#v",
					target,
					obj.Symbols,
				)
			}
		})
	}
}

func TestLinuxX86AndX32RejectRefNullReturnWithoutObject(t *testing.T) {
	src := `@export("ffi_ref_null_c")
func ffi_ref_null() -> ref:
    return 0
`

	for _, target := range []string{"linux-x86", "linux-x32"} {
		t.Run(target, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_ref_null.t4")
			outPath := filepath.Join(tmp, "ffi_ref_null.tobj")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			_, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				target,
				BuildOptions{Emit: EmitLibrary, Jobs: 1},
			)
			if err == nil {
				t.Fatalf("expected %s ref null-return diagnostic", target)
			}
			for _, want := range []string{
				"type mismatch",
				"expected 'ref'",
				"got 'i32'",
			} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("diagnostic = %v, want substring %q", err, want)
				}
			}
			if strings.Contains(err.Error(), "pointer C ABI boundary") {
				t.Fatalf("diagnostic = %v, should not be reported as a pointer C ABI boundary", err)
			}
			if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
				t.Fatalf("output object should not be written, stat error = %v", statErr)
			}
		})
	}
}

func TestLinuxFamilyBuildsExportedCIntFFIScalarObjects(t *testing.T) {
	src := `@export("ffi_c_int_identity_c")
func ffi_c_int_identity(n: c_int) -> c_int:
    return n
`

	for _, target := range []string{"linux-x86", "linux-x64", "linux-x32"} {
		t.Run(target, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_c_int_identity.t4")
			outPath := filepath.Join(tmp, "ffi_c_int_identity.tobj")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				target,
				BuildOptions{Emit: EmitLibrary, Jobs: 1},
			); err != nil {
				t.Fatalf("build %s c_int FFI object: %v", target, err)
			}
			obj, err := ReadObject(outPath)
			if err != nil {
				t.Fatalf("read object: %v", err)
			}
			if obj.Target != target {
				t.Fatalf("object target = %q, want %s", obj.Target, target)
			}
			if !abiSuiteObjectHasSymbolSignature(obj, "ffi_c_int_identity_c", 1, 1) {
				t.Fatalf(
					"%s object missing exported ffi_c_int_identity_c(1)->1 symbol: %#v",
					target,
					obj.Symbols,
				)
			}
		})
	}
}

func TestLinuxFamilyBuildsExportedCUIntFFIScalarObjects(t *testing.T) {
	src := `@export("ffi_c_uint_identity_c")
func ffi_c_uint_identity(n: c_uint) -> c_uint:
    return n
`

	for _, target := range []string{"linux-x86", "linux-x64", "linux-x32"} {
		t.Run(target, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_c_uint_identity.t4")
			outPath := filepath.Join(tmp, "ffi_c_uint_identity.tobj")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				target,
				BuildOptions{Emit: EmitLibrary, Jobs: 1},
			); err != nil {
				t.Fatalf("build %s c_uint FFI object: %v", target, err)
			}
			obj, err := ReadObject(outPath)
			if err != nil {
				t.Fatalf("read object: %v", err)
			}
			if obj.Target != target {
				t.Fatalf("object target = %q, want %s", obj.Target, target)
			}
			if !abiSuiteObjectHasSymbolSignature(obj, "ffi_c_uint_identity_c", 1, 1) {
				t.Fatalf(
					"%s object missing exported ffi_c_uint_identity_c(1)->1 symbol: %#v",
					target,
					obj.Symbols,
				)
			}
		})
	}
}

func TestLinuxX86AndX32BuildExportedILP32NativeLibcFFIScalarObjects(t *testing.T) {
	src := `@export("ffi_usize_identity_c")
func ffi_usize_identity(n: usize) -> usize:
    return n

@export("ffi_isize_identity_c")
func ffi_isize_identity(n: isize) -> isize:
    return n

@export("ffi_size_t_identity_c")
func ffi_size_t_identity(n: size_t) -> size_t:
    return n

@export("ffi_ssize_t_identity_c")
func ffi_ssize_t_identity(n: ssize_t) -> ssize_t:
    return n

@export("ffi_native_int_identity_c")
func ffi_native_int_identity(n: native_int) -> native_int:
    return n

@export("ffi_native_uint_identity_c")
func ffi_native_uint_identity(n: native_uint) -> native_uint:
    return n

@export("ffi_c_long_identity_c")
func ffi_c_long_identity(n: c_long) -> c_long:
    return n

@export("ffi_c_ulong_identity_c")
func ffi_c_ulong_identity(n: c_ulong) -> c_ulong:
    return n
`

	for _, target := range []string{"linux-x86", "linux-x32"} {
		t.Run(target, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_ilp32_native_libc_scalars.t4")
			outPath := filepath.Join(tmp, "ffi_ilp32_native_libc_scalars.tobj")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				target,
				BuildOptions{Emit: EmitLibrary, Jobs: 1},
			); err != nil {
				t.Fatalf("build %s ILP32 native/libc scalar FFI object: %v", target, err)
			}
			obj, err := ReadObject(outPath)
			if err != nil {
				t.Fatalf("read object: %v", err)
			}
			if obj.Target != target {
				t.Fatalf("object target = %q, want %s", obj.Target, target)
			}
			for _, symbol := range []string{
				"ffi_usize_identity_c",
				"ffi_isize_identity_c",
				"ffi_size_t_identity_c",
				"ffi_ssize_t_identity_c",
				"ffi_native_int_identity_c",
				"ffi_native_uint_identity_c",
				"ffi_c_long_identity_c",
				"ffi_c_ulong_identity_c",
			} {
				if !abiSuiteObjectHasSymbolSignature(obj, symbol, 1, 1) {
					t.Fatalf(
						"%s object missing exported %s(1)->1 symbol: %#v",
						target,
						symbol,
						obj.Symbols,
					)
				}
			}
		})
	}
}

func TestLinuxX86RejectsExportedPointerAndFunctionPointerFFIBoundaryUntilVerified(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "fnptr_param",
			src: `@export("ffi_callback_param_c")
func ffi_callback_param(cb: fn(Int) -> Int) -> Int:
    return 0
`,
			want: []string{
				"exported function 'ffi_callback_param'",
				"parameter 'cb'",
				"type 'fnptr'",
				"i386 pointer C ABI boundary is not verified on linux-x86",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_x86_pointer_"+tc.name+".t4")
			outPath := filepath.Join(tmp, "ffi_x86_pointer_"+tc.name+".tobj")
			if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			_, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				"linux-x86",
				BuildOptions{Emit: EmitLibrary, Jobs: 1},
			)
			if err == nil {
				t.Fatalf("expected x86 pointer/function-pointer FFI diagnostic")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("diagnostic = %v, want substring %q", err, want)
				}
			}
			if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
				t.Fatalf("output object should not be written, stat error = %v", statErr)
			}
		})
	}
}

func TestNativeTargetsRejectUnsupportedTargetLayoutScalarsWithSourceNativeDiagnostic(t *testing.T) {
	tests := []struct {
		target   string
		name     string
		typeName string
		src      string
	}{
		{
			target:   "linux-x64",
			name:     "x64_ref_param",
			typeName: "ref",
			src: `@export("ffi_ref_param_c")
func ffi_ref_param(n: ref) -> Int:
    return 0
`,
		},
		{
			target:   "linux-x64",
			name:     "x64_usize_param",
			typeName: "usize",
			src: `@export("ffi_usize_param_c")
func ffi_usize_param(n: usize) -> Int:
    return 0
`,
		},
		{
			target:   "linux-x64",
			name:     "x64_native_int_return",
			typeName: "native_int",
			src: `@export("ffi_native_int_return_c")
func ffi_native_int_return() -> native_int:
    return 0
`,
		},
		{
			target:   "linux-x64",
			name:     "x64_rawptr_param",
			typeName: "rawptr",
			src: `@export("ffi_rawptr_param_c")
func ffi_rawptr_param(n: rawptr) -> Int:
    return 0
`,
		},
		{
			target:   "linux-x64",
			name:     "x64_nullable_ptr_param",
			typeName: "nullable_ptr",
			src: `@export("ffi_nullable_ptr_param_c")
func ffi_nullable_ptr_param(n: nullable_ptr) -> Int:
    return 0
`,
		},
		{
			target:   "linux-x86",
			name:     "x86_u32_param",
			typeName: "u32",
			src: `@export("ffi_u32_param_c")
func ffi_u32_param(n: u32) -> Int:
    return 0
`,
		},
		{
			target:   "linux-x86",
			name:     "x86_f64_return",
			typeName: "f64",
			src: `@export("ffi_f64_return_c")
func ffi_f64_return() -> f64:
    return 0
`,
		},
		{
			target:   "linux-x32",
			name:     "x32_u32_param",
			typeName: "u32",
			src: `@export("ffi_u32_param_c")
func ffi_u32_param(n: u32) -> Int:
    return 0
`,
		},
		{
			target:   "linux-x32",
			name:     "x32_f64_return",
			typeName: "f64",
			src: `@export("ffi_f64_return_c")
func ffi_f64_return() -> f64:
    return 0
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.target+"/"+tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_target_layout_"+tc.name+".t4")
			outPath := filepath.Join(tmp, "ffi_target_layout_"+tc.name+".tobj")
			if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			_, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				tc.target,
				BuildOptions{Emit: EmitLibrary, Jobs: 1},
			)
			if err == nil {
				t.Fatalf("expected source-level target-layout scalar diagnostic")
			}
			for _, want := range []string{
				"target-layout scalar type '" + tc.typeName + "'",
				"not supported in source-level Tetra yet",
				"native-int/codegen support",
			} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("diagnostic = %v, want substring %q", err, want)
				}
			}
			if strings.Contains(err.Error(), "pointer C ABI boundary") {
				t.Fatalf("diagnostic = %v, should not be reported as a pointer C ABI boundary", err)
			}
			if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
				t.Fatalf("output object should not be written, stat error = %v", statErr)
			}
		})
	}
}

func TestLinuxX32RejectsExportedFunctionPointerFFIBoundaryUntilVerified(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "param",
			src: `@export("ffi_callback_param_c")
func ffi_callback_param(cb: fn(Int) -> Int) -> Int:
    return 0
`,
			want: []string{
				"exported function 'ffi_callback_param'",
				"parameter 'cb'",
				"x32 pointer C ABI boundary is not verified on linux-x32",
			},
		},
		{
			name: "return",
			src: `func identity(x: Int) -> Int:
    return x

@export("ffi_callback_return_c")
func ffi_callback_return() -> fn(Int) -> Int:
    return identity
`,
			want: []string{
				"exported function 'ffi_callback_return'",
				"return type",
				"x32 pointer C ABI boundary is not verified on linux-x32",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "ffi_x32_fnptr_"+tc.name+".t4")
			outPath := filepath.Join(tmp, "ffi_x32_fnptr_"+tc.name+".tobj")
			if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			_, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				"linux-x32",
				BuildOptions{Emit: EmitLibrary, Jobs: 1},
			)
			if err == nil {
				t.Fatalf("expected x32 function-pointer FFI diagnostic")
			}
			diag := DiagnosticFromError(err)
			if diag.Code != DiagnosticCodeTargetRuntime || diag.Severity != "error" {
				t.Fatalf("diagnostic identity = %#v", diag)
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("diagnostic = %v, want substring %q", err, want)
				}
			}
			if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
				t.Fatalf("output object should not be written, stat error = %v", statErr)
			}
		})
	}
}

// ---- filesystem_runtime_test.go ----

func TestFilesystemRuntimeRequiredSymbolsAndSignatures(t *testing.T) {
	got := requiredFilesystemRuntimeSymbols()
	if len(got) != 1 || got[0] != "__tetra_fs_exists" {
		t.Fatalf("filesystem runtime symbols = %#v, want __tetra_fs_exists", got)
	}
	sig, ok := runtimeObjectSignature("__tetra_fs_exists")
	if !ok {
		t.Fatalf("missing runtime signature for __tetra_fs_exists")
	}
	if sig.paramSlots != 3 || sig.returnSlots != 1 {
		t.Fatalf(
			"__tetra_fs_exists signature = params %d returns %d, want params 3 returns 1",
			sig.paramSlots,
			sig.returnSlots,
		)
	}
}

func TestLinuxX86FilesystemRuntimeObjectExportsFsExists(t *testing.T) {
	rt := buildLinuxX86FilesystemRuntimeObject()
	if rt.Target != "linux-x86" {
		t.Fatalf("runtime target = %q, want linux-x86", rt.Target)
	}
	if rt.Module != "__linux_x86_fsrt" {
		t.Fatalf("runtime module = %q, want __linux_x86_fsrt", rt.Module)
	}
	if len(rt.Symbols) != 1 || rt.Symbols[0].Name != "__tetra_fs_exists" ||
		rt.Symbols[0].Offset != 0 {
		t.Fatalf("runtime symbols = %#v, want __tetra_fs_exists at offset 0", rt.Symbols)
	}
	if len(rt.Data) != 0 || len(rt.Relocs) != 0 {
		t.Fatalf(
			"runtime object must be self-contained, data=%d relocs=%#v",
			len(rt.Data),
			rt.Relocs,
		)
	}
	annotateRuntimeObjectSignatures(rt)
	if err := validateFilesystemRuntimeObject(rt); err != nil {
		t.Fatalf("validate x86 filesystem runtime object: %v", err)
	}
	for name, needle := range map[string][]byte{
		"stack buffer":        {0x81, 0xEC, 0x00, 0x10, 0x00, 0x00},
		"embedded nul guard":  {0x84, 0xD2, 0x0F, 0x84},
		"access syscall":      {0xB8, 0x21, 0x00, 0x00, 0x00},
		"int80 syscall":       {0xCD, 0x80},
		"callee-saved return": {0x81, 0xC4, 0x00, 0x10, 0x00, 0x00, 0x5F, 0x5E, 0x5B, 0x5D, 0xC3},
	} {
		if !bytes.Contains(rt.Code, needle) {
			t.Fatalf("runtime code missing %s sequence % x in % x", name, needle, rt.Code)
		}
	}
}

func TestLinuxX32FilesystemRuntimeObjectExportsFsExists(t *testing.T) {
	rt := buildLinuxX32FilesystemRuntimeObject()
	if rt.Target != "linux-x32" {
		t.Fatalf("runtime target = %q, want linux-x32", rt.Target)
	}
	if rt.Module != "__linux_x32_fsrt" {
		t.Fatalf("runtime module = %q, want __linux_x32_fsrt", rt.Module)
	}
	if len(rt.Symbols) != 1 || rt.Symbols[0].Name != "__tetra_fs_exists" ||
		rt.Symbols[0].Offset != 0 {
		t.Fatalf("runtime symbols = %#v, want __tetra_fs_exists at offset 0", rt.Symbols)
	}
	if len(rt.Data) != 0 || len(rt.Relocs) != 0 {
		t.Fatalf(
			"runtime object must be self-contained, data=%d relocs=%#v",
			len(rt.Data),
			rt.Relocs,
		)
	}
	annotateRuntimeObjectSignatures(rt)
	if err := validateFilesystemRuntimeObject(rt); err != nil {
		t.Fatalf("validate x32 filesystem runtime object: %v", err)
	}
	for name, needle := range map[string][]byte{
		"stack buffer":        {0x48, 0x81, 0xEC, 0x00, 0x10, 0x00, 0x00},
		"embedded nul guard":  {0x84, 0xD2, 0x0F, 0x84},
		"x32 access syscall":  {0xB8, 0x15, 0x00, 0x00, 0x40},
		"syscall instruction": {0x0F, 0x05},
		"return":              {0xC9, 0xC3},
	} {
		if !bytes.Contains(rt.Code, needle) {
			t.Fatalf("runtime code missing %s sequence % x in % x", name, needle, rt.Code)
		}
	}
	if bytes.Contains(rt.Code, []byte{0xB8, 0x15, 0x00, 0x00, 0x00}) {
		t.Fatalf("x32 filesystem runtime emitted plain x64 access syscall: % x", rt.Code)
	}
}

func TestCollectFilesystemRuntimeUsage(t *testing.T) {
	prog, err := Parse([]byte(`
func probe(cap: cap.io) -> Bool
uses io:
    return core.fs_exists("README.md", cap)

func main() -> Int:
    return 0
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !collectFilesystemRuntimeUsage(checked) {
		t.Fatalf("filesystem runtime usage was not collected")
	}
}

func TestValidateFilesystemRuntimeObjectChecksSignatureMetadata(t *testing.T) {
	obj := runtimeObjectWithFilesystemRuntimeSignatures()
	if err := validateFilesystemRuntimeObject(obj); err != nil {
		t.Fatalf("validate filesystem runtime object: %v", err)
	}

	replaceRuntimeSymbolSignature(obj, "__tetra_fs_exists", 2, 1)
	err := validateFilesystemRuntimeObject(obj)
	if err == nil {
		t.Fatalf("expected filesystem runtime signature mismatch")
	}
	if !strings.Contains(
		err.Error(),
		"runtime object symbol '__tetra_fs_exists' signature mismatch",
	) ||
		!strings.Contains(err.Error(), "params=2 want=3") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingFilesystemSymbols(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_filesystem.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing_filesystem",
		Code:    []byte{0xC3},
		Symbols: runtimeObjectSymbols(requiredActorRuntimeSymbols()),
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	srcPath := filepath.Join(tmp, "filesystem_main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        if core.fs_exists("README.md", cap):
            return 0
    return 1
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "filesystem_main"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{RuntimeObjectPath: rtPath},
	)
	if err == nil {
		t.Fatalf("expected missing filesystem runtime symbol failure")
	}
	if !strings.Contains(
		err.Error(),
		"runtime object missing required symbol '__tetra_fs_exists'",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFilesystemRuntimeExistsBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    return 0
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want filesystem exists smoke success", exitCode)
	}
}

func TestLinuxX64FilesystemRuntimeComposesWithTaskScheduler(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func worker() -> Int:
    return 41

func main() -> Int
uses capability, io, runtime:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    let task: task.i32 = core.task_spawn_i32("worker")
    let value: Int = core.task_join_i32(task)
    if value != 41:
        return value
    return 0
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want x64 filesystem+scheduler composition success", exitCode)
	}
}

func TestX86FilesystemRuntimeExistsBuildsAndRunsWhenHostSupportsI386(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "filesystem_x86.tetra")
	outPath := filepath.Join(tmp, "filesystem-x86")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 filesystem runtime: %v", err)
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
		t.Fatalf("x86 filesystem runtime exit=%d stdout=%q, want 0", code, stdout)
	}
}

func TestX32FilesystemRuntimeExistsBuildsAndRunsWhenHostSupportsX32(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "filesystem_x32.tetra")
	outPath := filepath.Join(tmp, "filesystem-x32")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x32 filesystem runtime: %v", err)
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
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 0 {
		t.Fatalf("x32 filesystem runtime exit=%d stdout=%q, want 0", code, stdout)
	}
}

func TestX86FilesystemRuntimeComposesWithTaskScheduler(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "filesystem_mixed_x86.tetra")
	outPath := filepath.Join(tmp, "filesystem-mixed-x86")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses capability, io, runtime:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1})
	if err != nil {
		t.Fatalf("build mixed x86 filesystem+scheduler runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read mixed x86 executable: %v", err)
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
	if code != 41 {
		t.Fatalf("x86 filesystem+scheduler runtime exit=%d stdout=%q, want 41", code, stdout)
	}
}

func TestX32FilesystemRuntimeComposesWithTaskScheduler(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "filesystem_mixed_x32.tetra")
	outPath := filepath.Join(tmp, "filesystem-mixed-x32")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses capability, io, runtime:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1})
	if err != nil {
		t.Fatalf("build mixed x32 filesystem+scheduler runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read mixed x32 executable: %v", err)
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
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 41 {
		t.Fatalf("x32 filesystem+scheduler runtime exit=%d stdout=%q, want 41", code, stdout)
	}
}

func TestFilesystemRuntimeRejectsUnsupportedNativeTargets(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "filesystem_main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        if core.fs_exists("README.md", cap):
            return 0
    return 1
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	for _, tc := range []struct {
		target string
		want   string
	}{
		{target: "macos-x64", want: "macos-x64"},
		{target: "windows-x64", want: "windows-x64"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			outPath := filepath.Join(tmp, "filesystem-"+tc.target)
			_, err := BuildFileWithStatsOpt(srcPath, outPath, tc.target, BuildOptions{Jobs: 1})
			if err == nil {
				t.Fatalf("expected unsupported filesystem runtime diagnostic")
			}
			want := "filesystem runtime not supported on " + tc.want
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want %q", err, want)
			}
		})
	}
}

func runtimeObjectWithFilesystemRuntimeSignatures() *Object {
	obj := &Object{}
	for _, name := range requiredFilesystemRuntimeSymbols() {
		sig, ok := runtimeObjectSignature(name)
		if !ok {
			panic("missing filesystem runtime signature for " + name)
		}
		obj.Symbols = append(obj.Symbols, Symbol{
			Name:         name,
			HasSignature: true,
			ParamSlots:   sig.paramSlots,
			ReturnSlots:  sig.returnSlots,
		})
	}
	return obj
}

func runtimeObjectSymbols(names []string) []Symbol {
	symbols := make([]Symbol, 0, len(names))
	for _, name := range names {
		symbols = append(symbols, Symbol{Name: name, Offset: 0})
	}
	return symbols
}

// ---- first_class_callables_coverage_test.go ----

func TestP22FirstClassCallableCoverageProvesSafeABIWitnesses(t *testing.T) {
	report, err := BuildP22FirstClassCallableCoverage()
	if err != nil {
		t.Fatalf("BuildP22FirstClassCallableCoverage: %v", err)
	}
	if report.SchemaVersion != firstClassCallableCoverageSchemaV1 {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, firstClassCallableCoverageSchemaV1)
	}
	if report.Scope != firstClassCallableCoverageScopeP221 {
		t.Fatalf("scope = %q, want %q", report.Scope, firstClassCallableCoverageScopeP221)
	}
	if err := ValidateP22FirstClassCallableCoverage(report); err != nil {
		t.Fatalf("ValidateP22FirstClassCallableCoverage: %v", err)
	}

	rows := map[FirstClassCallableCoverageID]FirstClassCallableCoverageRow{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			t.Fatalf("coverage row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p22FirstClassCallableCoverageIDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("coverage missing row %s: %#v", id, report.Rows)
		}
	}

	p22AssertCallableRow(
		t,
		rows[FirstClassCallableFnPtrFastPath],
		[]string{"FnPtrSlotCount", "9-slot", "no heap environment", "fnptr"},
	)
	p22AssertCallableRow(
		t,
		rows[FirstClassCallableFatHandle],
		[]string{
			"CallableHandleSlotCount",
			"4-slot handle",
			"IRAllocBytes",
			"IRMemWritePtrOffset",
			"IRMemReadPtrOffset",
		},
	)
	p22AssertCallableRow(
		t,
		rows[FirstClassCallableCaptureSafetyClassifier],
		[]string{"semantics_memory_resources.go", "safe immutable by-value"},
	)
	p22AssertCallableRow(
		t,
		rows[FirstClassCallableMutableCaptureDiagnostics],
		[]string{"mutable by-reference capture", "global-escape", "heap-escape"},
	)
	p22AssertCallableRow(
		t,
		rows[FirstClassCallableResourceThreadDiagnostics],
		[]string{"pointer/resource capture", "thread-boundary callable escape"},
	)
	p22AssertCallableRow(
		t,
		rows[FirstClassCallableFixedABIWidth],
		[]string{
			"FnPtrEnvSlotCount = 8",
			"FnPtrSlotCount = 9",
			"CallableHandleSlotCount = 4",
			"fixed ABI width",
		},
	)
	p22AssertCallableRow(
		t,
		rows[FirstClassCallableInterfaceMetadata],
		[]string{".t4i", "ReturnFunctionHandleValue", "ReturnSlots = 4"},
	)
	p22AssertCallableRow(
		t,
		rows[FirstClassCallableStorageCallbackPaths],
		[]string{"aliases", "struct fields", "enum payloads", "callback arguments", "returns"},
	)

	witnesses := map[string]FirstClassCallableABIWitness{}
	for _, witness := range report.Witnesses {
		witnesses[witness.ID] = witness
	}
	fnptr := witnesses[firstClassCallableFnPtrWitnessID]
	if fnptr.ID == "" {
		t.Fatalf("missing fnptr witness: %#v", report.Witnesses)
	}
	if fnptr.CaptureCount != 1 || fnptr.UsesHandle || fnptr.FnPtrSlotCount != 9 ||
		fnptr.CallableHandleSlotCount != 4 ||
		fnptr.LocalSlotCount != 9 ||
		fnptr.AllocBytesCount != 0 {
		t.Fatalf("fnptr witness = %#v, want one-capture 9-slot fnptr without heap env", fnptr)
	}
	handle := witnesses[firstClassCallableHandleWitnessID]
	if handle.ID == "" {
		t.Fatalf("missing handle witness: %#v", report.Witnesses)
	}
	if handle.CaptureCount != 9 || !handle.UsesHandle || handle.FnPtrSlotCount != 9 ||
		handle.CallableHandleSlotCount != 4 ||
		handle.LocalSlotCount != 4 {
		t.Fatalf("handle witness = %#v, want nine-capture fixed 4-slot handle", handle)
	}
	if handle.AllocBytesCount != 1 || handle.EnvWriteCount != 9 || handle.EnvReadCount != 9 ||
		handle.CallArgSlots != 10 ||
		handle.CallRetSlots != 1 {
		t.Fatalf(
			"handle witness IR counts = %#v, want alloc=1 writes=9 reads=9 call arg/ret=10/1",
			handle,
		)
	}

	for _, nonClaim := range []string{
		"no variable-width callable ABI is claimed",
		"no exploding callable return slots are claimed",
		"no mutable by-reference capture support is claimed",
		"no pointer/resource capture support is claimed",
		"no thread-boundary callable transfer is claimed",
		"no runtime generic callable polymorphism is claimed",
		"no dynamic callable dispatch is claimed",
		"no unsafe lifetime relaxation is claimed",
		"no performance claim is made",
		"no runtime behavior change beyond the existing callable ABI is claimed",
		"safe-program semantics do not change",
	} {
		if !p22FirstClassCallableHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP22FirstClassCallableCoverageRejectsFakeClaimsAndDrift(t *testing.T) {
	base, err := BuildP22FirstClassCallableCoverage()
	if err != nil {
		t.Fatalf("BuildP22FirstClassCallableCoverage: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*FirstClassCallableCoverageReport)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness reference",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "bad handle ABI width",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.Witnesses[1].CallableHandleSlotCount = 5
			},
			want: "fixed ABI",
		},
		{
			name: "bad handle env read count",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.Witnesses[1].EnvReadCount = 8
			},
			want: "handle witness",
		},
		{
			name: "variable ABI width claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.VariableABIWidthClaimed = true
			},
			want: "variable-width",
		},
		{
			name: "exploding return slots claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.ExplodingReturnSlotsClaimed = true
			},
			want: "exploding",
		},
		{
			name: "mutable by-ref capture claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.MutableByRefCaptureClaimed = true
			},
			want: "mutable by-reference",
		},
		{
			name: "pointer resource capture claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.PointerResourceCaptureClaimed = true
			},
			want: "pointer/resource",
		},
		{
			name: "thread transfer claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.ThreadBoundaryCallableTransferClaimed = true
			},
			want: "thread-boundary",
		},
		{
			name: "runtime generic callable claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.RuntimeGenericPolymorphismClaimed = true
			},
			want: "runtime generic callable",
		},
		{
			name: "dynamic callable dispatch claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.DynamicCallableDispatchClaimed = true
			},
			want: "dynamic callable",
		},
		{
			name: "unsafe lifetime relaxation claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.UnsafeLifetimeRelaxationClaimed = true
			},
			want: "unsafe lifetime",
		},
		{
			name: "performance claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
		{
			name: "runtime behavior change claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.SafeSemanticsChanged = true
			},
			want: "safe-program semantics",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := cloneFirstClassCallableCoverage(base)
			tc.mutate(&report)
			err := ValidateP22FirstClassCallableCoverage(report)
			if err == nil {
				t.Fatalf("ValidateP22FirstClassCallableCoverage accepted fake report: %#v", report)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func p22AssertCallableRow(t *testing.T, row FirstClassCallableCoverageRow, wants []string) {
	t.Helper()
	combined := row.Name + " " + row.Status + " " + strings.Join(
		row.Evidence,
		" ",
	) + " " + strings.Join(
		row.Tests,
		" ",
	) + " " + strings.Join(
		row.Boundaries,
		" ",
	)
	for _, want := range wants {
		if !strings.Contains(combined, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}

func cloneFirstClassCallableCoverage(
	report FirstClassCallableCoverageReport,
) FirstClassCallableCoverageReport {
	report.Rows = append([]FirstClassCallableCoverageRow{}, report.Rows...)
	for i := range report.Rows {
		report.Rows[i].Evidence = append([]string{}, report.Rows[i].Evidence...)
		report.Rows[i].Tests = append([]string{}, report.Rows[i].Tests...)
		report.Rows[i].Boundaries = append([]string{}, report.Rows[i].Boundaries...)
		report.Rows[i].WitnessIDs = append([]string{}, report.Rows[i].WitnessIDs...)
	}
	report.Witnesses = append([]FirstClassCallableABIWitness{}, report.Witnesses...)
	report.NonClaims = append([]string{}, report.NonClaims...)
	return report
}

func p22FirstClassCallableHasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// ---- formal_core_v1_test.go ----

func TestP23FormalCoreV1CoversMachineCheckableCoreRules(t *testing.T) {
	report, err := BuildP23FormalCoreV1Report()
	if err != nil {
		t.Fatalf("BuildP23FormalCoreV1Report: %v", err)
	}
	if report.SchemaVersion != formalCoreV1Schema {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, formalCoreV1Schema)
	}
	if report.Scope != formalCoreV1ScopeP232 {
		t.Fatalf("scope = %q, want %q", report.Scope, formalCoreV1ScopeP232)
	}
	if err := ValidateP23FormalCoreV1Report(report); err != nil {
		t.Fatalf("ValidateP23FormalCoreV1Report: %v", err)
	}

	rows := map[FormalCoreV1ID]FormalCoreV1Row{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 ||
			len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p23FormalCoreV1IDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p23AssertFormalCoreRow(
		t,
		rows[FormalCoreV1Values],
		[]string{"differential", "stable observable", "i32"},
	)
	p23AssertFormalCoreRow(
		t,
		rows[FormalCoreV1BorrowsOwnedCopy],
		[]string{"borrow", "copy", "owned"},
	)
	p23AssertFormalCoreRow(
		t,
		rows[FormalCoreV1ProvenanceRegions],
		[]string{"provenance", "regions", "PLIR"},
	)
	p23AssertFormalCoreRow(
		t,
		rows[FormalCoreV1BoundsProofIDSemantics],
		[]string{"proof id", "proof guards", "CheckBoundsProofsWithPLIR"},
	)
	p23AssertFormalCoreRow(
		t,
		rows[FormalCoreV1AllocationLengthContract],
		[]string{"length contract", "negative", "overflow"},
	)
	p23AssertFormalCoreRow(
		t,
		rows[FormalCoreV1AllocationIntentLowering],
		[]string{"allocation intent", "ValidateAllocationLowering"},
	)
	p23AssertFormalCoreRow(
		t,
		rows[FormalCoreV1RawPointerBoundsMetadata],
		[]string{"raw pointer bounds", "allocation-base", "external/unknown"},
	)
	p23AssertFormalCoreRow(
		t,
		rows[FormalCoreV1CheckEliminationValidity],
		[]string{"unchecked", "proof id", "safe-semantics"},
	)

	witnesses := map[string]FormalCoreV1Witness{}
	for _, witness := range report.Witnesses {
		witnesses[witness.ID] = witness
	}
	spec := witnesses[p23FormalCoreSpecWitnessID]
	if !spec.FormalSpecValid || spec.FormalConcepts < 9 || spec.FormalRules < 7 {
		t.Fatalf("formal spec witness = %#v, want valid expanded concept/rule inventory", spec)
	}
	values := witnesses[p23FormalCoreValuesWitnessID]
	if values.ValueSamples == 0 || values.DifferentialLanes < 5 {
		t.Fatalf("values witness = %#v, want backend differential value evidence", values)
	}
	plir := witnesses[p23FormalCorePLIRWitnessID]
	if !plir.BorrowCopyFacts || !plir.ProvenanceRegionFacts {
		t.Fatalf("PLIR witness = %#v, want borrow/copy and provenance/region facts", plir)
	}
	proof := witnesses[p23FormalCoreProofWitnessID]
	if !proof.BoundsProofIDsChecked || !proof.MissingProofRejected ||
		!proof.CheckEliminationValidated {
		t.Fatalf(
			"proof witness = %#v, want proof-id validation and check-elimination rejection",
			proof,
		)
	}
	allocation := witnesses[p23FormalCoreAllocationWitnessID]
	if !allocation.AllocationLengthContractsChecked ||
		!allocation.InvalidAllocationLengthRejected ||
		!allocation.AllocationIntentLoweringValidated ||
		!allocation.AllocationIntentDriftRejected {
		t.Fatalf(
			"allocation witness = %#v, want length contract and lowering validation",
			allocation,
		)
	}
	raw := witnesses[p23FormalCoreRawPointerWitnessID]
	if raw.RawPointerBoundsCases < 4 || !raw.RawPointerImpossibleAddRejected ||
		!raw.RawPointerUnknownStayedChecked {
		t.Fatalf(
			("raw pointer witness = %#v, want allocation-base, derived, " +
				"rejected, and checked-unknown metadata"),
			raw,
		)
	}

	for _, nonClaim := range []string{
		"no full formal proof of Tetra is claimed",
		"no broad language theorem prover is claimed",
		"unsafe policy does not change",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p23FormalCoreHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP23FormalCoreV1RejectsFakeClaimsAndDrift(t *testing.T) {
	base, err := BuildP23FormalCoreV1Report()
	if err != nil {
		t.Fatalf("BuildP23FormalCoreV1Report: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*FormalCoreV1Report)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *FormalCoreV1Report) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *FormalCoreV1Report) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness",
			mutate: func(report *FormalCoreV1Report) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "formal spec invalid",
			mutate: func(report *FormalCoreV1Report) {
				report.FormalSpecValid = false
			},
			want: "formal spec",
		},
		{
			name: "missing values",
			mutate: func(report *FormalCoreV1Report) {
				report.ValueSamples = 0
			},
			want: "values",
		},
		{
			name: "missing PLIR facts",
			mutate: func(report *FormalCoreV1Report) {
				report.BorrowCopyFacts = false
			},
			want: "borrow",
		},
		{
			name: "missing proof ids",
			mutate: func(report *FormalCoreV1Report) {
				report.BoundsProofIDsChecked = false
			},
			want: "bounds proof",
		},
		{
			name: "missing allocation length",
			mutate: func(report *FormalCoreV1Report) {
				report.AllocationLengthContractsChecked = false
			},
			want: "allocation length",
		},
		{
			name: "missing raw pointer bounds",
			mutate: func(report *FormalCoreV1Report) {
				report.RawPointerBoundsCases = 0
			},
			want: "raw pointer",
		},
		{
			name: "full formal proof claim",
			mutate: func(report *FormalCoreV1Report) {
				report.FullFormalProofClaimed = true
			},
			want: "full formal proof",
		},
		{
			name: "broad language proof claim",
			mutate: func(report *FormalCoreV1Report) {
				report.BroadLanguageProofClaimed = true
			},
			want: "broad language",
		},
		{
			name: "unsafe policy change",
			mutate: func(report *FormalCoreV1Report) {
				report.UnsafePolicyChanged = true
			},
			want: "unsafe policy",
		},
		{
			name: "runtime behavior change",
			mutate: func(report *FormalCoreV1Report) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics change",
			mutate: func(report *FormalCoreV1Report) {
				report.SafeSemanticsChanged = true
			},
			want: "safe semantics",
		},
		{
			name: "performance claim",
			mutate: func(report *FormalCoreV1Report) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := base
			report.Rows = append([]FormalCoreV1Row(nil), base.Rows...)
			report.Witnesses = append([]FormalCoreV1Witness(nil), base.Witnesses...)
			report.NonClaims = append([]string(nil), base.NonClaims...)
			tc.mutate(&report)
			err := ValidateP23FormalCoreV1Report(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateP23FormalCoreV1Report error = %v, want %q", err, tc.want)
			}
		})
	}
}

func p23AssertFormalCoreRow(t *testing.T, row FormalCoreV1Row, wants []string) {
	t.Helper()
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}

// ---- fuzz_property_differential_v1_test.go ----

func TestP23FuzzPropertyDifferentialReportCoversMasterPlanTargets(t *testing.T) {
	report, err := BuildP23FuzzPropertyDifferentialReport()
	if err != nil {
		t.Fatalf("BuildP23FuzzPropertyDifferentialReport: %v", err)
	}
	if report.SchemaVersion != fuzzPropertyDifferentialSchema {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, fuzzPropertyDifferentialSchema)
	}
	if report.Scope != fuzzPropertyDifferentialScopeP231 {
		t.Fatalf("scope = %q, want %q", report.Scope, fuzzPropertyDifferentialScopeP231)
	}
	if err := ValidateP23FuzzPropertyDifferentialReport(report); err != nil {
		t.Fatalf("ValidateP23FuzzPropertyDifferentialReport: %v", err)
	}

	rows := map[FuzzPropertyDifferentialID]FuzzPropertyDifferentialRow{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 ||
			len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p23FuzzPropertyDifferentialIDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p23AssertFuzzRow(
		t,
		rows[FuzzPropertyDifferentialParserCheckerGeneratedPrograms],
		[]string{"generated source", "Parse", "Check"},
	)
	p23AssertFuzzRow(
		t,
		rows[FuzzPropertyDifferentialPLIRLoweringVerifierPipeline],
		[]string{"BuildPLIR", "Lower", "VerifyIRProgram"},
	)
	p23AssertFuzzRow(
		t,
		rows[FuzzPropertyDifferentialBackendMatrixExpansion],
		[]string{"CheckBackendMatrix", "SSA", "Machine IR", "randomized"},
	)
	p23AssertFuzzRow(
		t,
		rows[FuzzPropertyDifferentialNativeBackendBoundary],
		[]string{"native backend", "Linux x64", "explicit unavailable boundary"},
	)
	p23AssertFuzzRow(
		t,
		rows[FuzzPropertyDifferentialRuntimeAllocatorProperties],
		[]string{"AlignRegionBytes", "negative", "overflow"},
	)
	p23AssertFuzzRow(
		t,
		rows[FuzzPropertyDifferentialActorTransferStressBoundary],
		[]string{"TypedActorOwnershipTransferCoverage", "stress diagnostics", "PLIR moved facts"},
	)
	p23AssertFuzzRow(
		t,
		rows[FuzzPropertyDifferentialFuzzNightlySummaryGate],
		[]string{"fuzz-nightly.sh", "validate-fuzz-summary", "unstable-seeds"},
	)
	p23AssertFuzzRow(
		t,
		rows[FuzzPropertyDifferentialReducerFailureArtifacts],
		[]string{"reduced_to_single_sample", "reproducer"},
	)

	if report.ParserCheckerGeneratedPrograms < 4 {
		t.Fatalf(
			"parser/checker generated programs = %d, want at least 4",
			report.ParserCheckerGeneratedPrograms,
		)
	}
	if report.PLIRVerifierCases < report.ParserCheckerGeneratedPrograms ||
		report.LoweringVerifierCases < report.ParserCheckerGeneratedPrograms {
		t.Fatalf(
			"pipeline counts parser=%d plir=%d lowering=%d",
			report.ParserCheckerGeneratedPrograms,
			report.PLIRVerifierCases,
			report.LoweringVerifierCases,
		)
	}
	if report.BackendMatrixCases == 0 || report.BackendMatrixRandomizedSamples == 0 ||
		!report.BackendMatrixReducerRecorded {
		t.Fatalf("backend matrix coverage incomplete: %#v", report)
	}
	if report.NativeBackendHostSupported {
		if report.NativeBackendSamples == 0 {
			t.Fatalf("native host supported but native samples = 0: %#v", report)
		}
	} else if !strings.Contains(report.NativeBackendUnavailableReason, "linux/amd64") {
		t.Fatalf(
			"native unavailable reason = %q, want linux/amd64 boundary",
			report.NativeBackendUnavailableReason,
		)
	}
	if report.RuntimeAllocatorPropertyCases == 0 || !report.RuntimeAllocatorRejectsInvalid {
		t.Fatalf("allocator property coverage incomplete: %#v", report)
	}
	if !report.ActorTransferStressDiagnostics {
		t.Fatalf("actor transfer stress diagnostics not recorded: %#v", report)
	}
	if report.FuzzSummaryGateArtifacts < 4 || !report.NightlyLongFuzzBoundaryRecorded {
		t.Fatalf("fuzz summary gate incomplete: %#v", report)
	}
	for _, nonClaim := range []string{
		"no full program correctness claim is made",
		"no exhaustive fuzzing is claimed",
		"no full native differential suite is claimed",
		"no performance claim is made",
		"runtime behavior does not change",
		"safe-program semantics do not change",
	} {
		if !p23FuzzHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP23FuzzPropertyDifferentialRejectsFakeClaimsAndDrift(t *testing.T) {
	base, err := BuildP23FuzzPropertyDifferentialReport()
	if err != nil {
		t.Fatalf("BuildP23FuzzPropertyDifferentialReport: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*FuzzPropertyDifferentialReport)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "missing parser checker coverage",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.ParserCheckerGeneratedPrograms = 0
			},
			want: "parser/checker",
		},
		{
			name: "missing randomized backend samples",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.BackendMatrixRandomizedSamples = 0
			},
			want: "randomized",
		},
		{
			name: "missing reducer",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.BackendMatrixReducerRecorded = false
			},
			want: "reducer",
		},
		{
			name: "missing actor stress",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.ActorTransferStressDiagnostics = false
			},
			want: "actor transfer",
		},
		{
			name: "missing fuzz summary artifacts",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.FuzzSummaryGateArtifacts = 0
			},
			want: "fuzz summary",
		},
		{
			name: "missing nightly boundary",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.NightlyLongFuzzBoundaryRecorded = false
			},
			want: "nightly",
		},
		{
			name: "full correctness claim",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.FullCorrectnessClaimed = true
			},
			want: "full program correctness",
		},
		{
			name: "exhaustive fuzzing claim",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.ExhaustiveFuzzingClaimed = true
			},
			want: "exhaustive fuzzing",
		},
		{
			name: "full native differential claim",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.FullNativeDifferentialClaimed = true
			},
			want: "full native differential",
		},
		{
			name: "runtime behavior claim",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics claim",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.SafeSemanticsChanged = true
			},
			want: "safe semantics",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := base
			report.Rows = append([]FuzzPropertyDifferentialRow(nil), base.Rows...)
			report.Witnesses = append([]FuzzPropertyDifferentialWitness(nil), base.Witnesses...)
			report.NonClaims = append([]string(nil), base.NonClaims...)
			tc.mutate(&report)
			err := ValidateP23FuzzPropertyDifferentialReport(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf(
					"ValidateP23FuzzPropertyDifferentialReport error = %v, want %q",
					err,
					tc.want,
				)
			}
		})
	}
}

func p23AssertFuzzRow(t *testing.T, row FuzzPropertyDifferentialRow, wants []string) {
	t.Helper()
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}

// ---- fuzz_suite_test.go ----

func TestRunTargetFuzzChecksCoversX86AndX64Family(t *testing.T) {
	tests := []struct {
		target string
		names  []string
	}{
		{
			target: "x86",
			names: []string{
				"x86 layout fuzz",
				"x86 object signature fuzz",
				"x86 target alias fuzz",
			},
		},
		{
			target: "x64",
			names: []string{
				"x64 layout fuzz",
				"x64 object signature fuzz",
				"x64 target alias fuzz",
			},
		},
		{
			target: "windows-x64",
			names: []string{
				"windows-x64 layout fuzz",
				"windows-x64 object signature fuzz",
				"windows-x64 target alias fuzz",
			},
		},
		{
			target: "macos-x64",
			names: []string{
				"macos-x64 layout fuzz",
				"macos-x64 object signature fuzz",
				"macos-x64 target alias fuzz",
			},
		},
		{
			target: "x32",
			names: []string{
				"x32 layout fuzz",
				"x32 object signature fuzz",
				"x32 target alias fuzz",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			checks, err := RunTargetFuzzChecks(tt.target)
			if err != nil {
				t.Fatalf("RunTargetFuzzChecks(%s): %v", tt.target, err)
			}
			if len(checks) != len(tt.names) {
				t.Fatalf("checks = %#v, want %d checks", checks, len(tt.names))
			}
			for i, want := range tt.names {
				if checks[i].Name != want {
					t.Fatalf("check[%d] = %#v, want name %q", i, checks[i], want)
				}
				if checks[i].Error != "" {
					t.Fatalf("check[%d] = %#v, want passing %q", i, checks[i], want)
				}
			}
		})
	}
}

// ---- link_object_contract_test.go ----

func TestLinkObjectTargetMismatch(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	other := "windows-x64"
	if tgt.Triple == "windows-x64" {
		other = "linux-x64"
	}

	tmp := t.TempDir()
	objPath := filepath.Join(tmp, "lib.tobj")
	if err := WriteObject(objPath, &Object{
		Target:  other,
		Module:  "__testlib",
		Code:    []byte{0xC3},
		Symbols: []Symbol{{Name: "__testlib", Offset: 0}},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}

	outPath := filepath.Join(tmp, "app"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(
		filepath.Join("..", "examples", "smoke", "basic", "hello.tetra"),
		outPath,
		tgt.Triple,
		BuildOptions{
			LinkObjectPaths: []string{objPath},
		},
	)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "link object target mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildLinksInterfaceDependencyWithMatchingImplementationObject(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	libSrc := filepath.Join(tmp, filepath.FromSlash("lib/math/core.t4"))
	libObj := filepath.Join(tmp, "math.tobj")
	if err := writeFile(libSrc, string(src)); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildFileWithStatsOpt(
		libSrc,
		libObj,
		tgt.Triple,
		BuildOptions{Emit: EmitLibrary},
	); err != nil {
		t.Fatalf("build library: %v", err)
	}
	iface, err := GenerateInterfaceFromSource(src, libSrc)
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, ("module app.main\nimport math.core as math\nfunc main() -> Int:" +
		"\n    return math.add(40, 2)\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(
		filepath.Join(tmp, filepath.FromSlash("app/math/core.t4i")),
		string(iface),
	); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(tmp, "app-bin"+tgt.ExeExt)
	stats, err := BuildFileWithStatsOpt(
		appSrc,
		outPath,
		tgt.Triple,
		BuildOptions{LinkObjectPaths: []string{libObj}},
	)
	if err != nil {
		t.Fatalf("build with .t4i + matching .tobj: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("missing output: %v", err)
	}
	testkit.AssertModules(t, stats.InterfaceModules, []string{"math.core"})
}

func TestBuildLinksGeneratedInterfaceExtensionWithMatchingImplementationObject(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module engine.vec

pub struct Vec2:
    x: Int
    y: Int

pub extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x + self.y
`)
	libSrc := filepath.Join(tmp, filepath.FromSlash("lib/engine/vec.t4"))
	libObj := filepath.Join(tmp, "vec.tobj")
	if err := writeFile(libSrc, string(src)); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildFileWithStatsOpt(
		libSrc,
		libObj,
		tgt.Triple,
		BuildOptions{Emit: EmitLibrary},
	); err != nil {
		t.Fatalf("build library: %v", err)
	}
	iface, err := GenerateInterfaceFromSource(src, libSrc)
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, `module app.main
import engine.vec as vec

func main() -> Int:
    let v: vec.Vec2 = vec.Vec2(x: 40, y: 2)
    return vec.Vec2.sum(v)
`); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(
		filepath.Join(tmp, filepath.FromSlash("app/engine/vec.t4i")),
		string(iface),
	); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(tmp, "app-bin"+tgt.ExeExt)
	stats, err := BuildFileWithStatsOpt(
		appSrc,
		outPath,
		tgt.Triple,
		BuildOptions{LinkObjectPaths: []string{libObj}},
	)
	if err != nil {
		t.Fatalf(
			"build generated .t4i extension with matching .tobj: %v\ninterface:\n%s",
			err,
			iface,
		)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("missing output: %v", err)
	}
	testkit.AssertModules(t, stats.InterfaceModules, []string{"engine.vec"})
}

func TestBuildRejectsInterfaceDependencyWithoutImplementationObject(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	iface, err := GenerateInterfaceFromSource(
		src,
		filepath.Join(tmp, filepath.FromSlash("math/core.t4")),
	)
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/main.t4"))
	if err := writeFile(appSrc, ("module app.main\nimport math.core as math\nfunc main() -> Int:" +
		"\n    return math.add(40, 2)\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(
		filepath.Join(tmp, filepath.FromSlash("math/core.t4i")),
		string(iface),
	); err != nil {
		t.Fatal(err)
	}

	_, err = BuildFileWithStatsOpt(
		appSrc,
		filepath.Join(tmp, "app"+tgt.ExeExt),
		tgt.Triple,
		BuildOptions{},
	)
	if err == nil {
		t.Fatalf("expected missing implementation object error")
	}
	if !strings.Contains(
		err.Error(),
		"missing implementation object for interface module 'math.core'",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsInterfaceImplementationAPIHashMismatch(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	interfaceSrc := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	implSrc := []byte(`module math.core

pub func add(a: Int, b: Bool) -> Int:
    return a
`)
	libSrc := filepath.Join(tmp, filepath.FromSlash("lib/math/core.t4"))
	libObj := filepath.Join(tmp, "math.tobj")
	if err := writeFile(libSrc, string(implSrc)); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildFileWithStatsOpt(
		libSrc,
		libObj,
		tgt.Triple,
		BuildOptions{Emit: EmitLibrary},
	); err != nil {
		t.Fatalf("build library: %v", err)
	}
	iface, err := GenerateInterfaceFromSource(
		interfaceSrc,
		filepath.Join(tmp, filepath.FromSlash("app/math/core.t4")),
	)
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, ("module app.main\nimport math.core as math\nfunc main() -> Int:" +
		"\n    return math.add(40, 2)\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(
		filepath.Join(tmp, filepath.FromSlash("app/math/core.t4i")),
		string(iface),
	); err != nil {
		t.Fatal(err)
	}

	_, err = BuildFileWithStatsOpt(
		appSrc,
		filepath.Join(tmp, "app"+tgt.ExeExt),
		tgt.Triple,
		BuildOptions{LinkObjectPaths: []string{libObj}},
	)
	if err == nil {
		t.Fatalf("expected API hash mismatch")
	}
	if !strings.Contains(err.Error(), "public API hash mismatch for interface module 'math.core'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsDuplicateInterfaceImplementationObjects(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	libSrc := filepath.Join(tmp, filepath.FromSlash("lib/math/core.t4"))
	if err := writeFile(libSrc, string(src)); err != nil {
		t.Fatal(err)
	}
	var objs []string
	for _, name := range []string{"math-a.tobj", "math-b.tobj"} {
		objPath := filepath.Join(tmp, name)
		if _, err := BuildFileWithStatsOpt(
			libSrc,
			objPath,
			tgt.Triple,
			BuildOptions{Emit: EmitLibrary},
		); err != nil {
			t.Fatalf("build library %s: %v", name, err)
		}
		objs = append(objs, objPath)
	}
	iface, err := GenerateInterfaceFromSource(src, libSrc)
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, ("module app.main\nimport math.core as math\nfunc main() -> Int:" +
		"\n    return math.add(40, 2)\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(
		filepath.Join(tmp, filepath.FromSlash("app/math/core.t4i")),
		string(iface),
	); err != nil {
		t.Fatal(err)
	}

	_, err = BuildFileWithStatsOpt(
		appSrc,
		filepath.Join(tmp, "app"+tgt.ExeExt),
		tgt.Triple,
		BuildOptions{LinkObjectPaths: objs},
	)
	if err == nil {
		t.Fatalf("expected duplicate implementation provider error")
	}
	if !strings.Contains(
		err.Error(),
		"duplicate implementation object for interface module 'math.core'",
	) &&
		!strings.Contains(err.Error(), "duplicate symbol 'math.core.add'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsInterfaceImplementationMissingSymbol(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	apiHash, err := InterfaceFingerprintFromSource(
		src,
		filepath.Join(tmp, filepath.FromSlash("math/core.t4")),
	)
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	objPath := filepath.Join(tmp, "math.tobj")
	if err := WriteObject(objPath, &Object{
		Target:          tgt.Triple,
		Module:          "math.core",
		CompilerVersion: Version(),
		PublicAPIHash:   apiHash,
		Code:            []byte{0xC3},
		Symbols:         []Symbol{{Name: "math.core.other", Offset: 0}},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}
	iface, err := GenerateInterfaceFromSource(
		src,
		filepath.Join(tmp, filepath.FromSlash("app/math/core.t4")),
	)
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, ("module app.main\nimport math.core as math\nfunc main() -> Int:" +
		"\n    return math.add(40, 2)\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(
		filepath.Join(tmp, filepath.FromSlash("app/math/core.t4i")),
		string(iface),
	); err != nil {
		t.Fatal(err)
	}

	_, err = BuildFileWithStatsOpt(
		appSrc,
		filepath.Join(tmp, "app-bin"+tgt.ExeExt),
		tgt.Triple,
		BuildOptions{LinkObjectPaths: []string{objPath}},
	)
	if err == nil {
		t.Fatalf("expected missing implementation symbol error")
	}
	if !strings.Contains(
		err.Error(),
		"implementation object for interface module 'math.core' missing exported symbol 'math.core.add'",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsInterfaceImplementationSignatureMismatch(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	apiHash, err := InterfaceFingerprintFromSource(
		src,
		filepath.Join(tmp, filepath.FromSlash("math/core.t4")),
	)
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	objPath := filepath.Join(tmp, "math-wrong-abi.tobj")
	if err := WriteObject(objPath, &Object{
		Target:          tgt.Triple,
		Module:          "math.core",
		CompilerVersion: Version(),
		PublicAPIHash:   apiHash,
		Code:            []byte{0xC3},
		Symbols: []Symbol{{
			Name:         "math.core.add",
			Offset:       0,
			HasSignature: true,
			ParamSlots:   1,
			ReturnSlots:  1,
		}},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}
	iface, err := GenerateInterfaceFromSource(
		src,
		filepath.Join(tmp, filepath.FromSlash("app/math/core.t4")),
	)
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, ("module app.main\nimport math.core as math\nfunc main() -> Int:" +
		"\n    return math.add(40, 2)\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(
		filepath.Join(tmp, filepath.FromSlash("app/math/core.t4i")),
		string(iface),
	); err != nil {
		t.Fatal(err)
	}

	_, err = BuildFileWithStatsOpt(
		appSrc,
		filepath.Join(tmp, "app-bin"+tgt.ExeExt),
		tgt.Triple,
		BuildOptions{LinkObjectPaths: []string{objPath}},
	)
	if err == nil {
		t.Fatalf("expected implementation signature mismatch error")
	}
	if !strings.Contains(
		err.Error(),
		("implementation object for interface module 'math.core' "+
			"symbol 'math.core.add' signature mismatch"),
	) ||
		!strings.Contains(err.Error(), "params=1 want=2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsInterfaceImplementationMissingSignatureMetadata(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	apiHash, err := InterfaceFingerprintFromSource(
		src,
		filepath.Join(tmp, filepath.FromSlash("math/core.t4")),
	)
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	objPath := filepath.Join(tmp, "math-nosig.tobj")
	if err := WriteObject(objPath, &Object{
		Target:          tgt.Triple,
		Module:          "math.core",
		CompilerVersion: Version(),
		PublicAPIHash:   apiHash,
		Code:            []byte{0xC3},
		Symbols: []Symbol{{
			Name:   "math.core.add",
			Offset: 0,
		}},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}
	iface, err := GenerateInterfaceFromSource(
		src,
		filepath.Join(tmp, filepath.FromSlash("app/math/core.t4")),
	)
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, ("module app.main\nimport math.core as math\nfunc main() -> Int:" +
		"\n    return math.add(40, 2)\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(
		filepath.Join(tmp, filepath.FromSlash("app/math/core.t4i")),
		string(iface),
	); err != nil {
		t.Fatal(err)
	}

	_, err = BuildFileWithStatsOpt(
		appSrc,
		filepath.Join(tmp, "app-bin"+tgt.ExeExt),
		tgt.Triple,
		BuildOptions{LinkObjectPaths: []string{objPath}},
	)
	if err == nil {
		t.Fatalf("expected missing implementation signature metadata error")
	}
	if !strings.Contains(
		err.Error(),
		("implementation object for interface module 'math.core' " +
			"symbol 'math.core.add' missing signature metadata"),
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsInterfaceImplementationGenericExport(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	src := []byte(`module lib.generic

pub func id<T>(x: T) -> T:
    return x
`)
	apiHash, err := InterfaceFingerprintFromSource(
		src,
		filepath.Join(tmp, filepath.FromSlash("lib/generic.t4")),
	)
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	objPath := filepath.Join(tmp, "generic.tobj")
	if err := WriteObject(objPath, &Object{
		Target:          tgt.Triple,
		Module:          "lib.generic",
		CompilerVersion: Version(),
		PublicAPIHash:   apiHash,
		Code:            []byte{0xC3},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}
	iface, err := GenerateInterfaceFromSource(
		src,
		filepath.Join(tmp, filepath.FromSlash("app/lib/generic.t4")),
	)
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	appSrc := filepath.Join(tmp, filepath.FromSlash("app/app/main.t4"))
	if err := writeFile(appSrc, ("module app.main\nimport lib.generic as generic\nfunc main() " +
		"-> Int:\n    return 0\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(
		filepath.Join(tmp, filepath.FromSlash("app/lib/generic.t4i")),
		string(iface),
	); err != nil {
		t.Fatal(err)
	}

	_, err = BuildFileWithStatsOpt(
		appSrc,
		filepath.Join(tmp, "app-bin"+tgt.ExeExt),
		tgt.Triple,
		BuildOptions{LinkObjectPaths: []string{objPath}},
	)
	if err == nil {
		t.Fatalf("expected unsupported generic export error")
	}
	if !strings.Contains(
		err.Error(),
		("implementation object for interface module 'lib.generic' " +
			"cannot satisfy generic export 'lib.generic.id'"),
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadLinkObjectsRejectsDuplicatePath(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	objPath := filepath.Join(tmp, "lib.tobj")
	if err := WriteObject(objPath, &Object{
		Target:  tgt.Triple,
		Module:  "dup.path",
		Code:    []byte{0xC3},
		Symbols: []Symbol{{Name: "dup.path.entry", Offset: 0}},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}

	_, err := readLinkObjects([]string{objPath, filepath.Join(tmp, ".", "lib.tobj")}, tgt.Triple)
	if err == nil {
		t.Fatalf("expected duplicate path error")
	}
	if !strings.Contains(err.Error(), "duplicate link object path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadLinkObjectsRejectsMissingModuleIdentity(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	objPath := filepath.Join(tmp, "nomodule.tobj")
	if err := WriteObject(objPath, &Object{
		Target:  tgt.Triple,
		Code:    []byte{0xC3},
		Symbols: []Symbol{{Name: "nomodule.entry", Offset: 0}},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}

	_, err := readLinkObjects([]string{objPath}, tgt.Triple)
	if err == nil {
		t.Fatalf("expected missing module identity error")
	}
	if !strings.Contains(err.Error(), "link object has no module identity") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadLinkObjectsRejectsDuplicateSymbolsBeforeLinking(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	paths := []string{
		filepath.Join(tmp, "a.tobj"),
		filepath.Join(tmp, "b.tobj"),
	}
	for i, path := range paths {
		if err := WriteObject(path, &Object{
			Target:  tgt.Triple,
			Module:  []string{"a", "b"}[i],
			Code:    []byte{0xC3},
			Symbols: []Symbol{{Name: "shared.symbol", Offset: 0}},
		}); err != nil {
			t.Fatalf("write object %s: %v", path, err)
		}
	}

	_, err := readLinkObjects(paths, tgt.Triple)
	if err == nil {
		t.Fatalf("expected duplicate symbol error")
	}
	if !strings.Contains(err.Error(), "duplicate symbol 'shared.symbol' in link objects") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadLinkObjectsRejectsDuplicateSymbolsInsideObject(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	objPath := filepath.Join(tmp, "dup-symbol.tobj")
	if err := WriteObject(objPath, &Object{
		Target: tgt.Triple,
		Module: "dup.symbol",
		Code:   []byte{0xC3},
		Symbols: []Symbol{
			{Name: "dup.entry", Offset: 0},
			{Name: "dup.entry", Offset: 0},
		},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}

	_, err := readLinkObjects([]string{objPath}, tgt.Triple)
	if err == nil {
		t.Fatalf("expected duplicate symbol error")
	}
	if !strings.Contains(err.Error(), "duplicate symbol 'dup.entry' inside link object") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeFile(path string, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func TestLinkObjectLibraryBuildPath(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	libSrc := filepath.Join(tmp, "lib.tetra")
	libObj := filepath.Join(tmp, "lib.tobj")
	if err := os.WriteFile(
		libSrc,
		[]byte("@export(\"linked_answer\")\nfun answer(): i32 { return 42 }\n"),
		0o644,
	); err != nil {
		t.Fatalf("write library source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(
		libSrc,
		libObj,
		tgt.Triple,
		BuildOptions{Emit: EmitLibrary},
	); err != nil {
		t.Fatalf("build library: %v", err)
	}

	outPath := filepath.Join(tmp, "app"+tgt.ExeExt)
	if _, err := BuildFileWithStatsOpt(filepath.Join(
		"..",
		"examples",
		"smoke",
		"basic",
		"hello.tetra",
	), outPath, tgt.Triple, BuildOptions{
		LinkObjectPaths: []string{libObj},
	}); err != nil {
		t.Fatalf("build with link object: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("missing output: %v", err)
	}
}

func TestRepeatedLinkObjectsAreAccepted(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	var objs []string
	for _, name := range []string{"one", "two"} {
		srcPath := filepath.Join(tmp, name+".tetra")
		objPath := filepath.Join(tmp, name+".tobj")
		src := "@export(\"linked_" + name + "\")\nfun " + name + "(): i32 { return 1 }\n"
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatalf("write source: %v", err)
		}
		if _, err := BuildFileWithStatsOpt(
			srcPath,
			objPath,
			tgt.Triple,
			BuildOptions{Emit: EmitLibrary},
		); err != nil {
			t.Fatalf("build library %s: %v", name, err)
		}
		objs = append(objs, objPath)
	}

	outPath := filepath.Join(tmp, "app"+tgt.ExeExt)
	if _, err := BuildFileWithStatsOpt(filepath.Join(
		"..",
		"examples",
		"smoke",
		"basic",
		"hello.tetra",
	), outPath, tgt.Triple, BuildOptions{
		LinkObjectPaths: objs,
	}); err != nil {
		t.Fatalf("build with repeated link objects: %v", err)
	}
}

func TestLinkObjectDuplicateSymbolDiagnostic(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	var objs []string
	for _, name := range []string{"a", "b"} {
		srcPath := filepath.Join(tmp, name+".tetra")
		objPath := filepath.Join(tmp, name+".tobj")
		if err := os.WriteFile(
			srcPath,
			[]byte("@export(\"dup_symbol\")\nfun "+name+"(): i32 { return 1 }\n"),
			0o644,
		); err != nil {
			t.Fatalf("write source: %v", err)
		}
		if _, err := BuildFileWithStatsOpt(
			srcPath,
			objPath,
			tgt.Triple,
			BuildOptions{Emit: EmitLibrary},
		); err != nil {
			t.Fatalf("build library %s: %v", name, err)
		}
		objs = append(objs, objPath)
	}

	outPath := filepath.Join(tmp, "app"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(
		filepath.Join("..", "examples", "smoke", "basic", "hello.tetra"),
		outPath,
		tgt.Triple,
		BuildOptions{
			LinkObjectPaths: objs,
		},
	)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate symbol 'dup_symbol'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkObjectMissingSymbolDiagnostic(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tmp := t.TempDir()
	objPath := filepath.Join(tmp, "missing.tobj")
	if err := WriteObject(objPath, &Object{
		Target:  tgt.Triple,
		Module:  "__missing_ref",
		Code:    []byte{0xE8, 0, 0, 0, 0, 0xC3},
		Symbols: []Symbol{{Name: "__missing_ref_entry", Offset: 0}},
		Relocs:  []Reloc{{Kind: RelocCallRel32, At: 1, Name: "missing.symbol"}},
	}); err != nil {
		t.Fatalf("write object: %v", err)
	}

	outPath := filepath.Join(tmp, "app"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(
		filepath.Join("..", "examples", "smoke", "basic", "hello.tetra"),
		outPath,
		tgt.Triple,
		BuildOptions{
			LinkObjectPaths: []string{objPath},
		},
	)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unresolved symbol 'missing.symbol'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- manifest_test.go ----

func TestManifestRuntimeABIIncludesFullRequiredSymbolSets(t *testing.T) {
	manifest, err := GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}

	assertSymbolSequence(
		t,
		"actors_required_symbols",
		manifest.RuntimeABI.ActorsRequiredSymbols,
		requiredActorRuntimeSymbols(),
	)
	assertSymbolSequence(
		t,
		"actor_state_required_symbols",
		manifest.RuntimeABI.ActorStateRequiredSymbols,
		requiredActorStateRuntimeSymbols(),
	)
	assertSymbolSequence(
		t,
		"task_required_symbols",
		manifest.RuntimeABI.TaskRequiredSymbols,
		requiredTaskRuntimeSymbols(),
	)
	assertSymbolSequence(
		t,
		"task_group_required_symbols",
		manifest.RuntimeABI.TaskGroupRequiredSymbols,
		requiredTaskGroupRuntimeSymbols(),
	)
	assertSymbolSequence(
		t,
		"typed_task_required_symbols",
		manifest.RuntimeABI.TypedTaskRequiredSymbols,
		requiredTypedTaskRuntimeSymbols(8),
	)
	assertSymbolSequence(
		t,
		"time_required_symbols",
		manifest.RuntimeABI.TimeRequiredSymbols,
		requiredTimeRuntimeSymbols(),
	)
	assertSymbolSequence(
		t,
		"filesystem_required_symbols",
		manifest.RuntimeABI.FilesystemRequiredSymbols,
		requiredFilesystemRuntimeSymbols(),
	)
	assertSymbolSequence(
		t,
		"net_required_symbols",
		manifest.RuntimeABI.NetRequiredSymbols,
		requiredNetRuntimeSymbols(),
	)
	assertSymbolSequence(
		t,
		"surface_required_symbols",
		manifest.RuntimeABI.SurfaceRequiredSymbols,
		requiredSurfaceRuntimeSymbols(),
	)
}

func TestManifestIncludesLinuxNativePromotionGateMetadata(t *testing.T) {
	manifest, err := GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	byTriple := map[string]TargetManifest{}
	for _, target := range manifest.Targets {
		byTriple[target.Triple] = target
	}
	for _, tc := range []struct {
		triple        string
		runtimeStatus string
		stdlibStatus  string
		ffiStatus     string
		artifact      string
	}{
		{
			triple:        "linux-x64",
			runtimeStatus: "production",
			stdlibStatus:  "production",
			ffiStatus:     "scalar_object_smokes_partial",
			artifact:      "linux-x64-runner.json",
		},
		{
			triple:        "linux-x86",
			runtimeStatus: "partial_build_only",
			stdlibStatus:  "partial_build_only",
			ffiStatus:     "ilp32_scalar_object_smokes_partial",
			artifact:      "linux-x86-runner.json",
		},
		{
			triple:        "linux-x32",
			runtimeStatus: "partial_build_only",
			stdlibStatus:  "partial_build_only",
			ffiStatus:     "ilp32_scalar_object_smokes_partial",
			artifact:      "linux-x32-runner.json",
		},
	} {
		got := byTriple[tc.triple]
		if got.Triple == "" {
			t.Fatalf("manifest missing target %s", tc.triple)
		}
		if got.RuntimeStatus != tc.runtimeStatus || got.StdlibStatus != tc.stdlibStatus ||
			got.FFIStatus != tc.ffiStatus {
			t.Fatalf(
				"%s promotion metadata = runtime:%q stdlib:%q ffi:%q",
				tc.triple,
				got.RuntimeStatus,
				got.StdlibStatus,
				got.FFIStatus,
			)
		}
		if got.RunnerProbeCommand == "" ||
			got.ReleaseGate != "scripts/release/post_v0_4/linux-native-targets-smoke.sh" ||
			!stringSliceContains(got.EvidenceArtifacts, tc.artifact) {
			t.Fatalf(
				"%s evidence metadata = runner:%q gate:%q artifacts:%#v",
				tc.triple,
				got.RunnerProbeCommand,
				got.ReleaseGate,
				got.EvidenceArtifacts,
			)
		}
	}
}

func TestManifestIncludesMemoryCapabilityMatrixMetadata(t *testing.T) {
	manifest, err := GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	byTriple := map[string]TargetManifest{}
	for _, target := range manifest.Targets {
		byTriple[target.Triple] = target
	}
	for _, tc := range []struct {
		triple             string
		run                string
		rawDiagnostics     string
		regionLowering     string
		alignmentSemantics string
		claimLevel         string
	}{
		{"linux-x64", "yes", "yes", "yes/partial", "yes", "production/host_runtime"},
		{"linux-x86", "no/host-dependent", "partial", "partial", "partial", "build_lower_only"},
		{"linux-x32", "no/host-dependent", "partial", "partial", "special", "build_lower_only"},
		{
			"macos-x64",
			"host-required",
			"host-required",
			"host-required",
			"host-required",
			"build_lower_only unless run",
		},
		{
			"windows-x64",
			"host-required",
			"host-required",
			"host-required",
			"host-required",
			"build_lower_only unless run",
		},
		{
			"wasm32-wasi",
			"runner-smoke if available",
			"safe-only",
			"limited",
			"wasm rules",
			"artifact/runtime tiered",
		},
		{
			"wasm32-web",
			"browser-smoke if available",
			"safe-only",
			"limited",
			"wasm rules",
			"artifact/runtime tiered",
		},
	} {
		got := byTriple[tc.triple]
		if got.MemoryBuild != "yes" || got.MemoryLower != "yes" || got.MemoryRun != tc.run ||
			got.MemoryRawDiagnostics != tc.rawDiagnostics || got.MemoryRegionLowering != tc.regionLowering ||
			got.MemoryAlignmentSemantics != tc.alignmentSemantics || got.MemoryClaimLevel != tc.claimLevel {
			t.Fatalf(
				("%s memory capability metadata = build:%q lower:%q run:%q " +
					"raw:%q region:%q alignment:%q claim:%q"),
				tc.triple,
				got.MemoryBuild,
				got.MemoryLower,
				got.MemoryRun,
				got.MemoryRawDiagnostics,
				got.MemoryRegionLowering,
				got.MemoryAlignmentSemantics,
				got.MemoryClaimLevel,
			)
		}
	}
}

func TestManifestIncludesLinuxNativeSyscallPackMetadata(t *testing.T) {
	manifest, err := GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	byTriple := map[string]TargetManifest{}
	for _, target := range manifest.Targets {
		byTriple[target.Triple] = target
	}
	for _, tc := range []struct {
		triple      string
		instruction string
		numbering   string
		registers   []string
	}{
		{
			triple:      "linux-x64",
			instruction: "syscall",
			numbering:   "x86_64",
			registers:   []string{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"},
		},
		{
			triple:      "linux-x86",
			instruction: "int 0x80",
			numbering:   "i386",
			registers:   []string{"eax", "ebx", "ecx", "edx", "esi", "edi", "ebp"},
		},
		{
			triple:      "linux-x32",
			instruction: "syscall",
			numbering:   "x32_syscall_bit",
			registers:   []string{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"},
		},
	} {
		got := byTriple[tc.triple]
		if got.SyscallInstruction != tc.instruction || got.SyscallNumbering != tc.numbering ||
			got.SyscallErrorRange != "-4095..-1" ||
			!reflect.DeepEqual(got.SyscallArgRegisters, tc.registers) {
			t.Fatalf(
				"%s syscall metadata = instruction:%q numbering:%q regs:%#v error:%q",
				tc.triple,
				got.SyscallInstruction,
				got.SyscallNumbering,
				got.SyscallArgRegisters,
				got.SyscallErrorRange,
			)
		}
	}
}

func assertSymbolSequence(t *testing.T, field string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
}

func stringSliceContains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestManifestBuiltinsExposeStableUnsafePoliciesForPublicSurface(t *testing.T) {
	manifest, err := GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}

	byName := map[string]BuiltinManifest{}
	for _, builtin := range manifest.Builtins {
		byName[builtin.Name] = builtin
	}

	for name, wantEffects := range map[string]string{
		"core.cap_io":         "capability,io",
		"core.cap_mem":        "capability,mem",
		"core.load_i32":       "mem",
		"core.store_i32":      "mem",
		"core.mmio_read_i32":  "io,mmio",
		"core.mmio_write_i32": "io,mmio",
	} {
		got, ok := byName[name]
		if !ok {
			t.Fatalf("manifest missing builtin %s", name)
		}
		if got.UnsafePolicy != "always" {
			t.Fatalf("%s unsafe_policy = %q, want always", name, got.UnsafePolicy)
		}
		if got.UnsafeDetails != "" {
			t.Fatalf("%s unsafe_details = %q, want empty", name, got.UnsafeDetails)
		}
		if strings.Join(got.Effects, ",") != wantEffects {
			t.Fatalf("%s effects = %q, want %q", name, strings.Join(got.Effects, ","), wantEffects)
		}
	}

	const wantConditionalUnsafeDetails = ("requires unsafe when the island argument is not a scoped " +
		"island variable")
	for _, name := range []string{
		"core.island_make_u8",
		"core.island_make_u16",
		"core.island_make_i32",
		"core.island_make_bool",
	} {
		got, ok := byName[name]
		if !ok {
			t.Fatalf("manifest missing builtin %s", name)
		}
		if got.UnsafePolicy != "conditional" {
			t.Fatalf("%s unsafe_policy = %q, want conditional", name, got.UnsafePolicy)
		}
		if got.UnsafeDetails != wantConditionalUnsafeDetails {
			t.Fatalf(
				"%s unsafe_details = %q, want %q",
				name,
				got.UnsafeDetails,
				wantConditionalUnsafeDetails,
			)
		}
	}
}

// ---- memory_fuzz_oracle_v1_test.go ----

func TestMemoryFuzzOracleReportCoversMPC15CategoriesAndInvariants(t *testing.T) {
	report, err := BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	if report.SchemaVersion != MemoryFuzzOracleSchemaV1 {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, MemoryFuzzOracleSchemaV1)
	}
	if report.Scope != MemoryFuzzOracleScopeMPC15 {
		t.Fatalf("scope = %q, want %q", report.Scope, MemoryFuzzOracleScopeMPC15)
	}
	if err := ValidateMemoryFuzzOracleReport(report); err != nil {
		t.Fatalf("ValidateMemoryFuzzOracleReport: %v", err)
	}
	if report.Tier1ShortCISmokeCases == 0 || !report.Tier2NightlyBoundaryRecorded ||
		!report.Tier3ReleaseBlockingBoundaryRecorded {
		t.Fatalf("tier coverage incomplete: %#v", report)
	}

	rows := map[MemoryFuzzOracleCategory]MemoryFuzzOracleRow{}
	for _, row := range report.Rows {
		if row.Category == "" || row.Tier == "" || row.ExpectedResult == "" || row.Status == "" ||
			len(row.Evidence) == 0 ||
			len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 {
			t.Fatalf("oracle row missing metadata: %#v", row)
		}
		rows[row.Category] = row
	}
	for _, category := range memoryFuzzOracleCategories() {
		if _, ok := rows[category]; !ok {
			t.Fatalf("missing oracle category %s: %#v", category, report.Rows)
		}
	}
	assertMemoryFuzzOracleRow(
		t,
		rows[MemoryFuzzOracleCheckerRejectExpected],
		MemoryFuzzOraclePass,
		[]string{"checker reject", "borrow escape"},
	)
	assertMemoryFuzzOracleRow(
		t,
		rows[MemoryFuzzOracleRuntimeTrapExpected],
		MemoryFuzzOraclePass,
		[]string{"runtime trap", "bounds"},
	)
	assertMemoryFuzzOracleRow(
		t,
		rows[MemoryFuzzOracleReferenceOutputExpected],
		MemoryFuzzOraclePass,
		[]string{"compiled output", "reference"},
	)
	assertMemoryFuzzOracleRow(
		t,
		rows[MemoryFuzzOracleCompilerCrashBug],
		MemoryFuzzOracleBug,
		[]string{"compiler crash", "bug"},
	)
	assertMemoryFuzzOracleRow(
		t,
		rows[MemoryFuzzOracleMiscompileBug],
		MemoryFuzzOracleBug,
		[]string{"miscompile", "bug"},
	)
	assertMemoryFuzzOracleRow(
		t,
		rows[MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug],
		MemoryFuzzOracleBug,
		[]string{"unsafe_unknown", "safe_known"},
	)
	assertMemoryFuzzOracleRow(
		t,
		rows[MemoryFuzzOracleReportValidationFailureBug],
		MemoryFuzzOracleBug,
		[]string{"report validation failure", "MemoryFactGraph"},
	)

	invariants := map[MemoryFuzzInvariantID]MemoryFuzzInvariantRow{}
	for _, row := range report.Invariants {
		invariants[row.ID] = row
	}
	for _, id := range memoryFuzzInvariantIDs() {
		row, ok := invariants[id]
		if !ok {
			t.Fatalf("missing invariant %s: %#v", id, report.Invariants)
		}
		if row.Status != "covered" || len(row.Evidence) == 0 || len(row.Tests) == 0 {
			t.Fatalf("invariant %s incomplete: %#v", id, row)
		}
	}
	for _, nonClaim := range []string{
		"no exhaustive fuzzing is claimed",
		"no unsupported unsafe pointer safety is claimed",
		"no runtime behavior change",
		"no safe-program semantics change",
	} {
		if !memoryFuzzHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestMemoryFuzzOracleRejectsUnknownVocabularyStatus(t *testing.T) {
	report, err := BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	report.GeneratorSurfaces[0].Status = "looks_good_to_me"
	err = ValidateMemoryFuzzOracleReport(report)
	if err == nil {
		t.Fatalf("expected unknown generator surface status to fail")
	}
	if !strings.Contains(err.Error(), "unknown generator surface status") ||
		!strings.Contains(err.Error(), "looks_good_to_me") {
		t.Fatalf("error = %v, want unknown status rejection", err)
	}
	if !memoryvocab.KnownMemoryFuzzStatus(memoryvocab.FuzzStatusCovered) {
		t.Fatalf("shared memory vocabulary must include covered fuzz status")
	}
}

func TestClassifyMemoryFuzzOracleObservation(t *testing.T) {
	tests := []struct {
		name     string
		category MemoryFuzzOracleCategory
		obs      MemoryFuzzObservation
		want     MemoryFuzzOracleResult
	}{
		{
			name:     "checker reject expected",
			category: MemoryFuzzOracleCheckerRejectExpected,
			obs:      MemoryFuzzObservation{CheckerRejected: true},
			want:     MemoryFuzzOraclePass,
		},
		{
			name:     "runtime trap expected",
			category: MemoryFuzzOracleRuntimeTrapExpected,
			obs:      MemoryFuzzObservation{RuntimeTrapped: true},
			want:     MemoryFuzzOraclePass,
		},
		{
			name:     "reference equality expected",
			category: MemoryFuzzOracleReferenceOutputExpected,
			obs: MemoryFuzzObservation{
				ReferenceCompared: true,
				CompiledExitCode:  42,
				ReferenceExitCode: 42,
			},
			want: MemoryFuzzOraclePass,
		},
		{
			name:     "compiler crash is bug",
			category: MemoryFuzzOracleCompilerCrashBug,
			obs:      MemoryFuzzObservation{CompilerCrashed: true},
			want:     MemoryFuzzOracleBug,
		},
		{
			name:     "miscompile is bug",
			category: MemoryFuzzOracleMiscompileBug,
			obs: MemoryFuzzObservation{
				ReferenceCompared: true,
				CompiledExitCode:  7,
				ReferenceExitCode: 9,
			},
			want: MemoryFuzzOracleBug,
		},
		{
			name:     "unsafe_unknown optimized as safe is bug",
			category: MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug,
			obs:      MemoryFuzzObservation{UnsafeUnknownOptimizedAsSafe: true},
			want:     MemoryFuzzOracleBug,
		},
		{
			name:     "report validation failure is bug",
			category: MemoryFuzzOracleReportValidationFailureBug,
			obs:      MemoryFuzzObservation{ReportValidationFailed: true},
			want:     MemoryFuzzOracleBug,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyMemoryFuzzOracleObservation(tc.category, tc.obs)
			if got != tc.want {
				t.Fatalf(
					"ClassifyMemoryFuzzOracleObservation(%s, %#v) = %q, want %q",
					tc.category,
					tc.obs,
					got,
					tc.want,
				)
			}
		})
	}
}

func TestValidateMemoryFuzzOracleReportRejectsDrift(t *testing.T) {
	base, err := BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*MemoryFuzzOracleReport)
		want   string
	}{
		{
			name: "missing oracle category",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.Rows = report.Rows[1:]
			},
			want: "missing oracle_category",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "bug category downgraded",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.RowsByCategory(MemoryFuzzOracleCompilerCrashBug).ExpectedResult = MemoryFuzzOraclePass
			},
			want: "compiler_crash_is_bug",
		},
		{
			name: "missing invariant",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.Invariants = report.Invariants[1:]
			},
			want: "missing invariant",
		},
		{
			name: "missing tier 1",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.Tier1ShortCISmokeCases = 0
			},
			want: "Tier 1",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := cloneMemoryFuzzOracleReport(base)
			tc.mutate(&report)
			err := ValidateMemoryFuzzOracleReport(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateMemoryFuzzOracleReport error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestMemoryFuzzOracleReportCoversV12ReleaseEvidence(t *testing.T) {
	report, err := BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	if err := ValidateMemoryFuzzOracleReport(report); err != nil {
		t.Fatalf("ValidateMemoryFuzzOracleReport: %v", err)
	}

	requirements := map[MemoryFuzzRequirementID]MemoryFuzzRequirementRow{}
	for _, row := range report.Requirements {
		requirements[row.ID] = row
		if row.Status == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 {
			t.Fatalf("requirement %s missing release evidence: %#v", row.ID, row)
		}
	}
	wantRequirementStatuses := map[MemoryFuzzRequirementID]string{
		MemoryFuzzRequirementTier1V0V11Coverage:         "validated_narrow",
		MemoryFuzzRequirementCrashMiscompileArtifacts:   "validated_narrow",
		MemoryFuzzRequirementBlockingMemoryFailures:     "release_blocking",
		MemoryFuzzRequirementTier2NightlySeedTriage:     "boundary_recorded",
		MemoryFuzzRequirementTier3ReleasePassOrClassify: "release_blocking",
	}
	for _, id := range memoryFuzzRequirementIDs() {
		row, ok := requirements[id]
		if !ok {
			t.Fatalf("missing requirement %s: %#v", id, report.Requirements)
		}
		if row.Status != wantRequirementStatuses[id] {
			t.Fatalf(
				"requirement %s status = %q, want %q",
				id,
				row.Status,
				wantRequirementStatuses[id],
			)
		}
	}

	coverage := map[string]MemoryFuzzSliceCoverageRow{}
	for _, row := range report.SliceCoverage {
		coverage[row.SliceID] = row
		if row.Status != "covered" || len(row.Surface) == 0 || len(row.OracleCategories) == 0 ||
			len(row.Invariants) == 0 ||
			len(row.Evidence) == 0 ||
			len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 {
			t.Fatalf("slice coverage %s incomplete: %#v", row.SliceID, row)
		}
	}
	for _, sliceID := range []string{
		"v0",
		"v1",
		"v2",
		"v3",
		"v4",
		"v5",
		"v6",
		"v7",
		"v8",
		"v9",
		"v10",
		"v11",
	} {
		if _, ok := coverage[sliceID]; !ok {
			t.Fatalf(
				"missing deterministic Tier 1 slice coverage %s: %#v",
				sliceID,
				report.SliceCoverage,
			)
		}
	}

	for _, kind := range []string{
		"tier1_short_ci_smoke_summary_json",
		"compiler_crash_reproducer",
		"miscompile_reducer",
		"miscompile_reproducer",
	} {
		if !memoryFuzzHasArtifactKind(report.Artifacts, kind) {
			t.Fatalf("missing required artifact kind %q: %#v", kind, report.Artifacts)
		}
	}

	blocking := map[MemoryFuzzBlockingCaseID]MemoryFuzzBlockingCaseRow{}
	for _, row := range report.BlockingCases {
		blocking[row.ID] = row
		if row.Status != "blocks_release" || !row.BlocksRelease || len(row.Evidence) == 0 ||
			len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 {
			t.Fatalf("blocking case %s incomplete: %#v", row.ID, row)
		}
	}
	for _, id := range memoryFuzzBlockingCaseIDs() {
		if _, ok := blocking[id]; !ok {
			t.Fatalf("missing blocking case %s: %#v", id, report.BlockingCases)
		}
	}

	policies := map[MemoryFuzzTier]MemoryFuzzTierPolicyRow{}
	for _, row := range report.TierPolicies {
		policies[row.Tier] = row
	}
	tier2 := policies[MemoryFuzzTier2Nightly]
	if tier2.Status != "boundary_recorded" || !tier2.SeedsPreserved ||
		!tier2.UnstableTriageRequired ||
		!tier2.MinimizedReproducerRequired {
		t.Fatalf("Tier 2 policy incomplete: %#v", tier2)
	}
	tier3 := policies[MemoryFuzzTier3ReleaseFocused]
	if tier3.Status != "release_blocking" || !tier3.ReleasePromotionBlockedUntilClassified ||
		!tier3.MinimizedReproducerRequired {
		t.Fatalf("Tier 3 policy incomplete: %#v", tier3)
	}
}

func TestValidateMemoryFuzzOracleReportRejectsV12ReleaseEvidenceDrift(t *testing.T) {
	base, err := BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*MemoryFuzzOracleReport)
		want   string
	}{
		{
			name: "missing requirement",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.Requirements = report.Requirements[1:]
			},
			want: "missing requirement MEM-FUZZ-001",
		},
		{
			name: "missing v11 slice coverage",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.SliceCoverage = removeMemoryFuzzSliceCoverage(report.SliceCoverage, "v11")
			},
			want: "missing slice coverage v11",
		},
		{
			name: "compiler crash reproducer missing",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.Artifacts = removeMemoryFuzzArtifactKind(
					report.Artifacts,
					"compiler_crash_reproducer",
				)
			},
			want: "compiler_crash_reproducer",
		},
		{
			name: "miscompile reducer missing",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.Artifacts = removeMemoryFuzzArtifactKind(
					report.Artifacts,
					"miscompile_reducer",
				)
			},
			want: "miscompile_reducer",
		},
		{
			name: "unsafe optimized as safe does not block",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.BlockingCase(MemoryFuzzBlockingUnsafeUnknownOptimizedAsSafe).BlocksRelease = false
			},
			want: "blocks_release",
		},
		{
			name: "tier 2 seed preservation dropped",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.TierPolicy(MemoryFuzzTier2Nightly).SeedsPreserved = false
			},
			want: "Tier 2 nightly fuzz seed preservation",
		},
		{
			name: "tier 3 release classification dropped",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.TierPolicy(MemoryFuzzTier3ReleaseFocused).ReleasePromotionBlockedUntilClassified = false
			},
			want: "Tier 3 release-blocking memory fuzz",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := cloneMemoryFuzzOracleReport(base)
			tc.mutate(&report)
			err := ValidateMemoryFuzzOracleReport(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateMemoryFuzzOracleReport error = %v, want %q", err, tc.want)
			}
		})
	}
}

func assertMemoryFuzzOracleRow(
	t *testing.T,
	row MemoryFuzzOracleRow,
	wantResult MemoryFuzzOracleResult,
	wants []string,
) {
	t.Helper()
	if row.ExpectedResult != wantResult {
		t.Fatalf(
			"row %s expected_result = %q, want %q",
			row.Category,
			row.ExpectedResult,
			wantResult,
		)
	}
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.Category, want, row)
		}
	}
}

func memoryFuzzHasArtifactKind(artifacts []MemoryFuzzArtifact, kind string) bool {
	for _, artifact := range artifacts {
		if artifact.Kind == kind && artifact.Required {
			return true
		}
	}
	return false
}

func removeMemoryFuzzArtifactKind(
	artifacts []MemoryFuzzArtifact,
	kind string,
) []MemoryFuzzArtifact {
	var kept []MemoryFuzzArtifact
	for _, artifact := range artifacts {
		if artifact.Kind != kind {
			kept = append(kept, artifact)
		}
	}
	return kept
}

func removeMemoryFuzzSliceCoverage(
	rows []MemoryFuzzSliceCoverageRow,
	sliceID string,
) []MemoryFuzzSliceCoverageRow {
	var kept []MemoryFuzzSliceCoverageRow
	for _, row := range rows {
		if row.SliceID != sliceID {
			kept = append(kept, row)
		}
	}
	return kept
}

// ---- net_runtime_http_test.go ----

func TestNetRuntimeHTTPPlaintextServerBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close port reservation listener: %v", err)
	}

	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "test")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	srcPath := filepath.Join(srcDir, "http_plaintext_server.tetra")
	src := fmt.Sprintf(`
module test.http_plaintext_server

import lib.core.capability as capability
import lib.core.http as http
import lib.core.net as net

func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let io_cap: cap.io = capability.io()
        let server: Int = net.socket_tcp4(io_cap)
        if server < 0:
            return 40
        if net.set_nonblocking(server, io_cap) < 0:
            let close_nonblocking: Int = net.close(server, io_cap)
            return 41
        if net.bind_tcp4_loopback(server, %d, io_cap) < 0:
            let close_bind: Int = net.close(server, io_cap)
            return 42
        if net.listen(server, 8, io_cap) < 0:
            let close_listen: Int = net.close(server, io_cap)
            return 43
        let epfd: Int = net.epoll_create(io_cap)
        if epfd < 0:
            let close_epoll_server: Int = net.close(server, io_cap)
            return 44
        if net.epoll_ctl_add_read(epfd, server, io_cap) < 0:
            let close_ctl_epfd: Int = net.close(epfd, io_cap)
            let close_ctl_server: Int = net.close(server, io_cap)
            return 45
        let ready: Int = net.epoll_wait_one(epfd, 3000, io_cap)
        if ready != server:
            let close_wait_epfd: Int = net.close(epfd, io_cap)
            let close_wait_server: Int = net.close(server, io_cap)
            return 46
        let client: Int = net.accept(server, io_cap)
        if client < 0:
            let close_accept_epfd: Int = net.close(epfd, io_cap)
            let close_accept_server: Int = net.close(server, io_cap)
            return 47
        var req: []u8 = core.make_u8(512)
        let n: Int = net.read(client, req, 0, 512, io_cap)
        if n <= 0:
            let close_empty_client: Int = net.close(client, io_cap)
            let close_empty_epfd: Int = net.close(epfd, io_cap)
            let close_empty_server: Int = net.close(server, io_cap)
            return 48
        let route: Int = http.route_tech_empower_bytes(req, n)
        if route != http.route_plaintext():
            let close_bad_client: Int = net.close(client, io_cap)
            let close_bad_epfd: Int = net.close(epfd, io_cap)
            let close_bad_server: Int = net.close(server, io_cap)
            return 49
        var resp: []u8 = core.make_u8(192)
        let written: Int = http.write_plaintext_response(resp, "Tetra", "Mon, 01 Jan 2024 00:00:00 GMT", false)
        let sent: Int = net.write(client, resp, 0, written, io_cap)
        let client_closed: Int = net.close(client, io_cap)
        let epfd_closed: Int = net.close(epfd, io_cap)
        let server_closed: Int = net.close(server, io_cap)
        if sent != written:
            return 50
        if client_closed != 0 || epfd_closed != 0 || server_closed != 0:
            return 51
        return 0
    return 52
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "http_plaintext_server")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{
		DependencyRoots: []ModuleRoot{{Root: projectRoot(t)}},
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	conn, err := dialTCP4Localhost(ctx, port)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("dial server: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	request := "GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	if _, err := conn.Write([]byte(request)); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("write client request: %v", err)
	}
	response, err := io.ReadAll(conn)
	if err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf(
			"read client response: %v; stdout=%q stderr=%q",
			err,
			stdout.String(),
			stderr.String(),
		)
	}
	_ = conn.Close()
	got := string(response)
	for _, want := range []string{
		"HTTP/1.1 200 OK\r\n",
		"Server: Tetra\r\n",
		"Date: Mon, 01 Jan 2024 00:00:00 GMT\r\n",
		"Content-Type: text/plain\r\n",
		"Content-Length: 13\r\n",
		"Connection: close\r\n",
		"\r\nHello, World!",
	} {
		if !strings.Contains(got, want) {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.Fatalf("response missing %q:\n%s", want, got)
		}
	}
	err = cmd.Wait()
	if ctx.Err() != nil {
		t.Fatalf(
			"server timed out; stdout=%q stderr=%q response=%q",
			stdout.String(),
			stderr.String(),
			got,
		)
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			t.Fatalf(
				"server exit code %d; stdout=%q stderr=%q response=%q",
				exit.ExitCode(),
				stdout.String(),
				stderr.String(),
				got,
			)
		}
		t.Fatalf(
			"server wait: %v; stdout=%q stderr=%q response=%q",
			err,
			stdout.String(),
			stderr.String(),
			got,
		)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout mismatch: %q", stdout.String())
	}
}

func TestNetRuntimeHTTPPipelinedPlaintextJSONBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close port reservation listener: %v", err)
	}

	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "test")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	srcPath := filepath.Join(srcDir, "http_pipeline_server.tetra")
	src := fmt.Sprintf(`
module test.http_pipeline_server

import lib.core.capability as capability
import lib.core.http as http
import lib.core.net as net

func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let io_cap: cap.io = capability.io()
        let server: Int = net.socket_tcp4(io_cap)
        if server < 0:
            return 60
        if net.set_nonblocking(server, io_cap) < 0:
            let close_nonblocking: Int = net.close(server, io_cap)
            return 61
        if net.bind_tcp4_loopback(server, %d, io_cap) < 0:
            let close_bind: Int = net.close(server, io_cap)
            return 62
        if net.listen(server, 8, io_cap) < 0:
            let close_listen: Int = net.close(server, io_cap)
            return 63
        let epfd: Int = net.epoll_create(io_cap)
        if epfd < 0:
            let close_epoll_server: Int = net.close(server, io_cap)
            return 64
        if net.epoll_ctl_add_read(epfd, server, io_cap) < 0:
            let close_ctl_epfd: Int = net.close(epfd, io_cap)
            let close_ctl_server: Int = net.close(server, io_cap)
            return 65
        let ready: Int = net.epoll_wait_one(epfd, 3000, io_cap)
        if ready != server:
            let close_wait_epfd: Int = net.close(epfd, io_cap)
            let close_wait_server: Int = net.close(server, io_cap)
            return 66
        let client: Int = net.accept(server, io_cap)
        if client < 0:
            let close_accept_epfd: Int = net.close(epfd, io_cap)
            let close_accept_server: Int = net.close(server, io_cap)
            return 67
        var req: []u8 = core.make_u8(768)
        let n: Int = net.read(client, req, 0, 768, io_cap)
        if n <= 0:
            let close_empty_client: Int = net.close(client, io_cap)
            let close_empty_epfd: Int = net.close(epfd, io_cap)
            let close_empty_server: Int = net.close(server, io_cap)
            return 68
        let first_len: Int = http.request_head_len_bytes(req, n)
        if first_len <= 0:
            let close_first_client: Int = net.close(client, io_cap)
            let close_first_epfd: Int = net.close(epfd, io_cap)
            let close_first_server: Int = net.close(server, io_cap)
            return 69
        let second_len: Int = http.request_head_len_bytes_at(req, first_len, n - first_len)
        if second_len <= 0:
            let close_second_client: Int = net.close(client, io_cap)
            let close_second_epfd: Int = net.close(epfd, io_cap)
            let close_second_server: Int = net.close(server, io_cap)
            return 70
        let first_route: Int = http.route_tech_empower_bytes_at(req, 0, first_len)
        let second_route: Int = http.route_tech_empower_bytes_at(req, first_len, second_len)
        if first_route != http.route_plaintext() || second_route != http.route_json():
            let close_route_client: Int = net.close(client, io_cap)
            let close_route_epfd: Int = net.close(epfd, io_cap)
            let close_route_server: Int = net.close(server, io_cap)
            return 71
        let first_keep_alive: Bool = http.request_keep_alive_bytes_at(req, 0, first_len)
        let second_keep_alive: Bool = http.request_keep_alive_bytes_at(req, first_len, second_len)
        if !first_keep_alive || second_keep_alive:
            let close_keep_client: Int = net.close(client, io_cap)
            let close_keep_epfd: Int = net.close(epfd, io_cap)
            let close_keep_server: Int = net.close(server, io_cap)
            return 72
        var plain: []u8 = core.make_u8(192)
        var json: []u8 = core.make_u8(192)
        let plain_len: Int = http.write_plaintext_response(plain, "Tetra", "Mon, 01 Jan 2024 00:00:00 GMT", true)
        let json_len: Int = http.write_json_message_response(json, "Tetra", "Mon, 01 Jan 2024 00:00:00 GMT", "Hello, World!", false)
        let plain_sent: Int = net.write(client, plain, 0, plain_len, io_cap)
        let json_sent: Int = net.write(client, json, 0, json_len, io_cap)
        let client_closed: Int = net.close(client, io_cap)
        let epfd_closed: Int = net.close(epfd, io_cap)
        let server_closed: Int = net.close(server, io_cap)
        if plain_sent != plain_len || json_sent != json_len:
            return 73
        if client_closed != 0 || epfd_closed != 0 || server_closed != 0:
            return 74
        return 0
    return 75
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "http_pipeline_server")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{
		DependencyRoots: []ModuleRoot{{Root: projectRoot(t)}},
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	conn, err := dialTCP4Localhost(ctx, port)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("dial server: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	request := "GET /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n" +
		"GET /json HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	if _, err := conn.Write([]byte(request)); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("write client request: %v", err)
	}
	response, err := io.ReadAll(conn)
	if err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf(
			"read client response: %v; stdout=%q stderr=%q",
			err,
			stdout.String(),
			stderr.String(),
		)
	}
	_ = conn.Close()
	got := string(response)
	for _, want := range []string{
		("HTTP/1.1 200 OK\r\nServer: Tetra\r\nDate: Mon, 01 Jan 2024 00:" +
			"00:00 GMT\r\nContent-Type: text/plain\r\nContent-Length: " +
			"13\r\nConnection: keep-alive\r\n\r\nHello, World!"),
		("HTTP/1.1 200 OK\r\nServer: Tetra\r\nDate: Mon, 01 Jan 2024 00:" +
			"00:00 GMT\r\nContent-Type: application/json\r\nContent-Length: " +
			"27\r\nConnection: close\r\n\r\n{\"message\":\"Hello, World!\"}"),
	} {
		if !strings.Contains(got, want) {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.Fatalf("response missing %q:\n%s", want, got)
		}
	}
	err = cmd.Wait()
	if ctx.Err() != nil {
		t.Fatalf(
			"server timed out; stdout=%q stderr=%q response=%q",
			stdout.String(),
			stderr.String(),
			got,
		)
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			t.Fatalf(
				"server exit code %d; stdout=%q stderr=%q response=%q",
				exit.ExitCode(),
				stdout.String(),
				stderr.String(),
				got,
			)
		}
		t.Fatalf(
			"server wait: %v; stdout=%q stderr=%q response=%q",
			err,
			stdout.String(),
			stderr.String(),
			got,
		)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout mismatch: %q", stdout.String())
	}
}

// ---- net_runtime_linux_x64_epoll_test.go ----

func TestNetRuntimeEpollReadinessBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close port reservation listener: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_epoll_server.tetra")
	src := fmt.Sprintf(`
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let server: Int = core.net_socket_tcp4(cap)
        if server < 0:
            return 20
        if core.net_set_nonblocking(server, cap) < 0:
            let close_nonblocking: Int = core.net_close(server, cap)
            return 21
        if core.net_bind_tcp4_loopback(server, %d, cap) < 0:
            let close_bind: Int = core.net_close(server, cap)
            return 22
        if core.net_listen(server, 8, cap) < 0:
            let close_listen: Int = core.net_close(server, cap)
            return 23
        let epfd: Int = core.net_epoll_create(cap)
        if epfd < 0:
            let close_epoll_server: Int = core.net_close(server, cap)
            return 24
        if core.net_epoll_ctl_add_read(epfd, server, cap) < 0:
            let close_ctl_epfd: Int = core.net_close(epfd, cap)
            let close_ctl_server: Int = core.net_close(server, cap)
            return 25
        let ready: Int = core.net_epoll_wait_one(epfd, 3000, cap)
        if ready != server:
            let close_wait_epfd: Int = core.net_close(epfd, cap)
            let close_wait_server: Int = core.net_close(server, cap)
            return 26
        let client: Int = core.net_accept4(server, 0, cap)
        if client < 0:
            let close_accept_epfd: Int = core.net_close(epfd, cap)
            let close_accept_server: Int = core.net_close(server, cap)
            return 27
        var req: []u8 = core.make_u8(16)
        let n: Int = core.net_read(client, req, 0, 16, cap)
        if n != 4:
            let close_short_client: Int = core.net_close(client, cap)
            let close_short_epfd: Int = core.net_close(epfd, cap)
            let close_short_server: Int = core.net_close(server, cap)
            return 28
        var resp: []u8 = core.make_u8(2)
        resp[0] = 79
        resp[1] = 75
        let written: Int = core.net_write(client, resp, 0, 2, cap)
        let client_closed: Int = core.net_close(client, cap)
        let epfd_closed: Int = core.net_close(epfd, cap)
        let server_closed: Int = core.net_close(server, cap)
        if written != 2:
            return 29
        if client_closed != 0 || epfd_closed != 0 || server_closed != 0:
            return 30
        return 0
    return 31
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "net_epoll_server")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	conn, err := dialTCP4Localhost(ctx, port)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("dial server: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if _, err := conn.Write([]byte("PING")); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("write client request: %v", err)
	}
	reply := make([]byte, 2)
	if _, err := io.ReadFull(conn, reply); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf(
			"read client reply: %v; stdout=%q stderr=%q",
			err,
			stdout.String(),
			stderr.String(),
		)
	}
	_ = conn.Close()
	if string(reply) != "OK" {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("reply = %q, want OK", string(reply))
	}
	err = cmd.Wait()
	if ctx.Err() != nil {
		t.Fatalf("server timed out; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			t.Fatalf(
				"server exit code %d; stdout=%q stderr=%q",
				exit.ExitCode(),
				stdout.String(),
				stderr.String(),
			)
		}
		t.Fatalf("server wait: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout mismatch: %q", stdout.String())
	}
}

func TestNetRuntimeEpollWaitOneIntoBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close port reservation listener: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_epoll_event_server.tetra")
	src := fmt.Sprintf(`
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let server: Int = core.net_socket_tcp4(cap)
        if server < 0:
            return 80
        if core.net_set_nonblocking(server, cap) < 0:
            let close_nonblocking: Int = core.net_close(server, cap)
            return 81
        if core.net_bind_tcp4_loopback(server, %d, cap) < 0:
            let close_bind: Int = core.net_close(server, cap)
            return 82
        if core.net_listen(server, 8, cap) < 0:
            let close_listen: Int = core.net_close(server, cap)
            return 83
        let epfd: Int = core.net_epoll_create(cap)
        if epfd < 0:
            let close_epoll_server: Int = core.net_close(server, cap)
            return 84
        if core.net_epoll_ctl_add_read(epfd, server, cap) < 0:
            let close_ctl_epfd: Int = core.net_close(epfd, cap)
            let close_ctl_server: Int = core.net_close(server, cap)
            return 85
        var event: []i32 = core.make_i32(2)
        let status: Int = core.net_epoll_wait_one_into(epfd, event, 3000, cap)
        if status != 1:
            let close_status_epfd: Int = core.net_close(epfd, cap)
            let close_status_server: Int = core.net_close(server, cap)
            return 86
        if event[0] != server:
            let close_fd_epfd: Int = core.net_close(epfd, cap)
            let close_fd_server: Int = core.net_close(server, cap)
            return 87
        if event[1] %% 2 != 1:
            let close_flags_epfd: Int = core.net_close(epfd, cap)
            let close_flags_server: Int = core.net_close(server, cap)
            return 88
        let client: Int = core.net_accept4(server, 0, cap)
        if client < 0:
            let close_accept_epfd: Int = core.net_close(epfd, cap)
            let close_accept_server: Int = core.net_close(server, cap)
            return 89
        let client_closed: Int = core.net_close(client, cap)
        let epfd_closed: Int = core.net_close(epfd, cap)
        let server_closed: Int = core.net_close(server, cap)
        if client_closed != 0 || epfd_closed != 0 || server_closed != 0:
            return 90
        return 0
    return 91
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "net_epoll_event_server")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	conn, err := dialTCP4Localhost(ctx, port)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("dial server: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	_ = conn.Close()
	err = cmd.Wait()
	if ctx.Err() != nil {
		t.Fatalf("server timed out; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			t.Fatalf(
				"server exit code %d; stdout=%q stderr=%q",
				exit.ExitCode(),
				stdout.String(),
				stderr.String(),
			)
		}
		t.Fatalf("server wait: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout mismatch: %q", stdout.String())
	}
}

func TestNetStdlibAcceptNonblockingAndEpollFlagHelpersBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close port reservation listener: %v", err)
	}

	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "test")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	srcPath := filepath.Join(srcDir, "net_stdlib_event_server.tetra")
	src := fmt.Sprintf(`
module test.net_stdlib_event_server

import lib.core.capability as capability
import lib.core.net as net

func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let io_cap: cap.io = capability.io()
        let server: Int = net.socket_tcp4(io_cap)
        if server < 0:
            return 100
        if net.set_nonblocking(server, io_cap) < 0:
            let close_nonblocking: Int = net.close(server, io_cap)
            return 101
        if net.bind_tcp4_loopback(server, %d, io_cap) < 0:
            let close_bind: Int = net.close(server, io_cap)
            return 102
        if net.listen(server, 8, io_cap) < 0:
            let close_listen: Int = net.close(server, io_cap)
            return 103
        let epfd: Int = net.epoll_create(io_cap)
        if epfd < 0:
            let close_epoll_server: Int = net.close(server, io_cap)
            return 104
        if net.epoll_ctl_add_read(epfd, server, io_cap) < 0:
            let close_ctl_epfd: Int = net.close(epfd, io_cap)
            let close_ctl_server: Int = net.close(server, io_cap)
            return 105
        var event: []i32 = core.make_i32(2)
        let status: Int = net.epoll_wait_one_into(epfd, event, 3000, io_cap)
        if status != 1:
            let close_status_epfd: Int = net.close(epfd, io_cap)
            let close_status_server: Int = net.close(server, io_cap)
            return 106
        if net.epoll_event_fd(event) != server:
            let close_fd_epfd: Int = net.close(epfd, io_cap)
            let close_fd_server: Int = net.close(server, io_cap)
            return 107
        let flags: Int = net.epoll_event_flags(event)
        if !net.epoll_event_readable(flags):
            let close_read_epfd: Int = net.close(epfd, io_cap)
            let close_read_server: Int = net.close(server, io_cap)
            return 108
        if net.epoll_event_writable(flags) || net.epoll_event_has_error(flags):
            let close_flags_epfd: Int = net.close(epfd, io_cap)
            let close_flags_server: Int = net.close(server, io_cap)
            return 109
        let client: Int = net.accept_nonblocking(server, io_cap)
        if client < 0:
            let close_accept_epfd: Int = net.close(epfd, io_cap)
            let close_accept_server: Int = net.close(server, io_cap)
            return 110
        let nodelay: Int = net.set_tcp_nodelay(client, io_cap)
        let client_closed: Int = net.close(client, io_cap)
        let epfd_closed: Int = net.close(epfd, io_cap)
        let server_closed: Int = net.close(server, io_cap)
        if nodelay != 0:
            return 111
        if client_closed != 0 || epfd_closed != 0 || server_closed != 0:
            return 112
        return 0
    return 113
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "net_stdlib_event_server")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{
		DependencyRoots: []ModuleRoot{{Root: projectRoot(t)}},
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	conn, err := dialTCP4Localhost(ctx, port)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("dial server: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	_ = conn.Close()
	err = cmd.Wait()
	if ctx.Err() != nil {
		t.Fatalf("server timed out; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			t.Fatalf(
				"server exit code %d; stdout=%q stderr=%q",
				exit.ExitCode(),
				stdout.String(),
				stderr.String(),
			)
		}
		t.Fatalf("server wait: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout mismatch: %q", stdout.String())
	}
}

// ---- net_runtime_linux_x64_socket_test.go ----

func TestNetRuntimeSocketLifecycleBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 11
        let nb: Int = core.net_set_nonblocking(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if nb < 0:
            return 12
        if closed != 0:
            return 13
    return 0
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want networking socket lifecycle smoke success", exitCode)
	}
}

func TestNetRuntimeSocketOptionsBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 21
        let reuse: Int = core.net_set_reuseport(fd, cap)
        let nodelay: Int = core.net_set_tcp_nodelay(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if reuse != 0:
            return 22
        if nodelay != 0:
            return 23
        if closed != 0:
            return 24
    return 0
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want networking socket options smoke success", exitCode)
	}
}

func TestNetRuntimeEpollControlLifecycleBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 31
        let epfd: Int = core.net_epoll_create(cap)
        if epfd < 0:
            let close_fd: Int = core.net_close(fd, cap)
            return 32
        let add_rw: Int = core.net_epoll_ctl_add_read_write(epfd, fd, cap)
        let mod_read: Int = core.net_epoll_ctl_mod_read(epfd, fd, cap)
        let mod_rw: Int = core.net_epoll_ctl_mod_read_write(epfd, fd, cap)
        let deleted: Int = core.net_epoll_ctl_delete(epfd, fd, cap)
        let epfd_closed: Int = core.net_close(epfd, cap)
        let fd_closed: Int = core.net_close(fd, cap)
        if add_rw != 0:
            return 33
        if mod_read != 0:
            return 34
        if mod_rw != 0:
            return 35
        if deleted != 0:
            return 36
        if epfd_closed != 0 || fd_closed != 0:
            return 37
    return 0
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want epoll control lifecycle smoke success", exitCode)
	}
}

func TestNetRuntimeTCPClientConnectWriteBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("listen local TCP server: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	accepted := make(chan error, 1)
	go func() {
		conn, err := ln.AcceptTCP()
		if err != nil {
			accepted <- err
			return
		}
		defer conn.Close()
		if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			accepted <- err
			return
		}
		got := make([]byte, 2)
		if _, err := io.ReadFull(conn, got); err != nil {
			accepted <- err
			return
		}
		if string(got) != "PG" {
			accepted <- fmt.Errorf("server read %q, want PG", got)
			return
		}
		accepted <- nil
	}()

	stdout, exitCode := buildAndRunWithOptions(t, fmt.Sprintf(`
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 41
        if core.net_connect_tcp4_loopback(fd, %d, cap) != 0:
            let close_connect: Int = core.net_close(fd, cap)
            return 42
        var payload: []u8 = core.make_u8(2)
        payload[0] = 80
        payload[1] = 71
        let written: Int = core.net_write(fd, payload, 0, 2, cap)
        let closed: Int = core.net_close(fd, cap)
        if written != 2:
            return 43
        if closed != 0:
            return 44
    return 0
`, port), BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want TCP client connect/write smoke success", exitCode)
	}
	select {
	case err := <-accepted:
		if err != nil {
			t.Fatalf("accept/read from Tetra client: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("server did not receive Tetra client connection")
	}
}

func TestNetRuntimeTCPServerRecvSendBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close port reservation listener: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_recv_send_server.tetra")
	src := fmt.Sprintf(`
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let server: Int = core.net_socket_tcp4(cap)
        if server < 0:
            return 50
        if core.net_bind_tcp4_loopback(server, %d, cap) < 0:
            let close_bind: Int = core.net_close(server, cap)
            return 51
        if core.net_listen(server, 8, cap) < 0:
            let close_listen: Int = core.net_close(server, cap)
            return 52
        let client: Int = core.net_accept4(server, 0, cap)
        if client < 0:
            let close_accept: Int = core.net_close(server, cap)
            return 53
        var req: []u8 = core.make_u8(8)
        let n: Int = core.net_recv(client, req, 0, 8, cap)
        if n != 4:
            let close_short_client: Int = core.net_close(client, cap)
            let close_short_server: Int = core.net_close(server, cap)
            return 54
        if req[0] != 80 || req[1] != 79 || req[2] != 83 || req[3] != 84:
            let close_bad_client: Int = core.net_close(client, cap)
            let close_bad_server: Int = core.net_close(server, cap)
            return 55
        var resp: []u8 = core.make_u8(4)
        resp[0] = 80
        resp[1] = 79
        resp[2] = 78
        resp[3] = 71
        let sent: Int = core.net_send(client, resp, 0, 4, cap)
        let client_closed: Int = core.net_close(client, cap)
        let server_closed: Int = core.net_close(server, cap)
        if sent != 4:
            return 56
        if client_closed != 0:
            return 57
        if server_closed != 0:
            return 58
        return 0
    return 59
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "net_recv_send_server")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	conn, err := dialTCP4Localhost(ctx, port)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("dial server: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if _, err := conn.Write([]byte("POST")); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("write client request: %v", err)
	}
	reply := make([]byte, 4)
	if _, err := io.ReadFull(conn, reply); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf(
			"read client reply: %v; stdout=%q stderr=%q",
			err,
			stdout.String(),
			stderr.String(),
		)
	}
	_ = conn.Close()
	if string(reply) != "PONG" {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("reply = %q, want PONG", reply)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("server exit: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if stdout.String() != "" || stderr.String() != "" {
		t.Fatalf("server output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestNetRuntimeTCPServerAcceptReadWriteBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close port reservation listener: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_server.tetra")
	src := fmt.Sprintf(`
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let server: Int = core.net_socket_tcp4(cap)
        if server < 0:
            return 10
        if core.net_bind_tcp4_loopback(server, %d, cap) < 0:
            let close_bind: Int = core.net_close(server, cap)
            return 11
        if core.net_listen(server, 8, cap) < 0:
            let close_listen: Int = core.net_close(server, cap)
            return 12
        let client: Int = core.net_accept4(server, 0, cap)
        if client < 0:
            let close_accept: Int = core.net_close(server, cap)
            return 13
        var req: []u8 = core.make_u8(16)
        let n: Int = core.net_read(client, req, 0, 16, cap)
        if n != 4:
            let close_short_client: Int = core.net_close(client, cap)
            let close_short_server: Int = core.net_close(server, cap)
            return 14
        if req[0] != 80 || req[1] != 73 || req[2] != 78 || req[3] != 71:
            let close_bad_client: Int = core.net_close(client, cap)
            let close_bad_server: Int = core.net_close(server, cap)
            return 15
        var resp: []u8 = core.make_u8(2)
        resp[0] = 79
        resp[1] = 75
        let written: Int = core.net_write(client, resp, 0, 2, cap)
        let client_closed: Int = core.net_close(client, cap)
        let server_closed: Int = core.net_close(server, cap)
        if written != 2:
            return 16
        if client_closed != 0:
            return 17
        if server_closed != 0:
            return 18
        return 0
    return 19
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "net_server")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	conn, err := dialTCP4Localhost(ctx, port)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("dial server: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if _, err := conn.Write([]byte("PING")); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("write client request: %v", err)
	}
	reply := make([]byte, 2)
	if _, err := io.ReadFull(conn, reply); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf(
			"read client reply: %v; stdout=%q stderr=%q",
			err,
			stdout.String(),
			stderr.String(),
		)
	}
	_ = conn.Close()
	if string(reply) != "OK" {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("reply = %q, want OK", string(reply))
	}
	err = cmd.Wait()
	if ctx.Err() != nil {
		t.Fatalf("server timed out; stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			t.Fatalf(
				"server exit code %d; stdout=%q stderr=%q",
				exit.ExitCode(),
				stdout.String(),
				stderr.String(),
			)
		}
		t.Fatalf("server wait: %v; stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout mismatch: %q", stdout.String())
	}
}

// ---- net_runtime_target_helpers_test.go ----

func assertELF32Machine(t *testing.T, path string, label string, wantMachine uint16) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s executable: %v", label, err)
	}
	if len(data) < 20 {
		t.Fatalf("%s executable too small: %d bytes", label, len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		t.Fatalf("%s executable missing ELF magic: % x", label, data[:4])
	}
	if data[4] != 1 {
		t.Fatalf("%s executable must use ELFCLASS32, got %d", label, data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != wantMachine {
		t.Fatalf("%s executable machine = %#x, want %#x", label, got, wantMachine)
	}
}

type targetNetworkingSmoke struct {
	target      string
	label       string
	wantMachine uint16
}

func testTargetNetworkingSocketOptions(t *testing.T, smoke targetNetworkingSmoke) {
	t.Helper()
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_options_"+smoke.label+".tetra")
	outPath := filepath.Join(tmp, "net-options-"+smoke.label)
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 21
        let reuse: Int = core.net_set_reuseport(fd, cap)
        let nodelay: Int = core.net_set_tcp_nodelay(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if reuse != 0:
            return 22
        if nodelay != 0:
            return 23
        if closed != 0:
            return 24
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		smoke.target,
		BuildOptions{Jobs: 1},
	); err != nil {
		t.Fatalf("build %s net socket options runtime: %v", smoke.label, err)
	}
	assertELF32Machine(t, outPath, smoke.label+" net socket options", smoke.wantMachine)
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if stdout != "" || code != 0 {
		t.Fatalf(
			"%s net socket options runtime stdout=%q exit=%d, want empty/0",
			smoke.label,
			stdout,
			code,
		)
	}
}

func testTargetNetworkingTCPClientReadWrite(t *testing.T, smoke targetNetworkingSmoke) {
	t.Helper()
	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("listen local TCP server: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	accepted := make(chan error, 1)
	go func() {
		conn, err := ln.AcceptTCP()
		if err != nil {
			accepted <- err
			return
		}
		defer conn.Close()
		if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			accepted <- err
			return
		}
		got := make([]byte, 2)
		if _, err := io.ReadFull(conn, got); err != nil {
			accepted <- err
			return
		}
		if string(got) != "PG" {
			accepted <- fmt.Errorf("server read %q, want PG", got)
			return
		}
		if _, err := conn.Write([]byte("OK")); err != nil {
			accepted <- err
			return
		}
		accepted <- nil
	}()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_client_"+smoke.label+".tetra")
	outPath := filepath.Join(tmp, "net-client-"+smoke.label)
	src := fmt.Sprintf(`
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 41
        if core.net_connect_tcp4_loopback(fd, %d, cap) != 0:
            let close_connect: Int = core.net_close(fd, cap)
            return 42
        var payload: []u8 = core.make_u8(2)
        payload[0] = 80
        payload[1] = 71
        let written: Int = core.net_write(fd, payload, 0, 2, cap)
        if written != 2:
            let close_write: Int = core.net_close(fd, cap)
            return 43
        var reply: []u8 = core.make_u8(2)
        let n: Int = core.net_read(fd, reply, 0, 2, cap)
        let closed: Int = core.net_close(fd, cap)
        if n != 2:
            return 44
        if reply[0] != 79 || reply[1] != 75:
            return 45
        if closed != 0:
            return 46
    return 0
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		smoke.target,
		BuildOptions{Jobs: 1},
	); err != nil {
		t.Fatalf("build %s net client read/write runtime: %v", smoke.label, err)
	}
	assertELF32Machine(t, outPath, smoke.label+" net client read/write", smoke.wantMachine)
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if stdout != "" || code != 0 {
		t.Fatalf(
			"%s net client read/write runtime stdout=%q exit=%d, want empty/0",
			smoke.label,
			stdout,
			code,
		)
	}
	select {
	case err := <-accepted:
		if err != nil {
			t.Fatalf("accept/read/write from %s Tetra client: %v", smoke.label, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("server did not receive %s Tetra client connection", smoke.label)
	}
}

func testTargetNetworkingTCPServerRecvSend(t *testing.T, smoke targetNetworkingSmoke) {
	t.Helper()
	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close port reservation listener: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_server_"+smoke.label+".tetra")
	outPath := filepath.Join(tmp, "net-server-"+smoke.label)
	src := fmt.Sprintf(`
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let server: Int = core.net_socket_tcp4(cap)
        if server < 0:
            return 50
        if core.net_bind_tcp4_loopback(server, %d, cap) < 0:
            let close_bind: Int = core.net_close(server, cap)
            return 51
        if core.net_listen(server, 8, cap) < 0:
            let close_listen: Int = core.net_close(server, cap)
            return 52
        let client: Int = core.net_accept4(server, 0, cap)
        if client < 0:
            let close_accept: Int = core.net_close(server, cap)
            return 53
        var req: []u8 = core.make_u8(8)
        let n: Int = core.net_recv(client, req, 0, 8, cap)
        if n != 4:
            let close_short_client: Int = core.net_close(client, cap)
            let close_short_server: Int = core.net_close(server, cap)
            return 54
        if req[0] != 80 || req[1] != 79 || req[2] != 83 || req[3] != 84:
            let close_bad_client: Int = core.net_close(client, cap)
            let close_bad_server: Int = core.net_close(server, cap)
            return 55
        var resp: []u8 = core.make_u8(4)
        resp[0] = 80
        resp[1] = 79
        resp[2] = 78
        resp[3] = 71
        let sent: Int = core.net_send(client, resp, 0, 4, cap)
        let client_closed: Int = core.net_close(client, cap)
        let server_closed: Int = core.net_close(server, cap)
        if sent != 4:
            return 56
        if client_closed != 0:
            return 57
        if server_closed != 0:
            return 58
        return 0
    return 59
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		smoke.target,
		BuildOptions{Jobs: 1},
	); err != nil {
		t.Fatalf("build %s net server recv/send runtime: %v", smoke.label, err)
	}
	assertELF32Machine(t, outPath, smoke.label+" net server recv/send", smoke.wantMachine)
	runTargetTCPServerRecvSendOrSkip(t, outPath, smoke.label, port)
}

func testTargetNetworkingEpollControlLifecycle(t *testing.T, smoke targetNetworkingSmoke) {
	t.Helper()
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_epoll_control_"+smoke.label+".tetra")
	outPath := filepath.Join(tmp, "net-epoll-control-"+smoke.label)
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 31
        let epfd: Int = core.net_epoll_create(cap)
        if epfd < 0:
            let close_fd: Int = core.net_close(fd, cap)
            return 32
        let add_read: Int = core.net_epoll_ctl_add_read(epfd, fd, cap)
        let mod_read: Int = core.net_epoll_ctl_mod_read(epfd, fd, cap)
        let mod_rw: Int = core.net_epoll_ctl_mod_read_write(epfd, fd, cap)
        let del_read: Int = core.net_epoll_ctl_delete(epfd, fd, cap)
        let add_rw: Int = core.net_epoll_ctl_add_read_write(epfd, fd, cap)
        let del_rw: Int = core.net_epoll_ctl_delete(epfd, fd, cap)
        let epfd_closed: Int = core.net_close(epfd, cap)
        let fd_closed: Int = core.net_close(fd, cap)
        if add_read != 0:
            return 33
        if mod_read != 0:
            return 34
        if mod_rw != 0:
            return 35
        if del_read != 0:
            return 36
        if add_rw != 0:
            return 37
        if del_rw != 0:
            return 38
        if epfd_closed != 0 || fd_closed != 0:
            return 39
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		smoke.target,
		BuildOptions{Jobs: 1},
	); err != nil {
		t.Fatalf("build %s net epoll control runtime: %v", smoke.label, err)
	}
	assertELF32Machine(t, outPath, smoke.label+" net epoll control", smoke.wantMachine)
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if stdout != "" || code != 0 {
		t.Fatalf(
			"%s net epoll control runtime stdout=%q exit=%d, want empty/0",
			smoke.label,
			stdout,
			code,
		)
	}
}

func testTargetNetworkingEpollReadiness(t *testing.T, smoke targetNetworkingSmoke) {
	t.Helper()
	ln, err := netListenTCP4Localhost()
	if err != nil {
		t.Fatalf("reserve local TCP port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close port reservation listener: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_epoll_readiness_"+smoke.label+".tetra")
	outPath := filepath.Join(tmp, "net-epoll-readiness-"+smoke.label)
	src := fmt.Sprintf(`
func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let cap: cap.io = core.cap_io()
        let server: Int = core.net_socket_tcp4(cap)
        if server < 0:
            return 60
        if core.net_set_nonblocking(server, cap) < 0:
            let close_nonblocking: Int = core.net_close(server, cap)
            return 61
        if core.net_bind_tcp4_loopback(server, %d, cap) < 0:
            let close_bind: Int = core.net_close(server, cap)
            return 62
        if core.net_listen(server, 8, cap) < 0:
            let close_listen: Int = core.net_close(server, cap)
            return 63
        let epfd: Int = core.net_epoll_create(cap)
        if epfd < 0:
            let close_epoll_server: Int = core.net_close(server, cap)
            return 64
        if core.net_epoll_ctl_add_read(epfd, server, cap) < 0:
            let close_ctl_epfd: Int = core.net_close(epfd, cap)
            let close_ctl_server: Int = core.net_close(server, cap)
            return 65
        let ready: Int = core.net_epoll_wait_one(epfd, 3000, cap)
        if ready != server:
            let close_ready_epfd: Int = core.net_close(epfd, cap)
            let close_ready_server: Int = core.net_close(server, cap)
            return 66
        var event: []i32 = core.make_i32(2)
        let status: Int = core.net_epoll_wait_one_into(epfd, event, 3000, cap)
        if status != 1:
            let close_status_epfd: Int = core.net_close(epfd, cap)
            let close_status_server: Int = core.net_close(server, cap)
            return 67
        if event[0] != server:
            let close_fd_epfd: Int = core.net_close(epfd, cap)
            let close_fd_server: Int = core.net_close(server, cap)
            return 68
        if event[1] %% 2 != 1:
            let close_flags_epfd: Int = core.net_close(epfd, cap)
            let close_flags_server: Int = core.net_close(server, cap)
            return 69
        let client: Int = core.net_accept4(server, 0, cap)
        if client < 0:
            let close_accept_epfd: Int = core.net_close(epfd, cap)
            let close_accept_server: Int = core.net_close(server, cap)
            return 70
        let client_closed: Int = core.net_close(client, cap)
        let epfd_closed: Int = core.net_close(epfd, cap)
        let server_closed: Int = core.net_close(server, cap)
        if client_closed != 0 || epfd_closed != 0 || server_closed != 0:
            return 71
        return 0
    return 72
`, port)
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		smoke.target,
		BuildOptions{Jobs: 1},
	); err != nil {
		t.Fatalf("build %s net epoll readiness runtime: %v", smoke.label, err)
	}
	assertELF32Machine(t, outPath, smoke.label+" net epoll readiness", smoke.wantMachine)
	runTargetTCPServerReadinessOrSkip(t, outPath, smoke.label, port)
}

func runTargetTCPServerReadinessOrSkip(t *testing.T, outPath string, label string, port int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		if isUnsupportedTargetExecError(err, stdout.String()+stderr.String()) {
			t.Skipf("host cannot execute %s target binary %s: %v", label, outPath, err)
		}
		t.Fatalf("start %s readiness server: %v", label, err)
	}
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()
	conn, waitResult, err := dialTCP4LocalhostOrTargetExit(ctx, port, waitCh)
	if err != nil {
		if waitResult != nil {
			handleTargetProcessExitBeforeDial(
				t,
				outPath,
				label+" readiness server",
				waitResult.err,
				stdout.String(),
				stderr.String(),
			)
		}
		_ = cmd.Process.Kill()
		<-waitCh
		t.Fatalf(
			"dial %s readiness server: %v; stdout=%q stderr=%q",
			label,
			err,
			stdout.String(),
			stderr.String(),
		)
	}
	_ = conn.Close()
	err = <-waitCh
	if ctx.Err() != nil {
		t.Fatalf(
			"%s readiness server timed out; stdout=%q stderr=%q",
			label,
			stdout.String(),
			stderr.String(),
		)
	}
	if err != nil {
		handleTargetProcessWaitError(
			t,
			outPath,
			label+" readiness server",
			err,
			stdout.String(),
			stderr.String(),
		)
	}
	if stdout.String() != "" || stderr.String() != "" {
		t.Fatalf(
			"%s readiness server output stdout=%q stderr=%q",
			label,
			stdout.String(),
			stderr.String(),
		)
	}
}

func runTargetTCPServerRecvSendOrSkip(t *testing.T, outPath string, label string, port int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		if isUnsupportedTargetExecError(err, stdout.String()+stderr.String()) {
			t.Skipf("host cannot execute %s target binary %s: %v", label, outPath, err)
		}
		t.Fatalf("start %s server: %v", label, err)
	}
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()
	conn, waitResult, err := dialTCP4LocalhostOrTargetExit(ctx, port, waitCh)
	if err != nil {
		if waitResult != nil {
			handleTargetProcessExitBeforeDial(
				t,
				outPath,
				label+" server",
				waitResult.err,
				stdout.String(),
				stderr.String(),
			)
		}
		_ = cmd.Process.Kill()
		<-waitCh
		t.Fatalf(
			"dial %s server: %v; stdout=%q stderr=%q",
			label,
			err,
			stdout.String(),
			stderr.String(),
		)
	}
	if _, err := conn.Write([]byte("POST")); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		<-waitCh
		t.Fatalf("write %s client request: %v", label, err)
	}
	reply := make([]byte, 4)
	if _, err := io.ReadFull(conn, reply); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Kill()
		<-waitCh
		t.Fatalf(
			"read %s client reply: %v; stdout=%q stderr=%q",
			label,
			err,
			stdout.String(),
			stderr.String(),
		)
	}
	_ = conn.Close()
	if string(reply) != "PONG" {
		_ = cmd.Process.Kill()
		<-waitCh
		t.Fatalf("%s reply = %q, want PONG", label, reply)
	}
	err = <-waitCh
	if ctx.Err() != nil {
		t.Fatalf(
			"%s server timed out; stdout=%q stderr=%q",
			label,
			stdout.String(),
			stderr.String(),
		)
	}
	if err != nil {
		handleTargetProcessWaitError(
			t,
			outPath,
			label+" server",
			err,
			stdout.String(),
			stderr.String(),
		)
	}
	if stdout.String() != "" || stderr.String() != "" {
		t.Fatalf("%s server output stdout=%q stderr=%q", label, stdout.String(), stderr.String())
	}
}

type targetWaitResult struct {
	err error
}

func dialTCP4LocalhostOrTargetExit(
	ctx context.Context,
	port int,
	waitCh <-chan error,
) (*net.TCPConn, *targetWaitResult, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	var lastErr error
	for ctx.Err() == nil {
		select {
		case err := <-waitCh:
			return nil, &targetWaitResult{
					err: err,
				}, fmt.Errorf(
					"target process exited before accepting TCP connections",
				)
		default:
		}

		dialer := net.Dialer{Timeout: 50 * time.Millisecond}
		conn, err := dialer.DialContext(ctx, "tcp4", addr)
		if err == nil {
			return conn.(*net.TCPConn), nil, nil
		}
		lastErr = err

		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case err := <-waitCh:
			timer.Stop()
			return nil, &targetWaitResult{
					err: err,
				}, fmt.Errorf(
					"target process exited before accepting TCP connections",
				)
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	if lastErr != nil {
		return nil, nil, lastErr
	}
	return nil, nil, ctx.Err()
}

func handleTargetProcessExitBeforeDial(
	t *testing.T,
	outPath string,
	label string,
	err error,
	stdout string,
	stderr string,
) {
	t.Helper()
	if err == nil {
		t.Fatalf(
			"%s exited before accepting TCP connections; stdout=%q stderr=%q",
			label,
			stdout,
			stderr,
		)
	}
	handleTargetProcessWaitError(t, outPath, label, err, stdout, stderr)
}

func handleTargetProcessWaitError(
	t *testing.T,
	outPath string,
	label string,
	err error,
	stdout string,
	stderr string,
) {
	t.Helper()
	if isUnsupportedTargetSignalExit(err, syscall.SIGSYS) {
		t.Skipf(
			("host kernel rejected %s target binary %s with SIGSYS; " +
				"target execution is unsupported in this environment"),
			label,
			outPath,
		)
	}
	if exit, ok := err.(*exec.ExitError); ok {
		if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			t.Fatalf(
				"%s exited from signal %s; stdout=%q stderr=%q",
				label,
				status.Signal(),
				stdout,
				stderr,
			)
		}
		t.Fatalf("%s exit code %d; stdout=%q stderr=%q", label, exit.ExitCode(), stdout, stderr)
	}
	t.Fatalf("%s wait: %v; stdout=%q stderr=%q", label, err, stdout, stderr)
}

func isUnsupportedTargetSignalExit(err error, signal syscall.Signal) bool {
	if err == nil {
		return false
	}
	exit, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	status, ok := exit.Sys().(syscall.WaitStatus)
	return ok && status.Signaled() && status.Signal() == signal
}

func isUnsupportedTargetExecError(err error, output string) bool {
	if err == nil {
		return false
	}
	text := err.Error() + " " + output
	return strings.Contains(text, "exec format error") ||
		strings.Contains(text, "no such file or directory")
}

func runtimeObjectWithNetRuntimeSignatures() *Object {
	obj := &Object{}
	for _, name := range requiredNetRuntimeSymbols() {
		sig, ok := runtimeObjectSignature(name)
		if !ok {
			panic("missing networking runtime signature for " + name)
		}
		obj.Symbols = append(obj.Symbols, Symbol{
			Name:         name,
			HasSignature: true,
			ParamSlots:   sig.paramSlots,
			ReturnSlots:  sig.returnSlots,
		})
	}
	return obj
}

func netListenTCP4Localhost() (*net.TCPListener, error) {
	addr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	return net.ListenTCP("tcp4", addr)
}

func dialTCP4Localhost(ctx context.Context, port int) (*net.TCPConn, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	var lastErr error
	for ctx.Err() == nil {
		dialer := net.Dialer{Timeout: 50 * time.Millisecond}
		conn, err := dialer.DialContext(ctx, "tcp4", addr)
		if err == nil {
			return conn.(*net.TCPConn), nil
		}
		lastErr = err
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ctx.Err()
}

// ---- net_runtime_target_smoke_test.go ----

func TestNetRuntimeRejectsUnsupportedNativeTargets(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        return core.net_epoll_create(cap)
    return 1
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	for _, tc := range []struct {
		target string
		want   string
	}{
		{target: "macos-x64", want: "macos-x64"},
		{target: "windows-x64", want: "windows-x64"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			outPath := filepath.Join(tmp, "net-"+tc.target)
			_, err := BuildFileWithStatsOpt(srcPath, outPath, tc.target, BuildOptions{Jobs: 1})
			if err == nil {
				t.Fatalf("expected unsupported networking runtime diagnostic")
			}
			want := "networking runtime not supported on " + tc.want
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want %q", err, want)
			}
		})
	}
}

func TestX86NetworkingLifecycleRuntimeBuildsAndRunsWhenHostSupportsX86(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_socket_lifecycle_x86.tetra")
	outPath := filepath.Join(tmp, "net-socket-lifecycle-x86")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 11
        let nb: Int = core.net_set_nonblocking(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if nb < 0:
            return 12
        if closed != 0:
            return 13
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 net socket lifecycle runtime: %v", err)
	}
	assertELF32Machine(t, outPath, "x86 net socket lifecycle", 0x03)
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if stdout != "" || code != 0 {
		t.Fatalf("x86 net socket lifecycle runtime stdout=%q exit=%d, want empty/0", stdout, code)
	}
}

func TestX32NetworkingLifecycleRuntimeBuildsAndRunsWhenHostSupportsX32(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_socket_lifecycle_x32.tetra")
	outPath := filepath.Join(tmp, "net-socket-lifecycle-x32")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 11
        let nb: Int = core.net_set_nonblocking(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if nb < 0:
            return 12
        if closed != 0:
            return 13
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x32 net socket lifecycle runtime: %v", err)
	}
	assertELF32Machine(t, outPath, "x32 net socket lifecycle", 0x3e)
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if stdout != "" || code != 0 {
		t.Fatalf("x32 net socket lifecycle runtime stdout=%q exit=%d, want empty/0", stdout, code)
	}
}

func TestX86NetworkingSocketOptionsBuildsAndRunsWhenHostSupportsX86(t *testing.T) {
	testTargetNetworkingSocketOptions(t, targetNetworkingSmoke{
		target:      "x86",
		label:       "x86",
		wantMachine: 0x03,
	})
}

func TestX32NetworkingSocketOptionsBuildsAndRunsWhenHostSupportsX32(t *testing.T) {
	testTargetNetworkingSocketOptions(t, targetNetworkingSmoke{
		target:      "x32",
		label:       "x32",
		wantMachine: 0x3e,
	})
}

func TestX86NetworkingTCPClientReadWriteBuildsAndRunsWhenHostSupportsX86(t *testing.T) {
	testTargetNetworkingTCPClientReadWrite(t, targetNetworkingSmoke{
		target:      "x86",
		label:       "x86",
		wantMachine: 0x03,
	})
}

func TestX32NetworkingTCPClientReadWriteBuildsAndRunsWhenHostSupportsX32(t *testing.T) {
	testTargetNetworkingTCPClientReadWrite(t, targetNetworkingSmoke{
		target:      "x32",
		label:       "x32",
		wantMachine: 0x3e,
	})
}

func TestX86NetworkingTCPServerRecvSendBuildsAndRunsWhenHostSupportsX86(t *testing.T) {
	testTargetNetworkingTCPServerRecvSend(t, targetNetworkingSmoke{
		target:      "x86",
		label:       "x86",
		wantMachine: 0x03,
	})
}

func TestX32NetworkingTCPServerRecvSendBuildsAndRunsWhenHostSupportsX32(t *testing.T) {
	testTargetNetworkingTCPServerRecvSend(t, targetNetworkingSmoke{
		target:      "x32",
		label:       "x32",
		wantMachine: 0x3e,
	})
}

func TestX86NetworkingEpollControlLifecycleBuildsAndRunsWhenHostSupportsX86(t *testing.T) {
	testTargetNetworkingEpollControlLifecycle(t, targetNetworkingSmoke{
		target:      "x86",
		label:       "x86",
		wantMachine: 0x03,
	})
}

func TestX32NetworkingEpollControlLifecycleBuildsAndRunsWhenHostSupportsX32(t *testing.T) {
	testTargetNetworkingEpollControlLifecycle(t, targetNetworkingSmoke{
		target:      "x32",
		label:       "x32",
		wantMachine: 0x3e,
	})
}

func TestX86NetworkingEpollReadinessBuildsAndRunsWhenHostSupportsX86(t *testing.T) {
	testTargetNetworkingEpollReadiness(t, targetNetworkingSmoke{
		target:      "x86",
		label:       "x86",
		wantMachine: 0x03,
	})
}

func TestX32NetworkingEpollReadinessBuildsAndRunsWhenHostSupportsX32(t *testing.T) {
	testTargetNetworkingEpollReadiness(t, targetNetworkingSmoke{
		target:      "x32",
		label:       "x32",
		wantMachine: 0x3e,
	})
}

func TestX86NetworkingLifecycleRuntimeComposesWithTaskScheduler(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_socket_task_x86.tetra")
	outPath := filepath.Join(tmp, "net-socket-task-x86")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 7

func main() -> Int
uses capability, io, runtime:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 11
        let nb: Int = core.net_set_nonblocking(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if nb < 0:
            return 12
        if closed != 0:
            return 13
    let task: task.i32 = core.task_spawn_i32("worker")
    let value: Int = core.task_join_i32(task)
    return value - 7
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 net socket task runtime: %v", err)
	}
	assertELF32Machine(t, outPath, "x86 net socket task", 0x03)
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if stdout != "" || code != 0 {
		t.Fatalf("x86 net socket task runtime stdout=%q exit=%d, want empty/0", stdout, code)
	}
}

func TestX32NetworkingLifecycleRuntimeComposesWithTaskScheduler(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_socket_task_x32.tetra")
	outPath := filepath.Join(tmp, "net-socket-task-x32")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 7

func main() -> Int
uses capability, io, runtime:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 11
        let nb: Int = core.net_set_nonblocking(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if nb < 0:
            return 12
        if closed != 0:
            return 13
    let task: task.i32 = core.task_spawn_i32("worker")
    let value: Int = core.task_join_i32(task)
    return value - 7
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x32 net socket task runtime: %v", err)
	}
	assertELF32Machine(t, outPath, "x32 net socket task", 0x3e)
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if stdout != "" || code != 0 {
		t.Fatalf("x32 net socket task runtime stdout=%q exit=%d, want empty/0", stdout, code)
	}
}

// ---- net_runtime_test.go ----

func TestNetRuntimeRequiredSymbolsAndSignatures(t *testing.T) {
	got := requiredNetRuntimeSymbols()
	want := []string{
		"__tetra_net_socket_tcp4",
		"__tetra_net_bind_tcp4_loopback",
		"__tetra_net_connect_tcp4_loopback",
		"__tetra_net_listen",
		"__tetra_net_accept4",
		"__tetra_net_read",
		"__tetra_net_recv",
		"__tetra_net_write",
		"__tetra_net_send",
		"__tetra_net_epoll_create",
		"__tetra_net_epoll_ctl_add_read",
		"__tetra_net_epoll_ctl_add_read_write",
		"__tetra_net_epoll_ctl_mod_read",
		"__tetra_net_epoll_ctl_mod_read_write",
		"__tetra_net_epoll_ctl_delete",
		"__tetra_net_epoll_wait_one",
		"__tetra_net_epoll_wait_one_into",
		"__tetra_net_set_nonblocking",
		"__tetra_net_set_reuseport",
		"__tetra_net_set_tcp_nodelay",
		"__tetra_net_close",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("networking runtime symbols = %#v, want %#v", got, want)
	}
	tests := []struct {
		name   string
		params int
		rets   int
	}{
		{name: "__tetra_net_socket_tcp4", params: 1, rets: 1},
		{name: "__tetra_net_bind_tcp4_loopback", params: 3, rets: 1},
		{name: "__tetra_net_connect_tcp4_loopback", params: 3, rets: 1},
		{name: "__tetra_net_listen", params: 3, rets: 1},
		{name: "__tetra_net_accept4", params: 3, rets: 1},
		{name: "__tetra_net_read", params: 6, rets: 1},
		{name: "__tetra_net_recv", params: 6, rets: 1},
		{name: "__tetra_net_write", params: 6, rets: 1},
		{name: "__tetra_net_send", params: 6, rets: 1},
		{name: "__tetra_net_epoll_create", params: 1, rets: 1},
		{name: "__tetra_net_epoll_ctl_add_read", params: 3, rets: 1},
		{name: "__tetra_net_epoll_ctl_add_read_write", params: 3, rets: 1},
		{name: "__tetra_net_epoll_ctl_mod_read", params: 3, rets: 1},
		{name: "__tetra_net_epoll_ctl_mod_read_write", params: 3, rets: 1},
		{name: "__tetra_net_epoll_ctl_delete", params: 3, rets: 1},
		{name: "__tetra_net_epoll_wait_one", params: 3, rets: 1},
		{name: "__tetra_net_epoll_wait_one_into", params: 5, rets: 1},
		{name: "__tetra_net_set_nonblocking", params: 2, rets: 1},
		{name: "__tetra_net_set_reuseport", params: 2, rets: 1},
		{name: "__tetra_net_set_tcp_nodelay", params: 2, rets: 1},
		{name: "__tetra_net_close", params: 2, rets: 1},
	}
	for _, tt := range tests {
		sig, ok := runtimeObjectSignature(tt.name)
		if !ok {
			t.Fatalf("missing runtime signature for %s", tt.name)
		}
		if sig.paramSlots != tt.params || sig.returnSlots != tt.rets {
			t.Fatalf(
				"%s signature = params %d returns %d, want params %d returns %d",
				tt.name,
				sig.paramSlots,
				sig.returnSlots,
				tt.params,
				tt.rets,
			)
		}
	}
}

func TestLinuxX86BasicNetRuntimeObjectExportsSocketNonblockingClose(t *testing.T) {
	rt := buildLinuxX86BasicNetRuntimeObject()
	if rt.Target != "linux-x86" {
		t.Fatalf("runtime target = %q, want linux-x86", rt.Target)
	}
	if rt.Module != "__linux_x86_netrt" {
		t.Fatalf("runtime module = %q, want __linux_x86_netrt", rt.Module)
	}
	wantSymbols := []string{
		"__tetra_net_socket_tcp4",
		"__tetra_net_bind_tcp4_loopback",
		"__tetra_net_connect_tcp4_loopback",
		"__tetra_net_listen",
		"__tetra_net_accept4",
		"__tetra_net_read",
		"__tetra_net_recv",
		"__tetra_net_write",
		"__tetra_net_send",
		"__tetra_net_epoll_create",
		"__tetra_net_epoll_ctl_add_read",
		"__tetra_net_epoll_ctl_add_read_write",
		"__tetra_net_epoll_ctl_mod_read",
		"__tetra_net_epoll_ctl_mod_read_write",
		"__tetra_net_epoll_ctl_delete",
		"__tetra_net_epoll_wait_one",
		"__tetra_net_epoll_wait_one_into",
		"__tetra_net_set_nonblocking",
		"__tetra_net_set_reuseport",
		"__tetra_net_set_tcp_nodelay",
		"__tetra_net_close",
	}
	if !runtimeSymbolsMatch(rt.Symbols, wantSymbols) {
		t.Fatalf("runtime symbols = %#v, want %v in offset order", rt.Symbols, wantSymbols)
	}
	if len(rt.Data) != 0 || len(rt.Relocs) != 0 {
		t.Fatalf(
			"runtime object must be self-contained, data=%d relocs=%#v",
			len(rt.Data),
			rt.Relocs,
		)
	}
	annotateRuntimeObjectSignatures(rt)
	if err := validateRuntimeObjectSymbols(
		rt,
		"missing networking runtime object",
		wantSymbols,
	); err != nil {
		t.Fatalf("validate x86 basic net runtime object: %v", err)
	}
	for name, needle := range map[string][]byte{
		"socketcall syscall": {0xB8, 0x66, 0x00, 0x00, 0x00},
		"socket operation":   {0xBB, 0x01, 0x00, 0x00, 0x00},
		"bind operation":     {0xBB, 0x02, 0x00, 0x00, 0x00},
		"connect operation":  {0xBB, 0x03, 0x00, 0x00, 0x00},
		"listen operation":   {0xBB, 0x04, 0x00, 0x00, 0x00},
		"send operation":     {0xBB, 0x09, 0x00, 0x00, 0x00},
		"recv operation":     {0xBB, 0x0A, 0x00, 0x00, 0x00},
		"setsockopt op":      {0xBB, 0x0E, 0x00, 0x00, 0x00},
		"accept4 operation":  {0xBB, 0x12, 0x00, 0x00, 0x00},
		"read syscall":       {0xB8, 0x03, 0x00, 0x00, 0x00},
		"write syscall":      {0xB8, 0x04, 0x00, 0x00, 0x00},
		"epoll_create1":      {0xB8, 0x49, 0x01, 0x00, 0x00},
		"epoll_ctl":          {0xB8, 0xFF, 0x00, 0x00, 0x00},
		"epoll_wait":         {0xB8, 0x00, 0x01, 0x00, 0x00},
		"fcntl syscall":      {0xB8, 0x37, 0x00, 0x00, 0x00},
		"nonblocking flag":   {0x0D, 0x00, 0x08, 0x00, 0x00},
		"close syscall":      {0xB8, 0x06, 0x00, 0x00, 0x00},
		"int80 syscall":      {0xCD, 0x80},
		"preserved return":   {0x5B, 0x5D, 0xC3},
	} {
		if !bytes.Contains(rt.Code, needle) {
			t.Fatalf("runtime code missing %s sequence % x in % x", name, needle, rt.Code)
		}
	}
}

func TestLinuxX32BasicNetRuntimeObjectExportsSocketNonblockingClose(t *testing.T) {
	rt := buildLinuxX32BasicNetRuntimeObject()
	if rt.Target != "linux-x32" {
		t.Fatalf("runtime target = %q, want linux-x32", rt.Target)
	}
	if rt.Module != "__linux_x32_netrt" {
		t.Fatalf("runtime module = %q, want __linux_x32_netrt", rt.Module)
	}
	wantSymbols := []string{
		"__tetra_net_socket_tcp4",
		"__tetra_net_bind_tcp4_loopback",
		"__tetra_net_connect_tcp4_loopback",
		"__tetra_net_listen",
		"__tetra_net_accept4",
		"__tetra_net_read",
		"__tetra_net_recv",
		"__tetra_net_write",
		"__tetra_net_send",
		"__tetra_net_epoll_create",
		"__tetra_net_epoll_ctl_add_read",
		"__tetra_net_epoll_ctl_add_read_write",
		"__tetra_net_epoll_ctl_mod_read",
		"__tetra_net_epoll_ctl_mod_read_write",
		"__tetra_net_epoll_ctl_delete",
		"__tetra_net_epoll_wait_one",
		"__tetra_net_epoll_wait_one_into",
		"__tetra_net_set_nonblocking",
		"__tetra_net_set_reuseport",
		"__tetra_net_set_tcp_nodelay",
		"__tetra_net_close",
	}
	if !runtimeSymbolsMatch(rt.Symbols, wantSymbols) {
		t.Fatalf("runtime symbols = %#v, want %v in offset order", rt.Symbols, wantSymbols)
	}
	if len(rt.Data) != 0 || len(rt.Relocs) != 0 {
		t.Fatalf(
			"runtime object must be self-contained, data=%d relocs=%#v",
			len(rt.Data),
			rt.Relocs,
		)
	}
	annotateRuntimeObjectSignatures(rt)
	if err := validateRuntimeObjectSymbols(
		rt,
		"missing networking runtime object",
		wantSymbols,
	); err != nil {
		t.Fatalf("validate x32 basic net runtime object: %v", err)
	}
	for name, needle := range map[string][]byte{
		"x32 socket syscall":  {0xB8, 0x29, 0x00, 0x00, 0x40},
		"x32 bind syscall":    {0xB8, 0x31, 0x00, 0x00, 0x40},
		"x32 connect syscall": {0xB8, 0x2A, 0x00, 0x00, 0x40},
		"x32 listen syscall":  {0xB8, 0x32, 0x00, 0x00, 0x40},
		"x32 accept4 syscall": {0xB8, 0x20, 0x01, 0x00, 0x40},
		"x32 read syscall":    {0xB8, 0x00, 0x00, 0x00, 0x40},
		"x32 write syscall":   {0xB8, 0x01, 0x00, 0x00, 0x40},
		"x32 send syscall":    {0xB8, 0x2C, 0x00, 0x00, 0x40},
		"x32 recv syscall":    {0xB8, 0x05, 0x02, 0x00, 0x40},
		"x32 setsockopt":      {0xB8, 0x1D, 0x02, 0x00, 0x40},
		"x32 epoll_wait":      {0xB8, 0xE8, 0x00, 0x00, 0x40},
		"x32 epoll_ctl":       {0xB8, 0xE9, 0x00, 0x00, 0x40},
		"x32 epoll_create1":   {0xB8, 0x23, 0x01, 0x00, 0x40},
		"x32 fcntl syscall":   {0xB8, 0x48, 0x00, 0x00, 0x40},
		"nonblocking flag":    {0x0D, 0x00, 0x08, 0x00, 0x00},
		"x32 close syscall":   {0xB8, 0x03, 0x00, 0x00, 0x40},
		"syscall instruction": {0x0F, 0x05},
		"return":              {0xC3},
	} {
		if !bytes.Contains(rt.Code, needle) {
			t.Fatalf("runtime code missing %s sequence % x in % x", name, needle, rt.Code)
		}
	}
	if bytes.Contains(rt.Code, []byte{0xB8, 0x03, 0x00, 0x00, 0x00}) {
		t.Fatalf("x32 net close runtime emitted plain x64 close syscall: % x", rt.Code)
	}
}

func runtimeSymbolsMatch(symbols []Symbol, names []string) bool {
	if len(symbols) != len(names) {
		return false
	}
	var last uint32
	for i, name := range names {
		if symbols[i].Name != name {
			return false
		}
		if i > 0 && symbols[i].Offset <= last {
			return false
		}
		last = symbols[i].Offset
	}
	return true
}

func TestCollectNetRuntimeUsage(t *testing.T) {
	prog, err := Parse([]byte(`
func probe(cap: cap.io) -> Int
uses io:
    let fd: Int = core.net_socket_tcp4(cap)
    return core.net_close(fd, cap)

func main() -> Int:
    return 0
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !collectNetRuntimeUsage(checked) {
		t.Fatalf("networking runtime usage was not collected")
	}
}

func TestValidateNetRuntimeObjectChecksSignatureMetadata(t *testing.T) {
	obj := runtimeObjectWithNetRuntimeSignatures()
	if err := validateNetRuntimeObject(obj); err != nil {
		t.Fatalf("validate networking runtime object: %v", err)
	}

	replaceRuntimeSymbolSignature(obj, "__tetra_net_set_nonblocking", 1, 1)
	err := validateNetRuntimeObject(obj)
	if err == nil {
		t.Fatalf("expected networking runtime signature mismatch")
	}
	if !strings.Contains(
		err.Error(),
		"runtime object symbol '__tetra_net_set_nonblocking' signature mismatch",
	) ||
		!strings.Contains(err.Error(), "params=1 want=2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingNetSymbols(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if tgt.Triple != "linux-x64" {
		t.Skipf("networking runtime is linux-x64 only, host is %s", tgt.Triple)
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_net.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing_net",
		Code:    []byte{0xC3},
		Symbols: runtimeObjectSymbols(requiredActorRuntimeSymbols()),
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	srcPath := filepath.Join(tmp, "net_main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        return core.net_close(fd, cap)
    return 1
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "net_main"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{RuntimeObjectPath: rtPath},
	)
	if err == nil {
		t.Fatalf("expected missing networking runtime symbol failure")
	}
	if !strings.Contains(
		err.Error(),
		"runtime object missing required symbol '__tetra_net_socket_tcp4'",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---- protocol_trait_object_decision_test.go ----

func TestP22ProtocolTraitObjectDecisionKeepsStaticFastPath(t *testing.T) {
	report, err := BuildP22ProtocolTraitObjectDecision()
	if err != nil {
		t.Fatalf("BuildP22ProtocolTraitObjectDecision: %v", err)
	}
	if report.SchemaVersion != protocolTraitObjectDecisionSchemaV1 {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, protocolTraitObjectDecisionSchemaV1)
	}
	if report.Scope != protocolTraitObjectDecisionScopeP222 {
		t.Fatalf("scope = %q, want %q", report.Scope, protocolTraitObjectDecisionScopeP222)
	}
	if report.Decision != protocolTraitObjectDecisionKeepStaticOnly {
		t.Fatalf(
			"decision = %q, want %q",
			report.Decision,
			protocolTraitObjectDecisionKeepStaticOnly,
		)
	}
	if err := ValidateP22ProtocolTraitObjectDecision(report); err != nil {
		t.Fatalf("ValidateP22ProtocolTraitObjectDecision: %v", err)
	}

	rows := map[ProtocolTraitObjectDecisionID]ProtocolTraitObjectDecisionRow{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || row.Decision == "" ||
			len(row.Evidence) == 0 ||
			len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p22ProtocolTraitObjectDecisionIDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p22AssertProtocolTraitRow(
		t,
		rows[ProtocolTraitStaticConformanceFastPath],
		[]string{
			"static conformance",
			"compareProtocolRequirement",
			"known direct IRCall",
			"Vec2.draw",
		},
	)
	p22AssertProtocolTraitRow(
		t,
		rows[ProtocolTraitStaticProtocolBoundGenerics],
		[]string{
			"protocol-bound generics",
			"monomorphization",
			"id__T_Vec2",
			"no runtime generic values",
		},
	)
	p22AssertProtocolTraitRow(
		t,
		rows[ProtocolTraitRuntimeExistentialDecision],
		[]string{
			"keep_static_conformance_only",
			"unknown type 'Drawable'",
			"runtime protocol values remain unsupported",
		},
	)
	p22AssertProtocolTraitRow(
		t,
		rows[ProtocolTraitExplicitDynamicDispatchGate],
		[]string{"dynamic dispatch must be explicit", "report-visible", "not promoted"},
	)
	p22AssertProtocolTraitRow(
		t,
		rows[ProtocolTraitSpecializationStaticAbstraction],
		[]string{
			"P17.2",
			"P21.2",
			"known direct Stack IR function symbol",
			"Machine IR contains no OpCall",
		},
	)
	p22AssertProtocolTraitRow(
		t,
		rows[ProtocolTraitWitnessTableBoundary],
		[]string{"witness tables", "not emitted", "future ABI evidence"},
	)
	p22AssertProtocolTraitRow(
		t,
		rows[ProtocolTraitTraitObjectBoundary],
		[]string{"trait objects", "not promoted", "runtime existential"},
	)
	p22AssertProtocolTraitRow(
		t,
		rows[ProtocolTraitRegistryDocsAlignment],
		[]string{
			"FeatureRegistry",
			"language.protocol-conformance-mvp",
			"language.protocol-bound-generics-static",
		},
	)

	witnesses := map[string]ProtocolTraitObjectWitness{}
	for _, witness := range report.Witnesses {
		witnesses[witness.ID] = witness
	}
	static := witnesses[protocolTraitStaticConformanceWitnessID]
	if static.ID == "" {
		t.Fatalf("missing static conformance witness: %#v", report.Witnesses)
	}
	if static.ProtocolCount != 1 || static.ImplCount != 1 || !static.HasStaticMethodSig ||
		static.DirectCallTarget != "Vec2.draw" ||
		!static.LoweredDirectCall {
		t.Fatalf(
			"static conformance witness = %#v, want one protocol/impl and direct Vec2.draw IRCall",
			static,
		)
	}
	generic := witnesses[protocolTraitProtocolBoundGenericWitnessID]
	if generic.ID == "" {
		t.Fatalf("missing protocol-bound generic witness: %#v", report.Witnesses)
	}
	if generic.MonomorphizedSig != "id__T_Vec2" || !generic.MonomorphizedSigConcrete ||
		!generic.LoweredDirectCall {
		t.Fatalf(
			"protocol-bound generic witness = %#v, want concrete id__T_Vec2 direct call",
			generic,
		)
	}
	boundary := witnesses[protocolTraitRuntimeBoundaryWitnessID]
	if boundary.ID == "" {
		t.Fatalf("missing runtime boundary witness: %#v", report.Witnesses)
	}
	if !strings.Contains(boundary.RuntimeProtocolValueDiagnostic, "unknown type 'Drawable'") ||
		!strings.Contains(boundary.GenericRequirementCallDiagnostic, "not supported in this MVP") {
		t.Fatalf(
			"runtime boundary witness = %#v, want runtime value and generic-bound call diagnostics",
			boundary,
		)
	}
	specialization := witnesses[protocolTraitSpecializationWitnessID]
	if specialization.ID == "" {
		t.Fatalf("missing specialization witness: %#v", report.Witnesses)
	}
	if specialization.InliningSchema != "tetra.optimizer.inlining_specialization.v1" ||
		specialization.MachineSchema != "tetra.optimizer.specialization_machine_code.v1" ||
		!specialization.KnownDirectSymbolEvidence ||
		!specialization.SpecializationNoDynamicDispatch ||
		!specialization.MachineNoOpCall {
		t.Fatalf(
			"specialization witness = %#v, want P17/P21 known-direct no-dynamic-dispatch evidence",
			specialization,
		)
	}

	for _, nonClaim := range []string{
		"runtime protocol values are not promoted",
		"trait objects are not promoted",
		"witness tables are not promoted",
		"dynamic dispatch is not promoted",
		"conformance-table lookup is not promoted",
		"runtime existential ABI is not designed in this slice",
		"broad protocol specialization is not claimed",
		"performance is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
	} {
		if !p22ProtocolTraitHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP22ProtocolTraitObjectDecisionRejectsFakeClaimsAndDrift(t *testing.T) {
	base, err := BuildP22ProtocolTraitObjectDecision()
	if err != nil {
		t.Fatalf("BuildP22ProtocolTraitObjectDecision: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*ProtocolTraitObjectDecisionReport)
		want   string
	}{
		{
			name: "wrong decision",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.Decision = "promote_runtime_existentials"
			},
			want: "keep_static_conformance_only",
		},
		{
			name: "missing row",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness reference",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "bad static witness",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.Witnesses[0].LoweredDirectCall = false
			},
			want: "static conformance witness",
		},
		{
			name: "bad runtime boundary witness",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.Witnesses[2].RuntimeProtocolValueDiagnostic = ""
			},
			want: "runtime boundary witness",
		},
		{
			name: "bad specialization witness",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.Witnesses[3].MachineNoOpCall = false
			},
			want: "specialization witness",
		},
		{
			name: "runtime existential claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.RuntimeExistentialsPromoted = true
			},
			want: "runtime existential",
		},
		{
			name: "trait object claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.TraitObjectsPromoted = true
			},
			want: "trait object",
		},
		{
			name: "witness table claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.WitnessTablesPromoted = true
			},
			want: "witness table",
		},
		{
			name: "dynamic dispatch claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.DynamicDispatchPromoted = true
			},
			want: "dynamic dispatch",
		},
		{
			name: "conformance table claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.ConformanceTableLookupPromoted = true
			},
			want: "conformance-table",
		},
		{
			name: "runtime protocol value claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.RuntimeProtocolValuesPromoted = true
			},
			want: "runtime protocol value",
		},
		{
			name: "broad specialization claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.BroadSpecializationClaimed = true
			},
			want: "broad specialization",
		},
		{
			name: "performance claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
		{
			name: "runtime behavior claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.SafeSemanticsChanged = true
			},
			want: "safe-program semantics",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := cloneProtocolTraitObjectDecision(base)
			tc.mutate(&report)
			err := ValidateP22ProtocolTraitObjectDecision(report)
			if err == nil {
				t.Fatalf("ValidateP22ProtocolTraitObjectDecision accepted fake report: %#v", report)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func p22AssertProtocolTraitRow(t *testing.T, row ProtocolTraitObjectDecisionRow, wants []string) {
	t.Helper()
	combined := row.Name + " " + row.Status + " " + row.Decision + " " + strings.Join(
		row.Evidence,
		" ",
	) + " " + strings.Join(
		row.Tests,
		" ",
	) + " " + strings.Join(
		row.Boundaries,
		" ",
	)
	for _, want := range wants {
		if !strings.Contains(combined, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}

func cloneProtocolTraitObjectDecision(
	report ProtocolTraitObjectDecisionReport,
) ProtocolTraitObjectDecisionReport {
	report.Rows = append([]ProtocolTraitObjectDecisionRow{}, report.Rows...)
	for i := range report.Rows {
		report.Rows[i].Evidence = append([]string{}, report.Rows[i].Evidence...)
		report.Rows[i].Tests = append([]string{}, report.Rows[i].Tests...)
		report.Rows[i].Boundaries = append([]string{}, report.Rows[i].Boundaries...)
		report.Rows[i].WitnessIDs = append([]string{}, report.Rows[i].WitnessIDs...)
	}
	report.Witnesses = append([]ProtocolTraitObjectWitness{}, report.Witnesses...)
	report.NonClaims = append([]string{}, report.NonClaims...)
	return report
}

func p22ProtocolTraitHasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// ---- reports_internal_test.go ----

func TestValidateAllocationPlanReportRejectsMismatch(t *testing.T) {
	plan := &allocplan.Plan{
		Totals: allocplan.Totals{Stack: 1},
		Functions: []allocplan.FunctionPlan{{
			Name: "main",
			Allocations: []allocplan.Allocation{{
				ID:                    "xs",
				SiteID:                "allocsite:main:xs:line_1_1",
				ValueID:               "alloc_intent:xs",
				Builtin:               "core.make_u8",
				ElementType:           "u8",
				ElementSize:           1,
				LengthExpr:            "4",
				LengthStatus:          allocplan.LengthStatusNormal,
				ZeroGuardStatus:       "valid_empty_no_allocator",
				NegativeGuardStatus:   "reject_before_allocation",
				OverflowGuardStatus:   "reject_before_allocation",
				ByteSize:              4,
				Escape:                allocplan.EscapeNoEscape,
				Storage:               allocplan.StorageStack,
				PlannedStorage:        allocplan.StorageStack,
				ActualLoweringStorage: allocplan.StorageStack,
				ValidationStatus:      "validated_no_escape",
				LoweringStatus:        "stack_lowering",
				Reason:                "test",
			}},
		}},
	}
	report := wrapAllocationPlanReport(plan, "linux-x64")
	report.Totals.Stack = 0

	err := validateAllocationPlanReport(plan, report)
	if err == nil || !strings.Contains(err.Error(), "allocation report mismatch") {
		t.Fatalf("validateAllocationPlanReport error = %v, want mismatch rejection", err)
	}
}

func TestValidateMemoryReportForEmissionRejectsAlteredProjection(t *testing.T) {
	graph := emissionProjectionGraph(t)
	report := memoryfacts.BuildReportFromGraph(graph)
	report.Rows[0].CostClass = memoryfacts.CostConservativeFallback

	err := validateMemoryReportForEmission(graph, report)
	if err == nil || !strings.Contains(err.Error(), "validate memory report projection") ||
		!strings.Contains(err.Error(), "cost_class") {
		t.Fatalf(
			"validateMemoryReportForEmission error = %v, want projection cost_class rejection",
			err,
		)
	}
}

func TestValidateMemoryReportForEmissionRejectsDroppedProjectedFact(t *testing.T) {
	graph := emissionProjectionGraph(t)
	report := memoryfacts.BuildReportFromGraph(graph)
	report.Rows = report.Rows[:1]

	err := validateMemoryReportForEmission(graph, report)
	if err == nil || !strings.Contains(err.Error(), "missing report row") {
		t.Fatalf(
			"validateMemoryReportForEmission error = %v, want missing projected fact rejection",
			err,
		)
	}
}

func emissionProjectionGraph(t *testing.T) *memoryfacts.Graph {
	t.Helper()

	graph := memoryfacts.NewGraph("program")
	if _, err := graph.AddFact(memoryfacts.Fact{
		ID:               "fact:emission:borrow",
		FunctionID:       "main",
		SiteID:           "site:borrow",
		SourceStage:      memoryfacts.StagePLIR,
		Claim:            "borrowed view",
		ProvenanceClass:  memoryfacts.ProvenanceSafeBorrowed,
		BorrowState:      memoryfacts.BorrowImmutable,
		UnsafeClass:      memoryfacts.UnsafeSafe,
		ValidationState:  memoryfacts.ValidationPass,
		ValidatorName:    "test-emission",
		CostClass:        memoryfacts.CostDynamicCheckRequired,
		NormalBuildCheck: true,
	}); err != nil {
		t.Fatalf("add borrow fact: %v", err)
	}
	if _, err := graph.AddFact(memoryfacts.Fact{
		ID:              "fact:emission:borrow-sibling",
		FunctionID:      "main",
		SiteID:          "site:borrow-sibling",
		SourceStage:     memoryfacts.StagePLIR,
		Claim:           "borrowed sibling view",
		ProvenanceClass: memoryfacts.ProvenanceSafeBorrowed,
		BorrowState:     memoryfacts.BorrowImmutable,
		UnsafeClass:     memoryfacts.UnsafeSafe,
		CostClass:       memoryfacts.CostInstrumentationOnly,
	}); err != nil {
		t.Fatalf("add sibling borrow fact: %v", err)
	}
	return graph
}

func TestBuildLayoutReportRecordsP21DefaultReprCAndExportDecisions(t *testing.T) {
	checked := &semantics.CheckedProgram{
		Structs: []semantics.CheckedStruct{
			{
				Name:   "main.Packet",
				Module: "main",
				Decl: &frontend.StructDecl{
					Name: "Packet",
					Repr: frontend.StructReprDefault,
					Fields: []frontend.FieldDecl{
						{Name: "tag"},
						{Name: "code"},
					},
				},
			},
			{
				Name:   "main.Header",
				Module: "main",
				Decl: &frontend.StructDecl{
					Name: "Header",
					Repr: frontend.StructReprC,
					Fields: []frontend.FieldDecl{
						{Name: "tag"},
						{Name: "code"},
					},
				},
			},
		},
		Types: map[string]*semantics.TypeInfo{
			"main.Packet": {
				Name:      "main.Packet",
				Kind:      semantics.TypeStruct,
				Repr:      frontend.StructReprDefault,
				SlotCount: 2,
				Fields: []semantics.FieldInfo{
					{Name: "tag", TypeName: "c_int", Offset: 0, SlotCount: 1},
					{Name: "code", TypeName: "c_int", Offset: 1, SlotCount: 1},
				},
			},
			"main.Header": {
				Name:      "main.Header",
				Kind:      semantics.TypeStruct,
				Repr:      frontend.StructReprC,
				SlotCount: 2,
				Fields: []semantics.FieldInfo{
					{Name: "tag", TypeName: "c_int", Offset: 0, SlotCount: 1},
					{Name: "code", TypeName: "c_int", Offset: 1, SlotCount: 1},
				},
			},
		},
		Funcs: []semantics.CheckedFunc{
			{
				Name:   "main.ffi_header",
				Module: "main",
				Decl:   &frontend.FuncDecl{Name: "ffi_header", ExportName: "ffi_header_c"},
			},
		},
		FuncSigs: map[string]semantics.FuncSig{
			"main.ffi_header": {
				ParamNames: []string{"header"},
				ParamTypes: []string{"main.Header"},
				ReturnType: "c_int",
			},
		},
	}

	report := buildLayoutReport("linux-x64", checked)
	if report.SchemaVersion != 2 || report.Kind != "layout" || report.Policy != p21LayoutPolicy {
		t.Fatalf("layout report header = %+v", report.ReportEnvelope)
	}
	if report.Summary.Structs != 2 || report.Summary.DefaultCompilerOwned != 1 ||
		report.Summary.ReprCABILocked != 1 ||
		report.Summary.ExportedPublicABI != 1 {
		t.Fatalf("layout summary = %+v, want default/reprC/export counts", report.Summary)
	}
	byName := map[string]layoutDecisionRow{}
	for _, row := range report.Decisions {
		byName[row.Type] = row
	}
	packet := byName["main.Packet"]
	if packet.Decision != "compiler_owned_default" || packet.ABILocked ||
		packet.PublicABI != "not_public_abi" {
		t.Fatalf("default packet row = %+v", packet)
	}
	for _, want := range []string{
		"field_reordering",
		"padding_removal",
		"hot_cold_splitting",
		"scalar_replacement",
		"aos_to_soa",
	} {
		if !containsString(packet.AllowedTransforms, want) {
			t.Fatalf(
				"default packet allowed transforms = %+v, want %q",
				packet.AllowedTransforms,
				want,
			)
		}
	}
	header := byName["main.Header"]
	if header.Decision != "abi_locked_repr_c" || !header.ABILocked ||
		header.PublicABI != "exported_ffi_explicit_repr_c" {
		t.Fatalf("repr(C) header row = %+v", header)
	}
	if len(header.AllowedTransforms) != 0 ||
		!containsString(header.DeniedTransforms, "field_reordering") {
		t.Fatalf(
			"repr(C) transforms = allowed %+v denied %+v",
			header.AllowedTransforms,
			header.DeniedTransforms,
		)
	}
	if err := ValidateLayoutReport(report); err != nil {
		t.Fatalf("ValidateLayoutReport: %v", err)
	}
}

func TestValidateLayoutReportRejectsFakeP21Decisions(t *testing.T) {
	report := layoutReport{
		ReportEnvelope: reportEnvelope{SchemaVersion: 2, Kind: "layout", Target: "linux-x64"},
		Policy:         p21LayoutPolicy,
		Summary: layoutSummary{
			Structs:              2,
			DefaultCompilerOwned: 1,
			ReprCABILocked:       1,
			ExportedPublicABI:    1,
		},
		Decisions: []layoutDecisionRow{
			{
				Type:      "main.Packet",
				Repr:      frontend.StructReprDefault,
				Decision:  "compiler_owned_default",
				PublicABI: "not_public_abi",
				AllowedTransforms: []string{
					"field_reordering",
					"padding_removal",
					"hot_cold_splitting",
					"scalar_replacement",
					"aos_to_soa",
				},
				Reason: "default struct layout is compiler-owned",
			},
			{
				Type:      "main.Header",
				Repr:      frontend.StructReprC,
				ABILocked: true,
				Decision:  "abi_locked_repr_c",
				PublicABI: "exported_ffi_explicit_repr_c",
				DeniedTransforms: []string{
					"field_reordering",
					"padding_removal",
					"hot_cold_splitting",
					"scalar_replacement",
					"aos_to_soa",
				},
				Reason: "repr(C) locks layout",
			},
		},
	}
	report.Decisions[1].AllowedTransforms = []string{"field_reordering"}
	err := ValidateLayoutReport(report)
	if err == nil || !strings.Contains(err.Error(), "repr(C)") {
		t.Fatalf("ValidateLayoutReport accepted repr(C) layout freedom: %v", err)
	}

	report = buildMinimalValidLayoutReportForTest()
	report.Decisions[0].ABILocked = true
	err = ValidateLayoutReport(report)
	if err == nil || !strings.Contains(err.Error(), "default") {
		t.Fatalf("ValidateLayoutReport accepted default ABI lock: %v", err)
	}

	report = buildMinimalValidLayoutReportForTest()
	report.Decisions[0].PublicABI = "exported_ffi_missing_explicit_repr"
	err = ValidateLayoutReport(report)
	if err == nil || !strings.Contains(err.Error(), "explicit repr(C)") {
		t.Fatalf("ValidateLayoutReport accepted exported default-layout ABI row: %v", err)
	}

	report = buildMinimalValidLayoutReportForTest()
	report.Decisions[0].PublicABI = "exported_ffi_explicit_repr_c"
	err = ValidateLayoutReport(report)
	if err == nil || !strings.Contains(err.Error(), "without repr(C)") {
		t.Fatalf("ValidateLayoutReport accepted spoofed explicit repr(C) ABI row: %v", err)
	}
}

func buildMinimalValidLayoutReportForTest() layoutReport {
	return layoutReport{
		ReportEnvelope: reportEnvelope{SchemaVersion: 2, Kind: "layout", Target: "linux-x64"},
		Policy:         p21LayoutPolicy,
		Summary: layoutSummary{
			Structs:              2,
			DefaultCompilerOwned: 1,
			ReprCABILocked:       1,
			ExportedPublicABI:    1,
		},
		Decisions: []layoutDecisionRow{
			{
				Type:      "main.Packet",
				Repr:      frontend.StructReprDefault,
				Decision:  "compiler_owned_default",
				PublicABI: "not_public_abi",
				AllowedTransforms: []string{
					"field_reordering",
					"padding_removal",
					"hot_cold_splitting",
					"scalar_replacement",
					"aos_to_soa",
				},
				Reason: "default struct layout is compiler-owned",
			},
			{
				Type:      "main.Header",
				Repr:      frontend.StructReprC,
				ABILocked: true,
				Decision:  "abi_locked_repr_c",
				PublicABI: "exported_ffi_explicit_repr_c",
				DeniedTransforms: []string{
					"field_reordering",
					"padding_removal",
					"hot_cold_splitting",
					"scalar_replacement",
					"aos_to_soa",
				},
				Reason: "repr(C) locks layout",
			},
		},
	}
}

func TestWrapAllocationPlanReportV2IncludesRuntimeSummary(t *testing.T) {
	plan := &allocplan.Plan{
		Totals: allocplan.Totals{Stack: 1, ExplicitIsland: 1},
		Functions: []allocplan.FunctionPlan{{
			Name: "main",
			Allocations: []allocplan.Allocation{
				{
					ID:                    "xs",
					SiteID:                "allocsite:main:xs:line_1_1",
					ValueID:               "alloc_intent:xs",
					Builtin:               "core.make_u8",
					ElementType:           "u8",
					ElementSize:           1,
					LengthExpr:            "32",
					LengthStatus:          allocplan.LengthStatusNormal,
					ByteSize:              32,
					Escape:                allocplan.EscapeNoEscape,
					Storage:               allocplan.StorageStack,
					PlannedStorage:        allocplan.StorageStack,
					ActualLoweringStorage: allocplan.StorageStack,
					ValidationStatus:      "validated_no_escape",
					LoweringStatus:        "stack_lowering",
					RuntimePath:           "stack_frame",
					BytesRequested:        32,
					BytesReserved:         32,
					Reason:                "test stack",
				},
				{
					ID:                    "ys",
					SiteID:                "allocsite:main:ys:line_2_1",
					ValueID:               "alloc_intent:ys",
					Builtin:               "core.island_make_u8",
					ElementType:           "u8",
					ElementSize:           1,
					LengthExpr:            "17",
					LengthStatus:          allocplan.LengthStatusNormal,
					ByteSize:              17,
					Escape:                allocplan.EscapeNoEscape,
					Storage:               allocplan.StorageExplicitIsland,
					PlannedStorage:        allocplan.StorageExplicitIsland,
					ActualLoweringStorage: allocplan.StorageExplicitIsland,
					ValidationStatus:      "validated_explicit_island_scope",
					LoweringStatus:        "explicit_island_lowering",
					RuntimePath:           "explicit_island",
					BytesRequested:        17,
					BytesReserved:         32,
					RegionID:              "island:isl",
					Lifetime:              "island:isl:scope",
					Reason:                "test island",
				},
			},
		}},
	}

	report := wrapAllocationPlanReport(plan, "linux-x64")
	if report.SchemaVersion != 2 {
		t.Fatalf("allocation report schema_version = %d, want 2", report.SchemaVersion)
	}
	if report.Summary.AllocationCount != 2 {
		t.Fatalf("allocation_count = %d, want 2", report.Summary.AllocationCount)
	}
	if report.Summary.StorageClasses[string(allocplan.StorageStack)] != 1 ||
		report.Summary.StorageClasses[string(allocplan.StorageExplicitIsland)] != 1 {
		t.Fatalf(
			"storage class summary = %+v, want Stack and ExplicitIsland counts",
			report.Summary.StorageClasses,
		)
	}
	if report.Summary.ActualLoweringStorageClasses[string(allocplan.StorageStack)] != 1 ||
		report.Summary.ActualLoweringStorageClasses[string(allocplan.StorageExplicitIsland)] != 1 {
		t.Fatalf(
			"actual storage summary = %+v, want Stack and ExplicitIsland counts",
			report.Summary.ActualLoweringStorageClasses,
		)
	}
	if report.Summary.RuntimePaths["stack_frame"] != 1 ||
		report.Summary.RuntimePaths["explicit_island"] != 1 {
		t.Fatalf(
			"runtime path summary = %+v, want stack_frame and explicit_island counts",
			report.Summary.RuntimePaths,
		)
	}
	if report.Summary.BytesRequested != 49 || report.Summary.BytesReserved != 64 {
		t.Fatalf(
			"byte summary = requested %d reserved %d, want 49/64",
			report.Summary.BytesRequested,
			report.Summary.BytesReserved,
		)
	}
	if len(report.Summary.Regions) != 1 || report.Summary.Regions[0].RegionID != "island:isl" ||
		report.Summary.Regions[0].Lifetime != "island:isl:scope" {
		t.Fatalf("regions summary = %+v, want island region", report.Summary.Regions)
	}
	if err := validateAllocationPlanReport(plan, report); err != nil {
		t.Fatalf("validateAllocationPlanReport: %v", err)
	}
}

func TestWrapAllocationPlanReportV2IncludesFunctionTempRegionSummary(t *testing.T) {
	plan := &allocplan.Plan{
		Totals: allocplan.Totals{FunctionTempRegion: 1},
		Functions: []allocplan.FunctionPlan{{
			Name: "local_copy",
			Allocations: []allocplan.Allocation{{
				ID:                    "copied",
				SiteID:                "allocsite:local_copy:copied:line_4_5",
				ValueID:               "alloc_intent:copied",
				Builtin:               "core.slice_copy_u8",
				ElementType:           "u8",
				ElementSize:           1,
				LengthExpr:            "n",
				LengthStatus:          allocplan.LengthStatusNormal,
				ByteSize:              0,
				Escape:                allocplan.EscapeNoEscape,
				Storage:               allocplan.StorageFunctionTempRegion,
				PlannedStorage:        allocplan.StorageFunctionTempRegion,
				ActualLoweringStorage: allocplan.StorageFunctionTempRegion,
				ValidationStatus:      "validated_function_temp_region_scope",
				LoweringStatus:        "function_temp_region_lowering",
				RuntimePath:           "region",
				AllocatorClass:        "function_temp_region",
				BytesRequested:        0,
				BytesReserved:         0,
				RegionID:              "region:local_copy:temp",
				Lifetime:              "function:local_copy",
				DebugMode:             "region_reset_when_enabled",
				Reason:                "function-local temporary copy lowers through region enter/reset IR",
			}},
		}},
	}

	report := wrapAllocationPlanReport(plan, "linux-x64")
	if report.Summary.StorageClasses["FunctionTempRegion"] != 1 ||
		report.Summary.ActualLoweringStorageClasses["FunctionTempRegion"] != 1 ||
		report.Summary.RuntimePaths["region"] != 1 {
		t.Fatalf("function-temp region summary missing region counts: %+v", report.Summary)
	}
	if len(report.Summary.Regions) != 1 {
		t.Fatalf("regions summary = %+v, want one function-temp region", report.Summary.Regions)
	}
	region := report.Summary.Regions[0]
	if region.RegionID != "region:local_copy:temp" ||
		region.Lifetime != "function:local_copy" ||
		region.StorageClass != "FunctionTempRegion" ||
		region.RuntimePath != "region" ||
		region.AllocationCount != 1 {
		t.Fatalf("function-temp region summary row = %+v", region)
	}
	if err := validateAllocationPlanReport(plan, report); err != nil {
		t.Fatalf("validateAllocationPlanReport: %v", err)
	}
}

func TestValidateAllocationPlanReportRejectsRuntimeSummaryMismatch(t *testing.T) {
	plan := &allocplan.Plan{
		Totals: allocplan.Totals{Heap: 1},
		Functions: []allocplan.FunctionPlan{{
			Name: "main",
			Allocations: []allocplan.Allocation{{
				ID:                    "xs",
				SiteID:                "allocsite:main:xs:line_1_1",
				ValueID:               "alloc_intent:xs",
				Builtin:               "core.make_u8",
				ElementType:           "u8",
				ElementSize:           1,
				LengthExpr:            "5000",
				LengthStatus:          allocplan.LengthStatusNormal,
				ByteSize:              5000,
				Escape:                allocplan.EscapeReturn,
				Storage:               allocplan.StorageHeap,
				PlannedStorage:        allocplan.StorageHeap,
				ActualLoweringStorage: allocplan.StorageHeap,
				ValidationStatus:      "validated_heap_fallback",
				LoweringStatus:        "large_mmap_runtime",
				RuntimePath:           "large_mmap",
				BytesRequested:        5000,
				BytesReserved:         5000,
				Reason:                "test heap",
			}},
		}},
	}
	report := wrapAllocationPlanReport(plan, "linux-x64")
	report.Summary.RuntimePaths["large_mmap"] = 0

	err := validateAllocationPlanReport(plan, report)
	if err == nil || !strings.Contains(err.Error(), "allocation report mismatch") {
		t.Fatalf("validateAllocationPlanReport error = %v, want summary mismatch rejection", err)
	}
}

func TestBackendReportListsRegisterAndStackFallbackPaths(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "add",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "checked_get",
			ParamSlots:  3,
			LocalSlots:  3,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRIndexLoadI32},
				{Kind: ir.IRReturn},
			},
		},
	}})
	got := map[string]string{}
	for _, row := range report.Functions {
		got[row.Function] = row.BackendPath
	}
	if got["add"] != "register" {
		t.Fatalf("add backend_path = %q, want register (rows=%+v)", got["add"], report.Functions)
	}
	if got["checked_get"] != "stack" {
		t.Fatalf(
			"checked_get backend_path = %q, want stack fallback (rows=%+v)",
			got["checked_get"],
			report.Functions,
		)
	}
}

func TestBackendReportPromotesFunctionCallsBenchmarkMainCallLoop(t *testing.T) {
	src := `module p25.function_calls

func mix(a: Int, b: Int) -> Int:
    return (a * 3 + b) % 97

func main() -> Int:
    var i: Int = 0
    var total: Int = 0
    while i < 200000:
        total = total + mix(i, total)
        i = i + 1
    if total >= 0:
        return 0
    return 1
`
	file, err := ParseFile([]byte(src), "function_calls.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "p25.function_calls",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"p25.function_calls": file},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	prog, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	report := buildBackendReport("linux-x64", prog)
	rows := backendRowsByFunction(report.Functions)
	assertBackendCoverageRow(
		t,
		rows["p25.function_calls.mix"],
		"register",
		"register_path",
		"eligible_machine_ir_subset",
	)
	assertBackendCoverageRow(
		t,
		rows["p25.function_calls.main"],
		"register",
		"register_path",
		"eligible_machine_ir_subset",
	)
	if rows["p25.function_calls.main"].Detail != "machine-ir-call-loop" {
		t.Fatalf(
			"function_calls main detail = %q, want machine-ir-call-loop (rows=%+v)",
			rows["p25.function_calls.main"].Detail,
			report.Functions,
		)
	}
}

func TestBackendReportPromotesCompileTimeBenchmarkMainEqualityTailCallLoop(t *testing.T) {
	src := `module p25.compile_time

func f0(x: Int) -> Int:
    return x + 1

func f1(x: Int) -> Int:
    return f0(x) * 3

func f2(x: Int) -> Int:
    return f1(x) + f0(x)

func main() -> Int:
    var i: Int = 0
    var total: Int = 0
    while i < 200000:
        total = total + f2(i)
        i = i + 1
    if total == 0:
        return 1
    return 0
`
	file, err := ParseFile([]byte(src), "compile_time.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "p25.compile_time",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"p25.compile_time": file},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	prog, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	report := buildBackendReport("linux-x64", prog)
	rows := backendRowsByFunction(report.Functions)
	assertBackendCoverageRow(
		t,
		rows["p25.compile_time.f0"],
		"register",
		"register_path",
		"eligible_machine_ir_subset",
	)
	assertBackendCoverageRow(
		t,
		rows["p25.compile_time.f1"],
		"register",
		"register_path",
		"eligible_machine_ir_subset",
	)
	assertBackendCoverageRow(
		t,
		rows["p25.compile_time.f2"],
		"register",
		"register_path",
		"eligible_machine_ir_subset",
	)
	assertBackendCoverageRow(
		t,
		rows["p25.compile_time.main"],
		"register",
		"register_path",
		"eligible_machine_ir_subset",
	)
	if rows["p25.compile_time.main"].Detail != "machine-ir-call-loop" {
		t.Fatalf(
			"compile_time main detail = %q, want machine-ir-call-loop (rows=%+v)",
			rows["p25.compile_time.main"].Detail,
			report.Functions,
		)
	}
	machineRows := map[string]machineBackendFunctionReport{}
	for _, row := range report.MachineFunctions {
		machineRows[row.Function] = row
	}
	mainRow, ok := machineRows["p25.compile_time.main"]
	if !ok {
		t.Fatalf("machine report missing for p25.compile_time.main: %+v", report.MachineFunctions)
	}
	if mainRow.Path != "machine-ir-call-loop" || !mainRow.SSAVerified ||
		mainRow.SSAPath != "value-ssa-v1" {
		t.Fatalf("compile_time main machine row = %+v, want verified machine-ir-call-loop", mainRow)
	}
	if !containsReportString(mainRow.InstructionSelection, "call") {
		t.Fatalf(
			"compile_time main instruction selection = %+v, want call evidence",
			mainRow.InstructionSelection,
		)
	}
	if mainRow.Validation.MachineVerifier != "pass" ||
		mainRow.Validation.AllocationVerifier != "pass" ||
		mainRow.Validation.CallClobbers != "validated" ||
		mainRow.Validation.StackChurnOps != 0 {
		t.Fatalf(
			("compile_time main validation = %+v, want " +
				"verifier/allocation/clobber pass and no push/pop churn"),
			mainRow.Validation,
		)
	}
}

func TestBackendReportPromotesRecursionBenchmarkFibAndMain(t *testing.T) {
	src := `module p25.recursion

func fib(n: Int) -> Int:
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)

func main() -> Int:
    var i: Int = 0
    var total: Int = 0
    while i < 40:
        total = total + fib(10)
        i = i + 1
    if total == 2200:
        return 0
    return 1
`
	file, err := ParseFile([]byte(src), "recursion.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "p25.recursion",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"p25.recursion": file},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	prog, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	report := buildBackendReport("linux-x64", prog)
	rows := backendRowsByFunction(report.Functions)
	assertBackendCoverageRow(
		t,
		rows["p25.recursion.fib"],
		"register",
		"register_path",
		"eligible_machine_ir_subset",
	)
	assertBackendCoverageRow(
		t,
		rows["p25.recursion.main"],
		"register",
		"register_path",
		"eligible_machine_ir_subset",
	)
	if rows["p25.recursion.fib"].Detail != "machine-ir-recursive-fib" {
		t.Fatalf(
			"recursion fib detail = %q, want machine-ir-recursive-fib (rows=%+v)",
			rows["p25.recursion.fib"].Detail,
			report.Functions,
		)
	}
	if rows["p25.recursion.main"].Detail != "machine-ir-recursion-main-loop" {
		t.Fatalf(
			"recursion main detail = %q, want machine-ir-recursion-main-loop (rows=%+v)",
			rows["p25.recursion.main"].Detail,
			report.Functions,
		)
	}
	machineRows := map[string]machineBackendFunctionReport{}
	for _, row := range report.MachineFunctions {
		machineRows[row.Function] = row
	}
	for name, wantPath := range map[string]string{
		"p25.recursion.fib":  "machine-ir-recursive-fib",
		"p25.recursion.main": "machine-ir-recursion-main-loop",
	} {
		row, ok := machineRows[name]
		if !ok {
			t.Fatalf("machine report missing for %s: %+v", name, report.MachineFunctions)
		}
		if row.Path != wantPath || !row.SSAVerified || row.SSAPath != "value-ssa-v1" {
			t.Fatalf("machine row for %s = %+v, want verified %s", name, row, wantPath)
		}
		if !containsReportString(row.InstructionSelection, "call") {
			t.Fatalf(
				"machine row for %s instruction selection = %+v, want call evidence",
				name,
				row.InstructionSelection,
			)
		}
		if row.Validation.MachineVerifier != "pass" ||
			row.Validation.AllocationVerifier != "pass" ||
			row.Validation.CallClobbers != "validated" ||
			row.Validation.StackChurnOps != 0 {
			t.Fatalf(
				("machine row for %s validation = %+v, want " +
					"verifier/allocation/clobber pass and no push/pop churn"),
				name,
				row.Validation,
			)
		}
	}
}

func TestBackendReportPromotesIntegerLoopsBenchmarkMainModuloLoop(t *testing.T) {
	src := `module p25.integer_loops

func main() -> Int:
    var i: Int = 0
    var total: Int = 0
    while i < 200000:
        total = total + (i % 7)
        i = i + 1
    if total >= 0:
        return 0
    return 1
`
	file, err := ParseFile([]byte(src), "integer_loops.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "p25.integer_loops",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"p25.integer_loops": file},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	prog, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	report := buildBackendReport("linux-x64", prog)
	rows := backendRowsByFunction(report.Functions)
	mainRow := rows["p25.integer_loops.main"]
	assertBackendCoverageRow(t, mainRow, "register", "register_path", "eligible_machine_ir_subset")
	if mainRow.Detail != "machine-ir-const-modulo-loop" {
		t.Fatalf(
			"integer_loops main detail = %q, want machine-ir-const-modulo-loop (rows=%+v)",
			mainRow.Detail,
			report.Functions,
		)
	}
	for _, machineRow := range report.MachineFunctions {
		if machineRow.Function != "p25.integer_loops.main" {
			continue
		}
		if machineRow.Path != "machine-ir-const-modulo-loop" {
			t.Fatalf(
				"integer_loops machine path = %q, want machine-ir-const-modulo-loop",
				machineRow.Path,
			)
		}
		if !machineRow.SSAVerified || machineRow.SSAPath != "value-ssa-v1" {
			t.Fatalf(
				"integer_loops SSA gate = verified:%v path:%q, want value-ssa-v1 verified",
				machineRow.SSAVerified,
				machineRow.SSAPath,
			)
		}
		if !containsReportString(machineRow.InstructionSelection, "mod") {
			t.Fatalf(
				"integer_loops instruction selection = %+v, want mod evidence",
				machineRow.InstructionSelection,
			)
		}
		if machineRow.Validation.StackChurnOps != 0 ||
			machineRow.Validation.MachineVerifier != "pass" ||
			machineRow.Validation.AllocationVerifier != "pass" {
			t.Fatalf(
				"integer_loops machine validation = %+v, want verifier pass and no push/pop stack churn",
				machineRow.Validation,
			)
		}
		return
	}
	t.Fatalf("integer_loops machine report missing from %+v", report.MachineFunctions)
}

func TestBackendReportPromotesBoundsCheckLoopsBenchmarkMain(t *testing.T) {
	src := `module p25.bounds_check_loops

func main() -> Int
uses alloc, mem:
    let n: Int = 4096
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    i = 0
    while i < 200000:
        let idx: Int = (i * 17) % n
        total = total + xs[idx]
        i = i + 1
    if total >= 0:
        return 0
    return 1
`
	file, err := ParseFile([]byte(src), "bounds_check_loops.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "p25.bounds_check_loops",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"p25.bounds_check_loops": file},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	prog, err := lowerpkg.LowerWithOptions(checked, lowerpkg.Options{StackAllocationLowering: true})
	if err != nil {
		t.Fatalf("LowerWithOptions: %v", err)
	}

	report := buildBackendReport("linux-x64", prog)
	rows := backendRowsByFunction(report.Functions)
	mainRow := rows["p25.bounds_check_loops.main"]
	assertBackendCoverageRow(t, mainRow, "register", "register_path", "eligible_machine_ir_subset")
	if mainRow.Detail != "machine-ir-bounds-check-loops" {
		t.Fatalf(
			"bounds_check_loops main detail = %q, want machine-ir-bounds-check-loops (rows=%+v)",
			mainRow.Detail,
			report.Functions,
		)
	}
	for _, machineRow := range report.MachineFunctions {
		if machineRow.Function != "p25.bounds_check_loops.main" {
			continue
		}
		if machineRow.Path != "machine-ir-bounds-check-loops" {
			t.Fatalf(
				"bounds_check_loops machine path = %q, want machine-ir-bounds-check-loops",
				machineRow.Path,
			)
		}
		if !machineRow.SSAVerified || machineRow.SSAPath != "value-ssa-v1" {
			t.Fatalf(
				"bounds_check_loops SSA gate = verified:%v path:%q, want value-ssa-v1 verified",
				machineRow.SSAVerified,
				machineRow.SSAPath,
			)
		}
		for _, want := range []string{"index_store", "index_load", "mod"} {
			if !containsReportString(machineRow.InstructionSelection, want) {
				t.Fatalf(
					"bounds_check_loops instruction selection = %+v, want %q evidence",
					machineRow.InstructionSelection,
					want,
				)
			}
		}
		if machineRow.Validation.StackChurnOps != 0 ||
			machineRow.Validation.MachineVerifier != "pass" ||
			machineRow.Validation.AllocationVerifier != "pass" {
			t.Fatalf(
				"bounds_check_loops machine validation = %+v, want verifier pass and no push/pop stack churn",
				machineRow.Validation,
			)
		}
		return
	}
	t.Fatalf("bounds_check_loops machine report missing from %+v", report.MachineFunctions)
}

func TestBackendCoverageAuditClassifiesFallbackReasonsAndHotness(t *testing.T) {
	prog := &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "response_cost",
			ParamSlots:  1,
			LocalSlots:  1,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "slice_return",
			ReturnSlots: 2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "aggregate_return",
			ReturnSlots: 3,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "wide_call",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRConstI32, Imm: 3},
				{Kind: ir.IRConstI32, Imm: 4},
				{Kind: ir.IRConstI32, Imm: 5},
				{Kind: ir.IRConstI32, Imm: 6},
				{Kind: ir.IRConstI32, Imm: 7},
				{Kind: ir.IRCall, Name: "callee", ArgSlots: 7, RetSlots: 1},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "alloc_runtime",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 8},
				{Kind: ir.IRAllocBytes},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "branchy",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "checked_get",
			ParamSlots:  3,
			LocalSlots:  3,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRIndexLoadI32},
				{Kind: ir.IRReturn},
			},
		},
	}}
	report := buildBackendReport("linux-x64", prog)
	if len(report.Functions) != len(prog.Funcs) {
		t.Fatalf(
			"backend coverage rows = %d, want one row per %d functions: %+v",
			len(report.Functions),
			len(prog.Funcs),
			report.Functions,
		)
	}
	rows := backendRowsByFunction(report.Functions)
	assertBackendCoverageRow(
		t,
		rows["response_cost"],
		"register",
		"register_path",
		"eligible_machine_ir_subset",
	)
	if rows["response_cost"].HotnessRank != 1 ||
		rows["response_cost"].HotnessSource != (("examples/benchmarks/systems/techempower_"+
			"plaintext_kernel.tet")+
			"ra") {
		t.Fatalf(
			"response_cost hotness = rank %d source %q, want rank 1 from TechEmpower plaintext corpus row",
			rows["response_cost"].HotnessRank,
			rows["response_cost"].HotnessSource,
		)
	}
	assertBackendCoverageRow(
		t,
		rows["slice_return"],
		"stack",
		"unsupported_slice_string_return",
		"unsupported_slice_or_string_return_uses_stack_fallback",
	)
	assertBackendCoverageRow(
		t,
		rows["aggregate_return"],
		"stack",
		"unsupported_aggregate_return",
		"unsupported_aggregate_return_uses_stack_fallback",
	)
	assertBackendCoverageRow(
		t,
		rows["wide_call"],
		"stack",
		"unsupported_call_abi",
		"unsupported_call_abi_uses_stack_fallback",
	)
	assertBackendCoverageRow(
		t,
		rows["alloc_runtime"],
		"stack",
		"unsupported_effect_runtime_call",
		"unsupported_effect_runtime_call_uses_stack_fallback",
	)
	assertBackendCoverageRow(
		t,
		rows["branchy"],
		"stack",
		"unsupported_control_flow",
		"unsupported_control_flow_uses_stack_fallback",
	)
	assertBackendCoverageRow(
		t,
		rows["checked_get"],
		"stack",
		"stack_fallback",
		"unsupported_or_unproven_subset_uses_stack_fallback",
	)
	if rows["checked_get"].HotnessRank != 0 ||
		rows["checked_get"].HotnessSource != "not_in_benchmark_corpus" {
		t.Fatalf(
			"checked_get hotness = rank %d source %q, want explicit non-corpus marker",
			rows["checked_get"].HotnessRank,
			rows["checked_get"].HotnessSource,
		)
	}
}

func TestBackendCoverageStackSliceFallsThroughToControlFlowBlocker(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind ir.IRInstrKind
	}{
		{name: "stack_u8", kind: ir.IRStackSliceU8},
		{name: "stack_u16", kind: ir.IRStackSliceU16},
		{name: "stack_i32", kind: ir.IRStackSliceI32},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{{
				Name:        tc.name,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 4},
					{Kind: tc.kind, Local: 0, ArgSlots: 4, Imm: 4, Name: "xs"},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRJmpIfZero, Label: 1},
					{Kind: ir.IRConstI32, Imm: 1},
					{Kind: ir.IRReturn},
					{Kind: ir.IRLabel, Label: 1},
					{Kind: ir.IRConstI32, Imm: 2},
					{Kind: ir.IRReturn},
				},
			}}})

			rows := backendRowsByFunction(report.Functions)
			assertBackendCoverageRow(
				t,
				rows[tc.name],
				"stack",
				"unsupported_control_flow",
				"unsupported_control_flow_uses_stack_fallback",
			)
		})
	}
}

func TestBackendCoverageIndexStoreFallsThroughToControlFlowBlocker(t *testing.T) {
	for _, tc := range []struct {
		name      string
		stackKind ir.IRInstrKind
		storeKind ir.IRInstrKind
	}{
		{name: "store_i32", stackKind: ir.IRStackSliceI32, storeKind: ir.IRIndexStoreI32},
		{name: "store_u8", stackKind: ir.IRStackSliceU8, storeKind: ir.IRIndexStoreU8},
		{name: "store_u16", stackKind: ir.IRStackSliceU16, storeKind: ir.IRIndexStoreU16},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{{
				Name:        tc.name,
				LocalSlots:  2,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 4},
					{Kind: tc.stackKind, Local: 0, ArgSlots: 4, Imm: 4, Name: "xs"},
					{Kind: ir.IRLoadLocal, Local: 0},
					{Kind: ir.IRLoadLocal, Local: 1},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRConstI32, Imm: 7},
					{Kind: tc.storeKind},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRJmpIfZero, Label: 1},
					{Kind: ir.IRConstI32, Imm: 1},
					{Kind: ir.IRReturn},
					{Kind: ir.IRLabel, Label: 1},
					{Kind: ir.IRConstI32, Imm: 2},
					{Kind: ir.IRReturn},
				},
			}}})

			rows := backendRowsByFunction(report.Functions)
			assertBackendCoverageRow(
				t,
				rows[tc.name],
				"stack",
				"unsupported_control_flow",
				"unsupported_control_flow_uses_stack_fallback",
			)
		})
	}
}

func TestBackendCoverageAllocationLoopReportsMachineIRPath(t *testing.T) {
	report := buildBackendReport(
		"linux-x64",
		&ir.IRProgram{Funcs: []ir.IRFunc{allocationLoopBenchmarkIRFuncForReport()}},
	)
	rows := backendRowsByFunction(report.Functions)
	mainRow := rows["p25.allocation.main"]
	assertBackendCoverageRow(t, mainRow, "register", "register_path", "eligible_machine_ir_subset")
	if mainRow.Detail != "machine-ir-allocation-loop" {
		t.Fatalf(
			"allocation main detail = %q, want machine-ir-allocation-loop (rows=%+v)",
			mainRow.Detail,
			report.Functions,
		)
	}
	if mainRow.ABI.MultiSlotReturnPolicy != "single_slot_register_return" {
		t.Fatalf("allocation main ABI = %+v, want single-slot register return", mainRow.ABI)
	}
	if report.Summary.RegisterPath != 1 || report.Summary.StackFallback != 0 ||
		report.Summary.Categories["register_path"] != 1 {
		t.Fatalf(
			"allocation backend summary = %+v, want one register_path and no fallback",
			report.Summary,
		)
	}
	machineRows := map[string]machineBackendFunctionReport{}
	for _, row := range report.MachineFunctions {
		machineRows[row.Function] = row
	}
	machineRow, ok := machineRows["p25.allocation.main"]
	if !ok {
		t.Fatalf("machine report missing for p25.allocation.main: %+v", report.MachineFunctions)
	}
	if machineRow.Path != "machine-ir-allocation-loop" || !machineRow.SSAVerified ||
		machineRow.SSAPath != "value-ssa-v1" {
		t.Fatalf(
			"allocation main machine row = %+v, want verified machine-ir-allocation-loop",
			machineRow,
		)
	}
	for _, want := range []string{"index_store", "index_load"} {
		if !containsReportString(machineRow.InstructionSelection, want) {
			t.Fatalf(
				"allocation main instruction selection = %+v, want %q",
				machineRow.InstructionSelection,
				want,
			)
		}
	}
	if machineRow.Validation.MachineVerifier != "pass" ||
		machineRow.Validation.AllocationVerifier != "pass" ||
		machineRow.Validation.StackChurnOps != 0 {
		t.Fatalf(
			"allocation main validation = %+v, want verifier/allocation pass and no stack churn",
			machineRow.Validation,
		)
	}
}

func TestBackendCoverageRuntimeAllocationIRStillBlocksBeforeControlFlow(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind ir.IRInstrKind
	}{
		{name: "heap_alloc", kind: ir.IRAllocBytes},
		{name: "make_slice_u8", kind: ir.IRMakeSliceU8},
		{name: "make_slice_u16", kind: ir.IRMakeSliceU16},
		{name: "make_slice_i32", kind: ir.IRMakeSliceI32},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{{
				Name:        tc.name,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 4},
					{Kind: tc.kind},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRJmpIfZero, Label: 1},
					{Kind: ir.IRConstI32, Imm: 1},
					{Kind: ir.IRReturn},
					{Kind: ir.IRLabel, Label: 1},
					{Kind: ir.IRConstI32, Imm: 2},
					{Kind: ir.IRReturn},
				},
			}}})

			rows := backendRowsByFunction(report.Functions)
			assertBackendCoverageRow(
				t,
				rows[tc.name],
				"stack",
				"unsupported_effect_runtime_call",
				"unsupported_effect_runtime_call_uses_stack_fallback",
			)
		})
	}
}

func TestBackendCoverageRegionIslandIRUsesPreciseFallbackBeforeControlFlow(t *testing.T) {
	for _, tc := range []struct {
		name    string
		kind    ir.IRInstrKind
		feature string
	}{
		{name: "island_new", kind: ir.IRIslandNew, feature: "island_allocator"},
		{name: "island_make_slice_u8", kind: ir.IRIslandMakeSliceU8, feature: "island_allocator"},
		{name: "island_make_slice_u16", kind: ir.IRIslandMakeSliceU16, feature: "island_allocator"},
		{name: "island_make_slice_i32", kind: ir.IRIslandMakeSliceI32, feature: "island_allocator"},
		{name: "island_free", kind: ir.IRIslandFree, feature: "island_allocator"},
		{name: "island_reset", kind: ir.IRIslandReset, feature: "island_allocator"},
		{name: "region_enter", kind: ir.IRRegionEnter, feature: "region_allocator"},
		{name: "region_make_slice_u8", kind: ir.IRRegionMakeSliceU8, feature: "region_allocator"},
		{name: "region_make_slice_u16", kind: ir.IRRegionMakeSliceU16, feature: "region_allocator"},
		{name: "region_make_slice_i32", kind: ir.IRRegionMakeSliceI32, feature: "region_allocator"},
		{name: "region_reset", kind: ir.IRRegionReset, feature: "region_allocator"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{{
				Name:        tc.name,
				ReturnSlots: 1,
				Instrs: []ir.IRInstr{
					{Kind: ir.IRConstI32, Imm: 4},
					{Kind: tc.kind},
					{Kind: ir.IRConstI32, Imm: 0},
					{Kind: ir.IRJmpIfZero, Label: 1},
					{Kind: ir.IRConstI32, Imm: 1},
					{Kind: ir.IRReturn},
					{Kind: ir.IRLabel, Label: 1},
					{Kind: ir.IRConstI32, Imm: 2},
					{Kind: ir.IRReturn},
				},
			}}})

			rows := backendRowsByFunction(report.Functions)
			row := rows[tc.name]
			assertBackendCoverageRow(
				t,
				row,
				"stack",
				"unsupported_island_domain_primitive",
				"unsupported_island_domain_primitive_uses_stack_fallback",
			)
			if row.Detail != fmt.Sprintf("ir_kind=%d", tc.kind) {
				t.Fatalf("%s detail = %q, want ir_kind=%d", tc.name, row.Detail, tc.kind)
			}
			assertRuntimeFeatureEvidenceMarker(
				t,
				row.RuntimeFeatureEvidenceClass,
				row.RuntimeFeatureEvidenceMethod,
			)
			assertStringSliceEqual(
				t,
				tc.name+" runtime_features_required",
				row.RuntimeFeaturesRequired,
				[]string{tc.feature},
			)
			assertStringSliceEqual(
				t,
				tc.name+" runtime_features_linked",
				row.RuntimeFeaturesLinked,
				[]string{tc.feature},
			)
			assertStringSliceEqual(
				t,
				tc.name+" runtime_features_initialized",
				row.RuntimeFeaturesInitialized,
				[]string{tc.feature},
			)
		})
	}
}

func TestBackendCoverageSummaryCountsRowsAndCategories(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "add",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "slice_return",
			ReturnSlots: 2,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRReturn},
			},
		},
	}})
	if report.Summary.FunctionCount != 2 || report.Summary.RegisterPath != 1 ||
		report.Summary.StackFallback != 1 {
		t.Fatalf(
			"backend summary = %+v, want one register row and one stack fallback row",
			report.Summary,
		)
	}
	if report.Summary.Categories["register_path"] != 1 ||
		report.Summary.Categories["unsupported_slice_string_return"] != 1 {
		t.Fatalf(
			("backend summary categories = %+v, want register_path and " +
				"unsupported_slice_string_return counts"),
			report.Summary.Categories,
		)
	}
	if report.Summary.HotnessSource != "benchmark-corpus-static-map" {
		t.Fatalf(
			"backend summary hotness source = %q, want benchmark corpus source marker",
			report.Summary.HotnessSource,
		)
	}
}

func TestBackendReportRuntimeFeaturesAreEmptyForSimpleScalar(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "add",
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}}})

	assertRuntimeFeatureEvidenceMarker(
		t,
		report.Summary.RuntimeFeatureEvidenceClass,
		report.Summary.RuntimeFeatureEvidenceMethod,
	)
	assertStringSliceEqual(
		t,
		"summary runtime_features_required",
		report.Summary.RuntimeFeaturesRequired,
		[]string{},
	)
	assertStringSliceEqual(
		t,
		"summary runtime_features_linked",
		report.Summary.RuntimeFeaturesLinked,
		[]string{},
	)
	assertStringSliceEqual(
		t,
		"summary runtime_features_initialized",
		report.Summary.RuntimeFeaturesInitialized,
		[]string{},
	)
	assertStringSliceEqual(
		t,
		"summary runtime_lazy_init_blockers",
		report.Summary.RuntimeLazyInitBlockers,
		[]string{},
	)

	rows := backendRowsByFunction(report.Functions)
	row := rows["add"]
	assertRuntimeFeatureEvidenceMarker(
		t,
		row.RuntimeFeatureEvidenceClass,
		row.RuntimeFeatureEvidenceMethod,
	)
	assertStringSliceEqual(
		t,
		"add runtime_features_required",
		row.RuntimeFeaturesRequired,
		[]string{},
	)
	assertStringSliceEqual(t, "add runtime_features_linked", row.RuntimeFeaturesLinked, []string{})
	assertStringSliceEqual(
		t,
		"add runtime_features_initialized",
		row.RuntimeFeaturesInitialized,
		[]string{},
	)
	assertStringSliceEqual(
		t,
		"add runtime_lazy_init_blockers",
		row.RuntimeLazyInitBlockers,
		[]string{},
	)
}

func TestBackendReportRuntimeFeaturesClassifyHeapActorTaskAndUnknownRuntime(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "heap_alloc",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 8},
				{Kind: ir.IRAllocBytes},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "actor_send",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRCall, Name: "__tetra_actor_send_i32", ArgSlots: 2, RetSlots: 1},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "task_spawn",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRCall, Name: "__tetra_task_spawn_i32", ArgSlots: 1, RetSlots: 1},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "future_runtime",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRCall, Name: "__tetra_future_runtime_probe", ArgSlots: 0, RetSlots: 1},
				{Kind: ir.IRReturn},
			},
		},
	}})

	assertStringSliceEqual(
		t,
		"summary runtime_features_required",
		report.Summary.RuntimeFeaturesRequired,
		[]string{"actor_runtime", "heap_runtime", "task_runtime", "unknown_runtime"},
	)
	assertStringSliceEqual(
		t,
		"summary runtime_features_linked",
		report.Summary.RuntimeFeaturesLinked,
		[]string{"actor_runtime", "heap_runtime", "task_runtime"},
	)
	assertStringSliceEqual(
		t,
		"summary runtime_features_initialized",
		report.Summary.RuntimeFeaturesInitialized,
		[]string{"actor_runtime", "heap_runtime", "task_runtime"},
	)
	assertStringSliceEqual(
		t,
		"summary runtime_lazy_init_blockers",
		report.Summary.RuntimeLazyInitBlockers,
		[]string{"unknown_runtime_call:__tetra_future_runtime_probe"},
	)

	rows := backendRowsByFunction(report.Functions)
	assertStringSliceEqual(
		t,
		"heap_alloc runtime_features_required",
		rows["heap_alloc"].RuntimeFeaturesRequired,
		[]string{"heap_runtime"},
	)
	assertStringSliceEqual(
		t,
		"actor_send runtime_features_required",
		rows["actor_send"].RuntimeFeaturesRequired,
		[]string{"actor_runtime"},
	)
	assertStringSliceEqual(
		t,
		"task_spawn runtime_features_required",
		rows["task_spawn"].RuntimeFeaturesRequired,
		[]string{"task_runtime"},
	)
	assertStringSliceEqual(
		t,
		"future_runtime runtime_features_required",
		rows["future_runtime"].RuntimeFeaturesRequired,
		[]string{"unknown_runtime"},
	)
	assertStringSliceEqual(
		t,
		"future_runtime runtime_features_linked",
		rows["future_runtime"].RuntimeFeaturesLinked,
		[]string{},
	)
	assertStringSliceEqual(
		t,
		"future_runtime runtime_lazy_init_blockers",
		rows["future_runtime"].RuntimeLazyInitBlockers,
		[]string{"unknown_runtime_call:__tetra_future_runtime_probe"},
	)
}

func TestBackendReportRuntimeObjectPlanEvidenceForSimpleProgram(t *testing.T) {
	checked, irProg := checkedAndLoweredProgram(t, `
func main() -> Int:
    return 0
`)
	report := buildBackendReport("linux-x64", irProg)
	if err := annotateBackendReportRuntimeObjectPlan(
		&report,
		"linux-x64",
		checked,
		BuildOptions{},
	); err != nil {
		t.Fatalf("annotateBackendReportRuntimeObjectPlan: %v", err)
	}

	plan := report.Summary.RuntimeObjectPlan
	assertRuntimeObjectEvidenceMarker(t, plan)
	if plan.RuntimeUsed || plan.RuntimeObjectLinked || plan.RuntimeObjectInitialized {
		t.Fatalf("runtime object plan = %+v, want no runtime object for simple program", plan)
	}
	assertStringSliceEqual(
		t,
		"simple runtime_object_features_required",
		plan.RuntimeObjectFeaturesRequired,
		[]string{},
	)
	assertStringSliceEqual(
		t,
		"simple runtime_object_features_linked",
		plan.RuntimeObjectFeaturesLinked,
		[]string{},
	)
	assertStringSliceEqual(
		t,
		"simple runtime_object_features_initialized",
		plan.RuntimeObjectFeaturesInitialized,
		[]string{},
	)
	assertStringSliceEqual(
		t,
		"simple runtime_object_lazy_init_blockers",
		plan.RuntimeObjectLazyInitBlockers,
		[]string{},
	)
}

func TestBackendReportRuntimeObjectPlanEvidenceForTaskRuntime(t *testing.T) {
	checked, irProg := checkedAndLoweredProgram(t, `
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`)
	report := buildBackendReport("linux-x64", irProg)
	if err := annotateBackendReportRuntimeObjectPlan(
		&report,
		"linux-x64",
		checked,
		BuildOptions{},
	); err != nil {
		t.Fatalf("annotateBackendReportRuntimeObjectPlan: %v", err)
	}

	plan := report.Summary.RuntimeObjectPlan
	assertRuntimeObjectEvidenceMarker(t, plan)
	if !plan.RuntimeUsed || !plan.RuntimeObjectLinked || !plan.RuntimeObjectInitialized {
		t.Fatalf("runtime object plan = %+v, want linked/initialized task runtime object", plan)
	}
	assertStringSliceEqual(
		t,
		"task runtime_object_features_required",
		plan.RuntimeObjectFeaturesRequired,
		[]string{"task_runtime"},
	)
	assertStringSliceEqual(
		t,
		"task runtime_object_features_linked",
		plan.RuntimeObjectFeaturesLinked,
		[]string{"task_runtime"},
	)
	assertStringSliceEqual(
		t,
		"task runtime_object_features_initialized",
		plan.RuntimeObjectFeaturesInitialized,
		[]string{"task_runtime"},
	)
	assertStringSliceEqual(
		t,
		"task runtime_object_lazy_init_blockers",
		plan.RuntimeObjectLazyInitBlockers,
		[]string{},
	)
}

func TestBackendMachineReportsRequireSSAVerifiedPath(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "add",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "sum",
			ParamSlots:  2,
			LocalSlots:  4,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRCmpLtI32},
				{Kind: ir.IRJmpIfZero, Label: 2},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:while:test"},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRStoreLocal, Local: 2},
				{Kind: ir.IRLoadLocal, Local: 3},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRStoreLocal, Local: 3},
				{Kind: ir.IRJmp, Label: 1},
				{Kind: ir.IRLabel, Label: 2},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRReturn},
			},
		},
	}})
	if len(report.MachineFunctions) != 2 {
		t.Fatalf(
			"machine reports = %d, want add and slice sum paths: %+v",
			len(report.MachineFunctions),
			report.MachineFunctions,
		)
	}
	for _, row := range report.MachineFunctions {
		if !row.SSAVerified || row.SSAPath != "value-ssa-v1" {
			t.Fatalf(
				"machine report %s SSA gate = verified:%v path:%q, want value-ssa-v1 verified",
				row.Function,
				row.SSAVerified,
				row.SSAPath,
			)
		}
	}
}

func TestBackendMachineReportIncludesDivModInstructionSelection(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "div_mod",
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRDivI32},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRModI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}}})
	rows := backendRowsByFunction(report.Functions)
	assertBackendCoverageRow(
		t,
		rows["div_mod"],
		"register",
		"register_path",
		"eligible_machine_ir_subset",
	)
	if len(report.MachineFunctions) != 1 {
		t.Fatalf("machine reports = %+v, want div_mod machine report", report.MachineFunctions)
	}
	machineRow := report.MachineFunctions[0]
	for _, want := range []string{"div", "mod"} {
		if !containsReportString(machineRow.InstructionSelection, want) {
			t.Fatalf("instruction selection = %+v, want %s", machineRow.InstructionSelection, want)
		}
	}
	if machineRow.Validation.StackChurnOps != 0 ||
		machineRow.Validation.MachineVerifier != "pass" ||
		machineRow.Validation.AllocationVerifier != "pass" {
		t.Fatalf(
			"machine validation = %+v, want verifier pass and no push/pop stack churn",
			machineRow.Validation,
		)
	}
}

func TestBackendReportIncludesMultiSlotReturnABIBoundary(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "add",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "slice_header_return",
			ReturnSlots: 2,
			Instrs:      []ir.IRInstr{{Kind: ir.IRReturn}},
		},
		{
			Name:        "aggregate_return",
			ReturnSlots: 3,
			Instrs:      []ir.IRInstr{{Kind: ir.IRReturn}},
		},
		{
			Name:        "call_returns_header",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRCall, Name: "slice_header_return", ArgSlots: 0, RetSlots: 2},
				{Kind: ir.IRReturn},
			},
		},
	}})
	rows := backendRowsByFunction(report.Functions)
	if rows["add"].ABI.MultiSlotReturnPolicy != "single_slot_register_return" ||
		rows["add"].ABI.ReturnSlots != 1 {
		t.Fatalf("add ABI boundary = %+v, want single-slot register return", rows["add"].ABI)
	}
	if rows["slice_header_return"].ABI.MultiSlotReturnPolicy != ("unsupported_multi_slot_"+
		"return_stack_fallback") ||
		rows["slice_header_return"].ABI.ReturnSlots != 2 {
		t.Fatalf(
			"slice return ABI boundary = %+v, want unsupported multi-slot stack fallback",
			rows["slice_header_return"].ABI,
		)
	}
	if rows["aggregate_return"].ABI.MultiSlotReturnPolicy != ("unsupported_multi_slot_return_"+
		"stack_fallback") ||
		rows["aggregate_return"].ABI.ReturnSlots != 3 {
		t.Fatalf(
			"aggregate return ABI boundary = %+v, want unsupported aggregate stack fallback",
			rows["aggregate_return"].ABI,
		)
	}
	if rows["call_returns_header"].ABI.MultiSlotReturnPolicy != ("unsupported_call_multi_" +
		"slot_return_stack_fallback") {
		t.Fatalf(
			"call multi-return ABI boundary = %+v, want unsupported call multi-slot fallback",
			rows["call_returns_header"].ABI,
		)
	}
}

func TestBackendCoverageSummaryIncludesOrdinaryCorpusNoStackChurn(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		scalarCorpusFunc("response_cost", ir.IRAddI32),
		scalarCorpusFunc("flip_count", ir.IRMulI32),
		scalarCorpusFunc("safe_pair", ir.IRSubI32),
		{
			Name:        "branch",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRJmpIfZero, Label: 1},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRReturn},
				{Kind: ir.IRLabel, Label: 1},
				{Kind: ir.IRConstI32, Imm: 2},
				{Kind: ir.IRReturn},
			},
		},
	}})

	corpus := report.Summary.OrdinaryCorpus
	if corpus.FunctionCount != 4 || corpus.RegisterPath != 3 || corpus.RegisterNoStackChurn != 3 ||
		corpus.StackFallback != 1 {
		t.Fatalf(
			"ordinary corpus summary = %+v, want 4 functions, 3 register/no-churn, 1 fallback",
			corpus,
		)
	}
	if !corpus.RegisterNoStackChurnMajority {
		t.Fatalf("ordinary corpus summary = %+v, want no-stack-churn majority", corpus)
	}
	if corpus.StackFallbackReasons["unsupported_control_flow"] != 1 {
		t.Fatalf(
			"ordinary corpus fallback reasons = %+v, want unsupported_control_flow=1",
			corpus.StackFallbackReasons,
		)
	}
	if report.Summary.MachineRegisterNoStackChurn != 3 ||
		report.Summary.MachineRegisterWithStackChurn != 0 {
		t.Fatalf(
			"machine no-stack-churn summary = %+v, want three register paths without push/pop churn",
			report.Summary,
		)
	}
}

func TestBackendReportBoundsMultiSlotHeaderAndAggregateBoundaryEvidence(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "simple_pair_return",
			ReturnSlots: 2,
			Instrs:      []ir.IRInstr{{Kind: ir.IRReturn}},
		},
		{
			Name:        "string_header_return",
			ReturnSlots: 2,
			Instrs:      []ir.IRInstr{{Kind: ir.IRReturn}},
		},
		{
			Name:        "aggregate_return",
			ReturnSlots: 3,
			Instrs:      []ir.IRInstr{{Kind: ir.IRReturn}},
		},
		{
			Name:        "call_returns_header",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRCall, Name: "string_header_return", ArgSlots: 0, RetSlots: 2},
				{Kind: ir.IRReturn},
			},
		},
	}})
	rows := backendRowsByFunction(report.Functions)

	for _, name := range []string{"simple_pair_return", "string_header_return"} {
		row := rows[name]
		assertBackendCoverageRow(
			t,
			row,
			"stack",
			"unsupported_slice_string_return",
			"unsupported_slice_or_string_return_uses_stack_fallback",
		)
		if row.ABI.ValueClass != "unverified_header_or_pair" ||
			row.ABI.BoundaryStatus != "stack_fallback_until_multi_slot_abi_verified" {
			t.Fatalf(
				"%s ABI boundary = %+v, want bounded unverified header-or-pair stack fallback",
				name,
				row.ABI,
			)
		}
	}
	if rows["aggregate_return"].ABI.ValueClass != "unverified_aggregate" ||
		rows["aggregate_return"].ABI.BoundaryStatus != "stack_fallback_until_multi_slot_abi_verified" {
		t.Fatalf(
			"aggregate ABI boundary = %+v, want bounded aggregate stack fallback",
			rows["aggregate_return"].ABI,
		)
	}
	if rows["call_returns_header"].ABI.ValueClass != "callee_multi_slot_return_unverified" ||
		rows["call_returns_header"].ABI.BoundaryStatus != "stack_fallback_until_multi_slot_abi_verified" {
		t.Fatalf(
			"call multi-slot ABI boundary = %+v, want bounded callee multi-slot fallback",
			rows["call_returns_header"].ABI,
		)
	}
	if report.Summary.ABIBoundaries.MultiSlotReturnStackFallback != 3 ||
		report.Summary.ABIBoundaries.CallMultiSlotReturnStackFallback != 1 ||
		report.Summary.ABIBoundaries.ValueClasses["unverified_header_or_pair"] != 2 ||
		report.Summary.ABIBoundaries.ValueClasses["unverified_aggregate"] != 1 {
		t.Fatalf(
			"ABI boundary summary = %+v, want bounded multi-slot/header/aggregate evidence",
			report.Summary.ABIBoundaries,
		)
	}
}

func TestBackendMachineReportValidatesCallClobbersAndSpillReloadEvidence(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:        "apply",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCall, Name: "callee", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRReturn},
		},
	}}})
	if len(report.MachineFunctions) != 1 {
		t.Fatalf("machine reports = %+v, want call machine report", report.MachineFunctions)
	}
	callRow := report.MachineFunctions[0]
	if callRow.Validation.CallClobbers != "validated" ||
		callRow.Validation.SpillReload != "validated_no_spills" ||
		!containsReportString(callRow.InstructionSelection, "call") {
		t.Fatalf(
			"call validation = %+v selection=%+v, want clobbers validated and no spills",
			callRow.Validation,
			callRow.InstructionSelection,
		)
	}

	spillFn := machine.Function{
		Name:   "spill_reload_evidence",
		Target: "test",
		Params: []machine.VReg{"a"},
		Blocks: []machine.Block{{
			Name: "entry",
			Instrs: []machine.Instr{
				{Op: machine.OpSpill, Uses: []machine.VReg{"a"}, Imm: 0},
				{Op: machine.OpReload, Defs: []machine.VReg{"b"}, Imm: 0},
				{Op: machine.OpReturn, Uses: []machine.VReg{"b"}},
			},
		}},
	}
	spillRow, ok := buildMachineBackendFunctionReport(
		spillFn,
		"machine-ir-spill-reload-evidence",
		machine.LinuxX64CallerSaved(),
		true,
	)
	if !ok {
		t.Fatalf("buildMachineBackendFunctionReport did not accept spill/reload evidence function")
	}
	if spillRow.Validation.SpillReload != "validated_spill_reload_ops" ||
		spillRow.Validation.CallClobbers != "not_applicable" ||
		spillRow.Validation.MachineVerifier != "pass" ||
		spillRow.Validation.AllocationVerifier != "pass" {
		t.Fatalf(
			"spill/reload validation = %+v, want explicit spill/reload validation evidence",
			spillRow.Validation,
		)
	}
}

func TestBoundsAndProofReportsRemoveSliceSumProofTaggedStore(t *testing.T) {
	checked, irProg := checkedAndLoweredProgram(t, `
func main() -> Int
uses alloc, mem:
    let n: Int = 4096
    var xs: []i32 = make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    var r: Int = 0
    while r < 64:
        i = 0
        while i < n:
            total = total + xs[i]
            i = i + 1
        r = r + 1
    if total > 0:
        return 0
    return 1
`)
	bounds := buildBoundsReport(irProg, checked, "linux-x64")
	if bounds.Totals.Removed != 2 || bounds.Totals.Left != 0 {
		t.Fatalf(
			"slice_sum bounds totals = removed:%d left:%d, want removed:2 left:0; report=%+v",
			bounds.Totals.Removed,
			bounds.Totals.Left,
			bounds,
		)
	}
	if len(bounds.Functions) != 1 || bounds.Functions[0].Removed != 2 ||
		bounds.Functions[0].Left != 0 {
		t.Fatalf("slice_sum bounds row = %+v, want one row removed:2 left:0", bounds.Functions)
	}
	storeSite := findBoundsSiteByKind(t, bounds, "i32.store")
	if !storeSite.Removed || storeSite.ProofID == "" ||
		storeSite.Reason != "removed_by_while_range" {
		t.Fatalf("slice_sum store site = %+v, want removed proof-tagged while range", storeSite)
	}
	loadSite := findBoundsSiteByKind(t, bounds, "i32.load")
	if !loadSite.Removed || loadSite.ProofID == "" || loadSite.Reason != "removed_by_while_range" {
		t.Fatalf(
			"slice_sum load site = %+v, want preserved removed proof-tagged while range",
			loadSite,
		)
	}

	proofProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("PLIR: %v", err)
	}
	if err := plir.VerifyProgram(proofProg); err != nil {
		t.Fatalf("PLIR verify: %v", err)
	}
	proof := buildProofReport(proofProg, bounds, "linux-x64")
	if !proofEvidenceDominates(proof.Proofs, storeSite.ProofID, "bounds_check op") {
		t.Fatalf(
			"slice_sum store proof %q missing bounds_check dominance evidence: %+v",
			storeSite.ProofID,
			proof.Proofs,
		)
	}
	if !proofEvidenceDominates(proof.Proofs, loadSite.ProofID, "bounds_check op") {
		t.Fatalf(
			"slice_sum load proof %q missing existing bounds_check dominance evidence: %+v",
			loadSite.ProofID,
			proof.Proofs,
		)
	}
}

func TestP50BoundsAndProofReportsRemoveHashLookupCallBoundaryLoads(t *testing.T) {
	checked, irProg := checkedAndLoweredFileProgram(t, `
module p25.hash_table

func lookup(keys: []i32, values: []i32, n: Int, key: Int) -> Int
uses mem:
    var i: Int = 0
    while i < n:
        if keys[i] == key:
            return values[i]
        i = i + 1
    return 0

func main() -> Int
uses alloc, mem:
    let n: Int = 256
    var keys: []i32 = core.make_i32(n)
    var values: []i32 = core.make_i32(n)
    return lookup(keys, values, n, 7)
`)
	bounds := buildBoundsReport(irProg, checked, "linux-x64")
	if bounds.Totals.Removed != 2 || bounds.Totals.Left != 0 {
		t.Fatalf(
			"hash lookup bounds totals = removed:%d left:%d, want removed:2 left:0; report=%+v",
			bounds.Totals.Removed,
			bounds.Totals.Left,
			bounds,
		)
	}
	var lookupRow boundsFunctionRow
	for _, row := range bounds.Functions {
		if row.Function == "p25.hash_table.lookup" {
			lookupRow = row
			break
		}
	}
	if lookupRow.Function == "" || lookupRow.Removed != 2 || lookupRow.Left != 0 {
		t.Fatalf(
			"hash lookup bounds row = %+v, rows=%+v; want removed:2 left:0",
			lookupRow,
			bounds.Functions,
		)
	}
	if len(lookupRow.Sites) != 2 {
		t.Fatalf("hash lookup sites = %+v, want two removed load sites", lookupRow.Sites)
	}
	for _, site := range lookupRow.Sites {
		if !site.Removed || site.Kind != "i32.load" ||
			!strings.HasPrefix(site.ProofID, "proof:call-boundary:i:") ||
			site.Reason != "removed_by_call_boundary_range" {
			t.Fatalf("hash lookup bounds site = %+v, want removed call-boundary load", site)
		}
	}

	proofProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("PLIR: %v", err)
	}
	if err := plir.VerifyProgram(proofProg); err != nil {
		t.Fatalf("PLIR verify: %v", err)
	}
	proof := buildProofReport(proofProg, bounds, "linux-x64")
	for _, site := range lookupRow.Sites {
		if !proofEvidenceDominates(proof.Proofs, site.ProofID, "bounds_check op") {
			t.Fatalf(
				"hash lookup proof %q missing bounds_check dominance evidence: %+v",
				site.ProofID,
				proof.Proofs,
			)
		}
	}
}

func TestP55BoundsAndProofReportsRemoveAllocationLiteralZeroStoreLoad(t *testing.T) {
	checked, irProg := checkedAndLoweredFileProgram(t, `
module p25.allocation

func main() -> Int
uses alloc, mem:
    var checksum: Int = 0
    var r: Int = 0
    while r < 1024:
        var xs: []i32 = core.make_i32(32)
        xs[0] = r
        checksum = checksum + xs[0]
        r = r + 1
    if checksum > 0:
        return 0
    return 1
`)
	bounds := buildBoundsReport(irProg, checked, "linux-x64")
	if bounds.Totals.Removed != 2 || bounds.Totals.Left != 0 {
		t.Fatalf(
			"allocation_tetra bounds totals = removed:%d left:%d, want removed:2 left:0; report=%+v",
			bounds.Totals.Removed,
			bounds.Totals.Left,
			bounds,
		)
	}
	if len(bounds.Functions) != 1 || bounds.Functions[0].Function != "p25.allocation.main" ||
		bounds.Functions[0].Removed != 2 ||
		bounds.Functions[0].Left != 0 {
		t.Fatalf(
			"allocation_tetra bounds row = %+v, want p25.allocation.main removed:2 left:0",
			bounds.Functions,
		)
	}
	storeSite := findBoundsSiteByKind(t, bounds, "i32.store")
	if !storeSite.Removed ||
		!strings.HasPrefix(storeSite.ProofID, "proof:allocation-zero:literal0:xs:") ||
		storeSite.Reason != "removed_by_allocation_literal_zero_length" {
		t.Fatalf(
			"allocation_tetra store site = %+v, want removed allocation literal-zero proof",
			storeSite,
		)
	}
	loadSite := findBoundsSiteByKind(t, bounds, "i32.load")
	if !loadSite.Removed ||
		!strings.HasPrefix(loadSite.ProofID, "proof:allocation-zero:literal0:xs:") ||
		loadSite.Reason != "removed_by_allocation_literal_zero_length" {
		t.Fatalf(
			"allocation_tetra load site = %+v, want removed allocation literal-zero proof",
			loadSite,
		)
	}

	proofProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("PLIR: %v", err)
	}
	if err := plir.VerifyProgram(proofProg); err != nil {
		t.Fatalf("PLIR verify: %v", err)
	}
	proof := buildProofReport(proofProg, bounds, "linux-x64")
	if !proofEvidenceDominates(proof.Proofs, storeSite.ProofID, "bounds_check op") {
		t.Fatalf(
			"allocation_tetra store proof %q missing bounds_check dominance evidence: %+v",
			storeSite.ProofID,
			proof.Proofs,
		)
	}
	if !proofEvidenceDominates(proof.Proofs, loadSite.ProofID, "bounds_check op") {
		t.Fatalf(
			"allocation_tetra load proof %q missing bounds_check dominance evidence: %+v",
			loadSite.ProofID,
			proof.Proofs,
		)
	}
}

func TestP58BoundsAndProofReportsRemoveRegionIslandLiteralZeroStoreLoad(t *testing.T) {
	checked, irProg := checkedAndLoweredFileProgram(t, `
module p25.region_island_allocation

func main() -> Int
uses alloc, islands, mem:
    var checksum: Int = 0
    var r: Int = 0
    while r < 256:
        island(256) as isl:
            var xs: []i32 = core.island_make_i32(isl, 16)
            xs[0] = r
            checksum = checksum + xs[0]
        r = r + 1
    if checksum > 0:
        return 0
    return 1
`)
	bounds := buildBoundsReport(irProg, checked, "linux-x64")
	if bounds.Totals.Removed != 2 || bounds.Totals.Left != 0 {
		t.Fatalf(
			"region_island_allocation bounds totals = removed:%d left:%d, want removed:2 left:0; report=%+v",
			bounds.Totals.Removed,
			bounds.Totals.Left,
			bounds,
		)
	}
	if len(bounds.Functions) != 1 ||
		bounds.Functions[0].Function != "p25.region_island_allocation.main" ||
		bounds.Functions[0].Removed != 2 ||
		bounds.Functions[0].Left != 0 {
		t.Fatalf(
			("region_island_allocation bounds row = %+v, want " +
				"p25.region_island_allocation.main removed:2 left:0"),
			bounds.Functions,
		)
	}
	storeSite := findBoundsSiteByKind(t, bounds, "i32.store")
	if !storeSite.Removed ||
		!strings.HasPrefix(storeSite.ProofID, "proof:allocation-zero:literal0:xs:") ||
		storeSite.Reason != "removed_by_allocation_literal_zero_length" {
		t.Fatalf(
			"region_island_allocation store site = %+v, want removed allocation literal-zero proof",
			storeSite,
		)
	}
	loadSite := findBoundsSiteByKind(t, bounds, "i32.load")
	if !loadSite.Removed ||
		!strings.HasPrefix(loadSite.ProofID, "proof:allocation-zero:literal0:xs:") ||
		loadSite.Reason != "removed_by_allocation_literal_zero_length" {
		t.Fatalf(
			"region_island_allocation load site = %+v, want removed allocation literal-zero proof",
			loadSite,
		)
	}

	proofProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("PLIR: %v", err)
	}
	if err := plir.VerifyProgram(proofProg); err != nil {
		t.Fatalf("PLIR verify: %v", err)
	}
	proof := buildProofReport(proofProg, bounds, "linux-x64")
	if !proofEvidenceDominates(proof.Proofs, storeSite.ProofID, "bounds_check op") {
		t.Fatalf(
			"region_island_allocation store proof %q missing bounds_check dominance evidence: %+v",
			storeSite.ProofID,
			proof.Proofs,
		)
	}
	if !proofEvidenceDominates(proof.Proofs, loadSite.ProofID, "bounds_check op") {
		t.Fatalf(
			"region_island_allocation load proof %q missing bounds_check dominance evidence: %+v",
			loadSite.ProofID,
			proof.Proofs,
		)
	}

	backend := buildBackendReport("linux-x64", irProg)
	rows := backendRowsByFunction(backend.Functions)
	mainRow := rows["p25.region_island_allocation.main"]
	if mainRow.BackendPath != "register" || mainRow.Category != "register_path" ||
		mainRow.Reason != "eligible_machine_ir_subset" ||
		mainRow.Detail != "machine-ir-region-island-allocation-main" {
		t.Fatalf(
			"region_island_allocation backend row = %+v, want real machine register path",
			mainRow,
		)
	}
}

func TestP110BackendReportPromotesExactRegionIslandAllocationMain(t *testing.T) {
	_, irProg := checkedAndLoweredFileProgram(t, `
module p25.region_island_allocation

func main() -> Int
uses alloc, islands, mem:
    var checksum: Int = 0
    var r: Int = 0
    while r < 256:
        island(256) as isl:
            var xs: []i32 = core.island_make_i32(isl, 16)
            xs[0] = r
            checksum = checksum + xs[0]
        r = r + 1
    if checksum > 0:
        return 0
    return 1
`)
	mainFn := findIRFunc(t, irProg.Funcs, "p25.region_island_allocation.main")
	report := buildBackendReport("linux-x64", irProg)
	rows := backendRowsByFunction(report.Functions)
	mainRow := rows["p25.region_island_allocation.main"]
	if mainRow.BackendPath != "register" || mainRow.Category != "register_path" ||
		mainRow.Reason != "eligible_machine_ir_subset" {
		t.Fatalf(
			"region_island_allocation backend row = %+v, want register_path; instrs=%#v",
			mainRow,
			mainFn.Instrs,
		)
	}
	if mainRow.Detail != "machine-ir-region-island-allocation-main" {
		t.Fatalf(
			"region_island_allocation detail = %q, "+
				"want machine-ir-region-island-allocation-main; row=%+v instrs=%#v",
			mainRow.Detail,
			mainRow,
			mainFn.Instrs,
		)
	}
	if report.Summary.FunctionCount != 1 || report.Summary.RegisterPath != 1 ||
		report.Summary.StackFallback != 0 || report.Summary.Categories["register_path"] != 1 {
		t.Fatalf(
			"region_island_allocation backend summary = %+v, want one register_path and no fallback",
			report.Summary,
		)
	}
	machineRows := map[string]machineBackendFunctionReport{}
	for _, row := range report.MachineFunctions {
		machineRows[row.Function] = row
	}
	mainMachineRow, ok := machineRows["p25.region_island_allocation.main"]
	if !ok {
		t.Fatalf(
			"machine report missing for p25.region_island_allocation.main: %+v",
			report.MachineFunctions,
		)
	}
	if mainMachineRow.Path != "machine-ir-region-island-allocation-main" ||
		!mainMachineRow.SSAVerified ||
		mainMachineRow.Validation.StackChurnOps != 0 {
		t.Fatalf(
			"region_island_allocation machine row = %+v, want verified no-stack-churn path",
			mainMachineRow,
		)
	}
	for _, want := range []string{"index_store", "index_load", "cmp", "add"} {
		if !containsReportString(mainMachineRow.InstructionSelection, want) {
			t.Fatalf(
				"region_island_allocation instruction selection = %+v, want %q",
				mainMachineRow.InstructionSelection,
				want,
			)
		}
	}
	assertStringSliceEqual(
		t,
		"region_island_allocation runtime_features_required",
		mainRow.RuntimeFeaturesRequired,
		[]string{"island_allocator"},
	)
}

func TestP63BoundsAndProofReportsRemoveJsonWriteMessageObjectStores(t *testing.T) {
	checked, irProg := checkedAndLoweredFileProgram(t, `
module p25.json_parse_stringify

func write_message_object(dst: inout []u8) -> Int
uses mem:
    dst[0] = 123
    dst[1] = 34
    dst[2] = 109
    dst[3] = 101
    dst[4] = 115
    dst[5] = 115
    dst[6] = 97
    dst[7] = 103
    dst[8] = 101
    dst[9] = 34
    dst[10] = 58
    dst[11] = 34
    dst[12] = 72
    dst[13] = 101
    dst[14] = 108
    dst[15] = 108
    dst[16] = 111
    dst[17] = 44
    dst[18] = 32
    dst[19] = 87
    dst[20] = 111
    dst[21] = 114
    dst[22] = 108
    dst[23] = 100
    dst[24] = 33
    dst[25] = 34
    dst[26] = 125
    return 27

func main() -> Int
uses alloc, mem:
    var buf: []u8 = core.make_u8(128)
    var i: Int = 0
    var total: Int = 0
    while i < 2048:
        total = total + write_message_object(buf)
        i = i + 1
    if total == 55296:
        return 0
    return 1
`)
	bounds := buildBoundsReport(irProg, checked, "linux-x64")
	if bounds.Totals.Removed != 27 || bounds.Totals.Left != 0 {
		t.Fatalf(
			"json helper bounds totals = removed:%d left:%d, want removed:27 left:0; report=%+v",
			bounds.Totals.Removed,
			bounds.Totals.Left,
			bounds,
		)
	}
	var helperRow boundsFunctionRow
	for _, row := range bounds.Functions {
		if row.Function == "p25.json_parse_stringify.write_message_object" {
			helperRow = row
			break
		}
	}
	if helperRow.Function == "" || helperRow.Removed != 27 || helperRow.Left != 0 {
		t.Fatalf(
			"json helper bounds row = %+v, rows=%+v; want removed:27 left:0",
			helperRow,
			bounds.Functions,
		)
	}
	if len(helperRow.Sites) != 27 {
		t.Fatalf("json helper sites = %+v, want 27 removed store sites", helperRow.Sites)
	}
	proofProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("PLIR: %v", err)
	}
	if err := plir.VerifyProgram(proofProg); err != nil {
		t.Fatalf("PLIR verify: %v", err)
	}
	proof := buildProofReport(proofProg, bounds, "linux-x64")
	for _, site := range helperRow.Sites {
		if !site.Removed ||
			site.Kind != "u8.store" ||
			!strings.HasPrefix(site.ProofID, "proof:helper-summary:") ||
			site.Reason != "removed_by_helper_summary_range" {
			t.Fatalf("json helper bounds site = %+v, want removed helper-summary u8.store", site)
		}
		if !proofEvidenceDominates(proof.Proofs, site.ProofID, "bounds_check op") {
			t.Fatalf(
				"json helper proof %q missing bounds_check dominance evidence: %+v",
				site.ProofID,
				proof.Proofs,
			)
		}
	}
}

func TestP65BoundsAndProofReportsRemovePostgreSQLHelperOffsetAccesses(t *testing.T) {
	checked, irProg := checkedAndLoweredFileProgram(t, `
module p25.postgresql_single_multiple_update

func frame_data_row() -> Int:
    return 68

func frame_payload_start(offset: Int) -> Int:
    return offset + 5

func frame_type_at(src: []u8, offset: Int) -> Int
uses mem:
    return src[offset]

func write_i32_be_at(dst: inout []u8, start: Int, value: Int) -> Int
uses mem:
    dst[start] = (value / 16777216) % 256
    dst[start + 1] = (value / 65536) % 256
    dst[start + 2] = (value / 256) % 256
    dst[start + 3] = value % 256
    return start + 4

func write_i16_be_at(dst: inout []u8, start: Int, value: Int) -> Int
uses mem:
    dst[start] = (value / 256) % 256
    dst[start + 1] = value % 256
    return start + 2

func main() -> Int
uses alloc, mem:
    var frame: []u8 = core.make_u8(64)
    var i: Int = 0
    var total: Int = 0
    while i < 2048:
        frame[0] = frame_data_row()
        var pos: Int = write_i32_be_at(frame, 1, 12)
        pos = write_i16_be_at(frame, pos, 2)
        total = total + frame_type_at(frame, 0) + frame_payload_start(0)
        i = i + 1
    if total > 0:
        return 0
    return 1
`)
	bounds := buildBoundsReport(irProg, checked, "linux-x64")
	if bounds.Totals.Removed != 8 || bounds.Totals.Left != 0 {
		t.Fatalf(
			"postgresql helper-offset bounds totals = removed:%d left:%d, want removed:8 left:0; report=%+v",
			bounds.Totals.Removed,
			bounds.Totals.Left,
			bounds,
		)
	}
	rows := map[string]boundsFunctionRow{}
	for _, row := range bounds.Functions {
		rows[row.Function] = row
	}
	expected := map[string]struct {
		removed int
		kind    string
	}{
		"p25.postgresql_single_multiple_update.frame_type_at":   {removed: 1, kind: "u8.load"},
		"p25.postgresql_single_multiple_update.write_i32_be_at": {removed: 4, kind: "u8.store"},
		"p25.postgresql_single_multiple_update.write_i16_be_at": {removed: 2, kind: "u8.store"},
		"p25.postgresql_single_multiple_update.main":            {removed: 1, kind: "u8.store"},
	}
	for fnName, want := range expected {
		row := rows[fnName]
		if row.Function == "" || row.Removed != want.removed || row.Left != 0 ||
			len(row.Sites) != want.removed {
			t.Fatalf(
				"%s bounds row = %+v, want removed:%d left:0 sites:%d",
				fnName,
				row,
				want.removed,
				want.removed,
			)
		}
	}

	proofProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("PLIR: %v", err)
	}
	if err := plir.VerifyProgram(proofProg); err != nil {
		t.Fatalf("PLIR verify: %v", err)
	}
	proof := buildProofReport(proofProg, bounds, "linux-x64")
	for fnName, row := range rows {
		for _, site := range row.Sites {
			switch fnName {
			case "p25.postgresql_single_multiple_update.frame_type_at",
				"p25.postgresql_single_multiple_update.write_i32_be_at",
				"p25.postgresql_single_multiple_update.write_i16_be_at":
				if !site.Removed ||
					!strings.HasPrefix(site.ProofID, "proof:helper-offset:") ||
					site.Reason != "removed_by_helper_offset_range" {
					t.Fatalf(
						"postgresql helper-offset bounds site = %+v, want removed helper-offset",
						site,
					)
				}
				if !proofEvidenceDominates(proof.Proofs, site.ProofID, "bounds_check op") {
					t.Fatalf(
						"postgresql helper-offset proof %q missing bounds_check dominance evidence: %+v",
						site.ProofID,
						proof.Proofs,
					)
				}
			case "p25.postgresql_single_multiple_update.main":
				if !site.Removed ||
					!strings.HasPrefix(site.ProofID, "proof:allocation-zero:") ||
					site.Reason != "removed_by_allocation_literal_zero_length" {
					t.Fatalf(
						"postgresql allocation-zero bounds site = %+v, want retained allocation-zero proof",
						site,
					)
				}
			}
		}
	}
}

func TestP69BackendReportPromotesPostgreSQLFrameTypeAtButKeepsMainFallback(t *testing.T) {
	_, irProg := checkedAndLoweredFileProgram(t, `
module p25.postgresql_single_multiple_update

func frame_data_row() -> Int:
    return 68

func frame_payload_start(offset: Int) -> Int:
    return offset + 5

func frame_type_at(src: []u8, offset: Int) -> Int
uses mem:
    return src[offset]

func write_i32_be_at(dst: inout []u8, start: Int, value: Int) -> Int
uses mem:
    dst[start] = (value / 16777216) % 256
    dst[start + 1] = (value / 65536) % 256
    dst[start + 2] = (value / 256) % 256
    dst[start + 3] = value % 256
    return start + 4

func write_i16_be_at(dst: inout []u8, start: Int, value: Int) -> Int
uses mem:
    dst[start] = (value / 256) % 256
    dst[start + 1] = value % 256
    return start + 2

func main() -> Int
uses alloc, mem:
    var frame: []u8 = core.make_u8(64)
    var i: Int = 0
    var total: Int = 0
    while i < 2048:
        frame[0] = frame_data_row()
        var pos: Int = write_i32_be_at(frame, 1, 12)
        pos = write_i16_be_at(frame, pos, 2)
        total = total + frame_type_at(frame, 0) + frame_payload_start(0)
        i = i + 1
    if total > 0:
        return 0
    return 1
`)
	report := buildBackendReport("linux-x64", irProg)
	rows := backendRowsByFunction(report.Functions)
	frameType := rows["p25.postgresql_single_multiple_update.frame_type_at"]
	assertBackendCoverageRow(t, frameType, "register", "register_path", "eligible_machine_ir_subset")
	if frameType.Detail != "machine-ir-postgresql-frame-type-at" {
		t.Fatalf(
			"frame_type_at detail = %q, want machine-ir-postgresql-frame-type-at (rows=%+v)",
			frameType.Detail,
			report.Functions,
		)
	}

	if report.Summary.FunctionCount != 6 || report.Summary.RegisterPath != 6 ||
		report.Summary.StackFallback != 0 || report.Summary.Categories["register_path"] != 6 {
		t.Fatalf(
			"postgresql backend summary = %+v, want all exact PostgreSQL helpers on register path",
			report.Summary,
		)
	}
}

func TestP72BackendReportPromotesExactPostgreSQLInoutWritersAndMain(t *testing.T) {
	_, irProg := checkedAndLoweredFileProgram(t, `
module p25.postgresql_single_multiple_update

func frame_data_row() -> Int:
    return 68

func frame_payload_start(offset: Int) -> Int:
    return offset + 5

func frame_type_at(src: []u8, offset: Int) -> Int
uses mem:
    return src[offset]

func write_i32_be_at(dst: inout []u8, start: Int, value: Int) -> Int
uses mem:
    dst[start] = (value / 16777216) % 256
    dst[start + 1] = (value / 65536) % 256
    dst[start + 2] = (value / 256) % 256
    dst[start + 3] = value % 256
    return start + 4

func write_i16_be_at(dst: inout []u8, start: Int, value: Int) -> Int
uses mem:
    dst[start] = (value / 256) % 256
    dst[start + 1] = value % 256
    return start + 2

func main() -> Int
uses alloc, mem:
    var frame: []u8 = core.make_u8(64)
    var i: Int = 0
    var total: Int = 0
    while i < 2048:
        frame[0] = frame_data_row()
        var pos: Int = write_i32_be_at(frame, 1, 12)
        pos = write_i16_be_at(frame, pos, 2)
        total = total + frame_type_at(frame, 0) + frame_payload_start(0)
        i = i + 1
    if total > 0:
        return 0
    return 1
`)
	irProg.Funcs = append(irProg.Funcs,
		ir.IRFunc{
			Name:        "aggregate_return",
			ReturnSlots: 3,
			Instrs:      []ir.IRInstr{{Kind: ir.IRReturn}},
		},
		ir.IRFunc{
			Name:        "p25.http_plaintext_json.write_plaintext_response",
			ParamSlots:  4,
			LocalSlots:  4,
			ReturnSlots: 3,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: 72},
				{Kind: ir.IRIndexStoreU8, ProofID: "proof:helper-offset:0:dst:1:1"},
				{Kind: ir.IRLoadLocal, Local: 2},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRAddI32},
				{Kind: ir.IRReturn},
			},
		},
	)

	report := buildBackendReport("linux-x64", irProg)
	rows := backendRowsByFunction(report.Functions)
	for _, tc := range []struct {
		name   string
		detail string
	}{
		{
			name:   "p25.postgresql_single_multiple_update.write_i32_be_at",
			detail: "machine-ir-postgresql-inout-writer",
		},
		{
			name:   "p25.postgresql_single_multiple_update.write_i16_be_at",
			detail: "machine-ir-postgresql-inout-writer",
		},
		{
			name:   "p25.postgresql_single_multiple_update.main",
			detail: "machine-ir-postgresql-inout-writer-main",
		},
	} {
		row := rows[tc.name]
		assertBackendCoverageRow(t, row, "register", "register_path", "eligible_machine_ir_subset")
		if row.Detail != tc.detail {
			t.Fatalf("%s detail = %q, want %q (rows=%+v)", tc.name, row.Detail, tc.detail, report.Functions)
		}
		if row.Category == "unsupported_aggregate_return" || row.Category == "unsupported_call_abi" {
			t.Fatalf("%s still reports old PostgreSQL ABI blocker: %+v", tc.name, row)
		}
		if row.ABI.MultiSlotReturnPolicy != "single_slot_register_return" ||
			row.ABI.BoundaryStatus != "register_return_verified" {
			t.Fatalf("%s ABI boundary = %+v, want exact internal register verification", tc.name, row.ABI)
		}
	}

	assertBackendCoverageRow(
		t,
		rows["aggregate_return"],
		"stack",
		"unsupported_aggregate_return",
		"unsupported_aggregate_return_uses_stack_fallback",
	)
	assertBackendCoverageRow(
		t,
		rows["p25.http_plaintext_json.write_plaintext_response"],
		"stack",
		"unsupported_aggregate_return",
		"unsupported_aggregate_return_uses_stack_fallback",
	)
}

func TestP90BackendReportPromotesExactHelperSummaryWritersAndMain(t *testing.T) {
	assertExactHelperSummaryReport := func(
		t *testing.T,
		report backendReport,
		wantFunctionCount int,
		wantDetails map[string]string,
	) {
		t.Helper()
		rows := backendRowsByFunction(report.Functions)
		for name, detail := range wantDetails {
			row := rows[name]
			assertBackendCoverageRow(t, row, "register", "register_path", "eligible_machine_ir_subset")
			if row.Detail != detail {
				t.Fatalf("%s detail = %q, want %q (rows=%+v)", name, row.Detail, detail, report.Functions)
			}
			if row.ABI.MultiSlotReturnPolicy != "single_slot_register_return" ||
				row.ABI.ValueClass != "single_register_slot" ||
				row.ABI.BoundaryStatus != "register_return_verified" {
				t.Fatalf("%s ABI boundary = %+v, want exact helper-summary internal ABI", name, row.ABI)
			}
		}
		if report.Summary.FunctionCount != wantFunctionCount ||
			report.Summary.RegisterPath != wantFunctionCount ||
			report.Summary.StackFallback != 0 ||
			report.Summary.Categories["register_path"] != wantFunctionCount {
			t.Fatalf(
				"helper-summary backend summary = %+v, want function/register=%d stack_fallback=0",
				report.Summary,
				wantFunctionCount,
			)
		}
	}

	_, jsonIR := checkedAndLoweredFileProgram(t, `
module p25.json_parse_stringify

func write_message_object(dst: inout []u8) -> Int
uses mem:
    dst[0] = 123
    dst[1] = 34
    dst[2] = 109
    dst[3] = 101
    dst[4] = 115
    dst[5] = 115
    dst[6] = 97
    dst[7] = 103
    dst[8] = 101
    dst[9] = 34
    dst[10] = 58
    dst[11] = 34
    dst[12] = 72
    dst[13] = 101
    dst[14] = 108
    dst[15] = 108
    dst[16] = 111
    dst[17] = 44
    dst[18] = 32
    dst[19] = 87
    dst[20] = 111
    dst[21] = 114
    dst[22] = 108
    dst[23] = 100
    dst[24] = 33
    dst[25] = 34
    dst[26] = 125
    return 27

func main() -> Int
uses alloc, mem:
    var buf: []u8 = core.make_u8(128)
    var i: Int = 0
    var total: Int = 0
    while i < 2048:
        total = total + write_message_object(buf)
        i = i + 1
    if total == 55296:
        return 0
    return 1
`)
	assertExactHelperSummaryReport(
		t,
		buildBackendReport("linux-x64", jsonIR),
		2,
		map[string]string{
			"p25.json_parse_stringify.write_message_object": "machine-ir-inout-writer-helper-summary",
			"p25.json_parse_stringify.main":                 "machine-ir-inout-writer-helper-summary-caller",
		},
	)

	_, httpIR := checkedAndLoweredFileProgram(t, `
module p25.http_plaintext_json

func write_plaintext_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    dst[1] = 84
    dst[2] = 84
    dst[3] = 80
    dst[4] = 47
    dst[5] = 49
    dst[6] = 46
    dst[7] = 49
    dst[8] = 32
    dst[9] = 50
    dst[10] = 48
    dst[11] = 48
    dst[12] = 32
    dst[13] = 79
    dst[14] = 75
    dst[15] = 13
    dst[16] = 10
    dst[17] = 13
    dst[18] = 10
    dst[19] = 72
    dst[20] = 101
    dst[21] = 108
    dst[22] = 108
    dst[23] = 111
    return 24

func write_json_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    dst[1] = 84
    dst[2] = 84
    dst[3] = 80
    dst[4] = 47
    dst[5] = 49
    dst[6] = 46
    dst[7] = 49
    dst[8] = 32
    dst[9] = 50
    dst[10] = 48
    dst[11] = 48
    dst[12] = 32
    dst[13] = 79
    dst[14] = 75
    dst[15] = 13
    dst[16] = 10
    dst[17] = 13
    dst[18] = 10
    dst[19] = 123
    dst[20] = 125
    return 21

func main() -> Int
uses alloc, mem:
    var plain: []u8 = core.make_u8(192)
    var json_buf: []u8 = core.make_u8(192)
    var i: Int = 0
    var total: Int = 0
    while i < 1024:
        total = total + write_plaintext_response(plain)
        total = total + write_json_response(json_buf)
        i = i + 1
    if total > 0:
        return 0
    return 1
`)
	const helperSummaryCaller = "machine-ir-inout-writer-helper-summary-caller"
	assertExactHelperSummaryReport(
		t,
		buildBackendReport("linux-x64", httpIR),
		3,
		map[string]string{
			"p25.http_plaintext_json.write_plaintext_response": "machine-ir-inout-writer-helper-summary",
			"p25.http_plaintext_json.write_json_response":      "machine-ir-inout-writer-helper-summary",
			"p25.http_plaintext_json.main":                     helperSummaryCaller,
		},
	)
}

func TestP90BackendReportKeepsGenericAggregateAndMixedCallsFallback(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "aggregate_return",
			ReturnSlots: 3,
			Instrs:      []ir.IRInstr{{Kind: ir.IRReturn}},
		},
		{
			Name:        "p25.json_parse_stringify.write_message_object",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 3,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 123},
				{Kind: ir.IRIndexStoreU8, ProofID: "proof:helper-summary:wrong-shape"},
				{Kind: ir.IRConstI32, Imm: 1},
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "p25.json_parse_stringify.write_message_object_wrong",
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 3,
			Instrs:      []ir.IRInstr{{Kind: ir.IRReturn}},
		},
		{
			Name:        "p25.json_parse_stringify.main",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{
					Kind:     ir.IRCall,
					Name:     "p25.json_parse_stringify.write_message_object",
					ArgSlots: 2,
					RetSlots: 3,
				},
				{
					Kind:     ir.IRCall,
					Name:     "p25.json_parse_stringify.unverified_writer",
					ArgSlots: 2,
					RetSlots: 3,
				},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRReturn},
			},
		},
	}})
	rows := backendRowsByFunction(report.Functions)

	for _, name := range []string{
		"aggregate_return",
		"p25.json_parse_stringify.write_message_object",
		"p25.json_parse_stringify.write_message_object_wrong",
	} {
		assertBackendCoverageRow(
			t,
			rows[name],
			"stack",
			"unsupported_aggregate_return",
			"unsupported_aggregate_return_uses_stack_fallback",
		)
		if rows[name].Detail == "machine-ir-inout-writer-helper-summary" {
			t.Fatalf("%s incorrectly received helper-summary detail: %+v", name, rows[name])
		}
	}
	assertBackendCoverageRow(
		t,
		rows["p25.json_parse_stringify.main"],
		"stack",
		"unsupported_call_abi",
		"unsupported_call_abi_uses_stack_fallback",
	)
	const helperSummaryCaller = "machine-ir-inout-writer-helper-summary-caller"
	mainRow := rows["p25.json_parse_stringify.main"]
	if mainRow.Detail == helperSummaryCaller {
		t.Fatalf("mixed caller incorrectly received helper-summary caller detail: %+v", mainRow)
	}
}

func TestP93BackendReportPromotesExactParallelMapReduceMain(t *testing.T) {
	_, irProg := checkedAndLoweredFileProgram(t, `
module p25.parallel_map_reduce

func left_worker() -> Int:
    return 13

func mid_worker() -> Int:
    return 17

func right_worker() -> Int:
    return 12

func main() -> Int
uses runtime:
    let left: task.i32 = core.task_spawn_i32("left_worker")
    let mid: task.i32 = core.task_spawn_i32("mid_worker")
    let right: task.i32 = core.task_spawn_i32("right_worker")
    let total: Int = core.task_join_i32(left) + core.task_join_i32(mid) + core.task_join_i32(right)
    if total == 42:
        return 0
    return total
`)
	mainFn := findIRFunc(t, irProg.Funcs, "p25.parallel_map_reduce.main")
	spawnCalls := 0
	joinCalls := 0
	for _, instr := range mainFn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == "__tetra_task_spawn_i32" {
			spawnCalls++
			if instr.ArgSlots != 1 || instr.RetSlots != 2 {
				t.Fatalf("spawn call ABI = args %d rets %d, want 1 -> 2", instr.ArgSlots, instr.RetSlots)
			}
		}
		if instr.Kind == ir.IRCall && instr.Name == "__tetra_task_join_i32" {
			joinCalls++
			if instr.ArgSlots != 2 || instr.RetSlots != 1 {
				t.Fatalf("join call ABI = args %d rets %d, want 2 -> 1", instr.ArgSlots, instr.RetSlots)
			}
		}
	}
	if spawnCalls != 3 || joinCalls != 3 {
		t.Fatalf(
			"parallel_map_reduce calls = spawns %d joins %d, want 3/3: %#v",
			spawnCalls,
			joinCalls,
			mainFn.Instrs,
		)
	}

	report := buildBackendReport("linux-x64", irProg)
	rows := backendRowsByFunction(report.Functions)
	for _, name := range []string{
		"p25.parallel_map_reduce.left_worker",
		"p25.parallel_map_reduce.mid_worker",
		"p25.parallel_map_reduce.right_worker",
		"p25.parallel_map_reduce.main",
	} {
		assertBackendCoverageRow(t, rows[name], "register", "register_path", "eligible_machine_ir_subset")
		if rows[name].ABI.MultiSlotReturnPolicy != "single_slot_register_return" ||
			rows[name].ABI.BoundaryStatus != "register_return_verified" {
			t.Fatalf("%s ABI boundary = %+v, want verified register boundary", name, rows[name].ABI)
		}
	}
	if rows["p25.parallel_map_reduce.main"].Detail != "machine-ir-parallel-map-reduce-main" {
		t.Fatalf(
			"parallel_map_reduce main detail = %q, want machine-ir-parallel-map-reduce-main (rows=%+v)",
			rows["p25.parallel_map_reduce.main"].Detail,
			report.Functions,
		)
	}
	if report.Summary.FunctionCount != 4 || report.Summary.RegisterPath != 4 ||
		report.Summary.StackFallback != 0 || report.Summary.Categories["register_path"] != 4 {
		t.Fatalf(
			"parallel_map_reduce backend summary = %+v, want all four functions register",
			report.Summary,
		)
	}
	machineRows := map[string]machineBackendFunctionReport{}
	for _, row := range report.MachineFunctions {
		machineRows[row.Function] = row
	}
	mainRow := machineRows["p25.parallel_map_reduce.main"]
	if mainRow.Path != "machine-ir-parallel-map-reduce-main" ||
		!mainRow.SSAVerified ||
		mainRow.Validation.CallClobbers != "validated" ||
		mainRow.Validation.StackChurnOps != 0 {
		t.Fatalf(
			"parallel_map_reduce main machine row = %+v, want verified task spawn/join path",
			mainRow,
		)
	}
	for _, want := range []string{"call", "add", "cmp"} {
		if !containsReportString(mainRow.InstructionSelection, want) {
			t.Fatalf(
				"parallel_map_reduce main instruction selection = %+v, want %q",
				mainRow.InstructionSelection,
				want,
			)
		}
	}
}

func TestP93BackendReportKeepsUnrelatedMultiReturnCallsFallback(t *testing.T) {
	report := buildBackendReport("linux-x64", &ir.IRProgram{Funcs: []ir.IRFunc{
		{
			Name:        "different_runtime_ret2",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRCall, Name: "__tetra_task_poll_i32", ArgSlots: 2, RetSlots: 2},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "ret_slots_three_call",
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRCall, Name: "__tetra_task_spawn_i32", ArgSlots: 1, RetSlots: 3},
				{Kind: ir.IRReturn},
			},
		},
		{
			Name:        "aggregate_helper",
			ReturnSlots: 3,
			Instrs:      []ir.IRInstr{{Kind: ir.IRReturn}},
		},
	}})
	rows := backendRowsByFunction(report.Functions)
	for _, name := range []string{"different_runtime_ret2", "ret_slots_three_call"} {
		assertBackendCoverageRow(
			t,
			rows[name],
			"stack",
			"unsupported_call_abi",
			"unsupported_call_abi_uses_stack_fallback",
		)
		if rows[name].Detail == "machine-ir-parallel-map-reduce-main" {
			t.Fatalf("%s incorrectly received parallel map/reduce detail: %+v", name, rows[name])
		}
	}
	assertBackendCoverageRow(
		t,
		rows["aggregate_helper"],
		"stack",
		"unsupported_aggregate_return",
		"unsupported_aggregate_return_uses_stack_fallback",
	)
}

func TestP67BoundsAndProofReportsRemoveHTTPMultiHelperSummaryStores(t *testing.T) {
	checked, irProg := checkedAndLoweredFileProgram(t, `
module p25.http_plaintext_json

func write_plaintext_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    dst[1] = 84
    dst[2] = 84
    dst[3] = 80
    dst[4] = 47
    dst[5] = 49
    dst[6] = 46
    dst[7] = 49
    dst[8] = 32
    dst[9] = 50
    dst[10] = 48
    dst[11] = 48
    dst[12] = 32
    dst[13] = 79
    dst[14] = 75
    dst[15] = 13
    dst[16] = 10
    dst[17] = 13
    dst[18] = 10
    dst[19] = 72
    dst[20] = 101
    dst[21] = 108
    dst[22] = 108
    dst[23] = 111
    return 24

func write_json_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    dst[1] = 84
    dst[2] = 84
    dst[3] = 80
    dst[4] = 47
    dst[5] = 49
    dst[6] = 46
    dst[7] = 49
    dst[8] = 32
    dst[9] = 50
    dst[10] = 48
    dst[11] = 48
    dst[12] = 32
    dst[13] = 79
    dst[14] = 75
    dst[15] = 13
    dst[16] = 10
    dst[17] = 13
    dst[18] = 10
    dst[19] = 123
    dst[20] = 125
    return 21

func main() -> Int
uses alloc, mem:
    var plain: []u8 = core.make_u8(192)
    var json_buf: []u8 = core.make_u8(192)
    var i: Int = 0
    var total: Int = 0
    while i < 1024:
        total = total + write_plaintext_response(plain)
        total = total + write_json_response(json_buf)
        i = i + 1
    if total > 0:
        return 0
    return 1
`)
	bounds := buildBoundsReport(irProg, checked, "linux-x64")
	if bounds.Totals.Removed != 45 || bounds.Totals.Left != 0 {
		t.Fatalf(
			"http helper-summary bounds totals = removed:%d left:%d, want removed:45 left:0; report=%+v",
			bounds.Totals.Removed,
			bounds.Totals.Left,
			bounds,
		)
	}
	rows := map[string]boundsFunctionRow{}
	for _, row := range bounds.Functions {
		rows[row.Function] = row
	}
	expected := map[string]int{
		"p25.http_plaintext_json.write_plaintext_response": 24,
		"p25.http_plaintext_json.write_json_response":      21,
	}
	proofProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("PLIR: %v", err)
	}
	if err := plir.VerifyProgram(proofProg); err != nil {
		t.Fatalf("PLIR verify: %v", err)
	}
	proof := buildProofReport(proofProg, bounds, "linux-x64")
	for fnName, wantRemoved := range expected {
		row := rows[fnName]
		if row.Function == "" || row.Removed != wantRemoved || row.Left != 0 ||
			len(row.Sites) != wantRemoved {
			t.Fatalf(
				"%s bounds row = %+v, want removed:%d left:0 sites:%d; rows=%+v",
				fnName,
				row,
				wantRemoved,
				wantRemoved,
				bounds.Functions,
			)
		}
		for _, site := range row.Sites {
			if !site.Removed ||
				site.Kind != "u8.store" ||
				!strings.HasPrefix(site.ProofID, "proof:helper-summary:") ||
				strings.HasPrefix(site.ProofID, "proof:helper-offset:") ||
				site.Reason != "removed_by_helper_summary_range" {
				t.Fatalf("%s bounds site = %+v, want removed helper-summary u8.store", fnName, site)
			}
			if !proofEvidenceDominates(proof.Proofs, site.ProofID, "bounds_check op") {
				t.Fatalf(
					"%s proof %q missing bounds_check dominance evidence: %+v",
					fnName,
					site.ProofID,
					proof.Proofs,
				)
			}
		}
	}
}

func TestP96BackendReportPromotesExactSliceSumMain(t *testing.T) {
	src := `module p25.slice_sum

func main() -> Int
uses alloc, mem:
    let n: Int = 4096
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    var r: Int = 0
    while r < 64:
        i = 0
        while i < n:
            total = total + xs[i]
            i = i + 1
        r = r + 1
    if total > 0:
        return 0
    return 1
`
	file, err := ParseFile([]byte(src), "slice_sum.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "p25.slice_sum",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"p25.slice_sum": file},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	irProg, err := lowerpkg.LowerWithOptions(checked, lowerpkg.Options{StackAllocationLowering: true})
	if err != nil {
		t.Fatalf("LowerWithOptions: %v", err)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "p25.slice_sum.main")
	if _, ok, err := machine.ScalarI32SliceSumLoopFunctionFromStackIR(mainFn); err != nil || ok {
		t.Fatalf(
			"helper slice-sum recognizer ok=%v err=%v for full main row, want strict fallback",
			ok,
			err,
		)
	}

	report := buildBackendReport("linux-x64", irProg)
	rows := backendRowsByFunction(report.Functions)
	mainRow := rows["p25.slice_sum.main"]
	if mainRow.BackendPath != "register" || mainRow.Category != "register_path" ||
		mainRow.Reason != "eligible_machine_ir_subset" {
		t.Fatalf(
			"slice_sum backend row = %+v, want register_path; instrs=%#v",
			mainRow,
			mainFn.Instrs,
		)
	}
	if mainRow.Detail != "machine-ir-slice-sum-main" {
		t.Fatalf(
			"slice_sum main detail = %q, want machine-ir-slice-sum-main; row=%+v instrs=%#v",
			mainRow.Detail,
			mainRow,
			mainFn.Instrs,
		)
	}
	if report.Summary.FunctionCount != 1 || report.Summary.RegisterPath != 1 ||
		report.Summary.StackFallback != 0 || report.Summary.Categories["register_path"] != 1 {
		t.Fatalf(
			"slice_sum backend summary = %+v, want one register_path and no fallback",
			report.Summary,
		)
	}
	machineRows := map[string]machineBackendFunctionReport{}
	for _, row := range report.MachineFunctions {
		machineRows[row.Function] = row
	}
	mainMachineRow, ok := machineRows["p25.slice_sum.main"]
	if !ok {
		t.Fatalf("machine report missing for p25.slice_sum.main: %+v", report.MachineFunctions)
	}
	if mainMachineRow.Path != "machine-ir-slice-sum-main" ||
		!mainMachineRow.SSAVerified ||
		mainMachineRow.Validation.StackChurnOps != 0 {
		t.Fatalf(
			"slice_sum main machine row = %+v, want verified no-stack-churn path",
			mainMachineRow,
		)
	}
	for _, want := range []string{"index_store", "index_load", "cmp", "add"} {
		if !containsReportString(mainMachineRow.InstructionSelection, want) {
			t.Fatalf(
				"slice_sum main instruction selection = %+v, want %q",
				mainMachineRow.InstructionSelection,
				want,
			)
		}
	}
}

func TestP101BackendReportPromotesExactMatrixMultiplyMain(t *testing.T) {
	src := `module p25.matrix_multiply

func main() -> Int
uses alloc, mem:
    var a: []i32 = core.make_i32(9)
    var b: []i32 = core.make_i32(9)
    var c: []i32 = core.make_i32(9)
    var i: Int = 0
    while i < 9:
        a[i] = i + 1
        b[i] = 9 - i
        c[i] = 0
        i = i + 1
    var checksum: Int = 0
    var r: Int = 0
    while r < 2000:
        var row: Int = 0
        while row < 3:
            var col: Int = 0
            while col < 3:
                var k: Int = 0
                var total: Int = 0
                while k < 3:
                    total = total + a[row * 3 + k] * b[k * 3 + col]
                    k = k + 1
                c[row * 3 + col] = total
                col = col + 1
            row = row + 1
        checksum = checksum + c[r % 9]
        r = r + 1
    if checksum > 0:
        return 0
    return 1
`
	file, err := ParseFile([]byte(src), "matrix_multiply.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "p25.matrix_multiply",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"p25.matrix_multiply": file},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	irProg, err := lowerpkg.LowerWithOptions(checked, lowerpkg.Options{StackAllocationLowering: true})
	if err != nil {
		t.Fatalf("LowerWithOptions: %v", err)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "p25.matrix_multiply.main")

	report := buildBackendReport("linux-x64", irProg)
	rows := backendRowsByFunction(report.Functions)
	mainRow := rows["p25.matrix_multiply.main"]
	if mainRow.BackendPath != "register" || mainRow.Category != "register_path" ||
		mainRow.Reason != "eligible_machine_ir_subset" {
		t.Fatalf(
			"matrix_multiply backend row = %+v, want register_path; instrs=%#v",
			mainRow,
			mainFn.Instrs,
		)
	}
	if mainRow.Detail != "machine-ir-matrix-multiply-main" {
		t.Fatalf(
			"matrix_multiply main detail = %q, want machine-ir-matrix-multiply-main; row=%+v instrs=%#v",
			mainRow.Detail,
			mainRow,
			mainFn.Instrs,
		)
	}
	if report.Summary.FunctionCount != 1 || report.Summary.RegisterPath != 1 ||
		report.Summary.StackFallback != 0 || report.Summary.Categories["register_path"] != 1 {
		t.Fatalf(
			"matrix_multiply backend summary = %+v, want one register_path and no fallback",
			report.Summary,
		)
	}
	machineRows := map[string]machineBackendFunctionReport{}
	for _, row := range report.MachineFunctions {
		machineRows[row.Function] = row
	}
	mainMachineRow, ok := machineRows["p25.matrix_multiply.main"]
	if !ok {
		t.Fatalf("machine report missing for p25.matrix_multiply.main: %+v", report.MachineFunctions)
	}
	if mainMachineRow.Path != "machine-ir-matrix-multiply-main" ||
		!mainMachineRow.SSAVerified ||
		mainMachineRow.Validation.StackChurnOps != 0 {
		t.Fatalf(
			"matrix_multiply main machine row = %+v, want verified no-stack-churn path",
			mainMachineRow,
		)
	}
	for _, want := range []string{"index_store", "index_load", "mul", "mod", "cmp", "add"} {
		if !containsReportString(mainMachineRow.InstructionSelection, want) {
			t.Fatalf(
				"matrix_multiply main instruction selection = %+v, want %q",
				mainMachineRow.InstructionSelection,
				want,
			)
		}
	}
}

func TestP120BackendReportPromotesHashLookupAndHashTableMain(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "p25", "hash_table.tetra")
	outPath := filepath.Join(tmp, "hash_table_tetra")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.WriteFile(srcPath, []byte(`
module p25.hash_table

func lookup(keys: []i32, values: []i32, n: Int, key: Int) -> Int
uses mem:
    var i: Int = 0
    while i < n:
        if keys[i] == key:
            return values[i]
        i = i + 1
    return 0

func main() -> Int
uses alloc, mem:
    let n: Int = 256
    var keys: []i32 = core.make_i32(n)
    var values: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        keys[i] = i * 2 + 1
        values[i] = i + 7
        i = i + 1
    var checksum: Int = 0
    var q: Int = 0
    while q < n:
        let key: Int = q * 2 + 1
        checksum = checksum + lookup(keys, values, n, key)
        q = q + 1
    if checksum > 0:
        return 0
    return 1
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{
		Runtime: RuntimeBuiltin,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}
	raw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	var report backendReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode backend report: %v\n%s", err, raw)
	}
	rows := backendRowsByFunction(report.Functions)
	lookup := rows["p25.hash_table.lookup"]
	main := rows["p25.hash_table.main"]
	assertBackendCoverageRow(t, lookup, "register", "register_path", "eligible_machine_ir_subset")
	if lookup.Detail != "machine-ir-hash-table-lookup" {
		t.Fatalf(
			"hash lookup detail = %q, want machine-ir-hash-table-lookup (rows=%+v)",
			lookup.Detail,
			report.Functions,
		)
	}
	assertBackendCoverageRow(t, main, "register", "register_path", "eligible_machine_ir_subset")
	if main.Detail != "machine-ir-hash-table-main" {
		t.Fatalf(
			"hash table main detail = %q, want machine-ir-hash-table-main (rows=%+v)",
			main.Detail,
			report.Functions,
		)
	}
	if main.Category == "unsupported_control_flow" || main.Reason == "unsupported_control_flow_uses_stack_fallback" {
		t.Fatalf("hash table main kept unsupported_control_flow fallback row: %+v", main)
	}
	if report.Summary.FunctionCount != 2 || report.Summary.RegisterPath != 2 ||
		report.Summary.StackFallback != 0 {
		t.Fatalf(
			"hash table backend summary = %+v, want lookup+main register and no fallback",
			report.Summary,
		)
	}
	if report.Summary.Categories["register_path"] != 2 ||
		report.Summary.Categories["unsupported_control_flow"] != 0 {
		t.Fatalf(
			"hash table backend categories = %+v, want register_path=2 and unsupported_control_flow=0",
			report.Summary.Categories,
		)
	}
}

func TestP120WindowsBackendReportDoesNotPromoteHashTableMainWithoutEmitterSupport(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "p25", "hash_table.tetra")
	outPath := filepath.Join(tmp, "hash_table_tetra_windows")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.WriteFile(srcPath, []byte(`
module p25.hash_table

func lookup(keys: []i32, values: []i32, n: Int, key: Int) -> Int
uses mem:
    var i: Int = 0
    while i < n:
        if keys[i] == key:
            return values[i]
        i = i + 1
    return 0

func main() -> Int
uses alloc, mem:
    let n: Int = 256
    var keys: []i32 = core.make_i32(n)
    var values: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        keys[i] = i * 2 + 1
        values[i] = i + 7
        i = i + 1
    var checksum: Int = 0
    var q: Int = 0
    while q < n:
        let key: Int = q * 2 + 1
        checksum = checksum + lookup(keys, values, n, key)
        q = q + 1
    if checksum > 0:
        return 0
    return 1
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "windows-x64", BuildOptions{
		Runtime: RuntimeBuiltin,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt windows-x64: %v", err)
	}
	raw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	var report backendReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode backend report: %v\n%s", err, raw)
	}
	main := backendRowsByFunction(report.Functions)["p25.hash_table.main"]
	if main.BackendPath == "register" || main.Detail == "machine-ir-hash-table-main" {
		t.Fatalf(
			"windows-x64 hash table main backend row = %+v, want no machine-ir-hash-table-main report without emitter support",
			main,
		)
	}
}

func scalarCorpusFunc(name string, op ir.IRInstrKind) ir.IRFunc {
	return ir.IRFunc{
		Name:        name,
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: op},
			{Kind: ir.IRReturn},
		},
	}
}

func allocationLoopBenchmarkIRFuncForReport() ir.IRFunc {
	return ir.IRFunc{
		Name:        "p25.allocation.main",
		ExportName:  "main",
		LocalSlots:  20,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRLabel, Label: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1024},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 1},
			{Kind: ir.IRConstI32, Imm: 32},
			{Kind: ir.IRStackSliceI32, Local: 4, ArgSlots: 16, Imm: 32, Name: "xs"},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRIndexStoreI32},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexLoadI32},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 0},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRCmpGtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func backendRowsByFunction(rows []backendFunctionPathReport) map[string]backendFunctionPathReport {
	out := make(map[string]backendFunctionPathReport, len(rows))
	for _, row := range rows {
		out[row.Function] = row
	}
	return out
}

func assertBackendCoverageRow(
	t *testing.T,
	row backendFunctionPathReport,
	path string,
	category string,
	reason string,
) {
	t.Helper()
	if row.BackendPath != path || row.Category != category || row.Reason != reason {
		t.Fatalf(
			"backend row for %s = %+v, want path=%q category=%q reason=%q",
			row.Function,
			row,
			path,
			category,
			reason,
		)
	}
}

func assertRuntimeFeatureEvidenceMarker(t *testing.T, class string, method string) {
	t.Helper()
	if class != "lowered_ir_static_plan" || method != "backend_report_lowered_ir_scan_v1" {
		t.Fatalf(
			("runtime feature evidence marker = (%q, %q), want " +
				"lowered_ir_static_plan/backend_report_lowered_ir_scan_v1"),
			class,
			method,
		)
	}
}

func assertRuntimeObjectEvidenceMarker(t *testing.T, plan backendRuntimeObjectPlan) {
	t.Helper()
	if plan.EvidenceClass != "native_runtime_object_plan" ||
		plan.EvidenceMethod != "native_link_runtime_object_plan_v1" {
		t.Fatalf(
			("runtime object evidence marker = (%q, %q), want " +
				"native_runtime_object_plan/native_link_runtime_object_plan_v" +
				"1"),
			plan.EvidenceClass,
			plan.EvidenceMethod,
		)
	}
}

func assertStringSliceEqual(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", label, got, want)
	}
}

func checkedAndLoweredProgram(t *testing.T, src string) (*CheckedProgram, *IRProgram) {
	t.Helper()
	prog, err := Parse([]byte(src))
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
	return checked, irProg
}

func checkedAndLoweredFileProgram(t *testing.T, src string) (*CheckedProgram, *IRProgram) {
	t.Helper()
	file, err := ParseFile([]byte(src), "p25/hash_table.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: file.Module,
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{file.Module: file},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	return checked, irProg
}

func findBoundsSiteByKind(t *testing.T, report boundsReport, kind string) boundsCheckSite {
	t.Helper()
	for _, fn := range report.Functions {
		for _, site := range fn.Sites {
			if site.Kind == kind {
				return site
			}
		}
	}
	t.Fatalf("missing bounds site kind %q in %+v", kind, report)
	return boundsCheckSite{}
}

func proofEvidenceDominates(proofs []proofEvidence, proofID string, dominatesPrefix string) bool {
	for _, evidence := range proofs {
		if (evidence.ProofID == proofID || strings.HasPrefix(evidence.ProofID, proofID+":")) &&
			evidence.RemovedBoundsCheck &&
			strings.HasPrefix(evidence.Dominates, dominatesPrefix) {
			return true
		}
	}
	return false
}

func containsReportString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestBuildOptionsExposeNoBackendSemanticMode(t *testing.T) {
	buildOptionsType := reflect.TypeOf(BuildOptions{})
	for i := 0; i < buildOptionsType.NumField(); i++ {
		fieldName := strings.ToLower(buildOptionsType.Field(i).Name)
		for _, forbidden := range []string{
			"backend",
			"machine",
			"register",
			"pgo",
			"profile",
			"lto",
			"targetcpu",
			"target_cpu",
			"targetfeature",
		} {
			if strings.Contains(fieldName, forbidden) {
				t.Fatalf(
					("BuildOptions exposes semantic tuning field %q; " +
						"backend/profile/LTO/target-cpu selection must remain " +
						"internal or evidence-only"),
					buildOptionsType.Field(i).Name,
				)
			}
		}
	}
	if nativeCodegenOptions(BuildOptions{}).DisableMachinePaths {
		t.Fatalf(
			"native codegen options should not set DisableMachinePaths from public BuildOptions",
		)
	}
}

func TestNativeCodegenOptionsCarryRuntimeHeapTelemetry(t *testing.T) {
	got := nativeCodegenOptions(BuildOptions{
		EmitRuntimeHeapTelemetry:         true,
		RuntimeHeapTelemetryActorDomains: true,
		RuntimeHeapTelemetryDir:          "reports/heap",
	})
	if !got.EmitRuntimeHeapTelemetry || got.RuntimeHeapTelemetryDir != "reports/heap" {
		t.Fatalf(
			"native codegen telemetry options = enabled:%v dir:%q",
			got.EmitRuntimeHeapTelemetry,
			got.RuntimeHeapTelemetryDir,
		)
	}
	if !got.RuntimeHeapTelemetryActorDomains {
		t.Fatalf("native codegen telemetry actor-domain option was not preserved")
	}
}

func TestPerformanceReportIncludesBlockerDiagnostics(t *testing.T) {
	report := buildPerformanceReport("linux-x64")
	got := map[string]bool{}
	for _, blocker := range report.Blockers {
		got[blocker.Message] = true
		if blocker.Code == "" || blocker.Component == "" || blocker.Evidence == "" ||
			blocker.CostClass == "" {
			t.Fatalf("incomplete blocker row: %+v", blocker)
		}
	}
	for _, want := range []string{
		"left bounds check: missing dominance",
		"heap allocation: escapes through return",
		"heap allocation: unknown call",
		"heap allocation: local call boundary heap fallback",
		"not vectorized: no noalias proof",
		"not inlined: code-size budget",
		"register spill: live range pressure",
		"stack fallback: unsupported aggregate return",
		"actor copy: borrowed data crosses boundary",
	} {
		if !got[want] {
			t.Fatalf("performance report missing blocker %q: %+v", want, report.Blockers)
		}
	}
	if len(report.Claims) == 0 ||
		strings.Contains(strings.ToLower(strings.Join(report.Claims, " ")), "fastest language") {
		t.Fatalf("performance claims are not claim-disciplined: %+v", report.Claims)
	}
}

func TestPerformanceReportCoversP20BenchmarkBlockers(t *testing.T) {
	report := buildPerformanceReport("linux-x64")
	if report.SchemaVersion != 3 || report.MatrixScope != "p20.0_benchmark_matrix" {
		t.Fatalf(
			"performance report schema/scope = %d/%q, want P20.1 schema over P20.0 matrix",
			report.SchemaVersion,
			report.MatrixScope,
		)
	}
	if report.MatrixReport != ("reports/benchmark-matrix-hardening-v1/benchmarks/p20-matrix-" +
		"hardening-report.json") {
		t.Fatalf("matrix report path = %q", report.MatrixReport)
	}
	gotReasons := map[string]performanceBlockerRow{}
	for _, blocker := range report.Blockers {
		gotReasons[blocker.Code] = blocker
		if blocker.Message == "" || blocker.Evidence == "" || blocker.NextStep == "" ||
			blocker.CostClass == "" {
			t.Fatalf("incomplete blocker row: %+v", blocker)
		}
	}
	for code, want := range map[string]struct {
		message   string
		costClass string
	}{
		"bounds.missing_dominance": {
			message:   "left bounds check: missing dominance",
			costClass: "dynamic_check_required",
		},
		"allocation.return_escape": {
			message:   "heap allocation: escapes through return",
			costClass: "conservative_fallback",
		},
		"allocation.unknown_call": {
			message:   "heap allocation: unknown call",
			costClass: "conservative_fallback",
		},
		"allocation.local_call_heap_fallback": {
			message:   "heap allocation: local call boundary heap fallback",
			costClass: "conservative_fallback",
		},
		"vector.no_noalias_proof": {
			message:   "not vectorized: no noalias proof",
			costClass: "dynamic_check_required",
		},
		"inline.code_size_budget": {
			message:   "not inlined: code-size budget",
			costClass: "instrumentation_only",
		},
		"register_spill.live_range_pressure": {
			message:   "register spill: live range pressure",
			costClass: "instrumentation_only",
		},
		"stack_fallback.unsupported_aggregate_return": {
			message:   "stack fallback: unsupported aggregate return",
			costClass: "conservative_fallback",
		},
		"actor_copy.borrowed_data_boundary": {
			message:   "actor copy: borrowed data crosses boundary",
			costClass: "conservative_fallback",
		},
	} {
		row, ok := gotReasons[code]
		if !ok {
			t.Fatalf("performance report missing P20.1 blocker code %q: %+v", code, report.Blockers)
		}
		if row.Message != want.message || row.CostClass != want.costClass {
			t.Fatalf(
				"blocker %s = %+v, want message=%q cost_class=%q",
				code,
				row,
				want.message,
				want.costClass,
			)
		}
	}
	gotBenchmarks := map[string]performanceBenchmarkExplanation{}
	for _, row := range report.Benchmarks {
		gotBenchmarks[row.Benchmark] = row
		if row.Category == "" || row.Explanation == "" || row.NextStep == "" {
			t.Fatalf("incomplete benchmark explanation row: %+v", row)
		}
		if row.MatrixScope != report.MatrixScope || row.MatrixReport != report.MatrixReport {
			t.Fatalf(
				"benchmark row %s matrix linkage = %q/%q",
				row.Benchmark,
				row.MatrixScope,
				row.MatrixReport,
			)
		}
		if len(row.ReasonCodes) == 0 {
			t.Fatalf("benchmark row %s missing reason codes", row.Benchmark)
		}
		for _, code := range row.ReasonCodes {
			if _, ok := gotReasons[code]; !ok {
				t.Fatalf("benchmark row %s cites unknown reason code %q", row.Benchmark, code)
			}
		}
	}
	for _, want := range []string{
		"integer_loops_tetra",
		"slice_sum_tetra",
		"bounds_check_loops_tetra",
		"function_calls_tetra",
		"recursion_tetra",
		"matrix_multiply_tetra",
		"hash_table_tetra",
		"allocation_tetra",
		"region_island_allocation_tetra",
		"json_parse_stringify_tetra",
		"http_plaintext_json_tetra",
		"postgresql_single_multiple_update_tetra",
		"actor_ping_pong_tetra",
		"parallel_map_reduce_tetra",
		"startup_time_tetra",
		"binary_size_tetra",
		"compile_time_tetra",
	} {
		if _, ok := gotBenchmarks[want]; !ok {
			t.Fatalf("performance report missing P20.0 benchmark explanation %q", want)
		}
	}
	if err := ValidatePerformanceBlockerReport(report); err != nil {
		t.Fatalf("ValidatePerformanceBlockerReport: %v", err)
	}
	hash := gotBenchmarks["hash_table_tetra"]
	if !containsReasonCode(hash.ReasonCodes, "allocation.local_call_heap_fallback") ||
		containsReasonCode(hash.ReasonCodes, "allocation.unknown_call") {
		t.Fatalf(
			("hash_table_tetra reason codes = %+v, want local call heap " +
				"fallback without unknown-call blocker"),
			hash.ReasonCodes,
		)
	}
}

func TestValidatePerformanceBlockerReportRejectsWeakP20Evidence(t *testing.T) {
	report := buildPerformanceReport("linux-x64")
	report.Blockers = report.Blockers[:len(report.Blockers)-1]
	err := ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "actor_copy.borrowed_data_boundary") {
		t.Fatalf("accepted report missing actor-copy blocker: %v", err)
	}

	report = buildPerformanceReport("linux-x64")
	report.Benchmarks = report.Benchmarks[:len(report.Benchmarks)-1]
	err = ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "compile_time_tetra") {
		t.Fatalf("accepted report missing compile-time benchmark explanation: %v", err)
	}

	report = buildPerformanceReport("linux-x64")
	report.Benchmarks[0].ReasonCodes = []string{"unknown.reason"}
	err = ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "unknown reason code") {
		t.Fatalf("accepted unknown benchmark reason code: %v", err)
	}

	report = buildPerformanceReport("linux-x64")
	report.Benchmarks[0].Explanation = "TODO"
	err = ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("accepted placeholder explanation: %v", err)
	}

	report = buildPerformanceReport("linux-x64")
	report.Claims = []string{"This proves C++/Rust parity and measured speed superiority."}
	err = ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "C++/Rust parity") {
		t.Fatalf("accepted fake performance claim: %v", err)
	}

	report = buildPerformanceReport("linux-x64")
	report.Blockers[0].CostClass = "mystery_cost"
	err = ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "unknown cost_class") {
		t.Fatalf("accepted unknown cost class: %v", err)
	}

	report = buildPerformanceReport("linux-x64")
	report.Claims = append(
		report.Claims,
		"dynamic_check_required rows prove zero-cost bounds_check_eliminated",
	)
	err = ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "dynamic_check_required") {
		t.Fatalf("accepted fake dynamic zero-cost claim: %v", err)
	}

	report = buildPerformanceReport("linux-x64")
	report.Claims = append(report.Claims, "unsafe_unknown is optimized as trusted storage")
	err = ValidatePerformanceBlockerReport(report)
	if err == nil || !strings.Contains(err.Error(), "unsafe_unknown") {
		t.Fatalf("accepted trusted unsafe_unknown claim: %v", err)
	}
}

func containsReasonCode(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// ---- runtime_hardening_v1_test.go ----

func TestP24RuntimeHardeningV1CoversMasterPlanTargets(t *testing.T) {
	report, err := BuildP24RuntimeHardeningV1Report()
	if err != nil {
		t.Fatalf("BuildP24RuntimeHardeningV1Report: %v", err)
	}
	if report.SchemaVersion != runtimeHardeningV1Schema {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, runtimeHardeningV1Schema)
	}
	if report.Scope != runtimeHardeningV1ScopeP241 {
		t.Fatalf("scope = %q, want %q", report.Scope, runtimeHardeningV1ScopeP241)
	}
	if err := ValidateP24RuntimeHardeningV1Report(report); err != nil {
		t.Fatalf("ValidateP24RuntimeHardeningV1Report: %v", err)
	}

	rows := map[RuntimeHardeningV1ID]RuntimeHardeningV1Row{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 ||
			len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p24RuntimeHardeningV1IDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p24AssertRuntimeHardeningRow(
		t,
		rows[RuntimeHardeningDeterministicTraps],
		[]string{"trap_or_stable_status", "emitWasmTrapIf", "tetra panic"},
	)
	p24AssertRuntimeHardeningRow(
		t,
		rows[RuntimeHardeningOOMPolicy],
		[]string{"AllocationFailureTrapOrStatus", "reject before allocator", "stable trap/status"},
	)
	p24AssertRuntimeHardeningRow(
		t,
		rows[RuntimeHardeningStackOverflowGuard],
		[]string{"stack-depth consistency", "guard-page", "recursion-depth"},
	)
	p24AssertRuntimeHardeningRow(
		t,
		rows[RuntimeHardeningIntegerOverflowSemantics],
		[]string{"checkedNegI32", "foldConstBinaryI32", "byte-size overflow"},
	)
	p24AssertRuntimeHardeningRow(
		t,
		rows[RuntimeHardeningAllocatorCorruptionInstrumentation],
		[]string{"bounds_header", "stale or double free", "PerCoreSmallHeapAllocator"},
	)
	p24AssertRuntimeHardeningRow(
		t,
		rows[RuntimeHardeningRegionUseAfterFreeInstrumentation],
		[]string{"AllocationDebugDoubleFree", "AllocationDebugUseAfterFree", "region.temp"},
	)
	p24AssertRuntimeHardeningRow(
		t,
		rows[RuntimeHardeningActorMailboxOverflowPolicy],
		[]string{
			"ErrMailboxFull",
			"blocking_recv_yield",
			"message pool exhaustion returns checked -1",
			"drained message pool entries are reclaimed",
		},
	)
	p24AssertRuntimeHardeningRow(
		t,
		rows[RuntimeHardeningNetworkParserLimits],
		[]string{"ErrHeaderTooLarge", "ErrBodyTooLarge", "ErrFrameTooLarge", "ErrMalformedFrame"},
	)

	if !report.DeterministicTrapsReviewed || !report.OOMPolicyReviewed ||
		!report.StackOverflowGuardReviewed ||
		!report.IntegerOverflowSemanticsAudited ||
		!report.AllocatorCorruptionReviewed ||
		!report.RegionLifetimeReviewed ||
		!report.ActorMailboxOverflowPolicyReviewed ||
		!report.NetworkParserLimitsReviewed {
		t.Fatalf("runtime hardening flags missing: %#v", report)
	}
	for _, nonClaim := range []string{
		"full runtime-hardening proof is not claimed",
		"full stack-overflow protection is not claimed",
		"OOM recovery guarantee is not claimed",
		"full allocator-corruption detection proof is not claimed",
		"production actor-mailbox promotion is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p24RuntimeHardeningHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP24RuntimeHardeningV1RejectsFakeClaimsAndWeakEvidence(t *testing.T) {
	base, err := BuildP24RuntimeHardeningV1Report()
	if err != nil {
		t.Fatalf("BuildP24RuntimeHardeningV1Report: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*RuntimeHardeningV1Report)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "missing deterministic traps",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.DeterministicTrapsReviewed = false
			},
			want: "deterministic traps",
		},
		{
			name: "missing actor mailbox policy",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.ActorMailboxOverflowPolicyReviewed = false
			},
			want: "actor mailbox",
		},
		{
			name: "fake full hardening",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.FullRuntimeHardeningClaimed = true
			},
			want: "full runtime-hardening",
		},
		{
			name: "fake stack overflow protection",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.FullStackOverflowProtectionClaimed = true
			},
			want: "stack-overflow protection",
		},
		{
			name: "fake OOM recovery",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.FullOOMRecoveryClaimed = true
			},
			want: "OOM recovery",
		},
		{
			name: "fake allocator corruption detection",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.FullAllocatorCorruptionDetectionClaimed = true
			},
			want: "allocator-corruption",
		},
		{
			name: "fake production actor mailbox",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.ProductionActorMailboxClaimed = true
			},
			want: "production actor-mailbox",
		},
		{
			name: "runtime behavior change",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics change",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.SafeSemanticsChanged = true
			},
			want: "safe semantics",
		},
		{
			name: "performance claim",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := base
			report.Rows = append([]RuntimeHardeningV1Row(nil), base.Rows...)
			report.Witnesses = append([]RuntimeHardeningV1Witness(nil), base.Witnesses...)
			report.NonClaims = append([]string(nil), base.NonClaims...)
			tc.mutate(&report)
			err := ValidateP24RuntimeHardeningV1Report(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateP24RuntimeHardeningV1Report error = %v, want %q", err, tc.want)
			}
		})
	}
}

func p24AssertRuntimeHardeningRow(t *testing.T, row RuntimeHardeningV1Row, wants []string) {
	t.Helper()
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}

// ---- runtime_heap_telemetry_test.go ----

func TestLinuxX64RuntimeHeapTelemetrySidecarFromCompiledBinary(t *testing.T) {
	sidecar := buildRunReadHeapTelemetrySidecar(t, `func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return 0
`, "heap-smoke")
	requireHeapTelemetrySidecarIdentity(t, sidecar, "heap-smoke")
	if _, ok := sidecar["heap_allocation_count"].(float64); !ok {
		t.Fatalf(
			"sidecar heap_allocation_count = %#v, want numeric",
			sidecar["heap_allocation_count"],
		)
	}
	if _, ok := sidecar["heap_peak_bytes"].(float64); !ok {
		t.Fatalf("sidecar heap_peak_bytes = %#v, want numeric", sidecar["heap_peak_bytes"])
	}
	requireNoActorHeapTelemetryDomains(t, sidecar)
}

func TestLinuxX64RuntimeHeapTelemetryReportsZeroForStackMakeSlice(t *testing.T) {
	sidecar := buildRunReadHeapTelemetrySidecar(t, `func main() -> Int
uses alloc, mem:
    let n: Int = 4
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i + 1
        i = i + 1
    if xs[0] == 1:
        return 0
    return 1
`, "heap-stack-smoke")
	requireHeapTelemetrySidecarIdentity(t, sidecar, "heap-stack-smoke")
	if got, ok := sidecar["heap_allocation_count"].(float64); !ok || got != 0 {
		t.Fatalf(
			"sidecar heap_allocation_count = %#v, want 0 for stack-backed allocation",
			sidecar["heap_allocation_count"],
		)
	}
	if got, ok := sidecar["heap_peak_bytes"].(float64); !ok || got != 0 {
		t.Fatalf(
			"sidecar heap_peak_bytes = %#v, want 0 for stack-backed allocation",
			sidecar["heap_peak_bytes"],
		)
	}
	requireNoActorHeapTelemetryDomains(t, sidecar)
}

func TestLinuxX64RuntimeHeapTelemetryActorDomainsFromBuiltinRuntime(t *testing.T) {
	sidecar := buildRunReadHeapTelemetrySidecarWithOptions(t, `func pong() -> i32
uses actors:
    var v: i32 = core.recv()
    if v == 41:
        var _sent: i32 = core.send(core.sender(), 42)
        return 0
    return 1

func main() -> i32
uses actors:
    var p: actor = core.spawn("pong")
    var _sent: i32 = core.send(p, 41)
    var r: i32 = core.recv()
    if r == 42:
        return 0
    return 1
`, "heap-actor-smoke", BuildOptions{Runtime: RuntimeBuiltin})
	requireHeapTelemetrySidecarIdentity(t, sidecar, "heap-actor-smoke")
	requireActorHeapTelemetryDomainWithBytes(t, sidecar)
}

func TestLinuxX64RuntimeHeapTelemetryActorStackTrimReleasesExcessDoneStacks(t *testing.T) {
	src := strings.Join([]string{
		"func done() -> i32:",
		"    return 0",
		"",
		"func main() -> i32",
		"uses actors:",
		"    let _a00: actor = core.spawn(\"done\")",
		"    let _a01: actor = core.spawn(\"done\")",
		"    let _a02: actor = core.spawn(\"done\")",
		"    let _a03: actor = core.spawn(\"done\")",
		"    let _a04: actor = core.spawn(\"done\")",
		"    let _a05: actor = core.spawn(\"done\")",
		"    let _a06: actor = core.spawn(\"done\")",
		"    let _a07: actor = core.spawn(\"done\")",
		"    let _a08: actor = core.spawn(\"done\")",
		"    let _a09: actor = core.spawn(\"done\")",
		"    let _a10: actor = core.spawn(\"done\")",
		"    let _a11: actor = core.spawn(\"done\")",
		"    let _a12: actor = core.spawn(\"done\")",
		"    let _a13: actor = core.spawn(\"done\")",
		"    let _a14: actor = core.spawn(\"done\")",
		"    let _a15: actor = core.spawn(\"done\")",
		"    let _w00: actor.wait_result = core.actor_wait(_a00)",
		"    let _w01: actor.wait_result = core.actor_wait(_a01)",
		"    let _w02: actor.wait_result = core.actor_wait(_a02)",
		"    let _w03: actor.wait_result = core.actor_wait(_a03)",
		"    let _w04: actor.wait_result = core.actor_wait(_a04)",
		"    let _w05: actor.wait_result = core.actor_wait(_a05)",
		"    let _w06: actor.wait_result = core.actor_wait(_a06)",
		"    let _w07: actor.wait_result = core.actor_wait(_a07)",
		"    let _w08: actor.wait_result = core.actor_wait(_a08)",
		"    let _w09: actor.wait_result = core.actor_wait(_a09)",
		"    let _w10: actor.wait_result = core.actor_wait(_a10)",
		"    let _w11: actor.wait_result = core.actor_wait(_a11)",
		"    let _w12: actor.wait_result = core.actor_wait(_a12)",
		"    let _w13: actor.wait_result = core.actor_wait(_a13)",
		"    let _w14: actor.wait_result = core.actor_wait(_a14)",
		"    let _w15: actor.wait_result = core.actor_wait(_a15)",
		"    let _b00: actor = core.spawn(\"done\")",
		"    let _b01: actor = core.spawn(\"done\")",
		"    let _b02: actor = core.spawn(\"done\")",
		"    let _b03: actor = core.spawn(\"done\")",
		"    let _b04: actor = core.spawn(\"done\")",
		"    let _b05: actor = core.spawn(\"done\")",
		"    let _b06: actor = core.spawn(\"done\")",
		"    let _b07: actor = core.spawn(\"done\")",
		"    let _b08: actor = core.spawn(\"done\")",
		"    let _b09: actor = core.spawn(\"done\")",
		"    let _b10: actor = core.spawn(\"done\")",
		"    let _b11: actor = core.spawn(\"done\")",
		"    let _b12: actor = core.spawn(\"done\")",
		"    let _b13: actor = core.spawn(\"done\")",
		"    let _b14: actor = core.spawn(\"done\")",
		"    let _b15: actor = core.spawn(\"done\")",
		"    let _bw00: actor.wait_result = core.actor_wait(_b00)",
		"    let _bw01: actor.wait_result = core.actor_wait(_b01)",
		"    let _bw02: actor.wait_result = core.actor_wait(_b02)",
		"    let _bw03: actor.wait_result = core.actor_wait(_b03)",
		"    let _bw04: actor.wait_result = core.actor_wait(_b04)",
		"    let _bw05: actor.wait_result = core.actor_wait(_b05)",
		"    let _bw06: actor.wait_result = core.actor_wait(_b06)",
		"    let _bw07: actor.wait_result = core.actor_wait(_b07)",
		"    let _bw08: actor.wait_result = core.actor_wait(_b08)",
		"    let _bw09: actor.wait_result = core.actor_wait(_b09)",
		"    let _bw10: actor.wait_result = core.actor_wait(_b10)",
		"    let _bw11: actor.wait_result = core.actor_wait(_b11)",
		"    let _bw12: actor.wait_result = core.actor_wait(_b12)",
		"    let _bw13: actor.wait_result = core.actor_wait(_b13)",
		"    let _bw14: actor.wait_result = core.actor_wait(_b14)",
		"    let _bw15: actor.wait_result = core.actor_wait(_b15)",
		"    return 0",
		"",
	}, "\n")
	sidecar := buildRunReadHeapTelemetrySidecarWithOptions(t, src, "heap-actor-stack-trim-smoke", BuildOptions{Runtime: RuntimeBuiltin})
	requireHeapTelemetrySidecarIdentity(t, sidecar, "heap-actor-stack-trim-smoke")
	requireActorHeapTelemetryStackTrimmed(t, sidecar)
	requireActorHeapTelemetryLiveCountSeparatesDoneSlots(t, sidecar)
}

func buildRunReadHeapTelemetrySidecar(t *testing.T, src string, outputName string) map[string]any {
	t.Helper()
	return buildRunReadHeapTelemetrySidecarWithOptions(t, src, outputName, BuildOptions{})
}

func buildRunReadHeapTelemetrySidecarWithOptions(
	t *testing.T,
	src string,
	outputName string,
	opt BuildOptions,
) map[string]any {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux-x64 runtime heap telemetry smoke requires linux/amd64 host")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	outPath := filepath.Join(dir, outputName)
	telemetryDir := filepath.Join(dir, "heap-telemetry")
	if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	opt.Jobs = 1
	opt.EmitRuntimeHeapTelemetry = true
	opt.RuntimeHeapTelemetryDir = telemetryDir
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", opt); err != nil {
		t.Fatalf("BuildFileWithStatsOpt telemetry: %v", err)
	}
	if out, err := exec.Command(outPath).CombinedOutput(); err != nil {
		t.Fatalf("run telemetry binary: %v\n%s", err, string(out))
	}
	matches, err := filepath.Glob(filepath.Join(telemetryDir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("heap telemetry sidecars = %d (%v), want 1", len(matches), matches)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	var sidecar map[string]any
	if err := json.Unmarshal(raw, &sidecar); err != nil {
		t.Fatalf("sidecar JSON: %v\n%s", err, string(raw))
	}
	return sidecar
}

func requireHeapTelemetrySidecarIdentity(t *testing.T, sidecar map[string]any, program string) {
	t.Helper()
	if sidecar["schema"] != "tetra.runtime.heap_telemetry.v1" {
		t.Fatalf("sidecar schema = %#v", sidecar["schema"])
	}
	if sidecar["method"] != "tetra_linux_x64_heap_telemetry_v1" {
		t.Fatalf("sidecar method = %#v", sidecar["method"])
	}
	if sidecar["program"] != program {
		t.Fatalf("sidecar program = %#v, want %q", sidecar["program"], program)
	}
}

func requireNoActorHeapTelemetryDomains(t *testing.T, sidecar map[string]any) {
	t.Helper()
	for _, domain := range heapTelemetryDomains(t, sidecar) {
		if domain["kind"] == "actor" ||
			strings.HasPrefix(stringValue(domain["domain_id"]), "domain:actor:") {
			t.Fatalf(
				"non-actor heap telemetry sidecar unexpectedly contains actor domain: %#v",
				domain,
			)
		}
	}
}

func requireActorHeapTelemetryDomainWithBytes(t *testing.T, sidecar map[string]any) {
	t.Helper()
	for _, domain := range heapTelemetryDomains(t, sidecar) {
		if domain["kind"] != "actor" ||
			!strings.HasPrefix(stringValue(domain["domain_id"]), "domain:actor:") {
			continue
		}
		peak := numberValue(domain["peak_bytes"])
		copied := numberValue(domain["bytes_copied"])
		stackLive, hasStackLive := domain["stack_live_bytes"]
		stackReserved, hasStackReserved := domain["stack_reserved_bytes"]
		stackRetained, hasStackRetained := domain["stack_retained_bytes"]
		stackReleased, hasStackReleased := domain["stack_released_bytes"]
		if !hasStackLive || !hasStackReserved || !hasStackRetained || !hasStackReleased {
			t.Fatalf("actor heap telemetry domain missing stack byte fields: %#v", domain)
		}
		current := numberValue(domain["current_bytes"])
		mailboxCurrent := numberValue(domain["mailbox_current_bytes"])
		stackLiveBytes := numberValue(stackLive)
		if current < mailboxCurrent+stackLiveBytes {
			t.Fatalf(
				"actor heap telemetry current_bytes = %v below mailbox_current_bytes + stack_live_bytes in %#v",
				current,
				domain,
			)
		}
		if numberValue(stackReserved)+numberValue(stackRetained)+numberValue(stackReleased) > 0 &&
			peak > 0 && copied > 0 {
			return
		}
	}
	t.Fatalf(
		"heap telemetry sidecar missing actor domain with nonzero peak_bytes and bytes_copied: %#v",
		sidecar["domain_bytes"],
	)
}

func requireActorHeapTelemetryStackTrimmed(t *testing.T, sidecar map[string]any) {
	t.Helper()
	const stackSizeBytes = 64 * 1024
	const warmDoneStacks = 8
	var retained float64
	var released float64
	var actorDomains int
	for _, domain := range heapTelemetryDomains(t, sidecar) {
		if domain["kind"] != "actor" ||
			!strings.HasPrefix(stringValue(domain["domain_id"]), "domain:actor:") {
			continue
		}
		actorDomains++
		retained += numberValue(domain["stack_retained_bytes"])
		released += numberValue(domain["stack_released_bytes"])
	}
	if actorDomains < warmDoneStacks+2 {
		t.Fatalf("actor domains = %d, want a completed actor wave in %#v", actorDomains, sidecar)
	}
	if released < stackSizeBytes {
		t.Fatalf("stack_released_bytes total = %v, want at least one released actor stack", released)
	}
	if retained > warmDoneStacks*stackSizeBytes {
		t.Fatalf(
			"stack_retained_bytes total = %v, want bounded warm pool <= %d",
			retained,
			warmDoneStacks*stackSizeBytes,
		)
	}
}

func requireActorHeapTelemetryLiveCountSeparatesDoneSlots(t *testing.T, sidecar map[string]any) {
	t.Helper()
	recordCount, hasRecordCount := sidecar["actor_snapshot_record_count"].(float64)
	liveCount, hasLiveCount := sidecar["actor_live_count"].(float64)
	if !hasRecordCount || !hasLiveCount {
		t.Fatalf(
			"actor heap telemetry missing actor_snapshot_record_count/actor_live_count: %#v",
			sidecar,
		)
	}
	if recordCount <= liveCount {
		t.Fatalf(
			"actor_snapshot_record_count = %v, actor_live_count = %v; want reusable/done slots excluded from live count",
			recordCount,
			liveCount,
		)
	}
}

func heapTelemetryDomains(t *testing.T, sidecar map[string]any) []map[string]any {
	t.Helper()
	rawDomains, ok := sidecar["domain_bytes"].([]any)
	if !ok {
		t.Fatalf("sidecar domain_bytes = %#v, want array", sidecar["domain_bytes"])
	}
	domains := make([]map[string]any, 0, len(rawDomains))
	for _, raw := range rawDomains {
		domain, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("sidecar domain entry = %#v, want object", raw)
		}
		domains = append(domains, domain)
	}
	return domains
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func numberValue(v any) float64 {
	n, _ := v.(float64)
	return n
}

// ---- runtime_override_test.go ----

func TestRuntimeObjectOverrideActorsPingPong(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors", "actors_pingpong.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	world, err := LoadWorld(srcPath)
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	actorsUsed, actorEntries, _, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collect actor entries: %v", err)
	}
	if !actorsUsed || len(actorEntries) == 0 {
		t.Fatalf("expected actors usage")
	}

	var rt *Object
	switch tgt.Triple {
	case "linux-x64":
		rt, err = actorsrt.BuildLinuxX64(actorEntries)
	case "macos-x64":
		rt, err = actorsrt.BuildMacOSX64(actorEntries)
	case "windows-x64":
		rt, err = actorsrt.BuildWindowsX64(actorEntries)
	default:
		t.Fatalf("unsupported target: %s", tgt.Triple)
	}
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	rt.Target = tgt.Triple
	rt.Module = "__runtime"

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime.tobj")
	if err := WriteObject(rtPath, rt); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{RuntimeObjectPath: rtPath},
	); err != nil {
		t.Fatalf("build with runtime override: %v", err)
	}
	stdout, exitCode := runBinary(t, outPath)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestRuntimeObjectOverrideRelinksWhenRuntimeObjectChanges(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors", "actors_pingpong.tetra")
	world, err := LoadWorld(srcPath)
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	actorsUsed, actorEntries, _, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collect actor entries: %v", err)
	}
	if !actorsUsed || len(actorEntries) == 0 {
		t.Fatalf("expected actors usage")
	}

	rt, err := buildHostRuntimeObject(tgt.Triple, actorEntries)
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime.tobj")
	rt.Target = tgt.Triple
	rt.Module = "__runtime"
	if err := WriteObject(rtPath, rt); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	stats1, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{RuntimeObjectPath: rtPath},
	)
	if err != nil {
		t.Fatalf("build1 with runtime override: %v", err)
	}
	first, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read build1 output: %v", err)
	}

	rt.Code = append(rt.Code, 0x90)
	if err := WriteObject(rtPath, rt); err != nil {
		t.Fatalf("rewrite runtime object: %v", err)
	}
	stats2, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{RuntimeObjectPath: rtPath},
	)
	if err != nil {
		t.Fatalf("build2 with changed runtime override: %v", err)
	}
	second, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read build2 output: %v", err)
	}
	if sha256.Sum256(first) == sha256.Sum256(second) {
		t.Fatalf("output did not change after runtime object changed")
	}
	if len(stats1.CompiledModules) == 0 && len(stats1.CacheHits) == 0 {
		t.Fatalf("first build had no module activity")
	}
	if len(stats2.CacheHits) == 0 {
		t.Fatalf(
			"second build should still be able to reuse program module cache while relinking runtime",
		)
	}
}

func buildHostRuntimeObject(triple string, actorEntries []string) (*Object, error) {
	switch triple {
	case "linux-x64":
		return actorsrt.BuildLinuxX64(actorEntries)
	case "macos-x64":
		return actorsrt.BuildMacOSX64(actorEntries)
	case "windows-x64":
		return actorsrt.BuildWindowsX64(actorEntries)
	default:
		return nil, ctarget.UnsupportedTargetError{Triple: triple}
	}
}

func TestRuntimeObjectOverrideRejectsWithoutRuntimeUsage(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_unused.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_unused",
		Code:    []byte{0xC3},
		Symbols: []Symbol{{Name: "__tetra_entry", Offset: 0}},
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	srcPath := filepath.Join(tmp, "plain_main.t4")
	if err := os.WriteFile(srcPath, []byte(`func main() -> Int:
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "plain_main"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{RuntimeObjectPath: rtPath},
	)
	if err == nil {
		t.Fatalf("expected runtime object override without runtime usage to fail")
	}
	if !strings.Contains(err.Error(), "runtime object override requires runtime usage") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingRequiredSymbols(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_symbols.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing",
		Code:    []byte{0xC3},
		Symbols: []Symbol{{Name: "__tetra_entry", Offset: 0}},
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(
		filepath.Join("..", "examples", "actors", "actors_pingpong.tetra"),
		outPath,
		tgt.Triple,
		BuildOptions{RuntimeObjectPath: rtPath},
	)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "runtime object missing required symbol") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsSignatureMismatch(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors", "actors_pingpong.tetra")
	world, err := LoadWorld(srcPath)
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	actorsUsed, actorEntries, _, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collect actor entries: %v", err)
	}
	if !actorsUsed || len(actorEntries) == 0 {
		t.Fatalf("expected actors usage")
	}

	rt, err := buildHostRuntimeObject(tgt.Triple, actorEntries)
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	rt.Target = tgt.Triple
	rt.Module = "__runtime"
	annotateRuntimeObjectSignatures(rt)
	replaceRuntimeSymbolSignature(rt, "__tetra_actor_spawn", 2, 1)

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_wrong_signature.tobj")
	if err := WriteObject(rtPath, rt); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	_, err = BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{RuntimeObjectPath: rtPath},
	)
	if err == nil {
		t.Fatalf("expected signature mismatch error, got nil")
	}
	if !strings.Contains(
		err.Error(),
		"runtime object symbol '__tetra_actor_spawn' signature mismatch",
	) ||
		!strings.Contains(err.Error(), "params=2 want=1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingTaggedMessageSymbols(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	symbols := make([]Symbol, 0, len(requiredActorRuntimeSymbols()))
	for _, name := range []string{
		"__tetra_entry",
		"__tetra_actor_spawn",
		"__tetra_actor_send",
		"__tetra_actor_recv",
		"__tetra_actor_self",
		"__tetra_actor_sender",
	} {
		symbols = append(symbols, Symbol{Name: name, Offset: 0})
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_tagged_msg_symbols.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing_tagged_msg",
		Code:    []byte{0xC3},
		Symbols: symbols,
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	outPath := filepath.Join(tmp, "actors_tagged_stress"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(
		filepath.Join("..", "examples", "actors", "actors_tagged_stress.tetra"),
		outPath,
		tgt.Triple,
		BuildOptions{RuntimeObjectPath: rtPath},
	)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(
		err.Error(),
		"runtime object missing required symbol '__tetra_actor_send_msg'",
	) &&
		!strings.Contains(
			err.Error(),
			"runtime object missing required symbol '__tetra_actor_recv_msg'",
		) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingActorStateSymbols(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	symbols := make([]Symbol, 0, len(requiredActorRuntimeSymbols()))
	for _, name := range requiredActorRuntimeSymbols() {
		symbols = append(symbols, Symbol{Name: name, Offset: 0})
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_actor_state_symbols.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing_actor_state",
		Code:    []byte{0xC3},
		Symbols: symbols,
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	srcPath := filepath.Join(tmp, "actor_state_main.tetra")
	src := `actor Counter:
    var count: Int = 0
    func run() -> Int:
        count = count + 1
        return count

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Counter.run")
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "actor_state_main"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{RuntimeObjectPath: rtPath},
	)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(
		err.Error(),
		"runtime object missing required symbol '__tetra_actor_state_load'",
	) &&
		!strings.Contains(
			err.Error(),
			"runtime object missing required symbol '__tetra_actor_state_store'",
		) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingTimeSymbols(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	symbols := make([]Symbol, 0, len(requiredActorRuntimeSymbols()))
	for _, name := range requiredActorRuntimeSymbols() {
		symbols = append(symbols, Symbol{Name: name, Offset: 0})
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_time_symbols.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing_time",
		Code:    []byte{0xC3},
		Symbols: symbols,
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "time"+tgt.ExeExt)
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses runtime:
    return core.time_now_ms()
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{RuntimeObjectPath: rtPath},
	)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(
		err.Error(),
		"runtime object missing required symbol '__tetra_time_now_ms'",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsTargetMismatch(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	other := "windows-x64"
	if tgt.Triple == "windows-x64" {
		other = "linux-x64"
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_wrong_target.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  other,
		Module:  "__runtime_wrong_target",
		Code:    []byte{0xC3},
		Symbols: []Symbol{{Name: "__tetra_entry", Offset: 0}},
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(
		filepath.Join("..", "examples", "actors", "actors_pingpong.tetra"),
		outPath,
		tgt.Triple,
		BuildOptions{RuntimeObjectPath: rtPath},
	)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "runtime object target mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideBuildsForAllX64Targets(t *testing.T) {
	srcPath := filepath.Join("..", "examples", "actors", "actors_pingpong.tetra")
	world, err := LoadWorld(srcPath)
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	actorsUsed, actorEntries, _, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collect actor entries: %v", err)
	}
	if !actorsUsed || len(actorEntries) == 0 {
		t.Fatalf("expected actors usage")
	}

	for _, triple := range []string{"linux-x64", "macos-x64", "windows-x64"} {
		t.Run(triple, func(t *testing.T) {
			tgt, err := ctarget.Parse(triple)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			rt, err := buildHostRuntimeObject(triple, actorEntries)
			if err != nil {
				t.Fatalf("build runtime: %v", err)
			}
			rt.Target = triple
			rt.Module = "__runtime"
			annotateRuntimeObjectSignatures(rt)

			tmp := t.TempDir()
			rtPath := filepath.Join(tmp, "runtime.tobj")
			if err := WriteObject(rtPath, rt); err != nil {
				t.Fatalf("write runtime object: %v", err)
			}
			outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				triple,
				BuildOptions{RuntimeObjectPath: rtPath},
			); err != nil {
				t.Fatalf("build with runtime override: %v", err)
			}
			if _, err := os.Stat(outPath); err != nil {
				t.Fatalf("missing output: %v", err)
			}
		})
	}
}

// ---- security_review_gate_v1_test.go ----

func TestP24SecurityReviewGateV1CoversMasterPlanSurfacesAndArtifacts(t *testing.T) {
	report, err := BuildP24SecurityReviewGateV1Report()
	if err != nil {
		t.Fatalf("BuildP24SecurityReviewGateV1Report: %v", err)
	}
	if report.SchemaVersion != securityReviewGateV1Schema {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, securityReviewGateV1Schema)
	}
	if report.Scope != securityReviewGateV1ScopeP240 {
		t.Fatalf("scope = %q, want %q", report.Scope, securityReviewGateV1ScopeP240)
	}
	if err := ValidateP24SecurityReviewGateV1Report(report); err != nil {
		t.Fatalf("ValidateP24SecurityReviewGateV1Report: %v", err)
	}

	rows := map[SecurityReviewGateV1ID]SecurityReviewGateV1Row{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 ||
			len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p24SecurityReviewGateV1IDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p24AssertSecurityReviewGateRow(
		t,
		rows[SecurityReviewUnsafeAPISurface],
		[]string{"docs/spec/runtime/unsafe.md", "core.cap_mem", "core.alloc_bytes"},
	)
	p24AssertSecurityReviewGateRow(
		t,
		rows[SecurityReviewCapabilitySurface],
		[]string{"docs/spec/runtime/capabilities.md", "cap.mem", "uses"},
	)
	p24AssertSecurityReviewGateRow(
		t,
		rows[SecurityReviewMemoryAllocator],
		[]string{"RuntimeAllocationContracts", "raw-pointer-bounds-v1", "core.alloc_bytes"},
	)
	p24AssertSecurityReviewGateRow(
		t,
		rows[SecurityReviewNetworkRuntime],
		[]string{"IOReactorCoverage", "Linux epoll", "backpressure"},
	)
	p24AssertSecurityReviewGateRow(
		t,
		rows[SecurityReviewActorRuntime],
		[]string{"ActorRuntimeProductionBoundaryAudit", "message pool", "not a production"},
	)
	p24AssertSecurityReviewGateRow(
		t,
		rows[SecurityReviewDBProtocol],
		[]string{
			"ProductionPostgresCoverage",
			"SCRAM-SHA-256",
			"ErrFrameTooLarge",
			"ErrPoolExhausted",
		},
	)
	p24AssertSecurityReviewGateRow(
		t,
		rows[SecurityReviewPackageEcoSystem],
		[]string{"tetra.eco.publish.v1", "Tetra.lock", "validate-eco"},
	)
	p24AssertSecurityReviewGateRow(
		t,
		rows[SecurityReviewBuildScripts],
		[]string{
			"scripts/release/v1_0/security-review.sh",
			"Artifact Hashes",
			"current_release_version",
		},
	)
	p24AssertSecurityReviewGateRow(
		t,
		rows[SecurityReviewSupplyChain],
		[]string{"sha256", "go.sum", "trust snapshot", "no network trust claim"},
	)
	p24AssertSecurityReviewGateRow(
		t,
		rows[SecurityReviewArtifactSet],
		[]string{
			"security-review.md",
			"threat-model.md",
			"unsafe-surface-map.md",
			"capability-surface-map.md",
		},
	)

	witnesses := map[string]SecurityReviewGateV1Witness{}
	for _, witness := range report.Witnesses {
		witnesses[witness.ID] = witness
	}
	allocator := witnesses[p24SecurityReviewAllocatorWitnessID]
	if !allocator.MemoryAllocatorReviewed || allocator.RuntimeAllocationContracts < 5 ||
		allocator.RawPointerBoundsMetadataVersion != "raw-pointer-bounds-v1" {
		t.Fatalf(
			"allocator witness = %#v, want allocation contracts and raw-pointer bounds metadata",
			allocator,
		)
	}
	network := witnesses[p24SecurityReviewNetworkWitnessID]
	if !network.NetworkRuntimeReviewed || network.IOReactorRows < 10 {
		t.Fatalf("network witness = %#v, want IO reactor coverage", network)
	}
	actor := witnesses[p24SecurityReviewActorWitnessID]
	if !actor.ActorRuntimeReviewed || actor.ActorBoundaryRows < 4 {
		t.Fatalf("actor witness = %#v, want actor runtime boundary audit", actor)
	}
	db := witnesses[p24SecurityReviewDBWitnessID]
	if !db.DBProtocolReviewed || db.ProductionPostgresRows < 8 {
		t.Fatalf("DB witness = %#v, want production PostgreSQL coverage", db)
	}
	artifacts := witnesses[p24SecurityReviewArtifactsWitnessID]
	if !artifacts.SecurityReviewArtifactPresent || !artifacts.ThreatModelArtifactPresent ||
		!artifacts.UnsafeSurfaceMapPresent ||
		!artifacts.CapabilitySurfaceMapPresent {
		t.Fatalf("artifact witness missing required artifact: %#v", artifacts)
	}

	if !report.UnsafeAPISurfaceReviewed || !report.CapabilitySurfaceReviewed ||
		!report.MemoryAllocatorReviewed ||
		!report.NetworkRuntimeReviewed ||
		!report.ActorRuntimeReviewed ||
		!report.DBProtocolReviewed ||
		!report.PackageEcoSystemReviewed ||
		!report.BuildScriptsReviewed ||
		!report.SupplyChainReviewed {
		t.Fatalf("review flags missing: %#v", report)
	}
	if !report.SecurityReviewArtifactPresent || !report.ThreatModelArtifactPresent ||
		!report.UnsafeSurfaceMapPresent ||
		!report.CapabilitySurfaceMapPresent {
		t.Fatalf("required artifacts missing: %#v", report.Artifacts)
	}
	for _, nonClaim := range []string{
		"security certification is not claimed",
		"external penetration test is not claimed",
		"CVE-free status is not claimed",
		"release security signoff is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p24SecurityReviewHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP24SecurityReviewGateV1RejectsFakeClaimsAndWeakEvidence(t *testing.T) {
	base, err := BuildP24SecurityReviewGateV1Report()
	if err != nil {
		t.Fatalf("BuildP24SecurityReviewGateV1Report: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*SecurityReviewGateV1Report)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "missing artifact",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.Artifacts[0].Present = false
				report.SecurityReviewArtifactPresent = false
			},
			want: "security-review.md",
		},
		{
			name: "unsafe surface missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.UnsafeAPISurfaceReviewed = false
			},
			want: "unsafe API",
		},
		{
			name: "capability surface missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.CapabilitySurfaceReviewed = false
			},
			want: "capability",
		},
		{
			name: "memory allocator missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.MemoryAllocatorReviewed = false
			},
			want: "memory allocator",
		},
		{
			name: "network runtime missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.NetworkRuntimeReviewed = false
			},
			want: "network runtime",
		},
		{
			name: "actor runtime missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.ActorRuntimeReviewed = false
			},
			want: "actor runtime",
		},
		{
			name: "DB protocol missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.DBProtocolReviewed = false
			},
			want: "DB protocol",
		},
		{
			name: "package Eco missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.PackageEcoSystemReviewed = false
			},
			want: "package/Eco",
		},
		{
			name: "build scripts missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.BuildScriptsReviewed = false
			},
			want: "build scripts",
		},
		{
			name: "supply chain missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.SupplyChainReviewed = false
			},
			want: "supply chain",
		},
		{
			name: "security certification claim",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.SecurityCertifiedClaimed = true
			},
			want: "security certification",
		},
		{
			name: "external penetration claim",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.ExternalPenTestClaimed = true
			},
			want: "external penetration",
		},
		{
			name: "CVE free claim",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.CVEFreeClaimed = true
			},
			want: "CVE-free",
		},
		{
			name: "release signoff claim",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.ReleaseSignoffClaimed = true
			},
			want: "release signoff",
		},
		{
			name: "runtime behavior change",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics change",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.SafeSemanticsChanged = true
			},
			want: "safe semantics",
		},
		{
			name: "performance claim",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := base
			report.Rows = append([]SecurityReviewGateV1Row(nil), base.Rows...)
			report.Witnesses = append([]SecurityReviewGateV1Witness(nil), base.Witnesses...)
			report.Artifacts = append([]SecurityReviewArtifact(nil), base.Artifacts...)
			report.NonClaims = append([]string(nil), base.NonClaims...)
			tc.mutate(&report)
			err := ValidateP24SecurityReviewGateV1Report(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateP24SecurityReviewGateV1Report error = %v, want %q", err, tc.want)
			}
		})
	}
}

func p24AssertSecurityReviewGateRow(t *testing.T, row SecurityReviewGateV1Row, wants []string) {
	t.Helper()
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}

// ---- self_hosting_gate_v1_test.go ----

func TestP23SelfHostingGateV1BlocksPromotionUntilBootstrapEvidenceExists(t *testing.T) {
	report, err := BuildP23SelfHostingGateV1Report()
	if err != nil {
		t.Fatalf("BuildP23SelfHostingGateV1Report: %v", err)
	}
	if report.SchemaVersion != selfHostingGateV1Schema {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, selfHostingGateV1Schema)
	}
	if report.Scope != selfHostingGateV1ScopeP233 {
		t.Fatalf("scope = %q, want %q", report.Scope, selfHostingGateV1ScopeP233)
	}
	if err := ValidateP23SelfHostingGateV1Report(report); err != nil {
		t.Fatalf("ValidateP23SelfHostingGateV1Report: %v", err)
	}

	rows := map[SelfHostingGateV1ID]SelfHostingGateV1Row{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 ||
			len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p23SelfHostingGateV1IDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p23AssertSelfHostingGateRow(
		t,
		rows[SelfHostingGateSubsetDefinition],
		[]string{"verified subset", "not self-hosting"},
	)
	p23AssertSelfHostingGateRow(
		t,
		rows[SelfHostingGateSmallComponentCompile],
		[]string{"small compiler component", "blocked"},
	)
	p23AssertSelfHostingGateRow(
		t,
		rows[SelfHostingGateOutputComparison],
		[]string{"Go compiler output", "Tetra-compiled output", "blocked"},
	)
	p23AssertSelfHostingGateRow(
		t,
		rows[SelfHostingGateRegisterBackend],
		[]string{"register backend", "CheckBackendMatrix", "Machine IR"},
	)
	p23AssertSelfHostingGateRow(
		t,
		rows[SelfHostingGateOptimizerValidation],
		[]string{"optimizer validation", "translation validation v2"},
	)
	p23AssertSelfHostingGateRow(
		t,
		rows[SelfHostingGateAllocatorRuntime],
		[]string{"allocator/runtime", "RuntimeAllocationContracts"},
	)
	p23AssertSelfHostingGateRow(
		t,
		rows[SelfHostingGateStdlibSufficiency],
		[]string{"stdlib", "RegionAwareStdlibCoverage"},
	)
	p23AssertSelfHostingGateRow(
		t,
		rows[SelfHostingGateDeterministicBootstrap],
		[]string{"deterministic bootstrap", "blocked"},
	)
	p23AssertSelfHostingGateRow(
		t,
		rows[SelfHostingGateCrossPlatformBootstrap],
		[]string{"cross-platform bootstrap", "blocked"},
	)
	p23AssertSelfHostingGateRow(
		t,
		rows[SelfHostingGateNoSelfHostingClaim],
		[]string{"SelfHostingClaimed=false", "GateDecision.Allowed=false"},
	)

	witnesses := map[string]SelfHostingGateV1Witness{}
	for _, witness := range report.Witnesses {
		witnesses[witness.ID] = witness
	}
	subset := witnesses[p23SelfHostingSubsetWitnessID]
	if !subset.CompilerSubsetDefined || !strings.Contains(subset.SubsetName, "verified") {
		t.Fatalf("subset witness = %#v, want defined verified subset boundary", subset)
	}
	backend := witnesses[p23SelfHostingRegisterBackendWitnessID]
	if !backend.RegisterBackendEvidencePresent || backend.BackendMatrixLanes < 5 {
		t.Fatalf("backend witness = %#v, want register backend matrix evidence", backend)
	}
	optimizer := witnesses[p23SelfHostingOptimizerWitnessID]
	if !optimizer.OptimizerValidationEvidencePresent || optimizer.TranslationValidationRows < 6 {
		t.Fatalf("optimizer witness = %#v, want translation validation v2 evidence", optimizer)
	}
	allocator := witnesses[p23SelfHostingAllocatorRuntimeWitnessID]
	if !allocator.AllocatorRuntimeEvidencePresent || allocator.RuntimeAllocationContracts < 5 ||
		!allocator.PerCoreSmallHeapEvidencePresent {
		t.Fatalf(
			"allocator/runtime witness = %#v, want allocation contract and small heap evidence",
			allocator,
		)
	}
	stdlib := witnesses[p23SelfHostingStdlibWitnessID]
	if !stdlib.StdlibEvidencePresent || stdlib.StdlibRows < 10 {
		t.Fatalf("stdlib witness = %#v, want region-aware stdlib evidence", stdlib)
	}

	if !report.CompilerSubsetDefined || report.SmallCompilerComponentCompiled ||
		report.GoVsTetraOutputCompared ||
		report.DeterministicBootstrapChain ||
		report.CrossPlatformBootstrapStory {
		t.Fatalf(
			"self-host progress flags = subset:%v component:%v compare:%v bootstrap:%v cross:%v",
			report.CompilerSubsetDefined,
			report.SmallCompilerComponentCompiled,
			report.GoVsTetraOutputCompared,
			report.DeterministicBootstrapChain,
			report.CrossPlatformBootstrapStory,
		)
	}
	if !report.RegisterBackendEvidencePresent || !report.OptimizerValidationEvidencePresent ||
		!report.AllocatorRuntimeEvidencePresent ||
		!report.StdlibEvidencePresent {
		t.Fatalf("existing evidence flags missing: %#v", report)
	}
	if report.SelfHostingClaimed || report.GateDecision.Allowed {
		t.Fatalf("P23.3 must not promote self-hosting: %#v", report.GateDecision)
	}
	for _, missing := range []string{
		"small_compiler_component_compiled",
		"go_vs_tetra_output_compared",
		"deterministic_bootstrap_chain",
		"cross_platform_bootstrap_story",
	} {
		if !report.GateDecision.Missing(missing) {
			t.Fatalf("gate decision missing blocker %q: %#v", missing, report.GateDecision)
		}
	}
	for _, nonClaim := range []string{
		"Tetra is not self-hosting",
		"no Tetra compiler component is claimed to compile itself yet",
		"no deterministic bootstrap chain is claimed yet",
		"no cross-platform bootstrap story is claimed yet",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p23SelfHostingGateHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP23SelfHostingGateV1RejectsFakeClaimsAndWeakEvidence(t *testing.T) {
	base, err := BuildP23SelfHostingGateV1Report()
	if err != nil {
		t.Fatalf("BuildP23SelfHostingGateV1Report: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*SelfHostingGateV1Report)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *SelfHostingGateV1Report) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *SelfHostingGateV1Report) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness",
			mutate: func(report *SelfHostingGateV1Report) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "self hosting claim",
			mutate: func(report *SelfHostingGateV1Report) {
				report.SelfHostingClaimed = true
			},
			want: "self-hosting claim",
		},
		{
			name: "gate allowed",
			mutate: func(report *SelfHostingGateV1Report) {
				report.GateDecision.Allowed = true
			},
			want: "gate decision",
		},
		{
			name: "gate missing blockers omitted",
			mutate: func(report *SelfHostingGateV1Report) {
				report.GateDecision.MissingEvidence = nil
			},
			want: "gate decision",
		},
		{
			name: "compiler subset missing",
			mutate: func(report *SelfHostingGateV1Report) {
				report.CompilerSubsetDefined = false
			},
			want: "compiler subset",
		},
		{
			name: "register backend evidence missing",
			mutate: func(report *SelfHostingGateV1Report) {
				report.RegisterBackendEvidencePresent = false
			},
			want: "register backend",
		},
		{
			name: "optimizer evidence missing",
			mutate: func(report *SelfHostingGateV1Report) {
				report.OptimizerValidationEvidencePresent = false
			},
			want: "optimizer",
		},
		{
			name: "allocator runtime evidence missing",
			mutate: func(report *SelfHostingGateV1Report) {
				report.AllocatorRuntimeEvidencePresent = false
			},
			want: "allocator/runtime",
		},
		{
			name: "stdlib evidence missing",
			mutate: func(report *SelfHostingGateV1Report) {
				report.StdlibEvidencePresent = false
			},
			want: "stdlib",
		},
		{
			name: "fake compiler component",
			mutate: func(report *SelfHostingGateV1Report) {
				report.SmallCompilerComponentCompiled = true
			},
			want: "small compiler component",
		},
		{
			name: "fake output comparison",
			mutate: func(report *SelfHostingGateV1Report) {
				report.GoVsTetraOutputCompared = true
			},
			want: "output comparison",
		},
		{
			name: "fake deterministic bootstrap",
			mutate: func(report *SelfHostingGateV1Report) {
				report.DeterministicBootstrapChain = true
			},
			want: "deterministic bootstrap",
		},
		{
			name: "fake cross platform bootstrap",
			mutate: func(report *SelfHostingGateV1Report) {
				report.CrossPlatformBootstrapStory = true
			},
			want: "cross-platform bootstrap",
		},
		{
			name: "runtime behavior change",
			mutate: func(report *SelfHostingGateV1Report) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics change",
			mutate: func(report *SelfHostingGateV1Report) {
				report.SafeSemanticsChanged = true
			},
			want: "safe semantics",
		},
		{
			name: "performance claim",
			mutate: func(report *SelfHostingGateV1Report) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := base
			report.Rows = append([]SelfHostingGateV1Row(nil), base.Rows...)
			report.Witnesses = append([]SelfHostingGateV1Witness(nil), base.Witnesses...)
			report.NonClaims = append([]string(nil), base.NonClaims...)
			report.GateDecision.MissingEvidence = append(
				[]string(nil),
				base.GateDecision.MissingEvidence...)
			tc.mutate(&report)
			err := ValidateP23SelfHostingGateV1Report(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateP23SelfHostingGateV1Report error = %v, want %q", err, tc.want)
			}
		})
	}
}

func p23AssertSelfHostingGateRow(t *testing.T, row SelfHostingGateV1Row, wants []string) {
	t.Helper()
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}

// ---- surface_runtime_test.go ----

func TestSurfaceRuntimeRequiredSymbolsAndSignatures(t *testing.T) {
	got := requiredSurfaceRuntimeSymbols()
	want := []string{
		"__tetra_surface_open",
		"__tetra_surface_close",
		"__tetra_surface_poll_event_kind",
		"__tetra_surface_poll_event_x",
		"__tetra_surface_poll_event_y",
		"__tetra_surface_poll_event_button",
		"__tetra_surface_poll_event_into",
		"__tetra_surface_poll_event_text_len",
		"__tetra_surface_poll_event_text_into",
		"__tetra_surface_clipboard_write_text",
		"__tetra_surface_clipboard_read_text_into",
		"__tetra_surface_poll_composition_into",
		"__tetra_surface_begin_frame",
		"__tetra_surface_present_rgba",
		"__tetra_surface_now_ms",
		"__tetra_surface_request_redraw",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("surface runtime symbols = %#v, want %#v", got, want)
	}
	tests := []struct {
		name   string
		params int
		rets   int
	}{
		{name: "__tetra_surface_open", params: 4, rets: 1},
		{name: "__tetra_surface_close", params: 1, rets: 1},
		{name: "__tetra_surface_poll_event_kind", params: 1, rets: 1},
		{name: "__tetra_surface_poll_event_x", params: 1, rets: 1},
		{name: "__tetra_surface_poll_event_y", params: 1, rets: 1},
		{name: "__tetra_surface_poll_event_button", params: 1, rets: 1},
		{name: "__tetra_surface_poll_event_into", params: 3, rets: 1},
		{name: "__tetra_surface_poll_event_text_len", params: 1, rets: 1},
		{name: "__tetra_surface_poll_event_text_into", params: 3, rets: 1},
		{name: "__tetra_surface_clipboard_write_text", params: 3, rets: 1},
		{name: "__tetra_surface_clipboard_read_text_into", params: 3, rets: 1},
		{name: "__tetra_surface_poll_composition_into", params: 3, rets: 1},
		{name: "__tetra_surface_begin_frame", params: 1, rets: 1},
		{name: "__tetra_surface_present_rgba", params: 6, rets: 1},
		{name: "__tetra_surface_now_ms", params: 0, rets: 1},
		{name: "__tetra_surface_request_redraw", params: 1, rets: 1},
	}
	for _, tt := range tests {
		sig, ok := runtimeObjectSignature(tt.name)
		if !ok {
			t.Fatalf("missing runtime signature for %s", tt.name)
		}
		if sig.paramSlots != tt.params || sig.returnSlots != tt.rets {
			t.Fatalf(
				"%s signature = params %d returns %d, want params %d returns %d",
				tt.name,
				sig.paramSlots,
				sig.returnSlots,
				tt.params,
				tt.rets,
			)
		}
	}
}

func TestCollectSurfaceRuntimeUsage(t *testing.T) {
	prog, err := Parse([]byte(`
func probe() -> Int
uses surface:
    return core.surface_open("demo", 10, 10)

func main() -> Int:
    return 0
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !collectSurfaceRuntimeUsage(checked) {
		t.Fatalf("surface runtime usage was not collected")
	}
}

func TestValidateSurfaceRuntimeObjectChecksSignatureMetadata(t *testing.T) {
	obj := runtimeObjectWithSurfaceRuntimeSignatures()
	if err := validateSurfaceRuntimeObject(obj); err != nil {
		t.Fatalf("validate surface runtime object: %v", err)
	}

	replaceRuntimeSymbolSignature(obj, "__tetra_surface_present_rgba", 5, 1)
	err := validateSurfaceRuntimeObject(obj)
	if err == nil {
		t.Fatalf("expected surface runtime signature mismatch")
	}
	if !strings.Contains(
		err.Error(),
		"runtime object symbol '__tetra_surface_present_rgba' signature mismatch",
	) ||
		!strings.Contains(err.Error(), "params=5 want=6") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingSurfaceSymbols(t *testing.T) {
	tgt, ok := ctarget.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if tgt.Triple != "linux-x64" {
		t.Skipf("surface runtime is linux-x64 only for this slice, host is %s", tgt.Triple)
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_surface.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing_surface",
		Code:    []byte{0xC3},
		Symbols: runtimeObjectSymbols(requiredActorRuntimeSymbols()),
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	srcPath := filepath.Join(tmp, "surface_main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses surface:
    return core.surface_open("demo", 10, 10)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "surface_main"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{RuntimeObjectPath: rtPath},
	)
	if err == nil {
		t.Fatalf("expected missing surface runtime symbol failure")
	}
	if !strings.Contains(
		err.Error(),
		"runtime object missing required symbol '__tetra_surface_open'",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSurfaceRuntimeRejectsUnsupportedNativeTargets(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "surface_main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses surface:
    return core.surface_open("demo", 10, 10)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	for _, tc := range []struct {
		target string
		want   string
	}{
		{target: "macos-x64", want: "macos-x64"},
		{target: "windows-x64", want: "windows-x64"},
		{target: "x32", want: "linux-x32"},
		{target: "x86", want: "linux-x86"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			outPath := filepath.Join(tmp, "surface-"+tc.target)
			_, err := BuildFileWithStatsOpt(srcPath, outPath, tc.target, BuildOptions{Jobs: 1})
			if err == nil {
				t.Fatalf("expected unsupported surface runtime diagnostic")
			}
			want := "surface runtime not supported on " + tc.want
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want %q", err, want)
			}
		})
	}
}

func TestSurfaceRuntimeBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("demo", 2, 2)
    let pixels: []u8 = core.make_u8(16)
    let present: Int = core.surface_present_rgba(handle, pixels, 2, 2, 8)
    let first_close: Int = core.surface_close(handle)
    let second_close: Int = core.surface_close(handle)
    if handle > 2 && present == 0 && first_close == 0 && second_close != 0:
        return 42
    return 1
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want kernel-backed linux-x64 Surface host result 42", exitCode)
	}
}

func TestSurfaceRuntimePollEventCoordinatesLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses surface:
    let handle: Int = core.surface_open("event-probe", 320, 200)
    let kind: Int = core.surface_poll_event_kind(handle)
    let x: Int = core.surface_poll_event_x(handle)
    let y: Int = core.surface_poll_event_y(handle)
    let button: Int = core.surface_poll_event_button(handle)
    let closed: Int = core.surface_close(handle)
    if closed == 0 && kind == 5 && x == 48 && y == 96 && button == 1:
        return 42
    return kind
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf(
			"exit code = %d, want host-provided linux-x64 Surface pointer event result 42",
			exitCode,
		)
	}
}

func TestSurfaceRuntimePollEventTextPayloadLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("text-probe", 320, 200)
    var text: []u8 = core.make_u8(4)
    let text_len: Int = core.surface_poll_event_text_len(handle)
    let copied: Int = core.surface_poll_event_text_into(handle, text)
    let closed: Int = core.surface_close(handle)
    if closed == 0 && text_len == 2 && copied == 2 && text[0] == 79 && text[1] == 75:
        return 42
    return text_len + copied
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf(
			"exit code = %d, want host-provided linux-x64 Surface text payload result 42",
			exitCode,
		)
	}
}

func TestSurfaceRuntimePollEventBufferLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("event-buffer-probe", 320, 200)
    var event: []i32 = core.make_i32(9)
    let copied: Int = core.surface_poll_event_into(handle, event)
    let closed: Int = core.surface_close(handle)
    if closed == 0 && copied == 9 && event[0] == 5 && event[1] == 48 && event[2] == 96 && event[3] == 1 && event[4] == 0 && event[5] == 320 && event[6] == 200 && event[7] == 0 && event[8] == 0:
        return 42
    return copied
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf(
			"exit code = %d, want host-provided linux-x64 Surface event buffer result 42",
			exitCode,
		)
	}
}

func TestSurfaceRuntimePollEventBufferSequenceLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("event-sequence-probe", 320, 200)
    var first: []i32 = core.make_i32(9)
    var second: []i32 = core.make_i32(9)
    var third: []i32 = core.make_i32(9)
    let copied1: Int = core.surface_poll_event_into(handle, first)
    let copied2: Int = core.surface_poll_event_into(handle, second)
    let copied3: Int = core.surface_poll_event_into(handle, third)
    let closed: Int = core.surface_close(handle)
    if closed == 0 && copied1 == 9 && first[0] == 5 && first[1] == 48 && first[2] == 96 && first[3] == 1 && first[4] == 0 && first[5] == 320 && first[6] == 200 && first[7] == 0 && first[8] == 0 && copied2 == 9 && second[0] == 6 && second[1] == 0 && second[2] == 0 && second[3] == 0 && second[4] == 32 && second[5] == 320 && second[6] == 200 && second[7] == 1 && second[8] == 0 && copied3 == 9 && third[0] == 2 && third[1] == 0 && third[2] == 0 && third[3] == 0 && third[4] == 0 && third[5] == 400 && third[6] == 240 && third[7] == 2 && third[8] == 0:
        return 42
    return copied1 + copied2 + copied3
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf(
			"exit code = %d, want linux-x64 Surface poll_event_into sequence pointer/key/resize",
			exitCode,
		)
	}
}

func TestSurfaceRuntimePresentPreservesPollEventCursorLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("event-present-cursor-probe", 320, 200)
    var first: []i32 = core.make_i32(9)
    var second: []i32 = core.make_i32(9)
    var pixels: []u8 = core.make_u8(16)
    let copied1: Int = core.surface_poll_event_into(handle, first)
    let presented: Int = core.surface_present_rgba(handle, pixels, 2, 2, 8)
    let copied2: Int = core.surface_poll_event_into(handle, second)
    let closed: Int = core.surface_close(handle)
    if closed == 0 && presented == 0 && copied1 == 9 && first[0] == 5 && copied2 == 9 && second[0] == 6 && second[4] == 32:
        return 42
    return second[0]
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf(
			"exit code = %d, want present_rgba to preserve linux-x64 Surface event cursor",
			exitCode,
		)
	}
}

func TestSurfaceCounterExampleBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "surface-counter")
	if _, err := BuildFileWithStatsOpt(
		filepath.Join("..", "examples", "surface", "runtime", "surface_counter.tetra"),
		outPath,
		"linux-x64",
		BuildOptions{Jobs: 1},
	); err != nil {
		t.Fatalf("build surface counter: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	stdout, exitCode := runBinary(t, outPath)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want deterministic Surface counter result 1", exitCode)
	}
}

func TestSurfaceTextInputExampleBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "surface-text-input")
	if _, err := BuildFileWithStatsOpt(
		filepath.Join("..", "examples", "surface", "runtime", "surface_text_input.tetra"),
		outPath,
		"linux-x64",
		BuildOptions{Jobs: 1},
	); err != nil {
		t.Fatalf("build surface text input: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	stdout, exitCode := runBinary(t, outPath)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want TextBox-owned host text buffer result 42", exitCode)
	}
}

func TestSurfaceMigrationExamplesBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	cases := []struct {
		name         string
		src          string
		expectedExit int
	}{
		{
			name: "ui_web_smoke",
			src: filepath.Join(
				"..",
				"examples",
				"surface",
				"migration",
				"surface_migration_ui_web_smoke.tetra",
			),
			expectedExit: 26,
		},
		{
			name: "ui_native_shell_smoke",
			src: filepath.Join(
				"..",
				"examples",
				"surface",
				"migration",
				"surface_migration_ui_native_shell_smoke.tetra",
			),
			expectedExit: 27,
		},
		{
			name: "dogfood_web_ui",
			src: filepath.Join(
				"..",
				"examples",
				"surface",
				"migration",
				"surface_migration_dogfood_web_ui.tetra",
			),
			expectedExit: 43,
		},
		{
			name: "tetra_control_center",
			src: filepath.Join(
				"..",
				"examples",
				"surface",
				"migration",
				"surface_migration_tetra_control_center.tetra",
			),
			expectedExit: 5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			outPath := filepath.Join(tmp, tc.name)
			if _, err := BuildFileWithStatsOpt(
				tc.src,
				outPath,
				"linux-x64",
				BuildOptions{Jobs: 1},
			); err != nil {
				t.Fatalf("build %s: %v", tc.src, err)
			}
			if err := verifyELF(outPath); err != nil {
				t.Fatalf("verify ELF: %v", err)
			}
			stdout, exitCode := runBinary(t, outPath)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != tc.expectedExit {
				t.Fatalf(
					"exit code = %d, want deterministic Surface migration result %d",
					exitCode,
					tc.expectedExit,
				)
			}
		})
	}
}

func TestSurfaceCounterExampleBuildWASM32WebSurfaceHost(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "surface-counter.wasm")
	if _, err := BuildFileWithStatsOpt(
		filepath.Join("..", "examples", "surface", "runtime", "surface_counter.tetra"),
		outPath,
		"wasm32-web",
		BuildOptions{Jobs: 1},
	); err != nil {
		t.Fatalf("build surface counter wasm32-web: %v", err)
	}
	if raw, err := os.ReadFile(outPath); err != nil {
		t.Fatalf("read wasm output: %v", err)
	} else if len(raw) < 8 || string(raw[:4]) != "\x00asm" {
		header := raw
		if len(header) > 4 {
			header = header[:4]
		}
		t.Fatalf("wasm output has invalid header: % x", header)
	}

	loaderPath := strings.TrimSuffix(outPath, ".wasm") + ".mjs"
	loader, err := os.ReadFile(loaderPath)
	if err != nil {
		t.Fatalf("read wasm Surface loader: %v", err)
	}
	for _, want := range []string{
		"tetra_surface_host_v1",
		"createSurfaceHost(instanceRef)",
		"__tetra_surface_present_rgba",
	} {
		if !strings.Contains(string(loader), want) {
			t.Fatalf("wasm Surface loader missing %q:\n%s", want, loader)
		}
	}
	for _, sidecar := range []string{
		strings.TrimSuffix(outPath, ".wasm") + ".ui.json",
		strings.TrimSuffix(outPath, ".wasm") + ".ui.web.mjs",
		strings.TrimSuffix(outPath, ".wasm") + ".ui.html",
	} {
		if _, err := os.Stat(sidecar); err == nil {
			t.Fatalf("Surface wasm build must not emit legacy metadata UI sidecar %s", sidecar)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", sidecar, err)
		}
	}
}

func TestSurfaceTextInputExampleBuildWASM32WebSurfaceHost(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "surface-text-input.wasm")
	if _, err := BuildFileWithStatsOpt(
		filepath.Join("..", "examples", "surface", "runtime", "surface_text_input.tetra"),
		outPath,
		"wasm32-web",
		BuildOptions{Jobs: 1},
	); err != nil {
		t.Fatalf("build surface text input wasm32-web: %v", err)
	}
	if raw, err := os.ReadFile(outPath); err != nil {
		t.Fatalf("read wasm output: %v", err)
	} else if len(raw) < 8 || string(raw[:4]) != "\x00asm" {
		header := raw
		if len(header) > 4 {
			header = header[:4]
		}
		t.Fatalf("wasm output has invalid header: % x", header)
	}

	loaderPath := strings.TrimSuffix(outPath, ".wasm") + ".mjs"
	loader, err := os.ReadFile(loaderPath)
	if err != nil {
		t.Fatalf("read wasm Surface loader: %v", err)
	}
	for _, want := range []string{
		"tetra_surface_host_v1",
		"__tetra_surface_poll_event_text_into",
		"__tetra_surface_present_rgba",
	} {
		if !strings.Contains(string(loader), want) {
			t.Fatalf("wasm Surface text loader missing %q:\n%s", want, loader)
		}
	}
}

func TestTenSlotReturnDoesNotClobberBuiltinRuntimeSchedulerLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
struct Ten:
    a: Int
    b: Int
    c: Int
    d: Int
    e: Int
    f: Int
    g: Int
    h: Int
    i: Int
    j: Int

func make_ten() -> Ten:
    return Ten(a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8, i: 9, j: 10)

func main() -> Int
uses runtime:
    let ten: Ten = make_ten()
    let _: Int = core.time_now_ms()
    return ten.a
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf(
			"exit code = %d, want 10-slot return to preserve runtime scheduler state",
			exitCode,
		)
	}
}

func runtimeObjectWithSurfaceRuntimeSignatures() *Object {
	obj := &Object{}
	for _, name := range requiredSurfaceRuntimeSymbols() {
		sig, ok := runtimeObjectSignature(name)
		if !ok {
			panic("missing surface runtime signature for " + name)
		}
		obj.Symbols = append(obj.Symbols, Symbol{
			Name:         name,
			HasSignature: true,
			ParamSlots:   sig.paramSlots,
			ReturnSlots:  sig.returnSlots,
		})
	}
	return obj
}

// ---- task_runtime_cancellation_actor_test.go ----

func TestTaskCancellationCheckpointUngroupedTaskBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let canceled: Int = core.task_is_canceled()
    if canceled != 0:
        return 40 + canceled
    let checkpoint: task.error = core.task_checkpoint()
    if checkpoint != 0:
        return 50 + checkpoint
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    if result.error != 0:
        return 80 + result.error
    return result.value
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 7 {
		t.Fatalf("exit code = %d, want ungrouped checkpoint result 7", exitCode)
	}
}

func TestTaskCancellationCheckpointSeesSelfCanceledGroupBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    let _canceledGroup: task.group = core.task_group_cancel(group)
    let canceled: Int = core.task_is_canceled()
    let checkpoint: task.error = core.task_checkpoint()
    return canceled * 10 + checkpoint

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if result.error != 0:
        return 80 + result.error
    return result.value
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 11 {
		t.Fatalf("exit code = %d, want canceled checkpoint result 11", exitCode)
	}
}

func TestTaskCancellationCheckpointInheritedByNestedChildBuildAndRun(t *testing.T) {
	src := `
func child() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    let _canceledGroup: task.group = core.task_group_cancel(group)
    let canceled: Int = core.task_is_canceled()
    let checkpoint: task.error = core.task_checkpoint()
    return canceled * 10 + checkpoint

func worker() -> Int
uses runtime:
    let childTask: task.i32 = core.task_spawn_i32("child")
    let childResult: task.result_i32 = core.task_join_result_i32(childTask)
    if childResult.error != 0:
        return 70 + childResult.error
    return childResult.value

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if result.error != 0:
        return 80 + result.error
    return result.value
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 11 {
		t.Fatalf("exit code = %d, want nested canceled checkpoint result 11", exitCode)
	}
}

func TestTaskSpawnsActorAndReceivesMailboxReplyBuildAndRun(t *testing.T) {
	src := `
func actor_worker() -> Int
uses actors:
    let _sent: Int = core.send(core.sender(), 6)
    return 0

func worker() -> Int
uses actors:
    let _actor: actor = core.spawn("actor_worker")
    return 1

func main() -> Int
uses actors, runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let reply: actor.recv_result_i32 = core.recv_until(core.deadline_ms(3))
    if reply.error != 0:
        return 40 + reply.error
    let result: task.result_i32 = core.task_join_result_i32(task)
    if result.error != 0:
        return 80 + result.error
    return result.value + reply.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf("program timed out; task-spawned actor should reply to parent mailbox")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 7 {
		t.Fatalf("exit code = %d, want task/actor mailbox reply result 7", exitCode)
	}
}

func TestTaskGroupCancelWakesActorRecvUntilBeforeDeadlineBuildAndRun(t *testing.T) {
	src := `
func actor_waiter() -> Int
uses actors, runtime:
    let result: actor.recv_result_i32 = core.recv_until(core.deadline_ms(100))
    let _sent: Int = core.send(core.sender(), result.error)
    return result.error

func worker() -> Int
uses actors:
    let _actor: actor = core.spawn("actor_waiter")
    return 1

func main() -> Int
uses actors, runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let launched: task.result_i32 = core.task_join_result_i32(task)
    if launched.error != 0:
        return 20 + launched.error
    if launched.value != 1:
        return 30 + launched.value
    let _delay: Int = core.sleep_ms(2)
    group = core.task_group_cancel(group)
    let reply: actor.recv_result_i32 = core.recv_until(core.deadline_ms(5))
    let _closed: Int = core.task_group_close(group)
    if reply.error != 0:
        return 40 + reply.error
    if reply.value != 1:
        return 60 + reply.value
    let now: Int = core.time_now_ms()
    if now != 2:
        return 80 + now
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf(
			"program timed out; task group cancel should wake actor recv_until before the original deadline",
		)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want actor recv_until canceled wake at logical time 2", exitCode)
	}
}

func TestTaskGroupCancelWakesActorRecvMsgUntilBeforeDeadlineBuildAndRun(t *testing.T) {
	src := `
func actor_waiter() -> Int
uses actors, runtime:
    let result: actor.recv_msg_result = core.recv_msg_until(core.deadline_ms(100))
    let _sent: Int = core.send_msg(core.sender(), result.error, result.tag)
    return result.error

func worker() -> Int
uses actors:
    let _actor: actor = core.spawn("actor_waiter")
    return 1

func main() -> Int
uses actors, runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let launched: task.result_i32 = core.task_join_result_i32(task)
    if launched.error != 0:
        return 20 + launched.error
    if launched.value != 1:
        return 30 + launched.value
    let _delay: Int = core.sleep_ms(2)
    group = core.task_group_cancel(group)
    let reply: actor.recv_msg_result = core.recv_msg_until(core.deadline_ms(5))
    let _closed: Int = core.task_group_close(group)
    if reply.error != 0:
        return 40 + reply.error
    if reply.value != 1:
        return 60 + reply.value
    if reply.tag != 0:
        return 70 + reply.tag
    let now: Int = core.time_now_ms()
    if now != 2:
        return 80 + now
    return 0
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf(
			("program timed out; task group cancel should wake actor " +
				"recv_msg_until before the original deadline"),
		)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf(
			"exit code = %d, want actor recv_msg_until canceled wake at logical time 2",
			exitCode,
		)
	}
}

func TestTaskGroupCurrentInheritedByChildTaskBuildAndRun(t *testing.T) {
	src := `
func leaf() -> Int:
    return 1

func child() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    return core.task_group_status(group)

func worker() -> Int
uses runtime:
    let ownGroup: task.group = core.task_group_current()
    let ownStatus: Int = core.task_group_status(ownGroup)
    if ownStatus != 1:
        return 70 + ownStatus
    let childTask: task.i32 = core.task_spawn_i32("child")
    let childResult: task.result_i32 = core.task_join_result_i32(childTask)
    if childResult.error != 0:
        return 90 + childResult.error
    return childResult.value

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if result.error != 0:
        return 80 + result.error
    return result.value
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want inherited open group status 1", exitCode)
	}
}

func TestTaskGroupCurrentVisibleInGroupTaskBuildAndRun(t *testing.T) {
	src := `
func leaf() -> Int:
    return 1

func worker() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    return core.task_group_status(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if result.error != 0:
        return 80 + result.error
    return result.value
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want visible open group status 1", exitCode)
	}
}

func TestTaskGroupCloseMarksOpenGroupClosedBuildAndRun(t *testing.T) {
	src := `
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 70 + closeError
    return core.task_group_status(group)
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 3 {
		t.Fatalf("exit code = %d, want closed group status 3", exitCode)
	}
}

func TestTaskGroupClosePreservesCanceledStatusBuildAndRun(t *testing.T) {
	src := `
func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    group = core.task_group_cancel(group)
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 70 + closeError
    return core.task_group_status(group)
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want canceled group status 2", exitCode)
	}
}

func TestNestedTaskSpawnI32BuildAndRun(t *testing.T) {
	src := `
func child() -> Int:
    return 7

func worker() -> Int
uses runtime:
    let childTask: task.i32 = core.task_spawn_i32("child")
    let childResult: task.result_i32 = core.task_join_result_i32(childTask)
    if childResult.error != 0:
        return 90 + childResult.error
    return childResult.value

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    if result.error != 0:
        return 80 + result.error
    return result.value
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 7 {
		t.Fatalf("exit code = %d, want nested child result 7", exitCode)
	}
}

// ---- task_runtime_deadlines_test.go ----

func TestTimeRuntimeBuiltinsLowerToRuntimeCalls(t *testing.T) {
	src := []byte(`
func main() -> Int
uses runtime:
    let start: Int = core.time_now_ms()
    let err: Int = core.sleep_ms(5)
    let untilErr: Int = core.sleep_until(core.deadline_ms(6))
    let deadline: Int = core.deadline_ms(7)
    return start + err + untilErr + deadline
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
	for _, name := range []string{
		"__tetra_time_now_ms",
		"__tetra_sleep_ms",
		"__tetra_sleep_until_ms",
		"__tetra_deadline_ms",
	} {
		if !hasIRCall(mainFn, name) {
			t.Fatalf("main does not call %s: %#v", name, mainFn.Instrs)
		}
	}
}

func TestDeadlineAwareRuntimeBuiltinsCheckAndLower(t *testing.T) {
	src := []byte(`
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return 42

func main() -> Int
uses actors, runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let _sleepUntil: Int = core.sleep_until(core.deadline_ms(1))
    let joined: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(5))
    let recv: actor.recv_result_i32 = core.recv_until(core.deadline_ms(6))
    return joined.value + joined.error + recv.value + recv.error
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
	for _, name := range []string{
		"__tetra_sleep_until_ms",
		"__tetra_task_join_until_i32",
		"__tetra_actor_recv_until",
	} {
		if !hasIRCall(mainFn, name) {
			t.Fatalf("main does not call %s: %#v", name, mainFn.Instrs)
		}
	}
}

func TestWaitCompositionRuntimeBuiltinsCheckAndLower(t *testing.T) {
	src := []byte(`
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return 42

func main() -> Int
uses actors, runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let polled: task.result_i32 = core.task_poll_i32(task)
    let yielded: Int = core.yield()
    let ready: Bool = core.timer_ready(core.deadline_ms(0))
    let selected: task.result_i32 = core.select2_i32(task, core.deadline_ms(5))
    let recv: actor.recv_result_i32 = core.recv_poll()
    let msg: actor.recv_msg_result = core.recv_msg_until(core.deadline_ms(6))
    if ready:
        return polled.error + yielded + selected.value + recv.error + msg.error
    return 99
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
	for _, name := range []string{
		"__tetra_task_poll_i32",
		"__tetra_actor_yield_now",
		"__tetra_timer_ready_ms",
		"__tetra_task_join_until_i32",
		"__tetra_actor_recv_poll",
		"__tetra_actor_recv_msg_until",
	} {
		if !hasIRCall(mainFn, name) {
			t.Fatalf("main does not call %s: %#v", name, mainFn.Instrs)
		}
	}
}

func TestTimeRuntimeLogicalClockBuildAndRun(t *testing.T) {
	src := `
func main() -> Int
uses runtime:
    let start: Int = core.time_now_ms()
    let err: Int = core.sleep_ms(5)
    let after: Int = core.time_now_ms()
    let deadline: Int = core.deadline_ms(7)
    return (after - start) + deadline + err
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 17 {
		t.Fatalf("exit code = %d, want logical clock result 17", exitCode)
	}
}

func TestSleepUntilUsesAbsoluteDeadlineBuildAndRun(t *testing.T) {
	src := `
func main() -> Int
uses runtime:
    let start: Int = core.time_now_ms()
    let deadline: Int = core.deadline_ms(7)
    let err: Int = core.sleep_until(deadline)
    let after: Int = core.time_now_ms()
    let immediate: Int = core.sleep_until(deadline)
    let finalTime: Int = core.time_now_ms()
    if err != 0:
        return 20 + err
    if immediate != 0:
        return 30 + immediate
    return (after - start) * 10 + (finalTime - after)
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 70 {
		t.Fatalf("exit code = %d, want sleep_until absolute deadline result 70", exitCode)
	}
}

func TestTaskSleepTimersWakeInDeadlineOrderBuildAndRun(t *testing.T) {
	src := `
func slow() -> Int
uses runtime:
    let _err: Int = core.sleep_ms(5)
    return core.time_now_ms()

func fast() -> Int
uses runtime:
    let _err: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let slowTask: task.i32 = core.task_spawn_i32("slow")
    let fastTask: task.i32 = core.task_spawn_i32("fast")
    let _mainSleep: Int = core.sleep_ms(10)
    let fastValue: Int = core.task_join_i32(fastTask)
    let slowValue: Int = core.task_join_i32(slowTask)
    return fastValue * 10 + slowValue
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 25 {
		t.Fatalf("exit code = %d, want fast wake 2 and slow wake 5", exitCode)
	}
}

func TestTaskJoinUntilTimesOutThenFinalJoinCompletesBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(10)
    return 99

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let early: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(3))
    if early.error != 2:
        return 20 + early.error
    if early.value != 0:
        return 40 + early.value
    let afterTimeout: Int = core.time_now_ms()
    if afterTimeout != 3:
        return 60 + afterTimeout
    let final: task.result_i32 = core.task_join_result_i32(task)
    if final.error != 0:
        return 80 + final.error
    if final.value != 99:
        return final.value
    return core.time_now_ms()
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf("program timed out; task_join_until_i32 should wake on timeout deadline")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 10 {
		t.Fatalf("exit code = %d, want final join logical time 10 after timeout", exitCode)
	}
}

func TestTaskJoinUntilReturnsCompletedTaskBeforeDeadlineBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let result: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(5))
    if result.error != 0:
        return 20 + result.error
    return result.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf("program timed out; task_join_until_i32 should wake when task completes")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want joined worker value before deadline 2", exitCode)
	}
}

func TestTaskPollReturnsTimeoutUntilTaskCompletesBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(4)
    return 77

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let early: task.result_i32 = core.task_poll_i32(task)
    if early.error != 2:
        return 20 + early.error
    if early.value != 0:
        return 40 + early.value
    let _sleep: Int = core.sleep_ms(5)
    let late: task.result_i32 = core.task_poll_i32(task)
    if late.error != 0:
        return 60 + late.error
    return late.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf("program timed out; task_poll_i32 must not block")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 77 {
		t.Fatalf("exit code = %d, want completed poll value 77", exitCode)
	}
}

func TestYieldAllowsReadyTaskToRunBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses actors:
    let _sent: Int = core.send(core.sender(), 9)
    return 0

func main() -> Int
uses actors, runtime:
    let _task: task.i32 = core.task_spawn_i32("worker")
    let before: actor.recv_result_i32 = core.recv_poll()
    if before.error != 2:
        return 20 + before.error
    let yielded: Int = core.yield()
    if yielded != 0:
        return 40 + yielded
    let after: actor.recv_result_i32 = core.recv_poll()
    if after.error != 0:
        return 60 + after.error
    return after.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf("program timed out; yield should resume after another ready actor runs")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 9 {
		t.Fatalf("exit code = %d, want recv_poll value after yield", exitCode)
	}
}

func TestTimerReadyAndSelect2TaskTimerBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return 33

func main() -> Int
uses runtime:
    let deadline: Int = core.deadline_ms(2)
    if core.timer_ready(deadline):
        return 10
    let task: task.i32 = core.task_spawn_i32("worker")
    let selected: task.result_i32 = core.select2_i32(task, deadline)
    if selected.error != 2:
        return 20 + selected.error
    if !core.timer_ready(deadline):
        return 40
    let final: task.result_i32 = core.task_join_result_i32(task)
    if final.error != 0:
        return 60 + final.error
    return final.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf("program timed out; select2_i32 should wake on timer deadline")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 33 {
		t.Fatalf("exit code = %d, want final task value after select timeout", exitCode)
	}
}

func TestDocumentedTaskTimeBuiltinSelfHostParity(t *testing.T) {
	tests := []struct {
		name string
		src  string
		exit int
	}{
		{
			name: "sleep",
			src: `
func main() -> Int
uses runtime:
    let start: Int = core.time_now_ms()
    let err: Int = core.sleep_ms(3)
    if err != 0:
        return 20 + err
    let after: Int = core.time_now_ms()
    let untilErr: Int = core.sleep_until(core.deadline_ms(2))
    if untilErr != 0:
        return 40 + untilErr
    let finalTime: Int = core.time_now_ms()
    return (after - start) * 10 + (finalTime - after)
`,
			exit: 32,
		},
		{
			name: "deadline_join",
			src: `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let result: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(5))
    if result.error != 0:
        return 20 + result.error
    return result.value
`,
			exit: 2,
		},
		{
			name: "poll",
			src: `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return 31

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let early: task.result_i32 = core.task_poll_i32(task)
    if early.error != 2:
        return 20 + early.error
    if early.value != 0:
        return 40 + early.value
    let _sleep: Int = core.sleep_ms(3)
    let late: task.result_i32 = core.task_poll_i32(task)
    if late.error != 0:
        return 60 + late.error
    return late.value
`,
			exit: 31,
		},
		{
			name: "select2_timer",
			src: `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return 33

func main() -> Int
uses runtime:
    let deadline: Int = core.deadline_ms(2)
    let task: task.i32 = core.task_spawn_i32("worker")
    let selected: task.result_i32 = core.select2_i32(task, deadline)
    if selected.error != 2:
        return 20 + selected.error
    if !core.timer_ready(deadline):
        return 40
    let final: task.result_i32 = core.task_join_result_i32(task)
    if final.error != 0:
        return 60 + final.error
    return final.value
`,
			exit: 33,
		},
	}

	runtimes := []struct {
		name string
		mode RuntimeMode
	}{
		{name: "builtin", mode: RuntimeBuiltin},
		{name: "selfhost", mode: RuntimeSelfHost},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var want *struct {
				stdout string
				exit   int
			}
			for _, rt := range runtimes {
				rt := rt
				t.Run(rt.name, func(t *testing.T) {
					stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
						t,
						tt.src,
						BuildOptions{Runtime: rt.mode},
						250*time.Millisecond,
					)
					if timedOut {
						t.Fatalf("program timed out for runtime %s", rt.name)
					}
					got := struct {
						stdout string
						exit   int
					}{stdout: stdout, exit: exitCode}
					if want == nil {
						want = &got
					} else if got != *want {
						t.Fatalf("runtime parity mismatch: got=%#v want=%#v", got, *want)
					}
					if stdout != "" {
						t.Fatalf("stdout mismatch: %q", stdout)
					}
					if exitCode != tt.exit {
						t.Fatalf("exit code = %d, want %d", exitCode, tt.exit)
					}
				})
			}
		})
	}
}

func TestWaitCompositionRuntimeSelfHostBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses actors, runtime:
    let _sleep: Int = core.sleep_ms(1)
    let _sent: Int = core.send_msg(core.sender(), 13, 5)
    return 21

func main() -> Int
uses actors, runtime:
    let deadline: Int = core.deadline_ms(1)
    if core.timer_ready(deadline):
        return 10
    let task: task.i32 = core.task_spawn_i32("worker")
    let early: task.result_i32 = core.task_poll_i32(task)
    if early.error != 2:
        return 20 + early.error
    let yielded: Int = core.yield()
    if yielded != 0:
        return 40 + yielded
    let empty: actor.recv_result_i32 = core.recv_poll()
    if empty.error != 2:
        return 60 + empty.error
    let msg: actor.recv_msg_result = core.recv_msg_until(core.deadline_ms(3))
    if msg.error != 0:
        return 80 + msg.error
    if msg.value != 13:
        return 100 + msg.value
    if msg.tag != 5:
        return 120 + msg.tag
    if !core.timer_ready(deadline):
        return 140
    let selected: task.result_i32 = core.select2_i32(task, core.deadline_ms(5))
    if selected.error != 0:
        return 160 + selected.error
    return selected.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{Runtime: RuntimeSelfHost},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf("program timed out; selfhost wait composition should wake on message/task events")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 21 {
		t.Fatalf("exit code = %d, want selected task value 21", exitCode)
	}
}

func TestDeadlineAwareRuntimeSelfHostBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let _sleepUntil: Int = core.sleep_until(core.deadline_ms(1))
    let task: task.i32 = core.task_spawn_i32("worker")
    let early: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(2))
    if early.error != 2:
        return 20 + early.error
    let final: task.result_i32 = core.task_join_result_i32(task)
    if final.error != 0:
        return 40 + final.error
    return final.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{Runtime: RuntimeSelfHost},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf("program timed out; selfhost deadline-aware waits should advance logical time")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 6 {
		t.Fatalf("exit code = %d, want selfhost final wake time 6", exitCode)
	}
}

func TestTaskJoinWaitStateAllowsSleepingTaskDeadlineBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf(
			"program timed out; task_join_i32 should wait without starving sleeping task deadline",
		)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code = %d, want joined sleeping task wake time 5", exitCode)
	}
}

func TestTaskJoinWaitStateSelfHostRuntimeBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{Runtime: RuntimeSelfHost},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf("program timed out; selfhost task_join_i32 should park while child sleeps")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code = %d, want selfhost joined sleeping task wake time 5", exitCode)
	}
}

func TestTaskDeadlineBuiltinSelfHostParityBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let result: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(5))
    if result.error != 0:
        return 20 + result.error
    return result.value
`
	var want *struct {
		stdout string
		exit   int
	}
	for _, tc := range []struct {
		name string
		rt   RuntimeMode
	}{
		{name: "builtin", rt: RuntimeBuiltin},
		{name: "selfhost", rt: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
				t,
				src,
				BuildOptions{Runtime: tc.rt},
				250*time.Millisecond,
			)
			if timedOut {
				t.Fatalf("program timed out; deadline join should complete under runtime %d", tc.rt)
			}
			got := struct {
				stdout string
				exit   int
			}{stdout: stdout, exit: exitCode}
			if want == nil {
				want = &got
			} else if got != *want {
				t.Fatalf("runtime parity mismatch: got=%#v want=%#v", got, *want)
			}
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != 2 {
				t.Fatalf("exit code = %d, want deadline join value 2", exitCode)
			}
		})
	}
}

func TestTaskJoinResultWaitStateAllowsSleepingTaskDeadlineBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(3)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    if result.error != 0:
        return 20 + result.error
    return result.value
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf(
			"program timed out; task_join_result_i32 should wait without starving sleeping task deadline",
		)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 3 {
		t.Fatalf("exit code = %d, want joined result wake time 3", exitCode)
	}
}

func TestTaskJoinTypedWaitStateAllowsSleepingThrowBuildAndRun(t *testing.T) {
	src := `
enum WaitErr:
    case boom(Int)
    case stopped

func worker() -> Int throws WaitErr
uses runtime:
    let _sleep: Int = core.sleep_ms(4)
    throw WaitErr.boom(core.time_now_ms())

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<WaitErr>("worker")
    return catch core.task_join_i32_typed<WaitErr>(task):
    case WaitErr.boom(code):
        code
    case WaitErr.stopped:
        99
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf(
			"program timed out; typed task join should wait without starving sleeping task deadline",
		)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 4 {
		t.Fatalf("exit code = %d, want typed join throw payload 4", exitCode)
	}
}

func TestRuntimeSchedulerCanceledSleepingTaskReturnsCancelBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(10)
    let checkpoint: task.error = core.task_checkpoint()
    if checkpoint != 0:
        return core.time_now_ms()
    return 99

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let _delay: Int = core.sleep_ms(2)
    group = core.task_group_cancel(group)
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if result.value != 0:
        return result.value
    if result.error != 1:
        return 20 + result.error
    let now: Int = core.time_now_ms()
    if now != 2:
        return 40 + now
    return result.error
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want canceled task error 1 at logical time 2", exitCode)
	}
}

// ---- task_runtime_group_cancel_test.go ----

func TestTaskGroupLowersToRuntimeCalls(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    group = core.task_group_cancel(group)
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    return result.error
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
	for _, name := range []string{
		"__tetra_task_group_open",
		"__tetra_task_spawn_group_i32",
		"__tetra_task_group_cancel",
		"__tetra_task_group_close",
	} {
		if !hasIRCall(mainFn, name) {
			t.Fatalf("main does not call %s: %#v", name, mainFn.Instrs)
		}
	}
}

func TestTaskGroupCancelAfterSpawnBeforeJoinBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int:
    return 77

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    group = core.task_group_cancel(group)
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if result.value != 0:
        return result.value
    return result.error
`
	stdout, exitCode := buildAndRunWithOptions(t, src, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want canceled status 1", exitCode)
	}
}

func TestTaskGroupCancelWakesJoinUntilBeforeDeadlineBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(10)
    return 99

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let _delay: Int = core.sleep_ms(2)
    group = core.task_group_cancel(group)
    let result: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(20))
    let _closed: Int = core.task_group_close(group)
    if result.value != 0:
        return result.value
    if result.error != 1:
        return 30 + result.error
    let now: Int = core.time_now_ms()
    if now != 2:
        return 50 + now
    return result.error
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf("program timed out; canceled grouped task should wake join_until before deadline")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want canceled task error 1 at logical time 2", exitCode)
	}
}

func TestTaskGroupCancelWhileActorWaitsOnJoinReturnsCanceledBuildAndRun(t *testing.T) {
	src := `
func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 99

func canceller() -> Int
uses runtime:
    let _delay: Int = core.sleep_ms(2)
    let group: task.group = core.task_group_current()
    let _canceled: task.group = core.task_group_cancel(group)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let slow_task: task.i32 = core.task_spawn_group_i32(group, "slow")
    let _cancel_task: task.i32 = core.task_spawn_group_i32(group, "canceller")
    let result: task.result_i32 = core.task_join_result_i32(slow_task)
    let _closed: Int = core.task_group_close(group)
    if result.value != 0:
        return result.value
    if result.error != 1:
        return 30 + result.error
    let now: Int = core.time_now_ms()
    if now != 2:
        return 50 + now
    return result.error
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf(
			"program timed out; task group cancel should wake actor waiting on task_join_result_i32",
		)
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf(
			"exit code = %d, want join_result_i32 canceled error 1 while caller was already waiting",
			exitCode,
		)
	}
}

func TestTaskGroupCancelWhileActorWaitsOnJoinI32WakesWithZeroValueBuildAndRun(t *testing.T) {
	src := `
func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 99

func canceller() -> Int
uses runtime:
    let _delay: Int = core.sleep_ms(2)
    let group: task.group = core.task_group_current()
    let _canceled: task.group = core.task_group_cancel(group)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let slow_task: task.i32 = core.task_spawn_group_i32(group, "slow")
    let _cancel_task: task.i32 = core.task_spawn_group_i32(group, "canceller")
    let joined: Int = core.task_join_i32(slow_task)
    let _closed: Int = core.task_group_close(group)
    if joined != 0:
        return joined
    let now: Int = core.time_now_ms()
    if now != 2:
        return 50 + now
    return 1
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf("program timed out; task group cancel should wake actor waiting on task_join_i32")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf(
			"exit code = %d, want task_join_i32 to wake at cancellation time with raw zero value",
			exitCode,
		)
	}
}

func TestTaskGroupCancelWakesSelect2BeforeDeadlineBuildAndRun(t *testing.T) {
	src := `
func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(100)
    return 99

func canceller() -> Int
uses runtime:
    let _delay: Int = core.sleep_ms(2)
    let group: task.group = core.task_group_current()
    let _canceled: task.group = core.task_group_cancel(group)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let slow_task: task.i32 = core.task_spawn_group_i32(group, "slow")
    let _cancel_task: task.i32 = core.task_spawn_group_i32(group, "canceller")
    let selected: task.result_i32 = core.select2_i32(slow_task, core.deadline_ms(100))
    let _closed: Int = core.task_group_close(group)
    if selected.value != 0:
        return selected.value
    if selected.error != 1:
        return 30 + selected.error
    let now: Int = core.time_now_ms()
    if now != 2:
        return 50 + now
    return selected.error
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf("program timed out; task group cancel should wake select2_i32 before deadline")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want select2_i32 canceled error 1 before deadline", exitCode)
	}
}

func TestTaskGroupJoinUntilTimeoutThenCancelFinalJoinBuildAndRun(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(10)
    return 99

func main() -> Int
uses runtime:
    var group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let early: task.result_i32 = core.task_join_until_i32(task, core.deadline_ms(3))
    if early.error != 2:
        return 20 + early.error
    if early.value != 0:
        return 40 + early.value
    let atTimeout: Int = core.time_now_ms()
    if atTimeout != 3:
        return 60 + atTimeout
    group = core.task_group_cancel(group)
    let final: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if final.value != 0:
        return final.value
    if final.error != 1:
        return 80 + final.error
    return final.error
`
	stdout, exitCode, timedOut := buildAndRunWithOptionsTimeout(
		t,
		src,
		BuildOptions{},
		250*time.Millisecond,
	)
	if timedOut {
		t.Fatalf("program timed out; final join should observe cancellation after prior timeout")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want canceled final join after timeout", exitCode)
	}
}

func TestTaskGroupCurrentLowersToRuntimeCall(t *testing.T) {
	src := []byte(`
func main() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    if group == group:
        return 0
    return 1
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
	if !hasIRCall(mainFn, "__tetra_task_group_current") {
		t.Fatalf("main does not call __tetra_task_group_current: %#v", mainFn.Instrs)
	}
}

func TestTaskCancellationCheckpointLowersToRuntimeCalls(t *testing.T) {
	src := []byte(`
func main() -> Int
uses runtime:
    let canceled: Int = core.task_is_canceled()
    let checkpoint: task.error = core.task_checkpoint()
    return canceled + checkpoint
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
	for _, name := range []string{"__tetra_task_is_canceled", "__tetra_task_checkpoint"} {
		if !hasIRCall(mainFn, name) {
			t.Fatalf("main does not call %s: %#v", name, mainFn.Instrs)
		}
	}
}

func TestTimeRuntimeBuiltinsRequireRuntimeUse(t *testing.T) {
	src := []byte(`
func main() -> Int:
    return core.time_now_ms()
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected missing runtime effect error")
	}
	if !strings.Contains(err.Error(), "uses effect 'runtime'") {
		t.Fatalf("error = %v", err)
	}
}

// ---- task_runtime_helpers_test.go ----

func buildAndRunWithOptionsTimeout(
	t *testing.T,
	src string,
	opt BuildOptions,
	timeout time.Duration,
) (string, int, bool) {
	t.Helper()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")

	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", opt); err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := verifyELF(outPath); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, outPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return out.String(), -1, true
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return out.String(), exitErr.ProcessState.ExitCode(), false
		}
		t.Fatalf("run binary: %v", err)
	}
	return out.String(), cmd.ProcessState.ExitCode(), false
}

func runBinaryOrSkipUnsupportedTarget(t *testing.T, path string) (string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("target binary timed out after 2s: %s output=%q", path, out.String())
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() &&
				status.Signal() == syscall.SIGSYS {
				t.Skipf(
					("host kernel rejected target binary %s with SIGSYS; target " +
						"execution is unsupported in this environment"),
					path,
				)
			}
			return out.String(), exitErr.ProcessState.ExitCode()
		}
		text := err.Error() + " " + out.String()
		if strings.Contains(text, "exec format error") ||
			strings.Contains(text, "no such file or directory") {
			t.Skipf("host cannot execute target binary %s: %v", path, err)
		}
		t.Fatalf("run binary: %v", err)
	}
	return out.String(), cmd.ProcessState.ExitCode()
}

func findIRFunc(t *testing.T, funcs []IRFunc, name string) IRFunc {
	t.Helper()
	for _, fn := range funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("missing IR function %q", name)
	return IRFunc{}
}

func hasIRCall(fn IRFunc, name string) bool {
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == name {
			return true
		}
	}
	return false
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func hasPrefix(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func hasIRFuncPrefix(funcs []IRFunc, prefix string) bool {
	for _, fn := range funcs {
		if strings.HasPrefix(fn.Name, prefix) {
			return true
		}
	}
	return false
}

// ---- task_runtime_lowering_test.go ----

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
	_, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"linux-x64",
		BuildOptions{Runtime: RuntimeSelfHost},
	)
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
	_, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"linux-x64",
		BuildOptions{Runtime: RuntimeSelfHost},
	)
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
			_, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				"linux-x64",
				BuildOptions{Runtime: RuntimeSelfHost},
			)
			if err == nil {
				t.Fatalf("expected explicit selfhost typed task rejection")
			}
			if !strings.Contains(
				err.Error(),
				"self-host runtime does not support typed task handles",
			) {
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

// ---- task_runtime_targets_test.go ----

func TestX32ExecutableBuildsAutoSelfHostTimeRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "time_x32.tetra")
	outPath := filepath.Join(tmp, "time-x32")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses runtime:
    return core.time_now_ms()
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1})
	if err != nil {
		t.Fatalf("build x32 auto self-host time runtime: %v", err)
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

func TestX86TimeRuntimeBuildsAndRunsWhenHostSupportsI386(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "time_x86.tetra")
	outPath := filepath.Join(tmp, "time-x86")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    let _until: Int = core.sleep_until(core.deadline_ms(2))
    return core.time_now_ms()
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1})
	if err != nil {
		t.Fatalf("build x86 time runtime: %v", err)
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
	if code != 7 {
		t.Fatalf("x86 time runtime exit=%d stdout=%q, want 7", code, stdout)
	}
}

func TestX86SingleTaskRuntimeBuildsAndRunsWhenHostSupportsI386(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "task_x86.tetra")
	outPath := filepath.Join(tmp, "task-x86")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 single-task auto self-host runtime: %v", err)
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
	if code != 41 {
		t.Fatalf("x86 single-task runtime exit=%d stdout=%q, want 41", code, stdout)
	}
}

func TestX86TypedTaskRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(23)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "typed_task_x86.tetra")
			outPath := filepath.Join(tmp, "typed-task-x86")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				"x86",
				BuildOptions{Jobs: 1, Runtime: tc.runtime},
			); err != nil {
				t.Fatalf("build x86 typed-task %s self-host runtime: %v", tc.name, err)
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
			if code != 23 {
				t.Fatalf("x86 typed-task runtime exit=%d stdout=%q, want 23", code, stdout)
			}
		})
	}
}

func TestX86StagedTypedTaskRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
enum TaskErr:
    case boom(Int, Int, Int, Int, Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(1, 2, 3, 4, 5)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b, c, d, e):
        a + b + c + d + e
    case TaskErr.stopped:
        99
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "staged_typed_task_x86.tetra")
			outPath := filepath.Join(tmp, "staged-typed-task-x86")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				"x86",
				BuildOptions{Jobs: 1, Runtime: tc.runtime},
			); err != nil {
				t.Fatalf("build x86 staged typed-task %s self-host runtime: %v", tc.name, err)
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
			if code != 15 {
				t.Fatalf("x86 staged typed-task runtime exit=%d stdout=%q, want 15", code, stdout)
			}
		})
	}
}

func TestX86SingleTaskGroupRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    let status: Int = core.task_group_status(group)
    if status != 1:
        return 60 + status
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let before: Int = core.task_group_status(group)
    if before != 1:
        return 30 + before
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 80 + closeError
    let after: Int = core.task_group_status(group)
    if after != 3:
        return 90 + after
    if result.error != 0:
        return 100 + result.error
    return result.value
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "task_group_x86.tetra")
			outPath := filepath.Join(tmp, "task-group-x86")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				"x86",
				BuildOptions{Jobs: 1, Runtime: tc.runtime},
			); err != nil {
				t.Fatalf("build x86 task-group %s self-host runtime: %v", tc.name, err)
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
			if code != 7 {
				t.Fatalf("x86 task-group runtime exit=%d stdout=%q, want 7", code, stdout)
			}
		})
	}
}

func TestX86MultiSpawnTaskRuntimeBuildsSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "task_multi_spawn_x86.tetra")
	outPath := filepath.Join(tmp, "task-multi-spawn-x86")
	if err := os.WriteFile(srcPath, []byte(`
func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return core.time_now_ms()

func fast() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let slow_task: task.i32 = core.task_spawn_i32("slow")
    let fast_task: task.i32 = core.task_spawn_i32("fast")
    let fast_result: task.result_i32 = core.task_join_result_i32(fast_task)
    if fast_result.error != 0:
        return 20 + fast_result.error
    if fast_result.value != 2:
        return 40 + fast_result.value
    let slow_value: Int = core.task_join_i32(slow_task)
    return fast_result.value * 10 + slow_value
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 two-spawn task self-host runtime: %v", err)
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
	if code != 25 {
		t.Fatalf("x86 two-spawn task runtime exit=%d stdout=%q, want 25", code, stdout)
	}
}

func TestX86MultiSpawnTaskGroupRuntimeBuildsSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "task_group_multi_spawn_x86.tetra")
	outPath := filepath.Join(tmp, "task-group-multi-spawn-x86")
	if err := os.WriteFile(srcPath, []byte(`
func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return core.time_now_ms()

func fast() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    if core.task_group_status(group) != 1:
        return 70 + core.task_group_status(group)
    let _sleep: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let slow_task: task.i32 = core.task_spawn_group_i32(group, "slow")
    let fast_task: task.i32 = core.task_spawn_group_i32(group, "fast")
    let fast_result: task.result_i32 = core.task_join_result_i32(fast_task)
    if fast_result.error != 0:
        return 20 + fast_result.error
    if fast_result.value != 2:
        return 40 + fast_result.value
    let slow_value: Int = core.task_join_i32(slow_task)
    let close_error: Int = core.task_group_close(group)
    if close_error != 0:
        return 90 + close_error
    if core.task_group_status(group) != 3:
        return 100 + core.task_group_status(group)
    return fast_result.value * 10 + slow_value
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 two-spawn task-group self-host runtime: %v", err)
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
	if code != 25 {
		t.Fatalf("x86 two-spawn task-group runtime exit=%d stdout=%q, want 25", code, stdout)
	}
}

func TestX86TypedTaskGroupRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr uses runtime:
    let group: task.group = core.task_group_current()
    let status: Int = core.task_group_status(group)
    if status != 1:
        throw TaskErr.boom(60 + status)
    throw TaskErr.boom(23)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let before: Int = core.task_group_status(group)
    if before != 1:
        return 30 + before
    let task = core.task_spawn_group_i32_typed<TaskErr>(group, "worker")
    let result: Int = catch core.task_join_group_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 80 + closeError
    let after: Int = core.task_group_status(group)
    if after != 3:
        return 90 + after
    return result
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "typed_task_group_x86.tetra")
			outPath := filepath.Join(tmp, "typed-task-group-x86")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				"x86",
				BuildOptions{Jobs: 1, Runtime: tc.runtime},
			); err != nil {
				t.Fatalf("build x86 typed task-group %s self-host runtime: %v", tc.name, err)
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
			if code != 23 {
				t.Fatalf("x86 typed task-group runtime exit=%d stdout=%q, want 23", code, stdout)
			}
		})
	}
}

func TestX32MultiSpawnTaskRuntimeBuildsSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "task_multi_spawn_x32.tetra")
	outPath := filepath.Join(tmp, "task-multi-spawn-x32")
	if err := os.WriteFile(srcPath, []byte(`
func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return core.time_now_ms()

func fast() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let slow_task: task.i32 = core.task_spawn_i32("slow")
    let fast_task: task.i32 = core.task_spawn_i32("fast")
    let fast_result: task.result_i32 = core.task_join_result_i32(fast_task)
    if fast_result.error != 0:
        return 20 + fast_result.error
    if fast_result.value != 2:
        return 40 + fast_result.value
    let slow_value: Int = core.task_join_i32(slow_task)
    return fast_result.value * 10 + slow_value
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x32 two-spawn task self-host runtime: %v", err)
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
	if code != 25 {
		t.Fatalf("x32 two-spawn task runtime exit=%d stdout=%q, want 25", code, stdout)
	}
}

func TestX32MultiSpawnTypedTaskRuntimeBuildsSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "typed_task_multi_spawn_x32.tetra")
	outPath := filepath.Join(tmp, "typed-task-multi-spawn-x32")
	if err := os.WriteFile(srcPath, []byte(`
enum TaskErr:
    case boom(Int)
    case stopped

func slow() -> Int throws TaskErr uses runtime:
    let _sleep: Int = core.sleep_ms(4)
    throw TaskErr.boom(11)

func fast() -> Int throws TaskErr uses runtime:
    let _sleep: Int = core.sleep_ms(1)
    throw TaskErr.boom(7)

func main() -> Int
uses runtime:
    let slow_task = core.task_spawn_i32_typed<TaskErr>("slow")
    let fast_task = core.task_spawn_i32_typed<TaskErr>("fast")
    let fast_value: Int = catch core.task_join_i32_typed<TaskErr>(fast_task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        90
    let slow_value: Int = catch core.task_join_i32_typed<TaskErr>(slow_task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        91
    return fast_value * 10 + slow_value
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x32 two-spawn typed-task self-host runtime: %v", err)
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
	if code != 81 {
		t.Fatalf("x32 two-spawn typed-task runtime exit=%d stdout=%q, want 81", code, stdout)
	}
}

func TestX32MultiSpawnTaskGroupRuntimeBuildsSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "task_group_multi_spawn_x32.tetra")
	outPath := filepath.Join(tmp, "task-group-multi-spawn-x32")
	if err := os.WriteFile(srcPath, []byte(`
func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return core.time_now_ms()

func fast() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    if core.task_group_status(group) != 1:
        return 70 + core.task_group_status(group)
    let _sleep: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let slow_task: task.i32 = core.task_spawn_group_i32(group, "slow")
    let fast_task: task.i32 = core.task_spawn_group_i32(group, "fast")
    let fast_result: task.result_i32 = core.task_join_result_i32(fast_task)
    if fast_result.error != 0:
        return 20 + fast_result.error
    if fast_result.value != 2:
        return 40 + fast_result.value
    let slow_value: Int = core.task_join_i32(slow_task)
    let close_error: Int = core.task_group_close(group)
    if close_error != 0:
        return 90 + close_error
    if core.task_group_status(group) != 3:
        return 100 + core.task_group_status(group)
    return fast_result.value * 10 + slow_value
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x32 two-spawn task-group self-host runtime: %v", err)
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
	if code != 25 {
		t.Fatalf("x32 two-spawn task-group runtime exit=%d stdout=%q, want 25", code, stdout)
	}
}

func TestX32TypedTaskGroupRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr uses runtime:
    let group: task.group = core.task_group_current()
    let status: Int = core.task_group_status(group)
    if status != 1:
        throw TaskErr.boom(60 + status)
    throw TaskErr.boom(23)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let before: Int = core.task_group_status(group)
    if before != 1:
        return 30 + before
    let task = core.task_spawn_group_i32_typed<TaskErr>(group, "worker")
    let result: Int = catch core.task_join_group_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 80 + closeError
    let after: Int = core.task_group_status(group)
    if after != 3:
        return 90 + after
    return result
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "typed_task_group_x32.tetra")
			outPath := filepath.Join(tmp, "typed-task-group-x32")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				"x32",
				BuildOptions{Jobs: 1, Runtime: tc.runtime},
			); err != nil {
				t.Fatalf("build x32 typed task-group %s self-host runtime: %v", tc.name, err)
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
			stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
			if code != 23 {
				t.Fatalf("x32 typed task-group runtime exit=%d stdout=%q, want 23", code, stdout)
			}
		})
	}
}

func TestX32SingleTaskRuntimeBuildsAutoSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "task_single_x32.tetra")
	outPath := filepath.Join(tmp, "task-single-x32")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1})
	if err != nil {
		t.Fatalf("build x32 single-task auto self-host runtime: %v", err)
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

func TestX32TypedTaskRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(23)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "typed_task_x32.tetra")
			outPath := filepath.Join(tmp, "typed-task-x32")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				"x32",
				BuildOptions{Jobs: 1, Runtime: tc.runtime},
			); err != nil {
				t.Fatalf("build x32 typed-task %s self-host runtime: %v", tc.name, err)
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
			stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
			if code != 23 {
				t.Fatalf("x32 typed-task runtime exit=%d stdout=%q, want 23", code, stdout)
			}
		})
	}
}

func TestX32StagedTypedTaskRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
enum TaskErr:
    case boom(Int, Int, Int, Int, Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(1, 2, 3, 4, 5)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b, c, d, e):
        a + b + c + d + e
    case TaskErr.stopped:
        99
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "staged_typed_task_x32.tetra")
			outPath := filepath.Join(tmp, "staged-typed-task-x32")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				"x32",
				BuildOptions{Jobs: 1, Runtime: tc.runtime},
			); err != nil {
				t.Fatalf("build x32 staged typed-task %s self-host runtime: %v", tc.name, err)
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
			stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
			if code != 15 {
				t.Fatalf("x32 staged typed-task runtime exit=%d stdout=%q, want 15", code, stdout)
			}
		})
	}
}

func TestX32SingleTaskGroupRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    let status: Int = core.task_group_status(group)
    if status != 1:
        return 60 + status
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let before: Int = core.task_group_status(group)
    if before != 1:
        return 30 + before
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 80 + closeError
    let after: Int = core.task_group_status(group)
    if after != 3:
        return 90 + after
    if result.error != 0:
        return 100 + result.error
    return result.value
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "task_group_x32.tetra")
			outPath := filepath.Join(tmp, "task-group-x32")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				"x32",
				BuildOptions{Jobs: 1, Runtime: tc.runtime},
			); err != nil {
				t.Fatalf("build x32 task-group %s self-host runtime: %v", tc.name, err)
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
			stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
			if code != 7 {
				t.Fatalf("x32 task-group runtime exit=%d stdout=%q, want 7", code, stdout)
			}
		})
	}
}

func TestX32ExplicitBuiltinRuntimeStillRejects(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "time_x32_builtin.tetra")
	outPath := filepath.Join(tmp, "time-x32-builtin")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses runtime:
    return core.time_now_ms()
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"x32",
		BuildOptions{Jobs: 1, Runtime: RuntimeBuiltin},
	)
	if err == nil {
		t.Fatalf("expected x32 builtin runtime support diagnostic")
	}
	for _, want := range []string{
		"builtin runtime is not supported on target linux-x32",
		"runtime=selfhost",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
	if _, statErr := os.Stat(outPath); statErr == nil {
		t.Fatalf("x32 builtin runtime rejection wrote executable %s", outPath)
	}
}

// ---- task_runtime_test.go ----

func TestSelectRuntimeModeStabilizationMatrix(t *testing.T) {
	for _, tc := range []struct {
		name      string
		requested RuntimeMode
		usage     runtimeUsageProfile
		want      RuntimeMode
		wantErr   string
	}{
		{
			name:      "auto_actor_only_uses_selfhost",
			requested: RuntimeAuto,
			want:      RuntimeSelfHost,
		},
		{
			name:      "auto_multi_spawn_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{actorSpawnCount: 2},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_actor_state_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{actorStateUsed: true},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_task_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{tasksUsed: true},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_task_group_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{taskGroupsUsed: true},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_typed_task_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 4},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_staged_typed_task_slots_use_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 8},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_time_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{timeRuntimeUsed: true},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_filesystem_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{filesystemUsed: true},
			want:      RuntimeBuiltin,
		},
		{
			name:      "auto_networking_uses_builtin",
			requested: RuntimeAuto,
			usage:     runtimeUsageProfile{netUsed: true},
			want:      RuntimeBuiltin,
		},
		{
			name:      "explicit_selfhost_actor_only_allowed",
			requested: RuntimeSelfHost,
			want:      RuntimeSelfHost,
		},
		{
			name:      "explicit_selfhost_rejects_typed_tasks",
			requested: RuntimeSelfHost,
			usage:     runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 4},
			wantErr:   "self-host runtime does not support typed task handles",
		},
		{
			name:      "explicit_selfhost_rejects_multi_spawn",
			requested: RuntimeSelfHost,
			usage:     runtimeUsageProfile{actorSpawnCount: 2},
			wantErr:   "self-host runtime supports at most one spawned actor",
		},
		{
			name:      "explicit_builtin_allowed",
			requested: RuntimeBuiltin,
			usage: runtimeUsageProfile{
				typedTasksUsed:    true,
				typedTaskMaxSlots: 8,
				timeRuntimeUsed:   true,
			},
			want: RuntimeBuiltin,
		},
		{
			name:      "invalid_runtime_rejected",
			requested: RuntimeMode(99),
			wantErr:   "unsupported runtime mode: 99",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := selectRuntimeMode(tc.requested, tc.usage)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want contains %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("selectRuntimeMode returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("selectRuntimeMode = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRuntimeModeForLinuxX32AutoUsesSelfHostWhenSupported(t *testing.T) {
	usage := runtimeUsageProfile{timeRuntimeUsed: true}
	got, err := runtimeModeForNativeTarget("linux-x32", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 auto runtime mode = %v, want self-host for supported usage", got)
	}

	got, err = runtimeModeForNativeTarget("linux-x32", RuntimeBuiltin, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("explicit builtin runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeBuiltin {
		t.Fatalf(
			"x32 explicit builtin runtime mode = %v, want builtin to preserve explicit diagnostic",
			got,
		)
	}

	got, err = runtimeModeForNativeTarget("linux-x64", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x64 runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeBuiltin {
		t.Fatalf("x64 auto runtime mode = %v, want existing builtin preference", got)
	}

	usage = runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 4, actorSpawnCount: 1}
	got, err = runtimeModeForNativeTarget("linux-x32", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x32 typed task runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 typed task auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x32", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x32 explicit self-host typed task selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 explicit self-host typed task runtime mode = %v, want self-host", got)
	}

	usage = runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 4, actorSpawnCount: 2}
	got, err = runtimeModeForNativeTarget("linux-x32", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x32 multi-spawn typed task runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 multi-spawn typed task auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x32", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x32 explicit self-host multi-spawn typed task selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf(
			"x32 explicit self-host multi-spawn typed task runtime mode = %v, want self-host",
			got,
		)
	}

	usage = runtimeUsageProfile{taskGroupsUsed: true, actorSpawnCount: 1}
	got, err = runtimeModeForNativeTarget("linux-x32", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x32 task-group runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 task-group auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x32", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x32 explicit self-host task-group selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 explicit self-host task-group runtime mode = %v, want self-host", got)
	}

	usage = runtimeUsageProfile{
		taskGroupsUsed:    true,
		typedTasksUsed:    true,
		typedTaskMaxSlots: 4,
		actorSpawnCount:   1,
	}
	got, err = runtimeModeForNativeTarget("linux-x32", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x32 typed task-group runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 typed task-group auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x32", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x32 explicit self-host typed task-group selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x32 explicit self-host typed task-group runtime mode = %v, want self-host", got)
	}
}

func TestTargetSpecificSelfHostMultiSpawnOverrideStaysLinuxILP32Only(t *testing.T) {
	usage := runtimeUsageProfile{actorSpawnCount: 2}
	for _, target := range []string{"linux-x64", "macos-x64", "windows-x64"} {
		if got, err := selectRuntimeModeForNativeTarget(target, RuntimeSelfHost, usage); err == nil ||
			!strings.Contains(err.Error(), "self-host runtime supports at most one spawned actor") {
			t.Fatalf(
				"%s explicit self-host two-spawn selection = mode %v err %v, want generic one-spawn diagnostic",
				target,
				got,
				err,
			)
		}
	}
}

func TestRuntimeModeForLinuxX86AutoUsesSelfHostWhenSupported(t *testing.T) {
	usage := runtimeUsageProfile{taskGroupsUsed: true, actorSpawnCount: 1}
	got, err := runtimeModeForNativeTarget("linux-x86", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x86 task-group runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 task-group auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x86", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x86 explicit self-host task-group selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 explicit self-host task-group runtime mode = %v, want self-host", got)
	}

	usage = runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 4, actorSpawnCount: 1}
	got, err = runtimeModeForNativeTarget("linux-x86", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x86 typed task runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 typed task auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x86", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x86 explicit self-host typed task selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 explicit self-host typed task runtime mode = %v, want self-host", got)
	}

	usage = runtimeUsageProfile{typedTasksUsed: true, typedTaskMaxSlots: 8, actorSpawnCount: 1}
	got, err = runtimeModeForNativeTarget("linux-x86", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x86 staged typed task runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 staged typed task auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x86", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x86 explicit self-host staged typed task selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 explicit self-host staged typed task runtime mode = %v, want self-host", got)
	}

	usage = runtimeUsageProfile{taskGroupsUsed: true, actorSpawnCount: 2}
	got, err = runtimeModeForNativeTarget("linux-x86", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x86 multi-spawn task-group runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 multi-spawn task-group auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x86", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x86 explicit self-host multi-spawn task-group selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf(
			"x86 explicit self-host multi-spawn task-group runtime mode = %v, want self-host",
			got,
		)
	}

	usage = runtimeUsageProfile{
		taskGroupsUsed:    true,
		typedTasksUsed:    true,
		typedTaskMaxSlots: 4,
		actorSpawnCount:   1,
	}
	got, err = runtimeModeForNativeTarget("linux-x86", RuntimeAuto, RuntimeBuiltin, usage)
	if err != nil {
		t.Fatalf("x86 typed task-group runtimeModeForNativeTarget: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 typed task-group auto runtime mode = %v, want self-host", got)
	}
	got, err = selectRuntimeModeForNativeTarget("linux-x86", RuntimeSelfHost, usage)
	if err != nil {
		t.Fatalf("x86 explicit self-host typed task-group selection: %v", err)
	}
	if got != RuntimeSelfHost {
		t.Fatalf("x86 explicit self-host typed task-group runtime mode = %v, want self-host", got)
	}
}

func TestNativeRuntimeCapabilityTableDocumentsCurrentLinuxFamily(t *testing.T) {
	x64 := nativeRuntimeCapabilitiesForTarget("linux-x64")
	if !x64.actors || !x64.actorState || !x64.tasks || !x64.taskGroups || !x64.typedTasks ||
		!x64.time ||
		!x64.filesystem ||
		!x64.networking ||
		!x64.surface ||
		!x64.distributedActors ||
		x64.maxActorSpawns != unlimitedActorSpawns ||
		!x64.builtinRuntime ||
		!x64.selfHostActorsRuntime {
		t.Fatalf("linux-x64 runtime capabilities = %#v", x64)
	}

	x32 := nativeRuntimeCapabilitiesForTarget("linux-x32")
	if !x32.actors || !x32.actorState || !x32.tasks || !x32.taskGroups || !x32.typedTasks ||
		!x32.time ||
		!x32.filesystem ||
		!x32.networking ||
		x32.surface ||
		x32.distributedActors ||
		x32.maxActorSpawns != 2 ||
		x32.maxTypedTaskSlots != 8 ||
		x32.builtinRuntime ||
		!x32.selfHostActorsRuntime {
		t.Fatalf("linux-x32 runtime capabilities = %#v", x32)
	}

	x86 := nativeRuntimeCapabilitiesForTarget("linux-x86")
	if !x86.actors || !x86.actorState || !x86.tasks || !x86.taskGroups || !x86.typedTasks ||
		!x86.time ||
		!x86.timeOnlyWithoutScheduler ||
		!x86.filesystem ||
		!x86.networking ||
		x86.surface ||
		x86.distributedActors ||
		x86.maxActorSpawns != 2 ||
		x86.maxTypedTaskSlots != 8 ||
		x86.builtinRuntime ||
		!x86.selfHostActorsRuntime ||
		!x86.selfHostTimeRuntime {
		t.Fatalf("linux-x86 runtime capabilities = %#v", x86)
	}
}

func TestTaskSpawnI32CollectsRuntimeEntry(t *testing.T) {
	src := []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    if task.value != 1:
        return 60 + task.value
    if task.error != 0:
        return 70 + task.error
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
	used, entries, _, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collectActorEntries: %v", err)
	}
	if !used {
		t.Fatalf("task runtime was not collected")
	}
	if len(entries) != 2 || entries[0] != "main" || entries[1] != "worker" {
		t.Fatalf("entries = %#v, want [main worker]", entries)
	}
}

func TestRequiredTypedTaskRuntimeSymbolsSupportStagedSlotsUpToEight(t *testing.T) {
	base := map[string]struct{}{}
	for _, name := range requiredTypedTaskRuntimeSymbols(4) {
		base[name] = struct{}{}
	}
	for _, slots := range []int{2, 3, 4} {
		name := "__tetra_task_join_typed_" + strconv.Itoa(slots)
		if _, ok := base[name]; !ok {
			t.Fatalf("missing required typed task runtime symbol %q", name)
		}
	}
	if _, ok := base["__tetra_task_result_get"]; ok {
		t.Fatalf("non-staged symbol set should not require __tetra_task_result_get")
	}

	got := map[string]struct{}{}
	for _, name := range requiredTypedTaskRuntimeSymbols(8) {
		got[name] = struct{}{}
	}
	for _, slots := range []int{2, 3, 4, 5, 6, 7, 8} {
		name := "__tetra_task_join_typed_" + strconv.Itoa(slots)
		if _, ok := got[name]; !ok {
			t.Fatalf("missing required typed task runtime symbol %q", name)
		}
	}
	if _, ok := got["__tetra_task_result_get"]; !ok {
		t.Fatalf("missing required staged typed task runtime symbol %q", "__tetra_task_result_get")
	}
}

func TestRequiredTypedTaskRuntimeSymbolsClampToSupportedABIEnvelope(t *testing.T) {
	low := requiredTypedTaskRuntimeSymbols(0)
	if got, want := strings.Join(
		low,
		",",
	), strings.Join(
		requiredTypedTaskRuntimeSymbols(2),
		",",
	); got != want {
		t.Fatalf("low slot clamp = %q, want %q", got, want)
	}
	for _, forbidden := range []string{
		"__tetra_task_join_typed_0",
		"__tetra_task_join_typed_1",
		"__tetra_task_result_get",
	} {
		if containsString(low, forbidden) {
			t.Fatalf("low slot ABI unexpectedly contains %q: %#v", forbidden, low)
		}
	}

	high := requiredTypedTaskRuntimeSymbols(99)
	if got, want := strings.Join(
		high,
		",",
	), strings.Join(
		requiredTypedTaskRuntimeSymbols(8),
		",",
	); got != want {
		t.Fatalf("high slot clamp = %q, want %q", got, want)
	}
	for _, required := range []string{"__tetra_task_join_typed_8", "__tetra_task_result_get"} {
		if !containsString(high, required) {
			t.Fatalf("high slot ABI missing %q: %#v", required, high)
		}
	}
	if containsString(high, "__tetra_task_join_typed_9") {
		t.Fatalf("high slot ABI exceeded supported envelope: %#v", high)
	}
}

func TestValidateTypedTaskRuntimeObjectRejectsMissingStagedSymbols(t *testing.T) {
	obj := &Object{}
	for _, name := range requiredTypedTaskRuntimeSymbols(4) {
		obj.Symbols = append(obj.Symbols, Symbol{Name: name})
	}
	if err := validateTypedTaskRuntimeObject(obj, 8); err == nil {
		t.Fatalf("expected missing staged typed task runtime symbol failure")
	} else if !strings.Contains(err.Error(), "__tetra_task_result_get") {
		t.Fatalf("unexpected typed task runtime validation error: %v", err)
	}
}

func TestValidateTaskRuntimeObjectChecksSignatureMetadata(t *testing.T) {
	t.Run("correct metadata passes", func(t *testing.T) {
		obj := runtimeObjectWithTaskRuntimeSignatures()
		if err := validateTaskRuntimeObject(obj); err != nil {
			t.Fatalf("validate task runtime object: %v", err)
		}
	})

	t.Run("wrong arity fails", func(t *testing.T) {
		obj := runtimeObjectWithTaskRuntimeSignatures()
		replaceRuntimeSymbolSignature(obj, "__tetra_task_join_i32", 1, 1)
		err := validateTaskRuntimeObject(obj)
		if err == nil {
			t.Fatalf("expected wrong arity failure")
		}
		if !strings.Contains(
			err.Error(),
			"runtime object symbol '__tetra_task_join_i32' signature mismatch",
		) ||
			!strings.Contains(err.Error(), "params=1 want=2") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wrong return slot count fails", func(t *testing.T) {
		obj := runtimeObjectWithTaskRuntimeSignatures()
		replaceRuntimeSymbolSignature(obj, "__tetra_task_join_result_i32", 2, 1)
		err := validateTaskRuntimeObject(obj)
		if err == nil {
			t.Fatalf("expected wrong return slot failure")
		}
		if !strings.Contains(
			err.Error(),
			"runtime object symbol '__tetra_task_join_result_i32' signature mismatch",
		) ||
			!strings.Contains(err.Error(), "returns=1 want=2") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("legacy symbols without metadata remain compatible", func(t *testing.T) {
		obj := &Object{}
		for _, name := range requiredTaskRuntimeSymbols() {
			obj.Symbols = append(obj.Symbols, Symbol{Name: name})
		}
		if err := validateTaskRuntimeObject(obj); err != nil {
			t.Fatalf("legacy task runtime object should remain compatible: %v", err)
		}
	})
}

func runtimeObjectWithTaskRuntimeSignatures() *Object {
	obj := &Object{}
	for _, name := range requiredTaskRuntimeSymbols() {
		sig, ok := runtimeObjectSignature(name)
		if !ok {
			panic("missing task runtime signature for " + name)
		}
		obj.Symbols = append(obj.Symbols, Symbol{
			Name:         name,
			HasSignature: true,
			ParamSlots:   sig.paramSlots,
			ReturnSlots:  sig.returnSlots,
		})
	}
	return obj
}

func TestRuntimeObjectSignatureUsesSharedRuntimeABI(t *testing.T) {
	names := append([]string{}, requiredActorRuntimeSymbols()...)
	names = append(names, requiredActorStateRuntimeSymbols()...)
	names = append(names, requiredDistributedActorRuntimeSymbols()...)
	names = append(names, requiredTaskRuntimeSymbols()...)
	names = append(names, requiredTaskGroupRuntimeSymbols()...)
	names = append(names, requiredTypedTaskRuntimeSymbols(8)...)
	names = append(names, requiredTimeRuntimeSymbols()...)
	names = append(names, requiredFilesystemRuntimeSymbols()...)
	names = append(names, requiredSurfaceRuntimeSymbols()...)

	for _, name := range names {
		objectSig, ok := runtimeObjectSignature(name)
		if !ok {
			t.Fatalf("missing runtime object signature for %q", name)
		}
		sharedSig, ok := runtimeabi.SignatureForSymbol(name)
		if !ok {
			t.Fatalf("missing shared runtime ABI signature for %q", name)
		}
		if sharedSig.ParamSlots != objectSig.paramSlots ||
			sharedSig.ReturnSlots != objectSig.returnSlots {
			t.Fatalf(
				"%s ABI mismatch: shared params=%d returns=%d object params=%d returns=%d",
				name,
				sharedSig.ParamSlots,
				sharedSig.ReturnSlots,
				objectSig.paramSlots,
				objectSig.returnSlots,
			)
		}
	}
}

func replaceRuntimeSymbolSignature(obj *Object, name string, paramSlots int, returnSlots int) {
	for i := range obj.Symbols {
		if obj.Symbols[i].Name == name {
			obj.Symbols[i].HasSignature = true
			obj.Symbols[i].ParamSlots = paramSlots
			obj.Symbols[i].ReturnSlots = returnSlots
			return
		}
	}
	panic("missing runtime symbol " + name)
}

func TestRequiredTaskRuntimeSymbolsIncludeDeadlineAndCancellationABI(t *testing.T) {
	got := map[string]struct{}{}
	for _, name := range requiredTaskRuntimeSymbols() {
		got[name] = struct{}{}
	}
	for _, name := range []string{
		"__tetra_task_join_until_i32",
		"__tetra_task_poll_i32",
		"__tetra_task_is_canceled",
		"__tetra_task_checkpoint",
	} {
		if _, ok := got[name]; !ok {
			t.Fatalf(
				"required task runtime symbols missing deadline/cancellation ABI symbol %q",
				name,
			)
		}
	}
}

func TestRequiredTaskGroupRuntimeSymbolsIncludeCancellationABI(t *testing.T) {
	got := map[string]struct{}{}
	for _, name := range requiredTaskGroupRuntimeSymbols() {
		got[name] = struct{}{}
	}
	for _, name := range []string{
		"__tetra_task_group_open",
		"__tetra_task_group_close",
		"__tetra_task_group_cancel",
		"__tetra_task_group_current",
		"__tetra_task_group_status",
		"__tetra_task_spawn_group_i32",
	} {
		if _, ok := got[name]; !ok {
			t.Fatalf("required task group runtime symbols missing ABI symbol %q", name)
		}
	}
}

func TestValidateTaskGroupRuntimeObjectRejectsMissingCancellationSymbols(t *testing.T) {
	obj := &Object{}
	for _, name := range requiredTaskGroupRuntimeSymbols() {
		if name == "__tetra_task_group_cancel" {
			continue
		}
		obj.Symbols = append(obj.Symbols, Symbol{Name: name})
	}
	if err := validateTaskGroupRuntimeObject(obj); err == nil {
		t.Fatalf("expected missing task group cancellation symbol failure")
	} else if !strings.Contains(err.Error(), "__tetra_task_group_cancel") {
		t.Fatalf("unexpected task group runtime validation error: %v", err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// ---- tetra_bug_regression_test.go ----

func requireTetraBugLinuxAMD64(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
}

func requireTetraBugInterfaceOnlyBuild(t *testing.T, src string) {
	t.Helper()
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "out")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"linux-x64",
		BuildOptions{InterfaceOnly: true, Jobs: 4},
	); err != nil {
		t.Fatalf("interface-only build: %v", err)
	}
}

func TestTetraBug0001GenericInferenceDirectCallArgument(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func id<T>(x: T) -> T:
    return x

func value() -> Int:
    return 42

func main() -> Int:
    return id(value())
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0002ModuleExtensionStaticCall(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRunFiles(t, map[string]string{
		"app/main.tetra": `module app.main

struct Unit:
    value: Int

protocol Score:
    func score(self: Unit) -> Int

extension Unit:
    func score(self: Unit) -> Int:
        return self.value

impl Unit: Score

func main() -> Int:
    let unit: Unit = Unit(value: 42)
    return Unit.score(unit)
`,
	}, "app/main.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0003FunctionTypedStructFieldEnumPayload(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
struct Handler:
    cb: fn(Int) -> Int

enum Route:
    case direct(fn(Int) -> Int)
    case fallback

func main() -> Int:
    let offset: Int = 13
    let captured: ptr = fn(value: Int) -> Int:
        return value + offset
    let handler: Handler = Handler(cb: captured)
    let route: Route = Route.direct(handler.cb)
    match route:
    case Route.direct(route_cb):
        return route_cb(29)
    case Route.fallback:
        return 0
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0004FormatterPreservesFunctionTypedLocalAnnotation(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	src := `
struct Handler:
    cb: fn(Int) -> Int

enum Route:
    case direct(fn(Int) -> Int)
    case fallback

func main() -> Int:
    let offset: Int = 13
    let captured: ptr = fn(value: Int) -> Int:
        return value + offset
    let handler: Handler = Handler(cb: captured)
    let field_cb: fn(Int) -> Int = handler.cb
    let route: Route = Route.direct(field_cb)
    match route:
    case Route.direct(route_cb):
        return route_cb(29)
    case Route.fallback:
        return 0
`
	formatted, err := FormatSource([]byte(src), "callable_alias.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if !strings.Contains(string(formatted), "let field_cb: fn(Int) -> Int = handler.cb") {
		t.Fatalf("formatted source lost function-typed local annotation:\n%s", string(formatted))
	}
	stdout, code := buildAndRun(t, string(formatted))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0005ModuleActorEntrypointString(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRunFiles(t, map[string]string{
		"app/main.tetra": `module app.main

actor Router:
    func run() -> Int
    uses actors:
        let _sent: Int = core.send_msg(core.sender(), 42, 7)
        return 0

func main() -> Int
uses actors:
    let router: actor = core.spawn("Router.run")
    let _request: Int = core.send_msg(router, 41, 6)
    let reply: actor.msg = core.recv_msg()
    return reply.value
`,
	}, "app/main.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0006FormatterPreservesFunctionTypedGlobalAnnotation(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	src := `
var callback: fn(Int) -> Int = identity

func identity(value: Int) -> Int:
    return value

func main() -> Int:
    return callback(42)
`
	formatted, err := FormatSource([]byte(src), "global_fn_format.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if !strings.Contains(string(formatted), "var callback: fn(Int) -> Int = identity") {
		t.Fatalf("formatted source lost function-typed global annotation:\n%s", string(formatted))
	}
	stdout, code := buildAndRun(t, string(formatted))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0007DerivedPtrAddComposesProvenance(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        let base: ptr = core.alloc_bytes(16)
        let dst: ptr = core.ptr_add(base, 8, memory_cap)
        let dst_one: ptr = core.ptr_add(dst, 1, memory_cap)
        let _write: UInt8 = core.store_u8(dst_one, 42, memory_cap)
        return core.load_u8(dst_one, memory_cap)
    return 99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0008FormatterPreservesMutableActorState(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	src := `
actor Counter:
    var count: Int = 0
    func run() -> Int
    uses actors:
        count = count + 1
        let _reply: Int = core.send_msg(core.sender(), count, 1)
        return 0

func main() -> Int
uses actors:
    let counter: actor = core.spawn("Counter.run")
    let _request: Int = core.send_msg(counter, 0, 0)
    let reply: actor.msg = core.recv_msg()
    return reply.value
`
	formatted, err := FormatSource([]byte(src), "actor_state_var_format.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if !strings.Contains(string(formatted), "actor Counter:\n    var count: Int = 0") {
		t.Fatalf("formatted source lost mutable actor state:\n%s", string(formatted))
	}
	stdout, code := buildAndRun(t, string(formatted))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

func TestTetraBug0009DualBlockingRecvMsgFanIn(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code, timedOut := buildAndRunWithOptionsTimeout(t, `
actor Adder:
    func run() -> Int
    uses actors:
        let request: actor.msg = core.recv_msg()
        let _reply: Int = core.send_msg(core.sender(), request.value + 1, request.tag + 10)
        return 0

actor Multiplier:
    func run() -> Int
    uses actors:
        let request: actor.msg = core.recv_msg()
        let _reply: Int = core.send_msg(core.sender(), request.value * 2, request.tag + 20)
        return 0

func main() -> Int
uses actors:
    let adder: actor = core.spawn("Adder.run")
    let multiplier: actor = core.spawn("Multiplier.run")
    let _left: Int = core.send_msg(adder, 20, 1)
    let _right: Int = core.send_msg(multiplier, 21, 2)
    let first: actor.msg = core.recv_msg()
    let second: actor.msg = core.recv_msg()
    let value_total: Int = first.value + second.value
    let tag_total: Int = first.tag + second.tag
    if tag_total != 33:
        return tag_total
    if value_total == 63:
        return 0
    return value_total
`, BuildOptions{}, 500*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestTetraBug0010DualBlockingRecvValueFanIn(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code, timedOut := buildAndRunWithOptionsTimeout(t, `
func adder() -> Int
uses actors:
    let request: Int = core.recv()
    let _reply: Int = core.send(core.sender(), request + 1)
    return 0

func multiplier() -> Int
uses actors:
    let request: Int = core.recv()
    let _reply: Int = core.send(core.sender(), request * 2)
    return 0

func main() -> Int
uses actors:
    let adder_actor: actor = core.spawn("adder")
    let multiplier_actor: actor = core.spawn("multiplier")
    let _left: Int = core.send(adder_actor, 20)
    let _right: Int = core.send(multiplier_actor, 21)
    let first: Int = core.recv()
    let second: Int = core.recv()
    let total: Int = first + second
    if total == 63:
        return 0
    return total
`, BuildOptions{}, 500*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestTetraBug0011StructConstructorOptionalFieldCoercion(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
struct MaybeBox:
    value: Int?

func filled(value: Int) -> MaybeBox:
    return MaybeBox(value: value)

func main() -> Int:
    let box: MaybeBox = filled(42)
    if let value = box.value:
        return value
    return 0
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0012EnumConstructorOptionalPayloadCoercion(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
enum Route:
    case ready(Int?)
    case empty

func make_ready(value: Int) -> Route:
    return Route.ready(value)

func score(route: Route) -> Int:
    match route:
    case Route.ready(maybe):
        if let value = maybe:
            return value
        return 0
    case Route.empty:
        return 0

func main() -> Int:
    let route: Route = make_ready(42)
    return score(route)
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0013DerivedPtrParamLoopPreservesProvenance(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func copy_loop(dst: ptr, src: ptr, n: Int) -> Int
uses capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        var i: Int = 0
        while i < n:
            let sp: ptr = core.ptr_add(src, i, memory_cap)
            let dp: ptr = core.ptr_add(dst, i, memory_cap)
            let b: UInt8 = core.load_u8(sp, memory_cap)
            let _: UInt8 = core.store_u8(dp, b, memory_cap)
            i = i + 1
        return 0
    return 99

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        let source: ptr = core.alloc_bytes(8)
        let target: ptr = core.alloc_bytes(8)
        let source_two: ptr = core.ptr_add(source, 2, memory_cap)
        let source_three: ptr = core.ptr_add(source, 3, memory_cap)
        let target_one: ptr = core.ptr_add(target, 1, memory_cap)
        let target_two: ptr = core.ptr_add(target, 2, memory_cap)
        let _write_two: UInt8 = core.store_u8(source_two, 20, memory_cap)
        let _write_three: UInt8 = core.store_u8(source_three, 22, memory_cap)
        let _copy: Int = copy_loop(target_one, source_two, 2)
        return core.load_u8(target_one, memory_cap) + core.load_u8(target_two, memory_cap)
    return 98
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0014FormatterPreservesGenericProtocolRequirementTypeParams(t *testing.T) {
	src := `
struct Vec2:
    x: Int

protocol Mapper:
    func map<T>(self: Vec2, value: T) -> T
`
	formatted, err := FormatSource([]byte(src), "generic_protocol_fmt.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if !strings.Contains(string(formatted), "func map<T>(self: Vec2, value: T) -> T") {
		t.Fatalf(
			"formatted source lost generic protocol requirement type params:\n%s",
			string(formatted),
		)
	}
}

func TestTetraBug0015ImportedGenericExtensionStaticCallMonomorphizes(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRunFiles(t, map[string]string{
		"engine/core.tetra": `module engine.core

struct Vec2:
    x: Int
`,
		"app/ext.tetra": `module app.ext

import engine.core as core

extension core.Vec2:
    func map<T>(self: core.Vec2, value: T) -> T:
        return value
`,
		"app/main.tetra": `module app.main

import app.ext as ext
import engine.core as core

func main() -> Int:
    let value: core.Vec2 = core.Vec2(x: 7)
    return core.Vec2.map(value, 42)
`,
	}, "app/main.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0016MatchCasePayloadBindingsAreSiblingScoped(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
enum Msg:
    case left(Int)
    case right(Int)

func main() -> Int:
    let msg: Msg = Msg.left(42)
    match msg:
    case Msg.left(value):
        return value
    case Msg.right(value):
        return value
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0020GenericIdentityAcceptsFunctionTypedLocal(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func add_one(value: Int) -> Int:
    return value + 1

func id<T>(value: T) -> T:
    return value

func apply(cb: fn(Int) -> Int, value: Int) -> Int:
    return cb(value)

func main() -> Int:
    let handler: fn(Int) -> Int = add_one
    let routed: fn(Int) -> Int = id(handler)
    return apply(routed, 41)
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0022GenericCallbackAcceptsFunctionSymbol(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func add_one(value: Int) -> Int:
    return value + 1

func apply_generic<T>(cb: fn(T) -> T, value: T) -> T:
    return cb(value)

func main() -> Int:
    return apply_generic(add_one, 41)
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0031EnumPayloadAcceptsGenericStructInstantiation(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
struct Box<T>:
    value: T

enum Route:
    case ready(Box<Int>)
    case empty

func main() -> Int:
    let box: Box<Int> = Box<Int>(value: 42)
    let route: Route = Route.ready(box)
    match route:
    case Route.ready(payload):
        return payload.value
    case Route.empty:
        return 0
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0033TypedTaskHandleCanUsePublicTaskAnnotation(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(42)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        7
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0033TypedTaskPublicHandleContainersAndGroup(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
enum TaskErr:
    case boom(Int)
    case stopped

struct TaskBox:
    handle: task.i32

func worker_42() -> Int throws TaskErr:
    throw TaskErr.boom(42)

func worker_7() -> Int throws TaskErr:
    throw TaskErr.boom(7)

func worker_5() -> Int throws TaskErr:
    throw TaskErr.boom(5)

func worker_3() -> Int throws TaskErr:
    throw TaskErr.boom(3)

func join_public(task: task.i32) -> Int
uses runtime:
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        0

func main() -> Int
uses runtime:
    let task_param: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker_42")
    let a: Int = join_public(task_param)

    let task_field: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker_7")
    let box: TaskBox = TaskBox(handle: task_field)
    let b: Int = catch core.task_join_i32_typed<TaskErr>(box.handle):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        0

    let task_optional: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker_5")
    let maybe: task.i32? = task_optional
    var c: Int = 0
    if let task = maybe:
        c = catch core.task_join_i32_typed<TaskErr>(task):
        case TaskErr.boom(code):
            code
        case TaskErr.stopped:
            0

    let group: task.group = core.task_group_open()
    let task_group: task.i32 = core.task_spawn_group_i32_typed<TaskErr>(group, "worker_3")
    let d: Int = catch core.task_join_group_i32_typed<TaskErr>(task_group):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        0
    let _closed: Int = core.task_group_close(group)

    return a + b + c + d
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 57 {
		t.Fatalf("exit code = %d, want 57", code)
	}
}

func TestTetraBug0035TypedActorRejectsMismatchedEnumMessageType(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
enum Request:
    case value(Int)
    case stop

enum Reply:
    case failed(Int)
    case value(Int)

func worker() -> Int
uses actors:
    let msg: Reply = core.recv_typed<Reply>()
    match msg:
    case Reply.failed(code):
        let _failed: Int = core.send_typed(core.sender(), Reply.failed(code))
        return 0
    case Reply.value(value):
        let _reply: Int = core.send_typed(core.sender(), Reply.value(value + 1))
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _sent: Int = core.send_typed(peer, Request.value(41))
    let reply: Reply = core.recv_typed<Reply>()
    match reply:
    case Reply.failed(code):
        return code
    case Reply.value(value):
        return value
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 255 {
		t.Fatalf("exit code = %d, want 255 mismatch sentinel", code)
	}
}

func TestTetraBug0037GlobalFixedArrayWriteRoundTrips(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
var seed: [3]Int

func main() -> Int:
    if seed.len != 3:
        return 10 + seed.len
    seed[0] = 42
    return seed[0]
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0037ZeroedGlobalFixedArrayReadsZero(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
var seed: [3]Int

func main() -> Int:
    if seed.len != 3:
        return 10 + seed.len
    return seed[0]
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestTetraBug0037GlobalStructFixedArrayFieldRoundTrips(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
struct ArrayBox:
    items: [3]Int

var box: ArrayBox

func main() -> Int:
    if box.items.len != 3:
        return 10 + box.items.len
    box.items[0] = 42
    return box.items[0]
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0038ScalarInoutWritesBackCallerLocal(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func bump(value: inout Int) -> Int:
    value = value + 1
    return value

func main() -> Int:
    var score: Int = 41
    let result: Int = bump(score)
    if result == 42 && score == 42:
        return 0
    return result + score
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestTetraBug0038FunctionTypedInoutWritesBackCallerLocal(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func bump(value: inout Int) -> Int:
    value = value + 1
    return value

func main() -> Int:
    let cb: fn(inout Int) -> Int = bump
    var score: Int = 41
    let result: Int = cb(score)
    if result == 42 && score == 42:
        return 0
    return result + score
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestTetraBug0038OptionalPointerInoutWritesBackCallerLocal(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func replace(slot: inout ptr?, value: ptr?) -> ptr?:
    slot = value
    return value

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let cell: ptr = core.ptr_add(payload, 4, memory_cap)
        let _stored: Int = core.store_i32(cell, 42, memory_cap)
        var maybe_base: ptr? = none
        let next_base: ptr? = payload
        let _returned: ptr? = replace(maybe_base, next_base)
        if let base = maybe_base:
            let loaded_cell: ptr = core.ptr_add(base, 4, memory_cap)
            let value: Int = core.load_i32(loaded_cell, memory_cap)
            if value == 42:
                return 0
            return value
        return 95
    return 99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestTetraBug0048FunctionTypedOptionalReturnLocalCall(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func some_int() -> Int?:
    return 42

func main() -> Int:
    let cb: fn() -> Int? = some_int
    let result: Int? = cb()
    if let value = result:
        return value
    return 0
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0049GenericInferenceUsesGenericStructFieldSelection(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
struct Box<T>:
    value: T

func id<T>(value: T) -> T:
    return value

func main() -> Int:
    let box: Box<Int> = Box<Int>(value: 42)
    let value: Int = id(box.value)
    if value == 42:
        return 0
    return value
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestTetraBug0040SelfHostTaskGroupDiagnostic(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	src := `
func worker() -> Int:
    return 42

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let value: Int = core.task_join_i32(task)
    let close_error: Int = core.task_group_close(group)
    if close_error == 0:
        return value
    return close_error
`
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "selfhost_task_group.tetra")
	outPath := filepath.Join(tmp, "app")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"linux-x64",
		BuildOptions{Runtime: RuntimeSelfHost},
	)
	if err == nil {
		t.Fatalf("expected selfhost task-group diagnostic")
	}
	if !strings.Contains(err.Error(), "self-host runtime does not support task groups") {
		t.Fatalf("diagnostic = %v", err)
	}
}

func TestTetraBug0041IfLetPayloadBindingCanBeReusedAfterScopeExit(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func maybe(flag: Bool, value: Int) -> Int?:
    if flag:
        return value
    return none

func main() -> Int:
    let a: Int? = maybe(true, 20)
    if let value = a:
        let left: Int = value
    let b: Int? = maybe(true, 22)
    if let value = b:
        return value + 20
    return 99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0043FormatterPreservesPublicModifiers(t *testing.T) {
	src := `
module probes.pub_format_probe

pub import lib.core.capability as cap

pub struct Box:
    value: Int

pub enum Route:
    case ready(Int)

pub protocol Score:
    func score(route: Route) -> Int

pub extension Box:
    func score(self: Box) -> Int:
        return self.value

pub const answer: Int = 42

pub func score(route: Route) -> Int:
    return answer
`
	formatted, err := FormatSource([]byte(src), "pub_format_probe.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	for _, want := range []string{
		"pub import lib.core.capability as cap",
		"pub struct Box:",
		"pub enum Route:",
		"pub protocol Score:",
		"pub extension Box:",
		"pub const answer: Int = 42",
		"pub func score(route: Route) -> Int:",
	} {
		if !strings.Contains(string(formatted), want) {
			t.Fatalf("formatted source missing %q:\n%s", want, string(formatted))
		}
	}
}

func TestTetraBug0044FormatterPreservesSelectiveImports(t *testing.T) {
	src := `
module app.main

import lib.math.{add, Vec}

func main() -> Int:
    return add(40, 2)
`
	formatted, err := FormatSource([]byte(src), "selective_import_probe.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if !strings.Contains(string(formatted), "import lib.math.{add, Vec}") {
		t.Fatalf("formatted source lost selective import:\n%s", string(formatted))
	}
}

func TestTetraBug0045OptionalTaskGroupPayloadSpawnReturnsWorkerValue(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func worker() -> Int:
    return 42

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    if let handle = maybe:
        let task: task.i32 = core.task_spawn_group_i32(handle, "worker")
        let result: task.result_i32 = core.task_join_result_i32(task)
        let close_error: Int = core.task_group_close(handle)
        if result.error != 0:
            return 10 + result.error
        if close_error != 0:
            return 20 + close_error
        return result.value
    return 99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0045IfLetTaskSpawnRegistersRuntimeAndEntry(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func worker() -> Int:
    return 42

func main() -> Int
uses runtime:
    let maybe: Int? = 7
    if let value = maybe:
        let task: task.i32 = core.task_spawn_i32("worker")
        let result: task.result_i32 = core.task_join_result_i32(task)
        if result.error != 0:
            return 10 + result.error
        return result.value
    return 99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0045IfLetTypedTaskGroupSpawnEmitsWrapper(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
enum GroupErr:
    case stopped
    case code(Int)

func worker() -> Int throws GroupErr:
    return 42

func alias_group(group: task.group) -> task.group:
    return group

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    if let handle = maybe:
        let returned: task.group = alias_group(handle)
        let task = core.task_spawn_group_i32_typed<GroupErr>(returned, "worker")
        let value: Int = catch core.task_join_group_i32_typed<GroupErr>(task):
        case GroupErr.stopped:
            70
        case GroupErr.code(error_code):
            error_code
        let close_error: Int = core.task_group_close(returned)
        if close_error != 0:
            return 80 + close_error
        return value
    return 99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0046GenericIdentityPreservesTaskGroupResource(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func worker() -> Int:
    return 42

func id<T>(value: T) -> T:
    return value

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let returned: task.group = id(group)
    let task: task.i32 = core.task_spawn_group_i32(returned, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let close_error: Int = core.task_group_close(returned)
    if result.error != 0:
        return 10 + result.error
    if close_error != 0:
        return 20 + close_error
    return result.value
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0047IslandParameterCanReturnAggregateConstructor(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
struct IslandBox:
    region: island

func wrap_region(region: island) -> IslandBox:
    return IslandBox(region: region)

func main() -> Int
uses alloc, islands, mem:
    unsafe:
        let region: island = core.island_new(128)
        let boxed: IslandBox = wrap_region(region)
        var bytes: []u8 = core.island_make_u8(boxed.region, 3)
        bytes[0] = 12
        bytes[1] = 13
        bytes[2] = 17
        let total: Int = bytes[0] + bytes[1] + bytes[2]
        free(boxed.region)
        return total
    return 99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0050MatchExpressionCollectsTypedTaskRuntimeSymbols(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
enum Choice:
    case left
    case right

enum TaskErr:
    case stopped
    case code(Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.code(20)

func main() -> Int
uses runtime:
    let choice: Choice = Choice.left
    return match choice:
    case Choice.left:
        catch core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<TaskErr>("worker")):
        case TaskErr.stopped:
            70
        case TaskErr.code(code):
            code
    case Choice.right:
        99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 20 {
		t.Fatalf("exit code = %d, want 20", code)
	}
}

func TestTetraBug0051FormatterPreservesNestedCatchInMatchArm(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	src := `
enum Choice:
    case left
    case right

enum TaskErr:
    case stopped
    case code(Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.code(20)

func main() -> Int
uses runtime:
    let choice: Choice = Choice.left
    return match choice:
    case Choice.left:
        catch core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<TaskErr>("worker")):
        case TaskErr.stopped:
            70
        case TaskErr.code(code):
            code
    case Choice.right:
        99
`
	formatted, err := FormatSource([]byte(src), "match_catch_format.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := ("    case Choice.left:\n        catch " +
		"core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<" +
		"TaskErr>(\"worker\")):\n        case TaskErr.stopped:\n         " +
		"   70\n        case TaskErr.code(code):\n            code\n    " +
		"case Choice.right:")
	if !strings.Contains(string(formatted), want) {
		t.Fatalf("formatted source corrupted nested catch:\n%s", string(formatted))
	}
	stdout, code := buildAndRun(t, string(formatted))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 20 {
		t.Fatalf("exit code = %d, want 20", code)
	}
}

func TestTetraBug0052FormatterPreservesNestedMatchInCatchArm(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	src := `
enum Kind:
    case value(Int)
    case empty

enum Err:
    case nested(Kind)

func fail() -> Int throws Err:
    throw Err.nested(Kind.value(4))

func main() -> Int:
    return catch fail():
    case Err.nested(kind):
        match kind:
        case Kind.value(value):
            value + 38
        case Kind.empty:
            0
`
	formatted, err := FormatSource([]byte(src), "nested_match_in_catch_format.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := ("    case Err.nested(kind):\n        match kind:\n        case " +
		"Kind.value(value):\n            value + 38\n        case " +
		"Kind.empty:\n            0")
	if !strings.Contains(string(formatted), want) {
		t.Fatalf("formatted source corrupted nested match:\n%s", string(formatted))
	}
	stdout, code := buildAndRun(t, string(formatted))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0053AwaitedOptionalResourceLocalBuilds(t *testing.T) {
	requireTetraBugInterfaceOnlyBuild(t, `
async func maybe_task(handle: task.i32) -> task.i32?:
    let out: task.i32? = handle
    return out

async func relay_task(handle: task.i32) -> task.i32?:
    let out: task.i32? = await maybe_task(handle)
    return out

func main() -> Int:
    return 42
`)
}

func TestTetraBug0054AwaitedResourceAggregateLocalBuilds(t *testing.T) {
	requireTetraBugInterfaceOnlyBuild(t, `
struct TaskBox:
    handle: task.i32

async func box_task(handle: task.i32) -> TaskBox:
    return TaskBox(handle: handle)

async func local_task_box(handle: task.i32) -> TaskBox:
    let box: TaskBox = await box_task(handle)
    return box

func main() -> Int:
    return 42
`)
}

func TestTetraBug0055DirectAwaitedPointerReturnBuilds(t *testing.T) {
	requireTetraBugInterfaceOnlyBuild(t, `
async func derive(base: ptr, offset: Int, memory_cap: cap.mem) -> ptr
uses mem:
    unsafe:
        return core.ptr_add(base, offset, memory_cap)
    return base

async func relay(base: ptr, memory_cap: cap.mem) -> ptr
uses mem:
    return await derive(base, 4, memory_cap)

func main() -> Int:
    return 42
`)
}

// ---- translation_validation_v2_test.go ----

func TestP23TranslationValidationV2CoversSupportedOptimizerSubset(t *testing.T) {
	report, err := BuildP23TranslationValidationV2()
	if err != nil {
		t.Fatalf("BuildP23TranslationValidationV2: %v", err)
	}
	if report.SchemaVersion != translationValidationV2Schema {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, translationValidationV2Schema)
	}
	if report.Scope != translationValidationV2ScopeP230 {
		t.Fatalf("scope = %q, want %q", report.Scope, translationValidationV2ScopeP230)
	}
	if err := ValidateP23TranslationValidationV2(report); err != nil {
		t.Fatalf("ValidateP23TranslationValidationV2: %v", err)
	}

	rows := map[TranslationValidationV2ID]TranslationValidationV2Row{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 ||
			len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p23TranslationValidationV2IDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p23AssertTranslationRow(
		t,
		rows[TranslationValidationV2RegisteredPasses],
		[]string{"RegisteredPasses", "translation_validation", "validation metadata"},
	)
	p23AssertTranslationRow(
		t,
		rows[TranslationValidationV2SymbolicScalar],
		[]string{"symbolic", "scalar arithmetic", "semantic local equivalence"},
	)
	p23AssertTranslationRow(
		t,
		rows[TranslationValidationV2MemoryEquivalence],
		[]string{"i32 slice", "memory", "backend matrix"},
	)
	p23AssertTranslationRow(
		t,
		rows[TranslationValidationV2BoundsProofPreservation],
		[]string{"proof facts", "missing proof id", "bounds proof"},
	)
	p23AssertTranslationRow(
		t,
		rows[TranslationValidationV2AllocationPlanPreservation],
		[]string{"ValidateAllocationLowering", "allocation plan"},
	)
	p23AssertTranslationRow(
		t,
		rows[TranslationValidationV2MachineCheckableHashes],
		[]string{"sha256", "before", "after"},
	)

	witnesses := map[string]TranslationValidationV2Witness{}
	for _, witness := range report.Witnesses {
		witnesses[witness.ID] = witness
	}
	registered := witnesses[p23TranslationRegisteredPassesWitnessID]
	if registered.RegisteredPasses < 6 || !registered.RegisteredPassCoverageComplete ||
		!registered.TranslationMetadataPresent {
		t.Fatalf(
			"registered pass witness = %#v, want all registered passes covered with metadata",
			registered,
		)
	}
	scalar := witnesses[p23TranslationScalarWitnessID]
	if scalar.SymbolicScalarChecks == 0 || scalar.DifferentialSamples == 0 ||
		!scalar.SemanticMismatchRejected {
		t.Fatalf(
			"scalar witness = %#v, want symbolic checks, samples, and mismatch rejection",
			scalar,
		)
	}
	memory := witnesses[p23TranslationMemoryWitnessID]
	if memory.MemoryEquivalenceSamples == 0 || memory.DifferentialLanes < 5 ||
		!memory.MemoryMismatchRejected {
		t.Fatalf(
			"memory witness = %#v, want memory samples, matrix lanes, and mismatch rejection",
			memory,
		)
	}
	loop := witnesses[p23TranslationLoopWitnessID]
	if loop.LoopEquivalenceSamples == 0 || loop.DifferentialLanes < 5 {
		t.Fatalf("loop witness = %#v, want loop equivalence samples and matrix lanes", loop)
	}
	call := witnesses[p23TranslationCallInliningWitnessID]
	if call.CallEquivalenceSamples == 0 || !call.BeforeHadCall || call.AfterHadCall ||
		!call.TranslationValidated {
		t.Fatalf("call/inlining witness = %#v, want call removed by validated inlining", call)
	}
	proof := witnesses[p23TranslationProofWitnessID]
	if proof.ProofFactsCompared == 0 || !proof.BoundsProofsPreserved ||
		!proof.MissingProofRejected {
		t.Fatalf("proof witness = %#v, want proof preservation and missing proof rejection", proof)
	}
	allocation := witnesses[p23TranslationAllocationWitnessID]
	if !allocation.AllocationPlanValidated || !allocation.AllocationDriftRejected {
		t.Fatalf(
			"allocation witness = %#v, want allocation plan validation and drift rejection",
			allocation,
		)
	}
	hash := witnesses[p23TranslationHashWitnessID]
	if !strings.HasPrefix(hash.BeforeHash, "sha256:") ||
		!strings.HasPrefix(hash.AfterHash, "sha256:") ||
		!hash.HashesMachineCheckable ||
		!hash.HashesDistinct {
		t.Fatalf("hash witness = %#v, want machine-checkable distinct sha256 hashes", hash)
	}

	for _, nonClaim := range []string{
		"no full formal proof is claimed",
		"no exhaustive optimizer completeness is claimed",
		"no broad memory model or alias model is claimed",
		"no broad loop theorem prover is claimed",
		"no performance claim is made",
		"runtime behavior does not change",
		"safe-program semantics do not change",
	} {
		if !p23TranslationHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP23TranslationValidationV2RejectsFakeClaimsAndDrift(t *testing.T) {
	base, err := BuildP23TranslationValidationV2()
	if err != nil {
		t.Fatalf("BuildP23TranslationValidationV2: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*TranslationValidationV2Report)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *TranslationValidationV2Report) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *TranslationValidationV2Report) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness",
			mutate: func(report *TranslationValidationV2Report) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "incomplete registered pass coverage",
			mutate: func(report *TranslationValidationV2Report) {
				report.RegisteredPassCoverageComplete = false
			},
			want: "registered pass",
		},
		{
			name: "missing scalar evidence",
			mutate: func(report *TranslationValidationV2Report) {
				report.SymbolicScalarEquivalenceSamples = 0
			},
			want: "symbolic scalar",
		},
		{
			name: "missing memory evidence",
			mutate: func(report *TranslationValidationV2Report) {
				report.MemoryEquivalenceSamples = 0
			},
			want: "memory equivalence",
		},
		{
			name: "missing proof evidence",
			mutate: func(report *TranslationValidationV2Report) {
				report.BoundsProofsPreserved = false
			},
			want: "bounds proof",
		},
		{
			name: "missing allocation evidence",
			mutate: func(report *TranslationValidationV2Report) {
				report.AllocationPlanValidated = false
			},
			want: "allocation plan",
		},
		{
			name: "missing hash evidence",
			mutate: func(report *TranslationValidationV2Report) {
				report.BeforeAfterHashesMachineCheckable = false
			},
			want: "hash",
		},
		{
			name: "full formal proof claim",
			mutate: func(report *TranslationValidationV2Report) {
				report.FullFormalProofClaimed = true
			},
			want: "full formal proof",
		},
		{
			name: "performance claim",
			mutate: func(report *TranslationValidationV2Report) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
		{
			name: "runtime behavior claim",
			mutate: func(report *TranslationValidationV2Report) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics claim",
			mutate: func(report *TranslationValidationV2Report) {
				report.SafeSemanticsChanged = true
			},
			want: "safe semantics",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := base
			report.Rows = append([]TranslationValidationV2Row(nil), base.Rows...)
			report.Witnesses = append([]TranslationValidationV2Witness(nil), base.Witnesses...)
			report.NonClaims = append([]string(nil), base.NonClaims...)
			tc.mutate(&report)
			err := ValidateP23TranslationValidationV2(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateP23TranslationValidationV2 error = %v, want %q", err, tc.want)
			}
		})
	}
}

func p23AssertTranslationRow(t *testing.T, row TranslationValidationV2Row, wants []string) {
	t.Helper()
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}

// ---- wasm_policy_test.go ----

func TestWASMPolicyRejectsUnsafeCapabilityMemoryMMIOAndCtxSwitchIR(t *testing.T) {
	cases := []struct {
		name    string
		instr   ir.IRInstr
		builtin string
	}{
		{
			name:    "alloc_bytes",
			instr:   ir.IRInstr{Kind: ir.IRAllocBytes},
			builtin: "core.alloc_bytes",
		},
		{name: "cap_io", instr: ir.IRInstr{Kind: ir.IRCapIO}, builtin: "core.cap_io"},
		{name: "cap_mem", instr: ir.IRInstr{Kind: ir.IRCapMem}, builtin: "core.cap_mem"},
		{name: "load_i32", instr: ir.IRInstr{Kind: ir.IRMemReadI32}, builtin: "core.load_i32"},
		{name: "store_i32", instr: ir.IRInstr{Kind: ir.IRMemWriteI32}, builtin: "core.store_i32"},
		{name: "load_u8", instr: ir.IRInstr{Kind: ir.IRMemReadU8}, builtin: "core.load_u8"},
		{name: "store_u8", instr: ir.IRInstr{Kind: ir.IRMemWriteU8}, builtin: "core.store_u8"},
		{name: "load_ptr", instr: ir.IRInstr{Kind: ir.IRMemReadPtr}, builtin: "core.load_ptr"},
		{name: "store_ptr", instr: ir.IRInstr{Kind: ir.IRMemWritePtr}, builtin: "core.store_ptr"},
		{
			name:    "store_arch_ptr",
			instr:   ir.IRInstr{Kind: ir.IRMemWriteArchPtr},
			builtin: "core.store_arch_ptr",
		},
		{
			name:    "load_i32_offset",
			instr:   ir.IRInstr{Kind: ir.IRMemReadI32Offset},
			builtin: "core.load_i32",
		},
		{
			name:    "store_i32_offset",
			instr:   ir.IRInstr{Kind: ir.IRMemWriteI32Offset},
			builtin: "core.store_i32",
		},
		{
			name:    "load_u8_offset",
			instr:   ir.IRInstr{Kind: ir.IRMemReadU8Offset},
			builtin: "core.load_u8",
		},
		{
			name:    "store_u8_offset",
			instr:   ir.IRInstr{Kind: ir.IRMemWriteU8Offset},
			builtin: "core.store_u8",
		},
		{
			name:    "load_ptr_offset",
			instr:   ir.IRInstr{Kind: ir.IRMemReadPtrOffset},
			builtin: "core.load_ptr",
		},
		{
			name:    "store_ptr_offset",
			instr:   ir.IRInstr{Kind: ir.IRMemWritePtrOffset},
			builtin: "core.store_ptr",
		},
		{
			name:    "store_arch_ptr_offset",
			instr:   ir.IRInstr{Kind: ir.IRMemWriteArchPtrOffset},
			builtin: "core.store_arch_ptr",
		},
		{name: "ptr_add", instr: ir.IRInstr{Kind: ir.IRPtrAdd}, builtin: "core.ptr_add"},
		{
			name:    "mmio_read_i32",
			instr:   ir.IRInstr{Kind: ir.IRMmioReadI32},
			builtin: "core.mmio_read_i32",
		},
		{
			name:    "mmio_write_i32",
			instr:   ir.IRInstr{Kind: ir.IRMmioWriteI32},
			builtin: "core.mmio_write_i32",
		},
		{name: "ctx_switch", instr: ir.IRInstr{Kind: ir.IRCtxSwitch}, builtin: "core.ctx_switch"},
	}

	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		for _, tc := range cases {
			t.Run(target+"/"+tc.name, func(t *testing.T) {
				err := validateWASMIRPolicy(target, []IRFunc{{
					Name:        "main",
					ReturnSlots: 1,
					Instrs: []ir.IRInstr{
						tc.instr,
						{Kind: ir.IRConstI32, Imm: 0},
						{Kind: ir.IRReturn},
					},
				}})
				if err == nil {
					t.Fatalf("expected WASM policy rejection for %s", tc.builtin)
				}
				for _, want := range []string{target, tc.builtin, "unsupported on WASM targets by policy"} {
					if !strings.Contains(err.Error(), want) {
						t.Fatalf("error = %v, want substring %q", err, want)
					}
				}
			})
		}
	}
}

func TestWASMBuildRejectsCapabilityBuiltinBeforeBackendEmission(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))
	src := `module app.main
func main() -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        return 0
`
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		t.Run(target, func(t *testing.T) {
			outPath := filepath.Join(tmp, target+".wasm")
			_, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1})
			if err == nil {
				t.Fatalf("expected %s capability policy rejection", target)
			}
			for _, want := range []string{target, "core.cap_mem", "unsupported on WASM targets by policy"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error = %v, want substring %q", err, want)
				}
			}
		})
	}
}

// ---- wasm_runtime_diagnostics_test.go ----

func TestWasmRuntimeBuiltinsUseTargetAwareDiagnostics(t *testing.T) {
	cases := []struct {
		name        string
		src         string
		wantMessage string
	}{
		{
			name: "actors",
			src: `func worker() -> Int
uses actors:
    return core.recv()

func main() -> Int
uses actors:
    let worker: actor = core.spawn("worker")
    return 0
`,
			wantMessage: "actors runtime not supported on %s",
		},
		{
			name: "distributed-actors",
			src: `func main() -> Int
uses actors, runtime:
    return core.actor_node_status(2)
`,
			wantMessage: "distributed actors runtime not supported on %s",
		},
		{
			name: "task",
			src: `func worker() -> Int:
    return 42

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`,
			wantMessage: "task runtime not supported on %s",
		},
		{
			name: "time",
			src: `func main() -> Int
uses runtime:
    return core.time_now_ms()
`,
			wantMessage: "time runtime not supported on %s",
		},
		{
			name: "filesystem-classifier",
			src: `func main() -> Int:
    return 0
`,
			wantMessage: "filesystem runtime not supported on %s",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
				target := target
				t.Run(target, func(t *testing.T) {
					dir := t.TempDir()
					srcPath := filepath.Join(dir, "main.tetra")
					outPath := filepath.Join(dir, "app.wasm")
					if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
						t.Fatalf("write source: %v", err)
					}

					var err error
					if tc.name == "filesystem-classifier" {
						err = rejectUnsupportedWASMRuntimeBuiltins([]IRFunc{{
							Name: "main",
							Instrs: []ir.IRInstr{{
								Kind:     ir.IRCall,
								Name:     "__tetra_fs_exists",
								ArgSlots: 3,
								RetSlots: 1,
							}},
						}}, target)
					} else {
						_, err = BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1})
					}
					if err == nil {
						t.Fatalf("expected wasm runtime diagnostic")
					}
					diag := DiagnosticFromError(err)
					if diag.Code != "TETRA3003" || diag.Severity != "error" {
						t.Fatalf("diagnostic identity = %#v", diag)
					}
					want := strings.ReplaceAll(tc.wantMessage, "%s", target)
					if diag.Message != want {
						t.Fatalf("message = %q, want %q", diag.Message, want)
					}
					if strings.Contains(err.Error(), "unsupported IR instruction") ||
						strings.Contains(err.Error(), "unsupported symbol") {
						t.Fatalf("diagnostic leaked generic backend text: %v", err)
					}
				})
			}
		})
	}
}
