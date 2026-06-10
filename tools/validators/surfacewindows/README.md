# Surface Windows Target Validator

`tools/validators/surfacewindows` validates `tetra.surface.windows-target.v1`
boundary reports.

The package owns the Windows production/nonclaim boundary for Surface
target-host evidence. It keeps Windows out of the scoped production claim until
real target-host, input, accessibility, package, and lifecycle evidence satisfy
the validator's target-specific requirements.
