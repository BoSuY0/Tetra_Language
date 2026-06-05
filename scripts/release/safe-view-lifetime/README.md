# scripts/release/safe-view-lifetime

Focused release gate for Safe View Lifetime Contracts v1.

Entrypoint:

- `gate.sh` runs the Safe View Lifetime Contracts v1 evidence gate. It executes
  focused compiler, CLI, and tooling tests; regenerates and validates the
  feature manifest; verifies docs; builds proof/allocation reports for borrowed
  safe-view returns and explicit copied escapes; and records negative actor/task
  boundary diagnostics.

By default, artifacts are written under `reports/safe-view-lifetime` so repeated
local runs do not use tmpfs-backed `/tmp`. Pass `--report-dir DIR` to choose an
explicit evidence directory, including a disposable directory for one-off runs.

This gate covers supported slice and String byte-view surfaces only. It does not
claim named lifetimes, mutable aliasing, or a full Rust-like borrow checker.
