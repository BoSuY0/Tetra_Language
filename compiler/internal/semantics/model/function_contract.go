package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

const FunctionContractSchemaV1 = "tetra.semantic.function-contract.v1"

const (
	SummaryParamNone    = -1
	SummaryParamUnknown = -2
)

type FunctionContractV1 struct {
	Schema                string
	Name                  string
	Generic               bool
	Public                bool
	Params                []ParamContractV1
	Result                ResultContractV1
	Throws                *ThrowsContractV1
	Async                 bool
	Effects               []string
	Policy                FunctionPolicyContractV1
	TouchesMutableGlobals bool
	Digest                string
}

type ParamContractV1 struct {
	Name      string
	Type      string
	Ownership string
	Callable  *CallableTypeContractV1
}

type ResultContractV1 struct {
	Type            string
	Ownership       string
	Callable        *CallableValueContractV1
	RegionUnknown   bool
	RegionSummary   map[string]int
	ResourceUnknown bool
	ResourceSummary map[string][]ResourceProvenance
}

type ThrowsContractV1 struct {
	Type            string
	ResourceSummary map[string][]ResourceProvenance
}

type FunctionPolicyContractV1 struct {
	NoAlloc   bool
	NoBlock   bool
	Realtime  bool
	HasBudget bool
	Budget    int32
}

type CallableTypeContractV1 struct {
	ParamTypes      []string
	ParamOwnership  []string
	ReturnType      string
	ReturnOwnership string
	ThrowsType      string
	Effects         []string
}

type CallableValueContractV1 struct {
	CallableTypeContractV1
	Symbol                string
	ParamName             string
	Captures              []ClosureCaptureContractV1
	EscapeCaptures        []ClosureCaptureContractV1
	TouchesMutableGlobals bool
	EscapeKind            string
	HandleValue           bool
	Fields                map[string]FunctionFieldContractV1
	EnumPayloadFunctions  map[string]FunctionFieldContractV1
	EnumPayloadFields     map[string]FunctionFieldContractV1
}

type FunctionFieldContractV1 struct {
	CallableTypeContractV1
	Symbol                string
	ParamName             string
	Captures              []ClosureCaptureContractV1
	EscapeCaptures        []ClosureCaptureContractV1
	TouchesMutableGlobals bool
	ReturnSnapshotAlias   bool
	DirectSnapshotAlias   bool
	EscapeKind            string
	HandleValue           bool
}

type ClosureCaptureContractV1 struct {
	Name    string
	Type    string
	Mutable bool
}

