# Region-Aware Stdlib v1 Closure

Goal slice: P19.0 Region-aware Stdlib v1.

Baseline: `tetra.truthful-performance-core.baseline.20260602.v1`.

Status: complete for the bounded P19.0 slice after focused implementation and
verification.

## Scope

P19.0 adds a region-aware runtime standard-library helper layer and a
machine-checkable coverage report for the master-plan target list:
`StringBuilder`, `Vec<T>`/array equivalent, `HashMap<K,V>`, JSON
parser/builder, HTTP parser/builder, PostgreSQL protocol helpers, buffers, and
ring buffers.

The implemented collection helpers are deliberately narrow byte-oriented
runtime helpers. They prove region-first storage and report-visible copy/heap
boundaries without promoting broad public generic collection APIs.

## Implemented Rows

| Row | Status | Evidence |
|---|---|---|
| StringBuilder | `implemented_narrow` | `NewStringBuilder`, `StringBuilder.View`, `StorageReport.HiddenHeap=false` |
| Vec/Array equivalent | `implemented_narrow` | `NewVecBytes`, `AppendBorrowed`, borrowed `BytesView` |
| HashMap | `implemented_narrow` | `NewHashMapBytes`, fixed-capacity open addressing, retained key/value copy reports |
| JSON parser/builder | `implemented_narrow` | `ParseValueView`, `AppendValue`, `AppendString`, borrowed/copy JSON reports |
| HTTP parser/builder | `implemented_narrow` | `ParseRequestViewInRegion`, `AppendResponseWithReport`, request/response reports |
| PostgreSQL protocol helpers | `implemented_narrow` | `AppendBindFormat`, `DecodeDataRowBorrowed`, `RowDecodeReport` |
| Buffers | `implemented_narrow` | `NewByteBuffer`, `ByteBuffer.View`, `StorageReport` |
| Ring buffers | `implemented_narrow` | `NewRingBuffer`, contiguous borrowed views, wrapped copied snapshots |
| Borrowed views | `evidence_only` | `BytesView` storage/provenance metadata |
| Copy reports | `evidence_only` | `CopyOperations`, `BytesCopied`, JSON copied strings, wrapped ring snapshots |
| No hidden heap reports | `evidence_only` | `HiddenHeap=false`, `HeapAllocations=0`, report-visible heap fallback |
| Production boundaries | `boundary_documented` | validator rejects fake production web/db/result claims |

## Code Changes

- `compiler/internal/stdlibrt/collections.go` adds byte-oriented
  `StringBuilder`, `VecBytes`, `HashMapBytes`, and `RingBuffer` helpers.
- `StorageReport` now records `CopyOperations`, `BytesCopied`, and
  `BorrowedViews`.
- `compiler/internal/stdlibrt/region_aware_coverage.go` adds
  `RegionAwareStdlibCoverage()` and `ValidateRegionAwareStdlibCoverage()`.
- Focused tests cover real helper behavior, target-row coverage, and negative
  fake-claim validation.

## Graphify Navigation Evidence

Graphify MCP was used before concrete file inspection:

```text
query_graph: P19.0 Region-aware Stdlib v1 StringBuilder Vec Array HashMap JSON parser builder HTTP parser builder PostgreSQL protocol helpers buffers ring buffers region-first borrowed views copy only when needed no hidden heap report Tetra_Language
query_graph: compiler internal stdlibrt jsonrt httprt pgrt request region StringBuilder Vec HashMap ring buffer allocation report borrowed views copy reports hidden heap
get_neighbors: collections.go
get_neighbors: request_view.go
get_neighbors: AppendBindFormat()
get_neighbors: appendTypedPayload()
get_neighbors: ParseRequestViewInRegion()
shortest_path: collections.go -> ParseRequestView
shortest_path: collections.go -> AppendBindFormat
```

The graph identified `compiler/internal/stdlibrt/collections.go`,
`compiler/internal/jsonrt/view.go`, `compiler/internal/httprt/request_view.go`,
`compiler/internal/httprt/request_region.go`, and
`compiler/internal/pgrt/wire.go` as the concrete P19.0 surface.

## Verification Evidence

RED evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/stdlibrt -run 'TestRegionAwareStdlibHelpersUseRegionReportsAndBorrowedViews|TestRegionAwareStdlibCoverageCoversP19PlanList|TestRegionAwareStdlibCoverageRejectsFakeClaims' -count=1
```

Initial result: failed at compile time for the intended reason:
`NewStringBuilder`, `NewVecBytes`, `NewHashMapBytes`,
`HashMapBytesOptions`, `NewRingBuffer`, `RegionAwareStdlibCoverage`, and
`ValidateRegionAwareStdlibCoverage` did not exist.

Focused GREEN evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/stdlibrt -run 'TestRegionAwareStdlibHelpersUseRegionReportsAndBorrowedViews|TestRegionAwareStdlibCoverageCoversP19PlanList|TestRegionAwareStdlibCoverageRejectsFakeClaims' -count=1
```

Result: pass.

Relevant runtime package evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/stdlibrt ./compiler/internal/jsonrt ./compiler/internal/httprt ./compiler/internal/pgrt -count=1
```

Result: pass.

## Non-Claims

- P19.0 does not claim a full production web stack.
- P19.0 does not claim an official TechEmpower result.
- P19.0 does not claim a production PostgreSQL stack or production database
  pool.
- P19.0 does not add a broad public generic `Vec<T>` or `HashMap<K,V>` API.
- P19.0 does not change safe-program semantics or add a public runtime mode.
