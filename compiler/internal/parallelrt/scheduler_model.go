package parallelrt

import (
	"errors"
	"fmt"
	"strings"

	"tetra_language/compiler/internal/runtimeabi"
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
	Name               string   `json:"name"`
	Kind               string   `json:"kind"`
	Metric             string   `json:"metric"`
	Unit               string   `json:"unit"`
	BaselineValue      int      `json:"baseline_value"`
	MeasuredValue      int      `json:"measured_value"`
	ImprovementRatio   float64  `json:"improvement_ratio"`
	Evidence           string   `json:"evidence"`
	ClaimTier          string   `json:"claim_tier"`
	Claim              string   `json:"claim"`
	RawOutputArtifacts []string `json:"raw_output_artifacts"`
	Ran                bool     `json:"ran"`
	Pass               bool     `json:"pass"`
}

type PrototypeEvidence struct {
	Schema             string                    `json:"schema"`
	Benchmarks         []PrototypeBenchmark      `json:"benchmarks"`
	ActorMemoryDomains []ActorMemoryDomainReport `json:"actor_memory_domains"`
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
	if m == nil || fromCore < 0 || fromCore >= len(m.queues) || toCore < 0 ||
		toCore >= len(m.queues) {
		return TransferReport{}, ErrInvalidCore
	}
	box := NewTypedMailbox(MailboxConfig{
		Name:     fmt.Sprintf("actor%d", toCore),
		Capacity: m.mailboxCapacity,
		Backpressure: BackpressurePolicy{
			Mode: "blocking_recv_yield",
		},
	})
	domainMoves := m.domainMovesForMessage(msg, fmt.Sprintf("actor%d", toCore))
	report, err := box.classify(msg, m, fmt.Sprintf("actor%d", toCore))
	if err != nil {
		return TransferReport{}, err
	}
	if report.ZeroCopy {
		report.DomainMoves = domainMoves
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

func (m *SchedulerModel) domainMovesForMessage(
	msg Message,
	targetOwner string,
) []DomainOwnershipMove {
	if m == nil {
		return nil
	}
	var moves []DomainOwnershipMove
	for _, payload := range msg.Payloads {
		if payload.Kind != PayloadOwnedRegion {
			continue
		}
		region, ok := m.regions[payload.RegionName]
		if !ok {
			continue
		}
		owner := region.Owner
		if owner == "" {
			owner = payload.Owner
		}
		moves = append(moves, DomainOwnershipMove{
			RegionName:   payload.RegionName,
			FromDomainID: actorDomainID(owner),
			ToDomainID:   actorDomainID(targetOwner),
			BytesMoved:   positiveBytes(payload.SizeBytes),
			TransferMode: TransferZeroCopyMove,
		})
	}
	return moves
}

func PrototypeBenchmarks() ([]PrototypeBenchmark, error) {
	if _, _, err := actorPingPongFanoutQueueDepths(); err != nil {
		return nil, err
	}
	if _, _, err := zeroCopyRegionBytesCopied(); err != nil {
		return nil, err
	}
	rawArtifact := "reports/actor-runtime-foundation/P15/parallelrt-evidence.raw.json"
	return []PrototypeBenchmark{
		{
			Name:   "actor ping-pong benchmark prep",
			Kind:   "actor_benchmark_prep",
			Metric: "messages_round_trip",
			Unit:   "prep_only",
			Evidence: ("compiler/compiler_suite_test.go::TestActorsPingPongBuildAndRun " +
				"and examples/actors/actors_pingpong.tetra define the local Linux-x64 " +
				"actor ping-pong workload candidate"),
			ClaimTier: "tier0_local_smoke_only",
			Claim: ("Actor ping-pong benchmark prep row exists as Tier 0 local smoke " +
				"only; no measured result is published and cross-runtime comparison is " +
				"out of scope."),
			RawOutputArtifacts: []string{rawArtifact},
			Ran:                false,
			Pass:               true,
		},
		{
			Name:   "actor fanout/fanin benchmark prep",
			Kind:   "actor_benchmark_prep",
			Metric: "fanout_fanin_messages",
			Unit:   "prep_only",
			Evidence: ("compiler/internal/parallelrt two-core work stealing model " +
				"checks actor fanout/fanin scheduling shape without publishing throughput"),
			ClaimTier: "tier0_local_smoke_only",
			Claim: ("Actor fanout/fanin benchmark prep row exists as Tier 0 local " +
				"smoke only; it records local workload shape and leaves public benchmark " +
				"publication out of scope."),
			RawOutputArtifacts: []string{rawArtifact},
			Ran:                false,
			Pass:               true,
		},
		{
			Name:   "actor mailbox throughput benchmark prep",
			Kind:   "actor_benchmark_prep",
			Metric: "mailbox_messages",
			Unit:   "prep_only",
			Evidence: ("compiler/internal/parallelrt TypedMailbox and parallel " +
				"production actor mailbox cases define the local mailbox throughput " +
				"workload candidate"),
			ClaimTier: "tier0_local_smoke_only",
			Claim: ("Actor mailbox throughput benchmark prep row exists as Tier 0 " +
				"local smoke only; it publishes no measured result and no throughput " +
				"guarantee."),
			RawOutputArtifacts: []string{rawArtifact},
			Ran:                false,
			Pass:               true,
		},
		{
			Name:   "actor backpressure latency benchmark prep",
			Kind:   "actor_benchmark_prep",
			Metric: "backpressure_wait",
			Unit:   "prep_only",
			Evidence: ("compiler/internal/parallelrt ErrMailboxFull and blocking_recv_" +
				"yield metadata define the local backpressure latency diagnostic " +
				"candidate"),
			ClaimTier: "tier0_local_smoke_only",
			Claim: ("Actor backpressure latency benchmark prep row exists as Tier 0 " +
				"local smoke only; no real-world SLA is claimed."),
			RawOutputArtifacts: []string{rawArtifact},
			Ran:                false,
			Pass:               true,
		},
		{
			Name:   "zero_copy_move local typed mailbox benchmark prep",
			Kind:   "actor_transfer_prep",
			Metric: "owned_region_transfer",
			Unit:   "prep_only",
			Evidence: ("compiler/internal/parallelrt owned-region transfer report emits " +
				"zero_copy_move for local typed mailbox metadata only"),
			ClaimTier: "tier0_local_smoke_only",
			Claim: ("zero_copy_move local typed mailbox benchmark prep row exists as " +
				"Tier 0 local smoke only; it records local owned-region metadata and " +
				"leaves distributed or network transfer behavior out of scope."),
			RawOutputArtifacts: []string{rawArtifact},
			Ran:                false,
			Pass:               true,
		},
	}, nil
}

func CollectPrototypeEvidence() (PrototypeEvidence, error) {
	benchmarks, err := PrototypeBenchmarks()
	if err != nil {
		return PrototypeEvidence{}, err
	}
	domains, err := PrototypeActorMemoryDomains()
	if err != nil {
		return PrototypeEvidence{}, err
	}
	return PrototypeEvidence{
		Schema:             "tetra.parallelrt.prototype-evidence.v1",
		Benchmarks:         benchmarks,
		ActorMemoryDomains: domains,
	}, nil
}

func PrototypeActorMemoryDomains() ([]ActorMemoryDomainReport, error) {
	copyBox := NewTypedMailbox(MailboxConfig{
		Name:         "actor-mailbox-copy",
		Capacity:     4,
		ByteCapacity: 256,
		MessageBytes: 16,
		Backpressure: BackpressurePolicy{
			Mode: "blocking_recv_yield",
		},
	})
	if _, err := copyBox.Send(Message{
		Name: "Reading",
		Payloads: []Payload{{
			Name:      "value",
			Kind:      PayloadScalar,
			SizeBytes: 32,
		}},
	}); err != nil {
		return nil, err
	}

	regionBox := NewTypedMailbox(MailboxConfig{
		Name:         "actor-frame",
		Capacity:     2,
		ByteCapacity: 512,
		MessageBytes: 16,
		Backpressure: BackpressurePolicy{
			Mode: "blocking_recv_yield",
		},
	})
	if _, err := regionBox.Send(Message{
		Name: "Frame.region",
		Payloads: []Payload{{
			Name:       "bytes",
			Kind:       PayloadOwnedRegion,
			RegionName: "frame",
			Owner:      "sender",
			SizeBytes:  256,
		}},
	}); err != nil {
		return nil, err
	}
	if _, err := regionBox.Send(Message{
		Name: "Frame.region-too-large",
		Payloads: []Payload{{
			Name:       "bytes",
			Kind:       PayloadOwnedRegion,
			RegionName: "frame-large",
			Owner:      "sender",
			SizeBytes:  4096,
		}},
	}); !errors.Is(err, ErrMailboxFull) {
		return nil, fmt.Errorf(
			"prototype actor memory domain: large owned region send error = %v, want ErrMailboxFull",
			err,
		)
	}

	reports := []ActorMemoryDomainReport{
		copyBox.MemoryDomainReport(),
		regionBox.MemoryDomainReport(),
	}
	for _, report := range reports {
		if err := ValidateActorMemoryDomainReport(report); err != nil {
			return nil, err
		}
	}
	return reports, nil
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
	ByteCapacity int
	MessageBytes int
	Backpressure BackpressurePolicy
}

type TypedMailbox struct {
	cfg                 MailboxConfig
	messages            []queuedMailboxMessage
	queuedBytes         int
	peakQueuedBytes     int
	reclaimedBytes      int
	requestedBytes      int
	bytesCopied         int
	copyCount           int
	bytesMoved          int
	lastBackpressure    string
	lastBackpressureErr error
}

type TransferReport struct {
	Message      string                `json:"message"`
	TransferMode TransferMode          `json:"transfer_mode"`
	BytesCopied  int                   `json:"bytes_copied"`
	BytesMoved   int                   `json:"bytes_moved,omitempty"`
	ZeroCopy     bool                  `json:"zero_copy"`
	PayloadCount int                   `json:"payload_count"`
	RuntimePath  string                `json:"runtime_path"`
	DomainMoves  []DomainOwnershipMove `json:"domain_moves,omitempty"`
}

type DomainOwnershipMove struct {
	RegionName   string       `json:"region_name"`
	FromDomainID string       `json:"from_domain_id"`
	ToDomainID   string       `json:"to_domain_id"`
	BytesMoved   int          `json:"bytes_moved"`
	TransferMode TransferMode `json:"transfer_mode"`
}

type queuedMailboxMessage struct {
	message Message
	bytes   int
}

const ActorMemoryDomainSchemaV1 = "tetra.actors.memory-domain.v1"

type ActorMemoryDomainReport struct {
	SchemaVersion              string                   `json:"schema_version"`
	ActorID                    string                   `json:"actor_id"`
	EvidenceClass              string                   `json:"evidence_class"`
	EvidenceMethod             string                   `json:"evidence_method"`
	RuntimeMeasured            bool                     `json:"runtime_measured"`
	RuntimeBlockedReason       string                   `json:"runtime_blocked_reason,omitempty"`
	Domain                     runtimeabi.MemoryDomain  `json:"domain"`
	Mailbox                    ActorMailboxMemoryReport `json:"mailbox"`
	MessagePool                ActorMessagePoolReport   `json:"message_pool"`
	OwnedRegions               []ActorOwnedRegionReport `json:"owned_regions,omitempty"`
	Backpressure               ActorBackpressureReport  `json:"backpressure"`
	NonClaims                  []string                 `json:"non_claims"`
	ProductionRuntimeClaimed   bool                     `json:"production_runtime_claimed"`
	DistributedZeroCopyClaimed bool                     `json:"distributed_zero_copy_claimed"`
}

type ActorMailboxMemoryReport struct {
	CapacityMessages int    `json:"capacity_messages"`
	QueuedMessages   int    `json:"queued_messages"`
	CapacityBytes    int    `json:"capacity_bytes"`
	QueuedBytes      int    `json:"queued_bytes"`
	PeakQueuedBytes  int    `json:"peak_queued_bytes"`
	ReclaimedBytes   int    `json:"reclaimed_bytes"`
	MessageBytes     int    `json:"message_bytes"`
	BackpressureMode string `json:"backpressure_mode"`
}

type ActorMessagePoolReport struct {
	SlabBytes         int `json:"slab_bytes"`
	LiveBytes         int `json:"live_bytes"`
	ReclaimedBytes    int `json:"reclaimed_bytes"`
	CapacityBytes     int `json:"capacity_bytes"`
	MessageSlotsLive  int `json:"message_slots_live"`
	MessageSlotsLimit int `json:"message_slots_limit"`
}

type ActorOwnedRegionReport struct {
	RegionName string `json:"region_name"`
	DomainID   string `json:"domain_id"`
	OwnerID    string `json:"owner_id"`
	Bytes      int    `json:"bytes"`
}

type ActorBackpressureReport struct {
	Mode   string `json:"mode"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func NewTypedMailbox(cfg MailboxConfig) *TypedMailbox {
	if cfg.Capacity <= 0 {
		cfg.Capacity = 1
	}
	if cfg.MessageBytes <= 0 {
		cfg.MessageBytes = 16
	}
	if cfg.Backpressure.Mode == "" {
		cfg.Backpressure.Mode = "blocking_recv_yield"
	}
	if cfg.Name == "" {
		cfg.Name = "actor"
	}
	return &TypedMailbox{cfg: cfg, lastBackpressure: "available"}
}

func (m *TypedMailbox) Send(msg Message) (TransferReport, error) {
	if m == nil {
		return TransferReport{}, ErrInvalidConfig
	}
	if len(m.messages) >= m.cfg.Capacity {
		m.lastBackpressure = "message_limit_reached"
		m.lastBackpressureErr = ErrMailboxFull
		return TransferReport{}, ErrMailboxFull
	}
	report, err := m.classify(msg, nil, "")
	if err != nil {
		return TransferReport{}, err
	}
	messageBytes := m.messageFootprintBytes(report)
	if m.cfg.ByteCapacity > 0 && m.queuedBytes+messageBytes > m.cfg.ByteCapacity {
		m.lastBackpressure = "byte_limit_reached"
		m.lastBackpressureErr = ErrMailboxFull
		return TransferReport{}, ErrMailboxFull
	}
	m.messages = append(m.messages, queuedMailboxMessage{message: msg, bytes: messageBytes})
	m.requestedBytes += messageBytes
	m.bytesMoved += report.BytesMoved
	if report.BytesCopied > 0 {
		m.bytesCopied += report.BytesCopied
		m.copyCount++
	}
	m.queuedBytes += messageBytes
	if m.queuedBytes > m.peakQueuedBytes {
		m.peakQueuedBytes = m.queuedBytes
	}
	m.lastBackpressure = "available"
	m.lastBackpressureErr = nil
	return report, nil
}

func (m *TypedMailbox) Receive() (Message, bool) {
	if m == nil || len(m.messages) == 0 {
		return Message{}, false
	}
	entry := m.messages[0]
	copy(m.messages, m.messages[1:])
	m.messages[len(m.messages)-1] = queuedMailboxMessage{}
	m.messages = m.messages[:len(m.messages)-1]
	m.queuedBytes -= entry.bytes
	if m.queuedBytes < 0 {
		m.queuedBytes = 0
	}
	m.reclaimedBytes += entry.bytes
	if m.lastBackpressureErr == ErrMailboxFull && m.queuedBytes < m.byteCapacity() {
		m.lastBackpressure = "available"
		m.lastBackpressureErr = nil
	}
	return entry.message, true
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

func (m *TypedMailbox) QueuedBytes() int {
	if m == nil {
		return 0
	}
	return m.queuedBytes
}

func (m *TypedMailbox) ReclaimedBytes() int {
	if m == nil {
		return 0
	}
	return m.reclaimedBytes
}

func (m *TypedMailbox) MemoryDomainReport() ActorMemoryDomainReport {
	if m == nil {
		return ActorMemoryDomainReport{SchemaVersion: ActorMemoryDomainSchemaV1}
	}
	actorID := actorIDFromName(m.cfg.Name)
	capacityBytes := m.byteCapacity()
	ownedRegions := m.ownedRegionReports(actorID)
	domain := runtimeabi.MemoryDomain{
		DomainID:       actorDomainID(actorID),
		Kind:           runtimeabi.DomainActor,
		OwnerKind:      "actor",
		OwnerID:        actorID,
		Lifetime:       "actor:" + actorID,
		BudgetBytes:    int64(capacityBytes),
		RequestedBytes: int64(m.requestedBytes),
		ReservedBytes:  int64(capacityBytes),
		CommittedBytes: int64(capacityBytes),
		ReleasedBytes:  int64(m.reclaimedBytes),
		CurrentBytes:   int64(m.queuedBytes),
		PeakBytes:      int64(m.peakQueuedBytes),
		CopyCount:      m.copyCount,
		BytesCopied:    int64(m.bytesCopied),
	}
	return ActorMemoryDomainReport{
		SchemaVersion:   ActorMemoryDomainSchemaV1,
		ActorID:         actorID,
		EvidenceClass:   "local_parallelrt_model",
		EvidenceMethod:  "parallelrt_typed_mailbox_memory_domain_v1",
		RuntimeMeasured: false,
		RuntimeBlockedReason: ("production actor runtime per-actor byte sampler is not " +
			"implemented; this is local parallelrt model evidence"),
		Domain: domain,
		Mailbox: ActorMailboxMemoryReport{
			CapacityMessages: m.cfg.Capacity,
			QueuedMessages:   len(m.messages),
			CapacityBytes:    capacityBytes,
			QueuedBytes:      m.queuedBytes,
			PeakQueuedBytes:  m.peakQueuedBytes,
			ReclaimedBytes:   m.reclaimedBytes,
			MessageBytes:     m.cfg.MessageBytes,
			BackpressureMode: m.cfg.Backpressure.Mode,
		},
		MessagePool: ActorMessagePoolReport{
			SlabBytes:         m.cfg.MessageBytes * m.cfg.Capacity,
			LiveBytes:         m.queuedBytes,
			ReclaimedBytes:    m.reclaimedBytes,
			CapacityBytes:     capacityBytes,
			MessageSlotsLive:  len(m.messages),
			MessageSlotsLimit: m.cfg.Capacity,
		},
		OwnedRegions: ownedRegions,
		Backpressure: ActorBackpressureReport{
			Mode:   m.cfg.Backpressure.Mode,
			Status: m.lastBackpressure,
			Reason: backpressureReason(m.lastBackpressure),
		},
		NonClaims: []string{
			"full production actor runtime is not claimed",
			"distributed actor zero-copy is not claimed",
			"actor memory domain bytes are model/report evidence unless paired with runtime measurement",
		},
		ProductionRuntimeClaimed:   false,
		DistributedZeroCopyClaimed: false,
	}
}

func ValidateActorMemoryDomainReport(report ActorMemoryDomainReport) error {
	if report.SchemaVersion != ActorMemoryDomainSchemaV1 {
		return fmt.Errorf("actor memory domain report: schema = %q", report.SchemaVersion)
	}
	if strings.TrimSpace(report.ActorID) == "" {
		return fmt.Errorf("actor memory domain report: actor_id is required")
	}
	if strings.TrimSpace(report.EvidenceClass) == "" {
		return fmt.Errorf("actor memory domain report: evidence_class is required")
	}
	if strings.TrimSpace(report.EvidenceMethod) == "" {
		return fmt.Errorf("actor memory domain report: evidence_method is required")
	}
	if !report.RuntimeMeasured && strings.TrimSpace(report.RuntimeBlockedReason) == "" {
		return fmt.Errorf(
			"actor memory domain report: runtime_blocked_reason is required when runtime_measured=false",
		)
	}
	if report.ProductionRuntimeClaimed {
		return fmt.Errorf("actor memory domain report: production runtime claim is forbidden")
	}
	if report.DistributedZeroCopyClaimed {
		return fmt.Errorf("actor memory domain report: distributed zero-copy claim is forbidden")
	}
	if err := runtimeabi.ValidateMemoryDomain(report.Domain); err != nil {
		return fmt.Errorf("actor memory domain report: %w", err)
	}
	if report.Domain.Kind != runtimeabi.DomainActor {
		return fmt.Errorf(
			"actor memory domain report: domain kind = %q, want actor",
			report.Domain.Kind,
		)
	}
	if report.Mailbox.CapacityMessages <= 0 {
		return fmt.Errorf("actor memory domain report: mailbox capacity_messages must be positive")
	}
	if report.Mailbox.QueuedMessages < 0 || report.Mailbox.QueuedBytes < 0 ||
		report.Mailbox.ReclaimedBytes < 0 {
		return fmt.Errorf(
			"actor memory domain report: mailbox byte/message counts must not be negative",
		)
	}
	if report.Mailbox.CapacityBytes > 0 &&
		report.Mailbox.QueuedBytes > report.Mailbox.CapacityBytes {
		return fmt.Errorf("actor memory domain report: queued bytes exceed capacity")
	}
	if report.Mailbox.PeakQueuedBytes < report.Mailbox.QueuedBytes {
		return fmt.Errorf("actor memory domain report: peak queued bytes must be >= queued bytes")
	}
	if report.MessagePool.LiveBytes != report.Mailbox.QueuedBytes {
		return fmt.Errorf(
			"actor memory domain report: message pool live bytes = %d, want mailbox queued bytes %d",
			report.MessagePool.LiveBytes,
			report.Mailbox.QueuedBytes,
		)
	}
	if report.MessagePool.ReclaimedBytes != report.Mailbox.ReclaimedBytes {
		return fmt.Errorf(
			"actor memory domain report: message pool reclaimed bytes = %d, want mailbox reclaimed bytes %d",
			report.MessagePool.ReclaimedBytes,
			report.Mailbox.ReclaimedBytes,
		)
	}
	if report.MessagePool.CapacityBytes != report.Mailbox.CapacityBytes {
		return fmt.Errorf(
			"actor memory domain report: message pool capacity bytes = %d, want mailbox capacity bytes %d",
			report.MessagePool.CapacityBytes,
			report.Mailbox.CapacityBytes,
		)
	}
	ownedRegionBytes := sumOwnedRegionBytes(report.OwnedRegions)
	if ownedRegionBytes > report.Mailbox.QueuedBytes {
		return fmt.Errorf(
			"actor memory domain report: owned region bytes = %d, want <= queued bytes %d",
			ownedRegionBytes,
			report.Mailbox.QueuedBytes,
		)
	}
	if report.Domain.CurrentBytes != int64(report.Mailbox.QueuedBytes) {
		return fmt.Errorf(
			"actor memory domain report: domain current bytes = %d, want queued bytes %d",
			report.Domain.CurrentBytes,
			report.Mailbox.QueuedBytes,
		)
	}
	if report.Domain.CommittedBytes != int64(report.Mailbox.CapacityBytes) {
		return fmt.Errorf(
			"actor memory domain report: committed bytes = %d, want mailbox capacity bytes %d",
			report.Domain.CommittedBytes,
			report.Mailbox.CapacityBytes,
		)
	}
	if report.Domain.ReleasedBytes != int64(report.Mailbox.ReclaimedBytes) {
		return fmt.Errorf(
			"actor memory domain report: released bytes = %d, want reclaimed bytes %d",
			report.Domain.ReleasedBytes,
			report.Mailbox.ReclaimedBytes,
		)
	}
	for _, owned := range report.OwnedRegions {
		if strings.TrimSpace(owned.RegionName) == "" {
			return fmt.Errorf("actor memory domain report: owned region name is required")
		}
		if owned.DomainID != report.Domain.DomainID {
			return fmt.Errorf(
				"actor memory domain report: owned region %s domain_id = %q, want %q",
				owned.RegionName,
				owned.DomainID,
				report.Domain.DomainID,
			)
		}
		if owned.OwnerID != report.ActorID {
			return fmt.Errorf(
				"actor memory domain report: owned region %s owner_id = %q, want actor %q",
				owned.RegionName,
				owned.OwnerID,
				report.ActorID,
			)
		}
		if owned.Bytes <= 0 {
			return fmt.Errorf(
				"actor memory domain report: owned region %s bytes must be positive",
				owned.RegionName,
			)
		}
	}
	switch report.Backpressure.Status {
	case "available", "message_limit_reached", "byte_limit_reached":
	default:
		return fmt.Errorf(
			"actor memory domain report: unknown backpressure status %q",
			report.Backpressure.Status,
		)
	}
	if !containsText(report.NonClaims, "full production actor runtime is not claimed") {
		return fmt.Errorf("actor memory domain report: missing production runtime nonclaim")
	}
	if !containsText(report.NonClaims, "distributed actor zero-copy is not claimed") {
		return fmt.Errorf("actor memory domain report: missing distributed zero-copy nonclaim")
	}
	return nil
}

func (m *TypedMailbox) messageFootprintBytes(report TransferReport) int {
	if m == nil {
		return 0
	}
	bytes := m.cfg.MessageBytes + report.BytesCopied
	if report.BytesMoved > 0 {
		bytes += report.BytesMoved
	}
	if bytes < m.cfg.MessageBytes {
		return m.cfg.MessageBytes
	}
	return bytes
}

func (m *TypedMailbox) byteCapacity() int {
	if m == nil {
		return 0
	}
	if m.cfg.ByteCapacity > 0 {
		return m.cfg.ByteCapacity
	}
	return m.cfg.Capacity * m.cfg.MessageBytes
}

func (m *TypedMailbox) classify(
	msg Message,
	scheduler *SchedulerModel,
	targetOwner string,
) (TransferReport, error) {
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
		if payload.Kind == PayloadOwnedRegion && zeroCopy {
			report.BytesMoved += positiveBytes(payload.SizeBytes)
		}
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

func classifyPayload(
	payload Payload,
	scheduler *SchedulerModel,
	targetOwner string,
) (TransferMode, int, bool, error) {
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

func (m *TypedMailbox) ownedRegionReports(actorID string) []ActorOwnedRegionReport {
	if m == nil {
		return nil
	}
	var regions []ActorOwnedRegionReport
	domainID := actorDomainID(actorID)
	for _, entry := range m.messages {
		for _, payload := range entry.message.Payloads {
			if payload.Kind != PayloadOwnedRegion {
				continue
			}
			bytes := positiveBytes(payload.SizeBytes)
			if bytes == 0 {
				continue
			}
			regions = append(regions, ActorOwnedRegionReport{
				RegionName: payload.RegionName,
				DomainID:   domainID,
				OwnerID:    actorID,
				Bytes:      bytes,
			})
		}
	}
	return regions
}

func sumOwnedRegionBytes(regions []ActorOwnedRegionReport) int {
	total := 0
	for _, region := range regions {
		total += positiveBytes(region.Bytes)
	}
	return total
}

func actorIDFromName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "domain:")
	name = strings.TrimPrefix(name, "actor:")
	if name == "" {
		return "actor"
	}
	return cleanActorDomainPart(name)
}

func actorDomainID(actorID string) string {
	return "domain:actor:" + actorIDFromName(actorID)
}

func positiveBytes(bytes int) int {
	if bytes < 0 {
		return 0
	}
	return bytes
}

func cleanActorDomainPart(raw string) string {
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) == 0 {
		return "actor"
	}
	return strings.Join(fields, "_")
}

func backpressureReason(status string) string {
	switch status {
	case "message_limit_reached":
		return "mailbox message capacity reached"
	case "byte_limit_reached":
		return "mailbox byte capacity reached"
	default:
		return ""
	}
}

func containsText(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
