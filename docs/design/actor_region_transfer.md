# Actor Region Transfer Design

Status: P6.3 typed mailbox transfer model plus scheduler prototype evidence.

Actor runtime foundation scoped release truth is
`tetra.actor.production_foundation.v1`, produced by
`scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh` and
published through `.github/workflows/ci.yml` and
`.github/workflows/release-packages.yml`. Its final evidence includes
`reports/actor-runtime-foundation/final/actor-runtime-foundation-manifest.json`,
`reports/actor-runtime-foundation/final/artifact-hashes.json`,
`distributed-actors-linux-x64/distributed-actors-linux-x64.json`, and
`parallel-production-linux-x64/parallel-production-linux-x64.json`.
The machine-readable gate contract is
`scripts/release/post_v0_4/contracts/actor-runtime-foundation-linux-x64.json`.

This design keeps the transfer claim local: no full Erlang/OTP actor runtime
claim, no cluster membership or reconnect/retry production claim, no non-Linux
distributed actor runtime support claim, no distributed zero-copy pointer or
region transfer claim, and no formal race proof claim.

Actor zero-copy work now has a narrow local runtime slice. The safe sendability
model defines the payload matrix:

| Payload provenance | Required actor send behavior |
| --- | --- |
| Small scalar value | Copy. A scalar send is not a move operation. |
| Owned buffer | Copy or move, depending on the caller's transfer intent. |
| Owned region | Move. The receiver gets ownership and the sender loses access. |
| Borrowed view | Reject unless the source expression explicitly uses `.copy()`. |
| Unknown unsafe provenance | Reject unless an audited unsafe send contract is present. |
| `String` | Copy or move when owned; borrowed String views require `.copy()`. |

It must reject:

- borrowed references that can outlive the owner
- mutable aliases crossing actor boundaries
- unknown unsafe pointers without an audited unsafe send contract
- a moved region that remains usable by the sender

P6.0 diagnostics are intentionally stable enough to anchor follow-up runtime
slices:

```text
cannot send borrowed view across actor boundary; use .copy()
cannot use moved region after send
cannot send unknown unsafe provenance without audited contract
```

Region transfer has move semantics:

```text
move region R from actor A to actor B
sender loses access to R
receiver owns R
runtime does not copy R bytes
```

P6.1 implements the first runtime move form for local typed actor mailboxes:

```tetra
enum MoveMsg:
    case region(island, []i32)

core.send_typed(peer, MoveMsg.region(region, xs))
```

The checker accepts a non-`.copy()` slice payload only when it is known to be
created from `core.island_make_*` and the same message payload carries the
owning `island` value. The sender loses both the island handle and the
region-backed slice view after send. The builtin Linux-x64 typed mailbox stores
payload slots as 64-bit local slots, so pointer-like slice fields move without
byte copy. `--explain` writes `<output>.actor-transfer.json` with
typed mailbox metadata and per-payload copy/move rows.

The P6.2 actor-transfer report includes:

- `mailboxes[]` rows with `message_schema`, fixed local `capacity`,
  `backpressure`, `max_payload_slots`, `slot_width_bytes`, and
  `ownership_metadata`;
- scalar payload rows with `ownership: "copy"` and `transfer_mode: "copy"`;
- island payload rows with `ownership: "owned_region"` and
  `transfer_mode: "move"`;
- island-backed slice rows with `ownership: "owned_region_slice"`,
  `transfer_mode: "zero_copy_move"`, `runtime_path:
  "actor_mailbox_zero_copy_region_slot"`, `bytes_copied: 0`, and
  `zero_copy: true`.

MPC-12 makes that report boundary machine-readable: owned-region
`zero_copy_move` rows carry `claim_level: "evidence_only"`,
`boundary_scope: "local_typed_mailbox_owned_region_slice_move"`, and
`production_runtime_validated: false`. The row is real local runtime evidence,
but it is not a full production actor-runtime or distributed zero-copy claim.

Compiler facts required for broader forms:

- region is owned
- no active borrows
- not freed before send
- no sender use after send
- the receiver boundary is local or has an audited serialization contract

The v0 checker model lives in `compiler/internal/actorsafety`. It is not a
runtime rewrite. It encodes the first sendability rules: small scalars copy,
owned buffers and owned `String` values can copy or move, borrowed slices and
borrowed `String` views require explicit copy, owned regions must move, sender
use after region move is illegal, and unsafe pointers need an audited unsafe
send contract. P6.2 also validates typed mailbox metadata shape: a message
schema, positive capacity, and an explicit backpressure policy.

Distributed typed actor frames remain scalar/wire-value evidence. They do not
claim pointer or region zero-copy transfer across nodes.

Typed task spawn has no payload expression in the current source API. Its
boundary remains conservative: worker signatures and typed error payloads must
be sendable, String/slice task error payload transfer is not promoted to a
validated boundary copy path, and request/task region data may not escape its
explicit entry scope unless a later lifetime model proves that transfer.

P6.3 adds a checked scheduler prototype model in
`compiler/internal/parallelrt`. It is design and release evidence, not a
promotion of the production Linux-x64 actor runtime to multi-threaded
scheduling. The model covers:

- per-core run queues with single-core FIFO compatibility;
- two-core work stealing from another core's queue tail;
- bounded typed mailboxes with explicit `blocking_recv_yield` backpressure
  metadata;
- `ActorMemoryDomainReport` rows with mailbox byte capacity, queued bytes,
  reclaimed bytes, message-pool/slab bytes, and `byte_limit_reached`
  backpressure status;
- local owned-region `DomainMoves` from sender actor domain to receiver actor
  domain with `bytes_copied: 0`;
- actor ping-pong/fanout comparison rows in the
  `tetra.parallel.production.v1` smoke report;
- owned-region message transfer rows that report `zero_copy_move` with
  `bytes_copied: 0`, while borrowed views still require an explicit copy.

P6.4 Linux IO reactor evidence starts from the existing Linux epoll path:
`compiler/internal/netrt` owns host-side epoll polling, and the Linux-x64 actor
runtime emits `__tetra_net_epoll_*` helper symbols. This slice intentionally
does not claim io_uring, kqueue, IOCP, or a cross-platform event-loop
abstraction.
