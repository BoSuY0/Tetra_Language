package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/target"
)

func TestActorsPingPongBuildAndRun(t *testing.T) {
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

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Runtime: RuntimeBuiltin}); err != nil {
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

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Runtime: RuntimeSelfHost}); err != nil {
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
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors_tagged_stress.tetra")
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
			if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Runtime: tc.rt}); err != nil {
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

func TestActorsPingPongRuntimeModeParity(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors_pingpong.tetra")
	results := map[RuntimeMode]struct {
		stdout string
		exit   int
	}{}
	for _, rt := range []RuntimeMode{RuntimeBuiltin, RuntimeSelfHost} {
		tmp := t.TempDir()
		outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
		if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{Runtime: rt}); err != nil {
			t.Fatalf("build runtime %d: %v", rt, err)
		}
		stdout, exitCode := runBinary(t, outPath)
		results[rt] = struct {
			stdout string
			exit   int
		}{stdout: stdout, exit: exitCode}
	}

	if results[RuntimeBuiltin] != results[RuntimeSelfHost] {
		t.Fatalf("runtime parity mismatch: builtin=%#v selfhost=%#v", results[RuntimeBuiltin], results[RuntimeSelfHost])
	}
}

func TestActorsPingPongBuildsSelfHostRuntimeForAllX64Targets(t *testing.T) {
	srcPath := filepath.Join("..", "examples", "actors_pingpong.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	tmp := t.TempDir()
	for _, triple := range []string{"linux-x64", "macos-x64", "windows-x64"} {
		t.Run(triple, func(t *testing.T) {
			tgt, err := target.Parse(triple)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			outPath := filepath.Join(tmp, "actors_"+strings.ReplaceAll(triple, "-", "_")+tgt.ExeExt)
			if _, err := BuildFileWithStatsOpt(srcPath, outPath, triple, BuildOptions{Runtime: RuntimeSelfHost}); err != nil {
				t.Fatalf("build: %v", err)
			}
			if _, err := os.Stat(outPath); err != nil {
				t.Fatalf("missing output: %v", err)
			}
		})
	}
}

func TestCanonicalSelfHostRuntimeSources(t *testing.T) {
	tests := []struct {
		path       string
		wantModule string
	}{
		{filepath.Join("..", "__rt", "actors_sysv.tetra"), "__rt.actors_sysv"},
		{filepath.Join("..", "__rt", "actors_win64.tetra"), "__rt.actors_win64"},
		{filepath.Join("selfhostrt", "actors_sysv.tetra"), "__rt.actors_sysv"},
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
		{"win64", filepath.Join("..", "__rt", "actors_win64.tetra"), "windows-x64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			objPath := filepath.Join(tmp, "runtime.tobj")
			if _, err := BuildFileWithStatsOpt(tt.src, objPath, tt.target, BuildOptions{Emit: EmitLibrary}); err != nil {
				t.Fatalf("build runtime object: %v", err)
			}
			obj, err := ReadObject(objPath)
			if err != nil {
				t.Fatalf("read runtime object: %v", err)
			}
			assertObjectHasSymbols(t, obj, requiredActorRuntimeSymbols()...)
		})
	}
}

func TestActorGlueExportsProgramRuntimeSymbols(t *testing.T) {
	dispatchFn, err := buildActorDispatchFunc([]string{"main", "pong"})
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
