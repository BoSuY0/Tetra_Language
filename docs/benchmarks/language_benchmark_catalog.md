# Language Benchmark Catalog

Status: research catalog and local implementation map.

The internet benchmark landscape is too large to vendor wholesale into this
repository. This catalog records the major public benchmark families found for
programming languages and maps each one to a runnable Tetra kernel under
`examples/benchmarks/` when the current language surface can represent the core
shape.

| Benchmark family | Public source | Local Tetra status |
| --- | --- | --- |
| Computer Language Benchmarks Game | https://benchmarksgame-team.pages.debian.net/benchmarksgame/ | `clbg_fannkuch_redux.tetra`, `clbg_integer_mandelbrot.tetra` |
| Programming-Language-Benchmarks | https://programming-language-benchmarks.vercel.app/ | Shares the CLBG-compatible kernels above |
| Energy-Languages | https://github.com/greensoftwarelab/Energy-Languages | `energy_languages_checksum.tetra` |
| PLB2 | https://github.com/attractivechaos/plb2 | `plb2_bedcov_scan.tetra`, `plb2_matrix_multiply_i32.tetra`, `plb2_nqueen.tetra`, `plb2_sudoku_checksum.tetra` |
| PBBS | https://cmuparlay.github.io/pbbsbench/ | `pbbs_breadth_first_search.tetra`, `pbbs_integer_sort.tetra` |
| Are We Fast Yet | https://github.com/smarr/are-we-fast-yet | `awfy_closure_dispatch.tetra` |
| LLVM test-suite | https://llvm.org/docs/TestSuiteGuide.html | `llvm_loop_unroll_kernel.tetra` |
| PolyBench/C | https://www.cs.colostate.edu/~pouchet/software/polybench/polybench.html | `polybench_jacobi_i32.tetra` |
| NAS Parallel Benchmarks | https://www.nas.nasa.gov/software/npb.html | `nas_integer_cg.tetra` |
| SPEC CPU | https://spec.cs.miami.edu/cpu2026/docs/overview.html | `spec_cpu_branch_mix.tetra` as a non-SPEC integer proxy |
| TechEmpower Framework Benchmarks | https://github.com/TechEmpower/FrameworkBenchmarks | `techempower_plaintext_kernel.tetra` remains the current-surface language proxy; `docs/benchmarks/techempower_web_stack.md` tracks the real local TCP/HTTP/PostgreSQL stack and benchmark app |
| DaCapo | https://www.dacapobench.org/ | `jvm_dacapo_object_kernel.tetra` |
| Renaissance | https://github.com/renaissance-benchmarks/renaissance | `jvm_renaissance_streams_i32.tetra` |
| SPECjvm2008 | https://www.spec.org/osg/jvm2008/ | Covered only as a JVM-family reference; no official local port |
| pyperformance | https://pyperformance.readthedocs.io/ | `pyperformance_call_mix.tetra` |
| rustc-perf | https://rustc-dev-guide.rust-lang.org/tests/perf.html | `rustc_perf_frontend_mix.tetra` |

## Scope Rules

- The local files are deterministic kernels, not upstream-certified benchmark
  submissions.
- Floating-point, JVM, Python, Rust, C, C++, and Fortran-specific workloads are
  represented by current-surface proxies until Tetra grows the required runtime
  and numeric features. TechEmpower is separately tracked as a local
  HTTP/PostgreSQL runtime stack, not as an official upstream publication.
- Each runnable kernel returns exit code `0` on its expected checksum and is
  covered by `TestBenchmarkExamplesCompileAndRun`.
