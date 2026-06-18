package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfacevisual"
)

func TestSurfaceGoldenWritesValidatedVisualReportAndScenePNGs(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "golden")
	var stdout, stderr bytes.Buffer
	code := runSurfaceGolden([]string{"--out", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	reportPath := filepath.Join(outDir, "surface-visual-report.json")
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read visual report: %v", err)
	}
	if err := surfacevisual.ValidateReportWithRoot(raw, outDir); err != nil {
		t.Fatalf("ValidateReportWithRoot generated report = %v\n%s", err, raw)
	}
	for _, scene := range surfacevisual.RequiredScenes() {
		for _, rel := range []string{
			"baselines/" + scene + ".png",
			"current/" + scene + ".png",
			"diff/" + scene + ".png",
		} {
			if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
				t.Fatalf("missing generated %s: %v", rel, err)
			}
		}
	}
	if !strings.Contains(stdout.String(), "surface visual golden report") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestSurfaceGoldenRejectsMissingSceneDirectory(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "golden")
	var stdout, stderr bytes.Buffer
	code := runSurfaceGolden([]string{"--out", outDir, "--scene-dir", filepath.Join(t.TempDir(), "missing")}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q; want missing scene-dir rejection", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "scene directory") {
		t.Fatalf("stderr = %q, want scene directory rejection", stderr.String())
	}
}
