package abisuite

import (
	"os"
	"path/filepath"
)

type runtimeBuildSmokeOptions struct {
	target      string
	stem        string
	label       string
	src         string
	wantClass   byte
	wantMachine uint16
}

func CheckX86TimeRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkRuntimeBuildSmoke(runtimeBuildSmokeOptions{
		target:      "linux-x86",
		stem:        "x86-time-runtime",
		label:       "x86 time runtime",
		src:         timeRuntimeSmokeSource(),
		wantClass:   1,
		wantMachine: 0x03,
	}, deps)
}

func CheckX86FilesystemRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkRuntimeBuildSmoke(runtimeBuildSmokeOptions{
		target:      "linux-x86",
		stem:        "x86-filesystem-runtime",
		label:       "x86 filesystem runtime",
		src:         filesystemRuntimeSmokeSource(),
		wantClass:   1,
		wantMachine: 0x03,
	}, deps)
}

func CheckX86FilesystemSchedulerCompositionSmoke(deps RuntimeSmokeDeps) error {
	return checkRuntimeBuildSmoke(runtimeBuildSmokeOptions{
		target:      "linux-x86",
		stem:        "x86-filesystem-scheduler",
		label:       "x86 filesystem scheduler",
		src:         filesystemSchedulerRuntimeSmokeSource(),
		wantClass:   1,
		wantMachine: 0x03,
	}, deps)
}

func CheckX32TimeRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkRuntimeBuildSmoke(runtimeBuildSmokeOptions{
		target:      "linux-x32",
		stem:        "x32-time-runtime",
		label:       "x32 time runtime",
		src:         timeRuntimeSmokeSource(),
		wantClass:   1,
		wantMachine: 0x3e,
	}, deps)
}

func CheckX32FilesystemRuntimeSmoke(deps RuntimeSmokeDeps) error {
	return checkRuntimeBuildSmoke(runtimeBuildSmokeOptions{
		target:      "linux-x32",
		stem:        "x32-filesystem-runtime",
		label:       "x32 filesystem runtime",
		src:         filesystemRuntimeSmokeSource(),
		wantClass:   1,
		wantMachine: 0x3e,
	}, deps)
}

func CheckX32FilesystemSchedulerCompositionSmoke(deps RuntimeSmokeDeps) error {
	return checkRuntimeBuildSmoke(runtimeBuildSmokeOptions{
		target:      "linux-x32",
		stem:        "x32-filesystem-scheduler-runtime",
		label:       "x32 filesystem scheduler runtime",
		src:         filesystemSchedulerRuntimeSmokeSource(),
		wantClass:   1,
		wantMachine: 0x3e,
	}, deps)
}

func checkRuntimeBuildSmoke(opts runtimeBuildSmokeOptions, deps RuntimeSmokeDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-"+opts.stem+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, opts.stem+".tetra")
	outPath := filepath.Join(tmpDir, opts.stem)
	if err := os.WriteFile(srcPath, []byte(opts.src), 0o644); err != nil {
		return err
	}
	if err := buildExecutable(deps, srcPath, outPath, opts.target); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}
	return checkRuntimeSmokeELF(data, opts.label, opts.wantClass, opts.wantMachine)
}

func timeRuntimeSmokeSource() string {
	return `
func main() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    let _until: Int = core.sleep_until(core.deadline_ms(2))
    return core.time_now_ms()
`
}

func filesystemRuntimeSmokeSource() string {
	return `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        if !core.fs_exists("../README.md", cap):
            return 11
        if core.fs_exists("__tetra_missing_fs_exists_smoke__", cap):
            return 12
    return 0
`
}

func filesystemSchedulerRuntimeSmokeSource() string {
	return `
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
`
}
