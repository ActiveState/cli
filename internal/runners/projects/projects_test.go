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

func (c *configMock) GetStringMapStringSlice(key string) map[string][]string {
	return map[string][]string{
		"organizationname/test project": {"/some/local/path/"},
	}
}

func (c *configMock) AllKeys() []string {
	return []string{}
}

func (c *configMock) GetStringSlice(_ string) []string {
	return []string{}
}

func (c *configMock) Set(_ string, _ interface{}) error { return nil }

func TestProjects(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.Register("GET", "/organizations")
	httpmock.Register("GET", "/organizations/organizationName/projects")

	catcher := outputhelper.NewCatcher()
	pjs := newProjects(authentication.Get(), catcher.Outputer, &configMock{})

	projects, err := pjs.fetchProjects(false)
	assert.NoError(t, err, "Fetched projects")
	assert.Equal(t, 1, len(projects), "One project fetched")
	assert.Equal(t, "test project (test description)", projects[0].Name)
	assert.Equal(t, "organizationName", projects[0].Organization)
	assert.Equal(t, []string{"/some/local/path/"}, projects[0].LocalCheckouts)

	err = pjs.RunRemote(NewParams())
	assert.NoError(t, err, "Executed without error")
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
	pjs := newProjects(authentication.Get(), catcher.Outputer, &configMock{})

	projects, err := pjs.fetchProjects(false)
	assert.NoError(t, err, "Fetched projects")
	assert.Equal(t, 0, len(projects), "No projects returned")

	err = pjs.RunRemote(NewParams())
	assert.NoError(t, err, "Executed without error")
}

func TestClientError(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	catcher := outputhelper.NewCatcher()
	pjs := newProjects(authentication.Get(), catcher.Outputer, &configMock{})

	_, err := pjs.fetchProjects(false)
	assert.Error(t, err, "Should not be able to fetch organizations without mock")

	httpmock.Register("GET", "/organizations")
	_, err = pjs.fetchProjects(false)
	assert.Error(t, err, "Should not be able to fetch projects without mock")
}

func TestAuthError(t *testing.T) {
	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.RegisterWithCode("GET", "/organizations", 401)

	catcher := outputhelper.NewCatcher()
	pjs := newProjects(authentication.Get(), catcher.Outputer, &configMock{})

	_, err := pjs.fetchProjects(false)
	assert.Error(t, err, "Should not be able to fetch projects without being authenticated")

	err = pjs.RunRemote(NewParams())
	assert.Error(t, err, "Executed with error")
}
