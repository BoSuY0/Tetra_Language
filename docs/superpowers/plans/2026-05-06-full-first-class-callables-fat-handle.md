# Full First-Class Callables Fat Handle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task. Use subagents only after the user explicitly authorizes delegation. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote `language.full-first-class-callables` from `post-v1` to a real v0.4.0 production feature by adding a safe escaping callable ABI beside the current bounded `fnptr` fast path.

**Architecture:** Keep the existing `fnptr` layout (`target + 8 env slots`) as the local/direct fast path. Add a fixed-width fat callable handle for escaping function values so larger environments, aggregate storage, returns, globals, and cross-module transfer do not require increasing native return-slot counts. Semantics owns escape classification and capture safety; lowering chooses `fnptr` or fat handle from that classification.

**Tech Stack:** Go compiler pipeline, Tetra frontend/semantics/lower/IR/native x64 backends, existing release validators, Graphify code graph.

---

## Current Blocker Evidence

- `compiler/features.go` records `language.full-first-class-callables` as `FeatureStatusPostV1`.
- `docs/release/v0_4_0_completion_audit.md` records the callable model as partial.
- `docs/release/v0_4_0_callable_evidence_map.md` records Callable Level 1 and constrained Level 2 as current, with full first-class callables still blocked.
- `compiler/internal/semantics/types.go` caps `FnPtrEnvSlotCount` at `8`.
- `compiler/internal/backend/x64core/emit.go` supports native returns through 10 slots and reports `unsupported return slots` beyond that.
- A previous direct raise to 9 env slots made `fnptr` 10 slots and broke aggregate return paths, so the production path must avoid unbounded slot growth.

## File Structure

- Modify `compiler/internal/semantics/types.go`: add fixed fat-handle slot count and escape classification metadata.
- Create `compiler/internal/semantics/callable_escape.go`: central escape classifier and capture policy.
- Modify `compiler/internal/semantics/checker.go`: call classifier at function-typed assignment, return, global, struct-field, enum-payload, and callback boundaries.
- Modify `compiler/internal/semantics/diagnostics.go`: add stable diagnostics for heap/global/thread escape and unsupported capture kinds.
- Modify `compiler/internal/lower/callables.go`: lower escaping callables as fixed-width handles; keep `fnptr` fast path for local snapshots.
- Modify `compiler/internal/ir/ir.go`: add callable-handle IR instructions only if existing `IRCall` plus runtime helpers cannot express the handle cleanly.
- Modify `compiler/internal/backend/x64core/emit.go` and `compiler/internal/backend/x64abi/*.go` only for tests proving no new return-slot width is needed.
- Modify `compiler/tests/callables/function_typed_callable_test.go`, `compiler/tests/semantics/closures_semantic_clauses_test.go`, and `compiler/internal/lower/callable_test.go`: add red/green coverage for escaping handles and diagnostics.
- Modify `compiler/features.go`, `compiler/tests/semantics/features_test.go`, `docs/release/v0_4_0_callable_evidence_map.md`, `docs/release/v0_4_0_completion_audit.md`, and `docs/spec/current_supported_surface.md` only after implementation gates pass.

## Task 1: Lock the Current Failing Completion Baseline

**Files:**
- Read: `compiler/features.go`
- Read: `docs/release/v0_4_0_completion_audit.md`
- Read: `docs/release/v0_4_0_callable_evidence_map.md`

- [x] **Step 1: Generate live feature and target reports**

Run:

```bash
go run ./cli/cmd/tetra features --format=json > /tmp/tetra-v04-features.json
go run ./cli/cmd/tetra targets --format=json > /tmp/tetra-v04-targets.json
```

Expected: both commands exit 0.

- [x] **Step 2: Verify the readiness blocker is real**

Run:

```bash
go run ./tools/cmd/validate-v0-4-readiness \
  --features /tmp/tetra-v04-features.json \
  --targets /tmp/tetra-v04-targets.json \
  --manifest docs/generated/manifest.json \
  --scope-decisions docs/release/v0_4_0_scope_decisions.json
```

Expected before this plan is complete: exit 1 with `feature language.full-first-class-callables status = post-v1, want current` and missing implementation/tests/docs/release gate evidence.

- [x] **Step 3: Confirm current bounded callable gates remain green**

Run:

```bash
go test ./compiler/... -run 'Closure|Callable|FunctionType|Lifetime|Ownership|Generic' -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-features --report /tmp/tetra-v04-features.json
```

