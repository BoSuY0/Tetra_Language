package compiler

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"tetra_language/compiler/internal/actorsrt"
	"tetra_language/compiler/internal/backend/linux_x32"
	"tetra_language/compiler/internal/backend/linux_x64"
	"tetra_language/compiler/internal/backend/linux_x86"
	"tetra_language/compiler/internal/backend/macos_x64"
	"tetra_language/compiler/internal/backend/native_shell"
	"tetra_language/compiler/internal/backend/wasm32_wasi"
	"tetra_language/compiler/internal/backend/wasm32_web"
	"tetra_language/compiler/internal/backend/windows_x64"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/cache"
	"tetra_language/compiler/internal/deps"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/runtimeabi"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/internal/version"
	ctarget "tetra_language/compiler/target"
)

func BuildFile(inputPath, outputPath, target string) error {
	_, err := BuildFileWithStats(inputPath, outputPath, target)
	return err
}

type EmitMode int

const (
	EmitExe EmitMode = iota
	EmitObject
	EmitLibrary
)

type RuntimeMode int

const (
	RuntimeAuto RuntimeMode = iota
	RuntimeSelfHost
	RuntimeBuiltin
)

type BuildOptions struct {
	Jobs              int
	IslandsDebug      bool
	DebugInfo         bool
	ReleaseOptimize   bool
	Emit              EmitMode
	Runtime           RuntimeMode
	RuntimeObjectPath string
	LinkObjectPaths   []string
	ProjectRoot       string
	SourceRoots       []string
	DependencyRoots   []ModuleRoot
	InterfaceOnly     bool
}

type BuildStats struct {
	CompiledModules  []string
	CacheHits        []string
	LoweredModules   []string
	InterfaceModules []string
}

type linkedObject struct {
	path        string
	obj         *Object
	contentHash [32]byte
}

type nativeCodegenFunc func([]IRFunc, [][]byte) (*Object, error)

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

type moduleBuildJob struct {
	module  string
	srcHash [32]byte
	depHash [32]byte
}

type moduleBuildPlan struct {
	modules           []string
	publicAPIHashes   map[string]string
	buildTag          string
	objectsByModule   map[string]*Object
	objectlessModules map[string]bool
	toCompile         []moduleBuildJob
}

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

	return stats, nil
}

