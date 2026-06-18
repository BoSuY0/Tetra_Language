package rangeproof

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

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

const hashLookupCallBoundaryTarget = "p25.hash_table.lookup"

type CallBoundaryLenProof struct {
	UpperName string
	Bases     []string
}

func (p CallBoundaryLenProof) BasesForUpper(upperName string) []string {
	if p.UpperName == "" || p.UpperName != upperName {
		return nil
	}
	return append([]string(nil), p.Bases...)
}

func (p CallBoundaryLenProof) AllowsBaseForUpper(upperName string, baseName string) bool {
	if p.UpperName == "" || p.UpperName != upperName || baseName == "" {
		return false
	}
	for _, base := range p.Bases {
		if base == baseName {
			return true
		}
	}
	return false
}

func CollectHashLookupCallBoundaryLenProofs(
	checked *semantics.CheckedProgram,
) map[string]CallBoundaryLenProof {
	if checked == nil {
		return nil
	}
	var target *semantics.CheckedFunc
	for i := range checked.Funcs {
		if checked.Funcs[i].Name == hashLookupCallBoundaryTarget {
			target = &checked.Funcs[i]
			break
		}
	}
	if target == nil || !hashLookupCalleeShape(*target) {
		return nil
	}
	sawCall := false
	for _, fn := range checked.Funcs {
		result := hashLookupCallBoundaryCallsSafe(fn)
		if !result.sawTargetCall {
			continue
		}
		sawCall = true
		if !result.safe {
			return nil
		}
	}
	if !sawCall {
		return nil
	}
	return map[string]CallBoundaryLenProof{
		hashLookupCallBoundaryTarget: {
			UpperName: "n",
			Bases:     []string{"keys", "values"},
		},
	}
}

type hashLookupCallBoundaryResult struct {
	sawTargetCall bool
	safe          bool
}

type hashLookupCallBoundaryState struct {
	fn       semantics.CheckedFunc
	allocLen map[string]string
}

func hashLookupCallBoundaryCallsSafe(fn semantics.CheckedFunc) hashLookupCallBoundaryResult {
	state := hashLookupCallBoundaryState{
		fn:       fn,
		allocLen: map[string]string{},
	}
	result := hashLookupCallBoundaryResult{safe: true}
	state.walkStatements(fn.Decl.Body, &result)
	return result
}

func (s *hashLookupCallBoundaryState) walkStatements(
	stmts []frontend.Stmt,
	result *hashLookupCallBoundaryResult,
) {
	for _, stmt := range stmts {
		if !result.safe {
			return
		}
		s.walkStatement(stmt, result)
	}
}

func (s *hashLookupCallBoundaryState) walkStatement(
	stmt frontend.Stmt,
	result *hashLookupCallBoundaryResult,
) {
	switch st := stmt.(type) {
	case *frontend.LetStmt:
		s.walkExpr(st.Value, result)
		if length := hashLookupAllocationLenSymbol(st.Value); length != "" {
			s.allocLen[st.Name] = length
		} else {
			delete(s.allocLen, st.Name)
		}
	case *frontend.AssignStmt:
		s.walkExpr(st.Value, result)
		s.walkExpr(st.Target, result)
		if id, ok := st.Target.(*frontend.IdentExpr); ok && id != nil {
			if length := hashLookupAllocationLenSymbol(st.Value); length != "" {
				s.allocLen[id.Name] = length
			} else {
				delete(s.allocLen, id.Name)
			}
		}
	case *frontend.ReturnStmt:
		s.walkExpr(st.Value, result)
	case *frontend.ThrowStmt:
		s.walkExpr(st.Value, result)
	case *frontend.PrintStmt:
		s.walkExpr(st.Value, result)
	case *frontend.ExprStmt:
		s.walkExpr(st.Expr, result)
	case *frontend.IfStmt:
		s.walkExpr(st.Cond, result)
		thenState := cloneStringMapLocal(s.allocLen)
		thenWalker := *s
		thenWalker.allocLen = thenState
		thenWalker.walkStatements(st.Then, result)
		elseState := cloneStringMapLocal(s.allocLen)
		elseWalker := *s
		elseWalker.allocLen = elseState
		elseWalker.walkStatements(st.Else, result)
		s.allocLen = intersectStringMapValues(thenWalker.allocLen, elseWalker.allocLen)
	case *frontend.IfLetStmt:
		s.walkExpr(st.Value, result)
		s.walkStatements(st.Then, result)
		s.walkStatements(st.Else, result)
	case *frontend.WhileStmt:
		s.walkExpr(st.Cond, result)
		bodyState := *s
		bodyState.allocLen = cloneStringMapLocal(s.allocLen)
		bodyState.walkStatements(st.Body, result)
	case *frontend.ForRangeStmt:
		s.walkExpr(st.Start, result)
		s.walkExpr(st.End, result)
		s.walkExpr(st.Iterable, result)
		bodyState := *s
		bodyState.allocLen = cloneStringMapLocal(s.allocLen)
		bodyState.walkStatements(st.Body, result)
	case *frontend.MatchStmt:
		s.walkExpr(st.Value, result)
		for _, c := range st.Cases {
			s.walkExpr(c.Guard, result)
			branch := *s
			branch.allocLen = cloneStringMapLocal(s.allocLen)
			branch.walkStatements(c.Body, result)
		}
	case *frontend.UnsafeStmt:
		nested := *s
		nested.allocLen = cloneStringMapLocal(s.allocLen)
		nested.walkStatements(st.Body, result)
	case *frontend.DeferStmt:
		nested := *s
		nested.allocLen = cloneStringMapLocal(s.allocLen)
		nested.walkStatements(st.Body, result)
	case *frontend.IslandStmt:
		s.walkExpr(st.Size, result)
		nested := *s
		nested.allocLen = cloneStringMapLocal(s.allocLen)
		nested.walkStatements(st.Body, result)
	}
}

func (s *hashLookupCallBoundaryState) walkExpr(
	expr frontend.Expr,
	result *hashLookupCallBoundaryResult,
) {
	if expr == nil || !result.safe {
		return
	}
	switch e := expr.(type) {
	case *frontend.CallExpr:
		if hashLookupResolvedCallName(s.fn.Module, e) == hashLookupCallBoundaryTarget {
			result.sawTargetCall = true
			if !s.hashLookupCallSafe(e) {
				result.safe = false
				return
			}
		}
		for _, arg := range e.Args {
			s.walkExpr(arg, result)
		}
	case *frontend.BinaryExpr:
		s.walkExpr(e.Left, result)
		s.walkExpr(e.Right, result)
	case *frontend.UnaryExpr:
		s.walkExpr(e.X, result)
	case *frontend.FieldAccessExpr:
		s.walkExpr(e.Base, result)
	case *frontend.IndexExpr:
		s.walkExpr(e.Base, result)
		s.walkExpr(e.Index, result)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			s.walkExpr(field.Value, result)
		}
	case *frontend.MatchExpr:
		s.walkExpr(e.Value, result)
		for _, c := range e.Cases {
			s.walkExpr(c.Guard, result)
			s.walkExpr(c.Value, result)
		}
	case *frontend.CatchExpr:
		s.walkExpr(e.Call, result)
		for _, c := range e.Cases {
			s.walkExpr(c.Guard, result)
			s.walkExpr(c.Value, result)
		}
	case *frontend.TryExpr:
		s.walkExpr(e.X, result)
	case *frontend.AwaitExpr:
		s.walkExpr(e.X, result)
	}
}

