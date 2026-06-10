package main

import (
	"strings"
	"testing"
)

func TestValidateActorRuntimeFoundationRequiresManifestInReportDir(t *testing.T) {
	err := validateActorRuntimeFoundationReportDir(t.TempDir(), "e2c19b8ee276158f8eb2c54cf61e11bd84952893")
	if err == nil {
		t.Fatalf("expected missing actor foundation manifest to fail")
	}
	if !strings.Contains(err.Error(), "actor-runtime-foundation-manifest.json") {
		t.Fatalf("error = %v, want actor foundation manifest path", err)
	}
}
