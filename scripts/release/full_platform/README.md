# scripts/release/full_platform

Release evidence for the full-platform UI runtime promotion wave.

This directory is intentionally separate from `v0_4_0` and `post_v0_4`
Linux-only evidence. The gate here requires fresh reports for Linux, Windows,
macOS, and Web in one report directory. Windows and macOS reports only count as
production evidence when they were produced by a real target-host runtime runner;
blocked, build-only, metadata-only, sidecar-only, placeholder, or runtime-less
reports are rejected by the validators.

CI fan-in:

- Run `go run ./tools/cmd/platform-ui-runtime-smoke --target windows-x64
  --report windows-ui-runtime.json` on a real Windows amd64 runner.
- Run `go run ./tools/cmd/platform-ui-runtime-smoke --target macos-x64
  --report macos-ui-runtime.json` on a real macOS amd64 runner.
- Pass those reports into the Linux aggregation gate with
  `TETRA_WINDOWS_UI_RUNTIME_REPORT=/path/windows-ui-runtime.json` and
  `TETRA_MACOS_UI_RUNTIME_REPORT=/path/macos-ui-runtime.json`.

Manual target-host evidence:

- Check out the same Git commit on a real Windows amd64 host and run
  `bash scripts/release/full_platform/target-host-ui-runtime-smoke.sh
  --target windows-x64 --report windows-ui-runtime.json`.
- Check out the same Git commit on a real macOS amd64 host and run
  `bash scripts/release/full_platform/target-host-ui-runtime-smoke.sh
  --target macos-x64 --report macos-ui-runtime.json`.
- Copy those JSON reports to the Linux aggregation host and run
  `TETRA_WINDOWS_UI_RUNTIME_REPORT=/path/windows-ui-runtime.json
  TETRA_MACOS_UI_RUNTIME_REPORT=/path/macos-ui-runtime.json
  bash scripts/release/full_platform/ui-runtime-gate.sh --report-dir
  reports/full-platform-ui-runtime`.
- A GitHub Actions `startup_failure` with zero jobs is an infrastructure or
  account/repository availability blocker. It does not relax the evidence
  contract: use a working CI runner, self-hosted target-host runner, or manual
  target-host reports produced from the same Git commit.
- To record that blocker as diagnostic only evidence, run
  `bash scripts/release/full_platform/github-actions-startup-diagnostic.sh
  --repo OWNER/REPO --branch BRANCH --report
  reports/full-platform-ui-runtime/github-actions-startup-blocker.json`, then
  `go run ./tools/cmd/validate-actions-startup-blocker --report
  reports/full-platform-ui-runtime/github-actions-startup-blocker.json`.
  This report proves only that CI did not start jobs; it never replaces
  Windows/macOS runtime reports.

The wrappers copy those reports into the fresh report directory and re-run the
strict validators before the cross-platform gate accepts them. The validators
also require the report `version` and `git_head` to match the source checkout
used by the aggregation gate, so stale target-host evidence is rejected. The
target-host reports must include runtime trace markers for process spawn,
an OS-backed platform window API probe, platform widget tree construction,
platform event dispatch, platform timer/redraw work, window create/show/close,
widget tree load, layout, event dispatch, state updates, async commands,
timers, redraw, and error recovery.

The repository workflow `.github/workflows/full-platform-ui-runtime.yml`
automates that contract for GitHub Actions: `windows-2025` produces the
`windows-x64` report, `macos-15-intel` produces the `macos-x64` report, and an
`ubuntu-24.04` aggregation job downloads those reports and runs
`scripts/release/full_platform/ui-runtime-gate.sh`. The local Linux aggregation
gate remains intentionally strict: without target-host reports from those real
runner jobs, Windows and macOS stay blocked and cannot count as production
runtime evidence.

The target-host child probe uses real platform APIs: Win32 `user32.dll` controls
and messages on Windows and an AppKit window/control probe compiled with
`swiftc` on macOS. If those APIs are unavailable on the runner, the platform
report stays non-production and the validator rejects it.
