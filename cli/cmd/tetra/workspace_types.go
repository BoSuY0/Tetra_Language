package main

const (
	workspaceFileName = "Tetra.workspace"
	workspaceSchemaV1 = "tetra.workspace.v1"
)

type workspaceManifest struct {
	Path    string
	Root    string
	Schema  string
	Members []string
}

type workspaceReport struct {
	Status        string                  `json:"status,omitempty"`
	Root          string                  `json:"root"`
	WorkspacePath string                  `json:"workspace_path"`
	Members       []workspaceMemberReport `json:"members"`
}

type workspaceMemberReport struct {
	Path         string `json:"path"`
	ResolvedPath string `json:"resolved_path,omitempty"`
	CapsulePath  string `json:"capsule_path,omitempty"`
	CapsuleID    string `json:"capsule_id,omitempty"`
	Version      string `json:"version,omitempty"`
	Status       string `json:"status"`
	Detail       string `json:"detail,omitempty"`
}

type workspaceGraphReport struct {
	Status        string                  `json:"status,omitempty"`
	Root          string                  `json:"root"`
	WorkspacePath string                  `json:"workspace_path"`
	Nodes         []workspaceMemberReport `json:"nodes"`
	Edges         []workspaceGraphEdge    `json:"edges"`
}

type workspaceGraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	ID   string `json:"id"`
}

type workspaceExecutionReport struct {
	WorkspaceRoot string                           `json:"workspace_root"`
	Command       string                           `json:"command"`
	Target        string                           `json:"target,omitempty"`
	Total         int                              `json:"total"`
	Passed        int                              `json:"passed"`
	Failed        int                              `json:"failed"`
	Skipped       int                              `json:"skipped"`
	Members       []workspaceExecutionMemberReport `json:"members"`
}

type workspaceExecutionMemberReport struct {
	Path      string `json:"path"`
	CapsuleID string `json:"capsule_id,omitempty"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
	ExitCode  *int   `json:"exit_code,omitempty"`
}

type workspaceGraph struct {
	Workspace workspaceManifest
	Nodes     []workspaceMemberReport
	Edges     []workspaceGraphEdge
	Issues    []workspaceMemberReport
	ByRoot    map[string]workspaceMemberReport
}
