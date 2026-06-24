package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMemoryIslandsSurfaceReleaseDoc(t *testing.T, body string) string {
	t.Helper()
	doc := filepath.Join(t.TempDir(), "memory_islands_surface_scope.md")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return doc
}

func writeRAMContractDocsSet(t *testing.T, body string) ramContractCompilerDocPaths {
	t.Helper()
	dir := t.TempDir()
	paths := ramContractCompilerDocPaths{
		Design:    filepath.Join(dir, "ram_contract_compiler.md"),
		Spec:      filepath.Join(dir, "ram_contract_report_schema.md"),
		User:      filepath.Join(dir, "ram_contracts.md"),
		Readiness: filepath.Join(dir, "ram-contract-compiler-readiness.md"),
		Handoff:   filepath.Join(dir, "ram-contract-compiler-handoff.md"),
	}
	for _, path := range []string{
		paths.Design,
		paths.Spec,
		paths.User,
		paths.Readiness,
		paths.Handoff,
	} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return paths
}

func validRAMContractDocsBody() string {
	return strings.Join([]string{
		"RAM Contract Compiler",
		"RAM Contract Report Schema",
		"Using RAM Contracts",
		"RAM Contract Compiler Readiness Audit",
		"RAM Contract Compiler Handoff",
		"tetra.ram-contract-report.v1",
		"tetra.memory-grade-report.v1",
		"tetra.proof-store-summary.v1",
		"tetra.validation-pipeline-coverage.v1",
		"tetra.ram-blockers.v1",
		"compiler-owned facts",
		"MemoryFactGraph",
		"AllocPlan",
		"ProofStore",
		"heap-blockers.json",
		"copy-blockers.json",
		"ram-contract-fuzz-oracle.json",
		"ram-contract-report.json",
		"memory-grade-report.json",
		"proof-store-summary.json",
		"validation-pipeline-coverage.json",
		"TETRA4100",
		"--emit-ram-contract-report",
		"--fail-if-heap",
		"--fail-if-copy",
		"--fail-if-unbounded",
		"--memory-budget",
		"--ram-contract",
		"validate-ram-contract-report",
		"validate-memory-grade-report",
		"validate-proof-store-summary",
		"validate-validation-pipeline-coverage",
		"validate-heap-blockers",
		"validate-copy-blockers",
		"validate-ram-contract-fuzz-oracle",
		"validate-ram-contract-release",
		"scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh",
		".github/workflows/ci.yml",
		".github/workflows/release-packages.yml",
		"go test -buildvcs=false",
		"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
		"git diff --check",
		"reports/ram-contract-release",
		"Git head:",
		"Working tree:",
		"dirty working tree",
		"Verdict: `SCOPED_READY`",
		"Release gate:",
		"CI job:",
		"ram-contract-release-readiness-linux",
		"Package workflow:",
		"ram-contract-linux-x64",
		"Required artifacts:",
		"no zero heap for all programs claim",
		"no zero-copy for all programs claim",
		"no full formal proof claim",
		"no all-target RAM parity claim",
	}, "\n")
}

func validMemoryIslandsSurfaceReleaseDocBody() string {
	return strings.Join([]string{
		"Memory/Islands/Surface scoped release truth",
		"scripts/release/post_v0_4/memory-islands-surface-production-gate.sh",
		"tools/cmd/validate-memory-islands-surface-production",
		"tools/cmd/validate-island-proof",
		"--islands-debug",
		"islands-debug-smoke.json",
		"island-proof-verifier.json",
		"island-proof-fuzz-summary.json",
		"memory-islands-surface-production-manifest.json",
		"artifact-hashes.json",
		"leak/resource finalization evidence",
		"surface-v1-linux-web",
		"no Memory 100% claim",
		"no arbitrary unsafe external pointer safety",
		"no full formal proof",
		"no full target parity",
		"no all-target Surface claim",
		"no production object memory claim",
		"no production persistent memory claim",
		"not a clean release-candidate checkout claim",
	}, "\n")
}

func writeFinalMemoryIslandsSurfaceProductionAudit(t *testing.T, body string) string {
	t.Helper()
	doc := filepath.Join(t.TempDir(), "memory-islands-surface-final-production-readiness.md")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return doc
}

func writeMemoryIslandsFinalProductionReadinessAudit(t *testing.T, body string) string {
	t.Helper()
	doc := filepath.Join(t.TempDir(), "memory-islands-final-production-readiness.md")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return doc
}

func writeMemoryIslandsFinalActorBenchmarkHandoff(t *testing.T, body string) string {
	t.Helper()
	doc := filepath.Join(t.TempDir(), "memory-islands-final-production-handoff.md")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return doc
}

func writeActorRuntimeFoundationDoc(t *testing.T, body string) string {
	t.Helper()
	doc := filepath.Join(t.TempDir(), "actor-runtime-foundation.md")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return doc
}

