package compiler_test

import (
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func checkCapsuleFileProgram(src string) error {
	file, err := compiler.ParseFile([]byte(src), "capsule_mvp.tetra")
	if err != nil {
		return err
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		return err
	}
	_, err = compiler.Lower(checked)
	return err
}

func TestCapsuleMetadataPositiveNoRuntimeImpact(t *testing.T) {
	src := `
capsule App:
    id: "tetra://app"
    version: "0.1.0"
    target: "linux-x64"
    debug.enabled: true

func main() -> Int:
    return 0
`
	if err := checkCapsuleFileProgram(src); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestCapsuleMetadataSemanticDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "duplicate key",
			src: `
capsule App:
    id: "tetra://app"
    id: "tetra://dup"

func main() -> Int:
    return 0
`,
			want: "duplicate capsule metadata key 'id'",
		},
		{
			name: "invalid key shape",
			src: `
capsule App:
    ID: "tetra://app"

func main() -> Int:
    return 0
`,
			want: "invalid capsule metadata key 'ID'",
		},
		{
			name: "invalid value shape",
			src: `
capsule App:
    version: 1 + 2

func main() -> Int:
    return 0
`,
			want: "capsule metadata value for key 'version' must be a literal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkCapsuleFileProgram(tt.src)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestCapsuleAndPropertyAcceptedTogether(t *testing.T) {
	src := `
capsule App:
    id: "tetra://app"

property title: Int = 7

func main() -> Int:
    return title
`
	if err := checkCapsuleFileProgram(src); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}
