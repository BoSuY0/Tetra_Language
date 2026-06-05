# P2-semantics

Packet ID: P2-semantics

Objective: Add and prove compiler semantics checks for enum payload and
monomorphized generic wrapper borrowed view escape closure.

Context: v0 covers struct and optional borrow carriers. v1 must reject borrowed
enum/generic wrapper returned as owned, stored globally, mixed branch owners,
and `unsafe_unknown`; it must allow local use and `.copy()` owned escape.

Files / sources:

- `compiler/internal/semantics`
- `compiler/tests/semantics/borrow_copy_test.go`
- nearby borrow/lifetime diagnostics tests

Ownership: semantics tests and narrow checker implementation.

Do: follow RED -> GREEN for each behavior cluster.

Do not: broaden to interfaces, function-typed values, callbacks, async,
actor/task, raw pointer semantics, target parity, or broad noalias.

Expected output: tests, implementation, and packet result note.

Verification:

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v1-semantics go test ./compiler/tests/semantics -run 'MemoryIdealV1|BorrowedAggregate|Borrow' -count=1`
