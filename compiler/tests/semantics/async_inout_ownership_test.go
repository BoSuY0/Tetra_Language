package compiler_test

import "testing"

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_inout_base.t4": `module lib.ownership_async_inout_base

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x
`,
		"app/main.t4": `module app.main
import lib.ownership_async_inout_base as base

async func caller(x: borrow ptr, out: inout ptr) -> Int throws base.AsyncErr:
    out = try await base.producer(x)
    return 0

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_inout_base_await.t4": `module lib.ownership_async_inout_base_await

pub async func producer(x: borrow ptr) -> ptr:
    return x
`,
		"app/main.t4": `module app.main
import lib.ownership_async_inout_base_await as base

async func caller(x: borrow ptr, out: inout ptr) -> Int:
    out = await base.producer(x)
    return 0

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_base_await.t4": `module lib.ownership_async_callback_base_await

pub async func producer(x: borrow ptr) -> ptr:
    return x

pub async func relay(x: borrow ptr, cb: fn(inout ptr) -> Int) -> Int:
    let value: ptr = await producer(x)
    return cb(value)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_base_await as base

async func caller(x: borrow ptr, cb: fn(inout ptr) -> Int) -> Int:
    return await base.relay(x, cb)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayAwaitEnumPayloadCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_relay_base_await.t4": `module lib.ownership_async_relay_base_await

