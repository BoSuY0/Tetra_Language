package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/compiler"
)

type cliProjectContext struct {
	Found           bool
	Root            string
	CapsulePath     string
	LockPath        string
	Manifest        capsuleManifest
	Manifests       []capsuleManifest
	EntryPath       string
	SourceRoots     []string
	DependencyRoots []compiler.ModuleRoot
}

var defaultProjectSourceRoots = []string{"src", "ui", "tests", "drivers", "kernel", "game", "."}

func resolveCLIInput(input string) (string, compiler.WorldOptions, *cliProjectContext, error) {
	startDir, err := cliProjectStartDir(input)
	if err != nil {
		return "", compiler.WorldOptions{}, nil, err
	}
	ctx, err := discoverCLIProject(startDir)
	if err != nil {
		return "", compiler.WorldOptions{}, nil, err
	}
	if ctx == nil || !ctx.Found {
		if input == "" {
			input = defaultInputPath()
		}
		return input, compiler.WorldOptions{}, nil, nil
	}

	entry := input
	if entry == "" {
		entry = ctx.EntryPath
	} else if isProjectReference(input, ctx) {
		entry = ctx.EntryPath
	} else {
		entry, err = filepath.Abs(entry)
		if err != nil {
			return "", compiler.WorldOptions{}, nil, fmt.Errorf("resolve input path: %w", err)
		}
	}
	opt := compiler.WorldOptions{
		Root:            ctx.Root,
		SourceRoots:     append([]string(nil), ctx.SourceRoots...),
		DependencyRoots: append([]compiler.ModuleRoot(nil), ctx.DependencyRoots...),
	}
	return entry, opt, ctx, nil
}

func isProjectReference(input string, ctx *cliProjectContext) bool {
	if ctx == nil || !ctx.Found || strings.TrimSpace(input) == "" {
		return false
	}
	abs, err := filepath.Abs(input)
	if err != nil {
		return false
	}
	cleanAbs := filepath.Clean(abs)
	if info, err := os.Stat(cleanAbs); err == nil && info.IsDir() {
		return filepath.Clean(ctx.Root) == cleanAbs
	}
	return filepath.Clean(ctx.CapsulePath) == cleanAbs
}

func discoverCLIProject(startDir string) (*cliProjectContext, error) {
	capsulePath, root, ok, err := findProjectCapsule(startDir)
	if err != nil || !ok {
		return &cliProjectContext{}, err
	}
	manifest, err := parseCapsule(capsulePath)
	if err != nil {
		return nil, err
	}
	sourceRoots := projectSourceRoots(manifest)
	entryPath, err := resolveProjectEntry(root, manifest)
	if err != nil {
		return nil, err
	}
	dependencyRoots, dependencyManifests, err := projectDependencyGraph(root, manifest, map[string]int{root: projectDependencyVisiting}, []string{root})
	if err != nil {
		return nil, err
	}
	artifactRoots, err := projectArtifactInterfaceRoots(root, manifest.Artifacts)
	if err != nil {
		return nil, err
	}
	if capsuleHasInterfaceArtifacts(manifest.Artifacts) {
		dependencyRoots = nil
	}
	dependencyRoots = append(dependencyRoots, artifactRoots...)
	manifests := append([]capsuleManifest{manifest}, dependencyManifests...)
	return &cliProjectContext{
		Found:           true,
		Root:            root,
		CapsulePath:     capsulePath,
		LockPath:        findProjectLock(root),
		Manifest:        manifest,
		Manifests:       manifests,
		EntryPath:       entryPath,
		SourceRoots:     sourceRoots,
		DependencyRoots: dependencyRoots,
	}, nil
}

func findProjectCapsule(startDir string) (string, string, bool, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", "", false, err
	}
	for {
		for _, name := range []string{compiler.CapsuleFileName, compiler.LegacyCapsuleFileName} {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path, dir, true, nil
			} else if !os.IsNotExist(err) {
				return "", "", false, err
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", false, nil
		}
		dir = parent
	}
}

