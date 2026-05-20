package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/compiler"
)

func runEcoArtifacts(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra eco artifacts <build|check> [options]")
		return 2
	}
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra eco artifacts <build|check> [options]")
		return 0
	}
	switch args[0] {
	case "build":
		return runEcoArtifactsBuild(args[1:], stdout, stderr)
	case "check":
		return runEcoArtifactsCheck(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown eco artifacts command %q\n", args[0])
		return 2
	}
}

func runEcoArtifactsBuild(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco artifacts build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetFlag := fs.String("target", "", "native target triple for generated .tobj artifacts")
	lockPath := fs.String("lock", "", "path to write semantic lock; defaults to project Tetra.lock")
	checkOnly := fs.Bool("check", false, "dry-run and report pending artifact changes without writing files")
	allTargets := fs.Bool("all-targets", false, "build artifacts for every native target listed in Capsule.t4")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "eco artifacts build accepts at most one Capsule.t4 path")
		return 2
	}
	capsulePath := defaultCapsulePath()
	if fs.NArg() == 1 {
		capsulePath = fs.Arg(0)
	}
	if *checkOnly {
		issues, err := checkCapsuleArtifacts(capsulePath, *targetFlag, *lockPath, *allTargets)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if len(issues) > 0 {
			writeArtifactIssues(stdout, issues, true)
			return 1
		}
		fmt.Fprintf(stdout, "Artifacts current: %s\n", capsulePath)
		return 0
	}
	if err := buildCapsuleArtifacts(capsulePath, capsuleArtifactBuildOptions{
		Target:     *targetFlag,
		LockPath:   *lockPath,
		Jobs:       *jobs,
		AllTargets: *allTargets,
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Artifacts built: %s\n", capsulePath)
	return 0
}

func runEcoArtifactsCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco artifacts check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetFlag := fs.String("target", "", "native target triple for object freshness checks")
	lockPath := fs.String("lock", "", "path to semantic lock; defaults to project Tetra.lock")
	allTargets := fs.Bool("all-targets", false, "check artifacts for every native target listed in Capsule.t4")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "eco artifacts check accepts at most one Capsule.t4 path")
		return 2
	}
	capsulePath := defaultCapsulePath()
	if fs.NArg() == 1 {
		capsulePath = fs.Arg(0)
	}
	issues, err := checkCapsuleArtifacts(capsulePath, *targetFlag, *lockPath, *allTargets)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(issues) > 0 {
		writeArtifactIssues(stderr, issues, false)
		return 1
	}
	fmt.Fprintf(stdout, "Artifacts current: %s\n", capsulePath)
	return 0
}

type generatedCapsuleArtifact struct {
	Artifact capsuleArtifact
	Module   string
	Source   string
}

type capsuleSourceModule struct {
	Module string
	Path   string
}

type capsuleArtifactBuildOptions struct {
	Target     string
	LockPath   string
	Jobs       int
	AllTargets bool
}

type capsuleArtifactPlan struct {
	RootManifest        capsuleManifest
	Manifests           []capsuleManifest
	DependencyManifests []capsuleManifest
	Root                string
	Targets             []string
	Expected            []generatedCapsuleArtifact
}

type artifactIssue struct {
	Kind   string
	Module string
	Path   string
	Detail string
	Repair string
}

