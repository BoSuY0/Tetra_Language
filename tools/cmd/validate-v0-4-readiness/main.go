package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tetra_language/tools/validators/actordist"
	"tetra_language/tools/validators/nativeui"
)

type readinessInputs struct {
	ExpectedVersion string
	Manifest        []byte
	Features        []byte
	Targets         []byte
	ScopeDecisions  []byte
	RuntimeReports  map[string][]byte
}

type manifestReport struct {
	CompilerVersion string `json:"compiler_version"`
}

type featuresReport struct {
	Version  string `json:"version"`
	Features []struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Since  string `json:"since,omitempty"`
	} `json:"features"`
}

type targetsReport struct {
	Targets []struct {
		Triple               string `json:"triple"`
		Status               string `json:"status"`
		BuildOnly            bool   `json:"build_only"`
		RunSupported         bool   `json:"run_supported"`
		RunUnsupportedReason string `json:"run_unsupported_reason,omitempty"`
	} `json:"targets"`
}

type runtimeSmokeReport struct {
	Timestamp   string `json:"timestamp"`
	Target      string `json:"target"`
	BuildOnly   bool   `json:"build_only,omitempty"`
	Runner      string `json:"runner,omitempty"`
	Host        string `json:"host"`
	Unsupported bool   `json:"unsupported,omitempty"`
	Version     string `json:"version"`
	GitHead     string `json:"git_head,omitempty"`
	Total       int    `json:"total"`
	Passed      int    `json:"passed"`
	Failed      int    `json:"failed"`
	Cases       []struct {
		Name         string `json:"name"`
		ExpectedExit int    `json:"expected_exit"`
		ActualExit   *int   `json:"actual_exit,omitempty"`
		Ran          bool   `json:"ran"`
		Pass         bool   `json:"pass"`
		Unsupported  bool   `json:"unsupported,omitempty"`
		Error        string `json:"error,omitempty"`
	} `json:"cases"`
}

type scopeDecisionsReport struct {
	ReleaseVersion string `json:"release_version"`
	Status         string `json:"status"`
	Decisions      []struct {
		Kind     string           `json:"kind"`
		ID       string           `json:"id"`
		Decision string           `json:"decision"`
		Evidence decisionEvidence `json:"evidence,omitempty"`
	} `json:"decisions"`
}

type decisionEvidence struct {
	Implementation      []string `json:"implementation,omitempty"`
	Tests               []string `json:"tests,omitempty"`
	Docs                []string `json:"docs,omitempty"`
	ReleaseGateEvidence []string `json:"release_gate_evidence,omitempty"`
}

const (
	actorDistributedRuntimeDecision = "actors.distributed-runtime"
	actorTransportSchemaV1          = "tetra.actors.transport.v1"
	nativeUIRuntimeDecision         = "ui.native-runtime"
	nativeUISidecarSchemaV1         = "tetra.ui.native-shell.v1"
	uiBundleSchemaV1                = "tetra.ui.v0.4.0"
)

func main() {
	var runtimeReports runtimeReportFlags
	expectedVersion := flag.String("expected-version", "v0.4.0", "expected release version")
	manifestPath := flag.String("manifest", "docs/generated/manifest.json", "manifest JSON")
	featuresPath := flag.String(
		"features",
		"",
		"features JSON produced by ./tetra features --format=json",
	)
	targetsPath := flag.String(
		"targets",
		"",
		"targets JSON produced by ./tetra targets --format=json",
	)
	scopePath := flag.String(
		"scope-decisions",
		"docs/release/v0_4/data/v0_4_0_scope_decisions.json",
		"v0.4.0 scope decisions JSON",
	)
	flag.Var(
		&runtimeReports,
		"runtime-report",
		"external runtime smoke report as target=path; repeat for cross-host targets",
	)
	flag.Parse()

	inputs, err := readInputs(
		*expectedVersion,
		*manifestPath,
		*featuresPath,
		*targetsPath,
		*scopePath,
		runtimeReports,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate-v0-4-readiness: %v\n", err)
		os.Exit(2)
	}
	if err := validateReadiness(inputs); err != nil {
		fmt.Fprintf(os.Stderr, "validate-v0-4-readiness: %v\n", err)
		os.Exit(1)
	}
}

type runtimeReportFlags []string

func (r *runtimeReportFlags) String() string {
	return strings.Join(*r, ",")
}

func (r *runtimeReportFlags) Set(value string) error {
	target, path, ok := strings.Cut(value, "=")
	if !ok || strings.TrimSpace(target) == "" || strings.TrimSpace(path) == "" {
		return fmt.Errorf("runtime report must be target=path")
	}
	*r = append(*r, value)
	return nil
}

