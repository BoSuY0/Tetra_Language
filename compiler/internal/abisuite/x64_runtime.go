package abisuite

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	ctarget "tetra_language/compiler/target"
)

type RuntimeSmokeDeps struct {
	BuildExecutable            func(srcPath string, outPath string, target string) error
	BuildExecutableWithOptions func(srcPath string, outPath string, target string, opts RuntimeBuildOptions) error
	RunExecutable              func(path string) RuntimeRunResult
	HostGOOS                   string
	HostGOARCH                 string
}

type RuntimeBuildOptions struct {
	IslandsDebug bool
}

type RuntimeRunResult struct {
	ExitCode int
	Output   string
	TimedOut bool
	Err      error
}

func CheckSourceNativeScalarDiagnostics(tgt ctarget.Target, deps FFICheckDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-source-native-scalar-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	var cases []struct {
		name string
		src  string
	}
	if tgt.Triple == "linux-x86" || tgt.Triple == "linux-x32" {
		cases = []struct {
			name string
			src  string
		}{
			{
				name: "u32_param",
				src:  "func native_probe(n: u32) -> Int:\n    return 0\n",
			},
			{
				name: "u64_param",
				src:  "func native_probe(n: u64) -> Int:\n    return 0\n",
			},
			{
				name: "f64_return",
				src:  "func native_probe() -> f64:\n    return 0\n",
			},
		}
	} else {
		cases = []struct {
			name string
			src  string
		}{
			{
				name: "usize_param",
				src:  "func native_probe(n: usize) -> Int:\n    return 0\n",
			},
			{
				name: "size_t_param",
				src:  "func native_probe(n: size_t) -> Int:\n    return 0\n",
			},
			{
				name: "native_int_return",
				src:  "func native_probe() -> native_int:\n    return 0\n",
			},
			{
				name: "c_long_return",
				src:  "func native_probe() -> c_long:\n    return 0\n",
			},
		}
	}
	for _, tc := range cases {
		srcPath := filepath.Join(tmpDir, tgt.Triple+"_"+tc.name+".tetra")
		outPath := filepath.Join(tmpDir, tgt.Triple+"_"+tc.name+".tobj")
		if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
			return err
		}
		err := buildLibrary(deps, srcPath, outPath, tgt.Triple)
		if err == nil {
			return fmt.Errorf("%s accepted source-level target-layout scalar in %s", tgt.Triple, tc.name)
		}
		for _, want := range []string{
			"target-layout scalar type",
			"not supported in source-level Tetra yet",
			"native-int/codegen support",
		} {
			if !strings.Contains(err.Error(), want) {
				return fmt.Errorf("%s source native scalar diagnostic for %s = %q, want %q", tgt.Triple, tc.name, err.Error(), want)
			}
		}
		if strings.Contains(err.Error(), "unknown type") {
			return fmt.Errorf("%s source native scalar diagnostic for %s fell back to unknown type: %q", tgt.Triple, tc.name, err.Error())
		}
		if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
			return fmt.Errorf("%s source native scalar wrote object %s (stat err=%v)", tgt.Triple, outPath, statErr)
		}
	}
	return nil
}