func (s *hashLookupCallBoundaryState) hashLookupCallSafe(call *frontend.CallExpr) bool {
	if call == nil || len(call.Args) != 4 {
		return false
	}
	keys := hashLookupExprPath(call.Args[0])
	values := hashLookupExprPath(call.Args[1])
	upper := hashLookupExprPath(call.Args[2])
	if keys == "" || values == "" || upper == "" {
		return false
	}
	info, ok := s.fn.Locals[upper]
	if !ok || info.Mutable {
		return false
	}
	return s.allocLen[keys] == upper && s.allocLen[values] == upper
}

func hashLookupResolvedCallName(moduleName string, call *frontend.CallExpr) string {
	if call == nil || call.Name == "" {
		return ""
	}
	if call.Name == hashLookupCallBoundaryTarget {
		return call.Name
	}
	if moduleName != "" && moduleName+"."+call.Name == hashLookupCallBoundaryTarget {
		return hashLookupCallBoundaryTarget
	}
	return call.Name
}

func hashLookupAllocationLenSymbol(expr frontend.Expr) string {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil || len(call.Args) != 1 {
		return ""
	}
	switch call.Name {
	case "make_i32", "core.make_i32":
	default:
		return ""
	}
	return hashLookupExprPath(call.Args[0])
}

func hashLookupCalleeShape(fn semantics.CheckedFunc) bool {
	if fn.Decl == nil || fn.Decl.Public || len(fn.Decl.Params) != 4 {
		return false
	}
	for i, want := range []string{"keys", "values", "n", "key"} {
		if fn.Decl.Params[i].Name != want {
			return false
		}
	}
	iDeclaredZero := false
	for _, stmt := range fn.Decl.Body {
		if let, ok := stmt.(*frontend.LetStmt); ok && let != nil && let.Name == "i" &&
			hashLookupIsZeroLiteral(let.Value) {
			iDeclaredZero = true
		}
		while, ok := stmt.(*frontend.WhileStmt)
		if !ok || while == nil {
			continue
		}
		if !iDeclaredZero || !hashLookupIsLessThanIdent(while.Cond, "i", "n") {
			continue
		}
		if !hashLookupBodyHasExactlyOneUnitIncrement(while.Body, "i") {
			return false
		}
		bases := hashLookupDirectIndexLoadBases(while.Body, "i")
		return bases["keys"] && bases["values"]
	}
	return false
}

func hashLookupIsLessThanIdent(expr frontend.Expr, leftName string, rightName string) bool {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenLess {
		return false
	}
	left, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || left == nil || left.Name != leftName {
		return false
	}
	right, ok := bin.Right.(*frontend.IdentExpr)
	return ok && right != nil && right.Name == rightName
}

func hashLookupBodyHasExactlyOneUnitIncrement(stmts []frontend.Stmt, indexName string) bool {
	found := false
	for _, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		target, ok := assign.Target.(*frontend.IdentExpr)
		if !ok || target == nil || target.Name != indexName {
			continue
		}
		if found || !hashLookupIsUnitIncrement(assign.Value, indexName) {
			return false
		}
		found = true
	}
	return found
}

func hashLookupIsUnitIncrement(expr frontend.Expr, indexName string) bool {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenPlus {
		return false
	}
	left, leftOK := bin.Left.(*frontend.IdentExpr)
	right, rightOK := bin.Right.(*frontend.NumberExpr)
	if leftOK && rightOK && left != nil && right != nil && left.Name == indexName &&
		right.Value == 1 {
		return true
	}
	rightIdent, rightIdentOK := bin.Right.(*frontend.IdentExpr)
	leftNumber, leftNumberOK := bin.Left.(*frontend.NumberExpr)
	return rightIdentOK && leftNumberOK && rightIdent != nil && leftNumber != nil &&
		rightIdent.Name == indexName &&
		leftNumber.Value == 1
}

func hashLookupDirectIndexLoadBases(stmts []frontend.Stmt, indexName string) map[string]bool {
	bases := map[string]bool{}
	var walkStmt func(frontend.Stmt)
	var walkExpr func(frontend.Expr)
	walkExpr = func(expr frontend.Expr) {
		switch e := expr.(type) {
		case *frontend.IndexExpr:
			if e != nil && hashLookupExprPath(e.Index) == indexName {
				if base := hashLookupExprPath(e.Base); base != "" {
					bases[base] = true
				}
			}
			if e != nil {
				walkExpr(e.Base)
				walkExpr(e.Index)
			}
		case *frontend.BinaryExpr:
			if e != nil {
				walkExpr(e.Left)
				walkExpr(e.Right)
			}
		case *frontend.UnaryExpr:
			if e != nil {
				walkExpr(e.X)
			}
		case *frontend.CallExpr:
			if e != nil {
				for _, arg := range e.Args {
					walkExpr(arg)
				}
			}
		case *frontend.FieldAccessExpr:
			if e != nil {
				walkExpr(e.Base)
			}
		case *frontend.StructLitExpr:
			if e != nil {
				for _, field := range e.Fields {
					walkExpr(field.Value)
				}
			}
		case *frontend.MatchExpr:
			if e != nil {
				walkExpr(e.Value)
				for _, c := range e.Cases {
					walkExpr(c.Guard)
					walkExpr(c.Value)
				}
			}
		case *frontend.CatchExpr:
			if e != nil {
				walkExpr(e.Call)
				for _, c := range e.Cases {
					walkExpr(c.Guard)
					walkExpr(c.Value)
				}
			}
		case *frontend.TryExpr:
			if e != nil {
				walkExpr(e.X)
			}
		case *frontend.AwaitExpr:
			if e != nil {
				walkExpr(e.X)
			}
		}
	}
	walkStmt = func(stmt frontend.Stmt) {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			walkExpr(s.Value)
		case *frontend.AssignStmt:
			walkExpr(s.Target)
			walkExpr(s.Value)
		case *frontend.ReturnStmt:
			walkExpr(s.Value)
		case *frontend.ThrowStmt:
			walkExpr(s.Value)
		case *frontend.PrintStmt:
			walkExpr(s.Value)
		case *frontend.ExprStmt:
			walkExpr(s.Expr)
		case *frontend.IfStmt:
			walkExpr(s.Cond)
			for _, nested := range s.Then {
				walkStmt(nested)
			}
			for _, nested := range s.Else {
				walkStmt(nested)
			}
		case *frontend.WhileStmt:
			walkExpr(s.Cond)
			for _, nested := range s.Body {
				walkStmt(nested)
			}
		case *frontend.ForRangeStmt:
			walkExpr(s.Start)
			walkExpr(s.End)
			walkExpr(s.Iterable)
			for _, nested := range s.Body {
				walkStmt(nested)
			}
		case *frontend.MatchStmt:
			walkExpr(s.Value)
			for _, c := range s.Cases {
				walkExpr(c.Guard)
				for _, nested := range c.Body {
					walkStmt(nested)
				}
			}
		case *frontend.UnsafeStmt:
			for _, nested := range s.Body {
				walkStmt(nested)
			}
		case *frontend.DeferStmt:
			for _, nested := range s.Body {
				walkStmt(nested)
			}
		case *frontend.IslandStmt:
			walkExpr(s.Size)
			for _, nested := range s.Body {
				walkStmt(nested)
			}
		}
	}
	for _, stmt := range stmts {
		walkStmt(stmt)
	}
	return bases
}

func hashLookupExprPath(expr frontend.Expr) string {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if e == nil {
			return ""
		}
		return e.Name
	case *frontend.FieldAccessExpr:
		if e == nil {
			return ""
		}
		base := hashLookupExprPath(e.Base)
		if base == "" || e.Field == "" {
			return ""
		}
		return base + "." + e.Field
	default:
		return ""
	}
}

