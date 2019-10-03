package gql

import (
	"github.com/go-openapi/strfmt"
)

type Branch struct {
	BranchID     strfmt.UUID  `json:"branch_id"`
	CommitID     *strfmt.UUID `json:"commit_id"`
	Main         *bool        `json:"main"`
	ProjectID    *strfmt.UUID `json:"project_id"`
	TrackingType *string      `json:"tracking_type"` // graphql type: tracking_type
	Tracks       *strfmt.UUID `json:"tracks"`
	Label        string       `json:"label"`
}

type Branches []*Branch

type Organization struct {
	URLName string `json:"url_name"`
}

type ForkedProject struct {
	Name         string       `json:"name"`
	Organization Organization `json:"organization"`
}

type Project struct {
	Branches       Branches       `json:"branches"`
	Description    *string        `json:"description"`
	Name           string         `json:"name"`
	Added          Time           `json:"added"`
	CreatedBy      *strfmt.UUID   `json:"created_by"`
	ForkedFrom     *strfmt.UUID   `json:"forked_from"`
	ForkedProject  *ForkedProject `json:"forked_project"`
	Changed        Time           `json:"changed"`
	Managed        bool           `json:"managed"`
	OrganizationID strfmt.UUID    `json:"organization_id"`
	Private        bool           `json:"private"`
	ProjectID      strfmt.UUID    `json:"project_id"`
	RepoURL        *string        `json:"repo_url"`
}

type ProjectsResp struct {
	Projects []*Project `json:"projects"`
}

type ProjectResp struct {
	Project *Project `json:"project"`
}

func (psr *ProjectsResp) ProjectToProjectResp(index int) (*ProjectResp, error) {
	if psr.Projects == nil || index < 0 || len(psr.Projects) < index+1 {
		return nil, ErrNoValueAvailable
	}

	return &ProjectResp{Project: psr.Projects[index]}, nil
}

type ProjectClient interface {
	ProjectByOrgAndName(org, name string) (*ProjectResp, error)
}
