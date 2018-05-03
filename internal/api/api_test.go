package api

import (
	"testing"

	"github.com/ActiveState/cli/internal/api/client/authentication"

	"github.com/ActiveState/cli/internal/api/client/users"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) {
	RemoveAuth()
}

func setupUser(t *testing.T) *models.UserEditable {
	httpmock.Activate(Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/users")

	// Create test user
	testUser := &models.UserEditable{
		Username: "test",
		Email:    "test@test.tld",
		Password: "test",
		Name:     "test",
	}

	params := users.NewAddUserParams()
	params.SetUser(testUser)
	_, err := Client.Users.AddUser(params)
	assert.NoError(t, err, "Can create user")

	return testUser
}

func TestEndpoint(t *testing.T) {
	assert.Equal(t, constants.APIHostTesting, APIHost, "We are running against the testing api")
	assert.NotNil(t, Client, "ReInitialize initialized the Client")
}

func TestAuth(t *testing.T) {
	setup(t)
	user := setupUser(t)

	httpmock.Activate(Prefix)
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("POST", "/apikeys")
	httpmock.Register("GET", "/apikeys")
	httpmock.Register("DELETE", "/apikeys/"+constants.APITokenName)

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

func TestAuthFailure(t *testing.T) {
	setup(t)

	httpmock.Activate(Prefix)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("POST", "/login", 401)

	credentials := &models.Credentials{
		Username: "testFailure",
		Password: "testFailure",
	}
	_, err := Authenticate(credentials)
	assert.IsType(t, new(authentication.PostLoginUnauthorized), err, "Should fail to authenticate")

	viper.Set("apiToken", "testFailure")
	ReInitialize()
	assert.Empty(t, bearerToken, "Should not have authenticated")
	assert.Empty(t, viper.GetString("apiToken"), "", "apiToken should have cleared")
}
