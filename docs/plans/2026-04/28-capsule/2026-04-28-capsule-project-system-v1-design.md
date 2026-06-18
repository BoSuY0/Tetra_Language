# Capsule Project System v1 Design

**Goal:** make `Capsule.t4` the project root that drives entry discovery,
source roots, Eco metadata, and default CLI behavior.

## Observed Facts

- `Capsule.t4` is already the preferred file name, with legacy
  `Tetra.capsule` compatibility.
- T4 source formats are registered in `compiler/internal/formats` and exposed by
  `tetra formats`.
- The module loader currently loads from an entry file and resolves imports by
  module path.
- Eco capsule parsing currently lives in the `tetra` CLI package and feeds
  verify/pack/seed/needmap/trust/publish flows.

## Design

`Capsule.t4` remains Tetra-shaped source, but the project parser handles a
small manifest profile before the compiler treats `capsule` as normal language
syntax. The parser supports the existing flat fields and adds structured blocks:

- `entry "src/main.t4"` or `entry src/main.t4`
- `sources:` with one source root per line
- `targets:` with target triples or aliases (`linux`, `windows`, `macOS`,
  `web`, `wasi`)
- `deps:` with exact dependency pairs for graph checks
- `allow:` with one permission per line
- `policy:` with key/value pairs such as `unsafe deny` and
  `reproducible required`

Project discovery walks upward from the current directory or explicit input
file directory, preferring `Capsule.t4` over legacy `Tetra.capsule`. If the user
does not pass an entry file, CLI commands use the capsule entry. If no entry is
declared, they try `main.t4`, `src/main.t4`, `main.tetra`, and
`src/main.tetra`.

Module loading gains an explicit project root plus source roots. Imports resolve
from each source root, preferring `.t4` over `.tetra`, so `src/app/main.t4` can
import `ui/components/button.t4` without the entry file's directory becoming the
whole module universe.

Eco lock data includes project policy so `Tetra.lock` records the semantic
policy selected by the capsule.

## Non-Goals

- Do not make `capsule` a full compiler frontend construct in this pass.
- Do not implement a network dependency resolver for version ranges.
- Do not add real `tetraOS` target support until the target layer owns it.

## Verification

- Focused tests for compiler project loading, CLI project defaults, structured
  capsule parsing, and Eco lock policy.
- `go test ./compiler/... ./cli/... ./tools/...`
- `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`
