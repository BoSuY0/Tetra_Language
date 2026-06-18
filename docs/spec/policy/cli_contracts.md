# Tetra CLI Contracts

Status: current `v0.4.0` tooling contract.

Source of truth: the active release line and current public profile for this branch are `v0.4.0`.
This page documents the release-covered CLI/tooling contract for daily local development workflows;
future `v1.0.0` wording belongs in the v1 scope and release-gate documents, not in this current
contract.

The `tetra` CLI command surface is:

The core user compiler workflow is `check` and `build`: `check` validates source without writing
artifacts, and `build` applies the same language safety contract while emitting or linking
artifacts. The remaining commands are release-covered tooling around that workflow, not optional
safety levels.

Command records:

- `version`:
  - Behavior: print the compiler version.
  - Structured output: text only.
- `targets`:
  - Behavior: print supported, build-only, and planned targets.
  - Structured output: `--format=json`; `--format=toon`.
- `features`:
  - Behavior: print current, experimental, planned, and post-v1 features.
  - Structured output: `--format=json`; `--format=toon`.
- `formats`:
  - Behavior: print official T4 source, package, lock, and artifact formats.
  - Structured output: `--format=json`; `--format=toon`.
- `doctor`:
  - Behavior: check release metadata, source files, or project structure.
  - Structured output: `--format=json`; `--format=toon`.
- `actor-net`:
  - Behavior: run the loopback TCP broker used by distributed actor smokes.
  - Structured output: optional `--report <path>` JSON runtime report.
- `project`:
  - Behavior: inspect, sync, or edit discovered `Capsule.t4` dependencies.
  - Structured output: project info/deps JSON plus sync/deps text output.
- `workspace`:
  - Behavior: manage `Tetra.workspace` members and multi-capsule workflows.
  - Structured output: list/check/graph/build/test JSON plus sync/run text.
- `new`:
  - Behavior: scaffold a local T4 app project, optionally with `--lock`.
  - Structured output: text only.
- `check`:
  - Behavior: load and type-check one input or discovered project entry.
  - Extra behavior: `--interface-only` checks an API graph without `main`.
  - Structured output: `--diagnostics=json`; `--diagnostics=toon` on failure.
- `build`:
  - Behavior: build one input or discovered project entry.
  - Extra behavior: `--interface-only` validates without emitting an artifact.
  - Extra behavior: `--artifacts=auto` repairs project artifacts first.
  - Structured output: `--diagnostics=json`; `--diagnostics=toon` on failure.
- `run`:
  - Behavior: build and execute one host-runnable input or project directory.
  - Structured output: `--diagnostics=json`; `--diagnostics=toon` on failure.
- `fmt`:
  - Behavior: format one file to stdout, rewrite, or verify with `--check`.
  - Structured output: `--diagnostics=json`; `--diagnostics=toon` on failure.
- `test`:
  - Behavior: discover top-level `test "name":` blocks and run them on host.
  - Project behavior: project directories use discovered source roots.
  - Structured output: `--report=json`; `--report=toon`.
  - Alias output: `--format=json` or `--format=toon` for reports.
  - Failure output: `--diagnostics=json` or `--diagnostics=toon`.
- `surface`:
  - Behavior: run scoped Surface developer workflows such as `surface dev`.
  - Non-goal: currently a fast rebuild loop, not hot reload.
  - Structured output: `surface dev --report <path>` emits dev evidence.
  - Failure output: `--diagnostics=json`; `--diagnostics=toon`.
- `doc`:
  - Behavior: generate API docs for files, directories, or capsule roots.
  - Structured output: Markdown; diagnostics JSON/TOON on failure.
- `interface`:
  - Behavior: generate a `.t4i` interface file or verify it with `--check`.
  - Structured output: T4 interface text; diagnostics JSON/TOON on failure.
- `smoke`:
  - Behavior: build and optionally run the canonical smoke matrix.
  - List output: `--list --format=json` or `--list --format=toon`.
  - Report output: `--report <path> --report-format=json|toon|both`.
