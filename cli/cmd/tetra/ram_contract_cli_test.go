package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCommandRAMContractFlagsWriteReports(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	outPath := filepath.Join(dir, "app")
	src := `func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 4
    xs[1] = 5
    return xs[0] + xs[1]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCLI([]string{"build", "--target", "linux-x64", "--emit-ram-contract-report", "-o", outPath, srcPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(outPath + ".ram-contract.json"); err != nil {
		t.Fatalf("missing RAM contract report: %v", err)
	}
}

func TestBuildCommandFailIfHeapJSONDiagnostic(t *testing.T) {
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
	diag := runCLIJSONDiagnostic(t, []string{"build", "--diagnostics=json", "--target", "linux-x64", "--fail-if-heap", "-o", outPath, srcPath}, 1)
	if diag.Code != "TETRA4100" || !strings.Contains(diag.Message, "RAM_CONTRACT_HEAP") {
		t.Fatalf("diagnostic = %#v", diag)
	}
}
