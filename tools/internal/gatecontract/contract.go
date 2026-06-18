package gatecontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

const SchemaV1 = "tetra.gate-contract.v1"

type Contract struct {
	Schema               string              `json:"schema"`
	ID                   string              `json:"id"`
	Title                string              `json:"title"`
	Scope                string              `json:"scope"`
	Producer             string              `json:"producer"`
	Entrypoint           string              `json:"entrypoint"`
	FreshReportDirPolicy string              `json:"fresh_report_dir_policy"`
	HostPreconditions    []string            `json:"host_preconditions"`
	Steps                []Step              `json:"steps"`
	RequiredReports      []RequiredReport    `json:"required_reports"`
	Validators           []Validator         `json:"validators"`
	ArtifactHashes       *ArtifactHashPolicy `json:"artifact_hashes"`
	Claims               []Claim             `json:"claims"`
	Nonclaims            []Nonclaim          `json:"nonclaims"`
	CIArtifacts          []CIArtifact        `json:"ci_artifacts"`
}

type Step struct {
	ID                  string   `json:"id"`
	Kind                string   `json:"kind"`
	Command             string   `json:"command,omitempty"`
	CommandParts        []string `json:"command_parts,omitempty"`
	WorkingDir          string   `json:"working_dir"`
	Required            bool     `json:"required"`
	ReportOutputs       []string `json:"report_outputs"`
	ValidatorRefs       []string `json:"validator_refs"`
	HostPreconditions   []string `json:"host_preconditions"`
	BlockedStatusPolicy string   `json:"blocked_status_policy"`
}

type RequiredReport struct {
	Path                 string   `json:"path"`
	Schema               string   `json:"schema"`
	Validator            string   `json:"validator"`
	SameCommitRequired   bool     `json:"same_commit_required"`
	ArtifactHashRequired bool     `json:"artifact_hash_required"`
	ClaimRefs            []string `json:"claim_refs"`
}

type Validator struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	Command      string   `json:"command,omitempty"`
	CommandParts []string `json:"command_parts,omitempty"`
}

type ArtifactHashPolicy struct {
	Enabled   bool   `json:"enabled"`
	Required  bool   `json:"required"`
	Algorithm string `json:"algorithm"`
}

func (p ArtifactHashPolicy) RequiresReportHashes() bool {
	return p.Enabled || p.Required
}

type Claim struct {
	ID             string   `json:"id"`
	Statement      string   `json:"statement,omitempty"`
	StatementParts []string `json:"statement_parts,omitempty"`
}

type Nonclaim struct {
	ID             string   `json:"id"`
	Statement      string   `json:"statement,omitempty"`
	StatementParts []string `json:"statement_parts,omitempty"`
}

type CIArtifact struct {
	Path     string `json:"path"`
	Required bool   `json:"required"`
}

func Decode(raw []byte) (Contract, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return Contract{}, fmt.Errorf("empty gate contract")
	}
	if err := checkRequiredJSONFields(trimmed); err != nil {
		return Contract{}, err
	}
	var contract Contract
	if err := decodeStrictJSON(trimmed, &contract); err != nil {
		return Contract{}, err
	}
	if err := normalizeTextParts(&contract); err != nil {
		return Contract{}, err
	}
	if err := Validate(contract); err != nil {
		return Contract{}, err
	}
	return contract, nil
}

func Load(path string) (Contract, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Contract{}, err
	}
	contract, err := Decode(raw)
	if err != nil {
		return Contract{}, fmt.Errorf("load gate contract %q: %w", path, err)
	}
	return contract, nil
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}

func checkRequiredJSONFields(raw []byte) error {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	for _, field := range []string{
		"schema",
		"id",
		"title",
		"scope",
		"producer",
		"entrypoint",
		"fresh_report_dir_policy",
		"host_preconditions",
		"steps",
		"required_reports",
		"validators",
		"artifact_hashes",
		"claims",
		"nonclaims",
		"ci_artifacts",
	} {
		if _, ok := doc[field]; !ok {
			return fmt.Errorf("missing required field %q", field)
		}
	}
	if err := checkArrayObjectFields(doc["steps"], "steps", []string{
		"id",
		"kind",
		"working_dir",
		"required",
		"report_outputs",
		"validator_refs",
		"host_preconditions",
		"blocked_status_policy",
	}); err != nil {
		return err
	}
	if err := checkArrayObjectOneOf(doc["steps"], "steps", "command", "command_parts"); err != nil {
		return err
	}
	if err := checkArrayObjectFields(doc["required_reports"], "required_reports", []string{
		"path",
		"schema",
		"validator",
		"same_commit_required",
		"artifact_hash_required",
		"claim_refs",
	}); err != nil {
		return err
	}
	return nil
}

func normalizeTextParts(contract *Contract) error {
	for i := range contract.Steps {
		command, err := normalizeParts(
			fmt.Sprintf("steps[%d]", i),
			"command",
			contract.Steps[i].Command,
			contract.Steps[i].CommandParts,
		)
		if err != nil {
			return err
		}
		contract.Steps[i].Command = command
		contract.Steps[i].CommandParts = nil
	}
	for i := range contract.Validators {
		command, err := normalizeParts(
			fmt.Sprintf("validators[%d]", i),
			"command",
			contract.Validators[i].Command,
			contract.Validators[i].CommandParts,
		)
		if err != nil {
			return err
		}
		contract.Validators[i].Command = command
		contract.Validators[i].CommandParts = nil
	}
	for i := range contract.Claims {
		statement, err := normalizeParts(
			fmt.Sprintf("claims[%d]", i),
			"statement",
			contract.Claims[i].Statement,
			contract.Claims[i].StatementParts,
		)
		if err != nil {
			return err
		}
		contract.Claims[i].Statement = statement
		contract.Claims[i].StatementParts = nil
	}
	for i := range contract.Nonclaims {
		statement, err := normalizeParts(
			fmt.Sprintf("nonclaims[%d]", i),
			"statement",
			contract.Nonclaims[i].Statement,
			contract.Nonclaims[i].StatementParts,
		)
		if err != nil {
			return err
		}
		contract.Nonclaims[i].Statement = statement
		contract.Nonclaims[i].StatementParts = nil
	}
	return nil
}

func normalizeParts(label, field, value string, parts []string) (string, error) {
	if value != "" && len(parts) > 0 {
		return "", fmt.Errorf("%s: use %q or %q, not both", label, field, field+"_parts")
	}
	if len(parts) == 0 {
		return value, nil
	}
	for i, part := range parts {
		if part == "" {
			return "", fmt.Errorf("%s: %s_parts[%d] is empty", label, field, i)
		}
	}
	return strings.Join(parts, " "), nil
}

func checkArrayObjectFields(raw json.RawMessage, name string, required []string) error {
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return fmt.Errorf("%s: expected array of objects: %w", name, err)
	}
	if items == nil {
		return fmt.Errorf("%s: expected array of objects", name)
	}
	for i, item := range items {
		if item == nil {
			return fmt.Errorf("%s[%d]: expected object", name, i)
		}
		for _, field := range required {
			if _, ok := item[field]; !ok {
				return fmt.Errorf("%s[%d]: missing required field %q", name, i, field)
			}
		}
	}
	return nil
}

func checkArrayObjectOneOf(raw json.RawMessage, name, first, second string) error {
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return fmt.Errorf("%s: expected array of objects: %w", name, err)
	}
	for i, item := range items {
		if _, ok := item[first]; ok {
			continue
		}
		if _, ok := item[second]; ok {
			continue
		}
		return fmt.Errorf("%s[%d]: missing required field %q", name, i, first)
	}
	return nil
}
