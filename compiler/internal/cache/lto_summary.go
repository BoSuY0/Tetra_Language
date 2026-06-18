package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

const IncrementalModuleSummarySchemaVersion = "tetra.incremental.module_summary.v1"

type IncrementalModuleSummaryInput struct {
	Module           string
	Target           string
	BuildTag         string
	Source           []byte
	DependencyHash   [32]byte
	PublicAPIHash    string
	ExternalCallees  []string
	ExternalTypeDeps []string
}

type IncrementalModuleSummary struct {
	SchemaVersion    string   `json:"schema_version"`
	Module           string   `json:"module"`
	Target           string   `json:"target"`
	BuildTag         string   `json:"build_tag,omitempty"`
	SourceHash       string   `json:"source_hash"`
	DependencyHash   string   `json:"dependency_hash"`
	PublicAPIHash    string   `json:"public_api_hash"`
	ExternalCallees  []string `json:"external_callees,omitempty"`
	ExternalTypeDeps []string `json:"external_type_deps,omitempty"`
	ValidationRows   []string `json:"validation_rows"`
	CodegenConsumer  bool     `json:"codegen_consumer"`
	LinkerConsumer   bool     `json:"linker_consumer"`
}

func BuildIncrementalModuleSummary(
	input IncrementalModuleSummaryInput,
) (IncrementalModuleSummary, error) {
	sourceHash := sha256.Sum256(input.Source)
	summary := IncrementalModuleSummary{
		SchemaVersion:    IncrementalModuleSummarySchemaVersion,
		Module:           input.Module,
		Target:           input.Target,
		BuildTag:         input.BuildTag,
		SourceHash:       formatSummaryHash(sourceHash),
		DependencyHash:   formatSummaryHash(input.DependencyHash),
		PublicAPIHash:    input.PublicAPIHash,
		ExternalCallees:  append([]string(nil), input.ExternalCallees...),
		ExternalTypeDeps: append([]string(nil), input.ExternalTypeDeps...),
		ValidationRows: []string{
			"source_hash",
			"dependency_hash_contract",
			"public_api_hash",
			"cross_module_signature_inputs",
			"non_consumer_boundary",
		},
		CodegenConsumer: false,
		LinkerConsumer:  false,
	}
	summary = canonicalIncrementalModuleSummary(summary)
	if err := ValidateIncrementalModuleSummary(summary); err != nil {
		return IncrementalModuleSummary{}, err
	}
	return summary, nil
}

func MarshalIncrementalModuleSummary(summary IncrementalModuleSummary) ([]byte, error) {
	summary = canonicalIncrementalModuleSummary(summary)
	if err := ValidateIncrementalModuleSummary(summary); err != nil {
		return nil, err
	}
	out, err := json.Marshal(summary)
	if err != nil {
		return nil, fmt.Errorf("incremental module summary: marshal: %w", err)
	}
	return out, nil
}

func ParseIncrementalModuleSummary(raw []byte) (IncrementalModuleSummary, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var summary IncrementalModuleSummary
	if err := dec.Decode(&summary); err != nil {
		return IncrementalModuleSummary{}, fmt.Errorf("incremental module summary: decode: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return IncrementalModuleSummary{}, fmt.Errorf(
				"incremental module summary: trailing JSON value",
			)
		}
		return IncrementalModuleSummary{}, fmt.Errorf(
			"incremental module summary: trailing JSON: %w",
			err,
		)
	}
	summary = canonicalIncrementalModuleSummary(summary)
	if err := ValidateIncrementalModuleSummary(summary); err != nil {
		return IncrementalModuleSummary{}, err
	}
	return summary, nil
}

func ValidateIncrementalModuleSummary(summary IncrementalModuleSummary) error {
	if summary.SchemaVersion != IncrementalModuleSummarySchemaVersion {
		return fmt.Errorf(
			"incremental module summary: schema_version = %q, want %q",
			summary.SchemaVersion,
			IncrementalModuleSummarySchemaVersion,
		)
	}
	if strings.TrimSpace(summary.Module) == "" {
		return fmt.Errorf("incremental module summary: missing module")
	}
	if strings.TrimSpace(summary.Target) == "" {
		return fmt.Errorf("incremental module summary: missing target")
	}
	for name, value := range map[string]string{
		"source_hash":     summary.SourceHash,
		"dependency_hash": summary.DependencyHash,
		"public_api_hash": summary.PublicAPIHash,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("incremental module summary: missing %s", name)
		}
		if !strings.HasPrefix(value, "sha256:") {
			return fmt.Errorf("incremental module summary: %s must use sha256: prefix", name)
		}
	}
	for _, name := range summary.ExternalCallees {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("incremental module summary: empty external callee")
		}
	}
	for _, name := range summary.ExternalTypeDeps {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("incremental module summary: empty external type dependency")
		}
	}
	for _, row := range []string{
		"source_hash",
		"dependency_hash_contract",
		"public_api_hash",
		"cross_module_signature_inputs",
		"non_consumer_boundary",
	} {
		if !summaryHasString(summary.ValidationRows, row) {
			return fmt.Errorf("incremental module summary: missing validation row %q", row)
		}
	}
	if summary.CodegenConsumer {
		return fmt.Errorf(
			"incremental module summary: codegen consumer is not supported for evidence-only summary",
		)
	}
	if summary.LinkerConsumer {
		return fmt.Errorf(
			"incremental module summary: linker consumer is not supported for evidence-only summary",
		)
	}
	return nil
}

func canonicalIncrementalModuleSummary(summary IncrementalModuleSummary) IncrementalModuleSummary {
	summary.ExternalCallees = canonicalSummaryStrings(summary.ExternalCallees)
	summary.ExternalTypeDeps = canonicalSummaryStrings(summary.ExternalTypeDeps)
	summary.ValidationRows = canonicalSummaryStrings(summary.ValidationRows)
	return summary
}

func canonicalSummaryStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func summaryHasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func formatSummaryHash(hash [32]byte) string {
	return fmt.Sprintf("sha256:%x", hash)
}
