# Linux-x64 Distributed Actors Implementation Plan

**Goal:** Implement real production `actors.distributed-runtime` behavior for
`linux-x64` only, with executable loopback/TCP runtime evidence and no
transport-only proxy promotion.

**Context:** The design is documented in
`docs/plans/2026-05-16-linux-x64-distributed-actors-design.md`. The current
runtime is a local single-process actor scheduler under
`compiler/internal/actorsrt`; this plan extends Linux-x64 while keeping the
local actor/task ABI compatible.

**Execution:** Use test-driven implementation slices. Do not promote the feature
registry until runtime execution evidence and readiness validation pass.

## Task 1: Baseline And Guard Rails

**Goal:** Record current behavior and add regression guards around the existing
local actor ABI before adding distributed symbols.

**Files:**

- Inspect/modify `compiler/actors_test.go`
- Inspect/modify `compiler/task_runtime_test.go`
- Inspect/modify `compiler/runtime_override_test.go`
- Inspect/modify `compiler/internal/actorsrt/actor_state_symbols_test.go`

**Approach:**

- Add focused tests that assert existing actor runtime symbols and local
  ping-pong/task group behavior remain unchanged.
- Add a guard that `linux-x64` is the only target promoted by this wave.

**Verification:**

```sh
go test ./compiler -run 'Actors|TaskGroup|RuntimeObject' -count=1
go test ./compiler/internal/actorsrt -count=1
```

**Done when:** Current local actor/task runtime behavior is protected by tests
before distributed changes begin.

## Task 2: Wire Protocol Package

**Goal:** Add a shared wire protocol implementation for real broker/runtime
frames.

**Files:**

- Add `compiler/actorwire/`
- Add protocol tests in `compiler/actorwire`

**Approach:**

- Define frame constants, max payload slots, stable error codes, and
  encode/decode helpers.
- Use fixed-size little-endian binary frames.
- Reject wrong magic/version, invalid node ids, invalid slot counts, and
  truncated frames.

**Verification:**

```sh
go test ./compiler/actorwire -count=1
```

**Done when:** The broker and runtime can share tested protocol constants and
frame encoding semantics.

## Task 3: Loopback Broker Runtime Component

**Goal:** Implement a real TCP broker that routes actor network frames between
Linux-x64 node processes.

**Files:**

- Add `cli/internal/actornet/`
- Add `cli/cmd/tetra/actor_net.go`
- Modify `cli/cmd/tetra/main.go`
- Add broker/CLI tests under `cli/cmd/tetra` or `cli/internal/actornet`

**Approach:**

- Add a CLI command, tentatively `tetra actor-net`, that listens on
  `127.0.0.1:<port>`.
- Accept real TCP node connections.
- Route `hello`, `spawn_req`, `spawn_ack`, `send_i32`, `send_msg`,
  `send_typed`, `node_down`, and `error` frames.
- Write an optional JSON runtime report with connected nodes, frame counts,
  remote spawns, message counts, node failures, and broker status.

**Verification:**

```sh
go test ./cli/internal/actornet -count=1
go test ./cli/cmd/tetra -run 'ActorNet|Help' -count=1
```

**Done when:** Real loopback clients can exchange frames through the broker in
tests, and malformed/fake frames are rejected.

## Task 4: Semantics Surface For Distributed Actor Builtins

**Goal:** Add type/effect checking for explicit distributed actor builtins.

**Files:**

- Modify `compiler/internal/semantics/builtins.go`
- Modify `compiler/internal/semantics/exprs.go` if builtin-specific checks are
  needed
- Add tests in `compiler/tests/semantics/async_test.go` or a focused actor
  semantics test file
- Add CLI diagnostic tests if errors need stable JSON codes

**Approach:**

- Add:
  - `core.actor_node_connect(node_id: Int, port: Int) -> Int`
  - `core.spawn_remote(node_id: Int, name: str) -> actor`
  - `core.actor_node_status(node_id: Int) -> Int`
- Require `uses actors, runtime`.
- Keep remote payload transfer value-only through the existing typed actor
  message validator.
- Reject non-literal remote spawn names if the lowering still needs entry ids.

**Verification:**

```sh
go test ./compiler/tests/semantics -run 'Actor|Distributed|Runtime' -count=1
go test ./cli/cmd/tetra -run 'Actor|Distributed|Diagnostic' -count=1
```

**Done when:** New builtins are checked, documented in the builtin manifest, and
negative diagnostics are stable.

## Task 5: Lowering And Runtime Symbol Contract

**Goal:** Lower distributed actor builtins to Linux-x64 runtime symbols and
extend runtime validation.

**Files:**

- Modify `compiler/internal/lower/lower.go`
- Modify `compiler/compiler.go`
- Modify `compiler/manifest.go`
- Add/modify lower and runtime object tests

**Approach:**

- Lower `actor_node_connect` to
  `__tetra_actor_node_connect(node_id, port)`.
- Lower `spawn_remote` string literals to an entry id and call
  `__tetra_actor_spawn_remote(node_id, entry_id)`.
- Lower `actor_node_status` to `__tetra_actor_node_status(node_id)`.
- Add distributed actor runtime symbols to a Linux-x64-specific required symbol
  set, not to macOS/Windows production claims.
- Keep existing local runtime symbol validation unchanged for programs without
  distributed builtins.

**Verification:**

