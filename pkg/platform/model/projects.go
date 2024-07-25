package model

import (
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/graphql"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	clientProjects "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type ErrProjectNotFound struct {
	Organization string
	Project      string
}

func (e *ErrProjectNotFound) Error() string {
	return fmt.Sprintf("project not found: %s/%s", e.Organization, e.Project)
}

// LegacyFetchProjectByName is intended for legacy code which still relies on localised errors, do NOT use it for new code.
func LegacyFetchProjectByName(orgName string, projectName string) (*mono_models.Project, error) {
	auth, err := authentication.LegacyGet()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get auth")
	}
	project, err := FetchProjectByName(orgName, projectName, auth)
	if err == nil || !errs.Matches(err, &ErrProjectNotFound{}) {
		return project, err
	}
	if !auth.Authenticated() {
		return nil, errs.AddTips(
			locale.NewInputError("err_api_project_not_found", "", orgName, projectName),
			locale.T("tip_private_project_auth"))
	}
	return nil, errs.Pack(err, locale.NewInputError("err_api_project_not_found", "", orgName, projectName))
}

// FetchProjectByName fetches a project for an organization.
func FetchProjectByName(orgName string, projectName string, auth *authentication.Auth) (*mono_models.Project, error) {
	logging.Debug("fetching project (%s) in organization (%s)", projectName, orgName)

	request := request.ProjectByOrgAndName(orgName, projectName)

	gql := graphql.New(auth)
	response := model.Projects{}
	err := gql.Run(request, &response)
	if err != nil {
		return nil, errs.Wrap(err, "GraphQL request failed")
	}

	if len(response.Projects) == 0 {
		return nil, &ErrProjectNotFound{orgName, projectName}
	}

	return response.Projects[0].ToMonoProject()
}

// FetchOrganizationProjects fetches the projects for an organization
func FetchOrganizationProjects(orgName string, auth *authentication.Auth) ([]*mono_models.Project, error) {
	authClient, err := auth.Client()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get auth client")
	}
	projParams := clientProjects.NewListProjectsParams()
	projParams.SetOrganizationName(orgName)
	orgProjects, err := authClient.Projects.ListProjects(projParams, auth.ClientAuth())
	if err != nil {
		switch statusCode := api.ErrorCode(err); statusCode {
		case 401:
			return nil, locale.WrapExternalError(err, "err_api_not_authenticated")
		case 404:
			// NOT a project not found error; we didn't ask for a specific project.
			return nil, locale.WrapExternalError(err, "err_api_org_not_found")
		default:
			return nil, locale.WrapError(err, "err_api_unknown", "Unexpected API error")
		}
	}
	return orgProjects.Payload, nil
}

func LanguageByCommit(commitID strfmt.UUID, auth *authentication.Auth) (Language, error) {
	languages, err := FetchLanguagesForCommit(commitID, auth)
	if err != nil {
		return Language{}, err
	}

	if len(languages) == 0 {
		return Language{}, nil
	}

	return languages[0], nil
}

func FetchTimeStampForCommit(commitID strfmt.UUID, auth *authentication.Auth) (*time.Time, error) {
	_, atTime, err := FetchCheckpointForCommit(commitID, auth)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to fetch checkpoint for commit ID")
	}

	t := time.Time(atTime)
	return &t, nil
}

// DefaultBranchForProjectName retrieves the default branch for the given project owner/name.
func DefaultBranchForProjectName(owner, name string) (*mono_models.Branch, error) {
	proj, err := LegacyFetchProjectByName(owner, name)
	if err != nil {
		return nil, err
	}

	return DefaultBranchForProject(proj)
}

func BranchesForProject(owner, name string) ([]*mono_models.Branch, error) {
	proj, err := LegacyFetchProjectByName(owner, name)
	if err != nil {
		return nil, err
	}
	return proj.Branches, nil
}

func BranchNamesForProjectFiltered(owner, name string, excludes ...string) ([]string, error) {
	proj, err := LegacyFetchProjectByName(owner, name)
	if err != nil {
		return nil, err
	}
	branches := make([]string, 0)
	for _, branch := range proj.Branches {
		for _, exclude := range excludes {
			if branch.Label != exclude {
				branches = append(branches, branch.Label)
			}
		}
	}
	return branches, nil
}

