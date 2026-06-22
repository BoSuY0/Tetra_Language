# Actor System Messages v1

Status: Approved.
Owner approval: `TETRA-V1-ACTOR-SYSTEM-MESSAGES-B-2026-06-20`.
Scope: V1-P01 system-message API and isolated Linux x64 builtin-runtime lane.

## Decision

Tetra v1 uses a separate source API and a separate runtime-owned system queue
for actor system messages.

Ordinary actor receives consume only the user mailbox:

- `core.recv`
- `core.recv_msg`
- `core.recv_typed`
- user poll and timed receive variants

System receives consume only the system mailbox through `lib.core.actors`:

- `recv_system()`
- `poll_system()`
- `recv_system_until(deadline: Int)`

The runtime must not implement this design as one shared linked list plus
scanning, filtering, skipping, or reordering. User and system lanes have
separate heads, tails, counters, accounting, wait reasons, and wake rules.

## Alternatives

### A. Ordinary Receive Extension

Rejected. Extending ordinary receive results would make existing user receive
semantics observe runtime lifecycle traffic and would force every typed or
tagged user receive path to filter system events.

### B. Separate System Receive

Approved. This keeps ordinary mailbox semantics stable and gives lifecycle,
monitor, link, and cluster events an unforgeable runtime lane.

### C. User-Level Tagged Messages

Rejected. `core.send_msg(peer, value, tag)` and typed actor messages are
ordinary user messages and are forgeable by source code. They cannot represent
runtime-owned `exit`, `down`, or `node_down` events.

## Public API

The stable public source API lives in module `lib.core.actors`.

```tetra
pub enum ExitReason:
    case normal
    case shutdown(Int)
    case error(Int)
    case canceled
    case killed
    case node_down(Int)
    case protocol_error(Int)
    case runtime_error(Int)
    case unknown(Int, Int)

pub enum NodeDownReason:
    case graceful_leave
    case lease_expired
    case connection_lost
    case authentication_failed
    case protocol_mismatch
    case control_plane_unavailable
    case unknown(Int)

pub enum SystemMessage:
    case exit(actor, ExitReason)
    case down(actor.monitor, actor, ExitReason)
    case node_down(actor.node, NodeDownReason)

pub enum SystemReceiveResult:
    case message(SystemMessage)
    case empty
    case timeout
    case canceled
    case runtime_closed
    case invalid_state(Int)

pub func recv_system() -> SystemReceiveResult uses actors, runtime
pub func poll_system() -> SystemReceiveResult uses actors
pub func recv_system_until(deadline: Int) -> SystemReceiveResult uses actors, runtime
```

The same stable module owns the v1 lifecycle wrapper names approved by the
actor platform plan:

The status wrapper is backed by the raw core bridge
`core.actor_status_raw -> actor.status_result_raw`, whose `status_code` and
`result` slots preserve `ok`, `invalid`, and `stale` taxonomy before the stable
`StatusResult` decoder runs. `stop`, `link`, and `monitor` reuse that stable
preflight before calling lower-level raw lifecycle operations, so invalid and
stale local refs do not pass through the old one-slot status enum path. This raw
bridge is compiler/runtime-owned, not a user-facing lifecycle API.

`wait` and `wait_until` decode the raw `actor.wait_result` reason slot so local
invalid refs produce `WaitResult.invalid` and stale generation mismatches
produce `WaitResult.stale`, both with the public `actor.status.dead` status
slot at the raw ABI boundary.

```tetra
pub enum ActorStatus:
    case starting
    case ready
    case running
    case blocked
    case sleeping
    case waiting
    case stopping
    case exited_normal
    case exited_error(Int)
    case canceled
    case restarting
    case dead
    case unknown(Int)

pub enum StatusResult:
    case ok(ActorStatus)
    case invalid
    case stale
    case node_down

pub enum WaitResult:
    case exited(ExitReason)
    case timeout
    case canceled
    case invalid
    case stale
    case node_down

pub enum StopResult:
    case requested
    case already_exited(ExitReason)
    case invalid
    case stale
    case node_down

pub enum LinkResult:
    case linked
    case already_linked
    case target_exited(ExitReason)
    case resource_exhausted
    case invalid
    case stale
    case node_down

pub enum MonitorResult:
    case monitoring(actor.monitor)
    case target_already_exited(actor.monitor)
    case resource_exhausted
    case invalid
    case stale
    case node_down

pub func status(target: actor) -> StatusResult uses actors
pub func wait(target: actor) -> WaitResult uses actors, runtime
pub func wait_until(target: actor, deadline: Int) -> WaitResult uses actors, runtime
pub func stop(target: actor, reason: ExitReason) -> StopResult uses actors
pub func link(target: actor) -> LinkResult uses actors
pub func unlink(target: actor) -> Bool uses actors
pub func monitor(target: actor) -> MonitorResult uses actors
pub func demonitor(reference: actor.monitor, flush: Bool) -> Bool uses actors
pub func set_trap_exit(enabled: Bool) -> Bool uses actors
```

