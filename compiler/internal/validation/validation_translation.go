package validation

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
)

func ValidateTranslation(before *ir.IRProgram, after *ir.IRProgram) (TranslationReport, error) {
	if err := lower.VerifyProgram(before); err != nil {
		return TranslationReport{}, fmt.Errorf("translation validation: input IR invalid: %w", err)
	}
	if err := lower.VerifyProgram(after); err != nil {
		return TranslationReport{}, fmt.Errorf("translation validation: output IR invalid: %w", err)
	}
	beforeNames := functionNames(before)
	afterNames := functionNames(after)
	if len(beforeNames) != len(afterNames) {
		return TranslationReport{}, fmt.Errorf(
			"translation validation: function count changed from %d to %d",
			len(beforeNames),
			len(afterNames),
		)
	}
	for i := range beforeNames {
		if beforeNames[i] != afterNames[i] {
			return TranslationReport{}, fmt.Errorf(
				"translation validation: function set changed: before=%v after=%v",
				beforeNames,
				afterNames,
			)
		}
	}
	beforeFuncs := functionsByName(before)
	afterFuncs := functionsByName(after)
	for _, name := range beforeNames {
		if err := validateTranslationFunctionShape(beforeFuncs[name], afterFuncs[name]); err != nil {
			return TranslationReport{}, err
		}
	}
	beforeProofs, err := CheckBoundsProofs(before)
	if err != nil {
		return TranslationReport{}, fmt.Errorf(
			"translation validation: input proof validation failed: %w",
			err,
		)
	}
	afterProofs, err := CheckBoundsProofs(after)
	if err != nil {
		return TranslationReport{}, fmt.Errorf(
			"translation validation: output proof validation failed: %w",
			err,
		)
	}
	proofFactsCompared, err := validateProofFactMultiset(beforeProofs, afterProofs)
	if err != nil {
		return TranslationReport{}, err
	}
	semanticChecks, err := validateSemanticLocalEquivalence(beforeFuncs, afterFuncs, beforeNames)
	if err != nil {
		return TranslationReport{}, err
	}
	differentialSamples, err := validateDifferentialSamples(beforeFuncs, afterFuncs, beforeNames)
	if err != nil {
		return TranslationReport{}, err
	}
	return TranslationReport{
		FunctionsCompared:   len(beforeNames),
		Functions:           beforeNames,
		ProofFactsCompared:  proofFactsCompared,
		SemanticLocalChecks: semanticChecks,
		DifferentialSamples: differentialSamples,
	}, nil
}

func BuildOptimizationValidationMetadata(
	before *ir.IRProgram,
	after *ir.IRProgram,
	options OptimizationMetadataOptions,
) (OptimizationValidationMetadata, error) {
	report, err := ValidateTranslation(before, after)
	if err != nil {
		return OptimizationValidationMetadata{}, err
	}
	meta := OptimizationValidationMetadata{
		SchemaVersion:             "tetra.translation.validation.metadata.v1",
		PassName:                  options.PassName,
		InputKind:                 options.InputKind,
		OutputKind:                options.OutputKind,
		InputVerifier:             options.InputVerifier,
		OutputVerifier:            options.OutputVerifier,
		ValidationStrategy:        options.ValidationStrategy,
		RequiredFacts:             append([]string(nil), options.RequiredFacts...),
		PreservedFacts:            append([]string(nil), options.PreservedFacts...),
		InvalidatedFacts:          append([]string(nil), options.InvalidatedFacts...),
		ProofRule:                 options.ProofRule,
		TranslationValidationHook: options.TranslationValidationHook,
		ReportRows:                append([]string(nil), options.ReportRows...),
		NegativeTestMarker:        options.NegativeTestMarker,
		ProfileInputPolicy:        options.ProfileInputPolicy,
		ProfileInputDigest:        options.ProfileInputDigest,
		ProfileInputSchemaVersion: options.ProfileInputSchemaVersion,
		BeforeHash:                stableIRHash(before),
		AfterHash:                 stableIRHash(after),
		Functions:                 append([]string(nil), report.Functions...),
		Translation:               report,
	}
	if err := ValidateOptimizationValidationMetadata(meta); err != nil {
		return OptimizationValidationMetadata{}, err
	}
	return meta, nil
}

