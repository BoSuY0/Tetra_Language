package main

import (
	"encoding/json"
	"strings"
	"testing"

	"tetra_language/internal/toon"
)

const (
	cliContractsDoc     = "docs/spec/policy/cli_contracts.md"
	surfaceV1Doc        = "docs/spec/surface/surface_v1.md"
	surfaceMorphDoc     = "docs/spec/surface/morph/surface_morph.md"
	uiV1Doc             = "docs/spec/ui/ui_v1.md"
	v1ScopeDoc          = "docs/spec/flow/v1_scope.md"
	postV1ChecklistDoc  = "docs/release/policy/post_v1_promotion_checklist.md"
	standardLibraryDoc  = "docs/user/platform/standard_library_guide.md"
	wasmSurfaceGuideDoc = "docs/user/surface/wasm_ui_guide.md"
)

func validFeaturesReportJSON() []byte {
	return []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.3.0",
  "features": [
    {
      "id": "cli.core",
      "name": "Core CLI workflows",
      "status": "current",
      "since": "v0.2.0",
      "scope": "local workflows",
      "stability": "supported",
      "docs": ["docs/spec/policy/cli_contracts.md"]
    },
    {
      "id": "stdlib.experimental-mirrors",
      "name": "Experimental standard-library mirrors",
      "status": "experimental",
      "since": "v0.2.0",
      "scope": "mirrors",
      "stability": "experimental",
      "docs": ["docs/user/platform/standard_library_guide.md"]
    },
    {
      "id": "wasm.runtime-execution",
      "name": "WASM runtime execution",
      "status": "planned",
      "scope": "runner automation",
      "stability": "planned",
      "docs": ["docs/user/surface/wasm_ui_guide.md"]
    },
    {
      "id": "eco.distributed-network",
      "name": "Distributed EcoNet",
      "status": "post-v1",
      "scope": "distributed publishing",
      "stability": "deferred",
      "docs": ["docs/release/policy/post_v1_promotion_checklist.md"]
    }
  ]
}`)
}

func TestValidateFeaturesReportAcceptsExpectedShape(t *testing.T) {
	if err := validateFeaturesReport(validFeaturesReportJSON()); err != nil {
		t.Fatalf("validate features: %v", err)
	}
}

func TestValidateFeaturesReportAcceptsTOON(t *testing.T) {
	raw, err := toon.ConvertJSONToTOON(
		validFeaturesReportJSON(),
		toon.Options{Deterministic: true, Strict: true},
	)
	if err != nil {
		t.Fatalf("json->toon: %v", err)
	}
	if err := validateFeaturesReport(raw); err != nil {
		t.Fatalf("validate features TOON: %v\n%s", err, raw)
	}
}

func TestValidateFeaturesReportAcceptsSurfaceReleaseStatusVocabulary(t *testing.T) {
	raw := featuresReportJSON(
		t,
		"surface-v1",
		cliCoreFeature("current", "v0.2.0"),
		releaseCandidateSurfaceCoreFeature(),
		surfaceBlockSystemFeature(),
		surfaceMorphCapsuleFeature(),
		unsupportedSurfaceMacOSFeature(),
		legacyUIMetadataFeature(),
		fullV1GuaranteesFeature(),
		ecoDistributedNetworkFeature(),
	)
	if err := validateFeaturesReport(raw); err != nil {
		t.Fatalf("validate release status vocabulary: %v", err)
	}
}

func TestValidateFeaturesReportRequiresSurfaceBlockSystemWhenSurfaceCorePresent(t *testing.T) {
	raw := featuresReportJSON(
		t,
		"surface-v1",
		cliCoreFeature("current", "v0.2.0"),
		currentSurfaceCoreFeature(),
		fullV1GuaranteesFeature(),
		ecoDistributedNetworkFeature(),
	)
	err := validateFeaturesReport(raw)
	if err == nil {
		t.Fatalf("expected missing Surface Block System feature failure")
	}
	if !strings.Contains(err.Error(), "ui.surface-block-system") {
		t.Fatalf("error = %v, want ui.surface-block-system", err)
	}
}

func TestValidateFeaturesReportRequiresSurfaceMorphCapsuleWhenSurfaceCorePresent(t *testing.T) {
	raw := featuresReportJSON(
		t,
		"surface-v1",
		cliCoreFeature("current", "v0.2.0"),
		currentSurfaceCoreFeature(),
		surfaceBlockSystemFeature(),
		fullV1GuaranteesFeature(),
		ecoDistributedNetworkFeature(),
	)
	err := validateFeaturesReport(raw)
	if err == nil {
		t.Fatalf("expected missing Surface Morph Capsule feature failure")
	}
	if !strings.Contains(err.Error(), "ui.surface-morph-capsule") {
		t.Fatalf("error = %v, want ui.surface-morph-capsule", err)
	}
}

func TestValidateFeaturesReportRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{"schema":"tetra.features.v1","version":"v0.3.0","features":[],"extra":true}`)
	if err := validateFeaturesReport(raw); err == nil ||
		!strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown top-level field failure, got %v", err)
	}

	raw = []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.3.0",
  "features": [
    {
      "id": "cli.core",
      "name": "Core CLI workflows",
      "status": "current",
      "since": "v0.2.0",
      "scope": "local workflows",
      "stability": "supported",
      "docs": ["docs/spec/policy/cli_contracts.md"],
      "extra": true
    }
  ]
}`)
	if err := validateFeaturesReport(raw); err == nil ||
		!strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown nested field failure, got %v", err)
	}
}

func TestValidateFeaturesReportRejectsTrailingJSON(t *testing.T) {
	raw := append(validFeaturesReportJSON(), []byte(`{"schema":"tetra.features.v1"}`)...)
	if err := validateFeaturesReport(raw); err == nil {
		t.Fatalf("expected trailing JSON failure")
	}
}

func TestValidateFeaturesReportRejectsInvalidSchema(t *testing.T) {
	raw := []byte(`{"schema":"tetra.features.v2","version":"v0.3.0","features":[]}`)
	if err := validateFeaturesReport(raw); err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("expected schema failure, got %v", err)
	}
}

func TestValidateFeaturesReportRejectsInvalidStatus(t *testing.T) {
	raw := featuresReportJSON(
		t,
		"v0.3.0",
		cliCoreFeature("stable", "v0.2.0"),
	)
	if err := validateFeaturesReport(raw); err == nil ||
		!strings.Contains(err.Error(), "invalid status") {
		t.Fatalf("expected status failure, got %v", err)
	}
}

func TestValidateFeaturesReportAcceptsRegistryWithoutExperimentalStatus(t *testing.T) {
	raw := featuresReportJSON(
		t,
		"v0.4.0",
		cliCoreFeature("current", "v0.2.0"),
		fullV1GuaranteesFeature(),
		ecoDistributedNetworkFeature(),
	)
	if err := validateFeaturesReport(raw); err != nil {
		t.Fatalf("validate features without experimental status: %v", err)
	}
}

func TestValidateFeaturesReportRejectsMissingRequiredStatusCategory(t *testing.T) {
	raw := featuresReportJSON(
		t,
		"v0.3.0",
		cliCoreFeature("current", "v0.2.0"),
	)
	if err := validateFeaturesReport(raw); err == nil ||
		!strings.Contains(err.Error(), "missing planned status") {
		t.Fatalf("expected missing status category failure, got %v", err)
	}
}

func TestValidateFeaturesReportRejectsDuplicateIDs(t *testing.T) {
	raw := featuresReportJSON(
		t,
		"v0.3.0",
		cliCoreFeature("current", "v0.2.0"),
		featureEntry{
			ID:        "cli.core",
			Name:      "Core CLI workflows again",
			Status:    "planned",
			Scope:     "later",
			Stability: "planned",
			Docs:      []string{cliContractsDoc},
		},
	)
	if err := validateFeaturesReport(raw); err == nil ||
		!strings.Contains(err.Error(), "duplicate feature cli.core") {
		t.Fatalf("expected duplicate id failure, got %v", err)
	}
}

func TestValidateFeaturesReportRejectsUnsafeDocPaths(t *testing.T) {
	raw := featuresReportJSON(
		t,
		"v0.3.0",
		cliCoreFeatureWithDoc("current", "v0.2.0", "../README.md"),
	)
	if err := validateFeaturesReport(raw); err == nil ||
		!strings.Contains(err.Error(), "unsafe doc reference") {
		t.Fatalf("expected unsafe doc path failure, got %v", err)
	}
}

func TestValidateFeaturesReportRejectsNonDocsMarkdownPath(t *testing.T) {
	raw := featuresReportJSON(
		t,
		"v0.3.0",
		cliCoreFeatureWithDoc("current", "v0.2.0", "README.md"),
	)
	if err := validateFeaturesReport(raw); err == nil ||
		!strings.Contains(err.Error(), "must point at docs/*.md") {
		t.Fatalf("expected docs/*.md failure, got %v", err)
	}
}

func TestValidateFeaturesReportRejectsMissingDocFile(t *testing.T) {
	raw := featuresReportJSON(
		t,
		"v0.3.0",
		cliCoreFeatureWithDoc("current", "v0.2.0", "docs/spec/does_not_exist.md"),
	)
	if err := validateFeaturesReport(raw); err == nil ||
		!strings.Contains(err.Error(), "is not readable") {
		t.Fatalf("expected missing doc file failure, got %v", err)
	}
}

func TestValidateFeaturesReportRejectsCurrentFeatureWithoutSince(t *testing.T) {
	raw := featuresReportJSON(
		t,
		"v0.3.0",
		cliCoreFeature("current", ""),
	)
	if err := validateFeaturesReport(raw); err == nil ||
		!strings.Contains(err.Error(), "missing since") {
		t.Fatalf("expected missing since failure, got %v", err)
	}
}

func featuresReportJSON(
	t *testing.T,
	version string,
	features ...featureEntry,
) []byte {
	t.Helper()
	raw, err := json.Marshal(featuresReport{
		Schema:   "tetra.features.v1",
		Version:  version,
		Features: features,
	})
	if err != nil {
		t.Fatalf("marshal features report: %v", err)
	}
	return raw
}

func cliCoreFeature(status string, since string) featureEntry {
	return cliCoreFeatureWithDoc(status, since, cliContractsDoc)
}

func cliCoreFeatureWithDoc(status string, since string, doc string) featureEntry {
	return featureEntry{
		ID:        "cli.core",
		Name:      "Core CLI workflows",
		Status:    status,
		Since:     since,
		Scope:     "local workflows",
		Stability: "supported",
		Docs:      []string{doc},
	}
}

func releaseCandidateSurfaceCoreFeature() featureEntry {
	return featureEntry{
		ID:        "ui.surface-core",
		Name:      "Tetra Surface core",
		Status:    "release_candidate",
		Scope:     "surface-v1-linux-web",
		Stability: "release gate candidate",
		Docs:      []string{surfaceV1Doc},
	}
}

func currentSurfaceCoreFeature() featureEntry {
	return featureEntry{
		ID:        "ui.surface-core",
		Name:      "Tetra Surface core",
		Status:    "current",
		Since:     "surface-v1",
		Scope:     "surface-v1-linux-web",
		Stability: "current bounded Surface release",
		Docs:      []string{surfaceV1Doc},
	}
}

func surfaceBlockSystemFeature() featureEntry {
	return featureEntry{
		ID:     "ui.surface-block-system",
		Name:   "Tetra Surface Block System",
		Status: "experimental",
		Scope: "Block-first Surface architecture with Block as the core " +
			"Surface primitive and widgets as recipes/compatibility",
		Stability: "implementation track; not current; no production Block claim",
		Docs:      []string{surfaceV1Doc},
	}
}

func surfaceMorphCapsuleFeature() featureEntry {
	return featureEntry{
		ID:     "ui.surface-morph-capsule",
		Name:   "Tetra Surface Morph Capsule",
		Status: "experimental",
		Scope: "Morph Capsule layer that expands into Block evidence and is " +
			"validated by tetra.surface.morph.gate.v1",
		Stability: "experimental; not Surface v1 production support; " +
			"does not add core widget primitives",
		Docs: []string{surfaceMorphDoc},
	}
}

func unsupportedSurfaceMacOSFeature() featureEntry {
	return featureEntry{
		ID:        "ui.surface-macos-x64",
		Name:      "macOS Surface host",
		Status:    "unsupported",
		Scope:     "not in Surface v1",
		Stability: "no release evidence",
		Docs:      []string{surfaceV1Doc},
	}
}

func legacyUIMetadataFeature() featureEntry {
	return featureEntry{
		ID:        "ui.metadata-v1",
		Name:      "UI metadata v1 surface",
		Status:    "legacy_compatibility",
		Scope:     "legacy metadata compatibility",
		Stability: "compatibility bridge",
		Docs:      []string{uiV1Doc},
	}
}

func fullV1GuaranteesFeature() featureEntry {
	return featureEntry{
		ID:        "language.full-v1-guarantees",
		Name:      "Full v1.0 language guarantees",
		Status:    "planned",
		Scope:     "complete release contract",
		Stability: "planned",
		Docs:      []string{v1ScopeDoc},
	}
}

func ecoDistributedNetworkFeature() featureEntry {
	return featureEntry{
		ID:        "eco.distributed-network",
		Name:      "Distributed EcoNet",
		Status:    "post-v1",
		Scope:     "distributed publishing",
		Stability: "deferred",
		Docs:      []string{postV1ChecklistDoc},
	}
}