func findProjectLock(root string) string {
	path := filepath.Join(root, compiler.SemanticLockFileName)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func cliProjectStartDir(input string) (string, error) {
	if input == "" {
		return os.Getwd()
	}
	abs, err := filepath.Abs(input)
	if err != nil {
		return "", fmt.Errorf("resolve input path: %w", err)
	}
	info, err := os.Stat(abs)
	if err == nil && info.IsDir() {
		return abs, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	return filepath.Dir(abs), nil
}

func capsuleHasInterfaceArtifacts(artifacts []capsuleArtifact) bool {
	for _, artifact := range artifacts {
		if artifact.Kind == "interface" {
			return true
		}
	}
	return false
}

func projectArtifactInterfaceRoots(root string, artifacts []capsuleArtifact) ([]compiler.ModuleRoot, error) {
	var roots []compiler.ModuleRoot
	seen := map[string]struct{}{}
	for _, artifact := range artifacts {
		if artifact.Kind != "interface" {
			continue
		}
		relRoot, err := interfaceArtifactSourceRoot(root, artifact.Path)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[relRoot]; ok {
			continue
		}
		seen[relRoot] = struct{}{}
		roots = append(roots, compiler.ModuleRoot{
			Root:        root,
			SourceRoots: []string{relRoot},
		})
	}
	sort.Slice(roots, func(i, j int) bool {
		if roots[i].Root == roots[j].Root {
			return strings.Join(roots[i].SourceRoots, ",") < strings.Join(roots[j].SourceRoots, ",")
		}
		return roots[i].Root < roots[j].Root
	})
	return roots, nil
}

func projectArtifactObjectPaths(root string, artifacts []capsuleArtifact, target string) ([]string, error) {
	var paths []string
	for _, artifact := range artifacts {
		if artifact.Kind != "object" {
			continue
		}
		if artifact.Target != "" && target != "" && artifact.Target != target {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(artifact.Path))
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("artifact object %s: %w", artifact.Path, err)
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func interfaceArtifactSourceRoot(root string, relPath string) (string, error) {
	path := filepath.Join(root, filepath.FromSlash(relPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("artifact interface %s: %w", relPath, err)
	}
	moduleName := interfaceArtifactModuleName(raw)
	if moduleName == "" {
		return "", fmt.Errorf("artifact interface %s: missing module declaration", relPath)
	}
	moduleRel := filepath.ToSlash(moduleRelPathWithExtension(moduleName, compiler.T4InterfaceExtension))
	cleanRel := filepath.ToSlash(filepath.Clean(relPath))
	if cleanRel != moduleRel && !strings.HasSuffix(cleanRel, "/"+moduleRel) {
		return "", fmt.Errorf("artifact interface %s: module '%s' must be in %s", relPath, moduleName, moduleRel)
	}
	rootRel := strings.TrimSuffix(cleanRel, moduleRel)
	rootRel = strings.TrimSuffix(rootRel, "/")
	if rootRel == "" {
		return ".", nil
	}
	return rootRel, nil
}

func interfaceArtifactModuleName(raw []byte) string {
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "module ") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			fields := strings.Fields(name)
			if len(fields) == 0 {
				return ""
			}
			return fields[0]
		}
	}
	return ""
}

func resolveProjectEntry(root string, manifest capsuleManifest) (string, error) {
	if manifest.Entry != "" {
		rel, err := cleanProjectRelPath(manifest.Entry)
		if err != nil {
			return "", fmt.Errorf("%s: invalid entry: %w", manifest.Path, err)
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err != nil {
			return "", err
		}
		return path, nil
	}
	for _, rel := range []string{
		compiler.DefaultSourceFileName,
		filepath.ToSlash(filepath.Join("src", compiler.DefaultSourceFileName)),
		compiler.LegacySourceFileName,
		filepath.ToSlash(filepath.Join("src", compiler.LegacySourceFileName)),
	} {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("%s: missing project entry (set entry \"src/main.t4\" or add main.t4/src/main.t4)", manifest.Path)
}

func projectSourceRoots(manifest capsuleManifest) []string {
	roots := manifest.SourceRoots
	if len(roots) == 0 {
		roots = defaultProjectSourceRoots
	}
	return cleanProjectSourceRoots(roots)
}

func existingProjectSourcePaths(ctx *cliProjectContext) []string {
	if ctx == nil || !ctx.Found {
		return nil
	}
	seen := map[string]struct{}{}
	var paths []string
	for _, root := range ctx.SourceRoots {
		path := ctx.Root
		if root != "" {
			path = filepath.Join(ctx.Root, filepath.FromSlash(root))
		}
		if _, ok := seen[path]; ok {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func validateDiscoveredProjectLock(ctx *cliProjectContext, target string) error {
	if ctx == nil || !ctx.Found || ctx.LockPath == "" {
		return nil
	}
	if err := validateCapsuleGraph(ctx.Manifests, target); err != nil {
		return fmt.Errorf("%s: %w; repair with: %s", ctx.LockPath, err, projectSyncRepairCommand(ctx.Root, target, false))
	}
	issues, err := checkDeclaredCapsuleArtifacts(ctx.CapsulePath, target, ctx.LockPath, false)
	if err != nil {
		return fmt.Errorf("%s: %w", ctx.LockPath, err)
	}
	if len(issues) > 0 {
		issue := issues[0]
		detail := issue.Detail
		if detail != "" {
			detail = ": " + detail
		}
		repair := "; repair with: " + projectSyncRepairCommand(ctx.Root, target, false)
		if issue.Module != "" {
			return fmt.Errorf("%s: %s for %s at %s%s%s", ctx.LockPath, issue.Kind, issue.Module, issue.Path, detail, repair)
		}
		return fmt.Errorf("%s: %s at %s%s%s", ctx.LockPath, issue.Kind, issue.Path, detail, repair)
	}
	raw, err := os.ReadFile(ctx.LockPath)
	if err != nil {
		return fmt.Errorf("%s: %w", ctx.LockPath, err)
	}
	lock, err := decodeEcoLock(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", ctx.LockPath, err)
	}
	current, err := buildEcoLockWithArtifactHashes(ctx.Manifests)
	if err != nil {
		return fmt.Errorf("%s: %w", ctx.LockPath, err)
	}
	if lock.GraphSHA256 != current.GraphSHA256 {
		return fmt.Errorf("%s: project lock is stale: graph_sha256 %s, current %s; repair with: %s", ctx.LockPath, lock.GraphSHA256, current.GraphSHA256, projectSyncRepairCommand(ctx.Root, target, false))
	}
	return nil
}

func projectSyncRepairCommand(root string, target string, allTargets bool) string {
	var parts []string
	parts = append(parts, "tetra", "project", "sync")
	if allTargets {
		parts = append(parts, "--all-targets")
	} else if target != "" {
		parts = append(parts, "--target", target)
	}
	if root != "" {
		parts = append(parts, filepath.ToSlash(root))
	}
	return strings.Join(parts, " ")
}

func projectLinkObjects(ctx *cliProjectContext, target string, explicit []string) ([]string, error) {
	if ctx == nil || !ctx.Found {
		return append([]string(nil), explicit...), nil
	}
	projectObjects, err := projectArtifactObjectPaths(ctx.Root, ctx.Manifest.Artifacts, target)
	if err != nil {
		return nil, err
	}
	if len(projectObjects) == 0 {
		return append([]string(nil), explicit...), nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(projectObjects)+len(explicit))
	for _, path := range projectObjects {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	for _, path := range explicit {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out, nil
}

const (
	projectDependencyUnvisited = iota
	projectDependencyVisiting
	projectDependencyDone
)

func projectDependencyGraph(root string, manifest capsuleManifest, state map[string]int, stack []string) ([]compiler.ModuleRoot, []capsuleManifest, error) {
	var out []compiler.ModuleRoot
	var manifests []capsuleManifest
	for _, dep := range manifest.Dependencies {
		if dep.Path == "" {
			continue
		}
		depRoot, err := resolveDependencyProjectRoot(root, dep.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: dependency %s: %w", manifest.Path, dep.ID, err)
		}
		switch state[depRoot] {
		case projectDependencyVisiting:
			return nil, nil, fmt.Errorf("%s: capsule dependency cycle: %s", manifest.Path, describeProjectDependencyCycle(stack, depRoot))
		case projectDependencyDone:
			continue
		}
		state[depRoot] = projectDependencyVisiting
		capsulePath, err := findCapsulePath(depRoot)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: dependency %s: %w", manifest.Path, dep.ID, err)
		}
		depManifest, err := parseCapsule(capsulePath)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, compiler.ModuleRoot{
			Root:        depRoot,
			SourceRoots: projectSourceRoots(depManifest),
		})
		artifactRoots, err := projectArtifactInterfaceRoots(depRoot, depManifest.Artifacts)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, artifactRoots...)
		manifests = append(manifests, depManifest)
		transitiveRoots, transitiveManifests, err := projectDependencyGraph(depRoot, depManifest, state, append(stack, depRoot))
		if err != nil {
			return nil, nil, err
		}
		out = append(out, transitiveRoots...)
		manifests = append(manifests, transitiveManifests...)
		state[depRoot] = projectDependencyDone
	}
	return out, manifests, nil
}

func describeProjectDependencyCycle(stack []string, repeated string) string {
	start := 0
	for i, root := range stack {
		if root == repeated {
			start = i
			break
		}
	}
	cycle := append([]string(nil), stack[start:]...)
	cycle = append(cycle, repeated)
	for i := range cycle {
		cycle[i] = filepath.ToSlash(cycle[i])
	}
	return strings.Join(cycle, " -> ")
}

func resolveDependencyProjectRoot(root string, depPath string) (string, error) {
	if strings.TrimSpace(depPath) == "" {
		return "", fmt.Errorf("path is empty")
	}
	if strings.Contains(depPath, "\\") {
		return "", fmt.Errorf("path must use forward slashes")
	}
	path := filepath.FromSlash(depPath)
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		path = filepath.Dir(path)
	}
	return path, nil
}

func cleanProjectSourceRoots(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, root := range in {
		rel, err := cleanProjectRelPath(root)
		if err != nil {
			continue
		}
		if rel == "." {
			rel = ""
		}
		if _, ok := seen[rel]; ok {
			continue
		}
		seen[rel] = struct{}{}
		out = append(out, rel)
	}
	return out
}

func cleanProjectRelPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("path is empty")
	}
	if strings.Contains(value, "\\") {
		return "", fmt.Errorf("path must use forward slashes")
	}
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("path must be relative")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean == "." {
		return ".", nil
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path must stay inside project root")
	}
	return clean, nil
}
