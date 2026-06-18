# Surface Windows Target Boundary

Status: unsupported for Surface v1 production; beta only with real Windows
target-host evidence.

`tetra.surface.windows-target.v1` is the Surface-specific Windows boundary
report. It exists to prevent build-only Windows artifacts or Linux-host
synthetic reports from being counted as Windows Surface target-host evidence.

## Legal States

The validator accepts two states:

- `status:"nonclaim"` with `support_level:"unsupported"` and
  `evidence_kind:"nonclaim-boundary"`;
- `status:"beta"` with `support_level:"beta-target-host"` and
  `evidence_kind:"target-host-surface-beta"`.

Production is intentionally rejected in P17. A Windows production claim requires
future target-host Surface evidence plus later accessibility and packaging gates.

## Beta Evidence

A beta Windows target-host report must have:

- `target:"windows-x64"`;
- `host:"windows-x64"`;
- `surface_schema:"tetra.surface.v1"`;
- `app_shell_abi:"tetra.surface.app-shell.v1"`;
- native window, native input, clipboard, IME, DPI, menus, dialogs,
  notifications, accessibility bridge, and app-shell capability evidence;
- process evidence from a target-host runtime smoke.

Generic `tetra.ui.v1` platform UI runtime reports are not Surface production
evidence.

## Rejection Rules

The validator rejects:

- `support_level:"production"` or `production_claim:true`;
- `evidence_kind:"build-only"`;
- Linux-host synthetic reports claiming Windows target-host evidence;
- generic platform UI runtime evidence promoted as Surface evidence.

Primary CLI:

```text
go run ./tools/cmd/validate-surface-windows-target --report reports/surface-prod/P17-windows-target/windows-surface-boundary.json
```

The current repository evidence is a nonclaim boundary report, not a Windows
target-host pass.