func validActorRuntimeFoundationDocBody() string {
	return strings.Join([]string{
		"Actor runtime foundation scoped release truth",
		"tetra.actor.production_foundation.v1",
		"scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh",
		".github/workflows/ci.yml",
		".github/workflows/release-packages.yml",
		"reports/actor-runtime-foundation/final/actor-runtime-foundation-manifest.json",
		"reports/actor-runtime-foundation/final/artifact-hashes.json",
		"distributed-actors-linux-x64/distributed-actors-linux-x64.json",
		"parallel-production-linux-x64/parallel-production-linux-x64.json",
		"subordinate to current same-commit actor foundation gates",
		"no full Erlang/OTP actor runtime claim",
		"no cluster membership or reconnect/retry production claim",
		"no non-Linux distributed actor runtime support claim",
		"no distributed zero-copy pointer or region transfer claim",
		"no formal race proof claim",
		"no benchmark superiority, no C++/Rust parity, and no official benchmark claim",
		distributedRuntimeTargetMatrixDocBody(),
	}, "\n")
}

func distributedRuntimeTargetMatrixDocBody() string {
	return strings.Join([]string{
		"Distributed Runtime Target Matrix",
		"| Target | Distributed actor runtime status | Current evidence | Promotion requirement |",
		"|---|---|---|---|",
		("`linux-x64` | current scoped | executable " +
			"`tetra.actors.distributed-runtime.v1` smoke plus actor foundation gate " +
			"| keep same-commit distributed smoke, artifact hashes, and foundation " +
			"validator green |"),
		("`macos-x64` | unsupported / nonclaim | no distributed actor " +
			"symbols; actor net pump is no-op | add target runtime, smoke, validator," +
			" docs, and package gate before any support claim |"),
		("`windows-x64` | unsupported / nonclaim | no distributed actor " +
			"symbols; actor net pump is no-op | add target runtime, smoke, validator," +
			" docs, and package gate before any support claim |"),
		("`wasm32-wasi` | unsupported / nonclaim | no distributed actor " +
			"runtime gate | add target runtime, smoke, validator, docs, and package " +
			"gate before any support claim |"),
		("`wasm32-web` | unsupported / nonclaim | no distributed actor " +
			"runtime gate | add target runtime, smoke, validator, docs, and package " +
			"gate before any support claim |"),
	}, "\n")
}

func validActorRuntimeFoundationFeatures() []featureManifest {
	return []featureManifest{
		{
			ID:     "actors.distributed-runtime",
			Name:   "Distributed actor runtime for Linux x64",
			Status: "current",
			Since:  "v0.4.0",
			Scope: ("production Linux-x64 distributed actor runtime path with the " +
				"scoped actor runtime foundation gate"),
			Stability: ("current Linux-x64 runtime/lowering slice with executable " +
				"tetra.actors.distributed-runtime.v1 smoke evidence, " +
				"tetra.actor.production_foundation.v1 gate evidence, actor-runtime-" +
				"foundation-linux-x64-gate.sh, and strict nonclaims for non-Linux " +
				"distributed runtime, distributed zero-copy, cluster membership, " +
				"reconnect/retry production, and formal race proof"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/actors.md",
				"docs/user/platform/async_actors_guide.md",
				"docs/design/actor_region_transfer.md",
				"docs/audits/runtime/actors/actor-runtime-production-boundary-v1.md",
				"docs/checklists/actors_linux_smoke.md",
				"docs/checklists/actors_platform_smoke.md",
			},
		},
	}
}

func validFinalMemoryIslandsSurfaceProductionAuditBody() string {
	return strings.Join([]string{
		"# Memory/Islands/Surface Final Production Readiness Audit",
		"Git head: e2c19b8ee276158f8eb2c54cf61e11bd84952893",
		"Working tree: dirty working tree evidence, not a clean release-candidate checkout claim",
		"Memory verdict: `PROD_STABLE_SCOPED`",
		"Islands verdict: `PROD_STABLE_SCOPED`",
		"Surface verdict: `PROD_STABLE_SCOPED` for `surface-v1-linux-web`",
		"Integrated verdict: `PROD_STABLE_SCOPED`",
		("Commands: go test -buildvcs=false ./tools/cmd/verify-docs -run " +
			"'Final|Production|Audit|Overclaim' -count=1"),
		"Commands: git diff --check",
		"Commands: git status --short",
		("Commands: go run -buildvcs=false ./tools/cmd/verify-docs --" +
			"manifest docs/generated/manifest.json"),
		"Artifacts: reports/mis-ideal/P13/integrated",
		"Artifacts: reports/mis-ideal/P15/docs-manifest-overclaim.md",
		"Hashes: sha256 a7f3da4cab2494dda804bd3d4e5d00d7ccc403255b01eb07461b0bf126151953",
		"Changed Files: .github/workflows/ci.yml",
		"Changed Files: docs/generated/manifest.json",
		"Changed Files: docs/release/surface/memory_islands_surface_scope.md",
		"Changed Files: scripts/release/post_v0_4/memory-islands-surface-production-gate.sh",
		"Changed Files: tools/cmd/validate-island-proof",
		"Changed Files: tools/cmd/validate-memory-islands-surface-production",
		"Residual Risks: remote GitHub Actions run was not executed",
		("Residual Risks: tools/cmd/dump-project optional broad " +
			"scriptstest fixture still fails outside P14"),
		("Residual Risks: tools/validators/postv04prod optional broad " +
			"scriptstest fixture still fails outside P14"),
		"Nonclaims: no Memory 100% claim",
		"Nonclaims: no arbitrary unsafe external pointer safety",
		"Nonclaims: no full formal proof",
		"Nonclaims: no full target parity",
		"Nonclaims: no all-target Surface claim",
	}, "\n")
}