func ValidateFuncSigContract(name string, sig FuncSig, types map[string]*TypeInfo) error {
	paramCount := len(sig.ParamTypes)
	if err := validateOptionalParamLen("ParamNames", len(sig.ParamNames), paramCount); err != nil {
		return err
	}
	if err := validateOptionalParamLen("ParamOwnership", len(sig.ParamOwnership), paramCount); err != nil {
		return err
	}
	if err := validateOptionalParamLen("ParamFunctionTypes", len(sig.ParamFunctionTypes), paramCount); err != nil {
		return err
	}
	if err := validateOptionalParamLen("ParamFunctionParams", len(sig.ParamFunctionParams), paramCount); err != nil {
		return err
	}
	if err := validateOptionalParamLen("ParamFunctionOwnership", len(sig.ParamFunctionOwnership), paramCount); err != nil {
		return err
	}
	if err := validateOptionalParamLen("ParamFunctionReturns", len(sig.ParamFunctionReturns), paramCount); err != nil {
		return err
	}
	if err := validateOptionalParamLen("ParamFunctionReturnOwnership", len(sig.ParamFunctionReturnOwnership), paramCount); err != nil {
		return err
	}
	if err := validateOptionalParamLen("ParamFunctionThrows", len(sig.ParamFunctionThrows), paramCount); err != nil {
		return err
	}
	if err := validateOptionalParamLen("ParamFunctionEffects", len(sig.ParamFunctionEffects), paramCount); err != nil {
		return err
	}
	for i, ownership := range sig.ParamOwnership {
		if err := validateOwnership("ParamOwnership", i, ownership); err != nil {
			return err
		}
	}
	if err := validateOwnershipValue("ReturnOwnership", sig.ReturnOwnership); err != nil {
		return err
	}
	if err := validateEffects("Effects", sig.Effects); err != nil {
		return err
	}
	if err := validateSummaryParam("ReturnRegionParam", sig.ReturnRegionParam, paramCount); err != nil {
		return err
	}
	if err := validateSummaryParam("ReturnResourceParam", sig.ReturnResourceParam, paramCount); err != nil {
		return err
	}
	if sig.ReturnResourceParam >= 0 {
		if err := validateSummaryPath("ReturnResourcePath", sig.ReturnResourcePath); err != nil {
			return err
		}
	}
	if err := validateRegionSummary("ReturnRegionSummary", sig.ReturnRegionSummary, paramCount); err != nil {
		return err
	}
	if err := validateResourceSummary("ReturnResourceSummary", sig.ReturnResourceSummary, paramCount); err != nil {
		return err
	}
	if err := validateResourceSummary("ThrowResourceSummary", sig.ThrowResourceSummary, paramCount); err != nil {
		return err
	}
	if len(sig.ThrowResourceSummary) > 0 && sig.ThrowsType == "" {
		return fmt.Errorf("%s: ThrowResourceSummary requires ThrowsType", name)
	}
	if sig.Budget < 0 {
		return fmt.Errorf("%s: budget < 0", name)
	}
	if sig.HasRealtime && (!sig.HasNoAlloc || !sig.HasNoBlock) {
		return fmt.Errorf("%s: realtime requires noalloc and noblock", name)
	}
	if sig.Public && sig.Generic {
		return fmt.Errorf("%s: public generic export inconsistency", name)
	}
	for i := 0; i < paramCount; i++ {
		if paramFunctionTypeAt(sig, i) {
			if err := validateCallableType(fmt.Sprintf("Param[%d].Callable", i), paramCallableType(sig, i)); err != nil {
				return err
			}
		}
	}
	if sig.ReturnFunctionType {
		if err := validateCallableType("Result.Callable", returnCallableType(sig)); err != nil {
			return err
		}
	}
	if sig.ReturnFunctionHandleValue {
		if err := validateCallableEscape("Result.Callable", sig.ReturnFunctionEscapeKind, true); err != nil {
			return err
		}
		if sig.ReturnSlots != CallableHandleSlotCount {
			return fmt.Errorf("%s: ReturnFunctionHandleValue requires ReturnSlots=%d, got %d",
				name,
				CallableHandleSlotCount,
				sig.ReturnSlots,
			)
		}
	} else if sig.ReturnFunctionEscapeKind != "" {
		if err := validateCallableEscape("Result.Callable", sig.ReturnFunctionEscapeKind, false); err != nil {
			return err
		}
	}
	if err := validateFunctionFieldMap("ReturnFunctionFields", sig.ReturnFunctionFields); err != nil {
		return err
	}
	if err := validateFunctionFieldMap("ReturnEnumPayloadFunctions", sig.ReturnEnumPayloadFunctions); err != nil {
		return err
	}
	if err := validateFunctionFieldMap("ReturnEnumPayloadFields", sig.ReturnEnumPayloadFields); err != nil {
		return err
	}
	return nil
}