func buildCapsuleArtifacts(capsulePath string, opt capsuleArtifactBuildOptions) error {
	manifests, err := parseCapsuleGraphArgs([]string{capsulePath})
	if err != nil {
		return err
	}
	if len(manifests) == 0 {
		return fmt.Errorf("%s: no capsule graph", capsulePath)
	}
	rootManifest := manifests[0]
	root := filepath.Dir(rootManifest.Path)
	targets, err := resolveArtifactBuildTargets(rootManifest, opt.Target, opt.AllTargets)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("eco artifacts build requires at least one native object target")
	}
	for _, target := range targets {
		if err := validateCapsuleGraph(manifests, target); err != nil {
			return err
		}
	}
	lockPath := opt.LockPath
	if lockPath == "" {
		lockPath = filepath.Join(root, compiler.SemanticLockFileName)
	}
	jobs := opt.Jobs
	if jobs < 1 {
		jobs = 1
	}

	var generated []generatedCapsuleArtifact
	dependencyManifests := manifests[1:]
	for _, depManifest := range dependencyManifests {
		depRoot := filepath.Dir(depManifest.Path)
		modules, err := capsuleSourceModules(depManifest)
		if err != nil {
			return err
		}
		buildOpt, err := dependencyArtifactBuildOptions(depRoot, depManifest, jobs)
		if err != nil {
			return err
		}
		for _, module := range modules {
			ifaceRel := interfaceArtifactRelPath(module.Module)
			generated = append(generated, generatedCapsuleArtifact{
				Artifact: capsuleArtifact{Kind: "interface", Path: ifaceRel},
				Module:   module.Module,
				Source:   module.Path,
			})
			ifacePath := filepath.Join(root, filepath.FromSlash(ifaceRel))
			if err := os.MkdirAll(filepath.Dir(ifacePath), 0o755); err != nil {
				return err
			}
			iface, err := compiler.GenerateInterfaceFile(module.Path)
			if err != nil {
				return err
			}
			if err := os.WriteFile(ifacePath, iface, 0o644); err != nil {
				return err
			}

			for _, target := range targets {
				objectRel := objectArtifactRelPath(module.Module, target)
				objectPath := filepath.Join(root, filepath.FromSlash(objectRel))
				if err := os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
					return err
				}
				if _, err := compiler.BuildFileWithStatsOpt(module.Path, objectPath, target, buildOpt); err != nil {
					return err
				}
				generated = append(generated, generatedCapsuleArtifact{
					Artifact: capsuleArtifact{Kind: "object", Target: target, Path: objectRel},
					Module:   module.Module,
					Source:   module.Path,
				})
			}
		}
	}
	if len(dependencyManifests) > 0 {
		seedRel := seedArtifactRelPath(rootManifest.Name)
		seedPath := filepath.Join(root, filepath.FromSlash(seedRel))
		if err := os.MkdirAll(filepath.Dir(seedPath), 0o755); err != nil {
			return err
		}
		if err := writeDependencySeed(seedPath, dependencyManifests); err != nil {
			return err
		}
		generated = append(generated, generatedCapsuleArtifact{
			Artifact: capsuleArtifact{Kind: "seed", Path: seedRel},
		})
	}
	if err := appendGeneratedArtifactsToCapsule(rootManifest.Path, rootManifest.Artifacts, generated); err != nil {
		return err
	}
	updatedManifests, err := parseCapsuleGraphArgs([]string{rootManifest.Path})
	if err != nil {
		return err
	}
	for _, target := range targets {
		if err := validateCapsuleGraph(updatedManifests, target); err != nil {
			return err
		}
	}
	return writeEcoLock(lockPath, updatedManifests)
}

func resolveArtifactBuildTargets(manifest capsuleManifest, targetFlag string, allTargets bool) ([]string, error) {
	return resolveNativeCapsuleTargets(manifest, nativeTargetResolutionOptions{
		TargetFlag:   targetFlag,
		AllTargets:   allTargets,
		Command:      "eco artifacts build",
		RejectWASM:   true,
		RequireAny:   true,
		RequireHost:  true,
		HostHelpFlag: "--target",
	})
}

func resolveDeclaredArtifactTargets(manifest capsuleManifest, targetFlag string, allTargets bool) ([]string, error) {
	return resolveNativeCapsuleTargets(manifest, nativeTargetResolutionOptions{
		TargetFlag: targetFlag,
		AllTargets: allTargets,
	})
}

type nativeTargetResolutionOptions struct {
	TargetFlag   string
	AllTargets   bool
	Command      string
	RejectWASM   bool
	RequireAny   bool
	RequireHost  bool
	HostHelpFlag string
}

func resolveNativeCapsuleTargets(manifest capsuleManifest, opt nativeTargetResolutionOptions) ([]string, error) {
	if opt.TargetFlag != "" {
		return resolveNativeCapsuleTarget(opt.TargetFlag, opt)
	}
	if opt.AllTargets {
		targets, err := nativeCapsuleManifestTargets(manifest)
		if err != nil {
			return nil, err
		}
		if opt.RequireAny && len(targets) == 0 {
			return nil, fmt.Errorf("%s --all-targets found no native targets in Capsule.t4", opt.Command)
		}
		return targets, nil
	}
	if len(manifest.Targets) > 0 {
		return resolveNativeCapsuleTarget(manifest.Targets[0], opt)
	}
	host, ok := hostTarget()
	if !ok {
		if opt.RequireHost {
			helpFlag := opt.HostHelpFlag
			if helpFlag == "" {
				helpFlag = "--target"
			}
			return nil, fmt.Errorf("host target unsupported; pass %s", helpFlag)
		}
		return nil, nil
	}
	return []string{host}, nil
}

