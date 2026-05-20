# scripts/dev

Local developer workflow entrypoints live here.

Prefer names that describe the workflow: `format.sh`, `bootstrap.sh`,
`dump-project.sh`, and `fuzz-nightly.sh`.

`format.sh` is the canonical mutating formatter. There is no root-level
compatibility wrapper.

`bootstrap.sh` is the canonical local CLI binary builder. There is no root-level
compatibility wrapper.

`fuzz-nightly.sh` is the canonical bounded fuzz/property/stress workflow. There
is no root-level compatibility wrapper.

`dump-project.sh` is the canonical project snapshot workflow around
`tools/cmd/dump-project`. There is no root-level compatibility wrapper.
