package main

import "fmt"

func classifyCategory(category string, rows []benchmarkRow, threshold float64) (string, string) {
	tetra, ok := rowForLanguage(rows, "tetra")
	if !ok || tetra.Status != "measured" {
		return "blocked by missing feature", "Tetra did not produce a measured local row for this category."
	}
	if category == "binary size" {
		return classifyBinarySize(rows)
	}
	if category == "compile time" {
		return classifyCompileTime(rows, threshold)
	}
	if category == "actor ping-pong" || category == "parallel map/reduce" {
		return "blocked by actor/runtime limitation", "Current local actor/task runtime evidence is bounded and not a production parallel benchmark claim."
	}
	if category == "HTTP plaintext/json" || category == "PostgreSQL single/multiple/update" || category == "JSON parse/stringify" {
		return "invalid/inconclusive", "This Tier 1 run measures deterministic local helper kernels, not a full local service/database benchmark for this category."
	}
	if tetra.TetraMetadata != nil {
		if heapSensitiveCategory(category) && tetra.TetraMetadata.HeapAllocations > 0 {
			return "blocked by heap allocation", fmt.Sprintf("Tetra allocation report records %d heap allocations.", tetra.TetraMetadata.HeapAllocations)
		}
		if boundsSensitiveCategory(category) && tetra.TetraMetadata.BoundsLeft > 0 {
			return "blocked by bounds check", fmt.Sprintf("Tetra bounds report records %d bounds checks left.", tetra.TetraMetadata.BoundsLeft)
		}
		if tetra.TetraMetadata.BackendPath == "fallback" || tetra.TetraMetadata.BackendPath == "stack" {
			return "blocked by fallback backend", "Tetra backend report selected stack/fallback path for at least one function."
		}
	}
	competitors := measuredCompetitorMedians(rows)
	if len(competitors) != 3 || tetra.MedianRuntimeMS <= 0 {
		return "invalid/inconclusive", "One or more competitor rows did not produce measured local timing."
	}
	fastest := competitors[0]
	for _, value := range competitors[1:] {
		if value < fastest {
			fastest = value
		}
	}
	if tetra.MedianRuntimeMS < fastest*(1-threshold) {
		return "faster than C/C++/Rust locally", fmt.Sprintf("Tetra median %.3f ms is more than %.0f%% below the fastest local competitor median %.3f ms.", tetra.MedianRuntimeMS, threshold*100, fastest)
	}
	if tetra.MedianRuntimeMS <= fastest*(1+threshold) {
		return "comparable", fmt.Sprintf("Tetra median %.3f ms is within %.0f%% of the fastest local competitor median %.3f ms.", tetra.MedianRuntimeMS, threshold*100, fastest)
	}
	return "slower", fmt.Sprintf("Tetra median %.3f ms is more than %.0f%% above the fastest local competitor median %.3f ms.", tetra.MedianRuntimeMS, threshold*100, fastest)
}

func classifyBinarySize(rows []benchmarkRow) (string, string) {
	tetra, ok := rowForLanguage(rows, "tetra")
	if !ok || tetra.BinarySizeBytes <= 0 {
		return "invalid/inconclusive", "Tetra binary_size_bytes is missing for binary-size category."
	}
	sizes := map[string]int64{}
	for _, language := range []string{"c", "cpp", "rust"} {
		row, ok := rowForLanguage(rows, language)
		if !ok || row.BinarySizeBytes <= 0 {
			return "invalid/inconclusive", "One or more competitor binary_size_bytes values are missing for binary-size category."
		}
		sizes[language] = row.BinarySizeBytes
	}
	return "comparable", fmt.Sprintf("binary_size_bytes local evidence: Tetra=%d, C=%d, C++=%d, Rust=%d; no binary-size superiority or production-size claim is promoted.", tetra.BinarySizeBytes, sizes["c"], sizes["cpp"], sizes["rust"])
}

func classifyCompileTime(rows []benchmarkRow, threshold float64) (string, string) {
	tetra, ok := rowForLanguage(rows, "tetra")
	if !ok || tetra.CompileTimeMS <= 0 {
		return "invalid/inconclusive", "Tetra compile_time_ms is missing for compile-time category."
	}
	var competitors []float64
	for _, language := range []string{"c", "cpp", "rust"} {
		row, ok := rowForLanguage(rows, language)
		if !ok || row.CompileTimeMS <= 0 {
			return "invalid/inconclusive", "One or more competitor compile_time_ms values are missing for compile-time category."
		}
		competitors = append(competitors, row.CompileTimeMS)
	}
	fastest := competitors[0]
	for _, value := range competitors[1:] {
		if value < fastest {
			fastest = value
		}
	}
	if tetra.CompileTimeMS < fastest*(1-threshold) {
		return "faster than C/C++/Rust locally", fmt.Sprintf("Tetra compile_time_ms %.3f is more than %.0f%% below the fastest local competitor compile_time_ms %.3f.", tetra.CompileTimeMS, threshold*100, fastest)
	}
	if tetra.CompileTimeMS <= fastest*(1+threshold) {
		return "comparable", fmt.Sprintf("Tetra compile_time_ms %.3f is within %.0f%% of the fastest local competitor compile_time_ms %.3f.", tetra.CompileTimeMS, threshold*100, fastest)
	}
	return "slower", fmt.Sprintf("Tetra compile_time_ms %.3f is more than %.0f%% above the fastest local competitor compile_time_ms %.3f.", tetra.CompileTimeMS, threshold*100, fastest)
}
