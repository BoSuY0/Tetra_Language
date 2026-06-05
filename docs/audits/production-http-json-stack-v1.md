# Production HTTP/JSON Stack V1 Foundation

Status: P19.2 foundation evidence slice plus Date/cache and Linux
writev/sendfile helper evidence.

This audit records a source-first HTTP/JSON acceptance layer. It is not a full
production web stack, not an official TechEmpower result, not a PostgreSQL
production-stack promotion, not a P20 performance matrix, and not a C++/Rust
parity claim.

## Coverage

The machine-readable coverage API is:

- `compiler/internal/httprt/production_http_json_coverage.go`
- schema `tetra.stdlib.http_json.production_stack.v1`
- validator `ValidateProductionHTTPJSONCoverage`

Rows covered by the validator:

- HTTP/1.1 request-head parsing through `lib/core/http.tetra` and
  `compiler/internal/httprt/request_view.go`
- pipelined request-head slicing through source smoke and consumed-byte parser
  evidence
- headers, body, and keep-alive metadata
- zero-heap request-view/request-region evidence
- JSON parse/stringify evidence through `lib/core/json.tetra` and
  `compiler/internal/jsonrt`
- response builder evidence through source response writers and
  `AppendResponseWithReport`
- internal per-server UTC-second Date/cache helper
- Linux writev/sendfile helper evidence
- source-first benchmark gate

## Benchmark Gate

The checked dry-run benchmark artifact is:

- manifest:
  `reports/production-http-json-v1/benchmarks/http-json-source-first-manifest.json`
- report:
  `reports/production-http-json-v1/benchmarks/http-json-source-first-report.json`
- scope: `p19.2_http_json_source_first`

The scope requires Tetra-only `HTTP plaintext` and `HTTP JSON` rows, source
build commands with `tetra build ... --explain`, algorithm/input metadata, and
Tetra proof/allocation/bounds/P19.2 coverage artifacts. The generated report has
`ran=false`, so it records source-first gate coverage, not measured speed.

## Boundaries

- Date header injection/formatting is covered through `DateFunc` and server
  tests. `compiler/internal/webrt.HTTPDateCache` now caches formatted Date
  values per UTC second when `DateFunc` is not supplied, with
  `FormatWithReport` evidence for refresh/reuse tests.
- A source-level `lib.core.http` cached-date API, cross-worker/global Date
  cache, and Date performance claim remain out of scope for this slice.
- `compiler/internal/netrt.Writev` is implemented on Linux through
  `SYS_WRITEV`, with a connected-TCP test proving multi-buffer writes.
- `compiler/internal/netrt.Sendfile` is implemented on Linux through
  `syscall.Sendfile`, with a file-to-TCP test proving byte transfer and offset
  advancement.
- `webrt.flush` scatter/gather integration, an HTTP static-file sendfile path,
  non-Linux writev/sendfile parity, zero-copy production file-serving, and
  performance claims remain out of scope for this slice.
- Existing local TechEmpower runtime/DB reports remain separate evidence and do
  not become official upstream TechEmpower results.
- P20 owns broader benchmark matrices, external baselines, and performance
  claims.

## Verification

Focused evidence commands:

```sh
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/httprt -run 'TestProductionHTTPJSONCoverage' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./tools/cmd/truth-bench-harness -run 'TestP19HTTPJSONSourceFirst' -count=1
GOCACHE=$(pwd)/.cache/go-build-ideal-plan go run ./tools/cmd/truth-bench-harness --manifest reports/production-http-json-v1/benchmarks/http-json-source-first-manifest.json --out reports/production-http-json-v1/benchmarks/http-json-source-first-report.json
```
