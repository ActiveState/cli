package graph

type VersionResponse struct {
	Version Version `json:"version"`
}

type ProjectsResponse struct {
	Projects []*Project `json:"projects"`
}

type AvailableUpdateResponse struct {
	AvailableUpdate AvailableUpdate `json:"availableUpdate"`
}

type CheckMessagesResponse struct {
	Messages []*MessageInfo `json:"checkMessages"`
}

type GetProcessesInUseResponse struct {
	Processes []*ProcessInfo `json:"getProcessesInUse"`
}

type GetCommitResponse struct {
	AtTime     string `json:"atTime"`
	Expression string `json:"expression"`
	BuildPlan  string `json:"buildPlan"`
}
