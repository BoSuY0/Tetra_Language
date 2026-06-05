package compiler

import (
	"bytes"
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

func TestLinuxX86FilesystemRuntimeObjectExportsFsExists(t *testing.T) {
	rt := buildLinuxX86FilesystemRuntimeObject()
	if rt.Target != "linux-x86" {
		t.Fatalf("runtime target = %q, want linux-x86", rt.Target)
	}
	if rt.Module != "__linux_x86_fsrt" {
		t.Fatalf("runtime module = %q, want __linux_x86_fsrt", rt.Module)
	}
	if len(rt.Symbols) != 1 || rt.Symbols[0].Name != "__tetra_fs_exists" || rt.Symbols[0].Offset != 0 {
		t.Fatalf("runtime symbols = %#v, want __tetra_fs_exists at offset 0", rt.Symbols)
	}
	if len(rt.Data) != 0 || len(rt.Relocs) != 0 {
		t.Fatalf("runtime object must be self-contained, data=%d relocs=%#v", len(rt.Data), rt.Relocs)
	}
	annotateRuntimeObjectSignatures(rt)
	if err := validateFilesystemRuntimeObject(rt); err != nil {
		t.Fatalf("validate x86 filesystem runtime object: %v", err)
	}
	for name, needle := range map[string][]byte{
		"stack buffer":        {0x81, 0xEC, 0x00, 0x10, 0x00, 0x00},
		"embedded nul guard":  {0x84, 0xD2, 0x0F, 0x84},
		"access syscall":      {0xB8, 0x21, 0x00, 0x00, 0x00},
		"int80 syscall":       {0xCD, 0x80},
		"callee-saved return": {0x81, 0xC4, 0x00, 0x10, 0x00, 0x00, 0x5F, 0x5E, 0x5B, 0x5D, 0xC3},
	} {
		if !bytes.Contains(rt.Code, needle) {
			t.Fatalf("runtime code missing %s sequence % x in % x", name, needle, rt.Code)
		}
	}
}

func TestLinuxX32FilesystemRuntimeObjectExportsFsExists(t *testing.T) {
	rt := buildLinuxX32FilesystemRuntimeObject()
	if rt.Target != "linux-x32" {
		t.Fatalf("runtime target = %q, want linux-x32", rt.Target)
	}
	if rt.Module != "__linux_x32_fsrt" {
		t.Fatalf("runtime module = %q, want __linux_x32_fsrt", rt.Module)
	}
	if len(rt.Symbols) != 1 || rt.Symbols[0].Name != "__tetra_fs_exists" || rt.Symbols[0].Offset != 0 {
		t.Fatalf("runtime symbols = %#v, want __tetra_fs_exists at offset 0", rt.Symbols)
	}
	if len(rt.Data) != 0 || len(rt.Relocs) != 0 {
		t.Fatalf("runtime object must be self-contained, data=%d relocs=%#v", len(rt.Data), rt.Relocs)
	}
	annotateRuntimeObjectSignatures(rt)
	if err := validateFilesystemRuntimeObject(rt); err != nil {
		t.Fatalf("validate x32 filesystem runtime object: %v", err)
	}
	for name, needle := range map[string][]byte{
		"stack buffer":        {0x48, 0x81, 0xEC, 0x00, 0x10, 0x00, 0x00},
		"embedded nul guard":  {0x84, 0xD2, 0x0F, 0x84},
		"x32 access syscall":  {0xB8, 0x15, 0x00, 0x00, 0x40},
		"syscall instruction": {0x0F, 0x05},
		"return":              {0xC9, 0xC3},
	} {
		if !bytes.Contains(rt.Code, needle) {
			t.Fatalf("runtime code missing %s sequence % x in % x", name, needle, rt.Code)
		}
	}
	if bytes.Contains(rt.Code, []byte{0xB8, 0x15, 0x00, 0x00, 0x00}) {
		t.Fatalf("x32 filesystem runtime emitted plain x64 access syscall: % x", rt.Code)
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

func TestLinuxX64FilesystemRuntimeComposesWithTaskScheduler(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	stdout, exitCode := buildAndRunWithOptions(t, `
func worker() -> Int:
    return 41

func main() -> Int
uses capability, io, runtime:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    let task: task.i32 = core.task_spawn_i32("worker")
    let value: Int = core.task_join_i32(task)
    if value != 41:
        return value
    return 0
`, BuildOptions{})
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want x64 filesystem+scheduler composition success", exitCode)
	}
}

func TestX86FilesystemRuntimeExistsBuildsAndRunsWhenHostSupportsI386(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "filesystem_x86.tetra")
	outPath := filepath.Join(tmp, "filesystem-x86")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 filesystem runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read x86 executable: %v", err)
	}
	if len(data) < 20 {
		t.Fatalf("x86 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		t.Fatalf("x86 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 1 {
		t.Fatalf("x86 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x03 {
		t.Fatalf("x86 executable machine = %#x, want EM_386", got)
	}
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 0 {
		t.Fatalf("x86 filesystem runtime exit=%d stdout=%q, want 0", code, stdout)
	}
}

func TestX32FilesystemRuntimeExistsBuildsAndRunsWhenHostSupportsX32(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "filesystem_x32.tetra")
	outPath := filepath.Join(tmp, "filesystem-x32")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x32 filesystem runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read x32 executable: %v", err)
	}
	if len(data) < 20 {
		t.Fatalf("x32 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		t.Fatalf("x32 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 1 {
		t.Fatalf("x32 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x3e {
		t.Fatalf("x32 executable machine = %#x, want EM_X86_64", got)
	}
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 0 {
		t.Fatalf("x32 filesystem runtime exit=%d stdout=%q, want 0", code, stdout)
	}
}

func TestX86FilesystemRuntimeComposesWithTaskScheduler(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "filesystem_mixed_x86.tetra")
	outPath := filepath.Join(tmp, "filesystem-mixed-x86")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses capability, io, runtime:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1})
	if err != nil {
		t.Fatalf("build mixed x86 filesystem+scheduler runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read mixed x86 executable: %v", err)
	}
	if len(data) < 20 {
		t.Fatalf("x86 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		t.Fatalf("x86 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 1 {
		t.Fatalf("x86 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x03 {
		t.Fatalf("x86 executable machine = %#x, want EM_386", got)
	}
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 41 {
		t.Fatalf("x86 filesystem+scheduler runtime exit=%d stdout=%q, want 41", code, stdout)
	}
}

func TestX32FilesystemRuntimeComposesWithTaskScheduler(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "filesystem_mixed_x32.tetra")
	outPath := filepath.Join(tmp, "filesystem-mixed-x32")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses capability, io, runtime:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1})
	if err != nil {
		t.Fatalf("build mixed x32 filesystem+scheduler runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read mixed x32 executable: %v", err)
	}
	if len(data) < 20 {
		t.Fatalf("x32 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		t.Fatalf("x32 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 1 {
		t.Fatalf("x32 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x3e {
		t.Fatalf("x32 executable machine = %#x, want EM_X86_64", got)
	}
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 41 {
		t.Fatalf("x32 filesystem+scheduler runtime exit=%d stdout=%q, want 41", code, stdout)
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

	for _, tc := range []struct {
		target string
		want   string
	}{
		{target: "macos-x64", want: "macos-x64"},
		{target: "windows-x64", want: "windows-x64"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			outPath := filepath.Join(tmp, "filesystem-"+tc.target)
			_, err := BuildFileWithStatsOpt(srcPath, outPath, tc.target, BuildOptions{Jobs: 1})
			if err == nil {
				t.Fatalf("expected unsupported filesystem runtime diagnostic")
			}
			want := "filesystem runtime not supported on " + tc.want
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
