# scripts/tools

Implementation helpers used by scripts live here.

These files are not user-facing entrypoints. CI, developer, and release
workflows should expose canonical shell entrypoints from `scripts/ci`,
`scripts/dev`, or `scripts/release`, then delegate here when shared helper logic
is useful.
