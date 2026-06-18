package compiler

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/backend/linux_x64"
	"tetra_language/compiler/internal/backend/linux_x86"
	"tetra_language/compiler/internal/backend/macos_x64"
	"tetra_language/compiler/internal/backend/windows_x64"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/buildnative"
	"tetra_language/compiler/internal/buildplan"
	"tetra_language/compiler/internal/cache"
	"tetra_language/compiler/internal/deps"
	"tetra_language/compiler/internal/format/elf"
	"tetra_language/compiler/internal/format/macho"
	"tetra_language/compiler/internal/format/pe"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/formats"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/linker"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/internal/t4iface"
	"tetra_language/compiler/internal/validation"
	"tetra_language/compiler/internal/version"
	ctarget "tetra_language/compiler/target"
)

// ---- api.go ----

type Program = frontend.Program
type FileAST = frontend.FileAST
type CheckedProgram = semantics.CheckedProgram
type IRProgram = ir.IRProgram
type IRFunc = ir.IRFunc
type PLIRProgram = plir.Program
type UILoweredBundle = lower.UILoweredBundle
type UIToolkitBundle = lower.UIToolkitBundle

type Object = tobj.Object
type Symbol = tobj.Symbol
type Reloc = tobj.Reloc
type RelocKind = tobj.RelocKind

type World = module.World
type WorldOptions = module.LoadOptions
type ModuleRoot = module.ModuleRoot
type CheckOptions = semantics.CheckOptions

const (
	RelocCallRel32      = tobj.RelocCallRel32
	RelocIATDisp32      = tobj.RelocIATDisp32
	RelocDataDisp32     = tobj.RelocDataDisp32
	RelocFuncAddrDisp32 = tobj.RelocFuncAddrDisp32
	RelocDataAbs32      = tobj.RelocDataAbs32
	RelocFuncAddrAbs32  = tobj.RelocFuncAddrAbs32
)

func Parse(src []byte) (*Program, error) {
	return frontend.Parse(src)
}

func ParseFile(src []byte, filename string) (*FileAST, error) {
	return frontend.ParseFile(src, filename)
}

func NormalizeFlowForMigration(src []byte, filename string) ([]byte, error) {
	return frontend.NormalizeFlowForMigration(src, filename)
}

func LoadWorld(entryPath string) (*World, error) {
	return module.LoadWorld(entryPath)
}

func LoadWorldOpt(entryPath string, opt WorldOptions) (*World, error) {
	return module.LoadWorldOpt(entryPath, opt)
}

// Check validates a single already-parsed source program.
// Boundary: this API does not resolve filesystem modules/import graphs; use
// LoadWorld + CheckWorld for cross-module checking.
func Check(prog *Program) (*CheckedProgram, error) {
	return semantics.Check(prog)
}

// CheckWorld validates a module graph loaded from filesystem sources.
func CheckWorld(world *World) (*CheckedProgram, error) {
	return semantics.CheckWorld(world)
}

func CheckWorldOpt(world *World, opt CheckOptions) (*CheckedProgram, error) {
	return semantics.CheckWorldOpt(world, opt)
}

func Lower(checked *CheckedProgram) (*IRProgram, error) {
	return lower.Lower(checked)
}

func BuildPLIR(checked *CheckedProgram) (*PLIRProgram, error) {
	prog, err := plir.FromCheckedProgram(checked)
	if err != nil {
		return nil, err
	}
	if err := plir.VerifyProgram(prog); err != nil {
		return nil, err
	}
	return prog, nil
}

func FormatPLIR(prog *PLIRProgram) string {
	return plir.FormatText(prog)
}

func LowerModule(checked *CheckedProgram, module string) ([]IRFunc, error) {
	return lower.LowerModule(checked, module)
}

func LowerModules(checked *CheckedProgram) (map[string][]IRFunc, error) {
	return lower.LowerModules(checked)
}

func LowerUI(checked *CheckedProgram) (*UILoweredBundle, error) {
	return lower.LowerUI(checked)
}

func LowerUIToolkit(bundle *UILoweredBundle) (*UIToolkitBundle, error) {
	return lower.LowerUIToolkit(bundle)
}

func VerifyIRProgram(prog *IRProgram) error {
	return lower.VerifyProgram(prog)
}

func VerifyIRFunc(fn IRFunc) error {
	return lower.VerifyFunc(fn)
}

func CodegenObjectLinuxX64(funcs []IRFunc) (*Object, error) {
	if err := verifyIRFuncs(funcs); err != nil {
		return nil, err
	}
	return linux_x64.CodegenObjectLinuxX64(funcs)
}

func CodegenObjectLinuxX86(funcs []IRFunc) (*Object, error) {
	if err := verifyIRFuncs(funcs); err != nil {
		return nil, err
	}
	return linux_x86.CodegenObjectLinuxX86(funcs)
}

func CodegenObjectWindowsX64(funcs []IRFunc) (*Object, error) {
	if err := verifyIRFuncs(funcs); err != nil {
		return nil, err
	}
	return windows_x64.CodegenObjectWindowsX64(funcs)
}

func CodegenObjectMacOSX64(funcs []IRFunc) (*Object, error) {
	if err := verifyIRFuncs(funcs); err != nil {
		return nil, err
	}
	return macos_x64.CodegenObjectMacOSX64(funcs)
}

func verifyIRFuncs(funcs []IRFunc) error {
	for _, fn := range funcs {
		if err := lower.VerifyFunc(fn); err != nil {
			return err
		}
	}
	return nil
}

func LinkLinuxX64(objects []*Object, mainName string) (*elf.Image, error) {
	return linker.LinkLinuxX64(objects, mainName)
}

func LinkLinuxX32(objects []*Object, mainName string) (*elf.Image, error) {
	return linker.LinkLinuxX32(objects, mainName)
}

func LinkLinuxX86(objects []*Object, mainName string) (*elf.Image, error) {
	return linker.LinkLinuxX86(objects, mainName)
}

func LinkWindowsX64(objects []*Object, mainName string) (*pe.PEImage, error) {
	return linker.LinkWindowsX64(objects, mainName)
}

func LinkMacOSX64(objects []*Object, mainName string) (*macho.MachOImage, error) {
	return linker.LinkMacOSX64(objects, mainName)
}

func WriteELF64LinuxX64(path string, img *elf.Image) error {
	return elf.WriteELF64LinuxX64(path, img)
}

func WriteELF32LinuxX32(path string, img *elf.Image) error {
	return elf.WriteELF32LinuxX32(path, img)
}

func WriteELF32LinuxX86(path string, img *elf.Image) error {
	return elf.WriteELF32LinuxX86(path, img)
}

func WritePE64WindowsX64(path string, img *pe.PEImage) error {
	return pe.WritePE64WindowsX64(path, img)
}

func WriteMachO64MacOSX64(path string, img *macho.MachOImage) error {
	return macho.WriteMachO64MacOSX64(path, img)
}

func ReadObject(path string) (*Object, error) {
	return tobj.ReadObject(path)
}

func WriteObject(path string, obj *Object) error {
	return tobj.WriteObject(path, obj)
}

// ---- compiler.go ----

func BuildFile(inputPath, outputPath, target string) error {
	_, err := BuildFileWithStats(inputPath, outputPath, target)
	return err
}

type linkedObject struct {
	path        string
	obj         *Object
	contentHash [32]byte
}

type nativeCodegenFunc = buildnative.CodegenFunc

type nativeExecutableBackend struct {
	name         string
	os           ctarget.OS
	format       ctarget.Format
	codegen      func(x64.CodegenOptions) nativeCodegenFunc
	link         func(outputPath string, objects []*Object, mainName string) error
	actorRuntime func(actorEntries []string) (*Object, error)
}

type nativeBuildTarget struct {
	target  ctarget.Target
	triple  string
	backend nativeExecutableBackend
	codegen nativeCodegenFunc
}

type checkedBuildWorld struct {
	world   *World
	checked *semantics.CheckedProgram
}

type moduleBuildJob = buildplan.ModuleBuildJob

type moduleBuildPlan = buildplan.ModuleBuildPlan

func BuildFileWithStats(inputPath, outputPath, target string) (*BuildStats, error) {
	return BuildFileWithStatsOpt(inputPath, outputPath, target, BuildOptions{Jobs: 1})
}

// BuildFileWithStatsOpt compiles from an entry file path and always uses the
// module loader graph rooted at that entry path.
// Boundary: for in-memory single-source semantic checks use Parse + Check.
func BuildFileWithStatsOpt(
	inputPath, outputPath, target string,
	opt BuildOptions,
) (*BuildStats, error) {
	native, handled, stats, err := resolveExecutableBuildTarget(inputPath, outputPath, target, opt)
	if handled || err != nil {
		return stats, err
	}

	build, err := loadCheckedBuildWorld(inputPath, opt, !opt.InterfaceOnly, native.triple)
	if err != nil {
		return nil, translateTargetExportedFFISemanticError(err, native.triple)
	}
	if opt.InterfaceOnly {
		return interfaceOnlyBuildStats(build.world), nil
	}
	if err := validateTargetExportedFFIABI(build.checked, native.triple); err != nil {
		return nil, err
	}
	if err := validateNativeRuntimeBeforeCodegen(build.checked, native.triple); err != nil {
		return nil, err
	}
	if opt.EmitRuntimeHeapTelemetry {
		if strings.TrimSpace(opt.RuntimeHeapTelemetryProgram) == "" {
			opt.RuntimeHeapTelemetryProgram = filepath.Base(outputPath)
		}
		opt.RuntimeHeapTelemetryMain = build.checked.MainName
		actorDomains, err := runtimeHeapTelemetryActorDomainsForBuild(native, opt, build.checked)
		if err != nil {
			return nil, err
		}
		opt.RuntimeHeapTelemetryActorDomains = actorDomains
	}
	linkedObjects, err := prepareLinkedObjects(
		build.world,
		build.checked,
		opt.LinkObjectPaths,
		native.triple,
	)
	if err != nil {
		return nil, err
	}

	plan, stats, err := planNativeModuleBuild(
		build.world,
		build.checked,
		native.triple,
		opt,
		linkedObjects,
	)
	if err != nil {
		return nil, err
	}
	if err := compileNativeModulePlan(
		build.world,
		build.checked,
		native,
		opt,
		plan,
		stats,
	); err != nil {
		return nil, err
	}

	objects, err := objectsFromModulePlan(plan)
	if err != nil {
		return nil, err
	}
	if err := linkNativeExecutable(
		outputPath,
		native,
		opt,
		build.checked,
		objects,
		linkedObjects,
	); err != nil {
		return nil, err
	}
	if err := emitUIArtifacts(outputPath, native.triple, build.checked); err != nil {
		return nil, err
	}
	if err := emitExplainReports(outputPath, native.triple, build.checked, opt); err != nil {
		return nil, err
	}

	return stats, nil
}

func resolveExecutableBuildTarget(
	inputPath, outputPath, target string,
	opt BuildOptions,
) (nativeBuildTarget, bool, *BuildStats, error) {
	tgt, err := ctarget.Parse(target)
	if err != nil {
		return nativeBuildTarget{}, false, nil, err
	}
	if err := validateRuntimeHeapTelemetryBuildOptions(tgt, opt); err != nil {
		return nativeBuildTarget{}, false, nil, err
	}
	if opt.DebugInfo && !tgt.SupportsDebugInfo {
		return nativeBuildTarget{}, false, nil, fmt.Errorf(
			"target does not support debug info: %s",
			tgt.Triple,
		)
	}
	if opt.ReleaseOptimize && !tgt.SupportsReleaseOptimize {
		return nativeBuildTarget{}, false, nil, fmt.Errorf(
			"target does not support release optimization: %s",
			tgt.Triple,
		)
	}
	if tgt.Triple == "wasm32-wasi" {
		stats, err := buildWASM32WASIWithStatsOpt(inputPath, outputPath, tgt, opt)
		return nativeBuildTarget{}, true, stats, err
	}
	if tgt.Triple == "wasm32-web" {
		stats, err := buildWASM32WEBWithStatsOpt(inputPath, outputPath, tgt, opt)
		return nativeBuildTarget{}, true, stats, err
	}
	switch opt.Emit {
	case EmitExe:
		if ctarget.IsBuildOnlyTarget(tgt.Triple) && tgt.Triple != "linux-x32" &&
			tgt.Triple != "linux-x86" {
			reason := tgt.UnsupportedReason
			if reason == "" {
				reason = "executable support is not implemented yet"
			}
			return nativeBuildTarget{}, false, nil, fmt.Errorf(
				"target backend not implemented: %s (%s)",
				tgt.Triple,
				reason,
			)
		}
	case EmitObject, EmitLibrary:
		stats, err := buildObjectFileWithStatsOpt(inputPath, outputPath, tgt, opt)
		return nativeBuildTarget{}, true, stats, err
	default:
		return nativeBuildTarget{}, false, nil, fmt.Errorf("unsupported emit mode: %d", opt.Emit)
	}
	codegen, err := nativeCodegenForTarget(tgt, opt)
	if err != nil {
		return nativeBuildTarget{}, false, nil, err
	}
	backend, ok := nativeExecutableBackendForTarget(tgt)
	if !ok {
		return nativeBuildTarget{}, false, nil, fmt.Errorf("unsupported target: %s", tgt.Triple)
	}
	return nativeBuildTarget{
		target:  tgt,
		triple:  tgt.Triple,
		backend: backend,
		codegen: codegen,
	}, false, nil, nil
}

func nativeCodegenForTarget(tgt ctarget.Target, opt BuildOptions) (nativeCodegenFunc, error) {
	return buildnative.CodegenForTarget(tgt, opt)
}

func nativeExecutableBackendForTarget(tgt ctarget.Target) (nativeExecutableBackend, bool) {
	backend, ok := buildnative.ExecutableBackendForTarget(tgt)
	if !ok {
		return nativeExecutableBackend{}, false
	}
	return rootNativeExecutableBackend(backend), true
}

func nativeCodegenOptions(opt BuildOptions) x64.CodegenOptions {
	return buildnative.CodegenOptions(opt)
}

func nativeCodegenOptionsForTarget(tgt ctarget.Target, opt BuildOptions) x64.CodegenOptions {
	return buildnative.CodegenOptionsForTarget(tgt, opt)
}

func validateRuntimeHeapTelemetryBuildOptions(tgt ctarget.Target, opt BuildOptions) error {
	return buildnative.ValidateRuntimeHeapTelemetryBuildOptions(tgt, opt)
}

func runtimeHeapTelemetryActorDomainsForBuild(
	native nativeBuildTarget,
	opt BuildOptions,
	checked *semantics.CheckedProgram,
) (bool, error) {
	if !opt.EmitRuntimeHeapTelemetry || native.triple != "linux-x64" ||
		opt.RuntimeObjectPath != "" {
		return false, nil
	}
	actorsUsed, _, actorSpawnCount, err := collectActorEntries(checked)
	if err != nil {
		return false, err
	}
	actorStateUsed := collectActorStateRuntimeUsage(checked)
	actorRuntimeUsed, _ := collectActorRuntimeUsagePosition(checked)
	if !actorsUsed && !actorRuntimeUsed && !actorStateUsed {
		return false, nil
	}
	typedTasksUsed, typedTaskMaxSlots := collectTypedTaskRuntimeUsage(checked)
	netRuntimeUsage := collectNetRuntimeUsageProfile(checked)
	distributedActorsUsed, _ := collectDistributedActorRuntimeUsagePosition(checked)
	mode, err := selectRuntimeModeForNativeTarget(native.triple, opt.Runtime, runtimeUsageProfile{
		actorStateUsed:        actorStateUsed,
		tasksUsed:             collectTaskRuntimeUsage(checked),
		taskGroupsUsed:        collectTaskGroupRuntimeUsage(checked),
		typedTasksUsed:        typedTasksUsed,
		typedTaskMaxSlots:     typedTaskMaxSlots,
		timeRuntimeUsed:       collectTimeRuntimeUsage(checked),
		filesystemUsed:        collectFilesystemRuntimeUsage(checked),
		netUsed:               netRuntimeUsage.used,
		netRuntimeSymbols:     netRuntimeUsage.requiredSymbols(),
		surfaceUsed:           collectSurfaceRuntimeUsage(checked),
		distributedActorsUsed: distributedActorsUsed,
		actorSpawnCount:       actorSpawnCount,
	})
	if err != nil {
		return false, err
	}
	return mode == RuntimeBuiltin, nil
}

func nativeExecutableBackends() map[ctarget.OS]nativeExecutableBackend {
	backends := buildnative.ExecutableBackends()
	out := make(map[ctarget.OS]nativeExecutableBackend, len(backends))
	for os, backend := range backends {
		out[os] = rootNativeExecutableBackend(backend)
	}
	return out
}

func nativeLinuxX32ExecutableBackend() nativeExecutableBackend {
	return rootNativeExecutableBackend(buildnative.LinuxX32ExecutableBackend())
}

func nativeLinuxX86ExecutableBackend() nativeExecutableBackend {
	return rootNativeExecutableBackend(buildnative.LinuxX86ExecutableBackend())
}

func rootNativeExecutableBackend(backend buildnative.ExecutableBackend) nativeExecutableBackend {
	return nativeExecutableBackend{
		name:         backend.Name,
		os:           backend.OS,
		format:       backend.Format,
		codegen:      backend.Codegen,
		link:         backend.Link,
		actorRuntime: backend.ActorRuntime,
	}
}

func buildNativeExecutableBackend(backend nativeExecutableBackend) buildnative.ExecutableBackend {
	return buildnative.ExecutableBackend{
		Name:         backend.name,
		OS:           backend.os,
		Format:       backend.format,
		Codegen:      backend.codegen,
		Link:         backend.link,
		ActorRuntime: backend.actorRuntime,
	}
}

func loadCheckedBuildWorld(
	inputPath string,
	opt BuildOptions,
	requireMain bool,
	target string,
) (checkedBuildWorld, error) {
	world, err := loadWorldForBuild(inputPath, opt)
	if err != nil {
		return checkedBuildWorld{}, err
	}
	if err := validateTargetExportedFFIAST(world, target); err != nil {
		return checkedBuildWorld{}, err
	}
	checked, err := semantics.CheckWorldOpt(
		world,
		semanticsCheckOptionsForTarget(requireMain, target),
	)
	if err != nil {
		return checkedBuildWorld{}, err
	}
	return checkedBuildWorld{world: world, checked: checked}, nil
}

func semanticsCheckOptionsForTarget(requireMain bool, target string) semantics.CheckOptions {
	return semantics.CheckOptions{
		RequireMain:              requireMain,
		EnableILP32NativeScalars: target == "linux-x86" || target == "linux-x32",
	}
}

func prepareLinkedObjects(
	world *World,
	checked *semantics.CheckedProgram,
	paths []string,
	target string,
) ([]linkedObject, error) {
	linkedObjects, err := readLinkObjects(paths, target)
	if err != nil {
		return nil, err
	}
	if err := validateInterfaceImplementationProviders(world, checked, linkedObjects); err != nil {
		return nil, err
	}
	return linkedObjects, nil
}

func planNativeModuleBuild(
	world *World,
	checked *semantics.CheckedProgram,
	target string,
	opt BuildOptions,
	linkedObjects []linkedObject,
) (moduleBuildPlan, *BuildStats, error) {
	sigMap := cache.BuildSigMap(checked)
	depsByModule := deps.CollectExternalCalleesByModule(checked)
	typeDepsByModule := deps.CollectExternalTypesByModule(checked)
	typeSigMap, err := cache.BuildTypeSigMap(checked.Types)
	if err != nil {
		return moduleBuildPlan{}, nil, err
	}
	stats := &BuildStats{InterfaceModules: sortedInterfaceModules(world)}

	modules := buildplan.SourceModules(world)
	publicAPIHashes := make(map[string]string, len(modules))
	for _, module := range modules {
		file := world.ByModule[module]
		if file == nil {
			return moduleBuildPlan{}, nil, fmt.Errorf("missing module '%s'", module)
		}
		hash, err := InterfaceFingerprintFromSource(file.Src, file.Path)
		if err != nil {
			return moduleBuildPlan{}, nil, err
		}
		publicAPIHashes[module] = hash
	}

	buildTag := buildTagFromOptions(opt, linkedObjects)
	if targetSupportsStackAllocationLowering(target) {
		buildTag = buildplan.WithStackAllocationBuildTag(buildTag)
	}
	objectsByModule := make(map[string]*Object, len(modules))
	var toCompile []moduleBuildJob

	for _, module := range modules {
		file := world.ByModule[module]
		if file == nil {
			return moduleBuildPlan{}, nil, fmt.Errorf("missing module '%s'", module)
		}
		srcHash := sha256.Sum256(file.Src)
		depSet := depsByModule[module]
		var callees []string
		for name := range depSet {
			callees = append(callees, name)
		}
		typeSet := typeDepsByModule[module]
		var typeDeps []string
		for name := range typeSet {
			typeDeps = append(typeDeps, name)
		}
		callees = append(callees, moduleLocalFunctionSigDeps(module, sigMap)...)
		typeDeps = append(typeDeps, moduleLocalTypeSigDeps(module, typeSigMap)...)
		depHash, err := cache.DepSigHashFromDepsWithInterfaceHashes(
			callees,
			typeDeps,
			sigMap,
			typeSigMap,
			world.InterfaceHashes,
		)
		if err != nil {
			return moduleBuildPlan{}, nil, err
		}
		if !opt.EmitRuntimeHeapTelemetry {
			obj, hit, err := cache.LoadCachedObject(
				world.Root,
				target,
				buildTag,
				module,
				srcHash,
				depHash,
			)
			if err != nil {
				return moduleBuildPlan{}, nil, err
			}
			if hit {
				stats.CacheHits = append(stats.CacheHits, module)
				objectsByModule[module] = obj
				continue
			}
		}
		toCompile = append(
			toCompile,
			moduleBuildJob{Module: module, SrcHash: srcHash, DepHash: depHash},
		)
	}

	return moduleBuildPlan{
		Modules:           modules,
		PublicAPIHashes:   publicAPIHashes,
		BuildTag:          buildTag,
		ObjectsByModule:   objectsByModule,
		ObjectlessModules: make(map[string]bool),
		ToCompile:         toCompile,
	}, stats, nil
}

func moduleLocalFunctionSigDeps(module string, sigMap map[string]semantics.FuncSig) []string {
	return buildplan.ModuleLocalFunctionSigDeps(module, sigMap)
}

func moduleLocalTypeSigDeps(module string, typeSigMap map[string]string) []string {
	return buildplan.ModuleLocalTypeSigDeps(module, typeSigMap)
}

func compileNativeModulePlan(
	world *World,
	checked *semantics.CheckedProgram,
	native nativeBuildTarget,
	opt BuildOptions,
	plan moduleBuildPlan,
	stats *BuildStats,
) error {
	if len(plan.ToCompile) == 0 {
		sortBuildStats(stats)
		return nil
	}
	var allocationPlan *allocplan.Plan
	var allocationSummaryProgram *ir.IRProgram
	if targetSupportsStackAllocationLowering(native.triple) {
		plirProg, err := plir.FromCheckedProgram(checked)
		if err != nil {
			return err
		}
		if err := plir.VerifyProgram(plirProg); err != nil {
			return err
		}
		allocationPlan, err = allocplan.FromPLIRWithOptions(
			plirProg,
			allocationPlanOptionsForTarget(native.triple),
		)
		if err != nil {
			return err
		}
		allocationSummaryProgram, err = lower.LowerWithOptions(
			checked,
			lowerOptionsForTarget(native.triple),
		)
		if err != nil {
			return err
		}
	}
	jobs := buildplan.EffectiveWorkerCount(opt.Jobs, len(plan.ToCompile), runtime.NumCPU())

	jobsCh := make(chan moduleBuildJob)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errMu sync.Mutex
	var firstErr error

	setErr := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		errMu.Unlock()
	}

	getErr := func() error {
		errMu.Lock()
		defer errMu.Unlock()
		return firstErr
	}

	codegen := native.backend.codegen(nativeCodegenOptionsForTarget(native.target, opt))

	worker := func() {
		defer wg.Done()
		for job := range jobsCh {
			if getErr() != nil {
				continue
			}
			funcs, err := lower.LowerModuleWithOptions(
				checked,
				job.Module,
				lowerOptionsForTarget(native.triple),
			)
			if err != nil {
				setErr(err)
				continue
			}
			if err := validateTargetAtomicIR(funcs, native.target); err != nil {
				setErr(err)
				continue
			}
			if allocationPlan != nil {
				if err := validation.ValidateAllocationLoweringWithSummaryProgram(
					allocationPlanForIRFuncs(allocationPlan, funcs),
					&ir.IRProgram{Funcs: funcs},
					allocationSummaryProgram,
				); err != nil {
					setErr(err)
					continue
				}
			}
			mu.Lock()
			stats.LoweredModules = append(stats.LoweredModules, job.Module)
			mu.Unlock()

			dataPrefix := checked.GlobalDataByModule[job.Module]
			if len(funcs) == 0 {
				mu.Lock()
				plan.ObjectlessModules[job.Module] = true
				mu.Unlock()
				continue
			}
			obj, err := codegen(funcs, dataPrefix)
			if err != nil {
				setErr(err)
				continue
			}
			buildplan.ApplyModuleObjectMetadata(obj, buildplan.ModuleObjectMetadata{
				Target:          native.triple,
				Module:          job.Module,
				CompilerVersion: version.CompilerVersion,
				PublicAPIHash:   plan.PublicAPIHashes[job.Module],
				SrcHash:         job.SrcHash,
				WorldSigHash:    job.DepHash,
			})
			if !opt.EmitRuntimeHeapTelemetry {
				if err := cache.StoreCachedObject(world.Root, native.triple, plan.BuildTag, obj); err != nil {
					setErr(err)
					continue
				}
			}
			mu.Lock()
			stats.CompiledModules = append(stats.CompiledModules, job.Module)
			plan.ObjectsByModule[job.Module] = obj
			mu.Unlock()
		}
	}

	wg.Add(jobs)
	for i := 0; i < jobs; i++ {
		go worker()
	}
	for _, job := range plan.ToCompile {
		jobsCh <- job
	}
	close(jobsCh)
	wg.Wait()
	if err := getErr(); err != nil {
		return err
	}
	sortBuildStats(stats)
	return nil
}

func sortBuildStats(stats *BuildStats) {
	buildplan.SortStats(stats)
}

func objectsFromModulePlan(plan moduleBuildPlan) ([]*Object, error) {
	return buildplan.ObjectsFromModulePlan(plan)
}

// ---- diagnostics.go ----

type Diagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Severity string `json:"severity"`
	Hint     string `json:"hint,omitempty"`
}

const (
	DiagnosticCodeParse            = frontend.DiagnosticCodeParse
	DiagnosticCodeSemantic         = "TETRA2001"
	DiagnosticCodeSafetyOwnership  = semantics.DiagnosticCodeSafetyOwnership
	DiagnosticCodeSafetyLifetime   = semantics.DiagnosticCodeSafetyLifetime
	DiagnosticCodeSafetyEffect     = semantics.DiagnosticCodeSafetyEffect
	DiagnosticCodeSafetyPrivacy    = semantics.DiagnosticCodeSafetyPrivacy
	DiagnosticCodeSafetyBudget     = semantics.DiagnosticCodeSafetyBudget
	DiagnosticCodeIRVerifier       = lower.DiagnosticCodeIRVerifier
	DiagnosticCodeLowerUnsupported = lower.DiagnosticCodeLowerUnsupported
	DiagnosticCodeTargetRuntime    = "TETRA3003"
	DiagnosticCodeFormatter        = "TETRA_FMT001"
	DiagnosticCodeFormatterCheck   = "TETRA_FMT002"
)

var diagnosticPosRE = regexp.MustCompile(`^(?:(.+):)?(?:line )?([0-9]+):([0-9]+): (.*)$`)

type DiagnosticCodeInfo struct {
	Severity string
	Surface  string
}

func DiagnosticCodeRegistry() map[string]DiagnosticCodeInfo {
	return map[string]DiagnosticCodeInfo{
		DiagnosticCodeParse: {
			Severity: "error",
			Surface:  "parse/frontend",
		},
		DiagnosticCodeSemantic: {
			Severity: "error",
			Surface:  "semantic/compiler",
		},
		DiagnosticCodeSafetyOwnership: {
			Severity: "error",
			Surface:  "semantic safety/ownership",
		},
		DiagnosticCodeSafetyLifetime: {
			Severity: "error",
			Surface:  "semantic safety/lifetime",
		},
		DiagnosticCodeSafetyEffect: {
			Severity: "error",
			Surface:  "semantic safety/effect",
		},
		DiagnosticCodeSafetyPrivacy: {
			Severity: "error",
			Surface:  "semantic safety/privacy",
		},
		DiagnosticCodeSafetyBudget: {
			Severity: "error",
			Surface:  "semantic safety/budget",
		},
		DiagnosticCodeIRVerifier: {
			Severity: "error",
			Surface:  "ir verifier",
		},
		DiagnosticCodeLowerUnsupported: {
			Severity: "error",
			Surface:  "lowering unsupported",
		},
		DiagnosticCodeTargetRuntime: {
			Severity: "error",
			Surface:  "target runtime support",
		},
		DiagnosticCodeFormatter: {
			Severity: "error",
			Surface:  "formatter",
		},
		DiagnosticCodeFormatterCheck: {
			Severity: "error",
			Surface:  "formatter check",
		},
	}
}

func DiagnosticFromError(err error) Diagnostic {
	if err == nil {
		return Diagnostic{}
	}
	if coded, ok := err.(interface{ DiagnosticCode() string }); ok {
		return Diagnostic{
			Code:     defaultString(coded.DiagnosticCode(), DiagnosticCodeParse),
			Message:  err.Error(),
			Severity: "error",
		}
	}
	if info, ok := frontend.DiagnosticForError(err); ok {
		return Diagnostic{
			Code:     defaultString(info.Code, "TETRA0001"),
			Message:  info.Message,
			File:     info.File,
			Line:     info.Line,
			Column:   info.Column,
			Severity: defaultString(info.Severity, "error"),
			Hint:     info.Hint,
		}
	}
	msg := err.Error()
	diag := Diagnostic{
		Code:     DiagnosticCodeParse,
		Message:  msg,
		Severity: "error",
	}
	m := diagnosticPosRE.FindStringSubmatch(msg)
	if len(m) == 5 {
		diag.Code = DiagnosticCodeSemantic
		diag.File = m[1]
		diag.Line, _ = strconv.Atoi(m[2])
		diag.Column, _ = strconv.Atoi(m[3])
		diag.Message = m[4]
	}
	return diag
}

func defaultString(got string, fallback string) string {
	if got != "" {
		return got
	}
	return fallback
}

// ---- docs.go ----

func GenerateAPIDocs(paths []string) ([]byte, error) {
	files, err := collectDocFiles(paths)
	if err != nil {
		return nil, err
	}
	var parsed []*frontend.FileAST
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		file, err := frontend.ParseFile(raw, path)
		if err != nil {
			return nil, err
		}
		file.Path = path
		parsed = append(parsed, file)
	}
	titles := apiDocTitles(parsed)
	var body bytes.Buffer
	for i, file := range parsed {
		writeFileAPIDocs(&body, file, titles[i])
	}
	var b bytes.Buffer
	writeAPIDocsHeader(&b, len(parsed), body.String())
	b.Write(body.Bytes())
	return b.Bytes(), nil
}

func GenerateAPIDocsFromSource(src []byte, filename string) ([]byte, error) {
	file, err := frontend.ParseFile(src, filename)
	if err != nil {
		return nil, err
	}
	file.Path = filename
	var body bytes.Buffer
	writeFileAPIDocs(&body, file, apiDocTitles([]*frontend.FileAST{file})[0])
	var b bytes.Buffer
	writeAPIDocsHeader(&b, 1, body.String())
	b.Write(body.Bytes())
	return b.Bytes(), nil
}

type apiDocTitle struct {
	Heading      string
	Experimental bool
}

func apiDocTitles(files []*frontend.FileAST) []apiDocTitle {
	baseTitles := make([]string, len(files))
	counts := map[string]int{}
	for i, file := range files {
		title := file.Path
		if file.Module != "" {
			title = file.Module
		}
		baseTitles[i] = title
		counts[title]++
	}
	titles := make([]apiDocTitle, len(files))
	for i, file := range files {
		base := baseTitles[i]
		heading := base
		if counts[base] > 1 {
			path := filepath.ToSlash(file.Path)
			if path == "" {
				path = base
			}
			heading = fmt.Sprintf("%s (%s)", base, path)
		}
		experimental := isExperimentalModuleTitle(base)
		if experimental {
			heading += " (experimental)"
		}
		titles[i] = apiDocTitle{
			Heading:      heading,
			Experimental: experimental,
		}
	}
	return titles
}

func writeAPIDocsHeader(b *bytes.Buffer, moduleCount int, body string) {
	entryCount, hash := apiSurfaceMetadata(body)
	b.WriteString("# Tetra API Docs\n\n")
	fmt.Fprintf(
		b,
		("<!-- tetra-api-metadata: {\"schema\":\"tetra.api.v1alpha1\"," +
			"\"api_hash\":\"sha256:%s\",\"module_count\":%d,\"entry_count\":%d} " +
			"-->\n\n"),
		hash,
		moduleCount,
		entryCount,
	)
}

func apiSurfaceMetadata(body string) (int, string) {
	var surface []string
	entryCount := 0
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### "):
			surface = append(surface, trimmed)
		case strings.HasPrefix(trimmed, "- `"):
			surface = append(surface, trimmed)
			entryCount++
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(surface, "\n")))
	return entryCount, fmt.Sprintf("%x", sum[:])
}

