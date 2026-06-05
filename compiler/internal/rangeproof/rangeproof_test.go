package rangeproof

import (
	"reflect"
	"testing"
)

func TestLessThanLenRangeDerivation(t *testing.T) {
	r := LessThanLen("i", "xs")
	if r.Value != "i" {
		t.Fatalf("value = %q, want i", r.Value)
	}
	if r.Lower != Const(0) || !r.InclusiveLower {
		t.Fatalf("lower = %+v inclusive=%v, want inclusive const 0", r.Lower, r.InclusiveLower)
	}
	if r.Upper != Symbol("xs.len") || r.InclusiveUpper {
		t.Fatalf("upper = %+v inclusive=%v, want exclusive xs.len", r.Upper, r.InclusiveUpper)
	}
	if !reflect.DeepEqual(r.Derivation, []string{"non_negative", "less_than_len"}) {
		t.Fatalf("derivation = %#v", r.Derivation)
	}
}

func TestLessEqualLenMinusOneRangeDerivation(t *testing.T) {
	r := LessEqualLenMinusOne("i", "xs")
	if r.Upper != SymbolMinus("xs.len", 1) || !r.InclusiveUpper {
		t.Fatalf("upper = %+v inclusive=%v, want inclusive xs.len - 1", r.Upper, r.InclusiveUpper)
	}
	if !reflect.DeepEqual(r.Derivation, []string{"non_negative", "less_equal_len_minus_one"}) {
		t.Fatalf("derivation = %#v", r.Derivation)
	}
}

func TestAddSubByConstantPreserveDerivation(t *testing.T) {
	r := LessThanLen("i", "xs")
	added := AddConst(r, "j", 2)
	if added.Value != "j" || added.Lower != Const(2) || added.Upper != SymbolPlus("xs.len", 2) {
		t.Fatalf("added = %+v", added)
	}
	if got := added.Derivation[len(added.Derivation)-1]; got != "add_const:2" {
		t.Fatalf("added derivation = %#v", added.Derivation)
	}
	subbed := SubConst(added, "k", 2)
	if subbed.Value != "k" || subbed.Lower != Const(0) || subbed.Upper != Symbol("xs.len") {
		t.Fatalf("subbed = %+v", subbed)
	}
	if got := subbed.Derivation[len(subbed.Derivation)-1]; got != "sub_const:2" {
		t.Fatalf("subbed derivation = %#v", subbed.Derivation)
	}
}

func TestAddSubByConstantOverflowBecomesUnknown(t *testing.T) {
	r := Range{
		Value:          "i",
		Known:          true,
		Lower:          Const(maxInt64),
		Upper:          Const(maxInt64),
		InclusiveLower: true,
		InclusiveUpper: true,
		Derivation:     []string{"constant_range"},
	}
	if got := AddConst(r, "j", 1); got.Known {
		t.Fatalf("overflowing add range = %+v, want unknown", got)
	}
	if got := SubConst(r, "j", -1); got.Known {
		t.Fatalf("overflowing sub range = %+v, want unknown", got)
	}
}

func TestJoinAndWidenKeepCommonFacts(t *testing.T) {
	a := LessThanLen("i", "xs")
	b := LessEqualLenMinusOne("i", "xs")
	joined := Join(a, b)
	if !joined.Known || joined.Value != "i" {
		t.Fatalf("joined = %+v, want known i range", joined)
	}
	if joined.Lower != Const(0) || joined.Upper != Symbol("xs.len") || joined.InclusiveUpper {
		t.Fatalf("joined bounds = %+v", joined)
	}
	if got := joined.Derivation[len(joined.Derivation)-1]; got != "join" {
		t.Fatalf("joined derivation = %#v", joined.Derivation)
	}
	widened := Widen(a, b)
	if !reflect.DeepEqual(widened, joined) {
		t.Fatalf("widened = %+v, want join %+v", widened, joined)
	}
}

func TestClampFactsAreRepresentable(t *testing.T) {
	min := MinClamp("n", Const(0), Symbol("xs.len"))
	if min.Lower != Const(0) || min.Upper != Symbol("xs.len") || !min.InclusiveUpper {
		t.Fatalf("min clamp range = %+v", min)
	}
	max := MaxClamp("n", Const(0), Symbol("xs.len"))
	if max.Lower != Const(0) || max.Upper != Symbol("xs.len") || !max.InclusiveLower {
		t.Fatalf("max clamp range = %+v", max)
	}
}