Expected: all three commands exit 0.

## Task 2: Add Red Tests for Fat Handle Behavior

**Files:**
- Modify: `compiler/tests/callables/function_typed_callable_test.go`
- Modify: `compiler/tests/semantics/closures_semantic_clauses_test.go`

- [x] **Step 1: Add a runtime smoke for a 9-capture escaping callable return**

Add this test to `compiler/tests/callables/function_typed_callable_test.go` near the captured callable build smokes:

```go
func TestBuildFullCallableEscapedNineCaptureReturnSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func make() -> fn(Int) -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    return fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine

func main() -> Int:
    let cb: fn(Int) -> Int = make()
    return cb(-3)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
```

- [x] **Step 2: Add a runtime smoke for same-module global storage**

Add this test to `compiler/tests/callables/function_typed_callable_test.go`:

```go
func TestBuildFullCallableEscapedGlobalNineCaptureSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func install() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    cb = fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    return 0

func main() -> Int:
    let _: Int = install()
    return cb(-3)
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
```

- [x] **Step 3: Add a stable diagnostic for mutable capture global escape**

Add this test to `compiler/tests/semantics/closures_semantic_clauses_test.go`:

```go
func TestFullCallableGlobalEscapeRejectsMutableCaptureDiagnostic(t *testing.T) {
	src := []byte(`var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var total: Int = 1
    cb = fn(x: Int) -> Int:
        total = total + x
        return total
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected mutable capture global-escape diagnostic")
	}
	want := "global-escaped function value captures mutable local 'total'; mutable by-reference captures require a proven lifetime and synchronization model"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}
```

- [x] **Step 4: Run the red tests**

Run:

```bash
go test ./compiler -run 'TestBuildFullCallableEscapedNineCaptureReturnSmoke|TestBuildFullCallableEscapedGlobalNineCaptureSmoke|TestFullCallableGlobalEscapeRejectsMutableCaptureDiagnostic' -count=1
```

Expected before implementation: the first two tests fail with the current oversized-capture diagnostic or lowering failure; the diagnostic test fails until the new diagnostic is wired.

## Task 3: Add Callable ABI Metadata and Escape Classification

**Files:**
- Modify: `compiler/internal/semantics/types.go`
- Create: `compiler/internal/semantics/callable_escape.go`
- Modify: `compiler/internal/semantics/diagnostics.go`

- [x] **Step 1: Add fixed handle constants and escape-kind metadata**

Add to `compiler/internal/semantics/types.go` near the `FnPtrSlotCount` constants:

```go
const (
	CallableHandleSlotCount = 4
)

type CallableEscapeKind string

const (
	CallableEscapeLocalSnapshot CallableEscapeKind = "local-snapshot"
	CallableEscapeHeap          CallableEscapeKind = "heap"
	CallableEscapeGlobal        CallableEscapeKind = "global"
	CallableEscapeThread        CallableEscapeKind = "thread"
)
```

Extend `LocalInfo`, `GlobalInfo`, and `FunctionFieldInfo` with:

```go
	FunctionEscapeKind CallableEscapeKind
	FunctionHandleValue bool
```

- [x] **Step 2: Create the classifier**

Create `compiler/internal/semantics/callable_escape.go` with:

```go
package semantics

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
)

type callableEscapeBoundary string

const (
	callableBoundaryLocal       callableEscapeBoundary = "local"
	callableBoundaryReturn      callableEscapeBoundary = "return"
	callableBoundaryGlobal      callableEscapeBoundary = "global"
	callableBoundaryStructField callableEscapeBoundary = "struct-field"
	callableBoundaryEnumPayload callableEscapeBoundary = "enum-payload"
	callableBoundaryCallback    callableEscapeBoundary = "callback"
	callableBoundaryThread      callableEscapeBoundary = "thread"
)

