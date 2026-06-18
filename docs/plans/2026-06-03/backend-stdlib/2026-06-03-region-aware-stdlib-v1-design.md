# Region-Aware Stdlib v1 Design

Goal slice: P19.0 Region-aware Stdlib v1.

## Scope

P19.0 closes a bounded runtime-standard-library evidence gap. The current
request-region work already proves borrowed JSON views, borrowed HTTP request
views, region response buffers, and PostgreSQL borrowed row decode reports. The
missing part is a small reusable `stdlibrt` helper layer for the P19.0
collection targets and a machine-checkable coverage report that rejects hidden
heap claims.

This slice adds:

- region-backed `StringBuilder`, `VecBytes`, `HashMapBytes`, and `RingBuffer`
  helpers with `StorageReport` evidence;
- borrowed views and explicit copy boundaries where bytes leave caller-owned or
  region-owned storage;
- `RegionAwareStdlibCoverage()` plus a validator for every P19.0 target row;
- audit/report docs that preserve the P18.3 production non-claims.

This slice does not add a production HTTP stack, production PostgreSQL driver,
official TechEmpower result, broad generic collection API, or public safe-mode
flag.

## Implementation Shape

`compiler/internal/stdlibrt/collections.go` remains the local home because it
already defines `Region`, `BytesView`, `CollectionPlan`, `ByteBuffer`, and
`StorageReport`.

New helpers are intentionally byte-oriented and narrow:

- `StringBuilder` appends strings/bytes into caller-provided region capacity and
  returns borrowed region views.
- `VecBytes` appends single bytes and borrowed byte slices into region storage.
- `HashMapBytes` stores fixed-capacity byte keys/values in region storage and
  reports copy counts only when key/value bytes must be retained.
- `RingBuffer` provides bounded FIFO bytes in region storage, returning borrowed
  contiguous views where possible and explicit copied region snapshots when the
  readable window wraps.

The coverage report uses schema `tetra.stdlib.region_aware.v1` with rows for:
StringBuilder, Vec/Array equivalent, HashMap, JSON parser/builder, HTTP
parser/builder, PostgreSQL protocol helpers, buffers, ring buffers, borrowed
views, copy-only-when-needed reports, no-hidden-heap reports, and P18.3
production non-claim boundaries.

## Validation

Focused RED/GREEN:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/stdlibrt -run 'TestRegionAwareStdlibHelpersUseRegionReportsAndBorrowedViews|TestRegionAwareStdlibCoverageCoversP19PlanList|TestRegionAwareStdlibCoverageRejectsFakeClaims' -count=1
```

Relevant package/gate after GREEN:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/stdlibrt ./compiler/internal/jsonrt ./compiler/internal/httprt ./compiler/internal/pgrt -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics ./tools/cmd/validate-manifest -run 'FeatureRegistry|Manifest' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
graphify update .
```
