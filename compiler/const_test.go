package compiler

import (
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
