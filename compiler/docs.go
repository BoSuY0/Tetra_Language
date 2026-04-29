package compiler

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

func GenerateAPIDocs(paths []string) ([]byte, error) {
	files, err := collectDocFiles(paths)
	if err != nil {
		return nil, err
	}
	var parsed []*frontend.FileAST
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		file, err := frontend.ParseFile(raw, path)
		if err != nil {
			return nil, err
		}
		file.Path = path
		parsed = append(parsed, file)
	}
	var body bytes.Buffer
	for _, file := range parsed {
		writeFileAPIDocs(&body, file)
	}
	var b bytes.Buffer
	writeAPIDocsHeader(&b, len(parsed), body.String())
	b.Write(body.Bytes())
	return b.Bytes(), nil
}

func GenerateAPIDocsFromSource(src []byte, filename string) ([]byte, error) {
	file, err := frontend.ParseFile(src, filename)
	if err != nil {
		return nil, err
	}
	file.Path = filename
	var body bytes.Buffer
	writeFileAPIDocs(&body, file)
	var b bytes.Buffer
	writeAPIDocsHeader(&b, 1, body.String())
	b.Write(body.Bytes())
	return b.Bytes(), nil
}

func writeAPIDocsHeader(b *bytes.Buffer, moduleCount int, body string) {
	entryCount, hash := apiSurfaceMetadata(body)
	b.WriteString("# Tetra API Docs\n\n")
	fmt.Fprintf(b, "<!-- tetra-api-metadata: {\"schema\":\"tetra.api.v1alpha1\",\"api_hash\":\"sha256:%s\",\"module_count\":%d,\"entry_count\":%d} -->\n\n", hash, moduleCount, entryCount)
}

func apiSurfaceMetadata(body string) (int, string) {
	var surface []string
	entryCount := 0
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### "):
			surface = append(surface, trimmed)
		case strings.HasPrefix(trimmed, "- `"):
			surface = append(surface, trimmed)
			entryCount++
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(surface, "\n")))
	return entryCount, fmt.Sprintf("%x", sum[:])
}

