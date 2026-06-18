# P19.2 Production HTTP/JSON Stack V1 Design

Follow-up note: `docs/plans/2026-06-03-http-date-cache-helper-v1-design.md`
implements the narrow internal per-server UTC-second Date/cache helper. This
foundation design still describes the earlier source-first coverage batch.

Follow-up note: `docs/plans/2026-06-03-linux-writev-sendfile-helper-v1-design.md`
implements the narrow Linux `netrt.Writev`/`netrt.Sendfile` helper slice.

## Goal

Define the first P19.2 source-first HTTP/JSON acceptance slice without claiming
the full production web stack or P20 performance results. The slice should make
the existing HTTP/JSON implementation auditable and add a local benchmark gate
that starts from Tetra source.

## Observed Facts

- `lib/core/http.tetra` exposes source-level route classification,
  `request_head_len_bytes(_at)`, keep-alive helpers, response length helpers,
  and plaintext/JSON response writers.
- `lib/core/json.tetra` exposes source-level JSON string and TechEmpower
  message-object writers.
- `compiler/internal/httprt/http1.go` contains the string-copying HTTP/1.1
  parser, router, and response builder used by the current server path.
- `compiler/internal/httprt/request_view.go` adds a borrowed request-head view,
  body slicing, header metadata, keep-alive handling, and
  `AppendResponseWithReport`.
- `compiler/internal/httprt/request_view_test.go` already proves request-view
  header borrowing and an ordinary request-region HTTP/JSON path with zero heap
  allocations.
- `compiler/internal/jsonrt/view.go` adds a borrowed/region JSON view parser;
  `compiler/internal/jsonrt/json.go` contains deterministic JSON builders.
- `compiler/internal/webrt/server.go` supports keep-alive pipelining, response
  decoration with `Server`/`Date`, and local TCP server tests. It has a
  `DateFunc` hook. The follow-up Date/cache helper slice now adds the narrow
  per-server UTC-second cache.
- Existing TechEmpower validators cover local runtime/DB benchmark reports.
  They do not define a P19.2 source-first acceptance report.
- `tools/cmd/truth-bench-harness` supports `p8_full` and
  `p19.1_generic_collections`, but not a P19.2 HTTP/JSON source-first scope.

## Scope

In scope:

- A report-only `ProductionHTTPJSONCoverage` API in `compiler/internal/httprt`.
- Rows for HTTP/1.1 request-head parsing, pipelining, headers/body/keep-alive,
  zero-heap request views, JSON parse/stringify, response building, date/cache
  boundary, writev/sendfile boundary, and source-first benchmark evidence.
- Validator rejection of fake full-production web stack, official
  TechEmpower, production PostgreSQL, P20 broad-performance, runtime-behavior,
  and C++/Rust parity claims.
- A `p19.2_http_json_source_first` truth-benchmark scope requiring Tetra-only
  `HTTP plaintext` and `JSON parse` rows, Tetra proof/allocation/bounds
  artifacts, and P19.2 evidence artifacts.
- Documentation updates that keep P20 benchmark claims separate.

Out of scope:

- No official TechEmpower publication claim.
- No C++/Rust parity claim.
- No production PostgreSQL stack claim.
- No full production web stack claim.
- No new socket protocol, writev, or sendfile implementation in this slice.
  Date/cache implementation is handled by the follow-up helper design.

## Design

Add `compiler/internal/httprt/production_http_json_coverage.go` with schema
`tetra.stdlib.http_json.production_stack.v1`. The report is intentionally
evidence-first: implemented rows point at observed source/tests, while
unsupported production targets stay as boundary rows with missing facts.

Add `ValidateProductionHTTPJSONCoverage` alongside the report. The validator
will require every P19.2 row, evidence and boundary text, required facts, and
explicit non-claims. It will reject forbidden claim booleans and per-row claim
flags.

Extend `tools/cmd/truth-bench-harness` with
`p19.2_http_json_source_first`. This scope allows only Tetra rows for
`HTTP plaintext` and `JSON parse`, requires algorithm/input metadata, and
requires Tetra proof/allocation/bounds plus `tetra_reports`. This proves the
benchmark artifact is source-first and not merely a runtime-only TechEmpower
report.

## Test Strategy

- RED `compiler/internal/httprt` tests for the new coverage API and fake-claim
  rejection.
- RED `tools/cmd/truth-bench-harness` tests for the P19.2 scope and runtime-only
  / official-claim rejection.
- Existing parser/request-view/server/json tests remain the behavior evidence;
  this slice should not rewrite those runtime paths.
- Focused GREEN commands use
  `GOCACHE=$(pwd)/.cache/go-build-ideal-plan`.

## Execution Plan

1. Add RED coverage tests in `compiler/internal/httprt`.
   Verification:
   `GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/httprt -run 'TestProductionHTTPJSONCoverage' -count=1`.

2. Add RED source-first benchmark-scope tests in
   `tools/cmd/truth-bench-harness`.
   Verification:
   `GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./tools/cmd/truth-bench-harness -run 'TestP19HTTPJSONSourceFirst' -count=1`.

3. Implement the coverage report/validator and harness scope with the smallest
   report-only code change.
   Verification: rerun the focused commands.

4. Add P19.2 benchmark manifest/report artifacts under
   `reports/production-http-json-v1/benchmarks/` and update docs/features
   references without expanding claims.
   Verification: run the harness dry-run and docs/manifest validators.

5. Run focused HTTP/JSON runtime evidence tests, broader relevant package gate,
   `git diff --check`, non-claim scans, and `graphify update .`.

## Risks

- The existing server path still allocates connection buffers and uses the
  string-copying parser; the zero-heap claim must remain limited to the
  request-view/request-region slice.
- In this first slice, Date cache and writev/sendfile started as target rows.
  The follow-up Date/cache helper design implements only the internal
  per-server Date cache, and the follow-up Linux helper design implements only
  supported-platform `netrt.Writev`/`netrt.Sendfile`. `webrt.flush`
  scatter/gather integration, HTTP static-file sendfile paths, and performance
  claims remain out of scope.
- Local TechEmpower reports are runtime/DB evidence; they must not be treated as
  official upstream TechEmpower results.
