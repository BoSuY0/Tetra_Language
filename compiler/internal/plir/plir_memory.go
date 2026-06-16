package plir

import (
	"fmt"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/runtimeabi"
	"tetra_language/compiler/internal/semantics"
)

func (b *builder) recordAllocBytesCall(name string, call *frontend.CallExpr, targetName string) {
	if targetName == "" {
		targetName = b.syntheticTargetName("raw_alloc", call)
	}
	b.clearRawPointerMetadata(targetName)
	lengthArg := allocationLengthArg(name, call)
	lengthExpr := exprPath(lengthArg)
	if lengthExpr == "" {
		lengthExpr = "expr"
	}
	lengthConst, lengthConstKnown := b.constIntValue(lengthArg)
	rawBaseBytes := int64(0)
	if lengthConstKnown && lengthConst > 0 {
		rawBaseBytes = lengthConst
	}
	id := valueID(ValueAllocIntent, targetName)
	value := Value{
		ID:     id,
		Kind:   ValueAllocIntent,
		Type:   "ptr",
		Source: sourceString(call.At),
		Region: "raw_allocation:" + targetName,
		Alloc: &AllocIntent{
			ElementType:            "raw_bytes",
			ElementSize:            1,
			LengthExpr:             lengthExpr,
			LengthConstKnown:       lengthConstKnown,
			LengthConst:            lengthConst,
			ZeroGuardStatus:        "invalid_precondition",
			NegativeGuardStatus:    "reject_before_allocation",
			OverflowGuardStatus:    "reject_before_allocation",
			Builtin:                name,
			Source:                 sourceString(call.At),
			RawPointerBoundsStatus: string(runtimeabi.RawPointerBoundsAllocationBase),
			RawPointerBaseID:       targetName,
			RawPointerBaseBytes:    rawBaseBytes,
			RawPointerOffsetBytes:  0,
			RawSlicePolicy:         string(runtimeabi.RawSliceBoundsExternalUnknown),
		},
		Provenance:  Provenance{Kind: ProvenanceAllocation, Root: targetName},
		UnsafeClass: UnsafeVerifiedRoot,
		Lifetime:    Lifetime{Birth: sourceString(call.At), Owner: targetName},
		Mutable:     true,
		Escape:      EscapeConservative,
	}
	b.addValue(value)
	b.addOperation(Operation{
		Kind:        OpAllocIntent,
		Source:      sourceString(call.At),
		Inputs:      callInputs(call.Args),
		Outputs:     []string{id},
		UnsafeClass: UnsafeVerifiedRoot,
		Note:        "alloc_bytes raw allocation-base metadata: zero invalid, negative and overflow reject before allocation",
	})
	b.rawPointerRoots[targetName] = targetName
	b.rawPointerOffsets[targetName] = 0
	if rawBaseBytes > 0 {
		b.rawPointerBytes[targetName] = rawBaseBytes
	}
}

