package module

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/compiler/internal/formats"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/t4iface"
)

type World struct {
	EntryPath        string
	EntryModule      string
	Root             string
	SourceRoots      []string
	DependencyRoots  []ModuleRoot
	InterfaceModules map[string]bool
	InterfaceHashes  map[string]string
	Files            []*frontend.FileAST
	ByModule         map[string]*frontend.FileAST
}

type LoadOptions struct {
	Root            string
	SourceRoots     []string
	DependencyRoots []ModuleRoot
}

type ModuleRoot struct {
	Root        string
	SourceRoots []string
}

const (
	loadStateUnvisited = iota
	loadStateVisiting
	loadStateDone
)

func LoadWorld(entryPath string) (*World, error) {
	return LoadWorldOpt(entryPath, LoadOptions{})
}

func LoadWorldOpt(entryPath string, opt LoadOptions) (*World, error) {
	absEntry, err := filepath.Abs(entryPath)
	if err != nil {
		return nil, fmt.Errorf("resolve entry path: %w", err)
	}

	entryFile, err := parseModuleFileSetFromPath(absEntry)
	if err != nil {
		return nil, err
	}
	entryFile.Path = absEntry

	if entryFile.Module == "" && len(entryFile.Imports) > 0 {
		return nil, fmt.Errorf("%s: module declaration required for imports", absEntry)
	}

	sourceRoots := cleanSourceRoots(opt.SourceRoots)
	dependencyRoots, err := cleanModuleRoots(opt.DependencyRoots)
	if err != nil {
		return nil, err
	}
	root := filepath.Dir(absEntry)
	if opt.Root != "" {
		root, err = filepath.Abs(opt.Root)
		if err != nil {
			return nil, fmt.Errorf("resolve project root: %w", err)
		}
		rel, err := filepath.Rel(root, absEntry)
		if err != nil {
			return nil, fmt.Errorf("resolve project entry: %w", err)
		}
		if rel == "." || strings.HasPrefix(filepath.Clean(rel), "..") || filepath.IsAbs(rel) {
			return nil, fmt.Errorf("%s: entry is outside project root %s", absEntry, root)
		}
		if entryFile.Module != "" && !moduleRelPathMatchesInSourceRoots(entryFile.Module, rel, sourceRoots) {
			return nil, fmt.Errorf("%s: module '%s' must be in %s", absEntry, entryFile.Module, describeModuleSourceRootPaths(entryFile.Module, sourceRoots))
		}
	} else if entryFile.Module != "" {
		root, err = rootFromEntry(absEntry, entryFile.Module)
		if err != nil {
			return nil, err
		}
	}

	world := &World{
		EntryPath:        absEntry,
		EntryModule:      entryFile.Module,
		Root:             root,
		SourceRoots:      append([]string(nil), sourceRoots...),
		DependencyRoots:  append([]ModuleRoot(nil), dependencyRoots...),
		InterfaceModules: map[string]bool{},
		InterfaceHashes:  map[string]string{},
		Files:            []*frontend.FileAST{entryFile},
		ByModule:         map[string]*frontend.FileAST{},
	}
	world.ByModule[entryFile.Module] = entryFile

	state := map[string]int{}
	if entryFile.Module != "" {
		state[entryFile.Module] = loadStateVisiting
		if entryFile.InterfaceHash != "" {
			world.InterfaceModules[entryFile.Module] = true
			world.InterfaceHashes[entryFile.Module] = entryFile.InterfaceHash
		}
	}
	for _, imp := range entryFile.Imports {
		if err := world.loadModule(root, imp.Path, state, sourceRoots, dependencyRoots); err != nil {
			return nil, err
		}
	}
	if entryFile.Module != "" {
		state[entryFile.Module] = loadStateDone
	}

	return world, nil
}

