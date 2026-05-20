# Tetra CLI Contracts

Status: current `v0.4.0` tooling contract.

Source of truth: the active release line and current public profile for this
branch are `v0.4.0`. This page documents the release-covered CLI/tooling
contract for daily local development workflows; future `v1.0.0` wording belongs
in the v1 scope and release-gate documents, not in this current contract.

The `tetra` CLI command surface is:

| Command | Primary behavior | Structured output |
| --- | --- | --- |
| `version` | Print the compiler version. | Text only. |
| `targets` | Print supported, build-only, and planned targets. | `--format=json`. |
| `features` | Print the machine-readable current, experimental, planned, and post-v1 feature registry. | `--format=json`. |
| `formats` | Print the official T4 source, package, lock, and artifact format family. | `--format=json`. |
| `doctor` | Check local release-critical metadata and source files, or project structure when given a project path. | `--format=json`. |
| `actor-net` | Run the loopback TCP broker used by Linux-x64 distributed actor runtime smokes. | Optional `--report <path>` JSON runtime report. |
| `project` | Inspect, sync, or edit local dependencies for a discovered `Capsule.t4` project. | `project info --format=json`; `project deps list/check --format=json`; sync/deps text output. |
| `workspace` | Manage a local `Tetra.workspace` member list and run multi-capsule check/sync/build/test/run workflows. | `workspace list/check/graph/build/test --format=json`; sync/run text output. |
| `new` | Scaffold a local T4 app project, optionally with `--lock`. | Text only. |
| `check` | Load and type-check one input, defaulting to `Capsule.t4` `entry` or `main.t4`/`main.tetra`; a project directory argument uses the capsule entry; `--interface-only` checks an API graph without requiring `main`. | `--diagnostics=json` on failure. |
| `build` | Build one input, defaulting to `Capsule.t4` `entry` or `main.t4`/`main.tetra`; a project directory argument uses the capsule entry; `--interface-only` validates the graph without emitting an artifact; `--artifacts=auto` repairs project artifacts before compiling. | `--diagnostics=json` on failure. |
| `run` | Build and execute one host-runnable input or project directory, returning the program exit code. | `--diagnostics=json` on failure. |
| `fmt` | Format one file to stdout, rewrite with `--write`, or verify with `--check`. | `--diagnostics=json` on failure. |
| `test` | Discover top-level `test "name":` blocks and run them on the host target; project directories use discovered source roots. | `--report=json`; `--diagnostics=json` on command failure. |
| `doc` | Generate API docs for files/directories, or discovered `Capsule.t4` source roots when no paths are given. | Markdown output; `--diagnostics=json` on failure. |
| `interface` | Generate a `.t4i` interface file from one source file, or verify it with `--check`. | T4 interface output; `--diagnostics=json` on failure. |
| `smoke` | Build and optionally run the canonical smoke matrix. | `--list --format=json`; `--report <path>`. |
| `clean` | Remove local Tetra cache directories. | Text only. |
| `eco` | Run local capsule/package workflows. | Command-specific text/JSON files. |
| `lsp` | Run stdio LSP or one-shot stdio smoke analysis. | JSON-RPC or smoke JSON. |

The `lsp --stdio` contract is intentionally an editor-tooling baseline, not a
complete language server. Release transcripts must cover initialize,
didOpen/didChange diagnostics, document symbols, hover, completion,
definition, references, rename, formatting, code actions, shutdown, and exit;
`tools/cmd/validate-lsp-stdio` validates that captured transcript shape.
Requests must use JSON-RPC `"2.0"`. Request ids may be JSON numbers or strings;
responses and JSON-RPC error objects must echo the request id value and type.
Notifications omit `id` and do not receive responses. Invalid requests, unknown
request methods, and malformed params return JSON-RPC error objects when the
request id is available. Unknown request methods return `-32601`; malformed
params return `-32602`; invalid request envelopes, including non-`"2.0"`
`jsonrpc`, return `-32600`; invalid JSON returns `-32700`. For unopened
documents, read-only textDocument requests use protocol empty results: document
symbols, completion, formatting, code actions, and references return empty
arrays; hover, definition, and rename return `null`.
Rename support is a conservative single-file top-level symbol operation. It
renames identifiers that match an open document's top-level LSP symbol table,
skips common line comments and string literals, validates `newName` as a Tetra
identifier, and returns JSON-RPC `-32602` for invalid rename names. If the
document contains a same-named local binding or parameter that would make the
edit ambiguous, rename returns `null` instead of producing a workspace edit.
This contract intentionally does not claim project-wide or cross-module rename;
public API renames should still be reviewed through the resulting diff.