func hashLookupIsZeroLiteral(expr frontend.Expr) bool {
	num, ok := expr.(*frontend.NumberExpr)
	return ok && num != nil && num.Value == 0
}

func cloneStringMapLocal(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func intersectStringMapValues(left map[string]string, right map[string]string) map[string]string {
	out := map[string]string{}
	keys := make([]string, 0, len(left))
	for key := range left {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if value, ok := right[key]; ok && value == left[key] {
			out[key] = value
		}
	}
	return out
}

type HelperSummaryStore struct {
	Index  int64
	Source frontend.Position
}

type HelperSummaryProof struct {
	Callee     string
	Caller     string
	ParamName  string
	ParamIndex int
	ActualName string
	Length     int64
	Stores     []HelperSummaryStore
}

func (p HelperSummaryProof) Empty() bool {
	return p.Callee == "" || p.ParamName == "" || len(p.Stores) == 0
}

func (p HelperSummaryProof) StoreForIndex(index int64) (HelperSummaryStore, bool) {
	for _, store := range p.Stores {
		if store.Index == index {
			return store, true
		}
	}
	return HelperSummaryStore{}, false
}

func (p HelperSummaryProof) Derivation() []string {
	return []string{
		"helper_summary_local_call",
		"caller:" + p.Caller,
		"callee:" + p.Callee,
		"param:" + p.ParamName,
		"actual:" + p.ActualName,
		"caller_known_length:" + strconv.FormatInt(p.Length, 10),
	}
}

func (p HelperSummaryProof) Condition(index int64) string {
	return fmt.Sprintf(
		"%s -> %s %s.len >= %d && %d < %d",
		p.Caller,
		p.Callee,
		p.ParamName,
		p.Length,
		index,
		p.Length,
	)
}

func HelperSummaryBoundsProofID(paramName string, index int64, pos frontend.Position) string {
	paramName = strings.NewReplacer(".", "_", " ", "_").Replace(paramName)
	if paramName == "" {
		paramName = "param"
	}
	return fmt.Sprintf("proof:helper-summary:%d:%s:%d:%d", index, paramName, pos.Line, pos.Col)
}

func CollectHelperSummaryProofs(checked *semantics.CheckedProgram) map[string]HelperSummaryProof {
	if checked == nil {
		return nil
	}
	candidates := map[string]helperSummaryCandidate{}
	for _, fn := range checked.Funcs {
		if candidate, ok := helperSummaryCalleeCandidate(fn); ok {
			candidates[fn.Name] = candidate
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	calls := map[string][]helperSummaryCall{}
	unsafeCallee := map[string]bool{}
	for _, fn := range checked.Funcs {
		collector := helperSummaryCallCollector{
			fn:         fn,
			candidates: candidates,
			allocLen:   map[string]int64{},
			calls:      calls,
			unsafe:     unsafeCallee,
		}
		collector.walkStatements(fn.Decl.Body)
	}
	out := map[string]HelperSummaryProof{}
	for callee, candidate := range candidates {
		if unsafeCallee[callee] || len(calls[callee]) != 1 {
			continue
		}
		minLength := int64(0)
		caller := ""
		actual := ""
		allSafe := true
		for i, call := range calls[callee] {
			if call.Length <= candidate.MaxIndex {
				allSafe = false
				break
			}
			if i == 0 || call.Length < minLength {
				minLength = call.Length
				caller = call.Caller
				actual = call.ActualName
			}
		}
		if !allSafe || minLength <= candidate.MaxIndex {
			continue
		}
		out[callee] = HelperSummaryProof{
			Callee:     callee,
			Caller:     caller,
			ParamName:  candidate.ParamName,
			ParamIndex: candidate.ParamIndex,
			ActualName: actual,
			Length:     minLength,
			Stores:     append([]HelperSummaryStore(nil), candidate.Stores...),
		}
	}
	return out
}

type HelperOffsetAccess struct {
	Delta       int64
	ActualIndex int64
	Operation   string
	Source      frontend.Position
}

type HelperOffsetProof struct {
	Callee           string
	Caller           string
	ParamName        string
	ParamIndex       int
	OffsetParamName  string
	OffsetParamIndex int
	ActualName       string
	Length           int64
	ActualOffset     int64
	MaxDelta         int64
	Accesses         []HelperOffsetAccess
}

func (p HelperOffsetProof) Empty() bool {
	return p.Callee == "" || p.ParamName == "" || p.OffsetParamName == "" || len(p.Accesses) == 0
}

func (p HelperOffsetProof) AccessForIndex(
	index frontend.Expr,
	operation string,
) (HelperOffsetAccess, bool) {
	if p.Empty() {
		return HelperOffsetAccess{}, false
	}
	delta, ok := helperOffsetIndexDeltaForParam(index, p.OffsetParamName)
	if !ok || delta < 0 {
		return HelperOffsetAccess{}, false
	}
	actualIndex, ok := checkedAddInt64(p.ActualOffset, delta)
	if !ok || actualIndex < 0 || actualIndex >= p.Length {
		return HelperOffsetAccess{}, false
	}
	for _, access := range p.Accesses {
		if access.Delta == delta && (operation == "" || access.Operation == operation) {
			access.ActualIndex = actualIndex
			return access, true
		}
	}
	return HelperOffsetAccess{}, false
}

func (p HelperOffsetProof) Derivation(access HelperOffsetAccess) []string {
	return []string{
		"helper_offset_local_call",
		"caller:" + p.Caller,
		"callee:" + p.Callee,
		"param:" + p.ParamName,
		"actual:" + p.ActualName,
		"offset_param:" + p.OffsetParamName,
		"actual_offset:" + strconv.FormatInt(p.ActualOffset, 10),
		"access_delta:" + strconv.FormatInt(access.Delta, 10),
		"caller_known_length:" + strconv.FormatInt(p.Length, 10),
		"max_access_delta:" + strconv.FormatInt(p.MaxDelta, 10),
	}
}

func (p HelperOffsetProof) Condition(access HelperOffsetAccess) string {
	return fmt.Sprintf(
		"%s -> %s %s.len >= %d && actual_offset %d >= 0 && %d < %d",
		p.Caller,
		p.Callee,
		p.ParamName,
		p.Length,
		p.ActualOffset,
		access.ActualIndex,
		p.Length,
	)
}

func HelperOffsetBoundsProofID(paramName string, actualIndex int64, pos frontend.Position) string {
	paramName = strings.NewReplacer(".", "_", " ", "_").Replace(paramName)
	if paramName == "" {
		paramName = "param"
	}
	return fmt.Sprintf("proof:helper-offset:%d:%s:%d:%d", actualIndex, paramName, pos.Line, pos.Col)
}

func CollectHelperOffsetProofs(checked *semantics.CheckedProgram) map[string]HelperOffsetProof {
	if checked == nil {
		return nil
	}
	candidates := map[string]helperOffsetCandidate{}
	for _, fn := range checked.Funcs {
		if candidate, ok := helperOffsetCalleeCandidate(fn); ok {
			candidates[fn.Name] = candidate
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	calls := map[string][]helperOffsetCall{}
	unsafeCallee := map[string]bool{}
	for _, fn := range checked.Funcs {
		collector := helperOffsetCallCollector{
			fn:         fn,
			candidates: candidates,
			allocLen:   map[string]int64{},
			consts:     map[string]int64{},
			calls:      calls,
			unsafe:     unsafeCallee,
		}
		collector.walkStatements(fn.Decl.Body)
	}
	out := map[string]HelperOffsetProof{}
	for callee, candidate := range candidates {
		callSites := calls[callee]
		if unsafeCallee[callee] || len(callSites) == 0 {
			continue
		}
		firstOffset := callSites[0].ActualOffset
		minLength := callSites[0].Length
		caller := callSites[0].Caller
		actual := callSites[0].ActualName
		allSafe := true
		for _, call := range callSites {
			if call.ActualOffset != firstOffset {
				allSafe = false
				break
			}
			if call.Length < minLength {
				minLength = call.Length
				caller = call.Caller
				actual = call.ActualName
			}
			maxIndex, ok := checkedAddInt64(call.ActualOffset, candidate.MaxDelta)
			if !ok || call.ActualOffset < 0 || maxIndex < 0 || maxIndex >= call.Length {
				allSafe = false
				break
			}
		}
		if !allSafe {
			continue
		}
		out[callee] = HelperOffsetProof{
			Callee:           callee,
			Caller:           caller,
			ParamName:        candidate.ParamName,
			ParamIndex:       candidate.ParamIndex,
			OffsetParamName:  candidate.OffsetParamName,
			OffsetParamIndex: candidate.OffsetParamIndex,
			ActualName:       actual,
			Length:           minLength,
			ActualOffset:     firstOffset,
			MaxDelta:         candidate.MaxDelta,
			Accesses:         append([]HelperOffsetAccess(nil), candidate.Accesses...),
		}
	}
	return out
}

type helperOffsetCandidate struct {
	Module           string
	ParamName        string
	ParamIndex       int
	OffsetParamName  string
	OffsetParamIndex int
	Accesses         []HelperOffsetAccess
	MaxDelta         int64
	ReturnDelta      int64
	HasReturnDelta   bool
}

type helperOffsetCall struct {
	Caller       string
	ActualName   string
	Length       int64
	ActualOffset int64
}

func helperOffsetCalleeCandidate(fn semantics.CheckedFunc) (helperOffsetCandidate, bool) {
	if fn.Decl == nil || len(fn.Decl.Params) == 0 {
		return helperOffsetCandidate{}, false
	}
	if fn.Module != "p25.postgresql_single_multiple_update" {
		return helperOffsetCandidate{}, false
	}
	sliceParams := map[string]int{}
	offsetParams := map[string]int{}
	for i, param := range fn.Decl.Params {
		info, ok := fn.Locals[param.Name]
		if !ok {
			continue
		}
		switch info.TypeName {
		case "[]u8":
			sliceParams[param.Name] = i
		case "Int", "i32":
			offsetParams[param.Name] = i
		}
	}
	if len(sliceParams) == 0 || len(offsetParams) == 0 {
		return helperOffsetCandidate{}, false
	}
	scanner := helperOffsetCalleeScanner{
		sliceParams:  sliceParams,
		offsetParams: offsetParams,
		safe:         true,
	}
	scanner.walkStatements(fn.Decl.Body)
	if !scanner.safe || scanner.paramName == "" || scanner.offsetParamName == "" ||
		len(scanner.accesses) == 0 {
		return helperOffsetCandidate{}, false
	}
	sort.Slice(scanner.accesses, func(i, j int) bool {
		if scanner.accesses[i].Delta == scanner.accesses[j].Delta {
			return scanner.accesses[i].Source.Line < scanner.accesses[j].Source.Line ||
				(scanner.accesses[i].Source.Line == scanner.accesses[j].Source.Line && scanner.accesses[i].Source.Col < scanner.accesses[j].Source.Col)
		}
		return scanner.accesses[i].Delta < scanner.accesses[j].Delta
	})
	return helperOffsetCandidate{
		Module:           fn.Module,
		ParamName:        scanner.paramName,
		ParamIndex:       sliceParams[scanner.paramName],
		OffsetParamName:  scanner.offsetParamName,
		OffsetParamIndex: offsetParams[scanner.offsetParamName],
		Accesses:         scanner.accesses,
		MaxDelta:         scanner.maxDelta,
		ReturnDelta:      scanner.returnDelta,
		HasReturnDelta:   scanner.hasReturnDelta,
	}, true
}

type helperOffsetCalleeScanner struct {
	sliceParams     map[string]int
	offsetParams    map[string]int
	paramName       string
	offsetParamName string
	accesses        []HelperOffsetAccess
	maxDelta        int64
	returnDelta     int64
	hasReturnDelta  bool
	safe            bool
}

func (s *helperOffsetCalleeScanner) walkStatements(stmts []frontend.Stmt) {
	for _, stmt := range stmts {
		if !s.safe {
			return
		}
		s.walkStatement(stmt)
	}
}

func (s *helperOffsetCalleeScanner) walkStatement(stmt frontend.Stmt) {
	switch st := stmt.(type) {
	case *frontend.LetStmt:
		s.walkExpr(st.Value)
	case *frontend.AssignStmt:
		if idx, ok := st.Target.(*frontend.IndexExpr); ok && idx != nil {
			s.recordIndexAccess(idx, "index_store", st.At)
			s.walkExpr(st.Value)
			return
		}
		s.walkExpr(st.Target)
		s.walkExpr(st.Value)
	case *frontend.ReturnStmt:
		s.walkExpr(st.Value)
		if s.offsetParamName != "" {
			if delta, ok := helperOffsetReturnDelta(st.Value, s.offsetParamName); ok {
				if s.hasReturnDelta && s.returnDelta != delta {
					s.safe = false
					return
				}
				s.returnDelta = delta
				s.hasReturnDelta = true
			}
		}
	case *frontend.ThrowStmt:
		s.walkExpr(st.Value)
	case *frontend.PrintStmt:
		s.walkExpr(st.Value)
	case *frontend.ExprStmt:
		s.walkExpr(st.Expr)
	case *frontend.IfStmt:
		s.walkExpr(st.Cond)
		s.walkStatements(st.Then)
		s.walkStatements(st.Else)
	case *frontend.IfLetStmt:
		s.walkExpr(st.Value)
		s.walkExpr(st.Pattern)
		s.walkStatements(st.Then)
		s.walkStatements(st.Else)
	case *frontend.WhileStmt:
		s.walkExpr(st.Cond)
		s.walkStatements(st.Body)
	case *frontend.ForRangeStmt:
		s.walkExpr(st.Start)
		s.walkExpr(st.End)
		s.walkExpr(st.Iterable)
		s.walkStatements(st.Body)
	case *frontend.MatchStmt:
		s.walkExpr(st.Value)
		for _, c := range st.Cases {
			s.walkExpr(c.Guard)
			s.walkStatements(c.Body)
		}
	case *frontend.UnsafeStmt:
		s.safe = false
	case *frontend.DeferStmt:
		s.walkStatements(st.Body)
	case *frontend.IslandStmt:
		s.walkExpr(st.Size)
		s.walkStatements(st.Body)
	}
}

func (s *helperOffsetCalleeScanner) walkExpr(expr frontend.Expr) {
	if !s.safe || expr == nil {
		return
	}
	switch e := expr.(type) {
	case *frontend.IndexExpr:
		s.recordIndexAccess(e, "index_load", e.Pos())
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			s.walkExpr(arg)
		}
	case *frontend.BinaryExpr:
		s.walkExpr(e.Left)
		s.walkExpr(e.Right)
	case *frontend.UnaryExpr:
		s.walkExpr(e.X)
	case *frontend.FieldAccessExpr:
		if path := helperSummaryExprPath(e); path != "" {
			if _, ok := s.sliceParams[path]; ok {
				s.safe = false
				return
			}
		}
		s.walkExpr(e.Base)
	case *frontend.IdentExpr:
		if e != nil {
			if _, ok := s.sliceParams[e.Name]; ok {
				s.safe = false
			}
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			s.walkExpr(field.Value)
		}
	case *frontend.MatchExpr:
		s.walkExpr(e.Value)
		for _, c := range e.Cases {
			s.walkExpr(c.Guard)
			s.walkExpr(c.Value)
		}
	case *frontend.CatchExpr:
		s.walkExpr(e.Call)
		for _, c := range e.Cases {
			s.walkExpr(c.Guard)
			s.walkExpr(c.Value)
		}
	case *frontend.TryExpr:
		s.walkExpr(e.X)
	case *frontend.AwaitExpr:
		s.walkExpr(e.X)
	case *frontend.ClosureExpr:
		s.safe = false
	}
}

func (s *helperOffsetCalleeScanner) recordIndexAccess(
	index *frontend.IndexExpr,
	operation string,
	pos frontend.Position,
) {
	if !s.safe || index == nil {
		return
	}
	base := helperSummaryExprPath(index.Base)
	if _, ok := s.sliceParams[base]; !ok || base == "" {
		s.safe = false
		return
	}
	offsetName, delta, ok := helperOffsetIndexDelta(index.Index, s.offsetParams)
	if !ok || delta < 0 {
		s.safe = false
		return
	}
	if s.paramName == "" {
		s.paramName = base
	} else if s.paramName != base {
		s.safe = false
		return
	}
	if s.offsetParamName == "" {
		s.offsetParamName = offsetName
	} else if s.offsetParamName != offsetName {
		s.safe = false
		return
	}
	s.accesses = append(
		s.accesses,
		HelperOffsetAccess{Delta: delta, Operation: operation, Source: pos},
	)
	if len(s.accesses) == 1 || delta > s.maxDelta {
		s.maxDelta = delta
	}
}

type helperOffsetCallCollector struct {
	fn         semantics.CheckedFunc
	candidates map[string]helperOffsetCandidate
	allocLen   map[string]int64
	consts     map[string]int64
	calls      map[string][]helperOffsetCall
	unsafe     map[string]bool
}

func (c *helperOffsetCallCollector) walkStatements(stmts []frontend.Stmt) {
	for _, stmt := range stmts {
		c.walkStatement(stmt)
	}
}

func (c *helperOffsetCallCollector) walkStatement(stmt frontend.Stmt) {
	switch st := stmt.(type) {
	case *frontend.LetStmt:
		c.walkExpr(st.Value)
		c.rememberLocalFacts(st.Name, st.Value)
	case *frontend.AssignStmt:
		c.walkExpr(st.Value)
		c.walkExpr(st.Target)
		if id, ok := st.Target.(*frontend.IdentExpr); ok && id != nil {
			c.rememberLocalFacts(id.Name, st.Value)
		}
	case *frontend.ReturnStmt:
		c.walkExpr(st.Value)
	case *frontend.ThrowStmt:
		c.walkExpr(st.Value)
	case *frontend.PrintStmt:
		c.walkExpr(st.Value)
	case *frontend.ExprStmt:
		c.walkExpr(st.Expr)
	case *frontend.IfStmt:
		c.walkExpr(st.Cond)
		thenState := *c
		thenState.allocLen = cloneInt64MapLocal(c.allocLen)
		thenState.consts = cloneInt64MapLocal(c.consts)
		thenState.walkStatements(st.Then)
		elseState := *c
		elseState.allocLen = cloneInt64MapLocal(c.allocLen)
		elseState.consts = cloneInt64MapLocal(c.consts)
		elseState.walkStatements(st.Else)
		c.allocLen = intersectInt64MapValues(thenState.allocLen, elseState.allocLen)
		c.consts = intersectInt64MapValues(thenState.consts, elseState.consts)
	case *frontend.IfLetStmt:
		c.walkExpr(st.Value)
		c.walkExpr(st.Pattern)
		thenState := *c
		thenState.allocLen = cloneInt64MapLocal(c.allocLen)
		thenState.consts = cloneInt64MapLocal(c.consts)
		thenState.walkStatements(st.Then)
		elseState := *c
		elseState.allocLen = cloneInt64MapLocal(c.allocLen)
		elseState.consts = cloneInt64MapLocal(c.consts)
		elseState.walkStatements(st.Else)
	case *frontend.WhileStmt:
		c.walkExpr(st.Cond)
		bodyState := *c
		bodyState.allocLen = cloneInt64MapLocal(c.allocLen)
		bodyState.consts = cloneInt64MapLocal(c.consts)
		bodyState.walkStatements(st.Body)
	case *frontend.ForRangeStmt:
		c.walkExpr(st.Start)
		c.walkExpr(st.End)
		c.walkExpr(st.Iterable)
		bodyState := *c
		bodyState.allocLen = cloneInt64MapLocal(c.allocLen)
		bodyState.consts = cloneInt64MapLocal(c.consts)
		bodyState.walkStatements(st.Body)
	case *frontend.MatchStmt:
		c.walkExpr(st.Value)
		for _, mc := range st.Cases {
			c.walkExpr(mc.Guard)
			branch := *c
			branch.allocLen = cloneInt64MapLocal(c.allocLen)
			branch.consts = cloneInt64MapLocal(c.consts)
			branch.walkStatements(mc.Body)
		}
	case *frontend.UnsafeStmt:
		nested := *c
		nested.allocLen = cloneInt64MapLocal(c.allocLen)
		nested.consts = cloneInt64MapLocal(c.consts)
		nested.walkStatements(st.Body)
	case *frontend.DeferStmt:
		nested := *c
		nested.allocLen = cloneInt64MapLocal(c.allocLen)
		nested.consts = cloneInt64MapLocal(c.consts)
		nested.walkStatements(st.Body)
	case *frontend.IslandStmt:
		c.walkExpr(st.Size)
		nested := *c
		nested.allocLen = cloneInt64MapLocal(c.allocLen)
		nested.consts = cloneInt64MapLocal(c.consts)
		nested.walkStatements(st.Body)
	}
}

func (c *helperOffsetCallCollector) walkExpr(expr frontend.Expr) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *frontend.CallExpr:
		c.recordCall(e)
		for _, arg := range e.Args {
			c.walkExpr(arg)
		}
	case *frontend.BinaryExpr:
		c.walkExpr(e.Left)
		c.walkExpr(e.Right)
	case *frontend.UnaryExpr:
		c.walkExpr(e.X)
	case *frontend.FieldAccessExpr:
		c.walkExpr(e.Base)
	case *frontend.IndexExpr:
		c.walkExpr(e.Base)
		c.walkExpr(e.Index)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			c.walkExpr(field.Value)
		}
	case *frontend.MatchExpr:
		c.walkExpr(e.Value)
		for _, mc := range e.Cases {
			c.walkExpr(mc.Guard)
			c.walkExpr(mc.Value)
		}
	case *frontend.CatchExpr:
		c.walkExpr(e.Call)
		for _, mc := range e.Cases {
			c.walkExpr(mc.Guard)
			c.walkExpr(mc.Value)
		}
	case *frontend.TryExpr:
		c.walkExpr(e.X)
	case *frontend.AwaitExpr:
		c.walkExpr(e.X)
	case *frontend.ClosureExpr:
		if e.Decl != nil {
			for _, stmt := range e.Decl.Body {
				c.walkStatement(stmt)
			}
		}
	}
}

func (c *helperOffsetCallCollector) recordCall(call *frontend.CallExpr) {
	callee := helperSummaryResolvedCallName(c.fn.Module, call)
	candidate, ok := c.candidates[callee]
	if !ok {
		return
	}
	if c.fn.Module == "" || !strings.HasPrefix(callee, c.fn.Module+".") {
		c.unsafe[callee] = true
		return
	}
	if candidate.ParamIndex < 0 || candidate.ParamIndex >= len(call.Args) ||
		candidate.OffsetParamIndex < 0 || candidate.OffsetParamIndex >= len(call.Args) {
		c.unsafe[callee] = true
		return
	}
	actual := helperSummaryExprPath(call.Args[candidate.ParamIndex])
	length, lengthOK := c.allocLen[actual]
	actualOffset, offsetOK := c.constArgValue(call.Args[candidate.OffsetParamIndex])
	maxIndex, addOK := checkedAddInt64(actualOffset, candidate.MaxDelta)
	if actual == "" || !lengthOK || !offsetOK || !addOK || actualOffset < 0 || maxIndex < 0 ||
		maxIndex >= length {
		c.unsafe[callee] = true
		return
	}
	c.calls[callee] = append(c.calls[callee], helperOffsetCall{
		Caller:       c.fn.Name,
		ActualName:   actual,
		Length:       length,
		ActualOffset: actualOffset,
	})
}

func (c *helperOffsetCallCollector) rememberLocalFacts(name string, expr frontend.Expr) {
	if name == "" {
		return
	}
	if length, ok := helperSummaryAllocationLen(expr); ok {
		c.allocLen[name] = length
	} else if source := helperSummaryExprPath(expr); source != "" {
		if length, ok := c.allocLen[source]; ok {
			c.allocLen[name] = length
		} else {
			delete(c.allocLen, name)
		}
	} else {
		delete(c.allocLen, name)
	}
	if value, ok := c.callReturnConst(expr); ok {
		c.consts[name] = value
		return
	}
	info, infoOK := c.fn.Locals[name]
	if infoOK && !info.Mutable {
		if value, ok := c.constArgValue(expr); ok {
			c.consts[name] = value
			return
		}
	}
	delete(c.consts, name)
}

func (c *helperOffsetCallCollector) callReturnConst(expr frontend.Expr) (int64, bool) {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return 0, false
	}
	callee := helperSummaryResolvedCallName(c.fn.Module, call)
	candidate, ok := c.candidates[callee]
	if !ok || !candidate.HasReturnDelta || candidate.OffsetParamIndex < 0 ||
		candidate.OffsetParamIndex >= len(call.Args) {
		return 0, false
	}
	actualOffset, ok := c.constArgValue(call.Args[candidate.OffsetParamIndex])
	if !ok {
		return 0, false
	}
	return checkedAddInt64(actualOffset, candidate.ReturnDelta)
}

