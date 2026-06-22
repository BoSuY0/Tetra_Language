# Actors Runtime v1

Actors are an isolation + message-passing concurrency model built on top of Tetra’s existing
foundations:
Islands (region memory), and the explicit safe/unsafe boundary.

This document specifies the actor runtime and language surface included in the
v1 profile.

## Supported Targets

Actors are supported on x64 targets:
- `linux-x64`
- `macos-x64`
- `windows-x64`

**Build vs run:** the toolchain can always *build* these targets, but executing produced binaries is
only supported when
`host == target` (for example, `windows-x64` binaries are run only on Windows hosts).

The scoped production foundation claim below is narrower than the build matrix:
it is Linux-x64 actor/task runtime foundation evidence, not Windows/macOS
runtime production evidence and not a distributed actor target-parity claim.

## Goals

- Provide a simple concurrency story without GC or shared mutable state.
- Keep the user-facing API safe by default.
- Make the implementation small and auditable.

## Non-goals

- Multi-threaded scheduling.
- Broad zero-copy message passing of arbitrary region-backed data.
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

- `actor` — an opaque handle identifying an actor (current profile: small integer handle).

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
- pointer/resource/aggregate actor-state field types are rejected by the current actor-state field contract.

## Core Builtins

All actor builtins are **safe** (do not require `unsafe`), but functions that
call actor builtins or actor-using helpers must declare `uses actors`.

### `core.spawn(name: str) -> actor`

Spawns a new actor that executes the function named by `name`.

Current constraints:
- `name` must be a string literal known at compile time.
- The target function must exist and have the shape `func <name>() -> Int`.
- The target must be synchronous, non-throwing, and must not touch mutable
  global state.
- x64 targets are supported for v1.

### `core.send(to: actor, v: i32) -> i32`

Appends a message `(sender=self, value=v)` to `to`’s mailbox.

Returns `v` (current convenience).

### `core.recv() -> i32`

Receives a message from the current actor’s mailbox.

If the mailbox is empty, the actor **blocks** and yields to the scheduler until a message arrives.

### `core.sender() -> actor`

Returns the sender of the most recently received message in the current actor.

Valid only after a successful `core.recv()` (current profile: unspecified value otherwise).

### `core.self() -> actor`

Returns the handle of the current actor.

## Lifecycle Matrix

The current built-in x64 actor runtime has a narrow lifecycle model:

- `ready`: the actor is runnable.
- `blocked`: the actor is waiting in `core.recv()`, `core.recv_until(...)`, or
  `core.recv_msg_until(...)`.
- `sleeping`: the actor is waiting for a runtime timer.
- `waiting`: the actor is waiting on a task join.
- `done`: the actor function returned; the runtime does not schedule that
  actor again. Zero and nonzero actor entry returns become the same
  user-visible `done` state.

When the actor table is full, `core.spawn(...)` returns the raw invalid handle
value `-1`; later local sends to that handle return checked failure `-3`.
Once an actor reaches `done`, later local legacy, tagged, or typed sends to
that actor return checked failure `-4` before allocating a message node. A
message already queued in another actor's mailbox remains receivable after the
sender is done, and `core.sender()` for that receive may name a done actor.
Pending mailbox entries are drained into the runtime message-pool free list
when the actor reaches `done`; they are not delivered after completion. This is
a bounded local completion state, not a shutdown API. Other blocked, sleeping,
or waiting actors continue according to ordinary message, timer, and task-wait
readiness rules when one actor exits. In particular, nonzero actor entry
returns become the same user-visible `done` state as zero returns: later local
sends return `-4`, and there is no separate actor failure channel.

