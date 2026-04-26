package compiler

import (
	"crypto/sha256"
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
	"tetra_language/compiler/internal/backend/wasm32_wasi"
	"tetra_language/compiler/internal/backend/wasm32_web"
	"tetra_language/compiler/internal/backend/windows_x64"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/cache"
	"tetra_language/compiler/internal/deps"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
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
	Emit              EmitMode
	Runtime           RuntimeMode
	RuntimeObjectPath string
	LinkObjectPaths   []string
}

type BuildStats struct {
	CompiledModules []string
	CacheHits       []string
	LoweredModules  []string
}

func BuildFileWithStats(inputPath, outputPath, target string) (*BuildStats, error) {
	return BuildFileWithStatsOpt(inputPath, outputPath, target, BuildOptions{Jobs: 1})
}

func BuildFileWithStatsOpt(inputPath, outputPath, target string, opt BuildOptions) (*BuildStats, error) {
	tgt, err := ctarget.Parse(target)
	if err != nil {
		return nil, err
	}
	target = tgt.Triple
	if target == "wasm32-wasi" {
		return buildWASM32WASIWithStatsOpt(inputPath, outputPath, tgt, opt)
	}
	if target == "wasm32-web" {
		return buildWASM32WEBWithStatsOpt(inputPath, outputPath, tgt, opt)
	}
	if ctarget.IsBuildOnlyTarget(target) {
		return nil, fmt.Errorf("target backend not implemented: %s (codegen/link/run blocked)", target)
	}

	switch opt.Emit {
	case EmitExe:
		// continue
	case EmitObject, EmitLibrary:
		return buildObjectFileWithStatsOpt(inputPath, outputPath, tgt, opt)
	default:
		return nil, fmt.Errorf("unsupported emit mode: %d", opt.Emit)
	}

	codegenOptions := x64.CodegenOptions{IslandsDebug: opt.IslandsDebug}
	var codegen func([]IRFunc, [][]byte) (*Object, error)
	switch tgt.OS {
	case ctarget.OSLinux:
		codegen = func(funcs []IRFunc, dataPrefix [][]byte) (*Object, error) {
			return linux_x64.CodegenObjectLinuxX64WithOptionsAndDataPrefix(funcs, dataPrefix, codegenOptions)
		}
	case ctarget.OSWindows:
		codegen = func(funcs []IRFunc, dataPrefix [][]byte) (*Object, error) {
			return windows_x64.CodegenObjectWindowsX64WithOptionsAndDataPrefix(funcs, dataPrefix, codegenOptions)
		}
	case ctarget.OSMacOS:
		codegen = func(funcs []IRFunc, dataPrefix [][]byte) (*Object, error) {
			return macos_x64.CodegenObjectMacOSX64WithOptionsAndDataPrefix(funcs, dataPrefix, codegenOptions)
		}
	default:
		return nil, fmt.Errorf("unsupported target: %s", target)
	}

	world, err := LoadWorld(inputPath)
	if err != nil {
		return nil, err
	}

	checked, err := CheckWorld(world)
	if err != nil {
		return nil, err
	}

	sigMap := cache.BuildSigMap(checked)
	depsByModule := deps.CollectExternalCalleesByModule(checked)
	typeDepsByModule := deps.CollectExternalTypesByModule(checked)
	typeSigMap, err := cache.BuildTypeSigMap(checked.Types)
	if err != nil {
		return nil, err
	}
	stats := &BuildStats{}

	modules := make([]string, 0, len(world.ByModule))
	for module := range world.ByModule {
		modules = append(modules, module)
	}
	sort.Strings(modules)

	type moduleJob struct {
		module  string
		srcHash [32]byte
		depHash [32]byte
	}

	buildTag := buildTagFromOptions(opt)
	objectsByModule := make(map[string]*Object, len(modules))
	var toCompile []moduleJob

	for _, module := range modules {
		file := world.ByModule[module]
		if file == nil {
			return nil, fmt.Errorf("missing module '%s'", module)
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
		depHash, err := cache.DepSigHashFromDeps(callees, typeDeps, sigMap, typeSigMap)
		if err != nil {
			return nil, err
		}
		obj, hit, err := cache.LoadCachedObject(world.Root, target, buildTag, module, srcHash, depHash)
		if err != nil {
			return nil, err
		}
		if hit {
			stats.CacheHits = append(stats.CacheHits, module)
			objectsByModule[module] = obj
			continue
		}
		toCompile = append(toCompile, moduleJob{module: module, srcHash: srcHash, depHash: depHash})
	}

	if len(toCompile) > 0 {
		jobs := opt.Jobs
		if jobs <= 0 {
			jobs = runtime.NumCPU()
		}
		if jobs < 1 {
			jobs = 1
		}
		if jobs > len(toCompile) {
			jobs = len(toCompile)
		}

		jobsCh := make(chan moduleJob)
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
				obj, err := codegen(funcs, dataPrefix)
				if err != nil {
					setErr(err)
					continue
				}
				obj.Target = target
				obj.Module = job.module
				obj.SrcHash = job.srcHash
				obj.WorldSigHash = job.depHash
				if err := cache.StoreCachedObject(world.Root, target, buildTag, obj); err != nil {
					setErr(err)
					continue
				}
				mu.Lock()
				stats.CompiledModules = append(stats.CompiledModules, job.module)
				objectsByModule[job.module] = obj
				mu.Unlock()
			}
		}

		wg.Add(jobs)
		for i := 0; i < jobs; i++ {
			go worker()
		}
		for _, job := range toCompile {
			jobsCh <- job
		}
		close(jobsCh)
		wg.Wait()
		if err := getErr(); err != nil {
			return nil, err
		}
	}

	sort.Strings(stats.CacheHits)
	sort.Strings(stats.CompiledModules)
	sort.Strings(stats.LoweredModules)

	objects := make([]*Object, 0, len(modules))
	for _, module := range modules {
		obj := objectsByModule[module]
		if obj == nil {
			return nil, fmt.Errorf("missing object for module '%s'", module)
		}
		objects = append(objects, obj)
	}

	actorsUsed, actorEntries, err := collectActorEntries(checked)
	if err != nil {
		return nil, err
	}
	mainName := checked.MainName
	if opt.RuntimeObjectPath != "" && !actorsUsed {
		return nil, fmt.Errorf("runtime object override requires actors usage (no actor builtins found)")
	}
	if actorsUsed {
		runtimeMode := opt.Runtime
		switch runtimeMode {
		case RuntimeAuto:
			// Default to self-host runtime.
			runtimeMode = RuntimeSelfHost
		case RuntimeSelfHost, RuntimeBuiltin:
			// ok
		default:
			return nil, fmt.Errorf("unsupported runtime mode: %d", opt.Runtime)
		}

		var rt *Object
		needsDispatchGlue := true
		needsMainEntryIDGlue := true
		if opt.RuntimeObjectPath != "" {
			rt, err = ReadObject(opt.RuntimeObjectPath)
			if err != nil {
				return nil, fmt.Errorf("read runtime object: %w", err)
			}
			if rt.Target == "" {
				return nil, fmt.Errorf("runtime object has no target: %s", opt.RuntimeObjectPath)
			}
			if rt.Target != target {
				return nil, fmt.Errorf("runtime object target mismatch: got=%s want=%s", rt.Target, target)
			}
		} else {
			switch runtimeMode {
			case RuntimeSelfHost:
				rt, err = buildEmbeddedSelfHostActorsRuntimeObject(target, codegen)
			case RuntimeBuiltin:
				switch tgt.OS {
				case ctarget.OSLinux:
					rt, err = actorsrt.BuildLinuxX64(actorEntries)
				case ctarget.OSMacOS:
					rt, err = actorsrt.BuildMacOSX64(actorEntries)
				case ctarget.OSWindows:
					rt, err = actorsrt.BuildWindowsX64(actorEntries)
				default:
					return nil, fmt.Errorf("actors runtime is not supported on target %s", target)
				}
			}
			if err != nil {
				return nil, err
			}
		}
		if err := validateActorRuntimeObject(rt); err != nil {
			return nil, err
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
				dispatchFn, err := buildActorDispatchFunc(actorEntries)
				if err != nil {
					return nil, err
				}
				glueFuncs = append(glueFuncs, dispatchFn)
			}
			if needsMainEntryIDGlue {
				mainIDFn, err := buildActorMainEntryIDFunc(actorEntries[0])
				if err != nil {
					return nil, err
				}
				glueFuncs = append(glueFuncs, mainIDFn)
			}
			glueObj, err := codegen(glueFuncs, nil)
			if err != nil {
				return nil, err
			}
			glueObj.Target = target
			glueObj.Module = "__actorsglue"
			objects = append(objects, glueObj)
		}
		rt.Target = target
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

	if len(opt.LinkObjectPaths) > 0 {
		for _, path := range opt.LinkObjectPaths {
			if path == "" {
				continue
			}
			obj, err := ReadObject(path)
			if err != nil {
				return nil, fmt.Errorf("read link object: %w", err)
			}
			if obj.Target == "" {
				return nil, fmt.Errorf("link object has no target: %s", path)
			}
			if obj.Target != target {
				return nil, fmt.Errorf("link object target mismatch: got=%s want=%s (%s)", obj.Target, target, path)
			}
			objects = append(objects, obj)
		}
	}

	switch tgt.Format {
	case ctarget.FormatELF:
		img, err := LinkLinuxX64(objects, mainName)
		if err != nil {
			return nil, err
		}
		if err := WriteELF64LinuxX64(outputPath, img); err != nil {
			return nil, err
		}
	case ctarget.FormatPE:
		img, err := LinkWindowsX64(objects, mainName)
		if err != nil {
			return nil, err
		}
		if err := WritePE64WindowsX64(outputPath, img); err != nil {
			return nil, err
		}
	case ctarget.FormatMachO:
		img, err := LinkMacOSX64(objects, mainName)
		if err != nil {
			return nil, err
		}
		if err := WriteMachO64MacOSX64(outputPath, img); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported target format: %s", tgt.Format)
	}

	return stats, nil
}

