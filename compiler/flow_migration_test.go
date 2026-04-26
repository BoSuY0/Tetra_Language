package compiler

import (
	"strings"
	"testing"
)

func TestNormalizeFlowForMigrationAPI(t *testing.T) {
	src := []byte("func main() -> Int:\n    let x: Int = 1\n    if x > 0:\n        return x\n")
	got, err := NormalizeFlowForMigration(src, "main.tetra")
	if err != nil {
		t.Fatalf("NormalizeFlowForMigration: %v", err)
	}
	out := string(got)
	for _, want := range []string{"fun main() -> Int {", "val x: Int = 1", "if (x > 0) {"} {
		if !strings.Contains(out, want) {
			t.Fatalf("normalized output missing %q:\n%s", want, out)
		}
	}
}