func ValidateOptimizationValidationMetadata(meta OptimizationValidationMetadata) error {
	if meta.SchemaVersion != "tetra.translation.validation.metadata.v1" {
		return fmt.Errorf(
			"translation validation metadata: schema_version is %q",
			meta.SchemaVersion,
		)
	}
	if strings.TrimSpace(meta.PassName) == "" {
		return fmt.Errorf("translation validation metadata: missing pass_name")
	}
	if strings.TrimSpace(meta.InputKind) == "" || strings.TrimSpace(meta.OutputKind) == "" {
		return fmt.Errorf("translation validation metadata: missing IR kind")
	}
	if strings.TrimSpace(meta.InputVerifier) == "" || strings.TrimSpace(meta.OutputVerifier) == "" {
		return fmt.Errorf("translation validation metadata: missing input/output verifier")
	}
	if strings.TrimSpace(meta.ValidationStrategy) == "" {
		return fmt.Errorf("translation validation metadata: missing validation strategy")
	}
	if strings.TrimSpace(meta.ProofRule) == "" {
		return fmt.Errorf("translation validation metadata: missing proof rule")
	}
	if strings.TrimSpace(meta.TranslationValidationHook) == "" {
		return fmt.Errorf("translation validation metadata: missing translation validation hook")
	}
	if len(meta.ReportRows) == 0 {
		return fmt.Errorf("translation validation metadata: missing report rows")
	}
	if strings.TrimSpace(meta.NegativeTestMarker) == "" {
		return fmt.Errorf("translation validation metadata: missing negative-test marker")
	}
	if strings.TrimSpace(meta.ProfileInputPolicy) == "" {
		return fmt.Errorf("translation validation metadata: missing profile_input_policy")
	}
	if meta.ProfileInputDigest != "" {
		if !isStableHash(meta.ProfileInputDigest) {
			return fmt.Errorf(
				"translation validation metadata: profile input digest must be sha256",
			)
		}
		if strings.TrimSpace(meta.ProfileInputSchemaVersion) == "" {
			return fmt.Errorf(
				"translation validation metadata: missing profile input schema version",
			)
		}
	}
	if meta.ProfileInputSchemaVersion != "" && meta.ProfileInputDigest == "" {
		return fmt.Errorf(
			"translation validation metadata: profile input schema version requires digest",
		)
	}
	if !isStableHash(meta.BeforeHash) || !isStableHash(meta.AfterHash) {
		return fmt.Errorf("translation validation metadata: before/after hashes must be sha256")
	}
	if meta.Translation.FunctionsCompared == 0 || len(meta.Functions) == 0 {
		return fmt.Errorf("translation validation metadata: missing compared functions")
	}
	return nil
}

func stableIRHash(prog *ir.IRProgram) string {
	sum := sha256.Sum256([]byte(stableIRText(prog)))
	return fmt.Sprintf("sha256:%x", sum)
}

func isStableHash(value string) bool {
	return strings.HasPrefix(value, "sha256:") && len(value) == len("sha256:")+64
}

func stableIRText(prog *ir.IRProgram) string {
	if prog == nil {
		return "<nil>\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "main:%s index:%d funcs:%d\n", prog.MainName, prog.MainIndex, len(prog.Funcs))
	for _, fn := range prog.Funcs {
		fmt.Fprintf(
			&b,
			"func:%s export:%s params:%d locals:%d returns:%d budget:%t consent:%t\n",
			fn.Name,
			fn.ExportName,
			fn.ParamSlots,
			fn.LocalSlots,
			fn.ReturnSlots,
			fn.Policy.HasBudget,
			fn.Policy.HasConsent,
		)
		for _, instr := range fn.Instrs {
			fmt.Fprintf(
				&b,
				"  kind:%d imm:%d local:%d label:%d name:%s args:%d rets:%d proof:%s str:%x\n",
				instr.Kind,
				instr.Imm,
				instr.Local,
				instr.Label,
				instr.Name,
				instr.ArgSlots,
				instr.RetSlots,
				instr.ProofID,
				instr.Str,
			)
		}
	}
	return b.String()
}

func functionNames(prog *ir.IRProgram) []string {
	if prog == nil {
		return nil
	}
	out := make([]string, 0, len(prog.Funcs))
	for _, fn := range prog.Funcs {
		out = append(out, fn.Name)
	}
	sort.Strings(out)
	return out
}

func functionsByName(prog *ir.IRProgram) map[string]ir.IRFunc {
	out := map[string]ir.IRFunc{}
	if prog == nil {
		return out
	}
	for _, fn := range prog.Funcs {
		out[fn.Name] = fn
	}
	return out
}

func validateTranslationFunctionShape(before ir.IRFunc, after ir.IRFunc) error {
	if before.ParamSlots != after.ParamSlots {
		return fmt.Errorf(
			"translation validation: %s param slot count changed from %d to %d",
			before.Name,
			before.ParamSlots,
			after.ParamSlots,
		)
	}
	if before.ReturnSlots != after.ReturnSlots {
		return fmt.Errorf(
			"translation validation: %s return slot count changed from %d to %d",
			before.Name,
			before.ReturnSlots,
			after.ReturnSlots,
		)
	}
	if before.ExportName != after.ExportName {
		return fmt.Errorf(
			"translation validation: %s export name changed from %q to %q",
			before.Name,
			before.ExportName,
			after.ExportName,
		)
	}
	if before.Policy != after.Policy {
		return fmt.Errorf("translation validation: %s policy changed", before.Name)
	}
	return nil
}

func validateProofFactMultiset(before ProofReport, after ProofReport) (int, error) {
	beforeSet := proofFactMultiset(before)
	afterSet := proofFactMultiset(after)
	if !sameStringIntMap(beforeSet, afterSet) {
		return 0, fmt.Errorf(
			"translation validation: proof facts changed: before=%s after=%s",
			formatStringIntMap(beforeSet),
			formatStringIntMap(afterSet),
		)
	}
	total := 0
	for _, count := range beforeSet {
		total += count
	}
	return total, nil
}

func proofFactMultiset(report ProofReport) map[string]int {
	out := map[string]int{}
	for _, removed := range report.RemovedChecks {
		key := removed.Function + "\x00" + removed.Kind + "\x00" + removed.ProofID
		out[key]++
	}
	return out
}

func sameStringIntMap(left map[string]int, right map[string]int) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func formatStringIntMap(values map[string]int) string {
	if len(values) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteByte('{')
	for i, key := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(strings.ReplaceAll(key, "\x00", "/"))
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(values[key]))
	}
	b.WriteByte('}')
	return b.String()
}

