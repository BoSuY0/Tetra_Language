# Line Length Validator

`validate/line-length` enforces the repository line-length rule for maintained
source, scripts, docs, and CI files.

Final strict usage, after the baseline is empty:

```sh
go run -buildvcs=false ./tools/cmd/validate/line-length --max 100 --strict
```

Current rollout usage blocks new long lines while old debt is removed:

```sh
go run -buildvcs=false ./tools/cmd/validate/line-length \
  --max 100 \
  --baseline tools/cmd/validate/line-length/baseline.json
```

Regenerate the baseline deterministically:

```sh
go run -buildvcs=false ./tools/cmd/validate/line-length \
  --max 100 \
  --write-baseline tools/cmd/validate/line-length/baseline.json
```

The validator counts Unicode runes after stripping the newline. It scans
`compiler`, `cli`, `tools`, `lib`, `examples`, `docs`, `scripts`, and
`.github` by default.

Generated docs and evidence snapshots such as `docs/generated` and
`docs/baselines` are skipped because wrapping them would change machine-readable
fixtures instead of improving maintained prose.

Machine report JSON files ending in `_report.json` are skipped for the same
reason.

Historical prompt snapshots ending in `-prompt.md` are skipped; they are
archival inputs, not prose to hand-wrap during cleanup.

Release data JSON under `docs/release/*/data` is also skipped where the current
repo stores machine-readable release state.

Allowed automatic exceptions are long URLs, checksum evidence lines, Markdown
table separator lines, and fenced Markdown code blocks during baseline mode.

Use `line-length: ignore` only when a long line is unavoidable and reviewable.
The validator reports how many manual ignores were used.
