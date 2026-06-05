package pgrt

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestDecodeDataRowBorrowedDoesNotCopyCells(t *testing.T) {
	payload := appendDataRow(nil, []string{"7", "70"})
	row, report, err := DecodeDataRowBorrowed(payload, nil)
	if err != nil {
		t.Fatalf("DecodeDataRowBorrowed: %v", err)
	}
	if len(row) != 2 || string(row[0]) != "7" || string(row[1]) != "70" {
		t.Fatalf("row = %#v", row)
	}
	if report.BorrowedCells != 2 || report.CopiedCells != 0 || report.Storage != RowStorageBorrowed {
		t.Fatalf("decode report = %#v", report)
	}
	payload[len(payload)-1] = '1'
	if string(row[1]) != "71" {
		t.Fatalf("row cell did not borrow payload after mutation: %q", row[1])
	}
}

func TestAppendBindBinaryFormatsAndDecodeInt4(t *testing.T) {
	var payload []byte
	payload = AppendBindFormat(payload, "", "world_by_id", []int16{BinaryFormat}, [][]byte{AppendInt4Binary(nil, 7)}, []int16{BinaryFormat})
	frames := splitClientFrames(t, payload)
	if len(frames) != 1 || frames[0].Type != 'B' {
		t.Fatalf("frames = %#v, want one Bind frame", frames)
	}

	r := payloadReader{data: frames[0].Payload}
	portal, ok := r.cstring()
	if !ok || portal != "" {
		t.Fatalf("portal = %q ok=%v", portal, ok)
	}
	statement, ok := r.cstring()
	if !ok || statement != "world_by_id" {
		t.Fatalf("statement = %q ok=%v", statement, ok)
	}
	formatCount, ok := r.int16()
	if !ok || formatCount != 1 {
		t.Fatalf("format count = %d ok=%v", formatCount, ok)
	}
	format, ok := r.int16()
	if !ok || format != BinaryFormat {
		t.Fatalf("format = %d ok=%v", format, ok)
	}
	valueCount, ok := r.int16()
	if !ok || valueCount != 1 {
		t.Fatalf("value count = %d ok=%v", valueCount, ok)
	}
	valueLen, ok := r.int32()
	if !ok || valueLen != 4 {
		t.Fatalf("value length = %d ok=%v", valueLen, ok)
	}
	if got := binary.BigEndian.Uint32(r.data[r.off : r.off+4]); got != 7 {
		t.Fatalf("binary int4 payload = %d, want 7", got)
	}
	r.off += 4
	resultFormatCount, ok := r.int16()
	if !ok || resultFormatCount != 1 {
		t.Fatalf("result format count = %d ok=%v", resultFormatCount, ok)
	}
	resultFormat, ok := r.int16()
	if !ok || resultFormat != BinaryFormat {
		t.Fatalf("result format = %d ok=%v", resultFormat, ok)
	}
	if !r.done() {
		t.Fatalf("trailing bind payload bytes: %#v", r.data[r.off:])
	}

	encoded := AppendInt4Binary(nil, 70)
	decoded, err := DecodeInt4(encoded, BinaryFormat)
	if err != nil || decoded != 70 {
		t.Fatalf("DecodeInt4 binary = %d,%v want 70,nil", decoded, err)
	}
	decoded, err = DecodeInt4([]byte("71"), TextFormat)
	if err != nil || decoded != 71 {
		t.Fatalf("DecodeInt4 text = %d,%v want 71,nil", decoded, err)
	}
	if !bytes.Equal(AppendInt4Binary(nil, 7), []byte{0, 0, 0, 7}) {
		t.Fatalf("AppendInt4Binary did not encode big-endian int4")
	}
}
