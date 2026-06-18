# Surface I18n Localization Design

## Approval

- Source: active goal to implement
  `/home/tetra/Downloads/2026-06-10-tetra-surface-electron-react-beauty-production-implementation-plan.md`.
- Packet: `SURFACE-BEAUTY-P25`.
- Human decision needed: none recorded in
  `.workflow/surface-electron-react-beauty-production/CONTROL.md`.

## Observed Facts

- The current localized form reference app exists at
  `examples/surface_reference_localized_form.tetra`, but it only proves the
  reference shape through Surface, Block, and Morph.
- `lib/core/text.tetra` already keeps full bidi shaping unsupported through
  `full_bidi_supported() == false`.
- Release evidence is validator-gated through `tools/validators/surface`,
  `tools/cmd/validate-surface-release-state`, and
  `scripts/release/surface/release-gate.sh`.
- Current release summary fields already require separate evidence markers for
  templates, reference apps, packages, and crash reporting.

## Design

- Add a small `lib.core.i18n` API for bounded product UI localization:
  locale records, string table entries, fallback lookup, missing-key diagnostic
  codes, deterministic formatting hook records, and explicit text-direction
  records.
- Keep the API intentionally below full ICU: no plural rules, no shaping engine,
  no full bidi production claim, and no locale data bundle claim.
- Update the localized-form reference app to use `lib.core.i18n` while still
  resolving through Morph recipes to Block.
- Add `tetra.surface.i18n.v1` evidence with schema/model
  `surface-i18n-v1`. The report must prove `en-US` and `uk-UA` string tables,
  locale selection, fallback from `uk-UA` to `en-US`, missing-key diagnostics,
  formatting hooks, localized form execution, and RTL placeholder nonclaim.
- Wire the report into the release summary, release-state validator, release
  gate, generated manifest, docs, and artifact hashes.

## Failure Rules

- Missing fallback evidence fails.
- Missing-key diagnostics fail.
- Localized form evidence without the reference app passing fails.
- Any full ICU, full bidi shaping, or RTL production claim fails unless a
  future shaping proof exists.
- Reports that bypass the release gate or manifest feature fail release-state
  validation.

## Verification

- RED tests for the new i18n report validator, release summary field,
  release-state file/feature enforcement, and gate script wiring.
- GREEN targeted Go tests for validators, CLI, release-state, script structure,
  compiler semantics, and affected docs/manifest checks.
- Real `surface-i18n-smoke.sh` evidence under
  `reports/surface-electron-react-beauty-production/P25/`.
- Full Surface release gate and claim scanner after integration.
