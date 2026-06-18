package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const baselineSchema = "tetra.line-length-baseline.v1"

var defaultRoots = []string{
	"compiler",
	"cli",
	"tools",
	"lib",
	"examples",
	"docs",
	"scripts",
	".github",
}

var defaultExtensions = []string{
	".go",
	".tetra",
	".sh",
	".js",
	".mjs",
	".ts",
	".md",
	".yml",
	".yaml",
	".json",
	".toml",
}

var defaultExcludeDirs = []string{
	".git",
	".cache",
	".tetra_cache",
	"graphify-out",
	"node_modules",
	"vendor",
	"reports",
	"dumps",
}

var defaultExcludePaths = []string{
	"docs/baselines",
	"docs/generated",
	"docs/release/production/data",
	"docs/release/v0_3/data",
	"docs/release/v0_4/data",
	"tools/cmd/validate/line-length/baseline.json",
}

var defaultSkipSuffixes = []string{
	".lock",
	".sum",
	".min.js",
	".map",
	".svg",
	".png",
	".jpg",
	".wasm",
	".tar.gz",
	"_report.json",
	"-prompt.md",
}

var (
	urlPattern      = regexp.MustCompile(`https?://\S+`)
	checksumPattern = regexp.MustCompile(`(?i)\b(?:md5|sha1|sha256|sha512):[0-9a-f]{32,}\b`)
)

type lineConfig struct {
	Max                  int
	Roots                []string
	Extensions           map[string]bool
	ExcludeDirs          map[string]bool
	ExcludePaths         []string
	SkipSuffixes         []string
	IgnoreMarkdownFences bool
}

type lineBaselineFile struct {
	Schema     string              `json:"schema"`
	Max        int                 `json:"max"`
	Allowances []baselineAllowance `json:"allowances"`
}

type baselineAllowance struct {
	Path     string `json:"path"`
	LineHash string `json:"line_hash"`
	Length   int    `json:"length"`
	Reason   string `json:"reason"`
}

type lineViolation struct {
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Length   int    `json:"length"`
	Max      int    `json:"max"`
	LineHash string `json:"line_hash"`
}

type lineReport struct {
	OK                 bool            `json:"ok"`
	Max                int             `json:"max"`
	ManualIgnores      int             `json:"manual_ignores"`
	BaselineAllowances int             `json:"baseline_allowances,omitempty"`
	Violations         []lineViolation `json:"violations"`
}

type repeatedFlag []string

func main() {
	os.Exit(runLineLength(os.Args[1:], os.Stdout, os.Stderr, "."))
}

