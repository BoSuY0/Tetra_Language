package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/tools/internal/ramvalidate"
)

func validateMemory100RAMContractBundle(reportDir string, gitHead string) []string {
	ramDir := filepath.Join(reportDir, "ram-contract")
	return validateMemory100RAMContractBundleAt(ramDir, gitHead)
}

func validateMemory100RAMContractBundleAt(ramDir string, gitHead string) []string {
	reportPath := filepath.Join(ramDir, "ram-contract-report.json")
	gradePath := filepath.Join(ramDir, "memory-grade-report.json")
	proofPath := filepath.Join(ramDir, "proof-store-summary.json")
	pipelinePath := filepath.Join(ramDir, "validation-pipeline-coverage.json")
	heapPath := filepath.Join(ramDir, "heap-blockers.json")
	copyPath := filepath.Join(ramDir, "copy-blockers.json")

	var issues []string
	var report ramvalidate.Report
	reportOK := false
	if err := ramvalidate.ReadStrictJSONFile(reportPath, &report); err != nil {
		issues = append(issues, fmt.Sprintf("ram-contract-report.json: %v", err))
	} else {
		reportOK = true
		if err := ramvalidate.ValidateReport(report); err != nil {
			issues = append(issues, fmt.Sprintf("ram-contract-report.json: %v", err))
		}
		if gitHead != "" && report.GitHead != gitHead {
			issues = append(issues, fmt.Sprintf("ram-contract-report.json git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
		}
	}
	if err := ramvalidate.ValidateGradeReportFile(gradePath); err != nil {
		issues = append(issues, fmt.Sprintf("memory-grade-report.json: %v", err))
	}
	if err := ramvalidate.ValidateProofStoreSummaryFile(proofPath); err != nil {
		issues = append(issues, fmt.Sprintf("proof-store-summary.json: %v", err))
	}
	if err := ramvalidate.ValidatePipelineCoverageFile(pipelinePath); err != nil {
		issues = append(issues, fmt.Sprintf("validation-pipeline-coverage.json: %v", err))
	}
	if err := ramvalidate.ValidateBlockerReportFile(heapPath, "heap"); err != nil {
		issues = append(issues, fmt.Sprintf("heap-blockers.json: %v", err))
	}
	if err := ramvalidate.ValidateBlockerReportFile(copyPath, "copy"); err != nil {
		issues = append(issues, fmt.Sprintf("copy-blockers.json: %v", err))
	}

	var heapBlockers ramvalidate.BlockerReport
	heapOK := false
	if err := ramvalidate.ReadStrictJSONFile(heapPath, &heapBlockers); err != nil {
		issues = append(issues, fmt.Sprintf("heap-blockers.json: %v", err))
	} else {
		heapOK = true
	}
	var copyBlockers ramvalidate.BlockerReport
	copyOK := false
	if err := ramvalidate.ReadStrictJSONFile(copyPath, &copyBlockers); err != nil {
		issues = append(issues, fmt.Sprintf("copy-blockers.json: %v", err))
	} else {
		copyOK = true
	}
	if reportOK && heapOK && copyOK {
		issues = append(issues, validateMemory100RAMHeapCopyClassification(report, heapBlockers, copyBlockers)...)
	}
	return issues
}

func validateMemory100RAMHeapCopyClassification(report ramvalidate.Report, heapBlockers ramvalidate.BlockerReport, copyBlockers ramvalidate.BlockerReport) []string {
	rowsBySite := map[string]ramvalidate.Row{}
	heapRows := map[string]ramvalidate.Row{}
	copyRows := map[string]ramvalidate.Row{}
	var issues []string
	for i, row := range report.Rows {
		if strings.TrimSpace(row.SiteID) == "" {
			continue
		}
		rowsBySite[row.SiteID] = row
		if memory100RAMRowIsHeap(row) {
			heapRows[row.SiteID] = row
			if memory100RAMUnclassified(row.ValidationStatus) {
				issues = append(issues, fmt.Sprintf("ram-contract-report.json heap row %d site_id %q has unclassified validation_status %q", i, row.SiteID, row.ValidationStatus))
			}
			if len(nonEmptyMemory100Strings(row.Blockers)) == 0 {
				issues = append(issues, fmt.Sprintf("ram-contract-report.json heap row %d site_id %q has no classified blockers", i, row.SiteID))
			}
			for _, blocker := range row.Blockers {
				if memory100RAMUnclassified(blocker) {
					issues = append(issues, fmt.Sprintf("ram-contract-report.json heap row %d site_id %q has unclassified blocker %q", i, row.SiteID, blocker))
				}
			}
		}
		if memory100RAMRowIsCopy(row) {
			copyRows[row.SiteID] = row
			if memory100RAMUnclassified(row.ValidationStatus) {
				issues = append(issues, fmt.Sprintf("ram-contract-report.json copy row %d site_id %q has unclassified validation_status %q", i, row.SiteID, row.ValidationStatus))
			}
			if memory100RAMUnclassified(row.CopyReason) {
				issues = append(issues, fmt.Sprintf("ram-contract-report.json copy row %d site_id %q has unclassified copy_reason %q", i, row.SiteID, row.CopyReason))
			}
		}
	}

	heapBlockerSites := map[string]ramvalidate.BlockerRow{}
	for i, row := range heapBlockers.Rows {
		if memory100RAMUnclassified(strings.Join(row.Blockers, "\n")) {
			issues = append(issues, fmt.Sprintf("heap-blockers.json row %d site_id %q has unclassified blockers", i, row.SiteID))
		}
		heapBlockerSites[row.SiteID] = row
		ramRow, ok := rowsBySite[row.SiteID]
		if !ok {
			issues = append(issues, fmt.Sprintf("heap-blockers.json row %d site_id %q missing from ram-contract-report.json", i, row.SiteID))
			continue
		}
		if !memory100RAMRowIsHeap(ramRow) {
			issues = append(issues, fmt.Sprintf("heap-blockers.json row %d site_id %q is not a heap RAM report row", i, row.SiteID))
		}
	}
	for siteID := range heapRows {
		if _, ok := heapBlockerSites[siteID]; !ok {
			issues = append(issues, fmt.Sprintf("ram-contract-report.json heap row site_id %q missing from heap-blockers.json", siteID))
		}
	}

	copyBlockerSites := map[string]ramvalidate.BlockerRow{}
	for i, row := range copyBlockers.Rows {
		if memory100RAMUnclassified(row.CopyReason) {
			issues = append(issues, fmt.Sprintf("copy-blockers.json row %d site_id %q has unclassified copy_reason", i, row.SiteID))
		}
		copyBlockerSites[row.SiteID] = row
		ramRow, ok := rowsBySite[row.SiteID]
		if !ok {
			issues = append(issues, fmt.Sprintf("copy-blockers.json row %d site_id %q missing from ram-contract-report.json", i, row.SiteID))
			continue
		}
		if !memory100RAMRowIsCopy(ramRow) {
			issues = append(issues, fmt.Sprintf("copy-blockers.json row %d site_id %q is not a copy RAM report row", i, row.SiteID))
		}
	}
	for siteID := range copyRows {
		if _, ok := copyBlockerSites[siteID]; !ok {
			issues = append(issues, fmt.Sprintf("ram-contract-report.json copy row site_id %q missing from copy-blockers.json", siteID))
		}
	}
	return issues
}

func validateMemory100AllocationLoweringRAMConsistency(reportDir string) []string {
	allocationPath := filepath.Join(reportDir, "allocation-lowering", "allocation-lowering-report.json")
	ramPath := filepath.Join(reportDir, "ram-contract", "ram-contract-report.json")
	heapPath := filepath.Join(reportDir, "ram-contract", "heap-blockers.json")
	copyPath := filepath.Join(reportDir, "ram-contract", "copy-blockers.json")
	var allocation memory100AllocationLoweringReport
	if err := readMemory100JSON(allocationPath, &allocation); err != nil {
		return []string{fmt.Sprintf("allocation lowering invalid: %v", err)}
	}
	var report ramvalidate.Report
	if err := ramvalidate.ReadStrictJSONFile(ramPath, &report); err != nil {
		return []string{fmt.Sprintf("ram-contract-report.json: %v", err)}
	}
	var heapBlockers ramvalidate.BlockerReport
	if err := ramvalidate.ReadStrictJSONFile(heapPath, &heapBlockers); err != nil {
		return []string{fmt.Sprintf("heap-blockers.json: %v", err)}
	}
	var copyBlockers ramvalidate.BlockerReport
	if err := ramvalidate.ReadStrictJSONFile(copyPath, &copyBlockers); err != nil {
		return []string{fmt.Sprintf("copy-blockers.json: %v", err)}
	}

	proofBackedTrustedRows := 0
	rowsBySite := map[string]ramvalidate.Row{}
	proofBackedTrustedSites := map[string]struct{}{}
	heapSites := map[string]struct{}{}
	copySites := map[string]struct{}{}
	for _, row := range report.Rows {
		if strings.TrimSpace(row.SiteID) == "" {
			continue
		}
		rowsBySite[row.SiteID] = row
		if memory100RAMRowIsHeap(row) {
			heapSites[row.SiteID] = struct{}{}
		}
		if memory100RAMRowIsCopy(row) {
			copySites[row.SiteID] = struct{}{}
		}
		if memory100RAMTrustedPlacement(row.Placement) &&
			row.EscapeStatus == "no_escape" &&
			row.ValidationStatus == "validated" &&
			len(nonEmptyMemory100Strings(row.ProofIDs)) > 0 {
			proofBackedTrustedRows++
			proofBackedTrustedSites[row.SiteID] = struct{}{}
		}
	}
	heapBlockerSites := map[string]struct{}{}
	for _, row := range heapBlockers.Rows {
		if strings.TrimSpace(row.SiteID) != "" {
			heapBlockerSites[row.SiteID] = struct{}{}
		}
	}
	copyBlockerSites := map[string]struct{}{}
	for _, row := range copyBlockers.Rows {
		if strings.TrimSpace(row.SiteID) != "" {
			copyBlockerSites[row.SiteID] = struct{}{}
		}
	}

	var issues []string
	proofCoveredSites := map[string]struct{}{}
	for _, decision := range allocation.Decisions {
		status := memory100AllocationDecisionStatus(decision)
		coveredSites := memory100StringSet(decision.CoveredSiteIDs)
		for siteID := range coveredSites {
			row, ok := rowsBySite[siteID]
			if !ok {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s covered_site_ids site_id %q missing from ram-contract-report.json", decision.Name, siteID))
				continue
			}
			if !memory100AllocationActualStorageMatchesRAMRow(decision, row) {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s actual_lowering_storage %q contradicts ram-contract-report.json site_id %q placement %q", decision.Name, decision.ActualLoweringStorage, siteID, row.Placement))
			}
		}
		if status == "not_observed" {
			switch decision.Name {
			case "heap_fallback_blocker":
				if len(heapSites) > 0 {
					issues = append(issues, fmt.Sprintf("allocation lowering decision %s is not_observed but ram-contract-report.json has heap rows %v", decision.Name, memory100SortedSetKeys(heapSites)))
				}
			case "copy_blocker":
				if len(copySites) > 0 {
					issues = append(issues, fmt.Sprintf("allocation lowering decision %s is not_observed but ram-contract-report.json has copy rows %v", decision.Name, memory100SortedSetKeys(copySites)))
				}
			}
			continue
		}
		if strings.TrimSpace(decision.ProofArtifact) == "" {
			switch decision.Name {
			case "heap_fallback_blocker":
				for siteID := range coveredSites {
					if _, ok := heapSites[siteID]; !ok {
						issues = append(issues, fmt.Sprintf("allocation lowering decision %s covered_site_ids site_id %q is not a heap RAM row", decision.Name, siteID))
					}
					if _, ok := heapBlockerSites[siteID]; !ok {
						issues = append(issues, fmt.Sprintf("allocation lowering decision %s covered_site_ids site_id %q missing from heap-blockers.json", decision.Name, siteID))
					}
				}
				for _, siteID := range memory100MissingSetKeys(heapSites, coveredSites) {
					issues = append(issues, fmt.Sprintf("allocation lowering decision %s missing heap RAM row site_id %q in covered_site_ids", decision.Name, siteID))
				}
			case "copy_blocker":
				for siteID := range coveredSites {
					if _, ok := copySites[siteID]; !ok {
						issues = append(issues, fmt.Sprintf("allocation lowering decision %s covered_site_ids site_id %q is not a copy RAM row", decision.Name, siteID))
					}
					if _, ok := copyBlockerSites[siteID]; !ok {
						issues = append(issues, fmt.Sprintf("allocation lowering decision %s covered_site_ids site_id %q missing from copy-blockers.json", decision.Name, siteID))
					}
				}
				for _, siteID := range memory100MissingSetKeys(copySites, coveredSites) {
					issues = append(issues, fmt.Sprintf("allocation lowering decision %s missing copy RAM row site_id %q in covered_site_ids", decision.Name, siteID))
				}
			}
			continue
		}
		if proofBackedTrustedRows == 0 {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s is proof-backed but ram-contract-report.json has no proof-backed trusted rows", decision.Name))
		}
		for siteID := range coveredSites {
			if _, ok := proofBackedTrustedSites[siteID]; !ok {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s covered_site_ids site_id %q is not a proof-backed trusted RAM row", decision.Name, siteID))
				continue
			}
			if status == "proven" {
				proofCoveredSites[siteID] = struct{}{}
			}
		}
	}
	for _, siteID := range memory100MissingSetKeys(proofBackedTrustedSites, proofCoveredSites) {
		issues = append(issues, fmt.Sprintf("allocation lowering missing proof-backed trusted RAM row site_id %q in proven covered_site_ids", siteID))
	}
	return issues
}

func memory100StringSet(values []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	return set
}

func memory100MissingSetKeys(want map[string]struct{}, got map[string]struct{}) []string {
	var missing []string
	for value := range want {
		if _, ok := got[value]; !ok {
			missing = append(missing, value)
		}
	}
	sort.Strings(missing)
	return missing
}

func memory100SortedSetKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func memory100RAMRowIsHeap(row ramvalidate.Row) bool {
	return row.Placement == "heap_bounded" || row.Placement == "heap_unbounded"
}

func memory100RAMRowIsCopy(row ramvalidate.Row) bool {
	return strings.HasPrefix(row.Intent, "copy")
}

func memory100RAMUnclassified(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "unknown", "unclassified", "todo", "tbd", "none", "n/a":
		return true
	default:
		return false
	}
}

func memory100RAMTrustedPlacement(placement string) bool {
	switch placement {
	case "eliminated", "register", "stack", "static", "interned", "island", "region":
		return true
	default:
		return false
	}
}

func memory100AllocationActualStorageMatchesRAMRow(decision memory100AllocationLoweringDecision, row ramvalidate.Row) bool {
	actual := strings.TrimSpace(decision.ActualLoweringStorage)
	switch actual {
	case "Copy":
		return memory100RAMRowIsCopy(row)
	case "Eliminated":
		return row.Placement == "eliminated"
	case "Register":
		return row.Placement == "register"
	case "Stack":
		return row.Placement == "stack"
	case "Static":
		return row.Placement == "static"
	case "Interned":
		return row.Placement == "interned"
	case "Region", "FunctionTempRegion", "TaskRegion", "ActorMoveRegion":
		return row.Placement == "region"
	case "ExplicitIsland":
		return row.Placement == "island"
	case "Heap", "LargeMmap":
		return memory100RAMRowIsHeap(row)
	case "External":
		return row.Placement == "external"
	case "Rejected":
		return row.Placement == "rejected"
	default:
		return actual == ""
	}
}

func validateMemory100RawMemoryContract(path string, gitHead string) []string {
	var report struct {
		Status     string `json:"status"`
		GitHead    string `json:"git_head"`
		Operations []struct {
			Name            string   `json:"name"`
			SourceArtifacts []string `json:"source_artifacts"`
			PositiveTests   []string `json:"positive_tests"`
			NegativeTests   []string `json:"negative_tests"`
			NonClaims       []string `json:"non_claims"`
		} `json:"operations"`
	}
	if err := readMemory100JSON(path, &report); err != nil {
		return []string{fmt.Sprintf("raw memory contract invalid: %v", err)}
	}
	var issues []string
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("raw memory contract status is %q, want pass", report.Status))
	}
	if gitHead != "" && report.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("raw memory contract git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
	}
	byName := map[string]struct {
		SourceArtifacts []string
		PositiveTests   []string
		NegativeTests   []string
		NonClaims       []string
	}{}
	for _, operation := range report.Operations {
		byName[operation.Name] = struct {
			SourceArtifacts []string
			PositiveTests   []string
			NegativeTests   []string
			NonClaims       []string
		}{
			SourceArtifacts: operation.SourceArtifacts,
			PositiveTests:   operation.PositiveTests,
			NegativeTests:   operation.NegativeTests,
			NonClaims:       operation.NonClaims,
		}
	}
	for _, name := range []string{"core.alloc_bytes", "core.ptr_add", "raw_slice_from_parts", "raw_load_store_metadata", "memcpy_u8", "memset_u8", "cap.mem"} {
		operation, ok := byName[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("raw memory contract missing operation %s", name))
			continue
		}
		if len(nonEmptyMemory100Strings(operation.SourceArtifacts)) == 0 {
			issues = append(issues, fmt.Sprintf("raw memory contract operation %s missing source_artifacts", name))
		}
		if len(nonEmptyMemory100Strings(operation.PositiveTests))+len(nonEmptyMemory100Strings(operation.NegativeTests)) == 0 {
			issues = append(issues, fmt.Sprintf("raw memory contract operation %s missing positive_tests or negative_tests", name))
		}
		if name == "cap.mem" && len(nonEmptyMemory100Strings(operation.NonClaims)) == 0 {
			issues = append(issues, "raw memory contract operation cap.mem missing non_claims")
		}
		switch name {
		case "core.alloc_bytes":
			issues = append(issues, requireMemory100RawEvidence(name, "source_artifacts", operation.SourceArtifacts, "compiler/internal/runtimeabi/raw_pointer_bounds_test.go")...)
			issues = append(issues, requireMemory100RawEvidence(name, "positive_tests", operation.PositiveTests, "allocation-base metadata")...)
		case "core.ptr_add":
			issues = append(issues, requireMemory100RawEvidence(name, "source_artifacts", operation.SourceArtifacts, "compiler/internal/runtimeabi/raw_pointer_bounds_test.go")...)
			issues = append(issues, requireMemory100RawEvidence(name, "negative_tests", operation.NegativeTests, "negative offset", "upper bound", "access-width overflow")...)
		case "raw_slice_from_parts":
			issues = append(issues, requireMemory100RawEvidence(name, "source_artifacts", operation.SourceArtifacts, "compiler/internal/runtimeabi/raw_pointer_bounds_test.go", "compiler/tests/semantics/memory_ideal_v5_raw_pointer_test.go")...)
			issues = append(issues, requireMemory100RawEvidence(name, "negative_tests", operation.NegativeTests, "outside unsafe", "negative length", "i32 byte overflow")...)
		case "raw_load_store_metadata":
			issues = append(issues, requireMemory100RawEvidence(name, "source_artifacts", operation.SourceArtifacts, "compiler/internal/plir/plir_test.go", "compiler/internal/lower/raw_memory_test.go", "compiler/internal/memoryfacts/from_plir_test.go")...)
			issues = append(issues, requireMemory100RawEvidence(name, "positive_tests", operation.PositiveTests, "IRMemWriteI32Offset", "IRMemReadI32Offset", "raw memory gateway", "UnsafeChecked")...)
			issues = append(issues, requireMemory100RawEvidence(name, "negative_tests", operation.NegativeTests, "checked_external_unknown", "rejected_access_width_overflow")...)
		case "memcpy_u8":
			issues = append(issues, requireMemory100RawEvidence(name, "source_artifacts", operation.SourceArtifacts, "lib/core/memory.tetra")...)
			issues = append(issues, requireMemory100RawEvidence(name, "negative_tests", operation.NegativeTests, "negative length", "access-width overflow")...)
			issues = append(issues, requireMemory100RawEvidence(name, "non_claims", operation.NonClaims, "overlapping memcpy")...)
		case "memset_u8":
			issues = append(issues, requireMemory100RawEvidence(name, "source_artifacts", operation.SourceArtifacts, "lib/core/memory.tetra")...)
			issues = append(issues, requireMemory100RawEvidence(name, "negative_tests", operation.NegativeTests, "negative length", "access-width overflow")...)
		case "cap.mem":
			issues = append(issues, requireMemory100RawEvidence(name, "source_artifacts", operation.SourceArtifacts, "lib/core/capability.tetra")...)
			issues = append(issues, requireMemory100RawEvidence(name, "negative_tests", operation.NegativeTests, "unsafe_unknown", "overclaim")...)
			issues = append(issues, requireMemory100RawEvidence(name, "non_claims", operation.NonClaims, "no arbitrary external pointer safety claim")...)
		}
	}
	return issues
}