func resolveNativeCapsuleTarget(raw string, opt nativeTargetResolutionOptions) ([]string, error) {
	target, err := normalizeCapsuleTarget(raw)
	if err != nil {
		return nil, err
	}
	if isWASMTargetTriple(target) {
		if opt.RejectWASM {
			return nil, fmt.Errorf("%s requires a native object target, got %s", opt.Command, target)
		}
		return nil, nil
	}
	return []string{target}, nil
}

func nativeCapsuleManifestTargets(manifest capsuleManifest) ([]string, error) {
	seen := map[string]struct{}{}
	var targets []string
	for _, raw := range manifest.Targets {
		target, err := normalizeCapsuleTarget(raw)
		if err != nil {
			return nil, err
		}
		if isWASMTargetTriple(target) {
			continue
		}
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets, nil
}

func planCapsuleArtifacts(capsulePath string, targetFlag string, allTargets bool) (capsuleArtifactPlan, error) {
	manifests, err := parseCapsuleGraphArgs([]string{capsulePath})
	if err != nil {
		return capsuleArtifactPlan{}, err
	}
	if len(manifests) == 0 {
		return capsuleArtifactPlan{}, fmt.Errorf("%s: no capsule graph", capsulePath)
	}
	rootManifest := manifests[0]
	targets, err := resolveArtifactBuildTargets(rootManifest, targetFlag, allTargets)
	if err != nil {
		return capsuleArtifactPlan{}, err
	}
	return planCapsuleArtifactsForTargets(manifests, targets)
}

func planDeclaredCapsuleArtifacts(capsulePath string, targetFlag string, allTargets bool) (capsuleArtifactPlan, error) {
	manifests, err := parseCapsuleGraphArgs([]string{capsulePath})
	if err != nil {
		return capsuleArtifactPlan{}, err
	}
	if len(manifests) == 0 {
		return capsuleArtifactPlan{}, fmt.Errorf("%s: no capsule graph", capsulePath)
	}
	targets, err := resolveDeclaredArtifactTargets(manifests[0], targetFlag, allTargets)
	if err != nil {
		return capsuleArtifactPlan{}, err
	}
	return planCapsuleArtifactsForTargets(manifests, targets)
}

func planCapsuleArtifactsForTargets(manifests []capsuleManifest, targets []string) (capsuleArtifactPlan, error) {
	if len(manifests) == 0 {
		return capsuleArtifactPlan{}, fmt.Errorf("no capsule graph")
	}
	rootManifest := manifests[0]
	root := filepath.Dir(rootManifest.Path)
	dependencyManifests := manifests[1:]
	var expected []generatedCapsuleArtifact
	seenInterfaces := map[string]struct{}{}
	for _, depManifest := range dependencyManifests {
		modules, err := capsuleSourceModules(depManifest)
		if err != nil {
			return capsuleArtifactPlan{}, err
		}
		for _, module := range modules {
			ifaceRel := interfaceArtifactRelPath(module.Module)
			if _, ok := seenInterfaces[ifaceRel]; !ok {
				seenInterfaces[ifaceRel] = struct{}{}
				expected = append(expected, generatedCapsuleArtifact{
					Artifact: capsuleArtifact{Kind: "interface", Path: ifaceRel},
					Module:   module.Module,
					Source:   module.Path,
				})
			}
			for _, target := range targets {
				expected = append(expected, generatedCapsuleArtifact{
					Artifact: capsuleArtifact{Kind: "object", Target: target, Path: objectArtifactRelPath(module.Module, target)},
					Module:   module.Module,
					Source:   module.Path,
				})
			}
		}
	}
	if len(dependencyManifests) > 0 {
		expected = append(expected, generatedCapsuleArtifact{
			Artifact: capsuleArtifact{Kind: "seed", Path: seedArtifactRelPath(rootManifest.Name)},
		})
	}
	return capsuleArtifactPlan{
		RootManifest:        rootManifest,
		Manifests:           manifests,
		DependencyManifests: dependencyManifests,
		Root:                root,
		Targets:             targets,
		Expected:            expected,
	}, nil
}

func checkCapsuleArtifacts(capsulePath string, targetFlag string, lockPath string, allTargets bool) ([]artifactIssue, error) {
	return checkCapsuleArtifactsOpt(capsulePath, targetFlag, lockPath, allTargets, true)
}

func checkDeclaredCapsuleArtifacts(capsulePath string, targetFlag string, lockPath string, allTargets bool) ([]artifactIssue, error) {
	return checkCapsuleArtifactsOpt(capsulePath, targetFlag, lockPath, allTargets, false)
}

func checkCapsuleArtifactsOpt(capsulePath string, targetFlag string, lockPath string, allTargets bool, requireExpected bool) ([]artifactIssue, error) {
	var plan capsuleArtifactPlan
	var err error
	if requireExpected {
		plan, err = planCapsuleArtifacts(capsulePath, targetFlag, allTargets)
	} else {
		plan, err = planDeclaredCapsuleArtifacts(capsulePath, targetFlag, allTargets)
	}
	if err != nil {
		return nil, err
	}
	if lockPath == "" {
		lockPath = filepath.Join(plan.Root, compiler.SemanticLockFileName)
	}
	repair := artifactRepairCommand(plan.RootManifest.Path, plan.Targets, allTargets)
	var issues []artifactIssue
	declared := map[string]capsuleArtifact{}
	for _, artifact := range plan.RootManifest.Artifacts {
		declared[capsuleArtifactKey(artifact)] = artifact
	}
	for _, item := range plan.Expected {
		key := capsuleArtifactKey(item.Artifact)
		if _, ok := declared[key]; !ok {
			if requireExpected {
				issues = append(issues, artifactIssue{
					Kind:   "missing " + item.Artifact.Kind + " artifact",
					Module: item.Module,
					Path:   item.Artifact.Path,
					Detail: "Capsule.t4 does not declare this generated artifact",
					Repair: repair,
				})
			}
			continue
		}
		path := filepath.Join(plan.Root, filepath.FromSlash(item.Artifact.Path))
		raw, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, artifactIssue{
				Kind:   "missing " + item.Artifact.Kind + " artifact",
				Module: item.Module,
				Path:   item.Artifact.Path,
				Detail: err.Error(),
				Repair: repair,
			})
			continue
		}
		switch item.Artifact.Kind {
		case "interface":
			expected, err := sourcePublicAPIHash(item.Source)
			if err != nil {
				return nil, err
			}
			actual, err := compiler.InterfaceFingerprintFromT4I(raw)
			if err != nil {
				issues = append(issues, artifactIssue{Kind: "invalid interface artifact", Module: item.Module, Path: item.Artifact.Path, Detail: err.Error(), Repair: repair})
				continue
			}
			if actual != expected {
				issues = append(issues, artifactIssue{
					Kind:   "stale interface artifact",
					Module: item.Module,
					Path:   item.Artifact.Path,
					Detail: fmt.Sprintf("expected public API %s, artifact has %s", expected, actual),
					Repair: repair,
				})
			}
		case "object":
			obj, err := compiler.ReadObject(path)
			if err != nil {
				issues = append(issues, artifactIssue{Kind: "invalid object artifact", Module: item.Module, Path: item.Artifact.Path, Detail: err.Error(), Repair: repair})
				continue
			}
			if obj.Target != "" && obj.Target != item.Artifact.Target {
				issues = append(issues, artifactIssue{
					Kind:   "wrong-target object artifact",
					Module: item.Module,
					Path:   item.Artifact.Path,
					Detail: fmt.Sprintf("expected target %s, object has %s", item.Artifact.Target, obj.Target),
					Repair: repair,
				})
			}
			expectedAPI, err := sourcePublicAPIHash(item.Source)
			if err != nil {
				return nil, err
			}
			if obj.PublicAPIHash != expectedAPI {
				issues = append(issues, artifactIssue{
					Kind:   "stale object artifact",
					Module: item.Module,
					Path:   item.Artifact.Path,
					Detail: fmt.Sprintf("expected public API %s, object has %s", expectedAPI, obj.PublicAPIHash),
					Repair: repair,
				})
			}
			expectedSrc, err := sourceSHA256(item.Source)
			if err != nil {
				return nil, err
			}
			if obj.SrcHash != expectedSrc {
				issues = append(issues, artifactIssue{
					Kind:   "stale object artifact",
					Module: item.Module,
					Path:   item.Artifact.Path,
					Detail: "source hash changed since object build",
					Repair: repair,
				})
			}
		case "seed":
			var seed ecoSeed
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&seed); err != nil {
				issues = append(issues, artifactIssue{Kind: "invalid seed artifact", Path: item.Artifact.Path, Detail: err.Error(), Repair: repair})
				continue
			}
			currentLock, err := buildEcoLockWithArtifactHashes(plan.DependencyManifests)
			if err != nil {
				return nil, err
			}
			normalizeLock(&seed.Lock)
			if seed.Lock.GraphSHA256 != currentLock.GraphSHA256 {
				issues = append(issues, artifactIssue{
					Kind:   "stale seed artifact",
					Path:   item.Artifact.Path,
					Detail: fmt.Sprintf("expected dependency graph %s, seed has %s", currentLock.GraphSHA256, seed.Lock.GraphSHA256),
					Repair: repair,
				})
			}
		}
	}
	if _, err := os.Stat(lockPath); err != nil {
		if os.IsNotExist(err) {
			if requireExpected {
				issues = append(issues, artifactIssue{Kind: "missing lock", Path: filepath.ToSlash(lockPath), Detail: "Tetra.lock is required for locked project builds", Repair: repair})
			}
		} else {
			return nil, err
		}
	} else {
		raw, err := os.ReadFile(lockPath)
		if err != nil {
			return nil, err
		}
		lock, err := decodeEcoLock(raw)
		if err != nil {
			issues = append(issues, artifactIssue{Kind: "invalid lock", Path: filepath.ToSlash(lockPath), Detail: err.Error(), Repair: repair})
		} else {
			current, err := buildEcoLockWithArtifactHashes(plan.Manifests)
			if err != nil {
				return nil, err
			}
			if lock.GraphSHA256 != current.GraphSHA256 {
				issues = append(issues, artifactIssue{
					Kind:   "stale lock",
					Path:   filepath.ToSlash(lockPath),
					Detail: fmt.Sprintf("expected graph %s, lock has %s", current.GraphSHA256, lock.GraphSHA256),
					Repair: repair,
				})
			}
		}
	}
	return dedupeArtifactIssues(issues), nil
}

