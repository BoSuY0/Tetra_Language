package tobj

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestObjectRoundTripPreservesV4MetadataSymbolsAndRelocs(t *testing.T) {
	obj := &Object{
		Target:          "linux-x64",
		Module:          "app.main",
		CompilerVersion: "0.4.0-test",
		PublicAPIHash:   "api-hash",
		SrcHash:         sha256.Sum256([]byte("source")),
		WorldSigHash:    sha256.Sum256([]byte("world")),
		Code:            []byte{0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0xc3},
		Data:            []byte("literal!"),
		Symbols: []Symbol{
			{Name: "helper", Offset: 0, HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
			{Name: "main", Offset: 1, HasSignature: true, ParamSlots: 0, ReturnSlots: 1},
		},
		Relocs: []Reloc{
			{Kind: RelocCallRel32, At: 1, Name: "helper"},
			{Kind: RelocDataDisp32, At: 4, Addend: 7},
			{Kind: RelocIATDisp32, At: 8, Name: "kernel32.WriteFile"},
			{Kind: RelocFuncAddrDisp32, At: 0, Name: "helper"},
			{Kind: RelocDataAbs32, At: 2, Addend: 4},
			{Kind: RelocFuncAddrAbs32, At: 6, Name: "main"},
		},
	}

	path := filepath.Join(t.TempDir(), "main.tobj")
	if err := WriteObject(path, obj); err != nil {
		t.Fatalf("WriteObject: %v", err)
	}
	got, err := ReadObject(path)
	if err != nil {
		t.Fatalf("ReadObject: %v", err)
	}
	if !reflect.DeepEqual(got, obj) {
		t.Fatalf("round-trip mismatch\n got=%#v\nwant=%#v", got, obj)
	}
}

func TestWriteObjectRejectsNegativeSignatureSlots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.tobj")
	err := WriteObject(path, &Object{
		Target: "linux-x64",
		Code:   []byte{0xC3},
		Symbols: []Symbol{
			{Name: "bad", HasSignature: true, ParamSlots: -1},
		},
	})
	if err == nil {
		t.Fatalf("expected negative signature slot error")
	}
	if !strings.Contains(err.Error(), "negative symbol signature slots") {
		t.Fatalf("error = %v", err)
	}
	if data, readErr := os.ReadFile(path); readErr == nil && len(data) > 0 {
		t.Fatalf("partial object was written: % x", data)
	}
}

func TestWriteObjectRejectsSignatureSlotsAboveUint32(t *testing.T) {
	if strconv.IntSize <= 32 {
		t.Skip("int cannot represent values above uint32 on this platform")
	}
	path := filepath.Join(t.TempDir(), "bad.tobj")
	tooLarge := int(uint64(^uint32(0)) + 1)
	err := WriteObject(path, &Object{
		Target: "linux-x64",
		Code:   []byte{0xC3},
		Symbols: []Symbol{
			{Name: "bad", HasSignature: true, ParamSlots: tooLarge, ReturnSlots: 1},
		},
	})
	if err == nil {
		t.Fatalf("expected oversized signature slot error")
	}
	if !strings.Contains(err.Error(), "symbol signature slots out of range") {
		t.Fatalf("error = %v", err)
	}
	if data, readErr := os.ReadFile(path); readErr == nil && len(data) > 0 {
		t.Fatalf("partial object was written: % x", data)
	}
}

