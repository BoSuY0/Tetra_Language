# Runtime Allocation Contract

Status: P15.4 domain-aware allocation report evidence. This document defines the allocator behavior
that P5 runtime implementations must satisfy before claiming faster heap or
region allocation. P5.1 implements the first `linux-x64` safe-slice small heap
path; P5.2 hardens explicit island/region allocation; P5.3 lets the planner
name bounded function-local temporary regions while the backend still reports
the current heap fallback; P15.2 upgrades the small safe-slice evidence to a
per-core/thread-local allocator ABI with size-class free-list reuse; P15.3 adds
raw allocation-base metadata for unsafe `core.alloc_bytes` reports while keeping
arbitrary raw pointers checked and external/unknown; P15.4 adds domain-aware
allocation report metadata for process/default and explicit-island ownership
accounting without changing allocator behavior.

## Scope

The contract covers:

- raw unsafe `core.alloc_bytes`;
- safe `make_u8`, `make_u16`, `make_i32`, and `make_bool` allocation intents;
- explicit island allocation through `core.island_new` and
  `core.island_make_*`;
- reserved compiler-owned temporary regions such as `region.temp`.

`docs/spec/memory_backend_vnext.md` extends this contract with a target-neutral
MemoryBackend vocabulary for `reserve`, `commit`, `decommit`, `release`,
`trim`, and `footprint`. That vocabulary describes backend and report evidence;
it does not replace the allocation paths below and does not turn
allocation-report estimates into runtime RSS/footprint measurements.

P5.3 does not make implicit compiler-selected regions production-grade in
lowered code. Region rows must show both the planned region and the current
`actual_lowering_storage`; unsafe raw `core.alloc_bytes` now reports P15.3
allocation-base metadata, but arbitrary derived raw pointers and raw-slice
gateways remain checked external/unknown unless rooted in verified allocation
metadata.

## Shared Rules

All runtime allocation APIs have a contract entry in
`compiler/internal/runtimeabi.RuntimeAllocationContracts`.

Every contract defines:

- the API name;
- the runtime path (`heap`, `small_heap_bump`, `per_core_small_heap`, `large_mmap`,
  `planner_selected`, `explicit_island`, or `region`);
- minimum returned-pointer alignment;
- zero-size behavior;
- negative-size guard behavior;
- byte-size overflow guard behavior;
- allocator failure behavior;
- debug instrumentation hooks;
- report hooks.

Alignment is currently frozen at 16 bytes for compiler-owned heap, slice,
island, and region paths. A future target may use a stronger alignment, but it
must still satisfy this minimum.

Negative lengths and byte-size overflows are rejected before allocator access.
For island slice allocation, they are also rejected before island metadata is
read. This preserves the existing P0/P2 rule that safe length validation cannot
depend on report flags or allocator side effects.

## Zero Size

`core.alloc_bytes(0)` is an invalid unsafe runtime precondition and follows the
existing stable failure/status path.

Safe `make_*` with length zero creates a canonical empty slice with no backing
allocator access.

`core.island_make_*` with length zero creates a canonical empty slice without
reading or mutating island metadata.

`core.island_new(0)` is reserved as an empty island header path. It may allocate
only the region header/debug guard storage required by the current target.

## Runtime Paths

`heap`
: The conservative runtime heap class. On `linux-x64`, P5.1 routes non-empty
  safe `make_u8`, `make_u16`, `make_i32`, and `make_bool` requests through a
  shared per-function helper. Requests up to 4096 bytes use a writable
  bump-pointer chunk acquired from `mmap` in 64 KiB pages and rounded to a
  16-byte size class. Larger safe-slice requests use the helper's direct
  `mmap` fallback. Unsupported targets and unsafe raw allocation remain on the
  older conservative path.

`small_heap_bump`
: P5.1 `linux-x64` safe-slice fast path. The helper stores `bump` and `end`
  pointers in writable object data, refills from the OS only when the current
  chunk is empty or full, and returns 16-byte aligned pointers. It is a no-GC
  allocator. It remains the historical path name for pre-P15.2 evidence.

`per_core_small_heap`
: P15.2 safe-slice allocator path. The ABI records per-core metadata
  (`bump_offset`, `chunk_refills`, free lists, allocation/free/reuse counts),
  rounds small requests into the 16-byte size classes, refills from 64 KiB
  chunks, and allows reuse only from the same core and same size class. Stale
  and double-free handles are rejected by generation metadata in the ABI model.

`large_mmap`
: P5.1 safe-slice large fallback. Length checks and byte-size overflow checks
  still happen before entering the helper; the fallback maps the requested byte
  count directly.

