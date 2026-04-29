# Actors Runtime v1

Actors are an isolation + message-passing concurrency model built on top of Tetra’s existing foundations:
Islands (region memory), and the explicit safe/unsafe boundary.

This document specifies the actor runtime and language surface included in the
v1 profile.

## Supported targets (MVP)

Actors are supported on x64 targets:
- `linux-x64`
- `macos-x64`
- `windows-x64`

**Build vs run:** the toolchain can always *build* these targets, but executing produced binaries is only supported when
`host == target` (for example, `windows-x64` binaries are run only on Windows hosts).

## Goals

- Provide a simple concurrency story without GC or shared mutable state.
- Keep the user-facing API safe by default.
- Make the implementation small and auditable.

## Non-goals (MVP)

- Multi-threaded scheduling.
- Zero-copy message passing of region-backed data.
- Generic/typed messages beyond `i32`.
- Shared mutable actor state across OS threads.

## Model

- An **actor** is an isolated unit of execution with a **mailbox** (FIFO queue).
- Actors run under a **single-thread cooperative scheduler**.
- An actor can:
  - spawn new actors,
  - send messages,
  - receive messages (blocking, but implemented cooperatively).

## Types

- `actor` — an opaque handle identifying an actor (MVP: small integer handle).

## Actor Declarations (Current Subset)

Actor declarations are supported in the current language surface:

```tetra
actor Counter {
    var count: Int = 0
    val step: UInt8 = 1
    const boost: UInt16 = 2
    var err: task.error = 0

    func run() -> Int {
        count = count + step
        return count + boost + err
    }
}
```

Current actor-state field constraints:

- field declarations may use `var`, `val`, or `const`;
- supported scalar field types are `Int`, `Bool`, `UInt8`, `UInt16`, and
  `task.error`;
- field initializers are required and must be compile-time constants;
- pointer/resource/aggregate actor-state field types are rejected in this MVP.

## Core builtins (MVP)

All actor builtins are **safe** (do not require `unsafe`), but functions that
call actor builtins or actor-using helpers must declare `uses actors`.

### `core.spawn(name: str) -> actor`

Spawns a new actor that executes the function named by `name`.

MVP constraints:
- `name` must be a string literal known at compile time.
- The target function must exist and have the shape `func <name>() -> Int`.
- The target must be synchronous, non-throwing, and must not touch mutable
  global state.
- x64 targets are supported for v1.

### `core.send(to: actor, v: i32) -> i32`

Appends a message `(sender=self, value=v)` to `to`’s mailbox.

Returns `v` (MVP convenience).

### `core.recv() -> i32`

Receives a message from the current actor’s mailbox.

If the mailbox is empty, the actor **blocks** and yields to the scheduler until a message arrives.

### `core.sender() -> actor`

Returns the sender of the most recently received message in the current actor.

Valid only after a successful `core.recv()` (MVP: unspecified value otherwise).

### `core.self() -> actor`

Returns the handle of the current actor.

## Scheduling semantics

- Single OS thread.
- Cooperative: actors yield only when:
  - explicitly calling `core.yield()`,
  - blocked in `core.recv()`, `core.recv_until(deadline)`, or `core.recv_msg_until(deadline)`,
  - waiting in `core.task_join_i32()`, `core.task_join_result_i32()`, `core.task_join_until_i32(task, deadline)`, or `core.select2_i32(task, deadline)`,
  - sleeping in `core.sleep_ms()` or `core.sleep_until(deadline)`,
  - finished execution.
- Scheduler policy: round-robin over runnable actors (MVP).
- If no actor is ready but one or more actors have timed sleep, receive, or
  join deadlines, the runtime advances the logical clock to the nearest deadline
  and wakes due actors.
- `core.send()` wakes actors blocked in `core.recv()`; sleeping actors wake only
  through their deadline or task-group cancellation.

## Message Model

MVP messages are `i32` values plus an implicit sender handle. Tagged messages
are available through `core.send_msg(to, value, tag)` and `core.recv_msg()`,
which returns `actor.msg { value, tag }`.
Timed receive is available through `core.recv_until(deadline)`, which returns
`actor.recv_result_i32 { value, error }` and uses error `2` for timeout.
Nonblocking receive is available through `core.recv_poll()` with the same
result shape and timeout code. Tagged timed receive is available through
`core.recv_msg_until(deadline)`, which returns
`actor.recv_msg_result { value, tag, error }`.

## Runtime sources

The canonical self-host runtime sources live under `__rt/actors_sysv.tetra` and
`__rt/actors_win64.tetra`. The compiler embeds matching copies from
`compiler/selfhostrt/actors_sysv.tetra` and `compiler/selfhostrt/actors_win64.tetra`
when `--runtime=selfhost` is used, or when `--runtime=auto` can use the
self-host mailbox-only actor surface (no actor-state/task/time builtins).

The canonical modules are `__rt.actors_sysv` for `linux-x64`/`macos-x64` and
`__rt.actors_win64` for `windows-x64`.

The older `actors_poc_*` files are retained as historical PoC snapshots and
compatibility references.

Future extensions (post-MVP):
- Copy-based passing of `[]u8` into a receiver-owned island.
- Ownership transfer of message islands (move/consume semantics).
- Distributed actors and network mailboxes.
- Cancellation or structured-concurrency guarantees beyond the current
  cooperative task group handles.

## Runtime ABI surface (internal)

Actors are implemented by linking a runtime object that exports a small set of
reserved symbols (e.g. `__tetra_entry`, `__tetra_actor_*`). Tagged message
builtins are part of that ABI through `__tetra_actor_send_msg` and
`__tetra_actor_recv_msg`. The exact symbol list and calling conventions are
documented in `docs/spec/runtime_abi.md`.
