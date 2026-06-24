package flow

import (
	"encoding/binary"
	"errors"
	"hash/fnv"
	"sort"

	"tetra_language/compiler/internal/frontend"
)

var ErrFlowFixedPointNonConvergence = errors.New("flow fixed point did not converge")

type LoopOptions struct {
	MaxIterations int
	JoinOptions   JoinOptions
}

func Join(left, right FlowState, opts JoinOptions) FlowState {
	return JoinWithLabels(left, right, "", "", opts)
}

func JoinWithLabels(
	left, right FlowState,
	leftLabel, rightLabel string,
	opts JoinOptions,
) FlowState {
	switch {
	case !left.Reachable && !right.Reachable:
		out := left.Clone()
		out.Reachable = false
		return out
	case !left.Reachable:
		return right.Clone()
	case !right.Reachable:
		return left.Clone()
	}
	return FlowState{
		Reachable:               true,
		Availability:            joinAvailabilityMaps(left.Availability, right.Availability, leftLabel, rightLabel),
		OwnershipAliases:        joinMatchingStringMap(left.OwnershipAliases, right.OwnershipAliases),
		BorrowedPointerOwners:   joinLexicalStringUnion(left.BorrowedPointerOwners, right.BorrowedPointerOwners),
		RegionBindings:          joinRegionBindings(left.RegionBindings, right.RegionBindings, leftLabel, rightLabel),
		RegionConflicts:         joinRegionConflicts(left, right, leftLabel, rightLabel),
		ResourceIdentities:      joinResourceIdentities(left.ResourceIdentities, right.ResourceIdentities, opts),
		ResourceAvailability:    joinResourceAvailability(left.ResourceAvailability, right.ResourceAvailability, leftLabel, rightLabel),
		ResourceFinalization:    joinResourceFinalizationMaps(left.ResourceFinalization, right.ResourceFinalization),
		OwnedRegionSliceOwners:  joinMatchingStringMap(left.OwnedRegionSliceOwners, right.OwnedRegionSliceOwners),
		AwaitInvalidatedBorrows: joinPositionUnion(left.AwaitInvalidatedBorrows, right.AwaitInvalidatedBorrows),
		PendingDeferCaptures:    joinCaptureFrames(left.PendingDeferCaptures, right.PendingDeferCaptures),
	}
}

func FixedPointLoop(
	entry FlowState,
	analyze func(FlowState) (FlowState, error),
	opts LoopOptions,
) (FlowState, int, error) {
	maxIterations := opts.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 32
	}
	header := entry.Clone()
	for i := 0; i < maxIterations; i++ {
		bodyExit, err := analyze(header.Clone())
		if err != nil {
			return FlowState{}, i, err
		}
		next := Join(entry, bodyExit, opts.JoinOptions)
		if next.Digest() == header.Digest() {
			return next, i + 1, nil
		}
		header = next
	}
	return header, maxIterations, ErrFlowFixedPointNonConvergence
}

func joinAvailabilityMaps(
	left, right map[string]Availability,
	leftLabel, rightLabel string,
) map[string]Availability {
	keys := stringKeys(left, right)
	out := make(map[string]Availability, len(keys))
	for _, key := range keys {
		leftValue, leftOK := left[key]
		rightValue, rightOK := right[key]
		if !leftOK {
			leftValue = Availability{Kind: AvailabilityAvailable}
		}
		if !rightOK {
			rightValue = Availability{Kind: AvailabilityAvailable}
		}
		joined := joinAvailability(leftValue, rightValue, leftLabel, rightLabel)
		if joined.Kind != "" {
			out[key] = joined
		}
	}
	return out
}

