# Surface Linux Host Adapter

Status: current scoped linux-x64 production host-adapter evidence for Surface v1.

`tetra.surface.linux-host-adapter.v1` is the Linux target-host evidence object
for `linux-x64-release-window-v1`. It connects the already validated
real-window runtime report to the app-shell ABI, IME/composition, clipboard,
accessibility bridge, and packaging-scope evidence needed before Linux can be
treated as the first production desktop Surface target.

## Required Report Shape

The runtime report must include:

- `target:"linux-x64"` and `runtime:"surface-linux-x64"`;
- `host_evidence.level:"linux-x64-release-window-v1"`;
- `linux_host_adapter.schema:"tetra.surface.linux-host-adapter.v1"`;
- `linux_host_adapter.level:"linux-x64-production-host-adapter-v1"`;
- `backend:"wayland-shm-rgba-release-v1"`;
- true real-window, framebuffer, native-input, text-input, IME, clipboard,
  composition, accessibility-bridge, and app-shell booleans;
- `app_shell_abi:"tetra.surface.app-shell.v1"`;
- `packaging.scope:"linux-x64-unpacked-binary-v1"`.

The packaging scope is intentionally narrow. P16 proves an unpacked Linux
binary artifact can ship the scoped Surface app shell. It does not claim
installers, package repository integration, signing, notarization, or
auto-update. Those remain later packaging/security gates.

## Target-Host Traces

The adapter requires delivered target-host traces for:

- real window;
- framebuffer presentation;
- native input;
- text input;
- IME/composition;
- clipboard;
- accessibility bridge;
- app shell ABI;
- packaging scope.

The traces must be tied to concrete runtime artifacts such as the release-window
probe, clipboard harness, composition harness, accessibility bridge artifact, or
the produced component app binary. Metadata-only reports are not sufficient.

## Rejection Rules

The validator rejects:

- deterministic offscreen evidence promoted as Linux production evidence;
- blocked display runs counted as pass;
- old `linux-x64-real-window` evidence promoted to
  `linux-x64-release-window-v1`;
- missing app-shell ABI evidence;
- missing packaging scope;
- missing target-host traces.

When no `WAYLAND_DISPLAY` or `DISPLAY` is available, the release-window script
must write a blocked report with `production_claim:false` and exit non-zero. A
blocked target host is useful evidence about the environment, not production
pass evidence.

## Gate

Primary gate:

```text
bash scripts/release/surface/surface-linux-x64-release-window-smoke.sh --report-dir reports/surface-prod/linux
```

The strict runtime validator must accept only a same-run report that includes
`linux_host_adapter` evidence and all existing Linux release-window frame,
input, clipboard, composition, accessibility, toolkit, and artifact checks.
