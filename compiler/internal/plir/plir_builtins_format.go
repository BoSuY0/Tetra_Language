package plir

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func valueID(kind ValueKind, name string) string {
	if name == "" {
		name = "anon"
	}
	return string(kind) + ":" + name
}

func valueIDsForPath(path string) []string {
	return []string{
		valueID(ValueView, path),
		valueID(ValueAllocIntent, path),
		valueID(ValueLocal, path),
		valueID(ValueParam, path),
	}
}

func makeSliceElem(name string) (string, bool) {
	switch name {
	case "core.make_u8", "core.island_make_u8":
		return "u8", true
	case "core.make_u16", "core.island_make_u16":
		return "u16", true
	case "core.make_i32", "core.island_make_i32":
		return "i32", true
	case "core.make_bool", "core.island_make_bool":
		return "bool", true
	default:
		return "", false
	}
}

func rawSliceElem(name string) (string, bool) {
	switch name {
	case "core.raw_slice_u8_from_parts":
		return "u8", true
	case "core.raw_slice_u16_from_parts":
		return "u16", true
	case "core.raw_slice_i32_from_parts":
		return "i32", true
	case "core.raw_slice_bool_from_parts":
		return "bool", true
	default:
		return "", false
	}
}

func rawSliceBuiltin(name string) bool {
	_, ok := rawSliceElem(name)
	return ok
}

func rawMemoryAccessBuiltin(name string) bool {
	switch name {
	case "core.load_i32", "core.store_i32",
		"core.load_u8", "core.store_u8",
		"core.load_ptr", "core.store_ptr", "core.store_arch_ptr":
		return true
	default:
		return false
	}
}

func rawMemoryAccessWidthBytes(name string) int64 {
	switch name {
	case "core.load_i32", "core.store_i32":
		return 4
	case "core.load_ptr", "core.store_ptr", "core.store_arch_ptr":
		// MPC-8 runtime evidence is linux-x64; other targets stay build/lower scoped.
		return 8
	default:
		return 1
	}
}

func rawPointerRoot(path string) string {
	if !strings.HasSuffix(path, ".ptr") {
		return ""
	}
	rootPath := strings.TrimSuffix(path, ".ptr")
	if rootPath == "" || rootPath == "?" || rootPath == "expr" {
		return ""
	}
	if dot := strings.Index(rootPath, "."); dot >= 0 {
		return rootPath[:dot]
	}
	return rootPath
}

func sliceBorrowElem(name string) (string, bool) {
	if !strings.HasPrefix(name, "core.slice_borrow_") {
		return "", false
	}
	elem := strings.TrimPrefix(name, "core.slice_borrow_")
	switch elem {
	case "u8", "u16", "i32", "bool":
		return elem, true
	default:
		return "", false
	}
}

func sliceCopyElem(name string) (string, bool) {
	if !strings.HasPrefix(name, "core.slice_copy_") ||
		strings.HasPrefix(name, "core.slice_copy_into_") {
		return "", false
	}
	elem := strings.TrimPrefix(name, "core.slice_copy_")
	switch elem {
	case "u8", "u16", "i32", "bool":
		return elem, true
	default:
		return "", false
	}
}

func sliceCopyIntoBuiltin(name string) bool {
	if !strings.HasPrefix(name, "core.slice_copy_into_") {
		return false
	}
	switch strings.TrimPrefix(name, "core.slice_copy_into_") {
	case "u8", "u16", "i32", "bool":
		return true
	default:
		return false
	}
}

func stringBorrowBuiltin(name string) bool {
	return name == "core.string_borrow"
}

func stringCopyBuiltin(name string) bool {
	return name == "core.string_copy"
}

func stringCopyIntoBuiltin(name string) bool {
	return name == "core.string_copy_into"
}

func copyBuiltin(name string) bool {
	if stringCopyBuiltin(name) || sliceCopyElemName(name) || stringCopyIntoBuiltin(name) ||
		sliceCopyIntoBuiltin(name) {
		return true
	}
	return false
}

func sliceCopyElemName(name string) bool {
	_, ok := sliceCopyElem(name)
	return ok
}

