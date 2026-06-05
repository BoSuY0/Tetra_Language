package rangeproof

import "fmt"

const maxInt64 = int64(^uint64(0) >> 1)
const minInt64 = -maxInt64 - 1

type BoundKind string

const (
	BoundUnknown     BoundKind = "unknown"
	BoundConst       BoundKind = "const"
	BoundSymbol      BoundKind = "symbol"
	BoundSymbolPlus  BoundKind = "symbol_plus"
	BoundSymbolMinus BoundKind = "symbol_minus"
)

type Bound struct {
	Kind   BoundKind
	Symbol string
	Const  int64
}

type Range struct {
	Value          string
	Known          bool
	Lower          Bound
	Upper          Bound
	InclusiveLower bool
	InclusiveUpper bool
	Derivation     []string
}

func Unknown(value string) Range {
	return Range{Value: value}
}

func Const(value int64) Bound {
	return Bound{Kind: BoundConst, Const: value}
}

func Symbol(name string) Bound {
	if name == "" {
		return Bound{Kind: BoundUnknown}
	}
	return Bound{Kind: BoundSymbol, Symbol: name}
}

func SymbolPlus(name string, delta int64) Bound {
	if delta == 0 {
		return Symbol(name)
	}
	return Bound{Kind: BoundSymbolPlus, Symbol: name, Const: delta}
}

func SymbolMinus(name string, delta int64) Bound {
	if delta == 0 {
		return Symbol(name)
	}
	return Bound{Kind: BoundSymbolMinus, Symbol: name, Const: delta}
}

func LessThanLen(value string, base string) Range {
	return Range{
		Value:          value,
		Known:          true,
		Lower:          Const(0),
		Upper:          Symbol(base + ".len"),
		InclusiveLower: true,
		InclusiveUpper: false,
		Derivation:     []string{"non_negative", "less_than_len"},
	}
}

func LessEqualLenMinusOne(value string, base string) Range {
	return Range{
		Value:          value,
		Known:          true,
		Lower:          Const(0),
		Upper:          SymbolMinus(base+".len", 1),
		InclusiveLower: true,
		InclusiveUpper: true,
		Derivation:     []string{"non_negative", "less_equal_len_minus_one"},
	}
}

func AddConst(r Range, value string, delta int64) Range {
	if !r.Known {
		return Unknown(value)
	}
	out := r
	out.Value = value
	out.Lower = addBoundConst(out.Lower, delta)
	out.Upper = addBoundConst(out.Upper, delta)
	if !boundKnown(out.Lower) || !boundKnown(out.Upper) {
		return Unknown(value)
	}
	out.Derivation = appendDerivation(out.Derivation, fmt.Sprintf("add_const:%d", delta))
	return out
}

func SubConst(r Range, value string, delta int64) Range {
	if !r.Known {
		return Unknown(value)
	}
	out := r
	out.Value = value
	out.Lower = addBoundConst(out.Lower, -delta)
	out.Upper = addBoundConst(out.Upper, -delta)
	if !boundKnown(out.Lower) || !boundKnown(out.Upper) {
		return Unknown(value)
	}
	out.Derivation = appendDerivation(out.Derivation, fmt.Sprintf("sub_const:%d", delta))
	return out
}

func MinClamp(value string, lower Bound, upper Bound) Range {
	return Range{
		Value:          value,
		Known:          boundKnown(lower) && boundKnown(upper),
		Lower:          lower,
		Upper:          upper,
		InclusiveLower: true,
		InclusiveUpper: true,
		Derivation:     []string{"min_max_clamp"},
	}
}

func MaxClamp(value string, lower Bound, upper Bound) Range {
	return Range{
		Value:          value,
		Known:          boundKnown(lower) && boundKnown(upper),
		Lower:          lower,
		Upper:          upper,
		InclusiveLower: true,
		InclusiveUpper: true,
		Derivation:     []string{"min_max_clamp"},
	}
}

func Join(a Range, b Range) Range {
	if !a.Known || !b.Known || a.Value != b.Value {
		return Unknown(a.Value)
	}
	if a.Lower != b.Lower || a.InclusiveLower != b.InclusiveLower {
		return Unknown(a.Value)
	}
	upper, inclusiveUpper, ok := joinUpper(a.Upper, a.InclusiveUpper, b.Upper, b.InclusiveUpper)
	if !ok {
		return Unknown(a.Value)
	}
	return Range{
		Value:          a.Value,
		Known:          true,
		Lower:          a.Lower,
		Upper:          upper,
		InclusiveLower: a.InclusiveLower,
		InclusiveUpper: inclusiveUpper,
		Derivation:     appendDerivation(mergeDerivation(a.Derivation, b.Derivation), "join"),
	}
}

func Widen(previous Range, next Range) Range {
	return Join(previous, next)
}

func addBoundConst(bound Bound, delta int64) Bound {
	if delta == 0 {
		return bound
	}
	switch bound.Kind {
	case BoundConst:
		value, ok := checkedAddInt64(bound.Const, delta)
		if !ok {
			return Bound{Kind: BoundUnknown}
		}
		return Const(value)
	case BoundSymbol:
		if delta > 0 {
			return SymbolPlus(bound.Symbol, delta)
		}
		return SymbolMinus(bound.Symbol, -delta)
	case BoundSymbolPlus:
		value, ok := checkedAddInt64(bound.Const, delta)
		if !ok {
			return Bound{Kind: BoundUnknown}
		}
		return symbolOffset(bound.Symbol, value)
	case BoundSymbolMinus:
		value, ok := checkedAddInt64(-bound.Const, delta)
		if !ok {
			return Bound{Kind: BoundUnknown}
		}
		return symbolOffset(bound.Symbol, value)
	default:
		return Bound{Kind: BoundUnknown}
	}
}

func checkedAddInt64(left int64, right int64) (int64, bool) {
	if right > 0 && left > maxInt64-right {
		return 0, false
	}
	if right < 0 && left < minInt64-right {
		return 0, false
	}
	return left + right, true
}

func symbolOffset(symbol string, delta int64) Bound {
	switch {
	case delta == 0:
		return Symbol(symbol)
	case delta > 0:
		return SymbolPlus(symbol, delta)
	default:
		return SymbolMinus(symbol, -delta)
	}
}

func joinUpper(a Bound, aInclusive bool, b Bound, bInclusive bool) (Bound, bool, bool) {
	if a == b {
		return a, aInclusive || bInclusive, true
	}
	if a.Kind == BoundSymbol && b.Kind == BoundSymbolMinus && a.Symbol == b.Symbol && b.Const >= 0 {
		return a, false, true
	}
	if b.Kind == BoundSymbol && a.Kind == BoundSymbolMinus && b.Symbol == a.Symbol && a.Const >= 0 {
		return b, false, true
	}
	return Bound{}, false, false
}

func boundKnown(bound Bound) bool {
	return bound.Kind != "" && bound.Kind != BoundUnknown
}

func appendDerivation(in []string, item string) []string {
	out := append([]string(nil), in...)
	if item != "" {
		out = append(out, item)
	}
	return out
}

func mergeDerivation(a []string, b []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(a)+len(b))
	for _, item := range append(append([]string(nil), a...), b...) {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}
