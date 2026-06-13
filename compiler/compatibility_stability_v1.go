package compiler

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	compatibilityStabilityV1Schema    = "tetra.compatibility.stability.v1"
	compatibilityStabilityV1ScopeP242 = "p24.2_compatibility_stability"

	p24CompatibilityDiagnosticWitnessID  = "stable_diagnostic_codes"
	p24CompatibilitySchemaWitnessID      = "versioned_report_schemas"
	p24CompatibilityManifestWitnessID    = "manifest_compatibility_checks"
	p24CompatibilityMigrationWitnessID   = "breaking_change_migration_guide"
	p24CompatibilityDeprecationWitnessID = "deprecation_policy"
	p24CompatibilityArtifactsWitnessID   = "compatibility_stability_artifacts"
)

type CompatibilityStabilityV1ID string

const (
	CompatibilityStableDiagnosticCodes        CompatibilityStabilityV1ID = "stable_diagnostic_codes"
	CompatibilityVersionedReportSchemas       CompatibilityStabilityV1ID = "versioned_report_schemas"
	CompatibilityManifestChecks               CompatibilityStabilityV1ID = "manifest_compatibility_checks"
	CompatibilityBreakingChangeMigrationGuide CompatibilityStabilityV1ID = "breaking_change_migration_guide"
	CompatibilityDeprecationPolicy            CompatibilityStabilityV1ID = "deprecation_policy"
)

type CompatibilityStabilityV1Report struct {
	SchemaVersion string                            `json:"schema_version"`
	Scope         string                            `json:"scope"`
	Rows          []CompatibilityStabilityV1Row     `json:"rows"`
	Witnesses     []CompatibilityStabilityV1Witness `json:"witnesses"`
	Artifacts     []CompatibilityStabilityArtifact  `json:"artifacts"`
	NonClaims     []string                          `json:"non_claims"`

	StableDiagnosticCodesReviewed       bool `json:"stable_diagnostic_codes_reviewed"`
	VersionedReportSchemasReviewed      bool `json:"versioned_report_schemas_reviewed"`
	ManifestCompatibilityChecksReviewed bool `json:"manifest_compatibility_checks_reviewed"`
	BreakingChangeMigrationGuidePresent bool `json:"breaking_change_migration_guide_present"`
	DeprecationPolicyPresent            bool `json:"deprecation_policy_present"`

	FullBackwardCompatibilityClaimed       bool `json:"full_backward_compatibility_claimed"`
	FrozenDiagnosticMessagesClaimed        bool `json:"frozen_diagnostic_messages_claimed"`
	AutomaticMigrationClaimed              bool `json:"automatic_migration_claimed"`
	ManifestABIStabilityClaimed            bool `json:"manifest_abi_stability_claimed"`
	BreakingChangesWithoutMigrationClaimed bool `json:"breaking_changes_without_migration_claimed"`
	RemovalWithoutDeprecationClaimed       bool `json:"removal_without_deprecation_claimed"`
	RuntimeBehaviorChanged                 bool `json:"runtime_behavior_changed"`
	SafeSemanticsChanged                   bool `json:"safe_semantics_changed"`
	PerformanceClaimed                     bool `json:"performance_claimed"`
}

type CompatibilityStabilityV1Row struct {
	ID         CompatibilityStabilityV1ID `json:"id"`
	Name       string                     `json:"name"`
	Status     string                     `json:"status"`
	Evidence   []string                   `json:"evidence"`
	Tests      []string                   `json:"tests"`
	Boundaries []string                   `json:"boundaries"`
	WitnessIDs []string                   `json:"witness_ids"`
}

type CompatibilityStabilityArtifact struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Present bool   `json:"present"`
}