func (c *helperOffsetCallCollector) constArgValue(expr frontend.Expr) (int64, bool) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		if e == nil {
			return 0, false
		}
		return int64(e.Value), true
	case *frontend.UnaryExpr:
		if e == nil || e.Op != frontend.TokenMinus {
			return 0, false
		}
		value, ok := c.constArgValue(e.X)
		if !ok {
			return 0, false
		}
		return -value, true
	case *frontend.BinaryExpr:
		if e == nil {
			return 0, false
		}
		left, leftOK := c.constArgValue(e.Left)
		right, rightOK := c.constArgValue(e.Right)
		if !leftOK || !rightOK {
			return 0, false
		}
		switch e.Op {
		case frontend.TokenPlus:
			return checkedAddInt64(left, right)
		case frontend.TokenMinus:
			return checkedAddInt64(left, -right)
		default:
			return 0, false
		}
	case *frontend.IdentExpr:
		if e == nil || e.Name == "" {
			return 0, false
		}
		value, ok := c.consts[e.Name]
		return value, ok
	default:
		return 0, false
	}
}

func helperOffsetIndexDelta(
	index frontend.Expr,
	offsetParams map[string]int,
) (string, int64, bool) {
	for name := range offsetParams {
		if delta, ok := helperOffsetIndexDeltaForParam(index, name); ok {
			return name, delta, true
		}
	}
	return "", 0, false
}

