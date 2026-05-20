package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"tetra_language/compiler"
)

func collectTetraFiles(paths []string) ([]string, error) {
	seen := map[string]struct{}{}
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if isCLIManagedSourceFile(path) {
				if _, ok := seen[path]; !ok {
					seen[path] = struct{}{}
					files = append(files, path)
				}
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
			if isCLIManagedSourceFile(p) {
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

func isCLIManagedSourceFile(path string) bool {
	if !compiler.IsSourceFile(path) {
		return false
	}
	base := filepath.Base(path)
	return base != compiler.CapsuleFileName && base != compiler.LegacyCapsuleFileName
}

var moduleDeclRE = regexp.MustCompile(`(?m)^\s*module\s+([A-Za-z0-9_.]+)\s*$`)

func modulePathFromSource(src []byte) string {
	m := moduleDeclRE.FindSubmatch(src)
	if len(m) != 2 {
		return ""
	}
	return string(m[1])
}

func moduleRelPath(module string) string {
	return moduleRelPathWithExtension(module, compiler.T4SourceExtension)
}

func moduleRootFromEntry(entryPath string, module string) (string, error) {
	absEntry, err := filepath.Abs(entryPath)
	if err != nil {
		return "", fmt.Errorf("resolve module root: %w", err)
	}
	root := absEntry
	for range strings.Split(module, ".") {
		root = filepath.Dir(root)
	}
	rel, err := filepath.Rel(root, absEntry)
	if err != nil {
		return "", fmt.Errorf("resolve module root: %w", err)
	}
	if !cliModuleRelPathMatches(module, rel) {
		return "", fmt.Errorf("%s: module '%s' must be in %s (or legacy %s)", absEntry, module, moduleRelPathWithExtension(module, compiler.T4SourceExtension), moduleRelPathWithExtension(module, compiler.LegacyTetraSourceExtension))
	}
	return root, nil
}

func defaultInputPath() string {
	if fileExists(compiler.DefaultSourceFileName) {
		return compiler.DefaultSourceFileName
	}
	if fileExists(compiler.LegacySourceFileName) {
		return compiler.LegacySourceFileName
	}
	return compiler.DefaultSourceFileName
}

func moduleRelPathWithExtension(module string, extension string) string {
	return filepath.FromSlash(strings.ReplaceAll(module, ".", "/") + extension)
}

func cliModuleRelPathMatches(module, rel string) bool {
	cleanRel := filepath.Clean(rel)
	for _, ext := range compiler.SourceExtensions() {
		if cleanRel == filepath.Clean(moduleRelPathWithExtension(module, ext)) {
			return true
		}
	}
	return false
}

func rewriteModuleDecl(src []byte, module string) []byte {
	return []byte(moduleDeclRE.ReplaceAllString(string(src), "module "+module))
}

func runnerSourcePathForModuleFile(entryPath string, src []byte, runnerIndex int) (string, []byte, error) {
	module := modulePathFromSource(src)
	if module == "" {
		return "", nil, fmt.Errorf("runner source has imports but no module declaration")
	}
	root, err := moduleRootFromEntry(entryPath, module)
	if err != nil {
		return "", nil, err
	}
	parts := strings.Split(module, ".")
	parts[len(parts)-1] = fmt.Sprintf("__tetra_test_runner_%d", runnerIndex)
	runnerModule := strings.Join(parts, ".")
	runnerPath := filepath.Join(root, moduleRelPath(runnerModule))
	return runnerPath, rewriteModuleDecl(src, runnerModule), nil
}