func sourcePublicAPIHash(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return compiler.InterfaceFingerprintFromSource(raw, path)
}

func sourceSHA256(path string) ([32]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func artifactRepairCommand(capsulePath string, targets []string, allTargets bool) string {
	if allTargets {
		return fmt.Sprintf("tetra eco artifacts build --all-targets --lock %s %s", compiler.SemanticLockFileName, filepath.ToSlash(capsulePath))
	}
	target := ""
	if len(targets) > 0 {
		target = targets[0]
	}
	if target != "" {
		return fmt.Sprintf("tetra eco artifacts build --target %s --lock %s %s", target, compiler.SemanticLockFileName, filepath.ToSlash(capsulePath))
	}
	return fmt.Sprintf("tetra eco artifacts build --lock %s %s", compiler.SemanticLockFileName, filepath.ToSlash(capsulePath))
}

func writeArtifactIssues(w io.Writer, issues []artifactIssue, dryRun bool) {
	for _, issue := range issues {
		prefix := issue.Kind
		if dryRun && strings.HasPrefix(issue.Kind, "missing ") {
			prefix = "would generate " + strings.TrimPrefix(issue.Kind, "missing ")
		}
		if issue.Module != "" {
			fmt.Fprintf(w, "%s: %s (%s)", prefix, issue.Path, issue.Module)
		} else {
			fmt.Fprintf(w, "%s: %s", prefix, issue.Path)
		}
		if issue.Detail != "" {
			fmt.Fprintf(w, ": %s", issue.Detail)
		}
		if issue.Repair != "" {
			fmt.Fprintf(w, "\nrepair: %s", issue.Repair)
		}
		fmt.Fprintln(w)
	}
}

func dedupeArtifactIssues(issues []artifactIssue) []artifactIssue {
	seen := map[string]struct{}{}
	var out []artifactIssue
	for _, issue := range issues {
		key := issue.Kind + "\x00" + issue.Module + "\x00" + issue.Path + "\x00" + issue.Detail
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, issue)
	}
	return out
}

