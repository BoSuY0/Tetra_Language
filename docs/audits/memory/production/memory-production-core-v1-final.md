# Memory Production Core v1 Final Audit

Status: validated after the MPC-16 command set passed on 2026-06-04.

Memory Production Core v1 is scoped to supported safe-memory behavior, conservative unsafe
boundaries, compiler-owned facts, schema-versioned report projections, target-tiered claims,
cost-model classification, and oracle-backed memory fuzz smoke evidence.

Allowed row statuses:

- `implemented`
- `implemented_narrow`
- `validated`
- `conservative`
- `rejected`
- `future`
- `explicit_non_goal`

reports are projections of `MemoryFactGraph` and adjacent compiler-owned IR, allocation-plan,
lowering, runtime-ABI, target, and validator facts. Reports are not sources of truth.

Linked docs:

- `docs/audits/memory/production/memory-production-core-v1-artifact-map.md`
- `docs/audits/memory/production/memory-production-core-v1-nonclaims.md`

## Final Classification

Final row records:

- Row: MPC-0.
  Area: baseline and gap map.
  Final status: `implemented`.
  Evidence refs:
  `docs/audits/memory/production/memory-production-core-v1-baseline.md`,
  `docs/audits/memory/production/memory-production-core-v1-gap-map.md`.
  Scope notes: establishes current surface and gaps; not a proof of future
  rows.
  Overclaim risk: low.
- Row: MPC-1.
  Area: Memory Fact Graph v0.
  Final status: `validated`.
  Evidence refs: `compiler/internal/memoryfacts`,
  `docs/spec/memory/memory_report_schema_v1.md`, final command
  `go test ./compiler/internal/memoryfacts -count=1`.
  Scope notes: compiler-owned fact graph is the truth source for report
  projection.
  Overclaim risk: medium.
- Row: MPC-2.
  Area: Memory Report Schema v1 and validator.
  Final status: `validated`.
  Evidence refs: `tools/cmd/validate-memory-report`,
  `compiler/internal/memoryfacts/report.go`,
  `docs/spec/memory/memory_report_schema_v1.md`.
  Scope notes: schema validation rejects invalid projections; it does not
  infer facts the compiler did not own.
  Overclaim risk: medium.
- Row: Raw-bounds closure.
  Area: verified `core.alloc_bytes` roots and conservative unknown raw
  pointers.
  Final status: `validated`.
  Evidence refs: `compiler/internal/runtimeabi/raw_pointer_bounds.go`,
  `compiler/internal/plir`,
  `reports/memory-production-core-v1/mpc8/memory-production-linux-x64.json`.
  Scope notes: runtime evidence is linux-x64 scoped unless a target-specific
  row says otherwise.
  Overclaim risk: high.
- Row: MPC-3.
  Area: safe representation invariant hardening.
  Final status: `validated`.
  Evidence refs: `compiler/internal/semantics`, `compiler/tests/semantics`,
  `docs/audits/memory/production/memory-production-core-v1-supported-surface.md`.
  Scope notes: safe metadata assignment is rejected before lowering.
  Overclaim risk: medium.
- Row: MPC-4.
  Area: borrow/lifetime supported surface hardening.
  Final status: `implemented_narrow`.
  Evidence refs: `compiler/tests/safety`, `compiler/tests/semantics`,
  `compiler/internal/memoryfacts`.
  Scope notes: covers documented slice/String and local supported cases, not
  full Rust-like lifetime parity.
  Overclaim risk: high.
- Row: MPC-5.
  Area: mutable alias / inout conservative subset.
  Final status: `conservative`.
  Evidence refs: `compiler/internal/memoryfacts`,
  `tools/cmd/validate-memory-report`,
  `docs/audits/memory/production/memory-production-core-v1-supported-surface.md`.
  Scope notes: unknown/maybe/call-invalidated alias state blocks noalias
  promotion.
  Overclaim risk: high.
- Row: MPC-6.
  Area: provenance/resource summaries v2.
  Final status: `implemented_narrow`.
  Evidence refs: `compiler/internal/memoryfacts/fromplir/from_plir.go`,
  `tools/cmd/validate-memory-report`, summary tests.
  Scope notes: summary vocabulary covers PLIR-visible supported cases;
  unknown external/resource returns remain conservative.
  Overclaim risk: medium.
- Row: MPC-7.
  Area: unsafe fact classes.
  Final status: `validated`.
  Evidence refs: `compiler/internal/memoryfacts`, `compiler/internal/plir`,
  `docs/spec/runtime/unsafe.md`.
  Scope notes: `unsafe_unknown` cannot become safe provenance, noalias,
  trusted storage, or removed-check evidence.
  Overclaim risk: high.
- Row: MPC-8.
  Area: raw pointer verified-root bounds.
  Final status: `validated`.
  Evidence refs: `compiler/internal/runtimeabi`, `compiler/internal/plir`,
  `reports/memory-production-core-v1/mpc8/memory-production-linux-x64.json`.
  Scope notes: verified roots are allocation-base scoped; arbitrary external
  raw pointers remain conservative.
  Overclaim risk: high.
- Row: MPC-9.
  Area: raw slice gateway hardening.
  Final status: `validated`.
  Evidence refs: `compiler/internal/runtimeabi`, `compiler/internal/lower`,
  `reports/memory-production-core-v1/mpc9/memory-production-linux-x64.json`.
  Scope notes: raw slice runtime trap evidence is linux-x64 scoped; non-x64
  rows remain target-tiered.
  Overclaim risk: high.
