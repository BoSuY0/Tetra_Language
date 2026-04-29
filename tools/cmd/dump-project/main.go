package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type dumpOptions struct {
	root            string
	outputPath      string
	maxFileBytes    int64
	fileListPath    string
	useGit          bool
	includeDumps    bool
	includeIgnored  bool
	includeDotenv   bool
	onlyRelPrefixes []string
	excludePrefixes []string
	writeSummary    bool
}

const dumpArtifact = "tetra.release.v0_2_0.project-dump.v1"

type multiValue []string

func (m *multiValue) String() string {
	return strings.Join(*m, ",")
}

func (m *multiValue) Set(value string) error {
	*m = append(*m, value)
	return nil
}

var textExtensions = map[string]struct{}{
	".py":                     {},
	".pyi":                    {},
	".md":                     {},
	".txt":                    {},
	".toml":                   {},
	".json":                   {},
	".yaml":                   {},
	".yml":                    {},
	".ini":                    {},
	".cfg":                    {},
	".conf":                   {},
	".env":                    {},
	".sh":                     {},
	".bash":                   {},
	".zsh":                    {},
	".ps1":                    {},
	".bat":                    {},
	".cmd":                    {},
	".sql":                    {},
	".graphql":                {},
	".gql":                    {},
	".ts":                     {},
	".tsx":                    {},
	".js":                     {},
	".mjs":                    {},
	".cjs":                    {},
	".css":                    {},
	".html":                   {},
	".htm":                    {},
	".xml":                    {},
	".csv":                    {},
	".gitignore":              {},
	".gitattributes":          {},
	".editorconfig":           {},
	".pre-commit-config.yaml": {},
	".lock":                   {},
}

var binaryExtensions = map[string]struct{}{
	".png":    {},
	".jpg":    {},
	".jpeg":   {},
	".gif":    {},
	".webp":   {},
	".ico":    {},
	".pdf":    {},
	".zip":    {},
	".gz":     {},
	".bz2":    {},
	".xz":     {},
	".7z":     {},
	".rar":    {},
	".tar":    {},
	".tgz":    {},
	".mp3":    {},
	".mp4":    {},
	".mov":    {},
	".mkv":    {},
	".avi":    {},
	".wav":    {},
	".flac":   {},
	".sqlite": {},
	".db":     {},
	".bin":    {},
	".so":     {},
	".dylib":  {},
	".exe":    {},
	".dll":    {},
	".woff":   {},
	".woff2":  {},
	".ttf":    {},
	".otf":    {},
	".eot":    {},
}