- `clean`:
  - Behavior: remove local Tetra cache directories.
  - Structured output: text only.
- `eco`:
  - Behavior: run local capsule/package workflows.
  - Structured output: command-specific text, JSON, and TOON reports.
- `lsp`:
  - Behavior: run stdio LSP or one-shot stdio smoke analysis.
  - Structured output: JSON-RPC frames for stdio.
  - Smoke output: `--stdio-smoke --format=json` or `--format=toon`.

The `lsp --stdio` contract is intentionally an editor-tooling baseline, not a complete language
server. Release transcripts must cover initialize, didOpen/didChange diagnostics, document symbols,
hover, completion, definition, references, rename, formatting, code actions, shutdown, and exit;
`tools/cmd/validate-lsp-stdio` validates that captured transcript shape. Requests must use JSON-RPC
`"2.0"`. Request ids may be JSON numbers or strings; responses and JSON-RPC error objects must echo
the request id value and type. Notifications omit `id` and do not receive responses. Invalid
requests, unknown request methods, and malformed params return JSON-RPC error objects when the
request id is available. Unknown request methods return `-32601`; malformed params return `-32602`;
invalid request envelopes, including non-`"2.0"` `jsonrpc`, return `-32600`; invalid JSON returns
`-32700`. For unopened documents, read-only textDocument requests use protocol empty results:
document symbols, completion, formatting, code actions, and references return empty arrays; hover,
definition, and rename return `null`. Rename support is a conservative single-file top-level symbol
operation. It renames identifiers that match an open document's top-level LSP symbol table, skips
common line comments and string literals, validates `newName` as a Tetra identifier, and returns
JSON-RPC `-32602` for invalid rename names. If the document contains a same-named local binding or
parameter that would make the edit ambiguous, rename returns `null` instead of producing a workspace
edit. This contract intentionally does not claim project-wide or cross-module rename; public API
renames should still be reviewed through the resulting diff.
`lsp --stdio-smoke <path> --format=json|toon` is a separate one-shot smoke report and is not a
JSON-RPC transport. Validate it with `tools/cmd/validate-lsp-smoke --format=auto|json|toon`.

Exit codes:

- `0`: command succeeded.
- `1`: valid command failed during compile, check, validation, docs, smoke,
  tests, or IO.
- `2`: command-line usage, unsupported target, unsupported format, or invalid
  option.
- Program code: `tetra run` returns the built program exit code after a
  successful build.

JSON and TOON diagnostics use `code`, `message`, `severity`, and optional `file`, `line`, `column`,
and `hint`. Supported diagnostic modes are exactly `text`, `json`, and `toon`; `text` remains the
default. The stable code families are `TETRA0001` for parser/frontend diagnostics, `TETRA2001` for
positioned semantic/compiler diagnostics, `TETRA3001` for target-neutral IR verifier failures,
`TETRA3002` for unsupported lowering paths, and `TETRA_FMT*` for formatter diagnostics.

Diagnostic shape is validated by `go test ./tools/cmd/validate-diagnostic/... -count=1` and by
`tools/cmd/validate-diagnostic`, which accepts JSON or TOON. The validator rejects unknown fields,
missing `code`/`message`/`severity`, leading or trailing whitespace in stable string fields, invalid
severity values, and partial source positions.

JSON reports:

Schema/version policy: JSON reports include a top-level `schema` and `version` only when the report
bullet explicitly names those fields. All other JSON reports are intentionally schema-less for v1,
and their compatibility contract is the documented field set below: required fields keep their names
and JSON types, optional fields may be omitted, arrays remain arrays of the documented entry shapes,
and consumers must ignore additive unknown fields. Removing or renaming a documented field, changing
a documented JSON type, or making an optional field required requires a new CLI contract revision.
Conformance validators are stricter than forward-compatible consumers: the documented validator
tools reject unknown JSON fields so release evidence stays canonical for the contract revision being
validated.

