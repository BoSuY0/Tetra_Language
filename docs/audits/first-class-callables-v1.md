# First-class Callables v1

Status: P22.1 evidence/report contract.

Schema: `tetra.language.first_class_callables.v1`
Scope: `p22.1_first_class_callables_v1`

P22.1 closes the master-plan callable requirement by turning the existing
safe callable ABI into a same-branch report and validator. The report does not
add a new language mode. It builds concrete witnesses by parsing, checking, and
lowering representative programs so drift in semantic metadata or lowering
changes becomes visible.

## Coverage Rows

| Row | Current evidence | Boundary |
| --- | --- | --- |
| `fnptr_fast_path` | `FnPtrEnvSlotCount = 8` and `FnPtrSlotCount = 9`; a one-capture function-typed local lowers without `IRAllocBytes`. | Bounded safe captures use the 9-slot `fnptr` fast path only while they fit the eight-slot environment envelope. |
| `fat_callable_handle` | `CallableHandleSlotCount = 4`; a nine-capture callable lowers with one `IRAllocBytes`, nine `IRMemWritePtrOffset`, nine `IRMemReadPtrOffset`, and `IRCall` arg/ret slots `10/1`. | Larger safe immutable by-value captures use a fixed 4-slot handle; no exploding return-slot model is claimed. |
| `capture_safety_classifier` | `compiler/internal/semantics/callable_escape.go` and `closure_captures.go` classify local, return, global, callback, and thread boundaries. | Only safe immutable by-value captures are promoted. |
| `mutable_capture_escape_diagnostics` | Mutable capture global/heap escape remains covered by stable diagnostics. | No mutable by-reference capture support is claimed. |
| `resource_thread_escape_diagnostics` | Pointer/resource capture and thread-boundary callable escape stay rejected by the classifier. | No pointer/resource capture support or thread-boundary callable transfer is claimed. |
| `fixed_abi_width` | Witnesses record 9-slot `fnptr`, 4-slot handle, `ReturnSlots = 4`, and handle dispatch arg/ret slots `10/1`. | ABI width is fixed evidence, not a variable-width callable ABI. |
| `cross_module_interface_metadata` | Generated `.t4i` metadata preserves return function symbol, capture count, heap escape kind, handle flag, and `ReturnSlots = 4`. | Cross-module metadata does not imply dynamic callable dispatch. |
| `storage_and_callback_paths` | Existing semantics/lowering tests cover aliases, struct fields, enum payloads, callback arguments, and returns for safe callable values. | Runtime generic callable polymorphism and dynamic callable dispatch remain outside the current model. |

## Validator Contract

`ValidateP22FirstClassCallableCoverage` rejects:

- missing or duplicate rows;
- missing witness references;
- placeholder evidence;
- variable-width ABI claims;
- exploding return-slot claims;
- mutable by-reference capture support claims;
- pointer/resource capture support claims;
- thread-boundary callable transfer claims;
- runtime generic callable polymorphism claims;
- dynamic callable dispatch claims;
- unsafe lifetime relaxation claims;
- performance claims;
- runtime behavior changes;
- safe-program semantic changes.

## Non-claims

- No variable-width callable ABI is claimed.
- No exploding callable return slots are claimed.
- No mutable by-reference capture support is claimed.
- No pointer/resource capture support is claimed.
- No thread-boundary callable transfer is claimed.
- No runtime generic callable polymorphism is claimed.
- No dynamic callable dispatch is claimed.
- No unsafe lifetime relaxation is claimed.
- No performance claim is made.
- No runtime behavior change beyond the existing callable ABI is claimed.
- Safe-program semantics do not change.

## Verification

Focused evidence:

```text
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'P22FirstClassCallable|ValidateP22FirstClassCallable' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/semantics -run 'Callable|Closure|FunctionType' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/lower -run 'Callable|FunctionType' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics -run 'Callable|Closure|FunctionType|Interface|FeatureRegistry' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
```
