package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type safetyEvidence struct {
	Features       []byte
	CurrentSurface []byte
	OwnershipSpec  []byte
	EffectsSpec    []byte
}

type featuresReport struct {
	Schema   string         `json:"schema"`
	Version  string         `json:"version"`
	Features []featureEntry `json:"features"`
}

type featureEntry struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Since     string   `json:"since,omitempty"`
	Scope     string   `json:"scope"`
	Stability string   `json:"stability"`
	Docs      []string `json:"docs"`
}

type safetyReport struct {
	Schema           string   `json:"schema"`
	Status           string   `json:"status"`
	Version          string   `json:"version"`
	RequiredFeatures []string `json:"required_features"`
}

var requiredSafetyFeatures = []string{
	"safety.production-core",
	"safety.effects-mvp",
	"safety.capabilities-mvp",
	"safety.privacy-consent-mvp",
	"safety.budget-mvp",
	"language.ownership-markers-mvp",
	"language.resource-lifetime-mvp",
	"language.lifetime-ssa",
	"actors.task-transfer-safety",
	"language.full-first-class-callables",
}

var productionCoreRequiredPhrases = []string{
	"production local safety model",
	"ownership/lifetime/borrow/consume/inout",
	"resource finalization",
	"callable escape diagnostics",
	"effects/capabilities/privacy/consent/budget",
	"unsafe boundaries",
	"actor/task transfer safety",
	"pointer/MMIO/memory capability gates",
	"explicit diagnostics",
}

var forbiddenSafetyEvidencePhrases = []string{
	"placeholder",
	"mock",
	"fake",
	"Lifetime SSA solving is planned future work",
	"not a full SSA lifetime solver",
}

func main() {
	featuresPath := flag.String(
		"features",
		"",
		"features JSON produced by ./tetra features --format=json",
	)
	currentSurfacePath := flag.String(
		"current-surface",
		"docs/spec/core/current_supported_surface.md",
		"current supported surface docs",
	)
	ownershipSpecPath := flag.String(
		"ownership-spec",
		"docs/spec/runtime/ownership_v1.md",
		"ownership/lifetime safety docs",
	)
	effectsSpecPath := flag.String(
		"effects-spec",
		"docs/spec/runtime/effects_capabilities_privacy_v1.md",
		"effects/capabilities/privacy/budget docs",
	)
	outPath := flag.String("out", "", "optional JSON report path")
	flag.Parse()

	evidence, err := readSafetyEvidence(
		*featuresPath,
		*currentSurfacePath,
		*ownershipSpecPath,
		*effectsSpecPath,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate-safety-readiness: %v\n", err)
		os.Exit(2)
	}
	report, err := validateSafetyReadinessReport(evidence)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate-safety-readiness: %v\n", err)
		os.Exit(1)
	}
	if strings.TrimSpace(*outPath) != "" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "validate-safety-readiness: %v\n", err)
			os.Exit(1)
		}
		data = append(data, '\n')
		if err := os.WriteFile(*outPath, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "validate-safety-readiness: %v\n", err)
			os.Exit(1)
		}
	}
}

func readSafetyEvidence(
	featuresPath, currentSurfacePath, ownershipSpecPath, effectsSpecPath string,
) (safetyEvidence, error) {
	readRequired := func(path, label string) ([]byte, error) {
		if strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("%s path is required", label)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", label, err)
		}
		return raw, nil
	}
	features, err := readRequired(featuresPath, "features")
	if err != nil {
		return safetyEvidence{}, err
	}
	currentSurface, err := readRequired(currentSurfacePath, "current surface")
	if err != nil {
		return safetyEvidence{}, err
	}
	ownershipSpec, err := readRequired(ownershipSpecPath, "ownership spec")
	if err != nil {
		return safetyEvidence{}, err
	}
	effectsSpec, err := readRequired(effectsSpecPath, "effects spec")
	if err != nil {
		return safetyEvidence{}, err
	}
	return safetyEvidence{
		Features:       features,
		CurrentSurface: currentSurface,
		OwnershipSpec:  ownershipSpec,
		EffectsSpec:    effectsSpec,
	}, nil
}

func validateSafetyReadiness(evidence safetyEvidence) error {
	_, err := validateSafetyReadinessReport(evidence)
	return err
}

