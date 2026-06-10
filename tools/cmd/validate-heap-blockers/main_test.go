package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateHeapBlockersRejectsMissingBlocker(t *testing.T) {
	path := filepath.Join(t.TempDir(), "heap-blockers.json")
	raw := `{
  "schema_version":"tetra.ram-blockers.v1",
  "kind":"heap",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "rows":[{"site_id":"site","function":"main","intent":"heap_fallback","placement":"heap_unbounded","blockers":[],"contract_grade":"M5"}],
  "non_claims":["no Memory 100% claim"]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateHeapBlockers(path)
	if err == nil || !strings.Contains(err.Error(), "blockers") {
		t.Fatalf("validateHeapBlockers error = %v, want blockers rejection", err)
	}
}
