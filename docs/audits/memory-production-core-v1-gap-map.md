# Memory Production Core v1 Gap Map

Status: MPC-0 gap map for the Memory Production Core v1 plan.

The map keeps supported claims separate from future work. A gap is not a bug by
itself; it becomes a bug only if the compiler, docs, or reports claim the gap is
already solved.

## Current Gaps

| Gap | Status | Required future MPC | Conservative behavior now |
| --- | --- | --- | --- |
| General representation metadata namespace validator | partial | MPC-3 | Reject known unsupported assignments; do not expose metadata as normal user fields. |
| Full borrow/lifetime model beyond safe byte views | future | MPC-4 | Keep named lifetimes, generic lifetime parameters, arbitrary borrowed aggregates, and FFI lifetimes outside current claims. |
| Full mutable alias model | partial | MPC-5 | Reject ambiguous inout/mutable escape and aliasing paths. |
| Provenance/resource summaries v2 | partial | MPC-6 | Preserve existing summaries, and fall back to conservative rejection when summaries are missing. |
| Unsafe fact class promotion rules across all unsafe gateways | partial | MPC-7 | Unknown unsafe memory stays `unsafe_unknown`; only verified roots may produce bounded metadata. |
| Broader raw pointer verified-root operations | complete_narrow_slice | MPC-8 | `core.alloc_bytes` roots are bounded; external pointers remain conservative. |
| Raw slice gateway hardening for all element widths and targets | complete_narrow_slice | MPC-9 | Raw slices from unknown parts remain external/unknown and require unsafe. |
| Storage truth across stack/heap/island/regions | complete_narrow_slice | MPC-10 | Do not mark storage as validated unless actual lowering and validator agree. |
| Function-temp implicit region lowering | complete_narrow_slice | MPC-11 | Linux-x64 can validate `FunctionTempRegion` only when planned and actual lowering storage match and lowered IR contains function-temp region enter/make/reset evidence. Heap fallback remains evidence-only or conservative. |
| Actor/task/request memory boundary production semantics | complete_narrow_slice | MPC-12 | Copy or reject borrowed/unsafe crossings when ownership is unclear; actor zero-copy move rows are evidence-only unless production runtime validation exists. |
| Target capability matrix for memory behavior | partial | MPC-13 | Non-linux-x64 targets remain build/lower/artifact scoped unless runtime evidence exists. |
| Memory cost model | future | MPC-14 | No performance claim or storage optimization claim without validation. |
| Memory fuzz/property/stress oracle | partial | MPC-15 | Existing differential/property checks are bounded, not exhaustive. |
| Production gate and release audit | future | MPC-16 | Do not promote “Memory Production Core v1 complete” until all gates pass. |

## Immediate Closure

This goal closes the first vertical slice:

- compiler-owned `MemoryFactGraph` v0;
- `tetra.memory-report.v1` schema and validators;
- build integration for `.memory.json` report emission;
- verified-root projection for `core.alloc_bytes`;
- conservative projection for unknown raw pointers.

## Risk Controls

- Reports require `source_fact_id`, so they cannot invent truth.
- Storage and lowering rows require `lowered_artifact_id`.
- `safe_known` cannot be paired with `unsafe_unknown`.
- Validated claims require `validator_status: pass`.
- Planned stack, function-temp region, region, explicit-island, register, or
  eliminated storage that actually lowers as heap cannot be presented as a
  validated storage optimization.
- `FunctionTempRegion` allocation rows require matching actual
  `FunctionTempRegion` lowering plus lowered `IRRegionEnter`,
  `IRRegionMakeSlice*`, and `IRRegionReset` evidence before they can be
  validated.
- Explicit island storage claims require a named lowered island slice, region
  id, lifetime, active island handle, and validator rejection for return escape,
  use-after-free, or double-free paths in the supported IR surface.
- Actor/task/request boundary claims stay compiler-owned and conservative:
  borrowed actor payloads require `.copy()`, typed task String/slice payload
  transfer is not validated by the current payload-less spawn API, request/task
  region views cannot escape their entry scopes, and actor zero-copy report
  rows must remain `evidence_only` without production runtime validation.