type CompatibilityStabilityV1Witness struct {
	ID    string   `json:"id"`
	Kind  string   `json:"kind"`
	Paths []string `json:"paths,omitempty"`

	DiagnosticCodes               []string `json:"diagnostic_codes,omitempty"`
	DiagnosticRegistryCount       int      `json:"diagnostic_registry_count,omitempty"`
	DiagnosticCodesValid          bool     `json:"diagnostic_codes_valid,omitempty"`
	DiagnosticJSONValidatorStrict bool     `json:"diagnostic_json_validator_strict,omitempty"`
	DiagnosticReleaseDocsPresent  bool     `json:"diagnostic_release_docs_present,omitempty"`
	StableDiagnosticCodesReviewed bool     `json:"stable_diagnostic_codes_reviewed,omitempty"`

	SchemaIDs                      []string `json:"schema_ids,omitempty"`
	VersionedSchemaCount           int      `json:"versioned_schema_count,omitempty"`
	ReportSchemasStrict            bool     `json:"report_schemas_strict,omitempty"`
	VersionedReportSchemasReviewed bool     `json:"versioned_report_schemas_reviewed,omitempty"`

	ManifestCompilerVersion             string `json:"manifest_compiler_version,omitempty"`
	ManifestTargetCount                 int    `json:"manifest_target_count,omitempty"`
	ManifestFeatureCount                int    `json:"manifest_feature_count,omitempty"`
	ManifestRuntimeABIPresent           bool   `json:"manifest_runtime_abi_present,omitempty"`
	ManifestValidatorStrict             bool   `json:"manifest_validator_strict,omitempty"`
	ManifestFeatureRegistryLinked       bool   `json:"manifest_feature_registry_linked,omitempty"`
	ManifestRuntimeABIChecksPresent     bool   `json:"manifest_runtime_abi_checks_present,omitempty"`
	ManifestCompatibilityChecksReviewed bool   `json:"manifest_compatibility_checks_reviewed,omitempty"`

	MigrationGuidePresent               bool `json:"migration_guide_present,omitempty"`
	APIBreakingReviewPresent            bool `json:"api_breaking_review_present,omitempty"`
	PatchLineRulePresent                bool `json:"patch_line_rule_present,omitempty"`
	BreakingChangeMigrationGuidePresent bool `json:"breaking_change_migration_guide_present,omitempty"`

	DeprecationPolicyPresent   bool `json:"deprecation_policy_present,omitempty"`
	ReplacementPathRequired    bool `json:"replacement_path_required,omitempty"`
	RemovalDelayRequired       bool `json:"removal_delay_required,omitempty"`
	StdlibMajorLineRulePresent bool `json:"stdlib_major_line_rule_present,omitempty"`

	CompatibilityAuditArtifactPresent  bool `json:"compatibility_audit_artifact_present,omitempty"`
	CompatibilityDesignArtifactPresent bool `json:"compatibility_design_artifact_present,omitempty"`
	MigrationGuideArtifactPresent      bool `json:"migration_guide_artifact_present,omitempty"`
	DeprecationPolicyArtifactPresent   bool `json:"deprecation_policy_artifact_present,omitempty"`
}