func classifyCallableEscape(boundary callableEscapeBoundary, captures []frontend.ClosureCapture, types map[string]TypeInfo) (CallableEscapeKind, bool, error) {
	slots, err := functionCaptureSlotCount(captures, types)
	if err != nil {
		return "", false, err
	}
	if slots <= FnPtrEnvSlotCount && boundary != callableBoundaryThread {
		return CallableEscapeLocalSnapshot, false, nil
	}
	escapeKind := CallableEscapeHeap
	if boundary == callableBoundaryGlobal {
		escapeKind = CallableEscapeGlobal
	}
	if boundary == callableBoundaryThread {
		escapeKind = CallableEscapeThread
	}
	for _, capture := range captures {
		if capture.Mutable {
			return "", false, unsupportedCallableMutableCaptureEscapeError(escapeKind, capture.Name)
		}
		info, ok := types[capture.TypeName]
		if !ok {
			return "", false, fmt.Errorf("unknown capture type '%s'", capture.TypeName)
		}
		if typeContainsPtrOrResource(info, types) {
			return "", false, unsupportedCallableResourceCaptureEscapeError(capture.Name, capture.TypeName)
		}
	}
	return escapeKind, true, nil
}
```

- [x] **Step 3: Add stable diagnostics**

Add to `compiler/internal/semantics/diagnostics.go`:

```go
func unsupportedCallableMutableCaptureEscapeError(kind CallableEscapeKind, name string) error {
	return fmt.Errorf("%s-escaped function value captures mutable local '%s'; mutable by-reference captures require a proven lifetime and synchronization model", kind, name)
}

