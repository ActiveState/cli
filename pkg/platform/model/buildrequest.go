package model

import (
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/go-openapi/strfmt"
)

func NewHeadChefRequester(pj *mono_models.Project) (*headchef_models.V1Requester, error) {
	userID := strfmt.UUID("00010001-0001-0001-0001-000100010001")
	auth := authentication.Get()
	if auth.Authenticated() {
		userID = *auth.UserID()
	}
	return &headchef_models.V1Requester{
		OrganizationID: &pj.OrganizationID,
		ProjectID:      &pj.ProjectID,
		UserID:         userID,
	}, nil
}

func NewBuildRequest(pj *mono_models.Project) (*headchef_models.V1BuildRequest, error) {
	requester, err := NewHeadChefRequester(pj)
	if err != nil {
		return nil, err
	}

	format := "raw"
	return &headchef_models.V1BuildRequest{
		Requester: requester,
		Format:    &format,
	}, nil
}
