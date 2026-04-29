package compiler

import (
	"encoding/binary"
	"runtime"
	"strings"
	"testing"
)

func TestBuildConstGlobalSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `const base: i32 = 40
const delta = 2

func main() -> Int:
    return base + delta
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestFormatSourceConstGlobal(t *testing.T) {
	src := []byte(`const answer: i32 = 42
func main() -> Int:
    return answer
`)
	got, err := FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `const answer: i32 = 42

func main() -> Int:
    return answer
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestBuildConstBoolGlobalSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `const enabled = true

func main() -> Int:
    if enabled:
        return 42
    return 1
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildConstGlobalExpressionSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `const base: i32 = (20 + 2) * 2
const delta = 100 % 3
const enabled: Bool = (base + delta == 45) && !false

func main() -> Int:
    if enabled:
        return base + delta - 3
    return 1
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestGlobalVarConstInitializerSemantics(t *testing.T) {
	src := `
var base: Int = 40 + 2
var enabled: Bool = true && !false

func main() -> Int:
    if enabled:
        return base
    return 0
`
	file, err := ParseFile([]byte(src), "main.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "",
		Files:       []*FileAST{file},
		ByModule:    map[string]*FileAST{"": file},
	}
	if _, err := CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
}

func TestBuildGlobalVarConstInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var base: Int = 40 + 2
var enabled: Bool = true

func main() -> Int:
    if enabled:
        return base
    return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestPropertyDefaultInitializerSemantics(t *testing.T) {
	src := `
property base: Int
property enabled: Bool
property p: ptr
property b: UInt8
property w: UInt16
property terr: task.error

func main() -> Int:
    if enabled:
        return 1
    if terr != 0:
        return 2
    return base + b + w + 42
`
	file, err := ParseFile([]byte(src), "main.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "",
		Files:       []*FileAST{file},
		ByModule:    map[string]*FileAST{"": file},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	globals := checked.GlobalsByModule[""]
	data := checked.GlobalDataByModule[""]
	for _, name := range []string{"base", "enabled", "p", "b", "w", "terr"} {
		info, ok := globals[name]
		if !ok {
			t.Fatalf("missing global %q", name)
		}
		if info.DataIndex < 0 || info.DataIndex >= len(data) {
			t.Fatalf("global %q data index out of range: %d", name, info.DataIndex)
		}
		if got := binary.LittleEndian.Uint64(data[info.DataIndex]); got != 0 {
			t.Fatalf("global %q default = %d, want 0", name, got)
		}
	}
}

func TestBuildPropertyDefaultInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
property base: Int
property enabled: Bool
property p: ptr
property b: UInt8
property w: UInt16
property terr: task.error

func main() -> Int:
    if enabled:
        return 1
    if terr != 0:
        return 2
    return base + b + w + 42
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestGlobalVarExpandedScalarInitializerSemantics(t *testing.T) {
	src := `
var b: UInt8 = 41
var w: UInt16 = 1
var terr: task.error = 0

func main() -> Int:
    if terr != 0:
        return 0
    return b + w
`
	file, err := ParseFile([]byte(src), "main.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "",
		Files:       []*FileAST{file},
		ByModule:    map[string]*FileAST{"": file},
	}
	if _, err := CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
}

func TestGlobalVarStringLiteralInitializerSemantics(t *testing.T) {
	src := `
var title: String = "hello"

func main() -> Int:
    return 0
`
	file, err := ParseFile([]byte(src), "main.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "",
		Files:       []*FileAST{file},
		ByModule:    map[string]*FileAST{"": file},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	globals := checked.GlobalsByModule[""]
	info, ok := globals["title"]
	if !ok {
		t.Fatalf("missing global %q", "title")
	}
	if info.TypeName != "str" {
		t.Fatalf("global title type = %q, want str", info.TypeName)
	}
	data := checked.GlobalDataByModule[""]
	if info.DataIndex < 0 || info.DataIndex+1 >= len(data) {
		t.Fatalf("global title data index out of range: %d len=%d", info.DataIndex, len(data))
	}
	if got := binary.LittleEndian.Uint64(data[info.DataIndex+1]); got != 5 {
		t.Fatalf("global title len slot = %d, want 5", got)
	}
}

func TestGlobalValStringLiteralInitializerSemantics(t *testing.T) {
	src := `
val title: String = "hello"

func main() -> Int:
    return 0
`
	file, err := ParseFile([]byte(src), "main.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "",
		Files:       []*FileAST{file},
		ByModule:    map[string]*FileAST{"": file},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	globals := checked.GlobalsByModule[""]
	info, ok := globals["title"]
	if !ok {
		t.Fatalf("missing global %q", "title")
	}
	if info.TypeName != "str" {
		t.Fatalf("global title type = %q, want str", info.TypeName)
	}
	data := checked.GlobalDataByModule[""]
	if info.DataIndex < 0 || info.DataIndex+1 >= len(data) {
		t.Fatalf("global title data index out of range: %d len=%d", info.DataIndex, len(data))
	}
	if got := binary.LittleEndian.Uint64(data[info.DataIndex+1]); got != 5 {
		t.Fatalf("global title len slot = %d, want 5", got)
	}
}

func TestGlobalStringFieldAccessSemantics(t *testing.T) {
	src := `
val greeting: String = "hello wasm"

func main() -> Int:
    return greeting.len
`
	file, err := ParseFile([]byte(src), "main.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "",
		Files:       []*FileAST{file},
		ByModule:    map[string]*FileAST{"": file},
	}
	if _, err := CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
}

func TestBuildGlobalVarStringLiteralInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var title: String = "hello"

func main() -> Int:
    return title[1]
`
	_, code := buildAndRun(t, src)
	if code != int('e') {
		t.Fatalf("exit code mismatch: got %d, want %d", code, int('e'))
	}
}

func TestBuildGlobalVarStringFieldAccessAfterAssignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var title: String = "hello"

func main() -> Int:
    title = "bye"
    return title.len
`
	_, code := buildAndRun(t, src)
	if code != 3 {
		t.Fatalf("exit code mismatch: got %d, want 3", code)
	}
}

func TestBuildLocalStringFieldAccessSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int:
    let s: String = "bye"
    return s.len
`
	_, code := buildAndRun(t, src)
	if code != 3 {
		t.Fatalf("exit code mismatch: got %d, want 3", code)
	}
}

func TestBuildPropertyStringLiteralInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
property title: String = "hello"

func main() -> Int:
    return title[4]
`
	_, code := buildAndRun(t, src)
	if code != int('o') {
		t.Fatalf("exit code mismatch: got %d, want %d", code, int('o'))
	}
}

func TestGlobalVarPtrZeroInitializerSemantics(t *testing.T) {
	src := `
var p: ptr = 0

func main() -> Int:
    return 42
`
	file, err := ParseFile([]byte(src), "main.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "",
		Files:       []*FileAST{file},
		ByModule:    map[string]*FileAST{"": file},
	}
	if _, err := CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
}

func TestBuildGlobalVarPtrZeroInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var p: ptr = 0

func main() -> Int:
    return 42
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestGlobalVarPtrConstZeroInitializerSemantics(t *testing.T) {
	src := `
const z: Int = 0
var p: ptr = z

func main() -> Int:
    return 42
`
	file, err := ParseFile([]byte(src), "main.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &World{
		EntryModule: "",
		Files:       []*FileAST{file},
		ByModule:    map[string]*FileAST{"": file},
	}
	if _, err := CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
}

func TestBuildGlobalVarExpandedScalarInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var b: UInt8 = 41
var w: UInt16 = 1
var terr: task.error = 0

func main() -> Int:
    if terr != 0:
        return 0
    return b + w
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildPropertyExplicitInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
property base: Int = 40 + 2

func main() -> Int:
    return base
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestConstGlobalExpressionDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "unknown const",
			src: `const answer: i32 = missing + 1

func main() -> Int:
    return answer
`,
			want: "unknown constant 'missing'",
		},
		{
			name: "division by zero",
			src: `const answer: i32 = 1 / 0

func main() -> Int:
    return answer
`,
			want: "division by zero in global const expression",
		},
		{
			name: "modulo by zero",
			src: `const answer: i32 = 1 % 0

func main() -> Int:
    return answer
`,
			want: "modulo by zero in global const expression",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := buildOnly(t, tt.src)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got: %v", tt.want, err)
			}
		})
	}
}

func TestGlobalVarInitializerDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "non constant int initializer call",
			src: `
var answer: Int = forty() + 2

func forty() -> Int:
    return 40

func main() -> Int:
    return answer
`,
			want: "global var 'answer' initializer must be an i32 constant expression",
		},
		{
			name: "non constant bool initializer call",
			src: `
var enabled: Bool = enabled_now()

func enabled_now() -> Bool:
    return true

func main() -> Int:
    if enabled:
        return 1
    return 0
`,
			want: "global var 'enabled' initializer must be a bool constant expression",
		},
		{
			name: "string initializer must be literal",
			src: `
var msg: String = msg_now()

func msg_now() -> String:
    return "hi"

func main() -> Int:
    return 0
`,
			want: "global var 'msg' initializer must be a string literal",
		},
		{
			name: "u8 initializer out of range",
			src: `
var b: UInt8 = 256

func main() -> Int:
    return 0
`,
			want: "global var 'b' initializer must be within 0..255 for type u8",
		},
		{
			name: "u16 initializer out of range",
			src: `
var w: UInt16 = 70000

func main() -> Int:
    return 0
`,
			want: "global var 'w' initializer must be within 0..65535 for type u16",
		},
		{
			name: "ptr initializer non-zero for var",
			src: `
var p: ptr = 1

func main() -> Int:
    return 0
`,
			want: "global var 'p' of type ptr only supports initializer 0",
		},
		{
			name: "ptr initializer non-constant for var",
			src: `
var p: ptr = zero()

func zero() -> Int:
    return 0

func main() -> Int:
    return 0
`,
			want: "global var 'p' initializer for type ptr must be a constant 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := buildOnly(t, tt.src)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got: %v", tt.want, err)
			}
		})
	}
}

func TestPropertyInitializerDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "property string initializer must be literal",
			src: `
property title: String = title_now()

func title_now() -> String:
    return "hello"

func main() -> Int:
    return 0
`,
			want: "global val 'title' initializer must be a string literal",
		},
		{
			name: "property u16 initializer out of range",
			src: `
property w: UInt16 = 70000

func main() -> Int:
    return 0
`,
			want: "global val 'w' initializer must be within 0..65535 for type u16",
		},
		{
			name: "non constant property bool initializer",
			src: `
property enabled: Bool = enabled_now()

func enabled_now() -> Bool:
    return true

func main() -> Int:
    if enabled:
        return 1
    return 0
`,
			want: "global val 'enabled' initializer must be a bool constant expression",
		},
		{
			name: "property ptr non-zero initializer",
			src: `
property p: ptr = 1

func main() -> Int:
    return 0
`,
			want: "global val 'p' of type ptr only supports initializer 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := buildOnly(t, tt.src)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got: %v", tt.want, err)
			}
		})
	}
}

func TestBuildLocalConstSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    const answer: Int = 42
    return answer
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestFormatSourceLocalConst(t *testing.T) {
	src := []byte(`func main() -> Int:
    const answer: Int = 42
    return answer
`)
	got, err := FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func main() -> Int:
    const answer: Int = 42
    return answer
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}