func unsupportedCallableResourceCaptureEscapeError(name, typeName string) error {
	return fmt.Errorf("escaped function value captures local '%s' of type '%s'; pointer or resource captures require an explicit ownership transfer model", name, typeName)
}
```

- [ ] **Step 4: Run compiler tests for new compile errors**

Run:

```bash
go test ./compiler/internal/semantics ./compiler -run 'Callable|Closure|FunctionType' -count=1
```

Expected after this task: the command exits 0; the classifier is compiled but behavior is unchanged until Task 4 wires call sites.

## Task 4: Wire Escape Classification Through Semantics

**Files:**
- Modify: `compiler/internal/semantics/checker.go`
- Modify: `compiler/internal/semantics/function_types.go`
- Modify: `compiler/internal/semantics/exprs.go`

Progress note:

- 2026-05-06: direct closure-literal function-typed returns with oversized
  captures now pass semantic classification and record
  `ReturnFunctionEscapeKind = heap` plus `ReturnFunctionHandleValue = true`
  on `FuncSig`.
- 2026-05-06: function-typed local bindings initialized from those returned
  callable values now preserve `FunctionEscapeKind = heap` and
  `FunctionHandleValue = true` in `LocalInfo`, giving lowering a concrete
  metadata hook.
- 2026-05-06: direct oversized closure-literal assignment into same-module
  global function-typed storage now passes semantic classification when the
  global snapshot boundary is otherwise allowed.
- Runtime smokes still fail in lowering with captured target slot mismatch,
  which is the next implementation layer before this task can be marked
  complete.

- [ ] **Step 1: Replace oversized-capture rejection at function-typed return boundaries**

Find `captureSlots > FnPtrEnvSlotCount` branches in return analysis in `compiler/internal/semantics/checker.go`. For function-typed returns, replace direct rejection with `classifyCallableEscape(callableBoundaryReturn, captures, types)`. Store `FunctionEscapeKind` and `FunctionHandleValue` in the returned `FunctionFieldInfo` or return metadata that already carries `FunctionCaptures`.

- [ ] **Step 2: Replace oversized-capture rejection at local/global/field/payload storage boundaries**

For each `unsupportedFunctionTypedStorageCaptureError` call in `checker.go`, classify with the concrete boundary:

```go
kind, handle, err := classifyCallableEscape(callableBoundaryStructField, captures, types)
if err != nil {
	return err
}
fieldInfo.FunctionEscapeKind = kind
fieldInfo.FunctionHandleValue = handle
```

Use `callableBoundaryLocal`, `callableBoundaryGlobal`, `callableBoundaryStructField`, or `callableBoundaryEnumPayload` according to the target.

- [ ] **Step 3: Preserve metadata through aliases and cross-module interface stubs**

When copying `FunctionValue`, `FunctionCaptures`, `FunctionEscapeCaptures`, `FunctionParamName`, or target sets, also copy `FunctionEscapeKind` and `FunctionHandleValue`. Apply this to local aliases, struct fields, enum payloads, function-typed returns, and generated `.t4i` metadata paths.

- [ ] **Step 4: Run red tests again**

Run:

```bash
go test ./compiler -run 'TestBuildFullCallableEscapedNineCaptureReturnSmoke|TestBuildFullCallableEscapedGlobalNineCaptureSmoke|TestFullCallableGlobalEscapeRejectsMutableCaptureDiagnostic' -count=1
```

Expected after this task: the thread diagnostic passes; runtime smokes reach lowering and fail there until handle lowering exists.

## Task 5: Lower Fat Callable Handles Without Increasing Return Slots

**Files:**
- Modify: `compiler/internal/lower/callables.go`
- Modify: `compiler/internal/lower/lower.go`
- Modify: `compiler/internal/ir/ir.go` only when existing instructions cannot express the runtime helper calls.

- [ ] **Step 1: Add runtime helper names in lowering**

Add constants in `compiler/internal/lower/callables.go`:

```go
const (
	callableAllocRuntimeSymbol  = "__tetra_callable_alloc"
	callableSlotRuntimeSymbol   = "__tetra_callable_slot"
	callableInvokeRuntimeSymbol = "__tetra_callable_invoke"
)
```

- [ ] **Step 2: Emit a fixed 4-slot handle for `FunctionHandleValue` sources**

When lowering function-typed values with `FunctionHandleValue == true`, emit:

1. target symbol address
2. environment slot count
3. runtime allocation call
4. one runtime slot write per capture
5. final 4-slot handle value

The emitted stack shape must be exactly `CallableHandleSlotCount`, independent of capture count.

- [ ] **Step 3: Dispatch through handle-aware call path**

For function-typed direct calls, struct-field calls, enum-payload calls, callback arguments, and returned callables, branch on `FunctionHandleValue`. Existing `fnptr` dispatch remains unchanged. Handle dispatch calls `__tetra_callable_invoke` with the handle slots plus user arguments and expects the declared return slot count.

- [ ] **Step 4: Add lowerer IR assertions**

Add tests to `compiler/internal/lower/callable_test.go` proving:

```go
if apply.ReturnSlots > 10 {
	t.Fatalf("fat callable lowering must not increase native return slots: got %d", apply.ReturnSlots)
}
```

and proving emitted callable handle storage uses `semantics.CallableHandleSlotCount`, not capture count.

- [ ] **Step 5: Run lowering tests**

Run:

```bash
go test ./compiler/internal/lower -run 'Callable|FunctionType' -count=1
go test ./compiler -run 'TestBuildFullCallableEscapedNineCaptureReturnSmoke|TestBuildFullCallableEscapedGlobalNineCaptureSmoke' -count=1
```

Expected after this task: lowerer tests pass; runtime smokes may fail at backend/link/runtime helper implementation.

## Task 6: Implement Native Runtime Helpers for Handles

**Files:**
- Modify: `compiler/internal/backend/x64core/emit.go`
- Modify: `compiler/internal/backend/x64obj/builder.go` only if symbol patching needs helper registration.
- Modify: runtime support files that already define `__tetra_*` helper symbols.

- [ ] **Step 1: Locate existing helper symbol registration**

Run:

```bash
rg -n "__tetra_.*alloc|__tetra_.*slot|__tetra_.*invoke|RuntimeSymbol|helper" compiler/internal compiler -g'*.go'
```

Expected: identify the exact runtime helper registry used by native codegen.

- [ ] **Step 2: Add helper implementations**

Implement:

- `__tetra_callable_alloc(target: i64, slots: i64) -> handle[4]`
- `__tetra_callable_slot(handle[4], index: i64, value: i64) -> i64`
- `__tetra_callable_invoke(handle[4], argc: i64, arg_base: ptr) -> declared return slots`

The helper storage must be deterministic for tests and must not expose raw mutable capture references across thread boundaries.

- [ ] **Step 3: Run native backend tests**

Run:

```bash
go test ./compiler/internal/backend/... ./compiler -run 'Callable|Closure|FunctionType|ABI' -count=1
```

Expected after this task: runtime handle smokes pass on linux/amd64; ABI tests still pass without raising return-slot limits.

## Task 7: Complete Cross-Module and Container Matrix

**Files:**
- Modify: `compiler/tests/callables/function_typed_callable_test.go`
- Modify: `compiler/internal/lower/callable_test.go`
- Modify: `compiler/interface_test.go`
- Modify: `.t4i` generation code discovered by `rg -n "FunctionCaptures|FunctionEscape" compiler/internal compiler -g'*.go'`

- [x] **Step 1: Add cross-module returned handle smoke**

Add a `buildAndRunFiles` smoke where module `callbacks` returns a 9-capture closure as `fn(Int) -> Int`; app imports it and calls through local, struct field, enum payload, callback argument, and return alias.

- [x] **Step 2: Add interface metadata assertions**

Assert generated `.t4i` preserves:

- function signature
- target symbol identity
- capture slot count
- `FunctionHandleValue`
- `FunctionEscapeKind`

- [x] **Step 3: Add mutable/resource rejection matrix**

Add diagnostics for:

- mutable local capture escaping to thread
- pointer field capture escaping to heap/global/thread
- resource capture escaping to heap/global/thread
- imported mutable function-typed global requiring global-data ABI
- unsupported generic callable movement where type parameters are not fully inferred

- [x] **Step 4: Run matrix tests**

Run:

```bash
go test ./compiler -run 'FullCallable|Callable|Closure|FunctionType|Interface' -count=1
go test ./compiler/internal/lower -run 'Callable|FunctionType' -count=1
```

Expected: all new matrix tests pass.

## Task 8: Promote Feature Registry and Release Evidence Only After Gates

**Files:**
- Modify: `compiler/features.go`
- Modify: `compiler/tests/semantics/features_test.go`
- Modify: `docs/release/v0_4_0_callable_evidence_map.md`
- Modify: `docs/release/v0_4_0_completion_audit.md`
- Modify: `docs/spec/current_supported_surface.md`
- Modify: `docs/spec/v1_feature_status.md`

- [x] **Step 1: Run full callable gate before docs promotion**

Run:

```bash
go test ./compiler/... -run 'Closure|Callable|FunctionType|Lifetime|Ownership|Generic' -count=1
go test ./compiler/internal/backend/... -run 'Callable|Closure|FunctionType|ABI' -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-features --report /tmp/tetra-v04-features.json
```

Expected: all commands exit 0.

- [x] **Step 2: Promote registry after implementation evidence exists**

Change `language.full-first-class-callables` in `compiler/features.go` from `FeatureStatusPostV1` to `FeatureStatusCurrent` and set `Since: "v0.4.0"`. Update `compiler/tests/semantics/features_test.go` to require current status and scope text covering fat callable handle, escape classifier, capture matrix, and stable diagnostics.

- [x] **Step 3: Update release docs**

Update callable docs to state:

- `fnptr` remains the bounded fast path
- fat callable handles are the production escaping ABI
- mutable/resource/thread escapes have explicit diagnostics
- release gates include full callable matrix tests

- [x] **Step 4: Run readiness gate**

Run:

```bash
go run ./cli/cmd/tetra features --format=json > /tmp/tetra-v04-features.json
go run ./cli/cmd/tetra targets --format=json > /tmp/tetra-v04-targets.json
go run ./tools/cmd/validate-v0-4-readiness \
  --features /tmp/tetra-v04-features.json \
  --targets /tmp/tetra-v04-targets.json \
  --manifest docs/generated/manifest.json \
  --scope-decisions docs/release/v0_4_0_scope_decisions.json
