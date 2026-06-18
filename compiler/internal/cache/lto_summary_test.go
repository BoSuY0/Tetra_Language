package cache

import (
	"fmt"
	"strings"
	"testing"

	"tetra_language/compiler/internal/semantics"
)

func TestIncrementalModuleSummaryV1RecordsDependencyHashContractAndRejectsConsumers(t *testing.T) {
	sigs := map[string]semantics.FuncSig{
		"math.core.add": {ParamTypes: []string{"i32", "i32"}, ReturnType: "i32", Public: true},
	}
	typeSigs := map[string]string{
		"math.core.Vec": "struct{x:i32,y:i32}",
	}
	depHash1, err := DepSigHashFromDepsWithInterfaceHashes(
		[]string{"math.core.add"},
		[]string{"math.core.Vec"},
		sigs,
		typeSigs,
		map[string]string{"math.core": "sha256:1111"},
	)
	if err != nil {
		t.Fatalf("DepSigHashFromDepsWithInterfaceHashes hash1: %v", err)
	}
	depHash2, err := DepSigHashFromDepsWithInterfaceHashes(
		[]string{"math.core.add"},
		[]string{"math.core.Vec"},
		sigs,
		typeSigs,
		map[string]string{"math.core": "sha256:2222"},
	)
	if err != nil {
		t.Fatalf("DepSigHashFromDepsWithInterfaceHashes hash2: %v", err)
	}
	if depHash1 == depHash2 {
		t.Fatalf("interface hash drift did not affect dependency hash")
	}

	summary, err := BuildIncrementalModuleSummary(IncrementalModuleSummaryInput{
		Module:           "app.main",
		Target:           "linux-x64",
		BuildTag:         "alloc-stack-v1",
		Source:           []byte("module app.main\n"),
		DependencyHash:   depHash1,
		PublicAPIHash:    "sha256:api1111",
		ExternalCallees:  []string{"math.core.add"},
		ExternalTypeDeps: []string{"math.core.Vec"},
	})
	if err != nil {
		t.Fatalf("BuildIncrementalModuleSummary: %v", err)
	}
	if summary.SchemaVersion != IncrementalModuleSummarySchemaVersion {
		t.Fatalf("schema = %q", summary.SchemaVersion)
	}
	if summary.Module != "app.main" || summary.Target != "linux-x64" ||
		summary.BuildTag != "alloc-stack-v1" {
		t.Fatalf("summary identity = %#v", summary)
	}
	if !strings.HasPrefix(summary.SourceHash, "sha256:") {
		t.Fatalf("source hash = %q", summary.SourceHash)
	}
	if summary.DependencyHash != fmt.Sprintf("sha256:%x", depHash1) {
		t.Fatalf("dependency hash = %q, want hash1", summary.DependencyHash)
	}
	if summary.DependencyHash == fmt.Sprintf("sha256:%x", depHash2) {
		t.Fatalf("summary dependency hash ignored interface hash drift")
	}
	if summary.CodegenConsumer || summary.LinkerConsumer {
		t.Fatalf("summary must be non-consumer evidence only: %#v", summary)
	}
	for _, row := range []string{
		"source_hash",
		"dependency_hash_contract",
		"public_api_hash",
		"cross_module_signature_inputs",
		"non_consumer_boundary",
	} {
		if !containsString(summary.ValidationRows, row) {
			t.Fatalf("summary missing validation row %q: %#v", row, summary.ValidationRows)
		}
	}

	encoded, err := MarshalIncrementalModuleSummary(summary)
	if err != nil {
		t.Fatalf("MarshalIncrementalModuleSummary: %v", err)
	}
	decoded, err := ParseIncrementalModuleSummary(encoded)
	if err != nil {
		t.Fatalf("ParseIncrementalModuleSummary: %v", err)
	}
	reencoded, err := MarshalIncrementalModuleSummary(decoded)
	if err != nil {
		t.Fatalf("MarshalIncrementalModuleSummary(decoded): %v", err)
	}
	if string(encoded) != string(reencoded) {
		t.Fatalf(
			"summary JSON not canonical:\n got %s\nwant %s",
			string(reencoded),
			string(encoded),
		)
	}

	for name, mutate := range map[string]func(IncrementalModuleSummary) IncrementalModuleSummary{
		"wrong schema": func(s IncrementalModuleSummary) IncrementalModuleSummary {
			s.SchemaVersion = "tetra.incremental.module_summary.v2"
			return s
		},
		"missing dependency hash": func(s IncrementalModuleSummary) IncrementalModuleSummary {
			s.DependencyHash = ""
			return s
		},
		"missing dependency validation row": func(s IncrementalModuleSummary) IncrementalModuleSummary {
			s.ValidationRows = []string{
				"source_hash",
				"public_api_hash",
				"cross_module_signature_inputs",
				"non_consumer_boundary",
			}
			return s
		},
		"codegen consumer": func(s IncrementalModuleSummary) IncrementalModuleSummary {
			s.CodegenConsumer = true
			return s
		},
		"linker consumer": func(s IncrementalModuleSummary) IncrementalModuleSummary {
			s.LinkerConsumer = true
			return s
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateIncrementalModuleSummary(mutate(summary)); err == nil {
				t.Fatalf("ValidateIncrementalModuleSummary accepted %s", name)
			}
		})
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
