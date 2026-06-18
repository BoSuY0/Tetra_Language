package buildapi

type BuildStats struct {
	CompiledModules  []string
	CacheHits        []string
	LoweredModules   []string
	InterfaceModules []string
}