func (b *builder) recordRawPtrAddCall(name string, call *frontend.CallExpr, targetName string) {
	if targetName != "" {
		b.clearRawPointerMetadata(targetName)
	}
	inputs := callInputs(call.Args)
	outputs := []string(nil)
	if targetName != "" {
		outputs = []string{targetName}
	}
	base := callArgPath(call, 0)
	offset := callArgPath(call, 1)
	status := runtimeabi.RawPointerBoundsCheckedExternalUnknown
	unsafeClass := UnsafeUnknown
	note := fmt.Sprintf("%s raw_pointer_bounds: %s base:%s offset:%s", name, status, base, offset)
	if baseRoot := b.rawPointerBaseRoot(base); baseRoot != "" {
		status = runtimeabi.RawPointerBoundsDerivedOffset
		unsafeClass = UnsafeChecked
		validDerived := true
		offsetBytes, offsetKnown := evalConstInt64(callArg(call, 1))
		if !offsetKnown {
			status = runtimeabi.RawPointerBoundsCheckedExternalUnknown
			unsafeClass = UnsafeUnknown
			validDerived = false
			note = fmt.Sprintf("%s raw_pointer_bounds: %s base:%s offset:%s", name, status, baseRoot, offset)
		}
		if offsetKnown && offsetBytes < 0 {
			status = runtimeabi.RawPointerBoundsRejectedNegativeOffset
			validDerived = false
			note = fmt.Sprintf("%s raw_pointer_bounds: %s base:%s offset:%d width:%d", name, status, baseRoot, offsetBytes, int64(1))
		}
		baseOffset := int64(0)
		if prior, ok := b.rawPointerOffsetBytes(base); ok {
			baseOffset = prior
		}
		totalOffset := offsetBytes
		offsetSumOK := true
		if offsetKnown && offsetBytes >= 0 {
			totalOffset, offsetSumOK = checkedAddInt64(baseOffset, offsetBytes)
			if !offsetSumOK {
				status = runtimeabi.RawPointerBoundsRejectedAccessWidthOverflow
				validDerived = false
				note = fmt.Sprintf("%s raw_pointer_bounds: %s base:%s offset:%s", name, status, baseRoot, offset)
			}
		}
		if offsetSumOK && offsetKnown && offsetBytes >= 0 {
			if baseBytes, bytesKnown := b.rawPointerBaseByteSize(baseRoot); bytesKnown && offsetKnown {
				root, err := runtimeabi.NewRawAllocationBounds(baseRoot, baseBytes)
				if err == nil {
					derived, diag := runtimeabi.DeriveRawPointerBounds(root, totalOffset, 1)
					status = derived.Status
					validDerived = diag == nil
					note = fmt.Sprintf("%s raw_pointer_bounds: %s base:%s offset:%d width:%d", name, status, baseRoot, totalOffset, derived.AccessWidthBytes)
				}
			} else {
				note = fmt.Sprintf("%s raw_pointer_bounds: %s base:%s offset:%s", name, status, baseRoot, offset)
			}
		}
		if targetName != "" && validDerived {
			b.rawPointerRoots[targetName] = baseRoot
			if offsetKnown {
				b.rawPointerOffsets[targetName] = totalOffset
			}
		}
	}
	b.addOperation(Operation{Kind: OpUnsafe, Source: sourceString(call.At), Inputs: inputs, Outputs: outputs, UnsafeClass: unsafeClass, Note: note})
}

func (b *builder) recordRawMemoryAccessCall(name string, call *frontend.CallExpr, targetName string) {
	if targetName != "" {
		b.clearRawPointerMetadata(targetName)
	}
	inputs := callInputs(call.Args)
	outputs := []string(nil)
	if targetName != "" {
		outputs = []string{targetName}
	}
	ptr := callArgPath(call, 0)
	unsafeClass := b.rawPointerUnsafeClass(ptr)
	status := runtimeabi.RawPointerBoundsCheckedExternalUnknown
	if unsafeClass == UnsafeChecked {
		status = runtimeabi.RawPointerBoundsDerivedOffset
		if root := b.rawPointerBaseRoot(ptr); root != "" {
			if baseBytes, bytesKnown := b.rawPointerBaseByteSize(root); bytesKnown {
				if offsetBytes, offsetKnown := b.rawPointerOffsetBytes(ptr); offsetKnown {
					rootBounds, err := runtimeabi.NewRawAllocationBounds(root, baseBytes)
					if err == nil {
						derived, _ := runtimeabi.DeriveRawPointerBounds(rootBounds, offsetBytes, rawMemoryAccessWidthBytes(name))
						status = derived.Status
					}
				}
			}
		}
	}
	note := fmt.Sprintf("%s raw memory gateway: %s pointer:%s width:%d", name, status, ptr, rawMemoryAccessWidthBytes(name))
	if offsetBytes, offsetKnown := b.rawPointerOffsetBytes(ptr); offsetKnown {
		note = fmt.Sprintf("%s raw memory gateway: %s pointer:%s offset:%d width:%d", name, status, ptr, offsetBytes, rawMemoryAccessWidthBytes(name))
	}
	b.addOperation(Operation{Kind: OpUnsafe, Source: sourceString(call.At), Inputs: inputs, Outputs: outputs, UnsafeClass: unsafeClass, Note: note})
}