func requireMemory100RawEvidence(operation string, field string, values []string, wants ...string) []string {
	joined := strings.ToLower(strings.Join(nonEmptyMemory100Strings(values), "\n"))
	var issues []string
	for _, want := range wants {
		if !strings.Contains(joined, strings.ToLower(want)) {
			issues = append(issues, fmt.Sprintf("raw memory contract operation %s %s missing %q", operation, field, want))
		}
	}
	return issues
}

func validateMemory100AllocationLowering(path string, gitHead string) []string {
	var report memory100AllocationLoweringReport
	if err := readMemory100JSON(path, &report); err != nil {
		return []string{fmt.Sprintf("allocation lowering invalid: %v", err)}
	}
	var issues []string
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("allocation lowering status is %q, want pass", report.Status))
	}
	if gitHead != "" && report.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("allocation lowering git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
	}
	byName := map[string]struct {
		Status                string
		PlannedStorage        string
		ActualLoweringStorage string
		ProofArtifact         string
		BlockerArtifact       string
		BlockerReason         string
		BudgetImpact          string
		GradeImpact           string
		ValidatorCoverage     []string
		SourceArtifacts       []string
		CoveredSiteIDs        []string
	}{}
	for _, decision := range report.Decisions {
		byName[decision.Name] = struct {
			Status                string
			PlannedStorage        string
			ActualLoweringStorage string
			ProofArtifact         string
			BlockerArtifact       string
			BlockerReason         string
			BudgetImpact          string
			GradeImpact           string
			ValidatorCoverage     []string
			SourceArtifacts       []string
			CoveredSiteIDs        []string
		}{
			Status:                decision.Status,
			PlannedStorage:        decision.PlannedStorage,
			ActualLoweringStorage: decision.ActualLoweringStorage,
			ProofArtifact:         decision.ProofArtifact,
			BlockerArtifact:       decision.BlockerArtifact,
			BlockerReason:         decision.BlockerReason,
			BudgetImpact:          decision.BudgetImpact,
			GradeImpact:           decision.GradeImpact,
			ValidatorCoverage:     decision.ValidatorCoverage,
			SourceArtifacts:       decision.SourceArtifacts,
			CoveredSiteIDs:        decision.CoveredSiteIDs,
		}
	}
	for _, name := range []string{"stack_trusted_no_escape", "heap_fallback_blocker", "copy_blocker", "lowering_storage_match"} {
		decision, ok := byName[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("allocation lowering missing decision %s", name))
			continue
		}
		if strings.TrimSpace(decision.PlannedStorage) == "" || strings.TrimSpace(decision.ActualLoweringStorage) == "" {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s missing planned/actual storage", name))
		}
		if len(nonEmptyMemory100Strings(decision.SourceArtifacts)) == 0 {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s missing source_artifacts", name))
		}
		status := memory100AllocationDecisionStatus(memory100AllocationLoweringDecision{
			Status:          decision.Status,
			ProofArtifact:   decision.ProofArtifact,
			BlockerArtifact: decision.BlockerArtifact,
		})
		switch status {
		case "", "proven", "blocked", "not_observed":
		default:
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s has unknown status %q", name, decision.Status))
		}
		planned := strings.TrimSpace(decision.PlannedStorage)
		actual := strings.TrimSpace(decision.ActualLoweringStorage)
		if planned != "" && actual != "" && !strings.EqualFold(planned, actual) {
			if strings.TrimSpace(decision.BlockerArtifact) == "" {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s planned/actual mismatch %s -> %s requires blocker_artifact", name, planned, actual))
			}
			issues = append(issues, requireMemory100AllocationField(name, "blocker_reason", decision.BlockerReason)...)
			issues = append(issues, requireMemory100AllocationField(name, "budget_impact", decision.BudgetImpact)...)
			issues = append(issues, requireMemory100AllocationField(name, "grade_impact", decision.GradeImpact)...)
		}
		if status == "not_observed" {
			if strings.TrimSpace(decision.ProofArtifact) != "" || strings.TrimSpace(decision.BlockerArtifact) != "" {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s is not_observed but carries proof/blocker artifact", name))
			}
			if len(nonEmptyMemory100Strings(decision.CoveredSiteIDs)) > 0 {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s is not_observed but carries covered_site_ids", name))
			}
			continue
		}
		if len(nonEmptyMemory100Strings(decision.CoveredSiteIDs)) == 0 {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s missing covered_site_ids", name))
		}
		if status == "proven" && strings.TrimSpace(decision.ProofArtifact) == "" {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s status proven requires proof_artifact", name))
		}
		if status == "blocked" && strings.TrimSpace(decision.BlockerArtifact) == "" {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s status blocked requires blocker_artifact", name))
		}
		if strings.TrimSpace(decision.ProofArtifact) == "" && strings.TrimSpace(decision.BlockerArtifact) == "" {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s missing proof_artifact or blocker_artifact", name))
		}
		switch name {
		case "heap_fallback_blocker":
			if !strings.Contains(filepath.ToSlash(decision.BlockerArtifact), "ram-contract/heap-blockers.json") {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s blocker_artifact must reference ram-contract/heap-blockers.json", name))
			}
			issues = append(issues, requireMemory100AllocationField(name, "blocker_reason", decision.BlockerReason)...)
			issues = append(issues, requireMemory100AllocationField(name, "budget_impact", decision.BudgetImpact)...)
			issues = append(issues, requireMemory100AllocationField(name, "grade_impact", decision.GradeImpact)...)
			issues = append(issues, requireMemory100AllocationCoverage(name, decision.ValidatorCoverage, "validate-heap-blockers", "validate-ram-contract-release")...)
		case "copy_blocker":
			if !strings.Contains(filepath.ToSlash(decision.BlockerArtifact), "ram-contract/copy-blockers.json") {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s blocker_artifact must reference ram-contract/copy-blockers.json", name))
			}
			issues = append(issues, requireMemory100AllocationField(name, "blocker_reason", decision.BlockerReason)...)
			issues = append(issues, requireMemory100AllocationField(name, "budget_impact", decision.BudgetImpact)...)
			issues = append(issues, requireMemory100AllocationField(name, "grade_impact", decision.GradeImpact)...)
			issues = append(issues, requireMemory100AllocationCoverage(name, decision.ValidatorCoverage, "validate-copy-blockers", "validate-ram-contract-release")...)
		}
	}
	return issues
}

func memory100AllocationDecisionStatus(decision memory100AllocationLoweringDecision) string {
	status := strings.TrimSpace(decision.Status)
	if status != "" {
		return status
	}
	if strings.TrimSpace(decision.ProofArtifact) != "" {
		return "proven"
	}
	if strings.TrimSpace(decision.BlockerArtifact) != "" {
		return "blocked"
	}
	return ""
}

func requireMemory100AllocationField(decision string, field string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{fmt.Sprintf("allocation lowering decision %s missing %s", decision, field)}
	}
	return nil
}

func requireMemory100AllocationCoverage(decision string, values []string, wants ...string) []string {
	joined := strings.ToLower(strings.Join(nonEmptyMemory100Strings(values), "\n"))
	var issues []string
	for _, want := range wants {
		if !strings.Contains(joined, strings.ToLower(want)) {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s validator_coverage missing %q", decision, want))
		}
	}
	return issues
}
