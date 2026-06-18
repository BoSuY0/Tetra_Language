package compiler_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/memoryfacts"
)

func TestBuildReportsShowBorrowedReturnNoAllocationAndCopyOwnership(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func view(xs: borrow []u8) -> borrow []u8:
    return xs.window(1, 2).borrow()

func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    xs[1] = 20
    xs[2] = 22
    let returned: []u8 = view(xs)
    let copied: []u8 = returned.copy()
    if copied.len != 2:
        return 1
    return copied[0] + copied[1]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		Explain:          true,
		EmitPLIR:         true,
		EmitProof:        true,
		EmitAllocReport:  true,
		EmitMemoryReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	proofRaw, err := os.ReadFile(outPath + ".proof.json")
	if err != nil {
		t.Fatalf("read proof report: %v", err)
	}
	proofText := string(proofRaw)
	for _, want := range []string{
		`"kind": "borrowed_imm"`,
		`"kind": "no_escape"`,
		`"kind": "derived_window"`,
		`"kind": "owned"`,
		`"kind": "provenance_known"`,
	} {
		if !strings.Contains(proofText, want) {
			t.Fatalf("proof report missing %s:\n%s", want, proofText)
		}
	}

	allocRaw, err := os.ReadFile(outPath + ".alloc.json")
	if err != nil {
		t.Fatalf("read alloc report: %v", err)
	}
	allocText := string(allocRaw)
	if strings.Contains(allocText, `"value_id": "alloc_intent:returned"`) {
		t.Fatalf("borrowed return was reported as allocation:\n%s", allocText)
	}
	if !strings.Contains(allocText, `"value_id": "alloc_intent:copied"`) {
		t.Fatalf("copy result allocation intent missing:\n%s", allocText)
	}

	memoryRaw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report: %v", err)
	}
	for _, want := range []string{
		`"claim": "borrowed_imm"`,
		`"claim": "no_escape"`,
		`"claim": "borrow_owner"`,
		`"claim": "borrow_source_fact_id"`,
		`"claim": "copy_owned"`,
		`"claim": "copy_source_fact_id"`,
	} {
		if !strings.Contains(string(memoryRaw), want) {
			t.Fatalf("memory report missing %s:\n%s", want, memoryRaw)
		}
	}
}

func TestBuildReportsShowCopyIntoNoFreshAllocation(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func main() -> Int
uses alloc, mem:
    var src: []u8 = make_u8(2)
    var dst: []u8 = make_u8(2)
    let n: Int = src.copy_into(dst)
    return n
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitPLIR:         true,
		EmitProof:        true,
		EmitAllocReport:  true,
		EmitMemoryReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	allocRaw, err := os.ReadFile(outPath + ".alloc.json")
	if err != nil {
		t.Fatalf("read alloc report: %v", err)
	}
	allocText := string(allocRaw)
	if strings.Contains(allocText, "copy_into") || strings.Contains(allocText, "alloc_intent:n") {
		t.Fatalf("copy_into should not appear as a fresh allocation:\n%s", allocText)
	}
	for _, want := range []string{`"value_id": "alloc_intent:src"`, `"value_id": "alloc_intent:dst"`} {
		if !strings.Contains(allocText, want) {
			t.Fatalf("alloc report missing %s:\n%s", want, allocText)
		}
	}

	proofRaw, err := os.ReadFile(outPath + ".proof.json")
	if err != nil {
		t.Fatalf("read proof report: %v", err)
	}
	if !strings.Contains(string(proofRaw), "copies into caller-owned destination without allocation") {
		t.Fatalf("proof/PLIR report missing copy_into no-allocation note:\n%s", proofRaw)
	}

	memoryRaw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report: %v", err)
	}
	if !strings.Contains(string(memoryRaw), `"claim": "copy_into_destination_fact_id"`) {
		t.Fatalf("memory report missing copy_into destination fact:\n%s", memoryRaw)
	}
}

func TestBuildReportsShowInoutNoAliasProofFact(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func mutate(xs: inout []u8) -> Int
uses mem:
    xs[0] = 1
    return xs[0]

func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(1)
    return mutate(xs)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitProof:        true,
		EmitMemoryReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	proofRaw, err := os.ReadFile(outPath + ".proof.json")
	if err != nil {
		t.Fatalf("read proof report: %v", err)
	}
	proofText := string(proofRaw)
	for _, want := range []string{`"kind": "no_alias"`, `"value_id": "param:xs"`, `"kind": "borrowed_mut"`} {
		if !strings.Contains(proofText, want) {
			t.Fatalf("proof report missing %s:\n%s", want, proofText)
		}
	}

	memoryRaw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report: %v", err)
	}
	memoryText := string(memoryRaw)
	for _, want := range []string{
		`"claim": "no_alias"`,
		`"claim": "mutable_exclusive"`,
		`"claim": "start_inout_exclusive"`,
		`"claim": "end_inout_exclusive"`,
		`"alias_state": "mutable_exclusive"`,
	} {
		if !strings.Contains(memoryText, want) {
			t.Fatalf("memory report missing %s:\n%s", want, memoryText)
		}
	}
}

