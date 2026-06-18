# Surface I18n Validator

`tools/validators/surfacei18n` validates `tetra.surface.i18n-report.v1`
evidence for `surface-i18n-l10n-v1`.

The package owns locale-resource, string ID, formatting policy, layout
direction, and localized text checks for the scoped Linux/web Surface boundary.
It rejects full ICU, full Unicode editor, full bidi shaping, and unsupported
host locale production claims.
