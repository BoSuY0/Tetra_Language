package main

import (
	"strings"
	"testing"
)

func TestValidateCompletionAuditAcceptsCurrentBlockedAudit(t *testing.T) {
	audit := validBlockedCompletionAudit()
	err := validateCompletionAudit([]byte(audit), completionAuditOptions{
		ExpectedStatus: "not-achieved",
	})
	if err != nil {
		t.Fatalf("validateCompletionAudit failed: %v\n%s", err, audit)
	}
}

func TestValidateCompletionAuditRejectsMissingRequiredRequirement(t *testing.T) {
	audit := strings.Replace(validBlockedCompletionAudit(), "| Linux-x64 production scope is selected |", "| Full production scope is selected |", 1)
	err := validateCompletionAudit([]byte(audit), completionAuditOptions{
		ExpectedStatus: "not-achieved",
	})
	if err == nil {
		t.Fatalf("expected missing required requirement failure")
	}
	if !strings.Contains(err.Error(), "missing required checklist requirement") {
		t.Fatalf("error = %v, want missing required checklist requirement", err)
	}
}

func TestValidateCompletionAuditRequiresMemoryParallelCompilerProductionRows(t *testing.T) {
	audit := strings.Replace(validBlockedCompletionAudit(), "| Memory production core is production |", "| Memory production core is documented |", 1)
	err := validateCompletionAudit([]byte(audit), completionAuditOptions{
		ExpectedStatus: "not-achieved",
	})
	if err == nil {
		t.Fatalf("expected missing memory production requirement failure")
	}
	for _, want := range []string{"missing required checklist requirement", "Memory production core is production"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q", err, want)
		}
	}
}

func TestValidateCompletionAuditRejectsAchievedStatusWithFailingRows(t *testing.T) {
	audit := strings.Replace(validBlockedCompletionAudit(), "Status: not achieved.", "Status: achieved.", 1)
	err := validateCompletionAudit([]byte(audit), completionAuditOptions{
		ExpectedStatus: "achieved",
	})
	if err == nil {
		t.Fatalf("expected achieved audit with failing rows to fail")
	}
	if !strings.Contains(err.Error(), "achieved audit has non-passing checklist row") {
		t.Fatalf("error = %v, want non-passing checklist row", err)
	}
}

func TestValidateCompletionAuditRejectsBlockedAuditWithoutMissingWorkSummary(t *testing.T) {
	audit := strings.Replace(validBlockedCompletionAudit(), "## Missing Work Summary\n\nThe objective is not achieved.\n", "", 1)
	err := validateCompletionAudit([]byte(audit), completionAuditOptions{
		ExpectedStatus: "not-achieved",
	})
	if err == nil {
		t.Fatalf("expected missing summary failure")
	}
	if !strings.Contains(err.Error(), "missing work summary is required") {
		t.Fatalf("error = %v, want missing work summary failure", err)
	}
}

func TestValidateCompletionAuditRequiresReleaseEvidenceMatrix(t *testing.T) {
	audit := strings.Replace(validBlockedCompletionAudit(), releaseEvidenceMatrixFixture(), "", 1)
	err := validateCompletionAudit([]byte(audit), completionAuditOptions{
		ExpectedStatus: "not-achieved",
	})
	if err == nil {
		t.Fatalf("expected missing release evidence matrix failure")
	}
	if !strings.Contains(err.Error(), "missing \"Release Evidence Matrix\" section") {
		t.Fatalf("error = %v, want missing release evidence matrix failure", err)
	}
}

func TestValidateCompletionAuditRejectsMatrixRowWithoutNegativeTests(t *testing.T) {
	audit := strings.Replace(validBlockedCompletionAudit(), "positive: go test ./compiler/...; negative: validator fixtures reject stale evidence", "positive: go test ./compiler/...", 1)
	err := validateCompletionAudit([]byte(audit), completionAuditOptions{
		ExpectedStatus: "not-achieved",
	})
	if err == nil {
		t.Fatalf("expected negative-test evidence failure")
	}
	if !strings.Contains(err.Error(), "tests must include negative:") {
		t.Fatalf("error = %v, want negative-test evidence failure", err)
	}
}

