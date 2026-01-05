package tobj

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	objectMagic   = "TOBJ"
	objectVersion = 2
)

type Object struct {
	Target       string
	Module       string
	SrcHash      [32]byte
	WorldSigHash [32]byte
	Code         []byte
	Data         []byte
	Symbols      []Symbol
	Relocs       []Reloc
}

type Symbol struct {
	Name   string
	Offset uint32
}

type RelocKind uint8

const (
	RelocCallRel32  RelocKind = 1
	RelocIATDisp32  RelocKind = 2
	RelocDataDisp32 RelocKind = 3
)

type Reloc struct {
	Kind   RelocKind
	At     uint32
	Name   string
	Addend uint32
}

func WriteObject(path string, obj *Object) error {
	var buf bytes.Buffer
	if err := writeObject(&buf, obj); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func ReadObject(path string) (*Object, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return readObject(bytes.NewReader(data))
}

func writeObject(w io.Writer, obj *Object) error {
	if obj == nil {
		return fmt.Errorf("missing object")
	}
	if _, err := w.Write([]byte(objectMagic)); err != nil {
		return err
	}
	if err := writeU8(w, objectVersion); err != nil {
		return err
	}
	if err := writeString(w, obj.Target); err != nil {
		return err
	}
	if err := writeString(w, obj.Module); err != nil {
		return err
	}
	if _, err := w.Write(obj.SrcHash[:]); err != nil {
		return err
	}
	if _, err := w.Write(obj.WorldSigHash[:]); err != nil {
		return err
	}
	if err := writeU32(w, uint32(len(obj.Code))); err != nil {
		return err
	}
	if _, err := w.Write(obj.Code); err != nil {
		return err
	}
	if err := writeU32(w, uint32(len(obj.Data))); err != nil {
		return err
	}
	if _, err := w.Write(obj.Data); err != nil {
		return err
	}
	if err := writeU32(w, uint32(len(obj.Symbols))); err != nil {
		return err
	}
	for _, sym := range obj.Symbols {
		if err := writeString(w, sym.Name); err != nil {
			return err
		}
		if err := writeU32(w, sym.Offset); err != nil {
			return err
		}
	}
	if err := writeU32(w, uint32(len(obj.Relocs))); err != nil {
		return err
	}
	for _, reloc := range obj.Relocs {
		if err := writeU8(w, uint8(reloc.Kind)); err != nil {
			return err
		}
		if err := writeU32(w, reloc.At); err != nil {
			return err
		}
		if err := writeString(w, reloc.Name); err != nil {
			return err
		}
		if err := writeU32(w, reloc.Addend); err != nil {
			return err
		}
	}
	return nil
}

func readObject(r io.Reader) (*Object, error) {
	var magic [4]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return nil, err
	}
	if string(magic[:]) != objectMagic {
		return nil, fmt.Errorf("invalid object magic")
	}
	version, err := readU8(r)
	if err != nil {
		return nil, err
	}
	if version != objectVersion {
		return nil, fmt.Errorf("unsupported object version %d", version)
	}
	target, err := readString(r)
	if err != nil {
		return nil, err
	}
	module, err := readString(r)
	if err != nil {
		return nil, err
	}
	var srcHash [32]byte
	if _, err := io.ReadFull(r, srcHash[:]); err != nil {
		return nil, err
	}
	var worldSig [32]byte
	if _, err := io.ReadFull(r, worldSig[:]); err != nil {
		return nil, err
	}
	codeLen, err := readU32(r)
	if err != nil {
		return nil, err
	}
	code := make([]byte, codeLen)
	if _, err := io.ReadFull(r, code); err != nil {
		return nil, err
	}
	dataLen, err := readU32(r)
	if err != nil {
		return nil, err
	}
	data := make([]byte, dataLen)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	symCount, err := readU32(r)
	if err != nil {
		return nil, err
	}
	symbols := make([]Symbol, 0, symCount)
	for i := uint32(0); i < symCount; i++ {
		name, err := readString(r)
		if err != nil {
			return nil, err
		}
		offset, err := readU32(r)
		if err != nil {
			return nil, err
		}
		symbols = append(symbols, Symbol{Name: name, Offset: offset})
	}
	relocCount, err := readU32(r)
	if err != nil {
		return nil, err
	}
	relocs := make([]Reloc, 0, relocCount)
	for i := uint32(0); i < relocCount; i++ {
		kind, err := readU8(r)
		if err != nil {
			return nil, err
		}
		at, err := readU32(r)
		if err != nil {
			return nil, err
		}
		name, err := readString(r)
		if err != nil {
			return nil, err
		}
		addend, err := readU32(r)
		if err != nil {
			return nil, err
		}
		relocs = append(relocs, Reloc{Kind: RelocKind(kind), At: at, Name: name, Addend: addend})
	}
	return &Object{
		Target:       target,
		Module:       module,
		SrcHash:      srcHash,
		WorldSigHash: worldSig,
		Code:         code,
		Data:         data,
		Symbols:      symbols,
		Relocs:       relocs,
	}, nil
}

func writeU8(w io.Writer, v uint8) error {
	return binary.Write(w, binary.LittleEndian, v)
}

func writeU32(w io.Writer, v uint32) error {
	return binary.Write(w, binary.LittleEndian, v)
}

func readU8(r io.Reader) (uint8, error) {
	var v uint8
	if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
		return 0, err
	}
	return v, nil
}

func readU32(r io.Reader) (uint32, error) {
	var v uint32
	if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
		return 0, err
	}
	return v, nil
}

func writeString(w io.Writer, s string) error {
	if len(s) > 0xFFFF {
		return fmt.Errorf("string too long")
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(len(s))); err != nil {
		return err
	}
	_, err := w.Write([]byte(s))
	return err
}

func readString(r io.Reader) (string, error) {
	var n uint16
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return "", err
	}
	if n == 0 {
		return "", nil
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}
