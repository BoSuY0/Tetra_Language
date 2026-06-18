package compiler_test

import (
	"path/filepath"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

var benchmarkExampleCases = []string{
	"classic/clbg_fannkuch_redux.tetra",
	"classic/clbg_integer_mandelbrot.tetra",
	"classic/energy_languages_checksum.tetra",
	"classic/awfy_closure_dispatch.tetra",
	"classic/pyperformance_call_mix.tetra",
	"classic/rustc_perf_frontend_mix.tetra",
	"parallel/plb2_bedcov_scan.tetra",
	"parallel/plb2_matrix_multiply_i32.tetra",
	"parallel/plb2_nqueen.tetra",
	"parallel/plb2_sudoku_checksum.tetra",
	"parallel/pbbs_breadth_first_search.tetra",
	"parallel/pbbs_integer_sort.tetra",
	"systems/llvm_loop_unroll_kernel.tetra",
	"systems/polybench_jacobi_i32.tetra",
	"systems/nas_integer_cg.tetra",
	"systems/spec_cpu_branch_mix.tetra",
	"systems/techempower_plaintext_kernel.tetra",
	"jvm/jvm_dacapo_object_kernel.tetra",
	"jvm/jvm_renaissance_streams_i32.tetra",
}

func TestBenchmarkExamplesCompileAndRun(t *testing.T) {
	for _, name := range benchmarkExampleCases {
		t.Run(name, func(t *testing.T) {
			srcPath := testkit.RepoPath(t, "examples", "benchmarks", filepath.FromSlash(name))
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
			srcPath := testkit.RepoPath(b, "examples", "benchmarks", filepath.FromSlash(name))
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
			srcPath := testkit.RepoPath(b, "examples", "benchmarks", filepath.FromSlash(name))
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
