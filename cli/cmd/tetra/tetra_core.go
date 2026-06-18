package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"tetra_language/cli/internal/actornet"
	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
	"tetra_language/internal/outputformat"
	"time"
)

// ---- actor_net.go ----

func runActorNet(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("actor-net", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", "127.0.0.1:0", "loopback address to listen on")
	report := fs.String("report", "", "optional JSON runtime report path")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprintln(stdout, "usage: tetra actor-net [--addr 127.0.0.1:PORT] [--report PATH]")
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "actor-net does not accept positional arguments")
		return 2
	}

	broker, err := actornet.NewBroker(actornet.Config{
		Addr:       *addr,
		ReportPath: *report,
	})
	if err != nil {
		fmt.Fprintf(stderr, "actor-net: %v\n", err)
		return 1
	}
	defer broker.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintf(stdout, "Actor network broker listening on %s\n", broker.Addr())
	if err := broker.Serve(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(stderr, "actor-net: %v\n", err)
		return 1
	}
	return 0
}

// ---- main.go ----

var commandLookPath = exec.LookPath
var webRunnerProbe = probeWebRunner
var execNativeProgram = execProgram
var execNativeSurfaceProgram = execSurfaceHostedProgram
var linuxX86HostSupport = canRunLinuxX86OnHost
var linuxX32HostSupport = canRunLinuxX32OnHost

var linuxX86ProbeOnce sync.Once
var linuxX86ProbeResult bool
var linuxX32ProbeOnce sync.Once
var linuxX32ProbeResult bool

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

const supportedTargetsHelp = ("linux-x64, windows-x64, macos-x64, wasm32-wasi, wasm32-" +
	"web; build-only: linux-x86, linux-x32")

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	if isHelpArgs(args) {
		printUsage(stdout)
		return 0
	}
	switch args[0] {
	case "version":
		if len(args) > 1 {
			if isHelpArgs(args[1:]) {
				fmt.Fprintln(stdout, "usage: tetra version")
				return 0
			}
			fmt.Fprintln(stderr, "version does not accept arguments")
			return 2
		}
		fmt.Fprintln(stdout, compiler.Version())
		return 0
	case "targets":
		return runTargets(args[1:], stdout, stderr)
	case "features":
		return runFeatures(args[1:], stdout, stderr)
	case "formats":
		return runFormats(args[1:], stdout, stderr)
	case "new":
		return runNew(args[1:], stdout, stderr)
	case "project":
		return runProject(args[1:], stdout, stderr)
	case "workspace":
		return runWorkspace(args[1:], stdout, stderr)
	case "interface":
		return runInterface(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "actor-net":
		return runActorNet(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "build":
		return runBuild(args[1:], stdout, stderr)
	case "run":
		return runRun(args[1:], stdout, stderr)
	case "smoke":
		return runSmoke(args[1:], stdout, stderr)
	case "surface":
		return runSurface(args[1:], stdout, stderr)
	case "fmt":
		return runFmt(args[1:], stdout, stderr)
	case "test":
		return runTest(args[1:], stdout, stderr)
	case "doc":
		return runDoc(args[1:], stdout, stderr)
	case "clean":
		return runClean(args[1:], stdout, stderr)
	case "eco":
		return runEco(args[1:], stdout, stderr)
	case "lsp":
		return runLSP(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func isHelpArgs(args []string) bool {
	return len(args) == 1 && (args[0] == "-h" || args[0] == "--help")
}

func runBuild(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String(
		"target",
		"",
		"target triple ("+supportedTargetsHelp+"); defaults to Capsule.t4 first target, then host",
	)
	out := fs.String("o", "", "output path")
	allTargets := fs.Bool("all-targets", false, "build every target listed in Capsule.t4")
	interfaceOnly := fs.Bool(
		"interface-only",
		false,
		"type-check interface/API graph without emitting executable code",
	)
	islandsDebug := fs.Bool("islands-debug", false, "enable islands debug runtime checks")
	emit := fs.String("emit", "exe", "emit mode: exe, object, or library")
	explain := fs.Bool(
		"explain",
		false,
		"write explain, proof, bounds, allocation, backend, layout, and perf reports",
	)
	emitPLIR := fs.Bool("emit-plir", false, "write PLIR JSON and text reports")
	emitProof := fs.Bool("emit-proof", false, "write proof report")
	emitAllocReport := fs.Bool("emit-alloc-report", false, "write allocation report")
	emitBoundsReport := fs.Bool("emit-bounds-report", false, "write bounds-check report")
	emitMemoryReport := fs.Bool(
		"emit-memory-report",
		false,
		"write schema-versioned memory fact report",
	)
	emitRAMContractReport := fs.Bool(
		"emit-ram-contract-report",
		false,
		"write RAM contract, memory grade, proof-store, pipeline, and blocker reports",
	)
	emitRuntimeHeapTelemetry := fs.Bool(
		"emit-runtime-heap-telemetry",
		false,
		"write linux-x64 runtime heap telemetry sidecars",
	)
	runtimeHeapTelemetryDir := fs.String(
		"runtime-heap-telemetry-dir",
		"",
		"directory for linux-x64 runtime heap telemetry sidecars",
	)
	failIfHeap := fs.Bool(
		"fail-if-heap",
		false,
		"fail build if RAM contract evidence contains heap placement",
	)
	failIfCopy := fs.Bool(
		"fail-if-copy",
		false,
		"fail build if RAM contract evidence contains a required copy",
	)
	failIfUnbounded := fs.Bool(
		"fail-if-unbounded",
		false,
		"fail build if RAM contract evidence contains unbounded or unknown memory",
	)
	memoryBudget := fs.String(
		"memory-budget",
		"",
		"fail build if RAM contract budget exceeds bytes; supports b, kb, mb, gb suffixes",
	)
	ramContractFile := fs.String("ram-contract", "", "optional JSON RAM contract file")
	runtimeMode := fs.String("runtime", "auto", "actors runtime: auto, selfhost, or builtin")
	runtimeObject := fs.String("runtime-object", "", "actors runtime object override")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	artifactsMode := fs.String("artifacts", "strict", "artifact handling: strict or auto")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	var linkObjects multiFlag
	fs.Var(&linkObjects, "link-object", "extra TOBJ object to link")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if *artifactsMode != "strict" && *artifactsMode != "auto" {
		writeValidationDiagnostic(stderr, *diagnostics, "build --artifacts must be strict or auto")
		return 2
	}

	input := ""
	if fs.NArg() > 0 {
		input = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "build accepts at most one input path")
		return 2
	}
	requestedInput := input

	input, worldOpt, projectCtx, err := resolveCLIInput(input)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if *artifactsMode == "auto" && projectCtx != nil && projectCtx.Found {
		if err := buildCapsuleArtifacts(projectCtx.CapsulePath, capsuleArtifactBuildOptions{
			Target:     *target,
			LockPath:   projectCtx.LockPath,
			Jobs:       *jobs,
			AllTargets: *allTargets,
		}); err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		fmt.Fprintf(stdout, "Artifacts repaired: %s\n", projectCtx.CapsulePath)
		input, worldOpt, projectCtx, err = resolveCLIInput(requestedInput)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
	}

	if *allTargets {
		targets := projectBuildTargets(projectCtx)
		if len(targets) == 0 {
			writeValidationDiagnostic(
				stderr,
				*diagnostics,
				"build --all-targets requires targets in Capsule.t4",
			)
			return 2
		}
		targetLinkObjects, err := projectLinkObjects(projectCtx, "", []string(linkObjects))
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		opt, err := buildOptions(
			*emit,
			*runtimeMode,
			*islandsDebug,
			*runtimeObject,
			targetLinkObjects,
			*jobs,
		)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 2
		}
		applyBuildReportOptions(
			&opt,
			*explain,
			*emitPLIR,
			*emitProof,
			*emitAllocReport,
			*emitBoundsReport,
			*emitMemoryReport,
			*emitRAMContractReport,
		)
		if !applyRAMContractOptions(
			&opt,
			*failIfHeap,
			*failIfCopy,
			*failIfUnbounded,
			*memoryBudget,
			*ramContractFile,
			*diagnostics,
			stderr,
		) {
			return 2
		}
		opt.ProjectRoot = worldOpt.Root
		opt.SourceRoots = worldOpt.SourceRoots
		opt.DependencyRoots = worldOpt.DependencyRoots
		opt.InterfaceOnly = *interfaceOnly
		for _, rawTarget := range targets {
			tgt, ok := parseBuildTargetOrReport(rawTarget, *diagnostics, stderr)
			if !ok {
				return 2
			}
			if err := validateDiscoveredProjectLock(projectCtx, tgt.Triple); err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			targetLinkObjects, err := projectLinkObjects(
				projectCtx,
				tgt.Triple,
				[]string(linkObjects),
			)
			if err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			opt.LinkObjectPaths = targetLinkObjects
			if !applyRuntimeHeapTelemetryOptions(
				&opt,
				tgt.Triple,
				*emitRuntimeHeapTelemetry,
				*runtimeHeapTelemetryDir,
				*diagnostics,
				stderr,
			) {
				return 2
			}
			output := allTargetsOutput(*out, tgt, *emit)
			if _, err := compiler.BuildFileWithStatsOpt(input, output, tgt.Triple, opt); err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			if *interfaceOnly {
				fmt.Fprintf(stdout, "Interface-only build checked: %s (%s)\n", input, tgt.Triple)
			} else {
				fmt.Fprintf(stdout, "Built: %s\n", output)
			}
		}
		return 0
	}

	rawTarget := *target
	if rawTarget == "" {
		rawTarget = projectDefaultTarget(projectCtx)
	}
	tgt, ok := parseBuildTargetOrReport(rawTarget, *diagnostics, stderr)
	if !ok {
		return 2
	}
	output := *out
	if output == "" {
		output = defaultOutput(tgt, *emit)
	}

	if err := validateDiscoveredProjectLock(projectCtx, tgt.Triple); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}

	targetLinkObjects, err := projectLinkObjects(projectCtx, tgt.Triple, []string(linkObjects))
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	opt, err := buildOptions(
		*emit,
		*runtimeMode,
		*islandsDebug,
		*runtimeObject,
		targetLinkObjects,
		*jobs,
	)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 2
	}
	applyBuildReportOptions(
		&opt,
		*explain,
		*emitPLIR,
		*emitProof,
		*emitAllocReport,
		*emitBoundsReport,
		*emitMemoryReport,
		*emitRAMContractReport,
	)
	if !applyRAMContractOptions(
		&opt,
		*failIfHeap,
		*failIfCopy,
		*failIfUnbounded,
		*memoryBudget,
		*ramContractFile,
		*diagnostics,
		stderr,
	) {
		return 2
	}
	if !applyRuntimeHeapTelemetryOptions(
		&opt,
		tgt.Triple,
		*emitRuntimeHeapTelemetry,
		*runtimeHeapTelemetryDir,
		*diagnostics,
		stderr,
	) {
		return 2
	}
	opt.ProjectRoot = worldOpt.Root
	opt.SourceRoots = worldOpt.SourceRoots
	opt.DependencyRoots = worldOpt.DependencyRoots
	opt.InterfaceOnly = *interfaceOnly
	if _, err := compiler.BuildFileWithStatsOpt(input, output, tgt.Triple, opt); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if *interfaceOnly {
		fmt.Fprintf(stdout, "Interface-only build checked: %s\n", input)
	} else {
		fmt.Fprintf(stdout, "Built: %s\n", output)
	}
	return 0
}

func applyBuildReportOptions(
	opt *compiler.BuildOptions,
	explain bool,
	emitPLIR bool,
	emitProof bool,
	emitAllocReport bool,
	emitBoundsReport bool,
	emitMemoryReport bool,
	emitRAMContractReport bool,
) {
	opt.Explain = explain
	opt.EmitPLIR = emitPLIR
	opt.EmitProof = emitProof
	opt.EmitAllocReport = emitAllocReport
	opt.EmitBoundsReport = emitBoundsReport
	opt.EmitMemoryReport = emitMemoryReport
	opt.EmitRAMContractReport = emitRAMContractReport
}

func applyRAMContractOptions(
	opt *compiler.BuildOptions,
	failIfHeap bool,
	failIfCopy bool,
	failIfUnbounded bool,
	memoryBudget string,
	ramContractFile string,
	diagnostics string,
	stderr io.Writer,
) bool {
	opt.FailIfHeap = failIfHeap
	opt.FailIfCopy = failIfCopy
	opt.FailIfUnbounded = failIfUnbounded
	opt.RAMContractFile = ramContractFile
	if strings.TrimSpace(memoryBudget) == "" {
		return true
	}
	budget, err := parseMemoryBudgetBytes(memoryBudget)
	if err != nil {
		writeValidationDiagnostic(stderr, diagnostics, err.Error())
		return false
	}
	opt.MemoryBudgetBytes = budget
	return true
}

func applyRuntimeHeapTelemetryOptions(
	opt *compiler.BuildOptions,
	target string,
	enabled bool,
	telemetryDir string,
	diagnostics string,
	stderr io.Writer,
) bool {
	if strings.TrimSpace(telemetryDir) != "" && !enabled {
		writeValidationDiagnostic(
			stderr,
			diagnostics,
			"build --runtime-heap-telemetry-dir requires --emit-runtime-heap-telemetry",
		)
		return false
	}
	if !enabled {
		opt.EmitRuntimeHeapTelemetry = false
		opt.RuntimeHeapTelemetryDir = ""
		return true
	}
	if target != "linux-x64" {
		writeValidationDiagnostic(
			stderr,
			diagnostics,
			"build --emit-runtime-heap-telemetry currently supports linux-x64 only",
		)
		return false
	}
	if strings.TrimSpace(telemetryDir) == "" {
		writeValidationDiagnostic(
			stderr,
			diagnostics,
			"build --emit-runtime-heap-telemetry requires --runtime-heap-telemetry-dir",
		)
		return false
	}
	opt.EmitRuntimeHeapTelemetry = true
	opt.RuntimeHeapTelemetryDir = telemetryDir
	return true
}

func parseMemoryBudgetBytes(raw string) (int64, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return 0, nil
	}
	multiplier := int64(1)
	for _, suffix := range []struct {
		suffix string
		mul    int64
	}{
		{suffix: "gb", mul: 1024 * 1024 * 1024},
		{suffix: "g", mul: 1024 * 1024 * 1024},
		{suffix: "mb", mul: 1024 * 1024},
		{suffix: "m", mul: 1024 * 1024},
		{suffix: "kb", mul: 1024},
		{suffix: "k", mul: 1024},
		{suffix: "b", mul: 1},
	} {
		if strings.HasSuffix(value, suffix.suffix) {
			multiplier = suffix.mul
			value = strings.TrimSpace(strings.TrimSuffix(value, suffix.suffix))
			break
		}
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf(
			"build --memory-budget must be a non-negative byte count with optional kb/mb/gb suffix",
		)
	}
	return n * multiplier, nil
}

