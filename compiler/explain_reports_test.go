package compiler_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/memoryfacts"
)

func TestBuildExplainReportsTruthProofAndAllocationArtifacts(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
repr(C) struct Header:
    tag: c_int
    code: c_int

struct Packet:
    tag: Int
    code: Int

func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    for x in xs:
        total = total + x
    return total

func get(xs: []i32, i: Int) -> Int
uses mem:
    return xs[i]

func scratch() -> Int
uses alloc, mem:
    var buf: []u8 = make_u8(4)
    buf[0] = 7
    return 0

func returned_heap() -> []u8
uses alloc, mem:
    var out: []u8 = make_u8(4)
    return out

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs) + get(xs, 0)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		Explain:          true,
		EmitPLIR:         true,
		EmitProof:        true,
		EmitBoundsReport: true,
		EmitAllocReport:  true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	for _, suffix := range []string{".plir.txt", ".plir.json", ".proof.json", ".bounds.json", ".alloc.json", ".alloc.txt", ".explain.txt", ".backend.json", ".layout.json", ".perf.json"} {
		if _, err := os.Stat(outPath + suffix); err != nil {
			t.Fatalf("missing report %s: %v", outPath+suffix, err)
		}
	}
	var bounds struct {
		Totals struct {
			Removed int `json:"removed"`
			Left    int `json:"left"`
		} `json:"totals"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	if bounds.Totals.Removed == 0 || bounds.Totals.Left == 0 {
		t.Fatalf("bounds totals = %+v, want removed and left checks", bounds.Totals)
	}
	var alloc struct {
		Totals struct {
			Heap  int `json:"heap"`
			Stack int `json:"stack"`
		} `json:"totals"`
	}
	raw, err = os.ReadFile(outPath + ".alloc.json")
	if err != nil {
		t.Fatalf("read alloc report: %v", err)
	}
	if err := json.Unmarshal(raw, &alloc); err != nil {
		t.Fatalf("parse alloc report: %v", err)
	}
	if alloc.Totals.Heap == 0 || alloc.Totals.Stack == 0 {
		t.Fatalf("alloc totals = %+v, want conservative heap and planned stack allocations", alloc.Totals)
	}
	if !strings.Contains(string(raw), "fixed_small_read_only_local_call_no_escape") {
		t.Fatalf("alloc report missing call-aware stack evidence:\n%s", raw)
	}
	raw, err = os.ReadFile(outPath + ".perf.json")
	if err != nil {
		t.Fatalf("read perf report: %v", err)
	}
	perfText := string(raw)
	for _, want := range []string{
		"p20.0_benchmark_matrix",
		"reports/benchmark-matrix-hardening-v1/benchmarks/p20-matrix-hardening-report.json",
		"left bounds check: missing dominance",
		"heap allocation: escapes through return",
		"not vectorized: no noalias proof",
		"not inlined: code-size budget",
		"register spill: live range pressure",
		"stack fallback: unsupported aggregate return",
		"actor copy: borrowed data crosses boundary",
		"compile_time_tetra",
	} {
		if !strings.Contains(perfText, want) {
			t.Fatalf("perf report missing %q:\n%s", want, perfText)
		}
	}

	raw, err = os.ReadFile(outPath + ".layout.json")
	if err != nil {
		t.Fatalf("read layout report: %v", err)
	}
	layoutText := string(raw)
	for _, want := range []string{
		`"schema_version": 2`,
		`"policy": "p21.0_default_layout_freedom_v1"`,
		`"decisions"`,
		`"compiler_owned_default"`,
		`"field_reordering"`,
		`"repr(C) locks layout"`,
		`"public ABI/exported FFI requires explicit repr(C)"`,
	} {
		if !strings.Contains(layoutText, want) {
			t.Fatalf("layout report missing %q:\n%s", want, layoutText)
		}
	}
}

func TestBuildMemoryReportMarksSliceViewDynamicBoundsChecks(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func main() -> Int
uses alloc, mem:
    var bytes: []u8 = make_u8(4)
    var words: []u16 = make_u16(4)
    var nums: []i32 = make_i32(4)
    let b: []u8 = bytes.window(1, 2)
    let w: []u16 = words.prefix(2)
    let n: []i32 = nums.suffix(1)
    let text: String = "abcdef".window(1, 3)
    return b.len + w.len + n.len + text.len
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitMemoryReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	raw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report: %v", err)
	}
	if err := memoryfacts.ValidateReportJSON(raw); err != nil {
		t.Fatalf("ValidateReportJSON: %v\n%s", err, raw)
	}
	var report memoryfacts.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse memory report: %v", err)
	}
	var retained int
	for _, row := range report.Rows {
		if row.Claim != "bounds_check_retained_dynamic" {
			continue
		}
		retained++
		if row.ParentFactID == "" ||
			row.CostClass != memoryfacts.CostDynamicCheckRequired ||
			!row.NormalBuildCheck ||
			row.ValidatorName != "safe_view_bounds_validator" ||
			row.ValidatorStatus != memoryfacts.ValidatorPass ||
			!strings.Contains(row.Reason, "elem_width:") ||
			!strings.Contains(row.Reason, "elem_shift:") {
			t.Fatalf("safe view retained-bounds row = %+v", row)
		}
	}
	if retained < 4 {
		t.Fatalf("bounds_check_retained_dynamic rows = %d, want at least 4:\n%+v", retained, report.Rows)
	}
}

func TestBuildMemoryReportValidatesExplicitIslandHelperLowering(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func make_buf(isl: island, n: Int) -> []u8
uses alloc, islands, mem:
    var buf: []u8 = core.island_make_u8(isl, n)
    return buf

func main() -> Int
uses alloc, islands, mem:
    var result: Int = 0
    island(64) as isl:
        var out: []u8 = make_buf(isl, 4)
        out[0] = 7
        result = out[0]
    return result
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitMemoryReport: true,
		EmitAllocReport:  true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	raw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report: %v", err)
	}
	if err := memoryfacts.ValidateReportJSON(raw); err != nil {
		t.Fatalf("ValidateReportJSON: %v\n%s", err, raw)
	}
	var report memoryfacts.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse memory report: %v", err)
	}
	for _, row := range report.Rows {
		if row.ActualLoweringStorage == memoryfacts.StorageExplicitIsland &&
			row.ClaimLevel == memoryfacts.ClaimValidated &&
			row.LoweredArtifactID != "" {
			return
		}
	}
	t.Fatalf("missing validated ExplicitIsland memory row with lowered artifact: %+v", report.Rows)
}

func TestBuildExplainReportsMachineScalarBackendPath(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func add(a: Int, b: Int) -> Int:
    return a + b

func main() -> Int:
    return add(40, 2)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:    1,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	raw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	if !strings.Contains(string(raw), "machine-ir-scalar") {
		t.Fatalf("backend report missing machine scalar path:\n%s", raw)
	}
	cmd := exec.Command(outPath)
	runOut, runErr := cmd.CombinedOutput()
	if string(runOut) != "" {
		t.Fatalf("runtime stdout mismatch: %q", runOut)
	}
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok {
		t.Fatalf("runtime exit = %v, want exit status 42", runErr)
	}
	if exitErr.ExitCode() != 42 {
		t.Fatalf("runtime exit code = %d, want 42", exitErr.ExitCode())
	}
}

func TestBuildExplainReportsMachineLoopBackendPath(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_n(n: Int) -> Int:
    var i = 0
    var total = 0
    while i < n:
        total = total + i
        i = i + 1
    return total

func main() -> Int:
    return sum_n(10)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:    1,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	raw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	for _, want := range []string{
		"machine-ir-loop",
		`"liveness"`,
		`"allocation"`,
		`"spills": {}`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("backend report missing %s:\n%s", want, raw)
		}
	}
	cmd := exec.Command(outPath)
	runOut, runErr := cmd.CombinedOutput()
	if string(runOut) != "" {
		t.Fatalf("runtime stdout mismatch: %q", runOut)
	}
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok {
		t.Fatalf("runtime exit = %v, want exit status 45", runErr)
	}
	if exitErr.ExitCode() != 45 {
		t.Fatalf("runtime exit code = %d, want 45", exitErr.ExitCode())
	}
}

func TestBuildExplainReportsMachineSliceSumBackendPath(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(3)
    xs[0] = 1
    xs[1] = 2
    xs[2] = 3
    return sum(xs)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		Explain:          true,
		EmitBoundsReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	backendRaw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	for _, want := range []string{
		"machine-ir-slice-sum",
		"index_load",
		"proof:while:",
		`"liveness"`,
		`"allocation"`,
		`"spills": {}`,
	} {
		if !strings.Contains(string(backendRaw), want) {
			t.Fatalf("backend report missing %s:\n%s", want, backendRaw)
		}
	}
	boundsRaw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if !strings.Contains(string(boundsRaw), `"reason": "removed_by_while_range"`) {
		t.Fatalf("bounds report missing removed while range reason:\n%s", boundsRaw)
	}
	cmd := exec.Command(outPath)
	runOut, runErr := cmd.CombinedOutput()
	if string(runOut) != "" {
		t.Fatalf("runtime stdout mismatch: %q", runOut)
	}
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok {
		t.Fatalf("runtime exit = %v, want exit status 6", runErr)
	}
	if exitErr.ExitCode() != 6 {
		t.Fatalf("runtime exit code = %d, want 6", exitErr.ExitCode())
	}
}

func TestBuildExplainReportsMachineCallBackendPath(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func inc(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return inc(inc(40))
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:    1,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	raw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	for _, want := range []string{
		"machine-ir-call",
		"call inc",
		"abi:sysv",
		"clobbers:rax,rcx,rdx,rsi,rdi,r8,r9,r10,r11",
		`"liveness"`,
		`"allocation"`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("backend report missing %s:\n%s", want, raw)
		}
	}
	cmd := exec.Command(outPath)
	runOut, runErr := cmd.CombinedOutput()
	if string(runOut) != "" {
		t.Fatalf("runtime stdout mismatch: %q", runOut)
	}
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok {
		t.Fatalf("runtime exit = %v, want exit status 42", runErr)
	}
	if exitErr.ExitCode() != 42 {
		t.Fatalf("runtime exit code = %d, want 42", exitErr.ExitCode())
	}
}

func TestBackendReportOnlyExplainShowsSelectionFallbackPaths(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	plainOutPath := filepath.Join(dir, "plain")
	explainOutPath := filepath.Join(dir, "explain")
	src := `
func add(a: Int, b: Int) -> Int:
    return a + b

func get(xs: []i32, i: Int) -> Int
uses mem:
    return xs[i]

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 40
    return add(get(xs, 0), 2)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, plainOutPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("plain BuildFileWithStatsOpt: %v", err)
	}
	if _, err := os.Stat(plainOutPath + ".backend.json"); !os.IsNotExist(err) {
		t.Fatalf("plain build backend report stat = %v, want no backend report without --explain", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, explainOutPath, "linux-x64", compiler.BuildOptions{
		Jobs:    1,
		Explain: true,
	}); err != nil {
		t.Fatalf("explain BuildFileWithStatsOpt: %v", err)
	}
	raw, err := os.ReadFile(explainOutPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	for _, want := range []string{
		`"backend_path": "register"`,
		`"backend_path": "stack"`,
		`"function": "add"`,
		`"function": "get"`,
		`"reason": "unsupported_or_unproven_subset_uses_stack_fallback"`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("backend selection report missing %s:\n%s", want, raw)
		}
	}
	cmd := exec.Command(explainOutPath)
	runOut, runErr := cmd.CombinedOutput()
	if string(runOut) != "" {
		t.Fatalf("runtime stdout mismatch: %q", runOut)
	}
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok {
		t.Fatalf("runtime exit = %v, want exit status 42", runErr)
	}
	if exitErr.ExitCode() != 42 {
		t.Fatalf("runtime exit code = %d, want 42", exitErr.ExitCode())
	}
}
