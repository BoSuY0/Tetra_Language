package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCopyBlockersRejectsMissingCopyReason(t *testing.T) {
	path := filepath.Join(t.TempDir(), "copy-blockers.json")
	raw := `{
  "schema_version":"tetra.ram-blockers.v1",
  "kind":"copy",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "rows":[{"site_id":"site","function":"main","intent":"copy_heap_bounded","placement":"heap_bounded","contract_grade":"M4"}],
  "non_claims":["no Memory 100% claim"]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateCopyBlockers(path)
	if err == nil || !strings.Contains(err.Error(), "copy_reason") {
		t.Fatalf("validateCopyBlockers error = %v, want copy_reason rejection", err)
	}
}
