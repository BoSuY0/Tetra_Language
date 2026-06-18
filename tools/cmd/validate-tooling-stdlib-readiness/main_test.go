package main

import (
	"strings"
	"testing"
)

func TestValidateToolingStdlibReadinessAcceptsProductionEvidence(t *testing.T) {
	evidence := readinessEvidence{
		Features: []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.4.0",
  "features": [
    {
      "id": "cli.core",
      "name": "Core CLI workflows",
      "status": "current",
      "since": "v0.2.0",
      "scope": "check/build/run/fmt/test/doc/doctor/targets/features/formats/new/interface/project/workspace/smoke/eco/clean/version/lsp local workflows",
      "stability": "production daily-development workflows",
      "docs": ["docs/spec/policy/cli_contracts.md"]
    },
    {
      "id": "stdlib.core-current",
      "name": "Core standard library current profile",
      "status": "current",
      "since": "v0.4.0",
      "scope": "production lib.core modules for collections, strings, slices, math, IO, filesystem, networking, async, sync, testing, serialization, time, and crypto interfaces",
      "stability": "production API contracts with bounded host API claims",
      "docs": ["docs/spec/standard_library/stdlib.md"]
    },
    {
      "id": "stdlib.experimental-mirrors",
      "name": "Experimental standard-library mirrors",
      "status": "current",
      "since": "v0.4.0",
      "scope": "compatibility mirrors under lib.experimental.* forward to lib.core.* modules for legacy source compatibility",
      "stability": "compatibility bridge only",
      "docs": ["docs/spec/standard_library/stdlib.md"]
    }
  ]
}`),
		StdlibDocs: []byte(`# Standard Library

Production modules:
- lib.core.collections
- lib.core.strings
- lib.core.slices
- lib.core.math
- lib.core.io
- lib.core.filesystem
- lib.core.networking
- lib.core.async
- lib.core.sync
- lib.core.testing
- lib.core.serialization
- lib.core.time
- lib.core.crypto
`),
		CLIContracts: []byte(`# CLI

Status: current v0.4.0 production tooling contract.

Commands: check build run fmt test doc doctor project workspace lsp.
`),
	}

	if err := validateToolingStdlibReadiness(evidence); err != nil {
		t.Fatalf("validateToolingStdlibReadiness failed: %v", err)
	}
}

func TestValidateToolingStdlibReadinessRejectsTextualLSPMVPClaims(t *testing.T) {
	evidence := readinessEvidence{
		Features: []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.4.0",
  "features": [
    {
      "id": "cli.core",
      "name": "Core CLI workflows",
      "status": "current",
      "since": "v0.2.0",
      "scope": "check/build/run/fmt/test/doc/doctor/targets/features/formats/new/interface/project/workspace/smoke/eco/clean/version/lsp local workflows",
      "stability": "production daily-development workflows",
      "docs": ["docs/spec/policy/cli_contracts.md"]
    },
    {
      "id": "stdlib.core-current",
      "name": "Core standard library current profile",
      "status": "current",
      "since": "v0.4.0",
      "scope": "production lib.core modules for collections, strings, slices, math, IO, filesystem, networking, async, sync, testing, serialization, time, and crypto interfaces",
      "stability": "production API contracts with no provisional host API claims",
      "docs": ["docs/spec/standard_library/stdlib.md"]
    },
    {
      "id": "stdlib.experimental-mirrors",
      "name": "Experimental standard-library mirrors",
      "status": "current",
      "since": "v0.4.0",
      "scope": "compatibility mirrors under lib.experimental.* forward to lib.core.* modules for legacy source compatibility",
      "stability": "compatibility bridge only",
      "docs": ["docs/spec/standard_library/stdlib.md"]
    }
  ]
}`),
		StdlibDocs: []byte(`# Standard Library

Production modules:
- lib.core.collections
- lib.core.strings
- lib.core.slices
- lib.core.math
- lib.core.io
- lib.core.filesystem
- lib.core.networking
- lib.core.async
- lib.core.sync
- lib.core.testing
- lib.core.serialization
- lib.core.time
- lib.core.crypto
`),
		CLIContracts: []byte(`Status: current v0.4.0 production tooling contract.

Commands: check build run fmt test doc doctor project workspace lsp.

Rename support is a single-file textual MVP. Syntax-aware rename is not implemented.
`),
	}

	err := validateToolingStdlibReadiness(evidence)
	if err == nil {
		t.Fatalf("expected LSP MVP readiness failure")
	}
	for _, want := range []string{
		"docs/spec/policy/cli_contracts.md",
		"single-file textual MVP",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
}

func TestValidateToolingStdlibReadinessRejectsPlaceholderStdlibClaims(t *testing.T) {
	evidence := readinessEvidence{
		Features: []byte(`{
  "schema": "tetra.features.v1",
  "version": "v0.4.0",
  "features": [
    {
      "id": "cli.core",
      "name": "Core CLI workflows",
      "status": "current",
      "since": "v0.2.0",
      "scope": "check/build/run/fmt/test/doc/doctor/targets/features/formats/new/interface/project/workspace/smoke/eco/clean/version/lsp local workflows",
      "stability": "supported in the current v0.4.0 local profile",
      "docs": ["docs/spec/policy/cli_contracts.md"]
    },
    {
      "id": "stdlib.core-current",
      "name": "Core standard library current profile",
      "status": "current",
      "since": "v0.2.0",
      "scope": "release-covered lib.core helper modules with explicit placeholder labels for filesystem, networking, and crypto surfaces",
      "stability": "current import paths and smoke coverage; placeholder modules are not production host APIs",
      "docs": ["docs/spec/standard_library/stdlib.md"]
    },
    {
      "id": "stdlib.experimental-mirrors",
      "name": "Experimental standard-library mirrors",
      "status": "current",
      "since": "v0.4.0",
      "scope": "production compatibility mirrors under lib.experimental.* forward to lib.core.* modules for legacy source compatibility",
      "stability": "current compatibility bridge",
      "docs": ["docs/spec/standard_library/stdlib.md"]
    }
  ]
}`),
		StdlibDocs: []byte(
			"lib.core.filesystem is a stable placeholder interface, not a host filesystem implementation.",
		),
		CLIContracts: []byte(
			"Status: future v1 required tooling contract.\ncurrent public profile for this branch are `v0.3.0`.",
		),
	}

	err := validateToolingStdlibReadiness(evidence)
	if err == nil {
		t.Fatalf("expected placeholder readiness failure")
	}
	for _, want := range []string{
		"stdlib.core-current",
		"placeholder",
		"docs/spec/standard_library/stdlib.md",
		"docs/spec/policy/cli_contracts.md",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
}