Exit codes:

| Code | Meaning |
| --- | --- |
| `0` | Command succeeded. |
| `1` | Valid command failed during compile, check, validation, docs, smoke, tests, or IO. |
| `2` | Command-line usage, unsupported target, unsupported format, or invalid option. |
| program code | `tetra run` returns the built program exit code after a successful build. |

JSON diagnostics use `code`, `message`, `severity`, and optional `file`, `line`,
`column`, and `hint`. Supported diagnostic modes are exactly `text` and `json`.
The stable code families are `TETRA0001` for parser/frontend diagnostics,
`TETRA2001` for positioned semantic/compiler diagnostics, `TETRA3001` for
target-neutral IR verifier failures, `TETRA3002` for unsupported lowering
paths, and `TETRA_FMT*` for formatter diagnostics.

Diagnostic JSON shape is validated by
`go test ./tools/cmd/validate-diagnostic/... -count=1`. The validator rejects
unknown JSON fields, missing `code`/`message`/`severity`, leading or trailing
whitespace in stable string fields, invalid severity values, and partial source
positions.

JSON reports:

Schema/version policy: JSON reports include a top-level `schema` and `version`
only when the report bullet explicitly names those fields. All other JSON
reports are intentionally schema-less for v1, and their compatibility contract
is the documented field set below: required fields keep their names and JSON
types, optional fields may be omitted, arrays remain arrays of the documented
entry shapes, and consumers must ignore additive unknown fields. Removing or
renaming a documented field, changing a documented JSON type, or making an
optional field required requires a new CLI contract revision.
Conformance validators are stricter than forward-compatible consumers: the
documented validator tools reject unknown JSON fields so release evidence stays
canonical for the contract revision being validated.

- `targets --format=json` emits target metadata including `triple`, `os`,
  `arch`, `abi`, `format`, `exe_ext`, `build_only`, `run_mode`,
  `run_runner`, and `run_supported`. `run_mode` is one of `host_native`,
  `wasi_runner`, or `web_runner`.
  `run_supported` is the CLI contract for whether `tetra run --target <triple>`
  is runnable in the current host environment. Native targets are runnable only
  when they match the detected host. `wasm32-wasi` uses
  `run_mode: "wasi_runner"` and is runnable when the CLI can discover
  `wasmtime` or the Node WASI fallback. `wasm32-web` uses
  `run_mode: "web_runner"` and is runnable when the CLI can discover a
  Node web runtime runner. Browser automation remains UI-smoke evidence, not
  the production `wasm32-web` runtime runner. Missing runners set
  `run_supported: false` with a `run_unsupported_reason`. `run_runner` records
  the selected runner only when `run_supported` is true.
- `features --format=json` emits `schema`, `version`, and `features` entries
  with `id`, `name`, `status`, `scope`, `stability`, and `docs`. Status is one
  of `current`, `experimental`, `planned`, or `post-v1`.
- `formats --format=json` emits a top-level `formats` array. Each entry has
  `name`, `role`, `description`, either `extension` or `file_name`, and optional
  `primary`/`legacy` booleans. The entries match the manifest `formats` schema
  validated by `tools/cmd/validate-manifest`.
  The primary Todex package/fragment extension is `.tdx`; `.todex` is accepted
  only as a compatibility alias by Eco package commands and is not a separate
  manifest format entry.
- `doctor --format=json` emits top-level `status` plus named checks.
- `actor-net --report <path>` emits an `actornet` loopback broker report with
  runtime identity, transport, listen address, connection counts, routed frame
  counts, dropped frame counts, decode-error counts, and optional last error.
  It is broker evidence only; production distributed actor promotion still
  requires the executable `tetra.actors.distributed-runtime.v1` report accepted
  by `tools/cmd/validate-distributed-actor-runtime`.
- `test --report=json` emits `total`, `passed`, `failed`, `duration_ms`,
  `files`, and `results`; validate it with `tools/cmd/validate-test-report`.
- `smoke --list --format=json` emits the smoke matrix; validate it with
  `tools/cmd/validate-smoke-list`.
  Its top-level `run_supported` is smoke-list metadata and is not a substitute
  for `targets --format=json`: WASI/Web runtime evidence is recorded by
  `smoke --report <path>` and the dedicated WASI/Web smoke workflows.
- `smoke --report <path>` emits build/run evidence; validate it with
  `tools/cmd/smoke-report-to-checklist --validate-only`.
