package runtimeabi

import (
	"fmt"
	"strings"
)

type MemoryDomainKind string

const (
	DomainProcess  MemoryDomainKind = "process"
	DomainTask     MemoryDomainKind = "task"
	DomainActor    MemoryDomainKind = "actor"
	DomainIsland   MemoryDomainKind = "island"
	DomainRequest  MemoryDomainKind = "request"
	DomainExternal MemoryDomainKind = "external"
)

type MemoryDomain struct {
	DomainID       string           `json:"domain_id"`
	ParentDomainID string           `json:"parent_domain_id,omitempty"`
	Kind           MemoryDomainKind `json:"kind"`
	OwnerKind      string           `json:"owner_kind"`
	OwnerID        string           `json:"owner_id"`
	Lifetime       string           `json:"lifetime"`
	BudgetBytes    int64            `json:"budget_bytes,omitempty"`
	RequestedBytes int64            `json:"requested_bytes,omitempty"`
	ReservedBytes  int64            `json:"reserved_bytes,omitempty"`
	CommittedBytes int64            `json:"committed_bytes,omitempty"`
	ReleasedBytes  int64            `json:"released_bytes,omitempty"`
	CurrentBytes   int64            `json:"current_bytes,omitempty"`
	PeakBytes      int64            `json:"peak_bytes,omitempty"`
	CopyCount      int              `json:"copy_count,omitempty"`
	BytesCopied    int64            `json:"bytes_copied,omitempty"`
}

type MemoryDomainSummary struct {
	DomainID       string           `json:"domain_id"`
	ParentDomainID string           `json:"parent_domain_id,omitempty"`
	Kind           MemoryDomainKind `json:"kind"`
	OwnerKind      string           `json:"owner_kind"`
	OwnerID        string           `json:"owner_id"`
	Lifetime       string           `json:"lifetime"`
	RowCount       int              `json:"row_count"`
	BudgetBytes    int64            `json:"budget_bytes,omitempty"`
	RequestedBytes int64            `json:"requested_bytes,omitempty"`
	ReservedBytes  int64            `json:"reserved_bytes,omitempty"`
	CommittedBytes int64            `json:"committed_bytes,omitempty"`
	ReleasedBytes  int64            `json:"released_bytes,omitempty"`
	CurrentBytes   int64            `json:"current_bytes,omitempty"`
	PeakBytes      int64            `json:"peak_bytes,omitempty"`
	CopyCount      int              `json:"copy_count,omitempty"`
	BytesCopied    int64            `json:"bytes_copied,omitempty"`
}

func DefaultProcessMemoryDomain(requested int64, reserved int64) MemoryDomain {
	return MemoryDomain{
		DomainID:       "domain:process",
		Kind:           DomainProcess,
		OwnerKind:      "process",
		OwnerID:        "current",
		Lifetime:       "process",
		BudgetBytes:    requested,
		RequestedBytes: requested,
		ReservedBytes:  reserved,
	}
}

func IslandMemoryDomain(regionID string, lifetime string, requested int64, reserved int64) MemoryDomain {
	owner := cleanDomainOwner(regionID, "island")
	return MemoryDomain{
		DomainID:       domainID(DomainIsland, regionID, owner),
		Kind:           DomainIsland,
		OwnerKind:      "island",
		OwnerID:        owner,
		Lifetime:       defaultDomainString(strings.TrimSpace(lifetime), "island"),
		BudgetBytes:    requested,
		RequestedBytes: requested,
		ReservedBytes:  reserved,
	}
}

func ExternalMemoryDomain(ownerID string, lifetime string, requested int64, reserved int64) MemoryDomain {
	owner := cleanDomainOwner(ownerID, "external")
	return MemoryDomain{
		DomainID:       domainID(DomainExternal, ownerID, owner),
		Kind:           DomainExternal,
		OwnerKind:      "external",
		OwnerID:        owner,
		Lifetime:       defaultDomainString(strings.TrimSpace(lifetime), "external"),
		BudgetBytes:    requested,
		RequestedBytes: requested,
		ReservedBytes:  reserved,
	}
}

