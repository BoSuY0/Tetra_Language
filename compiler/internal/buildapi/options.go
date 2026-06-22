package buildapi

import "tetra_language/compiler/internal/module"

type BuildOptions struct {
	Jobs                             int
	IslandsDebug                     bool
	DebugInfo                        bool
	ReleaseOptimize                  bool
	Explain                          bool
	EmitPLIR                         bool
	EmitProof                        bool
	EmitAllocReport                  bool
	EmitBoundsReport                 bool
	EmitMemoryReport                 bool
	EmitRAMContractReport            bool
	EmitCompilerPhaseReport          bool
	EmitRuntimeHeapTelemetry         bool
	RuntimeHeapTelemetryActorDomains bool
	OwnedAllocDropLowering           bool
	CompilerPhaseReportPath          string
	RuntimeHeapTelemetryDir          string
	RuntimeHeapTelemetryProgram      string
	RuntimeHeapTelemetryMain         string
	FailIfHeap                       bool
	FailIfCopy                       bool
	FailIfUnbounded                  bool
	MemoryBudgetBytes                int64
	RAMContractFile                  string
	Emit                             EmitMode
	Runtime                          RuntimeMode
	RuntimeObjectPath                string
	SurfaceHostRequired              bool
	SurfaceHostDriver                string
	SurfaceHostProtocol              string
	SurfaceHostSocketPath            string
	LinkObjectPaths                  []string
	ProjectRoot                      string
	SourceRoots                      []string
	DependencyRoots                  []module.ModuleRoot
	InterfaceOnly                    bool
}