func BuildP24CompatibilityStabilityV1Report() (CompatibilityStabilityV1Report, error) {
	diagnosticWitness := buildP24CompatibilityDiagnosticWitness()
	schemaWitness := buildP24CompatibilitySchemaWitness()
	manifestWitness := buildP24CompatibilityManifestWitness()
	migrationWitness := buildP24CompatibilityMigrationWitness()
	deprecationWitness := buildP24CompatibilityDeprecationWitness()
	artifacts := p24CompatibilityStabilityArtifacts()
	artifactWitness := buildP24CompatibilityArtifactsWitness(artifacts)

	report := CompatibilityStabilityV1Report{
		SchemaVersion: compatibilityStabilityV1Schema,
		Scope:         compatibilityStabilityV1ScopeP242,
		Witnesses: []CompatibilityStabilityV1Witness{
			diagnosticWitness,
			schemaWitness,
			manifestWitness,
			migrationWitness,
			deprecationWitness,
			artifactWitness,
		},
		Artifacts: artifacts,
		Rows: []CompatibilityStabilityV1Row{
			p24CompatibilityStabilityRow(CompatibilityStableDiagnosticCodes, "Stable diagnostic codes", "reviewed_current_diagnostic_surface",
				[]string{
					"DiagnosticCodeRegistry records the public diagnostic code set, including parser/frontend TETRA0001, positioned semantic/compiler TETRA2001, safety code families, lowering/IR verifier codes, target runtime diagnostics, and formatter codes.",
					"tools/cmd/validate-diagnostic validates the tetra.release.v0_2_0.diagnostic-json.v1 JSON shape with strict unknown-field rejection while release notes document TETRA0001 and TETRA2001 compatibility.",
				},
				[]string{
					"go test ./compiler -run 'P24CompatibilityStability' -count=1",
					"go test ./compiler/tests/frontend -run 'DiagnosticCodeRegistry|Diagnostic' -count=1",
					"go test ./tools/cmd/validate-diagnostic -count=1",
				},
				[]string{
					"diagnostic codes and JSON object shape are stable evidence for the current release line",
					"diagnostic messages are not frozen and may improve while retaining stable codes/severity shape where promised",
				},
				[]string{p24CompatibilityDiagnosticWitnessID}),
			p24CompatibilityStabilityRow(CompatibilityVersionedReportSchemas, "Versioned report schemas", "reviewed_schema_version_surface",
				[]string{
					"Current evidence reports carry explicit versioned schemas such as tetra.translation.validation.v2, tetra.fuzz.property.differential.v1, tetra.formal_core.v1, tetra.self_hosting.gate.v1, tetra.security.review_gate.v1, tetra.runtime.hardening.v1, and tetra.compatibility.stability.v1.",
					"Compiler and tool validators reject unexpected schema_version or schema values for report families instead of silently accepting drift.",
				},
				[]string{
					"go test ./compiler -run 'P24CompatibilityStability' -count=1",
					"go test ./tools/cmd/validate-manifest ./tools/cmd/validate-diagnostic -count=1",
				},
				[]string{
					"versioned schema evidence does not promise automatic migration for every old report",
					"private or experimental artifacts remain governed by their local validators and release docs",
				},
				[]string{p24CompatibilitySchemaWitnessID}),
			p24CompatibilityStabilityRow(CompatibilityManifestChecks, "Manifest compatibility checks", "reviewed_manifest_validator_surface",
				[]string{
					"tools/cmd/validate-manifest validates tetra.release.v0_4_0.manifest-json.v1 with strict JSON decoding, target ordering/coverage, builtin metadata, FeatureRegistry entries, and runtime ABI symbol coverage.",
					"compiler.GetManifest builds the generated manifest from Version, formats, buildable targets, builtins, runtime ABI, and FeatureRegistry data for the same branch state.",
				},
				[]string{
					"go test ./tools/cmd/validate-manifest -count=1",
					"go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json",
					"go test ./compiler/tests/semantics -run 'FeatureRegistry' -count=1",
				},
				[]string{
					"manifest compatibility checks are current-branch validator evidence, not a future runtime ABI stability promise",
					"manifest changes still require regenerated docs/generated/manifest.json and matching release notes or migration guidance",
				},
				[]string{p24CompatibilityManifestWitnessID}),
			p24CompatibilityStabilityRow(CompatibilityBreakingChangeMigrationGuide, "Breaking-change migration guide", "documented_policy_present",
				[]string{
					"docs/release/breaking-change-migration-guide.md defines triage, migration steps, diagnostic/report/manifest handling, and release-note requirements for incompatible changes.",
					"docs/spec/api_diff_policy.md marks removed/changed API entries as breaking_requires_review and keeps release gate mode at --enforce no-change until versioned API compatibility rules exist.",
				},
				[]string{
					"go test ./compiler -run 'P24CompatibilityStability' -count=1",
					"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
				},
				[]string{
					"the migration guide is a release process artifact, not automatic source rewrite tooling",
					"security exceptions still require documented compatibility impact and mitigation",
				},
				[]string{p24CompatibilityMigrationWitnessID}),
			p24CompatibilityStabilityRow(CompatibilityDeprecationPolicy, "Deprecation policy", "documented_policy_present",
				[]string{
					"docs/release/deprecation_policy.md and docs/release/v1_0_x_maintenance_policy.md require a Deprecation Policy with a replacement path plus diagnostics or documentation.",
					"Stable lib.core breaking changes wait for a later major release line; removals wait for a later minor or major line unless a security fix requires a documented exception.",
				},
				[]string{
					"go test ./compiler -run 'P24CompatibilityStability' -count=1",
					"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
				},
				[]string{
					"deprecation policy does not authorize removals without replacement and migration notes",
					"experimental surfaces can remain less stable only where docs explicitly mark them experimental",
				},
				[]string{p24CompatibilityDeprecationWitnessID}),
		},
		NonClaims: []string{
			"full backward compatibility for all future versions is not claimed",
			"diagnostic messages are not frozen",
			"automatic migration for every breaking change is not claimed",
			"manifest/runtime ABI stability beyond current validated evidence is not claimed",
			"runtime behavior does not change",
			"safe-program semantics do not change",
			"no performance claim is made",
		},
		StableDiagnosticCodesReviewed:       diagnosticWitness.StableDiagnosticCodesReviewed,
		VersionedReportSchemasReviewed:      schemaWitness.VersionedReportSchemasReviewed,
		ManifestCompatibilityChecksReviewed: manifestWitness.ManifestCompatibilityChecksReviewed,
		BreakingChangeMigrationGuidePresent: migrationWitness.BreakingChangeMigrationGuidePresent,
		DeprecationPolicyPresent:            deprecationWitness.DeprecationPolicyPresent,
	}
	if err := ValidateP24CompatibilityStabilityV1Report(report); err != nil {
		return CompatibilityStabilityV1Report{}, err
	}
	return report, nil
}

