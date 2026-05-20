package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBuildReleaseStatePassesForCleanScopedArtifacts(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)
	initGit(t, root)
	writeFile(t, "compiler/internal/version/version.go", `package version

const CompilerVersion = "v0.4.0"
`)
	for _, path := range requiredRepoArtifacts() {
		writeFile(t, path, "{}\n")
	}
	reportDir := filepath.Join(root, "report")
	for _, path := range requiredReportArtifacts(reportDir) {
		writeFile(t, path, "{}\n")
	}
	gitAddCommit(t, root)

	report := buildReleaseState("v0.4.0", reportDir)
	if report.Status != "pass" {
		t.Fatalf("status = %s, issues = %#v", report.Status, report.Issues)
	}
	if !report.Git.Clean {
		t.Fatalf("expected clean git state: %#v", report.Git)
	}
}

func TestBuildReleaseStateFailsForDirtyWorktree(t *testing.T) {
	root := t.TempDir()
	chdir(t, root)
	initGit(t, root)
	writeFile(t, "compiler/internal/version/version.go", `package version

const CompilerVersion = "v0.4.0"
`)
	for _, path := range requiredRepoArtifacts() {
		writeFile(t, path, "{}\n")
	}
	gitAddCommit(t, root)
	writeFile(t, "dirty.txt", "dirty\n")

	report := buildReleaseState("v0.4.0", "")
	if report.Status != "fail" {
		t.Fatalf("expected fail for dirty worktree")
	}
	if report.Git.Clean || len(report.Git.Entries) == 0 {
		t.Fatalf("expected dirty git entries: %#v", report.Git)
	}
}

func TestRequiredReportArtifactsIncludeMemoryParallelCompilerProductionEvidence(t *testing.T) {
	reportDir := filepath.Join("report-root")
	for _, want := range []string{
		filepath.Join(reportDir, "artifacts", "memory-production-linux-x64.json"),
		filepath.Join(reportDir, "artifacts", "parallel-production-linux-x64.json"),
		filepath.Join(reportDir, "artifacts", "compiler-production-linux-x64.json"),
	} {
		found := false
		for _, got := range requiredReportArtifacts(reportDir) {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("required report artifacts missing %s", want)
		}
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	})
}

func initGit(t *testing.T, root string) {
	t.Helper()
	run(t, root, "git", "init", "-q")
	run(t, root, "git", "config", "user.name", "Release State Test")
	run(t, root, "git", "config", "user.email", "release-state-test@example.invalid")
}

func gitAddCommit(t *testing.T, root string) {
	t.Helper()
	run(t, root, "git", "add", "-A")
	run(t, root, "git", "commit", "-q", "-m", "fixture")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}
