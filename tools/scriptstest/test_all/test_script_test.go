package testall

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalTestScriptHasFrontendFocusedTarget(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts", "ci"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(
		filepath.Join(repoRoot(t), "scripts", "ci", "test.sh"),
		filepath.Join(root, "scripts", "ci", "test.sh"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(`#!/usr/bin/env bash
set -euo pipefail
case "$1" in
  rev-parse)
    exit 0
    ;;
  ls-files)
    exit 0
    ;;
  *)
    echo "unexpected git command: $*" >&2
    exit 9
    ;;
esac
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "gofmt"), []byte(`#!/usr/bin/env bash
set -euo pipefail
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	goLog := filepath.Join(root, "go.log")
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte(`#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${TETRA_TEST_GO_LOG:?}"
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/ci/test.sh", "--frontend-focused")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TETRA_TEST_GO_LOG="+goLog,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("frontend-focused target failed: %v\n%s", err, out)
	}
	rawLog, err := os.ReadFile(goLog)
	if err != nil {
		t.Fatalf("read go log: %v", err)
	}
	log := string(rawLog)
	for _, want := range []string{
		("test ./compiler/internal/frontend ./compiler -run " +
			"Lex|Parser|Flow|Diagnostic|Stabilization -count=1"),
		"test ./compiler/internal/backend/wasm32_web -count=1",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("frontend-focused target missing %q in go calls:\n%s", want, log)
		}
	}
	for _, forbidden := range []string{
		"test ./compiler/...",
		"test ./cli/...",
		"test ./tools/...",
	} {
		if strings.Contains(log, forbidden) {
			t.Fatalf(
				"frontend-focused target should not run canonical full suite command %q; go calls:\n%s",
				forbidden,
				log,
			)
		}
	}
	if !strings.Contains(string(out), "OK") {
		t.Fatalf("frontend-focused target should print OK; output:\n%s", out)
	}
}

func TestTOONFormatCheckCoversExpandedStructuredSurfaces(t *testing.T) {
	root := repoRoot(t)
	script := filepath.Join(root, "scripts", "ci", "toon-format-check.sh")
	if out, err := exec.Command("bash", "-n", script).CombinedOutput(); err != nil {
		t.Fatalf("bash -n failed: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("read toon format check: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"GOTMPDIR",
		"lsp --stdio-smoke",
		"lsp --stdio --format=toon",
		"smoke --list --target linux-x64 --format=toon",
		"smoke --target linux-x64 --run=false",
		"--report-format=both",
		"gen-manifest",
		"validate-test-all-summary",
		"eco verify",
		"eco seed export",
		"eco needmap",
		"compiler/internal/webrt",
		"OK toon-format-check",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("toon format check missing %q", want)
		}
	}
}

func TestCanonicalTestScriptArtifactFollowsTetraVersion(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scripts", "ci"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(
		filepath.Join(repoRoot(t), "scripts", "ci", "test.sh"),
		filepath.Join(root, "scripts", "ci", "test.sh"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(`#!/usr/bin/env bash
set -euo pipefail
case "$1" in
  rev-parse)
    exit 0
    ;;
  ls-files)
    exit 0
    ;;
  *)
    echo "unexpected git command: $*" >&2
    exit 9
    ;;
esac
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(binDir, "gofmt"),
		[]byte("#!/usr/bin/env bash\nset -euo pipefail\nexit 0\n"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(binDir, "go"),
		[]byte("#!/usr/bin/env bash\nset -euo pipefail\nexit 0\n"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "tetra"),
		[]byte("#!/usr/bin/env bash\nset -euo pipefail\nif [[ \"${1:-}\" == version ]]; then echo v0.4.0; exit 0; fi\nexit 9\n"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/ci/test.sh", "--frontend-focused")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("frontend-focused target failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Artifact: tetra.release.v0_4_0.go-test-suite.v1") {
		t.Fatalf("test.sh artifact should follow ./tetra version; output:\n%s", out)
	}
}

func TestCanonicalTestScriptSkipsDeletedTrackedGoFilesDuringFormatCheck(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts", "ci"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(
		filepath.Join(repoRoot(t), "scripts", "ci", "test.sh"),
		filepath.Join(root, "scripts", "ci", "test.sh"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "existing.go"),
		[]byte("package main\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "untracked.go"),
		[]byte("package main\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(`#!/usr/bin/env bash
set -euo pipefail
case "$1" in
  rev-parse)
    exit 0
    ;;
  ls-files)
    printf 'existing.go\0deleted.go\0untracked.go\0'
    exit 0
    ;;
  *)
    echo "unexpected git command: $*" >&2
    exit 9
    ;;
esac
`), 0o755); err != nil {
		t.Fatal(err)
	}
	gofmtLog := filepath.Join(root, "gofmt.log")
	if err := os.WriteFile(filepath.Join(binDir, "gofmt"), []byte(`#!/usr/bin/env bash
set -euo pipefail
for path in "$@"; do
  if [[ "$path" == -* ]]; then
    continue
  fi
  printf '%s\n' "$path" >>"${TETRA_TEST_GOFMT_LOG:?}"
  if [[ ! -e "$path" ]]; then
    echo "lstat $path: no such file or directory" >&2
    exit 123
  fi
done
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(binDir, "go"),
		[]byte("#!/usr/bin/env bash\nset -euo pipefail\nexit 0\n"),
		0o755,
	); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "scripts/ci/test.sh", "--frontend-focused")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TETRA_TEST_GOFMT_LOG="+gofmtLog,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("frontend-focused target should ignore deleted tracked Go files: %v\n%s", err, out)
	}
	rawLog, err := os.ReadFile(gofmtLog)
	if err != nil {
		t.Fatalf("read gofmt log: %v", err)
	}
	log := string(rawLog)
	for _, want := range []string{"existing.go", "untracked.go"} {
		if !strings.Contains(log, want) {
			t.Fatalf("format check did not inspect %s; gofmt log:\n%s", want, log)
		}
	}
	if strings.Contains(log, "deleted.go") {
		t.Fatalf("format check passed deleted tracked file to gofmt; gofmt log:\n%s", log)
	}
}

