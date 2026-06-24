package plir

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionFunctionSummaryConstructionStaysCentralized(t *testing.T) {
	root := repoRootForTest(t)
	allow := map[string]bool{
		filepath.ToSlash(filepath.Join("compiler", "internal", "plir", "plir.go")): true,
	}
	var offenders []string
	for _, relRoot := range []string{
		filepath.Join("compiler", "internal", "plir"),
		filepath.Join("compiler", "internal", "memoryfacts"),
		filepath.Join("compiler", "internal", "allocplan"),
	} {
		absRoot := filepath.Join(root, relRoot)
		err := filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if allow[rel] {
				return nil
			}
			fileSet := token.NewFileSet()
			file, err := parser.ParseFile(fileSet, path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(node ast.Node) bool {
				lit, ok := node.(*ast.CompositeLit)
				if !ok || !isFunctionSummaryCompositeType(lit.Type) {
					return true
				}
				pos := fileSet.Position(lit.Lbrace)
				offenders = append(offenders, rel+":"+itoa(pos.Line))
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", relRoot, err)
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("production direct FunctionSummary composite literals must use FunctionSummaryFromFuncSig:\n%s", strings.Join(offenders, "\n"))
	}
}

func isFunctionSummaryCompositeType(expr ast.Expr) bool {
	switch typ := expr.(type) {
	case *ast.Ident:
		return typ.Name == "FunctionSummary"
	case *ast.SelectorExpr:
		return typ.Sel.Name == "FunctionSummary"
	default:
		return false
	}
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			t.Fatalf("could not find repo root from %s", dir)
		}
		dir = next
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
