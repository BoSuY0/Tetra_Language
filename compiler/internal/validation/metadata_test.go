package validation

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestBuildOptimizationValidationMetadataRecordsMachineCheckableEvidence(t *testing.T) {
	before := singleReturnIR("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
		ir.IRInstr{Kind: ir.IRAddI32},
	)
	after := singleReturnIR("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
	)

	meta, err := BuildOptimizationValidationMetadata(before, after, OptimizationMetadataOptions{
		PassName:                  "basic-scalar",
		InputKind:                 "stack_ir",
		OutputKind:                "optimized_ir",
		InputVerifier:             "lower.VerifyProgram",
		OutputVerifier:            "lower.VerifyProgram",
		ValidationStrategy:        "translation_validation",
		RequiredFacts:             []string{"ir_verified"},
		PreservedFacts:            []string{"bounds_proofs"},
		InvalidatedFacts:          []string{"liveness"},
		ProofRule:                 "preserve_bounds_proofs_invalidate_liveness",
		TranslationValidationHook: "validation.ValidateTranslation",
		ReportRows:                []string{"input_verifier", "output_verifier", "proof_rule", "translation_validation_hook", "translation_report", "validation_metadata", "before_dump", "after_dump", "profile_input_policy"},
		NegativeTestMarker:        "compiler/internal/opt/manager_test.go::TestManagerRejectsIncompletePassContractEvidence",
		ProfileInputPolicy:        "unused",
		ProfileInputDigest:        "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		ProfileInputSchemaVersion: "tetra.optimizer.profile.v1",
	})
	if err != nil {
		t.Fatalf("BuildOptimizationValidationMetadata: %v", err)
	}
	if meta.SchemaVersion != "tetra.translation.validation.metadata.v1" {
		t.Fatalf("schema = %q", meta.SchemaVersion)
	}
	if meta.PassName != "basic-scalar" || meta.InputKind != "stack_ir" || meta.OutputKind != "optimized_ir" {
		t.Fatalf("metadata identity = %+v", meta)
	}
	if meta.InputVerifier != "lower.VerifyProgram" || meta.OutputVerifier != "lower.VerifyProgram" || meta.ProofRule == "" || meta.TranslationValidationHook != "validation.ValidateTranslation" {
		t.Fatalf("contract metadata = %+v", meta)
	}
	if meta.ProfileInputPolicy != "unused" || meta.ProfileInputDigest != "sha256:1111111111111111111111111111111111111111111111111111111111111111" || meta.ProfileInputSchemaVersion != "tetra.optimizer.profile.v1" {
		t.Fatalf("profile metadata = %+v", meta)
	}
	if !strings.HasPrefix(meta.BeforeHash, "sha256:") || !strings.HasPrefix(meta.AfterHash, "sha256:") {
		t.Fatalf("hashes = before %q after %q", meta.BeforeHash, meta.AfterHash)
	}
	if len(meta.Functions) != 1 || meta.Functions[0] != "main" {
		t.Fatalf("functions = %v", meta.Functions)
	}
	if meta.Translation.FunctionsCompared != 1 || meta.Translation.SemanticLocalChecks == 0 || meta.Translation.DifferentialSamples == 0 {
		t.Fatalf("translation evidence = %+v", meta.Translation)
	}
	if err := ValidateOptimizationValidationMetadata(meta); err != nil {
		t.Fatalf("ValidateOptimizationValidationMetadata: %v", err)
	}
}

func TestBuildOptimizationValidationMetadataRejectsSemanticMismatch(t *testing.T) {
	before := singleReturnIR("main", 0,
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 1},
	)
	after := singleReturnIR("main", 0,
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 2},
	)
	_, err := BuildOptimizationValidationMetadata(before, after, OptimizationMetadataOptions{
		PassName:                  "bad-fold",
		InputKind:                 "stack_ir",
		OutputKind:                "optimized_ir",
		InputVerifier:             "lower.VerifyProgram",
		OutputVerifier:            "lower.VerifyProgram",
		ValidationStrategy:        "translation_validation",
		ProofRule:                 "preserve_bounds_proofs_invalidate_liveness",
		TranslationValidationHook: "validation.ValidateTranslation",
		ReportRows:                []string{"input_verifier"},
		NegativeTestMarker:        "compiler/internal/opt/manager_test.go::TestManagerRejectsIncompletePassContractEvidence",
		ProfileInputPolicy:        "unused",
	})
	if err == nil || !strings.Contains(err.Error(), "semantic local equivalence") {
		t.Fatalf("BuildOptimizationValidationMetadata error = %v, want semantic mismatch", err)
	}
}
