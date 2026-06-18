# Line Length Limit Implementation Plan

**Status:** planning document, not implementation evidence. **Date:** 2026-06-17. **Scope:** source
files, scripts, repo docs, CI workflows, and validators. **Primary outcome:** enforce a readable
`100` character line limit without breaking generated artifacts, hashes, URLs, or evidence
snapshots.

## 1. Goal

Add a repository-level line length validator that rejects new long lines and lets the existing
long-line debt be reduced safely through a baseline ratchet.

The final state should be:

- active source, script, and docs files have no line above `100` characters;
- generated files, caches, reports, and vendored content are excluded;
- unavoidable long lines have explicit, reviewable reasons;
- CI and local validation use the same command;
- the rule is documented enough for contributors to follow it.

## 2. Current Facts

- The repo already has a structural quality validator at `tools/cmd/validate/directory-budget`.
- Its baseline lives at `tools/cmd/validate/directory-budget/baseline.json`.
- The final six-file migration uses a strict command shaped like:

```sh
go run -buildvcs=false ./tools/cmd/validate/directory-budget \
  --strict \
  --baseline tools/cmd/validate/directory-budget/baseline.json
```

- `AGENTS.md` requires persistent Go caches outside `/tmp`.
- `docs/generated`, report outputs, and cache folders can contain generated content that should not
  be hand-wrapped.
- `docs/plan` currently already has more than six root Markdown files, so new plan documents should
  go into a thematic subdirectory.

## 3. Proposed Validator

Add a new validator:

```text
tools/cmd/validate/line-length/
```

Expected files:

```text
tools/cmd/validate/line-length/main.go
tools/cmd/validate/line-length/main_test.go
tools/cmd/validate/line-length/baseline.json
tools/cmd/validate/line-length/README.md
```

The validator should support:

- `--max 100`
- `--strict`
- `--baseline <path>`
- `--root <path>`
- `--format text`
- optional `--format json` for CI/report consumers

Default roots:

```text
compiler
cli
tools
lib
examples
docs
scripts
.github
```

Default file extensions:

```text
.go
.tetra
.sh
.js
.mjs
.ts
.md
.yml
.yaml
.json
.toml
```

## 4. Counting Rule

Count visible characters, not bytes.

Recommended behavior:

- strip the trailing newline before counting;
- count Unicode runes;
- preserve tabs as one character at first;
- report the real length and configured max;
- include `path:line` in every diagnostic.

Example diagnostic:

```text
docs/spec/surface/surface_v1.md:42: line is 137 chars, max 100
```

If tab width later becomes important, add `--tab-width 4` in a follow-up.

## 5. Exclusions

The validator should skip generated or non-human-maintained paths:

```text
.git/
.cache/
.tetra_cache/
graphify-out/
node_modules/
vendor/
reports/
dumps/
docs/generated/
```

The validator should also skip files that are normally machine output:

```text
*.lock
*.sum
*.min.js
*.map
*.svg
*.png
*.jpg
*.wasm
*.tar.gz
```

## 6. Line-Level Exceptions

Some long lines are legitimate. The validator should allow them only through clear rules, not broad
silence.

Allowed automatic exceptions:

- lines that are only a URL;
- lines that contain a long `http://` or `https://` URL;
- SHA or checksum evidence lines, such as `sha256:...`;
- Markdown table separator lines;
- fenced code block lines in docs during the baseline phase;
- JSON lines inside generated or report directories, which are already skipped.

Manual escape hatch:

```text
line-length: ignore
```

Accepted comment forms:

```text
// line-length: ignore
# line-length: ignore
<!-- line-length: ignore -->
```

The validator must report how many manual ignores were used.

## 7. Baseline Design

Use a baseline first because the repo is large.

Suggested schema:

```json
{
  "schema": "tetra.line-length-baseline.v1",
  "max": 100,
  "allowances": [
    {
      "path": "docs/example.md",
      "line_hash": "sha256:...",
      "length": 137,
      "reason": "existing debt"
    }
  ]
}
```

Use a hash of the normalized line text instead of only line numbers. This avoids hiding a new long
line when edits shift line numbers.

Baseline behavior:

- non-strict mode allows known baseline entries;
- strict mode fails if any baseline entry remains;
- ratchet mode fails if the current long-line count grows;
- removed or shortened lines should be dropped from the baseline.

## 8. Implementation Tasks

### Task 1 - Add Validator Skeleton

**Goal:** add the new command and focused tests.

**Files:**

- add `tools/cmd/validate/line-length/main.go`;
- add `tools/cmd/validate/line-length/main_test.go`;
- add `tools/cmd/validate/line-length/README.md`.

**Approach:**

- walk configured roots;
- skip excluded paths;
- inspect configured extensions;
- count line lengths;
- emit sorted diagnostics for deterministic output.

**Verification:**

