package compiler

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestObjectRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.tobj")

	srcHash := sha256.Sum256([]byte("module source"))
	worldHash := sha256.Sum256([]byte("world sig"))
	obj := &Object{
		Target:          "linux-x64",
		Module:          "engine.render",
		CompilerVersion: Version(),
		PublicAPIHash:   "sha256:abcd",
		SrcHash:         srcHash,
		WorldSigHash:    worldHash,
		Code:            []byte{0x90, 0x90, 0xC3},
		Data:            []byte("hi"),
		Symbols: []Symbol{
			{Name: "engine.render.add_one", Offset: 0},
		},
		Relocs: []Reloc{
			{Kind: RelocCallRel32, At: 1, Name: "app.game.main", Addend: 0},
			{Kind: RelocIATDisp32, At: 2, Name: "kernel32.ExitProcess", Addend: 0},
			{Kind: RelocDataDisp32, At: 3, Name: "", Addend: 4},
		},
	}

	if err := WriteObject(path, obj); err != nil {
		t.Fatalf("write object: %v", err)
	}
	readObj, err := ReadObject(path)
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
	err := WriteObject(path, &Object{
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

func TestObjectReadRejectsInvalidMagicAndVersion(t *testing.T) {
	tmp := t.TempDir()

	t.Run("magic", func(t *testing.T) {
		path := filepath.Join(tmp, "bad_magic.tobj")
		if err := os.WriteFile(path, []byte("NOPE"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if _, err := ReadObject(path); err == nil || !strings.Contains(err.Error(), "invalid object magic") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("version", func(t *testing.T) {
		path := filepath.Join(tmp, "bad_version.tobj")
		if err := WriteObject(path, &Object{
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
		if _, err := ReadObject(path); err == nil || !strings.Contains(err.Error(), "unsupported object version") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
