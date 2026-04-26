# Tetra 0.6.1 to 0.6.3 Stabilization Plan

This roadmap covers the 0.6.x point-release line after `v0.6.0`. The line is
reserved for stability, diagnostics, and test-envelope hardening. It should not
grow the public language surface except where a stability fix needs a clearer
diagnostic.

## 0.6.1: Test Envelope Hardening

- `scripts/test_all.sh` supports quick/full gates, `--keep-going`, `--json-only`,
  stable exit codes, per-step logs, `summary.md`, and `summary.json`.
- CLI tests cover the wrapper interface and a fake-repo keep-going failure path
  without recursively invoking the real repository tests.
- Eco/tooling tests cover duplicate capsule IDs, target mismatch diagnostics,
  corrupt vault object detection, and smoke report JSON shape.

## 0.6.2: Negative Semantic Coverage

- Regression tests cover invalid optional use, invalid `if let`, wrong thrown
  error types, throwing `main`, duplicate `inout` arguments, missing MMIO
  effects, task runtime effects, and protocol signature mismatch.
- JSON diagnostic snapshots cover semantic failures in addition to parser and
  formatter errors. Parser/frontend diagnostics keep `TETRA0001`; positioned
  semantic/compiler diagnostics use `TETRA2001`.
- Text diagnostics remain compatible with existing CLI expectations.

## 0.6.3: Cross-Target Confidence

- Keep build-only smoke green for `linux-x64`, `macos-x64`, and `windows-x64`.
- Keep existing ELF, PE, Mach-O, TOBJ, runtime-object, and link-object tests in
  the full gate.
- Native execution remains host-target only; non-host targets are build-verified.

## Release Rule

Before a 0.6.x point release, run:

```sh
bash scripts/test_all.sh --full
bash scripts/release_v0_6_gate.sh
```

If the release version advances beyond `v0.6.0`, update the exact release gate
and generated manifest in the same patch.

## First v0.7 Language-Hardening Slice

- Statement `match` now accepts one-slot optional scrutinees with `case none:`,
  `case some(name):`, and `case _:` patterns.
- Terminal no-payload enum matches are treated as complete when all enum cases
  are covered.
- Duplicate `match` patterns are rejected for integer, enum, `none`, and
  `some(name)` cases.
- Flow collection `for value in collection:` is implemented for `String`,
  `[]u8`, and `[]i32`; general iterator protocols remain planned.
- `break` and `continue` are implemented for `while`, range `for`, and
  collection `for`; using either outside a loop is a semantic diagnostic.
- Unary `!` is implemented for `bool` and legacy int-like condition values.
- Top-level `const` immutable globals are implemented for the current one-slot
  global storage path, including numeric and boolean literal inference.
- Top-level immutable globals accept conservative constant expressions over
  literals and earlier same-file constants.
- Flow and legacy `else if` are implemented as parser/formatter sugar over
  nested `if` statements.
- Local `const` bindings are implemented as immutable local declarations with
  formatter preservation.
- Arithmetic compound assignment sugar `+=`, `-=`, `*=`, `/=`, and `%=` is
  implemented as parser/formatter sugar over normal assignment.
- General enum payload patterns remain planned.
