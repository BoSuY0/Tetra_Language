package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestX32ExecutableBuildsAutoSelfHostTimeRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "time_x32.tetra")
	outPath := filepath.Join(tmp, "time-x32")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses runtime:
    return core.time_now_ms()
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1})
	if err != nil {
		t.Fatalf("build x32 auto self-host time runtime: %v", err)
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
}

func TestX86TimeRuntimeBuildsAndRunsWhenHostSupportsI386(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "time_x86.tetra")
	outPath := filepath.Join(tmp, "time-x86")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    let _until: Int = core.sleep_until(core.deadline_ms(2))
    return core.time_now_ms()
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1})
	if err != nil {
		t.Fatalf("build x86 time runtime: %v", err)
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
	if code != 7 {
		t.Fatalf("x86 time runtime exit=%d stdout=%q, want 7", code, stdout)
	}
}

func TestX86SingleTaskRuntimeBuildsAndRunsWhenHostSupportsI386(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "task_x86.tetra")
	outPath := filepath.Join(tmp, "task-x86")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 single-task auto self-host runtime: %v", err)
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
	if code != 41 {
		t.Fatalf("x86 single-task runtime exit=%d stdout=%q, want 41", code, stdout)
	}
}

func TestX86TypedTaskRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(23)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "typed_task_x86.tetra")
			outPath := filepath.Join(tmp, "typed-task-x86")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1, Runtime: tc.runtime}); err != nil {
				t.Fatalf("build x86 typed-task %s self-host runtime: %v", tc.name, err)
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
			if code != 23 {
				t.Fatalf("x86 typed-task runtime exit=%d stdout=%q, want 23", code, stdout)
			}
		})
	}
}

func TestX86StagedTypedTaskRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
enum TaskErr:
    case boom(Int, Int, Int, Int, Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(1, 2, 3, 4, 5)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b, c, d, e):
        a + b + c + d + e
    case TaskErr.stopped:
        99
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "staged_typed_task_x86.tetra")
			outPath := filepath.Join(tmp, "staged-typed-task-x86")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1, Runtime: tc.runtime}); err != nil {
				t.Fatalf("build x86 staged typed-task %s self-host runtime: %v", tc.name, err)
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
			if code != 15 {
				t.Fatalf("x86 staged typed-task runtime exit=%d stdout=%q, want 15", code, stdout)
			}
		})
	}
}

func TestX86SingleTaskGroupRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    let status: Int = core.task_group_status(group)
    if status != 1:
        return 60 + status
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let before: Int = core.task_group_status(group)
    if before != 1:
        return 30 + before
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 80 + closeError
    let after: Int = core.task_group_status(group)
    if after != 3:
        return 90 + after
    if result.error != 0:
        return 100 + result.error
    return result.value
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "task_group_x86.tetra")
			outPath := filepath.Join(tmp, "task-group-x86")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1, Runtime: tc.runtime}); err != nil {
				t.Fatalf("build x86 task-group %s self-host runtime: %v", tc.name, err)
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
			if code != 7 {
				t.Fatalf("x86 task-group runtime exit=%d stdout=%q, want 7", code, stdout)
			}
		})
	}
}

