# Tetra v0.6.0 Usable Alpha Release Notes

Tetra v0.6.0 is a hardening release over v0.5.0 Integrated Alpha. It keeps the
same broad language scope and improves release identity, formatting coverage,
LSP basics, and local Eco packaging.

## Highlights

- `tetra version` reports `v0.6.0`; generated manifest metadata matches.
- `tetra fmt --check examples lib __rt compiler/selfhostrt` and
  `validate-flow-only` are part of the release gate.
- `tetra lsp --stdio` supports a minimal JSON-RPC stdio loop:
  initialize/shutdown, didOpen diagnostics, document symbols, and hover.
- `tetra eco pack --project` creates a local project bundle and preserves the
  existing single-manifest pack behavior.
- `scripts/release_v0_6_gate.sh` is the canonical final verification command.
- `scripts/test_all.sh` adds a v0.6.x stabilization wrapper with quick/full
  modes, per-step logs, and Markdown/JSON summaries.
- `docs/roadmap_0_6_x_stabilization.md` records the 0.6.x stabilization line:
  test envelope hardening, negative coverage, and cross-platform confidence.
- The 0.6.x wrapper now supports `--keep-going` and `--json-only` for fuller
  stabilization reports and machine-readable CI/editor integration; each JSON
  step includes its command line and exit code, and the summary includes
  top-level step and failure counts.
- `tools/cmd/validate-test-all-summary` validates `summary.json` counts,
  per-step command/exit/status consistency, unique step names/logs, and
  referenced log files.
- Report validators now reject duplicate smoke source paths, invalid smoke
  exit codes, incomplete test-result synthetic function metadata, and
  unsorted or malformed generated API docs.
- Docs verification now rejects unterminated `tetra doctest` fences instead of
  silently ignoring the dangling block.
- LSP validators now require stable diagnostics severity values, matching
  symbol/hover positions, diagnostic URIs, and initialize capabilities for
  hover/document symbols.
- Release gates now validate a small JSON diagnostic matrix covering semantic
  unknown calls, missing `uses io`, Flow tab indentation, planned actor syntax,
  and planned WASM targets.
- `tools/cmd/validate-diagnostic` now rejects unknown severity values and can
  require file/line/column positions for source diagnostics.
- `tools/cmd/validate-eco-lock` now rejects unsupported targets, duplicate
  target entries, duplicate dependencies, invalid capsule ids, and
  self-dependencies in local capsule graphs.
- `tools/cmd/validate-eco-unpack` now parses every unpacked `.tetra` source so
  malformed bundle contents fail local Eco verification.
- `tools/cmd/validate-eco-vault` now validates the allowed vault kind set and
  duplicate record identity while allowing the same object hash to back
  different record kinds.
- `tools/cmd/smoke-report-to-checklist --validate-only` now enforces supported
  targets, version shape, unique `.tetra` smoke sources, exit-code ranges, and
  ran/pass exit-code consistency.
- `tools/cmd/validate-flow-only` now rejects tabs and standalone legacy brace
  tokens while ignoring those characters inside line comments and string
  literals.
- `scripts/test_all.sh` keeps summary validation strict on passing runs, while
  preserving the original failure summary if the validator itself is
  unavailable on a failing run.
- `tools/cmd/validate-manifest` validates generated docs manifest structure,
  exact supported target coverage, builtin metadata, and actor runtime ABI
  symbol coverage in release gates.
- JSON diagnostics distinguish parser/frontend errors (`TETRA0001`) from
  positioned semantic/compiler errors (`TETRA2001`) while preserving existing
  text diagnostic output.
- CLI commands with `--diagnostics` now reject unsupported modes instead of
  silently falling back to text output.
- `tetra run --diagnostics=json` and `tetra test --diagnostics=json` now report
  host/target execution mismatches as structured JSON diagnostics.
- Diagnostics-enabled CLI validation paths such as extra build inputs,
  invalid formatter mode combinations, and unsupported test report formats now
  honor `--diagnostics=json`.
- `tetra test --report=json` now emits `files: []` and `results: []` for empty
  test suites instead of `null`.
- `tools/cmd/validate-test-report` validates `tetra test --report=json` shape
  aggregate counts, non-negative durations, and duplicate test names per file;
  `test_all --full` and `release_v0_6_gate.sh` use it.
- `tools/cmd/validate-diagnostic` validates the machine-readable JSON
  diagnostic object shape used by CLI/LSP tooling gates.
- `tools/cmd/validate-flow-only` now scans examples, libraries, and runtime
  sources in the v0.6 stabilization/release gates to catch legacy syntax drift.
- `wasm32-wasi` and `wasm32-web` now report a clear planned-target diagnostic
  instead of looking like arbitrary unknown triples while the real WASM backend
  remains on the v1.0 track.