func borrowOrViewBuiltin(name string) bool {
	if stringBorrowBuiltin(name) {
		return true
	}
	if _, ok := sliceBorrowElem(name); ok {
		return true
	}
	if _, _, ok := sliceViewElem(name); ok {
		return true
	}
	if _, _, ok := stringViewBuiltin(name); ok {
		return true
	}
	return false
}

func sliceViewElem(name string) (elem string, method string, ok bool) {
	if !strings.HasPrefix(name, "core.slice_") {
		return "", "", false
	}
	rest := strings.TrimPrefix(name, "core.slice_")
	for _, candidate := range []string{"window", "prefix", "suffix"} {
		prefix := candidate + "_"
		if !strings.HasPrefix(rest, prefix) {
			continue
		}
		elem = strings.TrimPrefix(rest, prefix)
		switch elem {
		case "u8", "u16", "i32", "bool":
			return elem, candidate, true
		default:
			return "", "", false
		}
	}
	return "", "", false
}

func stringViewBuiltin(name string) (valueType string, method string, ok bool) {
	if !strings.HasPrefix(name, "core.string_") {
		return "", "", false
	}
	method = strings.TrimPrefix(name, "core.string_")
	switch method {
	case "window", "prefix", "suffix":
		return "str", method, true
	default:
		return "", "", false
	}
}

type derivedWindowRange struct {
	base  string
	start string
	end   string
}

func (b *builder) sliceViewRange(method string, source string, call *frontend.CallExpr) string {
	if parent, ok := b.derivedWindowRangeForSource(source); ok {
		return composeSliceViewRange(parent, method, call)
	}
	return baseSliceViewRange(method, source, call)
}

func (b *builder) derivedWindowRangeForSource(source string) (derivedWindowRange, bool) {
	if source == "" {
		return derivedWindowRange{}, false
	}
	candidates := []string{
		valueID(ValueView, source),
		valueID(ValueAllocIntent, source),
		valueID(ValueLocal, source),
		valueID(ValueParam, source),
	}
	for _, candidate := range candidates {
		for _, fact := range b.facts {
			if fact.Kind != FactDerivedWindow || fact.ValueID != candidate {
				continue
			}
			return parseDerivedWindowRange(fact.Range)
		}
	}
	return derivedWindowRange{}, false
}

func parseDerivedWindowRange(text string) (derivedWindowRange, bool) {
	start := strings.LastIndex(text, "[")
	end := strings.LastIndex(text, "]")
	if start < 0 || end <= start {
		return derivedWindowRange{}, false
	}
	parts := strings.Split(text[start+1:end], "..")
	if len(parts) != 2 {
		return derivedWindowRange{}, false
	}
	base := strings.TrimSpace(text[:start])
	lo := strings.TrimSpace(parts[0])
	hi := strings.TrimSpace(parts[1])
	if base == "" || lo == "" || hi == "" {
		return derivedWindowRange{}, false
	}
	return derivedWindowRange{base: base, start: lo, end: hi}, true
}

func (b *builder) copyIntoOverlapStatus(source string, destination string) string {
	sourceRange, sourceRangeOK := b.derivedWindowRangeForSource(source)
	destinationRange, destinationRangeOK := b.derivedWindowRangeForSource(destination)
	if sourceRangeOK && destinationRangeOK {
		sourceBase := normalizeOverlapRoot(sourceRange.base)
		destinationBase := normalizeOverlapRoot(destinationRange.base)
		if sourceBase == "" || destinationBase == "" {
			return "unknown_conservative"
		}
		if sourceBase == destinationBase {
			if overlap, ok := staticDerivedRangesOverlap(sourceRange, destinationRange); ok {
				if overlap {
					return "known_overlap"
				}
				return "known_disjoint"
			}
			return "unknown_conservative"
		}
		return "distinct_roots"
	}
	sourceRoot, sourceKnown := b.copyIntoKnownRoot(source)
	destinationRoot, destinationKnown := b.copyIntoKnownRoot(destination)
	if !sourceKnown || !destinationKnown || sourceRoot == "" || destinationRoot == "" {
		return "unknown_conservative"
	}
	if sourceRoot == destinationRoot {
		return "unknown_conservative"
	}
	return "distinct_roots"
}

func (b *builder) copyIntoKnownRoot(path string) (string, bool) {
	provenance, known := b.derivedProvenance(path)
	if !known {
		return "", false
	}
	root := normalizeOverlapRoot(provenance.Root)
	if root == "" {
		root = normalizeOverlapRoot(path)
	}
	if root == "" || root == "?" || root == "expr" {
		return "", false
	}
	return root, true
}

