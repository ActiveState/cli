package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/api/client/users"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) {
	api.RemoveAuth()
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
}

func setupUser(t *testing.T) *models.UserEditable {
	// Create test user
	uid := xid.New().String()
	testUser := &models.UserEditable{
		Username: fmt.Sprintf("cli-test-%s", uid),
		Email:    fmt.Sprintf("%s@cli-test.tld", uid),
		Password: "testtest",
		Name:     "cli test",
	}

	params := users.NewAddUserParams()
	params.SetUser(testUser)
	_, err := api.Client.Users.AddUser(params)
	assert.NoError(t, err, "Can create user")

	return testUser
}

func TestExecuteNoArgs(t *testing.T) {
	setup(t)

	testCredentials = &models.Credentials{
		Username: "FailureTest",
		Password: "FailureTest",
	}

	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.Nil(t, api.Auth, "Did not authenticate")
}

func TestExecuteNoArgsLoginByPrompt(t *testing.T) {
	setup(t)
	user := setupUser(t)

	testCredentials = &models.Credentials{
		Username: user.Username,
		Password: user.Password,
	}

	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	//assert.NotNil(t, api.Auth, "Authenticated")
}
