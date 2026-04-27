package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type apiMetadata struct {
	Schema      string `json:"schema"`
	APIHash     string `json:"api_hash"`
	ModuleCount int    `json:"module_count"`
	EntryCount  int    `json:"entry_count"`
}

var internalMarkdownLinkRE = regexp.MustCompile(`\[[^\]]+\]\(#([^)]+)\)`)

func main() {
	var docsPath string
	flag.StringVar(&docsPath, "docs", "", "path to generated API docs markdown")
	flag.Parse()

	if docsPath == "" {
		fmt.Fprintln(os.Stderr, "error: --docs is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(docsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateAPIDocs(string(raw)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateAPIDocs(md string) error {
	if strings.TrimSpace(md) == "" {
		return fmt.Errorf("API docs are empty")
	}
	lines := strings.Split(md, "\n")
	if strings.TrimSpace(lines[0]) != "# Tetra API Docs" {
		return fmt.Errorf("missing # Tetra API Docs heading")
	}
	metadata, err := parseAPIMetadata(lines[1:])
	if err != nil {
		return err
	}
	hasModule := false
	hasSection := false
	hasEntry := false
	currentModule := ""
	currentSection := ""
	var modules []string
	var surface []string
	entryCount := 0
	seenModules := map[string]bool{}
	allowedSections := map[string]bool{
		"Enums":           true,
		"Extensions":      true,
		"Functions":       true,
		"Globals":         true,
		"Implementations": true,
		"Protocols":       true,
		"Doctests":        true,
		"States":          true,
		"Structs":         true,
		"Tests":           true,
		"Views":           true,
	}
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### ") {
			currentModule = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			currentSection = ""
			if currentModule == "" {
				return fmt.Errorf("empty module heading")
			}
			if seenModules[currentModule] {
				return fmt.Errorf("duplicate module heading %s", currentModule)
			}
			seenModules[currentModule] = true
			modules = append(modules, currentModule)
			surface = append(surface, trimmed)
			hasModule = true
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			if currentModule == "" {
				return fmt.Errorf("API section appears before module heading")
			}
			currentSection = strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
			if !allowedSections[currentSection] {
				return fmt.Errorf("unknown API section %s", currentSection)
			}
			hasSection = true
			continue
		}
		if strings.HasPrefix(trimmed, "- `") {
			if currentModule == "" || currentSection == "" {
				return fmt.Errorf("API entry appears before module section")
			}
			surface = append(surface, trimmed)
			entryCount++
			hasEntry = true
		}
	}
	if !hasModule {
		return fmt.Errorf("missing module headings")
	}
	if !hasSection {
		return fmt.Errorf("missing API sections")
	}
	if !hasEntry {
		return fmt.Errorf("missing API entry bullets")
	}
	if metadata.ModuleCount != len(modules) {
		return fmt.Errorf("API metadata module_count mismatch: expected %d, got %d", len(modules), metadata.ModuleCount)
	}
	if metadata.EntryCount != entryCount {
		return fmt.Errorf("API metadata entry_count mismatch: expected %d, got %d", entryCount, metadata.EntryCount)
	}
	wantHash := "sha256:" + hashAPISurface(surface)
	if metadata.APIHash != wantHash {
		return fmt.Errorf("API metadata api_hash mismatch: expected %s, got %s", wantHash, metadata.APIHash)
	}
	if err := validateInternalLinks(lines); err != nil {
		return err
	}
	return nil
}

func validateInternalLinks(lines []string) error {
	anchors := map[string]bool{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "# "):
			anchors[markdownAnchor(strings.TrimSpace(strings.TrimPrefix(trimmed, "# ")))] = true
		case strings.HasPrefix(trimmed, "## "):
			anchors[markdownAnchor(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))] = true
		case strings.HasPrefix(trimmed, "### "):
			anchors[markdownAnchor(strings.TrimSpace(strings.TrimPrefix(trimmed, "### ")))] = true
		}
	}
	for _, line := range lines {
		for _, match := range internalMarkdownLinkRE.FindAllStringSubmatch(line, -1) {
			if len(match) != 2 {
				continue
			}
			anchor := strings.TrimSpace(match[1])
			if !anchors[anchor] {
				return fmt.Errorf("broken internal link #%s", anchor)
			}
		}
	}
	return nil
}

func markdownAnchor(heading string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(heading) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func parseAPIMetadata(lines []string) (apiMetadata, error) {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			return apiMetadata{}, fmt.Errorf("missing tetra-api-metadata")
		}
		if !strings.HasPrefix(trimmed, "<!-- tetra-api-metadata:") || !strings.HasSuffix(trimmed, "-->") {
			return apiMetadata{}, fmt.Errorf("missing tetra-api-metadata")
		}
		raw := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "<!-- tetra-api-metadata:"), "-->"))
		var metadata apiMetadata
		if err := decodeStrictJSON([]byte(raw), &metadata); err != nil {
			return apiMetadata{}, fmt.Errorf("invalid tetra-api-metadata: %v", err)
		}
		if metadata.Schema != "tetra.api.v1alpha1" {
			return apiMetadata{}, fmt.Errorf("unsupported API metadata schema %q", metadata.Schema)
		}
		if !strings.HasPrefix(metadata.APIHash, "sha256:") {
			return apiMetadata{}, fmt.Errorf("API metadata api_hash must use sha256")
		}
		if metadata.ModuleCount <= 0 {
			return apiMetadata{}, fmt.Errorf("API metadata module_count must be positive")
		}
		if metadata.EntryCount < 0 {
			return apiMetadata{}, fmt.Errorf("API metadata entry_count must be non-negative")
		}
		return metadata, nil
	}
	return apiMetadata{}, fmt.Errorf("missing tetra-api-metadata")
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func hashAPISurface(surface []string) string {
	sum := sha256.Sum256([]byte(strings.Join(surface, "\n")))
	return fmt.Sprintf("%x", sum[:])
}
