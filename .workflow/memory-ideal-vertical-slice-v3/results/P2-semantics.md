# P2 Semantics Result

Status: accepted

## RED Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-semantics go test ./compiler/tests/semantics -run 'MemoryIdealV3|Interface|Protocol|Existential|DynamicDispatch|Borrow|NoAlias|Alias' -count=1`
  failed before fixture repair because the known/static protocol call used a
  non-borrow `BorrowView` self parameter, and the owned-return negative case
  expected older aggregate diagnostic wording.

## GREEN Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-semantics go test ./compiler/tests/semantics -run 'MemoryIdealV3|Interface|Protocol|Existential|DynamicDispatch|Borrow|NoAlias|Alias' -count=1`
  passed.
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v3-memoryfacts go test ./compiler/internal/memoryfacts -count=1`
  passed after adding the protocol/interface broad-noalias rejection case.

## Accepted Coverage

- `compiler/tests/semantics/borrow_copy_test.go` covers known/static concrete
  protocol target local use with `borrow BorrowView` self.
- `compiler/tests/semantics/borrow_copy_test.go` rejects borrowed interface /
  protocol-like aggregate return as owned.
- `compiler/tests/semantics/borrow_copy_test.go` rejects borrowed interface /
  protocol-like aggregate global storage.
- `compiler/tests/semantics/borrow_copy_test.go` keeps unknown dynamic protocol
  dispatch and runtime protocol values rejected instead of trusted.
- `compiler/internal/memoryfacts/report_test.go` rejects broad noalias through
  protocol/interface dispatch.

## Nonclaims Preserved

- No runtime protocol values.
- No full trait-object/protocol existential container semantics.
- No witness tables or dynamic dispatch ABI.
- No broad noalias.
