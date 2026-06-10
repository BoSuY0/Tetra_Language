# Surface Package Validator

`tools/validators/surfacepackage` validates
`tetra.surface.package-report.v1` evidence for
`surface-package-distribution-v1`.

The package owns package manifest, artifact hash, permission file,
target-adapter, signing, distribution, and update-channel checks for scoped
Surface packaging. It rejects unsigned macOS production and updater claims
without channel-signature evidence.
