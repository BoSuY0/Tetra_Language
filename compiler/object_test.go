package compiler

import (
	"bytes"
	"crypto/sha256"
	"path/filepath"
	"reflect"
	"testing"
)

func TestObjectRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.tobj")

	srcHash := sha256.Sum256([]byte("module source"))
	worldHash := sha256.Sum256([]byte("world sig"))
	obj := &Object{
		Target:       "linux-x64",
		Module:       "engine.render",
		SrcHash:      srcHash,
		WorldSigHash: worldHash,
		Code:         []byte{0x90, 0x90, 0xC3},
		Data:         []byte("hi"),
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
