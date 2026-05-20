package main

import (
	"strings"
	"testing"
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
      "docs": ["docs/spec/cli_contracts.md"]
    },
    {
      "id": "stdlib.experimental-mirrors",
      "name": "Experimental standard-library mirrors",
      "status": "experimental",
      "since": "v0.2.0",
      "scope": "mirrors",
      "stability": "experimental",
      "docs": ["docs/user/standard_library_guide.md"]
    },
    {
      "id": "wasm.runtime-execution",
      "name": "WASM runtime execution",
      "status": "planned",
      "scope": "runner automation",
      "stability": "planned",
      "docs": ["docs/user/wasm_ui_guide.md"]
    },
    {
      "id": "eco.distributed-network",
      "name": "Distributed EcoNet",
      "status": "post-v1",
      "scope": "distributed publishing",
      "stability": "deferred",
      "docs": ["docs/release/post_v1_promotion_checklist.md"]
    }
  ]
}`)
}

func TestValidateFeaturesReportAcceptsExpectedShape(t *testing.T) {
	if err := validateFeaturesReport(validFeaturesReportJSON()); err != nil {
		t.Fatalf("validate features: %v", err)
	}
}

func TestValidateFeaturesReportRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{"schema":"tetra.features.v1","version":"v0.3.0","features":[],"extra":true}`)
	if err := validateFeaturesReport(raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown top-level field failure, got %v", err)
	}

	raw = []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.3.0",
  "features": [
    {"id":"cli.core","name":"Core CLI workflows","status":"current","since":"v0.2.0","scope":"local workflows","stability":"supported","docs":["docs/spec/cli_contracts.md"],"extra":true}
  ]
}`)
	if err := validateFeaturesReport(raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
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
	raw := []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.3.0",
  "features": [
    {"id":"cli.core","name":"Core CLI workflows","status":"stable","since":"v0.2.0","scope":"local workflows","stability":"supported","docs":["docs/spec/cli_contracts.md"]}
  ]
}`)
	if err := validateFeaturesReport(raw); err == nil || !strings.Contains(err.Error(), "invalid status") {
		t.Fatalf("expected status failure, got %v", err)
	}
}

func TestValidateFeaturesReportAcceptsRegistryWithoutExperimentalStatus(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.4.0",
  "features": [
    {"id":"cli.core","name":"Core CLI workflows","status":"current","since":"v0.2.0","scope":"local workflows","stability":"supported","docs":["docs/spec/cli_contracts.md"]},
    {"id":"language.full-v1-guarantees","name":"Full v1.0 language guarantees","status":"planned","scope":"complete release contract","stability":"planned","docs":["docs/spec/v1_scope.md"]},
    {"id":"eco.distributed-network","name":"Distributed EcoNet","status":"post-v1","scope":"distributed publishing","stability":"deferred","docs":["docs/release/post_v1_promotion_checklist.md"]}
  ]
}`)
	if err := validateFeaturesReport(raw); err != nil {
		t.Fatalf("validate features without experimental status: %v", err)
	}
}

func TestValidateFeaturesReportRejectsMissingRequiredStatusCategory(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.3.0",
  "features": [
    {"id":"cli.core","name":"Core CLI workflows","status":"current","since":"v0.2.0","scope":"local workflows","stability":"supported","docs":["docs/spec/cli_contracts.md"]}
  ]
}`)
	if err := validateFeaturesReport(raw); err == nil || !strings.Contains(err.Error(), "missing planned status") {
		t.Fatalf("expected missing status category failure, got %v", err)
	}
}

func TestValidateFeaturesReportRejectsDuplicateIDs(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.3.0",
  "features": [
    {"id":"cli.core","name":"Core CLI workflows","status":"current","since":"v0.2.0","scope":"local workflows","stability":"supported","docs":["docs/spec/cli_contracts.md"]},
    {"id":"cli.core","name":"Core CLI workflows again","status":"planned","scope":"later","stability":"planned","docs":["docs/spec/cli_contracts.md"]}
  ]
}`)
	if err := validateFeaturesReport(raw); err == nil || !strings.Contains(err.Error(), "duplicate feature cli.core") {
		t.Fatalf("expected duplicate id failure, got %v", err)
	}
}

func TestValidateFeaturesReportRejectsUnsafeDocPaths(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.3.0",
  "features": [
    {"id":"cli.core","name":"Core CLI workflows","status":"current","since":"v0.2.0","scope":"local workflows","stability":"supported","docs":["../README.md"]}
  ]
}`)
	if err := validateFeaturesReport(raw); err == nil || !strings.Contains(err.Error(), "unsafe doc reference") {
		t.Fatalf("expected unsafe doc path failure, got %v", err)
	}
}

func TestValidateFeaturesReportRejectsNonDocsMarkdownPath(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.3.0",
  "features": [
    {"id":"cli.core","name":"Core CLI workflows","status":"current","since":"v0.2.0","scope":"local workflows","stability":"supported","docs":["README.md"]}
  ]
}`)
	if err := validateFeaturesReport(raw); err == nil || !strings.Contains(err.Error(), "must point at docs/*.md") {
		t.Fatalf("expected docs/*.md failure, got %v", err)
	}
}

func TestValidateFeaturesReportRejectsMissingDocFile(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.3.0",
  "features": [
    {"id":"cli.core","name":"Core CLI workflows","status":"current","since":"v0.2.0","scope":"local workflows","stability":"supported","docs":["docs/spec/does_not_exist.md"]}
  ]
}`)
	if err := validateFeaturesReport(raw); err == nil || !strings.Contains(err.Error(), "is not readable") {
		t.Fatalf("expected missing doc file failure, got %v", err)
	}
}

func TestValidateFeaturesReportRejectsCurrentFeatureWithoutSince(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.3.0",
  "features": [
    {"id":"cli.core","name":"Core CLI workflows","status":"current","scope":"local workflows","stability":"supported","docs":["docs/spec/cli_contracts.md"]}
  ]
}`)
	if err := validateFeaturesReport(raw); err == nil || !strings.Contains(err.Error(), "missing since") {
		t.Fatalf("expected missing since failure, got %v", err)
	}
}
