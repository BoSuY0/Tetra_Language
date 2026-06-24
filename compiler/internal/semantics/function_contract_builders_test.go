package semantics

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionFuncSigCompositeLiteralsStayCentralized(t *testing.T) {
	root := repoRootForTest(t)
	allow := map[string]bool{
		filepath.ToSlash(filepath.Join("compiler", "internal", "semantics", "function_contract_builders.go")): true,
	}
	var offenders []string
	for _, relRoot := range []string{
		filepath.Join("compiler", "internal", "semantics"),
		filepath.Join("compiler", "internal", "module"),
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
				if !ok {
					return true
				}
				if isFuncSigCompositeType(lit.Type) {
					pos := fileSet.Position(lit.Lbrace)
					offenders = append(offenders, rel+":"+itoa(pos.Line))
					return true
				}
				if isMapStringFuncSigCompositeType(lit.Type) {
					for _, elt := range lit.Elts {
						kv, ok := elt.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						valueLit, ok := kv.Value.(*ast.CompositeLit)
						if !ok {
							continue
						}
						pos := fileSet.Position(valueLit.Lbrace)
						offenders = append(offenders, rel+":"+itoa(pos.Line))
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", relRoot, err)
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("production direct FuncSig composite literals must use builders:\n%s", strings.Join(offenders, "\n"))
	}
}

func isMapStringFuncSigCompositeType(expr ast.Expr) bool {
	mapType, ok := expr.(*ast.MapType)
	if !ok || !isStringIdent(mapType.Key) {
		return false
	}
	return isFuncSigCompositeType(mapType.Value)
}

func isStringIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "string"
}

func isFuncSigCompositeType(expr ast.Expr) bool {
	switch typ := expr.(type) {
	case *ast.Ident:
		return typ.Name == "FuncSig"
	case *ast.SelectorExpr:
		return typ.Sel.Name == "FuncSig"
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
