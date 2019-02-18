package organizations_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/organizations"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/stretchr/testify/assert"
)

func TestOrganizations_FetchAll(t *testing.T) {
	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/organizations")

	orgs, fail := organizations.FetchAll()
	assert.NoError(t, fail.ToError(), "Fetched organizations")
	assert.Equal(t, 1, len(orgs), "One organization fetched")
	assert.Equal(t, "test-organization", orgs[0].Name)
}

func TestOrganizations_FetchByURLName_Succeeds(t *testing.T) {
	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("GET", "/organizations/FooOrg", 200)

	org, fail := organizations.FetchByURLName("FooOrg")
	assert.NoError(t, fail.ToError(), "Fetched organizations")
	assert.Equal(t, "FooOrg", org.Urlname)
	assert.Equal(t, "FooOrg Name", org.Name)
}

func TestOrganizations_FetchByURLName_NotFound(t *testing.T) {
	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("GET", "/organizations/BarOrg", 404)

	org, fail := organizations.FetchByURLName("BarOrg")
	assert.EqualError(t, fail, locale.T("err_api_org_not_found"))
	assert.Nil(t, org)
}
