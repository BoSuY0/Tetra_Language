package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateProvenanceReportMatchesCorpus(t *testing.T) {
	corpusDir := writeSampleCorpus(t)
	raw := sampleReportJSON(t, corpusDir, "3")

	if err := validateProvenanceReport(raw, corpusDir); err != nil {
		t.Fatalf("validateProvenanceReport() error = %v", err)
	}
}

func TestValidateProvenanceReportRejectsCountDrift(t *testing.T) {
	corpusDir := writeSampleCorpus(t)
	raw := sampleReportJSON(t, corpusDir, "4")

	err := validateProvenanceReport(raw, corpusDir)
	if err == nil || !strings.Contains(err.Error(), "totals.values") {
		t.Fatalf("validateProvenanceReport() error = %v, want totals.values drift", err)
	}
}

func TestValidateProvenanceReportRequiresUnknownReasonBuckets(t *testing.T) {
	corpusDir := writeSampleCorpus(t)
	raw := strings.Replace(string(sampleReportJSON(t, corpusDir, "3")),
		`"unknown_reason_buckets": [
    {
      "reason": "alias source has external or unknown provenance",
      "value_count": 1,
      "fact_count": 1,
      "artifacts": ["sample.proof.json"],
      "note": "sample unknown alias path"
    },
    {
      "reason": "raw slice gateway has external provenance unless an unsafe proof supplies more facts",
      "value_count": 1,
      "fact_count": 1,
      "artifacts": ["sample.proof.json"],
      "note": "sample external raw slice path"
    }
  ],`,
		`"unknown_reason_buckets": [],`, 1)

	err := validateProvenanceReport([]byte(raw), corpusDir)
	if err == nil || !strings.Contains(err.Error(), "unknown_reason_buckets") {
		t.Fatalf("validateProvenanceReport() error = %v, want missing reason bucket", err)
	}
}

func writeSampleCorpus(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	proof := `{
  "schema_version": 1,
  "kind": "proof",
  "target": "linux-x64",
  "bounds": {"totals": {"removed": 0, "left": 1}},
  "proofs": [{"proof_id": "proof:sample"}],
  "plir": {
    "funcs": [
      {
        "name": "sample.main",
        "values": [
          {"id": "param:src", "provenance": {"kind": "param", "root": "param:src"}},
          {"id": "view:raw", "provenance": {"kind": "external", "root": "raw_parts"}},
          {"id": "local:lost", "provenance": {"kind": "unknown"}}
        ],
        "facts": [
          {
            "id": "f0",
            "kind": "provenance_unknown",
            "value_id": "view:raw",
            "reason": "raw slice gateway has external provenance unless an unsafe proof supplies more facts"
          },
          {
            "id": "f1",
            "kind": "provenance_unknown",
            "value_id": "local:lost",
            "reason": "alias source has external or unknown provenance"
          },
          {
            "id": "f2",
            "kind": "provenance_known",
            "value_id": "param:src",
            "reason": "checked parameter/local memory value"
          }
        ],
        "proof_uses": [{"id": "u0"}]
      }
    ]
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "sample.proof.json"), []byte(proof), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "sample.bounds.json"),
		[]byte(`{"totals":{"removed":0,"left":1}}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	return dir
}

func sampleReportJSON(t *testing.T, corpusDir string, values string) []byte {
	t.Helper()
	report := `{
  "schema": "tetra.proof-coverage.provenance.v1",
  "generated_at": "2026-06-02",
  "corpus": {
    "path_glob": "` + filepath.ToSlash(filepath.Join(corpusDir, "*.proof.json")) + `",
    "proof_files": 1,
    "bounds_files": 1,
    "evidence_commands": ["go test fixture"]
  },
  "totals": {
    "artifacts": 1,
    "functions": 1,
    "values": ` + values + `,
    "facts": 3,
    "proof_uses": 1,
    "provenance_counts": {
      "allocation": 0,
      "actor_transfer": 0,
      "external": 1,
      "island": 0,
      "literal": 0,
      "missing": 0,
      "param": 1,
      "stack": 0,
      "unknown": 1
    },
    "provenance_unknown_facts": 2
  },
  "artifacts": [
    {
      "name": "sample",
      "source": "examples/sample.tetra",
      "proof": "sample.proof.json",
      "bounds": "sample.bounds.json",
      "functions": 1,
      "values": 3,
      "facts": 3,
      "proof_uses": 1,
      "provenance_counts": {
        "allocation": 0,
        "actor_transfer": 0,
        "external": 1,
        "island": 0,
        "literal": 0,
        "missing": 0,
        "param": 1,
        "stack": 0,
        "unknown": 1
      },
      "provenance_unknown_facts": 2
    }
  ],
  "unknown_reason_buckets": [
    {
      "reason": "alias source has external or unknown provenance",
      "value_count": 1,
      "fact_count": 1,
      "artifacts": ["sample.proof.json"],
      "note": "sample unknown alias path"
    },
    {
      "reason": "raw slice gateway has external provenance unless an unsafe proof supplies more facts",
      "value_count": 1,
      "fact_count": 1,
      "artifacts": ["sample.proof.json"],
      "note": "sample external raw slice path"
    }
  ],
  "external_reason_buckets": [
    {
      "reason": "raw slice gateway has external provenance unless an unsafe proof supplies more facts",
      "value_count": 1,
      "fact_count": 1,
      "artifacts": ["sample.proof.json"],
      "note": "sample external raw slice path"
    }
  ],
  "false_known_risk_points": [
    {
      "id": "sample-risk",
      "status": "covered_by_sample",
      "summary": "sample risk",
      "corpus_evidence": "sample corpus",
      "code_evidence": [
        {
          "path": "tools/cmd/validate-provenance-report/main_test.go",
          "line": 1,
          "symbol": "TestValidateProvenanceReportMatchesCorpus",
          "evidence": "unit test fixture"
        }
      ],
      "artifact_evidence": ["sample.proof.json"],
      "mitigation": "keep validator active"
    }
  ],
  "non_claims": ["sample report is not production corpus"]
}`
	return []byte(report)
}