func helperOffsetIndexDeltaForParam(index frontend.Expr, offsetParam string) (int64, bool) {
	if offsetParam == "" {
		return 0, false
	}
	if id, ok := index.(*frontend.IdentExpr); ok && id != nil && id.Name == offsetParam {
		return 0, true
	}
	bin, ok := index.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenPlus {
		return 0, false
	}
	if id, ok := bin.Left.(*frontend.IdentExpr); ok && id != nil && id.Name == offsetParam {
		delta, ok := helperSummaryConstInt(bin.Right)
		return delta, ok
	}
	if id, ok := bin.Right.(*frontend.IdentExpr); ok && id != nil && id.Name == offsetParam {
		delta, ok := helperSummaryConstInt(bin.Left)
		return delta, ok
	}
	return 0, false
}

func helperOffsetReturnDelta(expr frontend.Expr, offsetParam string) (int64, bool) {
	delta, ok := helperOffsetIndexDeltaForParam(expr, offsetParam)
	return delta, ok && delta >= 0
}

type helperSummaryCandidate struct {
	Module     string
	ParamName  string
	ParamIndex int
	Stores     []HelperSummaryStore
	MaxIndex   int64
}

type helperSummaryCall struct {
	Caller     string
	ActualName string
	Length     int64
}

func helperSummaryCalleeCandidate(fn semantics.CheckedFunc) (helperSummaryCandidate, bool) {
	if fn.Decl == nil || len(fn.Decl.Params) == 0 {
		return helperSummaryCandidate{}, false
	}
	paramIndex := -1
	paramName := ""
	for i, param := range fn.Decl.Params {
		if param.Ownership == "inout" {
			info, ok := fn.Locals[param.Name]
			if !ok || info.TypeName != "[]u8" {
				return helperSummaryCandidate{}, false
			}
			if paramIndex >= 0 {
				return helperSummaryCandidate{}, false
			}
			paramIndex = i
			paramName = param.Name
		}
	}
	if paramIndex < 0 {
		return helperSummaryCandidate{}, false
	}
	scanner := helperSummaryCalleeScanner{paramName: paramName, safe: true}
	scanner.walkStatements(fn.Decl.Body)
	if !scanner.safe || len(scanner.stores) == 0 {
		return helperSummaryCandidate{}, false
	}
	sort.Slice(scanner.stores, func(i, j int) bool {
		return scanner.stores[i].Index < scanner.stores[j].Index
	})
	return helperSummaryCandidate{
		Module:     fn.Module,
		ParamName:  paramName,
		ParamIndex: paramIndex,
		Stores:     scanner.stores,
		MaxIndex:   scanner.maxIndex,
	}, true
}

