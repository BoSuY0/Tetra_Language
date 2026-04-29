# Async And Actors Guide

Status: user guide for the release actor/task surface.

The current runtime ABI details are documented in `docs/spec/runtime_abi.md`.
Actor behavior and supported targets are documented in `docs/spec/actors.md`.

## Tasks

The v1.0 scope requires the release task ABI to be documented and tested before
any final release label. If a task feature is still described as an MVP or
planned feature in the specs, treat it as a limited baseline until release gate
evidence says otherwise.

The release-covered task smokes are `examples/task_smoke.tetra`,
`examples/task_sleep_deadline_smoke.tetra`, and
`examples/task_join_wait_smoke.tetra`; `examples/deadline_aware_waits_smoke.tetra`
covers absolute sleeps, bounded joins, and timed actor receive in one native
smoke. `examples/wait_composition_smoke.tetra` covers nonblocking task poll,
explicit yield, timer readiness, tagged receive deadlines, and the first
task/timer select surface. Bounded stress evidence is
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

Non-goals for this release remain distributed actors, multi-threaded
scheduling, cancellation guarantees, and structured concurrency beyond the
documented cooperative task group handles.

## Verification

```sh
go test ./compiler/... ./cli/... -run "Async|Await|Task|Actor|Actors|Runtime|Selfhost|ABI|Ownership|Stress" -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
./tetra smoke --target linux-x64 --run=true --report /tmp/tetra-actors-smoke.json
```
