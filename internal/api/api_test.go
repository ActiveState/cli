package api

import (
	"fmt"
	"testing"

	"github.com/ActiveState/cli/internal/api/client/users"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/rs/xid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) {
	RemoveAuth()
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
	_, err := Client.Users.AddUser(params)
	assert.NoError(t, err, "Can create user")

	return testUser
}

func TestEndpoint(t *testing.T) {
	assert.Equal(t, constants.APIHostStaging, APIHost, "We are running against the staging api")
	assert.NotNil(t, Client, "ReInitialize initialized the Client")
}

func TestApi(t *testing.T) {
	// We're just testing an easy to use API endpoint here, the point of this test is to test the lib, not the endpoint
	params := users.NewUniqueUsernameParams()
	params.SetUsername("DontCreateAUserWithThisName")
	res, err := Client.Users.UniqueUsername(params)
	assert.NoError(t, err)
	assert.Equal(t, int64(200), *res.Payload.Code, "Should return HTTP Code 200")
}

func TestAuth(t *testing.T) {
	setup(t)
	user := setupUser(t)

	credentials := &models.Credentials{
		Username: user.Username,
		Password: user.Password,
	}
	_, err := Authenticate(credentials)
	assert.NoError(t, err, "Can Authenticate")
	assert.NotEmpty(t, viper.GetString("apiToken"), "Authentication is persisted through token")
	assert.NotNil(t, Auth, "Authentication is persisted for this session")

	bearerToken = ""
	Auth = nil
	ReInitialize()
	assert.NotNil(t, Auth, "Authentication is still persisted for this session")

	RemoveAuth()
	_, err = Authenticate(credentials)
	assert.NoError(t, err, "Can Authenticate Again")
}

func TestAUthFailure(t *testing.T) {
	setup(t)

	credentials := &models.Credentials{
		Username: "testFailure",
		Password: "testFailure",
	}
	_, err := Authenticate(credentials)
	assert.Error(t, err, "Should fail to authenticate")

	viper.Set("apiToken", "testFailure")
	ReInitialize()
	assert.Empty(t, bearerToken, "Should not have authenticated")
	assert.Empty(t, viper.GetString("apiToken"), "", "apiToken should have cleared")
}