The raw core lifecycle ABI exposes `core.actor_status`,
`core.actor_status_raw`, `core.actor_wait`, `core.actor_wait_until`,
`core.actor_stop`, `core.actor_exit_reason`, `core.actor_link`,
`core.actor_unlink`, `core.actor_monitor`, `core.actor_demonitor`, and
`core.actor_set_trap_exit`. `core.actor_status_raw` returns the
runtime-owned `actor.status_result_raw` bridge with `status_code` and `result`
slots so `lib.core.actors.status` can distinguish `ok`, `invalid`, and
`stale` without treating invalid actor refs as enum values. User code should
prefer the stable
`lib.core.actors` wrappers: `ActorStatus`, `StatusResult`, `WaitResult`,
`StopResult`, `LinkResult`, `MonitorResult`, `status`, `wait`,
`wait_until`, `stop`, `link`, `unlink`, `monitor`, `demonitor`, and
`set_trap_exit`. Current Linux-x64 evidence covers local lifecycle status
observations, wait/wait-until done and timeout results, invalid/stale wait
refs mapping to public `dead` status plus `WaitResult.invalid` /
`WaitResult.stale` taxonomy, `StatusResult.invalid` and `StatusResult.stale`
for local actor status queries, invalid/stale preflight taxonomy for
`StopResult`, `LinkResult`, and `MonitorResult`, local stop requests, bounded
link/unlink propagation, monitor/demonitor cleanup, and trap-exit toggling.
This is not a supervision, restart, remote lifecycle, or OTP-style lifecycle
guarantee; several public result cases are reserved for later P06/P10
production semantics.

## Scheduling semantics

- Single OS thread.
- Cooperative: actors yield only when:
  - explicitly calling `core.yield()`,
  - blocked in `core.recv()`, `core.recv_until(deadline)`, or `core.recv_msg_until(deadline)`,
  - waiting in `core.task_join_i32()`, `core.task_join_result_i32()`,
    `core.task_join_until_i32(task, deadline)`, or `core.select2_i32(task, deadline)`,
  - sleeping in `core.sleep_ms()` or `core.sleep_until(deadline)`,
  - finished execution.
- Scheduler policy: round-robin over runnable actors (current profile).
- If no actor is ready but one or more actors have timed sleep, receive, or
  join deadlines, the runtime advances the logical clock to the nearest deadline
  and wakes due actors.
- Current executable evidence covers bounded cooperative progress for yielding
  runnable actors and deterministic deadline-order wake for sleeping actors.
  This is a single-thread cooperative fairness boundary, not preemptive
  scheduling or a production multi-threaded scheduler claim.
- `core.send()` wakes actors blocked in `core.recv()`; sleeping actors wake only
  through their deadline or task-group cancellation.
- Task-group cancellation has a scoped cooperative wake matrix:
  `core.recv_until(deadline)` returns
  `actor.recv_result_i32 { value: 0, error: 1 }`;
  `core.recv_msg_until(deadline)` returns
  `actor.recv_msg_result { value: 0, tag: 0, error: 1 }`;
  `core.task_join_result_i32(task)`,
  `core.task_join_until_i32(task, deadline)`, and
  `core.select2_i32(task, deadline)` return
  `task.result_i32 { value: 0, error: 1 }` when a task-group cancellation is
  observed while the caller is already waiting. Raw
  `core.task_join_i32(task)` has no error field, so its cancellation wake is
  exposed as raw value `0`; use the result or timed join APIs when code needs a
  checked cancellation status. Non-timed actor receives such as `core.recv()`
  and `core.recv_msg()` do not expose a cancellation result in the current
  profile; they remain message-oriented blocking APIs. This is not full actor
  supervision or a full structured-concurrency model for every blocking API.
- Successful `core.task_join_i32(task)`, `core.task_join_result_i32(task)`,
  `core.task_join_until_i32(task, deadline)`, and typed task joins consume the
  completed task result and then mark the target actor slot `reclaimable`.
  `core.task_poll_i32(task)` is non-consuming: it treats `done`/`reclaimable`
  task actors as terminal readiness but does not reclaim a completed task actor
  slot by itself. `core.task_group_close(group)` treats joined `reclaimable`
  task actors as terminal for group-close completion.

## Message Model

Current messages are `i32` values plus an implicit sender handle. Tagged messages
are available through `core.send_msg(to, value, tag)` and `core.recv_msg()`,
which returns `actor.msg { value, tag }`.
Timed receive is available through `core.recv_until(deadline)`, which returns
`actor.recv_result_i32 { value, error }` and uses error `2` for timeout.
Nonblocking receive is available through `core.recv_poll()` with the same
result shape and timeout code. Tagged timed receive is available through
`core.recv_msg_until(deadline)`, which returns
`actor.recv_msg_result { value, tag, error }`.

Typed actor messages are supported as an enum-only current profile:

- `core.send_typed(to, msg)` sends an enum message value to another actor and
  returns `i32`.
