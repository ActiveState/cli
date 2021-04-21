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
