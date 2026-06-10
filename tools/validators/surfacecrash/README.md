# Surface Crash Validator

`tools/validators/surfacecrash` validates `tetra.surface.crash-report.v1`
evidence for `surface-crash-diagnostics-v1`.

The package owns crash diagnostics, restart policy, report artifact, and
negative-guard checks for the scoped Linux/web Surface production boundary. It
rejects docs-only, fake crash-report, missing artifact, and unsupported broad
desktop production claims.
