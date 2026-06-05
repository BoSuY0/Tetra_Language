# P1-tests-model

Packet ID: P1-tests-model

Objective: Add and prove memoryfacts/report validator plus MiniMemoryModel v1
cases.

Context: v0 already validates aggregate/optional borrow carrier projections.
v1 adds enum payload and monomorphized generic wrapper projections.

Files / sources:

- `compiler/internal/memoryfacts`
- `compiler/internal/memorymodel`
- `tools/cmd/validate-memory-report`

Ownership: tests and implementation in those packages.

Do: follow RED -> GREEN for v1 facts, projections, and model cases.

Do not: change compiler semantics behavior in this packet.

Expected output: tests, implementation, and packet result note.

Verification:

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-memoryfacts go test ./compiler/internal/memoryfacts -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-mini go test ./compiler/internal/memorymodel -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation -count=1`
