# Surface macOS Target Validator

`tools/validators/surfacemacos` validates `tetra.surface.macos-target.v1`
boundary reports.

The package owns the macOS production/nonclaim boundary for Surface target-host
evidence. It keeps macOS out of the scoped production claim unless real
target-host, distribution, accessibility, and input evidence satisfy the
validator's target-specific requirements.
