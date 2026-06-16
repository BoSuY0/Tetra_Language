package declarations

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestValidateCapsuleDeclsAcceptsLiteralMetadata(t *testing.T) {
	file := &frontend.FileAST{
		Capsules: []*frontend.CapsuleDecl{
			{
				Entries: []frontend.CapsuleEntryDecl{
					{Key: "permissions.io", Value: &frontend.BoolLitExpr{Value: true}},
					{Key: "limits.max1", Value: &frontend.NumberExpr{Value: 1}},
				},
			},
		},
	}

	if err := ValidateCapsuleDecls(file); err != nil {
		t.Fatalf("ValidateCapsuleDecls returned error: %v", err)
	}
}

func TestValidateCapsuleDeclsRejectsDuplicateAndInvalidMetadata(t *testing.T) {
	duplicate := &frontend.FileAST{
		Capsules: []*frontend.CapsuleDecl{
			{Entries: []frontend.CapsuleEntryDecl{
				{Key: "permissions.io", Value: &frontend.BoolLitExpr{}},
				{Key: "permissions.io", Value: &frontend.BoolLitExpr{}},
			}},
		},
	}
	if err := ValidateCapsuleDecls(duplicate); err == nil || !strings.Contains(err.Error(), "duplicate capsule metadata key") {
		t.Fatalf("duplicate error = %v", err)
	}

	invalidKey := &frontend.FileAST{
		Capsules: []*frontend.CapsuleDecl{
			{Entries: []frontend.CapsuleEntryDecl{{Key: "Permissions.io", Value: &frontend.BoolLitExpr{}}}},
		},
	}
	if err := ValidateCapsuleDecls(invalidKey); err == nil || !strings.Contains(err.Error(), "invalid capsule metadata key") {
		t.Fatalf("invalid key error = %v", err)
	}

	nonLiteral := &frontend.FileAST{
		Capsules: []*frontend.CapsuleDecl{
			{Entries: []frontend.CapsuleEntryDecl{{Key: "permissions.io", Value: &frontend.IdentExpr{Name: "x"}}}},
		},
	}
	if err := ValidateCapsuleDecls(nonLiteral); err == nil || !strings.Contains(err.Error(), "must be a literal") {
		t.Fatalf("non-literal error = %v", err)
	}
}