- `eco seed export --out <path>` emits `tetra.eco.seed.v1`; validate it with
  `tools/cmd/validate-eco-seed --seed <path>`.
- `eco needmap --lock <lock> -o <path>` emits `tetra.eco.needmap.v1`;
  validate it with `tools/cmd/validate-eco-needmap --needmap <path>`.
- `eco trust snapshot --lock <lock> --store <vault> -o <path>` emits
  `tetra.eco.trust-snapshot.v1`; validate it with
  `tools/cmd/validate-eco-trust --trust <path>`.
- `eco materialize <package.tdx> -C <out>` emits
  `<out>/tetra.materialization.json` using `tetra.eco.materialization.v1`;
  validate it with
  `tools/cmd/validate-eco-materialization --materialization <path>`.
- `eco publish --channel stable` emits `tetra.eco.publish.v1`; validate it
  with `tools/cmd/validate-eco-publish --channel stable`.
- `eco tetrahub publish --channel stable` emits the same metadata schema under
  the configured TetraHub store path.
- `eco tetrahub mirror --from <store> --to <store> --id <id> --version <x.y.z>
  --target <triple> -o <report>` copies a validated local TetraHub package
  entry and emits `tetra.eco.mirror.v1`; validate it with
  `tools/cmd/validate-eco-mirror --mirror <report>`.
- `eco tetrahub fetch --url <http-url> --to <store> --id <id>
  --version <x.y.z> --target <triple> -o <report>` downloads a single
  TetraHub package entry over HTTP(S), verifies metadata/package/trust hashes,
  writes a local store entry, and emits `tetra.eco.mirror.v1`; validate it with
  `tools/cmd/validate-eco-mirror --mirror <report>`.
- `project info --format=json` emits `found`, `root`, `capsule_path`,
  `lock_path`, `entry_path`, `source_roots`, `targets`, dependency roots, and
  artifact counts.
- `project sync [path]` discovers `Capsule.t4`, writes or refreshes
  `Tetra.lock`, and generates declared local dependency artifacts. `--check`
  reports pending writes without modifying files; `--target`, `--all-targets`,
  and `--jobs` mirror the Eco artifact builder.
- `project deps list [path] --format=json` emits dependency objects with `id`,
  `version`, `path`, `resolved_path`, `status`, and optional `detail`.
- `project deps add --path <dep-project-path> [--id <id>] [--version <x.y.z>]
  [path]` appends a local path dependency to `Capsule.t4` and tells the user to
  run `tetra project sync`.
- `project deps remove --id <id> [--version <x.y.z>] [path]` removes a declared
  dependency; omitting `--version` is allowed only when the ID is unambiguous.
- `project deps check [path] --format=json` validates local dependency paths,
  ID/version matches, transitive graph shape, and dependency cycles.
- `workspace init [path]` creates `Tetra.workspace`.
- `workspace add <member-path> [--workspace <workspace-path>]` and
  `workspace remove <member-path> [--workspace <workspace-path>]` edit workspace
  membership without touching member capsules.
- `workspace list/check/graph [path] --format=json` emits workspace members,
  capsule IDs, versions, dependency edges, status, and optional details.
- `workspace sync [path] [--target <triple>] [--all-targets] [--check]
  [--jobs <n>]` runs project sync for workspace members in dependency order.
- `workspace build [path] [--target <triple>] [--all-targets] [--format=json]
  [-o <out-dir>]` builds members in dependency order. `-o` is a directory and
  outputs are written below per-member subdirectories.
- `workspace test [path] [--target <triple>] [--fail-fast] [--format=json]`
  runs member tests in dependency order.
- `workspace run <member-path> [--workspace <path>] [--target <triple>]
  [--artifacts=strict|auto]` runs one workspace member and returns the program
  exit code. The default `--artifacts=strict` never writes project artifacts;
  `--artifacts=auto` refreshes the selected member's lock and generated local
  dependency artifacts before compiling and running it.
- `workspace build/test --format=json` emits `workspace_root`, `command`,
  optional `target`, `total`, `passed`, `failed`, `skipped`, and per-member
  `path`, `capsule_id`, `status`, optional `detail`, and optional `exit_code`.

Plan250 CLI/tooling evidence from commit `b884653`:

