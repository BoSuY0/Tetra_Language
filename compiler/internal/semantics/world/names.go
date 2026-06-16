package world

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

const ImportSymbolPrefix = "\x00symbol:"

func CollectImportAliases(file *frontend.FileAST) (map[string]string, error) {
	aliases := make(map[string]string)
	topLevel := TopLevelDeclarationNames(file)
	for _, imp := range file.Imports {
		if imp.Path == "" {
			return nil, fmt.Errorf("%s: import path required", frontend.FormatPos(imp.At))
		}
		if len(imp.Items) > 0 {
			for _, item := range imp.Items {
				if item == "" {
					return nil, fmt.Errorf("%s: empty selective import", frontend.FormatPos(imp.At))
				}
				if _, exists := aliases[item]; exists {
					return nil, fmt.Errorf("%s: duplicate import alias '%s'", frontend.FormatPos(imp.At), item)
				}
				if _, exists := topLevel[item]; exists {
					return nil, fmt.Errorf("%s: import alias '%s' conflicts with declaration '%s'", frontend.FormatPos(imp.At), item, item)
				}
				aliases[item] = ImportSymbolPrefix + imp.Path + "." + item
			}
			continue
		}
		if imp.Alias == "" {
			return nil, fmt.Errorf("%s: import alias required", frontend.FormatPos(imp.At))
		}
		if _, exists := aliases[imp.Alias]; exists {
			return nil, fmt.Errorf("%s: duplicate import alias '%s'", frontend.FormatPos(imp.At), imp.Alias)
		}
		if _, exists := topLevel[imp.Alias]; exists {
			return nil, fmt.Errorf("%s: import alias '%s' conflicts with declaration '%s'", frontend.FormatPos(imp.At), imp.Alias, imp.Alias)
		}
		aliases[imp.Alias] = imp.Path
	}
	return aliases, nil
}

func ImportSymbolTarget(target string) (string, bool) {
	if !strings.HasPrefix(target, ImportSymbolPrefix) {
		return "", false
	}
	return strings.TrimPrefix(target, ImportSymbolPrefix), true
}

func TopLevelDeclarationNames(file *frontend.FileAST) map[string]struct{} {
	names := map[string]struct{}{}
	if file == nil {
		return names
	}
	for _, fn := range file.Funcs {
		names[fn.Name] = struct{}{}
	}
	for _, glob := range file.Globals {
		names[glob.Name] = struct{}{}
	}
	for _, st := range file.Structs {
		names[st.Name] = struct{}{}
	}
	for _, en := range file.Enums {
		names[en.Name] = struct{}{}
	}
	for _, state := range file.States {
		names[state.Name] = struct{}{}
	}
	for _, view := range file.Views {
		names[view.Name] = struct{}{}
	}
	for _, actor := range file.Actors {
		names[actor.Name] = struct{}{}
	}
	for _, proto := range file.Protocols {
		names[proto.Name] = struct{}{}
	}
	for _, capsule := range file.Capsules {
		if capsule == nil {
			continue
		}
		names[capsule.Name] = struct{}{}
	}
	return names
}

func QualifyName(module, name string) string {
	if module == "" {
		return name
	}
	return module + "." + name
}

func CheckedFuncFullName(module string, fn *frontend.FuncDecl) string {
	if fn != nil && fn.ExtensionOf != "" {
		return fn.Name
	}
	return QualifyName(module, fn.Name)
}
