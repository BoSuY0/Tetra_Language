package jsonrt

import (
	"errors"
	"testing"

	"tetra_language/compiler/internal/stdlibrt"
)

func TestParseValueViewBorrowsUnescapedStringsWithoutHeap(t *testing.T) {
	raw := []byte(`{"message":"hello","id":123}`)
	region := stdlibrt.NewRegion("json-request", 256)

	value, report, err := ParseValueView(raw, ParseViewOptions{Region: region})
	if err != nil {
		t.Fatalf("ParseValueView: %v", err)
	}
	if value.Kind != ObjectKind || len(value.Object) != 2 {
		t.Fatalf("value = %#v, want object with two members", value)
	}
	message := value.ObjectMember("message")
	if message == nil || message.Kind != StringKind {
		t.Fatalf("message member = %#v, want string", message)
	}
	if string(message.String.Bytes) != "hello" {
		t.Fatalf("message string bytes = %q, want hello", message.String.Bytes)
	}
	if message.String.Storage != stdlibrt.StorageBorrowed {
		t.Fatalf("message storage = %q, want borrowed", message.String.Storage)
	}
	if report.HeapAllocations != 0 || report.CopiedStrings != 0 || report.BorrowedStrings < 3 || report.UnsafeFacts {
		t.Fatalf("parse report = %#v", report)
	}
}

func TestParseValueViewCopiesEscapedStringsIntoRegionOnlyWhenNeeded(t *testing.T) {
	raw := []byte(`{"message":"hello\nworld","plain":"ok"}`)
	region := stdlibrt.NewRegion("json-request", 256)

	value, report, err := ParseValueView(raw, ParseViewOptions{Region: region})
	if err != nil {
		t.Fatalf("ParseValueView escaped: %v", err)
	}
	escaped := value.ObjectMember("message")
	if escaped == nil || string(escaped.String.Bytes) != "hello\nworld" {
		t.Fatalf("escaped string = %#v", escaped)
	}
	if escaped.String.Storage != stdlibrt.StorageRegion || escaped.String.RegionID != "json-request" {
		t.Fatalf("escaped string storage = %#v", escaped.String)
	}
	plain := value.ObjectMember("plain")
	if plain == nil || plain.String.Storage != stdlibrt.StorageBorrowed {
		t.Fatalf("plain string storage = %#v", plain)
	}
	if report.CopiedStrings != 1 || report.RegionTemporaries != 1 || report.HeapAllocations != 0 || report.UnsafeFacts {
		t.Fatalf("escaped parse report = %#v", report)
	}
}

func TestParseValueViewDecodesEscapedStringIntoRegionWithoutHeap(t *testing.T) {
	raw := []byte(`"hello\nworld"`)
	region := stdlibrt.NewRegion("json-request", 256)
	var value ValueView
	var report ParseViewReport
	var err error

	allocs := testing.AllocsPerRun(1000, func() {
		if resetErr := region.Reset(); resetErr != nil {
			err = resetErr
			return
		}
		value, report, err = ParseValueView(raw, ParseViewOptions{Region: region})
	})
	if err != nil {
		t.Fatalf("ParseValueView escaped string: %v", err)
	}
	if allocs != 0 {
		t.Fatalf("ParseValueView escaped string allocations = %.2f, want 0", allocs)
	}
	if value.Kind != StringKind || string(value.String.Bytes) != "hello\nworld" {
		t.Fatalf("escaped string value = %#v, want decoded string", value)
	}
	if value.String.Storage != stdlibrt.StorageRegion || value.String.RegionID != "json-request" {
		t.Fatalf("escaped string storage = %#v, want request region", value.String)
	}
	if report.HeapAllocations != 0 || report.RegionTemporaries != 1 || report.CopiedStrings != 1 || report.UnsafeFacts {
		t.Fatalf("escaped string report = %#v", report)
	}
}

func TestParseValueViewRejectsInvalidInputWithoutUnsafeFacts(t *testing.T) {
	_, report, err := ParseValueView([]byte(`{"message":`), ParseViewOptions{})
	if err == nil {
		t.Fatalf("ParseValueView accepted invalid JSON")
	}
	if !errors.Is(err, ErrInvalidViewJSON) {
		t.Fatalf("ParseValueView invalid error = %v, want ErrInvalidViewJSON", err)
	}
	if report.UnsafeFacts || report.BorrowedStrings != 0 || report.CopiedStrings != 0 {
		t.Fatalf("invalid parse report = %#v", report)
	}
}