TOON is available only as an opt-in second structured format for `targets`, `features`, `formats`,
`doctor`, structured diagnostics, `tetra test` reports, LSP smoke reports, selected Eco metadata
reports, release manifest mirrors, and selected path-based release reports in this revision. The
TOON contract is defined in `docs/spec/standard_library/toon_support.md`: JSON remains default and
canonical, and validators decode TOON into the same typed report models before running existing
validation. LSP JSON-RPC frames, canonical Eco store/package metadata, and canonical release
manifests remain JSON unless a command explicitly documents a TOON mirror or input.

- `targets --format=json` and `targets --format=toon` emit target metadata including `triple`, `os`,
  `arch`, `abi`, `data_model`, `format`, `exe_ext`, `build_only`, `run_mode`, `run_runner`,
  `run_supported`, pointer/register/native-int widths, endian, stack alignment, atomic widths, and
  any `unsupported_reason`. Linux native entries also expose `runtime_status`, `stdlib_status`,
  `ffi_status`, `runner_probe_command`, `release_gate`, and `evidence_artifacts`; these fields are
  promotion-gate metadata, not support claims. `linux-x86` and `linux-x32` keep `partial_build_only`
  runtime/stdlib status until real runner-backed gates pass. Linux native entries also expose their
  syscall pack through `syscall_instruction`, `syscall_numbering`, `syscall_arg_registers`, and
  `syscall_error_range`: x64 uses the x86_64 `syscall` pack, x86 uses the i386 `int 0x80` pack, and
  x32 uses x86_64 registers with `x32_syscall_bit` numbering. `run_mode` is one of `host_native`,
  `host_probed`, `wasi_runner`, or `web_runner`. `run_supported` is the CLI contract for whether
  `tetra run --target <triple>` is runnable in the current host environment. Native targets are
  runnable only when they match the detected host. Build-only native targets such as `linux-x86` and
  `linux-x32` use `run_mode: "host_probed"`: the CLI may run them only when the current host can
  execute that exact ABI, and a failed probe must report `run_supported: false` with a
  `run_unsupported_reason` that includes the host identity, the exact `runner_probe_command`, and
  the no-host-fallback reason. General target limitations stay in `unsupported_reason`.
  `wasm32-wasi` uses `run_mode: "wasi_runner"` and is runnable when the CLI can discover `wasmtime`
  or the Node WASI fallback. `wasm32-web` uses `run_mode: "web_runner"` and is runnable when the CLI
  can discover a Chromium-compatible browser runner. Browser automation is also the production Web
  UI smoke evidence when the UI validator accepts the report. Missing runners set
  `run_supported: false` with a `run_unsupported_reason`. `run_runner` records the selected runner
  only when `run_supported` is true. Linux native target promotion is additionally guarded by
  `tools/cmd/validate-linux-native-targets`: `linux-x64` must remain the runnable LP64 baseline,
  `linux-x86` must remain i386 SysV build-only until full runtime/stdlib/FFI evidence passes, and
  `linux-x32` must remain x32 SysV build-only with x86_64 registers plus 32-bit pointer/native-int
  facts until its own evidence passes. The validator rejects premature `supported` status,
  x32-to-x64/x86 collapse, host fallback, and fake/skipped/report-only target suite evidence, and
  requires a validated `artifact-hashes.json` manifest for the same report directory. Full-family
  evidence that includes x64, x86, and x32 also requires the all-targets brutal report. Per-target
  ABI, atomic, fuzz, and passing runner reports must carry matching top-level `target` identity.
  Passing Linux native runner reports must include the release runner smoke results
  `runner arithmetic`, `runner alloc memory`, `runner filesystem`, `runner stderr fd`,
  `runner time`, `runner network socket`, and `runner network options`, and `runner task join`, so a
  runnable report proves more than a trivial arithmetic executable. Blocked x86/x32 runner
  diagnostics must include the target, host identity, probe command, and no-host-fallback reason.
  Runner evidence must agree with target metadata: a passing runner report is valid only when
  `run_supported` is true for that target, while a no-host diagnostic is valid only when the same
  `targets.json` records `run_supported: false`.
