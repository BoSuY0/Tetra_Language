# Memory Ideal Vertical Slice v9 Storage Goal

<goal>
Implement **Memory Ideal Vertical Slice v9: escape-aware storage/lowering
validator**.

Completion means v8 / `MEM-REPORT-008` remains accepted as `validated_narrow`,
and the v0-v8 Memory Ideal evidence chain gains a narrow storage/lowering
slice that prevents trusted stack/region/task/actor storage claims when escape
evidence exists.

The v9 slice must cover exactly these requirement IDs:

- `MEM-STORAGE-001`: escaped value cannot lower as trusted stack/region
  storage.
- `MEM-STORAGE-002`: stack/region storage requires compiler-owned no-escape
  proof.
- `MEM-STORAGE-003`: heap/conservative fallback must preserve
  `source_fact_id` and reason.
- `MEM-STORAGE-004`: async/task/actor/FFI boundary escape keeps storage
  conservative.

This is a narrow storage evidence slice. It must not claim full region
inference, performance, target parity, production actor runtime proof, full
async lifetime system, arbitrary FFI lifetime proof, optimizer-wide allocation
correctness, arbitrary external pointer safety, or "Memory 100%".
</goal>

<context>
Read first on every continuation:

- `AGENTS.md`
- `GOAL.md`
- `graphify-out/GRAPH_REPORT.md`
- Graphify MCP context for allocation planning, escape classification,
  storage/lowering validation, `MemoryFactGraph`, report projection,
  `validate-memory-report`, and `validate-memory-correlation`
- `.workflow/memory-ideal-vertical-slice-v9-storage/plan.md`
- `.workflow/memory-ideal-vertical-slice-v9-storage/attempts.md`
- `.workflow/memory-ideal-vertical-slice-v9-storage/notes.md`
- `.workflow/memory-ideal-vertical-slice-v9-storage/control.md`
- `.workflow/memory-ideal-vertical-slice-v8-report/final-report.md`
- `docs/audits/memory-ideal-vslice-v8-report-correlation.md`
- `docs/audits/memory-ideal-vslice-v8-report-final.md`
- `docs/audits/memory-ideal-vslice-v7-ffi-correlation.md`
- `docs/audits/memory-ideal-vslice-v7-ffi-final.md`
- `docs/audits/memory-ideal-vslice-v6-bounds-correlation.md`
- `docs/audits/memory-ideal-vslice-v6-bounds-final.md`
- `docs/spec/memory_report_schema_v1.md`
- `docs/design/memory_production_core_v1.md`

Likely integration points:

- `compiler/internal/allocplan/plan.go`
- `compiler/internal/allocplan/plan_test.go`
- `compiler/internal/validation/validation.go`
- `compiler/internal/lower/lower.go`
- `compiler/internal/lower/allocation_stack_test.go`
- `compiler/internal/memoryfacts/from_plir.go`
- `compiler/internal/memoryfacts/from_plir_test.go`
- `compiler/internal/memoryfacts/report.go`
- `compiler/internal/memoryfacts/report_test.go`
- `compiler/internal/memorymodel`
- `compiler/tests/semantics`
- `compiler/tests/ownership`
- `tools/cmd/validate-memory-report`
- `tools/cmd/validate-memory-correlation`
- `tools/cmd/validate-memory-fuzz-oracle`
- `docs/audits/memory-ideal-vslice-v9-storage-correlation.md`
- `docs/audits/memory-ideal-vslice-v9-storage-final.md`
- `docs/spec/memory_report_schema_v1.md`
- `docs/design/memory_production_core_v1.md`
- `docs/generated/manifest.json`

Current baseline evidence:

- v8 final report accepts `MEM-REPORT-008` as `validated_narrow`, with
  `MemoryFactGraph` as truth source and reports as projections.
- `compiler/internal/allocplan.VerifyPlan` already rejects escaping
  allocations that use trusted local storage classes such as Stack, Register,
  Region, FunctionTempRegion, and non-empty Eliminated storage.
- Existing allocation tests cover returned copies, actor send, unknown-call
  retained copy, heap fallback, and stack/function-temp region paths.
- Existing memoryfacts tests already preserve stack/function-temp heap fallback
  rows as non-validated projection evidence. v9 must make this discipline an
  exact Memory Ideal slice with correlation rows, validators, negative tests,
  docs, and current gates.
- `git status --short` remains heavily dirty. This does not block v9 evidence
  but blocks any clean-release claim.
</context>