func joinAvailability(
	left, right Availability,
	leftLabel, rightLabel string,
) Availability {
	if left.Kind == "" {
		left.Kind = AvailabilityAvailable
	}
	if right.Kind == "" {
		right.Kind = AvailabilityAvailable
	}
	if left.Kind == AvailabilityAvailable && right.Kind == AvailabilityAvailable {
		if left == right {
			return left
		}
		return Availability{Kind: AvailabilityAvailable}
	}
	if left.Kind == AvailabilityConsumed && right.Kind == AvailabilityConsumed {
		return Availability{
			Kind: AvailabilityConsumed,
			Pos:  earliestPosition(left.Pos, right.Pos),
		}
	}
	if left.Kind == AvailabilityAvailable && right.Kind != AvailabilityAvailable {
		left, right = right, left
		leftLabel, rightLabel = rightLabel, leftLabel
	}
	return Availability{
		Kind:          AvailabilityMaybeConsumed,
		Pos:           earliestPosition(availabilityPosition(left), availabilityPosition(right)),
		LeftLabel:     leftLabel,
		LeftConsumed:  left.Kind != AvailabilityAvailable,
		LeftPos:       availabilityPosition(left),
		RightLabel:    rightLabel,
		RightConsumed: right.Kind != AvailabilityAvailable,
		RightPos:      availabilityPosition(right),
	}
}

func availabilityPosition(value Availability) frontend.Position {
	if value.Pos.Line != 0 || value.Pos.Col != 0 {
		return value.Pos
	}
	return earliestPosition(value.LeftPos, value.RightPos)
}

func joinRegionBindings(
	left, right map[string]RegionBinding,
	leftLabel, rightLabel string,
) map[string]RegionBinding {
	keys := stringKeys(left, right)
	out := make(map[string]RegionBinding, len(keys))
	for _, key := range keys {
		leftValue, leftOK := left[key]
		rightValue, rightOK := right[key]
		joined := joinRegionBinding(leftValue, leftOK, rightValue, rightOK)
		if joined.Kind != RegionNone && joined.Kind != "" {
			out[key] = joined
		}
		_ = leftLabel
		_ = rightLabel
	}
	return out
}

func joinRegionBinding(
	left RegionBinding,
	leftOK bool,
	right RegionBinding,
	rightOK bool,
) RegionBinding {
	if !leftOK {
		left = RegionBinding{Kind: RegionNone}
	}
	if !rightOK {
		right = RegionBinding{Kind: RegionNone}
	}
	if left.Kind == "" {
		left.Kind = RegionNone
	}
	if right.Kind == "" {
		right.Kind = RegionNone
	}
	if left.Kind == RegionUnknown || right.Kind == RegionUnknown {
		return RegionBinding{Kind: RegionUnknown}
	}
	if left.Kind == RegionNone && right.Kind == RegionNone {
		return RegionBinding{Kind: RegionNone}
	}
	if left.Kind == RegionExact && right.Kind == RegionExact && left.RegionID == right.RegionID {
		return left
	}
	return RegionBinding{Kind: RegionUnknown}
}

func joinRegionConflicts(
	left, right FlowState,
	leftLabel, rightLabel string,
) map[string]RegionConflict {
	out := cloneMap(left.RegionConflicts)
	for key, conflict := range right.RegionConflicts {
		if _, exists := out[key]; !exists {
			out[key] = conflict
		}
	}
	for _, key := range stringKeys(left.RegionBindings, right.RegionBindings) {
		leftValue, leftOK := left.RegionBindings[key]
		rightValue, rightOK := right.RegionBindings[key]
		joined := joinRegionBinding(leftValue, leftOK, rightValue, rightOK)
		if joined.Kind != RegionUnknown {
			continue
		}
		out[key] = canonicalRegionConflict(
			leftValue,
			leftOK,
			rightValue,
			rightOK,
			leftLabel,
			rightLabel,
		)
	}
	return out
}

func canonicalRegionConflict(
	left RegionBinding,
	leftOK bool,
	right RegionBinding,
	rightOK bool,
	leftLabel, rightLabel string,
) RegionConflict {
	_ = left
	_ = leftOK
	_ = right
	_ = rightOK
	_ = leftLabel
	_ = rightLabel
	return RegionConflict{LeftKind: RegionUnknown, RightKind: RegionUnknown}
}