func readInputs(
	expectedVersion, manifestPath, featuresPath, targetsPath, scopePath string,
	runtimeReportSpecs []string,
) (readinessInputs, error) {
	read := func(path, label string) ([]byte, error) {
		if path == "" {
			return nil, fmt.Errorf("%s path is required", label)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", label, err)
		}
		return raw, nil
	}

	manifest, err := read(manifestPath, "manifest")
	if err != nil {
		return readinessInputs{}, err
	}
	features, err := read(featuresPath, "features")
	if err != nil {
		return readinessInputs{}, err
	}
	targets, err := read(targetsPath, "targets")
	if err != nil {
		return readinessInputs{}, err
	}
	scope, err := read(scopePath, "scope decisions")
	if err != nil {
		return readinessInputs{}, err
	}
	runtimeReports := map[string][]byte{}
	for _, spec := range runtimeReportSpecs {
		target, path, _ := strings.Cut(spec, "=")
		target = strings.TrimSpace(target)
		path = strings.TrimSpace(path)
		if target == "" || path == "" {
			return readinessInputs{}, fmt.Errorf("runtime report must be target=path")
		}
		if _, exists := runtimeReports[target]; exists {
			return readinessInputs{}, fmt.Errorf("duplicate runtime report for %s", target)
		}
		raw, err := read(path, "runtime report "+target)
		if err != nil {
			return readinessInputs{}, err
		}
		runtimeReports[target] = raw
	}
	return readinessInputs{
		ExpectedVersion: expectedVersion,
		Manifest:        manifest,
		Features:        features,
		Targets:         targets,
		ScopeDecisions:  scope,
		RuntimeReports:  runtimeReports,
	}, nil
}

