package frontend

import (
	"strings"
	"testing"
)

func TestNormalizeFlowForMigrationRewritesCompatibilitySurface(t *testing.T) {
	src := []byte(`func main() -> Int:
    let x: Int = 1
    if x > 0:
        return x
    return 0
`)
	got, err := NormalizeFlowForMigration(src, "main.tetra")
	if err != nil {
		t.Fatalf("NormalizeFlowForMigration: %v", err)
	}
	out := string(got)
	for _, want := range []string{
		"fun main() -> Int {",
		"val x: Int = 1",
		"if (x > 0) {",
		"return x",
		"return 0",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("normalized output missing %q:\n%s", want, out)
		}
	}
}
