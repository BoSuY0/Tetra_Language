# Surface App-Shell Electron Feature Ledger

Goal: complete `SURFACE-BEAUTY-P16` by accounting for Electron-like app-shell
features through the existing Surface app-shell evidence path, without adding
Electron, React, DOM UI, user JavaScript app logic, or platform-native widgets
as the Surface UI layer.

## Observed Facts

- `lib/core/surface_app_shell.tetra` already defines scoped app-shell helpers
  for `ShellWindow` and `ShellFeature`.
- `tools/validators/surface/report.go` already validates
  `tetra.surface.linux-app-shell.v1` reports through `LinuxAppShellReport`.
- `tools/cmd/surface-runtime-smoke --mode linux-x64-release-app-shell` already
  emits `surface-linux-x64-release-app-shell.json`.
- `scripts/release/surface/surface-linux-x64-release-app-shell-smoke.sh` and
  `scripts/release/surface/release-gate.sh` already make the Linux app-shell
  report mandatory for the Surface v1 release gate.
- Current P12 evidence covers lifecycle, multi-window notes, resize/DPI/cursors,
  clipboard, IME, accessibility bridge, `app_menu` as a scoped adapter, and
  blocked-pass nonclaims for `file_dialog` and `notification`.

## Design

Extend the existing `linux_app_shell` report into an Electron feature ledger.
Do not create a second app-shell report schema for P16 unless a later packet
needs a cross-target product report. The ledger stays target-scoped:

- Linux supported/scoped rows:
  `app_menu`, `window_lifecycle`, `multi_window`, `clipboard`, `ime`,
  `accessibility_bridge`, `crash_recovery`, and `error_report`.
- Linux blocked/nonclaim rows:
  `dialog`, `file_picker`, `notification`, `tray`, and `deep_link`.
- Unsupported target rows:
  Windows/macOS remain owned by `tetra.surface.target-host-status.v1`; P16 must
  not turn their build-only artifacts into app-shell support.

Each ledger row must carry target, status, claimed, host trace, blocked reason
where needed, and no-native-widget evidence. Supported rows need local trace
artifact evidence. Blocked rows must be `blocked_pass`, `claimed:false`, and
carry an explicit blocked reason.

## Implementation Plan

1. Add RED tests in `tools/validators/surface/report_test.go` requiring
   `dialog`, `file_picker`, `tray`, `crash_recovery`, and `error_report`
   feature rows, and rejecting tray/notification/file-picker claims without
   target evidence.
2. Add RED tests in `tools/cmd/validate-surface-runtime/main_test.go` or the
   existing release validator path so `--release linux-app-shell` requires the
   P16 ledger rows.
3. Extend `LinuxAppShellFeatureReport` and validation in
   `tools/validators/surface/report.go` only as needed for typed feature
   ledger checks.
4. Extend `tools/cmd/surface-runtime-smoke/main.go` and
   `collectLinuxAppShellTraceEvidence` so generated reports and host trace
   artifacts include the P16 ledger.
5. Update `lib/core/surface_app_shell.tetra` and smoke examples with helper
   constants for the additional feature rows.
6. Update Surface docs, feature registry, and generated manifest to describe
   the supported subset and explicit blocked/nonclaims.
7. Generate fresh P16 release evidence and validate release-state, artifact
   hashes, claims, docs, manifest, package tests, and Graphify.

## Acceptance

P16 is complete when the release gate emits a validated app-shell feature ledger
that accounts for menus, dialogs, file picker, notifications, tray, lifecycle,
crash, and error behavior, with supported Linux evidence where claimed and
blocked/nonclaim rows where unsupported.

## Nonclaims

This design does not claim broad Electron parity, platform-native widget UI,
Linux tray/notification/file-picker/dialog support without target evidence,
Windows/macOS app-shell support, deep-link support, crash upload service, or
production security/permission coverage. Security and permissions remain P17.
