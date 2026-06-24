package resources

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func mustPath(t *testing.T, raw string) Path {
	t.Helper()
	path, err := ParsePath(raw)
	if err != nil {
		t.Fatalf("ParsePath(%q): %v", raw, err)
	}
	return path
}

func TestTypedPathWireCompatibility(t *testing.T) {
	root, err := Root("root")
	if err != nil {
		t.Fatalf("Root: %v", err)
	}
	if got := root.String(); got != "root" {
		t.Fatalf("Root string = %q, want root", got)
	}
	if got := root.Field("field").String(); got != "root.field" {
		t.Fatalf("Field string = %q, want root.field", got)
	}
	if got := root.EnumPayload(3, 2).String(); got != "root.$case3.payload2" {
		t.Fatalf("EnumPayload string = %q, want root.$case3.payload2", got)
	}
	if got := root.Element().String(); got != "root.$elem" {
		t.Fatalf("Element string = %q, want root.$elem", got)
	}
}

func TestTypedPathParentAndRelativeTo(t *testing.T) {
	root := mustPath(t, "root")
	child := mustPath(t, "root.field.leaf")
	parent, ok := child.Parent()
	if !ok || parent.String() != "root.field" {
		t.Fatalf("Parent = (%q, %v), want (root.field, true)", parent, ok)
	}
	enumPayload := mustPath(t, "root.$case3.payload2")
	enumParent, ok := enumPayload.Parent()
	if !ok || enumParent.String() != "root" {
		t.Fatalf("enum Parent = (%q, %v), want (root, true)", enumParent, ok)
	}
	relative, ok := child.RelativeTo(root)
	if !ok || relative.String() != "field.leaf" {
		t.Fatalf("RelativeTo root = (%q, %v), want (field.leaf, true)", relative, ok)
	}
	joined := JoinPath(root, relative)
	if joined != child {
		t.Fatalf("JoinPath(root, relative) = %q, want %q", joined, child)
	}
}

func TestTypedPathAliasLaws(t *testing.T) {
	tests := []struct {
		name       string
		left       Path
		right      Path
		wantAlias  bool
		wantParent bool
	}{
		{
			name:      "same path aliases",
			left:      mustPath(t, "root.field"),
			right:     mustPath(t, "root.field"),
			wantAlias: true,
		},
		{
			name:       "parent aliases child",
			left:       mustPath(t, "root"),
			right:      mustPath(t, "root.field"),
			wantAlias:  true,
			wantParent: true,
		},
		{
			name:      "child aliases parent",
			left:      mustPath(t, "root.field.leaf"),
			right:     mustPath(t, "root.field"),
			wantAlias: true,
		},
		{
			name:      "sibling fields do not alias",
			left:      mustPath(t, "root.left"),
			right:     mustPath(t, "root.right"),
			wantAlias: false,
		},
		{
			name:      "sibling enum payload slots do not alias",
			left:      mustPath(t, "root.$case1.payload0"),
			right:     mustPath(t, "root.$case1.payload1"),
			wantAlias: false,
		},
		{
			name:      "different roots do not alias",
			left:      mustPath(t, "left.field"),
			right:     mustPath(t, "right.field"),
			wantAlias: false,
		},
		{
			name:      "prefix collision does not alias",
			left:      mustPath(t, "a.b"),
			right:     mustPath(t, "a.bb"),
			wantAlias: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.left.Aliases(tt.right); got != tt.wantAlias {
				t.Fatalf("%q Aliases %q = %v, want %v", tt.left, tt.right, got, tt.wantAlias)
			}
			if got := tt.right.Aliases(tt.left); got != tt.wantAlias {
				t.Fatalf("%q Aliases %q = %v, want symmetry %v", tt.right, tt.left, got, tt.wantAlias)
			}
			if !tt.left.Aliases(tt.left) {
				t.Fatalf("%q does not alias itself", tt.left)
			}
			if tt.wantParent && !tt.left.IsAncestorOf(tt.right) {
				t.Fatalf("%q should be ancestor of %q", tt.left, tt.right)
			}
		})
	}

	root := mustPath(t, "root")
	child := mustPath(t, "root.field")
	grandchild := mustPath(t, "root.field.leaf")
	if !root.IsAncestorOf(child) || !child.IsAncestorOf(grandchild) ||
		!root.IsAncestorOf(grandchild) {
		t.Fatalf("ancestor transitivity failed for %q, %q, %q", root, child, grandchild)
	}
	if !grandchild.IsDescendantOf(root) {
		t.Fatalf("%q should be descendant of %q", grandchild, root)
	}
}

func TestTypedPathParseStringRoundTripAndMalformedSyntheticSegments(t *testing.T) {
	for _, raw := range []string{
		"root",
		"root.field",
		"root.$case3.payload2",
		"root.$elem",
		"root.field.$case0.payload1.$elem",
		"root[index.with.dot].field",
	} {
		path := mustPath(t, raw)
		if got := path.String(); got != raw {
			t.Fatalf("roundtrip %q -> %q", raw, got)
		}
	}
	for _, raw := range []string{
		"",
		".root",
		"root.",
		"root..field",
		"root.$case",
		"root.$case3",
		"root.$case3.field",
		"root.$case3.payload",
		"root.$elem0",
		"root.$unknown",
	} {
		if _, err := ParsePath(raw); err == nil {
			t.Fatalf("ParsePath(%q) succeeded, want malformed path rejection", raw)
		}
	}
	if _, err := Root(""); err == nil {
		t.Fatalf("Root(\"\") succeeded, want empty root rejection")
	}
}

func TestProductionOwnershipPathOperationsUseTypedAPI(t *testing.T) {
	root := repoRoot(t)
	targets := map[string][]string{
		"compiler/internal/semantics/resources/paths.go": {
			`fmt.Sprintf("$case%d.payload%d"`,
		},
		"compiler/internal/allocplan/plan.go": {
			`strings.HasPrefix(input, allocName+".")`,
		},
		"compiler/internal/plir/plir_proofs.go": {
			`strings.HasPrefix(proofPath, mutatedPath+".")`,
		},
		"compiler/internal/memoryfacts/fromplir/from_plir_borrow.go": {
			`strings.HasPrefix(ownerPath, owner+".")`,
			`strings.TrimPrefix(ownerPath, owner+".")`,
		},
		"compiler/internal/memoryfacts/fromplir/from_plir_summary.go": {
			`strings.HasPrefix(ownerPath, owner+".")`,
			`strings.TrimPrefix(ownerPath, owner+".")`,
		},
		"compiler/internal/semantics/semantics_memory_resources.go": {
			`strings.HasPrefix(name, prefixDot)`,
		},
		"compiler/internal/semantics/semantics_checker.go": {
			`strings.HasPrefix(path, prefix)`,
			`strings.TrimPrefix(path, prefix)`,
			`resourceFieldPath(prefix, "$elem")`,
		},
	}
	for rel, forbidden := range targets {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", rel, err)
		}
		text := string(raw)
		for _, pattern := range forbidden {
			if strings.Contains(text, pattern) {
				t.Fatalf("%s still contains ad-hoc ownership path operation %q", rel, pattern)
			}
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}