// DefaultBranchForProject retrieves the default branch for the given project
func DefaultBranchForProject(pj *mono_models.Project) (*mono_models.Branch, error) {
	for _, branch := range pj.Branches {
		if branch.Default {
			return branch, nil
		}
	}
	return nil, locale.NewError("err_no_default_branch")
}

// BranchForProjectNameByName retrieves the named branch for the given project
// org/name
func BranchForProjectNameByName(owner, name, branch string) (*mono_models.Branch, error) {
	proj, err := LegacyFetchProjectByName(owner, name)
	if err != nil {
		return nil, err
	}

	return BranchForProjectByName(proj, branch)
}

// BranchForProjectByName retrieves the named branch for the given project
func BranchForProjectByName(pj *mono_models.Project, name string) (*mono_models.Branch, error) {
	if name == "" {
		return nil, locale.NewInputError("err_empty_branch", "Empty branch name provided.")
	}

	for _, branch := range pj.Branches {
		if branch.Label != "" && branch.Label == name {
			return branch, nil
		}
	}

	return nil, locale.NewInputError(
		"err_no_matching_branch_label",
		"This project has no branch with label matching '[NOTICE]{{.V0}}[/RESET]'.",
		name,
	)
}

// CreateEmptyProject will create the project on the platform
func CreateEmptyProject(owner, name string, private bool, auth *authentication.Auth) (*mono_models.Project, error) {
	authClient, err := auth.Client()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get auth client")
	}
	addParams := projects.NewAddProjectParams()
	addParams.SetOrganizationName(owner)
	addParams.SetProject(&mono_models.Project{Name: name, Private: private})
	pj, err := authClient.Projects.AddProject(addParams, auth.ClientAuth())
	if err != nil {
		msg := api.ErrorMessageFromPayload(err)
		if errs.Matches(err, &projects.AddProjectConflict{}) || errs.Matches(err, &projects.AddProjectNotFound{}) {
			return nil, locale.WrapInputError(err, msg)
		}
		return nil, locale.WrapError(err, msg)
	}

	return pj.Payload, nil
}

func CreateCopy(sourceOwner, sourceName, targetOwner, targetName string, makePrivate bool, auth *authentication.Auth) (*mono_models.Project, error) {
	// Retrieve the source project that we'll be forking
	sourceProject, err := LegacyFetchProjectByName(sourceOwner, sourceName)
	if err != nil {
		return nil, locale.WrapExternalError(err, "err_fork_fetchProject", "Could not find the source project: {{.V0}}/{{.V1}}", sourceOwner, sourceName)
	}

	// Create the target project
	targetProject, err := CreateEmptyProject(targetOwner, targetName, false, auth)
	if err != nil {
		return nil, locale.WrapError(err, "err_fork_createProject", "Could not create project: {{.V0}}/{{.V1}}", targetOwner, targetName)
	}

	sourceBranch, err := DefaultBranchForProject(sourceProject)
	if err != nil {
		return nil, locale.WrapError(err, "err_branch_nodefault", "Project has no default branch.")
	}
	if sourceBranch.CommitID != nil {
		targetBranch, err := DefaultBranchForProject(targetProject)
		if err != nil {
			return nil, locale.WrapError(err, "err_branch_nodefault", "Project has no default branch.")
		}
		if err := UpdateBranchCommit(targetBranch.BranchID, *sourceBranch.CommitID, auth); err != nil {
			return nil, locale.WrapError(err, "err_fork_branchupdate", "Failed to update branch.")
		}
	}

	// Turn the target project private if this was requested (unfortunately this can't be done int the Creation step)
	if makePrivate {
		if err := MakeProjectPrivate(targetOwner, targetName, auth); err != nil {
			logging.Debug("Cannot make forked project private; deleting public fork.")
			if authClient, err2 := auth.Client(); err2 == nil {
				deleteParams := projects.NewDeleteProjectParams()
				deleteParams.SetOrganizationName(targetOwner)
				deleteParams.SetProjectName(targetName)
				if _, err3 := authClient.Projects.DeleteProject(deleteParams, auth.ClientAuth()); err3 != nil {
					err = errs.Pack(err, locale.WrapError(
						err3, "err_fork_private_but_project_created",
						"Your project was created but could not be made private. Please head over to {{.V0}} to manually update your privacy settings.",
						api.GetPlatformURL(fmt.Sprintf("%s/%s", targetOwner, targetName)).String()))
				}
			} else {
				err = errs.Pack(err, errs.Wrap(err2, "Could not get auth client"))
			}
			return nil, locale.WrapError(err, "err_fork_private", "Your fork could not be made private.")
		}
	}

	return targetProject, nil
}