func runRun(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String(
		"target",
		"",
		"target triple ("+supportedTargetsHelp+"); defaults to Capsule.t4 first target, then host",
	)
	out := fs.String("o", "", "output path")
	surfaceHost := fs.String(
		"surface-host",
		"",
		"native Surface host backend; current supported value is wayland",
	)
	surfaceHostReport := fs.String(
		"surface-host-report",
		"",
		"write native Surface host evidence report to path",
	)
	islandsDebug := fs.Bool("islands-debug", false, "enable islands debug runtime checks")
	runtimeMode := fs.String("runtime", "auto", "actors runtime: auto, selfhost, or builtin")
	runtimeObject := fs.String("runtime-object", "", "actors runtime object override")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	var linkObjects multiFlag
	fs.Var(&linkObjects, "link-object", "extra TOBJ object to link")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	input := ""
	if fs.NArg() > 0 {
		input = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "run accepts at most one input path")
		return 2
	}
	input, worldOpt, projectCtx, err := resolveCLIInput(input)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	rawTarget := *target
	if rawTarget == "" {
		rawTarget = projectDefaultTarget(projectCtx)
	}
	tgt, ok := parseBuildTargetOrReport(rawTarget, *diagnostics, stderr)
	if !ok {
		return 2
	}
	isWASI := tgt.Triple == "wasm32-wasi"
	isWeb := tgt.Triple == "wasm32-web"
	if strings.TrimSpace(*surfaceHost) != "" {
		if *surfaceHost != "wayland" {
			writeValidationDiagnostic(
				stderr,
				*diagnostics,
				fmt.Sprintf(
					"unsupported --surface-host %q; current supported value is wayland",
					*surfaceHost,
				),
			)
			return 2
		}
		if tgt.Triple != "linux-x64" {
			writeValidationDiagnostic(
				stderr,
				*diagnostics,
				"surface-host wayland requires linux-x64 target",
			)
			return 2
		}
	}
	if strings.TrimSpace(*surfaceHostReport) != "" && strings.TrimSpace(*surfaceHost) == "" {
		writeValidationDiagnostic(
			stderr,
			*diagnostics,
			"--surface-host-report requires --surface-host",
		)
		return 2
	}
	if ctarget.IsBuildOnlyTarget(tgt.Triple) && !isWASI && !canRunBuildOnlyNativeTargetOnHost(tgt) {
		writeTargetRuntimeDiagnostic(
			stderr,
			*diagnostics,
			fmt.Sprintf(
				"cannot run target %s: %s",
				tgt.Triple,
				buildOnlyNativeRunUnsupportedReason(tgt),
			),
		)
		return 2
	}
	if !isWASI && !isWeb {
		if !canRunNativeExecutableTargetOnHost(tgt) {
			writeTargetRuntimeDiagnostic(
				stderr,
				*diagnostics,
				fmt.Sprintf(
					"cannot run target %s on host %s/%s",
					tgt.Triple,
					runtime.GOOS,
					runtime.GOARCH,
				),
			)
			return 2
		}
	}
	output := *out
	tmpDir := ""
	if output == "" {
		var err error
		tmpDir, err = os.MkdirTemp("", "tetra-run-*")
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		defer os.RemoveAll(tmpDir)
		output = filepath.Join(tmpDir, defaultOutput(tgt, "exe"))
	}
	var surfaceRunOpt surfaceHostRunOptions
	surfaceHostTmpDir := ""
	if strings.TrimSpace(*surfaceHost) != "" {
		surfaceRunOpt = newSurfaceHostRunOptions(*surfaceHost)
		socketDir := tmpDir
		if socketDir == "" {
			var err error
			surfaceHostTmpDir, err = os.MkdirTemp("", "tetra-surface-host-run-*")
			if err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			defer os.RemoveAll(surfaceHostTmpDir)
			socketDir = surfaceHostTmpDir
		}
		surfaceRunOpt.SocketPath = filepath.Join(socketDir, "surface-host.sock")
		surfaceRunOpt.ReportPath = filepath.Join(socketDir, "surface-host-report.json")
		if strings.TrimSpace(*surfaceHostReport) != "" {
			surfaceRunOpt.ReportPath = *surfaceHostReport
		}
	}
	if err := validateDiscoveredProjectLock(projectCtx, tgt.Triple); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	targetLinkObjects, err := projectLinkObjects(projectCtx, tgt.Triple, []string(linkObjects))
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	opt, err := buildOptions(
		"exe",
		*runtimeMode,
		*islandsDebug,
		*runtimeObject,
		targetLinkObjects,
		*jobs,
	)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 2
	}
	opt.ProjectRoot = worldOpt.Root
	opt.SourceRoots = worldOpt.SourceRoots
	opt.DependencyRoots = worldOpt.DependencyRoots
	if strings.TrimSpace(*surfaceHost) != "" {
		opt.SurfaceHostRequired = true
		opt.SurfaceHostBackend = surfaceRunOpt.Backend
		opt.SurfaceHostProtocol = surfaceRunOpt.Protocol
		opt.SurfaceHostSocketPath = surfaceRunOpt.SocketPath
	}
	if _, err := compiler.BuildFileWithStatsOpt(input, output, tgt.Triple, opt); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if isWASI {
		exit, err := execWASMProgram(output, stdout, stderr)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		return exit
	}
	if isWeb {
		exit, err := execWebProgram(output, stdout, stderr)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		return exit
	}
	if strings.TrimSpace(*surfaceHost) != "" {
		return execNativeSurfaceProgram(output, surfaceRunOpt, stdout, stderr)
	}
	return execNativeProgram(output, stdout, stderr)
}

func runClean(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "remove cache entries for one target")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "clean does not accept positional arguments")
		return 2
	}
	if *target != "" {
		tgt, ok := parseBuildTargetOrReport(*target, "text", stderr)
		if !ok {
			return 2
		}
		for _, path := range []string{
			filepath.Join(".tetra_cache", tgt.Triple),
			filepath.Join("tetra_cache", tgt.Triple),
		} {
			if err := os.RemoveAll(path); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		}
		fmt.Fprintf(stdout, "Cleaned Tetra cache for %s\n", tgt.Triple)
		return 0
	}
	for _, path := range []string{".tetra_cache", "tetra_cache"} {
		if err := os.RemoveAll(path); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintln(stdout, "Cleaned Tetra cache")
	return 0
}

func buildOptions(
	emit string,
	runtimeMode string,
	islandsDebug bool,
	runtimeObject string,
	linkObjects []string,
	jobs int,
) (compiler.BuildOptions, error) {
	opt := compiler.BuildOptions{
		Jobs:              jobs,
		IslandsDebug:      islandsDebug,
		RuntimeObjectPath: runtimeObject,
		LinkObjectPaths:   linkObjects,
	}
	switch emit {
	case "", "exe":
		opt.Emit = compiler.EmitExe
	case "object":
		opt.Emit = compiler.EmitObject
	case "library":
		opt.Emit = compiler.EmitLibrary
	default:
		return opt, fmt.Errorf("unsupported --emit %q", emit)
	}
	switch runtimeMode {
	case "", "auto":
		opt.Runtime = compiler.RuntimeAuto
	case "selfhost":
		opt.Runtime = compiler.RuntimeSelfHost
	case "builtin":
		opt.Runtime = compiler.RuntimeBuiltin
	default:
		return opt, fmt.Errorf("unsupported --runtime %q", runtimeMode)
	}
	return opt, nil
}

func defaultTarget() string {
	if target, ok := hostTarget(); ok {
		return target
	}
	return "linux-x64"
}

func hostTarget() (string, bool) {
	tgt, ok := ctarget.Host()
	if !ok {
		return "", false
	}
	return tgt.Triple, true
}

func canRunBuildOnlyNativeTargetOnHost(tgt ctarget.Target) bool {
	switch tgt.Triple {
	case "linux-x86":
		return linuxX86HostSupport()
	case "linux-x32":
		return linuxX32HostSupport()
	default:
		return false
	}
}

func canRunNativeExecutableTargetOnHost(tgt ctarget.Target) bool {
	if canRunBuildOnlyNativeTargetOnHost(tgt) {
		return true
	}
	host, ok := hostTarget()
	return ok && host == tgt.Triple
}

func canRunLinuxX86OnHost() bool {
	if runtime.GOOS != "linux" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "386") {
		return false
	}
	linuxX86ProbeOnce.Do(func() {
		linuxX86ProbeResult = probeLinuxX86Execution()
	})
	return linuxX86ProbeResult
}

func canRunLinuxX32OnHost() bool {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return false
	}
	linuxX32ProbeOnce.Do(func() {
		linuxX32ProbeResult = probeLinuxX32Execution()
	})
	return linuxX32ProbeResult
}

func probeLinuxX86Execution() bool {
	dir, err := os.MkdirTemp("", "tetra-x86-probe-*")
	if err != nil {
		return false
	}
	defer os.RemoveAll(dir)
	srcPath := filepath.Join(dir, "probe.tetra")
	outPath := filepath.Join(dir, "probe")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		return false
	}
	if _, err := compiler.BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"x86",
		compiler.BuildOptions{Jobs: 1},
	); err != nil {
		return false
	}
	return execProgram(outPath, io.Discard, io.Discard) == 0
}

func probeLinuxX32Execution() bool {
	dir, err := os.MkdirTemp("", "tetra-x32-probe-*")
	if err != nil {
		return false
	}
	defer os.RemoveAll(dir)
	srcPath := filepath.Join(dir, "probe.tetra")
	outPath := filepath.Join(dir, "probe")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		return false
	}
	if _, err := compiler.BuildFileWithStatsOpt(
		srcPath,
		outPath,
		"x32",
		compiler.BuildOptions{Jobs: 1},
	); err != nil {
		return false
	}
	return execProgram(outPath, io.Discard, io.Discard) == 0
}

func buildOnlyNativeRunUnsupportedReason(tgt ctarget.Target) string {
	probe := strings.TrimSpace(tgt.RunnerProbeCommand)
	if probe == "" {
		probe = "tetra test --diagnostics=json --target " + strings.TrimPrefix(
			tgt.Triple,
			"linux-",
		) + " --format=json <runner-smoke.tetra>"
	}
	host := runtime.GOOS + "/" + runtime.GOARCH
	switch tgt.Triple {
	case "linux-x86":
		return fmt.Sprintf(
			"host %s does not support Linux i386 execution; no host fallback is allowed; probe command: %s",
			host,
			probe,
		)
	case "linux-x32":
		return fmt.Sprintf(
			("host %s does not support Linux x32 ABI execution; no host " +
				"fallback is allowed; probe command: %s"),
			host,
			probe,
		)
	default:
		if tgt.UnsupportedReason != "" {
			return tgt.UnsupportedReason
		}
		return ("build-only target emits artifacts only; unsupported runtime " +
			"execution because the CLI does not provide a production runtime runner")
	}
}

func defaultOutput(tgt ctarget.Target, emit string) string {
	switch emit {
	case "object", "library":
		return "app.tobj"
	default:
		return "app" + tgt.ExeExt
	}
}

func projectDefaultTarget(ctx *cliProjectContext) string {
	targets := projectBuildTargets(ctx)
	if len(targets) > 0 {
		return targets[0]
	}
	return defaultTarget()
}

func projectBuildTargets(ctx *cliProjectContext) []string {
	if ctx == nil || !ctx.Found || len(ctx.Manifest.Targets) == 0 {
		return nil
	}
	return append([]string(nil), ctx.Manifest.Targets...)
}

func allTargetsOutput(out string, tgt ctarget.Target, emit string) string {
	name := "app-" + tgt.Triple
	if emit == "object" || emit == "library" {
		name += ".tobj"
	} else {
		name += tgt.ExeExt
	}
	if out == "" {
		return name
	}
	if strings.HasSuffix(out, string(os.PathSeparator)) {
		return filepath.Join(out, name)
	}
	if info, err := os.Stat(out); err == nil && info.IsDir() {
		return filepath.Join(out, name)
	}
	return filepath.Join(out, name)
}