- `features --format=json` and `features --format=toon` emit `schema`, `version`, and `features`
  entries with `id`, `name`, `status`, `scope`, `stability`, and `docs`. Status is one of `current`,
  `experimental`, `planned`, or `post-v1`.
- `formats --format=json` and `formats --format=toon` emit a top-level `formats` array. Each entry
  has `name`, `role`, `description`, either `extension` or `file_name`, and optional
  `primary`/`legacy` booleans. The entries match the manifest `formats` schema validated by
  `tools/cmd/validate-manifest`. The primary Todex package/fragment extension is `.tdx`; `.todex` is
  accepted only as a compatibility alias by Eco package commands and is not a separate manifest
  format entry.
- `doctor --format=json` and `doctor --format=toon` emit top-level `status` plus named checks.
- `actor-net --report <path>` emits an `actornet` loopback broker report with runtime identity,
  transport, listen address, connection counts, routed frame counts, dropped frame counts,
  decode-error counts, and optional last error. It is broker evidence only; production distributed
  actor promotion still requires the executable `tetra.actors.distributed-runtime.v1` report
  accepted by `tools/cmd/validate-distributed-actor-runtime`.
- `test --report=json` and `test --report=toon` emit `total`, `passed`, `failed`, `duration_ms`,
  `files`, and `results`; single-target reports also include canonical `target` identity such as
  `linux-x64`, `linux-x86`, or `linux-x32`. Validate JSON or TOON with
  `tools/cmd/validate-test-report`. `test --format=json` and `test --format=toon` are aliases for
  the same report output, including target-suite commands such as
  `test --all-targets --brutal --format=json`; multi-target reports omit a single `target` value. If
  both `--report` and `--format` are provided, they must match.
- `smoke --list --format=json` and `smoke --list --format=toon` emit the smoke matrix; validate them
  with `tools/cmd/validate-smoke-list --format=json|toon`. Its top-level `run_supported` is
  smoke-list metadata and is not a substitute for `targets --format=json`: WASI/Web runtime evidence
  is recorded by `smoke --report <path>` and the dedicated WASI/Web smoke workflows.
- `smoke --report <path>` emits build/run evidence; validate it with
  `tools/cmd/smoke-report-to-checklist --validate-only --format=json`. `--report-format=both` writes
  the canonical JSON report plus a `.toon` mirror; validate the mirror with
  `tools/cmd/smoke-report-to-checklist --validate-only --format=toon`.
- `scripts/ci/test-all.sh --report-dir <dir>` writes canonical `summary.json`.
  `--report-format=toon|both` also writes `summary.toon` and TOON mirrors for selected path reports
  while `--json-only` stdout remains JSON.
- `surface dev --report <path>` emits `tetra.surface.dev-workflow.v1` fast rebuild evidence for
  Surface apps. The current evidence is scoped to `linux-x64` build caching and records initial
  build, warm-cache rebuild, and token/recipe/source changed rebuild steps plus source diagnostics.
  With `--morph-rendered-beauty-report <path>`, the report includes a validated `morph_to_pixels`
  chain for Morph tokens, recipe expansion, Block scene, render commands, frame artifact, golden
  artifact, and diff metrics. It is not a hot reload or React Fast Refresh claim; full process
  restart remains documented as fast rebuild until a real reload loop is proven.
- `new surface-app --template <kind> <path>` creates a Surface project from the current Block/Morph
  template set. Supported kinds are `command-palette`, `settings`, `dashboard`, `editor-shell`,
  `multi-window-notes`, and `studio-shell`, and `web-canvas`. The release smoke writes
  `tetra.surface.template-smoke.v1` evidence and validates it with
  `tools/cmd/validate-surface-template-smoke`.
- `scripts/release/surface/surface-reference-apps-smoke.sh --report-dir <dir>` checks, builds, runs,
  visually validates, and writes `tetra.surface.reference-app-suite.v1` evidence for the ten current
  Block/Morph reference app shapes. Validate the report with
  `tools/cmd/validate-surface-reference-apps --report <path>`.
