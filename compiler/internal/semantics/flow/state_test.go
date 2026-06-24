package flow

import (
	"errors"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func pos(line int) frontend.Position {
	return frontend.Position{Line: line, Col: 1}
}

func consumedAt(line int) Availability {
	return Availability{Kind: AvailabilityConsumed, Pos: pos(line)}
}

func available() Availability {
	return Availability{Kind: AvailabilityAvailable}
}

func reachableState() FlowState {
	return FlowState{
		Reachable:               true,
		Availability:            map[string]Availability{},
		OwnershipAliases:        map[string]string{},
		BorrowedPointerOwners:   map[string]string{},
		RegionBindings:          map[string]RegionBinding{},
		RegionConflicts:         map[string]RegionConflict{},
		ResourceIdentities:      map[string]ResourceIdentity{},
		ResourceAvailability:    map[int]Availability{},
		ResourceFinalization:    map[int]ResourceFinalization{},
		OwnedRegionSliceOwners:  map[string]string{},
		AwaitInvalidatedBorrows: map[int]frontend.Position{},
		PendingDeferCaptures:    []map[string]frontend.Position{},
	}
}

func TestFlowStateJoinLaws(t *testing.T) {
	a := reachableState()
	a.Availability["x"] = consumedAt(4)
	a.RegionBindings["buf"] = RegionBinding{Kind: RegionExact, RegionID: 7}
	a.OwnershipAliases["payload"] = "root.$elem"

	b := reachableState()
	b.Availability["x"] = available()
	b.RegionBindings["buf"] = RegionBinding{Kind: RegionExact, RegionID: 7}
	b.OwnershipAliases["payload"] = "root.$elem"
	b.BorrowedPointerOwners["view"] = "owner_b"

	c := reachableState()
	c.Availability["y"] = consumedAt(9)
	c.RegionBindings["buf"] = RegionBinding{Kind: RegionExact, RegionID: 8}
	c.BorrowedPointerOwners["view"] = "owner_a"

	ab := JoinWithLabels(a, b, "a", "b", JoinOptions{})
	if got := ab.Availability["x"].Kind; got != AvailabilityMaybeConsumed {
		t.Fatalf("Join consumed/available availability = %s, want %s", got, AvailabilityMaybeConsumed)
	}
	if got := ab.RegionBindings["buf"]; got.Kind != RegionExact || got.RegionID != 7 {
		t.Fatalf("same exact region join = %#v, want exact region 7", got)
	}
	if got := ab.OwnershipAliases["payload"]; got != "root.$elem" {
		t.Fatalf("matching ownership alias was not preserved: %q", got)
	}

	if Join(a, b, JoinOptions{}).Digest() != Join(b, a, JoinOptions{}).Digest() {
		t.Fatalf("join is not commutative")
	}
	if Join(a, a, JoinOptions{}).Digest() != a.Clone().Digest() {
		t.Fatalf("join is not idempotent")
	}
	leftAssoc := Join(Join(a, b, JoinOptions{}), c, JoinOptions{}).Digest()
	rightAssoc := Join(a, Join(b, c, JoinOptions{}), JoinOptions{}).Digest()
	if leftAssoc != rightAssoc {
		t.Fatalf("join is not associative:\nleft=%s\nright=%s", leftAssoc, rightAssoc)
	}
	abc := Join(Join(a, b, JoinOptions{}), c, JoinOptions{})
	if got := abc.RegionBindings["buf"].Kind; got != RegionUnknown {
		t.Fatalf("different exact region join = %s, want %s", got, RegionUnknown)
	}
	if got := abc.BorrowedPointerOwners["view"]; got != "owner_a" {
		t.Fatalf("borrowed pointer owner conflict = %q, want deterministic lexical minimum", got)
	}
}

func TestFlowStateJoinReachabilityUsesFallthroughOnly(t *testing.T) {
	continuation := reachableState()
	continuation.Availability["live"] = available()
	returned := reachableState()
	returned.Reachable = false
	returned.Availability["dead"] = consumedAt(3)

	joined := Join(continuation, returned, JoinOptions{})
	if !joined.Reachable {
		t.Fatalf("reachable fallthrough was dropped")
	}
	if _, ok := joined.Availability["dead"]; ok {
		t.Fatalf("unreachable return/throw state was joined into continuation")
	}

	none := Join(returned, returned, JoinOptions{})
	if none.Reachable {
		t.Fatalf("two unreachable exits should remain unreachable")
	}
}

func TestFlowStateBranchMatchIfLetAndCatchScenarios(t *testing.T) {
	branchAvailable := reachableState()
	branchAvailable.Availability["value"] = available()
	branchConsumed := reachableState()
	branchConsumed.Availability["value"] = consumedAt(5)
	if got := JoinWithLabels(
		branchAvailable,
		branchConsumed,
		"then",
		"else",
		JoinOptions{},
	).Availability["value"].Kind; got != AvailabilityMaybeConsumed {
		t.Fatalf("branch available/consumed join = %s, want %s", got, AvailabilityMaybeConsumed)
	}

	matchArm1 := reachableState()
	matchArm2 := reachableState()
	matchArm2.Availability["payload"] = consumedAt(8)
	matchArm3 := reachableState()
	matchJoin := Join(Join(matchArm1, matchArm2, JoinOptions{}), matchArm3, JoinOptions{})
	if got := matchJoin.Availability["payload"].Kind; got != AvailabilityMaybeConsumed {
		t.Fatalf("match 3-arm join = %s, want %s", got, AvailabilityMaybeConsumed)
	}

	ifLetSome := reachableState()
	ifLetSome.OwnershipAliases["payload"] = "optional.$elem"
	ifLetNone := reachableState()
	if got := Join(ifLetSome, ifLetNone, JoinOptions{}).OwnershipAliases["payload"]; got != "" {
		t.Fatalf("if-let alias escaped join: %q", got)
	}

	catchCall := reachableState()
	catchCall.RegionBindings["err"] = RegionBinding{Kind: RegionExact, RegionID: 3}
	catchArm := reachableState()
	catchArm.RegionBindings["err"] = RegionBinding{Kind: RegionExact, RegionID: 4}
	if got := Join(catchCall, catchArm, JoinOptions{}).RegionBindings["err"].Kind; got != RegionUnknown {
		t.Fatalf("catch arm region join = %s, want %s", got, RegionUnknown)
	}
}

func TestFlowStateResourceFinalizationJoin(t *testing.T) {
	live := reachableState()
	finalized := reachableState()
	finalized.ResourceFinalization[1] = ResourceFinalization{
		Kind:  FinalizationExact,
		State: "closed",
		Pos:   pos(7),
	}
	joined := Join(live, finalized, JoinOptions{})
	if got := joined.ResourceFinalization[1].Kind; got != FinalizationMaybe {
		t.Fatalf("live/finalized resource join = %s, want %s", got, FinalizationMaybe)
	}
	if !joined.ResourceFinalization[1].MayBeAvailable {
		t.Fatalf("live/finalized resource join must record may-be-available")
	}

	left := reachableState()
	left.ResourceFinalization[1] = ResourceFinalization{
		Kind:  FinalizationExact,
		State: "closed",
		Pos:   pos(9),
	}
	right := reachableState()
	right.ResourceFinalization[1] = ResourceFinalization{
		Kind:  FinalizationExact,
		State: "closed",
		Pos:   pos(4),
	}
	exact := Join(left, right, JoinOptions{}).ResourceFinalization[1]
	if exact.Kind != FinalizationExact || exact.State != "closed" || exact.Pos.Line != 4 {
		t.Fatalf("same finalization join = %#v, want earliest exact closed", exact)
	}
}

func TestFlowStateDigestDeterministic(t *testing.T) {
	want := ""
	for i := 0; i < 100; i++ {
		state := reachableState()
		state.Availability["b"] = consumedAt(2)
		state.Availability["a"] = available()
		state.RegionBindings["z"] = RegionBinding{Kind: RegionExact, RegionID: 1}
		state.RegionBindings["m"] = RegionBinding{Kind: RegionUnknown}
		state.PendingDeferCaptures = []map[string]frontend.Position{
			{"cleanup": pos(12), "other": pos(11)},
		}
		got := state.Digest()
		if i == 0 {
			want = got
			continue
		}
		if got != want {
			t.Fatalf("digest run %d = %s, want %s", i, got, want)
		}
	}
}

func TestFlowStateLoopFixedPoint(t *testing.T) {
	entry := reachableState()
	entry.Availability["x"] = available()

	seen := 0
	result, iterations, err := FixedPointLoop(entry, func(header FlowState) (FlowState, error) {
		seen++
		body := header.Clone()
		if body.Availability["x"].Kind == AvailabilityAvailable {
			body.Availability["x"] = consumedAt(20)
		}
		return body, nil
	}, LoopOptions{MaxIterations: 32})
	if err != nil {
		t.Fatalf("FixedPointLoop returned error: %v", err)
	}
	if seen < 2 || iterations < 1 {
		t.Fatalf("FixedPointLoop did not iterate to convergence: seen=%d iterations=%d", seen, iterations)
	}
	if got := result.Availability["x"].Kind; got != AvailabilityMaybeConsumed {
		t.Fatalf("loop result availability = %s, want %s", got, AvailabilityMaybeConsumed)
	}
}

func TestFlowStateLoopFixedPointRejectsNonConvergence(t *testing.T) {
	entry := reachableState()
	_, _, err := FixedPointLoop(entry, func(header FlowState) (FlowState, error) {
		next := header.Clone()
		next.RegionBindings[header.Digest()] = RegionBinding{Kind: RegionExact, RegionID: len(header.RegionBindings) + 1}
		return next, nil
	}, LoopOptions{MaxIterations: 3})
	if !errors.Is(err, ErrFlowFixedPointNonConvergence) {
		t.Fatalf("FixedPointLoop error = %v, want ErrFlowFixedPointNonConvergence", err)
	}
}
