package rangeproof

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	corerangeproof "tetra_language/compiler/internal/rangeproof"
	"tetra_language/compiler/internal/semantics"
)

func PathMatchesMutation(proofPath string, mutatedPath string) bool {
	if proofPath == "" || mutatedPath == "" {
		return false
	}
	return proofPath == mutatedPath || strings.HasPrefix(proofPath, mutatedPath+".")
}

func CloneBoolMap(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func CloneInt64Map(in map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func CloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func StaticRangeFromCondition(cond frontend.Expr) (string, string, corerangeproof.Range, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil {
		return "", "", corerangeproof.Range{}, false
	}
	left, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || left == nil {
		return "", "", corerangeproof.Range{}, false
	}
	switch bin.Op {
	case frontend.TokenLess, frontend.TokenBangEq:
		base := LenFieldBaseName(bin.Right)
		if base == "" {
			return "", "", corerangeproof.Range{}, false
		}
		return left.Name, base, corerangeproof.LessThanLen(left.Name, base), true
	case frontend.TokenLessEq:
		base := LenMinusOneBaseName(bin.Right)
		if base == "" {
			return "", "", corerangeproof.Range{}, false
		}
		return left.Name, base, corerangeproof.LessEqualLenMinusOne(left.Name, base), true
	default:
		return "", "", corerangeproof.Range{}, false
	}
}

func StaticRangeCondition(cond frontend.Expr) (string, string, bool) {
	indexName, baseName, _, ok := StaticRangeFromCondition(cond)
	return indexName, baseName, ok
}

func WhileRangeCondition(cond frontend.Expr) (string, string, bool) {
	return StaticRangeCondition(cond)
}

func StaticWhileRangeCondition(cond frontend.Expr) (string, string, bool) {
	return StaticRangeCondition(cond)
}

func BranchRangeCondition(cond frontend.Expr) (string, string, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenAmpAmp {
		return "", "", false
	}
	if indexName, baseName, ok := BranchRangeConditionParts(bin.Left, bin.Right); ok {
		return indexName, baseName, true
	}
	return BranchRangeConditionParts(bin.Right, bin.Left)
}

func BranchRangeConditionParts(lower frontend.Expr, upper frontend.Expr) (string, string, bool) {
	lowerIndex, ok := NonNegativeGuardIndex(lower)
	if !ok {
		return "", "", false
	}
	upperIndex, baseName, ok := WhileRangeCondition(upper)
	if !ok || upperIndex != lowerIndex {
		return "", "", false
	}
	return upperIndex, baseName, true
}

func ModuloDivisorName(expr frontend.Expr) (frontend.Expr, string, bool) {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenPercent {
		return nil, "", false
	}
	divisor, ok := bin.Right.(*frontend.IdentExpr)
	if !ok || divisor == nil || divisor.Name == "" {
		return nil, "", false
	}
	return bin.Left, divisor.Name, true
}

func NonNegativeGuardIndex(expr frontend.Expr) (string, bool) {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil {
		return "", false
	}
	if left, ok := bin.Left.(*frontend.IdentExpr); ok && left != nil &&
		bin.Op == frontend.TokenGreaterEq &&
		IsZeroNumber(bin.Right) {
		return left.Name, true
	}
	if right, ok := bin.Right.(*frontend.IdentExpr); ok && right != nil &&
		bin.Op == frontend.TokenLessEq &&
		IsZeroNumber(bin.Left) {
		return right.Name, true
	}
	return "", false
}

func IsZeroNumber(expr frontend.Expr) bool {
	num, ok := expr.(*frontend.NumberExpr)
	return ok && num != nil && num.Value == 0
}

func LenFieldBaseName(expr frontend.Expr) string {
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok || field == nil || field.Field != "len" {
		return ""
	}
	return SimpleExprPath(field.Base)
}

func LenMinusOneBaseName(expr frontend.Expr) string {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenMinus {
		return ""
	}
	right, ok := bin.Right.(*frontend.NumberExpr)
	if !ok || right == nil || right.Value != 1 {
		return ""
	}
	return LenFieldBaseName(bin.Left)
}

func IsZeroLiteral(expr frontend.Expr) bool {
	num, ok := expr.(*frontend.NumberExpr)
	return ok && num != nil && num.Value == 0
}

func IsRawSliceConstructor(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	name := call.Name
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	switch name {
	case "core.raw_slice_u8_from_parts",
		"core.raw_slice_u16_from_parts",
		"core.raw_slice_i32_from_parts",
		"core.raw_slice_bool_from_parts":
		return true
	default:
		return false
	}
}

func RawSliceElementShift(name string) int32 {
	switch name {
	case "core.raw_slice_u16_from_parts":
		return 1
	case "core.raw_slice_i32_from_parts", "core.raw_slice_bool_from_parts":
		return 2
	default:
		return 0
	}
}

func IsSliceCopyBuiltinName(name string) bool {
	return name == "core.string_copy" ||
		name == "core.string_copy_into" ||
		strings.HasPrefix(name, "core.slice_copy_")
}

func IsBorrowOrViewBuiltinName(name string) bool {
	return name == "core.string_borrow" ||
		name == "core.string_window" ||
		name == "core.string_prefix" ||
		name == "core.string_suffix" ||
		strings.HasPrefix(name, "core.slice_borrow_") ||
		strings.HasPrefix(name, "core.slice_window_") ||
		strings.HasPrefix(name, "core.slice_prefix_") ||
		strings.HasPrefix(name, "core.slice_suffix_")
}

func SimpleExprPath(expr frontend.Expr) string {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.FieldAccessExpr:
		base := SimpleExprPath(e.Base)
		if base == "" {
			return e.Field
		}
		return base + "." + e.Field
	default:
		return ""
	}
}

func WhileBoundsProofID(indexName string, baseName string, pos frontend.Position) string {
	return RangeBoundsProofID("while", indexName, baseName, pos)
}

func IfBoundsProofID(indexName string, baseName string, pos frontend.Position) string {
	return RangeBoundsProofID("if", indexName, baseName, pos)
}

func ModuloBoundsProofID(indexName string, baseName string, pos frontend.Position) string {
	return RangeBoundsProofID("modulo", indexName, baseName, pos)
}

func RangeBoundsProofID(
	kind string,
	indexName string,
	baseName string,
	pos frontend.Position,
) string {
	baseName = strings.NewReplacer(".", "_", " ", "_").Replace(baseName)
	if baseName == "" {
		baseName = "value"
	}
	return fmt.Sprintf("proof:%s:%s:%s:%d:%d", kind, indexName, baseName, pos.Line, pos.Col)
}

func CopyLoopBoundsProofID(name string, pos frontend.Position) string {
	name = strings.NewReplacer(".", "_", " ", "_").Replace(name)
	if name == "" {
		name = "copy"
	}
	return fmt.Sprintf("proof:copy-loop:%s:%d:%d", name, pos.Line, pos.Col)
}

func ForCollectionBoundsProofID(stmt *frontend.ForRangeStmt) string {
	kind := "for-collection"
	if IsViewCollectionIterable(stmt.Iterable) {
		kind = "for-collection-view"
	}
	return fmt.Sprintf("proof:%s:%s:%d:%d", kind, stmt.Name, stmt.At.Line, stmt.At.Col)
}

func IsViewCollectionIterable(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	name := call.Name
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	return strings.HasPrefix(name, "core.slice_window_") ||
		strings.HasPrefix(name, "core.slice_prefix_") ||
		strings.HasPrefix(name, "core.slice_suffix_") ||
		name == "core.string_window" ||
		name == "core.string_prefix" ||
		name == "core.string_suffix"
}