func ProjectFunctionContractV1(name string, sig FuncSig) (FunctionContractV1, error) {
	if err := ValidateFuncSigContract(name, sig, nil); err != nil {
		return FunctionContractV1{}, err
	}
	sig = CloneFuncSig(sig)
	contract := FunctionContractV1{
		Schema:  FunctionContractSchemaV1,
		Name:    name,
		Generic: sig.Generic,
		Public:  sig.Public,
		Params:  projectParams(sig),
		Result: ResultContractV1{
			Type:            sig.ReturnType,
			Ownership:       normalizeOwnership(sig.ReturnOwnership),
			Callable:        projectReturnCallable(sig),
			RegionUnknown:   sig.ReturnRegionParam == SummaryParamUnknown && len(sig.ReturnRegionSummary) == 0,
			RegionSummary:   projectRegionSummary(sig),
			ResourceUnknown: sig.ReturnResourceParam == SummaryParamUnknown && len(sig.ReturnResourceSummary) == 0,
			ResourceSummary: projectResourceSummary(sig),
		},
		Throws: projectThrows(sig),
		Async:  sig.Async,
		Effects: sortedEffectsCopy(
			sig.Effects,
		),
		Policy: FunctionPolicyContractV1{
			NoAlloc:   sig.HasNoAlloc,
			NoBlock:   sig.HasNoBlock,
			Realtime:  sig.HasRealtime,
			HasBudget: sig.HasBudget,
			Budget:    sig.Budget,
		},
		TouchesMutableGlobals: sig.TouchesMutableGlobals,
	}
	digest, err := functionContractDigest(contract)
	if err != nil {
		return FunctionContractV1{}, err
	}
	contract.Digest = digest
	return contract, nil
}

func FunctionContractDigest(name string, sig FuncSig) (string, error) {
	contract, err := ProjectFunctionContractV1(name, sig)
	if err != nil {
		return "", err
	}
	return contract.Digest, nil
}

func CloneFuncSig(sig FuncSig) FuncSig {
	out := sig
	out.ParamNames = cloneStrings(sig.ParamNames)
	out.ParamTypes = cloneStrings(sig.ParamTypes)
	out.ParamFunctionTypes = cloneBools(sig.ParamFunctionTypes)
	out.ParamFunctionParams = cloneStringSlices(sig.ParamFunctionParams)
	out.ParamFunctionOwnership = cloneStringSlices(sig.ParamFunctionOwnership)
	out.ParamFunctionReturns = cloneStrings(sig.ParamFunctionReturns)
	out.ParamFunctionReturnOwnership = cloneStrings(sig.ParamFunctionReturnOwnership)
	out.ParamFunctionThrows = cloneStrings(sig.ParamFunctionThrows)
	out.ParamFunctionEffects = cloneStringSlices(sig.ParamFunctionEffects)
	out.ParamOwnership = cloneStrings(sig.ParamOwnership)
	out.ReturnFunctionParams = cloneStrings(sig.ReturnFunctionParams)
	out.ReturnFunctionParamOwnership = cloneStrings(sig.ReturnFunctionParamOwnership)
	out.ReturnFunctionEffects = cloneStrings(sig.ReturnFunctionEffects)
	out.ReturnFunctionCaptures = cloneClosureCaptures(sig.ReturnFunctionCaptures)
	out.ReturnFunctionFields = cloneFunctionFieldMap(sig.ReturnFunctionFields)
	out.ReturnEnumPayloadFunctions = cloneFunctionFieldMap(sig.ReturnEnumPayloadFunctions)
	out.ReturnEnumPayloadFields = cloneFunctionFieldMap(sig.ReturnEnumPayloadFields)
	out.ReturnRegionSummary = cloneRegionSummary(sig.ReturnRegionSummary)
	out.ReturnResourceSummary = cloneResourceSummary(sig.ReturnResourceSummary)
	out.ThrowResourceSummary = cloneResourceSummary(sig.ThrowResourceSummary)
	out.Effects = cloneStrings(sig.Effects)
	return out
}

func projectParams(sig FuncSig) []ParamContractV1 {
	out := make([]ParamContractV1, 0, len(sig.ParamTypes))
	for i, typ := range sig.ParamTypes {
		param := ParamContractV1{
			Name:      stringAt(sig.ParamNames, i),
			Type:      typ,
			Ownership: normalizeOwnership(stringAt(sig.ParamOwnership, i)),
		}
		if paramFunctionTypeAt(sig, i) {
			callable := paramCallableType(sig, i)
			param.Callable = &callable
		}
		out = append(out, param)
	}
	return out
}