func (b *builder) rawPointerUnsafeClass(path string) UnsafeClass {
	if b.rawPointerBaseRoot(path) != "" {
		return UnsafeChecked
	}
	return UnsafeUnknown
}

func (b *builder) rawPointerBaseRoot(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "?" || path == "expr" {
		return ""
	}
	if root, ok := b.rawPointerRoots[path]; ok {
		return root
	}
	if dot := strings.Index(path, "."); dot > 0 {
		if root, ok := b.rawPointerRoots[path[:dot]]; ok {
			return root
		}
	}
	return ""
}

func (b *builder) rawPointerBaseByteSize(root string) (int64, bool) {
	root = strings.TrimSpace(root)
	if root == "" {
		return 0, false
	}
	bytes, ok := b.rawPointerBytes[root]
	return bytes, ok && bytes > 0
}

func (b *builder) rawPointerOffsetBytes(path string) (int64, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return 0, false
	}
	if offset, ok := b.rawPointerOffsets[path]; ok {
		return offset, true
	}
	if dot := strings.Index(path, "."); dot > 0 {
		if offset, ok := b.rawPointerOffsets[path[:dot]]; ok {
			return offset, true
		}
	}
	if root, ok := b.rawPointerRoots[path]; ok && root == path {
		return 0, true
	}
	return 0, false
}

func (b *builder) clearRawPointerMetadata(name string) {
	if name == "" {
		return
	}
	delete(b.rawPointerRoots, name)
	delete(b.rawPointerBytes, name)
	delete(b.rawPointerOffsets, name)
}

func (b *builder) recordMakeSliceCall(name string, elem string, call *frontend.CallExpr, targetName string) {
	if targetName == "" {
		targetName = b.syntheticTargetName("alloc", call)
	}
	id := valueID(ValueAllocIntent, targetName)
	prov := Provenance{Kind: ProvenanceAllocation, Root: targetName}
	region := "allocation:" + targetName
	zeroGuardStatus := "valid_empty_no_allocator"
	negativeGuardStatus := "reject_before_allocation"
	overflowGuardStatus := "reject_before_allocation"
	inputs := []string(nil)
	islandFactToken := islandTokenState{}
	if strings.HasPrefix(name, "core.island_make_") {
		islandRoot := callArgPath(call, 0)
		if islandRoot == "?" || islandRoot == "" {
			islandRoot = "island"
		}
		islandFactToken = b.islandTokenForPath(islandRoot)
		prov.Kind = ProvenanceIsland
		prov.Root = islandTokenRoot(islandFactToken.IslandID)
		region = islandFactToken.IslandID
		zeroGuardStatus = "valid_empty_no_metadata_access"
		negativeGuardStatus = "reject_before_metadata_access"
		overflowGuardStatus = "reject_before_metadata_access"
		inputs = callInputs(call.Args)
	}
	lengthArg := allocationLengthArg(name, call)
	lengthExpr := exprPath(lengthArg)
	if lengthExpr == "" {
		lengthExpr = "expr"
	}
	lengthConst, lengthConstKnown := b.constIntValue(lengthArg)
	value := Value{
		ID:     id,
		Kind:   ValueAllocIntent,
		Type:   "[]" + elem,
		Source: sourceString(call.At),
		Region: region,
		Alloc: &AllocIntent{
			ElementType:         elem,
			ElementSize:         elementSize(elem),
			LengthExpr:          lengthExpr,
			LengthConstKnown:    lengthConstKnown,
			LengthConst:         lengthConst,
			ZeroGuardStatus:     zeroGuardStatus,
			NegativeGuardStatus: negativeGuardStatus,
			OverflowGuardStatus: overflowGuardStatus,
			Builtin:             name,
			Source:              sourceString(call.At),
		},
		Provenance: prov,
		Lifetime:   Lifetime{Birth: sourceString(call.At), Owner: targetName},
		Mutable:    true,
		Escape:     EscapeConservative,
	}
	b.addValue(value)
	note := "make<" + elem + "> length contract: zero valid, negative and overflow reject before allocation"
	if prov.Kind == ProvenanceIsland {
		note = "island_make<" + elem + "> length contract: zero valid, negative and overflow reject before island metadata access"
	}
	b.addOperation(Operation{Kind: OpAllocIntent, Source: sourceString(call.At), Inputs: inputs, Outputs: []string{id}, Note: note})
	if prov.Kind == ProvenanceIsland {
		b.addFact(b.islandAllocationFact(FactProvenanceKnown, id, islandFactToken, "", "compiler-known allocation intent"))
		b.addFact(b.islandAllocationFact(FactLenStable, id, islandFactToken, "", "slice metadata is opaque in safe code"))
		b.addFact(b.islandAllocationFact(FactRegionAlive, id, islandFactToken, value.Region, ""))
		b.addFact(b.islandAllocationFact(FactAligned, id, islandFactToken, value.Region, "island region allocator returns 16-byte aligned payloads"))
		return
	}
	b.addFact(Fact{Kind: FactProvenanceKnown, ValueID: id, Reason: "compiler-known allocation intent"})
	b.addFact(Fact{Kind: FactLenStable, ValueID: id, Reason: "slice metadata is opaque in safe code"})
	b.addFact(Fact{Kind: FactRegionAlive, ValueID: id, Region: value.Region})
}