func validateReadiness(inputs readinessInputs) error {
	expectedVersion := inputs.ExpectedVersion
	if expectedVersion == "" {
		expectedVersion = "v0.4.0"
	}

	var manifest manifestReport
	var features featuresReport
	var targets targetsReport
	var scope scopeDecisionsReport
	if err := decodeJSON(inputs.Manifest, &manifest, "manifest"); err != nil {
		return err
	}
	if err := decodeJSON(inputs.Features, &features, "features"); err != nil {
		return err
	}
	if err := decodeJSON(inputs.Targets, &targets, "targets"); err != nil {
		return err
	}
	if err := decodeJSON(inputs.ScopeDecisions, &scope, "scope decisions"); err != nil {
		return err
	}

	var issues []string
	if manifest.CompilerVersion != expectedVersion {
		issues = append(
			issues,
			fmt.Sprintf(
				"manifest compiler_version = %s, want %s",
				manifest.CompilerVersion,
				expectedVersion,
			),
		)
	}
	if features.Version != expectedVersion {
		issues = append(
			issues,
			fmt.Sprintf("features version = %s, want %s", features.Version, expectedVersion),
		)
	}
	if scope.ReleaseVersion != expectedVersion {
		issues = append(
			issues,
			fmt.Sprintf(
				"scope release_version = %s, want %s",
				scope.ReleaseVersion,
				expectedVersion,
			),
		)
	}
	if !isAllowedScopeStatus(scope.Status) {
		issues = append(
			issues,
			fmt.Sprintf(
				"scope status = %s, want full-production-scope-selected or linux-x64-production-scope-selected",
				scope.Status,
			),
		)
	}

	featureStatus := map[string]string{}
	featureSince := map[string]string{}
	for _, feature := range features.Features {
		featureStatus[feature.ID] = feature.Status
		featureSince[feature.ID] = feature.Since
	}
	targetByTriple := map[string]struct {
		Status               string
		BuildOnly            bool
		RunSupported         bool
		RunUnsupportedReason string
	}{}
	for _, target := range targets.Targets {
		targetByTriple[target.Triple] = struct {
			Status               string
			BuildOnly            bool
			RunSupported         bool
			RunUnsupportedReason string
		}{
			Status:               target.Status,
			BuildOnly:            target.BuildOnly,
			RunSupported:         target.RunSupported,
			RunUnsupportedReason: target.RunUnsupportedReason,
		}
	}

	for _, decision := range scope.Decisions {
		switch {
		case decision.Kind == "feature" && decision.Decision == "implement":
			issues = append(issues, validateDecisionEvidence(decision.ID, decision.Evidence)...)
			status, ok := featureStatus[decision.ID]
			if !ok {
				issues = append(
					issues,
					fmt.Sprintf("feature %s missing from features report", decision.ID),
				)
				continue
			}
			if status != "current" {
				issues = append(
					issues,
					fmt.Sprintf("feature %s status = %s, want current", decision.ID, status),
				)
			}
			if featureSince[decision.ID] == "" {
				issues = append(issues, fmt.Sprintf("feature %s missing since", decision.ID))
			}
		case decision.Kind == "target-runtime" && decision.Decision == "implement-production-runtime":
			issues = append(issues, validateDecisionEvidence(decision.ID, decision.Evidence)...)
			target, ok := targetByTriple[decision.ID]
			if !ok {
				issues = append(
					issues,
					fmt.Sprintf("target %s missing from targets report", decision.ID),
				)
				continue
			}
			if target.Status != "supported" {
				issues = append(
					issues,
					fmt.Sprintf(
						"target %s status = %s, want supported",
						decision.ID,
						target.Status,
					),
				)
			}
			if target.BuildOnly {
				issues = append(issues, fmt.Sprintf("target %s build_only = true", decision.ID))
			}
			if !target.RunSupported {
				if rawReport, ok := inputs.RuntimeReports[decision.ID]; ok {
					issues = append(
						issues,
						validateRuntimeSmokeReport(decision.ID, rawReport, expectedVersion)...)
				} else {
					issue := fmt.Sprintf("target %s run_supported = false", decision.ID)
					if target.RunUnsupportedReason != "" {
						issue += ": " + target.RunUnsupportedReason
					}
					issues = append(issues, issue)
				}
			} else if rawReport, ok := inputs.RuntimeReports[decision.ID]; ok {
				issues = append(issues, validateRuntimeSmokeReport(decision.ID, rawReport, expectedVersion)...)
			}
		}
	}

	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func isAllowedScopeStatus(status string) bool {
	switch status {
	case "full-production-scope-selected", "linux-x64-production-scope-selected":
		return true
	default:
		return false
	}
}

func validateRuntimeSmokeReport(target string, raw []byte, expectedVersion string) []string {
	var report runtimeSmokeReport
	if err := decodeJSON(raw, &report, target+" runtime smoke report"); err != nil {
		return []string{err.Error()}
	}
	label := target + " runtime report"
	var issues []string
	if _, err := time.Parse(time.RFC3339, report.Timestamp); err != nil {
		issues = append(issues, fmt.Sprintf("%s timestamp is not RFC3339: %v", label, err))
	}
	if report.Target != target {
		issues = append(
			issues,
			fmt.Sprintf("%s target is %q, want %q", label, report.Target, target),
		)
	}
	if report.Host == "" {
		issues = append(issues, fmt.Sprintf("%s host is empty", label))
	}
	if isNativeRuntimeTarget(target) {
		if report.Host != target {
			issues = append(
				issues,
				fmt.Sprintf("%s host is %q, want %q", label, report.Host, target),
			)
		}
		if report.Runner != "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s runner is %q, want empty host-native runtime",
					label,
					report.Runner,
				),
			)
		}
	}
	if report.BuildOnly {
		issues = append(issues, fmt.Sprintf("%s build_only is true, want false", label))
	}
	if report.Unsupported {
		issues = append(issues, fmt.Sprintf("%s unsupported is true, want false", label))
	}
	if report.Version != expectedVersion {
		issues = append(
			issues,
			fmt.Sprintf("%s version is %q, want %q", label, report.Version, expectedVersion),
		)
	}
	if strings.TrimSpace(report.GitHead) == "" {
		issues = append(issues, fmt.Sprintf("%s git_head is empty", label))
	}
	if len(report.Cases) == 0 {
		issues = append(issues, fmt.Sprintf("%s contains no cases", label))
	}

	passed := 0
	for _, c := range report.Cases {
		if c.Pass {
			passed++
		}
	}
	total := len(report.Cases)
	failed := total - passed
	if report.Total != total || report.Passed != passed || report.Failed != failed {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s counts mismatch: got total=%d passed=%d failed=%d, computed total=%d passed=%d failed=%d",
				label,
				report.Total,
				report.Passed,
				report.Failed,
				total,
				passed,
				failed,
			),
		)
	}

	byName := map[string]struct{}{}
	for _, c := range report.Cases {
		if strings.TrimSpace(c.Name) == "" {
			issues = append(issues, fmt.Sprintf("%s contains a case with empty name", label))
			continue
		}
		if _, ok := byName[c.Name]; ok {
			issues = append(issues, fmt.Sprintf("%s duplicate case %s", label, c.Name))
			continue
		}
		byName[c.Name] = struct{}{}
		if !c.Ran {
			issues = append(issues, fmt.Sprintf("%s case %s did not run", label, c.Name))
		}
		if !c.Pass {
			issues = append(issues, fmt.Sprintf("%s case %s did not pass", label, c.Name))
		}
		if c.Unsupported {
			issues = append(issues, fmt.Sprintf("%s case %s is marked unsupported", label, c.Name))
		}
		if c.ActualExit == nil {
			issues = append(issues, fmt.Sprintf("%s case %s missing actual_exit", label, c.Name))
		} else if *c.ActualExit != c.ExpectedExit {
			issues = append(
				issues,
				fmt.Sprintf("%s case %s actual_exit is %d, want %d", label, c.Name, *c.ActualExit, c.ExpectedExit),
			)
		}
		if strings.TrimSpace(c.Error) != "" {
			issues = append(issues, fmt.Sprintf("%s case %s has error text", label, c.Name))
		}
	}
	if isNativeRuntimeTarget(target) {
		for _, name := range requiredNativeRuntimeSmokeCases() {
			if _, ok := byName[name]; !ok {
				issues = append(
					issues,
					fmt.Sprintf("%s missing required runtime case %s", label, name),
				)
			}
		}
	}
	return issues
}

func isNativeRuntimeTarget(target string) bool {
	switch target {
	case "linux-x64", "macos-x64", "windows-x64":
		return true
	default:
		return false
	}
}

