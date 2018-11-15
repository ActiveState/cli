package organizations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})
}

func TestOrganizations(t *testing.T) {
	setup(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/organizations")

	orgs, fail := FetchAll()
	assert.NoError(t, fail.ToError(), "Fetched organizations")
	assert.Equal(t, 1, len(orgs), "One organization fetched")
	assert.Equal(t, "test-organization", orgs[0].Name)

	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestOrganizations_FetchByURLName_Succeeds(t *testing.T) {
	setup(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("GET", "/organizations/FooOrg", 200)

	org, fail := FetchByURLName("FooOrg")
	assert.NoError(t, fail.ToError(), "Fetched organizations")
	assert.Equal(t, "FooOrg", org.Urlname)
	assert.Equal(t, "FooOrg Name", org.Name)
}

func TestOrganizations_FetchByURLName_NotFound(t *testing.T) {
	setup(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("GET", "/organizations/BarOrg", 404)

	org, fail := FetchByURLName("BarOrg")
	assert.EqualError(t, fail, locale.T("err_api_org_not_found"))
	assert.Nil(t, org)
}

func TestClientError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	_, fail := FetchAll()
	assert.Error(t, fail.ToError(), "Should not be able to fetch organizations without mock")

	err := Command.Execute()
	assert.NoError(t, err, "Command still executes without error")
	assert.Error(t, failures.Handled(), "Failure occurred")
}

func TestAuthError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("GET", "/organizations", 401)
	_, fail := FetchAll()
	assert.Error(t, fail.ToError(), "Should not be able to fetch projects without being authenticated")
	assert.True(t, fail.Type.Matches(api.FailAuth), "Failure should be due to auth")

	err := Command.Execute()
	assert.NoError(t, err, "Command still executes without error")
	assert.Error(t, failures.Handled(), "Failure occurred")
}

func TestAliases(t *testing.T) {
	cc := Command.GetCobraCmd()
	assert.True(t, cc.HasAlias("orgs"), "Command has alias.")
}
