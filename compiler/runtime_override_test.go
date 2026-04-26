package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/internal/actorsrt"
	"tetra_language/compiler/target"
)

func TestRuntimeObjectOverrideActorsPingPong(t *testing.T) {
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

	world, err := LoadWorld(srcPath)
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	actorsUsed, actorEntries, err := collectActorEntries(checked)
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
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath}); err != nil {
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

func TestRuntimeObjectOverrideRejectsMissingRequiredSymbols(t *testing.T) {
	tgt, ok := target.Host()
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
	_, err := BuildFileWithStatsOpt(filepath.Join("..", "examples", "actors_pingpong.tetra"), outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "runtime object missing required symbol") {
		t.Fatalf("unexpected error: %v", err)
	}
}
