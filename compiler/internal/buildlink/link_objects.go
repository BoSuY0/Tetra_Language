package buildlink

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/internal/version"
)

type LinkedObject struct {
	Path        string
	Object      *tobj.Object
	ContentHash [32]byte
}

func ReadLinkObjects(paths []string, target string) ([]LinkedObject, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	var linked []LinkedObject
	seenPaths := make(map[string]string, len(paths))
	seenSymbols := make(map[string]LinkedObject)
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
		obj, err := tobj.ReadObject(path)
		if err != nil {
			return nil, fmt.Errorf("read link object %s: %w", path, err)
		}
		if obj.Target == "" {
			return nil, fmt.Errorf("link object has no target: %s", path)
		}
		if obj.Target != target {
			return nil, fmt.Errorf(
				"link object target mismatch: got=%s want=%s (%s)",
				obj.Target,
				target,
				path,
			)
		}
		if obj.Module == "" {
			return nil, fmt.Errorf("link object has no module identity: %s", path)
		}
		if obj.CompilerVersion != "" && obj.CompilerVersion != version.CompilerVersion {
			return nil, fmt.Errorf(
				"link object compiler version mismatch: got=%s want=%s (%s)",
				obj.CompilerVersion,
				version.CompilerVersion,
				path,
			)
		}
		current := LinkedObject{Path: path, Object: obj, ContentHash: sha256.Sum256(raw)}
		if err := ValidateLinkedObjectSymbols(current, seenSymbols); err != nil {
			return nil, err
		}
		linked = append(linked, current)
	}
	return linked, nil
}

func ValidateLinkedObjectSymbols(current LinkedObject, seen map[string]LinkedObject) error {
	if current.Object == nil {
		return nil
	}
	local := make(map[string]struct{}, len(current.Object.Symbols))
	for _, sym := range current.Object.Symbols {
		if sym.Name == "" {
			return fmt.Errorf("link object has empty symbol name: %s", current.Path)
		}
		if _, exists := local[sym.Name]; exists {
			return fmt.Errorf("duplicate symbol '%s' inside link object %s", sym.Name, current.Path)
		}
		local[sym.Name] = struct{}{}
		if first, exists := seen[sym.Name]; exists {
			return fmt.Errorf(
				"duplicate symbol '%s' in link objects: %s and %s",
				sym.Name,
				first.Path,
				current.Path,
			)
		}
		seen[sym.Name] = current
	}
	return nil
}

func ValidateInterfaceImplementationProviders(
	world *module.World,
	checked *semantics.CheckedProgram,
	linked []LinkedObject,
) error {
	modules := SortedInterfaceModules(world)
	if len(modules) == 0 {
		return nil
	}
	providers := make(map[string]LinkedObject, len(modules))
	interfaceSet := make(map[string]struct{}, len(modules))
	for _, moduleName := range modules {
		interfaceSet[moduleName] = struct{}{}
	}
	for _, linked := range linked {
		obj := linked.Object
		if obj == nil {
			continue
		}
		if _, ok := interfaceSet[obj.Module]; !ok {
			continue
		}
		if first, exists := providers[obj.Module]; exists {
			return fmt.Errorf(
				"duplicate implementation object for interface module '%s': %s and %s",
				obj.Module,
				first.Path,
				linked.Path,
			)
		}
		if obj.PublicAPIHash == "" {
			return fmt.Errorf(
				"implementation object for interface module '%s' has no public API hash: %s",
				obj.Module,
				linked.Path,
			)
		}
		want := world.InterfaceHashes[obj.Module]
		if want == "" {
			return fmt.Errorf("missing interface hash for module '%s'", obj.Module)
		}
		if obj.PublicAPIHash != want {
			return fmt.Errorf(
				"public API hash mismatch for interface module '%s': object %s, interface %s (%s)",
				obj.Module,
				obj.PublicAPIHash,
				want,
				linked.Path,
			)
		}
		if err := ValidateInterfaceImplementationSymbols(
			world,
			checked,
			obj.Module,
			obj,
			linked.Path,
		); err != nil {
			return err
		}
		providers[obj.Module] = linked
	}
	for _, moduleName := range modules {
		if _, ok := providers[moduleName]; !ok {
			return fmt.Errorf(
				("missing implementation object for interface module '%s'; pass --" +
					"link-object with a matching TOBJ"),
				moduleName,
			)
		}
	}
	return nil
}

func ValidateInterfaceImplementationSymbols(
	world *module.World,
	checked *semantics.CheckedProgram,
	moduleName string,
	obj *tobj.Object,
	path string,
) error {
	symbols := make(map[string]tobj.Symbol, len(obj.Symbols))
	for _, sym := range obj.Symbols {
		symbols[sym.Name] = sym
	}
	for _, name := range UnsupportedInterfaceModuleGenericSymbols(world, moduleName) {
		return fmt.Errorf(
			("implementation object for interface module '%s' cannot satisfy " +
				"generic export '%s'; precompiled link objects require monomorphic " +
				"exported functions (%s)"),
			moduleName,
			name,
			path,
		)
	}
	for _, name := range ExpectedInterfaceModuleSymbols(world, moduleName) {
		sym, ok := symbols[name]
		if !ok {
			return fmt.Errorf(
				"implementation object for interface module '%s' missing exported symbol '%s' (%s)",
				moduleName,
				name,
				path,
			)
		}
		if !sym.HasSignature {
			return fmt.Errorf(
				"implementation object for interface module '%s' symbol '%s' missing signature metadata (%s)",
				moduleName,
				name,
				path,
			)
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
				("implementation object for interface module '%s' symbol '%s' " +
					"signature mismatch: params=%d want=%d returns=%d want=%d (%s)"),
				moduleName,
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

func UnsupportedInterfaceModuleGenericSymbols(world *module.World, moduleName string) []string {
	if world == nil || world.ByModule == nil {
		return nil
	}
	file := world.ByModule[moduleName]
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
			name = QualifyObjectSymbol(moduleName, fn.Name)
		}
		symbols = append(symbols, name)
	}
	sort.Strings(symbols)
	return symbols
}

func ExpectedInterfaceModuleSymbols(world *module.World, moduleName string) []string {
	if world == nil || world.ByModule == nil {
		return nil
	}
	file := world.ByModule[moduleName]
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
			name = QualifyObjectSymbol(moduleName, fn.Name)
		}
		symbols = append(symbols, name)
	}
	sort.Strings(symbols)
	return symbols
}

func QualifyObjectSymbol(moduleName, name string) string {
	if moduleName == "" || strings.HasPrefix(name, moduleName+".") {
		return name
	}
	return moduleName + "." + name
}

func SortedInterfaceModules(world *module.World) []string {
	if world == nil || len(world.InterfaceModules) == 0 {
		return nil
	}
	modules := make([]string, 0, len(world.InterfaceModules))
	for moduleName := range world.InterfaceModules {
		modules = append(modules, moduleName)
	}
	sort.Strings(modules)
	return modules
}
