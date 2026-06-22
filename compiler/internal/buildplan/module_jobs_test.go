package buildplan

import (
	"testing"

	"tetra_language/compiler/internal/format/tobj"
)

func TestEffectiveWorkerCount(t *testing.T) {
	tests := []struct {
		name      string
		requested int
		maxJobs   int
		fallback  int
		want      int
	}{
		{name: "uses requested jobs", requested: 2, maxJobs: 5, fallback: 8, want: 2},
		{name: "defaults from fallback", requested: 0, maxJobs: 5, fallback: 8, want: 5},
		{name: "floors fallback", requested: 0, maxJobs: 5, fallback: 0, want: 1},
		{name: "caps requested at pending jobs", requested: 8, maxJobs: 3, fallback: 8, want: 3},
		{name: "zero pending jobs stays zero", requested: 8, maxJobs: 0, fallback: 8, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EffectiveWorkerCount(tt.requested, tt.maxJobs, tt.fallback)
			if got != tt.want {
				t.Fatalf(
					"EffectiveWorkerCount(%d, %d, %d) = %d, want %d",
					tt.requested,
					tt.maxJobs,
					tt.fallback,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestEffectiveWorkerDecisionHonorsMemoryBudget(t *testing.T) {
	decision := EffectiveWorkerDecision(
		4,
		6,
		8,
		128*1024*1024,
		256*1024*1024,
	)
	if decision.Count != 1 {
		t.Fatalf("Count = %d, want 1; reason=%s", decision.Count, decision.Reason)
	}
	if decision.Reason == "" {
		t.Fatalf("Reason is empty")
	}
}

func TestEffectiveWorkerDecisionKeepsRequestedWhenBudgetAllows(t *testing.T) {
	decision := EffectiveWorkerDecision(
		3,
		6,
		8,
		1024*1024*1024,
		256*1024*1024,
	)
	if decision.Count != 3 {
		t.Fatalf("Count = %d, want 3; reason=%s", decision.Count, decision.Reason)
	}
}

func TestApplyModuleObjectMetadata(t *testing.T) {
	var srcHash [32]byte
	srcHash[0] = 0x11
	var depHash [32]byte
	depHash[0] = 0x22

	obj := &tobj.Object{}
	ApplyModuleObjectMetadata(obj, ModuleObjectMetadata{
		Target:          "linux-x64",
		Module:          "app.main",
		CompilerVersion: "test-version",
		PublicAPIHash:   "api-hash",
		SrcHash:         srcHash,
		WorldSigHash:    depHash,
	})

	if obj.Target != "linux-x64" || obj.Module != "app.main" {
		t.Fatalf("identity = (%q, %q), want linux-x64/app.main", obj.Target, obj.Module)
	}
	if obj.CompilerVersion != "test-version" || obj.PublicAPIHash != "api-hash" {
		t.Fatalf(
			"metadata = (%q, %q), want test-version/api-hash",
			obj.CompilerVersion,
			obj.PublicAPIHash,
		)
	}
	if obj.SrcHash != srcHash || obj.WorldSigHash != depHash {
		t.Fatalf("hash metadata was not applied")
	}
}
