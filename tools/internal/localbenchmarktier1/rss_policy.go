package localbenchmarktier1

import "strings"

const (
	schemaLocalRSSBudgetPolicyV1 = "tetra.local_benchmark.rss_budget_policy.v1"
	localRSSBudgetPolicyFile     = "rss-budget-policy.local.json"
)

type localRSSBudgetPolicy struct {
	Schema      string                `json:"schema"`
	Target      string                `json:"target"`
	HostProfile localRSSBudgetHost    `json:"host_profile"`
	Budgets     []localRSSBudgetEntry `json:"budgets"`
	NonClaims   []string              `json:"non_claims"`
}

type localRSSBudgetHost struct {
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	CPUs      int    `json:"cpus"`
	TargetCPU string `json:"target_cpu"`
	GitCommit string `json:"git_commit"`
}

type localRSSBudgetEntry struct {
	Category               string  `json:"category"`
	Language               string  `json:"language"`
	RSSPeakBudgetBytes     uint64  `json:"rss_peak_budget_bytes"`
	AllowedVariancePercent float64 `json:"allowed_variance_percent"`
	Reason                 string  `json:"reason"`
}

func writeLocalRSSBudgetPolicy(path string, report tier1Report) error {
	return writeJSON(path, buildLocalRSSBudgetPolicy(report))
}

func buildLocalRSSBudgetPolicy(report tier1Report) localRSSBudgetPolicy {
	policy := localRSSBudgetPolicy{
		Schema: schemaLocalRSSBudgetPolicyV1,
		Target: targetFromHost(report.Host),
		HostProfile: localRSSBudgetHost{
			GOOS:      report.Host.GOOS,
			GOARCH:    report.Host.GOARCH,
			CPUs:      report.Host.CPUs,
			TargetCPU: report.Host.TargetCPU,
			GitCommit: report.Host.GitCommit,
		},
		NonClaims: []string{
			"local RSS budget only",
			"no cross-machine RSS claim",
			"no official benchmark claim",
		},
	}
	for _, result := range report.Results {
		for _, row := range result.Rows {
			if row.Language != "tetra" || row.Status != "measured" ||
				row.TetraMetadata == nil || row.TetraMetadata.MemoryEvidence == nil {
				continue
			}
			peakMetric := row.TetraMetadata.MemoryEvidence.RSSPeak
			peakBytes, ok := measuredRSSPeakBudgetBytes(peakMetric)
			if !ok {
				continue
			}
			category := row.Category
			if strings.TrimSpace(category) == "" {
				category = result.Category
			}
			policy.Budgets = append(policy.Budgets, localRSSBudgetEntry{
				Category:               category,
				Language:               "tetra",
				RSSPeakBudgetBytes:     peakBytes,
				AllowedVariancePercent: 5,
				Reason: ("generated local host-pinned RSS budget from measured " +
					"row rss_peak evidence"),
			})
		}
	}
	return policy
}

func measuredRSSPeakBudgetBytes(metric memoryMetric) (uint64, bool) {
	if metric.EvidenceClass != "runtime_measured" ||
		strings.TrimSpace(metric.SourceArtifact) == "" {
		return 0, false
	}
	if metric.PeakBytes > 0 {
		return metric.PeakBytes, true
	}
	if metric.Bytes > 0 {
		return metric.Bytes, true
	}
	return 0, false
}

func targetFromHost(host tier1Host) string {
	arch := host.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	if host.GOOS == "" || arch == "" {
		return ""
	}
	return host.GOOS + "-" + arch
}
