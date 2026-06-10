# Surface Crash Diagnostics

Status: experimental production-candidate evidence for the Block System track.
It is not a claim of automatic crash recovery, telemetry upload, external crash
reporter integration, or Electron crash reporter compatibility.

`tetra.surface.crash-report.v1` records the structured crash diagnostics and
error reporting evidence required for Surface apps when a command, background
service, or runtime path fails. The required quality level is
`surface-crash-diagnostics-v1`.

## Contract

The report is valid only inside the scoped
`PROD_STABLE_SCOPED_LINUX_WEB_APP_UI` release boundary. It must compose with
the P24 dev-loop, P27 security, and P28 IPC/lifecycle reports for the same
commit before any production platform claim can use it.

Required policies:

- `crash-safe-diagnostics-v1`: crashes produce structured diagnostics instead
  of being silently swallowed.
- `supervised-restart-opt-in-v1`: recovery is scoped and explicit, not a broad
  automatic crash recovery promise.
- development panic/error overlay exists only for dev-loop evidence;
- production error hook records source locations, diagnostic code/severity,
  sanitized diagnostic bundles, and user/caller surfacing evidence;
- secret scrubbing is required before any report is accepted;
- expected negative cases are reported separately from real runtime crashes.

Required crash entries include source file, line, column, function, stable
diagnostic code, severity, message, recovery action, nonzero failure exit code
for unexpected crashes, secret-scan evidence, and a diagnostic bundle artifact
with sha256 and size.

## Fake-Claim Rejection

The crash diagnostics validator rejects:

- a crash counted as a pass;
- a swallowed crash;
- an error that is not surfaced to the user or caller;
- a report that includes secret material;
- a crash without source location;
- a crash without diagnostic bundle artifact;
- a production dev overlay;
- crash artifacts that are not tied to the same commit.

These reports distinguish expected negative validation cases from real crashes
so release gates do not hide bugs behind successful negative tests.

## Tetra API

`lib.core.surface` exposes compact crash diagnostics helpers:

- `SurfaceCrashPolicy`, `crash_policy`, and `crash_policy_valid`;
- `crash_report_valid`.

The helpers are policy values for code and report alignment. They do not
implement telemetry, upload external crash dumps, or promote crash recovery
beyond the scoped restart policy.
