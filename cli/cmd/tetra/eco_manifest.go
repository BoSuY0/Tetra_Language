package main

type capsuleManifest struct {
	ManifestSchema string
	Name           string
	ID             string
	Version        string
	Path           string
	Entry          string
	SourceRoots    []string
	Targets        []string
	Effects        []string
	Permissions    []string
	Dependencies   []capsuleDependency
	Artifacts      []capsuleArtifact
	Policy         map[string]string
}

type capsuleDependency struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Path    string `json:"path,omitempty"`
}

type capsuleArtifact struct {
	Kind   string `json:"kind"`
	Target string `json:"target,omitempty"`
	Path   string `json:"path"`
}
