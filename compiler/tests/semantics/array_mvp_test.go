package compiler_test

import (
	"os"
	"path/filepath"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

func TestArrayMVPCheckAcceptsIndexAndForOnFixedArray(t *testing.T) {
	testkit.RequireCheckOK(t, `
func touch(seed: [3]Int) -> Int:
  var xs: [3]Int = seed
  xs[0] = 40
  xs[1] = 2
  xs[2] = xs[0] + xs[1]
  var total: Int = 0
  for x in xs:
    total = total + x
  return total

func main() -> Int:
  return 0
`)
}

func TestArrayMVPBuildSmoke(t *testing.T) {
	src := `func touch(seed: [3]Int) -> Int:
    var xs: [3]Int = seed
    xs[0] = 40
    xs[1] = 2
    xs[2] = xs[0] + xs[1]
    var total: Int = 0
    for x in xs:
        total = total + x
    return total

func main() -> Int:
    return 0
`
	if err := buildOnly(t, src); err != nil {
		t.Fatalf("build: %v", err)
	}
}

func TestArrayMVPBuildSupportsZeroedFixedArrayFieldGlobal(t *testing.T) {
	src := `struct ArrayBox:
    items: [2]Int

var leaked: ArrayBox

func configure(seed: [2]Int) -> Int:
    leaked.items = seed
    return leaked.items[0]

func main() -> Int:
    return 0
`
	if err := buildOnly(t, src); err != nil {
		t.Fatalf("build: %v", err)
	}
}

func TestArrayMVPBuildSupportsOptionalFixedArrayGlobal(t *testing.T) {
	src := `var maybe: [2]Int? = none

func configure(seed: [2]Int) -> Int:
    maybe = seed
    if let xs = maybe:
        return xs[0]
    else:
        return 0

func main() -> Int:
    return 0
`
	if err := buildOnly(t, src); err != nil {
		t.Fatalf("build: %v", err)
	}
}

func TestArrayMVPWasmBuildSmoke(t *testing.T) {
	src := `func touch(seed: [3]Int) -> Int:
    var xs: [3]Int = seed
    xs[0] = 40
    xs[1] = 2
    xs[2] = xs[0] + xs[1]
    var total: Int = 0
    for x in xs:
        total = total + x
    return total

func main() -> Int:
    return 0
`
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		outPath := filepath.Join(tmp, "app-"+target+".wasm")
		if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, target, compiler.BuildOptions{Jobs: 1}); err != nil {
			t.Fatalf("build %s: %v", target, err)
		}
	}
}

func buildOnly(t *testing.T, src string) error {
	t.Helper()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")

	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	return compiler.BuildFile(srcPath, outPath, "linux-x64")
}

func TestArrayMVPRejectsUnsupportedElementType(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int:
  let xs: [2]String = 0
  return 0
`, "array element type 'str' is not supported")
}

func TestArrayMVPRejectsNonPositiveSize(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int:
  let xs: [0]Int = 0
  return 0
`, "array size must be positive constant")
}

func TestArrayMVPRejectsAssignmentToArrayLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func probe(seed: [2]Int) -> Int:
  var xs: [2]Int = seed
  xs.len = 7
  return 0

func main() -> Int:
  return 0
`, "cannot assign to fixed-array internals ('ptr'/'len'); assign elements via index instead")
}

func TestArrayMVPRejectsAssignmentToArrayPtr(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func probe(seed: [2]Int) -> Int:
  var xs: [2]Int = seed
  xs.ptr = 0
  return 0

func main() -> Int:
  return 0
`, "cannot assign to fixed-array internals ('ptr'/'len'); assign elements via index instead")
}

func TestArrayMVPRejectsAssignmentToNestedArrayLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Box:
  arr: [2]Int

func probe(b0: Box) -> Int:
  var b: Box = b0
  b.arr.len = 3
  return 0

func main() -> Int:
  return 0
`, "cannot assign to fixed-array internals ('ptr'/'len'); assign elements via index instead")
}
