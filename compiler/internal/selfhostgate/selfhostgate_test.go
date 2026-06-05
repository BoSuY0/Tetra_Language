package selfhostgate

import (
	"strings"
	"testing"
)

func TestEvaluateBlocksSelfHostingUntilVerifiedCoreIsReady(t *testing.T) {
	decision := Evaluate(Evidence{
		CompilerSubsetDefined:       true,
		RegisterBackendStable:       true,
		OptimizerValidated:          true,
		AllocatorStable:             false,
		StdlibStrongEnough:          true,
		SmallCompilerComponentBuilt: false,
		GoVsTetraOutputCompared:     false,
		DeterministicBootstrapChain: false,
		CrossPlatformBootstrapStory: false,
	})
	if decision.Allowed {
		t.Fatalf("self-host gate allowed incomplete evidence: %+v", decision)
	}
	if !decision.Missing("allocator_stable") {
		t.Fatalf("decision missing allocator blocker: %+v", decision)
	}
	if !decision.Missing("small_compiler_component_compiled") {
		t.Fatalf("decision missing small component blocker: %+v", decision)
	}
	if !decision.Missing("cross_platform_bootstrap_story") {
		t.Fatalf("decision missing cross-platform blocker: %+v", decision)
	}
	if !strings.Contains(decision.Reason, "blocked") {
		t.Fatalf("reason = %q, want blocked", decision.Reason)
	}
}

func TestEvaluateAllowsSelfHostingOnlyWhenAllCoreEvidenceIsReady(t *testing.T) {
	decision := Evaluate(Evidence{
		CompilerSubsetDefined:       true,
		RegisterBackendStable:       true,
		OptimizerValidated:          true,
		AllocatorStable:             true,
		StdlibStrongEnough:          true,
		SmallCompilerComponentBuilt: true,
		GoVsTetraOutputCompared:     true,
		DeterministicBootstrapChain: true,
		CrossPlatformBootstrapStory: true,
	})
	if !decision.Allowed || len(decision.MissingEvidence) != 0 {
		t.Fatalf("self-host gate decision = %+v, want allowed", decision)
	}
}