func dependencyArtifactBuildOptions(root string, manifest capsuleManifest, jobs int) (compiler.BuildOptions, error) {
	depRoots, _, err := projectDependencyGraph(root, manifest, map[string]int{root: projectDependencyVisiting}, []string{root})
	if err != nil {
		return compiler.BuildOptions{}, err
	}
	artifactRoots, err := projectArtifactInterfaceRoots(root, manifest.Artifacts)
	if err != nil {
		return compiler.BuildOptions{}, err
	}
	depRoots = append(depRoots, artifactRoots...)
	return compiler.BuildOptions{
		Jobs:            jobs,
		Emit:            compiler.EmitLibrary,
		ProjectRoot:     root,
		SourceRoots:     projectSourceRoots(manifest),
		DependencyRoots: depRoots,
	}, nil
}

func capsuleSourceModules(manifest capsuleManifest) ([]capsuleSourceModule, error) {
	root := filepath.Dir(manifest.Path)
	roots := projectSourceRoots(manifest)
	seenModules := map[string]string{}
	var modules []capsuleSourceModule
	for _, sourceRoot := range roots {
		dir := root
		if sourceRoot != "" {
			dir = filepath.Join(root, filepath.FromSlash(sourceRoot))
		}
		info, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				name := d.Name()
				if name == ".tetra" || name == ".tetra_cache" || name == "tetra_cache" {
					return filepath.SkipDir
				}
				return nil
			}
			if !compiler.IsSourceFile(path) {
				return nil
			}
			base := filepath.Base(path)
			if base == compiler.CapsuleFileName || base == compiler.LegacyCapsuleFileName {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			file, err := compiler.ParseFile(raw, path)
			if err != nil {
				return err
			}
			if file.Module == "" {
				return nil
			}
			if first, ok := seenModules[file.Module]; ok {
				return fmt.Errorf("%s: duplicate dependency module %s (first %s)", path, file.Module, first)
			}
			seenModules[file.Module] = path
			modules = append(modules, capsuleSourceModule{Module: file.Module, Path: path})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Module < modules[j].Module })
	return modules, nil
}