- `tetra targets` reports supported and planned targets in text or JSON, and
  release gates validate the JSON shape through `tools/cmd/validate-targets`.
- `scripts/test_all.sh` and `scripts/release_v0_6_gate.sh` verify that the
  short `./t` alias built by bootstrap reports the same version as `./tetra`.
- `tetra doctor` provides a local toolchain health check in text or JSON,
  including generated manifest target/runtime ABI surface validation; release
  gates validate the JSON report through `tools/cmd/validate-doctor`.
  It also verifies that `docs/generated/manifest.json` matches the compiler
  version and that every smoke case source exists without duplicate smoke names
  or source paths. It also checks the required self-host actor runtime
  `@export("__tetra_*")` surface in canonical and embedded runtime sources.
- `tetra smoke --list --format=json` exposes the canonical smoke matrix without
  building binaries; gates validate it through `tools/cmd/validate-smoke-list`.
- `tetra fmt --check --diagnostics=json` reports unformatted files as
  `TETRA_FMT002` diagnostics instead of plain text.
- LSP diagnostics now include single-file semantic checks, so editor clients see
  errors such as missing `uses` declarations without waiting for a full build.
- Imported files avoid single-file semantic checks in LSP mode until workspace
  graph analysis exists, preventing false unresolved-import diagnostics.
- On-disk `tetra lsp --stdio-smoke <file>` now loads imported module graphs for
  semantic diagnostics while stdio unsaved documents stay conservative.
- `tools/cmd/validate-lsp-smoke` validates `--stdio-smoke` JSON shape for
  diagnostics, symbols, and hovers in the full/release gates.
- `tools/cmd/validate-lsp-stdio` validates the framed JSON-RPC transcript from
  `tetra lsp --stdio`, replacing grep-only gate coverage with protocol checks.
- `tools/cmd/validate-api-docs` validates generated Markdown API docs so gates
  reject empty or malformed `gen-docs` output.
- `tools/cmd/validate-eco-lock` validates local Eco lock JSON for capsule
  shape, duplicate IDs, and dependency resolution in full/release gates.
- `tools/cmd/validate-eco-unpack` validates project bundle unpack output for a
  capsule manifest and `.tetra` sources under `src`.
- `tools/cmd/validate-eco-vault` validates local Todex vault records against
  stored object paths, SHA-256 hashes, and byte sizes.
- LSP document symbols now preserve symbol `detail` text and map global
  constants to the LSP `Constant` kind instead of the generic variable kind.
- The stdio LSP loop now handles full-text `textDocument/didChange` notifications
  and republishes diagnostics after edits.
- `textDocument/didClose` clears cached document state and publishes an empty
  diagnostics list so editor clients can drop stale errors.
- Smoke JSON reports now include top-level `total`, `passed`, and `failed`
  counts in addition to per-case results.
- `tools/cmd/smoke-report-to-checklist --validate-only` validates smoke report
  aggregate counts, target metadata, and per-case shape without mutating
  checklist files; `test_all --full` uses it for native host and cross-target
  smoke reports.
- Smoke report count validation remains backward-compatible with old reports
  that omit counts, but rejects incomplete or inconsistent counts when those
  fields are present.
- The first v0.7 language-hardening slice accepts statement `match` over
  one-slot optionals with `case none:`, `case some(name):`, and `_`.
- Terminal no-payload enum `match` no longer requires `_` when every enum case
  is covered.
- Duplicate `match` patterns now produce a semantic diagnostic instead of
  silently creating unreachable cases.
- Flow `for value in collection:` now lowers over `String`, `[]u8`, and `[]i32`
  using the existing indexed slice/string path.
- `break` and `continue` now work inside `while`, range `for`, and collection
  `for`, with clear diagnostics when used outside a loop.
- Unary `!` is now accepted for `bool` and legacy int-like condition values and
  returns `bool`.
- Top-level `const` immutable globals are accepted as a stable spelling for
  one-slot constant data, including numeric and boolean literal inference.
- Top-level immutable globals now accept conservative constant expressions
  using earlier constants, arithmetic, comparisons, unary `-`/`!`, and
  `&&`/`||`.
- Flow and legacy `else if` parse as nested `if` statements and the formatter
  emits the compact `else if` form.
- Local `const` immutable bindings are accepted and preserved by the formatter.
- Arithmetic compound assignment sugar `+=`, `-=`, `*=`, `/=`, and `%=` is
  accepted and formatted, lowering through the existing assignment path.

## Deferred beyond v0.6

v0.6 does not add payload enums, general iterator protocols, closures, full
ownership/lifetime solving, full structured concurrency, protocol-bound
generics, production-grade LSP, UI DSL/backends, package publishing,
proof-carrying capsules, EcoNet, or distributed Todex.
