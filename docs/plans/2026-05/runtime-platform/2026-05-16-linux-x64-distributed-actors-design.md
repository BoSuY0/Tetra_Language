# Linux-x64 Distributed Actors Production Design

Status: approved design for implementation planning.

## Goal

Promote `actors.distributed-runtime` from transport-artifact-only evidence to a
real production runtime path for `linux-x64`. The production claim is scoped to
Linux x86-64 only. macOS, Windows, and 32-bit/x32 Linux remain out of scope
until they have matching implementation and host evidence.

## Observed Facts

- The live target registry contains `linux-x64`, `windows-x64`, `macos-x64`,
  `wasm32-wasi`, and `wasm32-web`. There is no current `linux-x86`,
  `linux-x32`, or `linux-386` target.
- `linux-x64` is the only native target runnable on the current Linux host.
- The current actor runtime is embedded as a native runtime object under
  `compiler/internal/actorsrt`. It is a single-process cooperative scheduler
  with fixed local actor/mailbox/task-group tables.
- Actor lowering currently emits calls such as `__tetra_actor_spawn`,
  `__tetra_actor_send`, `__tetra_actor_recv`, typed send/receive transaction
  symbols, and task group/cancel/join symbols.
- `actor` is currently represented as a one-slot integer handle. Existing local
  handles use small non-negative values.
- Runtime object validation requires explicit `__tetra_*` symbols and
  signature metadata. Missing runtime symbols are hard errors.
- `tools/cmd/validate-actor-transport` validates only
  `tetra.actors.transport.v1` envelope/trace/hash artifacts. It is not runtime
  execution evidence.
- The readiness validator now rejects transport-only evidence for
  `actors.distributed-runtime`; production promotion requires runtime/lowering,
  tests, docs, and a report artifact that exercises distributed runtime
  execution.

## Production Boundary

The Linux-x64 production surface is:

- local actor/task behavior remains compatible with the existing ABI;
- distributed behavior is available only when a program explicitly connects to
  the Linux-x64 actor network runtime;
- node identity is an integer node id in the range `1..127`;
- local actor handles remain `0..127`;
- remote actor handles are encoded one-slot handles with the high bit set:
  `0x80000000 | (node_id << 16) | actor_id`;
- loopback TCP is the supported transport for the first production target;
- remote spawn creates an actor on a connected remote node by entry id;
- `core.send`, `core.send_msg`, and typed actor send transactions route remote
  handles over the network mailbox and keep local behavior unchanged;
- `core.recv`, `core.recv_msg`, timed/nonblocking receive, and typed receive
  consume the current node's local mailbox after network frames have been
  pumped into it;
- node failure is surfaced as explicit status/error results on the new
  distributed runtime API and must not hang the scheduler;
- task group cancel/join semantics remain local for this wave, but distributed
  actor operations must not break existing task group cancel/join behavior.

Non-goals for this wave:

- macOS/Windows production distributed runtime;
- 32-bit/x32 Linux targets;
- TLS, public internet routing, service discovery, federation, or clustering;
- remote task execution or cross-node task handles;
- arbitrary reference/pointer payload transfer across nodes;
- replacing the existing local actor ABI.

## Architecture

### Runtime Components

1. **Linux-x64 runtime client**
   - Extends the built-in Linux-x64 actor runtime object.
   - Uses Linux syscalls directly for `socket`, `connect`, `read`, `write`,
     `poll`, and `close`.
   - Maintains a broker socket, local node id, request sequence, pending remote
     spawn response state, and small fixed-size network frame buffer in the
     scheduler state.
   - Pumps network frames in scheduler yield/block/no-ready paths so remote
     messages can wake blocked receivers.

2. **Actor network broker**
   - Runs as a real CLI runtime component on loopback TCP.
   - Accepts node client connections, validates the wire protocol, routes
     spawn requests and actor message frames by node id, records failures, and
     can write an execution report.
   - This is a production runtime component, not a validator mock. Runtime
     smoke evidence must launch the broker and compiled Linux-x64 node
     executables.

3. **Compiler/lowering surface**
   - Adds explicit distributed actor builtins for connecting to the broker,
     checking node status, and remote spawn.
   - Keeps existing local actor lowering unchanged.
   - Routes existing send builtins through the runtime, where local vs remote
     handle dispatch is decided.

### Proposed Builtins