<constraints>
- Always communicate with the user in Ukrainian.
- Keep scope to `MEM-STORAGE-001` through `MEM-STORAGE-004`.
- `MemoryFactGraph` remains truth; reports and correlation docs are
  projections/evidence artifacts only.
- In scope:
  - tying escape facts/classification to storage and lowering decisions;
  - rejecting trusted Stack/Register/Region/FunctionTempRegion/TaskRegion/
    ActorMoveRegion/ExplicitIsland storage when escape evidence exists;
  - requiring compiler-owned no-escape proof before trusted stack/region
    storage can be validated;
  - preserving `source_fact_id`, `reason`, planned storage, and actual lowering
    storage on heap/conservative fallback rows;
  - keeping async/task/actor/FFI/unknown-call escape storage conservative;
  - exact v9 correlation rows, docs, schema/design notes, manifest evidence,
    and final workflow report.
- Non-scope:
  - full region inference;
  - performance claim;
  - target parity;
  - production actor runtime proof;
  - full async lifetime system;
  - arbitrary FFI lifetime proof;
  - optimizer-wide allocation correctness proof;
  - arbitrary external pointer safety;
  - "Memory 100%";
  - clean-release claim while `git status --short` is dirty.
- Do not widen existing v0-v8 claims. If a validator cannot prove a storage
  relationship, classify the row as conservative/rejected or record a blocker.
- Preserve unrelated dirty worktree changes. Do not revert or clean unrelated
  files.
- If code files change, run `graphify update .` before completion.
- Use persistent Go caches under `.cache/` or `$HOME/.cache`; never set
  `GOCACHE` to `/tmp`.
</constraints>

<scorecard>
Primary metric: 100% of the four v9 requirement IDs have code, tests,
validators, docs, exact correlation rows, and current command evidence.

Passing threshold:

- `MEM-STORAGE-001` through `MEM-STORAGE-004` are represented in v9
  correlation and final audit docs.
- Validators reject:
  - returned/escaped value lowered as trusted Stack/Register/Region/
    FunctionTempRegion storage;
  - global escape lowered as trusted region/local storage;
  - async/task/actor boundary escape lowered as trusted local storage;
  - FFI or unknown external may-retain escape lowered as trusted
    non-escaping storage;
  - heap/conservative fallback row missing `source_fact_id`;
  - heap/conservative fallback row missing `reason`;
  - trusted stack/region row without compiler-owned no-escape proof.
- Existing v5-v8 guardrails still reject unsafe_unknown safe/noalias
  promotion, broad noalias, missing dynamic normal-build checks,
  unsafe/external bounds-check elimination, FFI trust promotion, projection
  drift, and broad memory claim drift.

Scoring command or inspection path:

- Inspect `GOAL.md`, `.workflow/memory-ideal-vertical-slice-v9-storage/`,
  v9 audit docs, `compiler/internal/allocplan`, `compiler/internal/validation`,
  `compiler/internal/lower`, `compiler/internal/memoryfacts`, report
  validators, correlation validators, schema/design docs, and command evidence
  recorded in attempts/final-report.

Regression checks:

- focused allocplan/validation/lower/memoryfacts/memorymodel/tool tests;
- v0-v9 correlation regression;
- manifest/docs validation;
- broad `go test`;
- canonical `scripts/ci/test.sh`;
- `git diff --check`;
- `graphify update .` if code changed.

Stop condition:

- Stop and record a blocker in `GOAL.md` if v9 requires full region inference,
  optimizer-wide allocation correctness, target/performance proof, arbitrary
  FFI/runtime proof, production actor runtime proof, or a clean-release claim
  impossible under the current dirty worktree.
- Stop after the same focused or full gate fails twice for the same reason
  without new evidence.
</scorecard>

<done_when>
The goal is complete only when all are true:

- `.workflow/memory-ideal-vertical-slice-v9-storage/final-report.md` exists
  and contains accepted/rejected/conflict integration summary for all v9
  packets.
- `docs/audits/memory-ideal-vslice-v9-storage-correlation.md` exists and
  validates with exactly four rows:
  `MEM-STORAGE-001`, `MEM-STORAGE-002`, `MEM-STORAGE-003`, and
  `MEM-STORAGE-004`.
- `docs/audits/memory-ideal-vslice-v9-storage-final.md` exists and classifies
  `MEM-STORAGE-001` through `MEM-STORAGE-004` as `validated_narrow`,
  `conservative`, or `rejected`.
