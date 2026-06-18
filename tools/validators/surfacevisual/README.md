# Surface Visual Regression Validator

`tools/validators/surfacevisual` validates
`tetra.surface.visual-regression.v1` evidence for
`surface-visual-golden-v1`.

The package owns golden baseline, screenshot/frame, checksum, diff threshold,
target, and artifact evidence checks for Surface visual regression. It rejects
fake goldens, metadata-only visual proof, and stale artifact evidence.
