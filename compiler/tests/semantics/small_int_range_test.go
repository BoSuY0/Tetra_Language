package compiler_test

import (
	"testing"

	"tetra_language/compiler/internal/testkit"
)

func TestSmallIntLiteralRangeBoundaries(t *testing.T) {
	testkit.RequireCheckOK(t, `
func take_byte(value: UInt8) -> Int:
    return value

func take_word(value: UInt16) -> Int:
    return value

func byte_value() -> UInt8:
    return 255

func word_value() -> UInt16:
    return 65535

func main() -> Int:
    let b: UInt8 = 0
    let max_b: UInt8 = 255
    let expr_b: UInt8 = 128 + 127
    let w: UInt16 = 65530 + 5
    return take_byte(b) + take_byte(max_b) + take_word(w) + byte_value() + word_value()
`)
}

func TestSmallIntLiteralRangeRejectsOutOfRangeContextualValues(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "u8 local above max",
			src: `
func main() -> Int:
    let b: UInt8 = 256
    return b
`,
			want: "type mismatch: expected 'u8', got 'i32'",
		},
		{
			name: "u8 local below zero",
			src: `
func main() -> Int:
    let b: UInt8 = -1
    return b
`,
			want: "type mismatch: expected 'u8', got 'i32'",
		},
		{
			name: "u16 local above max",
			src: `
func main() -> Int:
    let w: UInt16 = 70000
    return w
`,
			want: "type mismatch: expected 'u16', got 'i32'",
		},
		{
			name: "u8 binary expression below zero",
			src: `
func main() -> Int:
    let b: UInt8 = 0 - 1
    return b
`,
			want: "type mismatch: expected 'u8', got 'i32'",
		},
		{
			name: "u8 binary expression above max",
			src: `
func main() -> Int:
    let b: UInt8 = 250 + 6
    return b
`,
			want: "type mismatch: expected 'u8', got 'i32'",
		},
		{
			name: "u16 binary expression above max",
			src: `
func main() -> Int:
    let w: UInt16 = 60000 + 6000
    return w
`,
			want: "type mismatch: expected 'u16', got 'i32'",
		},
		{
			name: "u16 overflow expression",
			src: `
func main() -> Int:
    let w: UInt16 = 65536 * 65536
    return w
`,
			want: "type mismatch: expected 'u16', got 'i32'",
		},
		{
			name: "function argument",
			src: `
func take_byte(value: UInt8) -> Int:
    return value

func main() -> Int:
    return take_byte(256)
`,
			want: "type mismatch for 'take_byte' arg 1",
		},
		{
			name: "function return",
			src: `
func byte_value() -> UInt8:
    return 300

func main() -> Int:
    return 0
`,
			want: "return type mismatch: expected 'u8', got 'i32'",
		},
		{
			name: "throw value",
			src: `
func fail() -> Int throws UInt8:
    throw 300

func main() -> Int:
    return 0
`,
			want: "throw type mismatch: expected 'u8', got 'i32'",
		},
		{
			name: "struct field",
			src: `
struct Header:
    byte: UInt8

func main() -> Int:
    let h: Header = Header(byte: 300)
    return h.byte
`,
			want: "type mismatch for field 'byte'",
		},
		{
			name: "enum payload",
			src: `
enum Packet:
    case byte(UInt8)

func main() -> Int:
    let p: Packet = Packet.byte(300)
    return 0
`,
			want: "enum case 'Packet.byte' payload 1 expects 'u8', got 'i32'",
		},
		{
			name: "local assignment",
			src: `
func main() -> Int:
    var b: UInt8 = 0
    b = 300
    return b
`,
			want: "type mismatch: expected 'u8', got 'i32'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.want)
		})
	}
}
