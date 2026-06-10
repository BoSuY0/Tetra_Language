# Surface Production Claim Validator

`tools/validators/surfaceprod` validates `tetra.surface.prod-claim.v1` claim
reports for `PROD_STABLE_SCOPED_LINUX_WEB_APP_UI` within
`surface-prod-scoped-linux-web`.

The package owns final claim governance for the scoped Surface production tier.
It checks capability claims, supported targets, target-host evidence,
nonclaims, and artifact evidence while rejecting fake or paper-only Electron,
React, CSS, GPU, accessibility, and cross-platform overclaims.