// MakeProjectPrivate turns the given project private
func MakeProjectPrivate(owner, name string, auth *authentication.Auth) error {
	authClient, err := auth.Client()
	if err != nil {
		return errs.Wrap(err, "Could not get auth client")
	}

	editParams := projects.NewEditProjectParams()
	yes := true
	editParams.SetProject(&mono_models.ProjectEditable{
		Private: &yes,
	})
	editParams.SetOrganizationName(owner)
	editParams.SetProjectName(name)

	_, err = authClient.Projects.EditProject(editParams, auth.ClientAuth())
	if err != nil {
		msg := api.ErrorMessageFromPayload(err)
		if errs.Matches(err, &projects.EditProjectBadRequest{}) {
			return locale.WrapExternalError(err, msg) // user does not have permission
		}
		return locale.WrapError(err, msg)
	}

	return nil
}

// ProjectURL creates a valid platform URL for the given project parameters
func ProjectURL(owner, name, commitID string) string {
	url := api.GetPlatformURL(fmt.Sprintf("%s/%s", owner, name))
	if commitID != "" {
		query := url.Query()
		query.Add("commitID", commitID)
		url.RawQuery = query.Encode()
	}
	return url.String()
}

func AddBranch(projectID strfmt.UUID, label string, auth *authentication.Auth) (strfmt.UUID, error) {
	var branchID strfmt.UUID
	authClient, err := auth.Client()
	if err != nil {
		return "", errs.Wrap(err, "Could not get auth client")
	}
	addParams := projects.NewAddBranchParams()
	addParams.SetProjectID(projectID)
	addParams.Body.Label = label

	res, err := authClient.Projects.AddBranch(addParams, auth.ClientAuth())
	if err != nil {
		msg := api.ErrorMessageFromPayload(err)
		return branchID, locale.WrapError(err, msg)
	}

	return res.Payload.BranchID, nil
}

func EditProject(owner, name string, project *mono_models.ProjectEditable, auth *authentication.Auth) error {
	authClient, err := auth.Client()
	if err != nil {
		return errs.Wrap(err, "Could not get auth client")
	}

	editParams := projects.NewEditProjectParams()
	editParams.SetOrganizationName(owner)
	editParams.SetProjectName(name)
	editParams.SetProject(project)

	_, err = authClient.Projects.EditProject(editParams, auth.ClientAuth())
	if err != nil {
		msg := api.ErrorMessageFromPayload(err)
		return locale.WrapError(err, msg)
	}

	return nil
}

func DeleteProject(owner, project string, auth *authentication.Auth) error {
	authClient, err := auth.Client()
	if err != nil {
		return errs.Wrap(err, "Could not get auth client")
	}

	params := projects.NewDeleteProjectParams()
	params.SetOrganizationName(owner)
	params.SetProjectName(project)

	_, err = authClient.Projects.DeleteProject(params, auth.ClientAuth())
	if err != nil {
		msg := api.ErrorMessageFromPayload(err)
		return locale.WrapError(err, msg)
	}

	return nil
}

func MoveProject(owner, project, newOwner string, auth *authentication.Auth) error {
	authClient, err := auth.Client()
	if err != nil {
		return errs.Wrap(err, "Could not get auth client")
	}

	params := projects.NewMoveProjectParams()
	params.SetOrganizationIdentifier(owner)
	params.SetProjectName(project)
	params.SetDestination(projects.MoveProjectBody{newOwner})

	_, err = authClient.Projects.MoveProject(params, auth.ClientAuth())
	if err != nil {
		msg := api.ErrorMessageFromPayload(err)
		return locale.WrapError(err, msg)
	}

	return nil
}