func runLineLength(args []string, stdout, stderr io.Writer, cwd string) int {
	flags := flag.NewFlagSet("validate-line-length", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var roots repeatedFlag
	flags.Var(&roots, "root", "root to scan; repeatable")
	rootsCSV := flags.String("roots", "", "comma-separated roots to scan")
	maxFlag := flags.Int("max", 100, "maximum visible characters per line")
	extensionsFlag := flags.String(
		"extensions",
		strings.Join(defaultExtensions, ","),
		"comma-separated extensions to scan",
	)
	baselinePath := flags.String("baseline", "", "optional baseline JSON")
	writeBaselinePath := flags.String("write-baseline", "", "write baseline JSON")
	formatFlag := flags.String("format", "text", "output format: text or json")
	strict := flags.Bool("strict", false, "reject all baseline allowances")
	ratchet := flags.Bool("ratchet", false, "require a baseline and reject new debt")

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *maxFlag < 1 {
		fmt.Fprintln(stderr, "error: --max must be positive")
		return 2
	}
	if *formatFlag != "text" && *formatFlag != "json" {
		fmt.Fprintln(stderr, "error: --format must be text or json")
		return 2
	}
	if *ratchet && *baselinePath == "" {
		fmt.Fprintln(stderr, "error: --ratchet requires --baseline")
		return 2
	}

	selectedRoots := append([]string(nil), roots...)
	selectedRoots = append(selectedRoots, splitCSV(*rootsCSV)...)
	if len(selectedRoots) == 0 {
		selectedRoots = append([]string(nil), defaultRoots...)
	}

	cfg := lineConfig{
		Max:                  *maxFlag,
		Roots:                normalizePaths(selectedRoots),
		Extensions:           extensionSet(splitCSV(*extensionsFlag)),
		ExcludeDirs:          stringSet(defaultExcludeDirs),
		ExcludePaths:         normalizePaths(defaultExcludePaths),
		SkipSuffixes:         append([]string(nil), defaultSkipSuffixes...),
		IgnoreMarkdownFences: !*strict,
	}
	if len(cfg.Roots) == 0 {
		fmt.Fprintln(stderr, "error: --root must name at least one root")
		return 2
	}
	if len(cfg.Extensions) == 0 {
		fmt.Fprintln(stderr, "error: --extensions must not be empty")
		return 2
	}

	var baseline *lineBaselineFile
	if *baselinePath != "" {
		loaded, err := readBaseline(*baselinePath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		baseline = &loaded
		if *strict && len(loaded.Allowances) > 0 {
			fmt.Fprintln(stderr, "strict mode does not allow baseline allowances")
			return 1
		}
	}

	violations, manualIgnores, err := scanLineLengths(cwd, cfg)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if *writeBaselinePath != "" {
		built := buildBaseline(cfg, violations)
		if err := writeBaseline(*writeBaselinePath, built); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(
			stdout,
			"wrote line length baseline: %s (%d allowances)\n",
			*writeBaselinePath,
			len(built.Allowances),
		)
		fmt.Fprintf(stdout, "manual ignores: %d\n", manualIgnores)
		return 0
	}

	activeViolations := violations
	if baseline != nil && !*strict {
		activeViolations = violationsAboveBaseline(violations, baseline)
	}

	report := lineReport{
		OK:                 len(activeViolations) == 0,
		Max:                cfg.Max,
		ManualIgnores:      manualIgnores,
		Violations:         activeViolations,
		BaselineAllowances: baselineAllowanceCount(baseline),
	}
	if *formatFlag == "json" {
		writeJSONReport(stdout, report)
		if report.OK {
			return 0
		}
		return 1
	}
	if report.OK {
		fmt.Fprintln(stdout, "line length OK")
		fmt.Fprintf(stdout, "manual ignores: %d\n", manualIgnores)
		return 0
	}
	printViolations(stderr, activeViolations, baseline != nil && !*strict)
	fmt.Fprintf(stderr, "manual ignores: %d\n", manualIgnores)
	return 1
}

func (f *repeatedFlag) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(*f, ",")
}

func (f *repeatedFlag) Set(value string) error {
	*f = append(*f, splitCSV(value)...)
	return nil
}

func scanLineLengths(cwd string, cfg lineConfig) ([]lineViolation, int, error) {
	var violations []lineViolation
	manualIgnores := 0
	for _, root := range cfg.Roots {
		rootAbs := filepath.Join(cwd, filepath.FromSlash(root))
		if _, err := os.Stat(rootAbs); err != nil {
			return nil, 0, fmt.Errorf("scan root %s: %w", root, err)
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
			if shouldSkipPath(rel, cfg) {
				return nil
			}
			if !isMaintainedFile(rel, cfg) {
				return nil
			}
			fileViolations, fileIgnores, err := scanFile(path, rel, cfg)
			if err != nil {
				return err
			}
			manualIgnores += fileIgnores
			violations = append(violations, fileViolations...)
			return nil
		})
		if err != nil {
			return nil, 0, err
		}
	}
	sortViolations(violations)
	return violations, manualIgnores, nil
}

func scanFile(path, rel string, cfg lineConfig) ([]lineViolation, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	var violations []lineViolation
	manualIgnores := 0
	inFence := false
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSuffix(scanner.Text(), "\r")
		length := utf8.RuneCountInString(line)
		if length > cfg.Max {
			if strings.Contains(line, "line-length: ignore") {
				manualIgnores++
			} else if !isAutomaticException(line, rel, inFence, cfg) {
				violations = append(violations, lineViolation{
					Path:     rel,
					Line:     lineNumber,
					Length:   length,
					Max:      cfg.Max,
					LineHash: hashLine(line),
				})
			}
		}
		if isFenceMarker(line) {
			inFence = !inFence
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("scan %s: %w", rel, err)
	}
	return violations, manualIgnores, nil
}

func shouldSkipDir(rel, name string, cfg lineConfig) bool {
	if rel == "." || rel == "" {
		return false
	}
	if cfg.ExcludeDirs[name] {
		return true
	}
	return shouldSkipPath(rel, cfg)
}

func shouldSkipPath(rel string, cfg lineConfig) bool {
	for _, excluded := range cfg.ExcludePaths {
		if rel == excluded || strings.HasPrefix(rel, excluded+"/") {
			return true
		}
	}
	return false
}

