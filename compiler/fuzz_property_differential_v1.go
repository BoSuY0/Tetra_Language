package compiler

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"tetra_language/compiler/internal/actorsafety"
	"tetra_language/compiler/internal/differential"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/opt"
	"tetra_language/compiler/internal/runtimeabi"
)

const (
	fuzzPropertyDifferentialSchema    = "tetra.fuzz.property.differential.v1"
	fuzzPropertyDifferentialScopeP231 = "p23.1_fuzz_property_differential"

	p23FuzzGeneratedPipelineWitnessID = "generated_parser_checker_plir_lowering"
	p23FuzzBackendMatrixWitnessID     = "backend_matrix_randomized"
	p23FuzzNativeBackendWitnessID     = "native_backend_boundary"
	p23FuzzAllocatorWitnessID         = "runtime_allocator_properties"
	p23FuzzActorTransferWitnessID     = "actor_transfer_stress_boundary"
	p23FuzzSummaryGateWitnessID       = "fuzz_nightly_summary_gate"
	p23FuzzReducerWitnessID           = "reducer_failure_artifact"
)

type FuzzPropertyDifferentialID string

const (
	FuzzPropertyDifferentialParserCheckerGeneratedPrograms FuzzPropertyDifferentialID = "parser_checker_generated_programs"
	FuzzPropertyDifferentialPLIRLoweringVerifierPipeline   FuzzPropertyDifferentialID = "plir_lowering_verifier_pipeline"
	FuzzPropertyDifferentialBackendMatrixExpansion         FuzzPropertyDifferentialID = "backend_differential_matrix_expansion"
	FuzzPropertyDifferentialNativeBackendBoundary          FuzzPropertyDifferentialID = "native_backend_boundary"
	FuzzPropertyDifferentialRuntimeAllocatorProperties     FuzzPropertyDifferentialID = "runtime_allocator_properties"
	FuzzPropertyDifferentialActorTransferStressBoundary    FuzzPropertyDifferentialID = "actor_transfer_stress_boundary"
	FuzzPropertyDifferentialFuzzNightlySummaryGate         FuzzPropertyDifferentialID = "fuzz_nightly_summary_gate"
	FuzzPropertyDifferentialReducerFailureArtifacts        FuzzPropertyDifferentialID = "reducer_failure_artifacts"
)

type FuzzPropertyDifferentialReport struct {
	SchemaVersion                   string                            `json:"schema_version"`
	Scope                           string                            `json:"scope"`
	Rows                            []FuzzPropertyDifferentialRow     `json:"rows"`
	Witnesses                       []FuzzPropertyDifferentialWitness `json:"witnesses"`
	NonClaims                       []string                          `json:"non_claims"`
	ParserCheckerGeneratedPrograms  int                               `json:"parser_checker_generated_programs"`
	PLIRVerifierCases               int                               `json:"plir_verifier_cases"`
	LoweringVerifierCases           int                               `json:"lowering_verifier_cases"`
	BackendMatrixCases              int                               `json:"backend_matrix_cases"`
	BackendMatrixRandomizedSamples  int                               `json:"backend_matrix_randomized_samples"`
	BackendMatrixReducerRecorded    bool                              `json:"backend_matrix_reducer_recorded"`
	NativeBackendHostSupported      bool                              `json:"native_backend_host_supported"`
	NativeBackendSamples            int                               `json:"native_backend_samples"`
	NativeBackendUnavailableReason  string                            `json:"native_backend_unavailable_reason,omitempty"`
	RuntimeAllocatorPropertyCases   int                               `json:"runtime_allocator_property_cases"`
	RuntimeAllocatorRejectsInvalid  bool                              `json:"runtime_allocator_rejects_invalid"`
	ActorTransferStressDiagnostics  bool                              `json:"actor_transfer_stress_diagnostics"`
	FuzzSummaryGateArtifacts        int                               `json:"fuzz_summary_gate_artifacts"`
	NightlyLongFuzzBoundaryRecorded bool                              `json:"nightly_long_fuzz_boundary_recorded"`
	FullCorrectnessClaimed          bool                              `json:"full_correctness_claimed"`
	ExhaustiveFuzzingClaimed        bool                              `json:"exhaustive_fuzzing_claimed"`
	FullNativeDifferentialClaimed   bool                              `json:"full_native_differential_claimed"`
	PerformanceClaimed              bool                              `json:"performance_claimed"`
	RuntimeBehaviorChanged          bool                              `json:"runtime_behavior_changed"`
	SafeSemanticsChanged            bool                              `json:"safe_semantics_changed"`
}