func resolveExecutableBuildTarget(inputPath, outputPath, target string, opt BuildOptions) (nativeBuildTarget, bool, *BuildStats, error) {
	tgt, err := ctarget.Parse(target)
	if err != nil {
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
	backend, ok := nativeExecutableBackendForTarget(tgt)
	if !ok {
		return nil, fmt.Errorf("unsupported target: %s", tgt.Triple)
	}
	return backend.codegen(nativeCodegenOptionsForTarget(tgt, opt)), nil
}

func nativeExecutableBackendForTarget(tgt ctarget.Target) (nativeExecutableBackend, bool) {
	if tgt.Triple == "linux-x86" {
		backend := nativeLinuxX86ExecutableBackend()
		return backend, true
	}
	if tgt.Arch != ctarget.ArchX64 {
		return nativeExecutableBackend{}, false
	}
	if tgt.Triple == "linux-x32" {
		backend := nativeLinuxX32ExecutableBackend()
		return backend, true
	}
	backend, ok := nativeExecutableBackends()[tgt.OS]
	if !ok || backend.format != tgt.Format {
		return nativeExecutableBackend{}, false
	}
	return backend, true
}

func nativeCodegenOptions(opt BuildOptions) x64.CodegenOptions {
	return nativeCodegenOptionsForTarget(ctarget.Target{}, opt)
}

func nativeCodegenOptionsForTarget(tgt ctarget.Target, opt BuildOptions) x64.CodegenOptions {
	return x64.CodegenOptions{
		IslandsDebug:       opt.IslandsDebug,
		DebugInfo:          opt.DebugInfo,
		ReleaseOptimize:    opt.ReleaseOptimize,
		PointerWidthBits:   tgt.PointerWidthBits,
		NativeIntWidthBits: tgt.NativeIntWidthBits,
		RegisterWidthBits:  tgt.RegisterWidthBits,
	}
}

func nativeExecutableBackends() map[ctarget.OS]nativeExecutableBackend {
	return map[ctarget.OS]nativeExecutableBackend{
		ctarget.OSLinux: {
			name:   "linux-x64",
			os:     ctarget.OSLinux,
			format: ctarget.FormatELF,
			codegen: func(opt x64.CodegenOptions) nativeCodegenFunc {
				return func(funcs []IRFunc, dataPrefix [][]byte) (*Object, error) {
					return linux_x64.CodegenObjectLinuxX64WithOptionsAndDataPrefix(funcs, dataPrefix, opt)
				}
			},
			link: func(outputPath string, objects []*Object, mainName string) error {
				img, err := LinkLinuxX64(objects, mainName)
				if err != nil {
					return err
				}
				return WriteELF64LinuxX64(outputPath, img)
			},
			actorRuntime: actorsrt.BuildLinuxX64,
		},
		ctarget.OSWindows: {
			name:   "windows-x64",
			os:     ctarget.OSWindows,
			format: ctarget.FormatPE,
			codegen: func(opt x64.CodegenOptions) nativeCodegenFunc {
				return func(funcs []IRFunc, dataPrefix [][]byte) (*Object, error) {
					return windows_x64.CodegenObjectWindowsX64WithOptionsAndDataPrefix(funcs, dataPrefix, opt)
				}
			},
			link: func(outputPath string, objects []*Object, mainName string) error {
				img, err := LinkWindowsX64(objects, mainName)
				if err != nil {
					return err
				}
				return WritePE64WindowsX64(outputPath, img)
			},
			actorRuntime: actorsrt.BuildWindowsX64,
		},
		ctarget.OSMacOS: {
			name:   "macos-x64",
			os:     ctarget.OSMacOS,
			format: ctarget.FormatMachO,
			codegen: func(opt x64.CodegenOptions) nativeCodegenFunc {
				return func(funcs []IRFunc, dataPrefix [][]byte) (*Object, error) {
					return macos_x64.CodegenObjectMacOSX64WithOptionsAndDataPrefix(funcs, dataPrefix, opt)
				}
			},
			link: func(outputPath string, objects []*Object, mainName string) error {
				img, err := LinkMacOSX64(objects, mainName)
				if err != nil {
					return err
				}
				return WriteMachO64MacOSX64(outputPath, img)
			},
			actorRuntime: actorsrt.BuildMacOSX64,
		},
	}
}

func nativeLinuxX32ExecutableBackend() nativeExecutableBackend {
	return nativeExecutableBackend{
		name:   "linux-x32",
		os:     ctarget.OSLinux,
		format: ctarget.FormatELF,
		codegen: func(opt x64.CodegenOptions) nativeCodegenFunc {
			return func(funcs []IRFunc, dataPrefix [][]byte) (*Object, error) {
				return linux_x32.CodegenObjectLinuxX32WithOptionsAndDataPrefix(funcs, dataPrefix, opt)
			}
		},
		link: func(outputPath string, objects []*Object, mainName string) error {
			img, err := LinkLinuxX32(objects, mainName)
			if err != nil {
				return err
			}
			return WriteELF32LinuxX32(outputPath, img)
		},
	}
}

func nativeLinuxX86ExecutableBackend() nativeExecutableBackend {
	return nativeExecutableBackend{
		name:   "linux-x86",
		os:     ctarget.OSLinux,
		format: ctarget.FormatELF,
		codegen: func(opt x64.CodegenOptions) nativeCodegenFunc {
			return func(funcs []IRFunc, dataPrefix [][]byte) (*Object, error) {
				return linux_x86.CodegenObjectLinuxX86WithOptionsAndDataPrefix(funcs, dataPrefix, opt)
			}
		},
		link: func(outputPath string, objects []*Object, mainName string) error {
			img, err := LinkLinuxX86(objects, mainName)
			if err != nil {
				return err
			}
			return WriteELF32LinuxX86(outputPath, img)
		},
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
	checked, err := semantics.CheckWorldOpt(world, semantics.CheckOptions{RequireMain: requireMain})
	if err != nil {
		return checkedBuildWorld{}, err
	}
	return checkedBuildWorld{world: world, checked: checked}, nil
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

	modules := make([]string, 0, len(world.ByModule))
	for module := range world.ByModule {
		if world.InterfaceModules[module] {
			continue
		}
		modules = append(modules, module)
	}
	sort.Strings(modules)
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
		depHash, err := cache.DepSigHashFromDepsWithInterfaceHashes(callees, typeDeps, sigMap, typeSigMap, world.InterfaceHashes)
		if err != nil {
			return moduleBuildPlan{}, nil, err
		}
		obj, hit, err := cache.LoadCachedObject(world.Root, target, buildTag, module, srcHash, depHash)
		if err != nil {
			return moduleBuildPlan{}, nil, err
		}
		if hit {
			stats.CacheHits = append(stats.CacheHits, module)
			objectsByModule[module] = obj
			continue
		}
		toCompile = append(toCompile, moduleBuildJob{module: module, srcHash: srcHash, depHash: depHash})
	}

	return moduleBuildPlan{
		modules:           modules,
		publicAPIHashes:   publicAPIHashes,
		buildTag:          buildTag,
		objectsByModule:   objectsByModule,
		objectlessModules: make(map[string]bool),
		toCompile:         toCompile,
	}, stats, nil
}

func compileNativeModulePlan(world *World, checked *semantics.CheckedProgram, native nativeBuildTarget, opt BuildOptions, plan moduleBuildPlan, stats *BuildStats) error {
	if len(plan.toCompile) == 0 {
		sortBuildStats(stats)
		return nil
	}
	jobs := opt.Jobs
	if jobs <= 0 {
		jobs = runtime.NumCPU()
	}
	if jobs < 1 {
		jobs = 1
	}
	if jobs > len(plan.toCompile) {
		jobs = len(plan.toCompile)
	}

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

	worker := func() {
		defer wg.Done()
		for job := range jobsCh {
			if getErr() != nil {
				continue
			}
			funcs, err := LowerModule(checked, job.module)
			if err != nil {
				setErr(err)
				continue
			}
			if err := validateTargetAtomicIR(funcs, native.target); err != nil {
				setErr(err)
				continue
			}
			mu.Lock()
			stats.LoweredModules = append(stats.LoweredModules, job.module)
			mu.Unlock()

			dataPrefix := checked.GlobalDataByModule[job.module]
			if len(funcs) == 0 {
				mu.Lock()
				plan.objectlessModules[job.module] = true
				mu.Unlock()
				continue
			}
			obj, err := native.codegen(funcs, dataPrefix)
			if err != nil {
				setErr(err)
				continue
			}
			obj.Target = native.triple
			obj.Module = job.module
			obj.CompilerVersion = version.CompilerVersion
			obj.PublicAPIHash = plan.publicAPIHashes[job.module]
			obj.SrcHash = job.srcHash
			obj.WorldSigHash = job.depHash
			if err := cache.StoreCachedObject(world.Root, native.triple, plan.buildTag, obj); err != nil {
				setErr(err)
				continue
			}
			mu.Lock()
			stats.CompiledModules = append(stats.CompiledModules, job.module)
			plan.objectsByModule[job.module] = obj
			mu.Unlock()
		}
	}

	wg.Add(jobs)
	for i := 0; i < jobs; i++ {
		go worker()
	}
	for _, job := range plan.toCompile {
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
	if stats == nil {
		return
	}
	sort.Strings(stats.CacheHits)
	sort.Strings(stats.CompiledModules)
	sort.Strings(stats.LoweredModules)
}

func objectsFromModulePlan(plan moduleBuildPlan) ([]*Object, error) {
	objects := make([]*Object, 0, len(plan.modules))
	for _, module := range plan.modules {
		obj := plan.objectsByModule[module]
		if obj == nil {
			if plan.objectlessModules[module] {
				continue
			}
			return nil, fmt.Errorf("missing object for module '%s'", module)
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

func linkNativeExecutable(outputPath string, native nativeBuildTarget, opt BuildOptions, checked *semantics.CheckedProgram, objects []*Object, linkedObjects []linkedObject) error {
	actorsUsed, actorEntries, actorSpawnCount, err := collectActorEntries(checked)
	if err != nil {
		return err
	}
	actorStateUsed, actorStatePos := collectActorStateRuntimeUsagePosition(checked)
	actorRuntimeUsed, actorRuntimePos := collectActorRuntimeUsagePosition(checked)
	tasksUsed, tasksPos := collectTaskRuntimeUsagePosition(checked)
	taskGroupsUsed := collectTaskGroupRuntimeUsage(checked)
	typedTasksUsed, typedTaskMaxSlots := collectTypedTaskRuntimeUsage(checked)
	timeRuntimeUsed, timeRuntimePos := collectTimeRuntimeUsagePosition(checked)
	filesystemRuntimeUsed, filesystemRuntimePos := collectFilesystemRuntimeUsagePosition(checked)
	netRuntimeUsed, netRuntimePos := collectNetRuntimeUsagePosition(checked)
	distributedActorsUsed, distributedActorsPos := collectDistributedActorRuntimeUsagePosition(checked)
	if filesystemRuntimeUsed && native.triple != "linux-x64" {
		return targetRuntimeDiagnostic(filesystemRuntimePos, native.triple, "filesystem")
	}
	if netRuntimeUsed && native.triple != "linux-x64" {
		return targetRuntimeDiagnostic(netRuntimePos, native.triple, "networking")
	}
	if distributedActorsUsed && native.triple != "linux-x64" {
		return targetRuntimeDiagnostic(distributedActorsPos, native.triple, "distributed actors")
	}
	if timeRuntimeUsed && native.triple == "linux-x86" {
		return targetRuntimeDiagnostic(timeRuntimePos, native.triple, "time")
	}
	if tasksUsed && native.triple == "linux-x86" {
		return targetRuntimeDiagnostic(tasksPos, native.triple, "task")
	}
	if actorRuntimeUsed && native.triple == "linux-x86" {
		return targetRuntimeDiagnostic(actorRuntimePos, native.triple, "actors")
	}
	if actorStateUsed && native.triple == "linux-x86" {
		return targetRuntimeDiagnostic(actorStatePos, native.triple, "actors")
	}
	if actorSpawnCount > 1 && native.triple == "linux-x32" {
		return targetRuntimeDiagnostic(actorRuntimePos, native.triple, "multi-spawn actors")
	}
	if taskGroupsUsed && native.triple == "linux-x32" {
		return targetRuntimeDiagnostic(tasksPos, native.triple, "task group")
	}
	if typedTasksUsed && native.triple == "linux-x32" {
		return targetRuntimeDiagnostic(tasksPos, native.triple, "typed task")
	}
	runtimeUsed := actorsUsed || actorStateUsed || tasksUsed || taskGroupsUsed || typedTasksUsed || timeRuntimeUsed || filesystemRuntimeUsed || netRuntimeUsed || distributedActorsUsed
	if runtimeUsed && len(actorEntries) == 0 {
		actorEntries = []string{checked.MainName}
	}
	mainName := checked.MainName
	if opt.RuntimeObjectPath != "" && !runtimeUsed {
		return fmt.Errorf("runtime object override requires runtime usage (no actor/task/time/filesystem/networking/distributed actor builtins found)")
	}
	if runtimeUsed {
		usage := runtimeUsageProfile{
			actorStateUsed:        actorStateUsed,
			tasksUsed:             tasksUsed,
			taskGroupsUsed:        taskGroupsUsed,
			typedTasksUsed:        typedTasksUsed,
			typedTaskMaxSlots:     typedTaskMaxSlots,
			timeRuntimeUsed:       timeRuntimeUsed,
			filesystemUsed:        filesystemRuntimeUsed,
			netUsed:               netRuntimeUsed,
			distributedActorsUsed: distributedActorsUsed,
			actorSpawnCount:       actorSpawnCount,
		}
		runtimeMode, err := selectRuntimeMode(opt.Runtime, usage)
		if err != nil {
			return err
		}
		runtimeMode, err = runtimeModeForNativeTarget(native.triple, opt.Runtime, runtimeMode, usage)
		if err != nil {
			return err
		}
		if native.triple == "linux-x32" && opt.RuntimeObjectPath == "" && runtimeMode == RuntimeBuiltin {
			return fmt.Errorf("builtin runtime is not supported on target linux-x32; use runtime=selfhost for supported self-host runtime builds or remove runtime builtins")
		}
		var rt *Object
		needsDispatchGlue := true
		needsMainEntryIDGlue := true
		if opt.RuntimeObjectPath != "" {
			rt, err = ReadObject(opt.RuntimeObjectPath)
			if err != nil {
				return fmt.Errorf("read runtime object: %w", err)
			}
			if rt.Target == "" {
				return fmt.Errorf("runtime object has no target: %s", opt.RuntimeObjectPath)
			}
			if rt.Target != native.triple {
				return fmt.Errorf("runtime object target mismatch: got=%s want=%s", rt.Target, native.triple)
			}
		} else {
			switch runtimeMode {
			case RuntimeSelfHost:
				rt, err = buildEmbeddedSelfHostActorsRuntimeObject(native.triple, native.codegen)
			case RuntimeBuiltin:
				if native.backend.actorRuntime == nil {
					return fmt.Errorf("actors runtime is not supported on target %s", native.triple)
				}
				rt, err = native.backend.actorRuntime(actorEntries)
			}
			if err != nil {
				return err
			}
			annotateRuntimeObjectSignatures(rt)
		}
		if err := validateActorRuntimeObject(rt); err != nil {
			return err
		}
		if actorStateUsed {
			if err := validateActorStateRuntimeObject(rt); err != nil {
				return err
			}
		}
		if tasksUsed {
			if err := validateTaskRuntimeObject(rt); err != nil {
				return err
			}
		}
		if taskGroupsUsed {
			if err := validateTaskGroupRuntimeObject(rt); err != nil {
				return err
			}
		}
		if typedTasksUsed {
			if err := validateTypedTaskRuntimeObject(rt, typedTaskMaxSlots); err != nil {
				return err
			}
		}
		if timeRuntimeUsed {
			if err := validateTimeRuntimeObject(rt); err != nil {
				return err
			}
		}
		if filesystemRuntimeUsed {
			if err := validateFilesystemRuntimeObject(rt); err != nil {
				return err
			}
		}
		if netRuntimeUsed {
			if err := validateNetRuntimeObject(rt); err != nil {
				return err
			}
		}
		if distributedActorsUsed {
			if err := validateDistributedActorRuntimeObject(rt); err != nil {
				return err
			}
		}

		for _, sym := range rt.Symbols {
			if sym.Name == "__tetra_actor_dispatch" {
				needsDispatchGlue = false
			}
			if sym.Name == "__tetra_actor_main_entry_id" {
				needsMainEntryIDGlue = false
			}
		}

		if needsDispatchGlue || needsMainEntryIDGlue {
			var glueFuncs []IRFunc
			if needsDispatchGlue {
				dispatchFn, err := buildActorDispatchFunc(actorEntries, checked)
				if err != nil {
					return err
				}
				glueFuncs = append(glueFuncs, dispatchFn)
			}
			if needsMainEntryIDGlue {
				mainIDFn, err := buildActorMainEntryIDFunc(actorEntries[0])
				if err != nil {
					return err
				}
				glueFuncs = append(glueFuncs, mainIDFn)
			}
			if err := verifyIRFuncs(glueFuncs); err != nil {
				return fmt.Errorf("generated actor glue verifier: %w", err)
			}
			glueObj, err := native.codegen(glueFuncs, nil)
			if err != nil {
				return err
			}
			glueObj.Target = native.triple
			glueObj.Module = "__actorsglue"
			objects = append(objects, glueObj)
		}
		rt.Target = native.triple
		switch {
		case opt.RuntimeObjectPath != "":
			rt.Module = "__runtime"
		case runtimeMode == RuntimeBuiltin:
			rt.Module = "__actorsrt"
		default:
			rt.Module = "__selfhostrt"
		}
		objects = append(objects, rt)
		mainName = "__tetra_entry"
	}

	for _, linked := range linkedObjects {
		objects = append(objects, linked.obj)
	}

	if native.backend.link == nil {
		return fmt.Errorf("target backend has no linker: %s", native.triple)
	}
	return native.backend.link(outputPath, objects, mainName)
}

type runtimeUsageProfile struct {
	actorStateUsed        bool
	tasksUsed             bool
	taskGroupsUsed        bool
	typedTasksUsed        bool
	typedTaskMaxSlots     int
	timeRuntimeUsed       bool
	filesystemUsed        bool
	netUsed               bool
	distributedActorsUsed bool
	actorSpawnCount       int
}

func selectRuntimeMode(requested RuntimeMode, usage runtimeUsageProfile) (RuntimeMode, error) {
	switch requested {
	case RuntimeAuto:
		// Default to self-host runtime when its ABI can express the program surface.
		if usage.actorStateUsed || usage.tasksUsed || usage.taskGroupsUsed || usage.typedTasksUsed || usage.timeRuntimeUsed || usage.filesystemUsed || usage.netUsed || usage.distributedActorsUsed || usage.typedTaskMaxSlots > 4 || usage.actorSpawnCount > 1 {
			return RuntimeBuiltin, nil
		}
		return RuntimeSelfHost, nil
	case RuntimeSelfHost:
		if usage.distributedActorsUsed {
			return 0, fmt.Errorf("self-host runtime does not support distributed actors; use runtime=auto or runtime=builtin")
		}
		if usage.taskGroupsUsed {
			return 0, fmt.Errorf("self-host runtime does not support task groups; use runtime=auto or runtime=builtin")
		}
		if usage.typedTasksUsed {
			return 0, fmt.Errorf("self-host runtime does not support typed task handles; use runtime=auto or runtime=builtin")
		}
		if usage.actorSpawnCount > 1 {
			return 0, fmt.Errorf("self-host runtime supports at most one spawned actor; use runtime=auto or runtime=builtin")
		}
		return RuntimeSelfHost, nil
	case RuntimeBuiltin:
		return RuntimeBuiltin, nil
	default:
		return 0, fmt.Errorf("unsupported runtime mode: %d", requested)
	}
}

func runtimeModeForNativeTarget(target string, requested RuntimeMode, selected RuntimeMode, usage runtimeUsageProfile) (RuntimeMode, error) {
	if target != "linux-x32" || requested != RuntimeAuto || selected != RuntimeBuiltin {
		return selected, nil
	}
	if _, err := selectRuntimeMode(RuntimeSelfHost, usage); err != nil {
		return selected, nil
	}
	return RuntimeSelfHost, nil
}

func requiredActorRuntimeSymbols() []string {
	return runtimeabi.RequiredActorSymbols()
}

func requiredActorStateRuntimeSymbols() []string {
	return runtimeabi.RequiredActorStateSymbols()
}

func requiredDistributedActorRuntimeSymbols() []string {
	return runtimeabi.RequiredDistributedActorSymbols()
}

func requiredTaskRuntimeSymbols() []string {
	return runtimeabi.RequiredTaskSymbols()
}

func requiredTaskGroupRuntimeSymbols() []string {
	return runtimeabi.RequiredTaskGroupSymbols()
}

func requiredTypedTaskRuntimeSymbols(maxSlots int) []string {
	return runtimeabi.RequiredTypedTaskSymbols(maxSlots)
}

func requiredTimeRuntimeSymbols() []string {
	return runtimeabi.RequiredTimeSymbols()
}

func requiredFilesystemRuntimeSymbols() []string {
	return runtimeabi.RequiredFilesystemSymbols()
}

func requiredNetRuntimeSymbols() []string {
	return runtimeabi.RequiredNetSymbols()
}

type runtimeObjectSlotSignature struct {
	paramSlots  int
	returnSlots int
}

func runtimeObjectSignature(name string) (runtimeObjectSlotSignature, bool) {
	sig, ok := runtimeabi.SignatureForSymbol(name)
	if !ok {
		return runtimeObjectSlotSignature{}, false
	}
	return runtimeObjectSlotSignature{paramSlots: sig.ParamSlots, returnSlots: sig.ReturnSlots}, true
}

func annotateRuntimeObjectSignatures(rt *Object) {
	if rt == nil {
		return
	}
	for i := range rt.Symbols {
		if rt.Symbols[i].HasSignature {
			continue
		}
		sig, ok := runtimeObjectSignature(rt.Symbols[i].Name)
		if !ok {
			continue
		}
		rt.Symbols[i].HasSignature = true
		rt.Symbols[i].ParamSlots = sig.paramSlots
		rt.Symbols[i].ReturnSlots = sig.returnSlots
	}
}

func validateRuntimeObjectSymbols(rt *Object, missingObject string, required []string) error {
	if rt == nil {
		return fmt.Errorf("%s", missingObject)
	}
	symbols := make(map[string]Symbol, len(rt.Symbols))
	for _, sym := range rt.Symbols {
		symbols[sym.Name] = sym
	}
	for _, name := range required {
		sym, ok := symbols[name]
		if !ok {
			return fmt.Errorf("runtime object missing required symbol '%s'", name)
		}
		expected, ok := runtimeObjectSignature(name)
		if !ok || !sym.HasSignature {
			continue
		}
		if sym.ParamSlots != expected.paramSlots || sym.ReturnSlots != expected.returnSlots {
			return fmt.Errorf(
				"runtime object symbol '%s' signature mismatch: params=%d want=%d returns=%d want=%d",
				name,
				sym.ParamSlots,
				expected.paramSlots,
				sym.ReturnSlots,
				expected.returnSlots,
			)
		}
	}
	return nil
}

func validateActorRuntimeObject(rt *Object) error {
	return validateRuntimeObjectSymbols(rt, "missing actors runtime object", requiredActorRuntimeSymbols())
}

func validateActorStateRuntimeObject(rt *Object) error {
	return validateRuntimeObjectSymbols(rt, "missing actors runtime object", requiredActorStateRuntimeSymbols())
}

func validateDistributedActorRuntimeObject(rt *Object) error {
	return validateRuntimeObjectSymbols(rt, "missing distributed actors runtime object", requiredDistributedActorRuntimeSymbols())
}

func validateTimeRuntimeObject(rt *Object) error {
	return validateRuntimeObjectSymbols(rt, "missing time runtime object", requiredTimeRuntimeSymbols())
}

func validateFilesystemRuntimeObject(rt *Object) error {
	return validateRuntimeObjectSymbols(rt, "missing filesystem runtime object", requiredFilesystemRuntimeSymbols())
}

func validateNetRuntimeObject(rt *Object) error {
	return validateRuntimeObjectSymbols(rt, "missing networking runtime object", requiredNetRuntimeSymbols())
}

func validateTypedTaskRuntimeObject(rt *Object, maxSlots int) error {
	return validateRuntimeObjectSymbols(rt, "missing typed task runtime object", requiredTypedTaskRuntimeSymbols(maxSlots))
}

func validateTaskRuntimeObject(rt *Object) error {
	return validateRuntimeObjectSymbols(rt, "missing task runtime object", requiredTaskRuntimeSymbols())
}

func validateTaskGroupRuntimeObject(rt *Object) error {
	return validateRuntimeObjectSymbols(rt, "missing task group runtime object", requiredTaskGroupRuntimeSymbols())
}

func buildObjectFileWithStatsOpt(inputPath, outputPath string, tgt ctarget.Target, opt BuildOptions) (*BuildStats, error) {
	requireMain := opt.Emit == EmitObject && !opt.InterfaceOnly
	codegenOptions := nativeCodegenOptionsForTarget(tgt, opt)

	world, err := loadWorldForBuild(inputPath, opt)
	if err != nil {
		return nil, err
	}
	if err := validateTargetExportedFFIAST(world, tgt.Triple); err != nil {
		return nil, err
	}
	checked, err := semantics.CheckWorldOpt(world, semantics.CheckOptions{RequireMain: requireMain})
	if err != nil {
		return nil, translateTargetExportedFFISemanticError(err, tgt.Triple)
	}
	if opt.InterfaceOnly {
		return interfaceOnlyBuildStats(world), nil
	}
	if err := rejectInterfaceModulesForCodegen(world); err != nil {
		return nil, err
	}
	if err := validateTargetExportedFFIABI(checked, tgt.Triple); err != nil {
		return nil, err
	}

	funcs, err := LowerModule(checked, world.EntryModule)
	if err != nil {
		return nil, err
	}
	if err := validateTargetAtomicIR(funcs, tgt); err != nil {
		return nil, err
	}

	var obj *Object
	dataPrefix := checked.GlobalDataByModule[world.EntryModule]
	switch tgt.OS {
	case ctarget.OSLinux:
		switch tgt.Triple {
		case "linux-x86":
			obj, err = linux_x86.CodegenObjectLinuxX86WithOptionsAndDataPrefix(funcs, dataPrefix, codegenOptions)
		case "linux-x64":
			obj, err = linux_x64.CodegenObjectLinuxX64WithOptionsAndDataPrefix(funcs, dataPrefix, codegenOptions)
		case "linux-x32":
			obj, err = linux_x32.CodegenObjectLinuxX32WithOptionsAndDataPrefix(funcs, dataPrefix, codegenOptions)
		default:
			return nil, fmt.Errorf("target backend not implemented: %s (object codegen blocked)", tgt.Triple)
		}
	case ctarget.OSWindows:
		obj, err = windows_x64.CodegenObjectWindowsX64WithOptionsAndDataPrefix(funcs, dataPrefix, codegenOptions)
	case ctarget.OSMacOS:
		obj, err = macos_x64.CodegenObjectMacOSX64WithOptionsAndDataPrefix(funcs, dataPrefix, codegenOptions)
	default:
		return nil, fmt.Errorf("unsupported target: %s", tgt.Triple)
	}
	if err != nil {
		return nil, err
	}

	obj.Target = tgt.Triple
	moduleName := world.EntryModule
	if moduleName == "" {
		moduleName = "__entry"
	}
	obj.Module = moduleName
	obj.CompilerVersion = version.CompilerVersion
	file := world.ByModule[world.EntryModule]
	if file != nil {
		obj.SrcHash = sha256.Sum256(file.Src)
		hash, err := InterfaceFingerprintFromSource(file.Src, file.Path)
		if err != nil {
			return nil, err
		}
		obj.PublicAPIHash = hash
	}
	obj.WorldSigHash = cache.WorldSigHash(checked)

	if err := WriteObject(outputPath, obj); err != nil {
		return nil, err
	}
	return &BuildStats{
		CompiledModules: []string{moduleName},
		LoweredModules:  []string{moduleName},
	}, nil
}

func buildWASM32WASIWithStatsOpt(inputPath, outputPath string, tgt ctarget.Target, opt BuildOptions) (*BuildStats, error) {
	if tgt.Triple != "wasm32-wasi" {
		return nil, fmt.Errorf("internal error: unexpected target for wasm backend: %s", tgt.Triple)
	}
	if opt.Emit != EmitExe {
		return nil, fmt.Errorf("wasm32-wasi supports only --emit=exe in this wave")
	}
	if opt.RuntimeObjectPath != "" {
		return nil, fmt.Errorf("wasm32-wasi does not support --runtime-object in this wave")
	}
	if len(opt.LinkObjectPaths) > 0 {
		return nil, fmt.Errorf("wasm32-wasi does not support --link-object in this wave")
	}

	world, err := loadWorldForBuild(inputPath, opt)
	if err != nil {
		return nil, err
	}
	checked, err := semantics.CheckWorldOpt(world, semantics.CheckOptions{RequireMain: !opt.InterfaceOnly})
	if err != nil {
		return nil, err
	}
	if opt.InterfaceOnly {
		return interfaceOnlyBuildStats(world), nil
	}
	if err := rejectInterfaceModulesForCodegen(world); err != nil {
		return nil, err
	}

	modules := make([]string, 0, len(world.ByModule))
	for module := range world.ByModule {
		modules = append(modules, module)
	}
	sort.Strings(modules)

	var funcs []IRFunc
	var dataPrefix [][]byte
	globalOffset := 0
	stats := &BuildStats{
		CompiledModules: make([]string, 0, len(modules)),
		LoweredModules:  make([]string, 0, len(modules)),
	}
	for _, module := range modules {
		moduleFuncs, err := LowerModule(checked, module)
		if err != nil {
			return nil, err
		}
		stats.LoweredModules = append(stats.LoweredModules, module)
		stats.CompiledModules = append(stats.CompiledModules, module)
		funcs = append(funcs, relocateWASMGlobalSlots(moduleFuncs, globalOffset)...)
		moduleData := checked.GlobalDataByModule[module]
		dataPrefix = append(dataPrefix, moduleData...)
		globalOffset += len(moduleData)
	}
	if err := validateWASMIRPolicy(tgt.Triple, funcs); err != nil {
		return nil, err
	}
	if err := rejectUnsupportedWASMRuntimeBuiltins(funcs, tgt.Triple); err != nil {
		return nil, err
	}

	obj, err := wasm32_wasi.CodegenObjectWithDataPrefix(funcs, checked.MainName, dataPrefix)
	if err != nil {
		return nil, err
	}
	wasmBytes, err := wasm32_wasi.LinkObject(obj)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(outputPath, wasmBytes, 0o755); err != nil {
		return nil, err
	}
	if err := emitUIArtifacts(outputPath, tgt.Triple, checked); err != nil {
		return nil, err
	}
	return stats, nil
}

func buildWASM32WEBWithStatsOpt(inputPath, outputPath string, tgt ctarget.Target, opt BuildOptions) (*BuildStats, error) {
	if tgt.Triple != "wasm32-web" {
		return nil, fmt.Errorf("internal error: unexpected target for wasm backend: %s", tgt.Triple)
	}
	if opt.Emit != EmitExe {
		return nil, fmt.Errorf("wasm32-web supports only --emit=exe in this wave")
	}
	if opt.RuntimeObjectPath != "" {
		return nil, fmt.Errorf("wasm32-web does not support --runtime-object in this wave")
	}
	if len(opt.LinkObjectPaths) > 0 {
		return nil, fmt.Errorf("wasm32-web does not support --link-object in this wave")
	}

	world, err := loadWorldForBuild(inputPath, opt)
	if err != nil {
		return nil, err
	}
	checked, err := semantics.CheckWorldOpt(world, semantics.CheckOptions{RequireMain: !opt.InterfaceOnly})
	if err != nil {
		return nil, err
	}
	if opt.InterfaceOnly {
		return interfaceOnlyBuildStats(world), nil
	}
	if err := rejectInterfaceModulesForCodegen(world); err != nil {
		return nil, err
	}

	modules := make([]string, 0, len(world.ByModule))
	for module := range world.ByModule {
		modules = append(modules, module)
	}
	sort.Strings(modules)

	var funcs []IRFunc
	var dataPrefix [][]byte
	globalOffset := 0
	stats := &BuildStats{
		CompiledModules: make([]string, 0, len(modules)),
		LoweredModules:  make([]string, 0, len(modules)),
	}
	for _, module := range modules {
		moduleFuncs, err := LowerModule(checked, module)
		if err != nil {
			return nil, err
		}
		stats.LoweredModules = append(stats.LoweredModules, module)
		stats.CompiledModules = append(stats.CompiledModules, module)
		funcs = append(funcs, relocateWASMGlobalSlots(moduleFuncs, globalOffset)...)
		moduleData := checked.GlobalDataByModule[module]
		dataPrefix = append(dataPrefix, moduleData...)
		globalOffset += len(moduleData)
	}
	if err := validateWASMIRPolicy(tgt.Triple, funcs); err != nil {
		return nil, err
	}
	if err := rejectUnsupportedWASMRuntimeBuiltins(funcs, tgt.Triple); err != nil {
		return nil, err
	}

	obj, err := wasm32_web.CodegenObjectWithDataPrefix(funcs, checked.MainName, dataPrefix)
	if err != nil {
		return nil, err
	}
	wasmBytes, err := wasm32_web.LinkObject(obj)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(outputPath, wasmBytes, 0o755); err != nil {
		return nil, err
	}

	loaderPath := wasmWebLoaderPath(outputPath)
	loader := wasm32_web.LoaderModule(filepath.Base(outputPath))
	if err := os.WriteFile(loaderPath, loader, 0o644); err != nil {
		return nil, err
	}
	if err := emitUIArtifacts(outputPath, tgt.Triple, checked); err != nil {
		return nil, err
	}
	return stats, nil
}

func wasmWebLoaderPath(outputPath string) string {
	ext := filepath.Ext(outputPath)
	if strings.EqualFold(ext, ".wasm") {
		return strings.TrimSuffix(outputPath, ext) + ".mjs"
	}
	return outputPath + ".mjs"
}

func relocateWASMGlobalSlots(funcs []IRFunc, offset int) []IRFunc {
	if offset == 0 {
		return funcs
	}
	out := make([]IRFunc, len(funcs))
	for i, fn := range funcs {
		out[i] = fn
		if len(fn.Instrs) == 0 {
			continue
		}
		out[i].Instrs = append([]ir.IRInstr(nil), fn.Instrs...)
		for j := range out[i].Instrs {
			switch out[i].Instrs[j].Kind {
			case ir.IRLoadGlobal, ir.IRStoreGlobal:
				out[i].Instrs[j].Local += offset
			}
		}
	}
	return out
}

func rejectUnsupportedWASMRuntimeBuiltins(funcs []IRFunc, target string) error {
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind != ir.IRCall {
				continue
			}
			runtimeName, ok := wasmRuntimeNameForBuiltin(instr.Name)
			if !ok {
				continue
			}
			return targetRuntimeDiagnostic(instr.Pos, target, runtimeName)
		}
	}
	return nil
}

func wasmRuntimeNameForBuiltin(name string) (string, bool) {
	switch {
	case strings.HasPrefix(name, "__tetra_actor_"):
		return "actors", true
	case strings.HasPrefix(name, "__tetra_task_"):
		return "task", true
	case strings.HasPrefix(name, "__tetra_fs_"):
		return "filesystem", true
	case strings.HasPrefix(name, "__tetra_net_"):
		return "networking", true
	case strings.HasPrefix(name, "__tetra_time_"), name == "__tetra_sleep_ms", name == "__tetra_sleep_until_ms", name == "__tetra_deadline_ms", name == "__tetra_timer_ready_ms":
		return "time", true
	default:
		return "", false
	}
}

func targetRuntimeDiagnostic(pos frontend.Position, target string, runtimeName string) error {
	hint := "Build this source for a native x64 target or remove the runtime builtin for this WASM target."
	if !strings.HasPrefix(target, "wasm32-") {
		hint = fmt.Sprintf("Build this source for linux-x64 or remove the %s runtime builtin for this target.", runtimeName)
	}
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code:     DiagnosticCodeTargetRuntime,
		Message:  fmt.Sprintf("%s runtime not supported on %s", runtimeName, target),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint:     hint,
	}}
}

type wasmIRPolicy struct {
	builtin  string
	category string
}

func validateWASMIRPolicy(target string, funcs []IRFunc) error {
	if !strings.HasPrefix(target, "wasm32-") {
		return nil
	}
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			policy, blocked := blockedWASMIRPolicy(instr.Kind)
			if !blocked {
				continue
			}
			return targetWASMPolicyDiagnostic(instr.Pos, target, policy)
		}
	}
	return nil
}

