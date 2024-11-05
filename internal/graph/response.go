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

type HashGlobsResponse struct {
	Response GlobResult `json:"hashGlobs"`
}
