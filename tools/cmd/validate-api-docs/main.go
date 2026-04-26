package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

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
	hasModule := false
	hasSection := false
	hasEntry := false
	currentModule := ""
	currentSection := ""
	var modules []string
	seenModules := map[string]bool{}
	allowedSections := map[string]bool{
		"Enums":           true,
		"Extensions":      true,
		"Functions":       true,
		"Globals":         true,
		"Implementations": true,
		"Protocols":       true,
		"Structs":         true,
		"Tests":           true,
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
	return nil
}
