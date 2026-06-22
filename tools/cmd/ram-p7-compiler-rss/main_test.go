package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultOutDirUsesP7CompilerRSSPrefix(t *testing.T) {
	got := defaultOutDir("0123456789abcdef0123456789abcdef01234567")
	want := filepath.Join("reports", "stabilization", "tetra-ram-p7-compiler-rss-0123456789ab")
	if got != want {
		t.Fatalf("defaultOutDir = %q, want %q", got, want)
	}
}

func TestReproducibleCommandIncludesOutDirOnlyWhenProvided(t *testing.T) {
	got := reproducibleCommand("reports/stabilization/tetra-ram-p7-compiler-rss-test", true, 5, true, "p7_5", true)
	want := []string{
		"go",
		"run",
		"./tools/cmd/ram-p7-compiler-rss",
		"--out-dir",
		"reports/stabilization/tetra-ram-p7-compiler-rss-test",
		"--samples",
		"5",
		"--matrix",
		"p7_5",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("provided command = %#v, want %#v", got, want)
	}

	got = reproducibleCommand("ignored", false, 1, false, "default", false)
	want = []string{"go", "run", "./tools/cmd/ram-p7-compiler-rss"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default command = %#v, want %#v", got, want)
	}
}

func TestComparisonOutPathDefaultsInsideCandidateBundle(t *testing.T) {
	got := comparisonOutPath("reports/stabilization/candidate", "")
	want := filepath.Join("reports", "stabilization", "candidate", "baseline-candidate-comparison.json")
	if got != want {
		t.Fatalf("comparisonOutPath = %q, want %q", got, want)
	}

	got = comparisonOutPath("ignored", "reports/stabilization/comparison.json")
	if got != "reports/stabilization/comparison.json" {
		t.Fatalf("explicit comparisonOutPath = %q", got)
	}
}

func TestReproducibleComparisonCommand(t *testing.T) {
	got := reproducibleComparisonCommand(
		"reports/stabilization/baseline",
		"reports/stabilization/candidate",
		"reports/stabilization/candidate/baseline-candidate-comparison.json",
	)
	want := []string{
		"go",
		"run",
		"./tools/cmd/ram-p7-compiler-rss",
		"--compare-baseline-dir",
		"reports/stabilization/baseline",
		"--compare-candidate-dir",
		"reports/stabilization/candidate",
		"--compare-out",
		"reports/stabilization/candidate/baseline-candidate-comparison.json",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("comparison command = %#v, want %#v", got, want)
	}
}