- `core.recv_typed<MessageEnum>()` receives the next typed enum message from
  the current actor mailbox.
- `send_typed` does not take explicit type arguments; `recv_typed` requires
  exactly one explicit enum type argument.
- The message type must be an enum. The enum payload is limited to value-only
  data that the checker can lower into actor message slots: current scalar
  payloads such as `Int`, `Bool`, and `UInt8`, nested structs/enums built only
  from supported value-only payloads, explicit owned copies of `String`/slice
  views, checked `island` transfer payloads, and the local P6.1 owned
  island-backed slice move form described below.
- Typed actor message payloads support at most 8 value slots. The enum tag is
  carried separately from those payload slots.
- Distributed typed actor frames carry `hash(enum type name) + case ordinal` as
  their wire tag. This keeps `send_typed` and `recv_typed<MessageEnum>()`
  aligned when messages cross the actor network boundary.
- Borrowed reference-shaped payloads such as borrowed `String` or slice views
  are rejected unless the payload expression explicitly uses `.copy()`. Raw
  pointer payloads, actor/task handles, and unrelated runtime handles are still
  rejected by the current value-only payload rule.
- `island` payload transfer is checked as a move and consumes the sent source
  (including nested struct/enum construction sources).
- P6.1 adds a narrow local zero-copy move for owned region-backed slices:
  a non-`.copy()` slice payload is accepted only when the checker knows it was
  created by `core.island_make_*` and the same typed message payload also
  carries the owning `island` value, for example
  `MoveMsg.region(region, xs)`. The sender loses access to both the island and
  the sent slice after `send_typed`; receiver-side pattern bindings own the
  received values.
- P6.2 adds typed mailbox metadata to
  `<output>.actor-transfer.json`. The report has `mailboxes[]` rows with the
  enum message schema, fixed local message capacity, explicit backpressure hook,
  max payload slots, slot width, and ownership metadata status. Its `sends[]`
  rows classify scalar payload copies, island moves, explicit view copies, and
  zero-copy island-backed slice moves with `ownership`, `transfer_mode`,
  `runtime_path`, `bytes_copied`, and `zero_copy` fields.
- Actor/task handle transfer in typed actor payloads is outside the current
  transfer contract and remains a stable rejection boundary under the same
  value-only payload rule.
- The race-safety rejection matrix is conservative: local immutable payloads
  copy, owned moved payloads consume sender access, borrowed views require an
  explicit `.copy()`, mutable global targets are rejected across actor/task
  boundaries, unsafe pointer payloads require an audited unsafe contract and
  remain rejected in safe typed actor payloads, and island region transfer is
  handled by the scoped actor/island proof track. This is not a formal
  concurrency proof and provides no lock/atomic shared-memory model.
- Linux-x64 parallel production evidence requires an `actor island boundary
  proof` case. That case exercises the Memory/Island handoff facts for
  actor/task/island boundaries and is validator-gated by `parallelprod`; it is
  not a request-island production claim.
- The same Linux-x64 parallel production report emits top-level
  `diagnostics[]` evidence for every negative actor/task case. Each row carries
  stable machine fields `case`, `code`, `severity`, `category`, `position`, and
  `expected_error`; the stable contract is the machine taxonomy and presence of
  scoped substring evidence, not frozen human diagnostic prose.
- Actor benchmark rows in the same report remain Tier 0/Tier 1 readiness
  evidence only. They make no benchmark superiority, no C++/Rust parity, and no official benchmark claim; higher-tier reproducibility requires a separate
  environment and artifact gate.
- Distributed typed actor frames still serialize fixed wire values. The P6.1
  zero-copy region-slice path is a local typed mailbox guarantee, not a
  cross-node pointer transfer guarantee.

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
  runnable actor. A completed actor remains `done` until `__tetra_actor_wait`
  or `__tetra_actor_wait_until` observes its result; only then does the runtime
  mark that slot reclaimable for future spawn reuse. Waited reclaimable actor
  slots reset stored initial stack frames instead of mapping a fresh stack for
  that slot; raw unobserved `done` slots are not reused. The public `actor`
  type is still opaque. A local legacy, tagged, or typed send to an invalid
  actor handle returns checked failure `-3` before allocating a message node.