type helperSummaryCalleeScanner struct {
	paramName string
	stores    []HelperSummaryStore
	maxIndex  int64
	safe      bool
}

func (s *helperSummaryCalleeScanner) walkStatements(stmts []frontend.Stmt) {
	for _, stmt := range stmts {
		if !s.safe {
			return
		}
		s.walkStatement(stmt)
	}
}

func (s *helperSummaryCalleeScanner) walkStatement(stmt frontend.Stmt) {
	switch st := stmt.(type) {
	case *frontend.LetStmt:
		if helperSummaryExprContainsPath(st.Value, s.paramName) {
			s.safe = false
		}
	case *frontend.AssignStmt:
		if idx, ok := st.Target.(*frontend.IndexExpr); ok && idx != nil && helperSummaryExprPath(
			idx.Base,
		) == s.paramName {
			index, ok := helperSummaryConstInt(idx.Index)
			if !ok || index < 0 || helperSummaryExprContainsPath(st.Value, s.paramName) {
				s.safe = false
				return
			}
			s.stores = append(s.stores, HelperSummaryStore{Index: index, Source: st.At})
			if len(s.stores) == 1 || index > s.maxIndex {
				s.maxIndex = index
			}
			return
		}
		if helperSummaryExprContainsPath(
			st.Target,
			s.paramName,
		) || helperSummaryExprContainsPath(
			st.Value,
			s.paramName,
		) {
			s.safe = false
		}
	case *frontend.ReturnStmt:
		if helperSummaryExprContainsPath(st.Value, s.paramName) {
			s.safe = false
		}
	case *frontend.ThrowStmt:
		if helperSummaryExprContainsPath(st.Value, s.paramName) {
			s.safe = false
		}
	case *frontend.PrintStmt:
		if helperSummaryExprContainsPath(st.Value, s.paramName) {
			s.safe = false
		}
	case *frontend.ExprStmt:
		if helperSummaryExprContainsPath(st.Expr, s.paramName) {
			s.safe = false
		}
	case *frontend.IfStmt:
		if helperSummaryExprContainsPath(st.Cond, s.paramName) {
			s.safe = false
			return
		}
		s.walkStatements(st.Then)
		s.walkStatements(st.Else)
	case *frontend.IfLetStmt:
		if helperSummaryExprContainsPath(
			st.Value,
			s.paramName,
		) || helperSummaryExprContainsPath(
			st.Pattern,
			s.paramName,
		) {
			s.safe = false
			return
		}
		s.walkStatements(st.Then)
		s.walkStatements(st.Else)
	case *frontend.WhileStmt:
		if helperSummaryExprContainsPath(st.Cond, s.paramName) {
			s.safe = false
			return
		}
		s.walkStatements(st.Body)
	case *frontend.ForRangeStmt:
		if helperSummaryExprContainsPath(st.Start, s.paramName) ||
			helperSummaryExprContainsPath(st.End, s.paramName) ||
			helperSummaryExprContainsPath(st.Iterable, s.paramName) {
			s.safe = false
			return
		}
		s.walkStatements(st.Body)
	case *frontend.MatchStmt:
		if helperSummaryExprContainsPath(st.Value, s.paramName) {
			s.safe = false
			return
		}
		for _, c := range st.Cases {
			if helperSummaryExprContainsPath(c.Guard, s.paramName) {
				s.safe = false
				return
			}
			s.walkStatements(c.Body)
		}
	case *frontend.UnsafeStmt:
		s.safe = false
	case *frontend.DeferStmt:
		if helperSummaryStatementsContainPath(st.Body, s.paramName) {
			s.safe = false
		}
	case *frontend.IslandStmt:
		if helperSummaryExprContainsPath(st.Size, s.paramName) {
			s.safe = false
			return
		}
		s.walkStatements(st.Body)
	}
}

