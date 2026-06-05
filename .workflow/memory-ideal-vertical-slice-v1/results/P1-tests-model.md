# P1-tests-model Result

Status: accepted

## RED Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-red-model go test ./compiler/internal/memorymodel -count=1` failed because `WrapperEnumPayload` and `WrapperGenericWrapper` were undefined.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-red-memoryfacts go test ./compiler/internal/memoryfacts -count=1` failed because v1 PLIR projections were missing and v1 derived rows did not require `parent_fact_id`.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-red-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1` failed because the CLI validator did not require v1 parent facts and correlation validation was hard-coded to v0 rows.

## GREEN Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-green-model go test ./compiler/internal/memorymodel -count=1` passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-green-memoryfacts go test ./compiler/internal/memoryfacts -count=1` passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-green-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1` passed.

## Changes Accepted

- Added MiniMemoryModel wrapper kinds for enum payload and generic wrapper.
- Added v1 memoryfacts projection claims and validator parent requirements.
- Updated the correlation validator to preserve v0 validation and accept exactly
  the v1 row set when v1 IDs are present.
