package compiler_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/memoryfacts"
)

// ---- explain_reports_alloc_test.go ----

func TestBuildAllocReportShowsValidEmptyConstructorNoAllocatorAccess(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(0)
    return xs.len
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:            1,
		EmitAllocReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var alloc struct {
		Functions []struct {
			Function    string `json:"name"`
			Allocations []struct {
				ID                    string `json:"id"`
				SiteID                string `json:"site_id"`
				Builtin               string `json:"builtin"`
				Storage               string `json:"storage"`
				PlannedStorage        string `json:"planned_storage"`
				ActualLoweringStorage string `json:"actual_lowering_storage"`
				LengthStatus          string `json:"length_status"`
				ZeroGuardStatus       string `json:"zero_guard_status"`
				NegativeGuardStatus   string `json:"negative_guard_status"`
				OverflowGuardStatus   string `json:"overflow_guard_status"`
				ValidationStatus      string `json:"validation_status"`
				LoweringStatus        string `json:"lowering_status"`
			} `json:"allocations"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".alloc.json")
	if err != nil {
		t.Fatalf("read alloc report: %v", err)
	}
	if err := json.Unmarshal(raw, &alloc); err != nil {
		t.Fatalf("parse alloc report: %v", err)
	}
	for _, fn := range alloc.Functions {
		if fn.Function != "main" {
			continue
		}
		for _, site := range fn.Allocations {
			if site.ID != "xs" {
				continue
			}
			if site.SiteID == "" || site.Builtin != "core.make_u8" {
				t.Fatalf("empty allocation report site missing stable id/builtin: %+v", site)
			}
			if site.PlannedStorage != site.Storage || site.ActualLoweringStorage == "" {
				t.Fatalf("empty allocation report missing planned/actual storage: %+v", site)
			}
			if site.ValidationStatus == "" || site.LoweringStatus == "" {
				t.Fatalf("empty allocation report missing validation/lowering status: %+v", site)
			}
			if site.Storage != "Eliminated" ||
				site.LengthStatus != "valid_empty_allocation" ||
				site.ZeroGuardStatus != "valid_empty_no_allocator" ||
				site.NegativeGuardStatus != "reject_before_allocation" ||
				site.OverflowGuardStatus != "reject_before_allocation" {
				t.Fatalf("empty allocation report site = %+v", site)
			}
			text, err := os.ReadFile(outPath + ".alloc.txt")
			if err != nil {
				t.Fatalf("read alloc text report: %v", err)
			}
			for _, want := range []string{
				"planned_storage: Eliminated",
				"actual_lowering_storage:",
				"length_status: valid_empty_allocation",
				"zero_guard: valid_empty_no_allocator",
			} {
				if !strings.Contains(string(text), want) {
					t.Fatalf("alloc text report missing %q:\n%s", want, text)
				}
			}
			return
		}
	}
	t.Fatalf("alloc report missing main/xs empty allocation: %+v", alloc.Functions)
}

func TestBuildAllocReportShowsStackLoweredActualStorage(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[0] = 10
    xs[1] = 11
    xs[2] = 12
    xs[3] = 9
    return xs[0] + xs[1] + xs[2] + xs[3]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:            1,
		EmitAllocReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var alloc struct {
		SchemaVersion int `json:"schema_version"`
		Summary       struct {
			AllocationCount              int            `json:"allocation_count"`
			StorageClasses               map[string]int `json:"storage_classes"`
			ActualLoweringStorageClasses map[string]int `json:"actual_lowering_storage_classes"`
			RuntimePaths                 map[string]int `json:"runtime_paths"`
			BytesRequested               int            `json:"bytes_requested"`
			BytesReserved                int            `json:"bytes_reserved"`
		} `json:"summary"`
		Functions []struct {
			Function    string `json:"name"`
			Allocations []struct {
				ID                    string `json:"id"`
				PlannedStorage        string `json:"planned_storage"`
				ActualLoweringStorage string `json:"actual_lowering_storage"`
				LoweringStatus        string `json:"lowering_status"`
				RuntimePath           string `json:"runtime_path"`
				BytesRequested        int    `json:"bytes_requested"`
				BytesReserved         int    `json:"bytes_reserved"`
				Reason                string `json:"reason"`
			} `json:"allocations"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".alloc.json")
	if err != nil {
		t.Fatalf("read alloc report: %v", err)
	}
	if err := json.Unmarshal(raw, &alloc); err != nil {
		t.Fatalf("parse alloc report: %v", err)
	}
	if alloc.SchemaVersion != 2 {
		t.Fatalf("alloc report schema_version = %d, want 2", alloc.SchemaVersion)
	}
	if alloc.Summary.AllocationCount == 0 ||
		alloc.Summary.StorageClasses["Stack"] == 0 ||
		alloc.Summary.ActualLoweringStorageClasses["Stack"] == 0 ||
		alloc.Summary.RuntimePaths["stack_frame"] == 0 ||
		alloc.Summary.BytesRequested == 0 ||
		alloc.Summary.BytesReserved == 0 {
		t.Fatalf("alloc report summary missing P5.4 counts: %+v", alloc.Summary)
	}
	for _, fn := range alloc.Functions {
		if fn.Function != "main" {
			continue
		}
		for _, site := range fn.Allocations {
			if site.ID != "xs" {
				continue
			}
			if site.PlannedStorage != "Stack" || site.ActualLoweringStorage != "Stack" ||
				site.LoweringStatus != "stack_lowering" {
				t.Fatalf(
					"stack allocation report site = %+v, want Stack/Stack stack_lowering",
					site,
				)
			}
			if site.RuntimePath != "stack_frame" || site.BytesRequested != 16 ||
				site.BytesReserved != 16 {
				t.Fatalf(
					"stack allocation runtime report site = %+v, want stack_frame bytes 16/16",
					site,
				)
			}
			if !strings.Contains(site.Reason, "fixed_i32_no_escape") {
				t.Fatalf(
					"stack allocation reason = %q, want fixed_i32_no_escape evidence",
					site.Reason,
				)
			}
			text, err := os.ReadFile(outPath + ".alloc.txt")
			if err != nil {
				t.Fatalf("read alloc text report: %v", err)
			}
			for _, want := range []string{
				"planned_storage: Stack",
				"actual_lowering_storage: Stack",
				"lowering_status: stack_lowering",
				"runtime_path: stack_frame",
				"bytes_requested: 16",
				"bytes_reserved: 16",
				"totals allocation_count:",
				"runtime_paths:stack_frame=",
			} {
				if !strings.Contains(string(text), want) {
					t.Fatalf("alloc text report missing %q:\n%s", want, text)
				}
			}
			return
		}
	}
	t.Fatalf("alloc report missing main/xs stack allocation: %+v", alloc.Functions)
}

func TestBuildOwnedAllocDropOptInExecutableContainsMunmapAndRuns(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        let _stored: Int = core.store_i32(p, 7, mem)
    return 7
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:                   1,
		OwnedAllocDropLowering: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}
	exe, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read executable: %v", err)
	}
	for _, want := range [][]byte{
		{0x8B, 0x77, 0xF8},
		{0x48, 0x81, 0xC7, 0xF8, 0xFF, 0xFF, 0xFF},
		{0xB8, 0x0B, 0x00, 0x00, 0x00, 0x0F, 0x05},
	} {
		if !bytes.Contains(exe, want) {
			t.Fatalf("owned alloc drop executable missing % x", want)
		}
	}
	cmd := exec.Command(outPath)
	runOut, runErr := cmd.CombinedOutput()
	if string(runOut) != "" {
		t.Fatalf("runtime stdout mismatch: %q", runOut)
	}
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok {
		t.Fatalf("runtime exit = %v, want exit status 7", runErr)
	}
	if exitErr.ExitCode() != 7 {
		t.Fatalf("runtime exit code = %d, want 7", exitErr.ExitCode())
	}
}