func projectReturnCallable(sig FuncSig) *CallableValueContractV1 {
	if !sig.ReturnFunctionType &&
		sig.ReturnFunctionSymbol == "" &&
		len(sig.ReturnFunctionCaptures) == 0 &&
		len(sig.ReturnFunctionFields) == 0 &&
		len(sig.ReturnEnumPayloadFunctions) == 0 &&
		len(sig.ReturnEnumPayloadFields) == 0 {
		return nil
	}
	out := CallableValueContractV1{
		CallableTypeContractV1: returnCallableType(sig),
		Symbol:                 sig.ReturnFunctionSymbol,
		ParamName:              sig.ReturnFunctionParamName,
		Captures:               projectCaptures(sig.ReturnFunctionCaptures),
		TouchesMutableGlobals:  sig.ReturnFunctionTouchesMutableGlobals,
		EscapeKind:             string(sig.ReturnFunctionEscapeKind),
		HandleValue:            sig.ReturnFunctionHandleValue,
		Fields:                 projectFunctionFieldMap(sig.ReturnFunctionFields),
		EnumPayloadFunctions:   projectFunctionFieldMap(sig.ReturnEnumPayloadFunctions),
		EnumPayloadFields:      projectFunctionFieldMap(sig.ReturnEnumPayloadFields),
	}
	return &out
}

func projectThrows(sig FuncSig) *ThrowsContractV1 {
	if sig.ThrowsType == "" && len(sig.ThrowResourceSummary) == 0 {
		return nil
	}
	return &ThrowsContractV1{
		Type:            sig.ThrowsType,
		ResourceSummary: cloneSortedResourceSummary(sig.ThrowResourceSummary),
	}
}

func projectRegionSummary(sig FuncSig) map[string]int {
	out := cloneRegionSummary(sig.ReturnRegionSummary)
	if len(out) == 0 && sig.ReturnRegionParam >= 0 {
		out = map[string]int{"": sig.ReturnRegionParam}
	}
	return sortedIntMap(out)
}

func projectResourceSummary(sig FuncSig) map[string][]ResourceProvenance {
	out := cloneResourceSummary(sig.ReturnResourceSummary)
	if len(out) == 0 && sig.ReturnResourceParam >= 0 {
		out = ReturnResourceSummary{
			sig.ReturnResourcePath: {{ParamIndex: sig.ReturnResourceParam, ParamPath: sig.ReturnResourcePath}},
		}
	}
	return cloneSortedResourceSummary(out)
}

func paramCallableType(sig FuncSig, i int) CallableTypeContractV1 {
	return CallableTypeContractV1{
		ParamTypes:      cloneStrings(stringSliceAt(sig.ParamFunctionParams, i)),
		ParamOwnership:  normalizeOwnershipSlice(stringSliceAt(sig.ParamFunctionOwnership, i)),
		ReturnType:      stringAt(sig.ParamFunctionReturns, i),
		ReturnOwnership: normalizeOwnership(stringAt(sig.ParamFunctionReturnOwnership, i)),
		ThrowsType:      stringAt(sig.ParamFunctionThrows, i),
		Effects:         sortedEffectsCopy(stringSliceAt(sig.ParamFunctionEffects, i)),
	}
}

func returnCallableType(sig FuncSig) CallableTypeContractV1 {
	return CallableTypeContractV1{
		ParamTypes:      cloneStrings(sig.ReturnFunctionParams),
		ParamOwnership:  normalizeOwnershipSlice(sig.ReturnFunctionParamOwnership),
		ReturnType:      sig.ReturnFunctionReturn,
		ReturnOwnership: normalizeOwnership(sig.ReturnFunctionReturnOwnership),
		ThrowsType:      sig.ReturnFunctionThrows,
		Effects:         sortedEffectsCopy(sig.ReturnFunctionEffects),
	}
}