func TestWriteObjectRejectsMalformedSymbolAndRelocNames(t *testing.T) {
	cases := []struct {
		name string
		obj  *Object
		want string
	}{
		{
			name: "empty_symbol_name",
			obj: &Object{
				Target:  "linux-x64",
				Module:  "app.main",
				Code:    []byte{0xC3},
				Symbols: []Symbol{{Name: "", Offset: 0}},
			},
			want: "empty symbol name",
		},
		{
			name: "empty_call_relocation_name",
			obj: &Object{
				Target:  "linux-x64",
				Module:  "app.main",
				Code:    []byte{0xE8, 0, 0, 0, 0, 0xC3},
				Symbols: []Symbol{{Name: "main", Offset: 0}},
				Relocs:  []Reloc{{Kind: RelocCallRel32, At: 1, Name: ""}},
			},
			want: "call relocation with empty symbol name",
		},
		{
			name: "empty_iat_relocation_name",
			obj: &Object{
				Target:  "windows-x64",
				Module:  "app.main",
				Code:    []byte{0xFF, 0x15, 0, 0, 0, 0, 0xC3},
				Symbols: []Symbol{{Name: "main", Offset: 0}},
				Relocs:  []Reloc{{Kind: RelocIATDisp32, At: 2, Name: ""}},
			},
			want: "IAT relocation with empty symbol name",
		},
		{
			name: "empty_function_address_relocation_name",
			obj: &Object{
				Target:  "linux-x64",
				Module:  "app.main",
				Code:    []byte{0x48, 0x8D, 0x05, 0, 0, 0, 0, 0xC3},
				Symbols: []Symbol{{Name: "main", Offset: 0}},
				Relocs:  []Reloc{{Kind: RelocFuncAddrDisp32, At: 3, Name: ""}},
			},
			want: "function address relocation with empty symbol name",
		},
		{
			name: "empty_absolute_function_address_relocation_name",
			obj: &Object{
				Target:  "linux-x86",
				Module:  "app.main",
				Code:    []byte{0xB8, 0, 0, 0, 0, 0xC3},
				Symbols: []Symbol{{Name: "main", Offset: 0}},
				Relocs:  []Reloc{{Kind: RelocFuncAddrAbs32, At: 1, Name: ""}},
			},
			want: "function address relocation with empty symbol name",
		},
		{
			name: "unsupported_relocation_kind",
			obj: &Object{
				Target:  "linux-x64",
				Module:  "app.main",
				Code:    []byte{0xC3},
				Symbols: []Symbol{{Name: "main", Offset: 0}},
				Relocs:  []Reloc{{Kind: RelocKind(255), At: 0, Name: "bad"}},
			},
			want: "unsupported relocation kind 255",
		},
		{
			name: "named_data_relocation",
			obj: &Object{
				Target:  "linux-x64",
				Module:  "app.main",
				Code:    []byte{0, 0, 0, 0},
				Data:    []byte("A"),
				Symbols: []Symbol{{Name: "main", Offset: 0}},
				Relocs:  []Reloc{{Kind: RelocDataDisp32, At: 0, Name: "data.symbol", Addend: 0}},
			},
			want: "data relocation symbol name must be empty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad.tobj")
			err := WriteObject(path, tc.obj)
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

func TestWriteObjectRejectsNonDataRelocationAddends(t *testing.T) {
	cases := []struct {
		name  string
		reloc Reloc
		want  string
	}{
		{
			name:  "call_relocation_addend",
			reloc: Reloc{Kind: RelocCallRel32, At: 1, Name: "main", Addend: 7},
			want:  "call relocation addend must be zero",
		},
		{
			name:  "iat_relocation_addend",
			reloc: Reloc{Kind: RelocIATDisp32, At: 2, Name: "kernel32.ExitProcess", Addend: 7},
			want:  "IAT relocation addend must be zero",
		},
		{
			name:  "function_address_relocation_addend",
			reloc: Reloc{Kind: RelocFuncAddrDisp32, At: 1, Name: "main", Addend: 7},
			want:  "function address relocation addend must be zero",
		},
		{
			name:  "absolute_function_address_relocation_addend",
			reloc: Reloc{Kind: RelocFuncAddrAbs32, At: 1, Name: "main", Addend: 7},
			want:  "function address relocation addend must be zero",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad.tobj")
			err := WriteObject(path, &Object{
				Target:  "linux-x64",
				Module:  "app.main",
				Code:    []byte{0xE8, 0, 0, 0, 0, 0xC3},
				Symbols: []Symbol{{Name: "main", Offset: 0}},
				Relocs:  []Reloc{tc.reloc},
			})
			if err == nil {
				t.Fatalf("expected non-data relocation addend error")
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

func TestWriteObjectRejectsMalformedSymbolAndRelocRanges(t *testing.T) {
	cases := []struct {
		name string
		obj  *Object
		want string
	}{
		{
			name: "symbol_offset_out_of_range",
			obj: &Object{
				Target:  "linux-x64",
				Module:  "app.main",
				Code:    []byte{0xC3},
				Symbols: []Symbol{{Name: "main", Offset: 1}},
			},
			want: "symbol offset out of range",
		},
		{
			name: "call_relocation_offset_out_of_range",
			obj: &Object{
				Target:  "linux-x64",
				Module:  "app.main",
				Code:    []byte{0xE8, 0, 0, 0},
				Symbols: []Symbol{{Name: "main", Offset: 0}},
				Relocs:  []Reloc{{Kind: RelocCallRel32, At: 1, Name: "main"}},
			},
			want: "relocation offset out of range",
		},
		{
			name: "iat_relocation_offset_out_of_range",
			obj: &Object{
				Target:  "windows-x64",
				Module:  "app.main",
				Code:    []byte{0xFF, 0x15, 0, 0},
				Symbols: []Symbol{{Name: "main", Offset: 0}},
				Relocs:  []Reloc{{Kind: RelocIATDisp32, At: 1, Name: "kernel32.ExitProcess"}},
			},
			want: "relocation offset out of range",
		},
		{
			name: "function_address_relocation_offset_out_of_range",
			obj: &Object{
				Target:  "linux-x64",
				Module:  "app.main",
				Code:    []byte{0x48, 0x8D, 0x05, 0, 0, 0},
				Symbols: []Symbol{{Name: "main", Offset: 0}},
				Relocs:  []Reloc{{Kind: RelocFuncAddrDisp32, At: 3, Name: "main"}},
			},
			want: "relocation offset out of range",
		},
		{
			name: "data_relocation_offset_out_of_range",
			obj: &Object{
				Target:  "linux-x64",
				Module:  "app.main",
				Code:    []byte{0x8D, 0, 0, 0},
				Data:    []byte("AB"),
				Symbols: []Symbol{{Name: "main", Offset: 0}},
				Relocs:  []Reloc{{Kind: RelocDataDisp32, At: 1, Addend: 0}},
			},
			want: "relocation offset out of range",
		},
		{
			name: "data_relocation_empty_data_section",
			obj: &Object{
				Target:  "linux-x64",
				Module:  "app.main",
				Code:    []byte{0, 0, 0, 0},
				Symbols: []Symbol{{Name: "main", Offset: 0}},
				Relocs:  []Reloc{{Kind: RelocDataDisp32, At: 0, Addend: 0}},
			},
			want: "data relocation in empty data section",
		},
		{
			name: "data_relocation_addend_out_of_range",
			obj: &Object{
				Target:  "linux-x64",
				Module:  "app.main",
				Code:    []byte{0, 0, 0, 0},
				Data:    []byte("A"),
				Symbols: []Symbol{{Name: "main", Offset: 0}},
				Relocs:  []Reloc{{Kind: RelocDataDisp32, At: 0, Addend: 1}},
			},
			want: "data relocation addend out of range",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad.tobj")
			err := WriteObject(path, tc.obj)
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

func TestReadObjectRejectsInvalidMagicAndBoolEncoding(t *testing.T) {
	if _, err := readObject(bytes.NewReader([]byte("NOPE"))); err == nil || !strings.Contains(err.Error(), "invalid object magic") {
		t.Fatalf("invalid magic error = %v", err)
	}

	var raw bytes.Buffer
	raw.WriteString(objectMagic)
	if err := writeU8(&raw, objectVersion); err != nil {
		t.Fatalf("write version: %v", err)
	}
	for _, s := range []string{"linux-x64", "app.main", "0.4.0-test", "api-hash"} {
		if err := writeString(&raw, s); err != nil {
			t.Fatalf("write string %q: %v", s, err)
		}
		if s == "app.main" {
			raw.Write(make([]byte, 64))
		}
	}
	if err := writeU32(&raw, 0); err != nil {
		t.Fatalf("write code len: %v", err)
	}
	if err := writeU32(&raw, 0); err != nil {
		t.Fatalf("write data len: %v", err)
	}
	if err := writeU32(&raw, 1); err != nil {
		t.Fatalf("write symbol count: %v", err)
	}
	if err := writeString(&raw, "main"); err != nil {
		t.Fatalf("write symbol name: %v", err)
	}
	if err := writeU32(&raw, 0); err != nil {
		t.Fatalf("write symbol offset: %v", err)
	}
	if err := writeU8(&raw, 2); err != nil {
		t.Fatalf("write invalid bool: %v", err)
	}
	if err := writeU32(&raw, 0); err != nil {
		t.Fatalf("write param slots: %v", err)
	}
	if err := writeU32(&raw, 0); err != nil {
		t.Fatalf("write return slots: %v", err)
	}

	if _, err := readObject(bytes.NewReader(raw.Bytes())); err == nil || !strings.Contains(err.Error(), "invalid bool value 2") {
		t.Fatalf("invalid bool error = %v", err)
	}
}

func TestReadObjectRejectsMalformedSymbolAndRelocRanges(t *testing.T) {
	cases := []struct {
		name string
		raw  func(t *testing.T) []byte
		want string
	}{
		{
			name: "symbol_offset_out_of_range",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeObjectPayloadForTest(t, &raw, []byte{0xC3}, nil)
				writeSymbolRecordForTest(t, &raw, "main", 1)
				writeRelocCountForTest(t, &raw, 0)
				return raw.Bytes()
			},
			want: "symbol offset out of range",
		},
		{
			name: "call_relocation_offset_out_of_range",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeObjectPayloadForTest(t, &raw, []byte{0xE8, 0, 0, 0}, nil)
				writeSymbolRecordForTest(t, &raw, "main", 0)
				writeRelocRecordForTest(t, &raw, RelocCallRel32, 1, "main", 0)
				return raw.Bytes()
			},
			want: "relocation offset out of range",
		},
		{
			name: "iat_relocation_offset_out_of_range",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeObjectPayloadForTest(t, &raw, []byte{0xFF, 0x15, 0, 0}, nil)
				writeSymbolRecordForTest(t, &raw, "main", 0)
				writeRelocRecordForTest(t, &raw, RelocIATDisp32, 1, "kernel32.ExitProcess", 0)
				return raw.Bytes()
			},
			want: "relocation offset out of range",
		},
		{
			name: "function_address_relocation_offset_out_of_range",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeObjectPayloadForTest(t, &raw, []byte{0x48, 0x8D, 0x05, 0, 0, 0}, nil)
				writeSymbolRecordForTest(t, &raw, "main", 0)
				writeRelocRecordForTest(t, &raw, RelocFuncAddrDisp32, 3, "main", 0)
				return raw.Bytes()
			},
			want: "relocation offset out of range",
		},
		{
			name: "data_relocation_offset_out_of_range",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeObjectPayloadForTest(t, &raw, []byte{0x8D, 0, 0, 0}, []byte("AB"))
				writeSymbolRecordForTest(t, &raw, "main", 0)
				writeRelocRecordForTest(t, &raw, RelocDataDisp32, 1, "", 0)
				return raw.Bytes()
			},
			want: "relocation offset out of range",
		},
		{
			name: "data_relocation_empty_data_section",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeObjectPayloadForTest(t, &raw, []byte{0, 0, 0, 0}, nil)
				writeSymbolRecordForTest(t, &raw, "main", 0)
				writeRelocRecordForTest(t, &raw, RelocDataDisp32, 0, "", 0)
				return raw.Bytes()
			},
			want: "data relocation in empty data section",
		},
		{
			name: "data_relocation_addend_out_of_range",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeObjectPayloadForTest(t, &raw, []byte{0, 0, 0, 0}, []byte("A"))
				writeSymbolRecordForTest(t, &raw, "main", 0)
				writeRelocRecordForTest(t, &raw, RelocDataDisp32, 0, "", 1)
				return raw.Bytes()
			},
			want: "data relocation addend out of range",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readObject(bytes.NewReader(tc.raw(t)))
			if err == nil {
				t.Fatalf("expected malformed range error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestReadObjectRejectsMalformedSymbolAndRelocNames(t *testing.T) {
	cases := []struct {
		name string
		raw  func(t *testing.T) []byte
		want string
	}{
		{
			name: "empty_symbol_name",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeObjectPayloadForTest(t, &raw, []byte{0xC3}, nil)
				writeSymbolRecordForTest(t, &raw, "", 0)
				writeRelocCountForTest(t, &raw, 0)
				return raw.Bytes()
			},
			want: "empty symbol name",
		},
		{
			name: "empty_call_relocation_name",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeObjectPayloadForTest(t, &raw, []byte{0xE8, 0, 0, 0, 0, 0xC3}, nil)
				writeSymbolRecordForTest(t, &raw, "main", 0)
				writeRelocRecordForTest(t, &raw, RelocCallRel32, 1, "", 0)
				return raw.Bytes()
			},
			want: "call relocation with empty symbol name",
		},
		{
			name: "empty_iat_relocation_name",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeObjectPayloadForTest(t, &raw, []byte{0xFF, 0x15, 0, 0, 0, 0, 0xC3}, nil)
				writeSymbolRecordForTest(t, &raw, "main", 0)
				writeRelocRecordForTest(t, &raw, RelocIATDisp32, 2, "", 0)
				return raw.Bytes()
			},
			want: "IAT relocation with empty symbol name",
		},
		{
			name: "empty_function_address_relocation_name",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeObjectPayloadForTest(t, &raw, []byte{0x48, 0x8D, 0x05, 0, 0, 0, 0, 0xC3}, nil)
				writeSymbolRecordForTest(t, &raw, "main", 0)
				writeRelocRecordForTest(t, &raw, RelocFuncAddrDisp32, 3, "", 0)
				return raw.Bytes()
			},
			want: "function address relocation with empty symbol name",
		},
		{
			name: "empty_absolute_function_address_relocation_name",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeObjectPayloadForTest(t, &raw, []byte{0xB8, 0, 0, 0, 0, 0xC3}, nil)
				writeSymbolRecordForTest(t, &raw, "main", 0)
				writeRelocRecordForTest(t, &raw, RelocFuncAddrAbs32, 1, "", 0)
				return raw.Bytes()
			},
			want: "function address relocation with empty symbol name",
		},
		{
			name: "unsupported_relocation_kind",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeObjectPayloadForTest(t, &raw, []byte{0xC3}, nil)
				writeSymbolRecordForTest(t, &raw, "main", 0)
				writeRelocRecordForTest(t, &raw, RelocKind(255), 0, "bad", 0)
				return raw.Bytes()
			},
			want: "unsupported relocation kind 255",
		},
		{
			name: "named_data_relocation",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeObjectPayloadForTest(t, &raw, []byte{0, 0, 0, 0}, []byte("A"))
				writeSymbolRecordForTest(t, &raw, "main", 0)
				writeRelocRecordForTest(t, &raw, RelocDataDisp32, 0, "data.symbol", 0)
				return raw.Bytes()
			},
			want: "data relocation symbol name must be empty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readObject(bytes.NewReader(tc.raw(t)))
			if err == nil {
				t.Fatalf("expected malformed object error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestReadObjectRejectsNonDataRelocationAddends(t *testing.T) {
	cases := []struct {
		name string
		kind RelocKind
		at   uint32
		sym  string
		want string
	}{
		{
			name: "call_relocation_addend",
			kind: RelocCallRel32,
			at:   1,
			sym:  "main",
			want: "call relocation addend must be zero",
		},
		{
			name: "iat_relocation_addend",
			kind: RelocIATDisp32,
			at:   2,
			sym:  "kernel32.ExitProcess",
			want: "IAT relocation addend must be zero",
		},
		{
			name: "function_address_relocation_addend",
			kind: RelocFuncAddrDisp32,
			at:   1,
			sym:  "main",
			want: "function address relocation addend must be zero",
		},
		{
			name: "absolute_function_address_relocation_addend",
			kind: RelocFuncAddrAbs32,
			at:   1,
			sym:  "main",
			want: "function address relocation addend must be zero",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var raw bytes.Buffer
			writeObjectHeaderForTest(t, &raw)
			writeObjectPayloadForTest(t, &raw, []byte{0xE8, 0, 0, 0, 0, 0xC3}, nil)
			writeSymbolRecordForTest(t, &raw, "main", 0)
			writeRelocRecordForTest(t, &raw, tc.kind, tc.at, tc.sym, 7)

			_, err := readObject(bytes.NewReader(raw.Bytes()))
			if err == nil {
				t.Fatalf("expected non-data relocation addend error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestReadObjectRejectsTruncatedDeclaredSections(t *testing.T) {
	cases := []struct {
		name string
		raw  func(t *testing.T) []byte
		want string
	}{
		{
			name: "code",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeU32ForTest(t, &raw, 64)
				return raw.Bytes()
			},
			want: "truncated object code section",
		},
		{
			name: "data",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeU32ForTest(t, &raw, 0)
				writeU32ForTest(t, &raw, 64)
				return raw.Bytes()
			},
			want: "truncated object data section",
		},
		{
			name: "symbols",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeU32ForTest(t, &raw, 0)
				writeU32ForTest(t, &raw, 0)
				writeU32ForTest(t, &raw, 1)
				return raw.Bytes()
			},
			want: "truncated object symbol table",
		},
		{
			name: "relocs",
			raw: func(t *testing.T) []byte {
				var raw bytes.Buffer
				writeObjectHeaderForTest(t, &raw)
				writeU32ForTest(t, &raw, 0)
				writeU32ForTest(t, &raw, 0)
				writeU32ForTest(t, &raw, 0)
				writeU32ForTest(t, &raw, 1)
				return raw.Bytes()
			},
			want: "truncated object relocation table",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readObject(bytes.NewReader(tc.raw(t)))
			if err == nil {
				t.Fatalf("expected truncated object error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func writeObjectPayloadForTest(t *testing.T, raw *bytes.Buffer, code, data []byte) {
	t.Helper()
	writeU32ForTest(t, raw, uint32(len(code)))
	if _, err := raw.Write(code); err != nil {
		t.Fatalf("write code: %v", err)
	}
	writeU32ForTest(t, raw, uint32(len(data)))
	if _, err := raw.Write(data); err != nil {
		t.Fatalf("write data: %v", err)
	}
}

func writeSymbolRecordForTest(t *testing.T, raw *bytes.Buffer, name string, offset uint32) {
	t.Helper()
	writeU32ForTest(t, raw, 1)
	if err := writeString(raw, name); err != nil {
		t.Fatalf("write symbol name: %v", err)
	}
	writeU32ForTest(t, raw, offset)
	if err := writeBool(raw, false); err != nil {
		t.Fatalf("write symbol signature flag: %v", err)
	}
	writeU32ForTest(t, raw, 0)
	writeU32ForTest(t, raw, 0)
}

func writeRelocRecordForTest(t *testing.T, raw *bytes.Buffer, kind RelocKind, at uint32, name string, addend uint32) {
	t.Helper()
	writeRelocCountForTest(t, raw, 1)
	if err := writeU8(raw, uint8(kind)); err != nil {
		t.Fatalf("write reloc kind: %v", err)
	}
	writeU32ForTest(t, raw, at)
	if err := writeString(raw, name); err != nil {
		t.Fatalf("write reloc name: %v", err)
	}
	writeU32ForTest(t, raw, addend)
}

func writeRelocCountForTest(t *testing.T, raw *bytes.Buffer, count uint32) {
	t.Helper()
	writeU32ForTest(t, raw, count)
}

func writeObjectHeaderForTest(t *testing.T, raw *bytes.Buffer) {
	t.Helper()
	raw.WriteString(objectMagic)
	if err := writeU8(raw, objectVersion); err != nil {
		t.Fatalf("write version: %v", err)
	}
	for _, s := range []string{"linux-x64", "app.main"} {
		if err := writeString(raw, s); err != nil {
			t.Fatalf("write string %q: %v", s, err)
		}
	}
	raw.Write(make([]byte, 64))
	for _, s := range []string{"0.4.0-test", "api-hash"} {
		if err := writeString(raw, s); err != nil {
			t.Fatalf("write string %q: %v", s, err)
		}
	}
}

func writeU32ForTest(t *testing.T, raw *bytes.Buffer, v uint32) {
	t.Helper()
	if err := writeU32(raw, v); err != nil {
		t.Fatalf("write u32 %d: %v", v, err)
	}
}