func (w *World) loadModule(root, module string, state map[string]int, sourceRoots []string, dependencyRoots []ModuleRoot) error {
	switch state[module] {
	case loadStateVisiting:
		return fmt.Errorf("import cycle detected at '%s'", module)
	case loadStateDone:
		return nil
	}
	state[module] = loadStateVisiting

	path, isInterface, err := resolveModulePath(root, module, sourceRoots, dependencyRoots)
	if err != nil {
		return fmt.Errorf("load module '%s': %w", module, err)
	}
	file, err := parseModuleFileSetFromPath(path)
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
	if isInterface {
		w.InterfaceModules[file.Module] = true
		if file.InterfaceHash != "" {
			w.InterfaceHashes[file.Module] = file.InterfaceHash
		}
	}
	w.Files = append(w.Files, file)

	for _, imp := range file.Imports {
		if err := w.loadModule(root, imp.Path, state, sourceRoots, dependencyRoots); err != nil {
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
	interfaceHash := ""
	if filepath.Ext(path) == formats.T4InterfaceExtension {
		hash, err := t4iface.ValidateHash(src)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		interfaceHash = hash
	}
	file, err := frontend.ParseFile(src, path)
	if err != nil {
		return nil, err
	}
	file.Path = path
	file.InterfaceHash = interfaceHash
	if err := validateImportPaths(file); err != nil {
		return nil, err
	}
	return file, nil
}

func parseModuleFileSetFromPath(path string) (*frontend.FileAST, error) {
	file, err := parseFileFromPath(path)
	if err != nil {
		return nil, err
	}
	if file.Module == "" || file.InterfaceHash != "" {
		return file, nil
	}
	fragmentPaths, err := moduleFragmentPaths(path)
	if err != nil {
		return nil, err
	}
	if len(fragmentPaths) == 0 {
		return file, nil
	}
	fragments := make([]*frontend.FileAST, 0, len(fragmentPaths))
	for _, fragmentPath := range fragmentPaths {
		fragment, err := parseFileFromPath(fragmentPath)
		if err != nil {
			return nil, fmt.Errorf("load module fragment '%s': %w", fragmentPath, err)
		}
		if fragment.InterfaceHash != "" {
			return nil, fmt.Errorf("%s: module fragments cannot be interface files", fragmentPath)
		}
		if fragment.Module != file.Module {
			return nil, fmt.Errorf("%s: module fragment declaration '%s' does not match primary module '%s'", fragmentPath, fragment.Module, file.Module)
		}
		mergeModuleFragment(file, fragment)
		fragments = append(fragments, fragment)
	}
	if err := validateImportPaths(file); err != nil {
		return nil, err
	}
	file.Src = mergedModuleSource(file.Module, file.Imports, append([]*frontend.FileAST{file}, fragments...))
	return file, nil
}

func moduleFragmentPaths(path string) ([]string, error) {
	ext := filepath.Ext(path)
	if ext != formats.T4SourceExtension && ext != formats.LegacyTetraSourceExtension {
		return nil, nil
	}
	stem := strings.TrimSuffix(filepath.Base(path), ext)
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), stem+".parts", "*"+ext))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

func mergeModuleFragment(file, fragment *frontend.FileAST) {
	file.Imports = append(file.Imports, fragment.Imports...)
	file.Capsules = append(file.Capsules, fragment.Capsules...)
	file.Enums = append(file.Enums, fragment.Enums...)
	file.Structs = append(file.Structs, fragment.Structs...)
	file.States = append(file.States, fragment.States...)
	file.Views = append(file.Views, fragment.Views...)
	file.Actors = append(file.Actors, fragment.Actors...)
	file.Protocols = append(file.Protocols, fragment.Protocols...)
	file.Extensions = append(file.Extensions, fragment.Extensions...)
	file.Impls = append(file.Impls, fragment.Impls...)
	file.Globals = append(file.Globals, fragment.Globals...)
	file.Funcs = append(file.Funcs, fragment.Funcs...)
	file.Tests = append(file.Tests, fragment.Tests...)
}

func mergedModuleSource(module string, imports []frontend.ImportDecl, files []*frontend.FileAST) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "module %s\n", module)
	seenImports := map[string]struct{}{}
	for _, imp := range imports {
		line := formatImportDecl(imp)
		if _, ok := seenImports[line]; ok {
			continue
		}
		seenImports[line] = struct{}{}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	for _, file := range files {
		b.WriteString(stripModuleAndImportLines(string(file.Src)))
		if !strings.HasSuffix(b.String(), "\n") {
			b.WriteByte('\n')
		}
	}
	return []byte(b.String())
}

func formatImportDecl(imp frontend.ImportDecl) string {
	var b strings.Builder
	if imp.Public {
		b.WriteString("pub ")
	}
	b.WriteString("import ")
	if len(imp.Items) > 0 {
		b.WriteString(imp.Path)
		b.WriteString(".{")
		for i, item := range imp.Items {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(item)
		}
		b.WriteByte('}')
		return b.String()
	}
	b.WriteString(imp.Path)
	if imp.Alias != "" {
		parts := strings.Split(imp.Path, ".")
		if imp.Alias != parts[len(parts)-1] {
			b.WriteString(" as ")
			b.WriteString(imp.Alias)
		}
	}
	return b.String()
}

func stripModuleAndImportLines(src string) string {
	var out []string
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") ||
			strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "pub import ") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
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
	return formats.ModuleRelPath(module, formats.T4SourceExtension)
}

func resolveModulePath(root, module string, sourceRoots []string, dependencyRoots []ModuleRoot) (string, bool, error) {
	matches, err := resolveModuleMatchesInRoot(root, module, sourceRoots)
	if err != nil {
		return "", false, err
	}
	for _, depRoot := range dependencyRoots {
		depMatches, err := resolveModuleMatchesInRoot(depRoot.Root, module, depRoot.SourceRoots)
		if err != nil {
			return "", false, err
		}
		matches = append(matches, depMatches...)
	}
	if len(matches) > 1 {
		paths := make([]string, 0, len(matches))
		for _, match := range matches {
			paths = append(paths, match.Path)
		}
		return "", false, fmt.Errorf("duplicate module '%s' (%s)", module, strings.Join(paths, ", "))
	}
	if len(matches) == 1 {
		return matches[0].Path, matches[0].IsInterface, nil
	}
	sourceCandidates := cleanSourceRoots(sourceRoots)
	if len(sourceCandidates) == 0 {
		sourceCandidates = []string{""}
	}
	return filepath.Join(root, sourceCandidates[0], moduleLoadCandidateRelPaths(module)[0]), false, nil
}

