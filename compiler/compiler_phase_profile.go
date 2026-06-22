package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

const compilerPhaseProfileSchema = "tetra.compiler.phase-profile.v1"

type compilerPhaseProfiler struct {
	report compilerPhaseProfileReport
}

var compilerProcessMemoryRelease = releaseCompilerProcessMemory

func releaseCompilerProcessMemory() {
	runtime.GC()
	debug.FreeOSMemory()
}

type compilerPhaseProfileReport struct {
	Schema                      string                      `json:"schema"`
	Target                      string                      `json:"target"`
	InputPath                   string                      `json:"input_path,omitempty"`
	OutputPath                  string                      `json:"output_path,omitempty"`
	ReportMode                  string                      `json:"report_mode"`
	RequestedJobs               int                         `json:"requested_jobs"`
	WorkerCount                 int                         `json:"worker_count"`
	WorkerReason                string                      `json:"worker_reason"`
	MemoryBudgetBytes           int64                       `json:"memory_budget_bytes,omitempty"`
	ModuleCount                 int                         `json:"module_count"`
	ModulesToCompile            int                         `json:"modules_to_compile"`
	CacheHitCount               int                         `json:"cache_hit_count"`
	CompiledModuleCount         int                         `json:"compiled_module_count"`
	LoweredModuleCount          int                         `json:"lowered_module_count"`
	ObjectCount                 int                         `json:"object_count"`
	IRFunctionCount             int                         `json:"ir_function_count"`
	SourceFileCount             int                         `json:"source_file_count"`
	CheckedFunctionCount        int                         `json:"checked_function_count"`
	CheckedTypeCount            int                         `json:"checked_type_count"`
	TransientIRFunctionCount    int                         `json:"transient_ir_function_count"`
	AllocationPlanFunctionCount int                         `json:"allocation_plan_function_count"`
	GoHeapPeakAllocBytes        uint64                      `json:"go_heap_peak_alloc_bytes"`
	RSSPeakBytes                uint64                      `json:"rss_peak_bytes"`
	RSSSupported                bool                        `json:"rss_supported"`
	Phases                      []compilerPhaseProfilePhase `json:"phases"`
	Notes                       []string                    `json:"notes,omitempty"`
}

type compilerPhaseProfilePhase struct {
	Name                        string `json:"name"`
	UnixNano                    int64  `json:"unix_nano"`
	GoHeapAllocBytes            uint64 `json:"go_heap_alloc_bytes"`
	GoHeapSysBytes              uint64 `json:"go_heap_sys_bytes"`
	RSSCurrentBytes             uint64 `json:"rss_current_bytes"`
	RSSSupported                bool   `json:"rss_supported"`
	ModuleCount                 int    `json:"module_count"`
	ModulesToCompile            int    `json:"modules_to_compile"`
	CacheHitCount               int    `json:"cache_hit_count"`
	CompiledModuleCount         int    `json:"compiled_module_count"`
	LoweredModuleCount          int    `json:"lowered_module_count"`
	ObjectCount                 int    `json:"object_count"`
	IRFunctionCount             int    `json:"ir_function_count"`
	SourceFileCount             int    `json:"source_file_count"`
	CheckedFunctionCount        int    `json:"checked_function_count"`
	CheckedTypeCount            int    `json:"checked_type_count"`
	TransientIRFunctionCount    int    `json:"transient_ir_function_count"`
	AllocationPlanFunctionCount int    `json:"allocation_plan_function_count"`
}

type compilerPhaseProfileCounts struct {
	ModuleCount                    int
	ModulesToCompile               int
	CacheHitCount                  int
	CompiledModuleCount            int
	LoweredModuleCount             int
	ObjectCount                    int
	SetObjectCount                 bool
	IRFunctionCount                int
	SourceFileCount                int
	SetSourceFileCount             bool
	CheckedFunctionCount           int
	SetCheckedFunctionCount        bool
	CheckedTypeCount               int
	SetCheckedTypeCount            bool
	TransientIRFunctionCount       int
	SetTransientIRFunctionCount    bool
	AllocationPlanFunctionCount    int
	SetAllocationPlanFunctionCount bool
}

func newCompilerPhaseProfiler(
	inputPath string,
	outputPath string,
	target string,
	opt BuildOptions,
) *compilerPhaseProfiler {
	if !opt.EmitCompilerPhaseReport {
		return nil
	}
	return &compilerPhaseProfiler{
		report: compilerPhaseProfileReport{
			Schema:            compilerPhaseProfileSchema,
			Target:            target,
			InputPath:         filepath.ToSlash(inputPath),
			OutputPath:        filepath.ToSlash(outputPath),
			ReportMode:        compilerPhaseProfileReportMode(opt),
			RequestedJobs:     opt.Jobs,
			MemoryBudgetBytes: opt.MemoryBudgetBytes,
			Notes: []string{
				"compiler phase profile records local process snapshots for P7 diagnosis",
				"no compiler RSS reduction or cross-host RSS budget claim is made by this artifact",
			},
		},
	}
}

func compilerPhaseProfileReportMode(opt BuildOptions) string {
	if opt.Explain {
		return "explain"
	}
	var modes []string
	if opt.EmitPLIR {
		modes = append(modes, "plir")
	}
	if opt.EmitProof {
		modes = append(modes, "proof")
	}
	if opt.EmitBoundsReport {
		modes = append(modes, "bounds")
	}
	if opt.EmitAllocReport {
		modes = append(modes, "alloc")
	}
	if opt.EmitMemoryReport {
		modes = append(modes, "memory")
	}
	if opt.EmitRAMContractReport {
		modes = append(modes, "ram_contract")
	}
	if len(modes) == 0 {
		return "off"
	}
	return strings.Join(modes, "+")
}