```sh
GOTELEMETRY=off \
GOCACHE="$PWD/.cache/go-build-line-length" \
GOTMPDIR="$PWD/.cache/go-tmp-line-length" \
go test -buildvcs=false ./tools/cmd/validate/line-length -count=1
```

**Done when:** focused tests cover pass, fail, excluded path, and URL exception.

### Task 2 - Add Baseline Mode

**Goal:** make rollout safe for the current repository.

**Files:**

- modify `tools/cmd/validate/line-length/main.go`;
- add `tools/cmd/validate/line-length/baseline.json`.

**Approach:**

- support `--baseline`;
- match baseline entries by path and line hash;
- fail on new long lines not present in the baseline;
- provide a deterministic way to regenerate the baseline.

**Verification:**

```sh
GOTELEMETRY=off \
GOCACHE="$PWD/.cache/go-build-line-length" \
GOTMPDIR="$PWD/.cache/go-tmp-line-length" \
go test -buildvcs=false ./tools/cmd/validate/line-length -count=1
```

**Done when:** tests prove new long lines fail while baseline debt passes.

### Task 3 - Wire Structure Tests

**Goal:** protect the validator from being bypassed.

**Files:**

- inspect `tools/scriptstest/structure`;
- add or update one structure test if there is room under the six-file rule.

**Approach:**

- test that `tools/cmd/validate/line-length` exists;
- test that the command path is documented;
- avoid brittle string checks where behavior tests are better.

**Verification:**

```sh
GOTELEMETRY=off \
GOCACHE="$PWD/.cache/go-build-line-length" \
GOTMPDIR="$PWD/.cache/go-tmp-line-length" \
go test -buildvcs=false ./tools/scriptstest/structure -count=1
```

**Done when:** structure tests know about the validator without adding a new over-budget directory.

### Task 4 - Add Local Release Gate

**Goal:** make the check easy to run locally and in CI.

**Files:**

- inspect `.github/workflows/ci.yml`;
- inspect `scripts/ci/test.sh`;
- inspect `scripts/ci/test-all.sh`;
- update only the smallest stable entrypoint.

**Approach:**

- add one local command that runs the line-length validator;
- use persistent cache paths under `.cache/`;
- keep CI and local command identical where possible.

**Verification:**

```sh
GOTELEMETRY=off \
GOCACHE="$PWD/.cache/go-build-line-length" \
GOTMPDIR="$PWD/.cache/go-tmp-line-length" \
go run -buildvcs=false ./tools/cmd/validate/line-length \
  --max 100 \
  --baseline tools/cmd/validate/line-length/baseline.json
```

**Done when:** the command is documented and can be run without hidden setup.

### Task 5 - Ratchet Existing Debt

**Goal:** reduce the baseline without risky mass rewrites.

**Files:**

- files listed in `tools/cmd/validate/line-length/baseline.json`.

**Approach:**

- split long Go calls and composite literals with `gofmt`;
- wrap Markdown prose naturally;
- keep tables readable, or move complex tables to lists;
- avoid reformatting generated files;
- do not touch historical prompt snapshots unless explicitly requested.

**Verification:**

```sh
GOTELEMETRY=off \
GOCACHE="$PWD/.cache/go-build-line-length" \
GOTMPDIR="$PWD/.cache/go-tmp-line-length" \
go run -buildvcs=false ./tools/cmd/validate/line-length \
  --max 100 \
  --baseline tools/cmd/validate/line-length/baseline.json
```

**Done when:** baseline count only goes down after each cleanup slice.

### Task 6 - Strict Mode

**Goal:** remove the baseline once the repo is clean.

**Files:**

- `tools/cmd/validate/line-length/baseline.json`;
- CI/local script entrypoint.

**Approach:**

- make `--strict` part of the final release-quality command;
- keep exclusions for generated paths;
- keep narrow automatic exceptions for URLs and hashes.

**Verification:**

```sh
GOTELEMETRY=off \
GOCACHE="$PWD/.cache/go-build-line-length" \
GOTMPDIR="$PWD/.cache/go-tmp-line-length" \
go run -buildvcs=false ./tools/cmd/validate/line-length \
  --max 100 \
  --strict
```

**Done when:** no baseline allowances remain and CI enforces the rule.

## 9. Acceptance Criteria

This plan is complete when:

- `tools/cmd/validate/line-length` exists and has tests;
- the validator checks active source, script, and docs files;
- generated/cache/report paths are excluded;
- URL and hash exceptions are covered by tests;
- baseline mode prevents new long lines;
- strict mode passes when the baseline is empty;
- CI or local release gates run the same validator command;
- documentation tells contributors how to wrap lines and request exceptions.

## 10. Recommended Execution

Use a ratcheted rollout:

1. implement the validator;
2. generate the first baseline;
3. block new long lines;
4. clean old long lines in small domain slices;
5. switch to strict mode after the baseline is empty.

This avoids one huge formatting PR and keeps review focused.
