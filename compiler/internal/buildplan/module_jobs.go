package buildplan

import (
	"fmt"

	"tetra_language/compiler/internal/format/tobj"
)

const DefaultCompilerWorkerCostBytes int64 = 256 * 1024 * 1024

type WorkerDecision struct {
	Count  int
	Reason string
}

type ModuleObjectMetadata struct {
	Target          string
	Module          string
	CompilerVersion string
	PublicAPIHash   string
	SrcHash         [32]byte
	WorldSigHash    [32]byte
}

func EffectiveWorkerCount(requested int, maxJobs int, fallback int) int {
	if maxJobs <= 0 {
		return 0
	}
	jobs := requested
	if jobs <= 0 {
		jobs = fallback
	}
	if jobs < 1 {
		jobs = 1
	}
	if jobs > maxJobs {
		jobs = maxJobs
	}
	return jobs
}

func EffectiveWorkerDecision(
	requested int,
	maxJobs int,
	fallback int,
	memoryBudgetBytes int64,
	workerCostBytes int64,
) WorkerDecision {
	base := EffectiveWorkerCount(requested, maxJobs, fallback)
	if base == 0 {
		return WorkerDecision{
			Count: 0,
			Reason: fmt.Sprintf(
				"requested_jobs=%d max_jobs=%d num_cpu=%d no pending module work",
				requested,
				maxJobs,
				fallback,
			),
		}
	}
	reason := fmt.Sprintf(
		"requested_jobs=%d max_jobs=%d num_cpu=%d",
		requested,
		maxJobs,
		fallback,
	)
	if memoryBudgetBytes <= 0 {
		return WorkerDecision{Count: base, Reason: reason + " memory_budget_bytes=unset"}
	}
	cost := workerCostBytes
	if cost <= 0 {
		cost = DefaultCompilerWorkerCostBytes
	}
	budgetWorkers := int(memoryBudgetBytes / cost)
	if budgetWorkers < 1 {
		budgetWorkers = 1
	}
	if budgetWorkers > maxJobs {
		budgetWorkers = maxJobs
	}
	count := base
	if budgetWorkers < count {
		count = budgetWorkers
	}
	return WorkerDecision{
		Count: count,
		Reason: fmt.Sprintf(
			"%s memory_budget_bytes=%d worker_cost_bytes=%d budget_workers=%d",
			reason,
			memoryBudgetBytes,
			cost,
			budgetWorkers,
		),
	}
}

func ApplyModuleObjectMetadata(obj *tobj.Object, metadata ModuleObjectMetadata) {
	if obj == nil {
		return
	}
	obj.Target = metadata.Target
	obj.Module = metadata.Module
	obj.CompilerVersion = metadata.CompilerVersion
	obj.PublicAPIHash = metadata.PublicAPIHash
	obj.SrcHash = metadata.SrcHash
	obj.WorldSigHash = metadata.WorldSigHash
}