func collectDocFiles(paths []string) ([]string, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	seen := map[string]struct{}{}
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if isDocSourceFile(path) {
				seen[path] = struct{}{}
				files = append(files, path)
			}
			continue
		}
		err = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") && p != path {
					return filepath.SkipDir
				}
				return nil
			}
			if isDocSourceFile(p) {
				if _, ok := seen[p]; !ok {
					seen[p] = struct{}{}
					files = append(files, p)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

func isDocSourceFile(path string) bool {
	if !IsSourceFile(path) {
		return false
	}
	base := filepath.Base(path)
	return base != CapsuleFileName && base != LegacyCapsuleFileName
}

func writeFileAPIDocs(b *bytes.Buffer, file *frontend.FileAST) {
	title := file.Path
	if file.Module != "" {
		title = file.Module
	}
	experimental := isExperimentalModuleTitle(title)
	if experimental {
		title += " (experimental)"
	}
	fmt.Fprintf(b, "## %s\n\n", title)
	if experimental {
		b.WriteString("Experimental module: compatibility is not guaranteed for v1.x.\n\n")
	}
	if len(file.Structs) > 0 {
		b.WriteString("### Structs\n\n")
		for _, st := range file.Structs {
			fmt.Fprintf(b, "- `%s`\n", st.Name)
			for _, field := range st.Fields {
				fmt.Fprintf(b, "  - `%s: %s`\n", field.Name, formatLSPTypeRef(field.Type))
			}
		}
		b.WriteByte('\n')
	}
	if len(file.States) > 0 {
		b.WriteString("### States\n\n")
		for _, st := range file.States {
			fmt.Fprintf(b, "- `state %s`\n", st.Name)
			for _, field := range st.Fields {
				kind := "val"
				if field.Mutable {
					kind = "var"
				} else if field.Const {
					kind = "const"
				}
				fmt.Fprintf(b, "  - `%s %s: %s`\n", kind, field.Name, formatLSPTypeRef(field.Type))
			}
		}
		b.WriteByte('\n')
	}
	if len(file.Views) > 0 {
		b.WriteString("### Views\n\n")
		for _, view := range file.Views {
			fmt.Fprintf(b, "- `view %s(state: %s)`\n", view.Name, formatLSPTypeRef(view.StateName))
			for _, binding := range view.Bindings {
				fmt.Fprintf(b, "  - `bind %s: %s`\n", binding.Name, formatLSPTypeRef(binding.Type))
			}
			for _, event := range view.Events {
				fmt.Fprintf(b, "  - `event %s -> %s`\n", event.Name, event.Command)
			}
			for _, command := range view.Commands {
				fmt.Fprintf(b, "  - `command %s`\n", command.Name)
			}
			for _, style := range view.Styles {
				fmt.Fprintf(b, "  - `style %s: %s`\n", style.Name, formatLSPTypeRef(style.Type))
			}
			for _, entry := range view.Accessibility {
				fmt.Fprintf(b, "  - `accessibility %s: %s`\n", entry.Name, formatLSPTypeRef(entry.Type))
			}
		}
		b.WriteByte('\n')
	}
	if len(file.Enums) > 0 {
		b.WriteString("### Enums\n\n")
		for _, en := range file.Enums {
			fmt.Fprintf(b, "- `%s`: ", en.Name)
			cases := make([]string, 0, len(en.Cases))
			for _, c := range en.Cases {
				cases = append(cases, c.Name)
			}
			b.WriteString(strings.Join(cases, ", "))
			b.WriteString("\n")
		}
		b.WriteByte('\n')
	}
	if len(file.Protocols) > 0 {
		b.WriteString("### Protocols\n\n")
		for _, proto := range file.Protocols {
			fmt.Fprintf(b, "- `protocol %s`\n", proto.Name)
			for _, req := range proto.Requirements {
				fmt.Fprintf(b, "  - `%s`\n", formatLSPFuncSigDecl(req))
			}
		}
		b.WriteByte('\n')
	}
	if len(file.Globals) > 0 {
		b.WriteString("### Globals\n\n")
		for _, glob := range file.Globals {
			fmt.Fprintf(b, "- `%s`\n", formatLSPGlobalDetail(glob))
		}
		b.WriteByte('\n')
	}
	if len(file.Impls) > 0 {
		b.WriteString("### Implementations\n\n")
		for _, impl := range file.Impls {
			fmt.Fprintf(b, "- `%s`\n", formatLSPImplDetail(impl))
		}
		b.WriteByte('\n')
	}
	if len(file.Funcs) > 0 {
		b.WriteString("### Functions\n\n")
		for _, fn := range file.Funcs {
			if fn.ExtensionOf != "" || fn.Synthetic {
				continue
			}
			fmt.Fprintf(b, "- `%s`\n", formatLSPFuncDetail(fn))
		}
		b.WriteByte('\n')
	}
	if len(file.Extensions) > 0 {
		b.WriteString("### Extensions\n\n")
		for _, ext := range file.Extensions {
			fmt.Fprintf(b, "- `%s`\n", formatLSPTypeRef(ext.Target))
			for _, fn := range ext.Methods {
				fmt.Fprintf(b, "  - `%s`\n", formatLSPFuncDetail(fn))
			}
		}
		b.WriteByte('\n')
	}
	if len(file.Tests) > 0 {
		b.WriteString("### Tests\n\n")
		for _, test := range file.Tests {
			fmt.Fprintf(b, "- `%s`\n", test.Name)
		}
		b.WriteByte('\n')
	}
	doctestCount := countTetraDoctests(file.Src)
	if doctestCount > 0 {
		b.WriteString("### Doctests\n\n")
		for i := 1; i <= doctestCount; i++ {
			fmt.Fprintf(b, "- doctest %d\n", i)
		}
		b.WriteByte('\n')
	}
}

func isExperimentalModuleTitle(title string) bool {
	return title == "lib.experimental" || strings.HasPrefix(title, "lib.experimental.")
}

func countTetraDoctests(src []byte) int {
	count := 0
	for _, line := range strings.Split(string(src), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "```tetra doctest" {
			count++
			continue
		}
		if strings.HasPrefix(trimmed, "//") {
			comment := strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
			if comment == "```tetra doctest" {
				count++
			}
		}
	}
	return count
}
