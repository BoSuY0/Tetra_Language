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
- Arbitrary generic actor mailboxes or reference-carrying typed messages.
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
- each actor supports at most 8 state slots; the checker rejects actor
  declarations that exceed this budget before lowering or runtime execution;
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

Typed actor messages are supported as an enum-only MVP:

- `core.send_typed(to, msg)` sends an enum message value to another actor and
  returns `i32`.
- `core.recv_typed<MessageEnum>()` receives the next typed enum message from
  the current actor mailbox.
- `send_typed` does not take explicit type arguments; `recv_typed` requires
  exactly one explicit enum type argument.
- The message type must be an enum. The enum payload is limited to value-only
  data that the checker can lower into actor message slots: current scalar
  payloads such as `Int`, `Bool`, and `UInt8`, nested structs/enums built only
  from supported value-only payloads, and checked `island` transfer payloads.
- Typed actor message payloads support at most 8 value slots. The enum tag is
  carried separately from those payload slots.
- Distributed typed actor frames carry `hash(enum type name) + case ordinal` as
  their wire tag. This keeps `send_typed` and `recv_typed<MessageEnum>()`
  aligned when messages cross the actor network boundary.
- Reference-shaped payloads such as `String`, pointer payloads, and unrelated
  runtime handles are rejected by the current value-only payload rule.
- `island` payload transfer is the only checked ownership-transfer path in this
  typed actor payload MVP. Sending an `island` payload is validated as a move
  and consumes the sent source (including nested struct/enum construction
  sources).
- Actor/task handle transfer in typed actor payloads is outside the current
  transfer contract and remains a stable rejection boundary under the same
  value-only payload rule.

Typed actor messages currently provide blocking send/receive only. There is no
typed equivalent of `recv_poll`, `recv_until`, or `recv_msg_until`, and there is
no per-actor generic mailbox type beyond the enum message type used at each
`send_typed`/`recv_typed` call site.

## Runtime Capacity Limits

The current actor runtime has fixed local capacities. These are implementation
limits for the v1 local runtime, not distributed scheduling or resource
isolation guarantees.

- Built-in x64 runtime actor table: `maxActors = 128`, including the main
  actor. This leaves capacity for 127 child actors. When the table is full,
  `__tetra_actor_spawn` returns the raw handle value `-1` and does not create a
  runnable actor. The public `actor` type is still opaque; sending to an
  invalid handle is outside the current guarantee.
- Built-in x64 runtime message pool: 64 KiB, bump-allocated, with no message
  reclamation during a run. The current message node size is 56 bytes, so 1170
  single-slot messages fit in the pool. The same node shape is used for tagged
  and typed messages, with typed payloads still capped at 8 value slots by the
  checker. Message pool overflow is not a checked runtime error in the current
  built-in runtime; behavior after the bump pointer passes the pool is
  unspecified and must not be treated as a recoverable capacity signal.
- Built-in x64 runtime actor state: 8 state slots per actor, each one `i32`
  storage cell. The checker enforces this limit for actor declarations and
  rejects programs that require more than 8 actor-state slots before lowering
  or runtime execution.
- The self-host actor runtime is a compatibility/smoke path for the current
  self-hosted ABI surface. It uses a smaller fixed actor/mailbox model and a
  different actor-state backing store, so the built-in x64 capacities above are
  the release evidence for capacity-sensitive behavior.

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
- Typed nonblocking/timed receive helpers if promoted by a future runtime
  profile.
- Generic actor mailbox APIs beyond the current enum-only message calls.
- Cancellation or structured-concurrency guarantees beyond the current
  cooperative task group handles.
- Non-Linux-x64 distributed actor runtime targets, multi-threaded actor
  scheduling, and production broker deployments beyond the current loopback TCP
  smoke envelope.

## Distributed Runtime Promotion Surface

`actors.distributed-runtime` is current for Linux-x64. The supported production
slice uses the builtin Linux-x64 actor runtime plus `actornet` loopback TCP
broker to exercise distributed node identity, remote actor handles, network
mailbox send/receive for i32, tagged, and typed messages, missing-node
failure/status propagation, and compatibility with existing cooperative task
cancel/join handles.

Promotion evidence is executable, not report-only:
`scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh` builds a fresh
CLI, starts `tetra actor-net`, compiles Linux-x64 actor nodes, runs cross-node
send/receive and failure cases over TCP, writes
`tetra.actors.distributed-runtime.v1`, and validates it through
`tools/cmd/validate-distributed-actor-runtime`. Negative evidence rejects
transport-only, fake, incomplete, and docs-only reports.

The current claim is deliberately platform-bounded. Non-Linux-x64 distributed
actor runtimes, multi-threaded actor scheduling, and broader structured
concurrency guarantees beyond the current cooperative task group handles require
separate promotion evidence.

## Transport Evidence Contract

`tools/cmd/validate-actor-transport` validates the current
`tetra.actors.transport.v1` JSON evidence contract for future distributed actor
runtime work. The report records a single message envelope, a deterministic
`message_sha256`, source/destination node names, a transport label, and an
ordered trace that must contain a source `send` followed by a destination
`receive` for the same message id.

This validator is release-gate evidence for transport artifact shape and
integrity only. It is not a distributed runtime implementation, network mailbox
ABI, retry protocol, ordering guarantee beyond the single recorded envelope, or
cluster membership protocol.

## Runtime ABI surface (internal)

Actors are implemented by linking a runtime object that exports a small set of
reserved symbols (e.g. `__tetra_entry`, `__tetra_actor_*`). Tagged message
builtins are part of that ABI through `__tetra_actor_send_msg` and
`__tetra_actor_recv_msg`. Typed message builtins lower to the multi-slot actor
message transaction ABI (`__tetra_actor_send_begin`,
`__tetra_actor_send_slot`, `__tetra_actor_send_commit`,
`__tetra_actor_recv_begin`, `__tetra_actor_recv_slot`, and
`__tetra_actor_recv_count`). The exact symbol list and calling conventions are
documented in `docs/spec/runtime_abi.md`.
