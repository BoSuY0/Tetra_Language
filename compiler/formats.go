package compiler

import "tetra_language/compiler/internal/formats"

type FormatInfo = formats.Info

const (
	T4SourceExtension          = formats.T4SourceExtension
	LegacyTetraSourceExtension = formats.LegacyTetraSourceExtension
	TodexFragmentExtension     = formats.TodexFragmentExtension
	T4SeedExtension            = formats.T4SeedExtension
	T4InterfaceExtension       = formats.T4InterfaceExtension
	T4ProofExtension           = formats.T4ProofExtension
	T4ReplayExtension          = formats.T4ReplayExtension
	T4QuestExtension           = formats.T4QuestExtension
	NeedMapExtension           = formats.NeedMapExtension

	CapsuleFileName       = formats.CapsuleFileName
	LegacyCapsuleFileName = formats.LegacyCapsuleFileName
	SemanticLockFileName  = formats.SemanticLockFileName
	DefaultSourceFileName = formats.DefaultSourceFileName
	LegacySourceFileName  = formats.LegacySourceFileName
	DefaultSeedFileName   = formats.DefaultSeedFileName
	DefaultNeedMapName    = formats.DefaultNeedMapName
)

func T4Formats() []FormatInfo {
	return formats.All()
}

func SourceExtensions() []string {
	return formats.SourceExtensions()
}

func IsSourceFile(path string) bool {
	return formats.IsSourceFile(path)
}