func ValidateP24CompatibilityStabilityV1Report(report CompatibilityStabilityV1Report) error {
	if report.SchemaVersion != compatibilityStabilityV1Schema {
		return fmt.Errorf("compatibility/stability v1: schema_version is %q", report.SchemaVersion)
	}
	if report.Scope != compatibilityStabilityV1ScopeP242 {
		return fmt.Errorf("compatibility/stability v1: scope is %q", report.Scope)
	}
	if report.FullBackwardCompatibilityClaimed {
		return fmt.Errorf("compatibility/stability v1: full backward compatibility claim is forbidden")
	}
	if report.FrozenDiagnosticMessagesClaimed {
		return fmt.Errorf("compatibility/stability v1: frozen diagnostic messages claim is forbidden")
	}
	if report.AutomaticMigrationClaimed {
		return fmt.Errorf("compatibility/stability v1: automatic migration claim is forbidden")
	}
	if report.ManifestABIStabilityClaimed {
		return fmt.Errorf("compatibility/stability v1: manifest/runtime ABI stability claim is forbidden")
	}
	if report.BreakingChangesWithoutMigrationClaimed {
		return fmt.Errorf("compatibility/stability v1: breaking change without migration guide claim is forbidden")
	}
	if report.RemovalWithoutDeprecationClaimed {
		return fmt.Errorf("compatibility/stability v1: removal without deprecation claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("compatibility/stability v1: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("compatibility/stability v1: safe semantics change claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("compatibility/stability v1: performance claim is forbidden")
	}
	if !report.StableDiagnosticCodesReviewed {
		return fmt.Errorf("compatibility/stability v1: stable diagnostic code review missing")
	}
	if !report.VersionedReportSchemasReviewed {
		return fmt.Errorf("compatibility/stability v1: versioned report schema review missing")
	}
	if !report.ManifestCompatibilityChecksReviewed {
		return fmt.Errorf("compatibility/stability v1: manifest compatibility checks missing")
	}
	if !report.BreakingChangeMigrationGuidePresent {
		return fmt.Errorf("compatibility/stability v1: breaking-change migration guide missing")
	}
	if !report.DeprecationPolicyPresent {
		return fmt.Errorf("compatibility/stability v1: deprecation policy missing")
	}
	for _, want := range []string{
		"full backward compatibility for all future versions is not claimed",
		"diagnostic messages are not frozen",
		"automatic migration for every breaking change is not claimed",
		"manifest/runtime ABI stability beyond current validated evidence is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p24CompatibilityStabilityHasString(report.NonClaims, want) {
			return fmt.Errorf("compatibility/stability v1: missing non-claim %q", want)
		}
	}
	if err := p24CompatibilityStabilityValidateArtifacts(report); err != nil {
		return err
	}
	return p24CompatibilityStabilityValidateRowsAndWitnesses(report.Rows, report.Witnesses)
}

func buildP24CompatibilityDiagnosticWitness() CompatibilityStabilityV1Witness {
	paths := []string{
		"compiler/diagnostics.go",
		"tools/cmd/validate-diagnostic/main.go",
		"tools/cmd/validate-diagnostic/main_test.go",
		"docs/roadmap_0_6_1_to_0_6_3.md",
		"docs/release_notes_v0_6.md",
	}
	registry := DiagnosticCodeRegistry()
	codes := make([]string, 0, len(registry))
	validCodes := true
	for code, info := range registry {
		codes = append(codes, code)
		if strings.TrimSpace(code) == "" || code != strings.TrimSpace(code) || strings.TrimSpace(info.Severity) == "" || strings.TrimSpace(info.Surface) == "" {
			validCodes = false
		}
	}
	sort.Strings(codes)
	required := []string{
		DiagnosticCodeParse,
		DiagnosticCodeSemantic,
		DiagnosticCodeSafetyOwnership,
		DiagnosticCodeSafetyLifetime,
		DiagnosticCodeSafetyEffect,
		DiagnosticCodeSafetyPrivacy,
		DiagnosticCodeSafetyBudget,
		DiagnosticCodeIRVerifier,
		DiagnosticCodeLowerUnsupported,
		DiagnosticCodeTargetRuntime,
		DiagnosticCodeFormatter,
		DiagnosticCodeFormatterCheck,
	}
	for _, code := range required {
		if _, ok := registry[code]; !ok {
			validCodes = false
		}
	}
	diagnosticStrictDecode := p24CompatibilityStabilityFileContains("tools/cmd/validate-diagnostic/main.go", "DisallowUnknownFields") ||
		(p24CompatibilityStabilityFileContains("tools/cmd/validate-diagnostic/main.go", "reportdecode.DecodeStrict") &&
			p24CompatibilityStabilityFileContains("tools/internal/reportdecode/reportdecode.go", "DisallowUnknownFields"))
	strictValidator := diagnosticStrictDecode &&
		p24CompatibilityStabilityFileContains("tools/cmd/validate-diagnostic/main.go", "tetra.release.v0_2_0.diagnostic-json.v1") &&
		p24CompatibilityStabilityFileContains("tools/cmd/validate-diagnostic/main_test.go", "TestValidateDiagnosticAcceptsStableShape")
	releaseDocs := p24CompatibilityStabilityFileContains("docs/roadmap_0_6_1_to_0_6_3.md", "TETRA0001") &&
		p24CompatibilityStabilityFileContains("docs/roadmap_0_6_1_to_0_6_3.md", "TETRA2001") &&
		p24CompatibilityStabilityFileContains("docs/release_notes_v0_6.md", "validate-diagnostic")
	return CompatibilityStabilityV1Witness{
		ID:                            p24CompatibilityDiagnosticWitnessID,
		Kind:                          "stable_diagnostic_codes",
		Paths:                         paths,
		DiagnosticCodes:               codes,
		DiagnosticRegistryCount:       len(registry),
		DiagnosticCodesValid:          validCodes,
		DiagnosticJSONValidatorStrict: strictValidator,
		DiagnosticReleaseDocsPresent:  releaseDocs,
		StableDiagnosticCodesReviewed: p24AllRepoPathsExist(paths) && len(registry) >= len(required) && validCodes && strictValidator && releaseDocs,
	}
}

func buildP24CompatibilitySchemaWitness() CompatibilityStabilityV1Witness {
	paths := []string{
		"compiler/translation_validation_v2.go",
		"compiler/fuzz_property_differential_v1.go",
		"compiler/formal_core_v1.go",
		"compiler/self_hosting_gate_v1.go",
		"compiler/security_review_gate_v1.go",
		"compiler/runtime_hardening_v1.go",
		"compiler/compatibility_stability_v1.go",
		"compiler/reports.go",
		"tools/cmd/validate-manifest/main.go",
		"tools/cmd/validate-diagnostic/main.go",
	}
	expectations := []struct {
		path   string
		schema string
	}{
		{"compiler/translation_validation_v2.go", "tetra.translation.validation.v2"},
		{"compiler/fuzz_property_differential_v1.go", "tetra.fuzz.property.differential.v1"},
		{"compiler/formal_core_v1.go", "tetra.formal_core.v1"},
		{"compiler/self_hosting_gate_v1.go", "tetra.self_hosting.gate.v1"},
		{"compiler/security_review_gate_v1.go", "tetra.security.review_gate.v1"},
		{"compiler/runtime_hardening_v1.go", "tetra.runtime.hardening.v1"},
		{"compiler/compatibility_stability_v1.go", compatibilityStabilityV1Schema},
		{"tools/cmd/validate-manifest/main.go", "tetra.release.v0_4_0.manifest-json.v1"},
		{"tools/cmd/validate-diagnostic/main.go", "tetra.release.v0_2_0.diagnostic-json.v1"},
	}
	var schemas []string
	for _, expectation := range expectations {
		if p24CompatibilityStabilityFileContains(expectation.path, expectation.schema) && p24CompatibilityStabilityLooksVersionedSchema(expectation.schema) {
			schemas = append(schemas, expectation.schema)
		}
	}
	sort.Strings(schemas)
	diagnosticSchemaStrictDecode := p24CompatibilityStabilityFileContains("tools/cmd/validate-diagnostic/main.go", "DisallowUnknownFields") ||
		(p24CompatibilityStabilityFileContains("tools/cmd/validate-diagnostic/main.go", "reportdecode.DecodeStrict") &&
			p24CompatibilityStabilityFileContains("tools/internal/reportdecode/reportdecode.go", "DisallowUnknownFields"))
	strict := p24CompatibilityStabilityFileContains("compiler/reports.go", "schema_version") &&
		p24CompatibilityStabilityFileContains("compiler/reports.go", "want 2") &&
		p24CompatibilityStabilityFileContains("compiler/reports.go", "want 3") &&
		p24CompatibilityStabilityFileContains("tools/cmd/validate-manifest/main.go", "DisallowUnknownFields") &&
		diagnosticSchemaStrictDecode
	return CompatibilityStabilityV1Witness{
		ID:                             p24CompatibilitySchemaWitnessID,
		Kind:                           "versioned_report_schemas",
		Paths:                          paths,
		SchemaIDs:                      schemas,
		VersionedSchemaCount:           len(schemas),
		ReportSchemasStrict:            strict,
		VersionedReportSchemasReviewed: p24AllRepoPathsExist(paths) && len(schemas) == len(expectations) && strict,
	}
}

func buildP24CompatibilityManifestWitness() CompatibilityStabilityV1Witness {
	paths := []string{
		"compiler/manifest.go",
		"compiler/features.go",
		"docs/generated/manifest.json",
		"tools/cmd/validate-manifest/main.go",
	}
	var manifest struct {
		CompilerVersion string            `json:"compiler_version"`
		Targets         []json.RawMessage `json:"targets"`
		Features        []json.RawMessage `json:"features"`
		RuntimeABI      map[string]any    `json:"runtime_abi"`
	}
	if raw, err := os.ReadFile(p24RepoPath("docs/generated/manifest.json")); err == nil {
		_ = json.Unmarshal(raw, &manifest)
	}
	featureRegistry := FeatureRegistry()
	featureRegistryLinked := len(featureRegistry) > 0 &&
		p24CompatibilityStabilityFileContains("compiler/manifest.go", "FeatureRegistry()") &&
		p24CompatibilityStabilityFileContains("tools/cmd/validate-manifest/main.go", "validateFeatures")
	strict := p24CompatibilityStabilityFileContains("tools/cmd/validate-manifest/main.go", "decodeStrictJSON") &&
		p24CompatibilityStabilityFileContains("tools/cmd/validate-manifest/main.go", "DisallowUnknownFields") &&
		p24CompatibilityStabilityFileContains("tools/cmd/validate-manifest/main.go", "manifestArtifact") &&
		p24CompatibilityStabilityFileContains("tools/cmd/validate-manifest/main.go", "tetra.release.v0_4_0.manifest-json.v1")
	runtimeChecks := p24CompatibilityStabilityFileContains("tools/cmd/validate-manifest/main.go", "validateRuntimeABI") &&
		p24CompatibilityStabilityFileContains("tools/cmd/validate-manifest/main.go", "ActorRuntimeTriples") &&
		p24CompatibilityStabilityFileContains("compiler/manifest.go", "RuntimeABI")
	return CompatibilityStabilityV1Witness{
		ID:                                  p24CompatibilityManifestWitnessID,
		Kind:                                "manifest_compatibility_checks",
		Paths:                               paths,
		ManifestCompilerVersion:             manifest.CompilerVersion,
		ManifestTargetCount:                 len(manifest.Targets),
		ManifestFeatureCount:                len(manifest.Features),
		ManifestRuntimeABIPresent:           len(manifest.RuntimeABI) > 0,
		ManifestValidatorStrict:             strict,
		ManifestFeatureRegistryLinked:       featureRegistryLinked,
		ManifestRuntimeABIChecksPresent:     runtimeChecks,
		ManifestCompatibilityChecksReviewed: p24AllRepoPathsExist(paths) && manifest.CompilerVersion != "" && len(manifest.Targets) > 0 && len(manifest.Features) > 0 && len(manifest.RuntimeABI) > 0 && strict && featureRegistryLinked && runtimeChecks,
	}
}

func buildP24CompatibilityMigrationWitness() CompatibilityStabilityV1Witness {
	paths := []string{
		"docs/release/breaking-change-migration-guide.md",
		"docs/spec/api_diff_policy.md",
		"docs/spec/current_supported_surface.md",
		"docs/roadmap_0_6_1_to_0_6_3.md",
	}
	guide := p24CompatibilityStabilityFileContains("docs/release/breaking-change-migration-guide.md", "Breaking Change Triage") &&
		p24CompatibilityStabilityFileContains("docs/release/breaking-change-migration-guide.md", "Migration Steps") &&
		p24CompatibilityStabilityFileContains("docs/release/breaking-change-migration-guide.md", "Report Schema") &&
		p24CompatibilityStabilityFileContains("docs/release/breaking-change-migration-guide.md", "Manifest")
	apiReview := p24CompatibilityStabilityFileContains("docs/spec/api_diff_policy.md", "breaking_requires_review") &&
		p24CompatibilityStabilityFileContains("docs/spec/api_diff_policy.md", "--enforce no-change")
	patchLine := p24CompatibilityStabilityFileContains("docs/spec/current_supported_surface.md", "Breaking language or project compatibility changes belong in a") &&
		p24CompatibilityStabilityFileContains("docs/spec/current_supported_surface.md", "later `x.0.0` line") &&
		p24CompatibilityStabilityFileContains("docs/roadmap_0_6_1_to_0_6_3.md", "Text diagnostics remain compatible")
	return CompatibilityStabilityV1Witness{
		ID:                                  p24CompatibilityMigrationWitnessID,
		Kind:                                "breaking_change_migration_guide",
		Paths:                               paths,
		MigrationGuidePresent:               guide,
		APIBreakingReviewPresent:            apiReview,
		PatchLineRulePresent:                patchLine,
		BreakingChangeMigrationGuidePresent: p24AllRepoPathsExist(paths) && guide && apiReview && patchLine,
	}
}

func buildP24CompatibilityDeprecationWitness() CompatibilityStabilityV1Witness {
	paths := []string{
		"docs/release/deprecation_policy.md",
		"docs/release/v1_0_x_maintenance_policy.md",
		"docs/spec/stdlib_naming_versioning.md",
	}
	policy := p24CompatibilityStabilityFileContains("docs/release/deprecation_policy.md", "Deprecation Policy") &&
		p24CompatibilityStabilityFileContains("docs/release/deprecation_policy.md", "replacement path") &&
		p24CompatibilityStabilityFileContains("docs/release/deprecation_policy.md", "removals wait")
	replacement := p24CompatibilityStabilityFileContains("docs/release/v1_0_x_maintenance_policy.md", "replacement path") &&
		p24CompatibilityStabilityFileContains("docs/release/v1_0_x_maintenance_policy.md", "diagnostics or documentation")
	removalDelay := p24CompatibilityStabilityFileContains("docs/release/v1_0_x_maintenance_policy.md", "removals wait for a later minor") ||
		p24CompatibilityStabilityFileContains("docs/release/v1_0_x_maintenance_policy.md", "removals wait for a later minor or major line")
	stdlibRule := p24CompatibilityStabilityFileContains("docs/spec/stdlib_naming_versioning.md", "Breaking changes to `lib.core.*` MUST wait for the next major")
	return CompatibilityStabilityV1Witness{
		ID:                         p24CompatibilityDeprecationWitnessID,
		Kind:                       "deprecation_policy",
		Paths:                      paths,
		DeprecationPolicyPresent:   p24AllRepoPathsExist(paths) && policy && replacement && removalDelay && stdlibRule,
		ReplacementPathRequired:    replacement,
		RemovalDelayRequired:       removalDelay,
		StdlibMajorLineRulePresent: stdlibRule,
	}
}

func buildP24CompatibilityArtifactsWitness(artifacts []CompatibilityStabilityArtifact) CompatibilityStabilityV1Witness {
	witness := CompatibilityStabilityV1Witness{
		ID:    p24CompatibilityArtifactsWitnessID,
		Kind:  "compatibility_stability_artifacts",
		Paths: make([]string, 0, len(artifacts)),
	}
	for _, artifact := range artifacts {
		witness.Paths = append(witness.Paths, artifact.Path)
		switch artifact.Path {
		case "docs/audits/compatibility-stability-v1.md":
			witness.CompatibilityAuditArtifactPresent = artifact.Present
		case "docs/plans/2026-06-03-p24.2-compatibility-stability-design.md":
			witness.CompatibilityDesignArtifactPresent = artifact.Present
		case "docs/release/breaking-change-migration-guide.md":
			witness.MigrationGuideArtifactPresent = artifact.Present
		case "docs/release/deprecation_policy.md":
			witness.DeprecationPolicyArtifactPresent = artifact.Present
		}
	}
	return witness
}

func p24CompatibilityStabilityValidateRowsAndWitnesses(rows []CompatibilityStabilityV1Row, witnesses []CompatibilityStabilityV1Witness) error {
	byWitness := map[string]CompatibilityStabilityV1Witness{}
	for _, witness := range witnesses {
		if strings.TrimSpace(witness.ID) == "" || strings.TrimSpace(witness.Kind) == "" {
			return fmt.Errorf("compatibility/stability v1: witness missing id or kind")
		}
		if _, exists := byWitness[witness.ID]; exists {
			return fmt.Errorf("compatibility/stability v1: duplicate witness %q", witness.ID)
		}
		byWitness[witness.ID] = witness
	}
	expected := map[CompatibilityStabilityV1ID]bool{}
	for _, id := range p24CompatibilityStabilityV1IDs() {
		expected[id] = true
	}
	seen := map[CompatibilityStabilityV1ID]bool{}
	for _, row := range rows {
		if !expected[row.ID] {
			return fmt.Errorf("compatibility/stability v1: unexpected row %q", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("compatibility/stability v1: duplicate row %q", row.ID)
		}
		seen[row.ID] = true
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Status) == "" {
			return fmt.Errorf("compatibility/stability v1: row %q missing name or status", row.ID)
		}
		if len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			return fmt.Errorf("compatibility/stability v1: row %q missing evidence, tests, boundaries, or witness ids", row.ID)
		}
		for _, text := range append(append(append([]string{}, row.Evidence...), row.Tests...), row.Boundaries...) {
			if p24CompatibilityStabilityIsPlaceholder(text) {
				return fmt.Errorf("compatibility/stability v1: row %q has placeholder evidence", row.ID)
			}
		}
		for _, id := range row.WitnessIDs {
			if _, ok := byWitness[id]; !ok {
				return fmt.Errorf("compatibility/stability v1: row %q references missing witness %q", row.ID, id)
			}
		}
	}
	for _, id := range p24CompatibilityStabilityV1IDs() {
		if !seen[id] {
			return fmt.Errorf("compatibility/stability v1: missing row %q", id)
		}
	}
	if witness := byWitness[p24CompatibilityDiagnosticWitnessID]; !witness.StableDiagnosticCodesReviewed || witness.DiagnosticRegistryCount < 10 || !witness.DiagnosticCodesValid || !witness.DiagnosticJSONValidatorStrict || !witness.DiagnosticReleaseDocsPresent {
		return fmt.Errorf("compatibility/stability v1: stable diagnostic witness incomplete")
	}
	if witness := byWitness[p24CompatibilitySchemaWitnessID]; !witness.VersionedReportSchemasReviewed || witness.VersionedSchemaCount < 8 || !witness.ReportSchemasStrict {
		return fmt.Errorf("compatibility/stability v1: versioned report schema witness incomplete")
	}
	if witness := byWitness[p24CompatibilityManifestWitnessID]; !witness.ManifestCompatibilityChecksReviewed || witness.ManifestCompilerVersion == "" || witness.ManifestTargetCount == 0 || witness.ManifestFeatureCount == 0 || !witness.ManifestRuntimeABIPresent || !witness.ManifestValidatorStrict || !witness.ManifestFeatureRegistryLinked || !witness.ManifestRuntimeABIChecksPresent {
		return fmt.Errorf("compatibility/stability v1: manifest compatibility witness incomplete")
	}
	if witness := byWitness[p24CompatibilityMigrationWitnessID]; !witness.BreakingChangeMigrationGuidePresent || !witness.MigrationGuidePresent || !witness.APIBreakingReviewPresent || !witness.PatchLineRulePresent {
		return fmt.Errorf("compatibility/stability v1: breaking-change migration witness incomplete")
	}
	if witness := byWitness[p24CompatibilityDeprecationWitnessID]; !witness.DeprecationPolicyPresent || !witness.ReplacementPathRequired || !witness.RemovalDelayRequired || !witness.StdlibMajorLineRulePresent {
		return fmt.Errorf("compatibility/stability v1: deprecation policy witness incomplete")
	}
	if witness := byWitness[p24CompatibilityArtifactsWitnessID]; !witness.CompatibilityAuditArtifactPresent || !witness.CompatibilityDesignArtifactPresent || !witness.MigrationGuideArtifactPresent || !witness.DeprecationPolicyArtifactPresent {
		return fmt.Errorf("compatibility/stability v1: compatibility/stability artifact witness incomplete")
	}
	return nil
}