func staticDerivedRangesOverlap(
	source derivedWindowRange,
	destination derivedWindowRange,
) (bool, bool) {
	sourceStart, ok := parseRangeConst(source.start)
	if !ok {
		return false, false
	}
	sourceEnd, ok := parseRangeConst(source.end)
	if !ok {
		return false, false
	}
	destinationStart, ok := parseRangeConst(destination.start)
	if !ok {
		return false, false
	}
	destinationEnd, ok := parseRangeConst(destination.end)
	if !ok {
		return false, false
	}
	return sourceStart < destinationEnd && destinationStart < sourceEnd, true
}

func parseRangeConst(text string) (int64, bool) {
	value, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	return value, err == nil
}

func normalizeOverlapRoot(root string) string {
	root = strings.TrimSpace(root)
	for strings.HasPrefix(root, "derived:") {
		root = strings.TrimPrefix(root, "derived:")
	}
	for _, prefix := range []string{"param:", "local:", "view:", "alloc_intent:"} {
		root = strings.TrimPrefix(root, prefix)
	}
	if dot := strings.Index(root, "."); dot > 0 {
		root = root[:dot]
	}
	return root
}

func composeSliceViewRange(
	parent derivedWindowRange,
	method string,
	call *frontend.CallExpr,
) string {
	switch method {
	case "window":
		start := callArgPath(call, 1)
		count := callArgPath(call, 2)
		lo := addRangeExpr(parent.start, start)
		return fmt.Sprintf("%s[%s..%s]", parent.base, lo, addRangeExpr(lo, count))
	case "prefix":
		count := callArgPath(call, 1)
		return fmt.Sprintf(
			"%s[%s..%s]",
			parent.base,
			parent.start,
			addRangeExpr(parent.start, count),
		)
	case "suffix":
		start := callArgPath(call, 1)
		return fmt.Sprintf("%s[%s..%s]", parent.base, addRangeExpr(parent.start, start), parent.end)
	default:
		return fmt.Sprintf("%s[%s..%s]", parent.base, parent.start, parent.end)
	}
}

func addRangeExpr(left string, right string) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || left == "0" {
		return right
	}
	if right == "" || right == "0" {
		return left
	}
	var constSum int64
	terms := []string{}
	addTerm := func(term string) {
		term = strings.TrimSpace(term)
		if term == "" || term == "0" {
			return
		}
		if value, err := strconv.ParseInt(term, 10, 64); err == nil {
			constSum += value
			return
		}
		terms = append(terms, term)
	}
	for _, part := range strings.Split(left, "+") {
		addTerm(part)
	}
	for _, part := range strings.Split(right, "+") {
		addTerm(part)
	}
	out := []string{}
	if constSum != 0 {
		out = append(out, strconv.FormatInt(constSum, 10))
	}
	out = append(out, terms...)
	if len(out) == 0 {
		return "0"
	}
	return strings.Join(out, "+")
}

func baseSliceViewRange(method string, source string, call *frontend.CallExpr) string {
	if source == "" {
		source = "source"
	}
	switch method {
	case "window":
		start := callArgPath(call, 1)
		count := callArgPath(call, 2)
		return fmt.Sprintf("%s[%s..%s]", source, start, addRangeExpr(start, count))
	case "prefix":
		count := callArgPath(call, 1)
		return fmt.Sprintf("%s[0..%s]", source, count)
	case "suffix":
		start := callArgPath(call, 1)
		return fmt.Sprintf("%s[%s..len]", source, start)
	default:
		return source + "[view]"
	}
}

