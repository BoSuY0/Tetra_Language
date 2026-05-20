package lower

import "testing"

func TestLowerFilesystemExistsBuiltinUsesRuntimeCall(t *testing.T) {
	prog := lowerCallableProgram(t, `
func probe(cap: cap.io) -> Bool
uses io:
    return core.fs_exists("README.md", cap)

func main() -> Int:
    return 0
`)
	probe := requireCallableFunc(t, prog, "probe")
	if countCall(probe.Instrs, "__tetra_fs_exists", 3, 1) != 1 {
		t.Fatalf("probe did not lower core.fs_exists to __tetra_fs_exists(3 -> 1): %#v", probe.Instrs)
	}
}
