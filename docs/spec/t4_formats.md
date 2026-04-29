# T4 Formats

Tetra source files use the T4 Source Format:

```text
.t4
```

The extension represents the four native aspects of the language: Data,
Effect, Time, and Eco. The legacy `.tetra` extension remains supported for
existing projects, but new source files should use `.t4`.

Official format family:

| Format | Role |
| --- | --- |
| `.t4` | Tetra source file |
| `.tdx` | Todex encrypted semantic fragment |
| `.t4s` | Tetra Seed offline bundle |
| `.t4i` | T4 interface file generated from source for fast surface checks |
| `.t4p` | T4 proof file |
| `.t4r` | T4 replay file |
| `.t4q` | T4 quest file |
| `.tneed` | NeedMap file |
| `Tetra.lock` | Tetra semantic lockfile |

Capsule manifests are Tetra source and should live in `Capsule.t4`. The legacy
`Tetra.capsule` filename remains accepted for existing local Eco bundles. CLI
commands discover `Capsule.t4`, use its `entry` as the default program, and use
its `sources` block as the module lookup roots. If a source module is absent,
the loader can use a matching `.t4i` file as an interface fallback for
type-checking.

`.t4i` files contain the public T4 interface surface plus a deterministic
`// t4i-hash: sha256:<hex>` header. The hash is computed from the generated
public interface body, so private implementation body edits do not change it.
When a source module uses `pub`, only `pub` declarations and `pub import`
re-exports are emitted into the interface file; legacy modules without `pub`
remain public-by-default for compatibility.
Function and type declarations loaded from `.t4i` are treated as interface-only
signatures: their stub bodies are not body-checked or linked. Public metadata
that is not yet emitted as executable interface syntax still participates in
the `.t4i` hash so stale API checks can detect surface changes.

The compiler validates the `.t4i` header before parsing an interface fallback.
Missing headers, malformed hashes, or hashes that no longer match the interface
body are compile-time errors. Interface fallback modules can be type-checked,
but normal codegen does not link them; use `tetra check --interface-only` or
`tetra build --interface-only` for API-only validation, or provide the source or
object implementation for a regular native build. A linked implementation
object must carry the same module name and public API hash as the `.t4i` file.
Inside `Capsule.t4`, an `artifacts:` block can bind project-local files into the
graph:

```t4 unsupported
artifacts:
    interface interfaces/math/core.t4i
    object linux-x64 artifacts/math-core.linux-x64.tobj
    seed seeds/tetra-core.t4s
```

Interface artifacts participate in module lookup, object artifacts are linked by
native `build`/`run` for their matching target, and seed artifacts are tracked in
`Tetra.lock`. `tetra project sync` can generate these entries from local path
dependencies while refreshing the project lock. `tetra project sync --check`
verifies that generated interfaces, target-aware objects, dependency seeds, and
`Tetra.lock` are current without writing files. The lower-level `tetra eco
artifacts build/check` commands remain available for explicit capsule graph
work, and `--all-targets` generates native object artifacts for every native
target declared in `Capsule.t4` while skipping build-only targets. `tetra build
--artifacts=auto` runs artifact repair before compiling; strict build mode only
validates and reports stale declared artifacts.

Example project layout:

```text
NotesApp/
    Capsule.t4
    Tetra.lock
    src/
        main.t4
        AppView.t4
        NotesStore.t4
    ui/
        Sidebar.t4
    quests/
        fix_search_bug.t4q
    replays/
        crash_empty_note.t4r
    proofs/
        no_secret_logging.t4p
    eco/
        missing.tneed
```
