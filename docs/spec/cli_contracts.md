# Tetra CLI Contracts

Status: v1 required tooling contract.

The `tetra` CLI command surface is:

| Command | Primary behavior | Structured output |
| --- | --- | --- |
| `version` | Print the compiler version. | Text only. |
| `targets` | Print supported, build-only, and planned targets. | `--format=json`. |
| `doctor` | Check local release-critical metadata and source files. | `--format=json`. |
| `check` | Load and type-check one input, defaulting to `main.tetra`. | `--diagnostics=json` on failure. |
| `build` | Build one input, defaulting to `main.tetra`, to `-o` or the target default output. | `--diagnostics=json` on failure. |
| `run` | Build and execute one host-runnable input, returning the program exit code. | `--diagnostics=json` on failure. |
| `fmt` | Format one file to stdout, rewrite with `--write`, or verify with `--check`. | `--diagnostics=json` on failure. |
| `test` | Discover top-level `test "name":` blocks and run them on the host target. | `--report=json`; `--diagnostics=json` on command failure. |
| `doc` | Generate API docs for files or directories. | Markdown output; `--diagnostics=json` on failure. |
| `smoke` | Build and optionally run the canonical smoke matrix. | `--list --format=json`; `--report <path>`. |
| `clean` | Remove local Tetra cache directories. | Text only. |
| `eco` | Run local capsule/package workflows. | Command-specific text/JSON files. |
| `lsp` | Run stdio LSP or one-shot stdio smoke analysis. | JSON-RPC or smoke JSON. |

Exit codes:

| Code | Meaning |
| --- | --- |
| `0` | Command succeeded. |
| `1` | Valid command failed during compile, check, validation, docs, smoke, tests, or IO. |
| `2` | Command-line usage, unsupported target, unsupported format, or invalid option. |
| program code | `tetra run` returns the built program exit code after a successful build. |

JSON diagnostics use `code`, `message`, `severity`, and optional `file`, `line`,
`column`, and `hint`. Supported diagnostic modes are exactly `text` and `json`.

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

Module imports:

- A file that imports another module must declare its own `module` path.
- Module paths map directly to files under the module root: `app.main` loads
  from `app/main.tetra`.
- Import paths must be unique within a file, and import aliases must not shadow
  top-level declarations in that file.
- Import cycles, missing imports, duplicate module declarations, and module
  declarations that do not match their import path are compile errors.

Cache safety:

- Native builds store object cache entries under `.tetra_cache/<target>/`.
- Cache keys include module path, target triple, build mode flags, compiler
  version, source hash, and the signatures of externally used functions/types.
- Source edits, dependency signature edits, target changes, compiler version
  changes, debug/release mode changes, and corrupted cache entries are treated
  as cache misses.
- `tetra clean` removes `.tetra_cache` and `tetra_cache` from the current
  working directory.
