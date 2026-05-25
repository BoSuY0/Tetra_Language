package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const maxDumpFileBytes = 10 * 1024 * 1024

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "create_dumps: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if wantsHelp(args) {
		printUsage()
		return nil
	}

	root, err := gitRoot()
	if err != nil {
		return err
	}
	dumpDir := filepath.Join(root, "dumps")
	if err := os.MkdirAll(dumpDir, 0o755); err != nil {
		return fmt.Errorf("create dumps directory: %w", err)
	}

	forwardArgs, outputPath, err := sanitizeArgs(root, dumpDir, args)
	if err != nil {
		return err
	}

	if _, err := removePreviousDumpFiles(dumpDir); err != nil {
		return err
	}

	cmdArgs := append([]string{"run", "./tools/cmd/dump-project"}, forwardArgs...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = root
	cmd.Stdin = os.Stdin
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.Stdout.Write(stdout.Bytes())
		os.Stderr.Write(stderr.Bytes())
		return fmt.Errorf("run dump-project: %w", err)
	}

	paths, err := splitDumpFile(outputPath, maxDumpFileBytes)
	if err != nil {
		return err
	}
	os.Stderr.Write(stderr.Bytes())
	printDumpResult(stdout.String(), paths)
	return nil
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

func printUsage() {
	fmt.Fprint(os.Stdout, `Usage: go run ./create_dumps.go [dump-project flags...]

Creates project dump artifacts under ./dumps/ while preserving gitignore filtering.
Each dump artifact is Markdown and is capped at 10 MiB per .md file.
Before creating a new dump, existing top-level files under ./dumps/ are removed.

Common examples:
  go run ./create_dumps.go
  go run ./create_dumps.go --all
  go run ./create_dumps.go --only compiler --out compiler_dump.txt

Output rules:
  --out may be a file name or a path inside dumps/.
  Plain file names are written as dumps/<name>.md.
  Larger dumps are split as dumps/<name>_part_001.md, dumps/<name>_part_002.md, ...
  .gitignore is always applied, including tracked files that match ignore rules.

Disabled here:
  --root, --no-git, --include-ignored, --include-dumps, --include-dotenv, --file-list
`)
}

func gitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve git root: %w", err)
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return "", fmt.Errorf("resolve git root: empty path")
	}
	return filepath.Clean(root), nil
}

func sanitizeArgs(root, dumpDir string, args []string) ([]string, string, error) {
	forward := make([]string, 0, len(args))
	outputPath := ""
	hasNoSummary := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, value, inline := splitFlagValue(arg)

		switch name {
		case "--include-dumps", "-include-dumps",
			"--include-dotenv", "-include-dotenv",
			"--include-ignored", "-include-ignored",
			"--file-list", "-file-list",
			"--no-git", "-no-git",
			"--root", "-root":
			return nil, "", fmt.Errorf("%s is disabled here so dumps stay gitignore-filtered for this project", name)
		case "--no-summary", "-no-summary":
			hasNoSummary = true
			forward = append(forward, arg)
		case "--out", "-out":
			if inline {
				out, err := normalizeOutputPath(root, dumpDir, value)
				if err != nil {
					return nil, "", err
				}
				forward = append(forward, name+"="+out)
				outputPath = out
				continue
			}
			if i+1 >= len(args) {
				return nil, "", fmt.Errorf("%s needs a path inside dumps/", name)
			}
			out, err := normalizeOutputPath(root, dumpDir, args[i+1])
			if err != nil {
				return nil, "", err
			}
			forward = append(forward, arg, out)
			outputPath = out
			i++
			continue
		}

		forward = append(forward, arg)
	}

	if outputPath == "" {
		outputPath = defaultOutputPath(root, dumpDir)
		forward = append(forward, "--out", outputPath)
	}
	if !hasNoSummary {
		forward = append(forward, "--no-summary")
	}
	return forward, outputPath, nil
}

func splitFlagValue(arg string) (name, value string, inline bool) {
	if !strings.HasPrefix(arg, "-") {
		return arg, "", false
	}
	if idx := strings.IndexByte(arg, '='); idx >= 0 {
		return arg[:idx], arg[idx+1:], true
	}
	return arg, "", false
}

