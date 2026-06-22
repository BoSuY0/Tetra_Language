package memorycorev2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testHead = "0123456789abcdef0123456789abcdef01234567"

func TestMemoryCoreV2ValidateReportAcceptsPositiveFixture(t *testing.T) {
	raw := readFixture(t, "positive.json")
	if err := ValidateReport(raw, Options{CurrentGitHead: testHead}); err != nil {
		t.Fatalf("ValidateReport failed: %v", err)
	}
}

func TestMemoryCoreV2ValidateReportRejectsNegativeFixture(t *testing.T) {
	raw := readFixture(t, "negative_broad_claim.json")
	err := ValidateReport(raw, Options{CurrentGitHead: testHead})
	if err == nil {
		t.Fatalf("expected negative fixture to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "forbidden") {
		t.Fatalf("error = %v, want forbidden broad claim", err)
	}
}

func TestMemoryCoreV2ValidateReportRejectsRequiredGuards(t *testing.T) {
	tests := []struct {
		name string
		edit func(string) string
		want string
	}{
		{
			name: "missing memory graph digest",
			edit: func(raw string) string {
				return strings.Replace(raw, `"memory_graph_digest": "memory-graph:sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"`, `"memory_graph_digest": ""`, 1)
			},
			want: "memory_graph_digest",
		},
		{
			name: "report only state",
			edit: func(raw string) string {
				return strings.Replace(raw, `"normal_build_state_built": true`, `"normal_build_state_built": false`, 1)
			},
			want: "normal_build_state_built",
		},
		{
			name: "route count mismatch",
			edit: func(raw string) string {
				return strings.Replace(raw, `"island_routes_direct": 16`, `"island_routes_direct": 15`, 1)
			},
			want: "route",
		},
		{
			name: "proofless optimizer rewrite",
			edit: func(raw string) string {
				return strings.Replace(raw, `"optimizer_rewrites_with_proof_ids": 4`, `"optimizer_rewrites_with_proof_ids": 3`, 1)
			},
			want: "optimizer",
		},
		{
			name: "unsupported backend marked supported",
			edit: func(raw string) string {
				return strings.Replace(raw, `"target": "wasm32-wasi",
      "operation": "reserve",
      "supported": false,`, `"target": "wasm32-wasi",
      "operation": "reserve",
      "supported": true,`, 1)
			},
			want: "unsupported",
		},
		{
			name: "memorymodel parity incomplete",
			edit: func(raw string) string {
				return strings.Replace(raw, `"memorymodel_outcomes_real_pipeline": 50`, `"memorymodel_outcomes_real_pipeline": 49`, 1)
			},
			want: "memorymodel",
		},
		{
			name: "failed requirement signed off",
			edit: func(raw string) string {
				return strings.Replace(raw, `"status": "pass",
      "evidence": "negative fixture proves the validator rejects report-only state"`, `"status": "fail",
      "evidence": "negative fixture proves the validator rejects report-only state"`, 1)
			},
			want: "final_signoff",
		},
	}
	raw := string(readFixture(t, "positive.json"))
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateReport([]byte(tc.edit(raw)), Options{CurrentGitHead: testHead})
			if err == nil {
				t.Fatalf("expected %s to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestMemoryCoreV2ClaimScanner(t *testing.T) {
	if err := ValidateClaimFile(filepath.Join("testdata", "claims_positive.md")); err != nil {
		t.Fatalf("ValidateClaimFile positive failed: %v", err)
	}
	err := ValidateClaimFile(filepath.Join("testdata", "claims_negative.md"))
	if err == nil {
		t.Fatalf("expected negative claims fixture to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "forbidden") {
		t.Fatalf("error = %v, want forbidden claim", err)
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
