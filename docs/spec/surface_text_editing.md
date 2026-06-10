# Tetra Surface Text Editing

Status: current scoped production editing-basics evidence for the Surface v1
text-input release path.

This document defines `tetra.surface.text-editing.v1`, the evidence block
embedded in `tetra.surface.text-input.v1` reports after `SURFACE-PROD-P09`.
It complements `tetra.surface.text-pipeline.v1`: the pipeline proves glyph and
measurement evidence, while text editing proves owned editable storage,
selection/caret behavior, target IME traces, clipboard ownership, undo unit
boundaries, and fake-claim rejection diagnostics.

## Scope

The supported editing-basics scope is intentionally narrow:

- editable `TextBox` blocks backed by owned UTF-8 byte buffers;
- app-grade forms and command palette search input;
- caret movement for left, right, home, and end;
- selection replacement and scalar-boundary clamping;
- target-specific IME composition start, update, commit, and cancel traces;
- clipboard read/write transfers through the Surface Host ABI using owned
  copies only;
- undo unit boundaries for text insertion, caret navigation, selection
  replacement, IME commit, clipboard copy, and clipboard paste;
- validation diagnostics for invalid UTF-8, borrowed text buffers crossing host
  boundaries, missing target IME traces, and rich text overclaims.

## Required Evidence

Every production text-input report must include:

- `schema = "tetra.surface.text-editing.v1"`;
- `level = "production-editing-basics-v1"`;
- `target` matching the parent `tetra.surface.text-input.v1` report target;
- `editable_blocks` with owned UTF-8 storage, form safety, command palette
  search safety, max byte budget, and UTF-8 validation;
- `edit_operations` for insert, caret navigation, selection replacement, IME
  commit, clipboard write, and clipboard read;
- `selection_model` with caret movement coverage, selection replacement,
  scalar boundary clamp, caret rectangles, and selection rectangles;
- `ime_traces` proving composition events for the reported target;
- `clipboard_transfers` proving `__tetra_surface_clipboard_write_text` and
  `__tetra_surface_clipboard_read_text_into` use UTF-8 owned copies and no
  borrowed views;
- `undo_units` with operation order links, named boundaries, reversibility, and
  coalescing metadata;
- `validation_diagnostics`, `host_boundary`, `nonclaims`, and
  `negative_guards` proving unsafe aliases and unsupported claims are rejected.

## Nonclaims

The editing-basics contract does not claim rich text, full editor-grade text
semantics, native platform text controls, or full Unicode editor semantics.
Those remain unsupported until later target evidence defines a wider text
model, shaping tier, accessibility model, and editor test corpus.

## Validator

`ValidateTextInputReport` rejects production text-input reports that omit or
weaken `text_editing`. The validator rejects:

- IME claims without a trace for the report target;
- borrowed text buffers crossing the host boundary;
- clipboard transfers that expose borrowed views or skip owned-copy evidence;
- missing undo unit boundaries;
- rich text claims inside the editing-basics release scope;
- missing validation diagnostics or negative guards.

Release text-input smoke modes generate the block for headless, `linux-x64`,
and `wasm32-web` targets through:

```sh
go run -buildvcs=false ./tools/cmd/surface-runtime-smoke \
  --mode headless-release-text-input \
  --source examples/surface_release_text_input.tetra \
  --report reports/surface-prod/text-input.json
```