func TestBuildAllocReportScopesStackStorageEvidencePerTarget(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	src := `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[0] = 10
    xs[1] = 11
    xs[2] = 12
    xs[3] = 9
    return xs[0] + xs[1] + xs[2] + xs[3]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	tests := []struct {
		target         string
		claimLevel     string
		evidenceScope  string
		wantStackLower bool
	}{
		{"linux-x64", "production/host_runtime", "host_runtime_verified", true},
		{"macos-x64", "build_lower_only unless run", "build_lower_only_target_host_required", true},
		{
			"windows-x64",
			"build_lower_only unless run",
			"build_lower_only_target_host_required",
			true,
		},
		{"wasm32-wasi", "artifact/runtime tiered", "artifact_runtime_tiered_safe_limited", false},
		{"wasm32-web", "artifact/runtime tiered", "artifact_runtime_tiered_safe_limited", false},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			outPath := filepath.Join(dir, strings.ReplaceAll(tt.target, "-", "_"))
			if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, tt.target, compiler.BuildOptions{
				Jobs:            1,
				EmitAllocReport: true,
			}); err != nil {
				t.Fatalf("BuildFileWithStatsOpt: %v", err)
			}

			var alloc struct {
				Target                 string `json:"target"`
				TargetMemoryClaimLevel string `json:"target_memory_claim_level"`
				StorageEvidenceScope   string `json:"storage_evidence_scope"`
				Summary                struct {
					ActualLoweringStorageClasses map[string]int `json:"actual_lowering_storage_classes"`
					RuntimePaths                 map[string]int `json:"runtime_paths"`
				} `json:"summary"`
			}
			raw, err := os.ReadFile(outPath + ".alloc.json")
			if err != nil {
				t.Fatalf("read alloc report: %v", err)
			}
			if err := json.Unmarshal(raw, &alloc); err != nil {
				t.Fatalf("parse alloc report: %v", err)
			}
			if alloc.Target != tt.target {
				t.Fatalf("alloc target = %q, want %q", alloc.Target, tt.target)
			}
			if alloc.TargetMemoryClaimLevel != tt.claimLevel {
				t.Fatalf(
					"%s target_memory_claim_level = %q, want %q",
					tt.target,
					alloc.TargetMemoryClaimLevel,
					tt.claimLevel,
				)
			}
			if alloc.StorageEvidenceScope != tt.evidenceScope {
				t.Fatalf(
					"%s storage_evidence_scope = %q, want %q",
					tt.target,
					alloc.StorageEvidenceScope,
					tt.evidenceScope,
				)
			}
			stackLowered := alloc.Summary.ActualLoweringStorageClasses["Stack"] > 0 ||
				alloc.Summary.RuntimePaths["stack_frame"] > 0
			if stackLowered != tt.wantStackLower {
				t.Fatalf(
					"%s stack lowering evidence = %v from summary %+v, want %v",
					tt.target,
					stackLowered,
					alloc.Summary,
					tt.wantStackLower,
				)
			}
			if tt.target != "linux-x64" &&
				alloc.TargetMemoryClaimLevel == "production/host_runtime" {
				t.Fatalf("%s inherited linux-x64 runtime production claim", tt.target)
			}
			if tt.target != "linux-x64" && alloc.StorageEvidenceScope == "host_runtime_verified" {
				t.Fatalf("%s inherited linux-x64 host runtime evidence scope", tt.target)
			}
		})
	}
}

func TestBuildAllocReportShowsFunctionTempRegionLoweredActualStorage(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	plainOutPath := filepath.Join(dir, "plain")
	outPath := filepath.Join(dir, "app")
	src := `
func main() -> Int
uses alloc, mem:
    let n: Int = 7
    var xs: []u8 = make_u8(8)
    xs[0] = 20
    let copied: []u8 = xs.window(0, n).copy()
    return copied.len
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, plainOutPath, "linux-x64", compiler.BuildOptions{
		Jobs: 1,
	}); err != nil {
		t.Fatalf("plain BuildFileWithStatsOpt: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:            1,
		EmitAllocReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}
	for _, path := range []string{plainOutPath, outPath} {
		cmd := exec.Command(path)
		runOut, runErr := cmd.CombinedOutput()
		if string(runOut) != "" {
			t.Fatalf("%s runtime stdout mismatch: %q", filepath.Base(path), runOut)
		}
		exitErr, ok := runErr.(*exec.ExitError)
		if !ok {
			t.Fatalf("%s runtime exit = %v, want exit status 7", filepath.Base(path), runErr)
		}
		if exitErr.ExitCode() != 7 {
			t.Fatalf("%s runtime exit code = %d, want 7", filepath.Base(path), exitErr.ExitCode())
		}
	}

	var alloc struct {
		Summary struct {
			StorageClasses               map[string]int `json:"storage_classes"`
			ActualLoweringStorageClasses map[string]int `json:"actual_lowering_storage_classes"`
			RuntimePaths                 map[string]int `json:"runtime_paths"`
			Regions                      []struct {
				RegionID        string `json:"region_id"`
				Lifetime        string `json:"lifetime"`
				StorageClass    string `json:"storage_class"`
				RuntimePath     string `json:"runtime_path"`
				AllocationCount int    `json:"allocation_count"`
			} `json:"regions"`
		} `json:"summary"`
		Functions []struct {
			Function    string `json:"name"`
			Allocations []struct {
				ID                    string `json:"id"`
				PlannedStorage        string `json:"planned_storage"`
				ActualLoweringStorage string `json:"actual_lowering_storage"`
				LoweringStatus        string `json:"lowering_status"`
				RuntimePath           string `json:"runtime_path"`
				AllocatorClass        string `json:"allocator_class"`
				RegionID              string `json:"region_id"`
				Lifetime              string `json:"lifetime"`
			} `json:"allocations"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".alloc.json")
	if err != nil {
		t.Fatalf("read alloc report: %v", err)
	}
	if err := json.Unmarshal(raw, &alloc); err != nil {
		t.Fatalf("parse alloc report: %v", err)
	}
	if alloc.Summary.StorageClasses["FunctionTempRegion"] == 0 ||
		alloc.Summary.ActualLoweringStorageClasses["FunctionTempRegion"] == 0 ||
		alloc.Summary.RuntimePaths["scoped_single_mapping_v0"] == 0 {
		t.Fatalf("function-temp region summary missing storage/runtime counts: %+v", alloc.Summary)
	}
	if len(alloc.Summary.Regions) == 0 {
		t.Fatalf("function-temp region summary missing region rows: %+v", alloc.Summary)
	}
	for _, region := range alloc.Summary.Regions {
		if region.RegionID == "region:main:temp" &&
			region.Lifetime == "function:main" &&
			region.StorageClass == "FunctionTempRegion" &&
			region.RuntimePath == "scoped_single_mapping_v0" &&
			region.AllocationCount == 1 {
			goto foundRegion
		}
	}
	t.Fatalf(
		"function-temp region summary rows = %+v, want region:main:temp FunctionTempRegion",
		alloc.Summary.Regions,
	)

foundRegion:
	for _, fn := range alloc.Functions {
		if fn.Function != "main" {
			continue
		}
		for _, site := range fn.Allocations {
			if site.ID != "copied" {
				continue
			}
			if site.PlannedStorage != "FunctionTempRegion" || site.ActualLoweringStorage != "FunctionTempRegion" || site.LoweringStatus != "function_temp_region_lowering" {
				t.Fatalf(
					"function-temp allocation report site = %+v, want FunctionTempRegion/FunctionTempRegion",
					site,
				)
			}
			if site.RuntimePath != "scoped_single_mapping_v0" || site.AllocatorClass != "function_temp_region" || site.RegionID != "region:main:temp" || site.Lifetime != "function:main" {
				t.Fatalf("function-temp runtime report site = %+v, want region evidence", site)
			}
			text, err := os.ReadFile(outPath + ".alloc.txt")
			if err != nil {
				t.Fatalf("read alloc text report: %v", err)
			}
			for _, want := range []string{
				"planned_storage: FunctionTempRegion",
				"actual_lowering_storage: FunctionTempRegion",
				"lowering_status: function_temp_region_lowering",
				"runtime_path: scoped_single_mapping_v0",
				"allocator_class: function_temp_region",
				"region_id: region:main:temp",
				"lifetime: function:main",
			} {
				if !strings.Contains(string(text), want) {
					t.Fatalf("alloc text report missing %q:\n%s", want, text)
				}
			}
			return
		}
	}
	t.Fatalf("alloc report missing main/copied function-temp allocation: %+v", alloc.Functions)
}

func TestBuildReportsShowBorrowCopyProvenanceAndAllocationIntent(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(3)
    xs[0] = 65
    xs[1] = 66
    xs[2] = 67
    let borrowed: []u8 = xs.window(1, 2).borrow()
    let copied: []u8 = borrowed.copy()
    return copied.len
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:            1,
		EmitProof:       true,
		EmitAllocReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	proof, err := os.ReadFile(outPath + ".proof.json")
	if err != nil {
		t.Fatalf("read proof report: %v", err)
	}
	for _, want := range []string{
		"borrowed_imm",
		"no_escape",
		"derived_window",
		"owned",
		"provenance_known",
	} {
		if !strings.Contains(string(proof), want) {
			t.Fatalf("proof report missing %q:\n%s", want, proof)
		}
	}

	var alloc struct {
		Functions []struct {
			Function    string `json:"name"`
			Allocations []struct {
				ID          string `json:"id"`
				ValueID     string `json:"value_id"`
				ElementType string `json:"element_type"`
				Storage     string `json:"storage"`
				Reason      string `json:"reason"`
			} `json:"allocations"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".alloc.json")
	if err != nil {
		t.Fatalf("read alloc report: %v", err)
	}
	if err := json.Unmarshal(raw, &alloc); err != nil {
		t.Fatalf("parse alloc report: %v", err)
	}
	var sawCopy bool
	for _, fn := range alloc.Functions {
		if fn.Function != "main" {
			continue
		}
		for _, site := range fn.Allocations {
			if site.ID == "borrowed" || site.ValueID == "view:borrowed" {
				t.Fatalf("borrowed view should not appear as allocation: %+v", site)
			}
			if site.ID == "copied" {
				sawCopy = true
				if site.ElementType != "u8" {
					t.Fatalf("copy allocation element type = %q, want u8", site.ElementType)
				}
				if site.Storage == "" || site.Reason == "" {
					t.Fatalf("copy allocation missing storage/reason: %+v", site)
				}
			}
		}
	}
	if !sawCopy {
		t.Fatalf("alloc report missing copied allocation intent: %+v", alloc.Functions)
	}
}

func TestBuildCommandEmitMemoryReportWritesSchemaV1(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let _: UInt8 = core.store_u8(core.ptr_add(p, 1, mem), 7, mem)
        return core.load_u8(core.ptr_add(p, 1, mem), mem)
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitMemoryReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var report struct {
		SchemaVersion string `json:"schema_version"`
		Rows          []struct {
			SourceFactID      string `json:"source_fact_id"`
			LoweredArtifactID string `json:"lowered_artifact_id"`
			Claim             string `json:"claim"`
			ClaimLevel        string `json:"claim_level"`
			ValidatorStatus   string `json:"validator_status"`
		} `json:"rows"`
	}
	raw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report: %v", err)
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse memory report: %v", err)
	}
	if report.SchemaVersion != "tetra.memory-report.v1" {
		t.Fatalf("schema_version = %q, want tetra.memory-report.v1", report.SchemaVersion)
	}
	var sawAllocBase bool
	var sawRepresentationMetadata bool
	for _, row := range report.Rows {
		if row.SourceFactID == "" {
			t.Fatalf("memory report row missing source_fact_id: %+v", row)
		}
		if row.Claim == "allocation_base_metadata" {
			sawAllocBase = true
			if row.LoweredArtifactID == "" || row.ClaimLevel != "validated" ||
				row.ValidatorStatus != "pass" {
				t.Fatalf(
					"allocation_base_metadata row = %+v, want lowered artifact and validated/pass",
					row,
				)
			}
		}
		if row.Claim == "safe_representation_metadata: not_user_assignable" {
			sawRepresentationMetadata = true
			if row.ClaimLevel != "validated" || row.ValidatorStatus != "pass" {
				t.Fatalf("safe_representation_metadata row = %+v, want validated/pass", row)
			}
		}
	}
	if !sawAllocBase {
		t.Fatalf("memory report missing allocation_base_metadata row:\n%s", raw)
	}
	if !sawRepresentationMetadata {
		t.Fatalf("memory report missing safe_representation_metadata row:\n%s", raw)
	}
}