type modulePathMatch struct {
	Path        string
	IsInterface bool
}

func resolveModuleMatchesInRoot(root, module string, sourceRoots []string) ([]modulePathMatch, error) {
	moduleCandidates := moduleLoadCandidateRelPaths(module)
	sourceCandidates := cleanSourceRoots(sourceRoots)
	if len(sourceCandidates) == 0 {
		sourceCandidates = []string{""}
	}
	var matches []modulePathMatch
	for _, sourceRoot := range sourceCandidates {
		for _, rel := range moduleCandidates {
			path := filepath.Join(root, sourceRoot, rel)
			if _, err := os.Stat(path); err == nil {
				matches = append(matches, modulePathMatch{
					Path:        path,
					IsInterface: filepath.Ext(path) == formats.T4InterfaceExtension,
				})
				break
			} else if !os.IsNotExist(err) {
				return nil, err
			}
		}
	}
	return matches, nil
}

func resolveModulePathInRoot(root, module string, sourceRoots []string) (string, bool, bool, error) {
	moduleCandidates := moduleLoadCandidateRelPaths(module)
	sourceCandidates := cleanSourceRoots(sourceRoots)
	if len(sourceCandidates) == 0 {
		sourceCandidates = []string{""}
	}
	for _, sourceRoot := range sourceCandidates {
		for _, rel := range moduleCandidates {
			path := filepath.Join(root, sourceRoot, rel)
			if _, err := os.Stat(path); err == nil {
				return path, filepath.Ext(path) == formats.T4InterfaceExtension, true, nil
			} else if !os.IsNotExist(err) {
				return "", false, false, err
			}
		}
	}
	return "", false, false, nil
}

func moduleLoadCandidateRelPaths(module string) []string {
	candidates := formats.ModuleCandidateRelPaths(module)
	candidates = append(candidates, formats.ModuleRelPath(module, formats.T4InterfaceExtension))
	return candidates
}

func rootFromEntry(entryPath, module string) (string, error) {
	parts := strings.Split(module, ".")
	root := entryPath
	for range parts {
		root = filepath.Dir(root)
	}
	rel, err := filepath.Rel(root, entryPath)
	if err != nil {
		return "", fmt.Errorf("resolve module root: %w", err)
	}
	if !moduleRelPathMatches(module, rel) {
		return "", fmt.Errorf("%s: module '%s' must be in %s (or legacy %s)", entryPath, module, formats.ModuleRelPath(module, formats.T4SourceExtension), formats.ModuleRelPath(module, formats.LegacyTetraSourceExtension))
	}
	return root, nil
}

func moduleRelPathMatches(module, rel string) bool {
	cleanRel := filepath.Clean(rel)
	for _, candidate := range formats.ModuleCandidateRelPaths(module) {
		if cleanRel == filepath.Clean(candidate) {
			return true
		}
	}
	return false
}

func moduleRelPathMatchesInSourceRoots(module, rel string, sourceRoots []string) bool {
	cleanRel := filepath.Clean(rel)
	roots := cleanSourceRoots(sourceRoots)
	if len(roots) == 0 {
		roots = []string{""}
	}
	for _, root := range roots {
		for _, candidate := range formats.ModuleCandidateRelPaths(module) {
			if cleanRel == filepath.Clean(filepath.Join(root, candidate)) {
				return true
			}
		}
	}
	return false
}

func describeModuleSourceRootPaths(module string, sourceRoots []string) string {
	roots := cleanSourceRoots(sourceRoots)
	if len(roots) == 0 {
		roots = []string{""}
	}
	var paths []string
	for _, root := range roots {
		paths = append(paths, filepath.ToSlash(filepath.Join(root, formats.ModuleRelPath(module, formats.T4SourceExtension))))
	}
	return strings.Join(paths, " or ")
}

func cleanSourceRoots(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, root := range in {
		root = filepath.Clean(root)
		if root == "." {
			root = ""
		}
		if strings.HasPrefix(root, "..") || filepath.IsAbs(root) {
			continue
		}
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		out = append(out, root)
	}
	return out
}

func cleanModuleRoots(in []ModuleRoot) ([]ModuleRoot, error) {
	seen := map[string]struct{}{}
	var out []ModuleRoot
	for _, item := range in {
		if item.Root == "" {
			continue
		}
		root, err := filepath.Abs(item.Root)
		if err != nil {
			return nil, fmt.Errorf("resolve dependency root: %w", err)
		}
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		out = append(out, ModuleRoot{
			Root:        root,
			SourceRoots: cleanSourceRoots(item.SourceRoots),
		})
	}
	return out, nil
}