func diagnosticRegionConflict(
	left RegionBinding,
	leftOK bool,
	right RegionBinding,
	rightOK bool,
	leftLabel, rightLabel string,
) RegionConflict {
	leftKind := normalizedRegionKind(left, leftOK)
	rightKind := normalizedRegionKind(right, rightOK)
	if leftKind == RegionExact && rightKind == RegionExact && left.RegionID > right.RegionID {
		left, right = right, left
		leftKind, rightKind = rightKind, leftKind
		leftLabel, rightLabel = rightLabel, leftLabel
	}
	if leftKind == RegionUnknown || rightKind == RegionUnknown {
		return RegionConflict{LeftKind: RegionUnknown, RightKind: RegionUnknown}
	}
	return RegionConflict{
		LeftLabel:   leftLabel,
		LeftKind:    leftKind,
		LeftRegion:  left.RegionID,
		RightLabel:  rightLabel,
		RightKind:   rightKind,
		RightRegion: right.RegionID,
	}
}

func normalizedRegionKind(value RegionBinding, ok bool) RegionBindingKind {
	if !ok || value.Kind == "" {
		return RegionNone
	}
	return value.Kind
}

func joinMatchingStringMap(left, right map[string]string) map[string]string {
	out := map[string]string{}
	for key, leftValue := range left {
		if leftValue == "" {
			continue
		}
		if rightValue, ok := right[key]; ok && rightValue == leftValue {
			out[key] = leftValue
		}
	}
	return out
}

func joinLexicalStringUnion(left, right map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range left {
		if value != "" {
			out[key] = value
		}
	}
	for key, value := range right {
		if value == "" {
			continue
		}
		if existing, exists := out[key]; exists && existing <= value {
			continue
		}
		out[key] = value
	}
	return out
}

func joinResourceIdentities(
	left, right map[string]ResourceIdentity,
	opts JoinOptions,
) map[string]ResourceIdentity {
	out := map[string]ResourceIdentity{}
	for _, key := range stringKeys(left, right) {
		leftValue, leftOK := left[key]
		rightValue, rightOK := right[key]
		switch {
		case leftOK && !rightOK:
			out[key] = leftValue
		case !leftOK && rightOK:
			out[key] = rightValue
		case leftValue == rightValue:
			out[key] = leftValue
		default:
			out[key] = mergeResourceIdentity(leftValue, rightValue, opts)
		}
	}
	return out
}

func mergeResourceIdentity(left, right ResourceIdentity, opts JoinOptions) ResourceIdentity {
	merge := ResourceMerge{
		LeftID:  left.ID,
		RightID: right.ID,
		Left:    left,
		Right:   right,
	}
	id := deterministicResourceMergeID(merge)
	if opts.AllocateResourceID != nil {
		id = opts.AllocateResourceID(merge)
	}
	out := ResourceIdentity{ID: id, Unknown: left.Unknown || right.Unknown}
	if left.HasParam && right.HasParam &&
		left.ParamIndex == right.ParamIndex &&
		left.ParamPath == right.ParamPath {
		out.HasParam = true
		out.ParamIndex = left.ParamIndex
		out.ParamPath = left.ParamPath
	} else if !left.HasParam && !right.HasParam {
		out.HasParam = false
	} else {
		out.Unknown = true
	}
	return out
}

func deterministicResourceMergeID(merge ResourceMerge) int {
	left, right := merge.LeftID, merge.RightID
	if left == right {
		return left
	}
	if left > right {
		left, right = right, left
	}
	hash := fnv.New32a()
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(left))
	_, _ = hash.Write(buf[:])
	_, _ = hash.Write([]byte{0})
	binary.LittleEndian.PutUint64(buf[:], uint64(right))
	_, _ = hash.Write(buf[:])
	return -100000000 - int(hash.Sum32()%100000000)
}

func joinResourceAvailability(
	left, right map[int]Availability,
	leftLabel, rightLabel string,
) map[int]Availability {
	keys := intKeys(left, right)
	out := make(map[int]Availability, len(keys))
	for _, key := range keys {
		leftValue, leftOK := left[key]
		rightValue, rightOK := right[key]
		if !leftOK {
			leftValue = Availability{Kind: AvailabilityAvailable}
		}
		if !rightOK {
			rightValue = Availability{Kind: AvailabilityAvailable}
		}
		out[key] = joinAvailability(leftValue, rightValue, leftLabel, rightLabel)
	}
	return out
}

