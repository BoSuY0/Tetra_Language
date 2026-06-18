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

func TestValidateValidationPipelineCoverageRejectsMissingBuildFileEntrypoint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coverage.json")
	raw := strings.Replace(
		validPipelineCoverageForTest(),
		`{"entrypoint":"BuildFileWithStatsOpt","artifact_path":"app","status":"validated_by_pipeline","validators":["ramcontract.ValidateReport"]},`,
		"",
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateValidationPipelineCoverage(path)
	if err == nil || !strings.Contains(err.Error(), "BuildFileWithStatsOpt") {
		t.Fatalf(
			"validateValidationPipelineCoverage error = %v, want missing BuildFileWithStatsOpt rejection",
			err,
		)
	}
}

func TestValidateValidationPipelineCoverageRejectsMissingObjectEntrypointWhenObjectArtifactPresent(
	t *testing.T,
) {
	path := filepath.Join(t.TempDir(), "coverage.json")
	raw := strings.Replace(
		validPipelineCoverageForTest(),
		`{"entrypoint":"buildObjectFileWithStatsOpt","status":"formal_exemption_with_reason","exemption":"not exercised by this linux-x64 RAM release fixture; object builds must carry their own RAM coverage evidence"},`,
		"",
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateValidationPipelineCoverage(path)
	if err == nil || !strings.Contains(err.Error(), "buildObjectFileWithStatsOpt") {
		t.Fatalf(
			"validateValidationPipelineCoverage error = %v, want missing object entrypoint rejection",
			err,
		)
	}
}

func TestValidateValidationPipelineCoverageRejectsInvalidExemption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coverage.json")
	raw := strings.Replace(
		validPipelineCoverageForTest(),
		("not exercised by this linux-x64 RAM release fixture; object " +
			"builds must carry their own RAM coverage evidence"),
		"todo",
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateValidationPipelineCoverage(path)
	if err == nil || !strings.Contains(err.Error(), "exemption") {
		t.Fatalf(
			"validateValidationPipelineCoverage error = %v, want invalid exemption rejection",
			err,
		)
	}
}

func validPipelineCoverageForTest() string {
	return `{
  "schema_version":"tetra.validation-pipeline-coverage.v1",
  "git_head":"e2c19b8ee276158f8eb2c54cf61e11bd84952893",
  "target":"linux-x64",
  "generated_by":"test",
  "entries":[
    {"entrypoint":"BuildFileWithStatsOpt","artifact_path":"app","status":"validated_by_pipeline","validators":["ramcontract.ValidateReport"]},
    {"entrypoint":"buildObjectFileWithStatsOpt","status":"formal_exemption_with_reason","exemption":"not exercised by this linux-x64 RAM release fixture; object builds must carry their own RAM coverage evidence"},
    {"entrypoint":"buildLibraryObjectWithStatsOpt","status":"formal_exemption_with_reason","exemption":"not exercised by this linux-x64 RAM release fixture; library builds must carry their own RAM coverage evidence"},
    {"entrypoint":"InterfaceOnly","status":"formal_exemption_with_reason","exemption":"interface-only mode does not produce a RAM artifact in this release fixture"},
    {"entrypoint":"wasm32-wasi-build","status":"formal_exemption_with_reason","exemption":"wasm32-wasi RAM coverage is target-specific and not claimed by this linux-x64 release fixture"},
    {"entrypoint":"wasm32-web-build","status":"formal_exemption_with_reason","exemption":"wasm32-web RAM coverage is target-specific and not claimed by this linux-x64 release fixture"},
    {"entrypoint":"explain-report-path","status":"formal_exemption_with_reason","exemption":"explain report path is not artifact-producing in this release fixture"}
  ],
  "non_claims":["no full formal proof claim"]
}`
}
