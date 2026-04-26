package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/compiler"
)

func main() {
	var dir string
	flag.StringVar(&dir, "dir", "", "path to unpacked Eco project bundle")
	flag.Parse()

	if dir == "" {
		fmt.Fprintln(os.Stderr, "error: --dir is required")
		os.Exit(2)
	}
	if err := validateEcoUnpack(dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateEcoUnpack(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	manifestPath := filepath.Join(dir, "Tetra.capsule")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing Tetra.capsule")
		}
		return err
	}
	if err := validateManifestText(string(raw)); err != nil {
		return err
	}
	srcDir := filepath.Join(dir, "src")
	if info, err := os.Stat(srcDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing .tetra sources under src")
		}
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("src is not a directory")
	}
	hasSource := false
	if err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".tetra" {
			hasSource = true
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := compiler.ParseFile(raw, filepath.ToSlash(path)); err != nil {
				return fmt.Errorf("%s: parse failed: %w", path, err)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if !hasSource {
		return fmt.Errorf("missing .tetra sources under src")
	}
	return nil
}

func validateManifestText(text string) error {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return fmt.Errorf("manifest is empty")
	}
	if !strings.Contains(trimmed, "capsule ") {
		return fmt.Errorf("manifest missing capsule declaration")
	}
	if !strings.Contains(trimmed, "\n  id ") && !strings.Contains(trimmed, "\nid ") {
		return fmt.Errorf("manifest missing id")
	}
	if !strings.Contains(trimmed, "\n  version ") && !strings.Contains(trimmed, "\nversion ") {
		return fmt.Errorf("manifest missing version")
	}
	if !strings.Contains(trimmed, "\n  target ") && !strings.Contains(trimmed, "\ntarget ") {
		return fmt.Errorf("manifest missing target")
	}
	return nil
}
