package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
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

func TestValidateMemory100ReportDirRejectsPlaceholderRawMemoryContract(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "raw-memory-contract", "raw-memory-contract.json"), memory100PlaceholderJSON("tetra.raw-memory-contract.v1", memory100TestHead))
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected placeholder raw memory contract to fail")
	}
	if got := err.Error(); !strings.Contains(got, "raw memory contract") || !strings.Contains(got, "operation") {
		t.Fatalf("error = %v, want raw memory contract operation rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsIncompleteRawPointerMatrix(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "raw-memory-contract", "raw-memory-contract.json"), map[string]any{
			"schema":   "tetra.raw-memory-contract.v1",
			"status":   "pass",
			"git_head": memory100TestHead,
			"operations": []any{
				map[string]any{"name": "core.alloc_bytes", "source_artifacts": []any{"compiler/internal/runtimeabi/raw_pointer_bounds_test.go"}, "positive_tests": []any{"allocation-base metadata"}},
				map[string]any{"name": "core.ptr_add", "source_artifacts": []any{"compiler/internal/runtimeabi/raw_pointer_bounds_test.go"}, "negative_tests": []any{"negative offset", "allocation upper bound"}},
				map[string]any{"name": "raw_slice_from_parts", "source_artifacts": []any{"compiler/tests/semantics/memory_ideal_v5_raw_pointer_test.go"}, "negative_tests": []any{"outside unsafe", "negative length", "i32 byte overflow"}},
				map[string]any{"name": "memcpy_u8", "source_artifacts": []any{"lib/core/memory.tetra"}, "positive_tests": []any{"cap.mem helper path"}, "negative_tests": []any{"negative length"}, "non_claims": []any{"no overlapping memcpy safety claim"}},
				map[string]any{"name": "memset_u8", "source_artifacts": []any{"lib/core/memory.tetra"}, "positive_tests": []any{"cap.mem helper path"}, "negative_tests": []any{"negative length"}},
				map[string]any{"name": "cap.mem", "source_artifacts": []any{"lib/core/capability.tetra"}, "negative_tests": []any{"unsafe_unknown promotion rejected", "cap.mem overclaim rejected"}, "non_claims": []any{"no arbitrary external pointer safety claim"}},
			},
		})
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected incomplete raw pointer matrix to fail")
	}
	if got := err.Error(); !strings.Contains(got, "raw memory contract") || !strings.Contains(got, "access-width overflow") {
		t.Fatalf("error = %v, want access-width overflow evidence rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMissingRawLoadStoreMetadata(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.raw-memory-contract.v1", memory100TestHead)
		var filtered []any
		for _, operation := range report["operations"].([]any) {
			if operation.(map[string]any)["name"] == "raw_load_store_metadata" {
				continue
			}
			filtered = append(filtered, operation)
		}
		report["operations"] = filtered
		writeMemory100JSON(t, filepath.Join(root, "raw-memory-contract", "raw-memory-contract.json"), report)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected missing raw load/store metadata evidence to fail")
	}
	if got := err.Error(); !strings.Contains(got, "raw memory contract") || !strings.Contains(got, "raw_load_store_metadata") {
		t.Fatalf("error = %v, want raw load/store metadata evidence rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsPlaceholderAllocationLowering(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"), memory100PlaceholderJSON("tetra.allocation-lowering.v1", memory100TestHead))
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected placeholder allocation lowering report to fail")
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") || !strings.Contains(got, "decision") {
		t.Fatalf("error = %v, want allocation lowering decision rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsPlaceholderProofStoreSummary(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "proof-store", "proof-store-summary.json"), memory100PlaceholderJSON("tetra.proof-store-summary.v1", memory100TestHead))
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected placeholder proof store summary to fail")
	}
	if got := err.Error(); !strings.Contains(got, "proof store summary") || !strings.Contains(got, "summary") {
		t.Fatalf("error = %v, want proof store summary content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsLoweringMismatchWithoutBlocker(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.allocation-lowering.v1", memory100TestHead)
		for _, raw := range report["decisions"].([]any) {
			decision := raw.(map[string]any)
			if decision["name"] == "stack_trusted_no_escape" {
				decision["planned_storage"] = "Stack"
				decision["actual_lowering_storage"] = "Heap"
				delete(decision, "blocker_artifact")
			}
		}
		writeMemory100JSON(t, filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"), report)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected planned/actual lowering mismatch without blocker to fail")
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") || !strings.Contains(got, "mismatch") || !strings.Contains(got, "blocker_artifact") {
		t.Fatalf("error = %v, want allocation lowering mismatch blocker rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsHeapCopyBlockersWithoutImpact(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.allocation-lowering.v1", memory100TestHead)
		for _, raw := range report["decisions"].([]any) {
			decision := raw.(map[string]any)
			switch decision["name"] {
			case "heap_fallback_blocker":
				delete(decision, "budget_impact")
			case "copy_blocker":
				delete(decision, "grade_impact")
			}
		}
		writeMemory100JSON(t, filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"), report)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected heap/copy blocker impact omissions to fail")
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") || (!strings.Contains(got, "budget_impact") && !strings.Contains(got, "grade_impact")) {
		t.Fatalf("error = %v, want heap/copy blocker impact rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsBlockedAllocationLoweringWithoutCoveredSites(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.allocation-lowering.v1", memory100TestHead)
		for _, raw := range report["decisions"].([]any) {
			decision := raw.(map[string]any)
			if decision["name"] == "heap_fallback_blocker" {
				delete(decision, "covered_site_ids")
			}
		}
		writeMemory100JSON(t, filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"), report)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected blocked allocation lowering without covered_site_ids to fail")
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") || !strings.Contains(got, "covered_site_ids") {
		t.Fatalf("error = %v, want allocation lowering covered_site_ids rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsCopyBlockerWhenNoCopyRowsObserved(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.allocation-lowering.v1", memory100TestHead)
		for _, raw := range report["decisions"].([]any) {
			decision := raw.(map[string]any)
			if decision["name"] == "copy_blocker" {
				decision["status"] = "blocked"
				decision["blocker_artifact"] = "ram-contract/copy-blockers.json"
				decision["covered_site_ids"] = []any{"site:main:heap"}
			}
		}
		writeMemory100JSON(t, filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"), report)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected copy blocker without copy RAM rows to fail")
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") || !strings.Contains(got, "copy") || !strings.Contains(got, "site:main:heap") {
		t.Fatalf("error = %v, want copy covered_site_ids/RAM row mismatch rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsAllocationLoweringActualStorageMismatchingRAMPlacement(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.allocation-lowering.v1", memory100TestHead)
		for _, raw := range report["decisions"].([]any) {
			decision := raw.(map[string]any)
			if decision["name"] == "heap_fallback_blocker" {
				decision["actual_lowering_storage"] = "Stack"
			}
		}
		writeMemory100JSON(t, filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"), report)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected actual lowering storage/RAM placement mismatch to fail")
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") || !strings.Contains(got, "actual_lowering_storage") || !strings.Contains(got, "site:main:heap") {
		t.Fatalf("error = %v, want actual lowering storage/RAM placement rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsUnbackedAllocationLoweringProofDecision(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.allocation-lowering.v1", memory100TestHead)
		for _, raw := range report["decisions"].([]any) {
			decision := raw.(map[string]any)
			switch decision["name"] {
			case "stack_trusted_no_escape", "lowering_storage_match":
				decision["status"] = "proven"
				decision["proof_artifact"] = "ram-contract/proof-store-summary.json"
			}
		}
		writeMemory100JSON(t, filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"), report)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected proof-backed allocation lowering without RAM proof-backed trusted rows to fail")
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") || !strings.Contains(got, "proof-backed") {
		t.Fatalf("error = %v, want allocation lowering proof-backed RAM consistency rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsTrustedRAMRowsWithoutAllocationLoweringCoverage(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		proofID := "proof:stack:noescape"
		writeMemory100JSON(t, filepath.Join(root, "ram-contract", "ram-contract-report.json"), map[string]any{
			"schema_version": "tetra.ram-contract-report.v1",
			"git_head":       memory100TestHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"rows": []any{
				map[string]any{
					"site_id":           "site:main:stack",
					"value_id":          "stack",
					"function":          "main",
					"intent":            "allocation",
					"requested_bytes":   64,
					"bounded":           true,
					"owner":             "function:main",
					"lifetime":          "function:main",
					"escape_status":     "no_escape",
					"placement":         "stack",
					"proof_ids":         []any{proofID},
					"blockers":          []any{},
					"contract_grade":    "M1",
					"validation_status": "validated",
				},
			},
			"proofs": []any{
				map[string]any{
					"proof_id":    proofID,
					"kind":        "allocation_placement",
					"subject":     "site:main:stack stack no_escape",
					"stable_hash": strings.Repeat("a", 64),
					"status":      "proven",
				},
			},
			"summary":    map[string]any{"row_count": 1, "artifact_grade": "M1", "heap_rows": 0, "copy_rows": 0, "unbounded_rows": 0, "budget_bytes": 64},
			"non_claims": []any{"no Memory 100% claim", "no full formal proof claim"},
		})
		writeMemory100JSON(t, filepath.Join(root, "ram-contract", "proof-store-summary.json"), map[string]any{
			"schema_version": "tetra.proof-store-summary.v1",
			"git_head":       memory100TestHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"proofs": []any{
				map[string]any{
					"proof_id":    proofID,
					"kind":        "allocation_placement",
					"subject":     "site:main:stack stack no_escape",
					"stable_hash": strings.Repeat("a", 64),
					"status":      "proven",
				},
			},
			"summary":    map[string]any{"proof_count": 1, "proven": 1, "conservative": 0, "rejected": 0, "unknown": 0},
			"non_claims": []any{"no full formal proof claim"},
		})
		writeMemory100JSON(t, filepath.Join(root, "ram-contract", "heap-blockers.json"), map[string]any{
			"schema_version": "tetra.ram-blockers.v1",
			"kind":           "heap",
			"git_head":       memory100TestHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"rows":           []any{},
			"non_claims":     []any{"no Memory 100% claim"},
		})
		allocation := memory100JSON("tetra.allocation-lowering.v1", memory100TestHead)
		for _, raw := range allocation["decisions"].([]any) {
			decision := raw.(map[string]any)
			if decision["name"] != "heap_fallback_blocker" {
				continue
			}
			decision["status"] = "not_observed"
			delete(decision, "blocker_artifact")
			delete(decision, "blocker_reason")
			delete(decision, "budget_impact")
			delete(decision, "grade_impact")
			delete(decision, "validator_coverage")
			delete(decision, "covered_site_ids")
			decision["source_artifacts"] = []any{"ram-contract/heap-blockers.json", "ram-contract/ram-contract-report.json"}
		}
		writeMemory100JSON(t, filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"), allocation)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected proof-backed trusted RAM row without allocation-lowering coverage to fail")
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") || !strings.Contains(got, "proof-backed trusted") || !strings.Contains(got, "site:main:stack") {
		t.Fatalf("error = %v, want proof-backed trusted RAM coverage rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsUnclassifiedRAMHeapRows(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "ram-contract", "ram-contract-report.json"), map[string]any{
			"schema_version": "tetra.ram-contract-report.v1",
			"git_head":       memory100TestHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"rows": []any{
				map[string]any{
					"site_id":           "site:main:heap",
					"value_id":          "heap",
					"function":          "main",
					"intent":            "heap_fallback",
					"requested_bytes":   64,
					"bounded":           true,
					"owner":             "function:main",
					"lifetime":          "function:main",
					"escape_status":     "unknown",
					"placement":         "heap_bounded",
					"proof_ids":         []any{},
					"blockers":          []any{},
					"contract_grade":    "M4",
					"validation_status": "unclassified",
				},
			},
			"proofs":     []any{},
			"summary":    map[string]any{"row_count": 1, "artifact_grade": "M4", "heap_rows": 1, "copy_rows": 0, "unbounded_rows": 0, "budget_bytes": 64},
			"non_claims": []any{"no Memory 100% claim", "no full formal proof claim"},
		})
		writeMemory100JSON(t, filepath.Join(root, "ram-contract", "heap-blockers.json"), map[string]any{
			"schema_version": "tetra.ram-blockers.v1",
			"kind":           "heap",
			"git_head":       memory100TestHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"rows":           []any{},
			"non_claims":     []any{"no Memory 100% claim"},
		})
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected unclassified RAM heap row to fail")
	}
	if got := err.Error(); !strings.Contains(got, "ram-contract") || (!strings.Contains(got, "unclassified") && !strings.Contains(got, "heap")) {
		t.Fatalf("error = %v, want RAM heap classification rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeRAMContractFuzzOracle(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "ram-contract", "fuzz", "ram-contract-fuzz-oracle.json"), map[string]any{
			"schema_version": "tetra.ram-contract-fuzz-oracle.v1",
			"git_head":       memory100TestHead,
			"generated_at":   "2026-06-10T11:00:00Z",
			"summary":        map[string]any{"mutations": 0, "rejected": 0},
			"observations":   []any{},
			"non_claims":     []any{"not Memory 100%"},
		})
		writeMemory100HashManifest(t, filepath.Join(root, "ram-contract"))
	})
	writeMemory100HashManifest(t, reportDir)

	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake RAM contract fuzz oracle to fail")
	}
	if got := err.Error(); !strings.Contains(got, "RAM contract fuzz") || !strings.Contains(got, "mutation") {
		t.Fatalf("error = %v, want RAM contract fuzz mutation rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsPlaceholderLeakResourceReport(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "leak-resource", "leak-resource-report.json"), memory100PlaceholderJSON("tetra.leak-resource.v1", memory100TestHead))
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected placeholder leak/resource report to fail")
	}
	if got := err.Error(); !strings.Contains(got, "leak/resource") || !strings.Contains(got, "check") {
		t.Fatalf("error = %v, want leak/resource check rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMissingSemanticSafetyMatrix(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		_ = os.Remove(filepath.Join(root, "semantic-safety", "memory-semantic-safety-matrix.json"))
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected missing semantic safety matrix to fail")
	}
	if got := err.Error(); !strings.Contains(got, "semantic") || !strings.Contains(got, "memory-semantic-safety-matrix.json") {
		t.Fatalf("error = %v, want semantic safety matrix rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMissingProofTransitionReport(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		_ = os.Remove(filepath.Join(root, "proof-transition", "proof-transition-report.json"))
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected missing proof transition report to fail")
	}
	if got := err.Error(); !strings.Contains(got, "proof_transition_report") || !strings.Contains(got, "proof-transition/proof-transition-report.json") {
		t.Fatalf("error = %v, want proof transition artifact rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMissingRuntimeMemoryContract(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		_ = os.Remove(filepath.Join(root, "runtime-memory", "runtime-memory-contract.json"))
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected missing runtime memory contract to fail")
	}
	if got := err.Error(); !strings.Contains(got, "runtime_memory_contract") || !strings.Contains(got, "runtime-memory/runtime-memory-contract.json") {
		t.Fatalf("error = %v, want runtime memory contract artifact rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsInvalidatedProofTransitionWithoutRecheck(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.proof-transition-report.v1", memory100TestHead)
		for _, raw := range report["rows"].([]any) {
			row := raw.(map[string]any)
			if row["name"] == "optimization_invalidates_bounds_proofs" {
				row["consumer_action"] = "consumed_directly"
			}
		}
		writeMemory100JSON(t, filepath.Join(root, "proof-transition", "proof-transition-report.json"), report)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected invalidated proof transition without recheck to fail")
	}
	if got := err.Error(); !strings.Contains(got, "proof transition") || !strings.Contains(got, "consumer_action") {
		t.Fatalf("error = %v, want proof transition consumer_action rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsRuntimeMemoryTargetOverclaim(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		manifest["target_matrix"] = []any{"linux-x64", "windows-x64"}
		report := memory100JSON("tetra.runtime-memory-contract.v1", memory100TestHead)
		for _, raw := range report["rows"].([]any) {
			row := raw.(map[string]any)
			if row["target"] == "windows-x64" {
				row["included_in_memory100_target_matrix"] = true
				row["runtime_status"] = "production"
				row["memory_run"] = "yes"
				row["memory_claim_level"] = "production_host_runtime"
				delete(row, "excluded_reason")
			}
		}
		writeMemory100JSON(t, filepath.Join(root, "runtime-memory", "runtime-memory-contract.json"), report)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected runtime memory target overclaim to fail")
	}
	if got := err.Error(); !strings.Contains(got, "runtime memory") || !strings.Contains(got, "windows-x64") {
		t.Fatalf("error = %v, want runtime memory windows target overclaim rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsIncompleteSemanticSafetyMatrix(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		matrix := memory100JSON("tetra.memory-semantic-safety-matrix.v1", memory100TestHead)
		rows := matrix["rows"].([]any)
		matrix["rows"] = rows[:len(rows)-1]
		writeMemory100JSON(t, filepath.Join(root, "semantic-safety", "memory-semantic-safety-matrix.json"), matrix)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected incomplete semantic safety matrix to fail")
	}
	if got := err.Error(); !strings.Contains(got, "semantic safety matrix") || !strings.Contains(got, "actor_task_non_sendable_transfer") {
		t.Fatalf("error = %v, want missing actor/task semantic row rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsPlaceholderClaimPolicy(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(t, filepath.Join(root, "docs-manifest", "claim-policy.json"), memory100PlaceholderJSON("tetra.memory-100.claim-policy.v1", memory100TestHead))
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected placeholder claim policy to fail")
	}
	if got := err.Error(); !strings.Contains(got, "claim policy") || !strings.Contains(got, "forbidden_claims") {
		t.Fatalf("error = %v, want claim policy forbidden_claims rejection", err)
	}
}

func writeMemory100Fixture(t *testing.T, mutate func(root string, manifest map[string]any)) string {
	t.Helper()
	root := t.TempDir()
	generatedAt := time.Date(2026, 6, 10, 11, 0, 0, 0, time.UTC).Format(time.RFC3339)

	required := []struct {
		Path   string
		Kind   string
		Schema string
	}{
		{"memory-production/memory-production-linux-x64.json", "memory_production_report", "tetra.memory.production.v1"},
		{"memory-production/memory-release-manifest.json", "memory_release_manifest", "tetra.memory.release-manifest.v1"},
		{"memory-production/artifact-hashes.json", "memory_production_hash_manifest", "tetra.release-artifact-hashes.v1alpha1"},
		{"ram-contract/ram-contract-release-manifest.json", "ram_contract_release_manifest", "tetra.ram-contract.release-manifest.v1"},
		{"ram-contract/ram-contract-report.json", "ram_contract_report", "tetra.ram-contract-report.v1"},
		{"ram-contract/memory-grade-report.json", "ram_memory_grade_report", "tetra.memory-grade-report.v1"},
		{"ram-contract/proof-store-summary.json", "ram_proof_store_summary", "tetra.proof-store-summary.v1"},
		{"ram-contract/validation-pipeline-coverage.json", "ram_validation_pipeline_coverage", "tetra.validation-pipeline-coverage.v1"},
		{"ram-contract/heap-blockers.json", "ram_heap_blockers", "tetra.ram-blockers.v1"},
		{"ram-contract/copy-blockers.json", "ram_copy_blockers", "tetra.ram-blockers.v1"},
		{"ram-contract/fuzz/ram-contract-fuzz-oracle.json", "ram_contract_fuzz_oracle", "tetra.ram-contract-fuzz-oracle.v1"},
		{"ram-contract/artifact-hashes.json", "ram_contract_hash_manifest", "tetra.release-artifact-hashes.v1alpha1"},
		{"raw-memory-contract/raw-memory-contract.json", "raw_memory_contract_report", "tetra.raw-memory-contract.v1"},
		{"allocation-lowering/allocation-lowering-report.json", "allocation_lowering_report", "tetra.allocation-lowering.v1"},
		{"proof-store/proof-store-summary.json", "proof_store_summary", "tetra.proof-store-summary.v1"},
		{"proof-transition/proof-transition-report.json", "proof_transition_report", "tetra.proof-transition-report.v1"},
		{"runtime-memory/runtime-memory-contract.json", "runtime_memory_contract", "tetra.runtime-memory-contract.v1"},
		{"memory-fuzz/memory-fuzz-oracle.json", "memory_fuzz_oracle_report", "tetra.memory-fuzz.oracle.v1"},
		{"memory-fuzz/artifact-hashes.json", "memory_fuzz_hash_manifest", "tetra.release-artifact-hashes.v1alpha1"},
		{"semantic-safety/memory-semantic-safety-matrix.json", "memory_semantic_safety_matrix", "tetra.memory-semantic-safety-matrix.v1"},
		{"leak-resource/leak-resource-report.json", "leak_resource_report", "tetra.leak-resource.v1"},
		{"integrated/memory-islands-surface-production-manifest.json", "integrated_memory_islands_surface_manifest", "tetra.memory-islands-surface.production-gate.v1"},
		{"integrated/artifact-hashes.json", "integrated_hash_manifest", "tetra.release-artifact-hashes.v1alpha1"},
		{"docs-manifest/claim-policy.json", "docs_claim_policy", "tetra.memory-100.claim-policy.v1"},
	}

	var artifactRefs []any
	for _, req := range required {
		writeMemory100JSON(t, filepath.Join(root, filepath.FromSlash(req.Path)), memory100ArtifactJSON(req.Kind, req.Schema, memory100TestHead))
		artifactRefs = append(artifactRefs, map[string]any{
			"path":   req.Path,
			"kind":   req.Kind,
			"schema": req.Schema,
		})
	}
	writeMemory100RAMContractReleaseManifest(t, filepath.Join(root, "ram-contract"), memory100TestHead)
	writeMemory100HashManifest(t, filepath.Join(root, "ram-contract"))
	memoryProductionDir := filepath.Join(root, "memory-production")
	writeMemory100MemoryReleaseManifest(t, memoryProductionDir, memory100TestHead)
	writeMemory100TargetReport(t, memoryProductionDir)
	writeMemory100MemoryFuzzBundle(t, filepath.Join(memoryProductionDir, "memory-fuzz-tier1"), memory100TestHead)
	writeMemory100IntegratedIslandProofEvidence(t, memoryProductionDir, memory100TestHead)
	writeMemory100RAMContractArtifacts(t, filepath.Join(memoryProductionDir, "ram-contract"), memory100TestHead)
	writeMemory100RAMContractReleaseManifest(t, filepath.Join(memoryProductionDir, "ram-contract"), memory100TestHead)
	writeMemory100HashManifest(t, filepath.Join(memoryProductionDir, "ram-contract"))
	writeMemory100HashManifest(t, memoryProductionDir)
	writeMemory100IntegratedBundle(t, filepath.Join(root, "integrated"), memory100TestHead)
	writeMemory100MemoryFuzzBundle(t, filepath.Join(root, "memory-fuzz"), memory100TestHead)
	reportPath := func(rel string) string {
		if rel == "" {
			return filepath.ToSlash(root)
		}
		return filepath.ToSlash(filepath.Join(root, filepath.FromSlash(rel)))
	}

	manifest := map[string]any{
		"schema":    "tetra.memory-100.prod-stable.v1",
		"status":    "pass",
		"verdict":   "MEMORY100_SCOPED_READY_LOCAL",
		"git_head":  memory100TestHead,
		"git_dirty": false,
		"git_status_short_branch": []any{
			"## main",
		},
		"generated_at":  generatedAt,
		"target_matrix": []any{"linux-x64"},
		"hash_manifest": "artifact-hashes.json",
		"claims": []any{
			"Memory/RAM production-stable criteria passed locally for the scoped target matrix.",
		},
		"non_claims": []any{
			"no universal Memory 100% claim",
			"no full formal proof claim",
			"no all-target memory parity claim",
			"no arbitrary unsafe external pointer safety claim",
			"no C/Rust parity or performance superiority claim",
		},
		"commands": []any{
			map[string]any{"name": "memory-production-gate", "command": "bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir " + reportPath("memory-production")},
			map[string]any{"name": "ram-contract-gate", "command": "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir " + reportPath("ram-contract")},
			map[string]any{"name": "integrated-gate", "command": "bash scripts/release/post_v0_4/memory-islands-surface-production-gate.sh --report-dir " + reportPath("integrated")},
			map[string]any{"name": "memory-fuzz-short", "command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir " + reportPath("memory-fuzz") + " --git-head " + memory100TestHead},
			map[string]any{"name": "memory-fuzz-validator", "command": "go run ./tools/cmd/validate-memory-fuzz-oracle --report " + reportPath("memory-fuzz/memory-fuzz-oracle.json") + " --artifact-dir " + reportPath("memory-fuzz") + " --current-git-head " + memory100TestHead},
			map[string]any{"name": "docs-claim-policy", "command": "go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json"},
			map[string]any{"name": "artifact-hashes-write", "command": "go run ./tools/cmd/validate-artifact-hashes --write --root " + reportPath("") + " --out " + reportPath("artifact-hashes.json")},
			map[string]any{"name": "memory-100-validator", "command": "go run ./tools/cmd/validate-memory-100-prod-stable --report-dir " + reportPath("") + " --current-git-head " + memory100TestHead},
		},
		"artifacts": artifactRefs,
	}
	if mutate != nil {
		mutate(root, manifest)
	}
	writeMemory100JSON(t, filepath.Join(root, "memory-100-prod-stable-manifest.json"), manifest)
	writeMemory100HashManifest(t, root)
	return root
}

func memory100ArtifactJSON(kind string, schema string, gitHead string) map[string]any {
	switch kind {
	case "memory_production_report":
		caseNames := []string{
			"allocator alloc/free lifecycle",
			"allocator failure semantics",
			"allocator invalid size precondition",
			"cap.mem unsafe boundary",
			"memcpy/memset capability path",
			"runtime bounds check",
			"raw ptr_add negative offset bounds",
			"raw ptr_add allocation upper bound",
			"raw allocation-base i32 access width",
			"raw allocation-base ptr access width",
			"raw slice negative length",
			"raw slice i32 length byte overflow",
			"raw pointer bounds metadata report",
			"memcpy/memset negative length",
			"reject use-after-free",
			"reject double-free",
			"reject borrow escape",
			"reject aliasing violation",
			"callable mutable capture heap escape",
			"reject actor task transfer violation",
			"heap closure handle coverage",
			"slice struct borrow escape coverage",
			"function-typed slice aggregate borrow escape coverage",
			"actornet broker close-without-cancel leak smoke",
			"compiler resource finalization diagnostics",
			"real memory examples",
			"stress allocator reuse",
			"deterministic memcpy/memset fuzz",
		}
		var cases []any
		for _, name := range caseNames {
			kind := "positive"
			expected := ""
			lower := strings.ToLower(name)
			if strings.Contains(lower, "stress") || strings.Contains(lower, "fuzz") || strings.Contains(lower, "leak smoke") || strings.Contains(lower, "diagnostics") {
				kind = "stress"
			}
			if strings.Contains(lower, "reject") || strings.Contains(lower, "negative") || strings.Contains(lower, "bounds") || strings.Contains(lower, "overflow") || strings.Contains(lower, "unsafe") || strings.Contains(lower, "invalid") || strings.Contains(lower, "failure") {
				kind = "negative"
				expected = "TETRA_MEMORY_CONTRACT"
			}
			row := map[string]any{"name": name, "kind": kind, "ran": true, "pass": true}
			if expected != "" {
				row["expected_error"] = expected
			}
			cases = append(cases, row)
		}
		contractNames := []string{
			"allocator runtime model",
			"allocator failure semantics",
			"ownership escape model",
			"unsafe cap.mem raw memory rules",
			"runtime bounds diagnostics",
			"raw pointer bounds metadata",
			"host resource leak and finalization checks",
			"actor task transfer rules",
		}
		var contracts []any
		for _, name := range contractNames {
			contracts = append(contracts, map[string]any{"name": name, "status": "pass", "evidence": "scoped linux-x64 release evidence"})
		}
		auditRequirements := []string{
			"stable allocator/runtime memory model",
			"ownership/borrow/consume escape model",
			"heap, slices, structs, and closures memory coverage",
			"unsafe/cap.mem/raw memory/memcpy/memset rules",
			"runtime bounds checks and diagnostics",
			"raw pointer bounds metadata",
			"stress/fuzz evidence",
			"measured memory benchmark improvement",
			"use-after-free, double-free, borrow escape, and aliasing safety",
			"actor/task transfer safety",
			"leak/resource finalization evidence",
			"real memory examples",
			"safe memory documentation",
			"release-gate entrypoint",
		}
		var audit []any
		for _, requirement := range auditRequirements {
			audit = append(audit, map[string]any{"requirement": requirement, "artifact": "memory-production-linux-x64.json", "evidence": "scoped linux-x64 release evidence", "result": "pass"})
		}
		exitZero := 0
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"target":   "linux-x64",
			"host":     "linux-x64",
			"runtime":  "memory-linux-x64",
			"source":   "memory-production-linux-x64-smoke.sh",
			"processes": []any{
				map[string]any{"name": "memory production build", "kind": "build", "path": "./compiler", "ran": true, "pass": true, "exit_code": exitZero},
				map[string]any{"name": "memory production app smoke", "kind": "app", "path": "./examples/memory_ownership_demo.tetra", "ran": true, "pass": true, "exit_code": exitZero},
				map[string]any{"name": "actornet close-without-cancel leak coverage", "kind": "stress", "path": "./cli/internal/actornet TestBrokerCloseWithoutCancelStopsServeWatcher", "ran": true, "pass": true, "exit_code": exitZero},
				map[string]any{"name": "compiler resource finalization diagnostics", "kind": "stress", "path": "./compiler/tests/runtime TestTaskHandleFinalization TestTaskGroupFinalization TestIslandFinalization", "ran": true, "pass": true, "exit_code": exitZero},
			},
			"benchmarks": []any{
				map[string]any{
					"name":              "small heap allocation syscall reduction",
					"kind":              "allocator",
					"metric":            "estimated_os_syscalls",
					"unit":              "syscalls",
					"baseline_value":    100,
					"measured_value":    50,
					"improvement_ratio": 2.0,
					"evidence":          "per_core_small_heap same_core_same_size_class_free_list allocation report schema v2 64KiB chunk refill",
					"ran":               true,
					"pass":              true,
				},
			},
			"contracts": contracts,
			"cases":     cases,
			"audit":     audit,
		}
	case "memory_release_manifest":
		return map[string]any{
			"schema":        schema,
			"target":        "linux-x64",
			"git_head":      gitHead,
			"generated_at":  "2026-06-10T11:00:00Z",
			"report_dir":    ".",
			"hash_manifest": "artifact-hashes.json",
			"commands":      []any{},
			"artifacts":     []any{},
		}
	case "ram_contract_report":
		return map[string]any{
			"schema_version": schema,
			"git_head":       gitHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"rows": []any{
				map[string]any{
					"site_id":           "site:main:heap",
					"value_id":          "heap",
					"function":          "main",
					"intent":            "heap_fallback",
					"requested_bytes":   8192,
					"bounded":           false,
					"owner":             "function:main",
					"lifetime":          "function:main",
					"escape_status":     "unknown",
					"placement":         "heap_unbounded",
					"proof_ids":         []any{},
					"blockers":          []any{"unknown_size"},
					"contract_grade":    "M5",
					"validation_status": "conservative",
				},
			},
			"proofs":     []any{},
			"summary":    map[string]any{"row_count": 1, "artifact_grade": "M5", "heap_rows": 1, "copy_rows": 0, "unbounded_rows": 1, "budget_bytes": 8192},
			"non_claims": []any{"no Memory 100% claim", "no full formal proof claim"},
		}
	case "ram_memory_grade_report":
		return map[string]any{
			"schema_version": schema,
			"git_head":       gitHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"artifact_grade": "M5",
			"functions":      []any{},
			"summary":        map[string]any{"row_count": 1, "artifact_grade": "M5", "heap_rows": 1, "copy_rows": 0, "unbounded_rows": 1, "budget_bytes": 8192},
			"non_claims":     []any{"no Memory 100% claim"},
		}
	case "ram_proof_store_summary", "proof_store_summary":
		return map[string]any{
			"schema_version": schema,
			"git_head":       gitHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"proofs":         []any{},
			"summary":        map[string]any{"proof_count": 0, "proven": 0, "conservative": 0, "rejected": 0, "unknown": 0},
			"non_claims":     []any{"no full formal proof claim"},
		}
	case "ram_validation_pipeline_coverage":
		return map[string]any{
			"schema_version": schema,
			"git_head":       gitHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"entries": []any{
				map[string]any{"entrypoint": "BuildFileWithStatsOpt", "artifact_path": "ram-contract-fixture", "status": "validated_by_pipeline", "validators": []any{"ramcontract.ValidateReport"}},
				map[string]any{"entrypoint": "buildObjectFileWithStatsOpt", "status": "formal_exemption_with_reason", "exemption": "not exercised by this linux-x64 RAM release fixture; object builds must carry their own RAM coverage evidence"},
				map[string]any{"entrypoint": "buildLibraryObjectWithStatsOpt", "status": "formal_exemption_with_reason", "exemption": "not exercised by this linux-x64 RAM release fixture; library builds must carry their own RAM coverage evidence"},
				map[string]any{"entrypoint": "InterfaceOnly", "status": "formal_exemption_with_reason", "exemption": "interface-only mode does not produce a RAM artifact in this release fixture"},
				map[string]any{"entrypoint": "wasm32-wasi-build", "status": "formal_exemption_with_reason", "exemption": "wasm32-wasi RAM coverage is target-specific and not claimed by this linux-x64 release fixture"},
				map[string]any{"entrypoint": "wasm32-web-build", "status": "formal_exemption_with_reason", "exemption": "wasm32-web RAM coverage is target-specific and not claimed by this linux-x64 release fixture"},
				map[string]any{"entrypoint": "explain-report-path", "status": "formal_exemption_with_reason", "exemption": "explain report path is not artifact-producing in this release fixture"},
			},
			"non_claims": []any{"pipeline coverage is not proof completeness"},
		}
	case "ram_heap_blockers":
		return map[string]any{
			"schema_version": schema,
			"kind":           "heap",
			"git_head":       gitHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"rows": []any{
				map[string]any{"site_id": "site:main:heap", "function": "main", "intent": "heap_fallback", "placement": "heap_unbounded", "blockers": []any{"unknown_size"}, "contract_grade": "M5"},
			},
			"non_claims": []any{"no Memory 100% claim"},
		}
	case "ram_copy_blockers":
		return map[string]any{
			"schema_version": schema,
			"kind":           "copy",
			"git_head":       gitHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"rows":           []any{},
			"non_claims":     []any{"no Memory 100% claim"},
		}
	case "ram_contract_fuzz_oracle":
		mutations := []string{
			"mutated_proof_id",
			"widened_grade",
			"missing_blocker",
			"budget_drift",
			"artifact_hash_drift",
			"forbidden_nonclaim_text",
		}
		var observations []any
		for _, mutation := range mutations {
			observations = append(observations, map[string]any{
				"mutation":          mutation,
				"rejected":          true,
				"validator":         "validate-" + strings.ReplaceAll(mutation, "_", "-"),
				"validator_command": "go run ./tools/cmd/validate-ram-contract-fuzz-oracle --test-fixture",
				"exit_code":         1,
				"output_excerpt":    "fixture rejected as expected",
				"mutated_file":      "mutations/" + mutation + "/ram-contract-report.json",
				"reason":            mutation + " rejected by validator with exit code 1",
			})
		}
		return map[string]any{
			"schema_version": schema,
			"git_head":       gitHead,
			"generated_at":   "2026-06-10T11:00:00Z",
			"observations":   observations,
			"summary":        map[string]any{"mutations": len(observations), "rejected": len(observations)},
			"non_claims":     []any{"not Memory 100%", "not a full formal proof", "not a performance benchmark"},
		}
	}
	return memory100JSON(schema, gitHead)
}

func memory100JSON(schema string, gitHead string) map[string]any {
	if strings.HasPrefix(schema, "tetra.release-artifact-hashes.") {
		return map[string]any{
			"schema":    schema,
			"root":      ".",
			"artifacts": []any{map[string]any{"path": "placeholder.json", "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size": 2}},
		}
	}
	switch schema {
	case "tetra.raw-memory-contract.v1":
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"operations": []any{
				map[string]any{"name": "core.alloc_bytes", "source_artifacts": []any{"compiler/internal/runtimeabi/raw_pointer_bounds_test.go", "memory-production/memory-production-linux-x64.json"}, "positive_tests": []any{"allocation-base metadata"}},
				map[string]any{"name": "core.ptr_add", "source_artifacts": []any{"compiler/internal/runtimeabi/raw_pointer_bounds_test.go", "memory-production/memory-production-linux-x64.json"}, "negative_tests": []any{"negative offset", "allocation upper bound", "access-width overflow"}},
				map[string]any{"name": "raw_slice_from_parts", "source_artifacts": []any{"compiler/internal/runtimeabi/raw_pointer_bounds_test.go", "compiler/tests/semantics/memory_ideal_v5_raw_pointer_test.go", "memory-production/memory-production-linux-x64.json"}, "negative_tests": []any{"outside unsafe", "negative length", "i32 byte overflow"}},
				map[string]any{"name": "raw_load_store_metadata", "source_artifacts": []any{"compiler/internal/plir/plir_test.go", "compiler/internal/lower/raw_memory_test.go", "compiler/internal/memoryfacts/from_plir_test.go", "memory-production/memory-production-linux-x64.json"}, "positive_tests": []any{"IRMemWriteI32Offset", "IRMemReadI32Offset", "core.store_u8/core.load_u8 raw memory gateway UnsafeChecked"}, "negative_tests": []any{"checked_external_unknown raw store/load remains conservative", "rejected_access_width_overflow raw load/store width rejection"}, "non_claims": []any{"no arbitrary external pointer safety claim"}},
				map[string]any{"name": "memcpy_u8", "source_artifacts": []any{"lib/core/memory.tetra", "compiler/internal/lower/raw_memory_test.go", "memory-production/memory-production-linux-x64.json"}, "positive_tests": []any{"cap.mem helper path"}, "negative_tests": []any{"negative length", "access-width overflow"}, "non_claims": []any{"no overlapping memcpy safety claim"}},
				map[string]any{"name": "memset_u8", "source_artifacts": []any{"lib/core/memory.tetra", "compiler/internal/lower/raw_memory_test.go", "memory-production/memory-production-linux-x64.json"}, "positive_tests": []any{"cap.mem helper path"}, "negative_tests": []any{"negative length", "access-width overflow"}},
				map[string]any{"name": "cap.mem", "source_artifacts": []any{"lib/core/capability.tetra", "compiler/internal/ramcontract/validate_test.go", "memory-production/memory-production-linux-x64.json"}, "negative_tests": []any{"unsafe_unknown promotion rejected", "cap.mem overclaim rejected"}, "non_claims": []any{"no arbitrary external pointer safety claim"}},
			},
		}
	case "tetra.allocation-lowering.v1":
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"decisions": []any{
				map[string]any{"name": "stack_trusted_no_escape", "status": "not_observed", "planned_storage": "Stack", "actual_lowering_storage": "Stack", "source_artifacts": []any{"ram-contract/ram-contract-report.json"}},
				map[string]any{"name": "heap_fallback_blocker", "status": "blocked", "planned_storage": "Heap", "actual_lowering_storage": "Heap", "blocker_artifact": "ram-contract/heap-blockers.json", "blocker_reason": "conservative heap fallback remains explicit until no-escape/lifetime proof is available", "budget_impact": "heap rows and budget bytes are accounted in ram-contract/memory-grade-report.json", "grade_impact": "heap fallback rows keep conservative RAM grade instead of trusted storage overclaim", "validator_coverage": []any{"validate-heap-blockers", "validate-ram-contract-release"}, "source_artifacts": []any{"ram-contract/heap-blockers.json", "ram-contract/ram-contract-report.json", "ram-contract/memory-grade-report.json"}, "covered_site_ids": []any{"site:main:heap"}},
				map[string]any{"name": "copy_blocker", "status": "not_observed", "planned_storage": "Copy", "actual_lowering_storage": "Copy", "source_artifacts": []any{"ram-contract/copy-blockers.json", "ram-contract/ram-contract-report.json"}},
				map[string]any{"name": "lowering_storage_match", "status": "not_observed", "planned_storage": "ExplicitIsland", "actual_lowering_storage": "ExplicitIsland", "source_artifacts": []any{"ram-contract/ram-contract-report.json"}},
			},
		}
	case "tetra.leak-resource.v1":
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"checks": []any{
				map[string]any{"name": "actornet_close_without_cancel", "kind": "stress", "evidence": "actornet broker close-without-cancel leak smoke", "source_artifacts": []any{"memory-production/memory-production-linux-x64.json"}},
				map[string]any{"name": "compiler_resource_finalization", "kind": "negative", "evidence": "compiler resource finalization diagnostics", "source_artifacts": []any{"memory-production/memory-production-linux-x64.json"}},
				map[string]any{"name": "surface_frame_escape", "kind": "negative", "evidence": "safe-view lifetime and Surface frame escape evidence", "source_artifacts": []any{"integrated/memory-islands-surface-production-manifest.json"}},
				map[string]any{"name": "actor_task_transfer", "kind": "negative", "evidence": "actor task transfer safety case", "source_artifacts": []any{"memory-production/memory-production-linux-x64.json"}},
			},
		}
	case "tetra.memory-semantic-safety-matrix.v1":
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"rows": []any{
				map[string]any{"name": "borrowed_view_return_escape", "kind": "negative", "evidence": "borrowed view escape via return is rejected", "source_artifacts": []any{"compiler/tests/ownership/ownership_test.go", "compiler/tests/semantics/borrow_escape_matrix_test.go"}, "tests": []any{"go test ./compiler/tests/ownership ./compiler/tests/semantics -run 'Borrow.*Return|BorrowEscape' -count=1"}},
				map[string]any{"name": "borrowed_view_owned_aggregate_escape", "kind": "negative", "evidence": "borrowed slice/ptr fields cannot escape through owned aggregate returns, consume, or inout calls", "source_artifacts": []any{"compiler/tests/ownership/ownership_test.go"}, "tests": []any{"go test ./compiler/tests/ownership -run 'Borrowed.*Aggregate|BorrowedSlice.*ConsumeInout|BorrowedPtr.*Struct' -count=1"}},
				map[string]any{"name": "borrowed_text_host_boundary_copy", "kind": "negative", "evidence": "borrowed text/view host and actor/task boundaries require explicit copy or are rejected", "source_artifacts": []any{"compiler/tests/semantics/string_view_test.go", "compiler/internal/actorsafety/sendability_test.go"}, "tests": []any{"go test ./compiler/tests/semantics ./compiler/internal/actorsafety -run 'Borrowed|StringView|copy|ActorBoundary' -count=1"}},
				map[string]any{"name": "inout_alias_escape", "kind": "negative", "evidence": "borrowed values cannot escape through inout assignment or aliasing", "source_artifacts": []any{"compiler/tests/ownership/ownership_test.go"}, "tests": []any{"go test ./compiler/tests/ownership -run 'Inout|Alias|BorrowedProjection' -count=1"}},
				map[string]any{"name": "surface_frame_escape", "kind": "negative", "evidence": "Surface frame/pixels borrowed views cannot escape lifecycle boundaries", "source_artifacts": []any{"compiler/tests/semantics/surface_stdlib_test.go", "integrated/safe-view-lifetime/safe-view-lifetime-summary.json"}, "tests": []any{"go test ./compiler/tests/semantics -run 'Surface|Frame|Pixels|SafeView' -count=1"}},
				map[string]any{"name": "use_after_present_close", "kind": "negative", "evidence": "use after present/close/free is rejected", "source_artifacts": []any{"compiler/tests/runtime/resource_finalization_test.go", "compiler/tests/ownership/ownership_test.go"}, "tests": []any{"go test ./compiler/tests/runtime ./compiler/tests/ownership -run 'Present|Close|UseAfter|Freed|Consume' -count=1"}},
				map[string]any{"name": "resource_finalizer_double_close", "kind": "negative", "evidence": "resource finalization diagnostics reject missing finalizer and double-close/double-free cases", "source_artifacts": []any{"compiler/tests/runtime/resource_finalization_test.go", "compiler/tests/safety/safety_diagnostics_test.go"}, "tests": []any{"go test ./compiler/tests/runtime ./compiler/tests/safety -run 'Resource|Finalization|Double|Close|Free' -count=1"}},
				map[string]any{"name": "actor_task_non_sendable_transfer", "kind": "negative", "evidence": "actor/task message transfer rejects borrowed or non-sendable memory/resources unless explicitly copied/moved", "source_artifacts": []any{"compiler/tests/ownership/actor_task_ownership_test.go", "compiler/internal/actorsafety/sendability_test.go", "compiler/internal/actorsafety/ownership_transfer_test.go"}, "tests": []any{"go test ./compiler/tests/ownership ./compiler/internal/actorsafety -run 'Actor|Task|Send|Transfer|Borrowed|NonSendable' -count=1"}},
			},
			"non_claims": []any{
				"no production actor runtime claim",
				"no universal leak-free program claim",
				"no full formal memory safety proof claim",
			},
		}
	case "tetra.proof-transition-report.v1":
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"rows": []any{
				map[string]any{
					"name":             "stable_hash_semantic_fields",
					"transition":       "invalidated",
					"evidence":         "StableHash includes semantic dominance/lifetime/epoch/invalidation/consumer fields and stale semantic mutation is blocked.",
					"consumer_action":  "blocked_by_proof_store_validate_recheck_required",
					"source_artifacts": []any{"compiler/internal/proof/term.go", "compiler/internal/proof/validate_test.go"},
					"tests":            []any{"go test ./compiler/internal/proof -run TestProofStoreRejectsStaleStableHashForSemanticFields -count=1"},
				},
				map[string]any{
					"name":             "bounds_proof_preserved_through_translation",
					"transition":       "preserved",
					"evidence":         "translation validation compares proof facts and preserves supported bounds proof evidence.",
					"before_artifact":  "compiler/translation_validation_v2.go",
					"after_artifact":   "compiler/translation_validation_v2_test.go",
					"source_artifacts": []any{"compiler/translation_validation_v2.go", "compiler/translation_validation_v2_test.go"},
					"tests":            []any{"go test ./compiler -run TestP23TranslationValidationV2CoversSupportedOptimizerSubset -count=1"},
				},
				map[string]any{
					"name":             "translation_missing_proof_requires_recheck",
					"transition":       "requires_recheck",
					"evidence":         "missing proof id after transform is rejected and requires recheck before unchecked use.",
					"consumer_action":  "recheck_or_block_unchecked_bounds_use",
					"source_artifacts": []any{"compiler/internal/validation/validation_test.go"},
					"tests":            []any{"go test ./compiler/internal/validation -run TestValidateTranslationRejectsMissingProofIDAfterTransform -count=1"},
				},
				map[string]any{
					"name":             "optimization_invalidates_bounds_proofs",
					"transition":       "invalidated",
					"evidence":         "optimizer proof rules require invalidated bounds facts to be declared, and consumers must recheck before reuse.",
					"consumer_action":  "recheck_required_before_consuming_invalidated_bounds_proof",
					"source_artifacts": []any{"compiler/internal/opt/manager.go", "compiler/internal/opt/manager_test.go"},
					"tests":            []any{"go test ./compiler/internal/opt -run 'Manager|Optimization' -count=1"},
				},
				map[string]any{
					"name":             "lowering_refines_bounds_proof_use",
					"transition":       "refined",
					"evidence":         "lowering refines live bounds proof use into proof-tagged unchecked load metadata.",
					"before_artifact":  "compiler/internal/plir/plir_test.go",
					"after_artifact":   "compiler/internal/lower/proof_bce_test.go",
					"source_artifacts": []any{"compiler/internal/plir/plir_test.go", "compiler/internal/lower/proof_bce_test.go"},
					"tests":            []any{"go test ./compiler/internal/plir ./compiler/internal/lower -run 'Proof|Invalidates|Unchecked' -count=1"},
				},
				map[string]any{
					"name":             "new_proof_requires_store_reference",
					"transition":       "new",
					"evidence":         "new proof use requires a proof store reference and unknown proof ids are blocked.",
					"after_artifact":   "compiler/internal/validation/validation_test.go",
					"source_artifacts": []any{"compiler/internal/validation/validation_test.go", "compiler/internal/proof/validate_test.go"},
					"tests":            []any{"go test ./compiler/internal/validation ./compiler/internal/proof -run 'UnknownLiveProof|MissingProofID|ProofStoreRejectsMissingProofID' -count=1"},
				},
			},
			"non_claims": []any{
				"no full formal proof claim",
				"no exhaustive optimizer proof-transition completeness claim",
			},
		}
	case "tetra.runtime-memory-contract.v1":
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"rows": []any{
				map[string]any{
					"target":                              "linux-x64",
					"included_in_memory100_target_matrix": true,
					"runtime_status":                      "production",
					"memory_run":                          "yes",
					"memory_claim_level":                  "production_host_runtime",
					"evidence":                            "linux-x64 runtime hardening and runtimeabi memory evidence is covered by Memory100 gate.",
					"source_artifacts":                    []any{"memory-production/targets.json", "compiler/runtime_hardening_v1.go", "compiler/internal/runtimeabi/runtimeabi_test.go"},
					"tests":                               []any{"go test ./compiler -run 'RuntimeHardening|RuntimeAllocation|RawPointerBoundsABI|ActorRuntimeProductionBoundary|OOM|Stack|Allocator|Region' -count=1"},
					"non_claims":                          []any{"no all-target memory parity claim", "no full runtime-hardening proof claim"},
				},
				map[string]any{
					"target":                              "windows-x64",
					"included_in_memory100_target_matrix": false,
					"runtime_status":                      "host_required",
					"memory_run":                          "host-required",
					"memory_claim_level":                  "host_required_nonclaim",
					"evidence":                            "windows-x64 requires target-host runtime evidence before Memory100 inclusion.",
					"excluded_reason":                     "no windows target-host runtime memory report in this aggregate",
					"source_artifacts":                    []any{"memory-production/targets.json"},
					"tests":                               []any{"go test ./tools/cmd/validate-memory-100-prod-stable -run RuntimeMemory -count=1"},
					"non_claims":                          []any{"no windows-x64 runtime memory production claim"},
				},
				map[string]any{
					"target":                              "macos-x64",
					"included_in_memory100_target_matrix": false,
					"runtime_status":                      "host_required",
					"memory_run":                          "host-required",
					"memory_claim_level":                  "host_required_nonclaim",
					"evidence":                            "macos-x64 requires target-host runtime evidence before Memory100 inclusion.",
					"excluded_reason":                     "no macos target-host runtime memory report in this aggregate",
					"source_artifacts":                    []any{"memory-production/targets.json"},
					"tests":                               []any{"go test ./tools/cmd/validate-memory-100-prod-stable -run RuntimeMemory -count=1"},
					"non_claims":                          []any{"no macos-x64 runtime memory production claim"},
				},
				map[string]any{
					"target":                              "wasm32-wasi",
					"included_in_memory100_target_matrix": false,
					"runtime_status":                      "tiered",
					"memory_run":                          "runner-smoke if available",
					"memory_claim_level":                  "artifact_runtime_tiered_nonclaim",
					"evidence":                            "wasm32-wasi remains artifact/runtime tiered and is not Memory100 production-host-runtime evidence.",
					"excluded_reason":                     "not part of current Memory100 linux-x64 target matrix",
					"source_artifacts":                    []any{"memory-production/targets.json"},
					"tests":                               []any{"go test ./tools/cmd/validate-memory-100-prod-stable -run RuntimeMemory -count=1"},
					"non_claims":                          []any{"no wasm32-wasi production host-runtime memory claim"},
				},
				map[string]any{
					"target":                              "wasm32-web",
					"included_in_memory100_target_matrix": false,
					"runtime_status":                      "tiered",
					"memory_run":                          "browser-smoke if available",
					"memory_claim_level":                  "artifact_runtime_tiered_nonclaim",
					"evidence":                            "wasm32-web remains artifact/runtime tiered and is not Memory100 production-host-runtime evidence.",
					"excluded_reason":                     "not part of current Memory100 linux-x64 target matrix",
					"source_artifacts":                    []any{"memory-production/targets.json"},
					"tests":                               []any{"go test ./tools/cmd/validate-memory-100-prod-stable -run RuntimeMemory -count=1"},
					"non_claims":                          []any{"no wasm32-web production host-runtime memory claim"},
				},
				map[string]any{
					"target":                              "linux-x86",
					"included_in_memory100_target_matrix": false,
					"runtime_status":                      "partial_build_only",
					"memory_run":                          "no/host-dependent",
					"memory_claim_level":                  "build_lower_only_nonclaim",
					"evidence":                            "linux-x86 remains build/lower scoped for Memory100 and is not production runtime evidence.",
					"excluded_reason":                     "build/lower-only memory evidence is not Memory100 production-host-runtime evidence",
					"source_artifacts":                    []any{"memory-production/targets.json"},
					"tests":                               []any{"go test ./tools/cmd/validate-memory-100-prod-stable -run RuntimeMemory -count=1"},
					"non_claims":                          []any{"no linux-x86 production runtime memory claim"},
				},
				map[string]any{
					"target":                              "linux-x32",
					"included_in_memory100_target_matrix": false,
					"runtime_status":                      "partial_build_only",
					"memory_run":                          "no/host-dependent",
					"memory_claim_level":                  "build_lower_only_nonclaim",
					"evidence":                            "linux-x32 remains build/lower scoped for Memory100 and is not production runtime evidence.",
					"excluded_reason":                     "build/lower-only memory evidence is not Memory100 production-host-runtime evidence",
					"source_artifacts":                    []any{"memory-production/targets.json"},
					"tests":                               []any{"go test ./tools/cmd/validate-memory-100-prod-stable -run RuntimeMemory -count=1"},
					"non_claims":                          []any{"no linux-x32 production runtime memory claim"},
				},
			},
			"non_claims": []any{
				"no all-target memory parity claim",
				"OOM recovery guarantee is not claimed",
				"full stack-overflow protection is not claimed",
				"full allocator-corruption detection proof is not claimed",
				"production actor runtime is not claimed",
			},
		}
	case "tetra.memory-100.claim-policy.v1":
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"allowed_claims": []any{
				"Memory/RAM production-stable criteria passed locally for the scoped target matrix only.",
			},
			"forbidden_claims": []any{
				"Memory is 100% ready",
				"fully proven memory safety",
				"full formal proof of memory safety",
				"all targets memory-stable",
				"all-target memory parity",
				"unsafe/raw memory is safe",
				"no leaks",
			},
			"non_claims": []any{
				"no universal Memory 100% claim",
				"no full formal proof claim",
			},
		}
	}
	return memory100PlaceholderJSON(schema, gitHead)
}

