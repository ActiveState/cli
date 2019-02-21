package projects

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/authentication"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})
}

func TestProjects(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.Register("GET", "/organizations")
	httpmock.Register("GET", "/organizations/organizationName/projects")

	projects, fail := fetchProjects()
	assert.NoError(t, fail.ToError(), "Fetched projects")
	assert.Equal(t, 1, len(projects), "One project fetched")
	assert.Equal(t, "test project", projects[0].Name)
	assert.Equal(t, "organizationName", projects[0].Organization)
	assert.Equal(t, "test description", projects[0].Description)

	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestProjectsEmpty(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.RegisterWithResponder("GET", "/organizations", func(req *http.Request) (int, string) {
		return 200, "organizations-empty"
	})

	projects, fail := fetchProjects()
	assert.NoError(t, fail.ToError(), "Fetched projects")
	assert.Equal(t, 0, len(projects), "No projects returned")

	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestClientError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	_, fail := fetchProjects()
	assert.Error(t, fail.ToError(), "Should not be able to fetch organizations without mock")

	httpmock.Register("GET", "/organizations")
	_, fail = fetchProjects()
	assert.Error(t, fail.ToError(), "Should not be able to fetch projects without mock")

	err := Command.Execute()
	assert.NoError(t, err, "Command still executes without error")
}

func TestAuthError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.RegisterWithCode("GET", "/organizations", 401)
	_, fail := fetchProjects()
	assert.Error(t, fail.ToError(), "Should not be able to fetch projects without being authenticated")
	assert.True(t, fail.Type.Matches(api.FailAuth), "Failure should be due to auth")

	err := Command.Execute()
	assert.NoError(t, err, "Command still executes without error")
	assert.Error(t, failures.Handled(), "Failure occurred")
}
