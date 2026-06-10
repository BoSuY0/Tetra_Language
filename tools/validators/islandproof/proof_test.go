package islandproof

import (
	"strings"
	"testing"
)

func TestValidateAcceptsValidProof(t *testing.T) {
	if err := Validate([]byte(validProofReport()), Options{MemoryReport: []byte(validMemoryReport())}); err != nil {
		t.Fatalf("Validate valid proof: %v", err)
	}
}

func TestValidateRejectsMissingProducerCommandMetadata(t *testing.T) {
	proof := strings.Replace(validProofReport(), `  "producer_command": "go run ./tools/cmd/validate-island-proof",`+"\n", "", 1)
	err := Validate([]byte(proof), Options{MemoryReport: []byte(validMemoryReport())})
	if err == nil || !strings.Contains(err.Error(), "producer_command") {
		t.Fatalf("Validate error = %v, want producer_command rejection", err)
	}
}

func TestValidateRejectsMalformedAndUnknownSchema(t *testing.T) {
	if err := Validate([]byte(`{"schema":"tetra.island.proof.v1"`), Options{MemoryReport: []byte(validMemoryReport())}); err == nil {
		t.Fatalf("Validate malformed JSON unexpectedly passed")
	}
	raw := strings.Replace(validProofReport(), `"schema": "tetra.island.proof.v1"`, `"schema": "tetra.island.proof.v0"`, 1)
	err := Validate([]byte(raw), Options{MemoryReport: []byte(validMemoryReport())})
	if err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("Validate schema error = %v, want schema rejection", err)
	}
}

