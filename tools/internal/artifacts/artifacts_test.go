package artifacts

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidateRequiredReportsAcceptsRegularNonEmptyReports(t *testing.T) {
	reportRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(reportRoot, "nested"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportRoot, "summary.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportRoot, "nested", "detail.json"), []byte("{\"ok\":true}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := ValidateRequiredReports(reportRoot, []string{"summary.json", "nested/detail.json"})
	if err != nil {
		t.Fatalf("ValidateRequiredReports: %v", err)
	}
	want := []string{
		filepath.Join(reportRoot, "summary.json"),
		filepath.Join(reportRoot, "nested", "detail.json"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ValidateRequiredReports paths = %#v, want %#v", got, want)
	}
}

func TestValidateRequiredReportRejectsUnsafeOrInvalidReports(t *testing.T) {
	reportRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(reportRoot, "empty.json"), nil, 0o644); err != nil {
		t.Fatalf("WriteFile empty: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(reportRoot, "dir-report"), 0o755); err != nil {
		t.Fatalf("MkdirAll dir-report: %v", err)
	}
	if err := os.WriteFile(filepath.Join(reportRoot, "real.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile real: %v", err)
	}
	if err := os.Symlink(filepath.Join(reportRoot, "real.json"), filepath.Join(reportRoot, "link.json")); err != nil {
		t.Fatalf("Symlink file: %v", err)
	}

	cases := []struct {
		name string
		path string
		want string
	}{
		{name: "empty", path: "", want: "empty"},
		{name: "absolute", path: filepath.Join(reportRoot, "real.json"), want: "absolute"},
		{name: "parent traversal", path: "../real.json", want: "parent traversal"},
		{name: "missing", path: "missing.json", want: "missing"},
		{name: "empty file", path: "empty.json", want: "empty file"},
		{name: "symlink file", path: "link.json", want: "symlink"},
		{name: "directory", path: "dir-report", want: "not a regular file"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := ValidateRequiredReport(reportRoot, tc.path); err == nil {
				t.Fatalf("ValidateRequiredReport(%q) = %q, nil error", tc.path, got)
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateRequiredReport(%q) error = %v, want substring %q", tc.path, err, tc.want)
			}
		})
	}
}

func TestHashCommandPlanUsesStableManifestContract(t *testing.T) {
	reportRoot := filepath.Join("reports", "surface-release-v1")

	plan, err := NewHashCommandPlan(reportRoot)
	if err != nil {
		t.Fatalf("NewHashCommandPlan: %v", err)
	}

	manifestPath := filepath.Join(reportRoot, HashManifestName)
	if HashManifestName != "artifact-hashes.json" {
		t.Fatalf("HashManifestName = %q", HashManifestName)
	}
	if HashManifestSchema != "tetra.release-artifact-hashes.v1alpha1" {
		t.Fatalf("HashManifestSchema = %q", HashManifestSchema)
	}
	if plan.ManifestPath != manifestPath {
		t.Fatalf("ManifestPath = %q, want %q", plan.ManifestPath, manifestPath)
	}
	wantWrite := CommandPlan{
		Name: "artifact-hashes-write",
		Args: []string{
			"go", "run", "./tools/cmd/validate-artifact-hashes",
			"--write", "--root", reportRoot,
			"--out", manifestPath,
		},
	}
	wantValidate := CommandPlan{
		Name: "artifact-hashes-validate",
		Args: []string{
			"go", "run", "./tools/cmd/validate-artifact-hashes",
			"--manifest", manifestPath,
		},
	}
	if !reflect.DeepEqual(plan.Write, wantWrite) {
		t.Fatalf("Write command = %#v, want %#v", plan.Write, wantWrite)
	}
	if !reflect.DeepEqual(plan.Validate, wantValidate) {
		t.Fatalf("Validate command = %#v, want %#v", plan.Validate, wantValidate)
	}
}

func TestHashCommandPlanRejectsEmptyReportRoot(t *testing.T) {
	if _, err := NewHashCommandPlan(""); err == nil {
		t.Fatalf("NewHashCommandPlan accepted empty report root")
	}
}
