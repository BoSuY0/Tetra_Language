# Tetra v0.3.0 Dirty Worktree Inventory

Status: active cleanup inventory for the `v0.3.0` release branch.

This inventory records why the current branch cannot produce a tag-ready
`--require-clean` release gate yet. It is based on local `git status` and
`git diff` inspection during the completion audit.

## Summary

Current dirty-state counts from `git status --porcelain --untracked-files=all`:

| Status | Count | Meaning |
| --- | ---: | --- |
| `M` | 174 | Modified tracked files |
| `D` | 6 | Deleted tracked files |
| `A` | 1 | Added staged file |
| `AM` | 1 | Added staged file with worktree modifications |
| `??` | 110 | Untracked files |

Top-level dirty entry distribution:

| Prefix | Count |
| --- | ---: |
| `.github` | 1 |
| `.orchestrator` | 10 |
| `README.md` | 1 |
| `cli` | 10 |
| `compiler` | 58 |
| `docs` | 123 |
| `examples` | 11 |
| `scripts` | 14 |
| `tools` | 64 |

## Deleted Tracked Files

The current branch deletes six historical todo/audit files:

- `docs/plans/2026-04-26-tetra-language-todo.md`
- `docs/plans/2026-04-27-tetra-stabilization-5000-todo.md`
- `docs/plans/2026-04-27-tetra-v0_1-to-v1_0-full-todo.md`
- `docs/plans/2026-04-27-tetra-v0_2_0-1000-todo.md`
- `docs/plans/todo_closure_map_2026-04-26.md`
- `docs/release/v0_1_2_todo_internal_audit_2026-04-27.md`

Release cleanup decision needed: either keep these deletions as intentional
historical-doc cleanup for `v0.3.0`, or restore them before the tag-ready pass.

## Likely Release Changes

These untracked/modified groups look like intentional `v0.3.0` release work and
should be reviewed for inclusion:

- `cli/cmd/tetra/*`: project/workspace/Eco/LSP/smoke CLI surface changes.
- `compiler/**`: parser, semantics, lowering, target, WASM, ownership, safety,
  and runtime tests for the promoted slices and MVP hardening.
- `tools/cmd/validate-*`: validators needed by the release gate and generated
  evidence checks.
- `scripts/release/v0_3_0/gate.sh`
- `scripts/release/v0_3_0/security-review.sh`
- `docs/checklists/v0_3_0_release_gate.md`
- `docs/spec/v0_3_scope.md`
- `docs/release-notes/v0_3_0.md`
- `docs/release/v0_3_0_final_handoff.md`
- `docs/release/v0_3_0_completion_audit.md`
- `docs/user/status.md`
- `docs/user/v0_3_preview.md`
- `docs/user/cli_cheatsheet.md`
- `docs/user/tutorial_path.md`
- `docs/contributing/compiler_pipeline.md`
- `examples/core_capability_smoke.tetra`
- `examples/wasm_*_smoke.tetra`

## Generated Evidence And Artifacts

These files are release/generated evidence candidates and need a policy decision
before tagging:

- `docs/generated/manifest.json`
- `docs/generated/v1_0/*.json`
- `docs/generated/v1_0/*.md`
- `docs/generated/v1_0/web-ui-smoke.*`
- `docs/generated/v1_0/wasi-smoke-artifacts/**`
- `docs/generated/v1_0/wasm-smoke-artifacts/**`

Important: generated `v1_0` paths may be compatibility or historical names.
They are not proof of `v1.0.0` readiness. For `v0.3.0`, each generated artifact
must be tied to the `v0.3.0` checklist or to a version-neutral reused invariant
before being treated as release evidence.

## Local Orchestration Artifacts

These entries look like local process artifacts, not public release source:

- `.orchestrator/state.md`
- `.orchestrator/waves/wave_49_report.md`
- `.orchestrator/waves/wave_50_report.md`
- `.orchestrator/waves/wave_51_report.md`
- `.orchestrator/waves/wave_52_report.md`
- `.orchestrator/waves/wave_53_report.md`
- `.orchestrator/waves/wave_54_report.md`
- `.orchestrator/waves/wave_55_report.md`
- `.orchestrator/waves/wave_56_report.md`
- `.orchestrator/waves/wave_57_report.md`

Release cleanup decision: keep these as local-only orchestration notes and
exclude them from release source. `.gitignore` now ignores `.orchestrator/` so
future local wave journals do not appear in `git status --porcelain
--untracked-files=all`. If long-term traceability is needed, archive the notes
outside the release source tree instead of committing them as release evidence.

## Locally Verified During Audit

The following checks passed during the current audit session:

```sh
go test ./compiler/... -run 'Enum|Match|TypedError|Generic|Protocol|Conformance|Extension|Closure|Callable|FunctionType|Ownership|Borrow|Consume|Inout|Lifetime|Resource|Island|Task' -count=1
go test ./cli/... ./tools/... -run 'Eco|Project|Workspace|Artifact|Capsule|Lock' -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
GOCACHE=/tmp/tetra-v0.3-go-cache GOENV=off bash scripts/dev/fuzz-nightly.sh --short --out-dir /tmp/tetra-v0.3-fuzz-short-audit-writable-cache
bash scripts/release/v1_0/web-smoke.sh --report /tmp/tetra-v0.3-web-ui-smoke-rerun.json
bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir /tmp/tetra-v0.3-stabilization-audit-rerun
env TETRA_RELEASE_GATE_CI_ALLOW_MISSING_SECURITY_SIGNOFF=1 TETRA_MACOS_RUNTIME_SMOKE_REPORT=docs/generated/v1_0/macos-smoke.json TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=docs/generated/v1_0/windows-smoke.json bash scripts/release/v0_3_0/gate.sh --report-dir /tmp/tetra-v0.3-gate-audit-crossbuild-runtime-blocked
```

The first fuzz attempt used the default Go cache and failed in the sandbox
because `/home/tetra/.cache/go-build/fuzz/...` was read-only. The rerun above
uses an explicit writable cache under `/tmp` and passed.

The standalone web UI browser smoke report
`/tmp/tetra-v0.3-web-ui-smoke-rerun.json` has `status: pass`. The subsequent
all-in-one stabilization archive
`/tmp/tetra-v0.3-stabilization-audit-rerun/summary.md` also has `status: pass`
with 38 steps and 0 failures.

The fresh release-gate audit report
`/tmp/tetra-v0.3-gate-audit-crossbuild-runtime-blocked/summary.md` is
identity-matched for `v0.3.0` but blocked. It confirms that existing
`docs/generated/v1_0/macos-smoke.json` and `windows-smoke.json` are
cross-target build artifacts from `host: linux-x64`, not valid native
`--run=true` runtime execution evidence.

## Cleanup Checklist

- Review all modified source and tests as release-intent changes.
- Decide whether deleted historical todo files stay deleted.
- Decide which generated evidence belongs in source control for `v0.3.0`.
- Keep `.orchestrator/` ignored as local-only process state, or archive it
  outside the release source tree if human traceability is required.
- Recollect fresh short fuzz and stabilization reports after source cleanup if
  the source tree changes again.
- Import macOS and Windows runtime smoke reports from matching hosts or CI.
- Add the human security signoff and detached hash.
- Run the `v0.3.0` gate, then rerun it with `--require-clean` on the intended
  tag commit.
