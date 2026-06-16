package compiler_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

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
			for _, want := range []string{"planned_storage: Eliminated", "actual_lowering_storage:", "length_status: valid_empty_allocation", "zero_guard: valid_empty_no_allocator"} {
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
			if site.PlannedStorage != "Stack" || site.ActualLoweringStorage != "Stack" || site.LoweringStatus != "stack_lowering" {
				t.Fatalf("stack allocation report site = %+v, want Stack/Stack stack_lowering", site)
			}
			if site.RuntimePath != "stack_frame" || site.BytesRequested != 16 || site.BytesReserved != 16 {
				t.Fatalf("stack allocation runtime report site = %+v, want stack_frame bytes 16/16", site)
			}
			if !strings.Contains(site.Reason, "fixed_small_no_escape") {
				t.Fatalf("stack allocation reason = %q, want fixed_small_no_escape evidence", site.Reason)
			}
			text, err := os.ReadFile(outPath + ".alloc.txt")
			if err != nil {
				t.Fatalf("read alloc text report: %v", err)
			}
			for _, want := range []string{"planned_storage: Stack", "actual_lowering_storage: Stack", "lowering_status: stack_lowering", "runtime_path: stack_frame", "bytes_requested: 16", "bytes_reserved: 16", "totals allocation_count:", "runtime_paths:stack_frame="} {
				if !strings.Contains(string(text), want) {
					t.Fatalf("alloc text report missing %q:\n%s", want, text)
				}
			}
			return
		}
	}
	t.Fatalf("alloc report missing main/xs stack allocation: %+v", alloc.Functions)
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
		{"windows-x64", "build_lower_only unless run", "build_lower_only_target_host_required", true},
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
				t.Fatalf("%s target_memory_claim_level = %q, want %q", tt.target, alloc.TargetMemoryClaimLevel, tt.claimLevel)
			}
			if alloc.StorageEvidenceScope != tt.evidenceScope {
				t.Fatalf("%s storage_evidence_scope = %q, want %q", tt.target, alloc.StorageEvidenceScope, tt.evidenceScope)
			}
			stackLowered := alloc.Summary.ActualLoweringStorageClasses["Stack"] > 0 || alloc.Summary.RuntimePaths["stack_frame"] > 0
			if stackLowered != tt.wantStackLower {
				t.Fatalf("%s stack lowering evidence = %v from summary %+v, want %v", tt.target, stackLowered, alloc.Summary, tt.wantStackLower)
			}
			if tt.target != "linux-x64" && alloc.TargetMemoryClaimLevel == "production/host_runtime" {
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
		alloc.Summary.RuntimePaths["region"] == 0 {
		t.Fatalf("function-temp region summary missing storage/runtime counts: %+v", alloc.Summary)
	}
	if len(alloc.Summary.Regions) == 0 {
		t.Fatalf("function-temp region summary missing region rows: %+v", alloc.Summary)
	}
	for _, region := range alloc.Summary.Regions {
		if region.RegionID == "region:main:temp" &&
			region.Lifetime == "function:main" &&
			region.StorageClass == "FunctionTempRegion" &&
			region.RuntimePath == "region" &&
			region.AllocationCount == 1 {
			goto foundRegion
		}
	}
	t.Fatalf("function-temp region summary rows = %+v, want region:main:temp FunctionTempRegion", alloc.Summary.Regions)

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
				t.Fatalf("function-temp allocation report site = %+v, want FunctionTempRegion/FunctionTempRegion", site)
			}
			if site.RuntimePath != "region" || site.AllocatorClass != "function_temp_region" || site.RegionID != "region:main:temp" || site.Lifetime != "function:main" {
				t.Fatalf("function-temp runtime report site = %+v, want region evidence", site)
			}
			text, err := os.ReadFile(outPath + ".alloc.txt")
			if err != nil {
				t.Fatalf("read alloc text report: %v", err)
			}
			for _, want := range []string{"planned_storage: FunctionTempRegion", "actual_lowering_storage: FunctionTempRegion", "lowering_status: function_temp_region_lowering", "runtime_path: region", "allocator_class: function_temp_region", "region_id: region:main:temp", "lifetime: function:main"} {
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
	for _, want := range []string{"borrowed_imm", "no_escape", "derived_window", "owned", "provenance_known"} {
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
			if row.LoweredArtifactID == "" || row.ClaimLevel != "validated" || row.ValidatorStatus != "pass" {
				t.Fatalf("allocation_base_metadata row = %+v, want lowered artifact and validated/pass", row)
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
