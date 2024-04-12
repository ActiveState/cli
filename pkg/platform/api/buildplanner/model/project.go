package model

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
	"github.com/go-openapi/strfmt"
)

// CreateProjectParams contains information for the project to create.
// When creating a project from scratch, the PlatformID, Language, Version, and Timestamp fields
// are used to create a buildexpression to use.
// When creating a project based off of another one, the Expr field is used (PlatformID, Language,
// Version, and Timestamp are ignored).
type CreateProjectParams struct {
	Owner       string
	Project     string
	PlatformID  strfmt.UUID
	Language    string
	Version     string
	Private     bool
	Description string
	Expr        *buildexpression.BuildExpression
}

func (bp *BuildPlanner) CreateProject(params *CreateProjectParams) (strfmt.UUID, error) {
	logging.Debug("CreateProject, owner: %s, project: %s, language: %s, version: %s", params.Owner, params.Project, params.Language, params.Version)

	expr := params.Expr
	if expr == nil {
		// Construct an initial buildexpression for the new project.
		var err error
		expr, err = buildexpression.NewEmpty()
		if err != nil {
			return "", errs.Wrap(err, "Unable to create initial buildexpression")
		}

		// Add the platform.
		if err := expr.UpdatePlatform(types.OperationAdded, params.PlatformID); err != nil {
			return "", errs.Wrap(err, "Unable to add platform")
		}

		// Create a requirement for the given language and version.
		versionRequirements, err := VersionStringToRequirements(params.Version)
		if err != nil {
			return "", errs.Wrap(err, "Unable to read version")
		}
		if err := expr.UpdateRequirement(types.OperationAdded, types.Requirement{
			Name:               params.Language,
			Namespace:          "language", // TODO: make this a constant DX-1738
			VersionRequirement: versionRequirements,
		}); err != nil {
			return "", errs.Wrap(err, "Unable to add language requirement")
		}
	}

	// Create the project.
	request := request.CreateProject(params.Owner, params.Project, params.Private, expr, params.Description)
	resp := &response.CreateProjectResult{}
	err := bp.client.Run(request, resp)
	if err != nil {
		return "", processBuildPlannerError(err, "Failed to create project")
	}

	if resp.ProjectCreated == nil {
		return "", errs.New("ProjectCreated is nil")
	}

	if response.IsErrorResponse(resp.ProjectCreated.Type) {
		return "", response.ProcessProjectCreatedError(resp.ProjectCreated, "Could not create project")
	}

	if resp.ProjectCreated.Commit == nil {
		return "", errs.New("ProjectCreated.Commit is nil")
	}

	return resp.ProjectCreated.Commit.CommitID, nil
}