| Evidence field | Value |
| --- | --- |
| Targets report | `reports/plan250/cli-tools/targets.json` |
| Supported targets | `linux-x64`, `windows-x64`, `macos-x64` |
| Runner-gated WASM targets | `wasm32-wasi`, `wasm32-web` |
| Planned targets | empty list |
| Smoke list report | `reports/plan250/cli-tools/smoke-list.json` |
| Smoke list shape | target `linux-x64`, `build_only: false`, `run_supported: true`, `total: 62` |
| Test report | `reports/plan250/cli-tools/tetra-test-report.json` |
| Test report shape | `total: 0`, `passed: 0`, `failed: 0`, `files: []`, `results: []` |
| Diagnostic validator | `go test ./tools/cmd/validate-diagnostic/... -count=1` |

Project scaffolding:

- `tetra new app [--lock] <NameOrPath>` creates `Capsule.t4`, `src/main.t4`,
  `tests/main_test.t4`, and `README.md` using the T4 Source Format.
- `--lock` also writes an initial `Tetra.lock` for the scaffold.
- The generated capsule uses `entry "src/main.t4"`, `source "src"`,
  `source "tests"`, the host default target, and `permission "io"`.
- Existing directories are never overwritten.

Module imports:

- A file that imports another module must declare its own `module` path.
- Module paths map directly to `.t4` files under the module root:
  `app.main` loads from `app/main.t4`. Legacy `app/main.tetra` remains
  accepted, and `.t4i` can be used as an interface fallback for type-checking.
- `.t4i` fallback files must contain a valid `// t4i-hash: sha256:<hex>`
  header. Missing or mismatched interface hashes are compile-time errors.
- When a discovered `Capsule.t4` has `deps:` entries with local paths, imports
  are also resolved from those dependency capsules' source roots. Local capsule
  dependency cycles are compile-time project errors.
- `project deps` edits and validates those local `deps:` entries; it does not
  refresh `Tetra.lock` or generated artifacts. Use `project sync` after
  dependency changes.
- `Tetra.workspace` is a local member list:
  `workspace "tetra.workspace.v1"` followed by `member "relative/path"` lines.
  Dependency truth remains in each member's `Capsule.t4`.
- When `Capsule.t4` has `artifacts:` entries, `interface <path.t4i>` adds an
  interface artifact to module resolution and `object <path.tobj>` is appended
  to native `build`/`run` link objects. Target-aware object entries use
  `object <target> <path.tobj>` and are linked only for the matching target.
  `seed <path.t4s>` is tracked by Eco locks but is not materialized during
  build.
- If a discovered project root contains `Tetra.lock`, `check`, `build`, and
  `run` validate the current capsule graph and artifact hashes against the lock
  before loading modules. Missing locks remain allowed; present locks are
  authoritative. Declared artifact entries are checked for stale interfaces,
  wrong-target/stale objects, stale seeds, and stale locks before compile work
  begins.
- `import module.path as alias` imports a module namespace. `import
  module.path.{Name, funcName}` imports selected public symbols directly.
- `pub` marks the public surface of a module. For compatibility, modules that
  use no `pub` declarations remain public-by-default; once a module uses `pub`,
  non-`pub` functions and types are private outside that module.
- `pub import module.path.{Name}` re-exports selected public symbols through the
  current module surface.
- Import aliases and selected names must not shadow top-level declarations in
  that file. Duplicate selected names are compile errors.
- Import cycles, missing imports, duplicate modules across active source roots,
  duplicate module declarations, and module declarations that do not match their
  import path are compile errors.

Interface contracts:

- `tetra interface -o path/module.t4i path/module.t4` writes the public module
  surface with a deterministic API hash.
- `tetra interface --check -o path/module.t4i path/module.t4` compares the
  existing interface file with the current public source surface and exits with
  a diagnostic when the API is stale.
- `tetra check --interface-only <input>` type-checks an interface/API graph
  without requiring an executable `main`.
- `tetra build --interface-only <input>` validates the build graph and cache
  inputs without writing an executable, object, library, or WASM artifact.
- `.t4i` function bodies are signature stubs only; interface modules are not
  lowered or linked.
- Regular native `tetra build` can link an interface-only dependency when a
  repeated `--link-object <path.tobj>` provides a matching implementation
  object. The object module must match the `.t4i` module, target must match the
  build target, public API hash must match the `.t4i` hash, and required
  function symbols must be exported.
- A discovered `Capsule.t4` `artifacts:` `object <path.tobj>` entry behaves as a
  project-local `--link-object`; explicit CLI `--link-object` flags are appended
  after project artifact objects.
- `tetra eco artifacts build --target <native-triple> --lock Tetra.lock
  Capsule.t4` generates dependency `.t4i`, target-aware `.tobj`, and `.t4s`
  artifacts from local path dependencies, updates the project `artifacts:`
  block, and refreshes the semantic lock.
