package gatecontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	Command             string   `json:"command"`
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
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Command string `json:"command"`
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
	ID        string `json:"id"`
	Statement string `json:"statement"`
}

type Nonclaim struct {
	ID        string `json:"id"`
	Statement string `json:"statement"`
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
		"command",
		"working_dir",
		"required",
		"report_outputs",
		"validator_refs",
		"host_preconditions",
		"blocked_status_policy",
	}); err != nil {
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
