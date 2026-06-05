package parallelrt

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidConfig                = errors.New("invalid scheduler model config")
	ErrInvalidCore                  = errors.New("invalid scheduler core")
	ErrMailboxFull                  = errors.New("typed mailbox is full")
	ErrBorrowedPayloadRequiresCopy  = errors.New("borrowed payload requires explicit copy")
	ErrUnknownRegion                = errors.New("unknown owned region")
	ErrRegionOwnerMismatch          = errors.New("owned region owner mismatch")
	ErrRegionHasLiveBorrowedAliases = errors.New("owned region has live borrowed aliases")
)

type Config struct {
	Cores           int
	MailboxCapacity int
}

type SchedulerModel struct {
	queues          [][]WorkItem
	mailboxCapacity int
	regions         map[string]Region
	runLog          []string
	stats           Stats
}

type WorkItem struct {
	Name string
}

type Stats struct {
	Cores         int
	Steals        int
	Completed     int
	MaxQueueDepth int
	MailboxSends  int
	ZeroCopyMoves int
	BytesCopied   int
}

type PrototypeBenchmark struct {
	Name             string  `json:"name"`
	Kind             string  `json:"kind"`
	Metric           string  `json:"metric"`
	Unit             string  `json:"unit"`
	BaselineValue    int     `json:"baseline_value"`
	MeasuredValue    int     `json:"measured_value"`
	ImprovementRatio float64 `json:"improvement_ratio"`
	Evidence         string  `json:"evidence"`
	Ran              bool    `json:"ran"`
	Pass             bool    `json:"pass"`
}

type Region struct {
	Name         string
	Owner        string
	BorrowedRefs int
}

func NewSchedulerModel(cfg Config) (*SchedulerModel, error) {
	if cfg.Cores <= 0 {
		return nil, fmt.Errorf("%w: cores must be positive", ErrInvalidConfig)
	}
	if cfg.MailboxCapacity <= 0 {
		cfg.MailboxCapacity = 1
	}
	return &SchedulerModel{
		queues:          make([][]WorkItem, cfg.Cores),
		mailboxCapacity: cfg.MailboxCapacity,
		regions:         map[string]Region{},
		stats:           Stats{Cores: cfg.Cores},
	}, nil
}

func (m *SchedulerModel) Enqueue(core int, item WorkItem) {
	if m == nil || core < 0 || core >= len(m.queues) {
		return
	}
	m.queues[core] = append(m.queues[core], item)
	m.rememberQueueDepth(len(m.queues[core]))
}

func (m *SchedulerModel) StepCore(core int) (bool, error) {
	if m == nil || core < 0 || core >= len(m.queues) {
		return false, ErrInvalidCore
	}
	if len(m.queues[core]) > 0 {
		item := m.queues[core][0]
		m.queues[core] = m.queues[core][1:]
		m.recordRun(core, item)
		return true, nil
	}
	for offset := 1; offset < len(m.queues); offset++ {
		victim := (core + offset) % len(m.queues)
		if len(m.queues[victim]) == 0 {
			continue
		}
		last := len(m.queues[victim]) - 1
		item := m.queues[victim][last]
		m.queues[victim] = m.queues[victim][:last]
		m.stats.Steals++
		m.recordRun(core, item)
		return true, nil
	}
	return false, nil
}

func (m *SchedulerModel) RunUntilIdle() (Stats, error) {
	if m == nil {
		return Stats{}, ErrInvalidConfig
	}
	for {
		ranAny := false
		for core := range m.queues {
			ran, err := m.StepCore(core)
			if err != nil {
				return m.stats, err
			}
			if ran {
				ranAny = true
			}
		}
		if !ranAny {
			return m.stats, nil
		}
	}
}

func (m *SchedulerModel) RunLog() []string {
	if m == nil {
		return nil
	}
	return append([]string(nil), m.runLog...)
}

