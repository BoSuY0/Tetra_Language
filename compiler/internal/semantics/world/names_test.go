package world

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestCollectImportAliasesHandlesModuleAndSelectiveImports(t *testing.T) {
	file := &frontend.FileAST{
		Imports: []frontend.ImportDecl{
			{Path: "math", Alias: "m"},
			{Path: "core", Items: []string{"print"}},
		},
	}

	aliases, err := CollectImportAliases(file)
	if err != nil {
		t.Fatalf("CollectImportAliases returned error: %v", err)
	}
	if aliases["m"] != "math" {
		t.Fatalf("module alias = %q, want math", aliases["m"])
	}
	target, ok := ImportSymbolTarget(aliases["print"])
	if !ok || target != "core.print" {
		t.Fatalf("selective alias target = (%q, %v), want core.print true", target, ok)
	}
}

func TestCollectImportAliasesRejectsDeclarationConflicts(t *testing.T) {
	file := &frontend.FileAST{
		Imports: []frontend.ImportDecl{{Path: "math", Alias: "value"}},
		Funcs:   []*frontend.FuncDecl{{Name: "value"}},
	}

	_, err := CollectImportAliases(file)
	if err == nil || !strings.Contains(err.Error(), "conflicts with declaration") {
		t.Fatalf("CollectImportAliases error = %v", err)
	}
}

func TestWorldNameHelpers(t *testing.T) {
	if got := QualifyName("core", "main"); got != "core.main" {
		t.Fatalf("QualifyName = %q, want core.main", got)
	}
	if got := QualifyName("", "main"); got != "main" {
		t.Fatalf("QualifyName empty module = %q, want main", got)
	}
	if got := CheckedFuncFullName("core", &frontend.FuncDecl{Name: "ext", ExtensionOf: "Type"}); got != "ext" {
		t.Fatalf("CheckedFuncFullName extension = %q, want ext", got)
	}
	if got := CheckedFuncFullName("core", &frontend.FuncDecl{Name: "main"}); got != "core.main" {
		t.Fatalf("CheckedFuncFullName = %q, want core.main", got)
	}
}
