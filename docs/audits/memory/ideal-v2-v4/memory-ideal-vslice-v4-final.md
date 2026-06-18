# Memory Ideal Vertical Slice v4 Final Audit

Status: validated_narrow

This audit closes the async/task/actor borrow-boundary slice for the current supported surface. It
extends the v0/v1/v2/v3 memory correlation pattern without implementing a full async lifetime
system, full production actor runtime, structured concurrency, cancellation model, distributed actor
memory model, target parity, broad noalias, or performance claims. `MemoryFactGraph` remains the
truth source; reports remain projections.

## Row Classifications

| requirement_id | classification   | evidence                                                                                                                                                                                                                                                                                                                                                                                                                                               | boundary                                                                                                                                                                                                                                    |
| -------------- | ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| MEM-BORROW-008 | conservative     | `compiler/tests/semantics/semantics_memory_surface_test.go` covers borrowed async result rejection and local borrowed use before `await`; `compiler/internal/memoryfacts_test/from_plir_test.go` proves `async_boundary_borrow_conservative`; `compiler/internal/memorymodel/mini_test.go` covers pre-suspension local use and across-await conservatism.                                                                                              | Borrowed views may be used locally before suspension. Crossing an async suspension remains conservative unless proven local and non-escaping.                                                                                               |
| MEM-BORROW-009 | validated_narrow | `compiler/internal/memoryfacts_test/from_plir_test.go` proves `task_boundary_borrow_rejected` with `task_boundary_borrow_validator`; `compiler/internal/memorymodel/mini_test.go` rejects borrowed task boundary crossing and accepts copied task crossing in the model; `compiler/tests/semantics/semantics_memory_surface_test.go` covers the current typed-task surface rejection for reference-shaped error payloads and unknown target rejection. | Current task APIs expose worker signatures and typed error payload types, not a general payload-send expression. Borrowed task boundary transfer is rejected/conservative unless an explicit copy path is available in the checked surface. |
| MEM-BORROW-010 | validated_narrow | `compiler/tests/semantics/semantics_memory_surface_test.go` covers actor `.copy()` acceptance, borrowed actor send rejection, and struct/optional/generic wrapper rejection; `compiler/internal/memoryfacts_test/from_plir_test.go` proves `actor_boundary_borrow_rejected`; `compiler/internal/memorymodel/mini_test.go` rejects borrowed actor boundary crossing and accepts copied/owned actor crossing.                                            | Typed actor sends reject borrowed slice/String views and wrappers unless the payload expression explicitly uses `.copy()` or an already-supported owned move path.                                                                          |
| MEM-ALIAS-004  | conservative     | `compiler/internal/memoryfacts_test/from_plir_test.go` proves `boundary_noalias_conservative` with `alias_state: invalidated_by_call`; `compiler/internal/memorymodel/mini_test.go` covers task and actor boundary noalias rejection; report validators continue rejecting broad noalias claims.                                                                                                                                                       | Task/actor boundaries never grant broad noalias in this slice. Boundary alias evidence remains conservative fallback.                                                                                                                       |

## Minimal Report Projection

Projected v4 claims:

| claim                                | source stage | validator                               | notes                                                                                                                                                  |
| ------------------------------------ | ------------ | --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `async_boundary_borrow_conservative` | plir         | `async_boundary_borrow_validator`       | Derived for borrowed views that may cross an async/await suspension boundary; projected as conservative fallback, not as a trusted lifetime-safe fact. |
| `task_boundary_borrow_rejected`      | plir         | `task_boundary_borrow_validator`        | Derived for borrowed views crossing task boundary without explicit copy; projected as rejected unsupported boundary evidence.                          |
| `actor_boundary_borrow_rejected`     | plir         | `actor_boundary_borrow_validator`       | Derived for borrowed views crossing actor boundary without explicit copy; projected as rejected unsupported boundary evidence.                         |
| `boundary_noalias_conservative`      | plir         | `boundary_alias_conservative_validator` | Derived for task/actor boundary alias evidence and projected as conservative fallback rather than validated noalias.                                   |

Validators reject derived v4 rows without `parent_fact_id`, safe claims from `unsafe_unknown`,
trusted boundary facts for unknown task/actor targets, and broad task/actor noalias wording.
`unsafe_unknown` remains rejected or conservative and cannot become a trusted borrowed source.

## Positive Coverage

- Borrowed views used before an async suspension stay local and non-escaping.
- Actor payloads using explicit `.copy()` are accepted on the current `core.send_typed` surface.
- The MiniMemoryModel accepts copied task/actor boundary crossing and owned actor boundary crossing
  where the checker already supports owned values.

## Negative Coverage

- Borrowed async results escaping through `await` are rejected or conservative.
- Borrowed actor payloads are rejected without `.copy()`.
- Struct, optional, and generic wrappers carrying borrowed actor payloads are rejected on the
  current typed-actor surface.
- The current typed-task surface rejects reference-shaped error payloads and unknown task targets
  instead of emitting trusted facts.
- Broad noalias through task/actor boundaries is rejected or conservative.

## Nonclaims

- No full production actor runtime.
- No full async lifetime system.
- No structured concurrency.
- No cancellation lifetime model.
- No distributed actor memory model.
- No zero-copy region move expansion beyond already-supported local actor ownership transfer
  evidence.
- No raw pointer expansion.
- No target parity.
- No broad noalias.
- No performance claim.