func projectFunctionFieldMap(in map[string]FunctionFieldInfo) map[string]FunctionFieldContractV1 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]FunctionFieldContractV1, len(in))
	for key, info := range in {
		out[key] = FunctionFieldContractV1{
			CallableTypeContractV1: CallableTypeContractV1{
				ParamTypes:      cloneStrings(info.FunctionParamTypes),
				ParamOwnership:  normalizeOwnershipSlice(info.FunctionParamOwnership),
				ReturnType:      info.FunctionReturnType,
				ReturnOwnership: normalizeOwnership(info.FunctionReturnOwnership),
				ThrowsType:      info.FunctionThrowsType,
				Effects:         sortedEffectsCopy(info.FunctionEffects),
			},
			Symbol:                info.FunctionValue,
			ParamName:             info.FunctionParamName,
			Captures:              projectCaptures(info.FunctionCaptures),
			EscapeCaptures:        projectCaptures(info.FunctionEscapeCaptures),
			TouchesMutableGlobals: info.FunctionTouchesMutableGlobals,
			ReturnSnapshotAlias:   info.FunctionReturnSnapshotAlias,
			DirectSnapshotAlias:   info.FunctionDirectSnapshotAlias,
			EscapeKind:            string(info.FunctionEscapeKind),
			HandleValue:           info.FunctionHandleValue,
		}
	}
	return out
}

func projectCaptures(in []frontend.ClosureCapture) []ClosureCaptureContractV1 {
	if len(in) == 0 {
		return nil
	}
	out := make([]ClosureCaptureContractV1, 0, len(in))
	for _, capture := range in {
		out = append(out, ClosureCaptureContractV1{
			Name:    capture.Name,
			Type:    captureTypeName(capture.Type),
			Mutable: capture.Mutable,
		})
	}
	return out
}

