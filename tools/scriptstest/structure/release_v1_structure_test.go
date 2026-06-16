package structure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readScriptstestFile(t *testing.T, rel string) ([]byte, error) {
	t.Helper()
	root := repoRoot(t)
	base := filepath.Join(root, "tools", "scriptstest")
	for _, dir := range []string{"", "structure", "surface", "postv04", "release_v040", "release_legacy", "release_v030"} {
		raw, err := os.ReadFile(filepath.Join(base, dir, rel))
		if err == nil || !os.IsNotExist(err) {
			return raw, err
		}
	}
	return nil, os.ErrNotExist
}

func TestReleaseV1TestsAreSplitByFixtureDomain(t *testing.T) {
	expected := map[string][]string{
		"release_v030_fixtures_test.go": {
			"releaseV030FakeRepo",
			"releaseV030RunnableFakeRepo",
			"runReleaseV030Gate",
			"envHasPrefix",
			"filteredReleaseV030GateEnv",
			"runReleaseV030RunnableGate",
			"runReleaseV030RunnableGateWithEnv",
			"writeReleaseV030RuntimeSmokeReports",
			"installReleaseV030SummaryEchoingGo",
			"installReleaseV030CanonicalArtifactGo",
			"installReleaseV030CIMissingSignoffFailingFinalArtifactHashGo",
			"installReleaseV030FailingFinalArtifactHashGo",
			"installReleaseV030FailingSecurityReviewSha256",
			"installReleaseV030PortablePythonCanonicalizers",
		},
		"release_v10_fixtures_test.go": {
			"releaseV10GateFakeRepo",
			"releaseV10WASISmokeFakeRepo",
			"releaseV10WebSmokeFakeRepo",
			"writeToolWrapper",
			"writeReleaseV10FakeBrowser",
			"runReleaseV10WebSmoke",
			"readWebSmokeReport",
		},
		"release_helpers_test.go": {
			"shellSingleQuote",
			"copyFile",
			"repoRoot",
		},
		"test_all_helpers_test.go": {
			"hasTestAllStep",
			"testAllStepLog",
			"readTestAllScript",
			"readReleaseV06GateScript",
			"runTestAll",
			"runTestAllSplit",
			"runTestAllFromWorkingDir",
			"decodeTestAllSummary",
		},
		"release_current_surface_test.go": {
			"TestCurrentSupportedSurfaceDocumentIsReleaseAligned",
		},
		"release_bootstrap_test.go": {
			"TestBootstrapBuildsTetraAndTAlias",
		},
		"release_v10_web_smoke_test.go": {
			"TestReleaseV10WebSmokeScriptValidatesReportBeforeExit",
			"TestReleaseV10WebSmokeScriptCapturesUISchemaEvidence",
			"TestReleaseV10WebSmokeScriptCapturesRuntimeSignals",
			"TestReleaseV10WebSmokeScript_bindHTTPServerToLoopback",
			"Test_release_v1_0_web_smokeDiscoversGoogleChromeFallback",
			"Test_release_v1_0_web_smokeBrowserArgOverridesDiscovery",
			"Test_release_v1_0_web_smokeMissingExplicitBrowserWritesBlockedReport",
			"Test_release_v1_0_web_smokeValidateWASMImportsFailureWritesStructuredFailReport",
		},
		"release_v10_wasi_smoke_test.go": {
			"TestReleaseV10WASISmokeScriptRejectsUISidecarsForDogfood",
			"TestReleaseV10WASISmokeUsesUnifiedCLIRuntimePath",
			"TestReleaseV10WASISmokeRunsUnifiedCLIAndValidatesReport",
		},
		"release_v10_gate_test.go": {
			"TestReleaseV10GateUsesRealV1Boundary",
			"TestReleaseV10GateRunsDedicatedV1Workflow",
		},
		"release_v10_policy_test.go": {
			"TestReleaseV10SmokeScriptsHaveDefaultReportPaths",
			"TestRoadmapV10RecordsExplicitCompatibilityAndSafetyPolicy",
		},
		"release_v011_gate_test.go": {
			"TestReleaseV011GateDocumentsMandatoryTargets",
			"TestReleaseV011GateKeepsCurrentValidators",
			"TestReleaseV011GateRecordsBinarySizeEvidenceBeforeRepro",
			"TestReleaseV011GateValidatesJSONDiagnostics",
			"TestReleaseV011GateRequiresSecurityReviewSignoff",
			"TestReleaseV011GateArchivesReleaseStateKnownIssuesAndHashes",
			"TestReleaseV011GateChecksGeneratedArtifactChurn",
			"TestReleaseV011GateRunsVersionPreflightBeforePackageTests",
		},
		"release_v012_gate_test.go": {
			"TestReleaseV012GateArchivesReleaseStateWithExpectedVersion",
		},
		"release_v013_gate_test.go": {
			"TestReleaseV013GateIsCanonicalPatchGate",
			"TestReleaseV013GateValidatesPerformanceEvidenceArtifact",
		},
		"release_v020_gate_test.go": {
			"TestReleaseV020GateDelegatesWithV020Boundary",
		},
		"release_v030_gate_static_test.go": {
			"TestReleaseV030GateUsesDedicatedV030Boundary",
			"TestReleaseV030ChecklistIsNonClaimingAndVersionScoped",
			"TestReleaseV030ChecklistAndGateRequireSecuritySignoff",
			"TestReleaseV030GateGoTestStepUnsetsReleaseInputEnv",
			"TestReleaseV030SecurityReviewWrapperUsesV030Name",
		},
		"release_v030_gate_evidence_test.go": {
			"TestReleaseV030GateRequireCleanRejectsDirtyWorktree",
			"TestReleaseV030GateValidatesFuzzArtifactsAfterShortFuzz",
			"TestReleaseV030GateRefreshesReleaseStateAfterFinalSummaryWrite",
			"TestReleaseV030GateWritesBlockedReleaseStateBeforeCIMissingSignoffExit",
			"TestReleaseV030GateValidatesGateSummaryArtifacts",
			"TestReleaseV030GateHashesEntireReportDirectory",
			"finalReleaseStateRefreshFollowsSummary",
		},
		"release_v030_gate_report_dir_test.go": {
			"TestReleaseV030GateRejectsExistingReportArtifacts",
			"TestReleaseV030GateRejectsSymlinkToExistingReportArtifacts",
			"TestReleaseV030GateRejectsDashPrefixedExistingReportArtifacts",
		},
		"release_v030_gate_residual_risks_test.go": {
			"TestReleaseV030GateRejectsUntriagedUnstableSeeds",
			"TestReleaseV030GateAcceptsTriagedUnstableSeeds",
			"TestReleaseV030GateWritesResidualRisksJSONArtifact",
			"TestReleaseV030GateAcceptsResidualRisksSourcePathStartingWithDash",
			"TestReleaseV030GateRejectsUnownedHighMediumResidualRisk",
			"TestReleaseV030GateRejectsResidualRisksJSONForWrongReleaseVersion",
			"TestReleaseV030GateRejectsMalformedResidualRisksJSON",
			"TestReleaseV030GateRejectsNullResidualRisksArray",
			"TestReleaseV030GateRejectsResidualRiskMissingRequiredFields",
			"TestReleaseV030RunnableGateFiltersAmbientResidualRisksEnv",
		},
		"release_v030_gate_security_signoff_test.go": {
			"TestReleaseV030GateCIModeAllowsMissingSecuritySignoffWithArtifact",
			"TestReleaseV030GateCIMissingSignoffWritesDetachedHashOutsideCanonicalManifest",
			"TestReleaseV030GateRequiresSecuritySignoffOutsideCIMode",
			"TestReleaseV030GateRequireCleanRequiresSecuritySignoffEvenInCIMode",
		},
		"release_v030_gate_security_signoff_acceptance_test.go": {
			"TestReleaseV030GateAcceptsSameRunSecuritySignoffWithFreshReportArtifacts",
			"TestReleaseV030GateAcceptsSecuritySignoffPathStartingWithDash",
			"TestReleaseV030GateWritesDetachedSecurityReviewHashOutsideCanonicalManifest",
		},
		"release_v030_gate_final_summary_test.go": {
			"TestReleaseV030GateBlocksFinalSummaryWhenPostSummaryArtifactHashCheckFails",
			"TestReleaseV030GateBlocksFinalSummaryWhenDetachedSecurityHashFails",
			"TestReleaseV030GateRecordsCIMissingSignoffFinalArtifactHashRefreshFailure",
			"TestReleaseV030GateCanonicalizesArtifactManifestWithPython3WhenPythonIsUnavailable",
			"TestReleaseV030GateRefreshesReleaseStateAfterFinalSummary",
			"countReleaseGateFailedSteps",
		},
		"release_v030_gate_runtime_smoke_test.go": {
			"TestReleaseV030GateRejectsBuildOnlyRuntimeSmokeEvidence",
			"TestReleaseV030GateRejectsWrongVersionRuntimeSmokeEvidence",
			"TestReleaseV030GateDoesNotArchivePartialRuntimeEvidenceWhenWindowsReportInvalid",
			"TestReleaseV030GateDoesNotArchivePartialRuntimeEvidenceWhenRuntimeCopyFails",
			"TestReleaseV030GateAcceptsRuntimeSmokeSourcePathStartingWithDash",
		},
		"release_v030_gate_runtime_smoke_schema_test.go": {
			"TestReleaseV030GateRejectsWrongGitHeadRuntimeSmokeEvidence",
			"TestReleaseV030GateRejectsRunnerRuntimeSmokeEvidence",
			"TestReleaseV030GateRejectsInvalidTimestampRuntimeSmokeEvidence",
			"TestReleaseV030GateRejectsLooseTimestampRuntimeSmokeEvidence",
			"TestReleaseV030GateRejectsMissingRequiredRuntimeSmokeCase",
			"TestReleaseV030GateRejectsRuntimeSmokeCaseErrorText",
			"TestReleaseV030GateRejectsEmptyRuntimeSmokeCaseName",
			"TestReleaseV030GateRejectsNonStringRuntimeSmokeCaseName",
			"TestReleaseV030GateRejectsNonBooleanRuntimeSmokeCaseStatus",
			"TestReleaseV030GateRejectsNonIntegerRuntimeSmokeCaseExitFields",
			"TestReleaseV030GateRejectsNonIntegerRuntimeSmokeCounts",
			"TestReleaseV030GateRejectsNonBooleanRuntimeSmokeIslandsDebug",
			"TestReleaseV030GateRejectsNonBooleanRuntimeSmokeBuildOnly",
			"TestReleaseV030GateRejectsNonStringRuntimeSmokeRunner",
		},
		"release_v040_gate_test.go": {
			"TestReleaseV040GateUsesDedicatedReadinessPreflight",
			"TestReleaseV040GateWritesBlockedSummaryOnReadinessFailure",
			"TestReleaseV040GateRejectsNonEmptyReportDirBeforeSideEffects",
		},
	}

	for path, symbols := range expected {
		raw, err := readScriptstestFile(t, path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(raw)
		for _, symbol := range symbols {
			if !strings.Contains(text, "func "+symbol+"(") {
				t.Fatalf("%s must contain %s", path, symbol)
			}
		}
	}

	helperRaw, err := readScriptstestFile(t, "test_all_helpers_test.go")
	if err != nil {
		t.Fatalf("read test_all_helpers_test.go: %v", err)
	}
	helperText := string(helperRaw)
	for _, want := range []string{
		"type testAllSummary struct",
		"const testAllFormatterStepName",
	} {
		if !strings.Contains(helperText, want) {
			t.Fatalf("test_all_helpers_test.go must contain %s", want)
		}
	}

	fixtureRaw, err := readScriptstestFile(t, "test_all_fixtures_test.go")
	if err != nil {
		t.Fatalf("read test_all_fixtures_test.go: %v", err)
	}
	if !strings.Contains(string(fixtureRaw), "func testAllFakeRepo(") {
		t.Fatalf("test_all_fixtures_test.go must contain testAllFakeRepo")
	}

	raw, err := readScriptstestFile(t, "release_v1_test.go")
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read release_v1_test.go: %v", err)
	}
	text := string(raw)
	for _, symbol := range []string{
		"releaseV030FakeRepo",
		"releaseV030RunnableFakeRepo",
		"runReleaseV030Gate",
		"envHasPrefix",
		"filteredReleaseV030GateEnv",
		"runReleaseV030RunnableGate",
		"runReleaseV030RunnableGateWithEnv",
		"writeReleaseV030RuntimeSmokeReports",
		"installReleaseV030SummaryEchoingGo",
		"installReleaseV030CanonicalArtifactGo",
		"installReleaseV030CIMissingSignoffFailingFinalArtifactHashGo",
		"installReleaseV030FailingFinalArtifactHashGo",
		"installReleaseV030FailingSecurityReviewSha256",
		"installReleaseV030PortablePythonCanonicalizers",
		"releaseV10GateFakeRepo",
		"releaseV10WASISmokeFakeRepo",
		"releaseV10WebSmokeFakeRepo",
		"writeToolWrapper",
		"writeReleaseV10FakeBrowser",
		"runReleaseV10WebSmoke",
		"readWebSmokeReport",
		"shellSingleQuote",
		"copyFile",
		"repoRoot",
		"hasTestAllStep",
		"testAllStepLog",
		"readTestAllScript",
		"readReleaseV06GateScript",
		"runTestAll",
		"runTestAllSplit",
		"runTestAllFromWorkingDir",
		"decodeTestAllSummary",
		"TestReleaseV10WebSmokeScriptValidatesReportBeforeExit",
		"TestReleaseV10WebSmokeScriptCapturesUISchemaEvidence",
		"TestReleaseV10WebSmokeScriptCapturesRuntimeSignals",
		"TestReleaseV10WebSmokeScript_bindHTTPServerToLoopback",
		"Test_release_v1_0_web_smokeDiscoversGoogleChromeFallback",
		"Test_release_v1_0_web_smokeBrowserArgOverridesDiscovery",
		"Test_release_v1_0_web_smokeMissingExplicitBrowserWritesBlockedReport",
		"Test_release_v1_0_web_smokeValidateWASMImportsFailureWritesStructuredFailReport",
		"TestReleaseV10WASISmokeScriptRejectsUISidecarsForDogfood",
		"TestReleaseV10WASISmokeUsesUnifiedCLIRuntimePath",
		"TestReleaseV10WASISmokeRunsUnifiedCLIAndValidatesReport",
		"TestReleaseV10GateUsesRealV1Boundary",
		"TestReleaseV10GateRunsDedicatedV1Workflow",
		"TestReleaseV10SmokeScriptsHaveDefaultReportPaths",
		"TestRoadmapV10RecordsExplicitCompatibilityAndSafetyPolicy",
		"TestReleaseV011GateDocumentsMandatoryTargets",
		"TestReleaseV011GateKeepsCurrentValidators",
		"TestReleaseV011GateRecordsBinarySizeEvidenceBeforeRepro",
		"TestReleaseV011GateValidatesJSONDiagnostics",
		"TestReleaseV011GateRequiresSecurityReviewSignoff",
		"TestReleaseV011GateArchivesReleaseStateKnownIssuesAndHashes",
		"TestReleaseV011GateChecksGeneratedArtifactChurn",
		"TestReleaseV011GateRunsVersionPreflightBeforePackageTests",
		"TestReleaseV012GateArchivesReleaseStateWithExpectedVersion",
		"TestReleaseV013GateIsCanonicalPatchGate",
		"TestReleaseV013GateValidatesPerformanceEvidenceArtifact",
		"TestReleaseV020GateDelegatesWithV020Boundary",
		"TestReleaseV030GateUsesDedicatedV030Boundary",
		"TestReleaseV030ChecklistIsNonClaimingAndVersionScoped",
		"TestReleaseV030ChecklistAndGateRequireSecuritySignoff",
		"TestReleaseV030GateGoTestStepUnsetsReleaseInputEnv",
		"TestReleaseV030SecurityReviewWrapperUsesV030Name",
		"TestReleaseV030GateRequireCleanRejectsDirtyWorktree",
		"TestReleaseV030GateValidatesFuzzArtifactsAfterShortFuzz",
		"TestReleaseV030GateRefreshesReleaseStateAfterFinalSummaryWrite",
		"TestReleaseV030GateWritesBlockedReleaseStateBeforeCIMissingSignoffExit",
		"TestReleaseV030GateValidatesGateSummaryArtifacts",
		"TestReleaseV030GateHashesEntireReportDirectory",
		"finalReleaseStateRefreshFollowsSummary",
		"TestReleaseV030GateRejectsExistingReportArtifacts",
		"TestReleaseV030GateRejectsSymlinkToExistingReportArtifacts",
		"TestReleaseV030GateRejectsDashPrefixedExistingReportArtifacts",
		"TestReleaseV030GateRejectsUntriagedUnstableSeeds",
		"TestReleaseV030GateAcceptsTriagedUnstableSeeds",
		"TestReleaseV030GateWritesResidualRisksJSONArtifact",
		"TestReleaseV030GateAcceptsResidualRisksSourcePathStartingWithDash",
		"TestReleaseV030GateRejectsUnownedHighMediumResidualRisk",
		"TestReleaseV030GateRejectsResidualRisksJSONForWrongReleaseVersion",
		"TestReleaseV030GateRejectsMalformedResidualRisksJSON",
		"TestReleaseV030GateRejectsNullResidualRisksArray",
		"TestReleaseV030GateRejectsResidualRiskMissingRequiredFields",
		"TestReleaseV030RunnableGateFiltersAmbientResidualRisksEnv",
		"TestReleaseV030GateCIModeAllowsMissingSecuritySignoffWithArtifact",
		"TestReleaseV030GateCIMissingSignoffWritesDetachedHashOutsideCanonicalManifest",
		"TestReleaseV030GateRequiresSecuritySignoffOutsideCIMode",
		"TestReleaseV030GateRequireCleanRequiresSecuritySignoffEvenInCIMode",
		"TestReleaseV030GateAcceptsSameRunSecuritySignoffWithFreshReportArtifacts",
		"TestReleaseV030GateAcceptsSecuritySignoffPathStartingWithDash",
		"TestReleaseV030GateWritesDetachedSecurityReviewHashOutsideCanonicalManifest",
		"TestReleaseV030GateBlocksFinalSummaryWhenPostSummaryArtifactHashCheckFails",
		"TestReleaseV030GateBlocksFinalSummaryWhenDetachedSecurityHashFails",
		"TestReleaseV030GateRecordsCIMissingSignoffFinalArtifactHashRefreshFailure",
		"TestReleaseV030GateCanonicalizesArtifactManifestWithPython3WhenPythonIsUnavailable",
		"TestReleaseV030GateRefreshesReleaseStateAfterFinalSummary",
		"countReleaseGateFailedSteps",
		"TestReleaseV030GateRejectsBuildOnlyRuntimeSmokeEvidence",
		"TestReleaseV030GateRejectsWrongVersionRuntimeSmokeEvidence",
		"TestReleaseV030GateDoesNotArchivePartialRuntimeEvidenceWhenWindowsReportInvalid",
		"TestReleaseV030GateDoesNotArchivePartialRuntimeEvidenceWhenRuntimeCopyFails",
		"TestReleaseV030GateAcceptsRuntimeSmokeSourcePathStartingWithDash",
		"TestReleaseV030GateRejectsWrongGitHeadRuntimeSmokeEvidence",
		"TestReleaseV030GateRejectsRunnerRuntimeSmokeEvidence",
		"TestReleaseV030GateRejectsInvalidTimestampRuntimeSmokeEvidence",
		"TestReleaseV030GateRejectsLooseTimestampRuntimeSmokeEvidence",
		"TestReleaseV030GateRejectsMissingRequiredRuntimeSmokeCase",
		"TestReleaseV030GateRejectsRuntimeSmokeCaseErrorText",
		"TestReleaseV030GateRejectsEmptyRuntimeSmokeCaseName",
		"TestReleaseV030GateRejectsNonStringRuntimeSmokeCaseName",
		"TestReleaseV030GateRejectsNonBooleanRuntimeSmokeCaseStatus",
		"TestReleaseV030GateRejectsNonIntegerRuntimeSmokeCaseExitFields",
		"TestReleaseV030GateRejectsNonIntegerRuntimeSmokeCounts",
		"TestReleaseV030GateRejectsNonBooleanRuntimeSmokeIslandsDebug",
		"TestReleaseV030GateRejectsNonBooleanRuntimeSmokeBuildOnly",
		"TestReleaseV030GateRejectsNonStringRuntimeSmokeRunner",
		"TestReleaseV040GateUsesDedicatedReadinessPreflight",
		"TestReleaseV040GateWritesBlockedSummaryOnReadinessFailure",
		"TestReleaseV040GateRejectsNonEmptyReportDirBeforeSideEffects",
		"TestCurrentSupportedSurfaceDocumentIsReleaseAligned",
		"TestBootstrapBuildsTetraAndTAlias",
	} {
		if strings.Contains(text, "\nfunc "+symbol+"(") {
			t.Fatalf("release_v1_test.go still contains %s", symbol)
		}
	}
}
