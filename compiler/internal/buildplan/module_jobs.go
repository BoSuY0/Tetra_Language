package buildplan

import "tetra_language/compiler/internal/format/tobj"

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
