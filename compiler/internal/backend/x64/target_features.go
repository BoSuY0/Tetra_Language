package x64

import (
	"fmt"
	"sort"
	"strings"
)

type TargetFeature string

const (
	TargetFeatureSSE2  TargetFeature = "sse2"
	TargetFeatureSSE41 TargetFeature = "sse4.1"
	TargetFeatureAVX2  TargetFeature = "avx2"
)

type TargetFeatureSource string

const (
	TargetFeatureSourcePortableBaseline TargetFeatureSource = "portable_baseline"
	TargetFeatureSourceExplicit         TargetFeatureSource = "explicit"
)

type TargetFeatures struct {
	Source   TargetFeatureSource `json:"source,omitempty"`
	Features []TargetFeature     `json:"features,omitempty"`
}

type TargetFeatureEvidence struct {
	Source                            string   `json:"source"`
	Features                          []string `json:"features,omitempty"`
	PortableBaselineFallback          bool     `json:"portable_baseline_fallback"`
	ChangesSafeSemantics              bool     `json:"changes_safe_semantics"`
	EnablesTargetSpecificOptimization bool     `json:"enables_target_specific_optimization"`
}

func (o CodegenOptions) EffectiveTargetFeatures() (TargetFeatures, error) {
	return ResolveTargetFeatures(o.EffectiveRegisterWidthBits(), o.TargetFeatures)
}

func (o CodegenOptions) TargetFeatureEvidence() (TargetFeatureEvidence, error) {
	features, err := o.EffectiveTargetFeatures()
	if err != nil {
		return TargetFeatureEvidence{}, err
	}
	return TargetFeatureEvidence{
		Source:                            string(features.Source),
		Features:                          features.FeatureNames(),
		PortableBaselineFallback:          features.Source == TargetFeatureSourcePortableBaseline,
		ChangesSafeSemantics:              false,
		EnablesTargetSpecificOptimization: false,
	}, nil
}

func (o CodegenOptions) AllowsTargetFeature(feature TargetFeature) (bool, error) {
	if err := validateKnownTargetFeature(feature); err != nil {
		return false, err
	}
	features, err := o.EffectiveTargetFeatures()
	if err != nil {
		return false, err
	}
	return features.Has(feature), nil
}

func ResolveTargetFeatures(
	registerWidthBits int,
	requested TargetFeatures,
) (TargetFeatures, error) {
	baseline, err := portableBaselineTargetFeatures(registerWidthBits)
	if err != nil {
		return TargetFeatures{}, err
	}
	if requested.Source == "" && len(requested.Features) == 0 {
		return TargetFeatures{
			Source:   TargetFeatureSourcePortableBaseline,
			Features: baseline,
		}, nil
	}
	source := requested.Source
	if source == "" {
		source = TargetFeatureSourceExplicit
	}
	switch source {
	case TargetFeatureSourcePortableBaseline:
		if len(requested.Features) != 0 {
			return TargetFeatures{}, fmt.Errorf(
				"target features: portable baseline source cannot include explicit features",
			)
		}
		return TargetFeatures{Source: TargetFeatureSourcePortableBaseline, Features: baseline}, nil
	case TargetFeatureSourceExplicit:
		features, err := canonicalTargetFeatures(requested.Features)
		if err != nil {
			return TargetFeatures{}, err
		}
		if err := requirePortableBaseline(features, baseline); err != nil {
			return TargetFeatures{}, err
		}
		return TargetFeatures{Source: TargetFeatureSourceExplicit, Features: features}, nil
	default:
		return TargetFeatures{}, fmt.Errorf("target features: unknown source %q", requested.Source)
	}
}

func (f TargetFeatures) Has(feature TargetFeature) bool {
	for _, candidate := range f.Features {
		if candidate == feature {
			return true
		}
	}
	return false
}

func (f TargetFeatures) FeatureNames() []string {
	out := make([]string, 0, len(f.Features))
	for _, feature := range f.Features {
		out = append(out, string(feature))
	}
	sort.Strings(out)
	return out
}

func portableBaselineTargetFeatures(registerWidthBits int) ([]TargetFeature, error) {
	switch registerWidthBits {
	case 32:
		return nil, nil
	case 64:
		return []TargetFeature{TargetFeatureSSE2}, nil
	default:
		return nil, fmt.Errorf("target features: unsupported register width %d", registerWidthBits)
	}
}

func canonicalTargetFeatures(features []TargetFeature) ([]TargetFeature, error) {
	seen := map[TargetFeature]bool{}
	out := make([]TargetFeature, 0, len(features))
	for _, feature := range features {
		normalized := TargetFeature(strings.ToLower(strings.TrimSpace(string(feature))))
		if err := validateKnownTargetFeature(normalized); err != nil {
			return nil, err
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		out = append(out, normalized)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

func validateKnownTargetFeature(feature TargetFeature) error {
	switch feature {
	case TargetFeatureSSE2, TargetFeatureSSE41, TargetFeatureAVX2:
		return nil
	default:
		return fmt.Errorf("target features: unknown target feature %q", feature)
	}
}

func requirePortableBaseline(features []TargetFeature, baseline []TargetFeature) error {
	have := TargetFeatures{Features: features}
	for _, feature := range baseline {
		if !have.Has(feature) {
			return fmt.Errorf(
				"target features: explicit set is below portable baseline: missing %s",
				feature,
			)
		}
	}
	return nil
}