func interfaceArtifactRelPath(module string) string {
	return filepath.ToSlash(filepath.Join("interfaces", moduleArtifactRelPrefix(module)+compiler.T4InterfaceExtension))
}

func objectArtifactRelPath(module string, target string) string {
	return filepath.ToSlash(filepath.Join("artifacts", moduleArtifactRelPrefix(module)+"."+target+".tobj"))
}

func seedArtifactRelPath(name string) string {
	slug := artifactSlug(name)
	if slug == "" {
		slug = "capsule"
	}
	return filepath.ToSlash(filepath.Join("seeds", slug+"-deps"+compiler.T4SeedExtension))
}

func moduleArtifactRelPrefix(module string) string {
	return strings.ReplaceAll(module, ".", "/")
}

func artifactSlug(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, ch := range name {
		ok := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if ok {
			b.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func writeDependencySeed(path string, manifests []capsuleManifest) error {
	lock, err := buildEcoLockWithArtifactHashes(manifests)
	if err != nil {
		return err
	}
	seed := ecoSeed{
		Schema:        ecoSeedSchemaV1,
		GeneratedUnix: 0,
		Lock:          lock,
		Capsules:      make([]ecoSeedItem, 0, len(lock.Capsules)),
	}
	for _, capsule := range lock.Capsules {
		seed.Capsules = append(seed.Capsules, ecoSeedItem{
			ID:          capsule.ID,
			Name:        capsule.Name,
			Version:     capsule.Version,
			Targets:     append([]string(nil), capsule.Targets...),
			Effects:     append([]string(nil), capsule.Effects...),
			Permissions: append([]string(nil), capsule.Permissions...),
			DependsOn:   append([]capsuleDependency(nil), capsule.Dependencies...),
		})
	}
	return writeJSONFile(path, seed)
}

func appendGeneratedArtifactsToCapsule(path string, existing []capsuleArtifact, generated []generatedCapsuleArtifact) error {
	var missing []capsuleArtifact
	seen := map[string]struct{}{}
	for _, artifact := range existing {
		seen[capsuleArtifactKey(artifact)] = struct{}{}
	}
	for _, item := range generated {
		key := capsuleArtifactKey(item.Artifact)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		missing = append(missing, item.Artifact)
	}
	if len(missing) == 0 {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := strings.TrimRight(string(raw), "\n")
	var b strings.Builder
	b.WriteString(text)
	b.WriteString("\n\n    artifacts:\n")
	for _, artifact := range missing {
		b.WriteString("        ")
		b.WriteString(formatCapsuleArtifactLine(artifact))
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func capsuleArtifactKey(artifact capsuleArtifact) string {
	return artifact.Kind + "\x00" + artifact.Target + "\x00" + artifact.Path
}

func formatCapsuleArtifactLine(artifact capsuleArtifact) string {
	if artifact.Target != "" {
		return artifact.Kind + " " + artifact.Target + " " + artifact.Path
	}
	return artifact.Kind + " " + artifact.Path
}