func requiredNativeRuntimeSmokeCases() []string {
	return []string{
		"actors_pingpong",
		"actor_sleep_pingpong",
		"task_smoke",
		"time_sleep_smoke",
		"task_sleep_deadline_smoke",
		"task_join_wait_smoke",
		"deadline_aware_waits_smoke",
		"wait_composition_smoke",
	}
}

func validateDecisionEvidence(id string, evidence decisionEvidence) []string {
	var issues []string
	required := []struct {
		name   string
		values []string
	}{
		{name: "implementation", values: evidence.Implementation},
		{name: "tests", values: evidence.Tests},
		{name: "docs", values: evidence.Docs},
		{name: "release_gate_evidence", values: evidence.ReleaseGateEvidence},
	}
	for _, item := range required {
		if len(nonEmptyStrings(item.values)) == 0 {
			issues = append(issues, fmt.Sprintf("decision %s missing evidence.%s", id, item.name))
		}
	}
	issues = append(
		issues,
		validateEvidencePaths(id, "implementation", evidence.Implementation, "")...)
	issues = append(issues, validateTestEvidence(id, evidence.Tests)...)
	issues = append(issues, validateEvidencePaths(id, "docs", evidence.Docs, "docs/")...)
	releaseIssues, hasReportArtifact := validateReleaseGateEvidence(
		id,
		evidence.ReleaseGateEvidence,
	)
	issues = append(issues, releaseIssues...)
	if len(nonEmptyStrings(evidence.ReleaseGateEvidence)) > 0 && !hasReportArtifact {
		issues = append(
			issues,
			fmt.Sprintf(
				"decision %s missing evidence.release_gate_evidence report artifact under reports/",
				id,
			),
		)
	}
	if id == actorDistributedRuntimeDecision {
		issues = append(issues, validateActorDistributedRuntimeEvidence(evidence)...)
	}
	if id == nativeUIRuntimeDecision {
		issues = append(issues, validateNativeUIRuntimeEvidence(evidence)...)
	}
	return issues
}

func validateNativeUIRuntimeEvidence(evidence decisionEvidence) []string {
	hasRuntimeImplementation := hasNativeUIRuntimeImplementationEvidence(evidence.Implementation)
	hasRuntimeTests := hasNativeUIRuntimeTestEvidence(evidence.Tests)
	hasRuntimeGateArtifact := hasNativeUIRuntimeGateArtifact(evidence.ReleaseGateEvidence)
	hasSidecarEvidence := hasNativeUISidecarOrMetadataEvidence(evidence)

	var issues []string
	if hasSidecarEvidence && !hasRuntimeImplementation && !hasRuntimeTests &&
		!hasRuntimeGateArtifact {
		issues = append(
			issues,
			fmt.Sprintf(
				("decision %s has only metadata/web/native-shell sidecar "+
					"evidence; requires real Linux-x64 native UI runtime evidence, not %s or "+
					"%s artifacts"),
				nativeUIRuntimeDecision,
				nativeUISidecarSchemaV1,
				uiBundleSchemaV1,
			),
		)
	}
	if !hasRuntimeImplementation {
		issues = append(
			issues,
			fmt.Sprintf(
				("decision %s requires real Linux-x64 native UI runtime "+
					"implementation evidence under tools/cmd/native-ui-runtime-smoke or "+
					"tools/validators/nativeui"),
				nativeUIRuntimeDecision,
			),
		)
	}
	if !hasRuntimeTests {
		issues = append(
			issues,
			fmt.Sprintf(
				("decision %s requires native UI runtime tests or smoke script "+
					"evidence covering widget load, click dispatch, state propagation, "+
					"negative dispatch paths, and close"),
				nativeUIRuntimeDecision,
			),
		)
	}
	if !hasNativeUIRuntimeDocsEvidence(evidence.Docs) {
		issues = append(
			issues,
			fmt.Sprintf(
				("decision %s docs evidence must include docs/spec/core/current_"+
					"supported_surface.md plus UI runtime docs/spec content"),
				nativeUIRuntimeDecision,
			),
		)
	}
	if !hasRuntimeGateArtifact {
		issues = append(
			issues,
			fmt.Sprintf(
				("decision %s requires a %s release-gate artifact under reports/, "+
					"not ui.metadata-v1, web, or native-shell sidecar evidence"),
				nativeUIRuntimeDecision,
				nativeui.SchemaV1,
			),
		)
	}
	issues = append(issues, validateNativeUIRuntimeGateArtifacts(evidence.ReleaseGateEvidence)...)
	return issues
}

func hasNativeUIRuntimeImplementationEvidence(values []string) bool {
	for _, value := range nonEmptyStrings(values) {
		normalized := normalizeEvidenceValue(value)
		if isNativeUISidecarOrMetadataEvidenceValue(normalized) {
			continue
		}
		if hasNativeUIRuntimeImplementationPrefix(normalized) {
			return true
		}
		if hasNativeUIRuntimeImplementationContent(normalized) {
			return true
		}
	}
	return false
}