func TestValidateRejectsRequiredProofFields(t *testing.T) {
	for _, tc := range []struct {
		name string
		old  string
		new  string
		want string
	}{
		{name: "git head", old: `"git_head": "0123456789abcdef0123456789abcdef01234567"`, new: `"git_head": ""`, want: "git_head"},
		{name: "proof id", old: `"proof_id": "proof:island:borrow:1"`, new: `"proof_id": ""`, want: "proof_id"},
		{name: "operation", old: `"operation": "island_borrow"`, new: `"operation": ""`, want: "operation"},
		{name: "subject", old: `"subject_base_id": "alloc:main:0"`, new: `"subject_base_id": ""`, want: "subject_base_id"},
		{name: "island", old: `"island_id": "island:main:0"`, new: `"island_id": ""`, want: "island_id"},
		{name: "epoch", old: `"epoch": 1`, new: `"epoch": 0`, want: "epoch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := strings.Replace(validProofReport(), tc.old, tc.new, 1)
			err := Validate([]byte(raw), Options{MemoryReport: []byte(validMemoryReport())})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateRejectsMissingProofProvenanceMetadata(t *testing.T) {
	proof := strings.Replace(validProofReport(), `      "provenance_class": "safe_known",`+"\n", "", 1)
	proof = strings.Replace(proof, `      "unsafe_class": "safe",`+"\n", "", 1)

	err := Validate([]byte(proof), Options{MemoryReport: []byte(validMemoryReport())})
	if err == nil || !strings.Contains(err.Error(), "provenance_class") || !strings.Contains(err.Error(), "unsafe_class") {
		t.Fatalf("Validate error = %v, want proof provenance metadata rejection", err)
	}
}

func TestValidateRejectsSameCommitMismatch(t *testing.T) {
	err := Validate([]byte(validProofReport()), Options{
		MemoryReport:      []byte(validMemoryReport()),
		RequireSameCommit: true,
		CurrentGitHead:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err == nil || !strings.Contains(err.Error(), "does not match current commit") {
		t.Fatalf("Validate error = %v, want same-commit rejection", err)
	}
}

func TestValidateRejectsProofWithoutMemoryFact(t *testing.T) {
	err := Validate([]byte(validProofReport()), Options{MemoryReport: []byte(`{"schema_version":"tetra.memory-report.v1","rows":[]}`)})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Validate error = %v, want missing memory fact rejection", err)
	}
}

func TestValidateRejectsMismatchedIslandBaseAndEpoch(t *testing.T) {
	for _, tc := range []struct {
		name string
		old  string
		new  string
		want string
	}{
		{name: "island", old: `"island_id": "island:main:0"`, new: `"island_id": "island:other"`, want: "island_id"},
		{name: "epoch", old: `"epoch": 1`, new: `"epoch": 2`, want: "epoch"},
		{name: "base", old: `"proof_subject_base_id": "alloc:main:0"`, new: `"proof_subject_base_id": "alloc:other"`, want: "subject_base_id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			memory := strings.Replace(validMemoryReport(), tc.old, tc.new, 1)
			if tc.name == "base" {
				memory = strings.Replace(memory, `"base_id": "alloc:main:0"`, `"base_id": "alloc:other"`, 1)
			}
			err := Validate([]byte(validProofReport()), Options{MemoryReport: []byte(memory)})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidateRejectsMemoryReportIdentityFieldDrift(t *testing.T) {
	for _, tc := range []struct {
		name string
		old  string
		new  string
		want string
	}{
		{name: "base id", old: `"base_id": "alloc:main:0"`, new: `"base_id": "alloc:other"`, want: "base_id"},
		{name: "proof subject base id", old: `"proof_subject_base_id": "alloc:main:0"`, new: `"proof_subject_base_id": "alloc:other"`, want: "proof_subject_base_id"},
		{name: "provenance class", old: `"provenance_class": "safe_known"`, new: `"provenance_class": "safe_owned"`, want: "provenance_class"},
		{name: "unsafe class", old: `"unsafe_class": "safe"`, new: `"unsafe_class": "unsafe_checked"`, want: "unsafe_class"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			memory := strings.Replace(validMemoryReport(), tc.old, tc.new, 1)
			err := Validate([]byte(validProofReport()), Options{MemoryReport: []byte(memory)})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate error = %v, want %q drift rejection", err, tc.want)
			}
		})
	}
}

func TestValidateRejectsMemoryReportValidatorSubstitution(t *testing.T) {
	memory := strings.Replace(validMemoryReport(), `"validator_name": "validate-island-proof"`, `"validator_name": "memory_report_validator"`, 1)
	err := Validate([]byte(validProofReport()), Options{MemoryReport: []byte(memory)})
	if err == nil || !strings.Contains(err.Error(), "validate-island-proof") {
		t.Fatalf("Validate error = %v, want validator substitution rejection", err)
	}
}

func TestValidateRejectsUnsafeUnknownPromotion(t *testing.T) {
	memory := strings.Replace(validMemoryReport(), `"provenance_class": "safe_known"`, `"provenance_class": "unsafe_unknown"`, 1)
	memory = strings.Replace(memory, `"unsafe_class": "safe"`, `"unsafe_class": "unsafe_unknown"`, 1)
	proof := strings.Replace(validProofReport(), `"provenance_class": "safe_known"`, `"provenance_class": "unsafe_unknown"`, 1)
	proof = strings.Replace(proof, `"unsafe_class": "safe"`, `"unsafe_class": "unsafe_unknown"`, 1)
	err := Validate([]byte(proof), Options{MemoryReport: []byte(memory)})
	if err == nil || !strings.Contains(err.Error(), "unsafe_unknown") {
		t.Fatalf("Validate error = %v, want unsafe_unknown rejection", err)
	}
}

func TestValidateRejectsNoAliasWithoutDistinctLiveIslands(t *testing.T) {
	proof := strings.Replace(validProofReport(), `"operation": "island_borrow"`, `"operation": "island_noalias"`, 1)
	proof = strings.Replace(proof, `"claim": "island_proof_verified"`, `"claim": "no_alias"`, 1)
	proof = strings.Replace(proof, `"distinct_live_islands": ["island:main:0", "island:other:0"]`, `"distinct_live_islands": ["island:main:0"]`, 1)
	memory := strings.Replace(validMemoryReport(), `"proof_operation": "island_borrow"`, `"proof_operation": "island_noalias"`, 1)
	err := Validate([]byte(proof), Options{MemoryReport: []byte(memory)})
	if err == nil || !strings.Contains(err.Error(), "distinct live islands") {
		t.Fatalf("Validate error = %v, want noalias distinct-islands rejection", err)
	}
}

func TestValidateRejectsBoundsProofWithoutDominance(t *testing.T) {
	proof := strings.Replace(validProofReport(), `"operation": "island_borrow"`, `"operation": "bounds_check_removed"`, 1)
	proof = strings.Replace(proof, `"proof_kind": "island_epoch"`, `"proof_kind": "bounds_check"`, 1)
	proof = strings.Replace(proof, `"dominance": "entry dominates island borrow",`, `"dominance": "",`, 1)
	err := Validate([]byte(proof), Options{MemoryReport: []byte(validMemoryReport())})
	if err == nil || !strings.Contains(err.Error(), "dominance") {
		t.Fatalf("Validate error = %v, want dominance rejection", err)
	}
}

func TestValidateRejectsStorageProofWithoutLowering(t *testing.T) {
	proof := strings.Replace(validProofReport(), `"operation": "island_borrow"`, `"operation": "storage_lowering"`, 1)
	proof = strings.Replace(proof, `"planned_storage": "ExplicitIsland",`+"\n      "+`"actual_lowering_storage": "ExplicitIsland",`, "", 1)
	err := Validate([]byte(proof), Options{MemoryReport: []byte(validMemoryReport())})
	if err == nil || !strings.Contains(err.Error(), "planned_storage") {
		t.Fatalf("Validate error = %v, want storage proof rejection", err)
	}
}

func TestValidateRejectsStorageProofHeapFallback(t *testing.T) {
	proof := strings.Replace(validProofReport(), `"operation": "island_borrow"`, `"operation": "storage_lowering"`, 1)
	proof = strings.Replace(proof, `"actual_lowering_storage": "ExplicitIsland"`, `"actual_lowering_storage": "Heap"`, 1)
	memory := strings.Replace(validMemoryReport(), `"actual_lowering_storage": "ExplicitIsland"`, `"actual_lowering_storage": "Heap"`, 1)
	memory = strings.Replace(memory, `"proof_operation": "island_borrow"`, `"proof_operation": "storage_lowering"`, 1)
	err := Validate([]byte(proof), Options{MemoryReport: []byte(memory)})
	if err == nil || !strings.Contains(err.Error(), "storage") || !strings.Contains(err.Error(), "Heap") {
		t.Fatalf("Validate error = %v, want storage heap fallback rejection", err)
	}
}

func validProofReport() string {
	return `{
  "schema": "tetra.island.proof.v1",
  "producer": "compiler/memoryfacts",
  "producer_command": "go run ./tools/cmd/validate-island-proof",
  "git_head": "0123456789abcdef0123456789abcdef01234567",
  "generated_at": "2026-06-08T21:30:00Z",
  "proofs": [
    {
      "proof_id": "proof:island:borrow:1",
      "operation": "island_borrow",
      "proof_kind": "island_epoch",
      "subject_base_id": "alloc:main:0",
      "island_id": "island:main:0",
      "epoch": 1,
      "source_fact_id": "fact:island:proof:1",
      "claim": "island_proof_verified",
      "provenance_class": "safe_known",
      "unsafe_class": "safe",
      "validator_name": "validate-island-proof",
      "validator_status": "pass",
      "planned_storage": "ExplicitIsland",
      "actual_lowering_storage": "ExplicitIsland",
      "dominance": "entry dominates island borrow",
      "distinct_live_islands": ["island:main:0", "island:other:0"]
    }
  ]
}`
}

func validMemoryReport() string {
	return `{
  "schema_version": "tetra.memory-report.v1",
  "rows": [
    {
      "program_id": "program",
      "function_id": "main",
      "site_id": "island:main:1",
      "source_fact_id": "fact:island:proof:1",
      "source_stage": "validation",
      "claim": "island_proof_verified",
      "claim_level": "validated",
      "provenance_class": "safe_known",
      "unsafe_class": "safe",
      "alias_state": "unique",
      "island_id": "island:main:0",
      "epoch": 1,
      "base_id": "alloc:main:0",
      "proof_id": "proof:island:borrow:1",
      "proof_kind": "island_epoch",
      "proof_subject_base_id": "alloc:main:0",
      "proof_operation": "island_borrow",
      "planned_storage": "ExplicitIsland",
      "actual_lowering_storage": "ExplicitIsland",
      "validator_name": "validate-island-proof",
      "validator_status": "pass",
      "cost_class": "instrumentation_only",
      "reason": "fixture island proof"
    }
  ]
}`
}