// ---- explain_reports_bounds_test.go ----

func TestBuildBoundsReportShowsWindowLoopCheckRemoval(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_window(xs: []i32) -> Int
uses mem:
    var total = 0
    for x in xs.window(1, 2):
        total = total + x
    return total

func get(xs: []i32, i: Int) -> Int
uses mem:
    return xs[i]

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(3)
    xs[0] = 1
    xs[1] = 2
    xs[2] = 3
    return sum_window(xs) + get(xs, 1)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Totals struct {
			Removed int `json:"removed"`
			Left    int `json:"left"`
		} `json:"totals"`
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
				Reason  string `json:"reason"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	if bounds.Totals.Removed == 0 || bounds.Totals.Left == 0 {
		t.Fatalf(
			"bounds totals = %+v, want removed window loop check and left external check",
			bounds.Totals,
		)
	}
	var sawWindowLoopRemoval bool
	for _, fn := range bounds.Functions {
		if fn.Function != "sum_window" {
			continue
		}
		for _, site := range fn.Sites {
			if site.Removed && site.ProofID != "" && site.Reason != "" {
				sawWindowLoopRemoval = true
			}
		}
	}
	if !sawWindowLoopRemoval {
		t.Fatalf(
			"bounds report did not show proof-tagged removal for sum_window: %+v",
			bounds.Functions,
		)
	}
}

func TestBuildBoundsReportShowsViewChainReason(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_chain(xs: []i32) -> Int
uses mem:
    let view: []i32 = xs.prefix(4).suffix(1)
    var total = 0
    for x in view:
        total = total + x
    return total

func sum_bad() -> Int:
    let view: String = core.string_window("abc", 4, 0)
    var total = 0
    for ch in view:
        total = total + ch
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[0] = 1
    xs[1] = 2
    xs[2] = 3
    xs[3] = 4
    return sum_chain(xs) + sum_bad()
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
		EmitProof:        true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
				Reason  string `json:"reason"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	var sawChain bool
	var sawBadChecked bool
	for _, fn := range bounds.Functions {
		for _, site := range fn.Sites {
			if fn.Function == "sum_chain" && site.Removed &&
				site.Reason == "removed_by_view_chain" &&
				site.ProofID != "" {
				sawChain = true
			}
			if fn.Function == "sum_bad" && !site.Removed && site.ProofID == "" {
				sawBadChecked = true
			}
			if fn.Function == "sum_bad" && (site.Removed || site.ProofID != "") {
				t.Fatalf("invalid view chain must not claim removed proof site: %+v", fn.Sites)
			}
		}
	}
	if !sawChain {
		t.Fatalf("bounds report missing removed_by_view_chain for sum_chain: %+v", bounds.Functions)
	}
	if !sawBadChecked {
		t.Fatalf(
			"bounds report missing checked invalid view site for sum_bad: %+v",
			bounds.Functions,
		)
	}
}