- `scripts/release/surface/surface-package-smoke.sh --report-dir <dir>` builds linux-x64 and
  wasm32-web Surface app packages for the default command-palette reference app. The same script
  accepts `--source <path> --app-id <id> --app-title <title> --expected-exit-code <n>` for validated
  product-slice package evidence such as
  `--source examples/surface/migration/surface_migration_tetra_control_center.tetra`
  with `--app-id studio-shell`.
  It verifies local asset hashes, unpacks and runs the linux-x64 package, records web bundle
  HTML/wasm/compiler-owned loader output, writes `tetra.surface.package.v1` evidence, and validates
  it with `tools/cmd/validate-surface-package --report <path>`.
- `scripts/release/surface/surface-crash-report-smoke.sh --report-dir <dir>` builds the
  command-palette reference app for linux-x64, records bounded command failure, host crash
  diagnostic capture, local trace/log collection, redacted `tetra.surface.diagnostic.v1` artifacts,
  and scoped restart evidence in `tetra.surface.crash-report.v1`. Validate the report with
  `tools/cmd/validate-surface-crash-report --report <path>`.
- `scripts/release/surface/surface-i18n-smoke.sh --report-dir <dir>` builds the localized-form
  reference app for linux-x64, records bounded string tables, locale selection, fallback lookup,
  missing-key diagnostics, deterministic formatting hooks, and RTL placeholder nonclaim evidence in
  `tetra.surface.i18n.v1`. Validate the report with
  `tools/cmd/validate-surface-i18n --report <path>`.
- `tools/cmd/surface-inspector --out <path>` emits `tetra.surface.inspector.v1` static tool evidence
  for Surface apps. It aggregates validated runtime reports into Block tree, Morph token, layout,
  paint, accessibility, event route, focus, perf-counter, source-location, and hidden-state scan
  sections. A `morph-rendered-beauty:<path>` input adds the Morph-to-pixels inspector sections for
  recipe expansions, Block scene nodes, render commands, frame artifacts, golden diff result, and
  the source-linked hash chain. Optional HTML output is a static tool report, not browser devtools,
  React devtools, DOM runtime UI, or target-host accessibility proof by itself.
- `eco seed export --out <path> [--format=json|toon|both]` emits `tetra.eco.seed.v1`; validate it
  with `tools/cmd/validate-eco-seed --seed <path> --format=auto|json|toon`.
- `eco needmap --lock <lock> -o <path> [--format=json|toon|both]` emits `tetra.eco.needmap.v1`;
  validate it with `tools/cmd/validate-eco-needmap --needmap <path> --format=auto|json|toon`.
- `eco trust snapshot --lock <lock> --store <vault> -o <path> [--format=json|toon|both]` emits
  `tetra.eco.trust-snapshot.v1`; validate it with
  `tools/cmd/validate-eco-trust --trust <path> --format=auto|json|toon`.
- `eco materialize <package.tdx> -C <out> [--metadata-format json|toon|both]` emits
  `<out>/tetra.materialization.json` and optionally `<out>/tetra.materialization.toon` using
  `tetra.eco.materialization.v1`; validate it with
  `tools/cmd/validate-eco-materialization --materialization <path> --format=auto|json|toon`.
- `eco publish --channel stable` emits `tetra.eco.publish.v1`; validate it with
  `tools/cmd/validate-eco-publish --channel stable`.
- `eco tetrahub publish --channel stable` emits the same metadata schema under the configured
  TetraHub store path.
- `eco tetrahub mirror` copies a validated local TetraHub package entry and
  emits `tetra.eco.mirror.v1`.
  - Required args: `--from`, `--to`, `--id`, `--version`, `--target`, and `-o`.
  - Optional output: `--format=json|toon|both`.
  - Validator: `tools/cmd/validate-eco-mirror --mirror <report>`.
