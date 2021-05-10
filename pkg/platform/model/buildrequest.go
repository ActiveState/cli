package model

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/go-openapi/strfmt"
)

func RequestBuild(recipeID, commitID strfmt.UUID, owner, project string) (headchef.BuildStatusEnum, *headchef_models.BuildStatusResponse, error) {
	var platProj *mono_models.Project
	if owner != "" && project != "" {
		var err error
		platProj, err = FetchProjectByName(owner, project)
		if err != nil {
			return headchef.Error, nil, locale.WrapError(err, "build_request_get_project_err", "Could not find project {{.V0}}/{{.V1}} on ActiveState Platform.", owner, project)
		}
	}

	buildAnnotations := headchef.BuildAnnotations{
		CommitID:     commitID.String(),
		Project:      project,
		Organization: owner,
	}

	orgID := strfmt.UUID(constants.ValidZeroUUID)
	projectID := strfmt.UUID(constants.ValidZeroUUID)
	if platProj != nil {
		orgID = platProj.OrganizationID
		projectID = platProj.ProjectID
	}

	return requestBuild(recipeID, orgID, projectID, buildAnnotations)
}

func requestBuild(recipeID, orgID, projID strfmt.UUID, annotations headchef.BuildAnnotations) (headchef.BuildStatusEnum, *headchef_models.BuildStatusResponse, error) {
	buildRequest, err := headchef.NewBuildRequest(recipeID, orgID, projID, annotations)
	if err != nil {
		return headchef.Error, nil, err
	}
	return headchef.InitClient().RequestBuildSync(buildRequest)
}