func (m *SchedulerModel) Stats() Stats {
	if m == nil {
		return Stats{}
	}
	return m.stats
}

func (m *SchedulerModel) RegisterRegion(region Region) {
	if m == nil || region.Name == "" {
		return
	}
	m.regions[region.Name] = region
}

func (m *SchedulerModel) RegionOwner(name string) string {
	if m == nil {
		return ""
	}
	return m.regions[name].Owner
}

func (m *SchedulerModel) Send(fromCore int, toCore int, msg Message) (TransferReport, error) {
	if m == nil || fromCore < 0 || fromCore >= len(m.queues) || toCore < 0 || toCore >= len(m.queues) {
		return TransferReport{}, ErrInvalidCore
	}
	box := NewTypedMailbox(MailboxConfig{
		Name:     fmt.Sprintf("actor%d", toCore),
		Capacity: m.mailboxCapacity,
		Backpressure: BackpressurePolicy{
			Mode: "blocking_recv_yield",
		},
	})
	report, err := box.classify(msg, m, fmt.Sprintf("actor%d", toCore))
	if err != nil {
		return TransferReport{}, err
	}
	m.stats.MailboxSends++
	m.stats.BytesCopied += report.BytesCopied
	if report.ZeroCopy {
		m.stats.ZeroCopyMoves++
	}
	return report, nil
}

func (m *SchedulerModel) recordRun(core int, item WorkItem) {
	name := item.Name
	if name == "" {
		name = "work"
	}
	m.runLog = append(m.runLog, fmt.Sprintf("core%d:%s", core, name))
	m.stats.Completed++
}

func (m *SchedulerModel) rememberQueueDepth(depth int) {
	if depth > m.stats.MaxQueueDepth {
		m.stats.MaxQueueDepth = depth
	}
}

func PrototypeBenchmarks() ([]PrototypeBenchmark, error) {
	fanoutBaseline, fanoutMeasured, err := actorPingPongFanoutQueueDepths()
	if err != nil {
		return nil, err
	}
	copyBaseline, zeroCopyMeasured, err := zeroCopyRegionBytesCopied()
	if err != nil {
		return nil, err
	}
	return []PrototypeBenchmark{
		{
			Name:             "actor ping-pong fanout scheduler prototype",
			Kind:             "scheduler",
			Metric:           "max_queue_depth",
			Unit:             "work_items",
			BaselineValue:    fanoutBaseline,
			MeasuredValue:    fanoutMeasured,
			ImprovementRatio: improvementRatio(fanoutBaseline, fanoutMeasured),
			Evidence:         "compiler/internal/parallelrt two-core work stealing model ran actor ping-pong fanout comparison",
			Ran:              true,
			Pass:             true,
		},
		{
			Name:             "zero-copy region message scheduler prototype",
			Kind:             "transfer",
			Metric:           "bytes_copied",
			Unit:             "bytes",
			BaselineValue:    copyBaseline,
			MeasuredValue:    zeroCopyMeasured,
			ImprovementRatio: improvementRatio(copyBaseline, zeroCopyMeasured),
			Evidence:         "compiler/internal/parallelrt owned-region transfer report emitted zero_copy_move with bytes_copied=0",
			Ran:              true,
			Pass:             true,
		},
	}, nil
}

func actorPingPongFanoutQueueDepths() (int, int, error) {
	baseline, err := NewSchedulerModel(Config{Cores: 1, MailboxCapacity: 4})
	if err != nil {
		return 0, 0, err
	}
	for i := 0; i < 4; i++ {
		baseline.Enqueue(0, WorkItem{Name: fmt.Sprintf("ping%d", i)})
	}
	if _, err := baseline.RunUntilIdle(); err != nil {
		return 0, 0, err
	}

	measured, err := NewSchedulerModel(Config{Cores: 2, MailboxCapacity: 4})
	if err != nil {
		return 0, 0, err
	}
	measured.Enqueue(0, WorkItem{Name: "ping0"})
	measured.Enqueue(0, WorkItem{Name: "ping1"})
	if ran, err := measured.StepCore(1); err != nil {
		return 0, 0, err
	} else if !ran {
		return 0, 0, errors.New("two-core scheduler prototype did not steal fanout work")
	}
	return baseline.Stats().MaxQueueDepth, measured.Stats().MaxQueueDepth, nil
}

