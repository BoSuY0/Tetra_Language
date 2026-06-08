package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestRepresentationMetadataAssignmentDiagnosticsNameFieldAndSafeNonAssignable(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "slice len",
			src: `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(2)
    xs.len = 9
    return 0
`,
			want: "representation metadata field 'len' is not user-assignable in safe code",
		},
		{
			name: "slice reserved owner",
			src: `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(2)
    xs.owner_id = 9
    return 0
`,
			want: "representation metadata field 'owner_id' is not user-assignable in safe code",
		},
		{
			name: "indexed through metadata",
			src: `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(2)
    xs.ptr[0] = 1
    return 0
`,
			want: "representation metadata field 'ptr' is not user-assignable in safe code",
		},
		{
			name: "generic wrapper",
			src: `
struct Box<T>:
    value: T

func main() -> Int
uses alloc, mem:
    var box: Box<[]u8> = Box<[]u8>{value: make_u8(2)}
    box.value.len = 9
    return 0
`,
			want: "representation metadata field 'len' is not user-assignable in safe code",
		},
		{
			name: "string metadata",
			src: `
func main() -> Int:
    var text: String = "*"
    text.ptr = 0
    return 0
`,
			want: "representation metadata field 'ptr' is not user-assignable in safe code",
		},
		{
			name: "fixed array metadata",
			src: `
func probe(seed: [2]Int) -> Int:
    var xs: [2]Int = seed
    xs.len = 7
    return 0

func main() -> Int:
    return 0
`,
			want: "representation metadata field 'len' is not user-assignable in safe code",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.want)
		})
	}
}

func TestRepresentationMetadataAssignmentRejectsImportedSliceField(t *testing.T) {
	requireCheckWorldFilesErrorContains(t, map[string]string{
		"src/lib/model.t4": `module lib.model

pub struct BufferBox:
    bytes: []u8
`,
		"src/app/main.t4": `module app.main
import lib.model as model

func main() -> Int
uses alloc, mem:
    var box: model.BufferBox = model.BufferBox(bytes: make_u8(2))
    box.bytes.len = 9
    return 0
`,
	}, "src/app/main.t4", "representation metadata field 'len' is not user-assignable in safe code")
}