func AggregateMemoryDomainSummary(domains []MemoryDomain) []MemoryDomainSummary {
	if len(domains) == 0 {
		return nil
	}
	byKey := map[string]MemoryDomainSummary{}
	for _, domain := range domains {
		if strings.TrimSpace(domain.DomainID) == "" {
			continue
		}
		key := memoryDomainSummaryKey(domain)
		summary := byKey[key]
		if summary.DomainID == "" {
			summary.DomainID = domain.DomainID
			summary.ParentDomainID = domain.ParentDomainID
			summary.Kind = domain.Kind
			summary.OwnerKind = domain.OwnerKind
			summary.OwnerID = domain.OwnerID
			summary.Lifetime = domain.Lifetime
		}
		summary.RowCount++
		summary.BudgetBytes += domain.BudgetBytes
		summary.RequestedBytes += domain.RequestedBytes
		summary.ReservedBytes += domain.ReservedBytes
		summary.CommittedBytes += domain.CommittedBytes
		summary.ReleasedBytes += domain.ReleasedBytes
		summary.CurrentBytes += domain.CurrentBytes
		if domain.PeakBytes > summary.PeakBytes {
			summary.PeakBytes = domain.PeakBytes
		}
		summary.CopyCount += domain.CopyCount
		summary.BytesCopied += domain.BytesCopied
		byKey[key] = summary
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sortStrings(keys)
	out := make([]MemoryDomainSummary, 0, len(keys))
	for _, key := range keys {
		out = append(out, byKey[key])
	}
	return out
}

func ValidateMemoryDomain(domain MemoryDomain) error {
	if strings.TrimSpace(domain.DomainID) == "" {
		return fmt.Errorf("memory domain: domain_id is required")
	}
	if !KnownMemoryDomainKind(domain.Kind) {
		return fmt.Errorf("memory domain %s: unknown domain kind %q", domain.DomainID, domain.Kind)
	}
	if strings.TrimSpace(domain.OwnerKind) == "" {
		return fmt.Errorf("memory domain %s: owner_kind is required", domain.DomainID)
	}
	if strings.TrimSpace(domain.OwnerID) == "" {
		return fmt.Errorf("memory domain %s: owner_id is required", domain.DomainID)
	}
	if strings.TrimSpace(domain.Lifetime) == "" {
		return fmt.Errorf("memory domain %s: lifetime is required", domain.DomainID)
	}
	if domain.BudgetBytes < 0 || domain.RequestedBytes < 0 || domain.ReservedBytes < 0 ||
		domain.CommittedBytes < 0 || domain.ReleasedBytes < 0 || domain.CurrentBytes < 0 ||
		domain.PeakBytes < 0 || domain.BytesCopied < 0 {
		return fmt.Errorf("memory domain %s: byte fields must not be negative", domain.DomainID)
	}
	if domain.CopyCount < 0 {
		return fmt.Errorf("memory domain %s: copy_count must not be negative", domain.DomainID)
	}
	if domain.PeakBytes < domain.CurrentBytes {
		return fmt.Errorf("memory domain %s: peak_bytes must be >= current_bytes", domain.DomainID)
	}
	if domain.BytesCopied > 0 && domain.CopyCount == 0 {
		return fmt.Errorf("memory domain %s: bytes_copied requires copy_count", domain.DomainID)
	}
	return nil
}

func KnownMemoryDomainKind(kind MemoryDomainKind) bool {
	switch kind {
	case DomainProcess, DomainTask, DomainActor, DomainIsland, DomainRequest, DomainExternal:
		return true
	default:
		return false
	}
}

func memoryDomainSummaryKey(domain MemoryDomain) string {
	return string(domain.Kind) + "\x00" + domain.DomainID + "\x00" + domain.ParentDomainID + "\x00" + domain.OwnerKind + "\x00" + domain.OwnerID + "\x00" + domain.Lifetime
}

func domainID(kind MemoryDomainKind, rawID string, fallback string) string {
	rawID = strings.TrimSpace(rawID)
	if rawID != "" {
		if strings.HasPrefix(rawID, "domain:") {
			return cleanDomainID(rawID)
		}
		return cleanDomainID("domain:" + rawID)
	}
	if fallback != "" {
		return cleanDomainID("domain:" + string(kind) + ":" + fallback)
	}
	return "domain:" + string(kind)
}

func cleanDomainOwner(raw string, fallback string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "domain:")
	for _, prefix := range []string{"process:", "task:", "actor:", "island:", "request:", "external:"} {
		raw = strings.TrimPrefix(raw, prefix)
	}
	if raw == "" {
		raw = fallback
	}
	return cleanDomainPart(raw)
}

func cleanDomainID(raw string) string {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	for i, part := range parts {
		parts[i] = cleanDomainPart(part)
	}
	return strings.Join(parts, ":")
}

func cleanDomainPart(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "unknown"
	}
	fields := strings.Fields(raw)
	if len(fields) > 0 {
		raw = strings.Join(fields, "_")
	}
	return raw
}

func defaultDomainString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