func hasNativeUIRuntimeImplementationPrefix(value string) bool {
	for _, prefix := range []string{
		"tools/cmd/native-ui-runtime-smoke/",
		"tools/validators/nativeui/",
		"tools/cmd/validate-native-ui-runtime/",
	} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func hasNativeUIRuntimeImplementationContent(path string) bool {
	raw, err := readFileFromRepoRoot(path)
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(raw))
	for _, marker := range []string{
		"runruntimescenario",
		"loadnativeruntime",
		"native-ui-linux-x64",
		"tetra.ui.native-runtime.v1",
		"nativeui.validatereport",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func hasNativeUIRuntimeTestEvidence(values []string) bool {
	for _, value := range nonEmptyStrings(values) {
		normalized := normalizeEvidenceValue(value)
		if isNativeUISidecarOrMetadataEvidenceValue(normalized) {
			continue
		}
		for _, marker := range []string{
			"native-ui-runtime-smoke",
			"validate-native-ui-runtime",
			"tools/validators/nativeui",
			"scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh",
			"reports/v0.4.0/native-ui-linux-x64.json",
		} {
			if strings.Contains(normalized, marker) {
				return true
			}
		}
	}
	return false
}

func hasNativeUIRuntimeDocsEvidence(values []string) bool {
	hasSurface := false
	hasUIDocs := false
	for _, value := range nonEmptyStrings(values) {
		normalized := normalizeEvidenceValue(value)
		if normalized == "docs/spec/core/current_supported_surface.md" {
			hasSurface = true
		}
		if normalized == "docs/spec/ui/ui_v0.4.0.md" ||
			normalized == "docs/user/surface/wasm_ui_guide.md" ||
			strings.Contains(normalized, "ui") {
			hasUIDocs = true
		}
	}
	return hasSurface && hasUIDocs
}

func hasNativeUIRuntimeGateArtifact(values []string) bool {
	for _, value := range nonEmptyStrings(values) {
		normalized := normalizeEvidenceValue(value)
		if !strings.HasPrefix(normalized, "reports/") {
			continue
		}
		raw, err := readFileFromRepoRoot(normalized)
		if err != nil {
			continue
		}
		lower := strings.ToLower(string(raw))
		if strings.Contains(lower, nativeUISidecarSchemaV1) ||
			strings.Contains(lower, "tetra.ui.web") {
			continue
		}
		if strings.Contains(lower, nativeui.SchemaV1) && nativeui.ValidateReport(raw) == nil {
			return true
		}
	}
	return false
}

func validateNativeUIRuntimeGateArtifacts(values []string) []string {
	var issues []string
	for _, value := range nonEmptyStrings(values) {
		normalized := normalizeEvidenceValue(value)
		if !strings.HasPrefix(normalized, "reports/") {
			continue
		}
		raw, err := readFileFromRepoRoot(normalized)
		if err != nil {
			continue
		}
		lower := strings.ToLower(string(raw))
		if strings.Contains(lower, nativeUISidecarSchemaV1) {
			issues = append(
				issues,
				fmt.Sprintf(
					("decision %s evidence.release_gate_evidence path %s is native-"+
						"shell sidecar-only %s evidence, not native runtime evidence"),
					nativeUIRuntimeDecision,
					value,
					nativeUISidecarSchemaV1,
				),
			)
			continue
		}
		if strings.Contains(lower, "tetra.ui.web") || strings.Contains(lower, "wasm32-web") {
			issues = append(
				issues,
				fmt.Sprintf(
					("decision %s evidence.release_gate_evidence path %s is web "+
						"runtime evidence, not Linux-x64 native UI runtime evidence"),
					nativeUIRuntimeDecision,
					value,
				),
			)
			continue
		}
		if strings.Contains(lower, nativeui.SchemaV1) {
			if err := nativeui.ValidateReport(raw); err != nil {
				issues = append(
					issues,
					fmt.Sprintf(
						"decision %s evidence.release_gate_evidence path %s invalid native UI runtime report: %v",
						nativeUIRuntimeDecision,
						value,
						err,
					),
				)
			}
		} else if strings.Contains(normalized, "native-ui") || strings.Contains(normalized, "ui") {
			issues = append(
				issues,
				fmt.Sprintf(("decision %s evidence.release_gate_evidence path %s must use %s "+
					"executable runtime report schema"), nativeUIRuntimeDecision, value, nativeui.SchemaV1),
			)
		}
	}
	return issues
}

func hasNativeUISidecarOrMetadataEvidence(evidence decisionEvidence) bool {
	for _, values := range [][]string{
		evidence.Implementation,
		evidence.Tests,
		evidence.Docs,
		evidence.ReleaseGateEvidence,
	} {
		for _, value := range nonEmptyStrings(values) {
			normalized := normalizeEvidenceValue(value)
			if isNativeUISidecarOrMetadataEvidenceValue(normalized) {
				return true
			}
			if strings.HasPrefix(normalized, "reports/") {
				raw, err := readFileFromRepoRoot(normalized)
				if err == nil {
					lower := strings.ToLower(string(raw))
					if strings.Contains(lower, nativeUISidecarSchemaV1) ||
						strings.Contains(lower, `"schema":"tetra.ui.v0.4.0"`) ||
						strings.Contains(lower, `"schema": "tetra.ui.v0.4.0"`) ||
						strings.Contains(lower, "wasm32-web") {
						return true
					}
				}
			}
		}
	}
	return false
}

func isNativeUISidecarOrMetadataEvidenceValue(value string) bool {
	value = strings.ToLower(value)
	for _, marker := range []string{
		"validate-native-ui-smoke",
		"native-shell",
		"native_shell",
		"ui.shell",
		nativeUISidecarSchemaV1,
		"wasm32-web",
		"wasm_ui",
		"ui.metadata",
		"metadata-v1",
	} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func validateActorDistributedRuntimeEvidence(evidence decisionEvidence) []string {
	hasRuntimeImplementation := hasActorDistributedRuntimeImplementationEvidence(
		evidence.Implementation,
	)
	hasRuntimeTests := hasActorDistributedRuntimeTestEvidence(evidence.Tests)
	hasRuntimeGateArtifact := hasActorDistributedRuntimeGateArtifact(evidence.ReleaseGateEvidence)
	hasTransportEvidence := hasActorTransportEvidence(evidence)

	var issues []string
	if hasTransportEvidence && !hasRuntimeImplementation && !hasRuntimeTests &&
		!hasRuntimeGateArtifact {
		issues = append(
			issues,
			fmt.Sprintf(
				("decision %s has only actor transport evidence; requires real "+
					"distributed actor runtime/lowering evidence, not %s envelope/trace/hash "+
					"validation"),
				actorDistributedRuntimeDecision,
				actorTransportSchemaV1,
			),
		)
	}
	if !hasRuntimeImplementation {
		issues = append(
			issues,
			fmt.Sprintf(
				("decision %s requires real distributed actor runtime/lowering "+
					"evidence under compiler runtime, lowering, IR, or backend paths"),
				actorDistributedRuntimeDecision,
			),
		)
	}
	if !hasRuntimeTests {
		issues = append(
			issues,
			fmt.Sprintf(
				("decision %s requires distributed actor runtime tests covering "+
					"cross-node send/receive plus failure, cancel, join, and diagnostics "+
					"behavior"),
				actorDistributedRuntimeDecision,
			),
		)
	}
	if !hasActorDistributedRuntimeDocsEvidence(evidence.Docs) {
		issues = append(
			issues,
			fmt.Sprintf(
				("decision %s docs evidence must include docs/spec/core/current_"+
					"supported_surface.md and actor docs"),
				actorDistributedRuntimeDecision,
			),
		)
	}
	if !hasRuntimeGateArtifact {
		issues = append(
			issues,
			fmt.Sprintf(
				("decision %s requires a distributed actor runtime release-gate "+
					"artifact under reports/, not actor transport-only evidence"),
				actorDistributedRuntimeDecision,
			),
		)
	}
	issues = append(
		issues,
		validateActorDistributedRuntimeGateArtifacts(evidence.ReleaseGateEvidence)...)
	return issues
}

func hasActorDistributedRuntimeImplementationEvidence(values []string) bool {
	for _, value := range nonEmptyStrings(values) {
		normalized := normalizeEvidenceValue(value)
		if isActorTransportEvidenceValue(normalized) {
			continue
		}
		if !hasRuntimeImplementationPrefix(normalized) {
			continue
		}
		if hasDistributedRuntimeMarker(normalized) ||
			hasDistributedRuntimeImplementationContent(normalized) {
			return true
		}
	}
	return false
}

func hasDistributedRuntimeImplementationContent(path string) bool {
	raw, err := readFileFromRepoRoot(path)
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(raw))
	for _, marker := range []string{
		"emitactornodeconnect",
		"emitactorspawnremote",
		"emitactornetpump",
		"__tetra_actor_node_connect",
		"__tetra_actor_spawn_remote",
		"__tetra_actor_node_status",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func hasRuntimeImplementationPrefix(value string) bool {
	for _, prefix := range []string{
		"compiler/internal/actorsrt/",
		"compiler/internal/backend/",
		"compiler/internal/ir/",
		"compiler/internal/lower/",
		"compiler/runtime/",
		"runtime/",
	} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func hasActorDistributedRuntimeTestEvidence(values []string) bool {
	for _, value := range nonEmptyStrings(values) {
		normalized := normalizeEvidenceValue(value)
		if isActorTransportEvidenceValue(normalized) {
			continue
		}
		if !isRuntimeTestEvidenceValue(normalized) {
			continue
		}
		if hasDistributedRuntimeMarker(normalized) {
			return true
		}
	}
	return false
}

func isRuntimeTestEvidenceValue(value string) bool {
	for _, marker := range []string{
		"go test ./compiler",
		"go test ./cli",
		"compiler/",
		"cli/",
		"reports/",
	} {
		if strings.HasPrefix(value, marker) || strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func hasActorDistributedRuntimeDocsEvidence(values []string) bool {
	hasSurface := false
	hasActorDocs := false
	for _, value := range nonEmptyStrings(values) {
		normalized := normalizeEvidenceValue(value)
		if normalized == "docs/spec/core/current_supported_surface.md" {
			hasSurface = true
		}
		if normalized == "docs/spec/runtime/actors.md" || strings.Contains(normalized, "actor") {
			hasActorDocs = true
		}
	}
	return hasSurface && hasActorDocs
}

func hasActorDistributedRuntimeGateArtifact(values []string) bool {
	for _, value := range nonEmptyStrings(values) {
		normalized := normalizeEvidenceValue(value)
		if !strings.HasPrefix(normalized, "reports/") {
			continue
		}
		if isActorTransportEvidenceValue(normalized) {
			continue
		}
		raw, err := readFileFromRepoRoot(normalized)
		if err != nil {
			continue
		}
		lower := strings.ToLower(string(raw))
		if strings.Contains(lower, actorTransportSchemaV1) {
			continue
		}
		if strings.Contains(lower, actordist.SchemaV1) && actordist.ValidateReport(raw) == nil {
			return true
		}
	}
	return false
}

func validateActorDistributedRuntimeGateArtifacts(values []string) []string {
	var issues []string
	for _, value := range nonEmptyStrings(values) {
		normalized := normalizeEvidenceValue(value)
		if !strings.HasPrefix(normalized, "reports/") {
			continue
		}
		raw, err := readFileFromRepoRoot(normalized)
		if err != nil {
			continue
		}
		if isActorTransportEvidenceValue(normalized) ||
			strings.Contains(strings.ToLower(string(raw)), actorTransportSchemaV1) {
			issues = append(
				issues,
				fmt.Sprintf(
					("decision %s evidence.release_gate_evidence path %s is transport-"+
						"only %s evidence, not distributed actor runtime evidence"),
					actorDistributedRuntimeDecision,
					value,
					actorTransportSchemaV1,
				),
			)
			continue
		}
		if strings.Contains(strings.ToLower(string(raw)), actordist.SchemaV1) {
			if err := actordist.ValidateReport(raw); err != nil {
				issues = append(
					issues,
					fmt.Sprintf(
						("decision %s evidence.release_gate_evidence path %s invalid "+
							"distributed actor runtime report: %v"),
						actorDistributedRuntimeDecision,
						value,
						err,
					),
				)
			}
		} else if hasDistributedRuntimeMarker(normalized) {
			issues = append(
				issues,
				fmt.Sprintf(("decision %s evidence.release_gate_evidence path %s must use %s "+
					"executable runtime report schema"), actorDistributedRuntimeDecision, value, actordist.SchemaV1),
			)
		}
	}
	return issues
}

func hasActorTransportEvidence(evidence decisionEvidence) bool {
	for _, values := range [][]string{
		evidence.Implementation,
		evidence.Tests,
		evidence.Docs,
		evidence.ReleaseGateEvidence,
	} {
		for _, value := range nonEmptyStrings(values) {
			normalized := normalizeEvidenceValue(value)
			if isActorTransportEvidenceValue(normalized) {
				return true
			}
			if strings.HasPrefix(normalized, "reports/") {
				raw, err := readFileFromRepoRoot(normalized)
				if err == nil &&
					strings.Contains(strings.ToLower(string(raw)), actorTransportSchemaV1) {
					return true
				}
			}
		}
	}
	return false
}

func isActorTransportEvidenceValue(value string) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, "validate-actor-transport") ||
		strings.Contains(value, "actor-transport") ||
		strings.Contains(value, actorTransportSchemaV1)
}

func hasDistributedRuntimeMarker(value string) bool {
	value = strings.ToLower(value)
	for _, marker := range []string{
		"distributed",
		"cross-node",
		"cross node",
		"remote actor",
		"remote-actor",
		"remote mailbox",
		"remote-mailbox",
		"network mailbox",
		"network-mailbox",
		"networked mailbox",
		"networked-mailbox",
		"cluster",
	} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func normalizeEvidenceValue(value string) string {
	return filepath.ToSlash(strings.ToLower(strings.TrimSpace(value)))
}

func validateTestEvidence(id string, values []string) []string {
	var issues []string
	for _, value := range nonEmptyStrings(values) {
		if isEvidenceCommand(value) {
			continue
		}
		issues = append(issues, validateEvidencePaths(id, "tests", []string{value}, "")...)
	}
	return issues
}

func isEvidenceCommand(value string) bool {
	value = strings.TrimSpace(value)
	for _, prefix := range []string{
		"go test ",
		"go run ",
		"bash ",
		"./tetra ",
		"./scripts/",
		"scripts/",
		"npm ",
		"node ",
		"python ",
		"python3 ",
	} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func validateReleaseGateEvidence(id string, paths []string) ([]string, bool) {
	var issues []string
	hasReportArtifact := false
	for _, rawPath := range nonEmptyStrings(paths) {
		path := filepath.ToSlash(strings.TrimSpace(rawPath))
		if filepath.IsAbs(rawPath) || path == ".." || strings.HasPrefix(path, "../") ||
			strings.Contains(path, "/../") {
			issues = append(
				issues,
				fmt.Sprintf(
					"decision %s evidence.release_gate_evidence path %s is unsafe",
					id,
					rawPath,
				),
			)
			continue
		}
		info, err := statFromRepoRoot(path)
		if err != nil {
			issues = append(
				issues,
				fmt.Sprintf(
					"decision %s evidence.release_gate_evidence path %s is not readable",
					id,
					rawPath,
				),
			)
			if strings.HasPrefix(path, "reports/") {
				hasReportArtifact = true
			}
			continue
		}
		if info.IsDir() {
			issues = append(
				issues,
				fmt.Sprintf(
					"decision %s evidence.release_gate_evidence path %s is a directory, want report file",
					id,
					rawPath,
				),
			)
			continue
		}
		if !strings.HasPrefix(path, "reports/") {
			continue
		}
		hasReportArtifact = true
		raw, err := readFileFromRepoRoot(path)
		if err != nil {
			issues = append(
				issues,
				fmt.Sprintf(
					"decision %s evidence.release_gate_evidence path %s is not readable",
					id,
					rawPath,
				),
			)
			continue
		}
		issues = append(issues, validateReleaseGateReportContent(id, rawPath, raw)...)
	}
	return issues, hasReportArtifact
}

func validateReleaseGateReportContent(id string, path string, raw []byte) []string {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return []string{
			fmt.Sprintf(
				"decision %s evidence.release_gate_evidence path %s is incomplete",
				id,
				path,
			),
		}
	}
	if json.Valid(raw) {
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err == nil {
			if isIncompleteJSONEvidence(decoded) {
				return []string{
					fmt.Sprintf(
						"decision %s evidence.release_gate_evidence path %s is incomplete",
						id,
						path,
					),
				}
			}
			if containsForbiddenStructuredEvidence(decoded) {
				return []string{
					fmt.Sprintf(
						"decision %s evidence.release_gate_evidence path %s contains forbidden filler wording",
						id,
						path,
					),
				}
			}
			return nil
		}
	}
	lower := strings.ToLower(text)
	for _, phrase := range []string{"fake", "mock", "placeholder"} {
		if strings.Contains(lower, phrase) {
			return []string{
				fmt.Sprintf(
					"decision %s evidence.release_gate_evidence path %s contains forbidden filler wording",
					id,
					path,
				),
			}
		}
	}
	return nil
}

func isIncompleteJSONEvidence(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		return len(typed) == 0
	case []any:
		return len(typed) == 0
	case string:
		return strings.TrimSpace(typed) == ""
	case nil:
		return true
	default:
		return true
	}
}

func containsForbiddenStructuredEvidence(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if forbiddenStructuredEvidenceValue(key, child) || containsForbiddenStructuredEvidence(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsForbiddenStructuredEvidence(child) {
				return true
			}
		}
	case string:
		return isForbiddenEvidenceWord(typed)
	}
	return false
}

func forbiddenStructuredEvidenceValue(key string, value any) bool {
	text, ok := value.(string)
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "status", "result", "outcome":
		return isForbiddenEvidenceWord(text)
	default:
		return false
	}
}

func isForbiddenEvidenceWord(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "fake", "mock", "placeholder":
		return true
	default:
		return false
	}
}

func validateEvidencePaths(id, bucket string, paths []string, requiredPrefix string) []string {
	var issues []string
	for _, rawPath := range nonEmptyStrings(paths) {
		path := filepath.ToSlash(strings.TrimSpace(rawPath))
		if filepath.IsAbs(rawPath) || path == ".." || strings.HasPrefix(path, "../") ||
			strings.Contains(path, "/../") {
			issues = append(
				issues,
				fmt.Sprintf("decision %s evidence.%s path %s is unsafe", id, bucket, rawPath),
			)
			continue
		}
		if requiredPrefix != "" && !strings.HasPrefix(path, requiredPrefix) {
			issues = append(
				issues,
				fmt.Sprintf(
					"decision %s evidence.%s path %s must be under %s",
					id,
					bucket,
					rawPath,
					strings.TrimSuffix(requiredPrefix, "/")+"/",
				),
			)
			continue
		}
		if _, err := statFromRepoRoot(path); err != nil {
			issues = append(
				issues,
				fmt.Sprintf("decision %s evidence.%s path %s is not readable", id, bucket, rawPath),
			)
		}
	}
	return issues
}

func statFromRepoRoot(path string) (os.FileInfo, error) {
	if info, err := os.Stat(filepath.FromSlash(path)); err == nil {
		return info, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, filepath.FromSlash(path))
		if info, err := os.Stat(candidate); err == nil {
			return info, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return nil, os.ErrNotExist
}

func readFileFromRepoRoot(path string) ([]byte, error) {
	if raw, err := os.ReadFile(filepath.FromSlash(path)); err == nil {
		return raw, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, filepath.FromSlash(path))
		if raw, err := os.ReadFile(candidate); err == nil {
			return raw, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return nil, os.ErrNotExist
}

func nonEmptyStrings(values []string) []string {
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func decodeJSON(raw []byte, out any, label string) error {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("%s JSON: %w", label, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("%s JSON: trailing data", label)
	}
	return nil
}
