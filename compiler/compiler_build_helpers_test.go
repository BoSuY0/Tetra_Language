package compiler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

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