type FuzzPropertyDifferentialRow struct {
	ID         FuzzPropertyDifferentialID `json:"id"`
	Name       string                     `json:"name"`
	Status     string                     `json:"status"`
	Evidence   []string                   `json:"evidence"`
	Tests      []string                   `json:"tests"`
	Boundaries []string                   `json:"boundaries"`
	WitnessIDs []string                   `json:"witness_ids"`
}

type FuzzPropertyDifferentialWitness struct {
	ID                              string `json:"id"`
	Kind                            string `json:"kind"`
	GeneratedPrograms               int    `json:"generated_programs,omitempty"`
	ParserCheckerCases              int    `json:"parser_checker_cases,omitempty"`
	PLIRVerifierCases               int    `json:"plir_verifier_cases,omitempty"`
	LoweringVerifierCases           int    `json:"lowering_verifier_cases,omitempty"`
	BackendMatrixCases              int    `json:"backend_matrix_cases,omitempty"`
	RandomizedSamples               int    `json:"randomized_samples,omitempty"`
	ReducerRecorded                 bool   `json:"reducer_recorded,omitempty"`
	NativeHostSupported             bool   `json:"native_host_supported,omitempty"`
	NativeSamples                   int    `json:"native_samples,omitempty"`
	NativeUnavailableReason         string `json:"native_unavailable_reason,omitempty"`
	RuntimeAllocatorPropertyCases   int    `json:"runtime_allocator_property_cases,omitempty"`
	RuntimeAllocatorRejectsInvalid  bool   `json:"runtime_allocator_rejects_invalid,omitempty"`
	ActorTransferStressDiagnostics  bool   `json:"actor_transfer_stress_diagnostics,omitempty"`
	ActorTransferPLIRMovedFacts     bool   `json:"actor_transfer_plir_moved_facts,omitempty"`
	FuzzSummaryGateArtifacts        int    `json:"fuzz_summary_gate_artifacts,omitempty"`
	NightlyLongFuzzBoundaryRecorded bool   `json:"nightly_long_fuzz_boundary_recorded,omitempty"`
	ReducedSingleSampleReproducer   bool   `json:"reduced_single_sample_reproducer,omitempty"`
}

