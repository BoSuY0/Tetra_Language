# Async And Actors Guide

Status: user guide for current local async/task/actor behavior. The current
support boundary is `docs/spec/current_supported_surface.md`; distributed actor
support, full cancellation, and structured concurrency remain outside the
current profile unless a future gate promotes them.

The current runtime ABI details are documented in `docs/spec/runtime_abi.md`.
Actor behavior and supported targets are documented in `docs/spec/actors.md`.

## Async Functions

The current async function MVP is checked synchronous lowering, not an
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

Typed task builtins are also covered by compiler tests:
`core.task_spawn_i32_typed<E>("worker")` and
`core.task_join_i32_typed<E>(task)`. Current MVP limits:

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

Typed actor messages are value-only. Enum payloads may cross module boundaries,
but the resolved payload must stay within the supported scalar/handle transfer
surface; reference-shaped payloads such as `String` are rejected with a
value-only payload diagnostic.

Distributed actors are supported for the Linux-x64 runtime path. The current
production surface covers the builtin Linux-x64 runtime with the `actornet`
loopback TCP broker, distributed node identity, remote actor handles, network
mailbox send/receive for i32, tagged, and typed messages, missing-node
failure/status propagation, and compatibility with existing cooperative task
cancel/join handles.

Non-goals for this release remain non-Linux-x64 distributed actor targets,
multi-threaded scheduling, full cancellation guarantees, and structured
concurrency beyond the documented cooperative task group handles. Transport-only
`tetra.actors.transport.v1` evidence is still not proof of distributed runtime
support; executable `tetra.actors.distributed-runtime.v1` smoke evidence is
required.

`tools/cmd/validate-actor-transport` exists for release evidence around the
transport shape. It validates a `tetra.actors.transport.v1` JSON
envelope/trace/hash report, but distributed actor runtime support is proven by
`scripts/release/v0_4_0/distributed-actors-linux-x64-smoke.sh` plus
`tools/cmd/validate-distributed-actor-runtime`.

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