func writeMemory100MemoryFuzzBundle(t *testing.T, dir string, gitHead string) {
	t.Helper()
	writeMemory100JSON(t, filepath.Join(dir, "memory-fuzz-oracle.json"), map[string]any{
		"schema_version": "tetra.memory-fuzz.oracle.v1",
		"scope":          "memory-production-core-v1",
		"status":         "pass",
		"tier":           "Tier 1 short CI smoke",
		"git_head":       gitHead,
	})
	if err := os.WriteFile(filepath.Join(dir, "summary.md"), []byte("# Memory Fuzz Short Summary\n\n- tier: `Tier 1 short CI smoke`\n- report: `memory-fuzz-oracle.json`\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeMemory100FuzzSummary(t, dir, gitHead, dir)
	writeMemory100JSON(t, filepath.Join(dir, "island-proof-fuzz-summary.json"), map[string]any{
		"schema_version": "tetra.island-proof-fuzz-summary.v1",
		"status":         "pass",
		"total":          11,
		"rejected":       11,
		"accepted":       0,
		"cases": []any{
			map[string]any{"name": "malformed_proof_json", "status": "rejected"},
			map[string]any{"name": "stale_epoch", "status": "rejected"},
			map[string]any{"name": "mismatched_island_id", "status": "rejected"},
			map[string]any{"name": "wrong_base_allocation", "status": "rejected"},
			map[string]any{"name": "broken_dominance", "status": "rejected"},
			map[string]any{"name": "missing_proof_id", "status": "rejected"},
			map[string]any{"name": "wrong_operation", "status": "rejected"},
			map[string]any{"name": "unsafe_unknown_promotion", "status": "rejected"},
			map[string]any{"name": "noalias_broad_proof", "status": "rejected"},
			map[string]any{"name": "storage_heap_fallback", "status": "rejected"},
			map[string]any{"name": "transform_lost_metadata", "status": "rejected"},
		},
	})
	writeMemory100MemoryFuzzReproducerDirs(t, dir)
	writeMemory100HashManifest(t, dir)
}

func writeMemory100MemoryFuzzReproducerDirs(t *testing.T, dir string) {
	t.Helper()
	for _, rel := range []string{"reproducers/compiler-crash", "reproducers/miscompile", "reducers/miscompile"} {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("create memory fuzz evidence dir %s: %v", rel, err)
		}
		if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("required release evidence slot for "+rel+"\n"), 0o644); err != nil {
			t.Fatalf("write memory fuzz evidence marker %s: %v", rel, err)
		}
	}
}

