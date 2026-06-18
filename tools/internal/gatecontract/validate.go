package gatecontract

import (
	"fmt"
	"strings"
)

func Validate(contract Contract) error {
	var issues []string

	if contract.Schema == "" {
		issues = append(issues, `missing required field "schema"`)
	} else if contract.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema = %q, want %q", contract.Schema, SchemaV1))
	}
	requireNonEmpty(&issues, "id", contract.ID)
	requireNonEmpty(&issues, "title", contract.Title)
	requireNonEmpty(&issues, "scope", contract.Scope)
	requireNonEmpty(&issues, "producer", contract.Producer)
	requireNonEmpty(&issues, "entrypoint", contract.Entrypoint)
	requireNonEmpty(&issues, "fresh_report_dir_policy", contract.FreshReportDirPolicy)
	requireSlicePresent(&issues, "host_preconditions", contract.HostPreconditions)
	requireSlicePresent(&issues, "nonclaims", contract.Nonclaims)
	requireSlicePresent(&issues, "ci_artifacts", contract.CIArtifacts)
	if contract.ArtifactHashes == nil {
		issues = append(issues, `missing required field "artifact_hashes"`)
	}
	if len(contract.Steps) == 0 {
		issues = append(issues, `missing required field "steps"`)
	}
	if len(contract.RequiredReports) == 0 {
		issues = append(issues, `missing required field "required_reports"`)
	}
	if len(contract.Validators) == 0 {
		issues = append(issues, `missing required field "validators"`)
	}
	if len(contract.Claims) == 0 {
		issues = append(issues, `missing required field "claims"`)
	}

	validatorIDs := map[string]struct{}{}
	for i, validator := range contract.Validators {
		if validator.ID == "" {
			issues = append(
				issues,
				fmt.Sprintf("validators[%d]: missing required field %q", i, "id"),
			)
			continue
		}
		if _, exists := validatorIDs[validator.ID]; exists {
			issues = append(issues, fmt.Sprintf("duplicate validator id %q", validator.ID))
		}
		validatorIDs[validator.ID] = struct{}{}
		if validator.Kind == "" {
			issues = append(
				issues,
				fmt.Sprintf("validator %q: missing required field %q", validator.ID, "kind"),
			)
		}
		if validator.Command == "" {
			issues = append(
				issues,
				fmt.Sprintf("validator %q: missing required field %q", validator.ID, "command"),
			)
		}
	}

	claimIDs := map[string]struct{}{}
	for i, claim := range contract.Claims {
		if claim.ID == "" {
			issues = append(issues, fmt.Sprintf("claims[%d]: missing required field %q", i, "id"))
			continue
		}
		if _, exists := claimIDs[claim.ID]; exists {
			issues = append(issues, fmt.Sprintf("duplicate claim id %q", claim.ID))
		}
		claimIDs[claim.ID] = struct{}{}
		if claim.Statement == "" {
			issues = append(
				issues,
				fmt.Sprintf("claim %q: missing required field %q", claim.ID, "statement"),
			)
		}
	}

	stepIDs := map[string]struct{}{}
	for i, step := range contract.Steps {
		stepLabel := fmt.Sprintf("steps[%d]", i)
		if step.ID == "" {
			issues = append(issues, fmt.Sprintf("%s: missing required field %q", stepLabel, "id"))
		} else {
			stepLabel = fmt.Sprintf("step %q", step.ID)
			if _, exists := stepIDs[step.ID]; exists {
				issues = append(issues, fmt.Sprintf("duplicate step id %q", step.ID))
			}
			stepIDs[step.ID] = struct{}{}
		}
		requireNestedNonEmpty(&issues, stepLabel, "kind", step.Kind)
		requireNestedNonEmpty(&issues, stepLabel, "command", step.Command)
		requireNestedNonEmpty(&issues, stepLabel, "working_dir", step.WorkingDir)
		requireNestedSlicePresent(&issues, stepLabel, "report_outputs", step.ReportOutputs)
		requireNestedSlicePresent(&issues, stepLabel, "validator_refs", step.ValidatorRefs)
		requireNestedSlicePresent(&issues, stepLabel, "host_preconditions", step.HostPreconditions)
		requireNestedNonEmpty(&issues, stepLabel, "blocked_status_policy", step.BlockedStatusPolicy)
		for j, ref := range step.ValidatorRefs {
			if ref == "" {
				issues = append(issues, fmt.Sprintf("%s validator_refs[%d] is empty", stepLabel, j))
				continue
			}
			if _, ok := validatorIDs[ref]; !ok {
				issues = append(
					issues,
					fmt.Sprintf(
						"%s validator_refs[%d] references missing validator %q",
						stepLabel,
						j,
						ref,
					),
				)
			}
		}
	}

	reportPaths := map[string]struct{}{}
	hashesRequired := contract.ArtifactHashes != nil &&
		contract.ArtifactHashes.RequiresReportHashes()
	for i, report := range contract.RequiredReports {
		reportLabel := fmt.Sprintf("required_reports[%d]", i)
		if report.Path == "" {
			issues = append(
				issues,
				fmt.Sprintf("%s: missing required field %q", reportLabel, "path"),
			)
		} else {
			reportLabel = fmt.Sprintf("required report %q", report.Path)
			if _, exists := reportPaths[report.Path]; exists {
				issues = append(issues, fmt.Sprintf("duplicate required report path %q", report.Path))
			}
			reportPaths[report.Path] = struct{}{}
		}
		requireNestedNonEmpty(&issues, reportLabel, "schema", report.Schema)
		requireNestedNonEmpty(&issues, reportLabel, "validator", report.Validator)
		requireNestedSlicePresent(&issues, reportLabel, "claim_refs", report.ClaimRefs)
		if report.Validator != "" {
			if _, ok := validatorIDs[report.Validator]; !ok {
				issues = append(
					issues,
					fmt.Sprintf(
						"%s references missing validator %q",
						reportLabel,
						report.Validator,
					),
				)
			}
		}
		if hashesRequired && !report.ArtifactHashRequired {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s has artifact_hash_required=false while artifact_hashes are required/enabled",
					reportLabel,
				),
			)
		}
		for j, ref := range report.ClaimRefs {
			if ref == "" {
				issues = append(issues, fmt.Sprintf("%s claim_refs[%d] is empty", reportLabel, j))
				continue
			}
			if _, ok := claimIDs[ref]; !ok {
				issues = append(
					issues,
					fmt.Sprintf(
						"%s claim_refs[%d] references missing claim %q",
						reportLabel,
						j,
						ref,
					),
				)
			}
		}
	}

	for i, nonclaim := range contract.Nonclaims {
		label := fmt.Sprintf("nonclaims[%d]", i)
		requireNestedNonEmpty(&issues, label, "id", nonclaim.ID)
		requireNestedNonEmpty(&issues, label, "statement", nonclaim.Statement)
	}

	for i, artifact := range contract.CIArtifacts {
		label := fmt.Sprintf("ci_artifacts[%d]", i)
		requireNestedNonEmpty(&issues, label, "path", artifact.Path)
	}

	if len(issues) > 0 {
		return fmt.Errorf("invalid gate contract: %s", strings.Join(issues, "; "))
	}
	return nil
}

func requireNonEmpty(issues *[]string, field string, value string) {
	if value == "" {
		*issues = append(*issues, fmt.Sprintf("missing required field %q", field))
	}
}

func requireSlicePresent[T any](issues *[]string, field string, value []T) {
	if value == nil {
		*issues = append(*issues, fmt.Sprintf("missing required field %q", field))
	}
}

func requireNestedNonEmpty(issues *[]string, owner string, field string, value string) {
	if value == "" {
		*issues = append(*issues, fmt.Sprintf("%s: missing required field %q", owner, field))
	}
}

func requireNestedSlicePresent[T any](issues *[]string, owner string, field string, value []T) {
	if value == nil {
		*issues = append(*issues, fmt.Sprintf("%s: missing required field %q", owner, field))
	}
}
