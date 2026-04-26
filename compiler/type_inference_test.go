package compiler

import "testing"

func TestLocalTypeInference(t *testing.T) {
	src := []byte(`
fun main(): i32 {
  let x = 40
  let y: i32 = 2
  return x + y
}
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := Check(prog); err != nil {
		t.Fatalf("check: %v", err)
	}
}

func TestFlowLetIsImmutable(t *testing.T) {
	src := []byte(`
func main() -> i32:
  let x = 1
  x = 2
  return x
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := Check(prog); err == nil {
		t.Fatalf("expected immutable Flow let assignment to fail")
	}
}
