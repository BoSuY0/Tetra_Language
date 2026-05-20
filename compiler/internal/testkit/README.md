# compiler/internal/testkit

Shared compiler test helpers live here once they are extracted from package-level
tests. Keep this package behavior-free: helpers may build fixtures, run compiler
pipelines, and assert diagnostics, but must not encode scenario ownership.

Initial migration targets:

- `buildAndRun` and `buildAndRunFiles` style helpers
- temporary module/fixture writers
- reusable diagnostic assertion helpers