func blockedWASMIRPolicy(kind ir.IRInstrKind) (wasmIRPolicy, bool) {
	switch kind {
	case ir.IRAllocBytes:
		return wasmIRPolicy{builtin: "core.alloc_bytes", category: "raw memory allocation"}, true
	case ir.IRCapIO:
		return wasmIRPolicy{builtin: "core.cap_io", category: "capability token construction"}, true
	case ir.IRCapMem:
		return wasmIRPolicy{builtin: "core.cap_mem", category: "capability token construction"}, true
	case ir.IRMemReadI32:
		return wasmIRPolicy{builtin: "core.load_i32", category: "raw memory access"}, true
	case ir.IRMemWriteI32:
		return wasmIRPolicy{builtin: "core.store_i32", category: "raw memory access"}, true
	case ir.IRMemReadU8:
		return wasmIRPolicy{builtin: "core.load_u8", category: "raw memory access"}, true
	case ir.IRMemWriteU8:
		return wasmIRPolicy{builtin: "core.store_u8", category: "raw memory access"}, true
	case ir.IRMemReadPtr:
		return wasmIRPolicy{builtin: "core.load_ptr", category: "raw pointer memory access"}, true
	case ir.IRMemWritePtr:
		return wasmIRPolicy{builtin: "core.store_ptr", category: "raw pointer memory access"}, true
	case ir.IRMemWriteArchPtr:
		return wasmIRPolicy{builtin: "core.store_arch_ptr", category: "raw architectural pointer memory access"}, true
	case ir.IRMemReadI32Offset:
		return wasmIRPolicy{builtin: "core.load_i32", category: "raw memory access"}, true
	case ir.IRMemWriteI32Offset:
		return wasmIRPolicy{builtin: "core.store_i32", category: "raw memory access"}, true
	case ir.IRMemReadU8Offset:
		return wasmIRPolicy{builtin: "core.load_u8", category: "raw memory access"}, true
	case ir.IRMemWriteU8Offset:
		return wasmIRPolicy{builtin: "core.store_u8", category: "raw memory access"}, true
	case ir.IRMemReadPtrOffset:
		return wasmIRPolicy{builtin: "core.load_ptr", category: "raw pointer memory access"}, true
	case ir.IRMemWritePtrOffset:
		return wasmIRPolicy{builtin: "core.store_ptr", category: "raw pointer memory access"}, true
	case ir.IRMemWriteArchPtrOffset:
		return wasmIRPolicy{builtin: "core.store_arch_ptr", category: "raw architectural pointer memory access"}, true
	case ir.IRPtrAdd:
		return wasmIRPolicy{builtin: "core.ptr_add", category: "raw pointer arithmetic"}, true
	case ir.IRMmioReadI32:
		return wasmIRPolicy{builtin: "core.mmio_read_i32", category: "MMIO"}, true
	case ir.IRMmioWriteI32:
		return wasmIRPolicy{builtin: "core.mmio_write_i32", category: "MMIO"}, true
	case ir.IRCtxSwitch:
		return wasmIRPolicy{builtin: "core.ctx_switch", category: "context switching"}, true
	default:
		return wasmIRPolicy{}, false
	}
}