- `eco tetrahub fetch` downloads one TetraHub package entry over HTTP(S),
  verifies metadata/package/trust hashes, writes a local store entry, and emits
  `tetra.eco.mirror.v1`.
  - Required args: `--url`, `--to`, `--id`, `--version`, `--target`, and `-o`.
  - Optional output: `--format=json|toon|both`.
  - Validator: `tools/cmd/validate-eco-mirror --mirror <report>`.
- `project info --format=json` emits `found`, `root`, `capsule_path`, `lock_path`, `entry_path`,
  `source_roots`, `targets`, dependency roots, and artifact counts.
- `project sync [path]` discovers `Capsule.t4`, writes or refreshes `Tetra.lock`, and generates
  declared local dependency artifacts. `--check` reports pending writes without modifying files;
  `--target`, `--all-targets`, and `--jobs` mirror the Eco artifact builder.
- `project deps list [path] --format=json` emits dependency objects with `id`, `version`, `path`,
  `resolved_path`, `status`, and optional `detail`.
- `project deps add --path <dep-project-path> [--id <id>] [--version <x.y.z>] [path]` appends a
  local path dependency to `Capsule.t4` and tells the user to run `tetra project sync`.
- `project deps remove --id <id> [--version <x.y.z>] [path]` removes a declared dependency; omitting
  `--version` is allowed only when the ID is unambiguous.
- `project deps check [path] --format=json` validates local dependency paths, ID/version matches,
  transitive graph shape, and dependency cycles.
- `workspace init [path]` creates `Tetra.workspace`.
- `workspace add <member-path> [--workspace <workspace-path>]` and
  `workspace remove <member-path> [--workspace <workspace-path>]` edit workspace membership without
  touching member capsules.
- `workspace list/check/graph [path] --format=json` emits workspace members, capsule IDs, versions,
  dependency edges, status, and optional details.
- `workspace sync [path] [--target <triple>] [--all-targets] [--check] [--jobs <n>]` runs project
  sync for workspace members in dependency order.
- `workspace build [path] [--target <triple>] [--all-targets] [--format=json] [-o <out-dir>]` builds
  members in dependency order. `-o` is a directory and outputs are written below per-member
  subdirectories.
- `workspace test [path] [--target <triple>] [--fail-fast] [--format=json]` runs member tests in
  dependency order.
- `workspace run <member-path> [--workspace <path>] [--target <triple>] [--artifacts=strict|auto]`
  runs one workspace member and returns the program exit code. The default `--artifacts=strict`
  never writes project artifacts; `--artifacts=auto` refreshes the selected member's lock and
  generated local dependency artifacts before compiling and running it.
- `workspace build/test --format=json` emits `workspace_root`, `command`, optional `target`,
  `total`, `passed`, `failed`, `skipped`, and per-member `path`, `capsule_id`, `status`, optional
  `detail`, and optional `exit_code`.

Plan250 CLI/tooling evidence from commit `b884653`:

- Targets report: `reports/plan250/cli-tools/targets.json`.
- Supported targets: `linux-x64`, `windows-x64`, `macos-x64`.
- Runner-gated WASM targets: `wasm32-wasi`, `wasm32-web`.
- Planned targets: empty list.
- Smoke list report: `reports/plan250/cli-tools/smoke-list.json`.
- Smoke list shape: target `linux-x64`, `build_only: false`,
  `run_supported: true`, `total: 62`.
- Test report: `reports/plan250/cli-tools/tetra-test-report.json`.
- Test report shape: `total: 0`, `passed: 0`, `failed: 0`, `files: []`,
  `results: []`.
- Diagnostic validator: `go test ./tools/cmd/validate-diagnostic/... -count=1`.

Project scaffolding:

- `tetra new app [--lock] <NameOrPath>` creates `Capsule.t4`, `src/main.t4`, `tests/main_test.t4`,
  and `README.md` using the T4 Source Format.
- `--lock` also writes an initial `Tetra.lock` for the scaffold.
- The generated capsule uses `entry "src/main.t4"`, `source "src"`, `source "tests"`, the host
  default target, and `permission "io"`.
