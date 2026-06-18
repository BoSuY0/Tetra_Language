package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/tools/validators/surface"
)

type morphRenderedBeautyContract = surface.MorphRenderedBeautyContract
type morphRenderedBeautyNegativeGuards = surface.MorphRenderedBeautyNegativeGuards
type morphRenderedBeautyReport = surface.MorphRenderedBeautyReport
type morphRenderedBeautyMorphEvidence = surface.MorphRenderedBeautyMorphEvidence
type morphRenderedBeautyBlockSceneSnapshot = surface.MorphRenderedBeautyBlockSceneSnapshot
type morphRenderedBeautyBlockSceneSpecCoverage = surface.MorphRenderedBeautyBlockSceneSpecCoverage
type morphRenderedBeautyRenderEvidence = surface.MorphRenderedBeautyRenderEvidence
type morphRenderedBeautyRendererStableProof = surface.MorphRenderedBeautyRendererStableProof
type morphRenderedBeautyRenderCommandStream = surface.MorphRenderedBeautyRenderCommandStream
type morphRenderedBeautyRenderCommand = surface.MorphRenderedBeautyRenderCommand
type morphRenderedBeautyPixelEvidence = surface.MorphRenderedBeautyPixelEvidence

func main() {
	contractPath := flag.String(
		"contract",
		"",
		"path to tetra.surface.morph-rendered-beauty.contract.v1 JSON",
	)
	reportPath := flag.String(
		"report",
		"",
		"optional path to tetra.surface.morph-rendered-beauty.v1 report JSON",
	)
	morphToPixelsChainOut := flag.String(
		"morph-to-pixels-chain-out",
		"",
		"optional path to write a validated Morph-to-pixels chain summary from --report",
	)
	flag.Parse()
	if strings.TrimSpace(*contractPath) == "" && strings.TrimSpace(*reportPath) == "" {
		fmt.Fprintln(os.Stderr, "error: --contract or --report is required")
		os.Exit(2)
	}
	if strings.TrimSpace(*contractPath) != "" {
		if err := validateMorphRenderedBeautyContractFile(*contractPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if strings.TrimSpace(*reportPath) != "" {
		report, err := readMorphRenderedBeautyReportFile(*reportPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := validateMorphRenderedBeautyReportValue(report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if strings.TrimSpace(*morphToPixelsChainOut) != "" {
			if err := writeMorphToPixelsChainFile(*reportPath, *morphToPixelsChainOut, report); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
}

func writeMorphToPixelsChainFile(
	reportPath string,
	outPath string,
	report surface.MorphRenderedBeautyReport,
) error {
	chain := surface.MorphToPixelsChainFromRenderedBeauty(filepath.ToSlash(reportPath), report)
	if err := surface.ValidateMorphToPixelsChainReport(
		chain,
		report.MorphEvidence.Source,
	); err != nil {
		return err
	}
	raw, err := json.Marshal(chain)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, append(raw, '\n'), 0o644)
}

func readMorphRenderedBeautyReportFile(path string) (surface.MorphRenderedBeautyReport, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return surface.MorphRenderedBeautyReport{}, err
	}
	var report surface.MorphRenderedBeautyReport
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return surface.MorphRenderedBeautyReport{}, err
	}
	return report, nil
}

func validateMorphRenderedBeautyContractFile(path string) error {
	return surface.ValidateMorphRenderedBeautyContractFile(path)
}

func validateMorphRenderedBeautyReportFile(path string) error {
	return surface.ValidateMorphRenderedBeautyReportFile(path)
}

func validateMorphRenderedBeautyContractValue(contract morphRenderedBeautyContract) error {
	return surface.ValidateMorphRenderedBeautyContractValue(contract)
}

func validateMorphRenderedBeautyReportValue(report morphRenderedBeautyReport) error {
	return surface.ValidateMorphRenderedBeautyReportValue(report)
}