Names may be adjusted during implementation if existing naming conventions
require it, but the surface should stay this small:

- `core.actor_node_connect(node_id: Int, port: Int) -> Int`
  - Connects the current Linux-x64 runtime to the loopback broker on
    `127.0.0.1:<port>`.
  - Returns `0` on success and a stable non-zero error code on failure.
  - Requires `uses actors, runtime`.

- `core.spawn_remote(node_id: Int, name: str) -> actor`
  - Spawns the named actor entry on the remote node.
  - Returns a remote actor handle on success and `-1` on failure.
  - Requires `uses actors, runtime`.

- `core.actor_node_status(node_id: Int) -> Int`
  - Returns `0` for connected/healthy and stable non-zero values for unknown,
    disconnected, or failed nodes.
  - Requires `uses actors, runtime`.

Existing send and receive builtins stay source-compatible:

- `core.send(to, value)` routes remote handles through the network mailbox;
- `core.send_msg(to, value, tag)` routes tagged remote messages;
- `core.send_typed(to, msg)` routes enum payload slots using the existing
  staged actor message transaction ABI;
- receive builtins stay local to the current node mailbox.

### Wire Protocol

Use a small binary protocol shared by the broker and emitted runtime constants.
All frame fields are fixed-width little-endian integers.

Common frame header:

- magic: `TADR` (`0x52444154`);
- version: `1`;
- type;
- source node id;
- destination node id;
- sequence id;
- actor id;
- tag;
- slot count;
- payload slots, up to 8 `i32` values.

Frame types:

- `hello`: node registers with the broker;
- `hello_ack`: broker accepts or rejects the node;
- `spawn_req`: source asks destination node to spawn an entry id;
- `spawn_ack`: destination replies with actor id or error;
- `send_i32`: value message;
- `send_msg`: tagged message;
- `send_typed`: typed message transaction with tag and slots;
- `node_down`: broker reports a missing/disconnected destination;
- `error`: protocol or runtime error.

### Failure Model

- Connection failure returns a non-zero connect error; it is not treated as a
  successful distributed runtime.
- Sending to an unknown or disconnected remote node records node status as
  failed and must not hang.
- Remote spawn waits for a bounded response through the runtime pump and returns
  `-1` on failure.
- Broker protocol errors are surfaced in broker stderr/report and node status.
- Local actor/task behavior must remain deterministic when no distributed
  connection is active.

### Evidence Model

Production evidence must be executable:

1. Start the broker on loopback.
2. Build two Linux-x64 Tetra node executables.
3. Run node B connected to the broker and blocked in the runtime scheduler.
4. Run node A connected to the broker.
5. Node A `spawn_remote`s an actor on node B, sends a value/tagged/typed
   message, receives a reply through its local mailbox, and exits with the
   expected code.
6. A negative smoke closes or omits node B and proves node A reports failure
   instead of hanging.
7. The broker writes a runtime report under `reports/` that records actual
   connections, frame counts, node statuses, spawned remote actor ids, and
   process exit codes.
8. `validate-v0-4-readiness` accepts that report shape for
   `actors.distributed-runtime` and continues rejecting transport-only,
   fake, incomplete, or docs-only evidence.

## Risks

- Hand-written Linux-x64 syscall emission is high-risk. Keep the runtime client
  small and isolate the broker protocol in a Go package with table-driven
  tests before touching x64 emission.
- Blocking network calls can deadlock the cooperative scheduler. Runtime socket
  reads must be nonblocking or poll-bounded.
- Existing `core.send` return semantics cannot become the only failure channel.
  Node status and remote spawn error returns provide the stable failure API.
- Feature promotion must stay scoped. The manifest/docs must not imply
  macOS/Windows/32-bit support.

## Verification Strategy

- Unit tests for protocol encode/decode and broker routing over real loopback
  TCP.
- Semantics tests for the new builtins and their effect requirements.
- Lowering tests proving new distributed builtins lower to runtime symbols and
  existing local actor calls stay unchanged.
- Runtime object symbol/signature tests for Linux-x64 distributed symbols.
- Linux-x64 integration smoke that compiles and executes real Tetra node
  binaries against the broker.
- Negative smoke for missing/disconnected node.
- Readiness validator tests for accepting real Linux-x64 distributed runtime
  evidence and rejecting transport-only/fake/incomplete reports.
- Manifest/docs validators plus full relevant Go tests.
- `graphify update .` after code changes.
