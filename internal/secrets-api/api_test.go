package secretsapi_test

import (
	"fmt"
	"testing"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretsAPI_NewClient_Success(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	client := secretsapi.NewClient("http", constants.SecretsAPIHostTesting, constants.SecretsAPIPath, "bearer123")
	require.NotNil(client)
	assert.NotNil(client.Auth)
	assert.Equal(fmt.Sprintf("http://%s%s", constants.SecretsAPIHostTesting, constants.SecretsAPIPath), client.BaseURI)

	rt, isRuntime := client.Transport.(*httptransport.Runtime)
	require.True(isRuntime, "client.Transport is a Runtime")
	assert.Equal(constants.SecretsAPIHostTesting, rt.Host)
	assert.Equal(constants.SecretsAPIPath, rt.BasePath)

	// validate that the client.Auth writer sets the bearer token using the one we provided
	mockClientRequest := new(MockClientRequest)
	mockClientRequest.On("SetHeaderParam", "Authorization", []string{"Bearer bearer123"}).Return(nil)

	authErr := client.Auth.AuthenticateRequest(mockClientRequest, nil)
	require.NoError(authErr)
	assert.True(mockClientRequest.AssertExpectations(t))
}

func TestSecretsAPI_Authenticated_Failure(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	client := secretsapi.NewTestClient("http", constants.SecretsAPIHostTesting, constants.SecretsAPIPath, "bearer123")
	require.NotNil(client)

	httpmock.Activate(client.BaseURI)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("GET", "/whoami", 401)

	uid, failure := client.Authenticated()
	assert.Nil(uid)
	assert.True(failure.Type.Matches(api.FailAuth), "should be an FailAuth failure")
}

func TestSecretsAPI_Authenticated_Success(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	client := secretsapi.NewTestClient("http", constants.SecretsAPIHostTesting, constants.SecretsAPIPath, "bearer123")
	require.NotNil(client)

	httpmock.Activate(client.BaseURI)
	defer httpmock.DeActivate()

	httpmock.RegisterWithCode("GET", "/whoami", 200)

	uid, failure := client.Authenticated()
	assert.Nil(failure)
	assert.Equal("11110000-1111-0000-1111-000011110000", uid.String())
}