func CheckX64PlatformObjectABISmoke(tgt ctarget.Target, deps FFICheckDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-x64-platform-object-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	stem := strings.ReplaceAll(tgt.Triple, "-", "_")
	srcPath := filepath.Join(tmpDir, stem+"_abi_smoke.tetra")
	outPath := filepath.Join(tmpDir, stem+"_abi_smoke.tobj")
	src := "@export(\"ffi_say_i32\")\nfun say(): i32 uses io {\n  print(\"" + tgt.Triple + " abi\\n\")\n  return 0\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildLibrary(deps, srcPath, outPath, tgt.Triple); err != nil {
		return err
	}
	obj, err := readObject(deps, outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("target mismatch: got %q want %s", obj.Target, tgt.Triple)
	}
	if !strings.Contains(string(obj.Data), tgt.Triple+" abi\n") {
		return fmt.Errorf("%s object data missing ABI smoke literal: %q", tgt.Triple, string(obj.Data))
	}
	if !ObjectHasSymbolSignature(obj, "ffi_say_i32", 0, 1) {
		return fmt.Errorf("%s object missing scalar exported ffi_say_i32 symbol: %#v", tgt.Triple, obj.Symbols)
	}
	if !ObjectHasRelocKind(obj, ObjectRelocDataDisp32) {
		return fmt.Errorf("%s object missing data displacement relocation: %#v", tgt.Triple, obj.Relocs)
	}
	switch tgt.Triple {
	case "macos-x64":
		if ObjectHasRelocKind(obj, ObjectRelocIATDisp32) {
			return fmt.Errorf("macos-x64 object unexpectedly has Windows IAT reloc: %#v", obj.Relocs)
		}
	case "windows-x64":
		for _, name := range []string{"kernel32.GetStdHandle", "kernel32.WriteFile"} {
			if !ObjectHasReloc(obj, ObjectRelocIATDisp32, name) {
				return fmt.Errorf("windows-x64 object missing IAT relocation %q: %#v", name, obj.Relocs)
			}
		}
	default:
		return fmt.Errorf("x64 platform object ABI smoke does not cover %s", tgt.Triple)
	}
	return nil
}

func CheckX64PointerFFIRegressionSmoke(deps FFICheckDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-x64-pointer-ffi-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "x64_pointer_ffi.tetra")
	outPath := filepath.Join(tmpDir, "x64_pointer_ffi.tobj")
	src := `@export("ffi_ptr_param_c")
func ffi_ptr_param(p: ptr) -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildLibrary(deps, srcPath, outPath, "linux-x64"); err != nil {
		return err
	}
	obj, err := readObject(deps, outPath)
	if err != nil {
		return err
	}
	if obj.Target != "linux-x64" {
		return fmt.Errorf("x64 pointer FFI object target = %q, want linux-x64", obj.Target)
	}
	if !ObjectHasSymbolSignature(obj, "ffi_ptr_param_c", 1, 1) {
		return fmt.Errorf("x64 pointer FFI object missing exported ffi_ptr_param_c(1)->1 symbol: %#v", obj.Symbols)
	}
	return nil
}

func CheckX64FilesystemSchedulerCompositionSmoke(deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-x64-filesystem-scheduler-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "x64_filesystem_scheduler.tetra")
	outPath := filepath.Join(tmpDir, "x64-filesystem-scheduler")
	src := `
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
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, "linux-x64"); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	return checkX64ELFExecutable(data, "x64 filesystem scheduler")
}

func CheckX64NetworkingRuntimeSmoke(deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-x64-networking-runtime-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "x64_networking_runtime.tetra")
	outPath := filepath.Join(tmpDir, "x64-networking-runtime")
	src := `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 11
        let nb: Int = core.net_set_nonblocking(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if nb < 0:
            return 12
        if closed != 0:
            return 13
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, "linux-x64"); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkX64ELFExecutable(data, "x64 networking runtime"); err != nil {
		return err
	}
	if hostGOOS(deps) != "linux" || hostGOARCH(deps) != "amd64" {
		return nil
	}
	result := runExecutable(deps, outPath)
	if result.TimedOut {
		return fmt.Errorf("x64 networking runtime executable timed out: %q", result.Output)
	}
	if result.Err != nil {
		return fmt.Errorf("run x64 networking runtime: %w output=%q", result.Err, result.Output)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("x64 networking runtime exit=%d output=%q, want 0", result.ExitCode, result.Output)
	}
	if result.Output != "" {
		return fmt.Errorf("x64 networking runtime output=%q, want empty", result.Output)
	}
	return nil
}

func CheckX64SchedulerRestrictionRegressionSmoke(deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-x64-scheduler-regression-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "x64_scheduler_regression.tetra")
	outPath := filepath.Join(tmpDir, "x64-scheduler-regression")
	src := `
