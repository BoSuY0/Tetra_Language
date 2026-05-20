package compiler_test

import (
	"os"
	"path/filepath"
	"testing"

	compiler "tetra_language/compiler"
)

func BenchmarkCompileRepresentativeExamples(b *testing.B) {
	examples := map[string]string{
		"flow_hello":  filepath.Join("..", "..", "..", "examples", "flow_hello.tetra"),
		"core_math":   filepath.Join("..", "..", "..", "examples", "core_math_smoke.tetra"),
		"dogfood_cli": filepath.Join("..", "..", "..", "examples", "projects", "dogfood_cli", "src", "main.tetra"),
	}
	for name, src := range examples {
		b.Run(name, func(b *testing.B) {
			outDir := b.TempDir()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				out := filepath.Join(outDir, name)
				if err := compiler.BuildFile(src, out, "linux-x64"); err != nil {
					b.Fatalf("compiler.BuildFile: %v", err)
				}
			}
		})
	}
}

func BenchmarkFormatRepresentativeSources(b *testing.B) {
	sources := map[string]string{
		"flow_hello": filepath.Join("..", "..", "..", "examples", "flow_hello.tetra"),
		"web_ui":     filepath.Join("..", "..", "..", "examples", "projects", "dogfood_web_ui", "src", "main.tetra"),
	}
	for name, path := range sources {
		raw, err := os.ReadFile(path)
		if err != nil {
			b.Fatalf("read %s: %v", path, err)
		}
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := compiler.FormatSource(raw, path); err != nil {
					b.Fatalf("compiler.FormatSource: %v", err)
				}
			}
		})
	}
}

func BenchmarkGenerateAPIDocsDogfoodProjects(b *testing.B) {
	paths := []string{
		filepath.Join("..", "..", "..", "examples", "projects", "dogfood_cli"),
		filepath.Join("..", "..", "..", "examples", "projects", "dogfood_web_ui"),
	}
	for i := 0; i < b.N; i++ {
		if _, err := compiler.GenerateAPIDocs(paths); err != nil {
			b.Fatalf("compiler.GenerateAPIDocs: %v", err)
		}
	}
}

func BenchmarkBinarySizeBaselines(b *testing.B) {
	cases := []struct {
		name   string
		target string
		src    string
	}{
		{name: "linux_flow_hello", target: "linux-x64", src: filepath.Join("..", "..", "..", "examples", "flow_hello.tetra")},
		{name: "macos_flow_hello", target: "macos-x64", src: filepath.Join("..", "..", "..", "examples", "flow_hello.tetra")},
		{name: "windows_flow_hello", target: "windows-x64", src: filepath.Join("..", "..", "..", "examples", "flow_hello.tetra")},
		{name: "wasi_dogfood", target: "wasm32-wasi", src: filepath.Join("..", "..", "..", "examples", "projects", "dogfood_wasi", "src", "main.tetra")},
		{name: "web_dogfood", target: "wasm32-web", src: filepath.Join("..", "..", "..", "examples", "projects", "dogfood_web_ui", "src", "main.tetra")},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			outDir := b.TempDir()
			var lastSize int64
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				out := filepath.Join(outDir, tc.name)
				if err := compiler.BuildFile(tc.src, out, tc.target); err != nil {
					b.Fatalf("compiler.BuildFile: %v", err)
				}
				info, err := os.Stat(out)
				if err != nil {
					b.Fatalf("stat output: %v", err)
				}
				lastSize = info.Size()
			}
			b.ReportMetric(float64(lastSize), "artifact_bytes")
		})
	}
}
