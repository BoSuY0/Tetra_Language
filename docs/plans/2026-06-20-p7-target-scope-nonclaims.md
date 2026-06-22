# P7/P8 Target-Scope Non-Claim Artifact

Date: 2026-06-20

## Goal

Make the P7 compiler RSS evidence bundle machine-readable about target scope, so Linux x64 process
RSS evidence cannot be mistaken for cross-target parity evidence.

## Implementation Plan

- Add a RED expectation that `ramcompilerrss.Run` writes `target-scope.json` and references it from
  `compiler-rss-manifest.json`.
- Record the host target, compiler target, one measured Linux x64 entry, and explicit non-claim
  entries for targets that require target-specific evidence.
- Regenerate the representative samples5 bundle and validate artifact hashes.
- Keep final P7/P8 open for full-repo RSS and actual target parity evidence.

## Evidence

The regenerated bundle is:

```text
reports/stabilization/tetra-ram-p7-compiler-rss-b452638a8af7-representative-samples5/
```

It includes `target-scope.json` with schema
`tetra.ram.p7-compiler-rss-target-scope.v1`. The artifact records:

```text
host_target:     linux/amd64
compiler_target: linux-x64
measured target: linux-x64 / host_rss_measured
non-claim targets:
  windows-x64
  macos-x64
  macos-arm64
  linux-x86
  linux-x32
  wasm32-wasi
  wasm32-web
```

The regenerated representative report comparison still passes:

```text
report-off median: 56918016
report-on median:  58900480
bound:             68329472
delta:             1982464
ratio:             1.0348
```

`artifact-hashes.json` contains `target-scope.json` with schema
`tetra.ram.p7-compiler-rss-target-scope.v1` and validates successfully.

## Boundary

This closes the machine-readable target non-claim artifact for the current Linux x64 compiler RSS
bundle only. It does not prove Windows, macOS, Linux 32-bit, WASM, full-repo compiler RSS, or final
P8 parity acceptance.
