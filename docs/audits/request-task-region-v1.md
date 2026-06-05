# Request / Task Region v1 Closure

Goal slice: P15.1 Request / Task Region v1.

Baseline: `tetra.truthful-performance-core.baseline.20260602.v1`.

Status: complete for slice after focused implementation and verification.

## Scope

This slice defines explicit reusable request and task region entry scopes for
runtime-side HTTP/JSON and task workloads. It connects the existing
region-aware HTTP request views, JSON view parsing, response buffering, and task
runtime model to an entry-scope API that injects a region and resets it at the
end of the handler or task.

The slice is intentionally evidence-bounded: it proves local runtime entry
behavior and near-zero heap allocation for an ordinary HTTP/JSON path. It does
not claim that every production `webrt.Server` route has been migrated to view
routers, nor that P15.2 per-core allocator work is complete.

## Implemented Rules

| Rule | Evidence |
|---|---|
| Request region lifetime is explicit and resettable. | `RequestRegionScope`, `RequestRegionReport.Lifetime == "request"`, `TestRequestRegionScopeInjectsRegionForHTTPJSONAndResetsAfterWrite` |
| Task region lifetime is explicit and resettable. | `TaskRegionScope`, `TaskRegionReport.Lifetime`, `TestTaskRegionScopeInjectsRegionAndResetsAfterTask` |
| Request entry injects the region into handler code. | `RequestRegionHandler func(RequestView, *stdlibrt.Region)`, focused request test |
| Task entry injects the region into task code. | `TaskRegionScope.Run`, focused task test |
| HTTP request and response buffers are planned through request region storage. | `ParseRequestViewInRegion`, `AppendResponseWithReport`, focused request test |
| JSON escaped string temporaries decode into the request region instead of hidden heap. | `TestParseValueViewDecodesEscapedStringIntoRegionWithoutHeap` |
| Region reset happens after request write and task completion. | `BytesUsedBeforeReset` plus `RegionUsed() == 0` assertions |
| Ordinary local HTTP/JSON path has zero heap allocations in the focused benchmark-style test. | `testing.AllocsPerRun(1000) == 0` in request-region test |

## Code Changes

- `compiler/internal/stdlibrt/collections.go` adds `Region.Reset()`.
- `compiler/internal/jsonrt/view.go` decodes escaped strings directly into a
  provided region when `ParseViewOptions.Region` is present.
- `compiler/internal/httprt/request_region.go` adds reusable request-region
  entry scope with handler injection, response writer handoff, reporting, and
  reset.
- `compiler/internal/httprt/request_view.go` reports region response-buffer
  overflow as a heap-allocation signal instead of silently claiming region-only
  storage.
- `compiler/internal/parallelrt/task_region.go` adds reusable task-region
  entry scope with task injection, reporting, and reset.
- `compiler/features.go`, `compiler/tests/semantics/features_test.go`, and
  `docs/generated/manifest.json` add this audit to the verified-track evidence
  chain.
- `tools/cmd/validate-manifest/main.go` and its tests accept the full current
  feature-registry lifecycle vocabulary, including `unsupported`,
  `release_candidate`, and `legacy_compatibility`, so regenerated manifests do
  not fail on existing Surface boundary entries.

## Graphify Navigation Evidence

Graphify MCP was used before concrete file inspection:

```text
query_graph: P15.1 Request Task Region v1 request region lifetime task region lifetime region injection in request task entry functions JSON HTTP buffers allocation planner reset at handler task exit near-zero heap ordinary requests
get_neighbors: ParseRequest()
get_neighbors: FromPLIRWithOptions()
get_neighbors: ValidateAllocationLowering()
shortest_path: ParseRequest() -> FromPLIRWithOptions()
shortest_path: checkTypedTaskBuiltin() -> FromPLIRWithOptions()
get_neighbors: ParseRequestView()
get_neighbors: AppendResponseWithReport()
get_neighbors: NewRegion()
get_neighbors: NewSchedulerModel()
query_graph: typed task builtin wrappers task entry lowering checkTypedTaskBuiltin collectTypedTaskWrappers spawn task runtime allocation region lifetime
```

