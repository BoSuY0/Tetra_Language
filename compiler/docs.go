package compiler

import (
	"bytes"
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
	var b bytes.Buffer
	b.WriteString("# Tetra API Docs\n\n")
	for _, file := range parsed {
		writeFileAPIDocs(&b, file)
	}
	return b.Bytes(), nil
}

func GenerateAPIDocsFromSource(src []byte, filename string) ([]byte, error) {
	file, err := frontend.ParseFile(src, filename)
	if err != nil {
		return nil, err
	}
	file.Path = filename
	var b bytes.Buffer
	b.WriteString("# Tetra API Docs\n\n")
	writeFileAPIDocs(&b, file)
	return b.Bytes(), nil
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
			if strings.HasSuffix(path, ".tetra") {
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
			if strings.HasSuffix(p, ".tetra") {
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

func writeFileAPIDocs(b *bytes.Buffer, file *frontend.FileAST) {
	title := file.Path
	if file.Module != "" {
		title = file.Module
	}
	fmt.Fprintf(b, "## %s\n\n", title)
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
				fmt.Fprintf(b, "  - `%s`\n", formatFuncSigDecl(req))
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
			if fn.ExtensionOf != "" {
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
}
