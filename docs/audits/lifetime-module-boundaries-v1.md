# Lifetime Module Boundaries v1 Closure

Goal slice: P14.3 Lifetime Across Module Boundaries.

Baseline: `tetra.truthful-performance-core.baseline.20260602.v1`.

Status: complete for slice after focused implementation and verification.

## Scope

This slice makes borrowed-return lifetime/provenance contracts visible and
hash-bound in generated `.t4i` interface artifacts. It does not introduce named
lifetimes or an interprocedural borrow checker. The contract is intentionally
narrow: a public `-> borrow` function whose return can be tied to a borrowed
parameter now emits an interface lifetime fact, and that fact participates in
the `.t4i` hash validated by imports.

## Implemented Rules

| Rule | Evidence |
|---|---|
| Generated interfaces preserve public borrowed-return contracts as explicit lifetime/provenance metadata. | `compiler/interface.go`, `TestGenerateInterfaceFromSourcePreservesBorrowedReturnContract` |
| The metadata records the borrowed return source parameter, parameter provenance, and call lifetime. | `// tetra-interface-lifetime: return=borrow source=<param> provenance=param lifetime=call` |
| The `.t4i` fingerprint changes when the borrowed-return source changes. | `TestInterfaceFingerprintTracksBorrowedReturnLifetimeSource` |
| Tampered borrowed-return lifetime metadata is rejected by `.t4i` hash validation. | `TestInterfaceFingerprintRejectsTamperedBorrowedReturnLifetimeMetadata` |
| Interface-only imports reject stale or mismatched borrowed-return metadata. | `TestBuildInterfaceOnlyModeRejectsTamperedBorrowedReturnLifetimeMetadata` |
| Cross-module borrowed-return calls continue to type-check from generated interfaces. | `TestGenerateInterfaceFromSourcePreservesBorrowedReturnContract` |

## Code Changes

- `compiler/interface.go` now detects borrowed-return source parameters from
  direct identifiers, field paths, semantic method calls, core slice/String
  view calls, and raw dotted method calls such as `a.borrow()`.
- `compiler/interface.go` now emits a `tetra-interface-lifetime` comment in
  generated function stubs for supported borrowed returns.
- `compiler/interface.go` now adds matching lifetime/provenance facts to the
  hash-only public surface, so the `.t4i` hash changes when the return source
  changes even if future stubs normalize bodies.
- `compiler/tests/semantics/interface_test.go` and `compiler/compiler_test.go`
  add focused cross-module and stale metadata coverage.

## Graphify Navigation Evidence

Graphify MCP was used before concrete file inspection:

```text
query_graph: P14.3 Lifetime Across Module Boundaries interface metadata borrowed return provenance hash validation imports tests GenerateInterfaceFromSource InterfaceFingerprintFromT4I
get_neighbors: GenerateInterfaceFromSource()
shortest_path: GenerateInterfaceFromSource() -> TestGenerateInterfaceFromSourcePreservesBorrowedReturnContract()
get_neighbors: InterfaceFingerprintFromT4I()
```

The graph identified `compiler/interface.go`, generated-interface tests,
`InterfaceFingerprintFromT4I()`, `ValidateHash()`, and interface-only compiler
tests as the relevant boundary.

## Verification Evidence

RED evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics -run 'BorrowedReturnLifetime|PreservesBorrowedReturnContract|InterfaceFingerprintTracksBorrowedReturnLifetimeSource|InterfaceFingerprintRejectsTamperedBorrowedReturnLifetimeMetadata' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'BuildInterfaceOnlyModeRejectsTamperedBorrowedReturnLifetimeMetadata' -count=1
```

Initial result: failed because generated `.t4i` stubs did not emit
`tetra-interface-lifetime` metadata, the public interface hash did not track the
borrowed-return source, and tampered fixtures could not find metadata to alter.

Focused GREEN evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics -run 'PreservesBorrowedReturnContract|InterfaceFingerprintTracksBorrowedReturnLifetimeSource|InterfaceFingerprintRejectsTamperedBorrowedReturnLifetimeMetadata' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'BuildInterfaceOnlyModeRejectsTamperedBorrowedReturnLifetimeMetadata' -count=1
```

Result: pass.

Relevant package evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics -run 'Interface|BorrowedReturn' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'InterfaceOnlyMode.*(Borrowed|Resource|Region|Tampered)|BuildRejectsInterface|BuildInterfaceOnlyModeAllowsT4I|BuildInterfaceOnlyModeRejectsTamperedBorrowedReturnLifetimeMetadata' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/t4iface -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/module -run 'T4Interface|Tampered|Hash' -count=1
```

Result: pass.

Final hygiene evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/tests/semantics ./compiler/internal/module ./compiler -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
git diff --check
graphify update .
```

Result: pass. Graphify rebuilt `18921 nodes, 60734 edges, 1095 communities`.

Additional final checks:

```bash
rg -n '[[:blank:]]$' GOAL.md PLAN.md ATTEMPTS.md NOTES.md CONTROL.md reports/lifetime-module-boundaries-v1/closure.md docs/audits/lifetime-module-boundaries-v1.md compiler/interface.go compiler/tests/semantics/interface_test.go compiler/compiler_test.go docs/generated/manifest.json
rg -n 'tetra_surface_release_promotion_v1_full_plan|source_plan: /home/tetra/Downloads/tetra_surface_release|Active slice: Section|Surface Release Promotion v1' GOAL.md PLAN.md ATTEMPTS.md NOTES.md CONTROL.md
```

Result: pass. The whitespace scan found no trailing whitespace. The drift scan
found only explicit drift-guard references in `GOAL.md` and `CONTROL.md`. After
the Graphify update, sidecars were again overwritten to the stale Surface
Release Promotion goal; they were recreated to Ideal with P14.3 complete and
P15.0 active before continuing.

## Non-Claims

- P14.3 does not implement named lifetime parameters.
- P14.3 does not implement a full interprocedural borrow graph.
- P14.3 does not allow safe code to forge provenance, lifetime, or interface
  metadata; tampering is rejected by `.t4i` hash validation.
- P14.3 does not remove runtime checks or change safe semantics through
  reports or interface-only mode.
