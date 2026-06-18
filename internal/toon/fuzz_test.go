package toon

import (
	"encoding/json"
	"testing"
)

func FuzzConvertTOONToJSON(f *testing.F) {
	for _, seed := range []string{
		"name: Ada\nactive: true\nscore: 42",
		"items[2]{id,name}:\n  1,Ada\n  2,Linus",
		"nested:\n  message: \"hello: toon\"\n  tags[2]: compiler,tools",
		"empty_array: []\nempty_object:",
		"bad[2]{a,b}:\n  1",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		jsonData, err := ConvertTOONToJSON([]byte(input), Options{
			Strict:        true,
			MaxDepth:      32,
			MaxIndent:     64,
			MaxArrayLen:   1024,
			MaxObjectKeys: 1024,
		})
		if err != nil {
			return
		}
		if !json.Valid(jsonData) {
			t.Fatalf("decoder returned invalid JSON:\nTOON:\n%s\nJSON:\n%s", input, jsonData)
		}
		if _, err := ConvertJSONToTOON(jsonData, Options{Deterministic: true, Strict: true}); err != nil {
			t.Fatalf("decoded JSON cannot re-encode to TOON: %v\nJSON:\n%s", err, jsonData)
		}
	})
}