type symbolicValue struct {
	expr    string
	isConst bool
	value   int32
}

func validateSemanticLocalEquivalence(
	before map[string]ir.IRFunc,
	after map[string]ir.IRFunc,
	names []string,
) (int, error) {
	checks := 0
	for _, name := range names {
		beforeExpr, beforeOK := symbolicReturnExpr(before[name])
		afterExpr, afterOK := symbolicReturnExpr(after[name])
		if !beforeOK || !afterOK {
			continue
		}
		checks++
		if beforeExpr != afterExpr {
			return checks, fmt.Errorf(
				"translation validation: semantic local equivalence failed for %s: before=%s after=%s",
				name,
				beforeExpr,
				afterExpr,
			)
		}
	}
	return checks, nil
}

func symbolicReturnExpr(fn ir.IRFunc) (string, bool) {
	if fn.ReturnSlots > 1 {
		return "", false
	}
	stack := []symbolicValue{}
	locals := map[int]symbolicValue{}
	for i := 0; i < fn.ParamSlots; i++ {
		locals[i] = symbolicExpr(fmt.Sprintf("param%d", i))
	}
	for _, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRConstI32:
			stack = append(stack, symbolicConst(instr.Imm))
		case ir.IRLoadLocal:
			value, ok := locals[instr.Local]
			if !ok {
				value = symbolicExpr(fmt.Sprintf("local%d", instr.Local))
			}
			stack = append(stack, value)
		case ir.IRStoreLocal:
			value, ok := popSymbolic(&stack)
			if !ok {
				return "", false
			}
			locals[instr.Local] = value
		case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
			ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
			right, left, ok := pop2Symbolic(&stack)
			if !ok {
				return "", false
			}
			stack = append(stack, symbolicBinary(instr.Kind, left, right))
		case ir.IRNegI32:
			value, ok := popSymbolic(&stack)
			if !ok {
				return "", false
			}
			stack = append(stack, symbolicNeg(value))
		case ir.IRReturn:
			if fn.ReturnSlots == 0 {
				return "void", len(stack) == 0
			}
			value, ok := popSymbolic(&stack)
			if !ok || len(stack) != 0 {
				return "", false
			}
			return value.expr, true
		default:
			return "", false
		}
	}
	return "", false
}

func symbolicConst(value int32) symbolicValue {
	return symbolicValue{expr: strconv.FormatInt(int64(value), 10), isConst: true, value: value}
}

func symbolicExpr(expr string) symbolicValue {
	return symbolicValue{expr: expr}
}

func popSymbolic(stack *[]symbolicValue) (symbolicValue, bool) {
	values := *stack
	if len(values) == 0 {
		return symbolicValue{}, false
	}
	value := values[len(values)-1]
	*stack = values[:len(values)-1]
	return value, true
}

