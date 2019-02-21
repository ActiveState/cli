package projects_test

import (
	"testing"

	"github.com/ActiveState/cli/pkg/platform/authentication"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/projects"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/stretchr/testify/assert"
)

func TestProjects_FetchByName_Succeeds(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.RegisterWithCode("GET", "/organizations/ActiveState/projects/CodeIntel", 200)

	project, fail := projects.FetchByName("ActiveState", "CodeIntel")
	assert.NoError(t, fail.ToError(), "Fetched project")
	assert.Equal(t, "CodeIntel", project.Name)
}

func TestProjects_FetchByName_NotFound(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.RegisterWithCode("GET", "/organizations/ActiveState/projects/CodeIntel", 404)

	project, fail := projects.FetchByName("ActiveState", "CodeIntel")
	assert.EqualError(t, fail.ToError(), locale.T("err_api_project_not_found"))
	assert.Nil(t, project)
}