func TestBuildBoundsAndProofReportsShowWhileRangeReason(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_while(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = i + 1
    return total

func get(xs: []i32, i: Int) -> Int
uses mem:
    return xs[i]

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum_while(xs) + get(xs, 0)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
		EmitProof:        true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	boundsRaw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	boundsText := string(boundsRaw)
	for _, want := range []string{
		`"reason": "removed_by_while_range"`,
		`"reason": "left_missing_dominance"`,
		`"proof_id": "proof:while:`,
	} {
		if !strings.Contains(boundsText, want) {
			t.Fatalf("bounds report missing %q:\n%s", want, boundsText)
		}
	}

	proofRaw, err := os.ReadFile(outPath + ".proof.json")
	if err != nil {
		t.Fatalf("read proof report: %v", err)
	}
	proofText := string(proofRaw)
	for _, want := range []string{
		`"reason": "while loop range proof"`,
		`"removed_bounds_check": true`,
		`"guard": "i < xs.len"`,
		`"fact": "i in [0, xs.len);`,
		`derivation: non_negative, less_than_len`,
	} {
		if !strings.Contains(proofText, want) {
			t.Fatalf("proof report missing %q:\n%s", want, proofText)
		}
	}
}

func TestBuildBoundsAndProofReportsShowCanonicalWhileIncrementReasons(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_commuted(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = 1 + i
    return total

func sum_step(xs: []i32) -> Int
uses mem:
    let step: Int = 1
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = i + step
    return total

func sum_alias(xs: []i32) -> Int
uses mem:
    let start: Int = 0
    let end: Int = xs.len
    var total = 0
    var i = start
    while i < end:
        total = total + xs[i]
        i = i + 1
    return total

func sum_bad(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = i + 2
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum_commuted(xs) + sum_step(xs) + sum_alias(xs) + sum_bad(xs)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
		EmitProof:        true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
				Reason  string `json:"reason"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	findSite := func(function string, removed bool) (reason string, proofID string, ok bool) {
		for _, fn := range bounds.Functions {
			if fn.Function != function {
				continue
			}
			for _, site := range fn.Sites {
				if site.Removed == removed {
					return site.Reason, site.ProofID, true
				}
			}
		}
		return "", "", false
	}
	for _, function := range []string{"sum_commuted", "sum_step", "sum_alias"} {
		if reason, proofID, ok := findSite(function, true); !ok ||
			reason != "removed_by_while_range" ||
			!strings.HasPrefix(proofID, "proof:while:") {
			t.Fatalf(
				"%s site = reason %q proof %q ok=%v, want removed_by_while_range with proof:while",
				function,
				reason,
				proofID,
				ok,
			)
		}
	}
	if reason, proofID, ok := findSite("sum_bad", false); !ok || proofID != "" ||
		reason == "removed_by_while_range" {
		t.Fatalf(
			"sum_bad site = reason %q proof %q ok=%v, want checked site without while removal",
			reason,
			proofID,
			ok,
		)
	}
}

func TestBuildBoundsReportShowsMutationInvalidationReason(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_reassign(xs: []i32, ys: []i32) -> Int
uses mem:
    var view: []i32 = xs
    var total = 0
    var i = 0
    while i < view.len:
        view = ys
        total = total + view[i]
        i = i + 1
    return total

func touch(view: inout []i32) -> Int
uses mem:
    return view.len

func sum_inout(view: inout []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < view.len:
        touch(view)
        total = total + view[i]
        i = i + 1
    return total

func sum_callback(view: inout []i32, cb: fn(inout []i32) -> Int uses mem) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < view.len:
        cb(view)
        total = total + view[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    var ys: []i32 = make_i32(1)
    xs[0] = 1
    ys[0] = 2
    return sum_reassign(xs, ys) + sum_inout(xs) + sum_callback(xs, touch)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
		EmitProof:        true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
				Reason  string `json:"reason"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}

	for _, wantFunction := range []string{"sum_reassign", "sum_inout", "sum_callback"} {
		var sawInvalidated bool
		for _, fn := range bounds.Functions {
			if fn.Function != wantFunction {
				continue
			}
			for _, site := range fn.Sites {
				if site.Removed || site.ProofID != "" {
					t.Fatalf(
						"%s mutation-invalidated site must remain checked without proof: %+v",
						wantFunction,
						fn.Sites,
					)
				}
				if site.Reason == "left_proof_invalidated_by_mutation" {
					sawInvalidated = true
				}
			}
		}
		if !sawInvalidated {
			t.Fatalf(
				"bounds report missing left_proof_invalidated_by_mutation for %s: %+v",
				wantFunction,
				bounds.Functions,
			)
		}
	}
}

func TestBuildBoundsReportShowsBranchGuardReasons(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func branch_remove(xs: []i32, i: Int) -> Int
uses mem:
    if i >= 0 && i < xs.len:
        return xs[i]
    return 0

func branch_missing_lower(xs: []i32, i: Int) -> Int
uses mem:
    if i < xs.len:
        return xs[i]
    return 0

func branch_not_dominating(xs: []i32, i: Int) -> Int
uses mem:
    if i >= 0 && i < xs.len:
        var j = i + 0
    return xs[i]

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 7
    return branch_remove(xs, 0) + branch_missing_lower(xs, 0) + branch_not_dominating(xs, 0)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
		EmitProof:        true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
				Reason  string `json:"reason"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	findSite := func(function string, removed bool) (reason string, proofID string, ok bool) {
		for _, fn := range bounds.Functions {
			if fn.Function != function {
				continue
			}
			for _, site := range fn.Sites {
				if site.Removed == removed {
					return site.Reason, site.ProofID, true
				}
			}
		}
		return "", "", false
	}

	if reason, proofID, ok := findSite("branch_remove", true); !ok ||
		reason != "removed_by_branch_guard" ||
		!strings.HasPrefix(proofID, "proof:if:") {
		t.Fatalf(
			"branch_remove site = reason %q proof %q ok=%v, want removed_by_branch_guard with proof:if",
			reason,
			proofID,
			ok,
		)
	}
	if reason, _, ok := findSite("branch_missing_lower", false); !ok ||
		reason != "left_missing_non_negative_lower_bound" {
		t.Fatalf(
			"branch_missing_lower reason = %q ok=%v, want left_missing_non_negative_lower_bound",
			reason,
			ok,
		)
	}
	if reason, _, ok := findSite("branch_not_dominating", false); !ok ||
		reason != "left_guard_not_dominating" {
		t.Fatalf(
			"branch_not_dominating reason = %q ok=%v, want left_guard_not_dominating",
			reason,
			ok,
		)
	}
}

func TestBuildBoundsReportDoesNotClaimProofForInvalidConstructorLoop(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_bad() -> Int
uses alloc, mem:
    var total = 0
    for x in make_i32(0 - 1):
        total = total + x
    return total

func main() -> Int
uses alloc, mem:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Totals struct {
			Removed int `json:"removed"`
			Left    int `json:"left"`
		} `json:"totals"`
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	for _, fn := range bounds.Functions {
		if fn.Function != "sum_bad" {
			continue
		}
		for _, site := range fn.Sites {
			if site.Removed || site.ProofID != "" {
				t.Fatalf(
					"invalid constructor loop must not claim removed proof site: %+v",
					fn.Sites,
				)
			}
		}
		return
	}
	t.Fatalf(
		"bounds report missing sum_bad checked site: %+v totals=%+v",
		bounds.Functions,
		bounds.Totals,
	)
}

func TestBuildBoundsReportShowsStringWindowLoopCheckRemoval(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_window(text: String) -> Int
uses mem:
    var total = 0
    for ch in text.window(1, 3):
        total = total + ch
    return total

func get(text: String, i: Int) -> Int
uses mem:
    return text[i]

func main() -> Int
uses mem:
    let text: String = "abcdef"
    return sum_window(text) + get(text, 1)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Totals struct {
			Removed int `json:"removed"`
			Left    int `json:"left"`
		} `json:"totals"`
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
				Reason  string `json:"reason"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	if bounds.Totals.Removed == 0 || bounds.Totals.Left == 0 {
		t.Fatalf(
			"bounds totals = %+v, want removed String window loop check and left external check",
			bounds.Totals,
		)
	}
	var sawStringWindowLoopRemoval bool
	for _, fn := range bounds.Functions {
		if fn.Function != "sum_window" {
			continue
		}
		for _, site := range fn.Sites {
			if site.Removed && site.ProofID != "" && site.Reason != "" {
				sawStringWindowLoopRemoval = true
			}
		}
	}
	if !sawStringWindowLoopRemoval {
		t.Fatalf(
			"bounds report did not show proof-tagged removal for sum_window: %+v",
			bounds.Functions,
		)
	}
}

func TestBuildBoundsReportDoesNotClaimProofForInvalidStringViewConstructorLoop(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_bad() -> Int:
    var total = 0
    for ch in core.string_window("abc", 4, 0):
        total = total + ch
    return total

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitBoundsReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	var bounds struct {
		Totals struct {
			Removed int `json:"removed"`
			Left    int `json:"left"`
		} `json:"totals"`
		Functions []struct {
			Function string `json:"function"`
			Sites    []struct {
				Removed bool   `json:"removed"`
				ProofID string `json:"proof_id"`
			} `json:"sites"`
		} `json:"functions"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	for _, fn := range bounds.Functions {
		if fn.Function != "sum_bad" {
			continue
		}
		for _, site := range fn.Sites {
			if site.Removed || site.ProofID != "" {
				t.Fatalf(
					"invalid String view constructor loop must not claim removed proof site: %+v",
					fn.Sites,
				)
			}
		}
		return
	}
	t.Fatalf(
		"bounds report missing sum_bad checked site: %+v totals=%+v",
		bounds.Functions,
		bounds.Totals,
	)
}

// ---- explain_reports_memory_test.go ----

func TestBuildReportsShowBorrowedReturnNoAllocationAndCopyOwnership(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func view(xs: borrow []u8) -> borrow []u8:
    return xs.window(1, 2).borrow()

func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    xs[1] = 20
    xs[2] = 22
    let returned: []u8 = view(xs)
    let copied: []u8 = returned.copy()
    if copied.len != 2:
        return 1
    return copied[0] + copied[1]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		Explain:          true,
		EmitPLIR:         true,
		EmitProof:        true,
		EmitAllocReport:  true,
		EmitMemoryReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	proofRaw, err := os.ReadFile(outPath + ".proof.json")
	if err != nil {
		t.Fatalf("read proof report: %v", err)
	}
	proofText := string(proofRaw)
	for _, want := range []string{
		`"kind": "borrowed_imm"`,
		`"kind": "no_escape"`,
		`"kind": "derived_window"`,
		`"kind": "owned"`,
		`"kind": "provenance_known"`,
	} {
		if !strings.Contains(proofText, want) {
			t.Fatalf("proof report missing %s:\n%s", want, proofText)
		}
	}

	allocRaw, err := os.ReadFile(outPath + ".alloc.json")
	if err != nil {
		t.Fatalf("read alloc report: %v", err)
	}
	allocText := string(allocRaw)
	if strings.Contains(allocText, `"value_id": "alloc_intent:returned"`) {
		t.Fatalf("borrowed return was reported as allocation:\n%s", allocText)
	}
	if !strings.Contains(allocText, `"value_id": "alloc_intent:copied"`) {
		t.Fatalf("copy result allocation intent missing:\n%s", allocText)
	}

	memoryRaw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report: %v", err)
	}
	for _, want := range []string{
		`"claim": "borrowed_imm"`,
		`"claim": "no_escape"`,
		`"claim": "borrow_owner"`,
		`"claim": "borrow_source_fact_id"`,
		`"claim": "copy_owned"`,
		`"claim": "copy_source_fact_id"`,
	} {
		if !strings.Contains(string(memoryRaw), want) {
			t.Fatalf("memory report missing %s:\n%s", want, memoryRaw)
		}
	}
}

func TestBuildReportsShowCopyIntoNoFreshAllocation(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func main() -> Int
uses alloc, mem:
    var src: []u8 = make_u8(2)
    var dst: []u8 = make_u8(2)
    let n: Int = src.copy_into(dst)
    return n
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitPLIR:         true,
		EmitProof:        true,
		EmitAllocReport:  true,
		EmitMemoryReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	allocRaw, err := os.ReadFile(outPath + ".alloc.json")
	if err != nil {
		t.Fatalf("read alloc report: %v", err)
	}
	allocText := string(allocRaw)
	var allocReport struct {
		Functions []struct {
			Allocations []struct {
				ValueID string `json:"value_id"`
				Builtin string `json:"builtin"`
			} `json:"allocations"`
		} `json:"functions"`
	}
	if err := json.Unmarshal(allocRaw, &allocReport); err != nil {
		t.Fatalf("parse alloc report: %v", err)
	}
	for _, fn := range allocReport.Functions {
		for _, alloc := range fn.Allocations {
			if alloc.ValueID == "alloc_intent:n" || strings.Contains(alloc.Builtin, "copy_into") {
				t.Fatalf("copy_into should not appear as a fresh allocation:\n%s", allocText)
			}
		}
	}
	for _, want := range []string{`"value_id": "alloc_intent:src"`, `"value_id": "alloc_intent:dst"`} {
		if !strings.Contains(allocText, want) {
			t.Fatalf("alloc report missing %s:\n%s", want, allocText)
		}
	}

	proofRaw, err := os.ReadFile(outPath + ".proof.json")
	if err != nil {
		t.Fatalf("read proof report: %v", err)
	}
	if !strings.Contains(
		string(proofRaw),
		"copies into caller-owned destination without allocation",
	) {
		t.Fatalf("proof/PLIR report missing copy_into no-allocation note:\n%s", proofRaw)
	}

	memoryRaw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report: %v", err)
	}
	if !strings.Contains(string(memoryRaw), `"claim": "copy_into_destination_fact_id"`) {
		t.Fatalf("memory report missing copy_into destination fact:\n%s", memoryRaw)
	}
}

func TestBuildReportsShowInoutNoAliasProofFact(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func mutate(xs: inout []u8) -> Int
uses mem:
    xs[0] = 1
    return xs[0]

func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(1)
    return mutate(xs)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitProof:        true,
		EmitMemoryReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	proofRaw, err := os.ReadFile(outPath + ".proof.json")
	if err != nil {
		t.Fatalf("read proof report: %v", err)
	}
	proofText := string(proofRaw)
	for _, want := range []string{
		`"kind": "no_alias"`,
		`"value_id": "param:xs"`,
		`"kind": "borrowed_mut"`,
	} {
		if !strings.Contains(proofText, want) {
			t.Fatalf("proof report missing %s:\n%s", want, proofText)
		}
	}

	memoryRaw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report: %v", err)
	}
	memoryText := string(memoryRaw)
	for _, want := range []string{
		`"claim": "no_alias"`,
		`"claim": "mutable_exclusive"`,
		`"claim": "start_inout_exclusive"`,
		`"claim": "end_inout_exclusive"`,
		`"alias_state": "mutable_exclusive"`,
	} {
		if !strings.Contains(memoryText, want) {
			t.Fatalf("memory report missing %s:\n%s", want, memoryText)
		}
	}
}

func TestBuildReportsShowFunctionSummaryFacts(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
enum MoveMsg:
    case take(island)

var g_counter: i32

func worker() -> Int:
    return 7

func borrowed(xs: borrow []u8) -> borrow []u8
uses mem:
    return xs.window(0, 1).borrow()

func owned(xs: borrow []u8) -> []u8
uses alloc, mem:
    return xs.copy()

func mutate(dst: inout []u8) -> Int
uses mem:
    dst[0] = 1
    return dst[0]

func consume_peer(peer: consume actor) -> Int
uses actors:
    return 0

func main() -> Int
uses actors, alloc, capability, islands, mem, runtime:
    var xs: []u8 = make_u8(2)
    xs[0] = 3
    let view: []u8 = borrowed(xs)
    let copied: []u8 = owned(view)
    var dst: []u8 = make_u8(2)
    let _mutated: Int = mutate(dst)
    g_counter = copied[0]
    let task: task.i32 = core.task_spawn_i32("worker")
    let joined: Int = core.task_join_i32(task)
    let peer: actor = core.spawn("worker")
    let peer2: actor = core.spawn("worker")
    let _consumed: Int = consume_peer(peer2)
    unsafe:
        let _mem: cap.mem = core.cap_mem()
        var isl: island = core.island_new(16)
        let _sent_actor: Int = core.send_typed(peer, MoveMsg.take(isl))
        let _raw: ptr = core.alloc_bytes(8)
    return g_counter + joined
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitPLIR:         true,
		EmitMemoryReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	memoryRaw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report: %v", err)
	}
	memoryText := string(memoryRaw)
	for _, want := range []string{
		`"claim": "returns_borrow_from_param"`,
		`"claim": "may_return_region"`,
		`"claim": "returns_owned_new_allocation"`,
		`"claim": "may_store_global"`,
		`"claim": "may_escape_to_actor"`,
		`"claim": "may_escape_to_task"`,
		`"claim": "may_consume_param"`,
		`"claim": "may_mutate_inout"`,
		`"claim": "requires_effects"`,
		`"source_fact_id": "plir:`,
	} {
		if !strings.Contains(memoryText, want) {
			t.Fatalf("memory report missing %s:\n%s", want, memoryText)
		}
	}
}

func TestBuildReportsMemoryFactIDsAndSiteIDsStableAcrossRuns(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	src := `
func borrowed(xs: borrow []u8) -> borrow []u8
uses mem:
    return xs.window(0, 1).borrow()

func owned(xs: borrow []u8) -> []u8
uses alloc, mem:
    return xs.copy()

func main() -> Int
uses alloc, capability, mem:
    var xs: []u8 = make_u8(2)
    xs[0] = 3
    let view: []u8 = borrowed(xs)
    let copied: []u8 = owned(view)
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let _: UInt8 = core.store_u8(core.ptr_add(p, 1, mem), 7, mem)
    return copied[0]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	first := buildMemoryReportFactKeys(t, srcPath, filepath.Join(dir, "app-one"))
	second := buildMemoryReportFactKeys(t, srcPath, filepath.Join(dir, "app-two"))
	if !reflect.DeepEqual(first, second) {
		t.Fatalf(
			"memory report source_fact_id/site_id sequence changed across runs:\nfirst:  %+v\nsecond: %+v",
			first,
			second,
		)
	}
}

func TestReportFlagsDoNotChangeBorrowedReturnFailure(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad.tetra")
	src := `
func bad(xs: borrow []u8) -> []u8:
    return xs.borrow()

func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(1)
    return bad(xs).len
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	options := []compiler.BuildOptions{
		{Jobs: 1},
		{Jobs: 1, EmitPLIR: true},
		{Jobs: 1, EmitProof: true},
		{Jobs: 1, EmitAllocReport: true},
		{Jobs: 1, EmitMemoryReport: true},
		{Jobs: 1, Explain: true},
	}
	for i, opt := range options {
		outPath := filepath.Join(dir, "bad-out-"+string(rune('a'+i)))
		_, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", opt)
		if err == nil ||
			!strings.Contains(
				err.Error(),
				"borrowed slice return requires '-> borrow []u8' or '.copy()'",
			) {
			t.Fatalf("option %d error = %v", i, err)
		}
	}
}

type memoryReportFactKey struct {
	SourceFactID string
	SiteID       string
}

func buildMemoryReportFactKeys(t *testing.T, srcPath string, outPath string) []memoryReportFactKey {
	t.Helper()
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitMemoryReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt %s: %v", outPath, err)
	}
	raw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report %s: %v", outPath, err)
	}
	var report memoryfacts.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse memory report %s: %v", outPath, err)
	}
	keys := make([]memoryReportFactKey, 0, len(report.Rows))
	for _, row := range report.Rows {
		if row.SourceFactID == "" || row.SiteID == "" {
			t.Fatalf("memory report row missing stable ids: %+v", row)
		}
		keys = append(keys, memoryReportFactKey{
			SourceFactID: string(row.SourceFactID),
			SiteID:       row.SiteID,
		})
	}
	return keys
}

