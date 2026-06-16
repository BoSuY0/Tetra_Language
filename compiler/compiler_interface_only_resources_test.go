package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildRejectsInterfaceOnlyDependencyWithoutInterfaceOnlyMode(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`), "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4":   "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n",
		"math/core.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "app"),
		"linux-x64",
		BuildOptions{Jobs: 1},
	)
	if err == nil {
		t.Fatalf("expected interface-only dependency build rejection")
	}
	if !strings.Contains(err.Error(), "missing implementation object for interface module 'math.core'") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildInterfaceOnlyModeAllowsT4IDependencyWithoutOutput(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`), "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4":   "module app.main\nimport math.core as math\nfunc main() -> Int:\n    return math.add(40, 2)\n",
		"math/core.t4i": string(iface),
	})

	outPath := filepath.Join(tmp, "out", "app")
	stats, err := BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		outPath,
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err != nil {
		t.Fatalf("BuildFileWithStatsOpt interface-only: %v", err)
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("interface-only build should not emit %s, stat err=%v", outPath, err)
	}
	if len(stats.InterfaceModules) != 1 || stats.InterfaceModules[0] != "math.core" {
		t.Fatalf("InterfaceModules = %#v, want [math.core]", stats.InterfaceModules)
	}
}

func TestBuildInterfaceOnlyModeRejectsTamperedBorrowedReturnLifetimeMetadata(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.views

pub func view(xs: borrow []u8) -> borrow []u8:
    return xs.borrow()
`), "lib/views.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	tampered := strings.Replace(string(iface), "source=xs", "source=ys", 1)
	if tampered == string(iface) {
		t.Fatalf("test fixture did not find borrowed return lifetime metadata:\n%s", iface)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.views as views

func relay(xs: borrow []u8) -> borrow []u8:
    return views.view(xs)

func main() -> Int:
    return 0
`,
		"lib/views.t4i": tampered,
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil || !strings.Contains(err.Error(), "invalid .t4i hash") {
		t.Fatalf("BuildFileWithStatsOpt tampered lifetime metadata error = %v, want invalid .t4i hash", err)
	}
}

func TestBuildInterfaceOnlyModePreservesTypedErrorResourceThrowProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub enum TaskErr:
    case wrap(task.i32)

pub func fail(task: task.i32) -> Int throws TaskErr
uses runtime:
    throw TaskErr.wrap(task)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return catch resources.fail(task):
    case resources.TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only typed-error resource provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesTypedErrorFieldLocalAliasResourceThrowProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub enum TaskErr:
    case wrap(task.i32)

pub func fail(box: TaskBox) -> Int throws TaskErr
uses runtime:
    let other: task.i32 = box.handle
    throw TaskErr.wrap(other)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    return catch resources.fail(box):
    case resources.TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only typed-error field local alias resource provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesTypedErrorResourceRethrowProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub enum TaskErr:
    case wrap(task.i32)

pub func fail(task: task.i32) -> Int throws TaskErr
uses runtime:
    throw TaskErr.wrap(task)

pub func wrapper(task: task.i32) -> Int throws TaskErr
uses runtime:
    return try fail(task)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return catch resources.wrapper(task):
    case resources.TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only typed-error rethrow resource provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesTypedErrorFieldLocalAliasResourceRethrowProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub enum TaskErr:
    case wrap(task.i32)

pub func fail(task: task.i32) -> Int throws TaskErr
uses runtime:
    throw TaskErr.wrap(task)

pub func wrapper(box: TaskBox) -> Int throws TaskErr
uses runtime:
    let other: task.i32 = box.handle
    return try fail(other)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    return catch resources.wrapper(box):
    case resources.TaskErr.wrap(other):
        core.task_join_i32(task) + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only typed-error field local alias rethrow resource provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub enum TaskMsg:
    case wrap(task.i32)

