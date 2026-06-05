# P2 Semantics

Objective: prove v2 callback/function-typed behavior through focused semantics
tests.

Ownership:
- `compiler/internal/semantics`
- `compiler/tests/semantics/borrow_copy_test.go`

Do:
- Add positive tests for known local callback, known function-typed field call,
  and `.copy()` before callback escape.
- Add negative tests for non-borrow callback parameter, callback return/global
  escape, consumed callback argument, `inout`, inout alias conservatism,
  unknown target, capturing callback, and broad noalias rejection.
- Prefer regression tests for existing conservative behavior before changing
  semantics.

Do not:
- Add captured closure support.
- Add escaping closure support.
- Add full callable ABI.

Expected output: `.workflow/memory-ideal-vertical-slice-v2/results/P2-semantics.md`.

Verification:
- `go test ./compiler/tests/semantics -run 'MemoryIdealV2|Callback|FunctionTyped|Borrow|Inout|Alias' -count=1`
