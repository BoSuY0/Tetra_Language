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
		Branch:    "release/v0.1.3",
		Version:   "v0.1.3",
		GitStatus: nil,
		ReadFile:  readFile,
		StatFile:  statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	if len(report.GeneratedArtifacts.Missing) == 0 {
		t.Fatalf("expected missing generated artifacts")
	}
}

func TestReleaseStateJSONAndTextReportEvidence(t *testing.T) {
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
		Version:   "v0.1.3",
		GitStatus: parseGitStatus(" M README.md\n"),
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
	for _, want := range []string{"status: pass", "branch: main", "version: v0.1.3", "dirty tracked files: 1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("text report missing %q:\n%s", want, text)
		}
	}
}

func TestReleaseStateRejectsStaleGateStepCount(t *testing.T) {
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	files["docs/generated/v1_0/release_gate_summary.json"] = []byte(`{"status":"pass","step_count":28,"failed_count":0}`)
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
		Version:   "v0.1.3",
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
