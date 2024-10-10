package buildplanner

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
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
	Script      *buildscript.BuildScript
}

func (b *BuildPlanner) CreateProject(params *CreateProjectParams) (strfmt.UUID, error) {
	logging.Debug("CreateProject, owner: %s, project: %s, language: %s, version: %s", params.Owner, params.Project, params.Language, params.Version)

	script := params.Script
	if script == nil {
		// Construct an initial buildexpression for the new project.
		var err error
		script, err = buildscript.New()
		if err != nil {
			return "", errs.Wrap(err, "Unable to create initial buildexpression")
		}

		// Add the platform.
		if err := script.AddPlatform(params.PlatformID); err != nil {
			return "", errs.Wrap(err, "Unable to add platform")
		}

		// Create a requirement for the given language and version.
		versionRequirements, err := VersionStringToRequirements(params.Version)
		if err != nil {
			return "", errs.Wrap(err, "Unable to read version")
		}
		if err := script.UpdateRequirement(types.OperationAdded, types.Requirement{
			Name:               params.Language,
			Namespace:          "language", // TODO: make this a constant DX-1738
			VersionRequirement: versionRequirements,
		}); err != nil {
			return "", errs.Wrap(err, "Unable to add language requirement")
		}
	}

	expression, err := script.MarshalBuildExpression()
	if err != nil {
		return "", errs.Wrap(err, "Marshalling build expression failed")
	}

	// Create the project.
	request := request.CreateProject(params.Owner, params.Project, params.Private, expression, params.Description)
	resp := &response.CreateProjectResult{}
	if err := b.client.Run(request, resp); err != nil {
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