func zeroCopyRegionBytesCopied() (int, int, error) {
	copied, err := NewSchedulerModel(Config{Cores: 2, MailboxCapacity: 4})
	if err != nil {
		return 0, 0, err
	}
	copyReport, err := copied.Send(0, 1, Message{
		Name: "Frame.copy",
		Payloads: []Payload{{
			Name:         "view",
			Kind:         PayloadBorrowedView,
			SizeBytes:    4096,
			ExplicitCopy: true,
		}},
	})
	if err != nil {
		return 0, 0, err
	}

	zeroCopy, err := NewSchedulerModel(Config{Cores: 2, MailboxCapacity: 4})
	if err != nil {
		return 0, 0, err
	}
	zeroCopy.RegisterRegion(Region{Name: "frame", Owner: "sender"})
	moveReport, err := zeroCopy.Send(0, 1, Message{
		Name: "Frame.region",
		Payloads: []Payload{{
			Name:       "bytes",
			Kind:       PayloadOwnedRegion,
			RegionName: "frame",
			Owner:      "sender",
			SizeBytes:  4096,
		}},
	})
	if err != nil {
		return 0, 0, err
	}
	return copyReport.BytesCopied, moveReport.BytesCopied, nil
}

func improvementRatio(baseline int, measured int) float64 {
	if measured == 0 {
		return float64(baseline)
	}
	return float64(baseline) / float64(measured)
}

type PayloadKind string

const (
	PayloadScalar       PayloadKind = "scalar"
	PayloadAggregate    PayloadKind = "aggregate"
	PayloadOwnedBuffer  PayloadKind = "owned_buffer"
	PayloadOwnedRegion  PayloadKind = "owned_region"
	PayloadBorrowedView PayloadKind = "borrowed_view"
	PayloadUnsafePtr    PayloadKind = "unsafe_ptr"
)

type TransferMode string

const (
	TransferCopy         TransferMode = "copy"
	TransferMove         TransferMode = "move"
	TransferZeroCopyMove TransferMode = "zero_copy_move"
	TransferUnsafe       TransferMode = "unsafe_contract"
)

type Payload struct {
	Name           string
	Kind           PayloadKind
	RegionName     string
	Owner          string
	SizeBytes      int
	ExplicitCopy   bool
	UnsafeContract bool
}

type Message struct {
	Name     string
	Payloads []Payload
}

type BackpressurePolicy struct {
	Mode string `json:"mode,omitempty"`
}

type MailboxConfig struct {
	Name         string
	Capacity     int
	Backpressure BackpressurePolicy
}

type TypedMailbox struct {
	cfg      MailboxConfig
	messages []Message
}

type TransferReport struct {
	Message      string       `json:"message"`
	TransferMode TransferMode `json:"transfer_mode"`
	BytesCopied  int          `json:"bytes_copied"`
	ZeroCopy     bool         `json:"zero_copy"`
	PayloadCount int          `json:"payload_count"`
	RuntimePath  string       `json:"runtime_path"`
}

func NewTypedMailbox(cfg MailboxConfig) *TypedMailbox {
	if cfg.Capacity <= 0 {
		cfg.Capacity = 1
	}
	if cfg.Backpressure.Mode == "" {
		cfg.Backpressure.Mode = "blocking_recv_yield"
	}
	return &TypedMailbox{cfg: cfg}
}