func TestCanonicalTestScriptUsageDocumentsFrontendFocusedTarget(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "ci", "test.sh"))
	if err != nil {
		t.Fatalf("read ci test script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/ci/test.sh [--frontend-focused]",
		"--frontend-focused",
		"go test ./compiler/... -count=1",
		"go test ./cli/... -count=1",
		"go test ./tools/... -count=1",
		"go test ./compiler/internal/frontend ./compiler -run",
		"go test ./compiler/internal/backend/wasm32_web",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("scripts/ci/test.sh missing usage/contract text %q", want)
		}
	}
	assertNoLegacyMention(t, text, "scripts/test.sh", "scripts/ci/test.sh usage")
}

func Test_cheatsheet_docs_example_eco_pack_unpack(t *testing.T) {
	root := repoRoot(t)
	commands := cheatsheetShellCommandsInSection(
		t,
		filepath.Join(root, "docs", "user", "start", "cli_cheatsheet.md"),
		"## Eco And Artifacts",
	)
	snippet := cheatsheetEcoPackUnpackSnippet(t, commands)
	want := []string{
		`tmp_dir="$(mktemp -d)"`,
		`./tetra eco pack --project examples/projects/hello_t4/Capsule.t4 -o "$tmp_dir/package.tdx"`,
		`./tetra eco unpack "$tmp_dir/package.tdx" -C "$tmp_dir/unpacked"`,
		`go run ./tools/cmd/validate-eco-unpack --dir "$tmp_dir/unpacked"`,
	}
	if strings.Join(snippet, "\n") != strings.Join(want, "\n") {
		t.Fatalf(
			"cheatsheet eco pack/unpack docs example is not the validated command form\nwant:\n%s\ngot:\n%s",
			strings.Join(want, "\n"),
			strings.Join(snippet, "\n"),
		)
	}

	script := filepath.Join(t.TempDir(), "cheatsheet-eco-pack-unpack.sh")
	if err := os.WriteFile(script, []byte(cheatsheetDocsExampleScript(snippet)), 0o755); err != nil {
		t.Fatalf("write docs example script: %v", err)
	}
	cmd := exec.Command("bash", script)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "TMPDIR="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cheatsheet eco pack/unpack docs example failed: %v\n%s", err, out)
	}
}

