package model

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
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