func p24CompatibilityStabilityValidateArtifacts(report CompatibilityStabilityV1Report) error {
	present := map[string]bool{}
	for _, artifact := range report.Artifacts {
		if strings.TrimSpace(artifact.Kind) == "" || strings.TrimSpace(artifact.Path) == "" {
			return fmt.Errorf("compatibility/stability v1: artifact missing kind or path")
		}
		present[artifact.Path] = artifact.Present
	}
	for _, path := range []string{
		"docs/audits/compatibility-stability-v1.md",
		"docs/plans/2026-06-03-p24.2-compatibility-stability-design.md",
		"docs/release/breaking-change-migration-guide.md",
		"docs/release/deprecation_policy.md",
	} {
		if !present[path] {
			return fmt.Errorf("compatibility/stability v1: required artifact %s missing", path)
		}
	}
	return nil
}

func p24CompatibilityStabilityV1IDs() []CompatibilityStabilityV1ID {
	return []CompatibilityStabilityV1ID{
		CompatibilityStableDiagnosticCodes,
		CompatibilityVersionedReportSchemas,
		CompatibilityManifestChecks,
		CompatibilityBreakingChangeMigrationGuide,
		CompatibilityDeprecationPolicy,
	}
}

func p24CompatibilityStabilityRow(id CompatibilityStabilityV1ID, name, status string, evidence, tests, boundaries, witnessIDs []string) CompatibilityStabilityV1Row {
	return CompatibilityStabilityV1Row{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   evidence,
		Tests:      tests,
		Boundaries: boundaries,
		WitnessIDs: witnessIDs,
	}
}

