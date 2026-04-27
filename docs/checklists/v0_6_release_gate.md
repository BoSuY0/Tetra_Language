# v0.6 Usable Alpha Release Gate

> Historical checkpoint. This gate documented the v0.6 release branch only.
> The current public baseline is `v0.1.0`; use
> `docs/checklists/v1_0_release_gate.md` only for future v1.0 release work.

Use this checklist before labeling a build or branch as v0.6 Usable Alpha.

## Docs

- [x] README describes v0.6 as a hardening/usability release.
- [x] `docs/roadmap_0_5_to_0_6.md` records completed v0.6 work and deferred
      post-v0.6 features.
- [x] `docs/release_notes_v0_6.md` distinguishes v0.6 support from future
      platform goals.
- [x] Generated docs manifest reports `v0.6.0` and passes schema validation.

## Verification

- [x] `go test ./compiler/... ./cli/... ./tools/...`
- [x] `bash scripts/test.sh`
- [x] `bash scripts/bootstrap.sh`
- [x] `./tetra version` prints `v0.6.0`
- [x] `./t version` matches `./tetra version`
- [x] `./tetra fmt --check examples lib __rt compiler/selfhostrt`
- [x] `go run ./tools/cmd/validate-flow-only examples lib __rt compiler/selfhostrt`
- [x] `./tetra targets --format=json` with `tools/cmd/validate-targets`
- [x] `./tetra doctor --format=json` with `tools/cmd/validate-doctor`
- [x] `./tetra check examples/flow_hello.tetra`
- [x] JSON diagnostic shape validation through `tools/cmd/validate-diagnostic`
- [x] `./tetra smoke --list --format=json` with `tools/cmd/validate-smoke-list`
- [x] `./tetra test examples`
- [x] `./tetra smoke --target linux-x64 --run=true --report ...` with validated
      JSON counts and case metadata
- [x] `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
- [x] Canonical gate: `bash scripts/release_v0_6_gate.sh`
- [x] v0.6.x wrapper: `bash scripts/test_all.sh --full`
- [x] v0.6.x machine-output wrapper: `bash scripts/test_all.sh --quick --json-only`

## Profile Smoke Coverage

- [x] LSP stdio transcript and validated `--stdio-smoke` JSON are exercised.
- [x] Generated API docs are validated for non-empty Markdown API structure.
- [x] Eco verify/lock JSON validation, vault add/list/verify/store validation,
      single-manifest pack, and validated project bundle unpack are exercised.
- [x] Build-only smoke passes for `linux-x64`, `macos-x64`, and `windows-x64`;
      native execution is verified only when host and target match.
- [x] `scripts/test_all.sh` writes per-step logs plus Markdown and JSON
      summaries for local stabilization runs.
- [x] `scripts/test_all.sh --keep-going` and `--json-only` are covered by CLI
      regression tests.
