package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckCommandJSONDiagnosticsForResourceUseAfterFreeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_resource_free.tetra")
	src := `func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        free(isl)
        free(isl)
    }
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'isl'")
}

func TestCheckCommandJSONDiagnosticsForResourceStructFieldAliasUseAfterFreeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_resource_struct_field_alias_free.tetra")
	src := `struct IslandBox:
    handle: island

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: IslandBox = IslandBox(handle: core.island_new(16))
        let alias: island = box.handle
        free(box.handle)
        free(alias)
    }
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'alias'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleResourceStructFieldAliasUseAfterFreeCode(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub struct IslandBox:
    handle: island
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: resources.IslandBox = resources.IslandBox(handle: core.island_new(16))
        let alias: island = box.handle
        free(box.handle)
        free(alias)
    }
    return 0
`)
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'alias'")
}

func TestCheckCommandJSONDiagnosticsForResourceEnumPayloadAliasUseAfterFreeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_resource_enum_payload_alias_free.tetra")
	src := `enum MoveMsg:
    case take(island)

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let msg: MoveMsg = MoveMsg.take(core.island_new(16))
        match msg:
        case MoveMsg.take(other):
            let alias: island = other
            free(other)
            free(alias)
    }
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'alias'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleResourceEnumPayloadAliasUseAfterFreeCode(t *testing.T) {
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
    case take(island)

pub func unwrap(msg: MoveMsg) -> island:
    match msg:
    case MoveMsg.take(handle):
        return handle
`)
	srcPath := filepath.Join(dir, "src", "app", "main.t4")
	writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let msg: resources.MoveMsg = resources.MoveMsg.take(core.island_new(16))
        let other: island = resources.unwrap(msg)
        match msg:
        case resources.MoveMsg.take(handle):
            free(handle)
            free(other)
    }
    return 0
`)
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'other'")
}

func TestCheckCommandJSONDiagnosticsForResourceOptionalPayloadFreeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_resource_optional_payload_free.tetra")
	src := `func use(value: island?) -> Int:
    return 0

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let maybe: island? = isl
        match maybe:
        case some(other):
            free(other)
            return use(maybe)
        case none:
            return 0
    }
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'maybe.$elem'")
}

func TestCheckCommandJSONDiagnosticsForResourceOptionalWrapperAliasUseAfterFreeCodes(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "struct",
			src: `struct MaybeBox:
    maybe: island?

func pass(box: MaybeBox) -> MaybeBox:
    return box

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let box: MaybeBox = MaybeBox(maybe: isl)
        let returned: MaybeBox = pass(box)
        if let other = returned.maybe:
            free(isl)
            free(other)
    }
    return 0
`,
		},
		{
			name: "enum",
			src: `enum MaybeEnvelope:
    case wrap(island?)
    case empty