- Existing directories are never overwritten.

Module imports:

- A file that imports another module must declare its own `module` path.
- Module paths map directly to `.t4` files under the module root: `app.main` loads from
  `app/main.t4`. Legacy `app/main.tetra` remains accepted, and `.t4i` can be used as an interface
  fallback for type-checking.
- `.t4i` fallback files must contain a valid `// t4i-hash: sha256:<hex>` header. Missing or
  mismatched interface hashes are compile-time errors.
- When a discovered `Capsule.t4` has `deps:` entries with local paths, imports are also resolved
  from those dependency capsules' source roots. Local capsule dependency cycles are compile-time
  project errors.
- `project deps` edits and validates those local `deps:` entries; it does not refresh `Tetra.lock`
  or generated artifacts. Use `project sync` after dependency changes.
- `Tetra.workspace` is a local member list: `workspace "tetra.workspace.v1"` followed by
  `member "relative/path"` lines. Dependency truth remains in each member's `Capsule.t4`.
- When `Capsule.t4` has `artifacts:` entries, `interface <path.t4i>` adds an interface artifact to
  module resolution and `object <path.tobj>` is appended to native `build`/`run` link objects.
  Target-aware object entries use `object <target> <path.tobj>` and are linked only for the matching
  target. `seed <path.t4s>` is tracked by Eco locks but is not materialized during build.
- If a discovered project root contains `Tetra.lock`, `check`, `build`, and `run` validate the
  current capsule graph and artifact hashes against the lock before loading modules. Missing locks
  remain allowed; present locks are authoritative. Declared artifact entries are checked for stale
  interfaces, wrong-target/stale objects, stale seeds, and stale locks before compile work begins.
- `import module.path as alias` imports a module namespace. `import module.path.{Name, funcName}`
  imports selected public symbols directly.
- `pub` marks the public surface of a module. For compatibility, modules that use no `pub`
  declarations remain public-by-default; once a module uses `pub`, non-`pub` functions and types are
  private outside that module.
- `pub import module.path.{Name}` re-exports selected public symbols through the current module
  surface.
- Import aliases and selected names must not shadow top-level declarations in that file. Duplicate
  selected names are compile errors.
- Import cycles, missing imports, duplicate modules across active source roots, duplicate module
  declarations, and module declarations that do not match their import path are compile errors.

Interface contracts:

- `tetra interface -o path/module.t4i path/module.t4` writes the public module surface with a
  deterministic API hash.
- `tetra interface --check -o path/module.t4i path/module.t4` compares the existing interface file
  with the current public source surface and exits with a diagnostic when the API is stale.
- `tetra check --interface-only <input>` type-checks an interface/API graph without requiring an
  executable `main`.
- `tetra build --interface-only <input>` validates the build graph and cache inputs without writing
  an executable, object, library, or WASM artifact.
- `.t4i` function bodies are signature stubs only; interface modules are not lowered or linked.
- Regular native `tetra build` can link an interface-only dependency when a repeated
  `--link-object <path.tobj>` provides a matching implementation object. The object module must
  match the `.t4i` module, target must match the build target, public API hash must match the `.t4i`
  hash, and required function symbols must be exported.
- A discovered `Capsule.t4` `artifacts:` `object <path.tobj>` entry behaves as a project-local
  `--link-object`; explicit CLI `--link-object` flags are appended after project artifact objects.
- `tetra eco artifacts build --target <native-triple> --lock Tetra.lock Capsule.t4` generates
  dependency `.t4i`, target-aware `.tobj`, and `.t4s` artifacts from local path dependencies,
  updates the project `artifacts:` block, and refreshes the semantic lock.
- `tetra eco artifacts check --target <native-triple> --lock Tetra.lock Capsule.t4` validates the
  expected generated artifact set without writing files and exits non-zero with repair hints when
  anything is missing or stale.
- `tetra eco artifacts build --check --target <native-triple> --lock Tetra.lock Capsule.t4` is the
  dry-run form of artifact generation. Missing generated files are reported as `would generate ...`.