func staticInvalidStringViewCall(name string, call *frontend.CallExpr) bool {
	if call == nil {
		return false
	}
	if target, aliasOK := semantics.ResolveBuiltinAlias(name); aliasOK {
		name = target
	}
	if !strings.HasPrefix(name, "core.string_") {
		return false
	}
	sourceLen, knownLen := staticStringByteLen(callArg(call, 0))
	if !knownLen {
		return false
	}
	switch name {
	case "core.string_window":
		if len(call.Args) != 3 {
			return false
		}
		start, startKnown := evalConstInt64(call.Args[1])
		count, countKnown := evalConstInt64(call.Args[2])
		if !startKnown || !countKnown {
			return false
		}
		return start < 0 || count < 0 || start > sourceLen || count > sourceLen-start
	case "core.string_prefix":
		if len(call.Args) != 2 {
			return false
		}
		count, known := evalConstInt64(call.Args[1])
		if !known {
			return false
		}
		return count < 0 || count > sourceLen
	case "core.string_suffix":
		if len(call.Args) != 2 {
			return false
		}
		start, known := evalConstInt64(call.Args[1])
		if !known {
			return false
		}
		return start < 0 || start > sourceLen
	default:
		return false
	}
}

func staticStringByteLen(expr frontend.Expr) (int64, bool) {
	lit, ok := expr.(*frontend.StringLitExpr)
	if !ok || lit == nil {
		return 0, false
	}
	return int64(len(lit.Value)), true
}

func elementSize(elem string) int {
	switch elem {
	case "u8":
		return 1
	case "u16":
		return 2
	case "i32", "bool":
		return 4
	case "raw_bytes":
		return 1
	default:
		return 0
	}
}

func sliceViewElementLayout(valueType string) (int, int) {
	switch valueType {
	case "str", "String", "[]u8":
		return 1, 0
	case "[]u16":
		return 2, 1
	case "[]i32", "[]bool":
		return 4, 2
	default:
		return 0, 0
	}
}

func allocationLengthArg(name string, call *frontend.CallExpr) frontend.Expr {
	if call == nil {
		return nil
	}
	index := 0
	if strings.HasPrefix(name, "core.island_make_") {
		index = 1
	}
	if index < 0 || index >= len(call.Args) {
		return nil
	}
	return call.Args[index]
}

func callArg(call *frontend.CallExpr, index int) frontend.Expr {
	if call == nil || index < 0 || index >= len(call.Args) {
		return nil
	}
	return call.Args[index]
}

func callResultPath(call *frontend.CallExpr) string {
	if call == nil {
		return ""
	}
	name := call.Name
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	if _, ok := makeSliceElem(name); ok {
		return syntheticCallPath("alloc", call)
	}
	if name == "core.alloc_bytes" {
		return syntheticCallPath("raw_alloc", call)
	}
	if rawSliceBuiltin(name) {
		return syntheticCallPath("raw_view", call)
	}
	if _, ok := sliceCopyElem(name); ok || stringCopyBuiltin(name) {
		return syntheticCallPath("copy", call)
	}
	if _, ok := sliceBorrowElem(name); ok || stringBorrowBuiltin(name) {
		return syntheticCallPath("borrow", call)
	}
	if _, _, ok := sliceViewElem(name); ok {
		return syntheticCallPath("slice_view", call)
	}
	if _, _, ok := stringViewBuiltin(name); ok {
		return syntheticCallPath("slice_view", call)
	}
	return ""
}

func exprStoresDirectlyIntoTarget(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	return callResultPath(call) != ""
}

func syntheticCallPath(prefix string, call *frontend.CallExpr) string {
	if call == nil || (call.At.Line == 0 && call.At.Col == 0) {
		return prefix
	}
	return fmt.Sprintf("%s_%d_%d", prefix, call.At.Line, call.At.Col)
}

func firstArgPath(call *frontend.CallExpr) string {
	if call == nil || len(call.Args) == 0 {
		return ""
	}
	return exprPath(call.Args[0])
}

func callArgPath(call *frontend.CallExpr, index int) string {
	if call == nil || index < 0 || index >= len(call.Args) {
		return "?"
	}
	path := exprPath(call.Args[index])
	if path == "" {
		return "expr"
	}
	return path
}

func callInputs(args []frontend.Expr) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if input := exprPath(arg); input != "" {
			out = append(out, input)
		}
	}
	return out
}

func isMemoryBackedType(typeName string) bool {
	return strings.HasPrefix(typeName, "[]") || typeName == "str" || typeName == "String"
}

func sourceString(pos frontend.Position) string {
	if pos.Line == 0 && pos.Col == 0 && pos.File == "" {
		return ""
	}
	return frontend.FormatPos(pos)
}

