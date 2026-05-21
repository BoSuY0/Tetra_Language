# Tetra Benchmark Corpus

Status: runnable scale-down benchmark kernels for the current local Tetra
compiler surface.

These programs are not official submissions to the upstream benchmark suites.
They are small, deterministic Tetra ports inspired by common language benchmark
families, chosen to compile and run on the current `v0.4.0` surface. Each
program returns exit code `0` when its checksum matches the expected result.

Run the corpus contract with:

```sh
go test ./compiler/tests/runtime -run TestBenchmarkExamplesCompileAndRun -count=1
```

Run Go benchmark timing over the existing compiler performance suite with:

```sh
go test ./compiler/tests/runtime -run '^$' -bench='Benchmark(CompileRepresentativeExamples|FormatRepresentativeSources|GenerateAPIDocsDogfoodProjects|BinarySizeBaselines)' -count=5
```

## Upstream Coverage Map

| Source family | Tetra kernel |
| --- | --- |
| Computer Language Benchmarks Game | `clbg_fannkuch_redux.tetra`, `clbg_integer_mandelbrot.tetra` |
| Energy-Languages | `energy_languages_checksum.tetra` |
| PLB2 | `plb2_bedcov_scan.tetra`, `plb2_matrix_multiply_i32.tetra`, `plb2_nqueen.tetra`, `plb2_sudoku_checksum.tetra` |
| PBBS | `pbbs_breadth_first_search.tetra`, `pbbs_integer_sort.tetra` |
| Are We Fast Yet | `awfy_closure_dispatch.tetra` |
| LLVM test-suite style kernels | `llvm_loop_unroll_kernel.tetra` |
| PolyBench | `polybench_jacobi_i32.tetra` |
| NAS Parallel Benchmarks | `nas_integer_cg.tetra` |
| SPEC CPU style integer mix | `spec_cpu_branch_mix.tetra` |
| TechEmpower plaintext path | `techempower_plaintext_kernel.tetra` |
| JVM suites | `jvm_dacapo_object_kernel.tetra`, `jvm_renaissance_streams_i32.tetra` |
| Python/Rust perf suites | `pyperformance_call_mix.tetra`, `rustc_perf_frontend_mix.tetra` |

## TechEmpower Web Stack

The runnable HTTP/PostgreSQL benchmark app lives outside this scale-down
language-kernel corpus at `benchmarks/techempower/tetra/`. See
`docs/benchmarks/techempower_web_stack.md` for endpoint coverage, local run
commands, and current upstream-submission limitations.
