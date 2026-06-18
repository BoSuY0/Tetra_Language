# Async And Actors Guide

Status: user guide for current local async/task/actor behavior. The current
support boundary is `docs/spec/current_supported_surface.md`; distributed actor
support, full cancellation, and structured concurrency remain outside the
current profile unless a future gate promotes them.

The current runtime ABI details are documented in `docs/spec/runtime_abi.md`.
Actor behavior and supported targets are documented in `docs/spec/actors.md`.

## Async Functions

The current async function baseline is checked synchronous lowering, not an
independent async scheduler guarantee. Compiler coverage verifies async parsing,
`await` checking/lowering, rejection of `await` outside async functions, and
linux/amd64 build/run coverage for the tracked examples.

`examples/async_smoke.tetra` is in the native linux-x64 smoke profile and exits
42. That smoke proves the file is accepted, lowered, linked, and runnable; its
`main` currently returns 42 directly, so it does not execute the example
`caller()`/`await answer()` path at runtime.

`examples/core_async_smoke.tetra` is aligned with the current stable stdlib
example for `lib.core.async`: it imports the stable module, defines
`core_async_probe()` with `await async_lib.pair_sum(20, 22)`, and `pair_sum`
itself awaits `ready(lhs)` and `ready(rhs)`. The current linux-x64 smoke list
marks this example as excluded from the default profile. Its present evidence is
compiler build/run coverage on linux/amd64 plus generated-doc verification; the
entrypoint returns through `select_or(42, 0)`, so this is still not proof of
runtime execution of `pair_sum`/`ready`.

Target reports must keep this boundary explicit. Native linux-x64 smoke can be
treated as runtime evidence for `examples/async_smoke.tetra` as described above.
macos-x64, windows-x64, wasm32-wasi, and wasm32-web smoke reports are
build-only unless a target-specific runner/browser gate records real execution.

## Tasks

The v1.0 scope requires the release task ABI to be documented and tested before
any final release label. If a task feature is still described as an MVP or
planned feature in the specs, treat it as a limited baseline until release gate
evidence says otherwise.

The release-covered task smokes are `examples/task_smoke.tetra`,
`examples/task_sleep_deadline_smoke.tetra`,
`examples/task_join_wait_smoke.tetra`, and
`examples/task_group_cancel_smoke.tetra`, plus
`examples/task_group_lifecycle_smoke.tetra` for task group status/close
lifecycle coverage.
`examples/task_group_cancel_smoke.tetra` proves the narrow cooperative
`core.task_group_cancel` behavior where cancel wakes a sleeping child before its
timer and `core.task_join_result_i32` reports cancellation error `1`.
`examples/task_group_lifecycle_smoke.tetra` covers the release-visible
open -> spawn/join -> close/status path plus canceled-group close behavior.
`examples/deadline_aware_waits_smoke.tetra` covers absolute sleeps, bounded
joins, and timed actor receive in one native smoke.
`examples/wait_composition_smoke.tetra` covers nonblocking task poll, explicit
yield, timer readiness, tagged receive deadlines, and the first task/timer
select surface. Bounded stress evidence is
`examples/task_bounded_stress.tetra`. These use the cooperative `task.i32`
runtime path and require `uses runtime`.

The current cancellation matrix is scoped. Task-group cancellation wakes
`core.recv_until`, `core.recv_msg_until`, `core.task_join_result_i32`,
`core.task_join_until_i32`, and `core.select2_i32` waiters with checked error
`1`. Raw `core.task_join_i32` has no error field, so cancellation wakes it with
raw value `0`; use the result or timed join APIs for a checked status.
Non-timed actor receives such as `core.recv()` and `core.recv_msg()` do not
publish a cancellation result in this profile.

Typed task builtins are also covered by compiler tests:
`core.task_spawn_i32_typed<E>("worker")` and
`core.task_join_i32_typed<E>(task)`. Current limits:

- typed task error argument must be an enum;
- typed handle layout uses direct runtime wrappers for `2..4` and staged
  runtime wrappers for `5..8`;
- worker targets stay zero-argument synchronous `i32` functions; `2..4`
  requires `throws E`, while staged `5..8` accepts either `func worker() -> Int`
  or `func worker() -> Int throws E`;
- one-slot typed errors reuse the existing `task.i32` path.

Common safety diagnostics in task code:

- `task_spawn_i32 target must have shape ...` means the worker must be a
  zero-argument synchronous `Int` function.
- `cannot use joined resource 'task'` means a task handle was joined or moved
  already; store the result instead of reusing the handle.
- `cannot use closed resource 'group'` means a task group was closed on a
  previous path or loop iteration.

## Actors

Actor examples should be checked through the release smoke path instead of
manual inspection. Native host smoke is mandatory for `linux-x64`; target
build-only smoke is mandatory for other release targets.

`examples/actors_pingpong.tetra` covers the base actor mailbox ABI.
`examples/actors_tagged_stress.tetra` covers tagged `actor.msg` delivery through
`core.send_msg(to, value, tag)` and `core.recv_msg()`. Both builtin and
self-host runtimes must preserve the same observable exit codes for these
examples.
Timed receive uses `core.recv_until(deadline)` and returns
`actor.recv_result_i32 { value, error }`, with error `2` for timeout.
`core.recv_poll()` uses the same result shape without blocking, and
`core.recv_msg_until(deadline)` returns
`actor.recv_msg_result { value, tag, error }` for tagged message waits.

Actor declarations are supported in the current subset, including actor-local
state fields declared as `var`/`val`/`const` with scalar types
`Int`/`Bool`/`UInt8`/`UInt16`/`task.error`. Actor-state initializers must be
compile-time constants.

Typed actor messages follow the P6.1 sendability contract. Enum payloads may
cross module boundaries, but the resolved payload must stay within the supported
transfer surface: small scalar values copy, borrowed `String`/slice views must
use explicit `.copy()`, and an owned slice created from `core.island_make_*` can
move zero-copy through the local typed mailbox when the same enum payload also
carries the owning `island`. The sender cannot use the moved island or slice
after `send_typed`. Raw pointer, actor/task handle, unrelated runtime handle,
and distributed pointer/region zero-copy payloads remain rejected.

When building with `--explain`, `<output>.actor-transfer.json` records typed
mailbox metadata (`message_schema`, fixed local capacity, backpressure hook)
and per-payload copy/move behavior, including scalar copies, island moves,
explicit view copies, and zero-copy island-backed slice moves.

The builtin Linux-x64 mailbox is bounded. Local `core.send`,
`core.send_msg`, and `core.send_typed` return checked backpressure `-2` when
the receiver mailbox is full. That failure is recoverable after the receiver
drains messages; retry logic should yield, poll, or otherwise let the receiver
run before sending again. A failed `send_typed` does not publish a partial enum
payload. This is a local bounded mailbox contract, not automatic retry,
reconnect, or distributed delivery.

Actor lifecycle is intentionally narrow. A spawned actor runs until its entry
function returns, then becomes `done`; zero and nonzero actor entry returns use
the same user-visible state. Later local sends to that actor return checked
failure `-4`. Messages already delivered to another actor remain receivable
even if their sender is now done. Messages still sitting in a done actor's own
mailbox are not drained by a shutdown phase. There is no actor status, actor
join, actor exit-code, actor close API, supervision tree, restart behavior,
linking, or OTP-style lifecycle contract in the current runtime.

The current actor scheduler is single-threaded and cooperative. Executable
runtime tests cover bounded progress for yielding runnable actors and
deterministic deadline-order wake for sleeping actors. That evidence is a
scoped fairness/starvation boundary for the current runtime, not a preemptive
or production multi-threaded scheduler guarantee.

