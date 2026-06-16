package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const baselineSchema = "tetra.directory-budget.v1"

type scanConfig struct {
	Limit        int
	Roots        []string
	Extensions   map[string]bool
	DocsMarkdown bool
	ExcludeDirs  map[string]bool
	ExcludePaths []string
}

type baselineFile struct {
	Schema     string         `json:"schema"`
	Limit      int            `json:"limit"`
	Roots      []string       `json:"roots"`
	Extensions []string       `json:"extensions"`
	Allowances map[string]int `json:"allowances"`
}

type directoryCount struct {
	Path  string
	Count int
	Files []string
}

type violation struct {
	directoryCount
	Allowed int
}

func main() {
	os.Exit(runDirectoryBudget(os.Args[1:], os.Stdout, os.Stderr, "."))
}

func runDirectoryBudget(args []string, stdout, stderr io.Writer, cwd string) int {
	flags := flag.NewFlagSet("validate-directory-budget", flag.ContinueOnError)
	flags.SetOutput(stderr)
	rootsFlag := flags.String("roots", "compiler,cli,tools,lib,examples,docs", "comma-separated roots to scan")
	limitFlag := flags.Int("limit", 6, "maximum active source/script files per directory")
	extensionsFlag := flags.String("extensions", ".go,.tetra,.sh,.mjs,.js,.ts", "comma-separated active source/script extensions")
	docsMarkdownFlag := flags.Bool("docs-markdown", true, "count Markdown files under docs/ as maintained docs")
	excludeDirsFlag := flags.String("exclude-dirs", ".cache,.tetra_cache,node_modules,generated", "comma-separated directory names to skip anywhere")
	excludePathsFlag := flags.String("exclude-paths", "docs/assets", "comma-separated repo-relative directory paths to skip")
	baselinePath := flags.String("baseline", "", "optional baseline JSON for ratcheting existing violations")
	writeBaselinePath := flags.String("write-baseline", "", "write current over-budget directories as a baseline JSON")
	strict := flags.Bool("strict", false, "ignore baseline allowances and require every directory to satisfy --limit")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *limitFlag < 1 {
		fmt.Fprintln(stderr, "error: --limit must be positive")
		return 2
	}

	cfg := scanConfig{
		Limit:        *limitFlag,
		Roots:        splitCSV(*rootsFlag),
		Extensions:   extensionSet(splitCSV(*extensionsFlag)),
		DocsMarkdown: *docsMarkdownFlag,
		ExcludeDirs:  stringSet(splitCSV(*excludeDirsFlag)),
		ExcludePaths: normalizePaths(splitCSV(*excludePathsFlag)),
	}
	if len(cfg.Roots) == 0 {
		fmt.Fprintln(stderr, "error: --roots must name at least one root")
		return 2
	}
	if len(cfg.Extensions) == 0 && !cfg.DocsMarkdown {
		fmt.Fprintln(stderr, "error: no active extensions configured")
		return 2
	}

	counts, err := scanDirectories(cwd, cfg)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if *writeBaselinePath != "" {
		baseline := buildBaseline(cfg, counts)
		if err := writeBaseline(*writeBaselinePath, baseline); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "wrote directory budget baseline: %s (%d allowances)\n", *writeBaselinePath, len(baseline.Allowances))
		return 0
	}

	var baseline *baselineFile
	if *baselinePath != "" && !*strict {
		loaded, err := readBaseline(*baselinePath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		baseline = &loaded
	}

	violations := findViolations(cfg.Limit, counts, baseline)
	if len(violations) == 0 {
		fmt.Fprintln(stdout, "directory budget OK")
		return 0
	}
	printViolations(stderr, violations, baseline != nil)
	return 1
}

func splitCSV(raw string) []string {
	var values []string
	for _, part := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func extensionSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		ext := strings.ToLower(strings.TrimSpace(value))
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		out[ext] = true
	}
	return out
}

func stringSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		if value != "" {
			out[value] = true
		}
	}
	return out
}

func normalizePaths(values []string) []string {
	var out []string
	for _, value := range values {
		normalized := normalizeRelativePath(value)
		if normalized != "" {
			out = append(out, normalized)
		}
	}
	sort.Strings(out)
	return out
}