func validMemoryIslandsFinalProductionReadinessAuditBody() string {
	return strings.Join([]string{
		"# Memory/Islands Final Production Readiness Audit",
		"Git head: e2c19b8ee276158f8eb2c54cf61e11bd84952893",
		"Working tree: dirty working tree evidence, not a clean release-candidate checkout claim.",
		"Memory verdict: `PROD_STABLE_SCOPED`",
		"Islands verdict: `PROD_STABLE_SCOPED`",
		"Integrated gate verdict: `PROD_STABLE_SCOPED`",
		"## Scope",
		"Memory/Islands scope: linux-x64 MemoryFactGraph-backed reports and island proof validation.",
		"Integrated gate scope: Memory/Islands plus existing scoped Surface dependency evidence.",
		"## Command Log",
		"`git status --short`",
		"`git rev-parse HEAD`",
		"`go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1`",
		("`go test -race -buildvcs=false ./compiler/internal/islandkernel " +
			"./compiler/internal/memoryfacts ./compiler/internal/memoryfacts_test ./" +
			"compiler/internal/semantics ./compiler/internal/plir ./compiler/" +
			"internal/validation ./cli/internal/actornet -count=1`"),
		("`bash scripts/release/post_v0_4/memory-production-linux-x64-" +
			"smoke.sh --report-dir reports/memory-islands-ideal/final/memory-" +
			"production`"),
		("`bash scripts/release/post_v0_4/memory-islands-surface-" +
			"production-gate.sh --report-dir reports/memory-islands-ideal/final/" +
			"integrated`"),
		"`go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`",
		"`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`",
		"`git diff --check`",
		"## Artifact Log",
		"`reports/memory-islands-ideal/final/memory-production`",
		"`reports/memory-islands-ideal/final/integrated`",
		"`reports/memory-islands-ideal/final/artifact-sha256.txt`",
		"## Artifact Hashes",
		"`0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef`",
		"## Residual Risks",
		("- dirty working tree blocks clean release-candidate proof until " +
			"committed or otherwise resolved."),
		"- remote GitHub Actions evidence is not present.",
		"## Nonclaims",
		"- no Memory 100% claim",
		"- no arbitrary unsafe external pointer safety",
		"- no full formal proof",
		"- no full target parity",
		"- no production actor runtime",
		"- no official benchmark result",
		"- not a clean release-candidate checkout claim",
	}, "\n")
}

func validMemoryIslandsFinalActorBenchmarkHandoffBody() string {
	return strings.Join([]string{
		"# Memory/Islands Final Production Audit and Actor Handoff",
		"Final verdict: `PROD_STABLE_SCOPED`",
		("Memory/Islands baseline: `docs/audits/memory/islands/memory-" +
			"islands-final-production-readiness.md` and `reports/memory-islands-" +
			"ideal/final/artifact-sha256.txt`."),
		("Actor handoff readiness: actor phase may start as a separate " +
			"Actor Runtime Production Foundation plan."),
		"Actor runtime production status: not started in this plan.",
		"Actor phase preconditions:",
		("- production actor gate must prove scheduler, mailbox " +
			"backpressure, message exhaustion/reclamation, race-safety, cross-target " +
			"distributed runtime gates, structured concurrency, and fake-evidence " +
			"rejection."),
		("- `docs/audits/runtime/actors/actor-runtime-production-boundary-" +
			"v1.md` remains the actor production boundary."),
		"- `MEMISL-P10` memory-boundary handoff evidence is an input, not actor runtime completion.",
		"Benchmark preconditions:",
		"- benchmark phase may start only as Tier 0/Tier 1 preparation until measured evidence exists.",
		"- no official benchmark result",
		"- no performance superiority",
		"- no C++/Rust parity",
		"- no measured speed comparison",
		"Nonclaims:",
		"- no production actor runtime",
		"- no actor production gate passed",
		"- no official benchmark result",
		"- no performance superiority",
		"- no `PROD_READY_PROVEN` claim",
	}, "\n")
}

func writeSurfaceReleaseDoc(t *testing.T, body string) string {
	t.Helper()
	doc := filepath.Join(t.TempDir(), "surface_v1.md")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return doc
}