func joinResourceFinalizationMaps(
	left, right map[int]ResourceFinalization,
) map[int]ResourceFinalization {
	keys := intKeys(left, right)
	out := make(map[int]ResourceFinalization, len(keys))
	for _, key := range keys {
		leftValue, leftOK := left[key]
		rightValue, rightOK := right[key]
		joined, ok := joinResourceFinalization(leftValue, leftOK, rightValue, rightOK)
		if ok {
			out[key] = joined
		}
	}
	return out
}

func joinResourceFinalization(
	left ResourceFinalization,
	leftOK bool,
	right ResourceFinalization,
	rightOK bool,
) (ResourceFinalization, bool) {
	if !leftOK && !rightOK {
		return ResourceFinalization{}, false
	}
	if !leftOK {
		left = ResourceFinalization{Kind: FinalizationLive}
	}
	if !rightOK {
		right = ResourceFinalization{Kind: FinalizationLive}
	}
	if left.Kind == "" {
		left.Kind = FinalizationLive
	}
	if right.Kind == "" {
		right.Kind = FinalizationLive
	}
	if left.Kind == FinalizationLive && right.Kind == FinalizationLive {
		return ResourceFinalization{}, false
	}
	if left.Kind == FinalizationExact && right.Kind == FinalizationExact && left.State == right.State {
		left.Pos = earliestPosition(left.Pos, right.Pos)
		return left, true
	}
	states := map[string]frontend.Position{}
	addFinalizationState(states, left)
	addFinalizationState(states, right)
	return ResourceFinalization{
		Kind:           FinalizationMaybe,
		State:          firstStringKey(states),
		Pos:            earliestPositionInMap(states),
		MayBeAvailable: left.Kind == FinalizationLive || right.Kind == FinalizationLive || left.MayBeAvailable || right.MayBeAvailable,
		States:         states,
	}, true
}

func addFinalizationState(dst map[string]frontend.Position, value ResourceFinalization) {
	if value.State != "" {
		if existing, ok := dst[value.State]; ok {
			dst[value.State] = earliestPosition(existing, value.Pos)
		} else {
			dst[value.State] = value.Pos
		}
	}
	for state, pos := range value.States {
		if existing, ok := dst[state]; ok {
			dst[state] = earliestPosition(existing, pos)
		} else {
			dst[state] = pos
		}
	}
}

func joinPositionUnion[K comparable](
	left, right map[K]frontend.Position,
) map[K]frontend.Position {
	out := cloneMap(left)
	for key, value := range right {
		if existing, ok := out[key]; ok {
			out[key] = earliestPosition(existing, value)
		} else {
			out[key] = value
		}
	}
	return out
}

func joinCaptureFrames(
	left, right []map[string]frontend.Position,
) []map[string]frontend.Position {
	count := len(left)
	if len(right) > count {
		count = len(right)
	}
	if count == 0 {
		return []map[string]frontend.Position{}
	}
	out := make([]map[string]frontend.Position, count)
	for i := 0; i < count; i++ {
		var leftFrame, rightFrame map[string]frontend.Position
		if i < len(left) {
			leftFrame = left[i]
		}
		if i < len(right) {
			rightFrame = right[i]
		}
		out[i] = joinPositionUnion(leftFrame, rightFrame)
	}
	return out
}

func earliestPosition(left, right frontend.Position) frontend.Position {
	if left.Line == 0 {
		return right
	}
	if right.Line == 0 {
		return left
	}
	if left.Line < right.Line || (left.Line == right.Line && left.Col <= right.Col) {
		return left
	}
	return right
}

func earliestPositionInMap(values map[string]frontend.Position) frontend.Position {
	var earliest frontend.Position
	for _, pos := range values {
		earliest = earliestPosition(earliest, pos)
	}
	return earliest
}

func firstStringKey(values map[string]frontend.Position) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func stringKeys[V any](left, right map[string]V) []string {
	seen := make(map[string]struct{}, len(left)+len(right))
	for key := range left {
		seen[key] = struct{}{}
	}
	for key := range right {
		seen[key] = struct{}{}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func intKeys[V any](left, right map[int]V) []int {
	seen := make(map[int]struct{}, len(left)+len(right))
	for key := range left {
		seen[key] = struct{}{}
	}
	for key := range right {
		seen[key] = struct{}{}
	}
	keys := make([]int, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}
