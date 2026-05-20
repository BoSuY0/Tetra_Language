package compiler_test

import "testing"

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitReturn(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async.t4": `module lib.ownership_async

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x
`,
		"app/main.t4": `module app.main
import lib.ownership_async as ownership

async func caller(x: borrow ptr) -> ptr throws ownership.AsyncErr:
    return try await ownership.producer(x)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleAwaitReturn(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async.t4": `module lib.ownership_async

pub async func producer(x: borrow ptr) -> ptr:
    return x
`,
		"app/main.t4": `module app.main
import lib.ownership_async as ownership

async func caller(x: borrow ptr) -> ptr:
    return await ownership.producer(x)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleTryAwaitGlobalAssign(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async.t4": `module lib.ownership_async

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x
`,
		"app/main.t4": `module app.main
import lib.ownership_async as ownership

var leaked: ptr = 0

async func caller(x: borrow ptr) -> Int throws ownership.AsyncErr:
    leaked = try await ownership.producer(x)
    return 0

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayTryAwaitReturn(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_base.t4": `module lib.ownership_async_base

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x
`,
		"lib/ownership_async_relay.t4": `module lib.ownership_async_relay
import lib.ownership_async_base as base

pub async func relay(x: borrow ptr) -> ptr throws base.AsyncErr:
    return try await base.producer(x)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_base as base
import lib.ownership_async_relay as relay

async func caller(x: borrow ptr) -> ptr throws base.AsyncErr:
    return try await relay.relay(x)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayAwaitReturn(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_base.t4": `module lib.ownership_async_base

pub async func producer(x: borrow ptr) -> ptr:
    return x
`,
		"lib/ownership_async_relay.t4": `module lib.ownership_async_relay
import lib.ownership_async_base as base

pub async func relay(x: borrow ptr) -> ptr:
    return await base.producer(x)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_base as base
import lib.ownership_async_relay as relay

async func caller(x: borrow ptr) -> ptr:
    return await relay.relay(x)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayTryAwaitGlobalAssign(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_base.t4": `module lib.ownership_async_base

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> ptr throws AsyncErr:
    return x
`,
		"lib/ownership_async_relay.t4": `module lib.ownership_async_relay
import lib.ownership_async_base as base

pub async func relay(x: borrow ptr) -> ptr throws base.AsyncErr:
    return try await base.producer(x)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_base as base
import lib.ownership_async_relay as relay

var leaked: ptr = 0

async func caller(x: borrow ptr) -> Int throws base.AsyncErr:
    leaked = try await relay.relay(x)
    return 0

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayAwaitOptionalReturn(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_base.t4": `module lib.ownership_async_base

pub struct Holder:
    value: ptr

pub async func producer(x: borrow ptr) -> Holder?:
    return Holder { value: x }
`,
		"lib/ownership_async_relay.t4": `module lib.ownership_async_relay
import lib.ownership_async_base as base

pub async func relay(x: borrow ptr) -> base.Holder?:
    return await base.producer(x)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_base as base
import lib.ownership_async_relay as relay

async func caller(x: borrow ptr) -> base.Holder?:
    return await relay.relay(x)

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayTryAwaitOptionalGlobalAssign(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_base.t4": `module lib.ownership_async_base

pub struct Holder:
    value: ptr

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> Holder? throws AsyncErr:
    return Holder { value: x }
`,
		"lib/ownership_async_relay.t4": `module lib.ownership_async_relay
import lib.ownership_async_base as base

pub async func relay(x: borrow ptr) -> base.Holder? throws base.AsyncErr:
    return try await base.producer(x)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_base as base
import lib.ownership_async_relay as relay

var leaked: base.Holder? = none

async func caller(x: borrow ptr) -> Int throws base.AsyncErr:
    leaked = try await relay.relay(x)
    return 0

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayMatchOptionalReturn(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_base.t4": `module lib.ownership_async_base

pub struct Holder:
    value: ptr

pub async func producer(x: borrow ptr) -> Holder?:
    return Holder { value: x }
`,
		"lib/ownership_async_relay.t4": `module lib.ownership_async_relay
import lib.ownership_async_base as base

pub async func relay(x: borrow ptr) -> base.Holder?:
    return await base.producer(x)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_base as base
import lib.ownership_async_relay as relay

async func caller(x: borrow ptr) -> ptr:
    match await relay.relay(x):
    case some(value):
        return value.value
    case none:
        return 0

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}

func TestAsyncRejectBorrowedEscapeViaCrossModuleRelayTryAwaitMatchOptionalGlobalAssign(t *testing.T) {
	files := map[string]string{
		"lib/ownership_async_base.t4": `module lib.ownership_async_base

pub struct Holder:
    value: ptr

pub enum AsyncErr:
    case failed

pub async func producer(x: borrow ptr) -> Holder? throws AsyncErr:
    return Holder { value: x }
`,
		"lib/ownership_async_relay.t4": `module lib.ownership_async_relay
import lib.ownership_async_base as base

pub async func relay(x: borrow ptr) -> base.Holder? throws base.AsyncErr:
    return try await base.producer(x)
`,
		"app/main.t4": `module app.main
import lib.ownership_async_base as base
import lib.ownership_async_relay as relay

var leaked: ptr = 0

async func caller(x: borrow ptr) -> Int throws base.AsyncErr:
    match try await relay.relay(x):
    case some(value):
        leaked = value.value
        return 0
    case none:
        return 0

func main() -> Int:
    return 0
`,
	}

	requireCheckWorldFilesErrorContains(t, files, "app/main.t4", "borrowed local 'x' cannot escape via return")
}