func pop2Symbolic(stack *[]symbolicValue) (right symbolicValue, left symbolicValue, ok bool) {
	right, ok = popSymbolic(stack)
	if !ok {
		return symbolicValue{}, symbolicValue{}, false
	}
	left, ok = popSymbolic(stack)
	if !ok {
		return symbolicValue{}, symbolicValue{}, false
	}
	return right, left, true
}

func symbolicBinary(kind ir.IRInstrKind, left symbolicValue, right symbolicValue) symbolicValue {
	if left.isConst && right.isConst {
		if value, ok := evalBinaryI32(kind, left.value, right.value); ok {
			return symbolicConst(value)
		}
	}
	switch kind {
	case ir.IRAddI32:
		if isSymbolicConst(right, 0) {
			return left
		}
		if isSymbolicConst(left, 0) {
			return right
		}
	case ir.IRSubI32:
		if isSymbolicConst(right, 0) {
			return left
		}
	case ir.IRMulI32:
		if isSymbolicConst(right, 1) {
			return left
		}
		if isSymbolicConst(left, 1) {
			return right
		}
		if isSymbolicConst(left, 0) || isSymbolicConst(right, 0) {
			return symbolicConst(0)
		}
	}
	if left.expr == right.expr {
		switch kind {
		case ir.IRCmpEqI32, ir.IRCmpGeI32, ir.IRCmpLeI32:
			return symbolicConst(1)
		case ir.IRCmpNeI32, ir.IRCmpLtI32, ir.IRCmpGtI32:
			return symbolicConst(0)
		}
	}
	leftExpr, rightExpr := left.expr, right.expr
	if isCommutativeSymbolicBinaryOp(kind) && rightExpr < leftExpr {
		leftExpr, rightExpr = rightExpr, leftExpr
	}
	if isMirroredComparisonSymbolicBinaryOp(kind) && rightExpr < leftExpr {
		leftExpr, rightExpr = rightExpr, leftExpr
		kind = mirroredComparisonSymbolicBinaryOp(kind)
	}
	return symbolicExpr(fmt.Sprintf("%s(%s,%s)", symbolicOpName(kind), leftExpr, rightExpr))
}

func symbolicNeg(value symbolicValue) symbolicValue {
	if value.isConst {
		return symbolicConst(-value.value)
	}
	return symbolicExpr("neg(" + value.expr + ")")
}

func isSymbolicConst(value symbolicValue, want int32) bool {
	return value.isConst && value.value == want
}

func isCommutativeSymbolicBinaryOp(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRAddI32, ir.IRMulI32, ir.IRCmpEqI32, ir.IRCmpNeI32:
		return true
	default:
		return false
	}
}

func isMirroredComparisonSymbolicBinaryOp(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRCmpLtI32, ir.IRCmpGtI32, ir.IRCmpLeI32, ir.IRCmpGeI32:
		return true
	default:
		return false
	}
}

func mirroredComparisonSymbolicBinaryOp(kind ir.IRInstrKind) ir.IRInstrKind {
	switch kind {
	case ir.IRCmpLtI32:
		return ir.IRCmpGtI32
	case ir.IRCmpGtI32:
		return ir.IRCmpLtI32
	case ir.IRCmpLeI32:
		return ir.IRCmpGeI32
	case ir.IRCmpGeI32:
		return ir.IRCmpLeI32
	default:
		return kind
	}
}

func symbolicOpName(kind ir.IRInstrKind) string {
	switch kind {
	case ir.IRAddI32:
		return "add"
	case ir.IRSubI32:
		return "sub"
	case ir.IRMulI32:
		return "mul"
	case ir.IRCmpEqI32:
		return "eq"
	case ir.IRCmpLtI32:
		return "lt"
	case ir.IRCmpGtI32:
		return "gt"
	case ir.IRCmpGeI32:
		return "ge"
	case ir.IRCmpLeI32:
		return "le"
	case ir.IRCmpNeI32:
		return "ne"
	default:
		return fmt.Sprintf("ir%d", kind)
	}
}

func validateDifferentialSamples(
	before map[string]ir.IRFunc,
	after map[string]ir.IRFunc,
	names []string,
) (int, error) {
	samples := 0
	for _, name := range names {
		beforeFn := before[name]
		afterFn := after[name]
		if beforeFn.ReturnSlots != 1 || beforeFn.ParamSlots > 2 {
			continue
		}
		for _, args := range translationSampleArgs(beforeFn.ParamSlots) {
			beforeValue, beforeOK := evalStraightLineReturn(beforeFn, args)
			afterValue, afterOK := evalStraightLineReturn(afterFn, args)
			if !beforeOK || !afterOK {
				continue
			}
			samples++
			if beforeValue != afterValue {
				return samples, fmt.Errorf(
					"translation validation: differential mismatch for %s args=%v: before=%d after=%d",
					name,
					args,
					beforeValue,
					afterValue,
				)
			}
		}
	}
	return samples, nil
}