```

Expected for callable slice: no `language.full-first-class-callables` status or evidence failure remains. Other v0.4.0 blockers may still fail until their features are implemented.

## Task 9: Final Verification and Graphify Update

**Files:**
- Update: `graphify-out/graph.json`
- Update: `graphify-out/GRAPH_REPORT.md`

- [x] **Step 1: Run final callable and docs gates**

Run:

```bash
go test ./compiler/... -run 'Closure|Callable|FunctionType|Lifetime|Ownership|Generic' -count=1
go test ./compiler/internal/backend/... -run 'Callable|Closure|FunctionType|ABI' -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
go run ./tools/cmd/validate-features --report /tmp/tetra-v04-features.json
git diff --check -- compiler/internal/semantics/types.go compiler/internal/semantics/callable_escape.go compiler/internal/semantics/checker.go compiler/internal/semantics/diagnostics.go compiler/internal/lower/callables.go compiler/tests/callables/function_typed_callable_test.go compiler/tests/semantics/closures_semantic_clauses_test.go compiler/internal/lower/callable_test.go compiler/features.go compiler/tests/semantics/features_test.go docs/release/v0_4_0_callable_evidence_map.md docs/release/v0_4_0_completion_audit.md docs/spec/current_supported_surface.md docs/spec/v1_feature_status.md
```

Expected: all commands exit 0.

- [x] **Step 2: Update Graphify after code changes**

Run:

```bash
graphify update .
```

Expected: graph rebuild exits 0 and updates `graphify-out/graph.json` plus `graphify-out/GRAPH_REPORT.md`.

- [x] **Step 3: Record remaining non-callable blockers**

Run:

```bash
go run ./tools/cmd/validate-v0-4-readiness \
  --features /tmp/tetra-v04-features.json \
  --targets /tmp/tetra-v04-targets.json \
  --manifest docs/generated/manifest.json \
  --scope-decisions docs/release/v0_4_0_scope_decisions.json