- Row: MPC-10.
  Area: storage truth: stack/heap/explicit island.
  Final status: `validated`.
  Evidence refs: `compiler/internal/allocplan`, `compiler/internal/lower`,
  `compiler/internal/validation`, `compiler/internal/memoryfacts`.
  Scope notes: planned storage and actual lowering storage remain separate;
  heap fallback is not a validated stack/region claim.
  Overclaim risk: high.
- Row: MPC-11.
  Area: function-temp implicit region narrow slice.
  Final status: `implemented_narrow`.
  Evidence refs: `compiler/internal/allocplan`, `compiler/internal/lower`,
  `compiler/internal/validation`, `compiler/internal/backend/x64core`.
  Scope notes: one narrow linux-x64 function-temp region path; broad region
  reuse/control-flow cleanup remains future.
  Overclaim risk: medium.
- Row: MPC-12.
  Area: actor/task/request conservative memory rules.
  Final status: `conservative`.
  Evidence refs: `compiler/tests/safety`, `compiler/tests/semantics`,
  `compiler/internal/actorsafety`, `compiler/internal/parallelrt`,
  `compiler/internal/httprt`.
  Scope notes: actor zero-copy move rows are evidence-only; full production
  actor runtime is not claimed.
  Overclaim risk: high.
- Row: MPC-13.
  Area: target capability matrix.
  Final status: `validated`.
  Evidence refs: `docs/audits/memory/islands/memory-target-capability-matrix.md`,
  `compiler/target`, `tools/cmd/validate-targets`.
  Scope notes: no cross-target memory production claim without target evidence.
  Overclaim risk: high.
- Row: MPC-14.
  Area: memory cost model.
  Final status: `validated`.
  Evidence refs: `docs/design/memory/memory_cost_model.md`,
  `compiler/internal/memoryfacts`, `compiler/compiler_reports.go`,
  `tools/cmd/validate-memory-report`.
  Scope notes: cost classes classify evidence; fake zero-cost or trusted
  unsafe optimization wording is rejected.
  Overclaim risk: medium.
- Row: MPC-15.
  Area: memory fuzz/property/stress with oracle.
  Final status: `validated`.
  Evidence refs: `docs/audits/memory/islands/memory-fuzz-oracle-v1.md`,
  `compiler/compiler_evidence_gates.go`, `tools/cmd/memory-fuzz-short`,
  `reports/memory-fuzz-short/mpc15/memory-fuzz-oracle.json`.
  Scope notes: Tier 1 is deterministic oracle smoke; random generation is not
  proof by itself.
  Overclaim risk: medium.
- Row: MPC-16.
  Area: production gate and final audit.
  Final status: `validated`.
  Evidence refs: this doc,
  `docs/audits/memory/production/memory-production-core-v1-artifact-map.md`,
  `docs/audits/memory/production/memory-production-core-v1-nonclaims.md`,
  `reports/memory-production-core-v1/test-all-quick/summary.json`,
  `reports/memory-production-core-v1/test-all-quick/summary.md`.
  Scope notes: full MPC-16 command set passed; quick output remains quick
  evidence, not a full/stabilization or benchmark claim.
  Overclaim risk: high.

## Conservative Boundaries

- Unknown unsafe memory is `conservative`, not safe.
- Unknown target runtime behavior is `conservative` or `future`, not target parity.
- Unsupported arbitrary external raw-pointer safety is `explicit_non_goal`.
- Full actor runtime production guarantees are `explicit_non_goal` outside the documented
  local/evidence-only slices.
- Full Rust-like borrow checker parity is `explicit_non_goal`.

## MPC-16 Command Evidence

The command evidence is recorded in `GOAL.md` progress and mirrored by
`docs/audits/memory/production/memory-production-core-v1-artifact-map.md`. The required commands
passed:

- Core package tests:

  ```sh
  GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-core \
    go test -p=1 \
    ./compiler/internal/memoryfacts \
    ./compiler/internal/plir \
    ./compiler/internal/validation \
    ./compiler/internal/allocplan \
    ./compiler/internal/lower \
    -count=1
  ```

- Compiler memory evidence subset:

  ```sh
  GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-compiler \
    go test -p=1 ./compiler \
    -run 'Memory|Borrow|Lifetime|Alias|Unsafe|Bounds|Alloc|Region|Island|Report' \
    -count=1
  ```

- Broad compiler/CLI/tools test:

  ```sh
  GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16 \
    go test -p=1 ./compiler/... ./cli/... ./tools/... -count=1
  ```

- CI smoke:

  ```sh
  GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16 \
    bash scripts/ci/test.sh
  ```

- Quick test-all evidence:

  ```sh
  GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-test-all \
    bash scripts/ci/test-all.sh \
    --quick \
    --keep-going \
    --report-dir reports/memory-production-core-v1/test-all-quick
  ```

- Test-all summary validation:

  ```sh
  GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-test-all \
    go run ./tools/cmd/validate-test-all-summary \
    --summary reports/memory-production-core-v1/test-all-quick/summary.json \
    --report-dir reports/memory-production-core-v1/test-all-quick
  ```

- Manifest validation:

  ```sh
  GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-docs \
    go run ./tools/cmd/validate-manifest \
    --manifest docs/generated/manifest.json
  ```

- Docs verification:

  ```sh
  GOTELEMETRY=off GOCACHE=$(pwd)/.cache/go-build-mpc16-docs \
    go run ./tools/cmd/verify-docs \
    --manifest docs/generated/manifest.json
  ```

- `git diff --check`
- `graphify update .`