func targetWASMPolicyDiagnostic(pos frontend.Position, target string, policy wasmIRPolicy) error {
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code:     DiagnosticCodeTargetRuntime,
		Message:  fmt.Sprintf("%s target does not support %s (%s); unsupported on WASM targets by policy", target, policy.builtin, policy.category),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint:     "Build this unsafe/capability memory path for a native x64 target, or replace it with the supported WASM-safe slice/island surface.",
	}}
}

func emitUIArtifacts(outputPath string, target string, checked *semantics.CheckedProgram) error {
	bundle, err := lower.LowerUI(checked)
	if err != nil {
		return err
	}
	if bundle == nil || len(bundle.Views) == 0 {
		return nil
	}
	base := uiArtifactBasePath(outputPath)
	uiJSONPath := base + ".ui.json"
	raw, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(uiJSONPath, raw, 0o644); err != nil {
		return err
	}
	toolkitBundle, err := lower.LowerUIToolkit(bundle)
	if err != nil {
		return err
	}
	if toolkitBundle != nil {
		toolkitPath := base + ".ui.toolkit.json"
		toolkitRaw, err := json.MarshalIndent(toolkitBundle, "", "  ")
		if err != nil {
			return err
		}
		toolkitRaw = append(toolkitRaw, '\n')
		if err := os.WriteFile(toolkitPath, toolkitRaw, 0o644); err != nil {
			return err
		}
	}
	if target == "wasm32-web" {
		uiModulePath := base + ".ui.web.mjs"
		uiModule := wasm32_web.UIModule(filepath.Base(uiJSONPath))
		if err := os.WriteFile(uiModulePath, uiModule, 0o644); err != nil {
			return err
		}
		htmlPath := base + ".ui.html"
		html := wasm32_web.UIHTMLPage(filepath.Base(outputPath), filepath.Base(wasmWebLoaderPath(outputPath)), filepath.Base(uiModulePath))
		if err := os.WriteFile(htmlPath, html, 0o644); err != nil {
			return err
		}
		return nil
	}
	if strings.HasPrefix(target, "wasm32-") {
		return nil
	}
	shellPath := base + ".ui.shell.txt"
	if err := os.WriteFile(shellPath, native_shell.Render(bundle), 0o644); err != nil {
		return err
	}
	shellJSONPath := base + ".ui.shell.json"
	if err := os.WriteFile(shellJSONPath, native_shell.RenderJSON(bundle), 0o644); err != nil {
		return err
	}
	return nil
}