pub enum Holder:
    case some(fn(inout ptr) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> ptr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int:
    let value: ptr = await producer(x)
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_base_await as base

async func caller(x: borrow ptr, h: base.Holder) -> Int:
    return await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayTryAwaitEnumPayloadCallbackInoutWithThrows(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_relay_base_throws_await.t4": `module lib.ownership_async_relay_base_throws_await

pub enum AsyncErr:
    case failed

pub enum Holder:
    case some(fn(inout ptr) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let value: ptr = try await producer(x)
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_base_throws_await as base

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayTryAwaitOptionalEnumPayloadCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_relay_optional_payload_await.t4": `module lib.ownership_async_relay_optional_payload_await

pub enum AsyncErr:
    case failed

pub struct Box:
    value: ptr?

pub enum Holder:
    case some(fn(inout ptr?) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> Box throws AsyncErr:
    return Box { value: x }

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let produced: Box = try await producer(x)
    let value: ptr? = produced.value
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_optional_payload_await as base

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitEnumPayloadCallbackInoutChain(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_relay_chain_base.t4": `module lib.ownership_async_relay_chain_base

pub enum AsyncErr:
    case failed

pub enum Holder:
    case some(fn(inout ptr) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let value: ptr = try await producer(x)
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"lib/ownership_async_relay_chain_mid.t4": `module lib.ownership_async_relay_chain_mid
import lib.ownership_async_relay_chain_base as base

pub async func relay(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_chain_base as base
import lib.ownership_async_relay_chain_mid as mid

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await mid.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitEnumPayloadCallbackInoutChain(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_relay_chain_base_no_throw.t4": `module lib.ownership_async_relay_chain_base_no_throw

pub enum Holder:
    case some(fn(inout ptr) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> ptr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int:
    let value: ptr = await producer(x)
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"lib/ownership_async_relay_chain_mid_no_throw.t4": `module lib.ownership_async_relay_chain_mid_no_throw
import lib.ownership_async_relay_chain_base_no_throw as base

pub async func relay(x: borrow ptr, h: base.Holder) -> Int:
    return await base.relay(x, h)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_chain_base_no_throw as base
import lib.ownership_async_relay_chain_mid_no_throw as mid

async func caller(x: borrow ptr, h: base.Holder) -> Int:
    return await mid.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitOptionalEnumPayloadCallbackInoutChain(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_relay_chain_optional_base.t4": `module lib.ownership_async_relay_chain_optional_base

pub enum AsyncErr:
    case failed

pub struct Box:
    value: ptr?

pub enum Holder:
    case some(fn(inout ptr?) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> Box throws AsyncErr:
    return Box { value: x }

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let produced: Box = try await producer(x)
    let value: ptr? = produced.value
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"lib/ownership_async_relay_chain_optional_mid.t4": `module lib.ownership_async_relay_chain_optional_mid
import lib.ownership_async_relay_chain_optional_base as base

pub async func relay(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_chain_optional_base as base
import lib.ownership_async_relay_chain_optional_mid as mid

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await mid.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitStructFieldCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_struct_await.t4": `module lib.ownership_async_callback_struct_await

pub struct Holder:
    cb: fn(inout ptr) -> Int

pub async func producer(x: borrow ptr) -> ptr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int:
    let value: ptr = await producer(x)
    return h.cb(value)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_struct_await as base

async func caller(x: borrow ptr, h: base.Holder) -> Int:
    return await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitOptionalStructFieldCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_struct_optional_await.t4": `module lib.ownership_async_callback_struct_optional_await

pub struct Box:
    value: ptr?

pub struct Holder:
    cb: fn(inout ptr) -> Int

pub async func producer(x: borrow ptr) -> Box:
    return Box { value: x }

pub async func relay(x: borrow ptr, h: Holder) -> Int:
    let produced: Box = await producer(x)
    let value: ptr? = produced.value
    match value:
    case some(raw):
        return h.cb(raw)
    case none:
        return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_struct_optional_await as base

async func caller(x: borrow ptr, h: base.Holder) -> Int:
    return await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitEnumPayloadCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_enum_await.t4": `module lib.ownership_async_callback_enum_await

pub enum Holder:
    case some(fn(inout ptr) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> ptr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int:
    let value: ptr = await producer(x)
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_enum_await as base

async func caller(x: borrow ptr, h: base.Holder) -> Int:
    return await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitOptionalEnumPayloadCallbackInoutChainNoThrow(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_relay_chain_optional_base_no_throw.t4": `module lib.ownership_async_relay_chain_optional_base_no_throw

pub struct Box:
    value: ptr?

pub enum Holder:
    case some(fn(inout ptr?) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> Box:
    return Box { value: x }

pub async func relay(x: borrow ptr, h: Holder) -> Int:
    let produced: Box = await producer(x)
    let value: ptr? = produced.value
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"lib/ownership_async_relay_chain_optional_mid_no_throw.t4": `module lib.ownership_async_relay_chain_optional_mid_no_throw
import lib.ownership_async_relay_chain_optional_base_no_throw as base

pub async func relay(x: borrow ptr, h: base.Holder) -> Int:
    return await base.relay(x, h)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_chain_optional_base_no_throw as base
import lib.ownership_async_relay_chain_optional_mid_no_throw as mid

async func caller(x: borrow ptr, h: base.Holder) -> Int:
    return await mid.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitStructFieldCallbackInoutChain(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_struct_chain_base.t4": `module lib.ownership_async_callback_struct_chain_base

pub struct Holder:
    cb: fn(inout ptr) -> Int

pub async func producer(x: borrow ptr) -> ptr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int:
    let value: ptr = await producer(x)
    return h.cb(value)
`,
		"lib/ownership_async_callback_struct_chain_mid.t4": `module lib.ownership_async_callback_struct_chain_mid
import lib.ownership_async_callback_struct_chain_base as base

pub async func relay(x: borrow ptr, h: base.Holder) -> Int:
    return await base.relay(x, h)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_struct_chain_base as base
import lib.ownership_async_callback_struct_chain_mid as mid

async func caller(x: borrow ptr, h: base.Holder) -> Int:
    return await mid.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayTryAwaitInoutAssignment(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_inout_base.t4": `module lib.ownership_async_inout_base

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x
`,
		"lib/ownership_async_inout_relay.t4": `module lib.ownership_async_inout_relay
import lib.ownership_async_inout_base as base

pub async func relay(x: borrow ptr, out: inout ptr) -> Int throws base.AsyncErr:
    out = try await base.producer(x)
    return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_inout_base as base
import lib.ownership_async_inout_relay as relay

async func caller(x: borrow ptr, out: inout ptr) -> Int throws base.AsyncErr:
    return try await relay.relay(x, out)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitStructFieldCallbackInoutChain(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_relay_struct_chain_base.t4": `module lib.ownership_async_relay_struct_chain_base

pub enum AsyncErr:
    case failed

pub struct Holder:
    cb: fn(inout ptr) -> Int

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let value: ptr = try await producer(x)
    return h.cb(value)
`,
		"lib/ownership_async_relay_struct_chain_mid.t4": `module lib.ownership_async_relay_struct_chain_mid
import lib.ownership_async_relay_struct_chain_base as base

pub async func relay(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_relay_struct_chain_base as base
import lib.ownership_async_relay_struct_chain_mid as mid

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await mid.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_base.t4": `module lib.ownership_async_callback_base

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

pub async func relay(x: borrow ptr, cb: fn(inout ptr) -> Int) -> Int throws AsyncErr:
    let value: ptr = try await producer(x)
    return cb(value)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_base as base

async func caller(x: borrow ptr, cb: fn(inout ptr) -> Int) -> Int throws base.AsyncErr:
    return try await base.relay(x, cb)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitStructFieldCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_struct.t4": `module lib.ownership_async_callback_struct

pub enum AsyncErr:
    case failed

pub struct Holder:
    cb: fn(inout ptr) -> Int

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let value: ptr = try await producer(x)
    return h.cb(value)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_struct as base

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitOptionalStructFieldCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_struct_optional.t4": `module lib.ownership_async_callback_struct_optional

pub enum AsyncErr:
    case failed

pub struct Holder:
    cb: fn(inout ptr) -> Int

pub struct Box:
    value: ptr?

pub async func producer(x: borrow ptr) -> Box throws AsyncErr:
    return Box { value: x }

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let value: Box = try await producer(x)
    match value.value:
    case some(raw):
        return h.cb(raw)
    case none:
        return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_struct_optional as base

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitEnumPayloadCallbackInout(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_callback_enum.t4": `module lib.ownership_async_callback_enum

pub enum AsyncErr:
    case failed

pub enum Holder:
    case some(fn(inout ptr) -> Int)
    case empty

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x

pub async func relay(x: borrow ptr, h: Holder) -> Int throws AsyncErr:
    let value: ptr = try await producer(x)
    match h:
    case Holder.some(cb):
        return cb(value)
    case Holder.empty:
        return 0
`,
		"app/main.t4": `module app.main
import lib.ownership_async_callback_enum as base

async func caller(x: borrow ptr, h: base.Holder) -> Int throws base.AsyncErr:
    return try await base.relay(x, h)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}
