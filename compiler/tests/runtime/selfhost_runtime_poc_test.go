package compiler_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/target"
)

func TestSelfHostActorsRuntimePoC(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	rtName := "actors_poc_sysv.tetra"
	if tgt.Triple == "windows-x64" {
		rtName = "actors_poc_win64.tetra"
	}
	rtSrc := filepath.Join("..", "..", "..", "__rt", rtName)
	if _, err := os.Stat(rtSrc); err != nil {
		t.Fatalf("missing runtime source: %v", err)
	}

	progSrc := filepath.Join("..", "..", "..", "examples", "actors", "actors_pingpong.tetra")
	if _, err := os.Stat(progSrc); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	tmp := t.TempDir()
	rtObj := filepath.Join(tmp, "actors_rt_poc.tobj")
	if _, err := compiler.BuildFileWithStatsOpt(
		rtSrc,
		rtObj,
		tgt.Triple,
		compiler.BuildOptions{Emit: compiler.EmitLibrary},
	); err != nil {
		t.Fatalf("build runtime object: %v", err)
	}

	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	if _, err := compiler.BuildFileWithStatsOpt(
		progSrc,
		outPath,
		tgt.Triple,
		compiler.BuildOptions{RuntimeObjectPath: rtObj},
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
