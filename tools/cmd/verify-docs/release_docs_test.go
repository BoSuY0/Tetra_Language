package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyReleaseTruthDocsRejectsMisleadingCurrentReleaseLanguage(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "current_supported_surface.md")
	body := strings.Join([]string{
		"# Current Surface",
		"",
		"The current public profile is v0.3.0.",
		"The current public baseline is v0.1.2.",
		"The current release is v0.6.",
		"Tetra is ready for v1.0.",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyReleaseTruthDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected misleading release language failure")
	}
	for _, want := range []string{"current.*v0.3", "v0.1.2", "current.*v0.6", "ready for v1.0"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyReleaseTruthDocsRejectsPerformanceAndTargetParityClaims(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "release_notes.md")
	body := strings.Join([]string{
		"# Release Notes",
		"",
		"Tetra is the fastest language in the official benchmark result.",
		"The package also proves target parity for memory production.",
		"The allocator has broad zero-cost performance across targets.",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyReleaseTruthDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected performance/target parity claim failure")
	}
	for _, want := range []string{"fastest language", "official benchmark", "target parity", "zero-cost performance"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyReleaseTruthDocsRejectsMemory100FormalProofAndLeakClaims(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "release_notes.md")
	body := strings.Join([]string{
		"# Release Notes",
		"",
		"The release proves full formal proof of memory safety.",
		"Memory production now has all-target memory parity.",
		"The memory model has no leaks.",
		"Memory 100% is guaranteed for users.",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyReleaseTruthDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected Memory100/formal/leak claim failure")
	}
	for _, want := range []string{"full formal proof", "all-target memory parity", "no leaks", "memory 100%"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyReleaseTruthDocsRejectsProductionPersistentObjectMemoryClaim(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "release_notes.md")
	body := strings.Join([]string{
		"# Release Notes",
		"",
		"Tetra now ships production object memory backed by persistent memory, Todium, memoryfield, WAL, FTS, vacuum, retention, stale memory, and false memory gates.",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyReleaseTruthDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected production persistent/object memory claim failure")
	}
	for _, want := range []string{"production object memory", "persistent memory", "todium", "memoryfield"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyReleaseTruthDocsAllowsPersistentObjectMemoryNonGoal(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "release_notes.md")
	body := strings.Join([]string{
		"# Release Notes",
		"",
		"Persistent/object memory is an explicit non-goal for this release: no production object memory, no production persistent memory, and no Todium or memoryfield production claim exists until retention/WAL/FTS/vacuum/stale/false-memory gates exist.",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifyReleaseTruthDocs([]string{doc}); err != nil {
		t.Fatalf("verifyReleaseTruthDocs: %v", err)
	}
}

func TestVerifyReleaseTruthDocsAllowsHistoricalTodoExclusion(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "2026-04-27-tetra-stabilization-5000-todo.md")
	body := "Historical TODO mentions current v0.6 and v0.1.2 for audit context.\n"
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifyReleaseTruthDocs([]string{doc}); err != nil {
		t.Fatalf("verifyReleaseTruthDocs: %v", err)
	}
}

func TestCurrentReleaseTruthDocPathsCoverCurrentUserAndSpecDocs(t *testing.T) {
	paths := currentReleaseTruthDocPaths()
	text := strings.Join(paths, "\n")
	for _, want := range []string{
		"README.md",
		"docs/spec/current_supported_surface.md",
		"docs/spec/surface_v1.md",
		"docs/spec/v0_2_scope.md",
		"docs/user/examples_index.md",
		"docs/user/getting_started.md",
		"docs/user/language_tour.md",
		"docs/user/surface_guide.md",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("currentReleaseTruthDocPaths missing %s in %v", want, paths)
		}
	}
	for _, forbidden := range []string{"docs/plans/", "docs/release-notes/"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("currentReleaseTruthDocPaths should not include historical %s paths: %v", forbidden, paths)
		}
	}
}

func TestVerifySurfaceReleaseDocsRejectsFakePromotionClaims(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{
			name: "macos-current",
			body: "macOS Surface is current for Surface v1.\nUnsupported targets: wasm32-wasi.\nbash scripts/release/surface/release-gate.sh\n",
			want: "macOS Surface",
		},
		{
			name: "windows-current",
			body: "Windows Surface is release-ready for Surface v1.\nUnsupported targets: wasm32-wasi.\nbash scripts/release/surface/release-gate.sh\n",
			want: "Windows Surface",
		},
		{
			name: "metadata-only-production-accessibility",
			body: "metadata-only accessibility is production accessibility.\nUnsupported targets: macOS, Windows, wasm32-wasi.\nbash scripts/release/surface/release-gate.sh\n",
			want: "metadata-only",
		},
		{
			name: "dom-ui-model",
			body: "DOM UI is the Surface model.\nUnsupported targets: macOS, Windows, wasm32-wasi.\nbash scripts/release/surface/release-gate.sh\n",
			want: "DOM UI",
		},
		{
			name: "user-js-allowed",
			body: "user JS app logic is allowed in Surface apps.\nUnsupported targets: macOS, Windows, wasm32-wasi.\nbash scripts/release/surface/release-gate.sh\n",
			want: "user JS",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := writeSurfaceReleaseDoc(t, tc.body)
			err := verifySurfaceReleaseDocs([]string{doc})
			if err == nil {
				t.Fatalf("expected Surface release docs fake-promotion failure")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestVerifySurfaceReleaseDocsRequireUnsupportedTargetsAndReleaseGate(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{
			name: "missing-unsupported-targets",
			body: "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas.\nbash scripts/release/surface/release-gate.sh\n",
			want: "unsupported targets",
		},
		{
			name: "missing-release-gate-command",
			body: "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. Unsupported targets: macOS, Windows, wasm32-wasi.\n",
			want: "release-gate.sh",
		},
		{
			name: "missing-claim-tier-vocabulary",
			body: "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets.\n\nbash scripts/release/surface/release-gate.sh\nbash scripts/release/surface/product-gate.sh\n",
			want: "PROD_STABLE_SCOPED",
		},
		{
			name: "missing-product-gate-command",
			body: "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets. Claim tiers: PROD_STABLE_SCOPED, BETA_TARGET_HOST, EXPERIMENTAL, UNSUPPORTED, NONCLAIM.\n\nbash scripts/release/surface/release-gate.sh\n",
			want: "product-gate.sh",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := writeSurfaceReleaseDoc(t, tc.body)
			err := verifySurfaceReleaseDocs([]string{doc})
			if err == nil {
				t.Fatalf("expected Surface release docs requirement failure")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}

	okDoc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets. Metadata-only accessibility is not production accessibility. DOM UI and user JavaScript app logic are outside the Surface model. Claim tiers: PROD_STABLE_SCOPED, BETA_TARGET_HOST, EXPERIMENTAL, UNSUPPORTED, NONCLAIM.\n\nbash scripts/release/surface/release-gate.sh\nbash scripts/release/surface/product-gate.sh\n")
	if err := verifySurfaceReleaseDocs([]string{okDoc}); err != nil {
		t.Fatalf("verifySurfaceReleaseDocs accepted doc: %v", err)
	}
}

func TestVerifySurfaceReleaseDocsRequireP28GovernancePerDocument(t *testing.T) {
	fullDoc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets. Claim tiers: PROD_STABLE_SCOPED, BETA_TARGET_HOST, EXPERIMENTAL, UNSUPPORTED, NONCLAIM.\n\nbash scripts/release/surface/release-gate.sh\nbash scripts/release/surface/product-gate.sh\n")
	missingTierDoc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets.\n\nbash scripts/release/surface/release-gate.sh\nbash scripts/release/surface/product-gate.sh\n")
	err := verifySurfaceReleaseDocs([]string{fullDoc, missingTierDoc})
	if err == nil {
		t.Fatalf("expected per-document claim-tier requirement failure")
	}
	if !strings.Contains(err.Error(), "PROD_STABLE_SCOPED") {
		t.Fatalf("error = %v, want PROD_STABLE_SCOPED diagnostic", err)
	}
}

func TestVerifySurfaceReleaseDocsRejectsMixedGPUProductionWithoutEvidenceClause(t *testing.T) {
	doc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets. Claim tiers: PROD_STABLE_SCOPED, BETA_TARGET_HOST, EXPERIMENTAL, UNSUPPORTED, NONCLAIM.\n\nSurface GPU rendering is production supported without additional evidence.\n\nbash scripts/release/surface/release-gate.sh\nbash scripts/release/surface/product-gate.sh\n")
	err := verifySurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected mixed GPU production claim failure")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "gpu") {
		t.Fatalf("error = %v, want GPU diagnostic", err)
	}
}

func TestVerifySurfaceReleaseDocsRejectsFinalCurrentClaimOwnership(t *testing.T) {
	doc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets. Claim tiers: PROD_STABLE_SCOPED, BETA_TARGET_HOST, EXPERIMENTAL, UNSUPPORTED, NONCLAIM.\n\nThe release gate is the source of truth for the final current claim.\n\nbash scripts/release/surface/release-gate.sh\nbash scripts/release/surface/product-gate.sh\n")
	err := verifySurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected final current claim ownership failure")
	}
	if !strings.Contains(err.Error(), "final current claim") {
		t.Fatalf("error = %v, want final current claim diagnostic", err)
	}
}

func TestSurfaceDocsOverclaimRejectsTmpEvidenceAsCurrentProof(t *testing.T) {
	doc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets. Metadata-only accessibility is not production accessibility. DOM UI and user JavaScript app logic are outside the Surface model.\n\nbash scripts/release/surface/release-gate.sh --report-dir /tmp/tetra-surface-release-v1-current\n")
	err := verifySurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected Surface release docs to reject /tmp current evidence")
	}
	if !strings.Contains(err.Error(), "/tmp") {
		t.Fatalf("error = %v, want /tmp rejection", err)
	}
}

func TestSurfaceOverclaimRejectsGPUAndNativeWidgets(t *testing.T) {
	doc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets. Metadata-only accessibility is not production accessibility. DOM UI and user JavaScript app logic are outside the Surface model.\n\nGPU rendering is production-supported for Surface v1. Platform-native widgets are release-ready.\n\nbash scripts/release/surface/release-gate.sh\n")
	err := verifySurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected Surface docs to reject GPU/native-widget overclaims")
	}
	for _, want := range []string{"GPU", "native widget"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestUnsupportedSurfaceTargetsRejectsCrossPlatformProductionClaim(t *testing.T) {
	doc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets.\n\nSurface is a production cross-platform UI runtime across macOS, Windows, linux, and wasm32-wasi.\n\nbash scripts/release/surface/release-gate.sh\n")
	err := verifySurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected Surface docs to reject cross-platform production overclaim")
	}
	if !strings.Contains(err.Error(), "cross-platform") {
		t.Fatalf("expected cross-platform in error, got %v", err)
	}
}

func TestSurfaceOverclaimRejectsRichTextScreenReaderDOMReactUserJS(t *testing.T) {
	doc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets.\n\nRich text editing is production-supported. Full screen-reader support is release-ready. DOM UI is production-supported. React apps are current Surface apps. User JS app logic is allowed in Surface apps.\n\nbash scripts/release/surface/release-gate.sh\n")
	err := verifySurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected Surface docs to reject rich-text/screen-reader/DOM/React/user-JS overclaims")
	}
	for _, want := range []string{"rich text", "screen-reader", "DOM UI", "React", "user JS"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestSurfaceBlockSystemRejectsCoreWidgetPrimitiveClaims(t *testing.T) {
	doc := writeSurfaceReleaseDoc(t, "Surface v1 scope is linux-x64 real-window and wasm32-web browser-canvas. macOS Surface, Windows Surface, and wasm32-wasi Surface UI are unsupported targets.\n\nButton is a core Surface primitive. TextField is a core Surface primitive. Card is a core Surface primitive.\n\nbash scripts/release/surface/release-gate.sh\n")
	err := verifySurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected Surface docs to reject core widget primitive claims")
	}
	for _, want := range []string{"Button", "TextField", "Card"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyMemoryIslandsSurfaceReleaseDocsRejectsIncompleteScope(t *testing.T) {
	doc := writeMemoryIslandsSurfaceReleaseDoc(t, "Memory/Islands/Surface scoped release truth.\n")
	err := verifyMemoryIslandsSurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected incomplete Memory/Islands/Surface release docs failure")
	}
	for _, want := range []string{"validate-island-proof", "memory-islands-surface-production-gate.sh"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyMemoryIslandsSurfaceReleaseDocsRejectsBroadOverclaim(t *testing.T) {
	doc := writeMemoryIslandsSurfaceReleaseDoc(t, strings.Join([]string{
		validMemoryIslandsSurfaceReleaseDocBody(),
		"Memory/Islands/Surface is fully production-ready across all targets.",
	}, "\n"))
	err := verifyMemoryIslandsSurfaceReleaseDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected Memory/Islands/Surface broad overclaim failure")
	}
	if !strings.Contains(err.Error(), "fully production-ready") {
		t.Fatalf("expected fully production-ready in error, got %v", err)
	}
}

func TestVerifyMemoryIslandsSurfaceReleaseDocsAcceptsScopedEvidence(t *testing.T) {
	doc := writeMemoryIslandsSurfaceReleaseDoc(t, validMemoryIslandsSurfaceReleaseDocBody())
	if err := verifyMemoryIslandsSurfaceReleaseDocs([]string{doc}); err != nil {
		t.Fatalf("verifyMemoryIslandsSurfaceReleaseDocs: %v", err)
	}
}

func TestVerifyFinalMemoryIslandsSurfaceProductionAuditRejectsMissingCommands(t *testing.T) {
	body := strings.ReplaceAll(validFinalMemoryIslandsSurfaceProductionAuditBody(), "git status --short", "")
	doc := writeFinalMemoryIslandsSurfaceProductionAudit(t, body)
	err := verifyFinalMemoryIslandsSurfaceProductionAudit([]string{doc})
	if err == nil {
		t.Fatalf("expected final production audit to reject missing command evidence")
	}
	if !strings.Contains(err.Error(), "git status --short") {
		t.Fatalf("expected git status command in error, got %v", err)
	}
}

func TestVerifyFinalMemoryIslandsSurfaceProductionAuditRejectsBroadReadyClaim(t *testing.T) {
	doc := writeFinalMemoryIslandsSurfaceProductionAudit(t, validFinalMemoryIslandsSurfaceProductionAuditBody()+"\nIntegrated: PROD_READY_PROVEN across all targets.\n")
	err := verifyFinalMemoryIslandsSurfaceProductionAudit([]string{doc})
	if err == nil {
		t.Fatalf("expected final production audit to reject broad PROD_READY_PROVEN claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "prod_ready_proven") {
		t.Fatalf("expected PROD_READY_PROVEN in error, got %v", err)
	}
}

func TestVerifyFinalMemoryIslandsSurfaceProductionAuditAcceptsScopedEvidence(t *testing.T) {
	doc := writeFinalMemoryIslandsSurfaceProductionAudit(t, validFinalMemoryIslandsSurfaceProductionAuditBody())
	if err := verifyFinalMemoryIslandsSurfaceProductionAudit([]string{doc}); err != nil {
		t.Fatalf("verifyFinalMemoryIslandsSurfaceProductionAudit: %v", err)
	}
}

func TestVerifyMemoryIslandsFinalProductionReadinessAuditRejectsMissingCommandLogArtifactHashesAndRisks(t *testing.T) {
	body := strings.ReplaceAll(validMemoryIslandsFinalProductionReadinessAuditBody(), "## Command Log", "## Commands")
	body = strings.ReplaceAll(body, "## Artifact Hashes", "## Hashes")
	body = strings.ReplaceAll(body, "## Residual Risks", "## Risks")
	doc := writeMemoryIslandsFinalProductionReadinessAudit(t, body)
	err := verifyMemoryIslandsFinalProductionReadinessAudit([]string{doc})
	if err == nil {
		t.Fatalf("expected final Memory/Islands audit to reject missing command log/artifact hash/residual risk sections")
	}
	for _, want := range []string{"command log", "artifact hashes", "residual risks"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyMemoryIslandsFinalProductionReadinessAuditRejectsBroadReadyClaim(t *testing.T) {
	doc := writeMemoryIslandsFinalProductionReadinessAudit(t, validMemoryIslandsFinalProductionReadinessAuditBody()+"\nMemory verdict: `PROD_READY_PROVEN`\n")
	err := verifyMemoryIslandsFinalProductionReadinessAudit([]string{doc})
	if err == nil {
		t.Fatalf("expected final Memory/Islands audit to reject broad ready claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "prod_ready_proven") {
		t.Fatalf("expected PROD_READY_PROVEN in error, got %v", err)
	}
}

func TestVerifyMemoryIslandsFinalProductionReadinessAuditAcceptsHonestScopedEvidence(t *testing.T) {
	doc := writeMemoryIslandsFinalProductionReadinessAudit(t, validMemoryIslandsFinalProductionReadinessAuditBody())
	if err := verifyMemoryIslandsFinalProductionReadinessAudit([]string{doc}); err != nil {
		t.Fatalf("verifyMemoryIslandsFinalProductionReadinessAudit: %v", err)
	}
}

func TestVerifyMemoryIslandsFinalActorBenchmarkHandoffRejectsActorProductionClaimWithoutGate(t *testing.T) {
	doc := writeMemoryIslandsFinalActorBenchmarkHandoff(t, validMemoryIslandsFinalActorBenchmarkHandoffBody()+"\nThe production actor runtime is ready now and the actor production gate passed.\n")
	err := verifyMemoryIslandsFinalActorBenchmarkHandoff([]string{doc})
	if err == nil {
		t.Fatalf("expected final actor handoff to reject production actor claim without gate evidence")
	}
	for _, want := range []string{"production actor runtime", "actor production gate passed"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyMemoryIslandsFinalActorBenchmarkHandoffRejectsBenchmarkOverclaim(t *testing.T) {
	doc := writeMemoryIslandsFinalActorBenchmarkHandoff(t, validMemoryIslandsFinalActorBenchmarkHandoffBody()+"\nBenchmark phase may claim an official benchmark result and C++/Rust parity.\n")
	err := verifyMemoryIslandsFinalActorBenchmarkHandoff([]string{doc})
	if err == nil {
		t.Fatalf("expected final actor handoff to reject benchmark overclaim")
	}
	for _, want := range []string{"official benchmark", "c++/rust parity"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyMemoryIslandsFinalActorBenchmarkHandoffAcceptsScopedPreconditions(t *testing.T) {
	doc := writeMemoryIslandsFinalActorBenchmarkHandoff(t, validMemoryIslandsFinalActorBenchmarkHandoffBody())
	if err := verifyMemoryIslandsFinalActorBenchmarkHandoff([]string{doc}); err != nil {
		t.Fatalf("verifyMemoryIslandsFinalActorBenchmarkHandoff: %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsRejectsBroadActorProductionClaim(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, validActorRuntimeFoundationDocBody()+".\nThe full production actor runtime is ready now.\n")
	err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures())
	if err == nil {
		t.Fatalf("expected actor foundation docs to reject broad production actor claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "full production actor runtime") {
		t.Fatalf("expected full production actor runtime in error, got %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsRejectsProdReadyProvenClaim(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, validActorRuntimeFoundationDocBody()+".\nActor foundation verdict: PROD_READY_PROVEN.\n")
	err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures())
	if err == nil {
		t.Fatalf("expected actor foundation docs to reject PROD_READY_PROVEN claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "prod_ready_proven") {
		t.Fatalf("expected PROD_READY_PROVEN in error, got %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsAllowsProdReadyProvenNonClaim(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, validActorRuntimeFoundationDocBody()+"\n`PROD_READY_PROVEN`: `NOT_CLAIMED`.\n")
	if err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures()); err != nil {
		t.Fatalf("verifyActorRuntimeFoundationDocs: %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsRejectsNonLinuxDistributedClaim(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, validActorRuntimeFoundationDocBody()+".\nWindows distributed actor runtime support is production-ready.\n")
	err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures())
	if err == nil {
		t.Fatalf("expected actor foundation docs to reject non-Linux distributed actor claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "windows distributed actor runtime") {
		t.Fatalf("expected Windows distributed actor runtime in error, got %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsRejectsDistributedZeroCopyClaim(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, validActorRuntimeFoundationDocBody()+".\nDistributed zero-copy pointer transfer is supported.\n")
	err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures())
	if err == nil {
		t.Fatalf("expected actor foundation docs to reject distributed zero-copy claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "distributed zero-copy") {
		t.Fatalf("expected distributed zero-copy in error, got %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsRejectsMissingDistributedTargetMatrix(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, strings.Replace(validActorRuntimeFoundationDocBody(), distributedRuntimeTargetMatrixDocBody(), "", 1))
	err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures())
	if err == nil {
		t.Fatalf("expected actor foundation docs to require distributed target matrix")
	}
	if !strings.Contains(err.Error(), "Distributed Runtime Target Matrix") {
		t.Fatalf("expected Distributed Runtime Target Matrix in error, got %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsRejectsMissingBenchmarkNonClaim(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, strings.Replace(validActorRuntimeFoundationDocBody(), "no benchmark superiority, no C++/Rust parity, and no official benchmark claim", "", 1))
	err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures())
	if err == nil {
		t.Fatalf("expected actor foundation docs to require actor benchmark nonclaim")
	}
	if !strings.Contains(err.Error(), "no benchmark superiority") {
		t.Fatalf("expected actor benchmark nonclaim in error, got %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsRejectsStaleManifestFeature(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, validActorRuntimeFoundationDocBody())
	features := validActorRuntimeFoundationFeatures()
	features[0].Stability = "current Linux-x64 runtime evidence without the final actor foundation gate"
	err := verifyActorRuntimeFoundationDocs([]string{doc}, features)
	if err == nil {
		t.Fatalf("expected actor foundation docs to reject stale manifest feature")
	}
	if !strings.Contains(err.Error(), "tetra.actor.production_foundation.v1") {
		t.Fatalf("expected production foundation schema in error, got %v", err)
	}
}

func TestVerifyActorRuntimeFoundationDocsAcceptsScopedGateEvidence(t *testing.T) {
	doc := writeActorRuntimeFoundationDoc(t, validActorRuntimeFoundationDocBody())
	if err := verifyActorRuntimeFoundationDocs([]string{doc}, validActorRuntimeFoundationFeatures()); err != nil {
		t.Fatalf("verifyActorRuntimeFoundationDocs: %v", err)
	}
}

func TestDefaultActorRuntimeFoundationDocPathsIncludeHistoricalFinalAudit(t *testing.T) {
	paths := defaultActorRuntimeFoundationDocPaths()
	want := filepath.FromSlash("docs/audits/actor-runtime-production-foundation-final.md")
	for _, path := range paths {
		if path == want {
			return
		}
	}
	t.Fatalf("defaultActorRuntimeFoundationDocPaths() missing %q: %#v", want, paths)
}

func TestVerifyRAMContractCompilerDocsRejectsIncompleteDocs(t *testing.T) {
	paths := writeRAMContractDocsSet(t, "RAM Contract Compiler\n")
	err := verifyRAMContractCompilerDocs(paths, []featureManifest{validVerifyDocsRAMContractFeature()})
	if err == nil {
		t.Fatalf("expected incomplete RAM contract docs failure")
	}
	for _, want := range []string{"tetra.ram-contract-report.v1", "ram-contract-linux-x64-smoke.sh"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyRAMContractCompilerDocsRejectsForbiddenClaim(t *testing.T) {
	paths := writeRAMContractDocsSet(t, validRAMContractDocsBody()+"\nRAM Contract Compiler proves zero heap for all programs.\n")
	err := verifyRAMContractCompilerDocs(paths, []featureManifest{validVerifyDocsRAMContractFeature()})
	if err == nil {
		t.Fatalf("expected RAM contract docs forbidden claim failure")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "zero heap for all programs") {
		t.Fatalf("expected zero heap claim in error, got %v", err)
	}
}

func TestVerifyRAMContractCompilerDocsRejectsUnsupportedValidatorFlag(t *testing.T) {
	paths := writeRAMContractDocsSet(t, validRAMContractDocsBody()+"\ngo run ./tools/cmd/validate-ram-contract-release --report reports/ram-contract-release\n")
	err := verifyRAMContractCompilerDocs(paths, []featureManifest{validVerifyDocsRAMContractFeature()})
	if err == nil {
		t.Fatalf("expected unsupported RAM contract validator flag failure")
	}
	if !strings.Contains(err.Error(), "validate-ram-contract-release --report") {
		t.Fatalf("expected unsupported flag in error, got %v", err)
	}
}

func TestVerifyRAMContractCompilerDocsRejectsStaleReadinessHead(t *testing.T) {
	if _, ok := currentGitHeadForDocs(); !ok {
		t.Skip("git head unavailable")
	}
	paths := writeRAMContractDocsSet(t, validRAMContractDocsBody())
	stale := "0000000000000000000000000000000000000000"
	body := validRAMContractDocsBody() + "\nGit head: " + stale + "\n"
	if err := os.WriteFile(paths.Readiness, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyRAMContractCompilerDocs(paths, []featureManifest{validVerifyDocsRAMContractFeature()})
	if err == nil {
		t.Fatalf("expected stale readiness git head failure")
	}
	if !strings.Contains(err.Error(), "stale readiness git head "+stale) {
		t.Fatalf("expected stale head in error, got %v", err)
	}
}

func TestVerifyRAMContractCompilerDocsAcceptsDirectParentReadinessHead(t *testing.T) {
	parent, ok := currentGitParentForDocs()
	if !ok {
		t.Skip("git parent unavailable")
	}
	paths := writeRAMContractDocsSet(t, validRAMContractDocsBody())
	body := validRAMContractDocsBody() + "\nGit head: " + parent + "\n"
	if err := os.WriteFile(paths.Readiness, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyRAMContractCompilerDocs(paths, []featureManifest{validVerifyDocsRAMContractFeature()}); err != nil {
		t.Fatalf("verifyRAMContractCompilerDocs accepted direct parent evidence head: %v", err)
	}
}

func TestVerifyRAMContractCompilerDocsAcceptsScopedEvidence(t *testing.T) {
	paths := writeRAMContractDocsSet(t, validRAMContractDocsBody())
	if err := verifyRAMContractCompilerDocs(paths, []featureManifest{validVerifyDocsRAMContractFeature()}); err != nil {
		t.Fatalf("verifyRAMContractCompilerDocs: %v", err)
	}
}
