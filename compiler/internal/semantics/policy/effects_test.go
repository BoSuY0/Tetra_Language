package policy

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestNormalizeEffectsExpandsAliasesAndGroups(t *testing.T) {
	got, err := NormalizeEffects([]string{"effects.cap.io", "cap.mem", "privacy"}, frontend.Position{}, nil)
	if err != nil {
		t.Fatalf("NormalizeEffects returned error: %v", err)
	}

	want := []string{"capability", "io", "mem", "mmio", "privacy"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeEffects() = %#v, want %#v", got, want)
	}
}

func TestNormalizeEffectDeclTracksExplicitEffectsAndCapabilityGroups(t *testing.T) {
	got, err := NormalizeEffectDecl([]string{"effects.cap.io", "cap.mem", "capsule.io"}, frontend.Position{}, nil)
	if err != nil {
		t.Fatalf("NormalizeEffectDecl returned error: %v", err)
	}

	for _, effect := range []string{"capability", "io", "mem", "mmio", "capsule.io"} {
		if _, ok := got.Declared[effect]; !ok {
			t.Fatalf("Declared missing %q in %#v", effect, got.Declared)
		}
	}
	for _, effect := range []string{"mem", "capsule.io"} {
		if _, ok := got.Explicit[effect]; !ok {
			t.Fatalf("Explicit missing %q in %#v", effect, got.Explicit)
		}
	}
	for _, effect := range []string{"capability", "io", "mmio"} {
		if _, ok := got.Explicit[effect]; ok {
			t.Fatalf("Explicit unexpectedly contains grouped member %q in %#v", effect, got.Explicit)
		}
	}
	if !got.HasCapGroup {
		t.Fatalf("HasCapGroup = false, want true")
	}
}

func TestNormalizeEffectsUsesDiagnosticCallbackForUnknownEffects(t *testing.T) {
	sentinel := errors.New("diagnostic callback was used")
	called := false
	_, err := NormalizeEffects([]string{"unknown.effect"}, frontend.Position{}, func(pos frontend.Position, format string, args ...interface{}) error {
		called = true
		if !strings.Contains(format, "unknown effect") {
			t.Fatalf("diagnostic format = %q, want unknown effect", format)
		}
		if len(args) != 1 || args[0] != "unknown.effect" {
			t.Fatalf("diagnostic args = %#v, want unknown.effect", args)
		}
		return sentinel
	})

	if !called {
		t.Fatalf("diagnostic callback was not called")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("NormalizeEffects error = %v, want sentinel", err)
	}
}