func validateSafetyReadinessReport(evidence safetyEvidence) (safetyReport, error) {
	var issues []string

	report, err := decodeFeaturesReport(evidence.Features)
	if err != nil {
		return safetyReport{}, err
	}
	featuresByID := map[string]featureEntry{}
	for _, feature := range report.Features {
		featuresByID[feature.ID] = feature
	}
	for _, id := range requiredSafetyFeatures {
		feature, ok := featuresByID[id]
		if !ok {
			issues = append(issues, fmt.Sprintf("features missing %s", id))
			continue
		}
		if feature.Status != "current" {
			issues = append(
				issues,
				fmt.Sprintf("feature %s status = %s, want current", id, feature.Status),
			)
		}
		if strings.TrimSpace(feature.Since) == "" {
			issues = append(issues, fmt.Sprintf("current feature %s missing since", id))
		}
	}
	if feature, ok := featuresByID["safety.production-core"]; ok {
		haystack := feature.Scope + " " + feature.Stability
		for _, phrase := range productionCoreRequiredPhrases {
			if !strings.Contains(haystack, phrase) {
				issues = append(
					issues,
					fmt.Sprintf("feature safety.production-core missing phrase %q", phrase),
				)
			}
		}
		for _, phrase := range []string{"MVP", "placeholder", "mock", "fake"} {
			if strings.Contains(strings.ToLower(haystack), strings.ToLower(phrase)) {
				issues = append(
					issues,
					fmt.Sprintf(
						"feature safety.production-core contains production-blocking phrase %q",
						phrase,
					),
				)
			}
		}
	}

	docs := map[string][]byte{
		"docs/spec/core/current_supported_surface.md":          evidence.CurrentSurface,
		"docs/spec/runtime/ownership_v1.md":                    evidence.OwnershipSpec,
		"docs/spec/runtime/effects_capabilities_privacy_v1.md": evidence.EffectsSpec,
	}
	for path, raw := range docs {
		text := string(raw)
		for _, phrase := range forbiddenSafetyEvidencePhrases {
			if containsForbiddenSafetyClaim(text, phrase) {
				issues = append(
					issues,
					fmt.Sprintf("%s contains production-blocking phrase %q", path, phrase),
				)
			}
		}
	}
	issues = append(
		issues,
		requireDocPhrases(
			"docs/spec/core/current_supported_surface.md",
			string(evidence.CurrentSurface),
			[]string{
				"Safety production core is current",
				"Lifetime SSA local join solver is current since `v0.4.0`",
				"Mutable by-reference captures, including callable mutable-capture",
				"stable JSON diagnostics",
			},
		)...)
	issues = append(
		issues,
		requireDocPhrases(
			"docs/spec/runtime/ownership_v1.md",
			string(evidence.OwnershipSpec),
			[]string{
				"current production surface",
				"SSA-like for branch, match, and loop joins",
				"borrow escape diagnostics",
				"use-after-transfer diagnostics",
				"worker effect boundary",
			},
		)...)
	issues = append(
		issues,
		requireDocPhrases(
			"docs/spec/runtime/effects_capabilities_privacy_v1.md",
			string(evidence.EffectsSpec),
			[]string{
				"Canonical `uses` effect names",
				"Unsafe Policy Public API Boundary",
				"Privacy And Consent",
				"Budget exhaustion uses the stable local policy-failure ABI",
				"Pointer/MMIO/memory operations require matching `uses` effects",
			},
		)...)

	if len(issues) > 0 {
		return safetyReport{}, errors.New(strings.Join(issues, "; "))
	}
	return safetyReport{
		Schema:           "tetra.safety-readiness.v1",
		Status:           "pass",
		Version:          report.Version,
		RequiredFeatures: append([]string(nil), requiredSafetyFeatures...),
	}, nil
}

func containsForbiddenSafetyClaim(text string, phrase string) bool {
	needle := strings.ToLower(phrase)
	for _, paragraph := range strings.Split(text, "\n\n") {
		lower := strings.ToLower(paragraph)
		if !strings.Contains(lower, needle) {
			continue
		}
		if isValidatorRejectionParagraph(lower) {
			continue
		}
		return true
	}
	return false
}

func isValidatorRejectionParagraph(paragraph string) bool {
	for _, marker := range []string{
		"rejects",
		"reject ",
		"rejected",
		"must reject",
		"validator rejects",
	} {
		if strings.Contains(paragraph, marker) {
			return true
		}
	}
	return false
}

func decodeFeaturesReport(raw []byte) (featuresReport, error) {
	var report featuresReport
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&report); err != nil {
		return featuresReport{}, fmt.Errorf("invalid features JSON: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return featuresReport{}, fmt.Errorf(
				"invalid features JSON: unexpected trailing JSON value",
			)
		}
		return featuresReport{}, fmt.Errorf("invalid features JSON: %w", err)
	}
	if report.Schema != "tetra.features.v1" {
		return featuresReport{}, fmt.Errorf(
			"features schema = %q, want tetra.features.v1",
			report.Schema,
		)
	}
	if strings.TrimSpace(report.Version) == "" {
		return featuresReport{}, fmt.Errorf("features version is required")
	}
	return report, nil
}

func requireDocPhrases(path, text string, phrases []string) []string {
	var issues []string
	for _, phrase := range phrases {
		if !strings.Contains(text, phrase) {
			issues = append(issues, fmt.Sprintf("%s missing phrase %q", path, phrase))
		}
	}
	return issues
}
