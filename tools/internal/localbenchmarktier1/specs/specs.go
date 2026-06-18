package specs

import (
	"path/filepath"
	"strings"
)

var RequiredCategories = []string{
	"integer loops",
	"slice sum",
	"bounds-check loops",
	"function calls",
	"recursion",
	"matrix multiply",
	"hash table",
	"allocation",
	"region/island allocation",
	"JSON parse/stringify",
	"HTTP plaintext/json",
	"PostgreSQL single/multiple/update",
	"actor ping-pong",
	"parallel map/reduce",
	"startup time",
	"binary size",
	"compile time",
}

var RequiredLanguages = []string{"tetra", "c", "cpp", "rust"}

type Spec struct {
	Name             string
	Category         string
	Language         string
	AlgorithmID      string
	InputDescription string
	BuildCommandKind string
	BuildArgs        []string
	SourceRelPath    string
	BinaryRelPath    string
	Source           string
}

func Build(outDir string) []Spec {
	var specs []Spec
	for _, category := range RequiredCategories {
		for _, language := range RequiredLanguages {
			name := slug(category) + "_" + language
			spec := Spec{
				Name:             name,
				Category:         category,
				Language:         language,
				AlgorithmID:      "p25.0." + slug(category),
				InputDescription: inputDescription(category),
				SourceRelPath: filepath.Join(
					outDir,
					"artifacts",
					"src",
					name+extensionFor(language),
				),
				BinaryRelPath: filepath.Join(outDir, "artifacts", "bin", name),
			}
			switch language {
			case "tetra":
				spec.BuildCommandKind = "tetra"
				spec.BuildArgs = []string{"tetra", "build", "--target", "linux-x64", "--explain"}
				if category != "actor ping-pong" {
					spec.SourceRelPath = filepath.Join(
						outDir,
						"artifacts",
						"src",
						"p25",
						slug(category)+".tetra",
					)
				}
				spec.Source = tetraSource(category)
			case "c":
				spec.BuildCommandKind = "clang"
				spec.BuildArgs = []string{"clang", "-O3"}
				spec.Source = cLikeSource(category)
			case "cpp":
				spec.BuildCommandKind = "clang++"
				spec.BuildArgs = []string{"clang++", "-O3"}
				spec.Source = cLikeSource(category)
			case "rust":
				spec.BuildCommandKind = "rustc"
				spec.BuildArgs = []string{"rustc", "-C", "opt-level=3"}
				spec.Source = rustSource(category)
			}
			specs = append(specs, spec)
		}
	}
	return specs
}

func extensionFor(language string) string {
	switch language {
	case "tetra":
		return ".tetra"
	case "c":
		return ".c"
	case "cpp":
		return ".cpp"
	case "rust":
		return ".rs"
	default:
		return ".txt"
	}
}

func slug(value string) string {
	replacer := strings.NewReplacer("/", "_", "-", "_")
	return strings.Join(strings.Fields(replacer.Replace(strings.ToLower(value))), "_")
}

func inputDescription(category string) string {
	return "deterministic P25.0 local Tier 1 " + category + (" workload with identical " +
		"intent across Tetra, C, C++, and Rust")
}