type helperSummaryCallCollector struct {
	fn         semantics.CheckedFunc
	candidates map[string]helperSummaryCandidate
	allocLen   map[string]int64
	calls      map[string][]helperSummaryCall
	unsafe     map[string]bool
}

func (c *helperSummaryCallCollector) walkStatements(stmts []frontend.Stmt) {
	for _, stmt := range stmts {
		c.walkStatement(stmt)
	}
}

func (c *helperSummaryCallCollector) walkStatement(stmt frontend.Stmt) {
	switch st := stmt.(type) {
	case *frontend.LetStmt:
		c.walkExpr(st.Value)
		c.rememberLocalLength(st.Name, st.Value)
	case *frontend.AssignStmt:
		c.walkExpr(st.Value)
		c.walkExpr(st.Target)
		if id, ok := st.Target.(*frontend.IdentExpr); ok && id != nil {
			c.rememberLocalLength(id.Name, st.Value)
		}
	case *frontend.ReturnStmt:
		c.walkExpr(st.Value)
	case *frontend.ThrowStmt:
		c.walkExpr(st.Value)
	case *frontend.PrintStmt:
		c.walkExpr(st.Value)
	case *frontend.ExprStmt:
		c.walkExpr(st.Expr)
	case *frontend.IfStmt:
		c.walkExpr(st.Cond)
		thenState := *c
		thenState.allocLen = cloneInt64MapLocal(c.allocLen)
		thenState.walkStatements(st.Then)
		elseState := *c
		elseState.allocLen = cloneInt64MapLocal(c.allocLen)
		elseState.walkStatements(st.Else)
		c.allocLen = intersectInt64MapValues(thenState.allocLen, elseState.allocLen)
	case *frontend.IfLetStmt:
		c.walkExpr(st.Value)
		c.walkExpr(st.Pattern)
		thenState := *c
		thenState.allocLen = cloneInt64MapLocal(c.allocLen)
		thenState.walkStatements(st.Then)
		elseState := *c
		elseState.allocLen = cloneInt64MapLocal(c.allocLen)
		elseState.walkStatements(st.Else)
	case *frontend.WhileStmt:
		c.walkExpr(st.Cond)
		bodyState := *c
		bodyState.allocLen = cloneInt64MapLocal(c.allocLen)
		bodyState.walkStatements(st.Body)
	case *frontend.ForRangeStmt:
		c.walkExpr(st.Start)
		c.walkExpr(st.End)
		c.walkExpr(st.Iterable)
		bodyState := *c
		bodyState.allocLen = cloneInt64MapLocal(c.allocLen)
		bodyState.walkStatements(st.Body)
	case *frontend.MatchStmt:
		c.walkExpr(st.Value)
		for _, mc := range st.Cases {
			c.walkExpr(mc.Guard)
			branch := *c
			branch.allocLen = cloneInt64MapLocal(c.allocLen)
			branch.walkStatements(mc.Body)
		}
	case *frontend.UnsafeStmt:
		nested := *c
		nested.allocLen = cloneInt64MapLocal(c.allocLen)
		nested.walkStatements(st.Body)
	case *frontend.DeferStmt:
		nested := *c
		nested.allocLen = cloneInt64MapLocal(c.allocLen)
		nested.walkStatements(st.Body)
	case *frontend.IslandStmt:
		c.walkExpr(st.Size)
		nested := *c
		nested.allocLen = cloneInt64MapLocal(c.allocLen)
		nested.walkStatements(st.Body)
	}
}

func (c *helperSummaryCallCollector) walkExpr(expr frontend.Expr) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *frontend.CallExpr:
		c.recordCall(e)
		for _, arg := range e.Args {
			c.walkExpr(arg)
		}
	case *frontend.BinaryExpr:
		c.walkExpr(e.Left)
		c.walkExpr(e.Right)
	case *frontend.UnaryExpr:
		c.walkExpr(e.X)
	case *frontend.FieldAccessExpr:
		c.walkExpr(e.Base)
	case *frontend.IndexExpr:
		c.walkExpr(e.Base)
		c.walkExpr(e.Index)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			c.walkExpr(field.Value)
		}
	case *frontend.MatchExpr:
		c.walkExpr(e.Value)
		for _, mc := range e.Cases {
			c.walkExpr(mc.Guard)
			c.walkExpr(mc.Value)
		}
	case *frontend.CatchExpr:
		c.walkExpr(e.Call)
		for _, mc := range e.Cases {
			c.walkExpr(mc.Guard)
			c.walkExpr(mc.Value)
		}
	case *frontend.TryExpr:
		c.walkExpr(e.X)
	case *frontend.AwaitExpr:
		c.walkExpr(e.X)
	case *frontend.ClosureExpr:
		if e.Decl != nil {
			for _, stmt := range e.Decl.Body {
				c.walkStatement(stmt)
			}
		}
	}
}