func (b *builder) islandAllocationFact(kind FactKind, valueID string, token islandTokenState, region string, reason string) Fact {
	return Fact{
		Kind:     kind,
		ValueID:  valueID,
		IslandID: token.IslandID,
		Epoch:    token.Epoch,
		Region:   region,
		Reason:   reason,
	}
}

func (b *builder) recordRawSliceCall(name string, elem string, call *frontend.CallExpr, targetName string) {
	if targetName == "" {
		targetName = b.syntheticTargetName("raw_view", call)
	}
	b.recordRawPointerExposure(call)
	id := valueID(ValueView, targetName)
	provenance := Provenance{Kind: ProvenanceExternal, Root: "raw_parts"}
	unsafeClass := UnsafeUnknown
	status := runtimeabi.RawSliceBoundsExternalUnknown
	note := name + " creates a conservative external-provenance view"
	ptr := callArgPath(call, 0)
	length := callArgPath(call, 1)
	if root := b.rawPointerBaseRoot(ptr); root != "" {
		if baseBytes, bytesKnown := b.rawPointerBaseByteSize(root); bytesKnown {
			if lengthConst, lengthKnown := evalConstInt64(callArg(call, 1)); lengthKnown {
				offsetBytes := int64(0)
				if offset, offsetKnown := b.rawPointerOffsetBytes(ptr); offsetKnown {
					offsetBytes = offset
				}
				ptrStatus := runtimeabi.RawPointerBoundsAllocationBase
				if offsetBytes != 0 {
					ptrStatus = runtimeabi.RawPointerBoundsDerivedOffset
				}
				sliceBounds := runtimeabi.RawSliceBoundsFromParts(runtimeabi.RawPointerBoundsMetadata{
					Status:                 ptrStatus,
					BaseID:                 root,
					BaseBytes:              baseBytes,
					OffsetBytes:            offsetBytes,
					VerifiedAllocationRoot: true,
				}, lengthConst, int64(elementSize(elem)))
				status = sliceBounds.Status
				switch status {
				case runtimeabi.RawSliceBoundsVerifiedAllocationRoot, runtimeabi.RawSliceBoundsRejectedNegativeLength, runtimeabi.RawSliceBoundsRejectedLengthOverflow:
					unsafeClass = UnsafeChecked
					provenance.Root = "raw_parts:" + root
					note = fmt.Sprintf("%s raw_slice_bounds: %s base:%s offset:%d length:%d length_bytes:%d elem_size:%d", name, status, root, offsetBytes, lengthConst, sliceBounds.LengthBytes, elementSize(elem))
				default:
					note = fmt.Sprintf("%s raw_slice_bounds: %s base:%s offset:%d length:%s elem_size:%d", name, status, root, offsetBytes, length, elementSize(elem))
				}
			}
		}
	}
	value := Value{
		ID:          id,
		Kind:        ValueView,
		Type:        "[]" + elem,
		Source:      sourceString(call.At),
		Region:      "external:" + targetName,
		Provenance:  provenance,
		UnsafeClass: unsafeClass,
		Lifetime:    Lifetime{Birth: sourceString(call.At), Owner: targetName},
		Mutable:     true,
		Escape:      EscapeConservative,
	}
	b.addValue(value)
	b.addOperation(Operation{Kind: OpUnsafe, Source: sourceString(call.At), Inputs: callInputs(call.Args), Outputs: []string{id}, UnsafeClass: unsafeClass, Note: note})
	if status == runtimeabi.RawSliceBoundsExternalUnknown {
		b.addFact(Fact{Kind: FactProvenanceUnknown, ValueID: id, Reason: "raw slice gateway has external provenance unless an unsafe proof supplies more facts"})
	}
	b.reclassifyMemoryBinding(targetName, provenance, "raw slice gateway has external provenance unless an unsafe proof supplies more facts")
}

