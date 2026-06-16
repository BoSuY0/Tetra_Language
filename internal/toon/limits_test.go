package toon

import (
	"strings"
	"testing"
)

func TestEncodeRejectsDepthLimit(t *testing.T) {
	input := []byte(`{"outer":{"inner":{"value":1}}}`)
	_, err := ConvertJSONToTOON(input, Options{MaxDepth: 1})
	assertTOONErrorCode(t, err, CodeLimitDepth)
}

func TestDecodeRejectsObjectKeyLimit(t *testing.T) {
	input := []byte("a: 1\nb: 2")
	_, err := ConvertTOONToJSON(input, Options{MaxObjectKeys: 1})
	assertTOONErrorCode(t, err, CodeLimitObjectKeys)
}

func TestDecodeRejectsMalformedIndentation(t *testing.T) {
	input := []byte("outer:\n child: 1")
	_, err := ConvertTOONToJSON(input, Options{Strict: true})
	assertTOONErrorCode(t, err, CodeMalformedDocument)
}

func TestMarshalDeterministicAcrossRuns(t *testing.T) {
	input := map[string]any{
		"zeta":  1,
		"alpha": []any{"b", "a"},
		"users": []any{
			map[string]any{"name": "Ada", "id": 1},
			map[string]any{"name": "Linus", "id": 2},
		},
	}
	first, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if strings.HasSuffix(string(first), " ") || strings.HasSuffix(string(first), "\n") {
		t.Fatalf("TOON output has trailing whitespace/newline: %q", first)
	}
	for i := 0; i < 50; i++ {
		next, err := Marshal(input)
		if err != nil {
			t.Fatalf("Marshal run %d failed: %v", i, err)
		}
		if string(next) != string(first) {
			t.Fatalf("Marshal run %d was nondeterministic\ngot:\n%s\nwant:\n%s", i, next, first)
		}
	}
}