func (c *helperSummaryCallCollector) recordCall(call *frontend.CallExpr) {
	callee := helperSummaryResolvedCallName(c.fn.Module, call)
	candidate, ok := c.candidates[callee]
	if !ok {
		return
	}
	if c.fn.Module == "" || !strings.HasPrefix(callee, c.fn.Module+".") {
		c.unsafe[callee] = true
		return
	}
	if candidate.ParamIndex < 0 || candidate.ParamIndex >= len(call.Args) {
		c.unsafe[callee] = true
		return
	}
	actual := helperSummaryExprPath(call.Args[candidate.ParamIndex])
	length, ok := c.allocLen[actual]
	if !ok || actual == "" {
		c.unsafe[callee] = true
		return
	}
	c.calls[callee] = append(c.calls[callee], helperSummaryCall{
		Caller:     c.fn.Name,
		ActualName: actual,
		Length:     length,
	})
}

func (c *helperSummaryCallCollector) rememberLocalLength(name string, expr frontend.Expr) {
	if name == "" {
		return
	}
	if length, ok := helperSummaryAllocationLen(expr); ok {
		c.allocLen[name] = length
		return
	}
	if source := helperSummaryExprPath(expr); source != "" {
		if length, ok := c.allocLen[source]; ok {
			c.allocLen[name] = length
			return
		}
	}
	delete(c.allocLen, name)
}

func helperSummaryResolvedCallName(moduleName string, call *frontend.CallExpr) string {
	if call == nil || call.Name == "" {
		return ""
	}
	name := call.Name
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	if strings.Contains(name, ".") {
		return name
	}
	if moduleName != "" {
		return moduleName + "." + name
	}
	return name
}

func helperSummaryAllocationLen(expr frontend.Expr) (int64, bool) {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil || len(call.Args) != 1 {
		return 0, false
	}
	name := call.Name
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	if name != "core.make_u8" && name != "make_u8" {
		return 0, false
	}
	length, ok := helperSummaryConstInt(call.Args[0])
	return length, ok && length >= 0
}

func helperSummaryConstInt(expr frontend.Expr) (int64, bool) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		if e == nil {
			return 0, false
		}
		return int64(e.Value), true
	case *frontend.UnaryExpr:
		if e == nil || e.Op != frontend.TokenMinus {
			return 0, false
		}
		value, ok := helperSummaryConstInt(e.X)
		if !ok {
			return 0, false
		}
		return -value, true
	default:
		return 0, false
	}
}

func helperSummaryStatementsContainPath(stmts []frontend.Stmt, path string) bool {
	scanner := helperSummaryPathScanner{path: path}
	for _, stmt := range stmts {
		scanner.walkStatement(stmt)
		if scanner.found {
			return true
		}
	}
	return false
}

type helperSummaryPathScanner struct {
	path  string
	found bool
}

func (s *helperSummaryPathScanner) walkStatement(stmt frontend.Stmt) {
	switch st := stmt.(type) {
	case *frontend.LetStmt:
		s.walkExpr(st.Value)
	case *frontend.AssignStmt:
		s.walkExpr(st.Target)
		s.walkExpr(st.Value)
	case *frontend.ReturnStmt:
		s.walkExpr(st.Value)
	case *frontend.ThrowStmt:
		s.walkExpr(st.Value)
	case *frontend.PrintStmt:
		s.walkExpr(st.Value)
	case *frontend.ExprStmt:
		s.walkExpr(st.Expr)
	case *frontend.IfStmt:
		s.walkExpr(st.Cond)
		for _, nested := range st.Then {
			s.walkStatement(nested)
		}
		for _, nested := range st.Else {
			s.walkStatement(nested)
		}
	case *frontend.IfLetStmt:
		s.walkExpr(st.Pattern)
		s.walkExpr(st.Value)
		for _, nested := range st.Then {
			s.walkStatement(nested)
		}
		for _, nested := range st.Else {
			s.walkStatement(nested)
		}
	case *frontend.WhileStmt:
		s.walkExpr(st.Cond)
		for _, nested := range st.Body {
			s.walkStatement(nested)
		}
	case *frontend.ForRangeStmt:
		s.walkExpr(st.Start)
		s.walkExpr(st.End)
		s.walkExpr(st.Iterable)
		for _, nested := range st.Body {
			s.walkStatement(nested)
		}
	case *frontend.MatchStmt:
		s.walkExpr(st.Value)
		for _, mc := range st.Cases {
			s.walkExpr(mc.Guard)
			for _, nested := range mc.Body {
				s.walkStatement(nested)
			}
		}
	case *frontend.UnsafeStmt:
		for _, nested := range st.Body {
			s.walkStatement(nested)
		}
	case *frontend.DeferStmt:
		for _, nested := range st.Body {
			s.walkStatement(nested)
		}
	case *frontend.IslandStmt:
		s.walkExpr(st.Size)
		for _, nested := range st.Body {
			s.walkStatement(nested)
		}
	}
}

func (s *helperSummaryPathScanner) walkExpr(expr frontend.Expr) {
	if s.found || expr == nil {
		return
	}
	if helperSummaryExprPath(expr) == s.path {
		s.found = true
		return
	}
	switch e := expr.(type) {
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			s.walkExpr(arg)
		}
	case *frontend.BinaryExpr:
		s.walkExpr(e.Left)
		s.walkExpr(e.Right)
	case *frontend.UnaryExpr:
		s.walkExpr(e.X)
	case *frontend.FieldAccessExpr:
		s.walkExpr(e.Base)
	case *frontend.IndexExpr:
		s.walkExpr(e.Base)
		s.walkExpr(e.Index)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			s.walkExpr(field.Value)
		}
	case *frontend.MatchExpr:
		s.walkExpr(e.Value)
		for _, mc := range e.Cases {
			s.walkExpr(mc.Guard)
			s.walkExpr(mc.Value)
		}
	case *frontend.CatchExpr:
		s.walkExpr(e.Call)
		for _, mc := range e.Cases {
			s.walkExpr(mc.Guard)
			s.walkExpr(mc.Value)
		}
	case *frontend.TryExpr:
		s.walkExpr(e.X)
	case *frontend.AwaitExpr:
		s.walkExpr(e.X)
	case *frontend.ClosureExpr:
		if e.Decl != nil {
			for _, stmt := range e.Decl.Body {
				s.walkStatement(stmt)
			}
		}
	}
}

func helperSummaryExprContainsPath(expr frontend.Expr, path string) bool {
	if path == "" {
		return false
	}
	scanner := helperSummaryPathScanner{path: path}
	scanner.walkExpr(expr)
	return scanner.found
}

func helperSummaryExprPath(expr frontend.Expr) string {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if e == nil {
			return ""
		}
		return e.Name
	case *frontend.FieldAccessExpr:
		if e == nil {
			return ""
		}
		base := helperSummaryExprPath(e.Base)
		if base == "" || e.Field == "" {
			return ""
		}
		return base + "." + e.Field
	default:
		return ""
	}
}

func cloneInt64MapLocal(in map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func intersectInt64MapValues(left map[string]int64, right map[string]int64) map[string]int64 {
	out := map[string]int64{}
	keys := make([]string, 0, len(left))
	for key := range left {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if value, ok := right[key]; ok && value == left[key] {
			out[key] = value
		}
	}
	return out
}