func (b *builder) recordRawPointerExposure(call *frontend.CallExpr) {
	root := rawPointerRoot(callArgPath(call, 0))
	if root == "" {
		return
	}
	b.rawExposedRoots[root] = true
}

func (b *builder) recordSliceViewCall(name string, valueType string, method string, call *frontend.CallExpr, targetName string) {
	if targetName == "" {
		targetName = b.syntheticTargetName("slice_view", call)
	}
	id := valueID(ValueView, targetName)
	source := firstArgPath(call)
	prov, known := b.derivedProvenance(source)
	sourceInvalid := b.exprIsInvalidView(callArg(call, 0))
	invalidView := staticInvalidStringViewCall(name, call) || sourceInvalid
	if invalidView {
		prov = Provenance{Kind: ProvenanceUnknown}
		known = false
	}
	value := Value{
		ID:         id,
		Kind:       ValueView,
		Type:       valueType,
		Source:     sourceString(call.At),
		Region:     "fn:" + b.fn.Name,
		Provenance: prov,
		Lifetime:   Lifetime{Birth: sourceString(call.At), Owner: targetName},
		Borrow:     BorrowImm,
		Mutable:    true,
		Escape:     EscapeNoEscape,
	}
	b.addValue(value)
	if invalidView {
		reason := "statically invalid String view has no constructed header"
		if sourceInvalid {
			reason = "view source is invalid before construction"
		}
		b.addOperation(Operation{Kind: OpSliceWindow, Source: sourceString(call.At), Inputs: callInputs(call.Args), Outputs: []string{id}, Note: name + " invalid range is rejected before construction"})
		b.addFact(Fact{Kind: FactProvenanceUnknown, ValueID: id, Reason: reason})
		b.reclassifyMemoryBinding(targetName, Provenance{Kind: ProvenanceUnknown}, reason)
		return
	}
	windowRange := b.sliceViewRange(method, source, call)
	width, shift := sliceViewElementLayout(valueType)
	b.addOperation(Operation{Kind: OpSliceWindow, Source: sourceString(call.At), Inputs: callInputs(call.Args), Outputs: []string{id}, Note: fmt.Sprintf("%s range %s elem_width:%d elem_shift:%d bounds_check:normal_build", name, windowRange, width, shift)})
	b.addFact(Fact{Kind: FactDerivedWindow, ValueID: id, Range: windowRange, Source: sourceString(call.At), Reason: "safe slice view range is checked before construction"})
	b.addFact(Fact{Kind: FactRegionAlive, ValueID: id, Region: value.Region})
	b.addFact(Fact{Kind: FactBorrowedImm, ValueID: id})
	b.addFact(Fact{Kind: FactNoEscape, ValueID: id, Reason: "slice view may not escape its owner"})
	if known {
		b.addFact(Fact{Kind: FactProvenanceKnown, ValueID: id, Reason: "slice view provenance is derived from source slice"})
		b.addFact(Fact{Kind: FactLenStable, ValueID: id, Reason: "safe slice view metadata is constructed by checked compiler builtin"})
		return
	}
	b.addFact(Fact{Kind: FactProvenanceUnknown, ValueID: id, Reason: "slice view source provenance is external or unknown"})
	b.reclassifyMemoryBinding(targetName, prov, "slice view source provenance is external or unknown")
}

