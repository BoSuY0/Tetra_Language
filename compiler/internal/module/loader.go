package module

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

type World struct {
	EntryPath   string
	EntryModule string
	Root        string
	Files       []*frontend.FileAST
	ByModule    map[string]*frontend.FileAST
}

const (
	loadStateUnvisited = iota
	loadStateVisiting
	loadStateDone
)

func LoadWorld(entryPath string) (*World, error) {
	absEntry, err := filepath.Abs(entryPath)
	if err != nil {
		return nil, fmt.Errorf("resolve entry path: %w", err)
	}

	entryFile, err := parseFileFromPath(absEntry)
	if err != nil {
		return nil, err
	}
	entryFile.Path = absEntry

	if entryFile.Module == "" && len(entryFile.Imports) > 0 {
		return nil, fmt.Errorf("%s: module declaration required for imports", absEntry)
	}

	root := filepath.Dir(absEntry)
	if entryFile.Module != "" {
		root, err = rootFromEntry(absEntry, entryFile.Module)
		if err != nil {
			return nil, err
		}
	}

	world := &World{
		EntryPath:   absEntry,
		EntryModule: entryFile.Module,
		Root:        root,
		Files:       []*frontend.FileAST{entryFile},
		ByModule:    map[string]*frontend.FileAST{},
	}
	world.ByModule[entryFile.Module] = entryFile

	state := map[string]int{}
	if entryFile.Module != "" {
		state[entryFile.Module] = loadStateVisiting
	}
	for _, imp := range entryFile.Imports {
		if err := world.loadModule(root, imp.Path, state); err != nil {
			return nil, err
		}
	}
	if entryFile.Module != "" {
		state[entryFile.Module] = loadStateDone
	}

	return world, nil
}

func (w *World) loadModule(root, module string, state map[string]int) error {
	switch state[module] {
	case loadStateVisiting:
		return fmt.Errorf("import cycle detected at '%s'", module)
	case loadStateDone:
		return nil
	}
	state[module] = loadStateVisiting

	path := filepath.Join(root, moduleToRelPath(module))
	file, err := parseFileFromPath(path)
	if err != nil {
		return fmt.Errorf("load module '%s': %w", module, err)
	}
	file.Path = path

	if file.Module == "" {
		return fmt.Errorf("%s: module declaration required", path)
	}
	if existing, ok := w.ByModule[file.Module]; ok {
		if existing.Path != file.Path {
			return fmt.Errorf("duplicate module '%s' (%s, %s)", file.Module, existing.Path, file.Path)
		}
		state[module] = loadStateDone
		return nil
	}
	if file.Module != module {
		return fmt.Errorf("%s: module declaration '%s' does not match import '%s'", path, file.Module, module)
	}

	w.ByModule[file.Module] = file
	w.Files = append(w.Files, file)

	for _, imp := range file.Imports {
		if err := w.loadModule(root, imp.Path, state); err != nil {
			return err
		}
	}
	state[module] = loadStateDone
	return nil
}

func parseFileFromPath(path string) (*frontend.FileAST, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}
	file, err := frontend.ParseFile(src, path)
	if err != nil {
		return nil, err
	}
	file.Path = path
	if err := validateImportPaths(file); err != nil {
		return nil, err
	}
	return file, nil
}

func validateImportPaths(file *frontend.FileAST) error {
	seen := make(map[string]frontend.Position, len(file.Imports))
	for _, imp := range file.Imports {
		if first, ok := seen[imp.Path]; ok {
			return fmt.Errorf("%s: duplicate import '%s' (first imported at %s)", frontend.FormatPos(imp.At), imp.Path, frontend.FormatPos(first))
		}
		seen[imp.Path] = imp.At
	}
	return nil
}

func moduleToRelPath(module string) string {
	return filepath.FromSlash(strings.ReplaceAll(module, ".", "/") + ".tetra")
}

func rootFromEntry(entryPath, module string) (string, error) {
	parts := strings.Split(module, ".")
	root := entryPath
	for range parts {
		root = filepath.Dir(root)
	}
	expectedRel := moduleToRelPath(module)
	rel, err := filepath.Rel(root, entryPath)
	if err != nil {
		return "", fmt.Errorf("resolve module root: %w", err)
	}
	if rel != expectedRel {
		return "", fmt.Errorf("%s: module '%s' must be in %s", entryPath, module, expectedRel)
	}
	return root, nil
}
