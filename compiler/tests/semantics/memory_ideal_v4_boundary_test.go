package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestMemoryIdealV4BorrowedViewUsedBeforeAsyncAwaitBoundary(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
async func ready() -> Int:
    return 1

async func caller(xs: borrow []u8) -> Int:
    let view: []u8 = xs.borrow()
    let before: Int = view.len
    let after: Int = await ready()
    return before + after

func main() -> Int:
    return 0
`)
}

func TestMemoryIdealV4BorrowedViewUsedBeforeTaskActorBoundary(t *testing.T) {
	t.Run("task_boundary_after_local_borrow_use", func(t *testing.T) {
		testkit.RequireFileCheckOK(t, `
func worker() -> Int:
    return 1

func main() -> Int
uses alloc, mem, runtime:
    var xs: []u8 = make_u8(1)
    let view: []u8 = xs.borrow()
    let before: Int = view.len
    let task: task.i32 = core.task_spawn_i32("worker")
    return before + core.task_join_i32(task)
`)
	})

	t.Run("actor_boundary_after_local_borrow_use", func(t *testing.T) {
		testkit.RequireFileCheckOK(t, `
func worker() -> Int:
    return 0

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(1)
    let view: []u8 = xs.borrow()
    let before: Int = view.len
    let peer: actor = core.spawn("worker")
    return core.send(peer, before)
`)
	})
}

func TestMemoryIdealV4BorrowedAsyncResultRejected(t *testing.T) {
	testkit.RequireFileCheckErrorContains(t, `
async func producer(x: borrow ptr) -> ptr:
    return x

async func caller(x: borrow ptr) -> ptr:
    return await producer(x)

func main() -> Int:
    return 0
`, "borrowed local 'x' cannot escape via return")
}

func TestMemoryIdealV4ActorBoundaryCopyAndBorrowDiagnostics(t *testing.T) {
	t.Run("copy_before_actor_send_accepted", func(t *testing.T) {
		testkit.RequireFileCheckOK(t, `
enum Msg:
    case bytes([]u8)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(2)
    xs[0] = 40
    xs[1] = 2
    return core.send_typed(core.self(), Msg.bytes(__method.copy(xs.borrow())))
`)
	})

	t.Run("borrowed_view_sent_to_actor_rejected", func(t *testing.T) {
		testkit.RequireFileCheckErrorContains(t, `
enum Msg:
    case bytes([]u8)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(2)
    return core.send_typed(core.self(), Msg.bytes(xs.borrow()))
`, "cannot send borrowed view across actor boundary")
	})

	t.Run("struct_wrapper_sent_to_actor_rejected", func(t *testing.T) {
		testkit.RequireFileCheckErrorContains(t, `
struct Box:
    bytes: []u8

enum Msg:
    case boxed(Box)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(2)
    return core.send_typed(core.self(), Msg.boxed(Box { bytes: xs.borrow() }))
`, "cannot cross actor boundary")
	})

	t.Run("optional_wrapper_sent_to_actor_rejected", func(t *testing.T) {
		testkit.RequireFileCheckErrorContains(t, `
enum Msg:
    case maybe([]u8?)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(2)
    let maybe: []u8? = xs.borrow()
    return core.send_typed(core.self(), Msg.maybe(maybe))
`, "optional wrapper")
	})

	t.Run("generic_wrapper_sent_to_actor_rejected", func(t *testing.T) {
		testkit.RequireFileCheckErrorContains(t, `
struct Box<T>:
    value: T

enum Msg:
    case boxed(Box<[]u8>)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(2)
    return core.send_typed(core.self(), Msg.boxed(Box<[]u8>{value: xs.borrow()}))
`, "cannot cross actor boundary")
	})
}

func TestMemoryIdealV4TaskBoundaryCurrentSurfaceDiagnostics(t *testing.T) {
	t.Run("copy_before_typed_task_boundary_accepted", func(t *testing.T) {
		testkit.RequireFileCheckOK(t, `
enum TaskErr:
    case failed

func worker() -> Int throws TaskErr:
    return 0

func main() -> Int
uses alloc, mem, runtime:
    var xs: []u8 = make_u8(1)
    xs[0] = 42
    let copied: []u8 = __method.copy(xs.borrow())
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.failed:
        copied[0]
`)
	})

	t.Run("typed_task_rejects_reference_shaped_error_payload", func(t *testing.T) {
		testkit.RequireFileCheckErrorContains(t, `
enum TaskErr:
    case bytes([]u8)

func worker() -> Int throws TaskErr:
    return 0

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.bytes(bytes):
        bytes.len
`, "typed task error payload must be sendable across task boundary")
	})

	t.Run("unknown_task_target_emits_no_trusted_boundary_facts", func(t *testing.T) {
		testkit.RequireFileCheckErrorContains(t, `
func worker() -> Int:
    return 0

func main() -> Int
uses runtime:
    let target: str = "worker"
    let task: task.i32 = core.task_spawn_i32(target)
    return core.task_join_i32(task)
`, "task_spawn_i32 expects a string literal")
	})
}

func TestMemoryIdealV4TaskActorBroadNoAliasRejected(t *testing.T) {
	t.Run("task_boundary_mutable_global_rejected", func(t *testing.T) {
		testkit.RequireFileCheckErrorContains(t, `
var g: Int

func worker() -> Int:
    g = g + 1
    return g

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32("worker")
    return core.task_join_i32(task)
`, "cannot cross task boundary")
	})

	t.Run("actor_boundary_mutable_global_rejected", func(t *testing.T) {
		testkit.RequireFileCheckErrorContains(t, `
var g: Int

func worker() -> Int:
    g = g + 1
    return g

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    return core.send(peer, 1)
`, "cannot cross actor boundary")
	})
}
