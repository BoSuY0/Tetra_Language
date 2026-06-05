# P1 MemoryFacts And MiniMemoryModel Result

Status: accepted

## RED Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-red-mini go test ./compiler/internal/memorymodel -count=1`
  failed because v3 model symbols did not exist:
  `WrapperInterfaceValue`, `WrapperProtocolDispatch`, `DispatchTargetKnown`,
  `OutcomeConservativeUnknownProtocolDispatch`,
  `EventProtocolDispatchCall`, and `OutcomeInvalidProtocolDispatchNoAlias`.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-red-memoryfacts go test ./compiler/internal/memoryfacts -count=1`
  failed because interface/protocol borrow contexts were still projected as
  generic aggregate/noalias facts, and v3 rows were not parent-validated.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-red-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1`
  failed because report validation did not require v3 parent facts and
  correlation validation only recognized v0/v1/v2 row sets.

## GREEN Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-mini go test ./compiler/internal/memorymodel -count=1`
  passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-memoryfacts go test ./compiler/internal/memoryfacts -count=1`
  passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1`
  passed.

## Accepted Changes

- `compiler/internal/memorymodel/mini.go` adds v3 wrapper kinds, dispatch target
  knownness, unknown protocol dispatch conservative outcome, and protocol
  dispatch noalias rejection.
- `compiler/internal/memoryfacts/from_plir.go` derives
  `interface_value_contains_borrow`,
  `protocol_dispatch_borrow_conservative`, and
  `protocol_dispatch_noalias_conservative`.
- `compiler/internal/memoryfacts/validate.go` and
  `tools/cmd/validate-memory-report/main.go` require parent facts for v3
  derived rows.
- `tools/cmd/validate-memory-correlation/main.go` accepts the exact v3 row set.

## Nonclaims Preserved

- No full trait-object/protocol existential runtime.
- No full dynamic dispatch or witness table support.
- No trusted unknown dynamic dispatch borrow facts.
- No broad noalias.