// ---- explain_reports_test.go ----

func TestBuildExplainReportsTruthProofAndAllocationArtifacts(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
repr(C) struct Header:
    tag: c_int
    code: c_int

struct Packet:
    tag: Int
    code: Int

func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    for x in xs:
        total = total + x
    return total

func get(xs: []i32, i: Int) -> Int
uses mem:
    return xs[i]

func scratch() -> Int
uses alloc, mem:
    var buf: []u8 = make_u8(4)
    buf[0] = 7
    return 0

func returned_heap() -> []u8
uses alloc, mem:
    var out: []u8 = make_u8(4)
    return out

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs) + get(xs, 0)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		Explain:          true,
		EmitPLIR:         true,
		EmitProof:        true,
		EmitBoundsReport: true,
		EmitAllocReport:  true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	for _, suffix := range []string{
		".plir.txt",
		".plir.json",
		".proof.json",
		".bounds.json",
		".alloc.json",
		".alloc.txt",
		".explain.txt",
		".backend.json",
		".layout.json",
		".perf.json",
	} {
		if _, err := os.Stat(outPath + suffix); err != nil {
			t.Fatalf("missing report %s: %v", outPath+suffix, err)
		}
	}
	var bounds struct {
		Totals struct {
			Removed int `json:"removed"`
			Left    int `json:"left"`
		} `json:"totals"`
	}
	raw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if err := json.Unmarshal(raw, &bounds); err != nil {
		t.Fatalf("parse bounds report: %v", err)
	}
	if bounds.Totals.Removed == 0 || bounds.Totals.Left == 0 {
		t.Fatalf("bounds totals = %+v, want removed and left checks", bounds.Totals)
	}
	var alloc struct {
		Totals struct {
			Heap  int `json:"heap"`
			Stack int `json:"stack"`
		} `json:"totals"`
	}
	raw, err = os.ReadFile(outPath + ".alloc.json")
	if err != nil {
		t.Fatalf("read alloc report: %v", err)
	}
	if err := json.Unmarshal(raw, &alloc); err != nil {
		t.Fatalf("parse alloc report: %v", err)
	}
	if alloc.Totals.Heap == 0 || alloc.Totals.Stack == 0 {
		t.Fatalf(
			"alloc totals = %+v, want conservative heap and planned stack allocations",
			alloc.Totals,
		)
	}
	if !strings.Contains(string(raw), "fixed_small_read_only_local_call_no_escape") {
		t.Fatalf("alloc report missing call-aware stack evidence:\n%s", raw)
	}
	raw, err = os.ReadFile(outPath + ".perf.json")
	if err != nil {
		t.Fatalf("read perf report: %v", err)
	}
	perfText := string(raw)
	for _, want := range []string{
		"p20.0_benchmark_matrix",
		"reports/benchmark-matrix-hardening-v1/benchmarks/p20-matrix-hardening-report.json",
		"left bounds check: missing dominance",
		"heap allocation: escapes through return",
		"not vectorized: no noalias proof",
		"not inlined: code-size budget",
		"register spill: live range pressure",
		"stack fallback: unsupported aggregate return",
		"actor copy: borrowed data crosses boundary",
		"compile_time_tetra",
	} {
		if !strings.Contains(perfText, want) {
			t.Fatalf("perf report missing %q:\n%s", want, perfText)
		}
	}

	raw, err = os.ReadFile(outPath + ".layout.json")
	if err != nil {
		t.Fatalf("read layout report: %v", err)
	}
	layoutText := string(raw)
	for _, want := range []string{
		`"schema_version": 2`,
		`"policy": "p21.0_default_layout_freedom_v1"`,
		`"decisions"`,
		`"compiler_owned_default"`,
		`"field_reordering"`,
		`"repr(C) locks layout"`,
		`"public ABI/exported FFI requires explicit repr(C)"`,
	} {
		if !strings.Contains(layoutText, want) {
			t.Fatalf("layout report missing %q:\n%s", want, layoutText)
		}
	}
}

