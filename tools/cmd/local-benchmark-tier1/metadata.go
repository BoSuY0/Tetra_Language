package main

import (
	"encoding/json"
	"os"
)

func collectTetraMetadata(name string, binaryPath string, optimizerArtifact string) *tetraMetadata {
	proof := binaryPath + ".proof.json"
	bounds := binaryPath + ".bounds.json"
	alloc := binaryPath + ".alloc.json"
	perf := binaryPath + ".perf.json"
	backend := binaryPath + ".backend.json"
	boundsLeft := readBoundsLeft(bounds)
	heap := readHeapAllocations(alloc)
	return &tetraMetadata{
		ProofReport:       proof,
		BoundsReport:      bounds,
		AllocationReport:  alloc,
		PerfBlockerReport: perf,
		BackendReport:     backend,
		BackendPath:       readBackendPath(backend),
		BoundsLeft:        boundsLeft,
		HeapAllocations:   heap,
		PerfBlockers:      readPerfBlockers(perf, name),
		OptimizerValidationMetadata: optimizerValidation{
			Status:   "current_supported_subset",
			Artifact: optimizerArtifact,
		},
	}
}

func missingTetraMetadata(binaryPath string, optimizerArtifact string) *tetraMetadata {
	return &tetraMetadata{
		ProofReport:       binaryPath + ".proof.json",
		BoundsReport:      binaryPath + ".bounds.json",
		AllocationReport:  binaryPath + ".alloc.json",
		PerfBlockerReport: binaryPath + ".perf.json",
		BackendReport:     binaryPath + ".backend.json",
		BackendPath:       "fallback",
		OptimizerValidationMetadata: optimizerValidation{
			Status:   "missing_build_artifacts",
			Artifact: optimizerArtifact,
		},
	}
}

func readBoundsLeft(path string) int {
	var report struct {
		Totals struct {
			Left int `json:"left"`
		} `json:"totals"`
	}
	if readJSON(path, &report) != nil {
		return 0
	}
	return report.Totals.Left
}

func readHeapAllocations(path string) int {
	var report struct {
		Totals struct {
			Heap int `json:"heap"`
		} `json:"totals"`
	}
	if readJSON(path, &report) != nil {
		return 0
	}
	return report.Totals.Heap
}

func readBackendPath(path string) string {
	var report struct {
		Summary struct {
			RegisterPath  int `json:"register_path"`
			StackFallback int `json:"stack_fallback"`
		} `json:"summary"`
	}
	if readJSON(path, &report) != nil {
		return "fallback"
	}
	if report.Summary.StackFallback > 0 {
		return "fallback"
	}
	if report.Summary.RegisterPath > 0 {
		return "register"
	}
	return "stack"
}

func readPerfBlockers(path string, benchmark string) []string {
	var report struct {
		Benchmarks []struct {
			Benchmark   string   `json:"benchmark"`
			ReasonCodes []string `json:"reason_codes"`
		} `json:"benchmarks"`
	}
	if readJSON(path, &report) != nil {
		return nil
	}
	for _, row := range report.Benchmarks {
		if row.Benchmark == benchmark {
			return append([]string(nil), row.ReasonCodes...)
		}
	}
	return nil
}

func readJSON(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}