func collectDocFiles(paths []string) ([]string, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	seen := map[string]struct{}{}
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if isDocSourceFile(path) {
				seen[path] = struct{}{}
				files = append(files, path)
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
			if isDocSourceFile(p) {
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

func isDocSourceFile(path string) bool {
	if !IsSourceFile(path) {
		return false
	}
	base := filepath.Base(path)
	return base != CapsuleFileName && base != LegacyCapsuleFileName
}

func writeFileAPIDocs(b *bytes.Buffer, file *frontend.FileAST, title apiDocTitle) {
	fmt.Fprintf(b, "## %s\n\n", title.Heading)
	if title.Experimental {
		b.WriteString("Experimental module: compatibility is not guaranteed for v1.x.\n\n")
	}
	if len(file.Structs) > 0 {
		b.WriteString("### Structs\n\n")
		for _, st := range file.Structs {
			fmt.Fprintf(b, "- `%s`\n", st.Name)
			for _, field := range st.Fields {
				fmt.Fprintf(b, "  - `%s: %s`\n", field.Name, formatLSPTypeRef(field.Type))
			}
		}
		b.WriteByte('\n')
	}
	if len(file.States) > 0 {
		b.WriteString("### States\n\n")
		for _, st := range file.States {
			fmt.Fprintf(b, "- `state %s`\n", st.Name)
			for _, field := range st.Fields {
				kind := "val"
				if field.Mutable {
					kind = "var"
				} else if field.Const {
					kind = "const"
				}
				fmt.Fprintf(b, "  - `%s %s: %s`\n", kind, field.Name, formatLSPTypeRef(field.Type))
			}
		}
		b.WriteByte('\n')
	}
	if len(file.Views) > 0 {
		b.WriteString("### Views\n\n")
		for _, view := range file.Views {
			fmt.Fprintf(b, "- `view %s(state: %s)`\n", view.Name, formatLSPTypeRef(view.StateName))
			for _, binding := range view.Bindings {
				fmt.Fprintf(b, "  - `bind %s: %s`\n", binding.Name, formatLSPTypeRef(binding.Type))
			}
			for _, event := range view.Events {
				fmt.Fprintf(b, "  - `event %s -> %s`\n", event.Name, event.Command)
			}
			for _, command := range view.Commands {
				fmt.Fprintf(b, "  - `command %s`\n", command.Name)
			}
			for _, style := range view.Styles {
				fmt.Fprintf(b, "  - `style %s: %s`\n", style.Name, formatLSPTypeRef(style.Type))
			}
			for _, entry := range view.Accessibility {
				fmt.Fprintf(
					b,
					"  - `accessibility %s: %s`\n",
					entry.Name,
					formatLSPTypeRef(entry.Type),
				)
			}
		}
		b.WriteByte('\n')
	}
	if len(file.Enums) > 0 {
		b.WriteString("### Enums\n\n")
		for _, en := range file.Enums {
			fmt.Fprintf(b, "- `%s`: ", en.Name)
			cases := make([]string, 0, len(en.Cases))
			for _, c := range en.Cases {
				cases = append(cases, c.Name)
			}
			b.WriteString(strings.Join(cases, ", "))
			b.WriteString("\n")
		}
		b.WriteByte('\n')
	}
	if len(file.Protocols) > 0 {
		b.WriteString("### Protocols\n\n")
		for _, proto := range file.Protocols {
			fmt.Fprintf(b, "- `protocol %s`\n", proto.Name)
			for _, req := range proto.Requirements {
				fmt.Fprintf(b, "  - `%s`\n", formatLSPFuncSigDecl(req))
			}
		}
		b.WriteByte('\n')
	}
	if len(file.Globals) > 0 {
		b.WriteString("### Globals\n\n")
		for _, glob := range file.Globals {
			fmt.Fprintf(b, "- `%s`\n", formatLSPGlobalDetail(glob))
		}
		b.WriteByte('\n')
	}
	if len(file.Impls) > 0 {
		b.WriteString("### Implementations\n\n")
		for _, impl := range file.Impls {
			fmt.Fprintf(b, "- `%s`\n", formatLSPImplDetail(impl))
		}
		b.WriteByte('\n')
	}
	if len(file.Funcs) > 0 {
		b.WriteString("### Functions\n\n")
		for _, fn := range file.Funcs {
			if fn.ExtensionOf != "" || fn.Synthetic {
				continue
			}
			fmt.Fprintf(b, "- `%s`\n", formatLSPFuncDetail(fn))
		}
		b.WriteByte('\n')
	}
	if len(file.Extensions) > 0 {
		b.WriteString("### Extensions\n\n")
		for _, ext := range file.Extensions {
			fmt.Fprintf(b, "- `%s`\n", formatLSPTypeRef(ext.Target))
			for _, fn := range ext.Methods {
				fmt.Fprintf(b, "  - `%s`\n", formatLSPFuncDetail(fn))
			}
		}
		b.WriteByte('\n')
	}
	if len(file.Tests) > 0 {
		b.WriteString("### Tests\n\n")
		for _, test := range file.Tests {
			fmt.Fprintf(b, "- `%s`\n", test.Name)
		}
		b.WriteByte('\n')
	}
	doctestCount := countTetraDoctests(file.Src)
	if doctestCount > 0 {
		b.WriteString("### Doctests\n\n")
		for i := 1; i <= doctestCount; i++ {
			fmt.Fprintf(b, "- doctest %d\n", i)
		}
		b.WriteByte('\n')
	}
}

func isExperimentalModuleTitle(title string) bool {
	return title == "lib.experimental" || strings.HasPrefix(title, "lib.experimental.")
}

func countTetraDoctests(src []byte) int {
	count := 0
	for _, line := range strings.Split(string(src), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "```tetra doctest" {
			count++
			continue
		}
		if strings.HasPrefix(trimmed, "//") {
			comment := strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
			if comment == "```tetra doctest" {
				count++
			}
		}
	}
	return count
}

// ---- features.go ----

// FeatureStatus is the release-truth lifecycle label for a public Tetra feature.
type FeatureStatus string

const (
	FeatureStatusCurrent             FeatureStatus = "current"
	FeatureStatusExperimental        FeatureStatus = "experimental"
	FeatureStatusReleaseCandidate    FeatureStatus = "release_candidate"
	FeatureStatusUnsupported         FeatureStatus = "unsupported"
	FeatureStatusLegacyCompatibility FeatureStatus = "legacy_compatibility"
	FeatureStatusPlanned             FeatureStatus = "planned"
	FeatureStatusPostV1              FeatureStatus = "post-v1"
)

// FeatureInfo is a machine-readable release-truth registry entry.
type FeatureInfo struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Status    FeatureStatus `json:"status"`
	Since     string        `json:"since,omitempty"`
	Scope     string        `json:"scope"`
	Stability string        `json:"stability"`
	Docs      []string      `json:"docs"`
}

// FeatureRegistry returns the canonical feature status registry for the current
// compiler/tooling surface. Keep this list conservative: current entries must
// reflect the supported surface, while future work stays planned or post-v1
// until promoted with release-gate evidence.
func FeatureRegistry() []FeatureInfo {
	features := []FeatureInfo{
		{
			ID:     "cli.core",
			Name:   "Core CLI workflows",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("check/build/run/fmt/test/doc/doctor/targets/features/formats" +
				"/new/interface/project/workspace/smoke/eco/clean/version/lsp" +
				" local workflows"),
			Stability: "supported in the current v0.4.0 local profile",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/policy/cli_contracts.md",
				"docs/user/start/cli_cheatsheet.md",
			},
		},
		{
			ID:     "targets.native",
			Name:   "Native target builds",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("linux-x64 build/run plus macos-x64, windows-x64, linux-x86, " +
				"and linux-x32 build-only release coverage with pointer, " +
				"rawptr, nullable_ptr, ref, c_int, c_uint, and the complete " +
				"ILP32 native/libc scalar FFI object evidence set, x86/x32 " +
				"allocator success/failure and island/free executable ABI " +
				"smoke evidence, current x86/x32 core.net runtime ABI " +
				"evidence, and explicit x86/x32 no-host-fallback, bounded " +
				"two-spawn x86/x32 self-host scheduler evidence, " +
				"function-pointer FFI diagnostics, remaining source " +
				"target-layout scalar diagnostics, Surface, distributed " +
				"actors, and actor-fanout diagnostics"),
			Stability: ("supported target metadata is validated by release checks; " +
				"linux-x64 keeps pointer plus c_int/c_uint @export object " +
				"regression smokes, linux-x86 and linux-x32 now build " +
				"canonical ptr/rawptr/nullable_ptr/ref plus c_int/c_uint " +
				"plus the complete ILP32 native/libc scalar @export object " +
				"smoke set and target-specific allocator success/failure " +
				"plus island/free executable ABI smokes, and both build-only " +
				"targets remain unpromoted until their remaining " +
				"FFI/runtime/stdlib runner gates pass"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/policy/cli_contracts.md",
			},
		},
		{
			ID:     "targets.wasm-artifact-preflight",
			Name:   "WASM artifact/import preflight",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("wasm32-wasi and wasm32-web artifact/import validation " +
				"through smoke --run=false, with runtime execution covered " +
				"by wasm.runtime-execution"),
			Stability: ("current deterministic artifact/import validation; this is " +
				"not runtime proof by itself"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/backend/wasm_backend_plan.md",
			},
		},
		{
			ID:     "language.flow",
			Name:   "Flow syntax profile",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("release-covered indentation syntax in examples, stdlib, " +
				"runtime, and self-host snippets"),
			Stability: "supported source syntax for the current release gate",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
			},
		},
		{
			ID:     "language.generics-mvp",
			Name:   "Static monomorphized generics MVP",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("generic functions with inferred value arguments are " +
				"statically monomorphized across modules; tiny generic " +
				"identity/wrapper calls may disappear through the internal " +
				"small-pure inliner after monomorphization; no runtime " +
				"generic values or dynamic dispatch"),
			Stability: ("supported static MVP; explicit type arguments, generic " +
				"structs, higher-ranked generics, full protocol-bound " +
				"generic dispatch, and broad specialization optimization " +
				"remain future/post-v1"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:     "language.layout-abi-policy",
			Name:   "Struct layout and ABI representation policy",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("default structs carry Tetra representation metadata and do " +
				"not promise C layout; repr(C) struct declarations parse and " +
				"check into ABI-locked metadata; public ABI/exported FFI " +
				"aggregate boundaries require explicit repr(C)"),
			Stability: ("current P21.0 default layout freedom v1 metadata/report " +
				"contract with .layout.json schema_version 2, policy " +
				"p21.0_default_layout_freedom_v1, decision rows " +
				"compiler_owned_default, abi_locked_repr_c, " +
				"exported_ffi_explicit_repr_c, and validator rejection for " +
				"fake layout freedoms; field_reordering, padding_removal, " +
				"hot_cold_splitting, scalar_replacement, and aos_to_soa " +
				"freedoms are explicitly unavailable for repr(C), while the " +
				"compiler-owned default layout freedom is report evidence " +
				"only and no field reordering, packing, hot/cold splitting, " +
				"scalar replacement, AoS-to-SoA transform, performance " +
				"change, runtime behavior change, or public ABI layout " +
				"without repr(C) is claimed"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/design/truthful_intent_architecture.md",
				"docs/design/explainable_one_build.md",
				"docs/audits/compiler/language/default-layout-freedom-v1.md",
			},
		},
		{
			ID:     "compiler.abi-verification",
			Name:   "ABI verification v1",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("P21.1 ABI verification v1 report schema " +
				"tetra.abi.verification.v1 with scope p21.1_abi_verification " +
				"covers linux-x64 SysV, linux-x86 i386 SysV, linux-x32 x32 " +
				"SysV, macos-x64 SysV, windows-x64 Win64, wasm32-wasi, and " +
				"wasm32-web target rows; task coverage includes " +
				"abi_test_corpus, struct_enum_slice_string_return_validation," +
				" call_boundary_validation, and ffi_repr_c_tests; native " +
				"rows reuse x86/x32/x64 classifier, aggregate, object, and " +
				"FFI repr(C) diagnostics; wasm rows validate compiler-owned " +
				"i32 slot ABI metadata and backend IRCall arg/return slot " +
				"matching"),
			Stability: ("current evidence/report contract only; no runtime execution " +
				"claim for build-only or wasm targets, no C ABI claim for " +
				"default structs, no native C aggregate ABI claim for wasm " +
				"targets, no performance claim, and no safe-program " +
				"semantics change"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/design/truthful_intent_architecture.md",
				"docs/design/explainable_one_build.md",
				"docs/audits/compiler/backend/abi-verification-v1.md",
			},
		},
		{
			ID:     "compiler.feature-surface-audit",
			Name:   "Full feature surface audit v1",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("P22.0 full feature surface audit report schema " +
				"tetra.language.feature_surface_audit.v1 with scope " +
				"p22.0_full_feature_surface_audit covers first-class " +
				"callables, closures, protocols/trait objects, runtime " +
				"generics, advanced enums/pattern matching, async typed " +
				"errors, structured concurrency, modules/packages, " +
				"macros/metaprogramming, UI/surface, and Eco/capsules; rows " +
				"copy current FeatureRegistry statuses and preserve " +
				"keep-current-bounded, keep-static-only, keep-post-v1, " +
				"unsupported, or experimental-gate decisions without " +
				"promoting a feature unless same-branch evidence exists"),
			Stability: ("current evidence/report contract only; no full v1 language " +
				"guarantee, runtime generic values, trait objects, runtime " +
				"protocol values, macro/metaprogramming system, full " +
				"structured concurrency, cross-platform production UI " +
				"runtime, distributed EcoNet, proof-carrying capsules, " +
				"performance claim, runtime behavior change, or safe-program " +
				"semantics change is claimed"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/design/truthful_intent_architecture.md",
				"docs/design/explainable_one_build.md",
				"docs/audits/compiler/language/full-feature-surface-audit-v1.md",
			},
		},
		{
			ID:     "compiler.ram-contracts",
			Name:   "RAM Contract Compiler reports",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("RAM Contract Compiler report evidence for linux-x64 build " +
				"outputs with tetra.ram-contract-report.v1, " +
				"tetra.memory-grade-report.v1, tetra.proof-store-summary.v1, " +
				"tetra.validation-pipeline-coverage.v1, heap-blockers.json, " +
				"copy-blockers.json, fuzz/ram-contract-fuzz-oracle.json, " +
				"artifact-hashes.json, ram-contract-release-manifest.json, " +
				"--emit-ram-contract-report, --fail-if-heap, --fail-if-copy, " +
				"--fail-if-unbounded, --memory-budget, --ram-contract, " +
				"TETRA4100 diagnostics, validate-ram-contract-report, " +
				"validate-memory-grade-report, validate-proof-store-summary, " +
				"validate-validation-pipeline-coverage, " +
				"validate-heap-blockers, validate-copy-blockers, " +
				"validate-ram-contract-fuzz-oracle, " +
				"validate-ram-contract-release, cross-file heap/copy/grade " +
				"release validation, and " +
				"scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh " +
				"evidence"),
			Stability: ("current report/gate contract only; no zero heap for all " +
				"programs claim, no zero-copy for all programs claim, no " +
				"full formal proof claim, no all-target RAM parity claim, no " +
				"production object memory claim, no production persistent " +
				"memory claim, no runtime behavior change, no performance " +
				"claim, and no safe-program semantics change is claimed"),
			Docs: []string{
				"docs/design/ram_contract_compiler.md",
				"docs/spec/memory/ram_contract_report_schema.md",
				"docs/user/platform/ram_contracts.md",
				"docs/audits/memory/ram-raw/ram-contract-compiler-readiness.md",
				"docs/audits/memory/ram-raw/ram-contract-compiler-handoff.md",
				"docs/audits/memory/ram-raw/raw-contract-implementation-verification-report.md",
			},
		},
		{
			ID:     "compiler.first-class-callables-v1",
			Name:   "First-class callables v1 evidence",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("P22.1 first-class callables v1 report schema " +
				"tetra.language.first_class_callables.v1 with scope " +
				"p22.1_first_class_callables_v1 covers the bounded fnptr " +
				"fast path, fat callable handle, capture safety classifier, " +
				"mutable capture escape diagnostics, resource/thread escape " +
				"diagnostics, fixed ABI width, cross-module interface " +
				"metadata, and storage/callback paths; witnesses parse, " +
				"check, and lower a one-capture 9-slot fnptr value without " +
				"heap environment allocation plus a nine-capture fixed " +
				"4-slot handle value with IRAllocBytes, nine " +
				"IRMemWritePtrOffset writes, nine IRMemReadPtrOffset reads, " +
				"and call ArgSlots 10 RetSlots 1; generated .t4i metadata " +
				"preserves ReturnFunctionHandleValue, heap escape kind, " +
				"capture count, target identity, and ReturnSlots = 4"),
			Stability: ("current evidence/report contract only for the existing safe " +
				"by-value callable model; no variable-width callable ABI, " +
				"exploding return slots, mutable by-reference capture " +
				"support, pointer/resource capture support, thread-boundary " +
				"callable transfer, runtime generic callable polymorphism, " +
				"dynamic callable dispatch, unsafe lifetime relaxation, " +
				"performance claim, runtime behavior change, or safe-program " +
				"semantics change is claimed"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/design/truthful_intent_architecture.md",
				"docs/design/explainable_one_build.md",
				"docs/audits/compiler/language/first-class-callables-v1.md",
				"docs/release/v0_4/v0_4_0_callable_evidence_map.md",
			},
		},
		{
			ID:     "compiler.protocol-trait-object-decision",
			Name:   "Protocol / trait object decision v1",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("P22.2 protocol / trait object decision report schema " +
				"tetra.language.protocol_trait_object_decision.v1 with scope " +
				"p22.2_protocol_trait_object_decision records decision " +
				"keep_static_conformance_only; rows cover static conformance " +
				"fast path, static protocol-bound generics, runtime " +
				"existential decision, explicit dynamic-dispatch gate, " +
				"specialization static abstraction, witness-table boundary, " +
				"trait-object boundary, and registry/docs alignment; " +
				"witnesses parse, check, and lower a static protocol impl " +
				"direct Vec2.draw IRCall, a protocol-bound generic concrete " +
				"id__T_Vec2 direct call, runtime protocol value rejection " +
				"with unknown type 'Drawable', generic-bound " +
				"requirement-call rejection, and P17/P21 known-direct " +
				"specialization evidence"),
			Stability: ("current evidence/report decision only; runtime protocol " +
				"values, trait objects, witness tables, dynamic dispatch, " +
				"conformance-table lookup, runtime existential ABI, broad " +
				"protocol specialization, performance, runtime behavior " +
				"change, and safe-program semantics change are not promoted " +
				"or claimed"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/design/truthful_intent_architecture.md",
				"docs/design/explainable_one_build.md",
				"docs/audits/compiler/language/protocol-trait-object-decision-v1.md",
				"docs/audits/compiler/optimizer/inlining-specialization-v1.md",
				"docs/audits/compiler/optimizer/specialization-machine-code-v1.md",
			},
		},
		{
			ID:     "compiler.verified-track",
			Name:   "Long-term verified track evidence",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("internal P11/P16/P17/P18/P19 verified track: differential " +
				"scalar-i32 stable IR interpreter compares source " +
				"interpreter, stack backend, register backend, and optimized " +
				"backend results; backend differential matrix v1 compares " +
				"supported source, Stack IR, optimized Stack IR, SSA, " +
				"Machine IR, and native execution lanes for scalar, " +
				"slice-sum, branch/loop, and call-loop rows; optimizer pass " +
				"contract v1 requires registered pass names, input/output " +
				"verifier evidence, proof preservation or invalidation rules," +
				" translation validation hooks, stable report rows, " +
				"negative-test markers, and validation metadata with sha256 " +
				"before/after hashes, function set, proof facts, semantic " +
				"checks, and differential samples; optimizer core coverage " +
				"v1 records a bounded evidence-backed P17.1 closure with " +
				"narrow safe const-denominator div_i32/mod_i32 constant " +
				"folding plus same-local comparison algebraic simplification," +
				" narrow SCCP constant-condition, known-local and stored " +
				"safe unary neg_i32 plus safe constant-expression facts " +
				"including safe const-denominator div_i32/mod_i32, constant " +
				"unary neg_i32 and binary-expression branch folding " +
				"including safe const-denominator div_i32/mod_i32 with unary " +
				"min-int and denominator 0 and -1 rejected, immediate and " +
				"forward-terminated single-predecessor label propagation " +
				"plus folded zero-branch target propagation for labels with " +
				"one incoming edge and no fallthrough predecessor, folded " +
				"nonzero-branch fallthrough propagation through immediate " +
				"labels with no explicit incoming branch/jump edges, dynamic " +
				"load-local zero-target and nonzero-fallthrough path facts, " +
				"dynamic zero-comparison eq/ne zero/nonzero path facts, " +
				"fallthrough-predecessor rejection, explicit-incoming " +
				"fallthrough-label rejection, and fallthrough pruning, " +
				"narrow Stack IR adjacent and stack-neutral separated " +
				"single-assignment mem2reg temp promotion including bounded " +
				"comparison-expression, safe const unary neg_i32, safe " +
				"known-local unary neg_i32, safe const " +
				"add_i32/sub_i32/mul_i32 arithmetic, safe known-local " +
				"add_i32/sub_i32/mul_i32 arithmetic, safe const-denominator " +
				"div_i32/mod_i32 producer temps, and safe known-local " +
				"div_i32/mod_i32 producer temps with unary min-int, " +
				"arithmetic overflow, source-local mutation, and denominator " +
				"0 and -1 rejected, bounded DCE for simple dead local stores," +
				" non-trapping comparison-expression stores, safe " +
				"known-local unary neg_i32 stores, safe known-local " +
				"add_i32/sub_i32/mul_i32 stores, safe const-denominator " +
				"div_i32/mod_i32 stores, and safe known-local " +
				"div_i32/mod_i32 stores with unary min-int, arithmetic " +
				"overflow, and denominator 0 and -1 rejected, a narrow " +
				"exact/commutative/mirrored-comparison local-load, " +
				"local-load/constant, unary local neg_i32, safe known-local " +
				"unary neg_i32 value, safe known-local " +
				"add_i32/sub_i32/mul_i32 value, safe known-local cmp_*_i32 " +
				"value, safe known-local div_i32/mod_i32 value, and safe " +
				"const-denominator div_i32/mod_i32 CSE/GVN slice in " +
				"basic-scalar including commutative add/mul/eq/ne and " +
				"mirrored lt/gt/le/ge operand canonicalization, narrow " +
				"proof-tagged LICM pure invariant comparison, add/sub/mul " +
				"arithmetic, known-local add_i32/sub_i32/mul_i32 " +
				"left-or-right operand hoisting, known-local cmp_*_i32 " +
				"left-or-right operand hoisting, safe const-denominator " +
				"div_i32/mod_i32 hoisting, and safe known-local " +
				"div_i32/mod_i32 denominator hoisting, and bounded hot-loop " +
				"shape evidence for scalar sum, scalar constant-stride sum, " +
				"scalar sum-of-squares, scalar product reduction bounded to " +
				"product *= index + 1, scalar branchy max reduction, scalar " +
				"affine sum with compile-time scale and bias 1..127, scalar " +
				"countdown, proof-tagged slice sum, proof-tagged slice " +
				"constant-stride sum, and call-loop machine IR rows; " +
				"inlining specialization coverage v1 records P17.2 target " +
				"rows with narrow monomorphized generic identity/wrapper, " +
				"small-pure inline-small-pure, payload enum known-case match " +
				"and proven-some optional match sccp-constant-branch " +
				"evidence, statically checked protocol/conformance " +
				"direct-call inline-small-pure evidence, statically resolved " +
				"extension-call inline-small-pure evidence, " +
				"inlined/not_inlined report reasons, the same 8-instruction " +
				"body cap, translation validation, constant_stack_store tag " +
				"tracking, known direct Stack IR function symbol boundaries, " +
				"and explicit non-claims for protocol-bound requirement " +
				"calls, witness tables, trait objects, runtime protocol " +
				"values, dynamic dispatch, and conformance-table lookup; " +
				"vectorization coverage v1 records P17.3 initial target rows " +
				"with proof-tagged sum []i32 candidate recognition, " +
				"range-proof evidence, noalias-not-required read-only " +
				"reduction evidence, safe unaligned i32x4 vector backend " +
				"lowering through vector-i32x4-slice-sum-plan, linux-x64 " +
				"native SIMD lowering for proof-tagged step=1 sum []i32, " +
				"scalar tail handling, scalar-i32-slice-sum fallback, " +
				"translation/differential validation against stack fallback, " +
				"proof-tagged copy []u8 vector backend lowering through " +
				"vector-u8x16-copy-plan, noalias required source/dest " +
				"disjoint owned-copy-result evidence, safe unaligned u8x16 " +
				"load/store, scalar-u8-copy fallback, linux-x64 native SIMD " +
				"lowering for proof-tagged copy []u8, copy []u8 " +
				"translation/differential validation against stack fallback, " +
				"proof-tagged simple map over []i32 guarded vector backend " +
				"lowering through vector-i32x4-map-add-const-plan, single " +
				"mutable slice in-place noalias-not-required evidence, safe " +
				"unaligned i32x4 map load/store, scalar-i32-map fallback, " +
				"linux-x64 native SIMD lowering for proof-tagged in-place " +
				"add-constant-1 map []i32, map []i32 " +
				"translation/differential validation against stack fallback, " +
				"proof-tagged memset/memcpy helper evidence through " +
				"vector-u8x16-memset-zero-plan, single mutable slice " +
				"zero-fill noalias-not-required evidence, safe unaligned " +
				"u8x16 zero-store, scalar-u8-memset-zero fallback, linux-x64 " +
				"native SIMD zero-fill lowering for proof-tagged " +
				"memset_zero_u8, memset_zero_u8 translation/differential " +
				"validation against stack fallback, memcpy helper via copy []" +
				"u8 evidence, and explicit no broad SIMD auto-vectorization, " +
				"checked/no-proof copy, overlapping copy, checked/no-proof " +
				"map, broader map-shape vectorization, arbitrary non-zero " +
				"memset, overlapping memcpy, checked/no-proof helper, " +
				"libc/runtime helper lowering, or performance claim; " +
				"PGO/LTO/target-cpu evidence v1 records " +
				"tetra.optimizer.profile.v1 canonical JSON profile " +
				"collection format with duplicate and negative counter " +
				"rejection, internal Options.ProfileInput optimizer profile " +
				"input API, profile_input_policy pass-contract metadata, " +
				"profile digest validation metadata, translation validation " +
				"for profile-input foundation runs, profile-guided rewrite " +
				"policy rejection, profile parsing is evidence-only, " +
				"target-cpu feature detection foundation with portable " +
				"baseline target-feature model, guarded codegen contract, no " +
				"target-specific rewrite, LTO/incremental module summary " +
				"foundation with tetra.incremental.module_summary.v1 " +
				"dependency hash contract and non-consumer boundary, no LTO " +
				"optimizer or incremental speedup claim, final " +
				"safe-semantics closure validator rejects fake " +
				"semantic-changing coverage, profile-guided rewrite policy, " +
				"target-specific optimization evidence, and " +
				"LTO/codegen/linker consumers, and no PGO, LTO, target-cpu, " +
				"or profile flag changes safe-program semantics; actor " +
				"runtime production-boundary audit v1 records " +
				"tetra.runtime.actor.production_boundary.v1 rows for current " +
				"actor runtime limits, scheduler prototype features, " +
				"production runtime acceptance, and full claim blockers, " +
				"with fake full production actor runtime claim rejection and " +
				"explicit non-claims for production multi-threaded actor " +
				"scheduling, non-Linux-x64 distributed actor runtime targets," +
				" message-pool exhaustion/reclamation, full cancellation and " +
				"structured concurrency, full race-safety proof, and " +
				"production broker deployment evidence; async I/O reactor v1 " +
				"records tetra.runtime.io_reactor.v1 rows for Linux epoll v1," +
				" io_uring future boundary, kqueue macOS boundary, IOCP " +
				"Windows boundary, WASI/web adapter boundary, nonblocking " +
				"accept/read/write, readiness polling, task wakeups from I/O " +
				"readiness, timer integration, cancellation, backpressure, " +
				"reactor report rows, HTTP smoke, DB smoke, stress evidence, " +
				"fake full production web-stack rejection, fake " +
				"cross-platform reactor parity rejection, fake io_uring " +
				"rejection, fake runtime-behavior-change rejection, and " +
				"clear production boundary per platform; region-aware stdlib " +
				"v1 records tetra.stdlib.region_aware.v1 rows for " +
				"byte-oriented StringBuilder, VecBytes, fixed-capacity " +
				"HashMapBytes, ByteBuffer, RingBuffer, borrowed JSON/HTTP " +
				"views, PostgreSQL protocol helper reports, " +
				"copy-only-when-needed reports, hidden-heap rejection, and " +
				"fake production web/db/result claim rejection; no full " +
				"production web stack, cross-platform reactor parity, " +
				"io_uring support, runtime behavior change, official " +
				"TechEmpower result, production HTTP/PostgreSQL stack " +
				"promotion, broad generic collection API, or public stdlib " +
				"mode is claimed; self-hosting gate requires register " +
				"backend, optimizer, allocator, and stdlib evidence before a " +
				"self-hosting claim; formal core spec covers values, " +
				"provenance, borrow/copy, bounds proofs, allocation intent, " +
				"raw pointer bounds metadata, and check-elimination validity"),
			Stability: ("current internal evidence only; not a public backend " +
				"selector, source interpreter mode, release optimization " +
				"mode, full self-hosting claim, or full formal proof of the " +
				"language"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/design/explainable_one_build.md",
				"docs/design/compiler/formal_core_semantics.md",
				"docs/design/truthful_intent_architecture.md",
				"docs/audits/master-plan/master-plan-final-20260602.md",
				"docs/audits/master-plan/master-plan-final-20260602-artifact-map.md",
				"docs/audits/performance/truthful-performance-core-baseline.md",
				"docs/audits/compiler/safety/safe-borrow-returns-v1.md",
				"docs/audits/compiler/safety/noalias-mutable-borrow-v1.md",
				"docs/audits/compiler/safety/lifetime-module-boundaries-v1.md",
				"docs/audits/memory/ram-raw/implicit-region-lowering-readiness-v1.md",
				"docs/audits/memory/ram-raw/request-task-region-v1.md",
				"docs/audits/runtime/actors/thread-per-core-allocator-v1.md",
				"docs/audits/memory/ram-raw/raw-pointer-bounds-metadata-v1.md",
				"docs/audits/compiler/backend/backend-coverage-audit-v1.md",
				"docs/audits/compiler/backend/value-ssa-ir-v1.md",
				"docs/audits/compiler/backend/register-backend-coverage-expansion-v1.md",
				"docs/audits/compiler/backend/backend-differential-validation-v1.md",
				"docs/audits/compiler/optimizer/optimizer-pass-contract-v1.md",
				"docs/audits/compiler/optimizer/optimizer-core-coverage-v1.md",
				"docs/audits/compiler/optimizer/inlining-specialization-v1.md",
				"docs/audits/compiler/optimizer/vectorization-v1.md",
				"docs/audits/compiler/optimizer/pgo-lto-target-cpu-v1.md",
				"docs/audits/runtime/actors/actor-runtime-production-boundary-v1.md",
				"docs/audits/runtime/actors/typed-actor-ownership-transfer-v1.md",
				"docs/audits/runtime/actors/per-core-scheduler-v1.md",
				"docs/audits/runtime/actors/async-io-reactor-v1.md",
				"docs/audits/runtime/services/region-aware-stdlib-v1.md",
			},
		},
		{
			ID:     "language.protocol-conformance-mvp",
			Name:   "Static protocol conformance MVP",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("protocol declarations and impl conformance are checked " +
				"statically against extension/static methods, including " +
				"generic requirement signature shape; no witness tables, " +
				"trait objects, or dynamic dispatch model"),
			Stability: ("supported static conformance MVP; runtime polymorphism and " +
				"dynamic dispatch remain post-v1 unless separately gated"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:     "language.callable-mvp",
			Name:   "Callable/function type MVP",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("Level 0 callable surface: function type references, narrow " +
				"symbol-backed non-capturing callable paths, and legacy ptr " +
				"closure local direct calls"),
			Stability: ("current constrained MVP; captured closure escape, storage, " +
				"and full first-class function values remain out of scope"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/policy/v1_feature_status.md",
			},
		},
		{
			ID:     "language.callable-level1",
			Name:   "Callable Level 1 non-capturing expansion",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("production non-capturing symbol-backed callable Level 1: " +
				"function-typed locals, aliases, callbacks, including " +
				"target-set-backed function-typed parameter aliases, " +
				"function-typed parameter storage into struct fields with " +
				"direct field calls or synchronous callback arguments, " +
				"function-typed parameter storage into enum payloads with " +
				"direct payload calls, reassignment, returned enum " +
				"propagation, or synchronous callback arguments, optional " +
				"argument labels on function-typed value calls including " +
				"captured fnptr locals with mixed labeled/unlabeled lists " +
				"rejected, symbol-backed returns, declared function-typed " +
				"local binding, symbol-backed function-typed globals for " +
				"same-module or namespace/selective imported public direct " +
				"calls plus local initialization/reassignment/direct " +
				"callback arguments, non-capturing closure-literal " +
				"function-typed globals, same-module mutable global " +
				"reassignment with direct calls, synchronous callback " +
				"arguments, function-typed returns, generated .t4i " +
				"function-typed parameter local-alias return metadata, and " +
				"local or nested local struct-field/enum-payload " +
				"storage/reassignment/returned-aggregate propagation, " +
				"imported mutable function-typed global boundary diagnostics," +
				" actor/task boundary diagnostics across core.spawn, " +
				"core.task_spawn_i32, core.task_spawn_i32_typed, " +
				"core.task_spawn_group_i32, and " +
				"core.task_spawn_group_i32_typed for workers that directly " +
				"dispatch through same-module or imported immutable " +
				"function-typed globals whose targets touch mutable globals, " +
				"pass mutable function-typed globals as synchronous callback " +
				"arguments, pass same-module or imported symbol-backed " +
				"callback arguments whose targets touch mutable globals, " +
				"pass same-module or imported direct function-typed " +
				"return-call callback arguments whose returned targets or " +
				"multi-return target sets touch mutable globals, preserve " +
				"that classification through local/field alias returns and " +
				"returned struct/enum aggregate fields or payloads across " +
				"module boundaries, directly call function-typed " +
				"locals/struct fields/enum payloads whose targets touch " +
				"mutable globals, reassign them into function-typed locals " +
				"or local struct fields/enum payloads, store them into local " +
				"function-typed struct fields/enum payloads, return them " +
				"from function-typed return helpers, or write mutable " +
				"function-typed globals, and inferable same-module/imported " +
				"generic-symbol initializers, non-capturing generic closure " +
				"literal binding/direct callback/return/mutable local or " +
				"nested struct field reassignment/nested struct field " +
				"initializer/enum payload initializer or reassignment, " +
				"function-typed returns including target-set-backed " +
				"function-typed parameter returns and direct returned-call " +
				"callback arguments, mutable local and nested struct field " +
				"reassignment, function-typed nested struct field " +
				"initializers, and enum payload initializers for inferable " +
				"same-module or imported generic symbols, and " +
				"signature-compatible mutable local reassignment with stable " +
				"diagnostics"),
			Stability: ("current constrained Level 1; generic callable movement is " +
				"limited to declared local initializers, symbol-backed " +
				"function-typed global initializers, same-module mutable " +
				"global reassignment/returns and local or nested local " +
				"struct-field/enum-payload " +
				"storage/reassignment/returned-aggregate propagation, direct " +
				"callback arguments, function-typed returns, mutable local " +
				"or nested struct field reassignment, struct field " +
				"initializers, and enum payload initializers; captured " +
				"closure escape beyond the fnptr Level 2 slice, " +
				"captured/global-escaping callable storage beyond the " +
				"same-module symbol-backed mutable global " +
				"snapshot/reassignment/return slice, and full first-class " +
				"function values remain out of scope"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/policy/v1_feature_status.md",
			},
		},
		{
			ID:     "language.callable-level2",
			Name:   "Callable Level 2 captured closure fnptr values",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("production captured closure Level 2 slice: local " +
				"Int/Bool/String/simple-struct/enum/optional captures " +
				"without ptr/resource payloads may enter fnptr-backed " +
				"function-typed locals, captured ptr closure aliases into " +
				"function-typed locals, mutable function-typed local " +
				"reassignment, same-module mutable function-typed global " +
				"snapshot reassignment from direct closure literals, " +
				"let-bound captured ptr closure locals, direct " +
				"same-module/imported function-typed return calls, immutable " +
				"local aliases initialized from those return calls, mutable " +
				"function-typed locals, local/nested struct fields, local " +
				"enum payloads, whole local or nested structs with function " +
				"fields reassigned from struct literals containing direct " +
				"closure literals or direct return calls, whole local enums " +
				"reassigned from enum constructors containing direct closure " +
				"literals or direct return calls, or same-module or " +
				"source-imported returned enum payloads or returned struct " +
				"enum payloads carrying direct closure literals, with " +
				"generated `.t4i` interface-only returned direct enum or " +
				"aggregate stubs preserving payload metadata for API-only " +
				"validation, or return alias chains that return captured " +
				"closure snapshots with later direct calls, synchronous " +
				"callback arguments, same-module or cross-module " +
				"function-typed returns, direct callback arguments after " +
				"cross-module returns, mutable local reassignments after " +
				"cross-module returns, local or cross-module returned " +
				"struct-field initializer/reassignment, local or " +
				"cross-module returned enum-payload initializer/reassignment," +
				" or throwing direct-try dispatch through that global, " +
				"direct synchronous callback arguments including direct " +
				"closure literals passed to imported callbacks, " +
				"function-typed returns including direct return of let-bound " +
				"captured ptr closure values, local struct fields or enum " +
				"payloads including direct closure-literal container " +
				"initializers in module-aware lowering, direct calls " +
				"including labeled direct calls on captured ptr closures, " +
				"synchronous callback parameters including imported " +
				"parameter-return callbacks, cross-module returned captured " +
				"closures used through locals or direct callback arguments, " +
				"cross-module struct-parameter function-field dispatch " +
				"including namespace/selective imported direct struct " +
				"constructors carrying closure literals or captured ptr " +
				"closure locals, cross-module enum-parameter " +
				"function-payload dispatch including direct " +
				"namespace/selective imported enum constructor arguments, " +
				"immutable local struct fields or enum payloads with up to " +
				"eight by-value snapshot environment slots, explicitly " +
				"declared immutable local direct-try bindings to throwing " +
				"function symbols or captured throwing closure literals, " +
				"captured throwing closure literals in mutable local " +
				"reassignment, direct callback arguments, function-typed " +
				"returns, immutable local struct-field or enum-payload " +
				"direct-try dispatch and aliases, and mutable local " +
				"struct-field or enum-payload reassignment direct-try " +
				"dispatch, declared function-typed returns of a concrete " +
				"throwing symbol followed by local direct-try dispatch, " +
				"immutable local struct-field and enum-payload direct-try " +
				"dispatch for concrete throwing symbols, immutable " +
				"same-module or imported-public function-typed global " +
				"direct-try dispatch/local alias/mutable local " +
				"reassignment/direct callback/struct-field " +
				"initializer/struct-field reassignment/enum-payload " +
				"reassignment paths for concrete throwing symbols, " +
				"same-module mutable function-typed global direct-try " +
				"dispatch, direct throwing callback arguments, and local " +
				"struct-field/enum-payload storage direct-try after " +
				"compatible concrete throwing-symbol initialization or " +
				"reassignment, and direct synchronous throwing " +
				"callback-parameter dispatch through `try cb(...)` when the " +
				"callback parameter type declares the same throws type"),
			Stability: ("current constrained Level 2 fast path; larger immutable " +
				"environments are promoted under " +
				"language.full-first-class-callables, while by-reference " +
				"mutable capture, pointer/resource capture, thread escape, " +
				"unsupported assignment sources, and generic/runtime " +
				"callable polymorphism beyond statically inferred " +
				"function-type surfaces report stable diagnostics or remain " +
				"governed by explicit future features"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/policy/v1_feature_status.md",
			},
		},
		{
			ID:     "language.semantic-clauses-mvp",
			Name:   "Semantic clause checker MVP",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("phase-1 noalloc/noblock/realtime checks on resolved direct " +
				"and supported callable paths"),
			Stability: "static checker MVP; proof-level guarantees remain future work",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/policy/v1_feature_status.md",
			},
		},
		{
			ID:     "safety.effects-mvp",
			Name:   "Effects and uses checker MVP",
			Status: FeatureStatusCurrent,
			Since:  "v0.3.0",
			Scope: ("stable uses effect names and groups with transitive call " +
				"propagation across resolved direct, generic, protocol, and " +
				"supported callable paths; missing uses declarations are " +
				"diagnostics; PLIR exposes checker-enforced optimizer facts " +
				"for " +
				"pure/no-alloc/no-mem-write/no-actor-send/no-unknown-escape " +
				"cases"),
			Stability: ("supported static MVP; no effect inference or proof-level " +
				"effect system guarantee is claimed, and optimizer facts are " +
				"emitted only from checked declared effects"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/effects_capabilities_privacy_v1.md",
				"docs/spec/runtime/capabilities.md",
			},
		},
		{
			ID:     "safety.capabilities-mvp",
			Name:   "Capabilities and unsafe boundary MVP",
			Status: FeatureStatusCurrent,
			Since:  "v0.3.0",
			Scope: ("cap.io and cap.mem opaque tokens are obtained only inside " +
				"unsafe blocks; raw memory/MMIO operations require the " +
				"matching uses effects, unsafe boundary, capability argument," +
				" and capsule permissions for attenuated groups"),
			Stability: ("supported compile-time gating MVP; not a broad safe-code " +
				"capability construction model and current MMIO/raw-memory " +
				"lowering remains minimal"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/capabilities.md",
				"docs/spec/runtime/unsafe.md",
				"docs/spec/runtime/effects_capabilities_privacy_v1.md",
			},
		},
		{
			ID:     "safety.privacy-consent-mvp",
			Name:   "Privacy and consent checker MVP",
			Status: FeatureStatusCurrent,
			Since:  "v0.3.0",
			Scope: ("uses privacy requires privacy semantic clauses; " +
				"secret.i32/SecretInt signatures and privacy builtins " +
				"require a consent token parameter with consent.token type"),
			Stability: ("supported static auditing and call-shape MVP; not " +
				"cryptographic isolation, and distributed consent " +
				"enforcement remains post-v1"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/effects_capabilities_privacy_v1.md",
				"docs/spec/standard_library/stdlib.md",
			},
		},
		{
			ID:     "safety.budget-mvp",
			Name:   "Budget clause lowering MVP",
			Status: FeatureStatusCurrent,
			Since:  "v0.3.0",
			Scope: ("budget(<non-negative integer constant>) requires uses " +
				"budget, lowers to deterministic budget guard instructions " +
				"with stable local-slot metadata, and enforces conservative " +
				"direct-call/task/actor budget context guardrails"),
			Stability: ("supported local lowering plus static edge guardrail MVP; " +
				"not cross-function runtime-wide aggregate accounting, and " +
				"distributed budget enforcement remains post-v1"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/effects_capabilities_privacy_v1.md",
			},
		},
		{
			ID:     "safety.production-core",
			Name:   "Production safety core",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("production local safety model for " +
				"ownership/lifetime/borrow/consume/inout checks, resource " +
				"finalization, callable escape diagnostics, " +
				"effects/capabilities/privacy/consent/budget policy, unsafe " +
				"boundaries, actor/task transfer safety, pointer/MMIO/memory " +
				"capability gates, Memory Production Core v1 report evidence " +
				"through compiler-owned facts rather than " +
				"report-reconstructed truth, a memory cost model with " +
				"zero_cost_proven, dynamic_check_required, " +
				"instrumentation_only, unsupported_rejected, and " +
				"conservative_fallback report classes, a memory fuzz oracle " +
				"with Tier 1 short CI smoke, Tier 2 nightly fuzz, Tier 3 " +
				"release-blocking focused memory fuzz, explicit oracle " +
				"categories, MEM-FUZZ-012 deterministic v0-v11 release " +
				"evidence rows, required crash/miscompile repro artifacts, " +
				"release-blocking unsafe/bounds/storage/report " +
				"classifications, memory production final audit with " +
				"artifact map and explicit nonclaims, validate-island-proof " +
				"independent-ish verifier evidence, --islands-debug " +
				"sanitizer smoke, island-proof-fuzz-summary deterministic " +
				"mutation evidence, leak/resource finalization evidence, and " +
				"an integrated Memory/Islands/Surface release gate with " +
				"memory-islands-surface-production-manifest.json and " +
				"artifact-hashes.json, and no Memory 100% claim or " +
				"unsupported unsafe pointer safety claim"),
			Stability: ("release-gated current profile with explicit diagnostics for " +
				"unsupported distributed, cryptographic, formal-proof, " +
				"runtime-wide guarantees, arbitrary unsafe external pointer " +
				"safety, full target parity, all-target Surface support, " +
				"clean release-candidate checkout claims, and no production " +
				"object memory or production persistent memory claim"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/runtime/effects_capabilities_privacy_v1.md",
				"docs/spec/runtime/unsafe.md",
				"docs/spec/memory/memory_report_schema_v1.md",
				"docs/spec/memory/islands.md",
				"docs/design/memory/memory_production_core_v1.md",
				"docs/design/memory/memory_cost_model.md",
				"docs/audits/memory/islands/memory-fuzz-oracle-v1.md",
				"docs/testing/fuzz_property_stress.md",
				"docs/audits/memory/production/memory-production-core-v1-baseline.md",
				"docs/audits/memory/production/memory-production-core-v1-gap-map.md",
				"docs/audits/memory/production/memory-production-core-v1-supported-surface.md",
				"docs/audits/memory/islands/memory-target-capability-matrix.md",
				"docs/audits/memory/production/memory-production-core-v1-final.md",
				"docs/audits/memory/production/memory-production-core-v1-artifact-map.md",
				"docs/audits/memory/production/memory-production-core-v1-nonclaims.md",
				"docs/release/surface/memory_islands_surface_scope.md",
				"docs/audits/memory/ideal-v0-v1/memory-ideal-vslice-v0-baseline.md",
				"docs/audits/memory/ideal-v0-v1/memory-ideal-vslice-v0-correlation.md",
				"docs/audits/memory/ideal-v0-v1/memory-ideal-vslice-v0-final.md",
				"docs/audits/memory/ideal-v0-v1/memory-ideal-vslice-v1-correlation.md",
				"docs/audits/memory/ideal-v0-v1/memory-ideal-vslice-v1-final.md",
				"docs/audits/memory/ideal-v2-v4/memory-ideal-vslice-v2-correlation.md",
				"docs/audits/memory/ideal-v2-v4/memory-ideal-vslice-v2-final.md",
				"docs/audits/memory/ideal-v2-v4/memory-ideal-vslice-v3-correlation.md",
				"docs/audits/memory/ideal-v2-v4/memory-ideal-vslice-v3-final.md",
				"docs/audits/memory/ideal-v2-v4/memory-ideal-vslice-v4-correlation.md",
				"docs/audits/memory/ideal-v2-v4/memory-ideal-vslice-v4-final.md",
				"docs/audits/memory/ideal-v5-v7/memory-ideal-vslice-v5-correlation.md",
				"docs/audits/memory/ideal-v5-v7/memory-ideal-vslice-v5-final.md",
				"docs/audits/memory/ideal-v5-v7/memory-ideal-vslice-v6-bounds-correlation.md",
				"docs/audits/memory/ideal-v5-v7/memory-ideal-vslice-v6-bounds-final.md",
				"docs/audits/memory/ideal-v5-v7/memory-ideal-vslice-v7-ffi-correlation.md",
				"docs/audits/memory/ideal-v5-v7/memory-ideal-vslice-v7-ffi-final.md",
				"docs/audits/memory/ideal-v8-v9/memory-ideal-vslice-v8-report-correlation.md",
				"docs/audits/memory/ideal-v8-v9/memory-ideal-vslice-v8-report-final.md",
				"docs/audits/memory/ideal-v8-v9/memory-ideal-vslice-v9-storage-correlation.md",
				"docs/audits/memory/ideal-v8-v9/memory-ideal-vslice-v9-storage-final.md",
				"docs/audits/memory/ideal-v10-v11/memory-ideal-vslice-v10-async-cancel-correlation.md",
				"docs/audits/memory/ideal-v10-v11/memory-ideal-vslice-v10-async-cancel-final.md",
				"docs/audits/memory/ideal-v10-v11/memory-ideal-vslice-v11-dynproto-correlation.md",
				"docs/audits/memory/ideal-v10-v11/memory-ideal-vslice-v11-dynproto-final.md",
			},
		},
		{
			ID:     "language.globals-properties-capsule-mvp",
			Name:   "Top-level globals, properties, and capsule metadata MVP",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("constant global initializers, property declarations, and " +
				"compile-time capsule metadata validation"),
			Stability: "supported MVP with explicit initializer/runtime limitations",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
			},
		},
		{
			ID:     "language.slice-mvp",
			Name:   "Native-first slice MVP",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("[]u8/[]u16/[]i32/[]bool helpers including make_* and " +
				"island_make_* allocation-length contracts, island " +
				"compile-compatible fallback paths, checked slice " +
				"window/prefix/suffix safe view constructors, proof-tagged " +
				"for-loop and supported while-loop bounds-check removal " +
				"through PLIR CFG/dominance/range facts, explicit " +
				"borrow/copy/copy_into methods, and checked String byte " +
				"window/prefix/suffix/borrow/copy/copy_into methods with " +
				"provenance-aware PLIR facts, allocation/proof/bounds report " +
				"evidence, and actor-boundary copy diagnostics"),
			Stability: "supported MVP with documented layout/runtime constraints",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/standard_library/stdlib.md",
			},
		},
		{
			ID:     "language.safe-view-lifetime-contracts-v1",
			Name:   "Safe View Lifetime Contracts v1",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("borrowed return signatures for supported slice/String byte " +
				"views, cross-module borrowed return preservation, " +
				"single-source borrowed return validation, recursive " +
				"hidden-borrow escape checks for " +
				"structs/enums/optionals/generic wrappers, actor and " +
				"typed-task copy-required boundaries, and PLIR/proof/alloc " +
				"evidence for borrow/copy/borrowed-return facts"),
			Stability: ("current conservative lifetime contract for safe view " +
				"surfaces; named lifetimes, generic lifetime parameters, " +
				"arbitrary borrowed aggregate returns, full Unicode String " +
				"lifetime semantics, Rust-like borrow checking, and " +
				"production FFI lifetime contracts remain outside this claim"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/design/truthful_safe_values.md",
				"docs/design/memory/provenance_lifetime_ir.md",
				"docs/design/truthful_intent_architecture.md",
				"docs/user/reference/examples_index.md",
			},
		},
		{
			ID:     "language.ownership-markers-mvp",
			Name:   "Ownership markers MVP",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("conservative borrow/inout/consume marker checks for local " +
				"calls, same-module/cross-module struct-field and " +
				"enum-payload partial consume with whole-value " +
				"call/let/return and enum wrapper-constructor rejection plus " +
				"stable TETRA2101 diagnostics including " +
				"same-module/cross-module CLI JSON evidence, " +
				"same-module/cross-module whole-copy rejection after partial " +
				"struct/enum consume with stable TETRA2101 CLI JSON evidence," +
				" mutable struct-field/whole-struct/whole-enum " +
				"reinitialization after partial consume, aliasing, " +
				"use-after-consume, and borrow escape diagnostics for scalar " +
				"ptr including same-module/cross-module scalar ptr consume " +
				"and inout assignment plus match/catch-expression return " +
				"escapes and typed-error throw ptr/region payload escapes, " +
				"same-module/cross-module borrowed scalar ptr escapes " +
				"through ptr-containing struct inout assignment, " +
				"same-module/cross-module fixed-array alias return plus " +
				"direct global assignment, optional global assignment, and " +
				"inout assignment escapes with stable TETRA2102 diagnostic " +
				"evidence, borrowed string alias return/global assignment " +
				"escapes with stable TETRA2102 CLI JSON evidence, " +
				"slice-containing struct literal/alias/nested " +
				"struct/enum-payload return and inout assignment escapes " +
				"plus slice-containing enum direct/alias return escapes with " +
				"stable same-module/cross-module TETRA2102 CLI JSON evidence," +
				" slice-containing struct/enum owned/consume/inout call " +
				"escapes with stable same-module/cross-module and imported " +
				"direct TETRA2101 CLI JSON evidence, function-typed " +
				"value/struct-field/enum-payload callback slice-containing " +
				"struct/enum owned/consume/inout call rejections with stable " +
				"TETRA2101 JSON diagnostic evidence, ptr/slice optional " +
				"assignment return/owned/consume/inout escape with stable " +
				"same-module/cross-module TETRA2101/TETRA2102 CLI JSON " +
				"evidence for slice optional assignment, " +
				"same-module/cross-module slice optional payload binding " +
				"owned/consume/inout call, inout-assignment, and global " +
				"assignment escapes with stable TETRA2101/TETRA2102 CLI JSON " +
				"evidence, same-module/cross-module ptr optional assignment " +
				"if-let/match global escape with stable TETRA2102 JSON " +
				"diagnostic evidence, same-module/cross-module ptr enum " +
				"alias return escape with stable TETRA2102 JSON diagnostic " +
				"evidence, same-module/cross-module slice optional-payload " +
				"inout/global assignment escapes with stable TETRA2102 JSON " +
				"diagnostic evidence, same-module/cross-module nested slice " +
				"enum-payload return/inout/global assignment escapes with " +
				"stable TETRA2102 JSON diagnostic evidence, " +
				"same-module/cross-module nested slice struct " +
				"return/inout/global assignment escapes with stable " +
				"TETRA2102 JSON diagnostic evidence, " +
				"same-module/cross-module direct slice global assignment " +
				"with stable TETRA2102 JSON diagnostic evidence, " +
				"same-module/cross-module optional ptr global assignment " +
				"with stable TETRA2102 JSON diagnostic evidence, and " +
				"same-module/cross-module optional aggregate global " +
				"assignment with stable TETRA2102 JSON diagnostic evidence, " +
				"and same-module/cross-module ptr-containing aggregate " +
				"whole/field/alias/nested-field return escapes with stable " +
				"TETRA2102 JSON diagnostic evidence, " +
				"same-module/cross-module whole-aggregate global assignment " +
				"with stable TETRA2102 JSON diagnostic evidence, " +
				"same-module/cross-module ptr-containing enum whole-value " +
				"global assignment with stable TETRA2102 JSON diagnostic " +
				"evidence, same-module/cross-module global field target " +
				"assignment with stable TETRA2102 JSON diagnostic evidence, " +
				"same-module/cross-module aggregate and nested-aggregate " +
				"global field escapes with stable TETRA2102 JSON diagnostic " +
				"evidence, same-module/cross-module ptr-containing and " +
				"nested ptr-containing aggregates plus ptr-containing enum " +
				"aggregates including whole-aggregate, whole-enum, global " +
				"field target, and global field escapes with stable " +
				"TETRA2102 CLI JSON evidence, optional ptr payloads " +
				"including same-module/cross-module whole-optional " +
				"use-after-payload-consume diagnostics with stable TETRA2101 " +
				"CLI JSON evidence and same-module/cross-module " +
				"optional-payload whole-value rejection after payload " +
				"consume/free with stable TETRA2101 JSON diagnostic evidence," +
				" same-module/cross-module pattern-bound enum payload and " +
				"if-let/match optional payload return, owned/consume/inout " +
				"call, inout-assignment, and global escapes with " +
				"same-module/cross-module ptr enum-payload " +
				"return/global/inout assignment escapes with stable " +
				"TETRA2102 JSON diagnostic evidence and " +
				"same-module/cross-module ptr optional-payload " +
				"return/global/inout assignment escapes with stable " +
				"TETRA2102 JSON diagnostic evidence, plus " +
				"same-module/cross-module ptr-containing/nested aggregate " +
				"owned/consume/inout call rejections with stable TETRA2101 " +
				"JSON diagnostic evidence, same-module/cross-module ptr " +
				"enum-payload owned/consume/inout call rejections with " +
				"stable TETRA2101 JSON diagnostic evidence, " +
				"same-module/cross-module ptr optional-payload " +
				"owned/consume/inout call rejections with stable TETRA2101 " +
				"JSON diagnostic evidence, and same-module/cross-module " +
				"slice optional-payload owned/consume/inout call rejections " +
				"with stable TETRA2101 JSON diagnostic evidence, " +
				"same-module/cross-module generic aggregate and optional-ptr " +
				"owned/consume/inout instantiations including " +
				"slice-containing struct/enum aggregate instantiations with " +
				"stable TETRA2101 CLI JSON evidence, " +
				"same-module/cross-module generic " +
				"borrow-aggregate/optional-ptr return diagnostics with " +
				"stable TETRA2102 CLI JSON evidence, " +
				"same-module/cross-module protocol parameter ownership " +
				"matching plus same-module/cross-module protocol impl " +
				"parameter ownership mismatch diagnostics with stable " +
				"TETRA2001 CLI JSON evidence and same-module/cross-module " +
				"generic protocol requirement parameter ownership mismatch " +
				"diagnostics with stable TETRA2001 JSON diagnostic evidence, " +
				"same-module/cross-module function-typed " +
				"value/struct-field/enum-payload optional-ptr " +
				"owned/consume/inout callback diagnostics with stable " +
				"TETRA2101 CLI JSON evidence, imported direct " +
				"owned/consume/inout call boundaries including struct, " +
				"enum-payload, and nested ptr-containing aggregate arguments," +
				" with imported direct ptr-containing/nested aggregate " +
				"owned/consume/inout call rejections with stable TETRA2101 " +
				"JSON diagnostic evidence, and supported mutable global " +
				"assignment boundaries"),
			Stability: ("supported conservative MVP; this is not a full SSA lifetime " +
				"solver and ambiguous lifetime merges remain diagnostics"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:     "language.resource-lifetime-mvp",
			Name:   "Resource lifetime MVP",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("conservative resource finalization checks for task handles, " +
				"task groups, island handles, region-backed slices, structs " +
				"containing them, branch/match/loop task-handle maybe-joined," +
				" task-group maybe-closed, and island maybe-freed merge " +
				"diagnostics; branch/match/loop resource finalization merge " +
				"diagnostics with stable TETRA2101 JSON evidence, stable " +
				"ownership safety JSON diagnostics for resource " +
				"use-after-free, double-join, and ambiguous-provenance cases " +
				"including same-module/cross-module struct-field and " +
				"enum-payload alias use-after-free, same-module/cross-module " +
				"struct-field and enum-payload alias use-after-free with " +
				"stable TETRA2101 JSON diagnostic evidence, plus task-group " +
				"use-after-close, struct-field aliases and enum-payload " +
				"aliases including same-module/cross-module " +
				"task-handle/task-group struct-field/enum-payload join/close " +
				"aliases, same-module/cross-module task-handle " +
				"struct-field/enum-payload alias join diagnostics with " +
				"stable TETRA2101 JSON diagnostic evidence, and " +
				"same-module/cross-module task-group " +
				"struct-field/enum-payload alias close diagnostics with " +
				"stable TETRA2101 JSON diagnostic evidence, " +
				"same-module/cross-module enum-constructor return resource " +
				"aliases with stable TETRA2101 CLI JSON evidence, " +
				"same-module typed-error throw/catch and rethrow-through-try " +
				"enum-payload resource aliases with stable TETRA2101 JSON " +
				"diagnostic evidence, generated .t4i " +
				"direct/local/aggregate-local-alias/aggregate-field-access/ag" +
				"gregate-field-local-alias resource return, " +
				"assignment/let/direct-if-let/direct-match/field-local/if-let" +
				"/match optional and nested/field-local nested optional " +
				"resource return, typed-error direct/field-local-alias throw," +
				" and rethrow-through-try direct/field-local-alias " +
				"provenance stubs, same-module/cross-module monomorphized " +
				"generic struct task-handle/task-group/island resource " +
				"aliases with stable TETRA2101 CLI JSON evidence, " +
				"if-let/match optional-payload return aliases including " +
				"nested struct-field and enum-payload wrappers with stable " +
				"same-module/cross-module TETRA2101 CLI JSON evidence, " +
				"same-module/cross-module task-handle/task-group " +
				"if-let/match optional-payload join/close aliases with " +
				"stable TETRA2101 CLI JSON evidence, " +
				"same-module/cross-module island whole-optional " +
				"use-after-payload-free diagnostics with stable TETRA2101 " +
				"CLI JSON evidence, same-module/cross-module transitive " +
				"interprocedural task-handle/task-group/island resource " +
				"aliases with stable TETRA2101 CLI JSON evidence, " +
				"same-module and cross-module transitive interprocedural " +
				"resource alias double-use, and ambiguous provenance " +
				"diagnostics"),
			Stability: ("supported conservative MVP; tracks common local scope and " +
				"control-flow merge cases, but is not a full SSA lifetime " +
				"solver"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:     "actors.task-transfer-safety",
			Name:   "Actor/task transfer safety MVP",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("conservative actor/task ownership transfer checks for " +
				"worker entrypoints, sendable results, handle transfer, " +
				"branch/match/loop actor consume reuse diagnostics with " +
				"stable TETRA2101 CLI JSON evidence, actor/task " +
				"use-after-transfer diagnostics with stable TETRA2101 CLI " +
				"JSON evidence, island transfer non-local-payload rejection " +
				"with stable TETRA2101 CLI JSON evidence, " +
				"same-module/cross-module transitive actor consume alias " +
				"diagnostics with stable TETRA2101 CLI JSON evidence, " +
				"same-module/cross-module monomorphized generic struct actor " +
				"consume alias diagnostics with stable TETRA2101 CLI JSON " +
				"evidence, same-module/cross-module task_group_cancel return " +
				"provenance diagnostics with stable TETRA2101 CLI JSON " +
				"evidence, same-module/cross-module actor if-let/match " +
				"optional-payload, struct-field, and enum-payload consume " +
				"alias diagnostics, same-module/cross-module actor " +
				"struct-field/enum-payload alias transfer diagnostics with " +
				"stable TETRA2101 JSON diagnostic evidence, " +
				"same-module/cross-module actor/task if-let/match " +
				"optional-payload alias transfer diagnostics with stable " +
				"TETRA2101 JSON diagnostic evidence, " +
				"same-module/cross-module task-handle " +
				"struct-field/enum-payload alias transfer diagnostics with " +
				"stable TETRA2101 JSON diagnostic evidence, " +
				"same-module/cross-module task-handle " +
				"struct-field/enum-payload alias join diagnostics with " +
				"stable TETRA2101 JSON diagnostic evidence, release-covered " +
				"cooperative task_group_cancel wake/join behavior, and task " +
				"group lifecycle status/close smokes"),
			Stability: ("supported conservative local MVP; distributed actors, full " +
				"race-safety proofs, full cancellation semantics, and " +
				"structured concurrency remain outside the current support " +
				"claim"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
				"docs/user/platform/async_actors_guide.md",
			},
		},
		{
			ID:     "language.lifetime-ssa",
			Name:   "Lifetime SSA local join solver",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("production SSA-like local lifetime join analysis for " +
				"ownership consume state, resource finalization state, " +
				"branch/match/loop flow snapshots, branch/match/loop " +
				"resource finalization merge diagnostics with stable " +
				"TETRA2101 JSON evidence, optional region-wrapper escapes " +
				"with stable TETRA2102 diagnostics, same-module and " +
				"interface-only cross-module per-field interprocedural " +
				"region summaries for aggregate returns from multiple island " +
				"parameters, including optional aggregate wrappers, enum " +
				"payload wrappers, branch aggregate wrappers, match " +
				"aggregate wrappers, if-let aggregate wrappers, mixed " +
				"safe/provenance aggregate branch and match returns, and " +
				"optional mixed safe/provenance aggregate branch merges, and " +
				"maybe-consumed diagnostics"),
			Stability: ("current local/control-flow solver; richer interprocedural " +
				"lifetime proofs, broad alias modeling, race proofs, and " +
				"full formal lifetime guarantees remain under full-v1 scope"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/ownership_v1.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:        "language.task-handles-mvp",
			Name:      "Typed task handle wrappers MVP",
			Status:    FeatureStatusCurrent,
			Since:     "v0.2.0",
			Scope:     "typed task handle wrappers for slot counts 2..8 in the current runtime path",
			Stability: "supported MVP; layouts above 8 are rejected",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/user/platform/async_actors_guide.md",
			},
		},
		{
			ID:     "eco.local-package-lifecycle",
			Name:   "Local Eco package lifecycle",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("local verify, lock generation/validation, pack/unpack, " +
				"vault, stable and beta publish metadata, target-aware " +
				"download, stable/beta TetraHub store fixtures, local mirror " +
				"reports, and single-origin HTTP(S) fetch into a verified " +
				"local store"),
			Stability: ("local tooling support with stable publish, mirror, and HTTP " +
				"fetch integrity metadata; distributed network ecosystem is " +
				"not implied"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/user/platform/eco_package_guide.md",
				"docs/spec/policy/eco_publishing_v1.md",
			},
		},
		{
			ID:     "stdlib.core-current",
			Name:   "Core standard library current profile",
			Status: FeatureStatusCurrent,
			Since:  "v0.2.0",
			Scope: ("release-covered lib.core helper modules with a " +
				"capability-gated linux-x64 filesystem exists slice plus " +
				"filesystem+scheduler composition and scheduler-restriction " +
				"regression smokes, x86 and x32 no-runtime " +
				"stdout/string-literal executable smokes, x86 and x32 stderr " +
				"fd runtime smokes through core.net_write(2), x86 and x32 " +
				"allocator success/failure executable smokes for " +
				"core.alloc_bytes plus raw store/load and checked " +
				"invalid-size/mmap-error exit lowering, x86 and x32 " +
				"island/free executable smokes for scoped island " +
				"allocation/free and debug free guard lowering, x86 and x32 " +
				"filesystem+scheduler self-host composition smokes, x86 and " +
				"x32 bounded two-spawn actors/task/task-group self-host " +
				"smokes, x86 and x32 typed-task self-host smokes, x86 and " +
				"x32 staged typed-task self-host smokes, x86 and x32 typed " +
				"task-group self-host smokes, and pure fs_exists " +
				"linux-x86/linux-x32 smokes, executable Linux TCP socket " +
				"client/server I/O helpers with recv/send, SO_REUSEPORT, " +
				"TCP_NODELAY, nonblocking accept convenience, and epoll " +
				"add/mod/delete plus wait-one readiness flag capture and " +
				"predicates, stable crypto interface helpers, stable " +
				"networking endpoint policy helpers, executable HTTP/1.1 " +
				"String and byte-buffer request-line routing, request-head " +
				"framing, and response byte-buffer helpers, executable JSON " +
				"byte-buffer response helpers, and internal P7/P19 runtime " +
				"evidence for region-aware collection/buffer storage " +
				"planning, P19 byte-oriented " +
				"StringBuilder/VecBytes/HashMapBytes/RingBuffer helpers, " +
				"borrowed JSON parsing, borrowed HTTP request-head parsing, " +
				"and PostgreSQL borrowed/binary row helpers"),
			Stability: ("current import paths and smoke coverage; filesystem exists " +
				"is host-backed on linux-x64 including filesystem+scheduler " +
				"composition and scheduler-restriction regression smokes, " +
				"x86 and x32 no-runtime stdout/string-literal executables " +
				"plus core.net_write fd=2 stderr runtime executables, " +
				"core.alloc_bytes allocator success/failure executable " +
				"smokes, and scoped island/free executable smokes are " +
				"covered by ABI smokes, composable with the x86 and x32 " +
				"self-host scheduler slices, x86 and x32 two-spawn " +
				"actors/task/task-group flows are covered by self-host " +
				"runtime smokes, x86 and x32 typed-task handles are covered " +
				"by self-host runtime smokes with staged typed-task coverage," +
				" x86 and x32 typed task-group composition are covered by " +
				"self-host runtime smokes, and pure fs_exists " +
				"linux-x86/linux-x32 smokes remain covered; full x86/x32 " +
				"allocator/free/panic parity remains unpromoted. net socket " +
				"open/bind/connect/listen/accept/read/recv/write/send/nonbloc" +
				"king/close plus SO_REUSEPORT, TCP_NODELAY, " +
				"SOCK_NONBLOCK/SOCK_CLOEXEC accept helpers, and epoll " +
				"create/add-read/add-read-write/mod-read/mod-read-write/delet" +
				"e/wait-one/wait-one-into helpers with " +
				"EPOLLIN/EPOLLOUT/EPOLLERR/EPOLLHUP predicates are " +
				"host-backed on linux-x64, crypto exposes deterministic " +
				"interface helpers, networking exposes deterministic " +
				"endpoint policy helpers, HTTP helpers classify TechEmpower " +
				"request lines from String text or caller-owned byte buffers," +
				" locate CRLFCRLF request-head boundaries for pipelined " +
				"buffers, and write compact response payloads into " +
				"caller-owned buffers, JSON helpers write compact response " +
				"bodies into caller-owned buffers, and P7/P19 internal " +
				"runtime helpers provide checked storage/provenance/copy " +
				"evidence without promoting broad generic collection APIs, " +
				"production web/db stacks, or official TechEmpower claims"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/standard_library/stdlib.md",
				"docs/user/platform/standard_library_guide.md",
				"docs/audits/runtime/services/region-aware-stdlib-v1.md",
			},
		},
		{
			ID:     "stdlib.experimental-mirrors",
			Name:   "Standard-library compatibility mirrors",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("production compatibility mirrors under lib.experimental.* " +
				"forward to lib.core.* modules for legacy source " +
				"compatibility"),
			Stability: ("current compatibility bridge; stable callers should import " +
				"lib.core.* directly, and no broader host API guarantee is " +
				"implied beyond the mirrored lib.core surface"),
			Docs: []string{
				"docs/spec/standard_library/stdlib.md",
				"docs/spec/standard_library/stdlib_naming_versioning.md",
				"docs/user/platform/standard_library_guide.md",
			},
		},
		{
			ID:     "language.enum-payload-match",
			Name:   "Enum payload constructors and exhaustive match/catch",
			Status: FeatureStatusCurrent,
			Since:  "v0.3.0",
			Scope: ("positional enum payload constructors and payload bindings " +
				"for match/catch/if-let, with exhaustive unguarded enum " +
				"match/catch coverage and stable diagnostics for arity, type," +
				" duplicate, default-order, and payload-syntax errors"),
			Stability: ("supported v0.3.0 static/runtime slice; cross-module enum " +
				"constructor/match paths are checked and lowered, while " +
				"advanced ADT constructors, nested destructuring patterns, " +
				"richer payload algebra, and guard expansion remain " +
				"future/post-v1"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/flow_syntax_v1.md",
				"docs/spec/flow/v0_3_scope.md",
			},
		},
		{
			ID:     "language.protocol-bound-generics-static",
			Name:   "Static protocol-bound generics",
			Status: FeatureStatusCurrent,
			Since:  "v0.3.0",
			Scope: ("generic function type parameters with protocol bounds are " +
				"validated statically during monomorphization, including " +
				"same-module and cross-module impl conformance with " +
				"parameter ownership markers, requirement signature shape, " +
				"and visibility diagnostics"),
			Stability: ("supported v0.3.0 static conformance slice; calling protocol " +
				"requirements through generic bounds, witness tables, trait " +
				"objects, runtime protocol values, and dynamic dispatch " +
				"remain unsupported"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/v0_3_scope.md",
				"docs/spec/flow/flow_syntax_v1.md",
			},
		},
		{
			ID:     "ui.metadata-v1",
			Name:   "UI metadata v0.4.0 surface",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("legacy metadata compatibility surface preserving the " +
				"production UI metadata contract for checked view/state " +
				"declarations, deterministic tetra.ui.v0.4.0 JSON, " +
				"browser-backed web command-dispatch runtime artifacts, " +
				"style metadata preview attributes, accessibility metadata " +
				"preview attributes, and native shell command-dispatch text " +
				"plus JSON trace sidecars with deterministic widget-tree " +
				"artifacts"),
			Stability: ("current metadata plus wasm32-web command dispatch covered " +
				"by post-v0.4 Web UI runtime smoke and native shell command " +
				"dispatch/widget-tree traces for lowered scalar state " +
				"operations; it is not the new Tetra Surface runtime, not " +
				"the pure-Tetra component model, and not a basis for new " +
				"Surface host claims; style and accessibility metadata are " +
				"preview attributes only, while executable Linux-x64 native " +
				"runtime evidence is tracked by ui.native-runtime"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/ui/ui_v1.md",
				"docs/spec/ui/ui_v0.4.0.md",
				"docs/spec/policy/v1_feature_status.md",
				"docs/user/surface/wasm_ui_guide.md",
			},
		},
		{
			ID:     "ui.toolkit-core",
			Name:   "UI Toolkit Core contract runtime",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("production platform-independent UI Toolkit Core contract " +
				"for tetra.ui.toolkit.v1 with widget model, layout model, " +
				"style model, accessibility model, event model, state " +
				"binding model, widget tree construction, layout measurement " +
				"and placement, event dispatch, state binding/update, focus " +
				"traversal, timer/async command/redraw/error recovery " +
				"evidence, deterministic compiler .ui.toolkit.json emission, " +
				"and validator-gated runtime trace artifacts"),
			Stability: ("current toolkit core only; validators reject metadata-only, " +
				"preview-only, runtime-less, native-shell sidecar-only, " +
				"web-only, docs-only, build-only, fake/mock/placeholder " +
				"evidence, and this does not claim GTK/Qt/OS platform " +
				"backend production, Windows/macOS GUI production, or full " +
				"cross-platform UI"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/ui/ui_toolkit_core.md",
				"docs/spec/ui/ui_v0.4.0.md",
			},
		},
		{
			ID:     "ui.surface-core",
			Name:   "Tetra Surface core",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("surface-v1-linux-web current release scope: pure-Tetra UI, " +
				"tiny Surface Host ABI, software RGBA framebuffer " +
				"presentation, owned/copy-safe event and text buffers, " +
				"developer fast rebuild evidence through tetra surface dev, " +
				"static Surface inspector report evidence, and release " +
				"evidence for headless, linux-x64 real-window, and " +
				"wasm32-web browser-canvas targets"),
			Stability: ("current only for the bounded Surface v1 linux/web release " +
				"scope; macOS, Windows, wasm32-wasi UI, GPU rendering, " +
				"platform widgets, DOM/user-JS app logic, dynamic " +
				"trait-object widgets, witness-table component dispatch, and " +
				"rich text editor claims remain unsupported or future work"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/user/surface/surface_guide.md",
				"docs/release/surface/surface_v1_release_contract.md",
				"docs/release/surface/surface_v1_release_notes.md",
				"docs/release/surface/surface_v1_release_audit.md",
			},
		},
		{
			ID:     "ui.surface-block-system",
			Name:   "Tetra Surface Block System",
			Status: FeatureStatusExperimental,
			Scope: ("Block-first Surface architecture implementation track with " +
				"`lib.core.block` data model support for Block as the core " +
				"Surface primitive for layout, paint, text, assets, " +
				"input/events, states, motion, and accessibility; existing " +
				"Button/Card/TextField-like helpers are " +
				"recipes/compatibility over Block rather than core widget " +
				"primitives; scoped `tetra.surface.block-system.gate.v1` " +
				"reports include `block_system.memory_budget` evidence under " +
				"reports/surface-block/p18-budget"),
			Stability: ("experimental and not current Surface v1 production support, " +
				"with same-commit target evidence for headless, linux-x64 " +
				"real-window, and wasm32-web browser-canvas Block-system " +
				"reports, validators, artifact hashes, and release-gate " +
				"integration; not production support and no production Block " +
				"claim, Electron, React, DOM, CSS runtime, user JavaScript, " +
				"Chromium, platform-native widget, GPU renderer, or " +
				"cross-platform desktop replacement claim is implied"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/surface/surface_v1.md",
				"docs/user/surface/surface_guide.md",
				"docs/user/reference/examples_index.md",
				"docs/release/surface/surface_v1_release_contract.md",
				"docs/release/surface/surface_v1_release_notes.md",
				"docs/release/surface/surface_v1_release_audit.md",
			},
		},
		{
			ID:     "ui.surface-morph-capsule",
			Name:   "Tetra Surface Morph Capsule",
			Status: FeatureStatusExperimental,
			Scope: ("experimental Morph Capsule authoring layer over the Surface " +
				"Block System; `lib.core.morph` defines scoped capsule " +
				"tokens, materials, seven affordances, state lenses, motion " +
				"presets, and eleven recipes that expand into Block graph " +
				"evidence for five " +
				"`examples/surface/morph_core/surface_morph_*.tetra` " +
				"reference apps; the recipe layer expands into Block and " +
				"never promotes Button/Card/TextField-style helpers to core " +
				"primitives; `tetra.surface.morph.gate.v1` records " +
				"deterministic headless same-commit Morph reports plus " +
				"artifact hashes and validates " +
				"`tetra.surface.token-graph.contract.v1` scoped token graph " +
				"diagnostics plus P08 recipe expansion evidence"),
			Stability: ("experimental evidence layer and not Surface v1 production " +
				"support; Morph does not add core widget primitives, " +
				"platform widgets, CSS cascade, DOM app logic, " +
				"React/Electron runtime, GPU renderer, or cross-target " +
				"desktop replacement support"),
			Docs: []string{
				"docs/spec/surface/morph/surface_morph.md",
				"docs/spec/surface/surface_token_graph.md",
				"docs/spec/core/current_supported_surface.md",
				"docs/user/surface/surface_guide.md",
				"docs/user/reference/examples_index.md",
				"docs/user/surface/surface_morph_recipe_cookbook.md",
				"docs/user/platform/standard_library_guide.md",
				"docs/release/surface/surface_v1_release_contract.md",
				"docs/release/surface/surface_v1_release_notes.md",
			},
		},
		{
			ID:     "ui.surface-headless",
			Name:   "Headless Tetra Surface runtime",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("release-test target for deterministic Surface runtime, " +
				"text/input, toolkit, accessibility, artifact-hash, and " +
				"validator evidence under surface-v1-linux-web"),
			Stability: ("current as a release evidence target, not as an end-user " +
				"platform claim; reports are validated by strict Surface v1 " +
				"release validators and artifact hashes"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/user/surface/surface_guide.md",
				"docs/release/surface/surface_v1_release_contract.md",
			},
		},
		{
			ID:     "ui.surface-linux-x64",
			Name:   "Linux-x64 Tetra Surface host",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("current linux-x64-release-window-v1 Surface target using " +
				"Wayland shm RGBA real-window evidence, native event pump, " +
				"text input, clipboard, IME/composition trace, toolkit, " +
				"accessibility bridge evidence, linux-app-shell-subset-v1, " +
				"electron-feature-ledger-v1 rows for app menu, lifecycle, " +
				"multi-window, clipboard, IME, accessibility bridge, scoped " +
				"crash/error reporting adapters, and blocked-pass " +
				"dialog/file picker/notification/tray/deep-link nonclaims, " +
				"plus surface-security-permission-v1 default-deny " +
				"filesystem/network policy, capability-checked IPC/process " +
				"boundaries, scoped clipboard policy, local hashed " +
				"asset/font/image safety, surface-performance-budget-v1 " +
				"local " +
				"startup/frame/memory/cache/framebuffer/binary-size/CPU-proxy" +
				" evidence, and surface-dev-workflow-v1 fast rebuild " +
				"developer workflow evidence"),
			Stability: ("current only for the proven linux-x64 real-window release " +
				"path, bounded app-shell feature ledger, scoped security " +
				"permission model, local deterministic performance budget " +
				"report, and fast rebuild developer workflow; no GTK, Qt, " +
				"platform widget, metadata sidecar playback, macOS, Windows, " +
				"unrestricted filesystem/network, remote asset fetch, " +
				"tray/notification/file-picker/dialog support, official " +
				"benchmark result, unsupported Electron speed comparison, " +
				"hot reload, Electron dev server, React Fast Refresh, or " +
				"broad Electron shell parity claim is implied"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/user/surface/surface_guide.md",
				"docs/release/surface/surface_v1_release_contract.md",
			},
		},
		{
			ID:     "ui.surface-dev-workflow-v1",
			Name:   "Tetra Surface developer fast loop",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("`tetra surface dev` emits tetra.surface.dev-workflow.v1 / " +
				"surface-dev-workflow-v1 evidence for linux-x64 build-cache " +
				"fast rebuilds, including initial build, warm-cache rebuild, " +
				"token-change, recipe-change, source-change, source " +
				"diagnostics, and artifact hashes"),
			Stability: ("current developer workflow evidence only; the loop is " +
				"documented as fast rebuild and may require process restart; " +
				"it is not a hot reload, Electron dev server, React Fast " +
				"Refresh, DOM-authored app UI reload, or full IDE server " +
				"claim"),
			Docs: []string{
				"docs/spec/policy/cli_contracts.md",
				"docs/spec/surface/surface_v1.md",
				"docs/user/surface/surface_guide.md",
				"docs/release/surface/surface_v1_release_contract.md",
				"docs/release/surface/surface_v1_release_notes.md",
			},
		},
		{
			ID:     "ui.surface-inspector-v1",
			Name:   "Tetra Surface inspector",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("tools/cmd/surface-inspector emits " +
				"tetra.surface.inspector.v1 / surface-inspector-v1 static " +
				"tool reports from validated Surface runtime reports, " +
				"exposing Block tree, Morph tokens, layout, paint, " +
				"accessibility, event routes, focus state, perf counters, " +
				"source locations, hidden-state scan results, JSON evidence, " +
				"and optional static HTML report"),
			Stability: ("current inspector evidence only; it is a static report tool " +
				"and not browser devtools, React devtools, DOM runtime UI, " +
				"hidden app state, an interactive runtime inspector, or a " +
				"substitute for target-host accessibility evidence"),
			Docs: []string{
				"docs/spec/policy/cli_contracts.md",
				"docs/spec/surface/surface_v1.md",
				"docs/spec/core/current_supported_surface.md",
				"docs/user/surface/surface_guide.md",
				"docs/release/surface/surface_v1_release_contract.md",
				"docs/release/surface/surface_v1_release_notes.md",
				"docs/release/surface/surface_v1_release_audit.md",
			},
		},
		{
			ID:     "ui.surface-project-templates-v1",
			Name:   "Tetra Surface project templates",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("`tetra new surface-app --template <kind>` generates " +
				"Block/Morph Surface app projects for command palette, " +
				"settings, dashboard, editor shell, multi-window notes, and " +
				"web-canvas starts; surface-template-smoke-v1 evidence " +
				"checks, builds, runs, inspects, visually tests, and " +
				"packages generated app paths without React, Electron, " +
				"DOM-authored app UI trees, CSS runtime, user JavaScript app " +
				"logic, core widget primitives, or platform widgets"),
			Stability: ("current onboarding template evidence for the bounded " +
				"Surface v1 Linux/web scope; templates are not a broad " +
				"Electron/React/CSS replacement claim and do not promote " +
				"Morph beyond the validated experimental authoring layer " +
				"over Block"),
			Docs: []string{
				"docs/spec/policy/cli_contracts.md",
				"docs/spec/surface/surface_v1.md",
				"docs/spec/core/current_supported_surface.md",
				"docs/user/surface/surface_guide.md",
				"docs/user/surface/surface_cookbook.md",
				"docs/user/reference/examples_index.md",
				"docs/release/surface/surface_v1_release_contract.md",
				"docs/release/surface/surface_v1_release_notes.md",
			},
		},
		{
			ID:     "ui.surface-reference-app-suite-v1",
			Name:   "Tetra Surface reference app suite",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("surface-reference-app-suite-v1 evidence covers ten polished " +
				"Block/Morph reference app shapes: command palette, settings," +
				" dashboard, editor shell, file manager/list-detail, " +
				"dialog/notification, localized form, accessibility-heavy " +
				"form, multi-window notes, and widgets-to-Morph migration " +
				"compatibility. Each app compiles, runs, uses stable Morph " +
				"recipes that resolve to Block, and records headless, " +
				"linux-x64 real-window, and wasm32-web browser-canvas visual," +
				" interaction, accessibility, performance, token/theme, " +
				"layout, and artifact-hash evidence."),
			Stability: ("current product-shape evidence for the bounded Surface v1 " +
				"Linux/web scope; this is not a broad Electron shell parity " +
				"claim, not a React runtime, not a CSS cascade runtime, not " +
				"DOM-authored app UI, and widget compatibility is limited to " +
				"the migration example."),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/spec/core/current_supported_surface.md",
				"docs/user/surface/surface_guide.md",
				"docs/user/reference/examples_index.md",
				"docs/release/surface/surface_v1_release_contract.md",
				"docs/release/surface/surface_v1_release_notes.md",
			},
		},
		{
			ID:     "ui.surface-packaging-v1",
			Name:   "Tetra Surface packaging and update story",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("surface-package-v1 evidence packages the command-palette " +
				"reference app and product-slice studio-shell flagship " +
				"source as linux-x64 and wasm32-web tar.gz artifacts, " +
				"records surface-app-package-v1 manifests, local asset " +
				"hashes, installed linux-x64 binary execution, web bundle " +
				"HTML/wasm/compiler-owned loader output, and a hash-pinned " +
				"update channel manifest"),
			Stability: ("current packaging evidence for the bounded Surface v1 " +
				"Linux/web scope; signing, notarization, automatic runtime " +
				"update, network update, Electron runtime, React runtime, " +
				"CSS cascade runtime, DOM-authored app UI tree, remote asset " +
				"fetch, and user JavaScript app logic remain explicit " +
				"nonclaims"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/spec/core/current_supported_surface.md",
				"docs/user/surface/surface_guide.md",
				"docs/release/surface/surface_v1_release_contract.md",
				"docs/release/surface/surface_v1_release_notes.md",
			},
		},
		{
			ID:     "ui.surface-crash-reporting-v1",
			Name:   "Tetra Surface crash recovery and error reporting",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("surface-crash-report-v1 evidence records bounded linux-x64 " +
				"command failure, host crash diagnostic capture, local " +
				"ring-buffer trace/log collection, redacted " +
				"tetra.surface.diagnostic.v1 artifacts, and scoped " +
				"restart/recovery evidence for the command-palette reference " +
				"app"),
			Stability: ("current diagnostic evidence for the bounded Surface v1 " +
				"Linux/web scope; diagnostics are local-only and redacted, " +
				"and validators reject user data leaks, network upload, " +
				"docs-only crash claims, Electron crash reporter dependency, " +
				"and restart claims without before/report/after evidence"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/spec/core/current_supported_surface.md",
				"docs/user/surface/surface_guide.md",
				"docs/release/surface/surface_v1_release_contract.md",
				"docs/release/surface/surface_v1_release_notes.md",
			},
		},
		{
			ID:     "ui.surface-i18n-v1",
			Name:   "Tetra Surface internationalization and localization",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("surface-i18n-v1 evidence records bounded string tables, " +
				"locale selection, fallback from uk-UA to en-US, missing-key " +
				"diagnostics, deterministic formatting hooks, localized-form " +
				"reference app execution, and RTL placeholder nonclaim " +
				"evidence"),
			Stability: ("current localization evidence for the bounded Surface v1 " +
				"Linux/web scope; it is not full ICU, full bidi shaping, RTL " +
				"production text layout, platform locale dependency, " +
				"third-party intl runtime, or a general Unicode text engine " +
				"claim"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/spec/core/current_supported_surface.md",
				"docs/user/surface/surface_guide.md",
				"docs/release/surface/surface_v1_release_contract.md",
				"docs/release/surface/surface_v1_release_notes.md",
			},
		},
		{
			ID:     "ui.surface-widget-migration-v1",
			Name:   "Tetra Surface widget migration compatibility",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("surface-widget-migration-v1 evidence keeps lib.core.widgets " +
				"supported as a Surface v1 compatibility layer, records the " +
				"exact release widget set, proves Panel/Button/TextBox " +
				"equivalence rows against Morph recipes that resolve to " +
				"Block, runs the migration reference app, and records Block " +
				"as the only core primitive"),
			Stability: ("current migration evidence for the bounded Surface v1 " +
				"Linux/web scope; it is not a future core widget primitive " +
				"promotion, not a breaking API change, not a docs-only " +
				"migration claim, and not a platform toolkit/runtime claim"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/spec/core/current_supported_surface.md",
				"docs/user/surface/surface_guide.md",
				"docs/release/surface/surface_v1_release_contract.md",
				"docs/release/surface/surface_v1_release_notes.md",
			},
		},
		{
			ID:     "ui.surface-web-wasm",
			Name:   "WASM web Tetra Surface",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("current wasm32-web-browser-canvas-release-v1 Surface target " +
				"with compiler-owned browser boot, DOM host canvas only, " +
				"browser canvas RGBA presentation/readback, browser input, " +
				"clipboard, composition, accessibility snapshot, " +
				"accessibility mirror evidence, and " +
				"tetra.surface.browser-surface.v1 report evidence"),
			Stability: ("current only for pure-Tetra apps running through the tiny " +
				"Surface Host ABI; DOM-authored app UI trees, React, user " +
				"JavaScript app logic, metadata-only UI sidecars, Node-only " +
				"promotion, and arbitrary browser widget claims are rejected " +
				"by validators"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/user/surface/surface_guide.md",
				"docs/release/surface/surface_v1_release_contract.md",
			},
		},
		{
			ID:     "ui.surface-component-model",
			Name:   "Tetra Surface component model",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("component-tree-api release subset where ordinary Tetra " +
				"structs use `lib.core.component`, helper-owned parent/child " +
				"links, stable ids, layout helpers, hit testing, focus " +
				"routing, root-to-leaf dispatch paths, and no manual " +
				"app-side tree bookkeeping"),
			Stability: ("current for the static release subset only; dynamic " +
				"trait-object child lists, witness-table component dispatch, " +
				"arbitrary reactive frameworks, and platform-native " +
				"component trees remain future work"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/user/surface/surface_guide.md",
				"docs/release/surface/surface_v1_release_contract.md",
			},
		},
		{
			ID:     "ui.surface-toolkit-v1",
			Name:   "Tetra Surface toolkit v1",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("production-widgets-v1 release subset in `lib.core.widgets`: " +
				"Text, Label, StatusText, Button, TextBox, Checkbox, Row, " +
				"Column, Panel, Stack, Scroll, and Spacer over the " +
				"ComponentTree helper API"),
			Stability: ("current for the release widget subset with owned/copy-safe " +
				"state and no magical widgets, platform widgets, DOM UI, " +
				"user JS, or demo-local widget structs; broader widget " +
				"libraries remain post-release work"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/user/surface/surface_guide.md",
				"docs/user/reference/examples_index.md",
				"docs/release/surface/surface_v1_release_notes.md",
			},
		},
		{
			ID:     "ui.surface-text-input-v1",
			Name:   "Tetra Surface text/input v1",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("production-text-input-v1 baseline covering UTF-8 byte " +
				"storage, invalid UTF-8 rejection, multiline byte storage, " +
				"caret, selection, selection clipboard transfer, clipboard " +
				"read/write, IME/composition trace, focused TextBox routing, " +
				"host-boundary copy semantics, settings/editor reference " +
				"traces, and scoped shaping-plan evidence"),
			Stability: ("current for the bounded Surface v1 text/input baseline; " +
				"full rich text, full bidi shaping, grapheme-cluster caret " +
				"movement, IDE-grade editing, arbitrary native text controls," +
				" and full Unicode editor semantics remain unsupported in " +
				"this release"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/user/surface/surface_guide.md",
				"docs/user/reference/examples_index.md",
				"docs/release/surface/surface_v1_release_notes.md",
			},
		},
		{
			ID:     "ui.surface-accessibility-v1",
			Name:   "Tetra Surface accessibility v1",
			Status: FeatureStatusCurrent,
			Since:  "surface-v1",
			Scope: ("platform-bridge-v1 accessibility for supported targets: " +
				"metadata tree exported through the Linux accessibility " +
				"bridge/probe path and wasm32-web browser accessibility " +
				"snapshot/mirror"),
			Stability: ("current for supported targets only; metadata-only reports, " +
				"DOM/ARIA claims without compiler-owned mirror evidence, " +
				"screen-reader claims, macOS/Windows accessibility, and full " +
				"AT-SPI claims remain unsupported without separate proof"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/user/surface/surface_guide.md",
				"docs/user/reference/examples_index.md",
				"docs/release/surface/surface_v1_release_notes.md",
			},
		},
		{
			ID:     "ui.surface-minimal-toolkit",
			Name:   "Tetra Surface minimal widget toolkit",
			Status: FeatureStatusExperimental,
			Scope: ("historical minimal-widgets-v1 evidence absorbed by " +
				"ui.surface-toolkit-v1; retained for backward report " +
				"references and regression evidence"),
			Stability: ("absorbed by ui.surface-toolkit-v1 and not a public current " +
				"release API; reports remain experimental historical " +
				"evidence and must not claim production toolkit support"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/user/surface/surface_guide.md",
				"docs/user/reference/examples_index.md",
			},
		},
		{
			ID:     "ui.surface-toolkit-reuse-v1",
			Name:   "Tetra Surface toolkit reuse v1",
			Status: FeatureStatusExperimental,
			Scope: ("historical toolkit-reuse-v1 multi-form evidence absorbed by " +
				"ui.surface-toolkit-v1; retained for backward report " +
				"references and regression evidence"),
			Stability: ("absorbed by ui.surface-toolkit-v1 and not a public current " +
				"release API; reports remain experimental historical " +
				"evidence and must not claim production toolkit support"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/user/surface/surface_guide.md",
				"docs/user/reference/examples_index.md",
			},
		},
		{
			ID:     "ui.surface-accessibility-metadata-tree-v1",
			Name:   "Tetra Surface accessibility metadata tree v1",
			Status: FeatureStatusExperimental,
			Scope: ("internal layer under ui.surface-accessibility-v1; retained " +
				"as historical metadata-tree evidence for roles, labels, " +
				"values, states, bounds, relationships, focus order, reading " +
				"order, actions, snapshots, and status updates"),
			Stability: ("internal layer under ui.surface-accessibility-v1 and not a " +
				"public production accessibility claim by itself; " +
				"metadata-only evidence must not claim platform " +
				"accessibility, DOM/ARIA, screen-reader, or full AT-SPI " +
				"support"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/user/surface/surface_guide.md",
				"docs/user/reference/examples_index.md",
			},
		},
		{
			ID:     "ui.surface-macos-x64",
			Name:   "macOS Surface host",
			Status: FeatureStatusUnsupported,
			Scope: ("unsupported for Surface v1; no production target evidence " +
				"exists for macOS real-window Surface; " +
				"tetra.surface.target-host-status.v1 records UNSUPPORTED " +
				"nonclaim evidence"),
			Stability: ("no production target evidence in surface-v1-linux-web and " +
				"no current macOS Surface support claim; build-only macOS " +
				"artifacts do not promote Surface runtime support"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/release/surface/surface_v1_release_contract.md",
			},
		},
		{
			ID:     "ui.surface-windows-x64",
			Name:   "Windows Surface host",
			Status: FeatureStatusUnsupported,
			Scope: ("unsupported for Surface v1; no production target evidence " +
				"exists for Windows real-window Surface; " +
				"tetra.surface.target-host-status.v1 records UNSUPPORTED " +
				"nonclaim evidence"),
			Stability: ("no production target evidence in surface-v1-linux-web and " +
				"no current Windows Surface support claim; build-only " +
				"Windows artifacts do not promote Surface runtime support"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/release/surface/surface_v1_release_contract.md",
			},
		},
		{
			ID:     "ui.surface-wasm32-wasi",
			Name:   "WASI Surface UI runtime",
			Status: FeatureStatusUnsupported,
			Scope: ("unsupported for Surface v1; wasm32-wasi has no Surface UI " +
				"runtime production target evidence"),
			Stability: ("no production target evidence in surface-v1-linux-web and " +
				"no current wasm32-wasi Surface UI support claim"),
			Docs: []string{
				"docs/spec/surface/surface_v1.md",
				"docs/release/surface/surface_v1_release_contract.md",
			},
		},
		{
			ID:     "wasm.runtime-execution",
			Name:   "WASM runtime execution",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("production WASI runner execution through wasmtime or the " +
				"Node WASI fallback plus browser-backed wasm32-web execution " +
				"through discovered Chromium-compatible runners"),
			Stability: ("current runner-backed WASM runtime support with explicit " +
				"missing-runner diagnostics; browser UI command dispatch " +
				"evidence remains separated from Linux-x64 native UI runtime " +
				"evidence"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/backend/wasm_backend_plan.md",
				"docs/user/surface/wasm_ui_guide.md",
			},
		},
		{
			ID:        "language.full-v1-guarantees",
			Name:      "Full v1.0 language guarantees",
			Status:    FeatureStatusPlanned,
			Scope:     "complete v1.0 release contract after mandatory release-gate evidence",
			Stability: "future label while repository remains on the v0.4.0 profile",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/flow/v1_scope.md",
			},
		},
		{
			ID:     "language.full-first-class-callables",
			Name:   "Full first-class callable/function-value semantics",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("production first-class callable/function-value semantics " +
				"for safe by-value captures: the bounded fnptr fast path " +
				"remains for up to eight environment slots, and larger " +
				"immutable Int/Bool/String/simple-aggregate captures use a " +
				"fixed 4-slot callable handle for local storage, mutable " +
				"local reassignment, returns, same-module global snapshots, " +
				"struct fields, enum payloads, synchronous callback " +
				"arguments, cross-module returned values, aliases, generated " +
				".t4i function-typed parameter local-alias return metadata, " +
				"and generated .t4i metadata"),
			Stability: ("current v0.4.0 safe-capture model with explicit escape " +
				"classification and stable JSON diagnostics for mutable " +
				"by-reference captures including callable mutable-capture " +
				"global-escape and callable mutable-capture heap-escape, " +
				"callable pointer/resource capture escape, function-typed " +
				"storage/return unsupported capture rejection, captured " +
				"callable/function-typed parameter global-storage escape, " +
				"unsupported function-value escape outside the fnptr ABI, " +
				"unsupported function-value call, capturing closure raw-ptr " +
				"escape, captured closure explicit type-arg rejection, " +
				"function-typed explicit type-arg rejection, generic closure " +
				"capture and generic callback-closure capture rejection, " +
				"generic closure pointer/direct-call rejection, " +
				"thread-boundary callable escape, imported mutable " +
				"function-typed global boundary, imported mutable " +
				"global-data ABI gaps, and unsupported dynamic/generic " +
				"callable movement"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/policy/v1_feature_status.md",
				"docs/release/v0_4/v0_4_0_callable_evidence_map.md",
			},
		},
		{
			ID:     "eco.distributed-network",
			Name:   "Distributed EcoNet and production publishing",
			Status: FeatureStatusPostV1,
			Scope: ("distributed EcoNet, production TetraHub publishing, global " +
				"trust scoring, proof-carrying capsules"),
			Stability: "deferred post-v1 unless explicitly promoted",
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/release/policy/post_v1_promotion_checklist.md",
			},
		},
		{
			ID:     "actors.distributed-runtime",
			Name:   "Distributed actor runtime for Linux x64",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("production Linux-x64 distributed actor runtime path with " +
				"actornet loopback TCP broker, distributed node identity, " +
				"remote actor handles, network mailbox send/receive for i32, " +
				"tagged, and typed frames, missing-node failure/status " +
				"propagation, compatibility with existing task cancel/join " +
				"handles, and scoped actor runtime foundation gate evidence " +
				"through tetra.actor.production_foundation.v1"),
			Stability: ("current Linux-x64 runtime/lowering slice with executable " +
				"tetra.actors.distributed-runtime.v1 smoke evidence, " +
				"tetra.actor.production_foundation.v1 gate evidence from " +
				"actor-runtime-foundation-linux-x64-gate.sh, and strict " +
				"validator rejection for transport-only or fake reports; " +
				"non-Linux-x64 targets, non-Linux distributed runtime, " +
				"distributed zero-copy transfer, cluster membership, " +
				"reconnect/retry production, formal race proof, " +
				"multi-threaded scheduling, and broader " +
				"structured-concurrency guarantees remain outside this claim"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/runtime/actors.md",
				"docs/user/platform/async_actors_guide.md",
				"docs/design/actor_region_transfer.md",
				"docs/audits/runtime/actors/actor-runtime-production-boundary-v1.md",
				"docs/checklists/actors_linux_smoke.md",
				"docs/checklists/actors_platform_smoke.md",
			},
		},
		{
			ID:     "ui.native-runtime",
			Name:   "Linux-x64 native UI runtime",
			Status: FeatureStatusCurrent,
			Since:  "v0.4.0",
			Scope: ("production Linux-x64 native UI runtime path that loads the " +
				"checked tetra.ui.v0.4.0/native-shell widget tree, creates " +
				"native runtime widget instances with IDs, hierarchy, bounds," +
				" text/value, enabled, and visible state, dispatches " +
				"click/activate events to lowered command operations, " +
				"propagates state and widget updates, records lifecycle " +
				"close, and reports negative invalid widget, malformed " +
				"metadata, unsupported event, and command failure cases"),
			Stability: ("current Linux-x64 deterministic native runtime slice with " +
				"executable tetra.ui.native-runtime.v1 smoke evidence and " +
				"strict validator rejection for metadata-only, web-only, " +
				"native-shell sidecar-only, fake, mock, or placeholder " +
				"evidence; macOS/Windows, GTK/Qt/OS widget backend claims, " +
				"platform accessibility integration, and broad " +
				"input/change/focus behavior remain outside this claim until " +
				"host-native reports exist"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/ui/ui_v1.md",
				"docs/spec/ui/ui_v0.4.0.md",
				"docs/user/surface/wasm_ui_guide.md",
			},
		},
		{
			ID:     "ui.platform-runtime",
			Name:   "Cross-platform UI runtime promotion gate",
			Status: FeatureStatusExperimental,
			Since:  "v0.4.0",
			Scope: ("tetra.ui.platform-runtime.v1 full-platform UI runtime " +
				"promotion gate for Linux, Windows, macOS, and Web evidence; " +
				"Windows/macOS require real Windows/macOS target-host " +
				"reports before they can count as production UI runtime " +
				"targets"),
			Stability: ("not production until the full-platform UI runtime promotion " +
				"gate passes with real Windows/macOS target-host reports and " +
				"rejects metadata-only, runtime-less, build-only, " +
				"sidecar-only, fake/mock/placeholder, and startup_failure " +
				"evidence as blockers rather than platform runtime proof"),
			Docs: []string{
				"docs/spec/core/current_supported_surface.md",
				"docs/spec/ui/ui_v1.md",
				"docs/user/surface/wasm_ui_guide.md",
			},
		},
	}
	out := make([]FeatureInfo, len(features))
	copy(out, features)
	for i := range out {
		out[i].Docs = append([]string(nil), features[i].Docs...)
		if out[i].ID == "compiler.verified-track" {
			out[i].Scope += ("; P17.3 simple map over []i32 executable evidence is " +
				"limited to proof-tagged in-place add-constant-1 linux-x64 " +
				"native SIMD through vector-i32x4-map-add-const-plan, single " +
				"mutable slice in-place noalias-not-required evidence, safe " +
				"unaligned i32x4 map load/store, scalar-i32-map fallback, " +
				"and stack-fallback translation/differential validation; " +
				"P17.3 memset/memcpy helper executable evidence is limited " +
				"to proof-tagged zero-fill memset_zero_u8 through " +
				"vector-u8x16-memset-zero-plan plus memcpy helper via copy []" +
				"u8 evidence; no checked/no-proof map, broader map-shape " +
				"vectorization, arbitrary non-zero memset, overlapping " +
				"memcpy, checked/no-proof helper, libc/runtime helper " +
				"lowering, or performance claim is made")
			out[i].Scope += ("; typed actor ownership transfer v1 records " +
				"tetra.actors.ownership_transfer.v1 rows for borrowed-view " +
				"copy boundaries, owned-region move, sender use-after-move " +
				"diagnostics, receiver ownership evidence, explicit copy " +
				"fallback, unsafe-send contract model evidence, semantics " +
				"transfer checker, PLIR moved facts with FactMoved and " +
				"OpActorSend for direct core.send_typed ownership transfers, " +
				"runtime mailbox representation, actor-transfer reports, " +
				"stress diagnostics, fake distributed zero-copy rejection, " +
				"and fake runtime-behavior-change rejection; no distributed " +
				"pointer or region zero-copy, safe typed actor raw pointer " +
				"payload, actor scheduler promotion, or production actor " +
				"runtime claim is made")
			out[i].Scope += ("; per-core scheduler v1 records " +
				"tetra.parallel.per_core_scheduler.v1 rows for per-core " +
				"queues, work stealing, bounded typed mailboxes, " +
				"backpressure, timers sleep/wake, structured task groups, " +
				"cancellation checkpoints, actor ping-pong, fanout/fanin, " +
				"task group cancel, backpressure overflow, mailbox fairness " +
				"with FIFO receive, stress evidence, race detector where " +
				"applicable, fake full production actor-runtime rejection, " +
				"fake runtime-behavior-change rejection, and fake all-target " +
				"race-detector rejection; no non-Linux distributed actor " +
				"runtime target, full production actor runtime, full " +
				"race-safety proof, scheduler performance claim, public " +
				"runtime mode, or safe-semantics flag change is claimed")
			out[i].Scope += ("; stable generic collections v1 records " +
				"tetra.stdlib.generic_collections.v1 rows for stable " +
				"Tetra-source Vec<T> and HashMap<K,V> caller-owned slice " +
				"views, generic value representation through genericTypeName " +
				"and mangleGenericName, generic-struct parameter inference " +
				"through bindGenericNamedTypeArgs, monomorphized " +
				"vec_from_slice<T> and hash_map_from_slices<K,V> operations, " +
				"common hash_map_get_i32_i32_or and hash_map_get_u8_i32_or " +
				"specializations, allocation-plan report linkage through " +
				"core.make_* caller allocations, and a checked " +
				"truth-bench-harness dry-run artifact for scope " +
				"p19.1_generic_collections with hash table Tetra/C++/Rust " +
				"equivalents, report path " +
				"reports/stable-generic-collections-v1/benchmarks/generic-col" +
				"lections-hash-table-report.json, matching " +
				"algorithm_id/input metadata, and Tetra " +
				"proof/allocation/bounds/performance report artifacts; no " +
				"allocator-backed production Vec<T>/HashMap<K,V> runtime, " +
				"generic hashing/equality protocol, C++/Rust parity, broad " +
				"production stdlib, hidden runtime allocator, measured speed " +
				"comparison, or official benchmark result is claimed")
			out[i].Scope += ("; production HTTP/JSON stack v1 foundation records " +
				"tetra.stdlib.http_json.production_stack.v1 rows for " +
				"HTTP/1.1 request-head parsing, pipelined request heads, " +
				"headers/body/keep-alive metadata, zero-heap request-view " +
				"evidence, JSON parse/stringify, response building, internal " +
				"per-server UTC-second Date cache helper evidence through " +
				"HTTPDateCache and FormatWithReport, Linux writev/sendfile " +
				"helper evidence through netrt.Writev and netrt.Sendfile, " +
				"and a checked truth-bench-harness dry-run artifact for " +
				"scope p19.2_http_json_source_first with Tetra-only HTTP " +
				"plaintext and HTTP JSON rows, report path " +
				"reports/production-http-json-v1/benchmarks/http-json-source-" +
				"first-report.json, matching algorithm_id/input metadata, " +
				"and Tetra proof/allocation/bounds/P19.2 coverage artifacts; " +
				"webrt.flush scatter/gather integration, HTTP static-file " +
				"sendfile path, and non-Linux writev/sendfile parity remain " +
				"documented boundaries, and no full production web stack, " +
				"official TechEmpower result, production PostgreSQL stack, " +
				"P20 performance matrix, C++/Rust parity, measured speed " +
				"comparison, source-level cached-date API, cross-worker Date " +
				"cache, zero-copy production file-serving, or runtime " +
				"behavior change is claimed")
			out[i].Scope += ("; production PostgreSQL driver/pool v1 closure records " +
				"tetra.stdlib.postgresql.production_driver.v1 rows for " +
				"startup/SCRAM, prepared statements, binary int4 helpers, " +
				"pooling/backpressure, borrowed DataRow decode, local DB " +
				"single query, DB multiple queries, DB updates, DB fortunes " +
				"endpoint workloads, a checked truth-bench-harness dry-run " +
				"artifact for scope p19.3_postgres_source_first with " +
				"Tetra-only DB rows, report path " +
				"reports/production-postgres-v1/benchmarks/postgres-source-fi" +
				"rst-report.json, matching algorithm_id/input metadata, and " +
				"Tetra proof/allocation/bounds/P19.3 coverage artifacts, " +
				"plus live local SCRAM benchmark honesty evidence through " +
				"validate-techempower-report on " +
				"docs/benchmarks/techempower_scram_single_query_local_report." +
				"json, " +
				"docs/benchmarks/techempower_scram_single_query_matrix_local_" +
				"report.json, and " +
				"docs/benchmarks/techempower_scram_endpoint_matrix_local_repo" +
				"rt.json; no official TechEmpower result, production " +
				"database benchmark, P20 performance matrix, C++/Rust parity," +
				" external production database deployment, full source-level " +
				"PostgreSQL driver API, measured speed comparison, or " +
				"runtime behavior change is claimed")
			out[i].Scope += ("; benchmark matrix hardening v1 records the " +
				"p20.0_benchmark_matrix truth-bench-harness contract with 68 " +
				"checked dry-run rows for 17 master-plan categories across " +
				"Tetra, C clang -O3, C++ clang++ -O3, and Rust rustc -C " +
				"opt-level=3, including matching algorithm_id/input metadata," +
				" raw output artifacts on every row, Tetra " +
				"proof/allocation/bounds/performance artifacts on every " +
				"Tetra row, report path " +
				"reports/benchmark-matrix-hardening-v1/benchmarks/p20-matrix-" +
				"hardening-report.json, and row target CPU consistency with " +
				"host target CPU; no measured speed comparison, C++/Rust " +
				"parity, official benchmark result, official TechEmpower " +
				"result, production database benchmark, P20.1 blocker " +
				"completeness, P20.2 claim-tier promotion, throughput " +
				"advantage, latency advantage, startup-time advantage, " +
				"binary-size advantage, or compile-time advantage is claimed")
			out[i].Scope += ("; performance blocker reports v1 records compiler " +
				".perf.json schema_version 3 for P20.1 with matrix scope " +
				"p20.0_benchmark_matrix, report path " +
				"reports/benchmark-matrix-hardening-v1/benchmarks/artifacts/p" +
				"20-matrix-hardening.perf.json, " +
				"ValidatePerformanceBlockerReport, the exact blocker reasons " +
				"left bounds check: missing dominance, heap allocation: " +
				"escapes through return, heap allocation: unknown call, heap " +
				"allocation: local call boundary heap fallback, not " +
				"vectorized: no noalias proof, not inlined: code-size budget," +
				" register spill: live range pressure, stack fallback: " +
				"unsupported aggregate return, and actor copy: borrowed data " +
				"crosses boundary, plus 17 P20.0 Tetra benchmark explanation " +
				"rows from integer_loops_tetra through compile_time_tetra; " +
				"no measured speed comparison, C++/Rust parity, official " +
				"benchmark result, official TechEmpower result, P20.2 " +
				"claim-tier promotion, optimizer behavior change, runtime " +
				"behavior change, blocker removal, throughput advantage, or " +
				"latency advantage is claimed")
			out[i].Scope += ("; claim tiers v1 records tetra.performance.claim_tiers.v1 " +
				"scope p20.2_claim_tiers with exact Tier 0 local smoke only, " +
				"Tier 1 local benchmark evidence, Tier 2 reproducible " +
				"cross-machine benchmark, Tier 3 independent reproduced " +
				"benchmark, and Tier 4 official upstream benchmark " +
				"submission policy rows, checked artifact " +
				"reports/claim-tiers-v1/claim-tier-report.json, current " +
				"P20.0/P20.1 public claim p20_current_local_smoke_only at " +
				"tier0_local_smoke_only, required evidence classes " +
				"local_smoke, local_benchmark, cross_machine_reproduction, " +
				"independent_reproduction, and official_upstream_submission, " +
				"and validator rejection for fake local benchmark evidence, " +
				"cross-machine benchmark, independent reproduced benchmark, " +
				"official upstream benchmark submission, official " +
				"TechEmpower, measured speed, throughput advantage, latency " +
				"advantage, and C++/Rust parity wording unless explicit " +
				"non-claims or matching tier evidence exist; current " +
				"P20.0/P20.1 evidence remains Tier 0 only")
			out[i].Scope += ("; specialization machine-code evidence v1 records " +
				"tetra.optimizer.specialization_machine_code.v1 scope " +
				"p21.2_specialization_v1_v2 rows for generics, " +
				"protocol/static conformance, extension methods, enum match " +
				"known cases, optionals, and collections; " +
				"BuildP21SpecializationMachineCodeWitness uses " +
				"inline-small-pure plus machine.ScalarIntFunctionFromStackIR " +
				"to prove a known direct helper call is present before " +
				"optimization, absent from optimized Stack IR, and absent as " +
				"OpCall from verified scalar Machine IR, with translation " +
				"validation; rows connect P17.2 monomorphized generic " +
				"identity/wrapper, statically checked protocol impl direct " +
				"calls, statically resolved extension method direct calls, " +
				"SCCP known enum discriminator branch folding, proven-some " +
				"optional presence branch folding, and P19.1 caller-owned " +
				"Vec<T>/HashMap<K,V> monomorphized helper evidence; " +
				"validator rejects placeholder evidence, missing target rows," +
				" fake broad specialization, fake dynamic dispatch, fake " +
				"runtime generic values, fake allocator-backed generic " +
				"collections, fake layout/ABI freedom, fake performance, and " +
				"fake safe-semantics changes")
			out[i].Scope += ("; translation validation v2 records " +
				"tetra.translation.validation.v2 scope " +
				"p23.0_translation_validation_v2 rows for registered " +
				"optimizer pass coverage, symbolic scalar equivalence, " +
				"supported i32 slice memory equivalence, bounds proof " +
				"preservation, allocation plan preservation, and " +
				"machine-checkable sha256 before/after optimization metadata;" +
				" witnesses run opt.NewManager over opt.RegisteredPasses, " +
				"validation.ValidateTranslation scalar and proof cases, " +
				"differential backend matrix loop/call/slice samples, " +
				"validation.ValidateAllocationLowering, and " +
				"BuildOptimizationValidationMetadata; validator rejects " +
				"missing rows/witnesses, placeholders, incomplete " +
				"registered-pass coverage, missing " +
				"scalar/memory/loop/call/proof/allocation/hash evidence, " +
				"fake full formal proof, fake exhaustive optimizer " +
				"completeness, fake broad memory or loop proof claims, fake " +
				"performance, fake runtime behavior change, and fake " +
				"safe-semantics changes")
			out[i].Scope += ("; fuzz/property/differential expansion v1 records " +
				"tetra.fuzz.property.differential.v1 scope " +
				"p23.1_fuzz_property_differential rows for parser/checker " +
				"generated programs, PLIR/lowering verifier pipeline, " +
				"backend differential matrix expansion, native backend " +
				"boundary, runtime allocator properties, actor transfer " +
				"stress boundary, fuzz nightly summary gate, and reducer " +
				"failure artifacts; witnesses run compiler.Parse, " +
				"compiler.Check, BuildPLIR, Lower, VerifyIRProgram, " +
				"differential.CheckBackendMatrix with deterministic " +
				"randomized samples, host-supported Linux x64 native backend " +
				"lane or explicit unavailable boundary, " +
				"runtimeabi.AlignRegionBytes valid/invalid allocator " +
				"properties, actorsafety.TypedActorOwnershipTransferCoverage " +
				"stress diagnostics and PLIR moved facts, " +
				"fuzz-nightly/validate-fuzz-summary artifact contract, and " +
				"reduced_to_single_sample mismatch reproducer; validator " +
				"rejects missing rows/witnesses, placeholders, missing " +
				"generated parser/checker cases, missing PLIR/lowering " +
				"verifier cases, missing backend randomized samples, missing " +
				"reducer evidence, missing native-host sample or explicit " +
				"non-host boundary, missing runtime allocator property " +
				"evidence, missing actor-transfer stress diagnostics, " +
				"missing fuzz summary artifacts or nightly boundary, fake " +
				"full program correctness, fake exhaustive fuzzing, fake " +
				"full native differential, fake performance, fake runtime " +
				"behavior change, and fake safe-semantics changes")
			out[i].Scope += ("; formal core v1 records tetra.formal_core.v1 scope " +
				"p23.2_formal_core_v1 rows for values, borrows and " +
				"owned/copy, provenance and regions, bounds proof id " +
				"semantics, allocation length contract, allocation intent " +
				"lowering, raw pointer bounds metadata, and " +
				"check-elimination validity; witnesses run " +
				"formalcore.ValidateSpec, differential.CheckBackendMatrix, " +
				"compiler.Parse, compiler.Check, BuildPLIR, " +
				"plir.VerifyProgram, validation.CheckBoundsProofsWithPLIR, " +
				"allocplan.FromPLIR, validation.ValidateAllocationLowering, " +
				"runtimeabi.NewRawAllocationBounds, " +
				"runtimeabi.DeriveRawPointerBounds, and " +
				"runtimeabi.RawSliceBoundsFromParts; validator rejects " +
				"missing rows/witnesses, placeholders, missing formal spec " +
				"validation, missing value samples, missing borrow/copy or " +
				"provenance/regions facts, missing bounds proof id or " +
				"check-elimination evidence, missing allocation length " +
				"contract evidence, missing allocation-intent lowering " +
				"evidence, missing raw pointer bounds metadata evidence, " +
				"fake full formal proof, fake broad language proof, fake " +
				"unsafe policy change, fake runtime behavior change, fake " +
				"safe-semantics changes, and fake performance")
			out[i].Scope += ("; self-hosting gate v1 records tetra.self_hosting.gate.v1 " +
				"scope p23.3_self_hosting_gate rows for self-host subset " +
				"definition, small compiler component compile boundary, Go " +
				"compiler output vs Tetra-compiled output comparison " +
				"boundary, register backend stability, optimizer validation " +
				"maturity, allocator/runtime stability, stdlib sufficiency, " +
				"deterministic bootstrap chain, cross-platform bootstrap " +
				"story, and no self-hosting claim; witnesses run " +
				"selfhostgate.Evaluate, differential.CheckBackendMatrix, " +
				"BuildP23TranslationValidationV2, " +
				"runtimeabi.RuntimeAllocationContracts, " +
				"runtimeabi.RuntimeRegionAllocatorConfig, " +
				"runtimeabi.RuntimePerCoreSmallHeapABI, and " +
				"stdlibrt.RegionAwareStdlibCoverage; current report requires " +
				"SelfHostingClaimed=false and GateDecision.Allowed=false, " +
				"records missing small compiler component, Go-vs-Tetra " +
				"output comparison, deterministic bootstrap chain, and " +
				"cross-platform bootstrap story blockers, and validator " +
				"rejects missing rows/witnesses, placeholders, weak compiler " +
				"subset/backend/optimizer/allocator/runtime/stdlib evidence, " +
				"fake self-hosting claim, fake small compiler component, " +
				"fake output comparison, fake deterministic bootstrap, fake " +
				"cross-platform bootstrap, fake runtime behavior change, " +
				"fake safe-semantics changes, and fake performance")
			out[i].Scope += ("; security review gate v1 records " +
				"tetra.security.review_gate.v1 scope " +
				"p24.0_security_review_gate rows for unsafe API surface, " +
				"capability surface, memory allocator, network runtime, " +
				"actor runtime, DB protocol, package/Eco system, build " +
				"scripts, supply chain, and required artifact set; artifacts " +
				"are docs/audits/security/security-review.md, " +
				"docs/audits/security/threat-model.md, " +
				"docs/audits/security/unsafe-surface-map.md, and " +
				"docs/audits/security/capability-surface-map.md; witnesses " +
				"run runtimeabi.RuntimeAllocationContracts, " +
				"runtimeabi.RuntimeRawPointerBoundsABI, " +
				"netrt.IOReactorCoverage, " +
				"actorsrt.ActorRuntimeProductionBoundaryAudit, " +
				"pgrt.ProductionPostgresCoverage, Eco validator path checks, " +
				"release security-review script checks, and artifact " +
				"presence checks; validator rejects missing rows/witnesses, " +
				"weak artifacts, fake security certification, fake external " +
				"penetration test, fake CVE-free status, fake release " +
				"security signoff, fake runtime behavior change, fake " +
				"safe-semantics changes, and fake performance")
			out[i].Scope += ("; runtime hardening v1 records tetra.runtime.hardening.v1 " +
				"scope p24.1_runtime_hardening rows for deterministic traps, " +
				"OOM policy, stack overflow guard boundary, integer overflow " +
				"semantics audit, allocator corruption detection " +
				"instrumentation, region double-free/use-after-free " +
				"instrumentation, actor mailbox overflow policy, and network " +
				"parser limits; witnesses run " +
				"runtimeabi.RuntimeAllocationContracts, " +
				"runtimeabi.RuntimeRegionAllocatorConfig, " +
				"runtimeabi.RuntimePerCoreSmallHeapABI, " +
				"runtimeabi.NewPerCoreSmallHeapAllocator, " +
				"parallelrt.NewTypedMailbox, " +
				"actorsrt.ActorRuntimeProductionBoundaryAudit, " +
				"httprt.ParseRequest, httprt.ParseRequestView, " +
				"pgrt.ReadFrame, backend trap/stack-depth file checks, and " +
				"optimizer overflow-semantics file checks; validator rejects " +
				"missing rows/witnesses, placeholders, missing " +
				"runtime-hardening artifacts, fake full runtime-hardening " +
				"proof, fake full stack-overflow protection, fake OOM " +
				"recovery, fake full allocator-corruption detection, fake " +
				"production actor-mailbox promotion, fake runtime behavior " +
				"change, fake safe-semantics changes, and fake performance")
			out[i].Scope += ("; compatibility/stability v1 records " +
				"tetra.compatibility.stability.v1 scope " +
				"p24.2_compatibility_stability rows for stable diagnostic " +
				"codes, versioned report schemas, manifest compatibility " +
				"checks, breaking-change migration guide, and deprecation " +
				"policy; witnesses read DiagnosticCodeRegistry, " +
				"validate-diagnostic, P21-P24 schema constants, " +
				"validate-manifest, docs/generated/manifest.json, " +
				"docs/spec/policy/api_diff_policy.md, " +
				"docs/release/policy/breaking-change-migration-guide.md, " +
				"docs/release/policy/deprecation_policy.md, " +
				"docs/release/v1_0/v1_0_x_maintenance_policy.md, and " +
				"docs/spec/standard_library/stdlib_naming_versioning.md; " +
				"validator rejects missing rows/witnesses, placeholders, " +
				"missing compatibility-stability artifacts, fake full " +
				"backward compatibility, fake frozen diagnostic messages, " +
				"fake automatic migration, fake manifest/runtime ABI " +
				"stability, fake breaking change without migration guide, " +
				"fake removal without deprecation, fake runtime behavior " +
				"change, fake safe-semantics changes, and fake performance")
			out[i].Docs = append(
				out[i].Docs,
				"docs/audits/compiler/language/stable-generic-collections-v1.md",
			)
			out[i].Docs = append(
				out[i].Docs,
				"docs/audits/runtime/services/production-http-json-stack-v1.md",
			)
			out[i].Docs = append(
				out[i].Docs,
				"docs/audits/runtime/services/production-postgres-driver-pool-v1.md",
			)
			out[i].Docs = append(
				out[i].Docs,
				"docs/audits/performance/benchmark-matrix-hardening-v1.md",
			)
			out[i].Docs = append(
				out[i].Docs,
				"docs/audits/performance/performance-blocker-reports-v1.md",
			)
			out[i].Docs = append(out[i].Docs, "docs/audits/performance/claim-tiers-v1.md")
			out[i].Docs = append(
				out[i].Docs,
				"docs/audits/compiler/optimizer/specialization-machine-code-v1.md",
			)
			out[i].Docs = append(
				out[i].Docs,
				"docs/audits/compiler/backend/translation-validation-v2.md",
			)
			out[i].Docs = append(
				out[i].Docs,
				"docs/audits/compiler/safety/fuzz-property-differential-v1.md",
			)
			out[i].Docs = append(out[i].Docs, "docs/audits/compiler/language/formal-core-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/compiler/safety/self-hosting-gate-v1.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/security/security-review.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/security/threat-model.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/security/unsafe-surface-map.md")
			out[i].Docs = append(out[i].Docs, "docs/audits/security/capability-surface-map.md")
			out[i].Docs = append(
				out[i].Docs,
				"docs/plans/2026-06-03/governance-p23-p24/2026-06-03-p24.0-security-review-gate-design.md",
			)
			out[i].Docs = append(
				out[i].Docs,
				"docs/audits/runtime/services/runtime-hardening-v1.md",
			)
			out[i].Docs = append(
				out[i].Docs,
				"docs/plans/2026-06-03/governance-p23-p24/2026-06-03-p24.1-runtime-hardening-design.md",
			)
			out[i].Docs = append(out[i].Docs, "docs/audits/security/compatibility-stability-v1.md")
			out[i].Docs = append(
				out[i].Docs,
				"docs/plans/2026-06-03/governance-p23-p24/2026-06-03-p24.2-compatibility-stability-design.md",
			)
			out[i].Docs = append(
				out[i].Docs,
				"docs/release/policy/breaking-change-migration-guide.md",
			)
			out[i].Docs = append(out[i].Docs, "docs/release/policy/deprecation_policy.md")
			out[i].Docs = append(out[i].Docs, "docs/benchmarks/truth_benchmark_harness.md")
		}
		if out[i].ID == "stdlib.core-current" {
			out[i].Scope += ("; stable generic collection source views expose " +
				"lib.core.collections.Vec<T> and HashMap<K,V> over " +
				"caller-owned slices, generic " +
				"vec_from_slice<T>/vec_len<T>/vec_get_or<T>/hash_map_from_sli" +
				"ces<K,V>/hash_map_len<K,V> helpers, and common " +
				"hash_map_get_i32_i32_or plus hash_map_get_u8_i32_or lookup " +
				"specializations")
			out[i].Scope += ("; P19.2 HTTP/JSON source-first evidence covers " +
				"lib.core.http request-head framing, pipelined local buffers," +
				" plaintext/JSON response byte-buffer helpers, lib.core.json " +
				"message-object writers, internal borrowed HTTP/JSON " +
				"request-region coverage, internal per-server UTC-second " +
				"Date cache evidence, and Linux netrt.Writev/netrt.Sendfile " +
				"helper evidence through " +
				"tetra.stdlib.http_json.production_stack.v1")
			out[i].Scope += ("; P19.3 PostgreSQL source-first and local SCRAM evidence " +
				"covers lib.core.postgres source rows for DB single query, " +
				"DB multiple queries, DB updates, and DB fortunes plus " +
				"internal runtime startup/SCRAM, prepared statements, binary " +
				"int4 helpers, pooling/backpressure, borrowed DataRow decode," +
				" and checked local SCRAM benchmark reports through " +
				"tetra.stdlib.postgresql.production_driver.v1, " +
				"p19.3_postgres_source_first, and validate-techempower-report")
			out[i].Stability += ("; generic collection views are source-level and " +
				"caller-owned, with no hidden allocator, resizing, generic " +
				"hashing/equality protocol, production runtime map/vector " +
				"claim, C++/Rust parity, or official benchmark result")
			out[i].Stability += ("; HTTP/JSON P19.2 evidence is source-first and local " +
				"dry-run only, with no production HTTP server promotion, " +
				"source-level cached-date API, cross-worker Date cache, " +
				"webrt.flush scatter/gather integration, HTTP static-file " +
				"sendfile path, zero-copy production file-serving, P20 " +
				"performance matrix, C++/Rust parity, or official " +
				"TechEmpower result")
			out[i].Stability += ("; PostgreSQL P19.3 evidence is source-first plus checked " +
				"local SCRAM evidence only, with no full source-level " +
				"PostgreSQL driver API, external production database " +
				"deployment, production database benchmark, P20 performance " +
				"matrix, C++/Rust parity, official TechEmpower result, " +
				"measured speed comparison, or runtime behavior change")
			out[i].Docs = append(
				out[i].Docs,
				"docs/audits/compiler/language/stable-generic-collections-v1.md",
			)
			out[i].Docs = append(
				out[i].Docs,
				"docs/audits/runtime/services/production-http-json-stack-v1.md",
			)
			out[i].Docs = append(
				out[i].Docs,
				"docs/audits/runtime/services/production-postgres-driver-pool-v1.md",
			)
		}
	}
	return out
}