```sh
go test ./compiler/internal/lower -run 'Actor|Distributed' -count=1
go test ./compiler -run 'RuntimeObject|Actor|Distributed' -count=1
go test ./tools/cmd/validate-manifest -count=1
```

**Done when:** IR and runtime object validation prove the compiler uses a real
distributed runtime symbol path.

## Task 6: Linux-x64 Runtime Client

**Goal:** Implement real Linux-x64 runtime networking over loopback TCP.

**Files:**

- Modify `compiler/internal/actorsrt/linux_x64.go`
- Modify `compiler/internal/actorsrt/linux_x64_emit.go`
- Modify `compiler/internal/backend/x64/emitter.go` only for missing generic
  instruction helpers required by the runtime emitter
- Add tests in `compiler/internal/actorsrt` and `compiler/actors_test.go`

**Approach:**

- Extend scheduler state for node id, broker fd, request sequence, network
  frame buffers, and node status table.
- Emit Linux syscalls for socket/connect/read/write/poll/close.
- Implement:
  - `__tetra_actor_node_connect`
  - `__tetra_actor_spawn_remote`
  - `__tetra_actor_node_status`
- Route remote actor handles in existing send/send_msg/typed transaction paths.
- Pump inbound frames on yield, recv block, timed recv, task wait, and no-ready
  scheduler states.
- Convert inbound send frames into local mailbox entries and wake blocked
  receivers.
- Convert inbound spawn frames into local actor spawns and send spawn acks.

**Verification:**

```sh
go test ./compiler/internal/actorsrt -run 'Distributed|Linux' -count=1
go test ./compiler -run 'DistributedActor|ActorsPingPong|TaskGroup' -count=1
```

**Done when:** Linux-x64 generated runtime objects contain the distributed
symbols and local actor behavior still passes.

## Task 7: Executable Distributed Actor Smoke

**Goal:** Prove distributed actor execution with real Linux-x64 node binaries
and broker TCP routing.

**Files:**

- Add examples under `examples/smoke/` or `examples/`
- Add a release/smoke script under `scripts/release/v0_4_0/` or existing
  release script structure
- Add a validator under `tools/cmd/validate-distributed-actor-runtime`
- Add tests under `tools/scriptstest`

**Approach:**

- Build node A and node B Tetra programs for `linux-x64`.
- Start `tetra actor-net` on loopback with a report path.
- Start node B; it connects and waits in the scheduler.
- Start node A; it connects, remote-spawns a responder on node B, sends
  i32/tagged/typed messages, receives a reply, and exits with an expected code.
- Run a negative smoke with missing node B and assert node A exits with the
  documented failure/status code without hanging.
- Validate the broker/runtime report shape and require real node process exit
  evidence.

**Verification:**

```sh
go test ./tools/cmd/validate-distributed-actor-runtime -count=1
go test ./tools/scriptstest -run 'DistributedActor|V040' -count=1
bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh --report-dir /tmp/tetra-distributed-actors-smoke
go run ./tools/cmd/validate-distributed-actor-runtime --report /tmp/tetra-distributed-actors-smoke/distributed-actors-linux-x64.json
```

**Done when:** The smoke launches real compiled Linux-x64 binaries and a real
TCP broker, and both positive and negative reports validate.

## Task 8: Readiness, Feature Registry, Manifest, And Docs Promotion

**Goal:** Promote only the proven Linux-x64 distributed actor surface.

**Files:**

- Modify `compiler/features.go`
- Regenerate `docs/generated/manifest.json`
- Regenerate `docs/generated/v1_0/manifest.json` if required by current repo
  policy
- Modify `docs/spec/current_supported_surface.md`
- Modify `docs/spec/actors.md`
- Modify `docs/spec/runtime_abi.md`
- Modify `docs/user/async_actors_guide.md`
- Modify `docs/release/v0_4_0_scope_decisions.json`
- Modify `tools/cmd/validate-v0-4-readiness`

**Approach:**

- Mark `actors.distributed-runtime` current only for the Linux-x64 production
  surface.
- Keep macOS/Windows/x86 explicitly out of scope.
- Teach readiness to accept the executable distributed runtime report and to
  continue rejecting transport-only/fake/incomplete evidence.
- Update docs with API, failure semantics, and production boundaries.

**Verification:**

```sh
go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json
go run ./tools/cmd/gen-manifest -o docs/generated/v1_0/manifest.json
go test ./tools/cmd/validate-v0-4-readiness -count=1
go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-manifest --manifest docs/generated/v1_0/manifest.json
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

**Done when:** Registry, manifests, docs, and readiness agree on the Linux-x64
production claim and reject proxy evidence.

## Task 9: Final Verification And Completion Audit

**Goal:** Prove the goal is complete against real artifacts.

**Verification:**

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
go run ./cli/cmd/tetra features --format=json > /tmp/tetra-v04-features.json
go run ./cli/cmd/tetra targets --format=json > /tmp/tetra-v04-targets.json
go run ./tools/cmd/validate-v0-4-readiness \
  --features /tmp/tetra-v04-features.json \
  --targets /tmp/tetra-v04-targets.json \
  --manifest docs/generated/manifest.json \
  --scope-decisions docs/release/v0_4_0_scope_decisions.json
graphify update .
```

**Done when:** The completion audit maps every goal requirement to concrete
files, command output, runtime reports, tests, and validators. No proxy signal
is accepted as proof of distributed runtime behavior.
