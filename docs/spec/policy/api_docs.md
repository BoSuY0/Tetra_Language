# API Docs Metadata

`tools/cmd/gen-docs` emits Markdown API docs with a small machine-readable metadata comment directly
below the top-level heading:

```md
# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:<hex>","module_count":1,"entry_count":1} -->
```

The metadata is intentionally alpha-scoped. It gives release gates and Eco-local tooling a stable
prototype for API diff checks without defining a final package registry format.

## Fields

- `schema`: metadata schema. The current accepted value is `tetra.api.v1alpha1`.
- `api_hash`: SHA-256 hash of the documented public API surface, prefixed with `sha256:`.
- `module_count`: number of documented module headings.
- `entry_count`: number of documented API entry bullets.

## Hash Surface

The hash input is the newline-joined list of:

- every `## <module>` heading
- every API entry bullet that starts with ``- ` ``

This keeps the prototype stable across prose-only documentation edits while still detecting public
API additions, removals, and signature changes.

`tools/cmd/validate-api-docs` rejects docs with missing metadata, unsupported schemas, invalid hash
prefixes, mismatched counts, or mismatched hashes.

## API Diff Policy

The baseline artifact format, diff schema, and release-gate command contract are defined in
[API Diff Policy](api_diff_policy.md).
