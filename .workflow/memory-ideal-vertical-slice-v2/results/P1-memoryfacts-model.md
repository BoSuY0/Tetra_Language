# P1 MemoryFacts And MiniMemoryModel Result

Status: accepted

## RED Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-red-memorymodel go test ./compiler/internal/memorymodel -count=1`
  failed because v2 model symbols did not exist:
  `WrapperFunctionValue`, `WrapperCallbackArg`,
  `CallbackTargetKnown`, `OutcomeConservativeUnknownCallbackTarget`, and
  callback inout event/outcome symbols.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-red-memoryfacts go test ./compiler/internal/memoryfacts -count=1`
  failed because function/callback borrow projections were still emitted as
  `aggregate_contains_borrow`, and v2 derived claims were not parent-validated.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-red-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1`
  failed because report validation did not require v2 parent facts and
  correlation validation only recognized v0/v1 row sets.

## GREEN Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-memorymodel go test ./compiler/internal/memorymodel -count=1`
  passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-memoryfacts go test ./compiler/internal/memoryfacts -count=1`
  passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1`
  passed.

## Accepted Changes

- `compiler/internal/memorymodel/mini.go` adds v2 wrapper kinds, callback target
  knownness, unknown callback target conservative outcome, and callback
  reentrant inout alias rejection.
- `compiler/internal/memoryfacts/from_plir.go` derives
  `function_value_contains_borrow`, `callback_arg_contains_borrow`, and
  `callback_inout_conservative`.
- `compiler/internal/memoryfacts/validate.go` and
  `tools/cmd/validate-memory-report/main.go` require parent facts for v2
  derived rows.
- `tools/cmd/validate-memory-correlation/main.go` accepts the exact v2 row set.

## Nonclaims Preserved

- No full callable ABI.
- No captured or escaping closure expansion.
- No broad noalias.
