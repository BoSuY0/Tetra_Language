package rambaseline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tetra_language/tools/internal/rsstelemetry"
)

func TestRunWritesPhaseAlignedBaselineBundle(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("P0 baseline harness requires linux procfs")
	}
	outDir := t.TempDir()
	result, err := Run(Options{
		OutDir:         outDir,
		Iterations:     2,
		WorkBytes:      64 * 1024,
		GitHead:        "0123456789abcdef0123456789abcdef01234567",
		GitStatusShort: "",
		Command:        []string{"ram-p0-baseline", "--test"},
		Now:            func() time.Time { return time.Unix(1700000000, 0).UTC() },
	})
	if err != nil {
		t.Fatalf("Run baseline harness: %v", err)
	}
	if result.OutDir != outDir {
		t.Fatalf("OutDir = %q, want %q", result.OutDir, outDir)
	}

	for _, path := range []string{
		result.ManifestPath,
		result.RSSPath,
		result.ValidatorOutputPath,
		filepath.Join(outDir, "host-fingerprint.json"),
		filepath.Join(outDir, "command-manifest.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing baseline artifact %s: %v", path, err)
		}
	}

	sample, err := rsstelemetry.ReadFile(result.RSSPath, outDir)
	if err != nil {
		t.Fatalf("ReadFile generated RSS sidecar: %v", err)
	}
	if sample.Schema != rsstelemetry.SchemaV2 ||
		sample.Method != rsstelemetry.MethodLinuxProcfsPhaseRSSSamplerV2 ||
		sample.WorkloadKind != "steady_state" {
		t.Fatalf("unexpected RSS sidecar identity: %+v", sample)
	}
	if sample.SampleCount < 5 {
		t.Fatalf("sample_count = %d, want phase-aligned samples", sample.SampleCount)
	}

	rawValidator, err := os.ReadFile(result.ValidatorOutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rawValidator), "pass") {
		t.Fatalf("validator output = %q, want pass", rawValidator)
	}

	rawManifest, err := os.ReadFile(result.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		t.Fatalf("manifest JSON: %v", err)
	}
	if manifest["schema"] != Schema {
		t.Fatalf("manifest schema = %v, want %s", manifest["schema"], Schema)
	}
	if manifest["git_head"] != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("manifest git_head = %v", manifest["git_head"])
	}
	if manifest["allocator_mode"] != "process_bump_small_heap_v0" {
		t.Fatalf("manifest allocator_mode = %v", manifest["allocator_mode"])
	}
}