func requiredActorRuntimeSymbols() []string {
	return []string{
		"__tetra_entry",
		"__tetra_actor_spawn",
		"__tetra_actor_send",
		"__tetra_actor_recv",
		"__tetra_actor_self",
		"__tetra_actor_sender",
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

func buildObjectFileWithStatsOpt(inputPath, outputPath string, tgt ctarget.Target, opt BuildOptions) (*BuildStats, error) {
	requireMain := opt.Emit == EmitObject
	codegenOptions := x64.CodegenOptions{IslandsDebug: opt.IslandsDebug}

	world, err := LoadWorld(inputPath)
	if err != nil {
		return nil, err
	}
	checked, err := semantics.CheckWorldOpt(world, semantics.CheckOptions{RequireMain: requireMain})
	if err != nil {
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

	world, err := LoadWorld(inputPath)
	if err != nil {
		return nil, err
	}
	checked, err := CheckWorld(world)
	if err != nil {
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

	world, err := LoadWorld(inputPath)
	if err != nil {
		return nil, err
	}
	checked, err := CheckWorld(world)
	if err != nil {
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
	return stats, nil
}

func wasmWebLoaderPath(outputPath string) string {
	ext := filepath.Ext(outputPath)
	if strings.EqualFold(ext, ".wasm") {
		return strings.TrimSuffix(outputPath, ext) + ".mjs"
	}
	return outputPath + ".mjs"
}

func buildTagFromOptions(opt BuildOptions) string {
	if opt.IslandsDebug {
		return "islands-debug"
	}
	return ""
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
			switch e.Name {
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
			case "core.send", "core.recv", "core.self", "core.sender":
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

func fnv1a32(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}

func buildActorDispatchFunc(entries []string) (IRFunc, error) {
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
