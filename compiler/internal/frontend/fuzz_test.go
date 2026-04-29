package frontend

import (
	"testing"
)

func FuzzLexer(f *testing.F) {
	f.Add([]byte("fn main(): i32 { return 42 }"))
	f.Add([]byte("let x = 1 + 2 * 3"))
	f.Add([]byte("if (a > b) { return a } else { return b }"))
	f.Add([]byte(""))
	f.Add([]byte{0x00, 0xFF})
	f.Add([]byte("a && b || c != d >= e <= f"))
	f.Add([]byte("10 / 3 % 2 * 5"))
	f.Add([]byte(`"hello\n\t\\\"world"`))
	f.Add([]byte("// comment\n42"))
	f.Add([]byte("test \"math\":\n    expect 40 + 2 == 42\n"))
	f.Add([]byte("test \"bad\\q\":\n    expect 1 == 1\n"))
	f.Add([]byte("test \"Привіт\":\r\n\texpect 1 == 1\r\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		l := newLexer(data, "fuzz")
		for {
			tok, err := l.nextToken()
			if err != nil {
				return
			}
			if tok.typ == TokenEOF {
				return
			}
		}
	})
}

func FuzzParser(f *testing.F) {
	f.Add([]byte("fn main(): i32 { return 0 }"))
	f.Add([]byte("fn f(a: i32, b: i32): i32 { return a * b + 1 }"))
	f.Add([]byte("fun main(): i32 { val x: i32 = 1 + 2; return x }"))
	f.Add([]byte("struct S { x: i32 } fn main() -> i32 { return 0 }"))
	f.Add([]byte("fn main() -> i32 { if (1) { return 1 } else { return 0 } return 0 }"))
	f.Add([]byte("fn main() -> i32 { while (0) { return 1 } return 0 }"))
	f.Add([]byte("fn main() -> i32 { return a && b || c }"))
	f.Add([]byte("fn main() -> i32 { return 6 * 7 }"))
	f.Add([]byte("test \"math\":\n    expect 40 + 2 == 42\n"))
	f.Add([]byte("test math:\n    expect 1 == 1\n"))
	f.Add([]byte("test \"Привіт\":\r\n    expect @\r\n"))
	f.Add([]byte(""))
	f.Add([]byte{0x00, 0xFF})

	f.Fuzz(func(t *testing.T, data []byte) {
		Parse(data) // must not panic
	})
}