func TestBuildCompilerPhaseProfileRecordsP7PhaseRSS(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[0] = 10
    xs[1] = 11
    xs[2] = 12
    xs[3] = 9
    return xs[0] + xs[1] + xs[2] + xs[3]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	stats, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:                    2,
		Explain:                 true,
		EmitCompilerPhaseReport: true,
	})
	if err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}
	profilePath := outPath + ".compiler-profile.json"
	if stats.CompilerPhaseProfilePath != profilePath {
		t.Fatalf(
			"CompilerPhaseProfilePath = %q, want %q",
			stats.CompilerPhaseProfilePath,
			profilePath,
		)
	}
	raw, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("read compiler phase profile: %v", err)
	}
	var profile struct {
		Schema          string `json:"schema"`
		Target          string `json:"target"`
		ReportMode      string `json:"report_mode"`
		RequestedJobs   int    `json:"requested_jobs"`
		WorkerCount     int    `json:"worker_count"`
		WorkerReason    string `json:"worker_reason"`
		RSSPeakBytes    uint64 `json:"rss_peak_bytes"`
		GoHeapPeakBytes uint64 `json:"go_heap_peak_alloc_bytes"`
		Phases          []struct {
			Name             string `json:"name"`
			UnixNano         int64  `json:"unix_nano"`
			GoHeapAllocBytes uint64 `json:"go_heap_alloc_bytes"`
			RSSCurrentBytes  uint64 `json:"rss_current_bytes"`
			ModuleCount      int    `json:"module_count"`
		} `json:"phases"`
	}
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatalf("parse compiler phase profile: %v\n%s", err, raw)
	}
	if profile.Schema != "tetra.compiler.phase-profile.v1" ||
		profile.Target != "linux-x64" ||
		profile.ReportMode != "explain" ||
		profile.RequestedJobs != 2 ||
		profile.WorkerCount <= 0 ||
		profile.WorkerReason == "" ||
		profile.RSSPeakBytes == 0 ||
		profile.GoHeapPeakBytes == 0 {
		t.Fatalf("unexpected compiler phase profile header: %+v", profile)
	}
	seen := map[string]bool{}
	for _, phase := range profile.Phases {
		seen[phase.Name] = true
		if phase.UnixNano == 0 ||
			phase.GoHeapAllocBytes == 0 ||
			phase.RSSCurrentBytes == 0 ||
			phase.ModuleCount == 0 {
			t.Fatalf("phase missing P7 memory/count fields: %+v", phase)
		}
	}
	for _, phase := range []string{
		"source_loading_parsing",
		"semantic_analysis",
		"plir_construction",
		"allocation_planning",
		"ir_lowering",
		"module_codegen",
		"object_retention_link",
		"report_generation",
		"final_cleanup",
	} {
		if !seen[phase] {
			t.Fatalf("compiler phase profile missing phase %q: %s", phase, raw)
		}
	}
}

