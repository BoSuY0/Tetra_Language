package specs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"tetra_language/compiler"
	"tetra_language/tools/internal/zeroheapbench"
)

func TestBuildCoversP20MatrixAndRequiredCompilers(t *testing.T) {
	specs := Build("reports/local-benchmark-tier1-v1")
	wantRows := len(RequiredCategories) * len(RequiredLanguages)
	if len(specs) != wantRows {
		t.Fatalf("specs = %d, want %d", len(specs), wantRows)
	}
	seen := map[string]bool{}
	for _, spec := range specs {
		key := spec.Category + "\x00" + spec.Language
		if seen[key] {
			t.Fatalf("duplicate spec for %s/%s", spec.Category, spec.Language)
		}
		seen[key] = true
		if spec.AlgorithmID == "" || spec.InputDescription == "" || spec.Source == "" {
			t.Fatalf("spec %s missing equivalence/source metadata: %#v", spec.Name, spec)
		}
		switch spec.Language {
		case "tetra":
			if spec.BuildCommandKind != "tetra" {
				t.Fatalf("tetra spec %s build kind = %q", spec.Name, spec.BuildCommandKind)
			}
		case "c":
			if !containsSequence(spec.BuildArgs, "clang", "-O3") {
				t.Fatalf("c spec %s build args = %#v, want clang -O3", spec.Name, spec.BuildArgs)
			}
		case "cpp":
			if !containsSequence(spec.BuildArgs, "clang++", "-O3") {
				t.Fatalf(
					"cpp spec %s build args = %#v, want clang++ -O3",
					spec.Name,
					spec.BuildArgs,
				)
			}
		case "rust":
			if !containsSequence(spec.BuildArgs, "rustc", "-C", "opt-level=3") {
				t.Fatalf(
					"rust spec %s build args = %#v, want rustc -C opt-level=3",
					spec.Name,
					spec.BuildArgs,
				)
			}
		default:
			t.Fatalf("unexpected language %q", spec.Language)
		}
	}
	for _, category := range RequiredCategories {
		for _, language := range RequiredLanguages {
			if !seen[category+"\x00"+language] {
				t.Fatalf("missing spec for %s/%s", category, language)
			}
		}
	}
}

func TestZeroHeapMicrobenchSpecsStayOutsideTier1Matrix(t *testing.T) {
	specs := zeroheapbench.BuildSpecs("reports/local-zero-heap-benchmark-v1")
	if len(specs) == 0 {
		t.Fatalf("zero-heap microbenchmark specs are empty")
	}
	if len(specs) != len(zeroheapbench.Categories) {
		t.Fatalf(
			"zero-heap specs = %d, want one Tetra spec per category %d",
			len(specs),
			len(zeroheapbench.Categories),
		)
	}

	p20 := map[string]bool{}
	for _, category := range RequiredCategories {
		p20[category] = true
	}
	seen := map[string]bool{}
	for _, spec := range specs {
		if spec.Language != "tetra" {
			t.Fatalf("zero-heap spec %s language = %q, want tetra-only", spec.Name, spec.Language)
		}
		if p20[spec.Category] {
			t.Fatalf("zero-heap spec %q must stay outside Tier 1 P20 matrix", spec.Category)
		}
		if seen[spec.Category] {
			t.Fatalf("duplicate zero-heap category %q", spec.Category)
		}
		seen[spec.Category] = true
		if spec.AlgorithmID == "" || spec.InputDescription == "" || spec.Source == "" {
			t.Fatalf("zero-heap spec %s missing metadata/source: %#v", spec.Name, spec)
		}
		if !containsSequence(
			spec.BuildArgs,
			"tetra",
			"build",
			"--target",
			"linux-x64",
			"--explain",
		) {
			t.Fatalf(
				"zero-heap spec %s build metadata = %#v, want tetra explain build",
				spec.Name,
				spec.BuildArgs,
			)
		}
		if !strings.Contains(spec.SourceRelPath, "zero_heap") {
			t.Fatalf(
				"zero-heap spec %s source path = %q, want zero_heap module artifact path",
				spec.Name,
				spec.SourceRelPath,
			)
		}
	}
	for _, category := range zeroheapbench.Categories {
		if !seen[category] {
			t.Fatalf("missing zero-heap microbenchmark category %q", category)
		}
	}
}