func exprPath(expr frontend.Expr) string {
	switch e := expr.(type) {
	case nil:
		return ""
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.FieldAccessExpr:
		base := exprPath(e.Base)
		if base == "" {
			return e.Field
		}
		return base + "." + e.Field
	case *frontend.NumberExpr:
		return fmt.Sprintf("%d", e.Value)
	case *frontend.StringLitExpr:
		return stringLiteralPath(string(e.Value))
	case *frontend.CallExpr:
		return callResultPath(e)
	case *frontend.UnaryExpr:
		x := exprPath(e.X)
		if x == "" {
			return ""
		}
		switch e.Op {
		case frontend.TokenMinus:
			return "-" + x
		default:
			return ""
		}
	case *frontend.BinaryExpr:
		left := exprPath(e.Left)
		right := exprPath(e.Right)
		if left == "" || right == "" {
			return ""
		}
		op := plirTokenString(e.Op)
		if op == "" {
			return ""
		}
		return left + " " + op + " " + right
	default:
		return ""
	}
}

func stringLiteralPath(value string) string {
	part := strings.NewReplacer(
		" ", "_",
		"\t", "_",
		"\n", "_",
		"\r", "_",
		".", "_",
		":", "_",
		"/", "_",
		"\\", "_",
		"\"", "_",
		"'", "_",
	).Replace(value)
	if part == "" {
		part = "empty"
	}
	if len(part) > 32 {
		part = part[:32]
	}
	return fmt.Sprintf("string:%d:%s", len(value), part)
}

func evalConstInt64(expr frontend.Expr) (int64, bool) {
	switch e := expr.(type) {
	case nil:
		return 0, false
	case *frontend.NumberExpr:
		return int64(e.Value), true
	case *frontend.UnaryExpr:
		v, ok := evalConstInt64(e.X)
		if !ok {
			return 0, false
		}
		switch e.Op {
		case frontend.TokenMinus:
			return -v, true
		default:
			return 0, false
		}
	case *frontend.BinaryExpr:
		left, ok := evalConstInt64(e.Left)
		if !ok {
			return 0, false
		}
		right, ok := evalConstInt64(e.Right)
		if !ok {
			return 0, false
		}
		switch e.Op {
		case frontend.TokenPlus:
			return left + right, true
		case frontend.TokenMinus:
			return left - right, true
		case frontend.TokenStar:
			return left * right, true
		case frontend.TokenSlash:
			if right == 0 {
				return 0, false
			}
			return left / right, true
		case frontend.TokenPercent:
			if right == 0 {
				return 0, false
			}
			return left % right, true
		default:
			return 0, false
		}
	default:
		return 0, false
	}
}

func checkedAddInt64(left int64, right int64) (int64, bool) {
	if right > 0 && left > math.MaxInt64-right {
		return 0, false
	}
	if right < 0 && left < math.MinInt64-right {
		return 0, false
	}
	return left + right, true
}

func plirTokenString(op frontend.TokenType) string {
	switch op {
	case frontend.TokenPlus:
		return "+"
	case frontend.TokenMinus:
		return "-"
	case frontend.TokenStar:
		return "*"
	case frontend.TokenSlash:
		return "/"
	case frontend.TokenPercent:
		return "%"
	case frontend.TokenLess:
		return "<"
	case frontend.TokenLessEq:
		return "<="
	case frontend.TokenGreater:
		return ">"
	case frontend.TokenGreaterEq:
		return ">="
	case frontend.TokenEqEq:
		return "=="
	case frontend.TokenBangEq:
		return "!="
	default:
		return ""
	}
}

func staticInvalidAllocationExpr(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	name := call.Name
	elem, ok := makeSliceElem(name)
	if !ok {
		if target, aliasOK := semantics.ResolveBuiltinAlias(name); aliasOK {
			name = target
			elem, ok = makeSliceElem(name)
		}
	}
	if !ok {
		return false
	}
	length, known := evalConstInt64(allocationLengthArg(name, call))
	if !known {
		return false
	}
	if length < 0 {
		return true
	}
	size := int64(elementSize(elem))
	return size > 0 && length*size > 2147483647
}

func staticInvalidIterableExpr(expr frontend.Expr) bool {
	return staticInvalidAllocationExpr(expr) || staticInvalidStringViewCallExpr(expr)
}

func staticInvalidStringViewCallExpr(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	return staticInvalidStringViewCall(call.Name, call)
}