- Builtin task join, timed join, and typed task join consume completed task
  results and then mark the target actor slot `reclaimable`, so successful
  sequential task lifetimes can exceed the 127 child actor concurrent cap. Task
  poll is non-consuming: it treats waited reclaimable target actor slots as
  terminal result states, matching `done` result reads, but does not reclaim a
  completed task actor slot by itself. Group close treats joined reclaimable
  task actors as terminal for completion. This is not a production mailbox
  payload-destructor ABI or payload-drop claim.
- Built-in x64 done actor send behavior: once a local actor has completed and
  its runtime status is `done`, later local legacy, tagged, or typed sends to
  that actor return checked failure `-4` before allocating a message node. The
  completed actor drains any pending mailbox nodes into the runtime message-pool
  free list before publishing `done`; those messages are not delivered. This is
  a bounded shutdown diagnostic. The public `lib.core.actors` lifecycle wrapper
  surface is present for status, wait, stop, link, unlink, monitor, demonitor,
  and trap-exit operations, with local runtime evidence for the current P01
  slice. It is not supervision, restart, remote lifecycle completion, OTP
  lifecycle behavior, or a general production shutdown framework.
- Built-in x64 per-actor mailbox depth: `maxActorMailboxMsgs = 256`. A local
  send to a full mailbox returns checked backpressure `-2` before allocating a
  message node. Receiving a message decrements the mailbox depth, so this
  backpressure is recoverable when the receiver drains messages. The same
  checked `-2` contract applies to local legacy, tagged, and typed sends; a
  failed typed send does not enqueue a partial typed payload. This is a bounded
  local mailbox policy, not a generic unbounded mailbox, automatic retry, or
  distributed delivery guarantee.
- Built-in x64 runtime message pool: 64 KiB, with a bump allocator plus a
  free list for drained message nodes. The current message node size is 96
  bytes because typed mailbox payload slots are stored as local 64-bit slots
  and the node carries a runtime-only system-message kind; this gives room for
  pointer-like local slice fields while keeping typed payloads capped at 8
  value slots by the checker. With the fixed 64 KiB pool, 682 single-slot live
  messages fit in the pool before reclamation. When a later local send would
  exceed the pool while those messages remain live, the built-in runtime
  returns checked failure `-1` and does not enqueue an overflow message.
  Drained message nodes are reclaimed and can be reused.
- Built-in x64 runtime actor state: 8 state slots per actor, each one `i32`
  storage cell. The checker enforces this limit for actor declarations and
  rejects programs that require more than 8 actor-state slots before lowering
  or runtime execution.
- The self-host actor runtime is a compatibility/smoke path for the current
  self-hosted ABI surface. It uses a smaller fixed actor/mailbox model and a
  different actor-state backing store, so the built-in x64 capacities above are
  the release evidence for capacity-sensitive behavior.

## Runtime sources

The canonical self-host runtime sources live under `__rt/actors_sysv.tetra`,
`__rt/actors_i386.tetra`, and `__rt/actors_win64.tetra`. The compiler embeds matching copies from
`compiler/selfhostrt/actors_sysv.tetra`, `compiler/selfhostrt/actors_i386.tetra`,
and `compiler/selfhostrt/actors_win64.tetra` when `--runtime=selfhost` is used,
or when `--runtime=auto` can use a self-host actor/task surface for a target
without a builtin runtime.

The canonical modules are `__rt.actors_sysv` for `linux-x64`/`macos-x64`/
`linux-x32`, `__rt.actors_i386` for `linux-x86`, and `__rt.actors_win64` for
`windows-x64`.

The older `actors_poc_*` files are retained as historical PoC snapshots and
compatibility references.

Future extensions:
- Copy-based passing of `[]u8` into a receiver-owned island.
- Distributed serialization contracts for owned-region payloads.
- Typed nonblocking/timed receive helpers if promoted by a future runtime
  profile.
- Generic actor mailbox APIs beyond the current enum-only message calls.
- Cancellation or structured-concurrency guarantees beyond the scoped
  cooperative task-group wake matrix above.
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

### Distributed Runtime Target Matrix