// ---- format.go ----

// FormatSource returns canonical Flow formatting for the supported v1 surface.
func FormatSource(src []byte, filename string) ([]byte, error) {
	comments, err := collectLineComments(src, filename)
	if err != nil {
		return nil, err
	}
	file, err := frontend.ParseFile(stripStandaloneBlockComments(src), filename)
	if err != nil {
		return nil, err
	}
	var p sourcePrinter
	p.file(file)
	return applyLineComments([]byte(p.b.String()), comments), nil
}

func stripStandaloneBlockComments(src []byte) []byte {
	lines := strings.Split(string(src), "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		commentAt, block := commentStart(line)
		if commentAt >= 0 && block && strings.TrimSpace(line[:commentAt]) == "" {
			out = append(out, "")
			commentLine := line[commentAt:]
			for strings.Index(commentLine, "*/") < 0 && i+1 < len(lines) {
				i++
				commentLine = strings.TrimRight(lines[i], "\r")
				out = append(out, "")
			}
			continue
		}
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\n"))
}

type lineComments struct {
	before   map[int][]string
	trailing []string
}

func collectLineComments(src []byte, filename string) (lineComments, error) {
	out := lineComments{before: make(map[int][]string)}
	var pending []string
	codeLine := 0
	lines := strings.Split(string(src), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		commentAt, block := commentStart(line)
		if commentAt >= 0 {
			if strings.TrimSpace(line[:commentAt]) != "" {
				return lineComments{}, &frontend.DiagnosticError{Info: frontend.Diagnostic{
					Code: DiagnosticCodeFormatter,
					Message: ("inline comments are not supported by tetra fmt for the v1.0 " +
						"profile; move the comment to its own line or format manually"),
					File:     filename,
					Line:     i + 1,
					Column:   commentAt + 1,
					Severity: "error",
					Hint:     "Move the comment to its own line before running tetra fmt.",
				}}
			}
			if !block {
				pending = append(pending, strings.TrimSpace(line[commentAt:]))
				continue
			}

			commentLine := line[commentAt:]
			for {
				end := strings.Index(commentLine, "*/")
				if end >= 0 {
					pending = append(pending, strings.TrimSpace(commentLine[:end+2]))
					if strings.TrimSpace(commentLine[end+2:]) != "" {
						col := commentAt + end + 3
						return lineComments{}, &frontend.DiagnosticError{Info: frontend.Diagnostic{
							Code: DiagnosticCodeFormatter,
							Message: ("inline comments are not supported by tetra fmt for the v1.0 " +
								"profile; move the comment to its own line or format manually"),
							File:     filename,
							Line:     i + 1,
							Column:   col,
							Severity: "error",
							Hint:     "Move the comment to its own line before running tetra fmt.",
						}}
					}
					break
				}

				pending = append(pending, strings.TrimSpace(commentLine))
				i++
				if i >= len(lines) {
					return lineComments{}, &frontend.DiagnosticError{Info: frontend.Diagnostic{
						Code:     DiagnosticCodeFormatterCheck,
						Message:  "unterminated block comment",
						File:     filename,
						Line:     len(lines),
						Column:   1,
						Severity: "error",
					}}
				}
				commentLine = strings.TrimRight(lines[i], "\r")
				commentAt = 0
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		if len(pending) > 0 {
			out.before[codeLine] = append(out.before[codeLine], pending...)
			pending = nil
		}
		codeLine += formattedCodeLineCount(line)
	}
	out.trailing = pending
	return out, nil
}

func formattedCodeLineCount(line string) int {
	trimmed := strings.TrimSpace(line)
	if isFunctionHeaderLine(trimmed) {
		count := 1
		modifiers := countFunctionModifiers(trimmed)
		if modifiers > 0 {
			count += modifiers
		}
		if strings.Contains(trimmed, " = ") {
			count++
		}
		return count
	}

	if closure, ok := closureHeaderSegment(trimmed); ok {
		count := 1
		modifiers := countFunctionModifiers(closure)
		if modifiers > 0 {
			count += modifiers
		}
		if strings.Contains(closure, " = ") {
			count++
		}
		return count
	}

	return 1
}

func countFunctionModifiers(line string) int {
	count := 1
	if strings.Contains(line, " uses ") {
		count++
	}
	for _, clause := range []string{" noalloc", " noblock", " realtime", " nothrow", " budget("} {
		count += strings.Count(line, clause)
	}
	return count - 1
}

func isFunctionHeaderLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "func ") ||
		strings.HasPrefix(trimmed, "async func ") ||
		strings.HasPrefix(trimmed, "fun ") ||
		strings.HasPrefix(trimmed, "async fun ")
}

func closureHeaderSegment(trimmed string) (string, bool) {
	fnAt := strings.Index(trimmed, "fn(")
	if fnAt < 0 {
		fnAt = strings.Index(trimmed, "fun(")
	}
	if fnAt < 0 {
		return "", false
	}
	assignAt := strings.LastIndex(trimmed[:fnAt], "=")
	if assignAt < 0 {
		return "", false
	}
	return strings.TrimSpace(trimmed[fnAt:]), true
}

func commentStart(line string) (int, bool) {
	inString := false
	escaped := false
	for i := 0; i+1 < len(line); i++ {
		ch := line[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			continue
		}
		if ch == '/' && line[i+1] == '/' {
			return i, false
		}
		if ch == '/' && line[i+1] == '*' {
			return i, true
		}
	}
	return -1, false
}

func applyLineComments(formatted []byte, comments lineComments) []byte {
	if len(comments.before) == 0 && len(comments.trailing) == 0 {
		return formatted
	}
	lines := strings.Split(strings.TrimSuffix(string(formatted), "\n"), "\n")
	var b bytes.Buffer
	codeLine := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			if pending := comments.before[codeLine]; len(pending) > 0 {
				indent := leadingWhitespace(line)
				for _, comment := range pending {
					b.WriteString(indent)
					b.WriteString(comment)
					b.WriteByte('\n')
				}
			}
			codeLine++
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	for _, comment := range comments.trailing {
		b.WriteString(comment)
		b.WriteByte('\n')
	}
	return b.Bytes()
}

func leadingWhitespace(line string) string {
	for i, r := range line {
		if r != ' ' && r != '\t' {
			return line[:i]
		}
	}
	return line
}

type sourcePrinter struct {
	b             bytes.Buffer
	closures      map[string]*frontend.FuncDecl
	emitSynthetic map[string]struct{}
}

type actorMethodGroup struct {
	Name    string
	Public  bool
	Fields  []frontend.StateFieldDecl
	Methods []*frontend.FuncDecl
}

func (p *sourcePrinter) file(file *frontend.FileAST) {
	p.closures = make(map[string]*frontend.FuncDecl, len(file.Funcs))
	p.emitSynthetic = make(map[string]struct{})
	for _, fn := range file.Funcs {
		if fn.Synthetic {
			p.closures[fn.Name] = fn
		}
	}

	if file.Module != "" {
		p.line(0, "module "+file.Module)
		p.blank()
	}
	for _, imp := range file.Imports {
		line := publicPrefix(imp.Public) + "import " + imp.Path
		if len(imp.Items) > 0 {
			line += ".{" + strings.Join(imp.Items, ", ") + "}"
		}
		if imp.Alias != "" {
			line += " as " + imp.Alias
		}
		p.line(0, line)
	}
	if len(file.Imports) > 0 {
		p.blank()
	}
	for _, capsule := range file.Capsules {
		p.capsuleDecl(capsule)
		p.blank()
	}
	for _, en := range file.Enums {
		p.enumDecl(en)
		p.blank()
	}
	for _, st := range file.Structs {
		p.structDecl(st)
		p.blank()
	}
	for _, proto := range file.Protocols {
		p.protocolDecl(proto)
		p.blank()
	}
	for _, ext := range file.Extensions {
		p.extensionDecl(ext)
		p.blank()
	}
	for _, impl := range file.Impls {
		p.implDecl(impl)
		p.blank()
	}
	for _, st := range file.States {
		p.stateDecl(st)
		p.blank()
	}
	for _, view := range file.Views {
		p.viewDecl(view)
		p.blank()
	}
	for _, g := range file.Globals {
		p.globalDecl(g)
	}
	if len(file.Globals) > 0 {
		p.blank()
	}
	actorGroups, actorMethodNames := collectActorMethodGroups(file)
	for _, group := range actorGroups {
		p.actorDecl(group)
		p.blank()
	}
	for _, fn := range file.Funcs {
		if fn.ExtensionOf != "" || fn.Synthetic {
			continue
		}
		if _, ok := actorMethodNames[fn.Name]; ok {
			continue
		}
		p.funcDecl(fn)
		p.blank()
	}
	for _, fn := range file.Funcs {
		if !fn.Synthetic {
			continue
		}
		if _, ok := p.emitSynthetic[fn.Name]; !ok {
			continue
		}
		p.funcDecl(fn)
		p.blank()
	}
	for _, test := range file.Tests {
		p.testDecl(test)
		p.blank()
	}
	out := p.b.String()
	p.b.Reset()
	p.b.WriteString(strings.TrimRight(out, "\n"))
	p.b.WriteByte('\n')
}

func collectActorMethodGroups(file *frontend.FileAST) ([]actorMethodGroup, map[string]struct{}) {
	methodNames := make(map[string]struct{})
	groupIndex := map[string]int{}
	var groups []actorMethodGroup
	if file != nil {
		for _, actor := range file.Actors {
			if actor == nil {
				continue
			}
			idx := len(groups)
			groupIndex[actor.Name] = idx
			groups = append(groups, actorMethodGroup{
				Name:   actor.Name,
				Public: actor.Public,
				Fields: append([]frontend.StateFieldDecl(nil), actor.Fields...),
			})
			for _, fn := range actor.Methods {
				if fn == nil {
					continue
				}
				groups[idx].Methods = append(groups[idx].Methods, fn)
				methodNames[fn.Name] = struct{}{}
			}
		}
	}
	for _, fn := range file.Funcs {
		actorName, _, ok := actorMethodName(fn)
		if !ok {
			continue
		}
		if _, exists := methodNames[fn.Name]; exists {
			continue
		}
		idx, exists := groupIndex[actorName]
		if !exists {
			idx = len(groups)
			groupIndex[actorName] = idx
			groups = append(groups, actorMethodGroup{Name: actorName})
		}
		groups[idx].Methods = append(groups[idx].Methods, fn)
		methodNames[fn.Name] = struct{}{}
	}
	return groups, methodNames
}

func actorMethodName(fn *frontend.FuncDecl) (string, string, bool) {
	if fn.ExtensionOf != "" || fn.Synthetic {
		return "", "", false
	}
	parts := strings.Split(fn.Name, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func publicPrefix(public bool) string {
	if public {
		return "pub "
	}
	return ""
}

func (p *sourcePrinter) actorDecl(group actorMethodGroup) {
	p.line(0, publicPrefix(group.Public)+"actor "+group.Name+":")
	for _, field := range group.Fields {
		kw := "val"
		if field.Mutable {
			kw = "var"
		} else if field.Const {
			kw = "const"
		}
		line := kw + " " + field.Name + ": " + formatTypeRef(field.Type)
		if field.Init != nil {
			line += " = " + p.formatExpr(field.Init)
		}
		p.line(1, line)
	}
	for _, fn := range group.Methods {
		_, methodName, _ := actorMethodName(fn)
		p.funcDeclWithNameAt(fn, methodName, 1)
	}
}

func (p *sourcePrinter) protocolDecl(proto *frontend.ProtocolDecl) {
	p.line(0, publicPrefix(proto.Public)+"protocol "+proto.Name+":")
	for _, req := range proto.Requirements {
		p.line(1, formatFuncSigDecl(req))
	}
}

func (p *sourcePrinter) capsuleDecl(capsule *frontend.CapsuleDecl) {
	p.line(0, publicPrefix(capsule.Public)+"capsule "+capsule.Name+":")
	for _, entry := range capsule.Entries {
		p.line(1, entry.Key+": "+p.formatExpr(entry.Value))
	}
}

func (p *sourcePrinter) enumDecl(en *frontend.EnumDecl) {
	p.line(0, publicPrefix(en.Public)+"enum "+en.Name+":")
	for _, c := range en.Cases {
		line := "case " + c.Name
		if len(c.Payload) > 0 {
			types := make([]string, 0, len(c.Payload))
			for _, typ := range c.Payload {
				types = append(types, formatTypeRef(typ))
			}
			line += "(" + strings.Join(types, ", ") + ")"
		}
		p.line(1, line)
	}
}

func (p *sourcePrinter) structDecl(st *frontend.StructDecl) {
	typeParams := ""
	if len(st.TypeParams) > 0 {
		typeParams = "<" + strings.Join(st.TypeParams, ", ") + ">"
	}
	reprPrefix := ""
	if st.Repr == frontend.StructReprC {
		reprPrefix = "repr(C) "
	}
	p.line(0, publicPrefix(st.Public)+reprPrefix+"struct "+st.Name+typeParams+":")
	for _, f := range st.Fields {
		p.line(1, f.Name+": "+formatTypeRef(f.Type))
	}
}

func (p *sourcePrinter) stateDecl(st *frontend.StateDecl) {
	p.line(0, publicPrefix(st.Public)+"state "+st.Name+":")
	for _, field := range st.Fields {
		kw := "val"
		if field.Mutable {
			kw = "var"
		} else if field.Const {
			kw = "const"
		}
		p.line(1, kw+" "+field.Name+": "+formatTypeRef(field.Type)+" = "+p.formatExpr(field.Init))
	}
}

func (p *sourcePrinter) viewDecl(view *frontend.ViewDecl) {
	p.line(
		0,
		publicPrefix(view.Public)+"view "+view.Name+"(state: "+formatTypeRef(view.StateName)+"):",
	)
	for _, binding := range view.Bindings {
		p.line(
			1,
			"bind "+binding.Name+": "+formatTypeRef(binding.Type)+" = "+p.formatExpr(binding.Value),
		)
	}
	for _, event := range view.Events {
		p.line(1, "event "+event.Name+" -> "+event.Command)
	}
	for _, cmd := range view.Commands {
		p.line(1, "command "+cmd.Name+":")
		p.stmts(cmd.Body, 2)
	}
	for _, style := range view.Styles {
		p.line(
			1,
			"style "+style.Name+": "+formatTypeRef(style.Type)+" = "+p.formatExpr(style.Value),
		)
	}
	for _, entry := range view.Accessibility {
		p.line(
			1,
			"accessibility "+entry.Name+": "+formatTypeRef(
				entry.Type,
			)+" = "+p.formatExpr(
				entry.Value,
			),
		)
	}
}

func (p *sourcePrinter) extensionDecl(ext *frontend.ExtensionDecl) {
	p.line(0, publicPrefix(ext.Public)+"extension "+formatTypeRef(ext.Target)+":")
	for _, fn := range ext.Methods {
		p.funcDeclWithNameAt(fn, strings.TrimPrefix(fn.Name, fn.ExtensionOf+"."), 1)
	}
}

func (p *sourcePrinter) implDecl(impl *frontend.ImplDecl) {
	p.line(0, "impl "+formatTypeRef(impl.Type)+": "+formatTypeRef(impl.Protocol))
}

func (p *sourcePrinter) globalDecl(g *frontend.GlobalDecl) {
	kw := "val"
	if g.Mutable {
		kw = "var"
	} else if g.Const {
		kw = "const"
	}
	line := kw + " " + g.Name
	line = publicPrefix(g.Public) + line
	if formatTypeRefPresent(g.Type) {
		line += ": " + formatTypeRef(g.Type)
	}
	if g.Init != nil {
		line += " = " + p.formatExpr(g.Init)
	}
	p.line(0, line)
}

func (p *sourcePrinter) funcDecl(fn *frontend.FuncDecl) {
	p.funcDeclWithName(fn, fn.Name)
}

func (p *sourcePrinter) funcDeclWithName(fn *frontend.FuncDecl, name string) {
	p.funcDeclWithNameAt(fn, name, 0)
}

func (p *sourcePrinter) funcDeclWithNameAt(fn *frontend.FuncDecl, name string, indent int) {
	if fn.ExportName != "" {
		p.line(indent, "@export("+strconv.Quote(fn.ExportName)+")")
	}
	typeParams := ""
	if len(fn.TypeParams) > 0 {
		typeParams = "<" + strings.Join(fn.TypeParams, ", ") + ">"
	}
	keyword := "func"
	if fn.Closure && !fn.Synthetic {
		keyword = "closure"
	}
	header := p.functionHeader(
		keyword,
		fn.Async,
		name+typeParams,
		fn.Params,
		fn.ReturnType,
		fn.HasThrows,
		fn.Throws,
	)
	if fn.Public {
		header = "pub " + header
	}
	p.emitHeaderWithModifiers(indent, header, fn.Uses, fn.SemanticClauses)
	p.stmts(fn.Body, indent+1)
}

func formatFuncSigDecl(sig frontend.FuncSigDecl) string {
	var params []string
	for _, param := range sig.Params {
		typ := formatTypeRef(param.Type)
		if param.Ownership != "" {
			typ = param.Ownership + " " + typ
		}
		params = append(params, param.Name+": "+typ)
	}
	prefix := "func "
	if sig.Async {
		prefix = "async func "
	}
	typeParams := ""
	if len(sig.TypeParams) > 0 {
		typeParams = "<" + strings.Join(sig.TypeParams, ", ") + ">"
	}
	out := prefix + sig.Name + typeParams + "(" + strings.Join(
		params,
		", ",
	) + ") -> " + formatTypeRef(
		sig.ReturnType,
	)
	if sig.HasThrows {
		out += " throws " + formatTypeRef(sig.Throws)
	}
	if len(sig.Uses) > 0 {
		uses := append([]string(nil), sig.Uses...)
		sort.Strings(uses)
		out += " uses " + strings.Join(uses, ", ")
	}
	return out
}

func (p *sourcePrinter) testDecl(test *frontend.TestDecl) {
	p.line(0, "test "+strconv.Quote(test.Name)+":")
	p.stmts(test.Body, 1)
}

func (p *sourcePrinter) stmts(stmts []frontend.Stmt, indent int) {
	for _, stmt := range stmts {
		p.stmt(stmt, indent)
	}
}

func (p *sourcePrinter) ifStmt(s *frontend.IfStmt, indent int, prefix string) {
	p.line(indent, prefix+" "+p.formatExpr(s.Cond)+":")
	p.stmts(s.Then, indent+1)
	p.elseStmts(s.Else, indent)
}

func (p *sourcePrinter) ifLetStmt(s *frontend.IfLetStmt, indent int, prefix string) {
	binding := s.Name
	if s.Pattern != nil {
		binding = p.formatExpr(s.Pattern)
	}
	p.line(indent, prefix+" let "+binding+" = "+p.formatExpr(s.Value)+":")
	p.stmts(s.Then, indent+1)
	p.elseStmts(s.Else, indent)
}

func (p *sourcePrinter) elseStmts(stmts []frontend.Stmt, indent int) {
	if len(stmts) == 0 {
		return
	}
	if len(stmts) == 1 {
		switch nested := stmts[0].(type) {
		case *frontend.IfStmt:
			p.ifStmt(nested, indent, "else if")
			return
		case *frontend.IfLetStmt:
			p.ifLetStmt(nested, indent, "else if")
			return
		}
	}
	p.line(indent, "else:")
	p.stmts(stmts, indent+1)
}

func (p *sourcePrinter) emitMatchExprStatement(indent int, prefix string, expr frontend.Expr) bool {
	match, ok := expr.(*frontend.MatchExpr)
	if !ok {
		return false
	}
	p.line(indent, prefix+"match "+p.formatExpr(match.Value)+":")
	for _, c := range match.Cases {
		guard := ""
		if c.Guard != nil {
			guard = " if " + p.formatExpr(c.Guard)
		}
		if c.Default {
			p.line(indent, "case _"+guard+":")
		} else {
			p.line(indent, "case "+p.formatExpr(c.Pattern)+guard+":")
		}
		p.exprBlock(indent+1, c.Value)
	}
	return true
}

func (p *sourcePrinter) emitCatchExprStatement(indent int, prefix string, expr frontend.Expr) bool {
	catch, ok := expr.(*frontend.CatchExpr)
	if !ok {
		return false
	}
	p.line(indent, prefix+"catch "+p.formatExpr(catch.Call)+":")
	for _, c := range catch.Cases {
		guard := ""
		if c.Guard != nil {
			guard = " if " + p.formatExpr(c.Guard)
		}
		if c.Default {
			p.line(indent, "case _"+guard+":")
		} else {
			p.line(indent, "case "+p.formatExpr(c.Pattern)+guard+":")
		}
		p.exprBlock(indent+1, c.Value)
	}
	return true
}

func (p *sourcePrinter) stmt(stmt frontend.Stmt, indent int) {
	switch s := stmt.(type) {
	case *frontend.PrintStmt:
		p.line(indent, "print("+p.formatExpr(s.Value)+")")
	case *frontend.ExpectStmt:
		p.line(indent, "expect "+p.formatExpr(s.Cond))
	case *frontend.ReturnStmt:
		if p.emitClosureValueStatement(indent, "return ", s.Value) {
			return
		}
		if p.emitMatchExprStatement(indent, "return ", s.Value) {
			return
		}
		if p.emitCatchExprStatement(indent, "return ", s.Value) {
			return
		}
		if p.emitMultilineExprStatement(indent, "return ", s.Value) {
			return
		}
		p.line(indent, "return "+p.formatExpr(s.Value))
	case *frontend.ThrowStmt:
		if p.emitClosureValueStatement(indent, "throw ", s.Value) {
			return
		}
		if p.emitMultilineExprStatement(indent, "throw ", s.Value) {
			return
		}
		p.line(indent, "throw "+p.formatExpr(s.Value))
	case *frontend.DeferStmt:
		p.line(indent, "defer:")
		p.stmts(s.Body, indent+1)
	case *frontend.BreakStmt:
		p.line(indent, "break")
	case *frontend.ContinueStmt:
		p.line(indent, "continue")
	case *frontend.LetStmt:
		kw := "let"
		if s.Mutable {
			kw = "var"
		} else if s.Const {
			kw = "const"
		}
		line := kw + " " + s.Name
		if formatTypeRefPresent(s.Type) {
			line += ": " + formatTypeRef(s.Type)
		}
		line += " = "
		if p.emitClosureValueStatement(indent, line, s.Value) {
			return
		}
		if p.emitMatchExprStatement(indent, line, s.Value) {
			return
		}
		if p.emitCatchExprStatement(indent, line, s.Value) {
			return
		}
		if p.emitMultilineExprStatement(indent, line, s.Value) {
			return
		}
		p.line(indent, line+p.formatExpr(s.Value))
	case *frontend.AssignStmt:
		if s.Op != 0 && s.CompoundValue != nil {
			p.line(
				indent,
				p.formatExpr(s.Target)+" "+compoundAssignmentOp(s.Op)+"= "+p.formatExpr(s.CompoundValue),
			)
			return
		}
		target := p.formatExpr(s.Target) + " = "
		if p.emitClosureValueStatement(indent, target, s.Value) {
			return
		}
		if p.emitMatchExprStatement(indent, target, s.Value) {
			return
		}
		if p.emitCatchExprStatement(indent, target, s.Value) {
			return
		}
		if p.emitMultilineExprStatement(indent, target, s.Value) {
			return
		}
		p.line(indent, target+p.formatExpr(s.Value))
	case *frontend.IfStmt:
		p.ifStmt(s, indent, "if")
	case *frontend.IfLetStmt:
		p.ifLetStmt(s, indent, "if")
	case *frontend.WhileStmt:
		p.line(indent, "while "+p.formatExpr(s.Cond)+":")
		p.stmts(s.Body, indent+1)
	case *frontend.ForRangeStmt:
		if s.Iterable != nil {
			p.line(indent, "for "+s.Name+" in "+p.formatExpr(s.Iterable)+":")
		} else {
			p.line(indent, "for "+s.Name+" in "+p.formatExpr(s.Start)+"..<"+p.formatExpr(s.End)+":")
		}
		p.stmts(s.Body, indent+1)
	case *frontend.MatchStmt:
		p.line(indent, "match "+p.formatExpr(s.Value)+":")
		for _, c := range s.Cases {
			guard := ""
			if c.Guard != nil {
				guard = " if " + p.formatExpr(c.Guard)
			}
			if c.Default {
				p.line(indent, "case _"+guard+":")
			} else {
				p.line(indent, "case "+p.formatExpr(c.Pattern)+guard+":")
			}
			p.stmts(c.Body, indent+1)
		}
	case *frontend.UnsafeStmt:
		p.line(indent, "unsafe:")
		p.stmts(s.Body, indent+1)
	case *frontend.IslandStmt:
		p.line(indent, "island("+p.formatExpr(s.Size)+") as "+s.Name+":")
		p.stmts(s.Body, indent+1)
	case *frontend.FreeStmt:
		p.line(indent, "free("+p.formatExpr(s.Value)+")")
	case *frontend.ExprStmt:
		if p.emitClosureValueStatement(indent, "", s.Expr) {
			return
		}
		if p.emitMultilineExprStatement(indent, "", s.Expr) {
			return
		}
		p.line(indent, p.formatExpr(s.Expr))
	}
}

func (p *sourcePrinter) line(indent int, s string) {
	p.b.WriteString(strings.Repeat(" ", indent*4))
	p.b.WriteString(s)
	p.b.WriteByte('\n')
}

func (p *sourcePrinter) exprBlock(indent int, expr frontend.Expr) {
	for _, line := range strings.Split(p.formatExpr(expr), "\n") {
		p.line(indent, line)
	}
}

func (p *sourcePrinter) emitMultilineExprStatement(
	indent int,
	prefix string,
	expr frontend.Expr,
) bool {
	if expr == nil {
		return false
	}
	formatted := p.formatExpr(expr)
	if !strings.Contains(formatted, "\n") {
		return false
	}
	lines := strings.Split(formatted, "\n")
	p.line(indent, prefix+lines[0])
	for _, line := range lines[1:] {
		p.line(indent, line)
	}
	return true
}

func (p *sourcePrinter) blank() {
	p.b.WriteByte('\n')
}

func formatTypeRefPresent(ref frontend.TypeRef) bool {
	if ref.Name != "" || ref.Elem != nil || ref.Return != nil {
		return true
	}
	if len(ref.Params) > 0 || len(ref.TypeArgs) > 0 {
		return true
	}
	switch ref.Kind {
	case frontend.TypeRefSlice,
		frontend.TypeRefArray,
		frontend.TypeRefOptional,
		frontend.TypeRefFunction:
		return true
	default:
		return false
	}
}

func formatTypeRef(ref frontend.TypeRef) string {
	switch ref.Kind {
	case frontend.TypeRefSlice:
		return "[]" + formatTypeRef(*ref.Elem)
	case frontend.TypeRefArray:
		return "[" + strconv.Itoa(ref.Len) + "]" + formatTypeRef(*ref.Elem)
	case frontend.TypeRefOptional:
		return formatTypeRef(*ref.Elem) + "?"
	case frontend.TypeRefFunction:
		params := make([]string, 0, len(ref.Params))
		for i, param := range ref.Params {
			formatted := formatTypeRef(param)
			if i < len(ref.ParamOwnership) && ref.ParamOwnership[i] != "" {
				formatted = ref.ParamOwnership[i] + " " + formatted
			}
			params = append(params, formatted)
		}
		ret := "?"
		if ref.Return != nil {
			ret = formatTypeRef(*ref.Return)
		}
		out := "fn(" + strings.Join(params, ", ") + ") -> " + ret
		if ref.Throws != nil {
			out += " throws " + formatTypeRef(*ref.Throws)
		}
		if len(ref.Uses) > 0 {
			out += " uses " + strings.Join(ref.Uses, ", ")
		}
		return out
	default:
		if len(ref.TypeArgs) == 0 {
			return ref.Name
		}
		args := make([]string, 0, len(ref.TypeArgs))
		for _, arg := range ref.TypeArgs {
			args = append(args, formatTypeRef(arg))
		}
		return ref.Name + "<" + strings.Join(args, ", ") + ">"
	}
}

func (p *sourcePrinter) functionHeader(
	keyword string,
	async bool,
	name string,
	params []frontend.ParamDecl,
	retType frontend.TypeRef,
	hasThrows bool,
	throws frontend.TypeRef,
) string {
	var out []string
	for _, param := range params {
		typ := formatTypeRef(param.Type)
		if param.Ownership != "" {
			typ = param.Ownership + " " + typ
		}
		out = append(out, param.Name+": "+typ)
	}

	head := keyword
	if async {
		head = "async " + head
	}
	if name != "" {
		head += " " + name
	}
	head += "(" + strings.Join(out, ", ") + ") -> " + formatTypeRef(retType)
	if hasThrows {
		head += " throws " + formatTypeRef(throws)
	}
	return head
}

func (p *sourcePrinter) emitHeaderWithModifiers(
	indent int,
	header string,
	uses []string,
	clauses []frontend.SemanticClause,
) {
	modifiers := p.functionModifiers(uses, clauses)
	if len(modifiers) == 0 {
		p.line(indent, header+":")
		return
	}

	p.line(indent, header)
	for i, mod := range modifiers {
		if i == len(modifiers)-1 {
			mod += ":"
		}
		p.line(indent, mod)
	}
}

func (p *sourcePrinter) functionModifiers(
	uses []string,
	clauses []frontend.SemanticClause,
) []string {
	out := make([]string, 0, len(clauses)+1)
	if len(uses) > 0 {
		sorted := append([]string(nil), uses...)
		sort.Strings(sorted)
		out = append(out, "uses "+strings.Join(sorted, ", "))
	}
	for _, clause := range clauses {
		out = append(out, p.formatSemanticClause(clause))
	}
	return out
}

func (p *sourcePrinter) formatSemanticClause(clause frontend.SemanticClause) string {
	if clause.Value == nil {
		return clause.Name
	}
	return clause.Name + "(" + p.formatExpr(clause.Value) + ")"
}

func (p *sourcePrinter) closureHeader(fn *frontend.FuncDecl) string {
	return p.functionHeader("fn", false, "", fn.Params, fn.ReturnType, fn.HasThrows, fn.Throws)
}

func (p *sourcePrinter) inlineFunctionModifiers(
	uses []string,
	clauses []frontend.SemanticClause,
) string {
	return strings.Join(p.functionModifiers(uses, clauses), " ")
}

func (p *sourcePrinter) closureExprDecl(expr frontend.Expr) (*frontend.FuncDecl, bool) {
	closureExpr, ok := expr.(*frontend.ClosureExpr)
	if !ok {
		return nil, false
	}
	closure, ok := p.closures[closureExpr.Name]
	if !ok {
		return nil, false
	}
	return closure, true
}

func (p *sourcePrinter) emitClosureValueStatement(
	indent int,
	prefix string,
	expr frontend.Expr,
) bool {
	closure, ok := p.closureExprDecl(expr)
	if !ok {
		return false
	}
	p.emitHeaderWithModifiers(
		indent,
		prefix+p.closureHeader(closure),
		closure.Uses,
		closure.SemanticClauses,
	)
	p.stmts(closure.Body, indent+1)
	return true
}

func singleReturnExpr(stmts []frontend.Stmt) (frontend.Expr, bool) {
	if len(stmts) != 1 {
		return nil, false
	}
	ret, ok := stmts[0].(*frontend.ReturnStmt)
	if !ok || ret.Value == nil {
		return nil, false
	}
	return ret.Value, true
}

func (p *sourcePrinter) formatExpr(expr frontend.Expr) string {
	return p.formatExprPrec(expr, 0)
}

func (p *sourcePrinter) formatExprPrec(expr frontend.Expr, parent int) string {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return strconv.Itoa(int(e.Value))
	case *frontend.BoolLitExpr:
		if e.Value {
			return "true"
		}
		return "false"
	case *frontend.NoneLitExpr:
		return "none"
	case *frontend.SomePatternExpr:
		return "some(" + e.Name + ")"
	case *frontend.EnumCasePatternExpr:
		if !e.HasPayload {
			return e.TypeName + "." + e.CaseName
		}
		return e.TypeName + "." + e.CaseName + "(" + strings.Join(e.Bindings, ", ") + ")"
	case *frontend.MatchExpr:
		var b strings.Builder
		b.WriteString("match ")
		b.WriteString(p.formatExpr(e.Value))
		b.WriteString(":")
		for _, c := range e.Cases {
			b.WriteByte('\n')
			guard := ""
			if c.Guard != nil {
				guard = " if " + p.formatExpr(c.Guard)
			}
			if c.Default {
				b.WriteString("case _")
				b.WriteString(guard)
				b.WriteString(":")
			} else {
				b.WriteString("case ")
				b.WriteString(p.formatExpr(c.Pattern))
				b.WriteString(guard)
				b.WriteString(":")
			}
			b.WriteByte('\n')
			b.WriteString("    ")
			b.WriteString(p.formatExpr(c.Value))
		}
		return b.String()
	case *frontend.CatchExpr:
		var b strings.Builder
		b.WriteString("catch ")
		b.WriteString(p.formatExpr(e.Call))
		b.WriteString(":")
		for _, c := range e.Cases {
			b.WriteByte('\n')
			guard := ""
			if c.Guard != nil {
				guard = " if " + p.formatExpr(c.Guard)
			}
			if c.Default {
				b.WriteString("case _")
				b.WriteString(guard)
				b.WriteString(":")
			} else {
				b.WriteString("case ")
				b.WriteString(p.formatExpr(c.Pattern))
				b.WriteString(guard)
				b.WriteString(":")
			}
			b.WriteByte('\n')
			b.WriteString("    ")
			b.WriteString(p.formatExpr(c.Value))
		}
		return b.String()
	case *frontend.StringLitExpr:
		return strconv.Quote(string(e.Value))
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.UnaryExpr:
		if e.Op == frontend.TokenBang {
			return "!" + p.formatExprPrec(e.X, 7)
		}
		return "-" + p.formatExprPrec(e.X, 7)
	case *frontend.TryExpr:
		return "try " + p.formatExprPrec(e.X, 7)
	case *frontend.AwaitExpr:
		return "await " + p.formatExprPrec(e.X, 7)
	case *frontend.BinaryExpr:
		prec := exprPrecedence(e.Op)
		left := p.formatExprPrec(e.Left, prec)
		right := p.formatExprPrec(e.Right, prec+1)
		out := left + " " + tokenOp(e.Op) + " " + right
		if prec < parent {
			return "(" + out + ")"
		}
		return out
	case *frontend.CallExpr:
		args := make([]string, 0, len(e.Args))
		for i, arg := range e.Args {
			if i < len(e.ArgLabels) && e.ArgLabels[i] != "" {
				args = append(args, e.ArgLabels[i]+": "+p.formatExpr(arg))
				continue
			}
			args = append(args, p.formatExpr(arg))
		}
		typeArgs := ""
		if len(e.TypeArgs) > 0 {
			parts := make([]string, 0, len(e.TypeArgs))
			for _, arg := range e.TypeArgs {
				parts = append(parts, formatTypeRef(arg))
			}
			typeArgs = "<" + strings.Join(parts, ", ") + ">"
		}
		return e.Name + typeArgs + "(" + strings.Join(args, ", ") + ")"
	case *frontend.StructLitExpr:
		fields := make([]string, 0, len(e.Fields))
		for _, f := range e.Fields {
			fields = append(fields, f.Name+": "+p.formatExpr(f.Value))
		}
		typ := formatTypeRef(e.Type)
		oneLine := typ + "(" + strings.Join(fields, ", ") + ")"
		if len([]rune(oneLine)) <= 100 {
			return oneLine
		}
		var b strings.Builder
		b.WriteString(typ)
		b.WriteString("(")
		for _, field := range fields {
			b.WriteString("\n    ")
			b.WriteString(field)
			b.WriteByte(',')
		}
		b.WriteString("\n)")
		return b.String()
	case *frontend.ClosureExpr:
		closure, ok := p.closures[e.Name]
		if !ok {
			return "<expr>"
		}
		value, ok := singleReturnExpr(closure.Body)
		if !ok {
			p.emitSynthetic[e.Name] = struct{}{}
			return e.Name
		}
		header := p.closureHeader(closure)
		if mods := p.inlineFunctionModifiers(closure.Uses, closure.SemanticClauses); mods != "" {
			header += " " + mods
		}
		return header + " = " + p.formatExpr(value)
	case *frontend.FieldAccessExpr:
		return p.formatExprPrec(e.Base, 8) + "." + e.Field
	case *frontend.IndexExpr:
		return p.formatExprPrec(e.Base, 8) + "[" + p.formatExpr(e.Index) + "]"
	default:
		return "<expr>"
	}
}

func exprPrecedence(op frontend.TokenType) int {
	switch op {
	case frontend.TokenPipePipe:
		return 1
	case frontend.TokenAmpAmp:
		return 2
	case frontend.TokenEqEq, frontend.TokenBangEq:
		return 3
	case frontend.TokenLess, frontend.TokenLessEq, frontend.TokenGreater, frontend.TokenGreaterEq:
		return 4
	case frontend.TokenPlus, frontend.TokenMinus:
		return 5
	case frontend.TokenStar, frontend.TokenSlash, frontend.TokenPercent:
		return 6
	default:
		return 0
	}
}

func tokenOp(op frontend.TokenType) string {
	switch op {
	case frontend.TokenPipePipe:
		return "||"
	case frontend.TokenAmpAmp:
		return "&&"
	case frontend.TokenEqEq:
		return "=="
	case frontend.TokenBangEq:
		return "!="
	case frontend.TokenLess:
		return "<"
	case frontend.TokenLessEq:
		return "<="
	case frontend.TokenGreater:
		return ">"
	case frontend.TokenGreaterEq:
		return ">="
	case frontend.TokenPlus:
		return "+"
	case frontend.TokenMinus:
		return "-"
	case frontend.TokenStar:
		return "*"
	case frontend.TokenSlash:
		return "/"
	case frontend.TokenPercent:
		return "%"
	default:
		return "?"
	}
}

func compoundAssignmentOp(op frontend.TokenType) string {
	switch op {
	case frontend.TokenPlus:
		return "+"
	case frontend.TokenMinus:
		return "-"
	case frontend.TokenStar:
		return "*"
	case frontend.TokenSlash:
		return "/"
	case frontend.TokenPercent:
		return "%"
	default:
		return "?"
	}
}

// ---- formats.go ----

type FormatInfo = formats.Info

const (
	T4SourceExtension          = formats.T4SourceExtension
	LegacyTetraSourceExtension = formats.LegacyTetraSourceExtension
	TodexFragmentExtension     = formats.TodexFragmentExtension
	T4SeedExtension            = formats.T4SeedExtension
	T4InterfaceExtension       = formats.T4InterfaceExtension
	T4ProofExtension           = formats.T4ProofExtension
	T4ReplayExtension          = formats.T4ReplayExtension
	T4QuestExtension           = formats.T4QuestExtension
	NeedMapExtension           = formats.NeedMapExtension

	CapsuleFileName       = formats.CapsuleFileName
	LegacyCapsuleFileName = formats.LegacyCapsuleFileName
	SemanticLockFileName  = formats.SemanticLockFileName
	DefaultSourceFileName = formats.DefaultSourceFileName
	LegacySourceFileName  = formats.LegacySourceFileName
	DefaultSeedFileName   = formats.DefaultSeedFileName
	DefaultNeedMapName    = formats.DefaultNeedMapName
)

func T4Formats() []FormatInfo {
	return formats.All()
}

func SourceExtensions() []string {
	return formats.SourceExtensions()
}

func IsSourceFile(path string) bool {
	return formats.IsSourceFile(path)
}

// ---- interface.go ----

func GenerateInterfaceFile(inputPath string) ([]byte, error) {
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, err
	}
	return GenerateInterfaceFromSource(raw, inputPath)
}

func GenerateInterfaceFromSource(src []byte, filename string) ([]byte, error) {
	file, err := frontend.ParseFile(src, filename)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if file.Module != "" {
		fmt.Fprintf(&b, "module %s\n\n", file.Module)
	}
	explicitPublic := interfaceUsesExplicitPublic(file)
	imports := interfaceImportsForPublicSurface(file, explicitPublic)
	for _, imp := range imports {
		writeInterfaceImport(&b, imp, explicitPublic)
	}
	if len(imports) > 0 {
		b.WriteByte('\n')
	}
	for _, en := range file.Enums {
		if interfaceDeclPublic(file, en.Public) {
			writeInterfaceEnum(&b, en, explicitPublic)
		}
	}
	for _, st := range file.Structs {
		if interfaceDeclPublic(file, st.Public) {
			writeInterfaceStruct(&b, st.Name, st.TypeParams, st.Fields, explicitPublic)
		}
	}
	for _, st := range file.States {
		if !interfaceDeclPublic(file, st.Public) {
			continue
		}
		var fields []frontend.FieldDecl
		for _, field := range st.Fields {
			fields = append(
				fields,
				frontend.FieldDecl{At: field.At, Name: field.Name, Type: field.Type},
			)
		}
		writeInterfaceStruct(&b, st.Name, nil, fields, explicitPublic)
	}
	for _, proto := range file.Protocols {
		if !interfaceDeclPublic(file, proto.Public) {
			continue
		}
		if explicitPublic {
			b.WriteString("pub ")
		}
		fmt.Fprintf(&b, "protocol %s:\n", proto.Name)
		for _, req := range proto.Requirements {
			fmt.Fprintf(&b, "    %s\n", formatLSPFuncSigDecl(req))
		}
		b.WriteByte('\n')
	}
	for _, ext := range file.Extensions {
		if !interfaceDeclPublic(file, ext.Public) {
			continue
		}
		writeInterfaceExtension(&b, ext, explicitPublic)
	}
	for _, impl := range file.Impls {
		if !interfaceImplPublic(file, impl, explicitPublic) {
			continue
		}
		writeInterfaceImpl(&b, impl)
	}
	for _, fn := range file.Funcs {
		if fn.Synthetic || fn.ExtensionOf != "" || !interfaceDeclPublic(file, fn.Public) {
			continue
		}
		if explicitPublic {
			b.WriteString("pub ")
		}
		fmt.Fprintf(&b, "%s:\n", formatLSPFuncDetail(fn))
		fmt.Fprintf(&b, "%s\n\n", interfaceFunctionBody(fn))
	}
	writeInterfaceHashOnlySurface(&b, file, explicitPublic)
	return t4iface.WithHashHeader(b.Bytes()), nil
}

func InterfaceFingerprintFromSource(src []byte, filename string) (string, error) {
	raw, err := GenerateInterfaceFromSource(src, filename)
	if err != nil {
		return "", err
	}
	hash, _, ok, err := t4iface.SplitHashHeader(raw)
	if err != nil {
		return "", err
	}
	if !ok {
		return t4iface.FingerprintBody(raw), nil
	}
	return hash, nil
}

func InterfaceFingerprintFromT4I(raw []byte) (string, error) {
	return t4iface.ValidateHash(raw)
}

func ValidateInterfaceAgainstSource(src []byte, iface []byte, filename string) error {
	expected, err := InterfaceFingerprintFromSource(src, filename)
	if err != nil {
		return err
	}
	actual, err := InterfaceFingerprintFromT4I(iface)
	if err != nil {
		return err
	}
	if expected != actual {
		return fmt.Errorf(
			"%s: public API mismatch: source %s, interface %s",
			filename,
			expected,
			actual,
		)
	}
	return nil
}

func writeInterfaceImport(b *bytes.Buffer, imp frontend.ImportDecl, explicitPublic bool) {
	if explicitPublic && imp.Public {
		b.WriteString("pub ")
	}
	if len(imp.Items) > 0 {
		fmt.Fprintf(b, "import %s.{%s}\n", imp.Path, strings.Join(imp.Items, ", "))
		return
	}
	if imp.Alias != "" {
		fmt.Fprintf(b, "import %s as %s\n", imp.Path, imp.Alias)
	} else {
		fmt.Fprintf(b, "import %s\n", imp.Path)
	}
}

func interfaceUsesExplicitPublic(file *frontend.FileAST) bool {
	if file == nil {
		return false
	}
	for _, imp := range file.Imports {
		if imp.Public {
			return true
		}
	}
	for _, en := range file.Enums {
		if en.Public {
			return true
		}
	}
	for _, st := range file.Structs {
		if st.Public {
			return true
		}
	}
	for _, st := range file.States {
		if st.Public {
			return true
		}
	}
	for _, view := range file.Views {
		if view.Public {
			return true
		}
	}
	for _, proto := range file.Protocols {
		if proto.Public {
			return true
		}
	}
	for _, ext := range file.Extensions {
		if ext.Public {
			return true
		}
	}
	for _, glob := range file.Globals {
		if glob.Public {
			return true
		}
	}
	for _, fn := range file.Funcs {
		if fn.Public {
			return true
		}
	}
	return false
}

func interfaceDeclPublic(file *frontend.FileAST, public bool) bool {
	if !interfaceUsesExplicitPublic(file) {
		return true
	}
	return public
}

func interfaceImportsForPublicSurface(
	file *frontend.FileAST,
	explicitPublic bool,
) []frontend.ImportDecl {
	if file == nil {
		return nil
	}
	if !explicitPublic {
		return append([]frontend.ImportDecl(nil), file.Imports...)
	}
	refs := interfacePublicTypeRefs(file, explicitPublic)
	out := make([]frontend.ImportDecl, 0, len(file.Imports))
	for _, imp := range file.Imports {
		if imp.Public || interfaceImportUsedByRefs(imp, refs) {
			out = append(out, imp)
		}
	}
	return out
}

func interfacePublicTypeRefs(file *frontend.FileAST, explicitPublic bool) map[string]struct{} {
	refs := map[string]struct{}{}
	add := func(ref frontend.TypeRef) {
		addInterfaceTypeRef(refs, ref)
	}
	for _, en := range file.Enums {
		if !interfaceDeclPublic(file, en.Public) {
			continue
		}
		for _, item := range en.Cases {
			for _, payload := range item.Payload {
				add(payload)
			}
		}
	}
	for _, st := range file.Structs {
		if !interfaceDeclPublic(file, st.Public) {
			continue
		}
		for _, field := range st.Fields {
			add(field.Type)
		}
	}
	for _, st := range file.States {
		if !interfaceDeclPublic(file, st.Public) {
			continue
		}
		for _, field := range st.Fields {
			add(field.Type)
		}
	}
	for _, view := range file.Views {
		if !interfaceDeclPublic(file, view.Public) {
			continue
		}
		add(view.StateName)
		for _, binding := range view.Bindings {
			add(binding.Type)
		}
		for _, style := range view.Styles {
			add(style.Type)
		}
		for _, item := range view.Accessibility {
			add(item.Type)
		}
	}
	for _, proto := range file.Protocols {
		if !interfaceDeclPublic(file, proto.Public) {
			continue
		}
		for _, req := range proto.Requirements {
			addInterfaceFuncSigTypeRefs(refs, req)
		}
	}
	for _, ext := range file.Extensions {
		if !interfaceDeclPublic(file, ext.Public) {
			continue
		}
		add(ext.Target)
		for _, method := range ext.Methods {
			addInterfaceFuncTypeRefs(refs, method)
		}
	}
	for _, impl := range file.Impls {
		if !interfaceImplPublic(file, impl, explicitPublic) {
			continue
		}
		add(impl.Type)
		add(impl.Protocol)
	}
	for _, glob := range file.Globals {
		if !interfaceDeclPublic(file, glob.Public) {
			continue
		}
		add(glob.Type)
	}
	for _, fn := range file.Funcs {
		if fn.Synthetic || fn.ExtensionOf != "" || !interfaceDeclPublic(file, fn.Public) {
			continue
		}
		addInterfaceFuncTypeRefs(refs, fn)
	}
	return refs
}

func addInterfaceFuncTypeRefs(refs map[string]struct{}, fn *frontend.FuncDecl) {
	if fn == nil {
		return
	}
	for _, bound := range fn.TypeParamBounds {
		addInterfaceTypeRef(refs, bound.Bound)
	}
	for _, param := range fn.Params {
		addInterfaceTypeRef(refs, param.Type)
	}
	addInterfaceTypeRef(refs, fn.ReturnType)
	if fn.HasThrows {
		addInterfaceTypeRef(refs, fn.Throws)
	}
}

func addInterfaceFuncSigTypeRefs(refs map[string]struct{}, sig frontend.FuncSigDecl) {
	for _, param := range sig.Params {
		addInterfaceTypeRef(refs, param.Type)
	}
	addInterfaceTypeRef(refs, sig.ReturnType)
	if sig.HasThrows {
		addInterfaceTypeRef(refs, sig.Throws)
	}
}

func addInterfaceTypeRef(refs map[string]struct{}, ref frontend.TypeRef) {
	for _, arg := range ref.TypeArgs {
		addInterfaceTypeRef(refs, arg)
	}
	switch ref.Kind {
	case frontend.TypeRefSlice, frontend.TypeRefArray, frontend.TypeRefOptional:
		if ref.Elem != nil {
			addInterfaceTypeRef(refs, *ref.Elem)
		}
	case frontend.TypeRefFunction:
		for _, param := range ref.Params {
			addInterfaceTypeRef(refs, param)
		}
		if ref.Return != nil {
			addInterfaceTypeRef(refs, *ref.Return)
		}
		if ref.Throws != nil {
			addInterfaceTypeRef(refs, *ref.Throws)
		}
	default:
		if ref.Name != "" {
			refs[ref.Name] = struct{}{}
		}
	}
}

func interfaceImportUsedByRefs(imp frontend.ImportDecl, refs map[string]struct{}) bool {
	if len(refs) == 0 {
		return false
	}
	alias := imp.Alias
	if alias == "" {
		alias = lastPathSegment(imp.Path)
	}
	for name := range refs {
		if name == alias || strings.HasPrefix(name, alias+".") ||
			strings.HasPrefix(name, imp.Path+".") {
			return true
		}
		for _, item := range imp.Items {
			if name == item || lastPathSegment(name) == item {
				return true
			}
		}
	}
	return false
}

func lastPathSegment(path string) string {
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func writeInterfaceHashOnlySurface(b *bytes.Buffer, file *frontend.FileAST, explicitPublic bool) {
	wroteHeader := false
	writeHeader := func() {
		if wroteHeader {
			return
		}
		b.WriteString("// hash-only public surface:\n")
		wroteHeader = true
	}
	for _, glob := range file.Globals {
		if !interfaceDeclPublic(file, glob.Public) {
			continue
		}
		writeHeader()
		fmt.Fprintf(b, "// global %s\n", formatLSPGlobalDetail(glob))
	}
	for _, view := range file.Views {
		if !interfaceDeclPublic(file, view.Public) {
			continue
		}
		writeHeader()
		fmt.Fprintf(b, "// view %s(%s)\n", view.Name, formatLSPTypeRef(view.StateName))
		for _, binding := range view.Bindings {
			fmt.Fprintf(
				b,
				"// view %s binding %s: %s\n",
				view.Name,
				binding.Name,
				formatLSPTypeRef(binding.Type),
			)
		}
		for _, event := range view.Events {
			fmt.Fprintf(b, "// view %s event %s -> %s\n", view.Name, event.Name, event.Command)
		}
		for _, command := range view.Commands {
			fmt.Fprintf(b, "// view %s command %s\n", view.Name, command.Name)
		}
		for _, style := range view.Styles {
			fmt.Fprintf(
				b,
				"// view %s style %s: %s\n",
				view.Name,
				style.Name,
				formatLSPTypeRef(style.Type),
			)
		}
		for _, item := range view.Accessibility {
			fmt.Fprintf(
				b,
				"// view %s accessibility %s: %s\n",
				view.Name,
				item.Name,
				formatLSPTypeRef(item.Type),
			)
		}
	}
	for _, fn := range file.Funcs {
		if fn.Synthetic || fn.ExtensionOf != "" || !interfaceDeclPublic(file, fn.Public) {
			continue
		}
		source, ok := interfaceBorrowedReturnExpr(fn)
		if !ok {
			continue
		}
		writeHeader()
		fmt.Fprintf(
			b,
			"// func %s lifetime return=borrow source=%s provenance=param lifetime=call\n",
			formatLSPFuncDetail(fn),
			source,
		)
	}
	if wroteHeader {
		b.WriteByte('\n')
	}
}

func InterfaceOutputPath(inputPath string) string {
	ext := filepath.Ext(inputPath)
	if ext == "" {
		return inputPath + T4InterfaceExtension
	}
	return strings.TrimSuffix(inputPath, ext) + T4InterfaceExtension
}

func writeInterfaceEnum(b *bytes.Buffer, en *frontend.EnumDecl, explicitPublic bool) {
	if explicitPublic {
		b.WriteString("pub ")
	}
	fmt.Fprintf(b, "enum %s:\n", en.Name)
	for _, item := range en.Cases {
		if len(item.Payload) == 0 {
			fmt.Fprintf(b, "    case %s\n", item.Name)
			continue
		}
		payloads := make([]string, 0, len(item.Payload))
		for _, payload := range item.Payload {
			payloads = append(payloads, formatLSPTypeRef(payload))
		}
		fmt.Fprintf(b, "    case %s(%s)\n", item.Name, strings.Join(payloads, ", "))
	}
	b.WriteByte('\n')
}

func writeInterfaceStruct(
	b *bytes.Buffer,
	name string,
	typeParams []string,
	fields []frontend.FieldDecl,
	explicitPublic bool,
) {
	if explicitPublic {
		b.WriteString("pub ")
	}
	fmt.Fprintf(b, "struct %s%s:\n", name, formatLSPTypeParams(typeParams, nil))
	if len(fields) == 0 {
		b.WriteString("    _empty: Int\n\n")
		return
	}
	for _, field := range fields {
		fmt.Fprintf(b, "    %s: %s\n", field.Name, formatLSPTypeRef(field.Type))
	}
	b.WriteByte('\n')
}

func writeInterfaceExtension(b *bytes.Buffer, ext *frontend.ExtensionDecl, explicitPublic bool) {
	if explicitPublic {
		b.WriteString("pub ")
	}
	target := formatLSPTypeRef(ext.Target)
	fmt.Fprintf(b, "extension %s:\n", target)
	for _, method := range ext.Methods {
		if method == nil || method.Synthetic {
			continue
		}
		fmt.Fprintf(b, "    %s:\n", formatInterfaceExtensionMethodDetail(method, target))
		body := strings.TrimSuffix(interfaceFunctionBody(method), "\n")
		for _, line := range strings.Split(body, "\n") {
			fmt.Fprintf(b, "    %s\n", line)
		}
	}
	b.WriteByte('\n')
}

func formatInterfaceExtensionMethodDetail(fn *frontend.FuncDecl, target string) string {
	if fn == nil {
		return ""
	}
	copyFn := *fn
	prefix := target + "."
	if strings.HasPrefix(copyFn.Name, prefix) {
		copyFn.Name = strings.TrimPrefix(copyFn.Name, prefix)
	}
	return formatLSPFuncDetail(&copyFn)
}

func writeInterfaceImpl(b *bytes.Buffer, impl *frontend.ImplDecl) {
	fmt.Fprintf(b, "impl %s\n\n", formatLSPImplName(impl))
}

func interfaceImplPublic(
	file *frontend.FileAST,
	impl *frontend.ImplDecl,
	explicitPublic bool,
) bool {
	if impl == nil {
		return false
	}
	if !explicitPublic {
		return true
	}
	return interfaceTypeRefPublic(file, impl.Type) && interfaceTypeRefPublic(file, impl.Protocol)
}

func interfaceTypeRefPublic(file *frontend.FileAST, ref frontend.TypeRef) bool {
	if file == nil || ref.Name == "" {
		return true
	}
	name := ref.Name
	if file.Module != "" {
		name = strings.TrimPrefix(name, file.Module+".")
	}
	for _, en := range file.Enums {
		if en.Name == name {
			return interfaceDeclPublic(file, en.Public)
		}
	}
	for _, st := range file.Structs {
		if st.Name == name {
			return interfaceDeclPublic(file, st.Public)
		}
	}
	for _, st := range file.States {
		if st.Name == name {
			return interfaceDeclPublic(file, st.Public)
		}
	}
	for _, proto := range file.Protocols {
		if proto.Name == name {
			return interfaceDeclPublic(file, proto.Public)
		}
	}
	return true
}

func interfaceReturnLiteral(ref frontend.TypeRef) string {
	if ref.Kind == frontend.TypeRefOptional {
		return "none"
	}
	name := canonicalLSPTypeName(formatLSPTypeRef(ref))
	switch name {
	case "bool":
		return "false"
	case "str":
		return "\"\""
	default:
		return "0"
	}
}

func interfaceReturnExpr(fn *frontend.FuncDecl) string {
	if expr, ok := interfaceTryReturnExpr(fn); ok {
		return expr
	}
	if expr, ok := interfaceOptionalParamReturnExpr(fn); ok {
		return expr
	}
	if expr, ok := interfaceOptionalAggregateReturnExpr(fn); ok {
		return expr
	}
	if fn.ReturnType.Kind == frontend.TypeRefFunction {
		if expr, ok := interfaceFunctionReturnExpr(fn); ok {
			return expr
		}
		return interfaceFunctionClosureLiteral(fn.ReturnType, "        ")
	}
	if expr, ok := interfaceBorrowedReturnExpr(fn); ok {
		return expr
	}
	if expr, ok := interfaceAggregateReturnExpr(fn); ok {
		return expr
	}
	if expr, ok := interfaceSameTypedParameterReturnExpr(fn); ok {
		return expr
	}
	return interfaceReturnLiteral(fn.ReturnType)
}

func interfaceTryReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || !fn.HasThrows {
		return "", false
	}
	paramNames := map[string]bool{}
	for _, param := range fn.Params {
		paramNames[param.Name] = true
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames); ok {
				aliases[s.Name] = value
			}
			continue
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if !ok {
				continue
			}
			aliases[target.Name] = value
			continue
		}
		ret, ok := stmt.(*frontend.ReturnStmt)
		if !ok {
			continue
		}
		if _, ok := ret.Value.(*frontend.TryExpr); !ok {
			continue
		}
		formatted, ok := interfaceAggregateStubExprWithAliases(ret.Value, aliases)
		if !ok {
			formatted, ok = interfaceContractExpr(ret.Value)
		}
		if !ok ||
			(!interfaceExprRefsAnyParam(
				ret.Value,
				paramNames,
			) && !interfaceExprRefsAnyAlias(
				ret.Value,
				aliases,
			)) {
			return "", false
		}
		return formatted, true
	}
	return "", false
}

func interfaceOptionalParamReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || fn.ReturnType.Kind != frontend.TypeRefOptional || fn.ReturnType.Elem == nil {
		return "", false
	}
	elemType := formatLSPTypeRef(*fn.ReturnType.Elem)
	paramNames := map[string]bool{}
	paramHasElemType := map[string]bool{}
	for _, param := range fn.Params {
		paramNames[param.Name] = true
		if formatLSPTypeRef(param.Type) == elemType {
			paramHasElemType[param.Name] = true
		}
	}
	optionalLocals := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if formatLSPTypeRef(s.Type) == formatLSPTypeRef(fn.ReturnType) {
				optionalLocals[s.Name] = ""
				if value, ok := s.Value.(*frontend.IdentExpr); ok && paramHasElemType[value.Name] {
					optionalLocals[s.Name] = value.Name
				} else if value, ok := interfaceParamPathExpr(s.Value, optionalLocals, paramNames); ok {
					optionalLocals[s.Name] = value
				}
			}
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := optionalLocals[target.Name]; !ok {
				continue
			}
			value, ok := interfaceParamPathExpr(s.Value, optionalLocals, paramNames)
			if !ok {
				continue
			}
			optionalLocals[target.Name] = value
		case *frontend.ReturnStmt:
			id, ok := s.Value.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if paramName := optionalLocals[id.Name]; paramName != "" {
				return paramName, true
			}
		case *frontend.IfLetStmt:
			value, ok := interfaceParamPathExpr(s.Value, optionalLocals, paramNames)
			if !ok || s.Name == "" {
				continue
			}
			branchAliases := interfaceAliasMapCopy(optionalLocals)
			branchAliases[s.Name] = value
			if expr, ok := interfaceOptionalReturnFromStmts(s.Then, branchAliases); ok {
				return expr, true
			}
		case *frontend.MatchStmt:
			value, ok := interfaceParamPathExpr(s.Value, optionalLocals, paramNames)
			if !ok {
				continue
			}
			for _, c := range s.Cases {
				name, ok := interfaceOptionalSomePatternName(c.Pattern)
				if !ok {
					continue
				}
				branchAliases := interfaceAliasMapCopy(optionalLocals)
				branchAliases[name] = value
				if expr, ok := interfaceOptionalReturnFromStmts(c.Body, branchAliases); ok {
					return expr, true
				}
			}
		}
	}
	return "", false
}

func interfaceOptionalAggregateReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || fn.ReturnType.Kind != frontend.TypeRefOptional {
		return "", false
	}
	paramNames := map[string]bool{}
	for _, param := range fn.Params {
		paramNames[param.Name] = true
	}
	aliases := map[string]string{}
	optionalAggregates := map[string]string{}
	return interfaceOptionalAggregateReturnFromStmts(
		fn.Body,
		aliases,
		optionalAggregates,
		paramNames,
		formatLSPTypeRef(fn.ReturnType),
	)
}

func interfaceOptionalAggregateReturnFromStmts(
	stmts []frontend.Stmt,
	aliases, optionalAggregates map[string]string,
	params map[string]bool,
	returnType string,
) (string, bool) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if formatLSPTypeRef(s.Type) == returnType {
				optionalAggregates[s.Name] = ""
				if value, ok := interfaceOptionalAggregateExpr(s.Value, aliases, params); ok {
					optionalAggregates[s.Name] = value
				}
				continue
			}
			if value, ok := interfaceParamPathExpr(s.Value, aliases, params); ok {
				aliases[s.Name] = value
			}
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := optionalAggregates[target.Name]; ok {
				value, ok := interfaceOptionalAggregateExpr(s.Value, aliases, params)
				if ok {
					optionalAggregates[target.Name] = value
				}
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			if value, ok := interfaceParamPathExpr(s.Value, aliases, params); ok {
				aliases[target.Name] = value
			}
		case *frontend.ReturnStmt:
			id, ok := s.Value.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if value := optionalAggregates[id.Name]; value != "" {
				return value, true
			}
		case *frontend.IfStmt:
			thenAliases := interfaceAliasMapCopy(aliases)
			thenAggregates := interfaceAliasMapCopy(optionalAggregates)
			thenExpr, thenReturned := interfaceOptionalAggregateReturnFromStmts(
				s.Then,
				thenAliases,
				thenAggregates,
				params,
				returnType,
			)

			elseAliases := interfaceAliasMapCopy(aliases)
			elseAggregates := interfaceAliasMapCopy(optionalAggregates)
			elseExpr, elseReturned := interfaceOptionalAggregateReturnFromStmts(
				s.Else,
				elseAliases,
				elseAggregates,
				params,
				returnType,
			)

			if thenReturned && elseReturned && thenExpr == elseExpr {
				return thenExpr, true
			}
			if !thenReturned && !elseReturned {
				interfaceMergeEqualAliasState(aliases, thenAliases, elseAliases)
				interfaceMergeEqualOptionalAggregateState(optionalAggregates, thenAggregates, elseAggregates)
			}
		case *frontend.IfLetStmt:
			thenAliases := interfaceAliasMapCopy(aliases)
			if value, ok := interfaceParamPathExpr(s.Value, aliases, params); ok && s.Name != "" {
				thenAliases[s.Name] = value
			}
			thenAggregates := interfaceAliasMapCopy(optionalAggregates)
			thenExpr, thenReturned := interfaceOptionalAggregateReturnFromStmts(
				s.Then,
				thenAliases,
				thenAggregates,
				params,
				returnType,
			)

			elseAliases := interfaceAliasMapCopy(aliases)
			elseAggregates := interfaceAliasMapCopy(optionalAggregates)
			elseExpr, elseReturned := interfaceOptionalAggregateReturnFromStmts(
				s.Else,
				elseAliases,
				elseAggregates,
				params,
				returnType,
			)

			if thenReturned && elseReturned && thenExpr == elseExpr {
				return thenExpr, true
			}
			if !thenReturned && !elseReturned {
				interfaceMergeEqualAliasState(aliases, thenAliases, elseAliases)
				interfaceMergeEqualOptionalAggregateState(optionalAggregates, thenAggregates, elseAggregates)
			}
		case *frontend.MatchStmt:
			if expr, ok := interfaceOptionalAggregateMatchReturnExpr(
				s,
				aliases,
				optionalAggregates,
				params,
				returnType,
			); ok {
				return expr, true
			}
		}
	}
	return "", false
}

func interfaceBorrowedReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || fn.ReturnOwnership != "borrow" {
		return "", false
	}
	params := map[string]bool{}
	paramOrder := []string{}
	returnType := formatLSPTypeRef(fn.ReturnType)
	for _, param := range fn.Params {
		if param.Ownership == "borrow" {
			params[param.Name] = true
			if formatLSPTypeRef(param.Type) == returnType {
				paramOrder = append(paramOrder, param.Name)
			}
		}
	}
	if len(params) == 0 {
		return "", false
	}
	for _, stmt := range fn.Body {
		ret, ok := stmt.(*frontend.ReturnStmt)
		if !ok {
			continue
		}
		if source, ok := interfaceBorrowedSourceParamExpr(ret.Value, params); ok {
			return source, true
		}
	}
	if len(paramOrder) == 1 {
		return paramOrder[0], true
	}
	return "", false
}

func interfaceBorrowedSourceParamExpr(expr frontend.Expr, params map[string]bool) (string, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if params[e.Name] {
			return e.Name, true
		}
	case *frontend.FieldAccessExpr:
		return interfaceBorrowedSourceParamExpr(e.Base, params)
	case *frontend.CallExpr:
		method, ok := interfaceBorrowedViewMethod(e.Name)
		if ok && len(e.Args) > 0 {
			switch method {
			case "borrow", "window", "prefix", "suffix":
				return interfaceBorrowedSourceParamExpr(e.Args[0], params)
			}
		}
		return interfaceBorrowedSourceParamMethodCall(e.Name, params)
	}
	return "", false
}