func writeMemory100MemoryReleaseManifest(t *testing.T, dir string, gitHead string) {
	t.Helper()
	writeMemory100JSON(t, filepath.Join(dir, "memory-release-manifest.json"), map[string]any{
		"schema":        "tetra.memory.release-manifest.v1",
		"target":        "linux-x64",
		"git_head":      gitHead,
		"generated_at":  "2026-06-10T11:00:00Z",
		"report_dir":    ".",
		"hash_manifest": "artifact-hashes.json",
		"commands":      memory100MemoryReleaseCommandsForDir(filepath.ToSlash(dir), gitHead),
		"artifacts":     memory100MemoryReleaseArtifactRefsForDir(filepath.ToSlash(dir), gitHead),
	})
}

func writeMemory100TargetReport(t *testing.T, dir string) {
	t.Helper()
	writeMemory100JSON(t, filepath.Join(dir, "targets.json"), map[string]any{
		"supported":  []any{"linux-x64", "windows-x64", "macos-x64", "wasm32-wasi", "wasm32-web"},
		"build_only": []any{"linux-x86", "linux-x32"},
		"planned":    []any{},
		"targets": []any{
			memory100TargetReportRow("linux-x64", "supported", "linux", "x64", "sysv", "elf", "", false, "host_native", true, true, map[string]any{
				"run_supported":              true,
				"runtime_status":             "production",
				"stdlib_status":              "production",
				"ffi_status":                 "scalar_object_smokes_partial",
				"memory_build":               "yes",
				"memory_lower":               "yes",
				"memory_run":                 "yes",
				"memory_raw_diagnostics":     "yes",
				"memory_region_lowering":     "yes/partial",
				"memory_alignment_semantics": "yes",
				"memory_claim_level":         "production/host_runtime",
				"runner_probe_command":       "tetra test --target x64 --format=json <runner-smoke.tetra>",
				"release_gate":               "scripts/release/post_v0_4/linux-native-targets-smoke.sh",
				"evidence_artifacts":         []any{"targets.json", "linux-x64-abi.json", "linux-x64-atomic-stress.json", "linux-x64-fuzz.json", "linux-x64-runner.json", "linux-native-targets-brutal.json", "artifact-hashes.json"},
				"syscall_instruction":        "syscall",
				"syscall_numbering":          "x86_64",
				"syscall_arg_registers":      []any{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"},
				"syscall_error_range":        "-4095..-1",
			}),
			memory100TargetReportRow("windows-x64", "supported", "windows", "x64", "win64", "pe", ".exe", false, "host_native", true, true, map[string]any{
				"run_supported":              false,
				"run_unsupported_reason":     "windows-x64 cannot run on host linux/amd64",
				"memory_build":               "yes",
				"memory_lower":               "yes",
				"memory_run":                 "host-required",
				"memory_raw_diagnostics":     "host-required",
				"memory_region_lowering":     "host-required",
				"memory_alignment_semantics": "host-required",
				"memory_claim_level":         "build_lower_only unless run",
			}),
			memory100TargetReportRow("macos-x64", "supported", "macos", "x64", "sysv", "macho", "", false, "host_native", true, true, map[string]any{
				"run_supported":              false,
				"run_unsupported_reason":     "macos-x64 cannot run on host linux/amd64",
				"memory_build":               "yes",
				"memory_lower":               "yes",
				"memory_run":                 "host-required",
				"memory_raw_diagnostics":     "host-required",
				"memory_region_lowering":     "host-required",
				"memory_alignment_semantics": "host-required",
				"memory_claim_level":         "build_lower_only unless run",
			}),
			memory100TargetReportRow("wasm32-wasi", "supported", "wasi", "wasm32", "wasi", "wasm", ".wasm", false, "wasi_runner", false, true, map[string]any{
				"run_supported":              true,
				"run_runner":                 "wasmtime",
				"memory_build":               "yes",
				"memory_lower":               "yes",
				"memory_run":                 "runner-smoke if available",
				"memory_raw_diagnostics":     "safe-only",
				"memory_region_lowering":     "limited",
				"memory_alignment_semantics": "wasm rules",
				"memory_claim_level":         "artifact/runtime tiered",
			}),
			memory100TargetReportRow("wasm32-web", "supported", "web", "wasm32", "web", "wasm", ".wasm", false, "web_runner", false, true, map[string]any{
				"run_supported":              true,
				"run_runner":                 "browser",
				"memory_build":               "yes",
				"memory_lower":               "yes",
				"memory_run":                 "browser-smoke if available",
				"memory_raw_diagnostics":     "safe-only",
				"memory_region_lowering":     "limited",
				"memory_alignment_semantics": "wasm rules",
				"memory_claim_level":         "artifact/runtime tiered",
			}),
			memory100TargetReportRow("linux-x86", "build_only", "linux", "x86", "i386-sysv", "elf", "", true, "host_probed", false, false, map[string]any{
				"run_supported":              true,
				"runtime_status":             "partial_build_only",
				"stdlib_status":              "partial_build_only",
				"ffi_status":                 "ilp32_scalar_object_smokes_partial",
				"memory_build":               "yes",
				"memory_lower":               "yes",
				"memory_run":                 "no/host-dependent",
				"memory_raw_diagnostics":     "partial",
				"memory_region_lowering":     "partial",
				"memory_alignment_semantics": "partial",
				"memory_claim_level":         "build_lower_only",
				"runner_probe_command":       "tetra test --diagnostics=json --target x86 --format=json <runner-smoke.tetra>",
				"release_gate":               "scripts/release/post_v0_4/linux-native-targets-smoke.sh",
				"evidence_artifacts":         []any{"targets.json", "linux-x86-abi.json", "linux-x86-atomic-stress.json", "linux-x86-fuzz.json", "linux-x86-runner.json", "linux-native-targets-brutal.json", "artifact-hashes.json"},
				"syscall_instruction":        "int 0x80",
				"syscall_numbering":          "i386",
				"syscall_arg_registers":      []any{"eax", "ebx", "ecx", "edx", "esi", "edi", "ebp"},
				"syscall_error_range":        "-4095..-1",
			}),
			memory100TargetReportRow("linux-x32", "build_only", "linux", "x64", "x32-sysv", "elf", "", true, "host_probed", false, false, map[string]any{
				"run_supported":              true,
				"runtime_status":             "partial_build_only",
				"stdlib_status":              "partial_build_only",
				"ffi_status":                 "ilp32_scalar_object_smokes_partial",
				"memory_build":               "yes",
				"memory_lower":               "yes",
				"memory_run":                 "no/host-dependent",
				"memory_raw_diagnostics":     "partial",
				"memory_region_lowering":     "partial",
				"memory_alignment_semantics": "special",
				"memory_claim_level":         "build_lower_only",
				"runner_probe_command":       "tetra test --diagnostics=json --target x32 --format=json <runner-smoke.tetra>",
				"release_gate":               "scripts/release/post_v0_4/linux-native-targets-smoke.sh",
				"evidence_artifacts":         []any{"targets.json", "linux-x32-abi.json", "linux-x32-atomic-stress.json", "linux-x32-fuzz.json", "linux-x32-runner.json", "linux-native-targets-brutal.json", "artifact-hashes.json"},
				"syscall_instruction":        "syscall",
				"syscall_numbering":          "x32_syscall_bit",
				"syscall_arg_registers":      []any{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"},
				"syscall_error_range":        "-4095..-1",
			}),
		},
	})
}

func memory100TargetReportRow(triple string, status string, osName string, arch string, abi string, format string, exeExt string, buildOnly bool, runMode string, supportsDebug bool, supportsRelease bool, extra map[string]any) map[string]any {
	row := map[string]any{
		"triple":                    triple,
		"status":                    status,
		"os":                        osName,
		"arch":                      arch,
		"abi":                       abi,
		"format":                    format,
		"exe_ext":                   exeExt,
		"build_only":                buildOnly,
		"run_mode":                  runMode,
		"supports_debug_info":       supportsDebug,
		"supports_release_optimize": supportsRelease,
	}
	for key, value := range extra {
		row[key] = value
	}
	return row
}

func memory100MemoryReleaseCommandsForDir(dir string, gitHead string) []any {
	return []any{
		map[string]any{"name": "memory-production-smoke", "command": "go run ./tools/cmd/memory-production-smoke --report " + dir + "/memory-production-linux-x64.json --git-head " + gitHead},
		map[string]any{"name": "target-report", "command": "go run ./cli/cmd/tetra targets --format=json > " + dir + "/targets.json"},
		map[string]any{"name": "validate-targets", "command": "go run ./tools/cmd/validate-targets --report " + dir + "/targets.json"},
		map[string]any{"name": "memory-fuzz-short", "command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir " + dir + "/memory-fuzz-tier1 --git-head " + gitHead},
		map[string]any{"name": "validate-memory-fuzz-oracle", "command": "go run ./tools/cmd/validate-memory-fuzz-oracle --report " + dir + "/memory-fuzz-tier1/memory-fuzz-oracle.json --artifact-dir " + dir + "/memory-fuzz-tier1 --current-git-head " + gitHead},
		map[string]any{"name": "ram-contract-gate", "command": "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir " + dir + "/ram-contract"},
		map[string]any{"name": "island-proof-verifier", "command": "go run ./tools/cmd/validate-island-proof --proof " + dir + "/island-proof-verifier.json --memory-report " + dir + "/island-proof-memory-report.json --current-git-head " + gitHead + " --require-same-commit"},
		map[string]any{"name": "artifact-hashes-write", "command": "go run ./tools/cmd/validate-artifact-hashes --write --root " + dir + " --out " + dir + "/artifact-hashes.json"},
		map[string]any{"name": "artifact-hashes-validate", "command": "go run ./tools/cmd/validate-artifact-hashes --manifest " + dir + "/artifact-hashes.json"},
	}
}

func memory100MemoryReleaseArtifactRefsForDir(dir string, gitHead string) []any {
	var artifacts []any
	for _, required := range requiredMemory100MemoryReleaseArtifacts {
		command := memory100MemoryReleaseArtifactCommand(dir, required.Kind, gitHead)
		artifacts = append(artifacts, map[string]any{
			"path":    required.Path,
			"kind":    required.Kind,
			"schema":  required.Schema,
			"target":  "linux-x64",
			"command": command,
		})
	}
	return artifacts
}

func memory100MemoryReleaseArtifactCommand(dir string, kind string, gitHead string) string {
	switch kind {
	case "memory_production_report":
		return "go run ./tools/cmd/memory-production-smoke --report " + dir + "/memory-production-linux-x64.json --git-head " + gitHead
	case "target_report":
		return "go run ./cli/cmd/tetra targets --format=json > " + dir + "/targets.json"
	case "memory_fuzz_oracle_report", "memory_fuzz_summary", "memory_fuzz_island_proof_summary":
		return "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir " + dir + "/memory-fuzz-tier1 --git-head " + gitHead
	case "island_proof_verifier_report", "island_proof_memory_report":
		return "go run ./tools/cmd/validate-island-proof --proof " + dir + "/island-proof-verifier.json --memory-report " + dir + "/island-proof-memory-report.json --current-git-head " + gitHead + " --require-same-commit"
	case "artifact_hash_manifest":
		return "go run ./tools/cmd/validate-artifact-hashes --write --root " + dir + " --out " + dir + "/artifact-hashes.json"
	default:
		return "bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh --report-dir " + dir + "/ram-contract"
	}
}

func writeMemory100RAMContractReleaseManifest(t *testing.T, dir string, gitHead string) {
	t.Helper()
	reportPath := func(rel string) string {
		if rel == "" {
			return filepath.ToSlash(dir)
		}
		return filepath.ToSlash(filepath.Join(dir, filepath.FromSlash(rel)))
	}
	commands := []any{
		map[string]any{"name": "ram-contract-build", "command": "go run ./cli/cmd/tetra build --target linux-x64 --emit-ram-contract-report --emit-memory-report --emit-alloc-report -o fixture.o fixture.tetra"},
		map[string]any{"name": "validate-ram-contract-report", "command": "go run ./tools/cmd/validate-ram-contract-report --report " + reportPath("ram-contract-report.json")},
		map[string]any{"name": "validate-memory-grade-report", "command": "go run ./tools/cmd/validate-memory-grade-report --report " + reportPath("memory-grade-report.json")},
		map[string]any{"name": "validate-proof-store-summary", "command": "go run ./tools/cmd/validate-proof-store-summary --report " + reportPath("proof-store-summary.json")},
		map[string]any{"name": "validate-validation-pipeline-coverage", "command": "go run ./tools/cmd/validate-validation-pipeline-coverage --report " + reportPath("validation-pipeline-coverage.json")},
		map[string]any{"name": "validate-heap-blockers", "command": "go run ./tools/cmd/validate-heap-blockers --report " + reportPath("heap-blockers.json")},
		map[string]any{"name": "validate-copy-blockers", "command": "go run ./tools/cmd/validate-copy-blockers --report " + reportPath("copy-blockers.json")},
		map[string]any{"name": "ram-contract-fuzz-short", "command": "go run ./tools/cmd/ram-contract-fuzz-short --report-dir " + reportPath("fuzz") + " --git-head " + gitHead},
		map[string]any{"name": "validate-ram-contract-fuzz-oracle", "command": "go run ./tools/cmd/validate-ram-contract-fuzz-oracle --report " + reportPath("fuzz/ram-contract-fuzz-oracle.json") + " --current-git-head " + gitHead},
		map[string]any{"name": "artifact-hashes-write", "command": "go run ./tools/cmd/validate-artifact-hashes --write --root " + reportPath("") + " --out " + reportPath("artifact-hashes.json")},
		map[string]any{"name": "artifact-hashes-validate", "command": "go run ./tools/cmd/validate-artifact-hashes --manifest " + reportPath("artifact-hashes.json")},
		map[string]any{"name": "ram-contract-release-validator", "command": "go run ./tools/cmd/validate-ram-contract-release --report-dir " + reportPath("") + " --current-git-head " + gitHead},
	}
	var artifacts []any
	for _, required := range requiredMemory100RAMContractReleaseArtifacts {
		artifacts = append(artifacts, map[string]any{
			"path":   required.Path,
			"kind":   required.Kind,
			"schema": required.Schema,
		})
	}
	writeMemory100JSON(t, filepath.Join(dir, "ram-contract-release-manifest.json"), map[string]any{
		"schema":        "tetra.ram-contract.release-manifest.v1",
		"status":        "pass",
		"target":        "linux-x64",
		"git_head":      gitHead,
		"generated_at":  "2026-06-10T11:00:00Z",
		"report_dir":    ".",
		"hash_manifest": "artifact-hashes.json",
		"commands":      commands,
		"artifacts":     artifacts,
		"non_claims": []any{
			"no Memory 100% claim",
			"no full formal proof claim",
			"no official benchmark or fastest-language claim",
			"local Linux-x64 scoped RAM contract evidence only",
		},
	})
}

func writeMemory100IntegratedBundle(t *testing.T, dir string, gitHead string) {
	t.Helper()
	memoryDir := filepath.Join(dir, "memory")
	writeMemory100JSON(t, filepath.Join(memoryDir, "memory-production-linux-x64.json"), memory100ArtifactJSON("memory_production_report", "tetra.memory.production.v1", gitHead))
	writeMemory100MemoryReleaseManifest(t, memoryDir, gitHead)
	writeMemory100TargetReport(t, memoryDir)
	writeMemory100IntegratedIslandProofEvidence(t, memoryDir, gitHead)
	writeMemory100MemoryFuzzBundle(t, filepath.Join(memoryDir, "memory-fuzz-tier1"), gitHead)

	ramDir := filepath.Join(memoryDir, "ram-contract")
	writeMemory100RAMContractArtifacts(t, ramDir, gitHead)
	writeMemory100RAMContractReleaseManifest(t, ramDir, gitHead)
	writeMemory100HashManifest(t, ramDir)
	writeMemory100HashManifest(t, memoryDir)

	writeMemory100JSON(t, filepath.Join(dir, "islands-debug-smoke.json"), map[string]any{
		"schema":        "tetra.release.v0_2_0.smoke-report.v1",
		"target":        "linux-x64",
		"git_head":      gitHead,
		"islands_debug": true,
		"total":         1,
		"passed":        1,
		"failed":        0,
		"cases": []any{
			map[string]any{"name": "islands_overflow", "src_path": "examples/islands_overflow.tetra", "expected_exit": 1, "actual_exit": 1, "ran": true, "pass": true},
		},
	})
	writeMemory100JSON(t, filepath.Join(dir, "surface-release-v1", "surface-release-summary.json"), map[string]any{
		"schema":        "tetra.surface.release.v1",
		"status":        "current",
		"git_head":      gitHead,
		"release_scope": "surface-v1-linux-web",
	})
	writeMemory100HashManifest(t, filepath.Join(dir, "surface-release-v1"))
	writeMemory100JSON(t, filepath.Join(dir, "surface-experimental-regression", "summary.json"), map[string]any{
		"schema":   "tetra.surface.experimental-regression.v1",
		"status":   "pass",
		"git_head": gitHead,
	})
	writeMemory100HashManifest(t, filepath.Join(dir, "surface-experimental-regression"))
	writeMemory100JSON(t, filepath.Join(dir, "safe-view-lifetime", "safe-view-lifetime-summary.json"), map[string]any{
		"schema":           "tetra.safe-view-lifetime.gate.v1",
		"status":           "pass",
		"bounded":          true,
		"release_blocking": true,
	})
	writeMemory100JSON(t, filepath.Join(dir, "surface-api-stability-v1", "surface-api-stability-summary.json"), map[string]any{
		"schema":                  "tetra.surface.api-stability.v1",
		"status":                  "pass",
		"release_scope":           "surface-v1-linux-web",
		"docs_manifest_validated": true,
	})

	writeMemory100JSON(t, filepath.Join(dir, "memory-islands-surface-production-manifest.json"), map[string]any{
		"schema":        "tetra.memory-islands-surface.production-gate.v1",
		"status":        "pass",
		"git_head":      gitHead,
		"generated_at":  "2026-06-10T11:00:00Z",
		"report_dir":    ".",
		"hash_manifest": "artifact-hashes.json",
		"commands":      memory100IntegratedCommandsForDir(filepath.ToSlash(dir), gitHead),
		"artifacts":     memory100IntegratedArtifactRefs(),
	})
	writeMemory100HashManifest(t, dir)
}

func writeMemory100RAMContractArtifacts(t *testing.T, dir string, gitHead string) {
	t.Helper()
	ramArtifacts := []struct {
		Path   string
		Kind   string
		Schema string
	}{
		{"ram-contract-release-manifest.json", "ram_contract_release_manifest", "tetra.ram-contract.release-manifest.v1"},
		{"ram-contract-report.json", "ram_contract_report", "tetra.ram-contract-report.v1"},
		{"memory-grade-report.json", "ram_memory_grade_report", "tetra.memory-grade-report.v1"},
		{"proof-store-summary.json", "ram_proof_store_summary", "tetra.proof-store-summary.v1"},
		{"validation-pipeline-coverage.json", "ram_validation_pipeline_coverage", "tetra.validation-pipeline-coverage.v1"},
		{"heap-blockers.json", "ram_heap_blockers", "tetra.ram-blockers.v1"},
		{"copy-blockers.json", "ram_copy_blockers", "tetra.ram-blockers.v1"},
		{"fuzz/ram-contract-fuzz-oracle.json", "ram_contract_fuzz_oracle", "tetra.ram-contract-fuzz-oracle.v1"},
	}
	for _, artifact := range ramArtifacts {
		writeMemory100JSON(t, filepath.Join(dir, filepath.FromSlash(artifact.Path)), memory100ArtifactJSON(artifact.Kind, artifact.Schema, gitHead))
	}
}

func memory100IntegratedCommandsForDir(dir string, gitHead string) []any {
	return []any{
		map[string]any{"name": "memory-production-gate", "command": "bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir " + dir + "/memory"},
		map[string]any{"name": "islands-debug-smoke", "command": "go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --islands-debug --report " + dir + "/islands-debug-smoke.json"},
		map[string]any{"name": "validate-islands-debug-smoke", "command": "go run ./tools/cmd/smoke-report-to-checklist --validate-only --report " + dir + "/islands-debug-smoke.json"},
		map[string]any{"name": "surface-release-gate", "command": "bash scripts/release/surface/release-gate.sh --report-dir " + dir + "/surface-release-v1"},
		map[string]any{"name": "surface-experimental-regression-gate", "command": "bash scripts/release/surface/gate.sh --report-dir " + dir + "/surface-experimental-regression"},
		map[string]any{"name": "safe-view-lifetime-gate", "command": "bash scripts/release/safe-view-lifetime/gate.sh --report-dir " + dir + "/safe-view-lifetime"},
		map[string]any{"name": "surface-api-stability-gate", "command": "bash scripts/release/surface/api-stability-gate.sh --report-dir " + dir + "/surface-api-stability-v1"},
		map[string]any{"name": "validate-manifest", "command": "go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json"},
		map[string]any{"name": "verify-docs", "command": "go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json"},
		map[string]any{"name": "artifact-hashes-write", "command": "go run ./tools/cmd/validate-artifact-hashes --write --root " + dir + " --out " + dir + "/artifact-hashes.json"},
		map[string]any{"name": "artifact-hashes-validate", "command": "go run ./tools/cmd/validate-artifact-hashes --manifest " + dir + "/artifact-hashes.json"},
		map[string]any{"name": "integrated-release-validator", "command": "go run ./tools/cmd/validate-memory-islands-surface-production --report-dir " + dir + " --current-git-head " + gitHead},
	}
}

func memory100IntegratedArtifactRefs() []any {
	var artifacts []any
	for _, required := range requiredMemory100IntegratedArtifacts {
		artifacts = append(artifacts, map[string]any{
			"path":   required.Path,
			"kind":   required.Kind,
			"schema": required.Schema,
		})
	}
	return artifacts
}

func writeMemory100IntegratedIslandProofEvidence(t *testing.T, memoryDir string, gitHead string) {
	t.Helper()
	writeMemory100JSON(t, filepath.Join(memoryDir, "island-proof-verifier.json"), map[string]any{
		"schema":           "tetra.island.proof.v1",
		"producer":         "tools/validators/islandproof/test-fixture",
		"producer_command": "go run ./tools/cmd/validate-island-proof",
		"git_head":         gitHead,
		"generated_at":     "2026-06-10T11:00:00Z",
		"proofs": []any{
			map[string]any{
				"proof_id":                "proof:test:island:borrow:1",
				"operation":               "island_borrow",
				"proof_kind":              "island_epoch",
				"subject_base_id":         "alloc:test:island:0",
				"island_id":               "island:test:0",
				"epoch":                   1,
				"source_fact_id":          "fact:test:island-proof:1",
				"claim":                   "island_proof_verified",
				"provenance_class":        "safe_known",
				"unsafe_class":            "safe",
				"validator_name":          "validate-island-proof",
				"validator_status":        "pass",
				"planned_storage":         "ExplicitIsland",
				"actual_lowering_storage": "ExplicitIsland",
				"dominance":               "entry dominates test island borrow",
				"distinct_live_islands":   []any{"island:test:0", "island:test:1"},
			},
		},
	})
	writeMemory100JSON(t, filepath.Join(memoryDir, "island-proof-memory-report.json"), map[string]any{
		"schema_version": "tetra.memory-report.v1",
		"rows": []any{
			map[string]any{
				"site_id":                 "island:test:borrow:1",
				"source_fact_id":          "fact:test:island-proof:1",
				"claim":                   "island_proof_verified",
				"claim_level":             "validated",
				"provenance_class":        "safe_known",
				"unsafe_class":            "safe",
				"alias_state":             "unique",
				"island_id":               "island:test:0",
				"epoch":                   1,
				"base_id":                 "alloc:test:island:0",
				"proof_id":                "proof:test:island:borrow:1",
				"proof_kind":              "island_epoch",
				"proof_subject_base_id":   "alloc:test:island:0",
				"proof_operation":         "island_borrow",
				"planned_storage":         "ExplicitIsland",
				"actual_lowering_storage": "ExplicitIsland",
				"validator_name":          "validate-island-proof",
				"validator_status":        "pass",
			},
		},
	})
}

func writeMemory100FuzzSummary(t *testing.T, dir string, gitHead string, commandDir string) {
	t.Helper()
	commandDir = filepath.ToSlash(commandDir)
	writeMemory100JSON(t, filepath.Join(dir, "summary.json"), map[string]any{
		"schema_version":            "tetra.memory-fuzz-short.summary.v1",
		"kind":                      "tier1_short_ci_smoke",
		"tier":                      "tier1_short_ci_smoke",
		"status":                    "pass",
		"observed_failures":         0,
		"classified_failures":       0,
		"unclassified_failures":     0,
		"release_blocking_failures": 0,
		"reproducibility_seeds": []string{
			"memory-fuzz:v0:seed:1000",
			"memory-fuzz:v1:seed:1001",
			"memory-fuzz:v2:seed:1002",
			"memory-fuzz:v3:seed:1003",
			"memory-fuzz:v4:seed:1004",
			"memory-fuzz:v5:seed:1005",
			"memory-fuzz:v6:seed:1006",
			"memory-fuzz:v7:seed:1007",
			"memory-fuzz:v8:seed:1008",
			"memory-fuzz:v9:seed:1009",
			"memory-fuzz:v10:seed:1010",
			"memory-fuzz:v11:seed:1011",
		},
		"artifacts": map[string]any{
			"artifact_hashes":           "artifact-hashes.json",
			"island_proof_fuzz_summary": "island-proof-fuzz-summary.json",
			"oracle_report":             "memory-fuzz-oracle.json",
			"summary_md":                "summary.md",
			"summary_json":              "summary.json",
		},
		"commands": []any{
			map[string]any{"name": "memory-fuzz-short", "command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir " + commandDir + " --git-head " + gitHead, "status": "pass"},
			map[string]any{"name": "validate-memory-fuzz-oracle", "command": "go run ./tools/cmd/validate-memory-fuzz-oracle --report " + commandDir + "/memory-fuzz-oracle.json --artifact-dir " + commandDir + " --current-git-head " + gitHead, "status": "pass"},
		},
	})
}

func memory100PlaceholderJSON(schema string, gitHead string) map[string]any {
	key := "schema"
	if strings.Contains(schema, "report") || strings.Contains(schema, "summary") || strings.Contains(schema, "oracle") || strings.Contains(schema, "coverage") || strings.Contains(schema, "blockers") {
		key = "schema_version"
	}
	return map[string]any{
		key:        schema,
		"status":   "pass",
		"git_head": gitHead,
	}
}

func writeMemory100JSON(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeMemory100HashManifest(t *testing.T, root string) {
	t.Helper()
	type artifact struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
		Schema string `json:"schema,omitempty"`
	}
	var artifacts []artifact
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "artifact-hashes.json" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		artifacts = append(artifacts, artifact{
			Path:   rel,
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
			Size:   int64(len(raw)),
			Schema: memory100TestSchema(raw),
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	writeMemory100JSON(t, filepath.Join(root, "artifact-hashes.json"), map[string]any{
		"schema":    "tetra.release-artifact-hashes.v1alpha1",
		"root":      ".",
		"artifacts": artifacts,
	})
}

func memory100TestSchema(raw []byte) string {
	var envelope struct {
		Schema        string `json:"schema"`
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	if envelope.Schema != "" {
		return envelope.Schema
	}
	return envelope.SchemaVersion
}