func TestValidateCompletionAuditRejectsPassMatrixRowWithBlockerEvidence(t *testing.T) {
	audit := strings.Replace(validBlockedCompletionAudit(), "ci: bash scripts/ci/test.sh | pass", "ci: blocked by dirty worktree | pass", 1)
	err := validateCompletionAudit([]byte(audit), completionAuditOptions{
		ExpectedStatus: "not-achieved",
	})
	if err == nil {
		t.Fatalf("expected dirty-green evidence failure")
	}
	if !strings.Contains(err.Error(), "pass status contains blocker evidence") {
		t.Fatalf("error = %v, want dirty-green evidence failure", err)
	}
}

func validBlockedCompletionAudit() string {
	return `# Tetra v0.4.0 Completion Audit

Status: not achieved.

## Prompt-To-Artifact Checklist

| Requirement | Required artifact or command | Current evidence | Result |
| --- | --- | --- | --- |
| Version is marked ` + "`v0.4.0`" + ` | ` + "`./tetra version`" + ` | Version metadata prints ` + "`v0.4.0`" + `. | pass for version metadata only |
| Manifest is marked ` + "`v0.4.0`" + ` | ` + "`docs/generated/manifest.json`" + ` | Manifest identity matches. | pass for manifest identity only |
| Linux-x64 production scope is selected | scope decisions | Scope status is ` + "`linux-x64-production-scope-selected`" + `. | pass for scope selection |
| Feature registry has no required non-production gap | ` + "`./tetra features --format=json`" + ` | Selected features are current; excluded gaps remain excluded. | pass for scoped release |
| Callable model is production | callable features and tests | Callable Level 1, Level 2, and selected first-class callables are current. | pass |
| Lifetime SSA is production for the selected surface | lifetime docs/tests | Local SSA-like solver covers selected surface. | pass |
| Memory production core is production | memory production smoke report | Memory production evidence is required. | pending final evidence |
| Parallel production core is production | parallel production smoke report | Parallel production evidence is required. | pending final evidence |
| Compiler production core is production | compiler production smoke report | Compiler production evidence is required. | pending final evidence |
| Standard library mirror policy is production | stdlib evidence | Compatibility mirror policy is current. | pass |
| UI metadata/runtime/native behavior is production | UI smoke evidence | Linux-x64 native runtime report exists. | pass |
| Distributed actors are production | actor runtime evidence | Linux-x64 distributed actor runtime report exists. | pass |
| Linux runtime is production | Linux host smoke report | Linux smoke passes. | pass |
| WASM runtime execution is production | wasm scope decision | Excluded from the scoped release. | not required |
| Distributed EcoNet is production | Eco network scope decision | Excluded from the scoped release. | not required |
| Windows runtime is production | Windows scope decision | Excluded from the scoped release. | not required |
| macOS runtime is production | macOS scope decision | Excluded from the scoped release. | not required |
| ` + "`v0.4.0`" + ` readiness preflight passes | readiness validator | Readiness passes. | pass |
| ` + "`v0.4.0`" + ` release gate exists | release gate summary | Final gate still needs collection. | pending final evidence |
| ` + "`v0.4.0`" + ` security review exists | security review signoff | No approved signoff exists. | pending final evidence |
| Generated docs verification covers the objective | verify docs | Needs rerun after scope edits. | pending rerun |
| Baseline tests pass | package tests | Needs rerun after scope edits. | pending rerun |
| Worktree is clean for release | ` + "`git status --porcelain --untracked-files=all`" + ` | Worktree is dirty. | blocked for tag-ready release |

` + releaseEvidenceMatrixFixture() + `

## Missing Work Summary

The objective is not achieved.
`
}

func releaseEvidenceMatrixFixture() string {
	return `## Release Evidence Matrix

| Requirement | File(s) | Tests | Docs | Evidence | Status |
| --- | --- | --- | --- | --- | --- |
| Compiler production core is production | implementation: compiler/compiler.go; compiler/internal/lower/lower.go | positive: go test ./compiler/...; negative: validator fixtures reject stale evidence | docs: docs/spec/current_supported_surface.md; manifest: docs/generated/manifest.json | report: compiler-production-linux-x64.json; graphify: graphify update .; ci: bash scripts/ci/test.sh | pass |
| Memory production core is production | implementation: compiler/internal/runtimeabi/small_heap.go | positive: go test ./compiler/internal/runtimeabi; negative: allocation validator rejects forged storage | docs: docs/spec/runtime_abi.md; manifest: docs/generated/manifest.json | report: memory-production-linux-x64.json; graphify: graphify update .; ci: bash scripts/ci/test.sh | pending final evidence |
`
}