func isMaintainedFile(rel string, cfg lineConfig) bool {
	lower := strings.ToLower(rel)
	for _, suffix := range cfg.SkipSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return false
		}
	}
	return cfg.Extensions[strings.ToLower(filepath.Ext(rel))]
}

func isAutomaticException(line, rel string, inFence bool, cfg lineConfig) bool {
	if urlPattern.MatchString(line) {
		return true
	}
	if checksumPattern.MatchString(line) {
		return true
	}
	if isMarkdownTableSeparator(line) {
		return true
	}
	if cfg.IgnoreMarkdownFences && isMarkdownFile(rel) && inFence {
		return true
	}
	return false
}

func isMarkdownFile(rel string) bool {
	return strings.EqualFold(filepath.Ext(rel), ".md")
}

func isFenceMarker(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

func isMarkdownTableSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.Contains(trimmed, "|") || !strings.Contains(trimmed, "-") {
		return false
	}
	withoutAllowed := strings.Map(func(r rune) rune {
		switch r {
		case '|', '-', ':', ' ':
			return -1
		default:
			return r
		}
	}, trimmed)
	return withoutAllowed == ""
}

func violationsAboveBaseline(
	violations []lineViolation,
	baseline *lineBaselineFile,
) []lineViolation {
	if baseline == nil {
		return violations
	}
	allowed := map[string]bool{}
	for _, allowance := range baseline.Allowances {
		allowed[baselineKey(allowance.Path, allowance.LineHash)] = true
	}
	var out []lineViolation
	for _, violation := range violations {
		if !allowed[baselineKey(violation.Path, violation.LineHash)] {
			out = append(out, violation)
		}
	}
	return out
}

func buildBaseline(cfg lineConfig, violations []lineViolation) lineBaselineFile {
	allowances := make([]baselineAllowance, 0, len(violations))
	for _, violation := range violations {
		allowances = append(allowances, baselineAllowance{
			Path:     violation.Path,
			LineHash: violation.LineHash,
			Length:   violation.Length,
			Reason:   "existing debt",
		})
	}
	sort.Slice(allowances, func(i, j int) bool {
		if allowances[i].Path != allowances[j].Path {
			return allowances[i].Path < allowances[j].Path
		}
		return allowances[i].LineHash < allowances[j].LineHash
	})
	return lineBaselineFile{
		Schema:     baselineSchema,
		Max:        cfg.Max,
		Allowances: allowances,
	}
}

func readBaseline(path string) (lineBaselineFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return lineBaselineFile{}, fmt.Errorf("read baseline: %w", err)
	}
	var baseline lineBaselineFile
	if err := json.Unmarshal(raw, &baseline); err != nil {
		return lineBaselineFile{}, fmt.Errorf("parse baseline: %w", err)
	}
	if baseline.Schema != baselineSchema {
		return lineBaselineFile{}, fmt.Errorf(
			"baseline schema = %q, want %q",
			baseline.Schema,
			baselineSchema,
		)
	}
	if baseline.Allowances == nil {
		baseline.Allowances = []baselineAllowance{}
	}
	return baseline, nil
}

func writeBaseline(path string, baseline lineBaselineFile) error {
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

func writeJSONReport(w io.Writer, report lineReport) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(report)
}

func printViolations(w io.Writer, violations []lineViolation, usingBaseline bool) {
	if usingBaseline {
		fmt.Fprintln(w, "line length violations above baseline:")
	} else {
		fmt.Fprintln(w, "line length violations:")
	}
	for _, violation := range violations {
		fmt.Fprintf(
			w,
			"- %s:%d: line is %d chars, max %d\n",
			violation.Path,
			violation.Line,
			violation.Length,
			violation.Max,
		)
	}
}

func hashLine(line string) string {
	sum := sha256.Sum256([]byte(strings.TrimSuffix(line, "\r")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func baselineKey(path, hash string) string {
	return path + "\x00" + hash
}

func baselineAllowanceCount(baseline *lineBaselineFile) int {
	if baseline == nil {
		return 0
	}
	return len(baseline.Allowances)
}

func sortViolations(violations []lineViolation) {
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].Path != violations[j].Path {
			return violations[i].Path < violations[j].Path
		}
		return violations[i].Line < violations[j].Line
	})
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

func normalizeRelativePath(path string) string {
	normalized := filepath.ToSlash(filepath.Clean(path))
	if normalized == "." {
		return "."
	}
	normalized = strings.TrimPrefix(normalized, "./")
	return strings.Trim(normalized, "/")
}