func (b *builder) recordBorrowCall(name string, valueType string, call *frontend.CallExpr, targetName string) {
	if targetName == "" {
		targetName = b.syntheticTargetName("borrow", call)
	}
	id := valueID(ValueView, targetName)
	source := firstArgPath(call)
	prov, known := b.derivedProvenance(source)
	value := Value{
		ID:         id,
		Kind:       ValueView,
		Type:       valueType,
		Source:     sourceString(call.At),
		Region:     "fn:" + b.fn.Name,
		Provenance: prov,
		Lifetime:   Lifetime{Birth: sourceString(call.At), Owner: targetName},
		Borrow:     BorrowImm,
		Mutable:    false,
		Escape:     EscapeNoEscape,
	}
	b.addValue(value)
	b.addOperation(Operation{Kind: OpCall, Source: sourceString(call.At), Inputs: callInputs(call.Args), Outputs: []string{id}, Note: name + " creates borrowed view without allocation"})
	b.addFact(Fact{Kind: FactBorrowedImm, ValueID: id, Reason: "explicit borrow view"})
	b.addFact(Fact{Kind: FactNoEscape, ValueID: id, Reason: "explicit borrowed view may not escape owner"})
	b.addFact(Fact{Kind: FactRegionAlive, ValueID: id, Region: value.Region})
	if known {
		b.addFact(Fact{Kind: FactProvenanceKnown, ValueID: id, Reason: "borrow preserves source provenance"})
		b.addFact(Fact{Kind: FactLenStable, ValueID: id, Reason: "borrowed view header is immutable in safe code"})
	} else {
		b.addFact(Fact{Kind: FactProvenanceUnknown, ValueID: id, Reason: "borrow source provenance is external or unknown"})
		b.reclassifyMemoryBinding(targetName, prov, "borrow source provenance is external or unknown")
	}
	b.copyDerivedWindowFacts(source, id, "borrow preserves derived window range")
}

