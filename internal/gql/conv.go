package gql

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/go-openapi/strfmt"
)

func (pr *ProjectResp) ToMonoProject() (*mono_models.Project, *failures.Failure) {
	p := mono_models.Project{
		Added:       strfmt.DateTime{},
		Branches:    nil,
		CreatedBy:   nil,
		Description: nil,
		ForkedFrom: &mono_models.ProjectForkedFrom{
			Organization: "",
			Project:      "",
		},
		Languages:      nil,
		LastEdited:     strfmt.DateTime{},
		Managed:        false,
		Name:           "",
		OrganizationID: "",
		Platforms:      nil,
		Private:        false,
		ProjectID:      "",
		RepoURL:        nil,
	}

	return &p, nil
}