func interfaceBorrowedSourceParamMethodCall(name string, params map[string]bool) (string, bool) {
	idx := strings.LastIndex(name, ".")
	if idx < 0 {
		return "", false
	}
	receiver := name[:idx]
	method := name[idx+1:]
	switch method {
	case "borrow", "window", "prefix", "suffix":
	default:
		return "", false
	}
	root := receiver
	if dot := strings.Index(root, "."); dot >= 0 {
		root = root[:dot]
	}
	if params[root] {
		return root, true
	}
	return "", false
}

func interfaceBorrowedViewMethod(name string) (string, bool) {
	if strings.HasPrefix(name, "__method.") {
		method := strings.TrimPrefix(name, "__method.")
		switch method {
		case "borrow", "window", "prefix", "suffix":
			return method, true
		}
	}
	if name == "core.string_borrow" {
		return "borrow", true
	}
	if strings.HasPrefix(name, "core.slice_borrow_") {
		return "borrow", true
	}
	if name == "core.string_window" || name == "core.string_prefix" ||
		name == "core.string_suffix" {
		return strings.TrimPrefix(name, "core.string_"), true
	}
	if strings.HasPrefix(name, "core.slice_window_") {
		return "window", true
	}
	if strings.HasPrefix(name, "core.slice_prefix_") {
		return "prefix", true
	}
	if strings.HasPrefix(name, "core.slice_suffix_") {
		return "suffix", true
	}
	return "", false
}

func interfaceOptionalAggregateMatchReturnExpr(
	match *frontend.MatchStmt,
	aliases, optionalAggregates map[string]string,
	params map[string]bool,
	returnType string,
) (string, bool) {
	if match == nil || len(match.Cases) == 0 {
		return "", false
	}
	var commonExpr string
	allReturned := true
	caseAliases := make([]map[string]string, 0, len(match.Cases))
	caseAggregates := make([]map[string]string, 0, len(match.Cases))
	for _, c := range match.Cases {
		if c.Guard != nil {
			return "", false
		}
		branchAliases := interfaceAliasMapCopy(aliases)
		branchAggregates := interfaceAliasMapCopy(optionalAggregates)
		expr, returned := interfaceOptionalAggregateReturnFromStmts(
			c.Body,
			branchAliases,
			branchAggregates,
			params,
			returnType,
		)
		if returned {
			if commonExpr == "" {
				commonExpr = expr
			} else if commonExpr != expr {
				return "", false
			}
		} else {
			allReturned = false
		}
		caseAliases = append(caseAliases, branchAliases)
		caseAggregates = append(caseAggregates, branchAggregates)
	}
	if allReturned && commonExpr != "" {
		return commonExpr, true
	}
	if !allReturned && commonExpr == "" {
		interfaceMergeEqualAliasStateAcross(aliases, caseAliases)
		interfaceMergeEqualOptionalAggregateStateAcross(optionalAggregates, caseAggregates)
	}
	return "", false
}

func interfaceMergeEqualAliasState(dst, left, right map[string]string) {
	for key := range dst {
		if left[key] == right[key] {
			dst[key] = left[key]
			continue
		}
		delete(dst, key)
	}
}

func interfaceMergeEqualOptionalAggregateState(dst, left, right map[string]string) {
	for key := range dst {
		value, ok := interfaceMergeOptionalAggregateValue(left[key], right[key])
		if ok {
			dst[key] = value
			continue
		}
		dst[key] = ""
	}
}

func interfaceMergeOptionalAggregateValue(values ...string) (string, bool) {
	merged := ""
	for _, value := range values {
		if value == "" {
			continue
		}
		if merged == "" {
			merged = value
			continue
		}
		if merged != value {
			return "", false
		}
	}
	return merged, true
}

func interfaceMergeEqualAliasStateAcross(dst map[string]string, states []map[string]string) {
	for key := range dst {
		value, ok := interfaceCommonStateValue(key, states)
		if ok {
			dst[key] = value
			continue
		}
		delete(dst, key)
	}
}

func interfaceMergeEqualOptionalAggregateStateAcross(
	dst map[string]string,
	states []map[string]string,
) {
	for key := range dst {
		values := make([]string, 0, len(states))
		for _, state := range states {
			values = append(values, state[key])
		}
		value, ok := interfaceMergeOptionalAggregateValue(values...)
		if ok {
			dst[key] = value
			continue
		}
		dst[key] = ""
	}
}

func interfaceCommonStateValue(key string, states []map[string]string) (string, bool) {
	if len(states) == 0 {
		return "", false
	}
	value, ok := states[0][key]
	if !ok {
		return "", false
	}
	for _, state := range states[1:] {
		if state[key] != value {
			return "", false
		}
	}
	return value, true
}

func interfaceOptionalAggregateExpr(
	expr frontend.Expr,
	aliases map[string]string,
	params map[string]bool,
) (string, bool) {
	if !interfaceDirectAggregateExpr(expr) {
		return "", false
	}
	formatted, ok := interfaceAggregateStubExprWithAliases(expr, aliases)
	if !ok {
		return "", false
	}
	if !interfaceExprRefsAnyParam(expr, params) && !interfaceExprRefsAnyAlias(expr, aliases) {
		return "", false
	}
	return formatted, true
}

func interfaceOptionalReturnFromStmts(
	stmts []frontend.Stmt,
	aliases map[string]string,
) (string, bool) {
	for _, stmt := range stmts {
		ret, ok := stmt.(*frontend.ReturnStmt)
		if !ok {
			continue
		}
		id, ok := ret.Value.(*frontend.IdentExpr)
		if !ok {
			continue
		}
		if paramName := aliases[id.Name]; paramName != "" {
			return paramName, true
		}
	}
	return "", false
}

func interfaceFunctionBody(fn *frontend.FuncDecl) string {
	if body, ok := interfaceFunctionMatchReturnBody(fn); ok {
		return body
	}
	if body, ok := interfaceReturnedClosureCaptureBody(fn); ok {
		return body
	}
	if expr, ok := interfaceThrowExpr(fn); ok {
		return "    throw " + expr
	}
	if body, ok := interfaceBorrowedReturnBody(fn); ok {
		return body
	}
	return "    return " + interfaceReturnExpr(fn)
}

func interfaceBorrowedReturnBody(fn *frontend.FuncDecl) (string, bool) {
	source, ok := interfaceBorrowedReturnExpr(fn)
	if !ok {
		return "", false
	}
	return fmt.Sprintf(
		("    // tetra-interface-lifetime: return=borrow source=%s " +
			"provenance=param lifetime=call\n    return %s"),
		source,
		source,
	), true
}

// ---- interface_returns.go ----

type interfaceCaptureStub struct {
	Name    string
	Type    frontend.TypeRef
	Mutable bool
}

func interfaceThrowExpr(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || !fn.HasThrows {
		return "", false
	}
	paramNames := map[string]bool{}
	for _, param := range fn.Params {
		paramNames[param.Name] = true
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames); ok {
				aliases[s.Name] = value
			}
			continue
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if !ok {
				continue
			}
			aliases[target.Name] = value
			continue
		}
		throwStmt, ok := stmt.(*frontend.ThrowStmt)
		if !ok {
			continue
		}
		formatted, ok := interfaceAggregateStubExprWithAliases(throwStmt.Value, aliases)
		if !ok {
			formatted, ok = interfaceContractExpr(throwStmt.Value)
		}
		if !ok ||
			(!interfaceExprRefsAnyParam(
				throwStmt.Value,
				paramNames,
			) && !interfaceExprRefsAnyAlias(
				throwStmt.Value,
				aliases,
			)) {
			return "", false
		}
		return formatted, true
	}
	return "", false
}

func interfaceContractExpr(expr frontend.Expr) (string, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name, true
	case *frontend.NumberExpr:
		return fmt.Sprintf("%d", e.Value), true
	case *frontend.BoolLitExpr:
		if e.Value {
			return "true", true
		}
		return "false", true
	case *frontend.NoneLitExpr:
		return "none", true
	case *frontend.StringLitExpr:
		return fmt.Sprintf("%q", string(e.Value)), true
	case *frontend.TryExpr:
		inner, ok := interfaceContractExpr(e.X)
		if !ok {
			return "", false
		}
		return "try " + inner, true
	case *frontend.FieldAccessExpr:
		base, ok := interfaceContractExpr(e.Base)
		if !ok {
			return "", false
		}
		return base + "." + e.Field, true
	case *frontend.CallExpr:
		args := make([]string, 0, len(e.Args))
		for i, arg := range e.Args {
			formatted, ok := interfaceContractExpr(arg)
			if !ok {
				return "", false
			}
			if i < len(e.ArgLabels) && e.ArgLabels[i] != "" {
				formatted = e.ArgLabels[i] + ": " + formatted
			}
			args = append(args, formatted)
		}
		return e.Name + "(" + strings.Join(args, ", ") + ")", true
	case *frontend.StructLitExpr:
		fields := make([]string, 0, len(e.Fields))
		for _, field := range e.Fields {
			formatted, ok := interfaceContractExpr(field.Value)
			if !ok {
				return "", false
			}
			fields = append(fields, field.Name+": "+formatted)
		}
		return formatLSPTypeRef(e.Type) + "(" + strings.Join(fields, ", ") + ")", true
	default:
		return "", false
	}
}

func interfaceExprRefsAnyParam(expr frontend.Expr, params map[string]bool) bool {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return params[e.Name]
	case *frontend.FieldAccessExpr:
		return interfaceExprRefsAnyParam(e.Base, params)
	case *frontend.TryExpr:
		return interfaceExprRefsAnyParam(e.X, params)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			if interfaceExprRefsAnyParam(arg, params) {
				return true
			}
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if interfaceExprRefsAnyParam(field.Value, params) {
				return true
			}
		}
	}
	return false
}

func interfaceExprRefsAnyAlias(expr frontend.Expr, aliases map[string]string) bool {
	if len(aliases) == 0 {
		return false
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return aliases[e.Name] != ""
	case *frontend.FieldAccessExpr:
		return interfaceExprRefsAnyAlias(e.Base, aliases)
	case *frontend.TryExpr:
		return interfaceExprRefsAnyAlias(e.X, aliases)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			if interfaceExprRefsAnyAlias(arg, aliases) {
				return true
			}
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if interfaceExprRefsAnyAlias(field.Value, aliases) {
				return true
			}
		}
	}
	return false
}

func interfaceReturnedClosureCaptureBody(fn *frontend.FuncDecl) (string, bool) {
	if fn.ReturnType.Kind != frontend.TypeRefFunction {
		return "", false
	}
	outerLocals := map[string]interfaceCaptureStub{}
	outerOrder := []string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if _, exists := outerLocals[s.Name]; !exists {
				outerOrder = append(outerOrder, s.Name)
			}
			outerLocals[s.Name] = interfaceCaptureStub{Name: s.Name, Type: s.Type, Mutable: s.Mutable}
		case *frontend.ReturnStmt:
			closure, ok := s.Value.(*frontend.ClosureExpr)
			if !ok || closure.Decl == nil {
				return "", false
			}
			used := map[string]bool{}
			interfaceCollectStmtIdents(closure.Decl.Body, used)
			for _, param := range closure.Decl.Params {
				delete(used, param.Name)
			}
			for local := range interfaceLocalNames(closure.Decl.Body) {
				delete(used, local)
			}
			captures := make([]interfaceCaptureStub, 0, len(outerOrder))
			for _, name := range outerOrder {
				if used[name] {
					captures = append(captures, outerLocals[name])
				}
			}
			if len(captures) == 0 {
				return "", false
			}
			var b strings.Builder
			for _, capture := range captures {
				decl := "let"
				if capture.Mutable {
					decl = "var"
				}
				fmt.Fprintf(
					&b,
					"    %s %s: %s = %s\n",
					decl,
					capture.Name,
					formatLSPTypeRef(capture.Type),
					interfaceReturnLiteral(capture.Type),
				)
			}
			fmt.Fprintf(&b, "    return %s", interfaceCapturedClosureLiteral(closure, captures, "        "))
			return b.String(), true
		}
	}
	return "", false
}

func interfaceCapturedClosureLiteral(
	closure *frontend.ClosureExpr,
	captures []interfaceCaptureStub,
	bodyIndent string,
) string {
	params := make([]string, 0, len(closure.Decl.Params))
	for _, param := range closure.Decl.Params {
		formatted := formatLSPTypeRef(param.Type)
		if param.Ownership != "" {
			formatted = param.Ownership + " " + formatted
		}
		params = append(params, fmt.Sprintf("%s: %s", param.Name, formatted))
	}
	ret := formatLSPTypeRef(closure.Decl.ReturnType)
	out := "fn(" + strings.Join(params, ", ") + ") -> " + ret
	if closure.Decl.HasThrows {
		out += " throws " + formatLSPTypeRef(closure.Decl.Throws)
	}
	if len(closure.Decl.Uses) > 0 {
		uses := append([]string(nil), closure.Decl.Uses...)
		sort.Strings(uses)
		out += " uses " + strings.Join(uses, ", ")
	}
	var b strings.Builder
	b.WriteString(out)
	b.WriteString(":\n")
	for i, capture := range captures {
		fmt.Fprintf(
			&b,
			"%slet __capture_keep%d: %s = %s\n",
			bodyIndent,
			i,
			formatLSPTypeRef(capture.Type),
			capture.Name,
		)
	}
	fmt.Fprintf(&b, "%sreturn %s", bodyIndent, interfaceReturnLiteral(closure.Decl.ReturnType))
	return b.String()
}

func interfaceLocalNames(stmts []frontend.Stmt) map[string]bool {
	names := map[string]bool{}
	for _, stmt := range stmts {
		if let, ok := stmt.(*frontend.LetStmt); ok {
			names[let.Name] = true
		}
	}
	return names
}

func interfaceCollectStmtIdents(stmts []frontend.Stmt, used map[string]bool) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			interfaceCollectExprIdents(s.Value, used)
		case *frontend.LetStmt:
			interfaceCollectExprIdents(s.Value, used)
		case *frontend.ExprStmt:
			interfaceCollectExprIdents(s.Expr, used)
		case *frontend.IfStmt:
			interfaceCollectExprIdents(s.Cond, used)
			interfaceCollectStmtIdents(s.Then, used)
			interfaceCollectStmtIdents(s.Else, used)
		case *frontend.MatchStmt:
			interfaceCollectExprIdents(s.Value, used)
			for _, c := range s.Cases {
				interfaceCollectExprIdents(c.Guard, used)
				interfaceCollectStmtIdents(c.Body, used)
			}
		}
	}
}

func interfaceCollectExprIdents(expr frontend.Expr, used map[string]bool) {
	switch e := expr.(type) {
	case nil:
		return
	case *frontend.IdentExpr:
		used[e.Name] = true
	case *frontend.BinaryExpr:
		interfaceCollectExprIdents(e.Left, used)
		interfaceCollectExprIdents(e.Right, used)
	case *frontend.UnaryExpr:
		interfaceCollectExprIdents(e.X, used)
	case *frontend.TryExpr:
		interfaceCollectExprIdents(e.X, used)
	case *frontend.AwaitExpr:
		interfaceCollectExprIdents(e.X, used)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			interfaceCollectExprIdents(arg, used)
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			interfaceCollectExprIdents(field.Value, used)
		}
	case *frontend.FieldAccessExpr:
		interfaceCollectExprIdents(e.Base, used)
	case *frontend.IndexExpr:
		interfaceCollectExprIdents(e.Base, used)
		interfaceCollectExprIdents(e.Index, used)
	case *frontend.MatchExpr:
		interfaceCollectExprIdents(e.Value, used)
		for _, c := range e.Cases {
			interfaceCollectExprIdents(c.Guard, used)
			interfaceCollectExprIdents(c.Value, used)
		}
	case *frontend.CatchExpr:
		interfaceCollectExprIdents(e.Call, used)
		for _, c := range e.Cases {
			interfaceCollectExprIdents(c.Guard, used)
			interfaceCollectExprIdents(c.Value, used)
		}
	}
}

func interfaceAggregateReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	paramNames := map[string]bool{}
	paramTypes := map[string]string{}
	for _, param := range fn.Params {
		formatted := formatLSPTypeRef(param.Type)
		paramNames[param.Name] = true
		paramTypes[param.Name] = formatted
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if s.Type.Kind == frontend.TypeRefOptional {
				aliases[s.Name] = ""
				if value, ok := s.Value.(*frontend.IdentExpr); ok && paramNames[value.Name] {
					aliases[s.Name] = value.Name
				} else if value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames); ok {
					aliases[s.Name] = value
				}
			} else if value, ok := s.Value.(*frontend.IdentExpr); ok &&
				paramTypes[value.Name] == formatLSPTypeRef(s.Type) {
				aliases[s.Name] = value.Name
			} else if value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames); ok {
				aliases[s.Name] = value
			}
			continue
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if !ok {
				continue
			}
			aliases[target.Name] = value
			continue
		case *frontend.IfLetStmt:
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if !ok || s.Name == "" {
				continue
			}
			branchAliases := interfaceAliasMapCopy(aliases)
			branchAliases[s.Name] = value
			if expr, ok := interfaceAggregateReturnFromBranches(
				s.Then,
				branchAliases,
				s.Else,
				aliases,
				paramNames,
			); ok {
				return expr, true
			}
			continue
		case *frontend.MatchStmt:
			value, ok := interfaceParamPathExpr(s.Value, aliases, paramNames)
			if ok {
				for _, c := range s.Cases {
					name, ok := interfaceOptionalSomePatternName(c.Pattern)
					if !ok {
						continue
					}
					branchAliases := interfaceAliasMapCopy(aliases)
					branchAliases[name] = value
					if expr, ok := interfaceAggregateReturnFromStmts(c.Body, branchAliases, paramNames); ok {
						return expr, true
					}
				}
			}
		}
		if expr, ok := interfaceAggregateReturnFromStmts([]frontend.Stmt{stmt}, aliases, paramNames); ok {
			return expr, true
		}
	}
	return "", false
}

func interfaceAggregateReturnFromStmts(
	stmts []frontend.Stmt,
	aliases map[string]string,
	params map[string]bool,
) (string, bool) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			if !interfaceDirectAggregateExpr(s.Value) {
				continue
			}
			if !interfaceExprContainsClosure(
				s.Value,
			) && !interfaceExprRefsAnyParam(
				s.Value,
				params,
			) && !interfaceExprRefsAnyAlias(
				s.Value,
				aliases,
			) {
				continue
			}
			expr, ok := interfaceAggregateStubExprWithAliases(s.Value, aliases)
			if ok {
				return expr, true
			}
		case *frontend.IfStmt:
			if expr, ok := interfaceAggregateReturnFromBranches(
				s.Then,
				aliases,
				s.Else,
				aliases,
				params,
			); ok {
				return expr, true
			}
		case *frontend.MatchStmt:
			if expr, ok := interfaceAggregateMatchReturnExpr(s, aliases, params); ok {
				return expr, true
			}
		}
	}
	return "", false
}

func interfaceAggregateReturnFromBranches(
	thenStmts []frontend.Stmt,
	thenAliases map[string]string,
	elseStmts []frontend.Stmt,
	elseAliases map[string]string,
	params map[string]bool,
) (string, bool) {
	thenExpr, thenOK := interfaceAggregateReturnFromStmts(thenStmts, thenAliases, params)
	elseExpr, elseOK := interfaceAggregateReturnFromStmts(elseStmts, elseAliases, params)
	if thenOK && elseOK {
		if thenExpr == elseExpr {
			return thenExpr, true
		}
		return "", false
	}
	if thenOK {
		return thenExpr, true
	}
	if elseOK {
		return elseExpr, true
	}
	return "", false
}

func interfaceAggregateMatchReturnExpr(
	match *frontend.MatchStmt,
	aliases map[string]string,
	params map[string]bool,
) (string, bool) {
	if match == nil || len(match.Cases) == 0 {
		return "", false
	}
	var commonExpr string
	for _, c := range match.Cases {
		if c.Guard != nil {
			return "", false
		}
		expr, ok := interfaceAggregateReturnFromStmts(c.Body, aliases, params)
		if !ok {
			continue
		}
		if commonExpr == "" {
			commonExpr = expr
			continue
		}
		if commonExpr != expr {
			return "", false
		}
	}
	return commonExpr, commonExpr != ""
}

func interfaceOptionalSomePatternName(expr frontend.Expr) (string, bool) {
	if some, ok := expr.(*frontend.SomePatternExpr); ok {
		return some.Name, some.Name != ""
	}
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call.Name != "some" || len(call.Args) != 1 {
		return "", false
	}
	id, ok := call.Args[0].(*frontend.IdentExpr)
	if !ok || id.Name == "" {
		return "", false
	}
	return id.Name, true
}

func interfaceAliasMapCopy(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func interfaceParamPathExpr(
	expr frontend.Expr,
	aliases map[string]string,
	params map[string]bool,
) (string, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if alias := aliases[e.Name]; alias != "" {
			return alias, true
		}
		if params[e.Name] {
			return e.Name, true
		}
	case *frontend.FieldAccessExpr:
		base, ok := interfaceParamPathExpr(e.Base, aliases, params)
		if !ok {
			return "", false
		}
		return base + "." + e.Field, true
	}
	return "", false
}

func interfaceDirectAggregateExpr(expr frontend.Expr) bool {
	switch expr.(type) {
	case *frontend.CallExpr, *frontend.StructLitExpr:
		return true
	default:
		return false
	}
}

func interfaceExprContainsClosure(expr frontend.Expr) bool {
	switch e := expr.(type) {
	case *frontend.ClosureExpr:
		return true
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			if interfaceExprContainsClosure(arg) {
				return true
			}
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if interfaceExprContainsClosure(field.Value) {
				return true
			}
		}
	}
	return false
}

func interfaceAggregateStubExpr(expr frontend.Expr) (string, bool) {
	return interfaceAggregateStubExprWithAliases(expr, nil)
}

func interfaceAggregateStubExprWithAliases(
	expr frontend.Expr,
	aliases map[string]string,
) (string, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if aliases != nil {
			if alias := aliases[e.Name]; alias != "" {
				return alias, true
			}
		}
		return e.Name, true
	case *frontend.TryExpr:
		inner, ok := interfaceAggregateStubExprWithAliases(e.X, aliases)
		if !ok {
			return "", false
		}
		return "try " + inner, true
	case *frontend.CallExpr:
		args := make([]string, 0, len(e.Args))
		for i, arg := range e.Args {
			formatted, ok := interfaceAggregateStubExprWithAliases(arg, aliases)
			if !ok {
				return "", false
			}
			if i < len(e.ArgLabels) && e.ArgLabels[i] != "" {
				formatted = e.ArgLabels[i] + ": " + formatted
			}
			args = append(args, formatted)
		}
		return e.Name + "(" + strings.Join(args, ", ") + ")", true
	case *frontend.StructLitExpr:
		fields := make([]string, 0, len(e.Fields))
		for _, field := range e.Fields {
			formatted, ok := interfaceAggregateStubExprWithAliases(field.Value, aliases)
			if !ok {
				return "", false
			}
			fields = append(fields, field.Name+": "+formatted)
		}
		return formatLSPTypeRef(e.Type) + "(" + strings.Join(fields, ", ") + ")", true
	case *frontend.FieldAccessExpr:
		base, ok := interfaceAggregateStubExprWithAliases(e.Base, aliases)
		if !ok {
			return "", false
		}
		return base + "." + e.Field, true
	case *frontend.ClosureExpr:
		ref, ok := interfaceClosureTypeRef(e)
		if !ok {
			return "", false
		}
		return interfaceInlineFunctionClosureLiteral(ref), true
	case *frontend.NumberExpr:
		return fmt.Sprintf("%d", e.Value), true
	case *frontend.BoolLitExpr:
		if e.Value {
			return "true", true
		}
		return "false", true
	case *frontend.NoneLitExpr:
		return "none", true
	default:
		return "", false
	}
}

func interfaceClosureTypeRef(closure *frontend.ClosureExpr) (frontend.TypeRef, bool) {
	if closure == nil || closure.Decl == nil {
		return frontend.TypeRef{}, false
	}
	params := make([]frontend.TypeRef, 0, len(closure.Decl.Params))
	ownership := make([]string, 0, len(closure.Decl.Params))
	for _, param := range closure.Decl.Params {
		params = append(params, param.Type)
		ownership = append(ownership, param.Ownership)
	}
	ret := closure.Decl.ReturnType
	ref := frontend.TypeRef{
		Kind:           frontend.TypeRefFunction,
		Params:         params,
		ParamOwnership: ownership,
		Return:         &ret,
		Uses:           append([]string(nil), closure.Decl.Uses...),
	}
	if closure.Decl.HasThrows {
		throws := closure.Decl.Throws
		ref.Throws = &throws
	}
	return ref, true
}

func interfaceSameTypedParameterReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	returnSig := formatLSPTypeRef(fn.ReturnType)
	sameTypedParams := map[string]bool{}
	for _, param := range fn.Params {
		if formatLSPTypeRef(param.Type) == returnSig {
			sameTypedParams[param.Name] = true
		}
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		if let, ok := stmt.(*frontend.LetStmt); ok {
			if formatLSPTypeRef(let.Type) != returnSig {
				continue
			}
			if id, ok := let.Value.(*frontend.IdentExpr); ok && sameTypedParams[id.Name] {
				aliases[let.Name] = id.Name
			}
			continue
		}
		ret, ok := stmt.(*frontend.ReturnStmt)
		if !ok {
			continue
		}
		id, ok := ret.Value.(*frontend.IdentExpr)
		if ok && sameTypedParams[id.Name] {
			return id.Name, true
		}
		if ok {
			if param := aliases[id.Name]; param != "" {
				return param, true
			}
		}
	}
	return "", false
}

func interfaceFunctionReturnExpr(fn *frontend.FuncDecl) (string, bool) {
	returnSig := formatLSPTypeRef(fn.ReturnType)
	functionParams := map[string]bool{}
	valueParams := map[string]bool{}
	for _, param := range fn.Params {
		valueParams[param.Name] = true
		if param.Type.Kind == frontend.TypeRefFunction &&
			formatLSPTypeRef(param.Type) == returnSig {
			functionParams[param.Name] = true
		}
	}
	aliases := map[string]string{}
	for _, stmt := range fn.Body {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if formatLSPTypeRef(s.Type) != returnSig {
				continue
			}
			if path, ok := interfaceFunctionReturnParamPath(
				s.Value,
				aliases,
				functionParams,
				valueParams,
			); ok {
				aliases[s.Name] = path
			}
			continue
		case *frontend.AssignStmt:
			target, ok := s.Target.(*frontend.IdentExpr)
			if !ok {
				continue
			}
			if _, ok := aliases[target.Name]; !ok {
				continue
			}
			if path, ok := interfaceFunctionReturnParamPath(
				s.Value,
				aliases,
				functionParams,
				valueParams,
			); ok {
				aliases[target.Name] = path
			}
			continue
		case *frontend.ReturnStmt:
			if path, ok := interfaceFunctionReturnParamPath(
				s.Value,
				aliases,
				functionParams,
				valueParams,
			); ok {
				return path, true
			}
		}
	}
	return "", false
}

