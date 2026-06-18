package toon

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestExampleFixturesMatchCanonicalJSON(t *testing.T) {
	for _, name := range []string{"diagnostic", "test-report", "release-summary"} {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join("..", "..", "examples", "toon")
			jsonData, err := os.ReadFile(filepath.Join(root, name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			toonData, err := os.ReadFile(filepath.Join(root, name+".toon"))
			if err != nil {
				t.Fatal(err)
			}

			convertedJSON, err := ConvertTOONToJSON(toonData, Options{Strict: true})
			if err != nil {
				t.Fatalf("convert fixture TOON to JSON: %v\n%s", err, toonData)
			}
			assertJSONEqual(t, convertedJSON, jsonData)

			regeneratedTOON, err := ConvertJSONToTOON(jsonData, Options{Deterministic: true, Strict: true})
			if err != nil {
				t.Fatalf("convert fixture JSON to TOON: %v", err)
			}
			golden := bytes.TrimSuffix(toonData, []byte("\n"))
			if !bytes.Equal(regeneratedTOON, golden) {
				t.Fatalf("TOON golden drift\ngot:\n%s\nwant:\n%s", regeneratedTOON, golden)
			}
		})
	}
}