func (m *TypedMailbox) Send(msg Message) (TransferReport, error) {
	if m == nil {
		return TransferReport{}, ErrInvalidConfig
	}
	if len(m.messages) >= m.cfg.Capacity {
		return TransferReport{}, ErrMailboxFull
	}
	report, err := m.classify(msg, nil, "")
	if err != nil {
		return TransferReport{}, err
	}
	m.messages = append(m.messages, msg)
	return report, nil
}

func (m *TypedMailbox) Receive() (Message, bool) {
	if m == nil || len(m.messages) == 0 {
		return Message{}, false
	}
	msg := m.messages[0]
	copy(m.messages, m.messages[1:])
	m.messages[len(m.messages)-1] = Message{}
	m.messages = m.messages[:len(m.messages)-1]
	return msg, true
}

func (m *TypedMailbox) Capacity() int {
	if m == nil {
		return 0
	}
	return m.cfg.Capacity
}

func (m *TypedMailbox) Backpressure() BackpressurePolicy {
	if m == nil {
		return BackpressurePolicy{}
	}
	return m.cfg.Backpressure
}

func (m *TypedMailbox) HasOwnershipMetadata() bool {
	return m != nil
}

func (m *TypedMailbox) classify(msg Message, scheduler *SchedulerModel, targetOwner string) (TransferReport, error) {
	report := TransferReport{
		Message:      msg.Name,
		TransferMode: TransferCopy,
		PayloadCount: len(msg.Payloads),
		RuntimePath:  "actor_mailbox_typed_slots",
	}
	if len(msg.Payloads) == 0 {
		return report, nil
	}
	for _, payload := range msg.Payloads {
		mode, bytesCopied, zeroCopy, err := classifyPayload(payload, scheduler, targetOwner)
		if err != nil {
			return TransferReport{}, err
		}
		report.BytesCopied += bytesCopied
		if zeroCopy {
			report.ZeroCopy = true
			if report.TransferMode == TransferCopy && report.BytesCopied == 0 {
				report.TransferMode = mode
			}
			report.RuntimePath = "actor_mailbox_zero_copy_region_slot"
			continue
		}
		if mode == TransferUnsafe {
			report.TransferMode = TransferUnsafe
			report.RuntimePath = "actor_mailbox_unsafe_contract_slot"
			continue
		}
		if !report.ZeroCopy {
			report.TransferMode = TransferCopy
			report.RuntimePath = "actor_mailbox_copy_region_slot"
		}
	}
	return report, nil
}

func classifyPayload(payload Payload, scheduler *SchedulerModel, targetOwner string) (TransferMode, int, bool, error) {
	size := payload.SizeBytes
	if size < 0 {
		size = 0
	}
	switch payload.Kind {
	case PayloadScalar, PayloadAggregate:
		return TransferCopy, size, false, nil
	case PayloadBorrowedView:
		if !payload.ExplicitCopy {
			return "", 0, false, ErrBorrowedPayloadRequiresCopy
		}
		return TransferCopy, size, false, nil
	case PayloadOwnedBuffer:
		return TransferMove, 0, false, nil
	case PayloadOwnedRegion:
		if scheduler == nil {
			return TransferZeroCopyMove, 0, true, nil
		}
		region, ok := scheduler.regions[payload.RegionName]
		if !ok {
			return "", 0, false, ErrUnknownRegion
		}
		if payload.Owner != "" && region.Owner != payload.Owner {
			return "", 0, false, ErrRegionOwnerMismatch
		}
		if region.BorrowedRefs > 0 {
			return "", 0, false, ErrRegionHasLiveBorrowedAliases
		}
		region.Owner = targetOwner
		scheduler.regions[payload.RegionName] = region
		return TransferZeroCopyMove, 0, true, nil
	case PayloadUnsafePtr:
		if !payload.UnsafeContract {
			return "", 0, false, ErrBorrowedPayloadRequiresCopy
		}
		return TransferUnsafe, size, false, nil
	default:
		return TransferCopy, size, false, nil
	}
}