func functionContractDigest(contract FunctionContractV1) (string, error) {
	payload := struct {
		Schema                string
		Name                  string
		Generic               bool
		Public                bool
		Params                []ParamContractV1
		Result                ResultContractV1
		Throws                *ThrowsContractV1
		Async                 bool
		Effects               []string
		Policy                FunctionPolicyContractV1
		TouchesMutableGlobals bool
	}{
		Schema:                contract.Schema,
		Name:                  contract.Name,
		Generic:               contract.Generic,
		Public:                contract.Public,
		Params:                contract.Params,
		Result:                contract.Result,
		Throws:                contract.Throws,
		Async:                 contract.Async,
		Effects:               contract.Effects,
		Policy:                contract.Policy,
		TouchesMutableGlobals: contract.TouchesMutableGlobals,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validateOptionalParamLen(name string, got, want int) error {
	if got == 0 || got == want {
		return nil
	}
	return fmt.Errorf("%s length = %d, want 0 or %d", name, got, want)
}

func validateOwnership(name string, index int, ownership string) error {
	if err := validateOwnershipValue(name, ownership); err != nil {
		return fmt.Errorf("%s[%d]: %w", name, index, err)
	}
	return nil
}

func validateOwnershipValue(name, ownership string) error {
	switch ownership {
	case "", "owned", "borrow", "inout", "consume":
		return nil
	default:
		return fmt.Errorf("%s has unknown ownership marker %q", name, ownership)
	}
}

func normalizeOwnership(ownership string) string {
	if ownership == "" {
		return "owned"
	}
	return ownership
}

func normalizeOwnershipSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	for i, ownership := range in {
		out[i] = normalizeOwnership(ownership)
	}
	return out
}

func validateEffects(name string, effects []string) error {
	seen := map[string]struct{}{}
	for _, effect := range effects {
		if _, ok := canonicalContractEffects[effect]; !ok {
			return fmt.Errorf("%s contains non-canonical effect %q", name, effect)
		}
		if _, ok := seen[effect]; ok {
			return fmt.Errorf("%s contains duplicate effect %q", name, effect)
		}
		seen[effect] = struct{}{}
	}
	return nil
}

func sortedEffectsCopy(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := cloneStrings(in)
	sort.Strings(out)
	return out
}

func validateSummaryParam(name string, param, paramCount int) error {
	if param == SummaryParamNone || param == SummaryParamUnknown {
		return nil
	}
	if param >= 0 && param < paramCount {
		return nil
	}
	return fmt.Errorf("%s has invalid parameter index %d", name, param)
}

func validateRegionSummary(name string, summary ReturnRegionSummary, paramCount int) error {
	for path, param := range summary {
		if err := validateSummaryPath(name, path); err != nil {
			return err
		}
		if param < 0 || param >= paramCount {
			return fmt.Errorf("%s[%q] has invalid parameter index %d", name, path, param)
		}
	}
	return nil
}

func validateResourceSummary(name string, summary ReturnResourceSummary, paramCount int) error {
	for path, provenances := range summary {
		if err := validateSummaryPath(name, path); err != nil {
			return err
		}
		seen := map[ResourceProvenance]struct{}{}
		for _, provenance := range provenances {
			if provenance.ParamIndex < 0 || provenance.ParamIndex >= paramCount {
				return fmt.Errorf("%s[%q] has invalid parameter index %d", name, path, provenance.ParamIndex)
			}
			if err := validateSummaryPath(name, provenance.ParamPath); err != nil {
				return err
			}
			if _, ok := seen[provenance]; ok {
				return fmt.Errorf("%s[%q] contains duplicate provenance %#v", name, path, provenance)
			}
			seen[provenance] = struct{}{}
		}
	}
	return nil
}

func validateSummaryPath(name, path string) error {
	if path == "" {
		return nil
	}
	if strings.HasPrefix(path, ".") ||
		strings.HasSuffix(path, ".") ||
		strings.Contains(path, "..") ||
		strings.Contains(path, " ") {
		return fmt.Errorf("%s has non-canonical path %q", name, path)
	}
	return nil
}

func validateCallableType(name string, callable CallableTypeContractV1) error {
	paramCount := len(callable.ParamTypes)
	if len(callable.ParamOwnership) != 0 && len(callable.ParamOwnership) != paramCount {
		return fmt.Errorf("%s ParamOwnership length = %d, want 0 or %d", name, len(callable.ParamOwnership), paramCount)
	}
	for i, ownership := range callable.ParamOwnership {
		if err := validateOwnership(name+".ParamOwnership", i, ownership); err != nil {
			return err
		}
	}
	if err := validateOwnershipValue(name+".ReturnOwnership", callable.ReturnOwnership); err != nil {
		return err
	}
	if err := validateEffects(name+".Effects", callable.Effects); err != nil {
		return err
	}
	return nil
}

func validateCallableEscape(name string, kind CallableEscapeKind, handle bool) error {
	if kind == "" && !handle {
		return nil
	}
	switch kind {
	case CallableEscapeLocalSnapshot, CallableEscapeHeap, CallableEscapeGlobal, CallableEscapeThread:
		return nil
	default:
		return fmt.Errorf("%s has unsupported callable escape kind %q", name, kind)
	}
}

func validateFunctionFieldMap(name string, fields map[string]FunctionFieldInfo) error {
	for key, field := range fields {
		if key == "" || strings.Contains(key, "..") || strings.Contains(key, " ") {
			return fmt.Errorf("%s has non-canonical key %q", name, key)
		}
		callable := CallableTypeContractV1{
			ParamTypes:      field.FunctionParamTypes,
			ParamOwnership:  field.FunctionParamOwnership,
			ReturnType:      field.FunctionReturnType,
			ReturnOwnership: field.FunctionReturnOwnership,
			ThrowsType:      field.FunctionThrowsType,
			Effects:         field.FunctionEffects,
		}
		if err := validateCallableType(name+"["+key+"]", callable); err != nil {
			return err
		}
		if field.FunctionHandleValue {
			if err := validateCallableEscape(name+"["+key+"]", field.FunctionEscapeKind, true); err != nil {
				return err
			}
		} else if field.FunctionEscapeKind != "" {
			if err := validateCallableEscape(name+"["+key+"]", field.FunctionEscapeKind, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func paramFunctionTypeAt(sig FuncSig, i int) bool {
	return boolAt(sig.ParamFunctionTypes, i)
}

func boolAt(in []bool, i int) bool {
	if i < 0 || i >= len(in) {
		return false
	}
	return in[i]
}

func stringAt(in []string, i int) string {
	if i < 0 || i >= len(in) {
		return ""
	}
	return in[i]
}

func stringSliceAt(in [][]string, i int) []string {
	if i < 0 || i >= len(in) {
		return nil
	}
	return in[i]
}

func captureTypeName(ref frontend.TypeRef) string {
	if ref.Name != "" {
		return ref.Name
	}
	if ref.Kind == frontend.TypeRefFunction {
		return "fn"
	}
	return ""
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	return append([]string(nil), in...)
}

func cloneBools(in []bool) []bool {
	if len(in) == 0 {
		return nil
	}
	return append([]bool(nil), in...)
}

func cloneStringSlices(in [][]string) [][]string {
	if len(in) == 0 {
		return nil
	}
	out := make([][]string, len(in))
	for i := range in {
		out[i] = cloneStrings(in[i])
	}
	return out
}

func cloneClosureCaptures(in []frontend.ClosureCapture) []frontend.ClosureCapture {
	if len(in) == 0 {
		return nil
	}
	return append([]frontend.ClosureCapture(nil), in...)
}

func cloneFunctionFieldMap(in map[string]FunctionFieldInfo) map[string]FunctionFieldInfo {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]FunctionFieldInfo, len(in))
	for key, info := range in {
		out[key] = cloneFunctionFieldInfo(info)
	}
	return out
}

func cloneFunctionFieldInfo(info FunctionFieldInfo) FunctionFieldInfo {
	out := info
	out.FunctionCaptures = cloneClosureCaptures(info.FunctionCaptures)
	out.FunctionEscapeCaptures = cloneClosureCaptures(info.FunctionEscapeCaptures)
	out.FunctionParamTypes = cloneStrings(info.FunctionParamTypes)
	out.FunctionParamOwnership = cloneStrings(info.FunctionParamOwnership)
	out.FunctionEffects = cloneStrings(info.FunctionEffects)
	return out
}

func cloneRegionSummary(in ReturnRegionSummary) ReturnRegionSummary {
	if len(in) == 0 {
		return nil
	}
	out := make(ReturnRegionSummary, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneResourceSummary(in ReturnResourceSummary) ReturnResourceSummary {
	if len(in) == 0 {
		return nil
	}
	out := make(ReturnResourceSummary, len(in))
	for key, provenances := range in {
		out[key] = append([]ResourceProvenance(nil), provenances...)
	}
	return out
}

func sortedIntMap(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out[key] = in[key]
	}
	return out
}

func cloneSortedResourceSummary(in ReturnResourceSummary) map[string][]ResourceProvenance {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]ResourceProvenance, len(in))
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		provenances := append([]ResourceProvenance(nil), in[key]...)
		sort.Slice(provenances, func(i, j int) bool {
			if provenances[i].ParamIndex != provenances[j].ParamIndex {
				return provenances[i].ParamIndex < provenances[j].ParamIndex
			}
			return provenances[i].ParamPath < provenances[j].ParamPath
		})
		out[key] = provenances
	}
	return out
}

var canonicalContractEffects = map[string]struct{}{
	"actors":     {},
	"alloc":      {},
	"budget":     {},
	"capability": {},
	"control":    {},
	"io":         {},
	"islands":    {},
	"link":       {},
	"mem":        {},
	"mmio":       {},
	"privacy":    {},
	"runtime":    {},
	"surface":    {},
}
