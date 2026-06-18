package plir

import "testing"

func findFunction(t *testing.T, prog *Program, name string) Function {
	t.Helper()
	for _, candidate := range prog.Funcs {
		if candidate.Name == name {
			return candidate
		}
	}
	t.Fatalf("missing PLIR function %s: %#v", name, prog.Funcs)
	return Function{}
}

func findValue(t *testing.T, fn Function, valueID string) Value {
	t.Helper()
	for _, value := range fn.Values {
		if value.ID == valueID {
			return value
		}
	}
	t.Fatalf("missing value %s in %s: %#v", valueID, fn.Name, fn.Values)
	return Value{}
}

func hasFactForValue(fn Function, kind FactKind, valueID string) bool {
	for _, fact := range fn.Facts {
		if fact.Kind == kind && fact.ValueID == valueID {
			return true
		}
	}
	return false
}

func findFactForValue(fn Function, kind FactKind, valueID string) (Fact, bool) {
	for _, fact := range fn.Facts {
		if fact.Kind == kind && fact.ValueID == valueID {
			return fact, true
		}
	}
	return Fact{}, false
}

func hasOperationKind(fn Function, kind OperationKind) bool {
	for _, op := range fn.Ops {
		if op.Kind == kind {
			return true
		}
	}
	return false
}
