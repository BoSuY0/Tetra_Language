package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/target"
)

func TestFilesystemRuntimeRequiredSymbolsAndSignatures(t *testing.T) {
	got := requiredFilesystemRuntimeSymbols()
	if len(got) != 1 || got[0] != "__tetra_fs_exists" {
		t.Fatalf("filesystem runtime symbols = %#v, want __tetra_fs_exists", got)
	}
	sig, ok := runtimeObjectSignature("__tetra_fs_exists")
	if !ok {
		t.Fatalf("missing runtime signature for __tetra_fs_exists")
	}
	if sig.paramSlots != 3 || sig.returnSlots != 1 {
		t.Fatalf("__tetra_fs_exists signature = params %d returns %d, want params 3 returns 1", sig.paramSlots, sig.returnSlots)
	}
}

func TestCollectFilesystemRuntimeUsage(t *testing.T) {
	prog, err := Parse([]byte(`
func probe(cap: cap.io) -> Bool
uses io:
    return core.fs_exists("README.md", cap)

func main() -> Int:
    return 0
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !collectFilesystemRuntimeUsage(checked) {
		t.Fatalf("filesystem runtime usage was not collected")
	}
}

func TestValidateFilesystemRuntimeObjectChecksSignatureMetadata(t *testing.T) {
	obj := runtimeObjectWithFilesystemRuntimeSignatures()
	if err := validateFilesystemRuntimeObject(obj); err != nil {
		t.Fatalf("validate filesystem runtime object: %v", err)
	}

	replaceRuntimeSymbolSignature(obj, "__tetra_fs_exists", 2, 1)
	err := validateFilesystemRuntimeObject(obj)
	if err == nil {
		t.Fatalf("expected filesystem runtime signature mismatch")
	}
	if !strings.Contains(err.Error(), "runtime object symbol '__tetra_fs_exists' signature mismatch") ||
		!strings.Contains(err.Error(), "params=2 want=3") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingFilesystemSymbols(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_filesystem.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing_filesystem",
		Code:    []byte{0xC3},
		Symbols: runtimeObjectSymbols(requiredActorRuntimeSymbols()),
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	srcPath := filepath.Join(tmp, "filesystem_main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        if core.fs_exists("README.md", cap):
            return 0
    return 1
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "filesystem_main"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath})
	if err == nil {
		t.Fatalf("expected missing filesystem runtime symbol failure")
	}
	if !strings.Contains(err.Error(), "runtime object missing required symbol '__tetra_fs_exists'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFilesystemRuntimeExistsBuildAndRunLinuxX64(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    return 0
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want filesystem exists smoke success", exitCode)
	}
}

func TestFilesystemRuntimeRejectsUnsupportedNativeTargets(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "filesystem_main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        if core.fs_exists("README.md", cap):
            return 0
    return 1
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	for _, target := range []string{"macos-x64", "windows-x64"} {
		t.Run(target, func(t *testing.T) {
			outPath := filepath.Join(tmp, "filesystem-"+target)
			_, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1})
			if err == nil {
				t.Fatalf("expected unsupported filesystem runtime diagnostic")
			}
			want := "filesystem runtime not supported on " + target
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want %q", err, want)
			}
		})
	}
}

func runtimeObjectWithFilesystemRuntimeSignatures() *Object {
	obj := &Object{}
	for _, name := range requiredFilesystemRuntimeSymbols() {
		sig, ok := runtimeObjectSignature(name)
		if !ok {
			panic("missing filesystem runtime signature for " + name)
		}
		obj.Symbols = append(obj.Symbols, Symbol{
			Name:         name,
			HasSignature: true,
			ParamSlots:   sig.paramSlots,
			ReturnSlots:  sig.returnSlots,
		})
	}
	return obj
}

func runtimeObjectSymbols(names []string) []Symbol {
	symbols := make([]Symbol, 0, len(names))
	for _, name := range names {
		symbols = append(symbols, Symbol{Name: name, Offset: 0})
	}
	return symbols
}