func execProgram(path string, stdout io.Writer, stderr io.Writer) int {
	cmd := exec.Command(path)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

const surfaceHostProtocol = "tetra.surface.host-ipc.v1"

type surfaceHostRunOptions struct {
	Backend     string
	Protocol    string
	SocketPath  string
	ReportPath  string
	HostBinary  string
	RequiredEnv map[string]string
}

func newSurfaceHostRunOptions(backend string) surfaceHostRunOptions {
	return surfaceHostRunOptions{
		Backend:  backend,
		Protocol: surfaceHostProtocol,
		RequiredEnv: map[string]string{
			"TETRA_SURFACE_HOST":          backend,
			"TETRA_SURFACE_HOST_REQUIRED": "1",
			"TETRA_SURFACE_HOST_PROTOCOL": surfaceHostProtocol,
		},
	}
}

func execSurfaceHostedProgram(
	path string,
	opt surfaceHostRunOptions,
	stdout io.Writer,
	stderr io.Writer,
) int {
	hostBinary := strings.TrimSpace(opt.HostBinary)
	if hostBinary == "" {
		var err error
		hostBinary, err = resolveSurfaceHostBinary()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	tmpDir, err := os.MkdirTemp("", "tetra-surface-host-*")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer os.RemoveAll(tmpDir)
	socketPath := strings.TrimSpace(opt.SocketPath)
	if socketPath == "" {
		socketPath = filepath.Join(tmpDir, "host.sock")
	}
	reportPath := strings.TrimSpace(opt.ReportPath)
	if reportPath == "" {
		reportPath = filepath.Join(tmpDir, "host-report.json")
	}
	hostCmd := exec.Command(
		hostBinary,
		"--backend",
		opt.Backend,
		"--socket",
		socketPath,
		"--report",
		reportPath,
	)
	hostCmd.Stdout = stderr
	hostCmd.Stderr = stderr
	if err := hostCmd.Start(); err != nil {
		fmt.Fprintf(stderr, "start Surface host: %v\n", err)
		return 1
	}
	hostDone := make(chan error, 1)
	go func() { hostDone <- hostCmd.Wait() }()
	if err := waitForSurfaceHostSocket(socketPath, hostDone, 2*time.Second); err != nil {
		stopSurfaceHostProcess(hostCmd, hostDone)
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer stopSurfaceHostProcess(hostCmd, hostDone)

	cmd := exec.Command(path)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = surfaceHostEnv(os.Environ(), opt, socketPath, reportPath)
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func resolveSurfaceHostBinary() (string, error) {
	if env := strings.TrimSpace(os.Getenv("TETRA_SURFACE_HOST_BIN")); env != "" {
		return env, nil
	}
	if path, err := commandLookPath("tetra-surface-host"); err == nil {
		return path, nil
	}
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "tetra-surface-host")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf(
		"tetra-surface-host not found; install it next to tetra or set TETRA_SURFACE_HOST_BIN",
	)
}

func surfaceHostEnv(
	base []string,
	opt surfaceHostRunOptions,
	socketPath string,
	reportPath string,
) []string {
	env := append([]string(nil), base...)
	values := map[string]string{}
	for key, value := range opt.RequiredEnv {
		values[key] = value
	}
	values["TETRA_SURFACE_HOST_SOCKET"] = socketPath
	values["TETRA_SURFACE_HOST_REPORT"] = reportPath
	for key, value := range values {
		env = setEnvValue(env, key, value)
	}
	return env
}

func setEnvValue(env []string, key string, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func waitForSurfaceHostSocket(
	socketPath string,
	hostDone <-chan error,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case err := <-hostDone:
			if err != nil {
				return fmt.Errorf("Surface host exited before socket was ready: %w", err)
			}
			return fmt.Errorf("Surface host exited before socket was ready")
		default:
		}
		if info, err := os.Stat(socketPath); err == nil && !info.IsDir() {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for Surface host socket %s", socketPath)
}

func stopSurfaceHostProcess(cmd *exec.Cmd, done <-chan error) {
	if cmd.Process == nil {
		return
	}
	if cmd.ProcessState != nil {
		return
	}
	_ = cmd.Process.Signal(os.Interrupt)
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		_ = cmd.Process.Kill()
		<-done
	}
}

type wasiRunner struct {
	Name   string
	Path   string
	Helper string
}

func discoverWASIRunner(repoRoot string) (wasiRunner, error) {
	if runner, err := commandLookPath("wasmtime"); err == nil {
		return wasiRunner{Name: "wasmtime", Path: runner}, nil
	}
	node, err := commandLookPath("node")
	if err != nil {
		return wasiRunner{}, fmt.Errorf(
			"cannot run target wasm32-wasi: missing WASI runner: need wasmtime or node",
		)
	}
	if repoRoot == "" {
		if root, rootErr := findRepoRoot(); rootErr == nil {
			repoRoot = root
		}
	}
	helper := filepath.Join(repoRoot, "scripts", "tools", "wasi_run_module.mjs")
	if repoRoot == "" || !fileExists(helper) {
		return wasiRunner{}, fmt.Errorf(
			"cannot run target wasm32-wasi: missing WASI node helper scripts/tools/wasi_run_module.mjs",
		)
	}
	return wasiRunner{Name: "node-wasi", Path: node, Helper: helper}, nil
}

func discoverWebRunner() (string, error) {
	var probeFailure string
	for _, candidate := range []string{"chromium", "chromium-browser", "google-chrome", "chrome"} {
		if runner, err := commandLookPath(candidate); err == nil {
			if err := webRunnerProbe(runner); err != nil {
				probeFailure = fmt.Sprintf("%s failed headless probe: %v", runner, err)
				continue
			}
			return runner, nil
		}
	}
	if probeFailure != "" {
		return "", fmt.Errorf(
			"cannot run target wasm32-web: browser runner unavailable: %s",
			probeFailure,
		)
	}
	return "", fmt.Errorf(
		("cannot run target wasm32-web: browser runner unavailable; " +
			"searched: chromium, chromium-browser, google-chrome, chrome"),
	)
}

type webRuntimeRunner struct {
	Name   string
	Path   string
	Helper string
}

func discoverWebRuntimeRunner(repoRoot string) (webRuntimeRunner, error) {
	node, err := commandLookPath("node")
	if err != nil {
		return webRuntimeRunner{}, fmt.Errorf(
			"cannot run target wasm32-web: missing web runtime runner: need node",
		)
	}
	if repoRoot == "" {
		if root, rootErr := findRepoRoot(); rootErr == nil {
			repoRoot = root
		}
	}
	helper := filepath.Join(repoRoot, "scripts", "tools", "web_run_module.mjs")
	if repoRoot == "" || !fileExists(helper) {
		return webRuntimeRunner{}, fmt.Errorf(
			"cannot run target wasm32-web: missing web node helper scripts/tools/web_run_module.mjs",
		)
	}
	return webRuntimeRunner{Name: "node-web", Path: node, Helper: helper}, nil
}

func probeWebRunner(runner string) error {
	cmd := exec.Command(
		runner,
		"--headless",
		"--no-sandbox",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--disable-crash-reporter",
		"--disable-breakpad",
		"--dump-dom",
		"about:blank",
	)
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

func execWASMProgram(path string, stdout io.Writer, stderr io.Writer) (int, error) {
	runner, err := discoverWASIRunner("")
	if err != nil {
		return 0, err
	}
	return execWASMProgramWithRunner(path, runner, stdout, stderr)
}

func execWASMProgramWithRunner(
	path string,
	runner wasiRunner,
	stdout io.Writer,
	stderr io.Writer,
) (int, error) {
	var cmd *exec.Cmd
	switch runner.Name {
	case "wasmtime":
		cmd = exec.Command(runner.Path, "run", path)
	case "node-wasi":
		cmd = exec.Command(runner.Path, runner.Helper, path)
	default:
		return 0, fmt.Errorf("unsupported WASI runner %q", runner.Name)
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 0, err
	}
	if cmd.ProcessState == nil {
		return 0, nil
	}
	return cmd.ProcessState.ExitCode(), nil
}

func execWebProgram(path string, stdout io.Writer, stderr io.Writer) (int, error) {
	runner, err := discoverWebRunner()
	if err != nil {
		return 0, err
	}
	return execWebProgramWithBrowserRunner(path, runner, stdout, stderr)
}

func execWebProgramWithRunner(
	path string,
	runner webRuntimeRunner,
	stdout io.Writer,
	stderr io.Writer,
) (int, error) {
	args := []string{runner.Helper, path}
	if runner.Name == "node-web" {
		args = append([]string{"--no-warnings"}, args...)
	}
	cmd := exec.Command(runner.Path, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), nil
		}
		return 0, err
	}
	if cmd.ProcessState == nil {
		return 0, nil
	}
	return cmd.ProcessState.ExitCode(), nil
}

func execWebProgramWithBrowserRunner(
	path string,
	runner string,
	stdout io.Writer,
	stderr io.Writer,
) (int, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return 0, err
	}
	dir := filepath.Dir(absPath)
	wasmFile := filepath.Base(absPath)
	loaderFile := strings.TrimSuffix(wasmFile, filepath.Ext(wasmFile)) + ".mjs"
	if !fileExists(filepath.Join(dir, loaderFile)) {
		return 0, fmt.Errorf(
			"cannot run target wasm32-web: missing web loader %s",
			filepath.Join(dir, loaderFile),
		)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		return 0, err
	}
	runnerHTML := filepath.Join(dir, ".tetra-web-runner.html")
	html := fmt.Sprintf(`<!doctype html>
<html>
  <body>
    <pre id="result">pending</pre>
    <script type="module">
      import { runTetra } from './%s';
      const el = document.getElementById('result');
      try {
        const code = await runTetra(new URL('./%s', import.meta.url));
        el.textContent = 'exit:' + (code | 0);
      } catch (err) {
        el.textContent = 'error:' + String(err && err.message ? err.message : err);
      }
    </script>
  </body>
</html>
`, loaderFile, wasmFile)
	if err := os.WriteFile(runnerHTML, []byte(html), 0o644); err != nil {
		return 0, err
	}
	defer os.Remove(runnerHTML)

	server := exec.Command(
		"python3",
		"-m",
		"http.server",
		strconv.Itoa(port),
		"--bind",
		"127.0.0.1",
		"--directory",
		dir,
	)
	server.Stdout = io.Discard
	server.Stderr = io.Discard
	if err := server.Start(); err != nil {
		return 0, fmt.Errorf("cannot run target wasm32-web: start local HTTP server: %w", err)
	}
	defer func() {
		if server.Process != nil {
			_ = server.Process.Kill()
			_, _ = server.Process.Wait()
		}
	}()
	time.Sleep(300 * time.Millisecond)

	url := fmt.Sprintf("http://127.0.0.1:%d/%s", port, filepath.Base(runnerHTML))
	cmd := exec.Command(
		runner,
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--virtual-time-budget=12000",
		"--dump-dom",
		url,
	)
	var dom bytes.Buffer
	cmd.Stdout = &dom
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("cannot run target wasm32-web: browser runner failed: %w", err)
	}
	result := extractWebRunnerResult(dom.String())
	if strings.HasPrefix(result, "exit:") {
		code, err := strconv.Atoi(strings.TrimPrefix(result, "exit:"))
		if err != nil {
			return 0, fmt.Errorf(
				"cannot run target wasm32-web: invalid browser exit result %q",
				result,
			)
		}
		return code & 0xff, nil
	}
	if strings.HasPrefix(result, "error:") {
		return 1, fmt.Errorf(
			"cannot run target wasm32-web: %s",
			strings.TrimPrefix(result, "error:"),
		)
	}
	return 1, fmt.Errorf("cannot run target wasm32-web: browser did not report an exit result")
}

func extractWebRunnerResult(dom string) string {
	re := regexp.MustCompile(`<pre id="result">([^<]*)</pre>`)
	m := re.FindStringSubmatch(dom)
	if len(m) != 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(dir, "go.work")) && fileExists(filepath.Join(dir, "examples")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root")
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func gitHead(root string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func writeDiagnostic(w io.Writer, mode string, err error) {
	if outputformat.Structured(mode) {
		writeDiagnosticObject(w, mode, compiler.DiagnosticFromError(err))
		return
	}
	fmt.Fprintln(w, err)
}

func writeDiagnosticWithHint(w io.Writer, mode string, message string, hint string) {
	if outputformat.Structured(mode) {
		writeDiagnosticObject(w, mode, compiler.Diagnostic{
			Code:     compiler.DiagnosticCodeParse,
			Message:  message,
			Severity: "error",
			Hint:     hint,
		})
		return
	}
	fmt.Fprintln(w, message)
}

func writeTargetRuntimeDiagnostic(w io.Writer, mode string, message string) {
	if outputformat.Structured(mode) {
		writeDiagnosticObject(w, mode, compiler.Diagnostic{
			Code:     compiler.DiagnosticCodeTargetRuntime,
			Message:  message,
			Severity: "error",
		})
		return
	}
	fmt.Fprintln(w, message)
}

func parseBuildTargetOrReport(
	rawTarget string,
	diagnosticsMode string,
	stderr io.Writer,
) (ctarget.Target, bool) {
	tgt, err := ctarget.Parse(rawTarget)
	if err == nil {
		return tgt, true
	}
	if targetErr, ok := err.(ctarget.UnsupportedTargetError); ok {
		msg := fmt.Sprintf(
			"unsupported target: %s; supported targets: %s; build-only targets: %s",
			targetErr.Triple,
			strings.Join(ctarget.SupportedTriples(), ", "),
			strings.Join(ctarget.BuildOnlyTriples(), ", "),
		)
		writeDiagnosticWithHint(
			stderr,
			diagnosticsMode,
			msg,
			"run `tetra targets` to list valid targets",
		)
		return ctarget.Target{}, false
	}
	writeDiagnostic(stderr, diagnosticsMode, err)
	return ctarget.Target{}, false
}

func writeValidationDiagnostic(w io.Writer, mode string, message string) {
	writeDiagnostic(w, mode, fmt.Errorf("%s", message))
}

func validateDiagnosticsMode(w io.Writer, mode string) bool {
	if mode == "text" || outputformat.Structured(mode) {
		return true
	}
	fmt.Fprintln(w, "unsupported --diagnostics format")
	return false
}

func writeDiagnosticObject(w io.Writer, mode string, diag compiler.Diagnostic) {
	if err := outputformat.WriteStructured(w, mode, diag); err != nil {
		fmt.Fprintln(w, err)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: tetra <command> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "core workflow:")
	fmt.Fprintln(w, "  check    validate source without emitting artifacts")
	fmt.Fprintln(w, "  build    validate source and emit or link artifacts")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "tooling commands:")
	fmt.Fprintln(w, "  version targets features formats doctor actor-net project workspace new")
	fmt.Fprintln(w, "  run smoke surface fmt test doc interface clean eco lsp")
}

// ---- project.go ----

type cliProjectContext struct {
	Found           bool
	Root            string
	CapsulePath     string
	LockPath        string
	Manifest        capsuleManifest
	Manifests       []capsuleManifest
	EntryPath       string
	SourceRoots     []string
	DependencyRoots []compiler.ModuleRoot
}

var defaultProjectSourceRoots = []string{"src", "ui", "tests", "drivers", "kernel", "game", "."}

func resolveCLIInput(input string) (string, compiler.WorldOptions, *cliProjectContext, error) {
	startDir, err := cliProjectStartDir(input)
	if err != nil {
		return "", compiler.WorldOptions{}, nil, err
	}
	ctx, err := discoverCLIProject(startDir)
	if err != nil {
		return "", compiler.WorldOptions{}, nil, err
	}
	if ctx == nil || !ctx.Found {
		if input == "" {
			input = defaultInputPath()
		}
		return input, compiler.WorldOptions{}, nil, nil
	}

	projectReference := isProjectReference(input, ctx)
	if input != "" && !projectReference {
		entry, err := filepath.Abs(input)
		if err != nil {
			return "", compiler.WorldOptions{}, nil, fmt.Errorf("resolve input path: %w", err)
		}
		return entry, compiler.WorldOptions{}, nil, nil
	}

	entry := ctx.EntryPath
	opt := compiler.WorldOptions{
		Root:            ctx.Root,
		SourceRoots:     append([]string(nil), ctx.SourceRoots...),
		DependencyRoots: append([]compiler.ModuleRoot(nil), ctx.DependencyRoots...),
	}
	return entry, opt, ctx, nil
}

func isProjectReference(input string, ctx *cliProjectContext) bool {
	if ctx == nil || !ctx.Found || strings.TrimSpace(input) == "" {
		return false
	}
	abs, err := filepath.Abs(input)
	if err != nil {
		return false
	}
	cleanAbs := filepath.Clean(abs)
	if info, err := os.Stat(cleanAbs); err == nil && info.IsDir() {
		return filepath.Clean(ctx.Root) == cleanAbs
	}
	return filepath.Clean(ctx.CapsulePath) == cleanAbs
}

func discoverCLIProject(startDir string) (*cliProjectContext, error) {
	capsulePath, root, ok, err := findProjectCapsule(startDir)
	if err != nil || !ok {
		return &cliProjectContext{}, err
	}
	manifest, err := parseCapsule(capsulePath)
	if err != nil {
		return nil, err
	}
	sourceRoots := projectSourceRoots(manifest)
	entryPath, err := resolveProjectEntry(root, manifest)
	if err != nil {
		return nil, err
	}
	dependencyRoots, dependencyManifests, err := projectDependencyGraph(
		root,
		manifest,
		map[string]int{root: projectDependencyVisiting},
		[]string{root},
	)
	if err != nil {
		return nil, err
	}
	artifactRoots, err := projectArtifactInterfaceRoots(root, manifest.Artifacts)
	if err != nil {
		return nil, err
	}
	if capsuleHasInterfaceArtifacts(manifest.Artifacts) {
		dependencyRoots = nil
	}
	dependencyRoots = append(dependencyRoots, artifactRoots...)
	dependencyRoots = appendProjectStdlibDependencyRoot(root, dependencyRoots)
	manifests := append([]capsuleManifest{manifest}, dependencyManifests...)
	return &cliProjectContext{
		Found:           true,
		Root:            root,
		CapsulePath:     capsulePath,
		LockPath:        findProjectLock(root),
		Manifest:        manifest,
		Manifests:       manifests,
		EntryPath:       entryPath,
		SourceRoots:     sourceRoots,
		DependencyRoots: dependencyRoots,
	}, nil
}

func appendProjectStdlibDependencyRoot(
	projectRoot string,
	roots []compiler.ModuleRoot,
) []compiler.ModuleRoot {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return roots
	}
	repoRoot = filepath.Clean(repoRoot)
	projectRoot = filepath.Clean(projectRoot)
	if repoRoot == "" || repoRoot == projectRoot {
		return roots
	}
	if !fileExists(filepath.Join(repoRoot, "lib", "core")) {
		return roots
	}
	return append(roots, compiler.ModuleRoot{Root: repoRoot})
}

func findProjectCapsule(startDir string) (string, string, bool, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", "", false, err
	}
	for {
		for _, name := range []string{compiler.CapsuleFileName, compiler.LegacyCapsuleFileName} {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path, dir, true, nil
			} else if !os.IsNotExist(err) {
				return "", "", false, err
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", false, nil
		}
		dir = parent
	}
}

