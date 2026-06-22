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
	candidates := []string{
		ModuleRelPath(module, T4SourceExtension),
		ModuleRelPath(module, LegacyTetraSourceExtension),
	}
	if extra := standardLibraryModuleRelPath(module); extra != "" {
		candidates = append(candidates,
			filepath.FromSlash(extra+T4SourceExtension),
			filepath.FromSlash(extra+LegacyTetraSourceExtension),
		)
	}
	return candidates
}

func standardLibraryModuleRelPath(module string) string {
	if strings.HasPrefix(module, "lib.core.") {
		name := strings.TrimPrefix(module, "lib.core.")
		if bucket := coreStdlibBucket(name); bucket != "" {
			return "lib/core/" + bucket + "/" + name
		}
	}
	if strings.HasPrefix(module, "lib.experimental.") {
		name := strings.TrimPrefix(module, "lib.experimental.")
		if bucket := experimentalStdlibBucket(name); bucket != "" {
			return "lib/experimental/" + bucket + "/" + name
		}
	}
	return ""
}

func coreStdlibBucket(name string) string {
	switch name {
	case "capability", "math", "strings", "sync", "testing", "time":
		return "base"
	case "collections", "crypto", "json", "serialization", "slices":
		return "data"
	case "filesystem", "http", "io", "net", "networking", "postgres":
		return "io"
	case "memory":
		return "memory"
	case "async":
		return "async"
	case "actors":
		return "actors"
	case "accessibility", "component", "draw", "i18n", "surface", "text":
		return "surface"
	case "style", "surface_app", "surface_app_shell", "widgets":
		return "widgets"
	case "block":
		return "block"
	case "morph":
		return "morph"
	default:
		return ""
	}
}

func experimentalStdlibBucket(name string) string {
	switch name {
	case "math", "strings", "sync", "testing", "time":
		return "base"
	case "collections", "crypto", "serialization", "slices":
		return "data"
	case "filesystem", "io", "networking":
		return "io"
	case "memory":
		return "memory"
	case "async":
		return "async"
	default:
		return ""
	}
}
