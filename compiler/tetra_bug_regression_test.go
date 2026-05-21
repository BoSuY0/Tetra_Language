package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func requireTetraBugLinuxAMD64(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
}

func requireTetraBugInterfaceOnlyBuild(t *testing.T, src string) {
	t.Helper()
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "out")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{InterfaceOnly: true, Jobs: 4}); err != nil {
		t.Fatalf("interface-only build: %v", err)
	}
}

func TestTetraBug0001GenericInferenceDirectCallArgument(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func id<T>(x: T) -> T:
    return x

func value() -> Int:
    return 42

func main() -> Int:
    return id(value())
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0002ModuleExtensionStaticCall(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRunFiles(t, map[string]string{
		"app/main.tetra": `module app.main

struct Unit:
    value: Int

protocol Score:
    func score(self: Unit) -> Int

extension Unit:
    func score(self: Unit) -> Int:
        return self.value

impl Unit: Score

func main() -> Int:
    let unit: Unit = Unit(value: 42)
    return Unit.score(unit)
`,
	}, "app/main.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0003FunctionTypedStructFieldEnumPayload(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
struct Handler:
    cb: fn(Int) -> Int

enum Route:
    case direct(fn(Int) -> Int)
    case fallback

func main() -> Int:
    let offset: Int = 13
    let captured: ptr = fn(value: Int) -> Int:
        return value + offset
    let handler: Handler = Handler(cb: captured)
    let route: Route = Route.direct(handler.cb)
    match route:
    case Route.direct(route_cb):
        return route_cb(29)
    case Route.fallback:
        return 0
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0004FormatterPreservesFunctionTypedLocalAnnotation(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	src := `
struct Handler:
    cb: fn(Int) -> Int

enum Route:
    case direct(fn(Int) -> Int)
    case fallback

func main() -> Int:
    let offset: Int = 13
    let captured: ptr = fn(value: Int) -> Int:
        return value + offset
    let handler: Handler = Handler(cb: captured)
    let field_cb: fn(Int) -> Int = handler.cb
    let route: Route = Route.direct(field_cb)
    match route:
    case Route.direct(route_cb):
        return route_cb(29)
    case Route.fallback:
        return 0
`
	formatted, err := FormatSource([]byte(src), "callable_alias.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if !strings.Contains(string(formatted), "let field_cb: fn(Int) -> Int = handler.cb") {
		t.Fatalf("formatted source lost function-typed local annotation:\n%s", string(formatted))
	}
	stdout, code := buildAndRun(t, string(formatted))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0005ModuleActorEntrypointString(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRunFiles(t, map[string]string{
		"app/main.tetra": `module app.main

actor Router:
    func run() -> Int
    uses actors:
        let _sent: Int = core.send_msg(core.sender(), 42, 7)
        return 0

func main() -> Int
uses actors:
    let router: actor = core.spawn("Router.run")
    let _request: Int = core.send_msg(router, 41, 6)
    let reply: actor.msg = core.recv_msg()
    return reply.value
`,
	}, "app/main.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0006FormatterPreservesFunctionTypedGlobalAnnotation(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	src := `
var callback: fn(Int) -> Int = identity

func identity(value: Int) -> Int:
    return value

func main() -> Int:
    return callback(42)
`
	formatted, err := FormatSource([]byte(src), "global_fn_format.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if !strings.Contains(string(formatted), "var callback: fn(Int) -> Int = identity") {
		t.Fatalf("formatted source lost function-typed global annotation:\n%s", string(formatted))
	}
	stdout, code := buildAndRun(t, string(formatted))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0007DerivedPtrAddComposesProvenance(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        let base: ptr = core.alloc_bytes(16)
        let dst: ptr = core.ptr_add(base, 8, memory_cap)
        let dst_one: ptr = core.ptr_add(dst, 1, memory_cap)
        let _write: UInt8 = core.store_u8(dst_one, 42, memory_cap)
        return core.load_u8(dst_one, memory_cap)
    return 99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0008FormatterPreservesMutableActorState(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	src := `
actor Counter:
    var count: Int = 0
    func run() -> Int
    uses actors:
        count = count + 1
        let _reply: Int = core.send_msg(core.sender(), count, 1)
        return 0

func main() -> Int
uses actors:
    let counter: actor = core.spawn("Counter.run")
    let _request: Int = core.send_msg(counter, 0, 0)
    let reply: actor.msg = core.recv_msg()
    return reply.value
`
	formatted, err := FormatSource([]byte(src), "actor_state_var_format.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if !strings.Contains(string(formatted), "actor Counter:\n    var count: Int = 0") {
		t.Fatalf("formatted source lost mutable actor state:\n%s", string(formatted))
	}
	stdout, code := buildAndRun(t, string(formatted))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

func TestTetraBug0009DualBlockingRecvMsgFanIn(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code, timedOut := buildAndRunWithOptionsTimeout(t, `
actor Adder:
    func run() -> Int
    uses actors:
        let request: actor.msg = core.recv_msg()
        let _reply: Int = core.send_msg(core.sender(), request.value + 1, request.tag + 10)
        return 0

actor Multiplier:
    func run() -> Int
    uses actors:
        let request: actor.msg = core.recv_msg()
        let _reply: Int = core.send_msg(core.sender(), request.value * 2, request.tag + 20)
        return 0

func main() -> Int
uses actors:
    let adder: actor = core.spawn("Adder.run")
    let multiplier: actor = core.spawn("Multiplier.run")
    let _left: Int = core.send_msg(adder, 20, 1)
    let _right: Int = core.send_msg(multiplier, 21, 2)
    let first: actor.msg = core.recv_msg()
    let second: actor.msg = core.recv_msg()
    let value_total: Int = first.value + second.value
    let tag_total: Int = first.tag + second.tag
    if tag_total != 33:
        return tag_total
    if value_total == 63:
        return 0
    return value_total
`, BuildOptions{}, 500*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestTetraBug0010DualBlockingRecvValueFanIn(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code, timedOut := buildAndRunWithOptionsTimeout(t, `
func adder() -> Int
uses actors:
    let request: Int = core.recv()
    let _reply: Int = core.send(core.sender(), request + 1)
    return 0

func multiplier() -> Int
uses actors:
    let request: Int = core.recv()
    let _reply: Int = core.send(core.sender(), request * 2)
    return 0

func main() -> Int
uses actors:
    let adder_actor: actor = core.spawn("adder")
    let multiplier_actor: actor = core.spawn("multiplier")
    let _left: Int = core.send(adder_actor, 20)
    let _right: Int = core.send(multiplier_actor, 21)
    let first: Int = core.recv()
    let second: Int = core.recv()
    let total: Int = first + second
    if total == 63:
        return 0
    return total
`, BuildOptions{}, 500*time.Millisecond)
	if timedOut {
		t.Fatalf("program timed out")
	}
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestTetraBug0011StructConstructorOptionalFieldCoercion(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
struct MaybeBox:
    value: Int?

func filled(value: Int) -> MaybeBox:
    return MaybeBox(value: value)

func main() -> Int:
    let box: MaybeBox = filled(42)
    if let value = box.value:
        return value
    return 0
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0012EnumConstructorOptionalPayloadCoercion(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
enum Route:
    case ready(Int?)
    case empty

func make_ready(value: Int) -> Route:
    return Route.ready(value)

func score(route: Route) -> Int:
    match route:
    case Route.ready(maybe):
        if let value = maybe:
            return value
        return 0
    case Route.empty:
        return 0

func main() -> Int:
    let route: Route = make_ready(42)
    return score(route)
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0013DerivedPtrParamLoopPreservesProvenance(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func copy_loop(dst: ptr, src: ptr, n: Int) -> Int
uses capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        var i: Int = 0
        while i < n:
            let sp: ptr = core.ptr_add(src, i, memory_cap)
            let dp: ptr = core.ptr_add(dst, i, memory_cap)
            let b: UInt8 = core.load_u8(sp, memory_cap)
            let _: UInt8 = core.store_u8(dp, b, memory_cap)
            i = i + 1
        return 0
    return 99

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        let source: ptr = core.alloc_bytes(8)
        let target: ptr = core.alloc_bytes(8)
        let source_two: ptr = core.ptr_add(source, 2, memory_cap)
        let source_three: ptr = core.ptr_add(source, 3, memory_cap)
        let target_one: ptr = core.ptr_add(target, 1, memory_cap)
        let target_two: ptr = core.ptr_add(target, 2, memory_cap)
        let _write_two: UInt8 = core.store_u8(source_two, 20, memory_cap)
        let _write_three: UInt8 = core.store_u8(source_three, 22, memory_cap)
        let _copy: Int = copy_loop(target_one, source_two, 2)
        return core.load_u8(target_one, memory_cap) + core.load_u8(target_two, memory_cap)
    return 98
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0014FormatterPreservesGenericProtocolRequirementTypeParams(t *testing.T) {
	src := `
struct Vec2:
    x: Int

protocol Mapper:
    func map<T>(self: Vec2, value: T) -> T
`
	formatted, err := FormatSource([]byte(src), "generic_protocol_fmt.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if !strings.Contains(string(formatted), "func map<T>(self: Vec2, value: T) -> T") {
		t.Fatalf("formatted source lost generic protocol requirement type params:\n%s", string(formatted))
	}
}

func TestTetraBug0015ImportedGenericExtensionStaticCallMonomorphizes(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRunFiles(t, map[string]string{
		"engine/core.tetra": `module engine.core

struct Vec2:
    x: Int
`,
		"app/ext.tetra": `module app.ext

import engine.core as core

extension core.Vec2:
    func map<T>(self: core.Vec2, value: T) -> T:
        return value
`,
		"app/main.tetra": `module app.main

import app.ext as ext
import engine.core as core

func main() -> Int:
    let value: core.Vec2 = core.Vec2(x: 7)
    return core.Vec2.map(value, 42)
`,
	}, "app/main.tetra")
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0016MatchCasePayloadBindingsAreSiblingScoped(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
enum Msg:
    case left(Int)
    case right(Int)

func main() -> Int:
    let msg: Msg = Msg.left(42)
    match msg:
    case Msg.left(value):
        return value
    case Msg.right(value):
        return value
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0020GenericIdentityAcceptsFunctionTypedLocal(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func add_one(value: Int) -> Int:
    return value + 1

func id<T>(value: T) -> T:
    return value

func apply(cb: fn(Int) -> Int, value: Int) -> Int:
    return cb(value)

func main() -> Int:
    let handler: fn(Int) -> Int = add_one
    let routed: fn(Int) -> Int = id(handler)
    return apply(routed, 41)
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0022GenericCallbackAcceptsFunctionSymbol(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func add_one(value: Int) -> Int:
    return value + 1

func apply_generic<T>(cb: fn(T) -> T, value: T) -> T:
    return cb(value)

func main() -> Int:
    return apply_generic(add_one, 41)
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0031EnumPayloadAcceptsGenericStructInstantiation(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
struct Box<T>:
    value: T

enum Route:
    case ready(Box<Int>)
    case empty

func main() -> Int:
    let box: Box<Int> = Box<Int>(value: 42)
    let route: Route = Route.ready(box)
    match route:
    case Route.ready(payload):
        return payload.value
    case Route.empty:
        return 0
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0033TypedTaskHandleCanUsePublicTaskAnnotation(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
enum TaskErr:
    case boom(Int)
    case stopped

func worker() -> Int throws TaskErr:
    throw TaskErr.boom(42)

func main() -> Int
uses runtime:
    let task: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker")
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        7
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0033TypedTaskPublicHandleContainersAndGroup(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
enum TaskErr:
    case boom(Int)
    case stopped

struct TaskBox:
    handle: task.i32

func worker_42() -> Int throws TaskErr:
    throw TaskErr.boom(42)

func worker_7() -> Int throws TaskErr:
    throw TaskErr.boom(7)

func worker_5() -> Int throws TaskErr:
    throw TaskErr.boom(5)

func worker_3() -> Int throws TaskErr:
    throw TaskErr.boom(3)

func join_public(task: task.i32) -> Int
uses runtime:
    return catch core.task_join_i32_typed<TaskErr>(task):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        0

func main() -> Int
uses runtime:
    let task_param: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker_42")
    let a: Int = join_public(task_param)

    let task_field: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker_7")
    let box: TaskBox = TaskBox(handle: task_field)
    let b: Int = catch core.task_join_i32_typed<TaskErr>(box.handle):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        0

    let task_optional: task.i32 = core.task_spawn_i32_typed<TaskErr>("worker_5")
    let maybe: task.i32? = task_optional
    var c: Int = 0
    if let task = maybe:
        c = catch core.task_join_i32_typed<TaskErr>(task):
        case TaskErr.boom(code):
            code
        case TaskErr.stopped:
            0

    let group: task.group = core.task_group_open()
    let task_group: task.i32 = core.task_spawn_group_i32_typed<TaskErr>(group, "worker_3")
    let d: Int = catch core.task_join_group_i32_typed<TaskErr>(task_group):
    case TaskErr.boom(code):
        code
    case TaskErr.stopped:
        0
    let _closed: Int = core.task_group_close(group)

    return a + b + c + d
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 57 {
		t.Fatalf("exit code = %d, want 57", code)
	}
}

func TestTetraBug0035TypedActorRejectsMismatchedEnumMessageType(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
enum Request:
    case value(Int)
    case stop

enum Reply:
    case failed(Int)
    case value(Int)

func worker() -> Int
uses actors:
    let msg: Reply = core.recv_typed<Reply>()
    match msg:
    case Reply.failed(code):
        let _failed: Int = core.send_typed(core.sender(), Reply.failed(code))
        return 0
    case Reply.value(value):
        let _reply: Int = core.send_typed(core.sender(), Reply.value(value + 1))
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("worker")
    let _sent: Int = core.send_typed(peer, Request.value(41))
    let reply: Reply = core.recv_typed<Reply>()
    match reply:
    case Reply.failed(code):
        return code
    case Reply.value(value):
        return value
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 255 {
		t.Fatalf("exit code = %d, want 255 mismatch sentinel", code)
	}
}

func TestTetraBug0037GlobalFixedArrayWriteRoundTrips(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
var seed: [3]Int

func main() -> Int:
    if seed.len != 3:
        return 10 + seed.len
    seed[0] = 42
    return seed[0]
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0037ZeroedGlobalFixedArrayReadsZero(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
var seed: [3]Int

func main() -> Int:
    if seed.len != 3:
        return 10 + seed.len
    return seed[0]
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestTetraBug0037GlobalStructFixedArrayFieldRoundTrips(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
struct ArrayBox:
    items: [3]Int

var box: ArrayBox

func main() -> Int:
    if box.items.len != 3:
        return 10 + box.items.len
    box.items[0] = 42
    return box.items[0]
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0038ScalarInoutWritesBackCallerLocal(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func bump(value: inout Int) -> Int:
    value = value + 1
    return value

func main() -> Int:
    var score: Int = 41
    let result: Int = bump(score)
    if result == 42 && score == 42:
        return 0
    return result + score
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestTetraBug0038FunctionTypedInoutWritesBackCallerLocal(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func bump(value: inout Int) -> Int:
    value = value + 1
    return value

func main() -> Int:
    let cb: fn(inout Int) -> Int = bump
    var score: Int = 41
    let result: Int = cb(score)
    if result == 42 && score == 42:
        return 0
    return result + score
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestTetraBug0038OptionalPointerInoutWritesBackCallerLocal(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func replace(slot: inout ptr?, value: ptr?) -> ptr?:
    slot = value
    return value

func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let memory_cap = core.cap_mem()
        let payload: ptr = core.alloc_bytes(8)
        let cell: ptr = core.ptr_add(payload, 4, memory_cap)
        let _stored: Int = core.store_i32(cell, 42, memory_cap)
        var maybe_base: ptr? = none
        let next_base: ptr? = payload
        let _returned: ptr? = replace(maybe_base, next_base)
        if let base = maybe_base:
            let loaded_cell: ptr = core.ptr_add(base, 4, memory_cap)
            let value: Int = core.load_i32(loaded_cell, memory_cap)
            if value == 42:
                return 0
            return value
        return 95
    return 99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestTetraBug0048FunctionTypedOptionalReturnLocalCall(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func some_int() -> Int?:
    return 42

func main() -> Int:
    let cb: fn() -> Int? = some_int
    let result: Int? = cb()
    if let value = result:
        return value
    return 0
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0049GenericInferenceUsesGenericStructFieldSelection(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
struct Box<T>:
    value: T

func id<T>(value: T) -> T:
    return value

func main() -> Int:
    let box: Box<Int> = Box<Int>(value: 42)
    let value: Int = id(box.value)
    if value == 42:
        return 0
    return value
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestTetraBug0040SelfHostTaskGroupDiagnostic(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	src := `
func worker() -> Int:
    return 42

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let value: Int = core.task_join_i32(task)
    let close_error: Int = core.task_group_close(group)
    if close_error == 0:
        return value
    return close_error
`
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "selfhost_task_group.tetra")
	outPath := filepath.Join(tmp, "app")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Runtime: RuntimeSelfHost})
	if err == nil {
		t.Fatalf("expected selfhost task-group diagnostic")
	}
	if !strings.Contains(err.Error(), "self-host runtime does not support task groups") {
		t.Fatalf("diagnostic = %v", err)
	}
}

func TestTetraBug0041IfLetPayloadBindingCanBeReusedAfterScopeExit(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func maybe(flag: Bool, value: Int) -> Int?:
    if flag:
        return value
    return none

func main() -> Int:
    let a: Int? = maybe(true, 20)
    if let value = a:
        let left: Int = value
    let b: Int? = maybe(true, 22)
    if let value = b:
        return value + 20
    return 99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0043FormatterPreservesPublicModifiers(t *testing.T) {
	src := `
module probes.pub_format_probe

pub import lib.core.capability as cap

pub struct Box:
    value: Int

pub enum Route:
    case ready(Int)

pub protocol Score:
    func score(route: Route) -> Int

pub extension Box:
    func score(self: Box) -> Int:
        return self.value

pub const answer: Int = 42

pub func score(route: Route) -> Int:
    return answer
`
	formatted, err := FormatSource([]byte(src), "pub_format_probe.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	for _, want := range []string{
		"pub import lib.core.capability as cap",
		"pub struct Box:",
		"pub enum Route:",
		"pub protocol Score:",
		"pub extension Box:",
		"pub const answer: Int = 42",
		"pub func score(route: Route) -> Int:",
	} {
		if !strings.Contains(string(formatted), want) {
			t.Fatalf("formatted source missing %q:\n%s", want, string(formatted))
		}
	}
}

func TestTetraBug0044FormatterPreservesSelectiveImports(t *testing.T) {
	src := `
module app.main

import lib.math.{add, Vec}

func main() -> Int:
    return add(40, 2)
`
	formatted, err := FormatSource([]byte(src), "selective_import_probe.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if !strings.Contains(string(formatted), "import lib.math.{add, Vec}") {
		t.Fatalf("formatted source lost selective import:\n%s", string(formatted))
	}
}

func TestTetraBug0045OptionalTaskGroupPayloadSpawnReturnsWorkerValue(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func worker() -> Int:
    return 42

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    if let handle = maybe:
        let task: task.i32 = core.task_spawn_group_i32(handle, "worker")
        let result: task.result_i32 = core.task_join_result_i32(task)
        let close_error: Int = core.task_group_close(handle)
        if result.error != 0:
            return 10 + result.error
        if close_error != 0:
            return 20 + close_error
        return result.value
    return 99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0045IfLetTaskSpawnRegistersRuntimeAndEntry(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func worker() -> Int:
    return 42

func main() -> Int
uses runtime:
    let maybe: Int? = 7
    if let value = maybe:
        let task: task.i32 = core.task_spawn_i32("worker")
        let result: task.result_i32 = core.task_join_result_i32(task)
        if result.error != 0:
            return 10 + result.error
        return result.value
    return 99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0045IfLetTypedTaskGroupSpawnEmitsWrapper(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
enum GroupErr:
    case stopped
    case code(Int)

func worker() -> Int throws GroupErr:
    return 42

func alias_group(group: task.group) -> task.group:
    return group

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let maybe: task.group? = group
    if let handle = maybe:
        let returned: task.group = alias_group(handle)
        let task = core.task_spawn_group_i32_typed<GroupErr>(returned, "worker")
        let value: Int = catch core.task_join_group_i32_typed<GroupErr>(task):
        case GroupErr.stopped:
            70
        case GroupErr.code(error_code):
            error_code
        let close_error: Int = core.task_group_close(returned)
        if close_error != 0:
            return 80 + close_error
        return value
    return 99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0046GenericIdentityPreservesTaskGroupResource(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
func worker() -> Int:
    return 42

func id<T>(value: T) -> T:
    return value

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let returned: task.group = id(group)
    let task: task.i32 = core.task_spawn_group_i32(returned, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let close_error: Int = core.task_group_close(returned)
    if result.error != 0:
        return 10 + result.error
    if close_error != 0:
        return 20 + close_error
    return result.value
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0047IslandParameterCanReturnAggregateConstructor(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
struct IslandBox:
    region: island

func wrap_region(region: island) -> IslandBox:
    return IslandBox(region: region)

func main() -> Int
uses alloc, islands, mem:
    unsafe:
        let region: island = core.island_new(128)
        let boxed: IslandBox = wrap_region(region)
        var bytes: []u8 = core.island_make_u8(boxed.region, 3)
        bytes[0] = 12
        bytes[1] = 13
        bytes[2] = 17
        let total: Int = bytes[0] + bytes[1] + bytes[2]
        free(boxed.region)
        return total
    return 99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0050MatchExpressionCollectsTypedTaskRuntimeSymbols(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	stdout, code := buildAndRun(t, `
enum Choice:
    case left
    case right

enum TaskErr:
    case stopped
    case code(Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.code(20)

func main() -> Int
uses runtime:
    let choice: Choice = Choice.left
    return match choice:
    case Choice.left:
        catch core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<TaskErr>("worker")):
        case TaskErr.stopped:
            70
        case TaskErr.code(code):
            code
    case Choice.right:
        99
`)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 20 {
		t.Fatalf("exit code = %d, want 20", code)
	}
}

func TestTetraBug0051FormatterPreservesNestedCatchInMatchArm(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	src := `
enum Choice:
    case left
    case right

enum TaskErr:
    case stopped
    case code(Int)

func worker() -> Int throws TaskErr:
    throw TaskErr.code(20)

func main() -> Int
uses runtime:
    let choice: Choice = Choice.left
    return match choice:
    case Choice.left:
        catch core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<TaskErr>("worker")):
        case TaskErr.stopped:
            70
        case TaskErr.code(code):
            code
    case Choice.right:
        99
`
	formatted, err := FormatSource([]byte(src), "match_catch_format.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := "    case Choice.left:\n        catch core.task_join_i32_typed<TaskErr>(core.task_spawn_i32_typed<TaskErr>(\"worker\")):\n        case TaskErr.stopped:\n            70\n        case TaskErr.code(code):\n            code\n    case Choice.right:"
	if !strings.Contains(string(formatted), want) {
		t.Fatalf("formatted source corrupted nested catch:\n%s", string(formatted))
	}
	stdout, code := buildAndRun(t, string(formatted))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 20 {
		t.Fatalf("exit code = %d, want 20", code)
	}
}

func TestTetraBug0052FormatterPreservesNestedMatchInCatchArm(t *testing.T) {
	requireTetraBugLinuxAMD64(t)

	src := `
enum Kind:
    case value(Int)
    case empty

enum Err:
    case nested(Kind)

func fail() -> Int throws Err:
    throw Err.nested(Kind.value(4))

func main() -> Int:
    return catch fail():
    case Err.nested(kind):
        match kind:
        case Kind.value(value):
            value + 38
        case Kind.empty:
            0
`
	formatted, err := FormatSource([]byte(src), "nested_match_in_catch_format.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := "    case Err.nested(kind):\n        match kind:\n        case Kind.value(value):\n            value + 38\n        case Kind.empty:\n            0"
	if !strings.Contains(string(formatted), want) {
		t.Fatalf("formatted source corrupted nested match:\n%s", string(formatted))
	}
	stdout, code := buildAndRun(t, string(formatted))
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if code != 42 {
		t.Fatalf("exit code = %d, want 42", code)
	}
}

func TestTetraBug0053AwaitedOptionalResourceLocalBuilds(t *testing.T) {
	requireTetraBugInterfaceOnlyBuild(t, `
async func maybe_task(handle: task.i32) -> task.i32?:
    let out: task.i32? = handle
    return out

async func relay_task(handle: task.i32) -> task.i32?:
    let out: task.i32? = await maybe_task(handle)
    return out

func main() -> Int:
    return 42
`)
}

func TestTetraBug0054AwaitedResourceAggregateLocalBuilds(t *testing.T) {
	requireTetraBugInterfaceOnlyBuild(t, `
struct TaskBox:
    handle: task.i32

async func box_task(handle: task.i32) -> TaskBox:
    return TaskBox(handle: handle)

async func local_task_box(handle: task.i32) -> TaskBox:
    let box: TaskBox = await box_task(handle)
    return box

func main() -> Int:
    return 42
`)
}

func TestTetraBug0055DirectAwaitedPointerReturnBuilds(t *testing.T) {
	requireTetraBugInterfaceOnlyBuild(t, `
async func derive(base: ptr, offset: Int, memory_cap: cap.mem) -> ptr
uses mem:
    unsafe:
        return core.ptr_add(base, offset, memory_cap)
    return base

async func relay(base: ptr, memory_cap: cap.mem) -> ptr
uses mem:
    return await derive(base, 4, memory_cap)

func main() -> Int:
    return 42
`)
}