func TestP7CompilerPhaseProfileDropsObjectReferencesBeforeReports(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func helper(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return helper(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Explain:                 true,
		EmitCompilerPhaseReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}
	raw, err := os.ReadFile(outPath + ".compiler-profile.json")
	if err != nil {
		t.Fatalf("read compiler phase profile: %v", err)
	}
	var profile struct {
		Phases []struct {
			Name        string `json:"name"`
			ObjectCount int    `json:"object_count"`
		} `json:"phases"`
	}
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatalf("parse compiler phase profile: %v\n%s", err, raw)
	}
	phaseObjects := map[string]int{}
	for _, phase := range profile.Phases {
		phaseObjects[phase.Name] = phase.ObjectCount
	}
	if got := phaseObjects["object_retention_link"]; got <= 0 {
		t.Fatalf("object_retention_link object_count = %d, want retained objects before link release", got)
	}
	for _, phase := range []string{"report_generation", "final_cleanup"} {
		if got := phaseObjects[phase]; got != 0 {
			t.Fatalf("%s object_count = %d, want 0 after link releases object references", phase, got)
		}
	}
}

func TestP7CompilerPhaseProfileReleasesTransientIRBeforeModuleCodegen(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func helper(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return helper(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Explain:                 true,
		EmitCompilerPhaseReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}
	raw, err := os.ReadFile(outPath + ".compiler-profile.json")
	if err != nil {
		t.Fatalf("read compiler phase profile: %v", err)
	}
	var profile struct {
		Phases []struct {
			Name                        string `json:"name"`
			TransientIRFunctionCount    int    `json:"transient_ir_function_count"`
			AllocationPlanFunctionCount int    `json:"allocation_plan_function_count"`
		} `json:"phases"`
	}
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatalf("parse compiler phase profile: %v\n%s", err, raw)
	}
	byPhase := map[string]struct {
		TransientIRFunctionCount    int
		AllocationPlanFunctionCount int
	}{}
	for _, phase := range profile.Phases {
		byPhase[phase.Name] = struct {
			TransientIRFunctionCount    int
			AllocationPlanFunctionCount int
		}{
			TransientIRFunctionCount:    phase.TransientIRFunctionCount,
			AllocationPlanFunctionCount: phase.AllocationPlanFunctionCount,
		}
	}
	if got := byPhase["plir_construction"].TransientIRFunctionCount; got <= 0 {
		t.Fatalf("plir_construction transient_ir_function_count = %d, want retained PLIR functions", got)
	}
	if got := byPhase["allocation_planning"].AllocationPlanFunctionCount; got <= 0 {
		t.Fatalf(
			"allocation_planning allocation_plan_function_count = %d, want retained allocation plan functions\n%s",
			got,
			raw,
		)
	}
	if got := byPhase["ir_lowering"].TransientIRFunctionCount; got <= 0 {
		t.Fatalf("ir_lowering transient_ir_function_count = %d, want retained lowering summary functions", got)
	}
	for _, phase := range []string{
		"module_codegen",
		"object_retention_link",
		"report_generation",
		"final_cleanup",
	} {
		counts, ok := byPhase[phase]
		if !ok {
			t.Fatalf("missing phase %q in compiler profile: %s", phase, raw)
		}
		if counts.TransientIRFunctionCount != 0 || counts.AllocationPlanFunctionCount != 0 {
			t.Fatalf(
				"%s transient counts = %+v, want no retained transient IR/allocation plan after module codegen",
				phase,
				counts,
			)
		}
	}
}

func TestP7CompilerPhaseProfileReleasesSemanticStateAtFinalCleanup(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func helper(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return helper(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Explain:                 true,
		EmitCompilerPhaseReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}
	raw, err := os.ReadFile(outPath + ".compiler-profile.json")
	if err != nil {
		t.Fatalf("read compiler phase profile: %v", err)
	}
	var profile struct {
		Phases []struct {
			Name                 string `json:"name"`
			SourceFileCount      int    `json:"source_file_count"`
			CheckedFunctionCount int    `json:"checked_function_count"`
			CheckedTypeCount     int    `json:"checked_type_count"`
		} `json:"phases"`
	}
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatalf("parse compiler phase profile: %v\n%s", err, raw)
	}
	byPhase := map[string]struct {
		SourceFileCount      int
		CheckedFunctionCount int
		CheckedTypeCount     int
	}{}
	for _, phase := range profile.Phases {
		byPhase[phase.Name] = struct {
			SourceFileCount      int
			CheckedFunctionCount int
			CheckedTypeCount     int
		}{
			SourceFileCount:      phase.SourceFileCount,
			CheckedFunctionCount: phase.CheckedFunctionCount,
			CheckedTypeCount:     phase.CheckedTypeCount,
		}
	}
	semanticCounts, ok := byPhase["semantic_analysis"]
	if !ok {
		t.Fatalf("missing semantic_analysis phase in compiler profile: %s", raw)
	}
	if semanticCounts.SourceFileCount <= 0 ||
		semanticCounts.CheckedFunctionCount <= 0 ||
		semanticCounts.CheckedTypeCount <= 0 {
		t.Fatalf("semantic_analysis retained counts = %+v, want source and checked graph counts", semanticCounts)
	}
	reportCounts, ok := byPhase["report_generation"]
	if !ok {
		t.Fatalf("missing report_generation phase in compiler profile: %s", raw)
	}
	if reportCounts.CheckedFunctionCount <= 0 || reportCounts.CheckedTypeCount <= 0 {
		t.Fatalf("report_generation retained counts = %+v, want checked graph retained for reports", reportCounts)
	}
	finalCounts, ok := byPhase["final_cleanup"]
	if !ok {
		t.Fatalf("missing final_cleanup phase in compiler profile: %s", raw)
	}
	if finalCounts.SourceFileCount != 0 ||
		finalCounts.CheckedFunctionCount != 0 ||
		finalCounts.CheckedTypeCount != 0 {
		t.Fatalf("final_cleanup retained counts = %+v, want released source and checked graph state", finalCounts)
	}
}

func TestBuildMemoryReportMarksSliceViewDynamicBoundsChecks(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func main() -> Int
uses alloc, mem:
    var bytes: []u8 = make_u8(4)
    var words: []u16 = make_u16(4)
    var nums: []i32 = make_i32(4)
    let b: []u8 = bytes.window(1, 2)
    let w: []u16 = words.prefix(2)
    let n: []i32 = nums.suffix(1)
    let text: String = "abcdef".window(1, 3)
    return b.len + w.len + n.len + text.len
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitMemoryReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	raw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report: %v", err)
	}
	if err := memoryfacts.ValidateReportJSON(raw); err != nil {
		t.Fatalf("ValidateReportJSON: %v\n%s", err, raw)
	}
	var report memoryfacts.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse memory report: %v", err)
	}
	var retained int
	for _, row := range report.Rows {
		if row.Claim != "bounds_check_retained_dynamic" {
			continue
		}
		retained++
		if row.ParentFactID == "" ||
			row.CostClass != memoryfacts.CostDynamicCheckRequired ||
			!row.NormalBuildCheck ||
			row.ValidatorName != "safe_view_bounds_validator" ||
			row.ValidatorStatus != memoryfacts.ValidatorPass ||
			!strings.Contains(row.Reason, "elem_width:") ||
			!strings.Contains(row.Reason, "elem_shift:") {
			t.Fatalf("safe view retained-bounds row = %+v", row)
		}
	}
	if retained < 4 {
		t.Fatalf(
			"bounds_check_retained_dynamic rows = %d, want at least 4:\n%+v",
			retained,
			report.Rows,
		)
	}
}

func TestBuildMemoryReportValidatesExplicitIslandHelperLowering(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func make_buf(isl: island, n: Int) -> []u8
uses alloc, islands, mem:
    var buf: []u8 = core.island_make_u8(isl, n)
    return buf

func main() -> Int
uses alloc, islands, mem:
    var result: Int = 0
    island(64) as isl:
        var out: []u8 = make_buf(isl, 4)
        out[0] = 7
        result = out[0]
    return result
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		EmitMemoryReport: true,
		EmitAllocReport:  true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	raw, err := os.ReadFile(outPath + ".memory.json")
	if err != nil {
		t.Fatalf("read memory report: %v", err)
	}
	if err := memoryfacts.ValidateReportJSON(raw); err != nil {
		t.Fatalf("ValidateReportJSON: %v\n%s", err, raw)
	}
	var report memoryfacts.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("parse memory report: %v", err)
	}
	for _, row := range report.Rows {
		if row.ActualLoweringStorage == memoryfacts.StorageExplicitIsland &&
			row.ClaimLevel == memoryfacts.ClaimValidated &&
			row.LoweredArtifactID != "" {
			return
		}
	}
	t.Fatalf("missing validated ExplicitIsland memory row with lowered artifact: %+v", report.Rows)
}

