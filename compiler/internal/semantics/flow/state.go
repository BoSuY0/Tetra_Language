package flow

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	"tetra_language/compiler/internal/frontend"
)

type AvailabilityKind string

const (
	AvailabilityAvailable     AvailabilityKind = "available"
	AvailabilityConsumed      AvailabilityKind = "consumed"
	AvailabilityMaybeConsumed AvailabilityKind = "maybe_consumed"
)

type RegionBindingKind string

const (
	RegionNone    RegionBindingKind = "none"
	RegionExact   RegionBindingKind = "exact"
	RegionUnknown RegionBindingKind = "unknown"
)

type FinalizationKind string

const (
	FinalizationLive  FinalizationKind = "live"
	FinalizationExact FinalizationKind = "finalized"
	FinalizationMaybe FinalizationKind = "maybe_finalized"
)

type Availability struct {
	Kind          AvailabilityKind  `json:"kind"`
	Pos           frontend.Position `json:"pos,omitempty"`
	LeftLabel     string            `json:"left_label,omitempty"`
	LeftConsumed  bool              `json:"left_consumed,omitempty"`
	LeftPos       frontend.Position `json:"left_pos,omitempty"`
	RightLabel    string            `json:"right_label,omitempty"`
	RightConsumed bool              `json:"right_consumed,omitempty"`
	RightPos      frontend.Position `json:"right_pos,omitempty"`
}

type RegionBinding struct {
	Kind     RegionBindingKind `json:"kind"`
	RegionID int               `json:"region_id,omitempty"`
}

type RegionConflict struct {
	LeftLabel   string            `json:"left_label,omitempty"`
	LeftKind    RegionBindingKind `json:"left_kind,omitempty"`
	LeftRegion  int               `json:"left_region,omitempty"`
	RightLabel  string            `json:"right_label,omitempty"`
	RightKind   RegionBindingKind `json:"right_kind,omitempty"`
	RightRegion int               `json:"right_region,omitempty"`
}

type ResourceIdentity struct {
	ID         int    `json:"id"`
	HasParam   bool   `json:"has_param,omitempty"`
	ParamIndex int    `json:"param_index,omitempty"`
	ParamPath  string `json:"param_path,omitempty"`
	Unknown    bool   `json:"unknown,omitempty"`
}

type ResourceFinalization struct {
	Kind           FinalizationKind             `json:"kind"`
	State          string                       `json:"state,omitempty"`
	Pos            frontend.Position            `json:"pos,omitempty"`
	MayBeAvailable bool                         `json:"may_be_available,omitempty"`
	States         map[string]frontend.Position `json:"states,omitempty"`
}

type ResourceMerge struct {
	LeftID  int
	RightID int
	Left    ResourceIdentity
	Right   ResourceIdentity
}

type JoinOptions struct {
	AllocateResourceID func(ResourceMerge) int
}

type FlowState struct {
	Reachable               bool
	Availability            map[string]Availability
	OwnershipAliases        map[string]string
	BorrowedPointerOwners   map[string]string
	RegionBindings          map[string]RegionBinding
	RegionConflicts         map[string]RegionConflict
	ResourceIdentities      map[string]ResourceIdentity
	ResourceAvailability    map[int]Availability
	ResourceFinalization    map[int]ResourceFinalization
	OwnedRegionSliceOwners  map[string]string
	AwaitInvalidatedBorrows map[int]frontend.Position
	PendingDeferCaptures    []map[string]frontend.Position
}

func NewReachableState() FlowState {
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
	}
}

func (s FlowState) Clone() FlowState {
	return FlowState{
		Reachable:               s.Reachable,
		Availability:            cloneMap(s.Availability),
		OwnershipAliases:        cloneMap(s.OwnershipAliases),
		BorrowedPointerOwners:   cloneMap(s.BorrowedPointerOwners),
		RegionBindings:          cloneMap(s.RegionBindings),
		RegionConflicts:         cloneMap(s.RegionConflicts),
		ResourceIdentities:      cloneMap(s.ResourceIdentities),
		ResourceAvailability:    cloneMap(s.ResourceAvailability),
		ResourceFinalization:    cloneResourceFinalizationMap(s.ResourceFinalization),
		OwnedRegionSliceOwners:  cloneMap(s.OwnedRegionSliceOwners),
		AwaitInvalidatedBorrows: cloneMap(s.AwaitInvalidatedBorrows),
		PendingDeferCaptures:    cloneCaptureFrames(s.PendingDeferCaptures),
	}
}

func (s FlowState) Digest() string {
	normalized := s.Clone()
	normalizeFlowState(&normalized)
	raw, err := json.Marshal(normalized)
	if err != nil {
		panic(fmt.Sprintf("flow state digest marshal failed: %v", err))
	}
	sum := fnv.New64a()
	_, _ = sum.Write(raw)
	return fmt.Sprintf("%016x", sum.Sum64())
}

func normalizeFlowState(s *FlowState) {
	if s.Availability == nil {
		s.Availability = map[string]Availability{}
	}
	if s.OwnershipAliases == nil {
		s.OwnershipAliases = map[string]string{}
	}
	if s.BorrowedPointerOwners == nil {
		s.BorrowedPointerOwners = map[string]string{}
	}
	if s.RegionBindings == nil {
		s.RegionBindings = map[string]RegionBinding{}
	}
	if s.RegionConflicts == nil {
		s.RegionConflicts = map[string]RegionConflict{}
	}
	if s.ResourceIdentities == nil {
		s.ResourceIdentities = map[string]ResourceIdentity{}
	}
	if s.ResourceAvailability == nil {
		s.ResourceAvailability = map[int]Availability{}
	}
	if s.ResourceFinalization == nil {
		s.ResourceFinalization = map[int]ResourceFinalization{}
	}
	if s.OwnedRegionSliceOwners == nil {
		s.OwnedRegionSliceOwners = map[string]string{}
	}
	if s.AwaitInvalidatedBorrows == nil {
		s.AwaitInvalidatedBorrows = map[int]frontend.Position{}
	}
	if s.PendingDeferCaptures == nil {
		s.PendingDeferCaptures = []map[string]frontend.Position{}
	}
}

func cloneMap[K comparable, V any](src map[K]V) map[K]V {
	if len(src) == 0 {
		return map[K]V{}
	}
	dst := make(map[K]V, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneResourceFinalizationMap(src map[int]ResourceFinalization) map[int]ResourceFinalization {
	if len(src) == 0 {
		return map[int]ResourceFinalization{}
	}
	dst := make(map[int]ResourceFinalization, len(src))
	for id, final := range src {
		copied := final
		copied.States = cloneMap(final.States)
		dst[id] = copied
	}
	return dst
}

func cloneCaptureFrames(src []map[string]frontend.Position) []map[string]frontend.Position {
	if len(src) == 0 {
		return []map[string]frontend.Position{}
	}
	dst := make([]map[string]frontend.Position, len(src))
	for i := range src {
		dst[i] = cloneMap(src[i])
	}
	return dst
}
