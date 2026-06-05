# P2 Semantics Result

Status: accepted

## RED Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-red-semantics go test ./compiler/tests/semantics -run 'MemoryIdealV2|Callback|FunctionTyped|Borrow|Inout|Alias' -count=1`
  failed because a function-typed local with signature
  `fn(borrow []u8) -> borrow []u8` could return a borrowed view as owned
  without a diagnostic.
- The same RED run also showed the capturing callback case was already
  conservative, but with the existing diagnostic
  `function-typed parameter 'cb' cannot be stored in global function-typed value
  'saved'`.

## GREEN Evidence

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-vslice-v2-semantics go test ./compiler/tests/semantics -run 'MemoryIdealV2|Callback|FunctionTyped|Borrow|Inout|Alias' -count=1`
  passed.

## Accepted Changes

- `compiler/internal/semantics/exprs.go` now propagates a narrow borrowed return
  region for function-typed local/field calls only when the function type
  return ownership is `borrow` and the call has visible borrowed argument
  provenance.
- `compiler/tests/semantics/borrow_copy_test.go` adds `MemoryIdealV2`
  positive/negative coverage for known local callback use, function-typed field
  use, `.copy()` before callback escape, non-borrow callback parameter, owned
  return/global escape through callback, consume/inout rejection, inout aliasing,
  unknown callback target conservatism, and capturing callback conservative
  rejection.

## Nonclaims Preserved

- No captured callback support was added.
- No unknown callback target trust was added.
- No async/task/actor, raw pointer, interface/protocol, or target parity scope
  was added.