func TestBuildExplainReportsMachineScalarBackendPath(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func add(a: Int, b: Int) -> Int:
    return a + b

func main() -> Int:
    return add(40, 2)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:    1,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	raw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	if !strings.Contains(string(raw), "machine-ir-scalar") {
		t.Fatalf("backend report missing machine scalar path:\n%s", raw)
	}
	cmd := exec.Command(outPath)
	runOut, runErr := cmd.CombinedOutput()
	if string(runOut) != "" {
		t.Fatalf("runtime stdout mismatch: %q", runOut)
	}
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok {
		t.Fatalf("runtime exit = %v, want exit status 42", runErr)
	}
	if exitErr.ExitCode() != 42 {
		t.Fatalf("runtime exit code = %d, want 42", exitErr.ExitCode())
	}
}

func TestBuildExplainReportsMachineLoopBackendPath(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum_n(n: Int) -> Int:
    var i = 0
    var total = 0
    while i < n:
        total = total + i
        i = i + 1
    return total

func main() -> Int:
    return sum_n(10)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:    1,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	raw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	for _, want := range []string{
		"machine-ir-loop",
		`"liveness"`,
		`"allocation"`,
		`"spills": {}`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("backend report missing %s:\n%s", want, raw)
		}
	}
	cmd := exec.Command(outPath)
	runOut, runErr := cmd.CombinedOutput()
	if string(runOut) != "" {
		t.Fatalf("runtime stdout mismatch: %q", runOut)
	}
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok {
		t.Fatalf("runtime exit = %v, want exit status 45", runErr)
	}
	if exitErr.ExitCode() != 45 {
		t.Fatalf("runtime exit code = %d, want 45", exitErr.ExitCode())
	}
}

func TestBuildExplainReportsMachineSliceSumBackendPath(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(3)
    xs[0] = 1
    xs[1] = 2
    xs[2] = 3
    return sum(xs)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:             1,
		Explain:          true,
		EmitBoundsReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	backendRaw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	for _, want := range []string{
		"machine-ir-slice-sum",
		"index_load",
		"proof:while:",
		`"liveness"`,
		`"allocation"`,
		`"spills": {}`,
	} {
		if !strings.Contains(string(backendRaw), want) {
			t.Fatalf("backend report missing %s:\n%s", want, backendRaw)
		}
	}
	boundsRaw, err := os.ReadFile(outPath + ".bounds.json")
	if err != nil {
		t.Fatalf("read bounds report: %v", err)
	}
	if !strings.Contains(string(boundsRaw), `"reason": "removed_by_while_range"`) {
		t.Fatalf("bounds report missing removed while range reason:\n%s", boundsRaw)
	}
	cmd := exec.Command(outPath)
	runOut, runErr := cmd.CombinedOutput()
	if string(runOut) != "" {
		t.Fatalf("runtime stdout mismatch: %q", runOut)
	}
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok {
		t.Fatalf("runtime exit = %v, want exit status 6", runErr)
	}
	if exitErr.ExitCode() != 6 {
		t.Fatalf("runtime exit code = %d, want 6", exitErr.ExitCode())
	}
}

func TestBuildExplainReportsMachineCallBackendPath(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	outPath := filepath.Join(dir, "app")
	src := `
func inc(x: Int) -> Int:
    return x + 1

func main() -> Int:
    return inc(inc(40))
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:    1,
		Explain: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}

	raw, err := os.ReadFile(outPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	for _, want := range []string{
		"machine-ir-call",
		"call inc",
		"abi:sysv",
		"clobbers:rax,rcx,rdx,rsi,rdi,r8,r9,r10,r11",
		`"liveness"`,
		`"allocation"`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("backend report missing %s:\n%s", want, raw)
		}
	}
	cmd := exec.Command(outPath)
	runOut, runErr := cmd.CombinedOutput()
	if string(runOut) != "" {
		t.Fatalf("runtime stdout mismatch: %q", runOut)
	}
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok {
		t.Fatalf("runtime exit = %v, want exit status 42", runErr)
	}
	if exitErr.ExitCode() != 42 {
		t.Fatalf("runtime exit code = %d, want 42", exitErr.ExitCode())
	}
}

func TestBackendReportOnlyExplainShowsSelectionFallbackPaths(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "app.tetra")
	plainOutPath := filepath.Join(dir, "plain")
	explainOutPath := filepath.Join(dir, "explain")
	src := `
func add(a: Int, b: Int) -> Int:
    return a + b

func get(xs: []i32, i: Int) -> Int
uses mem:
    return xs[i]

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 40
    return add(get(xs, 0), 2)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(
		srcPath,
		plainOutPath,
		"linux-x64",
		compiler.BuildOptions{Jobs: 1},
	); err != nil {
		t.Fatalf("plain BuildFileWithStatsOpt: %v", err)
	}
	if _, err := os.Stat(plainOutPath + ".backend.json"); !os.IsNotExist(err) {
		t.Fatalf(
			"plain build backend report stat = %v, want no backend report without --explain",
			err,
		)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, explainOutPath, "linux-x64", compiler.BuildOptions{
		Jobs:    1,
		Explain: true,
	}); err != nil {
		t.Fatalf("explain BuildFileWithStatsOpt: %v", err)
	}
	raw, err := os.ReadFile(explainOutPath + ".backend.json")
	if err != nil {
		t.Fatalf("read backend report: %v", err)
	}
	for _, want := range []string{
		`"backend_path": "register"`,
		`"backend_path": "stack"`,
		`"function": "add"`,
		`"function": "get"`,
		`"reason": "unsupported_or_unproven_subset_uses_stack_fallback"`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("backend selection report missing %s:\n%s", want, raw)
		}
	}
	cmd := exec.Command(explainOutPath)
	runOut, runErr := cmd.CombinedOutput()
	if string(runOut) != "" {
		t.Fatalf("runtime stdout mismatch: %q", runOut)
	}
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok {
		t.Fatalf("runtime exit = %v, want exit status 42", runErr)
	}
	if exitErr.ExitCode() != 42 {
		t.Fatalf("runtime exit code = %d, want 42", exitErr.ExitCode())
	}
}

// ---- plir_api_test.go ----

func TestCompilerBuildPLIRFormatsSliceLoopFacts(t *testing.T) {
	prog, err := compiler.Parse([]byte(`
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    for x in xs:
        total = total + x
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 1
    return sum(xs)
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	plirProg, err := compiler.BuildPLIR(checked)
	if err != nil {
		t.Fatalf("BuildPLIR: %v", err)
	}
	dump := compiler.FormatPLIR(plirProg)
	for _, want := range []string{"func sum", "fact index_in_range", "range: 0..xs.len"} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
}

// ---- ram_contract_build_test.go ----

func TestBuildFailIfHeapRejectsHeapFallback(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "heap.tetra")
	outPath := filepath.Join(dir, "app")
	src := `func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(5000)
    xs[0] = 7
    return xs[0]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:       1,
		FailIfHeap: true,
	})
	if err == nil || !strings.Contains(err.Error(), "RAM_CONTRACT_HEAP") {
		t.Fatalf("BuildFileWithStatsOpt error = %v, want RAM_CONTRACT_HEAP", err)
	}
}

func TestBuildEmitRAMContractReportWritesValidatedReports(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	outPath := filepath.Join(dir, "app")
	src := `func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 7
    xs[1] = 8
    return xs[0] + xs[1]
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:                  1,
		EmitRAMContractReport: true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt: %v", err)
	}
	for _, suffix := range []string{
		".ram-contract.json",
		".memory-grade.json",
		".proof-store-summary.json",
		".validation-pipeline-coverage.json",
	} {
		if _, err := os.Stat(outPath + suffix); err != nil {
			t.Fatalf("missing RAM report %s: %v", outPath+suffix, err)
		}
	}
}
