# Tetra CLI Contracts

Status: v1 required tooling contract.

The `tetra` CLI command surface is:

| Command | Primary behavior | Structured output |
| --- | --- | --- |
| `version` | Print the compiler version. | Text only. |
| `targets` | Print supported, build-only, and planned targets. | `--format=json`. |
| `doctor` | Check local release-critical metadata and source files, or project structure when given a project path. | `--format=json`. |
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

JSON reports:

- `targets --format=json` emits target metadata including `triple`, `os`,
  `arch`, `abi`, `format`, `exe_ext`, `build_only`, and `run_supported`.
- `doctor --format=json` emits top-level `status` plus named checks.
- `test --report=json` emits `total`, `passed`, `failed`, `duration_ms`,
  `files`, and `results`; validate it with `tools/cmd/validate-test-report`.
- `smoke --list --format=json` emits the smoke matrix; validate it with
  `tools/cmd/validate-smoke-list`.
- `smoke --report <path>` emits build/run evidence; validate it with
  `tools/cmd/smoke-report-to-checklist --validate-only`.
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
- `workspace run <member-path> [--workspace <path>] [--target <triple>]` runs
  one workspace member and returns the program exit code.
- `workspace build/test --format=json` emits `workspace_root`, `command`,
  optional `target`, `total`, `passed`, `failed`, `skipped`, and per-member
  `path`, `capsule_id`, `status`, optional `detail`, and optional `exit_code`.

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
  `Capsule.t4`, skipping build-only targets such as WASM object outputs.
- `tetra build --artifacts=auto` runs the artifact builder for the discovered
  project before compiling. The default `--artifacts=strict` never writes
  project artifacts and reports stale/missing declared artifacts as diagnostics.
- Regular `tetra build` rejects interface-only dependency modules loaded from
  `.t4i` when no matching implementation object is provided; use
  `--interface-only` for API-only validation.

Project targets:

- Explicit `--target` wins.
- Without `--target`, `build` and `run` use the first `targets:` entry in the
  discovered `Capsule.t4`; if there is no project target, they use the host
  default.
- `build --all-targets` builds every `targets:` entry from `Capsule.t4` and
  writes target-suffixed artifacts such as `app-linux-x64` and
  `app-wasm32-wasi.wasm`.

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