func findProjectLock(root string) string {
	path := filepath.Join(root, compiler.SemanticLockFileName)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func cliProjectStartDir(input string) (string, error) {
	if input == "" {
		return os.Getwd()
	}
	abs, err := filepath.Abs(input)
	if err != nil {
		return "", fmt.Errorf("resolve input path: %w", err)
	}
	info, err := os.Stat(abs)
	if err == nil && info.IsDir() {
		return abs, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	return filepath.Dir(abs), nil
}

func capsuleHasInterfaceArtifacts(artifacts []capsuleArtifact) bool {
	for _, artifact := range artifacts {
		if artifact.Kind == "interface" {
			return true
		}
	}
	return false
}

func projectArtifactInterfaceRoots(
	root string,
	artifacts []capsuleArtifact,
) ([]compiler.ModuleRoot, error) {
	var roots []compiler.ModuleRoot
	seen := map[string]struct{}{}
	for _, artifact := range artifacts {
		if artifact.Kind != "interface" {
			continue
		}
		relRoot, err := interfaceArtifactSourceRoot(root, artifact.Path)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[relRoot]; ok {
			continue
		}
		seen[relRoot] = struct{}{}
		roots = append(roots, compiler.ModuleRoot{
			Root:        root,
			SourceRoots: []string{relRoot},
		})
	}
	sort.Slice(roots, func(i, j int) bool {
		if roots[i].Root == roots[j].Root {
			return strings.Join(roots[i].SourceRoots, ",") < strings.Join(roots[j].SourceRoots, ",")
		}
		return roots[i].Root < roots[j].Root
	})
	return roots, nil
}

func projectArtifactObjectPaths(
	root string,
	artifacts []capsuleArtifact,
	target string,
) ([]string, error) {
	var paths []string
	for _, artifact := range artifacts {
		if artifact.Kind != "object" {
			continue
		}
		if artifact.Target != "" && target != "" && artifact.Target != target {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(artifact.Path))
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("artifact object %s: %w", artifact.Path, err)
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func interfaceArtifactSourceRoot(root string, relPath string) (string, error) {
	path := filepath.Join(root, filepath.FromSlash(relPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("artifact interface %s: %w", relPath, err)
	}
	moduleName := interfaceArtifactModuleName(raw)
	if moduleName == "" {
		return "", fmt.Errorf("artifact interface %s: missing module declaration", relPath)
	}
	moduleRel := filepath.ToSlash(
		moduleRelPathWithExtension(moduleName, compiler.T4InterfaceExtension),
	)
	cleanRel := filepath.ToSlash(filepath.Clean(relPath))
	if cleanRel != moduleRel && !strings.HasSuffix(cleanRel, "/"+moduleRel) {
		return "", fmt.Errorf(
			"artifact interface %s: module '%s' must be in %s",
			relPath,
			moduleName,
			moduleRel,
		)
	}
	rootRel := strings.TrimSuffix(cleanRel, moduleRel)
	rootRel = strings.TrimSuffix(rootRel, "/")
	if rootRel == "" {
		return ".", nil
	}
	return rootRel, nil
}

func interfaceArtifactModuleName(raw []byte) string {
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "module ") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			fields := strings.Fields(name)
			if len(fields) == 0 {
				return ""
			}
			return fields[0]
		}
	}
	return ""
}

func resolveProjectEntry(root string, manifest capsuleManifest) (string, error) {
	if manifest.Entry != "" {
		rel, err := cleanProjectRelPath(manifest.Entry)
		if err != nil {
			return "", fmt.Errorf("%s: invalid entry: %w", manifest.Path, err)
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err != nil {
			return "", err
		}
		return path, nil
	}
	for _, rel := range []string{
		compiler.DefaultSourceFileName,
		filepath.ToSlash(filepath.Join("src", compiler.DefaultSourceFileName)),
		compiler.LegacySourceFileName,
		filepath.ToSlash(filepath.Join("src", compiler.LegacySourceFileName)),
	} {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf(
		"%s: missing project entry (set entry \"src/main.t4\" or add main.t4/src/main.t4)",
		manifest.Path,
	)
}

func projectSourceRoots(manifest capsuleManifest) []string {
	roots := manifest.SourceRoots
	if len(roots) == 0 {
		roots = defaultProjectSourceRoots
	}
	return cleanProjectSourceRoots(roots)
}

func existingProjectSourcePaths(ctx *cliProjectContext) []string {
	if ctx == nil || !ctx.Found {
		return nil
	}
	seen := map[string]struct{}{}
	var paths []string
	for _, root := range ctx.SourceRoots {
		path := ctx.Root
		if root != "" {
			path = filepath.Join(ctx.Root, filepath.FromSlash(root))
		}
		if _, ok := seen[path]; ok {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func validateDiscoveredProjectLock(ctx *cliProjectContext, target string) error {
	if ctx == nil || !ctx.Found || ctx.LockPath == "" {
		return nil
	}
	if err := validateCapsuleGraph(ctx.Manifests, target); err != nil {
		return fmt.Errorf(
			"%s: %w; repair with: %s",
			ctx.LockPath,
			err,
			projectSyncRepairCommand(ctx.Root, target, false),
		)
	}
	issues, err := checkDeclaredCapsuleArtifacts(ctx.CapsulePath, target, ctx.LockPath, false)
	if err != nil {
		return fmt.Errorf("%s: %w", ctx.LockPath, err)
	}
	if len(issues) > 0 {
		issue := issues[0]
		detail := issue.Detail
		if detail != "" {
			detail = ": " + detail
		}
		repair := "; repair with: " + projectSyncRepairCommand(ctx.Root, target, false)
		if issue.Module != "" {
			return fmt.Errorf(
				"%s: %s for %s at %s%s%s",
				ctx.LockPath,
				issue.Kind,
				issue.Module,
				issue.Path,
				detail,
				repair,
			)
		}
		return fmt.Errorf("%s: %s at %s%s%s", ctx.LockPath, issue.Kind, issue.Path, detail, repair)
	}
	raw, err := os.ReadFile(ctx.LockPath)
	if err != nil {
		return fmt.Errorf("%s: %w", ctx.LockPath, err)
	}
	lock, err := decodeEcoLock(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", ctx.LockPath, err)
	}
	current, err := buildEcoLockWithArtifactHashes(ctx.Manifests)
	if err != nil {
		return fmt.Errorf("%s: %w", ctx.LockPath, err)
	}
	if lock.GraphSHA256 != current.GraphSHA256 {
		return fmt.Errorf(
			"%s: project lock is stale: graph_sha256 %s, current %s; repair with: %s",
			ctx.LockPath,
			lock.GraphSHA256,
			current.GraphSHA256,
			projectSyncRepairCommand(ctx.Root, target, false),
		)
	}
	return nil
}

func projectSyncRepairCommand(root string, target string, allTargets bool) string {
	var parts []string
	parts = append(parts, "tetra", "project", "sync")
	if allTargets {
		parts = append(parts, "--all-targets")
	} else if target != "" {
		parts = append(parts, "--target", target)
	}
	if root != "" {
		parts = append(parts, filepath.ToSlash(root))
	}
	return strings.Join(parts, " ")
}

func projectLinkObjects(
	ctx *cliProjectContext,
	target string,
	explicit []string,
) ([]string, error) {
	if ctx == nil || !ctx.Found {
		return append([]string(nil), explicit...), nil
	}
	projectObjects, err := projectArtifactObjectPaths(ctx.Root, ctx.Manifest.Artifacts, target)
	if err != nil {
		return nil, err
	}
	if len(projectObjects) == 0 {
		return append([]string(nil), explicit...), nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(projectObjects)+len(explicit))
	for _, path := range projectObjects {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	for _, path := range explicit {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out, nil
}

const (
	projectDependencyUnvisited = iota
	projectDependencyVisiting
	projectDependencyDone
)

func projectDependencyGraph(
	root string,
	manifest capsuleManifest,
	state map[string]int,
	stack []string,
) ([]compiler.ModuleRoot, []capsuleManifest, error) {
	var out []compiler.ModuleRoot
	var manifests []capsuleManifest
	for _, dep := range manifest.Dependencies {
		if dep.Path == "" {
			continue
		}
		depRoot, err := resolveDependencyProjectRoot(root, dep.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: dependency %s: %w", manifest.Path, dep.ID, err)
		}
		switch state[depRoot] {
		case projectDependencyVisiting:
			return nil, nil, fmt.Errorf(
				"%s: capsule dependency cycle: %s",
				manifest.Path,
				describeProjectDependencyCycle(stack, depRoot),
			)
		case projectDependencyDone:
			continue
		}
		state[depRoot] = projectDependencyVisiting
		capsulePath, err := findCapsulePath(depRoot)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: dependency %s: %w", manifest.Path, dep.ID, err)
		}
		depManifest, err := parseCapsule(capsulePath)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, compiler.ModuleRoot{
			Root:        depRoot,
			SourceRoots: projectSourceRoots(depManifest),
		})
		artifactRoots, err := projectArtifactInterfaceRoots(depRoot, depManifest.Artifacts)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, artifactRoots...)
		manifests = append(manifests, depManifest)
		transitiveRoots, transitiveManifests, err := projectDependencyGraph(
			depRoot,
			depManifest,
			state,
			append(stack, depRoot),
		)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, transitiveRoots...)
		manifests = append(manifests, transitiveManifests...)
		state[depRoot] = projectDependencyDone
	}
	return out, manifests, nil
}

func describeProjectDependencyCycle(stack []string, repeated string) string {
	start := 0
	for i, root := range stack {
		if root == repeated {
			start = i
			break
		}
	}
	cycle := append([]string(nil), stack[start:]...)
	cycle = append(cycle, repeated)
	for i := range cycle {
		cycle[i] = filepath.ToSlash(cycle[i])
	}
	return strings.Join(cycle, " -> ")
}

func resolveDependencyProjectRoot(root string, depPath string) (string, error) {
	if strings.TrimSpace(depPath) == "" {
		return "", fmt.Errorf("path is empty")
	}
	if strings.Contains(depPath, "\\") {
		return "", fmt.Errorf("path must use forward slashes")
	}
	path := filepath.FromSlash(depPath)
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		path = filepath.Dir(path)
	}
	return path, nil
}

func cleanProjectSourceRoots(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, root := range in {
		rel, err := cleanProjectRelPath(root)
		if err != nil {
			continue
		}
		if rel == "." {
			rel = ""
		}
		if _, ok := seen[rel]; ok {
			continue
		}
		seen[rel] = struct{}{}
		out = append(out, rel)
	}
	return out
}

func cleanProjectRelPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("path is empty")
	}
	if strings.Contains(value, "\\") {
		return "", fmt.Errorf("path must use forward slashes")
	}
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("path must be relative")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean == "." {
		return ".", nil
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path must stay inside project root")
	}
	return clean, nil
}

// ---- project_commands.go ----

func runProject(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra project <info|sync|deps> [options]")
		return 0
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, "project requires a subcommand")
		return 2
	}
	switch args[0] {
	case "info":
		return runProjectInfo(args[1:], stdout, stderr)
	case "sync":
		return runProjectSync(args[1:], stdout, stderr)
	case "deps":
		return runProjectDeps(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown project subcommand %q\n", args[0])
		return 2
	}
}

type projectDepsContext struct {
	Root        string
	CapsulePath string
	Manifest    capsuleManifest
}

type projectDepsReport struct {
	Status       string                    `json:"status,omitempty"`
	Root         string                    `json:"root,omitempty"`
	CapsulePath  string                    `json:"capsule_path,omitempty"`
	Dependencies []projectDependencyReport `json:"dependencies"`
}

type projectDependencyReport struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	Path         string `json:"path,omitempty"`
	ResolvedPath string `json:"resolved_path,omitempty"`
	Status       string `json:"status"`
	Detail       string `json:"detail,omitempty"`
}

func runProjectDeps(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra project deps <list|add|remove|check> [options]")
		return 2
	}
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra project deps <list|add|remove|check> [options]")
		return 0
	}
	switch args[0] {
	case "list":
		return runProjectDepsList(args[1:], stdout, stderr)
	case "add":
		return runProjectDepsAdd(args[1:], stdout, stderr)
	case "remove":
		return runProjectDepsRemove(args[1:], stdout, stderr)
	case "check":
		return runProjectDepsCheck(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown project deps command %q\n", args[0])
		return 2
	}
}