func TestX86MultiSpawnTaskRuntimeBuildsSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "task_multi_spawn_x86.tetra")
	outPath := filepath.Join(tmp, "task-multi-spawn-x86")
	if err := os.WriteFile(srcPath, []byte(`
func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return core.time_now_ms()

func fast() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let slow_task: task.i32 = core.task_spawn_i32("slow")
    let fast_task: task.i32 = core.task_spawn_i32("fast")
    let fast_result: task.result_i32 = core.task_join_result_i32(fast_task)
    if fast_result.error != 0:
        return 20 + fast_result.error
    if fast_result.value != 2:
        return 40 + fast_result.value
    let slow_value: Int = core.task_join_i32(slow_task)
    return fast_result.value * 10 + slow_value
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 two-spawn task self-host runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read x86 executable: %v", err)
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		t.Fatalf("x86 executable missing ELF magic or too small: len=%d", len(data))
	}
	if data[4] != 1 {
		t.Fatalf("x86 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x03 {
		t.Fatalf("x86 executable machine = %#x, want EM_386", got)
	}
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 25 {
		t.Fatalf("x86 two-spawn task runtime exit=%d stdout=%q, want 25", code, stdout)
	}
}

func TestX86MultiSpawnTaskGroupRuntimeBuildsSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "task_group_multi_spawn_x86.tetra")
	outPath := filepath.Join(tmp, "task-group-multi-spawn-x86")
	if err := os.WriteFile(srcPath, []byte(`
func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return core.time_now_ms()

func fast() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    if core.task_group_status(group) != 1:
        return 70 + core.task_group_status(group)
    let _sleep: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let slow_task: task.i32 = core.task_spawn_group_i32(group, "slow")
    let fast_task: task.i32 = core.task_spawn_group_i32(group, "fast")
    let fast_result: task.result_i32 = core.task_join_result_i32(fast_task)
    if fast_result.error != 0:
        return 20 + fast_result.error
    if fast_result.value != 2:
        return 40 + fast_result.value
    let slow_value: Int = core.task_join_i32(slow_task)
    let close_error: Int = core.task_group_close(group)
    if close_error != 0:
        return 90 + close_error
    if core.task_group_status(group) != 3:
        return 100 + core.task_group_status(group)
    return fast_result.value * 10 + slow_value
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 two-spawn task-group self-host runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read x86 executable: %v", err)
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		t.Fatalf("x86 executable missing ELF magic or too small: len=%d", len(data))
	}
	if data[4] != 1 {
		t.Fatalf("x86 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x03 {
		t.Fatalf("x86 executable machine = %#x, want EM_386", got)
	}
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 25 {
		t.Fatalf("x86 two-spawn task-group runtime exit=%d stdout=%q, want 25", code, stdout)
	}
}

func TestX86TypedTaskGroupRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr uses runtime:
    let group: task.group = core.task_group_current()
    let status: Int = core.task_group_status(group)
    if status != 1:
        throw TaskErr.boom(60 + status)
    throw TaskErr.boom(23)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let before: Int = core.task_group_status(group)
    if before != 1:
        return 30 + before
    let task = core.task_spawn_group_i32_typed<TaskErr>(group, "worker")
    let result: Int = catch core.task_join_group_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 80 + closeError
    let after: Int = core.task_group_status(group)
    if after != 3:
        return 90 + after
    return result
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "typed_task_group_x86.tetra")
			outPath := filepath.Join(tmp, "typed-task-group-x86")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1, Runtime: tc.runtime}); err != nil {
				t.Fatalf("build x86 typed task-group %s self-host runtime: %v", tc.name, err)
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
			if code != 23 {
				t.Fatalf("x86 typed task-group runtime exit=%d stdout=%q, want 23", code, stdout)
			}
		})
	}
}

