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

	BearerToken = ""
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
	assert.Empty(t, BearerToken, "Should not have authenticated")
	assert.Empty(t, viper.GetString("apiToken"), "", "apiToken should have cleared")
}

func TestErrorCode_WithoutPayload(t *testing.T) {
	setup(t)
	assert.Equal(t, 100, ErrorCode(&struct{ Code int }{
		Code: 100,
	}))
}

func TestErrorCode_WithoutPayload_NoCodeValue(t *testing.T) {
	setup(t)
	assert.Equal(t, -1, ErrorCode(&struct{ OtherCode int }{
		OtherCode: 100,
	}))
}

func TestErrorCode_WithPayload(t *testing.T) {
	setup(t)
	providedCode := 200
	codeValue := struct{ Code *int }{Code: &providedCode}
	payload := struct{ Payload struct{ Code *int } }{
		Payload: codeValue,
	}

	assert.Equal(t, 200, ErrorCode(&payload))
}

func TestErrorCode_WithPayload_CodeNotPointer(t *testing.T) {
	setup(t)
	providedCode := 300
	codeValue := struct{ Code int }{Code: providedCode}
	payload := struct{ Payload struct{ Code int } }{
		Payload: codeValue,
	}

	assert.Equal(t, 300, ErrorCode(&payload))
}

func TestErrorCode_WithPayload_NoCodeField(t *testing.T) {
	setup(t)
	providedCode := 400
	codeValue := struct{ OtherCode int }{OtherCode: providedCode}
	payload := struct{ Payload struct{ OtherCode int } }{
		Payload: codeValue,
	}

	assert.Equal(t, -1, ErrorCode(&payload))
}
