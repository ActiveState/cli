package projects

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/environment"
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

func TestProjects(t *testing.T) {
	setup(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/organizations")
	httpmock.Register("GET", "/organizations/organizationName/projects")

	projects, err := fetchProjects()
	assert.NoError(t, err, "Fetched projects")
	assert.Equal(t, 1, len(projects), "One project fetched")
	assert.Equal(t, "test project", projects[0].Name)
	assert.Equal(t, "organizationName", projects[0].Organization)
	assert.Equal(t, "test description", projects[0].Description)

	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
}

func TestClientError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	_, err := fetchProjects()
	assert.Error(t, err, "Should not be able to fetch organizations without mock")

	httpmock.Register("GET", "/organizations")
	_, err = fetchProjects()
	assert.Error(t, err, "Should not be able to fetch projects without mock")

	err = Command.Execute()
	assert.NoError(t, err, "Command still executes without error")
}
