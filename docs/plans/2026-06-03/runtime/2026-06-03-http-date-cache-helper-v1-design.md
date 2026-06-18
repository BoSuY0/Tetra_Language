# P19.2 HTTP Date Cache Helper V1 Design

**Goal:** implement the narrow P19.2 Date/cache helper for the local HTTP/JSON
server path without promoting the full production web stack.

**Context:** P19.2 lists `Date/cache helpers` as an HTTP/JSON target. The
foundation report currently records Date/cache as a documented boundary:
`compiler/internal/webrt/server.go::Config.DateFunc` gives deterministic tests,
`Server.date` formats `time.Now().UTC()` on each response, and
`compiler/internal/httprt/production_http_json_coverage.go` still says no
per-second Date cache exists.

## Design

- Add `compiler/internal/webrt/date_cache.go` with `HTTPDateCache`.
- Cache formatted IMF-fixdate strings by UTC Unix second.
- Expose `Format(now time.Time) string` for the server hot path and
  `FormatWithReport(now time.Time) (string, HTTPDateCacheReport)` for tests and
  P19.2 evidence.
- Add internal `Config.NowFunc func() time.Time` so server integration tests can
  prove cache reuse deterministically without overriding the Date header.
- Preserve `Config.DateFunc` as the highest-priority override for existing
  tests and explicit deterministic Date headers.
- Store one cache on `Server` and use it only when `DateFunc` is nil.
- Keep the helper narrow: no source-level `lib.core.http` cached-date API, no
  cross-worker/global cache, no writev/sendfile work, and no performance claim.

## Tasks

1. RED tests for `webrt`
   - Add direct cache tests proving one refresh within a UTC second and refresh
     at the next second.
   - Add server integration tests proving `Server.date` uses the cache when
     `DateFunc` is absent and preserves `DateFunc` override priority.
   - Verification:
     `GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/webrt -run 'TestHTTPDateCache|TestServerDate' -count=1`

2. RED tests for P19.2 coverage
   - Promote the Date/cache row to `implemented_narrow`.
   - Require facts for `HTTPDateCache`, `FormatWithReport`, deterministic
     server cache integration, and the remaining source-level cached-date
     boundary.
   - Verification:
     `GOCACHE=$(pwd)/.cache/go-build-ideal-plan go test ./compiler/internal/httprt -run 'TestProductionHTTPJSONCoverage' -count=1`

3. Implementation
   - Implement the helper and wire `Server.date` through it.
   - Update `ProductionHTTPJSONCoverage` and validator text.
   - Preserve no-claims for full production web stack, official TechEmpower,
     production PostgreSQL, P20 performance matrix, C++/Rust parity, measured
     speed, writev, and sendfile.

4. Documentation and sidecars
   - Update audit/progress docs, feature text/tests, current supported surface,
     stdlib spec, generated manifest, and goal sidecars.
   - Keep writev/sendfile as open P19.2 boundaries.

5. Verification
   - Focused webrt/httprt tests.
   - Relevant broader HTTP/JSON package gate.
   - Docs/manifest validators.
   - `git diff --check`.
   - `graphify update .` after code changes.
   - Clean the persistent Go cache after evidence runs.

## Done When

The Date/cache helper is complete when focused tests prove per-second cache
reuse and boundary refresh, P19.2 coverage promotes only the Date/cache row to
`implemented_narrow`, documentation no longer says Date cache is missing, and
all verification commands pass. P19.2 remains open until writev/sendfile and any
later acceptance gates are separately proven.