- `tetra eco artifacts check --target <native-triple> --lock Tetra.lock
  Capsule.t4` validates the expected generated artifact set without writing
  files and exits non-zero with repair hints when anything is missing or stale.
- `tetra eco artifacts build --check --target <native-triple> --lock Tetra.lock
  Capsule.t4` is the dry-run form of artifact generation. Missing generated
  files are reported as `would generate ...`.
- `tetra eco artifacts build --all-targets --lock Tetra.lock Capsule.t4`
  generates native object artifacts for every native target declared in
  `Capsule.t4`, skipping runner-gated WASM targets such as WASM object outputs.
- `tetra build --artifacts=auto` runs the artifact builder for the discovered
  project before compiling. The default `--artifacts=strict` never writes
  project artifacts and reports stale/missing declared artifacts as diagnostics.
- `tetra workspace run <member-path> --artifacts=auto` refreshes the selected
  workspace member's lock and generated local dependency artifacts before
  delegating to `tetra run`; strict mode remains the default.
- Regular `tetra build` rejects interface-only dependency modules loaded from
  `.t4i` when no matching implementation object is provided; use
  `--interface-only` for API-only validation.

Eco local package materialization:

- `tetra eco materialize <package.tdx> [--target <triple>]
  [--trust <trust.snapshot.json>] -C <out>` unpacks the package and writes
  deterministic `tetra.materialization.json` metadata. `--target` is optional;
  when omitted, materialization is unscoped and the metadata records an empty
  target. When provided and the package capsule declares targets, `--target`
  must match one of those declared targets.

Eco local package publishing:

- `tetra eco publish --package <package.tdx> --registry <path>
  [--target <triple>] [--trust <trust.snapshot.json>]` publishes into the local
  beta registry. `--target` is optional; when omitted, the published target is
  the first target declared by the package capsule, or `any` when the capsule
  declares no targets. When provided, `--target` must match a target declared by
  the package capsule.

Project targets:

- Explicit `--target` wins.
- Without `--target`, `build` and `run` use the first `targets:` entry in the
  discovered `Capsule.t4`; if there is no project target, they use the host
  default.
- `build --all-targets` builds every `targets:` entry from `Capsule.t4` and
  writes target-suffixed artifacts such as `app-linux-x64` and
  `app-wasm32-wasi.wasm`.

Actor transport evidence:

- `go run ./tools/cmd/validate-actor-transport --report <report.json>`
  validates `tetra.actors.transport.v1` evidence for a single actor message
  envelope, its `message_sha256`, and an ordered source-send/destination-receive
  trace. This is a validator contract only; it does not imply distributed actor
  runtime execution.

Native UI shell evidence:

- `go run ./tools/cmd/validate-native-ui-smoke --report <output>.ui.shell.json`
  validates `tetra.ui.native-shell.v1` sidecars emitted by native UI builds. It
  requires command-dispatch runtime identity, state/view evidence, event
  operation traces, post-dispatch bindings, and binding/action widgets.

Lowering and IR verification:

- Public `compiler.Lower`, `compiler.LowerModule`, and `compiler.LowerModules`
  verify lowered IR before returning it.
- Native public codegen wrappers reject invalid IR with `TETRA3001` before
  calling platform backends.
- The IR verifier checks main metadata, duplicate/empty function names, slot
  metadata, local slot bounds, branch labels, stack underflow, branch stack
  height joins, returns, calls, and unknown instruction kinds.
- Lowering paths that reach syntax without an IR translation return
  `TETRA3002` instead of falling through to a backend error.

Cache safety:

- Native builds store object cache entries under `.tetra_cache/<target>/`.
- Cache keys include module path, target triple, build mode flags, compiler
  version, source hash, and the signatures of externally used functions/types.
- When a dependency is loaded from `.t4i`, cache keys include that dependency's
  validated public API hash.
- Native builds that use `--link-object` include linked object content hashes in
  their cache mode key, so implementation-object changes are treated as cache
  misses for the final native build graph.
- Source edits, public dependency signature/interface edits, target changes,
  compiler version changes, debug/release mode changes, and corrupted cache
  entries are treated as cache misses.
- `tetra clean` removes `.tetra_cache` and `tetra_cache` from the current
  working directory.
- `tetra clean --target <triple>` performs target-specific cleanup: it validates
  `<triple>` as a supported target and removes only `.tetra_cache/<triple>` and
  `tetra_cache/<triple>` from the current working directory. Cache entries for
  other targets remain in place.
