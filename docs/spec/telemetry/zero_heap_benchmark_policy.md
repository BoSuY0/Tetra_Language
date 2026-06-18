# Zero-Heap Benchmark Policy

Status: local Tier 1 benchmark validator policy for the current memory optimization track.

This policy defines which successful Tetra benchmark rows must remain zero Tetra heap at runtime. It
does not claim zero RSS, zero process memory, zero heap for every Tetra program, or official
benchmark status.

## Evidence Boundary

For this policy, zero heap means all of the following are true for the Tetra row:

```text
tetra_metadata.heap_allocations == 0
memory_evidence.heap_alloc_bytes.evidence_class == runtime_measured
memory_evidence.heap_alloc_bytes.total_alloc_bytes == 0
memory_evidence.heap_alloc_bytes.allocation_count == 0
runtime heap sidecar heap_total_alloc_bytes == 0
runtime heap sidecar heap_allocation_count == 0
runtime heap sidecar heap_current_bytes == 0
runtime heap sidecar heap_peak_bytes == 0
```

The runtime heap sidecar must use the contract in `docs/spec/telemetry/runtime_heap_telemetry.md`.

These are not substitutes for heap evidence:

- RSS samples;
- allocation report estimates;
- Go `runtime.MemStats`;
- binary size;
- missing JSON fields;
- `unsupported` or `blocked` heap evidence on a measured linux-x64 Tetra row.

## Required Rows

The current zero-heap-required Tier 1 categories are:

```text
integer loops
function calls
hash table
startup time
```

Reasoning:

- `integer loops` and `function calls` have no user-level allocation need in the current Tier 1
  source.
- `hash table` is included only after the call-aware stack/const-length follow-up report proved
  `hash_table_tetra` runs with zero Tetra heap allocations.
- `startup time` is included because the current row has no user allocation and the runtime heap
  sidecar proves zero counted Tetra heap allocation.

## Not Yet Required

These rows are not zero-heap-required in the current policy:

```text
slice sum
bounds-check loops
allocation
region/island allocation
JSON parse/stringify
HTTP plaintext/json
PostgreSQL single/multiple/update
actor ping-pong
parallel map/reduce
binary size
compile time
```

`slice sum` and `bounds-check loops` are deliberately excluded because the current local report
still shows one measured heap allocation for each row.

`allocation` is excluded because it is an allocation benchmark; forcing it to zero heap would blur
benchmark intent.

`region/island allocation` is excluded until its build-failed or missing-feature state is closed
with fresh evidence.

JSON, HTTP, PostgreSQL, actor, and parallel rows are excluded until runtime allocator classes,
domain bytes, and actor/runtime limitations are separately closed.

`binary size` and `compile time` are meta rows. They may report zero runtime heap for the compiled
Tetra binary, but they are not part of the first zero-heap-required policy because their primary
measured dimension is not a simple program runtime memory path.

## Validator Rule

`tools/cmd/validate-local-benchmark-tier1` must reject a measured Tetra row in a zero-heap-required
category when any required heap field is non-zero, missing as runtime evidence, unsupported,
blocked, fake-measured, stale, or not backed by the row's heap sidecar.

Excluded rows still need truthful memory evidence. They must not fake heap, RSS, domain, or
allocation estimate fields, but they do not fail merely because they have runtime heap allocation.

## Promotion Rule

A category can be added to the zero-heap-required list only after a fresh local report proves all of
the required heap evidence fields are zero for the Tetra row and the validator has a regression test
that fails when heap allocation returns.

## Dedicated Microbenchmarks

Dedicated zero-heap microbenchmarks live outside the Tier 1 P20 comparable matrix. They are
Tetra-only compiler/runtime guardrails, not language performance comparisons.

Current dedicated categories:

```text
zero heap fixed local array sum
zero heap read-only local call slice
zero heap small struct copy
zero heap borrowed view sum
zero heap copy eliminated unused
```

Placement rule:

- keep these categories out of `requiredP20Categories`;
- keep language fixed to `tetra`;
- require source/build metadata for each category;
- use runtime heap telemetry when these specs are executed;
- promote a category into Tier 1 only if equivalent C, C++, and Rust rows are added with identical
  workload intent and without changing the benchmark claim.

Current implementation lives in:

```text
tools/internal/zeroheapbench
tools/cmd/local-benchmark-zero-heap
tools/cmd/validate-local-benchmark-zero-heap
```