func (b *builder) recordCopyCall(name string, valueType string, elem string, call *frontend.CallExpr, targetName string) {
	if targetName == "" {
		targetName = b.syntheticTargetName("copy", call)
	}
	id := valueID(ValueAllocIntent, targetName)
	source := firstArgPath(call)
	lengthExpr := "expr.len"
	if source != "" {
		lengthExpr = source + ".len"
	}
	lengthConst, lengthConstKnown := b.copyLengthConst(call)
	value := Value{
		ID:     id,
		Kind:   ValueAllocIntent,
		Type:   valueType,
		Source: sourceString(call.At),
		Region: "allocation:" + targetName,
		Alloc: &AllocIntent{
			ElementType:         elem,
			ElementSize:         elementSize(elem),
			LengthExpr:          lengthExpr,
			LengthConstKnown:    lengthConstKnown,
			LengthConst:         lengthConst,
			ZeroGuardStatus:     "valid_empty_no_allocator",
			NegativeGuardStatus: "reject_before_allocation",
			OverflowGuardStatus: "reject_before_allocation",
			Builtin:             name,
			Source:              sourceString(call.At),
		},
		Provenance: Provenance{Kind: ProvenanceAllocation, Root: targetName},
		Lifetime:   Lifetime{Birth: sourceString(call.At), Owner: targetName},
		Borrow:     BorrowNone,
		Mutable:    true,
		Escape:     EscapeConservative,
	}
	b.addValue(value)
	op := b.addOperation(Operation{Kind: OpAllocIntent, Source: sourceString(call.At), Inputs: callInputs(call.Args), Outputs: []string{id}, Note: name + " creates owned copy with new provenance"})
	b.addFact(Fact{Kind: FactOwned, ValueID: id, Reason: "copy result owns new storage"})
	b.addFact(Fact{Kind: FactProvenanceKnown, ValueID: id, Reason: "copy creates owned value with new provenance"})
	b.addFact(Fact{Kind: FactLenStable, ValueID: id, Reason: "copy result metadata is owned by the new allocation"})
	b.addFact(Fact{Kind: FactRegionAlive, ValueID: id, Region: value.Region})
	b.addCopyLoopRangeProof(name, call, op)
}

func (b *builder) addCopyLoopRangeProof(name string, call *frontend.CallExpr, op Operation) {
	if call == nil {
		return
	}
	proofID := copyLoopProofID(name, call.At)
	source := firstArgPath(call)
	if source == "" {
		source = "source"
	}
	indexName := fmt.Sprintf("copy:%s:%d:%d:index", proofNamePart(name, "copy"), call.At.Line, call.At.Col)
	indexID := valueID(ValueLoopIndex, indexName)
	b.addValue(Value{
		ID:         indexID,
		Kind:       ValueLoopIndex,
		Type:       "i32",
		Source:     sourceString(call.At),
		Region:     "fn:" + b.fn.Name,
		Provenance: Provenance{Kind: ProvenanceStack, Root: indexName},
		Lifetime:   Lifetime{Birth: "copy_loop:start", Death: "copy_loop:end", Owner: indexName},
		Escape:     EscapeNoEscape,
	})
	b.addFact(Fact{
		Kind:    FactIndexInRange,
		ValueID: indexID,
		Range:   "0.." + source + ".len",
		ProofID: proofID,
		Source:  sourceString(call.At),
		Reason:  "copy loop source index is dominated by index < source.len guard",
		Uses:    callInputs(call.Args),
	})
	b.proofGuards = append(b.proofGuards, ProofGuard{
		ID:        proofID,
		Kind:      "range",
		Block:     op.Block,
		OpID:      op.ID,
		Condition: indexName + " < " + source + ".len",
		Reason:    "copy loop range proof",
	})
	b.proofUses = append(b.proofUses, ProofUse{
		ProofID: proofID,
		Block:   op.Block,
		OpID:    op.ID,
		UseKind: "bounds_check",
		Source:  sourceString(call.At),
	})
	b.addBoundsProofTerm(rangeProof{
		ID:             proofID,
		IndexName:      indexName,
		IndexValueID:   indexID,
		Base:           source,
		Condition:      indexName + " < " + source + ".len",
		Source:         sourceString(call.At),
		RangeText:      "0.." + source + ".len",
		Lower:          Bound{Kind: BoundConst, Const: 0},
		Upper:          Bound{Kind: BoundSymbol, Symbol: source + ".len"},
		InclusiveLower: true,
		InclusiveUpper: false,
		Reason:         "copy loop range proof",
		Derivation:     []string{"non_negative", "less_than_len"},
	})
	b.rangeFacts = append(b.rangeFacts, RangeFact{
		Value:          indexID,
		Lower:          Bound{Kind: BoundConst, Const: 0},
		Upper:          Bound{Kind: BoundSymbol, Symbol: source + ".len"},
		InclusiveLower: true,
		InclusiveUpper: false,
		Source:         sourceString(call.At),
		ProofID:        proofID,
		Reason:         "copy loop range proof",
	})
}