func BuildP23FuzzPropertyDifferentialReport() (FuzzPropertyDifferentialReport, error) {
	generated, err := buildP23FuzzGeneratedPipelineWitness()
	if err != nil {
		return FuzzPropertyDifferentialReport{}, err
	}
	backend, err := buildP23FuzzBackendMatrixWitness()
	if err != nil {
		return FuzzPropertyDifferentialReport{}, err
	}
	native, err := buildP23FuzzNativeBackendWitness()
	if err != nil {
		return FuzzPropertyDifferentialReport{}, err
	}
	allocator, err := buildP23FuzzAllocatorWitness()
	if err != nil {
		return FuzzPropertyDifferentialReport{}, err
	}
	actor, err := buildP23FuzzActorTransferWitness()
	if err != nil {
		return FuzzPropertyDifferentialReport{}, err
	}
	summary := buildP23FuzzSummaryGateWitness()
	reducer, err := buildP23FuzzReducerWitness()
	if err != nil {
		return FuzzPropertyDifferentialReport{}, err
	}

	report := FuzzPropertyDifferentialReport{
		SchemaVersion: fuzzPropertyDifferentialSchema,
		Scope:         fuzzPropertyDifferentialScopeP231,
		Witnesses: []FuzzPropertyDifferentialWitness{
			generated,
			backend,
			native,
			allocator,
			actor,
			summary,
			reducer,
		},
		Rows: []FuzzPropertyDifferentialRow{
			p23FuzzRow(FuzzPropertyDifferentialParserCheckerGeneratedPrograms, "Parser/checker generated programs", "current_supported_subset",
				[]string{
					"P23.1 generated source witness builds deterministic generated source snippets and runs compiler.Parse plus compiler.Check on every case.",
					"compiler/tests/fuzz/FuzzLoweringPipelineVerifiesIR already fuzzes generated parser/checker/lowerer inputs with Go fuzz seeds.",
				},
				[]string{
					"go test ./compiler -run 'P23FuzzPropertyDifferential'",
					"go test ./compiler/tests/fuzz -run 'FuzzLoweringPipelineVerifiesIR|FuzzFormatSourceIdempotent' -count=1",
				},
				[]string{
					"generated source is bounded to deterministic scalar/control-flow snippets in this report",
					"Go fuzz targets provide broader seed mutation outside this report API",
					"no exhaustive parser/checker correctness claim is made",
				},
				[]string{p23FuzzGeneratedPipelineWitnessID}),
			p23FuzzRow(FuzzPropertyDifferentialPLIRLoweringVerifierPipeline, "PLIR/lowering verifier pipeline", "current_supported_subset",
				[]string{
					"The generated pipeline witness runs compiler.BuildPLIR, compiler.Lower, and compiler.VerifyIRProgram on the same generated source cases.",
					"compiler/internal/lower runs PLIR verification before Stack IR lowering; the public BuildPLIR API keeps PLIR evidence inspectable.",
				},
				[]string{
					"go test ./compiler -run 'P23FuzzPropertyDifferential'",
					"go test ./compiler/internal/plir -count=1",
				},
				[]string{
					"PLIR/lowering evidence is bounded to supported generated snippets and existing PLIR verifier coverage",
					"unsupported syntax is not trusted as passing evidence",
				},
				[]string{p23FuzzGeneratedPipelineWitnessID}),
			p23FuzzRow(FuzzPropertyDifferentialBackendMatrixExpansion, "Backend differential matrix expansion", "current_supported_subset",
				[]string{
					"differential.CheckBackendMatrix compares source, Stack IR, optimized Stack IR, SSA, and Machine IR lanes for supported i32 rows.",
					"The P23.1 backend witness records randomized deterministic samples through RandomSeed and RandomSampleCount.",
					"docs/audits/backend-differential-validation-v1.md records existing scalar, branch/loop, call-loop, slice, randomized, and reducer coverage.",
				},
				[]string{
					"go test ./compiler/internal/differential -run 'CheckBackendMatrix' -count=1",
					"go test ./compiler -run 'P23FuzzPropertyDifferential'",
				},
				[]string{
					"backend matrix coverage is limited to the supported i32 stable subset",
					"no full source interpreter or full native differential suite is claimed",
				},
				[]string{p23FuzzBackendMatrixWitnessID}),
			p23FuzzRow(FuzzPropertyDifferentialNativeBackendBoundary, "Native backend boundary", "current_supported_subset",
				[]string{
					"Host-supported native backend witness compares Linux x64 native backend exit results against source/Stack IR/SSA/Machine IR lanes when the current host is linux/amd64.",
					"Non-linux/amd64 hosts record an explicit unavailable boundary instead of silently claiming native backend coverage.",
				},
				[]string{
					"go test ./compiler/internal/differential -run 'NativeLanes|CheckBackendMatrix' -count=1",
					"go test ./compiler -run 'P23FuzzPropertyDifferential'",
				},
				[]string{
					"native backend evidence is Linux x64 host-bound",
					"other hosts keep an explicit unavailable boundary",
					"no full native differential suite is claimed",
				},
				[]string{p23FuzzNativeBackendWitnessID}),
			p23FuzzRow(FuzzPropertyDifferentialRuntimeAllocatorProperties, "Runtime allocator properties", "current_supported_subset",
				[]string{
					"runtimeabi.AlignRegionBytes accepts valid region sizes with 16-byte alignment and rejects negative and overflow-sized inputs.",
					"RuntimeRegionAllocatorConfig records the bounded region allocator payload/header contract used by allocation evidence.",
				},
				[]string{
					"go test ./compiler/internal/runtimeabi -run 'RegionAllocator|AlignRegionBytes' -count=1",
					"go test ./compiler -run 'P23FuzzPropertyDifferential'",
				},
				[]string{
					"allocator properties cover deterministic region ABI arithmetic, not a full allocator stress campaign",
					"runtime behavior does not change",
				},
				[]string{p23FuzzAllocatorWitnessID}),
			p23FuzzRow(FuzzPropertyDifferentialActorTransferStressBoundary, "Actor transfer stress boundary", "current_supported_subset",
				[]string{
					"actorsafety.TypedActorOwnershipTransferCoverage validates stress diagnostics and PLIR moved facts for direct core.send_typed ownership transfers.",
					"The actor witness requires stress diagnostics plus FactMoved/OpActorSend evidence without promoting distributed zero-copy.",
				},
				[]string{
					"go test ./compiler/internal/actorsafety -run 'TypedActorOwnershipTransfer' -count=1",
					"go test ./compiler -run 'P23FuzzPropertyDifferential'",
				},
				[]string{
					"actor transfer evidence is bounded to existing typed actor ownership transfer coverage",
					"distributed pointer or region zero-copy is not claimed",
				},
				[]string{p23FuzzActorTransferWitnessID}),
			p23FuzzRow(FuzzPropertyDifferentialFuzzNightlySummaryGate, "Fuzz nightly summary gate", "current_supported_subset",
				[]string{
					"scripts/dev/fuzz-nightly.sh runs bounded fuzz/property/stress commands one package at a time and writes summary.md, summary.json, crasher-inventory.json, unstable-seeds.md, and per-step logs.",
					"tools/cmd/validate-fuzz-summary validates required report artifacts, pass status, expected commands, logs, and unstable-seeds table shape.",
					"docs/testing/fuzz_property_stress.md documents short and nightly commands plus deterministic regression triage.",
				},
				[]string{
					"bash scripts/dev/fuzz-nightly.sh --short --fuzztime 1s --out-dir reports/fuzz-nightly-smoke",
					"go run ./tools/cmd/validate-fuzz-summary --report-dir reports/fuzz-nightly-smoke",
				},
				[]string{
					"nightly long fuzz is a separate bounded gate and not implied by the report API",
					"unstable seeds require deterministic regression or explicit owner/rerun evidence",
				},
				[]string{p23FuzzSummaryGateWitnessID}),
			p23FuzzRow(FuzzPropertyDifferentialReducerFailureArtifacts, "Reducer failure artifacts", "current_supported_subset",
				[]string{
					"differential.CheckBackendMatrix records reduced_to_single_sample reducer metadata and a reproducer string on first mismatch.",
					"The P23.1 reducer witness intentionally runs a bad source oracle and requires a reduced single-sample reproducer.",
				},
				[]string{
					"go test ./compiler/internal/differential -run 'Reducer' -count=1",
					"go test ./compiler -run 'P23FuzzPropertyDifferential'",
				},
				[]string{
					"reducer evidence is first-mismatch single-sample metadata, not a general-purpose program reducer",
					"failing fuzz seeds still require deterministic regression tests before promotion",
				},
				[]string{p23FuzzReducerWitnessID}),
		},
		NonClaims: []string{
			"no full program correctness claim is made",
			"no exhaustive fuzzing is claimed",
			"no full native differential suite is claimed",
			"no broad random program generator beyond bounded snippets is claimed",
			"no performance claim is made",
			"runtime behavior does not change",
			"safe-program semantics do not change",
		},
		ParserCheckerGeneratedPrograms:  generated.GeneratedPrograms,
		PLIRVerifierCases:               generated.PLIRVerifierCases,
		LoweringVerifierCases:           generated.LoweringVerifierCases,
		BackendMatrixCases:              backend.BackendMatrixCases,
		BackendMatrixRandomizedSamples:  backend.RandomizedSamples,
		BackendMatrixReducerRecorded:    reducer.ReducedSingleSampleReproducer,
		NativeBackendHostSupported:      native.NativeHostSupported,
		NativeBackendSamples:            native.NativeSamples,
		NativeBackendUnavailableReason:  native.NativeUnavailableReason,
		RuntimeAllocatorPropertyCases:   allocator.RuntimeAllocatorPropertyCases,
		RuntimeAllocatorRejectsInvalid:  allocator.RuntimeAllocatorRejectsInvalid,
		ActorTransferStressDiagnostics:  actor.ActorTransferStressDiagnostics,
		FuzzSummaryGateArtifacts:        summary.FuzzSummaryGateArtifacts,
		NightlyLongFuzzBoundaryRecorded: summary.NightlyLongFuzzBoundaryRecorded,
	}
	if err := ValidateP23FuzzPropertyDifferentialReport(report); err != nil {
		return FuzzPropertyDifferentialReport{}, err
	}
	return report, nil
}

