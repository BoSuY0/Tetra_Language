# v1.0 Performance Regression Report

- schema: `tetra.performance-regression.v1`
- command: `go test ./compiler -bench='Benchmark(CompileRepresentativeExamples|FormatRepresentativeSources|GenerateAPIDocsDogfoodProjects|BinarySizeBaselines)' -run '^$' -count=1`
- host: `linux amd64 Intel(R) Core(TM) i9-14900HX`
- threshold decision: accepted as current branch baseline capture; compare future RCs with `benchstat` before release promotion
- JSON artifact: `docs/generated/v1_0/performance-regression.json`
- summary.metric_count: `11`
- summary.total_iterations: `370162`
- summary.max_ns_per_op: `220157`
- summary.metrics_sha256: `sha256:8be60f159e4324422a8a0b02a735611dc748a12190742670a98b459d7c72cfa0`

| Metric | Iterations | ns/op | Artifact bytes | Decision |
| --- | ---: | ---: | ---: | --- |
| BenchmarkCompileRepresentativeExamples/core_math-32 | 9121 | 151356 |  | accepted baseline capture |
| BenchmarkCompileRepresentativeExamples/dogfood_cli-32 | 5614 | 220157 |  | accepted baseline capture |
| BenchmarkCompileRepresentativeExamples/flow_hello-32 | 12846 | 96690 |  | accepted baseline capture |
| BenchmarkFormatRepresentativeSources/flow_hello-32 | 218643 | 8025 |  | accepted baseline capture |
| BenchmarkFormatRepresentativeSources/web_ui-32 | 40100 | 27446 |  | accepted baseline capture |
| BenchmarkGenerateAPIDocsDogfoodProjects-32 | 16438 | 81948 |  | accepted baseline capture |
| BenchmarkBinarySizeBaselines/linux_flow_hello-32 | 15618 | 102192 | 4113 | accepted baseline capture |
| BenchmarkBinarySizeBaselines/macos_flow_hello-32 | 11173 | 120743 | 12288 | accepted baseline capture |
| BenchmarkBinarySizeBaselines/windows_flow_hello-32 | 11812 | 89155 | 3072 | accepted baseline capture |
| BenchmarkBinarySizeBaselines/wasi_dogfood-32 | 21150 | 71206 | 222 | accepted baseline capture |
| BenchmarkBinarySizeBaselines/web_dogfood-32 | 7647 | 143250 | 120 | accepted baseline capture |

Residual risk: this is single-host `-count=1` evidence. Release candidates
should rerun the threshold command with `-count=5` and compare against the prior
accepted artifact with `benchstat`.
