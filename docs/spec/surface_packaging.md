# Surface Packaging Evidence

Status: experimental Surface distribution evidence for the scoped Linux package
path. This is not a Windows installer production claim, not a macOS production
claim, and not an auto-update production claim.

`tetra.surface.package-report.v1` records the packaging boundary for
Surface apps that need distribution evidence rather than renderer-only proof.
The current level is `surface-package-distribution-v1` under
`surface-v1-scoped-linux-web-package`.

The supported package gate is:

```sh
bash scripts/release/surface/package-gate.sh \
  --report-dir reports/surface-prod/P26-package-gate
```

The gate creates a `tetra new surface-app` scaffold, runs
`tetra surface check`, produces a `.tdx` app package with
`tetra surface package`, builds a deterministic `surface-linux-tar-v1` archive,
extracts it for install smoke, runs the package launcher smoke, generates a
package report with `surface-package-report`, validates it with
`validate-surface-package-report`, and writes artifact hashes.

Required package evidence:

- Linux target is `linux-x64` and support level is `production` only for the
  scoped tar package path.
- Package files include the `.tdx` Surface app package, `surface-package.json`,
  `assets/surface-assets.json`, `permissions.json`, `host-adapter.json`, app
  source files, and `bin/surface-run.sh`.
- Every package file has sha256 and size evidence.
- The tar archive contains every declared package file with matching sha256 and
  size.
- Install smoke and launcher smoke are true for the same commit.
- Signature evidence is `sha256-checksum-manifest`; platform signing is not
  claimed by this Linux tar package.

Required rejection cases:

- unsigned macOS production package claims are rejected until signed and
  notarized bundle evidence exists;
- Windows production package claims are rejected until signed installer evidence
  exists;
- omitted package assets are rejected by tar/archive content validation;
- updater production claims are rejected without a defined update channel and
  signature verification.

Auto-update is a separate tier. It remains `separate-tier-nonclaim` until a
signed channel manifest and signature verification evidence exist.