func runProjectDepsList(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("project deps list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "project deps list accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	ctx, err := discoverProjectDepsContext(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	report := buildProjectDepsReport(ctx)
	switch *format {
	case "text", "":
		writeProjectDepsText(stdout, report)
		return 0
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runProjectDepsAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("project deps add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	pathFlag := fs.String("path", "", "local dependency project path")
	idFlag := fs.String("id", "", "dependency id; defaults to dependency Capsule.t4 id")
	versionFlag := fs.String(
		"version",
		"",
		"dependency version; defaults to dependency Capsule.t4 version",
	)
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "project deps add accepts at most one project path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	ctx, err := discoverProjectDepsContext(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	_, depManifest, relPath, err := resolveProjectDependencyAddPath(ctx.Root, *pathFlag)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	id := strings.TrimSpace(*idFlag)
	if id == "" {
		id = depManifest.ID
	}
	id, err = normalizeProjectDependencyID(id)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	version := strings.TrimSpace(*versionFlag)
	if version == "" {
		version = depManifest.Version
	}
	if !isCapsuleSemver(version) {
		fmt.Fprintln(stderr, "project deps add --version must use semver x.y.z")
		return 2
	}
	for _, dep := range ctx.Manifest.Dependencies {
		if dep.ID == id && dep.Version == version {
			fmt.Fprintf(stderr, "duplicate dependency %s %s\n", id, version)
			return 1
		}
	}
	dep := capsuleDependency{ID: id, Version: version, Path: relPath}
	if err := appendDependencyToCapsule(ctx.CapsulePath, dep); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Added dependency: %s %s %s\n", dep.ID, dep.Version, dep.Path)
	fmt.Fprintf(stdout, "run: %s\n", projectSyncRepairCommand(ctx.Root, "", false))
	return 0
}

func runProjectDepsRemove(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("project deps remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	idFlag := fs.String("id", "", "dependency id")
	versionFlag := fs.String("version", "", "dependency version")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "project deps remove accepts at most one project path")
		return 2
	}
	id, err := normalizeProjectDependencyID(*idFlag)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	version := strings.TrimSpace(*versionFlag)
	if version != "" && !isCapsuleSemver(version) {
		fmt.Fprintln(stderr, "project deps remove --version must use semver x.y.z")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	ctx, err := discoverProjectDepsContext(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	var matches []capsuleDependency
	for _, dep := range ctx.Manifest.Dependencies {
		if dep.ID == id && (version == "" || dep.Version == version) {
			matches = append(matches, dep)
		}
	}
	if len(matches) == 0 {
		fmt.Fprintf(stderr, "dependency not found: %s\n", id)
		return 1
	}
	if version == "" && len(matches) > 1 {
		fmt.Fprintf(stderr, "dependency %s has multiple versions; remove requires --version\n", id)
		return 2
	}
	removed, err := removeDependencyFromCapsule(ctx.CapsulePath, id, version)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if removed == 0 {
		fmt.Fprintf(stderr, "dependency not found: %s\n", id)
		return 1
	}
	if version == "" {
		version = matches[0].Version
	}
	fmt.Fprintf(stdout, "Removed dependency: %s %s\n", id, version)
	fmt.Fprintf(stdout, "run: %s\n", projectSyncRepairCommand(ctx.Root, "", false))
	return 0
}

func runProjectDepsCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("project deps check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "project deps check accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	ctx, err := discoverProjectDepsContext(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	report := buildProjectDepsReport(ctx)
	status := "pass"
	var issues []projectDependencyReport
	for _, dep := range report.Dependencies {
		if dep.Status != "ok" {
			status = "fail"
			issues = append(issues, dep)
		}
	}
	if status == "pass" {
		_, depManifests, err := projectDependencyGraph(
			ctx.Root,
			ctx.Manifest,
			map[string]int{ctx.Root: projectDependencyVisiting},
			[]string{ctx.Root},
		)
		if err != nil {
			status = "fail"
			issues = append(issues, projectDependencyReport{Status: "fail", Detail: err.Error()})
		} else {
			manifests := append([]capsuleManifest{ctx.Manifest}, depManifests...)
			if err := validateCapsuleGraph(manifests, ""); err != nil {
				status = "fail"
				issues = append(issues, projectDependencyReport{Status: "fail", Detail: err.Error()})
			}
		}
	}
	report.Status = status
	switch *format {
	case "text", "":
		if status == "pass" {
			fmt.Fprintf(stdout, "Dependencies OK: %d\n", len(report.Dependencies))
			return 0
		}
		writeProjectDependencyIssues(stderr, issues)
		return 1
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if status != "pass" {
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

type projectInfoReport struct {
	Found              bool           `json:"found"`
	Root               string         `json:"root,omitempty"`
	CapsulePath        string         `json:"capsule_path,omitempty"`
	LockPath           string         `json:"lock_path,omitempty"`
	EntryPath          string         `json:"entry_path,omitempty"`
	SourceRoots        []string       `json:"source_roots,omitempty"`
	Targets            []string       `json:"targets,omitempty"`
	DependencyRoots    []string       `json:"dependency_roots,omitempty"`
	ArtifactCounts     map[string]int `json:"artifact_counts,omitempty"`
	DependencyCapsules []string       `json:"dependency_capsules,omitempty"`
}

func runProjectInfo(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("project info", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "project info accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	report, err := buildProjectInfoReport(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	switch *format {
	case "text", "":
		if !report.Found {
			fmt.Fprintln(stdout, "Project: not found")
			return 1
		}
		fmt.Fprintf(stdout, "Project root: %s\n", report.Root)
		fmt.Fprintf(stdout, "Capsule: %s\n", report.CapsulePath)
		if report.LockPath != "" {
			fmt.Fprintf(stdout, "Lock: %s\n", report.LockPath)
		}
		fmt.Fprintf(stdout, "Entry: %s\n", report.EntryPath)
		fmt.Fprintf(stdout, "Source roots: %s\n", strings.Join(report.SourceRoots, ", "))
		fmt.Fprintf(stdout, "Targets: %s\n", strings.Join(report.Targets, ", "))
		return 0
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if !report.Found {
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runProjectSync(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("project sync", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetFlag := fs.String("target", "", "native target triple for generated .tobj artifacts")
	checkOnly := fs.Bool(
		"check",
		false,
		"dry-run and report pending project lock/artifact changes without writing files",
	)
	allTargets := fs.Bool(
		"all-targets",
		false,
		"sync artifacts for every native target listed in Capsule.t4",
	)
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *targetFlag != "" && *allTargets {
		fmt.Fprintln(stderr, "project sync accepts either --target or --all-targets, not both")
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "project sync accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	ctx, err := discoverCLIProject(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if ctx == nil || !ctx.Found {
		fmt.Fprintln(stderr, "project capsule not found")
		return 1
	}
	lockPath := filepath.Join(ctx.Root, compiler.SemanticLockFileName)
	if *checkOnly {
		issues, err := checkProjectSync(ctx, *targetFlag, lockPath, *allTargets)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		for i := range issues {
			issues[i].Repair = projectSyncRepairCommand(ctx.Root, *targetFlag, *allTargets)
		}
		if len(issues) > 0 {
			writeArtifactIssues(stdout, issues, true)
			return 1
		}
		fmt.Fprintf(stdout, "Project current: %s\n", ctx.Root)
		return 0
	}
	useArtifactBuilder, err := projectSyncUsesArtifactBuilder(
		ctx.Manifest,
		*targetFlag,
		*allTargets,
	)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if useArtifactBuilder {
		if err := buildCapsuleArtifacts(ctx.CapsulePath, capsuleArtifactBuildOptions{
			Target:     *targetFlag,
			LockPath:   lockPath,
			Jobs:       *jobs,
			AllTargets: *allTargets,
		}); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else {
		manifests, err := parseCapsuleGraphArgs([]string{ctx.CapsulePath})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := validateCapsuleGraph(manifests, *targetFlag); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := writeEcoLock(lockPath, manifests); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "Project synced: %s\n", ctx.Root)
	return 0
}

func checkProjectSync(
	ctx *cliProjectContext,
	targetFlag string,
	lockPath string,
	allTargets bool,
) ([]artifactIssue, error) {
	useArtifactBuilder, err := projectSyncUsesArtifactBuilder(ctx.Manifest, targetFlag, allTargets)
	if err != nil {
		return nil, err
	}
	if useArtifactBuilder {
		return checkCapsuleArtifacts(ctx.CapsulePath, targetFlag, lockPath, allTargets)
	}
	manifests, err := parseCapsuleGraphArgs([]string{ctx.CapsulePath})
	if err != nil {
		return nil, err
	}
	if err := validateCapsuleGraph(manifests, targetFlag); err != nil {
		return nil, err
	}
	return checkProjectLockOnly(lockPath, manifests)
}

func projectSyncUsesArtifactBuilder(
	manifest capsuleManifest,
	targetFlag string,
	allTargets bool,
) (bool, error) {
	targets, err := projectSyncNativeArtifactTargets(manifest, targetFlag, allTargets)
	if err != nil {
		return false, err
	}
	return len(targets) > 0, nil
}

func projectSyncNativeArtifactTargets(
	manifest capsuleManifest,
	targetFlag string,
	allTargets bool,
) ([]string, error) {
	return resolveNativeCapsuleTargets(manifest, nativeTargetResolutionOptions{
		TargetFlag: targetFlag,
		AllTargets: allTargets,
	})
}

func isWASMTargetTriple(triple string) bool {
	for _, wasmTriple := range ctarget.WASMTriples() {
		if triple == wasmTriple {
			return true
		}
	}
	return false
}

func checkProjectLockOnly(lockPath string, manifests []capsuleManifest) ([]artifactIssue, error) {
	if _, err := os.Stat(lockPath); err != nil {
		if os.IsNotExist(err) {
			return []artifactIssue{{
				Kind:   "missing lock",
				Path:   filepath.ToSlash(lockPath),
				Detail: "Tetra.lock is required for locked project builds",
			}}, nil
		}
		return nil, err
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, err
	}
	lock, err := decodeEcoLock(raw)
	if err != nil {
		return []artifactIssue{
			{Kind: "invalid lock", Path: filepath.ToSlash(lockPath), Detail: err.Error()},
		}, nil
	}
	current, err := buildEcoLockWithArtifactHashes(manifests)
	if err != nil {
		return nil, err
	}
	if lock.GraphSHA256 != current.GraphSHA256 {
		return []artifactIssue{{
			Kind: "stale lock",
			Path: filepath.ToSlash(lockPath),
			Detail: fmt.Sprintf(
				"expected graph %s, lock has %s",
				current.GraphSHA256,
				lock.GraphSHA256,
			),
		}}, nil
	}
	return nil, nil
}

func discoverProjectDepsContext(start string) (projectDepsContext, error) {
	startDir, err := cliProjectStartDir(start)
	if err != nil {
		return projectDepsContext{}, err
	}
	capsulePath, root, ok, err := findProjectCapsule(startDir)
	if err != nil {
		return projectDepsContext{}, err
	}
	if !ok {
		return projectDepsContext{}, fmt.Errorf("project capsule not found")
	}
	manifest, err := parseCapsule(capsulePath)
	if err != nil {
		return projectDepsContext{}, err
	}
	return projectDepsContext{Root: root, CapsulePath: capsulePath, Manifest: manifest}, nil
}

func buildProjectDepsReport(ctx projectDepsContext) projectDepsReport {
	report := projectDepsReport{
		Root:         ctx.Root,
		CapsulePath:  ctx.CapsulePath,
		Dependencies: []projectDependencyReport{},
	}
	for _, dep := range ctx.Manifest.Dependencies {
		report.Dependencies = append(report.Dependencies, describeProjectDependency(ctx.Root, dep))
	}
	return report
}

func describeProjectDependency(root string, dep capsuleDependency) projectDependencyReport {
	item := projectDependencyReport{
		ID:      dep.ID,
		Version: dep.Version,
		Path:    dep.Path,
		Status:  "ok",
	}
	if dep.Path == "" {
		item.Status = "missing"
		item.Detail = "dependency has no local path"
		return item
	}
	depRoot, err := resolveDependencyProjectRoot(root, dep.Path)
	if err != nil {
		item.Status = "missing"
		item.Detail = err.Error()
		return item
	}
	item.ResolvedPath = depRoot
	capsulePath, err := findCapsulePath(depRoot)
	if err != nil {
		item.Status = "missing"
		item.Detail = err.Error()
		return item
	}
	manifest, err := parseCapsule(capsulePath)
	if err != nil {
		item.Status = "invalid"
		item.Detail = err.Error()
		return item
	}
	if manifest.ID != dep.ID {
		item.Status = "mismatch"
		item.Detail = fmt.Sprintf("id mismatch: want %s, got %s", dep.ID, manifest.ID)
		return item
	}
	if manifest.Version != dep.Version {
		item.Status = "mismatch"
		item.Detail = fmt.Sprintf(
			"version mismatch: want %s, got %s",
			dep.Version,
			manifest.Version,
		)
		return item
	}
	return item
}

func writeProjectDepsText(w io.Writer, report projectDepsReport) {
	if len(report.Dependencies) == 0 {
		fmt.Fprintln(w, "Dependencies: none")
		return
	}
	fmt.Fprintln(w, "Dependencies:")
	for _, dep := range report.Dependencies {
		path := dep.Path
		if path == "" {
			path = "-"
		}
		fmt.Fprintf(w, "  %s %s %s %s\n", dep.ID, dep.Version, path, dep.Status)
		if dep.Detail != "" {
			fmt.Fprintf(w, "    detail: %s\n", dep.Detail)
		}
	}
}

func writeProjectDependencyIssues(w io.Writer, issues []projectDependencyReport) {
	for _, issue := range issues {
		if issue.ID == "" {
			fmt.Fprintln(w, issue.Detail)
			continue
		}
		path := issue.Path
		if path == "" {
			path = "-"
		}
		if issue.Detail != "" {
			fmt.Fprintf(w, "%s %s %s: %s\n", issue.ID, issue.Version, path, issue.Detail)
		} else {
			fmt.Fprintf(w, "%s %s %s: %s\n", issue.ID, issue.Version, path, issue.Status)
		}
	}
}

func resolveProjectDependencyAddPath(
	root string,
	depPath string,
) (string, capsuleManifest, string, error) {
	depPath = strings.TrimSpace(depPath)
	if depPath == "" {
		return "", capsuleManifest{}, "", fmt.Errorf("project deps add requires --path")
	}
	path := filepath.FromSlash(depPath)
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", capsuleManifest{}, "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", capsuleManifest{}, "", err
	}
	if !info.IsDir() {
		path = filepath.Dir(path)
	}
	capsulePath, err := findCapsulePath(path)
	if err != nil {
		return "", capsuleManifest{}, "", err
	}
	depRoot := filepath.Dir(capsulePath)
	if filepath.Clean(depRoot) == filepath.Clean(root) {
		return "", capsuleManifest{}, "", fmt.Errorf("project cannot depend on itself")
	}
	manifest, err := parseCapsule(capsulePath)
	if err != nil {
		return "", capsuleManifest{}, "", err
	}
	rel, err := filepath.Rel(root, depRoot)
	if err != nil {
		return "", capsuleManifest{}, "", err
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." || rel == "" {
		return "", capsuleManifest{}, "", fmt.Errorf("project cannot depend on itself")
	}
	if strings.ContainsAny(rel, " \t\r\n") {
		return "", capsuleManifest{}, "", fmt.Errorf(
			"dependency path %q contains whitespace, which Capsule.t4 deps do not support yet",
			rel,
		)
	}
	return depRoot, manifest, rel, nil
}

func normalizeProjectDependencyID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("dependency id is required")
	}
	if strings.ContainsAny(id, " \t\r\n") {
		return "", fmt.Errorf("dependency id must not contain whitespace")
	}
	if !strings.HasPrefix(id, "tetra://") {
		id = "tetra://" + id
	}
	return id, nil
}

func appendDependencyToCapsule(path string, dep capsuleDependency) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines, finalNewline := splitCapsuleText(raw)
	line := formatCapsuleDependencyLine(dep)
	if _, end, indent, ok := findCapsuleDepsSection(lines); ok {
		depIndent := indent + "    "
		lines = append(lines[:end], append([]string{depIndent + line}, lines[end:]...)...)
		return writeCapsuleText(path, lines, finalNewline)
	}
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
		lines = append(lines, "")
	}
	lines = append(lines, "    deps:", "        "+line)
	return writeCapsuleText(path, lines, true)
}

func removeDependencyFromCapsule(path string, id string, version string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	lines, finalNewline := splitCapsuleText(raw)
	section := ""
	var out []string
	removed := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			out = append(out, line)
			continue
		}
		if nextSection, ok := capsuleSectionHeader(trimmed); ok {
			section = nextSection
			out = append(out, line)
			continue
		}
		dep, depLine, err := parseDependencyLineForEdit(path, i+1, section, trimmed)
		if err != nil {
			return 0, err
		}
		if depLine && dep.ID == id && (version == "" || dep.Version == version) {
			removed++
			continue
		}
		out = append(out, line)
	}
	if removed == 0 {
		return 0, nil
	}
	return removed, writeCapsuleText(path, out, finalNewline)
}

func parseDependencyLineForEdit(
	path string,
	line int,
	section string,
	trimmed string,
) (capsuleDependency, bool, error) {
	if strings.HasPrefix(trimmed, "dependency ") {
		dep, err := parseCapsuleDependency(
			path,
			line,
			strings.TrimSpace(strings.TrimPrefix(trimmed, "dependency ")),
		)
		return dep, true, err
	}
	if section == "deps" {
		dep, err := parseCapsuleDependencyFields(path, line, strings.Fields(trimmed))
		return dep, true, err
	}
	return capsuleDependency{}, false, nil
}

func splitCapsuleText(raw []byte) ([]string, bool) {
	text := string(raw)
	finalNewline := strings.HasSuffix(text, "\n")
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil, finalNewline
	}
	return strings.Split(text, "\n"), finalNewline
}

func writeCapsuleText(path string, lines []string, finalNewline bool) error {
	text := strings.Join(lines, "\n")
	if finalNewline {
		text += "\n"
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

func findCapsuleDepsSection(lines []string) (int, int, string, bool) {
	section := ""
	start := -1
	indent := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if nextSection, ok := capsuleSectionHeader(trimmed); ok {
			if section == "deps" {
				return start, i, indent, true
			}
			section = nextSection
			if nextSection == "deps" {
				start = i
				indent = leadingWhitespace(line)
			}
			continue
		}
	}
	if section == "deps" {
		return start, len(lines), indent, true
	}
	return -1, -1, "", false
}

func leadingWhitespace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[:i]
}

func formatCapsuleDependencyLine(dep capsuleDependency) string {
	if dep.Path != "" {
		return dep.ID + " " + dep.Version + " " + dep.Path
	}
	return dep.ID + " " + dep.Version
}

func buildProjectInfoReport(start string) (projectInfoReport, error) {
	ctx, err := discoverCLIProject(start)
	if err != nil {
		return projectInfoReport{}, err
	}
	if ctx == nil || !ctx.Found {
		return projectInfoReport{Found: false}, nil
	}
	report := projectInfoReport{
		Found:              true,
		Root:               ctx.Root,
		CapsulePath:        ctx.CapsulePath,
		LockPath:           ctx.LockPath,
		EntryPath:          ctx.EntryPath,
		SourceRoots:        append([]string(nil), ctx.SourceRoots...),
		Targets:            append([]string(nil), ctx.Manifest.Targets...),
		ArtifactCounts:     map[string]int{},
		DependencyCapsules: []string{},
	}
	for _, root := range ctx.DependencyRoots {
		report.DependencyRoots = append(report.DependencyRoots, root.Root)
	}
	for _, manifest := range ctx.Manifests[1:] {
		report.DependencyCapsules = append(report.DependencyCapsules, manifest.Path)
	}
	for _, artifact := range ctx.Manifest.Artifacts {
		report.ArtifactCounts[artifact.Kind]++
	}
	sort.Strings(report.DependencyRoots)
	sort.Strings(report.DependencyCapsules)
	return report, nil
}

// ---- source_commands.go ----

func runDoc(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("doc", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outPath := fs.String("o", "", "output markdown path; stdout when empty")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		ctx, err := discoverCLIProject(".")
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if ctx != nil && ctx.Found {
			paths = existingProjectSourcePaths(ctx)
		}
	}
	docs, err := compiler.GenerateAPIDocs(paths)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if *outPath == "" {
		fmt.Fprint(stdout, string(docs))
		return 0
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if err := os.WriteFile(*outPath, docs, 0o644); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	fmt.Fprintf(stdout, "Wrote docs: %s\n", *outPath)
	return 0
}

func runCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	interfaceOnly := fs.Bool(
		"interface-only",
		false,
		"check interface/API surface without requiring executable output",
	)
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	input := ""
	if fs.NArg() > 0 {
		input = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "check accepts at most one input path")
		return 2
	}
	input, worldOpt, projectCtx, err := resolveCLIInput(input)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if err := validateDiscoveredProjectLock(projectCtx, ""); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	world, err := compiler.LoadWorldOpt(input, worldOpt)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	checkOpt := compiler.CheckOptions{RequireMain: true}
	if *interfaceOnly {
		checkOpt.RequireMain = false
	}
	if _, err := compiler.CheckWorldOpt(world, checkOpt); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	fmt.Fprintf(stdout, "Checked: %s\n", input)
	return 0
}

func runFmt(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("fmt", flag.ContinueOnError)
	fs.SetOutput(stderr)
	check := fs.Bool("check", false, "check whether files are formatted")
	write := fs.Bool("write", false, "rewrite files in place")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if *check && *write {
		writeValidationDiagnostic(
			stderr,
			*diagnostics,
			"fmt accepts only one of --check or --write",
		)
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		writeValidationDiagnostic(stderr, *diagnostics, "fmt requires at least one path")
		return 2
	}
	files, err := collectTetraFiles(paths)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if !*check && !*write && len(files) != 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "fmt stdout mode accepts exactly one file")
		return 2
	}
	dirty := false
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		formatted, err := compiler.FormatSource(raw, path)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if *check {
			if string(raw) != string(formatted) {
				dirty = true
				if *diagnostics == "json" || *diagnostics == "toon" {
					line, column := firstFormatterDiffPosition(raw, formatted)
					writeDiagnosticObject(stderr, *diagnostics, compiler.Diagnostic{
						Code:     compiler.DiagnosticCodeFormatterCheck,
						Message:  "not formatted",
						File:     path,
						Line:     line,
						Column:   column,
						Severity: "error",
						Hint:     "Run tetra fmt --write to update the file.",
					})
				} else {
					fmt.Fprintf(stderr, "%s: not formatted\n", path)
				}
			}
			continue
		}
		if *write {
			if string(raw) != string(formatted) {
				if err := os.WriteFile(path, formatted, 0o644); err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
			}
			continue
		}
		fmt.Fprint(stdout, string(formatted))
	}
	if dirty {
		return 1
	}
	return 0
}

func firstFormatterDiffPosition(raw []byte, formatted []byte) (int, int) {
	line := 1
	column := 1
	limit := len(raw)
	if len(formatted) < limit {
		limit = len(formatted)
	}
	for i := 0; i < limit; i++ {
		if raw[i] != formatted[i] {
			return line, column
		}
		if raw[i] == '\n' {
			line++
			column = 1
			continue
		}
		column++
	}
	return line, column
}

// ---- source_files.go ----

func collectTetraFiles(paths []string) ([]string, error) {
	seen := map[string]struct{}{}
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if isCLIManagedSourceFile(path) {
				if _, ok := seen[path]; !ok {
					seen[path] = struct{}{}
					files = append(files, path)
				}
			}
			continue
		}
		err = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") && p != path {
					return filepath.SkipDir
				}
				return nil
			}
			if isCLIManagedSourceFile(p) {
				if _, ok := seen[p]; !ok {
					seen[p] = struct{}{}
					files = append(files, p)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

func isCLIManagedSourceFile(path string) bool {
	if !compiler.IsSourceFile(path) {
		return false
	}
	base := filepath.Base(path)
	return base != compiler.CapsuleFileName && base != compiler.LegacyCapsuleFileName
}

var moduleDeclRE = regexp.MustCompile(`(?m)^\s*module\s+([A-Za-z0-9_.]+)\s*$`)

func modulePathFromSource(src []byte) string {
	m := moduleDeclRE.FindSubmatch(src)
	if len(m) != 2 {
		return ""
	}
	return string(m[1])
}

func moduleRelPath(module string) string {
	return moduleRelPathWithExtension(module, compiler.T4SourceExtension)
}

func moduleRootFromEntry(entryPath string, module string) (string, error) {
	absEntry, err := filepath.Abs(entryPath)
	if err != nil {
		return "", fmt.Errorf("resolve module root: %w", err)
	}
	root := absEntry
	for range strings.Split(module, ".") {
		root = filepath.Dir(root)
	}
	rel, err := filepath.Rel(root, absEntry)
	if err != nil {
		return "", fmt.Errorf("resolve module root: %w", err)
	}
	if !cliModuleRelPathMatches(module, rel) {
		return "", fmt.Errorf(
			"%s: module '%s' must be in %s (or legacy %s)",
			absEntry,
			module,
			moduleRelPathWithExtension(module, compiler.T4SourceExtension),
			moduleRelPathWithExtension(module, compiler.LegacyTetraSourceExtension),
		)
	}
	return root, nil
}

func defaultInputPath() string {
	if fileExists(compiler.DefaultSourceFileName) {
		return compiler.DefaultSourceFileName
	}
	if fileExists(compiler.LegacySourceFileName) {
		return compiler.LegacySourceFileName
	}
	return compiler.DefaultSourceFileName
}

func moduleRelPathWithExtension(module string, extension string) string {
	return filepath.FromSlash(strings.ReplaceAll(module, ".", "/") + extension)
}

func cliModuleRelPathMatches(module, rel string) bool {
	cleanRel := filepath.Clean(rel)
	for _, ext := range compiler.SourceExtensions() {
		if cleanRel == filepath.Clean(moduleRelPathWithExtension(module, ext)) {
			return true
		}
	}
	return false
}

func rewriteModuleDecl(src []byte, module string) []byte {
	return []byte(moduleDeclRE.ReplaceAllString(string(src), "module "+module))
}

func runnerSourcePathForModuleFile(
	entryPath string,
	src []byte,
	runnerIndex int,
) (string, []byte, error) {
	module := modulePathFromSource(src)
	if module == "" {
		return "", nil, fmt.Errorf("runner source has imports but no module declaration")
	}
	root, err := moduleRootFromEntry(entryPath, module)
	if err != nil {
		return "", nil, err
	}
	parts := strings.Split(module, ".")
	parts[len(parts)-1] = fmt.Sprintf("__tetra_test_runner_%d", runnerIndex)
	runnerModule := strings.Join(parts, ".")
	runnerPath := filepath.Join(root, moduleRelPath(runnerModule))
	return runnerPath, rewriteModuleDecl(src, runnerModule), nil
}

// ---- workspace.go ----

func runWorkspace(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(
			stderr,
			"usage: tetra workspace <init|add|remove|list|check|graph|sync|build|test|run> [options]",
		)
		return 2
	}
	if isHelpArgs(args) {
		fmt.Fprintln(
			stdout,
			"usage: tetra workspace <init|add|remove|list|check|graph|sync|build|test|run> [options]",
		)
		return 0
	}
	switch args[0] {
	case "init":
		return runWorkspaceInit(args[1:], stdout, stderr)
	case "add":
		return runWorkspaceAdd(args[1:], stdout, stderr)
	case "remove":
		return runWorkspaceRemove(args[1:], stdout, stderr)
	case "list":
		return runWorkspaceList(args[1:], stdout, stderr)
	case "check":
		return runWorkspaceCheck(args[1:], stdout, stderr)
	case "graph":
		return runWorkspaceGraph(args[1:], stdout, stderr)
	case "sync":
		return runWorkspaceSync(args[1:], stdout, stderr)
	case "build":
		return runWorkspaceBuild(args[1:], stdout, stderr)
	case "test":
		return runWorkspaceTest(args[1:], stdout, stderr)
	case "run":
		return runWorkspaceRun(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown workspace command %q\n", args[0])
		return 2
	}
}

func runWorkspaceInit(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "workspace init accepts at most one path")
		return 2
	}
	root := "."
	if len(args) == 1 {
		root = args[0]
	}
	absRoot, err := filepath.Abs(filepath.FromSlash(root))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.MkdirAll(absRoot, 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	path := filepath.Join(absRoot, workspaceFileName)
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, "%s already exists\n", path)
		return 2
	} else if !os.IsNotExist(err) {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.WriteFile(
		path,
		[]byte(fmt.Sprintf("workspace %q\n", workspaceSchemaV1)),
		0o644,
	); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Created workspace: %s\n", path)
	return 0
}

func runWorkspaceAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	memberPath, workspaceStart, code, err := parseWorkspaceMemberMutationArgs("workspace add", args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return code
	}
	workspace, err := loadWorkspace(workspaceStart)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	memberRel, err := normalizeWorkspaceMemberArg(workspace.Root, memberPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	memberRoot := filepath.Join(workspace.Root, filepath.FromSlash(memberRel))
	if _, err := findCapsulePath(memberRoot); err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", memberRel, err)
		return 1
	}
	for _, existing := range workspace.Members {
		if existing == memberRel {
			fmt.Fprintf(stderr, "duplicate workspace member %s\n", memberRel)
			return 1
		}
	}
	workspace.Members = append(workspace.Members, memberRel)
	if err := writeWorkspaceManifest(workspace); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Added workspace member: %s\n", memberRel)
	return 0
}

func runWorkspaceRemove(args []string, stdout io.Writer, stderr io.Writer) int {
	memberPath, workspaceStart, code, err := parseWorkspaceMemberMutationArgs(
		"workspace remove",
		args,
	)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return code
	}
	workspace, err := loadWorkspace(workspaceStart)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	memberRel, err := normalizeWorkspaceMemberArg(workspace.Root, memberPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	var out []string
	removed := false
	for _, member := range workspace.Members {
		if member == memberRel {
			removed = true
			continue
		}
		out = append(out, member)
	}
	if !removed {
		fmt.Fprintf(stderr, "workspace member not found: %s\n", memberRel)
		return 1
	}
	workspace.Members = out
	if err := writeWorkspaceManifest(workspace); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Removed workspace member: %s\n", memberRel)
	return 0
}

