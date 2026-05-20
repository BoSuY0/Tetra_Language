package compiler_test

import (
	"testing"

	compiler "tetra_language/compiler"
)

func TestT4FormatRegistryDeclaresOfficialFamily(t *testing.T) {
	formats := compiler.T4Formats()
	byName := map[string]compiler.FormatInfo{}
	for _, format := range formats {
		if format.Name == "" {
			t.Fatalf("format with empty name: %#v", format)
		}
		byName[format.Name] = format
	}

	tests := []struct {
		name     string
		ext      string
		role     string
		primary  bool
		legacy   bool
		fileName string
	}{
		{name: "T4 Source Format", ext: ".t4", role: "source", primary: true},
		{name: "Legacy Tetra Source Format", ext: ".tetra", role: "source", legacy: true},
		{name: "Todex Fragment", ext: ".tdx", role: "todex-fragment"},
		{name: "T4 Seed", ext: ".t4s", role: "offline-seed"},
		{name: "T4 Interface", ext: ".t4i", role: "interface"},
		{name: "T4 Proof", ext: ".t4p", role: "proof"},
		{name: "T4 Replay", ext: ".t4r", role: "replay"},
		{name: "T4 Quest", ext: ".t4q", role: "quest"},
		{name: "Tetra NeedMap", ext: ".tneed", role: "needmap"},
		{name: "Tetra Semantic Lock", role: "semantic-lock", fileName: "Tetra.lock"},
	}
	for _, tt := range tests {
		format, ok := byName[tt.name]
		if !ok {
			t.Fatalf("missing format %q in %#v", tt.name, formats)
		}
		if format.Extension != tt.ext || format.Role != tt.role || format.Primary != tt.primary || format.Legacy != tt.legacy || format.FileName != tt.fileName {
			t.Fatalf("%s = %#v", tt.name, format)
		}
	}
}

func TestSourceFileHelpersPreferT4AndKeepLegacyTetra(t *testing.T) {
	if got := compiler.SourceExtensions(); len(got) != 2 || got[0] != ".t4" || got[1] != ".tetra" {
		t.Fatalf("SourceExtensions() = %#v, want [.t4 .tetra]", got)
	}
	for _, path := range []string{"main.t4", "ui/CounterView.t4", "legacy/main.tetra"} {
		if !compiler.IsSourceFile(path) {
			t.Fatalf("IsSourceFile(%q) = false, want true", path)
		}
	}
	for _, path := range []string{"fragment.tdx", "seed.t4s", "Tetra.lock", "notes.txt"} {
		if compiler.IsSourceFile(path) {
			t.Fatalf("IsSourceFile(%q) = true, want false", path)
		}
	}
}
