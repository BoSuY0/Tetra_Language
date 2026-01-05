# Actors (MVP, v0.13 draft)

Actors are an isolation + message-passing concurrency model built on top of Tetra’s existing foundations:
Islands (region memory), and the explicit safe/unsafe boundary.

This document specifies the minimal (MVP) actor runtime and language surface needed to ship v0.13.

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

## Model

- An **actor** is an isolated unit of execution with a **mailbox** (FIFO queue).
- Actors run under a **single-thread cooperative scheduler**.
- An actor can:
  - spawn new actors,
  - send messages,
  - receive messages (blocking, but implemented cooperatively).

## Types

- `actor` — an opaque handle identifying an actor (MVP: small integer handle).

## Core builtins (MVP)

All actor builtins are **safe** (do not require `unsafe`).

### `core.spawn(name: str) -> actor`

Spawns a new actor that executes the function named by `name`.

MVP constraints:
- `name` must be a string literal known at compile time.
- The target function must exist and have the shape: `fun <name>(): i32`.
- x64 targets only in the first iteration (other architectures: planned).

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
  - blocked in `core.recv()`,
  - finished execution.
- Scheduler policy: round-robin over runnable actors (MVP).

## Memory

MVP messages are `i32` values plus an implicit sender handle.

Future extensions (post-MVP):
- Copy-based passing of `[]u8` into a receiver-owned island.
- Ownership transfer of message islands (move/consume semantics).

## Runtime ABI surface (internal)

Actors are implemented by linking a runtime object that exports a small set of reserved symbols (e.g. `__tetra_entry`,
`__tetra_actor_*`). The exact symbol list and calling conventions are documented in `docs/spec/runtime_abi.md`.
