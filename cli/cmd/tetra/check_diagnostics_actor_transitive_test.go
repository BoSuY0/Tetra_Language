package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
)

func TestCheckCommandJSONDiagnosticsForTransitiveActorAliasUseAfterTransferCodes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_actor_transitive_alias_transfer.tetra")
		src := `func worker() -> Int:
    return 0

func alias_one(peer: actor) -> actor:
    return peer

func alias_two(peer: actor) -> actor:
    return alias_one(peer)

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let other: actor = alias_two(peer)
    let _: Int = take_actor(peer)
    return core.send(other, 1)
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'other'")
	})

	t.Run("cross module", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub func alias_one(peer: actor) -> actor:
    return peer

pub func alias_two(peer: actor) -> actor:
    return alias_one(peer)
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let other: actor = resources.alias_two(peer)
    let _: Int = take_actor(peer)
    return core.send(other, 1)
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'other'")
	})
}

func TestCheckCommandJSONDiagnosticsForTaskGroupCancelReturnProvenanceCodes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_task_group_cancel_return_provenance.tetra")
		src := `func cancel(group: task.group) -> task.group
uses runtime:
    return core.task_group_cancel(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let canceled: task.group = cancel(group)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(canceled)
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'canceled'")
	})

	t.Run("cross module", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub func cancel(group: task.group) -> task.group
uses runtime:
    return core.task_group_cancel(group)
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let canceled: task.group = resources.cancel(group)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(canceled)
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'canceled'")
	})
}

func TestCheckCommandJSONDiagnosticsForTaskHandleGroupOptionalPayloadJoinCloseAliasCodes(t *testing.T) {
	t.Run("same module task-handle if-let optional-payload join", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_task_optional_payload_join_alias.tetra")
		src := `func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    if let other = maybe:
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
	})

	t.Run("cross module task-handle match optional-payload join", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub func pass(maybe: task.i32?) -> task.i32?:
    return maybe
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let maybe: task.i32? = task
    let returned: task.i32? = resources.pass(maybe)
    match returned:
    case some(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    case none:
        return 0
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
	})

	t.Run("same module task-group if-let optional-payload close", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_group_optional_payload_close_alias.tetra")
		src := `func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    if let other = maybe:
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'other'")
	})

	t.Run("cross module task-group match optional-payload close", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub func pass(maybe: task.group?) -> task.group?:
    return maybe
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    let returned: task.group? = resources.pass(maybe)
    match returned:
    case some(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    case none:
        return 0
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'other'")
	})
}

func TestCheckCommandJSONDiagnosticsForActorEnumPayloadAliasUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_actor_enum_payload_alias_transfer.tetra")
	src := `enum MoveMsg:
    case handoff(actor)

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let msg: MoveMsg = MoveMsg.handoff(peer)
    match msg:
    case MoveMsg.handoff(other):
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'other'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleActorEnumPayloadAliasUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub enum MoveMsg:
    case handoff(actor)

pub func pass(msg: MoveMsg) -> MoveMsg:
    return msg
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let msg: resources.MoveMsg = resources.MoveMsg.handoff(peer)
    let returned: resources.MoveMsg = resources.pass(msg)
    match returned:
    case resources.MoveMsg.handoff(other):
        let _: Int = take_actor(peer)
        return core.send(other, 1)
    return 0
`)
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'other'")
}

func TestCheckCommandJSONDiagnosticsForTaskStructFieldAliasUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_task_struct_field_alias_transfer.tetra")
	src := `struct TaskBox:
    handle: task.i32

func worker() -> Int:
    return 7

func pass(box: TaskBox) -> TaskBox:
    return box

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: TaskBox = TaskBox(handle: task)
    let returned: TaskBox = pass(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.handle)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'returned.handle'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleTaskStructFieldAliasUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub struct TaskBox:
    handle: task.i32

pub func pass(box: TaskBox) -> TaskBox:
    return box
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
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
`)
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'returned.handle'")
}

func TestCheckCommandJSONDiagnosticsForTaskEnumPayloadAliasUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_task_enum_payload_alias_transfer.tetra")
	src := `enum TaskMsg:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func pass(msg: TaskMsg) -> TaskMsg:
    return msg

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let msg: TaskMsg = TaskMsg.wrap(task)
    let returned: TaskMsg = pass(msg)
    match returned:
    case TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleTaskEnumPayloadAliasUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub enum TaskMsg:
    case wrap(task.i32)

pub func pass(msg: TaskMsg) -> TaskMsg:
    return msg
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let msg: resources.TaskMsg = resources.TaskMsg.wrap(task)
    let returned: resources.TaskMsg = resources.pass(msg)
    match returned:
    case resources.TaskMsg.wrap(other):
        let first: Int = core.task_join_i32(task)
        return first + core.task_join_i32(other)
    return 0
`)
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
}

func TestCheckCommandJSONDiagnosticsForPrivacyConsentSafetyCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_privacy.tetra")
	src := `func seal(token: consent.token) -> secret.i32
uses privacy:
    return core.secret_seal_i32(1, token)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONDiagnosticForPath(t, srcPath, srcPath, compiler.DiagnosticCodeSafetyPrivacy, "uses effect 'privacy' requires semantic clause 'privacy'")
}

func TestCheckCommandJSONDiagnosticsForRecursiveSecretSignaturePrivacyCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_secret_signature.tetra")
	src := `func seal(payload: secret.i32?) -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"check", "--diagnostics=json", srcPath}, 1)
	if diag.Code != compiler.DiagnosticCodeSafetyPrivacy || diag.File != srcPath || diag.Severity != "error" || diag.Message != "secret types in function signature require semantic clause 'privacy'" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestCheckCommandJSONDiagnosticsForTooManyInputs(t *testing.T) {
	diag := runCLIJSONDiagnostic(t, []string{"check", "--diagnostics=json", "one.tetra", "two.tetra"}, 2)
	if diag.Code != "TETRA0001" || diag.Message != "check accepts at most one input path" || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestCheckCommandRejectsLocalCapsuleDependencyCycle(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "App/Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
    deps:
        tetra://math 0.1.0 ../Math
`)
	writeCLIProjectFile(t, dir, "App/src/app/main.t4", "module app.main\nfunc main() -> Int:\n    return 0\n")
	writeCLIProjectFile(t, dir, "Math/Capsule.t4", `capsule Math:
    id "tetra://math"
    version "0.1.0"
    sources:
        src
    deps:
        tetra://app 0.1.0 ../App
`)
	writeCLIProjectFile(t, dir, "Math/src/math/core.t4", "module math.core\nfunc add(a: Int, b: Int) -> Int:\n    return a + b\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join(dir, "App")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"check"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected check failure for dependency cycle, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "capsule dependency cycle") {
		t.Fatalf("stderr = %q, want capsule dependency cycle", stderr.String())
	}
}