- Source/report validator coverage exists for:
  `storage_escape_validator`,
  `storage_no_escape_proof_validator`,
  `heap_fallback_reason_validator`, and
  `boundary_storage_conservative_validator` or clearly equivalent local names.
- RED tests were observed and recorded for:
  - escaped return value lowered as stack/region accepted;
  - global escape lowered as region/local trusted storage accepted;
  - async/task/actor boundary escape lowered as trusted local storage accepted;
  - FFI/unknown may-retain escape lowered as trusted non-escaping storage
    accepted;
  - heap fallback row missing `source_fact_id` or reason accepted;
  - stack/region row without no-escape proof accepted.
- GREEN implementation passes:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-memoryfacts go test ./compiler/internal/memoryfacts -count=1`.
- Allocation/validation/lower focused gates pass:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-allocplan go test ./compiler/internal/allocplan -run 'Storage|Escape|Region|Heap|Lower' -count=1`;
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-validation go test ./compiler/internal/validation -run 'Storage|Escape|Region|Heap|Lower' -count=1`;
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-lower go test ./compiler/internal/lower -run 'Storage|Escape|Region|Heap|Lower' -count=1`.
- Mini model and semantics/ownership gates pass:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-mini go test ./compiler/internal/memorymodel -count=1`;
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-semantics go test ./compiler/tests/semantics ./compiler/tests/ownership -run 'Memory|Borrow|Escape|Storage|Region|Heap|Actor|Task|Async|FFI|Raw|Pointer' -count=1`.
- Tool validators pass:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation ./tools/cmd/validate-memory-fuzz-oracle -count=1`.
- v9 correlation validates:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v9-storage-correlation.md`.
- v0-v9 correlation regression passes:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-regression bash -lc 'for f in docs/audits/memory-ideal-vslice-v*-correlation.md; do go run ./tools/cmd/validate-memory-correlation --file "$f"; done'`.
- Docs/manifest pass:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`;
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`.
- Full gates pass:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-broad go test ./compiler/... ./cli/... ./tools/... -count=1`;
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-ci bash scripts/ci/test.sh`.
- Hygiene/release caveat evidence is recorded:
  `git diff --check`;
  `git status --short`;
  `graphify update .` if code files changed.
- Final audit repeats nonclaims: no full region inference, no performance,
  no target parity, no production actor runtime proof, no full async lifetime
  system, no arbitrary FFI lifetime proof, no optimizer-wide allocation
  correctness proof, no arbitrary external pointer safety, no "Memory 100%",
  and no clean-release claim while worktree is dirty.
</done_when>

<feedback_loop>
Fast iterative checks:

- After memoryfacts/report changes:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-memoryfacts go test ./compiler/internal/memoryfacts -count=1`
  Expected runtime: seconds. Run after each RED/GREEN cluster touching graph,
  report, storage projection, or in-process validators.
- After allocation plan changes:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-allocplan go test ./compiler/internal/allocplan -run 'Storage|Escape|Region|Heap|Lower' -count=1`
  Expected runtime: seconds. Run after each storage/escape planner change.
- After validation/lower changes:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-validation go test ./compiler/internal/validation -run 'Storage|Escape|Region|Heap|Lower' -count=1`
  and
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-lower go test ./compiler/internal/lower -run 'Storage|Escape|Region|Heap|Lower' -count=1`.
- After tool validator changes:
  `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation ./tools/cmd/validate-memory-fuzz-oracle -count=1`.

Proxy validity:

- These focused checks directly exercise storage/escape/lowering evidence and
  the report/correlation layer v9 changes.

Slower escalation/final checks:

- mini model plus semantics/ownership gates;
- v0-v9 correlation regression;
- manifest/docs validation;
- broad `go test ./compiler/... ./cli/... ./tools/... -count=1`;
- canonical `scripts/ci/test.sh`;
- `graphify update .` after code changes.
</feedback_loop>

<workflow>
1. Re-read `AGENTS.md`, `GOAL.md`, Graphify MCP context, workflow files, v8
   final evidence, v8 audit docs, and current `git status --short`.
2. Inspect concrete code paths for `allocplan.classifyEscape`,
   `allocplan.chooseStorage`, `allocplan.VerifyPlan`,
   `validation.ValidateAllocationPlan`, `validation.ValidateAllocationLowering`,
   lower stack/region emission, `memoryfacts.FromPLIRAndAllocPlan`,
   `ValidateReport`, and `validate-memory-correlation`.
3. Update `.workflow/memory-ideal-vertical-slice-v9-storage/plan.md` with the
   current phase and RED-first strategy.
4. Add RED tests first for escaped trusted storage, missing no-escape proof,
   heap fallback source/reason preservation, boundary conservatism, and v9
   correlation exactness.
5. Run RED focused gates and record exact failures in `attempts.md`.
6. Implement the smallest GREEN changes:
   storage/escape validator(s), no-escape proof check(s), fallback reason
   preservation, boundary conservative rows, v9 exact row set, and docs
   claim boundaries.
7. Add v9 audit docs, schema/design notes, manifest entries, and final audit
   scaffold.
8. Run focused gates and fix only v9-related failures.
9. Run v0-v9 correlation regression.
10. Run docs/manifest gates, broad gates, `git diff --check`, `git status
    --short`, and `graphify update .` if code files changed.
11. Complete final report, update `GOAL.md ## Progress`, and call
    `update_goal complete` only when every `done_when` item is proven.
