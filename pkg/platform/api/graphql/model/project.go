package model

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

type Projects struct {
	Projects []*Project `json:"projects"`
}
