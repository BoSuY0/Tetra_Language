package memoryfacts

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestMCV2CanonicalVocabularyTypesAndJSON(t *testing.T) {
	var claim Claim = ClaimBorrowedImm
	var proofKind ProofKind = ProofBounds

	if StageOptimization != SourceStage("optimization") {
		t.Fatalf("StageOptimization = %q, want optimization", StageOptimization)
	}
	if !knownSourceStage(StageOptimization) {
		t.Fatalf("StageOptimization is not registered as a known source stage")
	}

	fact := Fact{
		ID:            "fact:vocabulary",
		FunctionID:    "main",
		SiteID:        "site",
		SourceStage:   StageOptimization,
		Claim:         claim,
		ProofKind:     proofKind,
		LifetimeBirth: "semantics:birth",
		LifetimeDeath: "lowering:death",
		LifetimeOwner: "owner:main",
		DecisionCode:  "optimizer:noalias:kept",
	}
	raw, err := json.Marshal(fact)
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(raw)
	for _, want := range []string{
		`"claim":"borrowed_imm"`,
		`"proof_kind":"bounds"`,
		`"lifetime_birth":"semantics:birth"`,
		`"lifetime_death":"lowering:death"`,
		`"lifetime_owner":"owner:main"`,
		`"decision_code":"optimizer:noalias:kept"`,
	} {
		if !strings.Contains(encoded, want) {
			t.Fatalf("encoded fact %s missing %s", encoded, want)
		}
	}
}

func TestMCV2CanonicalVocabularyUniqueness(t *testing.T) {
	for name, values := range map[string][]string{
		"source stages":       SourceStages(),
		"provenance classes":  ProvenanceClasses(),
		"unsafe classes":      UnsafeClasses(),
		"alias states":        AliasStates(),
		"storage classes":     StorageClasses(),
		"claim levels":        ClaimLevels(),
		"validator statuses":  ValidatorStatuses(),
		"cost classes":        CostClasses(),
		"report claims":       ReportClaims(),
		"parent claims":       ParentRequiredClaims(),
		"island proof claims": IslandKernelEvidenceClaims(),
	} {
		assertUniqueVocabulary(t, name, values)
	}
}

func TestMCV2MemoryFactsCoreImportsStdlibOnly(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatalf("unquote import in %s: %v", name, err)
			}
			firstSegment := strings.Split(path, "/")[0]
			if strings.HasPrefix(path, "tetra_language/") || strings.Contains(firstSegment, ".") {
				t.Fatalf("memoryfacts core must import stdlib only: %s imports %q", name, path)
			}
		}
	}
}

func assertUniqueVocabulary(t *testing.T, name string, values []string) {
	t.Helper()
	seen := map[string]struct{}{}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			t.Fatalf("%s contains duplicate value %q", name, value)
		}
		seen[value] = struct{}{}
	}
}