The graph identified `compiler/internal/httprt/request_view.go`,
`compiler/internal/jsonrt/view.go`, `compiler/internal/stdlibrt/collections.go`,
`compiler/internal/parallelrt/scheduler_model.go`, `compiler/internal/webrt`,
and allocation planner/report boundaries as the relevant P15.1 surface.

## Verification Evidence

RED evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/stdlibrt ./compiler/internal/jsonrt ./compiler/internal/httprt ./compiler/internal/parallelrt -run 'RegionResetRestoresRequestLifetimeCapacity|ParseValueViewDecodesEscapedStringIntoRegionWithoutHeap|RequestRegionScopeInjectsRegionForHTTPJSONAndResetsAfterWrite|TaskRegionScopeInjectsRegionAndResetsAfterTask' -count=1
```

Initial result: failed at compile time for the right reason: `Region.Reset`,
`NewRequestRegionScope`, `RequestRegionOptions`, `RequestRegionReport`,
`NewTaskRegionScope`, `TaskRegionOptions`, and `TaskRegionReport` did not
exist.

Focused GREEN evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/stdlibrt ./compiler/internal/jsonrt ./compiler/internal/httprt ./compiler/internal/parallelrt -run 'RegionResetRestoresRequestLifetimeCapacity|ParseValueViewDecodesEscapedStringIntoRegionWithoutHeap|RequestRegionScopeInjectsRegionForHTTPJSONAndResetsAfterWrite|TaskRegionScopeInjectsRegionAndResetsAfterTask' -count=1
```

Result: pass.

Relevant package evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/stdlibrt ./compiler/internal/jsonrt ./compiler/internal/httprt ./compiler/internal/parallelrt -count=1
```

Result: pass.

Relevant gate evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/stdlibrt ./compiler/internal/jsonrt ./compiler/internal/httprt ./compiler/internal/parallelrt ./compiler/tests/semantics ./tools/cmd/validate-manifest -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/webrt -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler -run 'Manifest|Allocation|Reports|Explain' -count=1
```

Result: pass.

Final hygiene evidence:

```bash
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json
git diff --check
rg -n '[[:blank:]]$' GOAL.md PLAN.md ATTEMPTS.md NOTES.md CONTROL.md reports/request-task-region-v1/closure.md docs/audits/request-task-region-v1.md compiler/internal/stdlibrt/collections.go compiler/internal/stdlibrt/collections_test.go compiler/internal/jsonrt/view.go compiler/internal/jsonrt/view_test.go compiler/internal/httprt/request_view.go compiler/internal/httprt/request_view_test.go compiler/internal/httprt/request_region.go compiler/internal/parallelrt/scheduler_model_test.go compiler/internal/parallelrt/task_region.go compiler/features.go compiler/tests/semantics/features_test.go tools/cmd/validate-manifest/main.go tools/cmd/validate-manifest/main_test.go docs/generated/manifest.json
rg -n 'source_plan: /home/tetra/Downloads/tetra_surface_release|Active slice: Section|Surface Release Promotion v1|tetra_surface_release_promotion_v1_full_plan' GOAL.md PLAN.md ATTEMPTS.md NOTES.md CONTROL.md
graphify update .
```

Result: pass. Whitespace scan found no matches. Drift scan found only explicit
guard references in `GOAL.md` and `CONTROL.md`. Graphify rebuilt
`18964 nodes, 60832 edges, 1077 communities`.

## Non-Claims

- P15.1 does not implement P15.2 thread-local or per-core allocators.
- P15.1 does not claim every production web server route has been migrated to
  request-view routing.
- P15.1 does not claim official TechEmpower performance results.
- P15.1 does not relax safe lifetime or provenance rules.
- P15.1 does not allow region-backed request/task data to escape beyond the
  entry scope.
