# P19.2 Linux Writev/Sendfile Helper V1 Design

**Goal:** implement the narrow supported-platform `writev`/`sendfile` helper
slice for P19.2 without claiming a full production web stack or performance
result.

**Context:** P19.2 lists `writev/sendfile where supported`. Current concrete
state:

- `compiler/internal/netrt/netrt_linux.go` provides Linux `Read`, `Recv`,
  `Write`, `Send`, `Accept4`, and epoll helpers.
- `compiler/internal/netrt/netrt_unsupported.go` returns `ErrUnsupported` for
  non-Linux networking helpers.
- `compiler/internal/webrt/server.go::flush` still writes one output buffer via
  `netrt.Write`.
- `compiler/internal/httprt/production_http_json_coverage.go` still records the
  `writev_sendfile_boundary` row as `boundary_documented`.
- Go exposes `syscall.Sendfile` on Linux, but not a public `syscall.Writev`
  wrapper in this environment; `SYS_WRITEV` and `syscall.Iovec` are available.

## Design

- Add `netrt.Writev(fd int, chunks [][]byte) (int, error)`.
  - Linux implementation builds `syscall.Iovec` entries for non-empty chunks
    and calls `SYS_WRITEV`.
  - Empty chunk lists and all-empty chunks return `(0, nil)`.
  - Partial writes remain the caller's responsibility, matching existing
    `Write`/`Send` helper style.
- Add `netrt.Sendfile(outFD int, inFD int, offset *int64, count int) (int,
  error)`.
  - Linux implementation delegates to `syscall.Sendfile`.
  - The caller owns retry/offset/count policy.
- Add non-Linux stubs that return `ErrUnsupported`.
- Keep `webrt.flush` unchanged for this slice. That avoids changing local HTTP
  scheduling behavior while helper semantics are proven.
- Promote only the P19.2 writev/sendfile helper row to `implemented_narrow`.
- Preserve explicit non-claims:
  - no HTTP static-file response path;
  - no `webrt.flush` scatter/gather integration;
  - no zero-copy production file-serving claim;
  - no throughput/performance claim;
  - no non-Linux platform parity.

## Tests

1. RED `netrt` tests
   - `TestWritevWritesMultipleBuffersOnConnectedTCP` proves scatter/gather
     writes concatenate multiple buffers on a real connected TCP FD.
   - `TestSendfileCopiesFileBytesToConnectedTCPAndAdvancesOffset` proves
     Linux `sendfile` copies a bounded file range to a connected TCP FD and
     advances the supplied offset.
   - Verification:
     `GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/netrt -run 'TestWritev|TestSendfile' -count=1`

2. RED P19.2 coverage test
   - Promote `ProductionHTTPJSONWritevSendfileBoundary` to
     `ProductionHTTPJSONImplementedNarrow`.
   - Require facts for `netrt.Writev`, `netrt.Sendfile`, both Linux tests, and
     the remaining `webrt.flush` / HTTP static-file boundaries.
   - Verification:
     `GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/httprt -run 'TestProductionHTTPJSONCoverage' -count=1`

3. Documentation and sidecars
   - Update audit/progress docs, feature text/tests, current supported surface,
     stdlib spec, generated manifest, and goal sidecars.

4. Final verification
   - Focused RED/GREEN.
   - Relevant `netrt/httprt/webrt/jsonrt/truth-bench-harness/semantics/docs`
     package gate.
   - CLI docs/manifest validators.
   - `git diff --check`.
   - `graphify update .` after code changes.
   - Clean the persistent Go cache after evidence runs.

## Done When

The slice is complete when Linux `Writev` and `Sendfile` helpers pass real-FD
tests, P19.2 coverage promotes the row to `implemented_narrow`, docs no longer
say the helpers are missing, and all verification commands pass. P19.2 still
must not claim full production web-stack behavior, HTTP static-file serving,
performance, or cross-platform parity.
