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

type DeprecationResponse struct {
	CheckDeprecation DeprecationInfo `json:"checkDeprecation"`
}

