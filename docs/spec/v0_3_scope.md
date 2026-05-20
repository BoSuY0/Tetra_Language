# Tetra v0.3.0 Next-Cycle Scope

Status: current scope contract for the `v0.3.0` minor line.

The current release truth stays in `docs/spec/current_supported_surface.md`.
This document records the promoted v0.3 slices and the candidates that remain
experimental, planned, or reporting-only.

## Goals

- Keep the compiler/tooling profile stable while promoting a small number of
  already-tested next-cycle slices.
- Improve CI and release hygiene so daily checks are closer to release checks.
- Reduce maintenance risk in CLI/Eco and compiler hotspot files without changing
  observable behavior.
- Improve user-facing documentation so new users can follow a short path before
  reading release engineering details.

## Candidate Promotion Slices

| Area | Candidate outcome | Required evidence before support claim |
| --- | --- | --- |
| Enum payload match | Promoted: positional enum payload constructors/bindings for match/catch/if-let with exhaustive unguarded enum match/catch diagnostics. | `go test ./compiler/... -run 'Enum|Match|TypedError' -count=1`; cross-module constructor/match is check/lower covered. |
| Callable Level 1 | Either promote a narrow non-capturing symbol-backed callable expansion or keep it experimental with stable diagnostics. | `go test ./compiler/... -run 'Closure|Callable|FunctionType' -count=1`; feature registry/docs alignment. |
| Static protocol-bound generics | Promoted: static protocol-bound generic argument validation during monomorphization, including same-module/cross-module conformance and visibility diagnostics; no dynamic dispatch. | `go test ./compiler/... -run 'Generic|Protocol|Conformance|Extension' -count=1`. |
| Ownership/resource safety | Reduce false negatives inside the conservative MVP while keeping lifetime SSA explicitly planned. | `go test ./compiler/... -run 'Ownership|Borrow|Consume|Inout|Lifetime|Resource|Island|Task' -count=1`. |
| Capsule/Eco artifacts | Make local path dependency artifact generation, validation, and repair guidance easier to use. | `go test ./cli/... ./tools/... -run 'Eco|Project|Workspace|Artifact|Capsule|Lock' -count=1`; `bash scripts/ci/test-all.sh --full --keep-going`. |
| WASI/Web smoke clarity | Distinguish build-only evidence, missing runner/browser dependencies, and real execution failures in reports. | WASI and Web smoke scripts plus their validators; target docs verification. |

## Engineering Hygiene Work

- `scripts/ci/test.sh` must remain non-mutating and fail when Go files need
  formatting.
- GitHub Actions should keep cross-platform smoke coverage and add Linux jobs for
  full wrapper, coverage, race detection, and short fuzz smoke.
- Large CLI/Eco/compiler files may be split mechanically inside the same package
  when tests prove behavior is unchanged.
- README should stay a landing page; detailed commands belong in
  `docs/user/cli_cheatsheet.md` and normative behavior belongs in specs.

## Non-Goals

- Declaring `v1.0.0` readiness.
- Claiming full lifetime SSA, dynamic protocol dispatch, full first-class
  captured closures, distributed actors, production TetraHub, or full native UI
  runtime behavior.
- Promoting WASM runtime parity beyond the smoke evidence actually collected by
  the gate.

## Verification Envelope

Before any `v0.3.0` support claim, collect fresh evidence for:

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
bash scripts/ci/test-all.sh --stabilization --keep-going --report-dir <report-dir>
bash scripts/dev/fuzz-nightly.sh --short --out-dir <report-dir>/fuzz-short
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```

Release candidates use `docs/checklists/v0_3_0_release_gate.md` and
`scripts/release/v0_3_0/gate.sh` before tagging.

## Remaining Preview Boundaries

- Callable Level 1 remains experimental unless a later gate promotes a narrower
  symbol-backed non-capturing slice.
- Ownership/resource safety remains a conservative MVP; full lifetime SSA stays
  planned.
- Capsule/Eco promotion is local artifact usability only, not distributed
  EcoNet/TetraHub production behavior.
- WASI/Web smoke clarity is reporting clarity; runtime/browser parity remains
  outside current support.
