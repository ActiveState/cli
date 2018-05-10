package organizations

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

func TestOrganizations(t *testing.T) {
	setup(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("GET", "/organizations")

	orgs, err := fetchOrganizations()
	assert.NoError(t, err, "Fetched organizations")
	assert.Equal(t, 1, len(orgs.Payload), "One organization fetched")
	assert.Equal(t, "test-organization", orgs.Payload[0].Name)

	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
}

func TestClientError(t *testing.T) {
	setup(t)

	httpmock.Activate(api.Prefix)
	defer httpmock.DeActivate()

	_, err := fetchOrganizations()
	assert.Error(t, err, "Should not be able to fetch organizations without mock")

	err = Command.Execute()
	assert.NoError(t, err, "Command still executes without error")
}
