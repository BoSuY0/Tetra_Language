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
	"tetra_language/compiler/internal/backend/linux_x64"
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
	modules         []string
	publicAPIHashes map[string]string
	buildTag        string
	objectsByModule map[string]*Object
	toCompile       []moduleBuildJob
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

	build, err := loadCheckedBuildWorld(inputPath, opt, !opt.InterfaceOnly)
	if err != nil {
		return nil, err
	}
	if opt.InterfaceOnly {
		return interfaceOnlyBuildStats(build.world), nil
	}
	linkedObjects, err := prepareLinkedObjects(build.world, opt.LinkObjectPaths, native.triple)
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
	if ctarget.IsBuildOnlyTarget(tgt.Triple) {
		return nativeBuildTarget{}, false, nil, fmt.Errorf("target backend not implemented: %s (codegen/link/run blocked)", tgt.Triple)
	}
	switch opt.Emit {
	case EmitExe:
		// continue
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
	return backend.codegen(nativeCodegenOptions(opt)), nil
}

func nativeExecutableBackendForTarget(tgt ctarget.Target) (nativeExecutableBackend, bool) {
	if tgt.Arch != ctarget.ArchX64 {
		return nativeExecutableBackend{}, false
	}
	backend, ok := nativeExecutableBackends()[tgt.OS]
	if !ok || backend.format != tgt.Format {
		return nativeExecutableBackend{}, false
	}
	return backend, true
}

