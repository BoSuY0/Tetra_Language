# P2-semantics Result

Status: accepted

## Evidence

- Added explicit `MemoryIdealV1` semantics tests for local enum payload use,
  local generic wrapper use, enum payload global storage rejection, generic
  wrapper global storage rejection, and `.copy()` owned escape.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-red-semantics go test ./compiler/tests/semantics -run 'MemoryIdealV1|BorrowedAggregate|Borrow' -count=1` passed before implementation because the current dirty checker already handled these paths.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-green-semantics go test ./compiler/tests/semantics -run 'MemoryIdealV1|BorrowedAggregate|Borrow' -count=1` passed after the non-semantics implementation.

## Changes Accepted

- No checker code change was required for v1 semantics. Existing
  `checkBorrowedAggregateEscape` already inspects enum payloads and
  monomorphized generic struct wrappers.