func Test_stdlib_standard_library_guide_tetra_doc_examples_invariant(t *testing.T) {
	root := repoRoot(t)
	guidePath := filepath.Join(root, "docs", "user", "platform", "standard_library_guide.md")
	raw, err := os.ReadFile(guidePath)
	if err != nil {
		t.Fatalf("read standard library guide: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"stable stdlib modules render as `lib.core.<name>`",
		"experimental stdlib modules render as `lib.experimental.<name>`",
		"`examples/flow/flow_hello.tetra`",
		("| Slice summation helpers (`sum_i32`, `weighted_sum_i32`, `sum_" +
			"u8`) | `import lib.core.slices as slices` | `examples/core/data/core_" +
			"slices_smoke.tetra` | `mem` |"),
		"`slices.sum_i32(values)`",
		"`slices.weighted_sum_i32(values)`",
		"`slices.sum_u8(values)`",
		("| ASCII length, ASCII sum, and empty checks (`ascii_len`, " +
			"`ascii_sum`, `is_empty`) | `import lib.core.strings as strings` | " +
			"`examples/core/data/core_strings_smoke.tetra` | none |"),
		"`strings.ascii_len(value)`",
		"`strings.ascii_sum(value)`",
		"`strings.is_empty(value)`",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("standard_library_guide missing generated-docs naming invariant %q", want)
		}
	}
	for _, forbidden := range []string{
		"Slice length and first/fallback helpers",
		"String length and prefix-style helpers",
		"prefix-style helpers",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf(
				"standard_library_guide contains outdated slices/strings wording %q",
				forbidden,
			)
		}
	}

	commands := markdownShellCommandsInSection(t, guidePath, "## Verification")
	workflow := strings.Join(commands, "\n")
	for _, want := range []string{
		`./tetra doc \`,
		`lib/core \`,
		`lib/experimental \`,
		`examples/core/memory/core_capability_smoke.tetra \`,
		`examples/core/data/core_math_smoke.tetra \`,
		`> reports/stdlib-api-docs.md`,
		`go run ./tools/cmd/validate-api-docs --docs reports/stdlib-api-docs.md`,
		`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`,
		`./tetra doc examples > reports/examples-docs.md`,
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf(
				"standard_library_guide verification workflow missing %q in commands:\n%s",
				want,
				workflow,
			)
		}
	}
	stdlibDocs := workflow[:strings.Index(
		workflow,
		`go run ./tools/cmd/validate-api-docs --docs reports/stdlib-api-docs.md`,
	)]
	if strings.Contains(stdlibDocs, "./tetra doc examples") ||
		!strings.Contains(stdlibDocs, "lib/core") ||
		!strings.Contains(stdlibDocs, "lib/experimental") {
		t.Fatalf(
			"standard_library_guide stdlib docs workflow must document lib/core and lib/experimental before examples-only docs:\n%s",
			workflow,
		)
	}
}

func Test_core_capability_smoke_examples_index_invariant(t *testing.T) {
	root := repoRoot(t)
	examplePath := filepath.Join(root, "examples", "core", "memory", "core_capability_smoke.tetra")
	rawExample, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("read core capability smoke example: %v", err)
	}
	example := string(rawExample)
	for _, want := range []string{
		"module examples.core.memory.core_capability_smoke",
		"import lib.core.capability as capability",
		"uses alloc, capability, io, mem, mmio:",
		"unsafe:",
		"let mem_cap: cap.mem = capability.mem()",
		"let io_cap: cap.io = capability.io()",
		"let stored: UInt8 = core.store_u8(mem_ptr, 17, mem_cap)",
		"let written: Int = core.mmio_write_i32(io_ptr, 25, io_cap)",
		"return out",
	} {
		if !strings.Contains(example, want) {
			t.Fatalf("core capability smoke example missing %q", want)
		}
	}

	rawIndex, err := os.ReadFile(
		filepath.Join(root, "docs", "user", "reference", "examples_index.md"),
	)
	if err != nil {
		t.Fatalf("read examples index: %v", err)
	}
	index := string(rawIndex)
	for _, want := range []string{
		("| `examples/core/memory/core_capability_smoke.tetra` | Current " +
			"core capability token acquisition smoke for `cap.mem` and `cap.io`. | " +
			"native | exits 42 using only caller-owned heap memory and local MMIO " +
			"storage; does not imply host permission grant |"),
	} {
		if !strings.Contains(index, want) {
			t.Fatalf("examples index missing capability smoke row %q", want)
		}
	}
}

func Test_examples_index_has_rows_for_every_core_smoke(t *testing.T) {
	root := repoRoot(t)
	coreSmokes, err := filepath.Glob(
		filepath.Join(root, "examples", "core", "*", "core_*_smoke.tetra"),
	)
	if err != nil {
		t.Fatalf("glob core smoke examples: %v", err)
	}
	if len(coreSmokes) == 0 {
		t.Fatal("no core smoke examples found")
	}
	rawIndex, err := os.ReadFile(
		filepath.Join(root, "docs", "user", "reference", "examples_index.md"),
	)
	if err != nil {
		t.Fatalf("read examples index: %v", err)
	}
	rows := examplesIndexRowsByPath(string(rawIndex))
	for _, examplePath := range coreSmokes {
		rel, err := filepath.Rel(root, examplePath)
		if err != nil {
			t.Fatalf("relative path for %s: %v", examplePath, err)
		}
		rel = filepath.ToSlash(rel)
		row, ok := rows[rel]
		if !ok {
			t.Fatalf("examples index missing core smoke row for %s", rel)
		}
		if row.purpose == "" {
			t.Fatalf("examples index core smoke row for %s missing purpose", rel)
		}
		if !strings.Contains(row.target, "native") && !strings.Contains(row.target, "wasm") {
			t.Fatalf(
				"examples index core smoke row for %s has invalid target group %q",
				rel,
				row.target,
			)
		}
		expected := strings.ToLower(row.expected)
		if !strings.Contains(expected, "exits ") && !strings.Contains(expected, "build-only") &&
			!strings.Contains(expected, "check-only") {
			t.Fatalf(
				"examples index core smoke row for %s missing expected behavior: %q",
				rel,
				row.expected,
			)
		}
	}
}

type examplesIndexRow struct {
	purpose  string
	target   string
	expected string
}

func examplesIndexRowsByPath(markdown string) map[string]examplesIndexRow {
	rows := map[string]examplesIndexRow{}
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") || strings.Contains(trimmed, "---") {
			continue
		}
		cols := splitMarkdownRow(trimmed)
		if len(cols) != 4 || strings.EqualFold(cols[0], "Example") {
			continue
		}
		path := strings.Trim(cols[0], "` ")
		if !strings.HasPrefix(path, "examples/") {
			continue
		}
		rows[path] = examplesIndexRow{
			purpose:  strings.TrimSpace(cols[1]),
			target:   strings.TrimSpace(cols[2]),
			expected: strings.TrimSpace(cols[3]),
		}
	}
	return rows
}

