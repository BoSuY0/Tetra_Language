package x64

import (
	"strings"
	"testing"
)

func TestTargetFeatureModelUsesPortableBaselineAndRejectsUnsafeDrift(t *testing.T) {
	baseline, err := (CodegenOptions{RegisterWidthBits: 64}).EffectiveTargetFeatures()
	if err != nil {
		t.Fatalf("EffectiveTargetFeatures baseline: %v", err)
	}
	if baseline.Source != TargetFeatureSourcePortableBaseline {
		t.Fatalf("baseline source = %q, want portable baseline", baseline.Source)
	}
	if !baseline.Has(TargetFeatureSSE2) {
		t.Fatalf("x64 portable baseline should include sse2: %#v", baseline)
	}
	if baseline.Has(TargetFeatureAVX2) {
		t.Fatalf("portable baseline should not include avx2: %#v", baseline)
	}

	legacy, err := (CodegenOptions{RegisterWidthBits: 32}).EffectiveTargetFeatures()
	if err != nil {
		t.Fatalf("EffectiveTargetFeatures legacy baseline: %v", err)
	}
	if legacy.Source != TargetFeatureSourcePortableBaseline || len(legacy.Features) != 0 {
		t.Fatalf("32-bit portable baseline = %#v, want empty portable baseline", legacy)
	}

	explicit, err := (CodegenOptions{
		RegisterWidthBits: 64,
		TargetFeatures: TargetFeatures{
			Source:   TargetFeatureSourceExplicit,
			Features: []TargetFeature{TargetFeatureAVX2, TargetFeatureSSE2, TargetFeatureAVX2},
		},
	}).EffectiveTargetFeatures()
	if err != nil {
		t.Fatalf("EffectiveTargetFeatures explicit: %v", err)
	}
	if explicit.Source != TargetFeatureSourceExplicit {
		t.Fatalf("explicit source = %q", explicit.Source)
	}
	if got := explicit.FeatureNames(); strings.Join(got, ",") != "avx2,sse2" {
		t.Fatalf("explicit features = %#v, want canonical avx2,sse2", got)
	}

	_, err = (CodegenOptions{
		RegisterWidthBits: 64,
		TargetFeatures: TargetFeatures{
			Source:   TargetFeatureSourceExplicit,
			Features: []TargetFeature{TargetFeatureAVX2},
		},
	}).EffectiveTargetFeatures()
	if err == nil || !strings.Contains(err.Error(), "portable baseline") {
		t.Fatalf("explicit x64 features without sse2 error = %v, want portable baseline rejection", err)
	}

	_, err = (CodegenOptions{
		RegisterWidthBits: 64,
		TargetFeatures: TargetFeatures{
			Source:   TargetFeatureSourceExplicit,
			Features: []TargetFeature{"future_magic"},
		},
	}).EffectiveTargetFeatures()
	if err == nil || !strings.Contains(err.Error(), "unknown target feature") {
		t.Fatalf("unknown target feature error = %v", err)
	}
}

func TestCodegenOptionsTargetFeatureGuardIsEvidenceOnly(t *testing.T) {
	opt := CodegenOptions{RegisterWidthBits: 64}
	evidence, err := opt.TargetFeatureEvidence()
	if err != nil {
		t.Fatalf("TargetFeatureEvidence: %v", err)
	}
	if evidence.Source != string(TargetFeatureSourcePortableBaseline) {
		t.Fatalf("evidence source = %q", evidence.Source)
	}
	if !evidence.PortableBaselineFallback || !containsTargetFeature(evidence.Features, string(TargetFeatureSSE2)) {
		t.Fatalf("evidence missing portable baseline sse2: %#v", evidence)
	}
	if evidence.ChangesSafeSemantics || evidence.EnablesTargetSpecificOptimization {
		t.Fatalf("target feature evidence must be semantics-neutral and not enable tuning: %#v", evidence)
	}

	if ok, err := opt.AllowsTargetFeature(TargetFeatureSSE2); err != nil || !ok {
		t.Fatalf("AllowsTargetFeature(sse2) = %v, %v; want true, nil", ok, err)
	}
	if ok, err := opt.AllowsTargetFeature(TargetFeatureAVX2); err != nil || ok {
		t.Fatalf("AllowsTargetFeature(avx2) = %v, %v; want false, nil", ok, err)
	}
	if _, err := opt.AllowsTargetFeature("future_magic"); err == nil {
		t.Fatalf("AllowsTargetFeature accepted unknown feature")
	}
}

func containsTargetFeature(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