func (b *builder) copyLengthConst(call *frontend.CallExpr) (int64, bool) {
	if call == nil || len(call.Args) == 0 {
		return 0, false
	}
	if n, ok := directViewLengthConst(call.Args[0]); ok {
		return n, true
	}
	source := firstArgPath(call)
	if source == "" {
		return 0, false
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
			if n, ok := derivedWindowLengthConst(fact.Range); ok {
				return n, true
			}
		}
	}
	return 0, false
}

func directViewLengthConst(expr frontend.Expr) (int64, bool) {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return 0, false
	}
	name := call.Name
	if target, aliasOK := semantics.ResolveBuiltinAlias(name); aliasOK {
		name = target
	}
	switch {
	case strings.HasPrefix(name, "core.slice_borrow_") || name == "core.string_borrow":
		if len(call.Args) != 1 {
			return 0, false
		}
		return directViewLengthConst(call.Args[0])
	case strings.HasPrefix(name, "core.slice_window_") || name == "core.string_window":
		if len(call.Args) != 3 {
			return 0, false
		}
		return evalConstInt64(call.Args[2])
	case strings.HasPrefix(name, "core.slice_prefix_") || name == "core.string_prefix":
		if len(call.Args) != 2 {
			return 0, false
		}
		return evalConstInt64(call.Args[1])
	default:
		return 0, false
	}
}

func derivedWindowLengthConst(rangeText string) (int64, bool) {
	start := strings.LastIndex(rangeText, "[")
	end := strings.LastIndex(rangeText, "]")
	if start < 0 || end <= start {
		return 0, false
	}
	parts := strings.Split(rangeText[start+1:end], "..")
	if len(parts) != 2 {
		return 0, false
	}
	lo := strings.TrimSpace(parts[0])
	hi := strings.TrimSpace(parts[1])
	if plus := strings.LastIndex(hi, "+"); plus >= 0 {
		prefix := strings.TrimSpace(hi[:plus])
		if prefix == lo {
			n, err := strconv.ParseInt(strings.TrimSpace(hi[plus+1:]), 10, 64)
			return n, err == nil
		}
	}
	if lo == "0" {
		n, err := strconv.ParseInt(hi, 10, 64)
		return n, err == nil
	}
	return 0, false
}

func (b *builder) copyDerivedWindowFacts(source string, dstValueID string, reason string) {
	if source == "" {
		return
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
			b.addFact(Fact{Kind: FactDerivedWindow, ValueID: dstValueID, Range: fact.Range, Source: fact.Source, Reason: reason, Uses: []string{candidate}})
			return
		}
	}
}

func (b *builder) derivedProvenance(source string) (Provenance, bool) {
	if source == "" {
		return Provenance{Kind: ProvenanceUnknown}, false
	}
	if strings.HasPrefix(source, "string:") {
		return Provenance{Kind: ProvenanceLiteral, Root: source}, true
	}
	for _, kind := range []ValueKind{ValueAllocIntent, ValueView, ValueParam, ValueLocal} {
		id := valueID(kind, source)
		if value, ok := b.values[id]; ok {
			switch value.Provenance.Kind {
			case ProvenanceUnknown, "":
				return Provenance{Kind: ProvenanceUnknown}, false
			case ProvenanceExternal:
				return Provenance{Kind: ProvenanceExternal, Root: "derived:" + value.Provenance.Root}, false
			default:
				return Provenance{Kind: value.Provenance.Kind, Root: "derived:" + value.Provenance.Root}, true
			}
		}
	}
	return Provenance{Kind: ProvenanceParam, Root: "derived:" + source}, true
}