func interfaceFunctionMatchReturnBody(fn *frontend.FuncDecl) (string, bool) {
	if fn == nil || fn.ReturnType.Kind != frontend.TypeRefFunction {
		return "", false
	}
	paramTypes := map[string]string{}
	for _, param := range fn.Params {
		paramTypes[param.Name] = formatLSPTypeRef(param.Type)
	}
	for _, stmt := range fn.Body {
		match, ok := stmt.(*frontend.MatchStmt)
		if !ok || match.Value == nil {
			continue
		}
		valueName := interfaceCallbackArgumentName(match.Value)
		if valueName == "" {
			continue
		}
		valueType := paramTypes[valueName]
		if valueType == "" {
			continue
		}
		var b strings.Builder
		fmt.Fprintf(&b, "    match %s:\n", valueName)
		preservedPayload := false
		for _, c := range match.Cases {
			if c.Guard != nil {
				return "", false
			}
			binding, hasBinding := interfacePatternBindingName(c.Pattern)
			pattern := "_"
			if !c.Default {
				pattern = interfaceFunctionMatchPattern(c.Pattern, valueType)
			}
			fmt.Fprintf(&b, "    case %s:\n", pattern)
			ret, ok := singleReturnExpr(c.Body)
			if !ok {
				return "", false
			}
			if id, ok := ret.(*frontend.IdentExpr); ok && hasBinding && id.Name == binding {
				fmt.Fprintf(&b, "        return %s\n", id.Name)
				preservedPayload = true
				continue
			}
			if expr, ok := interfaceContractExpr(ret); ok && expr != "" {
				fmt.Fprintf(&b, "        return %s\n", expr)
				continue
			}
			fmt.Fprintf(
				&b,
				"        return %s\n",
				interfaceFunctionClosureLiteral(fn.ReturnType, "            "),
			)
		}
		if preservedPayload {
			return strings.TrimRight(b.String(), "\n"), true
		}
	}
	return "", false
}

func interfacePatternBindingName(expr frontend.Expr) (string, bool) {
	switch e := expr.(type) {
	case *frontend.SomePatternExpr:
		return e.Name, e.Name != ""
	case *frontend.EnumCasePatternExpr:
		if len(e.Bindings) == 0 || e.Bindings[0] == "" {
			return "", false
		}
		return e.Bindings[0], true
	case *frontend.CallExpr:
		if len(e.Args) == 0 {
			return "", false
		}
		id, ok := e.Args[0].(*frontend.IdentExpr)
		if !ok || id.Name == "" {
			return "", false
		}
		return id.Name, true
	default:
		return "", false
	}
}

func interfaceFunctionMatchPattern(expr frontend.Expr, enumType string) string {
	switch e := expr.(type) {
	case *frontend.SomePatternExpr:
		if enumType != "" && !strings.HasSuffix(enumType, "?") {
			return enumType + ".some(" + e.Name + ")"
		}
	case *frontend.CallExpr:
		if enumType != "" && len(e.Args) > 0 {
			names := make([]string, 0, len(e.Args))
			for _, arg := range e.Args {
				id, ok := arg.(*frontend.IdentExpr)
				if !ok {
					return interfaceFormatExpr(expr)
				}
				names = append(names, id.Name)
			}
			return enumType + "." + interfaceShortName(e.Name) + "(" + strings.Join(names, ", ") + ")"
		}
	case *frontend.IdentExpr:
		if enumType != "" && !strings.HasSuffix(enumType, "?") {
			return enumType + "." + e.Name
		}
	case *frontend.EnumCasePatternExpr:
		if e.TypeName == "" && enumType != "" {
			if e.HasPayload {
				return enumType + "." + e.CaseName + "(" + strings.Join(e.Bindings, ", ") + ")"
			}
			return enumType + "." + e.CaseName
		}
	}
	return interfaceFormatExpr(expr)
}

func interfaceFormatExpr(expr frontend.Expr) string {
	var p sourcePrinter
	return p.formatExpr(expr)
}

func interfaceShortName(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 && idx+1 < len(name) {
		return name[idx+1:]
	}
	return name
}

func interfaceFunctionReturnParamPath(
	expr frontend.Expr,
	aliases map[string]string,
	functionParams, valueParams map[string]bool,
) (string, bool) {
	if id, ok := expr.(*frontend.IdentExpr); ok {
		if functionParams[id.Name] {
			return id.Name, true
		}
		if alias := aliases[id.Name]; alias != "" {
			return alias, true
		}
	}
	name := interfaceCallbackArgumentName(expr)
	if name == "" {
		return "", false
	}
	for paramName := range valueParams {
		if name == paramName || strings.HasPrefix(name, paramName+".") {
			return name, true
		}
	}
	return "", false
}

func interfaceCallbackArgumentName(expr frontend.Expr) string {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.FieldAccessExpr:
		base := interfaceCallbackArgumentName(e.Base)
		if base == "" {
			return ""
		}
		return base + "." + e.Field
	default:
		return ""
	}
}

func interfaceFunctionClosureLiteral(ref frontend.TypeRef, bodyIndent string) string {
	params := make([]string, 0, len(ref.Params))
	for i, param := range ref.Params {
		formatted := formatLSPTypeRef(param)
		if i < len(ref.ParamOwnership) && ref.ParamOwnership[i] != "" {
			formatted = ref.ParamOwnership[i] + " " + formatted
		}
		params = append(params, fmt.Sprintf("p%d: %s", i, formatted))
	}
	ret := "?"
	body := "0"
	if ref.Return != nil {
		ret = formatLSPTypeRef(*ref.Return)
		if ref.Return.Kind == frontend.TypeRefFunction {
			body = interfaceFunctionClosureLiteral(*ref.Return, bodyIndent+"    ")
		} else {
			body = interfaceReturnLiteral(*ref.Return)
		}
	}
	out := "fn(" + strings.Join(params, ", ") + ") -> " + ret
	if ref.Throws != nil {
		out += " throws " + formatLSPTypeRef(*ref.Throws)
	}
	if len(ref.Uses) > 0 {
		uses := append([]string(nil), ref.Uses...)
		sort.Strings(uses)
		out += " uses " + strings.Join(uses, ", ")
	}
	return out + ":\n" + bodyIndent + "return " + body
}

func interfaceInlineFunctionClosureLiteral(ref frontend.TypeRef) string {
	params := make([]string, 0, len(ref.Params))
	for i, param := range ref.Params {
		formatted := formatLSPTypeRef(param)
		if i < len(ref.ParamOwnership) && ref.ParamOwnership[i] != "" {
			formatted = ref.ParamOwnership[i] + " " + formatted
		}
		params = append(params, fmt.Sprintf("p%d: %s", i, formatted))
	}
	ret := "?"
	body := "0"
	if ref.Return != nil {
		ret = formatLSPTypeRef(*ref.Return)
		if ref.Return.Kind == frontend.TypeRefFunction {
			body = interfaceInlineFunctionClosureLiteral(*ref.Return)
		} else {
			body = interfaceReturnLiteral(*ref.Return)
		}
	}
	out := "fn(" + strings.Join(params, ", ") + ") -> " + ret
	if ref.Throws != nil {
		out += " throws " + formatLSPTypeRef(*ref.Throws)
	}
	if len(ref.Uses) > 0 {
		uses := append([]string(nil), ref.Uses...)
		sort.Strings(uses)
		out += " uses " + strings.Join(uses, ", ")
	}
	return out + " = " + body
}

// ---- lsp.go ----

type LSPSymbol struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Detail string `json:"detail,omitempty"`
}

type LSPHover struct {
	Name     string `json:"name"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Contents string `json:"contents"`
}

type LSPAnalysis struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Symbols     []LSPSymbol  `json:"symbols"`
	Hovers      []LSPHover   `json:"hovers"`
}

func AnalyzeLSPFile(path string) (LSPAnalysis, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return LSPAnalysis{}, err
	}
	out := AnalyzeLSPSource(raw, path)
	if len(out.Diagnostics) > 0 {
		return out, nil
	}
	file, err := frontend.ParseFile(raw, path)
	if err != nil {
		return out, nil
	}
	if len(file.Imports) == 0 {
		return out, nil
	}
	world, err := module.LoadWorld(path)
	if err != nil {
		out.Diagnostics = append(out.Diagnostics, DiagnosticFromError(err))
		return out, nil
	}
	if _, err := semantics.CheckWorldOpt(
		world,
		semantics.CheckOptions{RequireMain: false},
	); err != nil {
		out.Diagnostics = append(out.Diagnostics, DiagnosticFromError(err))
	}
	return out, nil
}

func AnalyzeLSPSource(src []byte, filename string) LSPAnalysis {
	out := LSPAnalysis{
		URI:         filename,
		Diagnostics: []Diagnostic{},
		Symbols:     []LSPSymbol{},
		Hovers:      []LSPHover{},
	}
	file, err := frontend.ParseFile(src, filename)
	if err != nil {
		out.Diagnostics = append(out.Diagnostics, DiagnosticFromError(err))
		return out
	}
	out.Symbols = collectLSPSymbols(file)
	out.Hovers = collectLSPHovers(file)
	if len(file.Imports) > 0 {
		return out
	}
	world := &module.World{
		EntryModule: "",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"": file},
	}
	if _, err := semantics.CheckWorldOpt(
		world,
		semantics.CheckOptions{RequireMain: false},
	); err != nil {
		out.Diagnostics = append(out.Diagnostics, DiagnosticFromError(err))
	}
	return out
}

func collectLSPSymbols(file *frontend.FileAST) []LSPSymbol {
	var symbols []LSPSymbol
	for _, st := range file.Structs {
		symbols = append(
			symbols,
			LSPSymbol{Name: st.Name, Kind: "struct", Line: st.At.Line, Column: st.At.Col},
		)
	}
	for _, st := range file.States {
		symbols = append(
			symbols,
			LSPSymbol{
				Name:   st.Name,
				Kind:   "state",
				Line:   st.At.Line,
				Column: st.At.Col,
				Detail: "state " + st.Name,
			},
		)
	}
	for _, view := range file.Views {
		symbols = append(
			symbols,
			LSPSymbol{
				Name:   view.Name,
				Kind:   "view",
				Line:   view.At.Line,
				Column: view.At.Col,
				Detail: "view " + view.Name,
			},
		)
	}
	for _, en := range file.Enums {
		symbols = append(
			symbols,
			LSPSymbol{Name: en.Name, Kind: "enum", Line: en.At.Line, Column: en.At.Col},
		)
	}
	for _, proto := range file.Protocols {
		symbols = append(
			symbols,
			LSPSymbol{
				Name:   proto.Name,
				Kind:   "protocol",
				Line:   proto.At.Line,
				Column: proto.At.Col,
				Detail: "protocol " + proto.Name,
			},
		)
	}
	for _, glob := range file.Globals {
		symbols = append(
			symbols,
			LSPSymbol{
				Name:   glob.Name,
				Kind:   globalSymbolKind(glob),
				Line:   glob.At.Line,
				Column: glob.At.Col,
				Detail: formatLSPGlobalDetail(glob),
			},
		)
	}
	for _, impl := range file.Impls {
		symbols = append(
			symbols,
			LSPSymbol{
				Name:   formatLSPImplName(impl),
				Kind:   "impl",
				Line:   impl.At.Line,
				Column: impl.At.Col,
				Detail: formatLSPImplDetail(impl),
			},
		)
	}
	for _, fn := range file.Funcs {
		if fn.ExtensionOf != "" || fn.Synthetic {
			continue
		}
		symbols = append(
			symbols,
			LSPSymbol{
				Name:   fn.Name,
				Kind:   "function",
				Line:   fn.Pos.Line,
				Column: fn.Pos.Col,
				Detail: formatLSPFuncDetail(fn),
			},
		)
	}
	for _, ext := range file.Extensions {
		for _, fn := range ext.Methods {
			symbols = append(
				symbols,
				LSPSymbol{
					Name:   fn.Name,
					Kind:   "extension-method",
					Line:   fn.Pos.Line,
					Column: fn.Pos.Col,
					Detail: formatLSPFuncDetail(fn),
				},
			)
		}
	}
	for _, test := range file.Tests {
		symbols = append(
			symbols,
			LSPSymbol{Name: test.Name, Kind: "test", Line: test.At.Line, Column: test.At.Col},
		)
	}
	return symbols
}

func collectLSPHovers(file *frontend.FileAST) []LSPHover {
	var hovers []LSPHover
	for _, fn := range file.Funcs {
		if fn.ExtensionOf != "" || fn.Synthetic {
			continue
		}
		hovers = append(hovers, LSPHover{
			Name:     fn.Name,
			Line:     fn.Pos.Line,
			Column:   fn.Pos.Col,
			Contents: formatLSPFuncDetail(fn),
		})
	}
	for _, ext := range file.Extensions {
		for _, fn := range ext.Methods {
			hovers = append(
				hovers,
				LSPHover{
					Name:     fn.Name,
					Line:     fn.Pos.Line,
					Column:   fn.Pos.Col,
					Contents: formatLSPFuncDetail(fn),
				},
			)
		}
	}
	for _, st := range file.Structs {
		hovers = append(
			hovers,
			LSPHover{
				Name:     st.Name,
				Line:     st.At.Line,
				Column:   st.At.Col,
				Contents: "struct " + st.Name,
			},
		)
	}
	for _, st := range file.States {
		hovers = append(
			hovers,
			LSPHover{
				Name:     st.Name,
				Line:     st.At.Line,
				Column:   st.At.Col,
				Contents: "state " + st.Name,
			},
		)
	}
	for _, view := range file.Views {
		hovers = append(
			hovers,
			LSPHover{
				Name:     view.Name,
				Line:     view.At.Line,
				Column:   view.At.Col,
				Contents: "view " + view.Name,
			},
		)
	}
	for _, en := range file.Enums {
		hovers = append(
			hovers,
			LSPHover{
				Name:     en.Name,
				Line:     en.At.Line,
				Column:   en.At.Col,
				Contents: "enum " + en.Name,
			},
		)
	}
	for _, proto := range file.Protocols {
		hovers = append(
			hovers,
			LSPHover{
				Name:     proto.Name,
				Line:     proto.At.Line,
				Column:   proto.At.Col,
				Contents: "protocol " + proto.Name,
			},
		)
	}
	for _, glob := range file.Globals {
		hovers = append(
			hovers,
			LSPHover{
				Name:     glob.Name,
				Line:     glob.At.Line,
				Column:   glob.At.Col,
				Contents: formatLSPGlobalDetail(glob),
			},
		)
	}
	for _, impl := range file.Impls {
		hovers = append(
			hovers,
			LSPHover{
				Name:     formatLSPImplName(impl),
				Line:     impl.At.Line,
				Column:   impl.At.Col,
				Contents: formatLSPImplDetail(impl),
			},
		)
	}
	return hovers
}

func globalSymbolKind(glob *frontend.GlobalDecl) string {
	if glob.Mutable {
		return "var"
	}
	if glob.Const {
		return "const"
	}
	return "val"
}

func formatLSPGlobalDetail(glob *frontend.GlobalDecl) string {
	out := globalSymbolKind(glob) + " " + glob.Name
	if glob.Type.Name != "" || glob.Type.Elem != nil {
		out += ": " + formatLSPTypeRef(glob.Type)
	}
	return out
}

func formatLSPImplName(impl *frontend.ImplDecl) string {
	return formatLSPTypeRef(impl.Type) + ": " + formatLSPTypeRef(impl.Protocol)
}

func formatLSPImplDetail(impl *frontend.ImplDecl) string {
	return "impl " + formatLSPImplName(impl)
}

func formatLSPFuncDetail(fn *frontend.FuncDecl) string {
	params := make([]string, 0, len(fn.Params))
	for _, param := range fn.Params {
		typ := formatLSPTypeRef(param.Type)
		if param.Ownership != "" {
			typ = param.Ownership + " " + typ
		}
		params = append(params, param.Name+": "+typ)
	}
	prefix := "func"
	if fn.Async {
		prefix = "async func"
	}
	typeParams := formatLSPTypeParams(fn.TypeParams, fn.TypeParamBounds)
	returnType := formatLSPTypeRef(fn.ReturnType)
	if fn.ReturnOwnership != "" {
		returnType = fn.ReturnOwnership + " " + returnType
	}
	detail := fmt.Sprintf(
		"%s %s%s(%s) -> %s",
		prefix,
		fn.Name,
		typeParams,
		strings.Join(params, ", "),
		returnType,
	)
	if fn.HasThrows {
		detail += " throws " + formatLSPTypeRef(fn.Throws)
	}
	if len(fn.Uses) > 0 {
		uses := append([]string(nil), fn.Uses...)
		sort.Strings(uses)
		detail += " uses " + strings.Join(uses, ", ")
	}
	return detail
}

func formatLSPTypeParams(names []string, bounds []frontend.TypeParamBound) string {
	if len(names) == 0 {
		return ""
	}
	byName := make(map[string]frontend.TypeRef, len(bounds))
	for _, bound := range bounds {
		byName[bound.Name] = bound.Bound
	}
	formatted := make([]string, 0, len(names))
	for _, name := range names {
		if bound, ok := byName[name]; ok {
			formatted = append(formatted, name+": "+formatLSPTypeRef(bound))
			continue
		}
		formatted = append(formatted, name)
	}
	return "<" + strings.Join(formatted, ", ") + ">"
}

func formatLSPFuncSigDecl(sig frontend.FuncSigDecl) string {
	params := make([]string, 0, len(sig.Params))
	for _, param := range sig.Params {
		typ := formatLSPTypeRef(param.Type)
		if param.Ownership != "" {
			typ = param.Ownership + " " + typ
		}
		params = append(params, param.Name+": "+typ)
	}
	prefix := "func"
	if sig.Async {
		prefix = "async func"
	}
	typeParams := formatLSPTypeParams(sig.TypeParams, nil)
	returnType := formatLSPTypeRef(sig.ReturnType)
	if sig.ReturnOwnership != "" {
		returnType = sig.ReturnOwnership + " " + returnType
	}
	detail := fmt.Sprintf(
		"%s %s%s(%s) -> %s",
		prefix,
		sig.Name,
		typeParams,
		strings.Join(params, ", "),
		returnType,
	)
	if sig.HasThrows {
		detail += " throws " + formatLSPTypeRef(sig.Throws)
	}
	if len(sig.Uses) > 0 {
		uses := append([]string(nil), sig.Uses...)
		sort.Strings(uses)
		detail += " uses " + strings.Join(uses, ", ")
	}
	return detail
}

func formatLSPTypeRef(ref frontend.TypeRef) string {
	switch ref.Kind {
	case frontend.TypeRefSlice:
		return "[]" + formatLSPTypeRef(*ref.Elem)
	case frontend.TypeRefArray:
		return fmt.Sprintf("[%d]%s", ref.Len, formatLSPTypeRef(*ref.Elem))
	case frontend.TypeRefOptional:
		return formatLSPTypeRef(*ref.Elem) + "?"
	case frontend.TypeRefFunction:
		params := make([]string, 0, len(ref.Params))
		for i, param := range ref.Params {
			formatted := formatLSPTypeRef(param)
			if i < len(ref.ParamOwnership) && ref.ParamOwnership[i] != "" {
				formatted = ref.ParamOwnership[i] + " " + formatted
			}
			params = append(params, formatted)
		}
		ret := "?"
		if ref.Return != nil {
			ret = formatLSPTypeRef(*ref.Return)
			if ref.ReturnOwnership != "" {
				ret = ref.ReturnOwnership + " " + ret
			}
		}
		out := "fn(" + strings.Join(params, ", ") + ") -> " + ret
		if ref.Throws != nil {
			out += " throws " + formatLSPTypeRef(*ref.Throws)
		}
		if len(ref.Uses) > 0 {
			uses := append([]string(nil), ref.Uses...)
			sort.Strings(uses)
			out += " uses " + strings.Join(uses, ", ")
		}
		return out
	default:
		name := canonicalLSPTypeName(ref.Name)
		if len(ref.TypeArgs) == 0 {
			return name
		}
		args := make([]string, 0, len(ref.TypeArgs))
		for _, arg := range ref.TypeArgs {
			args = append(args, formatLSPTypeRef(arg))
		}
		return name + "<" + strings.Join(args, ", ") + ">"
	}
}

func canonicalLSPTypeName(name string) string {
	switch name {
	case "Int", "i32":
		return "i32"
	case "UInt8", "Byte", "u8":
		return "u8"
	case "Bool", "bool":
		return "bool"
	case "String", "str":
		return "str"
	case "ConsentToken", "consent.token":
		return "consent.token"
	case "SecretInt", "secret.i32":
		return "secret.i32"
	default:
		return name
	}
}

// ---- manifest.go ----

type Manifest struct {
	CompilerVersion string            `json:"compiler_version"`
	Formats         []FormatManifest  `json:"formats"`
	Targets         []TargetManifest  `json:"targets"`
	Builtins        []BuiltinManifest `json:"builtins"`
	RuntimeABI      RuntimeManifest   `json:"runtime_abi"`
	Features        []FeatureInfo     `json:"features"`
}

type FormatManifest = formats.Info

type TargetManifest struct {
	Triple                   string   `json:"triple"`
	Status                   string   `json:"status"`
	OS                       string   `json:"os"`
	Arch                     string   `json:"arch"`
	ABI                      string   `json:"abi"`
	DataModel                string   `json:"data_model"`
	Format                   string   `json:"format"`
	ExeExt                   string   `json:"exe_ext"`
	CollectImports           bool     `json:"collect_imports"`
	RunMode                  string   `json:"run_mode"`
	UIRuntimeContract        string   `json:"ui_runtime_contract,omitempty"`
	UIRuntimeStatus          string   `json:"ui_runtime_status"`
	UIRuntimeEvidence        string   `json:"ui_runtime_evidence,omitempty"`
	PointerWidthBits         int      `json:"pointer_width_bits"`
	RegisterWidthBits        int      `json:"register_width_bits"`
	NativeIntWidthBits       int      `json:"native_int_width_bits"`
	Endian                   string   `json:"endian"`
	StackAlignmentBytes      int      `json:"stack_alignment_bytes"`
	MaxAtomicWidthBits       int      `json:"max_atomic_width_bits"`
	AtomicWidthBits          []int    `json:"atomic_width_bits"`
	AtomicPointerWidthBits   int      `json:"atomic_pointer_width_bits"`
	UnsupportedReason        string   `json:"unsupported_reason,omitempty"`
	RuntimeStatus            string   `json:"runtime_status,omitempty"`
	StdlibStatus             string   `json:"stdlib_status,omitempty"`
	FFIStatus                string   `json:"ffi_status,omitempty"`
	MemoryBuild              string   `json:"memory_build"`
	MemoryLower              string   `json:"memory_lower"`
	MemoryRun                string   `json:"memory_run"`
	MemoryRawDiagnostics     string   `json:"memory_raw_diagnostics"`
	MemoryRegionLowering     string   `json:"memory_region_lowering"`
	MemoryAlignmentSemantics string   `json:"memory_alignment_semantics"`
	MemoryClaimLevel         string   `json:"memory_claim_level"`
	RunnerProbeCommand       string   `json:"runner_probe_command,omitempty"`
	ReleaseGate              string   `json:"release_gate,omitempty"`
	EvidenceArtifacts        []string `json:"evidence_artifacts,omitempty"`
	SyscallInstruction       string   `json:"syscall_instruction,omitempty"`
	SyscallNumbering         string   `json:"syscall_numbering,omitempty"`
	SyscallArgRegisters      []string `json:"syscall_arg_registers,omitempty"`
	SyscallErrorRange        string   `json:"syscall_error_range,omitempty"`
	SupportsDebugInfo        bool     `json:"supports_debug_info"`
	SupportsReleaseOptimize  bool     `json:"supports_release_optimize"`
}

type BuiltinManifest struct {
	Name          string   `json:"name"`
	Aliases       []string `json:"aliases,omitempty"`
	ParamTypes    []string `json:"param_types,omitempty"`
	ReturnType    string   `json:"return_type"`
	Effects       []string `json:"effects,omitempty"`
	UnsafePolicy  string   `json:"unsafe_policy"`
	UnsafeDetails string   `json:"unsafe_details,omitempty"`
}

type RuntimeManifest struct {
	ReservedPrefix            string   `json:"reserved_prefix"`
	ActorsSupportedTargets    []string `json:"actors_supported_targets"`
	ActorsRequiredSymbols     []string `json:"actors_required_symbols"`
	ActorStateRequiredSymbols []string `json:"actor_state_required_symbols"`
	TaskRequiredSymbols       []string `json:"task_required_symbols"`
	TaskGroupRequiredSymbols  []string `json:"task_group_required_symbols"`
	TypedTaskRequiredSymbols  []string `json:"typed_task_required_symbols"`
	TimeRequiredSymbols       []string `json:"time_required_symbols,omitempty"`
	FilesystemRequiredSymbols []string `json:"filesystem_required_symbols,omitempty"`
	NetRequiredSymbols        []string `json:"net_required_symbols,omitempty"`
	SurfaceRequiredSymbols    []string `json:"surface_required_symbols,omitempty"`
	ActorsProgramGlueSymbols  []string `json:"actors_program_glue_symbols"`
}

func GetManifest() (Manifest, error) {
	builtins, err := semantics.DescribeBuiltins()
	if err != nil {
		return Manifest{}, err
	}
	builtinOut := make([]BuiltinManifest, 0, len(builtins))
	for _, b := range builtins {
		builtinOut = append(builtinOut, BuiltinManifest{
			Name:          b.Name,
			Aliases:       append([]string(nil), b.Aliases...),
			ParamTypes:    append([]string(nil), b.ParamTypes...),
			ReturnType:    b.ReturnType,
			Effects:       append([]string(nil), b.Effects...),
			UnsafePolicy:  b.UnsafePolicy,
			UnsafeDetails: b.UnsafeDetails,
		})
	}

	targets := ctarget.AllBuildable()
	targetOut := make([]TargetManifest, 0, len(targets))
	for _, t := range targets {
		targetOut = append(targetOut, TargetManifest{
			Triple:                   t.Triple,
			Status:                   fmt.Sprint(t.Status),
			OS:                       fmt.Sprint(t.OS),
			Arch:                     fmt.Sprint(t.Arch),
			ABI:                      fmt.Sprint(t.ABI),
			DataModel:                fmt.Sprint(t.DataModel),
			Format:                   fmt.Sprint(t.Format),
			ExeExt:                   t.ExeExt,
			CollectImports:           t.CollectImports,
			RunMode:                  fmt.Sprint(t.RunMode),
			UIRuntimeContract:        ctarget.UIRuntimeContract(t.Triple),
			UIRuntimeStatus:          ctarget.UIRuntimeStatus(t.Triple),
			UIRuntimeEvidence:        ctarget.UIRuntimeEvidence(t.Triple),
			PointerWidthBits:         t.PointerWidthBits,
			RegisterWidthBits:        t.RegisterWidthBits,
			NativeIntWidthBits:       t.NativeIntWidthBits,
			Endian:                   fmt.Sprint(t.Endian),
			StackAlignmentBytes:      t.StackAlignmentBytes,
			MaxAtomicWidthBits:       t.MaxAtomicWidthBits,
			AtomicWidthBits:          t.AtomicWidthBits(),
			AtomicPointerWidthBits:   manifestAtomicPointerWidthBits(t),
			UnsupportedReason:        t.UnsupportedReason,
			RuntimeStatus:            t.RuntimeStatus,
			StdlibStatus:             t.StdlibStatus,
			FFIStatus:                t.FFIStatus,
			MemoryBuild:              t.MemoryBuild,
			MemoryLower:              t.MemoryLower,
			MemoryRun:                t.MemoryRun,
			MemoryRawDiagnostics:     t.MemoryRawDiagnostics,
			MemoryRegionLowering:     t.MemoryRegionLowering,
			MemoryAlignmentSemantics: t.MemoryAlignmentSemantics,
			MemoryClaimLevel:         t.MemoryClaimLevel,
			RunnerProbeCommand:       t.RunnerProbeCommand,
			ReleaseGate:              t.ReleaseGate,
			EvidenceArtifacts:        append([]string(nil), t.EvidenceArtifacts...),
			SyscallInstruction:       t.SyscallInstruction,
			SyscallNumbering:         t.SyscallNumbering,
			SyscallArgRegisters:      append([]string(nil), t.SyscallArgRegisters...),
			SyscallErrorRange:        t.SyscallErrorRange,
			SupportsDebugInfo:        t.SupportsDebugInfo,
			SupportsReleaseOptimize:  t.SupportsReleaseOptimize,
		})
	}

	return Manifest{
		CompilerVersion: Version(),
		Formats:         formats.All(),
		Targets:         targetOut,
		Builtins:        builtinOut,
		RuntimeABI: RuntimeManifest{
			ReservedPrefix:            "__tetra_",
			ActorsSupportedTargets:    []string{"linux-x64", "macos-x64", "windows-x64"},
			ActorsRequiredSymbols:     requiredActorRuntimeSymbols(),
			ActorStateRequiredSymbols: requiredActorStateRuntimeSymbols(),
			TaskRequiredSymbols:       requiredTaskRuntimeSymbols(),
			TaskGroupRequiredSymbols:  requiredTaskGroupRuntimeSymbols(),
			TypedTaskRequiredSymbols:  requiredTypedTaskRuntimeSymbols(8),
			TimeRequiredSymbols:       requiredTimeRuntimeSymbols(),
			FilesystemRequiredSymbols: requiredFilesystemRuntimeSymbols(),
			NetRequiredSymbols:        requiredNetRuntimeSymbols(),
			SurfaceRequiredSymbols:    requiredSurfaceRuntimeSymbols(),
			ActorsProgramGlueSymbols: []string{
				"__tetra_actor_dispatch",
				"__tetra_actor_main_entry_id",
			},
		},
		Features: FeatureRegistry(),
	}, nil
}

func manifestAtomicPointerWidthBits(t ctarget.Target) int {
	layout, err := t.AtomicPointerLayout()
	if err != nil {
		return 0
	}
	return layout.WidthBits
}

// ---- test_runner.go ----

type TestRunnerSource struct {
	Name         string
	Filename     string
	Index        int
	FunctionName string
	Source       []byte
}

type TestRunnerResult struct {
	Name         string `json:"name"`
	Filename     string `json:"filename"`
	Index        int    `json:"index"`
	FunctionName string `json:"function_name"`
	ExitCode     int    `json:"exit_code"`
	Passed       bool   `json:"passed"`
	DurationMS   int64  `json:"duration_ms"`
	Error        string `json:"error,omitempty"`
}

type TestRunnerFileReport struct {
	Filename   string `json:"filename"`
	Total      int    `json:"total"`
	Passed     int    `json:"passed"`
	Failed     int    `json:"failed"`
	DurationMS int64  `json:"duration_ms"`
}

type TestRunnerReport struct {
	Total      int                    `json:"total"`
	Passed     int                    `json:"passed"`
	Failed     int                    `json:"failed"`
	Target     string                 `json:"target,omitempty"`
	DurationMS int64                  `json:"duration_ms"`
	Files      []TestRunnerFileReport `json:"files"`
	Results    []TestRunnerResult     `json:"results"`
}

func TestRunnerSources(src []byte, filename string) ([]TestRunnerSource, error) {
	file, err := frontend.ParseFile(src, filename)
	if err != nil {
		return nil, err
	}
	baseSrc := testRunnerBaseSource(file)
	out := make([]TestRunnerSource, 0, len(file.Tests))
	for i, test := range file.Tests {
		var b strings.Builder
		if baseSrc != "" {
			b.WriteString(baseSrc)
			b.WriteString("\n\n")
		}
		fnName := fmt.Sprintf("__tetra_test_%d_%s", i, sanitizeTestName(test.Name))
		b.WriteString("\nfunc ")
		b.WriteString(fnName)
		b.WriteString("() -> Int\n")
		b.WriteString(testRunnerUsesClause())
		b.WriteString(":\n")
		for _, stmt := range test.Body {
			writeTestStmt(&b, stmt, 1)
		}
		b.WriteString("    return 0\n\n")
		b.WriteString("func main() -> Int\n")
		b.WriteString(testRunnerUsesClause())
		b.WriteString(":\n")
		b.WriteString("    return ")
		b.WriteString(fnName)
		b.WriteString("()\n")
		out = append(out, TestRunnerSource{
			Name:         test.Name,
			Filename:     filename,
			Index:        i,
			FunctionName: fnName,
			Source:       []byte(b.String()),
		})
	}
	return out, nil
}

func testRunnerBaseSource(file *frontend.FileAST) string {
	base := *file
	base.Tests = nil
	if len(base.Funcs) > 0 {
		filtered := make([]*frontend.FuncDecl, 0, len(base.Funcs))
		for _, fn := range base.Funcs {
			if fn != nil && fn.Name == "main" {
				continue
			}
			filtered = append(filtered, fn)
		}
		base.Funcs = filtered
	}
	var p sourcePrinter
	p.file(&base)
	return strings.TrimSpace(p.b.String())
}

func testRunnerUsesClause() string {
	return "uses actors, alloc, capability, control, islands, io, link, mem, mmio, runtime"
}

func (s TestRunnerSource) Result(exitCode int, runErr error) TestRunnerResult {
	return s.ResultWithDuration(exitCode, runErr, 0)
}

func (s TestRunnerSource) ResultWithDuration(
	exitCode int,
	runErr error,
	durationMS int64,
) TestRunnerResult {
	result := TestRunnerResult{
		Name:         s.Name,
		Filename:     s.Filename,
		Index:        s.Index,
		FunctionName: s.FunctionName,
		ExitCode:     exitCode,
		Passed:       exitCode == 0 && runErr == nil,
		DurationMS:   durationMS,
	}
	if runErr != nil {
		result.Error = runErr.Error()
	} else if exitCode != 0 {
		result.Error = fmt.Sprintf("exit code %d", exitCode)
	}
	return result
}

func NewTestRunnerReport(results []TestRunnerResult) TestRunnerReport {
	report := TestRunnerReport{
		Total:   len(results),
		Files:   []TestRunnerFileReport{},
		Results: append([]TestRunnerResult{}, results...),
	}
	sort.SliceStable(report.Results, func(i, j int) bool {
		if report.Results[i].Filename != report.Results[j].Filename {
			return report.Results[i].Filename < report.Results[j].Filename
		}
		return report.Results[i].Index < report.Results[j].Index
	})
	byFile := map[string]*TestRunnerFileReport{}
	for _, result := range results {
		if result.Passed {
			report.Passed++
		} else {
			report.Failed++
		}
		report.DurationMS += result.DurationMS
		file := byFile[result.Filename]
		if file == nil {
			file = &TestRunnerFileReport{Filename: result.Filename}
			byFile[result.Filename] = file
		}
		file.Total++
		file.DurationMS += result.DurationMS
		if result.Passed {
			file.Passed++
		} else {
			file.Failed++
		}
	}
	filenames := make([]string, 0, len(byFile))
	for filename := range byFile {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	for _, filename := range filenames {
		report.Files = append(report.Files, *byFile[filename])
	}
	return report
}

func NewTestRunnerReportForTarget(results []TestRunnerResult, target string) TestRunnerReport {
	report := NewTestRunnerReport(results)
	report.Target = strings.TrimSpace(target)
	return report
}

func writeTestStmt(b *strings.Builder, stmt frontend.Stmt, indent int) {
	prefix := strings.Repeat(" ", indent*4)
	switch s := stmt.(type) {
	case *frontend.ExpectStmt:
		b.WriteString(prefix)
		b.WriteString("if ")
		b.WriteString(formatTestExpr(s.Cond))
		b.WriteString(":\n")
		b.WriteString(prefix)
		b.WriteString("    let __ok: Int = 0\n")
		b.WriteString(prefix)
		b.WriteString("else:\n")
		b.WriteString(prefix)
		b.WriteString("    return 1\n")
	default:
		var p sourcePrinter
		p.stmt(stmt, indent)
		b.WriteString(p.b.String())
	}
}

func formatTestExpr(expr frontend.Expr) string {
	var p sourcePrinter
	return p.formatExpr(expr)
}

var nonTestNameChar = regexp.MustCompile(`[^A-Za-z0-9_]+`)

func sanitizeTestName(name string) string {
	clean := nonTestNameChar.ReplaceAllString(name, "_")
	clean = strings.Trim(clean, "_")
	if clean == "" {
		return "case"
	}
	if clean[0] >= '0' && clean[0] <= '9' {
		return "case_" + clean
	}
	return clean
}

// ---- version.go ----

func Version() string {
	return version.CompilerVersion
}
