package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
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
		Branch:          "main",
		Version:         "v0.2.0",
		ExpectedVersion: "v0.2.0",
		ReportDir:       reportDir,
		GitStatus:       parseGitStatus("?? docs/generated/v1_0/release-state.json\n"),
		ReadFile:        readFile,
		StatFile:        statFile,
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

func TestReleaseStateReportsDirtyTrackedFilesWithoutFailing(t *testing.T) {
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
		Branch:          "main",
		Version:         "v0.2.0",
		ExpectedVersion: "v0.2.0",
		ReportDir:       reportDir,
		GitStatus:       parseGitStatus(" M README.md\n"),
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "pass" {
		t.Fatalf("status = %q issues=%v", report.Status, report.Issues)
	}
	if got, want := strings.Join(report.Git.DirtyTracked, ","), "README.md"; got != want {
		t.Fatalf("dirty tracked files = %q, want %q", got, want)
	}
}

func TestReleaseStateReportsUntrackedNonReleaseEntriesWithoutFailing(t *testing.T) {
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
		Branch:          "main",
		Version:         "v0.2.0",
		ExpectedVersion: "v0.2.0",
		ReportDir:       reportDir,
		GitStatus:       parseGitStatus("?? scratch.txt\n"),
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "pass" {
		t.Fatalf("status = %q issues=%v", report.Status, report.Issues)
	}
	if got, want := strings.Join(report.Git.UntrackedNonReleaseEntries, ","), "scratch.txt"; got != want {
		t.Fatalf("untracked non-release entries = %q, want %q", got, want)
	}
}

func TestReleaseStateRejectsStaleGateStepCount(t *testing.T) {
	reportDir := "/tmp/tetra-v0_2_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.2.0", expectedReleaseArtifact("v0.2.0"), "bash scripts/release/v0_2_0/gate.sh", reportDir, 28, 0, "pass")
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
		Branch:          "main",
		Version:         "v0.2.0",
		ExpectedVersion: "v0.2.0",
		ReportDir:       reportDir,
		GitStatus:       nil,
		ReadFile:        readFile,
		StatFile:        statFile,
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
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.1.3", "tetra.release.v0_1_3.gate-report.v1", "bash scripts/release/v0_1_3/gate.sh", reportDir, 33, 0, "pass")
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
		Branch:          "main",
		Version:         "v0.2.0",
		ExpectedVersion: "v0.2.0",
		ReportDir:       reportDir,
		GitStatus:       nil,
		ReadFile:        readFile,
		StatFile:        statFile,
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

