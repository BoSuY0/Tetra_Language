package buildapi

type EmitMode int

const (
	EmitExe EmitMode = iota
	EmitObject
	EmitLibrary
)

type RuntimeMode int

const (
	RuntimeAuto RuntimeMode = iota
	RuntimeSelfHost
	RuntimeBuiltin
)
