package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
)

func TestBuildFailIfHeapRejectsHeapFallback(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "heap.tetra")
	outPath := filepath.Join(dir, "app")
	src := `func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2048)
    xs[0] = 7
    return xs[0]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:       1,
		FailIfHeap: true,
	})
	if err == nil || !strings.Contains(err.Error(), "RAM_CONTRACT_HEAP") {
		t.Fatalf("BuildFileWithStatsOpt error = %v, want RAM_CONTRACT_HEAP", err)
	}
}

func TestBuildEmitRAMContractReportWritesValidatedReports(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	outPath := filepath.Join(dir, "app")
	src := `func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 7
    xs[1] = 8
    return xs[0] + xs[1]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:                  1,
		EmitRAMContractReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}
	for _, suffix := range []string{".ram-contract.json", ".memory-grade.json", ".proof-store-summary.json", ".validation-pipeline-coverage.json"} {
		if _, err := os.Stat(outPath + suffix); err != nil {
			t.Fatalf("missing RAM report %s: %v", outPath+suffix, err)
		}
	}
}