func translationSampleArgs(params int) [][]int32 {
	values := []int32{-2, -1, 0, 1, 2, 7}
	switch params {
	case 0:
		return [][]int32{{}}
	case 1:
		out := make([][]int32, 0, len(values))
		for _, value := range values {
			out = append(out, []int32{value})
		}
		return out
	case 2:
		out := make([][]int32, 0, len(values)*len(values))
		for _, left := range values {
			for _, right := range values {
				out = append(out, []int32{left, right})
			}
		}
		return out
	default:
		return nil
	}
}

func evalStraightLineReturn(fn ir.IRFunc, args []int32) (int32, bool) {
	if len(args) != fn.ParamSlots || fn.ReturnSlots != 1 {
		return 0, false
	}
	stack := []int32{}
	locals := map[int]int32{}
	for i, value := range args {
		locals[i] = value
	}
	for _, instr := range fn.Instrs {
		switch instr.Kind {
		case ir.IRConstI32:
			stack = append(stack, instr.Imm)
		case ir.IRLoadLocal:
			stack = append(stack, locals[instr.Local])
		case ir.IRStoreLocal:
			value, ok := popI32(&stack)
			if !ok {
				return 0, false
			}
			locals[instr.Local] = value
		case ir.IRAddI32, ir.IRSubI32, ir.IRMulI32, ir.IRDivI32, ir.IRModI32,
			ir.IRCmpEqI32, ir.IRCmpLtI32, ir.IRCmpGtI32,
			ir.IRCmpGeI32, ir.IRCmpLeI32, ir.IRCmpNeI32:
			right, left, ok := pop2I32(&stack)
			if !ok {
				return 0, false
			}
			value, ok := evalBinaryI32(instr.Kind, left, right)
			if !ok {
				return 0, false
			}
			stack = append(stack, value)
		case ir.IRNegI32:
			value, ok := popI32(&stack)
			if !ok {
				return 0, false
			}
			stack = append(stack, -value)
		case ir.IRReturn:
			value, ok := popI32(&stack)
			if !ok || len(stack) != 0 {
				return 0, false
			}
			return value, true
		default:
			return 0, false
		}
	}
	return 0, false
}

func popI32(stack *[]int32) (int32, bool) {
	values := *stack
	if len(values) == 0 {
		return 0, false
	}
	value := values[len(values)-1]
	*stack = values[:len(values)-1]
	return value, true
}

func pop2I32(stack *[]int32) (right int32, left int32, ok bool) {
	right, ok = popI32(stack)
	if !ok {
		return 0, 0, false
	}
	left, ok = popI32(stack)
	if !ok {
		return 0, 0, false
	}
	return right, left, true
}

func evalBinaryI32(kind ir.IRInstrKind, left int32, right int32) (int32, bool) {
	switch kind {
	case ir.IRAddI32:
		return left + right, true
	case ir.IRSubI32:
		return left - right, true
	case ir.IRMulI32:
		return left * right, true
	case ir.IRDivI32:
		if right == 0 {
			return 0, false
		}
		return left / right, true
	case ir.IRModI32:
		if right == 0 {
			return 0, false
		}
		return left % right, true
	case ir.IRCmpEqI32:
		return boolToI32(left == right), true
	case ir.IRCmpLtI32:
		return boolToI32(left < right), true
	case ir.IRCmpGtI32:
		return boolToI32(left > right), true
	case ir.IRCmpGeI32:
		return boolToI32(left >= right), true
	case ir.IRCmpLeI32:
		return boolToI32(left <= right), true
	case ir.IRCmpNeI32:
		return boolToI32(left != right), true
	default:
		return 0, false
	}
}

func boolToI32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

func boundsKind(kind ir.IRInstrKind) string {
	switch kind {
	case ir.IRIndexLoadI32, ir.IRIndexLoadI32Unchecked:
		return "i32.load"
	case ir.IRIndexLoadU8, ir.IRIndexLoadU8Unchecked:
		return "u8.load"
	case ir.IRIndexLoadU16, ir.IRIndexLoadU16Unchecked:
		return "u16.load"
	case ir.IRIndexStoreI32:
		return "i32.store"
	case ir.IRIndexStoreU8:
		return "u8.store"
	case ir.IRIndexStoreU16:
		return "u16.store"
	default:
		return fmt.Sprintf("ir.%d", kind)
	}
}