func TestReleaseStateRejectsStaleV030GateIdentityForV040(t *testing.T) {
	reportDir := "/tmp/tetra-v0_4_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.3.0", expectedReleaseArtifact("v0.3.0"), expectedReleaseGateCommand("v0.3.0"), reportDir, 1, 1, "blocked")
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
		Branch:          "main",
		Version:         "v0.4.0",
		ExpectedVersion: "v0.4.0",
		ReportDir:       reportDir,
		GitStatus:       nil,
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	got := strings.Join(report.Issues, "\n")
	for _, want := range []string{
		`last gate evidence release_version is "v0.3.0", want "v0.4.0"`,
		`last gate evidence release_artifact is "tetra.release.v0_3_0.gate-report.v1", want "tetra.release.v0_4_0.gate-report.v1"`,
		`last gate evidence release_gate_command is "bash scripts/release/v0_3_0/gate.sh", want "bash scripts/release/v0_4_0/gate.sh"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("issues did not reject stale v0.3.0 identity %q: %v", want, report.Issues)
		}
	}
}

func TestReleaseStateAcceptsV030GateIdentityFromReportDir(t *testing.T) {
	reportDir := "/tmp/tetra-v0_3_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	addRuntimeEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	addSecurityReviewEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.3.0", expectedReleaseArtifact("v0.3.0"), "bash scripts/release/v0_3_0/gate.sh", reportDir, 10, 0, "pass")
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
		Branch:          "main",
		Version:         "v0.3.0",
		ExpectedVersion: "v0.3.0",
		ReportDir:       reportDir,
		GitHead:         "abcdef0",
		GitStatus:       nil,
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "pass" {
		t.Fatalf("status = %q issues=%v", report.Status, report.Issues)
	}
	if report.LastGateEvidence.SummaryPath != reportDir+"/summary.json" {
		t.Fatalf("summary path = %q", report.LastGateEvidence.SummaryPath)
	}
}

func TestReleaseStateRejectsGateCommandWithCanonicalScriptOnlyAsSubstring(t *testing.T) {
	reportDir := "/tmp/tetra-v1_0-gate"
	version := "v1.0.0"
	gitHead := "abc1234"
	files := releaseStateV1FreshFiles(reportDir, version, gitHead)
	files[reportDir+"/summary.json"] = releaseSummaryJSON(version, expectedReleaseArtifact(version), "env FOO=1 bash scripts/release/v1_0/gate.sh", reportDir, 32, 0, "pass")
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
		Branch:          "main",
		Version:         version,
		ExpectedVersion: version,
		ReportDir:       reportDir,
		GitHead:         gitHead,
		GitStatus:       nil,
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail for non-canonical release_gate_command", report.Status)
	}
	got := strings.Join(report.Issues, "\n")
	if !strings.Contains(got, `release_gate_command is "env FOO=1 bash scripts/release/v1_0/gate.sh", want "bash scripts/release/v1_0/gate.sh"`) {
		t.Fatalf("issues did not reject non-canonical release_gate_command exactly: %v", report.Issues)
	}
}

func TestReleaseStateAcceptsV030PreAuditSummary(t *testing.T) {
	reportDir := "/tmp/tetra-v0_3_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	addRuntimeEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	addSecurityReviewEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.3.0", expectedReleaseArtifact("v0.3.0"), "bash scripts/release/v0_3_0/gate.sh", reportDir, 8, 0, "pass")
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
		Branch:          "main",
		Version:         "v0.3.0",
		ExpectedVersion: "v0.3.0",
		ReportDir:       reportDir,
		GitHead:         "abcdef0",
		GitStatus:       nil,
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "pass" {
		t.Fatalf("status = %q issues=%v", report.Status, report.Issues)
	}
}

func TestReleaseStateRequiresV030MacOSAndWindowsRuntimeEvidence(t *testing.T) {
	reportDir := "/tmp/tetra-v0_3_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	addSecurityReviewEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.3.0", expectedReleaseArtifact("v0.3.0"), expectedReleaseGateCommand("v0.3.0"), reportDir, 12, 0, "pass")
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
		Branch:          "main",
		Version:         "v0.3.0",
		ExpectedVersion: "v0.3.0",
		ReportDir:       reportDir,
		GitHead:         "abcdef0",
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	got := strings.Join(report.Issues, "\n")
	for _, want := range []string{
		"missing required runtime execution evidence: " + reportDir + "/artifacts/macos-runtime-smoke.json",
		"missing required runtime execution evidence: " + reportDir + "/artifacts/windows-runtime-smoke.json",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("issues did not mention missing runtime evidence %q: %v", want, report.Issues)
		}
	}
	for _, check := range report.RuntimeExecution.Required {
		wantCommand := "./tetra smoke --target " + check.Target + " --run=true --report " + check.Path
		if check.EvidenceCommand != wantCommand {
			t.Fatalf("%s evidence command = %q, want %q", check.Target, check.EvidenceCommand, wantCommand)
		}
	}
}

func TestReleaseStateRecordsRuntimeExecutionEvidenceHosts(t *testing.T) {
	reportDir := "/tmp/tetra-v0_3_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	addRuntimeEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	addSecurityReviewEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.3.0", expectedReleaseArtifact("v0.3.0"), expectedReleaseGateCommand("v0.3.0"), reportDir, 12, 0, "pass")
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
		Branch:          "main",
		Version:         "v0.3.0",
		ExpectedVersion: "v0.3.0",
		ReportDir:       reportDir,
		GitHead:         "abcdef0",
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "pass" {
		t.Fatalf("status = %q issues=%v", report.Status, report.Issues)
	}
	got := map[string]string{}
	for _, check := range report.RuntimeExecution.Required {
		got[check.Target] = check.Host
	}
	for _, target := range []string{"macos-x64", "windows-x64"} {
		if got[target] != target {
			t.Fatalf("%s runtime evidence host = %q, want %q", target, got[target], target)
		}
	}
}

func TestReleaseStateDefersMissingReportDirSecurityReviewEvidenceToFinalGate(t *testing.T) {
	reportDir := "/tmp/tetra-v0_3_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	addRuntimeEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.3.0", expectedReleaseArtifact("v0.3.0"), expectedReleaseGateCommand("v0.3.0"), reportDir, 12, 0, "pass")
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
		Branch:          "main",
		Version:         "v0.3.0",
		ExpectedVersion: "v0.3.0",
		ReportDir:       reportDir,
		GitHead:         "abcdef0",
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "pass" {
		t.Fatalf("status = %q issues=%v", report.Status, report.Issues)
	}
	if report.SecurityReview.Status != "deferred" {
		t.Fatalf("security review status = %q, want deferred", report.SecurityReview.Status)
	}
	wantValidator := "bash scripts/release/v0_3_0/security-review.sh --signoff /tmp/tetra-v0_3_0-gate/artifacts/security-review.md"
	if report.SecurityReview.ValidatorCommand != wantValidator {
		t.Fatalf("security review validator command = %q, want %q", report.SecurityReview.ValidatorCommand, wantValidator)
	}
}

func TestTextReportIncludesSecurityReviewEvidenceState(t *testing.T) {
	text := formatTextReport(releaseStateReport{
		Status:          "pass",
		Branch:          "main",
		Version:         "v0.3.0",
		ExpectedVersion: "v0.3.0",
		ReportDir:       "/tmp/tetra-v0_3_0-gate",
		SecurityReview: securityReviewEvidenceReport{
			Path:             "/tmp/tetra-v0_3_0-gate/artifacts/security-review.md",
			HashPath:         "/tmp/tetra-v0_3_0-gate/artifacts/security-review.md.sha256",
			Status:           "deferred",
			ValidatorCommand: "bash scripts/release/v0_3_0/security-review.sh --signoff /tmp/tetra-v0_3_0-gate/artifacts/security-review.md",
		},
		LastGateEvidence: gateEvidenceReport{Status: "pass", SummaryPath: "/tmp/tetra-v0_3_0-gate/summary.json"},
	})
	if !strings.Contains(text, "security review evidence: deferred (/tmp/tetra-v0_3_0-gate/artifacts/security-review.md)") {
		t.Fatalf("text report missing security review evidence state:\n%s", text)
	}
	if !strings.Contains(text, "security review validator: bash scripts/release/v0_3_0/security-review.sh --signoff /tmp/tetra-v0_3_0-gate/artifacts/security-review.md") {
		t.Fatalf("text report missing security review validator command:\n%s", text)
	}
}

func TestTextReportIncludesLastGateIdentitySummary(t *testing.T) {
	text := formatTextReport(releaseStateReport{
		Status:          "fail",
		Branch:          "main",
		Version:         "v0.3.0",
		ExpectedVersion: "v0.3.0",
		LastGateEvidence: gateEvidenceReport{
			Status:             "pass",
			SummaryPath:        "docs/generated/v1_0/release_gate_summary.json",
			ReleaseVersion:     "",
			ReleaseArtifact:    "",
			ReleaseGateCommand: "",
		},
	})
	if !strings.Contains(text, `last gate identity: fail (release_version=<unknown>, release_artifact=<unknown>, release_gate_command=<unknown>)`) {
		t.Fatalf("text report missing failed last gate identity summary:\n%s", text)
	}
}

func TestTextReportIncludesRuntimeExecutionEvidenceSummary(t *testing.T) {
	text := formatTextReport(releaseStateReport{
		Status:          "fail",
		Branch:          "main",
		Version:         "v0.3.0",
		ExpectedVersion: "v0.3.0",
		RuntimeExecution: runtimeExecutionEvidenceReport{
			Required: []runtimeExecutionEvidenceCheck{
				{Target: "macos-x64", Path: "artifacts/macos-runtime-smoke.json", Status: "missing", EvidenceCommand: "./tetra smoke --target macos-x64 --run=true --report artifacts/macos-runtime-smoke.json"},
				{Target: "windows-x64", Path: "artifacts/windows-runtime-smoke.json", Status: "pass", EvidenceCommand: "./tetra smoke --target windows-x64 --run=true --report artifacts/windows-runtime-smoke.json", Host: "windows-x64"},
			},
			Missing: []string{"artifacts/macos-runtime-smoke.json"},
		},
		LastGateEvidence: gateEvidenceReport{Status: "pass", SummaryPath: "docs/generated/v1_0/release_gate_summary.json"},
	})
	if !strings.Contains(text, "runtime execution evidence: 1/2 pass, 1 missing") {
		t.Fatalf("text report missing runtime execution evidence summary:\n%s", text)
	}
	if !strings.Contains(text, "runtime execution targets: macos-x64=missing, windows-x64=pass(host=windows-x64)") {
		t.Fatalf("text report missing runtime execution target details:\n%s", text)
	}
	if !strings.Contains(text, "runtime execution commands: ./tetra smoke --target macos-x64 --run=true --report artifacts/macos-runtime-smoke.json; ./tetra smoke --target windows-x64 --run=true --report artifacts/windows-runtime-smoke.json") {
		t.Fatalf("text report missing runtime execution commands:\n%s", text)
	}
}

func TestReleaseStateRequiresGeneratedSecurityReviewEvidenceWithoutReportDir(t *testing.T) {
	report := inspectSecurityReviewEvidence(func(path string) ([]byte, error) {
		return nil, errNotExist(path)
	}, "v0.3.0", "", "abcdef0")
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	got := strings.Join(report.Missing, "\n")
	for _, want := range []string{"artifacts/security-review.md", "artifacts/security-review.md.sha256"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing evidence did not include %q: %#v", want, report)
		}
	}
}

func TestReleaseStateRejectsBlockedV030SecurityReviewPlaceholder(t *testing.T) {
	reportDir := "/tmp/tetra-v0_3_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	addRuntimeEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.3.0", expectedReleaseArtifact("v0.3.0"), expectedReleaseGateCommand("v0.3.0"), reportDir, 12, 0, "pass")
	addSecurityReviewEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	files[reportDir+"/artifacts/security-review.md"] = []byte(`# v0.3.0 Security Review CI Placeholder

Decision: blocked: missing human security signoff
CI status: missing-security-signoff
Report directory: /tmp/tetra-v0_3_0-gate
`)
	files[reportDir+"/artifacts/security-review.md.sha256"] = detachedSecurityReviewHash(files[reportDir+"/artifacts/security-review.md"])
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
		Branch:          "main",
		Version:         "v0.3.0",
		ExpectedVersion: "v0.3.0",
		ReportDir:       reportDir,
		GitHead:         "abcdef0",
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	got := strings.Join(report.Issues, "\n")
	if !strings.Contains(got, "security review evidence: security-review.md decision is not an approval for v0.3.0") {
		t.Fatalf("issues did not reject blocked security review placeholder: %v", report.Issues)
	}
}

func TestReleaseStateRejectsPlaceholderSecurityReviewArtifactHashes(t *testing.T) {
	reportDir := "/tmp/tetra-v0_3_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	addRuntimeEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	addSecurityReviewEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.3.0", expectedReleaseArtifact("v0.3.0"), expectedReleaseGateCommand("v0.3.0"), reportDir, 12, 0, "pass")
	files[reportDir+"/artifacts/security-review.md"] = []byte(strings.ReplaceAll(string(files[reportDir+"/artifacts/security-review.md"]), strings.Repeat("1", 64), strings.Repeat("0", 64)))
	files[reportDir+"/artifacts/security-review.md.sha256"] = detachedSecurityReviewHash(files[reportDir+"/artifacts/security-review.md"])
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
		Branch:          "main",
		Version:         "v0.3.0",
		ExpectedVersion: "v0.3.0",
		ReportDir:       reportDir,
		GitHead:         "abcdef0",
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	got := strings.Join(report.Issues, "\n")
	if !strings.Contains(got, "security review evidence: security-review.md contains placeholder artifact hash") {
		t.Fatalf("issues did not reject placeholder artifact hash: %v", report.Issues)
	}
}

func TestReleaseStateRejectsBuildOnlyV030RuntimeEvidence(t *testing.T) {
	reportDir := "/tmp/tetra-v0_3_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	addSecurityReviewEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.3.0", expectedReleaseArtifact("v0.3.0"), expectedReleaseGateCommand("v0.3.0"), reportDir, 12, 0, "pass")
	files[reportDir+"/artifacts/macos-runtime-smoke.json"] = runtimeSmokeJSON("macos-x64", "macos-x64", "v0.3.0", "abcdef0", true, false)
	files[reportDir+"/artifacts/windows-runtime-smoke.json"] = runtimeSmokeJSON("windows-x64", "windows-x64", "v0.3.0", "abcdef0", false, true)
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
		Branch:          "main",
		Version:         "v0.3.0",
		ExpectedVersion: "v0.3.0",
		ReportDir:       reportDir,
		GitHead:         "abcdef0",
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	got := strings.Join(report.Issues, "\n")
	for _, want := range []string{
		"macos-runtime-smoke.json build_only is true, want false",
		"macos-runtime-smoke.json case actors_pingpong did not run",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("issues did not mention rejected runtime evidence %q: %v", want, report.Issues)
		}
	}
}

func TestReleaseStateRejectsUnsupportedV030RuntimeEvidence(t *testing.T) {
	reportDir := "/tmp/tetra-v0_3_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	addRuntimeEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	addSecurityReviewEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.3.0", expectedReleaseArtifact("v0.3.0"), expectedReleaseGateCommand("v0.3.0"), reportDir, 12, 0, "pass")

	var runtime map[string]interface{}
	if err := json.Unmarshal(files[reportDir+"/artifacts/macos-runtime-smoke.json"], &runtime); err != nil {
		t.Fatalf("decode runtime smoke fixture: %v", err)
	}
	runtime["unsupported"] = true
	raw, err := json.Marshal(runtime)
	if err != nil {
		t.Fatalf("encode runtime smoke fixture: %v", err)
	}
	files[reportDir+"/artifacts/macos-runtime-smoke.json"] = raw

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
		Branch:          "main",
		Version:         "v0.3.0",
		ExpectedVersion: "v0.3.0",
		ReportDir:       reportDir,
		GitHead:         "abcdef0",
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	got := strings.Join(report.Issues, "\n")
	if !strings.Contains(got, "macos-runtime-smoke.json unsupported is true, want false") {
		t.Fatalf("issues did not reject unsupported runtime evidence: %v", report.Issues)
	}
}

func TestReleaseStateRejectsUnsupportedV030RuntimeCaseEvidence(t *testing.T) {
	reportDir := "/tmp/tetra-v0_3_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	addRuntimeEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	addSecurityReviewEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.3.0", expectedReleaseArtifact("v0.3.0"), expectedReleaseGateCommand("v0.3.0"), reportDir, 12, 0, "pass")

	var runtime map[string]interface{}
	if err := json.Unmarshal(files[reportDir+"/artifacts/macos-runtime-smoke.json"], &runtime); err != nil {
		t.Fatalf("decode runtime smoke fixture: %v", err)
	}
	cases, ok := runtime["cases"].([]interface{})
	if !ok || len(cases) == 0 {
		t.Fatalf("runtime smoke fixture cases malformed: %#v", runtime["cases"])
	}
	firstCase, ok := cases[0].(map[string]interface{})
	if !ok {
		t.Fatalf("runtime smoke fixture case malformed: %#v", cases[0])
	}
	firstCase["unsupported"] = true
	raw, err := json.Marshal(runtime)
	if err != nil {
		t.Fatalf("encode runtime smoke fixture: %v", err)
	}
	files[reportDir+"/artifacts/macos-runtime-smoke.json"] = raw

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
		Branch:          "main",
		Version:         "v0.3.0",
		ExpectedVersion: "v0.3.0",
		ReportDir:       reportDir,
		GitHead:         "abcdef0",
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	got := strings.Join(report.Issues, "\n")
	if !strings.Contains(got, "macos-runtime-smoke.json case actors_pingpong is marked unsupported") {
		t.Fatalf("issues did not reject unsupported runtime case evidence: %v", report.Issues)
	}
}

func TestReleaseStateAcceptsV030MacOSAndWindowsRuntimeEvidence(t *testing.T) {
	reportDir := "/tmp/tetra-v0_3_0-gate"
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	addRuntimeEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	addSecurityReviewEvidenceFiles(files, reportDir, "v0.3.0", "abcdef0")
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.3.0", expectedReleaseArtifact("v0.3.0"), expectedReleaseGateCommand("v0.3.0"), reportDir, 12, 0, "pass")
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
		Branch:          "main",
		Version:         "v0.3.0",
		ExpectedVersion: "v0.3.0",
		ReportDir:       reportDir,
		GitHead:         "abcdef0",
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "pass" {
		t.Fatalf("status = %q issues=%v", report.Status, report.Issues)
	}
	if len(report.RuntimeExecution.Required) != 2 {
		t.Fatalf("runtime evidence count = %d, want 2", len(report.RuntimeExecution.Required))
	}
}

func TestReleaseStateRejectsStaleV1GeneratedEvidenceMetadata(t *testing.T) {
	reportDir := "/tmp/tetra-v1_0-gate"
	gitHead := "abcdef0"
	files := releaseStateV1FreshFiles(reportDir, "v1.0.0", gitHead)
	files[reportDir+"/artifacts/host-smoke.json"] = []byte(`{
  "timestamp": "2026-04-30T12:00:00Z",
  "target": "linux-x64",
  "version": "v0.1.1",
  "git_head": "4c9a01d",
  "cases": []
}`)
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
		Branch:          "main",
		Version:         "v1.0.0",
		ExpectedVersion: "v1.0.0",
		ReportDir:       reportDir,
		GitHead:         gitHead,
		GitStatus:       nil,
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	got := strings.Join(report.Issues, "\n")
	for _, want := range []string{"host-smoke.json version is \"v0.1.1\", want \"v1.0.0\"", "host-smoke.json git_head is \"4c9a01d\", want \"abcdef0\""} {
		if !strings.Contains(got, want) {
			t.Fatalf("issues did not mention stale v1 evidence %q: %v", want, report.Issues)
		}
	}
}

func TestReleaseStateAcceptsFreshV1GeneratedEvidenceMetadata(t *testing.T) {
	reportDir := "/tmp/tetra-v1_0-gate"
	gitHead := "abcdef0"
	files := releaseStateV1FreshFiles(reportDir, "v1.0.0", gitHead)
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
		Branch:          "main",
		Version:         "v1.0.0",
		ExpectedVersion: "v1.0.0",
		ReportDir:       reportDir,
		GitHead:         gitHead,
		GitStatus:       nil,
		ReadFile:        readFile,
		StatFile:        statFile,
		Freshness:       []freshnessCheck{{Name: "docs/generated/manifest.json", Status: "pass"}},
	})
	if report.Status != "pass" {
		t.Fatalf("status = %q issues=%v", report.Status, report.Issues)
	}
}

func TestReleaseStateV1ReportDirAuditsFreshArchiveInsteadOfDocsGenerated(t *testing.T) {
	reportDir := "/tmp/tetra-v1_0-gate"
	version := "v1.0.0"
	gitHead := "abcdef0"
	files := releaseStateV1FreshReportDirFiles(reportDir, version, gitHead)
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
		Branch:          "main",
		Version:         version,
		ExpectedVersion: version,
		ReportDir:       reportDir,
		GitHead:         gitHead,
		GitStatus:       nil,
		ReadFile:        readFile,
		StatFile:        statFile,
		Freshness:       []freshnessCheck{{Name: "docs/generated/manifest.json", Status: "pass"}},
	})
	if report.Status != "pass" {
		t.Fatalf("status = %q issues=%v", report.Status, report.Issues)
	}
	for _, check := range report.GeneratedArtifacts.Required {
		if strings.HasPrefix(check.Path, "docs/generated/v1_0/") {
			t.Fatalf("v1 report-dir audit should not require checked-in generated artifact %s", check.Path)
		}
	}
}

func TestReleaseStateRejectsMissingV1BackendSummary(t *testing.T) {
	reportDir := "/tmp/tetra-v1_0-gate"
	version := "v1.0.0"
	gitHead := "abcdef0"
	files := releaseStateV1FreshFiles(reportDir, version, gitHead)
	delete(files, reportDir+"/artifacts/backend-summary.md")
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
		Branch:          "main",
		Version:         version,
		ExpectedVersion: version,
		ReportDir:       reportDir,
		GitHead:         gitHead,
		GitStatus:       nil,
		ReadFile:        readFile,
		StatFile:        statFile,
		Freshness:       []freshnessCheck{{Name: "docs/generated/manifest.json", Status: "pass"}},
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	got := strings.Join(report.Issues, "\n")
	if !strings.Contains(got, "missing required release artifact: "+reportDir+"/artifacts/backend-summary.md") {
		t.Fatalf("issues did not require backend summary artifact: %v", report.Issues)
	}
}

func TestReleaseStateRejectsNonCanonicalV1GateCommand(t *testing.T) {
	reportDir := "/tmp/tetra-v1_0-gate"
	gitHead := "abcdef0"
	files := releaseStateV1FreshFiles(reportDir, "v1.0.0", gitHead)
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v1.0.0", expectedReleaseArtifact("v1.0.0"), "env bash scripts/release/v1_0/gate.sh --reuse-old-report", reportDir, 32, 0, "pass")
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
		Branch:          "main",
		Version:         "v1.0.0",
		ExpectedVersion: "v1.0.0",
		ReportDir:       reportDir,
		GitHead:         gitHead,
		GitStatus:       nil,
		ReadFile:        readFile,
		StatFile:        statFile,
	})
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	got := strings.Join(report.Issues, "\n")
	if !strings.Contains(got, `release_gate_command is "env bash scripts/release/v1_0/gate.sh --reuse-old-report", want "bash scripts/release/v1_0/gate.sh"`) {
		t.Fatalf("issues did not mention non-canonical gate command: %v", report.Issues)
	}
}

func TestValidationCommandFreshnessRejectsFailedSmokeEvidence(t *testing.T) {
	check := checkValidationCommands(".", "generated smoke evidence", []validationCommand{
		{Name: "go", Args: []string{"run", "./tools/cmd/validate-smoke-list", "--report", "docs/generated/v1_0/smoke-list.json"}},
	}, func(dir string, name string, args ...string) (string, error) {
		return "missing core_crypto_smoke", errors.New("exit status 1")
	})
	if check.Status != "fail" {
		t.Fatalf("status = %q, want fail", check.Status)
	}
	if !strings.Contains(check.Detail, "missing core_crypto_smoke") {
		t.Fatalf("detail did not include validator output: %#v", check)
	}
}

func TestValidationCommandFreshnessPassesWhenValidatorsPass(t *testing.T) {
	check := checkValidationCommands(".", "generated smoke evidence", []validationCommand{
		{Name: "go", Args: []string{"run", "./tools/cmd/validate-smoke-list", "--report", "docs/generated/v1_0/smoke-list.json"}},
		{Name: "go", Args: []string{"run", "./tools/cmd/smoke-report-to-checklist", "--validate-only", "--report", "docs/generated/v1_0/host-smoke.json"}},
	}, func(dir string, name string, args ...string) (string, error) {
		return "", nil
	})
	if check.Status != "pass" || check.Detail != "" {
		t.Fatalf("check = %#v", check)
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
		Branch:          "main",
		Version:         "v0.2.0",
		ExpectedVersion: "v0.2.0",
		GitStatus:       nil,
		ReadFile:        readFile,
		StatFile:        statFile,
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
		"started_at":           "2026-04-30T12:00:00Z",
		"ended_at":             "2026-04-30T12:01:00Z",
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
	files["docs/generated/v1_0/release_gate_summary.json"] = releaseSummaryJSON("v0.1.3", "tetra.release.v0_1_3.gate-report.v1", "bash scripts/release/v0_1_3/gate.sh", "/tmp/stale-v0_1_3-gate", 33, 0, "pass")
	files[reportDir+"/summary.json"] = releaseSummaryJSON("v0.2.0", expectedReleaseArtifact("v0.2.0"), "bash scripts/release/v0_2_0/gate.sh", reportDir, 33, 0, "pass")
	return files
}

func releaseStateV1FreshFiles(reportDir string, version string, gitHead string) map[string][]byte {
	files := map[string][]byte{}
	for _, path := range requiredReleaseArtifacts {
		files[path] = []byte("{}")
	}
	for path, raw := range releaseStateV1FreshReportDirFiles(reportDir, version, gitHead) {
		files[path] = raw
	}
	for _, path := range []string{
		"docs/generated/v1_0/host-smoke.json",
		"docs/generated/v1_0/linux-smoke.json",
		"docs/generated/v1_0/macos-smoke.json",
		"docs/generated/v1_0/windows-smoke.json",
		"docs/generated/v1_0/wasm32-wasi-artifact-smoke.json",
		"docs/generated/v1_0/wasm32-web-artifact-smoke.json",
		"docs/generated/v1_0/wasi-smoke.artifact.json",
		"docs/generated/v1_0/wasi-smoke.json",
		"docs/generated/v1_0/test-all/host-smoke.json",
	} {
		files[path] = []byte(`{"timestamp":"2026-04-30T12:00:00Z","version":"` + version + `","git_head":"` + gitHead + `","cases":[]}`)
	}
	files["docs/generated/manifest.json"] = []byte(`{"compiler_version":"` + version + `"}`)
	files["docs/generated/v1_0/manifest.json"] = []byte(`{"compiler_version":"` + version + `"}`)
	files["docs/generated/v1_0/binary-size-thresholds.json"] = []byte(`{"schema":"tetra.binary-size-thresholds.v1alpha1","compiler_version":"` + version + `"}`)
	files["docs/generated/v1_0/reproducible-build.json"] = []byte(`{"schema":"tetra.reproducible-build-proof.v1alpha1","compiler_version":"` + version + `"}`)
	files["docs/generated/v1_0/performance-regression.json"] = []byte(`{"schema":"tetra.performance-regression.v1","git_head":"` + gitHead + `"}`)
	files["docs/generated/v1_0/api-diff/api-diff.json"] = []byte(`{"schema":"tetra.api.diff.v1alpha1"}`)
	files["docs/generated/v1_0/web-ui-smoke.json"] = []byte(`{"schema":"tetra.web-ui-smoke.v1alpha1","generated_at":"2026-04-30T12:00:00Z"}`)
	files["docs/generated/v1_0/release-state.json"] = []byte(`{"schema":"tetra.release-state.v1alpha1","version":"` + version + `","expected_version":"` + version + `"}`)
	files["docs/generated/v1_0/test_all_full_summary.json"] = []byte(`{"started_at":"2026-04-30T12:00:00Z","ended_at":"2026-04-30T12:01:00Z"}`)
	files["docs/generated/v1_0/test-all/summary.json"] = []byte(`{"started_at":"2026-04-30T12:00:00Z","ended_at":"2026-04-30T12:01:00Z"}`)
	files["docs/generated/v1_0/release_gate_summary.json"] = releaseSummaryJSON(version, expectedReleaseArtifact(version), expectedReleaseGateCommand(version), reportDir, 32, 0, "pass")
	files[reportDir+"/summary.json"] = releaseSummaryJSON(version, expectedReleaseArtifact(version), expectedReleaseGateCommand(version), reportDir, 32, 0, "pass")
	addSecurityReviewEvidenceFiles(files, reportDir, version, gitHead)
	return files
}

func releaseStateV1FreshReportDirFiles(reportDir string, version string, gitHead string) map[string][]byte {
	files := map[string][]byte{}
	for _, path := range v1ReportDirRequiredArtifacts(reportDir) {
		files[path] = []byte("{}")
	}
	for _, path := range []string{
		reportDir + "/artifacts/host-smoke.json",
		reportDir + "/artifacts/linux-smoke.json",
		reportDir + "/artifacts/macos-smoke.json",
		reportDir + "/artifacts/windows-smoke.json",
		reportDir + "/artifacts/wasm32-wasi-artifact-smoke.json",
		reportDir + "/artifacts/wasm32-web-artifact-smoke.json",
		reportDir + "/artifacts/wasi-smoke.artifact.json",
		reportDir + "/artifacts/wasi-smoke.json",
		reportDir + "/artifacts/test-all/host-smoke.json",
	} {
		files[path] = []byte(`{"timestamp":"2026-04-30T12:00:00Z","version":"` + version + `","git_head":"` + gitHead + `","cases":[]}`)
	}
	files[reportDir+"/artifacts/manifest.json"] = []byte(`{"compiler_version":"` + version + `"}`)
	files[reportDir+"/artifacts/binary-size-thresholds.json"] = []byte(`{"schema":"tetra.binary-size-thresholds.v1alpha1","compiler_version":"` + version + `"}`)
	files[reportDir+"/artifacts/reproducible-build.json"] = []byte(`{"schema":"tetra.reproducible-build-proof.v1alpha1","compiler_version":"` + version + `"}`)
	files[reportDir+"/artifacts/performance-regression.json"] = []byte(`{"schema":"tetra.performance-regression.v1","git_head":"` + gitHead + `"}`)
	files[reportDir+"/artifacts/api-diff/api-diff.json"] = []byte(`{"schema":"tetra.api.diff.v1alpha1"}`)
	files[reportDir+"/artifacts/web-ui-smoke.json"] = []byte(`{"schema":"tetra.web-ui-smoke.v1alpha1","generated_at":"2026-04-30T12:00:00Z"}`)
	files[reportDir+"/summary.json"] = releaseSummaryJSON(version, expectedReleaseArtifact(version), expectedReleaseGateCommand(version), reportDir, 33, 0, "pass")
	files[reportDir+"/artifacts/test-all/summary.json"] = []byte(`{"started_at":"2026-04-30T12:00:00Z","ended_at":"2026-04-30T12:01:00Z"}`)
	addSecurityReviewEvidenceFiles(files, reportDir, version, gitHead)
	return files
}

func addRuntimeEvidenceFiles(files map[string][]byte, reportDir string, version string, gitHead string) {
	files[reportDir+"/artifacts/macos-runtime-smoke.json"] = runtimeSmokeJSON("macos-x64", "macos-x64", version, gitHead, false, true)
	files[reportDir+"/artifacts/windows-runtime-smoke.json"] = runtimeSmokeJSON("windows-x64", "windows-x64", version, gitHead, false, true)
}

func addSecurityReviewEvidenceFiles(files map[string][]byte, reportDir string, version string, gitHead string) {
	review := []byte(fmt.Sprintf("# %s Security Review Signoff\n\n"+
		"Reviewer: Release Reviewer <security@example.invalid>\n"+
		"Reviewed commit: %s\n"+
		"Report directory: %s\n"+
		"Decision: approved for %s release\n\n"+
		"## Evidence Commands\n\n"+
		"- `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`: pass, 2026-04-30, logs/security-docs.log\n\n"+
		"## Artifact Hashes\n\n"+
		"- summary.json: sha256:1111111111111111111111111111111111111111111111111111111111111111\n\n"+
		"## Residual Risks\n\n"+
		"- None\n", version, gitHead, reportDir, version))
	files[reportDir+"/artifacts/security-review.md"] = review
	files[reportDir+"/artifacts/security-review.md.sha256"] = detachedSecurityReviewHash(review)
}

func detachedSecurityReviewHash(review []byte) []byte {
	sum := sha256.Sum256(review)
	return []byte(fmt.Sprintf("%x  artifacts/security-review.md\n", sum))
}

func runtimeSmokeJSON(target string, host string, version string, gitHead string, buildOnly bool, ran bool) []byte {
	cases := []map[string]interface{}{}
	for _, tc := range []struct {
		name string
		src  string
		exit int
	}{
		{name: "actors_pingpong", src: "examples/actors_pingpong.tetra", exit: 0},
		{name: "actor_sleep_pingpong", src: "examples/actor_sleep_pingpong.tetra", exit: 0},
		{name: "task_smoke", src: "examples/task_smoke.tetra", exit: 42},
		{name: "time_sleep_smoke", src: "examples/time_sleep_smoke.tetra", exit: 0},
		{name: "task_sleep_deadline_smoke", src: "examples/task_sleep_deadline_smoke.tetra", exit: 0},
		{name: "task_join_wait_smoke", src: "examples/task_join_wait_smoke.tetra", exit: 5},
		{name: "deadline_aware_waits_smoke", src: "examples/deadline_aware_waits_smoke.tetra", exit: 0},
		{name: "wait_composition_smoke", src: "examples/wait_composition_smoke.tetra", exit: 0},
	} {
		cases = append(cases, map[string]interface{}{
			"name":          tc.name,
			"src_path":      tc.src,
			"out_path":      "/tmp/" + tc.name,
			"expected_exit": tc.exit,
			"actual_exit":   tc.exit,
			"ran":           ran,
			"pass":          true,
		})
	}
	raw, err := json.Marshal(map[string]interface{}{
		"timestamp":     "2026-04-30T12:00:00Z",
		"target":        target,
		"build_only":    buildOnly,
		"host":          host,
		"version":       version,
		"git_head":      gitHead,
		"islands_debug": false,
		"total":         len(cases),
		"passed":        len(cases),
		"failed":        0,
		"cases":         cases,
	})
	if err != nil {
		panic(err)
	}
	return raw
}
