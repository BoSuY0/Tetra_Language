package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const memory100TestHead = "0123456789abcdef0123456789abcdef01234567"

func TestValidateMemory100ReportDirAcceptsCompleteScopedFixture(t *testing.T) {
	reportDir := writeMemory100Fixture(t, nil)
	if err := validateMemory100ReportDir(reportDir, memory100TestHead); err != nil {
		t.Fatalf("validateMemory100ReportDir failed: %v", err)
	}
}

func TestValidateMemory100ReportDirRejectsStaleAggregateGitHead(t *testing.T) {
	reportDir := writeMemory100Fixture(t, nil)
	currentHead := "abcdefabcdefabcdefabcdefabcdefabcdefabcd"
	err := validateMemory100ReportDir(reportDir, currentHead)
	if err == nil {
		t.Fatalf("expected stale aggregate git head to fail")
	}
	if got := err.Error(); !strings.Contains(got, "Memory100 manifest git_head") || !strings.Contains(got, "current git head") {
		t.Fatalf("error = %v, want aggregate current git head rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsReportDirSymlink(t *testing.T) {
	reportDir := writeMemory100Fixture(t, nil)
	link := filepath.Join(t.TempDir(), "memory100-link")
	if err := os.Symlink(reportDir, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	err := validateMemory100ReportDir(link, memory100TestHead)
	if err == nil {
		t.Fatalf("expected report dir symlink to fail")
	}
	if got := err.Error(); !strings.Contains(got, "report dir must not be a symlink") {
		t.Fatalf("error = %v, want report dir symlink rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMissingRAMBundle(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		if err := os.RemoveAll(filepath.Join(root, "ram-contract")); err != nil {
			t.Fatalf("remove ram-contract bundle: %v", err)
		}
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected missing RAM bundle to fail")
	}
	if got := err.Error(); !strings.Contains(got, "ram-contract") && !strings.Contains(got, "ram_contract_report") {
		t.Fatalf("error = %v, want RAM bundle rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsForbiddenPublicClaim(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		manifest["claims"] = []any{"Memory is 100% ready"}
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected forbidden public claim to fail")
	}
	if !strings.Contains(err.Error(), "forbidden") || !strings.Contains(err.Error(), "Memory is 100% ready") {
		t.Fatalf("error = %v, want forbidden Memory100 claim rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMissingArtifactGitHead(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "memory-production", "memory-production-linux-x64.json"), map[string]any{
			"schema": "tetra.memory.production.v1",
			"status": "pass",
		})
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected missing artifact git_head to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory_production_report") || !strings.Contains(got, "git_head") {
		t.Fatalf("error = %v, want memory production git_head rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsStaleRequiredArtifactGitHead(t *testing.T) {
	staleHead := "abcdefabcdefabcdefabcdefabcdefabcdefabcd"
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "raw-memory-contract", "raw-memory-contract.json"), memory100ArtifactJSON("raw_memory_contract_report", "tetra.raw-memory-contract.v1", staleHead))
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected stale required artifact git head to fail")
	}
	if got := err.Error(); !strings.Contains(got, "raw_memory_contract_report") || !strings.Contains(got, "does not match Memory100 git_head") {
		t.Fatalf("error = %v, want required artifact git_head mismatch rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsEmptyRequiredArtifact(t *testing.T) {
	reportDir := writeMemory100Fixture(t, nil)
	path := filepath.Join(reportDir, "raw-memory-contract", "raw-memory-contract.json")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected empty required artifact to fail")
	}
	if got := err.Error(); !strings.Contains(got, "raw_memory_contract_report") || !strings.Contains(got, "is empty") {
		t.Fatalf("error = %v, want empty required artifact rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeMemoryProductionEnvelope(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "memory-production", "memory-production-linux-x64.json"), map[string]any{
			"schema":   "tetra.memory.production.v1",
			"status":   "pass",
			"git_head": memory100TestHead,
		})
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake memory production envelope to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory production") || !strings.Contains(got, "report") {
		t.Fatalf("error = %v, want memory production report content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeMemoryReleaseManifestEnvelope(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "memory-production", "memory-release-manifest.json"), map[string]any{
			"schema":   "tetra.memory.release-manifest.v1",
			"git_head": memory100TestHead,
		})
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake memory release manifest envelope to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory release manifest") || !strings.Contains(got, "command") {
		t.Fatalf("error = %v, want memory release manifest command/content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsStaleMemoryReleaseManifestGeneratedAt(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "memory-production")
		path := filepath.Join(memoryDir, "memory-release-manifest.json")
		var releaseManifest map[string]any
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &releaseManifest); err != nil {
			t.Fatal(err)
		}
		releaseManifest["generated_at"] = "2026-06-10T10:59:59Z"
		writeMemory100JSON(t, path, releaseManifest)
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected stale Memory release manifest generated_at to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory release manifest generated_at") || !strings.Contains(got, "older than child evidence") {
		t.Fatalf("error = %v, want stale Memory release manifest generated_at rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsCopiedMemoryReleaseManifestCommandProvenance(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "memory-production")
		copiedDir := "reports/memory-100/old-copy/memory-production"
		releaseManifest := memory100ArtifactJSON("memory_release_manifest", "tetra.memory.release-manifest.v1", memory100TestHead)
		releaseManifest["commands"] = memory100MemoryReleaseCommandsForDir(copiedDir, memory100TestHead)
		releaseManifest["artifacts"] = memory100MemoryReleaseArtifactRefsForDir(copiedDir, memory100TestHead)
		writeMemory100JSON(t, filepath.Join(memoryDir, "memory-release-manifest.json"), releaseManifest)
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected copied memory release manifest command provenance to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory release manifest command") || !strings.Contains(got, "current memory production report dir") {
		t.Fatalf("error = %v, want copied memory release manifest command provenance rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsUnexpectedAggregateManifestArtifactRef(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		artifacts := append([]any{}, manifest["artifacts"].([]any)...)
		artifacts = append(artifacts, map[string]any{
			"path":   "docs-only/fake-evidence.json",
			"kind":   "docs_only_fake_evidence",
			"schema": "tetra.fake-evidence.v1",
		})
		manifest["artifacts"] = artifacts
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected unexpected aggregate manifest artifact ref to fail")
	}
	if got := err.Error(); !strings.Contains(got, "unexpected Memory100 artifact kind docs_only_fake_evidence") {
		t.Fatalf("error = %v, want unexpected aggregate artifact ref rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsUnexpectedMemoryReleaseArtifactRef(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "memory-production")
		path := filepath.Join(memoryDir, "memory-release-manifest.json")
		var releaseManifest map[string]any
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &releaseManifest); err != nil {
			t.Fatal(err)
		}
		artifacts := append([]any{}, releaseManifest["artifacts"].([]any)...)
		artifacts = append(artifacts, map[string]any{
			"path":    "docs-only/fake-evidence.json",
			"kind":    "docs_only_fake_evidence",
			"schema":  "tetra.fake-evidence.v1",
			"target":  "linux-x64",
			"command": "echo docs-only fake evidence",
		})
		releaseManifest["artifacts"] = artifacts
		writeMemory100JSON(t, path, releaseManifest)
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected unexpected memory release artifact ref to fail")
	}
	if got := err.Error(); !strings.Contains(got, "unexpected memory release manifest artifact kind docs_only_fake_evidence") {
		t.Fatalf("error = %v, want unexpected memory release artifact ref rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMissingMemoryReleaseDeclaredArtifactHashCoverage(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "memory-production")
		if err := os.Remove(filepath.Join(memoryDir, "targets.json")); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected missing Memory release declared artifact hash coverage to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory production artifact-hashes.json") || !strings.Contains(got, "targets.json") {
		t.Fatalf("error = %v, want missing memory release declared artifact hash coverage rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeMemoryReleaseTargetsReport(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "memory-production")
		writeMemory100JSON(t, filepath.Join(memoryDir, "targets.json"), map[string]any{
			"supported":  []any{"linux-x64"},
			"build_only": []any{},
			"planned":    []any{},
			"targets":    []any{},
		})
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake Memory release targets report to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory release targets.json") || !strings.Contains(got, "supported") {
		t.Fatalf("error = %v, want Memory release targets content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeMemoryReleaseIslandProofVerifier(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "memory-production")
		writeMemory100JSON(t, filepath.Join(memoryDir, "island-proof-verifier.json"), map[string]any{
			"schema":   "tetra.island.proof.v1",
			"git_head": memory100TestHead,
		})
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake Memory release island proof verifier to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory release island proof verifier") {
		t.Fatalf("error = %v, want Memory release island proof content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeMemoryReleaseFuzzTier1Summary(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "memory-production")
		fuzzDir := filepath.Join(memoryDir, "memory-fuzz-tier1")
		writeMemory100JSON(t, filepath.Join(fuzzDir, "island-proof-fuzz-summary.json"), map[string]any{
			"schema_version": "tetra.island-proof-fuzz-summary.v1",
			"status":         "pass",
		})
		writeMemory100HashManifest(t, fuzzDir)
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake Memory release fuzz Tier 1 summary to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory release memory fuzz") || !strings.Contains(got, "island-proof") {
		t.Fatalf("error = %v, want Memory release fuzz content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeMemoryReleaseNestedRAMContractManifest(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "memory-production")
		ramDir := filepath.Join(memoryDir, "ram-contract")
		writeMemory100JSON(t, filepath.Join(ramDir, "ram-contract-release-manifest.json"), map[string]any{
			"schema":   "tetra.ram-contract.release-manifest.v1",
			"git_head": memory100TestHead,
		})
		writeMemory100HashManifest(t, ramDir)
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake Memory release nested RAM contract manifest to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory release RAM contract release manifest") {
		t.Fatalf("error = %v, want Memory release nested RAM contract manifest content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeMemoryReleaseNestedRAMContractReport(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "memory-production")
		ramDir := filepath.Join(memoryDir, "ram-contract")
		writeMemory100JSON(t, filepath.Join(ramDir, "ram-contract-report.json"), map[string]any{
			"schema_version": "tetra.ram-contract-report.v1",
			"git_head":       memory100TestHead,
			"target":         "linux-x64",
		})
		writeMemory100HashManifest(t, ramDir)
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake Memory release nested RAM contract report to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory release ram-contract-report.json") {
		t.Fatalf("error = %v, want Memory release nested RAM report content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeMemoryReleaseNestedRAMContractFuzzOracle(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "memory-production")
		ramDir := filepath.Join(memoryDir, "ram-contract")
		writeMemory100JSON(t, filepath.Join(ramDir, "fuzz", "ram-contract-fuzz-oracle.json"), map[string]any{
			"schema_version": "tetra.ram-contract-fuzz-oracle.v1",
			"git_head":       memory100TestHead,
			"generated_at":   "2026-06-10T11:00:00Z",
			"summary":        map[string]any{"mutations": 0, "rejected": 0},
			"observations":   []any{},
			"non_claims":     []any{"not Memory 100%"},
		})
		writeMemory100HashManifest(t, ramDir)
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake Memory release nested RAM contract fuzz oracle to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory release RAM contract fuzz") || !strings.Contains(got, "mutation") {
		t.Fatalf("error = %v, want Memory release nested RAM contract fuzz rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeRAMContractReleaseManifestEnvelope(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "ram-contract", "ram-contract-release-manifest.json"), map[string]any{
			"schema":   "tetra.ram-contract.release-manifest.v1",
			"git_head": memory100TestHead,
		})
		writeMemory100HashManifest(t, filepath.Join(root, "ram-contract"))
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake RAM contract release manifest envelope to fail")
	}
	if got := err.Error(); !strings.Contains(got, "RAM contract release manifest") || !strings.Contains(got, "command") {
		t.Fatalf("error = %v, want RAM contract release manifest command/content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeMemoryProductionHashManifest(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "memory-production", "artifact-hashes.json"), map[string]any{
			"schema":    memory100HashSchema,
			"root":      ".",
			"artifacts": []any{},
		})
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake memory production hash manifest to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory production artifact-hashes.json") || !strings.Contains(got, "artifacts") {
		t.Fatalf("error = %v, want memory production hash manifest content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeRAMContractHashManifest(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "ram-contract", "artifact-hashes.json"), map[string]any{
			"schema":    memory100HashSchema,
			"root":      ".",
			"artifacts": []any{},
		})
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake RAM contract hash manifest to fail")
	}
	if got := err.Error(); !strings.Contains(got, "ram contract artifact-hashes.json") || !strings.Contains(got, "artifacts") {
		t.Fatalf("error = %v, want RAM contract hash manifest content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsUnexpectedRAMReleaseArtifactRef(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		ramDir := filepath.Join(root, "ram-contract")
		path := filepath.Join(ramDir, "ram-contract-release-manifest.json")
		var releaseManifest map[string]any
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &releaseManifest); err != nil {
			t.Fatal(err)
		}
		artifacts := append([]any{}, releaseManifest["artifacts"].([]any)...)
		artifacts = append(artifacts, map[string]any{
			"path":   "docs-only/fake-evidence.json",
			"kind":   "docs_only_fake_evidence",
			"schema": "tetra.fake-evidence.v1",
		})
		releaseManifest["artifacts"] = artifacts
		writeMemory100JSON(t, path, releaseManifest)
		writeMemory100HashManifest(t, ramDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected unexpected RAM release artifact ref to fail")
	}
	if got := err.Error(); !strings.Contains(got, "unexpected RAM contract release manifest artifact kind docs_only_fake_evidence") {
		t.Fatalf("error = %v, want unexpected RAM release artifact ref rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsRAMFuzzValidatorWithoutCurrentGitHead(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		ramDir := filepath.Join(root, "ram-contract")
		path := filepath.Join(ramDir, "ram-contract-release-manifest.json")
		var releaseManifest map[string]any
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &releaseManifest); err != nil {
			t.Fatal(err)
		}
		commands, ok := releaseManifest["commands"].([]any)
		if !ok {
			t.Fatalf("commands = %T, want []any", releaseManifest["commands"])
		}
		for _, rawCommand := range commands {
			command, ok := rawCommand.(map[string]any)
			if !ok {
				t.Fatalf("command = %T, want map[string]any", rawCommand)
			}
			if command["name"] == "validate-ram-contract-fuzz-oracle" {
				text, _ := command["command"].(string)
				command["command"] = strings.Replace(text, " --current-git-head "+memory100TestHead, "", 1)
			}
		}
		writeMemory100JSON(t, path, releaseManifest)
		writeMemory100HashManifest(t, ramDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected RAM fuzz validator command without --current-git-head to fail")
	}
	if got := err.Error(); !strings.Contains(got, "validate-ram-contract-fuzz-oracle") || !strings.Contains(got, "--current-git-head") {
		t.Fatalf("error = %v, want RAM fuzz validator current git head rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeIntegratedRAMContractReleaseManifestEnvelope(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		ramDir := filepath.Join(root, "integrated", "memory", "ram-contract")
		writeMemory100JSON(t, filepath.Join(ramDir, "ram-contract-release-manifest.json"), map[string]any{
			"schema":   "tetra.ram-contract.release-manifest.v1",
			"git_head": memory100TestHead,
		})
		writeMemory100HashManifest(t, ramDir)
		writeMemory100HashManifest(t, filepath.Join(root, "integrated", "memory"))
		writeMemory100HashManifest(t, filepath.Join(root, "integrated"))
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake integrated RAM contract release manifest envelope to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated RAM contract release manifest") || !strings.Contains(got, "command") {
		t.Fatalf("error = %v, want integrated RAM contract release manifest content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeIntegratedRAMContractHashManifest(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		ramDir := filepath.Join(root, "integrated", "memory", "ram-contract")
		writeMemory100JSON(t, filepath.Join(ramDir, "artifact-hashes.json"), map[string]any{
			"schema":    memory100HashSchema,
			"root":      ".",
			"artifacts": []any{},
		})
		writeMemory100HashManifest(t, filepath.Join(root, "integrated", "memory"))
		writeMemory100HashManifest(t, filepath.Join(root, "integrated"))
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake integrated RAM contract hash manifest to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated RAM contract artifact-hashes.json") || !strings.Contains(got, "artifacts") {
		t.Fatalf("error = %v, want integrated RAM contract hash manifest rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeIntegratedRAMContractReport(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		ramDir := filepath.Join(root, "integrated", "memory", "ram-contract")
		writeMemory100JSON(t, filepath.Join(ramDir, "ram-contract-report.json"), map[string]any{
			"schema_version": "tetra.ram-contract-report.v1",
			"git_head":       memory100TestHead,
			"target":         "linux-x64",
		})
		writeMemory100HashManifest(t, ramDir)
		writeMemory100HashManifest(t, filepath.Join(root, "integrated", "memory"))
		writeMemory100HashManifest(t, filepath.Join(root, "integrated"))
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake integrated RAM contract report to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated ram-contract-report.json") {
		t.Fatalf("error = %v, want integrated RAM contract report content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsStaleExtraRootHashManifestEntry(t *testing.T) {
	reportDir := writeMemory100Fixture(t, nil)
	extraRel := "docs-manifest/extra-evidence.json"
	writeMemory100JSON(t, filepath.Join(reportDir, filepath.FromSlash(extraRel)), map[string]any{
		"schema":   "tetra.memory-100.extra-evidence.v1",
		"status":   "pass",
		"git_head": memory100TestHead,
	})
	var manifest memory100HashManifest
	raw, err := os.ReadFile(filepath.Join(reportDir, "artifact-hashes.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Artifacts = append(manifest.Artifacts, memory100HashArtifact{
		Path:   extraRel,
		SHA256: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Size:   0,
		Schema: "tetra.memory-100.extra-evidence.v1",
	})
	sort.Slice(manifest.Artifacts, func(i, j int) bool {
		return manifest.Artifacts[i].Path < manifest.Artifacts[j].Path
	})
	writeMemory100JSON(t, filepath.Join(reportDir, "artifact-hashes.json"), manifest)

	err = validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected stale extra root hash manifest entry to fail")
	}
	if got := err.Error(); !strings.Contains(got, extraRel) || !strings.Contains(got, "sha256 mismatch") {
		t.Fatalf("error = %v, want stale extra root hash manifest entry rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsSymlinkRequiredArtifact(t *testing.T) {
	reportDir := writeMemory100Fixture(t, nil)
	path := filepath.Join(reportDir, "memory-production", "memory-production-linux-x64.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(reportDir, "artifact-hashes.json"), path); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected symlink required artifact to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory_production_report") || !strings.Contains(got, "must not be a symlink") {
		t.Fatalf("error = %v, want symlink required artifact rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsHashlessUnlistedRootArtifact(t *testing.T) {
	reportDir := writeMemory100Fixture(t, nil)
	writeMemory100JSON(t, filepath.Join(reportDir, "hashless-extra.json"), map[string]any{
		"schema":   "tetra.fake.hashless.v1",
		"git_head": memory100TestHead,
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected unlisted root artifact to fail")
	}
	if got := err.Error(); !strings.Contains(got, "unlisted Memory100 artifact hashless-extra.json") {
		t.Fatalf("error = %v, want unlisted root artifact rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsPathTraversalArtifactRef(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		manifest["artifacts"] = append(manifest["artifacts"].([]any), map[string]any{
			"path":   "../escape.json",
			"kind":   "docs_claim_policy",
			"schema": "tetra.memory-100.claim-policy.v1",
		})
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected path-traversal artifact ref to fail")
	}
	if got := err.Error(); !strings.Contains(got, "artifact path") || !strings.Contains(got, "invalid") {
		t.Fatalf("error = %v, want path-traversal artifact ref rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsPathTraversalHashEntry(t *testing.T) {
	reportDir := writeMemory100Fixture(t, nil)
	var manifest memory100HashManifest
	raw, err := os.ReadFile(filepath.Join(reportDir, "artifact-hashes.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Artifacts = append(manifest.Artifacts, memory100HashArtifact{
		Path:   "../escape.json",
		SHA256: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Size:   0,
		Schema: "tetra.fake.escape.v1",
	})
	writeMemory100JSON(t, filepath.Join(reportDir, "artifact-hashes.json"), manifest)

	err = validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected path-traversal hash entry to fail")
	}
	if got := err.Error(); !strings.Contains(got, "Memory100 hash path") || !strings.Contains(got, "invalid") {
		t.Fatalf("error = %v, want path-traversal hash entry rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsHashlessUnlistedNestedArtifact(t *testing.T) {
	reportDir := writeMemory100Fixture(t, nil)
	writeMemory100JSON(t, filepath.Join(reportDir, "memory-production", "ram-contract", "hashless-extra.json"), map[string]any{
		"schema":   "tetra.fake.hashless.v1",
		"git_head": memory100TestHead,
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected unlisted nested artifact to fail")
	}
	if got := err.Error(); !strings.Contains(got, "hashless-extra.json") || !strings.Contains(got, "unlisted") {
		t.Fatalf("error = %v, want unlisted nested artifact rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsCopiedAggregateCommandProvenance(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		copiedRoot := "reports/memory-100/old-copy/aggregate"
		manifest["commands"] = []any{
			map[string]any{"name": "memory-production-gate", "command": "bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir " + copiedRoot + "/memory-production"},
			map[string]any{"name": "ram-contract-gate", "command": "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir " + copiedRoot + "/ram-contract"},
			map[string]any{"name": "integrated-gate", "command": "bash scripts/release/post_v0_4/memory-islands-surface-production-gate.sh --report-dir " + copiedRoot + "/integrated"},
			map[string]any{"name": "memory-fuzz-short", "command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir " + copiedRoot + "/memory-fuzz --git-head " + memory100TestHead},
			map[string]any{"name": "memory-fuzz-validator", "command": "go run ./tools/cmd/validate-memory-fuzz-oracle --report " + copiedRoot + "/memory-fuzz/memory-fuzz-oracle.json --artifact-dir " + copiedRoot + "/memory-fuzz --current-git-head " + memory100TestHead},
			map[string]any{"name": "docs-claim-policy", "command": "go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json"},
			map[string]any{"name": "artifact-hashes-write", "command": "go run ./tools/cmd/validate-artifact-hashes --write --root " + copiedRoot + " --out " + copiedRoot + "/artifact-hashes.json"},
			map[string]any{"name": "memory-100-validator", "command": "go run ./tools/cmd/validate-memory-100-prod-stable --report-dir " + copiedRoot + " --current-git-head " + memory100TestHead},
		}
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected copied aggregate command provenance to fail")
	}
	if got := err.Error(); !strings.Contains(got, "Memory100 command") || !strings.Contains(got, "current report dir") {
		t.Fatalf("error = %v, want copied aggregate command provenance rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsAggregateManifestOlderThanChildEvidence(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		manifest["generated_at"] = "2026-06-10T10:59:59Z"
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected stale aggregate manifest generated_at to fail")
	}
	if got := err.Error(); !strings.Contains(got, "generated_at") || !strings.Contains(got, "older than") {
		t.Fatalf("error = %v, want stale aggregate generated_at rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeIntegratedManifestEnvelope(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "integrated", "memory-islands-surface-production-manifest.json"), map[string]any{
			"schema":   "tetra.memory-islands-surface.production-gate.v1",
			"git_head": memory100TestHead,
		})
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake integrated manifest envelope to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated manifest") || !strings.Contains(got, "command") {
		t.Fatalf("error = %v, want integrated manifest command/content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsStaleIntegratedManifestGeneratedAt(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		integratedDir := filepath.Join(root, "integrated")
		path := filepath.Join(integratedDir, "memory-islands-surface-production-manifest.json")
		var integratedManifest map[string]any
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &integratedManifest); err != nil {
			t.Fatal(err)
		}
		integratedManifest["generated_at"] = "2026-06-10T10:59:59Z"
		writeMemory100JSON(t, path, integratedManifest)
		writeMemory100HashManifest(t, integratedDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected stale integrated manifest generated_at to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated manifest generated_at") || !strings.Contains(got, "older than child evidence") {
		t.Fatalf("error = %v, want stale integrated manifest generated_at rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsCopiedIntegratedManifestCommandProvenance(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		integratedDir := filepath.Join(root, "integrated")
		copiedDir := "reports/memory-100/old-copy/integrated"
		writeMemory100JSON(t, filepath.Join(integratedDir, "memory-islands-surface-production-manifest.json"), map[string]any{
			"schema":        "tetra.memory-islands-surface.production-gate.v1",
			"status":        "pass",
			"git_head":      memory100TestHead,
			"generated_at":  "2026-06-10T11:00:00Z",
			"report_dir":    ".",
			"hash_manifest": "artifact-hashes.json",
			"commands":      memory100IntegratedCommandsForDir(copiedDir, memory100TestHead),
			"artifacts":     memory100IntegratedArtifactRefs(),
		})
		writeMemory100HashManifest(t, integratedDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected copied integrated manifest command provenance to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated manifest command") || !strings.Contains(got, "current integrated report dir") {
		t.Fatalf("error = %v, want copied integrated manifest command provenance rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsUnexpectedIntegratedManifestArtifactRef(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		integratedDir := filepath.Join(root, "integrated")
		path := filepath.Join(integratedDir, "memory-islands-surface-production-manifest.json")
		var integratedManifest map[string]any
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &integratedManifest); err != nil {
			t.Fatal(err)
		}
		artifacts := append([]any{}, integratedManifest["artifacts"].([]any)...)
		artifacts = append(artifacts, map[string]any{
			"path":   "docs-only/fake-evidence.json",
			"kind":   "docs_only_fake_evidence",
			"schema": "tetra.fake-evidence.v1",
		})
		integratedManifest["artifacts"] = artifacts
		writeMemory100JSON(t, path, integratedManifest)
		writeMemory100HashManifest(t, integratedDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected unexpected integrated manifest artifact ref to fail")
	}
	if got := err.Error(); !strings.Contains(got, "unexpected integrated manifest artifact kind docs_only_fake_evidence") {
		t.Fatalf("error = %v, want unexpected integrated artifact ref rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMissingIntegratedMemoryReleaseDeclaredArtifactHashCoverage(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "integrated", "memory")
		if err := os.Remove(filepath.Join(memoryDir, "targets.json")); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, filepath.Join(root, "integrated"))
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected missing integrated Memory release declared artifact hash coverage to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated memory artifact-hashes.json") || !strings.Contains(got, "targets.json") {
		t.Fatalf("error = %v, want missing integrated memory release declared artifact hash coverage rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeIntegratedMemoryReleaseTargetsReport(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "integrated", "memory")
		writeMemory100JSON(t, filepath.Join(memoryDir, "targets.json"), map[string]any{
			"supported":  []any{"linux-x64"},
			"build_only": []any{},
			"planned":    []any{},
			"targets":    []any{},
		})
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, filepath.Join(root, "integrated"))
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake integrated Memory release targets report to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated memory release targets.json") || !strings.Contains(got, "supported") {
		t.Fatalf("error = %v, want integrated Memory release targets content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeIntegratedMemoryProductionReport(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "integrated", "memory")
		writeMemory100JSON(t, filepath.Join(memoryDir, "memory-production-linux-x64.json"), map[string]any{
			"schema":   "tetra.memory.production.v1",
			"status":   "pass",
			"git_head": memory100TestHead,
		})
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, filepath.Join(root, "integrated"))
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake integrated memory production report to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated memory production report") {
		t.Fatalf("error = %v, want integrated memory production report content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeIntegratedIslandProofVerifier(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "integrated", "memory")
		writeMemory100JSON(t, filepath.Join(memoryDir, "island-proof-verifier.json"), map[string]any{
			"schema":   "tetra.island.proof.v1",
			"status":   "pass",
			"git_head": memory100TestHead,
		})
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, filepath.Join(root, "integrated"))
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake integrated island proof verifier to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated island proof") {
		t.Fatalf("error = %v, want integrated island proof content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeIntegratedMemoryFuzzTier1Summary(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		memoryDir := filepath.Join(root, "integrated", "memory")
		fuzzDir := filepath.Join(memoryDir, "memory-fuzz-tier1")
		writeMemory100JSON(t, filepath.Join(fuzzDir, "island-proof-fuzz-summary.json"), map[string]any{
			"schema_version": "tetra.island-proof-fuzz-summary.v1",
			"status":         "pass",
		})
		writeMemory100HashManifest(t, fuzzDir)
		writeMemory100HashManifest(t, memoryDir)
		writeMemory100HashManifest(t, filepath.Join(root, "integrated"))
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake integrated memory fuzz Tier 1 summary to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated memory fuzz") {
		t.Fatalf("error = %v, want integrated memory fuzz content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeIntegratedIslandsDebugSmoke(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		integratedDir := filepath.Join(root, "integrated")
		writeMemory100JSON(t, filepath.Join(integratedDir, "islands-debug-smoke.json"), map[string]any{
			"schema":   "tetra.release.v0_2_0.smoke-report.v1",
			"target":   "linux-x64",
			"git_head": memory100TestHead,
		})
		writeMemory100HashManifest(t, integratedDir)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake integrated islands debug smoke to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated islands debug smoke") {
		t.Fatalf("error = %v, want integrated islands debug smoke content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeIntegratedSurfaceReleaseSummary(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		surfaceDir := filepath.Join(root, "integrated", "surface-release-v1")
		writeMemory100JSON(t, filepath.Join(surfaceDir, "surface-release-summary.json"), map[string]any{
			"schema":   "tetra.surface.release.v1",
			"status":   "pass",
			"git_head": memory100TestHead,
		})
		writeMemory100HashManifest(t, surfaceDir)
		writeMemory100HashManifest(t, filepath.Join(root, "integrated"))
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake integrated surface release summary to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated surface release summary") {
		t.Fatalf("error = %v, want integrated surface release summary content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeIntegratedSafeViewLifetimeSummary(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "integrated", "safe-view-lifetime", "safe-view-lifetime-summary.json"), map[string]any{
			"schema":           "tetra.safe-view-lifetime.gate.v1",
			"status":           "pass",
			"bounded":          false,
			"release_blocking": true,
		})
		writeMemory100HashManifest(t, filepath.Join(root, "integrated"))
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake integrated safe-view lifetime summary to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated safe-view lifetime summary") {
		t.Fatalf("error = %v, want integrated safe-view lifetime summary content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeIntegratedSurfaceAPIStabilitySummary(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "integrated", "surface-api-stability-v1", "surface-api-stability-summary.json"), map[string]any{
			"schema":                  "tetra.surface.api-stability.v1",
			"status":                  "pass",
			"release_scope":           "surface-v1-linux-web",
			"docs_manifest_validated": false,
		})
		writeMemory100HashManifest(t, filepath.Join(root, "integrated"))
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake integrated surface API stability summary to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated surface API stability summary") {
		t.Fatalf("error = %v, want integrated surface API stability summary content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeIntegratedSurfaceExperimentalRuntimeReport(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		experimentalDir := filepath.Join(root, "integrated", "surface-experimental-regression")
		writeMemory100JSON(t, filepath.Join(experimentalDir, "surface-headless.json"), map[string]any{
			"schema": "tetra.surface.runtime.v1",
			"status": "pass",
			"target": "headless",
		})
		writeMemory100HashManifest(t, experimentalDir)
		writeMemory100HashManifest(t, filepath.Join(root, "integrated"))
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake integrated surface experimental runtime report to fail")
	}
	if got := err.Error(); !strings.Contains(got, "integrated surface experimental runtime report") {
		t.Fatalf("error = %v, want integrated surface experimental runtime content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsIncompleteMemoryFuzzBundle(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		if err := os.Remove(filepath.Join(root, "memory-fuzz", "summary.md")); err != nil {
			t.Fatalf("remove memory fuzz summary.md: %v", err)
		}
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected incomplete memory fuzz bundle to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory fuzz") || !strings.Contains(got, "summary.md") {
		t.Fatalf("error = %v, want memory fuzz summary.md rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMissingMemoryFuzzReproducerDirs(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		if err := os.RemoveAll(filepath.Join(root, "memory-fuzz", "reproducers", "compiler-crash")); err != nil {
			t.Fatalf("remove memory fuzz compiler crash reproducer dir: %v", err)
		}
		writeMemory100HashManifest(t, filepath.Join(root, "memory-fuzz"))
	})
	writeMemory100HashManifest(t, reportDir)

	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected missing memory fuzz reproducer dir to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory fuzz") || !strings.Contains(got, "reproducers/compiler-crash") {
		t.Fatalf("error = %v, want missing memory fuzz reproducer dir rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsCopiedMemoryFuzzBundleProvenance(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		copiedDir := "reports/memory-100/old-copy/memory-fuzz"
		writeMemory100FuzzSummary(t, filepath.Join(root, "memory-fuzz"), memory100TestHead, copiedDir)
		writeMemory100HashManifest(t, filepath.Join(root, "memory-fuzz"))
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected copied memory fuzz bundle provenance to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory fuzz") || !strings.Contains(got, "current artifact dir") {
		t.Fatalf("error = %v, want copied memory fuzz command provenance rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsUnclassifiedMemoryFuzzFailures(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		summaryPath := filepath.Join(root, "memory-fuzz", "summary.json")
		raw, err := os.ReadFile(summaryPath)
		if err != nil {
			t.Fatalf("read memory fuzz summary: %v", err)
		}
		raw = []byte(strings.Replace(string(raw), `"unclassified_failures": 0`, `"unclassified_failures": 1`, 1))
		if err := os.WriteFile(summaryPath, raw, 0o644); err != nil {
			t.Fatalf("write memory fuzz summary: %v", err)
		}
		writeMemory100HashManifest(t, filepath.Join(root, "memory-fuzz"))
	})
	writeMemory100HashManifest(t, reportDir)

	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected unclassified memory fuzz failures to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory fuzz") || !strings.Contains(got, "unclassified_failures") {
		t.Fatalf("error = %v, want memory fuzz unclassified failure rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMemoryFuzzSummaryWithoutReproducibilitySeeds(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		summaryPath := filepath.Join(root, "memory-fuzz", "summary.json")
		raw, err := os.ReadFile(summaryPath)
		if err != nil {
			t.Fatalf("read memory fuzz summary: %v", err)
		}
		var summary map[string]any
		if err := json.Unmarshal(raw, &summary); err != nil {
			t.Fatalf("parse memory fuzz summary: %v", err)
		}
		delete(summary, "reproducibility_seeds")
		raw, err = json.MarshalIndent(summary, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(summaryPath, append(raw, '\n'), 0o644); err != nil {
			t.Fatalf("write memory fuzz summary: %v", err)
		}
		writeMemory100HashManifest(t, filepath.Join(root, "memory-fuzz"))
	})
	writeMemory100HashManifest(t, reportDir)

	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected missing memory fuzz reproducibility seeds to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory fuzz") || !strings.Contains(got, "reproducibility_seeds") {
		t.Fatalf("error = %v, want memory fuzz reproducibility seed rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMockMemoryFuzzSeed(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		summaryPath := filepath.Join(root, "memory-fuzz", "summary.json")
		raw, err := os.ReadFile(summaryPath)
		if err != nil {
			t.Fatalf("read memory fuzz summary: %v", err)
		}
		var summary map[string]any
		if err := json.Unmarshal(raw, &summary); err != nil {
			t.Fatalf("parse memory fuzz summary: %v", err)
		}
		seeds := summary["reproducibility_seeds"].([]any)
		seeds[0] = "mock-seed:v0:memory100"
		writeMemory100JSON(t, summaryPath, summary)
		writeMemory100HashManifest(t, filepath.Join(root, "memory-fuzz"))
	})
	writeMemory100HashManifest(t, reportDir)

	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected mock memory fuzz seed to fail")
	}
	if got := err.Error(); !strings.Contains(got, "memory fuzz") || !strings.Contains(got, "forbidden marker \"mock\"") {
		t.Fatalf("error = %v, want memory fuzz mock seed rejection", err)
	}
}

func TestValidateMemory100ReportDirAcceptsRelativeReportDirWithAbsoluteMemoryFuzzProvenance(t *testing.T) {
	reportDir := writeMemory100Fixture(t, nil)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relReportDir, err := filepath.Rel(cwd, reportDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateMemory100ReportDir(relReportDir, memory100TestHead); err != nil {
		t.Fatalf("validateMemory100ReportDir with relative report dir failed: %v", err)
	}
}

func TestValidateMemory100ReportDirRejectsDirtySnapshotClaimingClean(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		manifest["git_dirty"] = false
		manifest["git_status_short_branch"] = []any{
			"## main...origin/main [ahead 47]",
			" M tools/cmd/validate-memory-100-prod-stable/main.go",
		}
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected dirty status snapshot claiming clean to fail")
	}
	if got := err.Error(); !strings.Contains(got, "git_dirty") || !strings.Contains(got, "dirty") {
		t.Fatalf("error = %v, want git_dirty dirty snapshot rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsDirtyCleanReleaseVerdict(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		manifest["git_dirty"] = true
		manifest["git_status_short_branch"] = []any{
			"## main...origin/main [ahead 47]",
			" M tools/cmd/validate-memory-100-prod-stable/main.go",
		}
		manifest["verdict"] = "MEMORY100_CLEAN_RELEASE_CANDIDATE"
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected dirty clean-release verdict to fail")
	}
	if got := err.Error(); !strings.Contains(got, "dirty") || !strings.Contains(got, "verdict") {
		t.Fatalf("error = %v, want dirty verdict downgrade rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsDirtyLocalVerdict(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		manifest["git_dirty"] = true
		manifest["git_status_short_branch"] = []any{
			"## main...origin/main [ahead 47]",
			" M tools/cmd/validate-memory-100-prod-stable/main.go",
		}
		manifest["verdict"] = "MEMORY100_SCOPED_READY_LOCAL"
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected dirty local verdict to fail")
	}
	if got := err.Error(); !strings.Contains(got, "dirty") || !strings.Contains(got, "MEMORY100_SCOPED_READY_DIRTY") {
		t.Fatalf("error = %v, want dirty scoped verdict rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsCleanDirtyVerdict(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		manifest["verdict"] = "MEMORY100_SCOPED_READY_DIRTY"
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected clean dirty verdict to fail")
	}
	if got := err.Error(); !strings.Contains(got, "clean") || !strings.Contains(got, "MEMORY100_SCOPED_READY_LOCAL") {
		t.Fatalf("error = %v, want clean scoped verdict rejection", err)
	}
}