</workflow>

<working_memory>
Maintain these files:

- `.workflow/memory-ideal-vertical-slice-v9-storage/plan.md`
- `.workflow/memory-ideal-vertical-slice-v9-storage/attempts.md`
- `.workflow/memory-ideal-vertical-slice-v9-storage/notes.md`
- `.workflow/memory-ideal-vertical-slice-v9-storage/control.md`
- `.workflow/memory-ideal-vertical-slice-v9-storage/final-report.md` once
  integration is ready to close.

Update cadence:

- Update `GOAL.md ## Progress` at the start of each continuation turn and
  after each state change that affects acceptance evidence.
- Update `plan.md` when phase or strategy changes.
- Update `attempts.md` after each meaningful RED/GREEN cluster, failed gate,
  or successful final gate.
- Update `notes.md` for durable discoveries, caveats, blockers, and
  nonclaims.
- Keep progress terse: one line per state change, evidence by reference, not a
  transcript.
</working_memory>

<human_control_surface>
Maintain `.workflow/memory-ideal-vertical-slice-v9-storage/control.md` as the
compact operator panel for this goal.

Before each phase change, strategic pivot, expensive step, or sidecar
ingestion, reread `control.md`. If it changed, summarize the relevant change
in `plan.md` and adapt before proceeding.

Initial knobs:

- primary_priority: evidence_quality
- secondary_priority: behavior_preservation
- allowed_files:
  `compiler/internal/allocplan/**`,
  `compiler/internal/validation/**`,
  `compiler/internal/lower/**`,
  `compiler/internal/memoryfacts/**`,
  `compiler/internal/memorymodel/**`,
  `compiler/tests/semantics/**`,
  `compiler/tests/ownership/**`,
  `tools/cmd/validate-memory-report/**`,
  `tools/cmd/validate-memory-correlation/**`,
  `docs/audits/memory-ideal-vslice-v9-storage-*.md`,
  `docs/spec/memory_report_schema_v1.md`,
  `docs/design/memory_production_core_v1.md`,
  `docs/generated/manifest.json`,
  `.workflow/memory-ideal-vertical-slice-v9-storage/**`,
  `GOAL.md`
- protected_files: unrelated dirty files outside v9 scope
- require_approval_for: scope expansion, dependency change, broad schema
  redesign, performance/target/runtime claim, destructive cleanup of dirty
  worktree
- max_parallel_jobs: use parallel reads/tests when independent; avoid running
  broad CI concurrently with other heavy gates

`control.md` can narrow priorities or require approval, but it cannot silently
weaken `done_when`, the scorecard, or v9 nonclaims.
</human_control_surface>

<verification_loop>
Required focused gates:

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-memoryfacts go test ./compiler/internal/memoryfacts -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-allocplan go test ./compiler/internal/allocplan -run 'Storage|Escape|Region|Heap|Lower' -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-validation go test ./compiler/internal/validation -run 'Storage|Escape|Region|Heap|Lower' -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-lower go test ./compiler/internal/lower -run 'Storage|Escape|Region|Heap|Lower' -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-mini go test ./compiler/internal/memorymodel -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-semantics go test ./compiler/tests/semantics ./compiler/tests/ownership -run 'Memory|Borrow|Escape|Storage|Region|Heap|Actor|Task|Async|FFI|Raw|Pointer' -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-tools go test ./tools/cmd/validate-memory-report ./tools/cmd/validate-memory-correlation ./tools/cmd/validate-memory-fuzz-oracle -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-correlation go run ./tools/cmd/validate-memory-correlation --file docs/audits/memory-ideal-vslice-v9-storage-correlation.md`

Required regression/docs gates:

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-regression bash -lc 'for f in docs/audits/memory-ideal-vslice-v*-correlation.md; do go run ./tools/cmd/validate-memory-correlation --file "$f"; done'`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-manifest go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`

Required full gates:

- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-broad go test ./compiler/... ./cli/... ./tools/... -count=1`
- `GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-memory-v9-storage-ci bash scripts/ci/test.sh`
- `git diff --check`
- `git status --short`
- `graphify update .` after code changes

