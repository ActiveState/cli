package model

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
)

func BuildRequestorForProject(pj *mono_models.Project) (*headchef_models.BuildRequestRequester, *failures.Failure) {
	userID := strfmt.UUID("00010001-0001-0001-0001-000100010001")
	auth := authentication.Get()
	if auth.Authenticated() {
		userID = *auth.UserID()
	}
	return &headchef_models.BuildRequestRequester{
		OrganizationID: &pj.OrganizationID,
		ProjectID:      &pj.ProjectID,
		UserID:         &userID,
	}, nil
}

func BuildRequestForProject(pj *mono_models.Project) (*headchef_models.BuildRequest, *failures.Failure) {
	requestor, fail := BuildRequestorForProject(pj)
	if fail != nil {
		return nil, fail
	}

	uuid := strfmt.UUID(uuid.New().String())
	return &headchef_models.BuildRequest{
		BuildRequestID: &uuid,
		Requester:      requestor,
	}, nil
}
