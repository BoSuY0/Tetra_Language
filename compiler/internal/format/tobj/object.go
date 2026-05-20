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
	objectVersion = 4

	maxSignatureSlotValue = int64(^uint32(0))
)

type Object struct {
	Target          string
	Module          string
	CompilerVersion string
	PublicAPIHash   string
	SrcHash         [32]byte
	WorldSigHash    [32]byte
	Code            []byte
	Data            []byte
	Symbols         []Symbol
	Relocs          []Reloc
}

type Symbol struct {
	Name         string
	Offset       uint32
	HasSignature bool
	ParamSlots   int
	ReturnSlots  int
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
	if err := writeString(w, obj.CompilerVersion); err != nil {
		return err
	}
	if err := writeString(w, obj.PublicAPIHash); err != nil {
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
		if err := validateSymbolRecord(sym, len(obj.Code)); err != nil {
			return err
		}
		if err := writeString(w, sym.Name); err != nil {
			return err
		}
		if err := writeU32(w, sym.Offset); err != nil {
			return err
		}
		if err := writeBool(w, sym.HasSignature); err != nil {
			return err
		}
		if err := writeU32(w, uint32(sym.ParamSlots)); err != nil {
			return err
		}
		if err := writeU32(w, uint32(sym.ReturnSlots)); err != nil {
			return err
		}
	}
	if err := writeU32(w, uint32(len(obj.Relocs))); err != nil {
		return err
	}
	for _, reloc := range obj.Relocs {
		if err := validateRelocRecord(reloc, len(obj.Code), len(obj.Data)); err != nil {
			return err
		}
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
	if version != 2 && version != 3 && version != objectVersion {
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
	compilerVersion := ""
	publicAPIHash := ""
	if version >= 3 {
		compilerVersion, err = readString(r)
		if err != nil {
			return nil, err
		}
		publicAPIHash, err = readString(r)
		if err != nil {
			return nil, err
		}
	}
	codeLen, err := readU32(r)
	if err != nil {
		return nil, err
	}
	code, err := readObjectBytes(r, "code section", codeLen)
	if err != nil {
		return nil, err
	}
	dataLen, err := readU32(r)
	if err != nil {
		return nil, err
	}
	data, err := readObjectBytes(r, "data section", dataLen)
	if err != nil {
		return nil, err
	}
	symCount, err := readU32(r)
	if err != nil {
		return nil, err
	}
	if err := ensureObjectRecordTableAvailable(r, "symbol table", symCount, minSymbolRecordBytes(version)); err != nil {
		return nil, err
	}
	symCap, err := checkedObjectCountCapacity("symbol table", symCount)
	if err != nil {
		return nil, err
	}
	symbols := make([]Symbol, 0, symCap)
	for i := uint32(0); i < symCount; i++ {
		name, err := readString(r)
		if err != nil {
			return nil, err
		}
		offset, err := readU32(r)
		if err != nil {
			return nil, err
		}
		sym := Symbol{Name: name, Offset: offset}
		if version >= 4 {
			hasSignature, err := readBool(r)
			if err != nil {
				return nil, err
			}
			paramSlots, err := readU32(r)
			if err != nil {
				return nil, err
			}
			returnSlots, err := readU32(r)
			if err != nil {
				return nil, err
			}
			sym.HasSignature = hasSignature
			sym.ParamSlots = int(paramSlots)
			sym.ReturnSlots = int(returnSlots)
		}
		if err := validateSymbolRecord(sym, len(code)); err != nil {
			return nil, err
		}
		symbols = append(symbols, sym)
	}
	relocCount, err := readU32(r)
	if err != nil {
		return nil, err
	}
	if err := ensureObjectRecordTableAvailable(r, "relocation table", relocCount, minRelocRecordBytes); err != nil {
		return nil, err
	}
	relocCap, err := checkedObjectCountCapacity("relocation table", relocCount)
	if err != nil {
		return nil, err
	}
	relocs := make([]Reloc, 0, relocCap)
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
		reloc := Reloc{Kind: RelocKind(kind), At: at, Name: name, Addend: addend}
		if err := validateRelocRecord(reloc, len(code), len(data)); err != nil {
			return nil, err
		}
		relocs = append(relocs, reloc)
	}
	return &Object{
		Target:          target,
		Module:          module,
		CompilerVersion: compilerVersion,
		PublicAPIHash:   publicAPIHash,
		SrcHash:         srcHash,
		WorldSigHash:    worldSig,
		Code:            code,
		Data:            data,
		Symbols:         symbols,
		Relocs:          relocs,
	}, nil
}

func validateSymbolRecord(sym Symbol, codeLen int) error {
	if sym.Name == "" {
		return fmt.Errorf("empty symbol name")
	}
	if uint64(sym.Offset) >= uint64(codeLen) {
		return fmt.Errorf("symbol offset out of range for %q", sym.Name)
	}
	if sym.HasSignature && (sym.ParamSlots < 0 || sym.ReturnSlots < 0) {
		return fmt.Errorf("negative symbol signature slots for %q", sym.Name)
	}
	if sym.HasSignature && (int64(sym.ParamSlots) > maxSignatureSlotValue || int64(sym.ReturnSlots) > maxSignatureSlotValue) {
		return fmt.Errorf("symbol signature slots out of range for %q", sym.Name)
	}
	return nil
}

func validateRelocRecord(reloc Reloc, codeLen, dataLen int) error {
	switch reloc.Kind {
	case RelocCallRel32:
		if reloc.Name == "" {
			return fmt.Errorf("call relocation with empty symbol name")
		}
		if reloc.Addend != 0 {
			return fmt.Errorf("call relocation addend must be zero")
		}
	case RelocIATDisp32:
		if reloc.Name == "" {
			return fmt.Errorf("IAT relocation with empty symbol name")
		}
		if reloc.Addend != 0 {
			return fmt.Errorf("IAT relocation addend must be zero")
		}
	case RelocDataDisp32:
		if reloc.Name != "" {
			return fmt.Errorf("data relocation symbol name must be empty")
		}
	default:
		return fmt.Errorf("unsupported relocation kind %d", reloc.Kind)
	}
	if uint64(reloc.At)+4 > uint64(codeLen) {
		return fmt.Errorf("relocation offset out of range for kind %d", reloc.Kind)
	}
	if reloc.Kind == RelocDataDisp32 {
		if dataLen == 0 {
			return fmt.Errorf("data relocation in empty data section")
		}
		if uint64(reloc.Addend) >= uint64(dataLen) {
			return fmt.Errorf("data relocation addend out of range")
		}
	}
	return nil
}

func readObjectBytes(r io.Reader, section string, n uint32) ([]byte, error) {
	if err := ensureObjectBytesAvailable(r, section, n); err != nil {
		return nil, err
	}
	buf := make([]byte, int(n))
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("truncated object %s: %w", section, err)
	}
	return buf, nil
}

