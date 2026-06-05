# Runtime Hardening v1

P24.1 adds `tetra.runtime.hardening.v1` as a bounded current-state runtime
hardening report for scope `p24.1_runtime_hardening`.

## Evidence Rows

| Row | Current evidence | Boundary |
| --- | --- | --- |
| `deterministic_traps` | Allocation contracts use `trap_or_stable_status`; wasm backends expose `emitWasmTrapIf`; web panic import formats `tetra panic` diagnostics. | This is not a full trap taxonomy for every target. |
| `oom_policy` | Every `runtimeabi.RuntimeAllocationContract` carries `AllocationFailureTrapOrStatus`; invalid negative and byte-size overflow sizes reject before allocator access. | No OOM recovery guarantee is claimed. |
| `stack_overflow_guard` | Backend stack-depth consistency checks reject malformed lowered function shapes. | No guard-page or recursion-depth runtime stack-overflow protection is claimed. |
| `integer_overflow_semantics_audit` | Optimizer coverage keeps unsafe `checkedNegI32` and `foldConstBinaryI32` folds rejected; allocation byte-size overflow rejects before allocation; const diagnostics reject overflowing global const expressions. | This is not a whole-language integer-overflow proof. |
| `allocator_corruption_detection` | `bounds_header`, `raw-pointer-bounds-v1`, and small-heap stale/double-free rejection are covered by runtime ABI evidence. | No full allocator-corruption detection proof is claimed. |
| `region_double_free_use_after_free` | Explicit island contracts include double-free/use-after-free hooks; `region.temp` includes use-after-free and region-reset hooks; debug region headers are part of the ABI. | This is ABI instrumentation evidence, not a complete temporal-memory-safety proof. |
| `actor_mailbox_overflow_policy` | Typed mailbox model uses bounded capacity, `blocking_recv_yield`, FIFO receive, and `ErrMailboxFull`. | Built-in actor runtime message-pool overflow remains a documented blocker and is not promoted. |
| `network_parser_limits` | HTTP parsers reject oversized headers/bodies and malformed inputs; PostgreSQL `ReadFrame` rejects malformed and oversized frames before payload allocation. | This is local parser evidence, not a full production network-stack hardening proof. |

## Non-Claims

- Full runtime-hardening proof is not claimed.
- Full stack-overflow protection is not claimed.
- OOM recovery guarantee is not claimed.
- Full allocator-corruption detection proof is not claimed.
- Production actor-mailbox promotion is not claimed.
- Runtime behavior does not change.
- Safe-program semantics do not change.
- No performance claim is made.

## Verification

Focused validator:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'P24RuntimeHardening' -count=1
```

Relevant package gates:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/runtimeabi ./compiler/internal/parallelrt ./compiler/internal/actorsrt ./compiler/internal/httprt ./compiler/internal/pgrt -count=1
```