pub func wrap(task: task.i32) -> TaskMsg:
    return TaskMsg.wrap(task)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: resources.TaskMsg = resources.wrap(task)
    match returned:
    case resources.TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub func maybe(task: task.i32) -> task.i32?:
    var out: task.i32? = none
    out = task
    return out
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: task.i32? = resources.maybe(task)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesLocalAliasResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub func alias(task: task.i32) -> task.i32:
    let other: task.i32 = task
    return other
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let other: task.i32 = resources.alias(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(other)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only local alias resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesAggregateLocalAliasResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func box(task: task.i32) -> TaskBox:
    let other: task.i32 = task
    return TaskBox(handle: other)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: resources.TaskBox = resources.box(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only aggregate local alias resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'returned.handle'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesAggregateFieldResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func pass(box: TaskBox) -> TaskBox:
    return TaskBox(handle: box.handle)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    let returned: resources.TaskBox = resources.pass(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only aggregate field resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'returned.handle'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesAggregateFieldLocalAliasResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func pass(box: TaskBox) -> TaskBox:
    let other: task.i32 = box.handle
    return TaskBox(handle: other)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    let returned: resources.TaskBox = resources.pass(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only aggregate field local alias resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'returned.handle'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesLetOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub func maybe(task: task.i32) -> task.i32?:
    let out: task.i32? = task
    return out
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: task.i32? = resources.maybe(task)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only let optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesOptionalFieldLocalAliasResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func maybe(box: TaskBox) -> task.i32?:
    let out: task.i32? = box.handle
    return out
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: resources.TaskBox = resources.TaskBox(handle: task)
    let returned: task.i32? = resources.maybe(box)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only optional field local alias resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesDirectIfLetOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    maybe: task.i32?

pub func maybe(input: TaskBox) -> task.i32?:
    if let other = input.maybe:
        return other
    else:
        return none
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.TaskBox = resources.TaskBox(maybe: task)
    let returned: task.i32? = resources.maybe(input)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only direct if-let optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesDirectMatchOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    maybe: task.i32?

pub func maybe(input: TaskBox) -> task.i32?:
    match input.maybe:
    case some(other):
        return other
    case none:
        return none
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.TaskBox = resources.TaskBox(maybe: task)
    let returned: task.i32? = resources.maybe(input)
    if let other = returned:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only direct match optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesStructOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    maybe: task.i32?

pub func box(task: task.i32) -> TaskBox:
    var out: task.i32? = none
    out = task
    return TaskBox(maybe: out)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: resources.TaskBox = resources.box(task)
    if let other = returned.maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only struct optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesStructOptionalFieldLocalAliasResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct TaskBox:
    maybe: task.i32?

pub struct InputBox:
    handle: task.i32

pub func box(input: InputBox) -> TaskBox:
    let out: task.i32? = input.handle
    return TaskBox(maybe: out)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.InputBox = resources.InputBox(handle: task)
    let returned: resources.TaskBox = resources.box(input)
    if let other = returned.maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only struct optional field local alias resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesIfLetOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct InputBox:
    maybe: task.i32?

pub struct TaskBox:
    maybe: task.i32?

pub func box(input: InputBox) -> TaskBox:
    if let other = input.maybe:
        return TaskBox(maybe: other)
    else:
        return TaskBox(maybe: none)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.InputBox = resources.InputBox(maybe: task)
    let returned: resources.TaskBox = resources.box(input)
    if let other = returned.maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only if-let optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}

func TestBuildInterfaceOnlyModePreservesMatchOptionalResourceReturnProvenance(t *testing.T) {
	tmp := t.TempDir()
	iface, err := GenerateInterfaceFromSource([]byte(`module lib.resources

pub struct InputBox:
    maybe: task.i32?

pub struct TaskBox:
    maybe: task.i32?

pub func box(input: InputBox) -> TaskBox:
    match input.maybe:
    case some(other):
        return TaskBox(maybe: other)
    case none:
        return TaskBox(maybe: none)
`), "lib/resources.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let input: resources.InputBox = resources.InputBox(maybe: task)
    let returned: resources.TaskBox = resources.box(input)
    if let other = returned.maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    else:
        return 0
`,
		"lib/resources.t4i": string(iface),
	})

	_, err = BuildFileWithStatsOpt(
		filepath.Join(tmp, filepath.FromSlash("app/main.t4")),
		filepath.Join(tmp, "out", "app"),
		"linux-x64",
		BuildOptions{Jobs: 1, InterfaceOnly: true},
	)
	if err == nil {
		t.Fatalf("expected interface-only match optional resource return provenance diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot use joined resource 'other'") {
		t.Fatalf("error = %v, want joined resource alias diagnostic", err)
	}
}
