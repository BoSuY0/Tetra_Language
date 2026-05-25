package fuzz_test

import (
	"bytes"
	"fmt"
	"testing"

	compiler "tetra_language/compiler"
)

func FuzzFormatSourceIdempotent(f *testing.F) {
	for _, seed := range []string{
		"func main() -> Int:\n    return 0\n",
		"// leading comment\nfunc main() -> Int:\n    let x: Int = 1\n    return x\n",
		"func add(a: Int, b: Int) -> Int:\n    return a + b\n\nfunc main() -> Int:\n    return add(1, 2)\n",
		"func main() -> Int:\n    if 1 < 2:\n        return 1\n    return 0\n",
	} {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, raw []byte) {
		formatted, err := compiler.FormatSource(raw, "fuzz_format.tetra")
		if err != nil {
			return
		}
		again, err := compiler.FormatSource(formatted, "fuzz_format.tetra")
		if err != nil {
			t.Fatalf("formatted source did not reformat: %v\n%s", err, formatted)
		}
		if !bytes.Equal(formatted, again) {
			t.Fatalf("FormatSource is not idempotent\nfirst:\n%s\nsecond:\n%s", formatted, again)
		}
	})
}

func FuzzLoweringPipelineVerifiesIR(f *testing.F) {
	for _, seed := range [][]byte{
		{0, 1, 2, 3},
		{7, 11, 1, 0},
		{19, 23, 2, 5},
		{41, 1, 3, 8},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		src := loweringFuzzSource(data)
		prog, err := compiler.Parse([]byte(src))
		if err != nil {
			t.Fatalf("generated source did not parse: %v\n%s", err, src)
		}
		checked, err := compiler.Check(prog)
		if err != nil {
			t.Fatalf("generated source did not check: %v\n%s", err, src)
		}
		irProg, err := compiler.Lower(checked)
		if err != nil {
			t.Fatalf("generated source did not lower: %v\n%s", err, src)
		}
		if err := compiler.VerifyIRProgram(irProg); err != nil {
			t.Fatalf("generated source did not verify: %v\n%s", err, src)
		}
	})
}

func loweringFuzzSource(data []byte) string {
	a := boundedFuzzInt(data, 0)
	b := boundedFuzzInt(data, 1)
	c := boundedFuzzInt(data, 2)

	switch byteAt(data, 3) % 4 {
	case 0:
		return fmt.Sprintf("func main() -> Int:\n    let x: Int = %d\n    let y: Int = %d\n    return x + y\n", a, b)
	case 1:
		return fmt.Sprintf("func main() -> Int:\n    if %d < %d:\n        return %d\n    return %d\n", a, b, b, a)
	case 2:
		return fmt.Sprintf("func add(a: Int, b: Int) -> Int:\n    return a + b\n\nfunc main() -> Int:\n    return add(%d, %d)\n", a, b)
	default:
		limit := c % 8
		return fmt.Sprintf("func main() -> Int:\n    var total: Int = %d\n    for i in 0..<%d:\n        total = total + i\n    return total\n", a, limit)
	}
}

func boundedFuzzInt(data []byte, index int) int {
	return int(byteAt(data, index) % 64)
}

func byteAt(data []byte, index int) byte {
	if index < len(data) {
		return data[index]
	}
	return 0
}
