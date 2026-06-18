# Surface macOS Target Boundary

Status: unsupported for Surface v1 production; beta only with real macOS
target-host evidence.

`tetra.surface.macos-target.v1` is the Surface-specific macOS boundary report.
It prevents build-only macOS artifacts, Linux-host synthetic reports, generic
platform UI reports, non-notarized production distribution claims, and full
accessibility claims without a screen-reader bridge from being counted as macOS
Surface target-host evidence.

## Legal States

The validator accepts two states:

- `status:"nonclaim"` with `support_level:"unsupported"` and
  `evidence_kind:"nonclaim-boundary"`;
- `status:"beta"` with `support_level:"beta-target-host"` and
  `evidence_kind:"target-host-surface-beta"`.

Production is intentionally rejected in P18. A macOS production claim requires
future target-host Surface evidence plus later accessibility and packaging
gates.

## Beta Evidence

A beta macOS target-host report must have:

- `target:"macos-x64"`;
- `host:"macos-x64"`;
- `surface_schema:"tetra.surface.v1"`;
- `app_shell_abi:"tetra.surface.app-shell.v1"`;
- native window, native input, clipboard, IME, DPI, menu bar, dialogs,
  notifications, accessibility bridge, screen-reader bridge, and app-shell
  capability evidence;
- process evidence from a target-host runtime smoke;
- packaging scope that does not claim production distribution.

Generic `tetra.ui.v1` platform UI runtime reports are not Surface production
evidence.

## Rejection Rules

The validator rejects:

- `support_level:"production"` or `production_claim:true`;
- `evidence_kind:"build-only"`;
- Linux-host synthetic reports claiming macOS target-host evidence;
- generic platform UI runtime evidence promoted as Surface evidence;
- non-notarized production distribution claims;
- full accessibility claims without screen-reader bridge evidence.

Primary CLI:

```text
go run ./tools/cmd/validate-surface-macos-target --report reports/surface-prod/P18-macos-target/macos-surface-boundary.json
```

The current repository evidence is a nonclaim boundary report, not a macOS
target-host pass.