| Target | Distributed actor runtime status | Current evidence | Promotion requirement |
|---|---|---|---|
| `linux-x64` | current scoped | executable `tetra.actors.distributed-runtime.v1` smoke plus actor foundation gate | keep same-commit distributed smoke, artifact hashes, and foundation validator green |
| `macos-x64` | unsupported / nonclaim | no distributed actor symbols; actor net pump is no-op | add target runtime, smoke, validator, docs, and package gate before any support claim |
| `windows-x64` | unsupported / nonclaim | no distributed actor symbols; actor net pump is no-op | add target runtime, smoke, validator, docs, and package gate before any support claim |
| `wasm32-wasi` | unsupported / nonclaim | no distributed actor runtime gate | add target runtime, smoke, validator, docs, and package gate before any support claim |
| `wasm32-web` | unsupported / nonclaim | no distributed actor runtime gate | add target runtime, smoke, validator, docs, and package gate before any support claim |

Missing-node/node_down is status/failure evidence only. When the loopback
broker reports a missing destination with `node_down`, the Linux-x64 runtime
can surface that as `core.actor_node_status(...) == 1` after the network pump
observes the frame. This does not imply automatic retry, reconnect, restart,
supervision, or delivery retry; later delivery attempts are user-driven sends
against the same bounded distributed status surface.

Promotion evidence is executable, not report-only:
`scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh` builds a fresh
CLI, starts `tetra actor-net`, compiles Linux-x64 actor nodes, runs cross-node
send/receive and failure cases over TCP, writes
`tetra.actors.distributed-runtime.v1`, and validates it through
`tools/cmd/validate-distributed-actor-runtime`. The report records `git_head`,
`artifact_hashes`, and `frame_order`, and the release script writes and
validates `artifact-hashes.json` for the same report directory. Negative
evidence rejects transport-only, fake, stale-metadata, bad-frame-order,
incomplete, and docs-only reports.

Actor-runtime foundation leak/race evidence is scoped and release-gated by the
Linux-x64 actor slice: `go test -race` covers `actornet`, `actorsrt`,
`parallelrt`, and `actorsafety`, and parallel production evidence requires the
`actor broker leak cleanup` case. This is bounded cleanup/soak evidence, not an
exhaustive liveness proof.

## Actor Runtime Foundation Gate

Actor runtime foundation scoped release truth is
`tetra.actor.production_foundation.v1`, produced by
`scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh`.
The gate composes the Linux-x64 distributed actor runtime smoke, the
Linux-x64 parallel production smoke, focused actor/task tests, a race-enabled
actor slice, docs verification, artifact hash validation, and the
`tools/cmd/validate-actor-runtime-foundation` validator.

CI and package publishing must keep that gate in front of actor foundation
claims. `.github/workflows/ci.yml` runs `actor-runtime-foundation-linux` and
uploads `reports/actor-runtime-foundation/final/actor-runtime-foundation-manifest.json`,
`reports/actor-runtime-foundation/final/artifact-hashes.json`,
`distributed-actors-linux-x64/distributed-actors-linux-x64.json`, and
`parallel-production-linux-x64/parallel-production-linux-x64.json`.
`.github/workflows/release-packages.yml` runs the same gate before package
artifact upload, GitHub Release publishing, container publishing, and Homebrew
tap updates.

Foundation nonclaims are part of the contract:

- no full Erlang/OTP actor runtime claim;
- no cluster membership or reconnect/retry production claim;
- no non-Linux distributed actor runtime support claim;
- no distributed zero-copy pointer or region transfer claim;
- no formal race proof claim.

The current claim is deliberately platform-bounded. Non-Linux-x64 distributed
actor runtimes, multi-threaded actor scheduling, and broader structured
concurrency guarantees beyond the current cooperative task group handles require
separate promotion evidence. The distributed runtime report also carries
the exact foundation nonclaims above; validators reject positive claims for
those capabilities in this slice.

## Transport Evidence Contract

`tools/cmd/validate-actor-transport` validates the current
`tetra.actors.transport.v1` JSON evidence contract for future distributed actor
runtime work. The report records a single message envelope, a deterministic
`message_sha256`, source/destination node names, a transport label, and an
ordered trace that must contain a source `send` followed by a destination
`receive` for the same message id.

This validator is release-gate evidence for transport artifact shape and
integrity only. It is not a distributed runtime implementation, network mailbox
ABI, retry protocol, or ordering guarantee beyond the single recorded envelope.
It provides no membership protocol.

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