func pass(msg: MaybeEnvelope) -> MaybeEnvelope:
    return msg

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let msg: MaybeEnvelope = MaybeEnvelope.wrap(isl)
        let returned: MaybeEnvelope = pass(msg)
        match returned:
        case MaybeEnvelope.wrap(maybe):
            if let other = maybe:
                free(isl)
                free(other)
        case MaybeEnvelope.empty:
            return 0
    }
    return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_resource_optional_wrapper_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'other'")
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleResourceOptionalWrapperAliasUseAfterFreeCodes(t *testing.T) {
	tests := []struct {
		name   string
		libSrc string
		appSrc string
	}{
		{
			name: "struct",
			libSrc: `module lib.resources

pub struct MaybeBox:
    maybe: island?

pub func pass(box: MaybeBox) -> MaybeBox:
    return box
`,
			appSrc: `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let box: resources.MaybeBox = resources.MaybeBox(maybe: isl)
        let returned: resources.MaybeBox = resources.pass(box)
        if let other = returned.maybe:
            free(isl)
            free(other)
    }
    return 0
`,
		},
		{
			name: "enum",
			libSrc: `module lib.resources

pub enum MaybeEnvelope:
    case wrap(island?)
    case empty

pub func pass(msg: MaybeEnvelope) -> MaybeEnvelope:
    return msg
`,
			appSrc: `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let msg: resources.MaybeEnvelope = resources.MaybeEnvelope.wrap(isl)
        let returned: resources.MaybeEnvelope = resources.pass(msg)
        match returned:
        case resources.MaybeEnvelope.wrap(maybe):
            if let other = maybe:
                free(isl)
                free(other)
        case resources.MaybeEnvelope.empty:
            return 0
    }
    return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			writeCLIProjectFile(t, dir, "src/lib/resources.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", tt.appSrc)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'other'")
		})
	}
}

func TestCheckCommandJSONDiagnosticsForResourceDoubleJoinCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_resource_join.tetra")
	src := `func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(task)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'task'")
}

func TestCheckCommandJSONDiagnosticsForTaskGroupUseAfterCloseCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_task_group_close.tetra")
	src := `func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let first: Int = core.task_group_close(group)
    return first + core.task_group_close(group)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'group'")
}

func TestCheckCommandJSONDiagnosticsForResourceAmbiguousProvenanceCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_resource_provenance.tetra")
	src := `struct IslandBox:
    handle: island

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let left: island = core.island_new(16)
        let right: island = core.island_new(16)
        var box: IslandBox = IslandBox(handle: left)
        if 1:
            box = IslandBox(handle: right)
        free(box.handle)
    }
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "ambiguous resource provenance for 'box.handle'")
}

func TestCheckCommandJSONDiagnosticsForIslandTransferNonLocalPayloadCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_island_transfer_payload.tetra")
	src := `enum MoveMsg:
    case take(island)

func worker() -> Int:
    return 0

func main() -> Int
uses actors, alloc, islands, mem:
    let peer: actor = core.spawn("worker")
    unsafe:
        return core.send_typed(peer, MoveMsg.take(core.island_new(16)))
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "island transfer payload must be a local value")
}

func TestCheckCommandJSONDiagnosticsForActorUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_actor_use_after_transfer.tetra")
	src := `func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _: Int = take_actor(peer)
    return core.send(peer, 1)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'peer'")
}

func TestCheckCommandJSONDiagnosticsForActorBranchConsumeReuseCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_actor_branch_consume_reuse.tetra")
	src := `func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func use(flag: Int) -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    if flag:
        let _: Int = take_actor(peer)
    return core.send(peer, 1)

func main() -> Int
uses actors:
    return use(1)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'peer'")
}

func TestCheckCommandJSONDiagnosticsForActorMatchLoopConsumeReuseCodes(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "match",
			src: `enum Choice:
    case take
    case keep

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func use(choice: Choice) -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    match choice:
    case Choice.take:
        let taken: Int = take_actor(peer)
    case Choice.keep:
        let kept: Int = 0
    return core.send(peer, 1)

func main() -> Int
uses actors:
    return use(Choice.take)
`,
		},
		{
			name: "loop",
			src: `func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func use(limit: Int) -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    var i: Int = 0
    while i < limit:
        let _: Int = take_actor(peer)
        i = i + 1
    return core.send(peer, 1)

func main() -> Int
uses actors:
    return use(1)
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_actor_"+tt.name+"_consume_reuse.tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'peer'")
		})
	}
}

func TestCheckCommandJSONDiagnosticsForTaskUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_task_use_after_transfer.tetra")
	src := `func worker() -> Int:
    return 7

func take_task(task: consume task.i32) -> Int
uses runtime:
    return core.task_join_i32(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let value: Int = take_task(task)
    return value + core.task_join_i32(task)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'task'")
}

func TestCheckCommandJSONDiagnosticsForActorStructFieldAliasUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_actor_struct_field_alias_transfer.tetra")
	src := `struct ActorBox:
    peer: actor