func uiArtifactBasePath(outputPath string) string {
	ext := filepath.Ext(outputPath)
	switch {
	case strings.EqualFold(ext, ".wasm"):
		return strings.TrimSuffix(outputPath, ext)
	case strings.EqualFold(ext, ".exe"):
		return strings.TrimSuffix(outputPath, ext)
	default:
		return outputPath
	}
}

func loadWorldForBuild(inputPath string, opt BuildOptions) (*World, error) {
	if opt.ProjectRoot == "" && len(opt.SourceRoots) == 0 && len(opt.DependencyRoots) == 0 {
		return LoadWorld(inputPath)
	}
	return LoadWorldOpt(inputPath, WorldOptions{
		Root:            opt.ProjectRoot,
		SourceRoots:     opt.SourceRoots,
		DependencyRoots: opt.DependencyRoots,
	})
}

func rejectInterfaceModulesForCodegen(world *World) error {
	modules := sortedInterfaceModules(world)
	if len(modules) == 0 {
		return nil
	}
	return fmt.Errorf("interface-only module '%s' cannot be linked; use --interface-only or provide source/object implementation", modules[0])
}

func readLinkObjects(paths []string, target string) ([]linkedObject, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	var linked []linkedObject
	seenPaths := make(map[string]string, len(paths))
	seenSymbols := make(map[string]linkedObject)
	for _, path := range paths {
		if path == "" {
			continue
		}
		pathKey, err := filepath.Abs(path)
		if err != nil {
			pathKey = filepath.Clean(path)
		}
		if first, exists := seenPaths[pathKey]; exists {
			return nil, fmt.Errorf("duplicate link object path: %s and %s", first, path)
		}
		seenPaths[pathKey] = path
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read link object %s: %w", path, err)
		}
		obj, err := ReadObject(path)
		if err != nil {
			return nil, fmt.Errorf("read link object %s: %w", path, err)
		}
		if obj.Target == "" {
			return nil, fmt.Errorf("link object has no target: %s", path)
		}
		if obj.Target != target {
			return nil, fmt.Errorf("link object target mismatch: got=%s want=%s (%s)", obj.Target, target, path)
		}
		if obj.Module == "" {
			return nil, fmt.Errorf("link object has no module identity: %s", path)
		}
		if obj.CompilerVersion != "" && obj.CompilerVersion != version.CompilerVersion {
			return nil, fmt.Errorf("link object compiler version mismatch: got=%s want=%s (%s)", obj.CompilerVersion, version.CompilerVersion, path)
		}
		current := linkedObject{path: path, obj: obj, contentHash: sha256.Sum256(raw)}
		if err := validateLinkedObjectSymbols(current, seenSymbols); err != nil {
			return nil, err
		}
		linked = append(linked, current)
	}
	return linked, nil
}

func validateLinkedObjectSymbols(current linkedObject, seen map[string]linkedObject) error {
	if current.obj == nil {
		return nil
	}
	local := make(map[string]struct{}, len(current.obj.Symbols))
	for _, sym := range current.obj.Symbols {
		if sym.Name == "" {
			return fmt.Errorf("link object has empty symbol name: %s", current.path)
		}
		if _, exists := local[sym.Name]; exists {
			return fmt.Errorf("duplicate symbol '%s' inside link object %s", sym.Name, current.path)
		}
		local[sym.Name] = struct{}{}
		if first, exists := seen[sym.Name]; exists {
			return fmt.Errorf("duplicate symbol '%s' in link objects: %s and %s", sym.Name, first.path, current.path)
		}
		seen[sym.Name] = current
	}
	return nil
}

func validateInterfaceImplementationProviders(world *World, checked *semantics.CheckedProgram, linked []linkedObject) error {
	modules := sortedInterfaceModules(world)
	if len(modules) == 0 {
		return nil
	}
	providers := make(map[string]linkedObject, len(modules))
	interfaceSet := make(map[string]struct{}, len(modules))
	for _, module := range modules {
		interfaceSet[module] = struct{}{}
	}
	for _, linked := range linked {
		obj := linked.obj
		if obj == nil {
			continue
		}
		if _, ok := interfaceSet[obj.Module]; !ok {
			continue
		}
		if first, exists := providers[obj.Module]; exists {
			return fmt.Errorf("duplicate implementation object for interface module '%s': %s and %s", obj.Module, first.path, linked.path)
		}
		if obj.PublicAPIHash == "" {
			return fmt.Errorf("implementation object for interface module '%s' has no public API hash: %s", obj.Module, linked.path)
		}
		want := world.InterfaceHashes[obj.Module]
		if want == "" {
			return fmt.Errorf("missing interface hash for module '%s'", obj.Module)
		}
		if obj.PublicAPIHash != want {
			return fmt.Errorf("public API hash mismatch for interface module '%s': object %s, interface %s (%s)", obj.Module, obj.PublicAPIHash, want, linked.path)
		}
		if err := validateInterfaceImplementationSymbols(world, checked, obj.Module, obj, linked.path); err != nil {
			return err
		}
		providers[obj.Module] = linked
	}
	for _, module := range modules {
		if _, ok := providers[module]; !ok {
			return fmt.Errorf("missing implementation object for interface module '%s'; pass --link-object with a matching TOBJ", module)
		}
	}
	return nil
}

func validateInterfaceImplementationSymbols(world *World, checked *semantics.CheckedProgram, module string, obj *Object, path string) error {
	symbols := make(map[string]Symbol, len(obj.Symbols))
	for _, sym := range obj.Symbols {
		symbols[sym.Name] = sym
	}
	for _, name := range unsupportedInterfaceModuleGenericSymbols(world, module) {
		return fmt.Errorf("implementation object for interface module '%s' cannot satisfy generic export '%s'; precompiled link objects require monomorphic exported functions (%s)", module, name, path)
	}
	for _, name := range expectedInterfaceModuleSymbols(world, module) {
		sym, ok := symbols[name]
		if !ok {
			return fmt.Errorf("implementation object for interface module '%s' missing exported symbol '%s' (%s)", module, name, path)
		}
		if !sym.HasSignature {
			return fmt.Errorf("implementation object for interface module '%s' symbol '%s' missing signature metadata (%s)", module, name, path)
		}
		if checked == nil || checked.FuncSigs == nil {
			continue
		}
		want, ok := checked.FuncSigs[name]
		if !ok {
			continue
		}
		if sym.ParamSlots != want.ParamSlots || sym.ReturnSlots != want.ReturnSlots {
			return fmt.Errorf(
				"implementation object for interface module '%s' symbol '%s' signature mismatch: params=%d want=%d returns=%d want=%d (%s)",
				module,
				name,
				sym.ParamSlots,
				want.ParamSlots,
				sym.ReturnSlots,
				want.ReturnSlots,
				path,
			)
		}
	}
	return nil
}

func unsupportedInterfaceModuleGenericSymbols(world *World, module string) []string {
	if world == nil || world.ByModule == nil {
		return nil
	}
	file := world.ByModule[module]
	if file == nil {
		return nil
	}
	var symbols []string
	for _, fn := range file.Funcs {
		if fn == nil || fn.Synthetic || len(fn.TypeParams) == 0 {
			continue
		}
		name := fn.Name
		if fn.ExtensionOf == "" {
			name = qualifyObjectSymbol(module, fn.Name)
		}
		symbols = append(symbols, name)
	}
	sort.Strings(symbols)
	return symbols
}

func expectedInterfaceModuleSymbols(world *World, module string) []string {
	if world == nil || world.ByModule == nil {
		return nil
	}
	file := world.ByModule[module]
	if file == nil {
		return nil
	}
	var symbols []string
	for _, fn := range file.Funcs {
		if fn == nil || fn.Synthetic || len(fn.TypeParams) > 0 {
			continue
		}
		name := fn.Name
		if fn.ExtensionOf == "" {
			name = qualifyObjectSymbol(module, fn.Name)
		}
		symbols = append(symbols, name)
	}
	sort.Strings(symbols)
	return symbols
}

func qualifyObjectSymbol(module, name string) string {
	if module == "" || strings.HasPrefix(name, module+".") {
		return name
	}
	return module + "." + name
}

func interfaceOnlyBuildStats(world *World) *BuildStats {
	return &BuildStats{InterfaceModules: sortedInterfaceModules(world)}
}

func sortedInterfaceModules(world *World) []string {
	if world == nil || len(world.InterfaceModules) == 0 {
		return nil
	}
	modules := make([]string, 0, len(world.InterfaceModules))
	for module := range world.InterfaceModules {
		modules = append(modules, module)
	}
	sort.Strings(modules)
	return modules
}

