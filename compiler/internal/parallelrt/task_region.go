package parallelrt

import (
	"errors"

	"tetra_language/compiler/internal/stdlibrt"
)

var ErrInvalidTaskRegionScope = errors.New("invalid task region scope")

type TaskRegionOptions struct {
	RegionID string
	Lifetime string
	Capacity int
}

type TaskRegionReport struct {
	RegionID             string
	Lifetime             string
	RuntimePath          string
	BytesUsedBeforeReset int
	Reset                bool
}

type TaskRegionScope struct {
	region   *stdlibrt.Region
	lifetime string
}

func NewTaskRegionScope(opt TaskRegionOptions) *TaskRegionScope {
	regionID := opt.RegionID
	if regionID == "" {
		regionID = "task"
	}
	capacity := opt.Capacity
	if capacity <= 0 {
		capacity = 8192
	}
	lifetime := opt.Lifetime
	if lifetime == "" {
		lifetime = regionID
	}
	return &TaskRegionScope{
		region:   stdlibrt.NewRegion(regionID, capacity),
		lifetime: lifetime,
	}
}

func (s *TaskRegionScope) RegionUsed() int {
	if s == nil || s.region == nil {
		return 0
	}
	return s.region.Used()
}

func (s *TaskRegionScope) Run(
	name string,
	task func(*stdlibrt.Region) error,
) (report TaskRegionReport, err error) {
	if s == nil || s.region == nil || task == nil {
		return TaskRegionReport{}, ErrInvalidTaskRegionScope
	}
	report.RegionID = s.region.ID()
	report.Lifetime = s.lifetime
	report.RuntimePath = "task_region"
	defer func() {
		report.BytesUsedBeforeReset = s.region.Used()
		if resetErr := s.region.Reset(); resetErr != nil && err == nil {
			err = resetErr
		}
		report.Reset = true
	}()
	err = task(s.region)
	return
}