```

Expected after callable completion: any remaining failures name non-callable blockers such as distributed Eco, distributed actors, native UI runtime, or cross-host runtime evidence. Do not mark the active goal complete unless this command and the objective-specific audit prove every requirement in the user objective is covered.

## 2026-05-06 Narrow Slice Progress

- [x] Direct nine-capture local callable handle smoke is implemented and covered by `TestBuildFullCallableLocalNineCaptureSmoke`.
- [x] Direct nine-capture mutable local callable reassignment handle smoke is implemented and covered by `TestBuildFullCallableMutableLocalReassignNineCaptureSmoke`.
- [x] Direct nine-capture function-typed return handle smoke is implemented and covered by `TestBuildFullCallableEscapedNineCaptureReturnSmoke`.
- [x] Direct nine-capture same-module mutable global snapshot handle smoke is implemented and covered by `TestBuildFullCallableEscapedGlobalNineCaptureSmoke`.
- [x] Direct nine-capture immutable local struct-field initializer/direct-call handle smoke is implemented and covered by `TestBuildFullCallableStructFieldNineCaptureSmoke`, `TestFullCallableStructFieldNineCapturePassesSemanticClassification`, and `TestFullCallableStructFieldNineCaptureLowersHandleEnvironment`.
- [x] Direct nine-capture mutable local struct-field reassignment/direct-call handle smoke is implemented and covered by `TestBuildFullCallableStructFieldReassignNineCaptureSmoke`.
- [x] Direct nine-capture local enum-payload initializer/reassignment and pattern-bound direct-call handle smoke is implemented and covered by `TestBuildFullCallableEnumPayloadNineCaptureSmoke`, `TestBuildFullCallableEnumPayloadReassignNineCaptureSmoke`, `TestFullCallableEnumPayloadNineCapturePassesSemanticClassification`, and `TestFullCallableEnumPayloadNineCaptureLowersHandleEnvironment`.
- [x] Direct nine-capture closure-literal synchronous callback argument handle smoke is implemented and covered by `TestBuildFullCallableCallbackArgumentNineCaptureSmoke`.
- [x] Direct nine-capture handle-backed function-typed local synchronous callback argument smoke is implemented and covered by `TestBuildFullCallableLocalCallbackArgumentNineCaptureSmoke`.
- [x] Cross-module returned nine-capture callable handle matrix smoke is implemented and covered by `TestBuildFullCallableCrossModuleReturnedNineCaptureMatrixSmoke`.
- [x] Generated `.t4i` direct returned function-value stubs preserve nine-capture heap handle metadata and are covered by `TestGenerateInterfaceFromSourcePreservesReturnedFunctionHandleMetadata`.
- [x] Mutable/resource escape diagnostics are covered by source-level heap/global resource and heap mutable tests in `TestBuildFunctionTypedCallableMVPRejectsUnsupportedForms`, thread-boundary classifier tests in `TestClassifyCallableEscapeRejectsMutableCaptureAcrossThreadBoundary` and `TestClassifyCallableEscapeRejectsResourceCaptureAcrossThreadBoundary`, existing imported mutable-global ABI diagnostics, and existing unsupported generic callable movement diagnostics.
- [x] Twelve-capture callable aliases now move through function-typed returns, same-module mutable global snapshots, and synchronous callback arguments, covered by `TestBuildFullCallableReturnAliasTwelveCaptureSmoke`, `TestBuildFullCallableGlobalAliasTwelveCaptureSmoke`, and `TestBuildFullCallableCallbackAliasTwelveCaptureSmoke`.
- [x] Full first-class callables are promoted: `language.full-first-class-callables` is current since `v0.4.0`; readiness no longer reports callable-specific status or evidence blockers. Remaining readiness failures are non-callable or cross-host evidence blockers.

## Self-Review Notes

- The plan directly maps the user objective to ABI, capture matrix, escape control, storage/return/callback paths, diagnostics, docs, registry, readiness gate, and test evidence.
- The first implementation task is a red test task; implementation follows TDD.
- The plan avoids raising `FnPtrEnvSlotCount`, preserving the verified fast path and avoiding the aggregate return-slot failure already observed.
- The plan does not authorize subagents; it names `executing-plans` as the execution skill because the user has not requested delegation.
