package model

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
)

func BuildRequestorForProject(pj *models.Project) (*headchef_models.BuildRequestRequester, *failures.Failure) {
	auth := authentication.Get()
	if !auth.Authenticated() {
		return nil, authentication.FailNotAuthenticated.New(locale.T("err_api_not_authenticated"))
	}
	return &headchef_models.BuildRequestRequester{
		OrganizationID: &pj.OrganizationID,
		ProjectID:      &pj.ProjectID,
		UserID:         auth.UserID(),
	}, nil
}

func BuildRequestForProject(pj *models.Project) (*headchef_models.BuildRequest, *failures.Failure) {
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
