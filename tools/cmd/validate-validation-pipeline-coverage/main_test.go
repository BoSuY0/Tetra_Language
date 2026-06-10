package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateValidationPipelineCoverageRejectsArtifactWithoutValidators(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coverage.json")
	raw := `{
  "schema_version":"tetra.validation-pipeline-coverage.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "entries":[{"entrypoint":"BuildFileWithStatsOpt","artifact_path":"app","status":"validated_by_pipeline"}],
  "non_claims":["no full formal proof claim"]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateValidationPipelineCoverage(path)
	if err == nil || !strings.Contains(err.Error(), "validators") {
		t.Fatalf("validateValidationPipelineCoverage error = %v, want validators rejection", err)
	}
}