func worker() -> Int:
    return 0

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: ActorBox = ActorBox(peer: peer)
    let _: Int = take_actor(peer)
    return core.send(box.peer, 1)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'box.peer'")
}

func TestCheckCommandJSONDiagnosticsForCrossModuleActorStructFieldAliasUseAfterTransferCode(t *testing.T) {
	dir := t.TempDir()
	writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
	writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub struct ActorBox:
    peer: actor

pub func unwrap(box: ActorBox) -> actor:
    return box.peer
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
    let box: resources.ActorBox = resources.ActorBox(peer: peer)
    let other: actor = resources.unwrap(box)
    let _: Int = take_actor(peer)
    return core.send(other, 1)
`)
	assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'other'")
}

func TestCheckCommandJSONDiagnosticsForGenericActorStructFieldAliasUseAfterTransferCodes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_actor_generic_struct_field_alias_transfer.tetra")
		src := `struct Box<T>:
    value: T

func worker() -> Int:
    return 0

func pass_actor(box: Box<actor>) -> Box<actor>:
    return box

func take_actor(peer: consume actor) -> Int:
    return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let box: Box<actor> = Box<actor>{value: peer}
    let returned: Box<actor> = pass_actor(box)
    let _: Int = take_actor(peer)
    return core.send(returned.value, 1)
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'returned.value'")
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

pub struct Box<T>:
    value: T

pub func pass_actor(box: Box<actor>) -> Box<actor>:
    return box
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
    let box: resources.Box<actor> = resources.Box<actor>{value: peer}
    let returned: resources.Box<actor> = resources.pass_actor(box)
    let _: Int = take_actor(peer)
    return core.send(returned.value, 1)
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use consumed value 'returned.value'")
	})
}

