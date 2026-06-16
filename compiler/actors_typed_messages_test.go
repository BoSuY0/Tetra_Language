package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/target"
)

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
