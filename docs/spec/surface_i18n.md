# Surface Internationalization And Localization

Status: experimental production-candidate evidence for the Block System track.
It is not a full bidi production shaping claim, a full ICU/CLDR database claim,
or platform-native localization framework parity.

`tetra.surface.i18n-report.v1` records the locale resources, stable string IDs,
formatting hooks, translation asset packaging, and layout direction metadata
required for a basic localized Surface app. The required quality level is
`surface-i18n-l10n-v1`.

## Contract

The report is valid only inside the scoped
`PROD_STABLE_SCOPED_LINUX_WEB_APP_UI` release boundary. It must compose with
the P08 text pipeline, P21 asset pipeline, and P26 package report for the same
commit before any production platform claim can use it.

Required policies:

- `surface-i18n-l10n-hooks-v1`: locale resources are explicit app/package
  assets and every user-facing string has a stable string ID.
- `diagnostic-required`: missing locale resources produce diagnostics; missing
  locale resource silent fallback is rejected.
- number/date/plural formatting hooks exist as deterministic hooks, not as a
  full ICU/CLDR compatibility claim.
- translation asset packaging records locale manifest and locale resource
  hashes in the same-commit artifact set.
- LTR and RTL layout direction metadata is recorded for scoped layout behavior.
- full bidi shaping stays outside P30 unless future shaping-tier evidence
  explicitly promotes it.

Required locale resources include a default locale, at least one non-English
locale, and at least one RTL direction locale. Each resource must be packaged,
hash-addressed, and carry the required string IDs.

## Fake-Claim Rejection

The i18n validator rejects:

- full bidi claim without shaping evidence;
- missing locale resources;
- silent fallback for missing locale resources;
- missing string IDs;
- unpackaged translation assets;
- unsupported host localization production claims;
- full ICU/CLDR claims.

## Tetra API

`lib.core.i18n` exposes compact localization helpers:

- `Locale`, `StringID`, `LocaleResource`, `FormatPolicy`, and
  `LocalizationPolicy`;
- `locale_en_us`, `locale_es_es`, and `locale_ar_eg`;
- `locale_valid`, `string_id_valid`, `locale_resource_valid`,
  `format_policy_valid`, and `localization_policy_valid`;
- `localized_text_len` for report-aligned string-ID lookups in examples.

These helpers do not implement a full locale database or editor-grade Unicode
semantics. They provide explicit typed hooks that keep localization resources
visible to Surface validators and package gates.