func (p *compilerPhaseProfiler) setWorkerDecision(count int, reason string) {
	if p == nil {
		return
	}
	p.report.WorkerCount = count
	p.report.WorkerReason = strings.TrimSpace(reason)
}

func (p *compilerPhaseProfiler) capture(name string, counts compilerPhaseProfileCounts) {
	if p == nil {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	p.mergeCounts(counts)
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	rssBytes, rssOK := readCompilerProcessRSSBytes()
	if mem.Alloc > p.report.GoHeapPeakAllocBytes {
		p.report.GoHeapPeakAllocBytes = mem.Alloc
	}
	if rssOK && rssBytes > p.report.RSSPeakBytes {
		p.report.RSSPeakBytes = rssBytes
	}
	if rssOK {
		p.report.RSSSupported = true
	}
	p.report.Phases = append(p.report.Phases, compilerPhaseProfilePhase{
		Name:                        name,
		UnixNano:                    time.Now().UnixNano(),
		GoHeapAllocBytes:            mem.Alloc,
		GoHeapSysBytes:              mem.HeapSys,
		RSSCurrentBytes:             rssBytes,
		RSSSupported:                rssOK,
		ModuleCount:                 p.report.ModuleCount,
		ModulesToCompile:            p.report.ModulesToCompile,
		CacheHitCount:               p.report.CacheHitCount,
		CompiledModuleCount:         p.report.CompiledModuleCount,
		LoweredModuleCount:          p.report.LoweredModuleCount,
		ObjectCount:                 p.report.ObjectCount,
		IRFunctionCount:             p.report.IRFunctionCount,
		SourceFileCount:             p.report.SourceFileCount,
		CheckedFunctionCount:        p.report.CheckedFunctionCount,
		CheckedTypeCount:            p.report.CheckedTypeCount,
		TransientIRFunctionCount:    p.report.TransientIRFunctionCount,
		AllocationPlanFunctionCount: p.report.AllocationPlanFunctionCount,
	})
}

func (p *compilerPhaseProfiler) addNote(note string) {
	if p == nil {
		return
	}
	note = strings.TrimSpace(note)
	if note == "" {
		return
	}
	p.report.Notes = append(p.report.Notes, note)
}

func (p *compilerPhaseProfiler) mergeCounts(counts compilerPhaseProfileCounts) {
	if counts.ModuleCount >= 0 && counts.ModuleCount > 0 {
		p.report.ModuleCount = counts.ModuleCount
	}
	if counts.ModulesToCompile >= 0 && counts.ModulesToCompile > 0 {
		p.report.ModulesToCompile = counts.ModulesToCompile
	}
	if counts.CacheHitCount >= 0 && counts.CacheHitCount > 0 {
		p.report.CacheHitCount = counts.CacheHitCount
	}
	if counts.CompiledModuleCount >= 0 && counts.CompiledModuleCount > 0 {
		p.report.CompiledModuleCount = counts.CompiledModuleCount
	}
	if counts.LoweredModuleCount >= 0 && counts.LoweredModuleCount > 0 {
		p.report.LoweredModuleCount = counts.LoweredModuleCount
	}
	if counts.SetObjectCount || counts.ObjectCount > 0 {
		p.report.ObjectCount = counts.ObjectCount
	}
	if counts.IRFunctionCount >= 0 && counts.IRFunctionCount > 0 {
		p.report.IRFunctionCount = counts.IRFunctionCount
	}
	if counts.SetSourceFileCount || counts.SourceFileCount > 0 {
		p.report.SourceFileCount = counts.SourceFileCount
	}
	if counts.SetCheckedFunctionCount || counts.CheckedFunctionCount > 0 {
		p.report.CheckedFunctionCount = counts.CheckedFunctionCount
	}
	if counts.SetCheckedTypeCount || counts.CheckedTypeCount > 0 {
		p.report.CheckedTypeCount = counts.CheckedTypeCount
	}
	if counts.SetTransientIRFunctionCount || counts.TransientIRFunctionCount > 0 {
		p.report.TransientIRFunctionCount = counts.TransientIRFunctionCount
	}
	if counts.SetAllocationPlanFunctionCount || counts.AllocationPlanFunctionCount > 0 {
		p.report.AllocationPlanFunctionCount = counts.AllocationPlanFunctionCount
	}
}

func (p *compilerPhaseProfiler) write(path string) (string, error) {
	if p == nil {
		return "", nil
	}
	if strings.TrimSpace(p.report.WorkerReason) == "" {
		p.report.WorkerReason = "not_recorded"
	}
	if p.report.WorkerCount < 0 {
		p.report.WorkerCount = 0
	}
	outPath := strings.TrimSpace(path)
	if outPath == "" {
		outPath = p.report.OutputPath + ".compiler-profile.json"
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return "", err
	}
	if err := writeReport(outPath, p.report); err != nil {
		return "", err
	}
	return outPath, nil
}

func compilerPhaseProfilePath(outputPath string, opt BuildOptions) string {
	if strings.TrimSpace(opt.CompilerPhaseReportPath) != "" {
		return opt.CompilerPhaseReportPath
	}
	return outputPath + ".compiler-profile.json"
}
