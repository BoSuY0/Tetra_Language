package compiler

import buildapi "tetra_language/compiler/internal/buildapi"

type EmitMode = buildapi.EmitMode

const (
	EmitExe     = buildapi.EmitExe
	EmitObject  = buildapi.EmitObject
	EmitLibrary = buildapi.EmitLibrary
)

type RuntimeMode = buildapi.RuntimeMode

const (
	RuntimeAuto     = buildapi.RuntimeAuto
	RuntimeSelfHost = buildapi.RuntimeSelfHost
	RuntimeBuiltin  = buildapi.RuntimeBuiltin
)

type BuildOptions = buildapi.BuildOptions

type BuildStats = buildapi.BuildStats