func main() {
	rootDefault := defaultProjectRoot()

	var (
		rootFlag         = flag.String("root", "", "Root directory to dump (default: project root)")
		outFlag          = flag.String("out", "", "Output file path (default: dumps/<name>_dump_<timestamp>.txt)")
		fileListFlag     = flag.String("file-list", "", "Text file with rel paths to include (one per line)")
		maxFileBytesFlag = flag.Int64("max-file-bytes", 1_000_000, "Max file size to include")
		noGitFlag        = flag.Bool("no-git", false, "Do not use git ls-files; scan filesystem instead")
		allFlag          = flag.Bool("all", false, "Include all files (disables default --only prefixes)")
		includeDumpsFlag = flag.Bool("include-dumps", false, "Include dumps/ directory contents")
		includeIgnored   = flag.Bool("include-ignored", false, "Include git-ignored files")
		includeDotenv    = flag.Bool("include-dotenv", false, "Include .env and .env.* files")
		noSummary        = flag.Bool("no-summary", false, "Do not write _summary.txt")
	)
	var onlyArgs multiValue
	flag.Var(&onlyArgs, "only", "Include only paths under this prefix (repeatable)")
	var excludeArgs multiValue
	flag.Var(&excludeArgs, "exclude-prefix", "Exclude paths under this prefix (repeatable)")
	flag.Parse()

	root := rootDefault
	if *rootFlag != "" {
		root = resolvePath(rootDefault, *rootFlag)
	}

	outputPath := ""
	if *outFlag == "" {
		outputPath = defaultOutputPath(root)
	} else {
		outputPath = resolvePath(root, *outFlag)
	}

	fileListPath := ""
	if *fileListFlag != "" {
		fileListPath = resolvePath(root, *fileListFlag)
	}

	onlyPrefixes := normalizePrefixes(root, onlyArgs)
	if fileListPath == "" && len(onlyPrefixes) == 0 && !*allFlag {
		onlyPrefixes = defaultOnlyPrefixes()
	}
	excludePrefixes := normalizePrefixes(root, excludeArgs)
	excludePrefixes = append(excludePrefixes, defaultExcludePrefixes(*includeDumpsFlag)...)
	excludePrefixes = uniqueSorted(excludePrefixes)

	opts := dumpOptions{
		root:            root,
		outputPath:      outputPath,
		maxFileBytes:    *maxFileBytesFlag,
		fileListPath:    fileListPath,
		useGit:          !*noGitFlag,
		includeDumps:    *includeDumpsFlag,
		includeIgnored:  *includeIgnored,
		includeDotenv:   *includeDotenv,
		onlyRelPrefixes: onlyPrefixes,
		excludePrefixes: excludePrefixes,
		writeSummary:    !*noSummary,
	}

	included, skippedBinary, skippedLarge, err := buildDump(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dump failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Dump created: %s\n", opts.outputPath)
	fmt.Printf("Included: %d; skipped (binary): %d; skipped (too large): %d\n", included, skippedBinary, skippedLarge)
}

func defaultProjectRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		scriptDir := filepath.Dir(filename)
		if root := findProjectRoot(scriptDir); root != "" {
			return root
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if root := findProjectRoot(cwd); root != "" {
		return root
	}
	return cwd
}

func findProjectRoot(start string) string {
	dir := filepath.Clean(start)
	for {
		if fileExists(filepath.Join(dir, "go.work")) {
			return dir
		}
		if fileExists(filepath.Join(dir, "README.md")) && fileExists(filepath.Join(dir, "compiler", "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

func resolvePath(root, raw string) string {
	if raw == "" {
		return root
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw)
	}
	return filepath.Clean(filepath.Join(root, raw))
}

func normalizePrefixes(root string, values multiValue) []string {
	var out []string
	for _, raw := range values {
		if raw == "" {
			continue
		}
		if filepath.IsAbs(raw) {
			rel, ok := relToSlash(root, raw)
			if !ok {
				fmt.Fprintf(os.Stderr, "Skipped prefix outside root: %s\n", raw)
				continue
			}
			prefix := normalizeRel(rel)
			if prefix != "" {
				out = append(out, prefix)
			}
			continue
		}
		prefix := normalizeRel(filepath.ToSlash(raw))
		if prefix != "" {
			out = append(out, prefix)
		}
	}
	return out
}

func normalizeRel(rel string) string {
	cleaned := path.Clean(rel)
	cleaned = strings.TrimPrefix(cleaned, "./")
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func defaultOnlyPrefixes() []string {
	return []string{
		"cli",
		"compiler",
		"tools",
		"docs",
		"examples",
		"scripts",
		"README.md",
		"go.work",
		".gitignore",
	}
}

func defaultExcludePrefixes(includeDumps bool) []string {
	exclude := []string{
		".cache",
		".gocache",
		".tetra_cache",
		"tetra_cache",
		"examples/.tetra_cache",
		"bin",
		"_legacy",
		"dist",
		"out",
	}
	if !includeDumps {
		exclude = append(exclude, "dumps")
	}
	return exclude
}

func uniqueSorted(in []string) []string {
	m := make(map[string]struct{}, len(in))
	for _, v := range in {
		if v == "" {
			continue
		}
		m[v] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

func defaultOutputPath(root string) string {
	ts := time.Now().UTC().Format("20060102_150405Z")
	name := sanitizeName(filepath.Base(root))
	if name == "" {
		name = "project"
	}
	return filepath.Join(root, "dumps", fmt.Sprintf("%s_dump_%s.txt", name, ts))
}

func sanitizeName(name string) string {
	if name == "" {
		return ""
	}
	if !isASCII(name) {
		return "project"
	}
	lower := strings.ToLower(name)
	lower = strings.ReplaceAll(lower, " ", "_")
	return lower
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

func buildDump(opts dumpOptions) (int, int, int, error) {
	if err := os.MkdirAll(filepath.Dir(opts.outputPath), 0o755); err != nil {
		return 0, 0, 0, err
	}

	excludeRel := determineExcludes(opts.root, opts.outputPath)
	relPaths := collectRelPaths(opts, excludeRel)
	gitHead := gitHead(opts.root)
	now := time.Now().UTC().Format(time.RFC3339)

	outFile, err := os.Create(opts.outputPath)
	if err != nil {
		return 0, 0, 0, err
	}
	defer outFile.Close()
	writer := bufio.NewWriter(outFile)

	writeDumpHeader(writer, opts, now, gitHead, relPaths)

	included := 0
	skippedBinary := 0
	skippedLarge := 0
	for _, rel := range relPaths {
		inc, skipBin, skipLarge := dumpOneFile(writer, opts.root, rel, opts.maxFileBytes)
		included += inc
		skippedBinary += skipBin
		skippedLarge += skipLarge
	}
	if err := writer.Flush(); err != nil {
		return included, skippedBinary, skippedLarge, err
	}

	if opts.writeSummary {
		summaryPath := summaryPathFor(opts.outputPath)
		summaryText := fmt.Sprintf(
			"Artifact: %s\nIncluded files: %d\nSkipped (binary): %d\nSkipped (too large): %d\n",
			dumpArtifact,
			included,
			skippedBinary,
			skippedLarge,
		)
		_ = os.WriteFile(summaryPath, []byte(summaryText), 0o644)
	}

	return included, skippedBinary, skippedLarge, nil
}

func determineExcludes(root, outputPath string) map[string]struct{} {
	exclude := make(map[string]struct{})
	rel, ok := relToSlash(root, outputPath)
	if ok {
		exclude[rel] = struct{}{}
	}
	return exclude
}

func collectRelPaths(opts dumpOptions, excludeRel map[string]struct{}) []string {
	if opts.fileListPath != "" {
		relPaths := collectFromFileList(opts, excludeRel)
		return filterRelPaths(opts, relPaths)
	}

	paths := make(map[string]struct{})
	if opts.useGit {
		for _, rel := range gitLsFiles(opts.root, []string{}) {
			paths[rel] = struct{}{}
		}
		for _, rel := range gitLsFiles(opts.root, []string{"--others", "--exclude-standard"}) {
			paths[rel] = struct{}{}
		}
		if opts.includeIgnored {
			for _, rel := range gitLsFiles(opts.root, []string{"--others", "-i", "--exclude-standard"}) {
				paths[rel] = struct{}{}
			}
		}
	}

	if len(paths) == 0 {
		for _, rel := range rglobFiles(opts.root, opts.includeDumps, excludeRel) {
			paths[rel] = struct{}{}
		}
	}

	if opts.includeDotenv {
		ensureDotenv(opts.root, paths)
	} else {
		delete(paths, ".env")
		for rel := range paths {
			if strings.HasPrefix(rel, ".env.") {
				delete(paths, rel)
			}
		}
	}

	relPaths := setToSortedSlice(paths, excludeRel)
	return filterRelPaths(opts, relPaths)
}

func collectFromFileList(opts dumpOptions, excludeRel map[string]struct{}) []string {
	data, err := os.ReadFile(opts.fileListPath)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	var relPaths []string
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var rel string
		if filepath.IsAbs(line) {
			relPath, ok := relToSlash(opts.root, line)
			if !ok {
				continue
			}
			rel = normalizeRel(relPath)
		} else {
			rel = normalizeRel(filepath.ToSlash(line))
		}
		if rel == "" {
			continue
		}
		if _, skip := excludeRel[rel]; skip {
			continue
		}
		absPath := filepath.Join(opts.root, filepath.FromSlash(rel))
		info, err := os.Stat(absPath)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		relPaths = append(relPaths, rel)
	}

	sort.Slice(relPaths, func(i, j int) bool {
		return strings.ToLower(relPaths[i]) < strings.ToLower(relPaths[j])
	})
	return relPaths
}

func gitLsFiles(root string, extra []string) []string {
	args := []string{"-C", root, "ls-files"}
	args = append(args, extra...)
	args = append(args, "-z")

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	parts := bytes.Split(out, []byte{0})
	var paths []string
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		if !utf8.Valid(part) {
			continue
		}
		rel := normalizeRel(filepath.ToSlash(string(part)))
		if rel == "" {
			continue
		}
		paths = append(paths, rel)
	}
	return paths
}

func rglobFiles(root string, includeDumps bool, excludeRel map[string]struct{}) []string {
	skipDirs := map[string]struct{}{
		".git":          {},
		".cache":        {},
		".gocache":      {},
		".tetra_cache":  {},
		".venv":         {},
		"_legacy":       {},
		"__pycache__":   {},
		"bin":           {},
		"dist":          {},
		"node_modules":  {},
		"out":           {},
		"tetra_cache":   {},
		".pytest_cache": {},
		".mypy_cache":   {},
		".hypothesis":   {},
		".ruff_cache":   {},
		".tmp":          {},
	}
	if !includeDumps {
		skipDirs["dumps"] = struct{}{}
	}

	var relPaths []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, ok := relToSlash(root, path)
		if !ok {
			return nil
		}
		if rel == "" {
			return nil
		}
		parts := strings.Split(rel, "/")
		for _, part := range parts {
			if _, skip := skipDirs[part]; skip {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}
		if d.IsDir() {
			return nil
		}
		if _, skip := excludeRel[rel]; skip {
			return nil
		}
		relPaths = append(relPaths, rel)
		return nil
	})

	return relPaths
}

func setToSortedSlice(paths map[string]struct{}, excludeRel map[string]struct{}) []string {
	var relPaths []string
	for rel := range paths {
		if _, skip := excludeRel[rel]; skip {
			continue
		}
		relPaths = append(relPaths, rel)
	}

	sort.Slice(relPaths, func(i, j int) bool {
		return strings.ToLower(relPaths[i]) < strings.ToLower(relPaths[j])
	})

	return relPaths
}

func filterRelPaths(opts dumpOptions, relPaths []string) []string {
	out := filterExcludedDirParts(relPaths, defaultExcludedDirParts(opts.includeDumps))
	if len(opts.excludePrefixes) > 0 {
		out = filterExcludedPrefixes(out, opts.excludePrefixes)
	}
	if len(opts.onlyRelPrefixes) > 0 {
		out = filterPrefixes(out, opts.onlyRelPrefixes)
	}
	return out
}

func defaultExcludedDirParts(includeDumps bool) map[string]struct{} {
	exclude := map[string]struct{}{
		".cache":       {},
		".gocache":     {},
		".tetra_cache": {},
		"tetra_cache":  {},
		"bin":          {},
		"_legacy":      {},
		"dist":         {},
		"out":          {},
	}
	if !includeDumps {
		exclude["dumps"] = struct{}{}
	}
	return exclude
}

func filterExcludedDirParts(relPaths []string, excluded map[string]struct{}) []string {
	var out []string
	for _, rel := range relPaths {
		parts := strings.Split(rel, "/")
		skip := false
		for _, part := range parts {
			if _, ok := excluded[part]; ok {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		out = append(out, rel)
	}
	return out
}

func filterExcludedPrefixes(relPaths []string, prefixes []string) []string {
	var out []string
	for _, rel := range relPaths {
		if hasAnyPrefix(rel, prefixes) {
			continue
		}
		out = append(out, rel)
	}
	return out
}

func filterPrefixes(relPaths []string, prefixes []string) []string {
	var out []string
	for _, rel := range relPaths {
		if hasAnyPrefix(rel, prefixes) {
			out = append(out, rel)
		}
	}
	return out
}

func hasAnyPrefix(rel string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
			return true
		}
	}
	return false
}

func ensureDotenv(root string, paths map[string]struct{}) {
	envPath := filepath.Join(root, ".env")
	if info, err := os.Stat(envPath); err == nil && info.Mode().IsRegular() {
		paths[".env"] = struct{}{}
	}
	matches, err := filepath.Glob(filepath.Join(root, ".env.*"))
	if err != nil {
		return
	}
	for _, match := range matches {
		if info, err := os.Stat(match); err == nil && info.Mode().IsRegular() {
			rel, ok := relToSlash(root, match)
			if ok {
				paths[rel] = struct{}{}
			}
		}
	}
}

func gitHead(root string) string {
	cmd := exec.Command("git", "-C", root, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func writeDumpHeader(w *bufio.Writer, opts dumpOptions, now, gitHead string, relPaths []string) {
	w.WriteString("Project dump\n")
	w.WriteString("Warning: dump may contain secrets.\n")
	w.WriteString(fmt.Sprintf("Generated: %s\n", now))
	if gitHead != "" {
		w.WriteString(fmt.Sprintf("Git HEAD: %s\n", gitHead))
	}
	w.WriteString(fmt.Sprintf("Root: %s\n", opts.root))
	w.WriteString(fmt.Sprintf("Artifact schema: %s\n", dumpArtifact))
	w.WriteString(fmt.Sprintf("Max file bytes: %d\n", opts.maxFileBytes))
	w.WriteString(fmt.Sprintf("Files listed: %d\n", len(relPaths)))
	w.WriteString(fmt.Sprintf("Include dumps/: %v\n", boolToYesNo(opts.includeDumps)))
	w.WriteString(fmt.Sprintf("Include ignored: %v\n", boolToYesNo(opts.includeIgnored)))
	w.WriteString(fmt.Sprintf("Include .env*: %v\n", boolToYesNo(opts.includeDotenv)))
	if len(opts.onlyRelPrefixes) > 0 {
		w.WriteString(fmt.Sprintf("Only prefixes: %s\n", strings.Join(opts.onlyRelPrefixes, ", ")))
	}
	if len(opts.excludePrefixes) > 0 {
		w.WriteString(fmt.Sprintf("Excluded prefixes: %s\n", strings.Join(opts.excludePrefixes, ", ")))
	}
	w.WriteString("Note: non-UTF8 text is decoded with replacement.\n")
	w.WriteString("\n")
}

func boolToYesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func dumpOneFile(w *bufio.Writer, root, rel string, maxFileBytes int64) (int, int, int) {
	absPath := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(absPath)
	if err != nil {
		return 0, 0, 0
	}

	included := 0
	skippedBinary := 0
	skippedLarge := 0
	status := "OK"
	sha := "-"
	var text []byte

	if info.Size() > maxFileBytes {
		skippedLarge = 1
		status = "SKIP_TOO_LARGE"
	} else {
		data, err := os.ReadFile(absPath)
		if err != nil {
			status = "ERROR_READ"
		} else {
			sha = sha256Hex(data)
			sample := data
			if len(sample) > 8192 {
				sample = data[:8192]
			}
			if isKnownBinaryPath(rel) || (!forceTextByExtension(rel) && isBinary(sample)) {
				skippedBinary = 1
				status = "SKIP_BINARY"
			} else {
				included = 1
				text = bytes.ToValidUTF8(data, []byte("?"))
			}
		}
	}

	writeSeparator(w, '=')
	w.WriteString(fmt.Sprintf("FILE: %s\n", rel))
	w.WriteString(fmt.Sprintf("SIZE: %d\n", info.Size()))
	w.WriteString(fmt.Sprintf("SHA256: %s\n", sha))
	w.WriteString(fmt.Sprintf("STATUS: %s\n", status))
	writeSeparator(w, '-')
	if text == nil {
		w.WriteString("[content omitted]\n")
	} else {
		w.Write(text)
		if !bytes.HasSuffix(text, []byte("\n")) {
			w.WriteString("\n")
		}
	}

	return included, skippedBinary, skippedLarge
}

func writeSeparator(w *bufio.Writer, ch byte) {
	line := bytes.Repeat([]byte{ch}, 88)
	w.Write(line)
	w.WriteString("\n")
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func isKnownBinaryPath(rel string) bool {
	name := filepath.Base(rel)
	suffixes := splitSuffixes(name)
	if len(suffixes) == 0 {
		return false
	}
	full := strings.Join(suffixes, "")
	if full == ".tar.gz" || full == ".tar.bz2" || full == ".tar.xz" {
		return true
	}
	if _, ok := binaryExtensions[suffixes[len(suffixes)-1]]; ok {
		return true
	}
	return false
}

func forceTextByExtension(rel string) bool {
	name := strings.ToLower(filepath.Base(rel))
	switch name {
	case ".gitignore", ".gitattributes", ".editorconfig":
		return true
	}
	suffixes := splitSuffixes(name)
	if len(suffixes) == 0 {
		return false
	}
	if _, ok := textExtensions[suffixes[len(suffixes)-1]]; ok {
		return true
	}
	return false
}

func splitSuffixes(name string) []string {
	if strings.HasPrefix(name, ".") && strings.Count(name, ".") == 1 {
		return nil
	}
	base := name
	var suffixes []string
	for {
		ext := path.Ext(base)
		if ext == "" {
			break
		}
		suffixes = append([]string{strings.ToLower(ext)}, suffixes...)
		base = strings.TrimSuffix(base, ext)
	}
	return suffixes
}

func isBinary(sample []byte) bool {
	if len(sample) == 0 {
		return false
	}
	if bytes.IndexByte(sample, 0) >= 0 {
		return true
	}
	if utf8.Valid(sample) {
		return false
	}
	allowed := map[byte]struct{}{9: {}, 10: {}, 12: {}, 13: {}}
	ctrl := 0
	for _, b := range sample {
		if _, ok := allowed[b]; ok {
			continue
		}
		if b < 32 || b == 127 {
			ctrl++
		}
	}
	return float64(ctrl)/float64(max(1, len(sample))) > 0.30
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func relToSlash(root, absPath string) (string, bool) {
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return "", false
	}
	if rel == "." {
		return "", false
	}
	if strings.HasPrefix(rel, "..") {
		return "", false
	}
	return normalizeRel(filepath.ToSlash(rel)), true
}

func summaryPathFor(outputPath string) string {
	base := filepath.Base(outputPath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	summaryName := fmt.Sprintf("%s_summary.txt", stem)
	return filepath.Join(filepath.Dir(outputPath), summaryName)
}
