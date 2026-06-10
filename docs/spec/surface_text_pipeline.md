# Tetra Surface Text Pipeline

Status: current scoped Tier 1 text/glyph evidence for the Surface v1
text-input release path.

This document defines `tetra.surface.text-pipeline.v1`, the evidence block
embedded in `tetra.surface.text-input.v1` reports after `SURFACE-PROD-P08`.
Editing behavior, selection operations, target IME traces, clipboard owned-copy
transfers, undo boundaries, and rich text nonclaim enforcement are defined
separately in `docs/spec/surface_text_editing.md`.

The text pipeline is intentionally scoped. It proves deterministic Tier 1
Latin/UTF-8 text handling for the release text-input path, including font
manifest hashes, fallback chain, glyph runs, bounded glyph cache, Unicode scalar
and cluster boundary records, text measurement consistency, wrap/ellipsis/
alignment/baseline evidence, caret and selection rectangles, and IME
composition span geometry.

## Scope

The supported Tier 1 scope is:

- UTF-8 byte storage owned by Tetra Surface components;
- Latin and Common script glyph runs;
- deterministic fallback to the reported fallback family;
- deterministic measurement for repeated inputs;
- wrap, ellipsis, alignment, baseline, and line-height evidence;
- caret rectangle, selection rectangle, and IME composition span reports;
- bounded glyph cache with byte budget and eviction policy.

The current shaping engine is recorded as `deterministic-tetra-text-shaper`.
That engine is enough for the scoped Tier 1 release claim, but it is not a
HarfBuzz-class general shaping engine.

## Nonclaims

The report must carry exact nonclaims for unsupported text scope:

- full Unicode editor semantics;
- bidi production shaping;
- complex script shaping without HarfBuzz-class evidence;
- platform widget text controls.

Arabic, Devanagari, Thai, and other complex scripts remain unsupported in this
Tier 1 report until later target evidence proves a wider shaping tier.
Combining marks and bidi are Tier 2 topics. Full editor-grade shaping,
selection, and language behavior are Tier 3 topics.

## Required Evidence

Every production text-input report must include:

- `schema = "tetra.surface.text-pipeline.v1"`;
- `level = "scoped-latin-utf8-text-pipeline-v1"`;
- `font_manifest` entries with family, source, size, and SHA-256 hash;
- `font_fallbacks` with requested family, resolved family, fallback chain,
  coverage, and `missing_glyphs = 0` for the smoke fixture;
- `glyph_runs` with font family, script, direction, shaping tag, byte/scalar
  ranges, glyph ids, advances, clusters, baseline, and checksum;
- `glyph_caches` plus `glyph_cache_budget_bytes`,
  `glyph_cache_used_bytes`, bounded-cache flag, and eviction policy;
- `unicode_boundaries` for UTF-8 storage, scalar boundaries, cluster
  boundaries, unsupported scripts, and boundary cases;
- `shaping_scope` with Tier 1 support, unsupported scripts, and
  HarfBuzz-class future evidence decision;
- measurement, layout, caret, selection, and IME composition span evidence;
- `negative_guards` proving full-Unicode overclaim, missing fallback,
  unbounded glyph cache, and platform-widget text-control claims are rejected.

## Validator

`ValidateTextInputReport` rejects production text-input reports that omit the
`text_pipeline` block or weaken its scoped evidence. The validator rejects:

- missing font fallback diagnostics;
- unbounded glyph cache;
- full Unicode editor semantics claims without Tier 3 tests;
- unsupported script promotion without nonclaims;
- platform widget text-control evidence.

The release text-input smoke modes generate the block for headless,
`linux-x64`, and `wasm32-web` targets through:

```sh
go run -buildvcs=false ./tools/cmd/surface-runtime-smoke \
  --mode headless-release-text-input \
  --source examples/surface_release_text_input.tetra \
  --report reports/surface-prod/text-input.json
```
