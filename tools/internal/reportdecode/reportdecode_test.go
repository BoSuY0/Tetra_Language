package reportdecode

import (
	"strings"
	"testing"
)

type fixtureReport struct {
	Schema string `json:"schema"`
	Status string `json:"status"`
	Count  int    `json:"count"`
}

func TestDecodeStrictAcceptsJSONAndTOON(t *testing.T) {
	cases := []struct {
		name   string
		raw    []byte
		format string
	}{
		{
			name:   "auto json",
			raw:    []byte(`{"schema":"tetra.test.v1","status":"pass","count":2}`),
			format: "auto",
		},
		{
			name: "auto toon",
			raw: []byte(strings.Join([]string{
				"schema: tetra.test.v1",
				"status: pass",
				"count: 2",
			}, "\n")),
			format: "auto",
		},
		{
			name: "explicit toon",
			raw: []byte(strings.Join([]string{
				"schema: tetra.test.v1",
				"status: pass",
				"count: 2",
			}, "\n")),
			format: "toon",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got fixtureReport
			if err := DecodeStrictFormat(tc.raw, tc.format, &got); err != nil {
				t.Fatalf("DecodeStrictFormat: %v", err)
			}
			if got != (fixtureReport{Schema: "tetra.test.v1", Status: "pass", Count: 2}) {
				t.Fatalf("decoded report = %#v", got)
			}
		})
	}
}

func TestDecodeStrictRejectsUnknownFieldsAndTrailingJSON(t *testing.T) {
	cases := []struct {
		name string
		raw  []byte
	}{
		{
			name: "json unknown field",
			raw:  []byte(`{"schema":"tetra.test.v1","status":"pass","count":2,"extra":true}`),
		},
		{
			name: "toon unknown field",
			raw: []byte(strings.Join([]string{
				"schema: tetra.test.v1",
				"status: pass",
				"count: 2",
				"extra: true",
			}, "\n")),
		},
		{
			name: "trailing json",
			raw:  []byte(`{"schema":"tetra.test.v1","status":"pass","count":2} {}`),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got fixtureReport
			if err := DecodeStrict(tc.raw, &got); err == nil {
				t.Fatalf("DecodeStrict accepted invalid report")
			}
		})
	}
}

func TestDecodeStrictFormatRejectsUnsupportedFormat(t *testing.T) {
	var got fixtureReport
	err := DecodeStrictFormat([]byte(`{"schema":"tetra.test.v1","status":"pass","count":2}`), "yaml", &got)
	if err == nil || !strings.Contains(err.Error(), `unsupported report format "yaml"`) {
		t.Fatalf("DecodeStrictFormat unsupported error = %v", err)
	}
}