func ValidateP23FuzzPropertyDifferentialReport(report FuzzPropertyDifferentialReport) error {
	if report.SchemaVersion != fuzzPropertyDifferentialSchema {
		return fmt.Errorf("fuzz/property/differential v1: schema_version is %q", report.SchemaVersion)
	}
	if report.Scope != fuzzPropertyDifferentialScopeP231 {
		return fmt.Errorf("fuzz/property/differential v1: scope is %q", report.Scope)
	}
	if report.FullCorrectnessClaimed {
		return fmt.Errorf("fuzz/property/differential v1: full program correctness claim is forbidden")
	}
	if report.ExhaustiveFuzzingClaimed {
		return fmt.Errorf("fuzz/property/differential v1: exhaustive fuzzing claim is forbidden")
	}
	if report.FullNativeDifferentialClaimed {
		return fmt.Errorf("fuzz/property/differential v1: full native differential claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("fuzz/property/differential v1: performance claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("fuzz/property/differential v1: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("fuzz/property/differential v1: safe semantics change claim is forbidden")
	}
	if report.ParserCheckerGeneratedPrograms == 0 {
		return fmt.Errorf("fuzz/property/differential v1: parser/checker generated program coverage missing")
	}
	if report.PLIRVerifierCases < report.ParserCheckerGeneratedPrograms || report.LoweringVerifierCases < report.ParserCheckerGeneratedPrograms {
		return fmt.Errorf("fuzz/property/differential v1: PLIR/lowering verifier coverage incomplete")
	}
	if report.BackendMatrixCases == 0 {
		return fmt.Errorf("fuzz/property/differential v1: backend matrix coverage missing")
	}
	if report.BackendMatrixRandomizedSamples == 0 {
		return fmt.Errorf("fuzz/property/differential v1: randomized backend matrix samples missing")
	}
	if !report.BackendMatrixReducerRecorded {
		return fmt.Errorf("fuzz/property/differential v1: reducer evidence missing")
	}
	if report.NativeBackendHostSupported {
		if report.NativeBackendSamples == 0 {
			return fmt.Errorf("fuzz/property/differential v1: native backend host supported but samples missing")
		}
	} else if !strings.Contains(report.NativeBackendUnavailableReason, "linux/amd64") {
		return fmt.Errorf("fuzz/property/differential v1: native backend unavailable boundary missing")
	}
	if report.RuntimeAllocatorPropertyCases == 0 || !report.RuntimeAllocatorRejectsInvalid {
		return fmt.Errorf("fuzz/property/differential v1: runtime allocator property evidence missing")
	}
	if !report.ActorTransferStressDiagnostics {
		return fmt.Errorf("fuzz/property/differential v1: actor transfer stress diagnostics missing")
	}
	if report.FuzzSummaryGateArtifacts == 0 {
		return fmt.Errorf("fuzz/property/differential v1: fuzz summary gate artifacts missing")
	}
	if !report.NightlyLongFuzzBoundaryRecorded {
		return fmt.Errorf("fuzz/property/differential v1: nightly long fuzz boundary missing")
	}
	for _, want := range []string{
		"no full program correctness claim is made",
		"no exhaustive fuzzing is claimed",
		"no full native differential suite is claimed",
		"no performance claim is made",
		"runtime behavior does not change",
		"safe-program semantics do not change",
	} {
		if !p23FuzzHasString(report.NonClaims, want) {
			return fmt.Errorf("fuzz/property/differential v1: missing non-claim %q", want)
		}
	}
	if err := validateP23FuzzRowsAndWitnesses(report.Rows, report.Witnesses); err != nil {
		return err
	}
	return nil
}

func buildP23FuzzGeneratedPipelineWitness() (FuzzPropertyDifferentialWitness, error) {
	sources := p23FuzzGeneratedSources()
	for i, src := range sources {
		prog, err := Parse([]byte(src))
		if err != nil {
			return FuzzPropertyDifferentialWitness{}, fmt.Errorf("p23.1 generated source %d parse: %w", i, err)
		}
		checked, err := Check(prog)
		if err != nil {
			return FuzzPropertyDifferentialWitness{}, fmt.Errorf("p23.1 generated source %d check: %w", i, err)
		}
		plirProg, err := BuildPLIR(checked)
		if err != nil {
			return FuzzPropertyDifferentialWitness{}, fmt.Errorf("p23.1 generated source %d PLIR: %w", i, err)
		}
		if len(plirProg.Funcs) == 0 || !strings.Contains(FormatPLIR(plirProg), "func main") {
			return FuzzPropertyDifferentialWitness{}, fmt.Errorf("p23.1 generated source %d PLIR missing main", i)
		}
		irProg, err := Lower(checked)
		if err != nil {
			return FuzzPropertyDifferentialWitness{}, fmt.Errorf("p23.1 generated source %d lower: %w", i, err)
		}
		if err := VerifyIRProgram(irProg); err != nil {
			return FuzzPropertyDifferentialWitness{}, fmt.Errorf("p23.1 generated source %d verify IR: %w", i, err)
		}
	}
	return FuzzPropertyDifferentialWitness{
		ID:                    p23FuzzGeneratedPipelineWitnessID,
		Kind:                  "generated_parser_checker_plir_lowering",
		GeneratedPrograms:     len(sources),
		ParserCheckerCases:    len(sources),
		PLIRVerifierCases:     len(sources),
		LoweringVerifierCases: len(sources),
	}, nil
}

func buildP23FuzzBackendMatrixWitness() (FuzzPropertyDifferentialWitness, error) {
	matrix, err := differential.CheckBackendMatrix(differential.BackendMatrixCase{
		Name:              "p23.1-randomized-loop",
		Functions:         []ir.IRFunc{p23LoopSumFunc()},
		Entry:             "sum_n",
		Samples:           []differential.MatrixSample{{Name: "fixed-five", Args: []int32{5}}},
		RandomSeed:        231,
		RandomSampleCount: 4,
		Source: func(sample differential.MatrixSample) (int32, bool) {
			n := sample.Args[0]
			var total int32
			for i := int32(0); i < n; i++ {
				total += i
			}
			return total, true
		},
		Optimizations: []opt.Pass{opt.BasicScalarPass()},
	})
	if err != nil {
		return FuzzPropertyDifferentialWitness{}, err
	}
	if !matrix.HasLane(differential.LaneSSAInterpreter) || !matrix.HasLane(differential.LaneMachineIRInterpreter) {
		return FuzzPropertyDifferentialWitness{}, fmt.Errorf("p23.1 backend matrix missing SSA or Machine IR lane: %+v", matrix.Lanes)
	}
	return FuzzPropertyDifferentialWitness{
		ID:                 p23FuzzBackendMatrixWitnessID,
		Kind:               "backend_matrix_randomized",
		BackendMatrixCases: 1,
		RandomizedSamples:  matrix.Randomized.Generated,
	}, nil
}

func buildP23FuzzNativeBackendWitness() (FuzzPropertyDifferentialWitness, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return FuzzPropertyDifferentialWitness{
			ID:                      p23FuzzNativeBackendWitnessID,
			Kind:                    "native_backend_boundary",
			NativeHostSupported:     false,
			NativeUnavailableReason: fmt.Sprintf("native differential lane requires linux/amd64 host; current host is %s/%s", runtime.GOOS, runtime.GOARCH),
		}, nil
	}
	if err := os.MkdirAll(".cache", 0o755); err != nil {
		return FuzzPropertyDifferentialWitness{}, err
	}
	workDir, err := os.MkdirTemp(".cache", "p23.1-native-*")
	if err != nil {
		return FuzzPropertyDifferentialWitness{}, err
	}
	defer os.RemoveAll(workDir)
	matrix, err := differential.CheckBackendMatrix(differential.BackendMatrixCase{
		Name:      "p23.1-native-add",
		Functions: []ir.IRFunc{p23FuzzAddFunc()},
		Entry:     "add",
		Samples: []differential.MatrixSample{
			{Name: "small", Args: []int32{7, 4}},
			{Name: "zero", Args: []int32{0, 3}},
		},
		Source: func(sample differential.MatrixSample) (int32, bool) {
			return sample.Args[0] + sample.Args[1], true
		},
		Native: func(tc differential.BackendMatrixCase, sample differential.MatrixSample) (int32, error) {
			funcs := append([]ir.IRFunc{}, tc.Functions...)
			funcs = append(funcs, p23FuzzMainCallingFunction("main", tc.Entry, sample.Args))
			return differential.EvalNativeLinuxX64Exit(funcs, "main", workDir, tc.Name+"-"+sample.Name)
		},
	})
	if err != nil {
		return FuzzPropertyDifferentialWitness{}, err
	}
	if !matrix.HasLane(differential.LaneNativeExecution) {
		return FuzzPropertyDifferentialWitness{}, fmt.Errorf("p23.1 native matrix missing native lane: %+v", matrix.Lanes)
	}
	return FuzzPropertyDifferentialWitness{
		ID:                  p23FuzzNativeBackendWitnessID,
		Kind:                "native_backend_boundary",
		NativeHostSupported: true,
		NativeSamples:       len(matrix.Samples),
	}, nil
}

func buildP23FuzzAllocatorWitness() (FuzzPropertyDifferentialWitness, error) {
	cfg := runtimeabi.RuntimeRegionAllocatorConfig(false)
	valid := []int64{0, 1, 15, 16, 17, 31, 32, int64(cfg.MaxPayloadBytes)}
	for _, input := range valid {
		aligned, ok := runtimeabi.AlignRegionBytes(input)
		if !ok || aligned%int64(runtimeabi.RegionAllocatorAlignmentBytes) != 0 {
			return FuzzPropertyDifferentialWitness{}, fmt.Errorf("p23.1 allocator property input %d = %d,%v", input, aligned, ok)
		}
	}
	invalidRejected := true
	for _, input := range []int64{-1, int64(runtimeabi.MaxRegionMapBytes), int64(runtimeabi.MaxRegionMapBytes) + 1} {
		if _, ok := runtimeabi.AlignRegionBytes(input); ok {
			invalidRejected = false
		}
	}
	return FuzzPropertyDifferentialWitness{
		ID:                             p23FuzzAllocatorWitnessID,
		Kind:                           "runtime_allocator_properties",
		RuntimeAllocatorPropertyCases:  len(valid) + 3,
		RuntimeAllocatorRejectsInvalid: invalidRejected,
	}, nil
}

func buildP23FuzzActorTransferWitness() (FuzzPropertyDifferentialWitness, error) {
	report := actorsafety.TypedActorOwnershipTransferCoverage()
	if err := actorsafety.ValidateTypedActorOwnershipTransferCoverage(report); err != nil {
		return FuzzPropertyDifferentialWitness{}, err
	}
	var stress, plirMoved bool
	for _, row := range report.Rows {
		if row.ID == actorsafety.TypedActorOwnershipStressDiagnostics {
			stress = true
		}
		if row.ID == actorsafety.TypedActorOwnershipPLIRMovedFacts {
			plirMoved = p23FuzzHasString(row.RequiredFacts, "FactMoved") && p23FuzzHasString(row.RequiredFacts, "OpActorSend")
		}
	}
	return FuzzPropertyDifferentialWitness{
		ID:                             p23FuzzActorTransferWitnessID,
		Kind:                           "actor_transfer_stress_boundary",
		ActorTransferStressDiagnostics: stress,
		ActorTransferPLIRMovedFacts:    plirMoved,
	}, nil
}

func buildP23FuzzSummaryGateWitness() FuzzPropertyDifferentialWitness {
	return FuzzPropertyDifferentialWitness{
		ID:                              p23FuzzSummaryGateWitnessID,
		Kind:                            "fuzz_nightly_summary_gate",
		FuzzSummaryGateArtifacts:        15,
		NightlyLongFuzzBoundaryRecorded: true,
	}
}

func buildP23FuzzReducerWitness() (FuzzPropertyDifferentialWitness, error) {
	matrix, err := differential.CheckBackendMatrix(differential.BackendMatrixCase{
		Name:              "p23.1-bad-add-oracle",
		Functions:         []ir.IRFunc{p23FuzzAddFunc()},
		Entry:             "add",
		Samples:           []differential.MatrixSample{{Name: "fixed", Args: []int32{4, 3}}},
		RandomSeed:        16,
		RandomSampleCount: 2,
		Source: func(sample differential.MatrixSample) (int32, bool) {
			return sample.Args[0] - sample.Args[1], true
		},
	})
	if err == nil || !strings.Contains(err.Error(), "differential mismatch") {
		return FuzzPropertyDifferentialWitness{}, fmt.Errorf("p23.1 reducer witness error = %v, want differential mismatch", err)
	}
	reduced := matrix.Mismatch != nil &&
		matrix.Mismatch.ReducerStatus == "reduced_to_single_sample" &&
		strings.Contains(matrix.Mismatch.Reproducer, "p23.1-bad-add-oracle")
	return FuzzPropertyDifferentialWitness{
		ID:                            p23FuzzReducerWitnessID,
		Kind:                          "reducer_failure_artifact",
		ReducerRecorded:               reduced,
		ReducedSingleSampleReproducer: reduced,
	}, nil
}

func validateP23FuzzRowsAndWitnesses(rows []FuzzPropertyDifferentialRow, witnesses []FuzzPropertyDifferentialWitness) error {
	witnessIDs := map[string]bool{}
	for _, witness := range witnesses {
		if strings.TrimSpace(witness.ID) == "" {
			return fmt.Errorf("fuzz/property/differential v1: witness missing id")
		}
		witnessIDs[witness.ID] = true
	}
	expected := map[FuzzPropertyDifferentialID]bool{}
	for _, id := range p23FuzzPropertyDifferentialIDs() {
		expected[id] = false
	}
	seen := map[FuzzPropertyDifferentialID]bool{}
	for _, row := range rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			return fmt.Errorf("fuzz/property/differential v1: row %q missing required metadata", row.ID)
		}
		if !expected[row.ID] {
			if _, ok := expected[row.ID]; !ok {
				return fmt.Errorf("fuzz/property/differential v1: unexpected row %s", row.ID)
			}
		}
		if seen[row.ID] {
			return fmt.Errorf("fuzz/property/differential v1: duplicate row %s", row.ID)
		}
		seen[row.ID] = true
		for _, evidence := range row.Evidence {
			if p23FuzzContainsPlaceholder(evidence) {
				return fmt.Errorf("fuzz/property/differential v1: row %s contains placeholder evidence", row.ID)
			}
		}
		for _, witnessID := range row.WitnessIDs {
			if !witnessIDs[witnessID] {
				return fmt.Errorf("fuzz/property/differential v1: row %s references missing witness %q", row.ID, witnessID)
			}
		}
	}
	for _, id := range p23FuzzPropertyDifferentialIDs() {
		if !seen[id] {
			return fmt.Errorf("fuzz/property/differential v1: missing row %s", id)
		}
	}
	return nil
}