func p24CompatibilityStabilityArtifacts() []CompatibilityStabilityArtifact {
	return []CompatibilityStabilityArtifact{
		p24CompatibilityStabilityArtifact("compatibility_stability_audit", "docs/audits/compatibility-stability-v1.md"),
		p24CompatibilityStabilityArtifact("compatibility_stability_design", "docs/plans/2026-06-03-p24.2-compatibility-stability-design.md"),
		p24CompatibilityStabilityArtifact("breaking_change_migration_guide", "docs/release/breaking-change-migration-guide.md"),
		p24CompatibilityStabilityArtifact("deprecation_policy", "docs/release/deprecation_policy.md"),
	}
}

func p24CompatibilityStabilityArtifact(kind string, rel string) CompatibilityStabilityArtifact {
	_, err := os.Stat(p24RepoPath(rel))
	return CompatibilityStabilityArtifact{
		Kind:    kind,
		Path:    rel,
		Present: err == nil,
	}
}

func p24CompatibilityStabilityFileContains(rel string, want string) bool {
	data, err := os.ReadFile(p24RepoPath(rel))
	return err == nil && strings.Contains(string(data), want)
}

func p24CompatibilityStabilityLooksVersionedSchema(schema string) bool {
	last := strings.LastIndex(schema, ".v")
	if last < 0 || last+2 >= len(schema) {
		return false
	}
	for _, r := range schema[last+2:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func p24CompatibilityStabilityHasString(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func p24CompatibilityStabilityIsPlaceholder(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "" ||
		lower == "todo" ||
		lower == "tbd" ||
		strings.Contains(lower, "placeholder")
}