enum TaskErr:
    case boom(Int, Int)
    case stopped

func left() -> Int:
    return 7

func right() -> Int:
    return 8

func typed_worker() -> Int throws TaskErr:
    throw TaskErr.boom(10, 17)

func main() -> Int
uses runtime:
    let left_task: task.i32 = core.task_spawn_i32("left")
    let right_task: task.i32 = core.task_spawn_i32("right")
    let typed_task = core.task_spawn_i32_typed<TaskErr>("typed_worker")
    let left_value: Int = core.task_join_i32(left_task)
    let right_value: Int = core.task_join_i32(right_task)
    let typed_value: Int = catch core.task_join_i32_typed<TaskErr>(typed_task):
    case TaskErr.boom(first, second):
        first + second
    case TaskErr.stopped:
        99
    return left_value + right_value + typed_value
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, "linux-x64"); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	if err := checkX64ELFExecutable(data, "x64 scheduler regression"); err != nil {
		return err
	}
	if hostGOOS(deps) != "linux" || hostGOARCH(deps) != "amd64" {
		return nil
	}
	result := runExecutable(deps, outPath)
	if result.TimedOut {
		return fmt.Errorf("x64 scheduler regression executable timed out: %q", result.Output)
	}
	if result.Err != nil {
		return fmt.Errorf("run x64 scheduler regression: %w output=%q", result.Err, result.Output)
	}
	if result.ExitCode != 42 {
		return fmt.Errorf("x64 scheduler regression exit=%d output=%q, want 42", result.ExitCode, result.Output)
	}
	return nil
}

func buildExecutable(deps RuntimeSmokeDeps, srcPath string, outPath string, target string) error {
	return buildExecutableWithOptions(deps, srcPath, outPath, target, RuntimeBuildOptions{})
}

func buildExecutableWithOptions(deps RuntimeSmokeDeps, srcPath string, outPath string, target string, opts RuntimeBuildOptions) error {
	if deps.BuildExecutableWithOptions != nil {
		return deps.BuildExecutableWithOptions(srcPath, outPath, target, opts)
	}
	if opts.IslandsDebug {
		return fmt.Errorf("missing runtime smoke build executable-with-options callback")
	}
	if deps.BuildExecutable != nil {
		return deps.BuildExecutable(srcPath, outPath, target)
	}
	return fmt.Errorf("missing runtime smoke build executable callback")
}

func checkX64ELFExecutable(data []byte, label string) error {
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("%s output is not an ELF executable", label)
	}
	if data[4] != 2 {
		return fmt.Errorf("%s ELF class = %d, want ELFCLASS64", label, data[4])
	}
	if machine := uint16(data[18]) | uint16(data[19])<<8; machine != 0x3e {
		return fmt.Errorf("%s ELF machine = %#x, want EM_X86_64", label, machine)
	}
	return nil
}

func hostGOOS(deps RuntimeSmokeDeps) string {
	if deps.HostGOOS != "" {
		return deps.HostGOOS
	}
	return runtime.GOOS
}

func hostGOARCH(deps RuntimeSmokeDeps) string {
	if deps.HostGOARCH != "" {
		return deps.HostGOARCH
	}
	return runtime.GOARCH
}

func runExecutable(deps RuntimeSmokeDeps, path string) RuntimeRunResult {
	if deps.RunExecutable != nil {
		return deps.RunExecutable(path)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	output := out.String()
	if ctx.Err() == context.DeadlineExceeded {
		return RuntimeRunResult{Output: output, TimedOut: true}
	}
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			return RuntimeRunResult{Output: output, Err: err}
		}
		return RuntimeRunResult{ExitCode: exitErr.ProcessState.ExitCode(), Output: output}
	}
	return RuntimeRunResult{Output: output}
}
