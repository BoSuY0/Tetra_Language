# v0.5 Integrated Alpha Release Gate

Use this checklist before labeling a build or branch as v0.5 Integrated Alpha.

## Docs

- [x] README describes the staged profile and v0.5 Integrated Alpha surface.
- [x] `docs/roadmap_0_18_to_0_5.md` names the v0.18 baseline, completed
      v0.5 targets, and post-v0.5 deferred work.
- [x] Specs for Flow/Core syntax, unsafe/capabilities, Islands, actors, and
      runtime ABI match the implemented surface.
- [x] Generated docs manifest changes are intentional and reviewed.
- [x] `docs/release_notes_v0_5.md` distinguishes Integrated Alpha support from
      future platform goals.

## Verification

- [x] `go test ./compiler/...`
- [x] `go test ./cli/...`
- [x] `go test ./tools/...`
- [x] `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`
- [x] `./tetra smoke --target linux-x64 --run=true` on Linux.
- [x] Canonical gate: `bash scripts/release_v0_5_gate.sh`

## Profile Smoke Coverage

- [x] Flow hello and Core MVP examples build or run on the host target.
- [x] Bool, range `for`, enum/match, optionals, effects, and test-block examples are covered.
- [x] Islands, unsafe capability, and runtime/object-linking examples are covered.
- [x] Actor example behavior is checked with the intended runtime mode.
- [x] `tetra fmt --check`, `tetra test`, validated `tetra test --report=json`,
      and JSON diagnostics are exercised.
- [x] `tetra lsp --stdio-smoke` reports validated diagnostics, symbols, and
      hover data.
- [x] `go run ./tools/cmd/gen-docs` emits stable Markdown API docs.
- [x] `tetra eco verify --target ... --lock ...` validates a local multi-capsule
      dependency graph and writes provenance JSON.
- [x] `tetra eco vault add/list/verify` stores and verifies local Todex records.
- [x] v0.5 MVP examples/tests cover ownership markers, typed errors, async
      syntax, extensions, protocol declarations, generic signatures, local
      Capsule/Todex graphing, generated docs, and LSP diagnostics.
- [x] Build-only smoke passes for `linux-x64`, `macos-x64`, and `windows-x64`;
      smoke JSON reports are validated, and native execution is verified only
      when host and target match.

## Signoff

- [x] Known unsupported features are listed as deferred beyond v0.5.
- [x] Platform-specific gaps are recorded before release.
- [x] Release notes distinguish Integrated Alpha support from future Tetra
      platform goals.
