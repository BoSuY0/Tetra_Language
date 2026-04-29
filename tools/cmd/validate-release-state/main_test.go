package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseGitStatusClassifiesDirtyTrackedAndUntrackedReleaseArtifacts(t *testing.T) {
	entries := parseGitStatus(` M README.md
?? docs/generated/v1_0/release-state.json
?? scratch.txt
R  old.md -> docs/release/new.md
`)
	report := classifyGitStatus(entries)
	if got, want := strings.Join(report.DirtyTracked, ","), "README.md,docs/release/new.md"; got != want {
		t.Fatalf("dirty tracked = %q, want %q", got, want)
	}
	if got, want := strings.Join(report.UntrackedReleaseArtifacts, ","), "docs/generated/v1_0/release-state.json"; got != want {
		t.Fatalf("untracked release artifacts = %q, want %q", got, want)
	}
}

func TestAuditRequiredArtifactsFailsWhenReleaseEvidenceMissing(t *testing.T) {
	readFile := func(path string) ([]byte, error) {
		if path == "docs/generated/v1_0/release_gate_summary.json" {
			return []byte(`{"status":"pass","step_count":2,"failed_count":0}`), nil
		}
		return nil, errNotExist(path)
	}
	statFile := func(path string) (fileInfo, error) {
		if path == "docs/generated/v1_0/release_gate_summary.json" {
			return fakeFileInfo{size: 48}, nil
		}
		return nil, errNotExist(path)
	}
	report := buildReleaseStateReport(releaseStateInputs{
		Branch:          "release/v0.1.3",
		Version:         "v0.1.3",
		ExpectedVersion: "v0.1.3",
		GitStatus:       nil,
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	if len(report.GeneratedArtifacts.Missing) == 0 {
		t.Fatalf("expected missing generated artifacts")
	}
}

func TestReleaseStateJSONAndTextReportEvidence(t *testing.T) {
	reportDir := "/tmp/tetra-v0_2_0-gate"
	files := releaseStatePassFiles(reportDir)
	readFile := func(path string) ([]byte, error) {
		raw, ok := files[path]
		if !ok {
			return nil, errNotExist(path)
		}
		return raw, nil
	}
	statFile := func(path string) (fileInfo, error) {
		raw, ok := files[path]
		if !ok {
			return nil, errNotExist(path)
		}
		return fakeFileInfo{size: int64(len(raw))}, nil
	}
	report := buildReleaseStateReport(releaseStateInputs{
		Branch:    "main",
		Version:   "v0.2.0",
		ReportDir: reportDir,
		GitStatus: parseGitStatus("?? docs/generated/v1_0/release-state.json\n"),
		ReadFile:  readFile,
		StatFile:  statFile,
	})
	if report.Status != "pass" {
		t.Fatalf("status = %q issues=%v", report.Status, report.Issues)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"dirty_tracked"`) {
		t.Fatalf("json missing dirty tracked: %s", raw)
	}
	text := formatTextReport(report)
	for _, want := range []string{"status: pass", "branch: main", "version: v0.2.0", "dirty tracked files: 0"} {
		if !strings.Contains(text, want) {
			t.Fatalf("text report missing %q:\n%s", want, text)
		}
	}
}

func TestReleaseStateFailsWhenDirtyTrackedFilesPresent(t *testing.T) {
	reportDir := "/tmp/tetra-v0_2_0-gate"
	files := releaseStatePassFiles(reportDir)
	readFile := func(path string) ([]byte, error) {
		raw, ok := files[path]
		if !ok {
			return nil, errNotExist(path)
		}
		return raw, nil
	}
	statFile := func(path string) (fileInfo, error) {
		raw, ok := files[path]
		if !ok {
			return nil, errNotExist(path)
		}
		return fakeFileInfo{size: int64(len(raw))}, nil
	}
	report := buildReleaseStateReport(releaseStateInputs{
		Branch:    "main",
		Version:   "v0.2.0",
		ReportDir: reportDir,
		GitStatus: parseGitStatus(" M README.md\n"),
		ReadFile:  readFile,
		StatFile:  statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	if got := strings.Join(report.Issues, "\n"); !strings.Contains(got, "dirty tracked files detected") {
		t.Fatalf("issues did not mention dirty tracked files: %v", report.Issues)
	}
}

func TestReleaseStateFailsWhenUntrackedNonReleaseEntriesPresent(t *testing.T) {
	reportDir := "/tmp/tetra-v0_2_0-gate"
	files := releaseStatePassFiles(reportDir)
	readFile := func(path string) ([]byte, error) {
		raw, ok := files[path]
		if !ok {
			return nil, errNotExist(path)
		}
		return raw, nil
	}
	statFile := func(path string) (fileInfo, error) {
		raw, ok := files[path]
		if !ok {
			return nil, errNotExist(path)
		}
		return fakeFileInfo{size: int64(len(raw))}, nil
	}
	report := buildReleaseStateReport(releaseStateInputs{
		Branch:    "main",
		Version:   "v0.2.0",
		ReportDir: reportDir,
		GitStatus: parseGitStatus("?? scratch.txt\n"),
		ReadFile:  readFile,
		StatFile:  statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	if got := strings.Join(report.Issues, "\n"); !strings.Contains(got, "untracked non-release entries detected") {
		t.Fatalf("issues did not mention untracked non-release entries: %v", report.Issues)
	}
}

func TestReleaseStateRejectsStaleGateStepCount(t *testing.T) {
	reportDir := "/tmp/tetra-v0_2_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.2.0", expectedReleaseArtifact("v0.2.0"), "bash scripts/release_v0_2_0_gate.sh", reportDir, 28, 0, "pass")
	readFile := func(path string) ([]byte, error) {
		raw, ok := files[path]
		if !ok {
			return nil, errNotExist(path)
		}
		return raw, nil
	}
	statFile := func(path string) (fileInfo, error) {
		raw, ok := files[path]
		if !ok {
			return nil, errNotExist(path)
		}
		return fakeFileInfo{size: int64(len(raw))}, nil
	}
	report := buildReleaseStateReport(releaseStateInputs{
		Branch:    "main",
		Version:   "v0.2.0",
		ReportDir: reportDir,
		GitStatus: nil,
		ReadFile:  readFile,
		StatFile:  statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	if got := strings.Join(report.Issues, "\n"); !strings.Contains(got, "want at least 33") {
		t.Fatalf("issues did not mention stale step count: %v", report.Issues)
	}
}

func TestReleaseStateRejectsStaleV020GateIdentity(t *testing.T) {
	reportDir := "/tmp/tetra-v0_2_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.1.3", "tetra.release.v0_1_3.gate-report.v1", "bash scripts/release_v0_1_3_gate.sh", reportDir, 33, 0, "pass")
	readFile := func(path string) ([]byte, error) {
		raw, ok := files[path]
		if !ok {
			return nil, errNotExist(path)
		}
		return raw, nil
	}
	statFile := func(path string) (fileInfo, error) {
		raw, ok := files[path]
		if !ok {
			return nil, errNotExist(path)
		}
		return fakeFileInfo{size: int64(len(raw))}, nil
	}
	report := buildReleaseStateReport(releaseStateInputs{
		Branch:    "main",
		Version:   "v0.2.0",
		ReportDir: reportDir,
		GitStatus: nil,
		ReadFile:  readFile,
		StatFile:  statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	got := strings.Join(report.Issues, "\n")
	for _, want := range []string{"release_version", "release_artifact", "release_gate_command"} {
		if !strings.Contains(got, want) {
			t.Fatalf("issues did not mention stale %s identity: %v", want, report.Issues)
		}
	}
}

func TestReleaseStateRequiresReportDirForV020(t *testing.T) {
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	files["docs/generated/v1_0/release_gate_summary.json"] = []byte(`{"status":"pass","step_count":33,"failed_count":0}`)
	readFile := func(path string) ([]byte, error) {
		raw, ok := files[path]
		if !ok {
			return nil, errNotExist(path)
		}
		return raw, nil
	}
	statFile := func(path string) (fileInfo, error) {
		raw, ok := files[path]
		if !ok {
			return nil, errNotExist(path)
		}
		return fakeFileInfo{size: int64(len(raw))}, nil
	}
	report := buildReleaseStateReport(releaseStateInputs{
		Branch:    "main",
		Version:   "v0.2.0",
		GitStatus: nil,
		ReadFile:  readFile,
		StatFile:  statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	if got := strings.Join(report.Issues, "\n"); !strings.Contains(got, "report-dir is required for v0.2.0 release validation") {
		t.Fatalf("issues did not mention v0.2.0 report-dir requirement: %v", report.Issues)
	}
}

func releaseSummaryJSON(version string, artifact string, command string, reportDir string, stepCount int, failedCount int, status string) []byte {
	raw, err := json.Marshal(map[string]interface{}{
		"status":               status,
		"release_version":      version,
		"release_artifact":     artifact,
		"release_gate_command": command,
		"report_dir":           reportDir,
		"step_count":           stepCount,
		"failed_count":         failedCount,
	})
	if err != nil {
		panic(err)
	}
	return raw
}

func releaseStatePassFiles(reportDir string) map[string][]byte {
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	files["docs/generated/v1_0/release_gate_summary.json"] = releaseSummaryJSON("v0.1.3", "tetra.release.v0_1_3.gate-report.v1", "bash scripts/release_v0_1_3_gate.sh", "/tmp/stale-v0_1_3-gate", 33, 0, "pass")
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.2.0", expectedReleaseArtifact("v0.2.0"), "bash scripts/release_v0_2_0_gate.sh", reportDir, 33, 0, "pass")
	return files
}