func TestCheckCommandJSONDiagnosticsForGenericResourceAliasFinalizationCodes(t *testing.T) {
	t.Run("same module task-handle", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_task_generic_struct_alias_join.tetra")
		src := `struct Box<T>:
    value: T

func worker() -> Int:
    return 7

func pass_task(box: Box<task.i32>) -> Box<task.i32>:
    return box

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let box: Box<task.i32> = Box<task.i32>{value: task}
    let returned: Box<task.i32> = pass_task(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.value)
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'returned.value'")
	})

	t.Run("cross module task-handle", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub struct Box<T>:
    value: T

pub func pass_task(box: Box<task.i32>) -> Box<task.i32>:
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
    let box: resources.Box<task.i32> = resources.Box<task.i32>{value: task}
    let returned: resources.Box<task.i32> = resources.pass_task(box)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(returned.value)
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'returned.value'")
	})

	t.Run("same module task-group", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_group_generic_struct_alias_close.tetra")
		src := `struct Box<T>:
    value: T

func pass_group(box: Box<task.group>) -> Box<task.group>:
    return box

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let box: Box<task.group> = Box<task.group>{value: group}
    let returned: Box<task.group> = pass_group(box)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(returned.value)
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'returned.value'")
	})

	t.Run("cross module task-group", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub struct Box<T>:
    value: T

pub func pass_group(box: Box<task.group>) -> Box<task.group>:
    return box
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let box: resources.Box<task.group> = resources.Box<task.group>{value: group}
    let returned: resources.Box<task.group> = resources.pass_group(box)
    let _: Int = core.task_group_close(group)
    return core.task_group_close(returned.value)
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'returned.value'")
	})

	t.Run("same module island", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_island_generic_struct_alias_free.tetra")
		src := `struct Box<T>:
    value: T

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: Box<island> = Box<island>{value: core.island_new(16)}
        let alias: Box<island> = box
        free(box.value)
        free(alias.value)
    }
    return 0
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'alias.value'")
	})

	t.Run("cross module island", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub struct Box<T>:
    value: T
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let box: resources.Box<island> = resources.Box<island>{value: core.island_new(16)}
        let alias: resources.Box<island> = box
        free(box.value)
        free(alias.value)
    }
    return 0
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'alias.value'")
	})
}

func TestCheckCommandJSONDiagnosticsForTransitiveResourceAliasFinalizationCodes(t *testing.T) {
	t.Run("same module task-handle", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_task_transitive_alias_join.tetra")
		src := `func worker() -> Int:
    return 7

func alias_one(task: task.i32) -> task.i32:
    return task

func alias_two(task: task.i32) -> task.i32:
    return alias_one(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let other: task.i32 = alias_two(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(other)
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
	})

	t.Run("cross module task-handle", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub func alias_one(task: task.i32) -> task.i32:
    return task

pub func alias_two(task: task.i32) -> task.i32:
    return alias_one(task)
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func worker() -> Int:
    return 7

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let other: task.i32 = resources.alias_two(task)
    let first: Int = core.task_join_i32(task)
    return first + core.task_join_i32(other)
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
	})

	t.Run("same module task-group", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_group_transitive_alias_close.tetra")
		src := `func alias_one(group: task.group) -> task.group:
    return group

func alias_two(group: task.group) -> task.group:
    return alias_one(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let other: task.group = alias_two(group)
    let first: Int = core.task_group_close(group)
    return first + core.task_group_close(other)
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'other'")
	})

	t.Run("cross module task-group", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub func alias_one(group: task.group) -> task.group:
    return group

pub func alias_two(group: task.group) -> task.group:
    return alias_one(group)
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let other: task.group = resources.alias_two(group)
    let first: Int = core.task_group_close(group)
    return first + core.task_group_close(other)
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'other'")
	})

	t.Run("same module island", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_island_transitive_alias_free.tetra")
		src := `func alias_one(isl: island) -> island:
    return isl

func alias_two(isl: island) -> island:
    return alias_one(isl)

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = alias_two(isl)
        free(isl)
        free(other)
    }
    return 0
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'other'")
	})

	t.Run("cross module island", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub func alias_one(isl: island) -> island:
    return isl

pub func alias_two(isl: island) -> island:
    return alias_one(isl)
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let other: island = resources.alias_two(isl)
        free(isl)
        free(other)
    }
    return 0
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use freed resource 'other'")
	})
}

func TestCheckCommandJSONDiagnosticsForEnumConstructorReturnResourceAliasCodes(t *testing.T) {
	t.Run("same module task-handle", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_task_enum_constructor_return_alias_join.tetra")
		src := `enum TaskMsg:
    case wrap(task.i32)

func worker() -> Int:
    return 7

func wrap(task: task.i32) -> TaskMsg:
    return TaskMsg.wrap(task)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    let returned: TaskMsg = wrap(task)
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
	})

	t.Run("cross module task-handle", func(t *testing.T) {
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

pub func wrap(task: task.i32) -> TaskMsg:
    return TaskMsg.wrap(task)
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
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
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use joined resource 'other'")
	})

	t.Run("same module task-group", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_group_enum_constructor_return_alias_close.tetra")
		src := `enum GroupMsg:
    case wrap(task.group)

func wrap(group: task.group) -> GroupMsg:
    return GroupMsg.wrap(group)

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let returned: GroupMsg = wrap(group)
    match returned:
    case GroupMsg.wrap(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'other'")
	})

	t.Run("cross module task-group", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		writeCLIProjectFile(t, dir, "src/lib/resources.t4", `module lib.resources

pub enum GroupMsg:
    case wrap(task.group)

pub func wrap(group: task.group) -> GroupMsg:
    return GroupMsg.wrap(group)
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.resources as resources

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let returned: resources.GroupMsg = resources.wrap(group)
    match returned:
    case resources.GroupMsg.wrap(other):
        let _: Int = core.task_group_close(group)
        return core.task_group_close(other)
    return 0
`)
		assertCLIJSONOwnershipDiagnostic(t, srcPath, "cannot use closed resource 'other'")
	})

}
