package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestFormatSourceIdempotencePropertySuite(t *testing.T) {
	fixtures := map[string]string{
		"function":       "func main() -> Int:\n    return 42\n",
		"comments":       "// property fixture\nfunc main() -> Int:\n    // stable body\n    return 0\n",
		"control_escape": "func main() -> String:\n    return \"\b\"\n",
		"view": `state CounterState:
    var count: Int = 0

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> increment
    command increment:
        state.count = state.count + 1
    accessibility label: String = "Increment"
`,
		"test": "test \"property math\":\n    expect 40 + 2 == 42\n",
	}

	for name, src := range fixtures {
		t.Run(name, func(t *testing.T) {
			once, err := compiler.FormatSource([]byte(src), name+".tetra")
			if err != nil {
				t.Fatalf("format once: %v", err)
			}
			twice, err := compiler.FormatSource(once, name+".tetra")
			if err != nil {
				t.Fatalf("format twice: %v", err)
			}
			if string(twice) != string(once) {
				t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", once, twice)
			}
		})
	}
}

func TestFormatSourceUsesParserSupportedStringEscapes(t *testing.T) {
	src := []byte("func A()->A:\n    return \"\b\"\n")
	formatted, err := compiler.FormatSource(src, "control_escape.tetra")
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	if !strings.Contains(string(formatted), `return "\x08"`) {
		t.Fatalf("formatted source = %q, want parser-supported hex escape", formatted)
	}
	if _, err := compiler.FormatSource(formatted, "control_escape.tetra"); err != nil {
		t.Fatalf("reformat: %v\n%s", err, formatted)
	}
}

func TestFormatSourcePropertySuiteCoversCommentRejectionAndMalformedInput(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		wantCode    string
		wantMessage string
		wantLine    int
		wantColumn  int
	}{
		{
			name:        "inline_comment",
			src:         "func main() -> Int:\n    return 0 // trailing\n",
			wantCode:    "TETRA_FMT001",
			wantMessage: "inline comments are not supported",
			wantLine:    2,
			wantColumn:  14,
		},
		{
			name:        "tabbed_indent",
			src:         "func main() -> Int:\n\treturn 0\n",
			wantCode:    "TETRA0001",
			wantMessage: "tabs are not supported",
			wantLine:    2,
			wantColumn:  1,
		},
		{
			name:        "incomplete_function_body",
			src:         "func A()->A:\n ;",
			wantCode:    "TETRA0001",
			wantMessage: "expected indented block after ':'",
			wantLine:    2,
			wantColumn:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compiler.FormatSource([]byte(tt.src), tt.name+".tetra")
			if err == nil {
				t.Fatalf("expected formatter diagnostic")
			}
			diag := compiler.DiagnosticFromError(err)
			if diag.Code != tt.wantCode || diag.File != tt.name+".tetra" || diag.Line != tt.wantLine || diag.Column != tt.wantColumn || diag.Severity != "error" {
				t.Fatalf("diagnostic = %#v", diag)
			}
			if !strings.Contains(diag.Message, tt.wantMessage) {
				t.Fatalf("diagnostic message = %q, want substring %q", diag.Message, tt.wantMessage)
			}
		})
	}
}

func TestFormatSourceRepositoryParseFormatParseProperty(t *testing.T) {
	repoRoot := formatPropertyRepoRoot(t)
	roots := []string{
		filepath.Join(repoRoot, "examples"),
		filepath.Join(repoRoot, "lib"),
		filepath.Join(repoRoot, "__rt"),
		filepath.Join(repoRoot, "compiler", "selfhostrt"),
	}
	type corpusResult struct {
		formatted int
		skipped   int
	}
	results := map[string]corpusResult{}

	for _, root := range roots {
		root := root
		t.Run(filepath.ToSlash(root), func(t *testing.T) {
			files := tetraCorpusFiles(t, root)
			result := corpusResult{}
			for _, path := range files {
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read %s: %v", path, err)
				}
				before, err := compiler.ParseFile(raw, path)
				if err != nil {
					t.Fatalf("parse original %s: %v", path, err)
				}
				once, err := compiler.FormatSource(raw, path)
				if err != nil {
					if strings.Contains(err.Error(), "inline comments are not supported") {
						result.skipped++
						continue
					}
					t.Fatalf("format %s: %v", path, err)
				}
				after, err := compiler.ParseFile(once, path)
				if err != nil {
					t.Fatalf("parse formatted %s: %v\nformatted:\n%s", path, err, string(once))
				}
				if got, want := fileSurfaceSignature(after), fileSurfaceSignature(before); got != want {
					t.Fatalf("surface changed for %s:\ngot  %s\nwant %s", path, got, want)
				}
				twice, err := compiler.FormatSource(once, path)
				if err != nil {
					t.Fatalf("format twice %s: %v", path, err)
				}
				if string(twice) != string(once) {
					t.Fatalf("format not idempotent for %s", path)
				}
				result.formatted++
			}
			if result.formatted == 0 {
				t.Fatalf("no format-compatible corpus files under %s (skipped %d)", root, result.skipped)
			}
			results[root] = result
		})
	}
}

func formatPropertyRepoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repo root %s is missing go.mod: %v", root, err)
	}
	return root
}

func tetraCorpusFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("stat %s: %v", root, err)
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".tetra" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatalf("no .tetra files under %s", root)
	}
	return files
}

func fileSurfaceSignature(file *compiler.FileAST) string {
	var parts []string
	for _, imp := range file.Imports {
		parts = append(parts, "import:"+imp.Path+" as "+imp.Alias)
	}
	for _, glob := range file.Globals {
		parts = append(parts, "global:"+glob.Name)
	}
	for _, enum := range file.Enums {
		parts = append(parts, "enum:"+enum.Name)
	}
	for _, st := range file.Structs {
		parts = append(parts, "struct:"+st.Name)
	}
	for _, state := range file.States {
		parts = append(parts, "state:"+state.Name)
	}
	for _, view := range file.Views {
		parts = append(parts, "view:"+view.Name)
	}
	for _, proto := range file.Protocols {
		parts = append(parts, "protocol:"+proto.Name)
	}
	for _, ext := range file.Extensions {
		parts = append(parts, "extension:"+ext.Target.Name)
	}
	for _, impl := range file.Impls {
		parts = append(parts, "impl:"+impl.Type.Name+":"+impl.Protocol.Name)
	}
	for _, fn := range file.Funcs {
		parts = append(parts, "func:"+fn.Name)
	}
	for _, test := range file.Tests {
		parts = append(parts, "test:"+test.Name)
	}
	return strings.Join(parts, "|")
}