func scanDirectories(cwd string, cfg scanConfig) ([]directoryCount, error) {
	filesByDir := map[string][]string{}
	for _, root := range cfg.Roots {
		rootRel := normalizeRelativePath(root)
		if rootRel == "" {
			continue
		}
		rootAbs := filepath.Join(cwd, filepath.FromSlash(rootRel))
		if _, err := os.Stat(rootAbs); err != nil {
			return nil, fmt.Errorf("scan root %s: %w", rootRel, err)
		}
		err := filepath.WalkDir(rootAbs, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(cwd, path)
			if err != nil {
				return err
			}
			rel = normalizeRelativePath(rel)
			if entry.IsDir() {
				if shouldSkipDir(rel, entry.Name(), cfg) {
					return filepath.SkipDir
				}
				return nil
			}
			if !isActiveFile(rel, cfg) {
				return nil
			}
			dir := normalizeRelativePath(filepath.Dir(rel))
			filesByDir[dir] = append(filesByDir[dir], rel)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	counts := make([]directoryCount, 0, len(filesByDir))
	for dir, files := range filesByDir {
		sort.Strings(files)
		counts = append(counts, directoryCount{
			Path:  dir,
			Count: len(files),
			Files: files,
		})
	}
	sortDirectoryCounts(counts)
	return counts, nil
}

func shouldSkipDir(rel, name string, cfg scanConfig) bool {
	if rel == "." || rel == "" {
		return false
	}
	if cfg.ExcludeDirs[name] {
		return true
	}
	for _, excluded := range cfg.ExcludePaths {
		if rel == excluded || strings.HasPrefix(rel, excluded+"/") {
			return true
		}
	}
	return false
}

func isActiveFile(rel string, cfg scanConfig) bool {
	ext := strings.ToLower(filepath.Ext(rel))
	if cfg.Extensions[ext] {
		return true
	}
	return cfg.DocsMarkdown && ext == ".md" && (rel == "docs" || strings.HasPrefix(rel, "docs/"))
}

func sortDirectoryCounts(counts []directoryCount) {
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].Count != counts[j].Count {
			return counts[i].Count > counts[j].Count
		}
		return counts[i].Path < counts[j].Path
	})
}

func findViolations(limit int, counts []directoryCount, baseline *baselineFile) []violation {
	var violations []violation
	for _, count := range counts {
		allowed := limit
		if baseline != nil {
			if baselineAllowance := baseline.Allowances[count.Path]; baselineAllowance > allowed {
				allowed = baselineAllowance
			}
		}
		if count.Count > allowed {
			violations = append(violations, violation{
				directoryCount: count,
				Allowed:        allowed,
			})
		}
	}
	return violations
}

func buildBaseline(cfg scanConfig, counts []directoryCount) baselineFile {
	allowances := map[string]int{}
	for _, count := range counts {
		if count.Count > cfg.Limit {
			allowances[count.Path] = count.Count
		}
	}
	return baselineFile{
		Schema:     baselineSchema,
		Limit:      cfg.Limit,
		Roots:      append([]string(nil), cfg.Roots...),
		Extensions: activeExtensions(cfg),
		Allowances: allowances,
	}
}

func activeExtensions(cfg scanConfig) []string {
	var extensions []string
	for ext := range cfg.Extensions {
		extensions = append(extensions, ext)
	}
	if cfg.DocsMarkdown {
		extensions = append(extensions, "docs:.md")
	}
	sort.Strings(extensions)
	return extensions
}

func readBaseline(path string) (baselineFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return baselineFile{}, fmt.Errorf("read baseline: %w", err)
	}
	var baseline baselineFile
	if err := json.Unmarshal(raw, &baseline); err != nil {
		return baselineFile{}, fmt.Errorf("parse baseline: %w", err)
	}
	if baseline.Schema != baselineSchema {
		return baselineFile{}, fmt.Errorf("baseline schema = %q, want %q", baseline.Schema, baselineSchema)
	}
	if baseline.Allowances == nil {
		baseline.Allowances = map[string]int{}
	}
	return baseline, nil
}

func writeBaseline(path string, baseline baselineFile) error {
	raw, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func printViolations(w io.Writer, violations []violation, usingBaseline bool) {
	if usingBaseline {
		fmt.Fprintln(w, "directory budget violations above baseline:")
	} else {
		fmt.Fprintln(w, "directory budget violations:")
	}
	for _, violation := range violations {
		fmt.Fprintf(w, "- %s: %d active files (allowed %d)\n", violation.Path, violation.Count, violation.Allowed)
		for _, file := range violation.Files {
			fmt.Fprintf(w, "  - %s\n", file)
		}
	}
}

func normalizeRelativePath(path string) string {
	normalized := filepath.ToSlash(filepath.Clean(path))
	if normalized == "." {
		return "."
	}
	normalized = strings.TrimPrefix(normalized, "./")
	return strings.Trim(normalized, "/")
}