P01 implements the local wrapper surface over the current raw ABI. P06 and
P10/P11 remain responsible for full link/monitor/trap event production and
remote node production semantics.

Logical event mapping:

```text
actor.system.exit      -> SystemMessage.exit
actor.system.down      -> SystemMessage.down
actor.system.node_down -> SystemMessage.node_down
```

## Compiler Raw Contract

The compiler owns the raw receive value:

```text
actor.system_recv_raw
```

It is a `repr(C)` value with eight slots. The raw value has this field order:

```text
0 status       i32
1 kind         i32
2 subject      actor
3 monitor      actor.monitor
4 node         actor.node      # occupies node_id + node_epoch slots
6 reason_kind  i32
7 reason_code  i32
```

Source code may read fields for the private `lib.core.actors` decoder, but it
must not construct, assign, send, or serialize this raw value. The same runtime
ownership metadata applies to `actor.monitor` and `actor.node`.

## Runtime ABI

System receive uses a transaction pattern:

```text
__tetra_actor_recv_system_begin(mode: i32, deadline: i32) -> i32 status
__tetra_actor_recv_system_slot(index: i32) -> machine word
__tetra_actor_recv_system_count() -> i32
```

Modes:

```text
0 = blocking
1 = poll
2 = absolute-deadline
```

Statuses:

```text
0 = message
1 = empty
2 = timeout
3 = canceled
4 = runtime_closed
5 = invalid_state
```

Message slot order after `status=message`:

```text
0 kind
1 subject actor ref
2 monitor ref
3 node_id
4 node_epoch
5 reason_kind
6 reason_code
```

`__tetra_actor_recv_system_count()` returns `7` only for a staged message and
`0` for non-message statuses. `slot(index)` returns `0` for invalid indexes or
non-message status and must not expose stale scratch data.

## Queue Invariants

Required invariants:

```text
user_recv_consumed_system_event = 0
system_recv_consumed_user_event = 0
FIFO violations = 0
duplicate_down_events = 0
forged_system_events_accepted = 0
silent_system_event_drops = 0
system_event_live_bytes_after_shutdown = 0
```

User send wakes an actor only when it is waiting for the user mailbox or future
select. System enqueue wakes an actor only when it is waiting for the system
mailbox or future select. A system receive must not update `core.sender()`.

## Memory Bounds

Each actor owns bounded system mailbox accounting:

- head and tail;
- queued count;
- reserved credits;
- current, peak, and reclaimed bytes;
- overflow attempts.

The scheduler owns a bounded system event pool and free list. Creating a
monitor, link, or node-watch relation must reserve system-event credit before
the relation becomes active. If reservation fails, the relation is not created
and the public result is `resource_exhausted`. Runtime terminal events must not
be silently dropped because a mailbox is full.

## Ordering

Within one actor system lane, events are FIFO by runtime commit order.

For one target exit commit:

1. The terminal state and reason become immutable.
2. Trapped link `exit` events enqueue in deterministic relation-ID order.
3. Monitor `down` events enqueue in monitor-reference order.
4. Actor waiters wake.
5. Cleanup and finalizers proceed under the lifecycle contract.

No global ordering is claimed between independent target exits or between user
and system lanes.

## Unforgeability

`SystemMessage` is a public value type for pattern matching, not a permission to
inject into the runtime system lane.

Compiler/runtime guards must reject:

- `core.send_typed(peer, SystemMessage.exit(...))`;
- `core.recv_typed<SystemMessage>()`;
- raw `actor.system_recv_raw`, `actor.monitor`, or `actor.node` construction;
- ordinary actor send paths that try to select system kind by tag.

The diagnostic for canonical system messages should say:

```text
runtime system messages cannot be sent through the ordinary actor mailbox;
use actor lifecycle, link, monitor, or cluster APIs
```

User-defined enums with an unrelated canonical type, including a local enum
named `SystemMessage`, remain ordinary sendable values when their payloads are
otherwise legal.

## Nonclaims

P01 implements the source-level API, raw ABI, and isolated lane foundation for
the Linux x64 builtin runtime. It does not claim full link/monitor production
semantics, authenticated cluster membership, TLS/mTLS authority, reconnect
ordering, or real distributed node-down production. Local producers are
completed in P06, and authenticated remote node-down production is completed in
P10/P11.