func ensureObjectBytesAvailable(r io.Reader, section string, n uint32) error {
	if uint64(n) > uint64(maxObjectInt()) {
		return fmt.Errorf("object %s too large: %d bytes", section, n)
	}
	if rem, ok := objectRemainingBytes(r); ok && uint64(n) > uint64(rem) {
		return fmt.Errorf("truncated object %s: declared %d bytes, remaining %d", section, n, rem)
	}
	return nil
}

func ensureObjectRecordTableAvailable(r io.Reader, table string, count uint32, minRecordBytes int) error {
	if count == 0 {
		return nil
	}
	if rem, ok := objectRemainingBytes(r); ok {
		minBytes := uint64(count) * uint64(minRecordBytes)
		if minBytes > uint64(rem) {
			return fmt.Errorf("truncated object %s: declared %d records, remaining %d bytes", table, count, rem)
		}
	}
	return nil
}

func checkedObjectCountCapacity(table string, count uint32) (int, error) {
	if uint64(count) > uint64(maxObjectInt()) {
		return 0, fmt.Errorf("object %s count too large: %d", table, count)
	}
	return int(count), nil
}

func objectRemainingBytes(r io.Reader) (int, bool) {
	lr, ok := r.(interface{ Len() int })
	if !ok {
		return 0, false
	}
	return lr.Len(), true
}

func minSymbolRecordBytes(version uint8) int {
	if version >= 4 {
		return 2 + 4 + 1 + 4 + 4
	}
	return 2 + 4
}

const minRelocRecordBytes = 1 + 4 + 2 + 4

func maxObjectInt() int {
	return int(^uint(0) >> 1)
}

func writeU8(w io.Writer, v uint8) error {
	return binary.Write(w, binary.LittleEndian, v)
}

func writeBool(w io.Writer, v bool) error {
	if v {
		return writeU8(w, 1)
	}
	return writeU8(w, 0)
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

func readBool(r io.Reader) (bool, error) {
	v, err := readU8(r)
	if err != nil {
		return false, err
	}
	switch v {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("invalid bool value %d", v)
	}
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
