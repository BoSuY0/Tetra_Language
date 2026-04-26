package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	ctarget "tetra_language/compiler/target"
)

type capsuleManifest struct {
	Name         string
	ID           string
	Version      string
	Path         string
	Targets      []string
	Effects      []string
	Dependencies []capsuleDependency
}

type capsuleDependency struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type ecoLock struct {
	Capsules []ecoLockCapsule `json:"capsules"`
}

type ecoLockCapsule struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	Path         string              `json:"path"`
	Targets      []string            `json:"targets,omitempty"`
	Effects      []string            `json:"effects,omitempty"`
	Dependencies []capsuleDependency `json:"dependencies,omitempty"`
}

type ecoPackageMetadata struct {
	Schema      string                   `json:"schema"`
	Compression string                   `json:"compression"`
	MTimeUnix   int64                    `json:"mtime_unix"`
	FileCount   int                      `json:"file_count"`
	Files       []ecoPackageMetadataFile `json:"files"`
}

type ecoPackageMetadataFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type vaultRecord struct {
	Hash   string `json:"hash"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Size   int64  `json:"size"`
}

type vaultIndex struct {
	Records []vaultRecord `json:"records"`
}

var knownCapsuleEffects = map[string]string{
	"actors":     "actors",
	"alloc":      "alloc",
	"cap.io":     "io",
	"cap.mem":    "mem",
	"capability": "capability",
	"control":    "control",
	"io":         "io",
	"islands":    "islands",
	"link":       "link",
	"mem":        "mem",
	"mmio":       "mmio",
	"runtime":    "runtime",
}

func runEco(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra eco <verify|pack|unpack> [options]")
		return 2
	}
	switch args[0] {
	case "verify":
		return runEcoVerify(args[1:], stdout, stderr)
	case "pack":
		return runEcoPack(args[1:], stdout, stderr)
	case "unpack":
		return runEcoUnpack(args[1:], stdout, stderr)
	case "vault":
		return runEcoVault(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown eco command %q\n", args[0])
		return 2
	}
}

func runEcoVault(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra eco vault <add|list|verify> [options]")
		return 2
	}
	switch args[0] {
	case "add":
		return runEcoVaultAdd(args[1:], stdout, stderr)
	case "list":
		return runEcoVaultList(args[1:], stdout, stderr)
	case "verify":
		return runEcoVaultVerify(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown eco vault command %q\n", args[0])
		return 2
	}
}

func runEcoVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "validate capsule target compatibility")
	lockPath := fs.String("lock", "", "write dependency lock/provenance JSON")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"Tetra.capsule"}
	}
	manifests := make([]capsuleManifest, 0, len(paths))
	for _, path := range paths {
		manifest, err := parseCapsule(path)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		manifests = append(manifests, manifest)
	}
	if err := validateCapsuleGraph(manifests, *target); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *lockPath != "" {
		if err := writeEcoLock(*lockPath, manifests); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if len(manifests) == 1 {
		manifest := manifests[0]
		fmt.Fprintf(stdout, "Capsule OK: %s %s (%s)\n", manifest.Name, manifest.Version, manifest.ID)
		return 0
	}
	fmt.Fprintf(stdout, "Capsule graph OK: %d capsules\n", len(manifests))
	return 0
}

func runEcoPack(args []string, stdout io.Writer, stderr io.Writer) int {
	capsulePath, outPath, project, err := parseEcoPackArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	manifest, err := parseCapsule(capsulePath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if outPath == "" {
		outPath = manifest.Name + ".todex"
	}
	if project {
		if err := packCapsuleProject(manifest.Path, outPath); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else if err := packCapsule(manifest.Path, outPath); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Packed: %s\n", outPath)
	return 0
}

func runEcoUnpack(args []string, stdout io.Writer, stderr io.Writer) int {
	pkgPath, outDir, err := parseEcoUnpackArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if outDir == "" {
		outDir = "."
	}
	if err := unpackCapsule(pkgPath, outDir); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Unpacked: %s\n", outDir)
	return 0
}

func runEcoVaultAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco vault add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	store := fs.String("store", ".tetra/todex-vault", "local Todex vault directory")
	kind := fs.String("kind", "source", "record kind: source, interface, build, or test")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "eco vault add requires one file path")
		return 2
	}
	if !validVaultKind(*kind) {
		fmt.Fprintf(stderr, "unsupported vault kind %q\n", *kind)
		return 2
	}
	record, err := addVaultRecord(*store, fs.Arg(0), *kind)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Vault added: %s %s %s\n", record.Hash, record.Kind, record.Source)
	return 0
}

func runEcoVaultList(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco vault list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	store := fs.String("store", ".tetra/todex-vault", "local Todex vault directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eco vault list does not accept positional arguments")
		return 2
	}
	index, err := readVaultIndex(*store)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	sortVaultRecords(index.Records)
	for _, record := range index.Records {
		fmt.Fprintf(stdout, "%s %s %s %d\n", record.Hash, record.Kind, record.Source, record.Size)
	}
	return 0
}

func runEcoVaultVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco vault verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	store := fs.String("store", ".tetra/todex-vault", "local Todex vault directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eco vault verify does not accept positional arguments")
		return 2
	}
	index, err := readVaultIndex(*store)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	for _, record := range index.Records {
		if err := verifyVaultRecord(*store, record); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "Vault OK: %d records\n", len(index.Records))
	return 0
}

func parseCapsule(path string) (capsuleManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return capsuleManifest{}, err
	}
	manifest := capsuleManifest{Path: path}
	var (
		sawName    bool
		sawID      bool
		sawVersion bool
	)
	for i, line := range strings.Split(string(raw), "\n") {
		content := strings.TrimSpace(line)
		if content == "" || strings.HasPrefix(content, "//") || strings.HasPrefix(content, "#") {
			continue
		}
		if strings.HasPrefix(content, "capsule ") {
			if sawName {
				return capsuleManifest{}, fmt.Errorf("%s:%d: duplicate capsule declaration", path, i+1)
			}
			name := strings.TrimSpace(strings.TrimPrefix(content, "capsule "))
			name = strings.TrimSuffix(name, ":")
			if name == "" {
				return capsuleManifest{}, fmt.Errorf("%s:%d: capsule name is required", path, i+1)
			}
			manifest.Name = name
			sawName = true
			continue
		}
		if strings.HasPrefix(content, "id ") {
			if sawID {
				return capsuleManifest{}, fmt.Errorf("%s:%d: duplicate id field", path, i+1)
			}
			value, err := parseCapsuleString(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "id ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			if !strings.HasPrefix(value, "tetra://") {
				return capsuleManifest{}, fmt.Errorf("%s:%d: capsule id must use tetra:// prefix", path, i+1)
			}
			manifest.ID = value
			sawID = true
			continue
		}
		if strings.HasPrefix(content, "version ") {
			if sawVersion {
				return capsuleManifest{}, fmt.Errorf("%s:%d: duplicate version field", path, i+1)
			}
			value, err := parseCapsuleString(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "version ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			if !isCapsuleSemver(value) {
				return capsuleManifest{}, fmt.Errorf("%s:%d: capsule version must use semver x.y.z", path, i+1)
			}
			manifest.Version = value
			sawVersion = true
			continue
		}
		if strings.HasPrefix(content, "target ") {
			value, err := parseCapsuleString(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "target ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			if !isSupportedCapsuleTarget(value) {
				return capsuleManifest{}, fmt.Errorf("%s:%d: unsupported target %s", path, i+1, value)
			}
			if containsString(manifest.Targets, value) {
				return capsuleManifest{}, fmt.Errorf("%s:%d: duplicate target %s", path, i+1, value)
			}
			manifest.Targets = append(manifest.Targets, value)
			continue
		}
		if strings.HasPrefix(content, "effect ") {
			value, err := parseCapsuleString(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "effect ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			normalized, err := normalizeCapsuleEffect(value)
			if err != nil {
				return capsuleManifest{}, fmt.Errorf("%s:%d: %v", path, i+1, err)
			}
			if containsString(manifest.Effects, normalized) {
				return capsuleManifest{}, fmt.Errorf("%s:%d: duplicate effect %s", path, i+1, normalized)
			}
			manifest.Effects = append(manifest.Effects, normalized)
			continue
		}
		if strings.HasPrefix(content, "dependency ") {
			dep, err := parseCapsuleDependency(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "dependency ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			manifest.Dependencies = append(manifest.Dependencies, dep)
			continue
		}
		return capsuleManifest{}, fmt.Errorf("%s:%d: unknown capsule field", path, i+1)
	}
	if manifest.Name == "" {
		return capsuleManifest{}, fmt.Errorf("%s: missing capsule declaration", path)
	}
	if manifest.ID == "" {
		return capsuleManifest{}, fmt.Errorf("%s: missing capsule id", path)
	}
	if manifest.Version == "" {
		return capsuleManifest{}, fmt.Errorf("%s: missing capsule version", path)
	}
	return manifest, nil
}

func parseCapsuleDependency(path string, line int, value string) (capsuleDependency, error) {
	fields, err := splitQuotedFields(value)
	if err != nil {
		return capsuleDependency{}, fmt.Errorf("%s:%d: %v", path, line, err)
	}
	if len(fields) != 2 {
		return capsuleDependency{}, fmt.Errorf("%s:%d: dependency expects quoted id and version", path, line)
	}
	if !strings.HasPrefix(fields[0], "tetra://") {
		return capsuleDependency{}, fmt.Errorf("%s:%d: dependency id must use tetra:// prefix", path, line)
	}
	if !isCapsuleSemver(fields[1]) {
		return capsuleDependency{}, fmt.Errorf("%s:%d: dependency version must use semver x.y.z", path, line)
	}
	return capsuleDependency{ID: fields[0], Version: fields[1]}, nil
}

func splitQuotedFields(value string) ([]string, error) {
	var out []string
	rest := strings.TrimSpace(value)
	for rest != "" {
		if !strings.HasPrefix(rest, "\"") {
			return nil, fmt.Errorf("expected quoted string")
		}
		end := 1
		escaped := false
		for ; end < len(rest); end++ {
			ch := rest[end]
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				break
			}
		}
		if end >= len(rest) {
			return nil, fmt.Errorf("unterminated quoted string")
		}
		parsed, err := strconv.Unquote(rest[:end+1])
		if err != nil {
			return nil, fmt.Errorf("expected quoted string")
		}
		if parsed == "" {
			return nil, fmt.Errorf("string must not be empty")
		}
		out = append(out, parsed)
		rest = strings.TrimSpace(rest[end+1:])
	}
	return out, nil
}

func validateCapsuleGraph(manifests []capsuleManifest, target string) error {
	byID := make(map[string]capsuleManifest, len(manifests))
	for _, manifest := range manifests {
		if _, exists := byID[manifest.ID]; exists {
			return fmt.Errorf("duplicate capsule id %q", manifest.ID)
		}
		if target != "" && len(manifest.Targets) > 0 && !containsString(manifest.Targets, target) {
			return fmt.Errorf("%s: target mismatch for %s: does not support %s", manifest.Path, manifest.ID, target)
		}
		seenEffects := map[string]struct{}{}
		for _, effect := range manifest.Effects {
			if _, exists := seenEffects[effect]; exists {
				return fmt.Errorf("%s: duplicate effect %s", manifest.Path, effect)
			}
			seenEffects[effect] = struct{}{}
		}
		seenDeps := map[string]struct{}{}
		for _, dep := range manifest.Dependencies {
			key := dep.ID + "\x00" + dep.Version
			if _, exists := seenDeps[key]; exists {
				return fmt.Errorf("%s: duplicate dependency %s %s", manifest.Path, dep.ID, dep.Version)
			}
			seenDeps[key] = struct{}{}
		}
		byID[manifest.ID] = manifest
	}
	for _, manifest := range manifests {
		for _, dep := range manifest.Dependencies {
			found, ok := byID[dep.ID]
			if !ok {
				return fmt.Errorf("%s: missing dependency %s %s", manifest.Path, dep.ID, dep.Version)
			}
			if found.Version != dep.Version {
				return fmt.Errorf("%s: dependency %s version mismatch: want %s, got %s", manifest.Path, dep.ID, dep.Version, found.Version)
			}
			for _, effect := range found.Effects {
				if !containsString(manifest.Effects, effect) {
					return fmt.Errorf("%s: missing required effect %s for dependency %s", manifest.Path, effect, dep.ID)
				}
			}
		}
	}
	return nil
}

func writeEcoLock(path string, manifests []capsuleManifest) error {
	items := make([]ecoLockCapsule, 0, len(manifests))
	for _, manifest := range manifests {
		item := ecoLockCapsule{
			ID:           manifest.ID,
			Name:         manifest.Name,
			Version:      manifest.Version,
			Path:         filepath.Clean(manifest.Path),
			Targets:      sortedStrings(manifest.Targets),
			Effects:      sortedStrings(manifest.Effects),
			Dependencies: append([]capsuleDependency(nil), manifest.Dependencies...),
		}
		sort.Slice(item.Dependencies, func(i, j int) bool {
			if item.Dependencies[i].ID == item.Dependencies[j].ID {
				return item.Dependencies[i].Version < item.Dependencies[j].Version
			}
			return item.Dependencies[i].ID < item.Dependencies[j].ID
		})
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	raw, err := json.MarshalIndent(ecoLock{Capsules: items}, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func validVaultKind(kind string) bool {
	switch kind {
	case "source", "interface", "build", "test":
		return true
	default:
		return false
	}
}

func addVaultRecord(store string, path string, kind string) (vaultRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return vaultRecord{}, err
	}
	sum := sha256.Sum256(raw)
	hashHex := fmt.Sprintf("%x", sum[:])
	objectPath := vaultObjectPath(store, hashHex)
	if err := os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
		return vaultRecord{}, err
	}
	if _, err := os.Stat(objectPath); err != nil {
		if !os.IsNotExist(err) {
			return vaultRecord{}, err
		}
		if err := os.WriteFile(objectPath, raw, 0o644); err != nil {
			return vaultRecord{}, err
		}
	}
	record := vaultRecord{
		Hash:   "sha256:" + hashHex,
		Kind:   kind,
		Source: filepath.Clean(path),
		Size:   int64(len(raw)),
	}
	index, err := readVaultIndex(store)
	if err != nil {
		return vaultRecord{}, err
	}
	index.Records = upsertVaultRecord(index.Records, record)
	if err := writeVaultIndex(store, index); err != nil {
		return vaultRecord{}, err
	}
	return record, nil
}

func upsertVaultRecord(records []vaultRecord, record vaultRecord) []vaultRecord {
	for i, existing := range records {
		if existing.Hash == record.Hash && existing.Kind == record.Kind && existing.Source == record.Source {
			records[i] = record
			sortVaultRecords(records)
			return records
		}
	}
	records = append(records, record)
	sortVaultRecords(records)
	return records
}

func readVaultIndex(store string) (vaultIndex, error) {
	path := vaultIndexPath(store)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return vaultIndex{}, nil
		}
		return vaultIndex{}, err
	}
	var index vaultIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		return vaultIndex{}, err
	}
	sortVaultRecords(index.Records)
	return index, nil
}

func writeVaultIndex(store string, index vaultIndex) error {
	sortVaultRecords(index.Records)
	raw, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	path := vaultIndexPath(store)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func verifyVaultRecord(store string, record vaultRecord) error {
	const prefix = "sha256:"
	if !strings.HasPrefix(record.Hash, prefix) {
		return fmt.Errorf("vault record %s has unsupported hash", record.Source)
	}
	hashHex := strings.TrimPrefix(record.Hash, prefix)
	raw, err := os.ReadFile(vaultObjectPath(store, hashHex))
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	actual := fmt.Sprintf("%x", sum[:])
	if actual != hashHex {
		return fmt.Errorf("vault object mismatch for %s", record.Source)
	}
	if int64(len(raw)) != record.Size {
		return fmt.Errorf("vault object size mismatch for %s", record.Source)
	}
	return nil
}

func vaultIndexPath(store string) string {
	return filepath.Join(store, "records.json")
}

func vaultObjectPath(store string, hashHex string) string {
	return filepath.Join(store, "objects", "sha256", hashHex)
}

func sortVaultRecords(records []vaultRecord) {
	sort.Slice(records, func(i, j int) bool {
		if records[i].Hash == records[j].Hash {
			if records[i].Kind == records[j].Kind {
				return records[i].Source < records[j].Source
			}
			return records[i].Kind < records[j].Kind
		}
		return records[i].Hash < records[j].Hash
	})
}

func appendUniqueString(values []string, value string) []string {
	if containsString(values, value) {
		return values
	}
	return append(values, value)
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func normalizeCapsuleEffect(name string) (string, error) {
	normalized, ok := knownCapsuleEffects[name]
	if !ok {
		return "", fmt.Errorf("unknown effect %q", name)
	}
	return normalized, nil
}

func isCapsuleSemver(version string) bool {
	if version == "" {
		return false
	}
	main := version
	if idx := strings.IndexAny(version, "-+"); idx >= 0 {
		main = version[:idx]
	}
	parts := strings.Split(main, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

func isSupportedCapsuleTarget(target string) bool {
	for _, triple := range ctarget.SupportedTriples() {
		if triple == target {
			return true
		}
	}
	return false
}

func parseCapsuleString(path string, line int, value string) (string, error) {
	out, err := strconv.Unquote(value)
	if err != nil {
		return "", fmt.Errorf("%s:%d: expected quoted string", path, line)
	}
	if out == "" {
		return "", fmt.Errorf("%s:%d: string must not be empty", path, line)
	}
	return out, nil
}

func parseEcoPackArgs(args []string) (capsulePath string, outPath string, project bool, err error) {
	capsulePath = "Tetra.capsule"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project":
			project = true
		case "-o", "--out":
			i++
			if i >= len(args) {
				return "", "", false, fmt.Errorf("%s requires a value", args[i-1])
			}
			outPath = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", false, fmt.Errorf("unknown option %s", args[i])
			}
			if capsulePath != "Tetra.capsule" {
				return "", "", false, fmt.Errorf("eco pack accepts one capsule path")
			}
			capsulePath = args[i]
		}
	}
	return capsulePath, outPath, project, nil
}

func parseEcoUnpackArgs(args []string) (pkgPath string, outDir string, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-C", "--dir":
			i++
			if i >= len(args) {
				return "", "", fmt.Errorf("%s requires a value", args[i-1])
			}
			outDir = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", fmt.Errorf("unknown option %s", args[i])
			}
			if pkgPath != "" {
				return "", "", fmt.Errorf("eco unpack accepts one package path")
			}
			pkgPath = args[i]
		}
	}
	if pkgPath == "" {
		return "", "", fmt.Errorf("eco unpack requires a package path")
	}
	return pkgPath, outDir, nil
}

func packCapsule(capsulePath string, outPath string) error {
	return packFiles(filepath.Dir(capsulePath), []string{filepath.Base(capsulePath)}, outPath)
}

func packCapsuleProject(capsulePath string, outPath string) error {
	root := filepath.Dir(capsulePath)
	var relPaths []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".tetra_cache" || name == "tetra_cache" {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if filepath.Clean(path) == filepath.Clean(outPath) || strings.HasSuffix(rel, ".todex") {
			return nil
		}
		relPaths = append(relPaths, rel)
		return nil
	}); err != nil {
		return err
	}
	sort.Strings(relPaths)
	return packFiles(root, relPaths, outPath)
}

func packFiles(root string, relPaths []string, outPath string) error {
	const packageMetadataFile = "tetra.package.json"
	zeroTime := time.Unix(0, 0).UTC()
	cleanRelPaths := append([]string(nil), relPaths...)
	sort.Strings(cleanRelPaths)
	for _, rel := range cleanRelPaths {
		if filepath.ToSlash(rel) == packageMetadataFile {
			return fmt.Errorf("project already contains reserved file %s", packageMetadataFile)
		}
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()
	gz := gzip.NewWriter(out)
	gz.ModTime = zeroTime
	gz.OS = 255
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	metadata := ecoPackageMetadata{
		Schema:      "tetra.eco.package.v1",
		Compression: "gzip",
		MTimeUnix:   0,
		Files:       make([]ecoPackageMetadataFile, 0, len(cleanRelPaths)),
	}
	for _, rel := range cleanRelPaths {
		if rel == "" || strings.HasPrefix(filepath.Clean(rel), "..") || filepath.IsAbs(rel) {
			return fmt.Errorf("unsafe archive path %q", rel)
		}
		path := filepath.Join(root, rel)
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		info, err := in.Stat()
		if err != nil {
			_ = in.Close()
			return err
		}
		header := &tar.Header{
			Name:       filepath.ToSlash(rel),
			Mode:       0o644,
			Size:       info.Size(),
			Format:     tar.FormatPAX,
			ModTime:    zeroTime,
			AccessTime: zeroTime,
			ChangeTime: zeroTime,
		}
		if err := tw.WriteHeader(header); err != nil {
			_ = in.Close()
			return err
		}
		hash := sha256.New()
		if _, err := io.Copy(tw, io.TeeReader(in, hash)); err != nil {
			_ = in.Close()
			return err
		}
		if err := in.Close(); err != nil {
			return err
		}
		metadata.Files = append(metadata.Files, ecoPackageMetadataFile{
			Path:   filepath.ToSlash(rel),
			SHA256: "sha256:" + hex.EncodeToString(hash.Sum(nil)),
			Size:   info.Size(),
		})
	}
	metadata.FileCount = len(metadata.Files)
	rawMetadata, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	rawMetadata = append(rawMetadata, '\n')
	if err := tw.WriteHeader(&tar.Header{
		Name:       packageMetadataFile,
		Mode:       0o644,
		Size:       int64(len(rawMetadata)),
		Format:     tar.FormatPAX,
		ModTime:    zeroTime,
		AccessTime: zeroTime,
		ChangeTime: zeroTime,
	}); err != nil {
		return err
	}
	if _, err := tw.Write(rawMetadata); err != nil {
		return err
	}
	return nil
}

func unpackCapsule(pkgPath string, outDir string) error {
	in, err := os.Open(pkgPath)
	if err != nil {
		return err
	}
	defer in.Close()
	gz, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(header.Name)
		if name == "." || strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			return fmt.Errorf("unsafe archive path %q", header.Name)
		}
		outPath := filepath.Join(outDir, name)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
}
