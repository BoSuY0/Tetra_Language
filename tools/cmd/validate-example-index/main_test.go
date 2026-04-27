package main

import (
	"strings"
	"testing"
)

func TestValidateExampleIndexAcceptsDocumentedSmokeCase(t *testing.T) {
	smoke := []byte(`{"cases":[{"name":"flow_hello","src_path":"examples/flow_hello.tetra","target_group":"native","expected_exit":0}]}`)
	index := strings.Join([]string{
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
		"| `examples/flow_hello.tetra` | Minimal Flow build sanity check. | native | exits 0 |",
	}, "\n")
	if err := validateExampleIndex(smoke, index); err != nil {
		t.Fatalf("validateExampleIndex: %v", err)
	}
}

func TestValidateExampleIndexRejectsMissingSmokeCase(t *testing.T) {
	smoke := []byte(`{"cases":[{"name":"flow_hello","src_path":"examples/flow_hello.tetra","target_group":"native","expected_exit":0}]}`)
	index := strings.Join([]string{
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
		"| `examples/other.tetra` | Other example. | native | exits 0 |",
	}, "\n")
	err := validateExampleIndex(smoke, index)
	if err == nil {
		t.Fatalf("expected missing index entry failure")
	}
	if !strings.Contains(err.Error(), "missing examples/flow_hello.tetra") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateExampleIndexRejectsMissingExpectedBehavior(t *testing.T) {
	smoke := []byte(`{"cases":[{"name":"flow_hello","src_path":"examples/flow_hello.tetra","target_group":"native","expected_exit":42}]}`)
	index := strings.Join([]string{
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
		"| `examples/flow_hello.tetra` | Minimal Flow build sanity check. | native | returns successfully |",
	}, "\n")
	err := validateExampleIndex(smoke, index)
	if err == nil {
		t.Fatalf("expected missing expected behavior failure")
	}
	if !strings.Contains(err.Error(), "exit 42") {
		t.Fatalf("unexpected error: %v", err)
	}
}