func normalizeOutputPath(root, dumpDir, raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("--out needs a path inside dumps/")
	}

	cleaned := filepath.Clean(raw)
	if !filepath.IsAbs(cleaned) && filepath.Dir(cleaned) == "." {
		cleaned = filepath.Join("dumps", cleaned)
	}

	abs := cleaned
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, cleaned)
	}
	abs = filepath.Clean(abs)

	if abs == dumpDir {
		return "", fmt.Errorf("--out must be a file path inside dumps/, not the dumps directory")
	}
	if !isUnderDir(dumpDir, abs) {
		return "", fmt.Errorf("--out must stay inside %s", dumpDir)
	}
	return withMarkdownExtension(abs), nil
}

func isUnderDir(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func defaultOutputPath(root, dumpDir string) string {
	ts := time.Now().UTC().Format("20060102_150405Z")
	name := strings.ToLower(filepath.Base(root))
	name = strings.ReplaceAll(name, " ", "_")
	if name == "" || name == "." {
		name = "project"
	}
	return filepath.Join(dumpDir, fmt.Sprintf("%s_dump_%s.md", name, ts))
}

func withMarkdownExtension(path string) string {
	ext := filepath.Ext(path)
	if ext == ".md" {
		return path
	}
	if ext == "" {
		return path + ".md"
	}
	return strings.TrimSuffix(path, ext) + ".md"
}

func splitDumpFile(path string, maxBytes int64) ([]string, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("dump file limit must be positive")
	}
	if err := removeExistingChunks(path); err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat dump file: %w", err)
	}
	if info.Size() <= maxBytes {
		return []string{path}, nil
	}

	in, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open dump for splitting: %w", err)
	}

	chunkSize, err := intFromInt64(maxBytes)
	if err != nil {
		in.Close()
		return nil, err
	}
	buf := make([]byte, chunkSize)
	base := strings.TrimSuffix(path, filepath.Ext(path))
	var paths []string

	for part := 1; ; part++ {
		n, readErr := io.ReadFull(in, buf)
		if readErr == io.EOF {
			break
		}
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			in.Close()
			return nil, fmt.Errorf("read dump chunk: %w", readErr)
		}
		chunkPath := fmt.Sprintf("%s_part_%03d.md", base, part)
		if err := os.WriteFile(chunkPath, buf[:n], 0o644); err != nil {
			in.Close()
			return nil, fmt.Errorf("write dump chunk: %w", err)
		}
		paths = append(paths, chunkPath)
		if readErr == io.ErrUnexpectedEOF {
			break
		}
	}

	if err := in.Close(); err != nil {
		return nil, fmt.Errorf("close dump before replacing with chunks: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return nil, fmt.Errorf("remove oversized dump after splitting: %w", err)
	}
	return paths, nil
}

func removeExistingChunks(path string) error {
	base := strings.TrimSuffix(path, filepath.Ext(path))
	matches, err := filepath.Glob(base + "_part_*.md")
	if err != nil {
		return fmt.Errorf("find existing dump chunks: %w", err)
	}
	for _, match := range matches {
		if err := os.Remove(match); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove stale dump chunk %s: %w", match, err)
		}
	}
	return nil
}

func removePreviousDumpFiles(dumpDir string) (int, error) {
	entries, err := os.ReadDir(dumpDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read dumps directory: %w", err)
	}

	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dumpDir, entry.Name())
		if err := os.Remove(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return removed, fmt.Errorf("remove previous dump file %s: %w", path, err)
		}
		removed++
	}
	return removed, nil
}

func intFromInt64(value int64) (int, error) {
	out := int(value)
	if int64(out) != value {
		return 0, fmt.Errorf("dump file limit is too large for this platform")
	}
	return out, nil
}

func printDumpResult(toolOutput string, paths []string) {
	if len(paths) == 1 {
		fmt.Print(toolOutput)
		return
	}

	fmt.Printf("Dump split into %d Markdown files (max %d bytes each):\n", len(paths), maxDumpFileBytes)
	for _, path := range paths {
		fmt.Printf("  %s\n", path)
	}
	for _, line := range strings.Split(toolOutput, "\n") {
		if line == "" || strings.HasPrefix(line, "Dump created:") {
			continue
		}
		fmt.Println(line)
	}
}
