package compiler_test

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestObjectRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.tobj")

	srcHash := sha256.Sum256([]byte("module source"))
	worldHash := sha256.Sum256([]byte("world sig"))
	obj := &compiler.Object{
		Target:          "linux-x64",
		Module:          "engine.render",
		CompilerVersion: compiler.Version(),
		PublicAPIHash:   "sha256:abcd",
		SrcHash:         srcHash,
		WorldSigHash:    worldHash,
		Code:            []byte{0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0xC3},
		Data:            []byte("hello"),
		Symbols: []compiler.Symbol{
			{Name: "engine.render.add_one", Offset: 0},
		},
		Relocs: []compiler.Reloc{
			{Kind: compiler.RelocCallRel32, At: 1, Name: "app.game.main", Addend: 0},
			{Kind: compiler.RelocIATDisp32, At: 2, Name: "kernel32.ExitProcess", Addend: 0},
			{Kind: compiler.RelocDataDisp32, At: 3, Name: "", Addend: 4},
		},
	}

	if err := compiler.WriteObject(path, obj); err != nil {
		t.Fatalf("write object: %v", err)
	}
	readObj, err := compiler.ReadObject(path)
	if err != nil {
		t.Fatalf("read object: %v", err)
	}

	if readObj.Target != obj.Target || readObj.Module != obj.Module {
		t.Fatalf("header mismatch")
	}
	if readObj.CompilerVersion != obj.CompilerVersion || readObj.PublicAPIHash != obj.PublicAPIHash {
		t.Fatalf("metadata mismatch")
	}
	if readObj.SrcHash != obj.SrcHash || readObj.WorldSigHash != obj.WorldSigHash {
		t.Fatalf("hash mismatch")
	}
	if !bytes.Equal(readObj.Code, obj.Code) || !bytes.Equal(readObj.Data, obj.Data) {
		t.Fatalf("payload mismatch")
	}
	if !reflect.DeepEqual(readObj.Symbols, obj.Symbols) || !reflect.DeepEqual(readObj.Relocs, obj.Relocs) {
		t.Fatalf("tables mismatch")
	}
}

func TestObjectWriteRejectsTooLongHeaderString(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.tobj")
	longTarget := strings.Repeat("x", 0x10000)
	err := compiler.WriteObject(path, &compiler.Object{
		Target: longTarget,
		Module: "mod",
		Code:   []byte{0xC3},
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "string too long") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestObjectWriteRejectsMalformedSymbolAndRelocNames(t *testing.T) {
	cases := []struct {
		name string
		obj  *compiler.Object
		want string
	}{
		{
			name: "empty_symbol_name",
			obj: &compiler.Object{
				Target:  "linux-x64",
				Module:  "mod",
				Code:    []byte{0xC3},
				Symbols: []compiler.Symbol{{Name: "", Offset: 0}},
			},
			want: "empty symbol name",
		},
		{
			name: "empty_call_relocation_name",
			obj: &compiler.Object{
				Target:  "linux-x64",
				Module:  "mod",
				Code:    []byte{0xE8, 0, 0, 0, 0, 0xC3},
				Symbols: []compiler.Symbol{{Name: "main", Offset: 0}},
				Relocs:  []compiler.Reloc{{Kind: compiler.RelocCallRel32, At: 1, Name: ""}},
			},
			want: "call relocation with empty symbol name",
		},
		{
			name: "empty_iat_relocation_name",
			obj: &compiler.Object{
				Target:  "windows-x64",
				Module:  "mod",
				Code:    []byte{0xFF, 0x15, 0, 0, 0, 0, 0xC3},
				Symbols: []compiler.Symbol{{Name: "main", Offset: 0}},
				Relocs:  []compiler.Reloc{{Kind: compiler.RelocIATDisp32, At: 2, Name: ""}},
			},
			want: "IAT relocation with empty symbol name",
		},
		{
			name: "unsupported_relocation_kind",
			obj: &compiler.Object{
				Target:  "linux-x64",
				Module:  "mod",
				Code:    []byte{0xC3},
				Symbols: []compiler.Symbol{{Name: "main", Offset: 0}},
				Relocs:  []compiler.Reloc{{Kind: compiler.RelocKind(255), At: 0, Name: "bad"}},
			},
			want: "unsupported relocation kind 255",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad.tobj")
			err := compiler.WriteObject(path, tc.obj)
			if err == nil {
				t.Fatalf("expected malformed object error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
			if data, readErr := os.ReadFile(path); readErr == nil && len(data) > 0 {
				t.Fatalf("partial object was written: % x", data)
			}
		})
	}
}

func TestObjectWriteRejectsMalformedSymbolAndRelocRanges(t *testing.T) {
	cases := []struct {
		name string
		obj  *compiler.Object
		want string
	}{
		{
			name: "symbol_offset_out_of_range",
			obj: &compiler.Object{
				Target:  "linux-x64",
				Module:  "mod",
				Code:    []byte{0xC3},
				Symbols: []compiler.Symbol{{Name: "main", Offset: 1}},
			},
			want: "symbol offset out of range",
		},
		{
			name: "relocation_offset_out_of_range",
			obj: &compiler.Object{
				Target:  "linux-x64",
				Module:  "mod",
				Code:    []byte{0xE8, 0, 0, 0},
				Symbols: []compiler.Symbol{{Name: "main", Offset: 0}},
				Relocs:  []compiler.Reloc{{Kind: compiler.RelocCallRel32, At: 1, Name: "main"}},
			},
			want: "relocation offset out of range",
		},
		{
			name: "data_relocation_addend_out_of_range",
			obj: &compiler.Object{
				Target:  "linux-x64",
				Module:  "mod",
				Code:    []byte{0, 0, 0, 0},
				Data:    []byte("A"),
				Symbols: []compiler.Symbol{{Name: "main", Offset: 0}},
				Relocs:  []compiler.Reloc{{Kind: compiler.RelocDataDisp32, At: 0, Addend: 1}},
			},
			want: "data relocation addend out of range",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad.tobj")
			err := compiler.WriteObject(path, tc.obj)
			if err == nil {
				t.Fatalf("expected malformed range error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
			if data, readErr := os.ReadFile(path); readErr == nil && len(data) > 0 {
				t.Fatalf("partial object was written: % x", data)
			}
		})
	}
}

func TestObjectReadRejectsInvalidMagicAndVersion(t *testing.T) {
	tmp := t.TempDir()

	t.Run("magic", func(t *testing.T) {
		path := filepath.Join(tmp, "bad_magic.tobj")
		if err := os.WriteFile(path, []byte("NOPE"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if _, err := compiler.ReadObject(path); err == nil || !strings.Contains(err.Error(), "invalid object magic") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("version", func(t *testing.T) {
		path := filepath.Join(tmp, "bad_version.tobj")
		if err := compiler.WriteObject(path, &compiler.Object{
			Target: "linux-x64",
			Module: "mod",
			Code:   []byte{0xC3},
		}); err != nil {
			t.Fatalf("write: %v", err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if len(data) < 5 {
			t.Fatalf("object too short")
		}
		data[4] = 0xFF
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("rewrite: %v", err)
		}
		if _, err := compiler.ReadObject(path); err == nil || !strings.Contains(err.Error(), "unsupported object version") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