func buildTagFromOptions(opt BuildOptions, linkedObjects []linkedObject) string {
	var tags []string
	if opt.IslandsDebug {
		tags = append(tags, "islands-debug")
	}
	if opt.DebugInfo {
		tags = append(tags, "debug-info")
	}
	if opt.ReleaseOptimize {
		tags = append(tags, "release-opt")
	}
	if opt.InterfaceOnly {
		tags = append(tags, "interface-only")
	}
	if len(linkedObjects) > 0 {
		entries := make([]string, 0, len(linkedObjects))
		for _, linked := range linkedObjects {
			module := ""
			if linked.obj != nil {
				module = linked.obj.Module
			}
			entries = append(entries, fmt.Sprintf("%s:%x", module, linked.contentHash))
		}
		sort.Strings(entries)
		tags = append(tags, "link="+strings.Join(entries, ","))
	}
	return strings.Join(tags, "+")
}

func collectActorEntries(checked *semantics.CheckedProgram) (bool, []string, int, error) {
	if checked == nil {
		return false, nil, 0, nil
	}
	used := false
	spawnCount := 0
	targets := make(map[string]struct{})

	var walkExpr func(frontend.Expr) error
	var walkStmt func(frontend.Stmt) error

	walkExpr = func(expr frontend.Expr) error {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			switch name {
			case "core.spawn":
				used = true
				spawnCount++
				if len(e.Args) == 1 {
					if lit, ok := e.Args[0].(*frontend.StringLitExpr); ok {
						name := string(lit.Value)
						if name != "" {
							targets[name] = struct{}{}
						}
					}
				}
			case "core.spawn_remote":
				used = true
				if len(e.Args) == 2 {
					if lit, ok := e.Args[1].(*frontend.StringLitExpr); ok {
						name := string(lit.Value)
						if name != "" {
							targets[name] = struct{}{}
						}
					}
				}
			case "core.actor_node_connect", "core.actor_node_status":
				used = true
			case "core.task_spawn_i32":
				used = true
				spawnCount++
				if len(e.Args) == 1 {
					if lit, ok := e.Args[0].(*frontend.StringLitExpr); ok {
						name := string(lit.Value)
						if name != "" {
							targets[name] = struct{}{}
						}
					}
				}
			case "core.task_spawn_group_i32":
				used = true
				if len(e.Args) == 2 {
					if lit, ok := e.Args[1].(*frontend.StringLitExpr); ok {
						name := string(lit.Value)
						if name != "" {
							targets[name] = struct{}{}
						}
					}
				}
			case "core.task_spawn_i32_typed":
				used = true
				if len(e.TypeArgs) == 1 && e.TypeArgs[0].Name != "" && len(e.Args) == 1 {
					if lit, ok := e.Args[0].(*frontend.StringLitExpr); ok {
						name := string(lit.Value)
						if name != "" {
							targets[typedTaskRuntimeWrapperName(name, e.TypeArgs[0].Name)] = struct{}{}
						}
					}
				}
			case "core.task_spawn_group_i32_typed":
				used = true
				if len(e.TypeArgs) == 1 && e.TypeArgs[0].Name != "" && len(e.Args) == 2 {
					if lit, ok := e.Args[1].(*frontend.StringLitExpr); ok {
						name := string(lit.Value)
						if name != "" {
							targets[typedTaskRuntimeWrapperName(name, e.TypeArgs[0].Name)] = struct{}{}
						}
					}
				}
			case "core.task_group_open", "core.task_group_close", "core.task_group_cancel", "core.task_group_current", "core.task_group_status",
				"core.task_is_canceled", "core.task_checkpoint":
				used = true
			case "core.time_now_ms", "core.sleep_ms", "core.sleep_until", "core.deadline_ms", "core.timer_ready":
				used = true
			case "core.task_join_i32", "core.task_join_result_i32", "core.task_join_until_i32", "core.task_poll_i32", "core.select2_i32":
				used = true
			case "core.task_join_i32_typed", "core.task_join_group_i32_typed":
				used = true
			case "core.send", "core.send_msg", "core.send_typed", "core.recv", "core.recv_msg", "core.recv_poll", "core.recv_until", "core.recv_msg_until", "core.recv_typed", "core.self", "core.sender", "core.yield":
				used = true
			}
			for _, arg := range e.Args {
				if err := walkExpr(arg); err != nil {
					return err
				}
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				if err := walkExpr(field.Value); err != nil {
					return err
				}
			}
		case *frontend.FieldAccessExpr:
			return walkExpr(e.Base)
		case *frontend.IndexExpr:
			if err := walkExpr(e.Base); err != nil {
				return err
			}
			return walkExpr(e.Index)
		case *frontend.BinaryExpr:
			if err := walkExpr(e.Left); err != nil {
				return err
			}
			return walkExpr(e.Right)
		case *frontend.UnaryExpr:
			return walkExpr(e.X)
		case *frontend.TryExpr:
			return walkExpr(e.X)
		case *frontend.CatchExpr:
			if err := walkExpr(e.Call); err != nil {
				return err
			}
			for _, c := range e.Cases {
				if !c.Default {
					if err := walkExpr(c.Pattern); err != nil {
						return err
					}
				}
				if err := walkExpr(c.Guard); err != nil {
					return err
				}
				if err := walkExpr(c.Value); err != nil {
					return err
				}
			}
		case *frontend.MatchExpr:
			if err := walkExpr(e.Value); err != nil {
				return err
			}
			for _, c := range e.Cases {
				if !c.Default {
					if err := walkExpr(c.Pattern); err != nil {
						return err
					}
				}
				if err := walkExpr(c.Guard); err != nil {
					return err
				}
				if err := walkExpr(c.Value); err != nil {
					return err
				}
			}
		case *frontend.IdentExpr, *frontend.NumberExpr, *frontend.BoolLitExpr, *frontend.StringLitExpr:
			return nil
		default:
			return nil
		}
		return nil
	}

	walkStmt = func(stmt frontend.Stmt) error {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			return walkExpr(s.Value)
		case *frontend.ReturnStmt:
			return walkExpr(s.Value)
		case *frontend.ThrowStmt:
			return walkExpr(s.Value)
		case *frontend.DeferStmt:
			for _, inner := range s.Body {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
			return nil
		case *frontend.BreakStmt, *frontend.ContinueStmt:
			return nil
		case *frontend.LetStmt:
			return walkExpr(s.Value)
		case *frontend.AssignStmt:
			if err := walkExpr(s.Target); err != nil {
				return err
			}
			return walkExpr(s.Value)
		case *frontend.IfStmt:
			if err := walkExpr(s.Cond); err != nil {
				return err
			}
			for _, inner := range s.Then {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
			for _, inner := range s.Else {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
		case *frontend.IfLetStmt:
			if err := walkExpr(s.Value); err != nil {
				return err
			}
			if s.Pattern != nil {
				if err := walkExpr(s.Pattern); err != nil {
					return err
				}
			}
			for _, inner := range s.Then {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
			for _, inner := range s.Else {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
		case *frontend.WhileStmt:
			if err := walkExpr(s.Cond); err != nil {
				return err
			}
			for _, inner := range s.Body {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				if err := walkExpr(s.Iterable); err != nil {
					return err
				}
			} else {
				if err := walkExpr(s.Start); err != nil {
					return err
				}
				if err := walkExpr(s.End); err != nil {
					return err
				}
			}
			for _, inner := range s.Body {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
		case *frontend.MatchStmt:
			if err := walkExpr(s.Value); err != nil {
				return err
			}
			for _, c := range s.Cases {
				if !c.Default {
					if err := walkExpr(c.Pattern); err != nil {
						return err
					}
				}
				for _, inner := range c.Body {
					if err := walkStmt(inner); err != nil {
						return err
					}
				}
			}
		case *frontend.FreeStmt:
			return walkExpr(s.Value)
		case *frontend.UnsafeStmt:
			for _, inner := range s.Body {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
		case *frontend.IslandStmt:
			if err := walkExpr(s.Size); err != nil {
				return err
			}
			for _, inner := range s.Body {
				if err := walkStmt(inner); err != nil {
					return err
				}
			}
		default:
			return nil
		}
		return nil
	}

	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			if err := walkStmt(stmt); err != nil {
				return false, nil, 0, err
			}
		}
	}
	if !used {
		return false, nil, 0, nil
	}

	names := make([]string, 0, len(targets))
	for name := range targets {
		if name == checked.MainName {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	entries := append([]string{checked.MainName}, names...)
	return true, entries, spawnCount, nil
}

func collectActorStateRuntimeUsage(checked *semantics.CheckedProgram) bool {
	used, _ := collectActorStateRuntimeUsagePosition(checked)
	return used
}

func collectActorStateRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	if checked == nil {
		return false, frontend.Position{}
	}
	for _, fn := range checked.Funcs {
		if len(fn.ActorState) > 0 {
			if fn.Decl != nil {
				return true, fn.Decl.Pos
			}
			return true, frontend.Position{}
		}
	}
	return false, frontend.Position{}
}

func collectActorRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	if checked == nil {
		return false, frontend.Position{}
	}
	var used bool
	var first frontend.Position
	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)

	mark := func(pos frontend.Position) {
		if !used {
			used = true
			first = pos
		}
	}

	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			switch name {
			case "core.spawn",
				"core.send", "core.send_msg", "core.send_typed",
				"core.recv", "core.recv_msg", "core.recv_poll", "core.recv_until", "core.recv_msg_until", "core.recv_typed",
				"core.self", "core.sender", "core.yield":
				mark(e.At)
			}
			for _, arg := range e.Args {
				walkExpr(arg)
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				walkExpr(field.Value)
			}
		case *frontend.FieldAccessExpr:
			walkExpr(e.Base)
		case *frontend.IndexExpr:
			walkExpr(e.Base)
			walkExpr(e.Index)
		case *frontend.BinaryExpr:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *frontend.UnaryExpr:
			walkExpr(e.X)
		case *frontend.TryExpr:
			walkExpr(e.X)
		case *frontend.CatchExpr:
			walkExpr(e.Call)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		case *frontend.MatchExpr:
			walkExpr(e.Value)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		}
	}

	walkStmt = func(stmt frontend.Stmt) {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			walkExpr(s.Value)
		case *frontend.ReturnStmt:
			walkExpr(s.Value)
		case *frontend.ThrowStmt:
			walkExpr(s.Value)
		case *frontend.DeferStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.LetStmt:
			walkExpr(s.Value)
		case *frontend.AssignStmt:
			walkExpr(s.Target)
			walkExpr(s.Value)
		case *frontend.IfStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.IfLetStmt:
			walkExpr(s.Value)
			if s.Pattern != nil {
				walkExpr(s.Pattern)
			}
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.WhileStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				walkExpr(s.Iterable)
			} else {
				walkExpr(s.Start)
				walkExpr(s.End)
			}
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.MatchStmt:
			walkExpr(s.Value)
			for _, c := range s.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				for _, inner := range c.Body {
					walkStmt(inner)
				}
			}
		case *frontend.FreeStmt:
			walkExpr(s.Value)
		case *frontend.UnsafeStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.IslandStmt:
			walkExpr(s.Size)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		}
	}

	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			walkStmt(stmt)
		}
	}
	return used, first
}