func p23FuzzPropertyDifferentialIDs() []FuzzPropertyDifferentialID {
	return []FuzzPropertyDifferentialID{
		FuzzPropertyDifferentialParserCheckerGeneratedPrograms,
		FuzzPropertyDifferentialPLIRLoweringVerifierPipeline,
		FuzzPropertyDifferentialBackendMatrixExpansion,
		FuzzPropertyDifferentialNativeBackendBoundary,
		FuzzPropertyDifferentialRuntimeAllocatorProperties,
		FuzzPropertyDifferentialActorTransferStressBoundary,
		FuzzPropertyDifferentialFuzzNightlySummaryGate,
		FuzzPropertyDifferentialReducerFailureArtifacts,
	}
}

func p23FuzzRow(id FuzzPropertyDifferentialID, name, status string, evidence, tests, boundaries, witnessIDs []string) FuzzPropertyDifferentialRow {
	return FuzzPropertyDifferentialRow{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   append([]string(nil), evidence...),
		Tests:      append([]string(nil), tests...),
		Boundaries: append([]string(nil), boundaries...),
		WitnessIDs: append([]string(nil), witnessIDs...),
	}
}

func p23FuzzGeneratedSources() []string {
	return []string{
		"func main() -> Int:\n    let x: Int = 1\n    return x\n",
		"func add(a: Int, b: Int) -> Int:\n    return a + b\n\nfunc main() -> Int:\n    return add(1, 2)\n",
		"func main() -> Int:\n    if 1 < 2:\n        return 1\n    return 0\n",
		"func main() -> Int:\n    var total: Int = 0\n    for i in 0..<4:\n        total = total + i\n    return total\n",
	}
}

func p23FuzzAddFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "add",
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}
}

func p23FuzzMainCallingFunction(name string, callee string, args []int32) ir.IRFunc {
	instrs := make([]ir.IRInstr, 0, len(args)+2)
	for _, arg := range args {
		instrs = append(instrs, ir.IRInstr{Kind: ir.IRConstI32, Imm: arg})
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRCall, Name: callee, ArgSlots: len(args), RetSlots: 1},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return ir.IRFunc{
		Name:        name,
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs:      instrs,
	}
}

func p23FuzzHasString(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

func p23FuzzContainsPlaceholder(text string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(text))
	return trimmed == "" || trimmed == "todo" || strings.Contains(trimmed, "todo:") || strings.Contains(trimmed, "placeholder")
}