- `tetra eco artifacts build --all-targets --lock Tetra.lock Capsule.t4` generates native object
  artifacts for every native target declared in `Capsule.t4`, skipping runner-gated WASM targets
  such as WASM object outputs.
- `tetra build --artifacts=auto` runs the artifact builder for the discovered project before
  compiling. The default `--artifacts=strict` never writes project artifacts and reports
  stale/missing declared artifacts as diagnostics.
- `tetra workspace run <member-path> --artifacts=auto` refreshes the selected workspace member's
  lock and generated local dependency artifacts before delegating to `tetra run`; strict mode
  remains the default.
- Regular `tetra build` rejects interface-only dependency modules loaded from `.t4i` when no
  matching implementation object is provided; use `--interface-only` for API-only validation.

Eco local package materialization:

- `tetra eco materialize <package.tdx> [--target <triple>] [--trust <trust.snapshot.json>] -C <out>`
  unpacks the package and writes deterministic `tetra.materialization.json` metadata. `--target` is
  optional; when omitted, materialization is unscoped and the metadata records an empty target. When
  provided and the package capsule declares targets, `--target` must match one of those declared
  targets.

Eco local package publishing:

- `tetra eco publish --package <package.tdx> --registry <path>` publishes into
  the local beta registry.
  - Optional args: `--target <triple>` and `--trust <trust.snapshot.json>`.
  - Default target: first package capsule target, or `any` when absent.
  - Constraint: provided `--target` must match a declared package target.

Project targets:

- Explicit `--target` wins.
- Without `--target`, `build` and `run` use the first `targets:` entry in the discovered
  `Capsule.t4`; if there is no project target, they use the host default.
- `build --all-targets` builds every `targets:` entry from `Capsule.t4` and writes target-suffixed
  artifacts such as `app-linux-x64` and `app-wasm32-wasi.wasm`.

Actor transport evidence:

- `go run ./tools/cmd/validate-actor-transport --report <report.json>` validates
  `tetra.actors.transport.v1` evidence for a single actor message envelope, its `message_sha256`,
  and an ordered source-send/destination-receive trace. This is a validator contract only; it does
  not imply distributed actor runtime execution.

Native UI shell evidence:

- `go run ./tools/cmd/validate-native-ui-smoke --report <output>.ui.shell.json` validates
  `tetra.ui.native-shell.v1` sidecars emitted by native UI builds. It requires command-dispatch
  runtime identity, state/view evidence, event operation traces, post-dispatch bindings, and
  binding/action widgets.

Lowering and IR verification:

- Public `compiler.Lower`, `compiler.LowerModule`, and `compiler.LowerModules` verify lowered IR
  before returning it.
- Native public codegen wrappers reject invalid IR with `TETRA3001` before calling platform
  backends.
- The IR verifier checks main metadata, duplicate/empty function names, slot metadata, local slot
  bounds, branch labels, stack underflow, branch stack height joins, returns, calls, and unknown
  instruction kinds.
- Lowering paths that reach syntax without an IR translation return `TETRA3002` instead of falling
  through to a backend error.

Cache safety:

- Native builds store object cache entries under `.tetra_cache/<target>/`.
- Cache keys include module path, target triple, build mode flags, compiler version, source hash,
  and the signatures of externally used functions/types.
- When a dependency is loaded from `.t4i`, cache keys include that dependency's validated public API
  hash.
- Native builds that use `--link-object` include linked object content hashes in their cache mode
  key, so implementation-object changes are treated as cache misses for the final native build
  graph.
- Source edits, public dependency signature/interface edits, target changes, compiler version
  changes, debug/release mode changes, and corrupted cache entries are treated as cache misses.
- `tetra clean` removes `.tetra_cache` and `tetra_cache` from the current working directory.
- `tetra clean --target <triple>` performs target-specific cleanup: it validates `<triple>` as a
  supported target and removes only `.tetra_cache/<triple>` and `tetra_cache/<triple>` from the
  current working directory. Cache entries for other targets remain in place.
