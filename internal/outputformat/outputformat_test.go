package outputformat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/internal/toon"
)

func TestMediaTypesAndFormatPredicates(t *testing.T) {
	if MediaTypeJSON != "application/json" {
		t.Fatalf("MediaTypeJSON = %q", MediaTypeJSON)
	}
	if MediaTypeTOON != "text/toon; charset=utf-8" {
		t.Fatalf("MediaTypeTOON = %q", MediaTypeTOON)
	}
	if !Structured(JSON) || !Structured(TOON) {
		t.Fatalf("json/toon must be structured formats")
	}
	if Structured(Both) {
		t.Fatalf("both is a file/mirror mode, not a single structured stream")
	}
	if !StructuredOrBoth(Both) {
		t.Fatalf("both must be accepted for file/mirror output")
	}
}

func TestInferFormatFromPath(t *testing.T) {
	cases := []struct {
		path     string
		fallback string
		want     string
	}{
		{path: "report.json", fallback: JSON, want: JSON},
		{path: "report.toon", fallback: JSON, want: TOON},
		{path: "report", fallback: JSON, want: JSON},
		{path: "report", fallback: Both, want: Both},
	}
	for _, tc := range cases {
		got, err := InferFormatFromPath(tc.path, tc.fallback)
		if err != nil {
			t.Fatalf("InferFormatFromPath(%q, %q): %v", tc.path, tc.fallback, err)
		}
		if got != tc.want {
			t.Fatalf("InferFormatFromPath(%q, %q) = %q, want %q", tc.path, tc.fallback, got, tc.want)
		}
	}
	if _, err := InferFormatFromPath("report.yaml", JSON); err != nil {
		t.Fatalf("unknown extension should keep fallback: %v", err)
	}
	if _, err := InferFormatFromPath("report", "xml"); err == nil || !strings.Contains(err.Error(), "unsupported structured output format") {
		t.Fatalf("unsupported fallback error = %v", err)
	}
}

func TestOutputPathsForFormat(t *testing.T) {
	cases := []struct {
		path   string
		format string
		want   []OutputFile
	}{
		{path: "summary.json", format: JSON, want: []OutputFile{{Path: "summary.json", Format: JSON}}},
		{path: "summary.toon", format: TOON, want: []OutputFile{{Path: "summary.toon", Format: TOON}}},
		{path: "summary.json", format: Both, want: []OutputFile{{Path: "summary.json", Format: JSON}, {Path: "summary.toon", Format: TOON}}},
		{path: "summary.toon", format: Both, want: []OutputFile{{Path: "summary.json", Format: JSON}, {Path: "summary.toon", Format: TOON}}},
		{path: "summary", format: Both, want: []OutputFile{{Path: "summary.json", Format: JSON}, {Path: "summary.toon", Format: TOON}}},
	}
	for _, tc := range cases {
		got, err := OutputPathsForFormat(tc.path, tc.format)
		if err != nil {
			t.Fatalf("OutputPathsForFormat(%q, %q): %v", tc.path, tc.format, err)
		}
		if len(got) != len(tc.want) {
			t.Fatalf("OutputPathsForFormat(%q, %q) length = %d, want %d (%#v)", tc.path, tc.format, len(got), len(tc.want), got)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("OutputPathsForFormat(%q, %q)[%d] = %#v, want %#v", tc.path, tc.format, i, got[i], tc.want[i])
			}
		}
	}
}

func TestWriteStructuredFilesBothWritesJSONAndTOONFromSameModel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "summary.json")
	value := map[string]any{
		"schema": "tetra.test.v1",
		"status": "pass",
		"count":  float64(2),
	}
	written, err := WriteStructuredFiles(path, Both, value)
	if err != nil {
		t.Fatalf("WriteStructuredFiles both: %v", err)
	}
	if len(written) != 2 || written[0] != path || written[1] != filepath.Join(dir, "summary.toon") {
		t.Fatalf("written paths = %#v", written)
	}
	jsonRaw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}
	toonRaw, err := os.ReadFile(filepath.Join(dir, "summary.toon"))
	if err != nil {
		t.Fatalf("read toon: %v", err)
	}
	if !json.Valid(jsonRaw) {
		t.Fatalf("json output is invalid:\n%s", jsonRaw)
	}
	converted, err := toon.ConvertTOONToJSON(toonRaw, toon.Options{Strict: true})
	if err != nil {
		t.Fatalf("TOON output does not decode: %v\n%s", err, toonRaw)
	}
	var jsonValue any
	var toonValue any
	if err := json.Unmarshal(jsonRaw, &jsonValue); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if err := json.Unmarshal(converted, &toonValue); err != nil {
		t.Fatalf("unmarshal converted toon: %v", err)
	}
	if jsonValue.(map[string]any)["schema"] != toonValue.(map[string]any)["schema"] ||
		jsonValue.(map[string]any)["status"] != toonValue.(map[string]any)["status"] ||
		jsonValue.(map[string]any)["count"] != toonValue.(map[string]any)["count"] {
		t.Fatalf("json/toon semantic mismatch:\njson=%s\ntoon=%s\nconverted=%s", jsonRaw, toonRaw, converted)
	}
}

func TestDecodeStructuredStrictAcceptsJSONAndTOON(t *testing.T) {
	type report struct {
		Schema string `json:"schema"`
		Status string `json:"status"`
	}
	for _, tc := range []struct {
		name   string
		format string
		raw    []byte
	}{
		{name: "json", format: JSON, raw: []byte(`{"schema":"tetra.test.v1","status":"pass"}`)},
		{name: "toon", format: TOON, raw: []byte("schema: tetra.test.v1\nstatus: pass\n")},
		{name: "auto toon", format: Auto, raw: []byte("schema: tetra.test.v1\nstatus: pass\n")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got report
			if err := DecodeStructuredStrict(tc.raw, tc.format, &got); err != nil {
				t.Fatalf("DecodeStructuredStrict: %v", err)
			}
			if got.Schema != "tetra.test.v1" || got.Status != "pass" {
				t.Fatalf("decoded = %#v", got)
			}
		})
	}
}

func TestDecodeStructuredStrictRejectsUnknownFieldsAndUnsupportedFormat(t *testing.T) {
	type report struct {
		Schema string `json:"schema"`
	}
	var got report
	if err := DecodeStructuredStrict([]byte(`{"schema":"tetra.test.v1","extra":true}`), JSON, &got); err == nil {
		t.Fatalf("DecodeStructuredStrict accepted unknown field")
	}
	if err := DecodeStructuredStrict([]byte(`{"schema":"tetra.test.v1"}`), "yaml", &got); err == nil || !strings.Contains(err.Error(), `unsupported structured output format "yaml"`) {
		t.Fatalf("DecodeStructuredStrict unsupported error = %v", err)
	}
}
