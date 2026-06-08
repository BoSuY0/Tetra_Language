# UI Toolkit Core Validator

`tools/validators/uitoolkit` validates `tetra.ui.toolkit.v1` production
evidence for the backend-independent UI Toolkit Core.

It requires runtime trace artifacts, widget/layout/event/state/accessibility
coverage, runtime and stress process evidence, negative diagnostics, and audit
records. It rejects docs-only, metadata-only, preview-only, runtime-less,
native-shell sidecar-only, web-only, build-only, fake, mock, and placeholder
evidence.