func runWorkspaceList(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("workspace list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "workspace list accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	workspace, err := loadWorkspace(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	report := workspaceReport{
		Root:          workspace.Root,
		WorkspacePath: workspace.Path,
		Members:       describeWorkspaceMembers(workspace),
	}
	switch *format {
	case "text", "":
		writeWorkspaceListText(stdout, report)
		return 0
	case "json":
		return encodeWorkspaceJSON(stdout, stderr, report, false)
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runWorkspaceCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("workspace check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "workspace check accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	graph, err := buildWorkspaceGraph(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	report := workspaceReport{
		Root:          graph.Workspace.Root,
		WorkspacePath: graph.Workspace.Path,
		Members:       graph.Nodes,
		Status:        "pass",
	}
	if len(graph.Issues) > 0 {
		report.Status = "fail"
	}
	switch *format {
	case "text", "":
		if report.Status == "pass" {
			fmt.Fprintf(stdout, "Workspace OK: %d member(s)\n", len(report.Members))
			return 0
		}
		writeWorkspaceIssues(stderr, graph.Issues)
		return 1
	case "json":
		code := 0
		if report.Status != "pass" {
			code = 1
		}
		return encodeWorkspaceJSON(stdout, stderr, report, code != 0)
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runWorkspaceGraph(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("workspace graph", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "workspace graph accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	graph, err := buildWorkspaceGraph(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	status := "pass"
	if len(graph.Issues) > 0 {
		status = "fail"
	}
	report := workspaceGraphReport{
		Status:        status,
		Root:          graph.Workspace.Root,
		WorkspacePath: graph.Workspace.Path,
		Nodes:         graph.Nodes,
		Edges:         graph.Edges,
	}
	switch *format {
	case "text", "":
		writeWorkspaceGraphText(stdout, report)
		if status != "pass" {
			return 1
		}
		return 0
	case "json":
		code := 0
		if status != "pass" {
			code = 1
		}
		return encodeWorkspaceJSON(stdout, stderr, report, code != 0)
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func runWorkspaceSync(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("workspace sync", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetFlag := fs.String("target", "", "native target triple for generated .tobj artifacts")
	checkOnly := fs.Bool(
		"check",
		false,
		"dry-run and report pending project lock/artifact changes without writing files",
	)
	allTargets := fs.Bool(
		"all-targets",
		false,
		"sync artifacts for every native target listed in member Capsule.t4",
	)
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *targetFlag != "" && *allTargets {
		fmt.Fprintln(stderr, "workspace sync accepts either --target or --all-targets, not both")
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "workspace sync accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	graph, err := buildWorkspaceGraph(start)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(graph.Issues) > 0 {
		writeWorkspaceIssues(stderr, graph.Issues)
		return 1
	}
	order, err := workspaceSyncOrder(graph)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	for _, member := range order {
		ctx := workspaceProjectContext(member)
		lockPath := filepath.Join(ctx.Root, compiler.SemanticLockFileName)
		if *checkOnly {
			issues, err := checkProjectSync(ctx, *targetFlag, lockPath, *allTargets)
			if err != nil {
				fmt.Fprintf(stderr, "%s: %v\n", member.Path, err)
				return 1
			}
			for i := range issues {
				issues[i].Repair = "tetra workspace sync " + filepath.ToSlash(graph.Workspace.Root)
			}
			if len(issues) > 0 {
				writeArtifactIssues(stdout, issues, true)
				return 1
			}
			continue
		}
		if err := syncWorkspaceProject(ctx, *targetFlag, *allTargets, *jobs); err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", member.Path, err)
			return 1
		}
		fmt.Fprintf(stdout, "Workspace synced: %s\n", member.Path)
	}
	if *checkOnly {
		fmt.Fprintf(stdout, "Workspace current: %s\n", graph.Workspace.Root)
	}
	return 0
}

func runWorkspaceBuild(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("workspace build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetFlag := fs.String(
		"target",
		"",
		"target triple ("+supportedTargetsHelp+("); defaults to each member Capsule.t4 first "+
			"target, then host"),
	)
	allTargets := fs.Bool(
		"all-targets",
		false,
		"build every target listed in each member Capsule.t4",
	)
	interfaceOnly := fs.Bool(
		"interface-only",
		false,
		"type-check interface/API graph without emitting executable code",
	)
	islandsDebug := fs.Bool("islands-debug", false, "enable islands debug runtime checks")
	emit := fs.String("emit", "exe", "emit mode: exe, object, or library")
	runtimeMode := fs.String("runtime", "auto", "actors runtime: auto, selfhost, or builtin")
	runtimeObject := fs.String("runtime-object", "", "actors runtime object override")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	artifactsMode := fs.String("artifacts", "strict", "artifact handling: strict or auto")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	format := fs.String("format", "text", "workspace report format: text or json")
	outDir := ""
	fs.StringVar(&outDir, "o", "", "workspace output directory")
	fs.StringVar(&outDir, "out-dir", "", "workspace output directory")
	var linkObjects multiFlag
	fs.Var(&linkObjects, "link-object", "extra TOBJ object to link")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateWorkspaceReportFormat(stderr, *format) {
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if *targetFlag != "" && *allTargets {
		writeValidationDiagnostic(
			stderr,
			*diagnostics,
			"workspace build accepts either --target or --all-targets, not both",
		)
		return 2
	}
	if *artifactsMode != "strict" && *artifactsMode != "auto" {
		writeValidationDiagnostic(
			stderr,
			*diagnostics,
			"workspace build --artifacts must be strict or auto",
		)
		return 2
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "workspace build accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	graph, err := buildWorkspaceGraph(start)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if len(graph.Issues) > 0 {
		writeWorkspaceIssues(stderr, graph.Issues)
		return 1
	}
	order, err := workspaceSyncOrder(graph)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	report := workspaceExecutionReport{
		WorkspaceRoot: graph.Workspace.Root,
		Command:       "build",
		Target:        workspaceExecutionTargetLabel(*targetFlag, *allTargets),
		Total:         len(order),
	}
	statusByPath := map[string]string{}
	for _, member := range order {
		if dep, blocked := workspaceBlockedDependency(member.Path, graph, statusByPath); blocked {
			item := workspaceExecutionItem(
				member,
				"skipped",
				fmt.Sprintf("blocked by failed dependency %s", dep),
				0,
				false,
			)
			appendWorkspaceExecutionMember(&report, item)
			statusByPath[member.Path] = item.Status
			continue
		}
		memberArgs, err := workspaceBuildMemberArgs(member, workspaceBuildOptions{
			Target:        *targetFlag,
			AllTargets:    *allTargets,
			InterfaceOnly: *interfaceOnly,
			IslandsDebug:  *islandsDebug,
			Emit:          *emit,
			RuntimeMode:   *runtimeMode,
			RuntimeObject: *runtimeObject,
			Jobs:          *jobs,
			ArtifactsMode: *artifactsMode,
			Diagnostics:   *diagnostics,
			OutDir:        outDir,
			LinkObjects:   []string(linkObjects),
		})
		if err != nil {
			item := workspaceExecutionItem(member, "fail", err.Error(), 2, true)
			appendWorkspaceExecutionMember(&report, item)
			statusByPath[member.Path] = item.Status
			continue
		}
		var memberStdout, memberStderr bytes.Buffer
		code := runBuild(memberArgs, &memberStdout, &memberStderr)
		status := "pass"
		if code != 0 {
			status = "fail"
		}
		item := workspaceExecutionItem(
			member,
			status,
			workspaceCommandDetail(memberStdout, memberStderr),
			code,
			true,
		)
		appendWorkspaceExecutionMember(&report, item)
		statusByPath[member.Path] = item.Status
	}
	if err := writeWorkspaceExecutionReport(stdout, report, *format); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return workspaceExecutionExitCode(report)
}

type workspaceBuildOptions struct {
	Target        string
	AllTargets    bool
	InterfaceOnly bool
	IslandsDebug  bool
	Emit          string
	RuntimeMode   string
	RuntimeObject string
	Jobs          int
	ArtifactsMode string
	Diagnostics   string
	OutDir        string
	LinkObjects   []string
}

func workspaceBuildMemberArgs(
	member workspaceMemberReport,
	opt workspaceBuildOptions,
) ([]string, error) {
	args := []string{}
	if opt.Target != "" {
		args = append(args, "--target", opt.Target)
	}
	if opt.AllTargets {
		args = append(args, "--all-targets")
	}
	if opt.InterfaceOnly {
		args = append(args, "--interface-only")
	}
	if opt.IslandsDebug {
		args = append(args, "--islands-debug")
	}
	if opt.Emit != "" {
		args = append(args, "--emit", opt.Emit)
	}
	if opt.RuntimeMode != "" {
		args = append(args, "--runtime", opt.RuntimeMode)
	}
	if opt.RuntimeObject != "" {
		args = append(args, "--runtime-object", opt.RuntimeObject)
	}
	args = append(args, "--jobs", strconv.Itoa(opt.Jobs))
	if opt.ArtifactsMode != "" {
		args = append(args, "--artifacts", opt.ArtifactsMode)
	}
	if opt.Diagnostics != "" {
		args = append(args, "--diagnostics", opt.Diagnostics)
	}
	for _, path := range opt.LinkObjects {
		args = append(args, "--link-object", path)
	}
	output, err := workspaceBuildOutputPath(member, opt)
	if err != nil {
		return nil, err
	}
	if output != "" {
		args = append(args, "-o", output)
	}
	args = append(args, member.ResolvedPath)
	return args, nil
}

func workspaceBuildOutputPath(
	member workspaceMemberReport,
	opt workspaceBuildOptions,
) (string, error) {
	base := member.ResolvedPath
	if opt.OutDir != "" {
		base = filepath.Join(opt.OutDir, filepath.FromSlash(member.Path))
	}
	if opt.AllTargets {
		if err := os.MkdirAll(base, 0o755); err != nil {
			return "", err
		}
		return base, nil
	}
	rawTarget := opt.Target
	if rawTarget == "" {
		rawTarget = projectDefaultTarget(workspaceProjectContext(member))
	}
	tgt, err := ctarget.Parse(rawTarget)
	if err != nil {
		return "", err
	}
	output := filepath.Join(base, defaultOutput(tgt, opt.Emit))
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return "", err
	}
	return output, nil
}

func runWorkspaceTest(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("workspace test", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", defaultTarget(), "target triple ("+supportedTargetsHelp+")")
	failFast := fs.Bool("fail-fast", false, "stop after the first failed member")
	format := fs.String("format", "text", "workspace report format: text or json")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text, json, or toon")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateWorkspaceReportFormat(stderr, *format) {
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if fs.NArg() > 1 {
		writeValidationDiagnostic(stderr, *diagnostics, "workspace test accepts at most one path")
		return 2
	}
	start := "."
	if fs.NArg() == 1 {
		start = fs.Arg(0)
	}
	graph, err := buildWorkspaceGraph(start)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	if len(graph.Issues) > 0 {
		writeWorkspaceIssues(stderr, graph.Issues)
		return 1
	}
	order, err := workspaceSyncOrder(graph)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	report := workspaceExecutionReport{
		WorkspaceRoot: graph.Workspace.Root,
		Command:       "test",
		Target:        *target,
		Total:         len(order),
	}
	statusByPath := map[string]string{}
	failFastAfter := ""
	for _, member := range order {
		if failFastAfter != "" {
			item := workspaceExecutionItem(
				member,
				"skipped",
				fmt.Sprintf("fail-fast after %s", failFastAfter),
				0,
				false,
			)
			appendWorkspaceExecutionMember(&report, item)
			statusByPath[member.Path] = item.Status
			continue
		}
		if dep, blocked := workspaceBlockedDependency(member.Path, graph, statusByPath); blocked {
			item := workspaceExecutionItem(
				member,
				"skipped",
				fmt.Sprintf("blocked by failed dependency %s", dep),
				0,
				false,
			)
			appendWorkspaceExecutionMember(&report, item)
			statusByPath[member.Path] = item.Status
			continue
		}
		memberArgs := []string{
			"--target",
			*target,
			"--diagnostics",
			*diagnostics,
			"--report",
			"text",
			member.ResolvedPath,
		}
		var memberStdout, memberStderr bytes.Buffer
		code := runTest(memberArgs, &memberStdout, &memberStderr)
		status := "pass"
		if code != 0 {
			status = "fail"
		}
		item := workspaceExecutionItem(
			member,
			status,
			workspaceCommandDetail(memberStdout, memberStderr),
			code,
			true,
		)
		appendWorkspaceExecutionMember(&report, item)
		statusByPath[member.Path] = item.Status
		if status == "fail" && *failFast {
			failFastAfter = member.Path
		}
	}
	if err := writeWorkspaceExecutionReport(stdout, report, *format); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return workspaceExecutionExitCode(report)
}

type workspaceRunOptions struct {
	Member         string
	WorkspaceStart string
	Target         string
	Output         string
	IslandsDebug   bool
	RuntimeMode    string
	RuntimeObject  string
	Jobs           int
	ArtifactsMode  string
	Diagnostics    string
}

func runWorkspaceRun(args []string, stdout io.Writer, stderr io.Writer) int {
	opt, code, err := parseWorkspaceRunArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return code
	}
	if !validateDiagnosticsMode(stderr, opt.Diagnostics) {
		return 2
	}
	if opt.ArtifactsMode != "strict" && opt.ArtifactsMode != "auto" {
		writeValidationDiagnostic(
			stderr,
			opt.Diagnostics,
			"workspace run --artifacts must be strict or auto",
		)
		return 2
	}
	graph, err := buildWorkspaceGraph(opt.WorkspaceStart)
	if err != nil {
		writeDiagnostic(stderr, opt.Diagnostics, err)
		return 1
	}
	if len(graph.Issues) > 0 {
		writeWorkspaceIssues(stderr, graph.Issues)
		return 1
	}
	memberRel, err := normalizeWorkspaceMemberArg(graph.Workspace.Root, opt.Member)
	if err != nil {
		writeValidationDiagnostic(stderr, opt.Diagnostics, err.Error())
		return 2
	}
	member, ok := workspaceFindMember(graph, memberRel)
	if !ok {
		writeValidationDiagnostic(stderr, opt.Diagnostics, "workspace member not found: "+memberRel)
		return 2
	}
	if member.Status != "ok" {
		writeDiagnostic(stderr, opt.Diagnostics, fmt.Errorf("%s: %s", member.Path, member.Detail))
		return 1
	}
	if opt.ArtifactsMode == "auto" {
		if err := syncWorkspaceProject(
			workspaceProjectContext(member),
			opt.Target,
			false,
			opt.Jobs,
		); err != nil {
			writeDiagnostic(stderr, opt.Diagnostics, err)
			return 1
		}
	}
	runArgs := workspaceRunMemberArgs(member, opt)
	return runRun(runArgs, stdout, stderr)
}

func parseWorkspaceRunArgs(args []string) (workspaceRunOptions, int, error) {
	opt := workspaceRunOptions{
		WorkspaceStart: ".",
		RuntimeMode:    "auto",
		Jobs:           1,
		ArtifactsMode:  "strict",
		Diagnostics:    "text",
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--workspace":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			opt.WorkspaceStart = value
			i = next
		case strings.HasPrefix(arg, "--workspace="):
			opt.WorkspaceStart = strings.TrimPrefix(arg, "--workspace=")
		case arg == "--target":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			opt.Target = value
			i = next
		case strings.HasPrefix(arg, "--target="):
			opt.Target = strings.TrimPrefix(arg, "--target=")
		case arg == "-o":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			opt.Output = value
			i = next
		case strings.HasPrefix(arg, "-o="):
			opt.Output = strings.TrimPrefix(arg, "-o=")
		case arg == "--islands-debug":
			opt.IslandsDebug = true
		case arg == "--runtime":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			opt.RuntimeMode = value
			i = next
		case strings.HasPrefix(arg, "--runtime="):
			opt.RuntimeMode = strings.TrimPrefix(arg, "--runtime=")
		case arg == "--runtime-object":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			opt.RuntimeObject = value
			i = next
		case strings.HasPrefix(arg, "--runtime-object="):
			opt.RuntimeObject = strings.TrimPrefix(arg, "--runtime-object=")
		case arg == "--jobs":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			jobs, err := strconv.Atoi(value)
			if err != nil {
				return opt, 2, fmt.Errorf("workspace run --jobs must be an integer")
			}
			opt.Jobs = jobs
			i = next
		case strings.HasPrefix(arg, "--jobs="):
			jobs, err := strconv.Atoi(strings.TrimPrefix(arg, "--jobs="))
			if err != nil {
				return opt, 2, fmt.Errorf("workspace run --jobs must be an integer")
			}
			opt.Jobs = jobs
		case arg == "--artifacts":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			opt.ArtifactsMode = value
			i = next
		case strings.HasPrefix(arg, "--artifacts="):
			opt.ArtifactsMode = strings.TrimPrefix(arg, "--artifacts=")
		case arg == "--diagnostics":
			value, next, err := workspaceRunValue(args, i, arg)
			if err != nil {
				return opt, 2, err
			}
			opt.Diagnostics = value
			i = next
		case strings.HasPrefix(arg, "--diagnostics="):
			opt.Diagnostics = strings.TrimPrefix(arg, "--diagnostics=")
		case strings.HasPrefix(arg, "-"):
			return opt, 2, fmt.Errorf("unknown workspace run option %q", arg)
		default:
			if opt.Member != "" {
				return opt, 2, fmt.Errorf("workspace run accepts exactly one member path")
			}
			opt.Member = arg
		}
	}
	if opt.Member == "" {
		return opt, 2, fmt.Errorf("workspace run requires a member path")
	}
	return opt, 0, nil
}

func workspaceRunValue(args []string, index int, flagName string) (string, int, error) {
	if index+1 >= len(args) {
		return "", index, fmt.Errorf("workspace run requires %s value", flagName)
	}
	return args[index+1], index + 1, nil
}

func workspaceRunMemberArgs(member workspaceMemberReport, opt workspaceRunOptions) []string {
	args := []string{}
	if opt.Target != "" {
		args = append(args, "--target", opt.Target)
	}
	if opt.Output != "" {
		args = append(args, "-o", opt.Output)
	}
	if opt.IslandsDebug {
		args = append(args, "--islands-debug")
	}
	if opt.RuntimeMode != "" {
		args = append(args, "--runtime", opt.RuntimeMode)
	}
	if opt.RuntimeObject != "" {
		args = append(args, "--runtime-object", opt.RuntimeObject)
	}
	args = append(
		args,
		"--jobs",
		strconv.Itoa(opt.Jobs),
		"--diagnostics",
		opt.Diagnostics,
		member.ResolvedPath,
	)
	return args
}

func validateWorkspaceReportFormat(stderr io.Writer, format string) bool {
	if format == "text" || format == "json" {
		return true
	}
	fmt.Fprintln(stderr, "workspace --format must be text or json")
	return false
}

func workspaceExecutionTargetLabel(target string, allTargets bool) string {
	if allTargets {
		return "all-targets"
	}
	return target
}

func workspaceBlockedDependency(
	path string,
	graph workspaceGraph,
	statusByPath map[string]string,
) (string, bool) {
	for _, edge := range graph.Edges {
		if edge.From != path {
			continue
		}
		status := statusByPath[edge.To]
		if status == "fail" || status == "skipped" {
			return edge.To, true
		}
	}
	return "", false
}

func workspaceFindMember(graph workspaceGraph, memberPath string) (workspaceMemberReport, bool) {
	for _, member := range graph.Nodes {
		if member.Path == memberPath {
			return member, true
		}
	}
	return workspaceMemberReport{}, false
}

func workspaceExecutionItem(
	member workspaceMemberReport,
	status string,
	detail string,
	exitCode int,
	includeExitCode bool,
) workspaceExecutionMemberReport {
	item := workspaceExecutionMemberReport{
		Path:      member.Path,
		CapsuleID: member.CapsuleID,
		Status:    status,
		Detail:    strings.TrimSpace(detail),
	}
	if includeExitCode {
		code := exitCode
		item.ExitCode = &code
	}
	return item
}

func appendWorkspaceExecutionMember(
	report *workspaceExecutionReport,
	item workspaceExecutionMemberReport,
) {
	report.Members = append(report.Members, item)
	switch item.Status {
	case "pass":
		report.Passed++
	case "fail":
		report.Failed++
	case "skipped":
		report.Skipped++
	}
}

func workspaceCommandDetail(stdout bytes.Buffer, stderr bytes.Buffer) string {
	if detail := strings.TrimSpace(stderr.String()); detail != "" {
		return detail
	}
	return strings.TrimSpace(stdout.String())
}

func writeWorkspaceExecutionReport(
	w io.Writer,
	report workspaceExecutionReport,
	format string,
) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	default:
		for _, member := range report.Members {
			label := strings.ToUpper(member.Status)
			if member.Detail != "" && member.Status != "pass" {
				fmt.Fprintf(
					w,
					"%s %s: %s\n",
					label,
					member.Path,
					firstWorkspaceDetailLine(member.Detail),
				)
			} else {
				fmt.Fprintf(w, "%s %s\n", label, member.Path)
			}
		}
		fmt.Fprintf(
			w,
			"Workspace %s: %d/%d passed, %d failed, %d skipped\n",
			report.Command,
			report.Passed,
			report.Total,
			report.Failed,
			report.Skipped,
		)
		return nil
	}
}

func firstWorkspaceDetailLine(detail string) string {
	for _, line := range strings.Split(detail, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func workspaceExecutionExitCode(report workspaceExecutionReport) int {
	if report.Failed == 0 && report.Skipped == 0 {
		return 0
	}
	for _, member := range report.Members {
		if member.Status == "fail" && member.ExitCode != nil && *member.ExitCode == 2 {
			return 2
		}
	}
	return 1
}

func parseWorkspaceMemberMutationArgs(command string, args []string) (string, string, int, error) {
	workspaceStart := "."
	var member string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--workspace" {
			if i+1 >= len(args) {
				return "", "", 2, fmt.Errorf("%s requires --workspace value", command)
			}
			workspaceStart = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--workspace=") {
			workspaceStart = strings.TrimPrefix(arg, "--workspace=")
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return "", "", 2, fmt.Errorf("unknown %s option %q", command, arg)
		}
		if member != "" {
			return "", "", 2, fmt.Errorf("%s accepts exactly one member path", command)
		}
		member = arg
	}
	if member == "" {
		return "", "", 2, fmt.Errorf("%s requires a member path", command)
	}
	return member, workspaceStart, 0, nil
}

func loadWorkspace(start string) (workspaceManifest, error) {
	path, root, err := findWorkspacePath(start)
	if err != nil {
		return workspaceManifest{}, err
	}
	return parseWorkspace(path, root)
}

func findWorkspacePath(start string) (string, string, error) {
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	abs, err := filepath.Abs(filepath.FromSlash(start))
	if err != nil {
		return "", "", err
	}
	info, err := os.Stat(abs)
	if err == nil && !info.IsDir() && filepath.Base(abs) == workspaceFileName {
		return abs, filepath.Dir(abs), nil
	}
	if err != nil && !os.IsNotExist(err) {
		return "", "", err
	}
	dir := abs
	if err == nil && !info.IsDir() {
		dir = filepath.Dir(abs)
	}
	for {
		path := filepath.Join(dir, workspaceFileName)
		if _, err := os.Stat(path); err == nil {
			return path, dir, nil
		} else if !os.IsNotExist(err) {
			return "", "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", fmt.Errorf("%s not found", workspaceFileName)
		}
		dir = parent
	}
}

func parseWorkspace(path string, root string) (workspaceManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return workspaceManifest{}, err
	}
	workspace := workspaceManifest{Path: path, Root: root, Schema: workspaceSchemaV1}
	seenMembers := map[string]struct{}{}
	sawWorkspace := false
	for i, line := range strings.Split(string(raw), "\n") {
		content := strings.TrimSpace(line)
		if content == "" || strings.HasPrefix(content, "#") || strings.HasPrefix(content, "//") {
			continue
		}
		if strings.HasPrefix(content, "workspace ") {
			if sawWorkspace {
				return workspaceManifest{}, fmt.Errorf(
					"%s:%d: duplicate workspace declaration",
					path,
					i+1,
				)
			}
			value, err := parseCapsuleString(
				path,
				i+1,
				strings.TrimSpace(strings.TrimPrefix(content, "workspace ")),
			)
			if err != nil {
				return workspaceManifest{}, err
			}
			if value != workspaceSchemaV1 {
				return workspaceManifest{}, fmt.Errorf(
					"%s:%d: unsupported workspace schema %s",
					path,
					i+1,
					value,
				)
			}
			workspace.Schema = value
			sawWorkspace = true
			continue
		}
		if strings.HasPrefix(content, "member ") {
			value, err := parseCapsuleBareOrQuoted(
				path,
				i+1,
				strings.TrimSpace(strings.TrimPrefix(content, "member ")),
			)
			if err != nil {
				return workspaceManifest{}, err
			}
			member, err := cleanWorkspaceMemberPath(value)
			if err != nil {
				return workspaceManifest{}, fmt.Errorf("%s:%d: %v", path, i+1, err)
			}
			if _, ok := seenMembers[member]; ok {
				return workspaceManifest{}, fmt.Errorf(
					"%s:%d: duplicate workspace member %s",
					path,
					i+1,
					member,
				)
			}
			seenMembers[member] = struct{}{}
			workspace.Members = append(workspace.Members, member)
			continue
		}
		return workspaceManifest{}, fmt.Errorf("%s:%d: unknown workspace field", path, i+1)
	}
	if !sawWorkspace {
		return workspaceManifest{}, fmt.Errorf("%s: missing workspace declaration", path)
	}
	return workspace, nil
}

func writeWorkspaceManifest(workspace workspaceManifest) error {
	lines := []string{fmt.Sprintf("workspace %q", workspace.Schema)}
	for _, member := range workspace.Members {
		lines = append(lines, fmt.Sprintf("member %q", member))
	}
	lines = append(lines, "")
	return os.WriteFile(workspace.Path, []byte(strings.Join(lines, "\n")), 0o644)
}

func normalizeWorkspaceMemberArg(root string, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("workspace member path is required")
	}
	path := filepath.FromSlash(value)
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	return cleanWorkspaceMemberPath(filepath.ToSlash(rel))
}

func cleanWorkspaceMemberPath(value string) (string, error) {
	if strings.Contains(value, "\\") {
		return "", fmt.Errorf("member path must use forward slashes")
	}
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("member path must be relative")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean == "." || clean == "" {
		return "", fmt.Errorf("member path must not be empty")
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("member path must stay inside workspace root")
	}
	if strings.ContainsAny(clean, "\r\n\t") {
		return "", fmt.Errorf("member path must not contain control whitespace")
	}
	return clean, nil
}

func buildWorkspaceGraph(start string) (workspaceGraph, error) {
	workspace, err := loadWorkspace(start)
	if err != nil {
		return workspaceGraph{}, err
	}
	graph := workspaceGraph{Workspace: workspace, ByRoot: map[string]workspaceMemberReport{}}
	graph.Nodes = describeWorkspaceMembers(workspace)
	var issues []workspaceMemberReport
	byID := map[string]workspaceMemberReport{}
	for _, node := range graph.Nodes {
		if node.Status != "ok" {
			issues = append(issues, node)
			continue
		}
		rootKey := filepath.Clean(node.ResolvedPath)
		graph.ByRoot[rootKey] = node
		if prev, ok := byID[node.CapsuleID]; ok {
			issues = append(issues, workspaceMemberReport{
				Path:   node.Path,
				Status: "fail",
				Detail: fmt.Sprintf("duplicate capsule id %s (also %s)", node.CapsuleID, prev.Path),
			})
		} else {
			byID[node.CapsuleID] = node
		}
	}
	for _, node := range graph.Nodes {
		if node.Status != "ok" {
			continue
		}
		manifest, err := parseCapsule(node.CapsulePath)
		if err != nil {
			issues = append(
				issues,
				workspaceMemberReport{Path: node.Path, Status: "invalid", Detail: err.Error()},
			)
			continue
		}
		for _, dep := range manifest.Dependencies {
			if dep.Path == "" {
				continue
			}
			depRoot, err := resolveDependencyProjectRoot(node.ResolvedPath, dep.Path)
			if err != nil {
				issues = append(
					issues,
					workspaceMemberReport{
						Path:   node.Path,
						Status: "fail",
						Detail: fmt.Sprintf("%s: %v", dep.ID, err),
					},
				)
				continue
			}
			depNode, ok := graph.ByRoot[filepath.Clean(depRoot)]
			if !ok {
				issues = append(
					issues,
					workspaceMemberReport{
						Path:   node.Path,
						Status: "fail",
						Detail: fmt.Sprintf(
							"dependency %s path %s is not a workspace member",
							dep.ID,
							dep.Path,
						),
					},
				)
				continue
			}
			graph.Edges = append(
				graph.Edges,
				workspaceGraphEdge{From: node.Path, To: depNode.Path, ID: dep.ID},
			)
		}
		_, depManifests, err := projectDependencyGraph(
			node.ResolvedPath,
			manifest,
			map[string]int{node.ResolvedPath: projectDependencyVisiting},
			[]string{node.ResolvedPath},
		)
		if err != nil {
			issues = append(
				issues,
				workspaceMemberReport{Path: node.Path, Status: "fail", Detail: err.Error()},
			)
			continue
		}
		manifests := append([]capsuleManifest{manifest}, depManifests...)
		if err := validateCapsuleGraph(manifests, ""); err != nil {
			issues = append(
				issues,
				workspaceMemberReport{Path: node.Path, Status: "fail", Detail: err.Error()},
			)
		}
	}
	sort.Slice(graph.Edges, func(i, j int) bool {
		if graph.Edges[i].From == graph.Edges[j].From {
			return graph.Edges[i].To < graph.Edges[j].To
		}
		return graph.Edges[i].From < graph.Edges[j].From
	})
	graph.Issues = dedupeWorkspaceIssues(issues)
	return graph, nil
}

func describeWorkspaceMembers(workspace workspaceManifest) []workspaceMemberReport {
	var out []workspaceMemberReport
	for _, member := range workspace.Members {
		out = append(out, describeWorkspaceMember(workspace.Root, member))
	}
	return out
}

func describeWorkspaceMember(root string, member string) workspaceMemberReport {
	item := workspaceMemberReport{
		Path:         member,
		ResolvedPath: filepath.Join(root, filepath.FromSlash(member)),
		Status:       "ok",
	}
	info, err := os.Stat(item.ResolvedPath)
	if err != nil {
		item.Status = "missing"
		item.Detail = err.Error()
		return item
	}
	if !info.IsDir() {
		item.Status = "invalid"
		item.Detail = "member path is not a directory"
		return item
	}
	capsulePath, err := findCapsulePath(item.ResolvedPath)
	if err != nil {
		item.Status = "invalid"
		item.Detail = err.Error()
		return item
	}
	item.CapsulePath = capsulePath
	manifest, err := parseCapsule(capsulePath)
	if err != nil {
		item.Status = "invalid"
		item.Detail = err.Error()
		return item
	}
	item.CapsuleID = manifest.ID
	item.Version = manifest.Version
	return item
}

func workspaceSyncOrder(graph workspaceGraph) ([]workspaceMemberReport, error) {
	byPath := map[string]workspaceMemberReport{}
	deps := map[string][]string{}
	for _, node := range graph.Nodes {
		if node.Status == "ok" {
			byPath[node.Path] = node
		}
	}
	for _, edge := range graph.Edges {
		deps[edge.From] = append(deps[edge.From], edge.To)
	}
	for path := range deps {
		sort.Strings(deps[path])
	}
	state := map[string]int{}
	var order []workspaceMemberReport
	var visit func(string) error
	visit = func(path string) error {
		switch state[path] {
		case projectDependencyVisiting:
			return fmt.Errorf("workspace dependency cycle at %s", path)
		case projectDependencyDone:
			return nil
		}
		node, ok := byPath[path]
		if !ok {
			return fmt.Errorf("workspace member %s not found", path)
		}
		state[path] = projectDependencyVisiting
		for _, dep := range deps[path] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[path] = projectDependencyDone
		order = append(order, node)
		return nil
	}
	for _, member := range graph.Workspace.Members {
		if err := visit(member); err != nil {
			return nil, err
		}
	}
	return order, nil
}

func workspaceProjectContext(member workspaceMemberReport) *cliProjectContext {
	manifest, _ := parseCapsule(member.CapsulePath)
	return &cliProjectContext{
		Found:       true,
		Root:        member.ResolvedPath,
		CapsulePath: member.CapsulePath,
		LockPath:    findProjectLock(member.ResolvedPath),
		Manifest:    manifest,
	}
}

func syncWorkspaceProject(
	ctx *cliProjectContext,
	targetFlag string,
	allTargets bool,
	jobs int,
) error {
	lockPath := filepath.Join(ctx.Root, compiler.SemanticLockFileName)
	useArtifactBuilder, err := projectSyncUsesArtifactBuilder(ctx.Manifest, targetFlag, allTargets)
	if err != nil {
		return err
	}
	if useArtifactBuilder {
		return buildCapsuleArtifacts(ctx.CapsulePath, capsuleArtifactBuildOptions{
			Target:     targetFlag,
			LockPath:   lockPath,
			Jobs:       jobs,
			AllTargets: allTargets,
		})
	}
	manifests, err := parseCapsuleGraphArgs([]string{ctx.CapsulePath})
	if err != nil {
		return err
	}
	if err := validateCapsuleGraph(manifests, targetFlag); err != nil {
		return err
	}
	return writeEcoLock(lockPath, manifests)
}

func writeWorkspaceListText(w io.Writer, report workspaceReport) {
	if len(report.Members) == 0 {
		fmt.Fprintln(w, "Workspace members: none")
		return
	}
	fmt.Fprintln(w, "Workspace members:")
	for _, member := range report.Members {
		fmt.Fprintf(w, "  %s %s", member.Path, member.Status)
		if member.CapsuleID != "" {
			fmt.Fprintf(w, " %s %s", member.CapsuleID, member.Version)
		}
		if member.Detail != "" {
			fmt.Fprintf(w, ": %s", member.Detail)
		}
		fmt.Fprintln(w)
	}
}

func writeWorkspaceGraphText(w io.Writer, report workspaceGraphReport) {
	fmt.Fprintf(
		w,
		"Workspace graph: %d node(s), %d edge(s)\n",
		len(report.Nodes),
		len(report.Edges),
	)
	for _, edge := range report.Edges {
		fmt.Fprintf(w, "  %s -> %s (%s)\n", edge.From, edge.To, edge.ID)
	}
}

func writeWorkspaceIssues(w io.Writer, issues []workspaceMemberReport) {
	for _, issue := range issues {
		if issue.Detail != "" {
			fmt.Fprintf(w, "%s: %s\n", issue.Path, issue.Detail)
		} else {
			fmt.Fprintf(w, "%s: %s\n", issue.Path, issue.Status)
		}
	}
}

func encodeWorkspaceJSON(stdout io.Writer, stderr io.Writer, value any, fail bool) int {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if fail {
		return 1
	}
	return 0
}

func dedupeWorkspaceIssues(issues []workspaceMemberReport) []workspaceMemberReport {
	seen := map[string]struct{}{}
	var out []workspaceMemberReport
	for _, issue := range issues {
		key := issue.Path + "\x00" + issue.Status + "\x00" + issue.Detail
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, issue)
	}
	return out
}

// ---- workspace_types.go ----

const (
	workspaceFileName = "Tetra.workspace"
	workspaceSchemaV1 = "tetra.workspace.v1"
)

type workspaceManifest struct {
	Path    string
	Root    string
	Schema  string
	Members []string
}

type workspaceReport struct {
	Status        string                  `json:"status,omitempty"`
	Root          string                  `json:"root"`
	WorkspacePath string                  `json:"workspace_path"`
	Members       []workspaceMemberReport `json:"members"`
}

type workspaceMemberReport struct {
	Path         string `json:"path"`
	ResolvedPath string `json:"resolved_path,omitempty"`
	CapsulePath  string `json:"capsule_path,omitempty"`
	CapsuleID    string `json:"capsule_id,omitempty"`
	Version      string `json:"version,omitempty"`
	Status       string `json:"status"`
	Detail       string `json:"detail,omitempty"`
}

type workspaceGraphReport struct {
	Status        string                  `json:"status,omitempty"`
	Root          string                  `json:"root"`
	WorkspacePath string                  `json:"workspace_path"`
	Nodes         []workspaceMemberReport `json:"nodes"`
	Edges         []workspaceGraphEdge    `json:"edges"`
}

type workspaceGraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	ID   string `json:"id"`
}

type workspaceExecutionReport struct {
	WorkspaceRoot string                           `json:"workspace_root"`
	Command       string                           `json:"command"`
	Target        string                           `json:"target,omitempty"`
	Total         int                              `json:"total"`
	Passed        int                              `json:"passed"`
	Failed        int                              `json:"failed"`
	Skipped       int                              `json:"skipped"`
	Members       []workspaceExecutionMemberReport `json:"members"`
}

type workspaceExecutionMemberReport struct {
	Path      string `json:"path"`
	CapsuleID string `json:"capsule_id,omitempty"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
	ExitCode  *int   `json:"exit_code,omitempty"`
}

type workspaceGraph struct {
	Workspace workspaceManifest
	Nodes     []workspaceMemberReport
	Edges     []workspaceGraphEdge
	Issues    []workspaceMemberReport
	ByRoot    map[string]workspaceMemberReport
}