func collectTaskRuntimeUsage(checked *semantics.CheckedProgram) bool {
	used, _ := collectTaskRuntimeUsagePosition(checked)
	return used
}

func collectTaskRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	if checked == nil {
		return false, frontend.Position{}
	}
	var used bool
	var first frontend.Position
	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)

	mark := func(pos frontend.Position) {
		if !used {
			used = true
			first = pos
		}
	}

	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			switch name {
			case "core.task_spawn_i32", "core.task_spawn_group_i32", "core.task_spawn_i32_typed", "core.task_spawn_group_i32_typed",
				"core.task_join_i32", "core.task_join_result_i32", "core.task_join_until_i32", "core.task_poll_i32", "core.select2_i32",
				"core.task_join_i32_typed", "core.task_join_group_i32_typed",
				"core.task_group_open", "core.task_group_close", "core.task_group_cancel", "core.task_group_current", "core.task_group_status",
				"core.task_is_canceled", "core.task_checkpoint":
				mark(e.At)
			}
			for _, arg := range e.Args {
				walkExpr(arg)
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				walkExpr(field.Value)
			}
		case *frontend.FieldAccessExpr:
			walkExpr(e.Base)
		case *frontend.IndexExpr:
			walkExpr(e.Base)
			walkExpr(e.Index)
		case *frontend.BinaryExpr:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *frontend.UnaryExpr:
			walkExpr(e.X)
		case *frontend.TryExpr:
			walkExpr(e.X)
		case *frontend.CatchExpr:
			walkExpr(e.Call)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		case *frontend.MatchExpr:
			walkExpr(e.Value)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		}
	}

	walkStmt = func(stmt frontend.Stmt) {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			walkExpr(s.Value)
		case *frontend.ReturnStmt:
			walkExpr(s.Value)
		case *frontend.ThrowStmt:
			walkExpr(s.Value)
		case *frontend.DeferStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.LetStmt:
			walkExpr(s.Value)
		case *frontend.AssignStmt:
			walkExpr(s.Target)
			walkExpr(s.Value)
		case *frontend.IfStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.IfLetStmt:
			walkExpr(s.Value)
			if s.Pattern != nil {
				walkExpr(s.Pattern)
			}
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.WhileStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				walkExpr(s.Iterable)
			} else {
				walkExpr(s.Start)
				walkExpr(s.End)
			}
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.MatchStmt:
			walkExpr(s.Value)
			for _, c := range s.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				for _, inner := range c.Body {
					walkStmt(inner)
				}
			}
		case *frontend.FreeStmt:
			walkExpr(s.Value)
		case *frontend.UnsafeStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.IslandStmt:
			walkExpr(s.Size)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		}
	}

	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			walkStmt(stmt)
		}
	}
	return used, first
}

func collectTaskGroupRuntimeUsage(checked *semantics.CheckedProgram) bool {
	if checked == nil {
		return false
	}
	var used bool
	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)

	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			switch name {
			case "core.task_group_open", "core.task_group_close", "core.task_group_cancel", "core.task_group_current", "core.task_group_status",
				"core.task_is_canceled", "core.task_checkpoint",
				"core.task_spawn_group_i32", "core.task_spawn_group_i32_typed":
				used = true
			}
			for _, arg := range e.Args {
				walkExpr(arg)
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				walkExpr(field.Value)
			}
		case *frontend.FieldAccessExpr:
			walkExpr(e.Base)
		case *frontend.IndexExpr:
			walkExpr(e.Base)
			walkExpr(e.Index)
		case *frontend.BinaryExpr:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *frontend.UnaryExpr:
			walkExpr(e.X)
		case *frontend.TryExpr:
			walkExpr(e.X)
		case *frontend.CatchExpr:
			walkExpr(e.Call)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		case *frontend.MatchExpr:
			walkExpr(e.Value)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		}
	}

	walkStmt = func(stmt frontend.Stmt) {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			walkExpr(s.Value)
		case *frontend.ReturnStmt:
			walkExpr(s.Value)
		case *frontend.ThrowStmt:
			walkExpr(s.Value)
		case *frontend.DeferStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.LetStmt:
			walkExpr(s.Value)
		case *frontend.AssignStmt:
			walkExpr(s.Target)
			walkExpr(s.Value)
		case *frontend.IfStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.IfLetStmt:
			walkExpr(s.Value)
			if s.Pattern != nil {
				walkExpr(s.Pattern)
			}
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.WhileStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				walkExpr(s.Iterable)
			} else {
				walkExpr(s.Start)
				walkExpr(s.End)
			}
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.MatchStmt:
			walkExpr(s.Value)
			for _, c := range s.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				for _, inner := range c.Body {
					walkStmt(inner)
				}
			}
		case *frontend.FreeStmt:
			walkExpr(s.Value)
		case *frontend.UnsafeStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.IslandStmt:
			walkExpr(s.Size)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		}
	}

	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			walkStmt(stmt)
		}
	}
	return used
}

func collectTypedTaskRuntimeUsage(checked *semantics.CheckedProgram) (bool, int) {
	if checked == nil {
		return false, 0
	}
	var used bool
	maxSlots := 0
	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)

	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			switch name {
			case "core.task_spawn_i32_typed", "core.task_spawn_group_i32_typed", "core.task_join_i32_typed", "core.task_join_group_i32_typed":
				used = true
				if len(e.TypeArgs) == 1 && e.TypeArgs[0].Name != "" {
					if _, handleInfo, err := semantics.EnsureTypedTaskHandleType(e.TypeArgs[0].Name, checked.Types); err == nil {
						if handleInfo.SlotCount > maxSlots {
							maxSlots = handleInfo.SlotCount
						}
					}
				}
			}
			for _, arg := range e.Args {
				walkExpr(arg)
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				walkExpr(field.Value)
			}
		case *frontend.FieldAccessExpr:
			walkExpr(e.Base)
		case *frontend.IndexExpr:
			walkExpr(e.Base)
			walkExpr(e.Index)
		case *frontend.BinaryExpr:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *frontend.UnaryExpr:
			walkExpr(e.X)
		case *frontend.TryExpr:
			walkExpr(e.X)
		case *frontend.CatchExpr:
			walkExpr(e.Call)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		case *frontend.MatchExpr:
			walkExpr(e.Value)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		}
	}

	walkStmt = func(stmt frontend.Stmt) {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			walkExpr(s.Value)
		case *frontend.ReturnStmt:
			walkExpr(s.Value)
		case *frontend.ThrowStmt:
			walkExpr(s.Value)
		case *frontend.DeferStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.LetStmt:
			walkExpr(s.Value)
		case *frontend.AssignStmt:
			walkExpr(s.Target)
			walkExpr(s.Value)
		case *frontend.IfStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.IfLetStmt:
			walkExpr(s.Value)
			if s.Pattern != nil {
				walkExpr(s.Pattern)
			}
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.WhileStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				walkExpr(s.Iterable)
			} else {
				walkExpr(s.Start)
				walkExpr(s.End)
			}
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.MatchStmt:
			walkExpr(s.Value)
			for _, c := range s.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				for _, inner := range c.Body {
					walkStmt(inner)
				}
			}
		case *frontend.FreeStmt:
			walkExpr(s.Value)
		case *frontend.UnsafeStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.IslandStmt:
			walkExpr(s.Size)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		}
	}

	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			walkStmt(stmt)
		}
	}
	if used && maxSlots < 4 {
		maxSlots = 4
	}
	return used, maxSlots
}

func collectDistributedActorRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	if checked == nil {
		return false, frontend.Position{}
	}
	var used bool
	var first frontend.Position
	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)

	mark := func(pos frontend.Position) {
		if !used {
			used = true
			first = pos
		}
	}

	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			switch name {
			case "core.actor_node_connect", "core.spawn_remote", "core.actor_node_status":
				mark(e.At)
			}
			for _, arg := range e.Args {
				walkExpr(arg)
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				walkExpr(field.Value)
			}
		case *frontend.FieldAccessExpr:
			walkExpr(e.Base)
		case *frontend.IndexExpr:
			walkExpr(e.Base)
			walkExpr(e.Index)
		case *frontend.BinaryExpr:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *frontend.UnaryExpr:
			walkExpr(e.X)
		case *frontend.TryExpr:
			walkExpr(e.X)
		case *frontend.CatchExpr:
			walkExpr(e.Call)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		case *frontend.MatchExpr:
			walkExpr(e.Value)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		}
	}

	walkStmt = func(stmt frontend.Stmt) {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			walkExpr(s.Value)
		case *frontend.ReturnStmt:
			walkExpr(s.Value)
		case *frontend.ThrowStmt:
			walkExpr(s.Value)
		case *frontend.DeferStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.LetStmt:
			walkExpr(s.Value)
		case *frontend.AssignStmt:
			walkExpr(s.Target)
			walkExpr(s.Value)
		case *frontend.IfStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.IfLetStmt:
			walkExpr(s.Value)
			if s.Pattern != nil {
				walkExpr(s.Pattern)
			}
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.WhileStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				walkExpr(s.Iterable)
			} else {
				walkExpr(s.Start)
				walkExpr(s.End)
			}
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.MatchStmt:
			walkExpr(s.Value)
			for _, c := range s.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				for _, inner := range c.Body {
					walkStmt(inner)
				}
			}
		case *frontend.FreeStmt:
			walkExpr(s.Value)
		case *frontend.UnsafeStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.IslandStmt:
			walkExpr(s.Size)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		}
	}

	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			walkStmt(stmt)
		}
	}
	return used, first
}

func collectTimeRuntimeUsage(checked *semantics.CheckedProgram) bool {
	used, _ := collectTimeRuntimeUsagePosition(checked)
	return used
}

func collectTimeRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	if checked == nil {
		return false, frontend.Position{}
	}
	var used bool
	var first frontend.Position
	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)

	mark := func(pos frontend.Position) {
		if !used {
			used = true
			first = pos
		}
	}

	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			switch name {
			case "core.time_now_ms", "core.sleep_ms", "core.sleep_until", "core.deadline_ms", "core.timer_ready":
				mark(e.At)
			}
			for _, arg := range e.Args {
				walkExpr(arg)
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				walkExpr(field.Value)
			}
		case *frontend.FieldAccessExpr:
			walkExpr(e.Base)
		case *frontend.IndexExpr:
			walkExpr(e.Base)
			walkExpr(e.Index)
		case *frontend.BinaryExpr:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *frontend.UnaryExpr:
			walkExpr(e.X)
		case *frontend.TryExpr:
			walkExpr(e.X)
		case *frontend.CatchExpr:
			walkExpr(e.Call)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		case *frontend.MatchExpr:
			walkExpr(e.Value)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		}
	}

	walkStmt = func(stmt frontend.Stmt) {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			walkExpr(s.Value)
		case *frontend.ReturnStmt:
			walkExpr(s.Value)
		case *frontend.ThrowStmt:
			walkExpr(s.Value)
		case *frontend.DeferStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.LetStmt:
			walkExpr(s.Value)
		case *frontend.AssignStmt:
			walkExpr(s.Target)
			walkExpr(s.Value)
		case *frontend.IfStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.IfLetStmt:
			walkExpr(s.Value)
			if s.Pattern != nil {
				walkExpr(s.Pattern)
			}
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.WhileStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				walkExpr(s.Iterable)
			} else {
				walkExpr(s.Start)
				walkExpr(s.End)
			}
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.MatchStmt:
			walkExpr(s.Value)
			for _, c := range s.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				for _, inner := range c.Body {
					walkStmt(inner)
				}
			}
		case *frontend.FreeStmt:
			walkExpr(s.Value)
		case *frontend.UnsafeStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.IslandStmt:
			walkExpr(s.Size)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		}
	}

	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			walkStmt(stmt)
		}
	}
	return used, first
}

func collectFilesystemRuntimeUsage(checked *semantics.CheckedProgram) bool {
	used, _ := collectFilesystemRuntimeUsagePosition(checked)
	return used
}

func collectFilesystemRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	if checked == nil {
		return false, frontend.Position{}
	}
	var used bool
	var pos frontend.Position
	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)

	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			if name == "core.fs_exists" {
				used = true
				if pos.Line == 0 && pos.Col == 0 {
					pos = e.At
				}
			}
			for _, arg := range e.Args {
				walkExpr(arg)
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				walkExpr(field.Value)
			}
		case *frontend.FieldAccessExpr:
			walkExpr(e.Base)
		case *frontend.IndexExpr:
			walkExpr(e.Base)
			walkExpr(e.Index)
		case *frontend.BinaryExpr:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *frontend.UnaryExpr:
			walkExpr(e.X)
		case *frontend.TryExpr:
			walkExpr(e.X)
		case *frontend.CatchExpr:
			walkExpr(e.Call)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		case *frontend.MatchExpr:
			walkExpr(e.Value)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		}
	}

	walkStmt = func(stmt frontend.Stmt) {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			walkExpr(s.Value)
		case *frontend.ReturnStmt:
			walkExpr(s.Value)
		case *frontend.ThrowStmt:
			walkExpr(s.Value)
		case *frontend.DeferStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.LetStmt:
			walkExpr(s.Value)
		case *frontend.AssignStmt:
			walkExpr(s.Target)
			walkExpr(s.Value)
		case *frontend.IfStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.IfLetStmt:
			walkExpr(s.Value)
			if s.Pattern != nil {
				walkExpr(s.Pattern)
			}
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.WhileStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				walkExpr(s.Iterable)
			} else {
				walkExpr(s.Start)
				walkExpr(s.End)
			}
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.MatchStmt:
			walkExpr(s.Value)
			for _, c := range s.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				for _, inner := range c.Body {
					walkStmt(inner)
				}
			}
		case *frontend.FreeStmt:
			walkExpr(s.Value)
		case *frontend.UnsafeStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.IslandStmt:
			walkExpr(s.Size)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		}
	}

	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			walkStmt(stmt)
		}
	}
	return used, pos
}

func collectNetRuntimeUsage(checked *semantics.CheckedProgram) bool {
	used, _ := collectNetRuntimeUsagePosition(checked)
	return used
}

func collectNetRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	if checked == nil {
		return false, frontend.Position{}
	}
	var used bool
	var pos frontend.Position
	var walkExpr func(frontend.Expr)
	var walkStmt func(frontend.Stmt)

	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.CallExpr:
			name := e.Name
			if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
				name = builtin
			}
			switch name {
			case "core.net_socket_tcp4", "core.net_bind_tcp4_loopback", "core.net_connect_tcp4_loopback", "core.net_listen", "core.net_accept4",
				"core.net_read", "core.net_recv", "core.net_write", "core.net_send", "core.net_epoll_create", "core.net_epoll_ctl_add_read",
				"core.net_epoll_ctl_add_read_write", "core.net_epoll_ctl_mod_read",
				"core.net_epoll_ctl_mod_read_write", "core.net_epoll_ctl_delete",
				"core.net_epoll_wait_one", "core.net_epoll_wait_one_into",
				"core.net_set_nonblocking", "core.net_set_reuseport",
				"core.net_set_tcp_nodelay", "core.net_close":
				used = true
				if pos.Line == 0 && pos.Col == 0 {
					pos = e.At
				}
			}
			for _, arg := range e.Args {
				walkExpr(arg)
			}
		case *frontend.StructLitExpr:
			for _, field := range e.Fields {
				walkExpr(field.Value)
			}
		case *frontend.FieldAccessExpr:
			walkExpr(e.Base)
		case *frontend.IndexExpr:
			walkExpr(e.Base)
			walkExpr(e.Index)
		case *frontend.BinaryExpr:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *frontend.UnaryExpr:
			walkExpr(e.X)
		case *frontend.TryExpr:
			walkExpr(e.X)
		case *frontend.CatchExpr:
			walkExpr(e.Call)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		case *frontend.MatchExpr:
			walkExpr(e.Value)
			for _, c := range e.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				walkExpr(c.Guard)
				walkExpr(c.Value)
			}
		}
	}

	walkStmt = func(stmt frontend.Stmt) {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			walkExpr(s.Value)
		case *frontend.ReturnStmt:
			walkExpr(s.Value)
		case *frontend.ThrowStmt:
			walkExpr(s.Value)
		case *frontend.DeferStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.LetStmt:
			walkExpr(s.Value)
		case *frontend.AssignStmt:
			walkExpr(s.Target)
			walkExpr(s.Value)
		case *frontend.IfStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.IfLetStmt:
			walkExpr(s.Value)
			if s.Pattern != nil {
				walkExpr(s.Pattern)
			}
			for _, inner := range s.Then {
				walkStmt(inner)
			}
			for _, inner := range s.Else {
				walkStmt(inner)
			}
		case *frontend.WhileStmt:
			walkExpr(s.Cond)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				walkExpr(s.Iterable)
			} else {
				walkExpr(s.Start)
				walkExpr(s.End)
			}
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.MatchStmt:
			walkExpr(s.Value)
			for _, c := range s.Cases {
				if !c.Default {
					walkExpr(c.Pattern)
				}
				for _, inner := range c.Body {
					walkStmt(inner)
				}
			}
		case *frontend.FreeStmt:
			walkExpr(s.Value)
		case *frontend.UnsafeStmt:
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		case *frontend.IslandStmt:
			walkExpr(s.Size)
			for _, inner := range s.Body {
				walkStmt(inner)
			}
		}
	}

	for _, fn := range checked.Funcs {
		if fn.Decl == nil {
			continue
		}
		for _, stmt := range fn.Decl.Body {
			walkStmt(stmt)
		}
	}
	return used, pos
}

func fnv1a32(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

func typedTaskRuntimeWrapperName(target, errorType string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(target))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(errorType))
	return fmt.Sprintf("__tetra_task_typed_%08x", h.Sum32())
}

func buildActorDispatchFunc(entries []string, checked *semantics.CheckedProgram) (IRFunc, error) {
	if len(entries) == 0 {
		return IRFunc{}, fmt.Errorf("missing actor entries")
	}
	seen := make(map[uint32]string, len(entries))
	for _, name := range entries {
		id := fnv1a32(name)
		if other, exists := seen[id]; exists && other != name {
			return IRFunc{}, fmt.Errorf("actor entry ID collision: %q and %q both hash to %d", other, name, id)
		}
		seen[id] = name
	}

	initByEntry := map[string][]semantics.ActorStateField{}
	if checked != nil {
		for _, fn := range checked.Funcs {
			if len(fn.ActorState) == 0 {
				continue
			}
			fields := make([]semantics.ActorStateField, 0, len(fn.ActorState))
			for _, field := range fn.ActorState {
				fields = append(fields, field)
			}
			sort.Slice(fields, func(i, j int) bool {
				return fields[i].Slot < fields[j].Slot
			})
			initByEntry[fn.Name] = fields
		}
	}

	var instrs []ir.IRInstr
	localSlots := 1
	if len(initByEntry) > 0 {
		localSlots = 2
	}
	nextLabel := 1
	for _, name := range entries {
		id := int32(fnv1a32(name))
		skipLabel := nextLabel
		nextLabel++

		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
			ir.IRInstr{Kind: ir.IRConstI32, Imm: id},
			ir.IRInstr{Kind: ir.IRCmpEqI32},
			ir.IRInstr{Kind: ir.IRJmpIfZero, Label: skipLabel},
		)
		if fields, ok := initByEntry[name]; ok {
			for _, field := range fields {
				instrs = append(instrs,
					ir.IRInstr{Kind: ir.IRConstI32, Imm: int32(field.Slot)},
					ir.IRInstr{Kind: ir.IRConstI32, Imm: field.Init},
					ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_state_store", ArgSlots: 2, RetSlots: 1},
					ir.IRInstr{Kind: ir.IRStoreLocal, Local: 1},
				)
			}
		}
		instrs = append(instrs,
			ir.IRInstr{Kind: ir.IRCall, Name: name, ArgSlots: 0, RetSlots: 1},
			ir.IRInstr{Kind: ir.IRReturn},
			ir.IRInstr{Kind: ir.IRLabel, Label: skipLabel},
		)
	}

	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 1},
		ir.IRInstr{Kind: ir.IRReturn},
	)

	return IRFunc{
		Name:        "__tetra_actor_dispatch",
		ParamSlots:  1,
		LocalSlots:  localSlots,
		ReturnSlots: 1,
		Instrs:      instrs,
	}, nil
}

func buildActorMainEntryIDFunc(mainName string) (IRFunc, error) {
	if mainName == "" {
		return IRFunc{}, fmt.Errorf("missing main name")
	}
	id := int32(fnv1a32(mainName))
	return IRFunc{
		Name:        "__tetra_actor_main_entry_id",
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: id},
			{Kind: ir.IRReturn},
		},
	}, nil
}
