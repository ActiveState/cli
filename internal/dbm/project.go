package dbm

type Organization struct {
	DisplayName string `json:"display_name"`
	URLName     string `json:"url_name"`
}
type Branch struct {
	CommitID  string `json:"commit_id"`
	Main      bool   `json:"main"`
	ProjectID string `json:"project_id"`
}
type Project struct {
	Organization *Organization `json:"organization"`
	Branches     []*Branch     `json:"branches"`
	Description  string        `json:"description"`
	Name         string        `json:"name"`
}

type ProjectsResp struct {
	Projects []*Project `json:"projects"`
}

type ProjectResp struct {
	Project *Project `json:"project"`
}