func splitMarkdownRow(line string) []string {
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part))
	}
	return out
}

func cheatsheetShellCommandsInSection(t *testing.T, path string, heading string) []string {
	t.Helper()
	return markdownShellCommandsInSection(t, path, heading)
}

func markdownShellCommandsInSection(t *testing.T, path string, heading string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read markdown docs: %v", err)
	}
	var commands []string
	inSection := false
	inShellFence := false
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if inSection && trimmed != heading {
				break
			}
			inSection = trimmed == heading
			inShellFence = false
			continue
		}
		if !inSection {
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			if inShellFence {
				inShellFence = false
				continue
			}
			inShellFence = trimmed == "```sh" || trimmed == "```bash" || trimmed == "```shell"
			continue
		}
		if inShellFence && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			commands = append(commands, trimmed)
		}
	}
	if len(commands) == 0 {
		t.Fatalf("no shell commands found in markdown section %q", heading)
	}
	return commands
}

func cheatsheetEcoPackUnpackSnippet(t *testing.T, commands []string) []string {
	t.Helper()
	packIndex := -1
	validateIndex := -1
	for i, command := range commands {
		if packIndex < 0 && strings.Contains(command, "eco pack ") {
			packIndex = i
		}
		if packIndex >= 0 && strings.Contains(command, "validate-eco-unpack") {
			validateIndex = i
			break
		}
	}
	if packIndex < 0 {
		t.Fatal("cheatsheet missing eco pack docs example")
	}
	if validateIndex < 0 {
		t.Fatal("cheatsheet eco pack/unpack docs example missing validate-eco-unpack command")
	}
	start := packIndex
	if start > 0 && strings.HasPrefix(commands[start-1], "tmp_dir=") {
		start--
	}
	return commands[start : validateIndex+1]
}

func cheatsheetDocsExampleScript(commands []string) string {
	var script strings.Builder
	script.WriteString("set -euo pipefail\n")
	for _, command := range commands {
		if strings.HasPrefix(command, "./tetra ") {
			command = "go run ./cli/cmd/tetra " + strings.TrimPrefix(command, "./tetra ")
		}
		script.WriteString(command)
		script.WriteByte('\n')
	}
	return script.String()
}