func TestX32MultiSpawnTaskRuntimeBuildsSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "task_multi_spawn_x32.tetra")
	outPath := filepath.Join(tmp, "task-multi-spawn-x32")
	if err := os.WriteFile(srcPath, []byte(`
func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return core.time_now_ms()

func fast() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let slow_task: task.i32 = core.task_spawn_i32("slow")
    let fast_task: task.i32 = core.task_spawn_i32("fast")
    let fast_result: task.result_i32 = core.task_join_result_i32(fast_task)
    if fast_result.error != 0:
        return 20 + fast_result.error
    if fast_result.value != 2:
        return 40 + fast_result.value
    let slow_value: Int = core.task_join_i32(slow_task)
    return fast_result.value * 10 + slow_value
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x32 two-spawn task self-host runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read x32 executable: %v", err)
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		t.Fatalf("x32 executable missing ELF magic or too small: len=%d", len(data))
	}
	if data[4] != 1 {
		t.Fatalf("x32 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x3e {
		t.Fatalf("x32 executable machine = %#x, want EM_X86_64", got)
	}
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 25 {
		t.Fatalf("x32 two-spawn task runtime exit=%d stdout=%q, want 25", code, stdout)
	}
}

func TestX32MultiSpawnTypedTaskRuntimeBuildsSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "typed_task_multi_spawn_x32.tetra")
	outPath := filepath.Join(tmp, "typed-task-multi-spawn-x32")
	if err := os.WriteFile(srcPath, []byte(`
enum TaskErr:
    case boom(Int)
    case stopped

func slow() -> Int throws TaskErr uses runtime:
    let _sleep: Int = core.sleep_ms(4)
    throw TaskErr.boom(11)

func fast() -> Int throws TaskErr uses runtime:
    let _sleep: Int = core.sleep_ms(1)
    throw TaskErr.boom(7)

func main() -> Int
uses runtime:
    let slow_task = core.task_spawn_i32_typed<TaskErr>("slow")
    let fast_task = core.task_spawn_i32_typed<TaskErr>("fast")
    let fast_value: Int = catch core.task_join_i32_typed<TaskErr>(fast_task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        90
    let slow_value: Int = catch core.task_join_i32_typed<TaskErr>(slow_task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        91
    return fast_value * 10 + slow_value
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x32 two-spawn typed-task self-host runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read x32 executable: %v", err)
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		t.Fatalf("x32 executable missing ELF magic or too small: len=%d", len(data))
	}
	if data[4] != 1 {
		t.Fatalf("x32 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x3e {
		t.Fatalf("x32 executable machine = %#x, want EM_X86_64", got)
	}
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 81 {
		t.Fatalf("x32 two-spawn typed-task runtime exit=%d stdout=%q, want 81", code, stdout)
	}
}

func TestX32MultiSpawnTaskGroupRuntimeBuildsSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "task_group_multi_spawn_x32.tetra")
	outPath := filepath.Join(tmp, "task-group-multi-spawn-x32")
	if err := os.WriteFile(srcPath, []byte(`
func slow() -> Int
uses runtime:
    let _sleep: Int = core.sleep_ms(5)
    return core.time_now_ms()

func fast() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    if core.task_group_status(group) != 1:
        return 70 + core.task_group_status(group)
    let _sleep: Int = core.sleep_ms(2)
    return core.time_now_ms()

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let slow_task: task.i32 = core.task_spawn_group_i32(group, "slow")
    let fast_task: task.i32 = core.task_spawn_group_i32(group, "fast")
    let fast_result: task.result_i32 = core.task_join_result_i32(fast_task)
    if fast_result.error != 0:
        return 20 + fast_result.error
    if fast_result.value != 2:
        return 40 + fast_result.value
    let slow_value: Int = core.task_join_i32(slow_task)
    let close_error: Int = core.task_group_close(group)
    if close_error != 0:
        return 90 + close_error
    if core.task_group_status(group) != 3:
        return 100 + core.task_group_status(group)
    return fast_result.value * 10 + slow_value
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x32 two-spawn task-group self-host runtime: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read x32 executable: %v", err)
	}
	if len(data) < 20 || string(data[:4]) != "\x7fELF" {
		t.Fatalf("x32 executable missing ELF magic or too small: len=%d", len(data))
	}
	if data[4] != 1 {
		t.Fatalf("x32 executable must use ELFCLASS32, got %d", data[4])
	}
	if got := uint16(data[18]) | uint16(data[19])<<8; got != 0x3e {
		t.Fatalf("x32 executable machine = %#x, want EM_X86_64", got)
	}
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if code != 25 {
		t.Fatalf("x32 two-spawn task-group runtime exit=%d stdout=%q, want 25", code, stdout)
	}
}

func TestX32TypedTaskGroupRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr uses runtime:
    let group: task.group = core.task_group_current()
    let status: Int = core.task_group_status(group)
    if status != 1:
        throw TaskErr.boom(60 + status)
    throw TaskErr.boom(23)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let before: Int = core.task_group_status(group)
    if before != 1:
        return 30 + before
    let task = core.task_spawn_group_i32_typed<TaskErr>(group, "worker")
    let result: Int = catch core.task_join_group_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 80 + closeError
    let after: Int = core.task_group_status(group)
    if after != 3:
        return 90 + after
    return result
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "typed_task_group_x32.tetra")
			outPath := filepath.Join(tmp, "typed-task-group-x32")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1, Runtime: tc.runtime}); err != nil {
				t.Fatalf("build x32 typed task-group %s self-host runtime: %v", tc.name, err)
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
			if code != 23 {
				t.Fatalf("x32 typed task-group runtime exit=%d stdout=%q, want 23", code, stdout)
			}
		})
	}
}

func TestX32SingleTaskRuntimeBuildsAutoSelfHostRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "task_single_x32.tetra")
	outPath := filepath.Join(tmp, "task-single-x32")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 41

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1})
	if err != nil {
		t.Fatalf("build x32 single-task auto self-host runtime: %v", err)
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
}

func TestX32TypedTaskRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(23)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        9
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "typed_task_x32.tetra")
			outPath := filepath.Join(tmp, "typed-task-x32")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1, Runtime: tc.runtime}); err != nil {
				t.Fatalf("build x32 typed-task %s self-host runtime: %v", tc.name, err)
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
			if code != 23 {
				t.Fatalf("x32 typed-task runtime exit=%d stdout=%q, want 23", code, stdout)
			}
		})
	}
}

func TestX32StagedTypedTaskRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
enum TaskErr:
    case boom(Int, Int, Int, Int, Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(1, 2, 3, 4, 5)

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(a, b, c, d, e):
        a + b + c + d + e
    case TaskErr.stopped:
        99
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "staged_typed_task_x32.tetra")
			outPath := filepath.Join(tmp, "staged-typed-task-x32")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1, Runtime: tc.runtime}); err != nil {
				t.Fatalf("build x32 staged typed-task %s self-host runtime: %v", tc.name, err)
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
			if code != 15 {
				t.Fatalf("x32 staged typed-task runtime exit=%d stdout=%q, want 15", code, stdout)
			}
		})
	}
}

func TestX32SingleTaskGroupRuntimeBuildsSelfHostRuntime(t *testing.T) {
	src := `
func worker() -> Int
uses runtime:
    let group: task.group = core.task_group_current()
    let status: Int = core.task_group_status(group)
    if status != 1:
        return 60 + status
    return 7

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let before: Int = core.task_group_status(group)
    if before != 1:
        return 30 + before
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let closeError: Int = core.task_group_close(group)
    if closeError != 0:
        return 80 + closeError
    let after: Int = core.task_group_status(group)
    if after != 3:
        return 90 + after
    if result.error != 0:
        return 100 + result.error
    return result.value
`
	for _, tc := range []struct {
		name    string
		runtime RuntimeMode
	}{
		{name: "auto", runtime: RuntimeAuto},
		{name: "explicit_selfhost", runtime: RuntimeSelfHost},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			srcPath := filepath.Join(tmp, "task_group_x32.tetra")
			outPath := filepath.Join(tmp, "task-group-x32")
			if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
				t.Fatalf("write source: %v", err)
			}

			if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1, Runtime: tc.runtime}); err != nil {
				t.Fatalf("build x32 task-group %s self-host runtime: %v", tc.name, err)
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
			if code != 7 {
				t.Fatalf("x32 task-group runtime exit=%d stdout=%q, want 7", code, stdout)
			}
		})
	}
}

func TestX32ExplicitBuiltinRuntimeStillRejects(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "time_x32_builtin.tetra")
	outPath := filepath.Join(tmp, "time-x32-builtin")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses runtime:
    return core.time_now_ms()
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1, Runtime: RuntimeBuiltin})
	if err == nil {
		t.Fatalf("expected x32 builtin runtime support diagnostic")
	}
	for _, want := range []string{"builtin runtime is not supported on target linux-x32", "runtime=selfhost"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
	if _, statErr := os.Stat(outPath); statErr == nil {
		t.Fatalf("x32 builtin runtime rejection wrote executable %s", outPath)
	}
}
