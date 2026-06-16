package compiler

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/buildnative"
	"tetra_language/compiler/internal/buildplan"
	"tetra_language/compiler/internal/cache"
	"tetra_language/compiler/internal/deps"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/internal/validation"
	"tetra_language/compiler/internal/version"
	ctarget "tetra_language/compiler/target"
)

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
func BuildFileWithStatsOpt(inputPath, outputPath, target string, opt BuildOptions) (*BuildStats, error) {
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
	}
	linkedObjects, err := prepareLinkedObjects(build.world, build.checked, opt.LinkObjectPaths, native.triple)
	if err != nil {
		return nil, err
	}

	plan, stats, err := planNativeModuleBuild(build.world, build.checked, native.triple, opt, linkedObjects)
	if err != nil {
		return nil, err
	}
	if err := compileNativeModulePlan(build.world, build.checked, native, opt, plan, stats); err != nil {
		return nil, err
	}

	objects, err := objectsFromModulePlan(plan)
	if err != nil {
		return nil, err
	}
	if err := linkNativeExecutable(outputPath, native, opt, build.checked, objects, linkedObjects); err != nil {
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

func resolveExecutableBuildTarget(inputPath, outputPath, target string, opt BuildOptions) (nativeBuildTarget, bool, *BuildStats, error) {
	tgt, err := ctarget.Parse(target)
	if err != nil {
		return nativeBuildTarget{}, false, nil, err
	}
	if err := validateRuntimeHeapTelemetryBuildOptions(tgt, opt); err != nil {
		return nativeBuildTarget{}, false, nil, err
	}
	if opt.DebugInfo && !tgt.SupportsDebugInfo {
		return nativeBuildTarget{}, false, nil, fmt.Errorf("target does not support debug info: %s", tgt.Triple)
	}
	if opt.ReleaseOptimize && !tgt.SupportsReleaseOptimize {
		return nativeBuildTarget{}, false, nil, fmt.Errorf("target does not support release optimization: %s", tgt.Triple)
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
		if ctarget.IsBuildOnlyTarget(tgt.Triple) && tgt.Triple != "linux-x32" && tgt.Triple != "linux-x86" {
			reason := tgt.UnsupportedReason
			if reason == "" {
				reason = "executable support is not implemented yet"
			}
			return nativeBuildTarget{}, false, nil, fmt.Errorf("target backend not implemented: %s (%s)", tgt.Triple, reason)
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
	return nativeBuildTarget{target: tgt, triple: tgt.Triple, backend: backend, codegen: codegen}, false, nil, nil
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

func loadCheckedBuildWorld(inputPath string, opt BuildOptions, requireMain bool, target string) (checkedBuildWorld, error) {
	world, err := loadWorldForBuild(inputPath, opt)
	if err != nil {
		return checkedBuildWorld{}, err
	}
	if err := validateTargetExportedFFIAST(world, target); err != nil {
		return checkedBuildWorld{}, err
	}
	checked, err := semantics.CheckWorldOpt(world, semanticsCheckOptionsForTarget(requireMain, target))
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

func prepareLinkedObjects(world *World, checked *semantics.CheckedProgram, paths []string, target string) ([]linkedObject, error) {
	linkedObjects, err := readLinkObjects(paths, target)
	if err != nil {
		return nil, err
	}
	if err := validateInterfaceImplementationProviders(world, checked, linkedObjects); err != nil {
		return nil, err
	}
	return linkedObjects, nil
}

func planNativeModuleBuild(world *World, checked *semantics.CheckedProgram, target string, opt BuildOptions, linkedObjects []linkedObject) (moduleBuildPlan, *BuildStats, error) {
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
		depHash, err := cache.DepSigHashFromDepsWithInterfaceHashes(callees, typeDeps, sigMap, typeSigMap, world.InterfaceHashes)
		if err != nil {
			return moduleBuildPlan{}, nil, err
		}
		if !opt.EmitRuntimeHeapTelemetry {
			obj, hit, err := cache.LoadCachedObject(world.Root, target, buildTag, module, srcHash, depHash)
			if err != nil {
				return moduleBuildPlan{}, nil, err
			}
			if hit {
				stats.CacheHits = append(stats.CacheHits, module)
				objectsByModule[module] = obj
				continue
			}
		}
		toCompile = append(toCompile, moduleBuildJob{Module: module, SrcHash: srcHash, DepHash: depHash})
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

func compileNativeModulePlan(world *World, checked *semantics.CheckedProgram, native nativeBuildTarget, opt BuildOptions, plan moduleBuildPlan, stats *BuildStats) error {
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
		allocationPlan, err = allocplan.FromPLIRWithOptions(plirProg, allocationPlanOptionsForTarget(native.triple))
		if err != nil {
			return err
		}
		allocationSummaryProgram, err = lower.LowerWithOptions(checked, lowerOptionsForTarget(native.triple))
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
			funcs, err := lower.LowerModuleWithOptions(checked, job.Module, lowerOptionsForTarget(native.triple))
			if err != nil {
				setErr(err)
				continue
			}
			if err := validateTargetAtomicIR(funcs, native.target); err != nil {
				setErr(err)
				continue
			}
			if allocationPlan != nil {
				if err := validation.ValidateAllocationLoweringWithSummaryProgram(allocationPlanForIRFuncs(allocationPlan, funcs), &ir.IRProgram{Funcs: funcs}, allocationSummaryProgram); err != nil {
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
