package model

import (
	"fmt"
	"time"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/gql"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/gqldb/projdb"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api"
	clientProjects "github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type ProjectProvider interface {
	gql.ProjectClient
}

var prv = func() ProjectProvider {
	if condition.InTest() {
		return projdb.NewMock()
	}

	endpoint := constants.GraphqlURLStage
	if constants.APIEnv == "prod" {
		endpoint = constants.GraphqlURLProd
	}

	timeout := time.Second * 16

	gc := gqlclient.New(endpoint, nil, timeout)
	fmt.Println(endpoint)

	p, err := projdb.New(gc)
	if err != nil {
		panic(err)
	}
	return p
}()

var (
	// FailNoValidProject is a failure for the call api.GetProject
	FailNoValidProject = failures.Type("model.fail.novalidproject")

	// FailNoDefaultBranch is a failure in getting a project's default branch
	FailNoDefaultBranch = failures.Type("model.fail.nodefaultbranch")

	// FailCannotConvertModel is a failure to convert a new model to an existing model
	FailCannotConvertModel = failures.Type("model.fail.cannotconvertmodel")
)

// FetchProjectByName fetches a project for an organization.
func FetchProjectByName(orgName string, projectName string) (*mono_models.Project, *failures.Failure) {
	proj, err := prv.ProjectByOrgAndName(orgName, projectName)
	if err != nil {
		return nil, FailNoValidProject.Wrap(err)
	}

	mp, err := proj.ToMonoProject()
	if err != nil {
		return nil, FailCannotConvertModel.Wrap(err)
	}

	return mp, nil
}

// FetchOrganizationProjects fetches the projects for an organization
func FetchOrganizationProjects(orgName string) ([]*mono_models.Project, *failures.Failure) {
	projParams := clientProjects.NewListProjectsParams()
	projParams.SetOrganizationName(orgName)
	orgProjects, err := authentication.Client().Projects.ListProjects(projParams, authentication.ClientAuth())
	if err != nil {
		return nil, processProjectErrorResponse(err)
	}
	return orgProjects.Payload, nil
}

// DefaultBranchForProject retrieves the default branch for the given project
func DefaultBranchForProject(pj *mono_models.Project) (*mono_models.Branch, *failures.Failure) {
	for _, branch := range pj.Branches {
		if branch.Default {
			return branch, nil
		}
	}
	return nil, FailNoDefaultBranch.New(locale.T("err_no_default_branch"))
}

// ProjectURL creates a valid platform URL for the given project parameters
func ProjectURL(owner, name, commitID string) string {
	url := fmt.Sprintf("https://%s/%s/%s", constants.PlatformURL, owner, name)
	if commitID != "" {
		url = url + "?commitID=" + commitID
	}
	return url
}

func processProjectErrorResponse(err error, params ...string) *failures.Failure {
	switch statusCode := api.ErrorCode(err); statusCode {
	case 401:
		return api.FailAuth.New("err_api_not_authenticated")
	case 404:
		return api.FailProjectNotFound.New("err_api_project_not_found", params...)
	default:
		return api.FailUnknown.Wrap(err)
	}
}
