package frontend

import "testing"

func TestParseArrayTypeRefInParamAndLocal(t *testing.T) {
	src := []byte(`func sum(seed: [3]Int) -> Int:
    var xs: [3]Int = seed
    xs[0] = xs[0]
    return xs[0]
`)

	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Funcs) != 1 {
		t.Fatalf("func count = %d, want 1", len(prog.Funcs))
	}
	fn := prog.Funcs[0]
	if len(fn.Params) != 1 {
		t.Fatalf("param count = %d, want 1", len(fn.Params))
	}
	paramType := fn.Params[0].Type
	if paramType.Kind != TypeRefArray || paramType.Len != 3 || paramType.Elem == nil {
		t.Fatalf("param type = %#v, want [3]Int", paramType)
	}
	if paramType.Elem.Kind != TypeRefNamed || paramType.Elem.Name != "Int" {
		t.Fatalf("param elem = %#v, want Int", paramType.Elem)
	}

	stmt, ok := fn.Body[0].(*LetStmt)
	if !ok {
		t.Fatalf("stmt[0] = %T, want *LetStmt", fn.Body[0])
	}
	localType := stmt.Type
	if localType.Kind != TypeRefArray || localType.Len != 3 || localType.Elem == nil {
		t.Fatalf("local type = %#v, want [3]Int", localType)
	}
	if localType.Elem.Kind != TypeRefNamed || localType.Elem.Name != "Int" {
		t.Fatalf("local elem = %#v, want Int", localType.Elem)
	}
}