func TestBuildReportsShowFunctionSummaryFacts(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
enum MoveMsg:
    case take(island)

var g_counter: i32

func worker() -> Int:
    return 7

func borrowed(xs: borrow []u8) -> borrow []u8
uses mem:
    return xs.window(0, 1).borrow()

func owned(xs: borrow []u8) -> []u8
uses alloc, mem:
    return xs.copy()

func mutate(dst: inout []u8) -> Int
uses mem:
    dst[0] = 1
    return dst[0]

func consume_peer(peer: consume actor) -> Int
uses actors:
    return 0

func main() -> Int
uses actors, alloc, capability, islands, mem, runtime:
    var xs: []u8 = make_u8(2)
    xs[0] = 3
    let view: []u8 = borrowed(xs)
    let copied: []u8 = owned(view)
    var dst: []u8 = make_u8(2)
    let _mutated: Int = mutate(dst)
    g_counter = copied[0]
    let task: task.i32 = core.task_spawn_i32("worker")
    let joined: Int = core.task_join_i32(task)
    let peer: actor = core.spawn("worker")
    let peer2: actor = core.spawn("worker")
    let _consumed: Int = consume_peer(peer2)
    unsafe:
        let _mem: cap.mem = core.cap_mem()
        var isl: island = core.island_new(16)
        let _sent_actor: Int = core.send_typed(peer, MoveMsg.take(isl))
        let _raw: ptr = core.alloc_bytes(8)
    return g_counter + joined
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitPLIR:         true,
		EmitMemoryReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	memoryRaw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report: %v", err)
	}
	memoryText := string(memoryRaw)
	for _, want := range []string{
		`"claim": "returns_borrow_from_param"`,
		`"claim": "may_return_region"`,
		`"claim": "returns_owned_new_allocation"`,
		`"claim": "may_store_global"`,
		`"claim": "may_escape_to_actor"`,
		`"claim": "may_escape_to_task"`,
		`"claim": "may_consume_param"`,
		`"claim": "may_mutate_inout"`,
		`"claim": "requires_effects"`,
		`"source_fact_id": "plir:`,
	} {
		if !strings.Contains(memoryText, want) {
			t.Fatalf("memory report missing %s:\n%s", want, memoryText)
		}
	}
}

func TestBuildReportsMemoryFactIDsAndSiteIDsStableAcrossRuns(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	src := `
func borrowed(xs: borrow []u8) -> borrow []u8
uses mem:
    return xs.window(0, 1).borrow()

func owned(xs: borrow []u8) -> []u8
uses alloc, mem:
    return xs.copy()

func main() -> Int
uses alloc, capability, mem:
    var xs: []u8 = make_u8(2)
    xs[0] = 3
    let view: []u8 = borrowed(xs)
    let copied: []u8 = owned(view)
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let _: UInt8 = core.store_u8(core.ptr_add(p, 1, mem), 7, mem)
    return copied[0]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	first := buildMemoryReportFactKeys(t, srcPath, filepath.Join(dir, "app-one"))
	second := buildMemoryReportFactKeys(t, srcPath, filepath.Join(dir, "app-two"))
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("memory report source_fact_id/site_id sequence changed across runs:\nfirst:  %+v\nsecond: %+v", first, second)
	}
}

func TestReportFlagsDoNotChangeBorrowedReturnFailure(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad.tetra")
	src := `
func bad(xs: borrow []u8) -> []u8:
    return xs.borrow()

func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(1)
    return bad(xs).len
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	options := []compiler.BuildOptions{
		{Jobs: 1},
		{Jobs: 1, EmitPLIR: true},
		{Jobs: 1, EmitProof: true},
		{Jobs: 1, EmitAllocReport: true},
		{Jobs: 1, EmitMemoryReport: true},
		{Jobs: 1, Explain: true},
	}
	for i, opt := range options {
		outPath := filepath.Join(dir, "bad-out-"+string(rune('a'+i)))
		_, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", opt)
		if err == nil || !strings.Contains(err.Error(), "borrowed slice return requires '-> borrow []u8' or '.copy()'") {
			t.Fatalf("option %d error = %v", i, err)
		}
	}
}

type memoryReportFactKey struct {
	SourceFactID string
	SiteID       string
}

func buildMemoryReportFactKeys(t *testing.T, srcPath string, outPath string) []memoryReportFactKey {
	t.Helper()
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitMemoryReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt %s: %v", outPath, err)
	}
	raw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report %s: %v", outPath, err)
	}
	var report memoryfacts.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse memory report %s: %v", outPath, err)
	}
	keys := make([]memoryReportFactKey, 0, len(report.Rows))
	for _, row := range report.Rows {
		if row.SourceFactID == "" || row.SiteID == "" {
			t.Fatalf("memory report row missing stable ids: %+v", row)
		}
		keys = append(keys, memoryReportFactKey{
			SourceFactID: string(row.SourceFactID),
			SiteID:       row.SiteID,
		})
	}
	return keys
}
