package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type flowIssue struct {
	Path    string
	Line    int
	Column  int
	Message string
}

var legacyPatterns = []struct {
	re      *regexp.Regexp
	message string
}{
	{regexp.MustCompile(`^\s*(fun|fn)\b.*\{\s*$`), "legacy function syntax; use Flow 'func name(...) -> Type:'"},
	{regexp.MustCompile(`^\s*if\s*\(.*\)\s*\{\s*$`), "legacy braced if syntax; use Flow 'if condition:'"},
	{regexp.MustCompile(`^\s*while\s*\(.*\)\s*\{\s*$`), "legacy braced while syntax; use Flow 'while condition:'"},
	{regexp.MustCompile(`^\s*unsafe\s*\{\s*$`), "legacy braced unsafe syntax; use Flow 'unsafe:'"},
	{regexp.MustCompile(`^\s*island\s*\(.*\)\s+as\s+\w+\s*\{\s*$`), "legacy braced island syntax; use Flow 'island(size) as name:'"},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: validate-flow-only <file-or-dir>...")
		os.Exit(2)
	}
	issues, err := validatePaths(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(issues) > 0 {
		for _, issue := range issues {
			fmt.Fprintf(os.Stderr, "%s:%d:%d: %s\n", issue.Path, issue.Line, issue.Column, issue.Message)
		}
		os.Exit(1)
	}
}

func validatePaths(paths []string) ([]flowIssue, error) {
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if strings.HasSuffix(path, ".tetra") {
				files = append(files, path)
			}
			continue
		}
		if err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				name := d.Name()
				if strings.HasPrefix(name, ".") && p != path {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(p, ".tetra") {
				files = append(files, p)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	sort.Strings(files)

	var issues []flowIssue
	for _, file := range files {
		fileIssues, err := validateFile(file)
		if err != nil {
			return nil, err
		}
		issues = append(issues, fileIssues...)
	}
	return issues, nil
}

func validateFile(path string) ([]flowIssue, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var issues []flowIssue
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		code := stripStringsAndLineComment(line)
		trimmed := strings.TrimSpace(code)
		if trimmed == "" {
			continue
		}
		for _, pattern := range legacyPatterns {
			if pattern.re.MatchString(trimmed) || pattern.re.MatchString(code) {
				issues = append(issues, flowIssue{Path: path, Line: lineNo, Column: firstCodeColumn(line), Message: pattern.message})
			}
		}
		if strings.HasSuffix(trimmed, ";") {
			issues = append(issues, flowIssue{Path: path, Line: lineNo, Column: strings.LastIndex(code, ";") + 1, Message: "trailing semicolon; Flow syntax is semicolon-free"})
		}
		if col := strings.Index(code, "\t"); col >= 0 {
			issues = append(issues, flowIssue{Path: path, Line: lineNo, Column: col + 1, Message: "tabs are not supported in Flow indentation"})
		}
		if col := strings.Index(code, "{"); col >= 0 {
			issues = append(issues, flowIssue{Path: path, Line: lineNo, Column: col + 1, Message: "legacy brace token; Flow syntax uses indentation blocks"})
		}
		if col := strings.Index(code, "}"); col >= 0 {
			issues = append(issues, flowIssue{Path: path, Line: lineNo, Column: col + 1, Message: "legacy brace token; Flow syntax uses indentation blocks"})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return issues, nil
}

func stripStringsAndLineComment(line string) string {
	var out strings.Builder
	out.Grow(len(line))
	inString := false
	escaped := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if !inString && ch == '/' && i+1 < len(line) && line[i+1] == '/' {
			break
		}
		if inString {
			if escaped {
				escaped = false
				out.WriteByte(' ')
				continue
			}
			if ch == '\\' {
				escaped = true
				out.WriteByte(' ')
				continue
			}
			if ch == '"' {
				inString = false
			}
			out.WriteByte(' ')
			continue
		}
		if ch == '"' {
			inString = true
			out.WriteByte(' ')
			continue
		}
		out.WriteByte(ch)
	}
	return out.String()
}

func firstCodeColumn(line string) int {
	for i, r := range line {
		if r != ' ' && r != '\t' {
			return i + 1
		}
	}
	return 1
}