func FormatText(prog *Program) string {
	if prog == nil {
		return ""
	}
	var b strings.Builder
	for _, fn := range prog.Funcs {
		fmt.Fprintf(&b, "func %s\n", fn.Name)
		for _, value := range fn.Values {
			fmt.Fprintf(&b, "  value %s: %s", value.ID, value.Type)
			if value.Provenance.Kind != "" {
				fmt.Fprintf(&b, " provenance: %s", value.Provenance.Root)
				if value.Provenance.Root == "" {
					fmt.Fprintf(&b, " provenance: %s", value.Provenance.Kind)
				}
			}
			if value.Region != "" {
				fmt.Fprintf(&b, " region: %s", value.Region)
			}
			fmt.Fprintln(&b)
		}
		for _, fact := range fn.Facts {
			fmt.Fprintf(&b, "  fact %s", fact.Kind)
			if fact.ValueID != "" {
				fmt.Fprintf(&b, " value: %s", fact.ValueID)
			}
			if fact.Range != "" {
				fmt.Fprintf(&b, " range: %s", fact.Range)
			}
			if fact.IslandID != "" {
				fmt.Fprintf(
					&b,
					" island: %s epoch: %d base: %s",
					fact.IslandID,
					fact.Epoch,
					fact.BaseID,
				)
			}
			if fact.ProofID != "" {
				fmt.Fprintf(&b, " proof: %s", fact.ProofID)
			}
			if fact.Reason != "" {
				fmt.Fprintf(&b, " reason: %s", fact.Reason)
			}
			fmt.Fprintln(&b)
		}
		for _, rf := range fn.RangeFacts {
			fmt.Fprintf(
				&b,
				"  range %s lower: %s upper: %s proof: %s",
				rf.Value,
				formatBound(rf.Lower),
				formatBound(rf.Upper),
				rf.ProofID,
			)
			if rf.Reason != "" {
				fmt.Fprintf(&b, " reason: %s", rf.Reason)
			}
			if len(rf.Derivation) > 0 {
				fmt.Fprintf(&b, " derivation: %s", strings.Join(rf.Derivation, ","))
			}
			fmt.Fprintln(&b)
		}
		for _, block := range fn.Blocks {
			fmt.Fprintf(&b, "  block %s", block.ID)
			if block.Entry {
				fmt.Fprintf(&b, " entry")
			}
			if block.Exit {
				fmt.Fprintf(&b, " exit")
			}
			if len(block.Succs) > 0 {
				fmt.Fprintf(&b, " succs: %s", strings.Join(block.Succs, ","))
			}
			fmt.Fprintln(&b)
		}
		for _, guard := range fn.ProofGuards {
			fmt.Fprintf(
				&b,
				"  proof %s kind: %s block: %s guard: %s",
				guard.ID,
				guard.Kind,
				guard.Block,
				guard.Condition,
			)
			if guard.Reason != "" {
				fmt.Fprintf(&b, " reason: %s", guard.Reason)
			}
			fmt.Fprintln(&b)
		}
		for _, term := range fn.ProofTerms {
			fmt.Fprintf(
				&b,
				"  proof_term %s kind: %s subject: %s index: %s op: %s range: %s",
				term.ID,
				term.Kind,
				term.SubjectBaseID,
				term.IndexValueID,
				term.Operation,
				term.Range,
			)
			if term.IslandID != "" {
				fmt.Fprintf(
					&b,
					" island: %s epoch: %d base: %s",
					term.IslandID,
					term.Epoch,
					term.BaseID,
				)
			}
			fmt.Fprintln(&b)
		}
		for _, op := range fn.Ops {
			fmt.Fprintf(&b, "  op %s %s", op.ID, op.Kind)
			if op.Block != "" {
				fmt.Fprintf(&b, " block: %s", op.Block)
			}
			if op.Note != "" {
				fmt.Fprintf(&b, " %s", op.Note)
			}
			fmt.Fprintln(&b)
		}
	}
	return b.String()
}

func formatBound(bound Bound) string {
	switch bound.Kind {
	case BoundConst:
		return fmt.Sprintf("%d", bound.Const)
	case BoundSymbol:
		return bound.Symbol
	case BoundSymbolMinus:
		return fmt.Sprintf("%s-%d", bound.Symbol, bound.Const)
	default:
		return string(bound.Kind)
	}
}
