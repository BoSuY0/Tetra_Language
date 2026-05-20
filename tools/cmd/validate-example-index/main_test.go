package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestValidateExampleIndexAcceptsDocumentedSmokeCase(t *testing.T) {
	smoke := []byte(`{"total":1,"cases":[{"name":"flow_hello","src_path":"examples/flow_hello.tetra","target_group":"native","expected_exit":0}]}`)
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
	smoke := []byte(`{"total":1,"cases":[{"name":"flow_hello","src_path":"examples/flow_hello.tetra","target_group":"native","expected_exit":0}]}`)
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
	smoke := []byte(`{"total":1,"cases":[{"name":"flow_hello","src_path":"examples/flow_hello.tetra","target_group":"native","expected_exit":42}]}`)
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

func TestValidateExampleIndexRejectsUnknownSmokeFields(t *testing.T) {
	smoke := []byte(`{"total":1,"cases":[{"name":"flow_hello","src_path":"examples/flow_hello.tetra","target_group":"native","expected_exit":0}],"extra":true}`)
	index := strings.Join([]string{
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
		"| `examples/flow_hello.tetra` | Minimal Flow build sanity check. | native | exits 0 |",
	}, "\n")
	err := validateExampleIndex(smoke, index)
	if err == nil {
		t.Fatalf("expected strict JSON unknown field failure")
	}
	if !strings.Contains(err.Error(), "invalid smoke list JSON") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateExampleIndexRejectsWindowsStylePath(t *testing.T) {
	smoke := []byte(`{"total":1,"cases":[{"name":"flow_hello","src_path":"examples\\flow_hello.tetra","target_group":"native","expected_exit":0}]}`)
	index := strings.Join([]string{
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
		"| `examples/flow_hello.tetra` | Minimal Flow build sanity check. | native | exits 0 |",
	}, "\n")
	err := validateExampleIndex(smoke, index)
	if err == nil {
		t.Fatalf("expected portability path failure")
	}
	if !strings.Contains(err.Error(), "forward slashes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateExampleIndexAcceptsExcludedExamples(t *testing.T) {
	smoke := []byte(`{
  "total":1,
  "cases":[{"name":"flow_hello","src_path":"examples/flow_hello.tetra","target_group":"native","expected_exit":0}],
  "excluded_examples":[{"src_path":"examples/hello.tetra","reason":"legacy profile exclusion"}]
}`)
	index := strings.Join([]string{
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
		"| `examples/flow_hello.tetra` | Minimal Flow build sanity check. | native | exits 0 |",
		"| `examples/hello.tetra` | Legacy hello world. | native | exits 0 (excluded from linux-x64 smoke profile) |",
	}, "\n")
	if err := validateExampleIndex(smoke, index); err != nil {
		t.Fatalf("validateExampleIndex with exclusion: %v", err)
	}
}

func TestValidateExampleDocsAcceptsT4ProjectEntry(t *testing.T) {
	index := strings.Join([]string{
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
		"| `examples/projects/hello_t4/src/main.t4` | Minimal project-first app. | native | exits 0 |",
	}, "\n")
	if err := validateExampleDocs(index); err != nil {
		t.Fatalf("validateExampleDocs: %v", err)
	}
}

func TestValidateExampleDocsRejectsUnsupportedExtension(t *testing.T) {
	index := strings.Join([]string{
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
		"| `examples/not_an_example.txt` | Invalid extension. | native | exits 0 |",
	}, "\n")
	err := validateExampleDocs(index)
	if err == nil {
		t.Fatalf("expected unsupported extension failure")
	}
	if !strings.Contains(err.Error(), "must point to a .tetra or .t4 file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunAcceptsDocsFlagWithoutSmokeList(t *testing.T) {
	index := strings.Join([]string{
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
		"| `examples/projects/hello_t4/src/main.t4` | Minimal project-first app. | native | exits 0 |",
	}, "\n")
	docsPath := t.TempDir() + "/examples_index.md"
	if err := os.WriteFile(docsPath, []byte(index), 0o644); err != nil {
		t.Fatalf("write docs fixture: %v", err)
	}

	var stderr bytes.Buffer
	exitCode := runValidateExampleIndex([]string{"--docs", docsPath}, io.Discard, &stderr)
	if exitCode != 0 {
		t.Fatalf("runValidateExampleIndex exit = %d, stderr = %q", exitCode, stderr.String())
	}
}
