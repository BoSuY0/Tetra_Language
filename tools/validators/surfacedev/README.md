# Surface Dev Loop Validator

`tools/validators/surfacedev` validates `tetra.surface.dev-loop.v1` evidence
for `surface-fast-dev-loop-v1` plus `tetra.surface.template.v1` template
metadata.

The package owns hot reload, source mapping, template, diagnostics, and
schema-compatible owned-state preservation checks. It rejects missing template
evidence and any claim that incompatible state schemas are preserved.