The P6.3 per-core scheduler work is currently a checked prototype model in
`compiler/internal/parallelrt`, validated by the parallel production smoke
report. It covers single-core compatibility, two-core work stealing, bounded
typed mailboxes, and actor transfer benchmark rows, but it does not claim a
general production actor runtime or production multi-threaded scheduler.

Distributed actors are supported for the Linux-x64 runtime path. The current
production surface covers the builtin Linux-x64 runtime with the `actornet`
loopback TCP broker, distributed node identity, remote actor handles, network
mailbox send/receive for i32, tagged, and typed messages, missing-node
failure/status propagation, and compatibility with existing cooperative task
cancel/join handles.

Missing-node/node_down remains checked status evidence, not a retry system.
When the broker reports a missing destination, the runtime can expose node-down
status after the network pump observes the frame. That does not imply automatic
retry, reconnect, restart, supervision, or delivery retry.

Non-goals for this release remain non-Linux-x64 distributed actor targets,
production multi-threaded scheduling, full cancellation guarantees, and
structured concurrency beyond the documented cooperative task-group wake matrix.
Transport-only
`tetra.actors.transport.v1` evidence is still not proof of distributed runtime
support; executable `tetra.actors.distributed-runtime.v1` smoke evidence is
required.

`tools/cmd/validate-actor-transport` exists for release evidence around the
transport shape. It validates a `tetra.actors.transport.v1` JSON
envelope/trace/hash report, but distributed actor runtime support is proven by
`scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh` plus
`tools/cmd/validate-distributed-actor-runtime`. The distributed runtime report
must carry same-commit `git_head`, `artifact_hashes`, ordered frame evidence,
and the exact foundation nonclaims listed below.

The scoped actor foundation gate also runs a race-enabled actor slice and
requires `actor broker leak cleanup` in parallel production evidence. Treat that
as bounded broker/runtime cleanup evidence, not as a full liveness or cluster
availability guarantee.

The authoritative actor runtime foundation gate is
`scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh`. It
writes `tetra.actor.production_foundation.v1` evidence at
`reports/actor-runtime-foundation/final/actor-runtime-foundation-manifest.json`
and validates `reports/actor-runtime-foundation/final/artifact-hashes.json`,
`distributed-actors-linux-x64/distributed-actors-linux-x64.json`, and
`parallel-production-linux-x64/parallel-production-linux-x64.json`. CI records
that through `.github/workflows/ci.yml`; package publishing records it through
`.github/workflows/release-packages.yml` before upload/release/container/Homebrew
publish steps.
The machine-readable gate contract is
`scripts/release/post_v0_4/contracts/actor-runtime-foundation-linux-x64.json`.

Actor foundation nonclaims remain explicit: no full Erlang/OTP actor runtime
claim, no cluster membership or reconnect/retry production claim, no non-Linux
distributed actor runtime support claim, no distributed zero-copy pointer or
region transfer claim, and no formal race proof claim.

The parallel production JSON report also carries top-level `diagnostics[]`
entries for negative actor/task cases. Those rows stabilize machine-readable
`code`, `severity`, `category`, `position`, and matching `expected_error`
evidence without promising byte-for-byte stable human diagnostic wording.

Runtime parity means builtin and self-host runtime modes must agree for the
documented actor/task smokes where both modes are applicable. It does not claim
that every runtime override object is trusted: override objects are checked for
target metadata and required `__tetra_*` exports before linking.

## Verification

```sh
go test ./compiler/... ./cli/... -run "Async|Await|Task|Actor|Actors|Runtime|Selfhost|ABI|Ownership|Stress" -count=1
go test ./compiler -run "Plan250Safety|Plan250Runtime|Plan250Link" -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
./tetra smoke --target linux-x64 --run=true --report /tmp/tetra-actors-smoke.json
bash scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh --report-dir /tmp/tetra-distributed-actors
```