func TestIntegerLoopsTetraSourceBuildsWithRegisterModuloLoopBackend(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "p25", "integer_loops.tetra")
	outPath := filepath.Join(dir, "integer_loops")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatalf("create integer_loops module dir: %v", err)
	}
	if err := os.WriteFile(srcPath, []byte(tetraSource("integer loops")), 0o644); err != nil {
		t.Fatalf("write integer_loops source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:    1,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(integer loops): %v", err)
	}

	raw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	var report struct {
		Functions []struct {
			Function    string `json:"function"`
			BackendPath string `json:"backend_path"`
			Category    string `json:"category"`
			Detail      string `json:"detail"`
			Reason      string `json:"reason"`
		} `json:"functions"`
		MachineFunctions []struct {
			Function             string   `json:"function"`
			Path                 string   `json:"path"`
			SSAPath              string   `json:"ssa_path"`
			SSAVerified          bool     `json:"ssa_verified"`
			InstructionSelection []string `json:"instruction_selection"`
			Validation           struct {
				MachineVerifier    string `json:"machine_verifier"`
				AllocationVerifier string `json:"allocation_verifier"`
				StackChurnOps      int    `json:"stack_churn_ops"`
			} `json:"validation"`
		} `json:"machine_functions"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse backend report: %v\n%s", err, raw)
	}
	var foundRow bool
	for _, row := range report.Functions {
		if row.Function != "p25.integer_loops.main" {
			continue
		}
		foundRow = true
		if row.BackendPath != "register" || row.Category != "register_path" ||
			row.Detail != "machine-ir-const-modulo-loop" || row.Reason != "eligible_machine_ir_subset" {
			t.Fatalf(
				"integer_loops backend row = %+v, want register const-modulo machine path",
				row,
			)
		}
	}
	if !foundRow {
		t.Fatalf("integer_loops backend row missing: %+v", report.Functions)
	}
	for _, row := range report.MachineFunctions {
		if row.Function != "p25.integer_loops.main" {
			continue
		}
		if row.Path != "machine-ir-const-modulo-loop" || !row.SSAVerified ||
			row.SSAPath != "value-ssa-v1" {
			t.Fatalf("integer_loops machine row = %+v, want verified const-modulo path", row)
		}
		if !containsString(row.InstructionSelection, "mod") {
			t.Fatalf(
				"integer_loops instruction selection = %+v, want mod evidence",
				row.InstructionSelection,
			)
		}
		if row.Validation.MachineVerifier != "pass" ||
			row.Validation.AllocationVerifier != "pass" ||
			row.Validation.StackChurnOps != 0 {
			t.Fatalf(
				"integer_loops validation = %+v, want verifier pass and zero stack churn",
				row.Validation,
			)
		}
		return
	}
	t.Fatalf("integer_loops machine report missing: %+v", report.MachineFunctions)
}

func TestRecursionTetraSourceBuildsWithRegisterRecursionBackend(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "p25", "recursion.tetra")
	outPath := filepath.Join(dir, "recursion")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatalf("create recursion module dir: %v", err)
	}
	if err := os.WriteFile(srcPath, []byte(tetraSource("recursion")), 0o644); err != nil {
		t.Fatalf("write recursion source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:    1,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt(recursion): %v", err)
	}

	raw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	var report struct {
		Functions []struct {
			Function    string `json:"function"`
			BackendPath string `json:"backend_path"`
			Category    string `json:"category"`
			Detail      string `json:"detail"`
			Reason      string `json:"reason"`
		} `json:"functions"`
		MachineFunctions []struct {
			Function             string   `json:"function"`
			Path                 string   `json:"path"`
			SSAPath              string   `json:"ssa_path"`
			SSAVerified          bool     `json:"ssa_verified"`
			InstructionSelection []string `json:"instruction_selection"`
			Validation           struct {
				MachineVerifier    string `json:"machine_verifier"`
				AllocationVerifier string `json:"allocation_verifier"`
				CallClobbers       string `json:"call_clobbers"`
				StackChurnOps      int    `json:"stack_churn_ops"`
			} `json:"validation"`
		} `json:"machine_functions"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse backend report: %v\n%s", err, raw)
	}
	wantFunctions := map[string]string{
		"p25.recursion.fib":  "machine-ir-recursive-fib",
		"p25.recursion.main": "machine-ir-recursion-main-loop",
	}
	seenRows := map[string]bool{}
	for _, row := range report.Functions {
		wantDetail, ok := wantFunctions[row.Function]
		if !ok {
			continue
		}
		seenRows[row.Function] = true
		if row.BackendPath != "register" || row.Category != "register_path" ||
			row.Detail != wantDetail || row.Reason != "eligible_machine_ir_subset" {
			t.Fatalf("recursion backend row = %+v, want register %s machine path", row, wantDetail)
		}
	}
	for name := range wantFunctions {
		if !seenRows[name] {
			t.Fatalf("recursion backend row missing for %s: %+v", name, report.Functions)
		}
	}
	seenMachine := map[string]bool{}
	for _, row := range report.MachineFunctions {
		wantPath, ok := wantFunctions[row.Function]
		if !ok {
			continue
		}
		seenMachine[row.Function] = true
		if row.Path != wantPath || !row.SSAVerified || row.SSAPath != "value-ssa-v1" {
			t.Fatalf("recursion machine row = %+v, want verified %s", row, wantPath)
		}
		if !containsString(row.InstructionSelection, "call") {
			t.Fatalf(
				"recursion instruction selection = %+v, want call evidence",
				row.InstructionSelection,
			)
		}
		if row.Validation.MachineVerifier != "pass" ||
			row.Validation.AllocationVerifier != "pass" ||
			row.Validation.CallClobbers != "validated" ||
			row.Validation.StackChurnOps != 0 {
			t.Fatalf(
				"recursion validation = %+v, want verifier/allocation/clobber pass and zero stack churn",
				row.Validation,
			)
		}
	}
	for name := range wantFunctions {
		if !seenMachine[name] {
			t.Fatalf("recursion machine report missing for %s: %+v", name, report.MachineFunctions)
		}
	}
}

func containsSequence(items []string, want ...string) bool {
	if len(want) == 0 || len(want) > len(items) {
		return false
	}
	for i := 0; i <= len(items)-len(want); i++ {
		ok := true
		for j := range want {
			if items[i+j] != want[j] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
