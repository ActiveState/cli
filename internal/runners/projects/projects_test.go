package projects

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type configMock struct{}

func (c *configMock) GetStringMapString(key string) map[string]string {
	return nil
}

func TestProjects(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.Register("GET", "/organizations")
	httpmock.Register("GET", "/organizations/organizationName/projects")

	catcher := outputhelper.NewCatcher()
	pjs := NewProjects(catcher.Outputer, authentication.Get(), &configMock{})

	projects, fail := pjs.fetchProjects()
	assert.NoError(t, fail.ToError(), "Fetched projects")
	assert.Equal(t, 1, len(projects), "One project fetched")
	assert.Equal(t, "test project", projects[0].Name)
	assert.Equal(t, "organizationName", projects[0].Organization)

	fail = pjs.Run()
	assert.NoError(t, fail.ToError(), "Executed without error")
}

func TestProjectsEmpty(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.RegisterWithResponder("GET", "/organizations", func(req *http.Request) (int, string) {
		return 200, "organizations-empty"
	})

	catcher := outputhelper.NewCatcher()
	pjs := NewProjects(catcher.Outputer, authentication.Get(), &configMock{})

	projects, fail := pjs.fetchProjects()
	assert.NoError(t, fail.ToError(), "Fetched projects")
	assert.Equal(t, 0, len(projects), "No projects returned")

	fail = pjs.Run()
	assert.NoError(t, fail.ToError(), "Executed without error")
}

func TestClientError(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	catcher := outputhelper.NewCatcher()
	pjs := NewProjects(catcher.Outputer, authentication.Get(), &configMock{})

	_, fail := pjs.fetchProjects()
	assert.Error(t, fail.ToError(), "Should not be able to fetch organizations without mock")

	httpmock.Register("GET", "/organizations")
	_, fail = pjs.fetchProjects()
	assert.Error(t, fail.ToError(), "Should not be able to fetch projects without mock")
}

func TestAuthError(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.RegisterWithCode("GET", "/organizations", 401)

	catcher := outputhelper.NewCatcher()
	pjs := NewProjects(catcher.Outputer, authentication.Get(), &configMock{})

	_, fail := pjs.fetchProjects()
	assert.Error(t, fail.ToError(), "Should not be able to fetch projects without being authenticated")
	assert.True(t, fail.Type.Matches(api.FailAuth), "Failure should be due to auth")

	fail = pjs.Run()
	assert.Error(t, fail.ToError(), "Executed with error")
}
