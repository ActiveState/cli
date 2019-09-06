package model

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func NewModelsRequester(pj *mono_models.Project) (*headchef_models.Requester, *failures.Failure) {
	auth := authentication.Get()
	if !auth.Authenticated() {
		return nil, authentication.FailNotAuthenticated.New(locale.T("err_api_not_authenticated"))
	}
	return &headchef_models.Requester{
		OrganizationID: &pj.OrganizationID,
		ProjectID:      &pj.ProjectID,
		UserID:         auth.UserID(),
	}, nil
}

func NewBuildRequest(pj *mono_models.Project) (*headchef_models.V1BuildRequest, *failures.Failure) {
	requester, fail := NewModelsRequester(pj)
	if fail != nil {
		return nil, fail
	}

	return &headchef_models.V1BuildRequest{
		Requester: requester,
	}, nil
}
