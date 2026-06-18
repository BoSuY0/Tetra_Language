# Roadmap v0.15 → v0.16 (Runtime & Toolchain Stabilization)

Focus: make the runtime and object-linking path dependable after the v0.15 Core Language MVP,
without adding new language syntax.

## P0 — Runtime mode contract

- `--runtime=auto` selects the embedded self-host actors runtime when actor builtins are used.
- `--runtime=selfhost` forces embedded runtime compilation for `linux-x64`, `macos-x64`, and
  `windows-x64`.
- `--runtime=builtin` remains available as a compatibility fallback.
- `--runtime-object` must be target-matching and must export the required `__tetra_*` actor runtime
  symbols.

## P1 — Canonical self-host runtime sources

- Production runtime sources are `__rt/actors_sysv.tetra` and `__rt/actors_win64.tetra`.
- Embedded compiler copies live under `compiler/selfhostrt/` with matching module names.
- `actors_poc_*` files remain historical references only.

## P2 — Object linking ergonomics

- `--link-object` supports repeatable target-matching TOBJ libraries.
- Duplicate symbols, target mismatch, and unresolved linked symbols produce clear linker
  diagnostics.
- `--emit=library` plus `--link-object` is documented as the local library workflow.

## P3 — Deferred language work

Effects enforcement, ownership, protocols, extensions, async, UI, and EcoNet remain v0.17+ work.
`uses` stays parsed metadata in v0.16.