func nativeCodegenOptions(opt BuildOptions) x64.CodegenOptions {
	return x64.CodegenOptions{
		IslandsDebug:    opt.IslandsDebug,
		DebugInfo:       opt.DebugInfo,
		ReleaseOptimize: opt.ReleaseOptimize,
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

func loadCheckedBuildWorld(inputPath string, opt BuildOptions, requireMain bool) (checkedBuildWorld, error) {
	world, err := loadWorldForBuild(inputPath, opt)
	if err != nil {
		return checkedBuildWorld{}, err
	}
	checked, err := semantics.CheckWorldOpt(world, semantics.CheckOptions{RequireMain: requireMain})
	if err != nil {
		return checkedBuildWorld{}, err
	}
	return checkedBuildWorld{world: world, checked: checked}, nil
}

func prepareLinkedObjects(world *World, paths []string, target string) ([]linkedObject, error) {
	linkedObjects, err := readLinkObjects(paths, target)
	if err != nil {
		return nil, err
	}
	if err := validateInterfaceImplementationProviders(world, linkedObjects); err != nil {
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
		modules:         modules,
		publicAPIHashes: publicAPIHashes,
		buildTag:        buildTag,
		objectsByModule: objectsByModule,
		toCompile:       toCompile,
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
			mu.Lock()
			stats.LoweredModules = append(stats.LoweredModules, job.module)
			mu.Unlock()

			dataPrefix := checked.GlobalDataByModule[job.module]
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
			return nil, fmt.Errorf("missing object for module '%s'", module)
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

func linkNativeExecutable(outputPath string, native nativeBuildTarget, opt BuildOptions, checked *semantics.CheckedProgram, objects []*Object, linkedObjects []linkedObject) error {
	actorsUsed, actorEntries, err := collectActorEntries(checked)
	if err != nil {
		return err
	}
	actorStateUsed := collectActorStateRuntimeUsage(checked)
	tasksUsed := collectTaskRuntimeUsage(checked)
	taskGroupsUsed := collectTaskGroupRuntimeUsage(checked)
	typedTasksUsed, typedTaskMaxSlots := collectTypedTaskRuntimeUsage(checked)
	typedTaskStagedUsed := typedTaskMaxSlots > 4
	timeRuntimeUsed := collectTimeRuntimeUsage(checked)
	runtimeUsed := actorsUsed || actorStateUsed || tasksUsed || taskGroupsUsed || typedTasksUsed || timeRuntimeUsed
	if runtimeUsed && len(actorEntries) == 0 {
		actorEntries = []string{checked.MainName}
	}
	mainName := checked.MainName
	if opt.RuntimeObjectPath != "" && !runtimeUsed {
		return fmt.Errorf("runtime object override requires runtime usage (no actor/task/time builtins found)")
	}
	if runtimeUsed {
		runtimeMode := opt.Runtime
		switch runtimeMode {
		case RuntimeAuto:
			// Default to self-host runtime when its ABI can express the program surface.
			runtimeMode = RuntimeSelfHost
			if actorStateUsed || tasksUsed || taskGroupsUsed || typedTasksUsed || timeRuntimeUsed {
				runtimeMode = RuntimeBuiltin
			}
			if typedTaskStagedUsed {
				runtimeMode = RuntimeBuiltin
			}
		case RuntimeSelfHost, RuntimeBuiltin:
			// ok
		default:
			return fmt.Errorf("unsupported runtime mode: %d", opt.Runtime)
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

func requiredActorRuntimeSymbols() []string {
	return []string{
		"__tetra_entry",
		"__tetra_actor_spawn",
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
		"__tetra_actor_send_slot",
		"__tetra_actor_send_commit",
		"__tetra_actor_recv",
		"__tetra_actor_recv_msg",
		"__tetra_actor_recv_poll",
		"__tetra_actor_recv_until",
		"__tetra_actor_recv_msg_until",
		"__tetra_actor_recv_begin",
		"__tetra_actor_recv_slot",
		"__tetra_actor_recv_count",
		"__tetra_actor_self",
		"__tetra_actor_sender",
		"__tetra_actor_yield_now",
	}
}

func requiredActorStateRuntimeSymbols() []string {
	return []string{
		"__tetra_actor_state_load",
		"__tetra_actor_state_store",
	}
}

func requiredTaskRuntimeSymbols() []string {
	return []string{
		"__tetra_task_spawn_i32",
		"__tetra_task_join_i32",
		"__tetra_task_join_result_i32",
		"__tetra_task_join_until_i32",
		"__tetra_task_poll_i32",
		"__tetra_task_is_canceled",
		"__tetra_task_checkpoint",
	}
}

func requiredTaskGroupRuntimeSymbols() []string {
	return []string{
		"__tetra_task_group_open",
		"__tetra_task_group_close",
		"__tetra_task_group_cancel",
		"__tetra_task_group_current",
		"__tetra_task_group_status",
		"__tetra_task_spawn_group_i32",
	}
}

func requiredTypedTaskRuntimeSymbols(maxSlots int) []string {
	if maxSlots < 2 {
		maxSlots = 2
	}
	if maxSlots > 8 {
		maxSlots = 8
	}
	symbols := []string{
		"__tetra_task_result_begin",
		"__tetra_task_result_slot",
	}
	if maxSlots > 4 {
		symbols = append(symbols, "__tetra_task_result_get")
	}
	for slots := 2; slots <= maxSlots; slots++ {
		symbols = append(symbols, fmt.Sprintf("__tetra_task_join_typed_%d", slots))
	}
	return symbols
}

func requiredTimeRuntimeSymbols() []string {
	return []string{
		"__tetra_time_now_ms",
		"__tetra_sleep_ms",
		"__tetra_sleep_until_ms",
		"__tetra_deadline_ms",
		"__tetra_timer_ready_ms",
	}
}

func validateActorRuntimeObject(rt *Object) error {
	if rt == nil {
		return fmt.Errorf("missing actors runtime object")
	}
	symbols := make(map[string]struct{}, len(rt.Symbols))
	for _, sym := range rt.Symbols {
		symbols[sym.Name] = struct{}{}
	}
	for _, name := range requiredActorRuntimeSymbols() {
		if _, ok := symbols[name]; !ok {
			return fmt.Errorf("runtime object missing required symbol '%s'", name)
		}
	}
	return nil
}

func validateActorStateRuntimeObject(rt *Object) error {
	if rt == nil {
		return fmt.Errorf("missing actors runtime object")
	}
	symbols := make(map[string]struct{}, len(rt.Symbols))
	for _, sym := range rt.Symbols {
		symbols[sym.Name] = struct{}{}
	}
	for _, name := range requiredActorStateRuntimeSymbols() {
		if _, ok := symbols[name]; !ok {
			return fmt.Errorf("runtime object missing required symbol '%s'", name)
		}
	}
	return nil
}

func validateTimeRuntimeObject(rt *Object) error {
	if rt == nil {
		return fmt.Errorf("missing time runtime object")
	}
	symbols := make(map[string]struct{}, len(rt.Symbols))
	for _, sym := range rt.Symbols {
		symbols[sym.Name] = struct{}{}
	}
	for _, name := range requiredTimeRuntimeSymbols() {
		if _, ok := symbols[name]; !ok {
			return fmt.Errorf("runtime object missing required symbol '%s'", name)
		}
	}
	return nil
}

func validateTypedTaskRuntimeObject(rt *Object, maxSlots int) error {
	if rt == nil {
		return fmt.Errorf("missing typed task runtime object")
	}
	symbols := make(map[string]struct{}, len(rt.Symbols))
	for _, sym := range rt.Symbols {
		symbols[sym.Name] = struct{}{}
	}
	for _, name := range requiredTypedTaskRuntimeSymbols(maxSlots) {
		if _, ok := symbols[name]; !ok {
			return fmt.Errorf("runtime object missing required symbol '%s'", name)
		}
	}
	return nil
}

func validateTaskRuntimeObject(rt *Object) error {
	if rt == nil {
		return fmt.Errorf("missing task runtime object")
	}
	symbols := make(map[string]struct{}, len(rt.Symbols))
	for _, sym := range rt.Symbols {
		symbols[sym.Name] = struct{}{}
	}
	for _, name := range requiredTaskRuntimeSymbols() {
		if _, ok := symbols[name]; !ok {
			return fmt.Errorf("runtime object missing required symbol '%s'", name)
		}
	}
	return nil
}

func validateTaskGroupRuntimeObject(rt *Object) error {
	if rt == nil {
		return fmt.Errorf("missing task group runtime object")
	}
	symbols := make(map[string]struct{}, len(rt.Symbols))
	for _, sym := range rt.Symbols {
		symbols[sym.Name] = struct{}{}
	}
	for _, name := range requiredTaskGroupRuntimeSymbols() {
		if _, ok := symbols[name]; !ok {
			return fmt.Errorf("runtime object missing required symbol '%s'", name)
		}
	}
	return nil
}

func buildObjectFileWithStatsOpt(inputPath, outputPath string, tgt ctarget.Target, opt BuildOptions) (*BuildStats, error) {
	requireMain := opt.Emit == EmitObject && !opt.InterfaceOnly
	codegenOptions := x64.CodegenOptions{
		IslandsDebug:    opt.IslandsDebug,
		DebugInfo:       opt.DebugInfo,
		ReleaseOptimize: opt.ReleaseOptimize,
	}

	world, err := loadWorldForBuild(inputPath, opt)
	if err != nil {
		return nil, err
	}
	checked, err := semantics.CheckWorldOpt(world, semantics.CheckOptions{RequireMain: requireMain})
	if err != nil {
		return nil, err
	}
	if opt.InterfaceOnly {
		return interfaceOnlyBuildStats(world), nil
	}
	if err := rejectInterfaceModulesForCodegen(world); err != nil {
		return nil, err
	}

	funcs, err := LowerModule(checked, world.EntryModule)
	if err != nil {
		return nil, err
	}

	var obj *Object
	dataPrefix := checked.GlobalDataByModule[world.EntryModule]
	switch tgt.OS {
	case ctarget.OSLinux:
		obj, err = linux_x64.CodegenObjectLinuxX64WithOptionsAndDataPrefix(funcs, dataPrefix, codegenOptions)
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
		funcs = append(funcs, moduleFuncs...)
	}

	obj, err := wasm32_wasi.CodegenObject(funcs, checked.MainName)
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
		funcs = append(funcs, moduleFuncs...)
	}

	obj, err := wasm32_web.CodegenObject(funcs, checked.MainName)
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

func validateInterfaceImplementationProviders(world *World, linked []linkedObject) error {
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
		if err := validateInterfaceImplementationSymbols(world, obj.Module, obj, linked.path); err != nil {
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

func validateInterfaceImplementationSymbols(world *World, module string, obj *Object, path string) error {
	symbols := make(map[string]struct{}, len(obj.Symbols))
	for _, sym := range obj.Symbols {
		symbols[sym.Name] = struct{}{}
	}
	for _, name := range expectedInterfaceModuleSymbols(world, module) {
		if _, ok := symbols[name]; !ok {
			return fmt.Errorf("implementation object for interface module '%s' missing exported symbol '%s' (%s)", module, name, path)
		}
	}
	return nil
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
		if fn == nil || fn.Synthetic {
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

func collectActorEntries(checked *semantics.CheckedProgram) (bool, []string, error) {
	if checked == nil {
		return false, nil, nil
	}
	used := false
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
				if len(e.Args) == 1 {
					if lit, ok := e.Args[0].(*frontend.StringLitExpr); ok {
						name := string(lit.Value)
						if name != "" {
							targets[name] = struct{}{}
						}
					}
				}
			case "core.task_spawn_i32":
				used = true
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
			return walkExpr(e.Call)
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
				return false, nil, err
			}
		}
	}
	if !used {
		return false, nil, nil
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
	return true, entries, nil
}

func collectActorStateRuntimeUsage(checked *semantics.CheckedProgram) bool {
	if checked == nil {
		return false
	}
	for _, fn := range checked.Funcs {
		if len(fn.ActorState) > 0 {
			return true
		}
	}
	return false
}

func collectTaskRuntimeUsage(checked *semantics.CheckedProgram) bool {
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
			case "core.task_spawn_i32", "core.task_spawn_group_i32", "core.task_spawn_i32_typed", "core.task_spawn_group_i32_typed",
				"core.task_join_i32", "core.task_join_result_i32", "core.task_join_until_i32", "core.task_poll_i32", "core.select2_i32",
				"core.task_join_i32_typed", "core.task_join_group_i32_typed",
				"core.task_group_open", "core.task_group_close", "core.task_group_cancel", "core.task_group_current", "core.task_group_status",
				"core.task_is_canceled", "core.task_checkpoint":
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

func collectTimeRuntimeUsage(checked *semantics.CheckedProgram) bool {
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
			case "core.time_now_ms", "core.sleep_ms", "core.sleep_until", "core.deadline_ms", "core.timer_ready":
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
					ir.IRInstr{Kind: ir.IRCall, Name: "__tetra_actor_state_store", ArgSlots: 2, RetSlots: 0},
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
		LocalSlots:  1,
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
