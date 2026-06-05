package selfhostgate

import "strings"

type Evidence struct {
	CompilerSubsetDefined       bool
	RegisterBackendStable       bool
	OptimizerValidated          bool
	AllocatorStable             bool
	StdlibStrongEnough          bool
	SmallCompilerComponentBuilt bool
	GoVsTetraOutputCompared     bool
	DeterministicBootstrapChain bool
	CrossPlatformBootstrapStory bool
}

type Decision struct {
	Allowed         bool     `json:"allowed"`
	MissingEvidence []string `json:"missing_evidence,omitempty"`
	Reason          string   `json:"reason"`
}

func Evaluate(e Evidence) Decision {
	var missing []string
	if !e.CompilerSubsetDefined {
		missing = append(missing, "compiler_subset_defined")
	}
	if !e.RegisterBackendStable {
		missing = append(missing, "register_backend_stable")
	}
	if !e.OptimizerValidated {
		missing = append(missing, "optimizer_validated")
	}
	if !e.AllocatorStable {
		missing = append(missing, "allocator_stable")
	}
	if !e.StdlibStrongEnough {
		missing = append(missing, "stdlib_strong_enough")
	}
	if !e.SmallCompilerComponentBuilt {
		missing = append(missing, "small_compiler_component_compiled")
	}
	if !e.GoVsTetraOutputCompared {
		missing = append(missing, "go_vs_tetra_output_compared")
	}
	if !e.DeterministicBootstrapChain {
		missing = append(missing, "deterministic_bootstrap_chain")
	}
	if !e.CrossPlatformBootstrapStory {
		missing = append(missing, "cross_platform_bootstrap_story")
	}
	if len(missing) > 0 {
		return Decision{
			Allowed:         false,
			MissingEvidence: missing,
			Reason:          "blocked: self-hosting remains gated until " + strings.Join(missing, ", "),
		}
	}
	return Decision{Allowed: true, Reason: "allowed: verified core evidence is present"}
}

func (d Decision) Missing(name string) bool {
	for _, missing := range d.MissingEvidence {
		if missing == name {
			return true
		}
	}
	return false
}