If a check cannot run, record the exact command, failure mode, and
classification (`needs_reproduction`, `blocked`, or out-of-scope) in
`attempts.md` and `GOAL.md`.
</verification_loop>

<execution_rules>
- Follow `AGENTS.md`; communicate in Ukrainian.
- Check git status before edits and preserve unrelated user changes.
- Prefer `rg` over `grep`.
- Use `apply_patch` for manual file edits.
- Read context files before implementation.
- Batch independent file reads in parallel when supported.
- Use Graphify MCP before architecture/codebase decisions; after code changes,
  run `graphify update .` before completion.
- Keep the scorecard current and do not widen scope.
- Use the fastest representative feedback checks while iterating; reserve slow
  gates for escalation/final verification.
- Maintain workflow memory files for long-running context.
- Update `attempts.md` after each meaningful approach.
- Run focused tests before broad tests.
- Do not paper over failures or claim pass without current command evidence.
- Do not set `GOCACHE` to `/tmp`; use persistent `.cache/` paths.
- Do not claim clean release while `git status --short` is non-empty.
- Stop after the same focused or full gate fails twice for the same reason
  without new evidence.
- Keep final answer concise.
</execution_rules>

<output_contract>
Final output must include:

- v9 decision: accepted/rejected/conflict;
- row classifications for `MEM-STORAGE-001` through `MEM-STORAGE-004`;
- files changed and final evidence packet path;
- focused, regression, docs, full, hygiene, and Graphify gate results;
- dirty-worktree caveat from `git status --short`;
- explicit nonclaims;
- `update_goal complete` only after every `done_when` item is proven.
</output_contract>

## Progress

- Completed: v8 / `MEM-REPORT-008` accepted by user as `validated_narrow`,
  not "Memory 100%", and not a new memory semantics surface.
- Completed: active goal check returned no active goal before v9 setup.
- Completed: Graphify and file inspection found v9 storage/escape anchors:
  `compiler/internal/allocplan/plan.go` (`classifyEscape`, `chooseStorage`,
  `VerifyPlan`), `compiler/internal/validation/validation.go`,
  `compiler/internal/lower/allocation_stack_test.go`, and
  `compiler/internal/memoryfacts/from_plir.go`.
- Completed: current `git status --short` is heavily dirty; v9 evidence may
  proceed, but clean-release claim remains blocked.
- Completed: v9 workflow scaffold exists under
  `.workflow/memory-ideal-vertical-slice-v9-storage/`, and active goal was
  created for `MEM-STORAGE-009`.
- Completed: RED tests added and observed. Focused RED gates fail because
  `VerifyPlan` accepts escaped actual trusted lowering, trusted storage without
  no-escape proof, and heap fallback without reason; memoryfacts accepts
  fallback without reason; MiniMemoryModel lacks v9 storage fields/outcomes;
  correlation validator treats `MEM-STORAGE-*` rows as unexpected v0 rows.
- Completed: GREEN storage/correlation cluster passed for
  `compiler/internal/allocplan`, `compiler/internal/memoryfacts`,
  `compiler/internal/memorymodel`, and
  `tools/cmd/validate-memory-correlation` with v9 `.cache` GOCACHE paths.
- Completed: v9 docs and evidence packet exist:
  `docs/audits/memory-ideal-vslice-v9-storage-correlation.md`,
  `docs/audits/memory-ideal-vslice-v9-storage-final.md`, schema/design notes,
  manifest entries, and
  `.workflow/memory-ideal-vertical-slice-v9-storage/final-report.md`.
- Completed: all required focused, v9 correlation, v0-v9 regression,
  docs/manifest, broad `go test`, canonical `scripts/ci/test.sh`, hygiene,
  dirty-worktree, and Graphify gates passed or were recorded with evidence.
- Completed: `git status --short` remains heavily dirty, so v9 can claim
  `validated_narrow` evidence but cannot claim clean-release state.
