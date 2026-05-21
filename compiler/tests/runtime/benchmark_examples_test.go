package compiler_test

import (
	"path/filepath"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

var benchmarkExampleCases = []string{
	"clbg_fannkuch_redux.tetra",
	"clbg_integer_mandelbrot.tetra",
	"energy_languages_checksum.tetra",
	"plb2_bedcov_scan.tetra",
	"plb2_matrix_multiply_i32.tetra",
	"plb2_nqueen.tetra",
	"plb2_sudoku_checksum.tetra",
	"pbbs_breadth_first_search.tetra",
	"pbbs_integer_sort.tetra",
	"awfy_closure_dispatch.tetra",
	"llvm_loop_unroll_kernel.tetra",
	"polybench_jacobi_i32.tetra",
	"nas_integer_cg.tetra",
	"spec_cpu_branch_mix.tetra",
	"techempower_plaintext_kernel.tetra",
	"jvm_dacapo_object_kernel.tetra",
	"jvm_renaissance_streams_i32.tetra",
	"pyperformance_call_mix.tetra",
	"rustc_perf_frontend_mix.tetra",
}

func TestBenchmarkExamplesCompileAndRun(t *testing.T) {
	for _, name := range benchmarkExampleCases {
		t.Run(name, func(t *testing.T) {
			srcPath := testkit.RepoPath(t, "examples", "benchmarks", name)
			outPath := filepath.Join(t.TempDir(), "bench")
			if err := compiler.BuildFile(srcPath, outPath, "linux-x64"); err != nil {
				t.Fatalf("BuildFile(%s): %v", name, err)
			}
			stdout, exitCode := testkit.RunBinary(t, outPath)
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0", exitCode)
			}
		})
	}
}

func BenchmarkBenchmarkExamplesBuild(b *testing.B) {
	for _, name := range benchmarkExampleCases {
		b.Run(name, func(b *testing.B) {
			srcPath := testkit.RepoPath(b, "examples", "benchmarks", name)
			for i := 0; i < b.N; i++ {
				outPath := filepath.Join(b.TempDir(), "bench")
				if err := compiler.BuildFile(srcPath, outPath, "linux-x64"); err != nil {
					b.Fatalf("BuildFile(%s): %v", name, err)
				}
			}
		})
	}
}

func BenchmarkBenchmarkExamplesRun(b *testing.B) {
	for _, name := range benchmarkExampleCases {
		b.Run(name, func(b *testing.B) {
			srcPath := testkit.RepoPath(b, "examples", "benchmarks", name)
			outPath := filepath.Join(b.TempDir(), "bench")
			if err := compiler.BuildFile(srcPath, outPath, "linux-x64"); err != nil {
				b.Fatalf("BuildFile(%s): %v", name, err)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				stdout, exitCode := testkit.RunBinary(b, outPath)
				if stdout != "" {
					b.Fatalf("stdout = %q, want empty", stdout)
				}
				if exitCode != 0 {
					b.Fatalf("exit code = %d, want 0", exitCode)
				}
			}
		})
	}
}
