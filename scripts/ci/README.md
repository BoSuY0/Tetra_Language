# scripts/ci

CI-facing entrypoints live here.

`test.sh` is the canonical Go test-suite entrypoint. There is no root-level
compatibility wrapper.

`test-all.sh` is the canonical summarized release/stabilization test runner.
There is no root-level compatibility wrapper.

Other CI workflows may still delegate to stable legacy scripts until their
implementation slices migrate.
