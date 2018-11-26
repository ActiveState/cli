package projects_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/projects"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestProjects_FetchByName_Succeeds(t *testing.T) {
	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("GET", "/organizations/ActiveState/projects/CodeIntel", 200)

	project, fail := projects.FetchByName(&models.Organization{Urlname: "ActiveState"}, "CodeIntel")
	assert.NoError(t, fail.ToError(), "Fetched project")
	assert.Equal(t, "CodeIntel", project.Name)
}

func TestProjects_FetchByName_NotFound(t *testing.T) {
	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("GET", "/organizations/ActiveState/projects/CodeIntel", 404)

	project, fail := projects.FetchByName(&models.Organization{Urlname: "ActiveState"}, "CodeIntel")
	assert.EqualError(t, fail.ToError(), locale.T("err_api_project_not_found"))
	assert.Nil(t, project)
}