`planner_selected`
: Safe slice allocation intent. The allocation planner may select eliminated,
  stack, explicit island, region, heap, or large-OS storage only when validation
  proves the chosen class. Unsupported cases stay heap.

`explicit_island`
: User-written island storage. P5.2 uses a header-owned bump pointer with
  16-byte alignment for each non-empty `core.island_make_*` allocation. The
  backend rejects negative sizes and byte-size overflow before reading island
  metadata, rejects exhausted regions before committing the new bump, and frees
  the mapped island region in bulk at island lifetime end. `core.island_new`
  also rejects negative and too-large payload sizes before the host allocator is
  called.

`region`
: Compiler-owned scoped region storage. P5.3 models only the obvious
  function-local temporary copy region. Reports name `region:<function>:temp`,
  lifetime `function:<function>`, runtime path `region`, and the reason, while
  also saying `actual_lowering_storage: Heap` until implicit runtime region
  lowering exists.

## Failure And Debug Behavior

Allocator failures and invalid preconditions use stable trap/status behavior for
the target. Silent wraparound, target-dependent crashes, and metadata-only
success claims are not valid evidence.

Debug instrumentation hooks are part of the contract:

- heap paths expose allocation bounds metadata where supported;
- island paths expose double-free and use-after-free instrumentation in debug
  mode where supported. The current native island debug path keeps the header
  readable for the double-free marker and protects the payload with
  `mprotect(PROT_NONE)`/`VirtualProtect(PAGE_NOACCESS)` where the target ABI
  supports it;
- region paths expose reset and use-after-free instrumentation where supported.

## Report Hooks

Every allocation report format that claims P5 runtime allocation behavior must
be able to expose:

- storage class;
- runtime path;
- bytes requested;
- bytes reserved;
- allocator class;
- allocator scope;
- allocator reuse policy;
- region id when applicable;
- lifetime;
- domain id;
- domain kind;
- domain owner;
- domain lifetime;
- debug mode.

P5.1 allocation plan reports populate these hooks for constant-size `linux-x64`
heap safe-slice allocations: `runtime_path`, `allocator_class`,
`bytes_requested`, and `bytes_reserved`. P5.2 also populates the same byte
hooks plus `region_id`, `lifetime`, and `debug_mode` for explicit island slice
allocation intents. P5.3 adds region-planning hooks for bounded function-local
temporary copy allocations, including dynamic-length copies where byte counts
are not constant. P5.4 upgrades allocation reports to schema v2: every report
has a `summary` with allocation count, planned-storage counts,
actual-lowering counts, runtime-path counts, requested bytes, reserved bytes,
allocator-class counts, allocator-scope counts, allocator-reuse-policy counts,
and per-region summaries, and validation rejects any summary that does not
match the exact plan rows. P15.2 heap safe-slice rows use
`runtime_path: per_core_small_heap`, `allocator_class: small_<N>`,
`allocator_scope: core:0` for the single-threaded report model, and
`allocator_reuse_policy: same_core_same_size_class_free_list`. P15.4 adds a
nested `domain` object to allocation plan rows and per-domain summaries:
default heap/small-heap/stack/register/eliminated rows are charged to
`domain:process`, explicit islands are charged to `domain:<island-region>`,
and external rows are charged to an external domain. Planned function-temp and
actor-move storage remains conservative in allocation reports until runtime
ownership transfer evidence exists.

P5 allocator benchmark evidence classification is owned by
`tools/cmd/memory-production-smoke` and `tools/validators/memoryprod`. The
smoke tool builds a generated Linux-x64 small-allocation benchmark with
`--emit-alloc-report`, reads the schema-v2 allocation summary, and records the
small heap benchmark as `evidence_class: allocation_report_estimate` with
`method: allocation_report_summary`. The benchmark records the estimated
syscall reduction from 64 mmap-per-allocation calls to one 64 KiB chunk refill;
it does not claim runtime RSS, pprof, MemStats, `/usr/bin/time -v`, or `strace`
measurement unless a separate `runtime_measured` artifact is present.
Memory production release bundles now include `ram-measurement.json` as that
separate capture artifact, using schema `tetra.memory.ram-measurement.v1` and
MemStats snapshots when available. The artifact carries a `summary` and
required `metric_samples` for heap allocation bytes, requested/reserved bytes,
copied bytes, current/peak RSS, and per-actor domain bytes. The validator keeps
those metrics separated by evidence class: MemStats heap bytes may be
`runtime_measured`, allocation-report bytes must stay estimates when present,
and MemStats RSS is rejected as a fake RSS measurement unless a real RSS-capable
method such as `time_v`, `strace`, or `pprof` supplies that metric.
