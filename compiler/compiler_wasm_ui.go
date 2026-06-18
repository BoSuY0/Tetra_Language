package compiler

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"tetra_language/compiler/internal/backend/linux_x32"
	"tetra_language/compiler/internal/backend/linux_x64"
	"tetra_language/compiler/internal/backend/linux_x86"
	"tetra_language/compiler/internal/backend/macos_x64"
	"tetra_language/compiler/internal/backend/wasm32_wasi"
	"tetra_language/compiler/internal/backend/wasm32_web"
	"tetra_language/compiler/internal/backend/windows_x64"
	"tetra_language/compiler/internal/buildwasm"
	"tetra_language/compiler/internal/cache"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/internal/version"
	ctarget "tetra_language/compiler/target"
)

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
	checked, err := semantics.CheckWorldOpt(world, semanticsCheckOptionsForTarget(requireMain, tgt.Triple))
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
	if err := emitExplainReports(outputPath, tgt.Triple, checked, opt); err != nil {
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
	if err := emitExplainReports(outputPath, tgt.Triple, checked, opt); err != nil {
		return nil, err
	}
	return stats, nil
}

func wasmWebLoaderPath(outputPath string) string {
	return buildwasm.WebLoaderPath(outputPath)
}

func relocateWASMGlobalSlots(funcs []IRFunc, offset int) []IRFunc {
	return buildwasm.RelocateGlobalSlots(funcs, offset)
}

func rejectUnsupportedWASMRuntimeBuiltins(funcs []IRFunc, target string) error {
	pos, runtimeName, unsupported := buildwasm.FirstUnsupportedRuntimeBuiltin(funcs, target)
	if !unsupported {
		return nil
	}
	return targetRuntimeDiagnostic(pos, target, runtimeName)
}

func wasmRuntimeNameForBuiltin(name string, target string) (string, bool) {
	return buildwasm.RuntimeNameForBuiltin(name, target)
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
	pos, policy, blocked := buildwasm.FirstBlockedIRPolicy(target, funcs)
	if !blocked {
		return nil
	}
	return targetWASMPolicyDiagnostic(pos, target, rootWASMIRPolicy(policy))
}

func blockedWASMIRPolicy(kind ir.IRInstrKind) (wasmIRPolicy, bool) {
	policy, blocked := buildwasm.BlockedIRPolicy(kind)
	if !blocked {
		return wasmIRPolicy{}, false
	}
	return rootWASMIRPolicy(policy), true
}

func rootWASMIRPolicy(policy buildwasm.IRPolicy) wasmIRPolicy {
	return wasmIRPolicy{builtin: policy.Builtin, category: policy.Category}
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
	return buildwasm.EmitUIArtifacts(outputPath, target, checked)
}

func uiArtifactBasePath(outputPath string) string {
	return buildwasm.UIArtifactBasePath(outputPath)
}
