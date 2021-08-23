package graph

type VersionResponse struct {
	Version Version `json:"version"`
}

type ProjectsResponse struct {
	Projects []*Project `json:"projects"`
}

type UpdateResponse struct {
	DeferredUpdate DeferredUpdate `json:"update"`
}

type AvailableUpdateResponse struct {
	AvailableUpdate AvailableUpdate `json:"availableUpdate"`
}

type QuitResponse struct {
	Quit chan bool `json:"quit"`
}
