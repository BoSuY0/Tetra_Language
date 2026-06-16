package gatecontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeAcceptsValidMinimalContract(t *testing.T) {
	got, err := Decode(contractJSON(t, nil))
	if err != nil {
		t.Fatalf("Decode rejected valid minimal contract: %v", err)
	}

	if got.Schema != SchemaV1 {
		t.Fatalf("Schema = %q, want %q", got.Schema, SchemaV1)
	}
	if got.ID != "surface-release-crash-reporting" {
		t.Fatalf("ID = %q", got.ID)
	}
	if len(got.Steps) != 1 || got.Steps[0].ID != "run-crash-report-smoke" {
		t.Fatalf("Steps = %#v", got.Steps)
	}
}

func TestLoadReadsAndValidatesContractFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gate-contract.json")
	if err := os.WriteFile(path, contractJSON(t, nil), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load rejected valid contract file: %v", err)
	}
	if got.ID != "surface-release-crash-reporting" {
		t.Fatalf("ID = %q", got.ID)
	}
}

func TestDecodeRejectsInvalidContract(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(map[string]any)
		want   []string
	}{
		{
			name: "wrong schema",
			mutate: func(doc map[string]any) {
				doc["schema"] = "tetra.gate-contract.v0"
			},
			want: []string{"schema", SchemaV1},
		},
		{
			name: "unknown top level field",
			mutate: func(doc map[string]any) {
				doc["extra"] = true
			},
			want: []string{"unknown field", "extra"},
		},
		{
			name: "missing top level required field",
			mutate: func(doc map[string]any) {
				delete(doc, "title")
			},
			want: []string{"missing required field", "title"},
		},
		{
			name: "missing step required field",
			mutate: func(doc map[string]any) {
				step := doc["steps"].([]any)[0].(map[string]any)
				delete(step, "command")
			},
			want: []string{"steps[0]", "command"},
		},
		{
			name: "missing required report field",
			mutate: func(doc map[string]any) {
				report := doc["required_reports"].([]any)[0].(map[string]any)
				delete(report, "path")
			},
			want: []string{"required_reports[0]", "path"},
		},
		{
			name: "duplicate step IDs",
			mutate: func(doc map[string]any) {
				steps := doc["steps"].([]any)
				steps = append(steps, cloneObject(steps[0].(map[string]any)))
				doc["steps"] = steps
			},
			want: []string{"duplicate step id", "run-crash-report-smoke"},
		},
		{
			name: "duplicate report paths",
			mutate: func(doc map[string]any) {
				reports := doc["required_reports"].([]any)
				reports = append(reports, cloneObject(reports[0].(map[string]any)))
				doc["required_reports"] = reports
			},
			want: []string{"duplicate required report path", "reports/surface/crash-report.json"},
		},
		{
			name: "duplicate validator IDs",
			mutate: func(doc map[string]any) {
				validators := doc["validators"].([]any)
				validators = append(validators, cloneObject(validators[0].(map[string]any)))
				doc["validators"] = validators
			},
			want: []string{"duplicate validator id", "surface-crash-report"},
		},
		{
			name: "step references missing validator",
			mutate: func(doc map[string]any) {
				step := doc["steps"].([]any)[0].(map[string]any)
				step["validator_refs"] = []any{"missing-validator"}
			},
			want: []string{"validator_refs", "missing-validator"},
		},
		{
			name: "required report references missing validator",
			mutate: func(doc map[string]any) {
				report := doc["required_reports"].([]any)[0].(map[string]any)
				report["validator"] = "missing-validator"
			},
			want: []string{"required report", "missing-validator"},
		},
		{
			name: "required report missing artifact hash while hashes required",
			mutate: func(doc map[string]any) {
				report := doc["required_reports"].([]any)[0].(map[string]any)
				report["artifact_hash_required"] = false
			},
			want: []string{"artifact_hash_required", "reports/surface/crash-report.json"},
		},
		{
			name: "report references missing claim",
			mutate: func(doc map[string]any) {
				report := doc["required_reports"].([]any)[0].(map[string]any)
				report["claim_refs"] = []any{"claim:missing"}
			},
			want: []string{"claim_refs", "claim:missing"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decode(contractJSON(t, tc.mutate))
			if err == nil {
				t.Fatalf("Decode accepted invalid contract")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("Decode error = %v, want substring %q", err, want)
				}
			}
		})
	}
}

func contractJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	doc := map[string]any{
		"schema":                  SchemaV1,
		"id":                      "surface-release-crash-reporting",
		"title":                   "Surface crash reporting release gate",
		"scope":                   "surface/crash_reporting",
		"producer":                "scripts/release/surface/release-gate.sh",
		"entrypoint":              "scripts/release/surface/release-gate.sh",
		"fresh_report_dir_policy": "require-empty-or-new",
		"host_preconditions":      []any{"linux", "git-worktree"},
		"steps": []any{
			map[string]any{
				"id":                    "run-crash-report-smoke",
				"kind":                  "go-test",
				"command":               "go test ./tools/validators/surface -run TestValidateCrashReport -count=1",
				"working_dir":           ".",
				"required":              true,
				"report_outputs":        []any{"reports/surface/crash-report.json"},
				"validator_refs":        []any{"surface-crash-report"},
				"host_preconditions":    []any{"linux"},
				"blocked_status_policy": "block-release",
			},
		},
		"required_reports": []any{
			map[string]any{
				"path":                   "reports/surface/crash-report.json",
				"schema":                 "tetra.surface.crash-report.v1",
				"validator":              "surface-crash-report",
				"same_commit_required":   true,
				"artifact_hash_required": true,
				"claim_refs":             []any{"claim:surface-crash-reporting"},
			},
		},
		"validators": []any{
			map[string]any{
				"id":      "surface-crash-report",
				"kind":    "go-command",
				"command": "go run ./tools/cmd/validate-surface-crash-report --report reports/surface/crash-report.json",
			},
		},
		"artifact_hashes": map[string]any{
			"enabled":   true,
			"required":  true,
			"algorithm": "sha256",
		},
		"claims": []any{
			map[string]any{
				"id":        "claim:surface-crash-reporting",
				"statement": "Surface crash reporting evidence is present and validator-checked.",
			},
		},
		"nonclaims": []any{
			map[string]any{
				"id":        "nonclaim:full-surface-release",
				"statement": "This gate does not claim every Surface release requirement is complete.",
			},
		},
		"ci_artifacts": []any{
			map[string]any{
				"path":     "reports/surface/crash-report.json",
				"required": true,
			},
		},
	}
	if mutate != nil {
		mutate(doc)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal fixture: %v", err)
	}
	return raw
}

func cloneObject(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
