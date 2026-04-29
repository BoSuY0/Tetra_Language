package formats

import (
	"path/filepath"
	"strings"
)

const (
	T4SourceExtension          = ".t4"
	LegacyTetraSourceExtension = ".tetra"
	TodexFragmentExtension     = ".tdx"
	T4SeedExtension            = ".t4s"
	T4InterfaceExtension       = ".t4i"
	T4ProofExtension           = ".t4p"
	T4ReplayExtension          = ".t4r"
	T4QuestExtension           = ".t4q"
	NeedMapExtension           = ".tneed"

	CapsuleFileName       = "Capsule.t4"
	LegacyCapsuleFileName = "Tetra.capsule"
	SemanticLockFileName  = "Tetra.lock"
	DefaultSourceFileName = "main.t4"
	LegacySourceFileName  = "main.tetra"
	DefaultSeedFileName   = "tetra-seed.t4s"
	DefaultNeedMapName    = "tetra.tneed"
)

type Info struct {
	Name        string `json:"name"`
	Extension   string `json:"extension,omitempty"`
	FileName    string `json:"file_name,omitempty"`
	Role        string `json:"role"`
	Description string `json:"description"`
	Primary     bool   `json:"primary,omitempty"`
	Legacy      bool   `json:"legacy,omitempty"`
}

func All() []Info {
	return []Info{
		{
			Name:        "T4 Source Format",
			Extension:   T4SourceExtension,
			Role:        "source",
			Description: "Tetra source file for capsules, apps, kernels, drivers, UI, games, and tests.",
			Primary:     true,
		},
		{
			Name:        "Legacy Tetra Source Format",
			Extension:   LegacyTetraSourceExtension,
			Role:        "source",
			Description: "Legacy Tetra source file kept for backward compatibility.",
			Legacy:      true,
		},
		{
			Name:        "Todex Fragment",
			Extension:   TodexFragmentExtension,
			Role:        "todex-fragment",
			Description: "Todex encrypted semantic fragment.",
		},
		{
			Name:        "T4 Seed",
			Extension:   T4SeedExtension,
			Role:        "offline-seed",
			Description: "Tetra Seed offline bundle.",
		},
		{
			Name:        "T4 Interface",
			Extension:   T4InterfaceExtension,
			Role:        "interface",
			Description: "T4 interface file for fast type-checking without full source.",
		},
		{
			Name:        "T4 Proof",
			Extension:   T4ProofExtension,
			Role:        "proof",
			Description: "T4 proof and verification file.",
		},
		{
			Name:        "T4 Replay",
			Extension:   T4ReplayExtension,
			Role:        "replay",
			Description: "T4 replay file for reproducible bugs and desync reports.",
		},
		{
			Name:        "T4 Quest",
			Extension:   T4QuestExtension,
			Role:        "quest",
			Description: "T4 executable quest file.",
		},
		{
			Name:        "Tetra NeedMap",
			Extension:   NeedMapExtension,
			Role:        "needmap",
			Description: "NeedMap file describing missing Todex fragments for offline builds.",
		},
		{
			Name:        "Tetra Semantic Lock",
			FileName:    SemanticLockFileName,
			Role:        "semantic-lock",
			Description: "Tetra semantic lockfile for versions, policies, and reproducibility guarantees.",
		},
	}
}

func SourceExtensions() []string {
	return []string{T4SourceExtension, LegacyTetraSourceExtension}
}

func IsSourceFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case T4SourceExtension, LegacyTetraSourceExtension:
		return true
	default:
		return false
	}
}

func ModuleRelPath(module string, extension string) string {
	return filepath.FromSlash(strings.ReplaceAll(module, ".", "/") + extension)
}

func ModuleCandidateRelPaths(module string) []string {
	return []string{
		ModuleRelPath(module, T4SourceExtension),
		ModuleRelPath(module, LegacyTetraSourceExtension),
	}
}
