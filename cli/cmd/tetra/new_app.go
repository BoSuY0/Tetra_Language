package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/compiler"
)

func runNew(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra new app [--lock] <NameOrPath>")
		return 0
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, "new requires a template")
		return 2
	}
	switch args[0] {
	case "app":
		return runNewAppArgs(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown new template %q\n", args[0])
		return 2
	}
}

type newAppOptions struct {
	WriteLock bool
}

func runNewAppArgs(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra new app [--lock] <NameOrPath>")
		return 0
	}
	var path string
	var opt newAppOptions
	for _, arg := range args {
		switch arg {
		case "--lock":
			opt.WriteLock = true
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(stderr, "unknown new app option %q\n", arg)
				return 2
			}
			if path != "" {
				fmt.Fprintln(stderr, "usage: tetra new app [--lock] <NameOrPath>")
				return 2
			}
			path = arg
		}
	}
	if path == "" {
		fmt.Fprintln(stderr, "usage: tetra new app [--lock] <NameOrPath>")
		return 2
	}
	return runNewApp(path, opt, stdout, stderr)
}

func runNewApp(path string, opt newAppOptions, stdout io.Writer, stderr io.Writer) int {
	if strings.TrimSpace(path) == "" {
		fmt.Fprintln(stderr, "new app requires a name or path")
		return 2
	}
	targetDir := filepath.Clean(filepath.FromSlash(path))
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Fprintf(stderr, "%s already exists\n", targetDir)
		return 2
	} else if !os.IsNotExist(err) {
		fmt.Fprintln(stderr, err)
		return 1
	}
	name := capsuleNameFromPath(targetDir)
	if name == "" {
		fmt.Fprintln(stderr, "new app requires a valid app name")
		return 2
	}
	target := defaultTarget()
	files := map[string]string{
		"Capsule.t4": fmt.Sprintf(`manifest "tetra.capsule.v1"
capsule %s:
    id "tetra://apps/%s"
    version "0.1.0"
    entry "src/main.t4"
    source "src"
    source "tests"
    target "%s"
    permission "io"
`, name, capsuleSlug(name), target),
		"src/main.t4": `func main() -> Int:
    return 0
`,
		"tests/main_test.t4": `test "main returns success":
    expect 40 + 2 == 42
`,
		"README.md": fmt.Sprintf(`# %s

Run:

`+"```bash"+`
tetra check .
tetra build .
tetra run .
tetra test .
`+"```"+`
`, name),
	}
	for rel, content := range files {
		full := filepath.Join(targetDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "Created app: %s\n", targetDir)
	if opt.WriteLock {
		lockPath := filepath.Join(targetDir, compiler.SemanticLockFileName)
		if err := buildCapsuleArtifacts(filepath.Join(targetDir, compiler.CapsuleFileName), capsuleArtifactBuildOptions{
			LockPath: lockPath,
			Jobs:     1,
		}); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "Created lock: %s\n", lockPath)
	}
	return 0
}

func capsuleNameFromPath(path string) string {
	name := filepath.Base(filepath.Clean(path))
	var b strings.Builder
	capitalizeNext := true
	for _, r := range name {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if b.Len() == 0 && r >= '0' && r <= '9' {
				b.WriteByte('T')
			}
			if capitalizeNext && r >= 'a' && r <= 'z' {
				r = r - 'a' + 'A'
			}
			b.WriteRune(r)
			capitalizeNext = false
			continue
		}
		capitalizeNext = true
	}
	return b.String()
}

func capsuleSlug(name string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r - 'A' + 'a')
			lastDash = false
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
