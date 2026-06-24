package model

import (
	"reflect"
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestFunctionContractProjectDigestDeterministic(t *testing.T) {
	sig := sampleFunctionContractSig()
	var firstDigest string
	for i := 0; i < 100; i++ {
		contract, err := ProjectFunctionContractV1("pkg.make", sig)
		if err != nil {
			t.Fatalf("ProjectFunctionContractV1 iteration %d returned error: %v", i, err)
		}
		if contract.Schema != FunctionContractSchemaV1 {
			t.Fatalf("Schema = %q, want %q", contract.Schema, FunctionContractSchemaV1)
		}
		if !strings.HasPrefix(contract.Digest, "sha256:") || len(contract.Digest) != len("sha256:")+64 {
			t.Fatalf("Digest = %q, want sha256:<64 hex>", contract.Digest)
		}
		if contract.Result.Ownership != "owned" {
			t.Fatalf("Result.Ownership = %q, want owned", contract.Result.Ownership)
		}
		if !reflect.DeepEqual(contract.Effects, []string{"alloc", "io"}) {
			t.Fatalf("Effects = %#v, want sorted alloc/io", contract.Effects)
		}
		if got := contract.Result.ResourceSummary[""][0]; got != (ResourceProvenance{ParamIndex: 0, ParamPath: ""}) {
			t.Fatalf("first sorted resource provenance = %#v", got)
		}
		if contract.Params[1].Callable == nil || !reflect.DeepEqual(contract.Params[1].Callable.Effects, []string{"mem", "privacy"}) {
			t.Fatalf("param callable contract = %#v", contract.Params[1].Callable)
		}
		if contract.Result.Callable == nil || !contract.Result.Callable.HandleValue ||
			contract.Result.Callable.EscapeKind != string(CallableEscapeHeap) ||
			contract.Result.Callable.Symbol != "pkg.closure" {
			t.Fatalf("result callable contract = %#v", contract.Result.Callable)
		}
		if contract.Result.Callable.Fields["field"].ReturnSnapshotAlias != true {
			t.Fatalf("callable field contract did not preserve return snapshot alias: %#v", contract.Result.Callable.Fields["field"])
		}
		if contract.Throws == nil || contract.Throws.ResourceSummary["err"][0] != (ResourceProvenance{ParamIndex: 0, ParamPath: "err"}) {
			t.Fatalf("throws/resource contract = %#v", contract.Throws)
		}
		if !contract.Policy.NoAlloc || !contract.Policy.NoBlock || !contract.Policy.Realtime ||
			!contract.Policy.HasBudget || contract.Policy.Budget != 10 {
			t.Fatalf("policy contract = %#v", contract.Policy)
		}
		digest, err := FunctionContractDigest("pkg.make", sig)
		if err != nil {
			t.Fatalf("FunctionContractDigest returned error: %v", err)
		}
		if digest != contract.Digest {
			t.Fatalf("FunctionContractDigest = %q, projected digest = %q", digest, contract.Digest)
		}
		if firstDigest == "" {
			firstDigest = contract.Digest
		} else if contract.Digest != firstDigest {
			t.Fatalf("digest iteration %d = %q, first %q", i, contract.Digest, firstDigest)
		}
	}

	reordered := sampleFunctionContractSig()
	reordered.Effects = []string{"io", "alloc"}
	reordered.ReturnRegionSummary = ReturnRegionSummary{"tail": 1, "": 0}
	reordered.ReturnResourceSummary = ReturnResourceSummary{
		"leaf": {{ParamIndex: 1, ParamPath: "tail"}},
		"": {
			{ParamIndex: 1, ParamPath: "tail"},
			{ParamIndex: 0, ParamPath: ""},
		},
	}
	reorderedDigest, err := FunctionContractDigest("pkg.make", reordered)
	if err != nil {
		t.Fatalf("FunctionContractDigest reordered returned error: %v", err)
	}
	if reorderedDigest != firstDigest {
		t.Fatalf("digest changed for map/provenance order: %q vs %q", reorderedDigest, firstDigest)
	}
}

func TestCloneFuncSigDeepCopiesContractState(t *testing.T) {
	original := sampleFunctionContractSig()
	clone := CloneFuncSig(original)

	clone.ParamNames[0] = "mutated"
	clone.ParamFunctionParams[1][0] = "mutated"
	clone.ParamFunctionEffects[1][0] = "mutated"
	clone.ReturnFunctionCaptures[0].Name = "mutated"
	clone.ReturnFunctionFields["field"] = FunctionFieldInfo{
		FunctionParamTypes: []string{"mutated"},
	}
	clone.ReturnEnumPayloadFunctions["0:0"] = FunctionFieldInfo{
		FunctionParamTypes: []string{"mutated"},
	}
	clone.ReturnRegionSummary[""] = 9
	clone.ReturnResourceSummary[""][0].ParamIndex = 9
	clone.ThrowResourceSummary["err"][0].ParamPath = "mutated"
	clone.Effects[0] = "mutated"

	if original.ParamNames[0] != "value" {
		t.Fatalf("ParamNames alias changed original: %#v", original.ParamNames)
	}
	if original.ParamFunctionParams[1][0] != "i32" {
		t.Fatalf("ParamFunctionParams alias changed original: %#v", original.ParamFunctionParams)
	}
	if original.ParamFunctionEffects[1][0] != "privacy" {
		t.Fatalf("ParamFunctionEffects alias changed original: %#v", original.ParamFunctionEffects)
	}
	if original.ReturnFunctionCaptures[0].Name != "cap" {
		t.Fatalf("ReturnFunctionCaptures alias changed original: %#v", original.ReturnFunctionCaptures)
	}
	if original.ReturnFunctionFields["field"].FunctionParamTypes[0] != "i32" {
		t.Fatalf("ReturnFunctionFields alias changed original: %#v", original.ReturnFunctionFields)
	}
	if original.ReturnEnumPayloadFunctions["0:0"].FunctionParamTypes[0] != "i32" {
		t.Fatalf("ReturnEnumPayloadFunctions alias changed original: %#v", original.ReturnEnumPayloadFunctions)
	}
	if original.ReturnRegionSummary[""] != 0 {
		t.Fatalf("ReturnRegionSummary alias changed original: %#v", original.ReturnRegionSummary)
	}
	if original.ReturnResourceSummary[""][0].ParamIndex != 1 {
		t.Fatalf("ReturnResourceSummary alias changed original: %#v", original.ReturnResourceSummary)
	}
	if original.ThrowResourceSummary["err"][0].ParamPath != "err" {
		t.Fatalf("ThrowResourceSummary alias changed original: %#v", original.ThrowResourceSummary)
	}
	if original.Effects[0] != "io" {
		t.Fatalf("Effects alias changed original: %#v", original.Effects)
	}
}

func TestValidateFuncSigContractRejectsInvalidInvariants(t *testing.T) {
	tests := []struct {
		name string
		edit func(*FuncSig)
	}{
		{
			name: "inconsistent param array lengths",
			edit: func(sig *FuncSig) {
				sig.ParamNames = sig.ParamNames[:1]
			},
		},
		{
			name: "invalid ownership",
			edit: func(sig *FuncSig) {
				sig.ParamOwnership[0] = "lease"
			},
		},
		{
			name: "duplicate effect",
			edit: func(sig *FuncSig) {
				sig.Effects = []string{"io", "io"}
			},
		},
		{
			name: "invalid region param index",
			edit: func(sig *FuncSig) {
				sig.ReturnRegionSummary[""] = 7
			},
		},
		{
			name: "invalid resource param index",
			edit: func(sig *FuncSig) {
				sig.ReturnResourceSummary[""] = []ResourceProvenance{{ParamIndex: SummaryParamUnknown, ParamPath: ""}}
			},
		},
		{
			name: "duplicate resource provenance",
			edit: func(sig *FuncSig) {
				sig.ReturnResourceSummary[""] = []ResourceProvenance{
					{ParamIndex: 0, ParamPath: ""},
					{ParamIndex: 0, ParamPath: ""},
				}
			},
		},
		{
			name: "unsupported callable escape kind",
			edit: func(sig *FuncSig) {
				sig.ReturnFunctionEscapeKind = CallableEscapeKind("stack")
			},
		},
		{
			name: "handle return wrong slot count",
			edit: func(sig *FuncSig) {
				sig.ReturnSlots = FnPtrSlotCount
			},
		},
		{
			name: "throws summary without throws type",
			edit: func(sig *FuncSig) {
				sig.ThrowsType = ""
			},
		},
		{
			name: "negative budget",
			edit: func(sig *FuncSig) {
				sig.Budget = -1
			},
		},
		{
			name: "realtime without noalloc/noblock",
			edit: func(sig *FuncSig) {
				sig.HasNoAlloc = false
			},
		},
		{
			name: "public generic export",
			edit: func(sig *FuncSig) {
				sig.Generic = true
			},
		},
		{
			name: "noncanonical summary path",
			edit: func(sig *FuncSig) {
				sig.ReturnRegionSummary["bad..path"] = 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := sampleFunctionContractSig()
			tt.edit(&sig)
			if err := ValidateFuncSigContract("pkg.make", sig, nil); err == nil {
				t.Fatalf("ValidateFuncSigContract accepted invalid signature")
			}
		})
	}
}

func sampleFunctionContractSig() FuncSig {
	field := FunctionFieldInfo{
		FunctionValue:                 "pkg.field",
		FunctionParamName:             "fieldParam",
		FunctionCaptures:              []frontend.ClosureCapture{{Name: "fieldCap", Type: frontend.TypeRef{Name: "i32"}}},
		FunctionEscapeCaptures:        []frontend.ClosureCapture{{Name: "fieldEsc", Type: frontend.TypeRef{Name: "i32"}}},
		FunctionTouchesMutableGlobals: true,
		FunctionReturnSnapshotAlias:   true,
		FunctionDirectSnapshotAlias:   true,
		FunctionEscapeKind:            CallableEscapeGlobal,
		FunctionHandleValue:           true,
		FunctionParamTypes:            []string{"i32"},
		FunctionParamOwnership:        []string{""},
		FunctionReturnType:            "i32",
		FunctionReturnOwnership:       "",
		FunctionThrowsType:            "err",
		FunctionEffects:               []string{"runtime"},
	}
	return FuncSig{
		Public:                              true,
		HasNoAlloc:                          true,
		HasNoBlock:                          true,
		HasRealtime:                         true,
		HasBudget:                           true,
		Budget:                              10,
		ParamNames:                          []string{"value", "callback"},
		ParamTypes:                          []string{"i32", "fn"},
		ParamFunctionTypes:                  []bool{false, true},
		ParamFunctionParams:                 [][]string{nil, {"i32"}},
		ParamFunctionOwnership:              [][]string{nil, {""}},
		ParamFunctionReturns:                []string{"", "i32"},
		ParamFunctionReturnOwnership:        []string{"", ""},
		ParamFunctionThrows:                 []string{"", "err"},
		ParamFunctionEffects:                [][]string{nil, {"privacy", "mem"}},
		ParamOwnership:                      []string{"", "borrow"},
		ParamSlots:                          2,
		ReturnType:                          "fn",
		ReturnFunctionType:                  true,
		ReturnFunctionParams:                []string{"i32"},
		ReturnFunctionParamOwnership:        []string{""},
		ReturnFunctionReturn:                "i32",
		ReturnFunctionReturnOwnership:       "",
		ReturnFunctionThrows:                "err",
		ReturnFunctionEffects:               []string{"io"},
		ReturnFunctionSymbol:                "pkg.closure",
		ReturnFunctionParamName:             "cb",
		ReturnFunctionCaptures:              []frontend.ClosureCapture{{Name: "cap", Type: frontend.TypeRef{Name: "i32"}}},
		ReturnFunctionTouchesMutableGlobals: true,
		ReturnFunctionEscapeKind:            CallableEscapeHeap,
		ReturnFunctionHandleValue:           true,
		ReturnFunctionFields:                map[string]FunctionFieldInfo{"field": field},
		ReturnEnumPayloadFunctions:          map[string]FunctionFieldInfo{"0:0": field},
		ReturnEnumPayloadFields:             map[string]FunctionFieldInfo{"1:0": field},
		ThrowsType:                          "err",
		Async:                               true,
		ReturnSlots:                         CallableHandleSlotCount,
		ReturnRegionParam:                   SummaryParamUnknown,
		ReturnRegionSummary:                 ReturnRegionSummary{"": 0, "tail": 1},
		ReturnResourceParam:                 SummaryParamUnknown,
		ReturnResourceSummary: ReturnResourceSummary{
			"": {
				{ParamIndex: 1, ParamPath: "tail"},
				{ParamIndex: 0, ParamPath: ""},
			},
			"leaf": {{ParamIndex: 1, ParamPath: "tail"}},
		},
		ThrowResourceSummary: ReturnResourceSummary{
			"err": {{ParamIndex: 0, ParamPath: "err"}},
		},
		Effects:               []string{"io", "alloc"},
		TouchesMutableGlobals: true,
	}
}
